// Copyright 2026 KNIRV-NEXUS
// SPDX-License-Identifier: GPL-3.0-or-later

package fintech

import (
	"fmt"
	"math"
	"strings"

	"backend_server/internal/fintech/ontology"
)

// SemanticDistanceCalculator calculates semantic distance between reasoning and regulatory requirements
type SemanticDistanceCalculator struct {
	ontologyRegistry *ontology.OntologyRegistry

	// Weight vectors for different distance components
	IntentWeight     float64
	ActionWeight     float64
	ComplianceWeight float64
	OutcomeWeight    float64
}

// DistanceResult contains the calculated semantic distance
type DistanceResult struct {
	OverallDistance    float64 `json:"overall_distance"` // 0.0 = perfect alignment, 1.0 = complete misalignment
	IntentDistance     float64 `json:"intent_distance"`
	ActionDistance     float64 `json:"action_distance"`
	ComplianceDistance float64 `json:"compliance_distance"`
	OutcomeDistance    float64 `json:"outcome_distance"`

	// Component breakdown
	OntologyDistances map[string]float64 `json:"ontology_distances"`
	RuleViolations    []RuleViolation    `json:"rule_violations,omitempty"`
	AlignmentGaps     []AlignmentGap     `json:"alignment_gaps,omitempty"`
}

// RuleViolation represents a rule that was violated
type RuleViolation struct {
	RuleID          string  `json:"rule_id"`
	RuleName        string  `json:"rule_name"`
	OntologyID      string  `json:"ontology_id"`
	ExpectedOutcome string  `json:"expected_outcome"`
	ActualOutcome   string  `json:"actual_outcome"`
	Distance        float64 `json:"distance"`
	Severity        string  `json:"severity"`
}

// AlignmentGap represents a gap between expected and actual reasoning
type AlignmentGap struct {
	Category    string  `json:"category"` // intent, action, compliance, outcome
	Description string  `json:"description"`
	Expected    string  `json:"expected"`
	Actual      string  `json:"actual"`
	Distance    float64 `json:"distance"`
}

// NewSemanticDistanceCalculator creates a new calculator
func NewSemanticDistanceCalculator(registry *ontology.OntologyRegistry) *SemanticDistanceCalculator {
	return &SemanticDistanceCalculator{
		ontologyRegistry: registry,
		IntentWeight:     0.25,
		ActionWeight:     0.25,
		ComplianceWeight: 0.30,
		OutcomeWeight:    0.20,
	}
}

// CalculateDistance calculates the semantic distance for a trace
func (c *SemanticDistanceCalculator) CalculateDistance(trace *NRVTrace, expectedOutcome string) (*DistanceResult, error) {
	if trace == nil {
		return nil, fmt.Errorf("trace cannot be nil")
	}

	result := &DistanceResult{
		OntologyDistances: make(map[string]float64),
		RuleViolations:    make([]RuleViolation, 0),
		AlignmentGaps:     make([]AlignmentGap, 0),
	}

	// Calculate intent distance
	result.IntentDistance = c.calculateIntentDistance(trace)

	// Calculate action distance
	result.ActionDistance = c.calculateActionDistance(trace)

	// Calculate compliance distance
	complianceDist, violations, gaps := c.calculateComplianceDistance(trace)
	result.ComplianceDistance = complianceDist
	result.RuleViolations = violations
	result.AlignmentGaps = append(result.AlignmentGaps, gaps...)

	// Calculate outcome distance
	result.OutcomeDistance = c.calculateOutcomeDistance(trace, expectedOutcome)

	// Calculate weighted overall distance
	result.OverallDistance = c.weightedAverage(
		result.IntentDistance,
		result.ActionDistance,
		result.ComplianceDistance,
		result.OutcomeDistance,
	)

	// Calculate per-ontology distances
	for _, ontologyID := range trace.AppliedOntologies {
		dist := c.calculateOntologyDistance(trace, ontologyID)
		result.OntologyDistances[ontologyID] = dist
	}

	return result, nil
}

