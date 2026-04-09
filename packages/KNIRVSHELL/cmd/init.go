package cmd

import (
	"os"
	"path/filepath"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/internal/config"
	"github.com/spf13/cobra"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize KNIRVCHAIN CLI configuration",
	Long: `Initialize KNIRVCHAIN CLI configuration with default settings.
This command creates a configuration file in the user's home directory.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get configuration file path
		configFile, _ := cmd.Flags().GetString("config")
		if configFile == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return err
			}
			configFile = filepath.Join(home, ".knirv.yaml")
		}

		// Check if configuration file already exists
		if _, err := os.Stat(configFile); err == nil {
			overwrite, _ := cmd.Flags().GetBool("overwrite")
			if !overwrite {
				log.Warnf("Configuration file already exists at %s", configFile)
				log.Warn("Use --overwrite flag to overwrite existing configuration")
				return nil
			}
		}

		// Create default configuration
		cfg := &config.Config{
			NodeURL:         "http://localhost:8545",
			WalletDirectory: filepath.Join(filepath.Dir(configFile), "wallets"),
			LogLevel:        "info",
			DefaultFee:      1000000000000000, // 0.001 ETH in wei
			FileServer: struct {
				Enabled bool   `mapstructure:"enabled"`
				Port    int    `mapstructure:"port"`
				BaseURL string `mapstructure:"base_url"`
			}{
				Enabled: false,
				Port:    8080,
				BaseURL: "http://localhost:8080",
			},
			AI: struct {
				Provider     string  `mapstructure:"provider"`
				APIKey       string  `mapstructure:"api_key"`
				BaseURL      string  `mapstructure:"base_url"`
				DefaultModel string  `mapstructure:"default_model"`
				MaxTokens    int     `mapstructure:"max_tokens"`
				Temperature  float64 `mapstructure:"temperature"`
			}{
				Provider:     "openai",
				APIKey:       "",
				BaseURL:      "",
				DefaultModel: "gpt-4",
				MaxTokens:    4000,
				Temperature:  0.7,
			},
			UI: struct {
				EnableTUI      bool   `mapstructure:"enable_tui"`
				Theme          string `mapstructure:"theme"`
				ColorMode      string `mapstructure:"color_mode"`
				ShowIcons      bool   `mapstructure:"show_icons"`
				AnimationSpeed int    `mapstructure:"animation_speed"`
				CompactMode    bool   `mapstructure:"compact_mode"`
			}{
				EnableTUI:      true,
				Theme:          "default",
				ColorMode:      "256",
				ShowIcons:      true,
				AnimationSpeed: 100,
				CompactMode:    false,
			},
		}

		// Override with command line flags
		nodeURL, _ := cmd.Flags().GetString("node-url")
		if nodeURL != "" {
			cfg.NodeURL = nodeURL
		}

		walletDir, _ := cmd.Flags().GetString("wallet-dir")
		if walletDir != "" {
			cfg.WalletDirectory = walletDir
		}

		logLevel, _ := cmd.Flags().GetString("log-level")
		if logLevel != "" {
			cfg.LogLevel = logLevel
		}

		// Create wallet directory
		if err := os.MkdirAll(cfg.WalletDirectory, 0700); err != nil {
			return err
		}

		// Save configuration
		if err := config.SaveConfig(cfg, configFile); err != nil {
			return err
		}

		log.Infof("Configuration initialized at %s", configFile)
		log.Infof("Wallet directory: %s", cfg.WalletDirectory)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)

	// Flags
	initCmd.Flags().String("config", "", "Configuration file path")
	initCmd.Flags().Bool("overwrite", false, "Overwrite existing configuration")
	initCmd.Flags().String("node-url", "", "KNIRVCHAIN node URL")
	initCmd.Flags().String("wallet-dir", "", "Wallet directory path")
	initCmd.Flags().String("log-level", "", "Logging level (debug, info, warn, error)")
}
