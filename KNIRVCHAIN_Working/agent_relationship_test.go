package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"KNIRVCHAIN/config"
)

// compareValues compares two values, handling slices specially
func compareValues(expected, actual interface{}) bool {
	// Use reflect.DeepEqual for comprehensive comparison including slices
	return reflect.DeepEqual(expected, actual)
}

// TestAgentManager_CreateAgentRelationship tests creating relationships between agents
func TestAgentManager_CreateAgentRelationship(t *testing.T) {
	// Setup test environment
	tempDir, chromemManager, wallet, agentManager := setupTestEnvironment(t)
	defer os.RemoveAll(tempDir)
	defer chromemManager.Close()

	// Create two agents
	metadata := map[string]interface{}{"version": "1.0"}

	agent1, err := agentManager.CreateAgent(
		"Source Agent",
		"Source agent for relationship",
		"",
		wallet.GetAddress(),
		"assistant",
		metadata,
	)
	if err != nil {
		t.Fatalf("Failed to create source agent: %v", err)
	}

	agent2, err := agentManager.CreateAgent(
		"Target Agent",
		"Target agent for relationship",
		"",
		wallet.GetAddress(),
		"worker",
		metadata,
	)
	if err != nil {
		t.Fatalf("Failed to create target agent: %v", err)
	}

	// Add a small delay to allow ChromeDB to index the agents
	time.Sleep(100 * time.Millisecond)

	// Create relationship parameters
	relationshipParams := map[string]interface{}{
		"dependency_type": "service",
		"priority":        "high",
		"timeout":         30,
	}

	// Create the relationship
	relationship, err := agentManager.CreateAgentRelationship(
		agent1.ID,
		agent2.ID,
		"dependency",
		relationshipParams,
	)
	if err != nil {
		t.Fatalf("Failed to create agent relationship: %v", err)
	}

	// Verify relationship properties
	if relationship.SourceAgentId != agent1.ID {
		t.Errorf("Expected source agent ID '%s', got '%s'", agent1.ID, relationship.SourceAgentId)
	}
	if relationship.TargetAgentId != agent2.ID {
		t.Errorf("Expected target agent ID '%s', got '%s'", agent2.ID, relationship.TargetAgentId)
	}
	if relationship.RelationType != "dependency" {
		t.Errorf("Expected relation type 'dependency', got '%s'", relationship.RelationType)
	}
	if relationship.Status != "active" {
		t.Errorf("Expected status 'active', got '%s'", relationship.Status)
	}
	if relationship.ID == "" {
		t.Error("Relationship ID should not be empty")
	}
	if relationship.CreatedAt.IsZero() {
		t.Error("Relationship CreatedAt should not be zero")
	}
	if relationship.CreatedBy != wallet.GetAddress() {
		t.Errorf("Expected created by '%s', got '%s'", wallet.GetAddress(), relationship.CreatedBy)
	}

	// Verify parameters
	if depType, ok := relationship.Parameters["dependency_type"].(string); !ok || depType != "service" {
		t.Errorf("Expected dependency_type 'service', got '%v'", relationship.Parameters["dependency_type"])
	}
	if priority, ok := relationship.Parameters["priority"].(string); !ok || priority != "high" {
		t.Errorf("Expected priority 'high', got '%v'", relationship.Parameters["priority"])
	}

	// Test creating relationship with non-existent source agent
	_, err = agentManager.CreateAgentRelationship(
		"non-existent-source",
		agent2.ID,
		"dependency",
		relationshipParams,
	)
	if err == nil {
		t.Error("Expected error when creating relationship with non-existent source agent")
	}

	// Test creating relationship with non-existent target agent
	_, err = agentManager.CreateAgentRelationship(
		agent1.ID,
		"non-existent-target",
		"dependency",
		relationshipParams,
	)
	if err == nil {
		t.Error("Expected error when creating relationship with non-existent target agent")
	}
}

