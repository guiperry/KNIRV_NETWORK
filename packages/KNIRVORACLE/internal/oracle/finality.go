package oracle

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/knirvcorp/knirvoracle/internal/oracle/mmr"
	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
	"go.uber.org/zap"
)

// VerifierQuorumNumer/Denom define the attestation threshold: ≥ Numer/Denom of
// registered verifier nodes must approve. Defaults to 2/3 (merkle-math.md §3.3f).
const (
	DefaultVerifierQuorumNumer = 2
	DefaultVerifierQuorumDenom = 3
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
	var ids []string
	if err := json.Unmarshal(data, &ids); err != nil {
		return fmt.Errorf("decode verifiers: %w", err)
	}
	for _, id := range ids {
		o.verifiers[id] = true
	}
	return nil
}

// persistVerifiersLocked writes the verifier registry; caller holds verifiersMu.
func (o *Oracle) persistVerifiersLocked() error {
	if o.verifiersPath == "" {
		return nil
	}
	ids := make([]string, 0, len(o.verifiers))
	for id := range o.verifiers {
		ids = append(ids, id)
	}
	payload, err := json.MarshalIndent(ids, "", "  ")
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
func (o *Oracle) RegisterVerifier(id string) error {
	if id == "" {
		return fmt.Errorf("verifier id required")
	}
	o.verifiersMu.Lock()
	defer o.verifiersMu.Unlock()
	o.verifiers[id] = true
	return o.persistVerifiersLocked()
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
	// Tolerate a client omitting the hash: default it from the anchored record.
	if rec.CheckpointHash == ([32]byte{}) {
		rec.CheckpointHash = cp.LeafHash
	}
	if cp.Status != types.CheckpointProvisional {
		return nil, fmt.Errorf("checkpoint %d is %s, not provisional", rec.CheckpointLeaf, cp.Status)
	}
	// Window check: the Oracle must still be within the proof window.
	if o.consensusEngine != nil {
		if h := uint64(o.consensusEngine.GetHeight()); h > cp.FinalByHeight {
			return nil, fmt.Errorf("checkpoint %d missed its proof window (height %d > %d)", rec.CheckpointLeaf, h, cp.FinalByHeight)
		}
	}

	approved := 0
	for _, a := range rec.Attestations {
		if a.Approved {
			approved++
		}
	}
	if !o.attestationQuorum(approved) {
		return nil, fmt.Errorf("attestation quorum not met: %d approved", approved)
	}

	// Build + append the LeafFinality payload.
	leafData, err := json.Marshal(types.FinalityLeaf{
		Kind:           byte(types.LeafFinality),
		CheckpointLeaf: rec.CheckpointLeaf,
		CheckpointHash: fmt.Sprintf("%x", rec.CheckpointHash),
		ProofSystem:    rec.ProofSystem,
		Attestations:   rec.Attestations,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal finality leaf: %w", err)
	}
	leaf := mmr.LeafHash(leafData)
	pos, _ := o.checkpoint.mmr.AddRaw(leaf)
	_, _ = o.consensusEngine.AddAuditLeaf(leafData)

	cp.Status = types.CheckpointFinal
	fpos := pos
	cp.FinalityLeaf = &fpos
	cp.PendingAttestations = rec.Attestations
	o.checkpoint.leafLog = append(o.checkpoint.leafLog, leaf)

	if err := o.persistCheckpointLocked(); err != nil {
		return nil, err
	}
	o.commitAuditMMR()
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
	o.checkpoint.mu.Lock()
	defer o.checkpoint.mu.Unlock()

	rec, ok := o.checkpoint.records[recordKey(chainID, startHeight)]
	if !ok {
		return nil, fmt.Errorf("no checkpoint for %s/%d", chainID, startHeight)
	}
	if rec.Status != types.CheckpointProvisional {
		return nil, fmt.Errorf("checkpoint %s/%d is %s", chainID, startHeight, rec.Status)
	}
	// De-duplicate by verifier id.
	for _, a := range rec.PendingAttestations {
		if a.VerifierID == att.VerifierID {
			return rec, nil
		}
	}
	rec.PendingAttestations = append(rec.PendingAttestations, att)

	approved := 0
	for _, a := range rec.PendingAttestations {
		if a.Approved {
			approved++
		}
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
		ProofSystem:     "hashchain-v0",
		Attestations:    rec.PendingAttestations,
	}
	// Relock: SubmitFinality takes the write lock itself.
	o.checkpoint.mu.Unlock()
	defer o.checkpoint.mu.Lock()
	return o.SubmitFinality(fin)
}

// sweepExpired finalizes the window-miss policy: any provisional record whose
// proof window has elapsed without a finality leaf is tombstoned with a
// LeafRejection (merkle-math.md §3.3e). Caller holds o.checkpoint.mu.
// Returns the number of records rejected.
func (o *Oracle) sweepExpired() (int, error) {
	if o.consensusEngine == nil {
		return 0, nil
	}
	height := uint64(o.consensusEngine.GetHeight())
	rejected := 0
	for _, rec := range o.checkpoint.records {
		if rec == nil || rec.Status != types.CheckpointProvisional {
			continue
		}
		if height > rec.FinalByHeight {
			leafData, err := json.Marshal(types.RejectionLeaf{
				Kind:           byte(types.LeafRejection),
				CheckpointLeaf: rec.MMRPosition,
				Reason:         "window-miss",
			})
			if err != nil {
				return rejected, err
			}
			leaf := mmr.LeafHash(leafData)
			pos, _ := o.checkpoint.mmr.AddRaw(leaf)
			_, _ = o.consensusEngine.AddAuditLeaf(leafData)
			rec.Status = types.CheckpointRejected
			leafIndex := uint64(pos)
			rec.RejectionLeaf = &leafIndex
			o.checkpoint.leafLog = append(o.checkpoint.leafLog, leaf)
			rejected++
			o.logger.Warn("checkpoint rejected (window miss)",
				zap.Uint64("checkpoint_leaf", rec.MMRPosition),
				zap.Uint64("height", height),
				zap.Uint64("final_by_height", rec.FinalByHeight),
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
		o.commitAuditMMR()
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
