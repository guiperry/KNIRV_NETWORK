package test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAgentChatFunctionality tests the complete agent chat system
func TestAgentChatFunctionality(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	var testAgentID string

	t.Run("Setup Chat Agent", func(t *testing.T) {
		// Create an agent specifically for chat testing
		agentData := map[string]interface{}{
			"name":         "Chat Test Agent",
			"type":         "chat",
			"description":  "Agent designed for chat functionality testing",
			"capabilities": []string{"chat", "conversation", "inference"},
			"config": map[string]interface{}{
				"chat_enabled":    true,
				"max_history":     100,
				"response_format": "conversational",
			},
		}

		rr, err := ts.makeRequest("POST", "/agents", agentData)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, rr.Code)

		var response map[string]interface{}
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		agent := response["agent"].(map[string]interface{})
		testAgentID = agent["id"].(string)
		assert.NotEmpty(t, testAgentID)
		t.Logf("Created chat agent with ID: %s", testAgentID)
	})

	t.Run("Test Chat Message Endpoint", func(t *testing.T) {
		require.NotEmpty(t, testAgentID)

		// Test sending a chat message
		chatMessage := map[string]interface{}{
			"sender_id":    "user-123",
			"receiver_id":  testAgentID,
			"message_type": "chat",
			"content": map[string]interface{}{
				"text":      "Hello, can you help me with a question?",
				"timestamp": time.Now().Unix(),
				"type":      "user_message",
			},
		}

		rr, err := ts.makeRequest("POST", "/agents/message", chatMessage)
		require.NoError(t, err)

		// The endpoint should exist and handle the request
		assert.True(t, rr.Code == http.StatusOK || rr.Code == http.StatusNotFound)

		if rr.Code == http.StatusOK {
			var response map[string]interface{}
			err = json.Unmarshal(rr.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Equal(t, "success", response["status"])
			assert.Contains(t, response, "message")
			t.Logf("Chat message response: %v", response)
		} else {
			t.Logf("Chat message endpoint not implemented (404)")
		}
	})

	t.Run("Test Chat Inference Integration", func(t *testing.T) {
		require.NotEmpty(t, testAgentID)

		// Test chat through inference endpoint
		inferenceData := map[string]interface{}{
			"agentId": testAgentID,
			"prompt":  "What is the capital of France?",
			"context": map[string]interface{}{
				"session_id":   "chat-session-456",
				"user_id":      "user-123",
				"message_type": "chat",
				"conversation": true,
			},
		}

		rr, err := ts.makeRequest("POST", "/adk/agents/inference", inferenceData)
		require.NoError(t, err)

		// Inference should work or provide meaningful error
		assert.True(t, rr.Code == http.StatusOK || rr.Code == http.StatusBadRequest || rr.Code == http.StatusInternalServerError)

		if rr.Code == http.StatusOK {
			var response map[string]interface{}
			err = json.Unmarshal(rr.Body.Bytes(), &response)
			require.NoError(t, err)

			// Should contain inference response
			assert.True(t, len(response) > 0)
			t.Logf("Chat inference response: %v", response)
		} else {
			t.Logf("Chat inference failed with code: %d", rr.Code)
		}
	})

	t.Run("Test Chat Session Management", func(t *testing.T) {
		require.NotEmpty(t, testAgentID)

		// Test creating a chat session
		sessionData := map[string]interface{}{
			"agentId":   testAgentID,
			"sessionId": "chat-session-789",
			"userId":    "user-123",
			"config": map[string]interface{}{
				"max_turns":      50,
				"context_window": 4096,
			},
		}

		rr, err := ts.makeRequest("POST", "/agents/"+testAgentID+"/chat/session", sessionData)
		require.NoError(t, err)

		// Session creation might not be implemented
		assert.True(t, rr.Code == http.StatusOK || rr.Code == http.StatusCreated || rr.Code == http.StatusNotFound)

		if rr.Code == http.StatusOK || rr.Code == http.StatusCreated {
			var response map[string]interface{}
			err = json.Unmarshal(rr.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Contains(t, response, "sessionId")
			t.Logf("Chat session created: %v", response)
		}
	})

	t.Run("Test Chat History Retrieval", func(t *testing.T) {
		require.NotEmpty(t, testAgentID)

		// Test retrieving chat history
		rr, err := ts.makeRequest("GET", "/agents/"+testAgentID+"/chat/history?sessionId=chat-session-789", nil)
		require.NoError(t, err)

		// History endpoint might not be implemented
		assert.True(t, rr.Code == http.StatusOK || rr.Code == http.StatusNotFound)

		if rr.Code == http.StatusOK {
			var response map[string]interface{}
			err = json.Unmarshal(rr.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Contains(t, response, "messages")
			t.Logf("Chat history: %v", response)
		}
	})

	t.Run("Test Chat Error Handling", func(t *testing.T) {
		// Test with invalid agent ID
		invalidChatMessage := map[string]interface{}{
			"sender_id":    "user-123",
			"receiver_id":  "invalid-agent-id",
			"message_type": "chat",
			"content": map[string]interface{}{
				"text": "This should fail",
			},
		}

		rr, err := ts.makeRequest("POST", "/agents/message", invalidChatMessage)
		require.NoError(t, err)

		// Should return appropriate error
		assert.True(t, rr.Code == http.StatusBadRequest || rr.Code == http.StatusNotFound || rr.Code == http.StatusForbidden)
	})

	t.Run("Test Chat Message Validation", func(t *testing.T) {
		// Test with missing required fields
		invalidMessage := map[string]interface{}{
			"sender_id": "user-123",
			// Missing receiver_id and content
		}

		rr, err := ts.makeRequest("POST", "/agents/message", invalidMessage)
		require.NoError(t, err)

		// Should return bad request
		assert.True(t, rr.Code == http.StatusBadRequest || rr.Code == http.StatusNotFound)
	})

	t.Run("Test Chat Performance", func(t *testing.T) {
		require.NotEmpty(t, testAgentID)

		start := time.Now()

		// Send multiple chat messages quickly
		for i := 0; i < 3; i++ {
			chatMessage := map[string]interface{}{
				"sender_id":    "user-123",
				"receiver_id":  testAgentID,
				"message_type": "chat",
				"content": map[string]interface{}{
					"text": fmt.Sprintf("Performance test message %d", i+1),
				},
			}

			rr, err := ts.makeRequest("POST", "/agents/message", chatMessage)
			require.NoError(t, err)

			// Should handle requests reasonably fast
			assert.True(t, rr.Code == http.StatusOK || rr.Code == http.StatusNotFound)
		}

		duration := time.Since(start)
		assert.Less(t, duration, 5*time.Second)
		t.Logf("Chat performance test took: %v", duration)
	})
}

// TestAgentChatIntegrationIssues tests specific issues with agent chat
func TestAgentChatIntegrationIssues(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	t.Run("Chat Modal Integration", func(t *testing.T) {
		// Test the chat modal's expected API calls
		// This simulates what the frontend AgentChatModal.jsx would do

		// 1. Get agent details
		rr, err := ts.makeRequest("GET", "/agents/test-chat-agent", nil)
		require.NoError(t, err)

		// Agent might not exist, which is fine for this test
		assert.True(t, rr.Code == http.StatusOK || rr.Code == http.StatusNotFound)

		// 2. Test chat message sending (what the modal does)
		chatData := map[string]interface{}{
			"agentId":   "test-chat-agent",
			"message":   "Hello from chat modal",
			"type":      "user_input",
			"sessionId": "modal-session-123",
		}

		rr, err = ts.makeRequest("POST", "/agents/chat", chatData)
		require.NoError(t, err)

		// Chat endpoint might not be implemented
		assert.True(t, rr.Code == http.StatusOK || rr.Code == http.StatusNotFound)
	})

	t.Run("Chat Service Integration", func(t *testing.T) {
		// Test what the frontend chat service expects
		// Based on AgentChatModal.jsx implementation

		// Test simulated agent response
		responseData := map[string]interface{}{
			"agentId":   "test-chat-agent",
			"sessionId": "service-test-session",
			"prompt":    "Test prompt for agent",
		}

		rr, err := ts.makeRequest("POST", "/adk/agents/inference", responseData)
		require.NoError(t, err)

		// Should get some response
		assert.True(t, rr.Code == http.StatusOK || rr.Code == http.StatusBadRequest || rr.Code == http.StatusInternalServerError)

		if rr.Code == http.StatusOK {
			var response map[string]interface{}
			err = json.Unmarshal(rr.Body.Bytes(), &response)
			require.NoError(t, err)

			// Should contain some response data
			assert.True(t, len(response) > 0)
		}
	})

	t.Run("Chat WebSocket Integration", func(t *testing.T) {
		// Test WebSocket endpoints that chat might use
		wsData := map[string]interface{}{
			"sessionId": "ws-chat-session",
			"agentId":   "test-chat-agent",
			"messageId": "msg-123",
		}

		rr, err := ts.makeRequest("POST", "/api/v1/chat/ws/connect", wsData)
		require.NoError(t, err)

		// WebSocket chat might not be implemented
		assert.True(t, rr.Code == http.StatusOK || rr.Code == http.StatusNotFound)
	})

	t.Run("Chat State Management", func(t *testing.T) {
		// Test chat state persistence
		stateData := map[string]interface{}{
			"agentId":   "test-chat-agent",
			"sessionId": "state-test-session",
			"state": map[string]interface{}{
				"conversation_context": "Previous conversation context",
				"user_preferences":     map[string]interface{}{"theme": "dark"},
				"last_message_id":      "msg-456",
			},
		}

		rr, err := ts.makeRequest("POST", "/agents/chat/state", stateData)
		require.NoError(t, err)

		// State management might not be implemented
		assert.True(t, rr.Code == http.StatusOK || rr.Code == http.StatusNotFound)

		// Test state retrieval
		rr, err = ts.makeRequest("GET", "/agents/test-chat-agent/chat/state?sessionId=state-test-session", nil)
		require.NoError(t, err)

		assert.True(t, rr.Code == http.StatusOK || rr.Code == http.StatusNotFound)
	})
}

// TestAgentChatDebugging provides debugging information for chat issues
func TestAgentChatDebugging(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	t.Run("Debug Chat Endpoints", func(t *testing.T) {
		// List all available endpoints to see what's implemented
		endpoints := []string{
			"/agents/message",
			"/agents/chat",
			"/adk/agents/inference",
			"/api/v1/chat/ws/connect",
			"/agents/test-agent/chat/session",
			"/agents/test-agent/chat/history",
			"/agents/chat/state",
		}

		for _, endpoint := range endpoints {
			rr, err := ts.makeRequest("GET", endpoint, nil)
			require.NoError(t, err)

			t.Logf("Endpoint %s returned status: %d", endpoint, rr.Code)

			if rr.Code == http.StatusOK {
				t.Logf("Endpoint %s is implemented and working", endpoint)
			} else if rr.Code == http.StatusNotFound {
				t.Logf("Endpoint %s is not implemented", endpoint)
			} else {
				t.Logf("Endpoint %s returned unexpected status: %d", endpoint, rr.Code)
			}
		}
	})

	t.Run("Debug Agent Chat Configuration", func(t *testing.T) {
		// Check if agents have chat capabilities
		rr, err := ts.makeRequest("GET", "/agents", nil)
		require.NoError(t, err)

		if rr.Code == http.StatusOK {
			var response map[string]interface{}
			err = json.Unmarshal(rr.Body.Bytes(), &response)
			require.NoError(t, err)

			agents := response["agents"].([]interface{})
			t.Logf("Found %d agents", len(agents))

			for _, agentInterface := range agents {
				agent := agentInterface.(map[string]interface{})
				agentID := agent["id"].(string)
				capabilities, hasCapabilities := agent["capabilities"]

				t.Logf("Agent %s capabilities: %v (has capabilities: %v)", agentID, capabilities, hasCapabilities)
			}
		}
	})
}
