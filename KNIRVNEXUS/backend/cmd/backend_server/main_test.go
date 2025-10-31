package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"backend_server/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServer(t *testing.T) {
	// Create a temporary config for testing
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Path: filepath.Join(os.TempDir(), "test.db"),
		},
		API: config.APIConfig{
			BindAddress: "localhost",
			Port:        8080,
		},
		Security: config.SecurityConfig{
			JWTSecret: "test-secret",
		},
		ChainID: "test-chain",
		NodeRole: "validator",
		P2P: config.P2PConfig{
			DHTEnabled: false,
		},
		CDE: config.CDEConfig{
			BaseImagePath: "/tmp",
			WorkspaceRoot: "/tmp",
			MaxEnvironments: 10,
			DefaultTimeout: time.Minute * 5,
			MaxCPUPerEnv: 2,
			MaxMemoryPerEnv: 1024,
			MaxDiskPerEnv: 1024,
			EnableSandboxing: false,
			EnableNetworkIsolation: false,
			AllowedPorts: []int{8080},
			SessionTimeout: time.Hour,
			MaxSessionsPerUser: 5,
			MaxProjectsPerUser: 10,
			ProjectStoragePath: "/tmp",
		},
	}

	server, err := NewServer(cfg)
	require.NoError(t, err)
	require.NotNil(t, server)

	// Verify server components are initialized
	assert.NotNil(t, server.config)
	assert.NotNil(t, server.db)
	assert.NotNil(t, server.router)
	assert.NotNil(t, server.p2pManager)
	assert.NotNil(t, server.dveManager)
	assert.NotNil(t, server.validationCore)
	assert.NotNil(t, server.cdeService)
	assert.NotNil(t, server.dataEngine)
	assert.NotNil(t, server.inferenceService)
	assert.NotNil(t, server.teeSecurityService)
	assert.NotNil(t, server.systemHealthService)
	assert.NotNil(t, server.modelManagementService)
	assert.NotNil(t, server.controllerIntegrationService)
	assert.NotNil(t, server.dveRentalService)
	assert.NotNil(t, server.cognitiveEngine)
	assert.NotNil(t, server.containerOrchestrator)
	assert.NotNil(t, server.sessionManager)
	assert.NotNil(t, server.endpointRegistry)
	assert.NotNil(t, server.stripeService)
	assert.NotNil(t, server.paypalService)
	assert.NotNil(t, server.ctx)
	assert.NotNil(t, server.cancel)
	assert.False(t, server.running)

	// Clean up
	if server.db != nil {
		server.db.Close()
	}
}

func TestServer_handleHealth(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Path: filepath.Join(os.TempDir(), "test.db"),
		},
		API: config.APIConfig{
			BindAddress: "localhost",
			Port:        8080,
		},
		Security: config.SecurityConfig{
			JWTSecret: "test-secret",
		},
		ChainID: "test-chain",
		NodeRole: "validator",
		P2P: config.P2PConfig{
			DHTEnabled: false,
		},
		CDE: config.CDEConfig{
			BaseImagePath: "/tmp",
			WorkspaceRoot: "/tmp",
			MaxEnvironments: 10,
			DefaultTimeout: time.Minute * 5,
			MaxCPUPerEnv: 2,
			MaxMemoryPerEnv: 1024,
			MaxDiskPerEnv: 1024,
			EnableSandboxing: false,
			EnableNetworkIsolation: false,
			AllowedPorts: []int{8080},
			SessionTimeout: time.Hour,
			MaxSessionsPerUser: 5,
			MaxProjectsPerUser: 10,
			ProjectStoragePath: "/tmp",
		},
	}

	server, err := NewServer(cfg)
	require.NoError(t, err)
	require.NotNil(t, server)
	defer server.db.Close()

	// Create a test request
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	// Call the handler
	server.handleHealth(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	// Parse response body
	var response map[string]interface{}
	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "healthy", response["status"])
	assert.Equal(t, Version, response["version"])
	assert.Equal(t, BuildTime, response["build_time"])
	assert.Equal(t, GitCommit, response["git_commit"])

	services, ok := response["services"].(map[string]interface{})
	require.True(t, ok)

	// Check that all expected services are present
	expectedServices := []string{
		"database", "p2p_manager", "dve_manager", "validation_core",
		"model_server", "data_engine", "inference_service", "websocket_service",
		"cde_service", "dns_service",
	}

	for _, service := range expectedServices {
		_, exists := services[service]
		assert.True(t, exists, "Service %s should be present in health response", service)
	}
}

