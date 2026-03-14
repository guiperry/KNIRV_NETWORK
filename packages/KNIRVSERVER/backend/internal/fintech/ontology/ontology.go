package ontology

import (
	"context"
	"fmt"
	"time"
)

// RegulationCategory represents the category of financial regulation
type RegulationCategory string

const (
	CategoryAML    RegulationCategory = "AML"    // Anti-Money Laundering
	CategoryKYC    RegulationCategory = "KYC"    // Know Your Customer
	CategorySEC    RegulationCategory = "SEC"    // Securities and Exchange Commission
	CategoryBasel  RegulationCategory = "BASEL"  // Basel Accords
	CategoryMiFID  RegulationCategory = "MIFID"  // Markets in Financial Instruments Directive
	CategoryGDPR   RegulationCategory = "GDPR"   // General Data Protection Regulation
	CategoryPCI    RegulationCategory = "PCI"    // Payment Card Industry
	CategorySOX    RegulationCategory = "SOX"    // Sarbanes-Oxley
	CategoryFinCEN RegulationCategory = "FINCEN" // Financial Crimes Enforcement Network
)

// ComplianceStatus represents the compliance status of a check
type ComplianceStatus string

const (
	StatusCompliant     ComplianceStatus = "compliant"
	StatusViolated      ComplianceStatus = "violated"
	StatusWarning       ComplianceStatus = "warning"
	StatusNotApplicable ComplianceStatus = "not_applicable"
)

// SeverityLevel represents the severity of a compliance issue
type SeverityLevel string

const (
	SeverityCritical SeverityLevel = "critical"
	SeverityHigh     SeverityLevel = "high"
	SeverityMedium   SeverityLevel = "medium"
	SeverityLow      SeverityLevel = "low"
	SeverityInfo     SeverityLevel = "info"
)

// FinancialOntology defines the interface for financial regulatory ontologies
type FinancialOntology interface {
	// GetID returns the unique identifier for this ontology
	GetID() string

	// GetName returns the human-readable name
	GetName() string

	// GetCategory returns the regulation category
	GetCategory() RegulationCategory

	// GetVersion returns the ontology version
	GetVersion() string

	// GetDescription returns a description of this ontology
	GetDescription() string

	// Validate checks if an action complies with this ontology
	Validate(ctx context.Context, action *FinancialAction) (*ValidationResult, error)

	// GetRules returns all rules in this ontology
	GetRules() []Rule

	// GetRule returns a specific rule by ID
	GetRule(ruleID string) (Rule, error)
}

// Rule represents a single regulation rule
type Rule interface {
	// GetID returns the rule identifier
	GetID() string

	// GetName returns the rule name
	GetName() string

	// GetDescription returns the rule description
	GetDescription() string

	// GetCategory returns the rule category
	GetCategory() string

	// GetSeverity returns the severity if violated
	GetSeverity() SeverityLevel

	// Evaluate evaluates the rule against an action
	Evaluate(ctx context.Context, action *FinancialAction) (*RuleEvaluation, error)
}

// FinancialAction represents an action taken by a financial AI agent
type FinancialAction struct {
	ID           string                 `json:"id"`
	AgentID      string                 `json:"agent_id"`
	Type         ActionType             `json:"type"`
	Timestamp    time.Time              `json:"timestamp"`
	Intent       string                 `json:"intent"`
	Parameters   map[string]interface{} `json:"parameters"`
	Context      map[string]interface{} `json:"context"`
	Amount       *MonetaryValue         `json:"amount,omitempty"`
	Counterparty *CounterpartyInfo      `json:"counterparty,omitempty"`
	Instrument   *FinancialInstrument   `json:"instrument,omitempty"`
}

// ActionType represents the type of financial action
type ActionType string

const (
	ActionTrade          ActionType = "TRADE"
	ActionTransfer       ActionType = "TRANSFER"
	ActionSettlement     ActionType = "SETTLEMENT"
	ActionReconciliation ActionType = "RECONCILIATION"
	ActionReporting      ActionType = "REPORTING"
	ActionRiskAssessment ActionType = "RISK_ASSESSMENT"
	ActionKYC            ActionType = "KYC"
	ActionAMLCheck       ActionType = "AML_CHECK"
)

// MonetaryValue represents a monetary amount with currency
type MonetaryValue struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

// CounterpartyInfo represents information about a counterparty
type CounterpartyInfo struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name,omitempty"`
	Type         string                 `json:"type"` // individual, corporate, fund, etc.
	RiskRating   string                 `json:"risk_rating,omitempty"`
	Jurisdiction string                 `json:"jurisdiction,omitempty"`
	IsPEP        bool                   `json:"is_pep,omitempty"`
	IsSanctioned bool                   `json:"is_sanctioned,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// FinancialInstrument represents a financial instrument
type FinancialInstrument struct {
	ID       string `json:"id"`
	Symbol   string `json:"symbol,omitempty"`
	Name     string `json:"name,omitempty"`
	Type     string `json:"type"` // equity, bond, derivative, etc.
	Exchange string `json:"exchange,omitempty"`
	Currency string `json:"currency,omitempty"`
}

// ValidationResult represents the result of validating an action against an ontology
type ValidationResult struct {
	OntologyID    string             `json:"ontology_id"`
	OntologyName  string             `json:"ontology_name"`
	Category      RegulationCategory `json:"category"`
	OverallStatus ComplianceStatus   `json:"overall_status"`
	RuleResults   []RuleEvaluation   `json:"rule_results"`
	Score         float64            `json:"score"` // 0-1 compliance score
	Timestamp     time.Time          `json:"timestamp"`
}

// RuleEvaluation represents the result of evaluating a single rule
type RuleEvaluation struct {
	RuleID      string                 `json:"rule_id"`
	RuleName    string                 `json:"rule_name"`
	Status      ComplianceStatus       `json:"status"`
	Severity    SeverityLevel          `json:"severity"`
	Description string                 `json:"description"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// OntologyRegistry manages available financial ontologies