// TestAgentManager_UpdateAgentRelationship tests updating existing relationships
func TestAgentManager_UpdateAgentRelationship(t *testing.T) {
	// Setup test environment
	tempDir, chromemManager, wallet, agentManager := setupTestEnvironment(t)
	defer os.RemoveAll(tempDir)
	defer chromemManager.Close()

	// Create two agents
	metadata := map[string]interface{}{"version": "1.0"}

	agent1, err := agentManager.CreateAgent("Agent 1", "First agent", "", wallet.GetAddress(), "assistant", metadata)
	if err != nil {
		t.Fatalf("Failed to create agent 1: %v", err)
	}

	agent2, err := agentManager.CreateAgent("Agent 2", "Second agent", "", wallet.GetAddress(), "worker", metadata)
	if err != nil {
		t.Fatalf("Failed to create agent 2: %v", err)
	}

	// Add a small delay to allow ChromeDB to index the agents
	time.Sleep(100 * time.Millisecond)

	// Create initial relationship
	initialParams := map[string]interface{}{
		"priority": "low",
		"timeout":  10,
	}

	relationship, err := agentManager.CreateAgentRelationship(
		agent1.ID,
		agent2.ID,
		"collaboration",
		initialParams,
	)
	if err != nil {
		t.Fatalf("Failed to create relationship: %v", err)
	}

	// Update relationship parameters
	updatedParams := map[string]interface{}{
		"priority":    "high",
		"timeout":     60,
		"retry_count": 3,
	}

	err = agentManager.UpdateAgentRelationship(relationship.ID, updatedParams, "active")
	if err != nil {
		t.Fatalf("Failed to update relationship: %v", err)
	}

	// Retrieve the updated relationship
	updatedRelationship, err := agentManager.chromeManager.GetAgentRelationship(relationship.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve updated relationship: %v", err)
	}

	// Verify updates
	if priority, ok := updatedRelationship.Parameters["priority"].(string); !ok || priority != "high" {
		t.Errorf("Expected updated priority 'high', got '%v'", updatedRelationship.Parameters["priority"])
	}
	if timeout, ok := updatedRelationship.Parameters["timeout"].(float64); !ok || timeout != 60 {
		t.Errorf("Expected updated timeout 60, got '%v'", updatedRelationship.Parameters["timeout"])
	}
	if retryCount, ok := updatedRelationship.Parameters["retry_count"].(float64); !ok || retryCount != 3 {
		t.Errorf("Expected retry_count 3, got '%v'", updatedRelationship.Parameters["retry_count"])
	}

	// Verify UpdatedAt and UpdatedBy were set
	if updatedRelationship.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should not be zero after update")
	}
	if updatedRelationship.UpdatedBy != wallet.GetAddress() {
		t.Errorf("Expected updated by '%s', got '%s'", wallet.GetAddress(), updatedRelationship.UpdatedBy)
	}

	// Test updating non-existent relationship
	err = agentManager.UpdateAgentRelationship("non-existent-id", updatedParams, "active")
	if err == nil {
		t.Error("Expected error when updating non-existent relationship")
	}
}

