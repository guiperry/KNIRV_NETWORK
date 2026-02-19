package auth

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"backend_server/internal/database"
	"backend_server/internal/objects"

	"github.com/tidwall/buntdb"
)

// AuthConfig contains authentication configuration
type AuthConfig struct {
	PasswordMinLength      int           `json:"password_min_length" yaml:"password_min_length"`
	PasswordRequireUpper   bool          `json:"password_require_upper" yaml:"password_require_upper"`
	PasswordRequireLower   bool          `json:"password_require_lower" yaml:"password_require_lower"`
	PasswordRequireDigit   bool          `json:"password_require_digit" yaml:"password_require_digit"`
	PasswordRequireSpecial bool          `json:"password_require_special" yaml:"password_require_special"`
	MaxLoginAttempts       int           `json:"max_login_attempts" yaml:"max_login_attempts"`
	LockoutDuration        time.Duration `json:"lockout_duration" yaml:"lockout_duration"`
	SessionTimeout         time.Duration `json:"session_timeout" yaml:"session_timeout"`
	TokenExpiration        time.Duration `json:"token_expiration" yaml:"token_expiration"`
	RefreshTokenExpiration time.Duration `json:"refresh_token_expiration" yaml:"refresh_token_expiration"`
	EnableTwoFactor        bool          `json:"enable_two_factor" yaml:"enable_two_factor"`
}

// DefaultAuthConfig returns default authentication configuration
func DefaultAuthConfig() AuthConfig {
	return AuthConfig{
		PasswordMinLength:      8,
		PasswordRequireUpper:   true,
		PasswordRequireLower:   true,
		PasswordRequireDigit:   true,
		PasswordRequireSpecial: false,
		MaxLoginAttempts:       5,
		LockoutDuration:        15 * time.Minute,
		SessionTimeout:         30 * time.Minute,
		TokenExpiration:        24 * time.Hour,
		RefreshTokenExpiration: 7 * 24 * time.Hour,
		EnableTwoFactor:        false,
	}
}

// UserService handles user-related database operations
type UserService struct {
	db     *database.BuntDBManager
	config AuthConfig
}

// NewUserService creates a new user service
func NewUserService(db *database.BuntDBManager) *UserService {
	return &UserService{
		db:     db,
		config: DefaultAuthConfig(),
	}
}

// NewUserServiceWithConfig creates a new user service with custom configuration
func NewUserServiceWithConfig(db *database.BuntDBManager, config AuthConfig) *UserService {
	return &UserService{
		db:     db,
		config: config,
	}
}

// CreateUser creates a new user with hashed password
func (us *UserService) CreateUser(registration *objects.UserRegistration) (*objects.User, error) {
	// Validate input
	if err := us.validateRegistration(registration); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Check if username or email already exists
	if exists, err := us.usernameExists(registration.Username); err != nil {
		return nil, fmt.Errorf("username check failed: %w", err)
	} else if exists {
		return nil, fmt.Errorf("username already exists")
	}

	if exists, err := us.emailExists(registration.Email); err != nil {
		return nil, fmt.Errorf("email check failed: %w", err)
	} else if exists {
		return nil, fmt.Errorf("email already exists")
	}

	// Hash password
	hash, salt, err := objects.HashPassword(registration.Password)
	if err != nil {
		return nil, fmt.Errorf("password hashing failed: %w", err)
	}

	// Generate verification token
	verificationToken, err := objects.GenerateSecureToken(32)
	if err != nil {
		return nil, fmt.Errorf("token generation failed: %w", err)
	}

	// Create user
	now := time.Now()
	user := &objects.User{
		ID:                     fmt.Sprintf("user_%d", now.UnixNano()),
		Username:               registration.Username,
		Email:                  strings.ToLower(registration.Email),
		PasswordHash:           hash,
		Salt:                   salt,
		Role:                   "user", // Default role
		Status:                 "pending_verification",
		EmailVerified:          false,
		EmailVerificationToken: verificationToken,
		CreatedAt:              now,
		UpdatedAt:              now,
		FirstName:              registration.FirstName,
		LastName:               registration.LastName,
		Company:                registration.Company,
		Phone:                  registration.Phone,
		Timezone:               "UTC",
		Language:               "en",
	}

	// Store user
	log.Printf("DEBUG: Storing user with hash: %s, salt: %s", user.PasswordHash, user.Salt)
	if err := us.db.StoreJSON(fmt.Sprintf("users:profiles:%s", user.ID), user); err != nil {
		return nil, fmt.Errorf("user storage failed: %w", err)
	}
	log.Printf("DEBUG: User stored successfully")

	// Log audit event
	us.logAuditEvent("", "user_registration", "user", "", "", "", "User registration initiated", true)

	log.Printf("User created: %s (%s)", user.Username, user.ID)
	return user, nil
}

