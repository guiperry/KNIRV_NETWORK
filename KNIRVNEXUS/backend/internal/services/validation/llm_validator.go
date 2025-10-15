package validation

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ============================================================================
// EXTERNAL LLM-BASED VALIDATORS (Requires Implementation)
// ============================================================================

// LLMEvaluatorClient interface for external LLM calls
// IMPLEMENTATION NOTE: You need to implement this interface with your LLM provider
// Examples: OpenAI API, Anthropic Claude API, local model via Ollama, etc.
type LLMEvaluatorClient interface {
	// EvaluateReasoning sends a prompt to an LLM to evaluate reasoning
	// Returns: (score 0-1, explanation, error)
	EvaluateReasoning(ctx context.Context, prompt, output, criteria string) (float64, string, error)
	
	// CheckFactualClaim verifies a factual statement
	// Returns: (isAccurate, explanation, error)
	CheckFactualClaim(ctx context.Context, claim string) (bool, string, error)
}

// MockLLMEvaluator - Replace this with real implementation
type MockLLMEvaluator struct{}

func (m *MockLLMEvaluator) EvaluateReasoning(ctx context.Context, prompt, output, criteria string) (float64, string, error) {
	// TODO: Implement actual LLM API call here
	// Example with OpenAI:
	// - Format prompt with criteria
	// - Call OpenAI completion API
	// - Parse response for score and explanation
	
	// Mock implementation
	if strings.Contains(output, "step") || strings.Contains(output, "because") {
		return 0.85, "Output shows logical structure with reasoning markers", nil
	}
	return 0.4, "Output lacks clear reasoning structure", nil
}

func (m *MockLLMEvaluator) CheckFactualClaim(ctx context.Context, claim string) (bool, string, error) {
	// TODO: Implement actual fact-checking API call
	// Options:
	// - Use LLM with web search capability
	// - Call fact-checking API (e.g., Google Fact Check API)
	// - Query knowledge base
	
	// Mock implementation
	if strings.Contains(strings.ToLower(claim), "mars has oceans") {
		return false, "Mars does not have liquid surface oceans", nil
	}
	return true, "Claim appears factually consistent", nil
}

// LLMReasoningValidator uses an external LLM to evaluate reasoning quality
type LLMReasoningValidator struct {
	Client         LLMEvaluatorClient
	CriteriaPrompt string
	MinScore       float64
}

func (lv *LLMReasoningValidator) Name() string {
	return "LLMReasoningValidator"
}

func (lv *LLMReasoningValidator) Priority() int {
	return 60
}

func (lv *LLMReasoningValidator) Validate(ctx context.Context, response LLMResponse) ValidationResult {
	start := time.Now()
	
	score, explanation, err := lv.Client.EvaluateReasoning(
		ctx,
		response.Prompt,
		response.Output,
		lv.CriteriaPrompt,
	)
	
	if err != nil {
		return ValidationResult{
			ValidatorName: lv.Name(),
			IsValid:       false,
			Confidence:    0.0,
			Message:       fmt.Sprintf("LLM evaluation failed: %v", err),
			Duration:      time.Since(start),
		}
	}

	isValid := score >= lv.MinScore

	return ValidationResult{
		ValidatorName: lv.Name(),
		IsValid:       isValid,
		Confidence:    score,
		Message:       explanation,
		Details: map[string]interface{}{
			"score":     score,
			"min_score": lv.MinScore,
		},
		Duration: time.Since(start),
	}
}

// FactualAccuracyValidator uses external LLM or API to check facts
type FactualAccuracyValidator struct {
	Client LLMEvaluatorClient
}

func (fv *FactualAccuracyValidator) Name() string {
	return "FactualAccuracyValidator"
}

func (fv *FactualAccuracyValidator) Priority() int {
	return 110
}

func (fv *FactualAccuracyValidator) Validate(ctx context.Context, response LLMResponse) ValidationResult {
	start := time.Now()
	
	// Extract claims from output (simplified - in production, use NLP)
	claims := extractClaims(response.Output)
	
	inaccuracies := []string{}
	for _, claim := range claims {
		isAccurate, explanation, err := fv.Client.CheckFactualClaim(ctx, claim)
		if err != nil {
			return ValidationResult{
				ValidatorName: fv.Name(),
				IsValid:       false,
				Confidence:    0.0,
				Message:       fmt.Sprintf("Fact-checking failed: %v", err),
				Duration:      time.Since(start),
			}
		}
		
		if !isAccurate {
			inaccuracies = append(inaccuracies, fmt.Sprintf("%s: %s", claim, explanation))
		}
	}

	isValid := len(inaccuracies) == 0
	confidence := 1.0
	if !isValid {
		confidence = 0.0
	}

	return ValidationResult{
		ValidatorName: fv.Name(),
		IsValid:       isValid,
		Confidence:    confidence,
		Message:       fmt.Sprintf("Fact check: %d inaccuracies found", len(inaccuracies)),
		Details: map[string]interface{}{
			"inaccuracies": inaccuracies,
			"claims_checked": len(claims),
		},
		Duration: time.Since(start),
	}
}

