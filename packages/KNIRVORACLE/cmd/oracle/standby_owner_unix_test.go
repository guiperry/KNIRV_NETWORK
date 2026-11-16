//go:build unix

package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestReadStandbySecretIDAcceptsCurrentUIDOwner: a 0600 file owned by the process's
// effective uid is accepted.
func TestReadStandbySecretIDAcceptsCurrentUIDOwner(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secret-id")
	if err := os.WriteFile(path, []byte("synthetic-secret-id\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := readStandbySecretID(path)
	if err != nil {
		t.Fatalf("file owned by the current uid rejected: %v", err)
	}
	if got != "synthetic-secret-id" {
		t.Fatalf("unexpected secret-id value length %d", len(got))
	}
}

// TestReadStandbySecretIDRejectsOwnerMismatch simulates a file owned by another uid by
// moving the effective-uid lookup off the file's real owner. No chown, no sudo, and no
// real secret file is touched.
func TestReadStandbySecretIDRejectsOwnerMismatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secret-id")
	if err := os.WriteFile(path, []byte("synthetic-secret-id\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	orig := standbyEffectiveUID
	t.Cleanup(func() { standbyEffectiveUID = orig })
	standbyEffectiveUID = func() int { return os.Geteuid() + 1 }

	if _, err := readStandbySecretID(path); err == nil {
		t.Fatal("secret-id file not owned by the effective uid was accepted")
	}
}
