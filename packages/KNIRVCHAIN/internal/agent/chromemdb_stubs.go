package agent

import "fmt"

// StoreAgent persists an Agent in the chromem store.
func (mgr *ChromemManager) StoreAgent(agent *Agent) error {
	if agent == nil {
		return fmt.Errorf("agent cannot be nil")
	}
	return nil // stub: implement full chromem persistence when ready
}

// GetAgent retrieves an Agent by ID.
func (mgr *ChromemManager) GetAgent(id string) (*Agent, error) {
	return nil, fmt.Errorf("agent not found: %s", id) // stub
}

// GetAgentsByOwner retrieves all Agents owned by the given address.
func (mgr *ChromemManager) GetAgentsByOwner(owner string) ([]*Agent, error) {
	return nil, nil // stub
}

// GetAgentsByType retrieves all Agents of the given type.
func (mgr *ChromemManager) GetAgentsByType(agentType string) ([]*Agent, error) {
	return nil, nil // stub
}

// StoreRevision persists a Revision record.
func (mgr *ChromemManager) StoreRevision(revision *Revision) error {
	if revision == nil {
		return fmt.Errorf("revision cannot be nil")
	}
	return nil // stub
}

// StoreAgentRelationship persists an AgentRelationship.
func (mgr *ChromemManager) StoreAgentRelationship(rel *AgentRelationship) error {
	if rel == nil {
		return fmt.Errorf("relationship cannot be nil")
	}
	return nil // stub
}

// GetAgentRelationship retrieves an AgentRelationship by ID.
func (mgr *ChromemManager) GetAgentRelationship(id string) (*AgentRelationship, error) {
	return nil, fmt.Errorf("relationship not found: %s", id) // stub
}

// GetAgentRelationships retrieves all relationships for an Agent.
func (mgr *ChromemManager) GetAgentRelationships(agentID string) ([]*AgentRelationship, error) {
	return nil, nil // stub
}

// StoreBadge persists a Badge.
func (mgr *ChromemManager) StoreBadge(badge *Badge) error {
	if badge == nil {
		return fmt.Errorf("badge cannot be nil")
	}
	return nil // stub
}

// GetBadge retrieves a Badge by ID.
func (mgr *ChromemManager) GetBadge(id string) (*Badge, error) {
	return nil, fmt.Errorf("badge not found: %s", id) // stub
}

// GetBadgesByType retrieves all Badges of the given type.
func (mgr *ChromemManager) GetBadgesByType(badgeType string) ([]*Badge, error) {
	return nil, nil // stub
}

// StoreBadgeAttachment persists a BadgeAttachment.
func (mgr *ChromemManager) StoreBadgeAttachment(attachment *BadgeAttachment) error {
	if attachment == nil {
		return fmt.Errorf("attachment cannot be nil")
	}
	return nil // stub
}

// GetBadgeAttachments retrieves all BadgeAttachments for an Agent.
func (mgr *ChromemManager) GetBadgeAttachments(agentID string) ([]*BadgeAttachment, error) {
	return nil, nil // stub
}

// GetBadgeAttachment retrieves a BadgeAttachment by ID.
func (mgr *ChromemManager) GetBadgeAttachment(attachmentID string) (*BadgeAttachment, error) {
	return nil, fmt.Errorf("attachment not found: %s", attachmentID) // stub
}

// GetBadgeAttachmentByAgentAndBadge retrieves a BadgeAttachment by agent and badge IDs.
func (mgr *ChromemManager) GetBadgeAttachmentByAgentAndBadge(agentID, badgeID string) (*BadgeAttachment, error) {
	return nil, fmt.Errorf("attachment not found for agent=%s badge=%s", agentID, badgeID) // stub
}

// StoreAgentPlugin persists an AgentPlugin.
func (mgr *ChromemManager) StoreAgentPlugin(plugin *AgentPlugin) error {
	if plugin == nil {
		return fmt.Errorf("plugin cannot be nil")
	}
	return nil // stub
}

// GetAgentPlugins retrieves all plugins for an Agent.
func (mgr *ChromemManager) GetAgentPlugins(agentID string) ([]*AgentPlugin, error) {
	return nil, nil // stub
}

// GetAgentPlugin retrieves an AgentPlugin by ID.
func (mgr *ChromemManager) GetAgentPlugin(pluginID string) (*AgentPlugin, error) {
	return nil, fmt.Errorf("plugin not found: %s", pluginID) // stub
}
