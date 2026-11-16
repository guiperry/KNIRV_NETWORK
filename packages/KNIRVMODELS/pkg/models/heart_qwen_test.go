package models

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"

	"github.com/guiperry/knirv-sdk-go/models/frame"

	"knirvmodels/internal/ops"
	"knirvmodels/internal/policy"
	"knirvmodels/pkg/hashing/transformer"
	"knirvmodels/pkg/heartqwen"
)

type fakeLLM struct {
	replies []string
	err     error
}

func (f *fakeLLM) Complete(context.Context, heartqwen.Request) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	if len(f.replies) == 0 {
		// Out of script: fail the call instead of panicking inside the HTTP handler.
		// The caller then reports its own cause (e.g. a failed tinygo compile).
		return "", errors.New("fakeLLM: no scripted reply left")
	}
	r := f.replies[0]
	f.replies = f.replies[1:]
	return r, nil
}

func qwenHeart(llm heartqwen.LLM, live bool) *httptest.Server {
	reg := frame.Default()
	h := NewHeart(nil)
	h.UseQwen(&heartqwen.Engine{Reg: reg, Facts: func() frame.LiveFacts { return frame.LiveFacts{Model: "q"} },
		LLM: llm, Gate: policy.NewGate(reg, nil), Exec: ops.New()},
		func(context.Context) bool { return live }, nil)
	mux := http.NewServeMux()
	h.Mount(mux, "/models/heart")
	return httptest.NewServer(mux)
}

func gen(t *testing.T, base, body string, hdr map[string]string) (int, map[string]any) {
	t.Helper()
	req, _ := http.NewRequest("POST", base+"/models/heart/generate", strings.NewReader(body))
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var m map[string]any
	_ = json.NewDecoder(res.Body).Decode(&m)
	return res.StatusCode, m
}

func TestHeartGenerateIsQwen(t *testing.T) {
	s := qwenHeart(&fakeLLM{replies: []string{`{"action":"final","output":"hello from qwen"}`}}, true)
	defer s.Close()
	code, m := gen(t, s.URL, `{"prompt":"hi"}`, nil)
	if code != 200 || m["text"] != "hello from qwen" || m["backend"] != "qwen" || m["stage"] != "heart.chat" || m["frame_hash"] == "" {
		t.Fatalf("%d %v", code, m)
	}
}

func TestHeartGenerateFailureIs503SoTheHarnessFallsBack(t *testing.T) {
	s := qwenHeart(&fakeLLM{err: errors.New("llama down")}, true)
	defer s.Close()
	code, m := gen(t, s.URL, `{"prompt":"hi"}`, nil)
	if code != http.StatusServiceUnavailable || m["ready"] != false {
		t.Fatalf("%d %v", code, m)
	}
}

func TestHandsAssistantNeedsAnOperator(t *testing.T) {
	s := qwenHeart(&fakeLLM{replies: []string{`{"action":"final","output":"ok"}`, `{"action":"final","output":"ok"}`}}, true)
	defer s.Close()
	if code, _ := gen(t, s.URL, `{"prompt":"x","stage":"hands.assistant"}`, nil); code != http.StatusForbidden {
		t.Fatalf("user reached hands.assistant: %d", code)
	}
	if code, _ := gen(t, s.URL, `{"prompt":"x","stage":"hands.assistant"}`, map[string]string{PrivilegeHeader: "operator"}); code != 200 {
		t.Fatalf("operator refused: %d", code)
	}
}

func TestHeartHealthReportsQwen(t *testing.T) {
	for _, live := range []bool{true, false} {
		s := qwenHeart(&fakeLLM{}, live)
		res, err := http.Get(s.URL + "/models/heart/health")
		if err != nil {
			t.Fatal(err)
		}
		var h struct {
			Available bool   `json:"available"`
			Status    string `json:"status"`
		}
		_ = json.NewDecoder(res.Body).Decode(&h)
		res.Body.Close()
		s.Close()
		if h.Available != live || (live && h.Status != "qwen") {
			t.Fatalf("live=%v got %+v", live, h)
		}
	}
}

func finisher(t *testing.T) *transformer.WASMFinisher {
	t.Helper()
	path, err := exec.LookPath("tinygo")
	if err != nil {
		t.Skip("tinygo not installed")
	}
	return &transformer.WASMFinisher{Compiler: transformer.NewWASMCompiler(path, t.TempDir()), Verifier: transformer.NewBidirectionalVerifier()}
}

