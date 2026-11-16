package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const baoAuditSample = `{"time":"2026-11-12T00:00:01Z","type":"request","auth":{"client_token":"hmac-SECRET","accessor":"ACC"},"request":{"operation":"update","path":"knirv/sign/message/root-owner","mount_type":"knirv","client_address":"10.0.0.1","data":{"payload":"SECRET"}}}
{"time":"2026-11-12T00:00:02Z","type":"response","request":{"operation":"read","path":"knirv-kv/data/node/cloudflare","mount_type":"kv"},"error":"permission denied"}
not-json-line
{"time":"2026-11-12T00:00:03Z","type":"request","request":{"operation":"update","path":"knirv/sign/raw/failover-identity","mount_type":"knirv"}}
`

func writeAuditFixture(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "audit.log")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func getBaoAudit(t *testing.T, s *Server, query string) BaoAuditResponse {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/monitor/bao-audit"+query, nil)
	rec := httptest.NewRecorder()
	s.handleBaoAudit(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var out BaoAuditResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return out
}

// TestBaoAuditProjectionRedacts proves tokens, accessors, client addresses and request
// bodies never appear in the response, while operation/path/time/error do.
func TestBaoAuditProjectionRedacts(t *testing.T) {
	s := NewServer(&ServerConfig{BaoAuditLogPath: writeAuditFixture(t, baoAuditSample)})
	rec := httptest.NewRecorder()
	s.handleBaoAudit(rec, httptest.NewRequest(http.MethodGet, "/api/v1/monitor/bao-audit", nil))
	body := rec.Body.String()
	for _, secret := range []string{"hmac-SECRET", "ACC", "10.0.0.1", "payload"} {
		if strings.Contains(body, secret) {
			t.Fatalf("response leaked %q: %s", secret, body)
		}
	}
	out := getBaoAudit(t, s, "")
	if !out.Enabled || out.Scanned != 3 {
		t.Fatalf("enabled=%v scanned=%d, want true/3 (malformed line skipped)", out.Enabled, out.Scanned)
	}
	if out.OperationCnt["update"] != 2 || out.OperationCnt["read"] != 1 {
		t.Fatalf("operation counts = %v", out.OperationCnt)
	}
	// Newest first.
	if out.Entries[0].Path != "knirv/sign/raw/failover-identity" {
		t.Fatalf("newest entry = %+v", out.Entries[0])
	}
	if out.Entries[1].Error != "permission denied" {
		t.Fatalf("error text not surfaced: %+v", out.Entries[1])
	}
}

// TestBaoAuditLimitAndValidation covers the limit bound and bad input.
func TestBaoAuditLimitAndValidation(t *testing.T) {
	s := NewServer(&ServerConfig{BaoAuditLogPath: writeAuditFixture(t, baoAuditSample)})
	if got := len(getBaoAudit(t, s, "?limit=1").Entries); got != 1 {
		t.Fatalf("limit=1 returned %d entries", got)
	}
	for _, q := range []string{"?limit=0", "?limit=abc", "?limit=-3"} {
		rec := httptest.NewRecorder()
		s.handleBaoAudit(rec, httptest.NewRequest(http.MethodGet, "/api/v1/monitor/bao-audit"+q, nil))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("%s: status %d, want 400", q, rec.Code)
		}
	}
	rec := httptest.NewRecorder()
	s.handleBaoAudit(rec, httptest.NewRequest(http.MethodPost, "/api/v1/monitor/bao-audit", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST status %d, want 405", rec.Code)
	}
}

// TestBaoAuditDisabledAndMissing: no path, or no log yet, reports disabled, not an error.
func TestBaoAuditDisabledAndMissing(t *testing.T) {
	if out := getBaoAudit(t, NewServer(&ServerConfig{}), ""); out.Enabled {
		t.Fatal("empty path must report disabled")
	}
	missing := filepath.Join(t.TempDir(), "absent.log")
	if out := getBaoAudit(t, NewServer(&ServerConfig{BaoAuditLogPath: missing}), ""); out.Enabled {
		t.Fatal("missing log must report disabled")
	}
}

// TestBaoAuditRefusesSymlink: a symlinked audit log is refused, never followed.
func TestBaoAuditRefusesSymlink(t *testing.T) {
	target := writeAuditFixture(t, baoAuditSample)
	link := filepath.Join(t.TempDir(), "audit.log")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	s := NewServer(&ServerConfig{BaoAuditLogPath: link})
	rec := httptest.NewRecorder()
	s.handleBaoAudit(rec, httptest.NewRequest(http.MethodGet, "/api/v1/monitor/bao-audit", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("symlink audit log: status %d, want 500 refusal", rec.Code)
	}
}
