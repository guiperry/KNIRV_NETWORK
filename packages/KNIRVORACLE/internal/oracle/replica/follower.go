// Package replica implements the read-only side of Oracle replication.
//
// It intentionally owns only verified mirror files.  The Oracle supplies the
// transport (gateway, direct HTTP, or a test fake) and consumes the snapshot;
// keeping the protocol here prevents the standby from accidentally acquiring a
// second state-transition implementation.
package replica

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/guiperry/knirv-sdk-go/replication"
	"github.com/knirvcorp/knirvoracle/internal/oracle/ledger"
	"github.com/knirvcorp/knirvoracle/internal/oracle/mmr"
	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
)

var (
	// ErrGap is recoverable: callers should try the next source or retry once
	// that source has published the missing heads.
	ErrGap = errors.New("replica: source has a gap")
)

// Source is deliberately narrow and transport-neutral. Production can use the
// Gateway proxy while tests use an in-process fake without weakening any
// verification done by Follower.
type Source interface {
	Head(context.Context) (replication.SignedHead, error)
	HeadsAfter(context.Context, uint64) ([]replication.SignedHead, error)
	Leaves(context.Context, uint64, int) ([]string, error)
	State(context.Context, uint64, int) ([]StateRecord, error)
	Registry(context.Context) (RegistrySnapshot, error)
	Aux(context.Context) (AuxSnapshot, error)
	Checkpoints(context.Context, string, uint64, int) ([]*types.CheckpointRecord, error)
}

// StateRecord is the wire representation served by /replication/state.
type StateRecord struct {
	Seq      uint64          `json:"seq"`
	Kind     string          `json:"kind"`
	Payload  json.RawMessage `json:"payload"`
	PrevHash string          `json:"prev_hash"`
	Hash     string          `json:"hash"`
	Leaf     string          `json:"leaf"`
}

// RegistrySnapshot and AuxSnapshot are the versioned classes R3 and R6. Their
// hashes are checked against the signed head before any standby read model is
// replaced.
type RegistrySnapshot struct {
	Chains []*types.ChainRegistration `json:"chains"`
	Hash   string                     `json:"hash"`
}

type AuxSnapshot struct {
	SkillOwnership map[string]*types.SkillOwnershipRecord `json:"skill_ownership"`
	Rollups        map[string]*types.RollupRecord         `json:"rollups"`
	Hash           string                                 `json:"hash"`
}

// Snapshot is written atomically before Head is advanced. A consumer can
// hydrate a read model from it, but it must never mutate it.
type Snapshot struct {
	Head        replication.SignedHead    `json:"head"`
	Leaves      []string                  `json:"leaves"`
	Records     []ledger.Record           `json:"records"`
	State       *ledger.State             `json:"state"`
	Registry    RegistrySnapshot          `json:"registry"`
	Aux         AuxSnapshot               `json:"aux"`
	Checkpoints []*types.CheckpointRecord `json:"checkpoints"`
}

// Status is safe to expose from the Oracle status endpoint.
type Status struct {
	HeadSeq           uint64   `json:"head_seq"`
	LagSeq            uint64   `json:"lag_seq"`
	LagSeconds        int64    `json:"lag_seconds"`
	Diverged          bool     `json:"diverged"`
	Paused            bool     `json:"paused"`
	UnverifiedClasses []string `json:"unverified_classes,omitempty"`
}

// Follower verifies a root-signed sequence of heads and mirrors its committed
// MMR leaves and ledger records. Once Diverged becomes true it is absorbing:
// no future pull can overwrite the retained mirror.
type Follower struct {
	trustAnchor string
	path        string
	now         func() time.Time
	verifyHead  func(replication.SignedHead, string, time.Time) (replication.Head, error)
	compare     func(replication.Head, replication.Head) (replication.FollowResult, error)
	onApplied   func(Snapshot) error

	mu       sync.RWMutex
	snapshot Snapshot
	status   Status
}

