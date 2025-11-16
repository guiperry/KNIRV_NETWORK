package web

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"backend_server/internal/database"
	"backend_server/internal/web/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAuthHandlers(t *testing.T) {
	db := &database.BuntDBManager{}
	auth := &middleware.AuthMiddleware{}
	handlers := NewAuthHandlers(db, auth)

	assert.NotNil(t, handlers)
	assert.Equal(t, db, handlers.db)
	assert.Equal(t, auth, handlers.auth)
}

func TestAuthHandlers_Login_InvalidJSON(t *testing.T) {
	db := &database.BuntDBManager{}
	auth := &middleware.AuthMiddleware{}
	handlers := NewAuthHandlers(db, auth)

	req := httptest.NewRequest("POST", "/auth/login", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	handlers.Login(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid request")
}

func TestAuthHandlers_Login_ValidRequest(t *testing.T) {
	// This test would require mocking the database and auth middleware
	// For now, skip this test as it requires full database setup
	t.Skip("Requires full database and auth middleware setup")
}

func TestAuthHandlers_Refresh_InvalidJSON(t *testing.T) {
	db := &database.BuntDBManager{}
	auth := &middleware.AuthMiddleware{}
	handlers := NewAuthHandlers(db, auth)

	req := httptest.NewRequest("POST", "/auth/refresh", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	handlers.Refresh(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid request")
}

func TestAuthHandlers_Refresh_ValidRequest(t *testing.T) {
	// This test would require mocking the auth middleware
	// For now, skip this test as it requires auth middleware setup
	t.Skip("Requires auth middleware setup")
}

func TestAuthHandlers_Revoke_InvalidJSON(t *testing.T) {
	db := &database.BuntDBManager{}
	auth := &middleware.AuthMiddleware{}
	handlers := NewAuthHandlers(db, auth)

	req := httptest.NewRequest("POST", "/auth/revoke", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	handlers.Revoke(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid request")
}

func TestAuthHandlers_Revoke_ValidRequest(t *testing.T) {
	// This test would require mocking the auth middleware
	// For now, skip this test as it requires auth middleware setup
	t.Skip("Requires auth middleware setup")
}

func TestAuthHandlers_Register_InvalidJSON(t *testing.T) {
	db := &database.BuntDBManager{}
	auth := &middleware.AuthMiddleware{}
	handlers := NewAuthHandlers(db, auth)

	req := httptest.NewRequest("POST", "/auth/register", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	handlers.Register(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid request")
}

func TestAuthHandlers_Register_ValidRequest(t *testing.T) {
	// This test would require mocking the database
	// For now, skip this test as it requires database setup
	t.Skip("Requires database setup")
}

func TestAuthHandlers_VerifyEmail_InvalidJSON(t *testing.T) {
	db := &database.BuntDBManager{}
	auth := &middleware.AuthMiddleware{}
	handlers := NewAuthHandlers(db, auth)

	req := httptest.NewRequest("POST", "/auth/verify-email", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	handlers.VerifyEmail(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid request")
}

func TestAuthHandlers_RequestPasswordReset_InvalidJSON(t *testing.T) {
	db := &database.BuntDBManager{}
	auth := &middleware.AuthMiddleware{}
	handlers := NewAuthHandlers(db, auth)

	req := httptest.NewRequest("POST", "/auth/request-password-reset", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	handlers.RequestPasswordReset(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid request")
}

func TestAuthHandlers_RequestPasswordReset_ValidRequest(t *testing.T) {
	// This test would require mocking the database
	// For now, skip this test as it requires database setup
	t.Skip("Requires database setup")
}

func TestAuthHandlers_ResetPassword_InvalidJSON(t *testing.T) {
	db := &database.BuntDBManager{}
	auth := &middleware.AuthMiddleware{}
	handlers := NewAuthHandlers(db, auth)

	req := httptest.NewRequest("POST", "/auth/reset-password", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	handlers.ResetPassword(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid request")
}

func TestAuthHandlers_ChangePassword_NoAuthContext(t *testing.T) {
	db := &database.BuntDBManager{}
	auth := &middleware.AuthMiddleware{}
	handlers := NewAuthHandlers(db, auth)

	req := httptest.NewRequest("POST", "/auth/change-password", nil)
	w := httptest.NewRecorder()

	handlers.ChangePassword(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "unauthorized")
}

func TestAuthHandlers_ChangePassword_InvalidJSON(t *testing.T) {
	db := &database.BuntDBManager{}
	auth := &middleware.AuthMiddleware{}
	handlers := NewAuthHandlers(db, auth)

	req := httptest.NewRequest("POST", "/auth/change-password", bytes.NewReader([]byte("invalid json")))
	// Set auth context using context.WithValue
	ctx := req.Context()
	ctx = context.WithValue(ctx, middleware.AuthContextKey, &middleware.AuthContext{
		UserID:   "user123",
		Username: "testuser",
		Role:     "user",
		Token:    "token123",
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handlers.ChangePassword(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid request")
}

func TestAuthHandlers_UpdateProfile_NoAuthContext(t *testing.T) {
	db := &database.BuntDBManager{}
	auth := &middleware.AuthMiddleware{}
	handlers := NewAuthHandlers(db, auth)

	req := httptest.NewRequest("PUT", "/auth/profile", nil)
	w := httptest.NewRecorder()

	handlers.UpdateProfile(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "unauthorized")
}

func TestAuthHandlers_UpdateProfile_InvalidJSON(t *testing.T) {
	db := &database.BuntDBManager{}
	auth := &middleware.AuthMiddleware{}
	handlers := NewAuthHandlers(db, auth)

	req := httptest.NewRequest("PUT", "/auth/profile", bytes.NewReader([]byte("invalid json")))
	// Set auth context using context.WithValue
	ctx := req.Context()
	ctx = context.WithValue(ctx, middleware.AuthContextKey, &middleware.AuthContext{
		UserID:   "user123",
		Username: "testuser",
		Role:     "user",
		Token:    "token123",
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handlers.UpdateProfile(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid request")
}

func TestAuthHandlers_Me_NoAuthContext(t *testing.T) {
	db := &database.BuntDBManager{}
	auth := &middleware.AuthMiddleware{}
	handlers := NewAuthHandlers(db, auth)

	req := httptest.NewRequest("GET", "/auth/me", nil)
	w := httptest.NewRecorder()

	handlers.Me(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "unauthorized")
}

func TestAuthHandlers_Me_WithAuthContext(t *testing.T) {
	// This test would require mocking the auth middleware
	// For now, skip this test as it requires auth middleware setup
	t.Skip("Requires auth middleware setup")
}

// Test request/response structs
func TestLoginRequest_JSON(t *testing.T) {
	req := LoginRequest{
		Username: "testuser",
		Password: "password123",
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var decoded LoginRequest
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, req.Username, decoded.Username)
	assert.Equal(t, req.Password, decoded.Password)
}

func TestLoginResponse_JSON(t *testing.T) {
	expiresAt := time.Now().Add(12 * time.Hour)
	resp := LoginResponse{
		Token:     "jwt-token",
		Role:      "admin",
		ExpiresAt: expiresAt,
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var decoded LoginResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, resp.Token, decoded.Token)
	assert.Equal(t, resp.Role, decoded.Role)
	assert.True(t, resp.ExpiresAt.Equal(decoded.ExpiresAt))
}

func TestRegisterRequest_JSON(t *testing.T) {
	req := RegisterRequest{
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  "password123",
		FirstName: "John",
		LastName:  "Doe",
		Company:   "Test Corp",
		Phone:     "123-456-7890",
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var decoded RegisterRequest
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, req.Username, decoded.Username)
	assert.Equal(t, req.Email, decoded.Email)
	assert.Equal(t, req.FirstName, decoded.FirstName)
	assert.Equal(t, req.LastName, decoded.LastName)
	assert.Equal(t, req.Company, decoded.Company)
	assert.Equal(t, req.Phone, decoded.Phone)
}

func TestRegisterResponse_JSON(t *testing.T) {
	resp := RegisterResponse{
		UserID:   "user123",
		Username: "testuser",
		Email:    "test@example.com",
		Status:   "active",
		Message:  "User registered successfully",
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var decoded RegisterResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, resp.UserID, decoded.UserID)
	assert.Equal(t, resp.Username, decoded.Username)
	assert.Equal(t, resp.Email, decoded.Email)
	assert.Equal(t, resp.Status, decoded.Status)
	assert.Equal(t, resp.Message, decoded.Message)
}