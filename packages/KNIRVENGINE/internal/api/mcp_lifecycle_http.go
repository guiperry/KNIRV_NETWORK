package api

import (
	"KNIRVENGINE/desktop-client/internal/utils"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

// MCPServerProcess represents a running MCP server process
type MCPServerProcess struct {
	ServerID     string            `json:"server_id"`
	PID          int               `json:"pid"`
	Port         int               `json:"port"`
	Status       string            `json:"status"` // "starting" | "running" | "stopping" | "stopped" | "error"
	StartedAt    time.Time         `json:"started_at"`
	StoppedAt    time.Time         `json:"stopped_at,omitempty"`
	Command      string            `json:"command"`
	Arguments    []string          `json:"arguments"`
	Environment  map[string]string `json:"environment"`
	LogFile      string            `json:"log_file"`
	ErrorMsg     string            `json:"error_msg,omitempty"`
	HealthCheck  time.Time         `json:"last_health_check"`
	RestartCount int               `json:"restart_count"`
}

// MCPLifecycleService manages MCP server processes
type MCPLifecycleService struct {
	registryService     *MCPRegistryService
	installationService *MCPInstallationService
	processes           map[string]*MCPServerProcess
	mutex               sync.RWMutex
	portRange           struct {
		start int
		end   int
	}
	logDir string
}

// NewMCPLifecycleService creates a new lifecycle service
func NewMCPLifecycleService(registryService *MCPRegistryService, installationService *MCPInstallationService) *MCPLifecycleService {
	// Create logs directory in app data directory
	logDir := "./mcp_logs" // Default fallback
	if appLogDir, err := utils.GetMCPLogsDir(); err == nil {
		logDir = appLogDir
	}
	os.MkdirAll(logDir, 0755)

	service := &MCPLifecycleService{
		registryService:     registryService,
		installationService: installationService,
		processes:           make(map[string]*MCPServerProcess),
		logDir:              logDir,
	}

	// Set port range for MCP servers
	service.portRange.start = 8100
	service.portRange.end = 8200

	return service
}

// Start begins the lifecycle service with health monitoring
func (s *MCPLifecycleService) Start(ctx context.Context) error {
	// Start health monitoring
	go s.healthMonitor(ctx)
	return nil
}

// StartServer starts an MCP server
func (s *MCPLifecycleService) StartServer(ctx context.Context, serverID string, config map[string]string) error {
	// Get server information
	server, err := s.registryService.GetServerByID(serverID)
	if err != nil {
		return fmt.Errorf("server not found: %w", err)
	}

	// Check if server is installed
	if server.Status != "installed" {
		return fmt.Errorf("server is not installed")
	}

	// Check if already running
	s.mutex.Lock()
	if process, exists := s.processes[serverID]; exists && process.Status == "running" {
		s.mutex.Unlock()
		return fmt.Errorf("server is already running")
	}

	// Create process record
	process := &MCPServerProcess{
		ServerID:    serverID,
		Status:      "starting",
		StartedAt:   time.Now(),
		Environment: make(map[string]string),
		LogFile:     filepath.Join(s.logDir, fmt.Sprintf("%s.log", serverID)),
	}

	// Apply configuration
	for key, value := range config {
		process.Environment[key] = value
	}

	s.processes[serverID] = process
	s.mutex.Unlock()

	// Start process in background
	go s.startServerProcess(ctx, server, process)

	return nil
}

// startServerProcess starts the actual server process
func (s *MCPLifecycleService) startServerProcess(ctx context.Context, server *MCPServer, process *MCPServerProcess) {
	// Prepare command and arguments
	command, args := s.prepareCommand(server)
	process.Command = command
	process.Arguments = args

	// Create log file
	logFile, err := os.Create(process.LogFile)
	if err != nil {
		s.updateProcessError(process, fmt.Sprintf("Failed to create log file: %v", err))
		return
	}
	defer logFile.Close()

	// Create command
	cmd := exec.CommandContext(ctx, command, args...)

	// Set environment variables
	cmd.Env = os.Environ()
	for key, value := range process.Environment {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Set working directory
	serverDir := filepath.Join(s.installationService.installDir, server.ID)
	cmd.Dir = serverDir

	// Redirect output to log file
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	// Start the process
	if err := cmd.Start(); err != nil {
		s.updateProcessError(process, fmt.Sprintf("Failed to start process: %v", err))
		return
	}

	// Update process with PID
	s.mutex.Lock()
	process.PID = cmd.Process.Pid
	process.Status = "running"
	s.mutex.Unlock()

	// Update server status in registry
	s.registryService.mutex.Lock()
	if registryServer, exists := s.registryService.servers[server.ID]; exists {
		registryServer.Status = "running"
		registryServer.UpdatedAt = time.Now()
	}
	s.registryService.mutex.Unlock()

	// Wait for process to complete
	err = cmd.Wait()

	s.mutex.Lock()
	process.Status = "stopped"
	process.StoppedAt = time.Now()
	if err != nil {
		process.ErrorMsg = err.Error()
	}
	s.mutex.Unlock()

	// Update server status in registry
	s.registryService.mutex.Lock()
	if registryServer, exists := s.registryService.servers[server.ID]; exists {
		registryServer.Status = "installed"
		registryServer.UpdatedAt = time.Now()
	}
	s.registryService.mutex.Unlock()
}

// StopServer stops an MCP server
func (s *MCPLifecycleService) StopServer(serverID string) error {
	s.mutex.Lock()
	process, exists := s.processes[serverID]
	if !exists || process.Status != "running" {
		s.mutex.Unlock()
		return fmt.Errorf("server is not running")
	}

	process.Status = "stopping"
	s.mutex.Unlock()

	// Send termination signal to process
	if process.PID > 0 {
		if err := terminateProcess(process.PID, false); err != nil {
			return fmt.Errorf("failed to stop process: %w", err)
		}
	}

	// Wait for graceful shutdown (up to 10 seconds)
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			// Force kill if still running
			if process.PID > 0 {
				terminateProcess(process.PID, true)
			}
			s.mutex.Lock()
			process.Status = "stopped"
			process.StoppedAt = time.Now()
			s.mutex.Unlock()
			return nil
		case <-ticker.C:
			s.mutex.RLock()
			if process.Status == "stopped" {
				s.mutex.RUnlock()
				return nil
			}
			s.mutex.RUnlock()
		}
	}
}

