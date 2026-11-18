//go:build unix

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A mode widened after the Lstat check but before the read must be refused. The owner and
// mode checks run on the opened handle, so the seam chmods the checked file to 0644 in the
// open window and the handle's mode is what gets checked.
func TestReadStandbySecretIDRefusesModeChangedAfterCheck(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secret-id")
	if err := os.WriteFile(path, []byte("checked-synthetic\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	orig := standbyOpenFile
	t.Cleanup(func() { standbyOpenFile = orig })
	standbyOpenFile = func(name string) (*os.File, error) {
		if err := os.Chmod(name, 0o644); err != nil {
			return nil, err
		}
		return orig(name)
	}

	got, err := readStandbySecretID(path)
	if err == nil {
		t.Fatal("file whose mode was widened after the check was accepted")
	}
	if got != "" {
		t.Fatal("secret value returned from a widened file")
	}
	if !strings.Contains(err.Error(), "mode 0600") {
		t.Fatalf("expected the mode refusal, got: %v", err)
	}
}