// TestAgentManager_GetAgentRelationships tests retrieving relationships for an agent
func TestAgentManager_GetAgentRelationships(t *testing.T) {
	// Setup test environment
	tempDir, chromemManager, wallet, agentManager := setupTestEnvironment(t)
	defer os.RemoveAll(tempDir)
	defer chromemManager.Close()

	// Create three agents
	metadata := map[string]interface{}{"version": "1.0"}

	agent1, err := agentManager.CreateAgent("Agent 1", "First agent", "", wallet.GetAddress(), "assistant", metadata)
	if err != nil {
		t.Fatalf("Failed to create agent 1: %v", err)
	}

	agent2, err := agentManager.CreateAgent("Agent 2", "Second agent", "", wallet.GetAddress(), "worker", metadata)
	if err != nil {
		t.Fatalf("Failed to create agent 2: %v", err)
	}

	agent3, err := agentManager.CreateAgent("Agent 3", "Third agent", "", wallet.GetAddress(), "analyzer", metadata)
	if err != nil {
		t.Fatalf("Failed to create agent 3: %v", err)
	}

	// Create relationships where agent1 is the source
	params1 := map[string]interface{}{"type": "dependency"}
	relationship1, err := agentManager.CreateAgentRelationship(agent1.ID, agent2.ID, "dependency", params1)
	if err != nil {
		t.Fatalf("Failed to create relationship 1: %v", err)
	}

	params2 := map[string]interface{}{"type": "collaboration"}
	relationship2, err := agentManager.CreateAgentRelationship(agent1.ID, agent3.ID, "collaboration", params2)
	if err != nil {
		t.Fatalf("Failed to create relationship 2: %v", err)
	}

	// Create relationship where agent1 is the target
	params3 := map[string]interface{}{"type": "supervision"}
	relationship3, err := agentManager.CreateAgentRelationship(agent2.ID, agent1.ID, "supervision", params3)
	if err != nil {
		t.Fatalf("Failed to create relationship 3: %v", err)
	}

	// Get all relationships for agent1
	relationships, err := agentManager.GetAgentRelationships(agent1.ID)
	if err != nil {
		t.Fatalf("Failed to get agent relationships: %v", err)
	}

	// Should find 3 relationships (2 as source, 1 as target)
	if len(relationships) != 3 {
		t.Errorf("Expected 3 relationships for agent1, got %d", len(relationships))
	}

	// Verify all expected relationships are present
	foundRelationships := make(map[string]bool)
	for _, rel := range relationships {
		foundRelationships[rel.ID] = true
	}

	if !foundRelationships[relationship1.ID] {
		t.Error("Relationship 1 not found in results")
	}
	if !foundRelationships[relationship2.ID] {
		t.Error("Relationship 2 not found in results")
	}
	if !foundRelationships[relationship3.ID] {
		t.Error("Relationship 3 not found in results")
	}

	// Test getting relationships for agent with no relationships
	agent4, err := agentManager.CreateAgent("Agent 4", "Fourth agent", "", wallet.GetAddress(), "helper", metadata)
	if err != nil {
		t.Fatalf("Failed to create agent 4: %v", err)
	}

	emptyRelationships, err := agentManager.GetAgentRelationships(agent4.ID)
	if err != nil {
		t.Fatalf("Failed to get relationships for agent with no relationships: %v", err)
	}

	if len(emptyRelationships) != 0 {
		t.Errorf("Expected 0 relationships for agent4, got %d", len(emptyRelationships))
	}
}

