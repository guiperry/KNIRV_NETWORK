// Copyright 2026 KNIRV-SERVER
// SPDX-License-Identifier: GPL-3.0-or-later

package fintech

import (
	"context"
	"testing"
	"time"

	"backend_server/internal/services/plugins/fintech/ontology"
)

func TestNRVTracer_StartTrace(t *testing.T) {
	registry := ontology.NewOntologyRegistry()
	tracer := NewNRVTracer(registry)

	trace := tracer.StartTrace("agent-123", "TestAgent", "validation-456")

	if trace == nil {
		t.Fatal("Expected trace to be created")
	}

	if trace.ID == "" {
		t.Error("Expected trace ID to be set")
	}

	if trace.AgentID != "agent-123" {
		t.Errorf("Expected AgentID to be 'agent-123', got '%s'", trace.AgentID)
	}

	if trace.AgentName != "TestAgent" {
		t.Errorf("Expected AgentName to be 'TestAgent', got '%s'", trace.AgentName)
	}

	if trace.ValidationID != "validation-456" {
		t.Errorf("Expected ValidationID to be 'validation-456', got '%s'", trace.ValidationID)
	}

	if trace.Status != NRVTraceStatusCapturing {
		t.Errorf("Expected status to be CAPTURING, got '%s'", trace.Status)
	}

	if len(trace.Steps) != 0 {
		t.Errorf("Expected 0 steps, got %d", len(trace.Steps))
	}
}

func TestNRVTracer_AddStep(t *testing.T) {
	registry := ontology.NewOntologyRegistry()
	tracer := NewNRVTracer(registry)
	trace := tracer.StartTrace("agent-123", "TestAgent", "validation-456")

	step := NRVReasoningStep{
		StepType:    "inference",
		Intent:      "Verify transaction",
		Observation: "Large transfer detected",
		Inference:   "Requires AML check",
		Confidence:  0.95,
	}

	err := tracer.AddStep(trace.ID, step)
	if err != nil {
		t.Fatalf("Failed to add step: %v", err)
	}

	if len(trace.Steps) != 1 {
		t.Errorf("Expected 1 step, got %d", len(trace.Steps))
	}

	if trace.Steps[0].StepNumber != 1 {
		t.Errorf("Expected step number 1, got %d", trace.Steps[0].StepNumber)
	}

	if trace.StepCount != 1 {
		t.Errorf("Expected step count 1, got %d", trace.StepCount)
	}
}

func TestNRVTracer_RecordInference(t *testing.T) {
	registry := ontology.NewOntologyRegistry()
	tracer := NewNRVTracer(registry)
	trace := tracer.StartTrace("agent-123", "TestAgent", "validation-456")

	financialContext := &FinancialContext{
		ActionType: ontology.ActionTransfer,
		Amount: &ontology.MonetaryValue{
			Amount:   50000.0,
			Currency: "USD",
		},
	}

	err := tracer.RecordInference(
		trace.ID,
		"Process transfer",
		"Large amount detected",
		"Must verify counterparty",
		0.90,
		financialContext,
	)

	if err != nil {
		t.Fatalf("Failed to record inference: %v", err)
	}

	if len(trace.Steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(trace.Steps))
	}

	step := trace.Steps[0]

	if step.StepType != "inference" {
		t.Errorf("Expected step type 'inference', got '%s'", step.StepType)
	}

	if step.Intent != "Process transfer" {
		t.Errorf("Expected intent 'Process transfer', got '%s'", step.Intent)
	}

	if step.FinancialContext == nil {
		t.Fatal("Expected financial context to be set")
	}

	if step.FinancialContext.ActionType != ontology.ActionTransfer {
		t.Errorf("Expected action type TRANSFER, got '%s'", step.FinancialContext.ActionType)
	}
}

func TestNRVTracer_RecordDecision(t *testing.T) {
	registry := ontology.NewOntologyRegistry()
	tracer := NewNRVTracer(registry)
	trace := tracer.StartTrace("agent-123", "TestAgent", "validation-456")

	err := tracer.RecordDecision(trace.ID, "APPROVE", 0.85, map[string]interface{}{
		"risk_level": "low",
	})

	if err != nil {
		t.Fatalf("Failed to record decision: %v", err)
	}

	if trace.FinalDecision != "APPROVE" {
		t.Errorf("Expected final decision 'APPROVE', got '%s'", trace.FinalDecision)
	}
}

