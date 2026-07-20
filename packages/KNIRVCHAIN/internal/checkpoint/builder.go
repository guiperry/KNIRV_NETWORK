package checkpoint

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"sort"
	"sync"

	"KNIRVCHAIN/internal/database"
)

// CheckpointPrefix is the LevelDB key prefix for stored checkpoints.
const CheckpointPrefix = "checkpoint:"

// CheckpointSource exposes the minimal chain state the builder needs. The real
// *blockchain.BlockchainStruct satisfies it; tests inject a fixture.
type CheckpointSource interface {
	GetChainID() string
	// TipHeight returns the height of the best chain tip (len(Blocks)-1).
	TipHeight() uint64
	// AccumRootAt returns the chain AccumRoot committed at the given height.
	// If the height has no committed root, it returns the zero value.
	AccumRootAt(height uint64) ([32]byte, error)
	// NetworkAuthors returns the set of PoAu-D author addresses.
	NetworkAuthors() map[string]bool
}

// Builder produces, signs, and persists checkpoints. It hooks the end of block
// commit: when the chain tip advances past a checkpoint interval boundary AND
// the checkpoint's EndHeight is at least CheckpointFinalityDepth below the tip,
// a new checkpoint is built, signed by the author quorum, and stored.
type Builder struct {
	mu sync.Mutex

	src    CheckpointSource
	db     *database.LevelDB
	interval uint64
	finality uint64
	proposer string

	// lastEnd tracks the height covered by the most recent checkpoint so the
	// next one begins at lastEnd+1 (contiguity / PrevCheckHash chaining).
	lastEnd   uint64
	prevHash  [32]byte
	sealed    bool
}

// NewBuilder constructs a checkpoint Builder. interval is the block span per
// checkpoint; finality is the minimum depth below tip (defaults to
// CheckpointFinalityDepth when 0); proposer is the author address that builds.
func NewBuilder(src CheckpointSource, db *database.LevelDB, interval, finality uint64, proposer string) *Builder {
	if interval == 0 {
		interval = 1
	}
	if finality == 0 {
		finality = CheckpointFinalityDepth
	}
	return &Builder{
		src:      src,
		db:       db,
		interval: interval,
		finality: finality,
		proposer: proposer,
	}
}

// Config exposes builder tuning.
func (b *Builder) Config() (interval, finality uint64) {
	return b.interval, b.finality
}

// SetProgress restores builder continuity after load (used on restart).
func (b *Builder) SetProgress(lastEnd uint64, prevHash [32]byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.lastEnd = lastEnd
	b.prevHash = prevHash
}

// Progress returns the continuity cursor.
func (b *Builder) Progress() (lastEnd uint64, prevHash [32]byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.lastEnd, b.prevHash
}

// OnNewBlock is called after a block is committed. It returns a fully signed
// checkpoint if one is due, otherwise nil. A checkpoint is due when the tip has
// advanced past the next interval boundary and the candidate EndHeight sits at
// least finality blocks below the tip (never checkpointing shallow blocks).
func (b *Builder) OnNewBlock(newTip uint64) (*Checkpoint, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Next candidate checkpoint covers (lastEnd, lastEnd+interval].
	candidateEnd := b.lastEnd + b.interval
	if newTip < candidateEnd {
		return nil, nil
	}

	// EndHeight must be >= finality below the tip. If the candidate would cover
	// a block shallower than that, hold off until deeper blocks land.
	if candidateEnd > newTip-b.finality {
		return nil, nil
	}

	endHeight := candidateEnd
	root, err := b.src.AccumRootAt(endHeight)
	if err != nil {
		return nil, fmt.Errorf("accum root at %d: %w", endHeight, err)
	}

	cp := &Checkpoint{
		SchemaVersion: SchemaVersion,
		ChainID:       b.src.GetChainID(),
		StartHeight:   b.lastEnd + 1,
		EndHeight:     endHeight,
		Root:          root,
		PrevCheckHash: b.prevHash,
		Proposer:      b.proposer,
	}
	return cp, nil
}

// Finalize signs the checkpoint with the supplied author keys (one per address)
// and persists it, advancing the builder's continuity cursor. It does NOT post
// to the Oracle — that is the Poster's job (phase 2).
func (b *Builder) Finalize(cp *Checkpoint, signers map[string]*ecdsa.PrivateKey) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	authors := b.src.NetworkAuthors()
	for addr, k := range signers {
		if !authors[addr] {
			continue
		}
		if err := SignCheckpoint(cp, k); err != nil {
			return fmt.Errorf("sign with %s: %w", addr, err)
		}
	}
	if cp.StartHeight != b.lastEnd+1 {
		return fmt.Errorf("non-contiguous checkpoint: start %d != lastEnd+1 %d", cp.StartHeight, b.lastEnd+1)
	}
	if cp.PrevCheckHash != b.prevHash {
		return fmt.Errorf("prev check hash mismatch")
	}
	if !cp.VerifyQuorum(authors, 2, 3) {
		return fmt.Errorf("insufficient signature quorum")
	}

	if err := b.persist(cp); err != nil {
		return err
	}
	b.lastEnd = cp.EndHeight
	b.prevHash = sha256Checkpoint(cp)
	return nil
}

func sha256Checkpoint(cp *Checkpoint) [32]byte {
	// The PrevCheckHash of the NEXT checkpoint chains off this checkpoint's own
	// digest, giving a tamper-evident linked list.
	return cp.Digest()
}

func (b *Builder) persist(cp *Checkpoint) error {
	if b.db == nil {
		return nil
	}
	data, err := json.Marshal(cp)
	if err != nil {
		return err
	}
	key := fmt.Sprintf("%s%s:%d", CheckpointPrefix, cp.ChainID, cp.EndHeight)
	return b.db.PutBytes(key, data)
}

// LoadStored returns all checkpoints persisted for a chain, ordered by height.
func (b *Builder) LoadStored(chainID string) ([]*Checkpoint, error) {
	if b.db == nil {
		return nil, nil
	}
	keys, err := b.db.GetKeysWithPrefix(CheckpointPrefix)
	if err != nil {
		return nil, err
	}
	out := make([]*Checkpoint, 0, len(keys))
	for _, k := range keys {
		data, err := b.db.GetBytes(k)
		if err != nil || len(data) == 0 {
			continue
		}
		var cp Checkpoint
		if err := json.Unmarshal(data, &cp); err != nil {
			continue
		}
		if cp.ChainID != chainID {
			continue
		}
		c := cp
		out = append(out, &c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].EndHeight < out[j].EndHeight })
	return out, nil
}
