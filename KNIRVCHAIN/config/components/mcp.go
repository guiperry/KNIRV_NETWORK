package components

import (
	"fmt"
	"net/url"
)

// MCPConfig represents MCP (Model Context Protocol) component configuration
type MCPConfig struct {
	Enabled     bool   `json:"enabled" mapstructure:"enabled"`
	Port        int    `json:"port" mapstructure:"port"`
	Host        string `json:"host" mapstructure:"host"`
	MaxClients  int    `json:"max_clients" mapstructure:"max_clients"`
	Timeout     int    `json:"timeout" mapstructure:"timeout"`
	LogLevel    string `json:"log_level" mapstructure:"log_level"`
	TLSEnabled  bool   `json:"tls_enabled" mapstructure:"tls_enabled"`
	CertFile    string `json:"cert_file" mapstructure:"cert_file"`
	KeyFile     string `json:"key_file" mapstructure:"key_file"`
}

// GetConfigKey returns the configuration key prefix for MCP
func (c *MCPConfig) GetConfigKey() string {
	return "mcp"
}

// Validate validates the MCP configuration
func (c *MCPConfig) Validate() error {
	if c.Enabled {
		if c.Port <= 0 || c.Port > 65535 {
			return fmt.Errorf("invalid MCP port: %d (must be 1-65535)", c.Port)
		}
		
		if c.Host == "" {
			return fmt.Errorf("MCP host cannot be empty when enabled")
		}
		
		// Validate host format
		if _, err := url.Parse(fmt.Sprintf("http://%s", c.Host)); err != nil {
			return fmt.Errorf("invalid MCP host format: %s", c.Host)
		}
		
		if c.MaxClients <= 0 {
			return fmt.Errorf("MCP max_clients must be positive, got: %d", c.MaxClients)
		}
		
		if c.Timeout <= 0 {
			return fmt.Errorf("MCP timeout must be positive, got: %d", c.Timeout)
		}
		
		// Validate log level
		validLogLevels := map[string]bool{
			"debug": true, "info": true, "warn": true, "error": true, "fatal": true,
		}
		if !validLogLevels[c.LogLevel] {
			return fmt.Errorf("invalid MCP log level: %s (must be debug, info, warn, error, or fatal)", c.LogLevel)
		}
		
		// Validate TLS configuration
		if c.TLSEnabled {
			if c.CertFile == "" {
				return fmt.Errorf("MCP cert_file required when TLS is enabled")
			}
			if c.KeyFile == "" {
				return fmt.Errorf("MCP key_file required when TLS is enabled")
			}
		}
	}
	
	return nil
}

// GetDefaults returns default configuration values for MCP
func (c *MCPConfig) GetDefaults() map[string]interface{} {
	return map[string]interface{}{
		"enabled":     false,
		"port":        8080,
		"host":        "localhost",
		"max_clients": 100,
		"timeout":     30,
		"log_level":   "info",
		"tls_enabled": false,
		"cert_file":   "",
		"key_file":    "",
	}
}

// GetEnvironmentMappings returns environment variable mappings for MCP
func (c *MCPConfig) GetEnvironmentMappings() map[string]string {
	return map[string]string{
		"mcp.enabled":     "KNIRV_MCP_ENABLED",
		"mcp.port":        "KNIRV_MCP_PORT",
		"mcp.host":        "KNIRV_MCP_HOST",
		"mcp.max_clients": "KNIRV_MCP_MAX_CLIENTS",
		"mcp.timeout":     "KNIRV_MCP_TIMEOUT",
		"mcp.log_level":   "KNIRV_MCP_LOG_LEVEL",
		"mcp.tls_enabled": "KNIRV_MCP_TLS_ENABLED",
		"mcp.cert_file":   "KNIRV_MCP_CERT_FILE",
		"mcp.key_file":    "KNIRV_MCP_KEY_FILE",
	}
}

// NewMCPConfig creates a new MCP configuration with defaults
func NewMCPConfig() *MCPConfig {
	return &MCPConfig{
		Enabled:     false,
		Port:        8080,
		Host:        "localhost",
		MaxClients:  100,
		Timeout:     30,
		LogLevel:    "info",
		TLSEnabled:  false,
		CertFile:    "",
		KeyFile:     "",
	}
}

// GetEndpoint returns the MCP endpoint URL
func (c *MCPConfig) GetEndpoint() string {
	scheme := "http"
	if c.TLSEnabled {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s:%d", scheme, c.Host, c.Port)
}

// IsSecure returns true if TLS is enabled
func (c *MCPConfig) IsSecure() bool {
	return c.TLSEnabled
}

// GetMaxClients returns the maximum number of concurrent clients
func (c *MCPConfig) GetMaxClients() int {
	return c.MaxClients
}

// GetTimeoutSeconds returns the timeout in seconds
func (c *MCPConfig) GetTimeoutSeconds() int {
	return c.Timeout
}
