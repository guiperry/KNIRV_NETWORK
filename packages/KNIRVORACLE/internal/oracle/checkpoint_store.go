package oracle

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/knirvcorp/knirvoracle/internal/oracle/mmr"
	"github.com/knirvcorp/knirvoracle/internal/oracle/registry"
	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
	"go.uber.org/zap"
)

// checkpointState is the Oracle's append-only checkpoint pipeline: a per-foreign
// chain registry, the single audit MMR, and an indexed view of admitted
// checkpoint leaves. It is the Phase-2 admission authority (merkle-math.md §3.3c).
type checkpointState struct {
	mu         sync.RWMutex
	registry   *registry.Registry
	mmr        *mmr.MMR
	records    map[string]*types.CheckpointRecord // key: chainID/startHeight
	leafLog    []mmr.Hash                          // append-only leaf digest log for replay
	registryPath string
	recordsPath  string
	leafLogPath  string
}

func newCheckpointState(dataDir string) *checkpointState {
	cs := &checkpointState{
		registry: registry.New(),
		mmr:      mmr.New(),
		records:  make(map[string]*types.CheckpointRecord),
	}
	if dataDir != "" {
		cs.registryPath = filepath.Join(dataDir, "chain_registry.json")
		cs.recordsPath = filepath.Join(dataDir, "checkpoint_records.json")
		cs.leafLogPath = filepath.Join(dataDir, "mmr_leaf_log.json")
	}
	return cs
}

