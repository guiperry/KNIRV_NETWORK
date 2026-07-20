package oracle

import (
	"crypto/ecdsa"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/knirvcorp/knirvoracle/internal/oracle/consensus"
	"github.com/knirvcorp/knirvoracle/internal/oracle/mmr"
	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
	"go.uber.org/zap"
)

func newTestChain(t *testing.T, chainID string) (types.ChainRegistration, *ecdsa.PrivateKey) {
	t.Helper()
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("key: %v", err)
	}
	cpk := crypto.CompressPubkey(&key.PublicKey)
	addr := types.OracleAddress(&key.PublicKey)
	return types.ChainRegistration{
		ChainID:     chainID,
		Authors:     []types.RegisteredAuthor{{Address: addr, PubKey: cpk, Weight: 1}},
		QuorumNumer: 1,
		QuorumDenom: 1,
		ProofWindow: 256,
	}, key
}

// newTestOracle builds a minimal Oracle with a live checkpoint pipeline and a
// consensus engine so admission can compute FinalByHeight.
func newTestOracle(t *testing.T) *Oracle {
	t.Helper()
	o := &Oracle{
		checkpoint: newCheckpointState(""),
		consensusEngine: consensus.NewConsensusEngine("knirvchain-test", 0, false, zap.NewNop()),
		logger:    zap.NewNop(),
	}
	return o
}

func TestCheckpointAdmissionAppendsMMR(t *testing.T) {
	oracleInst := newTestOracle(t)
	reg, key := newTestChain(t, "knirvchain-1")

	cp := &types.Checkpoint{
		SchemaVersion: "knirv.checkpoint.v1",
		ChainID:       reg.ChainID,
		StartHeight:   1,
		EndHeight:     64,
		Proposer:      reg.Authors[0].Address,
	}
	if err := types.SignCheckpoint(cp, key); err != nil {
		t.Fatalf("sign: %v", err)
	}

	if _, err := oracleInst.SubmitCheckpoint(cp); err == nil {
		t.Fatal("expected admission to fail on unregistered chain")
	}

	if err := oracleInst.RegisterChain(&reg); err != nil {
		t.Fatalf("register: %v", err)
	}
	rootBefore := oracleInst.MMRRoot()
	rec, err := oracleInst.SubmitCheckpoint(cp)
	if err != nil {
		t.Fatalf("admit: %v", err)
	}
	if rec.MMRPosition != 0 {
		t.Fatalf("expected first leaf at 0, got %d", rec.MMRPosition)
	}
	rootAfter := oracleInst.MMRRoot()
	if rootAfter == rootBefore {
		t.Fatal("MMR root should change after append")
	}

	proof, err := oracleInst.MMRProof(rec.MMRPosition)
	if err != nil {
		t.Fatalf("proof: %v", err)
	}
	if !mmr.VerifyProof(rootAfter, rec.LeafHash, proof, proof.TreeSize) {
		t.Fatal("MMR inclusion proof failed")
	}

	cp2 := &types.Checkpoint{
		SchemaVersion: "knirv.checkpoint.v1",
		ChainID:       reg.ChainID,
		StartHeight:   65,
		EndHeight:     128,
		PrevCheckHash: cp.Digest(),
		Proposer:      reg.Authors[0].Address,
	}
	if err := types.SignCheckpoint(cp2, key); err != nil {
		t.Fatalf("sign2: %v", err)
	}
	if _, err := oracleInst.SubmitCheckpoint(cp2); err != nil {
		t.Fatalf("admit2: %v", err)
	}

	cpBad := &types.Checkpoint{
		SchemaVersion: "knirv.checkpoint.v1",
		ChainID:       reg.ChainID,
		StartHeight:   999,
		EndHeight:     1000,
		Proposer:      reg.Authors[0].Address,
	}
	if err := types.SignCheckpoint(cpBad, key); err != nil {
		t.Fatalf("signBad: %v", err)
	}
	if _, err := oracleInst.SubmitCheckpoint(cpBad); err == nil {
		t.Fatal("expected non-contiguous checkpoint to be rejected")
	}
}
