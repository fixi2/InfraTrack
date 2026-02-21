package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/fixi2/InfraTrack/internal/store"
	"github.com/spf13/cobra"
)

func newDoctorCmd(s store.SessionStore) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Run local diagnostics for InfraTrack setup",
		RunE: func(cmd *cobra.Command, _ []string) error {
			root := s.RootDir()
			configPath := filepath.Join(root, "config.yaml")
			sessionsPath := filepath.Join(root, "sessions.jsonl")
			activeSessionPath := filepath.Join(root, "active_session.json")

			fmt.Fprintln(cmd.OutOrStdout(), "InfraTrack doctor")
			fmt.Fprintln(cmd.OutOrStdout())
			fmt.Fprintf(cmd.OutOrStdout(), "OS: %s/%s\n", runtime.GOOS, runtime.GOARCH)
			fmt.Fprintf(cmd.OutOrStdout(), "Root dir: %s\n", root)
			fmt.Fprintf(cmd.OutOrStdout(), "Config file: %s\n", configPath)
			fmt.Fprintf(cmd.OutOrStdout(), "Sessions store: %s\n", sessionsPath)
			fmt.Fprintf(cmd.OutOrStdout(), "Active session file: %s\n", activeSessionPath)

			initialized, err := s.IsInitialized(cmd.Context())
			if err != nil {
				return fmt.Errorf("doctor: check initialization: %w", err)
			}
			if initialized {
				printOK(cmd.OutOrStdout(), "Initialization: OK")
			} else {
				printWarn(cmd.OutOrStdout(), "Initialization: NOT INITIALIZED")
				printHint(cmd.OutOrStdout(), "Run `infratrack init`.")
			}

			if err := ensureWritable(root); err != nil {
				printError(cmd.OutOrStdout(), "Writable check: FAILED (%v)", err)
			} else {
				printOK(cmd.OutOrStdout(), "Writable check: OK")
			}

			if path, err := exec.LookPath("infratrack"); err != nil {
				printWarn(cmd.OutOrStdout(), "Command lookup (`infratrack`): NOT FOUND in PATH")
				if runtime.GOOS == "windows" {
					printHint(cmd.OutOrStdout(), "If installed with winget, open a new terminal session.")
				}
			} else {
				printOK(cmd.OutOrStdout(), "Command lookup (`infratrack`): OK (%s)", path)
			}

			if runtime.GOOS == "windows" {
				localAppData := os.Getenv("LOCALAPPDATA")
				if localAppData != "" {
					windowsApps := filepath.Join(localAppData, "Microsoft", "WindowsApps")
					if !pathContainsDir(os.Getenv("PATH"), windowsApps) {
						fmt.Fprintf(cmd.OutOrStdout(), "WindowsApps PATH check: MISSING (%s)\n", windowsApps)
						fmt.Fprintln(cmd.OutOrStdout(), "Hint: add WindowsApps to PATH or reinstall App Installer.")
					} else {
						fmt.Fprintln(cmd.OutOrStdout(), "WindowsApps PATH check: OK")
					}
				}
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Tool availability:")
			for _, tool := range []string{"kubectl", "docker", "terraform"} {
				if path, err := exec.LookPath(tool); err != nil {
					fmt.Fprintf(cmd.OutOrStdout(), "- %s: MISSING\n", tool)
				} else {
					fmt.Fprintf(cmd.OutOrStdout(), "- %s: OK (%s)\n", tool, path)
				}
			}

			return nil
		},
	}
}

func ensureWritable(root string) error {
	if err := os.MkdirAll(root, 0o700); err != nil {
		return fmt.Errorf("create root dir: %w", err)
	}
	file, err := os.CreateTemp(root, "doctor-write-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	path := file.Name()
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("remove temp file: %w", err)
	}
	return nil
}

func pathContainsDir(pathEnv, target string) bool {
	if target == "" {
		return false
	}
	target = strings.ToLower(filepath.Clean(target))
	for _, part := range filepath.SplitList(pathEnv) {
		if strings.ToLower(filepath.Clean(strings.TrimSpace(part))) == target {
			return true
		}
	}
	return false
}
