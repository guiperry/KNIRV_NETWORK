package core

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/philippgille/chromem-go"
)

// AgentStorageAdapter adapts the existing UnifiedAgentStorage to the new AgentStorage interface
type AgentStorageAdapter struct {
	db         *chromem.DB
	collection *chromem.Collection
}

// NewAgentStorageAdapter creates a new agent storage adapter
func NewAgentStorageAdapter(dbPath string) (*AgentStorageAdapter, error) {
	// Create chromem-go database
	db := chromem.NewDB()

	// Create collection for agents
	collection, err := db.CreateCollection("agents", nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create agents collection: %v", err)
	}

	return &AgentStorageAdapter{
		db:         db,
		collection: collection,
	}, nil
}

// Store stores a unified agent
func (s *AgentStorageAdapter) Store(ctx context.Context, agent *UnifiedAgent) error {
	// Convert agent to JSON
	agentJSON, err := json.Marshal(agent)
	if err != nil {
		return fmt.Errorf("failed to marshal agent: %v", err)
	}

	// Create metadata for search
	metadata := map[string]string{
		"id":           agent.ID,
		"name":         agent.Name,
		"type":         agent.Type,
		"version":      agent.Version,
		"build_target": agent.BuildTarget,
		"status":       agent.Status,
		"collection":   agent.Collection,
		"created_at":   agent.CreatedAt.Format(time.RFC3339),
		"updated_at":   agent.UpdatedAt.Format(time.RFC3339),
	}

	// Add capabilities as metadata
	if len(agent.Capabilities) > 0 {
		capabilitiesJSON, _ := json.Marshal(agent.Capabilities)
		metadata["capabilities"] = string(capabilitiesJSON)
	}

	// Add target types as metadata
	if len(agent.TargetTypes) > 0 {
		targetTypesJSON, _ := json.Marshal(agent.TargetTypes)
		metadata["target_types"] = string(targetTypesJSON)
	}

	// Add tags as metadata
	if len(agent.Tags) > 0 {
		tagsJSON, _ := json.Marshal(agent.Tags)
		metadata["tags"] = string(tagsJSON)
	}

	// Create document for chromem-go
	doc := chromem.Document{
		ID:       agent.ID,
		Content:  string(agentJSON),
		Metadata: metadata,
	}

	// Store agent in chromem-go
	return s.collection.AddDocuments(ctx, []chromem.Document{doc}, 1)
}

// Get retrieves an agent by ID
func (s *AgentStorageAdapter) Get(ctx context.Context, id string) (*UnifiedAgent, error) {
	// Get document by ID
	doc, err := s.collection.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("agent not found: %s", id)
	}

	// Parse the JSON content
	var agent UnifiedAgent
	if err := json.Unmarshal([]byte(doc.Content), &agent); err != nil {
		return nil, fmt.Errorf("failed to unmarshal agent: %v", err)
	}

	return &agent, nil
}

// Update updates an existing agent
func (s *AgentStorageAdapter) Update(ctx context.Context, agent *UnifiedAgent) error {
	// Check if agent exists
	exists, err := s.Exists(ctx, agent.ID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("agent not found: %s", agent.ID)
	}

	// Delete the existing document
	if err := s.collection.Delete(ctx, nil, nil, agent.ID); err != nil {
		return fmt.Errorf("failed to delete existing agent: %v", err)
	}

	// Store the updated agent
	return s.Store(ctx, agent)
}

// Delete deletes an agent by ID
func (s *AgentStorageAdapter) Delete(ctx context.Context, id string) error {
	// Check if agent exists
	exists, err := s.Exists(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("agent not found: %s", id)
	}

	// Delete the document
	return s.collection.Delete(ctx, nil, nil, id)
}

// List lists agents with optional filtering
func (s *AgentStorageAdapter) List(ctx context.Context, filter map[string]interface{}) ([]*UnifiedAgent, error) {
	// Use Query to get all documents (chromem-go doesn't have GetAll)
	// Query with empty string and high limit to get all documents
	results, err := s.collection.Query(ctx, "", 1000, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query agents: %v", err)
	}

	var agents []*UnifiedAgent
	for _, result := range results {
		var agent UnifiedAgent
		if err := json.Unmarshal([]byte(result.Content), &agent); err != nil {
			continue // Skip invalid documents
		}

		// Apply filters
		if s.matchesFilter(&agent, filter) {
			agents = append(agents, &agent)
		}
	}

	return agents, nil
}

// Search searches for agents using semantic search
func (s *AgentStorageAdapter) Search(ctx context.Context, query string, limit int) ([]*UnifiedAgent, error) {
	// Perform semantic search
	results, err := s.collection.Query(ctx, query, limit, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to search agents: %v", err)
	}

	var agents []*UnifiedAgent
	for _, result := range results {
		var agent UnifiedAgent
		if err := json.Unmarshal([]byte(result.Content), &agent); err != nil {
			continue // Skip invalid documents
		}
		agents = append(agents, &agent)
	}

	return agents, nil
}

// Exists checks if an agent exists
func (s *AgentStorageAdapter) Exists(ctx context.Context, id string) (bool, error) {
	_, err := s.collection.GetByID(ctx, id)
	if err != nil {
		return false, nil // Agent doesn't exist
	}
	return true, nil
}

// Count counts agents with optional filtering
func (s *AgentStorageAdapter) Count(ctx context.Context, filter map[string]interface{}) (int, error) {
	agents, err := s.List(ctx, filter)
	if err != nil {
		return 0, err
	}
	return len(agents), nil
}

// matchesFilter checks if an agent matches the given filter
func (s *AgentStorageAdapter) matchesFilter(agent *UnifiedAgent, filter map[string]interface{}) bool {
	if len(filter) == 0 {
		return true
	}

	for key, value := range filter {
		switch key {
		case "type":
			if agent.Type != value.(string) {
				return false
			}
		case "status":
			if agent.Status != value.(string) {
				return false
			}
		case "build_target":
			if agent.BuildTarget != value.(string) {
				return false
			}
		case "collection":
			if agent.Collection != value.(string) {
				return false
			}
		case "owner_id":
			if agent.OwnerID != value.(int64) {
				return false
			}
		case "capabilities":
			requiredCaps := value.([]string)
			if !s.hasAllCapabilities(agent.Capabilities, requiredCaps) {
				return false
			}
		case "target_types":
			requiredTypes := value.([]string)
			if !s.hasAllTargetTypes(agent.TargetTypes, requiredTypes) {
				return false
			}
		}
	}

	return true
}

// hasAllCapabilities checks if the agent has all required capabilities
func (s *AgentStorageAdapter) hasAllCapabilities(agentCaps, requiredCaps []string) bool {
	capMap := make(map[string]bool)
	for _, cap := range agentCaps {
		capMap[cap] = true
	}

	for _, required := range requiredCaps {
		if !capMap[required] {
			return false
		}
	}

	return true
}

// hasAllTargetTypes checks if the agent has all required target types
func (s *AgentStorageAdapter) hasAllTargetTypes(agentTypes, requiredTypes []string) bool {
	typeMap := make(map[string]bool)
	for _, t := range agentTypes {
		typeMap[t] = true
	}

	for _, required := range requiredTypes {
		if !typeMap[required] {
			return false
		}
	}

	return true
}
