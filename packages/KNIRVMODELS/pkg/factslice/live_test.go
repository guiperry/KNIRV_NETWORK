package factslice

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"knirvmodels/pkg/evidence"
	"knirvmodels/pkg/livesearch"
)

// fakeLive stands in for the livesearch retriever.
type fakeLive struct {
	res   livesearch.Result
	err   error
	calls int32
	last  livesearch.Options
}

func (f *fakeLive) Retrieve(_ context.Context, _ evidence.Claim, o livesearch.Options) (livesearch.Result, error) {
	atomic.AddInt32(&f.calls, 1)
	f.last = o
	return f.res, f.err
}

func webHit(id, host, text string) evidence.Hit {
	return evidence.Hit{SpanID: "sha256:" + id, Snapshot: "sha256:snap" + id, Text: text, Tier: evidence.TierQuarantine,
		URL: "https://" + host + "/page", Source: livesearch.MetasearchName, FetchedAt: "2026-11-03T12:00:00Z"}
}

func liveResult(flash bool, hits ...evidence.Hit) livesearch.Result {
	return livesearch.Result{Result: evidence.Result{Hits: hits}, FlashUsed: flash, Mode: livesearch.ModeFlash,
		Queries: []string{"metasearch: moon distance kilometres"}}
}

func moonClaim() Claim {
	return Claim{Text: "The Moon is 384400 kilometres from Earth", Type: TypeNumber, Checkable: true}
}

func TestQuarantineEvidenceNeedsTwoIndependentRegisteredDomains(t *testing.T) {
	cases := []struct {
		name  string
		hits  []evidence.Hit
		want  Verdict
		gate  GateLevel
		cause string
	}{
		{"one page", []evidence.Hit{webHit("a", "www.nasa.gov", "The Moon orbits at 384400 kilometres.")}, NEI, GateUncommitted, "fewer than 2 independent"},
		{"two pages, one organisation", []evidence.Hit{webHit("a", "science.nasa.gov", "The Moon is 384400 kilometres away."), webHit("b", "www.nasa.gov", "Moon distance is 384400 kilometres.")}, NEI, GateUncommitted, "fewer than 2 independent"},
		{"two organisations", []evidence.Hit{webHit("a", "www.nasa.gov", "The Moon is 384400 kilometres away."), webHit("b", "www.esa.int", "Moon distance is 384400 kilometres.")}, Supported, GateSeen, ""},
		{"co.uk is one registered domain", []evidence.Hit{webHit("a", "news.bbc.co.uk", "The Moon is 384400 kilometres away."), webHit("b", "www.bbc.co.uk", "Moon distance is 384400 kilometres.")}, NEI, GateUncommitted, "fewer than 2 independent"},
	}
	for _, tc := range cases {
		ix := testIndexEmpty(t)
		live := &fakeLive{res: liveResult(true, tc.hits...)}
		e := engineFor(ix, citing("SUPPORTED", 0.97), fittedIdentity())
		e.Live = live
		r := e.CheckClaim(context.Background(), moonClaim(), Request{K: 6, Mode: "flash"})
		if r.Verdict != tc.want || r.Gate != tc.gate || (tc.cause != "" && !strings.Contains(r.Reason, tc.cause)) {
			t.Errorf("%s: %+v", tc.name, r)
		}
		if !r.FlashUsed || len(r.Queries) != 1 || live.last.Mode != livesearch.ModeFlash {
			t.Errorf("%s: live facts not recorded: %+v", tc.name, r)
		}
		if r.Gate > GateSeen {
			t.Errorf("%s: web evidence reached %s", tc.name, r.Gate)
		}
	}
}

// A refutation from one web page is not a refutation either.
func TestASingleWebPageCannotRefuteAClaim(t *testing.T) {
	e := engineFor(testIndexEmpty(t), citing("REFUTED", 0.97), fittedIdentity())
	e.Live = &fakeLive{res: liveResult(true, webHit("a", "www.example.org", "The Moon is 350000 kilometres away."))}
	r := e.CheckClaim(context.Background(), moonClaim(), Request{K: 6, Mode: "flash"})
	if r.Verdict != NEI || r.Gate == GateRefused || r.Gate == GateDiscouraged {
		t.Fatalf("%+v", r)
	}
}

