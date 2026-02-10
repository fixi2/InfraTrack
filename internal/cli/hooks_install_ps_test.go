package cli

import (
	"strings"
	"testing"
)

func TestUpsertPowerShellHookBlock(t *testing.T) {
	t.Parallel()

	original := "function global:prompt { \"PS> \" }\n"
	updated, changed, err := upsertPowerShellHookBlock(original)
	if err != nil {
		t.Fatalf("upsert block failed: %v", err)
	}
	if !changed {
		t.Fatal("expected upsert to change content")
	}
	if !strings.Contains(updated, psHookBeginMarker) || !strings.Contains(updated, psHookEndMarker) {
		t.Fatal("expected markers to be present after upsert")
	}

	second, changed2, err := upsertPowerShellHookBlock(updated)
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

	withBlock, _, err := upsertPowerShellHookBlock("Write-Host \"hello\"\n")
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
