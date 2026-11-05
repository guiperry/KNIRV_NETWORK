//go:build unix

package knirvbao

import (
	"os"
	"path/filepath"
	"testing"
)

func TestManagerLockDoesNotFollowSymlink(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(t.TempDir(), "existing")
	if err := os.WriteFile(target, []byte("untouched"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(root, "manager.lock")); err != nil {
		t.Fatal(err)
	}
	m := &Manager{}
	if err := m.lockRoot(root); err == nil {
		m.releaseLock()
		t.Fatal("symlinked lock was accepted")
	}
	data, err := os.ReadFile(target)
	if err != nil || string(data) != "untouched" {
		t.Fatal("lock target was modified")
	}
}
