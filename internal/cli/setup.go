package cli

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/fixi2/InfraTrack/internal/setup"
	"github.com/spf13/cobra"
)

func newSetupCmd() *cobra.Command {
	cfg := &setupCommandConfig{}
	var yes bool

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Install InfraTrack for the current user",
		RunE: func(cmd *cobra.Command, _ []string) error {
			scope, completion, err := parseSetupInputs(cfg.scopeText, cfg.completionRaw)
			if err != nil {
				return err
			}
			plan, err := setup.BuildPlan(setup.PlanInput{
				Scope:      scope,
				BinDir:     cfg.binDir,
				NoPath:     cfg.noPath,
				Completion: completion,
			})
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Target binary: %s\n", plan.TargetBinaryPath)
			if cfg.noPath {
				fmt.Fprintln(out, "PATH: unchanged (--no-path)")
			} else {
				fmt.Fprintf(out, "PATH: ensure %s is in user PATH\n", plan.TargetBinDir)
			}
			fmt.Fprintln(out, "Run `infratrack setup plan` for full dry-run details.")

			if !yes {
				ok, err := confirmSetupApply(cmd)
				if err != nil {
					return err
				}
				if !ok {
					fmt.Fprintln(out, "Cancelled.")
					return nil
				}
			}

			result, err := setup.Apply(setup.ApplyInput{
				Scope:      scope,
				BinDir:     cfg.binDir,
				NoPath:     cfg.noPath,
				Completion: completion,
			})
			if err != nil {
				return err
			}
			printSetupApplyResult(out, result, cfg.noPath, false, false)
			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&cfg.binDir, "bin-dir", "", "Install target directory for infratrack binary")
	cmd.PersistentFlags().StringVar(&cfg.scopeText, "scope", string(setup.ScopeUser), "Setup scope: user")
	cmd.PersistentFlags().BoolVar(&cfg.noPath, "no-path", false, "Do not modify PATH")
	cmd.PersistentFlags().StringVar(&cfg.completionRaw, "completion", string(setup.CompletionNone), "Completion setup mode: none")
	cmd.Flags().BoolVar(&yes, "yes", false, "Apply without interactive confirmation")

	cmd.AddCommand(newSetupPlanCmd(cfg))
	cmd.AddCommand(newSetupStatusCmd(cfg))
	cmd.AddCommand(newSetupApplyCmd(cfg))
	cmd.AddCommand(newSetupUndoCmd())
	return cmd
}

type setupCommandConfig struct {
	binDir        string
	scopeText     string
	completionRaw string
	noPath        bool
}

func newSetupStatusCmd(cfg *setupCommandConfig) *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show setup status",
		RunE: func(cmd *cobra.Command, _ []string) error {
			scope, err := setup.ResolveScope(strings.TrimSpace(cfg.scopeText))
			if err != nil {
				return err
			}
			status, err := setup.BuildStatus(scope, strings.TrimSpace(cfg.binDir))
			if err != nil {
				return err
			}

			if jsonOut {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(status)
			}

			out := cmd.OutOrStdout()
			fmt.Fprintln(out, "InfraTrack setup status")
			fmt.Fprintln(out, "----------------------")
			fmt.Fprintf(out, "OS                 : %s\n", status.OS)
			fmt.Fprintf(out, "Scope              : %s\n", status.Scope)
			fmt.Fprintf(out, "Current executable : %s\n", status.CurrentExe)
			fmt.Fprintf(out, "Target bin dir     : %s\n", status.BinDir)
			fmt.Fprintf(out, "Target binary      : %s\n", status.TargetBinaryPath)
			fmt.Fprintln(out, "")
			fmt.Fprintf(out, "Installed          : %s\n", statusWord(status.Installed))
			fmt.Fprintf(out, "PATH configured    : %s\n", statusWord(status.PathOK))
			fmt.Fprintf(out, "State file found   : %s\n", statusWord(status.StateFound))
			fmt.Fprintf(out, "Pending finalize   : %s\n", statusWord(status.PendingFinalize))
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOut, "json", false, "Print machine-readable JSON status")
	return cmd
}

func newSetupPlanCmd(cfg *setupCommandConfig) *cobra.Command {
	return &cobra.Command{
		Use:   "plan",
		Short: "Show detailed setup dry-run plan",
		RunE: func(cmd *cobra.Command, _ []string) error {
			scope, completion, err := parseSetupInputs(cfg.scopeText, cfg.completionRaw)
			if err != nil {
				return err
			}
			plan, err := setup.BuildPlan(setup.PlanInput{
				Scope:      scope,
				BinDir:     cfg.binDir,
				NoPath:     cfg.noPath,
				Completion: completion,
			})
			if err != nil {
				return err
			}
			printSetupPlan(cmd, plan)
			return nil
		},
	}
}

