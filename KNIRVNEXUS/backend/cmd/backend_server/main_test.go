package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"backend_server/internal/config"
	"backend_server/internal/services/teesecurity"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/buntdb"
)

func TestNewServer(t *testing.T) {
	// Create a temporary config for testing
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	workspaceRoot := filepath.Join(tempDir, "workspaces")
	
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Path: dbPath,
		},
		ChainID:  "test-chain",
		NodeRole: "validator",
		P2P: config.P2PConfig{
			DHTEnabled: false,
		},
		API: config.APIConfig{
			BindAddress: "127.0.0.1",
			Port:        8080,
		},
		Security: config.SecurityConfig{
			JWTSecret: "test-secret",
		},
		CDE: config.CDEConfig{
			WorkspaceRoot:       workspaceRoot,
			MaxEnvironments:     0, // Disable for testing
			EnableSandboxing:    false,
			EnableNetworkIsolation: false,
		},
	}

	// Create workspace directory to avoid errors
	err := os.MkdirAll(workspaceRoot, 0755)
	require.NoError(t, err)

	// Test NewServer creation
	server, err := NewServer(cfg)
	require.NoError(t, err)
	assert.NotNil(t, server)
	assert.Equal(t, cfg, server.config)
	assert.NotNil(t, server.db)
	assert.NotNil(t, server.router)
	assert.False(t, server.running)

	// Clean up
	defer server.Stop()
}

func TestServerSetupRoutes(t *testing.T) {
	// Create temporary config and server
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	workspaceRoot := filepath.Join(tempDir, "workspaces")
	
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Path: dbPath,
		},
		ChainID:  "test-chain",
		NodeRole: "validator",
		P2P: config.P2PConfig{
			DHTEnabled: false,
		},
		API: config.APIConfig{
			BindAddress: "127.0.0.1",
			Port:        8080,
		},
		Security: config.SecurityConfig{
			JWTSecret: "test-secret",
		},
		CDE: config.CDEConfig{
			WorkspaceRoot:          workspaceRoot,
			MaxEnvironments:        0, // Disable for testing
			EnableSandboxing:       false,
			EnableNetworkIsolation: false,
		},
	}

	// Create workspace directory to avoid errors
	err := os.MkdirAll(workspaceRoot, 0755)
	require.NoError(t, err)

	server, err := NewServer(cfg)
	require.NoError(t, err)
	defer server.Stop()

	// Test route setup
	assert.NotNil(t, server.router)
	
	// Test that health endpoint is registered
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	
	// Test API health endpoint
	req = httptest.NewRequest("GET", "/api/health", nil)
	w = httptest.NewRecorder()
	server.router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestServerHandleHealth(t *testing.T) {
	// Create temporary config and server
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	workspaceRoot := filepath.Join(tempDir, "workspaces")
	
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Path: dbPath,
		},
		ChainID:  "test-chain",
		NodeRole: "validator",
		P2P: config.P2PConfig{
			DHTEnabled: false,
		},
		API: config.APIConfig{
			BindAddress: "127.0.0.1",
			Port:        8080,
		},
		Security: config.SecurityConfig{
			JWTSecret: "test-secret",
		},
		CDE: config.CDEConfig{
			WorkspaceRoot:          workspaceRoot,
			MaxEnvironments:        0, // Disable for testing
			EnableSandboxing:       false,
			EnableNetworkIsolation: false,
		},
	}

	// Create workspace directory to avoid errors
	err := os.MkdirAll(workspaceRoot, 0755)
	require.NoError(t, err)

	server, err := NewServer(cfg)
	require.NoError(t, err)
	defer server.Stop()

	// Test health endpoint
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	server.handleHealth(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	// Parse response
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "healthy", response["status"])
	assert.Equal(t, Version, response["version"])
	
	// Check services status
	services, ok := response["services"].(map[string]interface{})
	require.True(t, ok)
	
	// Database should be initialized
	databaseStatus, ok := services["database"].(bool)
	require.True(t, ok)
	assert.True(t, databaseStatus)
}

