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
		name      string
		steps     []store.Step
		wantTitle string
		wantItems []string
	}{
		{
			name: "rollback suggested from rollout status deployment",
			steps: []store.Step{
				{Command: "kubectl rollout status deployment/api", Status: "OK", ExitCode: intPtr(0)},
			},
			wantTitle: "Rollback",
			wantItems: []string{
				"Verify root cause and deployment revision before undoing changes.",
				"`kubectl rollout undo deployment/api`",
			},
		},
		{
			name: "rollback suggested from set image deployment",
			steps: []store.Step{
				{Command: "kubectl set image deployment/web api=repo/app:v2", Status: "OK", ExitCode: intPtr(0)},
			},
			wantTitle: "Rollback",
			wantItems: []string{
				"Verify root cause and deployment revision before undoing changes.",
				"`kubectl rollout undo deployment/web`",
			},
		},
		{
			name: "no rollback for failed rollout status deployment",
			steps: []store.Step{
				{Command: "kubectl rollout status deployment/api", Status: "FAILED", ExitCode: intPtr(1)},
			},
			wantTitle: "Rollback",
			wantItems: []string{
				"Document the rollback command for this workflow before production use.",
			},
		},
		{
			name: "no rollback for apply without deployment name",
			steps: []store.Step{
				{Command: "kubectl apply -f deploy.yaml"},
			},
			wantTitle: "Rollback",
			wantItems: []string{
				"Document the rollback command for this workflow before production use.",
			},
		},
		{
			name: "no rollback for deploy alias without deployment slash",
			steps: []store.Step{
				{Command: "kubectl rollout status deploy/api"},
			},
			wantTitle: "Rollback",
			wantItems: []string{
				"Document the rollback command for this workflow before production use.",
			},
		},
		{
			name: "no rollback for echoed kubectl rollout",
			steps: []store.Step{
				{Command: "cmd /c echo kubectl rollout status deployment/api", Status: "OK", ExitCode: intPtr(0)},
			},
			wantTitle: "Rollback",
			wantItems: []string{
				"Document the rollback command for this workflow before production use.",
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

func TestRenderMarkdownWithOptionsComments(t *testing.T) {
	t.Parallel()

	session := &store.Session{
		ID:        "1",
		Title:     "Annotated export",
		StartedAt: time.Date(2026, 2, 12, 10, 0, 0, 0, time.UTC),
		Steps: []store.Step{
			{
				Command:    "kubectl apply -f deploy.yaml",
				Status:     "FAILED",
				Reason:     "nonzero_exit",
				ExitCode:   intPtr(1),
				DurationMS: 100,
			},
		},
	}

	got := RenderMarkdownWithOptions(session, MarkdownOptions{
		StepComments: map[int][]string{
			0: {"Check kube context before retrying."},
		},
		GlobalComments: []string{"Reviewed by on-call."},
	})

	if !strings.Contains(got, "Reviewer note:\n- Check kube context before retrying.") {
		t.Fatalf("missing step comment in markdown: %s", got)
	}
	if !strings.Contains(got, "## Export Comments") {
		t.Fatalf("missing export comments section: %s", got)
	}
	if !strings.Contains(got, "Applies to all flagged steps: Reviewed by on-call.") {
		t.Fatalf("missing global flagged comment in export comments: %s", got)
	}
	if !strings.Contains(got, "Results: OK 0 | FAILED 1 | REDACTED 0") {
		t.Fatalf("missing summary counters: %s", got)
	}
	if !strings.Contains(got, "Total duration: 100 ms") {
		t.Fatalf("missing total duration in summary: %s", got)
	}
}

func TestRenderMarkdownWithMultipleReviewerNotes(t *testing.T) {
	t.Parallel()

	session := &store.Session{
		ID:        "1",
		Title:     "Multiple reviewer notes",
		StartedAt: time.Date(2026, 2, 12, 10, 0, 0, 0, time.UTC),
		Steps: []store.Step{
			{
				Command:    "cmd /c exit 1",
				Status:     "FAILED",
				Reason:     "nonzero_exit",
				ExitCode:   intPtr(1),
				DurationMS: 100,
			},
		},
	}

	got := RenderMarkdownWithOptions(session, MarkdownOptions{
		StepComments: map[int][]string{
			0: {"First note", "Second note"},
		},
	})

	if !strings.Contains(got, "Reviewer notes:\n- First note\n- Second note") {
		t.Fatalf("missing reviewer notes section in markdown: %s", got)
	}
}

func TestStepTitleSnippetTruncatesLongCommand(t *testing.T) {
	t.Parallel()

	cmd := strings.Repeat("x", 200)
	got := stepTitleSnippet(cmd)
	if len([]rune(got)) > 72 {
		t.Fatalf("snippet exceeds max length: %d", len([]rune(got)))
	}
	if !strings.HasSuffix(got, "...") {
		t.Fatalf("snippet should end with ellipsis, got %q", got)
	}
}

func TestRenderMarkdownSummaryCountsInlineRedaction(t *testing.T) {
	t.Parallel()

	session := &store.Session{
		ID:        "1",
		Title:     "Inline redaction summary",
		StartedAt: time.Date(2026, 2, 22, 10, 0, 0, 0, time.UTC),
		Steps: []store.Step{
			{
				Command:    `cmd /c "echo TOKEN=[REDACTED]"`,
				Status:     "OK",
				ExitCode:   intPtr(0),
				DurationMS: 10,
			},
		},
	}

	got := RenderMarkdown(session)
	if !strings.Contains(got, "Results: OK 1 | FAILED 0 | REDACTED 1") {
		t.Fatalf("summary must count inline redaction: %s", got)
	}
}
