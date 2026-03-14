// agent_http_api.go
package agentify

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// AgentHTTPAPI provides an HTTP API for inference clients
type AgentHTTPAPI struct {
	inferencer     *AgentInferencer
	authMiddleware *AuthMiddleware
}

// NewAgentHTTPAPI creates a new HTTP API for inference clients
func NewAgentHTTPAPI(inferencer *AgentInferencer) *AgentHTTPAPI {
	// Create a default API key auth provider
	authProvider := NewAPIKeyAuthProvider()

	// Add a default API key for testing
	authProvider.AddAPIKey("test-api-key", "test-user")

	return &AgentHTTPAPI{
		inferencer:     inferencer,
		authMiddleware: NewAuthMiddleware(authProvider),
	}
}

// SetAuthProvider sets the authentication provider
func (a *AgentHTTPAPI) SetAuthProvider(provider AuthProvider) {
	a.authMiddleware = NewAuthMiddleware(provider)
}

// RegisterHandlers registers the HTTP handlers
func (a *AgentHTTPAPI) RegisterHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/v1/agents", a.handleListAgents)
	mux.HandleFunc("/v1/agents/activate", a.handleActivateAgent)
	mux.HandleFunc("/v1/agents/deactivate", a.handleDeactivateAgent)
	mux.HandleFunc("/v1/inference", a.handleInference)
	mux.HandleFunc("/v1/schema", a.handleGetSchema)
	mux.HandleFunc("/v1/capabilities", a.handleGetCapabilities)
	mux.HandleFunc("/v1/memory", a.handleMemory)
	mux.HandleFunc("/v1/tee", a.handleTEEInfo)
}

// handleListAgents handles requests to list available agents
func (a *AgentHTTPAPI) handleListAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	agents, err := a.inferencer.ListAvailableAgents(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"agents": agents,
	})
}

// handleActivateAgent handles requests to activate an agent
func (a *AgentHTTPAPI) handleActivateAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		AgentID   string                 `json:"agentId"`
		Version   string                 `json:"version"`
		SessionID string                 `json:"sessionId"`
		Config    map[string]interface{} `json:"config,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	if err := a.inferencer.ActivateAgent(ctx, req.AgentID, req.Version, req.SessionID, req.Config); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "activated",
		"agentId":   req.AgentID,
		"sessionId": req.SessionID,
	})
}

// handleDeactivateAgent handles requests to deactivate an agent
func (a *AgentHTTPAPI) handleDeactivateAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		SessionID string `json:"sessionId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	if err := a.inferencer.DeactivateAgent(ctx, req.SessionID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "deactivated",
		"sessionId": req.SessionID,
	})
}

// handleInference handles inference requests
func (a *AgentHTTPAPI) handleInference(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		SessionID  string                 `json:"sessionId"`
		Input      string                 `json:"input"`
		History    []*ConversationMessage `json:"history,omitempty"`
		Parameters map[string]interface{} `json:"parameters,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	inferenceReq := &InferenceRequest{
		Input:      req.Input,
		History:    req.History,
		SessionID:  req.SessionID,
		Parameters: req.Parameters,
	}

	response, err := a.inferencer.ProcessInference(ctx, req.SessionID, inferenceReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetSchema handles requests to get an agent's schema
func (a *AgentHTTPAPI) handleGetSchema(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.URL.Query().Get("sessionId")
	if sessionID == "" {
		http.Error(w, "Missing sessionId parameter", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	schema, err := a.inferencer.GetAgentSchema(ctx, sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(schema)
}

// handleGetCapabilities handles requests to get an agent's capabilities
func (a *AgentHTTPAPI) handleGetCapabilities(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.URL.Query().Get("sessionId")
	if sessionID == "" {
		http.Error(w, "Missing sessionId parameter", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	capabilities, err := a.inferencer.GetAgentCapabilities(ctx, sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(capabilities)
}

// handleMemory handles requests to get or set memory values
func (a *AgentHTTPAPI) handleMemory(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("sessionId")
	if sessionID == "" {
		http.Error(w, "Missing sessionId parameter", http.StatusBadRequest)
		return
	}

	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "Missing key parameter", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	switch r.Method {
	case http.MethodGet:
		// Get memory value
		value, err := a.inferencer.GetAgentMemory(ctx, sessionID, key)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"key":   key,
			"value": value,
		})

	case http.MethodPost:
		// Set memory value
		var req struct {
			Value interface{} `json:"value"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := a.inferencer.SetAgentMemory(ctx, sessionID, key, req.Value); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"key":    key,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleTEEInfo handles requests to get TEE information
func (a *AgentHTTPAPI) handleTEEInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.URL.Query().Get("sessionId")
	if sessionID == "" {
		http.Error(w, "Missing sessionId parameter", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	info, err := a.inferencer.GetTEEInfo(ctx, sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}
