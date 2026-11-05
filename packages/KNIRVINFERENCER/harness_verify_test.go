package inferencer

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	models "github.com/guiperry/knirv-sdk-go/models"
	"github.com/guiperry/knirv-sdk-go/models/frame"
)

// verifyDaemon is a fake KNIRVMODELS daemon: a registry with HEART and HASHER, HEART
// generating a fixed (wrong) answer, a capability manifest, and HASHER's verify route
// answering through a function the test supplies.
type verifyDaemon struct {
	mu       sync.Mutex
	requests []models.VerifyRequest
	headers  []http.Header
	verify   func(models.VerifyRequest) (int, any)
	heart    string
	hashes   map[string]string
}

func (d *verifyDaemon) client(t *testing.T) *models.Client {
	if d.hashes == nil {
		d.hashes = map[string]string{"hasher.extract": "aa", "hasher.judge": "bb", "heart.chat": "cc"}
	}
	mux := http.NewServeMux()
	mux.HandleFunc(models.RegistryPath, func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]models.ModelStatus{
			{Name: "heart", Version: "heart-v1", Ready: true, Loaded: true},
			{Name: "hasher", Version: "hasher-v1", Ready: true},
		})
	})
	mux.HandleFunc(models.HeartPrefix+"/health", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"available":true,"status":"online","loaded":true,"ready":true,"progress":1}`))
	})
	mux.HandleFunc(models.HeartPrefix+"/generate", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"text": d.heart})
	})
	mux.HandleFunc(models.ManifestPath, func(w http.ResponseWriter, _ *http.Request) {
		d.mu.Lock()
		defer d.mu.Unlock()
		_ = json.NewEncoder(w).Encode(frame.Manifest{Version: "2026-10-1", FrameHashes: d.hashes,
			Stages: make([]frame.Stage, 3), Tools: make([]frame.Tool, 5)})
	})
	mux.HandleFunc(models.VerifyPath, func(w http.ResponseWriter, r *http.Request) {
		var req models.VerifyRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		d.mu.Lock()
		d.requests = append(d.requests, req)
		d.headers = append(d.headers, r.Header.Clone())
		d.mu.Unlock()
		code, body := d.verify(req)
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(body)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return models.NewClient(srv.URL, nil)
}

func (d *verifyDaemon) last() models.VerifyRequest {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.requests[len(d.requests)-1]
}

const fact = "Eiffel Tower — height: 330 metres"

// truthful answers like the real daemon: a claim that contradicts the indexed fact is
// REFUTED with that fact as its citation; anything else is SUPPORTED at "seen".
func truthful(req models.VerifyRequest) (int, any) {
	resp := models.VerifyResponse{
		Status: models.VerifyDone, Chargeable: true, IndexVersion: "fs1-3-9f2c1a77", Mode: models.ModeIndex, Delivery: req.Delivery,
		Judge: &models.VerifyJudge{Model: "qwen-test", SHA256: "abc"}, Citations: []int{},
	}
	text := req.Text
	if len(req.Claims) > 0 {
		text = req.Claims[0].Text
	}
	ev := []models.VerifyEvidence{{SpanID: "sha256:" + strings.Repeat("c", 64), Snapshot: "sha256:" + strings.Repeat("9", 64), SourceTier: "reference", Text: fact}}
	if strings.Contains(text, "450 metres") {
		resp.Gate, resp.GateLevel, resp.Verdict, resp.Refused, resp.GateScore, resp.Confidence = "refused", 0, "refuted", true, 0.05, 0.05
		resp.Citations = []int{0}
		resp.Claims = []models.VerifyClaim{{Text: text, Type: "number", Verdict: "REFUTED", Source: "structured", Confidence: 0.95, Calibrated: 0.05,
			Gate: "refused", GateLevel: 0, Evidence: ev, Cited: []string{ev[0].SpanID}, Agreement: true}}
		return 200, resp
	}
	resp.Gate, resp.GateLevel, resp.Verdict, resp.GateScore, resp.Confidence = "seen", 3, "supported", 0.97, 0.97
	resp.Citations = []int{0}
	resp.Claims = []models.VerifyClaim{{Text: text, Type: "entity", Verdict: "SUPPORTED", Source: "judge", Confidence: 0.97, Calibrated: 0.97,
		Gate: "seen", GateLevel: 3, Evidence: ev, Cited: []string{ev[0].SpanID}, Agreement: true}}
	return 200, resp
}

func degraded(models.VerifyRequest) (int, any) {
	return 200, models.VerifyResponse{
		Status: models.VerifyDone, Gate: "uncommitted", GateLevel: 2, Verdict: "nei", Degraded: true, Mode: models.ModeIndex, Delivery: "inline",
		Claims:    []models.VerifyClaim{{Text: "x", Type: "general", Verdict: "NEI", Source: "none", Gate: "uncommitted", GateLevel: 2, Evidence: []models.VerifyEvidence{}, Degraded: true, Reason: "judge unavailable"}},
		Citations: []int{}, IndexVersion: "fs1-3-9f2c1a77",
	}
}

func hasherHarness(t *testing.T, d *verifyDaemon, opts ...HarnessOption) (*Harness, *HasherModel, *HeartModel) {
	t.Helper()
	_, heart, hasher := DaemonModels(d.client(t))
	return NewHarness([]HarnessModel{heart, hasher}, opts...), hasher, heart
}

func TestVerifyIsRoutedToHasherOnly(t *testing.T) {
	for name, r := range map[string]map[Kind][]string{"default": DefaultRoutes(), "user facing": UserFacingRoutes()} {
		if got := r[KindVerify]; len(got) != 1 || got[0] != "hasher" {
			t.Errorf("%s: verify route = %v", name, got)
		}
	}
	// Without HASHER registered the task fails and the trace says why.
	h := NewHarness([]HarnessModel{up("heart")})
	_, tr, err := h.Do(context.Background(), Task{Kind: KindVerify, Input: models.VerifyRequest{Text: "x"}})
	if err == nil || len(tr.Candidates) != 1 || tr.Candidates[0].Excluded != "not registered" {
		t.Fatalf("%v %+v", err, tr)
	}
	// A pin to a model that is not on the route cannot widen it.
	d := &verifyDaemon{verify: truthful}
	h, _, _ = hasherHarness(t, d)
	if _, _, err := h.Verify(context.Background(), models.VerifyRequest{Text: "x"}, Constraints{Pin: "heart"}); err == nil {
		t.Fatal("a pin to heart reached hasher's task")
	}
}

// Done-when: a wrong HEART answer is flagged with citations and a gate.
func TestWrongHeartAnswerIsFlaggedWithCitationsAndAGate(t *testing.T) {
	d := &verifyDaemon{verify: truthful, heart: "The Eiffel Tower is 450 metres tall."}
	dir := t.TempDir()
	store, err := NewTraceStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	h, _, _ := hasherHarness(t, d, WithTraceStore(store), WithRoutes(map[Kind][]string{KindGenerate: {"heart"}, KindVerify: {"hasher"}}))
	ctx := context.Background()

	gen, gtr, err := h.Do(ctx, Task{Kind: KindGenerate, Input: "How tall is the Eiffel Tower?"})
	if err != nil || gtr.Chosen != "heart" || gen.Text != d.heart {
		t.Fatalf("generate: %v %+v", err, gtr)
	}
	resp, vtr, err := h.Verify(ctx, models.VerifyRequest{Text: gen.Text, Delivery: models.DeliveryInline, Tenant: "acme"}, Constraints{NoFlash: true, Budget: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Gate != "refused" || resp.GateLevel != 0 || !resp.Refused || resp.Verdict != "refuted" || len(resp.Claims) != 1 {
		t.Fatalf("the wrong answer was not flagged: %+v", resp)
	}
	c := resp.Claims[0]
	if len(resp.Citations) != 1 || !strings.Contains(c.Evidence[resp.Citations[0]].Text, "330 metres") || c.Cited[0] != c.Evidence[0].SpanID {
		t.Fatalf("the flag must cite the contradicting fact: %+v", resp)
	}
	// The request reached the daemon with the constraints applied and the tenant header.
	req := d.last()
	if req.Mode != models.ModeIndex || req.BudgetMS != 5000 || req.Delivery != models.DeliveryInline || d.headers[0].Get("X-KNIRV-Tenant") != "acme" {
		t.Fatalf("%+v %v", req, d.headers[0])
	}

	// The Trace records the verification.
	v := vtr.Verify
	if vtr.Kind != KindVerify || vtr.Chosen != "hasher" || !vtr.NoFlash || v == nil || v.Mode != "index" || v.FlashUsed || v.Gate != "refused" ||
		v.GateLevel != 0 || v.HitRate != 1 || v.Claims != 1 || v.Degraded || v.Pending || v.IndexVersion != "fs1-3-9f2c1a77" || v.Judge != "qwen-test@abc" {
		t.Fatalf("%+v %+v", vtr, v)
	}
	raw, _ := json.Marshal(vtr)
	for _, banned := range []string{"450 metres", "330 metres", "Eiffel"} {
		if strings.Contains(string(raw), banned) {
			t.Fatalf("the trace carries text %q: %s", banned, raw)
		}
	}

	// A correct answer is not flagged.
	ok, _, err := h.Verify(ctx, models.VerifyRequest{Text: "The Eiffel Tower is 330 metres tall.", Delivery: models.DeliveryInline}, Constraints{})
	if err != nil || ok.Gate != "seen" || ok.Refused {
		t.Fatalf("%v %+v", err, ok)
	}

	// Restart: both traces come back from disk, newest first, with the verify fields.
	_ = store.Close()
	store2, err := NewTraceStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	h2, _, _ := hasherHarness(t, &verifyDaemon{verify: truthful}, WithTraceStore(store2))
	got := h2.Traces(10)
	if len(got) != 3 || got[0].Kind != KindVerify || got[1].Kind != KindVerify || got[2].Kind != KindGenerate {
		t.Fatalf("reloaded %d traces: %+v", len(got), got)
	}
	if got[1].ID != vtr.ID || got[1].Verify == nil || got[1].Verify.Gate != "refused" || !got[1].NoFlash {
		t.Fatalf("%+v", got[1])
	}
	if ev := h2.RoutingEvents(10); len(ev) != 3 || ev[2].Kind != "generate" {
		t.Fatalf("%+v", ev)
	}
}

func TestNoFlashForcesIndexAndRefusesAFlashAnswer(t *testing.T) {
	d := &verifyDaemon{verify: truthful}
	h, _, _ := hasherHarness(t, d)
	for name, c := range map[string]Constraints{"no_flash": {NoFlash: true}, "local_only": {LocalOnly: true}} {
		_, tr, err := h.Verify(context.Background(), models.VerifyRequest{Text: "x y", Mode: models.ModeHybrid}, c)
		if err != nil || d.last().Mode != models.ModeIndex || !strings.Contains(strings.Join(tr.Notes, " "), "no flash") {
			t.Fatalf("%s: %v mode=%q notes=%v", name, err, d.last().Mode, tr.Notes)
		}
	}
	// Without the constraint the mode the caller asked for is sent as asked.
	if _, _, err := h.Verify(context.Background(), models.VerifyRequest{Text: "x y", Mode: models.ModeFlash}, Constraints{}); err != nil || d.last().Mode != models.ModeFlash {
		t.Fatalf("%v %q", err, d.last().Mode)
	}
	// A daemon that ignores the request and runs a live lookup anyway is refused.
	d.verify = func(req models.VerifyRequest) (int, any) {
		code, body := truthful(req)
		r := body.(models.VerifyResponse)
		r.Mode = models.ModeFlash
		return code, r
	}
	resp, tr, err := h.Verify(context.Background(), models.VerifyRequest{Text: "x y"}, Constraints{NoFlash: true})
	if err == nil || resp != nil || !strings.Contains(tr.Error, "constraint violated") || !tr.NoFlash {
		t.Fatalf("a flash answer reached a no-flash caller: %v %+v", err, tr)
	}
}

func TestVerifyBudgetBecomesBudgetMsAndAPendingAnswerIsNotAVerdict(t *testing.T) {
	d := &verifyDaemon{verify: func(req models.VerifyRequest) (int, any) {
		return 200, models.VerifyResponse{
			Status: models.VerifyPending, JobID: "job-1", Gate: "uncommitted", GateLevel: 2, Verdict: "unchecked", Unverified: true,
			Claims: []models.VerifyClaim{}, Citations: []int{}, Mode: models.ModeIndex, Delivery: req.Delivery, IndexVersion: "v",
		}
	}}
	h, hasher, _ := hasherHarness(t, d)
	res, tr, err := h.Do(context.Background(), Task{Kind: KindVerify, Input: models.VerifyRequest{Text: "x y"}, Constraints: Constraints{Budget: 3 * time.Second}})
	if err != nil {
		t.Fatal(err)
	}
	req := d.last()
	if req.BudgetMS != 3000 || req.Delivery != models.DeliveryInline {
		t.Fatalf("%+v", req)
	}
	if !res.Degraded || tr.Verify == nil || !tr.Verify.Pending || tr.Verify.JobID != "job-1" || tr.Verify.Gate != "uncommitted" || !strings.Contains(strings.Join(tr.Notes, " "), "job-1") {
		t.Fatalf("%+v %+v", res, tr)
	}
	// A deferred delivery the caller chose is kept.
	_, _, _ = h.Verify(context.Background(), models.VerifyRequest{Text: "x y", Delivery: models.DeliveryDeferred}, Constraints{Budget: time.Second})
	if d.last().Delivery != models.DeliveryDeferred {
		t.Fatalf("%+v", d.last())
	}
	// HasherModel.Verify and VerifyJob go through the same client.
	if resp, err := hasher.Verify(context.Background(), models.VerifyRequest{Text: "x y"}, Constraints{}); err != nil || resp.JobID != "job-1" {
		t.Fatalf("%v %+v", err, resp)
	}
}

func TestVerifyBudgetExpiryFailsTheCallAndTheTraceSaysSo(t *testing.T) {
	d := &verifyDaemon{verify: func(models.VerifyRequest) (int, any) {
		time.Sleep(300 * time.Millisecond)
		return truthful(models.VerifyRequest{Text: "x"})
	}}
	h, _, _ := hasherHarness(t, d)
	start := time.Now()
	_, tr, err := h.Verify(context.Background(), models.VerifyRequest{Text: "x y"}, Constraints{Budget: 50 * time.Millisecond})
	if err == nil || tr.Error == "" || time.Since(start) > 2*time.Second {
		t.Fatalf("%v %+v", err, tr)
	}
}

func TestDegradedVerifyIsAResultNotAnErrorAndNeverAVerdict(t *testing.T) {
	d := &verifyDaemon{verify: degraded}
	h, _, _ := hasherHarness(t, d)
	resp, tr, err := h.Verify(context.Background(), models.VerifyRequest{Text: "x y"}, Constraints{})
	if err != nil || !resp.Degraded || resp.GateLevel != 2 || resp.Claims[0].Verdict != "NEI" {
		t.Fatalf("%v %+v", err, resp)
	}
	if !tr.Degraded || tr.Verify == nil || !tr.Verify.Degraded || tr.Verify.Gate != "uncommitted" || tr.Verify.HitRate != 0 {
		t.Fatalf("%+v", tr)
	}
	if st := h.Stats()["hasher"]; st.Failures != 0 || st.Circuit != "closed" {
		t.Fatalf("a degraded answer is not a failure: %+v", st)
	}
}

func TestMalformedVerifyAnswerIsRejected(t *testing.T) {
	d := &verifyDaemon{verify: func(req models.VerifyRequest) (int, any) {
		code, body := truthful(models.VerifyRequest{Text: "ok"})
		r := body.(models.VerifyResponse)
		r.Gate = "accepted" // does not match gate_level 3
		return code, r
	}}
	h, _, _ := hasherHarness(t, d)
	resp, tr, err := h.Verify(context.Background(), models.VerifyRequest{Text: "x y"}, Constraints{})
	if err == nil || resp != nil || !strings.Contains(tr.Error, "malformed") {
		t.Fatalf("%v %+v", err, tr)
	}
	if st := h.Stats()["hasher"]; st.Failures != 1 {
		t.Fatalf("%+v", st)
	}
	// A 503 from the daemon (no index) is an error too, with the SDK's sentinel text.
	d.verify = func(models.VerifyRequest) (int, any) { return 503, map[string]string{"error": "no index"} }
	if _, _, err := h.Verify(context.Background(), models.VerifyRequest{Text: "x y"}, Constraints{}); err == nil || !strings.Contains(err.Error(), "no evidence index") {
		t.Fatalf("%v", err)
	}
	// Bad input types are refused before any request.
	before := len(d.requests)
	if _, _, err := h.Do(context.Background(), Task{Kind: KindVerify, Input: 42}); err == nil || len(d.requests) != before {
		t.Fatalf("%v", err)
	}
}

func TestVerifyAcceptsATextPointerAndPlainText(t *testing.T) {
	d := &verifyDaemon{verify: truthful}
	h, _, _ := hasherHarness(t, d)
	for _, in := range []any{"The Eiffel Tower is 330 metres tall.", &models.VerifyRequest{Text: "The Eiffel Tower is 330 metres tall."}} {
		if _, _, err := h.Do(context.Background(), Task{Kind: KindVerify, Input: in}); err != nil {
			t.Fatalf("%T: %v", in, err)
		}
	}
	if d.last().Text == "" || d.last().Mode != models.ModeIndex {
		t.Fatalf("%+v", d.last())
	}
}

func TestPlatformStatusCarriesTheManifestAndLlamaCounters(t *testing.T) {
	d := &verifyDaemon{verify: truthful}
	client := d.client(t)
	_, _, hasher := DaemonModels(client)
	fl := &fakeLlama{}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{"status":"ok"}`)) })
	mux.HandleFunc("/v1/frame/stats", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"mode":"warn","constitution_version":"2026-10-1","framed":40,"unframed":2,"invalid":0,"rejected":0,"teacher":1}`))
	})
	mux.HandleFunc("/v1/queue/stats", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"free_slots":0,"depth":[0,1,2,3],"admitted":[5,6,7,8]}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	llama := fl.model(t, nil)
	llama.HTTP = &http.Client{Transport: &http.Transport{DialContext: func(ctx context.Context, n, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, n, srv.Listener.Addr().String())
	}}}
	h := NewHarness([]HarnessModel{hasher})
	s := BuildPlatformStatus(context.Background(), client, llama, h, 5)
	if s.Manifest == nil || s.Manifest.Error != "" || s.Manifest.Version != "2026-10-1" || s.Manifest.Stages != 3 || s.Manifest.Tools != 5 || len(s.Manifest.Hash) != 64 {
		t.Fatalf("manifest: %+v", s.Manifest)
	}
	first := s.Manifest.Hash
	d.mu.Lock()
	d.hashes["hasher.judge"] = "changed"
	d.mu.Unlock()
	if s2 := BuildPlatformStatus(context.Background(), client, nil, nil, 5); s2.Manifest.Hash == first {
		t.Fatal("the manifest hash did not change with a frame hash")
	}
	if s.Llama == nil || s.Llama.Frames == nil || s.Llama.Frames.Framed != 40 || s.Llama.Frames.Unframed != 2 || s.Llama.Frames.Mode != "warn" {
		t.Fatalf("framed/unframed counters: %+v", s.Llama)
	}
	if s.Llama.Queue == nil || s.Llama.Queue.Depth != [4]int{0, 1, 2, 3} || s.Llama.Queue.Admitted[3] != 8 {
		t.Fatalf("per-priority queue depth: %+v", s.Llama.Queue)
	}
	raw, _ := json.Marshal(s)
	if strings.Contains(string(raw), "aa") && strings.Contains(string(raw), "frame_hashes") {
		t.Fatalf("the status carries frame hashes: %s", raw)
	}

	// An unreadable manifest is reported, not hidden, and does not fail the document.
	dead := httptest.NewServer(http.NotFoundHandler())
	dead.Close()
	if s3 := BuildPlatformStatus(context.Background(), models.NewClient(dead.URL, nil), nil, nil, 5); s3.Manifest == nil || s3.Manifest.Error == "" {
		t.Fatalf("%+v", s3.Manifest)
	}
	if empty := BuildPlatformStatus(context.Background(), nil, nil, nil, 5); empty.Manifest != nil {
		t.Fatalf("%+v", empty.Manifest)
	}
}

// Task 3.8: when live lookup ran, the Trace records exactly what left the machine; and a
// no-flash caller refuses an answer that reports it, whatever mode it names.
func TestTraceRecordsOutboundQueriesAndNoFlashRefusesThem(t *testing.T) {
	d := &verifyDaemon{verify: func(req models.VerifyRequest) (int, any) {
		code, body := truthful(models.VerifyRequest{Text: "ok"})
		r := body.(models.VerifyResponse)
		r.Mode, r.FlashUsed, r.OutboundQueries = models.ModeHybrid, true, []string{"metasearch: moon distance kilometres"}
		return code, r
	}}
	h, _, _ := hasherHarness(t, d)
	resp, tr, err := h.Verify(context.Background(), models.VerifyRequest{Text: "x y", Mode: models.ModeHybrid}, Constraints{})
	if err != nil || !resp.FlashUsed {
		t.Fatalf("%v %+v", err, resp)
	}
	v := tr.Verify
	if v == nil || !v.FlashUsed || v.Mode != "hybrid" || len(v.OutboundQueries) != 1 || v.OutboundQueries[0] != "metasearch: moon distance kilometres" {
		t.Fatalf("%+v", v)
	}
	if _, tr, err := h.Verify(context.Background(), models.VerifyRequest{Text: "x y", Mode: models.ModeHybrid}, Constraints{NoFlash: true}); err == nil || !strings.Contains(tr.Error, "constraint violated") {
		t.Fatalf("a no-flash caller received a live answer: %v %+v", err, tr)
	}
}
