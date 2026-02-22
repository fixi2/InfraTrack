package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDoctorCommandExists(t *testing.T) {
	t.Parallel()

	root, err := NewRootCommand()
	if err != nil {
		t.Fatalf("NewRootCommand failed: %v", err)
	}

	cmd, _, err := root.Find([]string{"doctor"})
	if err != nil {
		t.Fatalf("root.Find(doctor) failed: %v", err)
	}
	if cmd == nil || cmd.Name() != "doctor" {
		t.Fatalf("doctor command not found")
	}
}

func TestDoctorCommandOutput(t *testing.T) {
	t.Parallel()

	root, err := NewRootCommand()
	if err != nil {
		t.Fatalf("NewRootCommand failed: %v", err)
	}

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"doctor"})

	if err := root.Execute(); err != nil {
		t.Fatalf("doctor command failed: %v", err)
	}

	text := out.String()
	for _, want := range []string{
		"=== Doctor ===",
		"Root dir:",
		"=== Tool Availability ===",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("doctor output missing %q in %q", want, text)
		}
	}
}

func TestPathContainsDir(t *testing.T) {
	t.Parallel()

	sep := string(os.PathListSeparator)
	var pathEnv string
	var target string
	var missing string

	if runtime.GOOS == "windows" {
		pathEnv = `C:\bin` + sep + `C:\Tools` + sep + `C:\Windows`
		target = `C:\Tools`
		missing = `C:\Missing`
	} else {
		pathEnv = "/usr/bin" + sep + "/opt/tools" + sep + "/usr/local/bin"
		target = "/opt/tools"
		missing = "/opt/missing"
	}

	if !pathContainsDir(pathEnv, filepath.Clean(target)) {
		t.Fatalf("expected pathContainsDir to find directory")
	}
	if pathContainsDir(pathEnv, filepath.Clean(missing)) {
		t.Fatalf("did not expect pathContainsDir to find missing directory")
	}
}
