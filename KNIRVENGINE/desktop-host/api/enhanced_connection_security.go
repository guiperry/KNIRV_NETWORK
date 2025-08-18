package api

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// EnhancedSecurityLevel represents the enhanced security level for connections
type EnhancedSecurityLevel string

const (
	EnhancedSecurityLevelBasic    EnhancedSecurityLevel = "basic"
	EnhancedSecurityLevelStandard EnhancedSecurityLevel = "standard"
	EnhancedSecurityLevelHigh     EnhancedSecurityLevel = "high"
	EnhancedSecurityLevelExtreme  EnhancedSecurityLevel = "extreme"
)

// EnhancedPermission represents a specific permission with fine-grained control
type EnhancedPermission struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Resource    string                 `json:"resource"`
	Action      string                 `json:"action"`
	Constraints map[string]interface{} `json:"constraints"`
}

// EnhancedSecurityContext holds enhanced security information for a connection
type EnhancedSecurityContext struct {
	UserID           int64                  `json:"userId"`
	SessionID        string                 `json:"sessionId"`
	Permissions      []EnhancedPermission   `json:"permissions"`
	Constraints      map[string]interface{} `json:"constraints"`
	ExpiresAt        time.Time              `json:"expiresAt"`
	CreatedAt        time.Time              `json:"createdAt"`
	LastActivity     time.Time              `json:"lastActivity"`
	SecurityLevel    EnhancedSecurityLevel  `json:"securityLevel"`
	AuthMethod       string                 `json:"authMethod"`
	IPAddress        string                 `json:"ipAddress"`
	UserAgent        string                 `json:"userAgent"`
	MFAVerified      bool                   `json:"mfaVerified"`
	RiskScore        float64                `json:"riskScore"`
	AuditTrail       []SecurityAuditEvent   `json:"auditTrail"`
	RevocationReason string                 `json:"revocationReason"`
}

// SecurityAuditEvent represents a security-related event for audit logging
type SecurityAuditEvent struct {
	Timestamp time.Time              `json:"timestamp"`
	EventType string                 `json:"eventType"`
	Resource  string                 `json:"resource"`
	Action    string                 `json:"action"`
	Status    string                 `json:"status"`
	IPAddress string                 `json:"ipAddress"`
	UserAgent string                 `json:"userAgent"`
	SessionID string                 `json:"sessionId"`
	UserID    int64                  `json:"userId"`
	Details   map[string]interface{} `json:"details"`
	RiskScore float64                `json:"riskScore"`
	Severity  string                 `json:"severity"`
}

// EnhancedConnectionSecurityManager manages enhanced security for target system connections
type EnhancedConnectionSecurityManager struct {
	sessions       map[string]*EnhancedSecurityContext
	permissionSets map[string][]EnhancedPermission
	auditLog       []SecurityAuditEvent
	mutex          sync.RWMutex
	auditMutex     sync.RWMutex
}

// NewEnhancedConnectionSecurityManager creates a new enhanced security manager
func NewEnhancedConnectionSecurityManager() *EnhancedConnectionSecurityManager {
	manager := &EnhancedConnectionSecurityManager{
		sessions:       make(map[string]*EnhancedSecurityContext),
		permissionSets: make(map[string][]EnhancedPermission),
		auditLog:       make([]SecurityAuditEvent, 0),
	}

	// Initialize default permission sets
	manager.initializeDefaultPermissionSets()

	return manager
}

