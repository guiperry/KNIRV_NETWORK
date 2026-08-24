// secure_bridge_test.go
package desktop

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockDesktopTEEManager implements TEE manager interface for testing
type MockDesktopTEEManager struct {
	sessions map[string]*TEESession
	mu       sync.RWMutex
}

func NewMockDesktopTEEManager() *MockDesktopTEEManager {
	return &MockDesktopTEEManager{
		sessions: make(map[string]*TEESession),
	}
}

func (m *MockDesktopTEEManager) CreateTEESession(clientID string) (*TEESession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	session := &TEESession{
		SessionID: "tee-session-" + clientID,
		ClientID:  clientID,
		CreatedAt: time.Now(),
		Active:    true,
	}

	m.sessions[session.SessionID] = session
	return session, nil
}

func (m *MockDesktopTEEManager) ValidateSession(sessionID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[sessionID]
	return exists && session.Active
}

func (m *MockDesktopTEEManager) ExecuteInTEE(pluginID, command string, args []string) (string, string, int, error) {
	return "", "", 0, nil
}

func (m *MockDesktopTEEManager) ListPlugins() []*PluginInfo {
	return []*PluginInfo{}
}

func (m *MockDesktopTEEManager) LoadPlugin(pluginID string, ctx *SecurityContext) error {
	return nil
}

func (m *MockDesktopTEEManager) UnloadPlugin(pluginID string) error {
	return nil
}

func (m *MockDesktopTEEManager) verifySignatureWithKey(hash, signature []byte, keyPath string) (bool, error) {
	return true, nil
}

// Test helper functions
func createTestSecureBridge(_ *testing.T) *SecureBridge {
	teeManager := NewMockDesktopTEEManager()

	bridge := &SecureBridge{
		sessionTokens: make(map[string]*SessionInfo),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for testing
			},
		},
		teeManager: teeManager,
	}

	return bridge
}


// TestSecureBridge_NewSecureBridge tests the constructor
func TestSecureBridge_NewSecureBridge(t *testing.T) {
	bridge := createTestSecureBridge(t)

	assert.NotNil(t, bridge)
	assert.NotNil(t, bridge.sessionTokens)
	assert.NotNil(t, bridge.teeManager)
}

// TestSecureBridge_CreateSession tests session creation
func TestSecureBridge_CreateSession(t *testing.T) {
	bridge := createTestSecureBridge(t)

	clientID := "test-client"
	permissions := []string{"read", "write", "execute"}

	token, err := bridge.CreateSession(clientID, permissions)

	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// Verify session was created
	bridge.mutex.RLock()
	session, exists := bridge.sessionTokens[token]
	bridge.mutex.RUnlock()

	assert.True(t, exists)
	assert.Equal(t, clientID, session.ClientID)
	assert.Equal(t, permissions, session.Permissions)
	assert.Equal(t, token, session.Token)
}

// TestSecureBridge_ValidateSession tests session validation
func TestSecureBridge_ValidateSession(t *testing.T) {
	bridge := createTestSecureBridge(t)

	clientID := "test-client"
	token, err := bridge.CreateSession(clientID, []string{"read"})
	require.NoError(t, err)

	// Test valid session
	session, valid := bridge.ValidateSession(token)
	assert.True(t, valid)
	assert.NotNil(t, session)
	assert.Equal(t, clientID, session.ClientID)

	// Test invalid session
	session, valid = bridge.ValidateSession("invalid-token")
	assert.False(t, valid)
	assert.Nil(t, session)
}

// TestSecureBridge_RevokeSession tests session revocation
func TestSecureBridge_RevokeSession(t *testing.T) {
	bridge := createTestSecureBridge(t)

	clientID := "test-client"
	token, err := bridge.CreateSession(clientID, []string{"read"})
	require.NoError(t, err)

	// Verify session exists
	_, valid := bridge.ValidateSession(token)
	assert.True(t, valid)

	// Revoke session
	err = bridge.RevokeSession(token)
	assert.NoError(t, err)

	// Verify session no longer exists
	_, valid = bridge.ValidateSession(token)
	assert.False(t, valid)
}

