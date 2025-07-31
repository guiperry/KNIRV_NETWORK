package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

// EconomicsIntegration handles integration with the economics service for KNIRVNEXUS
type EconomicsIntegration struct {
	economicsURL string
	httpClient   *http.Client
	enabled      bool
}

// ValidationRewardRequest represents a validation reward request
type ValidationRewardRequest struct {
	ValidatorID      string `json:"validator_id"`
	TargetID         string `json:"target_id"`
	ValidationResult bool   `json:"validation_result"`
}

// AgentActivityEvent represents agent activity for economics tracking
type AgentActivityEvent struct {
	AgentID     string                 `json:"agent_id"`
	Activity    string                 `json:"activity"`
	Success     bool                   `json:"success"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// EconomicsResponse represents a response from the economics service
type EconomicsResponse struct {
	Success bool                   `json:"success"`
	Data    map[string]interface{} `json:"data,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

// NewEconomicsIntegration creates a new economics integration instance for KNIRVNEXUS
func NewEconomicsIntegration() *EconomicsIntegration {
	economicsURL := os.Getenv("ECONOMICS_SERVICE_URL")
	if economicsURL == "" {
		economicsURL = "http://localhost:8090"
	}

	return &EconomicsIntegration{
		economicsURL: economicsURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		enabled: true,
	}
}

// ProcessValidationReward processes a validation reward through the economics service
func (ei *EconomicsIntegration) ProcessValidationReward(validatorID, targetID string, isValid bool) (*EconomicsResponse, error) {
	if !ei.enabled {
		return &EconomicsResponse{Success: true, Data: map[string]interface{}{"status": "disabled"}}, nil
	}

	request := ValidationRewardRequest{
		ValidatorID:      validatorID,
		TargetID:         targetID,
		ValidationResult: isValid,
	}

	return ei.makeRequest("POST", "/economics/validation/reward", request)
}

// RecordAgentActivity records agent activity with the economics service
func (ei *EconomicsIntegration) RecordAgentActivity(event AgentActivityEvent) error {
	if !ei.enabled {
		return nil
	}

	// For now, just log the activity
	// In a real implementation, this would send to the economics service
	log.Printf("Recording agent activity: %+v", event)

	// Update performance metrics based on activity
	if event.Success {
		ei.updatePerformanceMetrics(event.AgentID, event.Activity)
	}

	return nil
}

// updatePerformanceMetrics updates performance metrics for an agent
func (ei *EconomicsIntegration) updatePerformanceMetrics(agentID, activity string) {
	// This would send performance metrics to the economics service
	// For now, just log it
	log.Printf("Updating performance metrics for agent %s, activity: %s", agentID, activity)
}

// GetEconomicMetrics retrieves economic metrics from the economics service
func (ei *EconomicsIntegration) GetEconomicMetrics() (*EconomicsResponse, error) {
	if !ei.enabled {
		return &EconomicsResponse{Success: true, Data: map[string]interface{}{"status": "disabled"}}, nil
	}

	return ei.makeRequest("GET", "/economics/metrics", nil)
}

// GetServiceMetrics retrieves KNIRVNEXUS-specific metrics from the economics service
func (ei *EconomicsIntegration) GetServiceMetrics() (*EconomicsResponse, error) {
	if !ei.enabled {
		return &EconomicsResponse{Success: true, Data: map[string]interface{}{"status": "disabled"}}, nil
	}

	return ei.makeRequest("GET", "/economics/service/knirvnexus/metrics", nil)
}

// HealthCheck checks if the economics service is healthy
func (ei *EconomicsIntegration) HealthCheck() bool {
	if !ei.enabled {
		return true
	}

	resp, err := ei.makeRequest("GET", "/economics/health", nil)
	return err == nil && resp.Success
}

// makeRequest makes an HTTP request to the economics service
func (ei *EconomicsIntegration) makeRequest(method, endpoint string, data interface{}) (*EconomicsResponse, error) {
	url := ei.economicsURL + endpoint

	var reqBody []byte
	var err error
	if data != nil {
		reqBody, err = json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
	}

	var req *http.Request
	if reqBody != nil {
		req, err = http.NewRequest(method, url, bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest(method, url, nil)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := ei.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	var economicsResp EconomicsResponse
	if err := json.NewDecoder(resp.Body).Decode(&economicsResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return &economicsResp, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, economicsResp.Error)
	}

	return &economicsResp, nil
}

// Enable enables the economics integration
func (ei *EconomicsIntegration) Enable() {
	ei.enabled = true
	log.Println("Economics integration enabled")
}

// Disable disables the economics integration
func (ei *EconomicsIntegration) Disable() {
	ei.enabled = false
	log.Println("Economics integration disabled")
}

// IsEnabled returns whether the economics integration is enabled
func (ei *EconomicsIntegration) IsEnabled() bool {
	return ei.enabled
}

// StartBackgroundSync starts a background goroutine to sync with the economics service
func (ei *EconomicsIntegration) StartBackgroundSync(ctx context.Context) {
	if !ei.enabled {
		return
	}

	go func() {
		ticker := time.NewTicker(90 * time.Second) // Sync every 90 seconds
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				ei.performSync()
			}
		}
	}()
}

