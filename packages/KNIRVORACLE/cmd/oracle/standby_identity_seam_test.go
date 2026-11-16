package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestReadStandbySecretIDRejectsFileSwappedAfterCheck covers the os.SameFile branch.
// The open seam returns a different 0600 file than the one Lstat checked, which is the
// TOCTOU swap the check must catch. The swapped-in file's content must never be read.
func TestReadStandbySecretIDRejectsFileSwappedAfterCheck(t *testing.T) {
	dir := t.TempDir()
	checked := filepath.Join(dir, "secret-id")
	swapped := filepath.Join(dir, "swapped-secret-id")
	if err := os.WriteFile(checked, []byte("checked-synthetic\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(swapped, []byte("swapped-synthetic\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	orig := standbyOpenFile
	t.Cleanup(func() { standbyOpenFile = orig })
	standbyOpenFile = func(name string) (*os.File, error) {
		return orig(swapped)
	}

	got, err := readStandbySecretID(checked)
	if err == nil {
		t.Fatal("file swapped between the Lstat check and open was accepted")
	}
	if !strings.Contains(err.Error(), "changed between checks") {
		t.Fatalf("expected the SameFile branch error, got: %v", err)
	}
	if got != "" {
		t.Fatal("secret value returned from a swapped file")
	}
}
