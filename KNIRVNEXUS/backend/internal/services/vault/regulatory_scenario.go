package vault

import (
	"encoding/json"
	"time"
)

// RegulatoryScenarioType defines the type of regulatory stress test
type RegulatoryScenarioType string

const (
	// Market stress scenarios
	ScenarioFlashCrash      RegulatoryScenarioType = "FLASH_CRASH"
	ScenarioLiquidityCrisis RegulatoryScenarioType = "LIQUIDITY_CRISIS"
	ScenarioVolatilitySpike RegulatoryScenarioType = "VOLATILITY_SPIKE"

	// Compliance violation scenarios
	ScenarioAMLViolation   RegulatoryScenarioType = "AML_VIOLATION"
	ScenarioKYCFailure     RegulatoryScenarioType = "KYC_FAILURE"
	ScenarioMarketAbuse    RegulatoryScenarioType = "MARKET_ABUSE"
	ScenarioInsiderTrading RegulatoryScenarioType = "INSIDER_TRADING"

	// Capital adequacy scenarios
	ScenarioBaselViolation     RegulatoryScenarioType = "BASEL_VIOLATION"
	ScenarioLeverageBreach     RegulatoryScenarioType = "LEVERAGE_BREACH"
	ScenarioLiquidityShortfall RegulatoryScenarioType = "LIQUIDITY_SHORTFALL"

	// Fraud patterns
	ScenarioPonziPattern RegulatoryScenarioType = "PONZI_PATTERN"
	ScenarioLayering     RegulatoryScenarioType = "LAYERING"
	ScenarioSmurfing     RegulatoryScenarioType = "SMURFING"
)

// RegulatoryScenario represents a stress test scenario stored in the vault
type RegulatoryScenario struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        RegulatoryScenarioType `json:"type"`
	Regulation  string                 `json:"regulation"` // e.g., "AML", "KYC", "SEC", "BaselIII"
	Description string                 `json:"description"`

	// Test data that will be injected
	TestData map[string]interface{} `json:"test_data"`

	// Expected outcomes
	ExpectedViolations []ExpectedViolation `json:"expected_violations"`
	ExpectedOutcome    string              `json:"expected_outcome"` // pass, fail, review

	// Guardrail logic - code that checks if agent behaves correctly
	Guardrails []GuardrailRule `json:"guardrails"`

	// Severity and priority
	Severity string `json:"severity"` // critical, high, medium, low
	Priority int    `json:"priority"`
	IsActive bool   `json:"is_active"`

	// Metadata
	Version    string     `json:"version"`
	CreatedBy  string     `json:"created_by"`
	CreatedAt  time.Time  `json:"created_at"`
	ValidFrom  time.Time  `json:"valid_from"`
	ValidUntil *time.Time `json:"valid_until,omitempty"`

	// Audit trail
	ExecutionCount int        `json:"execution_count"`
	LastExecuted   *time.Time `json:"last_executed,omitempty"`
}

// ExpectedViolation defines what violation should be detected
type ExpectedViolation struct {
	RuleID      string `json:"rule_id"`
	RuleName    string `json:"rule_name"`
	Description string `json:"description"`
	MustDetect  bool   `json:"must_detect"` // If true, agent MUST catch this
}

