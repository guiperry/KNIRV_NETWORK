// agent_inferencer.go
package agentify

import (
	"context"
	"fmt"
	"sync"
)

// AgentInferencer manages the inference process through agent plugins
type AgentInferencer struct {
	pluginLoader *AgentPluginLoader
	activeAgents map[string]AgentPluginInterface
	sessions     map[string]string // Maps session IDs to agent IDs
	mutex        sync.RWMutex
}

// NewAgentInferencer creates a new agent inferencer
func NewAgentInferencer(pluginsDir string) *AgentInferencer {
	return &AgentInferencer{
		pluginLoader: NewAgentPluginLoader(pluginsDir),
		activeAgents: make(map[string]AgentPluginInterface),
		sessions:     make(map[string]string),
	}
}

// ActivateAgent activates an agent for a session
func (i *AgentInferencer) ActivateAgent(ctx context.Context, agentID, version, sessionID string, config map[string]interface{}) error {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	// Check if an agent is already active for this session
	if currentAgentID, ok := i.sessions[sessionID]; ok {
		if currentAgentID == agentID {
			// Agent is already active
			return nil
		}

		// Different agent is active, deactivate it
		if agent, ok := i.activeAgents[currentAgentID]; ok {
			agent.Stop()
			delete(i.activeAgents, currentAgentID)
		}
		delete(i.sessions, sessionID)
	}

	// Load the agent plugin
	agent, err := i.pluginLoader.LoadPlugin(agentID, version, config)
	if err != nil {
		return fmt.Errorf("failed to load agent plugin: %v", err)
	}

	// Start the agent
	if err := agent.Start(); err != nil {
		return fmt.Errorf("failed to start agent: %v", err)
	}

	// Store the active agent and session mapping
	i.activeAgents[agentID] = agent
	i.sessions[sessionID] = agentID

	return nil
}

// DeactivateAgent deactivates an agent for a session
func (i *AgentInferencer) DeactivateAgent(ctx context.Context, sessionID string) error {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	// Check if an agent is active for this session
	agentID, ok := i.sessions[sessionID]
	if !ok {
		// No agent active for this session
		return nil
	}

	// Get the agent
	agent, ok := i.activeAgents[agentID]
	if !ok {
		// Agent not found (should not happen)
		delete(i.sessions, sessionID)
		return nil
	}

	// Stop the agent
	if err := agent.Stop(); err != nil {
		return fmt.Errorf("failed to stop agent: %v", err)
	}

	// Remove the agent and session mapping
	delete(i.activeAgents, agentID)
	delete(i.sessions, sessionID)

	return nil
}

// ProcessInference processes an inference request through the appropriate agent
func (i *AgentInferencer) ProcessInference(ctx context.Context, sessionID string, request *InferenceRequest) (*InferenceResponse, error) {
	i.mutex.RLock()
	agentID, ok := i.sessions[sessionID]
	if !ok {
		i.mutex.RUnlock()
		return nil, fmt.Errorf("no agent active for session %s", sessionID)
	}

	agent, ok := i.activeAgents[agentID]
	i.mutex.RUnlock()

	if !ok {
		return nil, fmt.Errorf("agent %s not found", agentID)
	}

	// Set the session ID in the request
	request.SessionID = sessionID

	// Process the inference request through the agent
	return agent.ProcessInference(ctx, request)
}

// GetAgentCapabilities gets the capabilities of an agent
func (i *AgentInferencer) GetAgentCapabilities(ctx context.Context, sessionID string) (*AgentCapabilities, error) {
	i.mutex.RLock()
	agentID, ok := i.sessions[sessionID]
	if !ok {
		i.mutex.RUnlock()
		return nil, fmt.Errorf("no agent active for session %s", sessionID)
	}

	agent, ok := i.activeAgents[agentID]
	i.mutex.RUnlock()

	if !ok {
		return nil, fmt.Errorf("agent %s not found", agentID)
	}

	// Get the agent capabilities
	return agent.GetCapabilities(), nil
}

// GetAgentSchema gets the schema of an agent
func (i *AgentInferencer) GetAgentSchema(ctx context.Context, sessionID string) (*AgentSchema, error) {
	i.mutex.RLock()
	agentID, ok := i.sessions[sessionID]
	if !ok {
		i.mutex.RUnlock()
		return nil, fmt.Errorf("no agent active for session %s", sessionID)
	}

	agent, ok := i.activeAgents[agentID]
	i.mutex.RUnlock()

	if !ok {
		return nil, fmt.Errorf("agent %s not found", agentID)
	}

	// Get the agent schema
	return agent.GetSchema(), nil
}

// ListAvailableAgents lists the available agents
func (i *AgentInferencer) ListAvailableAgents(ctx context.Context) ([]string, error) {
	// Discover available plugins
	return i.pluginLoader.DiscoverPlugins()
}

// GetAgentMemory gets a value from the agent's memory
func (i *AgentInferencer) GetAgentMemory(ctx context.Context, sessionID string, key string) (interface{}, error) {
	i.mutex.RLock()
	agentID, ok := i.sessions[sessionID]
	if !ok {
		i.mutex.RUnlock()
		return nil, fmt.Errorf("no agent active for session %s", sessionID)
	}

	agent, ok := i.activeAgents[agentID]
	i.mutex.RUnlock()

	if !ok {
		return nil, fmt.Errorf("agent %s not found", agentID)
	}

	// Get the memory value
	return agent.GetMemory(key)
}

// SetAgentMemory sets a value in the agent's memory
func (i *AgentInferencer) SetAgentMemory(ctx context.Context, sessionID string, key string, value interface{}) error {
	i.mutex.RLock()
	agentID, ok := i.sessions[sessionID]
	if !ok {
		i.mutex.RUnlock()
		return fmt.Errorf("no agent active for session %s", sessionID)
	}

	agent, ok := i.activeAgents[agentID]
	i.mutex.RUnlock()

	if !ok {
		return fmt.Errorf("agent %s not found", agentID)
	}

	// Set the memory value
	return agent.SetMemory(key, value)
}

// GetTEEInfo gets information about the TEE for an agent
func (i *AgentInferencer) GetTEEInfo(ctx context.Context, sessionID string) (map[string]interface{}, error) {
	i.mutex.RLock()
	agentID, ok := i.sessions[sessionID]
	if !ok {
		i.mutex.RUnlock()
		return nil, fmt.Errorf("no agent active for session %s", sessionID)
	}

	agent, ok := i.activeAgents[agentID]
	i.mutex.RUnlock()

	if !ok {
		return nil, fmt.Errorf("agent %s not found", agentID)
	}

	// Get the TEE info
	return agent.GetTEEInfo(), nil
}