// TestAgentRelationshipTypes tests different types of relationships
func TestAgentRelationshipTypes(t *testing.T) {
	// Setup test environment
	tempDir, chromemManager, wallet, agentManager := setupTestEnvironment(t)
	defer os.RemoveAll(tempDir)
	defer chromemManager.Close()

	// Create agents
	metadata := map[string]interface{}{"version": "1.0"}

	parent, err := agentManager.CreateAgent("Parent Agent", "Parent agent", "", wallet.GetAddress(), "supervisor", metadata)
	if err != nil {
		t.Fatalf("Failed to create parent agent: %v", err)
	}

	child, err := agentManager.CreateAgent("Child Agent", "Child agent", "", wallet.GetAddress(), "worker", metadata)
	if err != nil {
		t.Fatalf("Failed to create child agent: %v", err)
	}

	peer, err := agentManager.CreateAgent("Peer Agent", "Peer agent", "", wallet.GetAddress(), "assistant", metadata)
	if err != nil {
		t.Fatalf("Failed to create peer agent: %v", err)
	}

	// Test different relationship types
	relationshipTypes := []struct {
		relationType string
		params       map[string]interface{}
	}{
		{
			relationType: "parent-child",
			params: map[string]interface{}{
				"hierarchy_level": 1,
				"permissions":     []string{"read", "execute"},
			},
		},
		{
			relationType: "dependency",
			params: map[string]interface{}{
				"dependency_type": "service",
				"required":        true,
			},
		},
		{
			relationType: "collaboration",
			params: map[string]interface{}{
				"collaboration_type": "peer",
				"shared_resources":   []string{"data", "compute"},
			},
		},
	}

	// Create relationships of different types
	for i, rt := range relationshipTypes {
		var sourceAgent, targetAgent *Agent
		switch i {
		case 0: // parent-child
			sourceAgent, targetAgent = parent, child
		case 1: // dependency
			sourceAgent, targetAgent = child, parent
		case 2: // collaboration
			sourceAgent, targetAgent = parent, peer
		}

		relationship, err := agentManager.CreateAgentRelationship(
			sourceAgent.ID,
			targetAgent.ID,
			rt.relationType,
			rt.params,
		)
		if err != nil {
			t.Fatalf("Failed to create %s relationship: %v", rt.relationType, err)
		}

		// Verify relationship type
		if relationship.RelationType != rt.relationType {
			t.Errorf("Expected relation type '%s', got '%s'", rt.relationType, relationship.RelationType)
		}

		// Verify parameters were stored correctly
		for key, expectedValue := range rt.params {
			if actualValue, exists := relationship.Parameters[key]; !exists {
				t.Errorf("Parameter '%s' not found in %s relationship", key, rt.relationType)
			} else {
				// Handle slice comparison specially since slices are not comparable
				if !compareValues(expectedValue, actualValue) {
					t.Errorf("Parameter '%s' mismatch in %s relationship: expected %v, got %v",
						key, rt.relationType, expectedValue, actualValue)
				}
			}
		}
	}

	// Verify all relationships were created
	parentRelationships, err := agentManager.GetAgentRelationships(parent.ID)
	if err != nil {
		t.Fatalf("Failed to get parent relationships: %v", err)
	}

	if len(parentRelationships) != 3 {
		t.Errorf("Expected 3 relationships for parent agent, got %d", len(parentRelationships))
	}

	// Verify relationship types are present
	foundTypes := make(map[string]bool)
	for _, rel := range parentRelationships {
		foundTypes[rel.RelationType] = true
	}

	expectedTypes := []string{"parent-child", "dependency", "collaboration"}
	for _, expectedType := range expectedTypes {
		if !foundTypes[expectedType] {
			t.Errorf("Expected relationship type '%s' not found", expectedType)
		}
	}
}

// TestAgentRelationshipStatusTransitions tests relationship status changes
func TestAgentRelationshipStatusTransitions(t *testing.T) {
	// Setup test environment
	tempDir, chromemManager, wallet, agentManager := setupTestEnvironment(t)
	defer os.RemoveAll(tempDir)
	defer chromemManager.Close()

	// Create two agents
	metadata := map[string]interface{}{"version": "1.0"}
	agent1, err := agentManager.CreateAgent("Agent 1", "First agent", "", wallet.GetAddress(), "assistant", metadata)
	if err != nil {
		t.Fatalf("Failed to create agent 1: %v", err)
	}

	agent2, err := agentManager.CreateAgent("Agent 2", "Second agent", "", wallet.GetAddress(), "worker", metadata)
	if err != nil {
		t.Fatalf("Failed to create agent 2: %v", err)
	}

	// Create relationship
	params := map[string]interface{}{"type": "test"}
	relationship, err := agentManager.CreateAgentRelationship(agent1.ID, agent2.ID, "dependency", params)
	if err != nil {
		t.Fatalf("Failed to create relationship: %v", err)
	}

	// Test status transitions
	statusTransitions := []string{"active", "inactive", "suspended", "terminated", "active"}

	for _, newStatus := range statusTransitions {
		err = agentManager.UpdateAgentRelationship(relationship.ID, params, newStatus)
		if err != nil {
			t.Fatalf("Failed to update relationship status to '%s': %v", newStatus, err)
		}

		// Verify status was updated
		updatedRel, err := agentManager.chromeManager.GetAgentRelationship(relationship.ID)
		if err != nil {
			t.Fatalf("Failed to retrieve relationship after status update: %v", err)
		}

		if updatedRel.Status != newStatus {
			t.Errorf("Expected status '%s', got '%s'", newStatus, updatedRel.Status)
		}
	}
}

