package main

import (
	"os"
	"path/filepath"
	"testing"

	"KNIRVORACLE/config"
)

// TestAgentManager_CreateAgent tests the basic Agent creation functionality
func TestAgentManager_CreateAgent(t *testing.T) {
	// Setup test environment
	tempDir, chromemManager, wallet, agentManager := setupTestEnvironment(t)
	defer os.RemoveAll(tempDir)
	defer chromemManager.Close()

	// Test data
	metadata := map[string]interface{}{
		"version": "1.0",
		"author":  "test",
	}

	// Test creating an agent
	agent, err := agentManager.CreateAgent(
		"Test Agent",
		"A test agent for verification",
		"https://example.com/image.png",
		wallet.GetAddress(),
		"assistant",
		metadata,
	)

	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Verify agent properties
	if agent.Name != "Test Agent" {
		t.Errorf("Expected agent name 'Test Agent', got '%s'", agent.Name)
	}
	if agent.Description != "A test agent for verification" {
		t.Errorf("Expected description 'A test agent for verification', got '%s'", agent.Description)
	}
	if agent.Owner != wallet.GetAddress() {
		t.Errorf("Expected owner '%s', got '%s'", wallet.GetAddress(), agent.Owner)
	}
	if agent.AgentType != "assistant" {
		t.Errorf("Expected agent type 'assistant', got '%s'", agent.AgentType)
	}
	if agent.ID == "" {
		t.Error("Agent ID should not be empty")
	}
	if agent.CreatedAt.IsZero() {
		t.Error("Agent CreatedAt should not be zero")
	}
}

// TestAgentManager_GetAgent tests agent retrieval functionality
func TestAgentManager_GetAgent(t *testing.T) {
	// Setup test environment
	tempDir, chromemManager, wallet, agentManager := setupTestEnvironment(t)
	defer os.RemoveAll(tempDir)
	defer chromemManager.Close()

	// Create an agent first
	metadata := map[string]interface{}{"version": "1.0"}
	agent, err := agentManager.CreateAgent(
		"Test Agent",
		"A test agent",
		"https://example.com/image.png",
		wallet.GetAddress(),
		"assistant",
		metadata,
	)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Test retrieving the agent
	retrievedAgent, err := agentManager.GetAgent(agent.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve agent: %v", err)
	}

	// Verify retrieved agent matches original
	if retrievedAgent.ID != agent.ID {
		t.Errorf("Expected agent ID '%s', got '%s'", agent.ID, retrievedAgent.ID)
	}
	if retrievedAgent.Name != agent.Name {
		t.Errorf("Expected agent name '%s', got '%s'", agent.Name, retrievedAgent.Name)
	}
	if retrievedAgent.Owner != agent.Owner {
		t.Errorf("Expected agent owner '%s', got '%s'", agent.Owner, retrievedAgent.Owner)
	}

	// Test retrieving non-existent agent
	_, err = agentManager.GetAgent("non-existent-id")
	if err == nil {
		t.Error("Expected error when retrieving non-existent agent")
	}
}

// TestAgentManager_GetAgentsByOwner tests retrieving agents by owner
func TestAgentManager_GetAgentsByOwner(t *testing.T) {
	// Setup test environment
	tempDir, chromemManager, wallet, agentManager := setupTestEnvironment(t)
	defer os.RemoveAll(tempDir)
	defer chromemManager.Close()

	// Create multiple agents
	metadata := map[string]interface{}{"version": "1.0"}

	agent1, err := agentManager.CreateAgent("Agent 1", "First agent", "", wallet.GetAddress(), "assistant", metadata)
	if err != nil {
		t.Fatalf("Failed to create agent 1: %v", err)
	}

	agent2, err := agentManager.CreateAgent("Agent 2", "Second agent", "", wallet.GetAddress(), "worker", metadata)
	if err != nil {
		t.Fatalf("Failed to create agent 2: %v", err)
	}

	// Create another wallet and agent for different owner
	wallet2, err := NewWallet()
	if err != nil {
		t.Fatalf("Failed to create second wallet: %v", err)
	}

	_, err = agentManager.CreateAgent("Agent 3", "Third agent", "", wallet2.GetAddress(), "analyzer", metadata)
	if err != nil {
		t.Fatalf("Failed to create agent 3: %v", err)
	}

	// Test retrieving agents by first owner
	agentsByOwner, err := agentManager.GetAgentsByOwner(wallet.GetAddress())
	if err != nil {
		t.Fatalf("Failed to get agents by owner: %v", err)
	}

	if len(agentsByOwner) != 2 {
		t.Errorf("Expected 2 agents for owner, got %d", len(agentsByOwner))
	}

	// Verify the agents belong to the correct owner
	for _, agent := range agentsByOwner {
		if agent.Owner != wallet.GetAddress() {
			t.Errorf("Expected agent owner '%s', got '%s'", wallet.GetAddress(), agent.Owner)
		}
	}

	// Verify agent IDs match
	foundAgent1, foundAgent2 := false, false
	for _, agent := range agentsByOwner {
		if agent.ID == agent1.ID {
			foundAgent1 = true
		}
		if agent.ID == agent2.ID {
			foundAgent2 = true
		}
	}
	if !foundAgent1 || !foundAgent2 {
		t.Error("Not all expected agents found in results")
	}
}

