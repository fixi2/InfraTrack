package capture

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestRunCommand_ClassifiesNotFound(t *testing.T) {
	t.Parallel()

	res, err := RunCommand(context.Background(), []string{"definitely-not-a-command-12345"}, t.TempDir())
	if err == nil {
		t.Fatalf("expected error")
	}
	if res.Status != "FAILED" {
		t.Fatalf("status mismatch: %s", res.Status)
	}
	if res.Reason != "command_not_found" {
		t.Fatalf("reason mismatch: %s", res.Reason)
	}
	if res.ExitCode != nil {
		t.Fatalf("expected exit code to be nil, got %v", *res.ExitCode)
	}
	if res.CLIExitCode == 0 {
		t.Fatalf("expected nonzero CLI exit code")
	}
}

func TestRunCommand_ClassifiesExitCodes(t *testing.T) {
	t.Parallel()

	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}

	t.Run("ok", func(t *testing.T) {
		t.Parallel()

		cmd := []string{exe, "-test.run=TestHelperProcess", "--", "--commandry-helper-process=1", "exit", "0"}
		res, err := RunCommand(context.Background(), cmd, t.TempDir())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Status != "OK" {
			t.Fatalf("status mismatch: %s", res.Status)
		}
		if res.ExitCode == nil || *res.ExitCode != 0 {
			t.Fatalf("exit code mismatch: %v", res.ExitCode)
		}
	})

	t.Run("nonzero", func(t *testing.T) {
		t.Parallel()

		cmd := []string{exe, "-test.run=TestHelperProcess", "--", "--commandry-helper-process=1", "exit", "7"}
		res, err := RunCommand(context.Background(), cmd, t.TempDir())
		if err == nil {
			t.Fatalf("expected error")
		}
		if res.Status != "FAILED" {
			t.Fatalf("status mismatch: %s", res.Status)
		}
		if res.Reason != "nonzero_exit" {
			t.Fatalf("reason mismatch: %s", res.Reason)
		}
		if res.ExitCode == nil || *res.ExitCode != 7 {
			t.Fatalf("exit code mismatch: %v", res.ExitCode)
		}
		if res.CLIExitCode != 7 {
			t.Fatalf("cli exit code mismatch: %d", res.CLIExitCode)
		}
	})
}

func TestHelperProcess(t *testing.T) {
	// Arguments after "--" are controlled by our tests.
	args := os.Args
	sep := -1
	for i := range args {
		if args[i] == "--" {
			sep = i
			break
		}
	}
	if sep < 0 || sep+1 >= len(args) {
		return
	}

	if args[sep+1] != "--commandry-helper-process=1" {
		return
	}
	if sep+2 >= len(args) {
		os.Exit(2)
	}

	switch args[sep+2] {
	case "exit":
		if sep+3 >= len(args) {
			os.Exit(2)
		}
		code := args[sep+3]

		// Avoid strconv (keep it tiny) and accept only single-digit codes we use in tests.
		switch code {
		case "0":
			time.Sleep(10 * time.Millisecond)
			os.Exit(0)
		case "7":
			time.Sleep(10 * time.Millisecond)
			os.Exit(7)
		default:
			os.Exit(2)
		}
	default:
		os.Exit(2)
	}
}
