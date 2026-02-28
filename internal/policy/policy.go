package policy

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const (
	RedactedValue     = "[REDACTED]"
	DeniedPlaceholder = "[REDACTED BY POLICY]"
)

type Result struct {
	Command string
	Denied  bool
}

type redactor struct {
	re   *regexp.Regexp
	repl string
}

type Policy struct {
	denylist        []*regexp.Regexp
	redact          []redactor
	enforceDenylist bool
}

type Options struct {
	DenylistPatterns  []string
	RedactionKeywords []string
	EnforceDenylist   bool
}

var credentialInImageRef = regexp.MustCompile(`^[^/\s:@]+:[^/\s@]+@`)
var uriUserinfo = regexp.MustCompile(`(?i)([a-z][a-z0-9+.\-]*://)([^/\s:@]+):([^/\s@]+)@`)

var defaultDenylistPatterns = []string{
	"env",
	"printenv",
	"cat ~/.ssh/*",
	"*id_rsa*",
	"*.pem",
	"*.key",
	"kubectl get secret -o yaml",
	"kubectl get secret -o json",
	"gcloud auth print-access-token",
}

var defaultRedactionKeywords = []string{
	"token",
	"secret",
	"password",
	"passwd",
	"authorization",
	"bearer",
	"api_key",
	"apikey",
	"private_key",
}

func NewDefault() *Policy {
	p, _ := New(Options{
		DenylistPatterns:  defaultDenylistPatterns,
		RedactionKeywords: defaultRedactionKeywords,
		EnforceDenylist:   false,
	})
	return p
}

func New(opts Options) (*Policy, error) {
	denyPatterns := opts.DenylistPatterns
	if len(denyPatterns) == 0 {
		denyPatterns = defaultDenylistPatterns
	}

	redactionKeywords := opts.RedactionKeywords
	if len(redactionKeywords) == 0 {
		redactionKeywords = defaultRedactionKeywords
	}

	denylist := make([]*regexp.Regexp, 0, len(denyPatterns))
	for _, pattern := range denyPatterns {
		if strings.TrimSpace(pattern) == "" {
			continue
		}
		re, err := compileDenyPattern(pattern)
		if err != nil {
			return nil, err
		}
		denylist = append(denylist, re)
	}

	return &Policy{
		denylist:        denylist,
		redact:          buildRedactors(redactionKeywords),
		enforceDenylist: opts.EnforceDenylist,
	}, nil
}

func buildRedactors(keywords []string) []redactor {
	keyPattern := keywordRegexPattern(keywords)
	keyValuePattern := keywordRegexPattern(filterKeywords(keywords, map[string]bool{
		"authorization": true,
		"bearer":        true,
	}))
	return []redactor{
		{
			re:   regexp.MustCompile(`(?i)(authorization\s*:\s*bearer\s+)([^\s"']+)`),
			repl: `${1}` + RedactedValue,
		},
		{
			re:   uriUserinfo,
			repl: `${1}` + RedactedValue + `:` + RedactedValue + `@`,
		},
		{
			re:   regexp.MustCompile(`(?i)(--(?:` + keyPattern + `)=)([^\s]+)`),
			repl: `${1}` + RedactedValue,
		},
		{
			re:   regexp.MustCompile(`(?i)(--(?:` + keyPattern + `)\s+)([^\s]+)`),
			repl: `${1}` + RedactedValue,
		},
		{
			re:   regexp.MustCompile(`(?i)(-p\s+)([^\s]+)`),
			repl: `${1}` + RedactedValue,
		},
		{
			re:   regexp.MustCompile(`(?i)(\b(?:` + keyValuePattern + `)\b\s*[:=]\s*)([^\s]+)`),
			repl: `${1}` + RedactedValue,
		},
		{
			re:   regexp.MustCompile(`(?i)(\b[A-Za-z_][A-Za-z0-9_]*=)"[^"]*"`),
			repl: `${1}"` + RedactedValue + `"`,
		},
		{
			re:   regexp.MustCompile(`(?i)(\b[A-Za-z_][A-Za-z0-9_]*=)'[^']*'`),
			repl: `${1}'` + RedactedValue + `'`,
		},
		{
			re:   regexp.MustCompile(`(?i)(\b[A-Za-z_][A-Za-z0-9_]*=)([^\s"']+)`),
			repl: `${1}` + RedactedValue,
		},
	}
}

func filterKeywords(keywords []string, deny map[string]bool) []string {
	out := make([]string, 0, len(keywords))
	for _, keyword := range keywords {
		k := strings.TrimSpace(strings.ToLower(keyword))
		if k == "" || deny[k] {
			continue
		}
		out = append(out, k)
	}
	return out
}

func keywordRegexPattern(keywords []string) string {
	if len(keywords) == 0 {
		return "token"
	}
	parts := make([]string, 0, len(keywords))
	for _, k := range keywords {
		k = strings.TrimSpace(strings.ToLower(k))
		if k == "" {
			continue
		}
		p := regexp.QuoteMeta(k)
		p = strings.ReplaceAll(p, `\_`, `[_-]?`)
		p = strings.ReplaceAll(p, `\-`, `[_-]?`)
		parts = append(parts, p)
	}
	if len(parts) == 0 {
		return "token"
	}
	return strings.Join(parts, "|")
}

