package capture

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"
)

type RunResult struct {
	StartedAt   time.Time
	Duration    time.Duration
	ExitCode    *int
	Status      string
	Reason      string
	CLIExitCode int
}

// RunCommand executes the provided command without capturing stdout or stderr.
func RunCommand(ctx context.Context, args []string, cwd string) (RunResult, error) {
	startedAt := time.Now().UTC()
	result := RunResult{
		StartedAt: startedAt,
	}

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = cwd
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	result.Duration = time.Since(startedAt)

	if err == nil {
		code := 0
		result.ExitCode = &code
		result.Status = "OK"
		result.Reason = ""
		result.CLIExitCode = 0
		return result, nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		code := exitErr.ExitCode()
		result.ExitCode = &code
		result.Status = "FAILED"
		result.Reason = "nonzero_exit"
		result.CLIExitCode = code
		return result, err
	}

	reason := classifyStartError(err)
	result.Status = "FAILED"
	result.Reason = reason
	result.ExitCode = nil
	result.CLIExitCode = cliExitCodeForStartFailure(reason)
	return result, err
}

func classifyStartError(err error) string {
	var execErr *exec.Error
	if errors.As(err, &execErr) && errors.Is(execErr.Err, exec.ErrNotFound) {
		return "command_not_found"
	}

	// On some platforms the underlying error may be a PathError.
	var pathErr *os.PathError
	if errors.As(err, &pathErr) && errors.Is(pathErr.Err, exec.ErrNotFound) {
		return "command_not_found"
	}

	return "start_failed"
}

func cliExitCodeForStartFailure(reason string) int {
	// Keep POSIX convention where practical, even on Windows, since we also
	// persist a structured reason and avoid writing a misleading exit code to the store.
	if reason == "command_not_found" {
		if runtime.GOOS == "windows" {
			// cmd.exe uses 9009 for "not recognized", but our wrapper is not cmd.exe.
			// Keep 127 to align with common tooling expectations.
			return 127
		}
		return 127
	}
	return 1
}

func (r RunResult) String() string {
	if r.ExitCode == nil {
		return fmt.Sprintf("%s (%s)", r.Status, r.Reason)
	}
	return fmt.Sprintf("%s (%s), exit=%d", r.Status, r.Reason, *r.ExitCode)
}
