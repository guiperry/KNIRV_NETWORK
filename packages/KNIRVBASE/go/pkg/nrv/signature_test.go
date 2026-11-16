package nrv

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"testing"

	"github.com/cloudflare/circl/sign/mldsa/mldsa65"
)

// testKey returns an ML-DSA-65 key pair as raw bytes.
func testKey(t *testing.T) (pk, sk []byte) {
	t.Helper()
	pub, priv, err := mldsa65.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}
	return pub.Bytes(), priv.Bytes()
}

func signMsg(t *testing.T, sk []byte, msg []byte) []byte {
	t.Helper()
	priv, err := mldsa65.Scheme().UnmarshalBinaryPrivateKey(sk)
	if err != nil {
		t.Fatalf("unmarshal sk: %v", err)
	}
	return mldsa65.Scheme().Sign(priv, msg, nil)
}

// buildContainer lays out a minimal container: brackets then payload, both
// signed with sk. A frame with withPayload=false is the legacy bracket-only form.
type fixture struct {
	reg  *Registry
	file []byte
	pk   []byte
	sk   []byte
}

func buildFixture(t *testing.T, withPayload bool) fixture {
	t.Helper()
	pk, sk := testKey(t)
	brackets := bytes.Repeat([]byte{0xAB}, BracketSize*2)
	file := append([]byte{}, brackets...)
	entry := FrameEntry{
		ID:       "frame-1",
		Brackets: BracketBinaryMap{Count: 2, Offset: 0, Length: len(brackets)},
	}
	var payload []byte
	if withPayload {
		p, err := EncodeFramePayload(FramePayload{TokenWindow: &TokenWindowPayload{Tokens: []uint32{1, 2, 3}, Start: 0}})
		if err != nil {
			t.Fatalf("encode payload: %v", err)
		}
		payload = p
		entry.Payload = &ModalityIndex{Offset: len(file), Length: len(payload)}
		file = append(file, payload...)
	}
	sig := signMsg(t, sk, FrameSigningMessage(brackets, payload, withPayload))
	reg := &Registry{
		Frames: []FrameEntry{entry},
		PQCManifest: PQCManifest{
			KeyID:           "k1",
			Algorithm:       SignatureAlgorithmMLDSA65,
			FrameSignatures: map[string]string{"frame-1": base64.StdEncoding.EncodeToString(sig)},
		},
	}
	return fixture{reg: reg, file: file, pk: pk, sk: sk}
}

func TestVerifyFrameSignature_ValidWithPayload(t *testing.T) {
	f := buildFixture(t, true)
	if err := VerifyFrameSignature(MLDSA65{}, f.pk, f.reg, f.file, "frame-1"); err != nil {
		t.Fatalf("valid signed frame with payload rejected: %v", err)
	}
	if err := VerifyAllFrameSignatures(MLDSA65{}, f.pk, f.reg, f.file); err != nil {
		t.Fatalf("VerifyAll rejected valid container: %v", err)
	}
}

func TestVerifyFrameSignature_ValidLegacyBracketsOnly(t *testing.T) {
	f := buildFixture(t, false)
	if err := VerifyFrameSignature(MLDSA65{}, f.pk, f.reg, f.file, "frame-1"); err != nil {
		t.Fatalf("legacy bracket-only signature rejected: %v", err)
	}
}

