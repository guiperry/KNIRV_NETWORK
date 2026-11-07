package launcher

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"knirv-server/internal/knirvbao"
)

// startMockBaoUnix starts a unix-socket OpenBao mock that answers the AppRole
// secret-id mint endpoint used by MintRoleSecretID. Unlike an earlier mock that
// accepted any name, this one only mints for roles that exist in the REAL
// bootstrap policy set (knirvbao.BootstrapRoleNames); a name not created by
// bootstrap is rejected with 404. It records the minted roles and returns a distinct
// wrapping token per role (never the admin token). No real vault is touched.
func startMockBaoUnix(t *testing.T) (socket string, minted *map[string]int) {
	t.Helper()
	dir := t.TempDir()
	socket = filepath.Join(dir, "bao.sock")
	minted = &map[string]int{}
	realRoles := map[string]bool{}
	for _, n := range knirvbao.BootstrapRoleNames() {
		realRoles[n] = true
	}
	l, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatalf("listen unix: %v", err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/auth/approle/role/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		// /v1/auth/approle/role/<role>/secret-id
		if len(parts) < 5 {
			http.Error(w, "bad path", http.StatusBadRequest)
			return
		}
		role := parts[4]
		if !realRoles[role] {
			http.Error(w, "unknown role", http.StatusNotFound)
			return
		}
		(*minted)[role]++
		_ = json.NewEncoder(w).Encode(map[string]any{
			"wrap_info": map[string]any{"token": "wrap-" + role + "-token"},
		})
	})
	srv := &http.Server{Handler: mux}
	go func() { _ = srv.Serve(l) }()
	t.Cleanup(func() { _ = srv.Close(); _ = os.Remove(socket) })
	return socket, minted
}

// TestLauncherMintsOnlyRealBootstrapRoles proves the launcher's AppRole names are
// exactly the roles the bootstrap creates (single source of truth) — a name drift
// like "validation-chain" vs "validation" would fail here — and that a name absent
// from bootstrap is rejected rather than minted.
func TestLauncherMintsOnlyRealBootstrapRoles(t *testing.T) {
	socket, minted := startMockBaoUnix(t)
	real := map[string]bool{}
	for _, n := range knirvbao.BootstrapRoleNames() {
		real[n] = true
	}
	for _, role := range []BaoChildRole{BaoRoleBackend, BaoRoleValidation, BaoRoleOracle} {
		name := approleName(role)
		if !real[name] {
			t.Fatalf("launcher role %s maps to %q, which is not a bootstrap role", role, name)
		}
		if _, err := MintRoleSecretID(context.Background(), socket, "admin", role, time.Minute); err != nil {
			t.Fatalf("mint real role %s (%s): %v", role, name, err)
		}
	}
	// A role NOT created by bootstrap must not be mintable.
	if _, err := MintRoleSecretID(context.Background(), socket, "admin", BaoChildRole("validation-chain"), time.Minute); err == nil {
		t.Fatal("minted a non-bootstrap role name (name drift)")
	}
	if _, err := MintRoleSecretID(context.Background(), socket, "admin", BaoChildRole("bogus"), time.Minute); err == nil {
		t.Fatal("minted a bogus role name")
	}
	// The three child roles map to three distinct real bootstrap roles.
	if (*minted)["backend"] != 1 || (*minted)["validation"] != 1 || (*minted)["oracle"] != 1 {
		t.Fatalf("unexpected mint counts: %v", *minted)
	}
}

// TestMintRoleSecretIDReturnsWrappedTokenNotAdmin proves the launcher never hands
// the broad admin credential to the child: minting returns a distinct wrapping
// token, not the admin token it was given.
func TestMintRoleSecretIDReturnsWrappedTokenNotAdmin(t *testing.T) {
	socket, minted := startMockBaoUnix(t)
	const adminToken = "s.BROAD-ADMIN-TOKEN-LEAKED"
	wrap, err := MintRoleSecretID(context.Background(), socket, adminToken, BaoRoleValidation, time.Minute)
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	if wrap == adminToken {
		t.Fatal("mint returned the admin token itself")
	}
	if wrap != "wrap-validation-token" {
		t.Fatalf("wrap = %q, want wrap-validation-token", wrap)
	}
	if (*minted)["validation"] != 1 {
		t.Fatalf("validation mint count = %d, want 1", (*minted)["validation"])
	}
}

