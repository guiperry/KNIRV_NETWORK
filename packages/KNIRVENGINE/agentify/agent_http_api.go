// agent_http_api.go
package agentify

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// AgentHTTPAPI provides an HTTP API for inference clients
type AgentHTTPAPI struct {
	// inferencer is an abstraction so tests can inject mocks. Use the concrete
	// *AgentInferencer in production; it implements this interface.
	inferencer     InferencerInterface
	authMiddleware *AuthMiddleware
	upgrader       websocket.Upgrader
}

// NewAgentHTTPAPI creates a new HTTP API for inference clients
// InferencerInterface defines the minimal set of methods the HTTP API needs
// from an inferencer. This allows tests to provide lightweight mocks.
type InferencerInterface interface {
	ListAvailableAgents(ctx context.Context) ([]string, error)
	ActivateAgent(ctx context.Context, agentID, version, sessionID string, config map[string]interface{}) error
	DeactivateAgent(ctx context.Context, sessionID string) error
	ProcessInference(ctx context.Context, sessionID string, request *InferenceRequest) (*InferenceResponse, error)
	GetAgentSchema(ctx context.Context, sessionID string) (*AgentSchema, error)
	GetAgentCapabilities(ctx context.Context, sessionID string) (*AgentCapabilities, error)
	GetTEEInfo(ctx context.Context, sessionID string) (map[string]interface{}, error)
	GetAgentMemory(ctx context.Context, sessionID, key string) (interface{}, error)
	SetAgentMemory(ctx context.Context, sessionID, key string, value interface{}) error
	CreateTerminal(ctx context.Context, sessionID string, rows, cols int) (string, error)
	ResizeTerminal(ctx context.Context, sessionID, terminalID string, rows, cols int) error
	WriteToTerminal(ctx context.Context, sessionID, terminalID string, data []byte) error
	ReadFromTerminal(ctx context.Context, sessionID, terminalID string) ([]byte, error)
	CloseTerminal(ctx context.Context, sessionID, terminalID string) error
	// WASM / plugin management helpers
	DiscoverWASMPluginZips(ctx context.Context) ([]*WASMPluginInfo, error)
	InstallWASMPlugin(ctx context.Context, zipPath string) (*WASMPluginInfo, error)
	UninstallWASMPlugin(ctx context.Context, agentID, version string) error
	ListInstalledWASMPlugins(ctx context.Context) ([]*WASMPluginInfo, error)
	GetAvailableAgentsDetailed(ctx context.Context) (map[string]interface{}, error)
	// GetTerminalOutputChannel returns a channel streaming terminal output for a session/terminal
	GetTerminalOutputChannel(ctx context.Context, sessionID, terminalID string) (<-chan []byte, error)
}

func NewAgentHTTPAPI(inferencer InferencerInterface) *AgentHTTPAPI {
	// Create a default API key auth provider
	authProvider := NewAPIKeyAuthProvider()

	// Add a default API key for testing
	authProvider.AddAPIKey("test-api-key", "test-user")

	return &AgentHTTPAPI{
		inferencer:     inferencer,
		authMiddleware: NewAuthMiddleware(authProvider),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Allow all origins for now - in production, this should be more restrictive
				return true
			},
		},
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

	// Terminal-related endpoints
	mux.HandleFunc("/v1/terminal/create", a.handleCreateTerminal)
	mux.HandleFunc("/v1/terminal/resize", a.handleResizeTerminal)
	mux.HandleFunc("/v1/terminal/write", a.handleWriteToTerminal)
	mux.HandleFunc("/v1/terminal/read", a.handleReadFromTerminal)
	mux.HandleFunc("/v1/terminal/close", a.handleCloseTerminal)
	mux.HandleFunc("/v1/terminal/ws", a.handleTerminalWebSocket)

	// WASM Plugin Management endpoints
	mux.HandleFunc("/v1/plugins/wasm/discover", a.handleDiscoverWASMPlugins)
	mux.HandleFunc("/v1/plugins/wasm/install", a.handleInstallWASMPlugin)
	mux.HandleFunc("/v1/plugins/wasm/uninstall", a.handleUninstallWASMPlugin)
	mux.HandleFunc("/v1/plugins/wasm/installed", a.handleListInstalledWASMPlugins)
	mux.HandleFunc("/v1/agents/detailed", a.handleGetAvailableAgentsDetailed)
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

// Terminal handler methods

// handleCreateTerminal handles requests to create a terminal session
func (a *AgentHTTPAPI) handleCreateTerminal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		SessionID string `json:"sessionId"`
		Rows      int    `json:"rows"`
		Cols      int    `json:"cols"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	terminalID, err := a.inferencer.CreateTerminal(ctx, req.SessionID, req.Rows, req.Cols)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"terminalId": terminalID,
		"sessionId":  req.SessionID,
		"rows":       req.Rows,
		"cols":       req.Cols,
	})
}

// handleResizeTerminal handles requests to resize a terminal session
func (a *AgentHTTPAPI) handleResizeTerminal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		SessionID  string `json:"sessionId"`
		TerminalID string `json:"terminalId"`
		Rows       int    `json:"rows"`
		Cols       int    `json:"cols"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	if err := a.inferencer.ResizeTerminal(ctx, req.SessionID, req.TerminalID, req.Rows, req.Cols); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "resized",
		"terminalId": req.TerminalID,
		"rows":       req.Rows,
		"cols":       req.Cols,
	})
}

