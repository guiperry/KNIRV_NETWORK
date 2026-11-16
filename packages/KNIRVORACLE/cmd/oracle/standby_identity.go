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
	secretID, err := readStandbySecretID(strings.TrimSpace(os.Getenv(envStandbySecretIDFile)))
	if err != nil {
		return nil, err
	}
	token, err := standbyAppRoleLogin(ctx, socket, roleID, secretID)
	if err != nil {
		return nil, err
	}
	return secrets.NewBaoProvider(socket, token), nil
}

// standbyOpenFile is a seam for the open step so tests can simulate a swap between the
// Lstat check and the open (the os.SameFile branch). Production uses os.Open.
var standbyOpenFile = os.Open

// readStandbySecretID reads the single-use secret ID from a file that must be a
// regular file accessible only by its owner (mode 0600 or stricter). Symlinks are
// rejected: the check uses Lstat, and after open the opened file must be the same
// file that was checked. The value is never logged or included in errors.
//
// The file is not deleted here. Whether the launcher or the standby removes it after
// login is an owner decision and is not implemented.
func readStandbySecretID(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("standby: %s is not set", envStandbySecretIDFile)
	}
	info, err := os.Lstat(path)
	if err != nil {
		return "", fmt.Errorf("standby: stat secret-id file: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 {
		return "", errors.New("standby: secret-id file must be a regular file accessible only by its owner (mode 0600 or stricter), symlinks not allowed")
	}
	if err := checkSecretIDOwner(info); err != nil {
		return "", err
	}
	f, err := standbyOpenFile(path)
	if err != nil {
		return "", fmt.Errorf("standby: open secret-id file: %w", err)
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil {
		return "", fmt.Errorf("standby: stat opened secret-id file: %w", err)
	}
	if !os.SameFile(info, opened) {
		return "", errors.New("standby: secret-id file changed between checks")
	}
	raw, err := io.ReadAll(f)
	if err != nil {
		return "", fmt.Errorf("standby: read secret-id file: %w", err)
	}
	secretID := strings.TrimSpace(string(raw))
	if secretID == "" {
		return "", errors.New("standby: secret-id file is empty")
	}
	return secretID, nil
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
