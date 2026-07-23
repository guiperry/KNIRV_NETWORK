package oracle

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

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
		checkpoint:      newCheckpointState(""),
		rollups:         make(map[string]*types.RollupRecord),
		consensusEngine: consensus.NewConsensusEngine("knirvchain-test", 0, false, "", zap.NewNop()),
		logger:          zap.NewNop(),
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
	if err := types.SignRegistration(&reg, key); err != nil {
		t.Fatalf("sign registration: %v", err)
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

// TestRollupProjectsToProvisionalCheckpoint verifies the Phase-3 legacy bridge:
// a RollupRecord submitted via SubmitRollup (by a registered chain, signed by
// one of its authors) is projected into the checkpoint MMR as a provisional,
// rollup-tagged leaf — admission requires an author signature quorum just
// like checkpoints do, but the projected checkpoint itself stays audit-only
// and cannot reach finality (see checkpoint.go's Source doc comment).
func TestRollupProjectsToProvisionalCheckpoint(t *testing.T) {
	oracleInst := newTestOracle(t)
	reg, key := newTestChain(t, "knirvchain-legacy")
	if err := types.SignRegistration(&reg, key); err != nil {
		t.Fatalf("sign registration: %v", err)
	}
	if err := oracleInst.RegisterChain(&reg); err != nil {
		t.Fatalf("register: %v", err)
	}

	rootBefore := oracleInst.MMRRoot()
	rec := &types.RollupRecord{
		ID:          "legacy-1",
		BatchRoot:   "aabbccddeeff00112233445566778899aabbccddeeff00112233445566778899",
		ChainID:     "knirvchain-legacy",
		StartHeight: 1,
		EndHeight:   100,
		Proposer:    reg.Authors[0].Address,
		Status:      types.RollupStatusSubmitted,
		SubmittedAt: time.Now().UTC(),
	}
	sigBytes, err := types.SignMessage(key, fmt.Sprintf("%x", rec.Digest()))
	if err != nil {
		t.Fatalf("sign rollup: %v", err)
	}
	rec.Signatures = []types.AuthorSig{{
		Address:   reg.Authors[0].Address,
		PubKeyHex: hex.EncodeToString(reg.Authors[0].PubKey),
		Signature: sigBytes,
	}}
	if err := oracleInst.SubmitRollup(rec); err != nil {
		t.Fatalf("submit rollup: %v", err)
	}
	rootAfter := oracleInst.MMRRoot()
	if rootAfter == rootBefore {
		t.Fatal("MMR root should change after rollup projection")
	}

	records := oracleInst.GetCheckpointRecords("knirvchain-legacy")
	if len(records) != 1 {
		t.Fatalf("expected 1 projected checkpoint record, got %d", len(records))
	}
	r := records[0]
	if r.Status != types.CheckpointProvisional {
		t.Fatalf("projected record status = %q, want provisional", r.Status)
	}
	if r.Source != "rollup:legacy-1" {
		t.Fatalf("projected record source = %q, want rollup:legacy-1", r.Source)
	}
	if r.Checkpoint.ChainID != "knirvchain-legacy" {
		t.Fatalf("projected record chain = %q", r.Checkpoint.ChainID)
	}

	proof, err := oracleInst.MMRProof(r.MMRPosition)
	if err != nil {
		t.Fatalf("proof: %v", err)
	}
	if !mmr.VerifyProof(rootAfter, r.LeafHash, proof, proof.TreeSize) {
		t.Fatal("rollup-projected leaf inclusion proof failed")
	}
}

func TestProjectedRollupCannotReachFinality(t *testing.T) {
	o := newPhase4Oracle(t)
	rec := projectTestRollup(t, o, "legacy-finality-bypass")
	registerPhase4Verifier(t, o, "v1")
	att := phase4Attestation(t, o, "v1", rec)
	if _, err := o.SubmitFinality(&types.FinalityRecord{CheckpointLeaf: rec.MMRPosition, ProofSystem: "hashchain-v0", Attestations: []types.VerifierAttestation{att}}); err == nil {
		t.Fatal("legacy projected checkpoint reached finality")
	}
	if _, err := o.SubmitAttestation(rec.Checkpoint.ChainID, rec.Checkpoint.StartHeight, att); err == nil {
		t.Fatal("legacy projected checkpoint accepted an incremental attestation")
	}
}

func TestInitialRegistrationRequiresQuorumAndExhaustedBondStopsAdmission(t *testing.T) {
	o := newTestOracle(t)
	reg, key := newTestChain(t, "bond-exhaustion")
	reg.Bond = 10
	reg.BondRemaining = 10
	if err := o.RegisterChain(&reg); err == nil {
		t.Fatal("unsigned initial registration was accepted")
	}
	if err := types.SignRegistration(&reg, key); err != nil {
		t.Fatal(err)
	}
	if err := o.RegisterChain(&reg); err != nil {
		t.Fatal(err)
	}
	if err := o.checkpoint.registry.ApplySlash(reg.ChainID, 10); err != nil {
		t.Fatal(err)
	}
	cp := &types.Checkpoint{SchemaVersion: "knirv.checkpoint.v1", ChainID: reg.ChainID, StartHeight: 1, EndHeight: 1, Proposer: reg.Authors[0].Address}
	if err := types.SignCheckpoint(cp, key); err != nil {
		t.Fatal(err)
	}
	if _, err := o.SubmitCheckpoint(cp); err == nil {
		t.Fatal("fully slashed chain admitted a checkpoint")
	}
}

// TestPhase3AdmissionCommitsAuditMMRAndRecovers proves the runtime wiring that
// was previously missing: the Oracle runs in non-validator mode, so the
// consensus loop never commits. SubmitCheckpoint (the HTTP admission path) must
// itself commit the audit leaf into the AppHash and persist it, and a reload
// from the same DataDir must reconstruct the identical AppHash.
func TestPhase3AdmissionCommitsAuditMMRAndRecovers(t *testing.T) {
	dir := t.TempDir()
	o := &Oracle{
		checkpoint:      newCheckpointState(dir),
		rollups:         make(map[string]*types.RollupRecord),
		consensusEngine: consensus.NewConsensusEngine("knirvchain-test", 0, false, dir, zap.NewNop()),
		logger:          zap.NewNop(),
	}
	reg, key := newTestChain(t, "knirvchain-1")
	if err := types.SignRegistration(&reg, key); err != nil {
		t.Fatal(err)
	}
	if err := o.RegisterChain(&reg); err != nil {
		t.Fatalf("register: %v", err)
	}
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
	if _, err := o.SubmitCheckpoint(cp); err != nil {
		t.Fatalf("admit: %v", err)
	}

	// The audit leaf log must now be persisted to disk.
	data, err := os.ReadFile(filepath.Join(dir, "audit_mmr_leaf_log.json"))
	if err != nil {
		t.Fatalf("audit mmr leaf log not persisted by admission: %v", err)
	}
	var leaves []mmr.Hash
	if err := json.Unmarshal(data, &leaves); err != nil {
		t.Fatalf("decode audit leaves: %v", err)
	}
	if len(leaves) != 1 {
		t.Fatalf("expected 1 persisted audit leaf, got %d", len(leaves))
	}

	// The AppHash must equal the checkpoint MMR root (same leaves, both MMRs).
	appHash := o.consensusEngine.GetApp().GetState().AppHash
	root := o.MMRRoot()
	if len(appHash) != 32 || bytes.Equal(appHash, make([]byte, 32)) {
		t.Fatalf("AppHash not set by admission: %x", appHash)
	}
	if !bytes.Equal(appHash, root[:]) {
		t.Fatalf("AppHash %x != checkpoint MMR root %x", appHash, root)
	}

	// Reload from the same DataDir (mirroring production NewOracle, which loads
	// the persisted state). The audit MMR and index MMR must both recover to
	// the same root.
	cs2, err := LoadCheckpointState(dir)
	if err != nil {
		t.Fatalf("reload checkpoint state: %v", err)
	}
	o2 := &Oracle{
		checkpoint:      cs2,
		rollups:         make(map[string]*types.RollupRecord),
		consensusEngine: consensus.NewConsensusEngine("knirvchain-test", 0, false, dir, zap.NewNop()),
		logger:          zap.NewNop(),
	}
	recommit, err := o2.consensusEngine.GetApp().Commit()
	if err != nil {
		t.Fatalf("recommit after reload: %v", err)
	}
	if !bytes.Equal(recommit, root[:]) {
		t.Fatalf("recovered AppHash %x != original root %x", recommit, root)
	}
	if got := o2.MMRRoot(); got != root {
		t.Fatalf("recovered checkpoint MMR %x != original %x", got, root)
	}
}
