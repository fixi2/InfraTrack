package export

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/fixi2/InfraTrack/internal/buildinfo"
	"github.com/fixi2/InfraTrack/internal/store"
)

var nonSlugChars = regexp.MustCompile(`[^a-z0-9]+`)
var (
	kubectlWord          = regexp.MustCompile(`\bkubectl\b`)
	helmWord             = regexp.MustCompile(`\bhelm\b`)
	dockerWord           = regexp.MustCompile(`\bdocker\b`)
	terraformWord        = regexp.MustCompile(`\bterraform\b`)
	kubectlApply         = regexp.MustCompile(`\bkubectl\b.*\bapply\b`)
	kubectlRolloutStatus = regexp.MustCompile(`\bkubectl\b.*\brollout\b.*\bstatus\b`)
	awsWord              = regexp.MustCompile(`\baws\b`)
	gcloudWord           = regexp.MustCompile(`\bgcloud\b`)
	azWord               = regexp.MustCompile(`\baz\b`)
	psqlWord             = regexp.MustCompile(`\bpsql\b`)
	mysqlWord            = regexp.MustCompile(`\bmysql\b`)
	rolloutRestartDeploy = regexp.MustCompile(`\bkubectl\b.*\brollout\b.*\brestart\b.*\bdeployment\/([a-z0-9._-]+)\b`)
	rolloutStatusDeploy  = regexp.MustCompile(`\bkubectl\b.*\brollout\b.*\bstatus\b.*\bdeployment\/([a-z0-9._-]+)\b`)
	setImageDeploy       = regexp.MustCompile(`\bkubectl\b.*\bset\b.*\bimage\b.*\bdeployment\/([a-z0-9._-]+)\b`)
)

func WriteMarkdown(session *store.Session, workingDir string) (string, error) {
	return WriteMarkdownWithOptions(session, workingDir, MarkdownOptions{})
}

type MarkdownOptions struct {
	StepComments   map[int][]string
	GlobalComments []string
}

func WriteMarkdownWithOptions(session *store.Session, workingDir string, opts MarkdownOptions) (string, error) {
	runbooksDir := filepath.Join(workingDir, "runbooks")
	if err := os.MkdirAll(runbooksDir, 0o755); err != nil {
		return "", fmt.Errorf("create runbooks directory: %w", err)
	}

	filename := RunbookFilename(session)
	outputPath := filepath.Join(runbooksDir, filename)

	body := RenderMarkdownWithOptions(session, opts)
	if err := os.WriteFile(outputPath, []byte(body), 0o644); err != nil {
		return "", fmt.Errorf("write markdown file: %w", err)
	}

	return outputPath, nil
}

func RunbookFilename(session *store.Session) string {
	ts := session.StartedAt.UTC().Format("20060102-150405")
	slug := slugify(session.Title)
	return fmt.Sprintf("%s-%s.md", ts, slug)
}

func RenderMarkdown(session *store.Session) string {
	return RenderMarkdownWithOptions(session, MarkdownOptions{})
}

