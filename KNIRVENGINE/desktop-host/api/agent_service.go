package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// Agent represents an inference agent
type Agent struct {
	ID        string    `json:"id"`
	OwnerID   int64     `json:"owner_id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Config    string    `json:"config"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AgentRepository defines the interface for agent data operations
type AgentRepository interface {
	GetAgentsByOwner(ctx context.Context, ownerID int64) ([]*Agent, error)
	GetAgentByID(ctx context.Context, id string) (*Agent, error)
	CreateAgent(ctx context.Context, agent *Agent) error
	UpdateAgent(ctx context.Context, agent *Agent) error
	DeleteAgent(ctx context.Context, id string) error
}

// AgentService manages agent lifecycle operations
type AgentService struct {
	repository AgentRepository
}

// InMemoryAgentRepository is a simple in-memory implementation
// Note: For production use, replace with database-backed implementation
type InMemoryAgentRepository struct {
	agents map[string]*Agent
}

func NewInMemoryAgentRepository() *InMemoryAgentRepository {
	return &InMemoryAgentRepository{
		agents: make(map[string]*Agent),
	}
}

func (m *InMemoryAgentRepository) GetAgentsByOwner(ctx context.Context, ownerID int64) ([]*Agent, error) {
	var result []*Agent
	for _, agent := range m.agents {
		if agent.OwnerID == ownerID {
			result = append(result, agent)
		}
	}
	return result, nil
}

func (m *InMemoryAgentRepository) GetAgentByID(ctx context.Context, id string) (*Agent, error) {
	if agent, exists := m.agents[id]; exists {
		return agent, nil
	}
	return nil, fmt.Errorf("agent not found")
}

func (m *InMemoryAgentRepository) CreateAgent(ctx context.Context, agent *Agent) error {
	m.agents[agent.ID] = agent
	return nil
}

func (m *InMemoryAgentRepository) UpdateAgent(ctx context.Context, agent *Agent) error {
	if _, exists := m.agents[agent.ID]; !exists {
		return fmt.Errorf("agent not found")
	}
	m.agents[agent.ID] = agent
	return nil
}

func (m *InMemoryAgentRepository) DeleteAgent(ctx context.Context, id string) error {
	if _, exists := m.agents[id]; !exists {
		return fmt.Errorf("agent not found")
	}
	delete(m.agents, id)
	return nil
}

// NewAgentService creates a new agent service
func NewAgentService(repo AgentRepository) *AgentService {
	return &AgentService{
		repository: repo,
	}
}

// RegisterHandlers registers the agent API handlers
func (s *AgentService) RegisterHandlers(router *mux.Router) {
	router.HandleFunc("/api/v1/agents", s.handleListAgents).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/v1/agents", s.handleCreateAgent).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/v1/agents/{id}", s.handleGetAgent).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/v1/agents/{id}", s.handleUpdateAgent).Methods("PUT", "OPTIONS")
	router.HandleFunc("/api/v1/agents/{id}", s.handleDeleteAgent).Methods("DELETE", "OPTIONS")

	// ADK-specific endpoints
	router.HandleFunc("/api/v1/agents/discover", s.handleDiscoverAgents).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/v1/agents/message", s.handleAgentMessage).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/v1/agents/tasks/{id}", s.handleGetTask).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/v1/agents/capabilities", s.handleAgentCapabilities).Methods("POST", "OPTIONS")
}

// handleListAgents handles GET /api/v1/agents
func (s *AgentService) handleListAgents(w http.ResponseWriter, r *http.Request) {
	// Handle CORS preflight
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.WriteHeader(http.StatusOK)
		return
	}

	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	// Get user ID from context
	userID, ok := r.Context().Value("userID").(int64)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get agents for user
	agents, err := s.repository.GetAgentsByOwner(r.Context(), userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get agents")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"agents": agents,
	})
}

