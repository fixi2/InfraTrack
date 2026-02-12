package export

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/fixi2/InfraTrack/internal/store"
)

func TestRenderMarkdownGolden(t *testing.T) {
	t.Parallel()

	session := &store.Session{
		ID:        "1",
		Title:     "Deploy to staging",
		StartedAt: time.Date(2026, 2, 3, 10, 0, 0, 0, time.UTC),
		Steps: []store.Step{
			{
				Timestamp:  time.Date(2026, 2, 3, 10, 0, 5, 0, time.UTC),
				Command:    "kubectl apply -f deploy.yaml",
				Status:     "OK",
				ExitCode:   intPtr(0),
				DurationMS: 820,
				CWD:        "/repo",
			},
			{
				Timestamp:  time.Date(2026, 2, 3, 10, 0, 9, 0, time.UTC),
				Command:    "kubectl rollout status deploy/api",
				Status:     "OK",
				ExitCode:   intPtr(0),
				DurationMS: 1450,
				CWD:        "/repo",
			},
		},
	}

	got := RenderMarkdown(session)
	goldenPath := filepath.Join("testdata", "session.golden.md")
	wantBytes, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden file: %v", err)
	}

	if normalizeNewlines(got) != normalizeNewlines(string(wantBytes)) {
		t.Fatalf("markdown mismatch\n--- got ---\n%s\n--- want ---\n%s", got, string(wantBytes))
	}
}

func intPtr(v int) *int {
	return &v
}

func normalizeNewlines(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
}

func TestRunbookFilename(t *testing.T) {
	t.Parallel()

	session := &store.Session{
		Title:     "Deploy to staging",
		StartedAt: time.Date(2026, 2, 3, 10, 0, 0, 0, time.UTC),
	}

	got := RunbookFilename(session)
	want := "20260203-100000-deploy-to-staging.md"
	if got != want {
		t.Fatalf("filename mismatch: got %q want %q", got, want)
	}
}

func TestDetectRollback(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		steps      []store.Step
		wantTitle  string
		wantItems  []string
	}{
		{
			name: "rollback suggested from rollout status deployment",
			steps: []store.Step{
				{Command: "kubectl rollout status deployment/api", Status: "OK", ExitCode: intPtr(0)},
			},
			wantTitle: "Rollback (suggested, use with caution)",
			wantItems: []string{
				"Suggested: use with caution. Verify root cause and deployment revision before undo.",
				"Suggested: `kubectl rollout undo deployment/api`",
			},
		},
		{
			name: "rollback suggested from set image deployment",
			steps: []store.Step{
				{Command: "kubectl set image deployment/web api=repo/app:v2", Status: "OK", ExitCode: intPtr(0)},
			},
			wantTitle: "Rollback (suggested, use with caution)",
			wantItems: []string{
				"Suggested: use with caution. Verify root cause and deployment revision before undo.",
				"Suggested: `kubectl rollout undo deployment/web`",
			},
		},
		{
			name: "no rollback for failed rollout status deployment",
			steps: []store.Step{
				{Command: "kubectl rollout status deployment/api", Status: "FAILED", ExitCode: intPtr(1)},
			},
			wantTitle: "Rollback",
			wantItems: []string{
				"TODO: Add rollback commands.",
			},
		},
		{
			name: "no rollback for apply without deployment name",
			steps: []store.Step{
				{Command: "kubectl apply -f deploy.yaml"},
			},
			wantTitle: "Rollback",
			wantItems: []string{
				"TODO: Add rollback commands.",
			},
		},
		{
			name: "no rollback for deploy alias without deployment slash",
			steps: []store.Step{
				{Command: "kubectl rollout status deploy/api"},
			},
			wantTitle: "Rollback",
			wantItems: []string{
				"TODO: Add rollback commands.",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotTitle, gotItems := detectRollback(tt.steps)
			if gotTitle != tt.wantTitle {
				t.Fatalf("title mismatch: got %q want %q", gotTitle, tt.wantTitle)
			}
			if !reflect.DeepEqual(gotItems, tt.wantItems) {
				t.Fatalf("items mismatch: got %#v want %#v", gotItems, tt.wantItems)
			}
		})
	}
}
