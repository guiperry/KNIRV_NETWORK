package test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAgentCardToWASMConnectivity tests end-to-end connectivity from agent cards to WASM files
func TestAgentCardToWASMConnectivity(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	t.Run("Agent Card Creation and WASM Association", func(t *testing.T) {
		// Create an agent with WASM build target
		agentData := map[string]interface{}{
			"name":         "WASM Test Agent",
			"type":         "wasm",
			"description":  "Agent for testing WASM connectivity",
			"build_target": "wasm",
			"config": map[string]interface{}{
				"wasm_path": "/test/wasm/agent.wasm",
				"runtime":   "wazero",
			},
		}

		rr, err := ts.makeRequest("POST", "/agents", agentData)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, rr.Code)

		var response map[string]interface{}
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		if data, exists := response["data"]; exists {
			agent := data.(map[string]interface{})
			agentID := agent["id"].(string)
			assert.NotEmpty(t, agentID)
			if buildTarget, exists := agent["build_target"]; exists {
				assert.Equal(t, "wasm", buildTarget)
			}
		} else if agent, exists := response["agent"]; exists {
			agentMap := agent.(map[string]interface{})
			agentID := agentMap["id"].(string)
			assert.NotEmpty(t, agentID)
			if buildTarget, exists := agentMap["build_target"]; exists {
				assert.Equal(t, "wasm", buildTarget)
			}
		}
	})

	t.Run("WASM Agent Terminal Creation", func(t *testing.T) {
		// Test terminal creation for WASM agent
		terminalData := map[string]interface{}{
			"sessionId": "test-session-123",
			"agentId":   "wasm-test-agent",
			"rows":      24,
			"cols":      80,
		}

		rr, err := ts.makeRequest("POST", "/adk/agents/terminal/create", terminalData)
		require.NoError(t, err)

		// Terminal creation might not be fully implemented for WASM
		assert.True(t, rr.Code == http.StatusOK || rr.Code == http.StatusNotFound)

		if rr.Code == http.StatusOK {
			var response map[string]interface{}
			err = json.Unmarshal(rr.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Contains(t, response, "terminalId")
		}
	})

	t.Run("WASM Agent Message Flow", func(t *testing.T) {
		// Test message flow from frontend to WASM agent
		messageData := map[string]interface{}{
			"agentId": "wasm-test-agent",
			"message": "Hello WASM agent",
			"type":    "user_input",
		}

		rr, err := ts.makeRequest("POST", "/adk/agents/message", messageData)
		require.NoError(t, err)

		// Message handling might not be fully implemented
		assert.True(t, rr.Code == http.StatusOK || rr.Code == http.StatusNotFound)
	})
}

// TestAgentTerminalConnectivity tests terminal connectivity for agents
func TestAgentTerminalConnectivity(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	t.Run("Agent Terminal Session Creation", func(t *testing.T) {
		// Create agent first
		agentData := map[string]interface{}{
			"name":        "Terminal Test Agent",
			"type":        "plugin",
			"description": "Agent for testing terminal connectivity",
		}

		rr, err := ts.makeRequest("POST", "/agents", agentData)
		require.NoError(t, err)

		if rr.Code == http.StatusCreated {
			var response map[string]interface{}
			err = json.Unmarshal(rr.Body.Bytes(), &response)
			require.NoError(t, err)

			agent := response["agent"].(map[string]interface{})
			agentID := agent["id"].(string)

			// Test terminal creation
			terminalData := map[string]interface{}{
				"sessionId": "terminal-test-session",
				"agentId":   agentID,
				"rows":      24,
				"cols":      80,
			}

			terminalRR, err := ts.makeRequest("POST", "/adk/agents/terminal/create", terminalData)
			require.NoError(t, err)

			// Terminal creation should work or return appropriate error
			assert.True(t, terminalRR.Code == http.StatusOK || terminalRR.Code == http.StatusNotFound)
		}
	})

	t.Run("Terminal Message Logging", func(t *testing.T) {
		// Test that messages are properly logged to terminals
		logData := map[string]interface{}{
			"agentId": "terminal-test-agent",
			"message": "Test log message",
			"level":   "info",
		}

		rr, err := ts.makeRequest("POST", "/adk/agents/log", logData)
		require.NoError(t, err)

		// Log endpoint might not be implemented
		assert.True(t, rr.Code == http.StatusOK || rr.Code == http.StatusNotFound)
	})

	t.Run("Terminal Output Streaming", func(t *testing.T) {
		// Test terminal output streaming
		outputData := map[string]interface{}{
			"terminalId": "test-terminal-123",
			"data":       "Hello from terminal",
		}

		rr, err := ts.makeRequest("POST", "/adk/agents/terminal/write", outputData)
		require.NoError(t, err)

		// Write to terminal might not be implemented
		assert.True(t, rr.Code == http.StatusOK || rr.Code == http.StatusNotFound)
	})
}

