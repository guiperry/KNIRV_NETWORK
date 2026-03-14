package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	// Test loading default configuration
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg == nil {
		t.Fatal("Config is nil")
	}

	// Test default values
	if cfg.Mode != "headless" {
		t.Errorf("Expected mode 'headless', got '%s'", cfg.Mode)
	}

	if cfg.API.Port != 8082 {
		t.Errorf("Expected API port 8082, got %d", cfg.API.Port)
	}

	if cfg.Database.Path == "" {
		t.Error("Database path should not be empty")
	}
}

func TestExpandPath(t *testing.T) {
	// Test expanding home directory
	home, err := expandPath("~/test")
	if err != nil {
		t.Fatalf("Failed to expand path: %v", err)
	}

	if home == "~/test" {
		t.Error("Path was not expanded")
	}

	// Test non-tilde path
	normal, err := expandPath("/tmp/test")
	if err != nil {
		t.Fatalf("Failed to expand path: %v", err)
	}

	if normal != "/tmp/test" {
		t.Errorf("Expected '/tmp/test', got '%s'", normal)
	}
}

func TestValidateConfig(t *testing.T) {
	// Test valid config
	cfg := &Config{
		Mode:     "headless",
		ChainID:  "test-chain",
		NodeRole: "test-role",
		Database: DatabaseConfig{Path: "/tmp/test.db"},
	}

	err := validateConfig(cfg)
	if err != nil {
		t.Errorf("Valid config should not error: %v", err)
	}

	// Test invalid mode
	cfg.Mode = "invalid"
	err = validateConfig(cfg)
	if err == nil {
		t.Error("Invalid mode should error")
	}
	cfg.Mode = "headless" // Reset

	// Test missing chain ID
	cfg.ChainID = ""
	cfg.Network.ChainID = ""
	err = validateConfig(cfg)
	if err == nil {
		t.Error("Missing chain ID should error")
	}
	cfg.ChainID = "test-chain" // Reset
}

func TestSetDefaults(t *testing.T) {
	// Clear any existing viper settings
	setDefaults()

	// Check some defaults are set
	if mode := os.Getenv("KNIRV_MODE"); mode == "" {
		// Environment variable not set, but viper should have defaults
		// This is hard to test directly, but we can check the function doesn't panic
	}
}

func TestGetAppDataDir(t *testing.T) {
	dir, err := getAppDataDir()
	if err != nil {
		t.Fatalf("Failed to get app data dir: %v", err)
	}

	if dir == "" {
		t.Error("App data dir should not be empty")
	}

	// Check if directory contains expected path
	expected := filepath.Join(".local", "share", "knirvnexus", "backend_server")
	if !strings.HasSuffix(dir, expected) {
		t.Errorf("Expected path to end with '%s', got '%s'", expected, dir)
	}
}

func TestConfigExpandPaths(t *testing.T) {
	// Create temporary directories for testing
	tempDir := t.TempDir()

	cfg := &Config{
		Database: DatabaseConfig{Path: filepath.Join(tempDir, "test.db")},
		CDE: CDEConfig{
			BaseImagePath:     filepath.Join(tempDir, "images"),
			WorkspaceRoot:     filepath.Join(tempDir, "workspaces"),
			ProjectStoragePath: filepath.Join(tempDir, "projects"),
		},
		Reports: ReportsConfig{StoragePath: filepath.Join(tempDir, "reports")},
		Log:     LogConfig{Output: filepath.Join(tempDir, "logs", "nexus.log")},
	}

	err := cfg.expandPaths()
	if err != nil {
		t.Fatalf("Failed to expand paths: %v", err)
	}

	// Check paths were expanded (should remain the same since they don't start with ~)
	if cfg.Database.Path != filepath.Join(tempDir, "test.db") {
		t.Errorf("Database path was unexpectedly changed: %s", cfg.Database.Path)
	}

	if cfg.CDE.BaseImagePath != filepath.Join(tempDir, "images") {
		t.Errorf("CDE base image path was unexpectedly changed: %s", cfg.CDE.BaseImagePath)
	}

	// Verify directories were created
	if _, err := os.Stat(cfg.CDE.BaseImagePath); os.IsNotExist(err) {
		t.Error("Base image directory was not created")
	}
}
