// Package handlers provides HTTP handlers for the API
package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"KNIRVENGINE/desktop-client/api/utils"
)

// Pre-compiled regex patterns for better performance
var (
	numberedListRegex = regexp.MustCompile(`^\d+[\.\)]`)
	percentRegex      = regexp.MustCompile(`confidence:?\s*(\d+)%`)
	decimalRegex      = regexp.MustCompile(`confidence:?\s*(0\.\d+)`)
	categoryRegex     = regexp.MustCompile(`category:?\s*(\w+)`)
)

// AIClient defines the interface for AI completion
type AIClient interface {
	GenerateCompletion(prompt string) (string, error)
}

// InferenceServiceClient wraps an inference service to implement AIClient
type InferenceServiceClient struct {
	service InferenceService
}

// InferenceService defines the interface for the inference service
type InferenceService interface {
	GenerateText(modelName string, promptText string, instructionText string) (string, error)
	IsRunning() bool
}

// NewInferenceServiceClient creates a new AI client from an inference service
func NewInferenceServiceClient(service InferenceService) *InferenceServiceClient {
	return &InferenceServiceClient{service: service}
}

// GenerateCompletion implements the AIClient interface
func (c *InferenceServiceClient) GenerateCompletion(prompt string) (string, error) {
	if !c.service.IsRunning() {
		return "", fmt.Errorf("inference service is not running")
	}
	// Use a default model and empty instruction for error analysis
	return c.service.GenerateText("", prompt, "Analyze this error and provide helpful suggestions.")
}

// ErrorInferenceRequest represents a request to analyze an error
type ErrorInferenceRequest struct {
	Prompt       string                 `json:"prompt"`
	ErrorContext map[string]interface{} `json:"error_context"`
}

// ErrorInferenceResponse represents the response from error analysis
type ErrorInferenceResponse struct {
	Analysis                string   `json:"analysis"`
	SuggestedFixes          []string `json:"suggested_fixes"`
	Confidence              float64  `json:"confidence"`
	Category                string   `json:"category"`
	EstimatedResolutionTime string   `json:"estimated_resolution_time"`
	RequiresUserAction      bool     `json:"requires_user_action"`
	AutomatedFixAvailable   bool     `json:"automated_fix_available"`
}

// ErrorInferenceHandler handles requests to analyze errors
type ErrorInferenceHandler struct {
	aiClient             AIClient
	troubleshootingStore *utils.TroubleshootingStore
}

// NewErrorInferenceHandler creates a new error inference handler
func NewErrorInferenceHandler(aiClient AIClient, troubleshootingStorePath string) (*ErrorInferenceHandler, error) {
	// Initialize the troubleshooting store
	store, err := utils.NewTroubleshootingStore(troubleshootingStorePath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize troubleshooting store: %w", err)
	}

	return &ErrorInferenceHandler{
		aiClient:             aiClient,
		troubleshootingStore: store,
	}, nil
}

// AnalyzeError handles requests to analyze errors
func (h *ErrorInferenceHandler) AnalyzeError(c *gin.Context) {
	var req ErrorInferenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Extract error information from context
	errorType := getStringValue(req.ErrorContext, "type", "unknown")
	errorMessage := getStringValue(req.ErrorContext, "message", "")
	errorDetails := getStringValue(req.ErrorContext, "details", "")

	// Extract symptoms from error context
	symptoms := []string{errorMessage}
	if errorDetails != "" {
		symptoms = append(symptoms, errorDetails)
	}

	// Search for relevant troubleshooting information
	results, err := h.troubleshootingStore.SearchByErrorType(errorType, symptoms, 3)
	if err != nil {
		log.Printf("Error searching troubleshooting store: %v", err)
		// Continue with empty results
		results = []utils.SearchResult{}
	}

	// Build enhanced prompt with troubleshooting information
	enhancedPrompt := h.buildEnhancedPrompt(req.Prompt, results, req.ErrorContext)

	// Call AI model for inference
	response, err := h.aiClient.GenerateCompletion(enhancedPrompt)
	if err != nil {
		log.Printf("Error calling AI model: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to analyze error"})
		return
	}

	// Parse the AI response
	inferenceResponse, err := parseAIResponse(response)
	if err != nil {
		log.Printf("Error parsing AI response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse AI response"})
		return
	}

	c.JSON(http.StatusOK, inferenceResponse)
}

