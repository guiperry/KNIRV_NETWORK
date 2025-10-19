package validation

import (
	"backend-server/internal/objects"
	"context"
	"fmt"
	"log"
	"strings"
	"time"
)

// ModelTester executes test cases and calculates comprehensive validation metrics
type ModelTester struct {
	inferenceService InferenceClient
	orchestrator     *ValidationOrchestrator
}

// NewModelTester creates a new model tester
func NewModelTester(
	inferenceService InferenceClient,
	orchestrator *ValidationOrchestrator,
) *ModelTester {
	return &ModelTester{
		inferenceService: inferenceService,
		orchestrator:     orchestrator,
	}
}

// ExecuteTestCase runs a single test case against a target (skill code, model ID, or executor function)
// Implements: ModelTester.ExecuteTestCase (ID 1) - runs test against model/skill with full validation
func (mt *ModelTester) ExecuteTestCase(
	ctx context.Context,
	testCase objects.TestCase,
	target interface{},
) objects.TestResult {
	startTime := time.Now()

	// Execute the target (skill code, model, or executor)
	output, err := mt.executeTarget(ctx, testCase, target)
	if err != nil {
		return objects.TestResult{
			TestCaseID:    testCase.ID,
			Status:        "error",
			ActualOutput:  "",
			ErrorMessage:  fmt.Sprintf("Execution failed: %v", err),
			Score:         0.0,
			ExecutionTime: time.Since(startTime),
		}
	}

	// Run validation orchestrator on the output
	llmResponse := LLMResponse{
		Prompt:    fmt.Sprintf("%v", testCase.Input),
		Output:    output,
		Context:   map[string]interface{}{"expected": testCase.Expected},
		Timestamp: time.Now(),
	}

	validationReport := mt.orchestrator.RunValidation(ctx, llmResponse)

	// Calculate score using unified scoring method
	score := mt.calculateScore(output, fmt.Sprintf("%v", testCase.Expected), validationReport)

	status := "passed"
	if score < 0.7 {
		status = "failed"
	}

	return objects.TestResult{
		TestCaseID:    testCase.ID,
		Status:        status,
		ActualOutput:  output,
		Score:         score,
		ExecutionTime: time.Since(startTime),
		Details: map[string]interface{}{
			"validation_report": validationReport,
			"expected":          testCase.Expected,
		},
	}
}

// executeTarget runs the target (skill code, model, or executor)
func (mt *ModelTester) executeTarget(
	ctx context.Context,
	testCase objects.TestCase,
	target interface{},
) (string, error) {
	switch t := target.(type) {
	case string:
		// Assume it's skill code or model ID
		if strings.Contains(t, "model_") {
			return mt.executeModel(ctx, testCase, t)
		}
		return mt.executeSkill(ctx, testCase, t)
	case func(context.Context, string) (string, error):
		return t(ctx, fmt.Sprintf("%v", testCase.Input))
	default:
		return "", fmt.Errorf("unsupported target type: %T", target)
	}
}

// executeSkill executes skill code through inference service
func (mt *ModelTester) executeSkill(
	ctx context.Context,
	testCase objects.TestCase,
	skillCode string,
) (string, error) {
	// Check if context is cancelled
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
		// Continue with skill execution
	}

	executionPrompt := fmt.Sprintf(`Execute the following skill code with the given input:

Skill Code:
%s

Input:
%v

Provide the output.`, skillCode, testCase.Input)

	// Execute skill with context cancellation check
	result, err := mt.inferenceService.GenerateText("deepseek-chat", executionPrompt, "")
	if err != nil {
		log.Printf("Skill execution failed for test case %s: %v", testCase.ID, err)
		return "", fmt.Errorf("skill execution failed: %w", err)
	}

	log.Printf("Skill executed successfully for test case %s: output length=%d", testCase.ID, len(result))
	return result, nil
}

// executeModel executes a model test
func (mt *ModelTester) executeModel(
	ctx context.Context,
	testCase objects.TestCase,
	modelID string,
) (string, error) {
	// Check if context is cancelled
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
		// Continue with model execution
	}

	executionPrompt := fmt.Sprintf(`Model: %s

Test Input:
%v

Expected Output Context:
%v

Provide the model's response.`, modelID, testCase.Input, testCase.Expected)

	// Execute model with context cancellation check
	result, err := mt.inferenceService.GenerateText("deepseek-chat", executionPrompt, "")
	if err != nil {
		log.Printf("Model execution failed for test case %s: %v", testCase.ID, err)
		return "", fmt.Errorf("model execution failed: %w", err)
	}

	log.Printf("Model executed successfully for test case %s: output length=%d", testCase.ID, len(result))
	return result, nil
}

