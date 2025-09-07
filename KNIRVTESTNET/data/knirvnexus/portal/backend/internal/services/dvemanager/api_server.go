package dvemanager

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"nexus-backend/internal/config"
)

// APIServer provides HTTP API endpoints for DVE Manager
type APIServer struct {
	dveManager *DVEManager
	config     *config.Config
	server     *http.Server
}

// NewAPIServer creates a new API server for DVE Manager
func NewAPIServer(dveManager *DVEManager, cfg *config.Config) *APIServer {
	return &APIServer{
		dveManager: dveManager,
		config:     cfg,
	}
}

// Start starts the HTTP API server
func (s *APIServer) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", s.handleHealth)

	// DVE nodes endpoints
	mux.HandleFunc("/api/dve-nodes", s.handleDVENodes)
	mux.HandleFunc("/api/dve-nodes/", s.handleDVENodeDetails)

	// Task management endpoints
	mux.HandleFunc("/api/tasks", s.handleTasks)
	mux.HandleFunc("/api/tasks/", s.handleTaskDetails)

	// System status endpoints
	mux.HandleFunc("/api/system/status", s.handleSystemStatus)
	mux.HandleFunc("/api/system/metrics", s.handleSystemMetrics)

	// CORS middleware for development
	handler := s.corsMiddleware(mux)

	addr := fmt.Sprintf("%s:%d", s.config.API.BindAddress, s.config.API.Port)
	s.server = &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("DVE Manager API server error: %v\n", err)
		}
	}()

	fmt.Printf("DVE Manager API server started on http://%s\n", addr)
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
		"service":   "dve-manager",
		"timestamp": time.Now().UTC(),
		"version":   "1.0.0",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleDVENodes handles DVE nodes listing
func (s *APIServer) handleDVENodes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		nodes := s.dveManager.GetAllNodes()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"nodes": nodes,
			"count": len(nodes),
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleDVENodeDetails handles individual DVE node operations
func (s *APIServer) handleDVENodeDetails(w http.ResponseWriter, r *http.Request) {
	// Extract node ID from URL path
	nodeID := r.URL.Path[len("/api/dve-nodes/"):]
	if nodeID == "" {
		http.Error(w, "Node ID required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case "GET":
		node, err := s.dveManager.GetNode(nodeID)
		if err != nil {
			http.Error(w, "Node not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(node)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleTasks handles task management
func (s *APIServer) handleTasks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		// Return mock tasks for now
		tasks := []map[string]interface{}{
			{
				"id":     "task-1",
				"type":   "validation",
				"status": "pending",
				"node":   "node-1",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"tasks": tasks,
			"count": len(tasks),
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleTaskDetails handles individual task operations
func (s *APIServer) handleTaskDetails(w http.ResponseWriter, r *http.Request) {
	taskID := r.URL.Path[len("/api/tasks/"):]
	if taskID == "" {
		http.Error(w, "Task ID required", http.StatusBadRequest)
		return
	}

	// Return mock task details for now
	task := map[string]interface{}{
		"id":     taskID,
		"type":   "validation",
		"status": "pending",
		"node":   "node-1",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(task)
}

// handleSystemStatus handles system status requests
func (s *APIServer) handleSystemStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"service":     "dve-manager",
		"status":      "running",
		"uptime":      time.Since(time.Now().Add(-time.Hour)).String(),
		"nodes_count": 0, // TODO: Get actual count
		"tasks_count": 0, // TODO: Get actual count
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(status)
}

// handleSystemMetrics handles system metrics requests
func (s *APIServer) handleSystemMetrics(w http.ResponseWriter, r *http.Request) {
	metrics := map[string]interface{}{
		"cpu_usage":    "25%",
		"memory_usage": "512MB",
		"disk_usage":   "2GB",
		"network_io":   "1MB/s",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(metrics)
}
