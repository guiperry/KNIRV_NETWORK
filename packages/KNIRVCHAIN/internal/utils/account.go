package utils

import (
	"crypto/sha256"
	"encoding/binary"
)

// maxSafeAccountNumber is 2^53-1, the largest integer a JSON client (a JavaScript
// number) represents exactly.
const maxSafeAccountNumber = (1 << 53) - 1

// AccountNumberFor is the account number a signer must use for an address. It is derived
// from the address, so no account registry is needed and every node computes the same value.
func AccountNumberFor(address string) uint64 {
	sum := sha256.Sum256([]byte(address))
	return binary.BigEndian.Uint64(sum[:8]) & maxSafeAccountNumber
}
