package factslice

import (
	"context"
	"fmt"
	"strings"

	"knirvmodels/pkg/evidence"
	"knirvmodels/pkg/livesearch"
	"knirvmodels/pkg/orgontology"
)

// Request is one text to verify.
type Request struct {
	Text   string
	Tenant string // visibility of user-tier and ontology-tier spans
	K      int    // spans retrieved per claim, default 8
	Tiers  []evidence.Tier
	// Mode is the retrieval mode: "index" (the default), "flash" or "hybrid". The last two
	// consult Engine.Live when one is set and live lookup is enabled there; otherwise the
	// claim is answered from the index.
	Mode string
}

// LiveSource retrieves evidence that may come from live providers, writing what it
// fetches to the quarantine tier. *livesearch.Retriever is the production one.
type LiveSource interface {
	Retrieve(ctx context.Context, c evidence.Claim, o livesearch.Options) (livesearch.Result, error)
}

// Witness reports whether a span has a verified provenance proof (a Merkle inclusion
// proof against a witnessed batch root, Task 3.9). Nil means none do, so no claim
// can reach "attested" yet.
type Witness func(h evidence.Hit) bool

// Engine is the verification pipeline: extract, retrieve, check, judge, calibrate, gate.
type Engine struct {
	Index      *evidence.Index
	Extractor  *Extractor
	Judge      *Judge
	Cal        *Calibrator // nil or unfitted: the identity, and the gate is capped at "seen"
	Thresholds Thresholds  // zero value: DefaultThresholds
	Witness    Witness

	// Cache, when set, holds finished claim results. CacheScope identifies what
	// else the result depends on: the judge model file hash, the adapter-set hash and
	// the judge frame hash (an active adapter changes the judge's output).
	Cache      *ClaimCache
	CacheScope string

	// Live serves "flash" and "hybrid" requests (Task 3.8). Nil means index only.
	Live LiveSource
}

func (e *Engine) thresholds() Thresholds {
	if e.Thresholds == (Thresholds{}) {
		return DefaultThresholds()
	}
	return e.Thresholds
}

// Verify extracts the claims of req.Text and checks every checkable one. An
// extraction failure returns the error and a result that is degraded (uncommitted,
// not chargeable).
func (e *Engine) Verify(ctx context.Context, req Request) (Result, error) {
	claims, err := e.Extractor.ExtractClaims(ctx, req.Text)
	if err != nil {
		res := Result{Calibrated: e.Cal.IsFitted()}
		res.Claims = []ClaimResult{{Degraded: true, Reason: err.Error(), Verdict: NEI, Source: SourceNone, Gate: GateUncommitted}}
		finish(&res)
		return res, err
	}
	return e.VerifyClaims(ctx, claims, req), nil
}

// VerifyClaims checks claims the caller already has (or Verify extracted). Claims
// marked not checkable are reported, not scored. Results come from the claim cache
// when one is set and the same claim was checked under the same index version,
// judge, adapters and frame.
func (e *Engine) VerifyClaims(ctx context.Context, claims []Claim, req Request) Result {
	res := Result{Calibrated: e.Cal.IsFitted()}
	for _, c := range claims {
		if !c.Checkable {
			res.Unchecked = append(res.Unchecked, c)
			continue
		}
		if err := ctx.Err(); err != nil {
			// Out of time: the remaining claims are pending, never guessed.
			res.Claims = append(res.Claims, ClaimResult{Claim: c, Verdict: NEI, Source: SourceNone, Pending: true,
				Reason: "not checked: " + err.Error(), Gate: GateUncommitted})
			continue
		}
		res.Claims = append(res.Claims, e.checkCached(ctx, c, req))
	}
	for _, c := range res.Claims {
		res.FlashUsed = res.FlashUsed || c.FlashUsed
		res.OutboundQueries = append(res.OutboundQueries, c.Queries...)
		res.Notes = append(res.Notes, c.LiveNotes...)
	}
	finish(&res)
	return res
}

