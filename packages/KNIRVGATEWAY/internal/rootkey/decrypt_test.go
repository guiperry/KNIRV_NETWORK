package rootkey

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/scrypt"
)

// appendVarint/appendLengthDelimitedField are minimal, test-only protobuf
// wire-format encoders — the mirror image of extractBytesField — used to
// build a fake root.key envelope+content pair without a protobuf runtime,
// the same way the real key_maker/root_key_encryptor tool would (just with
// far fewer fields).
func appendVarint(buf []byte, v uint64) []byte {
	for v >= 0x80 {
		buf = append(buf, byte(v)|0x80)
		v >>= 7
	}
	return append(buf, byte(v))
}

func appendLengthDelimitedField(buf []byte, fieldNum int, value []byte) []byte {
	tag := uint64(fieldNum)<<3 | 2
	buf = appendVarint(buf, tag)
	buf = appendVarint(buf, uint64(len(value)))
	return append(buf, value...)
}

// encryptForTest mirrors backend_server/internal/utils/crypto_utils.go's
// Encrypt exactly (AES-256-GCM, nonce prepended to ciphertext).
func encryptForTest(t *testing.T, plaintext, key []byte) []byte {
	t.Helper()
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatalf("aes.NewCipher: %v", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatalf("cipher.NewGCM: %v", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		t.Fatalf("generate nonce: %v", err)
	}
	return gcm.Seal(nonce, nonce, plaintext, nil)
}

func TestLoadCloudflareCredentials_RoundTrip(t *testing.T) {
	const wantToken = "cf-test-api-token-abc123"
	const wantAccountID = "cf-test-account-id-def456"
	const password = "correct horse battery staple"

	// Build the inner RootKeyFileContentProto-shaped message with a handful
	// of decoy fields interleaved, to prove field-number-based extraction
	// ignores everything it doesn't care about (matches how the real
	// message has ~30 other fields around these two).
	var content []byte
	content = appendLengthDelimitedField(content, 1, []byte("decoy-stripe-secret"))
	content = appendLengthDelimitedField(content, contentFieldCloudflareAPIToken, []byte(wantToken))
	content = appendLengthDelimitedField(content, 20, []byte("decoy-smtp-password"))
	content = appendLengthDelimitedField(content, contentFieldCloudflareAccountID, []byte(wantAccountID))
	content = appendLengthDelimitedField(content, 27, []byte("decoy-tunnel-token"))

	salt := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		t.Fatalf("generate salt: %v", err)
	}
	key, err := scrypt.Key([]byte(password), salt, scryptN, scryptR, scryptP, keyLen)
	if err != nil {
		t.Fatalf("scrypt.Key: %v", err)
	}
	encryptedContent := encryptForTest(t, content, key)

	var envelope []byte
	envelope = appendLengthDelimitedField(envelope, envelopeFieldEncryptedContent, encryptedContent)
	envelope = appendLengthDelimitedField(envelope, envelopeFieldSalt, salt)

	dir := t.TempDir()
	keyPath := filepath.Join(dir, "root.key")
	if err := os.WriteFile(keyPath, envelope, 0600); err != nil {
		t.Fatalf("write fake root.key: %v", err)
	}

	t.Setenv("KNIRV_ROOT_KEY_PATH", keyPath)
	t.Setenv("ORACLE_KEY_PASSWORD", password)

	creds, err := LoadCloudflareCredentials()
	if err != nil {
		t.Fatalf("LoadCloudflareCredentials: %v", err)
	}
	if creds.APIToken != wantToken {
		t.Errorf("APIToken = %q, want %q", creds.APIToken, wantToken)
	}
	if creds.AccountID != wantAccountID {
		t.Errorf("AccountID = %q, want %q", creds.AccountID, wantAccountID)
	}
}

func TestLoadCloudflareCredentials_WrongPassword(t *testing.T) {
	var content []byte
	content = appendLengthDelimitedField(content, contentFieldCloudflareAPIToken, []byte("token"))
	content = appendLengthDelimitedField(content, contentFieldCloudflareAccountID, []byte("account"))

	salt := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		t.Fatalf("generate salt: %v", err)
	}
	key, err := scrypt.Key([]byte("right-password"), salt, scryptN, scryptR, scryptP, keyLen)
	if err != nil {
		t.Fatalf("scrypt.Key: %v", err)
	}
	encryptedContent := encryptForTest(t, content, key)

	var envelope []byte
	envelope = appendLengthDelimitedField(envelope, envelopeFieldEncryptedContent, encryptedContent)
	envelope = appendLengthDelimitedField(envelope, envelopeFieldSalt, salt)

	dir := t.TempDir()
	keyPath := filepath.Join(dir, "root.key")
	if err := os.WriteFile(keyPath, envelope, 0600); err != nil {
		t.Fatalf("write fake root.key: %v", err)
	}

	t.Setenv("KNIRV_ROOT_KEY_PATH", keyPath)
	t.Setenv("ORACLE_KEY_PASSWORD", "wrong-password")

	if _, err := LoadCloudflareCredentials(); err == nil {
		t.Fatal("expected decryption to fail with the wrong password, got nil error")
	}
}

func TestLoadCloudflareCredentials_MissingPassword(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "root.key")
	if err := os.WriteFile(keyPath, []byte("irrelevant"), 0600); err != nil {
		t.Fatalf("write fake root.key: %v", err)
	}

	t.Setenv("KNIRV_ROOT_KEY_PATH", keyPath)
	t.Setenv("ORACLE_KEY_PASSWORD", "")

	if _, err := LoadCloudflareCredentials(); err == nil {
		t.Fatal("expected an error when ORACLE_KEY_PASSWORD is unset, got nil")
	}
}
