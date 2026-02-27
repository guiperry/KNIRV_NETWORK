package web

import (
	"encoding/json"
	"net/http"
	"time"

	"backend_server/internal/objects"
	"backend_server/internal/services/dvecreation"
	"backend_server/internal/web/middleware"

	"github.com/gorilla/mux"
)

type DVECreationHandlers struct {
	creationService *dvecreation.DVECreationService
}

func NewDVECreationHandlers(service *dvecreation.DVECreationService) *DVECreationHandlers {
	return &DVECreationHandlers{
		creationService: service,
	}
}

type DVECreationResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Message   string      `json:"message,omitempty"`
	Timestamp string      `json:"timestamp"`
}

func (h *DVECreationHandlers) CreateDVE(w http.ResponseWriter, r *http.Request) {
	var req objects.DVECreationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Extract user ID from JWT
	userID := middleware.GetUserIDFromRequest(r)
	if userID != "" {
		req.OwnerID = userID
	}

	resp, err := h.creationService.CreateDVENode(&req)
	if err != nil {
		h.sendError(w, "Failed to create DVE: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if !resp.Success {
		h.sendError(w, resp.Error, http.StatusBadRequest)
		return
	}

	h.sendJSON(w, resp, "DVE node created successfully", http.StatusCreated)
}

func (h *DVECreationHandlers) GetUserCreations(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserIDFromRequest(r)
	if userID == "" {
		h.sendError(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	creations, err := h.creationService.GetUserDVECreations(userID)
	if err != nil {
		h.sendError(w, "Failed to fetch creations: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.sendJSON(w, creations, "User DVE creations retrieved successfully", http.StatusOK)
}

func (h *DVECreationHandlers) GetCreation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	creationID := vars["id"]

	creation, err := h.creationService.GetDVECreation(creationID)
	if err != nil {
		h.sendError(w, "Creation not found", http.StatusNotFound)
		return
	}

	h.sendJSON(w, creation, "DVE creation retrieved successfully", http.StatusOK)
}

func (h *DVECreationHandlers) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.creationService.GetStats()
	if err != nil {
		h.sendError(w, "Failed to fetch stats: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.sendJSON(w, stats, "DVE creation statistics retrieved successfully", http.StatusOK)
}

func (h *DVECreationHandlers) sendError(w http.ResponseWriter, message string, code int) {
	response := DVECreationResponse{
		Success:   false,
		Error:     message,
		Timestamp: time.Now().Format(time.RFC3339),
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(response)
}

func (h *DVECreationHandlers) sendJSON(w http.ResponseWriter, data interface{}, message string, code int) {
	response := DVECreationResponse{
		Success:   true,
		Data:      data,
		Message:   message,
		Timestamp: time.Now().Format(time.RFC3339),
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(response)
}

func (h *DVECreationHandlers) RegisterRoutes(r *mux.Router, authMiddleware *middleware.AuthMiddleware) {
	apiRouter := r.PathPrefix("/api/dve-creation").Subrouter()

	if authMiddleware != nil {
		protectedRouter := apiRouter.PathPrefix("").Subrouter()
		protectedRouter.Use(authMiddleware.RequireAuth)
		protectedRouter.HandleFunc("/nodes", h.CreateDVE).Methods("POST", "OPTIONS")
		protectedRouter.HandleFunc("/nodes", h.GetUserCreations).Methods("GET", "OPTIONS")
		protectedRouter.HandleFunc("/nodes/{id}", h.GetCreation).Methods("GET", "OPTIONS")
		protectedRouter.HandleFunc("/stats", h.GetStats).Methods("GET", "OPTIONS")
	} else {
		apiRouter.HandleFunc("/nodes", h.CreateDVE).Methods("POST", "OPTIONS")
		apiRouter.HandleFunc("/nodes", h.GetUserCreations).Methods("GET", "OPTIONS")
		apiRouter.HandleFunc("/nodes/{id}", h.GetCreation).Methods("GET", "OPTIONS")
		apiRouter.HandleFunc("/stats", h.GetStats).Methods("GET", "OPTIONS")
	}
}
