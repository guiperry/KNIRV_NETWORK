package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/knirvcorp/knirvbase/internal/crypto/pqc"
	"github.com/knirvcorp/knirvbase/pkg/nrv"
	"github.com/stretchr/testify/require"
)

// writeSignedPayloadFrame writes one frame with a payload section through the real
// NRVWriter and returns the registry, the raw file bytes and the public key.
func writeSignedPayloadFrame(t *testing.T) (*nrv.Registry, []byte, []byte, string, *pqc.PQCKeyPair) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "signed.nrv")
	kp, err := pqc.GeneratePQCKeyPair("sig-key", "signature")
	require.NoError(t, err)
	w, err := NewNRVWriter(path, kp)
	require.NoError(t, err)

	brackets := make([]byte, 80*2)
	for i := range brackets {
		brackets[i] = byte(i * 7)
	}
	metas := []nrv.BracketMeta{
		{ID: "b1", Type: nrv.DeltaTypeI, Offset: 0},
		{ID: "b2", Type: nrv.DeltaTypeI, Offset: 80},
	}
	payload := nrv.FramePayload{TokenWindow: &nrv.TokenWindowPayload{Tokens: []uint32{5, 6, 7, 8}, Brackets: 2}}
	require.NoError(t, w.AppendFrameWithPayload("frame-p", brackets, payload, metas,
		nrv.ThermoAtmosphere{AvgTempC: 40}, nrv.LinguisticMapping{Token: "t", Unit: "word"}))
	require.NoError(t, w.Close())

	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	// Re-open to read the registry exactly as it was persisted.
	w2, err := NewNRVWriter(path, kp)
	require.NoError(t, err)
	reg := w2.GetRegistry()
	require.NoError(t, w2.Close())
	return reg, raw, kp.DilithiumPublicKeyBytes, path, kp
}

func TestAppendFrameWithPayload_SignedSectionVerifies(t *testing.T) {
	reg, raw, pk, _, _ := writeSignedPayloadFrame(t)
	require.Len(t, reg.Frames, 1)
	require.NotNil(t, reg.Frames[0].Payload, "payload section must be indexed")
	require.Equal(t, "ML-DSA-65", reg.PQCManifest.Algorithm)
	require.NoError(t, nrv.VerifyAllFrameSignatures(nrv.MLDSA65{}, pk, reg, raw))

	// The indexed payload decodes back to what was written.
	p := reg.Frames[0].Payload
	got, err := nrv.DecodeFramePayload(raw[p.Offset : p.Offset+p.Length])
	require.NoError(t, err)
	require.Equal(t, []uint32{5, 6, 7, 8}, got.TokenWindow.Tokens)
}

func TestAppendFrameWithPayload_TamperedPayloadFails(t *testing.T) {
	reg, raw, pk, _, _ := writeSignedPayloadFrame(t)
	p := reg.Frames[0].Payload
	raw[p.Offset+p.Length-1] ^= 0x01 // flip one payload byte
	err := nrv.VerifyAllFrameSignatures(nrv.MLDSA65{}, pk, reg, raw)
	require.ErrorIs(t, err, nrv.ErrBadSignature)
}

func TestAppendFrameWithPayload_ReaderVerifyFrame(t *testing.T) {
	_, _, _, path, kp := writeSignedPayloadFrame(t)
	r, err := NewNRVReader(path)
	require.NoError(t, err)
	defer r.Close()
	ok, err := r.VerifyFrame("frame-p", kp)
	require.NoError(t, err)
	require.True(t, ok, "correct key must verify the payload-bearing frame")

	other, err := pqc.GeneratePQCKeyPair("other", "signature")
	require.NoError(t, err)
	ok, err = r.VerifyFrame("frame-p", other)
	require.NoError(t, err)
	require.False(t, ok, "wrong key must not verify")
}
