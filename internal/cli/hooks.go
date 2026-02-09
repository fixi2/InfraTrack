package cli

import (
	"errors"
	"fmt"
	"time"

	"github.com/fixi2/InfraTrack/internal/hooks"
	"github.com/fixi2/InfraTrack/internal/policy"
	"github.com/fixi2/InfraTrack/internal/store"
	"github.com/spf13/cobra"
)

func newHooksCmd(s store.SessionStore, stateStore hooks.StateStore) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hooks",
		Short: "Manage hooks recording mode state",
	}

	cmd.AddCommand(
		newHooksStatusCmd(s, stateStore),
		newHooksEnableCmd(stateStore),
		newHooksDisableCmd(stateStore),
		newHooksConfigureCmd(stateStore),
	)
	return cmd
}

func newHooksStatusCmd(s store.SessionStore, stateStore hooks.StateStore) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show hooks mode state",
		RunE: func(cmd *cobra.Command, _ []string) error {
			state, err := stateStore.Load(cmd.Context())
			if err != nil {
				return fmt.Errorf("load hooks state: %w", err)
			}
			_, activeErr := s.GetActiveSession(cmd.Context())
			recording := activeErr == nil

			fmt.Fprintf(cmd.OutOrStdout(), "Hooks: %s\n", boolLabel(state.Enabled))
			fmt.Fprintf(cmd.OutOrStdout(), "Remind every: %d\n", state.RemindEvery)
			fmt.Fprintf(cmd.OutOrStdout(), "Recorded commands: %d\n", state.CommandCount)
			fmt.Fprintf(cmd.OutOrStdout(), "Session recording: %s\n", boolLabel(recording))
			return nil
		},
	}
}

func newHooksEnableCmd(stateStore hooks.StateStore) *cobra.Command {
	return &cobra.Command{
		Use:   "enable",
		Short: "Enable hooks mode",
		RunE: func(cmd *cobra.Command, _ []string) error {
			state, err := stateStore.Load(cmd.Context())
			if err != nil {
				return fmt.Errorf("load hooks state: %w", err)
			}
			state.Enabled = true
			if err := stateStore.Save(cmd.Context(), state); err != nil {
				return fmt.Errorf("save hooks state: %w", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Hooks mode enabled")
			return nil
		},
	}
}

func newHooksDisableCmd(stateStore hooks.StateStore) *cobra.Command {
	return &cobra.Command{
		Use:   "disable",
		Short: "Disable hooks mode",
		RunE: func(cmd *cobra.Command, _ []string) error {
			state, err := stateStore.Load(cmd.Context())
			if err != nil {
				return fmt.Errorf("load hooks state: %w", err)
			}
			state.Enabled = false
			if err := stateStore.Save(cmd.Context(), state); err != nil {
				return fmt.Errorf("save hooks state: %w", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Hooks mode disabled")
			return nil
		},
	}
}

func newHooksConfigureCmd(stateStore hooks.StateStore) *cobra.Command {
	var remindEvery int

	cmd := &cobra.Command{
		Use:     "configure",
		Aliases: []string{"config"},
		Short:   "Configure hooks mode settings",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if remindEvery <= 0 {
				return errors.New("remind-every must be greater than 0")
			}

			state, err := stateStore.Load(cmd.Context())
			if err != nil {
				return fmt.Errorf("load hooks state: %w", err)
			}
			state.RemindEvery = remindEvery
			if err := stateStore.Save(cmd.Context(), state); err != nil {
				return fmt.Errorf("save hooks state: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Updated hooks remind-every to %d\n", remindEvery)
			return nil
		},
	}

	cmd.Flags().IntVar(&remindEvery, "remind-every", 20, "Print [REC] reminder every N recorded commands")
	return cmd
}

func newHookCmd(s store.SessionStore, p *policy.Policy, stateStore hooks.StateStore) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "hook",
		Short:  "Internal hooks endpoint",
		Hidden: true,
	}
	cmd.AddCommand(newHookRecordCmd(s, p, stateStore))
	return cmd
}

func newHookRecordCmd(s store.SessionStore, p *policy.Policy, stateStore hooks.StateStore) *cobra.Command {
	var (
		rawCommand string
		cwd        string
		exitCode   int
		durationMS int64
		timestamp  string
	)

	cmd := &cobra.Command{
		Use:    "record",
		Short:  "Record one shell command event",
		Hidden: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ts := time.Time{}
			if timestamp != "" {
				parsed, err := time.Parse(time.RFC3339, timestamp)
				if err != nil {
					return fmt.Errorf("parse timestamp: %w", err)
				}
				ts = parsed
			}

			rec := hooks.NewRecorder(s, p, stateStore)
			result, err := rec.Record(cmd.Context(), hooks.RecordInput{
				Command:    rawCommand,
				CWD:        cwd,
				ExitCode:   exitCode,
				DurationMS: durationMS,
				Timestamp:  ts,
			})
			if err != nil {
				return err
			}

			if result.Reminder {
				fmt.Fprintln(cmd.ErrOrStderr(), "[REC] InfraTrack recording is active.")
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&rawCommand, "command", "", "Raw command line to record")
	cmd.Flags().StringVar(&cwd, "cwd", "", "Working directory of the command")
	cmd.Flags().IntVar(&exitCode, "exit-code", 0, "Command exit code")
	cmd.Flags().Int64Var(&durationMS, "duration-ms", 0, "Command duration in milliseconds")
	cmd.Flags().StringVar(&timestamp, "timestamp", "", "Command timestamp in RFC3339 format")
	_ = cmd.MarkFlagRequired("command")
	return cmd
}

func boolLabel(v bool) string {
	if v {
		return "enabled"
	}
	return "disabled"
}
