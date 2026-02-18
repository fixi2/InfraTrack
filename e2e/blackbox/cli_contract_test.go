package blackbox

import (
	"strings"
	"testing"
)

// Contract: C1
func TestCLIHelpAndVersionContract(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	help := h.mustRun("--help")
	if !strings.Contains(help.Stdout, "Usage:") || !strings.Contains(help.Stdout, "Available Commands:") {
		t.Fatalf("help output missing contract tokens:\n%s", help.Stdout)
	}

	help2 := h.mustRun("help")
	if !strings.Contains(help2.Stdout, "Usage:") {
		t.Fatalf("help command output missing usage:\n%s", help2.Stdout)
	}

	ver := h.mustRun("version")
	if strings.TrimSpace(ver.Stdout) == "" {
		t.Fatalf("version output is empty")
	}
}

// Contract: C1
func TestUnknownCommandContract(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	res := h.run("no-such-command")
	if res.ExitCode == 0 {
		t.Fatalf("unknown command must fail")
	}
	text := strings.ToLower(res.Stdout + "\n" + res.Stderr)
	if !strings.Contains(text, "unknown command") {
		t.Fatalf("expected unknown command diagnostic, got:\nstdout=%s\nstderr=%s", res.Stdout, res.Stderr)
	}
}
