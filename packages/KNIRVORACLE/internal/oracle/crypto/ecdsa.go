package crypto

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	knirvsigning "github.com/KNIRV/KNIRV_NETWORK/KNIRVSDK/go/signing"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
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
	// Strip 0x prefix if present
	if strings.HasPrefix(privateKeyHex, "0x") {
		privateKeyHex = privateKeyHex[2:]
	}

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

// PublicKeyToAddress derives the Cosmos-compatible KNIRV account identifier.
func PublicKeyToAddress(publicKey *ecdsa.PublicKey) types.Address {
	compressed := ethcrypto.CompressPubkey(publicKey)
	encoded, err := knirvsigning.Address(compressed, knirvsigning.DefaultAddressPrefix)
	if err != nil {
		panic(err)
	}
	hash, err := knirvsigning.DecodeAddress(encoded, knirvsigning.DefaultAddressPrefix)
	if err != nil {
		panic(err)
	}
	var addr types.Address
	copy(addr[:], hash)
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
