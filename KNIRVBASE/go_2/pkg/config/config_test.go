package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-config.yaml")

	configContent := `
node:
  id: "test-node-1"
  role: "validator"
  listen_addr: "127.0.0.1:8080"
  rpc_endpoint: "http://localhost:8545"

blockchain:
  data_dir: "/tmp/knirvchain"
  max_block_size: 1048576
  block_time: "5s"
  consensus: "poa"

wallet:
  private_key_path: "/path/to/key"
  contract_address: "0x1234567890123456789012345678901234567890"
  network_url: "https://testnet.xion.network"

cache:
  redis_url: "redis://localhost:6379"
  ttl: "1h"
  max_connections: 10

indexing:
  semantic:
    enabled: true
    dimension: 768
    hnsw_m: 16
    hnsw_ef_construction: 200
  temporal:
    enabled: true
  category:
    enabled: true
  fulltext:
    enabled: false

security:
  tls:
    enabled: true
    cert_file: "/path/to/cert.pem"
    key_file: "/path/to/key.pem"
  jwt:
    secret: "test-jwt-secret"
    token_duration: "1h"
  encryption:
    enabled: true
    key_rotation_days: 30

logging:
  level: "info"
  format: "json"
  output: "stdout"

monitoring:
  prometheus:
    enabled: true
    path: "/metrics"
  tracing:
    enabled: false
    jaeger_endpoint: "http://localhost:14268/api/traces"

performance:
  max_goroutines: 100
  request_timeout: "30s"
  write_buffer_size: 4096
  read_buffer_size: 4096
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Test node config
	if config.Node.ID != "test-node-1" {
		t.Errorf("Expected Node.ID 'test-node-1', got '%s'", config.Node.ID)
	}
	if config.Node.Role != "validator" {
		t.Errorf("Expected Node.Role 'validator', got '%s'", config.Node.Role)
	}
	if config.Node.ListenAddr != "127.0.0.1:8080" {
		t.Errorf("Expected Node.ListenAddr '127.0.0.1:8080', got '%s'", config.Node.ListenAddr)
	}

	// Test blockchain config
	if config.Blockchain.DataDir != "/tmp/knirvchain" {
		t.Errorf("Expected Blockchain.DataDir '/tmp/knirvchain', got '%s'", config.Blockchain.DataDir)
	}
	if config.Blockchain.MaxBlockSize != 1048576 {
		t.Errorf("Expected Blockchain.MaxBlockSize 1048576, got %d", config.Blockchain.MaxBlockSize)
	}

	// Test wallet config
	if config.Wallet.ContractAddr != "0x1234567890123456789012345678901234567890" {
		t.Errorf("Expected Wallet.ContractAddr '0x1234567890123456789012345678901234567890', got '%s'", config.Wallet.ContractAddr)
	}

	// Test cache config
	if config.Cache.RedisURL != "redis://localhost:6379" {
		t.Errorf("Expected Cache.RedisURL 'redis://localhost:6379', got '%s'", config.Cache.RedisURL)
	}

	// Test indexing config
	if !config.Indexing.Semantic.Enabled {
		t.Error("Expected Indexing.Semantic.Enabled to be true")
	}
	if config.Indexing.Semantic.Dimension != 768 {
		t.Errorf("Expected Indexing.Semantic.Dimension 768, got %d", config.Indexing.Semantic.Dimension)
	}

	// Test security config
	if !config.Security.TLS.Enabled {
		t.Error("Expected Security.TLS.Enabled to be true")
	}
	if config.Security.JWT.Secret != "test-jwt-secret" {
		t.Errorf("Expected Security.JWT.Secret 'test-jwt-secret', got '%s'", config.Security.JWT.Secret)
	}

	// Test logging config
	if config.Logging.Level != "info" {
		t.Errorf("Expected Logging.Level 'info', got '%s'", config.Logging.Level)
	}

	// Test monitoring config
	if !config.Monitoring.Prometheus.Enabled {
		t.Error("Expected Monitoring.Prometheus.Enabled to be true")
	}
	if config.Monitoring.Prometheus.Path != "/metrics" {
		t.Errorf("Expected Monitoring.Prometheus.Path '/metrics', got '%s'", config.Monitoring.Prometheus.Path)
	}

	// Test performance config
	if config.Performance.MaxGoroutines != 100 {
		t.Errorf("Expected Performance.MaxGoroutines 100, got %d", config.Performance.MaxGoroutines)
	}
}

func TestLoadConfigNonExistentFile(t *testing.T) {
	_, err := LoadConfig("/non/existent/config.yaml")
	if err == nil {
		t.Error("Expected error for non-existent config file")
	}
}

func TestLoadConfigInvalidYAML(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "invalid-config.yaml")

	invalidContent := `
invalid: yaml: content:
  - with
    bad: syntax
    [
`

	err := os.WriteFile(configPath, []byte(invalidContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid config file: %v", err)
	}

	_, err = LoadConfig(configPath)
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
}