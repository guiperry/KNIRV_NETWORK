//go:build unix

package cli

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

type fakeOwnerInfo struct{ uid uint32 }

func (f fakeOwnerInfo) Name() string       { return "secret-id" }
func (f fakeOwnerInfo) Size() int64        { return 8 }
func (f fakeOwnerInfo) Mode() os.FileMode  { return 0o600 }
func (f fakeOwnerInfo) ModTime() time.Time { return time.Time{} }
func (f fakeOwnerInfo) IsDir() bool        { return false }
func (f fakeOwnerInfo) Sys() any           { return &syscall.Stat_t{Uid: f.uid} }

// Q56: a file owned by another uid is refused before its contents are read.
func TestCheckSecretIDOwnerRefusesForeignUID(t *testing.T) {
	foreign := uint32(os.Getuid()) + 1
	if err := checkSecretIDOwner(fakeOwnerInfo{uid: foreign}); err == nil || !strings.Contains(err.Error(), "owned by the current user") {
		t.Fatalf("foreign-owned file must be refused, got %v", err)
	}
}

func TestCheckSecretIDOwnerAcceptsCurrentUID(t *testing.T) {
	if err := checkSecretIDOwner(fakeOwnerInfo{uid: uint32(os.Getuid())}); err != nil {
		t.Fatalf("current-uid file must be accepted: %v", err)
	}
	path := filepath.Join(t.TempDir(), "secret-id")
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := checkSecretIDOwner(fi); err != nil {
		t.Fatalf("real own file must be accepted: %v", err)
	}
}
