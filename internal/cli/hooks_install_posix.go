package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fixi2/InfraTrack/internal/textblock"
	"github.com/spf13/cobra"
)

const (
	bashHookBeginMarker = "# >>> commandry hooks (bash) >>>"
	bashHookEndMarker   = "# <<< commandry hooks (bash) <<<"
	zshHookBeginMarker  = "# >>> commandry hooks (zsh) >>>"
	zshHookEndMarker    = "# <<< commandry hooks (zsh) <<<"

	legacyBashHookBeginMarker = "# >>> infratrack hooks (bash) >>>"
	legacyBashHookEndMarker   = "# <<< infratrack hooks (bash) <<<"
	legacyZshHookBeginMarker  = "# >>> infratrack hooks (zsh) >>>"
	legacyZshHookEndMarker    = "# <<< infratrack hooks (zsh) <<<"
)

func newHooksInstallBashCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "bash",
		Short: "Install bash profile hook",
		RunE: func(cmd *cobra.Command, _ []string) error {
			path, err := bashProfilePath()
			if err != nil {
				return err
			}
			exe, err := os.Executable()
			if err != nil {
				return fmt.Errorf("resolve executable path: %w", err)
			}
			return installPosixHook(cmd, path, bashHookBeginMarker, bashHookEndMarker, bashHookBlock(exe))
		},
	}
}

func newHooksInstallZshCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "zsh",
		Short: "Install zsh profile hook",
		RunE: func(cmd *cobra.Command, _ []string) error {
			path, err := zshProfilePath()
			if err != nil {
				return err
			}
			exe, err := os.Executable()
			if err != nil {
				return fmt.Errorf("resolve executable path: %w", err)
			}
			return installPosixHook(cmd, path, zshHookBeginMarker, zshHookEndMarker, zshHookBlock(exe))
		},
	}
}

func newHooksUninstallBashCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "bash",
		Short: "Remove bash profile hook",
		RunE: func(cmd *cobra.Command, _ []string) error {
			path, err := bashProfilePath()
			if err != nil {
				return err
			}
			return uninstallPosixHook(cmd, path, bashHookBeginMarker, bashHookEndMarker)
		},
	}
}

func newHooksUninstallZshCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "zsh",
		Short: "Remove zsh profile hook",
		RunE: func(cmd *cobra.Command, _ []string) error {
			path, err := zshProfilePath()
			if err != nil {
				return err
			}
			return uninstallPosixHook(cmd, path, zshHookBeginMarker, zshHookEndMarker)
		},
	}
}

func installPosixHook(cmd *cobra.Command, path, begin, end, block string) error {
	current, err := readTextFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("read profile: %w", err)
	}
	updated, changed, err := upsertHookBlock(current, begin, end, block)
	if err != nil {
		return err
	}
	if err := writeTextFileAtomic(path, updated); err != nil {
		return fmt.Errorf("write profile: %w", err)
	}
	if changed {
		printOK(cmd.OutOrStdout(), "Installed hooks in %s", path)
	} else {
		printWarn(cmd.OutOrStdout(), "Hooks already installed in %s", path)
	}
	fmt.Fprintln(cmd.OutOrStdout(), "Open a new shell session to activate the hook.")
	return nil
}

func uninstallPosixHook(cmd *cobra.Command, path, begin, end string) error {
	current, err := readTextFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Fprintln(cmd.OutOrStdout(), "No hook block found.")
			return nil
		}
		return fmt.Errorf("read profile: %w", err)
	}
	updated, changed, err := replaceBetweenMarkers(current, begin, end, "")
	if err != nil {
		return err
	}
	if !changed {
		legacyBegin, legacyEnd := legacyPosixMarkers(begin, end)
		if legacyBegin != "" {
			updatedLegacy, changedLegacy, legacyErr := replaceBetweenMarkers(current, legacyBegin, legacyEnd, "")
			if legacyErr != nil {
				return legacyErr
			}
			updated = updatedLegacy
			changed = changedLegacy
		}
	}
	if !changed {
		fmt.Fprintln(cmd.OutOrStdout(), "No hook block found.")
		return nil
	}
	if err := writeTextFileAtomic(path, updated); err != nil {
		return fmt.Errorf("write profile: %w", err)
	}
	printOK(cmd.OutOrStdout(), "Removed hooks from %s", path)
	return nil
}

func upsertHookBlock(content, begin, end, block string) (string, bool, error) {
	legacyBegin, legacyEnd := legacyPosixMarkers(begin, end)
	if legacyBegin != "" && strings.Contains(content, legacyBegin) && strings.Contains(content, legacyEnd) &&
		(!strings.Contains(content, begin) || !strings.Contains(content, end)) {
		cleaned, changedLegacy, legacyErr := textblock.Remove(content, legacyBegin, legacyEnd)
		if legacyErr != nil {
			return "", false, errors.New("hook block markers are malformed")
		}
		content = cleaned
		updated, changed, err := textblock.Upsert(content, begin, end, block)
		if err != nil {
			return "", false, errors.New("hook block markers are malformed")
		}
		return updated, changed || changedLegacy, nil
	}

	updated, changed, err := textblock.Upsert(content, begin, end, block)
	if err != nil {
		return "", false, errors.New("hook block markers are malformed")
	}
	return updated, changed, nil
}