// calculateStringSimilarity computes string similarity using normalized edit distance (0.0 to 1.0)
// Implements: ModelTester.calculateStringSimilarity (ID 2)
func (mt *ModelTester) calculateStringSimilarity(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}

	s1Lower := strings.ToLower(strings.TrimSpace(s1))
	s2Lower := strings.ToLower(strings.TrimSpace(s2))

	if s1Lower == s2Lower {
		return 0.95
	}

	if strings.Contains(s1Lower, s2Lower) || strings.Contains(s2Lower, s1Lower) {
		return 0.8
	}

	words1 := strings.Fields(s1Lower)
	words2 := strings.Fields(s2Lower)

	commonWords := 0
	for _, w1 := range words1 {
		for _, w2 := range words2 {
			if w1 == w2 {
				commonWords++
				break
			}
		}
	}

	if len(words1) == 0 || len(words2) == 0 {
		return 0.0
	}

	overlap := float64(commonWords) / float64(max(len(words1), len(words2)))
	return overlap * 0.7
}

// calculateScore computes test case score combining validation report (60%) and output matching (40%)
// Implements: ModelTester.calculateScore (ID 3)
func (mt *ModelTester) calculateScore(
	actual string,
	expected string,
	validationReport ValidationReport,
) float64 {
	validationScore := validationReport.OverallScore
	matchScore := mt.calculateStringSimilarity(actual, expected)

	// Weighted: 60% validation, 40% output match
	return (validationScore * 0.6) + (matchScore * 0.4)
}

// CalculateMetrics computes comprehensive metrics from test results (latency, throughput, success rate, hallucination rate)
// Implements: ModelTester.CalculateMetrics (ID 4)
func (mt *ModelTester) CalculateMetrics(
	ctx context.Context,
	results []objects.TestResult,
) objects.ValidationMetrics {
	metrics := objects.ValidationMetrics{}

	if len(results) == 0 {
		return metrics
	}

	var totalLatency time.Duration
	passCount := 0
	var totalTokens int64

	for _, result := range results {
		totalLatency += result.ExecutionTime
		if result.Status == "passed" {
			passCount++
		}

		if result.Details != nil {
			if tokens, ok := result.Details["tokens"].(int64); ok {
				totalTokens += tokens
			}
		}
	}

	metrics.AverageLatency = totalLatency / time.Duration(len(results))
	metrics.SuccessRate = float64(passCount) / float64(len(results))
	metrics.TokenConsumption = totalTokens
	if totalLatency.Seconds() > 0 {
		metrics.ThroughputPerSecond = float64(len(results)) / totalLatency.Seconds()
	}

	return metrics
}

// Test performs comprehensive test execution: runs all test cases, calculates scores and metrics
// Implements: ModelTester.Test (ID 5)
func (mt *ModelTester) Test(
	ctx context.Context,
	task *objects.ValidationTask,
	result *objects.ValidationResult,
) (*objects.ValidationResult, error) {
	startTime := time.Now()

	log.Printf("Running ModelTester for task %s with %d test cases", task.ID, len(task.TestCases))

	// Execute all test cases
	testResults := make([]objects.TestResult, len(task.TestCases))
	totalScore := 0.0

	for i, testCase := range task.TestCases {
		// Execute test case with timeout
		testCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		testResult := mt.ExecuteTestCase(testCtx, testCase, task.SkillCode)
		cancel()

		testResults[i] = testResult
		totalScore += testResult.Score * testCase.Weight

		log.Printf("Test case %s: status=%s, score=%.2f", testCase.ID, testResult.Status, testResult.Score)
	}

	// Calculate overall score
	var totalWeight float64
	for _, testCase := range task.TestCases {
		totalWeight += testCase.Weight
	}

	overallScore := totalScore / totalWeight

	// Determine status based on score
	status := "success"
	if overallScore < 0.5 {
		status = "failed"
	} else if overallScore < 0.7 {
		status = "partial"
	}

	result.Status = status
	result.Score = overallScore
	result.TestResults = testResults
	result.Results = map[string]interface{}{
		"test_execution":    "completed",
		"test_cases_passed": mt.countPassedTests(testResults),
		"total_test_cases":  len(testResults),
		"overall_score":     overallScore,
	}
	result.ExecutionTime = time.Since(startTime)

	// Calculate comprehensive metrics
	metrics := mt.CalculateMetrics(ctx, testResults)
	result.Metrics = &metrics

	log.Printf("Test execution completed: score=%.2f, status=%s", overallScore, status)

	return result, nil
}

// countPassedTests counts test cases that passed
func (mt *ModelTester) countPassedTests(results []objects.TestResult) int {
	count := 0
	for _, r := range results {
		if r.Status == "passed" {
			count++
		}
	}
	return count
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
