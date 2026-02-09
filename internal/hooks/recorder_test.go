package hooks

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/fixi2/InfraTrack/internal/policy"
	"github.com/fixi2/InfraTrack/internal/store"
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

func TestRecorderSkipsInfraTrackCommand(t *testing.T) {
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
		Command:    "infratrack status",
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

	dir, err := os.MkdirTemp("", "infratrack-hooks-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})
	return dir
}
