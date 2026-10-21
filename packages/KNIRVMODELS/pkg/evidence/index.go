package evidence

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Tier is how far a source is trusted (plan §4.3, D3). Tiers are never merged:
// every span carries its tier into the verdict, and the gate reads it.
type Tier string

const (
	// TierReference is curated public corpus (Wikipedia, arXiv): shared by every tenant.
	TierReference Tier = "reference"
	// TierUser is a document an authorized user or admin submitted. Visible only to
	// its own tenant, so one tenant's private documents never become another's evidence.
	TierUser Tier = "user"
	// TierQuarantine is write-through from live web fetches. It is excluded from
	// search unless asked for and capped below the attested gate until witnessed.
	TierQuarantine Tier = "quarantine"
	// TierOntology is an organization's own ontology, loaded at onboarding. The tenant is
	// the organization. It is authoritative for that organization: its hits rank above
	// every other tier, and only the organization's current ontology version is visible
	// (see SetOntologyVersion), so a superseded fact is never cited.
	TierOntology Tier = "ontology"
)

func (t Tier) valid() bool {
	return t == TierReference || t == TierUser || t == TierQuarantine || t == TierOntology
}

// tenantScoped tiers are visible only to the tenant that wrote them.
func (t Tier) tenantScoped() bool { return t == TierUser || t == TierOntology }

// Limits that keep one submission from exhausting the host.
const (
	MaxDocBytes     = 8 << 20
	MaxSpansPerDoc  = 20000
	maxPostingsRead = 20000 // a key this common carries no information; skip its postings
	candidatePool   = 400   // spans fetched for filtering before the top-k cut
)

// Header is index.json: what every query must match to be meaningful.
type Header struct {
	KeyVersion   string `json:"key_version"`
	EmbedderID   string `json:"embedder_id"` // "" while the index is lexical-only
	ChunkWords   int    `json:"chunk_words"`
	ChunkOverlap int    `json:"chunk_overlap"`
	Created      string `json:"created"`
}

// Doc is one document to ingest.
type Doc struct {
	ID        string `json:"id,omitempty"`
	Title     string `json:"title,omitempty"`
	Text      string `json:"text"`
	URL       string `json:"url,omitempty"`
	License   string `json:"license,omitempty"`
	FetchedAt string `json:"fetched_at,omitempty"`
	Facts     []Fact `json:"facts,omitempty"`
	// Source and Revision identify a live-fetched page: where it came from and the exact
	// revision (Wikipedia revid, arXiv id+version) the text is, so a verdict can cite it
	// and a later recheck can tell whether it changed.
	Source   string `json:"source,omitempty"`
	Revision string `json:"revision,omitempty"`
}

// Fact is a structured (subject, property, value) triple, indexed under the exact
// K_sp key so a claim with the same subject and property finds it deterministically.
type Fact struct {
	Subject  string `json:"subject"`
	Property string `json:"property"`
	Value    string `json:"value"`
}

// Batch is a set of documents with one tier, tenant and provenance.
type Batch struct {
	Tier   Tier
	Tenant string // required for TierUser
	Docs   []Doc
}

// BatchResult describes what an ingest wrote.
type BatchResult struct {
	BatchID    string `json:"batch_id"`
	Segment    string `json:"segment"`
	Docs       int    `json:"docs"`
	Skipped    int    `json:"skipped_duplicates"`
	Spans      int    `json:"spans"`
	Root       string `json:"merkle_root"`
	IndexVer   string `json:"index_version"`
	Unreadable int    `json:"empty_docs"`
}

type segMeta struct {
	// Roots are the Merkle roots of every ingest batch the segment holds; a merged
	// segment lists all of its batches. The index version hashes them, so compaction
	// (which changes the segment layout, not the content) does not change it.
	Roots   []string     `json:"roots,omitempty"`
	BatchID string       `json:"batch_id"`
	Spans   int          `json:"spans"`
	Tiers   map[Tier]int `json:"tiers"`
	Root    string       `json:"merkle_root"`
	Created string       `json:"created"`
}

