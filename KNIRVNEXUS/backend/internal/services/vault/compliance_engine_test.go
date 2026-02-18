package vault

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComplianceScriptExecutor(t *testing.T) {
	executor := NewComplianceScriptExecutor()
	require.NotNil(t, executor, "Should create executor")

	// Test default configuration
	assert.Equal(t, 5*time.Second, executor.defaultTimeout)
	assert.True(t, executor.IsLanguageEnabled("javascript"))
	assert.False(t, executor.IsLanguageEnabled("python"))
	assert.False(t, executor.IsLanguageEnabled("shell"))
}

func TestComplianceScriptExecutorLanguageConfig(t *testing.T) {
	executor := NewComplianceScriptExecutor()

	// Test enabling/disabling languages
	executor.SetLanguageEnabled("python", true)
	assert.True(t, executor.IsLanguageEnabled("python"))

	executor.SetLanguageEnabled("javascript", false)
	assert.False(t, executor.IsLanguageEnabled("javascript"))

	// Case insensitive
	assert.True(t, executor.IsLanguageEnabled("Python"))
	assert.False(t, executor.IsLanguageEnabled("JavaScript"))
}

func TestExecuteGuardrailJavaScript(t *testing.T) {
	executor := NewComplianceScriptExecutor()

	tests := []struct {
		name           string
		guardrail      *GuardrailRule
		testData       map[string]interface{}
		agentOutput    string
		expectedStatus string
		expectError    bool
	}{
		{
			name: "simple boolean check - pass",
			guardrail: &GuardrailRule{
				ID:             "gr-001",
				Name:           "Check True",
				Language:       "javascript",
				Code:           `return true;`,
				IsMandatory:    true,
				ExpectedResult: true,
			},
			testData:       map[string]interface{}{},
			agentOutput:    "test output",
			expectedStatus: "passed",
		},
		{
			name: "simple boolean check - fail",
			guardrail: &GuardrailRule{
				ID:             "gr-002",
				Name:           "Check False",
				Language:       "javascript",
				Code:           `return false;`,
				IsMandatory:    false,
				ExpectedResult: true,
			},
			testData:       map[string]interface{}{},
			agentOutput:    "test output",
			expectedStatus: "failed",
		},
		{
			name: "check agent response content - pass",
			guardrail: &GuardrailRule{
				ID:             "gr-003",
				Name:           "Check Risk Mention",
				Language:       "javascript",
				Code:           `return agentResponse.includes("risk");`,
				IsMandatory:    true,
				ExpectedResult: true,
			},
			testData:       map[string]interface{}{},
			agentOutput:    "This is a high risk situation",
			expectedStatus: "passed",
		},
		{
			name: "check agent response content - fail",
			guardrail: &GuardrailRule{
				ID:             "gr-004",
				Name:           "Check Risk Mention",
				Language:       "javascript",
				Code:           `return agentResponse.includes("risk");`,
				IsMandatory:    true,
				ExpectedResult: true,
			},
			testData:       map[string]interface{}{},
			agentOutput:    "This is safe",
			expectedStatus: "failed",
		},
		{
			name: "check test data - pass",
			guardrail: &GuardrailRule{
				ID:             "gr-005",
				Name:           "Check Amount",
				Language:       "javascript",
				Code:           `return amount > 5000;`,
				IsMandatory:    true,
				ExpectedResult: true,
			},
			testData: map[string]interface{}{
				"amount": 10000,
			},
			agentOutput:    "",
			expectedStatus: "passed",
		},
		{
			name: "disabled language",
			guardrail: &GuardrailRule{
				ID:             "gr-006",
				Name:           "Python Check",
				Language:       "python",
				Code:           `return True`,
				IsMandatory:    true,
				ExpectedResult: true,
			},
			testData:       map[string]interface{}{},
			agentOutput:    "",
			expectedStatus: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := executor.ExecuteGuardrail(ctx, tt.guardrail, tt.testData, tt.agentOutput)

			if tt.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, result.Status)
			assert.Equal(t, tt.guardrail.ID, result.GuardrailID)
			assert.GreaterOrEqual(t, result.ExecutionTime, int64(0))
		})
	}
}

func TestExecuteGuardrailTimeout(t *testing.T) {
	executor := NewComplianceScriptExecutor()

	guardrail := &GuardrailRule{
		ID:             "gr-timeout",
		Name:           "Infinite Loop",
		Language:       "javascript",
		Code:           `while(true) {}`,
		IsMandatory:    true,
		TimeoutSec:     1, // 1 second timeout
		ExpectedResult: true,
	}

	ctx := context.Background()
	result, err := executor.ExecuteGuardrail(ctx, guardrail, map[string]interface{}{}, "")

	require.NoError(t, err)
	assert.Equal(t, "timeout", result.Status)
	assert.Contains(t, result.ErrorMessage, "timeout")
}

