//go:build openbao

package knirvbao

import (
	"context"
	"net/http"
	"os"
	"testing"
)

// TestTwoNodeStandbyToActiveSwap exercises the oracle-standby -> oracle-active failover
// swap (OpenBao.md §4.7, PendingImplementations 2a.6 step 4) across two nodes against a
// DISPOSABLE dev-mode vault. It never starts a vault. It skips unless the owner exports:
//
//	KNIRV_BAO_DEV_SOCKET      unix socket of an already-running dev-mode OpenBao
//	KNIRV_BAO_DEV_ROOT_TOKEN  that dev server's root token (disposable, dev-only)
//
// Run with: go test -tags openbao -run TestTwoNodeStandbyToActiveSwap ./internal/knirvbao/
func TestTwoNodeStandbyToActiveSwap(t *testing.T) {
	socket := os.Getenv("KNIRV_BAO_DEV_SOCKET")
	rootToken := os.Getenv("KNIRV_BAO_DEV_ROOT_TOKEN")
	if socket == "" || rootToken == "" {
		t.Skip("set KNIRV_BAO_DEV_SOCKET and KNIRV_BAO_DEV_ROOT_TOKEN to run against a disposable dev-mode vault")
	}
	ctx := context.Background()
	admin := newBaoAPI(socket)
	mgr := NewAppRoleManagerForSocket(socket)

	// Provision the two roles with the narrow policies from the design.
	policies := map[string]string{
		"oracle-standby": `path "secret/data/node/validator" { capabilities = ["read"] }`,
		"oracle-active":  `path "secret/data/node/payments" { capabilities = ["read"] }`,
	}
	_, _ = admin.request(ctx, http.MethodPost, "sys/auth/approle", rootToken, map[string]string{"type": "approle"}, nil)
	for name, hcl := range policies {
		if _, err := admin.request(ctx, http.MethodPut, "sys/policies/acl/"+name, rootToken, map[string]string{"policy": hcl}, nil); err != nil {
			t.Fatalf("write policy %s: %v", name, err)
		}
		if _, err := admin.request(ctx, http.MethodPost, "auth/approle/role/"+name, rootToken, map[string]any{"token_policies": []string{name}, "token_ttl": "10m"}, nil); err != nil {
			t.Fatalf("create role %s: %v", name, err)
		}
	}
	roleID := func(name string) string {
		var out struct {
			Data struct {
				RoleID string `json:"role_id"`
			} `json:"data"`
		}
		if _, err := admin.request(ctx, http.MethodGet, "auth/approle/role/"+name+"/role-id", rootToken, nil, &out); err != nil {
			t.Fatalf("role-id %s: %v", name, err)
		}
		return out.Data.RoleID
	}
	standbyRole, activeRole := roleID("oracle-standby"), roleID("oracle-active")

	// Node A (bootnode) starts as oracle-standby.
	secret, err := mgr.IssueSecretID(ctx, "oracle-standby")
	if err != nil {
		t.Fatalf("node A standby secret-id: %v", err)
	}
	standbyToken, err := mgr.Login(ctx, standbyRole, secret)
	if err != nil {
		t.Fatalf("node A standby login: %v", err)
	}

	// Node B is an independent standby (a second, unrelated node).
	secretB, err := mgr.IssueSecretID(ctx, "oracle-standby")
	if err != nil {
		t.Fatalf("node B standby secret-id: %v", err)
	}
	nodeBToken, err := mgr.Login(ctx, standbyRole, secretB)
	if err != nil {
		t.Fatalf("node B standby login: %v", err)
	}

	// Negative: a standby token cannot read the active-only payments path.
	if code, _ := admin.request(ctx, http.MethodGet, "secret/data/node/payments", standbyToken, nil, nil); code != http.StatusForbidden {
		t.Fatalf("standby token read payments: HTTP %d, want 403", code)
	}

	// Promotion: node A swaps to oracle-active; the standby token must be revoked.
	activeToken, err := mgr.SwapToActive(ctx, activeRole, standbyToken)
	if err != nil {
		t.Fatalf("swap to active: %v", err)
	}
	if code, _ := admin.request(ctx, http.MethodGet, "auth/token/lookup-self", standbyToken, nil, nil); code == http.StatusOK {
		t.Fatal("old standby token still valid after swap")
	}
	if code, _ := admin.request(ctx, http.MethodGet, "secret/data/node/payments", activeToken, nil, nil); code == http.StatusForbidden {
		t.Fatal("active token denied its own payments path")
	}

	// Node B is unaffected by node A's swap, and still cannot reach the active path.
	if code, _ := admin.request(ctx, http.MethodGet, "auth/token/lookup-self", nodeBToken, nil, nil); code != http.StatusOK {
		t.Fatalf("node B token affected by node A swap: HTTP %d", code)
	}
	if code, _ := admin.request(ctx, http.MethodGet, "secret/data/node/payments", nodeBToken, nil, nil); code != http.StatusForbidden {
		t.Fatalf("node B standby read payments: HTTP %d, want 403", code)
	}
}
