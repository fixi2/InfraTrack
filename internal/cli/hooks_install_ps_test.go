package cli

import (
	"strings"
	"testing"
)

func TestUpsertPowerShellHookBlock(t *testing.T) {
	t.Parallel()

	original := "function global:prompt { \"PS> \" }\n"
	updated, changed, err := upsertPowerShellHookBlock(original, "C:\\Commandry\\cmdry.exe")
	if err != nil {
		t.Fatalf("upsert block failed: %v", err)
	}
	if !changed {
		t.Fatal("expected upsert to change content")
	}
	if !strings.Contains(updated, psHookBeginMarker) || !strings.Contains(updated, psHookEndMarker) {
		t.Fatal("expected markers to be present after upsert")
	}

	second, changed2, err := upsertPowerShellHookBlock(updated, "C:\\Commandry\\cmdry.exe")
	if err != nil {
		t.Fatalf("second upsert failed: %v", err)
	}
	if changed2 {
		t.Fatal("second upsert should be idempotent")
	}
	if second != updated {
		t.Fatal("idempotent upsert returned different content")
	}
}

func TestRemovePowerShellHookBlock(t *testing.T) {
	t.Parallel()

	withBlock, _, err := upsertPowerShellHookBlock("Write-Host \"hello\"\n", "C:\\Commandry\\cmdry.exe")
	if err != nil {
		t.Fatalf("upsert block failed: %v", err)
	}

	updated, changed, err := removePowerShellHookBlock(withBlock)
	if err != nil {
		t.Fatalf("remove block failed: %v", err)
	}
	if !changed {
		t.Fatal("expected remove block to change content")
	}
	if strings.Contains(updated, psHookBeginMarker) || strings.Contains(updated, psHookEndMarker) {
		t.Fatal("expected markers to be removed")
	}
	if !strings.Contains(updated, "Write-Host \"hello\"") {
		t.Fatal("expected original content to remain")
	}
}

func TestReplaceBetweenMarkersMalformed(t *testing.T) {
	t.Parallel()

	_, _, err := replaceBetweenMarkers("x\n"+psHookBeginMarker+"\ny", psHookBeginMarker, psHookEndMarker, "")
	if err == nil {
		t.Fatal("expected malformed markers to return error")
	}
}

func TestHooksHomeDirOverride(t *testing.T) {
	t.Setenv("INFRATRACK_HOME_DIR", "/tmp/infratrack-test-home")
	got, err := hooksHomeDir()
	if err != nil {
		t.Fatalf("hooksHomeDir failed: %v", err)
	}
	if got != "/tmp/infratrack-test-home" {
		t.Fatalf("unexpected hooks home dir: %s", got)
	}
}

func TestPowerShellHookBlockUsesAbsolutePath(t *testing.T) {
	t.Parallel()
	block := powerShellHookBlock("C:\\Commandry\\cmdry.exe")
	if !strings.Contains(block, "& 'C:\\Commandry\\cmdry.exe' hook record") {
		t.Fatalf("expected absolute executable path in hook block, got: %s", block)
	}
	if !strings.Contains(block, "Join-Path $env:APPDATA \"commandry\"") {
		t.Fatalf("expected commandry root path in hook block, got: %s", block)
	}
}

func TestUpsertPowerShellHookBlockReplacesLegacyMarkers(t *testing.T) {
	t.Parallel()

	legacy := strings.Join([]string{
		legacyPSHookBeginMarker,
		"Write-Host legacy",
		legacyPSHookEndMarker,
		"",
	}, "\n")
	updated, changed, err := upsertPowerShellHookBlock(legacy, "C:\\Commandry\\cmdry.exe")
	if err != nil {
		t.Fatalf("upsert legacy block failed: %v", err)
	}
	if !changed {
		t.Fatal("expected legacy replacement change")
	}
	if strings.Contains(updated, legacyPSHookBeginMarker) || strings.Contains(updated, legacyPSHookEndMarker) {
		t.Fatalf("legacy markers must be removed: %s", updated)
	}
	if !strings.Contains(updated, psHookBeginMarker) || !strings.Contains(updated, psHookEndMarker) {
		t.Fatalf("new markers missing: %s", updated)
	}
}