func (e *Engine) checkCached(ctx context.Context, c Claim, req Request) ClaimResult {
	if e.Cache == nil || e.live(req) {
		// Live answers are never cached: a web page can change under a cached verdict.
		return e.CheckClaim(ctx, c, req)
	}
	key := e.cacheKey(c, req)
	if r, ok := e.Cache.Get(key); ok {
		r.Cached = true
		return r
	}
	r := e.CheckClaim(ctx, c, req)
	if !r.Degraded && !r.Failed && !r.Pending {
		e.Cache.Put(key, r)
	}
	return r
}

// CheckClaim runs one claim through ontology, retrieval, the deterministic value
// check and the judge, in that order; the first deterministic result wins over the
// model.
func (e *Engine) CheckClaim(ctx context.Context, c Claim, req Request) (r ClaimResult) {
	r = ClaimResult{Claim: c, Verdict: NEI, Source: SourceNone}
	th := e.thresholds()
	defer func() {
		r.Calibrated = e.Cal.Apply(c.Type, RawScore(r.Verdict, r.Confidence))
		r.Gate = Gate(r, th)
		if !e.Cal.IsFitted() && r.Gate > GateSeen {
			// Provisional: until the curve is fitted on owner-labelled gold data the
			// calibrated probability is not one, so nothing is accepted on it.
			r.Gate = GateSeen
		}
	}()

	// 1. The organization's ontology is authoritative for that organization.
	if req.Tenant != "" && c.Subject != "" && c.Property != "" && c.Value != "" {
		v, err := orgontology.Check(e.Index, req.Tenant, c.ev())
		if err != nil {
			r.Degraded, r.Reason = true, "ontology check failed: "+err.Error()
			return r
		}
		if v.Outcome == orgontology.Supported || v.Outcome == orgontology.Refuted {
			r.Evidence, r.Source, r.Confidence = v.Hits, SourceOntology, 0.99
			r.Verdict = Supported
			if v.Outcome == orgontology.Refuted {
				r.Verdict = Refuted
			}
			r.ReferenceTier, r.Agreement, r.Corroborated = true, true, true
			r.Cited = ids(v.Hits)
			r.Witnessed = e.witnessed(v.Hits)
			r.Reason = v.Reason
			return r
		}
	}

	// 2. Retrieval.
	k := req.K
	if k <= 0 {
		k = 8
	}
	var found evidence.Result
	var err error
	if e.live(req) {
		lr, lerr := e.Live.Retrieve(ctx, c.ev(), livesearch.Options{K: k, Tenant: req.Tenant, Mode: livesearch.Mode(req.Mode)})
		if lerr == nil {
			found = lr.Result
			r.FlashUsed, r.Queries = lr.FlashUsed, lr.Queries
			for _, why := range lr.Reasons {
				r.LiveNotes = append(r.LiveNotes, "live lookup: "+why)
			}
		} else {
			// An outage falls back to the index; the claim is not degraded, and says why.
			r.LiveNotes = append(r.LiveNotes, "live lookup failed, answered from the index: "+lerr.Error())
			found, err = e.Index.Search(c.ev(), evidence.Options{K: k, Tenant: req.Tenant, Tiers: req.Tiers})
		}
	} else {
		found, err = e.Index.Search(c.ev(), evidence.Options{K: k, Tenant: req.Tenant, Tiers: req.Tiers})
	}
	if err != nil {
		r.Degraded, r.Reason = true, "retrieval failed: "+err.Error()
		return r
	}
	r.Evidence = found.Hits
	if len(found.Hits) == 0 {
		r.Reason = "no evidence"
		return r
	}

	// 3. Deterministic value comparison on fact spans overrides the model.
	if v, hits, ok := structuredCheck(c, found.Hits); ok {
		r.Verdict, r.Source, r.Confidence = v, SourceStructured, 0.95
		r.Cited = ids(hits)
		r.Agreement = true
		r.ReferenceTier = allReferenceGrade(hits)
		r.Corroborated = distinctSnapshots(hits) >= 2 || anyTier(hits, evidence.TierOntology)
		r.Witnessed = e.witnessed(hits)
		r.Reason = "exact value comparison on fact spans"
		requireWebCorroboration(&r, hits)
		return r
	}

	// 4. The judge.
	j := e.Judge.Judge(ctx, c, found.Hits)
	r.Verdict, r.Source, r.Confidence, r.Margin = j.Verdict, SourceJudge, j.Confidence, j.Margin
	r.Agreement, r.DisagreeToRefuted, r.Degraded, r.Reason = j.Agreement, j.DisagreeToRefuted, j.Degraded, j.Reason
	if j.Degraded {
		r.Source = SourceNone
		return r
	}
	if j.Verdict == NEI {
		return r
	}
	cited := pick(found.Hits, j.Cited)
	r.Cited = j.Cited
	r.ReferenceTier = len(cited) > 0 && allReferenceGrade(cited)
	r.Corroborated = distinctSnapshots(cited) >= 2 || anyTier(cited, evidence.TierOntology)
	r.Witnessed = e.witnessed(cited)
	requireWebCorroboration(&r, cited)
	// False-refutation guard (3.3/3.4 follow-up, provisional proposal): a SUPPORTED or
	// REFUTED verdict must rest on evidence that is actually about the claim. If none of
	// the cited spans shares an entity or number with the claim, the citation is spurious
	// -- the judge latched onto an unrelated passage -- and the verdict is downgraded to
	// NEI. This is what caught "The capital of Atlantis is Poseidonis." being REFUTED with a
	// Paris passage in the live run. Claims with no extractable entity or number (purely
	// generic statements) are exempt: there is nothing to require overlap on.
	if (r.Verdict == Supported || r.Verdict == Refuted) && !citedOverlapsClaim(c, cited) {
		r.Verdict = NEI
		r.Corroborated, r.Agreement = false, false
		r.Reason = "the cited passage shares no entity or number with the claim"
	}
	return r
}

