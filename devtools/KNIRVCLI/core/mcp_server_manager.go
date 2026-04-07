package core

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// MCPServerManager manages MCP server operations
type MCPServerManager struct {
	apiClient   *APIClient
	fileManager *FileManager
	configDir   string
}

// MCPServerConfig represents the configuration for an MCP server
type MCPServerConfig struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	URL         string            `json:"url"`
	APIKey      string            `json:"api_key,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// NewMCPServerManager creates a new MCP server manager
func NewMCPServerManager(apiClient *APIClient, fileManager *FileManager, configDir string) (*MCPServerManager, error) {
	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	return &MCPServerManager{
		apiClient:   apiClient,
		fileManager: fileManager,
		configDir:   configDir,
	}, nil
}

// RegisterServer registers an MCP server
func (m *MCPServerManager) RegisterServer(ctx context.Context, config MCPServerConfig) error {
	// Validate server configuration
	if err := m.validateServerConfig(config); err != nil {
		return err
	}

	// Check if server already exists
	if m.serverExists(config.Name) {
		return fmt.Errorf("server with name '%s' already exists", config.Name)
	}

	// Set timestamps
	now := time.Now()
	config.CreatedAt = now
	config.UpdatedAt = now

	// Save server configuration
	if err := m.saveServerConfig(config); err != nil {
		return err
	}

	return nil
}

// UpdateServer updates an MCP server configuration
func (m *MCPServerManager) UpdateServer(ctx context.Context, config MCPServerConfig) error {
	// Validate server configuration
	if err := m.validateServerConfig(config); err != nil {
		return err
	}

	// Check if server exists
	if !m.serverExists(config.Name) {
		return fmt.Errorf("server with name '%s' does not exist", config.Name)
	}

	// Load existing configuration
	existingConfig, err := m.GetServer(config.Name)
	if err != nil {
		return err
	}

	// Update timestamps
	config.CreatedAt = existingConfig.CreatedAt
	config.UpdatedAt = time.Now()

	// Save server configuration
	if err := m.saveServerConfig(config); err != nil {
		return err
	}

	return nil
}

// GetServer gets an MCP server configuration
func (m *MCPServerManager) GetServer(name string) (*MCPServerConfig, error) {
	// Check if server exists
	if !m.serverExists(name) {
		return nil, fmt.Errorf("server with name '%s' does not exist", name)
	}

	// Load server configuration
	configPath := m.getServerConfigPath(name)
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read server configuration: %w", err)
	}

	// Parse server configuration
	var config MCPServerConfig
	if err := json.Unmarshal(configData, &config); err != nil {
		return nil, fmt.Errorf("failed to parse server configuration: %w", err)
	}

	return &config, nil
}

// ListServers lists all registered MCP servers
func (m *MCPServerManager) ListServers() ([]MCPServerConfig, error) {
	// Get server configuration files
	pattern := filepath.Join(m.configDir, "*.json")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to list server configurations: %w", err)
	}

	// Load server configurations
	var configs []MCPServerConfig
	for _, match := range matches {
		// Read configuration file
		configData, err := os.ReadFile(match)
		if err != nil {
			return nil, fmt.Errorf("failed to read server configuration: %w", err)
		}

		// Parse configuration
		var config MCPServerConfig
		if err := json.Unmarshal(configData, &config); err != nil {
			return nil, fmt.Errorf("failed to parse server configuration: %w", err)
		}

		configs = append(configs, config)
	}

	return configs, nil
}

// DeleteServer deletes an MCP server configuration
func (m *MCPServerManager) DeleteServer(name string) error {
	// Check if server exists
	if !m.serverExists(name) {
		return fmt.Errorf("server with name '%s' does not exist", name)
	}

	// Delete server configuration
	configPath := m.getServerConfigPath(name)
	if err := os.Remove(configPath); err != nil {
		return fmt.Errorf("failed to delete server configuration: %w", err)
	}

	return nil
}

// TestServerConnection tests the connection to an MCP server
func (m *MCPServerManager) TestServerConnection(ctx context.Context, name string) error {
	// Get server configuration
	config, err := m.GetServer(name)
	if err != nil {
		return err
	}

	// Create temporary API client for the server
	serverClient := NewAPIClient(
		config.URL,
		WithTimeout(5*time.Second),
		WithRetries(1),
		WithLogger(m.apiClient.logger),
	)

	// Test connection with health check
	if err := serverClient.HealthCheck(ctx); err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	return nil
}

// validateServerConfig validates a server configuration
func (m *MCPServerManager) validateServerConfig(config MCPServerConfig) error {
	if config.Name == "" {
		return fmt.Errorf("server name is required")
	}

	if config.URL == "" {
		return fmt.Errorf("server URL is required")
	}

	// TODO: Add more validation as needed

	return nil
}

// serverExists checks if a server configuration exists
func (m *MCPServerManager) serverExists(name string) bool {
	configPath := m.getServerConfigPath(name)
	_, err := os.Stat(configPath)
	return err == nil
}

// getServerConfigPath gets the path to a server configuration file
func (m *MCPServerManager) getServerConfigPath(name string) string {
	return filepath.Join(m.configDir, fmt.Sprintf("%s.json", name))
}

// saveServerConfig saves a server configuration
func (m *MCPServerManager) saveServerConfig(config MCPServerConfig) error {
	// Marshal configuration to JSON
	configData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal server configuration: %w", err)
	}

	// Save configuration to file
	configPath := m.getServerConfigPath(config.Name)
	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		return fmt.Errorf("failed to write server configuration: %w", err)
	}

	return nil
}
