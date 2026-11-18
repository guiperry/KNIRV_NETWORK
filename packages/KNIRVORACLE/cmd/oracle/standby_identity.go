package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/guiperry/knirv-sdk-go/secrets"
	knirvsigning "github.com/guiperry/knirv-sdk-go/signing"
	"github.com/knirvcorp/knirvoracle/internal/oracle"
	oraclecrypto "github.com/knirvcorp/knirvoracle/internal/oracle/crypto"
	"go.uber.org/zap"
)

// Standby identity (Task 8.8, plan 2a.6). A standby Oracle resolves its replication
// identity through secrets.Provider under the AppRole oracle-standby. The AppRole
// role ID is not secret and comes from the environment. The secret ID is read from
// a 0600 file that the launcher writes; the OpenBao side makes it single-use.
//
// This file must not read boot.key or root.key, not even as a fallback. The standby
// is a failover ledger, not an authority. TestStandbyPathHoldsNoBootKeyMaterial
// enforces this by scanning the standby path for boot-key access.
const (
	envStandbyBaoSocket    = "VALIDATION_CHAIN_BAO_SOCKET"
	envStandbyRoleID       = "ORACLE_STANDBY_BAO_ROLE_ID"
	envStandbySecretIDFile = "ORACLE_STANDBY_BAO_SECRET_ID_FILE"
)

// newStandbyProvider returns the OpenBao-backed provider for the standby, logged in
// as oracle-standby. It never falls back to the file backend, whose identities are
// boot.key and root.key.
func newStandbyProvider(ctx context.Context) (secrets.Provider, error) {
	socket := strings.TrimSpace(os.Getenv(envStandbyBaoSocket))
	if socket == "" {
		return nil, fmt.Errorf("standby: %s is not set", envStandbyBaoSocket)
	}
	roleID := strings.TrimSpace(os.Getenv(envStandbyRoleID))
	if roleID == "" {
		return nil, fmt.Errorf("standby: %s is not set", envStandbyRoleID)
	}
	secretIDPath := strings.TrimSpace(os.Getenv(envStandbySecretIDFile))
	secretID, readInfo, err := readStandbySecretIDFile(secretIDPath)
	if err != nil {
		return nil, err
	}
	token, err := standbyAppRoleLogin(ctx, socket, roleID, secretID)
	if err != nil {
		return nil, err
	}
	// Q47: the secret_id is single-use. Once login has consumed it, delete the file so no
	// reusable copy stays on disk. Same policy as the enroll path: if the delete fails,
	// the token is revoked and the standby does not start.
	if err := removeConsumedSecretIDFile(secretIDPath, readInfo); err != nil {
		revokeCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if rerr := standbyRevokeToken(revokeCtx, socket, token); rerr != nil {
			return nil, fmt.Errorf("standby: remove secret-id file after login: %w (revoke failed: %v)", err, rerr)
		}
		return nil, fmt.Errorf("standby: remove secret-id file after login: %w", err)
	}
	return secrets.NewBaoProvider(socket, token), nil
}

// maxStandbySecretIDBytes caps the secret-id file, the same 64 KiB rule as the KNIRVSERVER
// enroll reader. A larger file is refused, never truncated.
const maxStandbySecretIDBytes = 64 << 10

// standbyOpenFile is a seam for the open step so tests can simulate a swap between the
// Lstat check and the open (the os.SameFile branch). Production uses
// openStandbySecretIDFile, which never blocks on a FIFO or device (see the unix and
// non-unix files).
var standbyOpenFile = openStandbySecretIDFile

// readStandbySecretID reads the single-use secret ID from a file that must be a
// regular file accessible only by its owner (mode 0600 or stricter). Symlinks are
// rejected: the check uses Lstat, and after open the opened file must be the same
// file that was checked. The value is never logged or included in errors.
//
// The file is deleted by newStandbyProvider after a successful AppRole login (Q47),
// not here: this function only reads. A refused login leaves the file in place, as on
// the enroll path.
func readStandbySecretID(path string) (string, error) {
	secretID, _, err := readStandbySecretIDFile(path)
	return secretID, err
}