// GetUserByID retrieves a user by ID
func (us *UserService) GetUserByID(userID string) (*objects.User, error) {
	var user objects.User
	err := us.db.GetJSON(fmt.Sprintf("users:profiles:%s", userID), &user)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	return &user, nil
}

// GetUserByUsername retrieves a user by username
func (us *UserService) GetUserByUsername(username string) (*objects.User, error) {
	var user objects.User
	found := false

	err := us.db.ViewTransaction(func(tx *buntdb.Tx) error {
		return tx.Ascend("users_by_username", func(key, value string) bool {
			var u objects.User
			if err := us.db.GetJSON(key, &u); err == nil && u.Username == username {
				user = u
				found = true
				return false // Stop iteration
			}
			return true // Continue
		})
	})

	if err != nil {
		return nil, fmt.Errorf("database query failed: %w", err)
	}

	if !found {
		return nil, fmt.Errorf("user not found")
	}

	return &user, nil
}

// GetUserByEmail retrieves a user by email
func (us *UserService) GetUserByEmail(email string) (*objects.User, error) {
	var user objects.User
	found := false
	emailLower := strings.ToLower(email)

	err := us.db.ViewTransaction(func(tx *buntdb.Tx) error {
		return tx.Ascend("users_by_email", func(key, value string) bool {
			var u objects.User
			if err := us.db.GetJSON(key, &u); err == nil && strings.ToLower(u.Email) == emailLower {
				user = u
				found = true
				return false // Stop iteration
			}
			return true // Continue
		})
	})

	if err != nil {
		return nil, fmt.Errorf("database query failed: %w", err)
	}

	if !found {
		return nil, fmt.Errorf("user not found")
	}

	return &user, nil
}