func TestVerifyFrameSignature_Mutants(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(f *fixture, other []byte)
		want   error
	}{
		{"tampered payload byte", func(f *fixture, _ []byte) {
			f.file[len(f.file)-1] ^= 0x01
		}, ErrBadSignature},
		{"tampered bracket byte", func(f *fixture, _ []byte) {
			f.file[0] ^= 0x01
		}, ErrBadSignature},
		{"wrong key", func(f *fixture, other []byte) {
			f.pk = other
		}, ErrBadSignature},
		{"truncated signature", func(f *fixture, _ []byte) {
			raw, _ := base64.StdEncoding.DecodeString(f.reg.PQCManifest.FrameSignatures["frame-1"])
			f.reg.PQCManifest.FrameSignatures["frame-1"] = base64.StdEncoding.EncodeToString(raw[:len(raw)-1])
		}, ErrBadSignature},
		{"payload stripped from manifest", func(f *fixture, _ []byte) {
			f.reg.Frames[0].Payload = nil
		}, ErrBadSignature},
		{"signature from wrong algorithm name", func(f *fixture, _ []byte) {
			f.reg.PQCManifest.Algorithm = "Dilithium-3"
		}, ErrAlgorithmMismatch},
		{"unsigned frame", func(f *fixture, _ []byte) {
			delete(f.reg.PQCManifest.FrameSignatures, "frame-1")
		}, ErrUnsignedFrame},
		{"payload range outside file", func(f *fixture, _ []byte) {
			f.reg.Frames[0].Payload.Length = len(f.file) + 10
		}, ErrMalformed},
		{"signature not base64", func(f *fixture, _ []byte) {
			f.reg.PQCManifest.FrameSignatures["frame-1"] = "!!!"
		}, ErrBadSignature},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := buildFixture(t, true)
			otherPK, _ := testKey(t)
			tc.mutate(&f, otherPK)
			err := VerifyFrameSignature(MLDSA65{}, f.pk, f.reg, f.file, "frame-1")
			if err == nil {
				t.Fatalf("mutant %q was accepted", tc.name)
			}
			if !errors.Is(err, tc.want) {
				t.Fatalf("mutant %q: got %v, want %v", tc.name, err, tc.want)
			}
		})
	}
}

func TestVerifyAll_SkipsTombstoned(t *testing.T) {
	f := buildFixture(t, true)
	ts := int64(1)
	f.reg.Frames[0].Tombstone = &ts
	f.file[0] ^= 0x01 // would fail if verified
	if err := VerifyAllFrameSignatures(MLDSA65{}, f.pk, f.reg, f.file); err != nil {
		t.Fatalf("tombstoned frame should be skipped, got %v", err)
	}
}

func TestMLDSA65_MalformedInputIsFalseNotPanic(t *testing.T) {
	pk, sk := testKey(t)
	msg := []byte("m")
	sig := signMsg(t, sk, msg)
	var v MLDSA65
	if !v.Verify(pk, msg, sig) {
		t.Fatal("good signature rejected")
	}
	if v.Verify(pk[:len(pk)-1], msg, sig) {
		t.Error("short public key accepted")
	}
	if v.Verify(pk, msg, sig[:len(sig)-1]) {
		t.Error("truncated signature accepted")
	}
	if v.Verify(nil, msg, nil) {
		t.Error("empty input accepted")
	}
}

func TestFrameSigningMessage_LegacyIsRawBrackets(t *testing.T) {
	b := []byte{1, 2, 3}
	if got := FrameSigningMessage(b, nil, false); !bytes.Equal(got, b) {
		t.Fatalf("legacy form changed: %v", got)
	}
	framed := FrameSigningMessage(b, []byte{9}, true)
	if bytes.Equal(framed, b) || len(framed) <= len(b) {
		t.Fatal("framed form must differ from legacy form")
	}
}

// A nil registry must fail closed with an error, never panic.
func TestVerifyFrameSignature_NilRegistryFailsClosed(t *testing.T) {
	f := buildFixture(t, true)
	err := VerifyFrameSignature(MLDSA65{}, f.pk, nil, f.file, "frame-1")
	if err == nil {
		t.Fatal("nil registry must return an error")
	}
	if !errors.Is(err, ErrMalformed) {
		t.Fatalf("want ErrMalformed, got %v", err)
	}
	if err := VerifyAllFrameSignatures(MLDSA65{}, f.pk, nil, f.file); err == nil {
		t.Fatal("VerifyAllFrameSignatures with nil registry must return an error")
	}
}