// Evidence that includes a reference span is not subject to the web rule.
func TestReferenceEvidenceAlongsideTheWebIsNotHeldToTheWebRule(t *testing.T) {
	ix := testIndexEmpty(t)
	ref := evidence.Hit{SpanID: "sha256:r", Snapshot: "sha256:snapr", Text: "The Moon orbits the Earth at 384400 kilometres.", Tier: evidence.TierReference}
	e := engineFor(ix, citing("SUPPORTED", 0.97), fittedIdentity())
	e.Live = &fakeLive{res: liveResult(true, ref, webHit("a", "www.nasa.gov", "The Moon is 384400 kilometres away."))}
	r := e.CheckClaim(context.Background(), moonClaim(), Request{K: 6, Mode: "hybrid"})
	if r.Verdict != Supported {
		t.Fatalf("%+v", r)
	}
}

func TestLiveOutageFallsBackToTheIndexWithoutDegrading(t *testing.T) {
	ix := testIndex(t)
	e := engineFor(ix, citing("SUPPORTED", 0.97), fittedIdentity())
	e.Live = &fakeLive{err: errors.New("metasearch: connection refused")}
	r := e.CheckClaim(context.Background(), parisClaim(), Request{K: 6, Mode: "flash"})
	if r.Degraded || r.Verdict != Supported || len(r.Evidence) == 0 {
		t.Fatalf("an outage of the live path must not cost the index answer: %+v", r)
	}
	if len(r.LiveNotes) != 1 || !strings.Contains(r.LiveNotes[0], "answered from the index") || r.FlashUsed {
		t.Fatalf("%+v", r.LiveNotes)
	}
	// A degraded live result keeps its reasons.
	res := liveResult(false)
	res.Hits = []evidence.Hit{}
	e.Live = &fakeLive{res: livesearch.Result{Result: evidence.Result{Hits: nil}, Reasons: []string{"metasearch search failed: HTTP 503"}}}
	r = e.CheckClaim(context.Background(), moonClaim(), Request{K: 6, Mode: "hybrid"})
	if r.Degraded || len(r.LiveNotes) != 1 || !strings.Contains(r.LiveNotes[0], "HTTP 503") {
		t.Fatalf("%+v", r)
	}
}

func TestIndexModeNeverCallsTheLiveSourceAndLiveAnswersAreNotCached(t *testing.T) {
	ix := testIndex(t)
	live := &fakeLive{res: liveResult(false)}
	e := engineFor(ix, citing("SUPPORTED", 0.97), fittedIdentity())
	e.Live = live
	e.Cache = NewClaimCache(10, 0)
	for _, mode := range []string{"", "index"} {
		e.CheckClaim(context.Background(), parisClaim(), Request{K: 6, Mode: mode})
	}
	if live.calls != 0 {
		t.Fatalf("index mode called the live source %d times", live.calls)
	}
	// Same claim twice in flash mode: both go to the source (a web page can change under a cached verdict).
	live.res = liveResult(true, webHit("a", "www.nasa.gov", "The Moon is 384400 kilometres away."), webHit("b", "www.esa.int", "The Moon is 384400 kilometres away."))
	for i := 0; i < 2; i++ {
		e.VerifyClaims(context.Background(), []Claim{moonClaim()}, Request{K: 6, Mode: "flash"})
	}
	if live.calls != 2 || e.Cache.Len() != 0 {
		t.Fatalf("calls %d, cached %d", live.calls, e.Cache.Len())
	}
}

func TestResultAggregatesWhatLeftTheMachine(t *testing.T) {
	e := engineFor(testIndexEmpty(t), citing("SUPPORTED", 0.97), fittedIdentity())
	e.Live = &fakeLive{res: liveResult(true, webHit("a", "www.nasa.gov", "The Moon is 384400 kilometres away."), webHit("b", "www.esa.int", "The Moon is 384400 kilometres away."))}
	res := e.VerifyClaims(context.Background(), []Claim{moonClaim()}, Request{K: 6, Mode: "hybrid"})
	if !res.FlashUsed || len(res.OutboundQueries) != 1 || res.OutboundQueries[0] != "metasearch: moon distance kilometres" {
		t.Fatalf("%+v", res)
	}
	// Without live mode, nothing is reported as sent.
	res = engineFor(testIndex(t), citing("SUPPORTED", 0.97), fittedIdentity()).VerifyClaims(context.Background(), []Claim{parisClaim()}, Request{K: 6})
	if res.FlashUsed || len(res.OutboundQueries) != 0 {
		t.Fatalf("%+v", res)
	}
}

func testIndexEmpty(t *testing.T) *evidence.Index {
	t.Helper()
	ix, err := evidence.Open(t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(ix.Close)
	return ix
}
