package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The enroll reader refuses a secret_id file larger than the 64 KiB cap instead of
// returning a silently truncated prefix.
func TestReadSecretIDFileRefusesOversizedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secret-id")
	big := []byte(strings.Repeat("a", 70*1024))
	if err := os.WriteFile(path, big, 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := readSecretIDFile(path)
	if err == nil {
		t.Fatalf("70 KiB secret-id file accepted (value length %d); must be refused, not truncated", len(got))
	}
	if got != "" {
		t.Fatal("no secret value may be returned for an oversized file")
	}
	if !strings.Contains(err.Error(), "larger than") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Exactly the cap is accepted, so the limit is off by none.
func TestReadSecretIDFileAcceptsFileAtCap(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secret-id")
	if err := os.WriteFile(path, []byte(strings.Repeat("b", maxSecretIDBytes)), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := readSecretIDFile(path)
	if err != nil {
		t.Fatalf("file at the cap rejected: %v", err)
	}
	if len(got) != maxSecretIDBytes {
		t.Fatalf("value length %d, want %d", len(got), maxSecretIDBytes)
	}
}

// Symlinks are refused on this path, matching the standby reader, even when the target is
// an owner-only file.
func TestReadSecretIDFileRefusesSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "real-secret-id")
	if err := os.WriteFile(target, []byte("synthetic-secret-id\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "secret-id")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := readSecretIDFile(link); err == nil {
		t.Fatal("symlinked secret-id file accepted")
	}
	if _, err := readSecretIDFile(target); err != nil {
		t.Fatalf("direct 0600 file rejected: %v", err)
	}
}

// The owner check is called on the file actually read; a failing owner check must stop the
// read. This is the test that kills a removed owner-check call (mutant MA).
func TestReadSecretIDFileOwnerCheckSeamIsEnforced(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secret-id")
	if err := os.WriteFile(path, []byte("synthetic-secret-id\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	orig := secretIDOwnerCheck
	t.Cleanup(func() { secretIDOwnerCheck = orig })
	sentinel := errors.New("owner check refused (synthetic)")
	calls := 0
	secretIDOwnerCheck = func(fi os.FileInfo) error {
		calls++
		if !fi.Mode().IsRegular() {
			t.Errorf("owner check received a non-regular file info")
		}
		return sentinel
	}
	got, err := readSecretIDFile(path)
	if !errors.Is(err, sentinel) {
		t.Fatalf("owner check failure not enforced: err = %v", err)
	}
	if got != "" {
		t.Fatal("secret value returned despite failed owner check")
	}
	if calls != 1 {
		t.Fatalf("owner check called %d times, want 1", calls)
	}
}
