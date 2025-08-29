package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWASMVsPluginPerformance compares WASM and Plugin agent performance
func TestWASMVsPluginPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Test agents
	wasmAgentID := "perf_test_wasm"
	pluginAgentID := "perf_test_plugin"
	wasmSessionID := "perf_session_wasm"
	pluginSessionID := "perf_session_plugin"

	// Clean up
	defer cleanupAgent(t, wasmAgentID)
	defer cleanupAgent(t, pluginAgentID)

	t.Run("BuildAgents", func(t *testing.T) {
		// Build WASM agent
		testBuildAgent(t, wasmAgentID, "wasm")

		// Build Plugin agent
		testBuildAgent(t, pluginAgentID, "plugin")
	})

	t.Run("ActivateAgents", func(t *testing.T) {
		// Activate WASM agent
		testActivateWASMAgent(t, wasmAgentID, wasmSessionID)

		// Activate Plugin agent
		testActivatePluginAgent(t, pluginAgentID, pluginSessionID)
	})

	t.Run("PerformanceComparison", func(t *testing.T) {
		// Test inference performance
		wasmTime := measureInferenceTime(t, wasmSessionID, 10)
		pluginTime := measureInferenceTime(t, pluginSessionID, 10)

		t.Logf("WASM average inference time: %v", wasmTime)
		t.Logf("Plugin average inference time: %v", pluginTime)

		// Both should complete within reasonable time
		assert.Less(t, wasmTime, 5*time.Second, "WASM inference should complete within 5 seconds")
		assert.Less(t, pluginTime, 5*time.Second, "Plugin inference should complete within 5 seconds")

		// Log performance ratio
		ratio := float64(wasmTime) / float64(pluginTime)
		t.Logf("WASM/Plugin performance ratio: %.2f", ratio)
	})

	t.Run("MemoryUsage", func(t *testing.T) {
		// This is a placeholder for memory usage testing
		// In a real implementation, you would measure memory usage
		t.Log("Memory usage testing would be implemented here")
	})
}

// TestWASMAgentCompatibility tests WASM agent compatibility across different scenarios
func TestWASMAgentCompatibility(t *testing.T) {
	agentID := "compat_test_wasm"
	sessionID := "compat_session"

	defer cleanupAgent(t, agentID)

	t.Run("BuildAndActivate", func(t *testing.T) {
		testBuildAgent(t, agentID, "wasm")
		testActivateWASMAgent(t, agentID, sessionID)
	})

	t.Run("MultipleInferences", func(t *testing.T) {
		// Test multiple consecutive inferences
		for i := 0; i < 5; i++ {
			testSingleInference(t, sessionID, fmt.Sprintf("Test message %d", i+1))
		}
	})

	t.Run("ConcurrentInferences", func(t *testing.T) {
		// Test concurrent inferences
		done := make(chan bool, 3)

		for i := 0; i < 3; i++ {
			go func(id int) {
				defer func() { done <- true }()
				testSingleInference(t, sessionID, fmt.Sprintf("Concurrent test %d", id))
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < 3; i++ {
			select {
			case <-done:
				// Success
			case <-time.After(10 * time.Second):
				t.Fatal("Concurrent inference test timed out")
			}
		}
	})

	t.Run("LongRunningSession", func(t *testing.T) {
		// Test that WASM agent can handle long-running sessions
		start := time.Now()
		for time.Since(start) < 30*time.Second {
			testSingleInference(t, sessionID, "Long running test")
			time.Sleep(1 * time.Second)
		}
	})
}

// TestWASMAgentErrorHandling tests error handling in WASM agents
func TestWASMAgentErrorHandling(t *testing.T) {
	agentID := "error_test_wasm"
	sessionID := "error_session"

	defer cleanupAgent(t, agentID)

	t.Run("BuildAndActivate", func(t *testing.T) {
		testBuildAgent(t, agentID, "wasm")
		testActivateWASMAgent(t, agentID, sessionID)
	})

	t.Run("InvalidInferenceRequest", func(t *testing.T) {
		// Test with empty input
		inferenceRequest := map[string]interface{}{
			"sessionId":  sessionID,
			"input":      "",
			"parameters": map[string]interface{}{},
		}

		resp := makeAPIRequest(t, "POST", "/api/v1/adk/agents/inference", inferenceRequest)
		defer resp.Body.Close()

		// Should still handle gracefully
		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("InvalidSessionId", func(t *testing.T) {
		// Test with non-existent session
		inferenceRequest := map[string]interface{}{
			"sessionId":  "non_existent_session",
			"input":      "Test message",
			"parameters": map[string]interface{}{},
		}

		resp := makeAPIRequest(t, "POST", "/api/v1/adk/agents/inference", inferenceRequest)
		defer resp.Body.Close()

		// Should return appropriate error
		assert.NotEqual(t, 200, resp.StatusCode)
	})
}

// Helper functions

func testBuildAgent(t *testing.T, agentID, buildTarget string) {
	buildConfig := map[string]interface{}{
		"template_id": "standard",
		"config": map[string]interface{}{
			"agent_name":       agentID,
			"agentDescription": fmt.Sprintf("Performance test %s agent", buildTarget),
			"agent_type":       "llm",
			"build_target":     buildTarget,
			"model":            "deepseek-chat",
			"instruction":      "You are a performance test agent. Respond quickly and efficiently.",
		},
	}

	// Convert buildConfig to JSON
	jsonData, err := json.Marshal(buildConfig)
	if err != nil {
		t.Fatalf("Failed to marshal build config: %v", err)
	}

	resp, err := http.Post(fmt.Sprintf("%s/api/v1/agents/%s/build", testServerURL, agentID),
		"application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Skipf("Skipping test - could not connect to server: %v", err)
		return
	}
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode, "Unexpected status code")
	waitForBuildCompletion(t, agentID)
}

func testActivatePluginAgent(t *testing.T, agentID, sessionID string) {
	activationConfig := map[string]interface{}{
		"agentId":   agentID,
		"version":   "1.0",
		"sessionId": sessionID,
		"config":    map[string]interface{}{},
	}

	resp := makeAPIRequest(t, "POST", "/api/v1/adk/agents/activate", activationConfig)
	defer resp.Body.Close()

	require.Equal(t, 200, resp.StatusCode)
}

func measureInferenceTime(t *testing.T, sessionID string, iterations int) time.Duration {
	var totalTime time.Duration

	for i := 0; i < iterations; i++ {
		start := time.Now()
		testSingleInference(t, sessionID, fmt.Sprintf("Performance test iteration %d", i+1))
		totalTime += time.Since(start)
	}

	return totalTime / time.Duration(iterations)
}

func testSingleInference(t *testing.T, sessionID, input string) {
	inferenceRequest := map[string]interface{}{
		"sessionId":  sessionID,
		"input":      input,
		"parameters": map[string]interface{}{},
	}

	resp := makeAPIRequest(t, "POST", "/api/v1/adk/agents/inference", inferenceRequest)
	defer resp.Body.Close()

	require.Equal(t, 200, resp.StatusCode)
}
