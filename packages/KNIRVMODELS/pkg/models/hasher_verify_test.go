package models

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"knirvmodels/pkg/evidence"
	"knirvmodels/pkg/heartqwen"

	sdk "github.com/guiperry/knirv-sdk-go/models"
)

var passageRE = regexp.MustCompile(`<<<BEGIN PASSAGE ([0-9a-f]+)>>>`)

// verifyLLM is a fake model: it extracts the claims it is told to, and judges by a
// supplied function. It counts judge calls.
type verifyLLM struct {
	claims  []map[string]any
	judge   func(user string) (string, error)
	gate    chan struct{} // when set, judge blocks until it is closed
	judged  int32
	extract int32
}

func (l *verifyLLM) Complete(ctx context.Context, r heartqwen.Request) (string, error) {
	user := r.Messages[len(r.Messages)-1].Content
	switch r.Frame.Stage {
	case "hasher.extract":
		atomic.AddInt32(&l.extract, 1)
		b, _ := json.Marshal(map[string]any{"claims": l.claims})
		return string(b), nil
	case "hasher.judge":
		atomic.AddInt32(&l.judged, 1)
		if l.gate != nil {
			select {
			case <-l.gate:
			case <-ctx.Done():
				return "", ctx.Err()
			}
		}
		return l.judge(user)
	}
	return "", errors.New("unexpected stage " + r.Frame.Stage)
}

func yes(user string) (string, error) {
	var ids []string
	for _, m := range passageRE.FindAllStringSubmatch(user, -1) {
		ids = append(ids, m[1])
	}
	b, _ := json.Marshal(map[string]any{"verdict": "SUPPORTED", "confidence": 0.97, "spans": ids})
	return string(b), nil
}

