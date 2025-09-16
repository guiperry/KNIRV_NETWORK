package models

import (
	"testing"
	"time"
)

func TestUserRole_Constants(t *testing.T) {
	tests := []struct {
		name     string
		role     UserRole
		expected string
	}{
		{"Admin role", RoleAdmin, "admin"},
		{"Operator role", RoleOperator, "operator"},
		{"Validator role", RoleValidator, "validator"},
		{"Viewer role", RoleViewer, "viewer"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.role) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, string(tt.role))
			}
		})
	}
}

func TestGetPermissionsForRole_Admin(t *testing.T) {
	permissions := GetPermissionsForRole(RoleAdmin)

	// Admin should have all permissions
	if !permissions.CanManageNodes {
		t.Error("Admin should be able to manage nodes")
	}
	if !permissions.CanCreateTasks {
		t.Error("Admin should be able to create tasks")
	}
	if !permissions.CanViewSystemHealth {
		t.Error("Admin should be able to view system health")
	}
	if !permissions.CanManageUsers {
		t.Error("Admin should be able to manage users")
	}
	if !permissions.CanAccessTEEData {
		t.Error("Admin should be able to access TEE data")
	}
	if !permissions.CanGenerateReports {
		t.Error("Admin should be able to generate reports")
	}
	if !permissions.CanShareReports {
		t.Error("Admin should be able to share reports")
	}
	if !permissions.CanScheduleReports {
		t.Error("Admin should be able to schedule reports")
	}
	if !permissions.CanViewAuditLogs {
		t.Error("Admin should be able to view audit logs")
	}
	if !permissions.CanManageAlerts {
		t.Error("Admin should be able to manage alerts")
	}
}

func TestGetPermissionsForRole_Operator(t *testing.T) {
	permissions := GetPermissionsForRole(RoleOperator)

	// Test operator permissions
	if !permissions.CanManageNodes {
		t.Error("Operator should be able to manage nodes")
	}
	if !permissions.CanCreateTasks {
		t.Error("Operator should be able to create tasks")
	}
	if !permissions.CanViewSystemHealth {
		t.Error("Operator should be able to view system health")
	}
	if permissions.CanManageUsers {
		t.Error("Operator should NOT be able to manage users")
	}
	if !permissions.CanAccessTEEData {
		t.Error("Operator should be able to access TEE data")
	}
	if !permissions.CanGenerateReports {
		t.Error("Operator should be able to generate reports")
	}
	if !permissions.CanShareReports {
		t.Error("Operator should be able to share reports")
	}
	if permissions.CanScheduleReports {
		t.Error("Operator should NOT be able to schedule reports")
	}
	if permissions.CanViewAuditLogs {
		t.Error("Operator should NOT be able to view audit logs")
	}
	if !permissions.CanManageAlerts {
		t.Error("Operator should be able to manage alerts")
	}
}

func TestGetPermissionsForRole_Validator(t *testing.T) {
	permissions := GetPermissionsForRole(RoleValidator)

	// Test validator permissions
	if permissions.CanManageNodes {
		t.Error("Validator should NOT be able to manage nodes")
	}
	if !permissions.CanCreateTasks {
		t.Error("Validator should be able to create tasks")
	}
	if !permissions.CanViewSystemHealth {
		t.Error("Validator should be able to view system health")
	}
	if permissions.CanManageUsers {
		t.Error("Validator should NOT be able to manage users")
	}
	if permissions.CanAccessTEEData {
		t.Error("Validator should NOT be able to access TEE data")
	}
	if !permissions.CanGenerateReports {
		t.Error("Validator should be able to generate reports")
	}
	if !permissions.CanShareReports {
		t.Error("Validator should be able to share reports")
	}
	if permissions.CanScheduleReports {
		t.Error("Validator should NOT be able to schedule reports")
	}
	if permissions.CanViewAuditLogs {
		t.Error("Validator should NOT be able to view audit logs")
	}
	if permissions.CanManageAlerts {
		t.Error("Validator should NOT be able to manage alerts")
	}
}