// calculateIntentDistance measures how well agent intent aligns with financial goals
func (c *SemanticDistanceCalculator) calculateIntentDistance(trace *NRVTrace) float64 {
	if len(trace.Steps) == 0 {
		return 1.0 // Maximum distance if no steps
	}

	// Keywords that indicate suspicious or non-compliant intent
	suspiciousIntentPatterns := []string{
		"bypass", "circumvent", "avoid", "hide", "conceal",
		"override", "ignore", "skip", "fast track", "expedite",
		"unverified", "unauthorized", "unapproved",
	}

	compliantIntentPatterns := []string{
		"verify", "validate", "check", "confirm", "ensure",
		"comply", "adhere", "follow", "implement", "enforce",
		"screen", "audit", "review", "assess",
	}

	suspiciousCount := 0
	compliantCount := 0

	for _, step := range trace.Steps {
		intentLower := strings.ToLower(step.Intent)

		for _, pattern := range suspiciousIntentPatterns {
			if strings.Contains(intentLower, pattern) {
				suspiciousCount++
				break
			}
		}

		for _, pattern := range compliantIntentPatterns {
			if strings.Contains(intentLower, pattern) {
				compliantCount++
				break
			}
		}
	}

	// Calculate distance based on ratio
	totalSteps := len(trace.Steps)
	if totalSteps == 0 {
		return 1.0
	}

	// Distance increases with suspicious patterns, decreases with compliant patterns
	suspiciousRatio := float64(suspiciousCount) / float64(totalSteps)
	compliantRatio := float64(compliantCount) / float64(totalSteps)

	distance := suspiciousRatio - (compliantRatio * 0.5)
	if distance < 0 {
		distance = 0
	}
	if distance > 1 {
		distance = 1
	}

	return distance
}

// calculateActionDistance measures alignment between actions and regulatory requirements
func (c *SemanticDistanceCalculator) calculateActionDistance(trace *NRVTrace) float64 {
	if len(trace.Steps) == 0 {
		return 1.0
	}

	requiredActions := make(map[string]bool)
	performedActions := make(map[string]bool)

	// Define required actions based on financial context
	for _, step := range trace.Steps {
		if step.FinancialContext == nil {
			continue
		}

		switch step.FinancialContext.ActionType {
		case ontology.ActionTransfer:
			requiredActions["kyc_verification"] = false
			requiredActions["sanctions_screening"] = false
			requiredActions["amount_verification"] = false

			if step.FinancialContext.Counterparty != nil {
				performedActions["kyc_verification"] = true

				if step.FinancialContext.Counterparty.IsSanctioned {
					performedActions["sanctions_screening"] = true
				}
			}

			if step.FinancialContext.Amount != nil {
				performedActions["amount_verification"] = true
			}

		case ontology.ActionTrade:
			requiredActions["position_limit_check"] = false
			requiredActions["market_regulation_check"] = false

			if step.FinancialContext.Instrument != nil {
				performedActions["market_regulation_check"] = true
			}

		case ontology.ActionKYC:
			requiredActions["identity_verification"] = false
			requiredActions["risk_assessment"] = false

			if step.FinancialContext.Counterparty != nil {
				performedActions["identity_verification"] = true
				if step.FinancialContext.Counterparty.RiskRating != "" {
					performedActions["risk_assessment"] = true
				}
			}
		}
	}

	// Calculate distance based on missing required actions
	if len(requiredActions) == 0 {
		return 0.0 // No required actions means no distance
	}

	missingCount := 0
	for action := range requiredActions {
		if !performedActions[action] {
			missingCount++
		}
	}

	distance := float64(missingCount) / float64(len(requiredActions))
	return math.Min(distance, 1.0)
}

