package export

import (
	"reflect"
	"testing"

	"github.com/fixi2/InfraTrack/internal/store"
)

func TestDetectVerificationChecks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		steps []store.Step
		want  []string
	}{
		{
			name: "kubectl apply",
			steps: []store.Step{
				{Command: "kubectl apply -f deploy.yaml"},
			},
			want: []string{
				"Suggested check: `kubectl get pods` reports expected pod status.",
				"Suggested check: `kubectl rollout status deployment/<name>` completes successfully.",
			},
		},
		{
			name: "kubectl rollout status",
			steps: []store.Step{
				{Command: "kubectl rollout status deployment/api"},
			},
			want: []string{
				"Suggested check: `kubectl get pods` reports expected pod status.",
				"Suggested check: `kubectl rollout status deployment/<name>` completes successfully.",
			},
		},
		{
			name: "no kubectl",
			steps: []store.Step{
				{Command: `cmd /c echo "hello"`},
			},
			want: []string{
				"TODO: Define verification checks.",
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := detectVerificationChecks(tc.steps)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("detectVerificationChecks() mismatch\n got: %#v\nwant: %#v", got, tc.want)
			}
		})
	}
}