// initializeDefaultPermissionSets sets up default permission sets
func (m *EnhancedConnectionSecurityManager) initializeDefaultPermissionSets() {
	// Read-only permissions
	m.permissionSets["readonly"] = []EnhancedPermission{
		{
			Name:        "read_files",
			Description: "Read files from the target system",
			Resource:    "filesystem",
			Action:      "read",
			Constraints: map[string]interface{}{
				"max_file_size": 10 * 1024 * 1024, // 10MB
				"allowed_paths": []string{"/home", "/tmp"},
			},
		},
		{
			Name:        "list_directories",
			Description: "List directories on the target system",
			Resource:    "filesystem",
			Action:      "list",
			Constraints: map[string]interface{}{
				"allowed_paths": []string{"/home", "/tmp"},
			},
		},
		{
			Name:        "read_database",
			Description: "Read from databases",
			Resource:    "database",
			Action:      "read",
			Constraints: map[string]interface{}{
				"max_rows":       1000,
				"allowed_tables": []string{"public_*"},
			},
		},
	}

	// Standard permissions (read + limited write)
	m.permissionSets["standard"] = append(m.permissionSets["readonly"], []EnhancedPermission{
		{
			Name:        "write_files",
			Description: "Write files to the target system",
			Resource:    "filesystem",
			Action:      "write",
			Constraints: map[string]interface{}{
				"max_file_size": 5 * 1024 * 1024, // 5MB
				"allowed_paths": []string{"/home/user/documents", "/tmp"},
			},
		},
		{
			Name:        "execute_commands",
			Description: "Execute commands on the target system",
			Resource:    "system",
			Action:      "execute",
			Constraints: map[string]interface{}{
				"allowed_commands": []string{"ls", "cat", "grep", "find"},
				"timeout":          30, // 30 seconds
			},
		},
	}...)

	// Admin permissions (full access)
	m.permissionSets["admin"] = append(m.permissionSets["standard"], []EnhancedPermission{
		{
			Name:        "delete_files",
			Description: "Delete files from the target system",
			Resource:    "filesystem",
			Action:      "delete",
			Constraints: map[string]interface{}{
				"allowed_paths": []string{"/home/user/documents", "/tmp"},
			},
		},
		{
			Name:        "modify_system",
			Description: "Modify system settings",
			Resource:    "system",
			Action:      "modify",
			Constraints: map[string]interface{}{
				"allowed_settings": []string{"user_preferences", "application_settings"},
			},
		},
		{
			Name:        "manage_users",
			Description: "Manage users on the target system",
			Resource:    "users",
			Action:      "manage",
			Constraints: map[string]interface{}{
				"excluded_users": []string{"root", "admin"},
			},
		},
	}...)
}

// CreateSecurityContext creates a new enhanced security context for a user
func (m *EnhancedConnectionSecurityManager) CreateSecurityContext(
	userID int64,
	permissionSet string,
	securityLevel EnhancedSecurityLevel,
	authMethod string,
	ipAddress string,
	userAgent string,
	mfaVerified bool,
	constraints map[string]interface{},
) (*EnhancedSecurityContext, error) {
	m.mutex.Lock()

	// Generate session ID
	sessionID, err := generateEnhancedSessionID()
	if err != nil {
		m.mutex.Unlock()
		return nil, fmt.Errorf("failed to generate session ID: %v", err)
	}

	// Get permissions for the specified permission set
	permissions, ok := m.permissionSets[permissionSet]
	if !ok {
		m.mutex.Unlock()
		return nil, fmt.Errorf("invalid permission set: %s", permissionSet)
	}

	// Calculate risk score based on security factors
	riskScore := calculateRiskScore(securityLevel, authMethod, mfaVerified)

	// Create security context
	now := time.Now()
	ctx := &EnhancedSecurityContext{
		UserID:        userID,
		SessionID:     sessionID,
		Permissions:   permissions,
		Constraints:   constraints,
		ExpiresAt:     now.Add(24 * time.Hour), // Default 24 hour expiry
		CreatedAt:     now,
		LastActivity:  now,
		SecurityLevel: securityLevel,
		AuthMethod:    authMethod,
		IPAddress:     ipAddress,
		UserAgent:     userAgent,
		MFAVerified:   mfaVerified,
		RiskScore:     riskScore,
		AuditTrail:    make([]SecurityAuditEvent, 0),
	}

	// Store session
	m.sessions[sessionID] = ctx

	// Create audit event (but don't log it yet to avoid deadlock)
	auditEvent := SecurityAuditEvent{
		Timestamp: now,
		EventType: "session_created",
		Resource:  "security_context",
		Action:    "create",
		Status:    "success",
		IPAddress: ipAddress,
		UserAgent: userAgent,
		SessionID: sessionID,
		UserID:    userID,
		Details: map[string]interface{}{
			"security_level": string(securityLevel),
			"auth_method":    authMethod,
			"mfa_verified":   mfaVerified,
			"permission_set": permissionSet,
		},
		RiskScore: riskScore,
		Severity:  "info",
	}

	// Release the mutex before logging to avoid deadlock
	m.mutex.Unlock()

	// Log security event after releasing the mutex
	m.LogSecurityEvent(auditEvent)

	// Re-acquire mutex is not needed since we're returning
	return ctx, nil
}