// scriptedFinisher stands in for the render/compile/verify back end. It applies
// the one rule the test needs (a severity above 9 is rejected, with a reason) and
// verifies anything else, so no TinyGo or wazero run happens.
type scriptedFinisher struct{ calls int }

func (f *scriptedFinisher) Finish(_ context.Context, typ transformer.WASMType, raw json.RawMessage, _ string) (transformer.WASMDecision, error) {
	f.calls++
	d := transformer.WASMDecision{WASMType: typ, TurnCount: 1}
	if strings.Contains(string(raw), `"severity":999`) {
		d.BackwardVerifierMsg = "severity 999 is out of range 0..9"
		return d, nil
	}
	d.BidirectionalVerified, d.WazeroExecPassed = true, true
	d.WASMHash = "scripted-wasm-hash"
	return d, nil
}

func qwenHeartWithFinisher(llm heartqwen.LLM, fin wasmFinisher) *httptest.Server {
	reg := frame.Default()
	h := NewHeart(nil)
	h.UseQwen(&heartqwen.Engine{Reg: reg, Facts: func() frame.LiveFacts { return frame.LiveFacts{Model: "q"} },
		LLM: llm, Gate: policy.NewGate(reg, nil), Exec: ops.New()},
		func(context.Context) bool { return true }, fin)
	mux := http.NewServeMux()
	h.Mount(mux, "/models/heart")
	return httptest.NewServer(mux)
}

// fakeRegistry knows one stage, the resolution stage the test calls, and builds a
// frame locally. It needs no SDK core, so the test runs with no seeded
// KNIRV_SDK_DATA_DIR. The stage has no tools, so the gate is never consulted.
type fakeRegistry struct{}

func (fakeRegistry) Stage(id string) (frame.Stage, bool) {
	if id == wasmStage[transformer.WASMTypeResolution] {
		// Shape copied from the core's heart.wasm.resolution stage: tools make the
		// engine use the action envelope, and the schema makes the output structured.
		// Without them the finisher would be handed the wrong bytes.
		schema := json.RawMessage(`{"type":"object","required":["error_code","hint","severity","retryable"]}`)
		return frame.Stage{ID: id, Role: "test", Objective: "resolve", OutputSchema: schema,
			Tools: []string{"wasm.render", "wasm.verify", "rules.list"}, MaxOutput: 256, MaxToolCalls: 4}, true
	}
	return frame.Stage{}, false
}

func (fakeRegistry) Build(stage string, _ frame.LiveFacts) (frame.Frame, error) {
	sys := "fake frame for " + stage
	sum := sha256.Sum256([]byte(sys))
	return frame.Frame{Stage: stage, Version: frame.ConstitutionVersion, System: sys, Hash: hex.EncodeToString(sum[:])}, nil
}

// qwenHeartFake is qwenHeartWithFinisher with the registry injected, so no
// frame.Default() (and no SDK core) is touched. An empty frame.Registry in the gate
// denies every call, so a tool request could not run even if one appeared.
func qwenHeartFake(llm heartqwen.LLM, fin wasmFinisher) *httptest.Server {
	h := NewHeart(nil)
	h.UseQwen(&heartqwen.Engine{Reg: fakeRegistry{}, Facts: func() frame.LiveFacts { return frame.LiveFacts{Model: "q"} },
		LLM: llm, Gate: policy.NewGate(&frame.Registry{}, nil), Exec: ops.New()},
		func(context.Context) bool { return true }, fin)
	mux := http.NewServeMux()
	h.Mount(mux, "/models/heart")
	return httptest.NewServer(mux)
}