// TestAgentRelationshipParameterUpdates tests updating relationship parameters
func TestAgentRelationshipParameterUpdates(t *testing.T) {
	// Setup test environment
	tempDir, chromemManager, wallet, agentManager := setupTestEnvironment(t)
	defer os.RemoveAll(tempDir)
	defer chromemManager.Close()

	// Create two agents
	metadata := map[string]interface{}{"version": "1.0"}
	agent1, err := agentManager.CreateAgent("Agent 1", "First agent", "", wallet.GetAddress(), "assistant", metadata)
	if err != nil {
		t.Fatalf("Failed to create agent 1: %v", err)
	}

	agent2, err := agentManager.CreateAgent("Agent 2", "Second agent", "", wallet.GetAddress(), "worker", metadata)
	if err != nil {
		t.Fatalf("Failed to create agent 2: %v", err)
	}

	// Create relationship with initial parameters
	initialParams := map[string]interface{}{
		"priority":    "low",
		"timeout":     10,
		"retry_count": 1,
		"config": map[string]interface{}{
			"max_connections": 5,
			"buffer_size":     1024,
		},
	}

	relationship, err := agentManager.CreateAgentRelationship(agent1.ID, agent2.ID, "dependency", initialParams)
	if err != nil {
		t.Fatalf("Failed to create relationship: %v", err)
	}

	// Test parameter updates
	updatedParams := map[string]interface{}{
		"priority":    "high",
		"timeout":     60,
		"retry_count": 5,
		"new_param":   "added_value",
		"config": map[string]interface{}{
			"max_connections": 20,
			"buffer_size":     4096,
			"compression":     true,
		},
	}

	err = agentManager.UpdateAgentRelationship(relationship.ID, updatedParams, "active")
	if err != nil {
		t.Fatalf("Failed to update relationship parameters: %v", err)
	}

	// Verify all parameters were updated
	updatedRel, err := agentManager.chromeManager.GetAgentRelationship(relationship.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve updated relationship: %v", err)
	}

	// Check individual parameters
	if priority, ok := updatedRel.Parameters["priority"].(string); !ok || priority != "high" {
		t.Errorf("Expected priority 'high', got '%v'", updatedRel.Parameters["priority"])
	}

	if timeout, ok := updatedRel.Parameters["timeout"].(float64); !ok || timeout != 60 {
		t.Errorf("Expected timeout 60, got '%v'", updatedRel.Parameters["timeout"])
	}

	if newParam, ok := updatedRel.Parameters["new_param"].(string); !ok || newParam != "added_value" {
		t.Errorf("Expected new_param 'added_value', got '%v'", updatedRel.Parameters["new_param"])
	}

	// Check nested config parameters
	if config, ok := updatedRel.Parameters["config"].(map[string]interface{}); ok {
		if maxConn, ok := config["max_connections"].(float64); !ok || maxConn != 20 {
			t.Errorf("Expected max_connections 20, got '%v'", config["max_connections"])
		}
		if compression, ok := config["compression"].(bool); !ok || !compression {
			t.Errorf("Expected compression true, got '%v'", config["compression"])
		}
	} else {
		t.Error("Config parameter not found or not a map")
	}
}

