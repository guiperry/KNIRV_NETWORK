package ontology

import (
	"context"
	"fmt"
	"time"
)

// AMLOntology implements Anti-Money Laundering regulations
type AMLOntology struct {
	id          string
	name        string
	version     string
	description string
	rules       []Rule
}

// NewAMLOntology creates a new AML ontology
func NewAMLOntology() *AMLOntology {
	return &AMLOntology{
		id:          "aml-1.0",
		name:        "Anti-Money Laundering (AML)",
		version:     "1.0.0",
		description: "Anti-Money Laundering regulations including transaction monitoring, suspicious activity reporting, and customer due diligence",
		rules: []Rule{
			&AMLSuspiciousAmountRule{},
			&AMLHighFrequencyRule{},
			&AMLSanctionedEntityRule{},
			&AMLStructuringRule{},
		},
	}
}

func (o *AMLOntology) GetID() string                   { return o.id }
func (o *AMLOntology) GetName() string                 { return o.name }
func (o *AMLOntology) GetCategory() RegulationCategory { return CategoryAML }
func (o *AMLOntology) GetVersion() string              { return o.version }
func (o *AMLOntology) GetDescription() string          { return o.description }

func (o *AMLOntology) GetRules() []Rule { return o.rules }

func (o *AMLOntology) GetRule(ruleID string) (Rule, error) {
	for _, rule := range o.rules {
		if rule.GetID() == ruleID {
			return rule, nil
		}
	}
	return nil, fmt.Errorf("rule not found: %s", ruleID)
}

func (o *AMLOntology) Validate(ctx context.Context, action *FinancialAction) (*ValidationResult, error) {
	result := &ValidationResult{
		OntologyID:   o.id,
		OntologyName: o.name,
		Category:     CategoryAML,
		RuleResults:  make([]RuleEvaluation, 0),
		Timestamp:    time.Now(),
		Score:        1.0,
	}

	violated := false
	for _, rule := range o.rules {
		evaluation, err := rule.Evaluate(ctx, action)
		if err != nil {
			continue
		}
		result.RuleResults = append(result.RuleResults, *evaluation)

		if evaluation.Status == StatusViolated {
			violated = true
			if evaluation.Severity == SeverityCritical || evaluation.Severity == SeverityHigh {
				result.Score -= 0.25
			} else {
				result.Score -= 0.1
			}
		}
	}

	if result.Score < 0 {
		result.Score = 0
	}

	if violated {
		result.OverallStatus = StatusViolated
	} else {
		result.OverallStatus = StatusCompliant
	}

	return result, nil
}

// AML Suspicious Amount Rule
type AMLSuspiciousAmountRule struct{}

func (r *AMLSuspiciousAmountRule) GetID() string   { return "aml-suspicious-amount" }
func (r *AMLSuspiciousAmountRule) GetName() string { return "Suspicious Transaction Amount" }
func (r *AMLSuspiciousAmountRule) GetDescription() string {
	return "Flags transactions exceeding reporting thresholds"
}
func (r *AMLSuspiciousAmountRule) GetCategory() string        { return "Transaction Monitoring" }
func (r *AMLSuspiciousAmountRule) GetSeverity() SeverityLevel { return SeverityHigh }

func (r *AMLSuspiciousAmountRule) Evaluate(ctx context.Context, action *FinancialAction) (*RuleEvaluation, error) {
	eval := &RuleEvaluation{
		RuleID:      r.GetID(),
		RuleName:    r.GetName(),
		Severity:    r.GetSeverity(),
		Status:      StatusCompliant,
		Description: "Transaction amount within normal range",
	}

	if action.Amount == nil {
		return eval, nil
	}

	// FinCEN reporting threshold: $10,000 USD or equivalent
	threshold := 10000.0
	amount := action.Amount.Amount

	if action.Amount.Currency != "USD" {
		// Simplified: assume 1:1 for demonstration
		// In production, use real FX rates
		amount = action.Amount.Amount
	}

	if amount >= threshold {
		eval.Status = StatusViolated
		eval.Description = fmt.Sprintf("Transaction amount $%.2f exceeds reporting threshold of $%.2f", amount, threshold)
		eval.Details = map[string]interface{}{
			"amount":    amount,
			"threshold": threshold,
			"currency":  action.Amount.Currency,
		}
	}

	return eval, nil
}

