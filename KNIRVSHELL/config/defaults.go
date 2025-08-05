package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// setDefaults sets default values for configuration
func setDefaults(v *viper.Viper) {
	// Find home directory
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}

	// Default wallet directory
	walletDir := filepath.Join(home, ".knirv", "wallets")

	// Set defaults
	v.SetDefault("node_url", "http://localhost:8545")
	v.SetDefault("wallet_directory", walletDir)
	v.SetDefault("log_level", "info")
	v.SetDefault("default_fee", 1000000000000000) // 0.001 ETH in wei

	// File server defaults
	v.SetDefault("file_server.enabled", false)
	v.SetDefault("file_server.port", 8080)
	v.SetDefault("file_server.base_url", "http://localhost:8080")

	// AI defaults
	v.SetDefault("ai.provider", "openai")
	v.SetDefault("ai.base_url", "")
	v.SetDefault("ai.default_model", "gpt-4")
	v.SetDefault("ai.max_tokens", 4000)
	v.SetDefault("ai.temperature", 0.7)

	// UI defaults
	v.SetDefault("ui.enable_tui", true)
	v.SetDefault("ui.theme", "default")
	v.SetDefault("ui.color_mode", "256")
	v.SetDefault("ui.show_icons", true)
	v.SetDefault("ui.animation_speed", 100)
	v.SetDefault("ui.compact_mode", false)
}
