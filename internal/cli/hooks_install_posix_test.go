package cli

import (
	"strings"
	"testing"
)

func TestUpsertBashHookBlock(t *testing.T) {
	t.Parallel()

	original := "export PATH=$PATH:$HOME/bin\n"
	updated, changed, err := upsertHookBlock(original, bashHookBeginMarker, bashHookEndMarker, bashHookBlock())
	if err != nil {
		t.Fatalf("upsert bash block failed: %v", err)
	}
	if !changed {
		t.Fatal("expected upsert to change content")
	}
	if !strings.Contains(updated, bashHookBeginMarker) || !strings.Contains(updated, bashHookEndMarker) {
		t.Fatal("expected bash markers in content")
	}

	second, changed2, err := upsertHookBlock(updated, bashHookBeginMarker, bashHookEndMarker, bashHookBlock())
	if err != nil {
		t.Fatalf("second upsert failed: %v", err)
	}
	if changed2 {
		t.Fatal("expected idempotent upsert for bash block")
	}
	if second != updated {
		t.Fatal("expected unchanged content on second upsert")
	}
}

func TestRemoveBashHookBlock(t *testing.T) {
	t.Parallel()

	withBlock, _, err := upsertHookBlock("echo hi\n", bashHookBeginMarker, bashHookEndMarker, bashHookBlock())
	if err != nil {
		t.Fatalf("create block failed: %v", err)
	}

	updated, changed, err := replaceBetweenMarkers(withBlock, bashHookBeginMarker, bashHookEndMarker, "")
	if err != nil {
		t.Fatalf("remove block failed: %v", err)
	}
	if !changed {
		t.Fatal("expected change on remove")
	}
	if strings.Contains(updated, bashHookBeginMarker) || strings.Contains(updated, bashHookEndMarker) {
		t.Fatal("expected bash markers removed")
	}
	if !strings.Contains(updated, "echo hi") {
		t.Fatal("expected original non-hook content to remain")
	}
}

func TestUpsertZshHookBlock(t *testing.T) {
	t.Parallel()

	updated, changed, err := upsertHookBlock("", zshHookBeginMarker, zshHookEndMarker, zshHookBlock())
	if err != nil {
		t.Fatalf("upsert zsh block failed: %v", err)
	}
	if !changed {
		t.Fatal("expected upsert to change empty content")
	}
	if !strings.Contains(updated, zshHookBeginMarker) || !strings.Contains(updated, zshHookEndMarker) {
		t.Fatal("expected zsh markers in content")
	}
}
