package crypto

import (
	"crypto/ecdsa"
	"fmt"

	"backend_server/internal/oracle/types"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

// ValidateAddress checks if an address string is valid
func ValidateAddress(addr string) error {
	_, err := types.AddressFromString(addr)
	if err != nil {
		return fmt.Errorf("%w: %v", types.ErrInvalidAddress, err)
	}
	return nil
}

// IsValidAddress returns true if the address string is valid
func IsValidAddress(addr string) bool {
	return ValidateAddress(addr) == nil
}

// GenerateAddress generates a new random address
func GenerateAddress() (types.Address, error) {
	kp, err := GenerateKeyPair()
	if err != nil {
		return types.Address{}, err
	}
	return kp.Address, nil
}

// PublicKeyBytesToAddress converts public key bytes to an address
func PublicKeyBytesToAddress(publicKeyBytes []byte) (types.Address, error) {
	publicKey, err := ethcrypto.UnmarshalPubkey(publicKeyBytes)
	if err != nil {
		return types.Address{}, fmt.Errorf("failed to unmarshal public key: %w", err)
	}
	return PublicKeyToAddress(publicKey), nil
}

// ChecksumAddress returns the checksummed version of an address (EIP-55)
func ChecksumAddress(addr types.Address) string {
	// This uses the Ethereum checksum format
	return ethcrypto.PubkeyToAddress(*new(ecdsa.PublicKey)).Hex() // Placeholder - implement proper EIP-55
}
