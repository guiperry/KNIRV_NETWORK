// Copyright 2026 KNIRV-NEXUS
// SPDX-License-Identifier: GPL-3.0-or-later

package auth

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"backend_server/internal/database"
	"backend_server/internal/objects"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAuthConfig tests authentication configuration
func TestAuthConfig(t *testing.T) {
	t.Run("Default config", func(t *testing.T) {
		config := DefaultAuthConfig()
		assert.Equal(t, 8, config.PasswordMinLength)
		assert.True(t, config.PasswordRequireUpper)
		assert.True(t, config.PasswordRequireLower)
		assert.True(t, config.PasswordRequireDigit)
		assert.False(t, config.PasswordRequireSpecial)
		assert.Equal(t, 5, config.MaxLoginAttempts)
		assert.Equal(t, 15*time.Minute, config.LockoutDuration)
		assert.Equal(t, 30*time.Minute, config.SessionTimeout)
	})

	t.Run("Custom config", func(t *testing.T) {
		config := AuthConfig{
			PasswordMinLength:      12,
			PasswordRequireSpecial: true,
			MaxLoginAttempts:       3,
			LockoutDuration:        30 * time.Minute,
			SessionTimeout:         1 * time.Hour,
		}
		assert.Equal(t, 12, config.PasswordMinLength)
		assert.True(t, config.PasswordRequireSpecial)
		assert.Equal(t, 3, config.MaxLoginAttempts)
	})
}

