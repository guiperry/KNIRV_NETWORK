package models

import (
	"time"
)

// UserProfile represents a user in the system
type UserProfile struct {
	ID          string          `json:"id"`
	Email       string          `json:"email"`
	Username    string          `json:"username"`
	Role        string          `json:"role"` // "admin", "operator", "validator", "viewer"
	Permissions UserPermissions `json:"permissions"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	LastLogin   time.Time       `json:"last_login"`
	IsActive    bool            `json:"is_active"`

	// Profile information
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	Avatar    string `json:"avatar,omitempty"`
	Timezone  string `json:"timezone,omitempty"`
	Language  string `json:"language,omitempty"`
}

// UserPermissions represents user permissions in the system
type UserPermissions struct {
	CanManageNodes      bool `json:"can_manage_nodes"`
	CanCreateTasks      bool `json:"can_create_tasks"`
	CanViewSystemHealth bool `json:"can_view_system_health"`
	CanManageUsers      bool `json:"can_manage_users"`
	CanAccessTEEData    bool `json:"can_access_tee_data"`
	CanGenerateReports  bool `json:"can_generate_reports"`
	CanShareReports     bool `json:"can_share_reports"`
	CanScheduleReports  bool `json:"can_schedule_reports"`
	CanViewAuditLogs    bool `json:"can_view_audit_logs"`
	CanManageAlerts     bool `json:"can_manage_alerts"`
}

// UserSession represents a user session
type UserSession struct {
	Token     string    `json:"token"`
	UserID    string    `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	IPAddress string    `json:"ip_address"`
	UserModel string    `json:"user_model"`
	IsActive  bool      `json:"is_active"`
}

// UserRole represents available user roles
type UserRole string

const (
	RoleAdmin     UserRole = "admin"     // Full system access
	RoleOperator  UserRole = "operator"  // Node management and operations
	RoleValidator UserRole = "validator" // Task creation and validation
	RoleViewer    UserRole = "viewer"    // Read-only access
)

// GetPermissionsForRole returns default permissions for a given role
func GetPermissionsForRole(role UserRole) UserPermissions {
	switch role {
	case RoleAdmin:
		return UserPermissions{
			CanManageNodes:      true,
			CanCreateTasks:      true,
			CanViewSystemHealth: true,
			CanManageUsers:      true,
			CanAccessTEEData:    true,
			CanGenerateReports:  true,
			CanShareReports:     true,
			CanScheduleReports:  true,
			CanViewAuditLogs:    true,
			CanManageAlerts:     true,
		}
	case RoleOperator:
		return UserPermissions{
			CanManageNodes:      true,
			CanCreateTasks:      true,
			CanViewSystemHealth: true,
			CanManageUsers:      false,
			CanAccessTEEData:    true,
			CanGenerateReports:  true,
			CanShareReports:     true,
			CanScheduleReports:  false,
			CanViewAuditLogs:    false,
			CanManageAlerts:     true,
		}
	case RoleValidator:
		return UserPermissions{
			CanManageNodes:      false,
			CanCreateTasks:      true,
			CanViewSystemHealth: true,
			CanManageUsers:      false,
			CanAccessTEEData:    false,
			CanGenerateReports:  true,
			CanShareReports:     true,
			CanScheduleReports:  false,
			CanViewAuditLogs:    false,
			CanManageAlerts:     false,
		}
	case RoleViewer:
		return UserPermissions{
			CanManageNodes:      false,
			CanCreateTasks:      false,
			CanViewSystemHealth: true,
			CanManageUsers:      false,
			CanAccessTEEData:    false,
			CanGenerateReports:  true,
			CanShareReports:     false,
			CanScheduleReports:  false,
			CanViewAuditLogs:    false,
			CanManageAlerts:     false,
		}
	default:
		return UserPermissions{} // No permissions
	}
}

// AuditLog represents an audit log entry
type AuditLog struct {
	ID        string                 `json:"id"`
	UserID    string                 `json:"user_id"`
	Action    string                 `json:"action"`
	Resource  string                 `json:"resource"`
	Details   map[string]interface{} `json:"details"`
	IPAddress string                 `json:"ip_address"`
	UserModel string                 `json:"user_model"`
	Success   bool                   `json:"success"`
	Error     string                 `json:"error,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// OAuthProvider represents an OAuth provider configuration
type OAuthProvider struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	AuthURL      string   `json:"auth_url"`
	TokenURL     string   `json:"token_url"`
	UserInfoURL  string   `json:"user_info_url"`
	Scopes       []string `json:"scopes"`
	IsEnabled    bool     `json:"is_enabled"`
}

// UserPreferences represents user preferences and settings
type UserPreferences struct {
	UserID              string    `json:"user_id"`
	Theme               string    `json:"theme"` // "light", "dark", "auto"
	NotificationsEmail  bool      `json:"notifications_email"`
	NotificationsPush   bool      `json:"notifications_push"`
	DashboardLayout     string    `json:"dashboard_layout"`
	DefaultReportFormat string    `json:"default_report_format"`
	AutoRefreshInterval int       `json:"auto_refresh_interval"` // seconds
	UpdatedAt           time.Time `json:"updated_at"`
}

// APIKey represents an API key for programmatic access
type APIKey struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	Name        string     `json:"name"`
	Key         string     `json:"key"`
	Permissions []string   `json:"permissions"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	LastUsed    *time.Time `json:"last_used,omitempty"`
	IsActive    bool       `json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}