func verifierHasher(t *testing.T, llm *verifyLLM, live func(context.Context) bool) (*Hasher, *httptest.Server, string) {
	t.Helper()
	dir := t.TempDir()
	ix, err := evidence.Open(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ix.Ingest(evidence.Batch{Tier: evidence.TierReference, Docs: []evidence.Doc{
		{ID: "a", Title: "Paris", Text: "Paris is the capital and largest city of France. It lies on the Seine."},
		{ID: "b", Title: "France", Text: "The capital of France is Paris, which is also its seat of government."},
		{ID: "c", Title: "Eiffel Tower", Text: "The Eiffel Tower is a wrought-iron lattice tower in Paris.",
			Facts: []evidence.Fact{{Subject: "Eiffel Tower", Property: "height", Value: "330 metres"}}},
	}}); err != nil {
		t.Fatal(err)
	}
	if _, err := ix.Ingest(evidence.Batch{Tier: evidence.TierUser, Tenant: "acme", Docs: []evidence.Doc{{ID: "n", Text: "Acme internal note: the Paris office opens in 2027."}}}); err != nil {
		t.Fatal(err)
	}
	ix.Close()
	h := NewHasher(nil)
	h.UseIndex(dir, "")
	if live == nil {
		live = func(context.Context) bool { return true }
	}
	h.UseVerifier(VerifierConfig{
		LLM: llm, Live: live,
		Judge: func(context.Context) sdk.VerifyJudge {
			return sdk.VerifyJudge{Model: "qwen-test", SHA256: "abc", Adapters: "sha256:none"}
		},
	})
	mux := http.NewServeMux()
	h.Mount(mux, "/models/hasher")
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return h, srv, dir
}

func post(t *testing.T, srv *httptest.Server, path, tenant string, body any) (int, sdk.VerifyResponse, string) {
	t.Helper()
	raw, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, srv.URL+path, bytes.NewReader(raw))
	if tenant != "" {
		req.Header.Set(TenantHeader, tenant)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var buf bytes.Buffer
	buf.ReadFrom(res.Body)
	var out sdk.VerifyResponse
	_ = json.Unmarshal(buf.Bytes(), &out)
	if res.StatusCode == http.StatusOK {
		if err := out.Check(); err != nil {
			t.Errorf("the response breaks the contract: %v\n%s", err, buf.String())
		}
	}
	return res.StatusCode, out, buf.String()
}

func getJob(t *testing.T, srv *httptest.Server, id, tenant string) (int, sdk.VerifyResponse) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/models/hasher/verify/jobs/"+id, nil)
	if tenant != "" {
		req.Header.Set(TenantHeader, tenant)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var out sdk.VerifyResponse
	_ = json.NewDecoder(res.Body).Decode(&out)
	if res.StatusCode == http.StatusOK {
		if err := out.Check(); err != nil {
			t.Errorf("the job response breaks the contract: %v", err)
		}
	}
	return res.StatusCode, out
}

func waitJob(t *testing.T, srv *httptest.Server, id, tenant string) sdk.VerifyResponse {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		code, out := getJob(t, srv, id, tenant)
		if code == http.StatusOK && out.Status == sdk.VerifyDone {
			return out
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("job did not finish")
	return sdk.VerifyResponse{}
}

var parisClaim = []map[string]any{{"text": "Paris is the capital of France", "checkable": true}}

func TestVerifyInlineReturnsCitationsAndAGate(t *testing.T) {
	llm := &verifyLLM{claims: parisClaim, judge: yes}
	_, srv, _ := verifierHasher(t, llm, nil)
	code, out, raw := post(t, srv, "/models/hasher/verify", "", map[string]any{"text": "Paris is the capital of France.", "delivery": "inline"})
	if code != 200 || out.Status != sdk.VerifyDone || len(out.Claims) != 1 {
		t.Fatalf("%d %s", code, raw)
	}
	c := out.Claims[0]
	if c.Verdict != "SUPPORTED" || c.Source != "judge" || len(c.Cited) < 2 || c.Gate != "seen" || out.GateLevel != 3 {
		t.Fatalf("%+v", out)
	}
	// Valid citations: every cited span is in the evidence, and the legacy positions point at them.
	have := map[string]bool{}
	for _, e := range c.Evidence {
		have[e.SpanID] = true
	}
	for _, id := range c.Cited {
		if !have[id] {
			t.Fatalf("cited span %s is not in the evidence", id)
		}
	}
	if len(out.Citations) != len(c.Cited) {
		t.Fatalf("legacy citations %v", out.Citations)
	}
	for _, p := range out.Citations {
		if p < 0 || p >= len(c.Evidence) {
			t.Fatalf("citation %d out of range", p)
		}
	}
	if out.Judge == nil || out.Judge.Model != "qwen-test" || out.Judge.SHA256 != "abc" || out.IndexVersion == "" || out.Mode != "index" ||
		out.Calibrated || out.Degraded || !out.Chargeable || out.Verdict != "supported" || out.IsAccurate || out.Refused {
		t.Fatalf("%+v", out)
	}
	if !strings.Contains(strings.Join(out.Notes, " "), "calibration is not installed") {
		t.Fatalf("notes %v", out.Notes)
	}
}

// Done-when: a flagged hallucination returns valid citations and a gate.
func TestFlaggedHallucinationGetsCitationsAndARefusedGate(t *testing.T) {
	llm := &verifyLLM{claims: []map[string]any{{"text": "The Eiffel Tower is 450 metres tall", "subject": "Eiffel Tower", "property": "height", "value": "450 metres", "checkable": true}}, judge: yes}
	_, srv, _ := verifierHasher(t, llm, nil)
	code, out, raw := post(t, srv, "/models/hasher/verify", "", map[string]any{"text": "The Eiffel Tower is 450 metres tall.", "delivery": "inline"})
	if code != 200 || len(out.Claims) != 1 {
		t.Fatalf("%d %s", code, raw)
	}
	c := out.Claims[0]
	if c.Verdict != "REFUTED" || c.Source != "structured" || c.Gate != "refused" || out.GateLevel != 0 || !out.Refused || out.Verdict != "refuted" {
		t.Fatalf("%+v", out)
	}
	if len(c.Cited) != 1 || len(out.Citations) != 1 || !strings.Contains(c.Evidence[out.Citations[0]].Text, "330 metres") {
		t.Fatalf("the refutation must cite the fact that contradicts it: %+v", c)
	}
	if llm.judged != 0 {
		t.Fatal("the model judged a case code can decide")
	}
}

func TestDeferredIsTheDefaultAndTheJobIsTenantScoped(t *testing.T) {
	llm := &verifyLLM{claims: parisClaim, judge: yes}
	_, srv, _ := verifierHasher(t, llm, nil)
	code, out, _ := post(t, srv, "/models/hasher/verify", "acme", map[string]any{"text": "Paris is the capital of France."})
	if code != 200 || out.Status != sdk.VerifyPending || out.JobID == "" || out.Gate != "uncommitted" || out.GateLevel != 2 ||
		!out.Unverified || out.Chargeable || out.Delivery != "deferred" || len(out.Claims) != 0 {
		t.Fatalf("%d %+v", code, out)
	}
	final := waitJob(t, srv, out.JobID, "acme")
	if len(final.Claims) != 1 || final.Claims[0].Verdict != "SUPPORTED" || final.JobID != out.JobID {
		t.Fatalf("%+v", final)
	}
	if code, _ := getJob(t, srv, out.JobID, "globex"); code != http.StatusNotFound {
		t.Fatalf("another tenant read the job: %d", code)
	}
	if code, _ := getJob(t, srv, out.JobID, ""); code != http.StatusNotFound {
		t.Fatalf("an anonymous caller read a tenant's job: %d", code)
	}
	if code, _ := getJob(t, srv, "nope", "acme"); code != http.StatusNotFound {
		t.Fatalf("unknown job: %d", code)
	}
}

func TestInlineOutOfBudgetReturnsPendingAndTheJobCompletesLater(t *testing.T) {
	gate := make(chan struct{})
	llm := &verifyLLM{claims: parisClaim, judge: yes, gate: gate}
	_, srv, _ := verifierHasher(t, llm, nil)
	start := time.Now()
	code, out, _ := post(t, srv, "/models/hasher/verify", "", map[string]any{"text": "Paris is the capital of France.", "delivery": "inline", "budget_ms": 1})
	if code != 200 || out.Status != sdk.VerifyPending || out.JobID == "" || out.Gate != "uncommitted" || time.Since(start) > 5*time.Second {
		t.Fatalf("%d %+v after %v", code, out, time.Since(start))
	}
	if !strings.Contains(strings.Join(out.Notes, " "), "not ready within budget_ms") {
		t.Fatalf("notes %v", out.Notes)
	}
	close(gate)
	if final := waitJob(t, srv, out.JobID, ""); final.Claims[0].Verdict != "SUPPORTED" {
		t.Fatalf("%+v", final)
	}
}

// Llama down: never a verdict, never charged.
func TestJudgeDownIsDegradedUncommittedAndNeverAVerdict(t *testing.T) {
	llm := &verifyLLM{claims: parisClaim, judge: yes}
	_, srv, _ := verifierHasher(t, llm, func(context.Context) bool { return false })
	code, out, raw := post(t, srv, "/models/hasher/verify", "", map[string]any{"text": "Paris is the capital of France.", "delivery": "inline"})
	if code != 200 || !out.Degraded || out.Chargeable || out.Gate != "uncommitted" || out.GateLevel != 2 || out.Refused || out.IsAccurate {
		t.Fatalf("%d %s", code, raw)
	}
	if len(out.Claims) != 1 || out.Claims[0].Verdict != "NEI" || !out.Claims[0].Degraded || len(out.Claims[0].Cited) != 0 {
		t.Fatalf("%+v", out.Claims)
	}
	if llm.extract != 0 || llm.judged != 0 {
		t.Fatal("a model that is not up was called")
	}
	// A judge that errors mid-run degrades the same way.
	llm2 := &verifyLLM{claims: parisClaim, judge: func(string) (string, error) { return "", errors.New("llama exploded") }}
	_, srv2, _ := verifierHasher(t, llm2, nil)
	_, out, _ = post(t, srv2, "/models/hasher/verify", "", map[string]any{"text": "Paris is the capital of France.", "delivery": "inline"})
	if !out.Degraded || out.Chargeable || out.Gate != "uncommitted" || out.Claims[0].Verdict != "NEI" {
		t.Fatalf("%+v", out)
	}
}

func TestClaimCacheKeyedByIndexVersionAndTenant(t *testing.T) {
	llm := &verifyLLM{claims: parisClaim, judge: yes}
	h, srv, dir := verifierHasher(t, llm, nil)
	body := map[string]any{"text": "Paris is the capital of France.", "delivery": "inline"}
	_, first, _ := post(t, srv, "/models/hasher/verify", "", body)
	calls := atomic.LoadInt32(&llm.judged)
	_, second, _ := post(t, srv, "/models/hasher/verify", "", body)
	if !second.Claims[0].Cached || atomic.LoadInt32(&llm.judged) != calls || second.Claims[0].Verdict != first.Claims[0].Verdict {
		t.Fatalf("second request was not served from the cache: %+v", second.Claims[0])
	}
	// Another tenant never shares a cached result.
	_, third, _ := post(t, srv, "/models/hasher/verify", "acme", body)
	if third.Claims[0].Cached {
		t.Fatal("a cached result crossed tenants")
	}
	// A new index version invalidates.
	ix, err := h.Index()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ix.Ingest(evidence.Batch{Tier: evidence.TierReference, Docs: []evidence.Doc{{ID: "d", Text: "Paris has been the capital of France for centuries."}}}); err != nil {
		t.Fatal(err)
	}
	_, fourth, _ := post(t, srv, "/models/hasher/verify", "", body)
	if fourth.Claims[0].Cached || fourth.IndexVersion == first.IndexVersion {
		t.Fatalf("a changed index must not serve a stale verdict: %+v", fourth)
	}
	_ = dir
}

func TestVerifyRejectsBadRequests(t *testing.T) {
	llm := &verifyLLM{claims: parisClaim, judge: yes}
	_, srv, _ := verifierHasher(t, llm, nil)
	many := make([]map[string]any, 40)
	for i := range many {
		many[i] = map[string]any{"text": "claim"}
	}
	for name, body := range map[string]any{
		"empty":         map[string]any{},
		"bad mode":      map[string]any{"text": "x y z", "mode": "quantum"},
		"bad delivery":  map[string]any{"text": "x y z", "delivery": "sometime"},
		"too many":      map[string]any{"claims": many, "max_claims": 4},
		"huge domains":  map[string]any{"text": "x y z", "domains": make([]string, 40)},
		"empty domain":  map[string]any{"text": "x y z", "domains": []string{""}},
		"oversize text": map[string]any{"text": strings.Repeat("a", 70<<10)},
	} {
		if code, _, raw := post(t, srv, "/models/hasher/verify", "", body); code != http.StatusBadRequest {
			t.Errorf("%s: %d %s", name, code, raw)
		}
	}
	res, _ := http.Get(srv.URL + "/models/hasher/verify")
	if res.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("GET: %d", res.StatusCode)
	}
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/models/hasher/verify", strings.NewReader("{nope"))
	if res, _ := http.DefaultClient.Do(req); res.StatusCode != http.StatusBadRequest {
		t.Fatalf("bad json: %d", res.StatusCode)
	}
	h := NewHasher(nil)
	h.UseVerifier(VerifierConfig{LLM: llm})
	mux := http.NewServeMux()
	h.Mount(mux, "/models/hasher")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/models/hasher/verify", strings.NewReader(`{"text":"a b c"}`)))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("no index: %d", rec.Code)
	}
}

func TestFlashAndHybridAreAnsweredFromTheIndexAndSaySo(t *testing.T) {
	llm := &verifyLLM{claims: parisClaim, judge: yes}
	_, srv, _ := verifierHasher(t, llm, nil)
	for _, mode := range []string{"flash", "hybrid"} {
		_, out, _ := post(t, srv, "/models/hasher/verify", "", map[string]any{"text": "Paris is the capital of France.", "mode": mode, "delivery": "inline", "domains": []string{"geography"}})
		joined := strings.Join(out.Notes, " ")
		if out.Mode != "index" || !strings.Contains(joined, "live lookup is not enabled") || !strings.Contains(joined, "domains are recorded but not applied") || len(out.Claims) != 1 {
			t.Fatalf("%s: %+v", mode, out)
		}
	}
}

func TestCallerSuppliedClaimsAndMaxClaims(t *testing.T) {
	llm := &verifyLLM{judge: yes}
	_, srv, _ := verifierHasher(t, llm, nil)
	// Caller claims skip extraction; the rules still decide what is checkable.
	_, out, raw := post(t, srv, "/models/hasher/verify", "", map[string]any{"delivery": "inline", "claims": []map[string]any{
		{"text": "Paris is the capital of France"},
		{"text": "I think Paris is lovely"},
	}})
	if len(out.Claims) != 1 || len(out.Unchecked) != 1 || out.Unchecked[0].Why != "hedged" || llm.extract != 0 {
		t.Fatalf("%s", raw)
	}
	// More caller claims than max_claims is refused outright.
	three := []map[string]any{{"text": "Paris is on the Seine"}, {"text": "Paris is in France"}, {"text": "Paris is a city"}}
	if code, _, _ := post(t, srv, "/models/hasher/verify", "", map[string]any{"claims": three, "max_claims": 2}); code != http.StatusBadRequest {
		t.Fatalf("code %d", code)
	}
	// Extracted claims beyond max_claims are reported unchecked, not silently dropped.
	llm = &verifyLLM{judge: yes, claims: []map[string]any{
		{"text": "Paris is the capital of France", "checkable": true},
		{"text": "Paris lies on the Seine", "checkable": true},
		{"text": "Paris is a city in France", "checkable": true},
	}}
	_, srv2, _ := verifierHasher(t, llm, nil)
	_, out, raw = post(t, srv2, "/models/hasher/verify", "", map[string]any{"delivery": "inline", "max_claims": 1,
		"text": "Paris is the capital of France. Paris lies on the Seine. Paris is a city in France."})
	if len(out.Claims) != 1 || len(out.Unchecked) != 2 || out.Unchecked[0].Why != "over max_claims" || out.UncheckedFraction < 0.66 {
		t.Fatalf("%s", raw)
	}
	if !strings.Contains(strings.Join(out.Notes, " "), "beyond max_claims") {
		t.Fatalf("notes %v", out.Notes)
	}
}

func TestBusyQueueIsRefusedNotQueuedForever(t *testing.T) {
	gate := make(chan struct{})
	llm := &verifyLLM{claims: parisClaim, judge: yes, gate: gate}
	_, srv, _ := verifierHasher(t, llm, nil)
	var busy int
	for i := 0; i < verifyQueueDepth+4; i++ {
		code, _, _ := post(t, srv, "/models/hasher/verify", "", map[string]any{"text": "Paris is the capital of France."})
		if code == http.StatusServiceUnavailable {
			busy++
		}
	}
	close(gate)
	if busy < 4 {
		t.Fatalf("only %d requests refused", busy)
	}
}

func TestIndexStatusReportsStaleness(t *testing.T) {
	_, srv, _ := verifierHasher(t, &verifyLLM{claims: parisClaim, judge: yes}, nil)
	res, err := http.Get(srv.URL + "/models/hasher/index/status")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var st sdk.IndexStatus
	if err := json.NewDecoder(res.Body).Decode(&st); err != nil || st.Spans == 0 || st.Version == "" || st.LastIngest == "" || st.StaleSeconds < 0 {
		t.Fatalf("%+v %v", st, err)
	}
}

func TestSDKClientVerifyAndJob(t *testing.T) {
	llm := &verifyLLM{claims: parisClaim, judge: yes}
	_, srv, _ := verifierHasher(t, llm, nil)
	c := sdk.NewClient(srv.URL, nil)
	out, err := c.Verify(context.Background(), sdk.VerifyRequest{Text: "Paris is the capital of France.", Delivery: sdk.DeliveryInline, Tenant: "acme"})
	if err != nil || out.Status != sdk.VerifyDone || len(out.Claims) != 1 {
		t.Fatalf("%v %+v", err, out)
	}
	def, err := c.Verify(context.Background(), sdk.VerifyRequest{Text: "Paris is the capital of France.", Tenant: "acme"})
	if err != nil || def.Status != sdk.VerifyPending {
		t.Fatalf("%v %+v", err, def)
	}
	var job *sdk.VerifyResponse
	for i := 0; i < 500; i++ {
		if job, err = c.VerifyJob(context.Background(), def.JobID, "acme"); err == nil && job.Status == sdk.VerifyDone {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if job == nil || job.Status != sdk.VerifyDone {
		t.Fatalf("%v %+v", err, job)
	}
	if _, err := c.Verify(context.Background(), sdk.VerifyRequest{}); err == nil {
		t.Fatal("empty request accepted by the client")
	}
	if _, err := c.VerifyJob(context.Background(), "ghost", "acme"); err == nil {
		t.Fatal("unknown job returned no error")
	}
}