func call(t *testing.T, base, route, body string) (int, map[string]any) {
	t.Helper()
	res, err := http.Post(base+"/models/heart/"+route, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var m map[string]any
	_ = json.NewDecoder(res.Body).Decode(&m)
	return res.StatusCode, m
}

// The model chooses fields; code renders, verifies in wazero and compiles. A bad
// choice goes back to the model with the reason, and the second try succeeds.
func TestResolveOverQwenRetriesARejectedChoiceThenVerifies(t *testing.T) {
	llm := &fakeLLM{replies: []string{
		`{"action":"final","output":{"error_code":1,"hint":"x","severity":999,"retryable":true}}`, // out of range
		`{"action":"final","output":{"error_code":42,"hint":"retry with backoff","severity":2,"retryable":true}}`,
	}}
	fin := &scriptedFinisher{}
	s := qwenHeartFake(llm, fin)
	defer s.Close()
	code, m := call(t, s.URL, "resolve", `{"dve_session_id":"s1","error_type":"timeout","error_message":"upstream timed out"}`)
	if fin.calls != 2 {
		t.Fatalf("finisher calls = %d, want 2 (one rejected, one verified)", fin.calls)
	}
	if code != 200 || m["bidirectional_verified"] != true || m["wasm_hash"] == "" || m["turn_count"] != 2.0 || m["backend"] != "qwen" {
		t.Fatalf("%d %v", code, m)
	}
	if at, _ := m["attempts"].([]any); len(at) != 1 || !strings.Contains(at[0].(string), "severity") {
		t.Fatalf("the rejection reason must be recorded: %v", m["attempts"])
	}
}

func TestResolveNeedsASessionAndNeverVerifiesGarbage(t *testing.T) {
	llm := &fakeLLM{replies: []string{
		`{"action":"final","output":{"error_code":1,"hint":"x","severity":999,"retryable":true}}`,
		`{"action":"final","output":{"error_code":1,"hint":"x","severity":999,"retryable":true}}`,
		`{"action":"final","output":{"error_code":1,"hint":"x","severity":999,"retryable":true}}`,
	}}
	s := qwenHeartWithFinisher(llm, finisher(t))
	defer s.Close()
	if code, _ := call(t, s.URL, "resolve", `{"error_type":"t"}`); code != http.StatusUnprocessableEntity {
		t.Fatalf("no session: %d", code)
	}
	code, m := call(t, s.URL, "resolve", `{"dve_session_id":"s","error_type":"t"}`)
	if code != 200 || m["bidirectional_verified"] == true || m["turn_count"] != 3.0 {
		t.Fatalf("three bad choices must end unverified after 3 turns: %d %v", code, m)
	}
}

func TestAdviseAndPatchOverQwen(t *testing.T) {
	llm := &fakeLLM{replies: []string{
		`{"action":"final","output":{"policy_id":7,"severity":3,"action":1,"confidence":0.8}}`,
		`{"action":"final","output":{"target_component":"gateway","patch_data":"set limit=10","apply_priority":5}}`,
	}}
	s := qwenHeartWithFinisher(llm, finisher(t))
	defer s.Close()
	if code, m := call(t, s.URL, "advise", `{"name":"n","badge_type":"policy"}`); code != 200 || m["bidirectional_verified"] != true || m["wasm_type"] != "rule" {
		t.Fatalf("%d %v", code, m)
	}
	if code, m := call(t, s.URL, "patch", `{"component_id":"gateway","error_code":"E1","system_state":"degraded"}`); code != 200 || m["bidirectional_verified"] != true || m["wasm_type"] != "patch" {
		t.Fatalf("%d %v", code, m)
	}
}

func TestAnalyzeOverQwenKeepsTheLegacyShape(t *testing.T) {
	llm := &fakeLLM{replies: []string{`{"action":"final","output":{"analysis":"upstream timeout","confidence":1.7,"recommended_actions":["retry"]}}`}}
	s := qwenHeartWithFinisher(llm, nil)
	defer s.Close()
	code, m := call(t, s.URL, "analyze", `{"error_id":"e1","error_type":"timeout"}`)
	if code != 200 || m["inquiry_id"] != "e1" || m["analysis_summary"] != "upstream timeout" || m["confidence_score"] != 1.0 || m["backend"] != "qwen" {
		t.Fatalf("%d %v", code, m)
	}
}

func TestModuleRoutesFailOverWhenTheModelFails(t *testing.T) {
	s := qwenHeartWithFinisher(&fakeLLM{err: errors.New("llama down")}, finisher(t))
	defer s.Close()
	if code, _ := call(t, s.URL, "patch", `{"component_id":"c"}`); code != http.StatusServiceUnavailable {
		t.Fatalf("%d", code)
	}
}

// A typed-nil *WASMFinisher must not reach the interface as a live finisher:
// module routes answer 503, as they do with no finisher at all.
func TestModuleRoutesAnswer503ForTypedNilFinisher(t *testing.T) {
	llm := &fakeLLM{replies: []string{`{"action":"final","output":{"error_code":1,"hint":"x","severity":2,"retryable":true}}`}}
	var nilFin *transformer.WASMFinisher
	s := qwenHeartWithFinisher(llm, nilFin)
	defer s.Close()
	code, _ := call(t, s.URL, "resolve", `{"dve_session_id":"s1","error_type":"timeout"}`)
	if code != http.StatusServiceUnavailable {
		t.Fatalf("typed-nil finisher: status %d, want 503", code)
	}
}