// ValidateSecurityContext validates an enhanced security context
func (m *EnhancedConnectionSecurityManager) ValidateSecurityContext(
	sessionID string,
	ipAddress string,
	userAgent string,
) (*EnhancedSecurityContext, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	ctx, ok := m.sessions[sessionID]
	if !ok {
		m.logSecurityEventUnsafe(SecurityAuditEvent{
			Timestamp: time.Now(),
			EventType: "session_validation",
			Resource:  "security_context",
			Action:    "validate",
			Status:    "failure",
			IPAddress: ipAddress,
			UserAgent: userAgent,
			SessionID: sessionID,
			UserID:    0,
			Details: map[string]interface{}{
				"reason": "invalid_session_id",
			},
			RiskScore: 0.8,
			Severity:  "warning",
		})
		return nil, fmt.Errorf("invalid session ID")
	}

	// Check expiry
	if time.Now().After(ctx.ExpiresAt) {
		m.logSecurityEventUnsafe(SecurityAuditEvent{
			Timestamp: time.Now(),
			EventType: "session_validation",
			Resource:  "security_context",
			Action:    "validate",
			Status:    "failure",
			IPAddress: ipAddress,
			UserAgent: userAgent,
			SessionID: sessionID,
			UserID:    ctx.UserID,
			Details: map[string]interface{}{
				"reason": "session_expired",
			},
			RiskScore: 0.5,
			Severity:  "info",
		})
		return nil, fmt.Errorf("session expired")
	}

	// Check for IP address change (potential session hijacking)
	if ctx.SecurityLevel == EnhancedSecurityLevelHigh || ctx.SecurityLevel == EnhancedSecurityLevelExtreme {
		if ctx.IPAddress != ipAddress {
			m.logSecurityEventUnsafe(SecurityAuditEvent{
				Timestamp: time.Now(),
				EventType: "session_validation",
				Resource:  "security_context",
				Action:    "validate",
				Status:    "failure",
				IPAddress: ipAddress,
				UserAgent: userAgent,
				SessionID: sessionID,
				UserID:    ctx.UserID,
				Details: map[string]interface{}{
					"reason":         "ip_address_changed",
					"original_ip":    ctx.IPAddress,
					"new_ip":         ipAddress,
					"security_level": string(ctx.SecurityLevel),
				},
				RiskScore: 0.9,
				Severity:  "critical",
			})
			return nil, fmt.Errorf("security violation: IP address changed")
		}
	}

	// Update last activity
	ctx.LastActivity = time.Now()

	// Log successful validation
	m.logSecurityEventUnsafe(SecurityAuditEvent{
		Timestamp: time.Now(),
		EventType: "session_validation",
		Resource:  "security_context",
		Action:    "validate",
		Status:    "success",
		IPAddress: ipAddress,
		UserAgent: userAgent,
		SessionID: sessionID,
		UserID:    ctx.UserID,
		Details:   map[string]interface{}{},
		RiskScore: 0.1,
		Severity:  "info",
	})

	return ctx, nil
}

