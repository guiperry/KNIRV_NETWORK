package inferencer

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	models "github.com/guiperry/knirv-sdk-go/models"
)

// mirrorFake is a configurable daemon fake for the Phase 3 mirror tests. It records the
// raw request bodies of the HEAD and HEART generate calls (so FIX 7 can prove the HEAD
// request never carries a decision envelope while the HEART request does) and serves a
// HASHER verdict chosen per test (so FIX 6 can exercise refutation / unverified / clean).
type mirrorFake struct {
	mu sync.Mutex

	headBody  string // last HEAD generate request body
	heartBody string // last HEART generate request body

	headDown          bool // HEAD generate returns 503
	heartDown         bool // HEART generate returns 503
	headVerdict       string
	heartVerdict      string
	hasherUnverified  bool
	hasherDegraded    bool
	decideName        string // decision name surfaced by the decision core
	decideSelected    string
}

func (f *mirrorFake) server(t *testing.T) *models.Client {
	mux := http.NewServeMux()
	mux.HandleFunc(models.RegistryPath, func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(rows(true, true, true))
	})
	mux.HandleFunc(models.HeadPrefix+"/models", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"version":"head-9","config":{"max_seq_len":40,"summary_slots":8}}`))
	})
	mux.HandleFunc(models.HeadPrefix+"/generate", func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		f.mu.Lock()
		f.headBody = string(b)
		f.mu.Unlock()
		if f.headDown {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`{"text":"from head"}`))
	})
	mux.HandleFunc(models.HeartPrefix+"/generate", func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		f.mu.Lock()
		f.heartBody = string(b)
		f.mu.Unlock()
		if f.heartDown {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`{"text":"from heart","ready":true,"progress":1}`))
	})
	mux.HandleFunc(models.HeartDecisionPrefix+"/reason", func(w http.ResponseWriter, _ *http.Request) {
		name, sel := f.decideName, f.decideSelected
		if name == "" {
			name = "mirror"
			sel = "heart"
		}
		dec := `[{"name":"` + name + `","kind":"choice","options":["heart","head"],"probabilities":[0.9,0.1],"selected":"` + sel + `","confidence":0.9,"abstain":false}]`
		_, _ = w.Write([]byte(`{"plan_tokens":[],"halting_steps":1,"confidence":0.9,"decisions":` + dec + `,"model_version":"head-9"}`))
	})
	mux.HandleFunc(models.HasherPrefix+"/verify", func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		var req models.VerifyRequest
		_ = json.Unmarshal(b, &req)
		verdict := f.headVerdict
		if strings.Contains(req.Text, "heart") {
			verdict = f.heartVerdict
		}
		_ = json.NewEncoder(w).Encode(models.VerifyResponse{
			Status:     models.VerifyDone,
			Verdict:    verdict,
			Confidence: 0.9,
			Gate:       verdict,
			Unverified: f.hasherUnverified,
			Degraded:   f.hasherDegraded,
		})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return models.NewClient(srv.URL, nil)
}

// TestGenerateMirroredRefutedByHasher (FIX 6): HASHER refutes a candidate and it is
// disqualified; when it refutes both there is no survivor.
func TestGenerateMirroredRefutedByHasher(t *testing.T) {
	ctx := context.Background()

	// HEART refuted, HEAD clean -> HEAD must win and the trace records the refutation.
	f := &mirrorFake{headVerdict: "supported", heartVerdict: "refuted"}
	f.decideName = "mirror"
	f.decideSelected = "heart"
	c := f.server(t)
	mr, err := GenerateMirrored(ctx, c, "p", "", nil, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if mr.Winner != "head" || mr.Text != "from head" {
		t.Fatalf("refuted heart must lose to clean head: %+v", mr)
	}
	if !mr.Candidates["heart"].Refuted {
		t.Fatalf("heart candidate must be flagged refuted: %+v", mr.Candidates)
	}
	if !traceHasRefuted(mr.Trace, "heart") {
		t.Fatalf("mirror trace must record heart refuted: %+v", mr.Trace)
	}

	// Both refuted -> no survivor.
	f2 := &mirrorFake{headVerdict: "refuted", heartVerdict: "refuted"}
	c2 := f2.server(t)
	if _, err := GenerateMirrored(ctx, c2, "p", "", nil, "", nil); err == nil {
		t.Fatal("both refuted must error")
	}
}

// TestGenerateMirroredOneSide503 (FIX 6 + 7): one candidate's generation 503s but the
// other succeeds; the survivor wins cleanly.
func TestGenerateMirroredOneSide503(t *testing.T) {
	ctx := context.Background()

	// HEAD 503, HEART ok -> HEART wins; no decision envelope problem (HEAD never had one).
	f := &mirrorFake{headDown: true, headVerdict: "supported", heartVerdict: "supported"}
	c := f.server(t)
	mr, err := GenerateMirrored(ctx, c, "p", "", nil, "", nil)
	if err != nil || mr.Winner != "heart" || mr.Text != "from heart" {
		t.Fatalf("head down -> heart wins: %+v %v", mr, err)
	}

	// HEART 503, HEAD ok -> HEAD wins.
	f2 := &mirrorFake{heartDown: true, headVerdict: "supported", heartVerdict: "supported"}
	c2 := f2.server(t)
	mr2, err := GenerateMirrored(ctx, c2, "p", "", nil, "", nil)
	if err != nil || mr2.Winner != "head" || mr2.Text != "from head" {
		t.Fatalf("heart down -> head wins: %+v %v", mr2, err)
	}
}

// TestGenerateMirroredUnverifiedCacheNegative (FIX 6): HASHER returns unverified / cache
// negative. The flow degrades gracefully: it does not trust the candidate with a
// HASHER-derived score and falls back to supplied/default weights rather than erroring.
func TestGenerateMirroredUnverifiedCacheNegative(t *testing.T) {
	ctx := context.Background()
	f := &mirrorFake{hasherUnverified: true}
	c := f.server(t)
	mr, err := GenerateMirrored(ctx, c, "p", "", nil, "", nil)
	if err != nil {
		t.Fatalf("unverified HASHER must degrade gracefully, not error: %v", err)
	}
	if !traceHasUnverified(mr.Trace) {
		t.Fatalf("mirror trace must record the unverified/cache-negative outcome: %+v", mr.Trace)
	}
	// With both unverified and equal fallback weights (1.0 heart vs 0.9 head) HEART wins
	// the tie; crucially neither candidate was trusted as REFUTED.
	if mr.Refuted {
		t.Fatalf("unverified must not be treated as refuted: %+v", mr)
	}
}

// TestGenerateMirroredHeadNeverCarriesDecision (FIX 7): the HEAD request body never
// contains decision metadata; the HEART request body does, and carries the decision the
// core produced.
func TestGenerateMirroredHeadNeverCarriesDecision(t *testing.T) {
	ctx := context.Background()
	f := &mirrorFake{headVerdict: "supported", heartVerdict: "supported"}
	f.decideName = "mirror"
	f.decideSelected = "heart"
	c := f.server(t)
	mr, err := GenerateMirrored(ctx, c, "p", "", nil, "", nil)
	if err != nil {
		t.Fatal(err)
	}

	f.mu.Lock()
	headBody := f.headBody
	heartBody := f.heartBody
	f.mu.Unlock()

	if strings.Contains(strings.ToLower(headBody), "decision") {
		t.Fatalf("HEAD request must NOT carry a decision envelope: %s", headBody)
	}
	if !strings.Contains(heartBody, "decision") {
		t.Fatalf("HEART request MUST carry the decision envelope: %s", heartBody)
	}
	// The decision the core produced ("mirror") is reflected in the HEART request and the
	// mirror trace, but never in HEAD.
	if !strings.Contains(heartBody, "mirror") {
		t.Fatalf("HEART request must carry the decision-core output: %s", heartBody)
	}
	if strings.Contains(headBody, "mirror") {
		t.Fatalf("HEAD request must not carry decision content: %s", headBody)
	}
	if mr.Trace == nil || mr.Trace.Decision == nil || len(mr.Trace.Decision.Decisions) == 0 {
		t.Fatalf("mirror trace must include the HEART decision: %+v", mr.Trace)
	}
	if mr.Trace.Decision.Decisions[0].Name != "mirror" {
		t.Fatalf("mirror trace decision mismatch: %+v", mr.Trace.Decision)
	}
}

func traceHasRefuted(tr *MirrorTrace, model string) bool {
	if tr == nil {
		return false
	}
	for _, ct := range tr.Candidates {
		if ct.Model == model && ct.Refuted {
			return true
		}
	}
	return false
}

func traceHasUnverified(tr *MirrorTrace) bool {
	if tr == nil {
		return false
	}
	for _, ct := range tr.Candidates {
		if ct.Unverified {
			return true
		}
	}
	return false
}
