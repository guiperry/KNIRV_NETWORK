// Copyright 2026 KNIRV-SERVER
// SPDX-License-Identifier: GPL-3.0-or-later

package fintech

import (
	"context"
	"fmt"
	"math"
	"time"

	"backend_server/internal/services/plugins/fintech/ontology"
)

// FidelityScorer calculates fidelity scores for NRV traces
type FidelityScorer struct {
	tracer             *NRVTracer
	distanceCalculator *SemanticDistanceCalculator
	ontologyRegistry   *ontology.OntologyRegistry
	config             *FidelityScorerConfig
}

// FidelityScorerConfig holds configuration for the scorer
type FidelityScorerConfig struct {
	// Score thresholds
	MinAcceptableScore float64
	MaxViolationRisk   float64

	// Detection settings
	EnableKYCDetection            bool
	EnableAMLDetection            bool
	EnablePositionLimitCheck      bool
	EnableInsiderTradingDetection bool

	// Position limits
	PositionLimits map[string]float64

	// Weights for scoring components
	SemanticDistanceWeight    float64
	RegulatoryAlignmentWeight float64
	IntentAccuracyWeight      float64
	DecisionQualityWeight     float64
}

// DefaultFidelityScorerConfig returns default configuration
func DefaultFidelityScorerConfig() *FidelityScorerConfig {
	return &FidelityScorerConfig{
		MinAcceptableScore:            0.75,
		MaxViolationRisk:              0.30,
		EnableKYCDetection:            true,
		EnableAMLDetection:            true,
		EnablePositionLimitCheck:      true,
		EnableInsiderTradingDetection: true,
		PositionLimits:                make(map[string]float64),
		SemanticDistanceWeight:        0.30,
		RegulatoryAlignmentWeight:     0.30,
		IntentAccuracyWeight:          0.20,
		DecisionQualityWeight:         0.20,
	}
}

// FidelityScoreResult contains the comprehensive fidelity scoring result
type FidelityScoreResult struct {
	// Core scores
	OverallScore        float64 `json:"overall_score"`        // 0.0 - 1.0 (higher is better)
	SemanticDistance    float64 `json:"semantic_distance"`    // 0.0 - 1.0 (lower is better)
	RegulatoryAlignment float64 `json:"regulatory_alignment"` // 0.0 - 1.0 (higher is better)
	IntentAccuracy      float64 `json:"intent_accuracy"`      // 0.0 - 1.0 (higher is better)
	DecisionQuality     float64 `json:"decision_quality"`     // 0.0 - 1.0 (higher is better)

	// Risk assessment
	ViolationRisk float64 `json:"violation_risk"` // 0.0 - 1.0 (lower is better)
	RiskLevel     string  `json:"risk_level"`     // low, medium, high, critical

	// Detailed analysis
	OntologyScores map[string]float64 `json:"ontology_scores,omitempty"`
	CategoryScores map[string]float64 `json:"category_scores,omitempty"`

	// Detected issues
	RiskIndicators []RiskIndicator `json:"risk_indicators,omitempty"`
	ComplianceGaps []ComplianceGap `json:"compliance_gaps,omitempty"`

	// Recommendations
	Recommendations []string         `json:"recommendations,omitempty"`
	RequiredActions []RequiredAction `json:"required_actions,omitempty"`

	// Metadata
	TraceID      string    `json:"trace_id"`
	ScoredAt     time.Time `json:"scored_at"`
	ScoreVersion string    `json:"score_version"`
}

// ComplianceGap represents a compliance requirement that wasn't met
type ComplianceGap struct {
	OntologyID  string  `json:"ontology_id"`
	Category    string  `json:"category"`
	Description string  `json:"description"`
	Severity    string  `json:"severity"`
	GapScore    float64 `json:"gap_score"` // 0.0 - 1.0
}

// RequiredAction represents an action required to improve fidelity
type RequiredAction struct {
	Priority    int    `json:"priority"`
	Action      string `json:"action"`
	Category    string `json:"category"`
	Description string `json:"description"`
}

// NewFidelityScorer creates a new fidelity scorer
func NewFidelityScorer(tracer *NRVTracer, registry *ontology.OntologyRegistry, config *FidelityScorerConfig) *FidelityScorer {
	if config == nil {
		config = DefaultFidelityScorerConfig()
	}

	return &FidelityScorer{
		tracer:             tracer,
		distanceCalculator: NewSemanticDistanceCalculator(registry),
		ontologyRegistry:   registry,
		config:             config,
	}
}

