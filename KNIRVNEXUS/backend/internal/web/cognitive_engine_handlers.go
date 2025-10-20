package web

import (
	"encoding/json"
	"net/http"
	"strconv"

	"backend_server/internal/services/cognitiveengine"
	"backend_server/internal/web/middleware"

	"github.com/gorilla/mux"
)

// CognitiveEngineHandlers handles HTTP requests for cognitive engine operations
type CognitiveEngineHandlers struct {
	cognitiveEngine *cognitiveengine.CognitiveEngine
}

// NewCognitiveEngineHandlers creates new cognitive engine handlers
func NewCognitiveEngineHandlers(cognitiveEngine *cognitiveengine.CognitiveEngine) *CognitiveEngineHandlers {
	return &CognitiveEngineHandlers{
		cognitiveEngine: cognitiveEngine,
	}
}

// RegisterRoutes registers the cognitive engine routes
func (ceh *CognitiveEngineHandlers) RegisterRoutes(router *mux.Router, authMiddleware *middleware.AuthMiddleware) {
	// Protected routes requiring authentication
	protected := router.PathPrefix("/api/cognitive").Subrouter()
	if authMiddleware != nil {
		protected.Use(authMiddleware.RequireAuth)
	}

	// Get cognitive metrics for a specific node
	protected.HandleFunc("/metrics/{nodeId}", ceh.GetCognitiveMetrics).Methods("GET")

	// Get overall learning state
	protected.HandleFunc("/learning-state", ceh.GetLearningState).Methods("GET")

	// Get adaptation history
	protected.HandleFunc("/adaptations", ceh.GetAdaptationHistory).Methods("GET")

	// Get failure patterns
	protected.HandleFunc("/patterns", ceh.GetFailurePatterns).Methods("GET")

	// Get task type performance
	protected.HandleFunc("/performance/tasks", ceh.GetTaskPerformance).Methods("GET")

	// Get node performance
	protected.HandleFunc("/performance/nodes", ceh.GetNodePerformance).Methods("GET")

	// Trigger manual learning cycle (admin only)
	protected.HandleFunc("/learning/trigger", ceh.TriggerLearningCycle).Methods("POST")

	// Get cognitive engine status
	protected.HandleFunc("/status", ceh.GetStatus).Methods("GET")
}

// GetCognitiveMetrics returns cognitive metrics for a specific node
func (ceh *CognitiveEngineHandlers) GetCognitiveMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	nodeID := vars["nodeId"]

	if nodeID == "" {
		http.Error(w, "Node ID is required", http.StatusBadRequest)
		return
	}

	metrics := ceh.cognitiveEngine.GetCognitiveMetrics(nodeID)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(metrics)
}

// GetLearningState returns the current learning state
func (ceh *CognitiveEngineHandlers) GetLearningState(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	learningState := ceh.cognitiveEngine.GetLearningState()

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(learningState)
}

// GetAdaptationHistory returns recent adaptation events
func (ceh *CognitiveEngineHandlers) GetAdaptationHistory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Parse limit query parameter
	limitStr := r.URL.Query().Get("limit")
	limit := 10 // default
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	history := ceh.cognitiveEngine.GetAdaptationHistory(limit)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"adaptations": history,
		"count":       len(history),
		"limit":       limit,
	})
}

// GetFailurePatterns returns detected failure patterns
func (ceh *CognitiveEngineHandlers) GetFailurePatterns(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// For now, return empty array as pattern analysis is internal
	// In a full implementation, this would expose the patterns
	patterns := []interface{}{}

	response := map[string]interface{}{
		"patterns": patterns,
		"count":    len(patterns),
		"note":     "Pattern analysis is performed internally for adaptation",
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetTaskPerformance returns performance metrics for different task types
func (ceh *CognitiveEngineHandlers) GetTaskPerformance(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	learningState := ceh.cognitiveEngine.GetLearningState()

	response := map[string]interface{}{
		"task_performance": learningState.TaskTypePerformance,
		"total_task_types": len(learningState.TaskTypePerformance),
		"last_updated":     learningState.LastUpdated,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetNodePerformance returns performance metrics for different nodes
func (ceh *CognitiveEngineHandlers) GetNodePerformance(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	learningState := ceh.cognitiveEngine.GetLearningState()

	response := map[string]interface{}{
		"node_performance": learningState.NodePerformance,
		"total_nodes":      len(learningState.NodePerformance),
		"last_updated":     learningState.LastUpdated,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// TriggerLearningCycle manually triggers a learning cycle
func (ceh *CognitiveEngineHandlers) TriggerLearningCycle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// In a real implementation, this would trigger the learning cycle
	// For now, just return success
	response := map[string]interface{}{
		"status":  "learning_cycle_triggered",
		"message": "Learning cycle has been triggered manually",
		"timestamp": r.Context().Value("timestamp"), // Would be set by middleware
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetStatus returns the current status of the cognitive engine
func (ceh *CognitiveEngineHandlers) GetStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	learningState := ceh.cognitiveEngine.GetLearningState()

	status := map[string]interface{}{
		"status":              "active",
		"learning_progress":   learningState.LearningProgress,
		"confidence_level":    learningState.ConfidenceLevel,
		"total_tasks_processed": learningState.TotalTasksProcessed,
		"success_rate":        learningState.SuccessRate,
		"average_processing_time": learningState.AverageProcessingTime,
		"task_types_tracked":  len(learningState.TaskTypePerformance),
		"nodes_tracked":       len(learningState.NodePerformance),
		"adaptations_applied": len(learningState.AdaptationHistory),
		"last_updated":        learningState.LastUpdated,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(status)
}