// TestAgentRelationshipValidation tests relationship validation rules
func TestAgentRelationshipValidation(t *testing.T) {
	// Setup test environment
	tempDir, chromemManager, wallet, agentManager := setupTestEnvironment(t)
	defer os.RemoveAll(tempDir)
	defer chromemManager.Close()

	// Create agents
	metadata := map[string]interface{}{"version": "1.0"}
	agent1, err := agentManager.CreateAgent("Agent 1", "First agent", "", wallet.GetAddress(), "assistant", metadata)
	if err != nil {
		t.Fatalf("Failed to create agent 1: %v", err)
	}

	agent2, err := agentManager.CreateAgent("Agent 2", "Second agent", "", wallet.GetAddress(), "worker", metadata)
	if err != nil {
		t.Fatalf("Failed to create agent 2: %v", err)
	}

	// Test empty relationship type
	_, err = agentManager.CreateAgentRelationship(agent1.ID, agent2.ID, "", map[string]interface{}{})
	if err == nil {
		t.Error("Expected error for empty relationship type")
	}

	// Test self-relationship (agent relating to itself)
	_, err = agentManager.CreateAgentRelationship(agent1.ID, agent1.ID, "self-reference", map[string]interface{}{})
	if err == nil {
		t.Error("Expected error for self-relationship")
	}

	// Test empty agent IDs
	_, err = agentManager.CreateAgentRelationship("", agent2.ID, "dependency", map[string]interface{}{})
	if err == nil {
		t.Error("Expected error for empty source agent ID")
	}

	_, err = agentManager.CreateAgentRelationship(agent1.ID, "", "dependency", map[string]interface{}{})
	if err == nil {
		t.Error("Expected error for empty target agent ID")
	}

	// Test duplicate relationships (same source, target, and type)
	params := map[string]interface{}{"test": "value"}
	rel1, err := agentManager.CreateAgentRelationship(agent1.ID, agent2.ID, "dependency", params)
	if err != nil {
		t.Fatalf("Failed to create first relationship: %v", err)
	}

	// Creating another relationship with same parameters should succeed (different instances)
	rel2, err := agentManager.CreateAgentRelationship(agent1.ID, agent2.ID, "dependency", params)
	if err != nil {
		t.Fatalf("Failed to create second relationship: %v", err)
	}

	// Verify they have different IDs
	if rel1.ID == rel2.ID {
		t.Error("Duplicate relationships should have different IDs")
	}
}

