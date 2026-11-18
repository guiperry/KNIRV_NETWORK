package main

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// fakeStandbyBao is an OpenBao stand-in served over a unix socket (httptest on a real
// listener). It accepts one role/secret pair, returns a client token, and records revoke
// calls. Only synthetic values are used; no real vault is involved.
type fakeStandbyBao struct {
	mu        sync.Mutex
	loginCode int
	revoked   []string
	// beforeLogin, when set, runs inside the login request, i.e. after the secret-id file was
	// read and before newStandbyProvider deletes it. Tests use it to swap the file in that window.
	beforeLogin func()
}

func startFakeStandbyBao(t *testing.T, loginCode int) (socket string, fb *fakeStandbyBao) {
	t.Helper()
	// Short directory on purpose: unix socket paths are limited to ~108 bytes.
	dir, err := os.MkdirTemp("", "sbc")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	socket = filepath.Join(dir, "b.sock")
	ln, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	fb = &fakeStandbyBao{loginCode: loginCode}
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/auth/approle/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		fb.mu.Lock()
		code := fb.loginCode
		hook := fb.beforeLogin
		fb.mu.Unlock()
		if hook != nil {
			hook()
		}
		if code != http.StatusOK {
			w.WriteHeader(code)
			return
		}
		w.Write([]byte(`{"auth":{"client_token":"tok-synthetic-standby"}}`))
	})
	mux.HandleFunc("/v1/auth/token/revoke-self", func(w http.ResponseWriter, r *http.Request) {
		fb.mu.Lock()
		fb.revoked = append(fb.revoked, r.Header.Get("X-Vault-Token"))
		fb.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})
	srv := httptest.NewUnstartedServer(mux)
	srv.Listener.Close()
	srv.Listener = ln
	srv.Start()
	t.Cleanup(srv.Close)
	return socket, fb
}

func setStandbyEnv(t *testing.T, socket, secretPath string) {
	t.Helper()
	t.Setenv(envStandbyBaoSocket, socket)
	t.Setenv(envStandbyRoleID, "synthetic-standby-role")
	t.Setenv(envStandbySecretIDFile, secretPath)
}

// Q47: a successful standby AppRole login consumes the secret_id file.
func TestNewStandbyProviderDeletesSecretIDFileAfterLogin(t *testing.T) {
	socket, _ := startFakeStandbyBao(t, http.StatusOK)
	secretPath := filepath.Join(t.TempDir(), "secret-id")
	if err := os.WriteFile(secretPath, []byte("synthetic-secret-id\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	setStandbyEnv(t, socket, secretPath)

	if _, err := newStandbyProvider(context.Background()); err != nil {
		t.Fatalf("newStandbyProvider: %v", err)
	}
	if _, err := os.Stat(secretPath); !os.IsNotExist(err) {
		t.Fatalf("secret-id file must be gone after successful login, stat err = %v", err)
	}
}

// Same policy as the enroll path: a refused login does not consume the file.
func TestNewStandbyProviderKeepsSecretIDFileWhenLoginRefused(t *testing.T) {
	socket, fb := startFakeStandbyBao(t, http.StatusForbidden)
	secretPath := filepath.Join(t.TempDir(), "secret-id")
	if err := os.WriteFile(secretPath, []byte("synthetic-secret-id\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	setStandbyEnv(t, socket, secretPath)

	if _, err := newStandbyProvider(context.Background()); err == nil {
		t.Fatal("refused login must fail newStandbyProvider")
	}
	if _, err := os.Stat(secretPath); err != nil {
		t.Fatalf("secret-id file must be kept when login is refused: %v", err)
	}
	if len(fb.revoked) != 0 {
		t.Fatalf("no token was issued, but revoke was called %d times", len(fb.revoked))
	}
}

// Same policy as the enroll path: if the file cannot be removed after a successful login,
// the token is revoked and the standby does not start.
func TestNewStandbyProviderAbortsAndRevokesWhenSecretIDFileCannotBeRemoved(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root bypasses directory write permission; cannot force a remove failure")
	}
	socket, fb := startFakeStandbyBao(t, http.StatusOK)
	dir := t.TempDir()
	secretPath := filepath.Join(dir, "secret-id")
	if err := os.WriteFile(secretPath, []byte("synthetic-secret-id\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })
	setStandbyEnv(t, socket, secretPath)

	provider, err := newStandbyProvider(context.Background())
	if err == nil {
		t.Fatal("undeletable secret-id file must abort newStandbyProvider")
	}
	if provider != nil {
		t.Fatal("no provider may be returned after an aborted start")
	}
	if !strings.Contains(err.Error(), "remove secret-id file after login") {
		t.Fatalf("unexpected error: %v", err)
	}
	fb.mu.Lock()
	revoked := append([]string(nil), fb.revoked...)
	fb.mu.Unlock()
	if len(revoked) != 1 || revoked[0] != "tok-synthetic-standby" {
		t.Fatalf("token must be revoked exactly once on abort, revoke calls = %v", revoked)
	}
}
