package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const (
	bashHookBeginMarker = "# >>> infratrack hooks (bash) >>>"
	bashHookEndMarker   = "# <<< infratrack hooks (bash) <<<"
	zshHookBeginMarker  = "# >>> infratrack hooks (zsh) >>>"
	zshHookEndMarker    = "# <<< infratrack hooks (zsh) <<<"
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
			return installPosixHook(cmd, path, bashHookBeginMarker, bashHookEndMarker, bashHookBlock())
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
			return installPosixHook(cmd, path, zshHookBeginMarker, zshHookEndMarker, zshHookBlock())
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
		fmt.Fprintf(cmd.OutOrStdout(), "Installed hooks in %s\n", path)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Hooks already installed in %s\n", path)
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
		fmt.Fprintln(cmd.OutOrStdout(), "No hook block found.")
		return nil
	}
	if err := writeTextFileAtomic(path, updated); err != nil {
		return fmt.Errorf("write profile: %w", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Removed hooks from %s\n", path)
	return nil
}

func upsertHookBlock(content, begin, end, block string) (string, bool, error) {
	if strings.Contains(content, begin) && strings.Contains(content, end) {
		start := strings.Index(content, begin)
		finish := strings.Index(content, end)
		if start == -1 || finish == -1 || finish < start {
			return "", false, errors.New("hook block markers are malformed")
		}
		afterEnd := finish + len(end)
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
		if strings.Contains(content, bashHookBeginMarker) && strings.Contains(content, bashHookEndMarker) {
			state = "installed"
		} else {
			state = "present (no infratrack block)"
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
		if strings.Contains(content, zshHookBeginMarker) && strings.Contains(content, zshHookEndMarker) {
			state = "installed"
		} else {
			state = "present (no infratrack block)"
		}
	}
	return state == "installed", fmt.Sprintf("- %s: %s", path, state)
}

func bashHookBlock() string {
	return strings.Join([]string{
		bashHookBeginMarker,
		"__infratrack_hook_active=0",
		"__infratrack_hook_record() {",
		"  local __it_exit=$?",
		"  if [ \"${__infratrack_hook_active}\" = \"1\" ]; then return; fi",
		"  local __it_cmd",
		"  __it_cmd=$(history 1 | sed 's/^[ ]*[0-9]\\+[ ]*//')",
		"  [ -z \"$__it_cmd\" ] && return",
		"  case \"$__it_cmd\" in",
		"    infratrack*|it*) return ;;",
		"  esac",
		"  __infratrack_hook_active=1",
		"  infratrack hook record --command \"$__it_cmd\" --exit-code \"$__it_exit\" --duration-ms 0 --cwd \"$PWD\" >/dev/null 2>&1 || true",
		"  __infratrack_hook_active=0",
		"}",
		"if [ -n \"${PROMPT_COMMAND:-}\" ]; then",
		"  case \";$PROMPT_COMMAND;\" in",
		"    *\";__infratrack_hook_record;\"*) ;;",
		"    *) PROMPT_COMMAND=\"__infratrack_hook_record; $PROMPT_COMMAND\" ;;",
		"  esac",
		"else",
		"  PROMPT_COMMAND=\"__infratrack_hook_record\"",
		"fi",
		"if [ -n \"${PS1:-}\" ]; then",
		"  case \"$PS1\" in",
		"    \"[REC] \"*) ;;",
		"    *) PS1=\"[REC] $PS1\" ;;",
		"  esac",
		"fi",
		bashHookEndMarker,
	}, "\n")
}

func zshHookBlock() string {
	return strings.Join([]string{
		zshHookBeginMarker,
		"autoload -Uz add-zsh-hook",
		"typeset -g __infratrack_hook_active=0",
		"__infratrack_precmd() {",
		"  local __it_exit=$?",
		"  if [[ \"$__infratrack_hook_active\" == \"1\" ]]; then return; fi",
		"  local __it_cmd",
		"  __it_cmd=$(fc -ln -1)",
		"  [[ -z \"$__it_cmd\" ]] && return",
		"  case \"$__it_cmd\" in",
		"    infratrack*|it*) return ;;",
		"  esac",
		"  __infratrack_hook_active=1",
		"  infratrack hook record --command \"$__it_cmd\" --exit-code \"$__it_exit\" --duration-ms 0 --cwd \"$PWD\" >/dev/null 2>&1 || true",
		"  __infratrack_hook_active=0",
		"}",
		"add-zsh-hook precmd __infratrack_precmd",
		"if [[ -n \"${PROMPT:-}\" ]]; then",
		"  case \"$PROMPT\" in",
		"    \"[REC] \"*) ;;",
		"    *) PROMPT=\"[REC] $PROMPT\" ;;",
		"  esac",
		"fi",
		zshHookEndMarker,
	}, "\n")
}