type OntologyRegistry struct {
	onthologies map[string]FinancialOntology
}

// NewOntologyRegistry creates a new ontology registry
func NewOntologyRegistry() *OntologyRegistry {
	return &OntologyRegistry{
		onthologies: make(map[string]FinancialOntology),
	}
}

// Register adds an ontology to the registry
func (r *OntologyRegistry) Register(ontology FinancialOntology) error {
	if ontology == nil {
		return fmt.Errorf("ontology cannot be nil")
	}
	id := ontology.GetID()
	if id == "" {
		return fmt.Errorf("ontology ID cannot be empty")
	}
	r.onthologies[id] = ontology
	return nil
}

// Get retrieves an ontology by ID
func (r *OntologyRegistry) Get(id string) (FinancialOntology, error) {
	ontology, exists := r.onthologies[id]
	if !exists {
		return nil, fmt.Errorf("ontology not found: %s", id)
	}
	return ontology, nil
}

// List returns all registered ontologies
func (r *OntologyRegistry) List() []FinancialOntology {
	result := make([]FinancialOntology, 0, len(r.onthologies))
	for _, ontology := range r.onthologies {
		result = append(result, ontology)
	}
	return result
}

// ListByCategory returns ontologies filtered by category
func (r *OntologyRegistry) ListByCategory(category RegulationCategory) []FinancialOntology {
	result := make([]FinancialOntology, 0)
	for _, ontology := range r.onthologies {
		if ontology.GetCategory() == category {
			result = append(result, ontology)
		}
	}
	return result
}

// Unregister removes an ontology from the registry
func (r *OntologyRegistry) Unregister(id string) error {
	if _, exists := r.onthologies[id]; !exists {
		return fmt.Errorf("ontology not found: %s", id)
	}
	delete(r.onthologies, id)
	return nil
}

// OntologyLoader handles loading ontologies from various sources
type OntologyLoader struct {
	registry *OntologyRegistry
}

// NewOntologyLoader creates a new ontology loader
func NewOntologyLoader(registry *OntologyRegistry) *OntologyLoader {
	return &OntologyLoader{registry: registry}
}

// LoadBuiltInOntologies loads all built-in ontologies
func (l *OntologyLoader) LoadBuiltInOntologies() error {
	// AML Ontology
	amlOntology := NewAMLOntology()
	if err := l.registry.Register(amlOntology); err != nil {
		return fmt.Errorf("failed to register AML ontology: %w", err)
	}

	// KYC Ontology
	kycOntology := NewKYCOntology()
	if err := l.registry.Register(kycOntology); err != nil {
		return fmt.Errorf("failed to register KYC ontology: %w", err)
	}

	// SEC Ontology
	secOntology := NewSECOntology()
	if err := l.registry.Register(secOntology); err != nil {
		return fmt.Errorf("failed to register SEC ontology: %w", err)
	}

	// Basel III Ontology
	baselOntology := NewBaselOntology()
	if err := l.registry.Register(baselOntology); err != nil {
		return fmt.Errorf("failed to register Basel ontology: %w", err)
	}

	return nil
}

// ComplianceEngine executes compliance checks against multiple ontologies
type ComplianceEngine struct {
	registry *OntologyRegistry
}

// NewComplianceEngine creates a new compliance engine
func NewComplianceEngine(registry *OntologyRegistry) *ComplianceEngine {
	return &ComplianceEngine{registry: registry}
}

// ValidateAction validates a financial action against all applicable ontologies
func (e *ComplianceEngine) ValidateAction(ctx context.Context, action *FinancialAction, categories []RegulationCategory) (*ComprehensiveValidation, error) {
	result := &ComprehensiveValidation{
		ActionID:     action.ID,
		Timestamp:    time.Now(),
		Results:      make(map[string]*ValidationResult),
		OverallScore: 1.0,
	}

	ontologies := make([]FinancialOntology, 0)

	if len(categories) == 0 {
		// Check against all ontologies
		ontologies = e.registry.List()
	} else {
		// Check against specified categories
		for _, category := range categories {
			ontologies = append(ontologies, e.registry.ListByCategory(category)...)
		}
	}

	for _, ontology := range ontologies {
		validation, err := ontology.Validate(ctx, action)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", ontology.GetID(), err))
			continue
		}
		result.Results[ontology.GetID()] = validation

		// Update overall score (worst score wins)
		if validation.Score < result.OverallScore {
			result.OverallScore = validation.Score
		}

		// Check for violations
		if validation.OverallStatus == StatusViolated {
			result.Violations++
			result.IsCompliant = false
		}
	}

	if result.Violations == 0 {
		result.IsCompliant = true
	}

	return result, nil
}

// ComprehensiveValidation represents the result of validating against multiple ontologies
type ComprehensiveValidation struct {
	ActionID     string                       `json:"action_id"`
	Timestamp    time.Time                    `json:"timestamp"`
	Results      map[string]*ValidationResult `json:"results"`
	OverallScore float64                      `json:"overall_score"`
	IsCompliant  bool                         `json:"is_compliant"`
	Violations   int                          `json:"violations"`
	Errors       []string                     `json:"errors,omitempty"`
}
