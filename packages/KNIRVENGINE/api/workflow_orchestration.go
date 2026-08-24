package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"KNIRVENGINE/desktop-client/inference"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// WorkflowStatus represents the status of a workflow
type WorkflowStatus string

const (
	WorkflowStatusPending   WorkflowStatus = "pending"
	WorkflowStatusRunning   WorkflowStatus = "running"
	WorkflowStatusCompleted WorkflowStatus = "completed"
	WorkflowStatusFailed    WorkflowStatus = "failed"
	WorkflowStatusCancelled WorkflowStatus = "cancelled"
)

// WorkflowRequest represents a request to start a workflow
type WorkflowRequest struct {
	AgentID      string                 `json:"agent_id" binding:"required"`
	TargetID     string                 `json:"target_id" binding:"required"`
	CapabilityID string                 `json:"capability_id" binding:"required"`
	Input        map[string]interface{} `json:"input" binding:"required"`
}

// WorkflowResult represents the result of a workflow
type WorkflowResult struct {
	ID           string                 `json:"id"`
	Status       WorkflowStatus         `json:"status"`
	StartTime    time.Time              `json:"start_time"`
	EndTime      *time.Time             `json:"end_time,omitempty"`
	AgentID      string                 `json:"agent_id"`
	TargetID     string                 `json:"target_id"`
	CapabilityID string                 `json:"capability_id"`
	Input        map[string]interface{} `json:"input"`
	Output       map[string]interface{} `json:"output,omitempty"`
	Error        string                 `json:"error,omitempty"`
	OwnerID      int64                  `json:"owner_id"`
}

// WorkflowOrchestrationService manages workflow orchestration
type WorkflowOrchestrationService struct {
	workflows        map[string]*WorkflowResult
	mutex            sync.RWMutex
	inferenceService *inference.InferenceService
}

// SetInferenceService sets the inference service for the workflow orchestrator
func (s *WorkflowOrchestrationService) SetInferenceService(service *inference.InferenceService) {
	s.inferenceService = service
}

// NewWorkflowOrchestrationService creates a new workflow orchestration service
func NewWorkflowOrchestrationService() *WorkflowOrchestrationService {
	return &WorkflowOrchestrationService{
		workflows: make(map[string]*WorkflowResult),
	}
}

// StartWorkflow initiates a new workflow
func (s *WorkflowOrchestrationService) StartWorkflow(ctx context.Context, req WorkflowRequest, userID int64) (*WorkflowResult, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Generate workflow ID
	workflowID := uuid.New().String()

	// Create workflow result
	result := &WorkflowResult{
		ID:           workflowID,
		Status:       WorkflowStatusPending,
		StartTime:    time.Now(),
		AgentID:      req.AgentID,
		TargetID:     req.TargetID,
		CapabilityID: req.CapabilityID,
		Input:        req.Input,
		OwnerID:      userID,
	}

	// Store workflow in memory
	s.workflows[workflowID] = result

	// Start workflow in a goroutine with the provided context
	go s.executeWorkflow(ctx, result)

	return result, nil
}

// executeWorkflow runs a workflow asynchronously
func (s *WorkflowOrchestrationService) executeWorkflow(ctx context.Context, result *WorkflowResult) {
	s.mutex.Lock()
	// Update status to running
	result.Status = WorkflowStatusRunning
	s.mutex.Unlock()

	// Extract prompt from input
	prompt, ok := result.Input["prompt"].(string)
	if !ok {
		s.completeWorkflowWithError(result, "Missing or invalid prompt in input")
		return
	}

	// Simulate workflow execution
	log.Printf("Executing workflow %s with prompt: %s", result.ID, prompt)

	// Simulate processing time with context awareness
	var output string
	select {
	case <-time.After(2 * time.Second):
		// Generate a response based on the agent, target, and capability
		output = fmt.Sprintf("Response from agent %s using capability %s on target %s: Processed '%s'",
			result.AgentID, result.CapabilityID, result.TargetID, prompt)
	case <-ctx.Done():
		s.completeWorkflowWithError(result, fmt.Sprintf("Workflow cancelled: %v", ctx.Err()))
		return
	}

	// Complete workflow successfully
	s.completeWorkflowSuccess(result, output)
}

// completeWorkflowWithError marks a workflow as failed
func (s *WorkflowOrchestrationService) completeWorkflowWithError(result *WorkflowResult, errorMsg string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Update result
	result.Status = WorkflowStatusFailed
	result.Error = errorMsg
	endTime := time.Now()
	result.EndTime = &endTime

	log.Printf("Workflow %s failed: %s", result.ID, errorMsg)
}