// handleGetAgent handles GET /api/v1/agents/:id
func (s *AgentService) handleGetAgent(w http.ResponseWriter, r *http.Request) {
	// Handle CORS preflight
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.WriteHeader(http.StatusOK)
		return
	}

	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	// Get user ID from context
	userID, ok := r.Context().Value("userID").(int64)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get agent ID from path
	vars := mux.Vars(r)
	agentID := vars["id"]

	// Get agent
	agent, err := s.repository.GetAgentByID(r.Context(), agentID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Agent not found")
		return
	}

	// Verify ownership
	if agent.OwnerID != userID {
		respondWithError(w, http.StatusForbidden, "Access denied")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"agent": agent,
	})
}

// handleCreateAgent handles POST /api/v1/agents
func (s *AgentService) handleCreateAgent(w http.ResponseWriter, r *http.Request) {
	// Handle CORS preflight
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.WriteHeader(http.StatusOK)
		return
	}

	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	// Get user ID from context
	userID, ok := r.Context().Value("userID").(int64)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Parse request
	var agent Agent
	if err := parseJSONBody(r, &agent); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	// Set required fields
	agent.ID = uuid.New().String()
	agent.OwnerID = userID
	agent.CreatedAt = time.Now()

	// Create agent
	if err := s.repository.CreateAgent(r.Context(), &agent); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create agent")
		return
	}

	respondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"agent": agent,
	})
}

// handleUpdateAgent handles PUT /api/v1/agents/:id
func (s *AgentService) handleUpdateAgent(w http.ResponseWriter, r *http.Request) {
	// Handle CORS preflight
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "PUT, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.WriteHeader(http.StatusOK)
		return
	}

	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	// Get user ID from context
	userID, ok := r.Context().Value("userID").(int64)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get agent ID from path
	vars := mux.Vars(r)
	agentID := vars["id"]

	// Get existing agent
	existingAgent, err := s.repository.GetAgentByID(r.Context(), agentID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Agent not found")
		return
	}

	// Verify ownership
	if existingAgent.OwnerID != userID {
		respondWithError(w, http.StatusForbidden, "Access denied")
		return
	}

	// Parse request
	var agent Agent
	if err := parseJSONBody(r, &agent); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	// Preserve immutable fields
	agent.ID = agentID
	agent.OwnerID = userID
	agent.CreatedAt = existingAgent.CreatedAt

	// Update agent
	if err := s.repository.UpdateAgent(r.Context(), &agent); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update agent")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"agent": agent,
	})
}

// handleDeleteAgent handles DELETE /api/v1/agents/:id
func (s *AgentService) handleDeleteAgent(w http.ResponseWriter, r *http.Request) {
	// Handle CORS preflight
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.WriteHeader(http.StatusOK)
		return
	}

	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	// Get user ID from context
	userID, ok := r.Context().Value("userID").(int64)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get agent ID from path
	vars := mux.Vars(r)
	agentID := vars["id"]

	// Get existing agent
	existingAgent, err := s.repository.GetAgentByID(r.Context(), agentID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Agent not found")
		return
	}

	// Verify ownership
	if existingAgent.OwnerID != userID {
		respondWithError(w, http.StatusForbidden, "Access denied")
		return
	}

	// Delete agent
	if err := s.repository.DeleteAgent(r.Context(), agentID); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to delete agent")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Agent deleted successfully",
	})
}

// handleDiscoverAgents handles GET /api/v1/agents/discover
func (s *AgentService) handleDiscoverAgents(w http.ResponseWriter, r *http.Request) {
	// Handle CORS preflight
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.WriteHeader(http.StatusOK)
		return
	}

	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	// Get user ID from context
	userID, ok := r.Context().Value("userID").(int64)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get agents for user
	agents, err := s.repository.GetAgentsByOwner(r.Context(), userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to discover agents")
		return
	}

	// Format discovery response
	discoveryResponse := struct {
		Agents []struct {
			ID           string   `json:"id"`
			Name         string   `json:"name"`
			Type         string   `json:"type"`
			Capabilities []string `json:"capabilities"`
		} `json:"agents"`
	}{
		Agents: make([]struct {
			ID           string   `json:"id"`
			Name         string   `json:"name"`
			Type         string   `json:"type"`
			Capabilities []string `json:"capabilities"`
		}, len(agents)),
	}

	for i, agent := range agents {
		// Parse capabilities from config
		var capabilities []string
		var config map[string]interface{}
		if err := json.Unmarshal([]byte(agent.Config), &config); err == nil {
			if caps, ok := config["capabilities"].([]interface{}); ok {
				for _, cap := range caps {
					capabilities = append(capabilities, cap.(string))
				}
			}
		}

		discoveryResponse.Agents[i] = struct {
			ID           string   `json:"id"`
			Name         string   `json:"name"`
			Type         string   `json:"type"`
			Capabilities []string `json:"capabilities"`
		}{
			ID:           agent.ID,
			Name:         agent.Name,
			Type:         agent.Type,
			Capabilities: capabilities,
		}
	}

	respondWithJSON(w, http.StatusOK, discoveryResponse)
}

