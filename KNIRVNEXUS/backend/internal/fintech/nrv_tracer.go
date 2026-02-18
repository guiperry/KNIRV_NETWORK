// Copyright 2026 KNIRV-NEXUS
// SPDX-License-Identifier: GPL-3.0-or-later

package fintech

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"backend_server/internal/fintech/ontology"
	"backend_server/internal/reasoning/graph"
)

// NRVTraceStatus represents the status of an NRV trace
type NRVTraceStatus string

const (
	NRVTraceStatusCapturing NRVTraceStatus = "CAPTURING"
	NRVTraceStatusComplete  NRVTraceStatus = "COMPLETE"
	NRVTraceStatusFailed    NRVTraceStatus = "FAILED"
	NRVTraceStatusAnalyzed  NRVTraceStatus = "ANALYZED"
)

// NRVReasoningStep represents a single step in NRV-based reasoning
type NRVReasoningStep struct {
	StepNumber     int                    `json:"step_number"`
	Timestamp      time.Time              `json:"timestamp"`
	StepType       string                 `json:"step_type"` // inference, query, validation, decision
	Intent         string                 `json:"intent"`
	Observation    string                 `json:"observation"`
	Inference      string                 `json:"inference"`
	Action         string                 `json:"action,omitempty"`
	Confidence     float64                `json:"confidence"`
	OntologyRefs   []string               `json:"ontology_refs,omitempty"`
	RegulatoryRefs []string               `json:"regulatory_refs,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`

	// Financial context
	FinancialContext *FinancialContext `json:"financial_context,omitempty"`
}

// FinancialContext holds financial-specific context for reasoning
type FinancialContext struct {
	ActionType       ontology.ActionType           `json:"action_type"`
	Instrument       *ontology.FinancialInstrument `json:"instrument,omitempty"`
	Amount           *ontology.MonetaryValue       `json:"amount,omitempty"`
	Counterparty     *ontology.CounterpartyInfo    `json:"counterparty,omitempty"`
	RegulationChecks []string                      `json:"regulation_checks,omitempty"`
}

