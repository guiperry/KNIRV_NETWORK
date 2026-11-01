package rootkey

import (
	"fmt"
	"os"
	"strings"

	"github.com/guiperry/knirv-sdk-go/secrets"
)

// Scrypt parameters — must exactly match
// KNIRV_CORP/packages/server/backend_server/internal/utils/crypto_utils.go's
// ScryptN/ScryptR/ScryptP/KeyLen, since that's what encrypted this file.
const (
	scryptN = 32768
	scryptR = 8
	scryptP = 1
	keyLen  = 32
)

// Field numbers from KNIRV_CORP/packages/server/backend_server/internal/proto/root_key.proto.
const (
	envelopeFieldEncryptedContent = 1
	envelopeFieldSalt             = 2

	contentFieldCloudflareAPIToken  = 18
	contentFieldCloudflareAccountID = 26
)

// CloudflareCredentials holds the two root.key fields this package cares
// about. Every other field defined in root_key.proto (Stripe keys, SMTP
// password, TLS certs, ...) is intentionally never extracted — this package
// has no business touching them.
type CloudflareCredentials struct {
	AccountID string
	APIToken  string
}

// LoadCloudflareCredentials resolves root.key, decrypts it using
// ORACLE_KEY_PASSWORD (the same env var backend_server reads it with —
// inherited from the parent process that spawns KNIRVGATEWAY as a
// subprocess, see pkg/knirvgateway/manager.go's env := os.Environ()), and
// extracts cloudflare_account_id / cloudflare_api_token.
//
// This does real, non-trivial work (file I/O + scrypt, which is
// deliberately CPU/memory-hard) — callers should call it once at startup
// and cache the result, not per-request.
func LoadCloudflareCredentials() (*CloudflareCredentials, error) {
	path, err := ResolveRootKeyPath()
	if err != nil {
		return nil, fmt.Errorf("resolve root.key path: %w", err)
	}

	password := strings.TrimSpace(os.Getenv("ORACLE_KEY_PASSWORD"))
	if password == "" {
		return nil, fmt.Errorf("ORACLE_KEY_PASSWORD is not set; cannot decrypt root.key at %s", path)
	}

	fields, err := secrets.DecryptFields(path, []byte(password))
	if err != nil {
		return nil, fmt.Errorf("decrypt root.key at %s (wrong ORACLE_KEY_PASSWORD?): %w", path, err)
	}

	creds := &CloudflareCredentials{
		AccountID: fields.String(contentFieldCloudflareAccountID),
		APIToken:  fields.String(contentFieldCloudflareAPIToken),
	}
	if creds.APIToken == "" {
		return nil, fmt.Errorf("root.key at %s has no cloudflare_api_token set", path)
	}
	if creds.AccountID == "" {
		return nil, fmt.Errorf("root.key at %s has no cloudflare_account_id set", path)
	}

	return creds, nil
}