// calculateComplianceDistance measures regulatory compliance alignment
func (c *SemanticDistanceCalculator) calculateComplianceDistance(trace *NRVTrace) (float64, []RuleViolation, []AlignmentGap) {
	violations := make([]RuleViolation, 0)
	gaps := make([]AlignmentGap, 0)

	if len(trace.ComplianceChecks) == 0 {
		// No compliance checks means maximum distance
		return 1.0, violations, gaps
	}

	violatedCount := 0
	warningCount := 0

	for _, check := range trace.ComplianceChecks {
		if check.Status == "violated" {
			violatedCount++
			violations = append(violations, RuleViolation{
				RuleID:          check.CheckID,
				RuleName:        check.RegulationName,
				OntologyID:      check.OntologyID,
				ExpectedOutcome: "compliant",
				ActualOutcome:   "violated",
				Distance:        1.0,
				Severity:        "critical",
			})
		} else if check.Status == "warning" {
			warningCount++
			violations = append(violations, RuleViolation{
				RuleID:          check.CheckID,
				RuleName:        check.RegulationName,
				OntologyID:      check.OntologyID,
				ExpectedOutcome: "compliant",
				ActualOutcome:   "warning",
				Distance:        0.5,
				Severity:        "medium",
			})
		}
	}

	totalChecks := len(trace.ComplianceChecks)
	if totalChecks == 0 {
		return 1.0, violations, gaps
	}

	// Distance based on violations and warnings
	violationWeight := 1.0
	warningWeight := 0.5

	distance := (float64(violatedCount)*violationWeight + float64(warningCount)*warningWeight) / float64(totalChecks)
	distance = math.Min(distance, 1.0)

	// Add alignment gaps for missing ontology applications
	requiredOntologies := []string{"kyc", "aml", "sec"}
	for _, reqOntology := range requiredOntologies {
		found := false
		for _, applied := range trace.AppliedOntologies {
			if strings.Contains(strings.ToLower(applied), reqOntology) {
				found = true
				break
			}
		}

		if !found {
			gaps = append(gaps, AlignmentGap{
				Category:    "compliance",
				Description: fmt.Sprintf("Missing %s ontology application", strings.ToUpper(reqOntology)),
				Expected:    fmt.Sprintf("Apply %s regulations", strings.ToUpper(reqOntology)),
				Actual:      "Not applied",
				Distance:    0.3,
			})
		}
	}

	return distance, violations, gaps
}

// calculateOutcomeDistance measures distance between actual and expected outcomes
func (c *SemanticDistanceCalculator) calculateOutcomeDistance(trace *NRVTrace, expectedOutcome string) float64 {
	if expectedOutcome == "" {
		// No expected outcome defined, use heuristic
		return c.calculateHeuristicOutcomeDistance(trace)
	}

	actualOutcome := strings.ToLower(trace.FinalDecision)
	expectedLower := strings.ToLower(expectedOutcome)

	// Perfect match
	if actualOutcome == expectedLower {
		return 0.0
	}

	// Partial match based on semantic similarity
	similarity := c.calculateSemanticSimilarity(actualOutcome, expectedLower)
	distance := 1.0 - similarity

	return math.Min(distance, 1.0)
}

// calculateHeuristicOutcomeDistance uses heuristics when no expected outcome is defined
func (c *SemanticDistanceCalculator) calculateHeuristicOutcomeDistance(trace *NRVTrace) float64 {
	// Check for red flags in final decision
	redFlags := []string{
		"approve unverified",
		"bypass required",
		"skip mandatory",
		"override limits",
		"ignore violations",
	}

	decisionLower := strings.ToLower(trace.FinalDecision)

	for _, flag := range redFlags {
		if strings.Contains(decisionLower, flag) {
			return 0.8 // High distance for red flags
		}
	}

	// Check for positive indicators
	positiveIndicators := []string{
		"compliant",
		"verified",
		"approved after checks",
		"within limits",
		"cleared",
	}

	for _, indicator := range positiveIndicators {
		if strings.Contains(decisionLower, indicator) {
			return 0.1 // Low distance for positive outcomes
		}
	}

	return 0.5 // Neutral distance
}

// calculateOntologyDistance calculates distance for a specific ontology
func (c *SemanticDistanceCalculator) calculateOntologyDistance(trace *NRVTrace, ontologyID string) float64 {
	ontology, err := c.ontologyRegistry.Get(ontologyID)
	if err != nil {
		return 1.0 // Maximum distance if ontology not found
	}

	// Get all rules for this ontology
	rules := ontology.GetRules()
	if len(rules) == 0 {
		return 0.0 // No rules means no distance
	}

	// Count how many rules were evaluated
	evaluatedCount := 0
	violatedCount := 0

	for _, check := range trace.ComplianceChecks {
		if check.OntologyID == ontologyID {
			evaluatedCount++
			if check.Status == "violated" {
				violatedCount++
			}
		}
	}

	if evaluatedCount == 0 {
		// Ontology was applied but no checks recorded
		return 0.5
	}

	// Distance based on violation ratio
	distance := float64(violatedCount) / float64(len(rules))
	return math.Min(distance, 1.0)
}

