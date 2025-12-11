package types

import (
	"encoding/hex"
	"fmt"
	"strings"
)

// Address represents an Ethereum-style 20-byte address
type Address [20]byte

// String returns the hex-encoded address with 0x prefix
func (a Address) String() string {
	return "0x" + hex.EncodeToString(a[:])
}

// Bytes returns the address as a byte slice
func (a Address) Bytes() []byte {
	return a[:]
}

// IsZero returns true if the address is the zero address
func (a Address) IsZero() bool {
	for _, b := range a {
		if b != 0 {
			return false
		}
	}
	return true
}

// AddressFromString creates an Address from a hex string (with or without 0x prefix)
func AddressFromString(s string) (Address, error) {
	// Remove 0x prefix if present
	s = strings.TrimPrefix(s, "0x")

	// Decode hex string
	bytes, err := hex.DecodeString(s)
	if err != nil {
		return Address{}, fmt.Errorf("invalid hex string: %w", err)
	}

	if len(bytes) != 20 {
		return Address{}, fmt.Errorf("invalid address length: expected 20 bytes, got %d", len(bytes))
	}

	var addr Address
	copy(addr[:], bytes)
	return addr, nil
}

// AddressFromBytes creates an Address from a byte slice
func AddressFromBytes(b []byte) (Address, error) {
	if len(b) != 20 {
		return Address{}, fmt.Errorf("invalid address length: expected 20 bytes, got %d", len(b))
	}

	var addr Address
	copy(addr[:], b)
	return addr, nil
}

// ZeroAddress returns the zero address (0x0000...0000)
func ZeroAddress() Address {
	return Address{}
}