// handleAgentMessage handles POST /api/v1/agents/message
func (s *AgentService) handleAgentMessage(w http.ResponseWriter, r *http.Request) {
	// Handle CORS preflight
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.WriteHeader(http.StatusOK)
		return
	}

	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	// Get user ID from context
	userID, ok := r.Context().Value("userID").(int64)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Parse request
	var message struct {
		SenderID    string                 `json:"sender_id"`
		ReceiverID  string                 `json:"receiver_id"`
		MessageType string                 `json:"message_type"`
		Content     map[string]interface{} `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid message format")
		return
	}

	// Verify sender and receiver ownership
	sender, err := s.repository.GetAgentByID(r.Context(), message.SenderID)
	if err != nil || sender.OwnerID != userID {
		respondWithError(w, http.StatusForbidden, "Invalid sender")
		return
	}

	receiver, err := s.repository.GetAgentByID(r.Context(), message.ReceiverID)
	if err != nil || receiver.OwnerID != userID {
		respondWithError(w, http.StatusForbidden, "Invalid receiver")
		return
	}

	// Echo the message back for now
	// In production, this would route to the appropriate agent
	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "Message received",
		"content": message.Content,
	})
}

// handleGetTask handles GET /api/v1/agents/tasks/{id}
func (s *AgentService) handleGetTask(w http.ResponseWriter, r *http.Request) {
	// Handle CORS preflight
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.WriteHeader(http.StatusOK)
		return
	}

	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	// Get user ID from context
	userID, ok := r.Context().Value("userID").(int64)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get task ID from path
	vars := mux.Vars(r)
	taskID := vars["id"]

	// Return task status - in production this would query actual task state
	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"status": "success",
		"task": map[string]interface{}{
			"id":         taskID,
			"status":     "completed",
			"created_at": time.Now().Format(time.RFC3339),
			"owner_id":   userID,
		},
	})
}

// handleAgentCapabilities handles POST /api/v1/agents/capabilities
func (s *AgentService) handleAgentCapabilities(w http.ResponseWriter, r *http.Request) {
	// Handle CORS preflight
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.WriteHeader(http.StatusOK)
		return
	}

	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	// Get user ID from context
	userID, ok := r.Context().Value("userID").(int64)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Parse request
	var capabilityRequest struct {
		AgentID      string   `json:"agent_id"`
		Capabilities []string `json:"capabilities"`
	}

	if err := json.NewDecoder(r.Body).Decode(&capabilityRequest); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid capability request")
		return
	}

	// Verify agent ownership
	agent, err := s.repository.GetAgentByID(r.Context(), capabilityRequest.AgentID)
	if err != nil || agent.OwnerID != userID {
		respondWithError(w, http.StatusForbidden, "Invalid agent")
		return
	}

	// Update agent capabilities in config
	var config map[string]interface{}
	if err := json.Unmarshal([]byte(agent.Config), &config); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to parse agent config")
		return
	}

	config["capabilities"] = capabilityRequest.Capabilities
	updatedConfig, err := json.Marshal(config)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update agent config")
		return
	}

	agent.Config = string(updatedConfig)
	if err := s.repository.UpdateAgent(r.Context(), agent); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update agent")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"status":       "success",
		"agent_id":     agent.ID,
		"capabilities": capabilityRequest.Capabilities,
	})
}