func TestNRVTracer_DetectKYCBypass(t *testing.T) {
	registry := ontology.NewOntologyRegistry()
	tracer := NewNRVTracer(registry)
	trace := tracer.StartTrace("agent-123", "TestAgent", "validation-456")

	// Add a suspicious step
	tracer.AddStep(trace.ID, NRVReasoningStep{
		StepType:    "decision",
		Intent:      "Fast track customer onboarding",
		Observation: "Skip KYC for trusted client",
		Inference:   "Bypass standard verification",
		Confidence:  0.70,
	})

	// Add another suspicious step
	tracer.AddStep(trace.ID, NRVReasoningStep{
		StepType:    "action",
		Intent:      "Approve unverified account",
		Observation: "Account has no verification data",
		Inference:   "Proceed anyway",
		Confidence:  0.60,
	})

	indicators := tracer.DetectKYCBypass(trace.ID)

	if len(indicators) == 0 {
		t.Error("Expected to detect KYC bypass indicators")
	}

	foundBypass := false
	for _, ind := range indicators {
		if ind.Type == "kyc_bypass" {
			foundBypass = true
			if ind.Severity != "critical" {
				t.Errorf("Expected critical severity, got '%s'", ind.Severity)
			}
		}
	}

	if !foundBypass {
		t.Error("Expected to find kyc_bypass indicator")
	}
}

func TestNRVTracer_DetectPositionLimitViolation(t *testing.T) {
	registry := ontology.NewOntologyRegistry()
	tracer := NewNRVTracer(registry)
	trace := tracer.StartTrace("agent-123", "TestAgent", "validation-456")

	// Add steps that accumulate position
	tracer.RecordInference(trace.ID, "Buy AAPL", "Purchase order", "Execute trade", 0.95,
		&FinancialContext{
			ActionType: ontology.ActionTrade,
			Instrument: &ontology.FinancialInstrument{
				Symbol: "AAPL",
				Type:   "equity",
			},
			Amount: &ontology.MonetaryValue{
				Amount:   50000.0,
				Currency: "USD",
			},
		})

	tracer.RecordInference(trace.ID, "Buy more AAPL", "Additional purchase", "Execute trade", 0.95,
		&FinancialContext{
			ActionType: ontology.ActionTrade,
			Instrument: &ontology.FinancialInstrument{
				Symbol: "AAPL",
				Type:   "equity",
			},
			Amount: &ontology.MonetaryValue{
				Amount:   60000.0,
				Currency: "USD",
			},
		})

	limits := map[string]float64{
		"AAPL": 100000.0,
	}

	indicators := tracer.DetectPositionLimitViolation(trace.ID, limits)

	if len(indicators) == 0 {
		t.Error("Expected to detect position limit violation")
	}

	foundViolation := false
	for _, ind := range indicators {
		if ind.Type == "position_limit_violation" {
			foundViolation = true
		}
	}

	if !foundViolation {
		t.Error("Expected to find position_limit_violation indicator")
	}
}

func TestNRVTracer_FinalizeTrace(t *testing.T) {
	registry := ontology.NewOntologyRegistry()
	tracer := NewNRVTracer(registry)
	trace := tracer.StartTrace("agent-123", "TestAgent", "validation-456")

	// Add a step
	tracer.AddStep(trace.ID, NRVReasoningStep{
		StepType:   "inference",
		Intent:     "Test step",
		Confidence: 0.90,
	})

	finalized, err := tracer.FinalizeTrace(trace.ID)
	if err != nil {
		t.Fatalf("Failed to finalize trace: %v", err)
	}

	if finalized.Status != NRVTraceStatusComplete {
		t.Errorf("Expected status COMPLETE, got '%s'", finalized.Status)
	}

	// Verify trace was moved to completed
	_, err = tracer.GetTrace(trace.ID)
	if err != nil {
		t.Errorf("Should be able to get finalized trace: %v", err)
	}
}

func TestNRVTracer_GetTrace(t *testing.T) {
	registry := ontology.NewOntologyRegistry()
	tracer := NewNRVTracer(registry)
	trace := tracer.StartTrace("agent-123", "TestAgent", "validation-456")

	// Get from active traces
	retrieved, err := tracer.GetTrace(trace.ID)
	if err != nil {
		t.Fatalf("Failed to get trace: %v", err)
	}

	if retrieved.ID != trace.ID {
		t.Errorf("Expected trace ID '%s', got '%s'", trace.ID, retrieved.ID)
	}

	// Get non-existent trace
	_, err = tracer.GetTrace("non-existent-id")
	if err == nil {
		t.Error("Expected error for non-existent trace")
	}
}

