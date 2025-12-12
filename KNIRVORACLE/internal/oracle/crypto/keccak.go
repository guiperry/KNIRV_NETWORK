package crypto

import (
	"encoding/hex"

	"github.com/ethereum/go-ethereum/crypto"
)

// Keccak256 computes the Keccak256 hash of the input data
func Keccak256(data []byte) []byte {
	return crypto.Keccak256(data)
}

// Keccak256Hash computes the Keccak256 hash and returns it as a hex string
func Keccak256Hash(data []byte) string {
	return hex.EncodeToString(crypto.Keccak256(data))
}

// Keccak256HashWithPrefix computes the Keccak256 hash and returns it with 0x prefix
func Keccak256HashWithPrefix(data []byte) string {
	return "0x" + hex.EncodeToString(crypto.Keccak256(data))
}
