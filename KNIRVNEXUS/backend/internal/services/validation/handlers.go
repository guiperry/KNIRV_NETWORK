package validation

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"

	"backend-server/internal/web/middleware"
)

// Validation Task Handlers

// HandleCreateTask handles validation task creation requests
func (vc *ValidationCore) HandleCreateTask(w http.ResponseWriter, r *http.Request) {
	authCtx := middleware.GetAuthContext(r)
	if authCtx == nil {
		writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	var req CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Type == "" {
		writeError(w, http.StatusBadRequest, "Task type is required")
		return
	}

	task, err := vc.CreateValidationTask(&req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, task)
}

// HandleListTasks handles validation task listing requests
func (vc *ValidationCore) HandleListTasks(w http.ResponseWriter, r *http.Request) {
	authCtx := middleware.GetAuthContext(r)
	if authCtx == nil {
		writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	// Parse query parameters for filtering
	filter := &TaskFilter{
		Status: r.URL.Query().Get("status"),
		Type:   r.URL.Query().Get("type"),
	}

	// Parse priority if provided
	if priorityStr := r.URL.Query().Get("priority"); priorityStr != "" {
		// Use tagged switch for priority parsing
		switch priorityStr {
		case "high":
			filter.Priority = 3
		case "medium":
			filter.Priority = 2
		case "low":
			filter.Priority = 1
		default:
			// Handle unknown priority values
			log.Printf("Unknown priority value: %s", priorityStr)
		}
	}

	tasks, err := vc.GetValidationTasks(filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, tasks)
}

// HandleGetTask handles individual task retrieval requests
func (vc *ValidationCore) HandleGetTask(w http.ResponseWriter, r *http.Request) {
	authCtx := middleware.GetAuthContext(r)
	if authCtx == nil {
		writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	vars := mux.Vars(r)
	taskID := vars["id"]

	task, err := vc.GetValidationTask(taskID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Task not found")
		return
	}

	writeJSON(w, http.StatusOK, task)
}

// HandleExecuteTask handles task execution requests
func (vc *ValidationCore) HandleExecuteTask(w http.ResponseWriter, r *http.Request) {
	authCtx := middleware.GetAuthContext(r)
	if authCtx == nil {
		writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	vars := mux.Vars(r)
	taskID := vars["id"]

	// Get the task first
	task, err := vc.GetValidationTask(taskID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Task not found")
		return
	}

	// Execute the task
	_, err = vc.ExecuteValidation(task)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to execute task: "+err.Error())
		return
	}

	// Return execution started response (result will be available asynchronously)
	response := map[string]interface{}{
		"success": true,
		"message": "Task execution started",
		"task_id": taskID,
		"status":  "running",
	}

	writeJSON(w, http.StatusAccepted, response)
}

// Validation Result Handlers

// HandleGetTaskResults handles task result retrieval requests
func (vc *ValidationCore) HandleGetTaskResults(w http.ResponseWriter, r *http.Request) {
	authCtx := middleware.GetAuthContext(r)
	if authCtx == nil {
		writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	vars := mux.Vars(r)
	taskID := vars["id"]

	// This would retrieve results for the specified task
	// For now, return a placeholder response
	_ = taskID
	writeError(w, http.StatusNotImplemented, "Result retrieval not yet implemented")
}

// System Status Handlers

// HandleGetValidationStatus handles validation system status requests
func (vc *ValidationCore) HandleGetValidationStatus(w http.ResponseWriter, r *http.Request) {
	// This can be public for monitoring
	status := map[string]interface{}{
		"service":         "validation-core",
		"status":          "running",
		"running_tasks":   0,                      // TODO: Get actual count
		"completed_tasks": 0,                      // TODO: Get actual count
		"failed_tasks":    0,                      // TODO: Get actual count
		"timestamp":       "2024-01-01T00:00:00Z", // TODO: Use actual timestamp
	}

	writeJSON(w, http.StatusOK, status)
}

// HandleGetValidationMetrics handles validation metrics requests
func (vc *ValidationCore) HandleGetValidationMetrics(w http.ResponseWriter, r *http.Request) {
	authCtx := middleware.GetAuthContext(r)
	if authCtx == nil {
		writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	// This would return detailed validation metrics
	metrics := map[string]interface{}{
		"average_execution_time": 150.0,                  // TODO: Calculate actual metrics
		"success_rate":           0.95,                   // TODO: Calculate actual metrics
		"throughput":             10.0,                   // TODO: Calculate actual metrics
		"timestamp":              "2024-01-01T00:00:00Z", // TODO: Use actual timestamp
	}

	writeJSON(w, http.StatusOK, metrics)
}

// Queue Management Handlers

// HandleGetTaskQueue handles task queue status requests
func (vc *ValidationCore) HandleGetTaskQueue(w http.ResponseWriter, r *http.Request) {
	authCtx := middleware.GetAuthContext(r)
	if authCtx == nil {
		writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	// This would return current queue status
	queueStatus := map[string]interface{}{
		"pending_tasks": 0,                      // TODO: Get actual count
		"running_tasks": 0,                      // TODO: Get actual count
		"queue_length":  0,                      // TODO: Get actual count
		"timestamp":     "2024-01-01T00:00:00Z", // TODO: Use actual timestamp
	}

	writeJSON(w, http.StatusOK, queueStatus)
}

// Helper functions

// writeJSON writes a JSON response
func writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// writeError writes an error response
func writeError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
