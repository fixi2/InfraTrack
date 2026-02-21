package blackbox

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

type cliResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

type harness struct {
	t       *testing.T
	binPath string
	rootDir string
	workDir string
	env     []string
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	binPath := resolveBinaryPath(t)
	rootDir := t.TempDir()
	workDir := filepath.Join(rootDir, "work")
	appData := filepath.Join(rootDir, "appdata")
	localAppData := filepath.Join(rootDir, "localappdata")
	homeDir := filepath.Join(rootDir, "home")
	tmpDir := filepath.Join(rootDir, "tmp")
	for _, dir := range []string{workDir, appData, localAppData, homeDir, tmpDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	env := append([]string{}, os.Environ()...)
	env = append(env,
		"APPDATA="+appData,
		"LOCALAPPDATA="+localAppData,
		"USERPROFILE="+homeDir,
		"HOME="+homeDir,
		"XDG_CONFIG_HOME="+appData,
		"XDG_DATA_HOME="+appData,
		"TEMP="+tmpDir,
		"TMP="+tmpDir,
		"TMPDIR="+tmpDir,
		"INFRATRACK_HOME_DIR="+homeDir,
	)

	return &harness{
		t:       t,
		binPath: binPath,
		rootDir: rootDir,
		workDir: workDir,
		env:     env,
	}
}

func (h *harness) run(args ...string) cliResult {
	h.t.Helper()

	cmd := exec.Command(h.binPath, args...)
	cmd.Dir = h.workDir
	cmd.Env = h.env
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	code := 0
	if err != nil {
		var exitErr *exec.ExitError
		if ok := asExitError(err, &exitErr); ok {
			code = exitErr.ExitCode()
		} else {
			h.t.Fatalf("run %v failed: %v", args, err)
		}
	}
	return cliResult{
		ExitCode: code,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
	}
}

func (h *harness) mustRun(args ...string) cliResult {
	h.t.Helper()
	res := h.run(args...)
	if res.ExitCode != 0 {
		h.t.Fatalf("expected success, got exit=%d\nargs=%v\nstdout=%s\nstderr=%s", res.ExitCode, args, res.Stdout, res.Stderr)
	}
	return res
}

func (h *harness) initSession(title string) {
	h.mustRun("init")
	h.mustRun("start", title)
}

func (h *harness) stopSession() cliResult {
	return h.mustRun("stop")
}

func (h *harness) exportLastMD() string {
	h.t.Helper()
	out := h.mustRun("export", "--last", "-f", "md").Stdout
	path := parseRunbookPath(out)
	if path == "" {
		h.t.Fatalf("failed to parse runbook path from output: %s", out)
	}
	if _, err := os.Stat(path); err != nil {
		h.t.Fatalf("runbook does not exist: %s (%v)", path, err)
	}
	return path
}

func parseRunbookPath(output string) string {
	const prefix = "Exported runbook: "
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if idx := strings.Index(line, prefix); idx >= 0 {
			return strings.TrimSpace(line[idx+len(prefix):])
		}
	}
	return ""
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file %s: %v", path, err)
	}
	return string(data)
}

func normalizeRunbook(s string) string {
	lines := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.HasPrefix(line, "Duration: ") {
			out = append(out, "Duration: <normalized> ms")
			continue
		}
		if strings.Contains(line, "echo hello") {
			out = append(out, "<normalized-command>")
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func shellEchoCommand(msg string) []string {
	if runtime.GOOS == "windows" {
		return []string{"cmd", "/c", "echo", msg}
	}
	return []string{"sh", "-lc", "echo " + shellQuote(msg)}
}

func shellExitNonZeroCommand(code int) []string {
	if runtime.GOOS == "windows" {
		return []string{"cmd", "/c", "exit", fmt.Sprintf("%d", code)}
	}
	return []string{"sh", "-lc", fmt.Sprintf("exit %d", code)}
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func asExitError(err error, out **exec.ExitError) bool {
	e, ok := err.(*exec.ExitError)
	if !ok {
		return false
	}
	*out = e
	return true
}

func randomSentinel() string {
	return fmt.Sprintf("IT_SECRET_%d", time.Now().UnixNano())
}

func buildCmdLine(parts ...string) string {
	return strings.Join(parts, " ")
}