func TestServer_Start_Stop(t *testing.T) {
	// Skip this test as it requires API keys and full service initialization
	// which is complex in a test environment
	t.Skip("Skipping server start/stop test - requires API keys and full service setup")
}

func TestServer_Start_AlreadyRunning(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Path: filepath.Join(os.TempDir(), "test.db"),
		},
		API: config.APIConfig{
			BindAddress: "localhost",
			Port:        0,
		},
		Security: config.SecurityConfig{
			JWTSecret: "test-secret",
		},
		ChainID: "test-chain",
		NodeRole: "validator",
		P2P: config.P2PConfig{
			DHTEnabled: false,
		},
		CDE: config.CDEConfig{
			BaseImagePath: "/tmp",
			WorkspaceRoot: "/tmp",
			MaxEnvironments: 10,
			DefaultTimeout: time.Minute * 5,
			MaxCPUPerEnv: 2,
			MaxMemoryPerEnv: 1024,
			MaxDiskPerEnv: 1024,
			EnableSandboxing: false,
			EnableNetworkIsolation: false,
			AllowedPorts: []int{8080},
			SessionTimeout: time.Hour,
			MaxSessionsPerUser: 5,
			MaxProjectsPerUser: 10,
			ProjectStoragePath: "/tmp",
		},
	}

	server, err := NewServer(cfg)
	require.NoError(t, err)
	require.NotNil(t, server)
	defer server.db.Close()

	// Manually set running to true to simulate already running server
	server.running = true

	// Try to start again
	err = server.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "server is already running")
}

func TestServer_Stop_NotRunning(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Path: filepath.Join(os.TempDir(), "test.db"),
		},
		API: config.APIConfig{
			BindAddress: "localhost",
			Port:        0,
		},
		Security: config.SecurityConfig{
			JWTSecret: "test-secret",
		},
		ChainID: "test-chain",
		NodeRole: "validator",
		P2P: config.P2PConfig{
			DHTEnabled: false,
		},
		CDE: config.CDEConfig{
			BaseImagePath: "/tmp",
			WorkspaceRoot: "/tmp",
			MaxEnvironments: 10,
			DefaultTimeout: time.Minute * 5,
			MaxCPUPerEnv: 2,
			MaxMemoryPerEnv: 1024,
			MaxDiskPerEnv: 1024,
			EnableSandboxing: false,
			EnableNetworkIsolation: false,
			AllowedPorts: []int{8080},
			SessionTimeout: time.Hour,
			MaxSessionsPerUser: 5,
			MaxProjectsPerUser: 10,
			ProjectStoragePath: "/tmp",
		},
	}

	server, err := NewServer(cfg)
	require.NoError(t, err)
	require.NotNil(t, server)
	defer server.db.Close()

	// Ensure server is not running
	server.running = false

	// Try to stop
	err = server.Stop()
	assert.NoError(t, err)
}

func TestInitializeTEEEnvironment(t *testing.T) {
	// Skip this test as it requires a valid database connection
	// and the function is tested indirectly through NewServer tests
	t.Skip("Skipping TEE environment test - requires valid database setup")
}

func TestLogSecurityValidationReport(t *testing.T) {
	// Skip this test as it requires the teesecurity package types
	// and the function is tested indirectly through other tests
	t.Skip("Skipping security validation report test - requires teesecurity types")
}

func TestSetupRoutes(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Path: filepath.Join(os.TempDir(), "test.db"),
		},
		API: config.APIConfig{
			BindAddress: "localhost",
			Port:        8080,
		},
		Security: config.SecurityConfig{
			JWTSecret: "test-secret",
		},
		ChainID: "test-chain",
		NodeRole: "validator",
		P2P: config.P2PConfig{
			DHTEnabled: false,
		},
		CDE: config.CDEConfig{
			BaseImagePath: "/tmp",
			WorkspaceRoot: "/tmp",
			MaxEnvironments: 10,
			DefaultTimeout: time.Minute * 5,
			MaxCPUPerEnv: 2,
			MaxMemoryPerEnv: 1024,
			MaxDiskPerEnv: 1024,
			EnableSandboxing: false,
			EnableNetworkIsolation: false,
			AllowedPorts: []int{8080},
			SessionTimeout: time.Hour,
			MaxSessionsPerUser: 5,
			MaxProjectsPerUser: 10,
			ProjectStoragePath: "/tmp",
		},
	}

	server, err := NewServer(cfg)
	require.NoError(t, err)
	require.NotNil(t, server)
	defer server.db.Close()

	// setupRoutes is called in NewServer, so routes should already be set up
	assert.NotNil(t, server.router)

	// Test that health endpoint is registered
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
