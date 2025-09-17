package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"

	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/scrypt"
)

// Scrypt parameters
const (
	ScryptN = 32768 // CPU/memory cost factor (must be power of 2 > 1)
	ScryptR = 8     // Block size parameter
	ScryptP = 1     // Parallelization parameter
	SaltLen = 16    // Length of the salt in bytes
	KeyLen  = 32    // Desired key length in bytes (for AES-256)
)

// DeriveKeyFromPassword uses Scrypt to derive a key from a password and salt.
func DeriveKeyFromPassword(password, salt []byte, n, r, p, keyLen int) ([]byte, error) {
	key, err := scrypt.Key(password, salt, n, r, p, keyLen)
	if err != nil {
		return nil, fmt.Errorf("failed to derive key using scrypt: %w", err)
	}
	return key, nil
}

// GenerateSalt generates a random salt of a specified length.
func GenerateSalt(length int) ([]byte, error) {
	salt := make([]byte, length)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("failed to generate random salt: %w", err)
	}
	return salt, nil
}

// GenerateEncryptionKey generates a secure 32-byte encryption key using SHA-256
func GenerateEncryptionKey() []byte {
	// In production, replace with proper key management from secure storage
	key := sha256.Sum256([]byte("default-encryption-key"))
	return key[:]
}

// Encrypt encrypts data using AES-GCM
func Encrypt(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// Decrypt decrypts data using AES-GCM with the provided key.
func Decrypt(data, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher block: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// DeriveEncryptionKey generates a 32-byte encryption key from the base key string
func DeriveEncryptionKey() []byte {
	// Use a proper key derivation function (PBKDF2) to generate a 32-byte key
	salt := []byte("KNIRVORACLE-salt") // In production, use a secure random salt
	key := pbkdf2.Key([]byte(WALLET_ENCRYPTION_KEY), salt, 4096, 32, sha256.New)
	return key
}
