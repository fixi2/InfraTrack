package store

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestJSONStoreLifecycle(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := newRetryTempDir(t)
	s := NewJSONStore(root)

	_, err := s.StartSession(ctx, "deploy", "", time.Now().UTC())
	if !errors.Is(err, ErrNotInitialized) {
		t.Fatalf("expected ErrNotInitialized, got %v", err)
	}

	if err := s.Init(ctx); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	initialized, err := s.IsInitialized(ctx)
	if err != nil {
		t.Fatalf("is initialized failed: %v", err)
	}
	if !initialized {
		t.Fatal("expected initialized=true")
	}

	startedAt := time.Date(2026, 2, 3, 12, 0, 0, 0, time.UTC)
	session, err := s.StartSession(ctx, "Deploy to staging", "staging", startedAt)
	if err != nil {
		t.Fatalf("start session failed: %v", err)
	}
	if session.Title != "Deploy to staging" {
		t.Fatalf("unexpected title: %s", session.Title)
	}
	if session.Env != "staging" {
		t.Fatalf("unexpected env: %s", session.Env)
	}

	step := Step{
		Timestamp:  startedAt.Add(2 * time.Second),
		Command:    "kubectl apply -f deploy.yaml",
		Status:     "OK",
		Reason:     "",
		ExitCode:   intPtr(0),
		DurationMS: 1800,
		CWD:        "/repo",
	}
	if err := s.AddStep(ctx, step); err != nil {
		t.Fatalf("add step failed: %v", err)
	}

	active, err := s.GetActiveSession(ctx)
	if err != nil {
		t.Fatalf("get active session failed: %v", err)
	}
	if len(active.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(active.Steps))
	}

	endedAt := startedAt.Add(5 * time.Minute)
	stopped, err := s.StopSession(ctx, endedAt)
	if err != nil {
		t.Fatalf("stop session failed: %v", err)
	}
	if stopped.EndedAt == nil {
		t.Fatalf("expected ended_at to be set")
	}

	if _, err := s.GetActiveSession(ctx); !errors.Is(err, ErrNoActiveSession) {
		t.Fatalf("expected ErrNoActiveSession, got %v", err)
	}

	last, err := s.LastSession(ctx)
	if err != nil {
		t.Fatalf("last session failed: %v", err)
	}
	if last.Title != "Deploy to staging" {
		t.Fatalf("unexpected last session title: %s", last.Title)
	}
	if last.Env != "staging" {
		t.Fatalf("unexpected last session env: %s", last.Env)
	}
	if len(last.Steps) != 1 {
		t.Fatalf("expected 1 step in last session, got %d", len(last.Steps))
	}
	if last.Steps[0].Command != step.Command {
		t.Fatalf("unexpected command: %s", last.Steps[0].Command)
	}
	if last.Steps[0].Status != "OK" {
		t.Fatalf("unexpected status: %s", last.Steps[0].Status)
	}
	if last.Steps[0].ExitCode == nil || *last.Steps[0].ExitCode != 0 {
		t.Fatalf("unexpected exit code: %v", last.Steps[0].ExitCode)
	}
}

