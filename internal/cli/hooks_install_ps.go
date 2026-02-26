package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/fixi2/InfraTrack/internal/textblock"
	"github.com/spf13/cobra"
)

const (
	psHookBeginMarker = "# >>> commandry hooks >>>"
	psHookEndMarker   = "# <<< commandry hooks <<<"

	legacyPSHookBeginMarker = "# >>> infratrack hooks >>>"
	legacyPSHookEndMarker   = "# <<< infratrack hooks <<<"
)

func newHooksInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install shell hooks",
	}
	cmd.AddCommand(newHooksInstallPowerShellCmd(), newHooksInstallBashCmd(), newHooksInstallZshCmd())
	return cmd
}

func newHooksUninstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall shell hooks",
	}
	cmd.AddCommand(newHooksUninstallPowerShellCmd(), newHooksUninstallBashCmd(), newHooksUninstallZshCmd())
	return cmd
}

func newHooksInstallPowerShellCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "powershell",
		Short: "Install PowerShell profile hook",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if runtime.GOOS != "windows" {
				return errors.New("powershell hooks install is supported on Windows in this stage")
			}
			candidates := powerShellProfileCandidates()
			if len(candidates) == 0 {
				return errors.New("cannot resolve PowerShell profile path")
			}
			if !yes {
				ok, err := confirmInstall(cmd)
				if err != nil {
					return err
				}
				if !ok {
					fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
					return nil
				}
			}

			exe, err := os.Executable()
			if err != nil {
				return fmt.Errorf("resolve executable path: %w", err)
			}
			for _, path := range candidates {
				current, err := readTextFile(path)
				if err != nil && !errors.Is(err, os.ErrNotExist) {
					return fmt.Errorf("read profile %s: %w", path, err)
				}
				updated, changed, err := upsertPowerShellHookBlock(current, exe)
				if err != nil {
					return err
				}
				if err := writeTextFileAtomic(path, updated); err != nil {
					return fmt.Errorf("write profile %s: %w", path, err)
				}
				if changed {
					printOK(cmd.OutOrStdout(), "Installed PowerShell hooks in %s", path)
				} else {
					printWarn(cmd.OutOrStdout(), "PowerShell hooks already installed in %s", path)
				}
			}
			printHint(cmd.OutOrStdout(), "Open a new PowerShell session to activate the hook.")
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}

func newHooksUninstallPowerShellCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "powershell",
		Short: "Remove PowerShell profile hook",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if runtime.GOOS != "windows" {
				return errors.New("powershell hooks uninstall is supported on Windows in this stage")
			}

			foundAny := false
			removedAny := false
			for _, path := range powerShellProfileCandidates() {
				current, err := readTextFile(path)
				if err != nil {
					if errors.Is(err, os.ErrNotExist) {
						continue
					}
					return fmt.Errorf("read profile %s: %w", path, err)
				}
				foundAny = true
				updated, changed, err := removePowerShellHookBlock(current)
				if err != nil {
					return fmt.Errorf("remove hook block from %s: %w", path, err)
				}
				if !changed {
					continue
				}
				if err := writeTextFileAtomic(path, updated); err != nil {
					return fmt.Errorf("write profile %s: %w", path, err)
				}
				removedAny = true
				printOK(cmd.OutOrStdout(), "Removed PowerShell hooks from %s", path)
			}

			if !foundAny || !removedAny {
				printWarn(cmd.OutOrStdout(), "No PowerShell hook block found.")
			}
			return nil
		},
	}
}

func confirmInstall(cmd *cobra.Command) (bool, error) {
	fmt.Fprintln(cmd.OutOrStdout(), "This will update your PowerShell profile to add Commandry hook capture.")
	fmt.Fprintln(cmd.OutOrStdout(), "Type 'y' to continue:")
	reader := bufio.NewReader(cmd.InOrStdin())
	raw, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, fmt.Errorf("read confirmation: %w", err)
	}
	if err != nil && errors.Is(err, io.EOF) && strings.TrimSpace(raw) == "" {
		return false, errors.New("confirmation required; rerun with --yes for non-interactive install")
	}
	answer := strings.ToLower(strings.TrimSpace(raw))
	return answer == "y" || answer == "yes", nil
}

func powerShellInstallStatus() (bool, string) {
	candidates := powerShellProfileCandidates()
	if len(candidates) == 0 {
		return false, ""
	}

	lines := make([]string, 0, len(candidates))
	installedAny := false
	for _, path := range candidates {
		state := "not found"
		content, err := readTextFile(path)
		if err == nil {
			if (strings.Contains(content, psHookBeginMarker) && strings.Contains(content, psHookEndMarker)) ||
				(strings.Contains(content, legacyPSHookBeginMarker) && strings.Contains(content, legacyPSHookEndMarker)) {
				state = "installed"
				installedAny = true
			} else {
				state = "present (no commandry block)"
			}
		}
		lines = append(lines, fmt.Sprintf("- %s: %s", path, state))
	}
	return installedAny, strings.Join(lines, "\n")
}

