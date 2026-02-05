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
