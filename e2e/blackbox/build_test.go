package blackbox

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
)

var (
	updateGolden = flag.Bool("update", false, "update golden files")

	buildOnce sync.Once
	binPath   string
	buildErr  error
)

func resolveBinaryPath(t *testing.T) string {
	t.Helper()
	if override := os.Getenv("INFRATRACK_E2E_BIN"); override != "" {
		return override
	}
	buildOnce.Do(func() {
		tmpDir, err := os.MkdirTemp("", "infratrack-blackbox-bin-*")
		if err != nil {
			buildErr = err
			return
		}
		name := "infratrack"
		if isWindows() {
			name += ".exe"
		}
		binPath = filepath.Join(tmpDir, name)
		gocache := filepath.Join(repoRoot(), ".gocache")
		gotmp := filepath.Join(repoRoot(), ".gotmp")
		gomodcache := filepath.Join(repoRoot(), ".gomodcache")
		_ = os.MkdirAll(gocache, 0o755)
		_ = os.MkdirAll(gotmp, 0o755)
		_ = os.MkdirAll(gomodcache, 0o755)
		cmd := exec.Command("go", "build", "-o", binPath, "./cmd/infratrack")
		cmd.Dir = repoRoot()
		cmd.Env = append(os.Environ(),
			"GOCACHE="+gocache,
			"GOTMPDIR="+gotmp,
			"GOMODCACHE="+gomodcache,
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			buildErr = fmt.Errorf("go build failed: %w\n%s", err, string(out))
			return
		}
	})
	if buildErr != nil {
		t.Fatalf("resolve binary: %v", buildErr)
	}
	return binPath
}

func repoRoot() string {
	wd, _ := os.Getwd()
	// e2e/blackbox -> repo root
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

func isWindows() bool {
	return os.PathSeparator == '\\'
}