// CheckPermission checks if a security context has a specific permission
func (m *EnhancedConnectionSecurityManager) CheckPermission(
	sessionID string,
	resource string,
	action string,
	resourcePath string,
	ipAddress string,
	userAgent string,
) error {
	ctx, err := m.ValidateSecurityContext(sessionID, ipAddress, userAgent)
	if err != nil {
		return err
	}

	// Check if user has the required permission
	for _, p := range ctx.Permissions {
		if p.Resource == resource && p.Action == action {
			// Check constraints
			if err := m.checkPermissionConstraints(p, resourcePath); err == nil {
				// Log successful permission check
				m.LogSecurityEvent(SecurityAuditEvent{
					Timestamp: time.Now(),
					EventType: "permission_check",
					Resource:  resource,
					Action:    action,
					Status:    "success",
					IPAddress: ipAddress,
					UserAgent: userAgent,
					SessionID: sessionID,
					UserID:    ctx.UserID,
					Details: map[string]interface{}{
						"resource_path": resourcePath,
						"permission":    p.Name,
					},
					RiskScore: 0.1,
					Severity:  "info",
				})
				return nil
			}
		}
	}

	// Log failed permission check
	m.LogSecurityEvent(SecurityAuditEvent{
		Timestamp: time.Now(),
		EventType: "permission_check",
		Resource:  resource,
		Action:    action,
		Status:    "failure",
		IPAddress: ipAddress,
		UserAgent: userAgent,
		SessionID: sessionID,
		UserID:    ctx.UserID,
		Details: map[string]interface{}{
			"resource_path": resourcePath,
			"reason":        "permission_denied",
		},
		RiskScore: 0.7,
		Severity:  "warning",
	})

	return fmt.Errorf("permission denied: %s:%s for path %s", resource, action, resourcePath)
}

// checkPermissionConstraints checks if a permission's constraints are satisfied
func (m *EnhancedConnectionSecurityManager) checkPermissionConstraints(
	permission EnhancedPermission,
	resourcePath string,
) error {
	// Check path constraints for filesystem resources
	if permission.Resource == "filesystem" {
		if allowedPaths, ok := permission.Constraints["allowed_paths"].([]string); ok {
			pathAllowed := false
			for _, allowedPath := range allowedPaths {
				if strings.HasPrefix(resourcePath, allowedPath) {
					pathAllowed = true
					break
				}
			}
			if !pathAllowed {
				return fmt.Errorf("path not allowed: %s", resourcePath)
			}
		}
	}

	// Check command constraints for system resources
	if permission.Resource == "system" && permission.Action == "execute" {
		if allowedCommands, ok := permission.Constraints["allowed_commands"].([]string); ok {
			commandAllowed := false
			command := strings.Split(resourcePath, " ")[0] // Extract the command from the path
			for _, allowedCommand := range allowedCommands {
				if command == allowedCommand {
					commandAllowed = true
					break
				}
			}
			if !commandAllowed {
				return fmt.Errorf("command not allowed: %s", command)
			}
		}
	}

	// Check database constraints
	if permission.Resource == "database" {
		if allowedTables, ok := permission.Constraints["allowed_tables"].([]string); ok {
			tableAllowed := false
			for _, allowedTable := range allowedTables {
				if strings.HasSuffix(allowedTable, "*") {
					// Handle wildcard patterns
					prefix := allowedTable[:len(allowedTable)-1]
					if strings.HasPrefix(resourcePath, prefix) {
						tableAllowed = true
						break
					}
				} else if resourcePath == allowedTable {
					tableAllowed = true
					break
				}
			}
			if !tableAllowed {
				return fmt.Errorf("table not allowed: %s", resourcePath)
			}
		}
	}

	return nil
}

// RevokeSession revokes an enhanced security session
func (m *EnhancedConnectionSecurityManager) RevokeSession(
	sessionID string,
	reason string,
	ipAddress string,
	userAgent string,
) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	ctx, ok := m.sessions[sessionID]
	if !ok {
		return fmt.Errorf("invalid session ID")
	}

	// Set revocation reason
	ctx.RevocationReason = reason

	// Log revocation
	m.logSecurityEventUnsafe(SecurityAuditEvent{
		Timestamp: time.Now(),
		EventType: "session_revoked",
		Resource:  "security_context",
		Action:    "revoke",
		Status:    "success",
		IPAddress: ipAddress,
		UserAgent: userAgent,
		SessionID: sessionID,
		UserID:    ctx.UserID,
		Details: map[string]interface{}{
			"reason": reason,
		},
		RiskScore: 0.3,
		Severity:  "info",
	})

	delete(m.sessions, sessionID)
	return nil
}

