package api

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"

	"nexus-backend/internal/services/cde"
)

// CDE Environment Handlers

type CreateCDEEnvironmentRequest struct {
	Name   string                 `json:"name"`
	Type   cde.EnvironmentType    `json:"type"`
	Config map[string]interface{} `json:"config"`
}

func (s *APIServer) handleListCDEEnvironments(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r)
	if authCtx == nil {
		s.writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	if s.cdeService == nil {
		s.writeError(w, http.StatusServiceUnavailable, "CDE service not available")
		return
	}

	environments := s.cdeService.ListUserEnvironments(authCtx.UserID)
	s.writeJSON(w, http.StatusOK, environments)
}

func (s *APIServer) handleCreateCDEEnvironment(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r)
	if authCtx == nil {
		s.writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	if s.cdeService == nil {
		s.writeError(w, http.StatusServiceUnavailable, "CDE service not available")
		return
	}

	var req CreateCDEEnvironmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" {
		s.writeError(w, http.StatusBadRequest, "Environment name is required")
		return
	}

	environment, err := s.cdeService.CreateEnvironment(authCtx.UserID, req.Name, req.Type, req.Config)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusCreated, environment)
}

func (s *APIServer) handleGetCDEEnvironment(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r)
	if authCtx == nil {
		s.writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	if s.cdeService == nil {
		s.writeError(w, http.StatusServiceUnavailable, "CDE service not available")
		return
	}

	vars := mux.Vars(r)
	envID := vars["id"]

	environment, err := s.cdeService.GetEnvironment(envID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "Environment not found")
		return
	}

	// Check ownership
	if environment.UserID != authCtx.UserID {
		s.writeError(w, http.StatusForbidden, "Access denied")
		return
	}

	s.writeJSON(w, http.StatusOK, environment)
}

func (s *APIServer) handleDeleteCDEEnvironment(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r)
	if authCtx == nil {
		s.writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	if s.cdeService == nil {
		s.writeError(w, http.StatusServiceUnavailable, "CDE service not available")
		return
	}

	vars := mux.Vars(r)
	envID := vars["id"]

	environment, err := s.cdeService.GetEnvironment(envID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "Environment not found")
		return
	}

	// Check ownership
	if environment.UserID != authCtx.UserID {
		s.writeError(w, http.StatusForbidden, "Access denied")
		return
	}

	if err := s.cdeService.StopEnvironment(envID); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]string{"message": "Environment deleted successfully"})
}

func (s *APIServer) handleStartCDEEnvironment(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r)
	if authCtx == nil {
		s.writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	if s.cdeService == nil {
		s.writeError(w, http.StatusServiceUnavailable, "CDE service not available")
		return
	}

	vars := mux.Vars(r)
	envID := vars["id"]

	environment, err := s.cdeService.GetEnvironment(envID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "Environment not found")
		return
	}

	// Check ownership
	if environment.UserID != authCtx.UserID {
		s.writeError(w, http.StatusForbidden, "Access denied")
		return
	}

	// Environment start logic would be implemented here
	// For now, just return success
	s.writeJSON(w, http.StatusOK, map[string]string{"message": "Environment start initiated"})
}

func (s *APIServer) handleStopCDEEnvironment(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r)
	if authCtx == nil {
		s.writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	if s.cdeService == nil {
		s.writeError(w, http.StatusServiceUnavailable, "CDE service not available")
		return
	}

	vars := mux.Vars(r)
	envID := vars["id"]

	environment, err := s.cdeService.GetEnvironment(envID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "Environment not found")
		return
	}

	// Check ownership
	if environment.UserID != authCtx.UserID {
		s.writeError(w, http.StatusForbidden, "Access denied")
		return
	}

	if err := s.cdeService.StopEnvironment(envID); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]string{"message": "Environment stopped successfully"})
}

// CDE Session Handlers

type CreateCDESessionRequest struct {
	EnvironmentID  string `json:"environment_id"`
	ConnectionType string `json:"connection_type"`
}

func (s *APIServer) handleListCDESessions(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r)
	if authCtx == nil {
		s.writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	if s.cdeService == nil {
		s.writeError(w, http.StatusServiceUnavailable, "CDE service not available")
		return
	}

	sessions := s.cdeService.ListUserSessions(authCtx.UserID)
	s.writeJSON(w, http.StatusOK, sessions)
}

func (s *APIServer) handleCreateCDESession(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r)
	if authCtx == nil {
		s.writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	if s.cdeService == nil {
		s.writeError(w, http.StatusServiceUnavailable, "CDE service not available")
		return
	}

	var req CreateCDESessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.EnvironmentID == "" {
		s.writeError(w, http.StatusBadRequest, "Environment ID is required")
		return
	}

	if req.ConnectionType == "" {
		req.ConnectionType = "websocket" // Default connection type
	}

	session, err := s.cdeService.CreateSession(authCtx.UserID, req.EnvironmentID, req.ConnectionType)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusCreated, session)
}

// CDE Project Handlers

type CreateCDEProjectRequest struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Type        cde.ProjectType `json:"type"`
	Language    string          `json:"language"`
}

func (s *APIServer) handleListCDEProjects(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r)
	if authCtx == nil {
		s.writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	if s.cdeService == nil {
		s.writeError(w, http.StatusServiceUnavailable, "CDE service not available")
		return
	}

	projects := s.cdeService.ListUserProjects(authCtx.UserID)
	s.writeJSON(w, http.StatusOK, projects)
}

func (s *APIServer) handleCreateCDEProject(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r)
	if authCtx == nil {
		s.writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	if s.cdeService == nil {
		s.writeError(w, http.StatusServiceUnavailable, "CDE service not available")
		return
	}

	var req CreateCDEProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" {
		s.writeError(w, http.StatusBadRequest, "Project name is required")
		return
	}

	project, err := s.cdeService.CreateProject(authCtx.UserID, req.Name, req.Description, req.Type, req.Language)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusCreated, project)
}
