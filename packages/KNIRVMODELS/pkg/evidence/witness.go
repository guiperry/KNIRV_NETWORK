package evidence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Provenance witnessing (Task 3.9, three_layer_inference.md section 5.6).
//
// A batch of spans is hashed into a Merkle tree at ingest (that already existed). This file
// adds the three things the `attested` rung of the gate spectrum needs:
//
//  1. the batch's ordered leaf list is kept, so an inclusion proof for any span can be
//     rebuilt and the root recomputed;
//  2. a witness queue records which batch roots still wait to be witnessed (mined as an
//     assertion by the seeder, a follow-up) and which have been, with the attestation;
//  3. Index.Attested says whether a cited span is included in a witnessed root.
//
// Witnessing is never in the request path: Ingest only appends a queue line, a worker
// drains the queue later, and a verdict only reads.

const (
	batchDirName  = "batches"
	witnessLedger = "witness.jsonl"
)

// BatchRecord is the persisted leaf list of one ingest batch.
type BatchRecord struct {
	BatchID string   `json:"batch_id"`
	Root    string   `json:"merkle_root"`
	Tier    Tier     `json:"tier"`
	Created string   `json:"created"`
	Leaves  []string `json:"leaves"` // span ids in tree order
}

type leafRef struct {
	batchID string
	root    string
	index   int
}

// ProofStepJSON is a ProofStep as it travels.
type ProofStepJSON struct {
	Sibling string `json:"sibling"`
	Left    bool   `json:"left"`
}

// BatchProof proves that SpanID is a leaf of the batch tree whose root is Root.
type BatchProof struct {
	SpanID  string          `json:"span_id"`
	BatchID string          `json:"batch_id"`
	Root    string          `json:"merkle_root"`
	Index   int             `json:"index"`
	Steps   []ProofStepJSON `json:"steps"`
}

// VerifyBatchProof checks the proof against its own root.
func VerifyBatchProof(p BatchProof) bool {
	id, err := ParseDigest(p.SpanID)
	if err != nil {
		return false
	}
	root, err := ParseDigest(p.Root)
	if err != nil {
		return false
	}
	steps := make([]ProofStep, 0, len(p.Steps))
	for _, s := range p.Steps {
		sib, err := ParseDigest(s.Sibling)
		if err != nil {
			return false
		}
		steps = append(steps, ProofStep{Sibling: sib, Left: s.Left})
	}
	return VerifyProof(id, steps, root)
}

// recordBatch persists the leaf list and indexes it. Called from Ingest under ix.mu.
func (ix *Index) recordBatch(rec BatchRecord) error {
	dir := filepath.Join(ix.dir, batchDirName)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	b, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	if err := writeFileSync(filepath.Join(dir, rec.BatchID+".json"), b); err != nil {
		return err
	}
	for i, id := range rec.Leaves {
		ix.leafIdx[id] = leafRef{batchID: rec.BatchID, root: rec.Root, index: i}
	}
	// Disk-backed span-to-batch map (Task 3.9 item 5): append the mapping so a
	// future open reloads it from one file instead of re-parsing every batch.
	if err := ix.appendLeafIndex(rec); err != nil {
		return err
	}
	return nil
}

// loadBatches reads every persisted leaf list into the span index.
func (ix *Index) loadBatches() {
	names, _ := filepath.Glob(filepath.Join(ix.dir, batchDirName, "*.json"))
	sort.Strings(names)
	for _, n := range names {
		rec, err := readBatchFile(n)
		if err != nil {
			continue
		}
		for i, id := range rec.Leaves {
			ix.leafIdx[id] = leafRef{batchID: rec.BatchID, root: rec.Root, index: i}
		}
	}
}

func readBatchFile(path string) (BatchRecord, error) {
	var rec BatchRecord
	b, err := os.ReadFile(path)
	if err != nil {
		return rec, err
	}
	if err := json.Unmarshal(b, &rec); err != nil {
		return rec, err
	}
	if rec.BatchID == "" || rec.Root == "" {
		return rec, errors.New("evidence: incomplete batch record")
	}
	return rec, nil
}

// leafIndexFile is the append-only, disk-backed span-to-batch map (Task 3.9 item
// 5). Each ingested batch appends one line per leaf, so the in-memory index can be
// rebuilt by reading this single file at open instead of re-parsing every batch
// JSON. Older indexes without this file fall back to loadBatches.
const leafIndexFile = "leaf_index.jsonl"

type leafIndexEntry struct {
	SpanID  string `json:"span_id"`
	BatchID string `json:"batch_id"`
	Root    string `json:"root"`
	Index   int    `json:"index"`
}