// ExtendSession extends the expiry time of a session
func (m *EnhancedConnectionSecurityManager) ExtendSession(
	sessionID string,
	duration time.Duration,
	ipAddress string,
	userAgent string,
) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	ctx, ok := m.sessions[sessionID]
	if !ok {
		return fmt.Errorf("invalid session ID")
	}

	// Extend expiry time
	ctx.ExpiresAt = time.Now().Add(duration)

	// Log extension
	m.logSecurityEventUnsafe(SecurityAuditEvent{
		Timestamp: time.Now(),
		EventType: "session_extended",
		Resource:  "security_context",
		Action:    "extend",
		Status:    "success",
		IPAddress: ipAddress,
		UserAgent: userAgent,
		SessionID: sessionID,
		UserID:    ctx.UserID,
		Details: map[string]interface{}{
			"duration":   duration.String(),
			"new_expiry": ctx.ExpiresAt,
		},
		RiskScore: 0.2,
		Severity:  "info",
	})

	return nil
}

// logSecurityEventUnsafe logs a security event without acquiring the main mutex
// This should only be called when the caller already holds m.mutex
func (m *EnhancedConnectionSecurityManager) logSecurityEventUnsafe(event SecurityAuditEvent) {
	m.auditMutex.Lock()
	defer m.auditMutex.Unlock()

	// Add to global audit log
	m.auditLog = append(m.auditLog, event)

	// Add to session audit trail if applicable (we already hold m.mutex)
	if event.SessionID != "" {
		if ctx, ok := m.sessions[event.SessionID]; ok {
			ctx.AuditTrail = append(ctx.AuditTrail, event)
		}
	}

	// Log high severity events
	if event.Severity == "warning" || event.Severity == "critical" {
		log.Printf("Security Event [%s]: %s - %s:%s - %s - Session: %s, User: %d, IP: %s",
			event.Severity, event.EventType, event.Resource, event.Action, event.Status,
			event.SessionID, event.UserID, event.IPAddress)
	}
}

// LogSecurityEvent logs a security event to the audit log
func (m *EnhancedConnectionSecurityManager) LogSecurityEvent(event SecurityAuditEvent) {
	m.auditMutex.Lock()
	defer m.auditMutex.Unlock()

	// Add to global audit log
	m.auditLog = append(m.auditLog, event)

	// Add to session audit trail if applicable
	if event.SessionID != "" {
		m.mutex.RLock()
		ctx, ok := m.sessions[event.SessionID]
		m.mutex.RUnlock()

		if ok {
			m.mutex.Lock()
			ctx.AuditTrail = append(ctx.AuditTrail, event)
			m.mutex.Unlock()
		}
	}

	// Log high severity events
	if event.Severity == "warning" || event.Severity == "critical" {
		log.Printf("Security Event [%s]: %s - %s:%s - %s - Session: %s, User: %d, IP: %s",
			event.Severity, event.EventType, event.Resource, event.Action, event.Status,
			event.SessionID, event.UserID, event.IPAddress)
	}
}

// GetAuditLog returns the security audit log
func (m *EnhancedConnectionSecurityManager) GetAuditLog(
	startTime time.Time,
	endTime time.Time,
	eventTypes []string,
	severity string,
	userID int64,
	sessionID string,
) []SecurityAuditEvent {
	m.auditMutex.RLock()
	defer m.auditMutex.RUnlock()

	var filteredEvents []SecurityAuditEvent

	for _, event := range m.auditLog {
		// Filter by time range
		if !event.Timestamp.Before(startTime) && !event.Timestamp.After(endTime) {
			// Filter by event type if specified
			if len(eventTypes) > 0 {
				typeMatched := false
				for _, eventType := range eventTypes {
					if event.EventType == eventType {
						typeMatched = true
						break
					}
				}
				if !typeMatched {
					continue
				}
			}

			// Filter by severity if specified
			if severity != "" && event.Severity != severity {
				continue
			}

			// Filter by user ID if specified
			if userID != 0 && event.UserID != userID {
				continue
			}

			// Filter by session ID if specified
			if sessionID != "" && event.SessionID != sessionID {
				continue
			}

			filteredEvents = append(filteredEvents, event)
		}
	}

	return filteredEvents
}