// buildEnhancedPrompt builds an enhanced prompt with troubleshooting information
func (h *ErrorInferenceHandler) buildEnhancedPrompt(basePrompt string, results []utils.SearchResult, _ map[string]interface{}) string {
	var sb strings.Builder

	// Start with the base prompt
	sb.WriteString(basePrompt)
	sb.WriteString("\n\n")

	// Add relevant troubleshooting information
	if len(results) > 0 {
		sb.WriteString("RELEVANT TROUBLESHOOTING INFORMATION:\n\n")

		for i, result := range results {
			sb.WriteString(fmt.Sprintf("--- Relevant Issue #%d: %s ---\n", i+1, result.Chunk.Issue))
			sb.WriteString(fmt.Sprintf("Category: %s\n", result.Chunk.Category))
			sb.WriteString("Symptoms:\n")
			for _, symptom := range result.Chunk.Symptoms {
				sb.WriteString(fmt.Sprintf("- %s\n", symptom))
			}
			sb.WriteString("\nTroubleshooting Steps:\n")

			// Extract troubleshooting steps from content
			content := result.Chunk.Content
			if idx := strings.Index(content, "Troubleshooting Steps:"); idx != -1 {
				steps := content[idx+len("Troubleshooting Steps:"):]
				sb.WriteString(steps)
			}

			sb.WriteString("\n\n")
		}
	}

	// Add instructions to use the troubleshooting information
	sb.WriteString("Please use the above troubleshooting information to help analyze the error. " +
		"If the error matches any of the known issues, incorporate the relevant troubleshooting steps " +
		"in your response. If not, provide your best analysis based on the error details.\n\n")

	return sb.String()
}

// parseAIResponse parses the AI model's response into a structured format
func parseAIResponse(response string) (ErrorInferenceResponse, error) {
	var result ErrorInferenceResponse

	// Try to parse as JSON first
	if strings.Contains(response, "{") && strings.Contains(response, "}") {
		// Extract JSON part
		jsonStart := strings.Index(response, "{")
		jsonEnd := strings.LastIndex(response, "}") + 1
		if jsonStart != -1 && jsonEnd != -1 && jsonEnd > jsonStart {
			jsonStr := response[jsonStart:jsonEnd]
			err := json.Unmarshal([]byte(jsonStr), &result)
			if err == nil {
				return result, nil
			}
			// If JSON parsing fails, fall back to text parsing
		}
	}

	// Fall back to text parsing
	result.Analysis = extractSection(response, "analysis", "clear explanation")
	result.SuggestedFixes = extractList(response, "suggested", "fix", "step")
	result.Confidence = extractConfidence(response)
	result.Category = extractCategory(response)
	result.EstimatedResolutionTime = extractSection(response, "time", "minute", "hour")
	result.RequiresUserAction = !strings.Contains(strings.ToLower(response), "automated") ||
		strings.Contains(strings.ToLower(response), "user action")
	result.AutomatedFixAvailable = strings.Contains(strings.ToLower(response), "automated fix") &&
		!strings.Contains(strings.ToLower(response), "no automated fix")

	return result, nil
}

// Helper functions for text extraction
func extractSection(text, keyword1, keyword2 string, keywords ...string) string {
	lowerText := strings.ToLower(text)
	keywords = append([]string{keyword1, keyword2}, keywords...)

	// Find a section that contains all keywords
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lowerLine := strings.ToLower(line)

		allFound := true
		for _, keyword := range keywords {
			if !strings.Contains(lowerLine, strings.ToLower(keyword)) {
				allFound = false
				break
			}
		}

		if allFound && i+1 < len(lines) {
			// Return the next line or paragraph
			var result strings.Builder
			for j := i + 1; j < len(lines) && j < i+5; j++ {
				if lines[j] == "" {
					break
				}
				if result.Len() > 0 {
					result.WriteString(" ")
				}
				result.WriteString(lines[j])
			}
			if result.Len() > 0 {
				return result.String()
			}
			return lines[i] // Return the current line if next is empty
		}
	}

	// If no section found, look for sentences containing keywords
	for _, keyword := range keywords {
		idx := strings.Index(lowerText, strings.ToLower(keyword))
		if idx != -1 {
			// Find sentence boundaries
			start := idx
			for start > 0 && !strings.Contains(".!?", string(text[start-1])) {
				start--
			}
			end := idx
			for end < len(text) && !strings.Contains(".!?", string(text[end])) {
				end++
			}
			if end < len(text) {
				end++
			}
			return strings.TrimSpace(text[start:end])
		}
	}

	return "Not specified"
}

