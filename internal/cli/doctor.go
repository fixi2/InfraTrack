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

			fmt.Fprintln(cmd.OutOrStdout(), "=== Doctor ===")
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
				if supportsUnicode(cmd.OutOrStdout()) {
					printOK(cmd.OutOrStdout(), "Initialization")
				} else {
					fmt.Fprintln(cmd.OutOrStdout(), "Initialization: OK")
				}
			} else {
				if supportsUnicode(cmd.OutOrStdout()) {
					printWarn(cmd.OutOrStdout(), "Initialization: not initialized")
				} else {
					fmt.Fprintln(cmd.OutOrStdout(), "Initialization: NOT INITIALIZED")
				}
				printHint(cmd.OutOrStdout(), "Run `infratrack init`.")
			}

			if err := ensureWritable(root); err != nil {
				if supportsUnicode(cmd.OutOrStdout()) {
					printError(cmd.OutOrStdout(), "Writable check failed (%v)", err)
				} else {
					fmt.Fprintf(cmd.OutOrStdout(), "Writable check: FAILED (%v)\n", err)
				}
			} else {
				if supportsUnicode(cmd.OutOrStdout()) {
					printOK(cmd.OutOrStdout(), "Writable check")
				} else {
					fmt.Fprintln(cmd.OutOrStdout(), "Writable check: OK")
				}
			}

			if path, err := exec.LookPath("infratrack"); err != nil {
				if supportsUnicode(cmd.OutOrStdout()) {
					printWarn(cmd.OutOrStdout(), "Command executable `infratrack` in PATH: no")
				} else {
					fmt.Fprintln(cmd.OutOrStdout(), "Command executable `infratrack` in PATH: NO")
				}
				if runtime.GOOS == "windows" {
					printHint(cmd.OutOrStdout(), "If installed with winget, open a new terminal session.")
				}
			} else {
				if supportsUnicode(cmd.OutOrStdout()) {
					printOK(cmd.OutOrStdout(), "Command executable `infratrack` in PATH: yes (%s)", path)
				} else {
					fmt.Fprintf(cmd.OutOrStdout(), "Command executable `infratrack` in PATH: YES (%s)\n", path)
				}
			}

			if runtime.GOOS == "windows" {
				localAppData := os.Getenv("LOCALAPPDATA")
				if localAppData != "" {
					windowsApps := filepath.Join(localAppData, "Microsoft", "WindowsApps")
					if pathContainsDir(os.Getenv("PATH"), windowsApps) {
						fmt.Fprintln(cmd.OutOrStdout(), "WindowsApps in PATH: yes")
					} else {
						fmt.Fprintln(cmd.OutOrStdout(), "WindowsApps in PATH: no")
						printHint(cmd.OutOrStdout(), "Add WindowsApps to PATH or reinstall App Installer.")
					}
				}
			}

			fmt.Fprintln(cmd.OutOrStdout(), "=== Tool Availability ===")
			for _, tool := range []string{"kubectl", "docker", "terraform"} {
				if path, err := exec.LookPath(tool); err != nil {
					fmt.Fprintf(cmd.OutOrStdout(), "- %s (optional): MISSING\n", tool)
				} else {
					fmt.Fprintf(cmd.OutOrStdout(), "- %s (optional): OK (%s)\n", tool, path)
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
