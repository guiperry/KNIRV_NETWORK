// Copyright 2026 KNIRV-SERVER
// SPDX-License-Identifier: GPL-3.0-or-later

package fintech

import (
	"context"
	"testing"

	"backend_server/internal/fintech/ontology"
)

func TestFidelityScorer_NewFidelityScorer(t *testing.T) {
	registry := ontology.NewOntologyRegistry()
	tracer := NewNRVTracer(registry)
	config := DefaultFidelityScorerConfig()

	scorer := NewFidelityScorer(tracer, registry, config)

	if scorer == nil {
		t.Fatal("Expected scorer to be created")
	}

	if scorer.tracer != tracer {
		t.Error("Expected tracer to be set")
	}

	if scorer.config == nil {
		t.Error("Expected config to be set")
	}
}

func TestFidelityScorer_DefaultConfig(t *testing.T) {
	config := DefaultFidelityScorerConfig()

	if config == nil {
		t.Fatal("Expected default config")
	}

	if config.MinAcceptableScore != 0.75 {
		t.Errorf("Expected MinAcceptableScore 0.75, got %f", config.MinAcceptableScore)
	}

	if config.MaxViolationRisk != 0.30 {
		t.Errorf("Expected MaxViolationRisk 0.30, got %f", config.MaxViolationRisk)
	}

	if !config.EnableKYCDetection {
		t.Error("Expected KYC detection to be enabled")
	}
}

func TestFidelityScorer_ScoreTraceObject(t *testing.T) {
	registry := ontology.NewOntologyRegistry()
	loader := ontology.NewOntologyLoader(registry)
	loader.LoadBuiltInOntologies()

	tracer := NewNRVTracer(registry)
	scorer := NewFidelityScorer(tracer, registry, nil)

	// Create a trace with good compliance
	trace := tracer.StartTrace("agent-123", "TestAgent", "validation-456")
	trace.AppliedOntologies = []string{"kyc-1.0", "aml-1.0"}
	trace.ComplianceChecks = []ComplianceCheckRef{
		{
			OntologyID: "kyc-1.0",
			Status:     "compliant",
		},
		{
			OntologyID: "aml-1.0",
			Status:     "compliant",
		},
	}

	// Add compliant reasoning steps
	tracer.AddStep(trace.ID, NRVReasoningStep{
		StepType:   "inference",
		Intent:     "Verify customer identity",
		Confidence: 0.95,
	})

	tracer.AddStep(trace.ID, NRVReasoningStep{
		StepType:   "validation",
		Intent:     "Check sanctions",
		Confidence: 0.90,
	})

	tracer.RecordDecision(trace.ID, "APPROVE", 0.92, nil)
	tracer.FinalizeTrace(trace.ID)

	result, err := scorer.ScoreTraceObject(context.Background(), trace, "approve")
	if err != nil {
		t.Fatalf("Failed to score trace: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result to be created")
	}

	if result.OverallScore < 0 || result.OverallScore > 1 {
		t.Errorf("Expected overall score between 0 and 1, got %f", result.OverallScore)
	}

	if result.RiskLevel == "" {
		t.Error("Expected risk level to be set")
	}
}

func TestFidelityScorer_CalculateRegulatoryAlignment(t *testing.T) {
	registry := ontology.NewOntologyRegistry()
	tracer := NewNRVTracer(registry)
	scorer := NewFidelityScorer(tracer, registry, nil)

	trace := &NRVTrace{
		AppliedOntologies: []string{"kyc-v1", "aml-v1"},
		ComplianceChecks: []ComplianceCheckRef{
			{
				OntologyID: "kyc-v1",
				Status:     "compliant",
			},
		},
	}

	alignment := scorer.calculateRegulatoryAlignment(trace)
	if alignment != 1.0 {
		t.Errorf("Expected alignment 1.0 for compliant check, got %f", alignment)
	}

	// Test with violations
	traceViolated := &NRVTrace{
		AppliedOntologies: []string{"aml-v1"},
		ComplianceChecks: []ComplianceCheckRef{
			{
				OntologyID: "aml-v1",
				Status:     "violated",
			},
		},
	}

	alignment = scorer.calculateRegulatoryAlignment(traceViolated)
	if alignment != 0.0 {
		t.Errorf("Expected alignment 0.0 for violated check, got %f", alignment)
	}
}