// TestPasswordValidation tests enhanced password validation
func TestPasswordValidation(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := database.NewBuntDB(dbPath)
	require.NoError(t, err)
	defer db.Close()
	defer os.Remove(dbPath)

	tests := []struct {
		name     string
		config   AuthConfig
		password string
		wantErr  bool
		errMsg   string
	}{
		{
			name: "Valid password - all requirements",
			config: AuthConfig{
				PasswordMinLength:      8,
				PasswordRequireUpper:   true,
				PasswordRequireLower:   true,
				PasswordRequireDigit:   true,
				PasswordRequireSpecial: true,
			},
			password: "Test123!",
			wantErr:  false,
		},
		{
			name: "Too short",
			config: AuthConfig{
				PasswordMinLength: 10,
			},
			password: "Test123",
			wantErr:  true,
			errMsg:   "at least 10 characters",
		},
		{
			name: "Missing uppercase",
			config: AuthConfig{
				PasswordMinLength:    8,
				PasswordRequireUpper: true,
			},
			password: "test1234",
			wantErr:  true,
			errMsg:   "uppercase letter",
		},
		{
			name: "Missing lowercase",
			config: AuthConfig{
				PasswordMinLength:    8,
				PasswordRequireLower: true,
			},
			password: "TEST1234",
			wantErr:  true,
			errMsg:   "lowercase letter",
		},
		{
			name: "Missing digit",
			config: AuthConfig{
				PasswordMinLength:    8,
				PasswordRequireDigit: true,
			},
			password: "TestTest",
			wantErr:  true,
			errMsg:   "digit",
		},
		{
			name: "Missing special character",
			config: AuthConfig{
				PasswordMinLength:      8,
				PasswordRequireSpecial: true,
			},
			password: "Test1234",
			wantErr:  true,
			errMsg:   "special character",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userService := NewUserServiceWithConfig(db, tt.config)
			err := userService.validatePassword(tt.password)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestSessionManagement tests comprehensive session management
func TestSessionManagement(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := database.NewBuntDB(dbPath)
	require.NoError(t, err)
	defer db.Close()
	defer os.Remove(dbPath)

	userService := NewUserService(db)

	t.Run("Create session", func(t *testing.T) {
		session, err := userService.CreateSession("user123", "192.168.1.1", "Mozilla/5.0")
		require.NoError(t, err)
		assert.NotEmpty(t, session.ID)
		assert.Equal(t, "user123", session.UserID)
		assert.NotEmpty(t, session.Token)
		assert.Equal(t, "192.168.1.1", session.IPAddress)
		assert.Equal(t, "Mozilla/5.0", session.UserAgent)
		assert.True(t, session.IsActive)
		assert.False(t, session.IsExpired())
	})

	t.Run("Get session by ID", func(t *testing.T) {
		// Create session
		created, err := userService.CreateSession("user456", "10.0.0.1", "Chrome")
		require.NoError(t, err)

		// Retrieve session
		retrieved, err := userService.GetSession(created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, retrieved.ID)
		assert.Equal(t, created.UserID, retrieved.UserID)
		assert.Equal(t, created.Token, retrieved.Token)
	})

	t.Run("Get session by token", func(t *testing.T) {
		// Create session
		created, err := userService.CreateSession("user789", "172.16.0.1", "Firefox")
		require.NoError(t, err)

		// Retrieve by token
		retrieved, err := userService.GetSessionByToken(created.Token)
		require.NoError(t, err)
		assert.Equal(t, created.ID, retrieved.ID)
		assert.Equal(t, created.UserID, retrieved.UserID)
	})

	t.Run("Validate active session", func(t *testing.T) {
		// Create session
		created, err := userService.CreateSession("user_active", "10.1.1.1", "Safari")
		require.NoError(t, err)

		// Validate session
		validated, err := userService.ValidateSession(created.Token)
		require.NoError(t, err)
		assert.True(t, validated.IsActive)
		assert.False(t, validated.IsExpired())

		// Check last activity was updated
		assert.True(t, validated.LastActivity.After(created.LastActivity) || validated.LastActivity.Equal(created.LastActivity))
	})

	t.Run("Validate invalid token", func(t *testing.T) {
		_, err := userService.ValidateSession("invalid-token-12345")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid session")
	})

	t.Run("Refresh session", func(t *testing.T) {
		// Create session
		created, err := userService.CreateSession("user_refresh", "10.2.2.2", "Edge")
		require.NoError(t, err)

		originalExpiry := created.ExpiresAt

		// Wait a moment
		time.Sleep(100 * time.Millisecond)

		// Refresh session
		err = userService.RefreshSession(created.ID)
		require.NoError(t, err)

		// Get refreshed session
		refreshed, err := userService.GetSession(created.ID)
		require.NoError(t, err)
		assert.True(t, refreshed.ExpiresAt.After(originalExpiry))
	})

	t.Run("Expire session", func(t *testing.T) {
		// Create session
		created, err := userService.CreateSession("user_expire", "10.3.3.3", "Opera")
		require.NoError(t, err)

		// Expire session
		err = userService.ExpireSession(created.ID)
		require.NoError(t, err)

		// Verify session is expired
		expired, err := userService.GetSession(created.ID)
		require.NoError(t, err)
		assert.False(t, expired.IsActive)

		// Validate should fail
		_, err = userService.ValidateSession(created.Token)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expired or inactive")
	})

	t.Run("Expire all user sessions", func(t *testing.T) {
		userID := "user_multi_sessions"

		// Create multiple sessions for same user
		session1, err := userService.CreateSession(userID, "10.4.4.1", "Chrome")
		require.NoError(t, err)
		session2, err := userService.CreateSession(userID, "10.4.4.2", "Firefox")
		require.NoError(t, err)
		session3, err := userService.CreateSession(userID, "10.4.4.3", "Safari")
		require.NoError(t, err)

		// Expire all sessions
		err = userService.ExpireUserSessions(userID)
		require.NoError(t, err)

		// Verify all sessions are expired
		expired1, err := userService.GetSession(session1.ID)
		require.NoError(t, err)
		assert.False(t, expired1.IsActive)

		expired2, err := userService.GetSession(session2.ID)
		require.NoError(t, err)
		assert.False(t, expired2.IsActive)

		expired3, err := userService.GetSession(session3.ID)
		require.NoError(t, err)
		assert.False(t, expired3.IsActive)
	})

	t.Run("Cleanup expired sessions", func(t *testing.T) {
		// Create session with short timeout
		customConfig := DefaultAuthConfig()
		customConfig.SessionTimeout = 1 * time.Millisecond
		customService := NewUserServiceWithConfig(db, customConfig)

		session, err := customService.CreateSession("user_cleanup", "10.5.5.5", "Test")
		require.NoError(t, err)

		// Wait for session to expire
		time.Sleep(50 * time.Millisecond)

		// Verify session is expired
		expired, err := customService.GetSession(session.ID)
		require.NoError(t, err)
		assert.True(t, expired.IsExpired())

		// Cleanup expired sessions
		err = customService.CleanupExpiredSessions()
		require.NoError(t, err)

		// Session should be deleted
		_, err = customService.GetSession(session.ID)
		assert.Error(t, err)
	})
}

// TestUserCreationWithPasswordPolicies tests user creation with various password policies
func TestUserCreationWithPasswordPolicies(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := database.NewBuntDB(dbPath)
	require.NoError(t, err)
	defer db.Close()
	defer os.Remove(dbPath)

	t.Run("Create user with strict policy", func(t *testing.T) {
		strictConfig := AuthConfig{
			PasswordMinLength:      12,
			PasswordRequireUpper:   true,
			PasswordRequireLower:   true,
			PasswordRequireDigit:   true,
			PasswordRequireSpecial: true,
		}
		userService := NewUserServiceWithConfig(db, strictConfig)

		// Valid password meeting strict requirements
		registration := &objects.UserRegistration{
			Username:  "strictuser",
			Email:     "strict@example.com",
			Password:  "StrictPass123!",
			FirstName: "Strict",
			LastName:  "User",
		}

		user, err := userService.CreateUser(registration)
		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, "strictuser", user.Username)
	})

	t.Run("Fail to create user with weak password under strict policy", func(t *testing.T) {
		strictConfig := AuthConfig{
			PasswordMinLength:      12,
			PasswordRequireUpper:   true,
			PasswordRequireLower:   true,
			PasswordRequireDigit:   true,
			PasswordRequireSpecial: true,
		}
		userService := NewUserServiceWithConfig(db, strictConfig)

		// Weak password
		registration := &objects.UserRegistration{
			Username:  "weakuser",
			Email:     "weak@example.com",
			Password:  "weak",
			FirstName: "Weak",
			LastName:  "User",
		}

		_, err := userService.CreateUser(registration)
		assert.Error(t, err)
	})

	t.Run("Create user with lenient policy", func(t *testing.T) {
		lenientConfig := AuthConfig{
			PasswordMinLength:      6,
			PasswordRequireUpper:   false,
			PasswordRequireLower:   false,
			PasswordRequireDigit:   false,
			PasswordRequireSpecial: false,
		}
		userService := NewUserServiceWithConfig(db, lenientConfig)

		registration := &objects.UserRegistration{
			Username:  "lenientuser",
			Email:     "lenient@example.com",
			Password:  "simple",
			FirstName: "Lenient",
			LastName:  "User",
		}

		user, err := userService.CreateUser(registration)
		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, "lenientuser", user.Username)
	})
}

// TestAuthenticationFlow tests the complete authentication flow with sessions
func TestAuthenticationFlow(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := database.NewBuntDB(dbPath)
	require.NoError(t, err)
	defer db.Close()
	defer os.Remove(dbPath)

	userService := NewUserService(db)

	// Step 1: Create user
	registration := &objects.UserRegistration{
		Username:  "authuser",
		Email:     "auth@example.com",
		Password:  "AuthPass123",
		FirstName: "Auth",
		LastName:  "User",
	}

	user, err := userService.CreateUser(registration)
	require.NoError(t, err)
	assert.NotNil(t, user)

	// Step 2: Verify email (simulate)
	err = userService.VerifyEmail(user.EmailVerificationToken)
	require.NoError(t, err)

	// Step 3: Authenticate user
	authenticatedUser, err := userService.AuthenticateUser("authuser", "AuthPass123", "192.168.1.100")
	require.NoError(t, err)
	assert.Equal(t, user.ID, authenticatedUser.ID)

	// Step 4: Create session after login
	session, err := userService.CreateSession(authenticatedUser.ID, "192.168.1.100", "TestBrowser")
	require.NoError(t, err)
	assert.NotEmpty(t, session.Token)

	// Step 5: Validate session
	validatedSession, err := userService.ValidateSession(session.Token)
	require.NoError(t, err)
	assert.True(t, validatedSession.IsActive)

	// Step 6: Logout (expire session)
	err = userService.ExpireSession(session.ID)
	require.NoError(t, err)

	// Step 7: Verify session is no longer valid
	_, err = userService.ValidateSession(session.Token)
	assert.Error(t, err)
}
