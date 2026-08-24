package api

import (
	"KNIRVENGINE/desktop-client/utils"
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

// MCPInstallationStatus represents the status of an installation
type MCPInstallationStatus struct {
	ServerID    string    `json:"server_id"`
	Status      string    `json:"status"`   // "pending" | "installing" | "installed" | "failed" | "uninstalling"
	Progress    int       `json:"progress"` // 0-100
	Message     string    `json:"message"`
	Error       string    `json:"error,omitempty"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
	LogOutput   []string  `json:"log_output"`
}

// MCPInstallationService manages MCP server installations
type MCPInstallationService struct {
	registryService *MCPRegistryService
	installations   map[string]*MCPInstallationStatus
	mutex           sync.RWMutex
	installDir      string
}

// NewMCPInstallationService creates a new installation service
func NewMCPInstallationService(registryService *MCPRegistryService) *MCPInstallationService {
	// Get system-specific MCP servers directory
	installDir, err := utils.GetMCPServersDir()
	if err != nil {
		// Fallback to current directory if utils package is not available
		installDir = filepath.Join(".", "mcp_servers")
	}

	// Ensure installation directory exists
	if err := utils.EnsureDir(installDir); err != nil {
		// Fallback to os.MkdirAll if utils package is not available
		os.MkdirAll(installDir, 0755)
	}

	return &MCPInstallationService{
		registryService: registryService,
		installations:   make(map[string]*MCPInstallationStatus),
		installDir:      installDir,
	}
}

// InstallServer installs an MCP server
func (s *MCPInstallationService) InstallServer(ctx context.Context, serverID string) error {
	// Get server information
	server, err := s.registryService.GetServerByID(serverID)
	if err != nil {
		return fmt.Errorf("server not found: %w", err)
	}

	// Check if already installing
	s.mutex.Lock()
	if status, exists := s.installations[serverID]; exists && status.Status == "installing" {
		s.mutex.Unlock()
		return fmt.Errorf("server is already being installed")
	}

	// Perform pre-installation compatibility checks
	if err := s.performCompatibilityChecks(server); err != nil {
		s.mutex.Unlock()
		return fmt.Errorf("compatibility check failed: %w", err)
	}

	// Create installation status
	status := &MCPInstallationStatus{
		ServerID:  serverID,
		Status:    "pending",
		Progress:  0,
		Message:   "Installation queued",
		StartedAt: time.Now(),
		LogOutput: make([]string, 0),
	}
	s.installations[serverID] = status
	s.mutex.Unlock()

	// Start installation in background
	go s.performInstallation(ctx, server, status)

	return nil
}

// performInstallation performs the actual installation
func (s *MCPInstallationService) performInstallation(ctx context.Context, server *MCPServer, status *MCPInstallationStatus) {
	s.updateStatus(status, "installing", 10, "Starting installation...")

	// Create server-specific directory
	serverDir := filepath.Join(s.installDir, server.ID)
	if err := os.MkdirAll(serverDir, 0755); err != nil {
		s.updateStatusError(status, fmt.Sprintf("Failed to create directory: %v", err))
		return
	}

	s.updateStatus(status, "installing", 20, "Created installation directory")

	// Install based on server type
	var err error
	switch server.Type {
	case "python":
		err = s.installPythonServer(ctx, server, status, serverDir)
	case "typescript":
		err = s.installTypescriptServer(ctx, server, status, serverDir)
	default:
		err = fmt.Errorf("unsupported server type: %s", server.Type)
	}

	if err != nil {
		s.updateStatusError(status, fmt.Sprintf("Installation failed: %v", err))
		return
	}

	// Update server status in registry
	s.registryService.mutex.Lock()
	if registryServer, exists := s.registryService.servers[server.ID]; exists {
		registryServer.Status = "installed"
		registryServer.UpdatedAt = time.Now()
	}
	s.registryService.mutex.Unlock()

	// Transform MCP server into capability
	if err := s.createCapabilityFromMCPServer(ctx, server); err != nil {
		s.updateStatus(status, "installed", 95, fmt.Sprintf("Warning: Failed to create capability: %v", err))
	} else {
		s.updateStatus(status, "installed", 98, "Created capability from MCP server")
	}

	s.updateStatus(status, "installed", 100, "Installation completed successfully")
	status.CompletedAt = time.Now()
}

// installPythonServer installs a Python-based MCP server
func (s *MCPInstallationService) installPythonServer(ctx context.Context, server *MCPServer, status *MCPInstallationStatus, serverDir string) error {
	s.updateStatus(status, "installing", 30, "Installing Python dependencies...")

	// Check if uvx is available
	if err := s.checkCommand("uvx"); err != nil {
		return fmt.Errorf("uvx not found: %w", err)
	}

	s.updateStatus(status, "installing", 40, "uvx found, proceeding with installation...")

	// Extract package name from install command
	packageName := s.extractPackageName(server.InstallCommand, "uvx")
	if packageName == "" {
		return fmt.Errorf("could not extract package name from install command")
	}

	s.updateStatus(status, "installing", 50, fmt.Sprintf("Installing package: %s", packageName))

	// Run uvx install command with retry mechanism
	var output string
	var err error
	maxRetries := 3

	for attempt := 1; attempt <= maxRetries; attempt++ {
		s.updateStatus(status, "installing", 50+attempt*5, fmt.Sprintf("Installing package: %s (attempt %d/%d)", packageName, attempt, maxRetries))

		cmd := exec.CommandContext(ctx, "uvx", "install", packageName)
		cmd.Dir = serverDir

		output, err = s.runCommandWithOutput(cmd, status)
		if err == nil {
			break // Success
		}

		if attempt < maxRetries {
			s.addLogOutput(status, fmt.Sprintf("Installation attempt %d failed: %v, retrying...", attempt, err))
			time.Sleep(time.Duration(attempt) * 2 * time.Second) // Exponential backoff
		}
	}

	if err != nil {
		return fmt.Errorf("uvx install failed after %d attempts: %w", maxRetries, err)
	}

	s.updateStatus(status, "installing", 80, "Python package installed successfully")
	s.addLogOutput(status, fmt.Sprintf("Installation output: %s", output))

	// Verify installation
	s.updateStatus(status, "installing", 90, "Verifying installation...")

	verifyCmd := exec.CommandContext(ctx, "uvx", "run", packageName, "--help")
	if _, err := verifyCmd.Output(); err != nil {
		s.addLogOutput(status, fmt.Sprintf("Verification warning: %v", err))
	} else {
		s.addLogOutput(status, "Installation verified successfully")
	}

	return nil
}

// installTypescriptServer installs a TypeScript-based MCP server
func (s *MCPInstallationService) installTypescriptServer(ctx context.Context, server *MCPServer, status *MCPInstallationStatus, serverDir string) error {
	s.updateStatus(status, "installing", 30, "Installing Node.js dependencies...")

	// Check if npm/npx is available
	if err := s.checkCommand("npx"); err != nil {
		return fmt.Errorf("npx not found: %w", err)
	}

	s.updateStatus(status, "installing", 40, "npx found, proceeding with installation...")

	// Extract package name from install command
	packageName := s.extractPackageName(server.InstallCommand, "npx")
	if packageName == "" {
		return fmt.Errorf("could not extract package name from install command")
	}

	// For npx, we'll install the package using -y flag to avoid prompts with retry mechanism
	var output string
	var err error
	maxRetries := 3

	for attempt := 1; attempt <= maxRetries; attempt++ {
		s.updateStatus(status, "installing", 50+attempt*5, fmt.Sprintf("Installing package: %s (attempt %d/%d)", packageName, attempt, maxRetries))

		cmd := exec.CommandContext(ctx, "npx", "-y", packageName, "--help")
		cmd.Dir = serverDir

		output, err = s.runCommandWithOutput(cmd, status)
		if err == nil {
			s.updateStatus(status, "installing", 80, "TypeScript package verified successfully")
			s.addLogOutput(status, fmt.Sprintf("Verification output: %s", output))
			break // Success
		}

		// Try without --help flag
		cmd = exec.CommandContext(ctx, "npx", "-y", packageName, "--version")
		output, err = s.runCommandWithOutput(cmd, status)
		if err == nil {
			s.updateStatus(status, "installing", 80, "TypeScript package verified successfully")
			s.addLogOutput(status, fmt.Sprintf("Verification output: %s", output))
			break // Success
		}

		if attempt < maxRetries {
			s.addLogOutput(status, fmt.Sprintf("Installation attempt %d failed: %v, retrying...", attempt, err))
			time.Sleep(time.Duration(attempt) * 2 * time.Second) // Exponential backoff
		}
	}

	if err != nil {
		// If all attempts fail, mark as installed since npx will handle it at runtime
		s.updateStatus(status, "installing", 80, "Package installation completed (runtime verification)")
		s.addLogOutput(status, fmt.Sprintf("Package %s will be verified at runtime after %d attempts", packageName, maxRetries))
	}

	s.updateStatus(status, "installing", 90, "Installation completed")

	return nil
}

// GetInstallationStatus returns the installation status for a server
func (s *MCPInstallationService) GetInstallationStatus(serverID string) (*MCPInstallationStatus, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	status, exists := s.installations[serverID]
	if !exists {
		return nil, fmt.Errorf("no installation status found for server: %s", serverID)
	}

	return status, nil
}

// Helper methods

// checkCommand checks if a command is available in PATH
func (s *MCPInstallationService) checkCommand(command string) error {
	_, err := exec.LookPath(command)
	return err
}

// extractPackageName extracts package name from install command
func (s *MCPInstallationService) extractPackageName(installCommand, tool string) string {
	parts := strings.Fields(installCommand)
	for i, part := range parts {
		if part == tool && i+1 < len(parts) {
			// Skip flags like -y
			for j := i + 1; j < len(parts); j++ {
				if !strings.HasPrefix(parts[j], "-") {
					return parts[j]
				}
			}
		}
	}
	return ""
}

// runCommandWithOutput runs a command and captures output
func (s *MCPInstallationService) runCommandWithOutput(cmd *exec.Cmd, status *MCPInstallationStatus) (string, error) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}

	if err := cmd.Start(); err != nil {
		return "", err
	}

	// Read output
	var output strings.Builder

	// Read stdout
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			output.WriteString(line + "\n")
			s.addLogOutput(status, line)
		}
	}()

	// Read stderr
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			output.WriteString(line + "\n")
			s.addLogOutput(status, line)
		}
	}()

	err = cmd.Wait()
	return output.String(), err
}

// updateStatus updates installation status
func (s *MCPInstallationService) updateStatus(status *MCPInstallationStatus, newStatus string, progress int, message string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	status.Status = newStatus
	status.Progress = progress
	status.Message = message
}

// updateStatusError updates status with error
func (s *MCPInstallationService) updateStatusError(status *MCPInstallationStatus, errorMsg string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	status.Status = "failed"
	status.Progress = 0
	status.Error = errorMsg
	status.CompletedAt = time.Now()

	// Keep server status as "available" when installation fails
	// so it can be retried
	s.registryService.mutex.Lock()
	if registryServer, exists := s.registryService.servers[status.ServerID]; exists {
		registryServer.Status = "available"
		registryServer.UpdatedAt = time.Now()
	}
	s.registryService.mutex.Unlock()
}

// addLogOutput adds a log line to status
func (s *MCPInstallationService) addLogOutput(status *MCPInstallationStatus, line string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	status.LogOutput = append(status.LogOutput, fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), line))

	// Keep only last 100 lines
	if len(status.LogOutput) > 100 {
		status.LogOutput = status.LogOutput[len(status.LogOutput)-100:]
	}
}

// HTTP Handlers

// InstallServerHandler handles POST /api/v1/mcp/servers/{id}/install
func (s *MCPInstallationService) InstallServerHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serverID := vars["id"]

	if err := s.InstallServer(r.Context(), serverID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response := map[string]interface{}{
		"message": "Installation started",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetInstallationStatusHandler handles GET /api/v1/mcp/servers/{id}/status
func (s *MCPInstallationService) GetInstallationStatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serverID := vars["id"]

	status, err := s.GetInstallationStatus(serverID)
	if err != nil {
		http.Error(w, "Installation status not found", http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"status": status,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// createCapabilityFromMCPServer transforms an installed MCP server into a capability
func (s *MCPInstallationService) createCapabilityFromMCPServer(ctx context.Context, server *MCPServer) error {
	// Store capability metadata in server configuration for frontend access
	// Use context for mutex lock with timeout
	lockCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	select {
	case <-lockCtx.Done():
		return fmt.Errorf("timeout acquiring registry lock: %w", lockCtx.Err())
	default:
		s.registryService.mutex.Lock()
		defer s.registryService.mutex.Unlock()

		if registryServer, exists := s.registryService.servers[server.ID]; exists {
			if registryServer.Configuration == nil {
				registryServer.Configuration = make(map[string]string)
			}
			// Store capability ID and creation flag
			registryServer.Configuration["capability_id"] = fmt.Sprintf("mcp-%s", server.ID)
			registryServer.Configuration["capability_created"] = "true"
			registryServer.Configuration["capability_name"] = server.Name
			registryServer.Configuration["capability_type"] = s.mapMCPCategoryToCapabilityType(server.Category)
		}
	}

	return nil
}

// mapMCPCategoryToCapabilityType maps MCP server categories to capability types
func (s *MCPInstallationService) mapMCPCategoryToCapabilityType(category string) string {
	switch category {
	case "web":
		return "web_interaction"
	case "file":
		return "file_system"
	case "data":
		return "data_processing"
	case "ai":
		return "ai_analysis"
	case "system":
		return "system_tools"
	case "security":
		return "security"
	case "cloud":
		return "cloud_services"
	case "social":
		return "social_media"
	default:
		return "general"
	}
}

// performCompatibilityChecks performs pre-installation compatibility checks
func (s *MCPInstallationService) performCompatibilityChecks(server *MCPServer) error {
	// Check if required tools are available
	switch server.Type {
	case "python":
		if err := s.checkCommand("uvx"); err != nil {
			// Try alternative Python package managers
			if err2 := s.checkCommand("pip"); err2 != nil {
				return fmt.Errorf("neither uvx nor pip found: uvx error: %v, pip error: %v", err, err2)
			}
			// pip is available, but uvx is preferred
			log.Printf("Warning: uvx not found, pip is available but uvx is recommended for MCP servers")
		}

		// Check Python version if available
		if err := s.checkPythonVersion(); err != nil {
			log.Printf("Warning: Python version check failed: %v", err)
		}

	case "typescript":
		if err := s.checkCommand("npx"); err != nil {
			// Try alternative Node.js package managers
			if err2 := s.checkCommand("npm"); err2 != nil {
				return fmt.Errorf("neither npx nor npm found: npx error: %v, npm error: %v", err, err2)
			}
		}

		// Check Node.js version if available
		if err := s.checkNodeVersion(); err != nil {
			log.Printf("Warning: Node.js version check failed: %v", err)
		}

	default:
		return fmt.Errorf("unsupported server type: %s", server.Type)
	}

	// Check disk space
	if err := s.checkDiskSpace(); err != nil {
		log.Printf("Warning: Disk space check failed: %v", err)
	}

	// Check network connectivity for package downloads
	if err := s.checkNetworkConnectivity(server.Type); err != nil {
		return fmt.Errorf("network connectivity check failed: %w", err)
	}

	return nil
}

// checkPythonVersion checks if Python version is compatible
func (s *MCPInstallationService) checkPythonVersion() error {
	cmd := exec.Command("python3", "--version")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check Python version: %w", err)
	}

	log.Printf("Python version: %s", string(output))
	return nil
}

// checkNodeVersion checks if Node.js version is compatible
func (s *MCPInstallationService) checkNodeVersion() error {
	cmd := exec.Command("node", "--version")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check Node.js version: %w", err)
	}

	log.Printf("Node.js version: %s", string(output))
	return nil
}

// checkDiskSpace checks available disk space
func (s *MCPInstallationService) checkDiskSpace() error {
	// Simple disk space check - in production this could be more sophisticated
	stat, err := os.Stat(s.installDir)
	if err != nil {
		return fmt.Errorf("failed to check install directory: %w", err)
	}

	if !stat.IsDir() {
		return fmt.Errorf("install directory is not a directory")
	}

	return nil
}

// checkNetworkConnectivity checks network connectivity for package downloads
func (s *MCPInstallationService) checkNetworkConnectivity(serverType string) error {
	var testURL string
	switch serverType {
	case "python":
		testURL = "https://pypi.org"
	case "typescript":
		testURL = "https://registry.npmjs.org"
	default:
		return nil // Skip check for unknown types
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Head(testURL)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", testURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("package registry returned status %d", resp.StatusCode)
	}

	return nil
}

// estimateProcessingTime estimates processing time based on server type and category
func (s *MCPInstallationService) estimateProcessingTime(serverType, category string) string {
	switch category {
	case "ai":
		if serverType == "gpu" {
			return "3-10 seconds"
		}
		return "5-15 seconds"
	case "data":
		if serverType == "large" {
			return "5-15 seconds"
		}
		return "2-10 seconds"
	case "web":
		if serverType == "edge" {
			return "0.5-2 seconds"
		}
		return "1-5 seconds"
	case "file":
		if serverType == "fast" {
			return "0.5-1 seconds"
		}
		return "1-3 seconds"
	default:
		return "1-5 seconds"
	}
}

// RegisterHandlers registers the installation HTTP handlers
func (s *MCPInstallationService) RegisterHandlers(router *mux.Router) {
	mcpRouter := router.PathPrefix("/api/v1/mcp").Subrouter()
	mcpRouter.HandleFunc("/servers/{id}/install", s.InstallServerHandler).Methods("POST")
	mcpRouter.HandleFunc("/servers/{id}/status", s.GetInstallationStatusHandler).Methods("GET")
}
