package cli

import "testing"

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
}
