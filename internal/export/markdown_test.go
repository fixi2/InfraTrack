package export

import (
	"os"
	"path/filepath"
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

	if got != string(wantBytes) {
		t.Fatalf("markdown mismatch\n--- got ---\n%s\n--- want ---\n%s", got, string(wantBytes))
	}
}

func intPtr(v int) *int {
	return &v
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
