package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The standby reader refuses a secret-id file larger than the 64 KiB cap instead of
// sending a silently truncated or oversized value as the login secret_id.
func TestReadStandbySecretIDRefusesOversizedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secret-id")
	if err := os.WriteFile(path, []byte(strings.Repeat("a", 70*1024)), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := readStandbySecretID(path)
	if err == nil {
		t.Fatalf("70 KiB secret-id file accepted (value length %d); must be refused", len(got))
	}
	if got != "" {
		t.Fatal("no secret value may be returned for an oversized file")
	}
	if !strings.Contains(err.Error(), "larger than") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Exactly the cap is accepted, and one byte over is refused, so the limit is off by none.
func TestReadStandbySecretIDCapBoundary(t *testing.T) {
	dir := t.TempDir()
	atCap := filepath.Join(dir, "at-cap")
	if err := os.WriteFile(atCap, []byte(strings.Repeat("b", maxStandbySecretIDBytes)), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := readStandbySecretID(atCap)
	if err != nil {
		t.Fatalf("file at the cap rejected: %v", err)
	}
	if len(got) != maxStandbySecretIDBytes {
		t.Fatalf("value length %d, want %d", len(got), maxStandbySecretIDBytes)
	}

	over := filepath.Join(dir, "over-cap")
	if err := os.WriteFile(over, []byte(strings.Repeat("c", maxStandbySecretIDBytes+1)), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readStandbySecretID(over); err == nil {
		t.Fatal("64 KiB + 1 byte accepted")
	}
}
