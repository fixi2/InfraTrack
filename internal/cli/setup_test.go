package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fixi2/InfraTrack/internal/setup"
)

func TestSetupCommandsExist(t *testing.T) {
	t.Parallel()

	root, err := NewRootCommand()
	if err != nil {
		t.Fatalf("NewRootCommand failed: %v", err)
	}

	setupCmd, _, err := root.Find([]string{"setup"})
	if err != nil {
		t.Fatalf("root.Find(setup) failed: %v", err)
	}
	if setupCmd == nil || setupCmd.Name() != "setup" {
		t.Fatalf("setup command not found")
	}

	setupStatus, _, err := root.Find([]string{"setup", "status"})
	if err != nil {
		t.Fatalf("root.Find(setup status) failed: %v", err)
	}
	if setupStatus == nil || setupStatus.Name() != "status" {
		t.Fatalf("setup status command not found")
	}

	setupPlan, _, err := root.Find([]string{"setup", "plan"})
	if err != nil {
		t.Fatalf("root.Find(setup plan) failed: %v", err)
	}
	if setupPlan == nil || setupPlan.Name() != "plan" {
		t.Fatalf("setup plan command not found")
	}

	setupApply, _, err := root.Find([]string{"setup", "apply"})
	if err != nil {
		t.Fatalf("root.Find(setup apply) failed: %v", err)
	}
	if setupApply == nil || setupApply.Name() != "apply" {
		t.Fatalf("setup apply command not found")
	}
}

func TestSetupPlanOutput(t *testing.T) {
	t.Parallel()

	root, err := NewRootCommand()
	if err != nil {
		t.Fatalf("NewRootCommand failed: %v", err)
	}
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"setup", "plan", "--scope", "user", "--completion", "none"})

	if err := root.Execute(); err != nil {
		t.Fatalf("setup plan failed: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "=== Setup Plan (Dry Run) ===") {
		t.Fatalf("expected dry-run title, got: %s", got)
	}
	if !strings.Contains(got, "Actions:") {
		t.Fatalf("expected actions section, got: %s", got)
	}
	if !strings.Contains(got, "No changes were made.") {
		t.Fatalf("expected no-change note, got: %s", got)
	}
}

func TestSetupStatusJSON(t *testing.T) {
	t.Parallel()

	root, err := NewRootCommand()
	if err != nil {
		t.Fatalf("NewRootCommand failed: %v", err)
	}
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"setup", "status", "--json"})

	if err := root.Execute(); err != nil {
		t.Fatalf("setup status --json failed: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("status json decode failed: %v\npayload:\n%s", err, out.String())
	}
	if _, ok := decoded["installed"]; !ok {
		t.Fatalf("expected installed field in json: %v", decoded)
	}
	if _, ok := decoded["pathOk"]; !ok {
		t.Fatalf("expected pathOk field in json: %v", decoded)
	}
}

func TestSetupApplyYes(t *testing.T) {
	root, err := NewRootCommand()
	if err != nil {
		t.Fatalf("NewRootCommand failed: %v", err)
	}

	stateRoot := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", stateRoot)
	t.Setenv("APPDATA", stateRoot)

	binDir := t.TempDir()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"setup", "apply", "--yes", "--bin-dir", binDir, "--no-path"})

	if err := root.Execute(); err != nil {
		t.Fatalf("setup apply failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "[OK] Setup complete") {
		t.Fatalf("expected concise success output, got: %s", got)
	}
	if !strings.Contains(got, "- binary: ") {
		t.Fatalf("expected binary summary line, got: %s", got)
	}
	if !strings.Contains(got, "Use `infratrack setup status` for details.") {
		t.Fatalf("expected follow-up status hint, got: %s", got)
	}
}

func TestSetupCommandShowsQuickFlowAndCancels(t *testing.T) {
	t.Parallel()

	root, err := NewRootCommand()
	if err != nil {
		t.Fatalf("NewRootCommand failed: %v", err)
	}

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetIn(strings.NewReader("n\n"))
	root.SetArgs([]string{"setup", "--scope", "user", "--completion", "none", "--no-path"})

	if err := root.Execute(); err != nil {
		t.Fatalf("setup command failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Will install to:") {
		t.Fatalf("expected install destination line, got: %s", got)
	}
	if !strings.Contains(got, "Cancelled.") {
		t.Fatalf("expected cancellation output, got: %s", got)
	}
}

func TestSetupCommandYesShowsOnlyFinalConciseMessage(t *testing.T) {
	root, err := NewRootCommand()
	if err != nil {
		t.Fatalf("NewRootCommand failed: %v", err)
	}

	stateRoot := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", stateRoot)
	t.Setenv("APPDATA", stateRoot)

	exePath, err := os.Executable()
	if err != nil {
		t.Fatalf("resolve test executable path failed: %v", err)
	}
	binDir := filepath.Dir(exePath)
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"setup", "--yes", "--bin-dir", binDir, "--no-path"})

	if err := root.Execute(); err != nil {
		t.Fatalf("setup --yes failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "[OK] Setup complete") && !strings.Contains(got, "[WARN] Setup staged") {
		t.Fatalf("expected setup completion/staged output, got: %s", got)
	}
	if strings.Contains(got, "Binary: ") {
		t.Fatalf("did not expect binary summary in setup output, got: %s", got)
	}
	if strings.Count(got, "PATH:") != 1 {
		t.Fatalf("expected single path line in setup output, got: %s", got)
	}
}

func TestSetupUndoNoState(t *testing.T) {
	root, err := NewRootCommand()
	if err != nil {
		t.Fatalf("NewRootCommand failed: %v", err)
	}

	stateRoot := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", stateRoot)
	t.Setenv("APPDATA", stateRoot)

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"setup", "undo"})

	if err := root.Execute(); err != nil {
		t.Fatalf("setup undo failed: %v", err)
	}
	if !strings.Contains(out.String(), "Nothing to undo.") {
		t.Fatalf("expected no-op undo output, got: %s", out.String())
	}
}

func TestSetupUndoRevertsState(t *testing.T) {
	root, err := NewRootCommand()
	if err != nil {
		t.Fatalf("NewRootCommand failed: %v", err)
	}

	stateRoot := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", stateRoot)
	t.Setenv("APPDATA", stateRoot)

	binDir := filepath.Join(stateRoot, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin failed: %v", err)
	}
	binPath := setup.ResolveTargetBinaryPath(binDir)
	if err := os.WriteFile(binPath, []byte("x"), 0o700); err != nil {
		t.Fatalf("write bin failed: %v", err)
	}

	statePath, err := setup.DefaultStatePath()
	if err != nil {
		t.Fatalf("DefaultStatePath failed: %v", err)
	}
	if err := setup.SaveState(statePath, setup.StateFile{
		SchemaVersion:    setup.StateSchemaVersion,
		InstalledBinPath: binPath,
		Timestamp:        time.Now().UTC(),
	}); err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"setup", "undo"})

	if err := root.Execute(); err != nil {
		t.Fatalf("setup undo failed: %v", err)
	}
	if !strings.Contains(out.String(), "[OK] Setup changes reverted") {
		t.Fatalf("expected successful undo output, got: %s", out.String())
	}
}
