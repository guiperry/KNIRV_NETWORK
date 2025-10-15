package main

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// ============================================================================
// CORE DATA STRUCTURES
// ============================================================================

// LLMResponse encapsulates the input and output for validation
type LLMResponse struct {
	Prompt    string
	Output    string
	Context   map[string]interface{} // Additional metadata
	Timestamp time.Time
}

// ValidationResult holds the outcome of a validation check
type ValidationResult struct {
	ValidatorName string
	IsValid       bool
	Confidence    float64 // 0.0 to 1.0 for deterministic checks (usually 1.0)
	Message       string
	Details       map[string]interface{} // Additional diagnostic info
	Duration      time.Duration
}

// ValidationReport aggregates all validation results
type ValidationReport struct {
	Response       LLMResponse
	Results        []ValidationResult
	OverallValid   bool
	OverallScore   float64
	ExecutionTime  time.Duration
	FailureReasons []string
}

// Validator interface for all validation checks
type Validator interface {
	Name() string
	Validate(ctx context.Context, response LLMResponse) ValidationResult
	Priority() int // Higher priority validators run first
}

// ============================================================================
// VALIDATION ORCHESTRATOR
// ============================================================================

type ValidationOrchestrator struct {
	Validators      []Validator
	StopOnFailure   bool
	MinPassingScore float64
}

func NewValidationOrchestrator(stopOnFailure bool, minScore float64) *ValidationOrchestrator {
	return &ValidationOrchestrator{
		Validators:      []Validator{},
		StopOnFailure:   stopOnFailure,
		MinPassingScore: minScore,
	}
}

func (vo *ValidationOrchestrator) AddValidator(v Validator) {
	vo.Validators = append(vo.Validators, v)
}

func (vo *ValidationOrchestrator) RunValidation(ctx context.Context, response LLMResponse) ValidationReport {
	startTime := time.Now()
	report := ValidationReport{
		Response:       response,
		Results:        []ValidationResult{},
		OverallValid:   true,
		FailureReasons: []string{},
	}

	// Sort validators by priority (implement if needed)
	totalScore := 0.0
	validatorCount := 0

	for _, validator := range vo.Validators {
		select {
		case <-ctx.Done():
			report.OverallValid = false
			report.FailureReasons = append(report.FailureReasons, "Validation cancelled")
			return report
		default:
			result := validator.Validate(ctx, response)
			report.Results = append(report.Results, result)
			
			totalScore += result.Confidence
			validatorCount++

			if !result.IsValid {
				report.OverallValid = false
				report.FailureReasons = append(report.FailureReasons, 
					fmt.Sprintf("%s: %s", result.ValidatorName, result.Message))
				
				if vo.StopOnFailure {
					break
				}
			}
		}
	}

	if validatorCount > 0 {
		report.OverallScore = totalScore / float64(validatorCount)
	}
	
	report.ExecutionTime = time.Since(startTime)
	return report
}

// ============================================================================
// DETERMINISTIC VALIDATORS
// ============================================================================

// 1. KEYWORD PRESENCE VALIDATOR
type KeywordPresenceValidator struct {
	RequiredKeywords []string
	CaseSensitive    bool
}

func (kv *KeywordPresenceValidator) Name() string {
	return "KeywordPresenceValidator"
}

func (kv *KeywordPresenceValidator) Priority() int {
	return 100
}

func (kv *KeywordPresenceValidator) Validate(ctx context.Context, response LLMResponse) ValidationResult {
	start := time.Now()
	missing := []string{}
	output := response.Output
	
	if !kv.CaseSensitive {
		output = strings.ToLower(output)
	}

	for _, keyword := range kv.RequiredKeywords {
		searchTerm := keyword
		if !kv.CaseSensitive {
			searchTerm = strings.ToLower(keyword)
		}
		
		if !strings.Contains(output, searchTerm) {
			missing = append(missing, keyword)
		}
	}

	isValid := len(missing) == 0
	confidence := 1.0
	if !isValid {
		confidence = 0.0
	}

	return ValidationResult{
		ValidatorName: kv.Name(),
		IsValid:       isValid,
		Confidence:    confidence,
		Message:       fmt.Sprintf("Required keywords check: %d missing", len(missing)),
		Details: map[string]interface{}{
			"missing_keywords": missing,
			"required_count":   len(kv.RequiredKeywords),
		},
		Duration: time.Since(start),
	}
}

