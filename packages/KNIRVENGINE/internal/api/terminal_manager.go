package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// TerminalSession represents an active terminal session
type TerminalSession struct {
	ID           string    `json:"id"`
	CreatedAt    time.Time `json:"createdAt"`
	LastActivity time.Time `json:"lastActivity"`
	UserID       int64     `json:"userId"`
	Command      string    `json:"command"`
	WorkingDir   string    `json:"workingDir"`
	Env          []string  `json:"env"`
	Status       string    `json:"status"`
	ExitCode     int       `json:"exitCode"`
	
	// Private fields
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	stdout     io.ReadCloser
	stderr     io.ReadCloser
	ctx        context.Context
	cancelFunc context.CancelFunc
	mutex      sync.Mutex
	clients    map[*websocket.Conn]bool
	history    []TerminalHistoryEntry
}

// TerminalHistoryEntry represents a command history entry
type TerminalHistoryEntry struct {
	Command   string    `json:"command"`
	Timestamp time.Time `json:"timestamp"`
	Output    string    `json:"output"`
	ExitCode  int       `json:"exitCode"`
}

// TerminalManager manages terminal sessions
type TerminalManager struct {
	sessions      map[string]*TerminalSession
	mutex         sync.RWMutex
	defaultShell  string
	maxSessions   int
	sessionExpiry time.Duration
}

// NewTerminalManager creates a new terminal manager
func NewTerminalManager() *TerminalManager {
	// Determine default shell
	shell := os.Getenv("SHELL")
	if shell == "" {
		// Default to bash on Unix-like systems, cmd.exe on Windows
		if _, err := os.Stat("/bin/bash"); err == nil {
			shell = "/bin/bash"
		} else if _, err := os.Stat("/bin/sh"); err == nil {
			shell = "/bin/sh"
		} else {
			shell = "cmd.exe"
		}
	}

	return &TerminalManager{
		sessions:      make(map[string]*TerminalSession),
		defaultShell:  shell,
		maxSessions:   10,
		sessionExpiry: 30 * time.Minute,
	}
}

// CreateSession creates a new terminal session
func (m *TerminalManager) CreateSession(userID int64, workingDir string, env []string) (*TerminalSession, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Check if we've reached the maximum number of sessions
	if len(m.sessions) >= m.maxSessions {
		return nil, fmt.Errorf("maximum number of terminal sessions reached (%d)", m.maxSessions)
	}

	// Create a new session ID
	sessionID := uuid.New().String()

	// Create a cancelable context
	ctx, cancelFunc := context.WithCancel(context.Background())

	// Create the session
	session := &TerminalSession{
		ID:           sessionID,
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
		UserID:       userID,
		Command:      m.defaultShell,
		WorkingDir:   workingDir,
		Env:          env,
		Status:       "created",
		ctx:          ctx,
		cancelFunc:   cancelFunc,
		clients:      make(map[*websocket.Conn]bool),
		history:      make([]TerminalHistoryEntry, 0),
	}

	// Store the session
	m.sessions[sessionID] = session

	return session, nil
}

// GetSession retrieves a terminal session by ID
func (m *TerminalManager) GetSession(sessionID string) (*TerminalSession, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	session, ok := m.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("terminal session not found: %s", sessionID)
	}

	return session, nil
}

// ListSessions lists all terminal sessions for a user
func (m *TerminalManager) ListSessions(userID int64) []*TerminalSession {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var sessions []*TerminalSession
	for _, session := range m.sessions {
		if session.UserID == userID {
			sessions = append(sessions, session)
		}
	}

	return sessions
}

// CloseSession closes a terminal session
func (m *TerminalManager) CloseSession(sessionID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	session, ok := m.sessions[sessionID]
	if !ok {
		return fmt.Errorf("terminal session not found: %s", sessionID)
	}

	// Close the session
	if err := session.Close(); err != nil {
		return fmt.Errorf("failed to close terminal session: %v", err)
	}

	// Remove the session
	delete(m.sessions, sessionID)

	return nil
}

// CleanupExpiredSessions removes expired sessions
func (m *TerminalManager) CleanupExpiredSessions() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	now := time.Now()
	for id, session := range m.sessions {
		if now.Sub(session.LastActivity) > m.sessionExpiry {
			log.Printf("Closing expired terminal session: %s", id)
			session.Close()
			delete(m.sessions, id)
		}
	}
}

