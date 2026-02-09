package hooks

import (
	"context"
	"path/filepath"
	"testing"
)

func TestFileStateStoreDefaults(t *testing.T) {
	t.Parallel()

	root := newRetryTempDir(t)
	s := NewFileStateStore(root)
	state, err := s.Load(context.Background())
	if err != nil {
		t.Fatalf("load default state: %v", err)
	}
	if state.Enabled {
		t.Fatalf("expected disabled by default")
	}
	if state.RemindEvery != defaultRemindEvery {
		t.Fatalf("unexpected default remind every: %d", state.RemindEvery)
	}
}

func TestFileStateStoreSaveLoad(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := newRetryTempDir(t)
	s := NewFileStateStore(root)

	state := State{
		Enabled:      true,
		RemindEvery:  7,
		CommandCount: 14,
	}
	if err := s.Save(ctx, state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	got, err := s.Load(ctx)
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if !got.Enabled || got.RemindEvery != 7 || got.CommandCount != 14 {
		t.Fatalf("unexpected state: %+v", got)
	}

	if filepath.Base(s.path) != "hooks_state.json" {
		t.Fatalf("unexpected state file name: %s", s.path)
	}
}