// GuardrailRule defines a mandatory check during agent validation
type GuardrailRule struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`

	// Logic for the guardrail
	Language string `json:"language"` // "python", "javascript", "rego", "cel"
	Code     string `json:"code"`     // The validation logic

	// Execution config
	IsMandatory bool `json:"is_mandatory"` // If true, failure stops validation
	TimeoutSec  int  `json:"timeout_sec"`

	// Expected result
	ExpectedResult interface{} `json:"expected_result"`
}

// ScenarioResult represents the outcome of running a scenario
type ScenarioResult struct {
	ScenarioID  string    `json:"scenario_id"`
	ExecutionID string    `json:"execution_id"`
	Status      string    `json:"status"` // passed, failed, error, timeout
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
	DurationMs  int64     `json:"duration_ms"`

	// Guardrail results
	GuardrailResults []GuardrailResult `json:"guardrail_results"`

	// Violation detection results
	DetectedViolations []DetectedViolation `json:"detected_violations"`
	MissedViolations   []ExpectedViolation `json:"missed_violations"` // Expected but not found
	FalsePositives     []DetectedViolation `json:"false_positives"`   // Found but not expected

	// Scoring
	ComplianceScore float64 `json:"compliance_score"` // 0-100
	GuardrailScore  float64 `json:"guardrail_score"`  // 0-100
	OverallScore    float64 `json:"overall_score"`    // Weighted average

	// Evidence
	AgentOutput    string `json:"agent_output,omitempty"`
	ExecutionTrace string `json:"execution_trace,omitempty"`
	ErrorMessage   string `json:"error_message,omitempty"`
}

// GuardrailResult represents the outcome of a single guardrail check
type GuardrailResult struct {
	GuardrailID    string      `json:"guardrail_id"`
	GuardrailName  string      `json:"guardrail_name"`
	Status         string      `json:"status"` // passed, failed, error, timeout
	ActualResult   interface{} `json:"actual_result,omitempty"`
	ExpectedResult interface{} `json:"expected_result,omitempty"`
	ExecutionTime  int64       `json:"execution_time_ms"`
	ErrorMessage   string      `json:"error_message,omitempty"`
}

// DetectedViolation represents a violation the agent actually found
type DetectedViolation struct {
	RuleID      string  `json:"rule_id"`
	RuleName    string  `json:"rule_name"`
	Confidence  float64 `json:"confidence"`
	Description string  `json:"description"`
	Evidence    string  `json:"evidence,omitempty"`
}

// ScenarioFilter provides filtering for scenario queries
type ScenarioFilter struct {
	Type        RegulatoryScenarioType `json:"type,omitempty"`
	Regulation  string                 `json:"regulation,omitempty"`
	Severity    string                 `json:"severity,omitempty"`
	IsActive    *bool                  `json:"is_active,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	ValidBefore *time.Time             `json:"valid_before,omitempty"`
	Limit       int                    `json:"limit,omitempty"`
}

// Matches checks if a scenario matches the filter criteria
func (f *ScenarioFilter) Matches(scenario *RegulatoryScenario) bool {
	if f.Type != "" && scenario.Type != f.Type {
		return false
	}
	if f.Regulation != "" && scenario.Regulation != f.Regulation {
		return false
	}
	if f.Severity != "" && scenario.Severity != f.Severity {
		return false
	}
	if f.IsActive != nil && scenario.IsActive != *f.IsActive {
		return false
	}
	if f.ValidBefore != nil && scenario.ValidFrom.After(*f.ValidBefore) {
		return false
	}
	return true
}

// ToJSON serializes the scenario to JSON
func (s *RegulatoryScenario) ToJSON() ([]byte, error) {
	return json.MarshalIndent(s, "", "  ")
}

// FromJSON deserializes a scenario from JSON
func RegulatoryScenarioFromJSON(data []byte) (*RegulatoryScenario, error) {
	var scenario RegulatoryScenario
	if err := json.Unmarshal(data, &scenario); err != nil {
		return nil, err
	}
	return &scenario, nil
}

// IsValid checks if the scenario is currently valid (not expired)
func (s *RegulatoryScenario) IsValid() bool {
	if !s.IsActive {
		return false
	}
	if s.ValidUntil != nil && time.Now().After(*s.ValidUntil) {
		return false
	}
	return time.Now().After(s.ValidFrom)
}

// IncrementExecutionCount updates the execution tracking
func (s *RegulatoryScenario) IncrementExecutionCount() {
	s.ExecutionCount++
	now := time.Now()
	s.LastExecuted = &now
}

// Clone creates a deep copy of the scenario
func (s *RegulatoryScenario) Clone() *RegulatoryScenario {
	data, _ := s.ToJSON()
	clone, _ := RegulatoryScenarioFromJSON(data)
	return clone
}
