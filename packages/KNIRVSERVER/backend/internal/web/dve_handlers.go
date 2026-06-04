package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"backend_server/internal/objects"
	"backend_server/internal/services/dvecreation"
	"backend_server/internal/services/dvemanager"
	"backend_server/internal/services/session"
	"backend_server/internal/web/middleware"

	"github.com/gorilla/mux"
)

// wsBroadcaster is a minimal interface for pushing real-time events to connected clients.
type wsBroadcaster interface {
	Broadcast(event string, payload interface{})
}

type DVEHandlers struct {
	dveManager         *dvemanager.DVEManager
	dveCreationService *dvecreation.DVECreationService
	sessionManager     *session.SessionManager
	wsBroadcaster      wsBroadcaster
	agentService       interface {
		BroadcastAgentMessage(dveID string, message string)
	}
	knirvagentManager interface {
		IsRunning() bool
		HealthCheck(ctx context.Context) error
		GetBaseURL() string
		StartAgent(ctx context.Context, dveID string, startTimeout time.Duration) error
		RunningCount() int
		GetSocketPathForDVE(dveID string) (string, error)
	}
	workspaceFSHandlers *WorkspaceFSHandlers
}

// NewDVEHandlers creates the DVE HTTP handler set.  DVE creation operations are
// accessed through dveManager's delegation methods (merged from dvecreation).
func NewDVEHandlers(dveManager *dvemanager.DVEManager, dveCreationService *dvecreation.DVECreationService) *DVEHandlers {
	return &DVEHandlers{
		dveManager:         dveManager,
		dveCreationService: dveCreationService,
	}
}

// SetWebSocketService wires in a broadcaster so mutations immediately notify connected clients.
func (h *DVEHandlers) SetWebSocketService(ws wsBroadcaster) {
	h.wsBroadcaster = ws
}

// SetSessionManager wires in the session manager for SSH session creation.
func (h *DVEHandlers) SetSessionManager(sm *session.SessionManager) {
	h.sessionManager = sm
}

// SetAgentService wires in the agent service for broadcasting responses.
func (h *DVEHandlers) SetAgentService(as interface {
	BroadcastAgentMessage(dveID string, message string)
}) {
	h.agentService = as
}

// SetKnirvagentManager wires in the KNIRVAGENT supervisor manager for status endpoints.
func (h *DVEHandlers) SetKnirvagentManager(mgr interface {
	IsRunning() bool
	HealthCheck(ctx context.Context) error
	GetBaseURL() string
	StartAgent(ctx context.Context, dveID string, startTimeout time.Duration) error
	RunningCount() int
	GetSocketPathForDVE(dveID string) (string, error)
}) {
	h.knirvagentManager = mgr
}

// GetDVEWorkers handles GET /api/dve/workers — aggregates DVE nodes + tasks as active workers
func (h *DVEHandlers) GetDVEWorkers(w http.ResponseWriter, r *http.Request) {
	type ActiveWorker struct {
		ID             string            `json:"id"`
		Name           string            `json:"name"`
		Type           string            `json:"type"`
		Status         string            `json:"status"`
		LastActivity   string            `json:"lastActivity"`
		TasksCompleted int               `json:"tasksCompleted"`
		CPUUsage       float64           `json:"cpuUsage,omitempty"`
		MemoryUsage    float64           `json:"memoryUsage,omitempty"`
		Metadata       map[string]string `json:"metadata,omitempty"`
	}

	workers := make([]ActiveWorker, 0)

	if h.dveManager != nil {
		nodes := h.dveManager.GetAllNodes()
		for _, node := range nodes {
			workerStatus := "idle"
			switch node.Status {
			case "online", "active":
				workerStatus = "active"
			case "offline":
				workerStatus = "disconnected"
			case "error":
				workerStatus = "error"
			}

			tasks, _ := h.dveManager.GetNodeTasks(node.ID)
			completed := 0
			for _, t := range tasks {
				if t.Status == "completed" {
					completed++
				}
			}

			workers = append(workers, ActiveWorker{
				ID:             node.ID,
				Name:           node.Name,
				Type:           "agent",
				Status:         workerStatus,
				LastActivity:   node.LastHeartbeat.Format(time.RFC3339),
				TasksCompleted: completed,
				CPUUsage:       node.CPUUsage,
				MemoryUsage:    node.MemoryUsage,
				Metadata: map[string]string{
					"tee_type": node.TEEType,
					"location": node.Location,
				},
			})
		}

		// Add running tasks as workflow workers
		allTasks, err := h.dveManager.ListTasks("running", "")
		if err == nil {
			for _, task := range allTasks {
				workers = append(workers, ActiveWorker{
					ID:           task.ID,
					Name:         fmt.Sprintf("Task: %s", task.Type),
					Type:         "workflow",
					Status:       "active",
					LastActivity: task.UpdatedAt.Format(time.RFC3339),
					Metadata: map[string]string{
						"node_id": task.AssignedNodeID,
						"type":    task.Type,
					},
				})
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"workers":   workers,
		"total":     len(workers),
		"timestamp": getCurrentTimestamp(),
	})
}

// GetDVENodeTasksAlias handles GET /api/dve/{nodeId}/tasks (alias for monitor-panel compatibility)
func (h *DVEHandlers) GetDVENodeTasksAlias(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeID := vars["nodeId"]

	if h.dveManager == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"node_id":   nodeID,
			"tasks":     []interface{}{},
			"total":     0,
			"timestamp": getCurrentTimestamp(),
		})
		return
	}

	tasks, err := h.dveManager.GetNodeTasks(nodeID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	if tasks == nil {
		tasks = []*objects.ValidationTask{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"node_id": nodeID,
		"tasks":   tasks,
		"total":   len(tasks),
	})
}