func TestExecuteScenario(t *testing.T) {
	executor := NewComplianceScriptExecutor()

	scenario := &RegulatoryScenario{
		ID:   "test-scenario",
		Name: "Test Scenario",
		Type: ScenarioFlashCrash,
		TestData: map[string]interface{}{
			"market_data": map[string]interface{}{
				"drop": "-10%",
			},
		},
		Guardrails: []GuardrailRule{
			{
				ID:             "gr-001",
				Name:           "Check Drop",
				Language:       "javascript",
				Code:           `return market_data.drop.includes("-");`,
				IsMandatory:    true,
				ExpectedResult: true,
			},
			{
				ID:             "gr-002",
				Name:           "Check Response",
				Language:       "javascript",
				Code:           `return agentResponse.includes("alert");`,
				IsMandatory:    false,
				ExpectedResult: true,
			},
		},
	}

	ctx := context.Background()

	t.Run("all guardrails pass", func(t *testing.T) {
		result, err := executor.ExecuteScenario(ctx, scenario, "alert: market crash detected")
		require.NoError(t, err)
		assert.Equal(t, "passed", result.Status)
		assert.Len(t, result.GuardrailResults, 2)
		assert.Equal(t, 100.0, result.GuardrailScore) // All passed
	})

	t.Run("mandatory guardrail fails", func(t *testing.T) {
		result, err := executor.ExecuteScenario(ctx, scenario, "everything is fine")
		require.NoError(t, err)
		assert.Equal(t, "failed", result.Status)
		assert.Len(t, result.GuardrailResults, 2)
		// First guardrail should pass (checking market data), second should fail
	})
}

func TestResultsMatch(t *testing.T) {
	executor := NewComplianceScriptExecutor()

	tests := []struct {
		name     string
		actual   interface{}
		expected interface{}
		match    bool
	}{
		{"both nil", nil, nil, true},
		{"actual nil", nil, true, false},
		{"expected nil", true, nil, false},
		{"bool true-true", true, true, true},
		{"bool true-false", true, false, false},
		{"string match", "hello", "hello", true},
		{"string no match", "hello", "world", false},
		{"int match", 42, 42, true},
		{"int no match", 42, 43, false},
		{"int64 match", int64(42), int64(42), true},
		{"float64 match", 3.14, 3.14, true},
		{"int to float64", 42, 42.0, true},
		{"float64 to int", 42.0, 42, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := executor.resultsMatch(tt.actual, tt.expected)
			assert.Equal(t, tt.match, result)
		})
	}
}

func TestComplianceEngineInitialization(t *testing.T) {
	// Note: This test uses nil repository since we're testing initialization only
	config := &ComplianceConfig{
		MaxConcurrentExecutions: 5,
		DefaultTimeout:          10 * time.Second,
		PassThreshold:           75.0,
	}

	engine := NewComplianceEngine(nil, config)
	require.NotNil(t, engine)

	stats := engine.GetStatistics()
	assert.NotNil(t, stats)
	assert.Equal(t, int64(0), stats.TotalExecutions)
}

func TestDefaultComplianceConfig(t *testing.T) {
	config := DefaultComplianceConfig()

	assert.Equal(t, 10, config.MaxConcurrentExecutions)
	assert.Equal(t, 30*time.Second, config.DefaultTimeout)
	assert.True(t, config.EnableCaching)
	assert.Equal(t, 1*time.Hour, config.CacheTTL)
	assert.Equal(t, 0.4, config.GuardrailWeight)
	assert.Equal(t, 0.4, config.ViolationWeight)
	assert.Equal(t, 0.2, config.MandatoryPenalty)
	assert.Equal(t, 80.0, config.PassThreshold)
	assert.Equal(t, 60.0, config.ReviewThreshold)
}