// TestAgentRelationshipConcurrency tests concurrent relationship operations
func TestAgentRelationshipConcurrency(t *testing.T) {
	// Setup test environment
	tempDir, chromemManager, wallet, agentManager := setupTestEnvironment(t)
	defer os.RemoveAll(tempDir)
	defer chromemManager.Close()

	// Create agents
	metadata := map[string]interface{}{"version": "1.0"}
	agent1, err := agentManager.CreateAgent("Agent 1", "First agent", "", wallet.GetAddress(), "assistant", metadata)
	if err != nil {
		t.Fatalf("Failed to create agent 1: %v", err)
	}

	agent2, err := agentManager.CreateAgent("Agent 2", "Second agent", "", wallet.GetAddress(), "worker", metadata)
	if err != nil {
		t.Fatalf("Failed to create agent 2: %v", err)
	}

	// Create initial relationship
	params := map[string]interface{}{"counter": 0}
	relationship, err := agentManager.CreateAgentRelationship(agent1.ID, agent2.ID, "dependency", params)
	if err != nil {
		t.Fatalf("Failed to create relationship: %v", err)
	}

	// Test concurrent updates
	numGoroutines := 10
	done := make(chan bool, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			updateParams := map[string]interface{}{
				"counter":              index,
				"updated_by_goroutine": index,
			}
			err := agentManager.UpdateAgentRelationship(relationship.ID, updateParams, "active")
			if err != nil {
				errors <- err
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		select {
		case <-done:
			// Goroutine completed successfully
		case err := <-errors:
			t.Errorf("Concurrent update failed: %v", err)
		}
	}

	// Verify final state
	finalRel, err := agentManager.chromeManager.GetAgentRelationship(relationship.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve final relationship: %v", err)
	}

	// Should have some counter value (last one to write wins)
	if _, ok := finalRel.Parameters["counter"]; !ok {
		t.Error("Counter parameter should be present after concurrent updates")
	}
}

// TestAgentRelationshipComplexQueries tests complex relationship queries
func TestAgentRelationshipComplexQueries(t *testing.T) {
	// Setup test environment
	tempDir, chromemManager, wallet, agentManager := setupTestEnvironment(t)
	defer os.RemoveAll(tempDir)
	defer chromemManager.Close()

	// Create a network of agents
	metadata := map[string]interface{}{"version": "1.0"}
	agents := make([]*Agent, 5)
	for i := 0; i < 5; i++ {
		agent, err := agentManager.CreateAgent(
			fmt.Sprintf("Agent %d", i+1),
			fmt.Sprintf("Agent number %d", i+1),
			"",
			wallet.GetAddress(),
			"worker",
			metadata,
		)
		if err != nil {
			t.Fatalf("Failed to create agent %d: %v", i+1, err)
		}
		agents[i] = agent
	}

	// Create a complex relationship network
	// Agent 1 -> Agent 2 (dependency)
	// Agent 1 -> Agent 3 (collaboration)
	// Agent 2 -> Agent 4 (parent-child)
	// Agent 3 -> Agent 1 (supervision)
	// Agent 4 -> Agent 5 (extension)
	// Agent 5 -> Agent 1 (feedback)

	relationships := []struct {
		source, target int
		relType        string
		params         map[string]interface{}
	}{
		{0, 1, "dependency", map[string]interface{}{"priority": "high"}},
		{0, 2, "collaboration", map[string]interface{}{"shared_data": true}},
		{1, 3, "parent-child", map[string]interface{}{"hierarchy": 1}},
		{2, 0, "supervision", map[string]interface{}{"oversight": true}},
		{3, 4, "extension", map[string]interface{}{"extends": "functionality"}},
		{4, 0, "feedback", map[string]interface{}{"reports_to": true}},
	}

	createdRelationships := make([]*AgentRelationship, len(relationships))
	for i, rel := range relationships {
		relationship, err := agentManager.CreateAgentRelationship(
			agents[rel.source].ID,
			agents[rel.target].ID,
			rel.relType,
			rel.params,
		)
		if err != nil {
			t.Fatalf("Failed to create relationship %d: %v", i, err)
		}
		createdRelationships[i] = relationship
	}

	// Test querying relationships for Agent 1 (should be involved in 4 relationships)
	agent1Relationships, err := agentManager.GetAgentRelationships(agents[0].ID)
	if err != nil {
		t.Fatalf("Failed to get relationships for Agent 1: %v", err)
	}

	expectedAgent1Count := 4 // 2 as source, 2 as target
	if len(agent1Relationships) != expectedAgent1Count {
		t.Errorf("Expected %d relationships for Agent 1, got %d", expectedAgent1Count, len(agent1Relationships))
	}

	// Verify relationship types for Agent 1
	relationshipTypes := make(map[string]int)
	for _, rel := range agent1Relationships {
		relationshipTypes[rel.RelationType]++
	}

	expectedTypes := map[string]int{
		"dependency":    1,
		"collaboration": 1,
		"supervision":   1,
		"feedback":      1,
	}

	for expectedType, expectedCount := range expectedTypes {
		if actualCount, exists := relationshipTypes[expectedType]; !exists || actualCount != expectedCount {
			t.Errorf("Expected %d '%s' relationships for Agent 1, got %d", expectedCount, expectedType, actualCount)
		}
	}

	// Test querying relationships for Agent 5 (should be involved in 2 relationships)
	agent5Relationships, err := agentManager.GetAgentRelationships(agents[4].ID)
	if err != nil {
		t.Fatalf("Failed to get relationships for Agent 5: %v", err)
	}

	expectedAgent5Count := 2 // 1 as source, 1 as target
	if len(agent5Relationships) != expectedAgent5Count {
		t.Errorf("Expected %d relationships for Agent 5, got %d", expectedAgent5Count, len(agent5Relationships))
	}
}

// TestAgentRelationshipPersistence tests relationship persistence across manager restarts
func TestAgentRelationshipPersistence(t *testing.T) {
	// Setup test environment
	tempDir, chromemManager, wallet, agentManager := setupTestEnvironment(t)
	defer os.RemoveAll(tempDir)
	defer chromemManager.Close()

	// Create agents
	metadata := map[string]interface{}{"version": "1.0"}
	agent1, err := agentManager.CreateAgent("Agent 1", "First agent", "", wallet.GetAddress(), "assistant", metadata)
	if err != nil {
		t.Fatalf("Failed to create agent 1: %v", err)
	}

	agent2, err := agentManager.CreateAgent("Agent 2", "Second agent", "", wallet.GetAddress(), "worker", metadata)
	if err != nil {
		t.Fatalf("Failed to create agent 2: %v", err)
	}

	// Create relationship with complex parameters
	params := map[string]interface{}{
		"priority":    "high",
		"timeout":     60,
		"retry_count": 3,
		"metadata": map[string]interface{}{
			"created_by_test": true,
			"test_timestamp":  time.Now().Unix(),
		},
	}

	relationship, err := agentManager.CreateAgentRelationship(agent1.ID, agent2.ID, "dependency", params)
	if err != nil {
		t.Fatalf("Failed to create relationship: %v", err)
	}

	originalID := relationship.ID
	originalCreatedAt := relationship.CreatedAt

	// Close the current manager
	chromemManager.Close()

	// Create a new manager with the same database
	chromemConfig := &config.ChromemConfig{
		Path: filepath.Join(tempDir, "chromem_test"),
	}

	newChromemManager, err := NewChromemManager(chromemConfig)
	if err != nil {
		t.Fatalf("Failed to create new ChromeManager: %v", err)
	}
	defer newChromemManager.Close()

	newAgentManager := &AgentManager{
		chromeManager: newChromemManager,
	}

	// Retrieve the relationship with the new manager
	retrievedRel, err := newAgentManager.chromeManager.GetAgentRelationship(originalID)
	if err != nil {
		t.Fatalf("Failed to retrieve relationship with new manager: %v", err)
	}

	// Verify all properties persisted correctly
	if retrievedRel.ID != originalID {
		t.Errorf("Expected ID '%s', got '%s'", originalID, retrievedRel.ID)
	}

	if retrievedRel.SourceAgentId != agent1.ID {
		t.Errorf("Expected source agent ID '%s', got '%s'", agent1.ID, retrievedRel.SourceAgentId)
	}

	if retrievedRel.TargetAgentId != agent2.ID {
		t.Errorf("Expected target agent ID '%s', got '%s'", agent2.ID, retrievedRel.TargetAgentId)
	}

	if retrievedRel.RelationType != "dependency" {
		t.Errorf("Expected relation type 'dependency', got '%s'", retrievedRel.RelationType)
	}

	if !retrievedRel.CreatedAt.Equal(originalCreatedAt) {
		t.Errorf("Expected created at '%v', got '%v'", originalCreatedAt, retrievedRel.CreatedAt)
	}

	// Verify complex parameters persisted
	if priority, ok := retrievedRel.Parameters["priority"].(string); !ok || priority != "high" {
		t.Errorf("Expected priority 'high', got '%v'", retrievedRel.Parameters["priority"])
	}

	if metadata, ok := retrievedRel.Parameters["metadata"].(map[string]interface{}); ok {
		if createdByTest, ok := metadata["created_by_test"].(bool); !ok || !createdByTest {
			t.Errorf("Expected created_by_test true, got '%v'", metadata["created_by_test"])
		}
	} else {
		t.Error("Metadata parameter not found or not a map")
	}
}
