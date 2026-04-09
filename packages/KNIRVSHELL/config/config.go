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
	KNIRVNexus   ServiceConfig `mapstructure:"knirvserver"`
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
	Obsidian struct {
		VaultPath string `mapstructure:"vault_path"`
		VaultName string `mapstructure:"vault_name"`
	} `mapstructure:"obsidian"`
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

	return &config, nil
}

// SaveConfig saves the configuration to file
func SaveConfig(config *Config, configFile string) error {
	v := viper.New()

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
	v.Set("obsidian.vault_path", config.Obsidian.VaultPath)
	v.Set("obsidian.vault_name", config.Obsidian.VaultName)

	v.Set("knirv.network.environment", config.KNIRV.Network.Environment)
	v.Set("knirv.network.discovery.enabled", config.KNIRV.Network.Discovery.Enabled)
	v.Set("knirv.network.discovery.interval", config.KNIRV.Network.Discovery.Interval)
	v.Set("knirv.network.discovery.timeout", config.KNIRV.Network.Discovery.Timeout)

	v.Set("knirv.services.knirvoracle.url", config.KNIRV.Services.KNIRVRoot.URL)
	v.Set("knirv.services.knirvoracle.api_key", config.KNIRV.Services.KNIRVRoot.APIKey)
	v.Set("knirv.services.knirvoracle.timeout", config.KNIRV.Services.KNIRVRoot.Timeout)
	v.Set("knirv.services.knirvoracle.retries", config.KNIRV.Services.KNIRVRoot.Retries)
	v.Set("knirv.services.knirvoracle.enabled", config.KNIRV.Services.KNIRVRoot.Enabled)

	v.Set("knirv.services.knirvgateway.url", config.KNIRV.Services.KNIRVGateway.URL)
	v.Set("knirv.services.knirvgateway.api_key", config.KNIRV.Services.KNIRVGateway.APIKey)
	v.Set("knirv.services.knirvgateway.timeout", config.KNIRV.Services.KNIRVGateway.Timeout)
	v.Set("knirv.services.knirvgateway.retries", config.KNIRV.Services.KNIRVGateway.Retries)
	v.Set("knirv.services.knirvgateway.enabled", config.KNIRV.Services.KNIRVGateway.Enabled)

	v.Set("knirv.services.knirvserver.url", config.KNIRV.Services.KNIRVNexus.URL)
	v.Set("knirv.services.knirvserver.api_key", config.KNIRV.Services.KNIRVNexus.APIKey)
	v.Set("knirv.services.knirvserver.timeout", config.KNIRV.Services.KNIRVNexus.Timeout)
	v.Set("knirv.services.knirvserver.retries", config.KNIRV.Services.KNIRVNexus.Retries)
	v.Set("knirv.services.knirvserver.enabled", config.KNIRV.Services.KNIRVNexus.Enabled)

	v.Set("knirv.services.knirvgraph.url", config.KNIRV.Services.KNIRVGraph.URL)
	v.Set("knirv.services.knirvgraph.api_key", config.KNIRV.Services.KNIRVGraph.APIKey)
	v.Set("knirv.services.knirvgraph.timeout", config.KNIRV.Services.KNIRVGraph.Timeout)
	v.Set("knirv.services.knirvgraph.retries", config.KNIRV.Services.KNIRVGraph.Retries)
	v.Set("knirv.services.knirvgraph.enabled", config.KNIRV.Services.KNIRVGraph.Enabled)

	v.Set("knirv.wallet.directory", config.KNIRV.Wallet.Directory)
	v.Set("knirv.wallet.xion.enabled", config.KNIRV.Wallet.XION.Enabled)
	v.Set("knirv.wallet.xion.chain_id", config.KNIRV.Wallet.XION.ChainID)
	v.Set("knirv.wallet.xion.meta_account", config.KNIRV.Wallet.XION.MetaAccount)
	v.Set("knirv.wallet.xion.gasless", config.KNIRV.Wallet.XION.Gasless)
	v.Set("knirv.wallet.nrn.enabled", config.KNIRV.Wallet.NRN.Enabled)
	v.Set("knirv.wallet.nrn.faucet_url", config.KNIRV.Wallet.NRN.FaucetURL)
	v.Set("knirv.wallet.nrn.auto_refill", config.KNIRV.Wallet.NRN.AutoRefill)
	v.Set("knirv.wallet.nrn.min_balance", config.KNIRV.Wallet.NRN.MinBalance)

	v.Set("knirv.realtime.websocket.enabled", config.KNIRV.Realtime.WebSocket.Enabled)
	v.Set("knirv.realtime.websocket.reconnect_interval", config.KNIRV.Realtime.WebSocket.ReconnectInterval)
	v.Set("knirv.realtime.websocket.max_retries", config.KNIRV.Realtime.WebSocket.MaxRetries)
	v.Set("knirv.realtime.sse.enabled", config.KNIRV.Realtime.SSE.Enabled)
	v.Set("knirv.realtime.sse.timeout", config.KNIRV.Realtime.SSE.Timeout)
	v.Set("knirv.realtime.sse.buffer_size", config.KNIRV.Realtime.SSE.BufferSize)

	if configFile == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("error finding home directory: %w", err)
		}
		configFile = filepath.Join(home, ".knirv.yaml")
	}

	v.SetConfigFile(configFile)
	if err := v.WriteConfig(); err != nil {
		return fmt.Errorf("error writing config file: %w", err)
	}

	return nil
}
