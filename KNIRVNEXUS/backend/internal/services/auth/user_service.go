package auth

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"backend_server/internal/database"
	"backend_server/internal/models"

	"github.com/tidwall/buntdb"
)

// UserService handles user-related database operations
type UserService struct {
	db *database.BuntDBManager
}

// NewUserService creates a new user service
func NewUserService(db *database.BuntDBManager) *UserService {
	return &UserService{db: db}
}

// CreateUser creates a new user with hashed password
func (us *UserService) CreateUser(registration *models.UserRegistration) (*models.User, error) {
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
	hash, salt, err := models.HashPassword(registration.Password)
	if err != nil {
		return nil, fmt.Errorf("password hashing failed: %w", err)
	}

	// Generate verification token
	verificationToken, err := models.GenerateSecureToken(32)
	if err != nil {
		return nil, fmt.Errorf("token generation failed: %w", err)
	}

	// Create user
	now := time.Now()
	user := &models.User{
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
	if err := us.db.StoreJSON(fmt.Sprintf("users:profiles:%s", user.ID), user); err != nil {
		return nil, fmt.Errorf("user storage failed: %w", err)
	}

	// Log audit event
	us.logAuditEvent("", "user_registration", "user", user.ID, "", "", "User registration initiated", true)

	log.Printf("User created: %s (%s)", user.Username, user.ID)
	return user, nil
}

// GetUserByID retrieves a user by ID
func (us *UserService) GetUserByID(userID string) (*models.User, error) {
	var user models.User
	err := us.db.GetJSON(fmt.Sprintf("users:profiles:%s", userID), &user)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	return &user, nil
}

// GetUserByUsername retrieves a user by username
func (us *UserService) GetUserByUsername(username string) (*models.User, error) {
	var user models.User
	found := false

	err := us.db.ViewTransaction(func(tx *buntdb.Tx) error {
		return tx.Ascend("users_by_username", func(key, value string) bool {
			if strings.Contains(key, username) {
				if err := us.db.GetJSON(key, &user); err == nil {
					found = true
					return false // Stop iteration
				}
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
func (us *UserService) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	found := false

	err := us.db.ViewTransaction(func(tx *buntdb.Tx) error {
		return tx.Ascend("users_by_email", func(key, value string) bool {
			if strings.Contains(key, strings.ToLower(email)) {
				if err := us.db.GetJSON(key, &user); err == nil {
					found = true
					return false // Stop iteration
				}
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
func (us *UserService) UpdateUser(userID string, updates *models.UserUpdate) error {
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
func (us *UserService) ChangePassword(userID string, change *models.ChangePassword) error {
	user, err := us.GetUserByID(userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Verify current password
	if !models.VerifyPassword(change.CurrentPassword, user.PasswordHash, user.Salt) {
		return fmt.Errorf("current password is incorrect")
	}

	// Hash new password
	hash, salt, err := models.HashPassword(change.NewPassword)
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
	var user models.User
	found := false

	err := us.db.ViewTransaction(func(tx *buntdb.Tx) error {
		return tx.Ascend("", func(key, value string) bool {
			if strings.HasPrefix(key, "users:profiles:") {
				var u models.User
				if err := us.db.GetJSON(key, &u); err == nil {
					if u.EmailVerificationToken == token {
						user = u
						found = true
						return false
					}
				}
			}
			return true
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
	resetToken, err := models.GenerateSecureToken(32)
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
func (us *UserService) ResetPassword(reset *models.PasswordReset) error {
	// Find user by reset token
	var user models.User
	found := false

	err := us.db.ViewTransaction(func(tx *buntdb.Tx) error {
		return tx.Ascend("", func(key, value string) bool {
			if strings.HasPrefix(key, "users:profiles:") {
				var u models.User
				if err := us.db.GetJSON(key, &u); err == nil {
					if u.PasswordResetToken == reset.Token && u.PasswordResetExpires != nil && time.Now().Before(*u.PasswordResetExpires) {
						user = u
						found = true
						return false
					}
				}
			}
			return true
		})
	})

	if err != nil {
		return fmt.Errorf("database query failed: %w", err)
	}

	if !found {
		return fmt.Errorf("invalid or expired reset token")
	}

	// Hash new password
	hash, salt, err := models.HashPassword(reset.Password)
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
func (us *UserService) AuthenticateUser(username, password, ipAddress, userAgent string) (*models.User, error) {
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
	if !models.VerifyPassword(password, user.PasswordHash, user.Salt) {
		us.RecordLoginAttempt(username, ipAddress, false)
		return nil, fmt.Errorf("invalid credentials")
	}

	// Successful authentication
	us.RecordLoginAttempt(username, ipAddress, true)

	// Log audit event
	us.logAuditEvent(user.ID, "login", "user", user.ID, ipAddress, userAgent, "User login successful", true)

	return user, nil
}

// Private helper methods

func (us *UserService) validateRegistration(reg *models.UserRegistration) error {
	if len(reg.Username) < 3 || len(reg.Username) > 50 {
		return fmt.Errorf("username must be between 3 and 50 characters")
	}
	if len(reg.Password) < 8 {
		return fmt.Errorf("password must be at least 8 characters long")
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

func (us *UserService) usernameExists(username string) (bool, error) {
	count, err := us.db.CountKeys(fmt.Sprintf("users:profiles:*"))
	if err != nil {
		return false, err
	}

	if count == 0 {
		return false, nil
	}

	exists := false
	err = us.db.ViewTransaction(func(tx *buntdb.Tx) error {
		return tx.Ascend("users_by_username", func(key, value string) bool {
			var user models.User
			if err := us.db.GetJSON(key, &user); err == nil && user.Username == username {
				exists = true
				return false
			}
			return true
		})
	})

	return exists, err
}

func (us *UserService) emailExists(email string) (bool, error) {
	count, err := us.db.CountKeys(fmt.Sprintf("users:profiles:*"))
	if err != nil {
		return false, err
	}

	if count == 0 {
		return false, nil
	}

	exists := false
	emailLower := strings.ToLower(email)
	err = us.db.ViewTransaction(func(tx *buntdb.Tx) error {
		return tx.Ascend("users_by_email", func(key, value string) bool {
			var user models.User
			if err := us.db.GetJSON(key, &user); err == nil && strings.ToLower(user.Email) == emailLower {
				exists = true
				return false
			}
			return true
		})
	})

	return exists, err
}

func (us *UserService) recordRateLimitAttempt(identifier, ipAddress string, success bool) error {
	// Use both username/IP for rate limiting
	keys := []string{fmt.Sprintf("ratelimit:username:%s", identifier), fmt.Sprintf("ratelimit:ip:%s", ipAddress)}

	for _, key := range keys {
		var rateLimit models.RateLimit
		err := us.db.GetJSON(key, &rateLimit)
		if err != nil {
			// Create new rate limit entry
			rateLimit = models.RateLimit{
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
		var rateLimit models.RateLimit
		if err := us.db.GetJSON(key, &rateLimit); err == nil {
			if rateLimit.IsLocked() {
				return true, nil
			}
		}
	}

	return false, nil
}

func (us *UserService) logAuditEvent(userID, action, resource, resourceID, ipAddress, userAgent, details string, success bool) {
	auditLog := &models.AuditLog{
		ID:         fmt.Sprintf("audit_%d", time.Now().UnixNano()),
		UserID:     userID,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		Timestamp:  time.Now(),
		Success:    success,
		Details:    details,
	}

	key := fmt.Sprintf("audit:logs:%s", auditLog.ID)
	if err := us.db.StoreJSON(key, auditLog); err != nil {
		log.Printf("Failed to log audit event: %v", err)
	}
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