// GetDVENodeMetricsAlias handles GET /api/dve/{nodeId}/metrics (alias for monitor-panel compatibility)
func (h *DVEHandlers) GetDVENodeMetricsAlias(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeID := vars["nodeId"]

	if h.dveManager == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"node_id":   nodeID,
			"metrics":   map[string]interface{}{},
			"timestamp": getCurrentTimestamp(),
		})
		return
	}

	metrics, err := h.dveManager.GetNodeMetrics(nodeID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	metrics["timestamp"] = getCurrentTimestamp()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// GetP2PPeers handles GET /api/p2p/peers — returns connected P2P peer list from P2PManager
func (h *DVEHandlers) GetP2PPeers(w http.ResponseWriter, r *http.Request) {
	type PeerEntry struct {
		ID      string `json:"id"`
		Address string `json:"address"`
		Role    string `json:"role"`
		Status  string `json:"status"`
		Latency int64  `json:"latency_ms"`
	}

	peers := make([]PeerEntry, 0)

	if h.dveManager != nil {
		rawPeers := h.dveManager.GetConnectedP2PPeers()
		for _, p := range rawPeers {
			status := p.Status
			if status == "" {
				status = "connected"
			}
			peers = append(peers, PeerEntry{
				ID:      p.ID,
				Address: p.Address,
				Role:    p.Role,
				Status:  status,
				Latency: p.Latency,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"peers":     peers,
		"total":     len(peers),
		"timestamp": getCurrentTimestamp(),
	})
}

// PostAgentResponse handles POST /api/v1/dve/{nodeId}/agent/response
func (h *DVEHandlers) PostAgentResponse(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeID := vars["nodeId"]

	var req struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if h.agentService != nil {
		// Use an interface assertion or just call it if we know the type
		type broadcaster interface {
			BroadcastAgentMessage(dveID string, message string)
		}
		if b, ok := h.agentService.(broadcaster); ok {
			b.BroadcastAgentMessage(nodeID, req.Message)
		}
	}

	w.WriteHeader(http.StatusOK)
}

// CreateNodeSSHSession handles POST /api/dve/{nodeId}/ssh-session (Gap 1)
// Creates an SSH session for a DVE node so the ConsolePanel can connect via WebSocket.
func (h *DVEHandlers) CreateNodeSSHSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeID := vars["nodeId"]

	w.Header().Set("Content-Type", "application/json")

	if h.sessionManager == nil || h.dveManager == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": "session manager not available"})
		return
	}

	node, err := h.dveManager.GetNode(nodeID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "node not found"})
		return
	}

	// Decode optional request body for username/key overrides
	var req struct {
		Username   string `json:"username"`
		PrivateKey string `json:"private_key"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Username == "" {
		req.Username = "dve-admin"
	}
	if req.PrivateKey == "" {
		// Use node public key as placeholder; real deployment provides actual private key
		req.PrivateKey = node.PublicKey
	}

	sess, err := h.sessionManager.CreateSSHSession(nodeID, node.ID, req.Username, req.PrivateKey)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to create session: " + err.Error()})
		return
	}

	// Populate endpoint details from the node
	sess.Endpoint = node.IPAddress
	port := node.SSHPort
	if port == 0 {
		port = 22
	}
	sess.Port = port

	json.NewEncoder(w).Encode(map[string]interface{}{
		"session_id": sess.ID,
		"endpoint":   sess.Endpoint,
		"port":       sess.Port,
		"username":   sess.Username,
		"expires_at": sess.ExpiresAt,
		"ws_url":     fmt.Sprintf("/ws/ssh/%s", sess.ID),
	})
}

// getCurrentTimestamp returns the current timestamp in RFC3339 format
func getCurrentTimestamp() string {
	return time.Now().Format(time.RFC3339)
}

type DVENodeResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Total     int         `json:"total,omitempty"`
	Message   string      `json:"message,omitempty"`
	Error     string      `json:"error,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// GetDVENodes handles GET /api/dve-nodes
func (h *DVEHandlers) GetDVENodes(w http.ResponseWriter, r *http.Request) {
	if h.dveManager == nil {
		// Return empty nodes list instead of error when DVE manager is not initialized
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"nodes":     []interface{}{},
			"total":     0,
			"timestamp": getCurrentTimestamp(),
		})
		return
	}

	// Parse query parameters for filtering
	filter := &dvemanager.NodeFilter{
		Status:   r.URL.Query().Get("status"),
		TEEType:  r.URL.Query().Get("tee_type"),
		Location: r.URL.Query().Get("location"),
	}

	// Parse limit parameter
	limitStr := r.URL.Query().Get("limit")
	if limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			filter.Limit = limit
		}
	}

	nodes, err := h.dveManager.GetNodes(filter)
	if err != nil {
		// If error is "not found", return empty array instead of 500 error
		if strings.Contains(err.Error(), "not found") {
			response := DVENodeResponse{
				Success:   true,
				Data:      []interface{}{}, // Empty array
				Total:     0,
				Timestamp: getCurrentTimestamp(),
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		// For other errors, still return 500
		response := DVENodeResponse{
			Success:   false,
			Error:     "Failed to fetch DVE nodes: " + err.Error(),
			Timestamp: getCurrentTimestamp(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Apply role-based filtering for observer (Developer) role.
	// When the observer has existing creations, show only their nodes.
	// When they have no creations yet (new user / demo state), show all nodes
	// so that demo DVE nodes are visible and the dashboard is not empty.
	authCtx := middleware.GetAuthContext(r)
	if authCtx != nil && authCtx.Role == "observer" && h.dveManager != nil {
		creations, err := h.dveCreationService.GetUserDVECreations(authCtx.UserID)
		if err != nil || len(creations) == 0 {
			// No creations yet — fall through to return all nodes (including demo nodes).
		} else {
			// User has creations: filter to only their nodes.
			userNodeMap := make(map[string]bool)
			for _, creation := range creations {
				userNodeMap[creation.DVENodeID] = true
			}

			filteredNodes := make([]interface{}, 0)
			for _, node := range nodes {
				if userNodeMap[node.ID] {
					filteredNodes = append(filteredNodes, node)
				}
			}

			response := DVENodeResponse{
				Success:   true,
				Data:      filteredNodes,
				Total:     len(filteredNodes),
				Timestamp: getCurrentTimestamp(),
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}
	}

	// For admin, validator roles, and observers with no creations, return all nodes.
	response := DVENodeResponse{
		Success:   true,
		Data:      nodes,
		Total:     len(nodes),
		Timestamp: getCurrentTimestamp(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// PostDVENodes handles POST /api/dve-nodes (node registration)
func (h *DVEHandlers) PostDVENodes(w http.ResponseWriter, r *http.Request) {
	var req objects.RegisterNodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response := DVENodeResponse{
			Success:   false,
			Error:     "Invalid request body",
			Timestamp: getCurrentTimestamp(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Validate required fields
	if req.Name == "" {
		response := DVENodeResponse{
			Success:   false,
			Error:     "Node name is required",
			Timestamp: getCurrentTimestamp(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	if req.TEEType == "" {
		response := DVENodeResponse{
			Success:   false,
			Error:     "TEE type is required",
			Timestamp: getCurrentTimestamp(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	node, err := h.dveManager.RegisterNode(&req)
	if err != nil {
		response := DVENodeResponse{
			Success:   false,
			Error:     "Failed to register node: " + err.Error(),
			Timestamp: getCurrentTimestamp(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	if h.wsBroadcaster != nil {
		h.wsBroadcaster.Broadcast("dve-node-discovered", node)
	}

	response := DVENodeResponse{
		Success:   true,
		Data:      node,
		Message:   "DVE node registered successfully",
		Timestamp: getCurrentTimestamp(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// GetDVENode handles GET /api/dve-nodes/{id}
func (h *DVEHandlers) GetDVENode(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeID := vars["id"]

	if nodeID == "" {
		response := DVENodeResponse{
			Success:   false,
			Error:     "Node ID is required",
			Timestamp: getCurrentTimestamp(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	if h.dveManager == nil {
		response := DVENodeResponse{
			Success:   false,
			Error:     "Node not found",
			Timestamp: getCurrentTimestamp(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	node, err := h.dveManager.GetNode(nodeID)
	if err != nil {
		if strings.Contains(err.Error(), "node not found") {
			response := DVENodeResponse{
				Success:   false,
				Error:     "Node not found",
				Timestamp: getCurrentTimestamp(),
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(response)
			return
		} else {
			response := DVENodeResponse{
				Success:   false,
				Error:     "Failed to fetch node: " + err.Error(),
				Timestamp: getCurrentTimestamp(),
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}
	}

	response := DVENodeResponse{
		Success:   true,
		Data:      node,
		Timestamp: getCurrentTimestamp(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// UpdateDVENode handles PUT /api/dve-nodes/{id}
func (h *DVEHandlers) UpdateDVENode(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeID := vars["id"]

	if nodeID == "" {
		response := DVENodeResponse{
			Success:   false,
			Error:     "Node ID is required",
			Timestamp: getCurrentTimestamp(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	var updateReq map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
		response := DVENodeResponse{
			Success:   false,
			Error:     "Invalid request body",
			Timestamp: getCurrentTimestamp(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	node, err := h.dveManager.UpdateNode(nodeID, updateReq)
	if err != nil {
		response := DVENodeResponse{
			Success:   false,
			Error:     "Failed to update node: " + err.Error(),
			Timestamp: getCurrentTimestamp(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	if h.wsBroadcaster != nil {
		h.wsBroadcaster.Broadcast("dve-node-updated", node)
	}

	response := DVENodeResponse{
		Success:   true,
		Data:      node,
		Message:   "DVE node updated successfully",
		Timestamp: getCurrentTimestamp(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// DeleteDVENode handles DELETE /api/dve-nodes/{id}
func (h *DVEHandlers) DeleteDVENode(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeID := vars["id"]

	if nodeID == "" {
		response := DVENodeResponse{
			Success:   false,
			Error:     "Node ID is required",
			Timestamp: getCurrentTimestamp(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	err := h.dveManager.RemoveNode(nodeID)
	if err != nil {
		response := DVENodeResponse{
			Success:   false,
			Error:     "Failed to delete node: " + err.Error(),
			Timestamp: getCurrentTimestamp(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := DVENodeResponse{
		Success:   true,
		Message:   "DVE node deleted successfully",
		Timestamp: getCurrentTimestamp(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetDVENodeEndpoints handles GET /api/dve-nodes/{id}/endpoints
func (h *DVEHandlers) GetDVENodeEndpoints(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeID := vars["id"]

	if nodeID == "" {
		response := DVENodeResponse{
			Success:   false,
			Error:     "Node ID is required",
			Timestamp: getCurrentTimestamp(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get node details
	node, err := h.dveManager.GetNode(nodeID)
	if err != nil {
		if strings.Contains(err.Error(), "node not found") {
			response := DVENodeResponse{
				Success:   false,
				Error:     "Node not found",
				Timestamp: getCurrentTimestamp(),
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(response)
			return
		} else {
			response := DVENodeResponse{
				Success:   false,
				Error:     "Failed to fetch node: " + err.Error(),
				Timestamp: getCurrentTimestamp(),
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}
	}

	// Return endpoint information
	endpoints := map[string]interface{}{
		"ssh": map[string]interface{}{
			"port": node.SSHPort,
			"host": node.IPAddress,
		},
		"validation": map[string]interface{}{
			"port": node.ValidationPort,
			"host": node.IPAddress,
		},
		"error_resolution": map[string]interface{}{
			"port": node.ErrorResPort,
			"host": node.IPAddress,
		},
		"supported_tags": node.SupportedTags,
	}

	response := DVENodeResponse{
		Success:   true,
		Data:      endpoints,
		Message:   "Node endpoints retrieved successfully",
		Timestamp: getCurrentTimestamp(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetDVENodeSSHEndpoint handles GET /api/dve-nodes/{id}/ssh-endpoint
func (h *DVEHandlers) GetDVENodeSSHEndpoint(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeID := vars["id"]

	if nodeID == "" {
		response := DVENodeResponse{
			Success:   false,
			Error:     "Node ID is required",
			Timestamp: getCurrentTimestamp(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	node, err := h.dveManager.GetNode(nodeID)
	if err != nil {
		if strings.Contains(err.Error(), "node not found") {
			response := DVENodeResponse{
				Success:   false,
				Error:     "Node not found",
				Timestamp: getCurrentTimestamp(),
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(response)
			return
		} else {
			response := DVENodeResponse{
				Success:   false,
				Error:     "Failed to fetch node: " + err.Error(),
				Timestamp: getCurrentTimestamp(),
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}
	}

	endpoint := map[string]interface{}{
		"port":     node.SSHPort,
		"host":     node.IPAddress,
		"protocol": "ssh",
	}

	response := DVENodeResponse{
		Success:   true,
		Data:      endpoint,
		Message:   "SSH endpoint retrieved successfully",
		Timestamp: getCurrentTimestamp(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetDVENodeValidationEndpoint handles GET /api/dve-nodes/{id}/validation-endpoint
func (h *DVEHandlers) GetDVENodeValidationEndpoint(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeID := vars["id"]

	if nodeID == "" {
		response := DVENodeResponse{
			Success:   false,
			Error:     "Node ID is required",
			Timestamp: getCurrentTimestamp(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	node, err := h.dveManager.GetNode(nodeID)
	if err != nil {
		if strings.Contains(err.Error(), "node not found") {
			response := DVENodeResponse{
				Success:   false,
				Error:     "Node not found",
				Timestamp: getCurrentTimestamp(),
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(response)
			return
		} else {
			response := DVENodeResponse{
				Success:   false,
				Error:     "Failed to fetch node: " + err.Error(),
				Timestamp: getCurrentTimestamp(),
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}
	}

	endpoint := map[string]interface{}{
		"port":     node.ValidationPort,
		"host":     node.IPAddress,
		"protocol": "http",
		"url":      fmt.Sprintf("http://%s:%d", node.IPAddress, node.ValidationPort),
	}

	response := DVENodeResponse{
		Success:   true,
		Data:      endpoint,
		Message:   "Validation endpoint retrieved successfully",
		Timestamp: getCurrentTimestamp(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetDVENodeErrorResolutionEndpoint handles GET /api/dve-nodes/{id}/error-resolution-endpoint
func (h *DVEHandlers) GetDVENodeErrorResolutionEndpoint(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeID := vars["id"]

	if nodeID == "" {
		response := DVENodeResponse{
			Success:   false,
			Error:     "Node ID is required",
			Timestamp: getCurrentTimestamp(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	node, err := h.dveManager.GetNode(nodeID)
	if err != nil {
		if strings.Contains(err.Error(), "node not found") {
			response := DVENodeResponse{
				Success:   false,
				Error:     "Node not found",
				Timestamp: getCurrentTimestamp(),
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(response)
			return
		} else {
			response := DVENodeResponse{
				Success:   false,
				Error:     "Failed to fetch node: " + err.Error(),
				Timestamp: getCurrentTimestamp(),
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}
	}

	endpoint := map[string]interface{}{
		"port":     node.ErrorResPort,
		"host":     node.IPAddress,
		"protocol": "http",
		"url":      fmt.Sprintf("http://%s:%d", node.IPAddress, node.ErrorResPort),
	}

	response := DVENodeResponse{
		Success:   true,
		Data:      endpoint,
		Message:   "Error resolution endpoint retrieved successfully",
		Timestamp: getCurrentTimestamp(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetSupervisorAgentStatus handles GET /api/dve/{nodeId}/supervisor-agent/status.
// Returns the current health status of the KNIRVAGENT supervisor for this DVE.
func (h *DVEHandlers) GetSupervisorAgentStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if h.knirvagentManager == nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "unavailable",
			"error":  "KNIRVAGENT supervisor not configured",
		})
		return
	}

	nodeID := mux.Vars(r)["nodeId"]

	// Check per-DVE status
	status := "offline"
	if _, err := h.knirvagentManager.GetSocketPathForDVE(nodeID); err == nil {
		status = "online"
	} else if h.knirvagentManager.IsRunning() {
		status = "initializing"
	}
	running := status != "offline"

	healthCtx, healthCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer healthCancel()
	healthErr := h.knirvagentManager.HealthCheck(healthCtx)
	healthStatus := "healthy"
	if healthErr != nil {
		healthStatus = "unhealthy"
	}
	// Override health if we just provisioned (socket might not be ready yet)
	if status == "initializing" {
		healthStatus = "booting"
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     status,
		"health":     healthStatus,
		"running":    running,
		"base_url":   h.knirvagentManager.GetBaseURL(),
		"agent_type": "knirvagent",
		"role":       "supervisor",
		"node_id":    nodeID,
		"timestamp":  getCurrentTimestamp(),
	})
}

// GetSupervisorAgentSession handles GET /api/dve/{nodeId}/supervisor-agent/session
// Returns a WebSocket URL for real-time communication with the KNIRVAGENT supervisor.
// The websocket upgrade phase is responsible for waiting briefly on the socket
// if the agent is still finishing startup.
func (h *DVEHandlers) GetSupervisorAgentSession(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	nodeID := mux.Vars(r)["nodeId"]

	if h.knirvagentManager == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "KNIRVAGENT supervisor not configured",
		})
		return
	}

	// The session URL points to the backend WebSocket proxy which relays
	// I/O between the frontend terminal and the KNIRVAGENT subprocess.
	wsURL := fmt.Sprintf("/ws/dve/%s/agent", nodeID)
	running := false
	if _, err := h.knirvagentManager.GetSocketPathForDVE(nodeID); err == nil {
		running = true
	} else {
		startCtx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		if err := h.knirvagentManager.StartAgent(startCtx, nodeID, 30*time.Second); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "KNIRVAGENT could not start for this DVE: " + err.Error(),
			})
			return
		}
		running = true
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"ws_url":    wsURL,
		"base_url":  h.knirvagentManager.GetBaseURL(),
		"running":   running,
		"timestamp": getCurrentTimestamp(),
	})
}

// RegisterRoutes registers the DVE routes with the router
func (h *DVEHandlers) RegisterRoutes(r *mux.Router, authMiddleware *middleware.AuthMiddleware) {
	// Create a subrouter for DVE endpoints
	dveRouter := r.PathPrefix("/api/dve-nodes").Subrouter()

	// Routes with optional auth (for role-based filtering)
	if authMiddleware != nil {
		optionalAuthRouter := dveRouter.PathPrefix("").Subrouter()
		optionalAuthRouter.Use(authMiddleware.OptionalAuth)
		optionalAuthRouter.HandleFunc("", h.GetDVENodes).Methods("GET", "OPTIONS")
		optionalAuthRouter.HandleFunc("/{id}", h.GetDVENode).Methods("GET", "OPTIONS")
		optionalAuthRouter.HandleFunc("/{id}/endpoints", h.GetDVENodeEndpoints).Methods("GET", "OPTIONS")
		optionalAuthRouter.HandleFunc("/{id}/ssh-endpoint", h.GetDVENodeSSHEndpoint).Methods("GET", "OPTIONS")
		optionalAuthRouter.HandleFunc("/{id}/validation-endpoint", h.GetDVENodeValidationEndpoint).Methods("GET", "OPTIONS")
		optionalAuthRouter.HandleFunc("/{id}/error-resolution-endpoint", h.GetDVENodeErrorResolutionEndpoint).Methods("GET", "OPTIONS")
	} else {
		// If no auth middleware, allow public access
		dveRouter.HandleFunc("", h.GetDVENodes).Methods("GET", "OPTIONS")
		dveRouter.HandleFunc("/{id}", h.GetDVENode).Methods("GET", "OPTIONS")
		dveRouter.HandleFunc("/{id}/endpoints", h.GetDVENodeEndpoints).Methods("GET", "OPTIONS")
		dveRouter.HandleFunc("/{id}/ssh-endpoint", h.GetDVENodeSSHEndpoint).Methods("GET", "OPTIONS")
		dveRouter.HandleFunc("/{id}/validation-endpoint", h.GetDVENodeValidationEndpoint).Methods("GET", "OPTIONS")
		dveRouter.HandleFunc("/{id}/error-resolution-endpoint", h.GetDVENodeErrorResolutionEndpoint).Methods("GET", "OPTIONS")
	}

	// Protected routes for management
	if authMiddleware != nil {
		protectedDVERouter := dveRouter.PathPrefix("").Subrouter()
		protectedDVERouter.Use(authMiddleware.RequireAuth)
		protectedDVERouter.HandleFunc("", h.PostDVENodes).Methods("POST", "OPTIONS")
		protectedDVERouter.HandleFunc("/{id}", h.UpdateDVENode).Methods("PUT", "OPTIONS")
		protectedDVERouter.HandleFunc("/{id}", h.DeleteDVENode).Methods("DELETE", "OPTIONS")
	} else {
		// If no auth middleware, allow all routes (for testnet mode)
		dveRouter.HandleFunc("", h.PostDVENodes).Methods("POST", "OPTIONS")
		dveRouter.HandleFunc("/{id}", h.UpdateDVENode).Methods("PUT", "OPTIONS")
		dveRouter.HandleFunc("/{id}", h.DeleteDVENode).Methods("DELETE", "OPTIONS")
	}

	// /api/dve/ aliases — frontend monitor-panel and connections-panel use these paths
	dveAliasRouter := r.PathPrefix("/api/dve").Subrouter()
	dveAliasRouter.HandleFunc("/workers", h.GetDVEWorkers).Methods("GET", "OPTIONS")
	dveAliasRouter.HandleFunc("/{nodeId}/tasks", h.GetDVENodeTasksAlias).Methods("GET", "OPTIONS")
	dveAliasRouter.HandleFunc("/{nodeId}/metrics", h.GetDVENodeMetricsAlias).Methods("GET", "OPTIONS")

	// /api/p2p/ — P2P network topology endpoints (Gap 5)
	p2pRouter := r.PathPrefix("/api/p2p").Subrouter()
	p2pRouter.HandleFunc("/peers", h.GetP2PPeers).Methods("GET", "OPTIONS")

	// SSH session creation for DVE nodes — used by ConsolePanel (Gap 1)
	dveAliasRouter.HandleFunc("/{nodeId}/ssh-session", h.CreateNodeSSHSession).Methods("POST", "OPTIONS")

	// DVE Supervisor Agent (KNIRVAGENT) status and session endpoints
	if h.knirvagentManager != nil {
		dveAliasRouter.HandleFunc("/{nodeId}/supervisor-agent/status", h.GetSupervisorAgentStatus).Methods("GET", "OPTIONS")
		dveAliasRouter.HandleFunc("/{nodeId}/supervisor-agent/session", h.GetSupervisorAgentSession).Methods("GET", "OPTIONS")
	}

	// Workspace FS API routes (Phase 10 — OverlayFS file explorer)
	if h.workspaceFSHandlers != nil {
		wsFSRouter := r.PathPrefix("/api/dve/{id}/workspace").Subrouter()
		wsFSRouter.HandleFunc("/ls", h.workspaceFSHandlers.ListDir).Methods("GET", "OPTIONS")
		wsFSRouter.HandleFunc("/cat", h.workspaceFSHandlers.ReadFile).Methods("GET", "OPTIONS")
		wsFSRouter.HandleFunc("/download", h.workspaceFSHandlers.DownloadFile).Methods("GET", "OPTIONS")

		if authMiddleware != nil {
			protectedWS := wsFSRouter.PathPrefix("").Subrouter()
			protectedWS.Use(authMiddleware.RequireAuth)
			protectedWS.HandleFunc("/upload", h.workspaceFSHandlers.UploadFile).Methods("POST", "OPTIONS")
			protectedWS.HandleFunc("/rm", h.workspaceFSHandlers.DeleteFile).Methods("DELETE", "OPTIONS")
		}
	}
}
