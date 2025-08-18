package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// AgentTerminalHandler handles terminal sessions for agent plugins
type AgentTerminalHandler struct {
	terminalManager *TerminalManager
	agentRegistry   map[string]string // Maps agent IDs to their plugin paths
}

// NewAgentTerminalHandler creates a new agent terminal handler
func NewAgentTerminalHandler(terminalManager *TerminalManager) *AgentTerminalHandler {
	return &AgentTerminalHandler{
		terminalManager: terminalManager,
		agentRegistry:   make(map[string]string),
	}
}

// RegisterAgentPlugin registers an agent plugin with the terminal handler
func (h *AgentTerminalHandler) RegisterAgentPlugin(agentID, pluginPath string) {
	h.agentRegistry[agentID] = pluginPath
}

// GetAgentPluginPath gets the plugin path for an agent
func (h *AgentTerminalHandler) GetAgentPluginPath(agentID string) (string, bool) {
	path, ok := h.agentRegistry[agentID]
	return path, ok
}

// CreateTerminalRequest represents a request to create a terminal session
type CreateTerminalRequest struct {
	SessionID string `json:"sessionId"`
	AgentID   string `json:"agentId"`
	Rows      int    `json:"rows"`
	Cols      int    `json:"cols"`
}

// CreateTerminalResponse represents a response to a create terminal request
type CreateTerminalResponse struct {
	Success    bool   `json:"success"`
	TerminalID string `json:"terminalId,omitempty"`
	Error      string `json:"error,omitempty"`
}

// ResizeTerminalRequest represents a request to resize a terminal session
type ResizeTerminalRequest struct {
	SessionID  string `json:"sessionId"`
	TerminalID string `json:"terminalId"`
	Rows       int    `json:"rows"`
	Cols       int    `json:"cols"`
}

// CloseTerminalRequest represents a request to close a terminal session
type CloseTerminalRequest struct {
	SessionID  string `json:"sessionId"`
	TerminalID string `json:"terminalId"`
}

// TerminalLogEntry represents a terminal log entry
type TerminalLogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"` // stdout, stderr, system
	Data      []byte    `json:"data"`
	Source    string    `json:"source,omitempty"`
	ProcessID int       `json:"process_id,omitempty"`
}

// RegisterRoutes registers the agent terminal handler routes
func (h *AgentTerminalHandler) RegisterRoutes(router *mux.Router) {
	// Terminal management endpoints
	router.HandleFunc("/api/v1/terminal/create", h.handleCreateTerminal).Methods("POST")
	router.HandleFunc("/api/v1/terminal/resize", h.handleResizeTerminal).Methods("POST")
	router.HandleFunc("/api/v1/terminal/close", h.handleCloseTerminal).Methods("POST")
	router.HandleFunc("/api/v1/terminal/logs", h.handleGetLogs).Methods("GET")
	router.HandleFunc("/api/v1/terminal/ws", h.handleWebSocket)
}

// containsEnvVar checks if an environment variable with the given prefix exists in the slice
func containsEnvVar(env []string, prefix string) bool {
	for _, e := range env {
		if strings.HasPrefix(e, prefix) {
			return true
		}
	}
	return false
}

// handleCreateTerminal handles requests to create a terminal session
func (h *AgentTerminalHandler) handleCreateTerminal(w http.ResponseWriter, r *http.Request) {
	var req CreateTerminalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Validate request
	if req.SessionID == "" {
		http.Error(w, "Session ID is required", http.StatusBadRequest)
		return
	}

	// Get the agent plugin path if an agent ID is provided
	var env []string
	if req.AgentID != "" {
		pluginPath, ok := h.GetAgentPluginPath(req.AgentID)
		if ok {
			// Add agent-specific environment variables
			env = append(env, fmt.Sprintf("AGENT_ID=%s", req.AgentID))
			env = append(env, fmt.Sprintf("AGENT_PLUGIN_PATH=%s", pluginPath))
			env = append(env, fmt.Sprintf("SESSION_ID=%s", req.SessionID))
		}
	}

	// Always add session ID to environment
	if !containsEnvVar(env, "SESSION_ID=") {
		env = append(env, fmt.Sprintf("SESSION_ID=%s", req.SessionID))
	}

	// Create the terminal session
	session, err := h.terminalManager.CreateSession(0, "", env)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create terminal session: %v", err), http.StatusInternalServerError)
		return
	}

	// Start the terminal session
	if err := session.Start(); err != nil {
		http.Error(w, fmt.Sprintf("Failed to start terminal session: %v", err), http.StatusInternalServerError)
		return
	}

	// Resize the terminal if rows and cols are provided
	if req.Rows > 0 && req.Cols > 0 {
		// TODO: Implement terminal resize
	}

	// Return the terminal session ID
	response := CreateTerminalResponse{
		Success:    true,
		TerminalID: session.ID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleResizeTerminal handles requests to resize a terminal session
func (h *AgentTerminalHandler) handleResizeTerminal(w http.ResponseWriter, r *http.Request) {
	var req ResizeTerminalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Validate request
	if req.SessionID == "" || req.TerminalID == "" {
		http.Error(w, "Session ID and Terminal ID are required", http.StatusBadRequest)
		return
	}

	// Get the terminal session
	session, err := h.terminalManager.GetSession(req.TerminalID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Terminal session not found: %v", err), http.StatusNotFound)
		return
	}

	// TODO: Implement terminal resize
	// For now, just update the last activity time
	session.LastActivity = time.Now()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// handleCloseTerminal handles requests to close a terminal session
func (h *AgentTerminalHandler) handleCloseTerminal(w http.ResponseWriter, r *http.Request) {
	var req CloseTerminalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Validate request
	if req.SessionID == "" || req.TerminalID == "" {
		http.Error(w, "Session ID and Terminal ID are required", http.StatusBadRequest)
		return
	}

	// Close the terminal session
	if err := h.terminalManager.CloseSession(req.TerminalID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to close terminal session: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// handleGetLogs handles requests to get terminal logs
func (h *AgentTerminalHandler) handleGetLogs(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("sessionId")
	terminalID := r.URL.Query().Get("terminalId")

	// Validate request
	if sessionID == "" || terminalID == "" {
		http.Error(w, "Session ID and Terminal ID are required", http.StatusBadRequest)
		return
	}

	// Get the terminal session
	session, err := h.terminalManager.GetSession(terminalID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Terminal session not found: %v", err), http.StatusNotFound)
		return
	}

	// Get the terminal history
	history := session.GetHistory()

	// Convert history to log entries
	logs := make([]TerminalLogEntry, len(history))
	for i, entry := range history {
		logs[i] = TerminalLogEntry{
			Timestamp: entry.Timestamp,
			Type:      "stdout", // Assume stdout for now
			Data:      []byte(entry.Output),
			Source:    "terminal",
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"logs":    logs,
	})
}

// handleWebSocket handles WebSocket connections for terminal sessions
func (h *AgentTerminalHandler) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("sessionId")
	terminalID := r.URL.Query().Get("terminalId")

	// Validate request
	if sessionID == "" || terminalID == "" {
		http.Error(w, "Session ID and Terminal ID are required", http.StatusBadRequest)
		return
	}

	// Get the terminal session
	_, err := h.terminalManager.GetSession(terminalID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Terminal session not found: %v", err), http.StatusNotFound)
		return
	}

	// Use the existing terminal WebSocket handler
	h.terminalManager.HandleTerminalWebSocket(w, r, terminalID)
}
