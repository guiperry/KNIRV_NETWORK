package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"strings"
	"time"

	"github.com/google/uuid"
)

// AgentCoreServiceImpl implements the AgentCoreService interface
type AgentCoreServiceImpl struct {
	storage          AgentStorage
	discovery        AgentDiscovery
	lifecycleManager AgentLifecycleManager
	validator        AgentValidator
}

// NewAgentCoreService creates a new agent core service
func NewAgentCoreService(storage AgentStorage, discovery AgentDiscovery) *AgentCoreServiceImpl {
	return &AgentCoreServiceImpl{
		storage:          storage,
		discovery:        discovery,
		lifecycleManager: NewAgentLifecycleManager(),
		validator:        NewDefaultAgentValidator(),
	}
}

// CreateAgent creates a new agent
func (s *AgentCoreServiceImpl) CreateAgent(ctx context.Context, agent *UnifiedAgent) error {
	// Validate the agent
	if errors := s.validator.ValidateAgent(agent); len(errors) > 0 {
		return fmt.Errorf("validation failed: %v", errors)
	}

	// Set default values
	if agent.ID == "" {
		agent.ID = uuid.New().String()
	}
	if agent.CreatedAt.IsZero() {
		agent.CreatedAt = time.Now()
	}
	agent.UpdatedAt = time.Now()

	// Set default status
	if agent.Status == "" {
		agent.Status = "inactive"
	}

	// Set default terminal config if not provided
	if agent.DefaultTerminalConfig == nil {
		// Check if there's a terminal_config in the agent's Config
		if agent.Config != nil {
			if terminalConfig, ok := agent.Config["terminal_config"].(map[string]interface{}); ok {
				// Parse the terminal config using our utility functions
				parser := NewConfigParser()
				agent.DefaultTerminalConfig = parser.ParseTerminalConfig(terminalConfig)
			}
		}

		// If still nil, set default values
		if agent.DefaultTerminalConfig == nil {
			agent.DefaultTerminalConfig = &TerminalConfig{
				DefaultRows:    24,
				DefaultCols:    80,
				FontSize:       14,
				FontFamily:     "Menlo, Monaco, 'Courier New', monospace",
				Theme:          "dark",
				ScrollbackSize: 5000,
				AutoOpen:       false,
			}
		}
	}

	// Store the agent
	if err := s.storage.Store(ctx, agent); err != nil {
		return fmt.Errorf("failed to store agent: %w", err)
	}

	// Trigger lifecycle hook
	s.OnAgentCreated(agent)

	return nil
}

// GetAgent retrieves an agent by ID
func (s *AgentCoreServiceImpl) GetAgent(ctx context.Context, id string) (*UnifiedAgent, error) {
	if id == "" {
		return nil, fmt.Errorf("agent ID is required")
	}

	agent, err := s.storage.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	return agent, nil
}

// UpdateAgent updates an existing agent
func (s *AgentCoreServiceImpl) UpdateAgent(ctx context.Context, agent *UnifiedAgent) error {
	if agent.ID == "" {
		return fmt.Errorf("agent ID is required for update")
	}

	// Validate the agent
	if errors := s.validator.ValidateAgent(agent); len(errors) > 0 {
		return fmt.Errorf("validation failed: %v", errors)
	}

	// Check if agent exists
	exists, err := s.storage.Exists(ctx, agent.ID)
	if err != nil {
		return fmt.Errorf("failed to check agent existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("agent not found: %s", agent.ID)
	}

	// Update timestamp
	agent.UpdatedAt = time.Now()

	// Update the agent
	if err := s.storage.Update(ctx, agent); err != nil {
		return fmt.Errorf("failed to update agent: %w", err)
	}

	// Trigger lifecycle hook
	s.OnAgentUpdated(agent)

	return nil
}

// DeleteAgent deletes an agent by ID
func (s *AgentCoreServiceImpl) DeleteAgent(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("agent ID is required")
	}

	// Check if agent exists
	exists, err := s.storage.Exists(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to check agent existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("agent not found: %s", id)
	}

	// Delete the agent
	if err := s.storage.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete agent: %w", err)
	}

	// Trigger lifecycle hook
	s.OnAgentDeleted(id)

	return nil
}

// ListAgents lists agents with optional filtering
func (s *AgentCoreServiceImpl) ListAgents(ctx context.Context, filter map[string]interface{}) ([]*UnifiedAgent, error) {
	agents, err := s.storage.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents: %w", err)
	}

	return agents, nil
}