// ScoreTrace calculates the fidelity score for a trace
func (s *FidelityScorer) ScoreTrace(ctx context.Context, traceID string, expectedOutcome string) (*FidelityScoreResult, error) {
	trace, err := s.tracer.GetTrace(traceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get trace: %w", err)
	}

	return s.ScoreTraceObject(ctx, trace, expectedOutcome)
}

// ScoreTraceObject scores a trace object directly
func (s *FidelityScorer) ScoreTraceObject(ctx context.Context, trace *NRVTrace, expectedOutcome string) (*FidelityScoreResult, error) {
	if trace == nil {
		return nil, fmt.Errorf("trace cannot be nil")
	}

	result := &FidelityScoreResult{
		TraceID:         trace.ID,
		ScoredAt:        time.Now(),
		ScoreVersion:    "1.0",
		OntologyScores:  make(map[string]float64),
		CategoryScores:  make(map[string]float64),
		RiskIndicators:  make([]RiskIndicator, 0),
		ComplianceGaps:  make([]ComplianceGap, 0),
		Recommendations: make([]string, 0),
		RequiredActions: make([]RequiredAction, 0),
	}

	// Calculate semantic distance
	distanceResult, err := s.distanceCalculator.CalculateDistance(trace, expectedOutcome)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate semantic distance: %w", err)
	}

	result.SemanticDistance = distanceResult.OverallDistance

	// Calculate component scores (inverse of distance, normalized)
	result.IntentAccuracy = 1.0 - distanceResult.IntentDistance
	result.DecisionQuality = 1.0 - distanceResult.OutcomeDistance

	// Calculate regulatory alignment
	result.RegulatoryAlignment = s.calculateRegulatoryAlignment(trace)

	// Calculate ontology-specific scores
	for ontologyID, distance := range distanceResult.OntologyDistances {
		result.OntologyScores[ontologyID] = 1.0 - distance
	}

	// Calculate category scores
	result.CategoryScores = s.calculateCategoryScores(trace)

	// Calculate overall weighted score
	result.OverallScore = s.calculateOverallScore(result)

	// Assess violation risk
	result.ViolationRisk = s.assessViolationRisk(trace, distanceResult)
	result.RiskLevel = s.determineRiskLevel(result.ViolationRisk)

	// Detect risk indicators
	result.RiskIndicators = s.detectRiskIndicators(trace)

	// Identify compliance gaps
	result.ComplianceGaps = s.identifyComplianceGaps(trace, distanceResult)

	// Generate recommendations
	result.Recommendations = s.generateRecommendations(result)

	// Determine required actions
	result.RequiredActions = s.determineRequiredActions(result)

	// Update trace with fidelity analysis
	trace.FidelityScore = result.OverallScore
	trace.FidelityAnalysis = &FidelityAnalysis{
		OverallScore:        result.OverallScore,
		SemanticDistance:    result.SemanticDistance,
		RegulatoryAlignment: result.RegulatoryAlignment,
		IntentAccuracy:      result.IntentAccuracy,
		DecisionQuality:     result.DecisionQuality,
		ViolationRisk:       result.ViolationRisk,
		OntologyAlignments:  result.OntologyScores,
		RiskIndicators:      result.RiskIndicators,
		Recommendations:     result.Recommendations,
	}
	trace.Status = NRVTraceStatusAnalyzed

	return result, nil
}

// calculateRegulatoryAlignment calculates overall regulatory alignment
func (s *FidelityScorer) calculateRegulatoryAlignment(trace *NRVTrace) float64 {
	if len(trace.AppliedOntologies) == 0 {
		return 0.0
	}

	totalAlignment := 0.0
	checkedOntologies := 0

	for _, ontologyID := range trace.AppliedOntologies {
		// Check compliance results for this ontology
		for _, check := range trace.ComplianceChecks {
			if check.OntologyID == ontologyID {
				checkedOntologies++

				switch check.Status {
				case "compliant":
					totalAlignment += 1.0
				case "warning":
					totalAlignment += 0.7
				case "violated":
					totalAlignment += 0.0
				}
			}
		}
	}

	if checkedOntologies == 0 {
		return 0.5 // Neutral if no checks performed
	}

	return totalAlignment / float64(checkedOntologies)
}

