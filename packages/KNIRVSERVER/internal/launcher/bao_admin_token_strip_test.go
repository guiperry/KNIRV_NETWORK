package launcher

import (
	"strings"
	"testing"
)

// TestStripBaoAdminTokenRemovesAmbientAdminToken proves the Q43 contract: the
// KNIRV_BAO_TOKEN admin credential is removed from the child env copy, and nothing else is.
func TestStripBaoAdminTokenRemovesAmbientAdminToken(t *testing.T) {
	in := []string{
		"PATH=/usr/bin",
		"KNIRV_BAO_TOKEN=fake-admin-token-not-real",
		"KNIRV_BAO_SOCKET=/run/bao.sock",
		"KNIRV_BAO_WRAP_TOKEN_FILE=/tmp/x/bao-backend-wrap.txt",
		"KNIRV_BAO_TOKENX=keep-me",
		"VALIDATION_CHAIN_BAO_TOKEN=fake-validation-token",
	}
	out := StripBaoAdminToken(in)
	joined := strings.Join(out, "\n")
	if strings.Contains(joined, "fake-admin-token-not-real") {
		t.Fatalf("admin token leaked into child env: %v", out)
	}
	for _, entry := range out {
		if strings.HasPrefix(entry, "KNIRV_BAO_TOKEN=") {
			t.Fatalf("KNIRV_BAO_TOKEN entry survived: %q", entry)
		}
	}
	want := []string{
		"PATH=/usr/bin",
		"KNIRV_BAO_SOCKET=/run/bao.sock",
		"KNIRV_BAO_WRAP_TOKEN_FILE=/tmp/x/bao-backend-wrap.txt",
		"KNIRV_BAO_TOKENX=keep-me",
		"VALIDATION_CHAIN_BAO_TOKEN=fake-validation-token",
	}
	if len(out) != len(want) {
		t.Fatalf("got %d entries, want %d: %v", len(out), len(want), out)
	}
	for i := range want {
		if out[i] != want[i] {
			t.Fatalf("entry %d = %q, want %q", i, out[i], want[i])
		}
	}
}

// TestStripBaoAdminTokenDoesNotMutateInput proves the caller's slice is untouched.
func TestStripBaoAdminTokenDoesNotMutateInput(t *testing.T) {
	in := []string{"A=1", "KNIRV_BAO_TOKEN=fake", "B=2"}
	_ = StripBaoAdminToken(in)
	if in[1] != "KNIRV_BAO_TOKEN=fake" || len(in) != 3 {
		t.Fatalf("input slice mutated: %v", in)
	}
}
