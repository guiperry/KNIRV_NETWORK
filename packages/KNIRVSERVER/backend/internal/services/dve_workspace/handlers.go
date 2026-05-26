package dve_workspace

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/gorilla/mux"

	"backend_server/internal/web/middleware"
)

// CreateDVEEnvironmentRequest is the request body for creating a DVE environment.
type CreateDVEEnvironmentRequest struct {
	Name   string                 `json:"name"`
	Type   EnvironmentType        `json:"type"`
	Config map[string]interface{} `json:"config"`
}

// HandleListEnvironments handles the list environments endpoint
func (s *DVEService) HandleListEnvironments(w http.ResponseWriter, r *http.Request) {
	authCtx := middleware.GetAuthContext(r)
	if authCtx == nil {
		writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}
	environments := s.ListUserEnvironments(authCtx.UserID)
	writeJSON(w, http.StatusOK, environments)
}

// HandleCreateEnvironment handles the create environment endpoint
func (s *DVEService) HandleCreateEnvironment(w http.ResponseWriter, r *http.Request) {
	authCtx := middleware.GetAuthContext(r)
	if authCtx == nil {
		writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	var req CreateDVEEnvironmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "Environment name is required")
		return
	}

	environment, err := s.CreateEnvironment(authCtx.UserID, req.Name, req.Type, req.Config)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, environment)
}

// HandleGetEnvironment handles the get environment endpoint
func (s *DVEService) HandleGetEnvironment(w http.ResponseWriter, r *http.Request) {
	authCtx := middleware.GetAuthContext(r)
	if authCtx == nil {
		writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	vars := mux.Vars(r)
	envID := vars["id"]

	environment, err := s.GetEnvironment(envID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Environment not found")
		return
	}

	if environment.UserID != authCtx.UserID {
		writeError(w, http.StatusForbidden, "Access denied")
		return
	}

	writeJSON(w, http.StatusOK, environment)
}

// HandleDeleteEnvironment handles the delete environment endpoint
func (s *DVEService) HandleDeleteEnvironment(w http.ResponseWriter, r *http.Request) {
	authCtx := middleware.GetAuthContext(r)
	if authCtx == nil {
		writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	vars := mux.Vars(r)
	envID := vars["id"]

	environment, err := s.GetEnvironment(envID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Environment not found")
		return
	}

	if environment.UserID != authCtx.UserID {
		writeError(w, http.StatusForbidden, "Access denied")
		return
	}

	if err := s.StopEnvironment(envID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Environment deleted successfully"})
}

// HandleStartEnvironment handles the start environment endpoint
func (s *DVEService) HandleStartEnvironment(w http.ResponseWriter, r *http.Request) {
	authCtx := middleware.GetAuthContext(r)
	if authCtx == nil {
		writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	vars := mux.Vars(r)
	envID := vars["id"]

	environment, err := s.GetEnvironment(envID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Environment not found")
		return
	}

	if environment.UserID != authCtx.UserID {
		writeError(w, http.StatusForbidden, "Access denied")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Environment start initiated"})
}

// HandleStopEnvironment handles the stop environment endpoint
func (s *DVEService) HandleStopEnvironment(w http.ResponseWriter, r *http.Request) {
	authCtx := middleware.GetAuthContext(r)
	if authCtx == nil {
		writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	vars := mux.Vars(r)
	envID := vars["id"]

	environment, err := s.GetEnvironment(envID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Environment not found")
		return
	}

	if environment.UserID != authCtx.UserID {
		writeError(w, http.StatusForbidden, "Access denied")
		return
	}

	if err := s.StopEnvironment(envID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Environment stopped successfully"})
}

// DVE Session Handlers

// CreateDVESessionRequest is the request body for creating a DVE session.
type CreateDVESessionRequest struct {
	EnvironmentID  string `json:"environment_id"`
	ConnectionType string `json:"connection_type"`
}

// HandleListSessions handles the list sessions endpoint
func (s *DVEService) HandleListSessions(w http.ResponseWriter, r *http.Request) {
	authCtx := middleware.GetAuthContext(r)
	if authCtx == nil {
		writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	sessions := s.ListUserSessions(authCtx.UserID)
	writeJSON(w, http.StatusOK, sessions)
}

// HandleCreateSession handles the create session endpoint
func (s *DVEService) HandleCreateSession(w http.ResponseWriter, r *http.Request) {
	authCtx := middleware.GetAuthContext(r)
	if authCtx == nil {
		writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	var req CreateDVESessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.EnvironmentID == "" {
		writeError(w, http.StatusBadRequest, "Environment ID is required")
		return
	}

	if req.ConnectionType == "" {
		req.ConnectionType = "websocket"
	}

	session, err := s.CreateSession(authCtx.UserID, req.EnvironmentID, req.ConnectionType)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, session)
}

// DVE Project Handlers

// CreateDVEProjectRequest is the request body for creating a DVE project.
type CreateDVEProjectRequest struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Type        ProjectType `json:"type"`
	Language    string      `json:"language"`
}

// HandleListProjects handles the list projects endpoint
func (s *DVEService) HandleListProjects(w http.ResponseWriter, r *http.Request) {
	authCtx := middleware.GetAuthContext(r)
	if authCtx == nil {
		writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	projects := s.ListUserProjects(authCtx.UserID)
	writeJSON(w, http.StatusOK, projects)
}

// HandleCreateProject handles the create project endpoint
func (s *DVEService) HandleCreateProject(w http.ResponseWriter, r *http.Request) {
	authCtx := middleware.GetAuthContext(r)
	if authCtx == nil {
		writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	var req CreateDVEProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "Project name is required")
		return
	}

	project, err := s.CreateProject(authCtx.UserID, req.Name, req.Description, req.Type, req.Language)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, project)
}

// DVE Workspace Config Handlers

// HandleGetWorkspaceConfig returns the current DVE workspace configuration.
func (s *DVEService) HandleGetWorkspaceConfig(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	cfg := s.config
	s.mu.RUnlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"config": cfg,
	})
}

// HandleUpdateWorkspaceConfig hot-updates the DVE workspace configuration.
func (s *DVEService) HandleUpdateWorkspaceConfig(w http.ResponseWriter, r *http.Request) {
	var cfg DVEConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	s.mu.Lock()
	s.config = cfg
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]string{"message": "Configuration updated (affects new workspaces)"})
}

// HandleGetRootfsStatus checks if the BusyBox rootfs is bootstrapped.
func (s *DVEService) HandleGetRootfsStatus(w http.ResponseWriter, r *http.Request) {
	ready := false
	if s.config.BusyBoxRootfsPath != "" {
		marker := s.config.BusyBoxRootfsPath + "/.knirvdve-ready"
		if _, err := os.Stat(marker); err == nil {
			ready = true
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ready": ready,
		"path":  s.config.BusyBoxRootfsPath,
	})
}

// HandleBootstrapRootfs triggers one-time BusyBox rootfs setup.
func (s *DVEService) HandleBootstrapRootfs(w http.ResponseWriter, r *http.Request) {
	if err := EnsureBusyBoxRootfs(s.config.BusyBoxRootfsPath, s.config); err != nil {
		writeError(w, http.StatusInternalServerError, "Rootfs bootstrap failed: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "BusyBox rootfs bootstrapped successfully"})
}

// HandleGetWorkspaceStats returns active workspace statistics.
func (s *DVEService) HandleGetWorkspaceStats(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	numWorkspaces := len(s.namespaces)
	numEnvs := len(s.environments)
	s.mu.RUnlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"active_overlay_workspaces": numWorkspaces,
		"total_environments":        numEnvs,
		"config":                    s.config,
	})
}

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