// DiscoverAgents discovers agents from various sources
func (s *AgentCoreServiceImpl) DiscoverAgents(ctx context.Context) ([]*UnifiedAgent, error) {
	var allAgents []*UnifiedAgent

	// Discover from plugins
	pluginAgents, err := s.discovery.DiscoverFromPlugins(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("failed to discover plugin agents: %w", err)
	}
	allAgents = append(allAgents, pluginAgents...)

	// Discover from WASM
	wasmAgents, err := s.discovery.DiscoverFromWASM(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("failed to discover WASM agents: %w", err)
	}
	allAgents = append(allAgents, wasmAgents...)

	// Discover from templates
	templateAgents, err := s.discovery.DiscoverFromTemplates(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("failed to discover template agents: %w", err)
	}
	allAgents = append(allAgents, templateAgents...)

	return allAgents, nil
}

// RegisterDiscoveredAgent registers a discovered agent
func (s *AgentCoreServiceImpl) RegisterDiscoveredAgent(ctx context.Context, agentPath string) (*UnifiedAgent, error) {
	// Extract metadata based on file type
	var agent *UnifiedAgent
	var err error

	// Determine agent type from path
	if isPluginFile(agentPath) {
		agent, err = s.discovery.ExtractMetadataFromPlugin(ctx, agentPath)
	} else if isWASMFile(agentPath) {
		agent, err = s.discovery.ExtractMetadataFromWASM(ctx, agentPath)
	} else if isZipFile(agentPath) {
		// For zip files, we need to determine the content type
		// Extract metadata from the zip file based on its contents
		agents, err := s.discovery.DiscoverFromZip(ctx, agentPath)
		if err != nil {
			return nil, fmt.Errorf("failed to extract metadata from zip: %w", err)
		}
		if len(agents) == 0 {
			return nil, fmt.Errorf("no valid agents found in zip file: %s", agentPath)
		}
		// Use the first agent found in the zip
		agent = agents[0]
	} else {
		return nil, fmt.Errorf("unsupported agent file type: %s", agentPath)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to extract metadata: %w", err)
	}

	// Create the agent
	if err := s.CreateAgent(ctx, agent); err != nil {
		return nil, fmt.Errorf("failed to register agent: %w", err)
	}

	return agent, nil
}

// GetAgentConfig retrieves agent configuration
func (s *AgentCoreServiceImpl) GetAgentConfig(ctx context.Context, id string) (map[string]interface{}, error) {
	agent, err := s.GetAgent(ctx, id)
	if err != nil {
		return nil, err
	}

	return agent.Config, nil
}

// UpdateAgentConfig updates agent configuration
func (s *AgentCoreServiceImpl) UpdateAgentConfig(ctx context.Context, id string, config map[string]interface{}) error {
	// Validate config
	if errors := s.validator.ValidateConfig(config); len(errors) > 0 {
		return fmt.Errorf("config validation failed: %v", errors)
	}

	// Get existing agent
	agent, err := s.GetAgent(ctx, id)
	if err != nil {
		return err
	}

	// Parse the configuration using our utility functions
	parser := NewConfigParser()
	parsedConfig := parser.ParseAgentConfig(config)

	// Update agent metadata based on parsed config
	if parsedConfig.Name != "" {
		agent.Name = parsedConfig.Name
	}

	if parsedConfig.Type != "" {
		agent.Type = parsedConfig.Type
	}

	if parsedConfig.Version != "" {
		agent.Version = parsedConfig.Version
	}

	if parsedConfig.Description != "" {
		agent.Description = parsedConfig.Description
	}

	// Update terminal config if provided
	if terminalConfig, ok := config["terminal_config"].(map[string]interface{}); ok {
		agent.DefaultTerminalConfig = parser.ParseTerminalConfig(terminalConfig)
	}

	// Update config
	agent.Config = config
	agent.UpdatedAt = time.Now()

	// Update the agent
	return s.UpdateAgent(ctx, agent)
}

// SearchAgents searches for agents
func (s *AgentCoreServiceImpl) SearchAgents(ctx context.Context, query string, limit int) ([]*UnifiedAgent, error) {
	if query == "" {
		return nil, fmt.Errorf("search query is required")
	}

	agents, err := s.storage.Search(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search agents: %w", err)
	}

	return agents, nil
}

// OnAgentCreated triggers the agent created lifecycle hook
func (s *AgentCoreServiceImpl) OnAgentCreated(agent *UnifiedAgent) {
	s.lifecycleManager.TriggerCreated(agent)
}

// OnAgentUpdated triggers the agent updated lifecycle hook
func (s *AgentCoreServiceImpl) OnAgentUpdated(agent *UnifiedAgent) {
	s.lifecycleManager.TriggerUpdated(agent)
}

// OnAgentDeleted triggers the agent deleted lifecycle hook
func (s *AgentCoreServiceImpl) OnAgentDeleted(id string) {
	s.lifecycleManager.TriggerDeleted(id)
}

