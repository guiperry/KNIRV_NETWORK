package factslice

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/guiperry/knirv-sdk-go/models/frame"

	"knirvmodels/pkg/evidence"
	"knirvmodels/pkg/orgontology"
)

func testIndex(t *testing.T) *evidence.Index {
	t.Helper()
	ix, err := evidence.Open(t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(ix.Close)
	ref := func(docs ...evidence.Doc) {
		if _, err := ix.Ingest(evidence.Batch{Tier: evidence.TierReference, Docs: docs}); err != nil {
			t.Fatal(err)
		}
	}
	// Two independent reference sources for Paris (corroboration), one for the tower.
	ref(evidence.Doc{ID: "wiki-paris", Title: "Paris", Text: "Paris is the capital and largest city of France. It lies on the Seine."})
	ref(evidence.Doc{ID: "atlas-france", Title: "France", Text: "The capital of France is Paris, which is also its seat of government."})
	ref(evidence.Doc{ID: "wiki-tower", Title: "Eiffel Tower", Text: "The Eiffel Tower is a wrought-iron lattice tower in Paris.",
		Facts: []evidence.Fact{{Subject: "Eiffel Tower", Property: "height", Value: "330 metres"}, {Subject: "Eiffel Tower", Property: "opened", Value: "1889"}}})
	return ix
}

// citing is a model that reads the evidence honestly: it says verdict and cites
// everything shown.
func citing(verdict string, conf float64) *fakeLLM {
	return &fakeLLM{judge: honest(verdict, conf)}
}

func engineFor(ix *evidence.Index, llm *fakeLLM, cal *Calibrator) *Engine {
	reg := frame.Default()
	return &Engine{Index: ix, Extractor: &Extractor{LLM: llm, Reg: reg}, Judge: &Judge{LLM: llm, Reg: reg}, Cal: cal}
}

func claimReq() Request { return Request{K: 6} }

func fittedIdentity() *Calibrator { return &Calibrator{Fitted: true} } // empty curve: identity

func parisClaim() Claim {
	return Claim{Text: "Paris is the capital of France", Subject: "Paris", Property: "role", Value: "capital of France", Type: TypeEntity, Checkable: true}
}

func TestJudgedClaimWithCorroboratedReferenceEvidenceIsAcceptedOnlyWhenCalibratedAndWitnessed(t *testing.T) {
	ix := testIndex(t)
	llm := citing("SUPPORTED", 0.97)
	e := engineFor(ix, llm, fittedIdentity())
	e.Witness = func(evidence.Hit) bool { return true }
	r := e.CheckClaim(context.Background(), parisClaim(), claimReq())
	if r.Verdict != Supported || r.Source != SourceJudge || !r.ReferenceTier || !r.Corroborated || !r.Agreement || !r.Witnessed || r.Gate != GateAccepted {
		t.Fatalf("%+v", r)
	}
	if len(r.Cited) < 2 {
		t.Fatalf("cited %v", r.Cited)
	}

	// The same claim without provenance proofs is only "seen".
	e.Witness = nil
	if r := e.CheckClaim(context.Background(), parisClaim(), claimReq()); r.Gate != GateSeen {
		t.Fatalf("unwitnessed: %s", r.Gate)
	}
	// Until a curve is fitted on gold, nothing passes "seen" however good it looks.
	e = engineFor(ix, citing("SUPPORTED", 0.99), nil)
	e.Witness = func(evidence.Hit) bool { return true }
	if r := e.CheckClaim(context.Background(), parisClaim(), claimReq()); r.Gate != GateSeen {
		t.Fatalf("uncalibrated reached %s", r.Gate)
	}
}

func TestNoEvidenceIsRefusedAndTheJudgeIsNotCalled(t *testing.T) {
	llm := citing("SUPPORTED", 0.99)
	e := engineFor(testIndex(t), llm, fittedIdentity())
	c := Claim{Text: "Zorblax quantum flumes devour nebulae", Type: TypeGeneral, Checkable: true}
	r := e.CheckClaim(context.Background(), c, claimReq())
	if r.Gate != GateRefused || len(r.Evidence) != 0 || llm.count("hasher.judge") != 0 || r.Degraded {
		t.Fatalf("%+v calls=%d", r, llm.count("hasher.judge"))
	}
}

func TestJudgeDownIsDegradedUncommittedAndNotCharged(t *testing.T) {
	llm := &fakeLLM{
		extract: func(string) (string, error) {
			return extractJSON(map[string]any{"text": "Paris is the capital of France", "checkable": true}), nil
		},
		judge: func(string, int) (string, error) { return "", context.DeadlineExceeded },
	}
	e := engineFor(testIndex(t), llm, fittedIdentity())
	res, err := e.Verify(context.Background(), Request{Text: "Paris is the capital of France.", K: 6})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Claims) != 1 || !res.Claims[0].Degraded || res.Gate != GateUncommitted || res.Chargeable || res.IsAccurate || res.Refused {
		t.Fatalf("%+v", res)
	}
}