func compileDenyPattern(pattern string) (*regexp.Regexp, error) {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return nil, fmt.Errorf("denylist pattern cannot be empty")
	}
	glob := regexp.QuoteMeta(pattern)
	glob = strings.ReplaceAll(glob, `\*`, `.*`)
	glob = strings.ReplaceAll(glob, `\ `, `\s+`)
	re, err := regexp.Compile(`(?i)` + glob)
	if err != nil {
		return nil, fmt.Errorf("invalid denylist pattern %q: %w", pattern, err)
	}
	return re, nil
}

func LoadFromConfig(path string) (*Policy, error) {
	cfg, err := ParseConfigFile(path)
	if err != nil {
		return nil, err
	}
	return New(Options{
		DenylistPatterns:  cfg.Denylist,
		RedactionKeywords: cfg.RedactionKeywords,
		EnforceDenylist:   cfg.EnforceDenylist,
	})
}

func LoadFromConfigOrDefault(path string) (*Policy, error) {
	_, statErr := os.Stat(path)
	if statErr != nil {
		if os.IsNotExist(statErr) {
			return NewDefault(), nil
		}
		return nil, statErr
	}
	return LoadFromConfig(path)
}

func (p *Policy) EnforceDenylist() bool {
	return p.enforceDenylist
}

func (p *Policy) Apply(rawCommand string, args []string) Result {
	if p.isDenied(rawCommand, args) {
		return Result{
			Command: DeniedPlaceholder,
			Denied:  true,
		}
	}

	sanitized, preserved := preserveKubectlSetImageAssignments(rawCommand, args)
	for _, rule := range p.redact {
		sanitized = rule.re.ReplaceAllString(sanitized, rule.repl)
	}
	for placeholder, original := range preserved {
		sanitized = strings.ReplaceAll(sanitized, placeholder, original)
	}

	return Result{
		Command: sanitized,
		Denied:  false,
	}
}

func preserveKubectlSetImageAssignments(rawCommand string, args []string) (string, map[string]string) {
	if !isKubectlSetImage(args) {
		return rawCommand, nil
	}

	sanitized := rawCommand
	preserved := make(map[string]string)
	index := 0

	for _, arg := range args {
		if strings.HasPrefix(arg, "-") || !strings.Contains(arg, "=") {
			continue
		}
		if !isSafeImageAssignment(arg) {
			continue
		}

		placeholder := "__COMMANDRY_IMG_ASSIGN_" + strconv.Itoa(index) + "__"
		index++
		sanitized = strings.ReplaceAll(sanitized, arg, placeholder)
		preserved[placeholder] = arg
	}

	return sanitized, preserved
}

func isKubectlSetImage(args []string) bool {
	if len(args) < 3 {
		return false
	}

	binary := strings.ToLower(filepath.Base(args[0]))
	if binary != "kubectl" && binary != "kubectl.exe" {
		return false
	}

	return strings.EqualFold(args[1], "set") && strings.EqualFold(args[2], "image")
}

func isSafeImageAssignment(arg string) bool {
	parts := strings.SplitN(arg, "=", 2)
	if len(parts) != 2 {
		return false
	}

	key := strings.ToLower(parts[0])
	value := strings.ToLower(parts[1])
	if key == "" || value == "" {
		return false
	}

	if containsSensitiveKeyword(key) || containsSensitiveKeyword(value) {
		return false
	}

	return !credentialInImageRef.MatchString(parts[1])
}

func containsSensitiveKeyword(v string) bool {
	keywords := []string{
		"token", "secret", "password", "passwd", "authorization", "bearer", "api_key", "apikey", "private_key",
	}
	for _, keyword := range keywords {
		if strings.Contains(v, keyword) {
			return true
		}
	}
	return false
}

func (p *Policy) isDenied(rawCommand string, args []string) bool {
	if len(args) > 0 {
		binary := strings.ToLower(filepath.Base(args[0]))
		if binary == "env" || binary == "printenv" {
			return true
		}
		if isKubectlSecretOutputDenied(args) {
			return true
		}
	}

	for _, rule := range p.denylist {
		if rule.MatchString(rawCommand) {
			return true
		}
	}

	return false
}

func isKubectlSecretOutputDenied(args []string) bool {
	if len(args) < 5 {
		return false
	}
	binary := strings.ToLower(filepath.Base(args[0]))
	if binary != "kubectl" && binary != "kubectl.exe" {
		return false
	}
	if !strings.EqualFold(args[1], "get") || !strings.EqualFold(args[2], "secret") {
		return false
	}
	for i := 3; i < len(args)-1; i++ {
		if args[i] == "-o" && (strings.EqualFold(args[i+1], "yaml") || strings.EqualFold(args[i+1], "json")) {
			return true
		}
	}
	return false
}