// sigTerms returns the significant terms of s that a verdict must overlap: named
// entities and numbers (canonical), as extracted by the evidence package.
func sigTerms(s string) map[string]bool {
	f := evidence.ExtractFeatures(s)
	m := make(map[string]bool, len(f.Entities)+len(f.Numbers))
	for _, e := range f.Entities {
		m[e] = true
	}
	for _, n := range f.Numbers {
		m[n] = true
	}
	return m
}

// citedOverlapsClaim reports whether any span the verdict rests on shares at least
// one significant entity or number with the claim.
func citedOverlapsClaim(c Claim, cited []evidence.Hit) bool {
	claimTerms := sigTerms(c.Text)
	if len(claimTerms) == 0 {
		return true // nothing to require overlap on
	}
	for _, h := range cited {
		for t := range sigTerms(h.Text) {
			if claimTerms[t] {
				return true
			}
		}
	}
	return false
}

// live reports whether this request may use live providers.
func (e *Engine) live(req Request) bool {
	return e.Live != nil && (req.Mode == string(livesearch.ModeFlash) || req.Mode == string(livesearch.ModeHybrid))
}

// MinWebDomains is how many independent registered domains a verdict needs when every
// span it rests on is live web evidence (the quarantine tier).
const MinWebDomains = 2