func newSetupApplyCmd(cfg *setupCommandConfig) *cobra.Command {
	var yes bool
	var verbose bool

	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply setup changes",
		RunE: func(cmd *cobra.Command, _ []string) error {
			scope, completion, err := parseSetupInputs(cfg.scopeText, cfg.completionRaw)
			if err != nil {
				return err
			}

			if !yes {
				fmt.Fprintln(cmd.OutOrStdout(), "Run `infratrack setup plan` to review the full dry-run plan.")
				ok, err := confirmSetupApply(cmd)
				if err != nil {
					return err
				}
				if !ok {
					fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
					return nil
				}
			}

			result, err := setup.Apply(setup.ApplyInput{
				Scope:      scope,
				BinDir:     cfg.binDir,
				NoPath:     cfg.noPath,
				Completion: completion,
			})
			if err != nil {
				return err
			}
			printSetupApplyResult(cmd.OutOrStdout(), result, cfg.noPath, verbose, true)
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Apply without interactive confirmation")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "Print detailed apply steps")
	return cmd
}

func newSetupUndoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "undo",
		Short: "Undo setup changes",
		RunE: func(_ *cobra.Command, _ []string) error {
			return errors.New("setup undo is not implemented in this build yet")
		},
	}
}

func parseSetupInputs(scopeText, completionText string) (setup.Scope, setup.CompletionMode, error) {
	scope, err := setup.ResolveScope(scopeText)
	if err != nil {
		return "", "", err
	}
	if scope != setup.ScopeUser {
		return "", "", errors.New("only --scope user is available in v0.5.0 setup")
	}
	completion, err := setup.ResolveCompletion(completionText)
	if err != nil {
		return "", "", err
	}
	return scope, completion, nil
}

func confirmSetupApply(cmd *cobra.Command) (bool, error) {
	fmt.Fprint(cmd.OutOrStdout(), "Apply setup changes? [y/N]: ")
	reader := bufio.NewReader(cmd.InOrStdin())
	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, fmt.Errorf("read confirmation: %w", err)
	}
	answer := strings.TrimSpace(strings.ToLower(line))
	return answer == "y" || answer == "yes", nil
}

func printSetupPlan(cmd *cobra.Command, plan setup.Plan) {
	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "InfraTrack setup plan [DRY-RUN]")
	fmt.Fprintln(out, "--------------------------------")
	fmt.Fprintf(out, "Detected OS         : %s\n", plan.OS)
	fmt.Fprintf(out, "Scope               : %s\n", plan.Scope)
	fmt.Fprintf(out, "Current executable  : %s\n", plan.CurrentExe)
	fmt.Fprintf(out, "Target bin dir      : %s\n", plan.TargetBinDir)
	fmt.Fprintf(out, "Target binary       : %s\n", plan.TargetBinaryPath)
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "Planned actions:")
	for i, action := range plan.Actions {
		fmt.Fprintf(out, "  %d) %s\n", i+1, action)
	}
	if len(plan.Notes) > 0 {
		fmt.Fprintln(out, "")
		fmt.Fprintln(out, "Notes:")
		for _, note := range plan.Notes {
			fmt.Fprintf(out, "  - %s\n", note)
		}
	}
}

func printSetupApplyResult(out io.Writer, result setup.ApplyResult, noPath bool, verbose bool, showSummary bool) {
	if verbose {
		fmt.Fprintln(out, "InfraTrack setup apply")
		fmt.Fprintln(out, "---------------------")
		for i, action := range result.Actions {
			fmt.Fprintf(out, "%d) %s\n", i+1, action)
		}
		if len(result.Notes) > 0 {
			fmt.Fprintln(out, "")
			fmt.Fprintln(out, "Notes:")
			for _, note := range result.Notes {
				fmt.Fprintf(out, "- %s\n", note)
			}
		}
		return
	}

	if result.PendingFinalize {
		fmt.Fprintln(out, "Setup staged. Restart terminal, then run `infratrack setup apply` again.")
	} else {
		fmt.Fprintln(out, "Setup complete.")
	}
	if showSummary {
		fmt.Fprintf(out, "Binary: %s\n", result.InstalledBinPath)
		if noPath {
			fmt.Fprintln(out, "PATH: unchanged (--no-path)")
		} else {
			fmt.Fprintln(out, "PATH: pending (will be added in the next setup phase)")
		}
	}
	fmt.Fprintln(out, "Use `infratrack setup status` for details.")
}

func statusWord(v bool) string {
	if v {
		return "OK"
	}
	return "NO"
}
