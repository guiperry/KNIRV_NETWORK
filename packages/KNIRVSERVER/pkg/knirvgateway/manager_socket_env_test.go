package knirvgateway

import (
	"strings"
	"testing"
)

// TestSocketPathEnvIncludesEnrollSocketOnlyWhenSet checks that the gateway receives
// ENROLL_SOCKET_PATH exactly when an enrollment socket is configured, and that an empty
// path adds no entry (so the gateway registers no enrollment route).
func TestSocketPathEnvIncludesEnrollSocketOnlyWhenSet(t *testing.T) {
	with := socketPathEnv(&ManagerConfig{EnrollSocketPath: "/run/knirv/enroll.sock"})
	if !containsEnv(with, "ENROLL_SOCKET_PATH=/run/knirv/enroll.sock") {
		t.Fatalf("ENROLL_SOCKET_PATH missing from %v", with)
	}
	without := socketPathEnv(&ManagerConfig{})
	for _, e := range without {
		if strings.HasPrefix(e, "ENROLL_SOCKET_PATH=") {
			t.Fatalf("unexpected ENROLL_SOCKET_PATH with empty config: %q", e)
		}
	}
}

func containsEnv(env []string, want string) bool {
	for _, e := range env {
		if e == want {
			return true
		}
	}
	return false
}
