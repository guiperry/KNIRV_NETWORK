package objects

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"golang.org/x/crypto/argon2"
)

// User represents a system user with authentication and authorization data
type User struct {
	ID                string    `json:"id" buntdb:"id"`
	Username          string    `json:"username" buntdb:"username"`
	Email             string    `json:"email" buntdb:"email"`
	PasswordHash      string    `json:"password_hash" buntdb:"password_hash"`
	Salt              string    `json:"salt" buntdb:"salt"`
	Role              string    `json:"role" buntdb:"role"`
	Status            string    `json:"status" buntdb:"status"` // active, inactive, suspended, pending_verification
	EmailVerified     bool      `json:"email_verified" buntdb:"email_verified"`
	EmailVerificationToken string `json:"email_verification_token" buntdb:"email_verification_token"`
	PasswordResetToken string   `json:"password_reset_token" buntdb:"password_reset_token"`
	PasswordResetExpires *time.Time `json:"password_reset_expires" buntdb:"password_reset_expires"`
	CreatedAt         time.Time `json:"created_at" buntdb:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" buntdb:"updated_at"`
	LastLoginAt       *time.Time `json:"last_login_at,omitempty" buntdb:"last_login_at"`
	LoginAttempts     int       `json:"login_attempts" buntdb:"login_attempts"`
	LockedUntil       *time.Time `json:"locked_until" buntdb:"locked_until"`

	// Profile information
	FirstName         string `json:"first_name" buntdb:"first_name"`
	LastName          string `json:"last_name" buntdb:"last_name"`
	Company           string `json:"company,omitempty" buntdb:"company"`
	Phone             string `json:"phone,omitempty" buntdb:"phone"`

	// Security settings
	TwoFactorEnabled  bool   `json:"two_factor_enabled" buntdb:"two_factor_enabled"`
	TwoFactorSecret   string `json:"two_factor_secret" buntdb:"two_factor_secret"`

	// Preferences
	Timezone          string `json:"timezone" buntdb:"timezone"`
	Language          string `json:"language" buntdb:"language"`
}

// UserSession represents an active user session
type UserSession struct {
	ID           string    `json:"id" buntdb:"id"`
	UserID       string    `json:"user_id" buntdb:"user_id"`
	Token        string    `json:"token" buntdb:"token"`
	IPAddress    string    `json:"ip_address" buntdb:"ip_address"`
	UserAgent    string    `json:"user_agent" buntdb:"user_agent"`
	CreatedAt    time.Time `json:"created_at" buntdb:"created_at"`
	ExpiresAt    time.Time `json:"expires_at" buntdb:"expires_at"`
	LastActivity time.Time `json:"last_activity" buntdb:"last_activity"`
	IsActive     bool      `json:"is_active" buntdb:"is_active"`
}

// UserRegistration represents user registration data
type UserRegistration struct {
	Username  string `json:"username" validate:"required,min=3,max=50"`
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=8"`
	FirstName string `json:"first_name" validate:"required,min=1,max=50"`
	LastName  string `json:"last_name" validate:"required,min=1,max=50"`
	Company   string `json:"company,omitempty"`
	Phone     string `json:"phone,omitempty"`
}

// UserLogin represents login credentials
type UserLogin struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// PasswordResetRequest represents a password reset request
type PasswordResetRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// PasswordReset represents a password reset operation
type PasswordReset struct {
	Token    string `json:"token" validate:"required"`
	Password string `json:"password" validate:"required,min=8"`
}

// UserUpdate represents user profile update data
type UserUpdate struct {
	FirstName string `json:"first_name,omitempty" validate:"omitempty,min=1,max=50"`
	LastName  string `json:"last_name,omitempty" validate:"omitempty,min=1,max=50"`
	Company   string `json:"company,omitempty"`
	Phone     string `json:"phone,omitempty"`
	Timezone  string `json:"timezone,omitempty"`
	Language  string `json:"language,omitempty"`
}

// ChangePassword represents a password change operation
type ChangePassword struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=8"`
}

// Permission represents a system permission
type Permission struct {
	ID          string `json:"id" buntdb:"id"`
	Name        string `json:"name" buntdb:"name"`
	Description string `json:"description" buntdb:"description"`
	Resource    string `json:"resource" buntdb:"resource"`
	Action      string `json:"action" buntdb:"action"`
	CreatedAt   time.Time `json:"created_at" buntdb:"created_at"`
}

// Role represents a user role with associated permissions
type Role struct {
	ID          string       `json:"id" buntdb:"id"`
	Name        string       `json:"name" buntdb:"name"`
	Description string       `json:"description" buntdb:"description"`
	Permissions []string     `json:"permissions" buntdb:"permissions"` // Permission IDs
	CreatedAt   time.Time    `json:"created_at" buntdb:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at" buntdb:"updated_at"`
	IsSystem    bool         `json:"is_system" buntdb:"is_system"` // Cannot be deleted
}

