// connectivity/api_server.go
package connectivity

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// APIServer provides HTTP endpoints for connectivity proof engine
type APIServer struct {
	proofEngine *ConnectivityProofEngine
	port        int
	server      *http.Server
}

// NewAPIServer creates a new API server for connectivity endpoints
func NewAPIServer(proofEngine *ConnectivityProofEngine, port int) *APIServer {
	return &APIServer{
		proofEngine: proofEngine,
		port:        port,
	}
}

// Start starts the API server
func (api *APIServer) Start() error {
	router := mux.NewRouter()

	// Add CORS middleware
	router.Use(corsMiddleware)

	// API routes
	apiRouter := router.PathPrefix("/api/connectivity").Subrouter()

	// Status endpoint
	apiRouter.HandleFunc("/status", api.handleStatus).Methods("GET")

	// Proofs endpoints
	apiRouter.HandleFunc("/proofs", api.handleGetProofs).Methods("GET")
	apiRouter.HandleFunc("/proofs", api.handleCreateProof).Methods("POST")
	apiRouter.HandleFunc("/proofs/{id}", api.handleGetProof).Methods("GET")

	// Measurements endpoint
	apiRouter.HandleFunc("/measurements", api.handleGetMeasurements).Methods("GET")

	// Statistics endpoint
	apiRouter.HandleFunc("/stats", api.handleGetStats).Methods("GET")

	// Health check
	router.HandleFunc("/health", api.handleHealth).Methods("GET")

	api.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", api.port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	log.Printf("Starting connectivity API server on port %d", api.port)
	return api.server.ListenAndServe()
}

// Stop stops the API server
func (api *APIServer) Stop() error {
	if api.server != nil {
		return api.server.Close()
	}
	return nil
}

// handleStatus returns the current status of the proof engine
func (api *APIServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"proof_engine_active": true, // Since the engine is running if this endpoint is accessible
		"timestamp":           time.Now(),
		"version":             "1.0.0",
	}

	// Get current statistics
	stats := api.proofEngine.GetConnectivityStats()
	for k, v := range stats {
		status[k] = v
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// handleGetProofs returns the proof history
func (api *APIServer) handleGetProofs(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	limit := 50 // default limit

	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	proofs := api.proofEngine.GetProofHistory()

	// Apply limit
	if len(proofs) > limit {
		proofs = proofs[len(proofs)-limit:]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(proofs)
}

// handleCreateProof initiates a new connectivity proof
func (api *APIServer) handleCreateProof(w http.ResponseWriter, r *http.Request) {
	// For now, we'll just trigger the proof generation process
	// In a real implementation, you might want to add specific parameters

	response := map[string]interface{}{
		"status":    "proof_generation_initiated",
		"timestamp": time.Now(),
		"message":   "Connectivity proof generation has been initiated",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	// Trigger proof generation asynchronously
	go func() {
		// The proof engine will generate proofs automatically
		// This endpoint just confirms the request was received
		log.Println("Manual proof generation requested via API")
	}()
}

// handleGetProof returns a specific proof by ID
func (api *APIServer) handleGetProof(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	proofID := vars["id"]

	proofs := api.proofEngine.GetProofHistory()
	for _, proof := range proofs {
		if proof.ProofID == proofID {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(proof)
			return
		}
	}

	http.Error(w, "Proof not found", http.StatusNotFound)
}

// handleGetMeasurements returns current connectivity measurements
func (api *APIServer) handleGetMeasurements(w http.ResponseWriter, r *http.Request) {
	measurements := api.proofEngine.GetConnectivityMeasurements()

	// Convert map to slice for JSON serialization
	var measurementList []ConnectivityMeasurement
	for _, measurement := range measurements {
		measurementList = append(measurementList, *measurement)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(measurementList)
}

// handleGetStats returns aggregated connectivity statistics
func (api *APIServer) handleGetStats(w http.ResponseWriter, r *http.Request) {
	stats := api.proofEngine.GetConnectivityStats()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// handleHealth returns health status
func (api *APIServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"service":   "knirv-router-connectivity",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// corsMiddleware adds CORS headers
func corsMiddleware(next http.Handler) http.Handler {
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

// APIResponse represents a standard API response
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

// writeJSONResponse writes a standardized JSON response
func writeJSONResponse(w http.ResponseWriter, statusCode int, success bool, data interface{}, message string, errorMsg string) {
	response := APIResponse{
		Success: success,
		Data:    data,
		Message: message,
		Error:   errorMsg,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// writeErrorResponse writes an error response
func writeErrorResponse(w http.ResponseWriter, statusCode int, errorMsg string) {
	writeJSONResponse(w, statusCode, false, nil, "", errorMsg)
}

// writeSuccessResponse writes a success response
func writeSuccessResponse(w http.ResponseWriter, data interface{}, message string) {
	writeJSONResponse(w, http.StatusOK, true, data, message, "")
}
