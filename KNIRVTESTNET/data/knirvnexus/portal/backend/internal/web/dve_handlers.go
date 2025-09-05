package web

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"nexus-backend/internal/services/dvemanager"
	"nexus-backend/internal/web/middleware"

	"github.com/gorilla/mux"
)

type DVEHandlers struct {
	dveManager *dvemanager.DVEManager
}

func NewDVEHandlers(dveManager *dvemanager.DVEManager) *DVEHandlers {
	return &DVEHandlers{dveManager: dveManager}
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

// RegisterRoutes registers the DVE routes with the router
func (h *DVEHandlers) RegisterRoutes(r *mux.Router, authMiddleware *middleware.AuthMiddleware) {
	// Create a subrouter for DVE endpoints
	dveRouter := r.PathPrefix("/api/dve-nodes").Subrouter()

	// Public routes for monitoring
	dveRouter.HandleFunc("", h.GetDVENodes).Methods("GET")
	dveRouter.HandleFunc("/{id}", h.GetDVENode).Methods("GET")

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