func TestExtractionFailureIsDegradedAndReturned(t *testing.T) {
	llm := &fakeLLM{extract: func(string) (string, error) { return "garbage", nil }}
	res, err := engineFor(testIndex(t), llm, nil).Verify(context.Background(), Request{Text: "Paris is in France."})
	if err == nil || res.Gate != GateUncommitted || res.Chargeable {
		t.Fatalf("%v %+v", err, res)
	}
}

// Canary set: claims known to be false. A deterministic comparison on reference facts
// refutes them whatever the judge says, so a model that would call them SUPPORTED
// never gets the chance.
func TestKnownFalseClaimsAreRefutedWithoutTheJudge(t *testing.T) {
	llm := citing("SUPPORTED", 0.999) // the worst judge: says yes to everything
	e := engineFor(testIndex(t), llm, fittedIdentity())
	canaries := []Claim{
		{Text: "The Eiffel Tower is 450 metres tall", Subject: "Eiffel Tower", Property: "height", Value: "450 metres"},
		{Text: "The Eiffel Tower opened in 1925", Subject: "Eiffel Tower", Property: "opened", Value: "1925"},
		{Text: "The Eiffel Tower is 12 metres tall", Subject: "Eiffel Tower", Property: "height", Value: "12 metres"},
	}
	for _, c := range canaries {
		c.Checkable, c.Type = true, TypeNumber
		r := e.CheckClaim(context.Background(), c, claimReq())
		if r.Verdict != Refuted || r.Source != SourceStructured || r.Gate != GateRefused {
			t.Errorf("%q: %+v", c.Text, r)
		}
	}
	if llm.count("hasher.judge") != 0 {
		t.Fatalf("the judge was consulted %d times for deterministic cases", llm.count("hasher.judge"))
	}
	// The true value is supported by the same deterministic route.
	ok := Claim{Text: "The Eiffel Tower is 330 metres tall", Subject: "Eiffel Tower", Property: "height", Value: "330 metres", Type: TypeNumber, Checkable: true}
	r := e.CheckClaim(context.Background(), ok, claimReq())
	if r.Verdict != Supported || r.Source != SourceStructured || !r.ReferenceTier {
		t.Fatalf("%+v", r)
	}
	// A different wording of a non-numeric value is left to the judge, not decided by code.
	w := Claim{Text: "Paris is a village", Subject: "Eiffel Tower", Property: "location", Value: "a village", Checkable: true}
	if _, _, ok := structuredCheck(w, []evidence.Hit{{Fact: true, Text: "Eiffel Tower — location: Paris"}}); ok {
		t.Fatal("a differing wording was decided by code")
	}
}