// TestAgentManager_GetAgentsByType tests retrieving agents by type
func TestAgentManager_GetAgentsByType(t *testing.T) {
	// Setup test environment
	tempDir, chromemManager, wallet, agentManager := setupTestEnvironment(t)
	defer os.RemoveAll(tempDir)
	defer chromemManager.Close()

	// Create agents of different types
	metadata := map[string]interface{}{"version": "1.0"}

	agent1, err := agentManager.CreateAgent("Assistant 1", "First assistant", "", wallet.GetAddress(), "assistant", metadata)
	if err != nil {
		t.Fatalf("Failed to create assistant 1: %v", err)
	}

	agent2, err := agentManager.CreateAgent("Assistant 2", "Second assistant", "", wallet.GetAddress(), "assistant", metadata)
	if err != nil {
		t.Fatalf("Failed to create assistant 2: %v", err)
	}

	_, err = agentManager.CreateAgent("Worker 1", "First worker", "", wallet.GetAddress(), "worker", metadata)
	if err != nil {
		t.Fatalf("Failed to create worker: %v", err)
	}

	// Test retrieving assistants
	assistants, err := agentManager.GetAgentsByType("assistant")
	if err != nil {
		t.Fatalf("Failed to get agents by type: %v", err)
	}

	if len(assistants) != 2 {
		t.Errorf("Expected 2 assistants, got %d", len(assistants))
	}

	// Verify all returned agents are assistants
	for _, agent := range assistants {
		if agent.AgentType != "assistant" {
			t.Errorf("Expected agent type 'assistant', got '%s'", agent.AgentType)
		}
	}

	// Verify agent IDs match
	foundAgent1, foundAgent2 := false, false
	for _, agent := range assistants {
		if agent.ID == agent1.ID {
			foundAgent1 = true
		}
		if agent.ID == agent2.ID {
			foundAgent2 = true
		}
	}
	if !foundAgent1 || !foundAgent2 {
		t.Error("Not all expected assistants found in results")
	}

	// Test retrieving workers
	workers, err := agentManager.GetAgentsByType("worker")
	if err != nil {
		t.Fatalf("Failed to get workers by type: %v", err)
	}

	if len(workers) != 1 {
		t.Errorf("Expected 1 worker, got %d", len(workers))
	}
}

// setupTestEnvironment creates a test environment with temporary directories and initialized managers
func setupTestEnvironment(t *testing.T) (string, *ChromemManager, *Wallet, *AgentManager) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "agent_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	// Create ChromeDB config
	chromemConfig := &config.ChromemConfig{
		Path: filepath.Join(tempDir, "chromem_test"),
	}

	// Initialize ChromeManager
	chromemManager, err := NewChromemManager(chromemConfig)
	if err != nil {
		t.Fatalf("Failed to create ChromeManager: %v", err)
	}

	// Create test wallet
	wallet, err := NewWallet()
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	// Create AgentManager
	agentManager := NewAgentManager(chromemManager, nil, wallet, nil)

	return tempDir, chromemManager, wallet, agentManager
}
