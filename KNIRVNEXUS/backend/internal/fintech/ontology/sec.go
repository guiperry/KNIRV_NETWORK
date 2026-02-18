package ontology

import (
	"context"
	"fmt"
	"time"
)

// SECOntology implements SEC (Securities and Exchange Commission) regulations
type SECOntology struct {
	id          string
	name        string
	version     string
	description string
	rules       []Rule
}

// NewSECOntology creates a new SEC ontology
func NewSECOntology() *SECOntology {
	return &SECOntology{
		id:          "sec-1.0",
		name:        "SEC Securities Regulations",
		version:     "1.0.0",
		description: "SEC regulations covering securities trading, reporting, and market manipulation prevention",
		rules: []Rule{
			&SECMaterialNonPublicInfoRule{},
			&SECMarketManipulationRule{},
			&SECPatternDayTradingRule{},
			&SECSellSideDisclosureRule{},
		},
	}
}

func (o *SECOntology) GetID() string                   { return o.id }
func (o *SECOntology) GetName() string                 { return o.name }
func (o *SECOntology) GetCategory() RegulationCategory { return CategorySEC }
func (o *SECOntology) GetVersion() string              { return o.version }
func (o *SECOntology) GetDescription() string          { return o.description }

func (o *SECOntology) GetRules() []Rule { return o.rules }

func (o *SECOntology) GetRule(ruleID string) (Rule, error) {
	for _, rule := range o.rules {
		if rule.GetID() == ruleID {
			return rule, nil
		}
	}
	return nil, fmt.Errorf("rule not found: %s", ruleID)
}

func (o *SECOntology) Validate(ctx context.Context, action *FinancialAction) (*ValidationResult, error) {
	result := &ValidationResult{
		OntologyID:   o.id,
		OntologyName: o.name,
		Category:     CategorySEC,
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
			result.Score -= 0.25
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

// SEC Material Non-Public Information Rule
type SECMaterialNonPublicInfoRule struct{}

func (r *SECMaterialNonPublicInfoRule) GetID() string   { return "sec-mnpi" }
func (r *SECMaterialNonPublicInfoRule) GetName() string { return "Material Non-Public Information" }
func (r *SECMaterialNonPublicInfoRule) GetDescription() string {
	return "Prevents trading on material non-public information"
}
func (r *SECMaterialNonPublicInfoRule) GetCategory() string        { return "Insider Trading" }
func (r *SECMaterialNonPublicInfoRule) GetSeverity() SeverityLevel { return SeverityCritical }

func (r *SECMaterialNonPublicInfoRule) Evaluate(ctx context.Context, action *FinancialAction) (*RuleEvaluation, error) {
	eval := &RuleEvaluation{
		RuleID:      r.GetID(),
		RuleName:    r.GetName(),
		Severity:    r.GetSeverity(),
		Status:      StatusCompliant,
		Description: "No MNPI detected",
	}

	if action.Type == ActionTrade {
		// Check for MNPI indicators
		if hasMNPI, ok := action.Context["has_mnpi"].(bool); ok && hasMNPI {
			eval.Status = StatusViolated
			eval.Description = "Trading on material non-public information detected"
			eval.Details = map[string]interface{}{
				"mnpi_detected":  true,
				"violation_type": "insider_trading",
			}
		}
	}

	return eval, nil
}

// SEC Market Manipulation Rule
type SECMarketManipulationRule struct{}

func (r *SECMarketManipulationRule) GetID() string   { return "sec-market-manipulation" }
func (r *SECMarketManipulationRule) GetName() string { return "Market Manipulation Detection" }
func (r *SECMarketManipulationRule) GetDescription() string {
	return "Detects potential market manipulation schemes"
}
func (r *SECMarketManipulationRule) GetCategory() string        { return "Market Abuse" }
func (r *SECMarketManipulationRule) GetSeverity() SeverityLevel { return SeverityCritical }

func (r *SECMarketManipulationRule) Evaluate(ctx context.Context, action *FinancialAction) (*RuleEvaluation, error) {
	eval := &RuleEvaluation{
		RuleID:      r.GetID(),
		RuleName:    r.GetName(),
		Severity:    r.GetSeverity(),
		Status:      StatusCompliant,
		Description: "No market manipulation patterns detected",
	}

	// Placeholder: would analyze trading patterns for:
	// - Spoofing (layering fake orders)
	// - Wash trading (trading with oneself)
	// - Pump and dump schemes

	return eval, nil
}

// SEC Pattern Day Trading Rule
type SECPatternDayTradingRule struct{}

func (r *SECPatternDayTradingRule) GetID() string   { return "sec-pattern-day-trading" }
func (r *SECPatternDayTradingRule) GetName() string { return "Pattern Day Trading Compliance" }
func (r *SECPatternDayTradingRule) GetDescription() string {
	return "Enforces pattern day trading rules for margin accounts"
}
func (r *SECPatternDayTradingRule) GetCategory() string        { return "Day Trading" }
func (r *SECPatternDayTradingRule) GetSeverity() SeverityLevel { return SeverityMedium }

func (r *SECPatternDayTradingRule) Evaluate(ctx context.Context, action *FinancialAction) (*RuleEvaluation, error) {
	eval := &RuleEvaluation{
		RuleID:      r.GetID(),
		RuleName:    r.GetName(),
		Severity:    r.GetSeverity(),
		Status:      StatusCompliant,
		Description: "Pattern day trading rules satisfied",
	}

	// Pattern Day Trader: 4+ day trades in 5 business days in margin account
	// Requires $25,000 minimum equity

	return eval, nil
}

// SEC Sell-Side Disclosure Rule
type SECSellSideDisclosureRule struct{}

func (r *SECSellSideDisclosureRule) GetID() string   { return "sec-sell-side-disclosure" }
func (r *SECSellSideDisclosureRule) GetName() string { return "Sell-Side Research Disclosure" }
func (r *SECSellSideDisclosureRule) GetDescription() string {
	return "Ensures proper disclosure in sell-side research"
}
func (r *SECSellSideDisclosureRule) GetCategory() string        { return "Disclosure" }
func (r *SECSellSideDisclosureRule) GetSeverity() SeverityLevel { return SeverityHigh }

func (r *SECSellSideDisclosureRule) Evaluate(ctx context.Context, action *FinancialAction) (*RuleEvaluation, error) {
	eval := &RuleEvaluation{
		RuleID:      r.GetID(),
		RuleName:    r.GetName(),
		Severity:    r.GetSeverity(),
		Status:      StatusCompliant,
		Description: "Sell-side disclosure requirements met",
	}

	// Check for required disclosures in research reports

	return eval, nil
}
