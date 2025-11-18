package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"KNIRVCHAIN/config"
)

// TestHandleGetAgentFacts tests the /agent/agent-facts/{id} endpoint
func TestHandleGetAgentFacts(t *testing.T) {
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

	// Create blockchain with agent manager
	blockchain := &BlockchainStruct{}
	blockchain.SetAgentManager(agentManager)

	// Create blockchain server
	server := &BlockchainServer{
		BlockchainPtr: blockchain,
	}

	// Create agent profile
	agentProfile := map[string]interface{}{
		"agent_id":            "did:agent:test:api",
		"agent_name":          "urn:agent:test:api-agent",
		"facts_url":           "https://example.com/.agent-facts",
		"adaptive_router_url": "https://router.example.com",
		"ttl":                 3600,
		"signature":           "test_signature",
	}

	// Create an AI agent first
	agent, err := agentManager.CreateAIAgent(
		"Test Agent",
		"Test agent for API testing",
		"https://example.com/agent.png",
		"test_owner",
		agentProfile,
	)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Test successful request
	req := httptest.NewRequest("GET", fmt.Sprintf("/agent/agent-facts/%s", agent.ID), nil)
	w := httptest.NewRecorder()

	server.HandleGetAgentFacts(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Parse response
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// The response IS the AgentFacts metadata structure
	agentFactsMap := response

	// Verify key AgentFacts fields
	if _, ok := agentFactsMap["agent_id"]; !ok {
		t.Error("Agent ID not found in AgentFacts")
	}

	if _, ok := agentFactsMap["agent_name"]; !ok {
		t.Error("Agent name not found in AgentFacts")
	}

	if _, ok := agentFactsMap["facts_url"]; !ok {
		t.Error("Facts URL not found in AgentFacts")
	}

	if _, ok := agentFactsMap["adaptive_router_url"]; !ok {
		t.Error("Adaptive router URL not found in AgentFacts")
	}

	if _, ok := agentFactsMap["signature"]; !ok {
		t.Error("Signature not found in AgentFacts")
	}

	// Test non-existent agent
	req = httptest.NewRequest("GET", "/agent/agent-facts/nonexistent", nil)
	w = httptest.NewRecorder()

	server.HandleGetAgentFacts(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d for non-existent agent, got %d", http.StatusNotFound, w.Code)
	}

	t.Logf("Successfully tested GetAgentFacts API endpoint")
}

// TestHandleGetAgentCapabilities tests the /agent/capabilities/{agentId} endpoint
func TestHandleGetAgentCapabilities(t *testing.T) {
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

	// Create blockchain with agent manager
	blockchain := &BlockchainStruct{}
	blockchain.SetAgentManager(agentManager)

	// Create blockchain server
	server := &BlockchainServer{
		BlockchainPtr: blockchain,
	}

	// Create capabilities
	capabilities := []map[string]interface{}{
		{
			"name":            "Text Analysis",
			"capability_type": "tool",
			"description":     "Analyze text content",
			"metadata": map[string]interface{}{
				"input_type":  "text",
				"output_type": "analysis",
			},
		},
		{
			"name":            "Data Processing",
			"capability_type": "tool",
			"description":     "Process structured data",
			"metadata": map[string]interface{}{
				"input_type":  "data",
				"output_type": "results",
			},
		},
	}

	// Create agent profile
	agentProfile := map[string]interface{}{
		"agent_id":            "did:agent:test:capabilities",
		"agent_name":          "urn:agent:test:capabilities-agent",
		"facts_url":           "https://example.com/.agent-facts",
		"adaptive_router_url": "https://router.example.com",
		"ttl":                 3600,
		"signature":           "test_signature",
	}

	// Create agent with capabilities
	agent, err := agentManager.CreateAgentWithCapabilities(
		"Test Agent",
		"Test agent for capabilities API testing",
		"https://example.com/agent.png",
		"test_owner",
		agentProfile,
		capabilities,
	)
	if err != nil {
		t.Fatalf("Failed to create agent with capabilities: %v", err)
	}

	// Test successful request
	req := httptest.NewRequest("GET", fmt.Sprintf("/agent/capabilities/%s", agent.ID), nil)
	w := httptest.NewRecorder()

	server.HandleGetAgentCapabilities(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Parse response
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify response structure
	caps, ok := response["capabilities"]
	if !ok {
		t.Error("Capabilities not found in response")
	}

	capsList, ok := caps.([]interface{})
	if !ok {
		t.Error("Capabilities is not a list")
	}

	if len(capsList) != len(capabilities) {
		t.Errorf("Expected %d capabilities, got %d", len(capabilities), len(capsList))
	}

	// Verify first capability
	if len(capsList) > 0 {
		firstCap, ok := capsList[0].(map[string]interface{})
		if !ok {
			t.Error("First capability is not a map")
		} else {
			expectedName := capabilities[0]["name"].(string)
			if firstCap["name"] != expectedName {
				t.Errorf("Expected capability name %s, got %s", expectedName, firstCap["name"])
			}
		}
	}

	// Test non-existent agent
	req = httptest.NewRequest("GET", "/agent/capabilities/nonexistent", nil)
	w = httptest.NewRecorder()

	server.HandleGetAgentCapabilities(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d for non-existent agent, got %d", http.StatusNotFound, w.Code)
	}

	t.Logf("Successfully tested GetAgentCapabilities API endpoint")
}

// TestHandleInvokeAgentCapability tests the /agent/capability/invoke endpoint
func TestHandleInvokeAgentCapability(t *testing.T) {
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

	// Create blockchain with agent manager
	blockchain := &BlockchainStruct{}
	blockchain.SetAgentManager(agentManager)

	// Create blockchain server
	server := &BlockchainServer{
		BlockchainPtr: blockchain,
	}

	// Create capability
	capabilities := []map[string]interface{}{
		{
			"name":            "Test Capability",
			"capability_type": "tool",
			"description":     "A test capability",
			"metadata": map[string]interface{}{
				"input_type":  "text",
				"output_type": "response",
			},
		},
	}

	// Create agent profile
	agentProfile := map[string]interface{}{
		"agent_id":            "did:agent:test:invoke",
		"agent_name":          "urn:agent:test:invoke-agent",
		"facts_url":           "https://example.com/.agent-facts",
		"adaptive_router_url": "https://router.example.com",
		"ttl":                 3600,
		"signature":           "test_signature",
	}

	// Create agent with capability
	agent, err := agentManager.CreateAgentWithCapabilities(
		"Test Agent",
		"Test agent for capability invocation",
		"https://example.com/agent.png",
		"test_owner",
		agentProfile,
		capabilities,
	)
	if err != nil {
		t.Fatalf("Failed to create agent with capabilities: %v", err)
	}

	// Retrieve the agent to get the actual capability ID
	retrievedAgent, err := agentManager.GetAgent(agent.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve agent: %v", err)
	}

	// Get the first capability ID
	var capabilityID string
	if len(retrievedAgent.Capabilities) > 0 {
		capabilityID = retrievedAgent.Capabilities[0].ID
	} else {
		t.Fatal("No capabilities found on agent")
	}

	// Test successful invocation
	requestBody := map[string]interface{}{
		"agent_id":      agent.ID,
		"capability_id": capabilityID,
		"input_data": map[string]interface{}{
			"text": "Hello, world!",
		},
		"context": map[string]interface{}{
			"user_id": "test_user",
		},
	}

	requestJSON, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	req := httptest.NewRequest("POST", "/agent/capability/invoke", bytes.NewReader(requestJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.HandleInvokeAgentCapability(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Response: %s", http.StatusOK, w.Code, w.Body.String())
	}

	// Parse response
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify response structure
	if _, ok := response["context_record_id"]; !ok {
		t.Error("Context record ID not found in response")
	}

	if _, ok := response["status"]; !ok {
		t.Error("Status not found in response")
	}

	if _, ok := response["details"]; !ok {
		t.Error("Details not found in response")
	}

	if _, ok := response["output"]; !ok {
		t.Error("Output not found in response")
	}

	// Test invalid JSON
	req = httptest.NewRequest("POST", "/agent/capability/invoke", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	server.HandleInvokeAgentCapability(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for invalid JSON, got %d", http.StatusBadRequest, w.Code)
	}

	// Test missing required fields
	incompleteRequest := map[string]interface{}{
		"agent_id": agent.ID,
		// Missing capability_id
	}

	incompleteJSON, _ := json.Marshal(incompleteRequest)
	req = httptest.NewRequest("POST", "/agent/capability/invoke", bytes.NewReader(incompleteJSON))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	server.HandleInvokeAgentCapability(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for missing fields, got %d", http.StatusBadRequest, w.Code)
	}

	t.Logf("Successfully tested InvokeAgentCapability API endpoint")
}
