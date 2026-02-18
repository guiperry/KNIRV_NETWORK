package web

import (
	"encoding/json"
	"net/http"

	"backend_server/internal/services/active_memory"
	"backend_server/internal/web/middleware"

	"github.com/gorilla/mux"
)

// ActiveMemoryHandlers handles web requests for the Active Memory Layer
type ActiveMemoryHandlers struct {
	service *active_memory.ActiveMemoryService
}

// NewActiveMemoryHandlers creates a new instance of ActiveMemoryHandlers
func NewActiveMemoryHandlers(service *active_memory.ActiveMemoryService) *ActiveMemoryHandlers {
	return &ActiveMemoryHandlers{
		service: service,
	}
}

// RegisterRoutes registers the active memory routes
func (h *ActiveMemoryHandlers) RegisterRoutes(r *mux.Router, authMiddleware *middleware.AuthMiddleware) {
	memoryRouter := r.PathPrefix("/api/memory").Subrouter()

	// MCP-like tool endpoints
	memoryRouter.HandleFunc("/mcp/store", h.StoreInteraction).Methods("POST", "OPTIONS")
	memoryRouter.HandleFunc("/mcp/execute/{id}", h.ExecuteSolution).Methods("POST", "OPTIONS")
	
	// Add more routes as needed
}

// ExecuteSolution handles executing a solution logic stored in the vault
func (h *ActiveMemoryHandlers) ExecuteSolution(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var req struct {
		Params map[string]interface{} `json:"params"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// allow empty body
	}

	output, err := h.service.ExecuteSolution(r.Context(), id, req.Params)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
			"output":  output,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"output":  output,
	})
}

// StoreInteraction handles storing an agent interaction (MCP-like)
func (h *ActiveMemoryHandlers) StoreInteraction(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AgentID      string `json:"agent_id"`
		ErrorDesc    string `json:"error_description"`
		SolutionCode string `json:"solution_code"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.service.RecordInteraction(r.Context(), req.AgentID, req.ErrorDesc, req.SolutionCode); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Interaction recorded in encrypted Markdown fabric",
	})
}