// CleanupExpiredSessions removes expired sessions
func (m *EnhancedConnectionSecurityManager) CleanupExpiredSessions() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	now := time.Now()
	for sessionID, ctx := range m.sessions {
		if now.After(ctx.ExpiresAt) {
			// Log session expiry
			m.logSecurityEventUnsafe(SecurityAuditEvent{
				Timestamp: now,
				EventType: "session_expired",
				Resource:  "security_context",
				Action:    "cleanup",
				Status:    "success",
				IPAddress: ctx.IPAddress,
				UserAgent: ctx.UserAgent,
				SessionID: sessionID,
				UserID:    ctx.UserID,
				Details: map[string]interface{}{
					"created_at":    ctx.CreatedAt,
					"last_activity": ctx.LastActivity,
				},
				RiskScore: 0.1,
				Severity:  "info",
			})

			delete(m.sessions, sessionID)
		}
	}
}

// EnhancedSecureTargetSystemConnection wraps a target system connection with enhanced security
type EnhancedSecureTargetSystemConnection struct {
	connection      TargetSystemConnection
	securityManager *EnhancedConnectionSecurityManager
	sessionID       string
	securityLevel   EnhancedSecurityLevel
	ipAddress       string
	userAgent       string
}

// NewEnhancedSecureTargetSystemConnection creates a secure wrapper for a target system connection
func NewEnhancedSecureTargetSystemConnection(
	connection TargetSystemConnection,
	securityManager *EnhancedConnectionSecurityManager,
	sessionID string,
	securityLevel EnhancedSecurityLevel,
	ipAddress string,
	userAgent string,
) *EnhancedSecureTargetSystemConnection {
	return &EnhancedSecureTargetSystemConnection{
		connection:      connection,
		securityManager: securityManager,
		sessionID:       sessionID,
		securityLevel:   securityLevel,
		ipAddress:       ipAddress,
		userAgent:       userAgent,
	}
}

// Connect establishes the connection with enhanced security checks
func (s *EnhancedSecureTargetSystemConnection) Connect(ctx context.Context) error {
	// Check execute permission for connection
	if err := s.securityManager.CheckPermission(
		s.sessionID,
		"system",
		"connect",
		string(s.connection.GetType()),
		s.ipAddress,
		s.userAgent,
	); err != nil {
		return fmt.Errorf("connection denied: %v", err)
	}

	return s.connection.Connect(ctx)
}

// Disconnect closes the connection
func (s *EnhancedSecureTargetSystemConnection) Disconnect(ctx context.Context) error {
	return s.connection.Disconnect(ctx)
}

// IsConnected returns the connection status
func (s *EnhancedSecureTargetSystemConnection) IsConnected() bool {
	return s.connection.IsConnected()
}

// GetCapabilities returns available capabilities filtered by permissions
func (s *EnhancedSecureTargetSystemConnection) GetCapabilities() []string {
	allCapabilities := s.connection.GetCapabilities()
	var allowedCapabilities []string

	for _, capability := range allCapabilities {
		// Map capability to resource and action
		resource, action := mapCapabilityToResourceAction(capability)

		// Check if the capability is allowed
		err := s.securityManager.CheckPermission(
			s.sessionID,
			resource,
			action,
			"",
			s.ipAddress,
			s.userAgent,
		)

		if err == nil {
			allowedCapabilities = append(allowedCapabilities, capability)
		}
	}

	return allowedCapabilities
}

