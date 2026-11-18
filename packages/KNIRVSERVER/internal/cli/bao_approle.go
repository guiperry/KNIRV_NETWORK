package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

// baoAppRoleSession is a short-lived OpenBao client token obtained by AppRole login
// (role_id + secret_id). The enrollment CLI holds it only for one run and revokes it on
// Close, so no standing token file exists. Response bodies are never echoed into errors:
// login and revoke replies carry tokens.
type baoAppRoleSession struct {
	client *http.Client
	token  string
}

// loginBaoAppRole exchanges role_id and secret_id for a client token over the OpenBao
// Unix socket. The secret_id is never logged.
func loginBaoAppRole(ctx context.Context, socket, roleID, secretID string) (*baoAppRoleSession, error) {
	client := newBaoUnixClient(socket)
	body, err := json.Marshal(map[string]string{"role_id": roleID, "secret_id": secretID})
	if err != nil {
		return nil, fmt.Errorf("encode approle login: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://openbao/v1/auth/approle/login", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build approle login: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("approle login: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("approle login: read reply: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("approle login refused: HTTP %d", resp.StatusCode)
	}
	var out struct {
		Auth struct {
			ClientToken string `json:"client_token"`
		} `json:"auth"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("approle login: decode reply")
	}
	if strings.TrimSpace(out.Auth.ClientToken) == "" {
		return nil, fmt.Errorf("approle login: reply carried no client token")
	}
	return &baoAppRoleSession{client: client, token: out.Auth.ClientToken}, nil
}

// Token is the short-lived client token for the secrets provider. It is never printed.
func (s *baoAppRoleSession) Token() string { return s.token }

// Close revokes the session's own token (auth/token/revoke-self), so the credential
// does not outlive the enrollment run.
func (s *baoAppRoleSession) Close(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://openbao/v1/auth/token/revoke-self", nil)
	if err != nil {
		return fmt.Errorf("build token revoke: %w", err)
	}
	req.Header.Set("X-Vault-Token", s.token)
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("revoke approle token: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("revoke approle token: HTTP %d", resp.StatusCode)
	}
	return nil
}

func newBaoUnixClient(socket string) *http.Client {
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socket)
	}}
	return &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// maxSecretIDBytes caps the secret_id file. A larger file is refused, never truncated:
// a silently cut credential would fail later with a misleading login error.
const maxSecretIDBytes = 64 << 10

// secretIDOwnerCheck is the owner check used by readSecretIDFile. It is a package variable
// so tests can exercise the call site without a foreign-owned file (which needs root).
var secretIDOwnerCheck = checkSecretIDOwner

// secretIDOpenFile is the open step used by readSecretIDFile. It is a package variable so a
// test can swap the file between the Lstat check and the open (the os.SameFile branch).
// Production uses openSecretIDFile.
var secretIDOpenFile = openSecretIDFile

// readSecretIDFile reads the single-use AppRole secret_id. The path must name a regular
// file: symlinks, FIFOs, devices and directories are refused before the file is opened,
// so a special file can never block the read. The opened handle must be the same file that
// was checked, and the owner and mode checks run on that handle, so the file checked is the
// file read. The file must be no larger than maxSecretIDBytes. The value is trimmed and
// never printed.
func readSecretIDFile(path string) (string, error) {
	li, err := os.Lstat(path)
	if err != nil {
		return "", fmt.Errorf("stat secret-id file: %w", err)
	}
	if !li.Mode().IsRegular() {
		return "", fmt.Errorf("secret-id file must be a regular file (symlinks and special files are not allowed)")
	}
	f, err := secretIDOpenFile(path)
	if err != nil {
		return "", fmt.Errorf("open secret-id file: %w", err)
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return "", fmt.Errorf("stat secret-id file: %w", err)
	}
	if !fi.Mode().IsRegular() {
		return "", fmt.Errorf("secret-id file must be a regular file (symlinks and special files are not allowed)")
	}
	if !os.SameFile(li, fi) {
		return "", fmt.Errorf("secret-id file changed between checks")
	}
	if err := secretIDOwnerCheck(fi); err != nil {
		return "", err
	}
	if fi.Mode().Perm()&0o077 != 0 {
		return "", fmt.Errorf("secret-id file must be mode 0600 or stricter")
	}
	raw, err := io.ReadAll(io.LimitReader(f, maxSecretIDBytes+1))
	if err != nil {
		return "", fmt.Errorf("read secret-id file: %w", err)
	}
	if len(raw) > maxSecretIDBytes {
		return "", fmt.Errorf("secret-id file is larger than %d bytes", maxSecretIDBytes)
	}
	secretID := strings.TrimSpace(string(raw))
	if secretID == "" {
		return "", fmt.Errorf("secret-id file is empty")
	}
	return secretID, nil
}