func (m segMeta) roots() []string {
	if len(m.Roots) > 0 {
		return m.Roots
	}
	if m.Root != "" {
		return []string{m.Root}
	}
	return nil
}

// Index is the opened evidence index.
type Index struct {
	dir   string
	store *Store

	mu     sync.RWMutex
	header Header
	segs   []*segment
	metas  []segMeta
	seen   map[string]bool   // snapshot|tier|tenant already ingested
	ontVer map[string]string // tenant -> the ontology version that is current

	searches, withHits atomic.Uint64
}

// Open opens or creates an index. embedderID is pinned in a new index's header; an
// existing index keeps the one it was built with, and a mismatch is an error: a
// vector from one embedder is never compared with one from another (E1).
func Open(dir, embedderID string) (*Index, error) {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, err
	}
	st, err := OpenStore(filepath.Join(dir, "snapshots"))
	if err != nil {
		return nil, err
	}
	ix := &Index{dir: dir, store: st, seen: map[string]bool{}, ontVer: map[string]string{}}
	ix.loadOntologyVersions()
	hp := filepath.Join(dir, "index.json")
	if b, err := os.ReadFile(hp); err == nil {
		if err := json.Unmarshal(b, &ix.header); err != nil {
			return nil, fmt.Errorf("evidence: bad index.json: %w", err)
		}
		if ix.header.KeyVersion != KeyVersion {
			return nil, fmt.Errorf("evidence: index uses key version %q, this build uses %q: rebuild the index", ix.header.KeyVersion, KeyVersion)
		}
		if embedderID != "" && ix.header.EmbedderID != embedderID {
			return nil, fmt.Errorf("evidence: index embedder is %q, asked for %q: vectors from different embedders are never mixed", ix.header.EmbedderID, embedderID)
		}
	} else if os.IsNotExist(err) {
		ix.header = Header{KeyVersion: KeyVersion, EmbedderID: embedderID, ChunkWords: DefaultChunkWords,
			ChunkOverlap: DefaultChunkOverlap, Created: time.Now().UTC().Format(time.RFC3339)}
		b, _ := json.MarshalIndent(ix.header, "", " ")
		if err := writeFileSync(hp, b); err != nil {
			return nil, err
		}
	} else {
		return nil, err
	}
	names, _ := filepath.Glob(filepath.Join(dir, "segments", "seg-*"))
	sort.Strings(names)
	for _, n := range names {
		s, err := openSegment(n)
		if err != nil {
			continue // half-written build: ignored
		}
		ix.segs = append(ix.segs, s)
		var m segMeta
		if b, err := os.ReadFile(filepath.Join(n, "meta.json")); err == nil {
			_ = json.Unmarshal(b, &m)
		}
		ix.metas = append(ix.metas, m)
	}
	err = readLines(filepath.Join(dir, "docs.jsonl"), func(b []byte) error {
		var r struct {
			Key string `json:"key"`
		}
		if json.Unmarshal(b, &r) == nil {
			ix.seen[r.Key] = true
		}
		return nil
	})
	return ix, err
}

// Close releases the segment files.
func (ix *Index) Close() {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	for _, s := range ix.segs {
		s.close()
	}
	ix.segs = nil
}

// Store exposes the snapshot store (for verifying a cited span).
func (ix *Index) Store() *Store { return ix.store }