// Start starts the terminal session
func (s *TerminalSession) Start() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.cmd != nil {
		return fmt.Errorf("terminal session already started")
	}

	// Create the command
	s.cmd = exec.CommandContext(s.ctx, s.Command)

	// Set the working directory
	if s.WorkingDir != "" {
		s.cmd.Dir = s.WorkingDir
	}

	// Set environment variables
	if len(s.Env) > 0 {
		s.cmd.Env = append(os.Environ(), s.Env...)
	} else {
		s.cmd.Env = os.Environ()
	}

	// Create pipes for stdin, stdout, and stderr
	var err error
	s.stdin, err = s.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %v", err)
	}

	s.stdout, err = s.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %v", err)
	}

	s.stderr, err = s.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	// Start the command
	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %v", err)
	}

	s.Status = "running"
	s.LastActivity = time.Now()

	// Start goroutines to handle stdout and stderr
	go s.handleOutput(s.stdout, false)
	go s.handleOutput(s.stderr, true)

	// Start a goroutine to wait for the command to complete
	go func() {
		err := s.cmd.Wait()
		s.mutex.Lock()
		defer s.mutex.Unlock()

		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				s.ExitCode = exitErr.ExitCode()
			} else {
				s.ExitCode = 1
			}
			s.Status = "exited"
		} else {
			s.ExitCode = 0
			s.Status = "completed"
		}

		// Notify all clients that the session has ended
		s.broadcastToClients(map[string]interface{}{
			"type":     "session_end",
			"exitCode": s.ExitCode,
			"status":   s.Status,
		})
	}()

	return nil
}

// Close closes the terminal session
func (s *TerminalSession) Close() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Cancel the context to stop the command
	if s.cancelFunc != nil {
		s.cancelFunc()
	}

	// Close all client connections
	for client := range s.clients {
		client.Close()
	}
	s.clients = make(map[*websocket.Conn]bool)

	// Close pipes
	if s.stdin != nil {
		s.stdin.Close()
	}

	s.Status = "closed"
	return nil
}

// WriteInput writes input to the terminal
func (s *TerminalSession) WriteInput(input string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.Status != "running" {
		return fmt.Errorf("terminal session is not running")
	}

	if s.stdin == nil {
		return fmt.Errorf("stdin pipe is not available")
	}

	// Update last activity
	s.LastActivity = time.Now()

	// Add a newline if not present
	if len(input) > 0 && input[len(input)-1] != '\n' {
		input += "\n"
	}

	// Write to stdin
	_, err := s.stdin.Write([]byte(input))
	if err != nil {
		return fmt.Errorf("failed to write to stdin: %v", err)
	}

	// Add to history if it's a complete command (ends with newline)
	if len(input) > 0 && input[len(input)-1] == '\n' {
		s.history = append(s.history, TerminalHistoryEntry{
			Command:   input[:len(input)-1], // Remove the newline
			Timestamp: time.Now(),
			Output:    "", // Output will be captured by handleOutput
			ExitCode:  -1, // Not known yet
		})
	}

	return nil
}

// AddClient adds a WebSocket client to the session
func (s *TerminalSession) AddClient(conn *websocket.Conn) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.clients[conn] = true
	s.LastActivity = time.Now()

	// Send session info to the client
	conn.WriteJSON(map[string]interface{}{
		"type":        "session_info",
		"id":          s.ID,
		"status":      s.Status,
		"createdAt":   s.CreatedAt,
		"workingDir":  s.WorkingDir,
		"command":     s.Command,
		"exitCode":    s.ExitCode,
	})
}

// RemoveClient removes a WebSocket client from the session
func (s *TerminalSession) RemoveClient(conn *websocket.Conn) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	delete(s.clients, conn)
}

// GetHistory returns the command history for the session
func (s *TerminalSession) GetHistory() []TerminalHistoryEntry {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.history
}

// handleOutput reads from stdout or stderr and broadcasts to clients
func (s *TerminalSession) handleOutput(pipe io.ReadCloser, isStderr bool) {
	buffer := make([]byte, 4096)
	for {
		n, err := pipe.Read(buffer)
		if err != nil {
			if err != io.EOF {
				log.Printf("Error reading from pipe: %v", err)
			}
			break
		}

		if n > 0 {
			data := buffer[:n]
			s.mutex.Lock()
			
			// Update last activity
			s.LastActivity = time.Now()
			
			// Broadcast to clients
			s.broadcastToClients(map[string]interface{}{
				"type":    func() string { if isStderr { return "stderr" } else { return "stdout" } }(),
				"data":    string(data),
				"session": s.ID,
			})
			
			// Append to the last history entry's output if available
			if len(s.history) > 0 {
				lastIdx := len(s.history) - 1
				s.history[lastIdx].Output += string(data)
			}
			
			s.mutex.Unlock()
		}
	}
}

// broadcastToClients sends a message to all connected clients
func (s *TerminalSession) broadcastToClients(message map[string]interface{}) {
	for client := range s.clients {
		err := client.WriteJSON(message)
		if err != nil {
			log.Printf("Error sending message to client: %v", err)
			client.Close()
			delete(s.clients, client)
		}
	}
}

// TerminalSessionInfo represents public information about a terminal session
type TerminalSessionInfo struct {
	ID           string    `json:"id"`
	CreatedAt    time.Time `json:"createdAt"`
	LastActivity time.Time `json:"lastActivity"`
	Command      string    `json:"command"`
	WorkingDir   string    `json:"workingDir"`
	Status       string    `json:"status"`
	ExitCode     int       `json:"exitCode"`
	ClientCount  int       `json:"clientCount"`
}

