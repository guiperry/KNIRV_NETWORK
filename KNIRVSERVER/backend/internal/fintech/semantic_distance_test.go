// Copyright 2026 KNIRV-SERVER
// SPDX-License-Identifier: GPL-3.0-or-later

package fintech

import (
	"testing"

	"backend_server/internal/fintech/ontology"
)

func TestSemanticDistanceCalculator_CalculateDistance(t *testing.T) {
	registry := ontology.NewOntologyRegistry()
	loader := ontology.NewOntologyLoader(registry)
	loader.LoadBuiltInOntologies()

	calculator := NewSemanticDistanceCalculator(registry)

	trace := &NRVTrace{
		AgentID:           "agent-123",
		AppliedOntologies: []string{"kyc-1.0", "aml-1.0"},
		ComplianceChecks: []ComplianceCheckRef{
			{
				OntologyID:     "kyc-1.0",
				RegulationName: "KYC Global",
				Status:         "compliant",
			},
		},
		Steps: []NRVReasoningStep{
			{
				Intent:     "Verify customer identity",
				Confidence: 0.95,
			},
			{
				Intent:     "Check sanctions",
				Confidence: 0.90,
			},
		},
		FinalDecision: "APPROVE",
	}

	result, err := calculator.CalculateDistance(trace, "approve")
	if err != nil {
		t.Fatalf("Failed to calculate distance: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result to be created")
	}

	if result.OverallDistance < 0 || result.OverallDistance > 1 {
		t.Errorf("Expected overall distance between 0 and 1, got %f", result.OverallDistance)
	}
}

func TestSemanticDistanceCalculator_IntentDistance(t *testing.T) {
	registry := ontology.NewOntologyRegistry()
	calculator := NewSemanticDistanceCalculator(registry)

	// Test with compliant intent
	compliantTrace := &NRVTrace{
		Steps: []NRVReasoningStep{
			{
				Intent: "Verify and validate customer information",
			},
			{
				Intent: "Check compliance with regulations",
			},
		},
	}

	distance := calculator.calculateIntentDistance(compliantTrace)
	if distance > 0.3 {
		t.Errorf("Expected low distance for compliant intent, got %f", distance)
	}

	// Test with suspicious intent
	suspiciousTrace := &NRVTrace{
		Steps: []NRVReasoningStep{
			{
				Intent: "Bypass KYC verification",
			},
			{
				Intent: "Skip mandatory checks",
			},
		},
	}

	distance = calculator.calculateIntentDistance(suspiciousTrace)
	if distance < 0.5 {
		t.Errorf("Expected higher distance for suspicious intent, got %f", distance)
	}
}

func TestSemanticDistanceCalculator_ActionDistance(t *testing.T) {
	registry := ontology.NewOntologyRegistry()
	calculator := NewSemanticDistanceCalculator(registry)

	// Test transfer with KYC
	trace := &NRVTrace{
		Steps: []NRVReasoningStep{
			{
				FinancialContext: &FinancialContext{
					ActionType: ontology.ActionTransfer,
					Counterparty: &ontology.CounterpartyInfo{
						ID: "cust-123",
					},
					Amount: &ontology.MonetaryValue{
						Amount:   1000,
						Currency: "USD",
					},
				},
			},
		},
	}

	distance := calculator.calculateActionDistance(trace)
	if distance > 0.5 {
		t.Errorf("Expected lower distance when KYC data provided, got %f", distance)
	}

	// Test transfer without KYC
	traceNoKYC := &NRVTrace{
		Steps: []NRVReasoningStep{
			{
				FinancialContext: &FinancialContext{
					ActionType: ontology.ActionTransfer,
					Amount: &ontology.MonetaryValue{
						Amount:   1000,
						Currency: "USD",
					},
				},
			},
		},
	}

	distance = calculator.calculateActionDistance(traceNoKYC)
	if distance < 0.3 {
		t.Errorf("Expected higher distance when KYC missing, got %f", distance)
	}
}

func TestSemanticDistanceCalculator_ComplianceDistance(t *testing.T) {
	registry := ontology.NewOntologyRegistry()
	calculator := NewSemanticDistanceCalculator(registry)

	// Test with no compliance checks
	trace := &NRVTrace{
		ComplianceChecks: []ComplianceCheckRef{},
	}

	distance, _, _ := calculator.calculateComplianceDistance(trace)
	if distance != 1.0 {
		t.Errorf("Expected max distance with no checks, got %f", distance)
	}

	// Test with violations
	traceWithViolations := &NRVTrace{
		ComplianceChecks: []ComplianceCheckRef{
			{
				OntologyID: "aml-v1",
				Status:     "violated",
			},
			{
				OntologyID: "kyc-v1",
				Status:     "compliant",
			},
		},
	}

	distance, violations, _ := calculator.calculateComplianceDistance(traceWithViolations)
	if distance == 0 {
		t.Error("Expected non-zero distance with violations")
	}

	if len(violations) == 0 {
		t.Error("Expected violations to be recorded")
	}
}

func TestSemanticDistanceCalculator_OutcomeDistance(t *testing.T) {
	registry := ontology.NewOntologyRegistry()
	calculator := NewSemanticDistanceCalculator(registry)

	// Test perfect match
	trace := &NRVTrace{
		FinalDecision: "approve",
	}

	distance := calculator.calculateOutcomeDistance(trace, "approve")
	if distance != 0.0 {
		t.Errorf("Expected zero distance for perfect match, got %f", distance)
	}

	// Test no match
	distance = calculator.calculateOutcomeDistance(trace, "deny")
	if distance == 0.0 {
		t.Error("Expected non-zero distance for no match")
	}
}

