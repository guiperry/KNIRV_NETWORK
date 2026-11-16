package nrvtrainer

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudflare/circl/sign"
	"github.com/cloudflare/circl/sign/mldsa/mldsa65"
	"github.com/knirvcorp/knirvbase/pkg/nrv"
)

// canonicalSpec describes one canonical test asset. signKey nil writes no frame
// signature. algorithm overrides the manifest algorithm name when non-empty.
// tamper flips a bracket byte after signing.
type canonicalSpec struct {
	signKey   sign.PrivateKey
	algorithm string
	tamper    bool
}

// writeCanonicalSigned writes one valid-Z3 canonical frame "f1" with a payload.
// When signKey is set, the frame signature covers the same framed message that
// KNIRVBASE verifies.
func writeCanonicalSigned(t *testing.T, path string, spec canonicalSpec) {
	t.Helper()
	brackets := testBracket(t, 7)
	payload, err := json.Marshal(FramePayload{
		SupervisionEpisode: &SupervisionEpisodePayload{State: "state", Decision: "decision"},
	})
	if err != nil {
		t.Fatal(err)
	}
	bracketOffset := int64(headerSize + registryPadding)
	payloadOffset := bracketOffset + int64(len(brackets))

	manifest := map[string]any{
		"key_id":    "key-1",
		"algorithm": "ML-DSA-65",
	}
	if spec.algorithm != "" {
		manifest["algorithm"] = spec.algorithm
	}
	if spec.signKey != nil {
		msg := nrv.FrameSigningMessage(brackets, payload, true)
		sig := mldsa65.Scheme().Sign(spec.signKey, msg, nil)
		manifest["frame_signatures"] = map[string]string{"f1": base64.StdEncoding.EncodeToString(sig)}
	}
	registryJSON, err := json.Marshal(map[string]any{
		"version":    1,
		"dataset_id": "dataset-sig",
		"frames": []any{map[string]any{
			"id":       "f1",
			"z3":       map[string]any{"status": "VALID"},
			"brackets": map[string]any{"count": 1, "offset": bracketOffset, "length": len(brackets)},
			"payload":  map[string]any{"offset": payloadOffset, "length": len(payload)},
		}},
		"pqc_manifest": manifest,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(registryJSON) > registryPadding {
		t.Fatal("registry too large for test")
	}
	out := make([]byte, headerSize+registryPadding, int(payloadOffset)+len(payload))
	binary.LittleEndian.PutUint32(out[0:4], canonicalMagic)
	binary.LittleEndian.PutUint32(out[4:8], 1)
	copy(out[headerSize:], registryJSON)
	out = append(out, brackets...)
	out = append(out, payload...)
	binary.LittleEndian.PutUint32(out[8:12], uint32(len(out)))
	if spec.tamper {
		out[bracketOffset] ^= 0xFF
	}
	if err := os.WriteFile(path, out, 0o600); err != nil {
		t.Fatal(err)
	}
}

func newMLDSAKey(t *testing.T) (sign.PublicKey, sign.PrivateKey) {
	t.Helper()
	pk, sk, err := mldsa65.Scheme().GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	return pk, sk
}

// runReportOnly routes a file through the dispatcher and reports how many frames
// reached the trainer. It also returns the summary.
func runReportOnly(t *testing.T, path string, check *SignatureCheck) (*Result, int) {
	t.Helper()
	trained := 0
	dispatcher := New(&FuncTrainer{Name: TargetHASHER, Fn: func(_ context.Context, asset *Asset, _ Options) (TargetResult, error) {
		trained = len(asset.Frames)
		return TargetResult{Frames: len(asset.Frames)}, nil
	}})
	result, err := dispatcher.Train(context.Background(), path, Options{Targets: []Target{TargetHASHER}, Signature: check})
	if err != nil {
		t.Fatalf("Train: %v (signature status must never fail training)", err)
	}
	return result, trained
}

func TestSignatureSignedValidIsVerifiedAndTrains(t *testing.T) {
	pk, sk := newMLDSAKey(t)
	path := filepath.Join(t.TempDir(), "signed.nrv")
	writeCanonicalSigned(t, path, canonicalSpec{signKey: sk})
	pkBytes, err := pk.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	result, trained := runReportOnly(t, path, MLDSA65Check(pkBytes))
	if trained != 1 {
		t.Fatalf("trained frames = %d, want 1", trained)
	}
	s := result.Asset.SignatureCounts
	if s[SignatureVerified] != 1 || s[SignatureInvalid] != 0 || s[SignatureUnsigned] != 0 || s[SignatureVerifyError] != 0 {
		t.Fatalf("counts = %v", s)
	}
	if len(result.Asset.SignatureFlagged) != 0 || result.Asset.VerificationSkipped {
		t.Fatalf("summary = %#v", result.Asset)
	}
}

func TestSignatureUnsignedTrainsAndIsCounted(t *testing.T) {
	pk, _ := newMLDSAKey(t)
	pkBytes, _ := pk.MarshalBinary()
	path := filepath.Join(t.TempDir(), "unsigned.nrv")
	writeCanonicalSigned(t, path, canonicalSpec{})

	result, trained := runReportOnly(t, path, MLDSA65Check(pkBytes))
	if trained != 1 {
		t.Fatalf("trained frames = %d, want 1", trained)
	}
	if result.Asset.SignatureCounts[SignatureUnsigned] != 1 || len(result.Asset.SignatureFlagged) != 0 {
		t.Fatalf("summary = %#v", result.Asset)
	}
}

func TestSignatureTamperedIsInvalidFlaggedAndStillTrains(t *testing.T) {
	pk, sk := newMLDSAKey(t)
	pkBytes, _ := pk.MarshalBinary()
	path := filepath.Join(t.TempDir(), "tampered.nrv")
	writeCanonicalSigned(t, path, canonicalSpec{signKey: sk, tamper: true})

	result, trained := runReportOnly(t, path, MLDSA65Check(pkBytes))
	if trained != 1 {
		t.Fatalf("trained frames = %d, want 1 (report-only)", trained)
	}
	if result.Asset.SignatureCounts[SignatureInvalid] != 1 || result.Asset.SignatureCounts[SignatureVerified] != 0 {
		t.Fatalf("counts = %v", result.Asset.SignatureCounts)
	}
	if len(result.Asset.SignatureFlagged) != 1 || result.Asset.SignatureFlagged[0].Status != SignatureInvalid ||
		result.Asset.SignatureFlagged[0].Frame != "f1" || result.Asset.SignatureFlagged[0].Detail == "" {
		t.Fatalf("flagged = %#v", result.Asset.SignatureFlagged)
	}
}

func TestSignatureWrongKeyIsInvalidNotVerifyError(t *testing.T) {
	_, sk := newMLDSAKey(t)
	otherPK, _ := newMLDSAKey(t)
	otherBytes, _ := otherPK.MarshalBinary()
	path := filepath.Join(t.TempDir(), "wrongkey.nrv")
	writeCanonicalSigned(t, path, canonicalSpec{signKey: sk})

	result, trained := runReportOnly(t, path, MLDSA65Check(otherBytes))
	if trained != 1 {
		t.Fatalf("trained frames = %d, want 1", trained)
	}
	if result.Asset.SignatureCounts[SignatureInvalid] != 1 || result.Asset.SignatureCounts[SignatureVerifyError] != 0 {
		t.Fatalf("counts = %v", result.Asset.SignatureCounts)
	}
}

func TestSignatureNoTrustedKeyIsVerifyErrorNotInvalid(t *testing.T) {
	_, sk := newMLDSAKey(t)
	path := filepath.Join(t.TempDir(), "nokey.nrv")
	writeCanonicalSigned(t, path, canonicalSpec{signKey: sk})

	for name, check := range map[string]*SignatureCheck{
		"nil check":  nil,
		"empty key":  MLDSA65Check(nil),
		"nil scheme": {PublicKey: []byte{1}},
	} {
		t.Run(name, func(t *testing.T) {
			result, trained := runReportOnly(t, path, check)
			if trained != 1 {
				t.Fatalf("trained frames = %d, want 1", trained)
			}
			if result.Asset.SignatureCounts[SignatureVerifyError] != 1 || result.Asset.SignatureCounts[SignatureInvalid] != 0 {
				t.Fatalf("counts = %v", result.Asset.SignatureCounts)
			}
			if !result.Asset.VerificationSkipped {
				t.Fatal("VerificationSkipped = false, want true when no key is configured")
			}
		})
	}
}

func TestSignatureUnsupportedAlgorithmIsVerifyError(t *testing.T) {
	pk, sk := newMLDSAKey(t)
	pkBytes, _ := pk.MarshalBinary()
	path := filepath.Join(t.TempDir(), "dilithium.nrv")
	writeCanonicalSigned(t, path, canonicalSpec{signKey: sk, algorithm: "Dilithium-3"})

	result, trained := runReportOnly(t, path, MLDSA65Check(pkBytes))
	if trained != 1 {
		t.Fatalf("trained frames = %d, want 1", trained)
	}
	if result.Asset.SignatureCounts[SignatureVerifyError] != 1 || result.Asset.SignatureCounts[SignatureInvalid] != 0 {
		t.Fatalf("counts = %v", result.Asset.SignatureCounts)
	}
}

func TestSignatureNRV3IsUnsigned(t *testing.T) {
	path := filepath.Join(t.TempDir(), "asset.nrv")
	writeNRV3(t, path, []struct {
		id      string
		valid   bool
		tokens  []int
		payload FramePayload
	}{
		{id: "f1", valid: true, payload: FramePayload{TokenWindow: &TokenWindowPayload{Tokens: []uint32{1, 2, 3}}}},
		{id: "f2", valid: true, payload: FramePayload{TokenWindow: &TokenWindowPayload{Tokens: []uint32{4, 5, 6}}}},
	})
	pk, _ := newMLDSAKey(t)
	pkBytes, _ := pk.MarshalBinary()

	result, trained := runReportOnly(t, path, MLDSA65Check(pkBytes))
	if trained != 2 {
		t.Fatalf("trained frames = %d, want 2", trained)
	}
	if result.Asset.SignatureCounts[SignatureUnsigned] != 2 {
		t.Fatalf("counts = %v", result.Asset.SignatureCounts)
	}
}

// The Asset used by LoadAsset (no key) must still report per-frame status, so
// callers outside the dispatcher see the same classification.
func TestLoadAssetReportsSignatureStatusWithoutKey(t *testing.T) {
	_, sk := newMLDSAKey(t)
	path := filepath.Join(t.TempDir(), "signed.nrv")
	writeCanonicalSigned(t, path, canonicalSpec{signKey: sk})

	asset, err := LoadAsset(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(asset.Frames) != 1 || asset.Frames[0].Signature != SignatureVerifyError {
		t.Fatalf("frames = %#v", asset.Frames)
	}
}
