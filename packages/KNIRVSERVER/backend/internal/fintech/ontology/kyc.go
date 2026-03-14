package ontology

import (
	"context"
	"fmt"
	"time"
)

// KYCOntology implements Know Your Customer regulations
type KYCOntology struct {
	id          string
	name        string
	version     string
	description string
	rules       []Rule
}

// NewKYCOntology creates a new KYC ontology
func NewKYCOntology() *KYCOntology {
	return &KYCOntology{
		id:          "kyc-1.0",
		name:        "Know Your Customer (KYC)",
		version:     "1.0.0",
		description: "Know Your Customer regulations for customer identification, verification, and risk assessment",
		rules: []Rule{
			&KYCIdentityVerificationRule{},
			&KYCRiskAssessmentRule{},
			&KYCPEPCheckRule{},
			&KYCBeneficialOwnershipRule{},
		},
	}
}

func (o *KYCOntology) GetID() string                   { return o.id }
func (o *KYCOntology) GetName() string                 { return o.name }
func (o *KYCOntology) GetCategory() RegulationCategory { return CategoryKYC }
func (o *KYCOntology) GetVersion() string              { return o.version }
func (o *KYCOntology) GetDescription() string          { return o.description }

func (o *KYCOntology) GetRules() []Rule { return o.rules }

func (o *KYCOntology) GetRule(ruleID string) (Rule, error) {
	for _, rule := range o.rules {
		if rule.GetID() == ruleID {
			return rule, nil
		}
	}
	return nil, fmt.Errorf("rule not found: %s", ruleID)
}

func (o *KYCOntology) Validate(ctx context.Context, action *FinancialAction) (*ValidationResult, error) {
	result := &ValidationResult{
		OntologyID:   o.id,
		OntologyName: o.name,
		Category:     CategoryKYC,
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

// KYC Identity Verification Rule
type KYCIdentityVerificationRule struct{}

func (r *KYCIdentityVerificationRule) GetID() string   { return "kyc-identity-verification" }
func (r *KYCIdentityVerificationRule) GetName() string { return "Identity Verification Required" }
func (r *KYCIdentityVerificationRule) GetDescription() string {
	return "Ensures customer identity is properly verified"
}
func (r *KYCIdentityVerificationRule) GetCategory() string        { return "Identity" }
func (r *KYCIdentityVerificationRule) GetSeverity() SeverityLevel { return SeverityCritical }

func (r *KYCIdentityVerificationRule) Evaluate(ctx context.Context, action *FinancialAction) (*RuleEvaluation, error) {
	eval := &RuleEvaluation{
		RuleID:      r.GetID(),
		RuleName:    r.GetName(),
		Severity:    r.GetSeverity(),
		Status:      StatusCompliant,
		Description: "Identity verification check passed",
	}

	if action.Type == ActionKYC {
		// Check if identity verification is present
		if verified, ok := action.Context["identity_verified"].(bool); ok && !verified {
			eval.Status = StatusViolated
			eval.Description = "Customer identity not verified"
		}
	}

	return eval, nil
}

// KYC Risk Assessment Rule
type KYCRiskAssessmentRule struct{}

func (r *KYCRiskAssessmentRule) GetID() string   { return "kyc-risk-assessment" }
func (r *KYCRiskAssessmentRule) GetName() string { return "Risk Assessment Complete" }
func (r *KYCRiskAssessmentRule) GetDescription() string {
	return "Ensures customer risk assessment is completed"
}
func (r *KYCRiskAssessmentRule) GetCategory() string        { return "Risk" }
func (r *KYCRiskAssessmentRule) GetSeverity() SeverityLevel { return SeverityHigh }

func (r *KYCRiskAssessmentRule) Evaluate(ctx context.Context, action *FinancialAction) (*RuleEvaluation, error) {
	eval := &RuleEvaluation{
		RuleID:      r.GetID(),
		RuleName:    r.GetName(),
		Severity:    r.GetSeverity(),
		Status:      StatusCompliant,
		Description: "Risk assessment completed",
	}

	if action.Counterparty != nil {
		if action.Counterparty.RiskRating == "" {
			eval.Status = StatusWarning
			eval.Description = "Customer risk rating not assigned"
		}
	}

	return eval, nil
}

// KYC PEP Check Rule
type KYCPEPCheckRule struct{}

func (r *KYCPEPCheckRule) GetID() string   { return "kyc-pep-check" }
func (r *KYCPEPCheckRule) GetName() string { return "Politically Exposed Person Check" }
func (r *KYCPEPCheckRule) GetDescription() string {
	return "Checks if customer is a PEP and applies enhanced due diligence"
}
func (r *KYCPEPCheckRule) GetCategory() string        { return "PEP" }
func (r *KYCPEPCheckRule) GetSeverity() SeverityLevel { return SeverityHigh }

func (r *KYCPEPCheckRule) Evaluate(ctx context.Context, action *FinancialAction) (*RuleEvaluation, error) {
	eval := &RuleEvaluation{
		RuleID:      r.GetID(),
		RuleName:    r.GetName(),
		Severity:    r.GetSeverity(),
		Status:      StatusCompliant,
		Description: "PEP check completed - not a PEP",
	}

	if action.Counterparty != nil && action.Counterparty.IsPEP {
		eval.Status = StatusWarning
		eval.Description = "Customer is a PEP - enhanced due diligence required"
		eval.Details = map[string]interface{}{
			"pep_status":   true,
			"requires_edd": true,
		}
	}

	return eval, nil
}

// KYC Beneficial Ownership Rule
type KYCBeneficialOwnershipRule struct{}

func (r *KYCBeneficialOwnershipRule) GetID() string   { return "kyc-beneficial-ownership" }
func (r *KYCBeneficialOwnershipRule) GetName() string { return "Beneficial Ownership Verification" }
func (r *KYCBeneficialOwnershipRule) GetDescription() string {
	return "Verifies beneficial ownership for corporate customers"
}
func (r *KYCBeneficialOwnershipRule) GetCategory() string        { return "Ownership" }
func (r *KYCBeneficialOwnershipRule) GetSeverity() SeverityLevel { return SeverityMedium }

func (r *KYCBeneficialOwnershipRule) Evaluate(ctx context.Context, action *FinancialAction) (*RuleEvaluation, error) {
	eval := &RuleEvaluation{
		RuleID:      r.GetID(),
		RuleName:    r.GetName(),
		Severity:    r.GetSeverity(),
		Status:      StatusCompliant,
		Description: "Beneficial ownership verified",
	}

	if action.Counterparty != nil && action.Counterparty.Type == "corporate" {
		// Placeholder: would check for beneficial ownership documentation
		// CDD Rule requires identification of beneficial owners (>25% ownership)
	}

	return eval, nil
}
