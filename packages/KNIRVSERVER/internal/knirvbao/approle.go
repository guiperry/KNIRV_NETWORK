package knirvbao

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// vaultRequester is the subset of the OpenBao client used for AppRole secret-id rotation
// and token revocation. It is satisfied by *baoAPI in production and by a fake client in
// tests (DEC-2 / 2a.6 P1-2).
type vaultRequester interface {
	request(ctx context.Context, method, path, token string, body any, out any) (int, error)
}

// AppRoleManager performs the launcher-driven AppRole failover swap: issuing a fresh
// (single-use, short-TTL) secret-id, logging in to obtain a new token, and revoking the
// previously-held token so an old/standby credential cannot be reused after promotion.
type AppRoleManager struct {
	client vaultRequester
}

// NewAppRoleManager builds a manager over the given client (e.g. *baoAPI).
func NewAppRoleManager(client vaultRequester) *AppRoleManager {
	return &AppRoleManager{client: client}
}

// NewAppRoleManagerForSocket builds an AppRoleManager over a unix-socket OpenBao client
// (production call site used by the launcher failover swap).
func NewAppRoleManagerForSocket(socket string) *AppRoleManager {
	return NewAppRoleManager(newBaoAPI(socket))
}

type issueSecretIDResponse struct {
	Data struct {
		SecretID         string `json:"secret_id"`
		SecretIDAccessor string `json:"secret_id_accessor"`
	} `json:"data"`
}

type loginResponse struct {
	Auth struct {
		ClientToken string `json:"client_token"`
	} `json:"auth"`
}

// IssueSecretID issues a new secret-id for role.
func (m *AppRoleManager) IssueSecretID(ctx context.Context, role string) (string, error) {
	var resp issueSecretIDResponse
	code, err := m.client.request(ctx, http.MethodPut, "auth/approle/role/"+role+"/secret-id", "", nil, &resp)
	if err != nil {
		return "", err
	}
	if code < 200 || code >= 300 {
		return "", fmt.Errorf("issue secret-id for %s: HTTP %d", role, code)
	}
	if resp.Data.SecretID == "" {
		return "", fmt.Errorf("issue secret-id for %s: empty secret id", role)
	}
	return resp.Data.SecretID, nil
}

// Login exchanges role_id + secret_id for a client token.
func (m *AppRoleManager) Login(ctx context.Context, roleID, secretID string) (string, error) {
	var resp loginResponse
	body := map[string]string{"role_id": roleID, "secret_id": secretID}
	code, err := m.client.request(ctx, http.MethodPut, "auth/approle/login", "", body, &resp)
	if err != nil {
		return "", err
	}
	if code < 200 || code >= 300 {
		return "", fmt.Errorf("approle login: HTTP %d", code)
	}
	if resp.Auth.ClientToken == "" {
		return "", fmt.Errorf("approle login: empty token")
	}
	return resp.Auth.ClientToken, nil
}

// RevokeToken revokes targetToken using authToken's privileges (the new active token is
// used to revoke the old one).
func (m *AppRoleManager) RevokeToken(ctx context.Context, authToken, targetToken string) error {
	body := map[string]string{"token": targetToken}
	code, err := m.client.request(ctx, http.MethodPost, "auth/token/revoke", authToken, body, nil)
	if err != nil {
		return err
	}
	if code < 200 || code >= 300 {
		return fmt.Errorf("revoke token: HTTP %d", code)
	}
	return nil
}

// SwapToActive performs the failover swap: issue a fresh secret-id on the active role,
// login to obtain the new active token, and revoke the previously-held token. Returns the
// new active token. The old token is revoked to prevent reuse after demotion/promotion.
func (m *AppRoleManager) SwapToActive(ctx context.Context, roleID, oldToken string) (string, error) {
	secretID, err := m.IssueSecretID(ctx, "oracle-active")
	if err != nil {
		return "", err
	}
	newToken, err := m.Login(ctx, roleID, secretID)
	if err != nil {
		return "", err
	}
	if oldToken != "" {
		if err := m.RevokeToken(ctx, newToken, oldToken); err != nil {
			return "", err
		}
	}
	return newToken, nil
}

// MintWrappedSecretID uses the admin (bootstrap) credential ONLY to mint a single-use,
// response-wrapped AppRole secret-id for role. It returns the wrapping token, never the
// secret-id and never the admin token. This is the single OpenBao call path for child
// handoffs; the launcher wraps it (launcher.MintRoleSecretID) and does not speak to the
// socket itself.
func MintWrappedSecretID(ctx context.Context, socket, adminToken, role string, wrapTTL time.Duration) (string, error) {
	var out struct {
		WrapInfo struct {
			Token string `json:"token"`
		} `json:"wrap_info"`
	}
	headers := map[string]string{"X-Vault-Wrap-TTL": wrapTTL.String()}
	code, err := newBaoAPI(socket).requestWithHeaders(ctx, http.MethodPost, "auth/approle/role/"+role+"/secret-id", adminToken, headers, nil, &out)
	if err != nil {
		return "", fmt.Errorf("mint wrapped secret-id for %s: %w", role, err)
	}
	if code < 200 || code >= 300 {
		return "", fmt.Errorf("mint wrapped secret-id for %s: HTTP %d", role, code)
	}
	if out.WrapInfo.Token == "" {
		return "", fmt.Errorf("mint wrapped secret-id for %s: empty wrap token", role)
	}
	return out.WrapInfo.Token, nil
}
