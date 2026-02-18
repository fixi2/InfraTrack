package blackbox

import "testing"

// Contract: C5
func TestAliasAndFullCommandsEquivalent(t *testing.T) {
	t.Parallel()
	h1 := newHarness(t)
	h1.mustRun("init")
	h1.mustRun("start", "alias-case")
	h1.mustRun(append([]string{"run", "--"}, shellEchoCommand("alias")...)...)
	h1.mustRun("stp")
	r1 := normalizeRunbook(readFile(t, h1.exportLastMD()))

	h2 := newHarness(t)
	h2.mustRun("init")
	h2.mustRun("s", "alias-case")
	h2.mustRun(append([]string{"run", "--"}, shellEchoCommand("alias")...)...)
	h2.mustRun("stop")
	r2 := normalizeRunbook(readFile(t, h2.exportLastMD()))

	if r1 != r2 {
		t.Fatalf("alias/full command flows are not equivalent")
	}
}

// Contract: C5
func TestEquivalentExportFlags(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.initSession("flag-eq")
	h.mustRun(append([]string{"run", "--"}, shellEchoCommand("flag-eq")...)...)
	h.stopSession()

	p1 := parseRunbookPath(h.mustRun("export", "--last", "--md").Stdout)
	p2 := parseRunbookPath(h.mustRun("export", "--last", "--format", "md").Stdout)
	if normalizeRunbook(readFile(t, p1)) != normalizeRunbook(readFile(t, p2)) {
		t.Fatalf("--md and --format md are not equivalent")
	}
}
