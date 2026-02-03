package policy

import (
	"path/filepath"
	"regexp"
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
	denylist []*regexp.Regexp
	redact   []redactor
}

func NewDefault() *Policy {
	return &Policy{
		denylist: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\bcat\s+~\/\.ssh\/`),
			regexp.MustCompile(`(?i)\bid_rsa\b`),
			regexp.MustCompile(`(?i)\.(pem|key)(\s|$)`),
			regexp.MustCompile(`(?i)\bkubectl\s+get\s+secret\b.*\s-o\s+(yaml|json)\b`),
			regexp.MustCompile(`(?i)\bgcloud\s+auth\s+print-access-token\b`),
		},
		redact: []redactor{
			{
				re:   regexp.MustCompile(`(?i)(authorization\s*:\s*bearer\s+)([^\s"']+)`),
				repl: `${1}` + RedactedValue,
			},
			{
				re:   regexp.MustCompile(`(?i)(--(?:token|password|passwd|api[_-]?key|apikey|secret|private[_-]?key)=)([^\s]+)`),
				repl: `${1}` + RedactedValue,
			},
			{
				re:   regexp.MustCompile(`(?i)(--(?:token|password|passwd|api[_-]?key|apikey|secret|private[_-]?key)\s+)([^\s]+)`),
				repl: `${1}` + RedactedValue,
			},
			{
				re:   regexp.MustCompile(`(?i)(-p\s+)([^\s]+)`),
				repl: `${1}` + RedactedValue,
			},
			{
				re:   regexp.MustCompile(`(?i)(\b(?:token|secret|password|passwd|api[_-]?key|apikey|private[_-]?key)\b\s*[:=]\s*)([^\s]+)`),
				repl: `${1}` + RedactedValue,
			},
			{
				re:   regexp.MustCompile(`(?i)(\b[A-Za-z_][A-Za-z0-9_]*=)([^\s]+)`),
				repl: `${1}` + RedactedValue,
			},
		},
	}
}

func (p *Policy) Apply(rawCommand string, args []string) Result {
	if p.isDenied(rawCommand, args) {
		return Result{
			Command: DeniedPlaceholder,
			Denied:  true,
		}
	}

	sanitized := rawCommand
	for _, rule := range p.redact {
		sanitized = rule.re.ReplaceAllString(sanitized, rule.repl)
	}

	return Result{
		Command: sanitized,
		Denied:  false,
	}
}

func (p *Policy) isDenied(rawCommand string, args []string) bool {
	if len(args) > 0 {
		binary := strings.ToLower(filepath.Base(args[0]))
		if binary == "env" || binary == "printenv" {
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