// appendLeafIndex persists the span-to-batch mapping for one batch. Called from
// recordBatch under ix.mu so concurrent ingests serialize their appends.
func (ix *Index) appendLeafIndex(rec BatchRecord) error {
	f, err := os.OpenFile(filepath.Join(ix.dir, leafIndexFile), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	for i, id := range rec.Leaves {
		e := leafIndexEntry{SpanID: id, BatchID: rec.BatchID, Root: rec.Root, Index: i}
		b, err := json.Marshal(e)
		if err != nil {
			return err
		}
		if _, err := f.Write(append(b, '\n')); err != nil {
			return err
		}
	}
	return nil
}

// loadLeafIndex rebuilds ix.leafIdx from the disk-backed leaf index file. It
// returns true if the file existed and contributed any entries.
func (ix *Index) loadLeafIndex() bool {
	data, err := os.ReadFile(filepath.Join(ix.dir, leafIndexFile))
	if err != nil {
		return false
	}
	loaded := false
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var e leafIndexEntry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			continue
		}
		if e.SpanID == "" {
			continue
		}
		ix.leafIdx[e.SpanID] = leafRef{batchID: e.BatchID, root: e.Root, index: e.Index}
		loaded = true
	}
	return loaded
}

// ErrNoBatchRecord means the span has no persisted leaf list: it was ingested before leaf
// lists were kept, or it is not a span of this index. Such a span cannot be attested.
var ErrNoBatchRecord = errors.New("evidence: no batch record for this span")

// InclusionProof rebuilds the proof that spanID is part of its ingest batch.
func (ix *Index) InclusionProof(spanID string) (BatchProof, error) {
	ix.mu.RLock()
	ref, ok := ix.leafIdx[spanID]
	ix.mu.RUnlock()
	if !ok {
		return BatchProof{}, ErrNoBatchRecord
	}
	rec, err := readBatchFile(filepath.Join(ix.dir, batchDirName, ref.batchID+".json"))
	if err != nil {
		return BatchProof{}, fmt.Errorf("evidence: batch %s: %w", ref.batchID, err)
	}
	ids := make([]Digest, len(rec.Leaves))
	for i, l := range rec.Leaves {
		d, err := ParseDigest(l)
		if err != nil {
			return BatchProof{}, err
		}
		ids[i] = d
	}
	if ref.index >= len(ids) || rec.Leaves[ref.index] != spanID {
		return BatchProof{}, errors.New("evidence: batch record does not hold the span where the index says")
	}
	p := BatchProof{SpanID: spanID, BatchID: rec.BatchID, Root: rec.Root, Index: ref.index}
	for _, s := range MerkleProof(ids, ref.index) {
		p.Steps = append(p.Steps, ProofStepJSON{Sibling: s.Sibling.String(), Left: s.Left})
	}
	return p, nil
}

