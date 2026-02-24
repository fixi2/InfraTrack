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
				"Required tools are installed and available in PATH.",
				"Credentials and environment context are set for the target system.",
			},
		},
		{
			name: "kubectl",
			steps: []store.Step{
				{Command: "kubectl apply -f deploy.yaml"},
			},
			want: []string{
				"`kubectl` is installed and available in PATH.",
				"Kubernetes context and access are configured (`KUBECONFIG`/current-context).",
				"Sensitive values are not exposed in command arguments.",
			},
		},
		{
			name: "docker and terraform",
			steps: []store.Step{
				{Command: "docker build -t app:latest ."},
				{Command: "terraform apply"},
			},
			want: []string{
				"Docker CLI is installed and Docker daemon is running.",
				"Current user has permission to access Docker daemon.",
				"`terraform` CLI is installed and initialized for this workspace.",
				"Backend credentials and target workspace are configured.",
				"Sensitive values are not exposed in command arguments.",
			},
		},
		{
			name: "helm and cloud cli",
			steps: []store.Step{
				{Command: "helm upgrade --install api ./chart"},
				{Command: "aws eks update-kubeconfig --name staging"},
			},
			want: []string{
				"`helm` is installed and targets the intended Kubernetes context.",
				"Required chart repositories are configured and reachable.",
				"Cloud CLI authentication is active for the intended account/project/subscription.",
				"Required IAM permissions are available for the target resources.",
				"Sensitive values are not exposed in command arguments.",
			},
		},
		{
			name: "db cli",
			steps: []store.Step{
				{Command: `psql "host=db.example.com user=app"`},
			},
			want: []string{
				"Database client access is configured (host, port, user, SSL mode).",
				"Use least-privilege credentials and avoid exposing secrets in command arguments.",
				"Sensitive values are not exposed in command arguments.",
			},
		},
		{
			name: "none recognized",
			steps: []store.Step{
				{Command: `cmd /c echo "build started"`},
			},
			want: []string{
				"Required tools are installed and available in PATH.",
				"Credentials and environment context are set for the target system.",
			},
		},
		{
			name: "ignore echoed kubectl command",
			steps: []store.Step{
				{Command: `cmd /c echo kubectl apply -f deploy.yaml`},
			},
			want: []string{
				"Required tools are installed and available in PATH.",
				"Credentials and environment context are set for the target system.",
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
