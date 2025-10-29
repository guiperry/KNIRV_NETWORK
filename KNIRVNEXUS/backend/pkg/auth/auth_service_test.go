package auth

import (
	"testing"
	"time"

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