func TestGetPermissionsForRole_Viewer(t *testing.T) {
	permissions := GetPermissionsForRole(RoleViewer)

	// Test viewer permissions (most restrictive)
	if permissions.CanManageNodes {
		t.Error("Viewer should NOT be able to manage nodes")
	}
	if permissions.CanCreateTasks {
		t.Error("Viewer should NOT be able to create tasks")
	}
	if !permissions.CanViewSystemHealth {
		t.Error("Viewer should be able to view system health")
	}
	if permissions.CanManageUsers {
		t.Error("Viewer should NOT be able to manage users")
	}
	if permissions.CanAccessTEEData {
		t.Error("Viewer should NOT be able to access TEE data")
	}
	if !permissions.CanGenerateReports {
		t.Error("Viewer should be able to generate reports")
	}
	if permissions.CanShareReports {
		t.Error("Viewer should NOT be able to share reports")
	}
	if permissions.CanScheduleReports {
		t.Error("Viewer should NOT be able to schedule reports")
	}
	if permissions.CanViewAuditLogs {
		t.Error("Viewer should NOT be able to view audit logs")
	}
	if permissions.CanManageAlerts {
		t.Error("Viewer should NOT be able to manage alerts")
	}
}

func TestGetPermissionsForRole_InvalidRole(t *testing.T) {
	permissions := GetPermissionsForRole(UserRole("invalid"))

	// Invalid role should have no permissions
	if permissions.CanManageNodes ||
		permissions.CanCreateTasks ||
		permissions.CanViewSystemHealth ||
		permissions.CanManageUsers ||
		permissions.CanAccessTEEData ||
		permissions.CanGenerateReports ||
		permissions.CanShareReports ||
		permissions.CanScheduleReports ||
		permissions.CanViewAuditLogs ||
		permissions.CanManageAlerts {
		t.Error("Invalid role should have no permissions")
	}
}

func TestUserProfile_StructFields(t *testing.T) {
	now := time.Now()
	profile := UserProfile{
		ID:          "user-123",
		Email:       "test@example.com",
		Username:    "testuser",
		Role:        "admin",
		Permissions: UserPermissions{CanManageNodes: true},
		CreatedAt:   now,
		UpdatedAt:   now,
		LastLogin:   now,
		IsActive:    true,
		FirstName:   "Test",
		LastName:    "User",
		Avatar:      "avatar.jpg",
		Timezone:    "UTC",
		Language:    "en",
	}

	if profile.ID != "user-123" {
		t.Errorf("Expected ID 'user-123', got '%s'", profile.ID)
	}
	if profile.Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got '%s'", profile.Email)
	}
	if profile.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", profile.Username)
	}
	if profile.Role != "admin" {
		t.Errorf("Expected role 'admin', got '%s'", profile.Role)
	}
	if !profile.Permissions.CanManageNodes {
		t.Error("Expected CanManageNodes to be true")
	}
	if !profile.IsActive {
		t.Error("Expected IsActive to be true")
	}
	if profile.FirstName != "Test" {
		t.Errorf("Expected FirstName 'Test', got '%s'", profile.FirstName)
	}
	if profile.LastName != "User" {
		t.Errorf("Expected LastName 'User', got '%s'", profile.LastName)
	}
}

func TestUserSession_StructFields(t *testing.T) {
	now := time.Now()
	session := UserSession{
		Token:     "token-123",
		UserID:    "user-123",
		ExpiresAt: now.Add(time.Hour),
		CreatedAt: now,
		IPAddress: "192.168.1.1",
		UserAgent: "Mozilla/5.0",
		IsActive:  true,
	}

	if session.Token != "token-123" {
		t.Errorf("Expected token 'token-123', got '%s'", session.Token)
	}
	if session.UserID != "user-123" {
		t.Errorf("Expected UserID 'user-123', got '%s'", session.UserID)
	}
	if session.IPAddress != "192.168.1.1" {
		t.Errorf("Expected IPAddress '192.168.1.1', got '%s'", session.IPAddress)
	}
	if session.UserAgent != "Mozilla/5.0" {
		t.Errorf("Expected UserAgent 'Mozilla/5.0', got '%s'", session.UserAgent)
	}
	if !session.IsActive {
		t.Error("Expected IsActive to be true")
	}
}

