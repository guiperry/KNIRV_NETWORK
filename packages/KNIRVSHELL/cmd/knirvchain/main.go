package main

import (
	"fmt"
	"os"
	"time"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/core"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/ui"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/ui/screens"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Version information
	Version   = "1.0.0"
	BuildDate = "2025-06-18"
	GitCommit = "unknown"

	// Global flags
	cfgFile     string
	debug       bool
	apiURL      string
	walletDir   string
	logLevel    string
	colorMode   string
	theme       string
	interactive bool

	// Root command
	rootCmd = &cobra.Command{
		Use:   "knirvchain",
		Short: "KNIRVCHAIN CLI - Multi-Capability Protocol Command Line Interface",
		Long: `KNIRVCHAIN CLI is a command-line interface for interacting with the KNIRVCHAIN
Multi-Capability Protocol. It provides tools for managing wallets, capabilities,
servers, and operational procedures.`,
		Run: func(cmd *cobra.Command, args []string) {
			if interactive {
				runInteractiveMode()
			} else {
				cmd.Help()
			}
		},
	}

	// Version command
	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print the version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("KNIRVCHAIN CLI v%s\n", Version)
			fmt.Printf("Build Date: %s\n", BuildDate)
			fmt.Printf("Git Commit: %s\n", GitCommit)
		},
	}
)

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.knirvchain.yaml)")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug mode")
	rootCmd.PersistentFlags().StringVar(&apiURL, "api-url", "https://api.knirvchain.net", "API URL")
	rootCmd.PersistentFlags().StringVar(&walletDir, "wallet-dir", "", "wallet directory")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().StringVar(&colorMode, "color-mode", "truecolor", "color mode (16, 256, truecolor)")
	rootCmd.PersistentFlags().StringVar(&theme, "theme", "default", "UI theme (default, dark, light, high-contrast)")
	rootCmd.PersistentFlags().BoolVar(&interactive, "interactive", false, "run in interactive mode")

	// Add commands
	rootCmd.AddCommand(versionCmd)

	// Bind flags to viper
	viper.BindPFlag("api.url", rootCmd.PersistentFlags().Lookup("api-url"))
	viper.BindPFlag("wallet.directory", rootCmd.PersistentFlags().Lookup("wallet-dir"))
	viper.BindPFlag("log.level", rootCmd.PersistentFlags().Lookup("log-level"))
	viper.BindPFlag("ui.color_mode", rootCmd.PersistentFlags().Lookup("color-mode"))
	viper.BindPFlag("ui.theme", rootCmd.PersistentFlags().Lookup("theme"))
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".knirvchain" (without extension)
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".knirvchain")
	}

	// Set defaults
	viper.SetDefault("api.url", "https://api.knirvchain.net")
	viper.SetDefault("log.level", "info")
	viper.SetDefault("ui.color_mode", "truecolor")
	viper.SetDefault("ui.theme", "default")

	// If a config file is found, read it in
	if err := viper.ReadInConfig(); err == nil {
		if debug {
			fmt.Println("Using config file:", viper.ConfigFileUsed())
		}
	}

	// Set wallet directory if not specified
	if walletDir == "" {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)
		walletDir = fmt.Sprintf("%s/.knirvchain/wallets", home)
		viper.Set("wallet.directory", walletDir)
	}
}

func setupLogger() *logrus.Logger {
	// Create logger
	logger := logrus.New()

	// Set log level
	level, err := logrus.ParseLevel(viper.GetString("log.level"))
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// Set formatter
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: time.RFC3339,
	})

	return logger
}

func runInteractiveMode() {
	// Setup logger
	logger := setupLogger()

	// Create API client
	apiClient := core.NewAPIClient(
		viper.GetString("api.url"),
		core.WithLogger(logger),
		core.WithTimeout(10*time.Second),
		core.WithRetries(3),
	)

	// Create file manager
	fileManager, err := core.NewFileManager(viper.GetString("wallet.directory"))
	if err != nil {
		logger.Fatalf("Failed to create file manager: %v", err)
	}

	// Create wallet manager
	walletManager := core.NewWalletManager(viper.GetString("wallet.directory"), logger)

	// Create server manager
	serverManager, err := core.NewMCPServerManager(
		apiClient,
		fileManager,
		fmt.Sprintf("%s/servers", viper.GetString("wallet.directory")),
	)
	if err != nil {
		logger.Fatalf("Failed to create server manager: %v", err)
	}

	// Create procedure manager
	procedureManager, err := core.NewOpProcedureManager(
		apiClient,
		fileManager,
		fmt.Sprintf("%s/procedures", viper.GetString("wallet.directory")),
	)
	if err != nil {
		logger.Fatalf("Failed to create procedure manager: %v", err)
	}

	// Get theme
	theme := viper.GetString("ui.theme")

	// Create styles
	styles := ui.DefaultStyles(ui.GetThemeByName(theme))

	// Create screens
	mainMenu := screens.NewMainMenuScreen(styles, 80, 24)
	walletScreen := screens.NewWalletScreen(styles, walletManager, mainMenu)
	capabilityScreen := screens.NewCapabilityScreen(styles, apiClient, mainMenu)
	serverScreen := screens.NewServerScreen(styles, serverManager, mainMenu)
	procedureScreen := screens.NewProcedureScreen(styles, procedureManager, mainMenu)
	settingsScreen := screens.NewSettingsScreen(styles, mainMenu)

	// Add menu items
	mainMenu.AddMenuItem("Wallet Management", "Manage your KNIRVCHAIN wallets", walletScreen)
	mainMenu.AddMenuItem("Capability Management", "Manage MCP capabilities", capabilityScreen)
	mainMenu.AddMenuItem("Server Management", "Manage MCP servers", serverScreen)
	mainMenu.AddMenuItem("Procedure Management", "Manage operational procedures", procedureScreen)
	mainMenu.AddMenuItem("Settings", "Configure KNIRVCHAIN CLI", settingsScreen)

	// Run the UI
	if err := ui.Run(theme, debug, mainMenu); err != nil {
		logger.Fatalf("Error running UI: %v", err)
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
