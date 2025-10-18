package cde

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"

	"backend-server/internal/web/middleware"
)

// CDE Environment Handlers

type CreateCDEEnvironmentRequest struct {
	Name   string                 `json:"name"`
	Type   EnvironmentType        `json:"type"`
	Config map[string]interface{} `json:"config"`
}

// HandleListEnvironments handles the list environments endpoint
func (s *CDEService) HandleListEnvironments(w http.ResponseWriter, r *http.Request) {
	authCtx := middleware.GetAuthContext(r)
	if authCtx == nil {
		writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	environments := s.ListUserEnvironments(authCtx.UserID)
	writeJSON(w, http.StatusOK, environments)
}

// HandleCreateEnvironment handles the create environment endpoint
func (s *CDEService) HandleCreateEnvironment(w http.ResponseWriter, r *http.Request) {
	authCtx := middleware.GetAuthContext(r)
	if authCtx == nil {
		writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	var req CreateCDEEnvironmentRequest
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
func (s *CDEService) HandleGetEnvironment(w http.ResponseWriter, r *http.Request) {
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

	// Check ownership
	if environment.UserID != authCtx.UserID {
		writeError(w, http.StatusForbidden, "Access denied")
		return
	}

	writeJSON(w, http.StatusOK, environment)
}

// HandleDeleteEnvironment handles the delete environment endpoint
func (s *CDEService) HandleDeleteEnvironment(w http.ResponseWriter, r *http.Request) {
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

	// Check ownership
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
func (s *CDEService) HandleStartEnvironment(w http.ResponseWriter, r *http.Request) {
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

	// Check ownership
	if environment.UserID != authCtx.UserID {
		writeError(w, http.StatusForbidden, "Access denied")
		return
	}

	// Environment start logic would be implemented here
	// For now, just return success
	writeJSON(w, http.StatusOK, map[string]string{"message": "Environment start initiated"})
}

// HandleStopEnvironment handles the stop environment endpoint
func (s *CDEService) HandleStopEnvironment(w http.ResponseWriter, r *http.Request) {
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

	// Check ownership
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

// CDE Session Handlers

type CreateCDESessionRequest struct {
	EnvironmentID  string `json:"environment_id"`
	ConnectionType string `json:"connection_type"`
}

// HandleListSessions handles the list sessions endpoint
func (s *CDEService) HandleListSessions(w http.ResponseWriter, r *http.Request) {
	authCtx := middleware.GetAuthContext(r)
	if authCtx == nil {
		writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	sessions := s.ListUserSessions(authCtx.UserID)
	writeJSON(w, http.StatusOK, sessions)
}

// HandleCreateSession handles the create session endpoint
func (s *CDEService) HandleCreateSession(w http.ResponseWriter, r *http.Request) {
	authCtx := middleware.GetAuthContext(r)
	if authCtx == nil {
		writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	var req CreateCDESessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.EnvironmentID == "" {
		writeError(w, http.StatusBadRequest, "Environment ID is required")
		return
	}

	if req.ConnectionType == "" {
		req.ConnectionType = "websocket" // Default connection type
	}

	session, err := s.CreateSession(authCtx.UserID, req.EnvironmentID, req.ConnectionType)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, session)
}

// CDE Project Handlers

type CreateCDEProjectRequest struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Type        ProjectType `json:"type"`
	Language    string      `json:"language"`
}

// HandleListProjects handles the list projects endpoint
func (s *CDEService) HandleListProjects(w http.ResponseWriter, r *http.Request) {
	authCtx := middleware.GetAuthContext(r)
	if authCtx == nil {
		writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	projects := s.ListUserProjects(authCtx.UserID)
	writeJSON(w, http.StatusOK, projects)
}

// HandleCreateProject handles the create project endpoint
func (s *CDEService) HandleCreateProject(w http.ResponseWriter, r *http.Request) {
	authCtx := middleware.GetAuthContext(r)
	if authCtx == nil {
		writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	var req CreateCDEProjectRequest
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
