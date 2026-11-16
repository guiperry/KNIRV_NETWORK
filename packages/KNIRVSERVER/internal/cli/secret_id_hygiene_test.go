package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// Q47: the single-use secret_id file is deleted right after a successful AppRole login.
func TestResolveKeySourceBaoDeletesSecretIDFileAfterLogin(t *testing.T) {
	socket, _ := startMockBao(t, "s.resolved-token", 200)
	secretPath := filepath.Join(t.TempDir(), "secret-id")
	if err := os.WriteFile(secretPath, []byte("secret-1"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, release, err := resolveKeySource(context.Background(), "bao", socket, "role-1", secretPath, false, "", "")
	if err != nil {
		t.Fatalf("resolveKeySource: %v", err)
	}
	defer release()
	if _, err := os.Stat(secretPath); !os.IsNotExist(err) {
		t.Fatalf("secret-id file must be gone after successful login, stat err = %v", err)
	}
}

// Q47: a refused login does not consume the file, so it is not deleted.
func TestResolveKeySourceBaoKeepsSecretIDFileWhenLoginRefused(t *testing.T) {
	socket, _ := startMockBao(t, "unused", 403)
	secretPath := filepath.Join(t.TempDir(), "secret-id")
	if err := os.WriteFile(secretPath, []byte("secret-1"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := resolveKeySource(context.Background(), "bao", socket, "role-1", secretPath, false, "", ""); err == nil {
		t.Fatal("refused login must fail resolveKeySource")
	}
	if _, err := os.Stat(secretPath); err != nil {
		t.Fatalf("secret-id file must be kept when login is refused: %v", err)
	}
}

// Q47: if the file cannot be removed after login, the run aborts and the session is revoked.
func TestResolveKeySourceBaoAbortsWhenSecretIDFileCannotBeRemoved(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root bypasses directory write permission; cannot force a remove failure")
	}
	socket, m := startMockBao(t, "s.resolved-token", 200)
	dir := t.TempDir()
	secretPath := filepath.Join(dir, "secret-id")
	if err := os.WriteFile(secretPath, []byte("secret-1"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })
	if _, _, err := resolveKeySource(context.Background(), "bao", socket, "role-1", secretPath, false, "", ""); err == nil {
		t.Fatal("undeletable secret-id file must abort resolveKeySource")
	}
	m.mu.Lock()
	revoked := len(m.revoked)
	m.mu.Unlock()
	if revoked != 1 {
		t.Fatalf("session must be revoked on abort, revoke calls = %d", revoked)
	}
}

// Q56: a private file the current user owns is read normally.
func TestReadSecretIDFileAcceptsOwnPrivateFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secret-id")
	if err := os.WriteFile(path, []byte("  secret-own \n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := readSecretIDFile(path)
	if err != nil || got != "secret-own" {
		t.Fatalf("own private file: got %q err %v", got, err)
	}
}