// VerifyLedger recomputes every persisted batch's root from its leaves and checks it against
// the root the batch record and the evidence_writes.jsonl ledger claim. A mismatch means the
// leaf lists or the ledger were altered. It returns the number of batches checked.
func (ix *Index) VerifyLedger() (int, error) {
	ledger := map[string]string{} // batch id -> root
	err := readLines(filepath.Join(ix.dir, "evidence_writes.jsonl"), func(b []byte) error {
		var r struct {
			BatchID string `json:"batch_id"`
			Root    string `json:"merkle_root"`
		}
		if json.Unmarshal(b, &r) == nil && r.BatchID != "" {
			ledger[r.BatchID] = r.Root
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	names, _ := filepath.Glob(filepath.Join(ix.dir, batchDirName, "*.json"))
	checked := 0
	for _, n := range names {
		rec, err := readBatchFile(n)
		if err != nil {
			return checked, fmt.Errorf("%s: %w", filepath.Base(n), err)
		}
		ids := make([]Digest, len(rec.Leaves))
		for i, l := range rec.Leaves {
			if ids[i], err = ParseDigest(l); err != nil {
				return checked, fmt.Errorf("batch %s leaf %d: %w", rec.BatchID, i, err)
			}
		}
		if got := MerkleRoot(ids).String(); got != rec.Root {
			return checked, fmt.Errorf("batch %s: leaves hash to %s, record says %s", rec.BatchID, got, rec.Root)
		}
		if want, ok := ledger[rec.BatchID]; ok && want != rec.Root {
			return checked, fmt.Errorf("batch %s: the ledger says root %s, the record %s", rec.BatchID, want, rec.Root)
		}
		checked++
	}
	return checked, nil
}

// ---- the witness queue ----

// WitnessAttestation is what a witnesser returns for a root: Kind names the mechanism (for
// example "seeder-pow"), Ref identifies the mined assertion, Detail carries the mechanism's
// own fields. The queue does not interpret them; it only refuses an attestation with no Kind
// or no Ref, so a root is never marked witnessed by nothing.
type WitnessAttestation struct {
	Kind   string          `json:"kind"`
	Ref    string          `json:"ref"`
	Detail json.RawMessage `json:"detail,omitempty"`
}

// WitnessEntry is one batch root's place in the queue.
type WitnessEntry struct {
	BatchID     string              `json:"batch_id"`
	Root        string              `json:"merkle_root"`
	Tier        Tier                `json:"tier"`
	QueuedAt    string              `json:"queued_at"`
	NotBefore   string              `json:"not_before"` // not due before this time
	Status      string              `json:"status"`     // pending | witnessed
	WitnessedAt string              `json:"witnessed_at,omitempty"`
	Attestation *WitnessAttestation `json:"attestation,omitempty"`
}

// Witness statuses.
const (
	WitnessPending   = "pending"
	WitnessWitnessed = "witnessed"
)

// DefaultQuarantineWitnessDelay is how long a live-fetched batch waits before it is due:
// reference-tier batches are due at once, quarantine batches on a schedule (plan 5.6).
// Provisional (owner-level): one day.
const DefaultQuarantineWitnessDelay = 24 * time.Hour

// WitnessQueue is the append-only queue file next to the index. The newest line per root wins.
type WitnessQueue struct {
	path string
	now  func() time.Time
	// QuarantineDelay overrides DefaultQuarantineWitnessDelay.
	QuarantineDelay time.Duration

	mu      sync.Mutex
	entries map[string]*WitnessEntry
}

// OpenWitnessQueue loads the queue at path (created on first write).
func OpenWitnessQueue(path string, now func() time.Time) (*WitnessQueue, error) {
	if now == nil {
		now = time.Now
	}
	q := &WitnessQueue{path: path, now: now, QuarantineDelay: DefaultQuarantineWitnessDelay, entries: map[string]*WitnessEntry{}}
	err := readLines(path, func(b []byte) error {
		var e WitnessEntry
		if json.Unmarshal(b, &e) == nil && e.Root != "" {
			q.entries[e.Root] = &e
		}
		return nil
	})
	return q, err
}

func (q *WitnessQueue) appendLocked(e *WitnessEntry) error {
	q.entries[e.Root] = e
	return appendJSONL(q.path, e)
}

// Enqueue adds a batch root, once: a root already queued (or witnessed) is left alone.
func (q *WitnessQueue) Enqueue(batchID, root string, tier Tier) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if _, ok := q.entries[root]; ok {
		return nil
	}
	now := q.now().UTC()
	notBefore := now
	if tier == TierQuarantine {
		notBefore = now.Add(q.QuarantineDelay)
	}
	return q.appendLocked(&WitnessEntry{BatchID: batchID, Root: root, Tier: tier, Status: WitnessPending,
		QueuedAt: now.Format(time.RFC3339), NotBefore: notBefore.Format(time.RFC3339)})
}

// Due lists pending roots whose time has come, oldest first.
func (q *WitnessQueue) Due() []WitnessEntry {
	q.mu.Lock()
	defer q.mu.Unlock()
	now := q.now()
	var out []WitnessEntry
	for _, e := range q.entries {
		if e.Status != WitnessPending {
			continue
		}
		if t, err := time.Parse(time.RFC3339, e.NotBefore); err == nil && t.After(now) {
			continue
		}
		out = append(out, *e)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].QueuedAt != out[j].QueuedAt {
			return out[i].QueuedAt < out[j].QueuedAt
		}
		return out[i].Root < out[j].Root
	})
	return out
}

// MarkWitnessed records that root has been witnessed. The root must already be queued, and
// the attestation must name its mechanism and the assertion it produced.
func (q *WitnessQueue) MarkWitnessed(root string, a WitnessAttestation) error {
	if strings.TrimSpace(a.Kind) == "" || strings.TrimSpace(a.Ref) == "" {
		return errors.New("evidence: an attestation needs a kind and a reference")
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	e, ok := q.entries[root]
	if !ok {
		return fmt.Errorf("evidence: root %s is not in the witness queue", root)
	}
	if e.Status == WitnessWitnessed {
		return nil // already done; the first attestation stands
	}
	cp := *e
	cp.Status, cp.WitnessedAt, cp.Attestation = WitnessWitnessed, q.now().UTC().Format(time.RFC3339), &a
	return q.appendLocked(&cp)
}

// Entry returns the queue entry for a root.
func (q *WitnessQueue) Entry(root string) (WitnessEntry, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if e, ok := q.entries[root]; ok {
		return *e, true
	}
	return WitnessEntry{}, false
}

// WitnessStatus summarises the queue.
type WitnessStatus struct {
	Pending   int    `json:"pending"`
	Witnessed int    `json:"witnessed"`
	Oldest    string `json:"oldest_pending,omitempty"`
}

// Status counts the queue.
func (q *WitnessQueue) Status() WitnessStatus {
	q.mu.Lock()
	defer q.mu.Unlock()
	var s WitnessStatus
	for _, e := range q.entries {
		if e.Status == WitnessWitnessed {
			s.Witnessed++
			continue
		}
		s.Pending++
		if s.Oldest == "" || e.QueuedAt < s.Oldest {
			s.Oldest = e.QueuedAt
		}
	}
	return s
}

// Witnesser turns a batch root into an attestation (the seeder-backed implementation is a
// follow-up; this package ships none, so nothing is ever attested without a real mechanism).
type Witnesser interface {
	Witness(ctx context.Context, e WitnessEntry) (WitnessAttestation, error)
}

// Drain witnesses up to max due roots and returns how many succeeded. A failure leaves the
// root pending for the next pass and is returned after the loop; one bad root does not stop
// the others.
func (q *WitnessQueue) Drain(ctx context.Context, w Witnesser, max int) (int, error) {
	var firstErr error
	done := 0
	for _, e := range q.Due() {
		if max > 0 && done >= max {
			break
		}
		if ctx.Err() != nil {
			return done, ctx.Err()
		}
		a, err := w.Witness(ctx, e)
		if err == nil {
			err = q.MarkWitnessed(e.Root, a)
		}
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("witness %s: %w", e.Root, err)
			}
			continue
		}
		done++
	}
	return done, firstErr
}

