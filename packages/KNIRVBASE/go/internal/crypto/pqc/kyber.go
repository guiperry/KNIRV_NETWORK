package pqc

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"

	"github.com/cloudflare/circl/kem"
	"github.com/cloudflare/circl/kem/mlkem/mlkem768"
)

// KyberKeyPair represents an ML-KEM-768 key pair
type KyberKeyPair struct {
	PublicKey  kem.PublicKey
	PrivateKey kem.PrivateKey
	Scheme     kem.Scheme
}

// GenerateKyberKeyPair generates a new ML-KEM-768 key pair
func GenerateKyberKeyPair() (*KyberKeyPair, error) {
	scheme := mlkem768.Scheme()
	publicKey, privateKey, err := scheme.GenerateKeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to generate Kyber key pair: %w", err)
	}

	return &KyberKeyPair{
		PublicKey:  publicKey,
		PrivateKey: privateKey,
		Scheme:     scheme,
	}, nil
}

// KyberEncrypt encrypts plaintext using ML-KEM-768 (FIPS 203) KEM and AES-256-GCM
func KyberEncrypt(publicKey kem.PublicKey, plaintext []byte) ([]byte, error) {
	scheme := mlkem768.Scheme()

	// Generate shared secret using Kyber KEM
	ciphertext, sharedSecret, err := scheme.Encapsulate(publicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encapsulate: %w", err)
	}

	// Use shared secret to encrypt plaintext with AES-256-GCM
	encryptedData, err := aesEncrypt(sharedSecret, plaintext)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt data: %w", err)
	}

	// Combine Kyber ciphertext and AES-encrypted data
	result := make([]byte, scheme.CiphertextSize()+len(encryptedData))
	copy(result[:scheme.CiphertextSize()], ciphertext)
	copy(result[scheme.CiphertextSize():], encryptedData)

	return result, nil
}

// KyberDecrypt decrypts ciphertext using ML-KEM-768 (FIPS 203) KEM and AES-256-GCM
func KyberDecrypt(privateKey kem.PrivateKey, ciphertext []byte) ([]byte, error) {
	scheme := mlkem768.Scheme()

	if len(ciphertext) < scheme.CiphertextSize() {
		return nil, errors.New("ciphertext too short")
	}

	// Extract Kyber ciphertext and encrypted data
	kyberCiphertext := ciphertext[:scheme.CiphertextSize()]
	encryptedData := ciphertext[scheme.CiphertextSize():]

	// Decapsulate shared secret using Kyber KEM
	sharedSecret, err := scheme.Decapsulate(privateKey, kyberCiphertext)
	if err != nil {
		return nil, fmt.Errorf("failed to decapsulate: %w", err)
	}

	// Use shared secret to decrypt data with AES-256-GCM
	plaintext, err := aesDecrypt(sharedSecret, encryptedData)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data: %w", err)
	}

	return plaintext, nil
}

// MarshalPublicKey converts a public key to bytes
func (kp *KyberKeyPair) MarshalPublicKey() ([]byte, error) {
	return kp.PublicKey.MarshalBinary()
}

// MarshalPrivateKey converts a private key to bytes
func (kp *KyberKeyPair) MarshalPrivateKey() ([]byte, error) {
	return kp.PrivateKey.MarshalBinary()
}

// UnmarshalPublicKey converts bytes to a public key
func UnmarshalKyberPublicKey(data []byte) (kem.PublicKey, error) {
	scheme := mlkem768.Scheme()
	return scheme.UnmarshalBinaryPublicKey(data)
}

// UnmarshalPrivateKey converts bytes to a private key
func UnmarshalKyberPrivateKey(data []byte) (kem.PrivateKey, error) {
	scheme := mlkem768.Scheme()
	return scheme.UnmarshalBinaryPrivateKey(data)
}

// aesEncrypt encrypts data using AES-256-GCM with the provided key
func aesEncrypt(key []byte, plaintext []byte) ([]byte, error) {
	// The ML-KEM-768 shared secret is 32 uniform bytes and is used directly
	// as the AES-256 key. The TypeScript implementation does the same.
	if len(key) != 32 {
		return nil, fmt.Errorf("aes-256 key must be 32 bytes, got %d", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// aesDecrypt decrypts data using AES-256-GCM with the provided key
func aesDecrypt(key []byte, ciphertext []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("aes-256 key must be 32 bytes, got %d", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce := ciphertext[:nonceSize]
	ciphertext = ciphertext[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}