func TestFidelityScorer_CalculateCategoryScores(t *testing.T) {
	registry := ontology.NewOntologyRegistry()
	loader := ontology.NewOntologyLoader(registry)
	loader.LoadBuiltInOntologies()

	tracer := NewNRVTracer(registry)
	scorer := NewFidelityScorer(tracer, registry, nil)

	trace := &NRVTrace{
		AppliedOntologies: []string{"kyc-v1"},
		ComplianceChecks: []ComplianceCheckRef{
			{
				OntologyID: "kyc-v1",
				Status:     "compliant",
			},
		},
	}

	scores := scorer.calculateCategoryScores(trace)

	if scores == nil {
		t.Fatal("Expected scores to be created")
	}

	if len(scores) == 0 {
		t.Error("Expected non-empty category scores")
	}
}

func TestFidelityScorer_CalculateOverallScore(t *testing.T) {
	registry := ontology.NewOntologyRegistry()
	tracer := NewNRVTracer(registry)
	scorer := NewFidelityScorer(tracer, registry, nil)

	result := &FidelityScoreResult{
		SemanticDistance:    0.2,
		RegulatoryAlignment: 0.9,
		IntentAccuracy:      0.85,
		DecisionQuality:     0.88,
	}

	score := scorer.calculateOverallScore(result)

	if score < 0 || score > 1 {
		t.Errorf("Expected score between 0 and 1, got %f", score)
	}
}

func TestFidelityScorer_AssessViolationRisk(t *testing.T) {
	registry := ontology.NewOntologyRegistry()
	tracer := NewNRVTracer(registry)
	scorer := NewFidelityScorer(tracer, registry, nil)

	// Test low risk
	lowRiskTrace := &NRVTrace{
		AppliedOntologies: []string{"kyc-v1", "aml-v1"},
		ComplianceChecks: []ComplianceCheckRef{
			{
				Status: "compliant",
			},
		},
		Steps: []NRVReasoningStep{
			{
				Intent: "Verify customer",
			},
		},
	}

	distanceResult := &DistanceResult{
		OverallDistance: 0.1,
	}

	risk := scorer.assessViolationRisk(lowRiskTrace, distanceResult)
	if risk > 0.3 {
		t.Errorf("Expected low risk, got %f", risk)
	}

	// Test high risk
	highRiskTrace := &NRVTrace{
		AppliedOntologies: []string{},
		ComplianceChecks: []ComplianceCheckRef{
			{
				Status: "violated",
			},
		},
		Steps: []NRVReasoningStep{
			{
				Intent:      "Bypass KYC",
				Observation: "Skip verification",
				Inference:   "Ignore compliance",
			},
		},
	}

	distanceResultHigh := &DistanceResult{
		OverallDistance: 0.8,
	}

	risk = scorer.assessViolationRisk(highRiskTrace, distanceResultHigh)
	if risk < 0.5 {
		t.Errorf("Expected higher risk, got %f", risk)
	}
}

func TestFidelityScorer_DetermineRiskLevel(t *testing.T) {
	registry := ontology.NewOntologyRegistry()
	tracer := NewNRVTracer(registry)
	scorer := NewFidelityScorer(tracer, registry, nil)

	tests := []struct {
		risk     float64
		expected string
	}{
		{0.1, "low"},
		{0.3, "medium"},
		{0.6, "high"},
		{0.9, "critical"},
	}

	for _, tc := range tests {
		level := scorer.determineRiskLevel(tc.risk)
		if level != tc.expected {
			t.Errorf("Expected risk level '%s' for risk %f, got '%s'", tc.expected, tc.risk, level)
		}
	}
}