// completeWorkflowSuccess marks a workflow as completed
func (s *WorkflowOrchestrationService) completeWorkflowSuccess(result *WorkflowResult, output string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Update result
	result.Status = WorkflowStatusCompleted
	result.Output = map[string]interface{}{
		"text": output,
	}
	endTime := time.Now()
	result.EndTime = &endTime

	log.Printf("Workflow %s completed successfully", result.ID)
}

// GetWorkflow retrieves a workflow by ID
func (s *WorkflowOrchestrationService) GetWorkflow(id string) (*WorkflowResult, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	workflow, exists := s.workflows[id]
	if !exists {
		return nil, fmt.Errorf("workflow not found: %s", id)
	}

	return workflow, nil
}

// ListWorkflows retrieves all workflows for a user
func (s *WorkflowOrchestrationService) ListWorkflows(userID int64) ([]*WorkflowResult, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var results []*WorkflowResult
	for _, workflow := range s.workflows {
		if workflow.OwnerID == userID {
			results = append(results, workflow)
		}
	}

	return results, nil
}

// CancelWorkflow cancels a running workflow
func (s *WorkflowOrchestrationService) CancelWorkflow(id string, userID int64) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	workflow, exists := s.workflows[id]
	if !exists {
		return fmt.Errorf("workflow not found: %s", id)
	}

	if workflow.OwnerID != userID {
		return fmt.Errorf("access denied: workflow belongs to another user")
	}

	if workflow.Status != WorkflowStatusRunning && workflow.Status != WorkflowStatusPending {
		return fmt.Errorf("cannot cancel workflow with status: %s", workflow.Status)
	}

	workflow.Status = WorkflowStatusCancelled
	endTime := time.Now()
	workflow.EndTime = &endTime
	workflow.Error = "Workflow cancelled by user"

	return nil
}

// ClearAllWorkflows removes all workflows from memory
func (s *WorkflowOrchestrationService) ClearAllWorkflows() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Clear all workflows from memory
	s.workflows = make(map[string]*WorkflowResult)
	log.Println("All workflows cleared from memory")
	return nil
}

// RegisterWorkflowHandlers registers the workflow API handlers
func (s *WorkflowOrchestrationService) RegisterHandlers(router *mux.Router) {
	router.HandleFunc("/api/v1/workflows", s.handleStartWorkflow).Methods("POST")
	router.HandleFunc("/api/v1/workflows", s.handleListWorkflows).Methods("GET")
	router.HandleFunc("/api/v1/workflows/{id}", s.handleGetWorkflow).Methods("GET")
	router.HandleFunc("/api/v1/workflows/{id}/cancel", s.handleCancelWorkflow).Methods("POST")
}

// handleStartWorkflow handles POST /api/v1/workflows
func (s *WorkflowOrchestrationService) handleStartWorkflow(w http.ResponseWriter, r *http.Request) {
	// For simplicity, we'll use a fixed user ID
	userID := int64(1)

	var req WorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	result, err := s.StartWorkflow(r.Context(), req, userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to start workflow: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"workflow": result,
	})
}

// handleGetWorkflow handles GET /api/v1/workflows/{id}
func (s *WorkflowOrchestrationService) handleGetWorkflow(w http.ResponseWriter, r *http.Request) {
	// For simplicity, we'll use a fixed user ID
	userID := int64(1)

	vars := mux.Vars(r)
	id := vars["id"]

	workflow, err := s.GetWorkflow(id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get workflow: %v", err), http.StatusNotFound)
		return
	}

	// Check ownership
	if workflow.OwnerID != userID {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"workflow": workflow,
	})
}

// handleListWorkflows handles GET /api/v1/workflows
func (s *WorkflowOrchestrationService) handleListWorkflows(w http.ResponseWriter, r *http.Request) {
	// For simplicity, we'll use a fixed user ID
	userID := int64(1)

	workflows, err := s.ListWorkflows(userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list workflows: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"workflows": workflows,
	})
}

// handleCancelWorkflow handles POST /api/v1/workflows/{id}/cancel
func (s *WorkflowOrchestrationService) handleCancelWorkflow(w http.ResponseWriter, r *http.Request) {
	// For simplicity, we'll use a fixed user ID
	userID := int64(1)

	vars := mux.Vars(r)
	id := vars["id"]

	if err := s.CancelWorkflow(id, userID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to cancel workflow: %v", err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Workflow cancelled successfully",
	})
}
