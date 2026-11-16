package cli

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// mockBao serves the two OpenBao endpoints the enrollment CLI uses, on a Unix socket.
// It is a mock: no real vault is contacted or initialised.
type mockBao struct {
	mu        sync.Mutex
	loginSeen map[string]string
	revoked   []string
	loginCode int
	token     string
}

func startMockBao(t *testing.T, token string, loginCode int) (socket string, m *mockBao) {
	t.Helper()
	dir, err := os.MkdirTemp("", "bao")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	socket = filepath.Join(dir, "bao.sock")
	ln, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	m = &mockBao{loginSeen: map[string]string{}, loginCode: loginCode, token: token}
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/auth/approle/login", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		m.mu.Lock()
		m.loginSeen = body
		m.mu.Unlock()
		if m.loginCode != http.StatusOK {
			w.WriteHeader(m.loginCode)
			_, _ = w.Write([]byte(`{"errors":["permission denied"]}`))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"auth": map[string]any{"client_token": m.token}})
	})
	// The SDK's Select probes seal status before it builds the bao provider.
	mux.HandleFunc("/v1/sys/seal-status", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]bool{"initialized": true, "sealed": false})
	})
	mux.HandleFunc("/v1/auth/token/revoke-self", func(w http.ResponseWriter, r *http.Request) {
		m.mu.Lock()
		m.revoked = append(m.revoked, r.Header.Get("X-Vault-Token"))
		m.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})
	srv := &http.Server{Handler: mux}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })
	return socket, m
}

func TestLoginBaoAppRoleAndRevokeSelf(t *testing.T) {
	socket, m := startMockBao(t, "s.short-lived-token", http.StatusOK)
	sess, err := loginBaoAppRole(context.Background(), socket, "role-1", "secret-1")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if sess.Token() != "s.short-lived-token" {
		t.Fatalf("token = %q", sess.Token())
	}
	if m.loginSeen["role_id"] != "role-1" || m.loginSeen["secret_id"] != "secret-1" {
		t.Fatalf("login body = %v", m.loginSeen)
	}
	if err := sess.Close(context.Background()); err != nil {
		t.Fatalf("close: %v", err)
	}
	if len(m.revoked) != 1 || m.revoked[0] != "s.short-lived-token" {
		t.Fatalf("revoke-self not called with the session token: %v", m.revoked)
	}
}

func TestLoginBaoAppRoleRefusalDoesNotEchoReply(t *testing.T) {
	socket, _ := startMockBao(t, "unused", http.StatusForbidden)
	_, err := loginBaoAppRole(context.Background(), socket, "role-1", "secret-1")
	if err == nil || !strings.Contains(err.Error(), "HTTP 403") {
		t.Fatalf("refused login must report the status, got %v", err)
	}
	if strings.Contains(err.Error(), "permission denied") || strings.Contains(err.Error(), "secret-1") {
		t.Fatalf("error leaked reply or secret: %v", err)
	}
}

func TestReadSecretIDFileRequiresPrivateMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "secret-id")
	if err := os.WriteFile(path, []byte("secret-1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := readSecretIDFile(path); err == nil || !strings.Contains(err.Error(), "0600") {
		t.Fatalf("world-readable secret-id file must be refused, got %v", err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := readSecretIDFile(path)
	if err != nil || got != "secret-1" {
		t.Fatalf("private secret-id file: got %q err %v", got, err)
	}
}

func TestResolveKeySourceBaoLogsInAndReleases(t *testing.T) {
	socket, m := startMockBao(t, "s.resolved-token", http.StatusOK)
	secretPath := filepath.Join(t.TempDir(), "secret-id")
	if err := os.WriteFile(secretPath, []byte("secret-1"), 0o600); err != nil {
		t.Fatal(err)
	}
	keys, release, err := resolveKeySource(context.Background(), "bao", socket, "role-1", secretPath, false, "", "")
	if err != nil {
		t.Fatalf("resolveKeySource: %v", err)
	}
	if keys == nil {
		t.Fatal("expected a key source")
	}
	if m.loginSeen["role_id"] != "role-1" {
		t.Fatalf("login not performed with role id: %v", m.loginSeen)
	}
	release()
	if len(m.revoked) != 1 || m.revoked[0] != "s.resolved-token" {
		t.Fatalf("release must revoke the session token: %v", m.revoked)
	}
}
