package ontology

import (
	"context"
	"fmt"
	"time"
)

// BaselOntology implements Basel III banking regulations
type BaselOntology struct {
	id          string
	name        string
	version     string
	description string
	rules       []Rule
}

// NewBaselOntology creates a new Basel III ontology
func NewBaselOntology() *BaselOntology {
	return &BaselOntology{
		id:          "basel-3.0",
		name:        "Basel III Capital Requirements",
		version:     "3.0.0",
		description: "Basel III framework for capital adequacy, stress testing, and market liquidity risk",
		rules: []Rule{
			&BaselCapitalAdequacyRule{},
			&BaselLeverageRatioRule{},
			&BaselLiquidityCoverageRule{},
			&BaselStressTestRule{},
		},
	}
}

func (o *BaselOntology) GetID() string                   { return o.id }
func (o *BaselOntology) GetName() string                 { return o.name }
func (o *BaselOntology) GetCategory() RegulationCategory { return CategoryBasel }
func (o *BaselOntology) GetVersion() string              { return o.version }
func (o *BaselOntology) GetDescription() string          { return o.description }

func (o *BaselOntology) GetRules() []Rule { return o.rules }

func (o *BaselOntology) GetRule(ruleID string) (Rule, error) {
	for _, rule := range o.rules {
		if rule.GetID() == ruleID {
			return rule, nil
		}
	}
	return nil, fmt.Errorf("rule not found: %s", ruleID)
}

