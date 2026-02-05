package cli

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/fixi2/InfraTrack/internal/capture"
	"github.com/fixi2/InfraTrack/internal/export"
	"github.com/fixi2/InfraTrack/internal/policy"
	"github.com/fixi2/InfraTrack/internal/store"
	"github.com/fixi2/InfraTrack/internal/util"
	"github.com/spf13/cobra"
)

func NewRootCommand() (*cobra.Command, error) {
	rootDir, err := store.DefaultRootDir()
	if err != nil {
		return nil, fmt.Errorf("resolve config directory: %w", err)
	}

	s := store.NewJSONStore(rootDir)
	p := policy.NewDefault()

	rootCmd := &cobra.Command{
		Use:   "infratrack",
		Short: "Capture explicit command sessions into deterministic markdown runbooks",
	}

	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true
	rootCmd.AddCommand(
		newInitCmd(s),
		newStartCmd(s),
		newStopCmd(s),
		newStatusCmd(s),
		newRunCmd(s, p),
		newExportCmd(s),
	)

	return rootCmd, nil
}

func newInitCmd(s store.SessionStore) *cobra.Command {
	return &cobra.Command{
		Use:     "init",
		Aliases: []string{"i"},
		Short:   "Initialize local InfraTrack storage and config",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := s.Init(cmd.Context()); err != nil {
				return fmt.Errorf("initialize storage: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Initialized InfraTrack at %s\n", s.RootDir())
			return nil
		},
	}
}

func newStartCmd(s store.SessionStore) *cobra.Command {
	return &cobra.Command{
		Use:     "start <title>",
		Aliases: []string{"s"},
		Short:   "Start a recording session",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			title := strings.TrimSpace(args[0])
			if title == "" {
				return errors.New("title cannot be empty")
			}

			startedAt := time.Now().UTC()
			session, err := s.StartSession(cmd.Context(), title, startedAt)
			if err != nil {
				if errors.Is(err, store.ErrNotInitialized) {
					return errors.New("InfraTrack is not initialized. Run `infratrack init` first")
				}
				if errors.Is(err, store.ErrActiveSessionExists) {
					return errors.New("a session is already active. Run `infratrack stop` before starting a new one")
				}
				return fmt.Errorf("start session: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Started session %q at %s\n", session.Title, session.StartedAt.Format(time.RFC3339))
			return nil
		},
	}
}

func newStopCmd(s store.SessionStore) *cobra.Command {
	return &cobra.Command{
		Use:     "stop",
		Aliases: []string{"stp"},
		Short:   "Stop the active recording session",
		RunE: func(cmd *cobra.Command, _ []string) error {
			endedAt := time.Now().UTC()
			session, err := s.StopSession(cmd.Context(), endedAt)
			if err != nil {
				if errors.Is(err, store.ErrNoActiveSession) {
					return errors.New("no active session. Start one with `infratrack start \"<title>\"`")
				}
				return fmt.Errorf("stop session: %w", err)
			}

			fmt.Fprintf(
				cmd.OutOrStdout(),
				"Stopped session %q with %d recorded step(s)\n",
				session.Title,
				len(session.Steps),
			)
			return nil
		},
	}
}

func newStatusCmd(s store.SessionStore) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current InfraTrack session status",
		RunE: func(cmd *cobra.Command, _ []string) error {
			initialized, err := s.IsInitialized(cmd.Context())
			if err != nil {
				return fmt.Errorf("check initialization: %w", err)
			}

			if !initialized {
				fmt.Fprintln(cmd.OutOrStdout(), "Status: not initialized")
				fmt.Fprintln(cmd.OutOrStdout(), "Run `infratrack init` to create local config and storage")
				return nil
			}

			active, err := s.GetActiveSession(cmd.Context())
			if err != nil {
				if errors.Is(err, store.ErrNoActiveSession) {
					fmt.Fprintln(cmd.OutOrStdout(), "Status: initialized, no active session")
					return nil
				}

				return fmt.Errorf("read active session: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Status: recording\n")
			fmt.Fprintf(cmd.OutOrStdout(), "Title: %s\n", active.Title)
			fmt.Fprintf(cmd.OutOrStdout(), "Started: %s\n", active.StartedAt.Format(time.RFC3339))
			fmt.Fprintf(cmd.OutOrStdout(), "Recorded steps: %d\n", len(active.Steps))

			return nil
		},
	}
}