// Ingest indexes a batch into a new immutable segment and records it in the
// ledger. A document already ingested (same bytes, tier and tenant) is skipped.
func (ix *Index) Ingest(b Batch) (BatchResult, error) {
	var res BatchResult
	if !b.Tier.valid() {
		return res, fmt.Errorf("evidence: unknown tier %q", b.Tier)
	}
	if b.Tier.tenantScoped() && strings.TrimSpace(b.Tenant) == "" {
		return res, fmt.Errorf("evidence: %s-tier documents need a tenant", b.Tier)
	}
	if !b.Tier.tenantScoped() {
		b.Tenant = "" // only user and ontology documents are tenant scoped
	}
	if b.Tier == TierOntology {
		for _, d := range b.Docs {
			if d.Revision == "" {
				return res, errors.New("evidence: ontology documents need the ontology version as their revision")
			}
		}
	}
	if b.Tier == TierQuarantine {
		for _, d := range b.Docs {
			if d.URL == "" {
				return res, errors.New("evidence: quarantine documents need the URL they were fetched from")
			}
		}
	}
	ix.mu.Lock()
	defer ix.mu.Unlock()

	type pending struct {
		rec  SpanRec
		text string
		feat Features
		fact *Fact
	}
	var spans []pending
	var ledgerDocs []map[string]any
	df := map[string]int{}
	newSeen := map[string]bool{}

	for _, d := range b.Docs {
		if len(d.Text) > MaxDocBytes {
			return res, fmt.Errorf("evidence: document %q is %d bytes, limit %d", d.ID, len(d.Text), MaxDocBytes)
		}
		norm := NormalizeText(d.Text)
		type unit struct {
			snap Digest
			text string
			fact *Fact
		}
		var units []unit
		if norm != "" {
			snap, err := ix.store.Put([]byte(norm))
			if err != nil {
				return res, err
			}
			units = append(units, unit{snap: snap, text: norm})
		}
		for i := range d.Facts {
			f := d.Facts[i]
			if strings.TrimSpace(f.Subject) == "" || strings.TrimSpace(f.Property) == "" {
				continue
			}
			ft := NormalizeText(f.Subject + " — " + f.Property + ": " + f.Value)
			snap, err := ix.store.Put([]byte(ft))
			if err != nil {
				return res, err
			}
			units = append(units, unit{snap: snap, text: ft, fact: &f})
		}
		if len(units) == 0 {
			res.Unreadable++
			continue
		}
		dupKey := units[0].snap.Hex() + "|" + string(b.Tier) + "|" + b.Tenant
		if b.Tier == TierOntology {
			// The same text under a new ontology version is a new document: the old
			// version's spans stop being visible when the version changes.
			dupKey += "|" + d.Revision
		}
		if ix.seen[dupKey] || newSeen[dupKey] {
			res.Skipped++
			continue
		}
		newSeen[dupKey] = true
		res.Docs++
		ledgerDocs = append(ledgerDocs, map[string]any{"snapshot": units[0].snap.String(), "url": d.URL, "license": d.License, "bytes": len(norm)})
		for _, u := range units {
			var cs []Span
			if u.fact != nil {
				cs = []Span{{0, len(u.text)}}
			} else {
				cs = ChunkWords(u.text, ix.header.ChunkWords, ix.header.ChunkOverlap)
			}
			if len(cs) > MaxSpansPerDoc {
				return res, fmt.Errorf("evidence: document %q has %d spans, limit %d", d.ID, len(cs), MaxSpansPerDoc)
			}
			for _, c := range cs {
				rec := SpanRec{ID: SpanID(u.snap, c.Start, c.End).String(), Snapshot: u.snap.String(), Start: c.Start, End: c.End,
					Tier: b.Tier, Tenant: b.Tenant, Title: d.Title, URL: d.URL, License: d.License, FetchedAt: d.FetchedAt, Fact: u.fact != nil,
					Source: d.Source, Revision: d.Revision}
				spans = append(spans, pending{rec: rec, text: u.text[c.Start:c.End], fact: u.fact})
			}
		}
	}
	if len(spans) == 0 {
		res.IndexVer = ix.versionLocked()
		return res, nil
	}

	// Feature extraction is independent per span: spread it over the cores. The
	// document-frequency table is the one serial step between the two passes.
	parallel(len(spans), func(i int) { spans[i].feat = ExtractFeatures(spans[i].text) })
	for i := range spans {
		for _, t := range spans[i].feat.Terms {
			df[t]++
		}
	}
	recs := make([]SpanRec, len(spans))
	keys := make([][]WeightedKey, len(spans))
	parallel(len(spans), func(i int) {
		ks := ChunkKeys(spans[i].feat, df)
		if k, ok := TitleKey(spans[i].rec.Title); ok {
			ks = append(ks, WeightedKey{k, KindTitle})
		}
		if spans[i].fact != nil {
			ks = append(ks, WeightedKey{FactKey(spans[i].fact.Subject, spans[i].fact.Property), KindSubjectProperty})
		}
		keys[i] = ks
	})
	ids := make([]Digest, len(spans))
	tiers := map[Tier]int{}
	for i, p := range spans {
		recs[i] = p.rec
		d, _ := ParseDigest(p.rec.ID)
		ids[i] = d
		tiers[p.rec.Tier]++
	}
	root := MerkleRoot(ids)
	batchID := batchIDOf(ids)
	segName := ix.nextSegName()
	segDir := filepath.Join(ix.dir, "segments", segName)
	if err := os.MkdirAll(segDir, 0o750); err != nil {
		return res, err
	}
	meta := segMeta{Roots: []string{root.String()}, BatchID: batchID, Spans: len(recs), Tiers: tiers, Root: root.String(), Created: time.Now().UTC().Format(time.RFC3339)}
	mb, _ := json.Marshal(meta)
	if err := writeFileSync(filepath.Join(segDir, "meta.json"), mb); err != nil {
		return res, err
	}
	if err := writeSegment(segDir, recs, keys); err != nil {
		return res, err
	}
	seg, err := openSegment(segDir)
	if err != nil {
		return res, err
	}
	ix.segs = append(ix.segs, seg)
	ix.metas = append(ix.metas, meta)
	for k := range newSeen {
		ix.seen[k] = true
	}
	if err := ix.appendRegistry(newSeen); err != nil {
		return res, err
	}
	for len(ix.segs) > maxSegments {
		if err := ix.compactLocked(compactGroup); err != nil {
			return res, fmt.Errorf("evidence: compaction failed (the batch itself is indexed): %w", err)
		}
	}
	res.BatchID, res.Segment, res.Spans, res.Root, res.IndexVer = batchID, segName, len(recs), root.String(), ix.versionLocked()
	rec := map[string]any{"time": meta.Created, "batch_id": batchID, "tier": b.Tier, "tenant": b.Tenant, "segment": segName,
		"spans": len(recs), "merkle_root": res.Root, "index_version": res.IndexVer, "docs": ledgerDocs}
	return res, appendJSONL(filepath.Join(ix.dir, "evidence_writes.jsonl"), rec)
}

