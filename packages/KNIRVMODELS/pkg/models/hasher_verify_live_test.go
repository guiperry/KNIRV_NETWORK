package models

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"knirvmodels/pkg/evidence"
	"knirvmodels/pkg/livesearch"

	sdk "github.com/guiperry/knirv-sdk-go/models"
)

// fakeWeb is a live provider with two independent pages about the Moon.
type fakeWeb struct {
	mu      sync.Mutex
	queries []string
}

func (f *fakeWeb) Name() string { return livesearch.MetasearchName }
func (f *fakeWeb) Search(_ context.Context, q string, _ int) ([]livesearch.Ref, error) {
	f.mu.Lock()
	f.queries = append(f.queries, q)
	f.mu.Unlock()
	return []livesearch.Ref{
		{Source: livesearch.MetasearchName, ID: "https://www.nasa.gov/moon", Title: "Moon facts", URL: "https://www.nasa.gov/moon"},
		{Source: livesearch.MetasearchName, ID: "https://www.esa.int/moon", Title: "ESA Moon", URL: "https://www.esa.int/moon"},
	}, nil
}
func (f *fakeWeb) Fetch(_ context.Context, r livesearch.Ref) (livesearch.Page, error) {
	return livesearch.Page{Ref: r, Revision: "sha256:" + r.URL, Text: "The Moon orbits the Earth at an average distance of 384400 kilometres. Source page: " + r.Title + ".",
		License: "unknown", FetchedAt: time.Now().UTC().Format(time.RFC3339)}, nil
}
func (f *fakeWeb) Revisions(_ context.Context, refs []livesearch.Ref) (map[string]string, error) {
	out := map[string]string{}
	for _, r := range refs {
		out[r.ID] = "sha256:" + r.URL
	}
	return out, nil
}

func liveHasher(t *testing.T, enabled bool, llm *verifyLLM) (*httptest.Server, *fakeWeb) {
	t.Helper()
	h, srv, _ := verifierHasher(t, llm, nil)
	web := &fakeWeb{}
	h.UseLive(func(ix *evidence.Index, dir string) (*livesearch.Retriever, error) {
		return livesearch.NewRetriever(ix, dir, []livesearch.Provider{web}, livesearch.Config{Enabled: enabled})
	})
	return srv, web
}

var moonClaims = []map[string]any{{"text": "The Moon is 384400 kilometres from the Earth", "checkable": true}}

func TestFlashVerifyUsesTheWebWhenEnabledAndSaysWhatLeftTheMachine(t *testing.T) {
	llm := &verifyLLM{claims: moonClaims, judge: yes}
	srv, web := liveHasher(t, true, llm)
	code, out, raw := post(t, srv, "/models/hasher/verify", "", map[string]any{"text": "The Moon is 384400 kilometres from the Earth.", "mode": "flash", "delivery": "inline", "budget_ms": 20000})
	if code != 200 || out.Status != sdk.VerifyDone || len(out.Claims) != 1 {
		t.Fatalf("%d %s", code, raw)
	}
	if out.Mode != sdk.ModeFlash || !out.FlashUsed || len(out.OutboundQueries) != 1 || !strings.HasPrefix(out.OutboundQueries[0], "metasearch: ") {
		t.Fatalf("mode %q flash_used %v queries %v", out.Mode, out.FlashUsed, out.OutboundQueries)
	}
	c := out.Claims[0]
	if c.Verdict != "SUPPORTED" || c.Gate != "seen" {
		t.Fatalf("two independent registered domains support the claim: %+v", c)
	}
	tiers := map[string]bool{}
	for _, e := range c.Evidence {
		tiers[e.SourceTier] = true
	}
	if !tiers["quarantine"] {
		t.Fatalf("live pages are quarantine-tier evidence: %+v", c.Evidence)
	}
	if strings.Contains(strings.Join(out.Notes, " "), "not enabled") {
		t.Fatalf("notes %v", out.Notes)
	}
	web.mu.Lock()
	defer web.mu.Unlock()
	if len(web.queries) != 1 || strings.Contains(web.queries[0], ".") {
		t.Fatalf("outbound %v", web.queries)
	}
}

