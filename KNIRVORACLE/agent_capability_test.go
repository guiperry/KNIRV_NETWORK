package main

import (
	"os"
	"testing"
)

// TestAgentManager_AddCapabilityToAgent tests adding capabilities to agents
func TestAgentManager_AddCapabilityToAgent(t *testing.T) {
	// Setup test environment
	tempDir, chromemManager, wallet, agentManager := setupTestEnvironment(t)
	defer os.RemoveAll(tempDir)
	defer chromemManager.Close()

	// Create an agent first
	metadata := map[string]interface{}{"version": "1.0"}
	agent, err := agentManager.CreateAgent(
		"Test Agent",
		"A test agent",
		"",
		wallet.GetAddress(),
		"assistant",
		metadata,
	)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Create a capability to add
	capability := Capability{
		Name:           "Test Capability",
		Description:    "A test capability",
		CapabilityType: "tool",
		MCPServerURL:   "http://localhost:8080",
		Schema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"input": map[string]interface{}{
					"type": "string",
				},
			},
		},
		LocationHints: []string{"local"},
		Metadata: map[string]interface{}{
			"category": "test",
		},
	}

	// Add capability to agent
	err = agentManager.AddCapabilityToAgent(agent.ID, capability)
	if err != nil {
		t.Fatalf("Failed to add capability to agent: %v", err)
	}

	// Retrieve the updated agent
	updatedAgent, err := agentManager.GetAgent(agent.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve updated agent: %v", err)
	}

	// Verify the capability was added
	if len(updatedAgent.Capabilities) != 1 {
		t.Errorf("Expected 1 capability, got %d", len(updatedAgent.Capabilities))
	}

	addedCapability := updatedAgent.Capabilities[0]
	if addedCapability.Name != "Test Capability" {
		t.Errorf("Expected capability name 'Test Capability', got '%s'", addedCapability.Name)
	}
	if addedCapability.CapabilityType != "tool" {
		t.Errorf("Expected capability type 'tool', got '%s'", addedCapability.CapabilityType)
	}
	if addedCapability.Status != "active" {
		t.Errorf("Expected capability status 'active', got '%s'", addedCapability.Status)
	}
	if addedCapability.ID == "" {
		t.Error("Capability ID should not be empty")
	}
	if addedCapability.CreatedAt.IsZero() {
		t.Error("Capability CreatedAt should not be zero")
	}

	// Test adding capability to non-existent agent
	err = agentManager.AddCapabilityToAgent("non-existent-id", capability)
	if err == nil {
		t.Error("Expected error when adding capability to non-existent agent")
	}
}