// Execute executes an operation with enhanced security checks
func (s *EnhancedSecureTargetSystemConnection) Execute(
	ctx context.Context,
	operation string,
	params map[string]interface{},
) (interface{}, error) {
	// Map operation to resource and action
	resource, action := mapOperationToResourceAction(operation, s.connection.GetType())

	// Get resource path from params
	resourcePath := getResourcePathFromParams(operation, params)

	// Check if operation is allowed
	if err := s.securityManager.CheckPermission(
		s.sessionID,
		resource,
		action,
		resourcePath,
		s.ipAddress,
		s.userAgent,
	); err != nil {
		return nil, fmt.Errorf("operation denied: %v", err)
	}

	// Apply security constraints based on security level
	if err := s.applySecurityConstraints(operation, params); err != nil {
		return nil, fmt.Errorf("security constraint violation: %v", err)
	}

	// Execute the operation
	result, err := s.connection.Execute(ctx, operation, params)

	// Log the execution result
	s.securityManager.LogSecurityEvent(SecurityAuditEvent{
		Timestamp: time.Now(),
		EventType: "operation_executed",
		Resource:  resource,
		Action:    action,
		Status: func() string {
			if err == nil {
				return "success"
			} else {
				return "failure"
			}
		}(),
		IPAddress: s.ipAddress,
		UserAgent: s.userAgent,
		SessionID: s.sessionID,
		UserID:    0, // We don't have user ID here, would need to be passed in
		Details: map[string]interface{}{
			"operation":     operation,
			"resource_path": resourcePath,
			"error": func() string {
				if err != nil {
					return err.Error()
				} else {
					return ""
				}
			}(),
		},
		RiskScore: func() float64 {
			if err == nil {
				return 0.2
			} else {
				return 0.6
			}
		}(),
		Severity: func() string {
			if err == nil {
				return "info"
			} else {
				return "warning"
			}
		}(),
	})

	return result, err
}

// GetStatus returns detailed status information
func (s *EnhancedSecureTargetSystemConnection) GetStatus() map[string]interface{} {
	status := s.connection.GetStatus()

	// Add security information
	status["securityLevel"] = s.securityLevel
	status["sessionID"] = s.sessionID

	return status
}

// GetType returns the target system type
func (s *EnhancedSecureTargetSystemConnection) GetType() TargetSystemType {
	return s.connection.GetType()
}

// Helper methods

// applySecurityConstraints applies security constraints based on security level
func (s *EnhancedSecureTargetSystemConnection) applySecurityConstraints(
	operation string,
	params map[string]interface{},
) error {
	switch s.securityLevel {
	case EnhancedSecurityLevelBasic:
		return s.applyBasicConstraints(operation, params)
	case EnhancedSecurityLevelStandard:
		return s.applyStandardConstraints(operation, params)
	case EnhancedSecurityLevelHigh:
		return s.applyHighConstraints(operation, params)
	case EnhancedSecurityLevelExtreme:
		return s.applyExtremeConstraints(operation, params)
	default:
		return fmt.Errorf("unknown security level: %s", s.securityLevel)
	}
}

// applyBasicConstraints applies basic security constraints
func (s *EnhancedSecureTargetSystemConnection) applyBasicConstraints(
	operation string,
	params map[string]interface{},
) error {
	// Basic constraints: limit file sizes, prevent dangerous operations
	if operation == "write_file" {
		if data, ok := params["data"].(string); ok && len(data) > 10*1024*1024 { // 10MB limit
			return fmt.Errorf("file size exceeds limit")
		}
	}
	return nil
}