func (o *BaselOntology) Validate(ctx context.Context, action *FinancialAction) (*ValidationResult, error) {
	result := &ValidationResult{
		OntologyID:   o.id,
		OntologyName: o.name,
		Category:     CategoryBasel,
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

// Basel Capital Adequacy Rule (CET1 Ratio >= 4.5%, Total Capital Ratio >= 8%)
type BaselCapitalAdequacyRule struct{}

func (r *BaselCapitalAdequacyRule) GetID() string   { return "basel-capital-adequacy" }
func (r *BaselCapitalAdequacyRule) GetName() string { return "Capital Adequacy Ratio" }
func (r *BaselCapitalAdequacyRule) GetDescription() string {
	return "Ensures minimum capital adequacy ratios are maintained"
}
func (r *BaselCapitalAdequacyRule) GetCategory() string        { return "Capital" }
func (r *BaselCapitalAdequacyRule) GetSeverity() SeverityLevel { return SeverityCritical }

func (r *BaselCapitalAdequacyRule) Evaluate(ctx context.Context, action *FinancialAction) (*RuleEvaluation, error) {
	eval := &RuleEvaluation{
		RuleID:      r.GetID(),
		RuleName:    r.GetName(),
		Severity:    r.GetSeverity(),
		Status:      StatusCompliant,
		Description: "Capital adequacy ratios meet Basel III requirements",
	}

	if action.Type == ActionRiskAssessment {
		// Check capital ratios
		cet1Ratio, hasCET1 := action.Context["cet1_ratio"].(float64)
		totalCapitalRatio, hasTotal := action.Context["total_capital_ratio"].(float64)

		if hasCET1 && hasTotal {
			minCET1 := 0.045 // 4.5%
			minTotal := 0.08 // 8%

			if cet1Ratio < minCET1 {
				eval.Status = StatusViolated
				eval.Description = fmt.Sprintf("CET1 ratio %.2f%% below minimum %.2f%%", cet1Ratio*100, minCET1*100)
				eval.Details = map[string]interface{}{
					"cet1_ratio": cet1Ratio,
					"min_cet1":   minCET1,
					"shortfall":  minCET1 - cet1Ratio,
				}
			} else if totalCapitalRatio < minTotal {
				eval.Status = StatusViolated
				eval.Description = fmt.Sprintf("Total capital ratio %.2f%% below minimum %.2f%%", totalCapitalRatio*100, minTotal*100)
				eval.Details = map[string]interface{}{
					"total_capital_ratio": totalCapitalRatio,
					"min_total":           minTotal,
					"shortfall":           minTotal - totalCapitalRatio,
				}
			}
		}
	}

	return eval, nil
}

// Basel Leverage Ratio Rule (>= 3%)
type BaselLeverageRatioRule struct{}

func (r *BaselLeverageRatioRule) GetID() string   { return "basel-leverage-ratio" }
func (r *BaselLeverageRatioRule) GetName() string { return "Leverage Ratio" }
func (r *BaselLeverageRatioRule) GetDescription() string {
	return "Ensures minimum leverage ratio is maintained"
}
func (r *BaselLeverageRatioRule) GetCategory() string        { return "Leverage" }
func (r *BaselLeverageRatioRule) GetSeverity() SeverityLevel { return SeverityHigh }

func (r *BaselLeverageRatioRule) Evaluate(ctx context.Context, action *FinancialAction) (*RuleEvaluation, error) {
	eval := &RuleEvaluation{
		RuleID:      r.GetID(),
		RuleName:    r.GetName(),
		Severity:    r.GetSeverity(),
		Status:      StatusCompliant,
		Description: "Leverage ratio meets Basel III requirement (3%)",
	}

	if action.Type == ActionRiskAssessment {
		if leverageRatio, ok := action.Context["leverage_ratio"].(float64); ok {
			minRatio := 0.03 // 3%
			if leverageRatio < minRatio {
				eval.Status = StatusViolated
				eval.Description = fmt.Sprintf("Leverage ratio %.2f%% below minimum %.2f%%", leverageRatio*100, minRatio*100)
				eval.Details = map[string]interface{}{
					"leverage_ratio": leverageRatio,
					"min_required":   minRatio,
					"shortfall":      minRatio - leverageRatio,
				}
			}
		}
	}

	return eval, nil
}

// Basel Liquidity Coverage Ratio Rule (LCR >= 100%)
type BaselLiquidityCoverageRule struct{}

func (r *BaselLiquidityCoverageRule) GetID() string   { return "basel-liquidity-coverage" }
func (r *BaselLiquidityCoverageRule) GetName() string { return "Liquidity Coverage Ratio" }
func (r *BaselLiquidityCoverageRule) GetDescription() string {
	return "Ensures sufficient high-quality liquid assets"
}
func (r *BaselLiquidityCoverageRule) GetCategory() string        { return "Liquidity" }
func (r *BaselLiquidityCoverageRule) GetSeverity() SeverityLevel { return SeverityCritical }

func (r *BaselLiquidityCoverageRule) Evaluate(ctx context.Context, action *FinancialAction) (*RuleEvaluation, error) {
	eval := &RuleEvaluation{
		RuleID:      r.GetID(),
		RuleName:    r.GetName(),
		Severity:    r.GetSeverity(),
		Status:      StatusCompliant,
		Description: "Liquidity coverage ratio meets requirement (100%)",
	}

	if action.Type == ActionRiskAssessment {
		if lcr, ok := action.Context["liquidity_coverage_ratio"].(float64); ok {
			minLCR := 1.0 // 100%
			if lcr < minLCR {
				eval.Status = StatusViolated
				eval.Description = fmt.Sprintf("LCR %.2f%% below minimum %.2f%%", lcr*100, minLCR*100)
				eval.Details = map[string]interface{}{
					"lcr":          lcr,
					"min_required": minLCR,
					"shortfall":    minLCR - lcr,
				}
			}
		}
	}

	return eval, nil
}

// Basel Stress Test Rule
type BaselStressTestRule struct{}

func (r *BaselStressTestRule) GetID() string   { return "basel-stress-test" }
func (r *BaselStressTestRule) GetName() string { return "Stress Testing" }
func (r *BaselStressTestRule) GetDescription() string {
	return "Ensures regular stress testing is conducted"
}
func (r *BaselStressTestRule) GetCategory() string        { return "Stress Testing" }
func (r *BaselStressTestRule) GetSeverity() SeverityLevel { return SeverityHigh }

func (r *BaselStressTestRule) Evaluate(ctx context.Context, action *FinancialAction) (*RuleEvaluation, error) {
	eval := &RuleEvaluation{
		RuleID:      r.GetID(),
		RuleName:    r.GetName(),
		Severity:    r.GetSeverity(),
		Status:      StatusCompliant,
		Description: "Stress testing requirements met",
	}

	if action.Type == ActionRiskAssessment {
		// Check if stress test was performed
		if stressTestPassed, ok := action.Context["stress_test_passed"].(bool); ok && !stressTestPassed {
			eval.Status = StatusViolated
			eval.Description = "Stress test failed or not performed"
			eval.Details = map[string]interface{}{
				"stress_test_status": "failed",
			}
		}
	}

	return eval, nil
}
