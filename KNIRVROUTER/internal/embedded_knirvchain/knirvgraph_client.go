// Package embedded_knirvchain provides KNIRVGRAPH client integration
package embedded_knirvchain

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// HTTPKNIRVGraphClient implements KNIRVGraphClient using HTTP requests
type HTTPKNIRVGraphClient struct {
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration
}

// NewHTTPKNIRVGraphClient creates a new HTTP-based KNIRVGRAPH client
func NewHTTPKNIRVGraphClient(baseURL string) *HTTPKNIRVGraphClient {
	return &HTTPKNIRVGraphClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		timeout: 5 * time.Second,
	}
}

// QueryErrorCluster queries KNIRVGRAPH for matching error clusters
func (c *HTTPKNIRVGraphClient) QueryErrorCluster(ctx context.Context, errorContext *ErrorContext) (*SkillNodeResult, error) {
	// Create query payload
	queryPayload := map[string]interface{}{
		"error_type":           errorContext.ErrorType,
		"error_message":        errorContext.ErrorMessage,
		"agent_id":             errorContext.AgentID,
		"base_model_id":        errorContext.BaseModelID,
		"runtime_environment":  errorContext.RuntimeEnvironment,
		"task_description":     errorContext.TaskDescription,
		"timestamp":            errorContext.Timestamp,
	}

	// Serialize payload
	jsonData, err := json.Marshal(queryPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query payload: %v", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/api/v1/error-clusters/query", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "KNIRVROUTER-EmbeddedChain/1.0")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("KNIRVGRAPH query failed with status %d", resp.StatusCode)
	}

	// Parse response
	var response struct {
		Success bool `json:"success"`
		Data    struct {
			SkillURI    string  `json:"skill_uri"`
			SkillNodeID string  `json:"skill_node_id"`
			ClusterID   string  `json:"cluster_id"`
			Confidence  float64 `json:"confidence"`
		} `json:"data"`
		Error string `json:"error,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("KNIRVGRAPH query failed: %s", response.Error)
	}

	// Return skill node result
	return &SkillNodeResult{
		SkillURI:    response.Data.SkillURI,
		SkillNodeID: response.Data.SkillNodeID,
		ClusterID:   response.Data.ClusterID,
		Confidence:  response.Data.Confidence,
	}, nil
}

// SubmitErrorNode submits an error node to KNIRVGRAPH for clustering
func (c *HTTPKNIRVGraphClient) SubmitErrorNode(ctx context.Context, errorContext *ErrorContext) error {
	// Create submission payload
	submissionPayload := map[string]interface{}{
		"error_node": map[string]interface{}{
			"error_type":           errorContext.ErrorType,
			"error_message":        errorContext.ErrorMessage,
			"stack_trace":          errorContext.StackTrace,
			"source_code_snippet":  errorContext.SourceCodeSnippet,
			"agent_id":             errorContext.AgentID,
			"agent_version":        errorContext.AgentVersion,
			"base_model_id":        errorContext.BaseModelID,
			"os":                   errorContext.OS,
			"architecture":         errorContext.Architecture,
			"runtime_environment":  errorContext.RuntimeEnvironment,
			"task_description":     errorContext.TaskDescription,
			"input_data_hash":      errorContext.InputDataHash,
			"skill_invoked_id":     errorContext.SkillInvokedID,
			"agent_state_hash":     errorContext.AgentStateHash,
			"timestamp":            errorContext.Timestamp,
			"additional_context":   errorContext.AdditionalContext,
		},
		"submission_metadata": map[string]interface{}{
			"source":    "knirvrouter-embedded-chain",
			"version":   "1.0.0",
			"timestamp": time.Now().Unix(),
		},
	}

	// Serialize payload
	jsonData, err := json.Marshal(submissionPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal submission payload: %v", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/api/v1/error-nodes/submit", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "KNIRVROUTER-EmbeddedChain/1.0")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("error node submission failed with status %d", resp.StatusCode)
	}

	// Parse response
	var response struct {
		Success bool   `json:"success"`
		Error   string `json:"error,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("failed to decode response: %v", err)
	}

	if !response.Success {
		return fmt.Errorf("error node submission failed: %s", response.Error)
	}

	log.Printf("Successfully submitted error node to KNIRVGRAPH: %s", errorContext.ErrorType)
	return nil
}

// MockKNIRVGraphClient provides a mock implementation for testing
type MockKNIRVGraphClient struct {
	skillMappings map[string]*SkillNodeResult
}

// NewMockKNIRVGraphClient creates a new mock KNIRVGRAPH client
func NewMockKNIRVGraphClient() *MockKNIRVGraphClient {
	return &MockKNIRVGraphClient{
		skillMappings: map[string]*SkillNodeResult{
			"TypeError": {
				SkillURI:    "knirv://skill/javascript-type-checker-v1",
				SkillNodeID: "skill_node_type_checker_001",
				ClusterID:   "cluster_javascript_type_errors",
				Confidence:  0.85,
			},
			"ReferenceError": {
				SkillURI:    "knirv://skill/javascript-reference-resolver-v1",
				SkillNodeID: "skill_node_ref_resolver_001",
				ClusterID:   "cluster_javascript_reference_errors",
				Confidence:  0.80,
			},
			"SyntaxError": {
				SkillURI:    "knirv://skill/syntax-error-fixer-v2",
				SkillNodeID: "skill_node_syntax_fixer_002",
				ClusterID:   "cluster_syntax_errors",
				Confidence:  0.90,
			},
			"NullPointerException": {
				SkillURI:    "knirv://skill/null-pointer-guard-v1",
				SkillNodeID: "skill_node_null_guard_001",
				ClusterID:   "cluster_null_pointer_errors",
				Confidence:  0.88,
			},
		},
	}
}

// QueryErrorCluster returns a mock skill node result based on error type
func (m *MockKNIRVGraphClient) QueryErrorCluster(ctx context.Context, errorContext *ErrorContext) (*SkillNodeResult, error) {
	// Simulate network delay
	time.Sleep(50 * time.Millisecond)

	// Look up skill mapping
	if result, exists := m.skillMappings[errorContext.ErrorType]; exists {
		log.Printf("Mock KNIRVGRAPH: Found skill for error type %s: %s", errorContext.ErrorType, result.SkillURI)
		return result, nil
	}

	// Return generic skill for unknown error types
	return &SkillNodeResult{
		SkillURI:    "knirv://skill/generic-debugger-v1",
		SkillNodeID: "skill_node_generic_001",
		ClusterID:   "cluster_generic_errors",
		Confidence:  0.60,
	}, nil
}

// SubmitErrorNode logs the error node submission (mock implementation)
func (m *MockKNIRVGraphClient) SubmitErrorNode(ctx context.Context, errorContext *ErrorContext) error {
	log.Printf("Mock KNIRVGRAPH: Submitted error node - Type: %s, Agent: %s", 
		errorContext.ErrorType, errorContext.AgentID)
	return nil
}

// SetSkillMapping allows setting custom skill mappings for testing
func (m *MockKNIRVGraphClient) SetSkillMapping(errorType string, result *SkillNodeResult) {
	m.skillMappings[errorType] = result
}
