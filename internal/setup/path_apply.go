package setup

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	setupPathBeginMarker = "# >>> infratrack setup path >>>"
	setupPathEndMarker   = "# <<< infratrack setup path <<<"
)

var (
	readWindowsUserPathFn  = readWindowsUserPath
	writeWindowsUserPathFn = writeWindowsUserPath
	resolvePosixProfileFn  = resolvePosixProfilePath
)

type pathApplyResult struct {
	Changed      bool
	PathEntry    string
	FilesTouched []TouchedFile
	Action       string
	Note         string
}

func ensureUserPathConfigured(binDir string) (pathApplyResult, error) {
	if runtime.GOOS == "windows" {
		return ensureWindowsUserPathConfigured(binDir)
	}
	return ensurePosixUserPathConfigured(binDir)
}

func ensureWindowsUserPathConfigured(binDir string) (pathApplyResult, error) {
	current, err := readWindowsUserPathFn()
	if err != nil {
		return pathApplyResult{}, err
	}
	if PathContainsDir(current, binDir) {
		return pathApplyResult{
			Action: "PATH already configured (no change).",
		}, nil
	}
	next := buildWindowsUserPathValue(current, binDir)
	if err := writeWindowsUserPathFn(next); err != nil {
		return pathApplyResult{}, err
	}
	return pathApplyResult{
		Changed:   true,
		PathEntry: binDir,
		Action:    "Added target bin dir to user PATH.",
		Note:      "Restart terminal to load updated PATH.",
	}, nil
}

func buildWindowsUserPathValue(current, binDir string) string {
	parts := filepath.SplitList(current)
	normalizedTarget := normalizePathForCompare(binDir)
	filtered := make([]string, 0, len(parts)+1)
	filtered = append(filtered, binDir)
	for _, part := range parts {
		p := strings.TrimSpace(part)
		if p == "" {
			continue
		}
		if normalizePathForCompare(p) == normalizedTarget {
			continue
		}
		filtered = append(filtered, p)
	}
	return strings.Join(filtered, string(os.PathListSeparator))
}

func ensurePosixUserPathConfigured(binDir string) (pathApplyResult, error) {
	escapedBinDir, err := quotePOSIXSingle(binDir)
	if err != nil {
		return pathApplyResult{}, err
	}

	profile, err := resolvePosixProfileFn()
	if err != nil {
		return pathApplyResult{}, err
	}
	content, _ := os.ReadFile(profile)
	text := string(content)

	block := fmt.Sprintf("%s\nexport PATH=%s:\"$PATH\"\n%s\n", setupPathBeginMarker, escapedBinDir, setupPathEndMarker)
	updated, changed, err := upsertSetupMarkerBlock(text, block)
	if err != nil {
		return pathApplyResult{}, fmt.Errorf("prepare profile marker block: %w", err)
	}
	if !changed {
		return pathApplyResult{Action: "PATH already configured (no change)."}, nil
	}

	if err := os.MkdirAll(filepath.Dir(profile), 0o755); err != nil {
		return pathApplyResult{}, fmt.Errorf("create profile dir: %w", err)
	}
	if err := os.WriteFile(profile, []byte(updated), 0o600); err != nil {
		return pathApplyResult{}, fmt.Errorf("write profile: %w", err)
	}
	return pathApplyResult{
		Changed:      true,
		PathEntry:    binDir,
		FilesTouched: []TouchedFile{{Path: profile, Marker: setupPathBeginMarker}},
		Action:       fmt.Sprintf("Added PATH marker block to %s.", profile),
		Note:         "Open a new shell session to load updated PATH.",
	}, nil
}

func resolvePosixProfilePath() (string, error) {
	if v := strings.TrimSpace(os.Getenv("INFRATRACK_SETUP_PROFILE_FILE")); v != "" {
		return filepath.Clean(v), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, ".profile"), nil
}

func readWindowsUserPath() (string, error) {
	out, err := runPowershell(`[Environment]::GetEnvironmentVariable('Path', 'User')`)
	if err != nil {
		return "", fmt.Errorf("read user PATH: %w", err)
	}
	return strings.TrimSpace(out), nil
}

func writeWindowsUserPath(pathValue string) error {
	cmd := exec.Command(powershellExePath(), "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", "[Environment]::SetEnvironmentVariable('Path', $env:INFRATRACK_PATH_VALUE, 'User')")
	cmd.Env = append(os.Environ(), "INFRATRACK_PATH_VALUE="+pathValue)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("write user PATH: %s", msg)
	}
	return nil
}

func runPowershell(script string) (string, error) {
	cmd := exec.Command(powershellExePath(), "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", script)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("%s", msg)
	}
	return stdout.String(), nil
}

func powershellExePath() string {
	preferred := `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe`
	if _, err := os.Stat(preferred); err == nil {
		return preferred
	}

	for _, root := range []string{strings.TrimSpace(os.Getenv("SystemRoot")), strings.TrimSpace(os.Getenv("WINDIR"))} {
		if root == "" {
			continue
		}
		candidate := filepath.Join(root, "System32", "WindowsPowerShell", "v1.0", "powershell.exe")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	return preferred
}

func quotePOSIXSingle(v string) (string, error) {
	if strings.Contains(v, "\x00") || strings.Contains(v, "\n") || strings.Contains(v, "\r") {
		return "", errors.New("bin-dir contains unsupported control characters")
	}
	escaped := strings.ReplaceAll(v, `'`, `'\''`)
	return "'" + escaped + "'", nil
}

func upsertSetupMarkerBlock(content, block string) (string, bool, error) {
	begin, end, exists, err := findSetupMarkerSpan(content)
	if err != nil {
		return "", false, err
	}
	if exists {
		current := content[begin:end]
		if normalizeLineEndings(current) == normalizeLineEndings(block) {
			return content, false, nil
		}
		out := content[:begin] + block + content[end:]
		return out, true, nil
	}
	if strings.TrimSpace(content) == "" {
		return block, true, nil
	}
	return strings.TrimRight(content, "\r\n") + "\n\n" + block, true, nil
}

func findSetupMarkerSpan(content string) (int, int, bool, error) {
	begin := strings.Index(content, setupPathBeginMarker)
	end := strings.Index(content, setupPathEndMarker)
	if begin < 0 && end < 0 {
		return 0, 0, false, nil
	}
	if begin < 0 || end < 0 || end < begin {
		return 0, 0, false, errors.New("setup marker block is malformed")
	}
	nextBegin := strings.Index(content[begin+len(setupPathBeginMarker):], setupPathBeginMarker)
	if nextBegin >= 0 {
		return 0, 0, false, errors.New("multiple setup marker blocks found")
	}
	end += len(setupPathEndMarker)
	for end < len(content) && (content[end] == '\r' || content[end] == '\n') {
		end++
	}
	return begin, end, true, nil
}

func normalizeLineEndings(v string) string {
	return strings.ReplaceAll(v, "\r\n", "\n")
}
