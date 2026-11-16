package nrvtrainer

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/knirvcorp/knirvbase/pkg/nrv"
)

// A wrong-length trusted key is a configuration fault. KNIRVBASE's ML-DSA-65
// verifier returns a plain false for it, which would look like ErrBadSignature,
// so the trainer must catch it up front and report verify-error.
func TestSignatureTruncatedKeyIsVerifyErrorNotInvalid(t *testing.T) {
	pk, sk := newMLDSAKey(t)
	pkBytes, err := pk.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "truncated-key.nrv")
	writeCanonicalSigned(t, path, canonicalSpec{signKey: sk})

	result, trained := runReportOnly(t, path, MLDSA65Check(pkBytes[:len(pkBytes)-1]))
	if trained != 1 {
		t.Fatalf("trained frames = %d, want 1 (report-only)", trained)
	}
	if got := result.Asset.SignatureCounts[SignatureInvalid]; got != 0 {
		t.Fatalf("invalid = %d, want 0 (key fault must not look like a bad signature); counts=%v", got, result.Asset.SignatureCounts)
	}
	if got := result.Asset.SignatureCounts[SignatureVerifyError]; got != 1 {
		t.Fatalf("verify-error = %d, want 1; counts=%v", got, result.Asset.SignatureCounts)
	}
	if !result.Asset.VerificationSkipped {
		t.Fatal("VerificationSkipped = false, want true when the trusted key is unusable")
	}
	if len(result.Asset.SignatureFlagged) != 1 || !strings.Contains(result.Asset.SignatureFlagged[0].Detail, "ML-DSA-65 needs") {
		t.Fatalf("flagged = %+v, want key-length reason", result.Asset.SignatureFlagged)
	}
}

func TestSignatureKeyProblem(t *testing.T) {
	pk, _ := newMLDSAKey(t)
	good, err := pk.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	long := append(append([]byte{}, good...), 0)
	cases := map[string]struct {
		check *SignatureCheck
		want  bool // true = a key problem is expected
	}{
		"nil check":        {nil, true},
		"nil scheme":       {&SignatureCheck{PublicKey: good}, true},
		"empty key":        {MLDSA65Check(nil), true},
		"truncated by one": {MLDSA65Check(good[:len(good)-1]), true},
		"extended by one":  {MLDSA65Check(long), true},
		"valid key":        {MLDSA65Check(good), false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := tc.check.keyProblem()
			if (got != "") != tc.want {
				t.Fatalf("keyProblem() = %q, want problem=%v", got, tc.want)
			}
		})
	}
}

// Every KNIRVBASE failure kind maps to one status. Only ErrBadSignature is invalid.
func TestMapVerifyErrorKinds(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want SignatureStatus
	}{
		{"verifies", nil, SignatureVerified},
		{"bad signature", &nrv.SignatureError{FrameID: "f1", Kind: nrv.ErrBadSignature}, SignatureInvalid},
		{"bad signature bare", nrv.ErrBadSignature, SignatureInvalid},
		{"malformed: section outside file", &nrv.SignatureError{FrameID: "f1", Kind: nrv.ErrMalformed, Detail: "brackets: range [100,180) outside file of 1 bytes"}, SignatureVerifyError},
		{"malformed: payload outside file", &nrv.SignatureError{FrameID: "f1", Kind: nrv.ErrMalformed, Detail: "payload: range outside file"}, SignatureVerifyError},
		{"malformed: no registry", &nrv.SignatureError{FrameID: "f1", Kind: nrv.ErrMalformed, Detail: "no registry"}, SignatureVerifyError},
		{"algorithm mismatch", &nrv.SignatureError{FrameID: "f1", Kind: nrv.ErrAlgorithmMismatch}, SignatureVerifyError},
		{"unsigned frame", &nrv.SignatureError{FrameID: "f1", Kind: nrv.ErrUnsignedFrame}, SignatureVerifyError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, _ := mapVerifyError(tc.err)
			if got != tc.want {
				t.Fatalf("mapVerifyError(%v) = %q, want %q", tc.err, got, tc.want)
			}
		})
	}
}

// A verifier-side ErrMalformed (section outside the file) reaches the trainer
// through classifyFrameSignature and must be verify-error, not invalid.
func TestClassifySectionOutsideFileIsVerifyError(t *testing.T) {
	pk, _ := newMLDSAKey(t)
	pkBytes, err := pk.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	registry := nrv.Registry{
		PQCManifest: nrv.PQCManifest{
			Algorithm:       nrv.SignatureAlgorithmMLDSA65,
			FrameSignatures: map[string]string{"f1": "AAAA"}, // valid base64, so the section check is reached
		},
	}
	entry := nrv.FrameEntry{ID: "f1", Brackets: nrv.BracketBinaryMap{Count: 1, Offset: 100, Length: 80}}
	file := []byte{0} // the bracket section [100,180) is outside this 1-byte file

	status, detail := classifyFrameSignature(MLDSA65Check(pkBytes), "", &registry, file, entry)
	if status != SignatureVerifyError {
		t.Fatalf("status = %q (%s), want verify-error", status, detail)
	}
	if !strings.Contains(detail, "outside file") {
		t.Fatalf("detail = %q, want section-outside-file reason", detail)
	}
}
