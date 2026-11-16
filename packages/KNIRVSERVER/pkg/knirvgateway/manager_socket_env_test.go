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

// TestSocketPathEnvIncludesKnirvAndGitSockets checks that the gateway receives the
// proof-API and git socket paths exactly when they are configured.
func TestSocketPathEnvIncludesKnirvAndGitSockets(t *testing.T) {
	env := socketPathEnv(&ManagerConfig{
		KnirvProofSocketPath: "/run/knirv/knirvproof.sock",
		GitSocketPath:        "/run/knirv/git.sock",
	})
	if !containsEnv(env, "KNIRVPROOF_SOCKET_PATH=/run/knirv/knirvproof.sock") {
		t.Fatalf("KNIRVPROOF_SOCKET_PATH missing from %v", env)
	}
	if !containsEnv(env, "GIT_SOCKET_PATH=/run/knirv/git.sock") {
		t.Fatalf("GIT_SOCKET_PATH missing from %v", env)
	}
	for _, e := range socketPathEnv(&ManagerConfig{}) {
		if strings.HasPrefix(e, "KNIRVPROOF_SOCKET_PATH=") || strings.HasPrefix(e, "GIT_SOCKET_PATH=") {
			t.Fatalf("unexpected entry with empty config: %q", e)
		}
	}
}

// TestSocketPathEnvIncludesKnirvServerAppSocket checks that the gateway receives the
// KNIRVSERVER app socket path exactly when it is configured.
func TestSocketPathEnvIncludesKnirvServerAppSocket(t *testing.T) {
	env := socketPathEnv(&ManagerConfig{KnirvServerAppSocketPath: "/run/knirv/knirvserver-app.sock"})
	if !containsEnv(env, "KNIRVSERVER_APP_SOCKET_PATH=/run/knirv/knirvserver-app.sock") {
		t.Fatalf("KNIRVSERVER_APP_SOCKET_PATH missing from %v", env)
	}
	for _, e := range socketPathEnv(&ManagerConfig{}) {
		if strings.HasPrefix(e, "KNIRVSERVER_APP_SOCKET_PATH=") {
			t.Fatalf("unexpected KNIRVSERVER_APP_SOCKET_PATH with empty config: %q", e)
		}
	}
}
