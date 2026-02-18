package vault

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegulatoryScenarioTypes(t *testing.T) {
	// Test that all scenario types are defined
	scenarios := []RegulatoryScenarioType{
		ScenarioFlashCrash,
		ScenarioLiquidityCrisis,
		ScenarioVolatilitySpike,
		ScenarioAMLViolation,
		ScenarioKYCFailure,
		ScenarioMarketAbuse,
		ScenarioInsiderTrading,
		ScenarioBaselViolation,
		ScenarioLeverageBreach,
		ScenarioLiquidityShortfall,
		ScenarioPonziPattern,
		ScenarioLayering,
		ScenarioSmurfing,
	}

	for _, s := range scenarios {
		assert.NotEmpty(t, s, "Scenario type should not be empty")
	}
}

func TestRegulatoryScenarioValidation(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		scenario *RegulatoryScenario
		expected bool
	}{
		{
			name: "active and valid",
			scenario: &RegulatoryScenario{
				IsActive:   true,
				ValidFrom:  now.Add(-1 * time.Hour),
				ValidUntil: nil,
			},
			expected: true,
		},
		{
			name: "inactive",
			scenario: &RegulatoryScenario{
				IsActive:   false,
				ValidFrom:  now.Add(-1 * time.Hour),
				ValidUntil: nil,
			},
			expected: false,
		},
		{
			name: "expired",
			scenario: &RegulatoryScenario{
				IsActive:   true,
				ValidFrom:  now.Add(-2 * time.Hour),
				ValidUntil: timePtr(now.Add(-1 * time.Hour)),
			},
			expected: false,
		},
		{
			name: "not yet valid",
			scenario: &RegulatoryScenario{
				IsActive:   true,
				ValidFrom:  now.Add(1 * time.Hour),
				ValidUntil: nil,
			},
			expected: false,
		},
		{
			name: "valid with future expiration",
			scenario: &RegulatoryScenario{
				IsActive:   true,
				ValidFrom:  now.Add(-1 * time.Hour),
				ValidUntil: timePtr(now.Add(24 * time.Hour)),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.scenario.IsValid()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestScenarioClone(t *testing.T) {
	original := &RegulatoryScenario{
		ID:          "test-scenario-001",
		Name:        "Test Scenario",
		Type:        ScenarioFlashCrash,
		Regulation:  "SEC",
		Description: "Test description",
		Severity:    "critical",
		Priority:    1,
		IsActive:    true,
		Version:     "1.0.0",
		CreatedBy:   "test",
		CreatedAt:   time.Now(),
		ValidFrom:   time.Now(),
		TestData: map[string]interface{}{
			"key1": "value1",
			"key2": 123,
		},
		ExpectedViolations: []ExpectedViolation{
			{RuleID: "RULE-001", RuleName: "Test Rule", MustDetect: true},
		},
		Guardrails: []GuardrailRule{
			{ID: "GR-001", Name: "Test Guardrail", Language: "javascript"},
		},
	}

	clone := original.Clone()

	// Verify clone has same values
	assert.Equal(t, original.ID, clone.ID)
	assert.Equal(t, original.Name, clone.Name)
	assert.Equal(t, original.Type, clone.Type)
	assert.Equal(t, len(original.TestData), len(clone.TestData))
	assert.Equal(t, len(original.ExpectedViolations), len(clone.ExpectedViolations))
	assert.Equal(t, len(original.Guardrails), len(clone.Guardrails))

	// Verify clone is independent
	clone.Name = "Modified Name"
	assert.NotEqual(t, original.Name, clone.Name)
}

func TestScenarioFilter(t *testing.T) {
	scenario := &RegulatoryScenario{
		ID:         "test-001",
		Type:       ScenarioFlashCrash,
		Regulation: "SEC",
		Severity:   "critical",
		IsActive:   true,
		ValidFrom:  time.Now().Add(-1 * time.Hour),
	}

	tests := []struct {
		name     string
		filter   *ScenarioFilter
		expected bool
	}{
		{
			name:     "match by type",
			filter:   &ScenarioFilter{Type: ScenarioFlashCrash},
			expected: true,
		},
		{
			name:     "no match by type",
			filter:   &ScenarioFilter{Type: ScenarioAMLViolation},
			expected: false,
		},
		{
			name:     "match by regulation",
			filter:   &ScenarioFilter{Regulation: "SEC"},
			expected: true,
		},
		{
			name:     "match by severity",
			filter:   &ScenarioFilter{Severity: "critical"},
			expected: true,
		},
		{
			name:     "match by active status",
			filter:   &ScenarioFilter{IsActive: boolPtr(true)},
			expected: true,
		},
		{
			name:     "no match by inactive status",
			filter:   &ScenarioFilter{IsActive: boolPtr(false)},
			expected: false,
		},
		{
			name:     "match all criteria",
			filter:   &ScenarioFilter{Type: ScenarioFlashCrash, Regulation: "SEC", Severity: "critical"},
			expected: true,
		},
		{
			name:     "empty filter matches all",
			filter:   &ScenarioFilter{},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.filter.Matches(scenario)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetDefaultRegulatoryScenarios(t *testing.T) {
	scenarios := GetDefaultRegulatoryScenarios()
	require.NotEmpty(t, scenarios, "Should return default scenarios")
	assert.GreaterOrEqual(t, len(scenarios), 5, "Should have at least 5 default scenarios")

	// Verify each scenario has required fields
	for _, s := range scenarios {
		assert.NotEmpty(t, s.ID, "Scenario should have ID")
		assert.NotEmpty(t, s.Name, "Scenario should have name")
		assert.NotEmpty(t, s.Type, "Scenario should have type")
		assert.NotEmpty(t, s.Regulation, "Scenario should have regulation")
		assert.NotEmpty(t, s.Severity, "Scenario should have severity")
		assert.True(t, s.IsActive, "Default scenarios should be active")
		assert.NotEmpty(t, s.Guardrails, "Scenario should have guardrails")
		assert.NotEmpty(t, s.ExpectedViolations, "Scenario should have expected violations")
	}

	// Check for specific scenarios
	scenarioIDs := make(map[string]bool)
	for _, s := range scenarios {
		scenarioIDs[s.ID] = true
	}

	assert.True(t, scenarioIDs["scenario-flash-crash-001"], "Should have flash crash scenario")
	assert.True(t, scenarioIDs["scenario-basel-violation-001"], "Should have Basel violation scenario")
	assert.True(t, scenarioIDs["scenario-aml-structuring-001"], "Should have AML structuring scenario")
}

func TestScenarioJSONSerialization(t *testing.T) {
	scenario := &RegulatoryScenario{
		ID:        "test-json-001",
		Name:      "Test JSON",
		Type:      ScenarioFlashCrash,
		CreatedAt: time.Now(),
		TestData: map[string]interface{}{
			"amount": 1000.50,
			"count":  42,
		},
	}

	// Test serialization
	data, err := scenario.ToJSON()
	require.NoError(t, err, "Should serialize to JSON")
	assert.NotEmpty(t, data, "JSON should not be empty")

	// Test deserialization
	restored, err := RegulatoryScenarioFromJSON(data)
	require.NoError(t, err, "Should deserialize from JSON")
	assert.Equal(t, scenario.ID, restored.ID)
	assert.Equal(t, scenario.Name, restored.Name)
	assert.Equal(t, scenario.Type, restored.Type)
}

func TestScenarioResultJSON(t *testing.T) {
	result := &ScenarioResult{
		ScenarioID:  "test-001",
		ExecutionID: "exec-001",
		Status:      "passed",
		StartedAt:   time.Now(),
		CompletedAt: time.Now(),
		DurationMs:  100,
		GuardrailResults: []GuardrailResult{
			{GuardrailID: "gr-001", Status: "passed"},
		},
		DetectedViolations: []DetectedViolation{
			{RuleID: "rule-001", Confidence: 0.95},
		},
		ComplianceScore: 85.5,
		GuardrailScore:  90.0,
		OverallScore:    87.5,
	}

	// Test serialization
	data, err := result.ToJSON()
	require.NoError(t, err, "Should serialize result to JSON")

	// Test deserialization
	restored, err := ScenarioResultFromJSON(data)
	require.NoError(t, err, "Should deserialize result from JSON")
	assert.Equal(t, result.ScenarioID, restored.ScenarioID)
	assert.Equal(t, result.Status, restored.Status)
	assert.Equal(t, result.OverallScore, restored.OverallScore)
	assert.Len(t, restored.GuardrailResults, 1)
	assert.Len(t, restored.DetectedViolations, 1)
}

func TestIncrementExecutionCount(t *testing.T) {
	scenario := &RegulatoryScenario{
		ID:             "test-exec",
		ExecutionCount: 0,
	}

	assert.Equal(t, 0, scenario.ExecutionCount)
	assert.Nil(t, scenario.LastExecuted)

	scenario.IncrementExecutionCount()

	assert.Equal(t, 1, scenario.ExecutionCount)
	assert.NotNil(t, scenario.LastExecuted)
	assert.WithinDuration(t, time.Now(), *scenario.LastExecuted, time.Second)
}

// Helper functions
func timePtr(t time.Time) *time.Time {
	return &t
}

func boolPtr(b bool) *bool {
	return &b
}