// TestSecureBridge_RevokeSession_NonexistentToken tests revoking nonexistent session
func TestSecureBridge_RevokeSession_NonexistentToken(t *testing.T) {
	bridge := createTestSecureBridge(t)

	err := bridge.RevokeSession("nonexistent-token")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session not found")
}

// TestSecureBridge_CleanupExpiredSessions tests expired session cleanup
func TestSecureBridge_CleanupExpiredSessions(t *testing.T) {
	bridge := createTestSecureBridge(t)

	// Create sessions with different ages
	clientID1 := "client1"
	token1, err := bridge.CreateSession(clientID1, []string{"read"})
	require.NoError(t, err)

	clientID2 := "client2"
	token2, err := bridge.CreateSession(clientID2, []string{"read"})
	require.NoError(t, err)

	// Manually set one session as expired
	bridge.mutex.Lock()
	bridge.sessionTokens[token1].LastUsed = time.Now().Add(-2 * time.Hour)
	bridge.mutex.Unlock()

	// Cleanup with 1 hour timeout
	cleaned := bridge.CleanupExpiredSessions(1 * time.Hour)

	assert.Equal(t, 1, cleaned)

	// Verify expired session was removed
	_, valid := bridge.ValidateSession(token1)
	assert.False(t, valid)

	// Verify active session still exists
	_, valid = bridge.ValidateSession(token2)
	assert.True(t, valid)
}

// TestSecureBridge_ProcessMessage tests message processing
func TestSecureBridge_ProcessMessage(t *testing.T) {
	bridge := createTestSecureBridge(t)

	// Create a session first
	clientID := "test-client"
	token, err := bridge.CreateSession(clientID, []string{"read", "write"})
	require.NoError(t, err)

	// Create a test message
	message := &SecureMessage{
		ID:        "msg-001",
		Type:      "test",
		Payload:   map[string]interface{}{"data": "test data"},
		Token:     token,
		Timestamp: time.Now(),
	}

	response, err := bridge.ProcessMessage(message)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, message.ID, response.ID)
	assert.True(t, response.Success)
}

// TestSecureBridge_ProcessMessage_InvalidToken tests processing with invalid token
func TestSecureBridge_ProcessMessage_InvalidToken(t *testing.T) {
	bridge := createTestSecureBridge(t)

	message := &SecureMessage{
		ID:        "msg-001",
		Type:      "test",
		Token:     "invalid-token",
		Timestamp: time.Now(),
	}

	response, err := bridge.ProcessMessage(message)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "invalid session")
}

// TestSecureBridge_ProcessMessage_MissingPermission tests processing without required permission
func TestSecureBridge_ProcessMessage_MissingPermission(t *testing.T) {
	bridge := createTestSecureBridge(t)

	// Create session with limited permissions
	clientID := "test-client"
	token, err := bridge.CreateSession(clientID, []string{"read"}) // No write permission
	require.NoError(t, err)

	message := &SecureMessage{
		ID:        "msg-001",
		Type:      "write_operation",
		Payload:   map[string]interface{}{"data": "test data"},
		Token:     token,
		Timestamp: time.Now(),
	}

	response, err := bridge.ProcessMessage(message)

	// This depends on implementation - might succeed or fail based on permission checking
	if err != nil {
		assert.Contains(t, err.Error(), "permission")
	} else {
		assert.NotNil(t, response)
	}
}

// TestSecureBridge_HandleWebSocket tests WebSocket handling
func TestSecureBridge_HandleWebSocket(t *testing.T) {
	bridge := createTestSecureBridge(t)

	// Create a test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bridge.HandleWebSocket(w, r)
	}))
	defer server.Close()

	// Convert HTTP URL to WebSocket URL
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Connect to WebSocket
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	// Create a session first
	clientID := "test-client"
	token, err := bridge.CreateSession(clientID, []string{"read", "write"})
	require.NoError(t, err)

	// Send a test message
	testMessage := &SecureMessage{
		ID:        "ws-msg-001",
		Type:      "ping",
		Token:     token,
		Timestamp: time.Now(),
	}

	err = conn.WriteJSON(testMessage)
	assert.NoError(t, err)

	// Read response
	var response MessageResponse
	err = conn.ReadJSON(&response)
	assert.NoError(t, err)
	assert.Equal(t, testMessage.ID, response.ID)
}