func RenderMarkdownWithOptions(session *store.Session, opts MarkdownOptions) string {
	var b strings.Builder

	summary := buildStepSummary(session.Steps)

	b.WriteString("# ")
	b.WriteString(session.Title)
	b.WriteString("\n\n")

	b.WriteString("## Summary\n")
	b.WriteString("This runbook was generated from an explicit InfraTrack session.\n")
	b.WriteString(fmt.Sprintf("Recorded %d step(s).\n", len(session.Steps)))
	b.WriteString(fmt.Sprintf("Results: OK %d | FAILED %d | REDACTED %d\n", summary.ok, summary.failed, summary.redacted))
	b.WriteString(fmt.Sprintf("Total duration: %d ms\n\n", summary.totalDurationMS))

	b.WriteString("## Before You Run\n")
	for _, precondition := range detectPreconditions(session.Steps) {
		b.WriteString("- [ ] ")
		b.WriteString(precondition)
		b.WriteString("\n")
	}
	b.WriteString("\n")

	b.WriteString("## Steps\n")
	if len(session.Steps) == 0 {
		b.WriteString("1. TODO: No recorded steps.\n\n")
		b.WriteString("```sh\n")
		b.WriteString("# TODO: add command\n")
		b.WriteString("```\n\n")
	} else {
		for i, step := range session.Steps {
			status, reason := normalizeResult(step)
			b.WriteString(fmt.Sprintf("%d. [%s] %s\n\n", i+1, status, stepTitleSnippet(step.Command)))
			b.WriteString("```sh\n")
			b.WriteString(step.Command)
			b.WriteString("\n```\n")
			b.WriteString(fmt.Sprintf("Result: %s", status))
			if reason != "" {
				b.WriteString(fmt.Sprintf(" (%s)", reason))
			}
			b.WriteString("\n")
			if step.ExitCode != nil {
				b.WriteString(fmt.Sprintf("Exit code: %d\n", *step.ExitCode))
			}
			b.WriteString(fmt.Sprintf("Duration: %d ms\n\n", step.DurationMS))
			if comments := opts.StepComments[i]; len(comments) > 0 {
				if len(comments) == 1 {
					b.WriteString("Reviewer note:\n")
				} else {
					b.WriteString("Reviewer notes:\n")
				}
				for _, comment := range comments {
					b.WriteString("- ")
					b.WriteString(comment)
					b.WriteString("\n")
				}
				b.WriteString("\n")
			}
		}
	}

	b.WriteString("## Verification\n")
	for _, check := range detectVerificationChecks(session.Steps) {
		b.WriteString("- [ ] ")
		b.WriteString(check)
		b.WriteString("\n")
	}
	b.WriteString("\n")

	rollbackSectionTitle, rollbackItems := detectRollback(session.Steps)
	b.WriteString("## ")
	b.WriteString(rollbackSectionTitle)
	b.WriteString("\n")
	for _, item := range rollbackItems {
		b.WriteString("- ")
		b.WriteString(item)
		b.WriteString("\n")
	}
	b.WriteString("\n")

	if len(opts.GlobalComments) > 0 {
		b.WriteString("## Export Comments\n")
		for _, comment := range opts.GlobalComments {
			b.WriteString("- Applies to all flagged steps: ")
			b.WriteString(comment)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	b.WriteString("## Notes\n")
	b.WriteString(fmt.Sprintf("- Generated by InfraTrack %s.\n", buildinfo.String()))

	return b.String()
}

type stepSummary struct {
	ok              int
	failed          int
	redacted        int
	totalDurationMS int64
}

func buildStepSummary(steps []store.Step) stepSummary {
	s := stepSummary{}
	for _, step := range steps {
		status, _ := normalizeResult(step)
		switch status {
		case "OK":
			s.ok++
		case "FAILED":
			s.failed++
		case "REDACTED":
			s.redacted++
		}
		if status != "REDACTED" && hasInlineRedaction(step.Command) {
			s.redacted++
		}
		if step.DurationMS > 0 {
			s.totalDurationMS += step.DurationMS
		}
	}
	return s
}

func hasInlineRedaction(command string) bool {
	return strings.Contains(command, "[REDACTED]")
}

func stepTitleSnippet(command string) string {
	cmd := strings.TrimSpace(command)
	if cmd == "" {
		return "(empty command)"
	}
	const maxRunes = 72
	if utf8.RuneCountInString(cmd) <= maxRunes {
		return cmd
	}
	rs := []rune(cmd)
	if maxRunes <= 3 {
		return string(rs[:maxRunes])
	}
	return string(rs[:maxRunes-3]) + "..."
}

func normalizeResult(step store.Step) (string, string) {
	status := step.Status
	reason := step.Reason

	if status == "" {
		if step.ExitCode == nil {
			return "UNKNOWN", "unknown"
		}
		if *step.ExitCode == 0 {
			return "OK", ""
		}
		return "FAILED", "nonzero_exit"
	}

	if status == "FAILED" && reason == "" && step.ExitCode != nil && *step.ExitCode != 0 {
		reason = "nonzero_exit"
	}
	if status == "REDACTED" && reason == "" {
		reason = "policy_redacted"
	}

	return status, reason
}

func detectPreconditions(steps []store.Step) []string {
	if len(steps) == 0 {
		return []string{
			"Required tools are installed and available in PATH.",
			"Credentials and environment context are set for the target system.",
		}
	}

	hasKubectl := false
	hasHelm := false
	hasDocker := false
	hasTerraform := false
	hasCloudCLI := false
	hasDBCLI := false
	for _, step := range steps {
		cmd := guidanceCommand(step.Command)
		if cmd == "" {
			continue
		}
		hasKubectl = hasKubectl || kubectlWord.MatchString(cmd)
		hasHelm = hasHelm || helmWord.MatchString(cmd)
		hasDocker = hasDocker || dockerWord.MatchString(cmd)
		hasTerraform = hasTerraform || terraformWord.MatchString(cmd)
		hasCloudCLI = hasCloudCLI || awsWord.MatchString(cmd) || gcloudWord.MatchString(cmd) || azWord.MatchString(cmd)
		hasDBCLI = hasDBCLI || psqlWord.MatchString(cmd) || mysqlWord.MatchString(cmd)
	}

	preconditions := make([]string, 0, 10)
	if hasKubectl {
		preconditions = append(preconditions,
			"`kubectl` is installed and available in PATH.",
			"Kubernetes context and access are configured (`KUBECONFIG`/current-context).",
		)
	}
	if hasHelm {
		preconditions = append(preconditions,
			"`helm` is installed and targets the intended Kubernetes context.",
			"Required chart repositories are configured and reachable.",
		)
	}
	if hasDocker {
		preconditions = append(preconditions,
			"Docker CLI is installed and Docker daemon is running.",
			"Current user has permission to access Docker daemon.",
		)
	}
	if hasTerraform {
		preconditions = append(preconditions,
			"`terraform` CLI is installed and initialized for this workspace.",
			"Backend credentials and target workspace are configured.",
		)
	}
	if hasCloudCLI {
		preconditions = append(preconditions,
			"Cloud CLI authentication is active for the intended account/project/subscription.",
			"Required IAM permissions are available for the target resources.",
		)
	}
	if hasDBCLI {
		preconditions = append(preconditions,
			"Database client access is configured (host, port, user, SSL mode).",
			"Use least-privilege credentials and avoid exposing secrets in command arguments.",
		)
	}

	if len(preconditions) == 0 {
		return []string{
			"Required tools are installed and available in PATH.",
			"Credentials and environment context are set for the target system.",
		}
	}

	preconditions = append(preconditions, "Sensitive values are not exposed in command arguments.")
	return preconditions
}

func detectVerificationChecks(steps []store.Step) []string {
	hasKubectlApply := false
	hasKubectlRolloutStatus := false

	for _, step := range steps {
		cmd := guidanceCommand(step.Command)
		if cmd == "" {
			continue
		}
		hasKubectlApply = hasKubectlApply || kubectlApply.MatchString(cmd)
		hasKubectlRolloutStatus = hasKubectlRolloutStatus || kubectlRolloutStatus.MatchString(cmd)
	}

	if hasKubectlApply || hasKubectlRolloutStatus {
		return []string{
			"`kubectl get pods` reports expected pod status.",
			"`kubectl rollout status deployment/<name>` completes successfully.",
		}
	}

	return []string{
		"Validate that each command achieved the intended result.",
	}
}

func detectRollback(steps []store.Step) (string, []string) {
	deployments := make([]string, 0, 2)
	seen := make(map[string]struct{})

	for _, step := range steps {
		if !isSuccessfulStep(step) {
			continue
		}
		cmd := guidanceCommand(step.Command)
		if cmd == "" {
			continue
		}
		name := extractDeploymentName(cmd)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		deployments = append(deployments, name)
	}

	if len(deployments) == 0 {
		return "Rollback", []string{"Document the rollback command for this workflow before production use."}
	}

	items := []string{"Verify root cause and deployment revision before undoing changes."}
	for _, name := range deployments {
		items = append(items, fmt.Sprintf("`kubectl rollout undo deployment/%s`", name))
	}
	return "Rollback", items
}

func isSuccessfulStep(step store.Step) bool {
	if step.Status != "" {
		return strings.EqualFold(step.Status, "OK")
	}
	if step.ExitCode != nil {
		return *step.ExitCode == 0
	}
	return false
}

func extractDeploymentName(cmd string) string {
	patterns := []*regexp.Regexp{
		rolloutRestartDeploy,
		rolloutStatusDeploy,
		setImageDeploy,
	}
	for _, p := range patterns {
		m := p.FindStringSubmatch(cmd)
		if len(m) == 2 {
			return m[1]
		}
	}
	return ""
}

func guidanceCommand(command string) string {
	cmd := strings.TrimSpace(strings.ToLower(command))
	if cmd == "" {
		return ""
	}
	// Ignore echo-style wrappers so guidance is based on executed actions, not echoed text.
	echoPrefixes := []string{
		"echo ",
		"cmd /c echo ",
		"cmd.exe /c echo ",
		"sh -lc \"echo ",
		"sh -lc 'echo ",
		"bash -lc \"echo ",
		"bash -lc 'echo ",
		"zsh -lc \"echo ",
		"zsh -lc 'echo ",
		"powershell -command echo ",
		"powershell -noprofile -command echo ",
		"pwsh -command echo ",
		"pwsh -noprofile -command echo ",
	}
	for _, prefix := range echoPrefixes {
		if strings.HasPrefix(cmd, prefix) {
			return ""
		}
	}
	return cmd
}

func slugify(title string) string {
	slug := strings.ToLower(strings.TrimSpace(title))
	slug = nonSlugChars.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return "runbook"
	}
	return slug
}
