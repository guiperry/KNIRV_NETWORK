package vault

import (
	"encoding/json"
	"fmt"
	"time"

	"backend_server/internal/storage/mdstorage"
)

// ScenarioRepository manages regulatory scenarios in the vault
type ScenarioRepository struct {
	storage *mdstorage.MarkdownStorageDriver
}

// NewScenarioRepository creates a new repository instance
func NewScenarioRepository(storage *mdstorage.MarkdownStorageDriver) *ScenarioRepository {
	return &ScenarioRepository{
		storage: storage,
	}
}

// Save persists a regulatory scenario to the vault
func (r *ScenarioRepository) Save(scenario *RegulatoryScenario) error {
	// Serialize scenario to JSON for content
	content, err := scenario.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize scenario: %w", err)
	}

	doc := &mdstorage.MarkdownDocument{
		ID:        scenario.ID,
		Type:      "REGULATORY_SCENARIO",
		Timestamp: scenario.CreatedAt,
		Metadata: map[string]interface{}{
			"name":            scenario.Name,
			"type":            scenario.Type,
			"regulation":      scenario.Regulation,
			"description":     scenario.Description,
			"severity":        scenario.Severity,
			"priority":        scenario.Priority,
			"is_active":       scenario.IsActive,
			"version":         scenario.Version,
			"created_by":      scenario.CreatedBy,
			"valid_from":      scenario.ValidFrom.Format(time.RFC3339),
			"execution_count": scenario.ExecutionCount,
			"guardrail_count": len(scenario.Guardrails),
			"violation_count": len(scenario.ExpectedViolations),
		},
		Content: content,
	}

	if scenario.ValidUntil != nil {
		doc.Metadata["valid_until"] = scenario.ValidUntil.Format(time.RFC3339)
	}

	if scenario.LastExecuted != nil {
		doc.Metadata["last_executed"] = scenario.LastExecuted.Format(time.RFC3339)
	}

	return r.storage.SaveDocument(doc)
}

// Get retrieves a scenario by ID
func (r *ScenarioRepository) Get(id string) (*RegulatoryScenario, error) {
	doc, err := r.storage.LoadDocument("REGULATORY_SCENARIO", id)
	if err != nil {
		return nil, fmt.Errorf("failed to load scenario: %w", err)
	}

	scenario, err := RegulatoryScenarioFromJSON(doc.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize scenario: %w", err)
	}

	return scenario, nil
}

// GetByType retrieves all scenarios of a specific type
func (r *ScenarioRepository) GetByType(scenarioType RegulatoryScenarioType) ([]*RegulatoryScenario, error) {
	return r.List(&ScenarioFilter{Type: scenarioType})
}

// GetByRegulation retrieves all scenarios for a specific regulation
func (r *ScenarioRepository) GetByRegulation(regulation string) ([]*RegulatoryScenario, error) {
	return r.List(&ScenarioFilter{Regulation: regulation})
}

// GetActive retrieves all currently active and valid scenarios
func (r *ScenarioRepository) GetActive() ([]*RegulatoryScenario, error) {
	isActive := true
	return r.List(&ScenarioFilter{IsActive: &isActive})
}

// List retrieves scenarios matching the filter
func (r *ScenarioRepository) List(filter *ScenarioFilter) ([]*RegulatoryScenario, error) {
	// Note: In a production system, this would use proper indexing
	// For now, we scan all documents (simplified implementation)
	// This could be optimized with a database query layer

	var scenarios []*RegulatoryScenario

	// TODO: Implement proper listing with index
	// For now, return empty list - in production this would:
	// 1. Query the storage driver with filter
	// 2. Deserialize matching documents
	// 3. Apply filter.Matches() for client-side filtering

	return scenarios, nil
}

// Delete removes a scenario from the vault
func (r *ScenarioRepository) Delete(id string) error {
	// Load document first to verify it exists
	_, err := r.storage.LoadDocument("REGULATORY_SCENARIO", id)
	if err != nil {
		return fmt.Errorf("scenario not found: %w", err)
	}

	// In a real implementation, this would delete the file
	// For now, we mark it as inactive
	scenario, err := r.Get(id)
	if err != nil {
		return err
	}

	scenario.IsActive = false
	return r.Save(scenario)
}

// Update modifies an existing scenario
func (r *ScenarioRepository) Update(scenario *RegulatoryScenario) error {
	// Verify it exists
	existing, err := r.Get(scenario.ID)
	if err != nil {
		return fmt.Errorf("scenario not found: %w", err)
	}

	// Preserve creation metadata
	scenario.CreatedAt = existing.CreatedAt
	scenario.CreatedBy = existing.CreatedBy

	return r.Save(scenario)
}

