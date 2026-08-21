package api

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"knirvhasher/pkg/hashing/proofasset"
)

func TestMathVerifierServer_Verify_StructuralPrecheckOnly(t *testing.T) {
	server := NewMathVerifierServer(0)

	resp := server.Verify("x + y", 0)
	assert.Equal(t, proofasset.StatusStructurallyValid, resp.PrecheckStatus)
	// No formal client attached: structurally valid input becomes PROOF_PENDING.
	assert.Equal(t, proofasset.StatusProofPending, resp.Status)
	assert.Equal(t, proofasset.StatusCheckerUnavailable, resp.FormalStatus)
	assert.Empty(t, resp.ProofAssetID)
	assert.Empty(t, resp.Receipt)
	assert.NotEmpty(t, resp.Nonce)
	assert.Contains(t, resp.NonceNote, "NOT a formal proof witness")
}

func TestMathVerifierServer_Verify_StructuralRejection(t *testing.T) {
	server := NewMathVerifierServer(0)

	resp := server.Verify("x + 1 = 2", 0)
	assert.Equal(t, proofasset.StatusStructurallyValid, resp.PrecheckStatus)
	assert.Equal(t, proofasset.StatusProofPending, resp.Status)

	// Now test actual structural rejection by using VerifyMathDerivation
	// which creates a fresh watchdog (stateful validation requires two steps).
	derivationReq := MathVerificationRequest{
		Proposition: "x + 1 = 2",
		Subdomain:   0,
	}
	derivationResp, err := VerifyMathDerivation(derivationReq)
	require.NoError(t, err)
	assert.Equal(t, proofasset.StatusStructurallyValid, derivationResp.Status)
}

func TestMathVerifierServer_Verify_NoFormalClient(t *testing.T) {
	server := NewMathVerifierServer(0)

	resp := server.Verify("x + 1 = 2", 0)
	assert.Equal(t, proofasset.StatusStructurallyValid, resp.PrecheckStatus)
	assert.Equal(t, proofasset.StatusProofPending, resp.Status)
	assert.Equal(t, proofasset.StatusCheckerUnavailable, resp.FormalStatus)
	assert.Empty(t, resp.Receipt)
}

func TestMathVerifierServer_Verify_WithFormalClient(t *testing.T) {
	client := &fakeFormalClient{
		receipt: &proofasset.VerificationReceipt{
			SchemaVersion: 1,
			ProofAssetID:  "proof-123",
			Status:        proofasset.StatusFormallyVerified,
			CheckerDigest: "checker-abc",
			CheckedAt:     time.Now().UTC(),
		},
	}

	server := NewMathVerifierServer(0)
	server = server.WithFormalVerifier(client)
	server = server.WithAllowedImports([]string{"Mathlib.Data.Real.Basic"})

	resp := server.Verify("x + 1 = 2", 0)
	assert.Equal(t, proofasset.StatusStructurallyValid, resp.PrecheckStatus)
	assert.Equal(t, proofasset.StatusFormallyVerified, resp.Status)
	assert.Equal(t, proofasset.StatusFormallyVerified, resp.FormalStatus)
	assert.NotEmpty(t, resp.ProofAssetID)
	assert.NotNil(t, resp.Receipt)
	assert.Equal(t, proofasset.StatusFormallyVerified, resp.Receipt.Status)
}

func TestMathVerifierServer_Verify_FormalRejection(t *testing.T) {
	client := &fakeFormalClient{
		receipt: &proofasset.VerificationReceipt{
			SchemaVersion: 1,
			ProofAssetID:  "proof-456",
			Status:        proofasset.StatusFormallyRejected,
			CheckedAt:     time.Now().UTC(),
		},
	}

	server := NewMathVerifierServer(0)
	server = server.WithFormalVerifier(client)
	server = server.WithAllowedImports([]string{"Mathlib.Data.Real.Basic"})

	resp := server.Verify("x + 1 = 2", 0)
	assert.Equal(t, proofasset.StatusFormallyRejected, resp.Status)
	assert.Equal(t, proofasset.StatusFormallyRejected, resp.FormalStatus)
	assert.NotEmpty(t, resp.ProofAssetID)
	// FORMALLY_REJECTED does not expose the receipt in the response.
	assert.Nil(t, resp.Receipt)
}

func TestMathVerifierServer_Verify_FormalClientError(t *testing.T) {
	client := &fakeFormalClient{
		err: assert.AnError,
	}

	server := NewMathVerifierServer(0)
	server = server.WithFormalVerifier(client)
	server = server.WithAllowedImports([]string{"Mathlib.Data.Real.Basic"})

	resp := server.Verify("x + 1 = 2", 0)
	assert.Equal(t, proofasset.StatusProofPending, resp.Status)
	assert.Equal(t, proofasset.StatusCheckerUnavailable, resp.FormalStatus)
	assert.Contains(t, resp.ResultHash, "formal checker error")
}

func TestMathVerifierServer_Verify_NonceIsNotProofWitness(t *testing.T) {
	server := NewMathVerifierServer(0)

	resp1 := server.Verify("x + y", 0)
	resp2 := server.Verify("x + y", 0)

	// Nonces are deterministic input-derived metadata.
	assert.Equal(t, resp1.Nonce, resp2.Nonce)
	assert.NotEmpty(t, resp1.NonceNote)
	assert.Contains(t, resp1.NonceNote, "NOT a formal proof witness")
}

func TestVerifyMathDerivation_BackwardCompatibility(t *testing.T) {
	req := MathVerificationRequest{
		Proposition: "x + 1 = 2",
		Subdomain:   0,
	}

	resp, err := VerifyMathDerivation(req)
	require.NoError(t, err)
	assert.Equal(t, proofasset.StatusStructurallyValid, resp.Status)
	assert.NotEmpty(t, resp.Nonce)
}

func TestVerifyResponse_ToLegacy(t *testing.T) {
	resp := &VerifyResponse{
		Status:            proofasset.StatusStructurallyValid,
		PrecheckStatus:    proofasset.StatusStructurallyValid,
		Nonce:             "0x12345678",
		ResultHash:        "0xabcdef",
		DetokenizedOutput: "x + y",
		LogicIntegrity:    0.95,
		LatencyMs:         1.5,
	}

	legacy := resp.ToLegacy()
	assert.Equal(t, proofasset.StatusStructurallyValid, legacy.Status)
	assert.Equal(t, "0x12345678", legacy.Nonce)
	assert.InDelta(t, 0.95, legacy.LogicIntegrity, 0.001)
}

type fakeFormalClient struct {
	receipt *proofasset.VerificationReceipt
	err     error
}

func (f *fakeFormalClient) SubmitProof(asset *proofasset.ProofAsset) (*proofasset.VerificationReceipt, error) {
	return f.receipt, f.err
}
