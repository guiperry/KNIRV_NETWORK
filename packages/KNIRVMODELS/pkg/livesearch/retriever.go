package livesearch

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"knirvmodels/pkg/evidence"
)

// DefaultTTL is how long a fetched page is served from the index without asking the
// source whether it changed. After it, the page's revision is checked live: unchanged
// pages are kept (and the clock restarts), changed pages are fetched again.
const DefaultTTL = 30 * 24 * time.Hour

// Mode selects where evidence may come from.
type Mode string

const (
	ModeIndex  Mode = "index"  // the index only; never touches the network
	ModeFlash  Mode = "flash"  // always consult the live providers
	ModeHybrid Mode = "hybrid" // the index, and the live providers when the index has no page about the claim
)

// Config for a Retriever.
type Config struct {
	TTL      time.Duration
	MaxPages int // pages fetched per provider per claim
	Now      func() time.Time
	// Enabled is the deployment switch. When false every mode behaves as "index".
	Enabled bool
}

// Retriever answers "what could speak to this claim?" from the index first and the live
// providers when needed, and keeps the index current.
type Retriever struct {
	ix        *evidence.Index
	providers []Provider
	cfg       Config
	statePath string

	mu      sync.Mutex
	entries map[string]entry // by page URL
}

// entry is what the retriever knows about one fetched page: the revision it read, which
// snapshot holds that text, and when the source was last asked. The newest line per URL wins.
type entry struct {
	Source    string `json:"source"`
	ID        string `json:"id"`
	Title     string `json:"title"`
	URL       string `json:"url"`
	Revision  string `json:"revision"`
	Snapshot  string `json:"snapshot"`
	FetchedAt string `json:"fetched_at"`
	CheckedAt string `json:"checked_at"`
}