func TestSemanticDistanceCalculator_OntologyDistance(t *testing.T) {
	registry := ontology.NewOntologyRegistry()
	loader := ontology.NewOntologyLoader(registry)
	loader.LoadBuiltInOntologies()

	calculator := NewSemanticDistanceCalculator(registry)

	// Test with compliant check - using actual ontology ID
	trace := &NRVTrace{
		ComplianceChecks: []ComplianceCheckRef{
			{
				OntologyID: "kyc-1.0",
				Status:     "compliant",
			},
		},
	}

	distance := calculator.calculateOntologyDistance(trace, "kyc-1.0")
	if distance > 0.3 {
		t.Errorf("Expected low distance for compliant check, got %f", distance)
	}

	// Test with violated check
	traceViolated := &NRVTrace{
		ComplianceChecks: []ComplianceCheckRef{
			{
				OntologyID: "kyc-1.0",
				Status:     "violated",
			},
		},
	}

	distance = calculator.calculateOntologyDistance(traceViolated, "kyc-1.0")
	// Distance is based on violation ratio - should be > 0 since there's a violation
	if distance == 0 {
		t.Errorf("Expected non-zero distance for violated check, got %f", distance)
	}
}

func TestSemanticDistanceCalculator_SemanticSimilarity(t *testing.T) {
	registry := ontology.NewOntologyRegistry()
	calculator := NewSemanticDistanceCalculator(registry)

	// Test identical strings
	similarity := calculator.calculateSemanticSimilarity("approve transaction", "approve transaction")
	if similarity != 1.0 {
		t.Errorf("Expected similarity 1.0 for identical strings, got %f", similarity)
	}

	// Test no overlap
	similarity = calculator.calculateSemanticSimilarity("approve", "deny")
	if similarity != 0.0 {
		t.Errorf("Expected similarity 0.0 for no overlap, got %f", similarity)
	}

	// Test partial overlap
	similarity = calculator.calculateSemanticSimilarity("approve transaction", "approve transfer")
	if similarity == 0.0 {
		t.Error("Expected non-zero similarity for partial overlap")
	}
}

func TestSemanticDistanceCalculator_WeightedAverage(t *testing.T) {
	registry := ontology.NewOntologyRegistry()
	calculator := NewSemanticDistanceCalculator(registry)

	// Test with equal weights
	calculator.SetWeights(0.25, 0.25, 0.25, 0.25)

	avg := calculator.weightedAverage(0.2, 0.4, 0.6, 0.8)
	expected := (0.2 + 0.4 + 0.6 + 0.8) / 4

	if avg != expected {
		t.Errorf("Expected average %f, got %f", expected, avg)
	}

	// Test with custom weights
	calculator.SetWeights(0.5, 0.2, 0.2, 0.1)

	avg = calculator.weightedAverage(0.0, 0.0, 0.0, 1.0)
	// Should be heavily weighted toward the first component
	if avg > 0.3 {
		t.Errorf("Expected weighted average to favor first component, got %f", avg)
	}
}

func TestSemanticDistanceCalculator_CalculateRegulatoryAlignment(t *testing.T) {
	registry := ontology.NewOntologyRegistry()
	loader := ontology.NewOntologyLoader(registry)
	loader.LoadBuiltInOntologies()

	calculator := NewSemanticDistanceCalculator(registry)

	// Test with KYC trace - using actual ontology ID
	trace := &NRVTrace{
		AppliedOntologies: []string{"kyc-1.0"},
		ComplianceChecks: []ComplianceCheckRef{
			{
				OntologyID: "kyc-1.0",
				Status:     "compliant",
			},
		},
	}

	alignment := calculator.CalculateRegulatoryAlignment(trace, ontology.CategoryKYC)
	if alignment < 0.7 {
		t.Errorf("Expected high alignment for compliant KYC check, got %f", alignment)
	}
}

func TestSemanticDistanceCalculator_IdentifyCriticalGaps(t *testing.T) {
	registry := ontology.NewOntologyRegistry()
	calculator := NewSemanticDistanceCalculator(registry)

	// Create trace with alignment gaps
	result := &DistanceResult{
		AlignmentGaps: []AlignmentGap{
			{
				Category: "compliance",
				Distance: 0.9, // Critical
			},
			{
				Category: "intent",
				Distance: 0.3, // Low
			},
			{
				Category: "action",
				Distance: 0.8, // Critical
			},
		},
	}

	trace := &NRVTrace{}
	criticalGaps := calculator.IdentifyCriticalGaps(trace, result)

	if len(criticalGaps) != 2 {
		t.Errorf("Expected 2 critical gaps, got %d", len(criticalGaps))
	}
}

func TestSemanticDistanceCalculator_SetWeights(t *testing.T) {
	registry := ontology.NewOntologyRegistry()
	calculator := NewSemanticDistanceCalculator(registry)

	// Set custom weights
	calculator.SetWeights(0.4, 0.3, 0.2, 0.1)

	// Test that weights are normalized
	avg := calculator.weightedAverage(1.0, 1.0, 1.0, 1.0)
	if avg != 1.0 {
		t.Errorf("Expected average 1.0 for all ones, got %f", avg)
	}
}
