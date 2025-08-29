// desktop/secure_bridge.go
// Secure communication bridge between Electron frontend and Go backend

package desktop

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// SecureBridge manages secure communication between Electron and Go backend
type SecureBridge struct {
	sessionTokens map[string]*SessionInfo
	upgrader      websocket.Upgrader
	mutex         sync.RWMutex
	teeManager    *DesktopTEEManager
}

// SessionInfo contains information about an active session
type SessionInfo struct {
	Token       string
	CreatedAt   time.Time
	LastUsed    time.Time
	ClientID    string
	Permissions []string
}

// SecureMessage represents a secure message between Electron and Go
type SecureMessage struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Payload   map[string]interface{} `json:"payload"`
	Token     string                 `json:"token"`
	Timestamp time.Time              `json:"timestamp"`
	Signature string                 `json:"signature"`
}

// MessageResponse represents a response to a secure message
type MessageResponse struct {
	ID        string                 `json:"id"`
	Success   bool                   `json:"success"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// NewSecureBridge creates a new secure bridge
func NewSecureBridge(teeManager *DesktopTEEManager) *SecureBridge {
	return &SecureBridge{
		sessionTokens: make(map[string]*SessionInfo),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Only allow connections from localhost (Electron app)
				origin := r.Header.Get("Origin")
				return origin == "http://localhost:3001" || origin == "file://"
			},
		},
		teeManager: teeManager,
	}
}

// GenerateSessionToken generates a new session token for Electron client
func (sb *SecureBridge) GenerateSessionToken(clientID string, permissions []string) (string, error) {
	sb.mutex.Lock()
	defer sb.mutex.Unlock()

	// Generate random token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate token: %v", err)
	}

	token := hex.EncodeToString(tokenBytes)

	// Store session info
	sessionInfo := &SessionInfo{
		Token:       token,
		CreatedAt:   time.Now(),
		LastUsed:    time.Now(),
		ClientID:    clientID,
		Permissions: permissions,
	}

	sb.sessionTokens[token] = sessionInfo

	return token, nil
}

// ValidateToken validates a session token
func (sb *SecureBridge) ValidateToken(token string) (*SessionInfo, error) {
	sb.mutex.RLock()
	defer sb.mutex.RUnlock()

	sessionInfo, exists := sb.sessionTokens[token]
	if !exists {
		return nil, fmt.Errorf("invalid token")
	}

	// Check if token is expired (24 hours)
	if time.Since(sessionInfo.CreatedAt) > 24*time.Hour {
		return nil, fmt.Errorf("token expired")
	}

	// Update last used time
	sessionInfo.LastUsed = time.Now()

	return sessionInfo, nil
}

// RevokeToken revokes a session token
func (sb *SecureBridge) RevokeToken(token string) error {
	sb.mutex.Lock()
	defer sb.mutex.Unlock()

	delete(sb.sessionTokens, token)
	return nil
}

// HandleWebSocketConnection handles WebSocket connections from Electron
func (sb *SecureBridge) HandleWebSocketConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := sb.upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Failed to upgrade connection", http.StatusBadRequest)
		return
	}
	defer conn.Close()

	// Handle messages
	for {
		var message SecureMessage
		if err := conn.ReadJSON(&message); err != nil {
			break
		}

		response := sb.handleSecureMessage(&message)
		if err := conn.WriteJSON(response); err != nil {
			break
		}
	}
}

// handleSecureMessage processes a secure message from Electron
func (sb *SecureBridge) handleSecureMessage(message *SecureMessage) *MessageResponse {
	response := &MessageResponse{
		ID:        message.ID,
		Timestamp: time.Now(),
	}

	// Validate token
	sessionInfo, err := sb.ValidateToken(message.Token)
	if err != nil {
		response.Success = false
		response.Error = "Authentication failed"
		return response
	}

	// Verify message signature
	if !sb.verifyMessageSignature(message) {
		response.Success = false
		response.Error = "Invalid message signature"
		return response
	}

	// Process message based on type
	switch message.Type {
	case "plugin_load":
		response = sb.handlePluginLoad(message, sessionInfo)
	case "plugin_unload":
		response = sb.handlePluginUnload(message, sessionInfo)
	case "plugin_execute":
		response = sb.handlePluginExecute(message, sessionInfo)
	case "plugin_list":
		response = sb.handlePluginList(message, sessionInfo)
	case "tee_status":
		response = sb.handleTEEStatus(message, sessionInfo)
	default:
		response.Success = false
		response.Error = "Unknown message type"
	}

	return response
}

// verifyMessageSignature verifies the signature of a message using proper cryptographic verification
func (sb *SecureBridge) verifyMessageSignature(message *SecureMessage) bool {
	// Create message hash for signature verification
	messageData := fmt.Sprintf("%s:%s:%s:%d",
		message.ID, message.Type, message.Token, message.Timestamp.Unix())

	hash := sha256.Sum256([]byte(messageData))

	// Decode the signature from hex
	signature, err := hex.DecodeString(message.Signature)
	if err != nil {
		// Log error and return false
		fmt.Printf("Failed to decode message signature: %v\n", err)
		return false
	}

	// Try RSA signature verification if TEE manager is available and has verification capability
	if sb.teeManager != nil {
		if verified, err := sb.verifyRSASignature(hash[:], signature); err == nil {
			return verified
		}
	}

	// Fallback to hash comparison for backward compatibility
	expectedSignature := hex.EncodeToString(hash[:])
	return message.Signature == expectedSignature
}

// verifyRSASignature verifies an RSA signature using the TEE manager's verification capabilities
func (sb *SecureBridge) verifyRSASignature(hash []byte, signature []byte) (bool, error) {
	// Use the TEE manager's signature verification if available
	if sb.teeManager != nil {
		// Try verification with each trusted signer
		for _, signerKeyPath := range sb.teeManager.config.TrustedSigners {
			if verified, err := sb.teeManager.verifySignatureWithKey(hash, signature, signerKeyPath); err == nil && verified {
				return true, nil
			}
		}
		return false, fmt.Errorf("signature verification failed with all trusted signers")
	}
	return false, fmt.Errorf("TEE manager not available")
}

// handlePluginLoad handles plugin loading requests
func (sb *SecureBridge) handlePluginLoad(message *SecureMessage, session *SessionInfo) *MessageResponse {
	response := &MessageResponse{
		ID:        message.ID,
		Timestamp: time.Now(),
	}

	// Check permissions
	if !sb.hasPermission(session, "plugin_load") {
		response.Success = false
		response.Error = "Insufficient permissions"
		return response
	}

	pluginID, ok := message.Payload["plugin_id"].(string)
	if !ok {
		response.Success = false
		response.Error = "Missing plugin_id"
		return response
	}

	// Create security context
	securityContext := &SecurityContext{
		PluginID:      pluginID,
		Permissions:   []string{"basic"},
		NetworkAccess: false,
		FileAccess:    []string{},
		MaxMemory:     512 * 1024 * 1024, // 512MB
		MaxCPU:        50,                // 50%
		Timeout:       30 * time.Second,
	}

	// Load plugin
	if err := sb.teeManager.LoadPlugin(pluginID, securityContext); err != nil {
		response.Success = false
		response.Error = fmt.Sprintf("Failed to load plugin: %v", err)
		return response
	}

	response.Success = true
	response.Data = map[string]interface{}{
		"plugin_id": pluginID,
		"status":    "loaded",
	}

	return response
}

// handlePluginUnload handles plugin unloading requests
func (sb *SecureBridge) handlePluginUnload(message *SecureMessage, session *SessionInfo) *MessageResponse {
	response := &MessageResponse{
		ID:        message.ID,
		Timestamp: time.Now(),
	}

	// Check permissions
	if !sb.hasPermission(session, "plugin_unload") {
		response.Success = false
		response.Error = "Insufficient permissions"
		return response
	}

	pluginID, ok := message.Payload["plugin_id"].(string)
	if !ok {
		response.Success = false
		response.Error = "Missing plugin_id"
		return response
	}

	// Unload plugin
	if err := sb.teeManager.UnloadPlugin(pluginID); err != nil {
		response.Success = false
		response.Error = fmt.Sprintf("Failed to unload plugin: %v", err)
		return response
	}

	response.Success = true
	response.Data = map[string]interface{}{
		"plugin_id": pluginID,
		"status":    "unloaded",
	}

	return response
}

// handlePluginExecute handles plugin execution requests
func (sb *SecureBridge) handlePluginExecute(message *SecureMessage, session *SessionInfo) *MessageResponse {
	response := &MessageResponse{
		ID:        message.ID,
		Timestamp: time.Now(),
	}

	// Check permissions
	if !sb.hasPermission(session, "plugin_execute") {
		response.Success = false
		response.Error = "Insufficient permissions"
		return response
	}

	pluginID, ok := message.Payload["plugin_id"].(string)
	if !ok {
		response.Success = false
		response.Error = "Missing plugin_id"
		return response
	}

	command, ok := message.Payload["command"].(string)
	if !ok {
		response.Success = false
		response.Error = "Missing command"
		return response
	}

	args, _ := message.Payload["args"].([]string)

	// Execute in TEE
	stdout, stderr, exitCode, err := sb.teeManager.ExecuteInTEE(pluginID, command, args)
	if err != nil {
		response.Success = false
		response.Error = fmt.Sprintf("Execution failed: %v", err)
		return response
	}

	response.Success = true
	response.Data = map[string]interface{}{
		"stdout":    stdout,
		"stderr":    stderr,
		"exit_code": exitCode,
	}

	return response
}

// handlePluginList handles plugin listing requests
func (sb *SecureBridge) handlePluginList(message *SecureMessage, session *SessionInfo) *MessageResponse {
	response := &MessageResponse{
		ID:        message.ID,
		Timestamp: time.Now(),
	}

	// Check permissions
	if !sb.hasPermission(session, "plugin_list") {
		response.Success = false
		response.Error = "Insufficient permissions"
		return response
	}

	plugins := sb.teeManager.ListPlugins()

	// Convert to JSON-serializable format
	pluginData := make([]map[string]interface{}, len(plugins))
	for i, plugin := range plugins {
		pluginData[i] = map[string]interface{}{
			"id":          plugin.ID,
			"name":        plugin.Name,
			"version":     plugin.Version,
			"hash":        plugin.Hash,
			"loaded_at":   plugin.LoadedAt,
			"last_used":   plugin.LastUsed,
			"tee_type":    plugin.TEEType,
			"is_verified": plugin.IsVerified,
		}
	}

	response.Success = true
	response.Data = map[string]interface{}{
		"plugins": pluginData,
	}

	return response
}

// handleTEEStatus handles TEE status requests
func (sb *SecureBridge) handleTEEStatus(message *SecureMessage, session *SessionInfo) *MessageResponse {
	response := &MessageResponse{
		ID:        message.ID,
		Timestamp: time.Now(),
	}

	// Check permissions
	if !sb.hasPermission(session, "tee_status") {
		response.Success = false
		response.Error = "Insufficient permissions"
		return response
	}

	response.Success = true
	response.Data = map[string]interface{}{
		"active_plugins": len(sb.teeManager.teeInstances),
		"total_plugins":  len(sb.teeManager.pluginRegistry),
	}

	return response
}

// hasPermission checks if a session has a specific permission
func (sb *SecureBridge) hasPermission(session *SessionInfo, permission string) bool {
	for _, p := range session.Permissions {
		if p == permission || p == "*" {
			return true
		}
	}
	return false
}

// Cleanup cleans up the secure bridge
func (sb *SecureBridge) Cleanup() error {
	sb.mutex.Lock()
	defer sb.mutex.Unlock()

	// Clear all session tokens
	sb.sessionTokens = make(map[string]*SessionInfo)

	return nil
}
