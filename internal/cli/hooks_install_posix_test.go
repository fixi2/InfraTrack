package cli

import (
	"strings"
	"testing"
)

func TestUpsertBashHookBlock(t *testing.T) {
	t.Parallel()

	original := "export PATH=$PATH:$HOME/bin\n"
	updated, changed, err := upsertHookBlock(original, bashHookBeginMarker, bashHookEndMarker, bashHookBlock("/usr/local/bin/cmdry"))
	if err != nil {
		t.Fatalf("upsert bash block failed: %v", err)
	}
	if !changed {
		t.Fatal("expected upsert to change content")
	}
	if !strings.Contains(updated, bashHookBeginMarker) || !strings.Contains(updated, bashHookEndMarker) {
		t.Fatal("expected bash markers in content")
	}

	second, changed2, err := upsertHookBlock(updated, bashHookBeginMarker, bashHookEndMarker, bashHookBlock("/usr/local/bin/cmdry"))
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

	withBlock, _, err := upsertHookBlock("echo hi\n", bashHookBeginMarker, bashHookEndMarker, bashHookBlock("/usr/local/bin/cmdry"))
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

	updated, changed, err := upsertHookBlock("", zshHookBeginMarker, zshHookEndMarker, zshHookBlock("/usr/local/bin/cmdry"))
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

func TestHookBlocksUseAbsolutePath(t *testing.T) {
	t.Parallel()
	bashBlock := bashHookBlock("/usr/local/bin/cmdry")
	if !strings.Contains(bashBlock, "'/usr/local/bin/cmdry' hook record") {
		t.Fatalf("expected absolute path in bash block: %s", bashBlock)
	}
	zshBlock := zshHookBlock("/usr/local/bin/cmdry")
	if !strings.Contains(zshBlock, "'/usr/local/bin/cmdry' hook record") {
		t.Fatalf("expected absolute path in zsh block: %s", zshBlock)
	}
	if !strings.Contains(bashBlock, "$HOME/.config/commandry") {
		t.Fatalf("expected commandry state path in bash block: %s", bashBlock)
	}
	if !strings.Contains(zshBlock, "$HOME/.config/commandry") {
		t.Fatalf("expected commandry state path in zsh block: %s", zshBlock)
	}
	if !strings.Contains(bashBlock, "__commandry_should_prefix") {
		t.Fatalf("expected conditional REC helper in bash block: %s", bashBlock)
	}
	if !strings.Contains(zshBlock, "__commandry_should_prefix") {
		t.Fatalf("expected conditional REC helper in zsh block: %s", zshBlock)
	}
	if !strings.Contains(bashBlock, "trap '__commandry_preexec' DEBUG") {
		t.Fatalf("expected bash DEBUG trap preexec hook: %s", bashBlock)
	}
	if !strings.Contains(zshBlock, "add-zsh-hook preexec __commandry_preexec") {
		t.Fatalf("expected zsh preexec hook: %s", zshBlock)
	}
	if !strings.Contains(bashBlock, "__commandry_hook_ready=1") {
		t.Fatalf("expected bash block to enable ready flag after init: %s", bashBlock)
	}
	if !strings.Contains(zshBlock, "__commandry_hook_ready=1") {
		t.Fatalf("expected zsh block to enable ready flag after init: %s", zshBlock)
	}
	if strings.Contains(bashBlock, "__infratrack_") {
		t.Fatalf("unexpected legacy helper prefix in bash block: %s", bashBlock)
	}
	if strings.Contains(zshBlock, "__infratrack_") {
		t.Fatalf("unexpected legacy helper prefix in zsh block: %s", zshBlock)
	}
	if strings.Contains(strings.ToLower(bashBlock), "infratrack") {
		t.Fatalf("unexpected legacy brand token in bash block: %s", bashBlock)
	}
	if strings.Contains(strings.ToLower(zshBlock), "infratrack") {
		t.Fatalf("unexpected legacy brand token in zsh block: %s", zshBlock)
	}
	if !strings.Contains(bashBlock, "\"exit\"") {
		t.Fatalf("expected bash block to ignore exit command: %s", bashBlock)
	}
	if !strings.Contains(zshBlock, "\"exit\"") {
		t.Fatalf("expected zsh block to ignore exit command: %s", zshBlock)
	}
	if !strings.Contains(bashBlock, "PS1=\"${PS1#\\[REC\\] }\"") {
		t.Fatalf("expected bash block to remove REC prefix when inactive: %s", bashBlock)
	}
	if !strings.Contains(zshBlock, "PROMPT=\"${PROMPT#\\[REC\\] }\"") {
		t.Fatalf("expected zsh block to remove REC prefix when inactive: %s", zshBlock)
	}
}

func TestUpsertHookBlockMalformedMarkers(t *testing.T) {
	t.Parallel()
	content := bashHookEndMarker + "\n" + bashHookBeginMarker + "\n"
	_, _, err := upsertHookBlock(content, bashHookBeginMarker, bashHookEndMarker, bashHookBlock("/usr/local/bin/cmdry"))
	if err == nil {
		t.Fatal("expected malformed markers error")
	}
}

func TestUpsertHookBlockReplacesLegacyMarkers(t *testing.T) {
	t.Parallel()

	legacy := strings.Join([]string{
		legacyBashHookBeginMarker,
		"echo legacy",
		legacyBashHookEndMarker,
		"",
	}, "\n")
	updated, changed, err := upsertHookBlock(legacy, bashHookBeginMarker, bashHookEndMarker, bashHookBlock("/usr/local/bin/cmdry"))
	if err != nil {
		t.Fatalf("upsert from legacy failed: %v", err)
	}
	if !changed {
		t.Fatal("expected legacy replacement change")
	}
	if strings.Contains(updated, legacyBashHookBeginMarker) || strings.Contains(updated, legacyBashHookEndMarker) {
		t.Fatalf("legacy markers must be removed: %s", updated)
	}
	if !strings.Contains(updated, bashHookBeginMarker) || !strings.Contains(updated, bashHookEndMarker) {
		t.Fatalf("new markers missing: %s", updated)
	}
}
