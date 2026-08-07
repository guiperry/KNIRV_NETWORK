package types

import (
	"encoding/json"
	"fmt"

	knirvsigning "github.com/KNIRV/KNIRV_NETWORK/KNIRVSDK/go/signing"
)

// Address is the canonical 20-byte KNIRV account identifier, rendered as
// knirv-prefixed Bech32 on every external interface.
type Address [20]byte

func (a Address) String() string {
	value, err := knirvsigning.EncodeAddress(a[:], knirvsigning.DefaultAddressPrefix)
	if err != nil {
		panic(err)
	}
	return value
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

func AddressFromString(s string) (Address, error) {
	bytes, err := knirvsigning.DecodeAddress(s, knirvsigning.DefaultAddressPrefix)
	if err != nil {
		return Address{}, fmt.Errorf("invalid KNIRV address: %w", err)
	}

	if len(bytes) != 20 {
		return Address{}, fmt.Errorf("invalid address length: expected 20 bytes, got %d", len(bytes))
	}

	var addr Address
	copy(addr[:], bytes)
	return addr, nil
}

func (a Address) MarshalJSON() ([]byte, error) { return json.Marshal(a.String()) }

func (a *Address) UnmarshalJSON(data []byte) error {
	var encoded string
	if err := json.Unmarshal(data, &encoded); err != nil {
		return err
	}
	parsed, err := AddressFromString(encoded)
	if err != nil {
		return err
	}
	*a = parsed
	return nil
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