// NewRetriever keeps its page table next to the index (live_pages.jsonl).
func NewRetriever(ix *evidence.Index, indexDir string, providers []Provider, cfg Config) (*Retriever, error) {
	if cfg.TTL <= 0 {
		cfg.TTL = DefaultTTL
	}
	if cfg.MaxPages <= 0 {
		cfg.MaxPages = 3
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	r := &Retriever{ix: ix, providers: providers, cfg: cfg, statePath: filepath.Join(indexDir, "live_pages.jsonl"), entries: map[string]entry{}}
	f, err := os.Open(r.statePath)
	if err == nil {
		defer f.Close()
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 1<<20), 16<<20)
		for sc.Scan() {
			var e entry
			if json.Unmarshal(sc.Bytes(), &e) == nil && e.URL != "" {
				r.entries[e.URL] = e
			}
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	return r, nil
}

func (r *Retriever) save(e entry) error {
	r.entries[e.URL] = e
	b, _ := json.Marshal(e)
	f, err := os.OpenFile(r.statePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o640)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(append(b, '\n')); err != nil {
		return err
	}
	return f.Sync()
}

// Options scope a retrieval.
type Options struct {
	K      int
	Tenant string
	Mode   Mode
}

// Result is a retrieval with everything a verdict has to record about how the evidence
// was obtained.
type Result struct {
	evidence.Result
	Mode      Mode     `json:"mode"`
	FlashUsed bool     `json:"flash_used"`
	Queries   []string `json:"outbound_queries,omitempty"` // exactly what left the machine
	Degraded  bool     `json:"degraded"`
	Reasons   []string `json:"degraded_reasons,omitempty"`
	Stale     bool     `json:"stale,omitempty"` // some evidence is older than the TTL and could not be rechecked
	Fetched   int      `json:"pages_fetched"`
	Rechecked int      `json:"pages_rechecked"`
	Refetched int      `json:"pages_refetched"`
}

var researchCues = []string{"paper", "study", "arxiv", "algorithm", "neural", "network", "theorem", "model", "dataset", "benchmark",
	"transformer", "learning", "proof", "quantum", "researchers", "published"}

func looksLikeResearch(s string) bool {
	s = strings.ToLower(s)
	for _, c := range researchCues {
		if strings.Contains(s, c) {
			return true
		}
	}
	return false
}

// Retrieve gathers evidence for a claim.
func (r *Retriever) Retrieve(ctx context.Context, c evidence.Claim, o Options) (Result, error) {
	if o.K <= 0 {
		o.K = 8
	}
	if o.Mode == "" {
		o.Mode = ModeHybrid
	}
	if !r.cfg.Enabled || len(r.providers) == 0 {
		o.Mode = ModeIndex
	}
	res := Result{Mode: o.Mode}
	tiers := []evidence.Tier{evidence.TierReference, evidence.TierUser, evidence.TierOntology, evidence.TierQuarantine}
	search := func() (evidence.Result, error) {
		er, err := r.ix.Search(c, evidence.Options{K: o.K * 3, Tenant: o.Tenant, Tiers: tiers})
		if err != nil {
			return er, err
		}
		er.Hits = r.currentOnly(er.Hits)
		return er, nil
	}
	first, err := search()
	if err != nil {
		return res, err
	}
	res.Result = first
	if o.Mode == ModeIndex {
		r.finish(&res, o.K)
		return res, nil
	}

	// Cached live pages older than the TTL are checked against the source.
	r.recheck(ctx, &res)

	need := o.Mode == ModeFlash || !aboutTheClaim(c, res.Hits)
	if need {
		r.goLive(ctx, c, &res)
	}
	if res.FlashUsed || res.Rechecked > 0 || res.Refetched > 0 {
		again, err := search()
		if err != nil {
			return res, err
		}
		res.Hits, res.IndexVer = again.Hits, again.IndexVer
	}
	r.finish(&res, o.K)
	return res, nil
}

func (r *Retriever) finish(res *Result, k int) {
	if len(res.Hits) > k {
		res.Hits = res.Hits[:k]
	}
	if res.Hits == nil {
		res.Hits = []evidence.Hit{}
	}
	if res.FlashUsed {
		res.Retrieval = "live+lexical"
	}
	res.Degraded = len(res.Reasons) > 0
}

// currentOnly drops hits from a page version that has since been replaced: the page's
// current snapshot is the only one that may be cited.
func (r *Retriever) currentOnly(hits []evidence.Hit) []evidence.Hit {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := hits[:0:0]
	for _, h := range hits {
		if h.Source != "" {
			if e, ok := r.entries[h.URL]; ok && e.Snapshot != h.Snapshot {
				continue
			}
		}
		out = append(out, h)
	}
	return out
}

// aboutTheClaim is true when some hit is an article whose title is named in the claim:
// the index already holds a page about the thing the claim is about.
func aboutTheClaim(c evidence.Claim, hits []evidence.Hit) bool {
	claim := " " + evidence.Canon(c.Text+" "+c.Subject) + " "
	for _, h := range hits {
		if t := evidence.Canon(h.Title); t != "" && strings.Contains(claim, " "+t+" ") {
			return true
		}
	}
	return false
}

func (r *Retriever) provider(name string) Provider {
	for _, p := range r.providers {
		if p.Name() == name {
			return p
		}
	}
	return nil
}

// recheck applies the TTL to the live pages among the hits.
func (r *Retriever) recheck(ctx context.Context, res *Result) {
	now := r.cfg.Now()
	r.mu.Lock()
	stale := map[string][]entry{}
	seen := map[string]bool{}
	for _, h := range res.Hits {
		if h.Source == "" || seen[h.URL] {
			continue
		}
		seen[h.URL] = true
		e, ok := r.entries[h.URL]
		if !ok {
			continue
		}
		if t, err := time.Parse(time.RFC3339, e.CheckedAt); err != nil || now.Sub(t) >= r.cfg.TTL {
			stale[e.Source] = append(stale[e.Source], e)
		}
	}
	r.mu.Unlock()
	for src, es := range stale {
		p := r.provider(src)
		if p == nil {
			continue
		}
		var refs []Ref
		for _, e := range es {
			refs = append(refs, Ref{Source: e.Source, ID: e.ID, Title: e.Title, URL: e.URL})
		}
		revs, err := p.Revisions(ctx, refs)
		if err != nil {
			res.Stale = true
			res.Reasons = append(res.Reasons, fmt.Sprintf("could not recheck %d %s page(s) older than %s: %v", len(es), src, r.cfg.TTL, err))
			continue
		}
		var changed []Ref
		for _, e := range es {
			rev, ok := revs[e.ID]
			switch {
			case !ok:
				// Gone or renamed: keep what we have, but say it could not be confirmed.
				res.Stale = true
				res.Reasons = append(res.Reasons, fmt.Sprintf("%s page %q was not found on recheck", src, e.Title))
			case rev == e.Revision:
				e.CheckedAt = now.UTC().Format(time.RFC3339)
				r.mu.Lock()
				_ = r.save(e)
				r.mu.Unlock()
				res.Rechecked++
			default:
				changed = append(changed, Ref{Source: e.Source, ID: e.ID, Title: e.Title, URL: e.URL})
			}
		}
		if len(changed) > 0 {
			n := r.fetchAndStore(ctx, p, changed, res)
			res.Refetched += n
		}
	}
}

// goLive searches the providers with the minimized claim and stores what it fetches.
func (r *Retriever) goLive(ctx context.Context, c evidence.Claim, res *Result) {
	q, err := Minimize(c)
	if err != nil {
		res.Reasons = append(res.Reasons, err.Error())
		return
	}
	foundPages := false
	for _, p := range r.providers {
		// Wikipedia is the default. arXiv is asked for research-flavoured claims, or when
		// nothing earlier found a page: no point sending a claim about a tower to arXiv.
		if p.Name() == "arxiv" && !looksLikeResearch(c.Text) && (len(res.Hits) > 0 || foundPages) {
			continue
		}
		res.Queries = append(res.Queries, p.Name()+": "+q)
		refs, err := p.Search(ctx, q, r.cfg.MaxPages)
		if err != nil {
			res.Reasons = append(res.Reasons, fmt.Sprintf("%s search failed: %v", p.Name(), err))
			continue
		}
		res.FlashUsed = true
		if len(refs) > 0 {
			foundPages = true
		}
		var need []Ref
		now := r.cfg.Now()
		r.mu.Lock()
		for _, ref := range refs {
			if e, ok := r.entries[ref.URL]; ok {
				if t, err := time.Parse(time.RFC3339, e.CheckedAt); err == nil && now.Sub(t) < r.cfg.TTL {
					continue // already held and fresh
				}
			}
			need = append(need, ref)
		}
		r.mu.Unlock()
		if len(need) > 0 {
			res.Fetched += r.fetchAndStore(ctx, p, need, res)
		}
	}
}

// fetchAndStore fetches pages, writes them to the quarantine tier in one batch, and
// records the revision and snapshot of each. It returns how many pages were stored.
func (r *Retriever) fetchAndStore(ctx context.Context, p Provider, refs []Ref, res *Result) int {
	var docs []evidence.Doc
	var pages []Page
	for _, ref := range refs {
		pg, err := p.Fetch(ctx, ref)
		if err != nil {
			res.Reasons = append(res.Reasons, fmt.Sprintf("%s fetch of %q failed: %v", p.Name(), ref.Title, err))
			continue
		}
		pages = append(pages, pg)
		docs = append(docs, evidence.Doc{ID: pg.Revision, Title: pg.Title, Text: pg.Text, URL: pg.URL, License: pg.License,
			FetchedAt: pg.FetchedAt, Source: pg.Source, Revision: pg.Revision})
	}
	if len(docs) == 0 {
		return 0
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, err := r.ix.Ingest(evidence.Batch{Tier: evidence.TierQuarantine, Docs: docs}); err != nil {
		res.Reasons = append(res.Reasons, fmt.Sprintf("could not store fetched pages: %v", err))
		return 0
	}
	now := r.cfg.Now().UTC().Format(time.RFC3339)
	for _, pg := range pages {
		snap := evidence.SnapshotID([]byte(evidence.NormalizeText(pg.Text)))
		_ = r.save(entry{Source: pg.Source, ID: pg.ID, Title: pg.Title, URL: pg.URL, Revision: pg.Revision,
			Snapshot: snap.String(), FetchedAt: pg.FetchedAt, CheckedAt: now})
	}
	return len(pages)
}
