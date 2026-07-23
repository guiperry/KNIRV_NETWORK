package oracle

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
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
		verifiers:       make(map[string][]byte),
		logger:          zap.NewNop(),
	}
}

var phase4VerifierKeys sync.Map // map[*Oracle]map[string]ed25519.PrivateKey

func registerPhase4Verifier(t *testing.T, o *Oracle, id string) {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	if err := o.RegisterVerifier(id, publicKey); err != nil {
		t.Fatalf("RegisterVerifier: %v", err)
	}
	value, _ := phase4VerifierKeys.LoadOrStore(o, make(map[string]ed25519.PrivateKey))
	value.(map[string]ed25519.PrivateKey)[id] = privateKey
}

func phase4Attestation(t *testing.T, o *Oracle, id string, rec *types.CheckpointRecord) types.VerifierAttestation {
	t.Helper()
	value, ok := phase4VerifierKeys.Load(o)
	if !ok {
		t.Fatalf("no verifier keys for oracle")
	}
	key := value.(map[string]ed25519.PrivateKey)[id]
	att := types.VerifierAttestation{VerifierID: id, LeafIndex: rec.MMRPosition, Approved: true}
	att.Signature = ed25519.Sign(key, types.AttestationMessage(att.LeafIndex, rec.LeafHash, true))
	return att
}

// projectTestRollup registers a fresh chain and submits a signed rollup for
// it. Use projectTestRollupAs when the caller already registered chainID
// with its own key (e.g. to set Bond/BondOwner) — registering twice fails.
func projectTestRollup(t *testing.T, o *Oracle, chainID string) *types.CheckpointRecord {
	t.Helper()
	reg, key := newTestChain(t, chainID)
	if err := types.SignRegistration(&reg, key); err != nil {
		t.Fatalf("sign registration: %v", err)
	}
	if err := o.RegisterChain(&reg); err != nil {
		t.Fatalf("register: %v", err)
	}
	return projectTestRollupAs(t, o, chainID, reg.Authors[0].Address, reg.Authors[0].PubKey, key)
}

// projectTestRollupAs submits a signed rollup for a chain/author the caller
// already registered.
func projectTestRollupAs(t *testing.T, o *Oracle, chainID, proposer string, proposerPubKey []byte, key *ecdsa.PrivateKey) *types.CheckpointRecord {
	t.Helper()
	rr := &types.RollupRecord{
		ID:          "r-" + chainID,
		BatchRoot:   testBatchRoot,
		ChainID:     chainID,
		StartHeight: 1,
		EndHeight:   10,
		Proposer:    proposer,
		Status:      types.RollupStatusSubmitted,
		SubmittedAt: time.Now().UTC(),
	}
	sigBytes, err := types.SignMessage(key, fmt.Sprintf("%x", rr.Digest()))
	if err != nil {
		t.Fatalf("sign rollup: %v", err)
	}
	rr.Signatures = []types.AuthorSig{{
		Address:   proposer,
		PubKeyHex: hex.EncodeToString(proposerPubKey),
		Signature: sigBytes,
	}}
	if err := o.SubmitRollup(rr); err != nil {
		t.Fatalf("SubmitRollup: %v", err)
	}
	recs := o.GetCheckpointRecords(chainID)
	if len(recs) != 1 {
		t.Fatalf("want 1 projected record, got %d", len(recs))
	}
	return recs[0]
}

func admitTestCheckpoint(t *testing.T, o *Oracle, chainID string) *types.CheckpointRecord {
	t.Helper()
	reg, key := newTestChain(t, chainID)
	if err := types.SignRegistration(&reg, key); err != nil {
		t.Fatal(err)
	}
	if err := o.RegisterChain(&reg); err != nil {
		t.Fatalf("RegisterChain: %v", err)
	}
	rootBytes, _ := hex.DecodeString(testBatchRoot)
	var root [32]byte
	copy(root[:], rootBytes)
	cp := &types.Checkpoint{SchemaVersion: "knirv.checkpoint.v1", ChainID: chainID, StartHeight: 1, EndHeight: 10, Root: root, Proposer: reg.Authors[0].Address}
	if err := types.SignCheckpoint(cp, key); err != nil {
		t.Fatal(err)
	}
	rec, err := o.SubmitCheckpoint(cp)
	if err != nil {
		t.Fatalf("SubmitCheckpoint: %v", err)
	}
	return rec
}

