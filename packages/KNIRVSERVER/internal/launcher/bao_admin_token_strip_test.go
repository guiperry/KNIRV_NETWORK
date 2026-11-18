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

// TestBuildBackendEnvStripsAdminTokenFromInheritedAndLauncherEntries proves the backend
// child env built by buildBackendEnv (the function startBackend calls) carries no
// KNIRV_BAO_TOKEN, whether it came from the inherited environment or from the launcher's
// own entries, and keeps the handoff entries. This kills a removed strip call (mutant MF).
func TestBuildBackendEnvStripsAdminTokenFromInheritedAndLauncherEntries(t *testing.T) {
	inherited := []string{
		"PATH=/usr/bin",
		"KNIRV_BAO_TOKEN=fake-admin-token-not-real",
	}
	launcher := append([]string{
		"KNIRV_API_PORT=8082",
		"KNIRV_BAO_TOKEN=fake-admin-token-launcher-entry",
	}, HandoffEnv("/run/bao.sock", "/tmp/x/bao-backend-wrap.txt")...)
	inheritedCopy := append([]string(nil), inherited...)
	launcherCopy := append([]string(nil), launcher...)

	out := buildBackendEnv(inherited, launcher...)
	for _, entry := range out {
		if strings.HasPrefix(entry, "KNIRV_BAO_TOKEN=") {
			t.Fatalf("KNIRV_BAO_TOKEN entry survived: %q", entry)
		}
	}
	joined := strings.Join(out, "\n")
	if strings.Contains(joined, "fake-admin-token") {
		t.Fatalf("admin token value leaked into backend env")
	}
	want := []string{
		"PATH=/usr/bin",
		"KNIRV_API_PORT=8082",
		"KNIRV_BAO_SOCKET=/run/bao.sock",
		"KNIRV_BAO_WRAP_TOKEN_FILE=/tmp/x/bao-backend-wrap.txt",
	}
	if len(out) != len(want) {
		t.Fatalf("got %v, want %v", out, want)
	}
	for i := range want {
		if out[i] != want[i] {
			t.Fatalf("entry %d = %q, want %q", i, out[i], want[i])
		}
	}
	for i := range inherited {
		if inherited[i] != inheritedCopy[i] {
			t.Fatal("inherited slice mutated")
		}
	}
	for i := range launcher {
		if launcher[i] != launcherCopy[i] {
			t.Fatal("launcher slice mutated")
		}
	}
}