// Simple claim extraction (replace with proper NLP)
func extractClaims(output string) []string {
	sentences := strings.Split(output, ".")
	claims := []string{}
	for _, s := range sentences {
		s = strings.TrimSpace(s)
		if len(s) > 20 { // Basic heuristic
			claims = append(claims, s)
		}
	}
	return claims
}

// ============================================================================
// MAIN EXAMPLE
// ============================================================================

func main() {
	ctx := context.Background()

	// Create orchestrator
	orchestrator := NewValidationOrchestrator(false, 0.7)

	// Add deterministic validators
	orchestrator.AddValidator(&KeywordPresenceValidator{
		RequiredKeywords: []string{"water", "cycle", "evaporation"},
		CaseSensitive:    false,
	})

	orchestrator.AddValidator(&OutputLengthValidator{
		MinWords: 20,
		MaxWords: 200,
	})

	orchestrator.AddValidator(&ForbiddenContentValidator{
		ForbiddenPatterns: []string{"unicorn", "magic", "definitely", "absolutely"},
		UseRegex:          false,
	})

	orchestrator.AddValidator(&ContradictionDetector{
		ContradictionPairs: [][]string{
			{"always", "never"},
			{"all", "none"},
			{"true", "false"},
		},
	})

	orchestrator.AddValidator(&StructuralPatternValidator{
		RequiredPatterns: []string{"evaporate", "condense", "precipit"},
		MinOccurrences:   1,
	})

	// Add LLM-based validators (optional - requires implementation)
	mockLLM := &MockLLMEvaluator{}
	
	orchestrator.AddValidator(&LLMReasoningValidator{
		Client: mockLLM,
		CriteriaPrompt: "Evaluate if the explanation is logically structured with clear cause-and-effect relationships",
		MinScore: 0.7,
	})

	orchestrator.AddValidator(&FactualAccuracyValidator{
		Client: mockLLM,
	})

	// Test responses
	goodResponse := LLMResponse{
		Prompt: "Explain the water cycle",
		Output: "The water cycle is a continuous process where water evaporates from Earth's surface, rises into the atmosphere, cools and condenses into clouds, and eventually falls back as precipitation like rain or snow.",
		Timestamp: time.Now(),
	}

	badResponse := LLMResponse{
		Prompt: "Explain the water cycle",
		Output: "Water definitely always stays in one place and never moves. It's absolutely true that water doesn't evaporate.",
		Timestamp: time.Now(),
	}

	// Run validations
	fmt.Println("=== VALIDATING GOOD RESPONSE ===")
	report1 := orchestrator.RunValidation(ctx, goodResponse)
	printReport(report1)

	fmt.Println("\n=== VALIDATING BAD RESPONSE ===")
	report2 := orchestrator.RunValidation(ctx, badResponse)
	printReport(report2)
}

func printReport(report ValidationReport) {
	fmt.Printf("Overall Valid: %t\n", report.OverallValid)
	fmt.Printf("Overall Score: %.2f\n", report.OverallScore)
	fmt.Printf("Execution Time: %v\n\n", report.ExecutionTime)

	for _, result := range report.Results {
		status := "✓ PASS"
		if !result.IsValid {
			status = "✗ FAIL"
		}
		fmt.Printf("%s [%s] Confidence: %.2f, Duration: %v\n",
			status, result.ValidatorName, result.Confidence, result.Duration)
		fmt.Printf("  Message: %s\n", result.Message)
		if len(result.Details) > 0 {
			detailsJSON, _ := json.MarshalIndent(result.Details, "  ", "  ")
			fmt.Printf("  Details: %s\n", string(detailsJSON))
		}
		fmt.Println()
	}

	if !report.OverallValid && len(report.FailureReasons) > 0 {
		fmt.Println("Failure Reasons:")
		for i, reason := range report.FailureReasons {
			fmt.Printf("  %d. %s\n", i+1, reason)
		}
	}
}
