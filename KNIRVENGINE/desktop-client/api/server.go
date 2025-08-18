package api

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// Server represents the API server
type Server struct {
	router          *mux.Router
	terminalManager *TerminalManager
	agentTerminal   *AgentTerminalHandler
}

// NewServer creates a new API server
func NewServer() *Server {
	router := mux.NewRouter()
	terminalManager := NewTerminalManager()
	agentTerminal := NewAgentTerminalHandler(terminalManager)

	server := &Server{
		router:          router,
		terminalManager: terminalManager,
		agentTerminal:   agentTerminal,
	}

	// Register routes
	server.registerRoutes()

	// Start terminal session cleanup
	go server.cleanupExpiredSessions()

	return server
}

// registerRoutes registers all API routes
func (s *Server) registerRoutes() {
	// Register terminal routes
	s.agentTerminal.RegisterRoutes(s.router)

	// Add other API routes here
	// ...
}

// cleanupExpiredSessions periodically cleans up expired terminal sessions
func (s *Server) cleanupExpiredSessions() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.terminalManager.CleanupExpiredSessions()
	}
}

// Start starts the API server
func (s *Server) Start(addr string) error {
	log.Printf("Starting API server on %s", addr)
	return http.ListenAndServe(addr, s.router)
}

// RegisterAgentPlugin registers an agent plugin with the terminal handler
func (s *Server) RegisterAgentPlugin(agentID, pluginPath string) {
	s.agentTerminal.RegisterAgentPlugin(agentID, pluginPath)
}

// GetRouter returns the router for testing
func (s *Server) GetRouter() *mux.Router {
	return s.router
}