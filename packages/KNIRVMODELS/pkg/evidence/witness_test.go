package evidence

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func wdocs(prefix string, n int) []Doc {
	var docs []Doc
	for i := 0; i < n; i++ {
		docs = append(docs, Doc{ID: prefix + string(rune('a'+i)), Title: prefix + " " + string(rune('A'+i)),
			Text: "The " + prefix + " number " + string(rune('a'+i)) + " describes the harbour of town " + string(rune('A'+i)) + " in great detail and length."})
	}
	return docs
}

func wopen(t *testing.T) (*Index, string) {
	t.Helper()
	dir := t.TempDir()
	ix, err := Open(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(ix.Close)
	return ix, dir
}

func spansOf(t *testing.T, ix *Index, text string) []Hit {
	t.Helper()
	res, err := ix.Search(Claim{Text: text}, Options{K: 20, Tiers: []Tier{TierReference, TierQuarantine}})
	if err != nil {
		t.Fatal(err)
	}
	return res.Hits
}

func TestEveryIngestedSpanHasAVerifiableInclusionProof(t *testing.T) {
	ix, dir := wopen(t)
	r, err := ix.Ingest(Batch{Tier: TierReference, Docs: wdocs("Atlas", 5)})
	if err != nil {
		t.Fatal(err)
	}
	hits := spansOf(t, ix, "harbour of town")
	if len(hits) < 3 {
		t.Fatalf("%d hits", len(hits))
	}
	for _, h := range hits {
		p, err := ix.InclusionProof(h.SpanID)
		if err != nil || p.Root != r.Root || !VerifyBatchProof(p) {
			t.Fatalf("%v %+v", err, p)
		}
		bad := p
		bad.Root = Digest{9}.String()
		if VerifyBatchProof(bad) {
			t.Fatal("a proof verified against the wrong root")
		}
		// Flipping a step breaks the proof, except where the sibling is the node itself (an odd
		// level duplicates it), which is symmetric; some step in a real tree is not.
		broke := 0
		for i := range p.Steps {
			bad = p
			bad.Steps = append([]ProofStepJSON(nil), p.Steps...)
			bad.Steps[i].Left = !bad.Steps[i].Left
			if !VerifyBatchProof(bad) {
				broke++
			}
		}
		if len(p.Steps) > 1 && broke == 0 {
			t.Fatal("no flipped step broke the proof")
		}
		bad = p
		bad.SpanID = Digest{1}.String()
		if VerifyBatchProof(bad) {
			t.Fatal("a proof verified for a span that is not in the tree")
		}
	}
	if _, err := ix.InclusionProof(Digest{7}.String()); !errors.Is(err, ErrNoBatchRecord) {
		t.Fatalf("an unknown span: %v", err)
	}
	// Leaf lists survive a restart.
	ix.Close()
	ix2, err := Open(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	defer ix2.Close()
	if p, err := ix2.InclusionProof(hits[0].SpanID); err != nil || !VerifyBatchProof(p) {
		t.Fatalf("after reopen: %v", err)
	}
}

// Done-when: rebuilding the index from the same documents reproduces the same hashes, and
// the ledger checks out against the leaf lists.
func TestRebuildingReproducesTheSameRootsAndProofs(t *testing.T) {
	a, _ := wopen(t)
	b, _ := wopen(t)
	ra, err := a.Ingest(Batch{Tier: TierReference, Docs: wdocs("Atlas", 6)})
	if err != nil {
		t.Fatal(err)
	}
	rb, err := b.Ingest(Batch{Tier: TierReference, Docs: wdocs("Atlas", 6)})
	if err != nil {
		t.Fatal(err)
	}
	if ra.Root != rb.Root || ra.BatchID != rb.BatchID {
		t.Fatalf("roots differ: %s %s", ra.Root, rb.Root)
	}
	ha := spansOf(t, a, "harbour of town")
	for _, h := range ha {
		pa, _ := a.InclusionProof(h.SpanID)
		pb, err := b.InclusionProof(h.SpanID)
		if err != nil || pa.Root != pb.Root || pa.Index != pb.Index || len(pa.Steps) != len(pb.Steps) {
			t.Fatalf("proofs differ for %s: %v", h.SpanID, err)
		}
		for i := range pa.Steps {
			if pa.Steps[i] != pb.Steps[i] {
				t.Fatal("proof steps differ")
			}
		}
	}
	if n, err := a.VerifyLedger(); err != nil || n != 1 {
		t.Fatalf("ledger: %d %v", n, err)
	}
}

func TestTamperedLeafListOrLedgerIsDetected(t *testing.T) {
	ix, dir := wopen(t)
	if _, err := ix.Ingest(Batch{Tier: TierReference, Docs: wdocs("Atlas", 4)}); err != nil {
		t.Fatal(err)
	}
	if _, err := ix.VerifyLedger(); err != nil {
		t.Fatal(err)
	}
	files, _ := filepath.Glob(filepath.Join(dir, batchDirName, "*.json"))
	orig, _ := os.ReadFile(files[0])
	var rec BatchRecord
	_ = json.Unmarshal(orig, &rec)
	rec.Leaves[0], rec.Leaves[1] = rec.Leaves[1], rec.Leaves[0] // reorder: same leaves, other root
	b, _ := json.Marshal(rec)
	_ = os.WriteFile(files[0], b, 0o640)
	if _, err := ix.VerifyLedger(); err == nil || !strings.Contains(err.Error(), "leaves hash to") {
		t.Fatalf("a reordered leaf list went unnoticed: %v", err)
	}
	// The proof API refuses to hand out a proof from a record that no longer matches the index.
	if _, err := ix.InclusionProof(rec.Leaves[0]); err == nil {
		t.Fatal("a proof was built from a tampered record")
	}
	_ = os.WriteFile(files[0], orig, 0o640)
	if _, err := ix.VerifyLedger(); err != nil {
		t.Fatalf("restored: %v", err)
	}
	// A ledger line that names another root.
	led := filepath.Join(dir, "evidence_writes.jsonl")
	data, _ := os.ReadFile(led)
	_ = os.WriteFile(led, []byte(strings.Replace(string(data), rec.Root, Digest{3}.String(), 1)), 0o640)
	if _, err := ix.VerifyLedger(); err == nil || !strings.Contains(err.Error(), "ledger says") {
		t.Fatalf("a changed ledger root went unnoticed: %v", err)
	}
}

type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time { return c.t }

func TestIngestQueuesEachBatchAndQuarantineWaitsForItsSchedule(t *testing.T) {
	ix, dir := wopen(t)
	clk := &fakeClock{t: time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)}
	ix.witness.now = clk.now
	rr, _ := ix.Ingest(Batch{Tier: TierReference, Docs: wdocs("Atlas", 3)})
	rq, err := ix.Ingest(Batch{Tier: TierQuarantine, Docs: []Doc{{ID: "w", Title: "Web", URL: "https://www.nasa.gov/x", Text: "A web page about the moon and its distance from the earth in kilometres."}}})
	if err != nil {
		t.Fatal(err)
	}
	due := ix.Witnesses().Due()
	if len(due) != 1 || due[0].Root != rr.Root || due[0].Status != WitnessPending {
		t.Fatalf("only the reference batch is due now: %+v", due)
	}
	clk.t = clk.t.Add(25 * time.Hour)
	if due = ix.Witnesses().Due(); len(due) != 2 {
		t.Fatalf("after the schedule both are due: %+v", due)
	}
	if st := ix.Witnesses().Status(); st.Pending != 2 || st.Witnessed != 0 || st.Oldest == "" {
		t.Fatalf("%+v", st)
	}
	// Enqueue is idempotent.
	if err := ix.Witnesses().Enqueue(rq.BatchID, rq.Root, TierQuarantine); err != nil || ix.Witnesses().Status().Pending != 2 {
		t.Fatalf("%v", err)
	}
	// The queue survives a restart.
	ix.Close()
	ix2, _ := Open(dir, nil2())
	defer ix2.Close()
	if st := ix2.Witnesses().Status(); st.Pending != 2 {
		t.Fatalf("%+v", st)
	}
}

func nil2() string { return "" }

type fakeWitnesser struct {
	fail  map[string]bool
	calls int
}

func (f *fakeWitnesser) Witness(_ context.Context, e WitnessEntry) (WitnessAttestation, error) {
	f.calls++
	if f.fail[e.Root] {
		return WitnessAttestation{}, errors.New("seeder busy")
	}
	return WitnessAttestation{Kind: "test-witness", Ref: "assertion-" + e.BatchID, Detail: json.RawMessage(`{"nonce":7}`)}, nil
}

func TestMarkWitnessedNeedsARealAttestationAndAKnownRoot(t *testing.T) {
	ix, _ := wopen(t)
	r, _ := ix.Ingest(Batch{Tier: TierReference, Docs: wdocs("Atlas", 2)})
	q := ix.Witnesses()
	for _, a := range []WitnessAttestation{{}, {Kind: "x"}, {Ref: "y"}, {Kind: " ", Ref: " "}} {
		if err := q.MarkWitnessed(r.Root, a); err == nil {
			t.Errorf("attestation %+v accepted", a)
		}
	}
	if err := q.MarkWitnessed(Digest{5}.String(), WitnessAttestation{Kind: "k", Ref: "r"}); err == nil {
		t.Error("an unknown root was marked witnessed")
	}
	if err := q.MarkWitnessed(r.Root, WitnessAttestation{Kind: "k", Ref: "first"}); err != nil {
		t.Fatal(err)
	}
	if err := q.MarkWitnessed(r.Root, WitnessAttestation{Kind: "k", Ref: "second"}); err != nil {
		t.Fatal(err)
	}
	if e, _ := q.Entry(r.Root); e.Attestation.Ref != "first" || e.Status != WitnessWitnessed {
		t.Fatalf("the first attestation stands: %+v", e)
	}
}

func TestDrainWitnessesDueRootsAndLeavesFailuresPending(t *testing.T) {
	ix, dir := wopen(t)
	var roots []string
	for _, p := range []string{"Atlas", "Gazetteer", "Almanac"} {
		r, err := ix.Ingest(Batch{Tier: TierReference, Docs: wdocs(p, 2)})
		if err != nil {
			t.Fatal(err)
		}
		roots = append(roots, r.Root)
	}
	w := &fakeWitnesser{fail: map[string]bool{roots[1]: true}}
	n, err := ix.Witnesses().Drain(context.Background(), w, 0)
	if n != 2 || err == nil || !strings.Contains(err.Error(), "seeder busy") {
		t.Fatalf("n=%d err=%v", n, err)
	}
	if st := ix.Witnesses().Status(); st.Witnessed != 2 || st.Pending != 1 {
		t.Fatalf("%+v", st)
	}
	// max bounds one pass.
	w.fail = nil
	ix2, _ := wopen(t)
	for _, p := range []string{"A", "B", "C"} {
		_, _ = ix2.Ingest(Batch{Tier: TierReference, Docs: wdocs(p+"x", 2)})
	}
	if n, err := ix2.Witnesses().Drain(context.Background(), w, 2); n != 2 || err != nil {
		t.Fatalf("%d %v", n, err)
	}
	// The retry succeeds and witnessed state persists.
	if n, err := ix.Witnesses().Drain(context.Background(), w, 0); n != 1 || err != nil {
		t.Fatalf("%d %v", n, err)
	}
	ix.Close()
	ix3, _ := Open(dir, "")
	defer ix3.Close()
	if st := ix3.Witnesses().Status(); st.Witnessed != 3 || st.Pending != 0 {
		t.Fatalf("%+v", st)
	}
	// A cancelled context stops the pass.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	ix4, _ := wopen(t)
	_, _ = ix4.Ingest(Batch{Tier: TierReference, Docs: wdocs("Q", 2)})
	if _, err := ix4.Witnesses().Drain(ctx, w, 0); err == nil {
		t.Fatal("a cancelled context did not stop the drain")
	}
}

// Done-when: a span is attested only once its batch root is witnessed; quarantine-only
// evidence stays unattested until its own batch is.
func TestAttestedNeedsAVerifiedProofToAWitnessedRoot(t *testing.T) {
	ix, _ := wopen(t)
	ref, _ := ix.Ingest(Batch{Tier: TierReference, Docs: wdocs("Atlas", 3)})
	web, err := ix.Ingest(Batch{Tier: TierQuarantine, Docs: []Doc{{ID: "w", Title: "Web", URL: "https://www.nasa.gov/x",
		Text: "A web page about the Moon distance from Earth in kilometres, fetched live."}}})
	if err != nil {
		t.Fatal(err)
	}
	refHit := spansOf(t, ix, "harbour of town")[0]
	var webHit Hit
	for _, h := range spansOf(t, ix, "Moon distance from Earth kilometres") {
		if h.Tier == TierQuarantine {
			webHit = h
		}
	}
	if webHit.SpanID == "" {
		t.Fatal("no quarantine hit")
	}
	if ix.Attested(refHit) || ix.Attested(webHit) {
		t.Fatal("nothing is attested before any root is witnessed")
	}
	// Witness only the reference batch.
	if err := ix.Witnesses().MarkWitnessed(ref.Root, WitnessAttestation{Kind: "test", Ref: "a1"}); err != nil {
		t.Fatal(err)
	}
	if !ix.Attested(refHit) {
		t.Fatal("a span of a witnessed batch must be attested")
	}
	if ix.Attested(webHit) {
		t.Fatal("quarantine evidence is capped until its own batch is witnessed")
	}
	w, ok := ix.WitnessOf(refHit)
	if !ok || w.BatchRoot != ref.Root || w.Attestation.Ref != "a1" || !VerifyBatchProof(w.Proof) || w.WitnessedAt == "" {
		t.Fatalf("%+v", w)
	}
	if err := ix.Witnesses().MarkWitnessed(web.Root, WitnessAttestation{Kind: "test", Ref: "a2"}); err != nil {
		t.Fatal(err)
	}
	if !ix.Attested(webHit) {
		t.Fatal("after its batch is witnessed the web span is attested")
	}
	// A hit that is not a span of this index is never attested.
	if ix.Attested(Hit{SpanID: Digest{8}.String()}) {
		t.Fatal("an unknown span was attested")
	}
}

// TestSpanToBatchMapIsDiskBacked checks Task 3.9 item 5: the span-to-batch map is
// persisted to a single append-only file and reloaded on open (so reopen resolves
// inclusion proofs without re-parsing every batch JSON).
func TestSpanToBatchMapIsDiskBacked(t *testing.T) {
	ix, dir := wopen(t)
	if _, err := ix.Ingest(Batch{Tier: TierReference, Docs: wdocs("Atlas", 4)}); err != nil {
		t.Fatal(err)
	}
	ix.Close()

	data, err := os.ReadFile(filepath.Join(dir, leafIndexFile))
	if err != nil {
		t.Fatalf("disk-backed leaf index not written: %v", err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		t.Fatal("leaf index file is empty")
	}

	ix2, err := Open(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	defer ix2.Close()
	hits := spansOf(t, ix2, "harbour of town")
	if len(hits) == 0 {
		t.Fatal("no spans after reopen")
	}
	if p, err := ix2.InclusionProof(hits[0].SpanID); err != nil || !VerifyBatchProof(p) {
		t.Fatalf("inclusion proof after reopen: %v", err)
	}
}

// TestRebuildFromLedgerReproducesHashes (3.9 done-when): rebuilding the index
// from the same documents must reproduce the identical merkle roots recorded in
// the evidence_writes.jsonl flash ledger, batch id and all.
func TestRebuildFromLedgerReproducesHashes(t *testing.T) {
	ix1, dir1 := wopen(t)
	r1, err := ix1.Ingest(Batch{Tier: TierReference, Docs: wdocs("Atlas", 6)})
	if err != nil {
		t.Fatal(err)
	}
	l1 := readLedgerLine(t, dir1, r1.BatchID)
	if l1 == "" {
		t.Fatal("first ledger has no line for batch")
	}

	ix2, dir2 := wopen(t)
	r2, err := ix2.Ingest(Batch{Tier: TierReference, Docs: wdocs("Atlas", 6)})
	if err != nil {
		t.Fatal(err)
	}
	l2 := readLedgerLine(t, dir2, r2.BatchID)
	if l2 == "" {
		t.Fatal("second ledger has no line for batch")
	}

	if r1.BatchID != r2.BatchID {
		t.Fatalf("rebuild changed batch id: %s != %s", r1.BatchID, r2.BatchID)
	}
	if r1.Root != r2.Root {
		t.Fatalf("rebuild changed merkle root: %s != %s", r1.Root, r2.Root)
	}
	// The flash ledger must record the same merkle root for the same batch across
	// rebuilds (the only field allowed to differ is the ingest timestamp).
	if lr1, lr2 := ledgerRoot(t, l1), ledgerRoot(t, l2); lr1 != r1.Root || lr2 != r2.Root {
		t.Fatalf("ledger roots mismatch: %s / %s", lr1, lr2)
	}
	// And the index itself agrees the batches are intact.
	if n, err := ix2.VerifyLedger(); err != nil || n != 1 {
		t.Fatalf("VerifyLedger after rebuild: %d %v", n, err)
	}
}

func readLedgerLine(t *testing.T, dir, batchID string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "evidence_writes.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if ledgerBatchID(line) == batchID {
			return line
		}
	}
	return ""
}

func ledgerBatchID(line string) string {
	var r struct {
		BatchID string `json:"batch_id"`
	}
	if json.Unmarshal([]byte(line), &r) != nil {
		return ""
	}
	return r.BatchID
}

func ledgerRoot(t *testing.T, line string) string {
	t.Helper()
	var r struct {
		Root string `json:"merkle_root"`
	}
	if err := json.Unmarshal([]byte(line), &r); err != nil {
		t.Fatal(err)
	}
	return r.Root
}

// TestAttestedRequiresWitnessedMinedRoot (3.9 done-when): a span is only
// `attested` when it has a valid inclusion proof AND its batch root has been
// witnessed (a mined root). A quarantine-only batch is enqueued but not due, so
// without a witness the claim is capped below attested.
func TestAttestedRequiresWitnessedMinedRoot(t *testing.T) {
	// Reference batch: included and, once witnessed, attested.
	ref, dir := wopen(t)
	rr, err := ref.Ingest(Batch{Tier: TierReference, Docs: wdocs("Atlas", 4)})
	if err != nil {
		t.Fatal(err)
	}
	refHits := spansOf(t, ref, "harbour of town")
	if len(refHits) == 0 {
		t.Fatal("no reference hits")
	}
	if p, err := ref.InclusionProof(refHits[0].SpanID); err != nil || !VerifyBatchProof(p) {
		t.Fatalf("reference span should have a valid inclusion proof: %v", err)
	}
	if ref.Attested(refHits[0]) {
		t.Fatal("reference span is attested before its root is witnessed")
	}
	if err := ref.Witnesses().MarkWitnessed(rr.Root, WitnessAttestation{Kind: "seeder-pow", Ref: "assertion-1"}); err != nil {
		t.Fatal(err)
	}
	if !ref.Attested(refHits[0]) {
		t.Fatal("reference span should be attested once its root is witnessed")
	}

	// Quarantine batch: included, but queued with a delay so it is not yet due
	// and therefore cannot be witnessed through the normal path; it stays below
	// attested until a witnesser acts after the delay. Quarantine docs must carry
	// the URL they were fetched from.
	q, _ := wopen(t)
	qDocs := []Doc{{
		ID:    "q1",
		Title: "Nightjar a",
		Text:  "The Nightjar number a describes the harbour of town A in great detail and length.",
		URL:   "https://forum.example/nightjar-a",
	}}
	qr, err := q.Ingest(Batch{Tier: TierQuarantine, Docs: qDocs})
	if err != nil {
		t.Fatal(err)
	}
	qHits := spansOf(t, q, "harbour of town")
	if len(qHits) == 0 {
		t.Fatal("no quarantine hits")
	}
	if p, err := q.InclusionProof(qHits[0].SpanID); err != nil || !VerifyBatchProof(p) {
		t.Fatalf("quarantine span should still have a valid inclusion proof: %v", err)
	}
	if due := q.Witnesses().Due(); len(due) != 0 {
		t.Fatalf("quarantine root must not be due yet (delay), got %d due", len(due))
	}
	if q.Attested(qHits[0]) {
		t.Fatal("quarantine-only claim must be capped below attested (no witnessed root)")
	}
	// Only an actual witness (a mined root attestation) lifts it to attested.
	if err := q.Witnesses().MarkWitnessed(qr.Root, WitnessAttestation{Kind: "seeder-pow", Ref: "assertion-2"}); err != nil {
		t.Fatal(err)
	}
	if !q.Attested(qHits[0]) {
		t.Fatal("quarantine span should be attested once its root is witnessed")
	}
	_ = dir
}
