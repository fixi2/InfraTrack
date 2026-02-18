package blackbox

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
)

// Contract: C2, C3, C4
func TestVolumeRecordingAndExport(t *testing.T) {
	t.Parallel()
	count := envInt("IT_BLACKBOX_VOLUME_COUNT", 1000)
	if count <= 0 {
		t.Skip("volume disabled")
	}

	h := newHarness(t)
	h.initSession(fmt.Sprintf("volume-%d", count))

	for i := 0; i < count; i++ {
		msg := fmt.Sprintf("v-%d", i)
		h.mustRun(append([]string{"run", "--"}, shellEchoCommand(msg)...)...)
	}
	stop := h.stopSession()
	if !strings.Contains(stop.Stdout, fmt.Sprintf("%d recorded step(s)", count)) {
		t.Fatalf("expected %d recorded steps, got:\n%s", count, stop.Stdout)
	}

	runbook := readFile(t, h.exportLastMD())
	if !strings.Contains(runbook, fmt.Sprintf("Recorded %d step(s).", count)) {
		t.Fatalf("runbook summary does not match count %d", count)
	}
}

func envInt(name string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
