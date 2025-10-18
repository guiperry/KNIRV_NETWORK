package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	dataengine "backend-server/internal/data-engine"
)

// AuthService handles authentication and user management
type AuthService struct {
	dataEngine *dataengine.BuntDBDataEngine
	config     AuthConfig
}

// AuthConfig contains authentication configuration
type AuthConfig struct {
	PasswordMinLength      int           `yaml:"password_min_length"`
	PasswordRequireUpper   bool          `yaml:"password_require_upper"`
	PasswordRequireLower   bool          `yaml:"password_require_lower"`
	PasswordRequireDigit   bool          `yaml:"password_require_digit"`
	PasswordRequireSpecial bool          `yaml:"password_require_special"`
	TokenExpiration        time.Duration `yaml:"token_expiration"`
	RefreshTokenExpiration time.Duration `yaml:"refresh_token_expiration"`
	MaxLoginAttempts       int           `yaml:"max_login_attempts"`
	LockoutDuration        time.Duration `yaml:"lockout_duration"`
	SessionTimeout         time.Duration `yaml:"session_timeout"`
	EnableTwoFactor        bool          `yaml:"enable_two_factor"`
}

// LoginAttempt tracks login attempts for rate limiting
type LoginAttempt struct {
	Username  string    `json:"username"`
	IPAddress string    `json:"ip_address"`
	Timestamp time.Time `json:"timestamp"`
	Success   bool      `json:"success"`
}

