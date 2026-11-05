package models

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"knirvmodels/internal/ops"
	"knirvmodels/internal/policy"
	handspkg "knirvmodels/pkg/hands"
	"knirvmodels/pkg/hands/gpt"
	"knirvmodels/pkg/heartqwen"

	"github.com/guiperry/knirv-sdk-go/models/frame"
)

// scriptedGen returns a canned sequence of model outputs (action envelopes), one
// per loop turn, and records the prompts it was asked to complete so tests can
// assert tool results are fed back into the context.
type scriptedGen struct {
	outputs []string
	prompts []string
	i       int
	ready   bool
}

func (g *scriptedGen) GenerateText(prompt string, n int, temperature float32, topK int) (string, error) {
	g.prompts = append(g.prompts, prompt)
	if g.i >= len(g.outputs) {
		return `{"action":"final","output":"done"}`, nil
	}
	s := g.outputs[g.i]
	g.i++
	return s, nil
}
func (g *scriptedGen) IsReady() bool     { return g.ready }
func (g *scriptedGen) Path() string      { return "test" }
func (g *scriptedGen) Progress() float64 { return 1 }

// recordingExec records the tools it was asked to run and the tenant each ran for.
type recordingExec struct {
	calls   []string
	tenants []string
	results map[string][]byte
}

func (x *recordingExec) Exec(ctx context.Context, tool string, args json.RawMessage) ([]byte, error) {
	x.calls = append(x.calls, tool)
	x.tenants = append(x.tenants, heartqwen.TenantFrom(ctx))
	if r, ok := x.results[tool]; ok {
		return r, nil
	}
	return []byte(`{"result":"` + tool + `"}`), nil
}

