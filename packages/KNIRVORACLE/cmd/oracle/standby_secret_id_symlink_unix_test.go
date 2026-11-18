//go:build unix

package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

// N3a: after Lstat, the seam replaces the checked path with a symlink to a different
// owner-only file. The read must be refused, by O_NOFOLLOW (ELOOP) or by the SameFile check.
func TestReadStandbySecretIDRefusesSymlinkSwappedAfterCheck(t *testing.T) {
	dir := t.TempDir()
	checked := filepath.Join(dir, "secret-id")
	other := filepath.Join(dir, "other-secret-id")
	if err := os.WriteFile(checked, []byte("checked-synthetic\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(other, []byte("other-synthetic\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	orig := standbyOpenFile
	t.Cleanup(func() { standbyOpenFile = orig })
	standbyOpenFile = func(name string) (*os.File, error) {
		if err := os.Remove(name); err != nil {
			return nil, err
		}
		if err := os.Symlink(other, name); err != nil {
			return nil, err
		}
		return orig(name)
	}

	got, err := readStandbySecretID(checked)
	if err == nil {
		t.Fatal("symlink swapped in after the check was accepted")
	}
	if got != "" {
		t.Fatal("secret value returned through a swapped symlink")
	}
	if !strings.Contains(err.Error(), "changed between checks") && !errors.Is(err, syscall.ELOOP) {
		t.Fatalf("expected the SameFile refusal or the O_NOFOLLOW error (ELOOP), got: %v", err)
	}
}

// N3b (kills mutant SN): the symlink points to a second hard link of the SAME inode, so
// SameFile would pass if the open followed the link. Only O_NOFOLLOW refuses it, so this
// test requires the ELOOP error specifically. Without O_NOFOLLOW the secret would be read.
func TestReadStandbySecretIDRefusesSymlinkToSameInodeAfterCheck(t *testing.T) {
	dir := t.TempDir()
	checked := filepath.Join(dir, "secret-id")
	alias := filepath.Join(dir, "alias-secret-id")
	if err := os.WriteFile(checked, []byte("checked-synthetic\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(checked, alias); err != nil {
		t.Skipf("hard links unavailable: %v", err)
	}
	orig := standbyOpenFile
	t.Cleanup(func() { standbyOpenFile = orig })
	standbyOpenFile = func(name string) (*os.File, error) {
		if err := os.Remove(name); err != nil {
			return nil, err
		}
		if err := os.Symlink(alias, name); err != nil {
			return nil, err
		}
		return orig(name)
	}

	got, err := readStandbySecretID(checked)
	if err == nil {
		t.Fatal("symlink to the same inode swapped in after the check was followed")
	}
	if got != "" {
		t.Fatal("secret value read through a swapped symlink")
	}
	if !errors.Is(err, syscall.ELOOP) {
		t.Fatalf("expected the O_NOFOLLOW error (ELOOP), got: %v", err)
	}
}