func powerShellProfileCandidates() []string {
	home, err := hooksHomeDir()
	if err != nil || home == "" {
		return nil
	}
	return []string{
		filepath.Join(home, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1"),
		filepath.Join(home, "Documents", "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1"),
	}
}

func hooksHomeDir() (string, error) {
	if override := strings.TrimSpace(os.Getenv("INFRATRACK_HOME_DIR")); override != "" {
		return override, nil
	}
	return os.UserHomeDir()
}

func upsertPowerShellHookBlock(content, executablePath string) (string, bool, error) {
	block := powerShellHookBlock(executablePath)
	if strings.Contains(content, legacyPSHookBeginMarker) && strings.Contains(content, legacyPSHookEndMarker) &&
		(!strings.Contains(content, psHookBeginMarker) || !strings.Contains(content, psHookEndMarker)) {
		cleaned, changedLegacy, legacyErr := textblock.Remove(content, legacyPSHookBeginMarker, legacyPSHookEndMarker)
		if legacyErr != nil {
			return "", false, errors.New("hook block markers are malformed")
		}
		content = cleaned
		updated, changed, err := textblock.Upsert(content, psHookBeginMarker, psHookEndMarker, block)
		if err != nil {
			return "", false, errors.New("hook block markers are malformed")
		}
		return updated, changed || changedLegacy, nil
	}
	updated, changed, err := textblock.Upsert(content, psHookBeginMarker, psHookEndMarker, block)
	if err != nil {
		return "", false, errors.New("hook block markers are malformed")
	}
	return updated, changed, nil
}

func removePowerShellHookBlock(content string) (string, bool, error) {
	updated, changed, err := textblock.Remove(content, psHookBeginMarker, psHookEndMarker)
	if err != nil {
		return "", false, errors.New("hook block markers are malformed")
	}
	if !changed {
		legacyUpdated, legacyChanged, legacyErr := textblock.Remove(content, legacyPSHookBeginMarker, legacyPSHookEndMarker)
		if legacyErr != nil {
			return "", false, errors.New("hook block markers are malformed")
		}
		updated = legacyUpdated
		changed = legacyChanged
	}
	return updated, changed, nil
}

func replaceBetweenMarkers(content, begin, end, replacement string) (string, bool, error) {
	if replacement != "" {
		return "", false, errors.New("replacement mode is not supported")
	}
	updated, changed, err := textblock.Remove(content, begin, end)
	if err != nil {
		return "", false, errors.New("hook block markers are malformed")
	}
	return updated, changed, nil
}

func readTextFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func writeTextFileAtomic(path, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create parent dir: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(content), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func powerShellHookBlock(executablePath string) string {
	escapedPath := strings.ReplaceAll(executablePath, "'", "''")
	return strings.Join([]string{
		psHookBeginMarker,
		"if (-not $global:InfraTrackOriginalPrompt) {",
		"  $global:InfraTrackOriginalPrompt = $function:prompt",
		"}",
		"if (-not (Get-Variable -Name InfraTrackLastHistoryId -Scope Global -ErrorAction SilentlyContinue)) {",
		"  $global:InfraTrackLastHistoryId = -1",
		"}",
		"function global:prompt {",
		"  $infraTrackCwd = (Get-Location).Path",
		"  $infraTrackExit = $LASTEXITCODE",
		"  $infraTrackHist = Get-History -Count 1 -ErrorAction SilentlyContinue",
		"  if ($infraTrackHist -and $infraTrackHist.Id -ne $global:InfraTrackLastHistoryId) {",
		"    $global:InfraTrackLastHistoryId = $infraTrackHist.Id",
		"    if ($infraTrackHist.CommandLine -notmatch '^\\s*(cmdry(\\.exe)?|cmdr|infratrack(\\.exe)?|it)\\b') {",
		fmt.Sprintf("      & '%s' hook record --command $infraTrackHist.CommandLine --exit-code $infraTrackExit --duration-ms 0 --cwd $infraTrackCwd 2>$null", escapedPath),
		"    }",
		"  }",
		"  $infraTrackPrefix = \"\"",
		"  try {",
		"    $infraTrackRoot = Join-Path $env:APPDATA \"commandry\"",
		"    if (-not (Test-Path $infraTrackRoot -PathType Container)) {",
		"      $legacyRoot = Join-Path $env:APPDATA \"infratrack\"",
		"      if (Test-Path $legacyRoot -PathType Container) {",
		"        $infraTrackRoot = $legacyRoot",
		"      }",
		"    }",
		"    $infraTrackStatePath = Join-Path $infraTrackRoot \"hooks_state.json\"",
		"    $infraTrackActivePath = Join-Path $infraTrackRoot \"active_session.json\"",
		"    if ((Test-Path $infraTrackStatePath -PathType Leaf) -and (Test-Path $infraTrackActivePath -PathType Leaf)) {",
		"      $infraTrackState = Get-Content $infraTrackStatePath -Raw | ConvertFrom-Json",
		"      if ($infraTrackState.Enabled) {",
		"        $infraTrackPrefix = \"[REC] \"",
		"      }",
		"    }",
		"  } catch { }",
		"  if ($global:InfraTrackOriginalPrompt) {",
		"    return \"$infraTrackPrefix$(& $global:InfraTrackOriginalPrompt)\"",
		"  }",
		"  return \"$infraTrackPrefixPS $infraTrackCwd> \"",
		"}",
		psHookEndMarker,
	}, "\n")
}