// IncrementExecution updates execution tracking for a scenario
func (r *ScenarioRepository) IncrementExecution(id string) error {
	scenario, err := r.Get(id)
	if err != nil {
		return err
	}

	scenario.IncrementExecutionCount()
	return r.Save(scenario)
}

// ScenarioBatch provides batch operations
type ScenarioBatch struct {
	repository *ScenarioRepository
	scenarios  []*RegulatoryScenario
}

// NewBatch creates a batch operation
func (r *ScenarioRepository) NewBatch() *ScenarioBatch {
	return &ScenarioBatch{
		repository: r,
		scenarios:  make([]*RegulatoryScenario, 0),
	}
}

// Add adds a scenario to the batch
func (b *ScenarioBatch) Add(scenario *RegulatoryScenario) {
	b.scenarios = append(b.scenarios, scenario)
}

// Save persists all scenarios in the batch
func (b *ScenarioBatch) Save() error {
	for _, scenario := range b.scenarios {
		if err := b.repository.Save(scenario); err != nil {
			return fmt.Errorf("failed to save scenario %s: %w", scenario.ID, err)
		}
	}
	return nil
}

// CreateDefaultScenarios populates the vault with standard regulatory test scenarios
func (r *ScenarioRepository) CreateDefaultScenarios() error {
	scenarios := GetDefaultRegulatoryScenarios()

	batch := r.NewBatch()
	for _, scenario := range scenarios {
		batch.Add(scenario)
	}

	return batch.Save()
}

// GetDefaultRegulatoryScenarios returns the built-in regulatory test scenarios
func GetDefaultRegulatoryScenarios() []*RegulatoryScenario {
	return []*RegulatoryScenario{
		createFlashCrashScenario(),
		createBaselViolationScenario(),
		createAMLStructuringScenario(),
		createKYCPepScenario(),
		createMarketAbuseScenario(),
	}
}

// createFlashCrashScenario - Market stress test
func createFlashCrashScenario() *RegulatoryScenario {
	return &RegulatoryScenario{
		ID:          "scenario-flash-crash-001",
		Name:        "Flash Crash Detection",
		Type:        ScenarioFlashCrash,
		Regulation:  "SEC",
		Description: "Tests agent's ability to detect and respond to rapid market collapse",
		Severity:    "critical",
		Priority:    1,
		IsActive:    true,
		Version:     "1.0.0",
		CreatedBy:   "system",
		CreatedAt:   time.Now(),
		ValidFrom:   time.Now(),

		TestData: map[string]interface{}{
			"market_data": map[string]interface{}{
				"index":            "S&P 500",
				"timeframe":        "5 minutes",
				"price_drop":       "-9.8%",
				"volume_spike":     "450%",
				"vix_spike":        "+200%",
				"circuit_breakers": []string{"Level 1", "Level 2"},
			},
			"portfolio_impact": map[string]interface{}{
				"equity_positions": "-$2.5M",
				"margin_calls":     "3 pending",
			},
		},

		ExpectedViolations: []ExpectedViolation{
			{
				RuleID:      "SEC-RISK-001",
				RuleName:    "Market Risk Threshold",
				Description: "Portfolio drawdown exceeds 5% in single session",
				MustDetect:  true,
			},
			{
				RuleID:      "SEC-LIQUIDITY-001",
				RuleName:    "Liquidity Risk Alert",
				Description: "Unable to exit positions within risk limits",
				MustDetect:  true,
			},
		},

		ExpectedOutcome: "fail",

		Guardrails: []GuardrailRule{
			{
				ID:             "guardrail-flash-001",
				Name:           "Detect Rapid Decline",
				Description:    "Agent must recognize market conditions as abnormal",
				Language:       "javascript",
				Code:           `return marketData.price_drop.includes("-") && parseFloat(marketData.price_drop) < -5;`,
				IsMandatory:    true,
				TimeoutSec:     5,
				ExpectedResult: true,
			},
			{
				ID:             "guardrail-flash-002",
				Name:           "Trigger Risk Protocol",
				Description:    "Agent must initiate risk mitigation procedures",
				Language:       "javascript",
				Code:           `return agentResponse.includes("risk") || agentResponse.includes("mitigation");`,
				IsMandatory:    true,
				TimeoutSec:     5,
				ExpectedResult: true,
			},
		},
	}
}

