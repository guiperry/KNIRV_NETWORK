package web

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"nexus-backend/internal/models"
	"nexus-backend/internal/services/agentmanagement"
	"nexus-backend/internal/web/middleware"

	"github.com/gorilla/mux"
)

type AgentManagementHandlers struct {
	agentManagementService *agentmanagement.AgentManagementService
}

func NewAgentManagementHandlers(agentManagementService *agentmanagement.AgentManagementService) *AgentManagementHandlers {
	return &AgentManagementHandlers{agentManagementService: agentManagementService}
}

type AgentManagementResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Message   string      `json:"message,omitempty"`
	Error     string      `json:"error,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// GetAgents handles GET /api/agent-management/agents
func (h *AgentManagementHandlers) GetAgents(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters for filtering
	filter := &models.AgentFilter{}

	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = []string{status}
	}
	if agentType := r.URL.Query().Get("type"); agentType != "" {
		filter.Type = []string{agentType}
	}
	if author := r.URL.Query().Get("author"); author != "" {
		filter.Author = author
	}
	if search := r.URL.Query().Get("search"); search != "" {
		filter.Search = search
	}
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			filter.Limit = limit
		}
	}
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			filter.Offset = offset
		}
	}

	agents, err := h.agentManagementService.GetAllAgents(filter)
	if err != nil {
		response := AgentManagementResponse{
			Success:   false,
			Error:     "Failed to retrieve agents: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := AgentManagementResponse{
		Success:   true,
		Data:      agents,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetAgent handles GET /api/agent-management/agents/{id}
func (h *AgentManagementHandlers) GetAgent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["id"]

	if agentID == "" {
		response := AgentManagementResponse{
			Success:   false,
			Error:     "Agent ID is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	agent, err := h.agentManagementService.GetAgent(agentID)
	if err != nil {
		response := AgentManagementResponse{
			Success:   false,
			Error:     "Failed to retrieve agent: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := AgentManagementResponse{
		Success:   true,
		Data:      agent,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// PostAgent handles POST /api/agent-management/agents
func (h *AgentManagementHandlers) PostAgent(w http.ResponseWriter, r *http.Request) {
	var agent models.Agent
	if err := json.NewDecoder(r.Body).Decode(&agent); err != nil {
		response := AgentManagementResponse{
			Success:   false,
			Error:     "Invalid request body",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	if err := h.agentManagementService.CreateAgent(&agent); err != nil {
		response := AgentManagementResponse{
			Success:   false,
			Error:     "Failed to create agent: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := AgentManagementResponse{
		Success:   true,
		Data:      &agent,
		Message:   "Agent created successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// PutAgent handles PUT /api/agent-management/agents/{id}
func (h *AgentManagementHandlers) PutAgent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["id"]

	if agentID == "" {
		response := AgentManagementResponse{
			Success:   false,
			Error:     "Agent ID is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	var updates models.Agent
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		response := AgentManagementResponse{
			Success:   false,
			Error:     "Invalid request body",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	if err := h.agentManagementService.UpdateAgent(agentID, &updates); err != nil {
		response := AgentManagementResponse{
			Success:   false,
			Error:     "Failed to update agent: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := AgentManagementResponse{
		Success:   true,
		Message:   "Agent updated successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// DeleteAgent handles DELETE /api/agent-management/agents/{id}
func (h *AgentManagementHandlers) DeleteAgent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["id"]

	if agentID == "" {
		response := AgentManagementResponse{
			Success:   false,
			Error:     "Agent ID is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	if err := h.agentManagementService.DeleteAgent(agentID); err != nil {
		response := AgentManagementResponse{
			Success:   false,
			Error:     "Failed to delete agent: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := AgentManagementResponse{
		Success:   true,
		Message:   "Agent deleted successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// PostAgentAction handles POST /api/agent-management/agents/{id}/actions
func (h *AgentManagementHandlers) PostAgentAction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["id"]

	if agentID == "" {
		response := AgentManagementResponse{
			Success:   false,
			Error:     "Agent ID is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	var action models.AgentAction
	if err := json.NewDecoder(r.Body).Decode(&action); err != nil {
		response := AgentManagementResponse{
			Success:   false,
			Error:     "Invalid request body",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	if err := h.agentManagementService.ExecuteAgentAction(agentID, &action); err != nil {
		response := AgentManagementResponse{
			Success:   false,
			Error:     "Failed to execute action: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := AgentManagementResponse{
		Success:   true,
		Message:   "Action executed successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetAgentSummary handles GET /api/agent-management/summary
func (h *AgentManagementHandlers) GetAgentSummary(w http.ResponseWriter, r *http.Request) {
	summary := h.agentManagementService.GetAgentSummary()

	response := AgentManagementResponse{
		Success:   true,
		Data:      summary,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetAgentMetrics handles GET /api/agent-management/agents/{id}/metrics
func (h *AgentManagementHandlers) GetAgentMetrics(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["id"]

	if agentID == "" {
		response := AgentManagementResponse{
			Success:   false,
			Error:     "Agent ID is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	limit := 100
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	metrics, err := h.agentManagementService.GetAgentMetrics(agentID, limit)
	if err != nil {
		response := AgentManagementResponse{
			Success:   false,
			Error:     "Failed to retrieve metrics: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := AgentManagementResponse{
		Success:   true,
		Data:      metrics,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetAgentLogs handles GET /api/agent-management/agents/{id}/logs
func (h *AgentManagementHandlers) GetAgentLogs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["id"]

	if agentID == "" {
		response := AgentManagementResponse{
			Success:   false,
			Error:     "Agent ID is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	limit := 100
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	logs, err := h.agentManagementService.GetAgentLogs(agentID, limit)
	if err != nil {
		response := AgentManagementResponse{
			Success:   false,
			Error:     "Failed to retrieve logs: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := AgentManagementResponse{
		Success:   true,
		Data:      logs,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetAgentEvents handles GET /api/agent-management/agents/{id}/events
func (h *AgentManagementHandlers) GetAgentEvents(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["id"]

	limit := 100
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	events, err := h.agentManagementService.GetAgentEvents(agentID, limit)
	if err != nil {
		response := AgentManagementResponse{
			Success:   false,
			Error:     "Failed to retrieve events: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := AgentManagementResponse{
		Success:   true,
		Data:      events,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetAgentTemplates handles GET /api/agent-management/templates
func (h *AgentManagementHandlers) GetAgentTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := h.agentManagementService.GetAgentTemplates()
	if err != nil {
		response := AgentManagementResponse{
			Success:   false,
			Error:     "Failed to retrieve templates: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := AgentManagementResponse{
		Success:   true,
		Data:      templates,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// PostAgentTemplate handles POST /api/agent-management/templates
func (h *AgentManagementHandlers) PostAgentTemplate(w http.ResponseWriter, r *http.Request) {
	var template models.AgentTemplate
	if err := json.NewDecoder(r.Body).Decode(&template); err != nil {
		response := AgentManagementResponse{
			Success:   false,
			Error:     "Invalid request body",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	if err := h.agentManagementService.CreateAgentTemplate(&template); err != nil {
		response := AgentManagementResponse{
			Success:   false,
			Error:     "Failed to create template: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := AgentManagementResponse{
		Success:   true,
		Data:      &template,
		Message:   "Template created successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// RegisterRoutes registers the agent management routes with the router
func (h *AgentManagementHandlers) RegisterRoutes(r *mux.Router, authMiddleware *middleware.AuthMiddleware) {
	// Create a subrouter for agent management endpoints
	agentRouter := r.PathPrefix("/api/agent-management").Subrouter()

	// Protected routes for agent management
	if authMiddleware != nil {
		protectedAgentRouter := agentRouter.PathPrefix("").Subrouter()
		protectedAgentRouter.Use(authMiddleware.RequireAuth)

		// Agent CRUD operations
		protectedAgentRouter.HandleFunc("/agents", h.GetAgents).Methods("GET")
		protectedAgentRouter.HandleFunc("/agents", h.PostAgent).Methods("POST")
		protectedAgentRouter.HandleFunc("/agents/{id}", h.GetAgent).Methods("GET")
		protectedAgentRouter.HandleFunc("/agents/{id}", h.PutAgent).Methods("PUT")
		protectedAgentRouter.HandleFunc("/agents/{id}", h.DeleteAgent).Methods("DELETE")

		// Agent actions
		protectedAgentRouter.HandleFunc("/agents/{id}/actions", h.PostAgentAction).Methods("POST")

		// Agent monitoring
		protectedAgentRouter.HandleFunc("/agents/{id}/metrics", h.GetAgentMetrics).Methods("GET")
		protectedAgentRouter.HandleFunc("/agents/{id}/logs", h.GetAgentLogs).Methods("GET")
		protectedAgentRouter.HandleFunc("/agents/{id}/events", h.GetAgentEvents).Methods("GET")

		// Templates
		protectedAgentRouter.HandleFunc("/templates", h.GetAgentTemplates).Methods("GET")
		protectedAgentRouter.HandleFunc("/templates", h.PostAgentTemplate).Methods("POST")

		// Summary
		protectedAgentRouter.HandleFunc("/summary", h.GetAgentSummary).Methods("GET")
	} else {
		// If no auth middleware, allow all routes (for testnet mode)
		agentRouter.HandleFunc("/agents", h.GetAgents).Methods("GET")
		agentRouter.HandleFunc("/agents", h.PostAgent).Methods("POST")
		agentRouter.HandleFunc("/agents/{id}", h.GetAgent).Methods("GET")
		agentRouter.HandleFunc("/agents/{id}", h.PutAgent).Methods("PUT")
		agentRouter.HandleFunc("/agents/{id}", h.DeleteAgent).Methods("DELETE")
		agentRouter.HandleFunc("/agents/{id}/actions", h.PostAgentAction).Methods("POST")
		agentRouter.HandleFunc("/agents/{id}/metrics", h.GetAgentMetrics).Methods("GET")
		agentRouter.HandleFunc("/agents/{id}/logs", h.GetAgentLogs).Methods("GET")
		agentRouter.HandleFunc("/agents/{id}/events", h.GetAgentEvents).Methods("GET")
		agentRouter.HandleFunc("/templates", h.GetAgentTemplates).Methods("GET")
		agentRouter.HandleFunc("/templates", h.PostAgentTemplate).Methods("POST")
		agentRouter.HandleFunc("/summary", h.GetAgentSummary).Methods("GET")
	}

	// Handle OPTIONS requests for CORS
	agentRouter.Methods("OPTIONS").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}
