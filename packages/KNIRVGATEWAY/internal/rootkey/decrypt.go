package rootkey

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/scrypt"
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

// aesGCMDecrypt mirrors backend_server/internal/utils/crypto_utils.go's
// Decrypt exactly: AES-256-GCM, nonce prepended to the ciphertext (as
// produced by gcm.Seal(nonce, nonce, plaintext, nil) on the encrypt side).
func aesGCMDecrypt(data, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher block: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}

	return plaintext, nil
}

// parseEnvelope extracts encrypted_content (field 1) and salt (field 2) from
// the outer EncryptedRootKeyFile protobuf message.
func parseEnvelope(data []byte) (encryptedContent, salt []byte, err error) {
	encryptedContent, ok, err := extractBytesField(data, envelopeFieldEncryptedContent)
	if err != nil {
		return nil, nil, fmt.Errorf("parse envelope: %w", err)
	}
	if !ok {
		return nil, nil, fmt.Errorf("root.key missing encrypted_content field")
	}

	salt, ok, err = extractBytesField(data, envelopeFieldSalt)
	if err != nil {
		return nil, nil, fmt.Errorf("parse envelope: %w", err)
	}
	if !ok {
		return nil, nil, fmt.Errorf("root.key missing salt field")
	}

	return encryptedContent, salt, nil
}

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

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read root.key at %s: %w", path, err)
	}

	encryptedContent, salt, err := parseEnvelope(data)
	if err != nil {
		return nil, fmt.Errorf("root.key at %s: %w", path, err)
	}

	password := strings.TrimSpace(os.Getenv("ORACLE_KEY_PASSWORD"))
	if password == "" {
		return nil, fmt.Errorf("ORACLE_KEY_PASSWORD is not set; cannot decrypt root.key at %s", path)
	}

	key, err := scrypt.Key([]byte(password), salt, scryptN, scryptR, scryptP, keyLen)
	if err != nil {
		return nil, fmt.Errorf("derive decryption key: %w", err)
	}

	plaintext, err := aesGCMDecrypt(encryptedContent, key)
	if err != nil {
		return nil, fmt.Errorf("decrypt root.key at %s (wrong ORACLE_KEY_PASSWORD?): %w", path, err)
	}

	tokenBytes, _, err := extractBytesField(plaintext, contentFieldCloudflareAPIToken)
	if err != nil {
		return nil, fmt.Errorf("parse decrypted root.key content: %w", err)
	}
	accountBytes, _, err := extractBytesField(plaintext, contentFieldCloudflareAccountID)
	if err != nil {
		return nil, fmt.Errorf("parse decrypted root.key content: %w", err)
	}

	creds := &CloudflareCredentials{
		AccountID: string(accountBytes),
		APIToken:  string(tokenBytes),
	}
	if creds.APIToken == "" {
		return nil, fmt.Errorf("root.key at %s has no cloudflare_api_token set", path)
	}
	if creds.AccountID == "" {
		return nil, fmt.Errorf("root.key at %s has no cloudflare_account_id set", path)
	}

	return creds, nil
}
