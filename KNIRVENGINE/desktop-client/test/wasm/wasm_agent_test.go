package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testServerURL = "http://localhost:8081"
	testTimeout   = 30 * time.Second
)

// TestWASMAgentLifecycle tests the complete lifecycle of a WASM agent
func TestWASMAgentLifecycle(t *testing.T) {
	// Test data
	agentID := "wasm_test_agent"
	sessionID := "test_session_wasm"

	// Clean up any existing agent
	defer cleanupAgent(t, agentID)

	t.Run("BuildWASMAgent", func(t *testing.T) {
		testBuildWASMAgent(t, agentID)
	})

	t.Run("ActivateWASMAgent", func(t *testing.T) {
		testActivateWASMAgent(t, agentID, sessionID)
	})

	t.Run("WASMAgentInference", func(t *testing.T) {
		testWASMAgentInference(t, sessionID)
	})

	t.Run("WASMAgentCapabilities", func(t *testing.T) {
		testWASMAgentCapabilities(t, sessionID)
	})

	t.Run("DeactivateWASMAgent", func(t *testing.T) {
		testDeactivateWASMAgent(t, sessionID)
	})
}

// testBuildWASMAgent tests building a WASM agent
func testBuildWASMAgent(t *testing.T, agentID string) {
	buildConfig := map[string]interface{}{
		"template_id": "standard",
		"config": map[string]interface{}{
			"agent_name":       agentID,
			"agentDescription": "Test WASM agent for automated testing",
			"agent_type":       "llm",
			"build_target":     "wasm",
			"model":            "deepseek-chat",
			"instruction":      "You are a test WASM agent. Respond to all queries with helpful information.",
		},
	}

	// Convert buildConfig to JSON
	jsonData, err := json.Marshal(buildConfig)
	if err != nil {
		t.Fatalf("Failed to marshal build config: %v", err)
	}

	// Start build
	resp, err := http.Post(fmt.Sprintf("%s/api/v1/agents/%s/build", testServerURL, agentID),
		"application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Skipf("Skipping test - could not connect to server: %v", err)
		return
	}
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode, "Unexpected status code")

	var buildResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&buildResponse)
	require.NoError(t, err)

	assert.Equal(t, agentID, buildResponse["build_id"])
	assert.Contains(t, buildResponse["message"], "build started")

	// Wait for build completion
	waitForBuildCompletion(t, agentID)

	// Verify WASM file exists
	wasmPath := filepath.Join(os.Getenv("HOME"), ".config", "Agentic-Engine", "plugins", fmt.Sprintf("agent_%s_1.0.wasm", agentID))
	assert.FileExists(t, wasmPath)

	// Verify WASM file is not empty
	info, err := os.Stat(wasmPath)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(1000)) // Should be at least 1KB
}

// testActivateWASMAgent tests activating a WASM agent
func testActivateWASMAgent(t *testing.T, agentID, sessionID string) {
	activationConfig := map[string]interface{}{
		"agentId":   agentID,
		"version":   "1.0",
		"sessionId": sessionID,
		"config":    map[string]interface{}{},
	}

	resp := makeAPIRequest(t, "POST", "/api/v1/adk/agents/activate", activationConfig)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var activationResponse map[string]interface{}
	err := json.NewDecoder(resp.Body).Decode(&activationResponse)
	require.NoError(t, err)

	assert.Equal(t, "success", activationResponse["status"])
	assert.Equal(t, agentID, activationResponse["agentId"])
	assert.Equal(t, sessionID, activationResponse["sessionId"])
}

