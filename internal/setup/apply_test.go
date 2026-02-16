package setup

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestApplyInstallsBinaryAndWritesState(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", root)
	t.Setenv("APPDATA", filepath.Join(root, "AppData", "Roaming"))

	source := filepath.Join(root, "source-bin")
	if runtime.GOOS == "windows" {
		source += ".exe"
	}
	if err := os.WriteFile(source, []byte("infratrack-binary"), 0o700); err != nil {
		t.Fatalf("write source failed: %v", err)
	}

	binDir := filepath.Join(root, "bin")
	result, err := Apply(ApplyInput{
		Scope:            ScopeUser,
		BinDir:           binDir,
		NoPath:           true,
		Completion:       CompletionNone,
		SourceBinaryPath: source,
	})
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
	if result.InstalledBinPath == "" {
		t.Fatalf("expected installed binary path")
	}
	if _, err := os.Stat(result.InstalledBinPath); err != nil {
		t.Fatalf("installed binary not found: %v", err)
	}
	if _, found, err := LoadState(result.StatePath); err != nil || !found {
		t.Fatalf("expected state file (found=%v, err=%v)", found, err)
	}
}

func TestApplyWindowsStagingName(t *testing.T) {
	target := `C:\Users\me\AppData\Local\InfraTrack\bin\infratrack.exe`
	got := windowsStagingPath(target)
	if !strings.HasSuffix(strings.ToLower(got), "infratrack.new.exe") {
		t.Fatalf("unexpected staging path: %s", got)
	}
}
