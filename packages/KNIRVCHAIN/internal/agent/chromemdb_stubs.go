package agent

import (
	"context"
	"encoding/json"
	"fmt"
)

// ---- Agent ----

// StoreAgent persists an Agent in the chromem contextRecordCollection.
func (mgr *AgentChromemStore) StoreAgent(agent *Agent) error {
	if agent == nil {
		return fmt.Errorf("agent cannot be nil")
	}
	mgr.mu.Lock()
	defer mgr.mu.Unlock()

	if mgr.db == nil || mgr.db.GetContextCollection() == nil {
		return fmt.Errorf("context record collection not initialized")
	}

	data, err := json.Marshal(agent)
	if err != nil {
		return err
	}
	metadata := map[string]string{
		"type":       "agent",
		"owner":      agent.Owner,
		"agent_type": agent.AgentType,
	}
	return mgr.db.GetContextCollection().Add(
		context.Background(),
		[]string{agent.ID},
		nil,
		[]map[string]string{metadata},
		[]string{string(data)},
	)
}

// GetAgent retrieves an Agent by ID.
func (mgr *AgentChromemStore) GetAgent(id string) (*Agent, error) {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()

	if mgr.db == nil || mgr.db.GetContextCollection() == nil {
		return nil, fmt.Errorf("context record collection not initialized")
	}

	results, err := mgr.db.GetContextCollection().Query(
		context.Background(),
		id, 1,
		map[string]string{"type": "agent"},
		map[string]string{"documents": "true"},
	)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("agent not found: %s", id)
	}
	var a Agent
	if err := json.Unmarshal([]byte(results[0].Content), &a); err != nil {
		return nil, err
	}
	return &a, nil
}

// GetAgentsByOwner retrieves all Agents owned by the given address.
func (mgr *AgentChromemStore) GetAgentsByOwner(owner string) ([]*Agent, error) {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()

	if mgr.db == nil || mgr.db.GetContextCollection() == nil {
		return nil, nil
	}

	results, err := mgr.db.GetContextCollection().Query(
		context.Background(),
		"", 100,
		map[string]string{"type": "agent", "owner": owner},
		map[string]string{"documents": "true"},
	)
	if err != nil {
		return nil, err
	}
	agents := make([]*Agent, 0, len(results))
	for _, r := range results {
		var a Agent
		if err := json.Unmarshal([]byte(r.Content), &a); err == nil {
			agents = append(agents, &a)
		}
	}
	return agents, nil
}

// GetAgentsByType retrieves all Agents of the given type.
func (mgr *AgentChromemStore) GetAgentsByType(agentType string) ([]*Agent, error) {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()

	if mgr.db == nil || mgr.db.GetContextCollection() == nil {
		return nil, nil
	}

	results, err := mgr.db.GetContextCollection().Query(
		context.Background(),
		"", 100,
		map[string]string{"type": "agent"},
		map[string]string{"documents": "true"},
	)
	if err != nil {
		return nil, err
	}
	agents := make([]*Agent, 0)
	for _, r := range results {
		var a Agent
		if err := json.Unmarshal([]byte(r.Content), &a); err == nil && a.AgentType == agentType {
			agents = append(agents, &a)
		}
	}
	return agents, nil
}

// ---- Revision ----

// StoreRevision persists a Revision record.
func (mgr *AgentChromemStore) StoreRevision(revision *Revision) error {
	if revision == nil {
		return fmt.Errorf("revision cannot be nil")
	}
	mgr.mu.Lock()
	defer mgr.mu.Unlock()

	if mgr.db == nil || mgr.db.GetContextCollection() == nil {
		return fmt.Errorf("context record collection not initialized")
	}

	data, err := json.Marshal(revision)
	if err != nil {
		return err
	}
	metadata := map[string]string{
		"type":     "revision",
		"agent_id": revision.AgentId,
	}
	return mgr.db.GetContextCollection().Add(
		context.Background(),
		[]string{revision.ID},
		nil,
		[]map[string]string{metadata},
		[]string{string(data)},
	)
}

// ---- AgentRelationship ----

// StoreAgentRelationship persists an AgentRelationship.
func (mgr *AgentChromemStore) StoreAgentRelationship(rel *AgentRelationship) error {
	if rel == nil {
		return fmt.Errorf("relationship cannot be nil")
	}
	mgr.mu.Lock()
	defer mgr.mu.Unlock()

	if mgr.db == nil || mgr.db.GetContextCollection() == nil {
		return fmt.Errorf("context record collection not initialized")
	}

	data, err := json.Marshal(rel)
	if err != nil {
		return err
	}
	metadata := map[string]string{
		"type":     "relationship",
		"agent_id": rel.SourceAgentId,
	}
	return mgr.db.GetContextCollection().Add(
		context.Background(),
		[]string{rel.ID},
		nil,
		[]map[string]string{metadata},
		[]string{string(data)},
	)
}