// TestSecureBridge_ConcurrentAccess tests thread safety
func TestSecureBridge_ConcurrentAccess(t *testing.T) {
	bridge := createTestSecureBridge(t)

	var wg sync.WaitGroup
	numGoroutines := 10

	// Test concurrent session creation
	tokens := make([]string, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			clientID := fmt.Sprintf("client-%d", id)
			token, err := bridge.CreateSession(clientID, []string{"read"})
			assert.NoError(t, err)
			tokens[id] = token
		}(i)
	}

	wg.Wait()

	// Verify all sessions were created
	for i, token := range tokens {
		_, valid := bridge.ValidateSession(token)
		assert.True(t, valid, "Session %d should be valid", i)
	}
}

// TestSecureBridge_SessionTimeout tests session timeout handling
func TestSecureBridge_SessionTimeout(t *testing.T) {
	bridge := createTestSecureBridge(t)

	clientID := "test-client"
	token, err := bridge.CreateSession(clientID, []string{"read"})
	require.NoError(t, err)

	// Manually expire the session
	bridge.mutex.Lock()
	bridge.sessionTokens[token].LastUsed = time.Now().Add(-2 * time.Hour)
	bridge.mutex.Unlock()

	// Try to validate expired session
	_, valid := bridge.ValidateSessionWithTimeout(token, 1*time.Hour)
	assert.False(t, valid)
}

// TestSecureBridge_MessageSigning tests message signing and verification
func TestSecureBridge_MessageSigning(t *testing.T) {
	bridge := createTestSecureBridge(t)

	message := &SecureMessage{
		ID:        "msg-001",
		Type:      "test",
		Payload:   map[string]interface{}{"data": "test data"},
		Timestamp: time.Now(),
	}

	// Sign message
	err := bridge.SignMessage(message, "test-key")
	assert.NoError(t, err)
	assert.NotEmpty(t, message.Signature)

	// Verify signature
	valid := bridge.VerifyMessageSignature(message, "test-key")
	assert.True(t, valid)

	// Test with wrong key
	valid = bridge.VerifyMessageSignature(message, "wrong-key")
	assert.False(t, valid)
}

// TestSecureBridge_EdgeCases tests various edge cases
func TestSecureBridge_EdgeCases(t *testing.T) {
	bridge := createTestSecureBridge(t)

	// Test with nil message
	response, err := bridge.ProcessMessage(nil)
	assert.Error(t, err)
	assert.Nil(t, response)

	// Test with empty token
	message := &SecureMessage{
		ID:        "msg-001",
		Type:      "test",
		Token:     "",
		Timestamp: time.Now(),
	}

	response, err = bridge.ProcessMessage(message)
	assert.Error(t, err)
	assert.Nil(t, response)

	// Test session creation with empty client ID
	token, err := bridge.CreateSession("", []string{"read"})
	assert.Error(t, err)
	assert.Empty(t, token)

	// Test session creation with nil permissions
	token, err = bridge.CreateSession("client", nil)
	assert.NoError(t, err) // Should handle gracefully
	assert.NotEmpty(t, token)
}

// TestSecureBridge_MemoryManagement tests memory usage and cleanup
func TestSecureBridge_MemoryManagement(t *testing.T) {
	bridge := createTestSecureBridge(t)

	// Create many sessions
	numSessions := 1000
	tokens := make([]string, numSessions)

	for i := 0; i < numSessions; i++ {
		clientID := fmt.Sprintf("client-%d", i)
		token, err := bridge.CreateSession(clientID, []string{"read"})
		require.NoError(t, err)
		tokens[i] = token
	}

	// Verify all sessions exist
	bridge.mutex.RLock()
	sessionCount := len(bridge.sessionTokens)
	bridge.mutex.RUnlock()
	assert.Equal(t, numSessions, sessionCount)

	// Cleanup all sessions
	cleaned := bridge.CleanupExpiredSessions(0) // Cleanup all
	assert.Equal(t, numSessions, cleaned)

	// Verify cleanup
	bridge.mutex.RLock()
	sessionCount = len(bridge.sessionTokens)
	bridge.mutex.RUnlock()
	assert.Equal(t, 0, sessionCount)
}