// performSync performs a sync with the economics service
func (ei *EconomicsIntegration) performSync() {
	// Health check
	if !ei.HealthCheck() {
		log.Println("Economics service health check failed")
		return
	}

	// Get metrics
	metrics, err := ei.GetEconomicMetrics()
	if err != nil {
		log.Printf("Failed to get economic metrics: %v", err)
		return
	}

	log.Printf("Economics sync successful: %+v", metrics.Data)
}

// Integration helper functions for existing KNIRVNEXUS code

// IntegrateAgentValidation integrates agent validation with economics
func (ei *EconomicsIntegration) IntegrateAgentValidation(validatorID, agentID string, isValid bool) error {
	// Process validation reward
	_, err := ei.ProcessValidationReward(validatorID, agentID, isValid)
	if err != nil {
		log.Printf("Failed to process validation reward: %v", err)
		return err
	}

	// Record the validation activity
	event := AgentActivityEvent{
		AgentID:   validatorID,
		Activity:  "validation",
		Success:   isValid,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"target_agent": agentID,
			"result":       isValid,
		},
	}

	return ei.RecordAgentActivity(event)
}

// IntegrateWorkflowExecution integrates workflow execution with economics
func (ei *EconomicsIntegration) IntegrateWorkflowExecution(agentID, workflowID string, success bool, metadata map[string]interface{}) error {
	event := AgentActivityEvent{
		AgentID:   agentID,
		Activity:  "workflow_execution",
		Success:   success,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"workflow_id": workflowID,
			"metadata":    metadata,
		},
	}

	return ei.RecordAgentActivity(event)
}

// IntegrateInferenceRequest integrates inference requests with economics
func (ei *EconomicsIntegration) IntegrateInferenceRequest(agentID, requestID string, success bool, responseTime time.Duration) error {
	event := AgentActivityEvent{
		AgentID:   agentID,
		Activity:  "inference",
		Success:   success,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"request_id":    requestID,
			"response_time": responseTime.Milliseconds(),
		},
	}

	return ei.RecordAgentActivity(event)
}

// AddEconomicsEndpoints adds economics-related endpoints to the HTTP server
func (ei *EconomicsIntegration) AddEconomicsEndpoints() {
	// Add endpoint to get economics metrics
	http.HandleFunc("/api/economics/metrics", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		metrics, err := ei.GetEconomicMetrics()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(metrics)
	})

	// Add endpoint to get service-specific metrics
	http.HandleFunc("/api/economics/service-metrics", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		metrics, err := ei.GetServiceMetrics()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(metrics)
	})

	// Add endpoint to process validation rewards
	http.HandleFunc("/api/economics/validation", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req ValidationRewardRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		resp, err := ei.ProcessValidationReward(req.ValidatorID, req.TargetID, req.ValidationResult)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	// Add endpoint to check economics integration status
	http.HandleFunc("/api/economics/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		status := map[string]interface{}{
			"enabled":     ei.IsEnabled(),
			"healthy":     ei.HealthCheck(),
			"service_url": ei.economicsURL,
			"timestamp":   time.Now(),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"data":    status,
		})
	})
}