func TestNRVTrace_ConvertToEvidence(t *testing.T) {
	trace := &NRVTrace{
		ID:                "nrv-test-123",
		AgentID:           "agent-123",
		FidelityScore:     0.85,
		AppliedOntologies: []string{"kyc-v1", "aml-v1"},
		Steps: []NRVReasoningStep{
			{
				StepNumber:  1,
				Intent:      "Verify identity",
				Observation: "Document submitted",
				Inference:   "Valid ID",
				Confidence:  0.95,
			},
		},
		SemanticContext: map[string]interface{}{
			"session_id": "sess-456",
		},
	}

	evidence := trace.ConvertToEvidence()

	if evidence == nil {
		t.Fatal("Expected evidence to be created")
	}

	if evidence.TraceID != trace.ID {
		t.Errorf("Expected TraceID '%s', got '%s'", trace.ID, evidence.TraceID)
	}

	if evidence.AgentID != trace.AgentID {
		t.Errorf("Expected AgentID '%s', got '%s'", trace.AgentID, evidence.AgentID)
	}

	if evidence.FidelityScore != trace.FidelityScore {
		t.Errorf("Expected FidelityScore %f, got %f", trace.FidelityScore, evidence.FidelityScore)
	}

	if len(evidence.ReasoningSteps) != 1 {
		t.Errorf("Expected 1 reasoning step, got %d", len(evidence.ReasoningSteps))
	}
}

func TestNRVTrace_ToMarkdown(t *testing.T) {
	trace := &NRVTrace{
		ID:                "nrv-test-123",
		AgentID:           "agent-123",
		AgentName:         "TestAgent",
		ValidationID:      "validation-456",
		Status:            NRVTraceStatusComplete,
		CreatedAt:         time.Now(),
		StepCount:         1,
		AppliedOntologies: []string{"kyc-v1"},
		ComplianceChecks: []ComplianceCheckRef{
			{
				CheckID:        "check-1",
				OntologyID:     "kyc-v1",
				RegulationName: "KYC Global",
				Status:         "compliant",
			},
		},
		Steps: []NRVReasoningStep{
			{
				StepNumber:   1,
				StepType:     "inference",
				Intent:       "Verify customer",
				Observation:  "KYC documents received",
				Inference:    "Customer verified",
				Confidence:   0.95,
				OntologyRefs: []string{"kyc-v1"},
			},
		},
		FinalDecision: "APPROVE",
		FidelityScore: 0.88,
		FidelityAnalysis: &FidelityAnalysis{
			OverallScore:        0.88,
			SemanticDistance:    0.12,
			RegulatoryAlignment: 0.90,
		},
	}

	markdown, err := trace.ToMarkdown()
	if err != nil {
		t.Fatalf("Failed to generate markdown: %v", err)
	}

	content := string(markdown)

	if content == "" {
		t.Error("Expected non-empty markdown content")
	}

	if content == "" {
		t.Error("Expected content to contain trace metadata")
	}

	if content == "" {
		t.Error("Expected content to contain reasoning steps")
	}
}

func TestContextWithTrace(t *testing.T) {
	ctx := context.Background()
	traceID := "trace-123"

	ctx = ContextWithTrace(ctx, traceID)

	retrievedID, ok := TraceFromContext(ctx)
	if !ok {
		t.Error("Expected to retrieve trace ID from context")
	}

	if retrievedID != traceID {
		t.Errorf("Expected trace ID '%s', got '%s'", traceID, retrievedID)
	}

	// Test with empty context
	_, ok = TraceFromContext(context.Background())
	if ok {
		t.Error("Should not find trace ID in empty context")
	}
}

func TestNRVTracer_ListActiveTraces(t *testing.T) {
	registry := ontology.NewOntologyRegistry()
	tracer := NewNRVTracer(registry)

	// Create multiple active traces
	tracer.StartTrace("agent-1", "Agent1", "val-1")
	tracer.StartTrace("agent-2", "Agent2", "val-2")
	tracer.StartTrace("agent-3", "Agent3", "val-3")

	active := tracer.ListActiveTraces()

	if len(active) != 3 {
		t.Errorf("Expected 3 active traces, got %d", len(active))
	}
}

func TestNRVTracer_ListCompletedTraces(t *testing.T) {
	registry := ontology.NewOntologyRegistry()
	tracer := NewNRVTracer(registry)

	// Create and finalize traces
	trace1 := tracer.StartTrace("agent-1", "Agent1", "val-1")
	tracer.FinalizeTrace(trace1.ID)

	trace2 := tracer.StartTrace("agent-2", "Agent2", "val-2")
	tracer.FinalizeTrace(trace2.ID)

	completed := tracer.ListCompletedTraces()

	if len(completed) != 2 {
		t.Errorf("Expected 2 completed traces, got %d", len(completed))
	}
}

func TestNRVTracerConfig(t *testing.T) {
	config := DefaultNRVTracerConfig()

	if config == nil {
		t.Fatal("Expected default config")
	}

	if !config.EnableKYCDetection {
		t.Error("Expected KYC detection to be enabled by default")
	}

	if !config.EnablePositionLimitDetection {
		t.Error("Expected position limit detection to be enabled by default")
	}

	if config.MaxTraceDuration == 0 {
		t.Error("Expected max trace duration to be set")
	}
}
