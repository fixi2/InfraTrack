package policy

import (
	"strings"
	"testing"
)

func TestPolicyApply(t *testing.T) {
	t.Parallel()

	p := NewDefault()

	tests := []struct {
		name     string
		raw      string
		args     []string
		want     string
		denied   bool
		contains string
	}{
		{
			name:   "deny env command",
			raw:    "env",
			args:   []string{"env"},
			want:   DeniedPlaceholder,
			denied: true,
		},
		{
			name:   "deny kubectl secret output",
			raw:    "kubectl get secret app -o yaml",
			args:   []string{"kubectl", "get", "secret", "app", "-o", "yaml"},
			want:   DeniedPlaceholder,
			denied: true,
		},
		{
			name:     "redact bearer token",
			raw:      `curl -H "Authorization: Bearer abc123" https://example.com`,
			args:     []string{"curl"},
			denied:   false,
			contains: "Authorization: Bearer [REDACTED]",
		},
		{
			name:     "redact long options",
			raw:      "deploy --token=abc --password hunter2",
			args:     []string{"deploy"},
			denied:   false,
			contains: "--token=[REDACTED] --password [REDACTED]",
		},
		{
			name:     "redact short option p",
			raw:      "sshpass -p supersecret ssh user@example.com",
			args:     []string{"sshpass"},
			denied:   false,
			contains: "-p [REDACTED]",
		},
		{
			name:     "redact env assignment",
			raw:      "API_KEY=abcdef terraform apply",
			args:     []string{"terraform"},
			denied:   false,
			contains: "API_KEY=[REDACTED]",
		},
		{
			name:     "redact quoted env assignment keeps quote",
			raw:      `psql "host=db.example.com user=app"`,
			args:     []string{"psql", "host=db.example.com user=app"},
			denied:   false,
			contains: `psql "host=[REDACTED] user=[REDACTED]"`,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := p.Apply(tc.raw, tc.args)
			if got.Denied != tc.denied {
				t.Fatalf("denied mismatch: got %v want %v", got.Denied, tc.denied)
			}

			if tc.want != "" && got.Command != tc.want {
				t.Fatalf("command mismatch: got %q want %q", got.Command, tc.want)
			}

			if tc.contains != "" && !strings.Contains(got.Command, tc.contains) {
				t.Fatalf("command %q does not contain %q", got.Command, tc.contains)
			}
		})
	}
}
