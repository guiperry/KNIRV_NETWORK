package validation

import (
	"backend_server/internal/objects"
	"context"
	"fmt"
	"strings"
	"time"
)

// TestCaseExecutor executes individual test cases
type TestCaseExecutor struct {
	inferenceService InferenceClient
	orchestrator     *ValidationOrchestrator
}

// NewTestCaseExecutor creates a new test case executor
func NewTestCaseExecutor(
	inferenceService InferenceClient,
	orchestrator *ValidationOrchestrator,
) *TestCaseExecutor {
	return &TestCaseExecutor{
		inferenceService: inferenceService,
		orchestrator:     orchestrator,
	}
}

// ExecuteTestCase runs a single test case against skill code
func (tce *TestCaseExecutor) ExecuteTestCase(
	ctx context.Context,
	testCase objects.TestCase,
	skillCode string,
) objects.TestResult {
	startTime := time.Now()

	// Step 1: Execute the skill code with test input
	executionPrompt := fmt.Sprintf(`Execute the following skill code with the given input:

Skill Code:
%s

Input:
%s

Provide the output.`, skillCode, testCase.Input)

	output, err := tce.inferenceService.GenerateText("deepseek-chat", executionPrompt, "")
	if err != nil {
		return objects.TestResult{
			TestCaseID:    testCase.ID,
			Status:        "error",
			ActualOutput:  "Execution failed",
			ErrorMessage:  fmt.Sprintf("Execution failed: %v", err),
			Score:         0.0,
			ExecutionTime: time.Since(startTime),
		}
	}

	// Step 2: Run validation orchestrator on the output
	llmResponse := LLMResponse{
		Prompt:    fmt.Sprintf("%v", testCase.Input),
		Output:    output,
		Context:   map[string]interface{}{"expected": testCase.Expected},
		Timestamp: time.Now(),
	}

	validationReport := tce.orchestrator.RunValidation(ctx, llmResponse)

	// Step 3: Compare output with expected result
	score := tce.calculateScore(output, testCase.Expected, validationReport)

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
	}
}

// calculateScore computes the test case score
func (tce *TestCaseExecutor) calculateScore(
	actual string,
	expected string,
	validationReport ValidationReport,
) float64 {
	// Combine validation score with output matching
	validationScore := validationReport.OverallScore

	// Simple string similarity (can be enhanced with semantic similarity)
	matchScore := tce.calculateStringSimilarity(actual, expected)

	// Weighted combination: 60% validation, 40% output match
	finalScore := (validationScore * 0.6) + (matchScore * 0.4)

	return finalScore
}

// calculateStringSimilarity computes string similarity (0.0 to 1.0)
func (tce *TestCaseExecutor) calculateStringSimilarity(s1, s2 string) float64 {
	// Simple implementation - can be enhanced with Levenshtein distance
	if s1 == s2 {
		return 1.0
	}

	// Normalize and compare
	s1Lower := strings.ToLower(strings.TrimSpace(s1))
	s2Lower := strings.ToLower(strings.TrimSpace(s2))

	if s1Lower == s2Lower {
		return 0.95
	}

	// Check if one contains the other
	if strings.Contains(s1Lower, s2Lower) || strings.Contains(s2Lower, s1Lower) {
		return 0.8
	}

	// Calculate word overlap
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
	return overlap * 0.7 // Scale down for partial matches
}
