package objectserver

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"backend_server/internal/config"
	"backend_server/internal/database"
)

func TestNewModelServer(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "model_server_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test config and database
	cfg := &config.Config{}
	db := &database.BuntDBManager{}

	// Test successful creation
	server, err := NewModelServer(cfg, db)
	if err != nil {
		t.Fatalf("Failed to create model server: %v", err)
	}

	if server == nil {
		t.Fatal("Model server is nil")
	}

	if server.config != cfg {
		t.Error("Config not set correctly")
	}

	if server.db != db {
		t.Error("Database not set correctly")
	}

	if server.modelDir != "./models" {
		t.Error("Model directory not set correctly")
	}

	if server.maxModels != 10 {
		t.Error("Max models not set correctly")
	}

	if !server.enableRuntime {
		t.Error("Runtime should be enabled by default")
	}

	if !server.enableCORS {
		t.Error("CORS should be enabled by default")
	}

	if server.serverInfo == nil {
		t.Error("Server info should not be nil")
	}

	if server.serverInfo.Name != "KNIRV-NEXUS Plugin Model Server" {
		t.Error("Server name not set correctly")
	}

	if server.serverInfo.Version != "2.0.0" {
		t.Error("Server version not set correctly")
	}

	if server.running {
		t.Error("Server should not be running initially")
	}
}

func TestModelServer_Start(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "model_server_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test config and database
	cfg := &config.Config{}
	db := &database.BuntDBManager{}

	server, err := NewModelServer(cfg, db)
	if err != nil {
		t.Fatalf("Failed to create model server: %v", err)
	}

	// Test starting the server
	err = server.Start()
	if err != nil {
		t.Fatalf("Failed to start model server: %v", err)
	}

	if !server.IsRunning() {
		t.Error("Server should be running after start")
	}

	if server.runtimeManager == nil {
		t.Error("Runtime manager should be initialized")
	}

	// Test stopping the server
	err = server.Stop()
	if err != nil {
		t.Fatalf("Failed to stop model server: %v", err)
	}

	if server.IsRunning() {
		t.Error("Server should not be running after stop")
	}
}

func TestModelServer_Stop(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "model_server_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test config and database
	cfg := &config.Config{}
	db := &database.BuntDBManager{}

	server, err := NewModelServer(cfg, db)
	if err != nil {
		t.Fatalf("Failed to create model server: %v", err)
	}

	// Start the server first
	err = server.Start()
	if err != nil {
		t.Fatalf("Failed to start model server: %v", err)
	}

	// Test stopping the server
	err = server.Stop()
	if err != nil {
		t.Fatalf("Failed to stop model server: %v", err)
	}

	if server.IsRunning() {
		t.Error("Server should not be running after stop")
	}
}

func TestModelServer_IsRunning(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "model_server_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test config and database
	cfg := &config.Config{}
	db := &database.BuntDBManager{}

	server, err := NewModelServer(cfg, db)
	if err != nil {
		t.Fatalf("Failed to create model server: %v", err)
	}

	// Initially should not be running
	if server.IsRunning() {
		t.Error("Server should not be running initially")
	}

	// Start the server
	err = server.Start()
	if err != nil {
		t.Fatalf("Failed to start model server: %v", err)
	}

	// Should be running now
	if !server.IsRunning() {
		t.Error("Server should be running after start")
	}

	// Stop the server
	err = server.Stop()
	if err != nil {
		t.Fatalf("Failed to stop model server: %v", err)
	}

	// Should not be running anymore
	if server.IsRunning() {
		t.Error("Server should not be running after stop")
	}
}

func TestModelServer_GetServerInfo(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "model_server_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test config and database
	cfg := &config.Config{}
	db := &database.BuntDBManager{}

	server, err := NewModelServer(cfg, db)
	if err != nil {
		t.Fatalf("Failed to create model server: %v", err)
	}

	info := server.GetServerInfo()
	if info == nil {
		t.Fatal("Server info should not be nil")
	}

	if info.Name != "KNIRV-NEXUS Plugin Model Server" {
		t.Error("Server name not set correctly")
	}

	if info.Version != "2.0.0" {
		t.Error("Server version not set correctly")
	}

	if info.ModelDir != "./models" {
		t.Error("Model directory not set correctly")
	}
}

func TestModelServer_GetRuntimeManager(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "model_server_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test config and database
	cfg := &config.Config{}
	db := &database.BuntDBManager{}

	server, err := NewModelServer(cfg, db)
	if err != nil {
		t.Fatalf("Failed to create model server: %v", err)
	}

	// Runtime manager should be nil before starting
	if server.GetRuntimeManager() != nil {
		t.Error("Runtime manager should be nil before starting")
	}

	// Start the server
	err = server.Start()
	if err != nil {
		t.Fatalf("Failed to start model server: %v", err)
	}

	// Runtime manager should be initialized after starting
	if server.GetRuntimeManager() == nil {
		t.Error("Runtime manager should be initialized after starting")
	}
}

func TestEnsureModelDirectory(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "model_server_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testDir := filepath.Join(tempDir, "test_models")

	// Test creating a new directory
	err = ensureModelDirectory(testDir)
	if err != nil {
		t.Fatalf("Failed to create model directory: %v", err)
	}

	// Check if directory exists
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		t.Error("Model directory was not created")
	}

	// Test creating the same directory again (should not error)
	err = ensureModelDirectory(testDir)
	if err != nil {
		t.Fatalf("Failed to ensure existing model directory: %v", err)
	}
}

func TestModelServer_ServerInfo(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "model_server_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test config and database
	cfg := &config.Config{}
	db := &database.BuntDBManager{}

	server, err := NewModelServer(cfg, db)
	if err != nil {
		t.Fatalf("Failed to create model server: %v", err)
	}

	info := server.GetServerInfo()

	// Test that start time is set
	if info.StartTime.IsZero() {
		t.Error("Start time should be set")
	}

	// Test that port is initially 0
	if info.Port != 0 {
		t.Error("Port should be 0 initially")
	}
}

func TestModelServer_Configuration(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "model_server_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test config and database
	cfg := &config.Config{}
	db := &database.BuntDBManager{}

	server, err := NewModelServer(cfg, db)
	if err != nil {
		t.Fatalf("Failed to create model server: %v", err)
	}

	// Test default configuration values
	if server.modelDir != "./models" {
		t.Errorf("Expected modelDir './models', got '%s'", server.modelDir)
	}

	if server.maxModels != 10 {
		t.Errorf("Expected maxModels 10, got %d", server.maxModels)
	}

	if !server.enableRuntime {
		t.Error("Expected enableRuntime to be true")
	}

	if !server.enableCORS {
		t.Error("Expected enableCORS to be true")
	}
}

func TestModelServer_StartTime(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "model_server_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test config and database
	cfg := &config.Config{}
	db := &database.BuntDBManager{}

	before := time.Now()
	server, err := NewModelServer(cfg, db)
	if err != nil {
		t.Fatalf("Failed to create model server: %v", err)
	}
	after := time.Now()

	// Check that start time is within reasonable bounds
	if server.startTime.Before(before.Add(-time.Second)) || server.startTime.After(after.Add(time.Second)) {
		t.Error("Start time is not reasonable")
	}
}