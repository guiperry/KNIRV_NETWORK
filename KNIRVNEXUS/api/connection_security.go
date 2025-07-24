package api

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"
)

// SecurityLevel represents the security level for connections
type SecurityLevel string

const (
	SecurityLevelNone     SecurityLevel = "none"
	SecurityLevelBasic    SecurityLevel = "basic"
	SecurityLevelStandard SecurityLevel = "standard"
	SecurityLevelHigh     SecurityLevel = "high"
)

// Permission represents a specific permission
type Permission string

const (
	PermissionRead    Permission = "read"
	PermissionWrite   Permission = "write"
	PermissionExecute Permission = "execute"
	PermissionDelete  Permission = "delete"
	PermissionAdmin   Permission = "admin"
)

// SecurityContext holds security information for a connection
type SecurityContext struct {
	UserID      int64                  `json:"userId"`
	SessionID   string                 `json:"sessionId"`
	Permissions []Permission           `json:"permissions"`
	Constraints map[string]interface{} `json:"constraints"`
	ExpiresAt   time.Time              `json:"expiresAt"`
	CreatedAt   time.Time              `json:"createdAt"`
}

// ConnectionSecurityManager manages security for target system connections
type ConnectionSecurityManager struct {
	sessions map[string]*SecurityContext
	mutex    sync.RWMutex
}

// NewConnectionSecurityManager creates a new security manager
func NewConnectionSecurityManager() *ConnectionSecurityManager {
	return &ConnectionSecurityManager{
		sessions: make(map[string]*SecurityContext),
	}
}

// CreateSecurityContext creates a new security context for a user
func (m *ConnectionSecurityManager) CreateSecurityContext(userID int64, permissions []Permission, constraints map[string]interface{}) (*SecurityContext, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Generate session ID
	sessionID, err := generateSessionID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session ID: %v", err)
	}

	// Create security context
	ctx := &SecurityContext{
		UserID:      userID,
		SessionID:   sessionID,
		Permissions: permissions,
		Constraints: constraints,
		ExpiresAt:   time.Now().Add(24 * time.Hour), // Default 24 hour expiry
		CreatedAt:   time.Now(),
	}

	// Store session
	m.sessions[sessionID] = ctx

	return ctx, nil
}

// ValidateSecurityContext validates a security context
func (m *ConnectionSecurityManager) ValidateSecurityContext(sessionID string) (*SecurityContext, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	ctx, ok := m.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("invalid session ID")
	}

	// Check expiry
	if time.Now().After(ctx.ExpiresAt) {
		return nil, fmt.Errorf("session expired")
	}

	return ctx, nil
}

// CheckPermission checks if a security context has a specific permission
func (m *ConnectionSecurityManager) CheckPermission(sessionID string, permission Permission) error {
	ctx, err := m.ValidateSecurityContext(sessionID)
	if err != nil {
		return err
	}

	// Check if user has the required permission
	for _, p := range ctx.Permissions {
		if p == permission || p == PermissionAdmin {
			return nil
		}
	}

	return fmt.Errorf("permission denied: %s", permission)
}

// RevokeSession revokes a security session
func (m *ConnectionSecurityManager) RevokeSession(sessionID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.sessions, sessionID)
	return nil
}

// CleanupExpiredSessions removes expired sessions
func (m *ConnectionSecurityManager) CleanupExpiredSessions() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	now := time.Now()
	for sessionID, ctx := range m.sessions {
		if now.After(ctx.ExpiresAt) {
			delete(m.sessions, sessionID)
		}
	}
}

// SecureTargetSystemConnection wraps a target system connection with security
type SecureTargetSystemConnection struct {
	connection      TargetSystemConnection
	securityManager *ConnectionSecurityManager
	sessionID       string
	securityLevel   SecurityLevel
	allowedOps      map[string][]Permission
}

