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

	"github.com/spf13/cobra"
)

const (
	psHookBeginMarker = "# >>> infratrack hooks >>>"
	psHookEndMarker   = "# <<< infratrack hooks <<<"
)

func newHooksInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install shell hooks",
	}
	cmd.AddCommand(newHooksInstallPowerShellCmd())
	return cmd
}

func newHooksUninstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall shell hooks",
	}
	cmd.AddCommand(newHooksUninstallPowerShellCmd())
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
			path, err := choosePowerShellProfileForInstall()
			if err != nil {
				return err
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

			current, err := readTextFile(path)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("read profile: %w", err)
			}
			updated, changed, err := upsertPowerShellHookBlock(current)
			if err != nil {
				return err
			}
			if err := writeTextFileAtomic(path, updated); err != nil {
				return fmt.Errorf("write profile: %w", err)
			}

			if changed {
				fmt.Fprintf(cmd.OutOrStdout(), "Installed PowerShell hooks in %s\n", path)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "PowerShell hooks already installed in %s\n", path)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Open a new PowerShell session to activate the hook.")
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
				fmt.Fprintf(cmd.OutOrStdout(), "Removed PowerShell hooks from %s\n", path)
			}

			if !foundAny || !removedAny {
				fmt.Fprintln(cmd.OutOrStdout(), "No PowerShell hook block found.")
			}
			return nil
		},
	}
}

func confirmInstall(cmd *cobra.Command) (bool, error) {
	fmt.Fprintln(cmd.OutOrStdout(), "This will update your PowerShell profile to add InfraTrack hook capture.")
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
			if strings.Contains(content, psHookBeginMarker) && strings.Contains(content, psHookEndMarker) {
				state = "installed"
				installedAny = true
			} else {
				state = "present (no infratrack block)"
			}
		}
		lines = append(lines, fmt.Sprintf("- %s: %s", path, state))
	}
	return installedAny, strings.Join(lines, "\n")
}

func choosePowerShellProfileForInstall() (string, error) {
	candidates := powerShellProfileCandidates()
	if len(candidates) == 0 {
		return "", errors.New("cannot resolve PowerShell profile path")
	}
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return candidates[0], nil
}

func powerShellProfileCandidates() []string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return nil
	}
	return []string{
		filepath.Join(home, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1"),
		filepath.Join(home, "Documents", "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1"),
	}
}

func upsertPowerShellHookBlock(content string) (string, bool, error) {
	block := powerShellHookBlock()
	if strings.Contains(content, psHookBeginMarker) && strings.Contains(content, psHookEndMarker) {
		start := strings.Index(content, psHookBeginMarker)
		finish := strings.Index(content, psHookEndMarker)
		if start == -1 || finish == -1 || finish < start {
			return "", false, errors.New("hook block markers are malformed")
		}
		afterEnd := finish + len(psHookEndMarker)
		existing := content[start:afterEnd]
		if existing == block {
			return content, false, nil
		}
		updated := content[:start] + block + content[afterEnd:]
		return updated, true, nil
	}

	if content == "" {
		return block + "\n", true, nil
	}
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return content + "\n" + block + "\n", true, nil
}

func removePowerShellHookBlock(content string) (string, bool, error) {
	return replaceBetweenMarkers(content, psHookBeginMarker, psHookEndMarker, "")
}

func replaceBetweenMarkers(content, begin, end, replacement string) (string, bool, error) {
	start := strings.Index(content, begin)
	finish := strings.Index(content, end)
	if start == -1 && finish == -1 {
		return content, false, nil
	}
	if start == -1 || finish == -1 || finish < start {
		return "", false, errors.New("hook block markers are malformed")
	}
	afterEnd := finish + len(end)
	left := content[:start]
	right := content[afterEnd:]

	if strings.HasPrefix(right, "\r\n") {
		right = strings.TrimPrefix(right, "\r\n")
	} else if strings.HasPrefix(right, "\n") {
		right = strings.TrimPrefix(right, "\n")
	}
	if replacement == "" {
		updated := strings.TrimRight(left, "\r\n")
		if right != "" {
			if updated != "" {
				updated += "\n"
			}
			updated += strings.TrimLeft(right, "\r\n")
		}
		return updated, true, nil
	}

	updated := strings.TrimRight(left, "\r\n")
	if updated != "" {
		updated += "\n\n"
	}
	updated += replacement
	if right != "" {
		updated += "\n\n" + strings.TrimLeft(right, "\r\n")
	}
	return updated, updated != content, nil
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

func powerShellHookBlock() string {
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
		"    & infratrack hook record --command $infraTrackHist.CommandLine --exit-code $infraTrackExit --duration-ms 0 --cwd $infraTrackCwd 2>$null",
		"  }",
		"  $infraTrackPrefix = \"\"",
		"  try {",
		"    $infraTrackRoot = Join-Path $env:APPDATA \"infratrack\"",
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
