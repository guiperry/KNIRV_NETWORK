package authority

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
)

// Entropy kinds, the certificate's `entropy` field (AUTHORITY_SPEC.md section 4).
const (
	EntropyCSPRNG     = "csprng"
	EntropyCSPRNGASIC = "csprng+asic"
)

// ErrASICUnavailable is returned by the ASIC stub: the hardware path is Task 7.1.
var ErrASICUnavailable = errors.New("authority: ASIC entropy is not available (Task 7.1)")

// EntropySource supplies the key-derivation input for a session ceremony
// (AUTHORITY_SPEC.md section 4.1, steps 2 and 3).
//
// challenge is fresh CSPRNG output for this ceremony. A hardware source mines it
// and returns its contribution (nonce and device bits) in the bytes; the OS CSPRNG
// is the security floor either way (section 4.1: "the ASIC adds provenance, not a
// stronger entropy source"). kind is the value written to the certificate's
// `entropy` field.
type EntropySource interface {
	Draw(ctx context.Context, challenge [32]byte) (entropy []byte, kind string, err error)
}

// CSPRNG draws 32 bytes from the operating system's random source.
type CSPRNG struct{}

// Draw implements EntropySource. The challenge is not used: there is no hardware
// to bind it to.
func (CSPRNG) Draw(ctx context.Context, _ [32]byte) ([]byte, string, error) {
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return nil, "", fmt.Errorf("read OS random source: %w", err)
	}
	return buf, EntropyCSPRNG, nil
}

// ASIC is the placeholder for the hardware entropy path (Task 7.1). It always
// fails, so a configuration that asks for it falls back to the ceremony's failure
// handling instead of silently using software entropy while claiming csprng+asic.
type ASIC struct{}

// Draw implements EntropySource.
func (ASIC) Draw(context.Context, [32]byte) ([]byte, string, error) {
	return nil, "", ErrASICUnavailable
}
