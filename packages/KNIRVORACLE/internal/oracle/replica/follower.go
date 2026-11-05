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
	"reflect"
	"sort"
	"strings"
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
	Heads       []replication.SignedHead  `json:"heads,omitempty"`
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
	syncMu      sync.Mutex
	retries     map[string]sourceRetry

	mu       sync.RWMutex
	snapshot Snapshot
	status   Status
}

type sourceRetry struct {
	failures uint
	next     time.Time
}

func retryKey(index int, source Source) string {
	v := reflect.ValueOf(source)
	if v.IsValid() && v.Kind() == reflect.Pointer {
		return fmt.Sprintf("%T:%x", source, v.Pointer())
	}
	return fmt.Sprintf("%T:%d", source, index)
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
	f := &Follower{path: path, trustAnchor: trustAnchor, now: time.Now, verifyHead: replication.VerifyHead, compare: replication.Compare,
		status: Status{Paused: true, UnverifiedClasses: []string{"R1", "R2", "R3", "R4", "R5", "R6"}}}
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
	if statusBytes, err := os.ReadFile(path + ".status"); err == nil {
		var saved Status
		if err := json.Unmarshal(statusBytes, &saved); err != nil {
			return nil, fmt.Errorf("replica: decode status: %w", err)
		}
		// The sidecar is the durable, operator-auditable disposition of a
		// detected fork.  It intentionally overlays only the safety gates;
		// sequence and lag continue to come from the verified snapshot.
		f.status.Diverged = saved.Diverged
		if saved.Paused {
			f.status.Paused = true
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("replica: read status: %w", err)
	}
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

// ClearDivergence is the deliberately narrow recovery operation for a forked
// standby.  It never replaces or rewinds the mirrored snapshot: an operator
// must name the hash of the retained, independently-audited signed head and
// that head is verified again against the configured trust anchor.  Recovery
// stays paused until the normal authenticated sync loop proves extension.
func (f *Follower) ClearDivergence(expectedHeadHash string) error {
	expectedHeadHash = strings.TrimSpace(expectedHeadHash)
	if expectedHeadHash == "" {
		return fmt.Errorf("replica: expected head hash is required")
	}
	f.syncMu.Lock()
	defer f.syncMu.Unlock()

	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.status.Diverged {
		return fmt.Errorf("replica: standby is not diverged")
	}
	if f.snapshot.Head.Head.Seq == 0 {
		return fmt.Errorf("replica: no retained head to acknowledge")
	}
	verified, err := f.verifyHead(f.snapshot.Head, f.trustAnchor, f.now())
	if err != nil {
		return fmt.Errorf("replica: retained head no longer verifies: %w", err)
	}
	if verified != f.snapshot.Head.Head {
		return fmt.Errorf("replica: retained signed head does not match mirror")
	}
	actualHash, err := f.snapshot.Head.Head.Hash()
	if err != nil {
		return fmt.Errorf("replica: hash retained head: %w", err)
	}
	if !strings.EqualFold(expectedHeadHash, actualHash) {
		return fmt.Errorf("replica: expected head hash does not match retained mirror")
	}
	next := f.status
	next.Diverged = false
	next.Paused = true
	if err := f.persistStatus(next); err != nil {
		return err
	}
	f.status = next
	return nil
}

// SyncSources tries the configured sources in order. Failure of every source
// pauses eligibility until a later successful pull.
func (f *Follower) SyncSources(ctx context.Context, sources []Source) error {
	f.syncMu.Lock()
	defer f.syncMu.Unlock()
	if f.retries == nil {
		f.retries = make(map[string]sourceRetry)
	}
	active := make(map[string]bool, len(sources))
	for index, source := range sources {
		active[retryKey(index, source)] = true
	}
	for key := range f.retries {
		if !active[key] {
			delete(f.retries, key)
		}
	}
	var lastErr error
	for index, source := range sources {
		key := retryKey(index, source)
		retry := f.retries[key]
		if f.now().Before(retry.next) {
			continue
		}
		if err := f.Sync(ctx, source); err == nil {
			delete(f.retries, key)
			f.SetPaused(false)
			return nil
		} else {
			lastErr = err
			if retry.failures < 8 {
				retry.failures++
			}
			delay := time.Second << (retry.failures - 1)
			if delay > 5*time.Minute {
				delay = 5 * time.Minute
			}
			retry.next = f.now().Add(delay)
			f.retries[key] = retry
			if errors.Is(err, replication.ErrDiverged) {
				break
			}
		}
	}
	f.SetPaused(true)
	if lastErr == nil {
		lastErr = fmt.Errorf("replica: no source ready")
	}
	return lastErr
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
	out.Heads = append([]replication.SignedHead(nil), f.snapshot.Heads...)
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
	localHead := f.snapshot.Head.Head
	f.mu.RUnlock()

	remote, err := source.Head(ctx)
	if err != nil {
		return fmt.Errorf("replica: fetch head: %w", err)
	}
	head, err := f.verifyHead(remote, f.trustAnchor, f.now())
	if err != nil {
		return err
	}
	if head.Seq == 0 {
		return fmt.Errorf("%w: source has no verified head", ErrGap)
	}
	f.setRemote(head)
	if head.Seq == localSeq && localSeq != 0 && head != localHead {
		f.markDiverged(head)
		return fmt.Errorf("%w: conflicting head at sequence %d", replication.ErrDiverged, head.Seq)
	}
	if head.Seq < localSeq {
		return fmt.Errorf("%w: source head %d is behind local head %d", ErrGap, head.Seq, localSeq)
	}
	if head.Seq == localSeq {
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
	// A bounded page can stop short of the advertised head; retain the true
	// remaining lag rather than reporting the last applied page as current.
	f.setRemote(head)
	if f.Status().HeadSeq < head.Seq {
		return fmt.Errorf("%w: source did not supply all advertised heads", ErrGap)
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

	verified := replication.SignedHead{Head: head, Signed: signed.Signed}
	heads := append([]replication.SignedHead(nil), local.Heads...)
	if len(heads) == 0 && local.Head.Head.Seq != 0 {
		heads = append(heads, local.Head)
	}
	heads = append(heads, verified)
	if len(heads) > 1024 {
		heads = heads[len(heads)-1024:]
	}
	next := Snapshot{Head: verified, Heads: heads, Leaves: allLeaves, Records: records, State: state, Registry: registry, Aux: aux, Checkpoints: checkpoints}
	f.mu.RLock()
	onApplied := f.onApplied
	f.mu.RUnlock()
	// Persist first: after a crash, startup hydrates the durable snapshot. A
	// failed write must never expose a newer read model with an older head.
	staged := statusFor(next, head, false)
	staged.Paused = true
	if err := f.persist(next, staged); err != nil {
		return err
	}
	f.SetPaused(true)
	if onApplied != nil {
		if err := onApplied(next); err != nil {
			// Keep the staged snapshot for restart/retry: hydration may have
			// partially updated read-model files and cannot safely be undone.
			return fmt.Errorf("replica: hydrate read model: %w", err)
		}
	}
	if err := f.persist(next, statusFor(next, head, false)); err != nil {
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

func (f *Follower) persist(snapshot Snapshot, status Status) error {
	body, err := json.Marshal(struct {
		Snapshot Snapshot `json:"snapshot"`
		Status   Status   `json:"status"`
	}{Snapshot: snapshot, Status: status})
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
	// Keep the verified snapshot byte-identical when a source forks.
	// A failure to write this diagnostic sidecar cannot make the verified
	// snapshot unsafe, but leaving the in-memory gate set still fails closed.
	_ = f.persistStatus(f.status)
}

func (f *Follower) persistStatus(status Status) error {
	body, err := json.Marshal(status)
	if err != nil {
		return fmt.Errorf("replica: encode status: %w", err)
	}
	tmp := f.path + ".status.tmp"
	if err := os.WriteFile(tmp, body, 0600); err != nil {
		return fmt.Errorf("replica: write status: %w", err)
	}
	if err := os.Rename(tmp, f.path+".status"); err != nil {
		return fmt.Errorf("replica: replace status: %w", err)
	}
	return nil
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
	ordered := append(make([]*types.ChainRegistration, 0, len(chains)), chains...)
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