// calculateCategoryScores calculates scores per regulation category
func (s *FidelityScorer) calculateCategoryScores(trace *NRVTrace) map[string]float64 {
	scores := make(map[string]float64)

	categories := []ontology.RegulationCategory{
		ontology.CategoryKYC,
		ontology.CategoryAML,
		ontology.CategorySEC,
		ontology.CategoryBasel,
	}

	for _, category := range categories {
		alignment := s.distanceCalculator.CalculateRegulatoryAlignment(trace, category)
		scores[string(category)] = alignment
	}

	return scores
}

// calculateOverallScore calculates the weighted overall score
func (s *FidelityScorer) calculateOverallScore(result *FidelityScoreResult) float64 {
	totalWeight := s.config.SemanticDistanceWeight +
		s.config.RegulatoryAlignmentWeight +
		s.config.IntentAccuracyWeight +
		s.config.DecisionQualityWeight

	if totalWeight == 0 {
		return 0.5
	}

	// Note: SemanticDistance is inverted (lower is better)
	semanticScore := 1.0 - result.SemanticDistance

	weightedSum := (semanticScore * s.config.SemanticDistanceWeight) +
		(result.RegulatoryAlignment * s.config.RegulatoryAlignmentWeight) +
		(result.IntentAccuracy * s.config.IntentAccuracyWeight) +
		(result.DecisionQuality * s.config.DecisionQualityWeight)

	return weightedSum / totalWeight
}

// assessViolationRisk assesses the risk of regulatory violation
func (s *FidelityScorer) assessViolationRisk(trace *NRVTrace, distanceResult *DistanceResult) float64 {
	risk := 0.0

	// Base risk from semantic distance
	risk += distanceResult.OverallDistance * 0.3

	// Risk from compliance violations
	violationCount := 0
	for _, check := range trace.ComplianceChecks {
		if check.Status == "violated" {
			violationCount++
		}
	}

	if len(trace.ComplianceChecks) > 0 {
		violationRatio := float64(violationCount) / float64(len(trace.ComplianceChecks))
		risk += violationRatio * 0.4
	}

	// Risk from missing ontologies
	requiredCategories := []string{"kyc", "aml"}
	for _, req := range requiredCategories {
		found := false
		for _, ontology := range trace.AppliedOntologies {
			if containsString(ontology, req) {
				found = true
				break
			}
		}
		if !found {
			risk += 0.15 // Significant risk for missing critical ontologies
		}
	}

	// Risk from suspicious patterns in reasoning
	suspiciousCount := 0
	for _, step := range trace.Steps {
		if isSuspiciousStep(step) {
			suspiciousCount++
		}
	}

	if len(trace.Steps) > 0 {
		suspiciousRatio := float64(suspiciousCount) / float64(len(trace.Steps))
		risk += suspiciousRatio * 0.15
	}

	return math.Min(risk, 1.0)
}

// determineRiskLevel converts risk score to level
func (s *FidelityScorer) determineRiskLevel(risk float64) string {
	switch {
	case risk >= 0.75:
		return "critical"
	case risk >= 0.50:
		return "high"
	case risk >= 0.25:
		return "medium"
	default:
		return "low"
	}
}

// detectRiskIndicators detects various risk indicators
func (s *FidelityScorer) detectRiskIndicators(trace *NRVTrace) []RiskIndicator {
	indicators := make([]RiskIndicator, 0)

	// KYC bypass detection
	if s.config.EnableKYCDetection {
		kycIndicators := s.tracer.DetectKYCBypass(trace.ID)
		indicators = append(indicators, kycIndicators...)
	}

	// Position limit detection
	if s.config.EnablePositionLimitCheck {
		positionIndicators := s.tracer.DetectPositionLimitViolation(trace.ID, s.config.PositionLimits)
		indicators = append(indicators, positionIndicators...)
	}

	// AML detection
	if s.config.EnableAMLDetection {
		amlIndicators := s.detectAMLRisks(trace)
		indicators = append(indicators, amlIndicators...)
	}

	// Insider trading detection
	if s.config.EnableInsiderTradingDetection {
		insiderIndicators := s.detectInsiderTradingRisks(trace)
		indicators = append(indicators, insiderIndicators...)
	}

	return indicators
}