func extractList(text string, keywords ...string) []string {
	var result []string

	// Look for numbered lists
	lines := strings.Split(text, "\n")
	inList := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check if this line starts a list related to our keywords
		if !inList {
			lowerLine := strings.ToLower(trimmed)
			allFound := true
			for _, keyword := range keywords {
				if !strings.Contains(lowerLine, strings.ToLower(keyword)) {
					allFound = false
					break
				}
			}
			if allFound {
				inList = true
				continue
			}
		}

		// Check if this is a list item
		if inList {
			// Check for numbered list format: "1. ", "2) ", etc.
			if numberedListRegex.MatchString(trimmed) {
				// Extract the content after the number
				parts := strings.SplitN(trimmed, " ", 2)
				if len(parts) > 1 {
					result = append(result, strings.TrimSpace(parts[1]))
				}
			} else if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
				// Check for bullet list format
				result = append(result, strings.TrimSpace(trimmed[2:]))
			} else if trimmed == "" {
				// Empty line ends the list
				inList = false
			} else if len(result) > 0 {
				// If we're in a list but this line doesn't match list format,
				// it might be a continuation of the previous item
				result[len(result)-1] += " " + trimmed
			}
		}
	}

	// If we didn't find a proper list, look for sentences with keywords
	if len(result) == 0 {
		for _, keyword := range keywords {
			idx := strings.Index(strings.ToLower(text), strings.ToLower(keyword))
			if idx != -1 {
				// Find sentence
				start := idx
				for start > 0 && !strings.Contains(".!?", string(text[start-1])) {
					start--
				}
				end := idx
				for end < len(text) && !strings.Contains(".!?", string(text[end])) {
					end++
				}
				if end < len(text) {
					end++
				}
				result = append(result, strings.TrimSpace(text[start:end]))
			}
		}
	}

	// If still empty, add a default item
	if len(result) == 0 {
		result = append(result, "Check the error details and logs for more information")
	}

	return result
}

func extractConfidence(text string) float64 {
	lowerText := strings.ToLower(text)

	// Look for confidence expressed as percentage
	if matches := percentRegex.FindStringSubmatch(lowerText); len(matches) > 1 {
		if val, err := strconv.Atoi(matches[1]); err == nil {
			return float64(val) / 100.0
		}
	}

	// Look for confidence expressed as decimal
	if matches := decimalRegex.FindStringSubmatch(lowerText); len(matches) > 1 {
		if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
			return val
		}
	}

	// Look for confidence expressed in words
	if strings.Contains(lowerText, "high confidence") {
		return 0.9
	} else if strings.Contains(lowerText, "medium confidence") {
		return 0.7
	} else if strings.Contains(lowerText, "low confidence") {
		return 0.4
	}

	return 0.5 // Default confidence
}

func extractCategory(text string) string {
	lowerText := strings.ToLower(text)

	categories := map[string]string{
		"network":        "network",
		"authentication": "authentication",
		"authorization":  "authorization",
		"permission":     "permissions",
		"database":       "database",
		"configuration":  "configuration",
		"validation":     "validation",
		"syntax":         "code",
		"code":           "code",
		"timeout":        "performance",
		"performance":    "performance",
		"memory":         "resources",
		"cpu":            "resources",
		"disk":           "resources",
	}

	// Check for explicit category mention
	if matches := categoryRegex.FindStringSubmatch(lowerText); len(matches) > 1 {
		return matches[1]
	}

	// Check for category keywords
	for keyword, category := range categories {
		if strings.Contains(lowerText, keyword) {
			return category
		}
	}

	return "unknown"
}

// getStringValue safely extracts a string value from a map
func getStringValue(m map[string]interface{}, key, defaultValue string) string {
	if val, ok := m[key]; ok {
		if strVal, ok := val.(string); ok {
			return strVal
		}
	}
	return defaultValue
}