func bashProfilePath() (string, error) {
	home, err := hooksHomeDir()
	if err != nil || home == "" {
		return "", errors.New("cannot resolve home directory for bash profile")
	}
	return filepath.Join(home, ".bashrc"), nil
}

func zshProfilePath() (string, error) {
	home, err := hooksHomeDir()
	if err != nil || home == "" {
		return "", errors.New("cannot resolve home directory for zsh profile")
	}
	return filepath.Join(home, ".zshrc"), nil
}

func bashInstallStatus() (bool, string) {
	path, err := bashProfilePath()
	if err != nil {
		return false, ""
	}
	state := "not found"
	content, readErr := readTextFile(path)
	if readErr == nil {
		if (strings.Contains(content, bashHookBeginMarker) && strings.Contains(content, bashHookEndMarker)) ||
			(strings.Contains(content, legacyBashHookBeginMarker) && strings.Contains(content, legacyBashHookEndMarker)) {
			state = "installed"
		} else {
			state = "present (no commandry block)"
		}
	}
	return state == "installed", fmt.Sprintf("- %s: %s", path, state)
}

func zshInstallStatus() (bool, string) {
	path, err := zshProfilePath()
	if err != nil {
		return false, ""
	}
	state := "not found"
	content, readErr := readTextFile(path)
	if readErr == nil {
		if (strings.Contains(content, zshHookBeginMarker) && strings.Contains(content, zshHookEndMarker)) ||
			(strings.Contains(content, legacyZshHookBeginMarker) && strings.Contains(content, legacyZshHookEndMarker)) {
			state = "installed"
		} else {
			state = "present (no commandry block)"
		}
	}
	return state == "installed", fmt.Sprintf("- %s: %s", path, state)
}

func shellSingleQuote(value string) string {
	return strings.ReplaceAll(value, "'", "'\"'\"'")
}

func bashHookBlock(executablePath string) string {
	exe := shellSingleQuote(executablePath)
	return strings.Join([]string{
		bashHookBeginMarker,
		"__commandry_hook_active=0",
		"__commandry_hook_ready=0",
		"__commandry_should_prefix() {",
		"  local __it_root",
		"  if [ -n \"${APPDATA:-}\" ]; then",
		"    __it_root=\"$APPDATA/commandry\"",
		"  elif [ -n \"${XDG_CONFIG_HOME:-}\" ]; then",
		"    __it_root=\"$XDG_CONFIG_HOME/commandry\"",
		"  elif [ \"$(uname -s 2>/dev/null)\" = \"Darwin\" ]; then",
		"    __it_root=\"$HOME/Library/Application Support/commandry\"",
		"  else",
		"    __it_root=\"$HOME/.config/commandry\"",
		"  fi",
		"  local __it_state=\"$__it_root/hooks_state.json\"",
		"  local __it_active=\"$__it_root/active_session.json\"",
		"  [ -f \"$__it_state\" ] || return 1",
		"  [ -f \"$__it_active\" ] || return 1",
		"  grep -qi '\"enabled\"[[:space:]]*:[[:space:]]*true' \"$__it_state\" 2>/dev/null",
		"}",
		"__commandry_apply_ps1_prefix() {",
		"  [ -n \"${PS1:-}\" ] || return",
		"  if __commandry_should_prefix; then",
		"    case \"$PS1\" in",
		"      \"[REC] \"*) ;;",
		"      *) PS1=\"[REC] $PS1\" ;;",
		"    esac",
		"  else",
		"    case \"$PS1\" in",
		"      \"[REC] \"*) PS1=\"${PS1#\\[REC\\] }\" ;;",
		"    esac",
		"  fi",
		"}",
		"__commandry_hook_record() {",
		"  local __it_exit=$?",
		"  if [ \"${__commandry_hook_active}\" = \"1\" ]; then return; fi",
		"  local __it_cmd=\"$1\"",
		"  [ -z \"$__it_cmd\" ] && return",
		"  case \"$__it_cmd\" in",
		"    cmdry*|cmdr*|it*) return ;;",
		"  esac",
		"  __commandry_hook_active=1",
		fmt.Sprintf("  '%s' hook record --command \"$__it_cmd\" --exit-code \"$__it_exit\" --duration-ms 0 --cwd \"$PWD\" >/dev/null 2>&1 || true", exe),
		"  __commandry_hook_active=0",
		"  __commandry_apply_ps1_prefix",
		"}",
		"__commandry_preexec() {",
		"  [ \"${__commandry_hook_ready}\" = \"1\" ] || return",
		"  local __it_cmd=\"$BASH_COMMAND\"",
		"  case \"$__it_cmd\" in",
		"    __commandry_*|history*|trap*|PROMPT_COMMAND*|\"[ \"*|\"exit\"|\"logout\"|\"\") return ;;",
		"  esac",
		"  __commandry_hook_record \"$__it_cmd\"",
		"}",
		"trap '__commandry_preexec' DEBUG",
		"if [ -n \"${PROMPT_COMMAND:-}\" ]; then",
		"  case \";$PROMPT_COMMAND;\" in",
		"    *\";__commandry_apply_ps1_prefix;\"*) ;;",
		"    *) PROMPT_COMMAND=\"__commandry_apply_ps1_prefix; $PROMPT_COMMAND\" ;;",
		"  esac",
		"else",
		"  PROMPT_COMMAND=\"__commandry_apply_ps1_prefix\"",
		"fi",
		"__commandry_hook_ready=1",
		"__commandry_apply_ps1_prefix",
		bashHookEndMarker,
	}, "\n")
}