// TestAgentManager_RemoveCapabilityFromAgent tests removing capabilities from agents
func TestAgentManager_RemoveCapabilityFromAgent(t *testing.T) {
	// Setup test environment
	tempDir, chromemManager, wallet, agentManager := setupTestEnvironment(t)
	defer os.RemoveAll(tempDir)
	defer chromemManager.Close()

	// Create an agent
	metadata := map[string]interface{}{"version": "1.0"}
	agent, err := agentManager.CreateAgent(
		"Test Agent",
		"A test agent",
		"",
		wallet.GetAddress(),
		"assistant",
		metadata,
	)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Add multiple capabilities
	capability1 := Capability{
		Name:           "Capability 1",
		Description:    "First capability",
		CapabilityType: "tool",
		MCPServerURL:   "http://localhost:8080",
		Schema:         map[string]interface{}{"type": "object"},
		LocationHints:  []string{"local"},
		Metadata:       map[string]interface{}{"category": "test"},
	}

	capability2 := Capability{
		Name:           "Capability 2",
		Description:    "Second capability",
		CapabilityType: "service",
		MCPServerURL:   "http://localhost:8081",
		Schema:         map[string]interface{}{"type": "object"},
		LocationHints:  []string{"remote"},
		Metadata:       map[string]interface{}{"category": "test"},
	}

	err = agentManager.AddCapabilityToAgent(agent.ID, capability1)
	if err != nil {
		t.Fatalf("Failed to add capability 1: %v", err)
	}

	err = agentManager.AddCapabilityToAgent(agent.ID, capability2)
	if err != nil {
		t.Fatalf("Failed to add capability 2: %v", err)
	}

	// Get the updated agent to retrieve capability IDs
	updatedAgent, err := agentManager.GetAgent(agent.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve updated agent: %v", err)
	}

	if len(updatedAgent.Capabilities) != 2 {
		t.Fatalf("Expected 2 capabilities, got %d", len(updatedAgent.Capabilities))
	}

	// Remove the first capability
	capabilityToRemove := updatedAgent.Capabilities[0]
	err = agentManager.RemoveCapabilityFromAgent(agent.ID, capabilityToRemove.ID)
	if err != nil {
		t.Fatalf("Failed to remove capability: %v", err)
	}

	// Verify the capability was removed
	finalAgent, err := agentManager.GetAgent(agent.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve final agent: %v", err)
	}

	if len(finalAgent.Capabilities) != 1 {
		t.Errorf("Expected 1 capability after removal, got %d", len(finalAgent.Capabilities))
	}

	// Verify the remaining capability is the correct one
	remainingCapability := finalAgent.Capabilities[0]
	if remainingCapability.ID == capabilityToRemove.ID {
		t.Error("Removed capability still exists")
	}

	// Test removing capability from non-existent agent
	err = agentManager.RemoveCapabilityFromAgent("non-existent-id", capabilityToRemove.ID)
	if err == nil {
		t.Error("Expected error when removing capability from non-existent agent")
	}

	// Test removing non-existent capability
	err = agentManager.RemoveCapabilityFromAgent(agent.ID, "non-existent-capability-id")
	if err != nil {
		t.Fatalf("Removing non-existent capability should not fail: %v", err)
	}
}

// TestAgentManager_UpdateAgent tests updating agent information
func TestAgentManager_UpdateAgent(t *testing.T) {
	// Setup test environment
	tempDir, chromemManager, wallet, agentManager := setupTestEnvironment(t)
	defer os.RemoveAll(tempDir)
	defer chromemManager.Close()

	// Create an agent
	metadata := map[string]interface{}{"version": "1.0"}
	agent, err := agentManager.CreateAgent(
		"Original Agent",
		"Original description",
		"https://example.com/original.png",
		wallet.GetAddress(),
		"assistant",
		metadata,
	)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Update agent properties
	agent.Name = "Updated Agent"
	agent.Description = "Updated description"
	agent.ImageURL = "https://example.com/updated.png"
	agent.Metadata = map[string]interface{}{
		"version": "2.0",
		"updated": true,
	}

	// Update the agent
	err = agentManager.UpdateAgent(agent)
	if err != nil {
		t.Fatalf("Failed to update agent: %v", err)
	}

	// Retrieve the updated agent
	updatedAgent, err := agentManager.GetAgent(agent.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve updated agent: %v", err)
	}

	// Verify the updates
	if updatedAgent.Name != "Updated Agent" {
		t.Errorf("Expected updated name 'Updated Agent', got '%s'", updatedAgent.Name)
	}
	if updatedAgent.Description != "Updated description" {
		t.Errorf("Expected updated description 'Updated description', got '%s'", updatedAgent.Description)
	}
	if updatedAgent.ImageURL != "https://example.com/updated.png" {
		t.Errorf("Expected updated image URL 'https://example.com/updated.png', got '%s'", updatedAgent.ImageURL)
	}

	// Verify metadata was updated
	if version, ok := updatedAgent.Metadata["version"].(string); !ok || version != "2.0" {
		t.Errorf("Expected metadata version '2.0', got '%v'", updatedAgent.Metadata["version"])
	}
	if updated, ok := updatedAgent.Metadata["updated"].(bool); !ok || !updated {
		t.Errorf("Expected metadata updated to be true, got '%v'", updatedAgent.Metadata["updated"])
	}

	// Verify immutable fields weren't changed
	if updatedAgent.ID != agent.ID {
		t.Errorf("Agent ID should not change during update")
	}
	if updatedAgent.Owner != agent.Owner {
		t.Errorf("Agent owner should not change during update")
	}
	if updatedAgent.CreatedAt != agent.CreatedAt {
		t.Errorf("Agent CreatedAt should not change during update")
	}
}

