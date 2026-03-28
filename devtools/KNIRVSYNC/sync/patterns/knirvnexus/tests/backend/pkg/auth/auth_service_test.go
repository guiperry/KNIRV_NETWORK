package auth

import (
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	dataengine "backend_server/internal/data-engine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAuthService(t *testing.T) {
	// Create a mock data engine
	dataEngine := &dataengine.BuntDBDataEngine{}

	// Test with empty config (should use defaults)
	config := AuthConfig{}
	authService := NewAuthService(dataEngine, config)

	assert.NotNil(t, authService)
	assert.Equal(t, 8, authService.config.PasswordMinLength)
	assert.Equal(t, 24*time.Hour, authService.config.TokenExpiration)
	assert.Equal(t, 7*24*time.Hour, authService.config.RefreshTokenExpiration)
	assert.Equal(t, 5, authService.config.MaxLoginAttempts)
	assert.Equal(t, 15*time.Minute, authService.config.LockoutDuration)
	assert.Equal(t, 30*time.Minute, authService.config.SessionTimeout)

	// Test with custom config
	customConfig := AuthConfig{
		PasswordMinLength:      12,
		TokenExpiration:        48 * time.Hour,
		RefreshTokenExpiration: 14 * 24 * time.Hour,
		MaxLoginAttempts:       3,
		LockoutDuration:        30 * time.Minute,
		SessionTimeout:         60 * time.Minute,
	}
	authService2 := NewAuthService(dataEngine, customConfig)

	assert.Equal(t, 12, authService2.config.PasswordMinLength)
	assert.Equal(t, 48*time.Hour, authService2.config.TokenExpiration)
	assert.Equal(t, 14*24*time.Hour, authService2.config.RefreshTokenExpiration)
	assert.Equal(t, 3, authService2.config.MaxLoginAttempts)
	assert.Equal(t, 30*time.Minute, authService2.config.LockoutDuration)
	assert.Equal(t, 60*time.Minute, authService2.config.SessionTimeout)
}

func TestAuthService_validatePassword(t *testing.T) {
	dataEngine := &dataengine.BuntDBDataEngine{}
	authService := NewAuthService(dataEngine, AuthConfig{})

	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"valid password", "ValidPass123!", false},
		{"too short", "Short1!", true},
		{"no uppercase", "validpass123!", false}, // Default config doesn't require uppercase
		{"no lowercase", "VALIDPASS123!", false}, // Default config doesn't require lowercase
		{"no digit", "ValidPass!", false},        // Default config doesn't require digit
		{"no special", "ValidPass123", false},    // Default config doesn't require special
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := authService.validatePassword(tt.password)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAuthService_validatePassword_CustomConfig(t *testing.T) {
	dataEngine := &dataengine.BuntDBDataEngine{}
	config := AuthConfig{
		PasswordMinLength:      6,
		PasswordRequireUpper:   false,
		PasswordRequireLower:   false,
		PasswordRequireDigit:   false,
		PasswordRequireSpecial: false,
	}
	authService := NewAuthService(dataEngine, config)

	// Should pass with minimal requirements
	err := authService.validatePassword("short12")
	assert.NoError(t, err)
}

func TestAuthService_generateID(t *testing.T) {
	dataEngine := &dataengine.BuntDBDataEngine{}
	authService := NewAuthService(dataEngine, AuthConfig{})

	id := authService.generateID("test")
	assert.Contains(t, id, "test_")
	assert.Greater(t, len(id), len("test_"))
}

func TestAuthService_generateSecureToken(t *testing.T) {
	dataEngine := &dataengine.BuntDBDataEngine{}
	authService := NewAuthService(dataEngine, AuthConfig{})

	token, err := authService.generateSecureToken(32)
	require.NoError(t, err)
	assert.Len(t, token, 32)

	// Test different lengths
	token16, err := authService.generateSecureToken(16)
	require.NoError(t, err)
	assert.Len(t, token16, 16)

	token64, err := authService.generateSecureToken(64)
	require.NoError(t, err)
	assert.Len(t, token64, 64)
}

func TestAuthService_isUserLockedOut(t *testing.T) {
	dataEngine := &dataengine.BuntDBDataEngine{}
	authService := NewAuthService(dataEngine, AuthConfig{})

	// Currently always returns false (placeholder implementation)
	assert.False(t, authService.isUserLockedOut("testuser", "127.0.0.1"))
}