// 2. FORBIDDEN CONTENT VALIDATOR
type ForbiddenContentValidator struct {
	ForbiddenPatterns []string // Can be regex patterns
	UseRegex          bool
}

func (fv *ForbiddenContentValidator) Name() string {
	return "ForbiddenContentValidator"
}

func (fv *ForbiddenContentValidator) Priority() int {
	return 150
}

func (fv *ForbiddenContentValidator) Validate(ctx context.Context, response LLMResponse) ValidationResult {
	start := time.Now()
	found := []string{}
	output := strings.ToLower(response.Output)

	for _, pattern := range fv.ForbiddenPatterns {
		if fv.UseRegex {
			matched, err := regexp.MatchString(pattern, output)
			if err != nil {
				return ValidationResult{
					ValidatorName: fv.Name(),
					IsValid:       false,
					Confidence:    0.0,
					Message:       fmt.Sprintf("Regex error: %v", err),
					Duration:      time.Since(start),
				}
			}
			if matched {
				found = append(found, pattern)
			}
		} else {
			if strings.Contains(output, strings.ToLower(pattern)) {
				found = append(found, pattern)
			}
		}
	}

	isValid := len(found) == 0
	confidence := 1.0
	if !isValid {
		confidence = 0.0
	}

	return ValidationResult{
		ValidatorName: fv.Name(),
		IsValid:       isValid,
		Confidence:    confidence,
		Message:       fmt.Sprintf("Forbidden content check: %d violations", len(found)),
		Details: map[string]interface{}{
			"violations": found,
		},
		Duration: time.Since(start),
	}
}

// 3. OUTPUT LENGTH VALIDATOR
type OutputLengthValidator struct {
	MinWords int
	MaxWords int
	MinChars int
	MaxChars int
}

func (olv *OutputLengthValidator) Name() string {
	return "OutputLengthValidator"
}

func (olv *OutputLengthValidator) Priority() int {
	return 50
}

func (olv *OutputLengthValidator) Validate(ctx context.Context, response LLMResponse) ValidationResult {
	start := time.Now()
	words := strings.Fields(response.Output)
	wordCount := len(words)
	charCount := len(response.Output)

	issues := []string{}

	if olv.MinWords > 0 && wordCount < olv.MinWords {
		issues = append(issues, fmt.Sprintf("Too few words: %d (min: %d)", wordCount, olv.MinWords))
	}
	if olv.MaxWords > 0 && wordCount > olv.MaxWords {
		issues = append(issues, fmt.Sprintf("Too many words: %d (max: %d)", wordCount, olv.MaxWords))
	}
	if olv.MinChars > 0 && charCount < olv.MinChars {
		issues = append(issues, fmt.Sprintf("Too few chars: %d (min: %d)", charCount, olv.MinChars))
	}
	if olv.MaxChars > 0 && charCount > olv.MaxChars {
		issues = append(issues, fmt.Sprintf("Too many chars: %d (max: %d)", charCount, olv.MaxChars))
	}

	isValid := len(issues) == 0
	confidence := 1.0
	if !isValid {
		confidence = 0.0
	}

	return ValidationResult{
		ValidatorName: olv.Name(),
		IsValid:       isValid,
		Confidence:    confidence,
		Message:       strings.Join(issues, "; "),
		Details: map[string]interface{}{
			"word_count": wordCount,
			"char_count": charCount,
		},
		Duration: time.Since(start),
	}
}

// 4. STRUCTURAL PATTERN VALIDATOR
type StructuralPatternValidator struct {
	RequiredPatterns []string // e.g., "Step 1:", "Conclusion:", etc.
	MinOccurrences   int
}

func (spv *StructuralPatternValidator) Name() string {
	return "StructuralPatternValidator"
}

func (spv *StructuralPatternValidator) Priority() int {
	return 80
}