// testWASMAgentInference tests WASM agent inference
func testWASMAgentInference(t *testing.T, sessionID string) {
	inferenceRequest := map[string]interface{}{
		"sessionId":  sessionID,
		"input":      "Hello WASM agent! Please respond to this test message.",
		"parameters": map[string]interface{}{},
	}

	resp := makeAPIRequest(t, "POST", "/api/v1/adk/agents/inference", inferenceRequest)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var inferenceResponse map[string]interface{}
	err := json.NewDecoder(resp.Body).Decode(&inferenceResponse)
	require.NoError(t, err)

	assert.NotEmpty(t, inferenceResponse["output"])
	assert.NotEmpty(t, inferenceResponse["reasoning"])

	// Check metadata
	metadata, ok := inferenceResponse["metadata"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "wasm", metadata["agent_type"])
}

// testWASMAgentCapabilities tests WASM agent capabilities
func testWASMAgentCapabilities(t *testing.T, sessionID string) {
	capabilitiesRequest := map[string]interface{}{
		"sessionId": sessionID,
	}

	resp := makeAPIRequest(t, "POST", "/api/v1/adk/agents/capabilities", capabilitiesRequest)
	defer resp.Body.Close()

	// Note: This endpoint might not exist yet, so we'll check if it's implemented
	if resp.StatusCode == http.StatusNotFound {
		t.Skip("Capabilities endpoint not implemented yet")
		return
	}

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var capabilitiesResponse map[string]interface{}
	err := json.NewDecoder(resp.Body).Decode(&capabilitiesResponse)
	require.NoError(t, err)

	// Verify basic capabilities structure
	assert.NotNil(t, capabilitiesResponse)
}

// testDeactivateWASMAgent tests deactivating a WASM agent
func testDeactivateWASMAgent(t *testing.T, sessionID string) {
	deactivationRequest := map[string]interface{}{
		"sessionId": sessionID,
	}

	resp := makeAPIRequest(t, "POST", "/api/v1/adk/agents/deactivate", deactivationRequest)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var deactivationResponse map[string]interface{}
	err := json.NewDecoder(resp.Body).Decode(&deactivationResponse)
	require.NoError(t, err)

	assert.Equal(t, "success", deactivationResponse["status"])
}

// Helper functions

func makeAPIRequest(t *testing.T, method, endpoint string, data interface{}) *http.Response {
	var body io.Reader
	if data != nil {
		jsonData, err := json.Marshal(data)
		require.NoError(t, err)
		body = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, testServerURL+endpoint, body)
	require.NoError(t, err)

	if data != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{Timeout: testTimeout}
	resp, err := client.Do(req)
	require.NoError(t, err)

	return resp
}

func waitForBuildCompletion(t *testing.T, agentID string) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.Fatal("Build timeout exceeded")
		case <-ticker.C:
			resp := makeAPIRequest(t, "GET", fmt.Sprintf("/api/v1/agents/%s/build", agentID), nil)
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				continue
			}

			var buildStatus map[string]interface{}
			err := json.NewDecoder(resp.Body).Decode(&buildStatus)
			if err != nil {
				continue
			}

			status, ok := buildStatus["build_status"].(map[string]interface{})
			if !ok {
				continue
			}

			switch status["status"] {
			case "success":
				return
			case "error":
				t.Fatalf("Build failed: %v", status["message"])
			}
		}
	}
}

func cleanupAgent(t *testing.T, agentID string) {
	// Clean up WASM file
	wasmPath := filepath.Join(os.Getenv("HOME"), ".config", "Agentic-Engine", "plugins", fmt.Sprintf("agent_%s_1.0.wasm", agentID))
	if err := os.Remove(wasmPath); err != nil {
		t.Logf("Warning: Failed to cleanup WASM file %s: %v", wasmPath, err)
	} else {
		t.Logf("Successfully cleaned up WASM file: %s", wasmPath)
	}

	// Clean up any other agent artifacts
	// This is a best-effort cleanup
	pluginDir := filepath.Join(os.Getenv("HOME"), ".config", "Agentic-Engine", "plugins", agentID)
	if err := os.RemoveAll(pluginDir); err != nil {
		t.Logf("Warning: Failed to cleanup plugin directory %s: %v", pluginDir, err)
	} else {
		t.Logf("Successfully cleaned up plugin directory: %s", pluginDir)
	}
}