// handleWriteToTerminal handles requests to write data to a terminal session
func (a *AgentHTTPAPI) handleWriteToTerminal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		SessionID  string `json:"sessionId"`
		TerminalID string `json:"terminalId"`
		Data       string `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	if err := a.inferencer.WriteToTerminal(ctx, req.SessionID, req.TerminalID, []byte(req.Data)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "written",
		"terminalId": req.TerminalID,
	})
}

// handleReadFromTerminal handles requests to read data from a terminal session
func (a *AgentHTTPAPI) handleReadFromTerminal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.URL.Query().Get("sessionId")
	if sessionID == "" {
		http.Error(w, "Missing sessionId parameter", http.StatusBadRequest)
		return
	}

	terminalID := r.URL.Query().Get("terminalId")
	if terminalID == "" {
		http.Error(w, "Missing terminalId parameter", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	data, err := a.inferencer.ReadFromTerminal(ctx, sessionID, terminalID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"terminalId": terminalID,
		"data":       string(data),
	})
}

// handleCloseTerminal handles requests to close a terminal session
func (a *AgentHTTPAPI) handleCloseTerminal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		SessionID  string `json:"sessionId"`
		TerminalID string `json:"terminalId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	if err := a.inferencer.CloseTerminal(ctx, req.SessionID, req.TerminalID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "closed",
		"terminalId": req.TerminalID,
	})
}

// handleTerminalWebSocket handles WebSocket connections for terminal sessions
func (a *AgentHTTPAPI) handleTerminalWebSocket(w http.ResponseWriter, r *http.Request) {
	// Get query parameters
	sessionID := r.URL.Query().Get("sessionId")
	if sessionID == "" {
		http.Error(w, "Missing sessionId parameter", http.StatusBadRequest)
		return
	}

	terminalID := r.URL.Query().Get("terminalId")
	if terminalID == "" {
		http.Error(w, "Missing terminalId parameter", http.StatusBadRequest)
		return
	}

	// Upgrade the HTTP connection to WebSocket
	conn, err := a.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade WebSocket connection: %v", err)
		return
	}
	defer conn.Close()

	log.Printf("WebSocket connection established for terminal %s (session %s)", terminalID, sessionID)

	// Get the terminal output channel
	ctx := r.Context()
	outputChan, err := a.getTerminalOutputChannel(ctx, sessionID, terminalID)
	if err != nil {
		log.Printf("Failed to get terminal output channel: %v", err)
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Error: %v", err)))
		return
	}

	// Handle WebSocket communication
	go a.handleTerminalOutput(conn, outputChan)
	a.handleTerminalInput(conn, sessionID, terminalID)
}

// getTerminalOutputChannel gets the output channel for a terminal session
func (a *AgentHTTPAPI) getTerminalOutputChannel(ctx context.Context, sessionID, terminalID string) (<-chan []byte, error) {
	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Delegate to the inferencer's terminal output accessor
	return a.inferencer.GetTerminalOutputChannel(ctx, sessionID, terminalID)
}

// handleTerminalOutput handles sending terminal output to the WebSocket client
func (a *AgentHTTPAPI) handleTerminalOutput(conn *websocket.Conn, outputChan <-chan []byte) {
	for data := range outputChan {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("Failed to write terminal output to WebSocket: %v", err)
			break
		}
	}
}

// WASM Plugin Management Handlers

// handleDiscoverWASMPlugins handles requests to discover available WASM plugin zip files
func (a *AgentHTTPAPI) handleDiscoverWASMPlugins(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	plugins, err := a.inferencer.DiscoverWASMPluginZips(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"plugins": plugins,
	})
}

// handleInstallWASMPlugin handles requests to install a WASM plugin from a zip file
func (a *AgentHTTPAPI) handleInstallWASMPlugin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ZipPath string `json:"zipPath"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.ZipPath == "" {
		http.Error(w, "Missing zipPath parameter", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	pluginInfo, err := a.inferencer.InstallWASMPlugin(ctx, req.ZipPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "installed",
		"plugin": pluginInfo,
	})
}

// handleUninstallWASMPlugin handles requests to uninstall a WASM plugin
func (a *AgentHTTPAPI) handleUninstallWASMPlugin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		AgentID string `json:"agentId"`
		Version string `json:"version"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.AgentID == "" || req.Version == "" {
		http.Error(w, "Missing agentId or version parameter", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	if err := a.inferencer.UninstallWASMPlugin(ctx, req.AgentID, req.Version); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "uninstalled",
		"agentId": req.AgentID,
		"version": req.Version,
	})
}

// handleListInstalledWASMPlugins handles requests to list installed WASM plugins
func (a *AgentHTTPAPI) handleListInstalledWASMPlugins(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	plugins, err := a.inferencer.ListInstalledWASMPlugins(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"plugins": plugins,
	})
}

// handleGetAvailableAgentsDetailed handles requests to get detailed information about all available agents
func (a *AgentHTTPAPI) handleGetAvailableAgentsDetailed(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	agentsInfo, err := a.inferencer.GetAvailableAgentsDetailed(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agentsInfo)
}

// handleTerminalInput handles receiving input from the WebSocket client and sending it to the terminal
func (a *AgentHTTPAPI) handleTerminalInput(conn *websocket.Conn, sessionID, terminalID string) {
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Failed to read from WebSocket: %v", err)
			break
		}

		// Write the input to the terminal
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := a.inferencer.WriteToTerminal(ctx, sessionID, terminalID, message); err != nil {
			log.Printf("Failed to write to terminal: %v", err)
		}
		cancel()
	}
}
