package core

import (
	"crypto/sha256"
	"encoding/binary"
)

const DefaultKNIRVDifficultyBits uint8 = 12

// CanonicalSHA256 provides the canonical Double SHA-256 implementation
// This is the reference implementation used across all hashing methods
type CanonicalSHA256 struct {
	difficultyBits uint8
}

// NewCanonicalSHA256 creates a new canonical SHA-256 instance
func NewCanonicalSHA256() *CanonicalSHA256 {
	return NewCanonicalSHA256WithDifficulty(DefaultKNIRVDifficultyBits)
}

// NewCanonicalSHA256WithDifficulty constructs a KNIRV PoW verifier. The
// target is a KNIRV protocol parameter, not Bitcoin's difficulty-one target.
func NewCanonicalSHA256WithDifficulty(bits uint8) *CanonicalSHA256 {
	return &CanonicalSHA256{difficultyBits: bits}
}

func (c *CanonicalSHA256) DifficultyBits() uint8 { return c.difficultyBits }

// ComputeSHA256 computes a single SHA-256 hash
func (c *CanonicalSHA256) ComputeSHA256(data []byte) [32]byte {
	return sha256.Sum256(data)
}

// ComputeDoubleSHA256 computes SHA256(SHA256(data)) - Bitcoin's hash function
// This is the canonical implementation that all methods should use
func (c *CanonicalSHA256) ComputeDoubleSHA256(data []byte) [32]byte {
	first := sha256.Sum256(data)
	return sha256.Sum256(first[:])
}

// ComputeDoubleSHA256WithNonce computes Double SHA-256 for an 80-byte Bitcoin header
// with the nonce field replaced with the specified nonce value
func (c *CanonicalSHA256) ComputeDoubleSHA256WithNonce(header []byte, nonce uint32) ([32]byte, error) {
	if len(header) != 80 {
		return [32]byte{}, &HashError{
			Type:    ErrorInvalidInput,
			Message: "header must be exactly 80 bytes",
			Context: map[string]interface{}{
				"header_length": len(header),
				"nonce":         nonce,
			},
		}
	}

	// Create a copy to avoid modifying the original
	workHeader := make([]byte, 80)
	copy(workHeader, header)

	// Set nonce in header (bytes 76-79, little-endian)
	workHeader[76] = byte(nonce)
	workHeader[77] = byte(nonce >> 8)
	workHeader[78] = byte(nonce >> 16)
	workHeader[79] = byte(nonce >> 24)

	return c.ComputeDoubleSHA256(workHeader), nil
}

// IsValidProofOfWork checks the KNIRV-owned leading-zero-bit target.
func (c *CanonicalSHA256) IsValidProofOfWork(hash [32]byte) bool {
	fullBytes := int(c.difficultyBits / 8)
	for i := 0; i < fullBytes; i++ {
		if hash[i] != 0 {
			return false
		}
	}
	remainingBits := c.difficultyBits % 8
	return remainingBits == 0 || hash[fullBytes]>>(8-remainingBits) == 0
}

// ComputeProofOfWork binds arbitrary assertion bytes and an explicit nonce.
// Payload shape and nonce placement are KNIRV-owned, not inherited from a
// Bitcoin header.
func (c *CanonicalSHA256) ComputeProofOfWork(payload []byte, nonce uint32) [32]byte {
	h := sha256.New()
	h.Write([]byte("KNIRV-POW-V1\x00"))
	h.Write(payload)
	var encodedNonce [4]byte
	binary.BigEndian.PutUint32(encodedNonce[:], nonce)
	h.Write(encodedNonce[:])
	return sha256.Sum256(h.Sum(nil))
}

// MineForNonce searches a caller-controlled nonce interval against the KNIRV
// target. It returns an error rather than a non-witness when no nonce matches.
func (c *CanonicalSHA256) MineForNonce(payload []byte, nonceStart, nonceEnd uint32) (uint32, error) {
	if len(payload) == 0 {
		return 0, &HashError{Type: ErrorInvalidInput, Message: "proof payload must not be empty"}
	}
	if nonceEnd < nonceStart {
		return 0, &HashError{Type: ErrorInvalidInput, Message: "nonce end precedes nonce start"}
	}
	for nonce := nonceStart; ; nonce++ {
		if c.IsValidProofOfWork(c.ComputeProofOfWork(payload, nonce)) {
			return nonce, nil
		}
		if nonce == nonceEnd {
			break
		}
	}
	return 0, &HashError{Type: ErrorOperationFailed, Message: "no valid nonce in search interval"}
}

// ExtractNonce extracts nonce from an 80-byte Bitcoin header
func (c *CanonicalSHA256) ExtractNonce(header []byte) (uint32, error) {
	if len(header) < 80 {
		return 0, &HashError{
			Type:    ErrorInvalidInput,
			Message: "header must be at least 80 bytes",
			Context: map[string]interface{}{
				"header_length": len(header),
			},
		}
	}

	return binary.LittleEndian.Uint32(header[76:80]), nil
}

// HashError represents errors that can occur during hashing operations
type HashError struct {
	Type    ErrorType
	Message string
	Context map[string]interface{}
}

func (e *HashError) Error() string {
	return e.Message
}

// ErrorType represents different types of hashing errors
type ErrorType int

const (
	ErrorInvalidInput ErrorType = iota
	ErrorHardwareUnavailable
	ErrorOperationFailed
	ErrorTimeout
	ErrorResourceBusy
)