// TestCapabilityLifecycle tests the complete lifecycle of capabilities
func TestCapabilityLifecycle(t *testing.T) {
	// Setup test environment
	tempDir, chromemManager, wallet, agentManager := setupTestEnvironment(t)
	defer os.RemoveAll(tempDir)
	defer chromemManager.Close()

	// Create an agent
	metadata := map[string]interface{}{"version": "1.0"}
	agent, err := agentManager.CreateAgent(
		"Test Agent",
		"A test agent",
		"",
		wallet.GetAddress(),
		"assistant",
		metadata,
	)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Test adding multiple capabilities
	capabilities := []Capability{
		{
			Name:           "File Reader",
			Description:    "Reads files from disk",
			CapabilityType: "tool",
			MCPServerURL:   "http://localhost:8080",
			Schema:         map[string]interface{}{"type": "object"},
			LocationHints:  []string{"local"},
			Metadata:       map[string]interface{}{"category": "io"},
		},
		{
			Name:           "Web Scraper",
			Description:    "Scrapes web pages",
			CapabilityType: "service",
			MCPServerURL:   "http://localhost:8081",
			Schema:         map[string]interface{}{"type": "object"},
			LocationHints:  []string{"remote"},
			Metadata:       map[string]interface{}{"category": "web"},
		},
		{
			Name:           "Data Processor",
			Description:    "Processes data",
			CapabilityType: "processor",
			MCPServerURL:   "http://localhost:8082",
			Schema:         map[string]interface{}{"type": "object"},
			LocationHints:  []string{"local", "remote"},
			Metadata:       map[string]interface{}{"category": "data"},
		},
	}

	// Add all capabilities
	for _, cap := range capabilities {
		err = agentManager.AddCapabilityToAgent(agent.ID, cap)
		if err != nil {
			t.Fatalf("Failed to add capability '%s': %v", cap.Name, err)
		}
	}

	// Verify all capabilities were added
	updatedAgent, err := agentManager.GetAgent(agent.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve agent: %v", err)
	}

	if len(updatedAgent.Capabilities) != 3 {
		t.Errorf("Expected 3 capabilities, got %d", len(updatedAgent.Capabilities))
	}

	// Verify capability properties
	capabilityNames := make(map[string]bool)
	for _, cap := range updatedAgent.Capabilities {
		capabilityNames[cap.Name] = true

		// Verify all capabilities have required fields
		if cap.ID == "" {
			t.Errorf("Capability '%s' has empty ID", cap.Name)
		}
		if cap.CreatedAt.IsZero() {
			t.Errorf("Capability '%s' has zero CreatedAt", cap.Name)
		}
		if cap.Status != "active" {
			t.Errorf("Capability '%s' has status '%s', expected 'active'", cap.Name, cap.Status)
		}
	}

	// Verify all expected capabilities are present
	expectedNames := []string{"File Reader", "Web Scraper", "Data Processor"}
	for _, name := range expectedNames {
		if !capabilityNames[name] {
			t.Errorf("Expected capability '%s' not found", name)
		}
	}

	// Remove one capability and verify
	capabilityToRemove := updatedAgent.Capabilities[1] // Remove Web Scraper
	err = agentManager.RemoveCapabilityFromAgent(agent.ID, capabilityToRemove.ID)
	if err != nil {
		t.Fatalf("Failed to remove capability: %v", err)
	}

	// Verify capability was removed
	finalAgent, err := agentManager.GetAgent(agent.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve final agent: %v", err)
	}

	if len(finalAgent.Capabilities) != 2 {
		t.Errorf("Expected 2 capabilities after removal, got %d", len(finalAgent.Capabilities))
	}

	// Verify the correct capability was removed
	for _, cap := range finalAgent.Capabilities {
		if cap.ID == capabilityToRemove.ID {
			t.Error("Removed capability still exists")
		}
		if cap.Name == "Web Scraper" {
			t.Error("Web Scraper capability should have been removed")
		}
	}
}
