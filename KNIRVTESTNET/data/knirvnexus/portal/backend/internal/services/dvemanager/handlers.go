package dvemanager

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"

	"nexus-backend/internal/models"
)

// DVE Node Handlers

// HandleRegisterNode handles node registration requests
func (dm *DVEManager) HandleRegisterNode(w http.ResponseWriter, r *http.Request) {
	var req RegisterNodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "Node name is required")
		return
	}

	if req.TEEType == "" {
		writeError(w, http.StatusBadRequest, "TEE type is required")
		return
	}

	node, err := dm.RegisterNode(&req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, node)
}

// HandleListNodes handles node listing requests
func (dm *DVEManager) HandleListNodes(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters for filtering
	filter := &NodeFilter{
		Status:   r.URL.Query().Get("status"),
		TEEType:  r.URL.Query().Get("tee_type"),
		Location: r.URL.Query().Get("location"),
	}

	nodes, err := dm.GetNodes(filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, nodes)
}

// HandleGetNode handles individual node retrieval requests
func (dm *DVEManager) HandleGetNode(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeID := vars["id"]

	node, exists := dm.GetNode(nodeID)
	if !exists {
		writeError(w, http.StatusNotFound, "Node not found")
		return
	}

	writeJSON(w, http.StatusOK, node)
}

// HandleUpdateNodeStatus handles node status update requests
func (dm *DVEManager) HandleUpdateNodeStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeID := vars["id"]

	var req struct {
		Status string `json:"status"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Status == "" {
		writeError(w, http.StatusBadRequest, "Status is required")
		return
	}

	if err := dm.UpdateNodeStatus(nodeID, req.Status); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Node status updated successfully"})
}

// Task Allocation Handlers

// HandleAllocateTask handles task allocation requests
func (dm *DVEManager) HandleAllocateTask(w http.ResponseWriter, r *http.Request) {
	var task models.ValidationTask
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	node, err := dm.AllocateTask(&task)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := map[string]interface{}{
		"task":           task,
		"allocated_node": node,
		"message":        "Task allocated successfully",
	}

	writeJSON(w, http.StatusOK, response)
}

// System Health Handlers

// HandleGetSystemHealth handles system health requests
func (dm *DVEManager) HandleGetSystemHealth(w http.ResponseWriter, r *http.Request) {
	health, err := dm.GetSystemHealth()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, health)
}

// HandleGetSystemStatus handles system status requests
func (dm *DVEManager) HandleGetSystemStatus(w http.ResponseWriter, r *http.Request) {
	allNodes := dm.GetAllNodes()
	activeNodes := 0
	for _, node := range allNodes {
		if node.Status == "active" {
			activeNodes++
		}
	}

	status := map[string]interface{}{
		"total_nodes":  len(allNodes),
		"active_nodes": activeNodes,
		"status":       dm.calculateOverallStatus(activeNodes, len(allNodes)),
		"timestamp":    "2024-01-01T00:00:00Z", // TODO: Use actual timestamp
	}

	writeJSON(w, http.StatusOK, status)
}

// Metrics Handlers

// HandleGetMetrics handles metrics requests
func (dm *DVEManager) HandleGetMetrics(w http.ResponseWriter, r *http.Request) {
	// This would return detailed metrics
	// For now, return basic metrics
	metrics := map[string]interface{}{
		"response_time":    dm.calculateAverageResponseTime(),
		"network_latency":  dm.calculateNetworkLatency(),
		"tee_health_score": dm.calculateTEEHealthScore(),
		"timestamp":        "2024-01-01T00:00:00Z", // TODO: Use actual timestamp
	}

	writeJSON(w, http.StatusOK, metrics)
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