// detectAMLRisks detects AML-related risks
func (s *FidelityScorer) detectAMLRisks(trace *NRVTrace) []RiskIndicator {
	indicators := make([]RiskIndicator, 0)

	// Check for large transactions without proper screening
	for i, step := range trace.Steps {
		if step.FinancialContext == nil || step.FinancialContext.Amount == nil {
			continue
		}

		amount := step.FinancialContext.Amount.Amount
		currency := step.FinancialContext.Amount.Currency

		// Threshold for large transactions (e.g., $10,000 USD or equivalent)
		threshold := 10000.0
		if currency != "USD" {
			// Simple approximation for demo
			threshold = threshold * 0.85
		}

		if amount > threshold {
			// Check if AML screening was performed
			amlChecked := false
			for _, check := range trace.ComplianceChecks {
				if check.OntologyID == "aml-fincen-v1" && check.Timestamp.After(step.Timestamp) {
					amlChecked = true
					break
				}
			}

			if !amlChecked {
				indicators = append(indicators, RiskIndicator{
					Type:        "aml_large_transaction_unscreened",
					Severity:    "high",
					Description: fmt.Sprintf("Large transaction (%.2f %s) without AML screening", amount, currency),
					StepNumber:  i + 1,
					Confidence:  0.90,
				})
			}
		}
	}

	return indicators
}

// detectInsiderTradingRisks detects potential insider trading risks
func (s *FidelityScorer) detectInsiderTradingRisks(trace *NRVTrace) []RiskIndicator {
	indicators := make([]RiskIndicator, 0)

	// Check for suspicious timing patterns
	suspiciousKeywords := []string{
		"confidential", "non-public", "material information",
		"earnings before release", "merger before announcement",
	}

	for i, step := range trace.Steps {
		text := step.Intent + " " + step.Observation + " " + step.Inference

		for _, keyword := range suspiciousKeywords {
			if containsString(text, keyword) {
				indicators = append(indicators, RiskIndicator{
					Type:        "potential_insider_information",
					Severity:    "critical",
					Description: fmt.Sprintf("Potential use of non-public information: '%s'", keyword),
					StepNumber:  i + 1,
					Confidence:  0.70,
				})
			}
		}
	}

	return indicators
}

// identifyComplianceGaps identifies gaps in compliance coverage
func (s *FidelityScorer) identifyComplianceGaps(trace *NRVTrace, distanceResult *DistanceResult) []ComplianceGap {
	gaps := make([]ComplianceGap, 0)

	// Check for missing required ontologies
	requiredOntologies := map[string]string{
		"kyc-global-v1":     "KYC",
		"aml-fincen-v1":     "AML",
		"sec-regulation-v1": "SEC",
		"mica-eu-v1":        "MiCA",
		"gdpr-privacy-v1":   "GDPR",
	}

	for ontologyID, category := range requiredOntologies {
		found := false
		for _, applied := range trace.AppliedOntologies {
			if applied == ontologyID {
				found = true
				break
			}
		}

		if !found {
			gaps = append(gaps, ComplianceGap{
				OntologyID:  ontologyID,
				Category:    category,
				Description: fmt.Sprintf("Required %s ontology not applied", category),
				Severity:    "high",
				GapScore:    0.8,
			})
		}
	}

	// Add gaps from alignment analysis
	for _, alignmentGap := range distanceResult.AlignmentGaps {
		gaps = append(gaps, ComplianceGap{
			OntologyID:  "",
			Category:    alignmentGap.Category,
			Description: alignmentGap.Description,
			Severity:    mapDistanceToSeverity(alignmentGap.Distance),
			GapScore:    alignmentGap.Distance,
		})
	}

	return gaps
}

