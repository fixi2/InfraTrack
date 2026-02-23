package blackbox

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func assertGolden(t *testing.T, goldenPath string, got string) {
	t.Helper()
	goldenAbs := filepath.Join(repoRoot(), "e2e", "blackbox", "testdata", goldenPath)
	if *updateGolden {
		if err := os.WriteFile(goldenAbs, []byte(got), 0o644); err != nil {
			t.Fatalf("update golden %s: %v", goldenAbs, err)
		}
		return
	}
	wantBytes, err := os.ReadFile(goldenAbs)
	if err != nil {
		t.Fatalf("read golden %s: %v", goldenAbs, err)
	}
	if normalizeGoldenNewlines(string(wantBytes)) != normalizeGoldenNewlines(got) {
		t.Fatalf("golden mismatch for %s", goldenPath)
	}
}

func normalizeGoldenNewlines(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
}

// Contract: C4
func TestGoldenHelpOutput(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	got := h.mustRun("--help").Stdout
	assertGolden(t, "help.golden.txt", got)
}

// Contract: C4
func TestGoldenRunbookSkeleton(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.initSession("golden")
	h.mustRun(append([]string{"run", "--"}, shellEchoCommand("hello")...)...)
	h.stopSession()
	runbook := normalizeRunbook(readFile(t, h.exportLastMD()))
	assertGolden(t, "runbook.golden.txt", runbook)
}