// Open creates or reloads a standby mirror at path. The path must live in the
// standby Oracle data directory; callers never pass an active data directory.
func Open(path, trustAnchor string) (*Follower, error) {
	if path == "" {
		return nil, fmt.Errorf("replica: state path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, fmt.Errorf("replica: create directory: %w", err)
	}
	f := &Follower{path: path, trustAnchor: trustAnchor, now: time.Now, verifyHead: replication.VerifyHead, compare: replication.Compare}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return f, nil
	}
	if err != nil {
		return nil, fmt.Errorf("replica: read state: %w", err)
	}
	var persisted struct {
		Snapshot Snapshot `json:"snapshot"`
		Status   Status   `json:"status"`
	}
	if err := json.Unmarshal(data, &persisted); err != nil {
		return nil, fmt.Errorf("replica: decode state: %w", err)
	}
	if err := validateSnapshot(persisted.Snapshot); err != nil {
		return nil, fmt.Errorf("replica: invalid persisted mirror: %w", err)
	}
	f.snapshot, f.status = persisted.Snapshot, persisted.Status
	f.status.HeadSeq = f.snapshot.Head.Head.Seq
	return f, nil
}

// Status returns a copy so callers cannot clear divergence accidentally.
func (f *Follower) Status() Status {
	f.mu.RLock()
	defer f.mu.RUnlock()
	out := f.status
	out.UnverifiedClasses = append([]string(nil), f.status.UnverifiedClasses...)
	return out
}

// SetPaused records whether the standby currently has a usable authenticated
// source. A paused standby retains its last verified mirror but deliberately
// reports zero eligibility to the failover scorer.
func (f *Follower) SetPaused(paused bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.status.Paused = paused
}

// SetOnApplied installs the Oracle-owned read-model hydrator. It receives only
// a fully verified candidate snapshot and may reject it before the follower
// writes its head, preserving the atomic "data then head" advancement rule.
func (f *Follower) SetOnApplied(apply func(Snapshot) error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.onApplied = apply
}

// Snapshot returns a deep-enough copy for read-only consumers.
func (f *Follower) Snapshot() Snapshot {
	f.mu.RLock()
	defer f.mu.RUnlock()
	out := f.snapshot
	out.Leaves = append([]string(nil), f.snapshot.Leaves...)
	out.Records = append([]ledger.Record(nil), f.snapshot.Records...)
	return out
}

// Sync follows one source up to its current head. A bad signature or a source
// outage is returned without changing local data. A non-extending history is
// different: it marks the follower diverged and stays that way until an
// explicit operator action replaces the mirror.
func (f *Follower) Sync(ctx context.Context, source Source) error {
	if source == nil {
		return fmt.Errorf("replica: source is nil")
	}
	f.mu.RLock()
	if f.status.Diverged {
		f.mu.RUnlock()
		return replication.ErrDiverged
	}
	localSeq := f.snapshot.Head.Head.Seq
	f.mu.RUnlock()

	remote, err := source.Head(ctx)
	if err != nil {
		return fmt.Errorf("replica: fetch head: %w", err)
	}
	head, err := f.verifyHead(remote, f.trustAnchor, f.now())
	if err != nil {
		return err
	}
	if head.Seq <= localSeq {
		f.setRemote(head)
		return nil
	}

	heads := []replication.SignedHead{remote}
	if head.Seq > localSeq+1 {
		heads, err = source.HeadsAfter(ctx, localSeq)
		if err != nil {
			return fmt.Errorf("replica: fetch missing heads: %w", err)
		}
		sort.Slice(heads, func(i, j int) bool { return heads[i].Head.Seq < heads[j].Head.Seq })
	}
	for _, signed := range heads {
		if err := f.applyHead(ctx, source, signed); err != nil {
			return err
		}
	}
	return nil
}