func TestFinalityDegradedAdmission(t *testing.T) {
	o := newPhase4Oracle(t)
	rec := admitTestCheckpoint(t, o, "knirvchain-f")
	registerPhase4Verifier(t, o, "v1")
	rootBefore := o.MMRRoot()

	fin := &types.FinalityRecord{
		SchemaVersion:  "knirv.finality.v1",
		CheckpointLeaf: rec.MMRPosition,
		ProofSystem:    "hashchain-v0",
		Attestations:   []types.VerifierAttestation{phase4Attestation(t, o, "v1", rec)},
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
		registerPhase4Verifier(t, o, v)
	}
	if got := o.VerifierCount(); got != 3 {
		t.Fatalf("VerifierCount = %d, want 3", got)
	}
	rec := admitTestCheckpoint(t, o, "knirvchain-q")

	// 1 of 3 approvals = below the strict 2/3 quorum -> must fail.
	_, err := o.SubmitFinality(&types.FinalityRecord{
		CheckpointLeaf: rec.MMRPosition,
		ProofSystem:    "hashchain-v0",
		Attestations:   []types.VerifierAttestation{phase4Attestation(t, o, "v1", rec)},
	})
	if err == nil {
		t.Fatal("expected quorum failure with 1/3 approvals")
	}

	// 2 of 3 approvals -> quorum met -> success.
	_, err = o.SubmitFinality(&types.FinalityRecord{
		CheckpointLeaf: rec.MMRPosition,
		ProofSystem:    "hashchain-v0",
		Attestations: []types.VerifierAttestation{
			phase4Attestation(t, o, "v1", rec), phase4Attestation(t, o, "v2", rec),
		},
	})
	if err != nil {
		t.Fatalf("expected quorum success with 2/3 approvals: %v", err)
	}
}