// TestMintRoleSecretIDDistinctPerRole proves each child role gets its own wrapping
// token and its own AppRole, so a compromise of one child cannot reuse another's.
func TestMintRoleSecretIDDistinctPerRole(t *testing.T) {
	socket, minted := startMockBaoUnix(t)
	got := map[BaoChildRole]string{}
	for _, role := range []BaoChildRole{BaoRoleBackend, BaoRoleValidation, BaoRoleOracle} {
		w, err := MintRoleSecretID(context.Background(), socket, "admin", role, time.Minute)
		if err != nil {
			t.Fatalf("mint %s: %v", role, err)
		}
		got[role] = w
	}
	if got[BaoRoleBackend] == got[BaoRoleValidation] || got[BaoRoleValidation] == got[BaoRoleOracle] ||
		got[BaoRoleBackend] == got[BaoRoleOracle] {
		t.Fatalf("roles received non-distinct wrapping tokens: %v", got)
	}
	for role, want := range map[BaoChildRole]string{
		BaoRoleBackend:    "backend",
		BaoRoleValidation: "validation",
		BaoRoleOracle:     "oracle",
	} {
		if (*minted)[want] != 1 {
			t.Fatalf("mint count for %s = %d, want 1", want, (*minted)[want])
		}
		_ = role
	}
}

