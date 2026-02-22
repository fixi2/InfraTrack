package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/fixi2/InfraTrack/internal/export"
	"github.com/fixi2/InfraTrack/internal/store"
)

const (
	maxCommentLen       = 500
	commandPreviewLimit = 120
)

type flaggedStep struct {
	StepIndex int
	Number    int
	Labels    []string
	Command   string
}

func collectFlaggedSteps(session *store.Session) []flaggedStep {
	flagged := make([]flaggedStep, 0, len(session.Steps))
	for i, step := range session.Steps {
		labels := make([]string, 0, 2)
		if isPolicyRedacted(step) {
			labels = append(labels, "REDACTED")
		}
		if isFailedStep(step) {
			labels = append(labels, "FAILED")
		}
		if len(labels) == 0 {
			continue
		}
		cmd := step.Command
		if !isPolicyRedacted(step) {
			cmd = previewCommand(cmd)
		}
		flagged = append(flagged, flaggedStep{
			StepIndex: i,
			Number:    len(flagged) + 1,
			Labels:    labels,
			Command:   cmd,
		})
	}
	return flagged
}

func isFailedStep(step store.Step) bool {
	if strings.EqualFold(step.Status, "FAILED") {
		return true
	}
	return step.ExitCode != nil && *step.ExitCode != 0
}

func isPolicyRedacted(step store.Step) bool {
	return strings.EqualFold(step.Status, "REDACTED") || strings.EqualFold(step.Reason, "policy_redacted") || strings.Contains(step.Command, "[REDACTED BY POLICY]")
}

func previewCommand(command string) string {
	command = strings.TrimSpace(command)
	r := []rune(command)
	if len(r) <= commandPreviewLimit {
		return command
	}
	return string(r[:commandPreviewLimit]) + "..."
}

func promptForExportAnnotations(in io.Reader, out io.Writer, session *store.Session) export.MarkdownOptions {
	flagged := collectFlaggedSteps(session)
	if len(flagged) == 0 {
		return export.MarkdownOptions{}
	}
	reader := bufio.NewReader(in)

	fmt.Fprintln(out, buildAnnotationPrompt(flagged))
	answer, ok := readLine(reader)
	if !ok || !isYesAnswer(answer) {
		return export.MarkdownOptions{}
	}

	stepComments := make(map[int][]string)
	globalComments := make([]string, 0, 2)

	for {
		fmt.Fprintln(out, "=== Flagged Steps ===")
		fmt.Fprintln(out, "[0] Global comment for all flagged steps")
		for _, f := range flagged {
			fmt.Fprintf(out, "[%d] [%s] %s\n", f.Number, strings.Join(f.Labels, ", "), f.Command)
		}
		fmt.Fprintln(out, "Select step numbers (space-separated, Enter to finish):")
		selectionLine, ok := readLine(reader)
		if !ok {
			break
		}
		selectionLine = strings.TrimSpace(selectionLine)
		if selectionLine == "" {
			break
		}

		selectedGlobal, selectedSteps, err := parseSelection(selectionLine, flagged)
		if err != nil {
			fmt.Fprintf(out, "Invalid selection: %v\n", err)
			continue
		}

		fmt.Fprintln(out, "Enter comment:")
		commentLine, ok := readLine(reader)
		if !ok {
			break
		}
		comment := sanitizeComment(commentLine)
		if comment == "" {
			fmt.Fprintln(out, "Empty comment ignored.")
			continue
		}

		if selectedGlobal {
			globalComments = append(globalComments, comment)
		}
		for _, idx := range selectedSteps {
			stepComments[idx] = append(stepComments[idx], comment)
		}

		fmt.Fprintln(out, "Comment saved. Add another? Select more steps or press Enter to finish.")
	}

	return export.MarkdownOptions{
		StepComments:   stepComments,
		GlobalComments: globalComments,
	}
}

func buildAnnotationPrompt(flagged []flaggedStep) string {
	hasFailed := false
	hasRedacted := false
	for _, f := range flagged {
		for _, label := range f.Labels {
			if label == "FAILED" {
				hasFailed = true
			}
			if label == "REDACTED" {
				hasRedacted = true
			}
		}
	}

	switch {
	case hasFailed && hasRedacted:
		return fmt.Sprintf("Detected %d flagged step(s) with failures and policy redactions. Add export comments? [Y/n]", len(flagged))
	case hasRedacted:
		return fmt.Sprintf("Detected %d flagged step(s) with policy redactions. Add export comments? [Y/n]", len(flagged))
	default:
		return fmt.Sprintf("Detected %d flagged step(s) with command failures. Add export comments? [Y/n]", len(flagged))
	}
}

func parseSelection(input string, flagged []flaggedStep) (bool, []int, error) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return false, nil, fmt.Errorf("no selections")
	}

	allowed := make(map[int]int, len(flagged))
	for _, f := range flagged {
		allowed[f.Number] = f.StepIndex
	}

	selectedGlobal := false
	stepSet := make(map[int]struct{})
	for _, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil {
			return false, nil, fmt.Errorf("invalid number %q", part)
		}
		if n == 0 {
			selectedGlobal = true
			continue
		}
		stepIndex, ok := allowed[n]
		if !ok {
			return false, nil, fmt.Errorf("step %d is not in the list", n)
		}
		stepSet[stepIndex] = struct{}{}
	}

	steps := make([]int, 0, len(stepSet))
	for idx := range stepSet {
		steps = append(steps, idx)
	}
	sort.Ints(steps)
	return selectedGlobal, steps, nil
}

func sanitizeComment(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.Map(func(r rune) rune {
		if r == '\t' {
			return ' '
		}
		if r == 0x1b {
			return -1
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
	s = strings.Join(strings.Fields(s), " ")
	r := []rune(s)
	if len(r) > maxCommentLen {
		s = string(r[:maxCommentLen])
	}
	return strings.TrimSpace(s)
}

func readLine(reader *bufio.Reader) (string, bool) {
	line, err := reader.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			return strings.TrimSpace(line), true
		}
		return "", false
	}
	return strings.TrimSpace(line), true
}

func isYesAnswer(s string) bool {
	s = strings.TrimSpace(strings.ToLower(s))
	return s == "" || s == "y" || s == "yes"
}

func isInteractiveSession() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}
