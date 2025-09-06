package cmd

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile      string
	nodeURL      string
	verbose      bool
	logLevel     string
	logFormat    string
	aiProvider   string
	aiModel      string
	enableTUI    bool
	theme        string
	colorMode    string
	noAnimations bool

	log = logrus.New()

	rootCmd = &cobra.Command{
		Use:   "knirv",
		Short: "KNIRVCHAIN CLI tool for blockchain interaction",
		Long: `KNIRVCHAIN CLI is a comprehensive command-line interface for interacting with the KNIRVCHAIN blockchain.
It enables developers to manage wallets, register capabilities, and interact with the blockchain ecosystem.`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Set up logging based on flags
			if verbose {
				log.SetLevel(logrus.DebugLevel)
			} else {
				switch logLevel {
				case "debug":
					log.SetLevel(logrus.DebugLevel)
				case "info":
					log.SetLevel(logrus.InfoLevel)
				case "warn":
					log.SetLevel(logrus.WarnLevel)
				case "error":
					log.SetLevel(logrus.ErrorLevel)
				default:
					log.SetLevel(logrus.InfoLevel)
				}
			}

			// Set log format
			if logFormat == "json" {
				log.SetFormatter(&logrus.JSONFormatter{})
			} else {
				log.SetFormatter(&logrus.TextFormatter{
					FullTimestamp: true,
				})
			}
		},
	}
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// GetRootCmd returns the root command for use in REPL mode
func GetRootCmd() *cobra.Command {
	return rootCmd
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.knirv.yaml)")
	rootCmd.PersistentFlags().StringVar(&nodeURL, "node-url", "", "URL of the KNIRVCHAIN node")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "enable verbose logging")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "set logging level (debug, info, warn, error)")
	rootCmd.PersistentFlags().StringVar(&logFormat, "log-format", "text", "set logging format (text, json)")
	rootCmd.PersistentFlags().StringVar(&aiProvider, "ai-provider", "openai", "AI provider to use (openai, anthropic)")
	rootCmd.PersistentFlags().StringVar(&aiModel, "ai-model", "", "AI model to use for generation")
	rootCmd.PersistentFlags().BoolVar(&enableTUI, "tui", false, "enable terminal UI mode with bubbletea")
	rootCmd.PersistentFlags().StringVar(&theme, "theme", "default", "set UI theme (default, dark, light, high-contrast)")
	rootCmd.PersistentFlags().StringVar(&colorMode, "color-mode", "256", "set color mode (16, 256, truecolor)")
	rootCmd.PersistentFlags().BoolVar(&noAnimations, "no-animations", false, "disable UI animations")

	// Bind flags to viper
	viper.BindPFlag("node_url", rootCmd.PersistentFlags().Lookup("node-url"))
	viper.BindPFlag("log_level", rootCmd.PersistentFlags().Lookup("log-level"))
	viper.BindPFlag("log_format", rootCmd.PersistentFlags().Lookup("log-format"))
	viper.BindPFlag("ai.provider", rootCmd.PersistentFlags().Lookup("ai-provider"))
	viper.BindPFlag("ai.default_model", rootCmd.PersistentFlags().Lookup("ai-model"))
	viper.BindPFlag("ui.enable_tui", rootCmd.PersistentFlags().Lookup("tui"))
	viper.BindPFlag("ui.theme", rootCmd.PersistentFlags().Lookup("theme"))
	viper.BindPFlag("ui.color_mode", rootCmd.PersistentFlags().Lookup("color-mode"))
	viper.BindPFlag("ui.animation_speed", rootCmd.PersistentFlags().Lookup("no-animations"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		if err != nil {
			// Log the error but don't exit; Viper might still find the config
			// through other means (e.g., --config flag, env vars), or the config
			// might not be strictly required for all commands or test scenarios.
			log.Warnf("Error finding home directory: %v. Will not search for config in home directory.", err)
		} else {
			// Search config in home directory with name ".knirv" (without extension).
			viper.AddConfigPath(home)
			viper.SetConfigName(".knirv")
		}
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		log.Debugf("Using config file: %s", viper.ConfigFileUsed())
	}
}