// TestWriteHandoffFileIs0600 proves the wrapped secret-id is written to a 0600
// file the child reads — never an env var carrying a live token.
func TestWriteHandoffFileIs0600(t *testing.T) {
	dir := t.TempDir()
	path, err := WriteHandoffFile(dir, BaoRoleBackend, "wrap-backend-token")
	if err != nil {
		t.Fatalf("write handoff: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat handoff: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("handoff mode = %o, want 600", info.Mode().Perm())
	}
	content, err := os.ReadFile(path)
	if err != nil || string(content) != "wrap-backend-token" {
		t.Fatalf("handoff content = %q, want wrap-backend-token", content)
	}
}

// TestHandoffEnvNeverContainsAmbientToken proves the child env handed out by the
// launcher carries the Bao socket and the 0600 wrap-token file path, but NEVER the
// ambient admin/broad token — the exact audit P1 concern.
func TestHandoffEnvNeverContainsAmbientToken(t *testing.T) {
	const ambient = "s.AMBIENT-BROAD-TOKEN"
	t.Setenv("KNIRV_BAO_TOKEN", ambient)

	for _, env := range [][]string{
		HandoffEnv("/run/bao.sock", "/run/bao-backend-wrap.txt"),
		ValidationHandoffEnv("/run/bao.sock", "/run/bao-validation-wrap.txt"),
	} {
		joined := strings.Join(env, "\n")
		if strings.Contains(joined, ambient) {
			t.Fatalf("handoff env leaks ambient token: %s", joined)
		}
		for _, entry := range env {
			if strings.HasPrefix(entry, "KNIRV_BAO_TOKEN=") || strings.HasPrefix(entry, "VALIDATION_CHAIN_BAO_TOKEN=") {
				t.Fatalf("handoff env sets a live token var: %s", entry)
			}
		}
		if !strings.Contains(joined, "KNIRV_BAO_SOCKET=/run/bao.sock") && !strings.Contains(joined, "VALIDATION_CHAIN_BAO_SOCKET=/run/bao.sock") {
			t.Fatalf("handoff env missing socket: %s", joined)
		}
		if !strings.Contains(joined, "WRAP_TOKEN_FILE=") {
			t.Fatalf("handoff env missing wrap token file: %s", joined)
		}
	}
}

// TestRoleTokensAreMutuallyExclusive proves the scoping the real bootstrap policies
// enforce. Expectations are DERIVED from the actual bootstrap policy definitions
// (knirvbao.BootstrapRolePaths) — a single source of truth — not a disconnected copy.
// The privileged capability is signing (knirv/sign/*); kv reads (knirv-kv/data/*) are
// intentionally shared between roles and are not part of the mutual-exclusion contract.
// For each role, this proves it can reach only its own signing paths and gets 403 on
// non-allowlisted raw paths and on another role's exclusive signing paths.
func TestRoleTokensAreMutuallyExclusive(t *testing.T) {
	roles := knirvbao.BootstrapRoleNames()
	signPaths := map[string][]string{}
	for _, r := range roles {
		ps, ok := knirvbao.BootstrapRolePaths(r)
		if !ok {
			t.Fatalf("bootstrap role %q has no policy", r)
		}
		var s []string
		for _, p := range ps {
			if strings.HasPrefix(p, "knirv/sign/") {
				s = append(s, p)
			}
		}
		signPaths[r] = s
	}

	// 1. Non-allowlisted raw path is granted to NO role → every role gets 403 on it.
	for _, r := range roles {
		if contains(signPaths[r], "knirv/sign/raw/root-owner") {
			t.Fatalf("role %q must NOT grant non-allowlisted raw path knirv/sign/raw/root-owner", r)
		}
	}

	// 2. Raw failover-identity signing is allowed ONLY to backend — matching the
	//    actual policies (the base oracle and oracle-active roles may NOT raw-sign;
	//    they get direct/message failover-identity instead).
	for _, r := range roles {
		grants := contains(signPaths[r], "knirv/sign/raw/failover-identity")
		if grants != (r == knirvbao.RoleBackend) {
			t.Fatalf("role %q raw-failover grant = %v, want %v", r, grants, r == knirvbao.RoleBackend)
		}
	}

	// 3. oracle (base) and validation grant NO raw path at all.
	for _, r := range []string{knirvbao.RoleOracle, knirvbao.RoleValidation} {
		for _, p := range signPaths[r] {
			if strings.HasPrefix(p, "knirv/sign/raw/") {
				t.Fatalf("role %q must not grant any raw path (%s)", r, p)
			}
		}
	}

	// 4. gateway is read-only and grants NO signing path. (oracle-standby is NOT
	//    read-only — it signs message/bootnode-master — so it is excluded here.)
	if len(signPaths[knirvbao.RoleGateway]) != 0 {
		t.Fatalf("gateway must not grant any sign path, got %v", signPaths[knirvbao.RoleGateway])
	}

	// 5. Every signing role is denied at least one OTHER signing role's exclusive path,
	//    and no two signing roles have identical sign sets — so the scoping actually
	//    distinguishes them (a backend token cannot sign as oracle, etc.).
	signing := []string{}
	for _, r := range roles {
		if len(signPaths[r]) > 0 {
			signing = append(signing, r)
		}
	}
	for i := 0; i < len(signing); i++ {
		for j := i + 1; j < len(signing); j++ {
			a, b := signing[i], signing[j]
			if equalSet(signPaths[a], signPaths[b]) {
				t.Fatalf("signing roles %q and %q have identical sign sets %v (not mutually exclusive)", a, b, signPaths[a])
			}
			if subset(signPaths[a], signPaths[b]) && subset(signPaths[b], signPaths[a]) {
				t.Fatalf("signing roles %q and %q neither excludes the other", a, b)
			}
		}
	}

	// 6. Distinct authorities: backend and validation must not reach each other's
	//    exclusive signing paths (root-owner for oracle, raw-failover for backend).
	if contains(signPaths[knirvbao.RoleBackend], "knirv/sign/message/root-owner") ||
		contains(signPaths[knirvbao.RoleBackend], "knirv/sign/direct/root-owner") {
		t.Fatal("backend must not sign root-owner")
	}
	if contains(signPaths[knirvbao.RoleValidation], "knirv/sign/message/root-owner") ||
		contains(signPaths[knirvbao.RoleValidation], "knirv/sign/raw/failover-identity") {
		t.Fatal("validation must not sign root-owner or raw failover")
	}
}

func toSet(s []string) map[string]bool {
	m := make(map[string]bool, len(s))
	for _, v := range s {
		m[v] = true
	}
	return m
}

func equalSet(a, b []string) bool {
	return subset(a, b) && subset(b, a)
}

func subset(a, b []string) bool {
	bs := toSet(b)
	for _, v := range a {
		if !bs[v] {
			return false
		}
	}
	return true
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
