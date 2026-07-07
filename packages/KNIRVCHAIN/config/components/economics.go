package components

import (
	"fmt"
	"net/url"
	"strconv"
)

// EconomicsConfig represents economics service component configuration
type EconomicsConfig struct {
	Enabled       bool    `json:"enabled" mapstructure:"enabled"`
	Port          string  `json:"port" mapstructure:"port"`
	NRNContract   string  `json:"nrn_contract" mapstructure:"nrn_contract"`
	XionRPC       string  `json:"xion_rpc" mapstructure:"xion_rpc"`
	LogLevel      string  `json:"log_level" mapstructure:"log_level"`
	KNIRVChainURL string  `json:"knirvchain_url" mapstructure:"knirvchain_url"`
	KNIRVNexusURL string  `json:"knirvnexus_url" mapstructure:"knirvnexus_url"`
	KNIRVRootURL  string  `json:"knirvoracle_url" mapstructure:"knirvoracle_url"`
	KNIRVGraphURL string  `json:"knirvgraph_url" mapstructure:"knirvgraph_url"`
	DatabasePath  string  `json:"database_path" mapstructure:"database_path"`
	TokenSymbol   string  `json:"token_symbol" mapstructure:"token_symbol"`
	TokenDecimals int     `json:"token_decimals" mapstructure:"token_decimals"`
	USDPerToken   float64 `json:"usd_per_token" mapstructure:"usd_per_token"`
	ETHPerToken   float64 `json:"eth_per_token" mapstructure:"eth_per_token"`
}

// GetConfigKey returns the configuration key prefix for economics
func (c *EconomicsConfig) GetConfigKey() string {
	return "economics"
}

// Validate validates the economics configuration
func (c *EconomicsConfig) Validate() error {
	if c.Enabled {
		// Validate port
		if c.Port == "" {
			return fmt.Errorf("economics port cannot be empty when enabled")
		}

		if port, err := strconv.Atoi(c.Port); err != nil {
			return fmt.Errorf("invalid economics port format: %s", c.Port)
		} else if port <= 0 || port > 65535 {
			return fmt.Errorf("invalid economics port: %d (must be 1-65535)", port)
		}

		// Validate NRN contract address
		if c.NRNContract == "" {
			return fmt.Errorf("economics NRN contract cannot be empty when enabled")
		}

		// Validate Xion RPC URL
		if c.XionRPC == "" {
			return fmt.Errorf("economics Xion RPC URL cannot be empty when enabled")
		}

		if _, err := url.Parse(c.XionRPC); err != nil {
			return fmt.Errorf("invalid economics Xion RPC URL: %s", c.XionRPC)
		}

		// Validate log level
		validLogLevels := map[string]bool{
			"debug": true, "info": true, "warn": true, "error": true, "fatal": true,
		}
		if !validLogLevels[c.LogLevel] {
			return fmt.Errorf("invalid economics log level: %s (must be debug, info, warn, error, or fatal)", c.LogLevel)
		}

		// Validate component URLs if provided
		if c.KNIRVChainURL != "" {
			if _, err := url.Parse(c.KNIRVChainURL); err != nil {
				return fmt.Errorf("invalid economics KNIRVChain URL: %s", c.KNIRVChainURL)
			}
		}

		if c.KNIRVNexusURL != "" {
			if _, err := url.Parse(c.KNIRVNexusURL); err != nil {
				return fmt.Errorf("invalid economics KNIRVNexus URL: %s", c.KNIRVNexusURL)
			}
		}

		if c.KNIRVRootURL != "" {
			if _, err := url.Parse(c.KNIRVRootURL); err != nil {
				return fmt.Errorf("invalid economics KNIRVRoot URL: %s", c.KNIRVRootURL)
			}
		}

		if c.KNIRVGraphURL != "" {
			if _, err := url.Parse(c.KNIRVGraphURL); err != nil {
				return fmt.Errorf("invalid economics KNIRVGraph URL: %s", c.KNIRVGraphURL)
			}
		}

		// Validate token configuration
		if c.TokenDecimals < 0 || c.TokenDecimals > 18 {
			return fmt.Errorf("invalid economics token decimals: %d (must be 0-18)", c.TokenDecimals)
		}

		if c.USDPerToken < 0 {
			return fmt.Errorf("economics USD per token cannot be negative: %f", c.USDPerToken)
		}

		if c.ETHPerToken < 0 {
			return fmt.Errorf("economics ETH per token cannot be negative: %f", c.ETHPerToken)
		}
	}

	return nil
}

