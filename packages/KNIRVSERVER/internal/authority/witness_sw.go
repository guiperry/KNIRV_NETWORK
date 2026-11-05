package authority

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	sdk "github.com/guiperry/knirv-sdk-go/authority"
)

// ErrNonceExhausted means every 32-bit nonce failed for this header. The caller
// owns the header, so it extends ntime and calls again (section 2.2: the nonce
// space is per header).
var ErrNonceExhausted = errors.New("authority: nonce space exhausted for this header")

// SoftwareWitnesser mines in the Rust core through the SDK, so the target
// arithmetic exists once (repository rule 8) and cannot drift from Verify.
type SoftwareWitnesser struct{}

// Kind implements Witnesser.
func (SoftwareWitnesser) Kind() string { return sdk.WitnessSoftware }

// Mine implements Witnesser. The SDK miner cannot be interrupted once started,
// so a cancelled context abandons the result and returns promptly; the abandoned
// search finishes on its own goroutine. At the T1 and T2 targets that is
// immediate; T3 (about 2^24 hashes) is not enabled by default policy.
func (SoftwareWitnesser) Mine(ctx context.Context, header [80]byte, targetBits uint32) (uint32, [32]byte, error) {
	var zero [32]byte
	if err := ctx.Err(); err != nil {
		return 0, zero, err
	}
	version := header[2]
	tier := sdk.Tier(header[3])
	if !tier.Valid() || tier == sdk.TierT0 {
		return 0, zero, fmt.Errorf("authority: header tier %d cannot carry a witness", header[3])
	}
	var prev, merkle [32]byte
	copy(prev[:], header[4:36])
	copy(merkle[:], header[36:68])
	ntime := binary.BigEndian.Uint32(header[68:72])
	nbits := binary.BigEndian.Uint32(header[72:76])
	if nbits != targetBits {
		return 0, zero, fmt.Errorf("authority: header nbits %#x differs from target %#x", nbits, targetBits)
	}
	start := binary.BigEndian.Uint32(header[76:80])

	type result struct {
		header sdk.Header
		nonce  uint32
		err    error
	}
	done := make(chan result, 1)
	go func() {
		h, nonce, err := sdk.Mine(prev, merkle, ntime, nbits, start, version, tier)
		done <- result{h, nonce, err}
	}()
	select {
	case <-ctx.Done():
		return 0, zero, ctx.Err()
	case r := <-done:
		if r.err != nil {
			if strings.Contains(r.err.Error(), "exhausted") {
				return 0, zero, ErrNonceExhausted
			}
			return 0, zero, r.err
		}
		hash := r.header.Witness()
		// Authoritative re-check with the same function Verify uses.
		if !sdk.MeetsTarget(hash, nbits) {
			return 0, zero, errors.New("authority: mined header does not meet its target")
		}
		return r.nonce, hash, nil
	}
}
