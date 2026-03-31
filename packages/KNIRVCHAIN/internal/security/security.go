package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

type NodeEncryption struct {
	iterations int
	keyLength  int
}

func NewNodeEncryption() *NodeEncryption {
	return &NodeEncryption{
		iterations: 100000,
		keyLength:  32,
	}
}

func (m *NodeEncryption) DeriveKey(userSecret string, salt []byte) []byte {
	return pbkdf2.Key(
		[]byte(userSecret),
		salt,
		m.iterations,
		m.keyLength,
		sha256.New,
	)
}

func (m *NodeEncryption) EncryptNode(data []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

func (m *NodeEncryption) DecryptNode(encrypted []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(encrypted) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := encrypted[:nonceSize], encrypted[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

func (m *NodeEncryption) GenerateSalt() ([]byte, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}
	return salt, nil
}

func (m *NodeEncryption) EncodeKey(key []byte) string {
	return base64.URLEncoding.EncodeToString(key)
}

func (m *NodeEncryption) DecodeKey(encoded string) ([]byte, error) {
	return base64.URLEncoding.DecodeString(encoded)
}

func (m *NodeEncryption) ComputeNodeHash(data []byte) string {
	h := sha256.New()
	h.Write(data)
	return base64.URLEncoding.EncodeToString(h.Sum(nil))
}

type EncryptedNode struct {
	Ciphertext []byte
	Salt       []byte
	Version    int
}

func (m *NodeEncryption) EncryptNodeWithSalt(data []byte, key []byte) (*EncryptedNode, error) {
	ciphertext, err := m.EncryptNode(data, key)
	if err != nil {
		return nil, err
	}

	salt, err := m.GenerateSalt()
	if err != nil {
		return nil, err
	}

	return &EncryptedNode{
		Ciphertext: ciphertext,
		Salt:       salt,
		Version:    1,
	}, nil
}

func (m *NodeEncryption) DecryptNodeWithSalt(encrypted *EncryptedNode, key []byte) ([]byte, error) {
	return m.DecryptNode(encrypted.Ciphertext, key)
}
