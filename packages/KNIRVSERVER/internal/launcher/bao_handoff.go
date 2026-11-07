package launcher

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"knirv-server/internal/knirvbao"
)

// BaoChildRole identifies a subprocess that needs a scoped OpenBao token. Each
// role gets its own AppRole and its own short-lived token; none of them ever
// receives the broad bootstrap/admin token the launcher holds.
type BaoChildRole string

const (
	BaoRoleBackend    BaoChildRole = "backend"
	BaoRoleValidation BaoChildRole = "validation"
	BaoRoleOracle     BaoChildRole = "oracle"
)

// approleName maps a child role to its OpenBao AppRole. The names are the canonical
// identities defined once in knirvbao (RoleBackend, RoleValidation, RoleOracle); the
// launcher only ever mints a wrapped secret ID for a real bootstrap role, so the role
// a child authenticates for can never drift from what bootstrap created.
func approleName(role BaoChildRole) string {
	switch role {
	case BaoRoleBackend:
		return knirvbao.RoleBackend
	case BaoRoleValidation:
		return knirvbao.RoleValidation
	case BaoRoleOracle:
		return knirvbao.RoleOracle
	}
	return string(role)
}

// MintRoleSecretID uses the admin (bootstrap) credential ONLY to mint a
// single-use wrapped AppRole secret ID for child `role`. The wrapping token — not
// the secret ID and never the admin token — is what gets handed to the child. The
// child unwraps it, logs in for its own role, and receives only its short-lived
// role token. This keeps the broad admin credential inside the launcher/manager
// and never propagates it to a subprocess.
func MintRoleSecretID(ctx context.Context, socket, adminToken string, role BaoChildRole, wrapTTL time.Duration) (string, error) {
	url := fmt.Sprintf("http://localhost/v1/auth/approle/role/%s/secret-id", approleName(role))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("X-Vault-Token", adminToken)
	req.Header.Set("X-Vault-Wrap-TTL", wrapTTL.String())

	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, "unix", socket)
			},
		},
		Timeout: 15 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("bao: mint secret-id for %s: %w", role, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("bao: mint secret-id for %s: http %d", role, resp.StatusCode)
	}
	var out struct {
		WrapInfo struct {
			Token string `json:"token"`
		} `json:"wrap_info"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("bao: decode secret-id wrap for %s: %w", role, err)
	}
	if out.WrapInfo.Token == "" {
		return "", fmt.Errorf("bao: empty wrap token for %s", role)
	}
	return out.WrapInfo.Token, nil
}

// SwapChildToActive performs the launcher-driven AppRole failover swap for a child that
// currently holds a (standby) token and must be promoted to the active role: it issues a
// fresh secret-id on the active role, logs in for a new active token, and revokes the
// previously-held token so a demoted standby credential cannot be reused. The broad admin
// token is never handed to the child — only the role_id and the child's current token
// drive the swap. This is the real failover transition call site (DEC-2 / 2a.6 P1-2).
func SwapChildToActive(ctx context.Context, socket, roleID, oldToken string) (string, error) {
	mgr := knirvbao.NewAppRoleManagerForSocket(socket)
	return mgr.SwapToActive(ctx, roleID, oldToken)
}

// PromoteChildToActive is the spec 8.6 failover/promotion transition hook. When a standby
// child is promoted to active — a failover CONFIRMED names this node as the current root,
// or a RECLAIMED demotes the prior active — the launcher's promotion path calls this to
// swap the child's AppRole token from its standby role to the active role and revoke the
// old token. It is the single reviewed entry point for that credential swap (it delegates
// to SwapChildToActive), so the failover monitor never calls the low-level swap directly.
func PromoteChildToActive(ctx context.Context, socket, roleID, oldToken string) (string, error) {
	return SwapChildToActive(ctx, socket, roleID, oldToken)
}

// WriteHandoffFile writes payload to a 0600 file under handoffDir named for the
// role. The child reads this file — never an env var carrying a live token — to
// obtain its wrapped secret-id token. Returned path is absolute.
func WriteHandoffFile(handoffDir string, role BaoChildRole, payload string) (string, error) {
	if err := os.MkdirAll(handoffDir, 0o700); err != nil {
		return "", fmt.Errorf("bao handoff dir: %w", err)
	}
	path := filepath.Join(handoffDir, fmt.Sprintf("bao-%s-wrap.txt", role))
	if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
		return "", fmt.Errorf("bao handoff write: %w", err)
	}
	return path, nil
}

// HandoffEnv returns the env entries to pass to a child for Bao access WITHOUT
// ever including the ambient admin/broad token. The child receives the OpenBao
// socket and the path to its 0600 wrapped-secret-id handoff file; it authenticates
// for its own role and obtains only its short-lived role token.
func HandoffEnv(socket, handoffPath string) []string {
	return []string{
		fmt.Sprintf("KNIRV_BAO_SOCKET=%s", socket),
		fmt.Sprintf("KNIRV_BAO_WRAP_TOKEN_FILE=%s", handoffPath),
	}
}

// ValidationHandoffEnv is HandoffEnv specialized for the Validation Chain child,
// which reads the VALIDATION_CHAIN_BAO_* contract.
func ValidationHandoffEnv(socket, handoffPath string) []string {
	return []string{
		fmt.Sprintf("VALIDATION_CHAIN_BAO_SOCKET=%s", socket),
		fmt.Sprintf("VALIDATION_CHAIN_BAO_WRAP_TOKEN_FILE=%s", handoffPath),
	}
}
