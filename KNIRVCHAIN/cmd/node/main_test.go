package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetEnv(t *testing.T) {
	// Test with existing env var
	os.Setenv("TEST_KEY", "test_value")
	defer os.Unsetenv("TEST_KEY")

	result := getEnv("TEST_KEY", "default")
	if result != "test_value" {
		t.Errorf("Expected 'test_value', got '%s'", result)
	}

	// Test with non-existing env var
	result = getEnv("NON_EXISTING_KEY", "default")
	if result != "default" {
		t.Errorf("Expected 'default', got '%s'", result)
	}
}

func TestLoadConfig(t *testing.T) {
	// Set test environment variables
	testEnv := map[string]string{
		"KNIRVBASE_NODE_ID":     "test-node",
		"KNIRVBASE_LISTEN_ADDR": "127.0.0.1:9999",
		"KNIRVBASE_DATA_DIR":    "/tmp/test-data",
		"KNIRVBASE_WALLET_KEY":  "test-key",
		"KNIRVBASE_NETWORK_URL": "https://test.network",
		"KNIRVBASE_CHAIN_ID":    "test-chain",
		"KNIRVBASE_LOG_LEVEL":   "debug",
	}

	// Set environment variables
	for key, value := range testEnv {
		os.Setenv(key, value)
		defer os.Unsetenv(key)
	}

	config := loadConfig()

	if config.NodeID != "test-node" {
		t.Errorf("Expected NodeID 'test-node', got '%s'", config.NodeID)
	}
	if config.ListenAddr != "127.0.0.1:9999" {
		t.Errorf("Expected ListenAddr '127.0.0.1:9999', got '%s'", config.ListenAddr)
	}
	if config.DataDir != "/tmp/test-data" {
		t.Errorf("Expected DataDir '/tmp/test-data', got '%s'", config.DataDir)
	}
	if config.WalletKey != "test-key" {
		t.Errorf("Expected WalletKey 'test-key', got '%s'", config.WalletKey)
	}
	if config.NetworkURL != "https://test.network" {
		t.Errorf("Expected NetworkURL 'https://test.network', got '%s'", config.NetworkURL)
	}
	if config.ChainID != "test-chain" {
		t.Errorf("Expected ChainID 'test-chain', got '%s'", config.ChainID)
	}
	if config.LogLevel != "debug" {
		t.Errorf("Expected LogLevel 'debug', got '%s'", config.LogLevel)
	}
}

func TestLoadConfigDefaults(t *testing.T) {
	// Clear any existing env vars
	envKeys := []string{
		"KNIRVBASE_NODE_ID",
		"KNIRVBASE_LISTEN_ADDR",
		"KNIRVBASE_DATA_DIR",
		"KNIRVBASE_WALLET_KEY",
		"KNIRVBASE_NETWORK_URL",
		"KNIRVBASE_CHAIN_ID",
		"KNIRVBASE_LOG_LEVEL",
	}

	for _, key := range envKeys {
		os.Unsetenv(key)
	}

	config := loadConfig()

	expected := &Config{
		NodeID:     "knirvbase-node-1",
		ListenAddr: "localhost:8080",
		DataDir:    filepath.Join(os.TempDir(), "knirvbase"),
		WalletKey:  "mock",
		NetworkURL: "https://api.xion.network",
		ChainID:    "xion-mainnet-1",
		LogLevel:   "info",
	}

	if config.NodeID != expected.NodeID {
		t.Errorf("Expected NodeID '%s', got '%s'", expected.NodeID, config.NodeID)
	}
	if config.ListenAddr != expected.ListenAddr {
		t.Errorf("Expected ListenAddr '%s', got '%s'", expected.ListenAddr, config.ListenAddr)
	}
	if config.DataDir != expected.DataDir {
		t.Errorf("Expected DataDir '%s', got '%s'", expected.DataDir, config.DataDir)
	}
	if config.WalletKey != expected.WalletKey {
		t.Errorf("Expected WalletKey '%s', got '%s'", expected.WalletKey, config.WalletKey)
	}
	if config.NetworkURL != expected.NetworkURL {
		t.Errorf("Expected NetworkURL '%s', got '%s'", expected.NetworkURL, config.NetworkURL)
	}
	if config.ChainID != expected.ChainID {
		t.Errorf("Expected ChainID '%s', got '%s'", expected.ChainID, config.ChainID)
	}
	if config.LogLevel != expected.LogLevel {
		t.Errorf("Expected LogLevel '%s', got '%s'", expected.LogLevel, config.LogLevel)
	}
}

func TestPrintBanner(t *testing.T) {
	// This test ensures printBanner doesn't panic and produces output
	// We can't easily capture stdout in a simple test, but we can ensure it doesn't crash
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("printBanner panicked: %v", r)
		}
	}()
	printBanner()
}