// requireWebCorroboration applies the quarantine rule: a verdict resting only on spans
// from the quarantine tier (pages fetched live) counts only if those pages come from at
// least MinWebDomains independent registered domains (news.bbc.co.uk and www.bbc.co.uk
// are one). A SUPPORTED or REFUTED verdict that fails the rule becomes NEI, because one
// web page is not enough to flag a claim true or false. A verdict that also rests on a
// reference, user or ontology span is not affected. Provisional (owner-level): the
// threshold of two and the choice to apply it to refutations as well.
func requireWebCorroboration(r *ClaimResult, cited []evidence.Hit) {
	if r.Verdict == NEI || len(cited) == 0 {
		return
	}
	var urls []string
	for _, h := range cited {
		if h.Tier != evidence.TierQuarantine {
			return
		}
		urls = append(urls, h.URL)
	}
	if livesearch.IndependentDomains(urls) >= MinWebDomains {
		return
	}
	r.Verdict, r.Corroborated, r.Agreement = NEI, false, false
	r.Reason = fmt.Sprintf("web evidence from fewer than %d independent registered domains is not enough", MinWebDomains)
}

func (e *Engine) witnessed(hits []evidence.Hit) bool {
	if e.Witness == nil || len(hits) == 0 {
		return false
	}
	for _, h := range hits {
		if !e.Witness(h) {
			return false
		}
	}
	return true
}

func ids(hits []evidence.Hit) []string {
	out := make([]string, len(hits))
	for i, h := range hits {
		out[i] = h.SpanID
	}
	return out
}

func pick(hits []evidence.Hit, spanIDs []string) []evidence.Hit {
	want := map[string]bool{}
	for _, id := range spanIDs {
		want[id] = true
	}
	var out []evidence.Hit
	for _, h := range hits {
		if want[h.SpanID] {
			out = append(out, h)
		}
	}
	return out
}

func allReferenceGrade(hits []evidence.Hit) bool {
	for _, h := range hits {
		if !referenceGrade(h.Tier) {
			return false
		}
	}
	return len(hits) > 0
}

func anyTier(hits []evidence.Hit, t evidence.Tier) bool {
	for _, h := range hits {
		if h.Tier == t {
			return true
		}
	}
	return false
}

func distinctSnapshots(hits []evidence.Hit) int {
	seen := map[string]bool{}
	for _, h := range hits {
		seen[h.Snapshot] = true
	}
	return len(seen)
}

// ---- structured value check ------------------------------------------------------

// parseFactText reads "subject — property: value", the form a fact span is stored in.
func parseFactText(t string) (s, p, v string, ok bool) {
	s, rest, found := strings.Cut(t, " — ")
	if !found {
		return
	}
	p, v, found = strings.Cut(rest, ": ")
	if !found {
		return
	}
	return s, p, v, true
}

// structuredCheck compares the claim's value with the value of fact spans for the
// same subject and property (the K_sp hits). A number or date is compared
// numerically after normalization; other values only by canonical equality. It
// decides only when it can: equal wins, and a different number on every matching
// fact is a refutation. A different non-numeric wording is left to the judge.
func structuredCheck(c Claim, hits []evidence.Hit) (Verdict, []evidence.Hit, bool) {
	if c.Subject == "" || c.Property == "" || c.Value == "" {
		return NEI, nil, false
	}
	wantS, wantP := evidence.Canon(c.Subject), evidence.Canon(c.Property)
	want := numbersIn(c.Value)
	var equal, differ []evidence.Hit
	for _, h := range hits {
		if !h.Fact {
			continue
		}
		s, p, v, ok := parseFactText(h.Text)
		if !ok || evidence.Canon(s) != wantS || evidence.Canon(p) != wantP {
			continue
		}
		have := numbersIn(v)
		switch {
		case len(want) > 0 && len(have) > 0:
			if sameStrings(want, have) {
				equal = append(equal, h)
			} else {
				differ = append(differ, h)
			}
		case evidence.Canon(v) == evidence.Canon(c.Value):
			equal = append(equal, h)
		}
	}
	switch {
	case len(equal) > 0:
		return Supported, equal, true
	case len(differ) > 0:
		return Refuted, differ, true
	}
	return NEI, nil, false
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	m := map[string]int{}
	for _, x := range a {
		m[x]++
	}
	for _, x := range b {
		m[x]--
	}
	for _, n := range m {
		if n != 0 {
			return false
		}
	}
	return true
}