// LoadCheckpointState reloads the registry, checkpoint records, and MMR leaf log
// from disk. A missing file yields the corresponding empty state (first-run).
func LoadCheckpointState(dataDir string) (*checkpointState, error) {
	cs := newCheckpointState(dataDir)
	if dataDir == "" {
		return cs, nil
	}
	reg, err := registry.Load(dataDir)
	if err != nil {
		return nil, err
	}
	cs.registry = reg

	if data, err := os.ReadFile(cs.recordsPath); err == nil {
		var recs []*types.CheckpointRecord
		if jerr := json.Unmarshal(data, &recs); jerr != nil {
			return nil, fmt.Errorf("decode checkpoint records: %w", jerr)
		}
		for _, r := range recs {
			if r != nil {
				cs.records[recordKey(r.Checkpoint.ChainID, r.Checkpoint.StartHeight)] = r
			}
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read checkpoint records: %w", err)
	}

	if data, err := os.ReadFile(cs.leafLogPath); err == nil {
		var leaves []mmr.Hash
		if jerr := json.Unmarshal(data, &leaves); jerr != nil {
			return nil, fmt.Errorf("decode mmr leaf log: %w", jerr)
		}
		cs.leafLog = leaves
		cs.mmr = mmr.FromLeaves(leaves)
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read mmr leaf log: %w", err)
	}

	return cs, nil
}

func recordKey(chainID string, startHeight uint64) string {
	return fmt.Sprintf("%s/%d", chainID, startHeight)
}

// RegisterChain adds or replaces a chain registration (POST /oracle/v3/registry/register).
func (o *Oracle) RegisterChain(c *types.ChainRegistration) error {
	if err := o.checkpoint.registry.Register(c); err != nil {
		return err
	}
	return o.persistRegistry()
}

// RotateChain applies a quorum-signed author-set rotation (POST /oracle/v3/registry/rotate).
func (o *Oracle) RotateChain(c *types.ChainRegistration) error {
	// Continuity: the rotating party must hold the current registration and prove
	// quorum over the new one. For now we require the new registration to carry
	// signatures from the existing author set (verified against the live registry).
	cur, ok := o.checkpoint.registry.Get(c.ChainID)
	if !ok {
		return fmt.Errorf("chain %s not registered", c.ChainID)
	}
	if err := verifyRegistryRotation(cur, c); err != nil {
		return err
	}
	if err := o.checkpoint.registry.Register(c); err != nil {
		return err
	}
	return o.persistRegistry()
}

// SubmitCheckpoint verifies the checkpoint against the registry and, on success,
// appends a LeafCheckpoint to the MMR and records its position. It enforces the
// 5-step admission (merkle-math.md §3.3c).
func (o *Oracle) SubmitCheckpoint(cp *types.Checkpoint) (*types.CheckpointRecord, error) {
	o.checkpoint.mu.Lock()
	defer o.checkpoint.mu.Unlock()

	if err := o.checkpoint.registry.VerifyCheckpointQuorum(cp); err != nil {
		return nil, err
	}

	// Build the stable leaf payload: canonical checkpoint body (no signatures).
	leafData, err := json.Marshal(cp.CanonicalBytes())
	if err != nil {
		return nil, fmt.Errorf("marshal checkpoint leaf: %w", err)
	}
	leaf := mmr.LeafHash(leafData)

	// Append to BOTH the index-side MMR (for proofs) and the consensus audit MMR
	// (so the next Commit() bags the leaf into the Oracle AppHash). They carry the
	// same leaf data, so both roots stay consistent.
	pos, _ := o.checkpoint.mmr.AddRaw(leaf)
	auditPos, _ := o.consensusEngine.AddAuditLeaf(leafData)
	_ = auditPos

	rec := &types.CheckpointRecord{
		Checkpoint:  *cp,
		MMRPosition: pos,
		LeafHash:    [32]byte(leaf),
		Status:      types.CheckpointProvisional,
		ReceivedAt:  time.Now().UTC(),
	}
	// FinalByHeight uses the live Oracle height (best-effort; 0 if unknown).
	if o.consensusEngine != nil {
		rec.FinalByHeight = uint64(o.consensusEngine.GetHeight()) + proofWindowFor(o, cp.ChainID)
	}

	o.checkpoint.records[recordKey(cp.ChainID, cp.StartHeight)] = rec
	o.checkpoint.leafLog = append(o.checkpoint.leafLog, leaf)

	// Advance the registry continuity cursor.
	_ = o.checkpoint.registry.Advance(cp.ChainID, cp.EndHeight, cp.Digest())

	if err := o.persistCheckpointLocked(); err != nil {
		return nil, err
	}
	// Phase 3: commit the audit leaf into the Oracle AppHash and persist it so
	// the AppHash recovers across restarts. The Oracle runs non-validator, so
	// this must be done at admission time (the consensus loop does not commit).
	o.commitAuditMMR()
	o.logger.Info("checkpoint admitted",
		zap.String("chain_id", cp.ChainID),
		zap.Uint64("start", cp.StartHeight),
		zap.Uint64("end", cp.EndHeight),
		zap.Uint64("mmr_position", pos),
	)
	return rec, nil
}

func proofWindowFor(o *Oracle, chainID string) uint64 {
	if reg, ok := o.checkpoint.registry.Get(chainID); ok && reg.ProofWindow > 0 {
		return reg.ProofWindow
	}
	return registry.DefaultProofWindow
}

// ProjectRollup bridges a legacy, unsigned RollupRecord into the new checkpoint
// MMR (merkle-math.md Phase 3): it appends a provisional checkpoint leaf WITHOUT
// requiring the author quorum (legacy rollups carry no signatures), so the same
// MMR/AppHash that anchors signed checkpoints also covers rollup history. The
// record's Source is tagged "rollup:<id>" and it is never promoted to final.
func (o *Oracle) ProjectRollup(rec *types.RollupRecord) (*types.CheckpointRecord, error) {
	o.checkpoint.mu.Lock()
	defer o.checkpoint.mu.Unlock()

	var root [32]byte
	if b, err := hex.DecodeString(strip0x(rec.BatchRoot)); err == nil && len(b) == 32 {
		copy(root[:], b)
	}

	cp := &types.Checkpoint{
		SchemaVersion: "knirv.checkpoint.v1.rollup",
		ChainID:       rec.ChainID,
		StartHeight:   rec.StartHeight,
		EndHeight:     rec.EndHeight,
		Root:          root,
		Proposer:      "rollup:" + rec.ID,
	}

	leafData, err := json.Marshal(cp.CanonicalBytes())
	if err != nil {
		return nil, fmt.Errorf("marshal rollup leaf: %w", err)
	}
	leaf := mmr.LeafHash(leafData)

	pos, _ := o.checkpoint.mmr.AddRaw(leaf)
	_, _ = o.consensusEngine.AddAuditLeaf(leafData)

	record := &types.CheckpointRecord{
		Checkpoint:  *cp,
		MMRPosition: pos,
		LeafHash:    [32]byte(leaf),
		Status:      types.CheckpointProvisional,
		ReceivedAt:  time.Now().UTC(),
		Source:      "rollup:" + rec.ID,
	}
	if o.consensusEngine != nil {
		record.FinalByHeight = uint64(o.consensusEngine.GetHeight()) + registry.DefaultProofWindow
	}

	o.checkpoint.records[recordKey(cp.ChainID, cp.StartHeight)] = record
	o.checkpoint.leafLog = append(o.checkpoint.leafLog, leaf)

	if err := o.persistCheckpointLocked(); err != nil {
		return nil, err
	}
	// Phase 3: commit the projected rollup leaf into the AppHash and persist it.
	o.commitAuditMMR()
	o.logger.Info("rollup projected to checkpoint leaf",
		zap.String("rollup_id", rec.ID),
		zap.Uint64("mmr_position", pos),
	)
	return record, nil
}

func strip0x(s string) string {
	if len(s) >= 2 && s[0] == '0' && (s[1] == 'x' || s[1] == 'X') {
		return s[2:]
	}
	return s
}

// GetCheckpointRecords returns all admitted checkpoint records for a chain.
func (o *Oracle) GetCheckpointRecords(chainID string) []*types.CheckpointRecord {
	o.checkpoint.mu.RLock()
	defer o.checkpoint.mu.RUnlock()
	out := make([]*types.CheckpointRecord, 0)
	for _, r := range o.checkpoint.records {
		if r.Checkpoint.ChainID == chainID {
			out = append(out, r)
		}
	}
	return out
}

// MMRRoot returns the current bagged MMR root (the Oracle AppHash source).
func (o *Oracle) MMRRoot() mmr.Hash {
	o.checkpoint.mu.RLock()
	defer o.checkpoint.mu.RUnlock()
	return o.checkpoint.mmr.BagRoot()
}

// CheckpointCount returns the number of admitted checkpoint leaves in the MMR.
func (o *Oracle) CheckpointCount() uint64 {
	o.checkpoint.mu.RLock()
	defer o.checkpoint.mu.RUnlock()
	return o.checkpoint.mmr.Size()
}

// MMRProof returns an inclusion proof for the leaf at the given index.
func (o *Oracle) MMRProof(index uint64) (mmr.Proof, error) {
	o.checkpoint.mu.RLock()
	defer o.checkpoint.mu.RUnlock()
	return o.checkpoint.mmr.GenerateProof(index)
}

// --- persistence ---

func (o *Oracle) persistRegistry() error {
	return o.checkpoint.registry.Persist() // path already set by registry.Load
}

func (o *Oracle) persistCheckpointLocked() error {
	if o.checkpoint.recordsPath == "" {
		return nil
	}
	recs := make([]*types.CheckpointRecord, 0, len(o.checkpoint.records))
	for _, r := range o.checkpoint.records {
		recs = append(recs, r)
	}
	payload, err := json.MarshalIndent(recs, "", "  ")
	if err != nil {
		return err
	}
	if err := writeAtomic(o.checkpoint.recordsPath, payload); err != nil {
		return err
	}
	leaves := make([]mmr.Hash, len(o.checkpoint.leafLog))
	copy(leaves, o.checkpoint.leafLog)
	lpayload, err := json.MarshalIndent(leaves, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(o.checkpoint.leafLogPath, lpayload)
}

func writeAtomic(path string, payload []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, payload, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// verifyRegistryRotation checks that the new registration's author set signed
// over the new registration body with quorum against the current set. The
// rotation body carries AuthorSigs in the same shape as checkpoints.
func verifyRegistryRotation(cur *types.ChainRegistration, next *types.ChainRegistration) error {
	// Reuse the checkpoint quorum machinery by treating the registration as a
	// pseudo-checkpoint over its marshaled body.
	body, err := json.Marshal(next)
	if err != nil {
		return err
	}
	digest := types.RegistrationDigest(body)
	// Build a transient checkpoint-shaped signature set.
	sigs := next.RotationSigs
	seen := make(map[string]bool)
	validWeight := uint64(0)
	totalWeight := uint64(0)
	for _, a := range cur.Authors {
		totalWeight += weightOf(a.Weight)
	}
	for _, sig := range sigs {
		if seen[sig.Address] {
			continue
		}
		w, ok := registryWeight(cur, sig.Address)
		if !ok {
			continue
		}
		if !types.VerifyRegistrationSig(digest, sig) {
			continue
		}
		seen[sig.Address] = true
		validWeight += w
	}
	if validWeight*2 < totalWeight {
		return fmt.Errorf("rotation quorum not met: %d/%d", validWeight, totalWeight)
	}
	return nil
}

func registryWeight(c *types.ChainRegistration, addr string) (uint64, bool) {
	for _, a := range c.Authors {
		if a.Address == addr {
			return weightOf(a.Weight), true
		}
	}
	return 0, false
}

func weightOf(w uint64) uint64 {
	if w == 0 {
		return 1
	}
	return w
}