// GetServerProcess returns process information for a server
func (s *MCPLifecycleService) GetServerProcess(serverID string) (*MCPServerProcess, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	process, exists := s.processes[serverID]
	if !exists {
		return nil, fmt.Errorf("no process found for server: %s", serverID)
	}

	return process, nil
}

// GetRunningServers returns all running servers
func (s *MCPLifecycleService) GetRunningServers() []*MCPServerProcess {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var running []*MCPServerProcess
	for _, process := range s.processes {
		if process.Status == "running" {
			running = append(running, process)
		}
	}

	return running
}

// Helper methods

// prepareCommand prepares the command and arguments for starting a server
func (s *MCPLifecycleService) prepareCommand(server *MCPServer) (string, []string) {
	parts := strings.Fields(server.RunCommand)
	if len(parts) == 0 {
		return "", nil
	}

	command := parts[0]
	args := parts[1:]

	// Add stdio transport (default for MCP)
	if server.Type == "typescript" || server.Type == "python" {
		args = append(args, "--transport", "stdio")
	}

	return command, args
}

// updateProcessError updates process with error status
func (s *MCPLifecycleService) updateProcessError(process *MCPServerProcess, errorMsg string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	process.Status = "error"
	process.ErrorMsg = errorMsg
	process.StoppedAt = time.Now()
}

// healthMonitor monitors server health
func (s *MCPLifecycleService) healthMonitor(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.performHealthChecks()
		}
	}
}

// performHealthChecks checks health of all running servers
func (s *MCPLifecycleService) performHealthChecks() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for serverID, process := range s.processes {
		if process.Status == "running" {
			// Check if process is still alive
			if process.PID > 0 {
				if !isProcessAlive(process.PID) {
					// Process is dead
					process.Status = "stopped"
					process.StoppedAt = time.Now()
					process.ErrorMsg = "Process died unexpectedly"

					// Update registry
					if registryServer, exists := s.registryService.servers[serverID]; exists {
						registryServer.Status = "installed"
						registryServer.UpdatedAt = time.Now()
					}
				} else {
					// Process is alive, update health check time
					process.HealthCheck = time.Now()
				}
			}
		}
	}
}

// HTTP Handlers

// StartServerHandler handles POST /api/v1/mcp/servers/{id}/start
func (s *MCPLifecycleService) StartServerHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serverID := vars["id"]

	var config map[string]string
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		config = make(map[string]string)
	}

	if err := s.StartServer(r.Context(), serverID, config); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response := map[string]interface{}{
		"message": "Server start initiated",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// StopServerHandler handles POST /api/v1/mcp/servers/{id}/stop
func (s *MCPLifecycleService) StopServerHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serverID := vars["id"]

	if err := s.StopServer(serverID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response := map[string]interface{}{
		"message": "Server stop initiated",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetRunningServersHandler handles GET /api/v1/mcp/servers/running
func (s *MCPLifecycleService) GetRunningServersHandler(w http.ResponseWriter, r *http.Request) {
	running := s.GetRunningServers()

	response := map[string]interface{}{
		"servers": running,
		"count":   len(running),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// RegisterHandlers registers the lifecycle HTTP handlers
func (s *MCPLifecycleService) RegisterHandlers(router *mux.Router) {
	mcpRouter := router.PathPrefix("/api/v1/mcp").Subrouter()
	mcpRouter.HandleFunc("/servers/{id}/start", s.StartServerHandler).Methods("POST")
	mcpRouter.HandleFunc("/servers/{id}/stop", s.StopServerHandler).Methods("POST")
	mcpRouter.HandleFunc("/servers/running", s.GetRunningServersHandler).Methods("GET")
}