func (f *Follower) applyHead(ctx context.Context, source Source, signed replication.SignedHead) error {
	head, err := f.verifyHead(signed, f.trustAnchor, f.now())
	if err != nil {
		return err
	}
	f.mu.RLock()
	local := f.snapshot
	diverged := f.status.Diverged
	f.mu.RUnlock()
	if diverged {
		return replication.ErrDiverged
	}
	if local.Head.Head.Seq == 0 {
		if head.Seq != 1 || head.PrevHeadHash != "" {
			return ErrGap
		}
	} else {
		if head.ConsensusHeight < local.Head.Head.ConsensusHeight {
			f.markDiverged(head)
			return fmt.Errorf("%w: consensus height moved backwards", replication.ErrDiverged)
		}
		result, compareErr := f.compare(local.Head.Head, head)
		if result == replication.FollowOld {
			return nil
		}
		if result == replication.FollowGap {
			return ErrGap
		}
		if compareErr != nil || result == replication.FollowDiverged {
			f.markDiverged(head)
			if compareErr != nil {
				return compareErr
			}
			return replication.ErrDiverged
		}
	}

	leaves, err := f.fetchLeaves(ctx, source, uint64(len(local.Leaves)), head.MMRSize)
	if err != nil {
		return err
	}
	allLeaves := append(append([]string(nil), local.Leaves...), leaves...)
	if uint64(len(allLeaves)) != head.MMRSize || !matchesRoot(allLeaves, head.MMRRoot) {
		f.markDiverged(head)
		return fmt.Errorf("%w: MMR leaves do not extend local head", replication.ErrDiverged)
	}
	records, state, err := f.fetchState(ctx, source, local.Records, local.State, head.StateLogSize, allLeaves)
	if err != nil {
		if errors.Is(err, replication.ErrDiverged) {
			f.markDiverged(head)
		}
		return err
	}
	root, err := ledger.StateRoot(state)
	if err != nil {
		return err
	}
	if root != head.StateRoot {
		f.markDiverged(head)
		return fmt.Errorf("%w: state root mismatch", replication.ErrDiverged)
	}
	registry, err := source.Registry(ctx)
	if err != nil {
		return fmt.Errorf("replica: fetch registry: %w", err)
	}
	if registry.Hash != head.RegistryHash || registryHash(registry.Chains) != head.RegistryHash {
		f.markDiverged(head)
		return fmt.Errorf("%w: registry hash mismatch", replication.ErrDiverged)
	}
	checkpoints, err := fetchCheckpoints(ctx, source, registry.Chains)
	if err != nil {
		return err
	}
	if err := verifyCheckpoints(checkpoints, allLeaves); err != nil {
		f.markDiverged(head)
		return err
	}
	aux, err := source.Aux(ctx)
	if err != nil {
		return fmt.Errorf("replica: fetch aux: %w", err)
	}
	if aux.Hash != head.AuxHash || auxHash(aux) != head.AuxHash {
		f.markDiverged(head)
		return fmt.Errorf("%w: auxiliary snapshot hash mismatch", replication.ErrDiverged)
	}

	next := Snapshot{Head: replication.SignedHead{Head: head, Signed: signed.Signed}, Leaves: allLeaves, Records: records, State: state, Registry: registry, Aux: aux, Checkpoints: checkpoints}
	f.mu.RLock()
	onApplied := f.onApplied
	f.mu.RUnlock()
	if onApplied != nil {
		if err := onApplied(next); err != nil {
			return fmt.Errorf("replica: hydrate read model: %w", err)
		}
	}
	if err := f.persist(next); err != nil {
		return err
	}
	f.mu.Lock()
	f.snapshot = next
	f.status = statusFor(next, head, false)
	f.mu.Unlock()
	return nil
}

func (f *Follower) fetchLeaves(ctx context.Context, source Source, from, want uint64) ([]string, error) {
	var out []string
	for from+uint64(len(out)) < want {
		page, err := source.Leaves(ctx, from+uint64(len(out)), 1024)
		if err != nil {
			return nil, fmt.Errorf("replica: fetch leaves: %w", err)
		}
		if len(page) == 0 {
			return nil, ErrGap
		}
		out = append(out, page...)
	}
	if from+uint64(len(out)) != want {
		return nil, fmt.Errorf("%w: source returned too many leaves", replication.ErrDiverged)
	}
	return out, nil
}

