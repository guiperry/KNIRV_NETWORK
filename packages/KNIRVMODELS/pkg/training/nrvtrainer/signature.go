package nrvtrainer

import (
	"errors"
	"fmt"

	"github.com/cloudflare/circl/sign/mldsa/mldsa65"
	"github.com/knirvcorp/knirvbase/pkg/nrv"
)

// SignatureStatus is the per-frame report of a .nrv frame signature check.
//
// The trainer only reports these values today. No status drops a frame from
// training: a frame that is unsigned or invalid still trains, and the dispatcher
// lists it in the summary. Enforcement waits for the owner's key distribution.
type SignatureStatus string

const (
	// SignatureVerified: the frame carries a signature and it verifies under the
	// configured public key.
	SignatureVerified SignatureStatus = "verified"
	// SignatureUnsigned: the frame carries no signature (NRV3 has no signature
	// section at all, so all of its frames are unsigned).
	SignatureUnsigned SignatureStatus = "unsigned"
	// SignatureInvalid: the frame is signed and the signature does not verify
	// under the trusted key (KNIRVBASE nrv.ErrBadSignature). This is the only
	// status that means the frame content is not what was signed.
	SignatureInvalid SignatureStatus = "invalid"
	// SignatureVerifyError: the check could not reach a verdict. No usable trusted
	// key is configured (missing, wrong length, or not a valid ML-DSA-65 key), the
	// manifest names an algorithm the verifier does not support, or the verifier
	// reported a malformed frame section. This is not evidence that the frame is
	// bad, so it is reported separately from invalid.
	SignatureVerifyError SignatureStatus = "verify-error"
)

// SignatureCheck configures frame-signature verification. A nil check, or one
// without a usable key, means no trusted public key is available: signed frames
// then report SignatureVerifyError, and nothing is verified.
type SignatureCheck struct {
	Scheme    nrv.SignatureScheme
	PublicKey []byte
}

// MLDSA65Check verifies ML-DSA-65 frame signatures against publicKey with the
// KNIRVBASE binding. The caller owns key selection; the trainer does not look up
// keys itself.
func MLDSA65Check(publicKey []byte) *SignatureCheck {
	return &SignatureCheck{Scheme: nrv.MLDSA65{}, PublicKey: publicKey}
}

// keyProblem validates the trusted key once per load. It returns "" when the key
// can be used, or the reason no frame can be verified with it. A key of the wrong
// length or one that does not parse is a configuration fault, so it must never be
// reported per frame as a bad signature.
func (c *SignatureCheck) keyProblem() string {
	if c == nil || c.Scheme == nil || len(c.PublicKey) == 0 {
		return "no trusted public key configured"
	}
	if _, ok := c.Scheme.(nrv.MLDSA65); ok {
		if len(c.PublicKey) != mldsa65.PublicKeySize {
			return fmt.Sprintf("trusted public key is %d bytes; ML-DSA-65 needs %d", len(c.PublicKey), mldsa65.PublicKeySize)
		}
		if _, err := mldsa65.Scheme().UnmarshalBinaryPublicKey(c.PublicKey); err != nil {
			return "trusted public key does not parse as ML-DSA-65: " + err.Error()
		}
	}
	return ""
}

// classifyFrameSignature checks one canonical frame. keyProblem is the result of
// check.keyProblem() for this load, computed once by the caller. It verifies only
// that frame by handing KNIRVBASE a one-frame registry that shares the manifest.
// Checking the whole registry for every frame would make a large asset quadratic.
func classifyFrameSignature(check *SignatureCheck, keyProblem string, registry *nrv.Registry, file []byte, entry nrv.FrameEntry) (SignatureStatus, string) {
	if registry.PQCManifest.FrameSignatures[entry.ID] == "" {
		return SignatureUnsigned, ""
	}
	if keyProblem != "" {
		return SignatureVerifyError, keyProblem
	}
	single := nrv.Registry{PQCManifest: registry.PQCManifest, Frames: []nrv.FrameEntry{entry}}
	err := nrv.VerifyFrameSignature(check.Scheme, check.PublicKey, &single, file, entry.ID)
	return mapVerifyError(err)
}

// mapVerifyError turns a KNIRVBASE verification result into a status. Only
// ErrBadSignature means the signature does not verify under the trusted key.
// Every other failure is the check not reaching a verdict: ErrMalformed covers
// verifier-side faults ("section outside file", "no registry"), and
// ErrAlgorithmMismatch and ErrUnsignedFrame are not frame content problems. The
// loader bounds-checks brackets and payload before classification, so a frame
// section fault is not reachable from a loaded asset in practice.
func mapVerifyError(err error) (SignatureStatus, string) {
	switch {
	case err == nil:
		return SignatureVerified, ""
	case errors.Is(err, nrv.ErrBadSignature):
		return SignatureInvalid, err.Error()
	default:
		return SignatureVerifyError, err.Error()
	}
}

// tallySignatures counts frame statuses on the asset, leaving tombstoned frames
// out (they are not in Frames).
func (a *Asset) tallySignatures() {
	a.SignatureCounts = map[SignatureStatus]int{
		SignatureVerified:    0,
		SignatureUnsigned:    0,
		SignatureInvalid:     0,
		SignatureVerifyError: 0,
	}
	for _, f := range a.Frames {
		a.SignatureCounts[f.Signature]++
	}
}