// NewSecureTargetSystemConnection creates a secure wrapper for a target system connection
func NewSecureTargetSystemConnection(
	connection TargetSystemConnection,
	securityManager *ConnectionSecurityManager,
	sessionID string,
	securityLevel SecurityLevel,
) *SecureTargetSystemConnection {
	return &SecureTargetSystemConnection{
		connection:      connection,
		securityManager: securityManager,
		sessionID:       sessionID,
		securityLevel:   securityLevel,
		allowedOps:      getDefaultOperationPermissions(connection.GetType()),
	}
}

// Connect establishes the connection with security checks
func (s *SecureTargetSystemConnection) Connect(ctx context.Context) error {
	// Check execute permission for connection
	if err := s.securityManager.CheckPermission(s.sessionID, PermissionExecute); err != nil {
		return fmt.Errorf("connection denied: %v", err)
	}

	return s.connection.Connect(ctx)
}

// Disconnect closes the connection
func (s *SecureTargetSystemConnection) Disconnect(ctx context.Context) error {
	return s.connection.Disconnect(ctx)
}

// IsConnected returns the connection status
func (s *SecureTargetSystemConnection) IsConnected() bool {
	return s.connection.IsConnected()
}

// GetCapabilities returns available capabilities filtered by permissions
func (s *SecureTargetSystemConnection) GetCapabilities() []string {
	allCapabilities := s.connection.GetCapabilities()
	var allowedCapabilities []string

	for _, capability := range allCapabilities {
		if s.isOperationAllowed(capability) {
			allowedCapabilities = append(allowedCapabilities, capability)
		}
	}

	return allowedCapabilities
}

// Execute executes an operation with security checks
func (s *SecureTargetSystemConnection) Execute(ctx context.Context, operation string, params map[string]interface{}) (interface{}, error) {
	// Check if operation is allowed
	if !s.isOperationAllowed(operation) {
		return nil, fmt.Errorf("operation not allowed: %s", operation)
	}

	// Check required permissions for the operation
	requiredPerms, ok := s.allowedOps[operation]
	if ok {
		for _, perm := range requiredPerms {
			if err := s.securityManager.CheckPermission(s.sessionID, perm); err != nil {
				return nil, fmt.Errorf("operation denied: %v", err)
			}
		}
	}

	// Apply security constraints based on security level
	if err := s.applySecurityConstraints(operation, params); err != nil {
		return nil, fmt.Errorf("security constraint violation: %v", err)
	}

	return s.connection.Execute(ctx, operation, params)
}

// GetStatus returns detailed status information
func (s *SecureTargetSystemConnection) GetStatus() map[string]interface{} {
	status := s.connection.GetStatus()

	// Add security information
	status["securityLevel"] = s.securityLevel
	status["sessionID"] = s.sessionID

	// Check if user has admin permission to see detailed security info
	if err := s.securityManager.CheckPermission(s.sessionID, PermissionAdmin); err == nil {
		if ctx, err := s.securityManager.ValidateSecurityContext(s.sessionID); err == nil {
			status["securityContext"] = map[string]interface{}{
				"userID":      ctx.UserID,
				"permissions": ctx.Permissions,
				"expiresAt":   ctx.ExpiresAt,
			}
		}
	}

	return status
}

// GetType returns the target system type
func (s *SecureTargetSystemConnection) GetType() TargetSystemType {
	return s.connection.GetType()
}

// Helper methods

// isOperationAllowed checks if an operation is allowed based on permissions
func (s *SecureTargetSystemConnection) isOperationAllowed(operation string) bool {
	requiredPerms, ok := s.allowedOps[operation]
	if !ok {
		// If operation is not in the allowed list, check if user has admin permission
		return s.securityManager.CheckPermission(s.sessionID, PermissionAdmin) == nil
	}

	// Check if user has any of the required permissions
	for _, perm := range requiredPerms {
		if s.securityManager.CheckPermission(s.sessionID, perm) == nil {
			return true
		}
	}

	return false
}