func (f *Follower) fetchState(ctx context.Context, source Source, current []ledger.Record, currentState *ledger.State, want uint64, allLeaves []string) ([]ledger.Record, *ledger.State, error) {
	records := append([]ledger.Record(nil), current...)
	state, err := cloneState(currentState)
	if err != nil {
		return nil, nil, err
	}
	for uint64(len(records)) < want {
		page, err := source.State(ctx, uint64(len(records)), 256)
		if err != nil {
			return nil, nil, fmt.Errorf("replica: fetch state: %w", err)
		}
		if len(page) == 0 {
			return nil, nil, ErrGap
		}
		for _, wire := range page {
			record := ledger.Record{Seq: wire.Seq, Kind: wire.Kind, Payload: wire.Payload, PrevHash: wire.PrevHash, Hash: wire.Hash}
			previous := ""
			if len(records) > 0 {
				previous = records[len(records)-1].Hash
			}
			if err := ledger.ValidateRecord(record, previous, uint64(len(records)+1)); err != nil {
				return nil, nil, fmt.Errorf("%w: %v", replication.ErrDiverged, err)
			}
			if record.Seq != 1 {
				wantLeaf := mmr.LeafHash([]byte("state:" + record.Hash))
				if hex.EncodeToString(wantLeaf[:]) != wire.Leaf || !contains(allLeaves, wire.Leaf) {
					return nil, nil, fmt.Errorf("%w: state record %d is not committed to the MMR", replication.ErrDiverged, record.Seq)
				}
			}
			if err := ledger.Apply(state, record); err != nil {
				return nil, nil, fmt.Errorf("%w: apply state record %d: %v", replication.ErrDiverged, record.Seq, err)
			}
			records = append(records, record)
			if uint64(len(records)) == want {
				break
			}
		}
	}
	return records, state, nil
}

func cloneState(in *ledger.State) (*ledger.State, error) {
	if in == nil {
		return ledger.NewState(), nil
	}
	body, err := json.Marshal(in)
	if err != nil {
		return nil, fmt.Errorf("replica: encode local state: %w", err)
	}
	var out ledger.State
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("replica: decode local state: %w", err)
	}
	return &out, nil
}

func (f *Follower) persist(snapshot Snapshot) error {
	body, err := json.Marshal(struct {
		Snapshot Snapshot `json:"snapshot"`
		Status   Status   `json:"status"`
	}{Snapshot: snapshot, Status: statusFor(snapshot, snapshot.Head.Head, false)})
	if err != nil {
		return fmt.Errorf("replica: encode mirror: %w", err)
	}
	tmp := f.path + ".tmp"
	if err := os.WriteFile(tmp, body, 0600); err != nil {
		return fmt.Errorf("replica: write mirror: %w", err)
	}
	return os.Rename(tmp, f.path)
}

func (f *Follower) markDiverged(remote replication.Head) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.status.Diverged = true
	f.status.LagSeq = sequenceLag(remote.Seq, f.snapshot.Head.Head.Seq)
	f.status.LagSeconds = age(remote.ProducedAt, f.now())
	_ = f.persist(f.snapshot) // best-effort; never discard a mirror on a fork
}

func (f *Follower) setRemote(remote replication.Head) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.status.LagSeq = sequenceLag(remote.Seq, f.snapshot.Head.Head.Seq)
	f.status.LagSeconds = age(remote.ProducedAt, f.now())
}

func statusFor(snapshot Snapshot, remote replication.Head, diverged bool) Status {
	seq := snapshot.Head.Head.Seq
	return Status{HeadSeq: seq, LagSeq: sequenceLag(remote.Seq, seq), LagSeconds: age(remote.ProducedAt, time.Now()), Diverged: diverged}
}

func sequenceLag(remote, local uint64) uint64 {
	if remote <= local {
		return 0
	}
	return remote - local
}

func age(producedAt int64, now time.Time) int64 {
	if producedAt == 0 {
		return 0
	}
	seconds := now.Unix() - producedAt
	if seconds < 0 {
		return 0
	}
	return seconds
}

