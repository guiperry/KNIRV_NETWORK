package tracer

import (
	"testing"

	"backend_server/internal/services/plugins/fintech/ontology"
)

func TestStartTrace(t *testing.T) {
	registry := ontology.NewOntologyRegistry()
	nrv := NewNRVTracer(registry)

	trace := nrv.StartTrace("agent-1", "Test Agent", "val-1")

	if trace == nil {
		t.Fatal("expected non-nil trace from StartTrace")
	}
	if trace.AgentID != "agent-1" {
		t.Errorf("expected AgentID 'agent-1', got %q", trace.AgentID)
	}
	if trace.AgentName != "Test Agent" {
		t.Errorf("expected AgentName 'Test Agent', got %q", trace.AgentName)
	}
	if trace.ValidationID != "val-1" {
		t.Errorf("expected ValidationID 'val-1', got %q", trace.ValidationID)
	}
	if trace.Status != NRVTraceStatusCapturing {
		t.Errorf("expected status %q, got %q", NRVTraceStatusCapturing, trace.Status)
	}
}

func TestRecordInference(t *testing.T) {
	registry := ontology.NewOntologyRegistry()
	nrv := NewNRVTracer(registry)

	trace := nrv.StartTrace("agent-2", "Agent 2", "val-2")

	err := nrv.RecordInference(trace.ID, "buy_intent", "market_observation", "execute_buy", 0.92, nil)
	if err != nil {
		t.Fatalf("RecordInference() failed: %v", err)
	}

	// Verify step was recorded
	updated, err := nrv.GetTrace(trace.ID)
	if err != nil {
		t.Fatalf("GetTrace() failed: %v", err)
	}
	if updated.StepCount == 0 && len(updated.Steps) == 0 {
		t.Error("expected at least one step after RecordInference")
	}
}

func TestFinalizeTrace(t *testing.T) {
	registry := ontology.NewOntologyRegistry()
	nrv := NewNRVTracer(registry)

	trace := nrv.StartTrace("agent-3", "Agent 3", "val-3")
	_ = nrv.RecordInference(trace.ID, "intent", "obs", "inf", 0.8, nil)
	_ = nrv.RecordDecision(trace.ID, "approved", 0.85, map[string]interface{}{"reason": "test"})

	finalized, err := nrv.FinalizeTrace(trace.ID)
	if err != nil {
		t.Fatalf("FinalizeTrace() failed: %v", err)
	}
	if finalized.Status != NRVTraceStatusComplete {
		t.Errorf("expected status %q after finalization, got %q", NRVTraceStatusComplete, finalized.Status)
	}
}