// AuditLog represents an authentication/authorization audit event
type AuditLog struct {
	ID          string    `json:"id" buntdb:"id"`
	UserID      string    `json:"user_id" buntdb:"user_id"`
	Action      string    `json:"action" buntdb:"action"`
	Resource    string    `json:"resource" buntdb:"resource"`
	ResourceID  string    `json:"resource_id,omitempty" buntdb:"resource_id"`
	IPAddress   string    `json:"ip_address" buntdb:"ip_address"`
	UserAgent   string    `json:"user_agent" buntdb:"user_agent"`
	Timestamp   time.Time `json:"timestamp" buntdb:"timestamp"`
	Success     bool      `json:"success" buntdb:"success"`
	Details     string    `json:"details,omitempty" buntdb:"details"`
}

// UserRole represents different user roles in the system
type UserRole string

// UserPermissions represents the permissions for a user role
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

// Role constants
const (
	RoleAdmin  UserRole = "admin"
	RoleViewer UserRole = "viewer"
)

// GetPermissionsForRole returns the permissions for a given role
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
		return UserPermissions{} // No permissions for unknown roles
	}
}

// RateLimit represents rate limiting data for login attempts
type RateLimit struct {
	Key       string    `json:"key" buntdb:"key"` // IP or username
	Attempts  int       `json:"attempts" buntdb:"attempts"`
	LastAttempt time.Time `json:"last_attempt" buntdb:"last_attempt"`
	LockedUntil *time.Time `json:"locked_until,omitempty" buntdb:"locked_until"`
}

// HashPassword generates a secure password hash using Argon2id
func HashPassword(password string) (string, string, error) {
	// Generate salt
	salt := make([]byte, 32)
	if _, err := rand.Read(salt); err != nil {
		return "", "", err
	}

	// Hash password with Argon2id
	hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)

	return hex.EncodeToString(hash), hex.EncodeToString(salt), nil
}

// VerifyPassword verifies a password against its hash
func VerifyPassword(password, hash, salt string) bool {
	hashBytes, err := hex.DecodeString(hash)
	if err != nil {
		return false
	}

	saltBytes, err := hex.DecodeString(salt)
	if err != nil {
		return false
	}

	computedHash := argon2.IDKey([]byte(password), saltBytes, 1, 64*1024, 4, 32)
	return string(computedHash) == string(hashBytes)
}

// GenerateSecureToken generates a cryptographically secure random token
func GenerateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// IsValid validates user data
func (u *User) IsValid() bool {
	return u.Username != "" && u.Email != "" && u.Role != ""
}

// IsAccountActive checks if the user account is active
func (u *User) IsAccountActive() bool {
	return u.Status == "active" && u.EmailVerified
}

// IsLocked checks if the user account is currently locked
func (u *User) IsLocked() bool {
	if u.LockedUntil == nil {
		return false
	}
	return time.Now().Before(*u.LockedUntil)
}

// CanLogin checks if the user can attempt to login
func (u *User) CanLogin() bool {
	return u.IsAccountActive() && !u.IsLocked()
}

// HasPermission checks if the user has a specific permission
func (u *User) HasPermission(permission string, rolePermissions map[string][]string) bool {
	permissions, exists := rolePermissions[u.Role]
	if !exists {
		return false
	}

	for _, p := range permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// GetFullName returns the user's full name
func (u *User) GetFullName() string {
	if u.FirstName != "" && u.LastName != "" {
		return u.FirstName + " " + u.LastName
	}
	return u.Username
}

// IsExpired checks if the session has expired
func (s *UserSession) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// IsSessionActive checks if the session is active
func (s *UserSession) IsSessionActive() bool {
	return s.IsActive && !s.IsExpired()
}

// UpdateActivity updates the last activity timestamp
func (s *UserSession) UpdateActivity() {
	s.LastActivity = time.Now()
}

// IsLocked checks if the rate limit is currently locked
func (r *RateLimit) IsLocked() bool {
	if r.LockedUntil == nil {
		return false
	}
	return time.Now().Before(*r.LockedUntil)
}

// CanAttempt checks if another attempt is allowed
func (r *RateLimit) CanAttempt() bool {
	return !r.IsLocked()
}

// RecordAttempt records a login attempt
func (r *RateLimit) RecordAttempt(maxAttempts int, lockDuration time.Duration) {
	r.Attempts++
	r.LastAttempt = time.Now()

	if r.Attempts >= maxAttempts {
		lockUntil := time.Now().Add(lockDuration)
		r.LockedUntil = &lockUntil
	}
}

// Reset resets the rate limit counters
func (r *RateLimit) Reset() {
	r.Attempts = 0
	r.LockedUntil = nil
}
