package service

import (
	"context"
	"fmt"

	"KNIRV_Engine/agent/core"
)

// AgentService provides a unified interface for all agent operations
type AgentService struct {
	core      core.AgentCoreService
	storage   core.AgentStorage
	discovery core.AgentDiscovery
}

// ServiceConfig represents configuration for the agent service
type ServiceConfig struct {
	DBPath       string `json:"db_path"`
	PluginsDir   string `json:"plugins_dir"`
	WASMDir      string `json:"wasm_dir"`
	TemplatesDir string `json:"templates_dir"`
	OutputDir    string `json:"output_dir"`
	DataDir      string `json:"data_dir"`
}

// NewAgentService creates a new agent service
func NewAgentService(config *ServiceConfig) (*AgentService, error) {
	// Initialize storage
	storage, err := core.NewAgentStorageAdapter(config.DBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage: %v", err)
	}

	// Initialize discovery
	discovery := core.NewAgentDiscovery(config.PluginsDir, config.WASMDir, config.TemplatesDir)

	// Initialize core service
	coreService := core.NewAgentCoreService(storage, discovery)

	return &AgentService{
		core:      coreService,
		storage:   storage,
		discovery: discovery,
	}, nil
}

// Core CRUD operations - delegate to core service

// CreateAgent creates a new agent
func (s *AgentService) CreateAgent(ctx context.Context, agent *core.UnifiedAgent) error {
	return s.core.CreateAgent(ctx, agent)
}

// GetAgent retrieves an agent by ID
func (s *AgentService) GetAgent(ctx context.Context, id string) (*core.UnifiedAgent, error) {
	return s.core.GetAgent(ctx, id)
}

// UpdateAgent updates an existing agent
func (s *AgentService) UpdateAgent(ctx context.Context, agent *core.UnifiedAgent) error {
	return s.core.UpdateAgent(ctx, agent)
}

// DeleteAgent deletes an agent by ID
func (s *AgentService) DeleteAgent(ctx context.Context, id string) error {
	return s.core.DeleteAgent(ctx, id)
}

// ListAgents lists agents with optional filtering
func (s *AgentService) ListAgents(ctx context.Context, filter map[string]interface{}) ([]*core.UnifiedAgent, error) {
	return s.core.ListAgents(ctx, filter)
}

// Discovery operations - delegate to core service

// DiscoverAgents discovers agents from various sources
func (s *AgentService) DiscoverAgents(ctx context.Context) ([]*core.UnifiedAgent, error) {
	return s.core.DiscoverAgents(ctx)
}

// RegisterDiscoveredAgent registers a discovered agent
func (s *AgentService) RegisterDiscoveredAgent(ctx context.Context, agentPath string) (*core.UnifiedAgent, error) {
	return s.core.RegisterDiscoveredAgent(ctx, agentPath)
}

// Configuration operations - delegate to core service

// GetAgentConfig retrieves agent configuration
func (s *AgentService) GetAgentConfig(ctx context.Context, id string) (map[string]interface{}, error) {
	return s.core.GetAgentConfig(ctx, id)
}

// UpdateAgentConfig updates agent configuration
func (s *AgentService) UpdateAgentConfig(ctx context.Context, id string, config map[string]interface{}) error {
	return s.core.UpdateAgentConfig(ctx, id, config)
}

// Search operations - delegate to core service

// SearchAgents searches for agents
func (s *AgentService) SearchAgents(ctx context.Context, query string, limit int) ([]*core.UnifiedAgent, error) {
	return s.core.SearchAgents(ctx, query, limit)
}

// Lifecycle hooks - delegate to core service

// OnAgentCreated triggers the agent created lifecycle hook
func (s *AgentService) OnAgentCreated(agent *core.UnifiedAgent) {
	s.core.OnAgentCreated(agent)
}

// OnAgentUpdated triggers the agent updated lifecycle hook
func (s *AgentService) OnAgentUpdated(agent *core.UnifiedAgent) {
	s.core.OnAgentUpdated(agent)
}