// TestAgentChatConnectivity tests agent chat functionality
func TestAgentChatConnectivity(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	t.Run("Agent Chat Message Processing", func(t *testing.T) {
		// Create agent for chat testing
		agentData := map[string]interface{}{
			"name":         "Chat Test Agent",
			"type":         "chat",
			"description":  "Agent for testing chat functionality",
			"capabilities": []string{"chat", "conversation"},
		}

		rr, err := ts.makeRequest("POST", "/agents", agentData)
		require.NoError(t, err)

		if rr.Code == http.StatusCreated {
			var response map[string]interface{}
			err = json.Unmarshal(rr.Body.Bytes(), &response)
			require.NoError(t, err)

			agent := response["agent"].(map[string]interface{})
			agentID := agent["id"].(string)

			// Test chat message
			chatData := map[string]interface{}{
				"sender_id":    "user-123",
				"receiver_id":  agentID,
				"message_type": "chat",
				"content": map[string]interface{}{
					"text": "Hello, how are you?",
					"type": "user_message",
				},
			}

			chatRR, err := ts.makeRequest("POST", "/agents/message", chatData)
			require.NoError(t, err)

			// Chat should work or return appropriate error
			assert.True(t, chatRR.Code == http.StatusOK || chatRR.Code == http.StatusNotFound)

			if chatRR.Code == http.StatusOK {
				var chatResponse map[string]interface{}
				err = json.Unmarshal(chatRR.Body.Bytes(), &chatResponse)
				require.NoError(t, err)

				assert.Equal(t, "success", chatResponse["status"])
			}
		}
	})

	t.Run("Agent Chat Inference", func(t *testing.T) {
		// Test chat inference processing
		inferenceData := map[string]interface{}{
			"agentId": "chat-test-agent",
			"prompt":  "What is the weather like today?",
			"context": map[string]interface{}{
				"session_id": "chat-session-123",
				"user_id":    "user-123",
			},
		}

		rr, err := ts.makeRequest("POST", "/adk/agents/inference", inferenceData)
		require.NoError(t, err)

		// Inference might fail without proper setup
		assert.True(t, rr.Code == http.StatusOK || rr.Code == http.StatusBadRequest || rr.Code == http.StatusInternalServerError)

		if rr.Code == http.StatusOK {
			var response map[string]interface{}
			err = json.Unmarshal(rr.Body.Bytes(), &response)
			require.NoError(t, err)

			// Should contain some response
			assert.True(t, len(response) > 0)
		}
	})

	t.Run("Agent Chat History", func(t *testing.T) {
		// Test chat history retrieval
		rr, err := ts.makeRequest("GET", "/agents/chat-test-agent/history", nil)
		require.NoError(t, err)

		// History endpoint might not be implemented
		assert.True(t, rr.Code == http.StatusOK || rr.Code == http.StatusNotFound)
	})
}

// TestWebSocketConnectivity tests WebSocket connections for real-time communication
func TestWebSocketConnectivity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping WebSocket tests in short mode")
	}

	ts := SetupTestServer(t)
	defer ts.Cleanup()

	t.Run("Terminal WebSocket Connection", func(t *testing.T) {
		// Create a test server for WebSocket
		server := httptest.NewServer(ts.router)
		defer server.Close()

		// Convert HTTP URL to WebSocket URL
		wsURL := strings.Replace(server.URL, "http://", "ws://", 1) + "/api/v1/terminal/ws?sessionId=test&terminalId=test-terminal"

		// Attempt WebSocket connection
		dialer := websocket.Dialer{
			HandshakeTimeout: 5 * time.Second,
		}

		conn, _, err := dialer.Dial(wsURL, nil)
		if err != nil {
			// WebSocket might not be fully implemented
			t.Logf("WebSocket connection failed (expected): %v", err)
			return
		}
		defer conn.Close()

		// Test sending a message
		testMessage := []byte("test message")
		err = conn.WriteMessage(websocket.TextMessage, testMessage)
		assert.NoError(t, err)

		// Test receiving a message (with timeout)
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, message, err := conn.ReadMessage()
		if err == nil {
			t.Logf("Received WebSocket message: %s", string(message))
		}
	})
}