func matchesRoot(leaves []string, expected string) bool {
	raw := make([]mmr.Hash, 0, len(leaves))
	for _, text := range leaves {
		decoded, err := hex.DecodeString(text)
		if err != nil || len(decoded) != len(mmr.Hash{}) {
			return false
		}
		var leaf mmr.Hash
		copy(leaf[:], decoded)
		raw = append(raw, leaf)
	}
	root := mmr.FromLeaves(raw).BagRoot()
	return hex.EncodeToString(root[:]) == expected
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func validateSnapshot(snapshot Snapshot) error {
	if snapshot.Head.Head.Seq == 0 {
		return nil
	}
	if !matchesRoot(snapshot.Leaves, snapshot.Head.Head.MMRRoot) {
		return fmt.Errorf("MMR root mismatch")
	}
	previous := ""
	for i, record := range snapshot.Records {
		if err := ledger.ValidateRecord(record, previous, uint64(i+1)); err != nil {
			return err
		}
		previous = record.Hash
	}
	state, err := ledger.Replay(snapshot.Records)
	if err != nil {
		return err
	}
	root, err := ledger.StateRoot(state)
	if err != nil || root != snapshot.Head.Head.StateRoot {
		return fmt.Errorf("state root mismatch")
	}
	if snapshot.Registry.Hash != snapshot.Head.Head.RegistryHash || registryHash(snapshot.Registry.Chains) != snapshot.Head.Head.RegistryHash {
		return fmt.Errorf("registry hash mismatch")
	}
	if snapshot.Aux.Hash != snapshot.Head.Head.AuxHash || auxHash(snapshot.Aux) != snapshot.Head.Head.AuxHash {
		return fmt.Errorf("auxiliary snapshot hash mismatch")
	}
	if err := verifyCheckpoints(snapshot.Checkpoints, snapshot.Leaves); err != nil {
		return err
	}
	return nil
}

func fetchCheckpoints(ctx context.Context, source Source, chains []*types.ChainRegistration) ([]*types.CheckpointRecord, error) {
	all := make([]*types.CheckpointRecord, 0)
	for _, chain := range chains {
		if chain == nil || chain.ChainID == "" {
			return nil, fmt.Errorf("%w: invalid replicated chain", replication.ErrDiverged)
		}
		var after uint64
		for {
			page, err := source.Checkpoints(ctx, chain.ChainID, after, 256)
			if err != nil {
				return nil, fmt.Errorf("replica: fetch checkpoints for %s: %w", chain.ChainID, err)
			}
			if len(page) == 0 {
				break
			}
			all = append(all, page...)
			last := page[len(page)-1]
			if last == nil || last.Checkpoint.EndHeight <= after {
				return nil, fmt.Errorf("%w: checkpoint page for %s did not advance", replication.ErrDiverged, chain.ChainID)
			}
			after = last.Checkpoint.EndHeight
			if len(page) < 256 {
				break
			}
		}
	}
	return all, nil
}

func verifyCheckpoints(records []*types.CheckpointRecord, leaves []string) error {
	seen := make(map[string]bool, len(records))
	for _, record := range records {
		if record == nil || record.Checkpoint.ChainID == "" || record.MMRPosition >= uint64(len(leaves)) {
			return fmt.Errorf("%w: invalid checkpoint replica record", replication.ErrDiverged)
		}
		key := fmt.Sprintf("%s/%d", record.Checkpoint.ChainID, record.Checkpoint.StartHeight)
		if seen[key] {
			return fmt.Errorf("%w: duplicate checkpoint %s", replication.ErrDiverged, key)
		}
		seen[key] = true
		if hex.EncodeToString(record.LeafHash[:]) != leaves[record.MMRPosition] {
			return fmt.Errorf("%w: checkpoint %s is not committed at its MMR position", replication.ErrDiverged, key)
		}
	}
	return nil
}

func registryHash(chains []*types.ChainRegistration) string {
	ordered := append([]*types.ChainRegistration(nil), chains...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].ChainID < ordered[j].ChainID })
	body, err := json.Marshal(ordered)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func auxHash(aux AuxSnapshot) string {
	if aux.SkillOwnership == nil {
		aux.SkillOwnership = map[string]*types.SkillOwnershipRecord{}
	}
	if aux.Rollups == nil {
		aux.Rollups = map[string]*types.RollupRecord{}
	}
	body, err := json.Marshal(map[string]any{"skill_ownership": aux.SkillOwnership, "rollups": aux.Rollups})
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}
