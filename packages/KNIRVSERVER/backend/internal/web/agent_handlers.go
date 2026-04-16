// Copyright 2026 KNIRV-SERVER
// SPDX-License-Identifier: GPL-3.0-or-later

package web

import (
	"encoding/json"
	"net/http"

	"backend_server/internal/services/agent"
	"backend_server/internal/web/middleware"

	"github.com/gorilla/mux"
)

// AgentHandlers provides HTTP handlers for the DVE Agent Command Center
type AgentHandlers struct {
	agentService *agent.AgentService
}

// NewAgentHandlers creates agent HTTP handlers
func NewAgentHandlers(agentService *agent.AgentService) *AgentHandlers {
	return &AgentHandlers{agentService: agentService}
}

type agentResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Timestamp string      `json:"timestamp"`
}

func writeAgentJSON(w http.ResponseWriter, status int, resp agentResponse) {
	resp.Timestamp = getCurrentTimestamp()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(resp)
}

// GetAgentStatus handles GET /api/dve/{id}/agent/status
func (h *AgentHandlers) GetAgentStatus(w http.ResponseWriter, r *http.Request) {
	dveID := mux.Vars(r)["id"]
	status, err := h.agentService.GetAgentStatus(dveID)
	if err != nil {
		writeAgentJSON(w, http.StatusInternalServerError, agentResponse{Error: err.Error()})
		return
	}
	writeAgentJSON(w, http.StatusOK, agentResponse{Success: true, Data: status})
}

// LaunchAgent handles POST /api/dve/{id}/agent/launch
func (h *AgentHandlers) LaunchAgent(w http.ResponseWriter, r *http.Request) {
	dveID := mux.Vars(r)["id"]
	status, err := h.agentService.LaunchAgent(r.Context(), dveID)
	if err != nil {
		writeAgentJSON(w, http.StatusInternalServerError, agentResponse{Error: err.Error()})
		return
	}
	writeAgentJSON(w, http.StatusCreated, agentResponse{Success: true, Data: status})
}

// StopAgent handles DELETE /api/dve/{id}/agent
func (h *AgentHandlers) StopAgent(w http.ResponseWriter, r *http.Request) {
	dveID := mux.Vars(r)["id"]
	if err := h.agentService.StopAgent(r.Context(), dveID); err != nil {
		writeAgentJSON(w, http.StatusInternalServerError, agentResponse{Error: err.Error()})
		return
	}
	writeAgentJSON(w, http.StatusOK, agentResponse{Success: true, Data: map[string]string{"message": "agent stopped"}})
}

// GetAgentTasks handles GET /api/dve/{id}/agent/tasks
func (h *AgentHandlers) GetAgentTasks(w http.ResponseWriter, r *http.Request) {
	dveID := mux.Vars(r)["id"]
	tasks, err := h.agentService.GetTasks(dveID)
	if err != nil {
		writeAgentJSON(w, http.StatusInternalServerError, agentResponse{Error: err.Error()})
		return
	}
	if tasks == nil {
		tasks = []*agent.AgentTask{}
	}
	writeAgentJSON(w, http.StatusOK, agentResponse{Success: true, Data: tasks})
}

// SubmitAgentTask handles POST /api/dve/{id}/agent/tasks
func (h *AgentHandlers) SubmitAgentTask(w http.ResponseWriter, r *http.Request) {
	dveID := mux.Vars(r)["id"]

	var req agent.AgentTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAgentJSON(w, http.StatusBadRequest, agentResponse{Error: "invalid request body: " + err.Error()})
		return
	}

	if req.Title == "" {
		writeAgentJSON(w, http.StatusBadRequest, agentResponse{Error: "task title is required"})
		return
	}

	task, err := h.agentService.SubmitTask(dveID, &req)
	if err != nil {
		writeAgentJSON(w, http.StatusInternalServerError, agentResponse{Error: err.Error()})
		return
	}
	writeAgentJSON(w, http.StatusCreated, agentResponse{Success: true, Data: task})
}

// ConnectAgent handles POST /api/v1/dve/agents/connect
// Allows external agents to connect to a DVE and utilize its resources
func (h *AgentHandlers) ConnectAgent(w http.ResponseWriter, r *http.Request) {
	var connectReq struct {
		AgentID      string                 `json:"agent_id"`
		AgentName    string                 `json:"agent_name"`
		Capabilities []string               `json:"capabilities"`
		Endpoint     string                 `json:"endpoint"` // Agent's endpoint for communication
		Metadata     map[string]interface{} `json:"metadata"`
	}

	if err := json.NewDecoder(r.Body).Decode(&connectReq); err != nil {
		writeAgentJSON(w, http.StatusBadRequest, agentResponse{Error: "invalid request body: " + err.Error()})
		return
	}

	if connectReq.AgentID == "" {
		writeAgentJSON(w, http.StatusBadRequest, agentResponse{Error: "agent_id is required"})
		return
	}

	// Register external agent connection
	agentConn := &agent.AgentConnection{
		AgentID:      connectReq.AgentID,
		AgentName:    connectReq.AgentName,
		Endpoint:     connectReq.Endpoint,
		Capabilities: connectReq.Capabilities,
		Metadata:     connectReq.Metadata,
	}

	status, err := h.agentService.ConnectAgent(r.Context(), agentConn)
	if err != nil {
		writeAgentJSON(w, http.StatusInternalServerError, agentResponse{Error: err.Error()})
		return
	}

	writeAgentJSON(w, http.StatusOK, agentResponse{Success: true, Data: map[string]interface{}{
		"message": "agent connected",
		"status":  status,
	}})
}

// RegisterRoutes registers all agent routes with the router
func (h *AgentHandlers) RegisterRoutes(r *mux.Router, authMiddleware *middleware.AuthMiddleware) {
	apiRouter := r.PathPrefix("/api/dve/{id}/agent").Subrouter()
	if authMiddleware != nil {
		apiRouter.Use(authMiddleware.OptionalAuth)
	}
	apiRouter.HandleFunc("/status", h.GetAgentStatus).Methods("GET", "OPTIONS")
	apiRouter.HandleFunc("/launch", h.LaunchAgent).Methods("POST", "OPTIONS")
	apiRouter.HandleFunc("", h.StopAgent).Methods("DELETE", "OPTIONS")
	apiRouter.HandleFunc("/tasks", h.GetAgentTasks).Methods("GET", "OPTIONS")
	apiRouter.HandleFunc("/tasks", h.SubmitAgentTask).Methods("POST", "OPTIONS")
	apiRouter.HandleFunc("/tasks/{taskID}", h.GetAgentTask).Methods("GET", "OPTIONS")

	// External agent connection endpoint
	agentsRouter := r.PathPrefix("/api/v1/dve/agents").Subrouter()
	if authMiddleware != nil {
		agentsRouter.Use(authMiddleware.OptionalAuth)
	}
	agentsRouter.HandleFunc("/connect", h.ConnectAgent).Methods("POST", "OPTIONS")
}

// GetAgentTask handles GET /api/dve/{id}/agent/tasks/{taskID}
func (h *AgentHandlers) GetAgentTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["taskID"]

	task, err := h.agentService.GetTask(taskID)
	if err != nil {
		writeAgentJSON(w, http.StatusNotFound, agentResponse{Error: err.Error()})
		return
	}
	writeAgentJSON(w, http.StatusOK, agentResponse{Success: true, Data: task})
}