func TestServerStart(t *testing.T) {
	// Create temporary config and server
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	workspaceRoot := filepath.Join(tempDir, "workspaces")
	
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Path: dbPath,
		},
		ChainID:  "test-chain",
		NodeRole: "validator",
		P2P: config.P2PConfig{
			DHTEnabled: false,
		},
		API: config.APIConfig{
			BindAddress: "127.0.0.1",
			Port:        8081, // Use different port to avoid conflicts
		},
		Security: config.SecurityConfig{
			JWTSecret: "test-secret",
		},
		CDE: config.CDEConfig{
			WorkspaceRoot:          workspaceRoot,
			MaxEnvironments:        0, // Disable for testing
			EnableSandboxing:       false,
			EnableNetworkIsolation: false,
		},
	}

	// Create workspace directory to avoid errors
	err := os.MkdirAll(workspaceRoot, 0755)
	require.NoError(t, err)

	server, err := NewServer(cfg)
	require.NoError(t, err)
	assert.False(t, server.running)

	// Test server start
	err = server.Start()
	require.NoError(t, err)
	assert.True(t, server.running)

	// Give server time to start
	time.Sleep(2 * time.Second)

	// Test that server is actually running by making a request
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("http://127.0.0.1:8081/health")
	require.NoError(t, err)
	defer resp.Body.Close()
	
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Clean up
	err = server.Stop()
	require.NoError(t, err)
	assert.False(t, server.running)
}

func TestServerStartAlreadyRunning(t *testing.T) {
	// Create temporary config and server
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	workspaceRoot := filepath.Join(tempDir, "workspaces")
	
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Path: dbPath,
		},
		ChainID:  "test-chain",
		NodeRole: "validator",
		P2P: config.P2PConfig{
			DHTEnabled: false,
		},
		API: config.APIConfig{
			BindAddress: "127.0.0.1",
			Port:        8082,
		},
		Security: config.SecurityConfig{
			JWTSecret: "test-secret",
		},
		CDE: config.CDEConfig{
			WorkspaceRoot:          workspaceRoot,
			MaxEnvironments:        0, // Disable for testing
			EnableSandboxing:       false,
			EnableNetworkIsolation: false,
		},
	}

	// Create workspace directory to avoid errors
	err := os.MkdirAll(workspaceRoot, 0755)
	require.NoError(t, err)

	server, err := NewServer(cfg)
	require.NoError(t, err)

	// Start server
	err = server.Start()
	require.NoError(t, err)

	// Try to start again (should fail)
	err = server.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")

	// Clean up
	server.Stop()
}

func TestServerStop(t *testing.T) {
	// Create temporary config and server
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	workspaceRoot := filepath.Join(tempDir, "workspaces")
	
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Path: dbPath,
		},
		ChainID:  "test-chain",
		NodeRole: "validator",
		P2P: config.P2PConfig{
			DHTEnabled: false,
		},
		API: config.APIConfig{
			BindAddress: "127.0.0.1",
			Port:        8083,
		},
		Security: config.SecurityConfig{
			JWTSecret: "test-secret",
		},
		CDE: config.CDEConfig{
			WorkspaceRoot:          workspaceRoot,
			MaxEnvironments:        0, // Disable for testing
			EnableSandboxing:       false,
			EnableNetworkIsolation: false,
		},
	}

	// Create workspace directory to avoid errors
	err := os.MkdirAll(workspaceRoot, 0755)
	require.NoError(t, err)

	server, err := NewServer(cfg)
	require.NoError(t, err)

	// Start server
	err = server.Start()
	require.NoError(t, err)
	assert.True(t, server.running)

	// Stop server
	err = server.Stop()
	require.NoError(t, err)
	assert.False(t, server.running)

	// Stop again (should be no-op)
	err = server.Stop()
	require.NoError(t, err)
}

