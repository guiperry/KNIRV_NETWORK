package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// MinimalConfig contains only the essential configuration values that should be saved to file
// All other values will be derived from the settings matrix
type MinimalConfig struct {
	// Core network settings
	Port       uint64 `json:"port,omitempty"`
	P2PPort    uint64 `json:"p2p_port,omitempty"`
	WalletPort uint64 `json:"wallet_port,omitempty"`
	AltGUIPort uint64 `json:"alt_gui_port,omitempty"`

	// Database paths
	BlockchainDatabasePath string `json:"shared_database_path,omitempty"`
	SearchableDatabasePath string `json:"local_database_path,omitempty"`
	ReflectionDatabasePath string `json:"reflection_database_path,omitempty"`

	// Blockchain addresses
	MinersAddress string `json:"miners_address,omitempty"`
	MasterAddress string `json:"master_address,omitempty"`

	// Core flags
	NoWalletServer bool `json:"no_wallet_server,omitempty"`
	ClientOnly     bool `json:"client_only,omitempty"`
	UseGUI         bool `json:"use_gui,omitempty"`

	// Network settings
	ReflectionURLs []string `json:"reflection_urls,omitempty"`
	ChainID        string   `json:"chain_id,omitempty"`

	// Installation status
	InstallComplete bool `json:"install_complete,omitempty"`

	// Custom overrides for specific roles
	// These will only be saved if they differ from the matrix defaults
	CustomSettings map[string]interface{} `json:"custom_settings,omitempty"`
	PublicIPInfo   map[string]interface{} `json:"public_ip_info,omitempty"`
}

// ToMinimalConfig converts a full Config to a MinimalConfig for saving to file
func ToMinimalConfig(cfg *Config, role Role) *MinimalConfig {
	// Get the default settings for this role from the matrix
	matrixSettings, exists := RoleSettingsMatrix[role]
	if !exists {
		log.Printf("Warning: No default settings found for role %s in matrix. Saving full config.", role)
		return toMinimalConfigFallback(cfg)
	}

	// Create a minimal config with only the essential values
	minCfg := &MinimalConfig{
		Port:                   cfg.Port,
		P2PPort:                cfg.P2PPort,
		WalletPort:             cfg.WalletPort,
		AltGUIPort:             cfg.AltGUIPort,
		BlockchainDatabasePath: cfg.BlockchainDatabasePath,
		SearchableDatabasePath: cfg.SearchableDatabasePath,
		ReflectionDatabasePath: cfg.ReflectionDatabasePath,
		MinersAddress:          cfg.MinersAddress,
		MasterAddress:          cfg.MasterAddress,
		NoWalletServer:         cfg.NoWalletServer,
		ClientOnly:             cfg.ClientOnly,
		UseGUI:                 cfg.UseGUI,
		ReflectionURLs:         cfg.ReflectionURLs,
		ChainID:                cfg.ChainID,
		InstallComplete:        cfg.InstallComplete,
		CustomSettings:         make(map[string]interface{}),
		PublicIPInfo:           cfg.PublicIPInfo, // Always include if present
	}

	// Check for custom overrides that differ from matrix defaults
	// Only save values that differ from the matrix defaults

	// Example: Check if PaymentProcessor settings differ from defaults
	if cfg.PaymentProcessor.Enabled != matrixSettings.PaymentProcessor.Enabled {
		minCfg.CustomSettings["payment_processor.enabled"] = cfg.PaymentProcessor.Enabled
	}

	if cfg.PaymentProcessor.TokenSymbol != matrixSettings.PaymentProcessor.TokenSymbol &&
		cfg.PaymentProcessor.TokenSymbol != "" {
		minCfg.CustomSettings["payment_processor.token_symbol"] = cfg.PaymentProcessor.TokenSymbol
	}

	// Add other custom settings checks as needed
	// Only include settings that have been explicitly changed from defaults

	// If no custom settings, set to nil to avoid empty map in JSON
	if len(minCfg.CustomSettings) == 0 {
		minCfg.CustomSettings = nil
	}

	return minCfg
}

// toMinimalConfigFallback is used when no matrix settings are found
func toMinimalConfigFallback(cfg *Config) *MinimalConfig {
	return &MinimalConfig{
		Port:                   cfg.Port,
		P2PPort:                cfg.P2PPort,
		WalletPort:             cfg.WalletPort,
		AltGUIPort:             cfg.AltGUIPort,
		BlockchainDatabasePath: cfg.BlockchainDatabasePath,
		SearchableDatabasePath: cfg.SearchableDatabasePath,
		ReflectionDatabasePath: cfg.ReflectionDatabasePath,
		MinersAddress:          cfg.MinersAddress,
		MasterAddress:          cfg.MasterAddress,
		NoWalletServer:         cfg.NoWalletServer,
		ClientOnly:             cfg.ClientOnly,
		UseGUI:                 cfg.UseGUI,
		ReflectionURLs:         cfg.ReflectionURLs,
		ChainID:                cfg.ChainID,
		InstallComplete:        cfg.InstallComplete,
		PublicIPInfo:           cfg.PublicIPInfo,
	}
}

// SaveMinimalConfig saves only the essential configuration to the specified JSON file path
func SaveMinimalConfig(configPath string, cfg *Config, role Role) (string, error) {
	if cfg == nil {
		return "", fmt.Errorf("SaveMinimalConfig: cannot save nil config")
	}
	if configPath == "" {
		return "", fmt.Errorf("SaveMinimalConfig: configPath cannot be empty")
	}

	// Skip saving for Root role for security reasons
	if cfg.IsRoot {
		log.Printf("SECURITY: Skipping config file save for Root role to prevent sensitive data exposure")
		return configPath, nil
	}

	log.Printf("Saving minimal configuration to: %s", configPath)

	// Convert to minimal config
	minCfg := ToMinimalConfig(cfg, role)

	// Create directory if it doesn't exist
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory %s: %w", dir, err)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(minCfg, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal minimal config: %w", err)
	}

	// Write the configuration to the file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return configPath, fmt.Errorf("failed to write config file %s: %w", configPath, err)
	}

	return configPath, nil
}

// SaveMinimalConfigToUserDir saves the minimal configuration to the user config directory
func SaveMinimalConfigToUserDir(cfg *Config, role Role) error {
	// Skip saving for Root role for security reasons
	if role == Root {
		log.Printf("SECURITY: Skipping config file save for Root role to prevent sensitive data exposure")
		return nil
	}

	roleSpecificFilename := fmt.Sprintf("%s_config.json", strings.ToLower(role.String()))

	// Get the role-specific data directory
	roleDataDir, err := GetDataDir(role)
	if err != nil {
		return fmt.Errorf("could not get role-specific data directory: %w", err)
	}

	// First try to save in the role-specific data directory
	targetSavePath := filepath.Join(roleDataDir, roleSpecificFilename)
	_, err = SaveMinimalConfig(targetSavePath, cfg, role)
	if err != nil {
		return fmt.Errorf("failed to save minimal config to role-specific dir %s: %w", targetSavePath, err)
	}

	log.Printf("Saved minimal config to role-specific data dir: %s", targetSavePath)
	return nil
}
