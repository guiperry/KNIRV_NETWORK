package main

import (
	"os"
	"path/filepath"
	"testing"

	"KNIRVROOT/config"
)

// TestCreateAIAgent tests the CreateAIAgent method with AgentFacts metadata
func TestCreateAIAgent(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "agent_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize ChromemDB manager
	chromemConfig := &config.ChromemConfig{
		Path: filepath.Join(tempDir, "chromem_test"),
	}

	chromemManager, err := NewChromemManager(chromemConfig)
	if err != nil {
		t.Fatalf("Failed to create ChromemManager: %v", err)
	}
	defer chromemManager.Close()

	// Create test wallet
	wallet, err := NewWallet()
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	// Create agent manager
	agentManager := NewAgentManager(chromemManager, nil, wallet, nil)

	// Test data
	agentName := "Test AI Agent"
	agentDescription := "A test AI agent for Phase 4 testing"
	imageURL := "https://example.com/agent.png"
	owner := "test_owner_address"

	// Create agent profile for AgentFacts
	agentProfile := map[string]interface{}{
		"agent_id":            "did:agent:test:12345",
		"agent_name":          "urn:agent:test:ai-agent",
		"facts_url":           "https://example.com/.agent-facts",
		"adaptive_router_url": "https://router.example.com",
		"ttl":                 3600,
		"signature":           "test_signature",
	}

	// Create AI Agent with AgentFacts
	agent, err := agentManager.CreateAIAgent(
		agentName,
		agentDescription,
		imageURL,
		owner,
		agentProfile,
	)

	if err != nil {
		t.Fatalf("Failed to create AI agent: %v", err)
	}

	// Verify agent was created
	if agent == nil {
		t.Fatal("Agent is nil")
	}

	// Verify basic fields
	if agent.Name != agentName {
		t.Errorf("Expected agent name %s, got %s", agentName, agent.Name)
	}

	if agent.Description != agentDescription {
		t.Errorf("Expected agent description %s, got %s", agentDescription, agent.Description)
	}

	// Note: AgentType is not set by CreateAIAgent, it's set by CreateAgent
	// We'll verify the metadata instead

	if agent.Owner != owner {
		t.Errorf("Expected agent owner %s, got %s", owner, agent.Owner)
	}

	// Verify AgentFacts metadata
	if agent.Metadata == nil {
		t.Fatal("Agent metadata is nil")
	}

	// The metadata IS the AgentFacts structure
	agentFactsMap := agent.Metadata

	// Verify AgentFacts standard fields
	agentID, ok := agentFactsMap["agent_id"].(string)
	if !ok {
		t.Error("Agent ID not found in AgentFacts")
	}
	expectedAgentID := agentProfile["agent_id"].(string)
	if agentID != expectedAgentID {
		t.Errorf("Expected agent ID %s, got %s", expectedAgentID, agentID)
	}

	// Verify agent name
	agentName, ok = agentFactsMap["agent_name"].(string)
	if !ok {
		t.Error("Agent name not found in AgentFacts")
	}
	expectedAgentName := agentProfile["agent_name"].(string)
	if agentName != expectedAgentName {
		t.Errorf("Expected agent name %s, got %s", expectedAgentName, agentName)
	}

	// Verify facts URL
	factsURL, ok := agentFactsMap["facts_url"].(string)
	if !ok {
		t.Error("Facts URL not found in AgentFacts")
	}
	expectedFactsURL := agentProfile["facts_url"].(string)
	if factsURL != expectedFactsURL {
		t.Errorf("Expected facts URL %s, got %s", expectedFactsURL, factsURL)
	}

	// Verify adaptive router URL
	adaptiveRouterURL, ok := agentFactsMap["adaptive_router_url"].(string)
	if !ok {
		t.Error("Adaptive router URL not found in AgentFacts")
	}
	expectedRouterURL := agentProfile["adaptive_router_url"].(string)
	if adaptiveRouterURL != expectedRouterURL {
		t.Errorf("Expected adaptive router URL %s, got %s", expectedRouterURL, adaptiveRouterURL)
	}

	// Verify signature field exists
	signature, ok := agentFactsMap["signature"].(string)
	if !ok {
		t.Error("Signature not found in AgentFacts")
	}
	expectedSignature := agentProfile["signature"].(string)
	if signature != expectedSignature {
		t.Errorf("Expected signature %s, got %s", expectedSignature, signature)
	}

	// Verify agent can be retrieved
	retrievedAgent, err := agentManager.GetAgent(agent.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve agent: %v", err)
	}

	if retrievedAgent.ID != agent.ID {
		t.Errorf("Retrieved agent ID mismatch: expected %s, got %s", agent.ID, retrievedAgent.ID)
	}

	t.Logf("Successfully created AI agent with ID: %s", agent.ID)
}

