package hooks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	defaultRemindEvery = 20
)

type State struct {
	Enabled      bool      `json:"enabled"`
	RemindEvery  int       `json:"remind_every"`
	CommandCount int64     `json:"command_count"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type StateStore interface {
	Load(ctx context.Context) (State, error)
	Save(ctx context.Context, state State) error
}

type FileStateStore struct {
	path string
}

func NewFileStateStore(rootPath string) *FileStateStore {
	return &FileStateStore{
		path: filepath.Join(rootPath, "hooks_state.json"),
	}
}

func (s *FileStateStore) Load(_ context.Context) (State, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return defaultState(), nil
		}
		return State{}, fmt.Errorf("read hooks state: %w", err)
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, fmt.Errorf("decode hooks state: %w", err)
	}
	normalizeState(&state)
	return state, nil
}

func (s *FileStateStore) Save(_ context.Context, state State) error {
	normalizeState(&state)
	state.UpdatedAt = time.Now().UTC()

	payload, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal hooks state: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("create hooks state directory: %w", err)
	}

	return os.WriteFile(s.path, payload, 0o600)
}

func defaultState() State {
	return State{
		Enabled:      false,
		RemindEvery:  defaultRemindEvery,
		CommandCount: 0,
	}
}

func normalizeState(state *State) {
	if state.RemindEvery < 0 {
		state.RemindEvery = defaultRemindEvery
	}
	if state.CommandCount < 0 {
		state.CommandCount = 0
	}
}