// AML High Frequency Rule
type AMLHighFrequencyRule struct{}

func (r *AMLHighFrequencyRule) GetID() string   { return "aml-high-frequency" }
func (r *AMLHighFrequencyRule) GetName() string { return "High Frequency Pattern Detection" }
func (r *AMLHighFrequencyRule) GetDescription() string {
	return "Detects unusually high transaction frequency"
}
func (r *AMLHighFrequencyRule) GetCategory() string        { return "Pattern Detection" }
func (r *AMLHighFrequencyRule) GetSeverity() SeverityLevel { return SeverityMedium }

func (r *AMLHighFrequencyRule) Evaluate(ctx context.Context, action *FinancialAction) (*RuleEvaluation, error) {
	eval := &RuleEvaluation{
		RuleID:      r.GetID(),
		RuleName:    r.GetName(),
		Severity:    r.GetSeverity(),
		Status:      StatusCompliant,
		Description: "Transaction frequency within normal range",
	}

	// Placeholder: would check historical data for frequency patterns
	// For now, assume compliant

	return eval, nil
}

// AML Sanctioned Entity Rule
type AMLSanctionedEntityRule struct{}

func (r *AMLSanctionedEntityRule) GetID() string   { return "aml-sanctioned-entity" }
func (r *AMLSanctionedEntityRule) GetName() string { return "Sanctioned Entity Check" }
func (r *AMLSanctionedEntityRule) GetDescription() string {
	return "Checks if counterparty is on sanctions lists"
}
func (r *AMLSanctionedEntityRule) GetCategory() string        { return "Sanctions Screening" }
func (r *AMLSanctionedEntityRule) GetSeverity() SeverityLevel { return SeverityCritical }

func (r *AMLSanctionedEntityRule) Evaluate(ctx context.Context, action *FinancialAction) (*RuleEvaluation, error) {
	eval := &RuleEvaluation{
		RuleID:      r.GetID(),
		RuleName:    r.GetName(),
		Severity:    r.GetSeverity(),
		Status:      StatusCompliant,
		Description: "Counterparty not on sanctions list",
	}

	if action.Counterparty == nil {
		return eval, nil
	}

	if action.Counterparty.IsSanctioned {
		eval.Status = StatusViolated
		eval.Description = "Counterparty is on sanctions list - transaction prohibited"
		eval.Details = map[string]interface{}{
			"counterparty_id": action.Counterparty.ID,
			"is_sanctioned":   true,
		}
	}

	return eval, nil
}

// AML Structuring Rule (structuring transactions to evade reporting)
type AMLStructuringRule struct{}

func (r *AMLStructuringRule) GetID() string   { return "aml-structuring" }
func (r *AMLStructuringRule) GetName() string { return "Structuring Detection" }
func (r *AMLStructuringRule) GetDescription() string {
	return "Detects potential structuring to evade reporting"
}
func (r *AMLStructuringRule) GetCategory() string        { return "Structuring" }
func (r *AMLStructuringRule) GetSeverity() SeverityLevel { return SeverityHigh }

func (r *AMLStructuringRule) Evaluate(ctx context.Context, action *FinancialAction) (*RuleEvaluation, error) {
	eval := &RuleEvaluation{
		RuleID:      r.GetID(),
		RuleName:    r.GetName(),
		Severity:    r.GetSeverity(),
		Status:      StatusCompliant,
		Description: "No structuring pattern detected",
	}

	// Placeholder: would analyze pattern of transactions
	// Structuring: multiple transactions just below reporting threshold

	return eval, nil
}
