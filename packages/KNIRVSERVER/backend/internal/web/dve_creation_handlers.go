package web

import (
	"encoding/json"
	"net/http"
	"time"

	"backend_server/internal/database"
	"backend_server/internal/objects"
	"backend_server/internal/services/container"
	"backend_server/internal/services/dvemanager"
	"backend_server/internal/services/endpoints"
	"backend_server/internal/services/session"
	"backend_server/internal/web/middleware"

	"github.com/gorilla/mux"
)

// DVECreationHandlers exposes the DVE lifecycle (create, session, SSH) endpoints.
// All DVE creation operations are delegated to dveManager which owns the merged
// DVECreationService.
type DVECreationHandlers struct {
	dveManager            *dvemanager.DVEManager
	containerOrchestrator *container.ContainerOrchestrator
	sessionManager        *session.SessionManager
	endpointRegistry      *endpoints.EndpointRegistry
	db                    *database.BuntDBManager
}

func NewDVECreationHandlers(
	dveManager *dvemanager.DVEManager,
	containerOrchestrator *container.ContainerOrchestrator,
	sessionManager *session.SessionManager,
	endpointRegistry *endpoints.EndpointRegistry,
	db *database.BuntDBManager,
) *DVECreationHandlers {
	return &DVECreationHandlers{
		dveManager:            dveManager,
		containerOrchestrator: containerOrchestrator,
		sessionManager:        sessionManager,
		endpointRegistry:      endpointRegistry,
		db:                    db,
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

	resp, err := h.dveManager.CreateDVENode(&req)
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

	creations, err := h.dveManager.GetUserDVECreations(userID)
	if err != nil {
		h.sendError(w, "Failed to fetch creations: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Ensure a nil slice serialises as [] not null so the frontend
	// Array.isArray() check succeeds when the user has no creations yet.
	if creations == nil {
		creations = make([]*objects.DVECreation, 0)
	}

	h.sendJSON(w, creations, "User DVE creations retrieved successfully", http.StatusOK)
}

func (h *DVECreationHandlers) GetCreation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	creationID := vars["id"]

	creation, err := h.dveManager.GetDVECreation(creationID)
	if err != nil {
		h.sendError(w, "Creation not found", http.StatusNotFound)
		return
	}

	h.sendJSON(w, creation, "DVE creation retrieved successfully", http.StatusOK)
}

func (h *DVECreationHandlers) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.dveManager.GetCreationStats()
	if err != nil {
		h.sendError(w, "Failed to fetch stats: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.sendJSON(w, stats, "DVE creation statistics retrieved successfully", http.StatusOK)
}

func (h *DVECreationHandlers) CreateSSHSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	creationID := vars["id"]

	creation, err := h.dveManager.GetDVECreation(creationID)
	if err != nil {
		h.sendError(w, "Creation not found", http.StatusNotFound)
		return
	}

	// Use actual SSH info from the creation object
	username := "dve-admin"
	if creation.SSHPrivateKey == "" && h.containerOrchestrator != nil {
		// Try to recover private key from orchestrator if not in creation object
		key, err := h.containerOrchestrator.GetSSHPrivateKey(creation.DVENodeID)
		if err == nil {
			creation.SSHPrivateKey = key
		}
	}

	if creation.SSHPrivateKey == "" {
		h.sendError(w, "SSH private key not available for this DVE", http.StatusInternalServerError)
		return
	}

	session, err := h.sessionManager.CreateSSHSession(
		creation.ID,
		creation.DVENodeID,
		username,
		creation.SSHPrivateKey,
	)

	if err != nil {
		h.sendError(w, "Failed to create SSH session: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Populate endpoint details from creation object
	session.Endpoint = creation.IPAddress
	if session.Endpoint == "" {
		session.Endpoint = "localhost"
	}
	session.Port = creation.SSHPort

	h.sendJSON(w, session, "SSH session created successfully", http.StatusCreated)
}

func (h *DVECreationHandlers) CreateValidationSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	creationID := vars["id"]

	creation, err := h.dveManager.GetDVECreation(creationID)
	if err != nil {
		h.sendError(w, "Creation not found", http.StatusNotFound)
		return
	}

	session, err := h.sessionManager.CreateValidationSession(creation.ID, "reasoning")
	if err != nil {
		h.sendError(w, "Failed to create validation session: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.sendJSON(w, session, "Validation session created successfully", http.StatusCreated)
}

func (h *DVECreationHandlers) CreateErrorResolutionSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	creationID := vars["id"]

	creation, err := h.dveManager.GetDVECreation(creationID)
	if err != nil {
		h.sendError(w, "Creation not found", http.StatusNotFound)
		return
	}

	session, err := h.sessionManager.CreateErrorResolutionSession(creation.ID, []string{"logic", "fact-check", "resource"})
	if err != nil {
		h.sendError(w, "Failed to create error resolution session: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.sendJSON(w, session, "Error resolution session created successfully", http.StatusCreated)
}

func (h *DVECreationHandlers) GetFullAccessInfo(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	creationID := vars["id"]

	creation, err := h.dveManager.GetDVECreation(creationID)
	if err != nil {
		h.sendError(w, "Creation not found", http.StatusNotFound)
		return
	}

	// Aggregate access info
	accessInfo := objects.DVEAccessInfo{
		ContainerInfo: &objects.ContainerInfo{
			ContainerID: creation.DVENodeID,
			Status:      creation.Status,
			AllocatedResources: objects.AllocatedResources{
				CPUCores: int(creation.ResourceLimits.MaxCPU),
				MemoryGB: int(creation.ResourceLimits.MaxMemory / (1024 * 1024 * 1024)),
				DiskGB:   int(creation.ResourceLimits.MaxDisk / (1024 * 1024 * 1024)),
			},
		},
	}

	h.sendJSON(w, accessInfo, "Access info retrieved successfully", http.StatusOK)
}

func (h *DVECreationHandlers) GetSSHSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	creationID := vars["id"]

	// Get all sessions for this creation and find SSH session
	sessions, err := h.sessionManager.GetSessionsByCreationID(creationID)
	if err != nil {
		h.sendError(w, "SSH session not found: "+err.Error(), http.StatusNotFound)
		return
	}
	for _, s := range sessions {
		if sshSess, ok := s.(*objects.SSHSession); ok {
			h.sendJSON(w, sshSess, "SSH session retrieved successfully", http.StatusOK)
			return
		}
	}

	h.sendError(w, "SSH session not found", http.StatusNotFound)
}

func (h *DVECreationHandlers) GetValidationSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	creationID := vars["id"]

	sessions, _ := h.sessionManager.GetSessionsByCreationID(creationID)
	for _, s := range sessions {
		if valSess, ok := s.(*objects.ValidationSession); ok {
			h.sendJSON(w, valSess, "Validation session retrieved successfully", http.StatusOK)
			return
		}
	}

	h.sendError(w, "Validation session not found", http.StatusNotFound)
}

func (h *DVECreationHandlers) GetErrorResolutionSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	creationID := vars["id"]

	sessions, _ := h.sessionManager.GetSessionsByCreationID(creationID)
	for _, s := range sessions {
		if errSess, ok := s.(*objects.ErrorResolutionSession); ok {
			h.sendJSON(w, errSess, "Error resolution session retrieved successfully", http.StatusOK)
			return
		}
	}

	h.sendError(w, "Error resolution session not found", http.StatusNotFound)
}

func (h *DVECreationHandlers) TerminateSSHSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	creationID := vars["id"]

	// Get all sessions for this creation and find SSH session
	sessions, err := h.sessionManager.GetSessionsByCreationID(creationID)
	if err == nil {
		for _, s := range sessions {
			if sshSess, ok := s.(*objects.SSHSession); ok {
				h.sessionManager.TerminateSSHSession(sshSess.ID)
				break
			}
		}
	}

	h.sendJSON(w, nil, "SSH session terminated", http.StatusOK)
}

func (h *DVECreationHandlers) DownloadSSHPrivateKey(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["sessionId"]

	sshSession, err := h.sessionManager.GetSSHSession(sessionID)
	if err != nil || sshSession == nil {
		h.sendError(w, "Invalid or expired SSH session", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename=dve-ssh-key.pem")
	w.Header().Set("Content-Type", "application/x-pem-file")
	w.Write([]byte(sshSession.PrivateKey))
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
		protectedRouter.HandleFunc("/nodes/{id}/access", h.GetFullAccessInfo).Methods("GET", "OPTIONS")
		protectedRouter.HandleFunc("/nodes/{id}/ssh-session", h.CreateSSHSession).Methods("POST", "OPTIONS")
		protectedRouter.HandleFunc("/nodes/{id}/ssh-session", h.GetSSHSession).Methods("GET", "OPTIONS")
		protectedRouter.HandleFunc("/nodes/{id}/ssh-session", h.TerminateSSHSession).Methods("DELETE", "OPTIONS")
		protectedRouter.HandleFunc("/nodes/{id}/validation-session", h.CreateValidationSession).Methods("POST", "OPTIONS")
		protectedRouter.HandleFunc("/nodes/{id}/validation-session", h.GetValidationSession).Methods("GET", "OPTIONS")
		protectedRouter.HandleFunc("/nodes/{id}/error-resolution-session", h.CreateErrorResolutionSession).Methods("POST", "OPTIONS")
		protectedRouter.HandleFunc("/nodes/{id}/error-resolution-session", h.GetErrorResolutionSession).Methods("GET", "OPTIONS")
		protectedRouter.HandleFunc("/sessions/ssh/{sessionId}/private-key", h.DownloadSSHPrivateKey).Methods("GET", "OPTIONS")
		protectedRouter.HandleFunc("/stats", h.GetStats).Methods("GET", "OPTIONS")
	} else {
		apiRouter.HandleFunc("/nodes", h.CreateDVE).Methods("POST", "OPTIONS")
		apiRouter.HandleFunc("/nodes", h.GetUserCreations).Methods("GET", "OPTIONS")
		apiRouter.HandleFunc("/nodes/{id}", h.GetCreation).Methods("GET", "OPTIONS")
		apiRouter.HandleFunc("/nodes/{id}/access", h.GetFullAccessInfo).Methods("GET", "OPTIONS")
		apiRouter.HandleFunc("/nodes/{id}/ssh-session", h.CreateSSHSession).Methods("POST", "OPTIONS")
		apiRouter.HandleFunc("/nodes/{id}/ssh-session", h.GetSSHSession).Methods("GET", "OPTIONS")
		apiRouter.HandleFunc("/nodes/{id}/ssh-session", h.TerminateSSHSession).Methods("DELETE", "OPTIONS")
		apiRouter.HandleFunc("/nodes/{id}/validation-session", h.CreateValidationSession).Methods("POST", "OPTIONS")
		apiRouter.HandleFunc("/nodes/{id}/validation-session", h.GetValidationSession).Methods("GET", "OPTIONS")
		apiRouter.HandleFunc("/nodes/{id}/error-resolution-session", h.CreateErrorResolutionSession).Methods("POST", "OPTIONS")
		apiRouter.HandleFunc("/nodes/{id}/error-resolution-session", h.GetErrorResolutionSession).Methods("GET", "OPTIONS")
		apiRouter.HandleFunc("/sessions/ssh/{sessionId}/private-key", h.DownloadSSHPrivateKey).Methods("GET", "OPTIONS")
		apiRouter.HandleFunc("/stats", h.GetStats).Methods("GET", "OPTIONS")
	}
}