// GetAgentRelationship retrieves an AgentRelationship by ID.
func (mgr *AgentChromemStore) GetAgentRelationship(id string) (*AgentRelationship, error) {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()

	if mgr.db == nil || mgr.db.GetContextCollection() == nil {
		return nil, fmt.Errorf("relationship not found: %s", id)
	}

	results, err := mgr.db.GetContextCollection().Query(
		context.Background(),
		id, 1,
		map[string]string{"type": "relationship"},
		map[string]string{"documents": "true"},
	)
	if err != nil || len(results) == 0 {
		return nil, fmt.Errorf("relationship not found: %s", id)
	}
	var rel AgentRelationship
	if err := json.Unmarshal([]byte(results[0].Content), &rel); err != nil {
		return nil, err
	}
	return &rel, nil
}

// GetAgentRelationships retrieves all relationships for an Agent.
func (mgr *AgentChromemStore) GetAgentRelationships(agentID string) ([]*AgentRelationship, error) {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()

	if mgr.db == nil || mgr.db.GetContextCollection() == nil {
		return nil, nil
	}

	results, err := mgr.db.GetContextCollection().Query(
		context.Background(),
		"", 100,
		map[string]string{"type": "relationship", "agent_id": agentID},
		map[string]string{"documents": "true"},
	)
	if err != nil {
		return nil, err
	}
	rels := make([]*AgentRelationship, 0, len(results))
	for _, r := range results {
		var rel AgentRelationship
		if err := json.Unmarshal([]byte(r.Content), &rel); err == nil {
			rels = append(rels, &rel)
		}
	}
	return rels, nil
}

// ---- Badge ----

// StoreBadge persists a Badge.
func (mgr *AgentChromemStore) StoreBadge(badge *Badge) error {
	if badge == nil {
		return fmt.Errorf("badge cannot be nil")
	}
	mgr.mu.Lock()
	defer mgr.mu.Unlock()

	if mgr.db == nil || mgr.db.GetContextCollection() == nil {
		return fmt.Errorf("context record collection not initialized")
	}

	data, err := json.Marshal(badge)
	if err != nil {
		return err
	}
	metadata := map[string]string{
		"type":       "badge",
		"badge_type": badge.BadgeType,
	}
	return mgr.db.GetContextCollection().Add(
		context.Background(),
		[]string{badge.ID},
		nil,
		[]map[string]string{metadata},
		[]string{string(data)},
	)
}

// GetBadge retrieves a Badge by ID.
func (mgr *AgentChromemStore) GetBadge(id string) (*Badge, error) {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()

	if mgr.db == nil || mgr.db.GetContextCollection() == nil {
		return nil, fmt.Errorf("badge not found: %s", id)
	}

	results, err := mgr.db.GetContextCollection().Query(
		context.Background(),
		id, 1,
		map[string]string{"type": "badge"},
		map[string]string{"documents": "true"},
	)
	if err != nil || len(results) == 0 {
		return nil, fmt.Errorf("badge not found: %s", id)
	}
	var b Badge
	if err := json.Unmarshal([]byte(results[0].Content), &b); err != nil {
		return nil, err
	}
	return &b, nil
}

// GetBadgesByType retrieves all Badges of the given type.
func (mgr *AgentChromemStore) GetBadgesByType(badgeType string) ([]*Badge, error) {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()

	if mgr.db == nil || mgr.db.GetContextCollection() == nil {
		return nil, nil
	}

	results, err := mgr.db.GetContextCollection().Query(
		context.Background(),
		"", 100,
		map[string]string{"type": "badge", "badge_type": badgeType},
		map[string]string{"documents": "true"},
	)
	if err != nil {
		return nil, err
	}
	badges := make([]*Badge, 0, len(results))
	for _, r := range results {
		var b Badge
		if err := json.Unmarshal([]byte(r.Content), &b); err == nil {
			badges = append(badges, &b)
		}
	}
	return badges, nil
}

// ---- BadgeAttachment ----

// StoreBadgeAttachment persists a BadgeAttachment.
func (mgr *AgentChromemStore) StoreBadgeAttachment(attachment *BadgeAttachment) error {
	if attachment == nil {
		return fmt.Errorf("attachment cannot be nil")
	}
	mgr.mu.Lock()
	defer mgr.mu.Unlock()

	if mgr.db == nil || mgr.db.GetContextCollection() == nil {
		return fmt.Errorf("context record collection not initialized")
	}

	data, err := json.Marshal(attachment)
	if err != nil {
		return err
	}
	metadata := map[string]string{
		"type":     "badge_attachment",
		"agent_id": attachment.AgentId,
		"badge_id": attachment.BadgeId,
	}
	return mgr.db.GetContextCollection().Add(
		context.Background(),
		[]string{attachment.ID},
		nil,
		[]map[string]string{metadata},
		[]string{string(data)},
	)
}

