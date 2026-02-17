package setup

import (
	"bytes"
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
	profile, err := resolvePosixProfileFn()
	if err != nil {
		return pathApplyResult{}, err
	}
	content, _ := os.ReadFile(profile)
	text := string(content)
	if strings.Contains(text, setupPathBeginMarker) && strings.Contains(text, setupPathEndMarker) {
		return pathApplyResult{
			Action: "PATH already configured (no change).",
		}, nil
	}
	block := fmt.Sprintf("%s\nexport PATH=\"%s:$PATH\"\n%s\n", setupPathBeginMarker, binDir, setupPathEndMarker)
	if strings.TrimSpace(text) == "" {
		text = block
	} else {
		text = strings.TrimRight(text, "\r\n") + "\n\n" + block
	}
	if err := os.MkdirAll(filepath.Dir(profile), 0o755); err != nil {
		return pathApplyResult{}, fmt.Errorf("create profile dir: %w", err)
	}
	if err := os.WriteFile(profile, []byte(text), 0o600); err != nil {
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
	script := fmt.Sprintf("[Environment]::SetEnvironmentVariable('Path', \"%s\", 'User')", escapePowerShellDoubleQuoted(pathValue))
	if _, err := runPowershell(script); err != nil {
		return fmt.Errorf("write user PATH: %w", err)
	}
	return nil
}

func runPowershell(script string) (string, error) {
	cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", script)
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

func escapePowerShellDoubleQuoted(v string) string {
	v = strings.ReplaceAll(v, "`", "``")
	v = strings.ReplaceAll(v, "\"", "`\"")
	return v
}
