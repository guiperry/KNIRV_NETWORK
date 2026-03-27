package crypto

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"backend_server/internal/oracle/types"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

// KeyPair represents an ECDSA key pair with associated address
type KeyPair struct {
	PrivateKey *ecdsa.PrivateKey
	PublicKey  *ecdsa.PublicKey
	Address    types.Address
}

// GenerateKeyPair generates a new ECDSA key pair
func GenerateKeyPair() (*KeyPair, error) {
	privateKey, err := ecdsa.GenerateKey(ethcrypto.S256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	address := PublicKeyToAddress(&privateKey.PublicKey)

	return &KeyPair{
		PrivateKey: privateKey,
		PublicKey:  &privateKey.PublicKey,
		Address:    address,
	}, nil
}

// PrivateKeyFromHex creates a KeyPair from a hex-encoded private key
func PrivateKeyFromHex(privateKeyHex string) (*KeyPair, error) {
	privateKey, err := ethcrypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	address := PublicKeyToAddress(&privateKey.PublicKey)

	return &KeyPair{
		PrivateKey: privateKey,
		PublicKey:  &privateKey.PublicKey,
		Address:    address,
	}, nil
}

// PrivateKeyHex returns the private key as a hex string
func (kp *KeyPair) PrivateKeyHex() string {
	return hex.EncodeToString(ethcrypto.FromECDSA(kp.PrivateKey))
}

// PublicKeyHex returns the public key as a hex string
func (kp *KeyPair) PublicKeyHex() string {
	return hex.EncodeToString(ethcrypto.FromECDSAPub(kp.PublicKey))
}

// Sign signs a message using the private key
// The message is hashed with Keccak256 before signing
func (kp *KeyPair) Sign(message []byte) ([]byte, error) {
	hash := ethcrypto.Keccak256(message)
	signature, err := ethcrypto.Sign(hash, kp.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign message: %w", err)
	}
	return signature, nil
}

// SignHash signs a pre-hashed message using the private key
func (kp *KeyPair) SignHash(hash []byte) ([]byte, error) {
	signature, err := ethcrypto.Sign(hash, kp.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign hash: %w", err)
	}
	return signature, nil
}

// SignMessage signs a message and returns the signature as a hex string
func SignMessage(privateKeyHex string, message []byte) (string, error) {
	kp, err := PrivateKeyFromHex(privateKeyHex)
	if err != nil {
		return "", err
	}

	signature, err := kp.Sign(message)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(signature), nil
}

// VerifySignature verifies a signature against a message and public key
func VerifySignature(publicKeyBytes []byte, message []byte, signature []byte) bool {
	hash := ethcrypto.Keccak256(message)
	// Remove recovery ID (last byte) if present
	if len(signature) == 65 {
		signature = signature[:64]
	}
	return ethcrypto.VerifySignature(publicKeyBytes, hash, signature)
}

// RecoverAddress recovers the address from a signature
func RecoverAddress(message []byte, signature []byte) (types.Address, error) {
	hash := ethcrypto.Keccak256(message)
	publicKey, err := ethcrypto.SigToPub(hash, signature)
	if err != nil {
		return types.Address{}, fmt.Errorf("failed to recover public key: %w", err)
	}

	return PublicKeyToAddress(publicKey), nil
}

// PublicKeyToAddress converts an ECDSA public key to an Ethereum-style address
func PublicKeyToAddress(publicKey *ecdsa.PublicKey) types.Address {
	publicKeyBytes := ethcrypto.FromECDSAPub(publicKey)
	hash := ethcrypto.Keccak256(publicKeyBytes[1:]) // Skip the 0x04 prefix
	var addr types.Address
	copy(addr[:], hash[12:]) // Take last 20 bytes
	return addr
}

// AddressFromPrivateKey derives the address from a private key hex string
func AddressFromPrivateKey(privateKeyHex string) (types.Address, error) {
	kp, err := PrivateKeyFromHex(privateKeyHex)
	if err != nil {
		return types.Address{}, err
	}
	return kp.Address, nil
}