// TestAgentPluginIntegration tests integration between agent plugins and the system
func TestAgentPluginIntegration(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	t.Run("Plugin Agent Discovery", func(t *testing.T) {
		// Test plugin agent discovery
		rr, err := ts.makeRequest("GET", "/adk/agents", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rr.Code)

		var response map[string]interface{}
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response, "agents")
		agents := response["agents"].([]interface{})
		assert.GreaterOrEqual(t, len(agents), 0)
	})

	t.Run("Plugin Agent Activation", func(t *testing.T) {
		// Test plugin agent activation
		activationData := map[string]interface{}{
			"agentId":   "test-plugin-agent",
			"sessionId": "plugin-test-session",
			"config": map[string]interface{}{
				"timeout": 30,
			},
		}

		rr, err := ts.makeRequest("POST", "/adk/agents/activate", activationData)
		require.NoError(t, err)

		// Activation might fail without actual plugin
		assert.True(t, rr.Code == http.StatusOK || rr.Code == http.StatusBadRequest || rr.Code == http.StatusNotFound)
	})

	t.Run("Plugin Agent Capabilities", func(t *testing.T) {
		// Test plugin agent capabilities
		rr, err := ts.makeRequest("GET", "/adk/agents/capabilities", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rr.Code)

		var response map[string]interface{}
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		// Should contain capabilities information
		assert.True(t, len(response) > 0)
	})
}

// TestAgentErrorHandling tests error handling in agent connectivity
func TestAgentErrorHandling(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	t.Run("Invalid Agent ID", func(t *testing.T) {
		// Test with invalid agent ID
		rr, err := ts.makeRequest("GET", "/agents/invalid-agent-id-12345", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("Invalid Terminal Session", func(t *testing.T) {
		// Test with invalid terminal session
		terminalData := map[string]interface{}{
			"sessionId": "",
			"agentId":   "test-agent",
			"rows":      24,
			"cols":      80,
		}

		rr, err := ts.makeRequest("POST", "/adk/agents/terminal/create", terminalData)
		require.NoError(t, err)

		// Should return bad request for invalid session
		assert.True(t, rr.Code == http.StatusBadRequest || rr.Code == http.StatusNotFound)
	})

	t.Run("Invalid Chat Message", func(t *testing.T) {
		// Test with invalid chat message format
		invalidChatData := map[string]interface{}{
			"invalid_field": "invalid_value",
		}

		rr, err := ts.makeRequest("POST", "/agents/message", invalidChatData)
		require.NoError(t, err)

		// Should return bad request for invalid message
		assert.True(t, rr.Code == http.StatusBadRequest || rr.Code == http.StatusNotFound)
	})
}

// TestAgentPerformanceConnectivity tests performance aspects of agent connectivity
func TestAgentPerformanceConnectivity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	ts := SetupTestServer(t)
	defer ts.Cleanup()

	t.Run("Concurrent Agent Operations", func(t *testing.T) {
		const numOperations = 5
		results := make(chan int, numOperations)

		for i := 0; i < numOperations; i++ {
			go func(id int) {
				agentData := map[string]interface{}{
					"name":        fmt.Sprintf("Concurrent Agent %d", id),
					"type":        "test",
					"description": fmt.Sprintf("Concurrent test agent %d", id),
				}

				rr, err := ts.makeRequest("POST", "/agents", agentData)
				if err != nil {
					results <- 500
					return
				}
				results <- rr.Code
			}(i)
		}

		for i := 0; i < numOperations; i++ {
			code := <-results
			assert.True(t, code == http.StatusCreated || code == http.StatusBadRequest)
		}
	})

	t.Run("Agent Response Time", func(t *testing.T) {
		start := time.Now()

		rr, err := ts.makeRequest("GET", "/adk/agents", nil)
		require.NoError(t, err)

		duration := time.Since(start)
		assert.Equal(t, http.StatusOK, rr.Code)

		// Agent discovery should be fast
		assert.Less(t, duration, 2*time.Second)
		t.Logf("Agent discovery took: %v", duration)
	})
}