// UpdateUser updates user profile information
func (us *UserService) UpdateUser(userID string, updates *objects.UserUpdate) error {
	user, err := us.GetUserByID(userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Apply updates
	if updates.FirstName != "" {
		user.FirstName = updates.FirstName
	}
	if updates.LastName != "" {
		user.LastName = updates.LastName
	}
	if updates.Company != "" {
		user.Company = updates.Company
	}
	if updates.Phone != "" {
		user.Phone = updates.Phone
	}
	if updates.Timezone != "" {
		user.Timezone = updates.Timezone
	}
	if updates.Language != "" {
		user.Language = updates.Language
	}

	user.UpdatedAt = time.Now()

	// Store updated user
	if err := us.db.StoreJSON(fmt.Sprintf("users:profiles:%s", user.ID), user); err != nil {
		return fmt.Errorf("user update failed: %w", err)
	}

	return nil
}

// ChangePassword changes a user's password
func (us *UserService) ChangePassword(userID string, change *objects.ChangePassword) error {
	user, err := us.GetUserByID(userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Verify current password
	if !objects.VerifyPassword(change.CurrentPassword, user.PasswordHash, user.Salt) {
		return fmt.Errorf("current password is incorrect")
	}

	// Hash new password
	hash, salt, err := objects.HashPassword(change.NewPassword)
	if err != nil {
		return fmt.Errorf("password hashing failed: %w", err)
	}

	// Update password
	user.PasswordHash = hash
	user.Salt = salt
	user.UpdatedAt = time.Now()

	// Store updated user
	if err := us.db.StoreJSON(fmt.Sprintf("users:profiles:%s", user.ID), user); err != nil {
		return fmt.Errorf("password update failed: %w", err)
	}

	return nil
}

// VerifyEmail verifies a user's email address
func (us *UserService) VerifyEmail(token string) error {
	// Find user by verification token
	var user objects.User
	found := false

	err := us.db.ViewTransaction(func(tx *buntdb.Tx) error {
		return tx.AscendKeys("*", func(key, value string) bool {
			if strings.HasPrefix(key, "users:profiles:") {
				var u objects.User
				if err := json.Unmarshal([]byte(value), &u); err == nil && u.EmailVerificationToken == token {
					user = u
					found = true
					return false // Stop iteration
				}
			}
			return true // Continue
		})
	})

	if err != nil {
		return fmt.Errorf("database query failed: %w", err)
	}

	if !found {
		return fmt.Errorf("invalid verification token")
	}

	// Update user
	user.EmailVerified = true
	user.Status = "active"
	user.EmailVerificationToken = ""
	user.UpdatedAt = time.Now()

	if err := us.db.StoreJSON(fmt.Sprintf("users:profiles:%s", user.ID), &user); err != nil {
		return fmt.Errorf("email verification failed: %w", err)
	}

	return nil
}

// InitiatePasswordReset initiates a password reset for a user
func (us *UserService) InitiatePasswordReset(email string) error {
	user, err := us.GetUserByEmail(email)
	if err != nil {
		// Don't reveal if email exists or not for security
		return nil
	}

	// Generate reset token
	resetToken, err := objects.GenerateSecureToken(32)
	if err != nil {
		return fmt.Errorf("token generation failed: %w", err)
	}

	// Set reset token and expiry
	expiresAt := time.Now().Add(1 * time.Hour)
	user.PasswordResetToken = resetToken
	user.PasswordResetExpires = &expiresAt
	user.UpdatedAt = time.Now()

	if err := us.db.StoreJSON(fmt.Sprintf("users:profiles:%s", user.ID), user); err != nil {
		return fmt.Errorf("password reset initiation failed: %w", err)
	}

	// TODO: Send email with reset token
	log.Printf("Password reset initiated for user: %s", user.Email)

	return nil
}

// ResetPassword resets a user's password using a reset token
func (us *UserService) ResetPassword(reset *objects.PasswordReset) error {
	// Find user by reset token
	var user objects.User
	found := false

	err := us.db.ViewTransaction(func(tx *buntdb.Tx) error {
		return tx.AscendKeys("*", func(key, value string) bool {
			if strings.HasPrefix(key, "users:profiles:") {
				var u objects.User
				if err := json.Unmarshal([]byte(value), &u); err == nil {
					if u.PasswordResetToken == reset.Token && u.PasswordResetExpires != nil && time.Now().Before(*u.PasswordResetExpires) {
						user = u
						found = true
						return false // Stop iteration
					}
				}
			}
			return true // Continue
		})
	})

	if err != nil {
		return fmt.Errorf("database query failed: %w", err)
	}

	if !found {
		return fmt.Errorf("invalid or expired reset token")
	}

	// Hash new password
	hash, salt, err := objects.HashPassword(reset.Password)
	if err != nil {
		return fmt.Errorf("password hashing failed: %w", err)
	}

	// Update password and clear reset token
	user.PasswordHash = hash
	user.Salt = salt
	user.PasswordResetToken = ""
	user.PasswordResetExpires = nil
	user.UpdatedAt = time.Now()

	if err := us.db.StoreJSON(fmt.Sprintf("users:profiles:%s", user.ID), &user); err != nil {
		return fmt.Errorf("password reset failed: %w", err)
	}

	return nil
}

// RecordLoginAttempt records a login attempt and handles account locking
func (us *UserService) RecordLoginAttempt(username string, ipAddress string, success bool) error {
	user, err := us.GetUserByUsername(username)
	if err != nil {
		// User not found, still record the attempt for rate limiting
		return us.recordRateLimitAttempt(username, ipAddress, success)
	}

	if success {
		// Successful login
		now := time.Now()
		user.LastLoginAt = &now
		user.LoginAttempts = 0
		user.LockedUntil = nil
		user.UpdatedAt = time.Now()

		if err := us.db.StoreJSON(fmt.Sprintf("users:profiles:%s", user.ID), user); err != nil {
			log.Printf("Failed to update last login: %v", err)
		}
	} else {
		// Failed login
		user.LoginAttempts++
		user.UpdatedAt = time.Now()

		// Lock account after 5 failed attempts
		if user.LoginAttempts >= 5 {
			lockUntil := time.Now().Add(30 * time.Minute)
			user.LockedUntil = &lockUntil
		}

		if err := us.db.StoreJSON(fmt.Sprintf("users:profiles:%s", user.ID), user); err != nil {
			log.Printf("Failed to update login attempts: %v", err)
		}
	}

	return us.recordRateLimitAttempt(username, ipAddress, success)
}

// AuthenticateUser authenticates a user with username/password
func (us *UserService) AuthenticateUser(username, password, ipAddress string) (*objects.User, error) {
	// Check rate limiting first
	if locked, err := us.isRateLimited(username, ipAddress); err != nil {
		return nil, fmt.Errorf("rate limit check failed: %w", err)
	} else if locked {
		return nil, fmt.Errorf("too many login attempts, please try again later")
	}

	// Get user
	user, err := us.GetUserByUsername(username)
	if err != nil {
		// Record failed attempt
		us.RecordLoginAttempt(username, ipAddress, false)
		return nil, fmt.Errorf("invalid credentials")
	}

	// Check if account can login
	if !user.CanLogin() {
		us.RecordLoginAttempt(username, ipAddress, false)
		if user.IsLocked() {
			return nil, fmt.Errorf("account is temporarily locked due to too many failed login attempts")
		}
		return nil, fmt.Errorf("account is not active")
	}

	// Verify password
	if !objects.VerifyPassword(password, user.PasswordHash, user.Salt) {
		us.RecordLoginAttempt(username, ipAddress, false)
		return nil, fmt.Errorf("invalid credentials")
	}

	// Successful authentication
	us.RecordLoginAttempt(username, ipAddress, true)

	// Log audit event
	us.logAuditEvent(user.ID, "login", "user", user.ID, ipAddress, "", "User login successful", true)

	return user, nil
}

// Private helper methods

func (us *UserService) validateRegistration(reg *objects.UserRegistration) error {
	if len(reg.Username) < 3 || len(reg.Username) > 50 {
		return fmt.Errorf("username must be between 3 and 50 characters")
	}

	// Use enhanced password validation from config
	if err := us.validatePassword(reg.Password); err != nil {
		return err
	}

	if len(reg.FirstName) < 1 || len(reg.FirstName) > 50 {
		return fmt.Errorf("first name must be between 1 and 50 characters")
	}
	if len(reg.LastName) < 1 || len(reg.LastName) > 50 {
		return fmt.Errorf("last name must be between 1 and 50 characters")
	}
	// Basic email validation
	if !strings.Contains(reg.Email, "@") {
		return fmt.Errorf("invalid email format")
	}
	return nil
}

// validatePassword validates password strength based on configuration
func (us *UserService) validatePassword(password string) error {
	if len(password) < us.config.PasswordMinLength {
		return fmt.Errorf("password must be at least %d characters long", us.config.PasswordMinLength)
	}

	if us.config.PasswordRequireUpper && !strings.ContainsAny(password, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
		return fmt.Errorf("password must contain at least one uppercase letter")
	}

	if us.config.PasswordRequireLower && !strings.ContainsAny(password, "abcdefghijklmnopqrstuvwxyz") {
		return fmt.Errorf("password must contain at least one lowercase letter")
	}

	if us.config.PasswordRequireDigit && !strings.ContainsAny(password, "0123456789") {
		return fmt.Errorf("password must contain at least one digit")
	}

	if us.config.PasswordRequireSpecial && !strings.ContainsAny(password, "!@#$%^&*()_+-=[]{}|;:,.<>?") {
		return fmt.Errorf("password must contain at least one special character")
	}

	return nil
}

func (us *UserService) usernameExists(username string) (bool, error) {
	exists := false
	err := us.db.GetObjectsByPrefix("users:profiles:", func(key string, value []byte) bool {
		var user objects.User
		if err := json.Unmarshal(value, &user); err == nil && user.Username == username {
			exists = true
			return false
		}
		return true
	})

	return exists, err
}

func (us *UserService) emailExists(email string) (bool, error) {
	exists := false
	emailLower := strings.ToLower(email)
	err := us.db.GetObjectsByPrefix("users:profiles:", func(key string, value []byte) bool {
		var user objects.User
		if err := json.Unmarshal(value, &user); err == nil && strings.ToLower(user.Email) == emailLower {
			exists = true
			return false
		}
		return true
	})

	return exists, err
}

func (us *UserService) recordRateLimitAttempt(identifier, ipAddress string, success bool) error {
	// Use both username/IP for rate limiting
	keys := []string{fmt.Sprintf("ratelimit:username:%s", identifier), fmt.Sprintf("ratelimit:ip:%s", ipAddress)}

	for _, key := range keys {
		var rateLimit objects.RateLimit
		err := us.db.GetJSON(key, &rateLimit)
		if err != nil {
			// Create new rate limit entry
			rateLimit = objects.RateLimit{
				Key:         key,
				Attempts:    0,
				LastAttempt: time.Now(),
			}
		}

		if success {
			rateLimit.Reset()
		} else {
			rateLimit.RecordAttempt(10, 15*time.Minute) // 10 attempts, 15 min lock
		}

		if err := us.db.StoreJSON(key, &rateLimit); err != nil {
			log.Printf("Failed to store rate limit: %v", err)
		}
	}

	return nil
}

func (us *UserService) isRateLimited(username, ipAddress string) (bool, error) {
	keys := []string{fmt.Sprintf("ratelimit:username:%s", username), fmt.Sprintf("ratelimit:ip:%s", ipAddress)}

	for _, key := range keys {
		var rateLimit objects.RateLimit
		if err := us.db.GetJSON(key, &rateLimit); err == nil {
			if rateLimit.IsLocked() {
				return true, nil
			}
		}
	}

	return false, nil
}

func (us *UserService) logAuditEvent(userID, action, resource, resourceID, ipAddress, userAgent, details string, success bool) {
	auditLog := &objects.AuditLog{
		ID:         fmt.Sprintf("audit_%d", time.Now().UnixNano()),
		UserID:     userID,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Details:    fmt.Sprintf("%s", details),
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		Success:    success,
		Timestamp:  time.Now(),
	}

	key := fmt.Sprintf("audit:logs:%s", auditLog.ID)
	if err := us.db.StoreJSON(key, auditLog); err != nil {
		log.Printf("Failed to log audit event: %v", err)
	}
}

// CreateSession creates a new user session
func (us *UserService) CreateSession(userID, ipAddress, userAgent string) (*objects.UserSession, error) {
	now := time.Now()

	// Generate session token
	token, err := objects.GenerateSecureToken(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate session token: %w", err)
	}

	session := &objects.UserSession{
		ID:           fmt.Sprintf("session_%d", now.UnixNano()),
		UserID:       userID,
		Token:        token,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		CreatedAt:    now,
		ExpiresAt:    now.Add(us.config.SessionTimeout),
		LastActivity: now,
		IsActive:     true,
	}

	// Store session
	sessionKey := fmt.Sprintf("sessions:%s", session.ID)
	if err := us.db.StoreJSON(sessionKey, session); err != nil {
		return nil, fmt.Errorf("failed to store session: %w", err)
	}

	log.Printf("AuthService: Session created for user %s: %s", userID, session.ID)
	return session, nil
}

// GetSession retrieves a session by ID
func (us *UserService) GetSession(sessionID string) (*objects.UserSession, error) {
	var session objects.UserSession
	sessionKey := fmt.Sprintf("sessions:%s", sessionID)
	if err := us.db.GetJSON(sessionKey, &session); err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}
	return &session, nil
}

// GetSessionByToken retrieves a session by token
func (us *UserService) GetSessionByToken(token string) (*objects.UserSession, error) {
	var session objects.UserSession
	found := false

	err := us.db.GetObjectsByPrefix("sessions:", func(key string, value []byte) bool {
		var s objects.UserSession
		if err := json.Unmarshal(value, &s); err == nil && s.Token == token {
			session = s
			found = true
			return false // Stop iteration
		}
		return true // Continue
	})

	if err != nil {
		return nil, fmt.Errorf("database query failed: %w", err)
	}

	if !found {
		return nil, fmt.Errorf("session not found")
	}

	return &session, nil
}

// ValidateSession validates a session and updates its activity
func (us *UserService) ValidateSession(token string) (*objects.UserSession, error) {
	session, err := us.GetSessionByToken(token)
	if err != nil {
		return nil, fmt.Errorf("invalid session: %w", err)
	}

	if !session.IsSessionActive() {
		return nil, fmt.Errorf("session expired or inactive")
	}

	// Update last activity
	session.UpdateActivity()
	if err := us.db.StoreJSON(fmt.Sprintf("sessions:%s", session.ID), session); err != nil {
		log.Printf("Failed to update session activity: %v", err)
	}

	return session, nil
}

// RefreshSession extends the session expiration time
func (us *UserService) RefreshSession(sessionID string) error {
	session, err := us.GetSession(sessionID)
	if err != nil {
		return err
	}

	if !session.IsActive {
		return fmt.Errorf("session is not active")
	}

	// Extend expiration
	session.ExpiresAt = time.Now().Add(us.config.SessionTimeout)
	session.UpdateActivity()

	if err := us.db.StoreJSON(fmt.Sprintf("sessions:%s", session.ID), session); err != nil {
		return fmt.Errorf("failed to refresh session: %w", err)
	}

	log.Printf("AuthService: Session refreshed: %s", sessionID)
	return nil
}

// ExpireSession marks a session as inactive
func (us *UserService) ExpireSession(sessionID string) error {
	session, err := us.GetSession(sessionID)
	if err != nil {
		return err
	}

	session.IsActive = false
	if err := us.db.StoreJSON(fmt.Sprintf("sessions:%s", session.ID), session); err != nil {
		return fmt.Errorf("failed to expire session: %w", err)
	}

	log.Printf("AuthService: Session expired: %s", sessionID)
	return nil
}

// ExpireUserSessions expires all sessions for a user
func (us *UserService) ExpireUserSessions(userID string) error {
	var sessions []objects.UserSession

	// Find all sessions for the user
	err := us.db.GetObjectsByPrefix("sessions:", func(key string, value []byte) bool {
		var s objects.UserSession
		if err := json.Unmarshal(value, &s); err == nil && s.UserID == userID {
			sessions = append(sessions, s)
		}
		return true // Continue
	})

	if err != nil {
		return fmt.Errorf("failed to find user sessions: %w", err)
	}

	// Expire all sessions
	for _, session := range sessions {
		if err := us.ExpireSession(session.ID); err != nil {
			log.Printf("Failed to expire session %s: %v", session.ID, err)
		}
	}

	log.Printf("AuthService: Expired %d sessions for user %s", len(sessions), userID)
	return nil
}

// CleanupExpiredSessions removes expired sessions from the database
func (us *UserService) CleanupExpiredSessions() error {
	var expiredSessions []string

	// Find expired sessions
	err := us.db.GetObjectsByPrefix("sessions:", func(key string, value []byte) bool {
		var s objects.UserSession
		if err := json.Unmarshal(value, &s); err == nil && s.IsExpired() {
			expiredSessions = append(expiredSessions, key)
		}
		return true
	})

	if err != nil {
		return fmt.Errorf("failed to find expired sessions: %w", err)
	}

	// Delete expired sessions
	for _, key := range expiredSessions {
		if err := us.db.DeleteKey(key); err != nil {
			log.Printf("Failed to delete expired session %s: %v", key, err)
		}
	}

	log.Printf("AuthService: Cleaned up %d expired sessions", len(expiredSessions))
	return nil
}

// GetClientIP extracts the real client IP from various headers
func GetClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (most common)
	xForwardedFor := r.Header.Get("X-Forwarded-For")
	if xForwardedFor != "" {
		// Take the first IP if there are multiple
		ips := strings.Split(xForwardedFor, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			if net.ParseIP(ip) != nil {
				return ip
			}
		}
	}

	// Check X-Real-IP header
	xRealIP := r.Header.Get("X-Real-IP")
	if xRealIP != "" {
		if net.ParseIP(xRealIP) != nil {
			return xRealIP
		}
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
