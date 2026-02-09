package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestCommandAliases(t *testing.T) {
	t.Parallel()

	root, err := NewRootCommand()
	if err != nil {
		t.Fatalf("NewRootCommand failed: %v", err)
	}

	tests := []struct {
		name    string
		input   []string
		wantUse string
	}{
		{name: "init alias", input: []string{"i"}, wantUse: "init"},
		{name: "start alias", input: []string{"s"}, wantUse: "start"},
		{name: "run alias", input: []string{"r"}, wantUse: "run"},
		{name: "stop alias", input: []string{"stp"}, wantUse: "stop"},
		{name: "export alias", input: []string{"x"}, wantUse: "export"},
		{name: "alias command", input: []string{"alias"}, wantUse: "alias"},
		{name: "version alias", input: []string{"v"}, wantUse: "version"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cmd, _, err := root.Find(tc.input)
			if err != nil {
				t.Fatalf("root.Find failed: %v", err)
			}
			if cmd == nil {
				t.Fatalf("command not found for %v", tc.input)
			}
			if cmd.Name() != tc.wantUse {
				t.Fatalf("resolved to %q, want %q", cmd.Name(), tc.wantUse)
			}
		})
	}
}

func TestSessionsCommandExists(t *testing.T) {
	t.Parallel()

	root, err := NewRootCommand()
	if err != nil {
		t.Fatalf("NewRootCommand failed: %v", err)
	}

	cmd, _, err := root.Find([]string{"sessions", "list"})
	if err != nil {
		t.Fatalf("root.Find(sessions list) failed: %v", err)
	}
	if cmd == nil || cmd.Name() != "list" {
		t.Fatalf("sessions list command not found")
	}
}

func TestHooksCommandsExist(t *testing.T) {
	t.Parallel()

	root, err := NewRootCommand()
	if err != nil {
		t.Fatalf("NewRootCommand failed: %v", err)
	}

	hooksStatus, _, err := root.Find([]string{"hooks", "status"})
	if err != nil {
		t.Fatalf("root.Find(hooks status) failed: %v", err)
	}
	if hooksStatus == nil || hooksStatus.Name() != "status" {
		t.Fatalf("hooks status command not found")
	}

	hookRecord, _, err := root.Find([]string{"hook", "record"})
	if err != nil {
		t.Fatalf("root.Find(hook record) failed: %v", err)
	}
	if hookRecord == nil || hookRecord.Name() != "record" {
		t.Fatalf("hook record command not found")
	}
}

func TestAliasCommandOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		shell    string
		contains string
		wantErr  bool
	}{
		{name: "powershell", shell: "powershell", contains: "Set-Alias -Name it -Value infratrack"},
		{name: "bash", shell: "bash", contains: "alias it='infratrack'"},
		{name: "zsh", shell: "zsh", contains: "alias it='infratrack'"},
		{name: "cmd", shell: "cmd", contains: "doskey it=infratrack $*"},
		{name: "unsupported", shell: "fish", wantErr: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			root, err := NewRootCommand()
			if err != nil {
				t.Fatalf("NewRootCommand failed: %v", err)
			}

			var out bytes.Buffer
			root.SetOut(&out)
			root.SetErr(&out)
			root.SetArgs([]string{"alias", "--shell", tc.shell})

			err = root.Execute()
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(out.String(), tc.contains) {
				t.Fatalf("output %q does not contain %q", out.String(), tc.contains)
			}
		})
	}
}

func TestShortFlags(t *testing.T) {
	t.Parallel()

	root, err := NewRootCommand()
	if err != nil {
		t.Fatalf("NewRootCommand failed: %v", err)
	}

	startCmd, _, err := root.Find([]string{"start"})
	if err != nil {
		t.Fatalf("root.Find(start) failed: %v", err)
	}
	if startCmd.Flags().ShorthandLookup("e") == nil {
		t.Fatalf("short flag -e is not configured for start")
	}

	exportCmd, _, err := root.Find([]string{"export"})
	if err != nil {
		t.Fatalf("root.Find(export) failed: %v", err)
	}
	if exportCmd.Flags().ShorthandLookup("l") == nil {
		t.Fatalf("short flag -l is not configured for export")
	}
	if exportCmd.Flags().ShorthandLookup("f") == nil {
		t.Fatalf("short flag -f is not configured for export")
	}
	if exportCmd.Flags().Lookup("session") == nil {
		t.Fatalf("flag --session is not configured for export")
	}
}
