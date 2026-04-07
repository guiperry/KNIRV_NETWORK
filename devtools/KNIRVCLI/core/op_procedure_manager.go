package core

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// OpProcedureManager manages operational procedure operations
type OpProcedureManager struct {
	apiClient   *APIClient
	fileManager *FileManager
	configDir   string
}

// OpProcedureConfig represents the configuration for an operational procedure
type OpProcedureConfig struct {
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Description string                 `json:"description"`
	Author      string                 `json:"author"`
	License     string                 `json:"license"`
	Steps       []OpProcedureStep      `json:"steps"`
	Parameters  []OpProcedureParameter `json:"parameters"`
	Metadata    map[string]string      `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// OpProcedureStep represents a step in an operational procedure
type OpProcedureStep struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"`
	Action      string                 `json:"action"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Condition   string                 `json:"condition,omitempty"`
	OnSuccess   string                 `json:"on_success,omitempty"`
	OnFailure   string                 `json:"on_failure,omitempty"`
}

// OpProcedureParameter represents a parameter for an operational procedure
type OpProcedureParameter struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Type        string      `json:"type"`
	Required    bool        `json:"required"`
	Default     interface{} `json:"default,omitempty"`
	Enum        []string    `json:"enum,omitempty"`
}

// NewOpProcedureManager creates a new operational procedure manager
func NewOpProcedureManager(apiClient *APIClient, fileManager *FileManager, configDir string) (*OpProcedureManager, error) {
	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	return &OpProcedureManager{
		apiClient:   apiClient,
		fileManager: fileManager,
		configDir:   configDir,
	}, nil
}

// RegisterProcedure registers an operational procedure
func (m *OpProcedureManager) RegisterProcedure(ctx context.Context, config OpProcedureConfig) error {
	// Validate procedure configuration
	if err := m.validateProcedureConfig(config); err != nil {
		return err
	}

	// Check if procedure already exists
	if m.procedureExists(config.Name, config.Version) {
		return fmt.Errorf("procedure with name '%s' and version '%s' already exists", config.Name, config.Version)
	}

	// Set timestamps
	now := time.Now()
	config.CreatedAt = now
	config.UpdatedAt = now

	// Save procedure configuration
	if err := m.saveProcedureConfig(config); err != nil {
		return err
	}

	return nil
}

// UpdateProcedure updates an operational procedure configuration
func (m *OpProcedureManager) UpdateProcedure(ctx context.Context, config OpProcedureConfig) error {
	// Validate procedure configuration
	if err := m.validateProcedureConfig(config); err != nil {
		return err
	}

	// Check if procedure exists
	if !m.procedureExists(config.Name, config.Version) {
		return fmt.Errorf("procedure with name '%s' and version '%s' does not exist", config.Name, config.Version)
	}

	// Load existing configuration
	existingConfig, err := m.GetProcedure(config.Name, config.Version)
	if err != nil {
		return err
	}

	// Update timestamps
	config.CreatedAt = existingConfig.CreatedAt
	config.UpdatedAt = time.Now()

	// Save procedure configuration
	if err := m.saveProcedureConfig(config); err != nil {
		return err
	}

	return nil
}

// GetProcedure gets an operational procedure configuration
func (m *OpProcedureManager) GetProcedure(name, version string) (*OpProcedureConfig, error) {
	// Check if procedure exists
	if !m.procedureExists(name, version) {
		return nil, fmt.Errorf("procedure with name '%s' and version '%s' does not exist", name, version)
	}

	// Load procedure configuration
	configPath := m.getProcedureConfigPath(name, version)
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read procedure configuration: %w", err)
	}

	// Parse procedure configuration
	var config OpProcedureConfig
	if err := json.Unmarshal(configData, &config); err != nil {
		return nil, fmt.Errorf("failed to parse procedure configuration: %w", err)
	}

	return &config, nil
}

// ListProcedures lists all registered operational procedures
func (m *OpProcedureManager) ListProcedures() ([]OpProcedureConfig, error) {
	// Get procedure configuration files
	pattern := filepath.Join(m.configDir, "*.json")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to list procedure configurations: %w", err)
	}

	// Load procedure configurations
	var configs []OpProcedureConfig
	for _, match := range matches {
		// Read configuration file
		configData, err := os.ReadFile(match)
		if err != nil {
			return nil, fmt.Errorf("failed to read procedure configuration: %w", err)
		}

		// Parse configuration
		var config OpProcedureConfig
		if err := json.Unmarshal(configData, &config); err != nil {
			return nil, fmt.Errorf("failed to parse procedure configuration: %w", err)
		}

		configs = append(configs, config)
	}

	return configs, nil
}

// DeleteProcedure deletes an operational procedure configuration
func (m *OpProcedureManager) DeleteProcedure(name, version string) error {
	// Check if procedure exists
	if !m.procedureExists(name, version) {
		return fmt.Errorf("procedure with name '%s' and version '%s' does not exist", name, version)
	}

	// Delete procedure configuration
	configPath := m.getProcedureConfigPath(name, version)
	if err := os.Remove(configPath); err != nil {
		return fmt.Errorf("failed to delete procedure configuration: %w", err)
	}

	return nil
}

// ExecuteProcedure executes an operational procedure
func (m *OpProcedureManager) ExecuteProcedure(ctx context.Context, name, version string, params map[string]interface{}) error {
	// Get procedure configuration
	config, err := m.GetProcedure(name, version)
	if err != nil {
		return err
	}

	// Validate parameters
	if err := m.validateParameters(config, params); err != nil {
		return err
	}

	// TODO: Implement procedure execution engine
	// For now, just log the steps
	for i, step := range config.Steps {
		m.apiClient.logger.Infof("Executing step %d: %s", i+1, step.Name)
		// In a real implementation, we would execute each step based on its type and action
	}

	return nil
}

// validateProcedureConfig validates a procedure configuration
func (m *OpProcedureManager) validateProcedureConfig(config OpProcedureConfig) error {
	if config.Name == "" {
		return fmt.Errorf("procedure name is required")
	}

	if config.Version == "" {
		return fmt.Errorf("procedure version is required")
	}

	if len(config.Steps) == 0 {
		return fmt.Errorf("procedure must have at least one step")
	}

	// Validate each step
	for i, step := range config.Steps {
		if step.Name == "" {
			return fmt.Errorf("step %d name is required", i+1)
		}

		if step.Type == "" {
			return fmt.Errorf("step %d type is required", i+1)
		}

		if step.Action == "" {
			return fmt.Errorf("step %d action is required", i+1)
		}
	}

	return nil
}

// validateParameters validates procedure parameters
func (m *OpProcedureManager) validateParameters(config *OpProcedureConfig, params map[string]interface{}) error {
	// Check for required parameters
	for _, param := range config.Parameters {
		if param.Required {
			if _, ok := params[param.Name]; !ok {
				return fmt.Errorf("required parameter '%s' is missing", param.Name)
			}
		}
	}

	// TODO: Add type validation and enum validation

	return nil
}

// procedureExists checks if a procedure configuration exists
func (m *OpProcedureManager) procedureExists(name, version string) bool {
	configPath := m.getProcedureConfigPath(name, version)
	_, err := os.Stat(configPath)
	return err == nil
}

// getProcedureConfigPath gets the path to a procedure configuration file
func (m *OpProcedureManager) getProcedureConfigPath(name, version string) string {
	return filepath.Join(m.configDir, fmt.Sprintf("%s-%s.json", name, version))
}

// saveProcedureConfig saves a procedure configuration
func (m *OpProcedureManager) saveProcedureConfig(config OpProcedureConfig) error {
	// Marshal configuration to JSON
	configData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal procedure configuration: %w", err)
	}

	// Save configuration to file
	configPath := m.getProcedureConfigPath(config.Name, config.Version)
	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		return fmt.Errorf("failed to write procedure configuration: %w", err)
	}

	return nil
}
