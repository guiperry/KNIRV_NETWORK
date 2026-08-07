package oracle

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"time"

	knirvsigning "github.com/KNIRV/KNIRV_NETWORK/KNIRVSDK/go/signing"
	"github.com/knirvcorp/knirvoracle/internal/oracle/consensus"
	"github.com/knirvcorp/knirvoracle/internal/oracle/mmr"
	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
	"go.uber.org/zap"
)

// VerifierQuorumNumer/Denom define the attestation threshold: ≥ Numer/Denom of
// registered verifier nodes must approve. Defaults to 2/3 (merkle-math.md §3.3f).
const (
	DefaultVerifierQuorumNumer  = 2
	DefaultVerifierQuorumDenom  = 3
	DefaultCheckpointSlashNumer = 1
	DefaultCheckpointSlashDenom = 10
)

// loadVerifiers reads the persisted verifier registry from disk.
func (o *Oracle) loadVerifiers() error {
	if o.verifiersPath == "" {
		return nil
	}
	data, err := os.ReadFile(o.verifiersPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var keys map[string][]byte
	if err := json.Unmarshal(data, &keys); err != nil {
		var ids []string
		if legacyErr := json.Unmarshal(data, &ids); legacyErr != nil {
			return fmt.Errorf("decode verifiers: %w", err)
		}
		keys = make(map[string][]byte, len(ids))
		for _, id := range ids {
			keys[id] = nil
		}
	}
	for id, key := range keys {
		if len(key) != 33 {
			return fmt.Errorf("verifier %q uses a legacy/non-secp256k1 key; re-register it with a canonical KNIRV service key", id)
		}
		address, err := knirvsigning.Address(key, knirvsigning.DefaultAddressPrefix)
		if err != nil || address != id {
			return fmt.Errorf("verifier %q public key does not match its canonical KNIRV address", id)
		}
		o.verifiers[id] = append([]byte(nil), key...)
	}
	return nil
}

// persistVerifiersLocked writes the verifier registry; caller holds verifiersMu.
func (o *Oracle) persistVerifiersLocked() error {
	if o.verifiersPath == "" {
		return nil
	}
	payload, err := json.MarshalIndent(o.verifiers, "", "  ")
	if err != nil {
		return err
	}
	tmp := o.verifiersPath + ".tmp"
	if err := os.WriteFile(tmp, payload, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, o.verifiersPath)
}

// RegisterVerifier adds a validation-chain node to the attestation quorum set.
func (o *Oracle) RegisterVerifier(id string, publicKey ...[]byte) error {
	if id == "" {
		return fmt.Errorf("verifier id required")
	}
	var key []byte
	if len(publicKey) > 0 {
		key = publicKey[0]
	}
	if len(key) != 33 {
		return fmt.Errorf("verifier secp256k1 public key must be 33 compressed bytes")
	}
	address, err := knirvsigning.Address(key, knirvsigning.DefaultAddressPrefix)
	if err != nil {
		return fmt.Errorf("derive verifier address: %w", err)
	}
	if id != address {
		return fmt.Errorf("verifier_id must be the canonical service-key address %s", address)
	}
	o.verifiersMu.Lock()
	defer o.verifiersMu.Unlock()
	o.verifiers[id] = append([]byte(nil), key...)
	return o.persistVerifiersLocked()
}

func (o *Oracle) verifyAttestations(cp *types.CheckpointRecord, attestations []types.VerifierAttestation) (int, error) {
	o.verifiersMu.RLock()
	defer o.verifiersMu.RUnlock()
	if len(o.verifiers) == 0 {
		return 0, fmt.Errorf("no verifier nodes registered")
	}
	seen := make(map[string]bool)
	approved := 0
	for _, att := range attestations {
		if seen[att.VerifierID] {
			continue
		}
		key, ok := o.verifiers[att.VerifierID]
		if !ok {
			return 0, fmt.Errorf("unregistered verifier %q", att.VerifierID)
		}
		if att.LeafIndex != cp.MMRPosition {
			return 0, fmt.Errorf("attestation leaf mismatch")
		}
		message := types.AttestationMessage(att.LeafIndex, cp.LeafHash, att.Approved)
		signed := knirvsigning.SignedMessage{Envelope: att.Envelope, Signature: att.Signature, PublicKey: key, Address: att.VerifierID}
		if err := knirvsigning.VerifyMessagePayload(signed, "knirv.oracle", "verifier-attestation", cp.Checkpoint.ChainID, message, time.Now()); err != nil {
			return 0, fmt.Errorf("invalid attestation signature from %q", att.VerifierID)
		}
		seen[att.VerifierID] = true
		if att.Approved {
			approved++
		}
	}
	return approved, nil
}

// VerifierCount returns the number of registered verifier nodes.
func (o *Oracle) VerifierCount() int {
	o.verifiersMu.RLock()
	defer o.verifiersMu.RUnlock()
	return len(o.verifiers)
}

// attestationQuorum reports whether `approvals` distinct approvals satisfy the
// ≥2/3-of-registered-verifiers threshold. When no verifiers are registered
// (single-node / optimistic bootstrap) any single attestation is accepted.
func (o *Oracle) attestationQuorum(approvals int) bool {
	o.verifiersMu.RLock()
	defer o.verifiersMu.RUnlock()
	total := len(o.verifiers)
	if total == 0 {
		return approvals >= 1
	}
	need := (total*DefaultVerifierQuorumNumer + DefaultVerifierQuorumDenom - 1) / DefaultVerifierQuorumDenom
	return approvals >= need
}

// recordByPosition returns the checkpoint record carrying the given MMR leaf
// position, or nil. Caller holds o.checkpoint.mu (read or write).
func (o *Oracle) recordByPosition(pos uint64) *types.CheckpointRecord {
	for _, rec := range o.checkpoint.records {
		if rec != nil && rec.MMRPosition == pos {
			return rec
		}
	}
	return nil
}

// verifyTransitionProof is the Oracle-side cheap verifier (merkle-math.md Phase 5,
// §3.3f step 4). The full SNARK pairing check and hashchain-v0 re-execution are
// delegated to the verifier chains; the Oracle enforces (a) the proof's public
// inputs bind to the anchored checkpoint (PostRoot == checkpoint.Root, PreRoot ==
// checkpoint.PrevCheckHash when present) and (b) SNARK systems require an enrolled
// verification key for the chain. A tampered or unbound proof is rejected here, so
// SubmitFinality never appends a LeafFinality for it.
func (o *Oracle) verifyTransitionProof(rec *types.FinalityRecord, cp *types.CheckpointRecord) error {
	sys := rec.ProofSystem
	if sys == "" {
		sys = "hashchain-v0"
	}

	bind := func(tp types.TransitionProof) error {
		if tp.SchemaVersion != "knirv.transition-proof.v1" || tp.ChainID != cp.Checkpoint.ChainID {
			return fmt.Errorf("transition proof schema/chain mismatch")
		}
		if tp.StartHeight != cp.Checkpoint.StartHeight || tp.EndHeight != cp.Checkpoint.EndHeight {
			return fmt.Errorf("transition proof height range does not match checkpoint")
		}
		if tp.PostRoot != cp.Checkpoint.Root {
			return fmt.Errorf("post_root %x != checkpoint root %x", tp.PostRoot, cp.Checkpoint.Root)
		}
		expectedPreRoot := [32]byte{}
		if cp.Checkpoint.StartHeight > 1 {
			for _, candidate := range o.checkpoint.records {
				if candidate != nil && candidate.Checkpoint.ChainID == cp.Checkpoint.ChainID && candidate.Checkpoint.EndHeight+1 == cp.Checkpoint.StartHeight {
					expectedPreRoot = candidate.Checkpoint.Root
					break
				}
			}
		}
		if tp.PreRoot != expectedPreRoot {
			return fmt.Errorf("pre_root %x != preceding accumulator %x", tp.PreRoot, expectedPreRoot)
		}
		return nil
	}

	switch sys {
	case "hashchain-v0":
		if len(rec.TransitionProof) == 0 {
			if reg, ok := o.checkpoint.registry.Get(cp.Checkpoint.ChainID); ok && reg.PreferredProofSystem == "plonk" {
				return fmt.Errorf("plonk-enabled chain requires an attested hashchain-v0 fallback proof")
			}
			// No proof supplied: rely on the attestation quorum (optimistic).
			o.logger.Warn("finality submitted without transition proof (hashchain-v0 optimistic)")
			return nil
		}
		var tp types.TransitionProof
		if err := json.Unmarshal(rec.TransitionProof, &tp); err != nil {
			return fmt.Errorf("decode transition proof: %w", err)
		}
		return bind(tp)

	case "groth16", "plonk":
		// SNARK verification is delegated to the verifier chains. The Oracle
		// enforces key enrollment + public-input binding so a proof cannot be
		// accepted without a registered verifier key for the chain.
		reg, ok := o.checkpoint.registry.Get(cp.Checkpoint.ChainID)
		if !ok || len(reg.VerificationKey) == 0 {
			return fmt.Errorf("no verification key registered for chain %s; cannot accept %s proof", cp.Checkpoint.ChainID, sys)
		}
		if reg.PreferredProofSystem != "" && reg.PreferredProofSystem != sys {
			return fmt.Errorf("proof system %q does not match registered %q", sys, reg.PreferredProofSystem)
		}
		if len(rec.TransitionProof) == 0 {
			return fmt.Errorf("snark proof required but transition_proof is empty")
		}
		var tp types.TransitionProof
		if err := json.Unmarshal(rec.TransitionProof, &tp); err != nil {
			return fmt.Errorf("decode transition proof: %w", err)
		}
		if tp.ProofSystem != sys || len(tp.Proof) == 0 || len(tp.PublicInputs) == 0 {
			return fmt.Errorf("incomplete %s proof artifacts", sys)
		}
		return bind(tp)

	default:
		return fmt.Errorf("unsupported proof system %q", sys)
	}
}

// SubmitFinality admits a finality record: the checkpoint leaf must exist and be
// provisional, be within its proof window, and carry an attestation quorum
// (merkle-math.md §3.3f). On success it appends a LeafFinality MMR leaf and
// flips the indexed record to final. The transition proof is delegated to the
// verifier chains (Phase 5 adds Oracle-side SNARK verification); here we trust
// the attestation quorum, which is exactly the optimistic split.
func (o *Oracle) SubmitFinality(rec *types.FinalityRecord) (*types.CheckpointRecord, error) {
	if rec == nil {
		return nil, fmt.Errorf("finality record required")
	}
	o.checkpoint.mu.Lock()
	defer o.checkpoint.mu.Unlock()

	cp := o.recordByPosition(rec.CheckpointLeaf)
	if cp == nil {
		return nil, fmt.Errorf("no checkpoint at MMR position %d", rec.CheckpointLeaf)
	}
	if cp.Source != "" {
		return nil, fmt.Errorf("projected checkpoint %d from %q cannot be finalized", rec.CheckpointLeaf, cp.Source)
	}
	// Tolerate a client omitting the hash: default it from the anchored record.
	if rec.CheckpointHash == ([32]byte{}) {
		rec.CheckpointHash = cp.LeafHash
	}
	if cp.Status != types.CheckpointProvisional {
		return nil, fmt.Errorf("checkpoint %d is %s, not provisional", rec.CheckpointLeaf, cp.Status)
	}
	// Window check: the wall clock must still be within the proof window —
	// this no longer depends on the Oracle's own (heartbeat-only) block
	// height, so it is correct even while that ticker is disabled.
	if now := time.Now(); now.After(cp.FinalBy) {
		return nil, fmt.Errorf("checkpoint %d missed its proof window (now %s > %s)", rec.CheckpointLeaf, now.Format(time.RFC3339), cp.FinalBy.Format(time.RFC3339))
	}

	approved, err := o.verifyAttestations(cp, rec.Attestations)
	if err != nil {
		return nil, err
	}
	if !o.attestationQuorum(approved) {
		return nil, fmt.Errorf("attestation quorum not met: %d approved", approved)
	}

	// Phase 5: the Oracle runs the cheap verifier itself (merkle-math.md §3.3f
	// step 4). hashchain-v0 → bind proof roots to the anchored checkpoint; SNARK
	// systems → require an enrolled verification key + binding, with the actual
	// pairing check delegated to the verifier chains.
	if err := o.verifyTransitionProof(rec, cp); err != nil {
		if rejectErr := o.rejectCheckpointLocked(cp, "proof-fail: "+err.Error()); rejectErr != nil {
			return nil, fmt.Errorf("transition proof rejected: %v (record rejection failed: %v)", err, rejectErr)
		}
		if commitErr := o.commitAuditMMR(); commitErr != nil {
			return nil, commitErr
		}
		return nil, fmt.Errorf("transition proof rejected: %w", err)
	}

	// Build + append the LeafFinality payload.
	leafData, err := json.Marshal(types.FinalityLeaf{
		Kind:           byte(mmr.LeafFinality),
		CheckpointLeaf: rec.CheckpointLeaf,
		CheckpointHash: fmt.Sprintf("%x", rec.CheckpointHash),
		ProofSystem:    rec.ProofSystem,
		Attestations:   rec.Attestations,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal finality leaf: %w", err)
	}
	pos, _ := o.appendAuditLeafLocked(leafData)

	cp.Status = types.CheckpointFinal
	fpos := pos
	cp.FinalityLeaf = &fpos
	cp.PendingAttestations = rec.Attestations

	if err := o.persistCheckpointLocked(); err != nil {
		return nil, err
	}
	if err := o.commitAuditMMR(); err != nil {
		return nil, err
	}
	o.logger.Info("checkpoint finalized",
		zap.Uint64("checkpoint_leaf", rec.CheckpointLeaf),
		zap.Uint64("finality_leaf", pos),
		zap.String("proof_system", rec.ProofSystem),
	)
	return cp, nil
}

// SubmitAttestation records a single validation-chain attestation for a
// provisional checkpoint. Once the attestation quorum is reached the record is
// finalized automatically (merkle-math.md §3.3f step 3).
func (o *Oracle) SubmitAttestation(chainID string, startHeight uint64, att types.VerifierAttestation) (*types.CheckpointRecord, error) {
	return o.SubmitProofAttestation(chainID, startHeight, nil, "", att)
}

// SubmitProofAttestation accumulates independently signed verifier approvals
// for one exact transition proof. A different proof cannot reuse earlier
// attestations; quorum promotes the checkpoint through SubmitFinality.
func (o *Oracle) SubmitProofAttestation(chainID string, startHeight uint64, transitionProof []byte, proofSystem string, att types.VerifierAttestation) (*types.CheckpointRecord, error) {
	o.checkpoint.mu.Lock()
	defer o.checkpoint.mu.Unlock()

	rec, ok := o.checkpoint.records[recordKey(chainID, startHeight)]
	if !ok {
		return nil, fmt.Errorf("no checkpoint for %s/%d", chainID, startHeight)
	}
	if rec.Source != "" {
		return nil, fmt.Errorf("projected checkpoint %s/%d from %q cannot be finalized", chainID, startHeight, rec.Source)
	}
	if rec.Status != types.CheckpointProvisional {
		return nil, fmt.Errorf("checkpoint %s/%d is %s", chainID, startHeight, rec.Status)
	}
	if _, err := o.verifyAttestations(rec, []types.VerifierAttestation{att}); err != nil {
		return nil, err
	}
	if len(transitionProof) > 0 {
		if len(rec.PendingTransitionProof) > 0 && !bytes.Equal(rec.PendingTransitionProof, transitionProof) {
			return nil, fmt.Errorf("attestation targets a different transition proof")
		}
		if rec.PendingProofSystem != "" && rec.PendingProofSystem != proofSystem {
			return nil, fmt.Errorf("attestation proof system mismatch")
		}
		rec.PendingTransitionProof = append([]byte(nil), transitionProof...)
		rec.PendingProofSystem = proofSystem
	}
	// De-duplicate by verifier id.
	for _, a := range rec.PendingAttestations {
		if a.VerifierID == att.VerifierID {
			return rec, nil
		}
	}
	rec.PendingAttestations = append(rec.PendingAttestations, att)

	approved, err := o.verifyAttestations(rec, rec.PendingAttestations)
	if err != nil {
		return nil, err
	}
	if !o.attestationQuorum(approved) {
		if err := o.persistCheckpointLocked(); err != nil {
			return nil, err
		}
		return rec, nil
	}

	// Quorum reached → finalize via the same path as a one-shot SubmitFinality.
	fin := &types.FinalityRecord{
		SchemaVersion:   "knirv.finality.v1",
		CheckpointLeaf:  rec.MMRPosition,
		CheckpointHash:  rec.LeafHash,
		TransitionProof: rec.PendingTransitionProof,
		ProofSystem:     rec.PendingProofSystem,
		Attestations:    rec.PendingAttestations,
	}
	if fin.ProofSystem == "" {
		fin.ProofSystem = "hashchain-v0"
	}
	// Relock: SubmitFinality takes the write lock itself.
	o.checkpoint.mu.Unlock()
	defer o.checkpoint.mu.Lock()
	return o.SubmitFinality(fin)
}

// sweepExpired finalizes the window-miss policy: any provisional record whose
// proof window has elapsed (wall-clock, see CheckpointRecord.FinalBy) without
// a finality leaf is tombstoned with a LeafRejection (merkle-math.md §3.3e).
// This runs on its own wall-clock ticker (see sweepLoop) and no longer
// depends on the Oracle's consensus height at all, so it works correctly
// whether or not that heartbeat is enabled. Caller holds o.checkpoint.mu.
// Returns the number of records rejected.
func (o *Oracle) sweepExpired() (int, error) {
	now := time.Now()
	rejected := 0
	for _, rec := range o.checkpoint.records {
		if rec == nil || rec.Status != types.CheckpointProvisional {
			continue
		}
		if now.After(rec.FinalBy) {
			if err := o.rejectCheckpointLocked(rec, "window-miss"); err != nil {
				return rejected, err
			}
			rejected++
			o.logger.Warn("checkpoint rejected (window miss)",
				zap.Uint64("checkpoint_leaf", rec.MMRPosition),
				zap.Time("now", now),
				zap.Time("final_by", rec.FinalBy),
			)
		}
	}
	if rejected > 0 {
		if err := o.persistCheckpointLocked(); err != nil {
			return rejected, err
		}
	}
	return rejected, nil
}

// rejectCheckpointLocked applies the economics policy, appends the resulting
// evidence-bearing tombstone, and queues consensus evidence. Caller holds
// checkpoint.mu.
func (o *Oracle) rejectCheckpointLocked(rec *types.CheckpointRecord, reason string) error {
	if rec == nil || rec.Status != types.CheckpointProvisional {
		return fmt.Errorf("checkpoint is not provisional")
	}
	height := uint64(0)
	if o.consensusEngine != nil {
		height = uint64(o.consensusEngine.GetHeight())
	}
	var slashEvidence *types.SlashingEvidence
	if reg, ok := o.checkpoint.registry.Get(rec.Checkpoint.ChainID); ok && reg.BondRemaining > 0 && o.economicsEngine != nil {
		requested := (reg.BondRemaining*DefaultCheckpointSlashNumer + DefaultCheckpointSlashDenom - 1) / DefaultCheckpointSlashDenom
		bond, actual, err := o.economicsEngine.SlashChainBond(rec.Checkpoint.ChainID, reason, new(big.Int).SetUint64(requested))
		if err != nil {
			return fmt.Errorf("slash chain bond: %w", err)
		}
		amount := actual.Uint64()
		if err := o.checkpoint.registry.ApplySlash(rec.Checkpoint.ChainID, amount); err != nil {
			return err
		}
		slashEvidence = &types.SlashingEvidence{
			SchemaVersion: "knirv.slashing-evidence.v1", ChainID: rec.Checkpoint.ChainID,
			BondOwner: bond.Owner.String(), CheckpointLeaf: rec.MMRPosition,
			Reason: reason, Amount: amount, OracleHeight: height, OccurredAt: time.Now().UTC(),
		}
	}

	leafData, err := json.Marshal(types.RejectionLeaf{
		Kind: byte(mmr.LeafRejection), CheckpointLeaf: rec.MMRPosition,
		Reason: reason, SlashingEvidence: slashEvidence,
	})
	if err != nil {
		return err
	}
	pos, _ := o.appendAuditLeafLocked(leafData)
	if o.consensusEngine != nil {
		if slashEvidence != nil {
			_ = o.consensusEngine.AddEvidence(consensus.CheckpointSlashingEvidence{
				SchemaVersion: slashEvidence.SchemaVersion, ChainID: slashEvidence.ChainID,
				BondOwner: slashEvidence.BondOwner, CheckpointLeaf: slashEvidence.CheckpointLeaf,
				Reason: slashEvidence.Reason, Amount: slashEvidence.Amount,
				OracleHeight: slashEvidence.OracleHeight, OccurredAt: slashEvidence.OccurredAt,
			})
		}
	}
	rec.Status = types.CheckpointRejected
	leafIndex := uint64(pos)
	rec.RejectionLeaf = &leafIndex
	if err := o.persistRegistry(); err != nil {
		return err
	}
	return o.persistCheckpointLocked()
}

// sweepOnce runs the window-miss sweeper and anchors any rejection leaves. It
// is safe to call from a background goroutine (it takes checkpoint.mu only for
// the sweep duration); it must NOT be called while holding checkpoint.mu.
func (o *Oracle) sweepOnce() {
	o.checkpoint.mu.Lock()
	n, err := o.sweepExpired()
	o.checkpoint.mu.Unlock()
	if err != nil {
		o.logger.Warn("sweeper failed", zap.Error(err))
		return
	}
	if n > 0 {
		// Anchor the rejection leaves into the AppHash.
		if err := o.commitAuditMMR(); err != nil {
			o.logger.Warn("failed to anchor rejection leaves", zap.Error(err))
			return
		}
		o.logger.Info("sweeper rejected expired checkpoints", zap.Int("count", n))
	}
}

// sweepLoop runs sweepOnce on a fixed interval until the Oracle context is
// cancelled. The Oracle runs non-validator, so there is no block-production
// loop to drive the sweeper; this goroutine is its driver (merkle-math.md §3.3e).
func (o *Oracle) sweepLoop() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-o.ctx.Done():
			return
		case <-ticker.C:
			o.sweepOnce()
		}
	}
}
