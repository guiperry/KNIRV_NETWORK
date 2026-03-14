// test_agent_http_api.go
package agentify_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"KNIRVCHAIN/internal/inference/agentify"
)

func TestAgentHTTPAPI(t *testing.T) {
	// Create a new Agent Inferencer
	inferencer := agentify.NewAgentInferencer("./plugins")

	// Create a new HTTP API
	api := agentify.NewAgentHTTPAPI(inferencer)

	// Create a test server
	mux := http.NewServeMux()
	api.RegisterHandlers(mux)
	server := httptest.NewServer(mux)
	defer server.Close()

	// Test listing agents
	t.Run("ListAgents", func(t *testing.T) {
		// Create a request
		req, err := http.NewRequest("GET", server.URL+"/v1/agents", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		// Set the API key
		req.Header.Set("Authorization", "Bearer test-api-key")

		// Send the request
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		// Check the response
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Unexpected status code: %d", resp.StatusCode)
		}

		// Parse the response
		var response struct {
			Agents []string `json:"agents"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}
	})

	// Test activating an agent
	t.Run("ActivateAgent", func(t *testing.T) {
		// Create the request body
		body := map[string]interface{}{
			"agentId":   "example",
			"version":   "1.0",
			"sessionId": "test-session",
		}
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("Failed to marshal request body: %v", err)
		}

		// Create a request
		req, err := http.NewRequest("POST", server.URL+"/v1/agents/activate", bytes.NewBuffer(bodyBytes))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		// Set the API key and content type
		req.Header.Set("Authorization", "Bearer test-api-key")
		req.Header.Set("Content-Type", "application/json")

		// Send the request
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		// Check the response
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Unexpected status code: %d", resp.StatusCode)
		}

		// Parse the response
		var response struct {
			Status    string `json:"status"`
			AgentID   string `json:"agentId"`
			SessionID string `json:"sessionId"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Check the response
		if response.Status != "activated" {
			t.Fatalf("Unexpected status: %s", response.Status)
		}
		if response.AgentID != "example" {
			t.Fatalf("Unexpected agent ID: %s", response.AgentID)
		}
		if response.SessionID != "test-session" {
			t.Fatalf("Unexpected session ID: %s", response.SessionID)
		}
	})

	// Test processing an inference request
	t.Run("ProcessInference", func(t *testing.T) {
		// Create the request body
		body := map[string]interface{}{
			"sessionId": "test-session",
			"input":     "Hello, world!",
		}
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("Failed to marshal request body: %v", err)
		}

		// Create a request
		req, err := http.NewRequest("POST", server.URL+"/v1/inference", bytes.NewBuffer(bodyBytes))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		// Set the API key and content type
		req.Header.Set("Authorization", "Bearer test-api-key")
		req.Header.Set("Content-Type", "application/json")

		// Send the request
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		// Check the response
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Unexpected status code: %d", resp.StatusCode)
		}

		// Parse the response
		var response agentify.InferenceResponse
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}
	})

	// Test getting agent capabilities
	t.Run("GetAgentCapabilities", func(t *testing.T) {
		// Create a request
		req, err := http.NewRequest("GET", server.URL+"/v1/capabilities?sessionId=test-session", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		// Set the API key
		req.Header.Set("Authorization", "Bearer test-api-key")

		// Send the request
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		// Check the response
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Unexpected status code: %d", resp.StatusCode)
		}

		// Parse the response
		var response agentify.AgentCapabilities
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}
	})

	// Test getting agent schema
	t.Run("GetAgentSchema", func(t *testing.T) {
		// Create a request
		req, err := http.NewRequest("GET", server.URL+"/v1/schema?sessionId=test-session", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		// Set the API key
		req.Header.Set("Authorization", "Bearer test-api-key")

		// Send the request
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		// Check the response
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Unexpected status code: %d", resp.StatusCode)
		}

		// Parse the response
		var response agentify.AgentSchema
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}
	})

	// Test setting and getting memory
	t.Run("Memory", func(t *testing.T) {
		// Create the request body for setting memory
		body := map[string]interface{}{
			"value": "test-value",
		}
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("Failed to marshal request body: %v", err)
		}

		// Create a request to set memory
		req, err := http.NewRequest("POST", server.URL+"/v1/memory?sessionId=test-session&key=test-key", bytes.NewBuffer(bodyBytes))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		// Set the API key and content type
		req.Header.Set("Authorization", "Bearer test-api-key")
		req.Header.Set("Content-Type", "application/json")

		// Send the request
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		// Check the response
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Unexpected status code: %d", resp.StatusCode)
		}

		// Create a request to get memory
		req, err = http.NewRequest("GET", server.URL+"/v1/memory?sessionId=test-session&key=test-key", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		// Set the API key
		req.Header.Set("Authorization", "Bearer test-api-key")

		// Send the request
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		// Check the response
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Unexpected status code: %d", resp.StatusCode)
		}

		// Parse the response
		var response struct {
			Key   string      `json:"key"`
			Value interface{} `json:"value"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Check the response
		if response.Key != "test-key" {
			t.Fatalf("Unexpected key: %s", response.Key)
		}
		if response.Value != "test-value" {
			t.Fatalf("Unexpected value: %v", response.Value)
		}
	})

	// Test getting TEE info
	t.Run("GetTEEInfo", func(t *testing.T) {
		// Create a request
		req, err := http.NewRequest("GET", server.URL+"/v1/tee?sessionId=test-session", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		// Set the API key
		req.Header.Set("Authorization", "Bearer test-api-key")

		// Send the request
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		// Check the response
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Unexpected status code: %d", resp.StatusCode)
		}

		// Parse the response
		var response map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}
	})

	// Test deactivating an agent
	t.Run("DeactivateAgent", func(t *testing.T) {
		// Create the request body
		body := map[string]interface{}{
			"sessionId": "test-session",
		}
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("Failed to marshal request body: %v", err)
		}

		// Create a request
		req, err := http.NewRequest("POST", server.URL+"/v1/agents/deactivate", bytes.NewBuffer(bodyBytes))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		// Set the API key and content type
		req.Header.Set("Authorization", "Bearer test-api-key")
		req.Header.Set("Content-Type", "application/json")

		// Send the request
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		// Check the response
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Unexpected status code: %d", resp.StatusCode)
		}

		// Parse the response
		var response struct {
			Status    string `json:"status"`
			SessionID string `json:"sessionId"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Check the response
		if response.Status != "deactivated" {
			t.Fatalf("Unexpected status: %s", response.Status)
		}
		if response.SessionID != "test-session" {
			t.Fatalf("Unexpected session ID: %s", response.SessionID)
		}
	})
}