func zshHookBlock(executablePath string) string {
	exe := shellSingleQuote(executablePath)
	return strings.Join([]string{
		zshHookBeginMarker,
		"autoload -Uz add-zsh-hook",
		"typeset -g __commandry_hook_active=0",
		"typeset -g __commandry_hook_ready=0",
		"__commandry_should_prefix() {",
		"  local __it_root",
		"  if [[ -n \"${APPDATA:-}\" ]]; then",
		"    __it_root=\"$APPDATA/commandry\"",
		"  elif [[ -n \"${XDG_CONFIG_HOME:-}\" ]]; then",
		"    __it_root=\"$XDG_CONFIG_HOME/commandry\"",
		"  elif [[ \"$(uname -s 2>/dev/null)\" == \"Darwin\" ]]; then",
		"    __it_root=\"$HOME/Library/Application Support/commandry\"",
		"  else",
		"    __it_root=\"$HOME/.config/commandry\"",
		"  fi",
		"  local __it_state=\"$__it_root/hooks_state.json\"",
		"  local __it_active=\"$__it_root/active_session.json\"",
		"  [[ -f \"$__it_state\" ]] || return 1",
		"  [[ -f \"$__it_active\" ]] || return 1",
		"  grep -qi '\"enabled\"[[:space:]]*:[[:space:]]*true' \"$__it_state\" 2>/dev/null",
		"}",
		"__commandry_apply_prompt_prefix() {",
		"  [[ -n \"${PROMPT:-}\" ]] || return",
		"  if __commandry_should_prefix; then",
		"    case \"$PROMPT\" in",
		"      \"[REC] \"*) ;;",
		"      *) PROMPT=\"[REC] $PROMPT\" ;;",
		"    esac",
		"  else",
		"    case \"$PROMPT\" in",
		"      \"[REC] \"*) PROMPT=\"${PROMPT#\\[REC\\] }\" ;;",
		"    esac",
		"  fi",
		"}",
		"__commandry_hook_record() {",
		"  local __it_exit=$?",
		"  if [[ \"$__commandry_hook_active\" == \"1\" ]]; then return; fi",
		"  local __it_cmd=\"$1\"",
		"  [[ -z \"$__it_cmd\" ]] && return",
		"  case \"$__it_cmd\" in",
		"    cmdry*|cmdr*|it*) return ;;",
		"  esac",
		"  __commandry_hook_active=1",
		fmt.Sprintf("  '%s' hook record --command \"$__it_cmd\" --exit-code \"$__it_exit\" --duration-ms 0 --cwd \"$PWD\" >/dev/null 2>&1 || true", exe),
		"  __commandry_hook_active=0",
		"}",
		"__commandry_preexec() {",
		"  [[ \"$__commandry_hook_ready\" == \"1\" ]] || return",
		"  local __it_cmd=\"$1\"",
		"  case \"$__it_cmd\" in",
		"    __commandry_*|\"[ \"*|\"exit\"|\"logout\"|\"\") return ;;",
		"  esac",
		"  __commandry_hook_record \"$__it_cmd\"",
		"}",
		"__commandry_precmd() {",
		"  __commandry_apply_prompt_prefix",
		"}",
		"add-zsh-hook preexec __commandry_preexec",
		"add-zsh-hook precmd __commandry_precmd",
		"__commandry_hook_ready=1",
		"__commandry_apply_prompt_prefix",
		zshHookEndMarker,
	}, "\n")
}

func legacyPosixMarkers(begin, end string) (string, string) {
	switch {
	case begin == bashHookBeginMarker && end == bashHookEndMarker:
		return legacyBashHookBeginMarker, legacyBashHookEndMarker
	case begin == zshHookBeginMarker && end == zshHookEndMarker:
		return legacyZshHookBeginMarker, legacyZshHookEndMarker
	default:
		return "", ""
	}
}
