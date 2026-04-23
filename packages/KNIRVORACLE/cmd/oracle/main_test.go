package main

import (
	"path/filepath"
	"testing"
)

func TestResolveSocketPathPrefersEnvWhenFlagIsDefault(t *testing.T) {
	t.Setenv("ORACLE_SOCKET_PATH", "/tmp/oracle-from-env.sock")

	got, err := resolveSocketPath("/var/run/knirv/oracle.sock")
	if err != nil {
		t.Fatalf("resolveSocketPath() error = %v", err)
	}
	if got != "/tmp/oracle-from-env.sock" {
		t.Fatalf("resolveSocketPath() = %q, want env path", got)
	}
}

func TestResolveSocketPathFallsBackToAppDataDir(t *testing.T) {
	xdgDataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", xdgDataHome)
	t.Setenv("ORACLE_SOCKET_PATH", "")

	got, err := resolveSocketPath("/var/run/knirv/oracle.sock")
	if err != nil {
		t.Fatalf("resolveSocketPath() error = %v", err)
	}

	want := filepath.Join(xdgDataHome, "knirvserver", "sockets", "oracle.sock")
	if got != want {
		t.Fatalf("resolveSocketPath() = %q, want %q", got, want)
	}
}
