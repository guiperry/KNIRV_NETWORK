package inferencer

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestInferenceServiceRoutesThroughHarness(t *testing.T) {
	svc, _, _ := newLoadTestService(0, 0)
	svc.dreamAttempts = append(svc.dreamAttempts, LLMAttempt{
		Instance: &stubLLM{response: "dream-stub"},
		Config:   LLMAttemptConfig{ProviderName: "llama", ModelName: "local-llama", MaxTokens: 64, IsPrimary: true},
	})
	heart := up("heart")
	heart.do = func(Task) (Result, error) { return Result{Text: "heart text"}, nil }
	fl := &fakeLlama{}
	llama := fl.model(t, nil)
	llama.Base = svc.LlamaServerTask
	h := NewHarness([]HarnessModel{heart, llama, svc.ExternalModel()}, WithRoutes(UserFacingRoutes()))
	svc.SetHarness(h)
	if svc.Harness() != h {
		t.Fatal("harness not attached")
	}
	ctx := context.Background()

	// Generation: HEART when ready.
	got, err := svc.GenerateTextWithContext(ctx, "", "unique prompt one", "")
	if err != nil || got != "heart text" {
		t.Fatalf("%q %v", got, err)
	}
	// ... the external chain when it is not (and the trace says why).
	heart.ready = Readiness{Detail: "idle"}
	got, err = svc.GenerateTextWithContext(ctx, "", "unique prompt two", "")
	if err != nil || got != "primary-stub-response" {
		t.Fatalf("%q %v", got, err)
	}
	if tr := h.Traces(1)[0]; tr.Chosen != "external" || tr.Candidates[0].Excluded == "" {
		t.Fatalf("%+v", tr)
	}
	// The user-facing table does not substitute the small llama for the provider chain.
	for _, c := range h.Traces(1)[0].Candidates {
		if c.Model == "llama" {
			t.Fatal("llama must not be a Generate candidate in the user-facing table")
		}
	}

	// A named model bypasses the harness exactly as before.
	if got, err = svc.GenerateTextWithContext(ctx, "local-llama", "unique prompt three", ""); err != nil || got != "primary-stub-response" {
		t.Fatalf("%q %v", got, err)
	}

	// Dream passes: the llama provider via the harness, unchanged provider behaviour.
	before := len(h.Traces(0))
	got, err = svc.GenerateTextWithProvider("llama", "dream")
	if err != nil || got != "dream-stub" {
		t.Fatalf("%q %v", got, err)
	}
	if tr := h.Traces(1)[0]; len(h.Traces(0)) != before+1 || tr.Kind != KindServerTask || tr.Chosen != "llama" {
		t.Fatalf("%+v", tr)
	}
	// Other providers are not the harness's business.
	if _, err = svc.GenerateTextWithProvider("nope", "x"); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("%v", err)
	}

	// Planning reaches the orchestrator through the external model (none configured here).
	if _, err = svc.ExecuteComplexTask(ctx, "plan"); err == nil || !errors.Is(err, ErrNoRoute) || !strings.Contains(err.Error(), "orchestrator") {
		t.Fatalf("%v", err)
	}
	if tr := h.Traces(1)[0]; tr.Kind != KindPlan || len(tr.Fallbacks) != 1 {
		t.Fatalf("%+v", tr)
	}
}

func TestInferenceServiceWithoutHarnessIsUnchanged(t *testing.T) {
	svc, _, _ := newLoadTestService(0, 0)
	got, err := svc.GenerateTextWithContext(context.Background(), "", "plain prompt", "")
	if err != nil || got != "primary-stub-response" {
		t.Fatalf("%q %v", got, err)
	}
	if svc.Harness() != nil {
		t.Fatal("no harness unless one is set")
	}
}
