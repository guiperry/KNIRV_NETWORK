package validation

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// InferenceClient defines the interface for inference operations
type InferenceClient interface {
	GenerateText(modelName string, promptText string, instructionText string) (string, error)
}

// LLMEvaluator wraps the inference service for validation
type LLMEvaluator struct {
	inferenceClient InferenceClient
}

// NewLLMEvaluator creates a new LLM evaluator
func NewLLMEvaluator(inferenceClient InferenceClient) *LLMEvaluator {
	return &LLMEvaluator{
		inferenceClient: inferenceClient,
	}
}

// ReasoningQualityValidator evaluates reasoning quality using LLM
type ReasoningQualityValidator struct {
	evaluator *LLMEvaluator
	criteria  string
}

func (rqv *ReasoningQualityValidator) Name() string {
	return "ReasoningQualityValidator"
}

func (rqv *ReasoningQualityValidator) Priority() int {
	return 200
}

func (rqv *ReasoningQualityValidator) Validate(ctx context.Context, response LLMResponse) ValidationResult {
	start := time.Now()

	evaluationPrompt := fmt.Sprintf(`You are evaluating the reasoning quality of an LLM response.

Original Prompt: %s

LLM Response: %s

Evaluation Criteria: %s

Provide a score from 0.0 to 1.0 (0=poor, 1=excellent) and a brief explanation.

Respond in JSON format:
{"score": 0.85, "explanation": "..."}`, response.Prompt, response.Output, rqv.criteria)

	// Use inference service to evaluate
	result, err := rqv.evaluator.inferenceClient.GenerateText("deepseek-chat", evaluationPrompt, "")
	if err != nil {
		return ValidationResult{
			ValidatorName: rqv.Name(),
			IsValid:       false,
			Confidence:    0.0,
			Message:       fmt.Sprintf("LLM evaluation failed: %v", err),
			Duration:      time.Since(start),
		}
	}

	// Parse JSON response
	var evalResult struct {
		Score       float64 `json:"score"`
		Explanation string  `json:"explanation"`
	}

	if err := json.Unmarshal([]byte(result), &evalResult); err != nil {
		return ValidationResult{
			ValidatorName: rqv.Name(),
			IsValid:       false,
			Confidence:    0.0,
			Message:       fmt.Sprintf("Failed to parse evaluation: %v", err),
			Duration:      time.Since(start),
		}
	}

	isValid := evalResult.Score >= 0.7 // Threshold for passing

	return ValidationResult{
		ValidatorName: rqv.Name(),
		IsValid:       isValid,
		Confidence:    evalResult.Score,
		Message:       evalResult.Explanation,
		Details: map[string]interface{}{
			"score": evalResult.Score,
		},
		Duration: time.Since(start),
	}
}

// FactualityValidator checks factual accuracy with evidence grounding
type FactualityValidator struct {
	evaluator        *LLMEvaluator
	evidenceChunks   []string
	requireCitations bool
	minConfidence    float64
}

func (fv *FactualityValidator) Name() string {
	return "FactualityValidator"
}

func (fv *FactualityValidator) Priority() int {
	return 250 // Highest priority for factuality
}

func (fv *FactualityValidator) Validate(ctx context.Context, response LLMResponse) ValidationResult {
	start := time.Now()

	// Build evidence context
	evidenceContext := ""
	if len(fv.evidenceChunks) > 0 {
		evidenceContext = "Evidence:\n" + strings.Join(fv.evidenceChunks, "\n\n")
	}

	evaluationPrompt := fmt.Sprintf(`You are evaluating the factual accuracy of an LLM response.

%s

Question: %s

LLM Response: %s

Evaluate:
1. Is the response factually accurate based on the evidence?
2. Are claims properly grounded in evidence?
3. What is the confidence level (0.0-1.0)?
4. Should this response be refused due to insufficient evidence?

Respond in JSON format:
{
    "is_accurate": true/false,
    "confidence": 0.85,
    "citations": [0, 2, 5],
    "refused": false,
    "explanation": "..."
}`, evidenceContext, response.Prompt, response.Output)

	result, err := fv.evaluator.inferenceClient.GenerateText("deepseek-chat", evaluationPrompt, "")
	if err != nil {
		return ValidationResult{
			ValidatorName: fv.Name(),
			IsValid:       false,
			Confidence:    0.0,
			Message:       fmt.Sprintf("Factuality check failed: %v", err),
			Duration:      time.Since(start),
		}
	}

	var evalResult struct {
		IsAccurate  bool      `json:"is_accurate"`
		Confidence  float64   `json:"confidence"`
		Citations   []int     `json:"citations"`
		Refused     bool      `json:"refused"`
		Explanation string    `json:"explanation"`
	}

	if err := json.Unmarshal([]byte(result), &evalResult); err != nil {
		return ValidationResult{
			ValidatorName: fv.Name(),
			IsValid:       false,
			Confidence:    0.0,
			Message:       fmt.Sprintf("Failed to parse factuality result: %v", err),
			Duration:      time.Since(start),
		}
	}

	// Apply refusal gate (from factuality_slice_integration.md)
	if evalResult.Refused || evalResult.Confidence < fv.minConfidence {
		return ValidationResult{
			ValidatorName: fv.Name(),
			IsValid:       false,
			Confidence:    evalResult.Confidence,
			Message:       "Insufficient evidence or low confidence - response should be refused",
			Details: map[string]interface{}{
				"refused":     true,
				"confidence":  evalResult.Confidence,
				"explanation": evalResult.Explanation,
			},
			Duration: time.Since(start),
		}
	}

	isValid := evalResult.IsAccurate && evalResult.Confidence >= fv.minConfidence

	return ValidationResult{
		ValidatorName: fv.Name(),
		IsValid:       isValid,
		Confidence:    evalResult.Confidence,
		Message:       evalResult.Explanation,
		Details: map[string]interface{}{
			"is_accurate": evalResult.IsAccurate,
			"confidence":  evalResult.Confidence,
			"citations":   evalResult.Citations,
			"refused":     evalResult.Refused,
		},
		Duration: time.Since(start),
	}
}