// GetInfo returns public information about the terminal session
func (s *TerminalSession) GetInfo() TerminalSessionInfo {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return TerminalSessionInfo{
		ID:           s.ID,
		CreatedAt:    s.CreatedAt,
		LastActivity: s.LastActivity,
		Command:      s.Command,
		WorkingDir:   s.WorkingDir,
		Status:       s.Status,
		ExitCode:     s.ExitCode,
		ClientCount:  len(s.clients),
	}
}

// HandleTerminalWebSocket handles WebSocket connections for terminal sessions
func (m *TerminalManager) HandleTerminalWebSocket(w http.ResponseWriter, r *http.Request, sessionID string) {
	// Upgrade HTTP connection to WebSocket
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for now
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade to WebSocket: %v", err)
		return
	}
	defer conn.Close()

	// Get the terminal session
	session, err := m.GetSession(sessionID)
	if err != nil {
		conn.WriteJSON(map[string]interface{}{
			"type":  "error",
			"error": fmt.Sprintf("Terminal session not found: %s", sessionID),
		})
		return
	}

	// Add the client to the session
	session.AddClient(conn)
	defer session.RemoveClient(conn)

	// Start the session if it's not already running
	if session.Status == "created" {
		if err := session.Start(); err != nil {
			conn.WriteJSON(map[string]interface{}{
				"type":  "error",
				"error": fmt.Sprintf("Failed to start terminal session: %v", err),
			})
			return
		}
	}

	// Handle incoming messages
	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading WebSocket message: %v", err)
			break
		}

		// Handle text messages
		if messageType == websocket.TextMessage {
			var message map[string]interface{}
			if err := json.Unmarshal(p, &message); err != nil {
				log.Printf("Error parsing WebSocket message: %v", err)
				continue
			}

			// Handle different message types
			switch message["type"] {
			case "input":
				// Write input to the terminal
				if input, ok := message["data"].(string); ok {
					if err := session.WriteInput(input); err != nil {
						conn.WriteJSON(map[string]interface{}{
							"type":  "error",
							"error": fmt.Sprintf("Failed to write input: %v", err),
						})
					}
				}
			case "resize":
				// Handle terminal resize events
				// This would be implemented if using a PTY
			case "ping":
				// Respond to ping with pong
				conn.WriteJSON(map[string]interface{}{
					"type": "pong",
				})
			}
		}
	}
}

// RegisterHandlers registers the terminal API handlers
func (m *TerminalManager) RegisterHandlers(router *mux.Router) {
	router.HandleFunc("/api/v1/terminals", m.handleListTerminals).Methods("GET")
	router.HandleFunc("/api/v1/terminals", m.handleCreateTerminal).Methods("POST")
	router.HandleFunc("/api/v1/terminals/{id}", m.handleGetTerminal).Methods("GET")
	router.HandleFunc("/api/v1/terminals/{id}", m.handleCloseTerminal).Methods("DELETE")
	router.HandleFunc("/api/v1/terminals/{id}/history", m.handleGetTerminalHistory).Methods("GET")
	router.HandleFunc("/api/v1/terminals/{id}/ws", m.handleTerminalWebSocket)
}

// handleListTerminals handles GET /api/v1/terminals
func (m *TerminalManager) handleListTerminals(w http.ResponseWriter, r *http.Request) {
	// For simplicity, we'll use a fixed user ID
	userID := int64(1)

	sessions := m.ListSessions(userID)
	
	// Convert to public info
	sessionInfos := make([]TerminalSessionInfo, len(sessions))
	for i, session := range sessions {
		sessionInfos[i] = session.GetInfo()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"terminals": sessionInfos,
	})
}

// handleCreateTerminal handles POST /api/v1/terminals
func (m *TerminalManager) handleCreateTerminal(w http.ResponseWriter, r *http.Request) {
	var request struct {
		WorkingDir string   `json:"workingDir"`
		Command    string   `json:"command"`
		Env        []string `json:"env"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// For simplicity, we'll use a fixed user ID
	userID := int64(1)

	// Create the terminal session
	session, err := m.CreateSession(userID, request.WorkingDir, request.Env)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create terminal session: %v", err), http.StatusInternalServerError)
		return
	}

	// Set custom command if provided
	if request.Command != "" {
		session.Command = request.Command
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(session.GetInfo())
}

// handleGetTerminal handles GET /api/v1/terminals/{id}
func (m *TerminalManager) handleGetTerminal(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	session, err := m.GetSession(id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Terminal session not found: %v", err), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session.GetInfo())
}

// handleCloseTerminal handles DELETE /api/v1/terminals/{id}
func (m *TerminalManager) handleCloseTerminal(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if err := m.CloseSession(id); err != nil {
		http.Error(w, fmt.Sprintf("Failed to close terminal session: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleGetTerminalHistory handles GET /api/v1/terminals/{id}/history
func (m *TerminalManager) handleGetTerminalHistory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	session, err := m.GetSession(id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Terminal session not found: %v", err), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"history": session.GetHistory(),
	})
}

// handleTerminalWebSocket handles WebSocket connections for terminal sessions
func (m *TerminalManager) handleTerminalWebSocket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	m.HandleTerminalWebSocket(w, r, id)
}