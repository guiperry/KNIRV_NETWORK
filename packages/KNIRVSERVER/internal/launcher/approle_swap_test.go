package launcher

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"knirv-server/internal/knirvbao"
)

// fakeBao is an in-process OpenBao over a unix socket that records the swap sequence:
// it mints a secret-id for the active role, logs in, and revokes the old token.
type fakeBao struct {
		mu         sync.Mutex
	revokedTok string
	authTok    string
	loginBody  map[string]string
	secretID   string
	clientTok  string
}

func (f *fakeBao) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/auth/approle/role/" + knirvbao.RoleOracleActive + "/secret-id":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]string{"secret_id": f.secretID, "secret_id_accessor": "acc"},
			})
		case "/v1/auth/approle/login":
			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)
			f.mu.Lock()
			f.loginBody = body
			f.mu.Unlock()
			_ = json.NewEncoder(w).Encode(map[string]any{
				"auth": map[string]string{"client_token": f.clientTok},
			})
		case "/v1/auth/token/revoke":
			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)
			f.mu.Lock()
			f.authTok = r.Header.Get("X-Vault-Token")
			f.revokedTok = body["token"]
			f.mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}
}

func startFakeBao(t *testing.T, fb *fakeBao) string {
	t.Helper()
	sock := filepath.Join(t.TempDir(), "bao.sock")
	l, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := &http.Server{Handler: fb.handler()}
	go func() { _ = srv.Serve(l) }()
	t.Cleanup(func() { _ = srv.Close(); _ = l.Close() })
	return sock
}

// TestLauncherSwapChildToActive exercises the real launcher call site (SwapChildToActive)
// against a fake OpenBao over a unix socket: the swap must issue a fresh secret-id on the
// active role, log in for a new token, and revoke the previously-held (standby) token.
func TestLauncherSwapChildToActive(t *testing.T) {
	fb := &fakeBao{secretID: "sid-active", clientTok: "tok-active"}
	sock := startFakeBao(t, fb)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	newTok, err := SwapChildToActive(ctx, sock, "role-oracle-active", "tok-old-standby")
	if err != nil {
		t.Fatalf("swap: %v", err)
	}
	if newTok != "tok-active" {
		t.Fatalf("expected new active token, got %q", newTok)
	}

	fb.mu.Lock()
	defer fb.mu.Unlock()
	if fb.loginBody["role_id"] != "role-oracle-active" {
		t.Fatalf("login must use the provided role_id, got %q", fb.loginBody["role_id"])
	}
	if fb.loginBody["secret_id"] != "sid-active" {
		t.Fatalf("login must use the freshly issued active secret-id, got %q", fb.loginBody["secret_id"])
	}
	if fb.revokedTok != "tok-old-standby" {
		t.Fatalf("old standby token must be revoked, got target=%q", fb.revokedTok)
	}
	if fb.authTok != "tok-active" {
		t.Fatalf("revoke must be authorized by the new active token, got auth=%q", fb.authTok)
	}
	fmt.Sprint(fb)
}

// TestLauncherPromoteChildToActive exercises the spec 8.6 failover/promotion transition
// hook (PromoteChildToActive) — the function the launcher's promotion path calls when a
// standby is promoted to active. It must perform the same swap (issue active secret-id,
// log in, revoke old standby token) through the hook, proving the production call site is
// wired and not dead.
func TestLauncherPromoteChildToActive(t *testing.T) {
	fb := &fakeBao{secretID: "sid-active", clientTok: "tok-active"}
	sock := startFakeBao(t, fb)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	newTok, err := PromoteChildToActive(ctx, sock, "role-oracle-active", "tok-old-standby")
	if err != nil {
		t.Fatalf("promote: %v", err)
	}
	if newTok != "tok-active" {
		t.Fatalf("expected new active token, got %q", newTok)
	}

	fb.mu.Lock()
	defer fb.mu.Unlock()
	if fb.loginBody["role_id"] != "role-oracle-active" {
		t.Fatalf("promotion login must use the active role_id, got %q", fb.loginBody["role_id"])
	}
	if fb.loginBody["secret_id"] != "sid-active" {
		t.Fatalf("promotion login must use freshly issued active secret-id, got %q", fb.loginBody["secret_id"])
	}
	if fb.revokedTok != "tok-old-standby" {
		t.Fatalf("promotion must revoke the old standby token, got target=%q", fb.revokedTok)
	}
	if fb.authTok != "tok-active" {
		t.Fatalf("promotion revoke must be authorized by the new active token, got auth=%q", fb.authTok)
	}
}