func newRunCmd(s store.SessionStore, p *policy.Policy) *cobra.Command {
	return &cobra.Command{
		Use:     "run -- <command> [args...]",
		Aliases: []string{"r"},
		Short:   "Execute a command and capture sanitized metadata for the active session",
		Args:    cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("usage: infratrack run -- <command> [args...]")
			}

			if _, err := s.GetActiveSession(cmd.Context()); err != nil {
				if errors.Is(err, store.ErrNoActiveSession) {
					return errors.New("no active session. Run `infratrack start \"<title>\"` before `infratrack run`")
				}
				return fmt.Errorf("check active session: %w", err)
			}

			rawCommand := util.JoinCommand(args)
			sanitized := p.Apply(rawCommand, args)

			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("get working directory: %w", err)
			}

			result, runErr := capture.RunCommand(cmd.Context(), args, cwd)
			step := store.Step{
				Timestamp:  result.StartedAt,
				Command:    sanitized.Command,
				Status:     result.Status,
				Reason:     result.Reason,
				ExitCode:   result.ExitCode,
				DurationMS: result.Duration.Milliseconds(),
				CWD:        cwd,
			}
			if sanitized.Denied {
				step.Status = "REDACTED"
				step.Reason = "policy_redacted"
			}

			if err := s.AddStep(cmd.Context(), step); err != nil {
				return fmt.Errorf("record step: %w", err)
			}

			if runErr != nil {
				if result.Reason == "command_not_found" && runtime.GOOS == "windows" {
					if isWindowsShellBuiltin(args[0]) {
						fmt.Fprintf(
							cmd.ErrOrStderr(),
							"Hint: %q is a Windows shell builtin. Try `infratrack run -- cmd /c %s`.\n",
							args[0],
							sanitized.Command,
						)
					}
				}

				return &ExitError{
					Code: result.CLIExitCode,
					Err:  fmt.Errorf("command execution failed: %w", runErr),
				}
			}

			fmt.Fprintf(
				cmd.OutOrStdout(),
				"Recorded step (%d ms, exit %s)\n",
				step.DurationMS,
				formatExitCode(step.ExitCode),
			)
			return nil
		},
	}
}

func formatExitCode(code *int) string {
	if code == nil {
		return "n/a"
	}
	return fmt.Sprintf("%d", *code)
}

func isWindowsShellBuiltin(cmd string) bool {
	switch strings.ToLower(cmd) {
	case "echo", "dir", "copy", "type", "del", "erase", "move", "ren", "rename", "set":
		return true
	default:
		return false
	}
}

func newExportCmd(s store.SessionStore) *cobra.Command {
	var (
		exportLast bool
		exportMD   bool
	)

	cmd := &cobra.Command{
		Use:     "export",
		Aliases: []string{"x"},
		Short:   "Export a completed session as markdown",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !exportLast {
				return errors.New("MVP currently supports only `infratrack export --last --md`")
			}
			if !exportMD {
				return errors.New("only markdown export is supported in MVP. Use `--md`")
			}

			session, err := s.LastSession(cmd.Context())
			if err != nil {
				if errors.Is(err, store.ErrNoSessions) {
					return errors.New("no completed sessions found. Run start -> run -> stop first")
				}
				return fmt.Errorf("load last session: %w", err)
			}

			workingDir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("get current directory: %w", err)
			}

			outPath, err := export.WriteMarkdown(session, workingDir)
			if err != nil {
				return fmt.Errorf("export markdown: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Exported runbook: %s\n", outPath)
			return nil
		},
	}

	cmd.Flags().BoolVar(&exportLast, "last", false, "Export the most recent completed session")
	cmd.Flags().BoolVar(&exportMD, "md", false, "Export markdown output")
	return cmd
}