// TestConfigureAgentCapabilities tests the ConfigureAgentCapabilities method
func TestConfigureAgentCapabilities(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "agent_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize ChromemDB manager
	chromemConfig := &config.ChromemConfig{
		Path: filepath.Join(tempDir, "chromem_test"),
	}

	chromemManager, err := NewChromemManager(chromemConfig)
	if err != nil {
		t.Fatalf("Failed to create ChromemManager: %v", err)
	}
	defer chromemManager.Close()

	// Create test wallet
	wallet, err := NewWallet()
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	// Create agent manager
	agentManager := NewAgentManager(chromemManager, nil, wallet, nil)

	// First create an agent
	agentProfile := map[string]interface{}{
		"agent_id":            "did:agent:test:config",
		"agent_name":          "urn:agent:test:config-agent",
		"facts_url":           "https://example.com/.agent-facts",
		"adaptive_router_url": "https://router.example.com",
		"ttl":                 3600,
		"signature":           "test_signature",
	}

	agent, err := agentManager.CreateAIAgent(
		"Test Agent",
		"Test agent for capability configuration",
		"https://example.com/agent.png",
		"test_owner",
		agentProfile,
	)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Define capabilities to configure
	capabilities := []map[string]interface{}{
		{
			"name":            "Text Processing",
			"capability_type": "tool",
			"description":     "Process and analyze text",
			"metadata": map[string]interface{}{
				"input_type":  "text",
				"output_type": "analysis",
			},
		},
		{
			"name":            "Data Analysis",
			"capability_type": "tool",
			"description":     "Analyze structured data",
			"metadata": map[string]interface{}{
				"input_type":  "data",
				"output_type": "insights",
			},
		},
	}

	// Configure capabilities
	err = agentManager.ConfigureAgentCapabilities(agent.ID, capabilities)
	if err != nil {
		t.Fatalf("Failed to configure agent capabilities: %v", err)
	}

	// Retrieve agent and verify capabilities were added
	updatedAgent, err := agentManager.GetAgent(agent.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve updated agent: %v", err)
	}

	if len(updatedAgent.Capabilities) != len(capabilities) {
		t.Errorf("Expected %d capabilities, got %d", len(capabilities), len(updatedAgent.Capabilities))
	}

	// Verify each capability was added correctly
	for i, expectedCap := range capabilities {
		if i >= len(updatedAgent.Capabilities) {
			t.Errorf("Missing capability at index %d", i)
			continue
		}

		actualCap := updatedAgent.Capabilities[i]
		expectedName := expectedCap["name"].(string)
		expectedType := expectedCap["capability_type"].(string)

		if actualCap.Name != expectedName {
			t.Errorf("Capability %d name mismatch: expected %s, got %s", i, expectedName, actualCap.Name)
		}

		if actualCap.CapabilityType != expectedType {
			t.Errorf("Capability %d type mismatch: expected %s, got %s", i, expectedType, actualCap.CapabilityType)
		}
	}

	t.Logf("Successfully configured %d capabilities for agent %s", len(capabilities), agent.ID)
}

// TestCreateAgentWithCapabilities tests the CreateAgentWithCapabilities convenience method
func TestCreateAgentWithCapabilities(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "agent_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize ChromemDB manager
	chromemConfig := &config.ChromemConfig{
		Path: filepath.Join(tempDir, "chromem_test"),
	}

	chromemManager, err := NewChromemManager(chromemConfig)
	if err != nil {
		t.Fatalf("Failed to create ChromemManager: %v", err)
	}
	defer chromemManager.Close()

	// Create test wallet
	wallet, err := NewWallet()
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	// Create agent manager
	agentManager := NewAgentManager(chromemManager, nil, wallet, nil)

	// Define capabilities
	capabilities := []map[string]interface{}{
		{
			"name":            "Image Processing",
			"capability_type": "tool",
			"description":     "Process and analyze images",
			"metadata": map[string]interface{}{
				"input_type":  "image",
				"output_type": "analysis",
			},
		},
		{
			"name":            "Speech Recognition",
			"capability_type": "tool",
			"description":     "Convert speech to text",
			"metadata": map[string]interface{}{
				"input_type":  "audio",
				"output_type": "text",
			},
		},
	}

	// Create agent profile
	agentProfile := map[string]interface{}{
		"agent_id":            "did:agent:test:multimodal",
		"agent_name":          "urn:agent:test:multimodal-agent",
		"facts_url":           "https://example.com/.agent-facts",
		"adaptive_router_url": "https://router.example.com",
		"ttl":                 3600,
		"signature":           "test_signature",
	}

	// Create agent with capabilities in one operation
	agent, err := agentManager.CreateAgentWithCapabilities(
		"Multi-Modal Agent",
		"Agent with multiple capabilities",
		"https://example.com/agent.png",
		"test_owner",
		agentProfile,
		capabilities,
	)

	if err != nil {
		t.Fatalf("Failed to create agent with capabilities: %v", err)
	}

	// Verify agent was created
	if agent == nil {
		t.Fatal("Agent is nil")
	}

	// Retrieve the agent to get the updated capabilities
	retrievedAgent, err := agentManager.GetAgent(agent.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve agent: %v", err)
	}

	// Verify capabilities were configured
	if len(retrievedAgent.Capabilities) != len(capabilities) {
		t.Errorf("Expected %d capabilities, got %d", len(capabilities), len(retrievedAgent.Capabilities))
	}

	// Verify AgentFacts metadata exists (metadata IS the AgentFacts structure)
	if retrievedAgent.Metadata == nil {
		t.Fatal("Agent metadata is nil")
	}

	// Verify AgentFacts fields exist
	if _, ok := retrievedAgent.Metadata["agent_id"]; !ok {
		t.Error("Agent ID not found in AgentFacts metadata")
	}
	if _, ok := retrievedAgent.Metadata["facts_url"]; !ok {
		t.Error("Facts URL not found in AgentFacts metadata")
	}

	if len(retrievedAgent.Capabilities) != len(capabilities) {
		t.Errorf("Retrieved agent capabilities count mismatch: expected %d, got %d",
			len(capabilities), len(retrievedAgent.Capabilities))
	}

	t.Logf("Successfully created agent with capabilities. Agent ID: %s, Capabilities: %d",
		agent.ID, len(agent.Capabilities))
}