// GetBadgeAttachments retrieves all BadgeAttachments for an Agent.
func (mgr *AgentChromemStore) GetBadgeAttachments(agentID string) ([]*BadgeAttachment, error) {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()

	if mgr.db == nil || mgr.db.GetContextCollection() == nil {
		return nil, nil
	}

	results, err := mgr.db.GetContextCollection().Query(
		context.Background(),
		"", 100,
		map[string]string{"type": "badge_attachment", "agent_id": agentID},
		map[string]string{"documents": "true"},
	)
	if err != nil {
		return nil, err
	}
	attachments := make([]*BadgeAttachment, 0, len(results))
	for _, r := range results {
		var a BadgeAttachment
		if err := json.Unmarshal([]byte(r.Content), &a); err == nil {
			attachments = append(attachments, &a)
		}
	}
	return attachments, nil
}

// GetBadgeAttachment retrieves a BadgeAttachment by ID.
func (mgr *AgentChromemStore) GetBadgeAttachment(attachmentID string) (*BadgeAttachment, error) {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()

	if mgr.db == nil || mgr.db.GetContextCollection() == nil {
		return nil, fmt.Errorf("attachment not found: %s", attachmentID)
	}

	results, err := mgr.db.GetContextCollection().Query(
		context.Background(),
		attachmentID, 1,
		map[string]string{"type": "badge_attachment"},
		map[string]string{"documents": "true"},
	)
	if err != nil || len(results) == 0 {
		return nil, fmt.Errorf("attachment not found: %s", attachmentID)
	}
	var a BadgeAttachment
	if err := json.Unmarshal([]byte(results[0].Content), &a); err != nil {
		return nil, err
	}
	return &a, nil
}

// GetBadgeAttachmentByAgentAndBadge retrieves a BadgeAttachment by agent and badge IDs.
func (mgr *AgentChromemStore) GetBadgeAttachmentByAgentAndBadge(agentID, badgeID string) (*BadgeAttachment, error) {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()

	if mgr.db == nil || mgr.db.GetContextCollection() == nil {
		return nil, fmt.Errorf("attachment not found for agent=%s badge=%s", agentID, badgeID)
	}

	results, err := mgr.db.GetContextCollection().Query(
		context.Background(),
		"", 10,
		map[string]string{"type": "badge_attachment", "agent_id": agentID, "badge_id": badgeID},
		map[string]string{"documents": "true"},
	)
	if err != nil || len(results) == 0 {
		return nil, fmt.Errorf("attachment not found for agent=%s badge=%s", agentID, badgeID)
	}
	var a BadgeAttachment
	if err := json.Unmarshal([]byte(results[0].Content), &a); err != nil {
		return nil, err
	}
	return &a, nil
}

// ---- AgentPlugin ----

// StoreAgentPlugin persists an AgentPlugin.
func (mgr *AgentChromemStore) StoreAgentPlugin(plugin *AgentPlugin) error {
	if plugin == nil {
		return fmt.Errorf("plugin cannot be nil")
	}
	mgr.mu.Lock()
	defer mgr.mu.Unlock()

	if mgr.db == nil || mgr.db.GetContextCollection() == nil {
		return fmt.Errorf("context record collection not initialized")
	}

	data, err := json.Marshal(plugin)
	if err != nil {
		return err
	}
	metadata := map[string]string{
		"type":     "plugin",
		"agent_id": plugin.AgentId,
	}
	return mgr.db.GetContextCollection().Add(
		context.Background(),
		[]string{plugin.ID},
		nil,
		[]map[string]string{metadata},
		[]string{string(data)},
	)
}

// GetAgentPlugins retrieves all plugins for an Agent.
func (mgr *AgentChromemStore) GetAgentPlugins(agentID string) ([]*AgentPlugin, error) {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()

	if mgr.db == nil || mgr.db.GetContextCollection() == nil {
		return nil, nil
	}

	results, err := mgr.db.GetContextCollection().Query(
		context.Background(),
		"", 100,
		map[string]string{"type": "plugin", "agent_id": agentID},
		map[string]string{"documents": "true"},
	)
	if err != nil {
		return nil, err
	}
	plugins := make([]*AgentPlugin, 0, len(results))
	for _, r := range results {
		var p AgentPlugin
		if err := json.Unmarshal([]byte(r.Content), &p); err == nil {
			plugins = append(plugins, &p)
		}
	}
	return plugins, nil
}

// GetAgentPlugin retrieves an AgentPlugin by ID.
func (mgr *AgentChromemStore) GetAgentPlugin(pluginID string) (*AgentPlugin, error) {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()

	if mgr.db == nil || mgr.db.GetContextCollection() == nil {
		return nil, fmt.Errorf("plugin not found: %s", pluginID)
	}

	results, err := mgr.db.GetContextCollection().Query(
		context.Background(),
		pluginID, 1,
		map[string]string{"type": "plugin"},
		map[string]string{"documents": "true"},
	)
	if err != nil || len(results) == 0 {
		return nil, fmt.Errorf("plugin not found: %s", pluginID)
	}
	var p AgentPlugin
	if err := json.Unmarshal([]byte(results[0].Content), &p); err != nil {
		return nil, err
	}
	return &p, nil
}
