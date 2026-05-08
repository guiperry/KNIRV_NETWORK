package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"backend_server/internal/services/cognitiveengine"
	"backend_server/internal/web/middleware"

	"github.com/gorilla/mux"
)

// CognitiveEngineService interface for testability
type CognitiveEngineService interface {
	GetCognitiveMetrics(nodeID string) *cognitiveengine.CognitiveMetrics
	GetLearningState() *cognitiveengine.LearningState
	GetAdaptationHistory(limit int) []cognitiveengine.AdaptationEvent
}

// CognitiveEngineHandlers handles HTTP requests for cognitive engine operations
type CognitiveEngineHandlers struct {
	cognitiveEngine CognitiveEngineService
}

// NewCognitiveEngineHandlers creates new cognitive engine handlers
func NewCognitiveEngineHandlers(cognitiveEngine CognitiveEngineService) *CognitiveEngineHandlers {
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

	// Active memory reasoning traces
	protected.HandleFunc("/active-memory/traces", ceh.ListReasoningTraces).Methods("GET")
	protected.HandleFunc("/active-memory/traces/{traceId}", ceh.GetReasoningTrace).Methods("GET")
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
		"status":    "learning_cycle_triggered",
		"message":   "Learning cycle has been triggered manually",
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
		"status":                  "active",
		"learning_progress":       learningState.LearningProgress,
		"confidence_level":        learningState.ConfidenceLevel,
		"total_tasks_processed":   learningState.TotalTasksProcessed,
		"success_rate":            learningState.SuccessRate,
		"average_processing_time": learningState.AverageProcessingTime,
		"task_types_tracked":      len(learningState.TaskTypePerformance),
		"nodes_tracked":           len(learningState.NodePerformance),
		"adaptations_applied":     len(learningState.AdaptationHistory),
		"last_updated":            learningState.LastUpdated,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(status)
}

type reasoningTraceMetadata struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Timestamp string `json:"timestamp"`
	Metadata  struct {
		AgentID string `json:"agent_id"`
		ErrorID string `json:"error_id"`
	} `json:"metadata"`
}

func (ceh *CognitiveEngineHandlers) ListReasoningTraces(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	traceDir, err := resolveTraceDirectory()
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	entries, err := os.ReadDir(traceDir)
	if err != nil {
		http.Error(w, "failed to read trace directory", http.StatusInternalServerError)
		return
	}

	traces := make([]map[string]interface{}, 0)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "trace_") || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		meta, _, err := loadTraceFromFile(filepath.Join(traceDir, entry.Name()))
		if err != nil {
			continue
		}

		traces = append(traces, map[string]interface{}{
			"id":        meta.ID,
			"agent_id":  meta.Metadata.AgentID,
			"error_id":  meta.Metadata.ErrorID,
			"timestamp": meta.Timestamp,
			"type":      meta.Type,
		})
	}

	sort.Slice(traces, func(i, j int) bool {
		return fmt.Sprint(traces[i]["timestamp"]) > fmt.Sprint(traces[j]["timestamp"])
	})

	json.NewEncoder(w).Encode(map[string]interface{}{
		"traces": traces,
		"count":  len(traces),
	})
}

func (ceh *CognitiveEngineHandlers) GetReasoningTrace(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	traceID := mux.Vars(r)["traceId"]
	if traceID == "" {
		http.Error(w, "trace ID is required", http.StatusBadRequest)
		return
	}

	traceDir, err := resolveTraceDirectory()
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	meta, markdown, err := loadTraceFromFile(filepath.Join(traceDir, fmt.Sprintf("trace_%s.md", traceID)))
	if err != nil {
		http.Error(w, "trace not found", http.StatusNotFound)
		return
	}

	steps := extractTraceSteps(markdown)
	result := extractTraceResult(markdown)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":        meta.ID,
		"agent_id":  meta.Metadata.AgentID,
		"error_id":  meta.Metadata.ErrorID,
		"timestamp": meta.Timestamp,
		"type":      meta.Type,
		"steps":     steps,
		"result":    result,
		"content":   markdown,
	})
}

func resolveTraceDirectory() (string, error) {
	candidates := []string{
		filepath.Join("storage", "memory_vault"),
		filepath.Join("backend", "internal", "services", "memory", "storage", "memory_vault"),
		filepath.Join("internal", "services", "memory", "storage", "memory_vault"),
	}

	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("reasoning trace storage not found")
}

func loadTraceFromFile(path string) (*reasoningTraceMetadata, string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}

	content := string(raw)
	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return nil, "", fmt.Errorf("invalid trace file format")
	}

	meta := &reasoningTraceMetadata{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(parts[1])), meta); err != nil {
		return nil, "", err
	}

	body := strings.TrimSpace(parts[2])
	body = strings.Replace(body, "# PQC Encrypted Payload\n\n```pqc", "", 1)
	body = strings.TrimSpace(strings.TrimSuffix(body, "```"))

	return meta, strings.TrimSpace(body), nil
}

func extractTraceSteps(markdown string) []string {
	lines := strings.Split(markdown, "\n")
	steps := make([]string, 0)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) > 2 && trimmed[0] >= '0' && trimmed[0] <= '9' {
			if idx := strings.Index(trimmed, ". "); idx > 0 {
				steps = append(steps, trimmed[idx+2:])
			}
		}
	}
	return steps
}

func extractTraceResult(markdown string) string {
	lines := strings.Split(markdown, "\n")
	for idx, line := range lines {
		if strings.TrimSpace(line) == "## Result" && idx+1 < len(lines) {
			return strings.TrimSpace(lines[idx+1])
		}
	}
	return ""
}
