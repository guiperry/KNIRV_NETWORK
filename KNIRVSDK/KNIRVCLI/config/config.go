package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

// ServiceEndpoints represents endpoints for a KNIRV service
type ServiceEndpoints struct {
	API          string `mapstructure:"api"`
	WebSocket    string `mapstructure:"websocket"`
	Economics    string `mapstructure:"economics"`
	Health       string `mapstructure:"health"`
	PoAuD        string `mapstructure:"poaud"`
	Agentic      string `mapstructure:"agentic"`
	Inference    string `mapstructure:"inference"`
	Plugins      string `mapstructure:"plugins"`
	NRV          string `mapstructure:"nrv"`
	Graph        string `mapstructure:"graph"`
	Transactions string `mapstructure:"transactions"`
}

// ServiceConfig represents configuration for a KNIRV service
type ServiceConfig struct {
	URL       string           `mapstructure:"url"`
	APIKey    string           `mapstructure:"api_key"`
	Endpoints ServiceEndpoints `mapstructure:"endpoints"`
	Timeout   time.Duration    `mapstructure:"timeout"`
	Retries   int              `mapstructure:"retries"`
	Enabled   bool             `mapstructure:"enabled"`
}

// NetworkConfig represents network-level configuration
type NetworkConfig struct {
	Environment string `mapstructure:"environment"`
	Discovery   struct {
		Enabled  bool          `mapstructure:"enabled"`
		Interval time.Duration `mapstructure:"interval"`
		Timeout  time.Duration `mapstructure:"timeout"`
	} `mapstructure:"discovery"`
}

// ServicesConfig represents all KNIRV services configuration
type ServicesConfig struct {
	KNIRVRoot    ServiceConfig `mapstructure:"knirvoracle"`
	KNIRVGateway ServiceConfig `mapstructure:"knirvgateway"`
	KNIRVNexus   ServiceConfig `mapstructure:"knirvnexus"`
	KNIRVGraph   ServiceConfig `mapstructure:"knirvgraph"`
}

// WalletConfig represents wallet configuration
type WalletConfig struct {
	Directory string `mapstructure:"directory"`
	XION      struct {
		Enabled     bool   `mapstructure:"enabled"`
		ChainID     string `mapstructure:"chain_id"`
		MetaAccount bool   `mapstructure:"meta_account"`
		Gasless     bool   `mapstructure:"gasless"`
	} `mapstructure:"xion"`
	NRN struct {
		Enabled    bool   `mapstructure:"enabled"`
		FaucetURL  string `mapstructure:"faucet_url"`
		AutoRefill bool   `mapstructure:"auto_refill"`
		MinBalance string `mapstructure:"min_balance"`
	} `mapstructure:"nrn"`
}

// RealtimeConfig represents real-time communication configuration
type RealtimeConfig struct {
	WebSocket struct {
		Enabled           bool          `mapstructure:"enabled"`
		ReconnectInterval time.Duration `mapstructure:"reconnect_interval"`
		MaxRetries        int           `mapstructure:"max_retries"`
	} `mapstructure:"websocket"`
	SSE struct {
		Enabled    bool          `mapstructure:"enabled"`
		Timeout    time.Duration `mapstructure:"timeout"`
		BufferSize int           `mapstructure:"buffer_size"`
	} `mapstructure:"sse"`
}

// Config represents the enhanced application configuration
type Config struct {
	// Legacy fields for backward compatibility
	NodeURL         string `mapstructure:"node_url"`
	WalletDirectory string `mapstructure:"wallet_directory"`
	LogLevel        string `mapstructure:"log_level"`
	DefaultFee      uint64 `mapstructure:"default_fee"`

	// Enhanced configuration
	KNIRV struct {
		Network  NetworkConfig  `mapstructure:"network"`
		Services ServicesConfig `mapstructure:"services"`
		Wallet   WalletConfig   `mapstructure:"wallet"`
		Realtime RealtimeConfig `mapstructure:"realtime"`
	} `mapstructure:"knirv"`

	FileServer struct {
		Enabled bool   `mapstructure:"enabled"`
		Port    int    `mapstructure:"port"`
		BaseURL string `mapstructure:"base_url"`
	} `mapstructure:"file_server"`
	AI struct {
		Provider     string  `mapstructure:"provider"`
		APIKey       string  `mapstructure:"api_key"`
		BaseURL      string  `mapstructure:"base_url"`
		DefaultModel string  `mapstructure:"default_model"`
		MaxTokens    int     `mapstructure:"max_tokens"`
		Temperature  float64 `mapstructure:"temperature"`
	} `mapstructure:"ai"`
	UI struct {
		EnableTUI      bool   `mapstructure:"enable_tui"`
		Theme          string `mapstructure:"theme"`
		ColorMode      string `mapstructure:"color_mode"`
		ShowIcons      bool   `mapstructure:"show_icons"`
		AnimationSpeed int    `mapstructure:"animation_speed"`
		CompactMode    bool   `mapstructure:"compact_mode"`
	} `mapstructure:"ui"`
}

// LoadConfig loads the configuration from file
func LoadConfig(configFile string) (*Config, error) {
	var config Config

	// Set up viper
	v := viper.New()

	if configFile != "" {
		// Use config file from the flag
		v.SetConfigFile(configFile)
	} else {
		// Find home directory
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("error finding home directory: %w", err)
		}

		// Search config in home directory with name ".knirv" (without extension)
		v.AddConfigPath(home)
		v.SetConfigName(".knirv")
	}

	// Set defaults
	setDefaults(v)

	// Read environment variables
	v.SetEnvPrefix("KNIRV")
	v.AutomaticEnv()

	// Read in config file
	if err := v.ReadInConfig(); err != nil {
		// It's okay if config file doesn't exist
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	// Unmarshal config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Ensure wallet directory exists
	if config.WalletDirectory != "" {
		if err := os.MkdirAll(config.WalletDirectory, 0700); err != nil {
			return nil, fmt.Errorf("error creating wallet directory: %w", err)
		}
	}

	return &config, nil
}

// SaveConfig saves the configuration to file
func SaveConfig(config *Config, configFile string) error {
	v := viper.New()

	// Set config values
	v.Set("node_url", config.NodeURL)
	v.Set("wallet_directory", config.WalletDirectory)
	v.Set("log_level", config.LogLevel)
	v.Set("default_fee", config.DefaultFee)
	v.Set("file_server.enabled", config.FileServer.Enabled)
	v.Set("file_server.port", config.FileServer.Port)
	v.Set("file_server.base_url", config.FileServer.BaseURL)
	v.Set("ai.provider", config.AI.Provider)
	v.Set("ai.api_key", config.AI.APIKey)
	v.Set("ai.base_url", config.AI.BaseURL)
	v.Set("ai.default_model", config.AI.DefaultModel)
	v.Set("ai.max_tokens", config.AI.MaxTokens)
	v.Set("ai.temperature", config.AI.Temperature)
	v.Set("ui.enable_tui", config.UI.EnableTUI)
	v.Set("ui.theme", config.UI.Theme)
	v.Set("ui.color_mode", config.UI.ColorMode)
	v.Set("ui.show_icons", config.UI.ShowIcons)
	v.Set("ui.animation_speed", config.UI.AnimationSpeed)
	v.Set("ui.compact_mode", config.UI.CompactMode)

	// Determine config file path
	if configFile == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("error finding home directory: %w", err)
		}
		configFile = filepath.Join(home, ".knirv.yaml")
	}

	// Write config file
	v.SetConfigFile(configFile)
	if err := v.WriteConfig(); err != nil {
		return fmt.Errorf("error writing config file: %w", err)
	}

	return nil
}