// LoadAgent loads an agent based on its build target and path
func (s *AgentCoreServiceImpl) LoadAgent(ctx context.Context, agent *UnifiedAgent) error {
	switch agent.BuildTarget {
	case "plugin":
		return s.loadPluginAgent(ctx, agent)
	case "wasm":
		return s.loadWASMAgent(ctx, agent)
	case "template":
		return s.loadTemplateAgent(ctx, agent)
	default:
		return fmt.Errorf("unsupported build target: %s", agent.BuildTarget)
	}
}

// loadPluginAgent loads a Go plugin agent
func (s *AgentCoreServiceImpl) loadPluginAgent(ctx context.Context, agent *UnifiedAgent) error {
	pluginPath := agent.PluginPath

	// Handle zip files
	if strings.HasSuffix(pluginPath, ".zip") {
		return s.loadAgentFromZip(ctx, agent, "plugin")
	}

	// Check if plugin file exists
	if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
		return fmt.Errorf("plugin file not found: %s", pluginPath)
	}

	// Load the plugin
	p, err := plugin.Open(pluginPath)
	if err != nil {
		return fmt.Errorf("failed to load plugin: %v", err)
	}

	// Look up the Plugin symbol
	pluginSymbol, err := p.Lookup("Plugin")
	if err != nil {
		return fmt.Errorf("failed to find Plugin symbol: %v", err)
	}

	// Store plugin reference in agent config for later use
	if agent.Config == nil {
		agent.Config = make(map[string]interface{})
	}
	agent.Config["_plugin_loaded"] = true
	agent.Config["_plugin_symbol"] = pluginSymbol

	agent.Status = "loaded"
	return s.UpdateAgent(ctx, agent)
}

// loadWASMAgent loads a WASM agent
func (s *AgentCoreServiceImpl) loadWASMAgent(ctx context.Context, agent *UnifiedAgent) error {
	wasmPath := agent.PluginPath

	// Handle zip files
	if strings.HasSuffix(wasmPath, ".zip") {
		return s.loadAgentFromZip(ctx, agent, "wasm")
	}

	// Check if WASM file exists
	if _, err := os.Stat(wasmPath); os.IsNotExist(err) {
		return fmt.Errorf("WASM file not found: %s", wasmPath)
	}

	// For WASM, we'll mark as loaded and let the WASM loader handle the actual loading
	if agent.Config == nil {
		agent.Config = make(map[string]interface{})
	}
	agent.Config["_wasm_loaded"] = true

	agent.Status = "loaded"
	return s.UpdateAgent(ctx, agent)
}

// loadTemplateAgent loads a template-based agent
func (s *AgentCoreServiceImpl) loadTemplateAgent(ctx context.Context, agent *UnifiedAgent) error {
	// Template agents are built on-demand, so just mark as ready
	if agent.Config == nil {
		agent.Config = make(map[string]interface{})
	}
	agent.Config["_template_ready"] = true

	agent.Status = "ready"
	return s.UpdateAgent(ctx, agent)
}

// loadAgentFromZip loads an agent from a zip file
func (s *AgentCoreServiceImpl) loadAgentFromZip(ctx context.Context, agent *UnifiedAgent, agentType string) error {
	// For zip files, we mark them as ready for extraction/loading
	if agent.Config == nil {
		agent.Config = make(map[string]interface{})
	}
	agent.Config["_zip_ready"] = true
	agent.Config["_zip_type"] = agentType

	agent.Status = "ready"
	return s.UpdateAgent(ctx, agent)
}

// StartAgent starts a loaded agent
func (s *AgentCoreServiceImpl) StartAgent(ctx context.Context, agentID string) error {
	agent, err := s.GetAgent(ctx, agentID)
	if err != nil {
		return err
	}

	if agent.Status != "loaded" && agent.Status != "ready" {
		return fmt.Errorf("agent must be loaded before starting")
	}

	// Update status to running
	agent.Status = "running"
	return s.UpdateAgent(ctx, agent)
}

// StopAgent stops a running agent
func (s *AgentCoreServiceImpl) StopAgent(ctx context.Context, agentID string) error {
	agent, err := s.GetAgent(ctx, agentID)
	if err != nil {
		return err
	}

	if agent.Status != "running" {
		return fmt.Errorf("agent is not running")
	}

	// Update status to stopped
	agent.Status = "stopped"
	return s.UpdateAgent(ctx, agent)
}

// Helper functions
func isPluginFile(path string) bool {
	ext := filepath.Ext(path)
	return ext == ".so" || ext == ".dll" || ext == ".dylib"
}

func isWASMFile(path string) bool {
	return filepath.Ext(path) == ".wasm"
}

// isZipFile checks if a file is a zip archive
// Currently unused but kept for future implementation of zip-based agent packaging
func isZipFile(path string) bool {
	return filepath.Ext(path) == ".zip"
}
