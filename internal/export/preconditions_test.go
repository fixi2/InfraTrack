package export

import (
	"reflect"
	"testing"

	"github.com/fixi2/InfraTrack/internal/store"
)

func TestDetectPreconditions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		steps []store.Step
		want  []string
	}{
		{
			name:  "no steps",
			steps: nil,
			want: []string{
				"TODO: Verify required tools and access are available.",
				"TODO: Confirm no secrets are needed in commands.",
			},
		},
		{
			name: "kubectl",
			steps: []store.Step{
				{Command: "kubectl apply -f deploy.yaml"},
			},
			want: []string{
				"Suggested: `kubectl` is installed and available in PATH.",
				"Suggested: Kubernetes context and access are configured (`KUBECONFIG`/current-context).",
				"Suggested: Confirm no secrets are needed in commands.",
			},
		},
		{
			name: "docker and terraform",
			steps: []store.Step{
				{Command: "docker build -t app:latest ."},
				{Command: "terraform apply"},
			},
			want: []string{
				"Suggested: Docker CLI is installed and Docker daemon is running.",
				"Suggested: Current user has permission to access Docker daemon.",
				"Suggested: `terraform` CLI is installed and initialized for this workspace.",
				"Suggested: Backend credentials and target workspace are configured.",
				"Suggested: Confirm no secrets are needed in commands.",
			},
		},
		{
			name: "helm and cloud cli",
			steps: []store.Step{
				{Command: "helm upgrade --install api ./chart"},
				{Command: "aws eks update-kubeconfig --name staging"},
			},
			want: []string{
				"Suggested: `helm` is installed and points to the intended Kubernetes context.",
				"Suggested: Required chart repositories are configured and reachable.",
				"Suggested: Cloud CLI authentication is active for the intended account/project/subscription.",
				"Suggested: Required IAM permissions are available for the target resources.",
				"Suggested: Confirm no secrets are needed in commands.",
			},
		},
		{
			name: "db cli",
			steps: []store.Step{
				{Command: `psql "host=db.example.com user=app"`},
			},
			want: []string{
				"Suggested: Database client access is configured (host, port, user, SSL mode).",
				"Suggested: Use least-privilege credentials and avoid exposing secrets in command arguments.",
				"Suggested: Confirm no secrets are needed in commands.",
			},
		},
		{
			name: "none recognized",
			steps: []store.Step{
				{Command: `cmd /c echo "build started"`},
			},
			want: []string{
				"TODO: Verify required tools and access are available.",
				"TODO: Confirm no secrets are needed in commands.",
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := detectPreconditions(tc.steps)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("detectPreconditions() mismatch\n got: %#v\nwant: %#v", got, tc.want)
			}
		})
	}
}
