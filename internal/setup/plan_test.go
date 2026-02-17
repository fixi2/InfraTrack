package setup

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBuildPlanDryRun(t *testing.T) {
	t.Setenv("PATH", "")

	plan, err := BuildPlan(PlanInput{
		Scope:      ScopeUser,
		BinDir:     t.TempDir(),
		NoPath:     false,
		Completion: CompletionNone,
	})
	if err != nil {
		t.Fatalf("BuildPlan failed: %v", err)
	}
	if len(plan.Actions) == 0 {
		t.Fatalf("expected actions in plan")
	}
	if len(plan.Notes) != 0 {
		t.Fatalf("expected no notes in compact plan, got %v", plan.Notes)
	}
}

func TestBuildStatusUsesState(t *testing.T) {
	appData := t.TempDir()
	t.Setenv("APPDATA", appData)
	t.Setenv("XDG_CONFIG_HOME", appData)

	statePath, err := DefaultStatePath()
	if err != nil {
		t.Fatalf("DefaultStatePath failed: %v", err)
	}
	if err := SaveState(statePath, StateFile{
		SchemaVersion:    StateSchemaVersion,
		InstalledBinPath: "C:\\Users\\me\\AppData\\Local\\InfraTrack\\bin\\infratrack.exe",
		PendingFinalize:  true,
		Timestamp:        time.Now().UTC(),
	}); err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	binDir := t.TempDir()
	targetBinary := ResolveTargetBinaryPath(binDir)
	if err := os.WriteFile(targetBinary, []byte("x"), 0o600); err != nil {
		t.Fatalf("write target binary failed: %v", err)
	}
	t.Setenv("PATH", binDir)

	status, err := BuildStatus(ScopeUser, binDir)
	if err != nil {
		t.Fatalf("BuildStatus failed: %v", err)
	}
	if !status.Installed {
		t.Fatalf("expected installed=true")
	}
	if !status.PathOK {
		t.Fatalf("expected pathOk=true")
	}
	if !status.StateFound {
		t.Fatalf("expected stateFound=true")
	}
	if !status.PendingFinalize {
		t.Fatalf("expected pendingFinalize=true")
	}
}

func TestBuildStatusReadsWindowsUserPath(t *testing.T) {
	if os.PathSeparator != '\\' {
		t.Skip("windows-only")
	}

	prevRead := readWindowsUserPathFn
	defer func() { readWindowsUserPathFn = prevRead }()

	binDir := t.TempDir()
	readWindowsUserPathFn = func() (string, error) {
		return strings.Join([]string{`C:\Tools`, binDir}, ";"), nil
	}

	status, err := BuildStatus(ScopeUser, binDir)
	if err != nil {
		t.Fatalf("BuildStatus failed: %v", err)
	}
	if !status.PathOK {
		t.Fatalf("expected pathOk=true from user PATH")
	}
}

func TestPathContainsDirNormalizesWindowsCaseAndSlashes(t *testing.T) {
	if os.PathSeparator != '\\' {
		t.Skip("windows-only normalization assertion")
	}
	pathEnv := `C:\Tools;C:/Users/Vladislav/AppData/Local/InfraTrack/bin/;D:\Other`
	if !PathContainsDir(pathEnv, `c:\users\vladislav\appdata\local\infratrack\bin`) {
		t.Fatalf("expected normalized path match")
	}
}

func TestStateRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "setup-state.json")
	in := StateFile{
		SchemaVersion:    StateSchemaVersion,
		CreatedDirs:      []string{"a", "b"},
		InstalledBinPath: "bin/infratrack",
		PathEntryAdded:   "bin",
		FilesTouched:     []TouchedFile{{Path: "profile", Marker: "BEGIN"}},
		PendingFinalize:  false,
		Timestamp:        time.Now().UTC(),
	}
	if err := SaveState(path, in); err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}
	out, found, err := LoadState(path)
	if err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}
	if !found {
		t.Fatalf("expected found=true")
	}
	blob, _ := json.Marshal(out)
	if !strings.Contains(string(blob), `"schemaVersion":1`) {
		t.Fatalf("expected schemaVersion in saved state: %s", string(blob))
	}
}