// applyStandardConstraints applies standard security constraints
func (s *EnhancedSecureTargetSystemConnection) applyStandardConstraints(
	operation string,
	params map[string]interface{},
) error {
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
func (s *EnhancedSecureTargetSystemConnection) applyHighConstraints(
	operation string,
	params map[string]interface{},
) error {
	if err := s.applyStandardConstraints(operation, params); err != nil {
		return err
	}

	// High constraints: very restrictive
	if operation == "execute_command" {
		if command, ok := params["command"].(string); ok {
			// Only allow a very limited set of commands
			allowedCommands := []string{"ls", "cat", "echo", "pwd"}
			commandName := strings.Split(command, " ")[0]

			allowed := false
			for _, allowedCmd := range allowedCommands {
				if commandName == allowedCmd {
					allowed = true
					break
				}
			}

			if !allowed {
				return fmt.Errorf("command not allowed in high security mode: %s", commandName)
			}
		}
	}

	return nil
}

// applyExtremeConstraints applies extreme security constraints
func (s *EnhancedSecureTargetSystemConnection) applyExtremeConstraints(
	operation string,
	params map[string]interface{},
) error {
	if err := s.applyHighConstraints(operation, params); err != nil {
		return err
	}

	// Extreme constraints: read-only operations only
	writeOps := []string{"write_file", "delete_file", "execute_command", "create_directory"}
	for _, writeOp := range writeOps {
		if operation == writeOp {
			return fmt.Errorf("write operations disabled in extreme security mode")
		}
	}

	return nil
}

// mapCapabilityToResourceAction maps a capability to a resource and action
func mapCapabilityToResourceAction(capability string) (string, string) {
	// Map common capabilities to resource:action pairs
	capabilityMap := map[string]struct {
		Resource string
		Action   string
	}{
		"read_file":        {"filesystem", "read"},
		"write_file":       {"filesystem", "write"},
		"delete_file":      {"filesystem", "delete"},
		"list_directory":   {"filesystem", "list"},
		"create_directory": {"filesystem", "write"},
		"execute_command":  {"system", "execute"},
		"get_system_info":  {"system", "read"},
		"query_database":   {"database", "read"},
		"execute_query":    {"database", "write"},
		"navigate":         {"browser", "read"},
		"click":            {"browser", "write"},
		"type":             {"browser", "write"},
		"screenshot":       {"browser", "read"},
	}

	if mapping, ok := capabilityMap[capability]; ok {
		return mapping.Resource, mapping.Action
	}

	// Default mapping for unknown capabilities
	return "unknown", "unknown"
}

// mapOperationToResourceAction maps an operation to a resource and action
func mapOperationToResourceAction(operation string, targetType TargetSystemType) (string, string) {
	// First try to map based on operation name
	resource, action := mapCapabilityToResourceAction(operation)
	if resource != "unknown" {
		return resource, action
	}

	// If not found, map based on target type
	switch targetType {
	case TargetTypeBrowser:
		return "browser", "read"
	case TargetTypeFilesystem:
		return "filesystem", "read"
	case TargetTypeDatabase:
		return "database", "read"
	case TargetTypeSystem:
		return "system", "read"
	default:
		return "unknown", "unknown"
	}
}

// getResourcePathFromParams extracts the resource path from operation parameters
func getResourcePathFromParams(operation string, params map[string]interface{}) string {
	// Extract path based on operation type
	if operation == "read_file" || operation == "write_file" || operation == "delete_file" {
		if path, ok := params["path"].(string); ok {
			return path
		}
	}

	if operation == "execute_command" {
		if command, ok := params["command"].(string); ok {
			return command
		}
	}

	if operation == "query_database" || operation == "execute_query" {
		if query, ok := params["query"].(string); ok {
			return query
		}
		if table, ok := params["table"].(string); ok {
			return table
		}
	}

	// Default to empty string if no path found
	return ""
}

// calculateRiskScore calculates a risk score based on security factors
func calculateRiskScore(
	securityLevel EnhancedSecurityLevel,
	authMethod string,
	mfaVerified bool,
) float64 {
	// Base risk score
	riskScore := 0.5

	// Adjust based on security level
	switch securityLevel {
	case EnhancedSecurityLevelBasic:
		riskScore += 0.2
	case EnhancedSecurityLevelStandard:
		riskScore += 0.0
	case EnhancedSecurityLevelHigh:
		riskScore -= 0.1
	case EnhancedSecurityLevelExtreme:
		riskScore -= 0.2
	}

	// Adjust based on authentication method
	switch authMethod {
	case "password":
		riskScore += 0.1
	case "oauth":
		riskScore -= 0.1
	case "certificate":
		riskScore -= 0.2
	}

	// Adjust based on MFA
	if mfaVerified {
		riskScore -= 0.3
	} else {
		riskScore += 0.2
	}

	// Ensure risk score is between 0 and 1
	if riskScore < 0 {
		riskScore = 0
	}
	if riskScore > 1 {
		riskScore = 1
	}

	return riskScore
}

// generateEnhancedSessionID generates a secure session ID
func generateEnhancedSessionID() (string, error) {
	// Generate a UUID
	id := uuid.New().String()

	// Add some randomness
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	// Create a hash of the UUID and random bytes
	hash := sha256.New()
	hash.Write([]byte(id))
	hash.Write(bytes)

	return hex.EncodeToString(hash.Sum(nil)), nil
}