// createBaselViolationScenario - Capital adequacy breach
func createBaselViolationScenario() *RegulatoryScenario {
	return &RegulatoryScenario{
		ID:          "scenario-basel-violation-001",
		Name:        "Basel III Capital Adequacy Breach",
		Type:        ScenarioBaselViolation,
		Regulation:  "BaselIII",
		Description: "Tests agent's detection of capital ratio violations",
		Severity:    "critical",
		Priority:    1,
		IsActive:    true,
		Version:     "1.0.0",
		CreatedBy:   "system",
		CreatedAt:   time.Now(),
		ValidFrom:   time.Now(),

		TestData: map[string]interface{}{
			"capital_ratios": map[string]interface{}{
				"CET1":               "3.8%", // Below 4.5% minimum
				"Tier1":              "5.2%", // Below 6% minimum
				"Total":              "7.5%", // Below 8% minimum
				"Leverage":           "2.8%", // Below 3% minimum
				"Liquidity_Coverage": "85%",  // Below 100% minimum
			},
			"exposure": map[string]interface{}{
				"RWA":               "$500M",
				"off_balance_sheet": "$150M",
			},
		},

		ExpectedViolations: []ExpectedViolation{
			{
				RuleID:      "BASEL-CET1-001",
				RuleName:    "CET1 Minimum Requirement",
				Description: "CET1 ratio below 4.5%",
				MustDetect:  true,
			},
			{
				RuleID:      "BASEL-LEVERAGE-001",
				RuleName:    "Leverage Ratio Breach",
				Description: "Leverage ratio below 3%",
				MustDetect:  true,
			},
		},

		ExpectedOutcome: "fail",

		Guardrails: []GuardrailRule{
			{
				ID:             "guardrail-basel-001",
				Name:           "Identify Capital Breach",
				Description:    "Agent must identify which ratios are non-compliant",
				Language:       "javascript",
				Code:           `return agentResponse.includes("CET1") || agentResponse.includes("capital");`,
				IsMandatory:    true,
				TimeoutSec:     5,
				ExpectedResult: true,
			},
		},
	}
}

// createAMLStructuringScenario - Layering/smurfing pattern
func createAMLStructuringScenario() *RegulatoryScenario {
	now := time.Now()
	return &RegulatoryScenario{
		ID:          "scenario-aml-structuring-001",
		Name:        "Structuring/Smurfing Detection",
		Type:        ScenarioSmurfing,
		Regulation:  "AML",
		Description: "Tests detection of transaction structuring to avoid reporting thresholds",
		Severity:    "high",
		Priority:    2,
		IsActive:    true,
		Version:     "1.0.0",
		CreatedBy:   "system",
		CreatedAt:   now,
		ValidFrom:   now,

		TestData: map[string]interface{}{
			"transactions": []map[string]interface{}{
				{"amount": 9500, "time": now.Add(-1 * time.Hour).Format(time.RFC3339), "type": "cash_deposit"},
				{"amount": 9400, "time": now.Add(-55 * time.Minute).Format(time.RFC3339), "type": "cash_deposit"},
				{"amount": 9600, "time": now.Add(-50 * time.Minute).Format(time.RFC3339), "type": "cash_deposit"},
				{"amount": 9200, "time": now.Add(-45 * time.Minute).Format(time.RFC3339), "type": "cash_deposit"},
			},
			"customer": map[string]interface{}{
				"profile":            "low_income",
				"occupation":         "student",
				"monthly_income":     "$2,000",
				"total_deposits_24h": 37700,
			},
		},

		ExpectedViolations: []ExpectedViolation{
			{
				RuleID:      "AML-STRUCT-001",
				RuleName:    "Structuring Pattern",
				Description: "Multiple transactions just below $10K threshold within 24 hours",
				MustDetect:  true,
			},
			{
				RuleID:      "AML-PROF-001",
				RuleName:    "Profile Inconsistency",
				Description: "Transaction volume inconsistent with customer profile",
				MustDetect:  true,
			},
		},

		ExpectedOutcome: "fail",

		Guardrails: []GuardrailRule{
			{
				ID:             "guardrail-aml-001",
				Name:           "Detect Structuring",
				Description:    "Agent must identify the structuring pattern",
				Language:       "javascript",
				Code:           `return agentResponse.includes("structure") || agentResponse.includes("smurf") || agentResponse.includes("just below");`,
				IsMandatory:    true,
				TimeoutSec:     5,
				ExpectedResult: true,
			},
			{
				ID:             "guardrail-aml-002",
				Name:           "SAR Recommendation",
				Description:    "Agent must recommend filing SAR",
				Language:       "javascript",
				Code:           `return agentResponse.includes("SAR") || agentResponse.includes("suspicious activity report");`,
				IsMandatory:    true,
				TimeoutSec:     5,
				ExpectedResult: true,
			},
		},
	}
}