func batchIDOf(ids []Digest) string {
	h := sha256.New()
	for _, d := range ids {
		h.Write(d[:])
	}
	return "b-" + hex.EncodeToString(h.Sum(nil))[:16]
}

func (ix *Index) appendRegistry(keys map[string]bool) error {
	f, err := os.OpenFile(filepath.Join(ix.dir, "docs.jsonl"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o640)
	if err != nil {
		return err
	}
	defer f.Close()
	for k := range keys {
		line, _ := json.Marshal(map[string]string{"key": k})
		if _, err := f.Write(append(line, '\n')); err != nil {
			return err
		}
	}
	return f.Sync()
}

func appendJSONL(path string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o640)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(append(b, '\n')); err != nil {
		return err
	}
	return f.Sync()
}

// versionLocked names the index by what it contains: the sorted Merkle roots of its
// ingest batches. It changes when a batch is added; it does not change when segments
// are merged, and two machines holding the same batches report the same version.
func (ix *Index) versionLocked() string {
	var roots []string
	for _, m := range ix.metas {
		roots = append(roots, m.roots()...)
	}
	sort.Strings(roots)
	h := sha256.New()
	for _, r := range roots {
		h.Write([]byte(r))
	}
	return fmt.Sprintf("%s-%d-%s", KeyVersion, len(roots), hex.EncodeToString(h.Sum(nil))[:8])
}

// Version is the current index version.
func (ix *Index) Version() string {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	return ix.versionLocked()
}

