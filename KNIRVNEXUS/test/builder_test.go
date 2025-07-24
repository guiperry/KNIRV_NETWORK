package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
)

const baseURL = "http://localhost:8081/api/v1"

func TestAgentBuilder(t *testing.T) {
	fmt.Println("Testing Agent Builder Integration...")

	// Test 1: Get available templates
	fmt.Println("\n1. Testing GET /templates")
	testGetTemplates()

	// Test 2: Create an agent and build plugin
	fmt.Println("\n2. Testing agent creation and plugin building")
	agentID := testCreateAgent()
	if agentID != "" {
		testBuildAgent(agentID)
		testGetBuildStatus(agentID)
	}

	// Test 3: Test sub-agent functionality
	fmt.Println("\n3. Testing sub-agent functionality")
	if agentID != "" {
		testSubAgents(agentID)
	}

	// Test 4: Get compiled plugins
	fmt.Println("\n4. Testing GET /plugins")
	testGetPlugins()

	fmt.Println("\nAgent Builder Integration Test Complete!")
}

func testGetTemplates() {
	resp, err := http.Get(baseURL + "/templates")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Status: %d\n", resp.StatusCode)
	fmt.Printf("Response: %s\n", string(body))
}

func testCreateAgent() string {
	agentData := map[string]interface{}{
		"name": "Test Agent",
		"type": "standard",
		"config": `{
			"collection": "test",
			"image_url": "",
			"capabilities": ["test"],
			"target_types": [],
			"status": "active"
		}`,
	}

	jsonData, _ := json.Marshal(agentData)
	resp, err := http.Post(baseURL+"/agents", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Error creating agent: %v\n", err)
		return ""
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Create Agent Status: %d\n", resp.StatusCode)
	fmt.Printf("Create Agent Response: %s\n", string(body))

	// Extract agent ID from response
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err == nil {
		if agent, ok := response["agent"].(map[string]interface{}); ok {
			if id, ok := agent["id"].(string); ok {
				return id
			}
		}
	}
	return ""
}

func testBuildAgent(agentID string) {
	buildData := map[string]interface{}{
		"template_id": "standard",
		"config": map[string]interface{}{
			"agent_name":         "Test Agent",
			"agent_description":  "A test agent for integration testing",
			"model":              "gpt-4",
			"instruction":        "You are a helpful test agent.",
			"use_search":         false,
			"use_code_execution": false,
		},
	}

	jsonData, _ := json.Marshal(buildData)
	resp, err := http.Post(baseURL+"/agents/"+agentID+"/build", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Error building agent: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Build Agent Status: %d\n", resp.StatusCode)
	fmt.Printf("Build Agent Response: %s\n", string(body))
}

func testGetBuildStatus(agentID string) {
	resp, err := http.Get(baseURL + "/agents/" + agentID + "/build")
	if err != nil {
		fmt.Printf("Error getting build status: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Build Status: %d\n", resp.StatusCode)
	fmt.Printf("Build Status Response: %s\n", string(body))
}

func testSubAgents(parentID string) {
	// Test spawning a Python sub-agent
	subAgentData := map[string]interface{}{
		"template": "python",
		"config": map[string]interface{}{
			"name": "Python Test Sub-Agent",
		},
	}

	jsonData, _ := json.Marshal(subAgentData)
	resp, err := http.Post(baseURL+"/agents/"+parentID+"/sub-agents", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Error spawning sub-agent: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Spawn Sub-Agent Status: %d\n", resp.StatusCode)
	fmt.Printf("Spawn Sub-Agent Response: %s\n", string(body))

	// Test getting sub-agents
	time.Sleep(100 * time.Millisecond) // Small delay
	resp2, err := http.Get(baseURL + "/agents/" + parentID + "/sub-agents")
	if err != nil {
		fmt.Printf("Error getting sub-agents: %v\n", err)
		return
	}
	defer resp2.Body.Close()

	body2, _ := io.ReadAll(resp2.Body)
	fmt.Printf("Get Sub-Agents Status: %d\n", resp2.StatusCode)
	fmt.Printf("Get Sub-Agents Response: %s\n", string(body2))
}

func testGetPlugins() {
	resp, err := http.Get(baseURL + "/plugins")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Status: %d\n", resp.StatusCode)
	fmt.Printf("Response: %s\n", string(body))
}
