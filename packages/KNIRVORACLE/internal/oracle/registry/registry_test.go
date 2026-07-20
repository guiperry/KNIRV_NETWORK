package registry

import (
	"testing"

	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
)

func TestRegistryRoundTrip(t *testing.T) {
	dir := t.TempDir()
	r, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	a1, _ := genAuthor(t)
	a2, _ := genAuthor(t)
	a3, _ := genAuthor(t)

	reg := &types.ChainRegistration{
		ChainID:       "test-chain",
		Authors:       []types.RegisteredAuthor{a1, a2, a3},
		QuorumNumer:   2,
		QuorumDenom:   3,
		LastHeight:    0,
		LastCheckHash: [32]byte{},
		Bond:          1000,
		ProofWindow:   256,
	}
	if err := r.Register(reg); err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := r.Persist(); err != nil {
		t.Fatalf("persist: %v", err)
	}

	// Reload from disk.
	r2, err := Load(dir)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	got, ok := r2.Get("test-chain")
	if !ok {
		t.Fatal("chain not found after reload")
	}
	if len(got.Authors) != 3 {
		t.Fatalf("authors = %d, want 3", len(got.Authors))
	}
	if got.Bond != 1000 {
		t.Fatalf("bond = %d, want 1000", got.Bond)
	}
	if got.ProofWindow != 256 {
		t.Fatalf("proof window = %d, want 256", got.ProofWindow)
	}
}

func TestRegistryVerification(t *testing.T) {
	dir := t.TempDir()
	r, _ := Load(dir)

	a1, k1 := genAuthor(t)
	a2, k2 := genAuthor(t)
	a3, _ := genAuthor(t)
	reg := &types.ChainRegistration{
		ChainID:       "test-chain",
		Authors:       []types.RegisteredAuthor{a1, a2, a3},
		QuorumNumer:   2,
		QuorumDenom:   3,
		LastHeight:    0,
		LastCheckHash: [32]byte{},
	}
	if err := r.Register(reg); err != nil {
		t.Fatalf("register: %v", err)
	}

	// Build a checkpoint and sign with 2 of 3 authors.
	cp := &types.Checkpoint{
		SchemaVersion: "knirv.checkpoint.v1",
		ChainID:       "test-chain",
		StartHeight:   1,
		EndHeight:     32,
		PrevCheckHash: [32]byte{},
		Proposer:      a1.Address,
	}
	pubHex := hexOf(a1.PubKey)
	sig1 := signCheckpoint(t, cp, k1, a1.Address, pubHex)
	cp.Signatures = append(cp.Signatures, sig1)

	// Only 1 signature -> quorum must fail.
	if err := r.VerifyCheckpointQuorum(cp); err == nil {
		t.Fatal("expected quorum failure with 1 sig")
	}

	// Add second signature.
	pubHex2 := hexOf(a2.PubKey)
	sig2 := signCheckpoint(t, cp, k2, a2.Address, pubHex2)
	cp.Signatures = append(cp.Signatures, sig2)

	if err := r.VerifyCheckpointQuorum(cp); err != nil {
		t.Fatalf("quorum should pass: %v", err)
	}

	// Advance continuity and re-verify a contiguous next checkpoint.
	hash := cp.Digest()
	if err := r.Advance("test-chain", 32, hash); err != nil {
		t.Fatalf("advance: %v", err)
	}
	cp2 := &types.Checkpoint{
		SchemaVersion: "knirv.checkpoint.v1",
		ChainID:       "test-chain",
		StartHeight:   33,
		EndHeight:     64,
		PrevCheckHash: hash,
		Proposer:      a1.Address,
	}
	sig1b := signCheckpoint(t, cp2, k1, a1.Address, pubHex)
	sig2b := signCheckpoint(t, cp2, k2, a2.Address, pubHex2)
	cp2.Signatures = []types.AuthorSig{sig1b, sig2b}
	if err := r.VerifyCheckpointQuorum(cp2); err != nil {
		t.Fatalf("contiguous quorum should pass: %v", err)
	}

	// A non-contiguous start must fail.
	cp3 := &types.Checkpoint{
		SchemaVersion: "knirv.checkpoint.v1",
		ChainID:       "test-chain",
		StartHeight:   100,
		EndHeight:     132,
		PrevCheckHash: hash,
		Proposer:      a1.Address,
	}
	sig3 := signCheckpoint(t, cp3, k1, a1.Address, pubHex)
	cp3.Signatures = []types.AuthorSig{sig3}
	if err := r.VerifyCheckpointQuorum(cp3); err == nil {
		t.Fatal("expected non-contiguity failure")
	}
}