// Hit is one retrieved passage.
type Hit struct {
	SpanID    string   `json:"span_id"`
	Snapshot  string   `json:"snapshot"`
	Start     int      `json:"start"`
	End       int      `json:"end"`
	Text      string   `json:"text"`
	Tier      Tier     `json:"source_tier"`
	Title     string   `json:"title,omitempty"`
	URL       string   `json:"url,omitempty"`
	License   string   `json:"license,omitempty"`
	FetchedAt string   `json:"fetched_at,omitempty"`
	Score     float64  `json:"score"`
	Matched   []string `json:"matched"` // key families that matched
	Fact      bool     `json:"fact,omitempty"`
	Source    string   `json:"source,omitempty"`
	Revision  string   `json:"revision,omitempty"`
}

// Options scope a search.
type Options struct {
	K      int
	Tenant string // visibility of user-tier spans
	Tiers  []Tier // default: reference and user. Quarantine only when listed.
}

// Result is a search outcome with the facts a verdict must record.
type Result struct {
	Hits       []Hit  `json:"hits"`
	Retrieval  string `json:"retrieval"` // "lexical": no embedder is pinned, so no vector search ran
	IndexVer   string `json:"index_version"`
	EmbedderID string `json:"embedder_id"`
}

var kindWeight = map[Kind]float64{KindTitle: 6, KindSubjectProperty: 8, KindEntity: 3, KindNumber: 2, KindMinHash: 2.5, KindTerm: 1.5}

type cand struct {
	seg   int
	ord   uint32
	score float64
	kinds map[Kind]bool
}

// Search finds passages for a claim: the union of its key postings, scored by the
// rarity of the keys they share, filtered by tier and tenant, cut to K.
func (ix *Index) Search(c Claim, o Options) (Result, error) {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	ix.searches.Add(1)
	if o.K <= 0 {
		o.K = 8
	}
	tiers := o.Tiers
	if len(tiers) == 0 {
		tiers = []Tier{TierReference, TierUser, TierOntology}
	}
	allow := map[Tier]bool{}
	for _, t := range tiers {
		allow[t] = true
	}
	res := Result{Hits: []Hit{}, Retrieval: "lexical", IndexVer: ix.versionLocked(), EmbedderID: ix.header.EmbedderID}
	total := 0
	for _, m := range ix.metas {
		total += m.Spans
	}
	if total == 0 {
		return res, nil
	}
	scores := map[[2]uint64]*cand{}
	for _, wk := range ClaimKeys(c) {
		var counts []int
		var lists [][]uint32
		sum := 0
		for _, s := range ix.segs {
			n, err := s.count(wk.Key)
			if err != nil {
				return res, err
			}
			counts = append(counts, n)
			sum += n
		}
		if sum == 0 || sum > maxPostingsRead {
			continue
		}
		idf := math.Log(1 + float64(total)/float64(sum))
		w := kindWeight[wk.Kind] * idf
		for si, s := range ix.segs {
			if counts[si] == 0 {
				lists = append(lists, nil)
				continue
			}
			ords, err := s.lookup(wk.Key)
			if err != nil {
				return res, err
			}
			lists = append(lists, ords)
		}
		for si, ords := range lists {
			for _, ord := range ords {
				id := [2]uint64{uint64(si), uint64(ord)}
				cd := scores[id]
				if cd == nil {
					cd = &cand{seg: si, ord: ord, kinds: map[Kind]bool{}}
					scores[id] = cd
				}
				cd.score += w
				cd.kinds[wk.Kind] = true
			}
		}
	}
	pool := make([]*cand, 0, len(scores))
	for _, cd := range scores {
		pool = append(pool, cd)
	}
	sort.Slice(pool, func(i, j int) bool {
		if pool[i].score != pool[j].score {
			return pool[i].score > pool[j].score
		}
		if pool[i].seg != pool[j].seg {
			return pool[i].seg < pool[j].seg
		}
		return pool[i].ord < pool[j].ord
	})
	if len(pool) > candidatePool {
		pool = pool[:candidatePool]
	}
	for _, cd := range pool {
		if len(res.Hits) >= rerankPool {
			break
		}
		rec, err := ix.segs[cd.seg].span(cd.ord)
		if err != nil {
			return res, err
		}
		if !allow[rec.Tier] || rec.Tier.tenantScoped() && rec.Tenant != o.Tenant {
			continue
		}
		if rec.Tier == TierOntology && rec.Revision != ix.ontVer[rec.Tenant] {
			continue // a superseded (or never-activated) ontology version
		}
		snap, err := ParseDigest(rec.Snapshot)
		if err != nil {
			return res, err
		}
		text, err := ix.store.ReadSpan(snap, rec.Start, rec.End)
		if err != nil {
			return res, err
		}
		var kinds []string
		for k := range cd.kinds {
			kinds = append(kinds, string(k))
		}
		sort.Strings(kinds)
		res.Hits = append(res.Hits, Hit{SpanID: rec.ID, Snapshot: rec.Snapshot, Start: rec.Start, End: rec.End, Text: text,
			Tier: rec.Tier, Title: rec.Title, URL: rec.URL, License: rec.License, FetchedAt: rec.FetchedAt,
			Score: math.Round(cd.score*1000) / 1000, Matched: kinds, Fact: rec.Fact, Source: rec.Source, Revision: rec.Revision})
	}
	res.Hits = rerank(c, res.Hits)
	// The organization's own ontology outranks every other source (stable otherwise).
	sort.SliceStable(res.Hits, func(i, j int) bool {
		return res.Hits[i].Tier == TierOntology && res.Hits[j].Tier != TierOntology
	})
	if len(res.Hits) > o.K {
		res.Hits = res.Hits[:o.K]
	}
	if len(res.Hits) > 0 {
		ix.withHits.Add(1)
	}
	return res, nil
}

