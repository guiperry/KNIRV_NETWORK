package api

import (
	"Agentic_Engine/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/gorilla/mux"
)

// MCPServerConfig represents configuration for an MCP server
type MCPServerConfig struct {
	ServerID      string            `json:"server_id"`
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	Environment   map[string]string `json:"environment"`
	Arguments     []string          `json:"arguments"`
	WorkingDir    string            `json:"working_dir"`
	AutoStart     bool              `json:"auto_start"`
	RestartOnFail bool              `json:"restart_on_fail"`
	MaxRestarts   int               `json:"max_restarts"`
	HealthCheck   struct {
		Enabled  bool `json:"enabled"`
		Interval int  `json:"interval"` // seconds
		Timeout  int  `json:"timeout"`  // seconds
	} `json:"health_check"`
	Resources struct {
		MaxMemory int `json:"max_memory"` // MB
		MaxCPU    int `json:"max_cpu"`    // percentage
	} `json:"resources"`
	Security struct {
		Sandboxed    bool     `json:"sandboxed"`
		AllowedPaths []string `json:"allowed_paths"`
		AllowedHosts []string `json:"allowed_hosts"`
		AllowedPorts []int    `json:"allowed_ports"`
		ReadOnlyMode bool     `json:"read_only_mode"`
	} `json:"security"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// MCPConfigService manages MCP server configurations
type MCPConfigService struct {
	registryService *MCPRegistryService
	configs         map[string]*MCPServerConfig
	mutex           sync.RWMutex
	configDir       string
}

// NewMCPConfigService creates a new configuration service
func NewMCPConfigService(registryService *MCPRegistryService) *MCPConfigService {
	// Get system-specific MCP config directory
	configDir, err := utils.GetMCPConfigDir()
	if err != nil {
		// Fallback to current directory if utils package is not available
		configDir = filepath.Join(".", "mcp_configs")
	}

	// Ensure config directory exists
	if err := utils.EnsureDir(configDir); err != nil {
		// Fallback to os.MkdirAll if utils package is not available
		os.MkdirAll(configDir, 0755)
	}

	service := &MCPConfigService{
		registryService: registryService,
		configs:         make(map[string]*MCPServerConfig),
		configDir:       configDir,
	}

	// Load existing configurations
	service.loadConfigurations()

	return service
}

// loadConfigurations loads configurations from disk
func (s *MCPConfigService) loadConfigurations() {
	files, err := filepath.Glob(filepath.Join(s.configDir, "*.json"))
	if err != nil {
		return
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		var config MCPServerConfig
		if err := json.Unmarshal(data, &config); err != nil {
			continue
		}

		s.configs[config.ServerID] = &config
	}
}

// saveConfiguration saves a configuration to disk
func (s *MCPConfigService) saveConfiguration(config *MCPServerConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	filename := filepath.Join(s.configDir, fmt.Sprintf("%s.json", config.ServerID))
	return os.WriteFile(filename, data, 0644)
}

// GetServerConfig returns configuration for a server
func (s *MCPConfigService) GetServerConfig(serverID string) (*MCPServerConfig, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	config, exists := s.configs[serverID]
	if !exists {
		// Create default configuration
		server, err := s.registryService.GetServerByID(serverID)
		if err != nil {
			return nil, fmt.Errorf("server not found: %w", err)
		}

		config = s.createDefaultConfig(server)
		s.configs[serverID] = config
	}

	return config, nil
}

// UpdateServerConfig updates configuration for a server
func (s *MCPConfigService) UpdateServerConfig(serverID string, updates map[string]interface{}) (*MCPServerConfig, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	config, exists := s.configs[serverID]
	if !exists {
		// Create default configuration first
		server, err := s.registryService.GetServerByID(serverID)
		if err != nil {
			return nil, fmt.Errorf("server not found: %w", err)
		}
		config = s.createDefaultConfig(server)
	}

	// Apply updates
	if name, ok := updates["name"].(string); ok {
		config.Name = name
	}
	if description, ok := updates["description"].(string); ok {
		config.Description = description
	}
	if env, ok := updates["environment"].(map[string]interface{}); ok {
		config.Environment = make(map[string]string)
		for k, v := range env {
			if str, ok := v.(string); ok {
				config.Environment[k] = str
			}
		}
	}
	if args, ok := updates["arguments"].([]interface{}); ok {
		config.Arguments = make([]string, 0, len(args))
		for _, arg := range args {
			if str, ok := arg.(string); ok {
				config.Arguments = append(config.Arguments, str)
			}
		}
	}
	if workingDir, ok := updates["working_dir"].(string); ok {
		config.WorkingDir = workingDir
	}
	if autoStart, ok := updates["auto_start"].(bool); ok {
		config.AutoStart = autoStart
	}
	if restartOnFail, ok := updates["restart_on_fail"].(bool); ok {
		config.RestartOnFail = restartOnFail
	}
	if maxRestarts, ok := updates["max_restarts"].(float64); ok {
		config.MaxRestarts = int(maxRestarts)
	}

	// Update health check settings
	if healthCheck, ok := updates["health_check"].(map[string]interface{}); ok {
		if enabled, ok := healthCheck["enabled"].(bool); ok {
			config.HealthCheck.Enabled = enabled
		}
		if interval, ok := healthCheck["interval"].(float64); ok {
			config.HealthCheck.Interval = int(interval)
		}
		if timeout, ok := healthCheck["timeout"].(float64); ok {
			config.HealthCheck.Timeout = int(timeout)
		}
	}

	// Update resource settings
	if resources, ok := updates["resources"].(map[string]interface{}); ok {
		if maxMemory, ok := resources["max_memory"].(float64); ok {
			config.Resources.MaxMemory = int(maxMemory)
		}
		if maxCPU, ok := resources["max_cpu"].(float64); ok {
			config.Resources.MaxCPU = int(maxCPU)
		}
	}

	// Update security settings
	if security, ok := updates["security"].(map[string]interface{}); ok {
		if sandboxed, ok := security["sandboxed"].(bool); ok {
			config.Security.Sandboxed = sandboxed
		}
		if allowedPaths, ok := security["allowed_paths"].([]interface{}); ok {
			config.Security.AllowedPaths = make([]string, 0, len(allowedPaths))
			for _, path := range allowedPaths {
				if str, ok := path.(string); ok {
					config.Security.AllowedPaths = append(config.Security.AllowedPaths, str)
				}
			}
		}
		if allowedHosts, ok := security["allowed_hosts"].([]interface{}); ok {
			config.Security.AllowedHosts = make([]string, 0, len(allowedHosts))
			for _, host := range allowedHosts {
				if str, ok := host.(string); ok {
					config.Security.AllowedHosts = append(config.Security.AllowedHosts, str)
				}
			}
		}
		if allowedPorts, ok := security["allowed_ports"].([]interface{}); ok {
			config.Security.AllowedPorts = make([]int, 0, len(allowedPorts))
			for _, port := range allowedPorts {
				if num, ok := port.(float64); ok {
					config.Security.AllowedPorts = append(config.Security.AllowedPorts, int(num))
				}
			}
		}
		if readOnlyMode, ok := security["read_only_mode"].(bool); ok {
			config.Security.ReadOnlyMode = readOnlyMode
		}
	}

	config.UpdatedAt = fmt.Sprintf("%d", getCurrentTimestamp())
	s.configs[serverID] = config

	// Save to disk
	if err := s.saveConfiguration(config); err != nil {
		return nil, fmt.Errorf("failed to save configuration: %w", err)
	}

	return config, nil
}

// DeleteServerConfig deletes configuration for a server
func (s *MCPConfigService) DeleteServerConfig(serverID string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	delete(s.configs, serverID)

	// Remove from disk
	filename := filepath.Join(s.configDir, fmt.Sprintf("%s.json", serverID))
	if err := os.Remove(filename); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete configuration file: %w", err)
	}

	return nil
}

// GetAllConfigs returns all server configurations
func (s *MCPConfigService) GetAllConfigs() map[string]*MCPServerConfig {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Create a copy to avoid race conditions
	result := make(map[string]*MCPServerConfig)
	for k, v := range s.configs {
		result[k] = v
	}

	return result
}

// createDefaultConfig creates a default configuration for a server
func (s *MCPConfigService) createDefaultConfig(server *MCPServer) *MCPServerConfig {
	timestamp := fmt.Sprintf("%d", getCurrentTimestamp())

	config := &MCPServerConfig{
		ServerID:      server.ID,
		Name:          server.Name,
		Description:   server.Description,
		Environment:   make(map[string]string),
		Arguments:     []string{},
		WorkingDir:    "",
		AutoStart:     false,
		RestartOnFail: true,
		MaxRestarts:   3,
		CreatedAt:     timestamp,
		UpdatedAt:     timestamp,
	}

	// Set default health check settings
	config.HealthCheck.Enabled = true
	config.HealthCheck.Interval = 30
	config.HealthCheck.Timeout = 10

	// Set default resource limits
	config.Resources.MaxMemory = 512 // 512 MB
	config.Resources.MaxCPU = 50     // 50%

	// Set default security settings
	config.Security.Sandboxed = true
	config.Security.AllowedPaths = []string{"/tmp", "/var/tmp"}
	config.Security.AllowedHosts = []string{"localhost", "127.0.0.1"}
	config.Security.AllowedPorts = []int{80, 443, 8080}
	config.Security.ReadOnlyMode = false

	return config
}

// getCurrentTimestamp returns current Unix timestamp
func getCurrentTimestamp() int64 {
	return 1640995200 // Placeholder timestamp
}

// HTTP Handlers

// GetServerConfigHandler handles GET /api/v1/mcp/servers/{id}/config
func (s *MCPConfigService) GetServerConfigHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serverID := vars["id"]

	config, err := s.GetServerConfig(serverID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"config": config,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// UpdateServerConfigHandler handles PUT /api/v1/mcp/servers/{id}/config
func (s *MCPConfigService) UpdateServerConfigHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serverID := vars["id"]

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	config, err := s.UpdateServerConfig(serverID, updates)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"config": config,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// DeleteServerConfigHandler handles DELETE /api/v1/mcp/servers/{id}/config
func (s *MCPConfigService) DeleteServerConfigHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serverID := vars["id"]

	if err := s.DeleteServerConfig(serverID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"message": "Configuration deleted successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetAllConfigsHandler handles GET /api/v1/mcp/configs
func (s *MCPConfigService) GetAllConfigsHandler(w http.ResponseWriter, r *http.Request) {
	configs := s.GetAllConfigs()

	response := map[string]interface{}{
		"configs": configs,
		"count":   len(configs),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// RegisterHandlers registers the configuration HTTP handlers
func (s *MCPConfigService) RegisterHandlers(router *mux.Router) {
	mcpRouter := router.PathPrefix("/api/v1/mcp").Subrouter()
	mcpRouter.HandleFunc("/servers/{id}/config", s.GetServerConfigHandler).Methods("GET")
	mcpRouter.HandleFunc("/servers/{id}/config", s.UpdateServerConfigHandler).Methods("PUT")
	mcpRouter.HandleFunc("/servers/{id}/config", s.DeleteServerConfigHandler).Methods("DELETE")
	mcpRouter.HandleFunc("/configs", s.GetAllConfigsHandler).Methods("GET")
}