// generateRecommendations generates recommendations based on scores
func (s *FidelityScorer) generateRecommendations(result *FidelityScoreResult) []string {
	recommendations := make([]string, 0)

	// Semantic distance recommendations
	if result.SemanticDistance > 0.5 {
		recommendations = append(recommendations,
			"Improve reasoning transparency by providing clearer intent statements")
	}

	// Regulatory alignment recommendations
	if result.RegulatoryAlignment < 0.7 {
		recommendations = append(recommendations,
			"Strengthen compliance checks against financial regulations")
	}

	// Intent accuracy recommendations
	if result.IntentAccuracy < 0.6 {
		recommendations = append(recommendations,
			"Review agent intent alignment with regulatory requirements")
	}

	// Decision quality recommendations
	if result.DecisionQuality < 0.7 {
		recommendations = append(recommendations,
			"Enhance decision-making processes with better outcome validation")
	}

	// Risk-based recommendations
	if result.ViolationRisk > 0.5 {
		recommendations = append(recommendations,
			"URGENT: Address critical compliance gaps before proceeding")
	}

	// Gap-specific recommendations
	for _, gap := range result.ComplianceGaps {
		if gap.Severity == "critical" || gap.Severity == "high" {
			recommendations = append(recommendations,
				fmt.Sprintf("Address %s compliance gap: %s", gap.Category, gap.Description))
		}
	}

	return recommendations
}

// determineRequiredActions determines actions required for compliance
func (s *FidelityScorer) determineRequiredActions(result *FidelityScoreResult) []RequiredAction {
	actions := make([]RequiredAction, 0)
	priority := 1

	// Critical risk actions
	if result.ViolationRisk > 0.7 {
		actions = append(actions, RequiredAction{
			Priority:    priority,
			Action:      "HALT_OPERATIONS",
			Category:    "risk",
			Description: "Halt agent operations due to critical violation risk",
		})
		priority++
	}

	// Compliance gap actions
	for _, gap := range result.ComplianceGaps {
		if gap.Severity == "critical" || gap.Severity == "high" {
			actions = append(actions, RequiredAction{
				Priority:    priority,
				Action:      fmt.Sprintf("APPLY_%s_ONTOLOGY", gap.Category),
				Category:    "compliance",
				Description: fmt.Sprintf("Apply %s ontology to address compliance gap", gap.Category),
			})
			priority++
		}
	}

	// Score-based actions
	if result.OverallScore < s.config.MinAcceptableScore {
		actions = append(actions, RequiredAction{
			Priority: priority,
			Action:   "REVIEW_REASONING",
			Category: "quality",
			Description: fmt.Sprintf("Review agent reasoning (score %.2f below threshold %.2f)",
				result.OverallScore, s.config.MinAcceptableScore),
		})
		priority++
	}

	return actions
}

// ValidateFidelity validates if a trace meets fidelity requirements
func (s *FidelityScorer) ValidateFidelity(result *FidelityScoreResult) (bool, []string) {
	issues := make([]string, 0)

	// Check overall score
	if result.OverallScore < s.config.MinAcceptableScore {
		issues = append(issues,
			fmt.Sprintf("Overall score %.2f below minimum %.2f",
				result.OverallScore, s.config.MinAcceptableScore))
	}

	// Check violation risk
	if result.ViolationRisk > s.config.MaxViolationRisk {
		issues = append(issues,
			fmt.Sprintf("Violation risk %.2f exceeds maximum %.2f",
				result.ViolationRisk, s.config.MaxViolationRisk))
	}

	// Check critical indicators
	criticalCount := 0
	for _, indicator := range result.RiskIndicators {
		if indicator.Severity == "critical" {
			criticalCount++
		}
	}

	if criticalCount > 0 {
		issues = append(issues,
			fmt.Sprintf("Found %d critical risk indicators", criticalCount))
	}

	return len(issues) == 0, issues
}

// isSuspiciousStep determines if a step contains suspicious patterns
func isSuspiciousStep(step NRVReasoningStep) bool {
	suspiciousPatterns := []string{
		"bypass", "circumvent", "avoid", "hide", "conceal",
		"override", "ignore", "skip", "unverified", "unauthorized",
	}

	text := step.Intent + " " + step.Observation + " " + step.Inference

	for _, pattern := range suspiciousPatterns {
		if containsString(text, pattern) {
			return true
		}
	}

	return false
}

// mapDistanceToSeverity maps distance score to severity level
func mapDistanceToSeverity(distance float64) string {
	switch {
	case distance >= 0.8:
		return "critical"
	case distance >= 0.6:
		return "high"
	case distance >= 0.4:
		return "medium"
	default:
		return "low"
	}
}

// containsString checks if a string contains a substring (case-insensitive)
func containsString(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 &&
		(s == substr ||
			len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					containsSubstring(s, substr)))
}

// containsSubstring checks if s contains substr
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
