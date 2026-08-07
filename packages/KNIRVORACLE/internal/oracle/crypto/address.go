package crypto

import (
	"fmt"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
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

// ChecksumAddress returns the canonical external representation. KNIRV does
// not use Ethereum EIP-55 addresses; its checksum is part of Bech32.
func ChecksumAddress(addr types.Address) string {
	return addr.String()
}