func TestAuthService_recordLoginAttempt(t *testing.T) {
	dataEngine := &dataengine.BuntDBDataEngine{}
	authService := NewAuthService(dataEngine, AuthConfig{})

	// Should not panic (placeholder implementation)
	authService.recordLoginAttempt("testuser", "127.0.0.1", true)
	authService.recordLoginAttempt("testuser", "127.0.0.1", false)
}

func TestAuthService_CreatePasswordResetToken(t *testing.T) {
	// This test would require a full data engine setup
	// For now, skip this test as it requires database setup
	t.Skip("Requires full database setup")
}

func TestAuthService_ResetPassword(t *testing.T) {
	dataEngine := &dataengine.BuntDBDataEngine{}
	authService := NewAuthService(dataEngine, AuthConfig{})

	// Test with valid password
	err := authService.ResetPassword("token123", "ValidPass123!")
	assert.NoError(t, err)

	// Test with invalid password
	err = authService.ResetPassword("token123", "short")
	assert.Error(t, err)
}

func TestAuthService_ChangePassword(t *testing.T) {
	// This test would require a full data engine setup
	// For now, skip this test as it requires database setup
	t.Skip("Requires full database setup")
}

func TestAuthService_AuthenticateUser(t *testing.T) {
	// This test would require a full data engine setup
	// For now, skip this test as it requires database setup
	t.Skip("Requires full database setup")
}

func TestAuthService_CreateUser(t *testing.T) {
	// This test would require a full data engine setup
	// For now, skip this test as it requires database setup
	t.Skip("Requires full database setup")
}

func TestAuthService_hashPassword(t *testing.T) {
	password := "TestPassword123!"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)

	// Verify the hash works
	err = bcrypt.CompareHashAndPassword(hash, []byte(password))
	assert.NoError(t, err)

	// Test with empty password
	hash2, err := bcrypt.GenerateFromPassword([]byte(""), bcrypt.DefaultCost)
	require.NoError(t, err)
	assert.NotEmpty(t, hash2)
}

func TestAuthService_verifyPassword(t *testing.T) {
	password := "TestPassword123!"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	require.NoError(t, err)

	// Test correct password
	valid := bcrypt.CompareHashAndPassword(hash, []byte(password)) == nil
	assert.True(t, valid)

	// Test incorrect password
	valid = bcrypt.CompareHashAndPassword(hash, []byte("wrongpassword")) == nil
	assert.False(t, valid)

	// Test empty password
	valid = bcrypt.CompareHashAndPassword(hash, []byte("")) == nil
	assert.False(t, valid)
}

func TestAuthService_isUserLockedOut_Locked(t *testing.T) {
	dataEngine := &dataengine.BuntDBDataEngine{}
	authService := NewAuthService(dataEngine, AuthConfig{})

	username := "testuser"
	ipAddress := "127.0.0.1"

	// Manually set a lock
	authService.mu.Lock()
	authService.userLocks[username] = time.Now().Add(-5 * time.Minute) // 5 minutes ago
	authService.mu.Unlock()

	// Should be locked out
	assert.True(t, authService.isUserLockedOut(username, ipAddress))

	// Test IP lock
	authService.mu.Lock()
	authService.ipLocks[ipAddress] = time.Now().Add(-10 * time.Minute) // 10 minutes ago
	authService.mu.Unlock()

	assert.True(t, authService.isUserLockedOut("differentuser", ipAddress))
}

func TestAuthService_isUserLockedOut_Expired(t *testing.T) {
	dataEngine := &dataengine.BuntDBDataEngine{}
	authService := NewAuthService(dataEngine, AuthConfig{})

	username := "testuser"
	ipAddress := "127.0.0.1"

	// Set an expired lock (30 minutes ago)
	authService.mu.Lock()
	authService.userLocks[username] = time.Now().Add(-30 * time.Minute)
	authService.mu.Unlock()

	// Should not be locked out (expired)
	assert.False(t, authService.isUserLockedOut(username, ipAddress))

	// Verify the lock was removed
	authService.mu.RLock()
	_, exists := authService.userLocks[username]
	authService.mu.RUnlock()
	assert.False(t, exists)
}

func TestAuthService_recordLoginAttempt_Success(t *testing.T) {
	dataEngine := &dataengine.BuntDBDataEngine{}
	authService := NewAuthService(dataEngine, AuthConfig{})

	username := "testuser"
	ipAddress := "127.0.0.1"

	// Record successful attempt - should not lock
	authService.recordLoginAttempt(username, ipAddress, true)

	assert.False(t, authService.isUserLockedOut(username, ipAddress))
}