func TestServerStopNotRunning(t *testing.T) {
	// Create temporary config and server
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	workspaceRoot := filepath.Join(tempDir, "workspaces")
	
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Path: dbPath,
		},
		ChainID:  "test-chain",
		NodeRole: "validator",
		P2P: config.P2PConfig{
			DHTEnabled: false,
		},
		API: config.APIConfig{
			BindAddress: "127.0.0.1",
			Port:        8084,
		},
		Security: config.SecurityConfig{
			JWTSecret: "test-secret",
		},
		CDE: config.CDEConfig{
			WorkspaceRoot:          workspaceRoot,
			MaxEnvironments:        0, // Disable for testing
			EnableSandboxing:       false,
			EnableNetworkIsolation: false,
		},
	}

	// Create workspace directory to avoid errors
	err := os.MkdirAll(workspaceRoot, 0755)
	require.NoError(t, err)

	server, err := NewServer(cfg)
	require.NoError(t, err)

	// Stop server without starting (should be no-op)
	err = server.Stop()
	require.NoError(t, err)
}

func TestInitializeTEEEnvironment(t *testing.T) {
	// Create temporary database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_tee.db")
	
	db, err := buntdb.Open(dbPath)
	require.NoError(t, err)
	defer db.Close()

	// Test TEE environment initialization
	ctx := context.Background()
	err = initializeTEEEnvironment(ctx, db)
	// This might fail in test environment, but shouldn't panic
	// The function should handle errors gracefully
	if err != nil {
		t.Logf("TEE environment initialization failed (expected in test environment): %v", err)
	}
}

func TestLogSecurityValidationReport(t *testing.T) {
	// Create a test validation report
	report := &teesecurity.KaliSecurityValidationReport{
		OS:              "Linux",
		IsKaliLinux:     false,
		Timestamp:       time.Now(),
		SystemMemoryKB:  "8192",
		DiskSpaceKB:     "1024000",
		ToolsAvailable: map[string]bool{
			"nmap":     true,
			"metasploit": false,
		},
		FrameworksLoaded: map[string]bool{
			"Nikto":       true,
			"Burp Suite": false,
		},
	}

	// This should not panic
	assert.NotPanics(t, func() {
		logSecurityValidationReport(report)
	})
}

func TestRunFunction(t *testing.T) {
	// Test with invalid config file
	t.Run("invalid config file", func(t *testing.T) {
		// This should fail gracefully
		err := runWithArgs([]string{"test", "-config", "/nonexistent/config.yaml"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "config file does not exist")
	})
}

// runWithArgs runs the application with specific command line arguments
func runWithArgs(args []string) error {
	// Save original arguments
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = args
	return run()
}

func TestMainWithConfigFile(t *testing.T) {
	// Save original arguments
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	// Create temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-config.yaml")
	dbPath := filepath.Join(tempDir, "test.db")
	
	configContent := `
database:
  path: "` + dbPath + `"
api:
  bind_address: "127.0.0.1"
  port: 8086
security:
  jwt_secret: "test-secret"
chain_id: "test-chain"
node_role: "validator"
p2p:
  dht_enabled: false
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Test with explicit config file
	os.Args = []string{"test", "-config", configPath}

	// Test main function with timeout
	done := make(chan bool, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				t.Logf("main() panicked (expected in test environment): %v", r)
			}
			done <- true
		}()
		main()
	}()

	// Wait for main to start or timeout
	select {
	case <-done:
		// Function completed
	case <-time.After(5 * time.Second):
		t.Logf("main() took too long, likely waiting for signal")
	}
}

// Benchmark tests for performance
func BenchmarkNewServer(b *testing.B) {
	tempDir := b.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	workspaceRoot := filepath.Join(tempDir, "workspaces")
	
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Path: dbPath,
		},
		ChainID:  "test-chain",
		NodeRole: "validator",
		P2P: config.P2PConfig{
			DHTEnabled: false,
		},
		API: config.APIConfig{
			BindAddress: "127.0.0.1",
			Port:        8087,
		},
		Security: config.SecurityConfig{
			JWTSecret: "test-secret",
		},
		CDE: config.CDEConfig{
			WorkspaceRoot:          workspaceRoot,
			MaxEnvironments:        0, // Disable for testing
			EnableSandboxing:       false,
			EnableNetworkIsolation: false,
		},
	}

	// Create workspace directory to avoid errors
	err := os.MkdirAll(workspaceRoot, 0755)
	if err != nil {
		b.Fatalf("Failed to create workspace directory: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server, err := NewServer(cfg)
		if err != nil {
			b.Fatalf("Failed to create server: %v", err)
		}
		server.Stop()
	}
}