// GetDefaults returns default configuration values for economics
func (c *EconomicsConfig) GetDefaults() map[string]interface{} {
	return map[string]interface{}{
		"enabled":         false,
		"port":            "8090",
		"nrn_contract":    "",
		"xion_rpc":        "",
		"log_level":       "info",
		"knirvchain_url":  "",
		"knirvnexus_url":  "",
		"knirvoracle_url": "",
		"knirvgraph_url":  "",
		"database_path":   "",
		"token_symbol":    "NRN",
		"token_decimals":  18,
		"usd_per_token":   0.0,
		"eth_per_token":   0.0,
	}
}

// GetEnvironmentMappings returns environment variable mappings for economics
func (c *EconomicsConfig) GetEnvironmentMappings() map[string]string {
	return map[string]string{
		"economics.enabled":         "KNIRV_ECONOMICS_ENABLED",
		"economics.port":            "KNIRV_ECONOMICS_PORT",
		"economics.nrn_contract":    "KNIRV_ECONOMICS_NRN_CONTRACT",
		"economics.xion_rpc":        "KNIRV_ECONOMICS_XION_RPC",
		"economics.log_level":       "KNIRV_ECONOMICS_LOG_LEVEL",
		"economics.knirvchain_url":  "KNIRV_ECONOMICS_KNIRVCHAIN_URL",
		"economics.knirvnexus_url":  "KNIRV_ECONOMICS_KNIRVNEXUS_URL",
		"economics.knirvoracle_url": "KNIRV_ECONOMICS__URL",
		"economics.knirvgraph_url":  "KNIRV_ECONOMICS_KNIRVGRAPH_URL",
		"economics.database_path":   "KNIRV_ECONOMICS_DATABASE_PATH",
		"economics.token_symbol":    "KNIRV_ECONOMICS_TOKEN_SYMBOL",
		"economics.token_decimals":  "KNIRV_ECONOMICS_TOKEN_DECIMALS",
		"economics.usd_per_token":   "KNIRV_ECONOMICS_USD_PER_TOKEN",
		"economics.eth_per_token":   "KNIRV_ECONOMICS_ETH_PER_TOKEN",
	}
}

// NewEconomicsConfig creates a new economics configuration with defaults
func NewEconomicsConfig() *EconomicsConfig {
	return &EconomicsConfig{
		Enabled:       false,
		Port:          "8090",
		NRNContract:   "",
		XionRPC:       "",
		LogLevel:      "info",
		KNIRVChainURL: "",
		KNIRVNexusURL: "",
		KNIRVRootURL:  "",
		KNIRVGraphURL: "",
		DatabasePath:  "",
		TokenSymbol:   "NRN",
		TokenDecimals: 18,
		USDPerToken:   0.0,
		ETHPerToken:   0.0,
	}
}

// GetPortInt returns the port as an integer
func (c *EconomicsConfig) GetPortInt() (int, error) {
	return strconv.Atoi(c.Port)
}

// GetEndpoint returns the economics service endpoint URL
func (c *EconomicsConfig) GetEndpoint() string {
	return fmt.Sprintf("http://localhost:%s", c.Port)
}

// IsValidContract returns true if the NRN contract address is valid
func (c *EconomicsConfig) IsValidContract() bool {
	return c.NRNContract != "" && len(c.NRNContract) > 10 // Basic validation
}

// GetComponentURLs returns all configured component URLs
func (c *EconomicsConfig) GetComponentURLs() map[string]string {
	urls := make(map[string]string)

	if c.KNIRVChainURL != "" {
		urls["knirvchain"] = c.KNIRVChainURL
	}
	if c.KNIRVNexusURL != "" {
		urls["knirvserver"] = c.KNIRVNexusURL
	}
	if c.KNIRVRootURL != "" {
		urls["knirvchain"] = c.KNIRVRootURL
	}
	if c.KNIRVGraphURL != "" {
		urls["knirvgraph"] = c.KNIRVGraphURL
	}

	return urls
}
