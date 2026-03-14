package dvemanager

import (
	"backend_server/internal/objects"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
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

	node, err := dm.GetNode(nodeID)
	if err != nil {
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
	var task objects.ValidationTask
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
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
	}

	writeJSON(w, http.StatusOK, status)
}

// Metrics Handlers

// HandleGetMetrics handles metrics requests
func (dm *DVEManager) HandleGetMetrics(w http.ResponseWriter, r *http.Request) {
	metrics := map[string]interface{}{
		"response_time":    dm.calculateAverageResponseTime(),
		"network_latency":  dm.calculateNetworkLatency(),
		"tee_health_score": dm.calculateTEEHealthScore(),
		"timestamp":        time.Now().UTC().Format(time.RFC3339),
	}

	writeJSON(w, http.StatusOK, metrics)
}

// HandleDeleteNode handles node deletion requests
func (dm *DVEManager) HandleDeleteNode(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeID := vars["id"]

	if err := dm.RemoveNode(nodeID); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Node deleted successfully"})
}

// HandleGetNodeTasks handles getting all tasks for a specific node
func (dm *DVEManager) HandleGetNodeTasks(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeID := vars["nodeId"]

	node, err := dm.GetNode(nodeID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Node not found")
		return
	}

	tasks, err := dm.GetNodeTasks(nodeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"node_id":   nodeID,
		"node_name": node.Name,
		"tasks":     tasks,
		"total":     len(tasks),
	})
}

// HandleCreateNodeTask handles creating a new task for a specific node
func (dm *DVEManager) HandleCreateNodeTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeID := vars["nodeId"]

	node, err := dm.GetNode(nodeID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Node not found")
		return
	}

	var task objects.ValidationTask
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	task.AssignedNodeID = nodeID
	task.Status = "pending"
	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()

	if err := dm.CreateTask(&task); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"node_id":   nodeID,
		"node_name": node.Name,
		"task":      task,
	})
}

// HandleGetNodeTask handles getting a specific task for a node
func (dm *DVEManager) HandleGetNodeTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeID := vars["nodeId"]
	taskID := vars["taskId"]

	node, err := dm.GetNode(nodeID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Node not found")
		return
	}

	task, err := dm.GetTask(taskID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Task not found")
		return
	}

	if task.AssignedNodeID != nodeID {
		writeError(w, http.StatusNotFound, "Task not found for this node")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"node_id":   nodeID,
		"node_name": node.Name,
		"task":      task,
	})
}

// HandleUpdateNodeTask handles updating a task for a node
func (dm *DVEManager) HandleUpdateNodeTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeID := vars["nodeId"]
	taskID := vars["taskId"]

	task, err := dm.GetTask(taskID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Task not found")
		return
	}

	if task.AssignedNodeID != nodeID {
		writeError(w, http.StatusNotFound, "Task not found for this node")
		return
	}

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	updatedTask, err := dm.UpdateTask(taskID, updates)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, updatedTask)
}

// HandleGetNodeMetrics handles getting metrics for a specific node
func (dm *DVEManager) HandleGetNodeMetrics(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeID := vars["nodeId"]

	node, err := dm.GetNode(nodeID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Node not found")
		return
	}

	metrics, err := dm.GetNodeMetrics(nodeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"node_id":   nodeID,
		"node_name": node.Name,
		"metrics":   metrics,
		"timestamp": time.Now(),
	})
}

// HandleGetNodeMetricsHistory handles getting historical metrics for a node
func (dm *DVEManager) HandleGetNodeMetricsHistory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeID := vars["nodeId"]

	node, err := dm.GetNode(nodeID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Node not found")
		return
	}

	limit := r.URL.Query().Get("limit")
	history, err := dm.GetNodeMetricsHistory(nodeID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"node_id":   nodeID,
		"node_name": node.Name,
		"history":   history,
		"timestamp": time.Now(),
	})
}

// HandleListTasks handles listing all tasks
func (dm *DVEManager) HandleListTasks(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	nodeID := r.URL.Query().Get("node_id")

	tasks, err := dm.ListTasks(status, nodeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"tasks": tasks,
		"total": len(tasks),
	})
}

// HandleGetTask handles getting a specific task
func (dm *DVEManager) HandleGetTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	task, err := dm.GetTask(taskID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Task not found")
		return
	}

	writeJSON(w, http.StatusOK, task)
}

// HandleGetNetworkTopology handles getting the network topology
func (dm *DVEManager) HandleGetNetworkTopology(w http.ResponseWriter, r *http.Request) {
	nodes := dm.GetAllNodes()

	nodeLinks := make([]map[string]interface{}, 0)
	nodeList := make([]map[string]interface{}, 0)

	for _, node := range nodes {
		nodeInfo := map[string]interface{}{
			"id":       node.ID,
			"name":     node.Name,
			"status":   node.Status,
			"location": node.Location,
			"ip":       node.IPAddress,
			"tee_type": node.TEEType,
		}
		nodeList = append(nodeList, nodeInfo)
	}

	if dm.p2pManager != nil {
		peers := dm.p2pManager.GetConnectedPeers()
		for _, peer := range peers {
			link := map[string]interface{}{
				"peer_id": peer.ID,
				"type":    "p2p",
			}
			nodeLinks = append(nodeLinks, link)
		}
	}

	topology := map[string]interface{}{
		"nodes": nodeList,
		"links": nodeLinks,
	}

	writeJSON(w, http.StatusOK, topology)
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
