package agentserver

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"nexus-backend/internal/config"
	"nexus-backend/internal/database"
)

// AgentServer represents the agent server service
type AgentServer struct {
	config *config.Config
	db     *database.BuntDBManager

	// Configuration
	agentDir      string
	maxAgents     int
	enableRuntime bool
	enableCORS    bool

	// Components
	runtimeManager *RuntimeManager

	// Server info
	serverInfo *ServerInfo
	startTime  time.Time
	running    bool
}

// ServerInfo represents information about this server instance
type ServerInfo struct {
	Name      string    `json:"name"`
	Port      int       `json:"port"`
	AgentDir  string    `json:"agent_dir"`
	StartTime time.Time `json:"start_time"`
	Version   string    `json:"version"`
}

// AgentInfo represents information about a plugin agent
type AgentInfo struct {
	Name         string    `json:"name"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"last_modified"`
	Hash         string    `json:"hash,omitempty"`
}

// UploadResponse represents the response from an upload operation
type UploadResponse struct {
	Success  bool   `json:"success"`
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
	Message  string `json:"message,omitempty"`
}

// ListResponse represents the response from a list operation
type ListResponse struct {
	Agents []AgentInfo `json:"agents"`
	Count  int         `json:"count"`
}

// NewAgentServer creates a new agent server instance
func NewAgentServer(config *config.Config, db *database.BuntDBManager) (*AgentServer, error) {
	// Set default values
	agentDir := "./agents"
	maxAgents := 10
	enableRuntime := true
	enableCORS := true

	// Create agent directory if it doesn't exist
	if err := ensureAgentDirectory(agentDir); err != nil {
		return nil, fmt.Errorf("failed to create agent directory: %w", err)
	}

	serverInfo := &ServerInfo{
		Name:      "KNIRV-NEXUS Plugin Agent Server",
		Port:      0, // Will be set by the main server
		AgentDir:  agentDir,
		StartTime: time.Now(),
		Version:   "2.0.0",
	}

	service := &AgentServer{
		config:        config,
		db:            db,
		agentDir:      agentDir,
		maxAgents:     maxAgents,
		enableRuntime: enableRuntime,
		enableCORS:    enableCORS,
		serverInfo:    serverInfo,
		startTime:     time.Now(),
		running:       false,
	}

	return service, nil
}

// Start starts the agent server service
func (as *AgentServer) Start() error {
	log.Println("Starting Agent Server service...")

	// Initialize runtime manager if enabled
	if as.enableRuntime {
		ctx := context.Background()
		var err error
		as.runtimeManager, err = NewRuntimeManager(ctx, as.agentDir, as.maxAgents)
		if err != nil {
			return fmt.Errorf("failed to create runtime manager: %w", err)
		}

		if err := as.runtimeManager.Start(); err != nil {
			return fmt.Errorf("failed to start runtime manager: %w", err)
		}

		log.Printf("Runtime manager started with max %d agents", as.maxAgents)
	}

	as.running = true
	log.Println("Agent Server service started successfully")
	return nil
}

// Stop stops the agent server service
func (as *AgentServer) Stop() error {
	log.Println("Stopping Agent Server service...")

	as.running = false

	// Stop runtime manager if it exists
	if as.runtimeManager != nil {
		if err := as.runtimeManager.Stop(); err != nil {
			log.Printf("Error stopping runtime manager: %v", err)
		}
	}

	log.Println("Agent Server service stopped")
	return nil
}

// IsRunning returns whether the service is running
func (as *AgentServer) IsRunning() bool {
	return as.running
}

// GetServerInfo returns server information
func (as *AgentServer) GetServerInfo() *ServerInfo {
	return as.serverInfo
}

// GetRuntimeManager returns the runtime manager (if enabled)
func (as *AgentServer) GetRuntimeManager() *RuntimeManager {
	return as.runtimeManager
}

// ensureAgentDirectory creates the agent directory if it doesn't exist
func ensureAgentDirectory(agentDir string) error {
	return os.MkdirAll(agentDir, 0755)
}
