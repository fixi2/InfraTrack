package buildinfo

import "testing"

func TestStringReturnsVersion(t *testing.T) {
	original := Version
	t.Cleanup(func() {
		Version = original
	})

	Version = "v9.9.9"
	if got := String(); got != "v9.9.9" {
		t.Fatalf("String() = %q, want %q", got, "v9.9.9")
	}
}