// Status is GET /models/hasher/index/status.
type Status struct {
	Version     string       `json:"index_version"`
	KeyVersion  string       `json:"key_version"`
	EmbedderIDs []string     `json:"embedder_ids"`
	Retrieval   string       `json:"retrieval"`
	Segments    int          `json:"segments"`
	Spans       int          `json:"spans"`
	Tiers       map[Tier]int `json:"tiers"`
	ChunkWords  int          `json:"chunk_words"`
	Searches    uint64       `json:"searches"`
	HitRate     float64      `json:"hit_rate"`
	LastBatch   string       `json:"last_batch,omitempty"`
	LastIngest  string       `json:"last_ingest,omitempty"`
}

// Status reports the index without reading any postings.
func (ix *Index) Status() Status {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	st := Status{Version: ix.versionLocked(), KeyVersion: ix.header.KeyVersion, Retrieval: "lexical", Segments: len(ix.segs),
		Tiers: map[Tier]int{}, ChunkWords: ix.header.ChunkWords, EmbedderIDs: []string{}}
	if ix.header.EmbedderID != "" {
		// Pinned for when vector search lands; until then every query is lexical.
		st.EmbedderIDs = []string{ix.header.EmbedderID}
	}
	for _, m := range ix.metas {
		st.Spans += m.Spans
		for t, n := range m.Tiers {
			st.Tiers[t] += n
		}
		st.LastBatch, st.LastIngest = m.BatchID, m.Created
	}
	st.Searches = ix.searches.Load()
	if st.Searches > 0 {
		st.HitRate = math.Round(float64(ix.withHits.Load())/float64(st.Searches)*1000) / 1000
	}
	return st
}

// parallel runs f(0..n-1) on up to GOMAXPROCS goroutines.
func parallel(n int, f func(i int)) {
	w := runtime.GOMAXPROCS(0)
	if w > n {
		w = n
	}
	if w <= 1 {
		for i := 0; i < n; i++ {
			f(i)
		}
		return
	}
	var wg sync.WaitGroup
	next := atomic.Int64{}
	for g := 0; g < w; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				i := int(next.Add(1)) - 1
				if i >= n {
					return
				}
				f(i)
			}
		}()
	}
	wg.Wait()
}

// nextSegName is one past the highest existing number. Compaction leaves gaps, so
// counting segments would eventually reuse a live name.
func (ix *Index) nextSegName() string {
	max := 0
	for _, s := range ix.segs {
		var n int
		fmt.Sscanf(s.name, "seg-%d", &n)
		if n > max {
			max = n
		}
	}
	return fmt.Sprintf("seg-%06d", max+1)
}