// calculateSemanticSimilarity calculates similarity between two strings (0.0 - 1.0)
func (c *SemanticDistanceCalculator) calculateSemanticSimilarity(str1, str2 string) float64 {
	// Simple implementation using word overlap
	words1 := strings.Fields(str1)
	words2 := strings.Fields(str2)

	if len(words1) == 0 || len(words2) == 0 {
		return 0.0
	}

	// Create word sets
	set1 := make(map[string]bool)
	for _, w := range words1 {
		set1[strings.ToLower(w)] = true
	}

	// Count matches
	matches := 0
	for _, w := range words2 {
		if set1[strings.ToLower(w)] {
			matches++
		}
	}

	// Jaccard similarity
	union := len(words1) + len(words2) - matches
	if union == 0 {
		return 0.0
	}

	return float64(matches) / float64(union)
}

// weightedAverage calculates weighted average of distances
func (c *SemanticDistanceCalculator) weightedAverage(intentDist, actionDist, complianceDist, outcomeDist float64) float64 {
	totalWeight := c.IntentWeight + c.ActionWeight + c.ComplianceWeight + c.OutcomeWeight

	if totalWeight == 0 {
		return 0.5 // Neutral if no weights
	}

	weightedSum := (intentDist * c.IntentWeight) +
		(actionDist * c.ActionWeight) +
		(complianceDist * c.ComplianceWeight) +
		(outcomeDist * c.OutcomeWeight)

	return weightedSum / totalWeight
}

// SetWeights allows customization of distance component weights
func (c *SemanticDistanceCalculator) SetWeights(intent, action, compliance, outcome float64) {
	total := intent + action + compliance + outcome
	if total == 0 {
		return
	}

	// Normalize to sum to 1
	c.IntentWeight = intent / total
	c.ActionWeight = action / total
	c.ComplianceWeight = compliance / total
	c.OutcomeWeight = outcome / total
}

// CalculateRegulatoryAlignment calculates how well reasoning aligns with specific regulations
func (c *SemanticDistanceCalculator) CalculateRegulatoryAlignment(trace *NRVTrace, category ontology.RegulationCategory) float64 {
	// Get ontologies for this category
	ontologies := c.ontologyRegistry.ListByCategory(category)
	if len(ontologies) == 0 {
		return 0.0
	}

	totalAlignment := 0.0
	appliedCount := 0

	for _, ontology := range ontologies {
		// Check if this ontology was applied
		wasApplied := false
		for _, appliedID := range trace.AppliedOntologies {
			if appliedID == ontology.GetID() {
				wasApplied = true
				break
			}
		}

		if wasApplied {
			dist := c.calculateOntologyDistance(trace, ontology.GetID())
			alignment := 1.0 - dist
			totalAlignment += alignment
			appliedCount++
		}
	}

	if appliedCount == 0 {
		return 0.0 // No ontologies applied from this category
	}

	return totalAlignment / float64(appliedCount)
}

// IdentifyCriticalGaps identifies the most critical alignment gaps
func (c *SemanticDistanceCalculator) IdentifyCriticalGaps(trace *NRVTrace, result *DistanceResult) []AlignmentGap {
	criticalGaps := make([]AlignmentGap, 0)

	for _, gap := range result.AlignmentGaps {
		if gap.Distance >= 0.7 {
			criticalGaps = append(criticalGaps, gap)
		}
	}

	// Sort by distance descending (most critical first)
	// Simple bubble sort for small lists
	for i := 0; i < len(criticalGaps); i++ {
		for j := i + 1; j < len(criticalGaps); j++ {
			if criticalGaps[j].Distance > criticalGaps[i].Distance {
				criticalGaps[i], criticalGaps[j] = criticalGaps[j], criticalGaps[i]
			}
		}
	}

	return criticalGaps
}