func TestComplianceEngineStatistics(t *testing.T) {
	defaultConfig := DefaultComplianceConfig()
	engine := NewComplianceEngine(nil, &defaultConfig)

	// Initial stats should be zero
	stats := engine.GetStatistics()
	assert.Equal(t, int64(0), stats.TotalExecutions)

	// Update with a result
	result := &ScenarioResult{
		ScenarioID:  "test-001",
		Status:      "passed",
		DurationMs:  100,
		CompletedAt: time.Now(),
	}
	engine.updateStatistics(result)

	// Stats should reflect the update
	stats = engine.GetStatistics()
	assert.Equal(t, int64(1), stats.TotalExecutions)
	assert.Equal(t, int64(1), stats.PassedExecutions)
	assert.Equal(t, int64(100), stats.AverageExecutionTime)

	// Add more results
	engine.updateStatistics(&ScenarioResult{ScenarioID: "test-002", Status: "failed", DurationMs: 200, CompletedAt: time.Now()})
	engine.updateStatistics(&ScenarioResult{ScenarioID: "test-003", Status: "error", DurationMs: 50, CompletedAt: time.Now()})

	stats = engine.GetStatistics()
	assert.Equal(t, int64(3), stats.TotalExecutions)
	assert.Equal(t, int64(1), stats.PassedExecutions)
	assert.Equal(t, int64(1), stats.FailedExecutions)
	assert.Equal(t, int64(1), stats.ErrorExecutions)

	// Reset statistics
	engine.ResetStatistics()
	stats = engine.GetStatistics()
	assert.Equal(t, int64(0), stats.TotalExecutions)
}

func TestCalculateComplianceScore(t *testing.T) {
	defaultConfig := DefaultComplianceConfig()
	engine := NewComplianceEngine(nil, &defaultConfig)

	scenario := &RegulatoryScenario{
		ExpectedViolations: []ExpectedViolation{
			{RuleID: "rule-001", MustDetect: true},
			{RuleID: "rule-002", MustDetect: true},
			{RuleID: "rule-003", MustDetect: false},
		},
	}

	result := &ScenarioResult{
		DetectedViolations: []DetectedViolation{
			{RuleID: "rule-001"},
			{RuleID: "rule-002"},
		},
	}

	score := engine.calculateComplianceScore(scenario, result)
	assert.Greater(t, score, 0.0)
	assert.LessOrEqual(t, score, 100.0)
}

func TestDetermineStatus(t *testing.T) {
	defaultConfig := DefaultComplianceConfig()
	engine := NewComplianceEngine(nil, &defaultConfig)

	tests := []struct {
		name   string
		report *ValidationReport
		status string
	}{
		{
			name:   "critical failures",
			report: &ValidationReport{CriticalFailures: 1, OverallScore: 90.0},
			status: "failed",
		},
		{
			name:   "passed with high score",
			report: &ValidationReport{CriticalFailures: 0, OverallScore: 85.0},
			status: "passed",
		},
		{
			name:   "review threshold",
			report: &ValidationReport{CriticalFailures: 0, OverallScore: 65.0},
			status: "review",
		},
		{
			name:   "failed low score",
			report: &ValidationReport{CriticalFailures: 0, OverallScore: 50.0},
			status: "failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := engine.determineStatus(tt.report)
			assert.Equal(t, tt.status, status)
		})
	}
}

func TestGenerateSummary(t *testing.T) {
	defaultConfig := DefaultComplianceConfig()
	engine := NewComplianceEngine(nil, &defaultConfig)

	tests := []struct {
		name            string
		report          *ValidationReport
		containsText    []string
		notContainsText []string
	}{
		{
			name: "passed status",
			report: &ValidationReport{
				Status:       "passed",
				AgentID:      "agent-001",
				OverallScore: 85.0,
			},
			containsText: []string{"passed", "agent-001", "85.0"},
		},
		{
			name: "failed status",
			report: &ValidationReport{
				Status:            "failed",
				AgentID:           "agent-002",
				OverallScore:      45.0,
				CriticalFailures:  2,
				MandatoryFailures: 3,
			},
			containsText: []string{"FAILED", "agent-002", "45.0", "2 critical", "3 mandatory"},
		},
		{
			name: "review status",
			report: &ValidationReport{
				Status:       "review",
				AgentID:      "agent-003",
				OverallScore: 65.0,
			},
			containsText: []string{"manual review", "agent-003", "65.0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := engine.generateSummary(tt.report)
			for _, text := range tt.containsText {
				assert.Contains(t, summary, text)
			}
		})
	}
}

func TestGenerateRecommendations(t *testing.T) {
	defaultConfig := DefaultComplianceConfig()
	engine := NewComplianceEngine(nil, &defaultConfig)

	tests := []struct {
		name               string
		report             *ValidationReport
		minRecommendations int
	}{
		{
			name: "passed with high score",
			report: &ValidationReport{
				Status:       "passed",
				OverallScore: 95.0,
			},
			minRecommendations: 1,
		},
		{
			name: "critical failures",
			report: &ValidationReport{
				Status:           "failed",
				CriticalFailures: 2,
			},
			minRecommendations: 1,
		},
		{
			name: "guardrail failures",
			report: &ValidationReport{
				Status:            "failed",
				MandatoryFailures: 3,
			},
			minRecommendations: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recommendations := engine.generateRecommendations(tt.report)
			assert.GreaterOrEqual(t, len(recommendations), tt.minRecommendations)
		})
	}
}