func (spv *StructuralPatternValidator) Validate(ctx context.Context, response LLMResponse) ValidationResult {
	start := time.Now()
	output := response.Output
	patternCounts := make(map[string]int)

	for _, pattern := range spv.RequiredPatterns {
		count := strings.Count(strings.ToLower(output), strings.ToLower(pattern))
		patternCounts[pattern] = count
	}

	issues := []string{}
	for pattern, count := range patternCounts {
		if count < spv.MinOccurrences {
			issues = append(issues, fmt.Sprintf("Pattern '%s' appears %d times (min: %d)", 
				pattern, count, spv.MinOccurrences))
		}
	}

	isValid := len(issues) == 0
	confidence := 1.0
	if !isValid {
		confidence = 0.0
	}

	return ValidationResult{
		ValidatorName: spv.Name(),
		IsValid:       isValid,
		Confidence:    confidence,
		Message:       fmt.Sprintf("Structural pattern check: %d issues", len(issues)),
		Details: map[string]interface{}{
			"pattern_counts": patternCounts,
			"issues":         issues,
		},
		Duration: time.Since(start),
	}
}

// 5. CONTRADICTION DETECTOR (Deterministic Pattern-Based)
type ContradictionDetector struct {
	ContradictionPairs [][]string // e.g., [["always", "never"], ["true", "false"]]
}

func (cd *ContradictionDetector) Name() string {
	return "ContradictionDetector"
}

func (cd *ContradictionDetector) Priority() int {
	return 120
}

func (cd *ContradictionDetector) Validate(ctx context.Context, response LLMResponse) ValidationResult {
	start := time.Now()
	output := strings.ToLower(response.Output)
	contradictions := []string{}

	for _, pair := range cd.ContradictionPairs {
		if len(pair) != 2 {
			continue
		}
		
		term1, term2 := strings.ToLower(pair[0]), strings.ToLower(pair[1])
		if strings.Contains(output, term1) && strings.Contains(output, term2) {
			contradictions = append(contradictions, fmt.Sprintf("'%s' vs '%s'", pair[0], pair[1]))
		}
	}

	isValid := len(contradictions) == 0
	confidence := 1.0
	if !isValid {
		confidence = 0.0
	}

	return ValidationResult{
		ValidatorName: cd.Name(),
		IsValid:       isValid,
		Confidence:    confidence,
		Message:       fmt.Sprintf("Found %d potential contradictions", len(contradictions)),
		Details: map[string]interface{}{
			"contradictions": contradictions,
		},
		Duration: time.Since(start),
	}
}

// 6. JSON FORMAT VALIDATOR (if output should be JSON)
type JSONFormatValidator struct {
	RequireValidJSON bool
	RequiredKeys     []string
}

func (jv *JSONFormatValidator) Name() string {
	return "JSONFormatValidator"
}

func (jv *JSONFormatValidator) Priority() int {
	return 90
}

func (jv *JSONFormatValidator) Validate(ctx context.Context, response LLMResponse) ValidationResult {
	start := time.Now()
	
	var data map[string]interface{}
	err := json.Unmarshal([]byte(response.Output), &data)
	
	if err != nil {
		return ValidationResult{
			ValidatorName: jv.Name(),
			IsValid:       false,
			Confidence:    0.0,
			Message:       fmt.Sprintf("Invalid JSON: %v", err),
			Duration:      time.Since(start),
		}
	}

	missingKeys := []string{}
	for _, key := range jv.RequiredKeys {
		if _, exists := data[key]; !exists {
			missingKeys = append(missingKeys, key)
		}
	}

	isValid := len(missingKeys) == 0
	confidence := 1.0
	if !isValid {
		confidence = 0.0
	}

	return ValidationResult{
		ValidatorName: jv.Name(),
		IsValid:       isValid,
		Confidence:    confidence,
		Message:       fmt.Sprintf("JSON validation: %d missing keys", len(missingKeys)),
		Details: map[string]interface{}{
			"missing_keys": missingKeys,
			"found_keys":   getKeys(data),
		},
		Duration: time.Since(start),
	}
}

func getKeys(m map[string]interface{}) []string {
	keys := []string{}
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

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