// OnAgentDeleted triggers the agent deleted lifecycle hook
func (s *AgentService) OnAgentDeleted(id string) {
	s.core.OnAgentDeleted(id)
}

// Additional convenience methods

// GetAgentsByType retrieves agents by type
func (s *AgentService) GetAgentsByType(ctx context.Context, agentType string) ([]*core.UnifiedAgent, error) {
	filter := map[string]interface{}{
		"type": agentType,
	}
	return s.ListAgents(ctx, filter)
}

// GetAgentsByStatus retrieves agents by status
func (s *AgentService) GetAgentsByStatus(ctx context.Context, status string) ([]*core.UnifiedAgent, error) {
	filter := map[string]interface{}{
		"status": status,
	}
	return s.ListAgents(ctx, filter)
}

// GetAgentsByBuildTarget retrieves agents by build target
func (s *AgentService) GetAgentsByBuildTarget(ctx context.Context, buildTarget string) ([]*core.UnifiedAgent, error) {
	filter := map[string]interface{}{
		"build_target": buildTarget,
	}
	return s.ListAgents(ctx, filter)
}

// GetAgentsByCollection retrieves agents by collection
func (s *AgentService) GetAgentsByCollection(ctx context.Context, collection string) ([]*core.UnifiedAgent, error) {
	filter := map[string]interface{}{
		"collection": collection,
	}
	return s.ListAgents(ctx, filter)
}

// GetActiveAgents retrieves all active agents
func (s *AgentService) GetActiveAgents(ctx context.Context) ([]*core.UnifiedAgent, error) {
	return s.GetAgentsByStatus(ctx, "active")
}

// GetInactiveAgents retrieves all inactive agents
func (s *AgentService) GetInactiveAgents(ctx context.Context) ([]*core.UnifiedAgent, error) {
	return s.GetAgentsByStatus(ctx, "inactive")
}

// GetPluginAgents retrieves all plugin-based agents
func (s *AgentService) GetPluginAgents(ctx context.Context) ([]*core.UnifiedAgent, error) {
	return s.GetAgentsByBuildTarget(ctx, "plugin")
}

// GetWASMAgents retrieves all WASM-based agents
func (s *AgentService) GetWASMAgents(ctx context.Context) ([]*core.UnifiedAgent, error) {
	return s.GetAgentsByBuildTarget(ctx, "wasm")
}

// GetTemplateAgents retrieves all template agents
func (s *AgentService) GetTemplateAgents(ctx context.Context) ([]*core.UnifiedAgent, error) {
	return s.GetAgentsByBuildTarget(ctx, "template")
}

// ActivateAgent activates an agent
func (s *AgentService) ActivateAgent(ctx context.Context, id string) error {
	agent, err := s.GetAgent(ctx, id)
	if err != nil {
		return err
	}

	agent.Status = "active"
	return s.UpdateAgent(ctx, agent)
}

// DeactivateAgent deactivates an agent
func (s *AgentService) DeactivateAgent(ctx context.Context, id string) error {
	agent, err := s.GetAgent(ctx, id)
	if err != nil {
		return err
	}

	agent.Status = "inactive"
	return s.UpdateAgent(ctx, agent)
}

// CountAgents counts agents with optional filtering
func (s *AgentService) CountAgents(ctx context.Context, filter map[string]interface{}) (int, error) {
	return s.storage.Count(ctx, filter)
}

// AgentExists checks if an agent exists
func (s *AgentService) AgentExists(ctx context.Context, id string) (bool, error) {
	return s.storage.Exists(ctx, id)
}

// GetCoreService returns the core service for advanced operations
func (s *AgentService) GetCoreService() core.AgentCoreService {
	return s.core
}

// GetStorage returns the storage interface for direct access
func (s *AgentService) GetStorage() core.AgentStorage {
	return s.storage
}

// GetDiscovery returns the discovery interface for direct access
func (s *AgentService) GetDiscovery() core.AgentDiscovery {
	return s.discovery
}