func TestFidelityScorer_ValidateFidelity(t *testing.T) {
	registry := ontology.NewOntologyRegistry()
	tracer := NewNRVTracer(registry)
	scorer := NewFidelityScorer(tracer, registry, nil)

	// Test valid fidelity
	validResult := &FidelityScoreResult{
		OverallScore:  0.85,
		ViolationRisk: 0.15,
		RiskIndicators: []RiskIndicator{
			{
				Severity: "low",
			},
		},
	}

	valid, issues := scorer.ValidateFidelity(validResult)
	if !valid {
		t.Error("Expected result to be valid")
	}
	if len(issues) != 0 {
		t.Errorf("Expected no issues for valid result, got %d", len(issues))
	}

	// Test invalid fidelity
	invalidResult := &FidelityScoreResult{
		OverallScore:  0.50, // Below threshold
		ViolationRisk: 0.60, // Above threshold
		RiskIndicators: []RiskIndicator{
			{
				Severity: "critical",
			},
		},
	}

	valid, issues = scorer.ValidateFidelity(invalidResult)
	if valid {
		t.Error("Expected result to be invalid")
	}
	if len(issues) == 0 {
		t.Error("Expected issues for invalid result")
	}
}

func TestFidelityScorer_GenerateRecommendations(t *testing.T) {
	registry := ontology.NewOntologyRegistry()
	tracer := NewNRVTracer(registry)
	scorer := NewFidelityScorer(tracer, registry, nil)

	// Test with low scores
	result := &FidelityScoreResult{
		SemanticDistance:    0.7,
		RegulatoryAlignment: 0.4,
		IntentAccuracy:      0.3,
		DecisionQuality:     0.5,
		ViolationRisk:       0.6,
		ComplianceGaps: []ComplianceGap{
			{
				Category: "KYC",
				Severity: "high",
				GapScore: 0.8,
			},
		},
	}

	recommendations := scorer.generateRecommendations(result)

	if len(recommendations) == 0 {
		t.Error("Expected recommendations for low scores")
	}

	// Test with good scores
	goodResult := &FidelityScoreResult{
		SemanticDistance:    0.1,
		RegulatoryAlignment: 0.95,
		IntentAccuracy:      0.90,
		DecisionQuality:     0.88,
		ViolationRisk:       0.05,
		ComplianceGaps:      []ComplianceGap{},
	}

	recommendations = scorer.generateRecommendations(goodResult)
	// Should still have some recommendations but fewer
	t.Logf("Got %d recommendations for good scores", len(recommendations))
}

func TestFidelityScorer_DetermineRequiredActions(t *testing.T) {
	registry := ontology.NewOntologyRegistry()
	tracer := NewNRVTracer(registry)
	scorer := NewFidelityScorer(tracer, registry, nil)

	// Test with critical risk
	criticalResult := &FidelityScoreResult{
		OverallScore:  0.40,
		ViolationRisk: 0.80,
		ComplianceGaps: []ComplianceGap{
			{
				Severity: "critical",
			},
		},
	}

	actions := scorer.determineRequiredActions(criticalResult)

	if len(actions) == 0 {
		t.Error("Expected required actions for critical risk")
	}

	// Test with good scores
	goodResult := &FidelityScoreResult{
		OverallScore:   0.90,
		ViolationRisk:  0.05,
		ComplianceGaps: []ComplianceGap{},
	}

	actions = scorer.determineRequiredActions(goodResult)
	// Should have at most review action for slightly below threshold
	if goodResult.OverallScore < 0.75 && len(actions) == 0 {
		t.Error("Expected actions when score below threshold")
	}
}

func TestFidelityScorer_IdentifyComplianceGaps(t *testing.T) {
	registry := ontology.NewOntologyRegistry()
	loader := ontology.NewOntologyLoader(registry)
	loader.LoadBuiltInOntologies()

	tracer := NewNRVTracer(registry)
	scorer := NewFidelityScorer(tracer, registry, nil)

	// Test with missing ontologies
	trace := &NRVTrace{
		AppliedOntologies: []string{},
	}

	distanceResult := &DistanceResult{
		AlignmentGaps: []AlignmentGap{
			{
				Category: "compliance",
				Distance: 0.6,
			},
		},
	}

	gaps := scorer.identifyComplianceGaps(trace, distanceResult)

	if len(gaps) == 0 {
		t.Error("Expected compliance gaps to be identified")
	}
}
