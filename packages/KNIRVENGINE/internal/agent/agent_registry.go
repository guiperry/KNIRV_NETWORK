package agent

import (
	"context"
	"fmt"
)

// AgentRegistry manages agent storage and retrieval using UnifiedAgentStorage
// This is now a thin wrapper around UnifiedAgentStorage for backward compatibility
type AgentRegistry struct {
	storage *UnifiedAgentStorage
}

// NewAgentRegistry creates a new agent registry backed by UnifiedAgentStorage
func NewAgentRegistry(dbPath string) (*AgentRegistry, error) {
	// Create the unified storage
	storage, err := NewUnifiedAgentStorage(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create unified storage: %v", err)
	}

	return &AgentRegistry{
		storage: storage,
	}, nil
}

// RegisterAgent stores an agent configuration in the registry
func (r *AgentRegistry) RegisterAgent(agentID string, config map[string]interface{}) error {
	return r.storage.RegisterAgentConfig(context.Background(), agentID, config)
}

// GetAgent retrieves an agent configuration from the registry
func (r *AgentRegistry) GetAgent(agentID string) (map[string]interface{}, error) {
	return r.storage.GetAgentConfig(context.Background(), agentID)
}

// DeleteAgent removes an agent from the registry
func (r *AgentRegistry) DeleteAgent(agentID string) error {
	ctx := context.Background()

	// First check if the agent exists
	_, err := r.GetAgent(agentID)
	if err != nil {
		// Agent doesn't exist, log and return success
		fmt.Printf("Agent %s not found in registry, nothing to delete\n", agentID)
		return nil
	}

	// Delete the agent using unified storage
	err = r.storage.DeleteAgent(ctx, agentID)
	if err != nil {
		return fmt.Errorf("failed to delete agent: %v", err)
	}

	fmt.Printf("Successfully deleted agent %s from registry\n", agentID)
	return nil
}

// ListAgents returns a list of all agent IDs
func (r *AgentRegistry) ListAgents() ([]string, error) {
	ctx := context.Background()
	agents, err := r.storage.ListAgents(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents: %v", err)
	}

	// Extract IDs from the agents
	ids := make([]string, len(agents))
	for i, agent := range agents {
		ids[i] = agent.ID
	}

	return ids, nil
}

// Close is a no-op since chromem.DB doesn't have a Close method
// This method is provided for interface compatibility
func (r *AgentRegistry) Close() error {
	// The chromem.DB doesn't have a Close method in the current version
	// This is a no-op for now
	return nil
}
