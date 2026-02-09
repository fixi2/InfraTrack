package hooks

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fixi2/InfraTrack/internal/policy"
	"github.com/fixi2/InfraTrack/internal/store"
)

type RecordInput struct {
	Command    string
	CWD        string
	ExitCode   int
	DurationMS int64
	Timestamp  time.Time
}

type RecordResult struct {
	Recorded      bool
	SkippedReason string
	Reminder      bool
	Step          store.Step
}

type Recorder struct {
	store      store.SessionStore
	policy     *policy.Policy
	stateStore StateStore
}

func NewRecorder(sessionStore store.SessionStore, pol *policy.Policy, stateStore StateStore) *Recorder {
	return &Recorder{
		store:      sessionStore,
		policy:     pol,
		stateStore: stateStore,
	}
}

func (r *Recorder) Record(ctx context.Context, input RecordInput) (RecordResult, error) {
	raw := strings.TrimSpace(input.Command)
	if raw == "" {
		return RecordResult{}, errors.New("command cannot be empty")
	}

	if r.stateStore != nil {
		state, err := r.stateStore.Load(ctx)
		if err != nil {
			return RecordResult{}, fmt.Errorf("load hooks state: %w", err)
		}
		if !state.Enabled {
			return RecordResult{Recorded: false, SkippedReason: "hooks_disabled"}, nil
		}
	}

	args := splitCommand(raw)
	if isInfraTrackInvocation(args) {
		return RecordResult{Recorded: false, SkippedReason: "self_command"}, nil
	}

	if _, err := r.store.GetActiveSession(ctx); err != nil {
		if errors.Is(err, store.ErrNoActiveSession) || errors.Is(err, store.ErrNotInitialized) {
			return RecordResult{Recorded: false, SkippedReason: "no_active_session"}, nil
		}
		return RecordResult{}, fmt.Errorf("check active session: %w", err)
	}

	sanitized := r.policy.Apply(raw, args)
	step := store.Step{
		Timestamp:  normalizeTimestamp(input.Timestamp),
		Command:    sanitized.Command,
		DurationMS: clampDuration(input.DurationMS),
		CWD:        input.CWD,
	}
	if sanitized.Denied {
		step.Status = "REDACTED"
		step.Reason = "policy_redacted"
	} else if input.ExitCode == 0 {
		code := 0
		step.Status = "OK"
		step.ExitCode = &code
	} else {
		code := input.ExitCode
		step.Status = "FAILED"
		step.Reason = "nonzero_exit"
		step.ExitCode = &code
	}

	if err := r.store.AddStep(ctx, step); err != nil {
		return RecordResult{}, fmt.Errorf("record hook step: %w", err)
	}

	reminder, err := r.bumpCounter(ctx)
	if err != nil {
		return RecordResult{}, err
	}

	return RecordResult{
		Recorded: true,
		Reminder: reminder,
		Step:     step,
	}, nil
}

func (r *Recorder) bumpCounter(ctx context.Context) (bool, error) {
	if r.stateStore == nil {
		return false, nil
	}

	state, err := r.stateStore.Load(ctx)
	if err != nil {
		return false, fmt.Errorf("load hooks state: %w", err)
	}

	state.CommandCount++
	if err := r.stateStore.Save(ctx, state); err != nil {
		return false, fmt.Errorf("save hooks state: %w", err)
	}

	return state.Enabled && state.RemindEvery > 0 && state.CommandCount%int64(state.RemindEvery) == 0, nil
}

func splitCommand(raw string) []string {
	parts := strings.Fields(raw)
	if len(parts) == 0 {
		return nil
	}
	return parts
}

func isInfraTrackInvocation(args []string) bool {
	if len(args) == 0 {
		return false
	}
	binary := strings.ToLower(filepath.Base(strings.Trim(args[0], `"'`)))
	return binary == "infratrack" || binary == "infratrack.exe"
}

func normalizeTimestamp(ts time.Time) time.Time {
	if ts.IsZero() {
		return time.Now().UTC()
	}
	return ts.UTC()
}

func clampDuration(v int64) int64 {
	if v < 0 {
		return 0
	}
	return v
}

func ParseExitCode(v string) (int, error) {
	code, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return 0, fmt.Errorf("parse exit code: %w", err)
	}
	return code, nil
}
