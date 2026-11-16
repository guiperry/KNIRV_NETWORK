package nrv

import (
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
)

// SignatureAlgorithmMLDSA65 is the only algorithm name a .nrv PQCManifest may
// carry for frame signatures. Verification refuses any other name, so a registry
// that claims "Dilithium-3" (a pre-FIPS-204 scheme with different signature bytes)
// can never be accepted as ML-DSA-65.
const SignatureAlgorithmMLDSA65 = "ML-DSA-65"

// frameSigDomain separates the framed signing message from any other ML-DSA-65
// message the same key may sign.
const frameSigDomain = "KNIRV-NRV-FRAME-SIG/v1\x00"

// SignatureScheme is the verification port for .nrv frame signatures. The core
// (KNIRVSDK Rust) is the crypto source of truth; a binding implements this
// interface. MLDSA65 is the transitional Go binding (see mldsa65.go).
type SignatureScheme interface {
	// Algorithm returns the PQCManifest algorithm name the scheme verifies.
	Algorithm() string
	// Verify reports whether signature is a valid signature of message under
	// publicKey. It must return false, never panic, on malformed input.
	Verify(publicKey, message, signature []byte) bool
}

// FrameSigningMessage returns the exact bytes a frame signature covers.
//
// A frame without a payload section keeps the legacy form: the signature covers
// the bracket binary alone (KNIRVBASE_Upgrade.md §3.3), so existing signed frames
// still verify. A frame with a payload section is signed over a domain-separated,
// length-prefixed encoding of both sections, so tampering with either one fails
// and the two forms can never be confused with each other.
func FrameSigningMessage(brackets []byte, payload []byte, hasPayload bool) []byte {
	if !hasPayload {
		return brackets
	}
	out := make([]byte, 0, len(frameSigDomain)+16+len(brackets)+len(payload))
	out = append(out, frameSigDomain...)
	var n [8]byte
	binary.LittleEndian.PutUint64(n[:], uint64(len(brackets)))
	out = append(out, n[:]...)
	out = append(out, brackets...)
	binary.LittleEndian.PutUint64(n[:], uint64(len(payload)))
	out = append(out, n[:]...)
	out = append(out, payload...)
	return out
}

// Signature failure kinds, matchable with errors.Is.
var (
	ErrAlgorithmMismatch = errors.New("nrv signature: algorithm mismatch")
	ErrUnsignedFrame     = errors.New("nrv signature: frame has no signature")
	ErrBadSignature      = errors.New("nrv signature: signature does not verify")
	ErrMalformed         = errors.New("nrv signature: malformed frame section")
)

// SignatureError names the frame and the reason a signature check failed.
type SignatureError struct {
	FrameID string
	Kind    error
	Detail  string
}

func (e *SignatureError) Error() string {
	if e.Detail == "" {
		return fmt.Sprintf("%v (frame %q)", e.Kind, e.FrameID)
	}
	return fmt.Sprintf("%v (frame %q): %s", e.Kind, e.FrameID, e.Detail)
}

func (e *SignatureError) Unwrap() error { return e.Kind }

// VerifyFrameSignature checks one live frame of a container held in file (the
// whole file, as bytes or a mapping). It fails closed: a missing signature, a
// section outside the file, an undecodable signature or a failed verification
// all return an error.
func VerifyFrameSignature(scheme SignatureScheme, publicKey []byte, reg *Registry, file []byte, frameID string) error {
	if scheme == nil {
		return &SignatureError{FrameID: frameID, Kind: ErrAlgorithmMismatch, Detail: "no scheme"}
	}
	if reg == nil {
		return &SignatureError{FrameID: frameID, Kind: ErrMalformed, Detail: "no registry"}
	}
	if reg.PQCManifest.Algorithm != scheme.Algorithm() {
		return &SignatureError{FrameID: frameID, Kind: ErrAlgorithmMismatch,
			Detail: fmt.Sprintf("manifest %q, verifier %q", reg.PQCManifest.Algorithm, scheme.Algorithm())}
	}
	var entry *FrameEntry
	for i := range reg.Frames {
		if reg.Frames[i].ID == frameID {
			entry = &reg.Frames[i]
			break
		}
	}
	if entry == nil {
		return &SignatureError{FrameID: frameID, Kind: ErrMalformed, Detail: "frame not in registry"}
	}
	sigB64, ok := reg.PQCManifest.FrameSignatures[frameID]
	if !ok || sigB64 == "" {
		return &SignatureError{FrameID: frameID, Kind: ErrUnsignedFrame}
	}
	sig, err := base64.StdEncoding.DecodeString(sigB64)
	if err != nil {
		return &SignatureError{FrameID: frameID, Kind: ErrBadSignature, Detail: "signature is not base64"}
	}
	brackets, err := section(file, int64(entry.Brackets.Offset), int64(entry.Brackets.Length))
	if err != nil {
		return &SignatureError{FrameID: frameID, Kind: ErrMalformed, Detail: "brackets: " + err.Error()}
	}
	var payload []byte
	hasPayload := entry.Payload != nil
	if hasPayload {
		payload, err = section(file, int64(entry.Payload.Offset), int64(entry.Payload.Length))
		if err != nil {
			return &SignatureError{FrameID: frameID, Kind: ErrMalformed, Detail: "payload: " + err.Error()}
		}
	}
	msg := FrameSigningMessage(brackets, payload, hasPayload)
	if !scheme.Verify(publicKey, msg, sig) {
		return &SignatureError{FrameID: frameID, Kind: ErrBadSignature}
	}
	return nil
}

// VerifyAllFrameSignatures verifies every live (non-tombstoned) frame. Tombstoned
// frames are skipped because their data is logically deleted. It stops at the
// first failure and returns it.
func VerifyAllFrameSignatures(scheme SignatureScheme, publicKey []byte, reg *Registry, file []byte) error {
	if reg == nil {
		return &SignatureError{Kind: ErrMalformed, Detail: "no registry"}
	}
	for _, f := range reg.Frames {
		if f.Tombstone != nil {
			continue
		}
		if err := VerifyFrameSignature(scheme, publicKey, reg, file, f.ID); err != nil {
			return err
		}
	}
	return nil
}

// section returns file[off:off+length] after checking the range is inside file.
func section(file []byte, off, length int64) ([]byte, error) {
	if off < 0 || length < 0 || off > int64(len(file)) || length > int64(len(file))-off {
		return nil, fmt.Errorf("range [%d,%d) outside file of %d bytes", off, off+length, len(file))
	}
	return file[off : off+length], nil
}
