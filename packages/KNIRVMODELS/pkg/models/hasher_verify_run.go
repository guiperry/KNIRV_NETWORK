package models

import (
	"context"
	"strconv"
	"strings"

	"knirvmodels/pkg/evidence"
	"knirvmodels/pkg/factslice"

	"github.com/guiperry/knirv-sdk-go/models/frame"

	sdk "github.com/guiperry/knirv-sdk-go/models"
)

// run executes one admitted job and publishes its result. Exactly one job uses the
// judge at a time (llama has one slot); the rest wait their turn inside the queue bound.
func (v *verifier) run(job *verifyJob, ix *evidence.Index, req sdk.VerifyRequest, tenant string) {
	ctx, cancel := context.WithTimeout(context.Background(), verifyRunLimit)
	defer cancel()
	defer func() {
		<-v.queue
		close(job.done)
	}()
	select {
	case v.sem <- struct{}{}:
		defer func() { <-v.sem }()
	case <-ctx.Done():
		job.resp = v.degraded(ix, req, "waited too long for the judge")
		return
	}
	job.resp = v.execute(ctx, ix, req, tenant)
	job.resp.JobID = jobID(job)
}

// degraded is the answer when no verdict can be produced: gate uncommitted, no
// charge, no guess.
func (v *verifier) degraded(ix *evidence.Index, req sdk.VerifyRequest, why string) sdk.VerifyResponse {
	resp := v.pending(ix, req, &verifyJob{})
	resp.Status, resp.Unverified, resp.Degraded, resp.Chargeable = sdk.VerifyDone, false, true, false
	resp.JobID = ""
	text := strings.TrimSpace(req.Text)
	if len(req.Claims) > 0 {
		text = req.Claims[0].Text
	}
	if len(text) > 200 {
		text = text[:200]
	}
	resp.Claims = []sdk.VerifyClaim{{Text: text, Type: string(factslice.TypeGeneral), Verdict: string(factslice.NEI), Source: string(factslice.SourceNone),
		Gate: factslice.GateUncommitted.String(), GateLevel: int(factslice.GateUncommitted), Evidence: []sdk.VerifyEvidence{}, Degraded: true, Reason: why}}
	resp.Notes = append(resp.Notes, why)
	return resp
}

func (v *verifier) execute(ctx context.Context, ix *evidence.Index, req sdk.VerifyRequest, tenant string) sdk.VerifyResponse {
	if v.cfg.Live != nil && !v.cfg.Live(ctx) {
		return v.degraded(ix, req, "judge unavailable: the llama wrapper is not ready")
	}
	judge := sdk.VerifyJudge{Model: "unknown"}
	if v.cfg.Judge != nil {
		judge = v.cfg.Judge(ctx)
	}
	engine := &factslice.Engine{
		Index:      ix,
		Extractor:  &factslice.Extractor{LLM: v.cfg.LLM, Reg: v.cfg.Reg, Facts: v.cfg.Facts},
		Judge:      &factslice.Judge{LLM: v.cfg.LLM, Reg: v.cfg.Reg, Facts: v.cfg.Facts},
		Cal:        v.cfg.Cal,
		Thresholds: v.cfg.Thresholds,
		Witness:    witnessFor(v.cfg.Witness, ix),
		Cache:      v.cache,
		CacheScope: v.scope(judge),
	}
	// Live lookup (Task 3.8) only when the deployment switched it on; otherwise a flash or
	// hybrid request is answered from the index and says so.
	live := false
	if req.Mode == sdk.ModeFlash || req.Mode == sdk.ModeHybrid {
		if ret, err := v.retriever(); err == nil && ret.Enabled() {
			engine.Live, live = ret, true
		}
	}
	fr := factslice.Request{Tenant: tenant, K: 8}
	if live {
		fr.Mode = req.Mode
	}

	var claims []factslice.Claim
	if len(req.Claims) > 0 {
		for _, c := range req.Claims {
			claims = append(claims, factslice.Prepare(c.Text, c.Subject, c.Property, c.Value))
		}
	} else {
		var err error
		claims, err = engine.Extractor.ExtractClaims(ctx, req.Text)
		if err != nil {
			return v.degraded(ix, req, "claim extraction failed: "+err.Error())
		}
	}
	var over []factslice.Claim
	if n := countChecked(claims); n > req.MaxClaims {
		kept, seen := claims[:0:0], 0
		for _, c := range claims {
			if c.Checkable {
				if seen++; seen > req.MaxClaims {
					c.Checkable, c.Why = false, "over max_claims"
					over = append(over, c)
					continue
				}
			}
			kept = append(kept, c)
		}
		claims = append(kept, over...)
	}
	res := engine.VerifyClaims(ctx, claims, fr)
	return toResponse(res, req, ix, judge, len(over), live)
}

func countChecked(cs []factslice.Claim) int {
	n := 0
	for _, c := range cs {
		if c.Checkable {
			n++
		}
	}
	return n
}

// scope is what a cached result depends on besides the claim and the index version:
// the judge model file, the adapter set, and the extract and judge frames.
func (v *verifier) scope(j sdk.VerifyJudge) string {
	parts := []string{j.Model, j.SHA256, j.Adapters}
	for _, stage := range []string{"hasher.extract", "hasher.judge"} {
		lf := v.facts()
		if fr, err := v.cfg.Reg.Build(stage, lf); err == nil {
			parts = append(parts, fr.Hash)
		}
	}
	return strings.Join(parts, "|")
}

