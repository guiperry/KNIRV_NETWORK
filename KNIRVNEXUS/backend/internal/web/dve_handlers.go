package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"backend_server/internal/services/dvemanager"
	"backend_server/internal/services/dverental"
	"backend_server/internal/web/middleware"

	"github.com/gorilla/mux"
)

type DVEHandlers struct {
	dveManager       *dvemanager.DVEManager
	dveRentalService *dverental.DVERentalService
}

func NewDVEHandlers(dveManager *dvemanager.DVEManager, dveRentalService *dverental.DVERentalService) *DVEHandlers {
	return &DVEHandlers{
		dveManager:       dveManager,
		dveRentalService: dveRentalService,
	}
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

	// Apply role-based filtering for observer (Developer) role
	authCtx := middleware.GetAuthContext(r)
	if authCtx != nil && authCtx.Role == "observer" && h.dveRentalService != nil {
		// Get the list of node IDs that this user has rented
		rentedNodeIDs, err := h.dveRentalService.GetUserRentedNodeIDs(authCtx.UserID)
		if err != nil {
			// Log error but don't fail the request - return empty array for developers with no rentals
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

		// Create a map for quick lookup
		rentedNodeMap := make(map[string]bool)
		for _, nodeID := range rentedNodeIDs {
			rentedNodeMap[nodeID] = true
		}

		// Filter nodes to only include those the user has rented
		filteredNodes := make([]interface{}, 0)
		for _, node := range nodes {
			// Access ID field directly from the DVENode struct
			if rentedNodeMap[node.ID] {
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

	// For admin and validator roles, return all nodes
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
	var req dvemanager.RegisterNodeRequest
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

	node, err := h.dveManager.GetNode(nodeID)
	if err != nil {
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

	if node == nil {
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

	if node == nil {
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

	if node == nil {
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

	endpoint := map[string]interface{}{
		"port": node.SSHPort,
		"host": node.IPAddress,
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

	if node == nil {
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

	endpoint := map[string]interface{}{
		"port": node.ValidationPort,
		"host": node.IPAddress,
		"protocol": "http",
		"url": fmt.Sprintf("http://%s:%d", node.IPAddress, node.ValidationPort),
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

	if node == nil {
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

	endpoint := map[string]interface{}{
		"port": node.ErrorResPort,
		"host": node.IPAddress,
		"protocol": "http",
		"url": fmt.Sprintf("http://%s:%d", node.IPAddress, node.ErrorResPort),
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

// RegisterRoutes registers the DVE routes with the router
func (h *DVEHandlers) RegisterRoutes(r *mux.Router, authMiddleware *middleware.AuthMiddleware) {
	// Create a subrouter for DVE endpoints
	dveRouter := r.PathPrefix("/api/dve-nodes").Subrouter()

	// Routes with optional auth (for role-based filtering)
	if authMiddleware != nil {
		optionalAuthRouter := dveRouter.PathPrefix("").Subrouter()
		optionalAuthRouter.Use(authMiddleware.OptionalAuth)
		optionalAuthRouter.HandleFunc("", h.GetDVENodes).Methods("GET")
		optionalAuthRouter.HandleFunc("/{id}", h.GetDVENode).Methods("GET")
		optionalAuthRouter.HandleFunc("/{id}/endpoints", h.GetDVENodeEndpoints).Methods("GET")
		optionalAuthRouter.HandleFunc("/{id}/ssh-endpoint", h.GetDVENodeSSHEndpoint).Methods("GET")
		optionalAuthRouter.HandleFunc("/{id}/validation-endpoint", h.GetDVENodeValidationEndpoint).Methods("GET")
		optionalAuthRouter.HandleFunc("/{id}/error-resolution-endpoint", h.GetDVENodeErrorResolutionEndpoint).Methods("GET")
	} else {
		// If no auth middleware, allow public access
		dveRouter.HandleFunc("", h.GetDVENodes).Methods("GET")
		dveRouter.HandleFunc("/{id}", h.GetDVENode).Methods("GET")
		dveRouter.HandleFunc("/{id}/endpoints", h.GetDVENodeEndpoints).Methods("GET")
		dveRouter.HandleFunc("/{id}/ssh-endpoint", h.GetDVENodeSSHEndpoint).Methods("GET")
		dveRouter.HandleFunc("/{id}/validation-endpoint", h.GetDVENodeValidationEndpoint).Methods("GET")
		dveRouter.HandleFunc("/{id}/error-resolution-endpoint", h.GetDVENodeErrorResolutionEndpoint).Methods("GET")
	}

	// Protected routes for management
	if authMiddleware != nil {
		protectedDVERouter := dveRouter.PathPrefix("").Subrouter()
		protectedDVERouter.Use(authMiddleware.RequireAuth)
		protectedDVERouter.HandleFunc("", h.PostDVENodes).Methods("POST")
		protectedDVERouter.HandleFunc("/{id}", h.UpdateDVENode).Methods("PUT")
		protectedDVERouter.HandleFunc("/{id}", h.DeleteDVENode).Methods("DELETE")
	} else {
		// If no auth middleware, allow all routes (for testnet mode)
		dveRouter.HandleFunc("", h.PostDVENodes).Methods("POST")
		dveRouter.HandleFunc("/{id}", h.UpdateDVENode).Methods("PUT")
		dveRouter.HandleFunc("/{id}", h.DeleteDVENode).Methods("DELETE")
	}

	// Handle OPTIONS requests for CORS
	dveRouter.Methods("OPTIONS").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}