func TestFinalityAttestationAccumulation(t *testing.T) {
	o := newPhase4Oracle(t)
	for _, v := range []string{"v1", "v2", "v3"} {
		registerPhase4Verifier(t, o, v)
	}
	rec := admitTestCheckpoint(t, o, "knirvchain-acc")

	// First attestation alone should not finalize.
	if _, err := o.SubmitAttestation("knirvchain-acc", rec.Checkpoint.StartHeight, phase4Attestation(t, o, "v1", rec)); err != nil {
		t.Fatalf("SubmitAttestation(1): %v", err)
	}
	if got := o.GetCheckpointRecords("knirvchain-acc")[0].Status; got != types.CheckpointProvisional {
		t.Fatalf("status after 1 attestation = %s, want provisional", got)
	}

	// Second attestation reaches quorum and finalizes.
	if _, err := o.SubmitAttestation("knirvchain-acc", rec.Checkpoint.StartHeight, phase4Attestation(t, o, "v2", rec)); err != nil {
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

func TestNonValidatorClockExpiresRealProofWindow(t *testing.T) {
	engine := consensus.NewConsensusEngine("clocked-oracle", 10*time.Millisecond, false, t.TempDir(), zap.NewNop())
	o := &Oracle{checkpoint: newCheckpointState(t.TempDir()), rollups: make(map[string]*types.RollupRecord), consensusEngine: engine, verifiers: make(map[string][]byte), logger: zap.NewNop()}
	if err := engine.Start(); err != nil {
		t.Fatal(err)
	}
	defer engine.Stop()
	rec := projectTestRollup(t, o, "clock-window")
	rec.FinalByHeight = uint64(engine.GetHeight()) + 1
	deadline := time.Now().Add(time.Second)
	for uint64(engine.GetHeight()) <= rec.FinalByHeight && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if uint64(engine.GetHeight()) <= rec.FinalByHeight {
		t.Fatal("non-validator Oracle height did not advance")
	}
	o.sweepOnce()
	if rec.Status != types.CheckpointRejected {
		t.Fatalf("expired checkpoint status = %s", rec.Status)
	}
}

// --- Phase 5: Oracle-side transition-proof verification (§3.3f step 4) ---

func marshalTransitionProof(t *testing.T, tp types.TransitionProof) []byte {
	t.Helper()
	b, err := json.Marshal(tp)
	if err != nil {
		t.Fatalf("marshal transition proof: %v", err)
	}
	return b
}

func registerTestChain(t *testing.T, o *Oracle, chainID, proofSystem string, vk []byte) {
	t.Helper()
	if _, ok := o.GetChainRegistration(chainID); !ok {
		reg, key := newTestChain(t, chainID)
		if err := types.SignRegistration(&reg, key); err != nil {
			t.Fatal(err)
		}
		if err := o.RegisterChain(&reg); err != nil {
			t.Fatalf("RegisterChain: %v", err)
		}
	}
	if err := o.SetChainVerificationKey(chainID, proofSystem, vk); err != nil {
		t.Fatalf("SetChainVerificationKey: %v", err)
	}
}

func TestFinalityHashchainV0Verify(t *testing.T) {
	o := newPhase4Oracle(t)
	rec := admitTestCheckpoint(t, o, "knirvchain-v5")
	registerPhase4Verifier(t, o, "v1")
	// Valid hashchain-v0 proof binds PostRoot to the anchored checkpoint root.
	tp := types.TransitionProof{
		SchemaVersion: "knirv.transition-proof.v1",
		ChainID:       rec.Checkpoint.ChainID,
		StartHeight:   rec.Checkpoint.StartHeight,
		EndHeight:     rec.Checkpoint.EndHeight,
		PreRoot:       rec.Checkpoint.PrevCheckHash, // zero for rollup projections
		PostRoot:      rec.Checkpoint.Root,
		ProofSystem:   "hashchain-v0",
		Proof:         []byte("re-execution-trace-hash"),
	}
	fin := &types.FinalityRecord{
		SchemaVersion:   "knirv.finality.v1",
		CheckpointLeaf:  rec.MMRPosition,
		ProofSystem:     "hashchain-v0",
		TransitionProof: marshalTransitionProof(t, tp),
		Attestations:    []types.VerifierAttestation{phase4Attestation(t, o, "v1", rec)},
	}
	out, err := o.SubmitFinality(fin)
	if err != nil {
		t.Fatalf("SubmitFinality with valid hashchain-v0 proof: %v", err)
	}
	if out.Status != types.CheckpointFinal {
		t.Fatalf("status = %s, want final", out.Status)
	}
}

func TestFinalityTamperedProofRejected(t *testing.T) {
	o := newPhase4Oracle(t)
	rec := admitTestCheckpoint(t, o, "knirvchain-tamper")
	registerPhase4Verifier(t, o, "v1")
	// Tampered proof: PostRoot does not match the anchored checkpoint root.
	tp := types.TransitionProof{
		SchemaVersion: "knirv.transition-proof.v1",
		ChainID:       rec.Checkpoint.ChainID,
		StartHeight:   rec.Checkpoint.StartHeight,
		EndHeight:     rec.Checkpoint.EndHeight,
		PostRoot:      [32]byte{0xde, 0xad}, // wrong
		ProofSystem:   "hashchain-v0",
	}
	fin := &types.FinalityRecord{
		CheckpointLeaf:  rec.MMRPosition,
		ProofSystem:     "hashchain-v0",
		TransitionProof: marshalTransitionProof(t, tp),
		Attestations:    []types.VerifierAttestation{phase4Attestation(t, o, "v1", rec)},
	}
	if _, err := o.SubmitFinality(fin); err == nil {
		t.Fatal("expected rejection of tampered transition proof")
	}
	if got := o.GetCheckpointRecords("knirvchain-tamper")[0].Status; got != types.CheckpointRejected {
		t.Fatalf("status = %s, want rejected tombstone", got)
	}
}

func TestFinalitySNARKRequiresEnrolledKey(t *testing.T) {
	o := newPhase4Oracle(t)
	registerPhase4Verifier(t, o, "v1")

	// (a) No verification key enrolled for the chain -> SNARK proof rejected.
	rec := admitTestCheckpoint(t, o, "knirvchain-snark")
	tp := types.TransitionProof{
		SchemaVersion: "knirv.transition-proof.v1",
		ChainID:       rec.Checkpoint.ChainID,
		StartHeight:   rec.Checkpoint.StartHeight,
		EndHeight:     rec.Checkpoint.EndHeight,
		PostRoot:      rec.Checkpoint.Root,
		ProofSystem:   "plonk",
		Proof:         []byte("snark-bytes"),
		PublicInputs:  []byte("{}"),
	}
	_, err := o.SubmitFinality(&types.FinalityRecord{
		CheckpointLeaf:  rec.MMRPosition,
		ProofSystem:     "plonk",
		TransitionProof: marshalTransitionProof(t, tp),
		Attestations:    []types.VerifierAttestation{phase4Attestation(t, o, "v1", rec)},
	})
	if err == nil {
		t.Fatal("expected rejection of plonk proof without enrolled verification key")
	}

	// (b) Enroll a verification key -> plonk proof accepted (delegated verify).
	rec2 := admitTestCheckpoint(t, o, "knirvchain-snark2")
	registerTestChain(t, o, "knirvchain-snark2", "plonk", []byte("fakevk-bytes"))
	tp2 := types.TransitionProof{
		SchemaVersion: "knirv.transition-proof.v1",
		ChainID:       rec2.Checkpoint.ChainID,
		StartHeight:   rec2.Checkpoint.StartHeight,
		EndHeight:     rec2.Checkpoint.EndHeight,
		PostRoot:      rec2.Checkpoint.Root,
		ProofSystem:   "plonk",
		Proof:         []byte("snark-bytes"),
		PublicInputs:  []byte("{}"),
	}
	out, err := o.SubmitFinality(&types.FinalityRecord{
		CheckpointLeaf:  rec2.MMRPosition,
		ProofSystem:     "plonk",
		TransitionProof: marshalTransitionProof(t, tp2),
		Attestations:    []types.VerifierAttestation{phase4Attestation(t, o, "v1", rec2)},
	})
	if err != nil {
		t.Fatalf("SubmitFinality with enrolled plonk key: %v", err)
	}
	if out.Status != types.CheckpointFinal {
		t.Fatalf("status = %s, want final", out.Status)
	}
}
