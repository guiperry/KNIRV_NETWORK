package oracle

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
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
	mu           sync.RWMutex
	registry     *registry.Registry
	mmr          *mmr.MMR
	records      map[string]*types.CheckpointRecord // key: chainID/startHeight
	leafLog      []mmr.Hash                         // append-only leaf digest log for replay
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
	if c == nil {
		return fmt.Errorf("chain registration required")
	}
	if _, exists := o.checkpoint.registry.Get(c.ChainID); exists {
		return fmt.Errorf("chain %s is already registered; use the rotation endpoint", c.ChainID)
	}
	digest, err := types.ChainRegistrationDigest(c)
	if err != nil {
		return fmt.Errorf("registration digest: %w", err)
	}
	if err := registry.VerifySignatureQuorum(c, digest, c.RotationSigs); err != nil {
		return fmt.Errorf("unauthorized initial registration: %w", err)
	}
	if c.Bond > 0 && c.BondOwner == "" && len(c.Authors) > 0 {
		c.BondOwner = c.Authors[0].Address
	}
	locked := false
	if c.Bond > 0 && o.economicsEngine != nil {
		owner, err := types.AddressFromString(c.BondOwner)
		if err != nil {
			return fmt.Errorf("invalid bond owner: %w", err)
		}
		if _, err := o.economicsEngine.LockChainBond(c.ChainID, owner, new(big.Int).SetUint64(c.Bond)); err != nil {
			return fmt.Errorf("lock chain bond: %w", err)
		}
		locked = true
	}
	if err := o.checkpoint.registry.Register(c); err != nil {
		if locked {
			_ = o.economicsEngine.ReleaseChainBond(c.ChainID)
		}
		return err
	}
	return o.persistRegistry()
}

// SetChainVerificationKey enrolls a SNARK verification key + preferred proof
// system for a registered chain (merkle-math.md Phase 5). After this, the Oracle
// accepts groth16/plonk finality records for the chain; hashchain-v0 needs no key.
func (o *Oracle) SetChainVerificationKey(chainID, proofSystem string, vk []byte) error {
	if err := o.checkpoint.registry.SetVerificationKey(chainID, proofSystem, vk); err != nil {
		return err
	}
	return o.persistRegistry()
}

func (o *Oracle) GetChainRegistration(chainID string) (*types.ChainRegistration, bool) {
	reg, ok := o.checkpoint.registry.Get(chainID)
	if !ok {
		return nil, false
	}
	copyReg := *reg
	copyReg.Authors = append([]types.RegisteredAuthor(nil), reg.Authors...)
	copyReg.VerificationKey = append([]byte(nil), reg.VerificationKey...)
	return &copyReg, true
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
	// Author rotation cannot silently replace, release, or replenish the
	// economics-backed bond; that requires a separate governance operation.
	c.Bond = cur.Bond
	c.BondOwner = cur.BondOwner
	c.BondRemaining = cur.BondRemaining
	c.BondSlashed = cur.BondSlashed
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
	pos, leaf := o.appendAuditLeafLocked(leafData)

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

	// Advance the registry continuity cursor.
	_ = o.checkpoint.registry.Advance(cp.ChainID, cp.EndHeight, cp.Digest())

	if err := o.persistCheckpointLocked(); err != nil {
		return nil, err
	}
	// Phase 3: commit the audit leaf into the Oracle AppHash and persist it so
	// the AppHash recovers across restarts. The Oracle runs non-validator, so
	// this must be done at admission time (the consensus loop does not commit).
	if err := o.commitAuditMMR(); err != nil {
		return nil, err
	}
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
	pos, leaf := o.appendAuditLeafLocked(leafData)

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

	if err := o.persistCheckpointLocked(); err != nil {
		return nil, err
	}
	// Phase 3: commit the projected rollup leaf into the AppHash and persist it.
	if err := o.commitAuditMMR(); err != nil {
		return nil, err
	}
	o.logger.Info("rollup projected to checkpoint leaf",
		zap.String("rollup_id", rec.ID),
		zap.Uint64("mmr_position", pos),
	)
	return record, nil
}

// appendAuditLeafLocked is the only checkpoint-pipeline append path. The
// consensus application and proof index share checkpoint.mmr, so this performs
// one mutation and records the corresponding leaf for recovery.
func (o *Oracle) appendAuditLeafLocked(leafData []byte) (uint64, mmr.Hash) {
	if o.consensusEngine != nil {
		o.consensusEngine.SetAuditMMR(o.checkpoint.mmr)
		pos, leaf := o.consensusEngine.AddAuditLeaf(leafData)
		o.checkpoint.leafLog = append(o.checkpoint.leafLog, leaf)
		return pos, leaf
	}
	leaf := mmr.LeafHash(leafData)
	pos, _ := o.checkpoint.mmr.AddRaw(leaf)
	o.checkpoint.leafLog = append(o.checkpoint.leafLog, leaf)
	return pos, leaf
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
	digest, err := types.ChainRegistrationDigest(next)
	if err != nil {
		return err
	}
	return registry.VerifySignatureQuorum(cur, digest, next.RotationSigs)
}