// readStandbySecretIDFile is readStandbySecretID that also returns the FileInfo of the
// inode it read (the opened handle). newStandbyProvider passes that info to
// removeConsumedSecretIDFile so the delete targets the file that was read.
//
// Owner and mode are checked on the opened handle, after SameFile, the same order as the
// enroll reader (internal/cli/bao_approle.go). The Lstat check only confirms a regular file.
func readStandbySecretIDFile(path string) (string, os.FileInfo, error) {
	if path == "" {
		return "", nil, fmt.Errorf("standby: %s is not set", envStandbySecretIDFile)
	}
	notRegular := errors.New("standby: secret-id file must be a regular file (symlinks and special files are not allowed)")
	info, err := os.Lstat(path)
	if err != nil {
		return "", nil, fmt.Errorf("standby: stat secret-id file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", nil, notRegular
	}
	f, err := standbyOpenFile(path)
	if err != nil {
		return "", nil, fmt.Errorf("standby: open secret-id file: %w", err)
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil {
		return "", nil, fmt.Errorf("standby: stat opened secret-id file: %w", err)
	}
	if !opened.Mode().IsRegular() {
		return "", nil, notRegular
	}
	if !os.SameFile(info, opened) {
		return "", nil, errors.New("standby: secret-id file changed between checks")
	}
	if err := checkSecretIDOwner(opened); err != nil {
		return "", nil, err
	}
	if opened.Mode().Perm()&0o077 != 0 {
		return "", nil, errors.New("standby: secret-id file must be mode 0600 or stricter")
	}
	raw, err := io.ReadAll(io.LimitReader(f, maxStandbySecretIDBytes+1))
	if err != nil {
		return "", nil, fmt.Errorf("standby: read secret-id file: %w", err)
	}
	if len(raw) > maxStandbySecretIDBytes {
		return "", nil, fmt.Errorf("standby: secret-id file is larger than %d bytes", maxStandbySecretIDBytes)
	}
	secretID := strings.TrimSpace(string(raw))
	if secretID == "" {
		return "", nil, errors.New("standby: secret-id file is empty")
	}
	return secretID, opened, nil
}

// removeConsumedSecretIDFile deletes the secret-id file after login, but only if the path
// still names the inode that was read. If the path was replaced after the read, nothing is
// removed and an error is returned, so the file that was not read is not deleted.
//
// Residual window: this is a path-based remove, so a swap between the Lstat here and the
// os.Remove is not closed. Closing it fully needs unlinkat on a directory handle.
func removeConsumedSecretIDFile(path string, readInfo os.FileInfo) error {
	cur, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if readInfo == nil || !os.SameFile(readInfo, cur) {
		return errors.New("secret-id file changed after it was read; not removed")
	}
	return os.Remove(path)
}

// standbyAppRoleLogin exchanges role_id and secret_id for a client token over the
// OpenBao unix socket. Only the transport is local; the token is then used by the SDK
// BaoProvider. Error text carries the HTTP status only, never the request body.
func standbyAppRoleLogin(ctx context.Context, socket, roleID, secretID string) (string, error) {
	client := &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, "unix", socket)
			},
		},
	}
	body, err := json.Marshal(map[string]string{"role_id": roleID, "secret_id": secretID})
	if err != nil {
		return "", fmt.Errorf("standby: encode approle login: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, "http://openbao/v1/auth/approle/login", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("standby: build approle login: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("standby: approle login transport: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("standby: approle login: HTTP %d", resp.StatusCode)
	}
	var out struct {
		Auth struct {
			ClientToken string `json:"client_token"`
		} `json:"auth"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("standby: decode approle login: %w", err)
	}
	if out.Auth.ClientToken == "" {
		return "", errors.New("standby: approle login returned an empty token")
	}
	return out.Auth.ClientToken, nil
}

// standbyRevokeToken revokes the token obtained by standbyAppRoleLogin (auth/token/
// revoke-self). It is used when the run must abort after a login, so the credential does
// not outlive the failed start. Error text carries the HTTP status only.
func standbyRevokeToken(ctx context.Context, socket, token string) error {
	client := &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, "unix", socket)
			},
		},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://openbao/v1/auth/token/revoke-self", nil)
	if err != nil {
		return fmt.Errorf("standby: build token revoke: %w", err)
	}
	req.Header.Set("X-Vault-Token", token)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("standby: revoke token transport: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("standby: revoke token: HTTP %d", resp.StatusCode)
	}
	return nil
}

// standbyReplicationSignerName is the provider identity name for replication pulls.
// The SDK maps "boot" to the OpenBao identity bootnode-master (KNIRVSDK core
// secrets.rs). The oracle-standby policy allows signing only for bootnode-master.
// "validation" maps to checkpoint-signer, which that policy denies.
const standbyReplicationSignerName = "boot"

// resolveStandbyValidatorSigner returns the standby's replication identity from the
// provider. Under OpenBao this resolves through the plugin under the oracle-standby
// policy, which allows replication-pull signing only.
func resolveStandbyValidatorSigner(ctx context.Context, provider secrets.Provider) (knirvsigning.RemoteSigner, error) {
	if provider == nil {
		return nil, errors.New("standby: no secrets provider")
	}
	signer, err := provider.Signer(ctx, standbyReplicationSignerName)
	if err != nil {
		return nil, fmt.Errorf("standby: validator identity: %w", err)
	}
	if signer == nil {
		return nil, errors.New("standby: validator identity is nil")
	}
	return signer, nil
}

// applyStandbyIdentity fills the standby config's identity fields.
//
//   - OwnerPrivateKey: a fresh in-memory key, not persisted and never used to sign.
//     The NRN token constructor needs an owner handle, and boot.key is not an option.
//   - ReplicationSigner: the provider-issued standby identity. Without it, pulls are
//     disabled by clearing ReplicationSources. The oracle must not fall back to the
//     ephemeral owner key for pulls.
//
// A missing or failing identity degrades the standby (no pulls) and logs a WARN. It
// does not stop the standby from starting.
func applyStandbyIdentity(ctx context.Context, cfg *oracle.OracleConfig, provider secrets.Provider, logger *zap.Logger) error {
	owner, err := oraclecrypto.GenerateKeyPair()
	if err != nil {
		return fmt.Errorf("standby: create in-memory owner handle: %w", err)
	}
	cfg.OwnerPrivateKey = owner.PrivateKeyHex()
	cfg.ReplicationSigner = nil

	signer, err := resolveStandbyValidatorSigner(ctx, provider)
	if err != nil {
		logger.Warn("standby replication pulls disabled: no standby identity", zap.Error(err))
		cfg.ReplicationSources = nil
		return nil
	}
	cfg.ReplicationSigner = signer
	logger.Info("Standby Oracle identity resolved from secrets provider (oracle-standby)")
	return nil
}
