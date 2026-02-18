package blackbox

import (
	"path/filepath"
	"strings"
	"testing"
)

// Contract: C7
func TestSetupLifecycleContract(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	binDir := filepath.Join(h.workDir, "setup-bin")

	plan := h.mustRun("setup", "plan", "--bin-dir", binDir)
	if !strings.Contains(plan.Stdout, "No changes were made.") {
		t.Fatalf("setup plan must be non-mutating preview:\n%s", plan.Stdout)
	}

	apply := h.mustRun("setup", "apply", "--yes", "--bin-dir", binDir, "--no-path")
	if !strings.Contains(apply.Stdout, "[OK] Setup complete") {
		t.Fatalf("setup apply did not complete:\n%s", apply.Stdout)
	}

	status := h.mustRun("setup", "status", "--bin-dir", binDir)
	if !strings.Contains(status.Stdout, "Installed          : OK") {
		t.Fatalf("setup status must report installed OK:\n%s", status.Stdout)
	}

	undo := h.mustRun("setup", "undo")
	if !strings.Contains(undo.Stdout, "[OK] Setup changes reverted") {
		t.Fatalf("setup undo must revert changes:\n%s", undo.Stdout)
	}

	undo2 := h.mustRun("setup", "undo")
	if !strings.Contains(undo2.Stdout, "Nothing to undo.") {
		t.Fatalf("second setup undo must be no-op:\n%s", undo2.Stdout)
	}
}
