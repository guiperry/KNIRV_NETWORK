package nrv

import (
	"github.com/cloudflare/circl/sign/mldsa/mldsa65"
)

// MLDSA65 is the transitional Go binding for ML-DSA-65 (FIPS 204) verification,
// backed by circl. The KNIRVSDK Rust core is the crypto source of truth and does
// not yet expose ML-DSA-65 (it has ed25519 and secp256k1 only); when it does, this
// type should become a thin wrapper over the core verify call. See the F_nrv_signature
// report, handoff H1.
type MLDSA65 struct{}

// Algorithm returns SignatureAlgorithmMLDSA65.
func (MLDSA65) Algorithm() string { return SignatureAlgorithmMLDSA65 }

// Verify returns false for any key or signature of the wrong length, and for any
// verification failure. It never panics on malformed input.
func (MLDSA65) Verify(publicKey, message, signature []byte) (ok bool) {
	defer func() {
		if recover() != nil {
			ok = false
		}
	}()
	if len(publicKey) != mldsa65.PublicKeySize || len(signature) != mldsa65.SignatureSize {
		return false
	}
	pk, err := mldsa65.Scheme().UnmarshalBinaryPublicKey(publicKey)
	if err != nil {
		return false
	}
	return mldsa65.Scheme().Verify(pk, message, signature, nil)
}
