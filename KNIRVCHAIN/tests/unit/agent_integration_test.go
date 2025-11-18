package main

import (
	"os"
	"testing"
)

// TestAgentSystemIntegration tests the complete Agent system integration
func TestAgentSystemIntegration(t *testing.T) {
	// Setup test environment
	tempDir, chromemManager, wallet, agentManager := setupTestEnvironment(t)
	defer os.RemoveAll(tempDir)
	defer chromemManager.Close()

	// Test scenario: Create a complete Agent ecosystem with capabilities, relationships, and badges

	// Step 1: Create multiple agents with different types
	metadata := map[string]interface{}{"version": "1.0"}

	// Create a supervisor agent
	supervisor, err := agentManager.CreateAgent(
		"Supervisor Agent",
		"Manages and coordinates other agents",
		"https://example.com/supervisor.png",
		wallet.GetAddress(),
		"supervisor",
		metadata,
	)
	if err != nil {
		t.Fatalf("Failed to create supervisor agent: %v", err)
	}

	// Create worker agents
	worker1, err := agentManager.CreateAgent(
		"Data Worker",
		"Processes data and generates reports",
		"https://example.com/worker1.png",
		wallet.GetAddress(),
		"worker",
		metadata,
	)
	if err != nil {
		t.Fatalf("Failed to create worker1: %v", err)
	}

	worker2, err := agentManager.CreateAgent(
		"Analysis Worker",
		"Performs data analysis and insights",
		"https://example.com/worker2.png",
		wallet.GetAddress(),
		"worker",
		metadata,
	)
	if err != nil {
		t.Fatalf("Failed to create worker2: %v", err)
	}

	// Step 2: Add capabilities to agents
	dataProcessingCapability := Capability{
		Name:           "Data Processing",
		Description:    "Processes large datasets",
		CapabilityType: "processor",
		MCPServerURL:   "http://localhost:8080",
		Schema:         map[string]interface{}{"type": "object"},
		LocationHints:  []string{"local"},
		Metadata:       map[string]interface{}{"category": "data"},
	}

	analysisCapability := Capability{
		Name:           "Statistical Analysis",
		Description:    "Performs statistical analysis",
		CapabilityType: "analyzer",
		MCPServerURL:   "http://localhost:8081",
		Schema:         map[string]interface{}{"type": "object"},
		LocationHints:  []string{"local"},
		Metadata:       map[string]interface{}{"category": "analysis"},
	}

	coordinationCapability := Capability{
		Name:           "Agent Coordination",
		Description:    "Coordinates multiple agents",
		CapabilityType: "coordinator",
		MCPServerURL:   "http://localhost:8082",
		Schema:         map[string]interface{}{"type": "object"},
		LocationHints:  []string{"local"},
		Metadata:       map[string]interface{}{"category": "management"},
	}

	// Add capabilities to appropriate agents
	err = agentManager.AddCapabilityToAgent(worker1.ID, dataProcessingCapability)
	if err != nil {
		t.Fatalf("Failed to add data processing capability: %v", err)
	}

	err = agentManager.AddCapabilityToAgent(worker2.ID, analysisCapability)
	if err != nil {
		t.Fatalf("Failed to add analysis capability: %v", err)
	}

	err = agentManager.AddCapabilityToAgent(supervisor.ID, coordinationCapability)
	if err != nil {
		t.Fatalf("Failed to add coordination capability: %v", err)
	}

	// Step 3: Create relationships between agents
	// Supervisor manages both workers
	supervisorWorker1Params := map[string]interface{}{
		"management_level": "direct",
		"priority":         "high",
	}
	relationship1, err := agentManager.CreateAgentRelationship(
		supervisor.ID,
		worker1.ID,
		"management",
		supervisorWorker1Params,
	)
	if err != nil {
		t.Fatalf("Failed to create supervisor-worker1 relationship: %v", err)
	}

	supervisorWorker2Params := map[string]interface{}{
		"management_level": "direct",
		"priority":         "high",
	}
	_, err = agentManager.CreateAgentRelationship(
		supervisor.ID,
		worker2.ID,
		"management",
		supervisorWorker2Params,
	)
	if err != nil {
		t.Fatalf("Failed to create supervisor-worker2 relationship: %v", err)
	}

	// Workers collaborate with each other
	collaborationParams := map[string]interface{}{
		"collaboration_type": "data_pipeline",
		"data_flow":          "worker1_to_worker2",
	}
	_, err = agentManager.CreateAgentRelationship(
		worker1.ID,
		worker2.ID,
		"collaboration",
		collaborationParams,
	)
	if err != nil {
		t.Fatalf("Failed to create worker collaboration relationship: %v", err)
	}

	// Step 4: Attach badges to agents
	performanceBadgeParams := map[string]interface{}{
		"reason":   "Excellent performance in Q1",
		"score":    95,
		"category": "performance",
		"quarter":  "Q1_2024",
	}
	_, err = agentManager.AttachBadgeToAgent(worker1.ID, "badge_performance_q1", performanceBadgeParams)
	if err != nil {
		t.Fatalf("Failed to attach performance badge to worker1: %v", err)
	}

	leadershipBadgeParams := map[string]interface{}{
		"reason":           "Outstanding leadership and coordination",
		"leadership_score": 98,
		"category":         "leadership",
	}
	_, err = agentManager.AttachBadgeToAgent(supervisor.ID, "badge_leadership_001", leadershipBadgeParams)
	if err != nil {
		t.Fatalf("Failed to attach leadership badge to supervisor: %v", err)
	}

	// Step 5: Verify the complete system state

	// Verify agents exist and have correct properties
	retrievedSupervisor, err := agentManager.GetAgent(supervisor.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve supervisor: %v", err)
	}
	if len(retrievedSupervisor.Capabilities) != 1 {
		t.Errorf("Supervisor should have 1 capability, got %d", len(retrievedSupervisor.Capabilities))
	}
	if len(retrievedSupervisor.AttachedBadges) != 1 {
		t.Errorf("Supervisor should have 1 badge, got %d", len(retrievedSupervisor.AttachedBadges))
	}

	retrievedWorker1, err := agentManager.GetAgent(worker1.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve worker1: %v", err)
	}
	if len(retrievedWorker1.Capabilities) != 1 {
		t.Errorf("Worker1 should have 1 capability, got %d", len(retrievedWorker1.Capabilities))
	}
	if len(retrievedWorker1.AttachedBadges) != 1 {
		t.Errorf("Worker1 should have 1 badge, got %d", len(retrievedWorker1.AttachedBadges))
	}

	retrievedWorker2, err := agentManager.GetAgent(worker2.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve worker2: %v", err)
	}
	if len(retrievedWorker2.Capabilities) != 1 {
		t.Errorf("Worker2 should have 1 capability, got %d", len(retrievedWorker2.Capabilities))
	}

	// Verify relationships exist
	supervisorRelationships, err := agentManager.GetAgentRelationships(supervisor.ID)
	if err != nil {
		t.Fatalf("Failed to get supervisor relationships: %v", err)
	}
	if len(supervisorRelationships) != 2 {
		t.Errorf("Supervisor should have 2 relationships, got %d", len(supervisorRelationships))
	}

	worker1Relationships, err := agentManager.GetAgentRelationships(worker1.ID)
	if err != nil {
		t.Fatalf("Failed to get worker1 relationships: %v", err)
	}
	if len(worker1Relationships) != 2 {
		t.Errorf("Worker1 should have 2 relationships (1 as source, 1 as target), got %d", len(worker1Relationships))
	}

	worker2Relationships, err := agentManager.GetAgentRelationships(worker2.ID)
	if err != nil {
		t.Fatalf("Failed to get worker2 relationships: %v", err)
	}
	if len(worker2Relationships) != 2 {
		t.Errorf("Worker2 should have 2 relationships (1 as source, 1 as target), got %d", len(worker2Relationships))
	}

	// Step 6: Test querying by owner and type
	agentsByOwner, err := agentManager.GetAgentsByOwner(wallet.GetAddress())
	if err != nil {
		t.Fatalf("Failed to get agents by owner: %v", err)
	}
	if len(agentsByOwner) != 3 {
		t.Errorf("Should have 3 agents for owner, got %d", len(agentsByOwner))
	}

	workerAgents, err := agentManager.GetAgentsByType("worker")
	if err != nil {
		t.Fatalf("Failed to get worker agents: %v", err)
	}
	if len(workerAgents) != 2 {
		t.Errorf("Should have 2 worker agents, got %d", len(workerAgents))
	}

	supervisorAgents, err := agentManager.GetAgentsByType("supervisor")
	if err != nil {
		t.Fatalf("Failed to get supervisor agents: %v", err)
	}
	if len(supervisorAgents) != 1 {
		t.Errorf("Should have 1 supervisor agent, got %d", len(supervisorAgents))
	}

	// Step 7: Test updating relationships
	updatedParams := map[string]interface{}{
		"management_level": "indirect",
		"priority":         "medium",
		"delegation_level": "high",
	}
	err = agentManager.UpdateAgentRelationship(relationship1.ID, updatedParams, "active")
	if err != nil {
		t.Fatalf("Failed to update relationship: %v", err)
	}

	// Verify relationship was updated
	updatedRelationship, err := agentManager.GetChromeManager().GetAgentRelationship(relationship1.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve updated relationship: %v", err)
	}
	if level, ok := updatedRelationship.Parameters["management_level"].(string); !ok || level != "indirect" {
		t.Errorf("Relationship parameter not updated correctly")
	}

	t.Logf("Integration test completed successfully!")
	t.Logf("Created %d agents, %d relationships, %d capabilities, %d badge attachments",
		len(agentsByOwner), 3, 3, 2)
}
