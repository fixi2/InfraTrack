package store

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestJSONStoreLifecycle(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
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

func intPtr(v int) *int {
	return &v
}
