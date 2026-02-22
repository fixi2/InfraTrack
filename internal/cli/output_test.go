package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestOutputRolesNonTTYASCII(t *testing.T) {
	t.Cleanup(func() { configureOutput(false) })
	configureOutput(false)

	var out bytes.Buffer
	printOK(&out, "ok message")
	printWarn(&out, "warn message")
	printError(&out, "error message")
	printHint(&out, "hint message")

	got := out.String()
	for _, want := range []string{
		"[OK] ok message",
		"[WARN] warn message",
		"[ERROR] error message",
		"Hint:\n   hint message",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, got)
		}
	}
	if strings.Contains(got, "\x1b[") {
		t.Fatalf("did not expect ANSI sequences in non-TTY output, got:\n%s", got)
	}
}

func TestRunWithSpinnerNonTTY(t *testing.T) {
	var out bytes.Buffer
	calls := 0
	err := runWithSpinner(&out, "Working...", func() error {
		calls++
		return nil
	})
	if err != nil {
		t.Fatalf("runWithSpinner returned error: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected wrapped function to be called once, got %d", calls)
	}
}

func TestPrintHintsMultiUsesArrows(t *testing.T) {
	var out bytes.Buffer
	printHints(&out, "first", "second")
	got := out.String()
	if !strings.Contains(got, "Hint:\n") {
		t.Fatalf("expected Hint header, got:\n%s", got)
	}
	if !strings.Contains(got, "   -> first\n") {
		t.Fatalf("expected ASCII arrow hint line, got:\n%s", got)
	}
	if !strings.Contains(got, "   -> second\n") {
		t.Fatalf("expected ASCII arrow hint line, got:\n%s", got)
	}
}
