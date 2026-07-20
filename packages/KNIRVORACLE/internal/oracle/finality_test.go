package oracle

import (
	"testing"
	"time"

	"github.com/knirvcorp/knirvoracle/internal/oracle/consensus"
	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
	"go.uber.org/zap"
)

const testBatchRoot = "aabbccddeeff00112233445566778899aabbccddeeff00112233445566778899"

func newPhase4Oracle(t *testing.T) *Oracle {
	t.Helper()
	return &Oracle{
		checkpoint:      newCheckpointState(t.TempDir()),
		rollups:         make(map[string]*types.RollupRecord),
		consensusEngine: consensus.NewConsensusEngine("knirvchain-phase4", 0, false, t.TempDir(), zap.NewNop()),
		verifiers:       make(map[string]bool),
		logger:          zap.NewNop(),
	}
}

func projectTestRollup(t *testing.T, o *Oracle, chainID string) *types.CheckpointRecord {
	t.Helper()
	rr := &types.RollupRecord{
		ID:          "r-" + chainID,
		BatchRoot:   testBatchRoot,
		ChainID:     chainID,
		StartHeight: 1,
		EndHeight:   10,
		Status:      types.RollupStatusSubmitted,
		SubmittedAt: time.Now().UTC(),
	}
	if err := o.SubmitRollup(rr); err != nil {
		t.Fatalf("SubmitRollup: %v", err)
	}
	recs := o.GetCheckpointRecords(chainID)
	if len(recs) != 1 {
		t.Fatalf("want 1 projected record, got %d", len(recs))
	}
	return recs[0]
}

func TestFinalityDegradedAdmission(t *testing.T) {
	o := newPhase4Oracle(t)
	rec := projectTestRollup(t, o, "knirvchain-f")
	rootBefore := o.MMRRoot()

	fin := &types.FinalityRecord{
		SchemaVersion:  "knirv.finality.v1",
		CheckpointLeaf: rec.MMRPosition,
		ProofSystem:   "hashchain-v0",
		Attestations: []types.VerifierAttestation{
			{VerifierID: "v1", LeafIndex: rec.MMRPosition, Approved: true},
		},
	}
	out, err := o.SubmitFinality(fin)
	if err != nil {
		t.Fatalf("SubmitFinality: %v", err)
	}
	if out.Status != types.CheckpointFinal {
		t.Fatalf("status = %s, want final", out.Status)
	}
	if out.FinalityLeaf == nil {
		t.Fatal("FinalityLeaf index not recorded")
	}
	rootAfter := o.MMRRoot()
	if rootAfter == rootBefore {
		t.Fatal("MMR should advance after a finality leaf")
	}
	// Inclusion proof of the finality leaf must verify against the new root.
	if _, err := o.MMRProof(*out.FinalityLeaf); err != nil {
		t.Fatalf("MMRProof(finalityLeaf): %v", err)
	}
}

func TestFinalityQuorumEnforced(t *testing.T) {
	o := newPhase4Oracle(t)
	for _, v := range []string{"v1", "v2", "v3"} {
		if err := o.RegisterVerifier(v); err != nil {
			t.Fatalf("RegisterVerifier: %v", err)
		}
	}
	if got := o.VerifierCount(); got != 3 {
		t.Fatalf("VerifierCount = %d, want 3", got)
	}
	rec := projectTestRollup(t, o, "knirvchain-q")

	// 1 of 3 approvals = below the strict 2/3 quorum -> must fail.
	_, err := o.SubmitFinality(&types.FinalityRecord{
		CheckpointLeaf: rec.MMRPosition,
		ProofSystem:   "hashchain-v0",
		Attestations: []types.VerifierAttestation{
			{VerifierID: "v1", LeafIndex: rec.MMRPosition, Approved: true},
		},
	})
	if err == nil {
		t.Fatal("expected quorum failure with 1/3 approvals")
	}

	// 2 of 3 approvals -> quorum met -> success.
	_, err = o.SubmitFinality(&types.FinalityRecord{
		CheckpointLeaf: rec.MMRPosition,
		ProofSystem:   "hashchain-v0",
		Attestations: []types.VerifierAttestation{
			{VerifierID: "v1", LeafIndex: rec.MMRPosition, Approved: true},
			{VerifierID: "v2", LeafIndex: rec.MMRPosition, Approved: true},
		},
	})
	if err != nil {
		t.Fatalf("expected quorum success with 2/3 approvals: %v", err)
	}
}

func TestFinalityAttestationAccumulation(t *testing.T) {
	o := newPhase4Oracle(t)
	for _, v := range []string{"v1", "v2", "v3"} {
		if err := o.RegisterVerifier(v); err != nil {
			t.Fatalf("RegisterVerifier: %v", err)
		}
	}
	rec := projectTestRollup(t, o, "knirvchain-acc")

	// First attestation alone should not finalize.
	if _, err := o.SubmitAttestation("knirvchain-acc", rec.Checkpoint.StartHeight, types.VerifierAttestation{
		VerifierID: "v1", LeafIndex: rec.MMRPosition, Approved: true,
	}); err != nil {
		t.Fatalf("SubmitAttestation(1): %v", err)
	}
	if got := o.GetCheckpointRecords("knirvchain-acc")[0].Status; got != types.CheckpointProvisional {
		t.Fatalf("status after 1 attestation = %s, want provisional", got)
	}

	// Second attestation reaches quorum and finalizes.
	if _, err := o.SubmitAttestation("knirvchain-acc", rec.Checkpoint.StartHeight, types.VerifierAttestation{
		VerifierID: "v2", LeafIndex: rec.MMRPosition, Approved: true,
	}); err != nil {
		t.Fatalf("SubmitAttestation(2): %v", err)
	}
	if got := o.GetCheckpointRecords("knirvchain-acc")[0].Status; got != types.CheckpointFinal {
		t.Fatalf("status after 2 attestations = %s, want final", got)
	}
}

func TestFinalityWindowMissSweeper(t *testing.T) {
	o := newPhase4Oracle(t)
	rec := projectTestRollup(t, o, "knirvchain-sweep")
	// Force a window miss: oracle height (1) now exceeds the finality deadline.
	rec.FinalByHeight = 0
	rootBefore := o.MMRRoot()
	o.sweepOnce()
	rootAfter := o.MMRRoot()
	if rootAfter == rootBefore {
		t.Fatal("sweeper should append a rejection leaf")
	}
	got := o.GetCheckpointRecords("knirvchain-sweep")[0]
	if got.Status != types.CheckpointRejected {
		t.Fatalf("status = %s, want rejected", got.Status)
	}
	if got.RejectionLeaf == nil {
		t.Fatal("RejectionLeaf index not recorded")
	}
}