// ---- Index integration ----

// Witnesses is the index's witness queue.
func (ix *Index) Witnesses() *WitnessQueue { return ix.witness }

// HitWitness is the provenance record of a cited span.
type HitWitness struct {
	BatchRoot   string             `json:"batch_root"`
	Proof       BatchProof         `json:"proof"`
	Attestation WitnessAttestation `json:"attestation"`
	WitnessedAt string             `json:"witnessed_at,omitempty"`
}

// WitnessOf returns the provenance of a hit: its inclusion proof and the attestation of the
// root it is included in. ok is false unless the proof verifies and the root is witnessed.
func (ix *Index) WitnessOf(h Hit) (HitWitness, bool) {
	p, err := ix.InclusionProof(h.SpanID)
	if err != nil || !VerifyBatchProof(p) {
		return HitWitness{}, false
	}
	e, ok := ix.witness.Entry(p.Root)
	if !ok || e.Status != WitnessWitnessed || e.Attestation == nil {
		return HitWitness{}, false
	}
	return HitWitness{BatchRoot: p.Root, Proof: p, Attestation: *e.Attestation, WitnessedAt: e.WitnessedAt}, true
}

// Attested reports whether a hit has verified provenance: its span is included in a witnessed
// batch root. It is the Witness hook of the verification engine (factslice.Witness).
func (ix *Index) Attested(h Hit) bool {
	_, ok := ix.WitnessOf(h)
	return ok
}

// ---- Background witnessing scheduler ----

// WitnessSchedulerInterval is how often the scheduler drains the witness queue.
const WitnessSchedulerInterval = 30 * time.Second

// WitnessSchedulerMax is the per-tick batch size for Drain.
const WitnessSchedulerMax = 64

// StartWitnessScheduler runs a background goroutine that drains the witness
// queue through w, off the request path, until ctx is cancelled or
// StopWitnessScheduler is called. It is safe to call more than once — only the
// first call starts a loop. The default SeederWitnesser (pkg/evidence) talks to
// the 3_DATA_SEEDER socket (Task 3.9 follow-up).
func (ix *Index) StartWitnessScheduler(ctx context.Context, w Witnesser) {
	if w == nil {
		return
	}
	ix.startWitnessOnce.Do(func() {
		schedCtx, cancel := context.WithCancel(ctx)
		ix.witnessMu.Lock()
		ix.witnessCancel = cancel
		ix.witnessDone = make(chan struct{})
		ix.witnessMu.Unlock()
		go func() {
			defer close(ix.witnessDone)
			ix.witnessSchedulerLoop(schedCtx, w)
		}()
	})
}

// StopWitnessScheduler cancels the background drain loop (if any) and waits for it
// to exit, so callers can tear down without racing the writer. Safe to call more
// than once and when the scheduler never started.
func (ix *Index) StopWitnessScheduler() {
	ix.witnessMu.Lock()
	cancel := ix.witnessCancel
	done := ix.witnessDone
	ix.witnessMu.Unlock()
	if cancel == nil {
		return
	}
	cancel()
	if done != nil {
		<-done
	}
}

func (ix *Index) witnessSchedulerLoop(ctx context.Context, w Witnesser) {
	ticker := time.NewTicker(WitnessSchedulerInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := ix.witness.Drain(ctx, w, WitnessSchedulerMax); err != nil {
				// Non-fatal: the next tick retries. Drain marks individual
				// roots witnessed/errored, so a transient seeder outage does
				// not block the queue.
				_ = err
			}
		}
	}
}