func TestFlashVerifyIsAnsweredFromTheIndexWhenTheSwitchIsOff(t *testing.T) {
	llm := &verifyLLM{claims: moonClaims, judge: yes}
	srv, web := liveHasher(t, false, llm)
	for _, mode := range []string{"flash", "hybrid"} {
		_, out, _ := post(t, srv, "/models/hasher/verify", "", map[string]any{"text": "The Moon is 384400 kilometres from the Earth.", "mode": mode, "delivery": "inline", "budget_ms": 20000})
		if out.Mode != sdk.ModeIndex || out.FlashUsed || len(out.OutboundQueries) != 0 || !strings.Contains(strings.Join(out.Notes, " "), "live lookup is not enabled") {
			t.Fatalf("%s: %+v", mode, out)
		}
	}
	web.mu.Lock()
	defer web.mu.Unlock()
	if len(web.queries) != 0 {
		t.Fatalf("a deployment with the switch off contacted a provider: %v", web.queries)
	}
}

func TestIndexModeOnAnEnabledDeploymentStaysOffTheWeb(t *testing.T) {
	llm := &verifyLLM{claims: moonClaims, judge: yes}
	srv, web := liveHasher(t, true, llm)
	_, out, _ := post(t, srv, "/models/hasher/verify", "", map[string]any{"text": "The Moon is 384400 kilometres from the Earth.", "delivery": "inline", "budget_ms": 20000})
	if out.Mode != sdk.ModeIndex || out.FlashUsed {
		t.Fatalf("%+v", out)
	}
	web.mu.Lock()
	defer web.mu.Unlock()
	if len(web.queries) != 0 {
		t.Fatalf("index mode contacted a provider: %v", web.queries)
	}
}

func TestResponseCheckRejectsIndexModeThatClaimsToHaveUsedTheWeb(t *testing.T) {
	_ = http.StatusOK
	good := sdk.VerifyResponse{Status: sdk.VerifyDone, Gate: "uncommitted", GateLevel: 2, Mode: sdk.ModeIndex, Claims: []sdk.VerifyClaim{}, Citations: []int{}}
	if err := good.Check(); err != nil {
		t.Fatal(err)
	}
	bad := good
	bad.FlashUsed = true
	if bad.Check() == nil {
		t.Fatal("index mode with flash_used passed Check")
	}
	bad = good
	bad.OutboundQueries = []string{"metasearch: x"}
	if bad.Check() == nil {
		t.Fatal("index mode with outbound queries passed Check")
	}
}

type testWitnesser struct{}

func (testWitnesser) Witness(_ context.Context, e evidence.WitnessEntry) (evidence.WitnessAttestation, error) {
	return evidence.WitnessAttestation{Kind: "test", Ref: "assertion-" + e.BatchID}, nil
}

// Task 3.9: evidence carries its provenance only once its batch root is witnessed.
func TestVerifyEvidenceCarriesAWitnessOnlyAfterItsBatchIsWitnessed(t *testing.T) {
	llm := &verifyLLM{claims: parisClaim, judge: yes}
	h, srv, _ := verifierHasher(t, llm, nil)
	body := map[string]any{"text": "Paris is the capital of France.", "delivery": "inline", "budget_ms": 20000}
	_, before, _ := post(t, srv, "/models/hasher/verify", "", body)
	for _, e := range before.Claims[0].Evidence {
		if e.Witness != nil {
			t.Fatalf("a span with no witnessed root claims provenance: %+v", e)
		}
	}
	ix, err := h.Index()
	if err != nil {
		t.Fatal(err)
	}
	if n, err := ix.Witnesses().Drain(context.Background(), testWitnesser{}, 0); err != nil || n == 0 {
		t.Fatalf("%d %v", n, err)
	}
	_, after, _ := post(t, srv, "/models/hasher/verify", "acme", body) // another tenant: not served from the cache
	got := 0
	for _, e := range after.Claims[0].Evidence {
		if e.SourceTier == "reference" {
			if e.Witness == nil || !strings.HasPrefix(e.Witness.Proof, "merkle:") || !strings.HasPrefix(e.Witness.BatchRoot, "sha256:") || !strings.HasPrefix(e.Witness.Attestation, "test:assertion-") {
				t.Fatalf("a witnessed reference span must carry its provenance: %+v", e)
			}
			got++
		}
	}
	if got == 0 {
		t.Fatal("no reference evidence")
	}
}