// applySecurityConstraints applies security constraints based on security level
func (s *SecureTargetSystemConnection) applySecurityConstraints(operation string, params map[string]interface{}) error {
	switch s.securityLevel {
	case SecurityLevelNone:
		return nil
	case SecurityLevelBasic:
		return s.applyBasicConstraints(operation, params)
	case SecurityLevelStandard:
		return s.applyStandardConstraints(operation, params)
	case SecurityLevelHigh:
		return s.applyHighConstraints(operation, params)
	default:
		return fmt.Errorf("unknown security level: %s", s.securityLevel)
	}
}

// applyBasicConstraints applies basic security constraints
func (s *SecureTargetSystemConnection) applyBasicConstraints(operation string, params map[string]interface{}) error {
	// Basic constraints: limit file sizes, prevent dangerous operations
	if operation == "write_file" {
		if data, ok := params["data"].(string); ok && len(data) > 10*1024*1024 { // 10MB limit
			return fmt.Errorf("file size exceeds limit")
		}
	}
	return nil
}

// applyStandardConstraints applies standard security constraints
func (s *SecureTargetSystemConnection) applyStandardConstraints(operation string, params map[string]interface{}) error {
	if err := s.applyBasicConstraints(operation, params); err != nil {
		return err
	}

	// Standard constraints: path validation, command filtering
	if operation == "read_file" || operation == "write_file" {
		if path, ok := params["path"].(string); ok {
			if strings.Contains(path, "..") || strings.HasPrefix(path, "/etc") || strings.HasPrefix(path, "/sys") {
				return fmt.Errorf("path access denied")
			}
		}
	}

	if operation == "execute_command" {
		if command, ok := params["command"].(string); ok {
			dangerousCommands := []string{"rm -rf", "sudo", "su", "chmod 777", "mkfs"}
			for _, dangerous := range dangerousCommands {
				if strings.Contains(strings.ToLower(command), dangerous) {
					return fmt.Errorf("dangerous command blocked")
				}
			}
		}
	}

	return nil
}

// applyHighConstraints applies high security constraints
func (s *SecureTargetSystemConnection) applyHighConstraints(operation string, params map[string]interface{}) error {
	if err := s.applyStandardConstraints(operation, params); err != nil {
		return err
	}

	// High constraints: very restrictive
	writeOps := []string{"write_file", "delete_file", "execute_command", "create_directory"}
	for _, writeOp := range writeOps {
		if operation == writeOp {
			return fmt.Errorf("write operations disabled in high security mode")
		}
	}

	return nil
}

// getDefaultOperationPermissions returns default permission mappings for operations
func getDefaultOperationPermissions(targetType TargetSystemType) map[string][]Permission {
	switch targetType {
	case TargetTypeBrowser:
		return map[string][]Permission{
			"navigate":       {PermissionRead},
			"click":          {PermissionExecute},
			"type":           {PermissionWrite},
			"screenshot":     {PermissionRead},
			"get_text":       {PermissionRead},
			"execute_script": {PermissionExecute},
			"set_cookies":    {PermissionWrite},
		}
	case TargetTypeFilesystem:
		return map[string][]Permission{
			"read_file":        {PermissionRead},
			"write_file":       {PermissionWrite},
			"delete_file":      {PermissionDelete},
			"list_directory":   {PermissionRead},
			"create_directory": {PermissionWrite},
		}
	case TargetTypeDatabase:
		return map[string][]Permission{
			"query":       {PermissionRead},
			"execute":     {PermissionWrite},
			"transaction": {PermissionWrite},
			"list_tables": {PermissionRead},
		}
	case TargetTypeSystem:
		return map[string][]Permission{
			"execute_command": {PermissionExecute},
			"get_system_info": {PermissionRead},
			"get_environment": {PermissionRead},
		}
	default:
		return map[string][]Permission{}
	}
}

// generateSessionID generates a secure session ID
func generateSessionID() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	hash := sha256.Sum256(bytes)
	return hex.EncodeToString(hash[:]), nil
}