func postHands(t *testing.T, url, body string, headers map[string]string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

// T0: a real core-authorized read tool (manifest.get, authorized for the
// hands.assistant stage) is executed by the loop through the real policy gate.
func TestHandsPlanExecutesT0Tool(t *testing.T) {
	t.Setenv("KNIRV_HANDS_ENABLED", "1")
	reg := frame.Default()
	ex := &recordingExec{results: map[string][]byte{"manifest.get": []byte(`{"ok":true}`)}}
	gen := &scriptedGen{ready: true, outputs: []string{
		`{"action":"tool","tool":"manifest.get","args":{}}`,
		`{"action":"final","output":"done"}`,
	}}
	h := NewHandsWith(func() (generator, error) { return gen, nil }, policy.NewGate(reg, nil), nil, ex, reg)
	mux := http.NewServeMux()
	h.Mount(mux, "/models/hands")
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp := postHands(t, srv.URL+"/models/hands/act", `{"prompt":"read a fact"}`, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var got struct {
		Trace     []heartqwen.Step `json:"trace"`
		Proposals []heartqwen.Step `json:"proposals"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(ex.calls) != 1 || ex.calls[0] != "manifest.get" {
		t.Fatalf("expected manifest.get executed once, got %v", ex.calls)
	}
	if len(got.Trace) != 1 || got.Trace[0].Tool != "manifest.get" || got.Trace[0].Verdict != "execute" {
		t.Fatalf("trace missing execute step: %+v", got.Trace)
	}
	if len(got.Proposals) != 0 {
		t.Fatalf("no proposals expected, got %+v", got.Proposals)
	}
}

// T2 (provisional): a real T2 tool the hands.assistant stage does NOT authorize is
// never executed by the loop. NOTE: production HANDS does not yet authorize any T2
// tool for the hands stage, so T2 proposal/execution is provisional and pending an
// owner contract; this test pins the conservative behavior (not executed, surfaced
// as an unauthorized/denied call) rather than claiming T2 execution works.
func TestHandsPlanT2ToolNotExecuted(t *testing.T) {
	t.Setenv("KNIRV_HANDS_ENABLED", "1")
	reg := frame.Default()
	ex := &recordingExec{}
	gen := &scriptedGen{ready: true, outputs: []string{
		`{"action":"tool","tool":"head.vision","args":{}}`,
		`{"action":"final","output":"done"}`,
	}}
	h := NewHandsWith(func() (generator, error) { return gen, nil }, policy.NewGate(reg, nil), nil, ex, reg)
	mux := http.NewServeMux()
	h.Mount(mux, "/models/hands")
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp := postHands(t, srv.URL+"/models/hands/act", `{"prompt":"look at an image"}`, map[string]string{"X-KNIRV-Privilege": "operator"})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var got struct {
		Trace     []heartqwen.Step `json:"trace"`
		Proposals []heartqwen.Step `json:"proposals"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(ex.calls) != 0 {
		t.Fatalf("T2 must not be executed, got %v", ex.calls)
	}
	if len(got.Proposals) != 0 {
		t.Fatalf("T2 is provisional/pending owner contract; expected no proposal, got %+v", got.Proposals)
	}
	if len(got.Trace) != 1 || got.Trace[0].Verdict != "deny" {
		t.Fatalf("expected deny step for unauthorized T2 tool, got %+v", got.Trace)
	}
}

// T3/forbidden: a tool the stage does not allow (a real production tool not
// authorized for hands.assistant) is denied by the gate and the denial is written
// to the audit chain.
func TestHandsPlanDeniesUnauthorizedToolAndAudits(t *testing.T) {
	t.Setenv("KNIRV_HANDS_ENABLED", "1")
	reg := frame.Default()
	auditPath := t.TempDir() + "/tool_audit.jsonl"
	audit, err := policy.OpenAudit(auditPath)
	if err != nil {
		t.Fatal(err)
	}
	ex := &recordingExec{}
	gen := &scriptedGen{ready: true, outputs: []string{
		`{"action":"tool","tool":"heart.decide","args":{}}`,
		`{"action":"final","output":"done"}`,
	}}
	h := NewHandsWith(func() (generator, error) { return gen, nil }, policy.NewGate(reg, audit), audit, ex, reg)
	mux := http.NewServeMux()
	h.Mount(mux, "/models/hands")
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp := postHands(t, srv.URL+"/models/hands/act", `{"prompt":"hack"}`, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var got struct {
		Trace []heartqwen.Step `json:"trace"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(ex.calls) != 0 {
		t.Fatalf("forbidden tool must not execute, got %v", ex.calls)
	}
	if len(got.Trace) != 1 || got.Trace[0].Verdict != "deny" {
		t.Fatalf("expected deny step, got %+v", got.Trace)
	}
	data, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"outcome":"deny"`) {
		t.Fatalf("audit missing deny entry:\n%s", data)
	}
}

// Injected tool-result data: the loop feeds the executed tool's result back into
// the prompt as delimited data on the following turn.
func TestHandsPlanFeedsToolResultBack(t *testing.T) {
	t.Setenv("KNIRV_HANDS_ENABLED", "1")
	reg := frame.Default()
	ex := &recordingExec{results: map[string][]byte{"manifest.get": []byte("FACT: disk 80% full")}}
	gen := &scriptedGen{ready: true, outputs: []string{
		`{"action":"tool","tool":"manifest.get","args":{}}`,
		`{"action":"final","output":"done"}`,
	}}
	h := NewHandsWith(func() (generator, error) { return gen, nil }, policy.NewGate(reg, nil), nil, ex, reg)
	mux := http.NewServeMux()
	h.Mount(mux, "/models/hands")
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp := postHands(t, srv.URL+"/models/hands/act", `{"prompt":"status?"}`, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if len(gen.prompts) < 2 {
		t.Fatalf("expected >=2 model turns, got %d", len(gen.prompts))
	}
	if !strings.Contains(gen.prompts[1], "TOOL_RESULT") || !strings.Contains(gen.prompts[1], "disk 80% full") {
		t.Fatalf("tool result not fed back: %q", gen.prompts[1])
	}
}

// Tenant propagation: the caller's tenant reaches the tool executor unchanged.
func TestHandsPlanPropagatesTenant(t *testing.T) {
	t.Setenv("KNIRV_HANDS_ENABLED", "1")
	reg := frame.Default()
	ex := &recordingExec{results: map[string][]byte{"manifest.get": []byte("ok")}}
	gen := &scriptedGen{ready: true, outputs: []string{
		`{"action":"tool","tool":"manifest.get","args":{}}`,
		`{"action":"final","output":"done"}`,
	}}
	h := NewHandsWith(func() (generator, error) { return gen, nil }, policy.NewGate(reg, nil), nil, ex, reg)
	mux := http.NewServeMux()
	h.Mount(mux, "/models/hands")
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp := postHands(t, srv.URL+"/models/hands/act", `{"prompt":"tenant scoped"}`, map[string]string{"X-KNIRV-Tenant": "acme"})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if len(ex.tenants) != 1 || ex.tenants[0] != "acme" {
		t.Fatalf("tenant not propagated, got %v", ex.tenants)
	}
}

// Fix 1 (audit): a real core-authorized T0 tool is executed by the HANDS loop
// through the concrete Standard executor, and every decision is written to the
// audit chain with a verifiable entry.
func TestHandsExecutesRealCoreToolAndAudits(t *testing.T) {
	t.Setenv("KNIRV_HANDS_ENABLED", "1")
	reg := frame.Default()
	auditPath := t.TempDir() + "/hands_tool_audit.jsonl"
	audit, err := policy.OpenAudit(auditPath)
	if err != nil {
		t.Fatal(err)
	}
	tools := ops.Standard(ops.Deps{
		Manifest: func() any { return map[string]any{"constitution_version": "test"} },
	})
	gate := policy.NewGate(reg, audit)
	gen := &scriptedGen{ready: true, outputs: []string{
		`{"action":"tool","tool":"manifest.get","args":{}}`,
		`{"action":"final","output":"done"}`,
	}}
	h := NewHandsWith(func() (generator, error) { return gen, nil }, gate, audit, tools, reg)
	mux := http.NewServeMux()
	h.Mount(mux, "/models/hands")
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp := postHands(t, srv.URL+"/models/hands/act", `{"prompt":"show manifest"}`, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var got struct {
		Trace []heartqwen.Step `json:"trace"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.Trace) != 1 || got.Trace[0].Tool != "manifest.get" || got.Trace[0].Verdict != "execute" {
		t.Fatalf("expected manifest.get executed, got %+v", got.Trace)
	}
	data, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"tool":"manifest.get"`) || !strings.Contains(string(data), `"outcome":"execute"`) {
		t.Fatalf("audit missing execute entry for manifest.get:\n%s", data)
	}
}

// Fix 2 regression: an oversized N is rejected (400), not run.
func TestHandsGenerateRejectsOversizedN(t *testing.T) {
	t.Setenv("KNIRV_HANDS_ENABLED", "1")
	gen := &scriptedGen{}
	h := NewHandsWith(func() (generator, error) { return gen, nil }, policy.NewGate(frame.Default(), nil), nil, &recordingExec{}, frame.Default())
	mux := http.NewServeMux()
	h.Mount(mux, "/models/hands")
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp := postHands(t, srv.URL+"/models/hands/generate", `{"prompt":"x","n":99999}`, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for oversized N", resp.StatusCode)
	}
	// A bounded N is accepted (reaching the untrained gate -> 503).
	resp2 := postHands(t, srv.URL+"/models/hands/generate", `{"prompt":"x","n":32}`, nil)
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503 for untrained generate", resp2.StatusCode)
	}
}

// Fix 2 regression: an oversized request body is rejected (400).
func TestHandsPlanRejectsOversizedBody(t *testing.T) {
	t.Setenv("KNIRV_HANDS_ENABLED", "1")
	h := NewHandsWith(func() (generator, error) { return &scriptedGen{}, nil }, policy.NewGate(frame.Default(), nil), nil, &recordingExec{}, frame.Default())
	mux := http.NewServeMux()
	h.Mount(mux, "/models/hands")
	srv := httptest.NewServer(mux)
	defer srv.Close()

	big := `{"prompt":"` + strings.Repeat("x", 200000) + `"}`
	resp := postHands(t, srv.URL+"/models/hands/plan", big, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for oversized body", resp.StatusCode)
	}
}

// Fix 2: with no checkpoint in HandsDir, generate stays 503 (untrained, no training).
func TestHandsGenerateNoCheckpointReturns503(t *testing.T) {
	t.Setenv("KNIRV_HANDS_ENABLED", "1")
	dir := t.TempDir()
	t.Setenv("KNIRV_HANDS_DIR", dir) // empty: no model-latest
	cfg := &gpt.Config{
		VocabSize: 16, EmbedDim: 8, NumHeads: 2, NumLayers: 1,
		ContextLen: 16, FFNHiddenDim: 16, DecayAlpha: 1.0,
	}
	h := NewHandsWith(
		func() (generator, error) { return handspkg.NewEngine(cfg, ""), nil },
		policy.NewGate(frame.Default(), nil), nil, &recordingExec{}, frame.Default(),
	)
	mux := http.NewServeMux()
	h.Mount(mux, "/models/hands")
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp := postHands(t, srv.URL+"/models/hands/generate", `{"prompt":"hi","n":8}`, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503 (no checkpoint)", resp.StatusCode)
	}
}

// Fix 2: with an existing checkpoint in HandsDir/model-latest, generate loads it
// and returns 200 (ready). Uses a tiny config so the save/load is instant.
func TestHandsGenerateLoadsCheckpoint(t *testing.T) {
	t.Setenv("KNIRV_HANDS_ENABLED", "1")
	dir := t.TempDir()
	t.Setenv("KNIRV_HANDS_DIR", dir)
	cp := filepath.Join(dir, "model-latest")
	cfg := &gpt.Config{
		VocabSize: 16, EmbedDim: 8, NumHeads: 2, NumLayers: 1,
		ContextLen: 16, FFNHiddenDim: 16, DecayAlpha: 1.0,
	}
	saved := handspkg.NewEngine(cfg, "")
	if err := saved.Save(cp); err != nil {
		t.Fatal(err)
	}
	h := NewHandsWith(
		func() (generator, error) {
			eng := handspkg.NewEngine(cfg, "")
			if err := eng.Load(cp); err != nil {
				return nil, err
			}
			return eng, nil
		},
		policy.NewGate(frame.Default(), nil), nil, &recordingExec{}, frame.Default(),
	)
	mux := http.NewServeMux()
	h.Mount(mux, "/models/hands")
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp := postHands(t, srv.URL+"/models/hands/generate", `{"prompt":"hi","n":8}`, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (checkpoint loaded)", resp.StatusCode)
	}
}