func TestOntologyOverridesTheJudgeAndPublicSources(t *testing.T) {
	ix := testIndex(t)
	cfg := `{"organization_id":"acme","name":"Acme","ontology":{"decision_rights":{"refund_approval":"manager"}}}`
	var oc orgontology.Config
	if err := json.Unmarshal([]byte(cfg), &oc); err != nil {
		t.Fatal(err)
	}
	if _, err := orgontology.Sync(ix, oc); err != nil {
		t.Fatal(err)
	}
	llm := citing("SUPPORTED", 0.999)
	e := engineFor(ix, llm, fittedIdentity())
	req := Request{Tenant: "acme", K: 6}
	wrong := Claim{Text: "Refund approval belongs to the director", Subject: "decision rights", Property: "refund approval", Value: "director", Type: TypeEntity, Checkable: true}
	r := e.CheckClaim(context.Background(), wrong, req)
	if r.Verdict != Refuted || r.Source != SourceOntology || r.Gate != GateRefused || r.Reason == "" {
		t.Fatalf("%+v", r)
	}
	right := wrong
	right.Value = "manager"
	r = e.CheckClaim(context.Background(), right, req)
	if r.Verdict != Supported || r.Source != SourceOntology || !r.ReferenceTier || !r.Corroborated {
		t.Fatalf("%+v", r)
	}
	if llm.count("hasher.judge") != 0 {
		t.Fatal("the judge decided an ontology case")
	}
	// Another tenant sees no opinion and falls through to retrieval.
	r = e.CheckClaim(context.Background(), wrong, Request{Tenant: "other", K: 6})
	if r.Source == SourceOntology {
		t.Fatalf("one tenant's ontology answered for another: %+v", r)
	}
}

func TestVerifyEndToEndMixesCheckableAndUncheckable(t *testing.T) {
	llm := &fakeLLM{
		extract: func(string) (string, error) {
			return extractJSON(
				map[string]any{"text": "Paris is the capital of France", "checkable": true},
				map[string]any{"text": "I think Paris is lovely", "checkable": true},
			), nil
		},
		judge: honest("SUPPORTED", 0.97),
	}
	e := engineFor(testIndex(t), llm, fittedIdentity())
	e.Witness = func(evidence.Hit) bool { return true }
	res, err := e.Verify(context.Background(), Request{Text: "Paris is the capital of France. I think Paris is lovely.", K: 6})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Claims) != 1 || len(res.Unchecked) != 1 || res.UncheckedFraction != 0.5 {
		t.Fatalf("%+v", res)
	}
	if res.Gate != GateAccepted || !res.IsAccurate || res.Refused || !res.Chargeable || !res.Calibrated {
		t.Fatalf("%+v", res)
	}
	if len(res.Citations) == 0 || res.Confidence < 0.9 {
		t.Fatalf("citations %v confidence %v", res.Citations, res.Confidence)
	}
	for _, p := range res.Citations {
		if p < 0 || p >= len(res.Claims[0].Evidence) {
			t.Fatalf("citation %d out of range", p)
		}
	}
	// Uncalibrated, the same text is not accepted.
	e.Cal = nil
	res, _ = e.Verify(context.Background(), Request{Text: "Paris is the capital of France. I think Paris is lovely.", K: 6})
	if res.Gate > GateSeen || res.IsAccurate || res.Calibrated {
		t.Fatalf("%+v", res)
	}
	if !strings.Contains(strings.ToLower(res.Gate.String()), "seen") {
		t.Fatalf("gate %s", res.Gate)
	}
}

// citedOverlapsClaim is the unit form of the false-refutation guard: a verdict must
// rest on a span that is actually about the claim.
func TestCitedOverlapsClaim(t *testing.T) {
	parisSpan := evidence.Hit{Text: "Paris is the capital and largest city of France."}
	atlantisSpan := evidence.Hit{Text: "Atlantis is a lost city; Poseidonis was its capital."}
	atlantis := Claim{Text: "The capital of Atlantis is Poseidonis."}
	paris := Claim{Text: "Paris is the capital of France."}
	if !citedOverlapsClaim(paris, []evidence.Hit{parisSpan}) {
		t.Fatal("a Paris claim should overlap a Paris span")
	}
	if citedOverlapsClaim(atlantis, []evidence.Hit{parisSpan}) {
		t.Fatal("an Atlantis claim must not overlap a Paris span (the live false-refutation case)")
	}
	if !citedOverlapsClaim(atlantis, []evidence.Hit{atlantisSpan}) {
		t.Fatal("an Atlantis claim should overlap an Atlantis span")
	}
	// Exempt when the claim carries no extractable entity or number: a generic claim
	// must not be downgraded for lack of overlap.
	generic := Claim{Text: "It is a nice day."}
	if !citedOverlapsClaim(generic, []evidence.Hit{parisSpan}) {
		t.Fatal("a generic claim with no entities must not be downgraded")
	}
	// A shared named entity is enough to overlap even when the numbers differ.
	num := Claim{Text: "The Eiffel Tower is 450 metres tall."}
	numSpan := evidence.Hit{Text: "The Eiffel Tower is 330 metres tall."}
	if !citedOverlapsClaim(num, []evidence.Hit{numSpan}) {
		t.Fatal("a claim and span sharing the entity Eiffel Tower should overlap")
	}
}

