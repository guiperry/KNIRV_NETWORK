package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	dataengine "nexus-backend/internal/services/data-engine"
	"nexus-backend/pkg/host"
)

func setupTestServer(t *testing.T) *APIServer {
	// Create test host controller
	hostConfig := host.HostConfig{
		EnableMonitoring: false,
		EnableP2P:        false,
		EnableTEE:        false,
	}
	hostController, err := host.NewHostController(hostConfig)
	require.NoError(t, err)

	// Create test data engine
	dataEngineConfig := dataengine.DataEngineConfig{
		RetentionDays: 7,
	}
	dataEngine, err := dataengine.NewBuntDBDataEngine(":memory:", dataEngineConfig)
	require.NoError(t, err)

	// Create API config
	apiConfig := APIConfig{
		Port:           8080,
		Host:           "localhost",
		EnableCORS:     true,
		CORSOrigins:    []string{"*"},
		JWTSecret:      "test-secret",
		RateLimitRPS:   100,
		RequestTimeout: 30 * time.Second,
	}

	// Create API server
	server, err := NewAPIServer(
		hostController,
		dataEngine,
		nil, // agentServer
		nil, // inferenceService
		nil, // cdeService
		apiConfig,
	)
	require.NoError(t, err)

	return server
}

func TestAPIServerCreation(t *testing.T) {
	server := setupTestServer(t)
	assert.NotNil(t, server)
	assert.False(t, server.IsRunning())
}

func TestAPIServerStartStop(t *testing.T) {
	server := setupTestServer(t)

	// Test start
	err := server.Start()
	assert.NoError(t, err)
	assert.True(t, server.IsRunning())

	// Test stop
	err = server.Stop()
	assert.NoError(t, err)
	assert.False(t, server.IsRunning())
}

func TestHealthEndpoint(t *testing.T) {
	server := setupTestServer(t)

	req, err := http.NewRequest("GET", "/api/v1/health", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response APIResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)
}

func TestStatusEndpoint(t *testing.T) {
	server := setupTestServer(t)

	req, err := http.NewRequest("GET", "/api/v1/status", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response APIResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)
}

func TestUserRegistration(t *testing.T) {
	server := setupTestServer(t)

	registerReq := RegisterRequest{
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  "password123",
		FirstName: "Test",
		LastName:  "User",
	}

	reqBody, err := json.Marshal(registerReq)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)

	var response APIResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)
}

func TestUserLogin(t *testing.T) {
	server := setupTestServer(t)

	// First register a user
	registerReq := RegisterRequest{
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  "password123",
		FirstName: "Test",
		LastName:  "User",
	}

	reqBody, err := json.Marshal(registerReq)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusCreated, rr.Code)

	// Now try to login
	loginReq := LoginRequest{
		Username: "testuser",
		Password: "password123",
	}

	reqBody, err = json.Marshal(loginReq)
	require.NoError(t, err)

	req, err = http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr = httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response APIResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)

	// Check that we got a token
	loginResp, ok := response.Data.(map[string]interface{})
	assert.True(t, ok)
	assert.Contains(t, loginResp, "token")
}

func TestProtectedEndpoint(t *testing.T) {
	server := setupTestServer(t)

	// Try to access protected endpoint without token
	req, err := http.NewRequest("GET", "/api/v1/users/me", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestCORSHeaders(t *testing.T) {
	server := setupTestServer(t)

	req, err := http.NewRequest("OPTIONS", "/api/v1/health", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "http://localhost:3000")

	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Header().Get("Access-Control-Allow-Origin"), "*")
}

func TestRateLimiting(t *testing.T) {
	server := setupTestServer(t)

	// This test would need to be implemented based on the actual rate limiting logic
	// For now, just test that the endpoint responds
	req, err := http.NewRequest("GET", "/api/v1/health", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestJSONValidation(t *testing.T) {
	server := setupTestServer(t)

	// Send invalid JSON
	req, err := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer([]byte("invalid json")))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestErrorHandling(t *testing.T) {
	server := setupTestServer(t)

	// Test 404
	req, err := http.NewRequest("GET", "/api/v1/nonexistent", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestRequestIDMiddleware(t *testing.T) {
	server := setupTestServer(t)

	req, err := http.NewRequest("GET", "/api/v1/health", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.NotEmpty(t, rr.Header().Get("X-Request-ID"))
}

func TestSecurityHeaders(t *testing.T) {
	server := setupTestServer(t)

	req, err := http.NewRequest("GET", "/api/v1/health", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	// Note: Security headers would be added by the security middleware
	// This test verifies the endpoint works
}

func TestConcurrentRequests(t *testing.T) {
	server := setupTestServer(t)

	// Test concurrent requests
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()

			req, err := http.NewRequest("GET", "/api/v1/health", nil)
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			server.router.ServeHTTP(rr, req)

			assert.Equal(t, http.StatusOK, rr.Code)
		}()
	}

	// Wait for all requests to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}
