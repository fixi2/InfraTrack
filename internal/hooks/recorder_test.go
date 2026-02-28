package hooks

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/fixi2/Commandry/internal/policy"
	"github.com/fixi2/Commandry/internal/store"
)

func TestRecorderRecord(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := newRetryTempDir(t)
	sessionStore := store.NewJSONStore(root)
	if err := sessionStore.Init(ctx); err != nil {
		t.Fatalf("init store: %v", err)
	}
	if _, err := sessionStore.StartSession(ctx, "hooks", "", time.Now().UTC()); err != nil {
		t.Fatalf("start session: %v", err)
	}

	stateStore := NewFileStateStore(root)
	state := defaultState()
	state.Enabled = true
	state.RemindEvery = 2
	if err := stateStore.Save(ctx, state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	rec := NewRecorder(sessionStore, policy.NewDefault(), stateStore)
	first, err := rec.Record(ctx, RecordInput{
		Command:    "echo hello",
		CWD:        root,
		ExitCode:   0,
		DurationMS: 12,
	})
	if err != nil {
		t.Fatalf("record first: %v", err)
	}
	if !first.Recorded || first.Reminder {
		t.Fatalf("unexpected first result: %+v", first)
	}

	second, err := rec.Record(ctx, RecordInput{
		Command:    "echo world",
		CWD:        root,
		ExitCode:   0,
		DurationMS: 8,
	})
	if err != nil {
		t.Fatalf("record second: %v", err)
	}
	if !second.Recorded || !second.Reminder {
		t.Fatalf("unexpected second result: %+v", second)
	}
}

func TestRecorderNoReminderWhenDisabled(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := newRetryTempDir(t)
	sessionStore := store.NewJSONStore(root)
	if err := sessionStore.Init(ctx); err != nil {
		t.Fatalf("init store: %v", err)
	}
	if _, err := sessionStore.StartSession(ctx, "hooks", "", time.Now().UTC()); err != nil {
		t.Fatalf("start session: %v", err)
	}

	stateStore := NewFileStateStore(root)
	state := defaultState()
	state.Enabled = true
	state.RemindEvery = 0
	if err := stateStore.Save(ctx, state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	rec := NewRecorder(sessionStore, policy.NewDefault(), stateStore)
	for i := 0; i < 5; i++ {
		res, err := rec.Record(ctx, RecordInput{
			Command:    "echo hello",
			CWD:        root,
			ExitCode:   0,
			DurationMS: 1,
		})
		if err != nil {
			t.Fatalf("record %d: %v", i, err)
		}
		if res.Reminder {
			t.Fatalf("expected reminders to stay disabled")
		}
	}
}

func TestRecorderSkipsSelfCommands(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := newRetryTempDir(t)
	sessionStore := store.NewJSONStore(root)
	if err := sessionStore.Init(ctx); err != nil {
		t.Fatalf("init store: %v", err)
	}
	if _, err := sessionStore.StartSession(ctx, "hooks", "", time.Now().UTC()); err != nil {
		t.Fatalf("start session: %v", err)
	}

	stateStore := NewFileStateStore(root)
	state := defaultState()
	state.Enabled = true
	if err := stateStore.Save(ctx, state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	rec := NewRecorder(sessionStore, policy.NewDefault(), stateStore)
	result, err := rec.Record(ctx, RecordInput{
		Command:    "cmdry status",
		ExitCode:   0,
		DurationMS: 5,
	})
	if err != nil {
		t.Fatalf("record self command: %v", err)
	}
	if result.Recorded {
		t.Fatalf("expected self command to be skipped")
	}
	if result.SkippedReason != "self_command" {
		t.Fatalf("unexpected skip reason: %s", result.SkippedReason)
	}

	resultAlias, err := rec.Record(ctx, RecordInput{
		Command:    "it status",
		ExitCode:   0,
		DurationMS: 5,
	})
	if err != nil {
		t.Fatalf("record alias self command: %v", err)
	}
	if resultAlias.Recorded {
		t.Fatalf("expected alias self command to be skipped")
	}
	if resultAlias.SkippedReason != "self_command" {
		t.Fatalf("unexpected alias skip reason: %s", resultAlias.SkippedReason)
	}

	resultCmdry, err := rec.Record(ctx, RecordInput{
		Command:    "cmdry status",
		ExitCode:   0,
		DurationMS: 5,
	})
	if err != nil {
		t.Fatalf("record cmdry self command: %v", err)
	}
	if resultCmdry.Recorded {
		t.Fatalf("expected cmdry self command to be skipped")
	}
	if resultCmdry.SkippedReason != "self_command" {
		t.Fatalf("unexpected cmdry skip reason: %s", resultCmdry.SkippedReason)
	}

	resultCmdr, err := rec.Record(ctx, RecordInput{
		Command:    "cmdr status",
		ExitCode:   0,
		DurationMS: 5,
	})
	if err != nil {
		t.Fatalf("record cmdr self command: %v", err)
	}
	if resultCmdr.Recorded {
		t.Fatalf("expected cmdr self command to be skipped")
	}
	if resultCmdr.SkippedReason != "self_command" {
		t.Fatalf("unexpected cmdr skip reason: %s", resultCmdr.SkippedReason)
	}
}

func TestRecorderAppliesPolicy(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := newRetryTempDir(t)
	sessionStore := store.NewJSONStore(root)
	if err := sessionStore.Init(ctx); err != nil {
		t.Fatalf("init store: %v", err)
	}
	if _, err := sessionStore.StartSession(ctx, "hooks", "", time.Now().UTC()); err != nil {
		t.Fatalf("start session: %v", err)
	}

	stateStore := NewFileStateStore(root)
	state := defaultState()
	state.Enabled = true
	if err := stateStore.Save(ctx, state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	rec := NewRecorder(sessionStore, policy.NewDefault(), stateStore)
	if _, err := rec.Record(ctx, RecordInput{
		Command:    `curl -H "Authorization: Bearer abc123" https://example.com`,
		ExitCode:   1,
		DurationMS: 10,
	}); err != nil {
		t.Fatalf("record command: %v", err)
	}

	last, err := sessionStore.GetActiveSession(ctx)
	if err != nil {
		t.Fatalf("active session: %v", err)
	}
	if got := last.Steps[len(last.Steps)-1].Command; got == `curl -H "Authorization: Bearer abc123" https://example.com` {
		t.Fatalf("expected command to be redacted, got %q", got)
	}
}

func TestRecorderSkipsWhenHooksDisabled(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := newRetryTempDir(t)
	sessionStore := store.NewJSONStore(root)
	if err := sessionStore.Init(ctx); err != nil {
		t.Fatalf("init store: %v", err)
	}
	if _, err := sessionStore.StartSession(ctx, "hooks", "", time.Now().UTC()); err != nil {
		t.Fatalf("start session: %v", err)
	}

	rec := NewRecorder(sessionStore, policy.NewDefault(), NewFileStateStore(root))
	result, err := rec.Record(ctx, RecordInput{
		Command:    "echo hi",
		ExitCode:   0,
		DurationMS: 3,
	})
	if err != nil {
		t.Fatalf("record command: %v", err)
	}
	if result.Recorded {
		t.Fatalf("expected skip when hooks are disabled")
	}
	if result.SkippedReason != "hooks_disabled" {
		t.Fatalf("unexpected skip reason: %s", result.SkippedReason)
	}
}

func newRetryTempDir(t *testing.T) string {
	t.Helper()

	dir, err := os.MkdirTemp("", "cmdry-hooks-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})
	return dir
}