func (v *verifier) facts() (lf frame.LiveFacts) {
	if v.cfg.Facts != nil {
		return v.cfg.Facts()
	}
	return frame.LiveFacts{}
}

func toResponse(res factslice.Result, req sdk.VerifyRequest, ix *evidence.Index, judge sdk.VerifyJudge, dropped int, live bool) sdk.VerifyResponse {
	st := ix.Status()
	embedder := ""
	if len(st.EmbedderIDs) > 0 {
		embedder = st.EmbedderIDs[0]
	}
	out := sdk.VerifyResponse{
		Status: sdk.VerifyDone, Gate: res.Gate.String(), GateLevel: int(res.Gate), GateScore: round3(res.Confidence),
		Confidence: round3(res.Confidence), Claims: []sdk.VerifyClaim{}, UncheckedFraction: res.UncheckedFraction,
		Chargeable: res.Chargeable, Calibrated: res.Calibrated, IndexVersion: st.Version, EmbedderID: embedder,
		Judge: &judge, Mode: appliedMode(req, live), Delivery: req.Delivery, Domains: req.Domains, Notes: modeNotes(req, live),
		FlashUsed: res.FlashUsed, OutboundQueries: res.OutboundQueries,
		Refused: res.Refused, IsAccurate: res.IsAccurate, Citations: res.Citations,
	}
	if out.Citations == nil {
		out.Citations = []int{}
	}
	var supported, refuted int
	for _, c := range res.Claims {
		cl := sdk.VerifyClaim{
			Text: c.Claim.Text, Subject: c.Claim.Subject, Property: c.Claim.Property, Value: c.Claim.Value, Type: string(c.Claim.Type),
			Verdict: string(c.Verdict), Source: string(c.Source), Confidence: round3(c.Confidence), Calibrated: round3(c.Calibrated),
			Gate: c.Gate.String(), GateLevel: int(c.Gate), Evidence: make([]sdk.VerifyEvidence, 0, len(c.Evidence)), Cited: c.Cited,
			Agreement: c.Agreement, Degraded: c.Degraded || c.Failed || c.Pending, Reason: c.Reason, Cached: c.Cached,
		}
		if c.Pending {
			out.Status = sdk.VerifyPending
			out.Unverified = true
		}
		for _, h := range c.Evidence {
			ev := sdk.VerifyEvidence{SpanID: h.SpanID, Snapshot: h.Snapshot, SourceTier: string(h.Tier),
				Text: h.Text, Title: h.Title, URL: h.URL, FetchedAt: h.FetchedAt, License: h.License}
			if w, ok := ix.WitnessOf(h); ok {
				ev.Witness = &sdk.VerifyWitness{BatchRoot: w.BatchRoot, Proof: proofString(w.Proof), Attestation: w.Attestation.Kind + ":" + w.Attestation.Ref}
			}
			cl.Evidence = append(cl.Evidence, ev)
		}
		if cl.Degraded {
			out.Degraded = true
		}
		switch c.Verdict {
		case factslice.Supported:
			supported++
		case factslice.Refuted:
			refuted++
		}
		out.Claims = append(out.Claims, cl)
	}
	for _, u := range res.Unchecked {
		out.Unchecked = append(out.Unchecked, sdk.VerifyUnchecked{Text: u.Text, Why: u.Why})
	}
	switch {
	case len(res.Claims) == 0:
		out.Verdict = "unchecked"
	case supported > 0 && refuted > 0:
		out.Verdict = "mixed"
	case supported == len(res.Claims):
		out.Verdict = "supported"
	case refuted > 0 && supported == 0:
		out.Verdict = "refuted"
	default:
		out.Verdict = "nei"
	}
	out.Notes = append(out.Notes, res.Notes...)
	if !res.Calibrated {
		out.Notes = append(out.Notes, "calibration is not installed: no claim is taken past 'seen'")
	}
	if dropped > 0 {
		out.Notes = append(out.Notes, "claims beyond max_claims were not checked")
	}
	return out
}

func round3(f float64) float64 { return float64(int64(f*1000+0.5)) / 1000 }

// witnessFor chooses the provenance check: an explicit one from the configuration, else the
// index's own (a span is attested when its batch root has been witnessed). Because no
// witnesser is shipped yet (the seeder side of Task 3.9 is a follow-up), nothing is attested
// in production until one is installed.
func witnessFor(configured factslice.Witness, ix *evidence.Index) factslice.Witness {
	if configured != nil {
		return configured
	}
	return ix.Attested
}

// proofString renders an inclusion proof compactly: index, then each sibling as L: or R: and its hash.
func proofString(p evidence.BatchProof) string {
	parts := []string{"merkle:" + p.BatchID + "#" + itoa(p.Index)}
	for _, st := range p.Steps {
		side := "R"
		if st.Left {
			side = "L"
		}
		parts = append(parts, side+":"+strings.TrimPrefix(st.Sibling, "sha256:"))
	}
	return strings.Join(parts, "/")
}

func itoa(n int) string { return strconv.Itoa(n) }