// A SUPPORTED/REFUTED verdict whose only cited span is about a different subject is a
// spurious citation and is downgraded to NEI (3.3/3.4 follow-up).
func TestFalseRefutationGuardDowngradesUnrelatedCitation(t *testing.T) {
	ix, err := evidence.Open(t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(ix.Close)
	// Only a Paris passage exists; the claim is about Atlantis and shares no entity or
	// number with it. They overlap only on the generic term "capital", which is enough
	// for lexical retrieval to return the passage but NOT enough for the false-
	// refutation guard, which requires an entity/number overlap.
	if _, err := ix.Ingest(evidence.Batch{Tier: evidence.TierReference, Docs: []evidence.Doc{{ID: "paris", Text: "The capital of France is Paris, which is also its seat of government."}}}); err != nil {
		t.Fatal(err)
	}
	llm := &fakeLLM{judge: honest("REFUTED", 1.0)} // a model that wrongly refutes, citing what it was shown
	e := engineFor(ix, llm, fittedIdentity())
	c := Claim{Text: "The capital of Atlantis is Poseidonis.", Type: TypeEntity, Checkable: true}
	r := e.CheckClaim(context.Background(), c, claimReq())
	// The fixture is deterministic: the claim and the only passage share the term
	// "capital", so lexical retrieval must return at least one hit and the judge must be
	// reached. If retrieval returns nothing we must FAIL, not skip, or a retrieval/index
	// regression would silently drop the guard's coverage while CI stays green.
	if len(r.Evidence) == 0 {
		t.Fatalf("lexical retrieval returned no evidence for the claim; the false-refutation judge path is untested")
	}
	if r.Verdict != NEI || r.Reason == "" || llm.count("hasher.judge") == 0 {
		t.Fatalf("an unrelated citation should be downgraded to NEI: %+v", r)
	}
}

// TestFalseRefutationGuardNoEvidence covers the opposite path from
// TestFalseRefutationGuardDowngradesUnrelatedCitation: when lexical retrieval returns
// nothing, the engine must not call the judge and must return a safe NEI ("no evidence")
// rather than guessing. Kept separate so the downgrade test can require a hit.
func TestFalseRefutationGuardNoEvidence(t *testing.T) {
	ix, err := evidence.Open(t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(ix.Close)
	llm := &fakeLLM{judge: honest("REFUTED", 1.0)}
	e := engineFor(ix, llm, fittedIdentity())
	c := Claim{Text: "The capital of Atlantis is Poseidonis.", Type: TypeEntity, Checkable: true}
	r := e.CheckClaim(context.Background(), c, claimReq())
	if len(r.Evidence) != 0 {
		t.Fatalf("expected no evidence from an empty index, got %d hits", len(r.Evidence))
	}
	if r.Verdict != NEI || r.Reason != "no evidence" || llm.count("hasher.judge") != 0 {
		t.Fatalf("empty retrieval should yield a safe NEI without the judge: %+v", r)
	}
}

// The guard must not change a legitimate verdict: a Paris claim resting on a Paris
// passage stays SUPPORTED.
func TestFalseRefutationGuardKeepsLegitimateVerdicts(t *testing.T) {
	ix := testIndex(t)
	llm := citing("SUPPORTED", 0.97)
	e := engineFor(ix, llm, fittedIdentity())
	e.Witness = func(evidence.Hit) bool { return true }
	r := e.CheckClaim(context.Background(), parisClaim(), claimReq())
	if r.Verdict != Supported || r.Gate != GateAccepted {
		t.Fatalf("a Paris claim with Paris evidence should stay SUPPORTED: %+v", r)
	}
}

func TestCustomThresholdsAreUsed(t *testing.T) {
	ix := testIndex(t)
	e := engineFor(ix, citing("SUPPORTED", 0.80), fittedIdentity())
	e.Witness = func(evidence.Hit) bool { return true }
	if r := e.CheckClaim(context.Background(), parisClaim(), claimReq()); r.Gate != GateAttested {
		t.Fatalf("default thresholds: %s", r.Gate)
	}
	th := DefaultThresholds()
	th.Accepted, th.Committed = 0.75, 0.80
	e.Thresholds = th
	if r := e.CheckClaim(context.Background(), parisClaim(), claimReq()); r.Gate != GateAccepted {
		t.Fatalf("custom thresholds: %s", r.Gate)
	}
}

// The three HASHER stages are registered in the SDK frame registry (core
// frame_defaults.json), where heart.chat is, and build a frame.
func TestHasherStagesAreRegisteredInTheFrameRegistry(t *testing.T) {
	reg := frame.Default()
	for _, id := range []string{"hasher.extract", "hasher.judge", "hasher.query"} {
		s, ok := reg.Stage(id)
		if !ok || s.Layer != "HASHER" || len(s.OutputSchema) == 0 || s.MaxToolCalls != 0 {
			t.Errorf("%s: %+v ok=%v", id, s, ok)
		}
		fr, err := reg.Build(id, frame.LiveFacts{})
		if err != nil || fr.Hash == "" || fr.Stage != id {
			t.Errorf("%s: build %v %+v", id, err, fr)
		}
	}
	if _, ok := reg.Stage("heart.chat"); !ok {
		t.Fatal("heart.chat vanished")
	}
}

// TestVerifyFlagsHallucinationWithCitationsAndGate is the end-to-end form of the
// 3.4 Done-when: a known hallucination sent through Engine.Verify returns a flagged
// verdict, valid citations (span ids the engine actually retrieved, never invented)
// and a denial-gate level. The extractor yields the fabricated claim; the judge
// refutes it and cites the Eiffel Tower passage it was shown; the false-refutation
// guard leaves the verdict intact because the cited span shares the "Eiffel Tower"
// entity with the claim.
func TestVerifyFlagsHallucinationWithCitationsAndGate(t *testing.T) {
	ix := testIndex(t)
	const hallucination = "The Eiffel Tower was painted entirely gold in 2020"
	llm := &fakeLLM{
		extract: func(string) (string, error) {
			return extractJSON(map[string]any{"text": hallucination, "checkable": true}), nil
		},
		judge: honest("REFUTED", 0.9),
	}
	e := engineFor(ix, llm, fittedIdentity())
	res, err := e.Verify(context.Background(), Request{Text: hallucination + ".", K: 6})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Claims) != 1 {
		t.Fatalf("expected exactly one checkable claim, got %d: %+v", len(res.Claims), res)
	}
	cr := res.Claims[0]
	if cr.Verdict != Refuted {
		t.Fatalf("the hallucination should be flagged REFUTED, got %s: %+v", cr.Verdict, cr)
	}
	// Valid citations: every cited span id must be one the engine retrieved, i.e. the
	// judge may only lean on evidence that exists, never invent a span.
	evIDs := map[string]bool{}
	for _, h := range cr.Evidence {
		evIDs[h.SpanID] = true
	}
	if len(cr.Evidence) == 0 {
		t.Fatalf("no evidence was retrieved for the hallucination, so it cannot cite anything: %+v", cr)
	}
	if len(cr.Cited) == 0 {
		t.Fatalf("a flagged claim must return citations: %+v", cr)
	}
	for _, id := range cr.Cited {
		if !evIDs[id] {
			t.Fatalf("cited span %q is not among the retrieved evidence %v", id, evIDs)
		}
	}
	// The response-level citations point at real positions in the claim's evidence.
	if len(res.Citations) == 0 {
		t.Fatalf("response citations empty: %+v", res)
	}
	for _, p := range res.Citations {
		if p < 0 || p >= len(cr.Evidence) {
			t.Fatalf("response citation %d out of range (evidence %d)", p, len(cr.Evidence))
		}
	}
	// A flagged hallucination must not reach an accepting gate: it is at most
	// "discouraged" (refused or discouraged).
	if res.Gate > GateDiscouraged {
		t.Fatalf("a flagged hallucination must not reach an accepting gate, got %s", res.Gate)
	}
	if res.IsAccurate {
		t.Fatalf("a flagged hallucination must not be marked accurate")
	}
}
