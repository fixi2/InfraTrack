package blackbox

import (
	"strings"
	"testing"
)

// Contract: C2
func TestSessionLifecycleBlackBox(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	h.mustRun("init")
	noStartRun := h.run(append([]string{"run", "--"}, shellEchoCommand("no-start")...)...)
	if noStartRun.ExitCode == 0 {
		t.Fatalf("run without start must fail")
	}

	h.mustRun("start", "core-e2e")
	h.mustRun(append([]string{"run", "--"}, shellEchoCommand("hello")...)...)
	nonzero := h.run(append([]string{"run", "--"}, shellExitNonZeroCommand(7)...)...)
	if nonzero.ExitCode == 0 {
		t.Fatalf("non-zero command must fail")
	}
	notFound := h.run("run", "--", "it-command-does-not-exist-987654")
	if notFound.ExitCode == 0 {
		t.Fatalf("command-not-found scenario must fail")
	}

	stop := h.stopSession()
	if !strings.Contains(stop.Stdout, "Stopped session") {
		t.Fatalf("stop output missing contract text:\n%s", stop.Stdout)
	}

	afterStop := h.run(append([]string{"run", "--"}, shellEchoCommand("after-stop")...)...)
	if afterStop.ExitCode == 0 {
		t.Fatalf("run after stop must fail")
	}

	runbookPath := h.exportLastMD()
	runbook := readFile(t, runbookPath)
	if !strings.Contains(runbook, "Result: OK") {
		t.Fatalf("runbook missing OK result:\n%s", runbook)
	}
	if !strings.Contains(runbook, "FAILED") {
		t.Fatalf("runbook missing failed result:\n%s", runbook)
	}
}

// Contract: C4
func TestExportLastEqualsExportSessionID(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.initSession("export-eq")
	h.mustRun(append([]string{"run", "--"}, shellEchoCommand("eq")...)...)
	h.stopSession()

	list := h.mustRun("sessions", "list", "-n", "1").Stdout
	lines := strings.Split(strings.TrimSpace(list), "\n")
	if len(lines) < 2 {
		t.Fatalf("sessions list missing data:\n%s", list)
	}
	sessionID := strings.TrimSpace(strings.Split(lines[1], "\t")[0])
	if sessionID == "" {
		t.Fatalf("failed to parse session id:\n%s", list)
	}

	byLast := parseRunbookPath(h.mustRun("export", "--last", "-f", "md").Stdout)
	byID := parseRunbookPath(h.mustRun("export", "--session", sessionID, "-f", "md").Stdout)
	if byLast == "" || byID == "" {
		t.Fatalf("failed to parse runbook paths")
	}
	c1 := normalizeRunbook(readFile(t, byLast))
	c2 := normalizeRunbook(readFile(t, byID))
	if c1 != c2 {
		t.Fatalf("export --last and --session content differ")
	}
}
