package web

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"nexus-backend/internal/services/validation"
	"nexus-backend/internal/web/middleware"

	"github.com/gorilla/mux"
)

type ValidationHandlers struct {
	validationCore *validation.ValidationCore
}

func NewValidationHandlers(validationCore *validation.ValidationCore) *ValidationHandlers {
	return &ValidationHandlers{validationCore: validationCore}
}

type ValidationTaskResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Total     int         `json:"total,omitempty"`
	Message   string      `json:"message,omitempty"`
	Error     string      `json:"error,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// GetValidationTasks handles GET /api/validation-tasks
func (h *ValidationHandlers) GetValidationTasks(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters for filtering
	filter := &validation.TaskFilter{
		Status:      r.URL.Query().Get("status"),
		Type:        r.URL.Query().Get("type"),
		RequestedBy: r.URL.Query().Get("requested_by"),
	}

	// Parse priority parameter
	priorityStr := r.URL.Query().Get("priority")
	if priorityStr != "" {
		if priority, err := strconv.Atoi(priorityStr); err == nil {
			filter.Priority = priority
		}
	}

	// Parse limit parameter
	limitStr := r.URL.Query().Get("limit")
	if limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			filter.Limit = limit
		}
	}

	tasks, err := h.validationCore.GetValidationTasks(filter)
	if err != nil {
		// If error is "not found", return empty array instead of 500 error
		if strings.Contains(err.Error(), "not found") {
			response := ValidationTaskResponse{
				Success:   true,
				Data:      []interface{}{}, // Empty array
				Total:     0,
				Timestamp: time.Now().Format(time.RFC3339),
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		// For other errors, still return 500
		response := ValidationTaskResponse{
			Success:   false,
			Error:     "Failed to fetch validation tasks: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := ValidationTaskResponse{
		Success:   true,
		Data:      tasks,
		Total:     len(tasks),
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// PostValidationTasks handles POST /api/validation-tasks (task creation)
func (h *ValidationHandlers) PostValidationTasks(w http.ResponseWriter, r *http.Request) {
	var req validation.CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response := ValidationTaskResponse{
			Success:   false,
			Error:     "Invalid request body",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Validate required fields
	if req.Type == "" {
		response := ValidationTaskResponse{
			Success:   false,
			Error:     "Task type is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	task, err := h.validationCore.CreateValidationTask(&req)
	if err != nil {
		response := ValidationTaskResponse{
			Success:   false,
			Error:     "Failed to create validation task: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := ValidationTaskResponse{
		Success:   true,
		Data:      task,
		Message:   "Validation task created successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// GetValidationTask handles GET /api/validation-tasks/{id}
func (h *ValidationHandlers) GetValidationTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	if taskID == "" {
		response := ValidationTaskResponse{
			Success:   false,
			Error:     "Task ID is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	task, err := h.validationCore.GetValidationTask(taskID)
	if err != nil {
		response := ValidationTaskResponse{
			Success:   false,
			Error:     "Failed to fetch task: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	if task == nil {
		response := ValidationTaskResponse{
			Success:   false,
			Error:     "Task not found",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := ValidationTaskResponse{
		Success:   true,
		Data:      task,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ExecuteValidationTask handles POST /api/validation-tasks/{id}/execute
func (h *ValidationHandlers) ExecuteValidationTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	if taskID == "" {
		response := ValidationTaskResponse{
			Success:   false,
			Error:     "Task ID is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get the task first
	task, err := h.validationCore.GetValidationTask(taskID)
	if err != nil {
		response := ValidationTaskResponse{
			Success:   false,
			Error:     "Task not found",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Execute the task
	_, err = h.validationCore.ExecuteValidation(task)
	if err != nil {
		response := ValidationTaskResponse{
			Success:   false,
			Error:     "Failed to execute task: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := ValidationTaskResponse{
		Success: true,
		Message: "Task execution started",
		Data: map[string]interface{}{
			"task_id": taskID,
			"status":  "running",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(response)
}

// RegisterRoutes registers the validation routes with the router
func (h *ValidationHandlers) RegisterRoutes(r *mux.Router, authMiddleware *middleware.AuthMiddleware) {
	// Create a subrouter for validation endpoints
	validationRouter := r.PathPrefix("/api/validation-tasks").Subrouter()

	// Public routes for monitoring
	validationRouter.HandleFunc("", h.GetValidationTasks).Methods("GET")
	validationRouter.HandleFunc("/{id}", h.GetValidationTask).Methods("GET")

	// Protected routes for management
	if authMiddleware != nil {
		protectedValidationRouter := validationRouter.PathPrefix("").Subrouter()
		protectedValidationRouter.Use(authMiddleware.RequireAuth)
		protectedValidationRouter.HandleFunc("", h.PostValidationTasks).Methods("POST")
		protectedValidationRouter.HandleFunc("/{id}/execute", h.ExecuteValidationTask).Methods("POST")
	} else {
		// If no auth middleware, allow all routes (for testnet mode)
		validationRouter.HandleFunc("", h.PostValidationTasks).Methods("POST")
		validationRouter.HandleFunc("/{id}/execute", h.ExecuteValidationTask).Methods("POST")
	}

	// Handle OPTIONS requests for CORS
	validationRouter.Methods("OPTIONS").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}