func TestAuditLog_StructFields(t *testing.T) {
	now := time.Now()
	details := map[string]interface{}{
		"action": "create",
		"count":  1,
	}

	auditLog := AuditLog{
		ID:        "audit-123",
		UserID:    "user-123",
		Action:    "CREATE_NODE",
		Resource:  "dve_node",
		Details:   details,
		IPAddress: "192.168.1.1",
		UserAgent: "Mozilla/5.0",
		Success:   true,
		Error:     "",
		Timestamp: now,
	}

	if auditLog.ID != "audit-123" {
		t.Errorf("Expected ID 'audit-123', got '%s'", auditLog.ID)
	}
	if auditLog.Action != "CREATE_NODE" {
		t.Errorf("Expected Action 'CREATE_NODE', got '%s'", auditLog.Action)
	}
	if auditLog.Resource != "dve_node" {
		t.Errorf("Expected Resource 'dve_node', got '%s'", auditLog.Resource)
	}
	if !auditLog.Success {
		t.Error("Expected Success to be true")
	}
	if auditLog.Details["action"] != "create" {
		t.Errorf("Expected Details action 'create', got '%v'", auditLog.Details["action"])
	}
}

func TestOAuthProvider_StructFields(t *testing.T) {
	provider := OAuthProvider{
		ID:           "google",
		Name:         "Google",
		ClientID:     "client-123",
		ClientSecret: "secret-123",
		AuthURL:      "https://accounts.google.com/oauth/authorize",
		TokenURL:     "https://oauth2.googleapis.com/token",
		UserInfoURL:  "https://www.googleapis.com/oauth2/v2/userinfo",
		Scopes:       []string{"openid", "email", "profile"},
		IsEnabled:    true,
	}

	if provider.ID != "google" {
		t.Errorf("Expected ID 'google', got '%s'", provider.ID)
	}
	if provider.Name != "Google" {
		t.Errorf("Expected Name 'Google', got '%s'", provider.Name)
	}
	if len(provider.Scopes) != 3 {
		t.Errorf("Expected 3 scopes, got %d", len(provider.Scopes))
	}
	if !provider.IsEnabled {
		t.Error("Expected IsEnabled to be true")
	}
}

func TestUserPreferences_StructFields(t *testing.T) {
	now := time.Now()
	prefs := UserPreferences{
		UserID:              "user-123",
		Theme:               "dark",
		NotificationsEmail:  true,
		NotificationsPush:   false,
		DashboardLayout:     "grid",
		DefaultReportFormat: "pdf",
		AutoRefreshInterval: 30,
		UpdatedAt:           now,
	}

	if prefs.UserID != "user-123" {
		t.Errorf("Expected UserID 'user-123', got '%s'", prefs.UserID)
	}
	if prefs.Theme != "dark" {
		t.Errorf("Expected Theme 'dark', got '%s'", prefs.Theme)
	}
	if !prefs.NotificationsEmail {
		t.Error("Expected NotificationsEmail to be true")
	}
	if prefs.NotificationsPush {
		t.Error("Expected NotificationsPush to be false")
	}
	if prefs.AutoRefreshInterval != 30 {
		t.Errorf("Expected AutoRefreshInterval 30, got %d", prefs.AutoRefreshInterval)
	}
}

func TestAPIKey_StructFields(t *testing.T) {
	now := time.Now()
	expiresAt := now.Add(time.Hour * 24 * 30) // 30 days
	lastUsed := now.Add(-time.Hour)           // 1 hour ago

	apiKey := APIKey{
		ID:          "key-123",
		UserID:      "user-123",
		Name:        "Production API Key",
		Key:         "ak_123456789",
		Permissions: []string{"read:nodes", "write:tasks"},
		ExpiresAt:   &expiresAt,
		LastUsed:    &lastUsed,
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if apiKey.ID != "key-123" {
		t.Errorf("Expected ID 'key-123', got '%s'", apiKey.ID)
	}
	if apiKey.Name != "Production API Key" {
		t.Errorf("Expected Name 'Production API Key', got '%s'", apiKey.Name)
	}
	if len(apiKey.Permissions) != 2 {
		t.Errorf("Expected 2 permissions, got %d", len(apiKey.Permissions))
	}
	if apiKey.ExpiresAt == nil {
		t.Error("Expected ExpiresAt to be set")
	}
	if apiKey.LastUsed == nil {
		t.Error("Expected LastUsed to be set")
	}
	if !apiKey.IsActive {
		t.Error("Expected IsActive to be true")
	}
}