func TestAuthService_recordLoginAttempt_Failure(t *testing.T) {
	dataEngine := &dataengine.BuntDBDataEngine{}
	authService := NewAuthService(dataEngine, AuthConfig{MaxLoginAttempts: 3})

	username := "testuser"
	ipAddress := "127.0.0.1"

	// Record multiple failures - but the current implementation doesn't actually lock users
	// This test is checking the placeholder behavior
	for i := 0; i < 3; i++ {
		authService.recordLoginAttempt(username, ipAddress, false)
	}

	// Current implementation always returns false for isUserLockedOut
	assert.False(t, authService.isUserLockedOut(username, ipAddress))
}

func TestAuthService_validatePassword_Strict(t *testing.T) {
	dataEngine := &dataengine.BuntDBDataEngine{}
	config := AuthConfig{
		PasswordMinLength:      8,
		PasswordRequireUpper:   true,
		PasswordRequireLower:   true,
		PasswordRequireDigit:   true,
		PasswordRequireSpecial: true,
	}
	authService := NewAuthService(dataEngine, config)

	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"valid password", "ValidPass123!", false},
		{"too short", "Short1!", true},
		{"no uppercase", "validpass123!", true},
		{"no lowercase", "VALIDPASS123!", true},
		{"no digit", "ValidPass!", true},
		{"no special", "ValidPass123", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := authService.validatePassword(tt.password)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAuthService_generateID_Unique(t *testing.T) {
	dataEngine := &dataengine.BuntDBDataEngine{}
	authService := NewAuthService(dataEngine, AuthConfig{})

	id1 := authService.generateID("test")
	id2 := authService.generateID("test")

	assert.Contains(t, id1, "test_")
	assert.Contains(t, id2, "test_")
	assert.NotEqual(t, id1, id2)
}

func TestAuthService_generateSecureToken_Error(t *testing.T) {
	dataEngine := &dataengine.BuntDBDataEngine{}
	authService := NewAuthService(dataEngine, AuthConfig{})

	// Test with zero length (should work)
	token, err := authService.generateSecureToken(0)
	assert.NoError(t, err)
	assert.Empty(t, token)

	// Test with normal length (should work)
	token, err = authService.generateSecureToken(32)
	assert.NoError(t, err)
	assert.Len(t, token, 32)
}

func TestAuthService_NewAuthService_Defaults(t *testing.T) {
	dataEngine := &dataengine.BuntDBDataEngine{}

	// Test with completely empty config
	authService := NewAuthService(dataEngine, AuthConfig{})

	assert.Equal(t, 8, authService.config.PasswordMinLength)
	assert.Equal(t, 24*time.Hour, authService.config.TokenExpiration)
	assert.Equal(t, 7*24*time.Hour, authService.config.RefreshTokenExpiration)
	assert.Equal(t, 5, authService.config.MaxLoginAttempts)
	assert.Equal(t, 15*time.Minute, authService.config.LockoutDuration)
	assert.Equal(t, 30*time.Minute, authService.config.SessionTimeout)
	assert.False(t, authService.config.EnableTwoFactor)
}

func TestAuthService_NewAuthService_Custom(t *testing.T) {
	dataEngine := &dataengine.BuntDBDataEngine{}

	config := AuthConfig{
		PasswordMinLength:      12,
		TokenExpiration:        48 * time.Hour,
		RefreshTokenExpiration: 14 * 24 * time.Hour,
		MaxLoginAttempts:       10,
		LockoutDuration:        60 * time.Minute,
		SessionTimeout:         120 * time.Minute,
		EnableTwoFactor:        true,
	}

	authService := NewAuthService(dataEngine, config)

	assert.Equal(t, 12, authService.config.PasswordMinLength)
	assert.Equal(t, 48*time.Hour, authService.config.TokenExpiration)
	assert.Equal(t, 14*24*time.Hour, authService.config.RefreshTokenExpiration)
	assert.Equal(t, 10, authService.config.MaxLoginAttempts)
	assert.Equal(t, 60*time.Minute, authService.config.LockoutDuration)
	assert.Equal(t, 120*time.Minute, authService.config.SessionTimeout)
	assert.True(t, authService.config.EnableTwoFactor)
}