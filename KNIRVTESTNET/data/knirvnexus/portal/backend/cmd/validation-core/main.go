package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"nexus-backend/internal/config"
	"nexus-backend/internal/database"
	"nexus-backend/internal/services/validation"
	"nexus-backend/pkg/gui"
	"nexus-backend/pkg/p2p"

	"github.com/spf13/viper"
)

func main() {
	// Parse command line flags
	var (
		guiMode    = flag.Bool("gui", false, "Enable GUI mode for local administration")
		configFile = flag.String("config", "", "Configuration file path")
		port       = flag.Int("port", 0, "Service port (overrides config)")
		guiPort    = flag.Int("gui-port", 0, "GUI port (overrides config)")
		testnet    = flag.Bool("testnet", false, "Enable testnet mode")
	)
	flag.Parse()

	// Initialize configuration with viper
	var cfg *config.Config
	var err error

	if *configFile != "" {
		// Load specific config file
		viper.SetConfigFile(*configFile)
		if err := viper.ReadInConfig(); err != nil {
			log.Fatalf("Failed to read config file %s: %v", *configFile, err)
		}
		cfg = &config.Config{}
		if err := viper.Unmarshal(cfg); err != nil {
			log.Fatalf("Failed to unmarshal config: %v", err)
		}
	} else {
		// Load with defaults
		cfg, err = config.LoadWithDefaults()
		if err != nil {
			log.Fatalf("Failed to load configuration: %v", err)
		}
	}

	// Handle testnet mode
	if *testnet {
		log.Println("Starting Validation Core in testnet mode")
		cfg.Testnet = true
	}

	// Load configuration file if specified
	if *configFile != "" {
		viper.SetConfigFile(*configFile)
		if err := viper.ReadInConfig(); err != nil {
			log.Printf("Warning: Could not read config file: %v", err)
		}
	}

	// Override config with CLI flags
	if *guiMode {
		viper.Set("gui.enabled", true)
		viper.Set("mode", "gui")
	}
	if *port != 0 {
		viper.Set("api.port", *port)
	}
	if *guiPort != 0 {
		viper.Set("gui.port", *guiPort)
	}

	// Reload configuration with overrides
	cfg, err = config.LoadWithDefaults()
	if err != nil {
		log.Fatalf("Failed to reload configuration: %v", err)
	}

	// Initialize database
	db, err := database.NewBuntDB(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Get chain ID (prefer Network.ChainID, fallback to ChainID)
	chainID := cfg.Network.ChainID
	if chainID == "" {
		chainID = cfg.ChainID
	}

	// Initialize P2P manager
	p2pManager, err := p2p.NewDVEP2PManager(chainID, "dve-validator", db.GetDB())
	if err != nil {
		log.Fatalf("Failed to initialize P2P manager: %v", err)
	}
	defer p2pManager.Stop()

	// Initialize Validation Core service
	validationCore, err := validation.NewValidationCore(db.GetDB(), p2pManager, cfg)
	if err != nil {
		log.Fatalf("Failed to initialize Validation Core: %v", err)
	}

	// Create context for services
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize GUI server if enabled
	var guiServer *gui.Server
	if cfg.GUI.Enabled && cfg.Mode == "gui" {
		log.Println("Starting in GUI mode - No authentication required")
		guiServer = gui.NewServer(cfg)
		if err := guiServer.Start(ctx); err != nil {
			log.Printf("Failed to start GUI server: %v", err)
		}
	} else {
		log.Println("Starting in headless mode - API only")
	}

	// Start P2P networking
	p2pManager.Start()

	// Start Validation Core service

	go func() {
		if err := validationCore.Start(ctx); err != nil {
			log.Printf("Validation Core error: %v", err)
		}
	}()

	log.Printf("Validation Core started on chain %s", chainID)

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down Validation Core...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := validationCore.Stop(shutdownCtx); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}

	// Stop GUI server if running
	if guiServer != nil {
		if err := guiServer.Stop(shutdownCtx); err != nil {
			log.Printf("Error stopping GUI server: %v", err)
		}
	}

	log.Println("Validation Core stopped")
}

func init() {
	// Set default configuration values for validation-core
	viper.SetDefault("database.path", "./data/validation-core.db")
	viper.SetDefault("chain_id", "knirv-nexus-mainnet")
	viper.SetDefault("node_role", "dve-validator")
	viper.SetDefault("p2p.port", 4001)
	viper.SetDefault("api.port", 8081)
	viper.SetDefault("gui.port", 9081)
	viper.SetDefault("log.level", "info")
	viper.SetDefault("validation.timeout", "300s")
	viper.SetDefault("validation.max_concurrent", 10)

	// Environment variable bindings
	viper.BindEnv("database.path", "KNIRV_DATABASE_PATH")
	viper.BindEnv("chain_id", "KNIRV_CHAIN_ID")
	viper.BindEnv("node_role", "KNIRV_NODE_ROLE")
	viper.BindEnv("p2p.port", "KNIRV_P2P_PORT")
	viper.BindEnv("api.port", "KNIRV_API_PORT")
	viper.BindEnv("gui.port", "KNIRV_GUI_PORT")
	viper.BindEnv("log.level", "KNIRV_LOG_LEVEL")
	viper.BindEnv("validation.timeout", "KNIRV_VALIDATION_TIMEOUT")
	viper.BindEnv("validation.max_concurrent", "KNIRV_VALIDATION_MAX_CONCURRENT")
}