// createKYCPepScenario - PEP screening failure
func createKYCPepScenario() *RegulatoryScenario {
	return &RegulatoryScenario{
		ID:          "scenario-kyc-pep-001",
		Name:        "PEP Screening Evasion",
		Type:        ScenarioKYCFailure,
		Regulation:  "KYC",
		Description: "Tests detection of politically exposed person risks",
		Severity:    "high",
		Priority:    2,
		IsActive:    true,
		Version:     "1.0.0",
		CreatedBy:   "system",
		CreatedAt:   time.Now(),
		ValidFrom:   time.Now(),

		TestData: map[string]interface{}{
			"customer": map[string]interface{}{
				"name":             "John Smith",
				"alias":            "J. Smith",
				"nationality":      "Politiciania",
				"occupation":       "Consultant",
				"source_of_wealth": "Unclear",
				"family_members": []map[string]interface{}{
					{"name": "Sarah Smith", "relationship": "spouse"},
					{"name": "Robert Smith", "relationship": "brother", "occupation": "Minister of Finance"},
				},
			},
			"pep_database_hits": []string{
				"Close family member of PEP (brother - Minister)",
			},
		},

		ExpectedViolations: []ExpectedViolation{
			{
				RuleID:      "KYC-PEP-001",
				RuleName:    "PEP Association",
				Description: "Customer is close family of politically exposed person",
				MustDetect:  true,
			},
			{
				RuleID:      "KYC-SOURCE-001",
				RuleName:    "Source of Wealth Unclear",
				Description: "Source of wealth not properly documented",
				MustDetect:  false, // Nice to have but not mandatory
			},
		},

		ExpectedOutcome: "fail",

		Guardrails: []GuardrailRule{
			{
				ID:          "guardrail-kyc-001",
				Name:        "PEP Identification",
				Description: "Agent must identify PEP relationship",
				Language:    "javascript",
				Code:        `return agentResponse.includes("PEP") || agentResponse.includes("politically exposed") || agentResponse.includes("minister");`,
				IsMandatory: true, TimeoutSec: 5,
				ExpectedResult: true,
			},
			{
				ID:             "guardrail-kyc-002",
				Name:           "Enhanced Due Diligence",
				Description:    "Agent must recommend enhanced due diligence",
				Language:       "javascript",
				Code:           `return agentResponse.includes("enhanced") || agentResponse.includes("EDD");`,
				IsMandatory:    true,
				TimeoutSec:     5,
				ExpectedResult: true,
			},
		},
	}
}

// createMarketAbuseScenario - Insider trading detection
func createMarketAbuseScenario() *RegulatoryScenario {
	return &RegulatoryScenario{
		ID:          "scenario-sec-abuse-001",
		Name:        "Market Abuse Detection",
		Type:        ScenarioMarketAbuse,
		Regulation:  "SEC",
		Description: "Tests detection of suspicious trading patterns",
		Severity:    "critical",
		Priority:    1,
		IsActive:    true,
		Version:     "1.0.0",
		CreatedBy:   "system",
		CreatedAt:   time.Now(),
		ValidFrom:   time.Now(),

		TestData: map[string]interface{}{
			"order_pattern": map[string]interface{}{
				"symbol":     "ACME",
				"order_type": "options",
				"side":       "buy",
				"volume":     "500% of average daily volume",
				"timing":     "48 hours before earnings announcement",
				"price_move": "+15% after order",
			},
			"corporate_event": map[string]interface{}{
				"type":   "earnings",
				"date":   "2024-01-15",
				"status": "unpublished",
				"result": "significant beat - +25% EPS",
			},
		},

		ExpectedViolations: []ExpectedViolation{
			{
				RuleID:      "SEC-MNPI-001",
				RuleName:    "Material Non-Public Information",
				Description: "Trading on unpublished earnings information",
				MustDetect:  true,
			},
			{
				RuleID:      "SEC-VOLUME-001",
				RuleName:    "Suspicious Volume",
				Description: "Trading volume significantly exceeds normal patterns",
				MustDetect:  true,
			},
		},

		ExpectedOutcome: "fail",

		Guardrails: []GuardrailRule{
			{
				ID:             "guardrail-sec-001",
				Name:           "MNPI Detection",
				Description:    "Agent must identify potential MNPI violation",
				Language:       "javascript",
				Code:           `return agentResponse.includes("non-public") || agentResponse.includes("insider") || agentResponse.includes("material");`,
				IsMandatory:    true,
				TimeoutSec:     5,
				ExpectedResult: true,
			},
			{
				ID:             "guardrail-sec-002",
				Name:           "Suspicious Timing",
				Description:    "Agent must question timing of trades",
				Language:       "javascript",
				Code:           `return agentResponse.includes("timing") || agentResponse.includes("before announcement");`,
				IsMandatory:    true,
				TimeoutSec:     5,
				ExpectedResult: true,
			},
		},
	}
}

// ToJSON serializes to JSON
func (s *ScenarioResult) ToJSON() ([]byte, error) {
	return json.MarshalIndent(s, "", "  ")
}

// FromJSON deserializes from JSON
func ScenarioResultFromJSON(data []byte) (*ScenarioResult, error) {
	var result ScenarioResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
