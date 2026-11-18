package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A file swapped in between the Lstat check and the open must be refused, and the swapped-in
// file's content must never be read. The seam returns the real opener applied to a different
// 0600 file, so only the os.SameFile check can catch it. Mutant MI (that check disabled) makes
// this test fail.
func TestReadSecretIDFileRejectsFileSwappedAfterCheck(t *testing.T) {
	dir := t.TempDir()
	checked := filepath.Join(dir, "secret-id")
	swapped := filepath.Join(dir, "swapped-secret-id")
	if err := os.WriteFile(checked, []byte("checked-synthetic\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(swapped, []byte("swapped-synthetic\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	orig := secretIDOpenFile
	t.Cleanup(func() { secretIDOpenFile = orig })
	secretIDOpenFile = func(string) (*os.File, error) {
		return openSecretIDFile(swapped)
	}

	got, err := readSecretIDFile(checked)
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
