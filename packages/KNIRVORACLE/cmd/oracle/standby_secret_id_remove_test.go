package main

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// N2: the path is replaced after the secret-id file was read and before it is deleted. The
// login request is the window (the fake Bao runs beforeLogin after the read and before the
// delete). The replacement must not be removed, and the start must abort with the token revoked.
func TestNewStandbyProviderDoesNotRemoveFileSwappedAfterRead(t *testing.T) {
	socket, fb := startFakeStandbyBao(t, http.StatusOK)
	secretPath := filepath.Join(t.TempDir(), "secret-id")
	if err := os.WriteFile(secretPath, []byte("checked-synthetic\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	const swapped = "swapped-synthetic\n"
	fb.mu.Lock()
	// Rename the read file away first, so its inode stays allocated. Otherwise the replacement
	// can reuse the freed inode number and SameFile would wrongly match it.
	held := filepath.Join(filepath.Dir(secretPath), "held-synthetic")
	fb.beforeLogin = func() {
		if err := os.Rename(secretPath, held); err != nil {
			t.Errorf("rename: %v", err)
			return
		}
		if err := os.WriteFile(secretPath, []byte(swapped), 0o600); err != nil {
			t.Errorf("write replacement: %v", err)
		}
	}
	fb.mu.Unlock()
	setStandbyEnv(t, socket, secretPath)

	provider, err := newStandbyProvider(context.Background())
	if err == nil {
		t.Fatal("start must abort when the secret-id path was replaced after the read")
	}
	if provider != nil {
		t.Fatal("no provider may be returned after an aborted start")
	}
	if !strings.Contains(err.Error(), "changed after it was read") {
		t.Fatalf("unexpected error: %v", err)
	}
	data, rerr := os.ReadFile(secretPath)
	if rerr != nil {
		t.Fatalf("replacement file must not be removed: %v", rerr)
	}
	if string(data) != swapped {
		t.Fatal("replacement file content changed")
	}
	fb.mu.Lock()
	revoked := append([]string(nil), fb.revoked...)
	fb.mu.Unlock()
	if len(revoked) != 1 || revoked[0] != "tok-synthetic-standby" {
		t.Fatalf("token must be revoked exactly once on abort, revoke calls = %v", revoked)
	}
}