func TestJSONStoreListSessionsAndByID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := newRetryTempDir(t)
	s := NewJSONStore(root)
	if err := s.Init(ctx); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	base := time.Date(2026, 2, 6, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 3; i++ {
		start := base.Add(time.Duration(i) * time.Minute)
		title := "Session " + string(rune('A'+i))
		session, err := s.StartSession(ctx, title, "", start)
		if err != nil {
			t.Fatalf("start session %d failed: %v", i, err)
		}
		if err := s.AddStep(ctx, Step{
			Timestamp:  start.Add(5 * time.Second),
			Command:    "echo hello",
			Status:     "OK",
			ExitCode:   intPtr(0),
			DurationMS: 10,
		}); err != nil {
			t.Fatalf("add step %d failed: %v", i, err)
		}
		if _, err := s.StopSession(ctx, start.Add(10*time.Second)); err != nil {
			t.Fatalf("stop session %d failed: %v", i, err)
		}

		gotByID, err := s.SessionByID(ctx, session.ID)
		if err != nil {
			t.Fatalf("session by id %d failed: %v", i, err)
		}
		if gotByID.ID != session.ID {
			t.Fatalf("session id mismatch: got %s want %s", gotByID.ID, session.ID)
		}
	}

	recent, err := s.ListSessions(ctx, 2)
	if err != nil {
		t.Fatalf("list sessions failed: %v", err)
	}
	if len(recent) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(recent))
	}
	if !recent[0].StartedAt.After(recent[1].StartedAt) {
		t.Fatalf("sessions are not sorted by recency")
	}

	_, err = s.SessionByID(ctx, "missing")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestJSONStoreLastSessionLargeRecord(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := newRetryTempDir(t)
	s := NewJSONStore(root)
	if err := s.Init(ctx); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	start := time.Date(2026, 2, 7, 12, 0, 0, 0, time.UTC)
	if _, err := s.StartSession(ctx, "Large session", "", start); err != nil {
		t.Fatalf("start session failed: %v", err)
	}

	largeCmd := "cmd /c echo " + strings.Repeat("x", 16*1024)
	for i := 0; i < 8; i++ {
		if err := s.AddStep(ctx, Step{
			Timestamp:  start.Add(time.Duration(i+1) * time.Second),
			Command:    largeCmd,
			Status:     "OK",
			ExitCode:   intPtr(0),
			DurationMS: 10,
		}); err != nil {
			t.Fatalf("add step %d failed: %v", i, err)
		}
	}

	if _, err := s.StopSession(ctx, start.Add(20*time.Second)); err != nil {
		t.Fatalf("stop session failed: %v", err)
	}

	last, err := s.LastSession(ctx)
	if err != nil {
		t.Fatalf("last session failed: %v", err)
	}
	if len(last.Steps) != 8 {
		t.Fatalf("expected 8 steps, got %d", len(last.Steps))
	}
	if last.Steps[0].Command != largeCmd {
		t.Fatalf("unexpected command payload length: got %d want %d", len(last.Steps[0].Command), len(largeCmd))
	}
}

func TestJSONStoreAddStepConcurrentNoLostUpdates(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := newRetryTempDir(t)
	s := NewJSONStore(root)
	if err := s.Init(ctx); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	start := time.Date(2026, 2, 20, 12, 0, 0, 0, time.UTC)
	if _, err := s.StartSession(ctx, "Concurrent steps", "", start); err != nil {
		t.Fatalf("start session failed: %v", err)
	}

	const total = 100
	var wg sync.WaitGroup
	wg.Add(total)
	for i := 0; i < total; i++ {
		i := i
		go func() {
			defer wg.Done()
			err := s.AddStep(ctx, Step{
				Timestamp:  start.Add(time.Duration(i+1) * time.Second),
				Command:    "echo step-" + strconv.Itoa(i),
				Status:     "OK",
				ExitCode:   intPtr(0),
				DurationMS: int64(i + 1),
			})
			if err != nil {
				t.Errorf("add step %d failed: %v", i, err)
			}
		}()
	}
	wg.Wait()

	active, err := s.GetActiveSession(ctx)
	if err != nil {
		t.Fatalf("get active session failed: %v", err)
	}
	if got := len(active.Steps); got != total {
		t.Fatalf("unexpected step count: got %d want %d", got, total)
	}
}

func intPtr(v int) *int {
	return &v
}

func newRetryTempDir(t *testing.T) string {
	t.Helper()

	dir, err := os.MkdirTemp("", "commandry-store-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}

	t.Cleanup(func() {
		var lastErr error
		for i := 0; i < 20; i++ {
			if err := os.RemoveAll(dir); err == nil {
				return
			} else {
				lastErr = err
				time.Sleep(15 * time.Millisecond)
			}
		}
		t.Fatalf("cleanup temp dir failed: %v", lastErr)
	})

	return dir
}
