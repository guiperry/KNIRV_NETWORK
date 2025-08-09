package validation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"nexus-backend/internal/config"
)

// APIServer provides HTTP API endpoints for Validation Core
type APIServer struct {
	validationCore *ValidationCore
	config         *config.Config
	server         *http.Server
}

// NewAPIServer creates a new API server for Validation Core
func NewAPIServer(validationCore *ValidationCore, cfg *config.Config) *APIServer {
	return &APIServer{
		validationCore: validationCore,
		config:         cfg,
	}
}

// Start starts the HTTP API server
func (s *APIServer) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", s.handleHealth)

	// Validation endpoints
	mux.HandleFunc("/api/validation-tasks", s.handleValidationTasks)
	mux.HandleFunc("/api/validation-tasks/", s.handleValidationTaskDetails)

	// Results endpoints
	mux.HandleFunc("/api/validation-results", s.handleValidationResults)
	mux.HandleFunc("/api/validation-results/", s.handleValidationResultDetails)

	// System status endpoints
	mux.HandleFunc("/api/system/status", s.handleSystemStatus)
	mux.HandleFunc("/api/system/metrics", s.handleSystemMetrics)

	// CORS middleware for development
	handler := s.corsMiddleware(mux)

	// Use port 8081 for validation core (different from DVE Manager's 8080)
	port := 8081
	if s.config.API.Port != 8080 {
		port = s.config.API.Port
	}

	addr := fmt.Sprintf("%s:%d", s.config.API.BindAddress, port)
	s.server = &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Validation Core API server error: %v\n", err)
		}
	}()

	fmt.Printf("Validation Core API server started on http://%s\n", addr)
	return nil
}

// Stop stops the HTTP API server
func (s *APIServer) Stop(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}

// corsMiddleware adds CORS headers for development
func (s *APIServer) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// handleHealth handles health check requests
func (s *APIServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"service":   "validation-core",
		"timestamp": time.Now().UTC(),
		"version":   "1.0.0",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleValidationTasks handles validation tasks listing
func (s *APIServer) handleValidationTasks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		// Return mock validation tasks for now
		tasks := []map[string]interface{}{
			{
				"id":          "task-1",
				"type":        "data_validation",
				"status":      "pending",
				"assigned_to": "validator-1",
				"created_at":  time.Now().Add(-time.Hour).UTC(),
			},
			{
				"id":          "task-2",
				"type":        "computation_validation",
				"status":      "completed",
				"assigned_to": "validator-2",
				"created_at":  time.Now().Add(-2 * time.Hour).UTC(),
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"tasks": tasks,
			"count": len(tasks),
		})

	case "POST":
		// Create new validation task
		var taskRequest map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&taskRequest); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Mock task creation
		task := map[string]interface{}{
			"id":          fmt.Sprintf("task-%d", time.Now().Unix()),
			"type":        taskRequest["type"],
			"status":      "pending",
			"assigned_to": taskRequest["assigned_to"],
			"created_at":  time.Now().UTC(),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(task)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleValidationTaskDetails handles individual validation task operations
func (s *APIServer) handleValidationTaskDetails(w http.ResponseWriter, r *http.Request) {
	taskID := r.URL.Path[len("/api/validation-tasks/"):]
	if taskID == "" {
		http.Error(w, "Task ID required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case "GET":
		// Return mock task details
		task := map[string]interface{}{
			"id":          taskID,
			"type":        "data_validation",
			"status":      "pending",
			"assigned_to": "validator-1",
			"created_at":  time.Now().Add(-time.Hour).UTC(),
			"details": map[string]interface{}{
				"data_hash":    "0x1234567890abcdef",
				"validator_id": "validator-1",
				"timeout":      "300s",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(task)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleValidationResults handles validation results
func (s *APIServer) handleValidationResults(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		// Return mock validation results
		results := []map[string]interface{}{
			{
				"id":           "result-1",
				"task_id":      "task-1",
				"validator":    "validator-1",
				"result":       "valid",
				"confidence":   0.95,
				"completed_at": time.Now().Add(-30 * time.Minute).UTC(),
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"results": results,
			"count":   len(results),
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleValidationResultDetails handles individual validation result operations
func (s *APIServer) handleValidationResultDetails(w http.ResponseWriter, r *http.Request) {
	resultID := r.URL.Path[len("/api/validation-results/"):]
	if resultID == "" {
		http.Error(w, "Result ID required", http.StatusBadRequest)
		return
	}

	// Return mock result details
	result := map[string]interface{}{
		"id":           resultID,
		"task_id":      "task-1",
		"validator":    "validator-1",
		"result":       "valid",
		"confidence":   0.95,
		"completed_at": time.Now().Add(-30 * time.Minute).UTC(),
		"details": map[string]interface{}{
			"proof":       "0xabcdef1234567890",
			"attestation": "SGX_ATTESTATION_DATA",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}

// handleSystemStatus handles system status requests
func (s *APIServer) handleSystemStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"service":         "validation-core",
		"status":          "running",
		"uptime":          time.Since(time.Now().Add(-time.Hour)).String(),
		"active_tasks":    2,
		"completed_tasks": 15,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(status)
}

// handleSystemMetrics handles system metrics requests
func (s *APIServer) handleSystemMetrics(w http.ResponseWriter, r *http.Request) {
	metrics := map[string]interface{}{
		"cpu_usage":       "30%",
		"memory_usage":    "768MB",
		"disk_usage":      "1.5GB",
		"validation_rate": "5 tasks/min",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(metrics)
}
