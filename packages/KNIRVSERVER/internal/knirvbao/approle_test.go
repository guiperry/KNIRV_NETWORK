package knirvbao

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"testing"
)

// fakeVault is an in-process OpenBao stand-in that records requests and issues deterministic
// secret-ids / tokens, so the AppRole rotation + revocation logic can be tested without a
// live vault.
type fakeVault struct {
	mu       sync.Mutex
	requests []fakeReq
	revoked  []string
}

type fakeReq struct {
	method, path, token string
	roleID, secretID    string
}

func (f *fakeVault) request(ctx context.Context, method, path, token string, body any, out any) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	roleID, secretID := "", ""
	if m, ok := body.(map[string]string); ok {
		roleID = m["role_id"]
		secretID = m["secret_id"]
		if tk, ok := m["token"]; ok && path == "auth/token/revoke" {
			f.revoked = append(f.revoked, tk)
		}
	}
	f.requests = append(f.requests, fakeReq{method, path, token, roleID, secretID})

	switch {
	case strings.HasSuffix(path, "/secret-id") && method == http.MethodPut:
		role := strings.TrimSuffix(strings.TrimPrefix(path, "auth/approle/role/"), "/secret-id")
		if r, ok := out.(*issueSecretIDResponse); ok {
			r.Data.SecretID = "sid-" + role
			r.Data.SecretIDAccessor = "acc-" + role
		}
		return 200, nil
	case path == "auth/approle/login" && method == http.MethodPut:
		tok := "tok-" + roleID
		if r, ok := out.(*loginResponse); ok {
			r.Auth.ClientToken = tok
		}
		return 200, nil
	case path == "auth/token/revoke" && method == http.MethodPost:
		return 200, nil
	}
	return 404, nil
}

// TestAppRoleSwapRevokesOldToken is the P1-2 acceptance test: after a standby->active
// swap, the previously-held token must be revoked so it cannot be reused.
func TestAppRoleSwapRevokesOldToken(t *testing.T) {
	f := &fakeVault{}
	m := NewAppRoleManager(f)
	old := "tok-old-standby"
	newTok, err := m.SwapToActive(context.Background(), "oracle-active-roleid", old)
	if err != nil {
		t.Fatalf("swap: %v", err)
	}
	if newTok != "tok-oracle-active-roleid" {
		t.Fatalf("unexpected new token: %s", newTok)
	}
	if len(f.revoked) != 1 || f.revoked[0] != old {
		t.Fatalf("expected old token %q revoked, got %v", old, f.revoked)
	}
	// The revoke must be authorized by the NEW token, not the old one.
	for _, r := range f.requests {
		if r.path == "auth/token/revoke" && r.token == old {
			t.Fatal("revoke must not be authorized by the old token")
		}
	}
}

// TestAppRoleSwapWithoutOldToken covers the cold-start case (no prior token to revoke).
func TestAppRoleSwapWithoutOldToken(t *testing.T) {
	f := &fakeVault{}
	m := NewAppRoleManager(f)
	newTok, err := m.SwapToActive(context.Background(), "oracle-active-roleid", "")
	if err != nil {
		t.Fatal(err)
	}
	if newTok == "" {
		t.Fatal("expected new token")
	}
	if len(f.revoked) != 0 {
		t.Fatalf("no revocation expected, got %v", f.revoked)
	}
}

// TestAppRoleIssueAndRevoke exercises the atomic pieces directly.
func TestAppRoleIssueAndRevoke(t *testing.T) {
	f := &fakeVault{}
	m := NewAppRoleManager(f)
	sid, err := m.IssueSecretID(context.Background(), "oracle-standby")
	if err != nil || sid != "sid-oracle-standby" {
		t.Fatalf("issue: %v %q", err, sid)
	}
	tok, err := m.Login(context.Background(), "rid", sid)
	if err != nil || tok != "tok-rid" {
		t.Fatalf("login: %v %q", err, tok)
	}
	if err := m.RevokeToken(context.Background(), tok, "tok-to-kill"); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if len(f.revoked) != 1 || f.revoked[0] != "tok-to-kill" {
		t.Fatalf("revoke not recorded: %v", f.revoked)
	}
}
