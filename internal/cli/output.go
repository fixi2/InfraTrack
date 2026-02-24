package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

func printOK(out io.Writer, format string, args ...any) {
	printRole(out, roleOK, format, args...)
}

func printWarn(out io.Writer, format string, args ...any) {
	printRole(out, roleWarn, format, args...)
}

func printError(out io.Writer, format string, args ...any) {
	printRole(out, roleError, format, args...)
}

func printHint(out io.Writer, format string, args ...any) {
	printHints(out, fmt.Sprintf(format, args...))
}

func printTips(out io.Writer, tips ...string) {
	printHints(out, tips...)
}

type messageRole int

const (
	roleOK messageRole = iota
	roleWarn
	roleError
	roleHint
)

type outputSettings struct {
	noColor bool
}

var (
	outputCfgMu sync.RWMutex
	outputCfg   = outputSettings{}
)

func configureOutput(noColor bool) {
	outputCfgMu.Lock()
	outputCfg.noColor = noColor
	outputCfgMu.Unlock()
}

func printRole(out io.Writer, role messageRole, format string, args ...any) {
	label := styleLabel(out, role)
	fmt.Fprintf(out, "%s %s\n", label, fmt.Sprintf(format, args...))
}

func printHints(out io.Writer, hints ...string) {
	filtered := make([]string, 0, len(hints))
	for _, h := range hints {
		h = strings.TrimSpace(h)
		if h == "" {
			continue
		}
		filtered = append(filtered, h)
	}
	if len(filtered) == 0 {
		return
	}

	if len(filtered) == 1 {
		fmt.Fprintln(out, "Tip:")
		fmt.Fprintf(out, "   %s\n", filtered[0])
		return
	}

	fmt.Fprintln(out, "Tips:")
	arrow := "->"
	if supportsUnicode(out) {
		arrow = "\u2192"
	}
	for _, h := range filtered {
		fmt.Fprintf(out, "   %s %s\n", arrow, h)
	}
}

func styleLabel(out io.Writer, role messageRole) string {
	icon := roleASCII(role)
	if supportsUnicode(out) {
		icon = roleIcon(role)
	}
	if supportsColor(out) {
		return colorize(icon, roleColor(role))
	}
	return icon
}

func roleASCII(role messageRole) string {
	switch role {
	case roleOK:
		return "[OK]"
	case roleWarn:
		return "[WARN]"
	case roleError:
		return "[ERROR]"
	default:
		return "Tip:"
	}
}

func roleIcon(role messageRole) string {
	switch role {
	case roleOK:
		return "\u2713"
	case roleWarn:
		return "!"
	case roleError:
		return "\u2715"
	default:
		return "\u2192"
	}
}

func roleColor(role messageRole) int {
	switch role {
	case roleOK:
		return 32 // green
	case roleWarn:
		return 33 // yellow
	case roleError:
		return 31 // red
	default:
		return 36 // cyan
	}
}

func supportsColor(out io.Writer) bool {
	outputCfgMu.RLock()
	forceNoColor := outputCfg.noColor
	outputCfgMu.RUnlock()

	if forceNoColor || os.Getenv("NO_COLOR") != "" || os.Getenv("CLICOLOR") == "0" {
		return false
	}
	if os.Getenv("CLICOLOR_FORCE") != "" && os.Getenv("CLICOLOR_FORCE") != "0" {
		return true
	}
	if !isTTY(out) {
		return false
	}
	return !strings.EqualFold(strings.TrimSpace(os.Getenv("TERM")), "dumb")
}

func supportsUnicode(out io.Writer) bool {
	outputCfgMu.RLock()
	forceNoColor := outputCfg.noColor
	outputCfgMu.RUnlock()

	if forceNoColor || os.Getenv("NO_COLOR") != "" || os.Getenv("CLICOLOR") == "0" {
		return false
	}
	if os.Getenv("INFRATRACK_ASCII") == "1" {
		return false
	}
	if !isTTY(out) {
		return false
	}
	return !strings.EqualFold(strings.TrimSpace(os.Getenv("TERM")), "dumb")
}

func isTTY(out io.Writer) bool {
	f, ok := out.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

func colorize(s string, code int) string {
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", code, s)
}

func runWithSpinner(out io.Writer, label string, fn func() error) error {
	if !spinnerEnabled(out) {
		return fn()
	}

	label = strings.TrimSpace(label)
	if label == "" {
		label = "Working..."
	}

	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		frames := []string{"|", "/", "-", `\`}
		frameIdx := 0
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		render := func(frame string) {
			if supportsColor(out) {
				frame = colorize(frame, 36)
			}
			fmt.Fprintf(out, "\r%s %s", frame, label)
		}

		render(frames[frameIdx])
		for {
			select {
			case <-done:
				clearSpinnerLine(out, label)
				return
			case <-ticker.C:
				frameIdx = (frameIdx + 1) % len(frames)
				render(frames[frameIdx])
			}
		}
	}()

	err := fn()
	close(done)
	wg.Wait()
	return err
}

func spinnerEnabled(out io.Writer) bool {
	if !isTTY(out) {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(os.Getenv("TERM")), "dumb") {
		return false
	}
	if os.Getenv("INFRATRACK_NO_SPINNER") == "1" {
		return false
	}
	return true
}

func clearSpinnerLine(out io.Writer, label string) {
	blank := strings.Repeat(" ", len(label)+4)
	fmt.Fprintf(out, "\r%s\r", blank)
}
