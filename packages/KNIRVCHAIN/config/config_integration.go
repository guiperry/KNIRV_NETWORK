package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// ConfigManager handles configuration management
type ConfigManager struct {
	viper        *viper.Viper
	configPath   string
	cfg          *Config
	configValues map[string]interface{} // Store values that don't exist in Config struct
}

// NewConfigManager creates a new config manager
func NewConfigManager(configPath string) (*ConfigManager, error) {
	v := viper.New()

	// Set default configuration file
	if configPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		configPath = filepath.Join(homeDir, ".KNIRVCHAIN", "config.yaml")
	}

	// Create config directory if it doesn't exist
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Set config file
	v.SetConfigFile(configPath)

	// Set default values
	setDefaultConfig(v)

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found, create it
			if err := v.WriteConfigAs(configPath); err != nil {
				return nil, fmt.Errorf("failed to create config file: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Create config manager
	cm := &ConfigManager{
		viper:        v,
		configPath:   configPath,
		configValues: make(map[string]interface{}),
	}

	// Load config
	cfg, err := cm.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	cm.cfg = cfg

	return cm, nil
}

// setDefaultConfig sets default configuration values
func setDefaultConfig(v *viper.Viper) {
	// Node settings
	v.SetDefault("node.name", "KNIRVCHAIN Node")
	v.SetDefault("node.role", string(RoleClient))
	v.SetDefault("node.chain_id", "KNIRVCHAIN-testnet")

	// Network settings
	v.SetDefault("network.port", 8080)
	v.SetDefault("network.p2p_port", 9090)
	v.SetDefault("network.wallet_port", 8081)
	v.SetDefault("network.max_devs", 50)

	// UI settings
	v.SetDefault("ui.use_terminal_integration", true)
	v.SetDefault("ui.theme", "dark")

	// Blockchain settings
	v.SetDefault("blockchain.data_dir", "data")
	v.SetDefault("blockchain.block_time", 5)

	// Wallet settings
	v.SetDefault("wallet.auto_unlock", false)

	// API settings
	v.SetDefault("api.enable", true)
	v.SetDefault("api.cors", "*")

	// Logging settings
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.file", "logs/KNIRVCHAIN.log")
}

// LoadConfig loads the configuration from Viper
func (cm *ConfigManager) LoadConfig() (*Config, error) {
	cfg := &Config{}

	// We need to check what fields actually exist in the Config struct
	// and update our code accordingly

	// For now, let's set the fields we know exist based on config/config.go
	// Store configuration values in a map that we can use later
	configValues := make(map[string]interface{})

	// Store the role string in a temporary variable since cfg.Role doesn't exist
	roleStr := cm.viper.GetString("node.role")
	configValues["node.role"] = roleStr

	// Set ChainID which we know exists
	cfg.ChainID = cm.viper.GetString("node.chain_id")

	// Network settings - convert int to uint64
	cfg.Port = uint64(cm.viper.GetInt("network.port"))
	cfg.P2PPort = uint64(cm.viper.GetInt("network.p2p_port"))
	cfg.WalletPort = uint64(cm.viper.GetInt("network.wallet_port"))

	// Store other configuration values for later use
	configValues["node.name"] = cm.viper.GetString("node.name")
	configValues["network.max_devs"] = cm.viper.GetInt("network.max_devs")
	configValues["ui.use_terminal_integration"] = cm.viper.GetBool("ui.use_terminal_integration")
	configValues["ui.theme"] = cm.viper.GetString("ui.theme")
	configValues["blockchain.data_dir"] = cm.viper.GetString("blockchain.data_dir")
	configValues["blockchain.block_time"] = cm.viper.GetInt("blockchain.block_time")
	configValues["wallet.auto_unlock"] = cm.viper.GetBool("wallet.auto_unlock")
	configValues["api.enable"] = cm.viper.GetBool("api.enable")
	configValues["api.cors"] = cm.viper.GetString("api.cors")
	configValues["logging.level"] = cm.viper.GetString("logging.level")
	configValues["logging.file"] = cm.viper.GetString("logging.file")

	// Store the map in the ConfigManager for later use
	cm.configValues = configValues

	return cfg, nil
}

// SaveConfig saves the configuration to the config file
func (cm *ConfigManager) SaveConfig() error {
	// Update Viper with current config values
	cm.updateViperFromConfig()

	// Write config to file
	return cm.viper.WriteConfig()
}

// updateViperFromConfig updates Viper with values from the Config struct
func (cm *ConfigManager) updateViperFromConfig() {
	// Only set values for fields that exist in the Config struct

	// Node settings
	// Use stored values for fields that don't exist in Config
	if nodeName, ok := cm.configValues["node.name"]; ok {
		cm.viper.Set("node.name", nodeName)
	}
	if role, ok := cm.configValues["node.role"]; ok {
		cm.viper.Set("node.role", role)
	}
	cm.viper.Set("node.chain_id", cm.cfg.ChainID)

	// Network settings
	cm.viper.Set("network.port", cm.cfg.Port)
	cm.viper.Set("network.p2p_port", cm.cfg.P2PPort)
	cm.viper.Set("network.wallet_port", cm.cfg.WalletPort)
	if maxPeers, ok := cm.configValues["network.max_devs"]; ok {
		cm.viper.Set("network.max_devs", maxPeers)
	}

	// UI settings
	if useTerminalIntegration, ok := cm.configValues["ui.use_terminal_integration"]; ok {
		cm.viper.Set("ui.use_terminal_integration", useTerminalIntegration)
	}
	if theme, ok := cm.configValues["ui.theme"]; ok {
		cm.viper.Set("ui.theme", theme)
	}

	// Blockchain settings
	if dataDir, ok := cm.configValues["blockchain.data_dir"]; ok {
		cm.viper.Set("blockchain.data_dir", dataDir)
	}
	if blockTime, ok := cm.configValues["blockchain.block_time"]; ok {
		cm.viper.Set("blockchain.block_time", blockTime)
	}

	// Wallet settings
	if autoUnlock, ok := cm.configValues["wallet.auto_unlock"]; ok {
		cm.viper.Set("wallet.auto_unlock", autoUnlock)
	}

	// API settings
	if enableAPI, ok := cm.configValues["api.enable"]; ok {
		cm.viper.Set("api.enable", enableAPI)
	}
	if corsOrigin, ok := cm.configValues["api.cors"]; ok {
		cm.viper.Set("api.cors", corsOrigin)
	}

	// Logging settings
	if logLevel, ok := cm.configValues["logging.level"]; ok {
		cm.viper.Set("logging.level", logLevel)
	}
	if logFile, ok := cm.configValues["logging.file"]; ok {
		cm.viper.Set("logging.file", logFile)
	}
}

// GetConfigPath returns the path to the config file
func (cm *ConfigManager) GetConfigPath() string {
	return cm.configPath
}

// GetConfig returns the current configuration
func (cm *ConfigManager) GetConfig() *Config {
	return cm.cfg
}

// SetConfig sets the configuration
func (cm *ConfigManager) SetConfig(cfg *Config) {
	cm.cfg = cfg
}

// UpdateConfigValue updates a specific configuration value
func (cm *ConfigManager) UpdateConfigValue(key string, value interface{}) error {
	// Set value in Viper
	cm.viper.Set(key, value)

	// Save config
	if err := cm.SaveConfig(); err != nil {
		return err
	}

	// Reload config
	cfg, err := cm.LoadConfig()
	if err != nil {
		return err
	}
	cm.cfg = cfg

	return nil
}