// NRVTrace represents a complete reasoning trace with financial ontologies
type NRVTrace struct {
	// Metadata
	ID        string         `json:"id"`
	Version   string         `json:"version"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	Status    NRVTraceStatus `json:"status"`

	// Context
	AgentID        string `json:"agent_id"`
	AgentName      string `json:"agent_name"`
	ValidationID   string `json:"validation_id"`
	EvidencePackID string `json:"evidence_pack_id,omitempty"`

	// Reasoning
	Steps         []NRVReasoningStep `json:"steps"`
	StepCount     int                `json:"step_count"`
	FinalDecision string             `json:"final_decision,omitempty"`

	// Ontology integration
	AppliedOntologies []string             `json:"applied_ontologies,omitempty"`
	ComplianceChecks  []ComplianceCheckRef `json:"compliance_checks,omitempty"`

	// Fidelity scoring
	FidelityScore    float64           `json:"fidelity_score,omitempty"`
	FidelityAnalysis *FidelityAnalysis `json:"fidelity_analysis,omitempty"`

	// Semantic context
	SemanticContext   map[string]interface{} `json:"semantic_context,omitempty"`
	RegulatoryContext *RegulatoryContext     `json:"regulatory_context,omitempty"`
}

// ComplianceCheckRef references a compliance check performed during reasoning
type ComplianceCheckRef struct {
	CheckID        string    `json:"check_id"`
	OntologyID     string    `json:"ontology_id"`
	RegulationName string    `json:"regulation_name"`
	Status         string    `json:"status"`
	Timestamp      time.Time `json:"timestamp"`
}

// RegulatoryContext holds regulatory-specific context
type RegulatoryContext struct {
	ApplicableRegulations []string `json:"applicable_regulations"`
	RiskLevel             string   `json:"risk_level"`
	Jurisdiction          string   `json:"jurisdiction,omitempty"`
	SanctionsScreened     bool     `json:"sanctions_screened"`
	PEPChecked            bool     `json:"pep_checked"`
}

// FidelityAnalysis contains detailed fidelity scoring results
type FidelityAnalysis struct {
	OverallScore        float64 `json:"overall_score"`
	SemanticDistance    float64 `json:"semantic_distance"` // 0.0 = perfect alignment, 1.0 = complete misalignment
	RegulatoryAlignment float64 `json:"regulatory_alignment"`
	IntentAccuracy      float64 `json:"intent_accuracy"`
	DecisionQuality     float64 `json:"decision_quality"`
	ViolationRisk       float64 `json:"violation_risk"` // 0.0 = no risk, 1.0 = certain violation

	// Detailed breakdown
	OntologyAlignments map[string]float64 `json:"ontology_alignments,omitempty"`
	RiskIndicators     []RiskIndicator    `json:"risk_indicators,omitempty"`
	Recommendations    []string           `json:"recommendations,omitempty"`
}

// RiskIndicator represents a specific risk identified in reasoning
type RiskIndicator struct {
	Type        string  `json:"type"` // kyc_bypass, position_limit, aml_gap, etc.
	Severity    string  `json:"severity"`
	Description string  `json:"description"`
	StepNumber  int     `json:"step_number,omitempty"`
	Confidence  float64 `json:"confidence"`
}

// NRVTracer captures and manages NRV-based reasoning traces
type NRVTracer struct {
	activeTraces     map[string]*NRVTrace
	completedTraces  map[string]*NRVTrace
	ontologyRegistry *ontology.OntologyRegistry
}

// NewNRVTracer creates a new NRV tracer
func NewNRVTracer(registry *ontology.OntologyRegistry) *NRVTracer {
	return &NRVTracer{
		activeTraces:     make(map[string]*NRVTrace),
		completedTraces:  make(map[string]*NRVTrace),
		ontologyRegistry: registry,
	}
}

// StartTrace begins capturing a new NRV trace
func (t *NRVTracer) StartTrace(agentID, agentName, validationID string) *NRVTrace {
	now := time.Now()
	trace := &NRVTrace{
		ID:                fmt.Sprintf("nrv-%d", now.UnixNano()),
		Version:           "1.0",
		CreatedAt:         now,
		UpdatedAt:         now,
		Status:            NRVTraceStatusCapturing,
		AgentID:           agentID,
		AgentName:         agentName,
		ValidationID:      validationID,
		Steps:             make([]NRVReasoningStep, 0),
		AppliedOntologies: make([]string, 0),
		ComplianceChecks:  make([]ComplianceCheckRef, 0),
		SemanticContext:   make(map[string]interface{}),
	}

	t.activeTraces[trace.ID] = trace
	return trace
}

// AddStep adds a reasoning step to the trace
func (t *NRVTracer) AddStep(traceID string, step NRVReasoningStep) error {
	trace, exists := t.activeTraces[traceID]
	if !exists {
		return fmt.Errorf("trace not found: %s", traceID)
	}

	step.StepNumber = len(trace.Steps) + 1
	step.Timestamp = time.Now()

	trace.Steps = append(trace.Steps, step)
	trace.StepCount = len(trace.Steps)
	trace.UpdatedAt = time.Now()

	return nil
}

// RecordInference records an inference step with financial context
func (t *NRVTracer) RecordInference(traceID string, intent, observation, inference string, confidence float64, financialContext *FinancialContext) error {
	step := NRVReasoningStep{
		StepType:         "inference",
		Intent:           intent,
		Observation:      observation,
		Inference:        inference,
		Confidence:       confidence,
		FinancialContext: financialContext,
	}

	return t.AddStep(traceID, step)
}

// RecordValidation records a validation step against ontologies
func (t *NRVTracer) RecordValidation(traceID string, action *ontology.FinancialAction, result *ontology.ValidationResult) error {
	step := NRVReasoningStep{
		StepType:     "validation",
		Intent:       fmt.Sprintf("Validate %s action", action.Type),
		Observation:  fmt.Sprintf("Action: %s", action.Intent),
		Inference:    fmt.Sprintf("Compliance check against %s", result.OntologyName),
		Confidence:   result.Score,
		OntologyRefs: []string{result.OntologyID},
	}

	if action != nil {
		step.FinancialContext = &FinancialContext{
			ActionType:   action.Type,
			Instrument:   action.Instrument,
			Amount:       action.Amount,
			Counterparty: action.Counterparty,
		}
	}

	if err := t.AddStep(traceID, step); err != nil {
		return err
	}

	// Track compliance check
	trace := t.activeTraces[traceID]
	checkRef := ComplianceCheckRef{
		CheckID:        fmt.Sprintf("check-%d", len(trace.ComplianceChecks)),
		OntologyID:     result.OntologyID,
		RegulationName: result.OntologyName,
		Status:         string(result.OverallStatus),
		Timestamp:      time.Now(),
	}
	trace.ComplianceChecks = append(trace.ComplianceChecks, checkRef)

	// Track applied ontology
	if !contains(trace.AppliedOntologies, result.OntologyID) {
		trace.AppliedOntologies = append(trace.AppliedOntologies, result.OntologyID)
	}

	return nil
}

// RecordDecision records a final decision step
func (t *NRVTracer) RecordDecision(traceID, decision string, confidence float64, metadata map[string]interface{}) error {
	step := NRVReasoningStep{
		StepType:   "decision",
		Intent:     "Final decision",
		Inference:  decision,
		Confidence: confidence,
		Metadata:   metadata,
	}

	if err := t.AddStep(traceID, step); err != nil {
		return err
	}

	trace := t.activeTraces[traceID]
	trace.FinalDecision = decision

	return nil
}

// DetectKYCBypass attempts to detect KYC bypass attempts in reasoning
func (t *NRVTracer) DetectKYCBypass(traceID string) []RiskIndicator {
	trace, exists := t.activeTraces[traceID]
	if !exists {
		return nil
	}

	indicators := make([]RiskIndicator, 0)

	kycKeywords := []string{
		"skip kyc", "bypass kyc", "ignore kyc", "fast track",
		"unverified", "skip verification", "auto-approve",
		"override compliance", "circumvent",
	}

	for _, step := range trace.Steps {
		combinedText := strings.ToLower(step.Intent + " " + step.Observation + " " + step.Inference)

		for _, keyword := range kycKeywords {
			if strings.Contains(combinedText, keyword) {
				indicators = append(indicators, RiskIndicator{
					Type:        "kyc_bypass",
					Severity:    "critical",
					Description: fmt.Sprintf("Potential KYC bypass detected: '%s'", keyword),
					StepNumber:  step.StepNumber,
					Confidence:  0.85,
				})
			}
		}

		// Check for missing counterparty info when required
		if step.FinancialContext != nil && step.FinancialContext.Counterparty == nil {
			if step.FinancialContext.ActionType == ontology.ActionTransfer ||
				step.FinancialContext.ActionType == ontology.ActionTrade {
				// Check if any KYC ontology was applied
				hasKYC := false
				for _, ref := range step.OntologyRefs {
					if strings.Contains(strings.ToLower(ref), "kyc") {
						hasKYC = true
						break
					}
				}

				if !hasKYC {
					indicators = append(indicators, RiskIndicator{
						Type:        "missing_kyc",
						Severity:    "high",
						Description: "Counterparty transfer without KYC verification",
						StepNumber:  step.StepNumber,
						Confidence:  0.75,
					})
				}
			}
		}
	}

	return indicators
}

// DetectPositionLimitViolation checks for position limit violations
func (t *NRVTracer) DetectPositionLimitViolation(traceID string, limits map[string]float64) []RiskIndicator {
	trace, exists := t.activeTraces[traceID]
	if !exists {
		return nil
	}

	indicators := make([]RiskIndicator, 0)

	// Track positions by instrument
	positions := make(map[string]float64)

	for _, step := range trace.Steps {
		if step.FinancialContext != nil && step.FinancialContext.Instrument != nil {
			symbol := step.FinancialContext.Instrument.Symbol

			if step.FinancialContext.Amount != nil {
				amount := step.FinancialContext.Amount.Amount
				positions[symbol] += amount

				// Check against limits
				if limit, exists := limits[symbol]; exists {
					if positions[symbol] > limit {
						indicators = append(indicators, RiskIndicator{
							Type:     "position_limit_violation",
							Severity: "critical",
							Description: fmt.Sprintf("Position limit exceeded for %s: %.2f > %.2f",
								symbol, positions[symbol], limit),
							StepNumber: step.StepNumber,
							Confidence: 0.95,
						})
					}
				}
			}
		}
	}

	return indicators
}

// FinalizeTrace completes the trace and moves it to completed traces
func (t *NRVTracer) FinalizeTrace(traceID string) (*NRVTrace, error) {
	trace, exists := t.activeTraces[traceID]
	if !exists {
		return nil, fmt.Errorf("trace not found: %s", traceID)
	}

	trace.Status = NRVTraceStatusComplete
	trace.UpdatedAt = time.Now()

	delete(t.activeTraces, traceID)
	t.completedTraces[traceID] = trace

	return trace, nil
}

// GetTrace retrieves a trace by ID
func (t *NRVTracer) GetTrace(traceID string) (*NRVTrace, error) {
	if trace, exists := t.activeTraces[traceID]; exists {
		return trace, nil
	}

	if trace, exists := t.completedTraces[traceID]; exists {
		return trace, nil
	}

	return nil, fmt.Errorf("trace not found: %s", traceID)
}

// ListActiveTraces returns all active traces
func (t *NRVTracer) ListActiveTraces() []*NRVTrace {
	traces := make([]*NRVTrace, 0, len(t.activeTraces))
	for _, trace := range t.activeTraces {
		traces = append(traces, trace)
	}
	return traces
}

// ListCompletedTraces returns all completed traces
func (t *NRVTracer) ListCompletedTraces() []*NRVTrace {
	traces := make([]*NRVTrace, 0, len(t.completedTraces))
	for _, trace := range t.completedTraces {
		traces = append(traces, trace)
	}
	return traces
}

// ConvertToEvidence converts the NRV trace to NRVTraceEvidence for evidence packs
func (trace *NRVTrace) ConvertToEvidence() *NRVTraceEvidence {
	evidence := &NRVTraceEvidence{
		TraceID:         trace.ID,
		AgentID:         trace.AgentID,
		FidelityScore:   trace.FidelityScore,
		OntologyRefs:    trace.AppliedOntologies,
		ReasoningSteps:  make([]ReasoningStep, 0, len(trace.Steps)),
		SemanticContext: trace.SemanticContext,
	}

	for _, step := range trace.Steps {
		evidenceStep := ReasoningStep{
			StepNumber:  step.StepNumber,
			Timestamp:   step.Timestamp,
			Intent:      step.Intent,
			Observation: step.Observation,
			Action:      step.Action,
			Confidence:  step.Confidence,
			Metadata:    step.Metadata,
		}
		evidence.ReasoningSteps = append(evidence.ReasoningSteps, evidenceStep)
	}

	return evidence
}

// ToMarkdown exports the trace as Markdown
func (trace *NRVTrace) ToMarkdown() ([]byte, error) {
	content := fmt.Sprintf(`# NRV Reasoning Trace: %s

## Metadata

- **Trace ID**: %s
- **Agent ID**: %s
- **Agent Name**: %s
- **Validation ID**: %s
- **Status**: %s
- **Created**: %s
- **Steps**: %d
- **Fidelity Score**: %.2f

## Applied Ontologies

`,
		trace.ID,
		trace.ID,
		trace.AgentID,
		trace.AgentName,
		trace.ValidationID,
		trace.Status,
		trace.CreatedAt.Format(time.RFC3339),
		trace.StepCount,
		trace.FidelityScore,
	)

	for _, ontology := range trace.AppliedOntologies {
		content += fmt.Sprintf("- %s\n", ontology)
	}

	// Compliance checks
	if len(trace.ComplianceChecks) > 0 {
		content += "\n## Compliance Checks\n\n"
		content += "| Check ID | Ontology | Regulation | Status |\n"
		content += "|----------|----------|------------|--------|\n"

		for _, check := range trace.ComplianceChecks {
			content += fmt.Sprintf("| %s | %s | %s | %s |\n",
				check.CheckID, check.OntologyID, check.RegulationName, check.Status)
		}
	}

	// Reasoning steps
	content += "\n## Reasoning Steps\n\n"

	for _, step := range trace.Steps {
		content += fmt.Sprintf("### Step %d: %s\n\n", step.StepNumber, step.StepType)
		content += fmt.Sprintf("- **Intent**: %s\n", step.Intent)
		content += fmt.Sprintf("- **Observation**: %s\n", step.Observation)
		if step.Inference != "" {
			content += fmt.Sprintf("- **Inference**: %s\n", step.Inference)
		}
		if step.Action != "" {
			content += fmt.Sprintf("- **Action**: %s\n", step.Action)
		}
		content += fmt.Sprintf("- **Confidence**: %.2f%%\n", step.Confidence*100)

		if len(step.OntologyRefs) > 0 {
			content += fmt.Sprintf("- **Ontologies**: %s\n", strings.Join(step.OntologyRefs, ", "))
		}

		if step.FinancialContext != nil {
			content += "\n**Financial Context**:\n"
			content += fmt.Sprintf("- Action Type: %s\n", step.FinancialContext.ActionType)
			if step.FinancialContext.Instrument != nil {
				content += fmt.Sprintf("- Instrument: %s (%s)\n",
					step.FinancialContext.Instrument.Symbol,
					step.FinancialContext.Instrument.Type)
			}
			if step.FinancialContext.Amount != nil {
				content += fmt.Sprintf("- Amount: %.2f %s\n",
					step.FinancialContext.Amount.Amount,
					step.FinancialContext.Amount.Currency)
			}
		}

		content += "\n"
	}

	// Final decision
	if trace.FinalDecision != "" {
		content += fmt.Sprintf("\n## Final Decision\n\n%s\n", trace.FinalDecision)
	}

	// Fidelity analysis
	if trace.FidelityAnalysis != nil {
		content += "\n## Fidelity Analysis\n\n"
		content += fmt.Sprintf("- **Overall Score**: %.2f\n", trace.FidelityAnalysis.OverallScore)
		content += fmt.Sprintf("- **Semantic Distance**: %.2f\n", trace.FidelityAnalysis.SemanticDistance)
		content += fmt.Sprintf("- **Regulatory Alignment**: %.2f\n", trace.FidelityAnalysis.RegulatoryAlignment)
		content += fmt.Sprintf("- **Intent Accuracy**: %.2f\n", trace.FidelityAnalysis.IntentAccuracy)
		content += fmt.Sprintf("- **Decision Quality**: %.2f\n", trace.FidelityAnalysis.DecisionQuality)
		content += fmt.Sprintf("- **Violation Risk**: %.2f\n", trace.FidelityAnalysis.ViolationRisk)

		if len(trace.FidelityAnalysis.RiskIndicators) > 0 {
			content += "\n### Risk Indicators\n\n"
			for _, risk := range trace.FidelityAnalysis.RiskIndicators {
				content += fmt.Sprintf("- **%s** (%s): %s (confidence: %.2f)\n",
					risk.Type, risk.Severity, risk.Description, risk.Confidence)
			}
		}

		if len(trace.FidelityAnalysis.Recommendations) > 0 {
			content += "\n### Recommendations\n\n"
			for _, rec := range trace.FidelityAnalysis.Recommendations {
				content += fmt.Sprintf("- %s\n", rec)
			}
		}
	}

	content += "\n---\n*Generated by KNIRVNEXUS NRV Financial Semantic Engine*\n"

	return []byte(content), nil
}

// ToJSON exports the trace as JSON
func (trace *NRVTrace) ToJSON() ([]byte, error) {
	return json.MarshalIndent(trace, "", "  ")
}

// Helper function
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// GenerateContextRecord generates a ContextRecord for the reasoning graph engine
func (trace *NRVTrace) GenerateContextRecord(errorID string) *graph.ContextRecord {
	steps := make([]string, 0, len(trace.Steps))
	for _, step := range trace.Steps {
		stepDesc := fmt.Sprintf("Step %d [%s]: %s -> %s",
			step.StepNumber, step.StepType, step.Intent, step.Inference)
		steps = append(steps, stepDesc)
	}

	return &graph.ContextRecord{
		ID:        trace.ID,
		ErrorID:   errorID,
		AgentID:   trace.AgentID,
		Trace:     steps,
		Result:    trace.FinalDecision,
		Timestamp: trace.CreatedAt,
	}
}

// NRVTracerConfig holds configuration for the NRV tracer
type NRVTracerConfig struct {
	EnableKYCDetection           bool
	EnablePositionLimitDetection bool
	PositionLimits               map[string]float64
	MaxTraceDuration             time.Duration
	AutoAnalyze                  bool
}

// DefaultNRVTracerConfig returns default configuration
func DefaultNRVTracerConfig() *NRVTracerConfig {
	return &NRVTracerConfig{
		EnableKYCDetection:           true,
		EnablePositionLimitDetection: true,
		PositionLimits:               make(map[string]float64),
		MaxTraceDuration:             5 * time.Minute,
		AutoAnalyze:                  true,
	}
}

// ContextWithTrace adds an NRV trace to a context
func ContextWithTrace(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, "nrv_trace_id", traceID)
}

// TraceFromContext retrieves the trace ID from context
func TraceFromContext(ctx context.Context) (string, bool) {
	traceID, ok := ctx.Value("nrv_trace_id").(string)
	return traceID, ok
}
