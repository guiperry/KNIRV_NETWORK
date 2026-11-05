package transformer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"knirvmodels/pkg/hands"
	gpt "knirvmodels/pkg/hands/gpt"
	sdk "github.com/guiperry/knirv-sdk-go/models"
)

// newTestGenerateService builds a HEARTService backed by a HANDS engine. When
// modelReady is true the engine reports as trained so the generation path is
// reachable; generation itself is stubbed via generateTextFn so the test exercises
// the handler's decision logic without loading the full model or the SDK core.
func newTestGenerateService(t *testing.T, modelReady bool) *HEARTService {
	t.Helper()
	cfg := &gpt.Config{
		VocabSize:  100,
		EmbedDim:   16,
		NumHeads:   2,
		NumLayers:  1,
		ContextLen: 8,
	}
	engine := hands.NewEngine(cfg, "")
	if modelReady {
		engine.Model().SetTrained(true)
		engine.Model().SetProgress(1.0)
	}
	hs := &HEARTService{engine: engine}
	hs.generateTextFn = func(prompt string, n int, temperature float32, topK int) (string, error) {
		return "generated:" + prompt, nil
	}
	return hs
}

func strptr(s string) *string { return &s }

// TestHeartGenerateSelectedDecision verifies that when a selected (non-abstaining)
// decision envelope is supplied, generation proceeds and the decision metadata is
// echoed back in the response.
func TestHeartGenerateSelectedDecision(t *testing.T) {
	hs := newTestGenerateService(t, true)

	reqBody, _ := json.Marshal(HeartGenerateRequest{
		Prompt: "fix the crash",
		Decision: &DecisionEnvelope{
			Decisions: []sdk.Decision{
				{Name: "action", Selected: strptr("restart-service"), Abstain: false},
			},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/heart/generate", bytes.NewReader(reqBody))
	w := httptest.NewRecorder()
	hs.handleGenerate(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var resp HeartGenerateResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Abstained {
		t.Fatalf("expected generation to proceed, got abstained response: %+v", resp)
	}
	if !strings.Contains(resp.Text, "fix the crash") {
		t.Fatalf("generated text missing prompt frame, got %q", resp.Text)
	}
	if resp.Decision == nil || len(resp.Decision.Decisions) == 0 {
		t.Fatalf("response missing decision metadata: %+v", resp)
	}
	got := resp.Decision.Decisions[0]
	if got.Selected == nil || *got.Selected != "restart-service" {
		t.Fatalf("decision selection not preserved: %+v", got)
	}
}

// TestHeartGenerateAbstainingDecision verifies that an abstaining decision
// short-circuits generation: no text is produced and the response reflects the
// abstention.
func TestHeartGenerateAbstainingDecision(t *testing.T) {
	hs := newTestGenerateService(t, true)

	reqBody, _ := json.Marshal(HeartGenerateRequest{
		Prompt: "fix the crash",
		Decision: &DecisionEnvelope{
			Abstained: true,
			Decisions: []sdk.Decision{{Name: "action", Abstain: true}},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/heart/generate", bytes.NewReader(reqBody))
	w := httptest.NewRecorder()
	hs.handleGenerate(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var resp HeartGenerateResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !resp.Abstained {
		t.Fatalf("expected abstention short-circuit, got text=%q", resp.Text)
	}
	if resp.Text != "" {
		t.Fatalf("expected no generated text on abstention, got %q", resp.Text)
	}
	if resp.Decision == nil || !resp.Decision.Abstained {
		t.Fatalf("response decision not marked abstained: %+v", resp.Decision)
	}
}

// TestHeartGeneratePlainIgnoresUnavailableCore pins the migration regression: a
// plain {"prompt"} request (what SDK HeartGenerate and the harness send) must NOT
// consult the decision core, so an unavailable core must not 503/abstain it. Only
// WithDecision opts into the core.
func TestHeartGeneratePlainIgnoresUnavailableCore(t *testing.T) {
	hs := newTestGenerateService(t, true)
	hs.decisionReasonFn = func(ctx context.Context, tokens []uint32) (json.RawMessage, error) {
		return nil, errors.New("core down")
	}

	// Plain request: no WithDecision, no Decision envelope.
	plain, _ := json.Marshal(HeartGenerateRequest{Prompt: "hi"})
	pr := httptest.NewRequest(http.MethodPost, "/heart/generate", bytes.NewReader(plain))
	pw := httptest.NewRecorder()
	hs.handleGenerate(pw, pr)
	if pw.Code != http.StatusOK {
		t.Fatalf("plain generate: status=%d, want 200 (core must be ignored); body=%s", pw.Code, pw.Body.String())
	}
	var presp HeartGenerateResponse
	if err := json.Unmarshal(pw.Body.Bytes(), &presp); err != nil {
		t.Fatalf("decode plain response: %v", err)
	}
	if presp.Abstained || presp.Error != "" {
		t.Fatalf("plain generate must not abstain or error on unavailable core: %+v", presp)
	}
	if !strings.Contains(presp.Text, "hi") {
		t.Fatalf("plain generate lost prompt frame: %q", presp.Text)
	}

	// WithDecision=true: the core is consulted and is unavailable -> 503.
	withDec, _ := json.Marshal(HeartGenerateRequest{Prompt: "hi", WithDecision: true})
	wr := httptest.NewRequest(http.MethodPost, "/heart/generate", bytes.NewReader(withDec))
	ww := httptest.NewRecorder()
	hs.handleGenerate(ww, wr)
	if ww.Code != http.StatusServiceUnavailable {
		t.Fatalf("with-decision generate: status=%d, want 503; body=%s", ww.Code, ww.Body.String())
	}
	if !strings.Contains(ww.Body.String(), "decision core unavailable") {
		t.Fatalf("with-decision generate should report unavailable core: %s", ww.Body.String())
	}
}

// TestHeartGenerateWithDecisionCoreAbstains pins that WithDecision consults the
// core and an abstaining core short-circuits generation.
func TestHeartGenerateWithDecisionCoreAbstains(t *testing.T) {
	hs := newTestGenerateService(t, true)
	hs.decisionReasonFn = func(ctx context.Context, tokens []uint32) (json.RawMessage, error) {
		return json.RawMessage(`{"decisions":[{"name":"action","abstain":true}]}`), nil
	}

	body, _ := json.Marshal(HeartGenerateRequest{Prompt: "do it", WithDecision: true})
	req := httptest.NewRequest(http.MethodPost, "/heart/generate", bytes.NewReader(body))
	w := httptest.NewRecorder()
	hs.handleGenerate(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200; body=%s", w.Code, w.Body.String())
	}
	var resp HeartGenerateResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !resp.Abstained {
		t.Fatalf("expected abstention from core, got text=%q", resp.Text)
	}
	if resp.Text != "" {
		t.Fatalf("expected no text on abstention, got %q", resp.Text)
	}
}

// TestHeartGenerateWithDecisionCoreSelects pins that WithDecision consults the
// core and a selecting core proceeds to generation with the decision echoed.
func TestHeartGenerateWithDecisionCoreSelects(t *testing.T) {
	hs := newTestGenerateService(t, true)
	hs.decisionReasonFn = func(ctx context.Context, tokens []uint32) (json.RawMessage, error) {
		return json.RawMessage(`{"decisions":[{"name":"action","selected":"restart"}]}`), nil
	}

	body, _ := json.Marshal(HeartGenerateRequest{Prompt: "do it", WithDecision: true})
	req := httptest.NewRequest(http.MethodPost, "/heart/generate", bytes.NewReader(body))
	w := httptest.NewRecorder()
	hs.handleGenerate(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200; body=%s", w.Code, w.Body.String())
	}
	var resp HeartGenerateResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Abstained {
		t.Fatalf("expected generation to proceed, got abstained")
	}
	if resp.Decision == nil || len(resp.Decision.Decisions) == 0 || resp.Decision.Decisions[0].Selected == nil || *resp.Decision.Decisions[0].Selected != "restart" {
		t.Fatalf("core selection not echoed: %+v", resp.Decision)
	}
	if !strings.Contains(resp.Text, "do it") {
		t.Fatalf("generated text missing prompt frame: %q", resp.Text)
	}
}