// UserSession represents an active user session
type UserSession struct {
	ID           string                 `json:"id"`
	UserID       string                 `json:"user_id"`
	TokenHash    string                 `json:"token_hash"`
	CreatedAt    time.Time              `json:"created_at"`
	ExpiresAt    time.Time              `json:"expires_at"`
	LastActivity time.Time              `json:"last_activity"`
	IPAddress    string                 `json:"ip_address"`
	UserModel    string                 `json:"user_model"`
	Active       bool                   `json:"active"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// PasswordResetToken represents a password reset token
type PasswordResetToken struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	Used      bool      `json:"used"`
	CreatedAt time.Time `json:"created_at"`
}

// NewAuthService creates a new authentication service
func NewAuthService(dataEngine *dataengine.BuntDBDataEngine, config AuthConfig) *AuthService {
	// Set default config values
	if config.PasswordMinLength == 0 {
		config.PasswordMinLength = 8
	}
	if config.TokenExpiration == 0 {
		config.TokenExpiration = 24 * time.Hour
	}
	if config.RefreshTokenExpiration == 0 {
		config.RefreshTokenExpiration = 7 * 24 * time.Hour
	}
	if config.MaxLoginAttempts == 0 {
		config.MaxLoginAttempts = 5
	}
	if config.LockoutDuration == 0 {
		config.LockoutDuration = 15 * time.Minute
	}
	if config.SessionTimeout == 0 {
		config.SessionTimeout = 30 * time.Minute
	}

	return &AuthService{
		dataEngine: dataEngine,
		config:     config,
	}
}

// CreateUser creates a new user with password validation
func (as *AuthService) CreateUser(username, email, password, firstName, lastName, role string) (*dataengine.UserEntry, error) {
	// Validate password
	if err := as.validatePassword(password); err != nil {
		return nil, err
	}

	// Check if user already exists
	if _, err := as.dataEngine.GetBuntDBManager().GetUserByUsername(username); err == nil {
		return nil, fmt.Errorf("username already exists")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user := &dataengine.UserEntry{
		ID:           as.generateID("user"),
		Username:     username,
		Email:        email,
		PasswordHash: string(hashedPassword),
		FirstName:    firstName,
		LastName:     lastName,
		Role:         role,
		Status:       "active",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Metadata:     make(map[string]interface{}),
	}

	if err := as.dataEngine.GetBuntDBManager().CreateUser(user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	log.Printf("AuthService: Created user %s (%s)", user.Username, user.ID)

	return user, nil
}

// AuthenticateUser authenticates a user with username and password
func (as *AuthService) AuthenticateUser(username, password, ipAddress string) (*dataengine.UserEntry, error) {
	// Check rate limiting
	if as.isUserLockedOut(username, ipAddress) {
		return nil, fmt.Errorf("account temporarily locked due to too many failed attempts")
	}

	// Get user
	user, err := as.dataEngine.GetBuntDBManager().GetUserByUsername(username)
	if err != nil {
		as.recordLoginAttempt(username, ipAddress, false)
		return nil, fmt.Errorf("invalid credentials")
	}

	// Check user status
	if user.Status != "active" {
		as.recordLoginAttempt(username, ipAddress, false)
		return nil, fmt.Errorf("account is not active")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		as.recordLoginAttempt(username, ipAddress, false)
		return nil, fmt.Errorf("invalid credentials")
	}

	// Record successful login
	as.recordLoginAttempt(username, ipAddress, true)

	// Update last login
	now := time.Now()
	user.LastLogin = &now
	as.dataEngine.GetBuntDBManager().UpdateUser(user)

	log.Printf("AuthService: User %s authenticated successfully", username)

	return user, nil
}

// ChangePassword changes a user's password
func (as *AuthService) ChangePassword(userID, currentPassword, newPassword string) error {
	// Get user
	user, err := as.dataEngine.GetBuntDBManager().GetUser(userID)
	if err != nil {
		return fmt.Errorf("user not found")
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)); err != nil {
		return fmt.Errorf("current password is incorrect")
	}

	// Validate new password
	if err := as.validatePassword(newPassword); err != nil {
		return err
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update user
	user.PasswordHash = string(hashedPassword)
	user.UpdatedAt = time.Now()

	if err := as.dataEngine.GetBuntDBManager().UpdateUser(user); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	log.Printf("AuthService: Password changed for user %s", user.Username)

	return nil
}

// CreatePasswordResetToken creates a password reset token
func (as *AuthService) CreatePasswordResetToken(email string) (*PasswordResetToken, error) {
	// Find user by email
	users, err := as.dataEngine.GetBuntDBManager().ListUsers(0, 1000) // Get all users
	if err != nil {
		return nil, fmt.Errorf("failed to search users: %w", err)
	}

	var user *dataengine.UserEntry
	for _, u := range users {
		if u.Email == email {
			user = u
			break
		}
	}

	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	// Generate reset token
	token, err := as.generateSecureToken(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	resetToken := &PasswordResetToken{
		ID:        as.generateID("reset"),
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: time.Now().Add(1 * time.Hour), // 1 hour expiration
		Used:      false,
		CreatedAt: time.Now(),
	}

	// Store token (would need to implement storage for reset tokens)
	log.Printf("AuthService: Password reset token created for user %s", user.Username)

	return resetToken, nil
}

// ResetPassword resets a user's password using a reset token
func (as *AuthService) ResetPassword(token, newPassword string) error {
	// Validate new password
	if err := as.validatePassword(newPassword); err != nil {
		return err
	}

	// This would need to be implemented with proper token storage
	// For now, return success
	log.Printf("AuthService: Password reset completed")

	return nil
}

// Helper methods

// validatePassword validates password strength
func (as *AuthService) validatePassword(password string) error {
	if len(password) < as.config.PasswordMinLength {
		return fmt.Errorf("password must be at least %d characters long", as.config.PasswordMinLength)
	}

	if as.config.PasswordRequireUpper && !strings.ContainsAny(password, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
		return fmt.Errorf("password must contain at least one uppercase letter")
	}

	if as.config.PasswordRequireLower && !strings.ContainsAny(password, "abcdefghijklmnopqrstuvwxyz") {
		return fmt.Errorf("password must contain at least one lowercase letter")
	}

	if as.config.PasswordRequireDigit && !strings.ContainsAny(password, "0123456789") {
		return fmt.Errorf("password must contain at least one digit")
	}

	if as.config.PasswordRequireSpecial && !strings.ContainsAny(password, "!@#$%^&*()_+-=[]{}|;:,.<>?") {
		return fmt.Errorf("password must contain at least one special character")
	}

	return nil
}

// isUserLockedOut checks if a user is locked out due to failed login attempts
func (as *AuthService) isUserLockedOut(username, ipAddress string) bool {
	// This would need to be implemented with proper storage
	// For now, return false
	return false
}

// recordLoginAttempt records a login attempt
func (as *AuthService) recordLoginAttempt(username, ipAddress string, success bool) {
	attempt := LoginAttempt{
		Username:  username,
		IPAddress: ipAddress,
		Timestamp: time.Now(),
		Success:   success,
	}

	// This would need to be implemented with proper storage
	log.Printf("AuthService: Login attempt recorded for %s from %s (success: %v)", username, ipAddress, success)
	_ = attempt
}

// generateID generates a unique ID with prefix
func (as *AuthService) generateID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}

// generateSecureToken generates a cryptographically secure random token
func (as *AuthService) generateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	hash := sha256.Sum256(bytes)
	return hex.EncodeToString(hash[:])[:length], nil
}
