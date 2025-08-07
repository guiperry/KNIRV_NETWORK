package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/knirv/nexus-backend/internal/config"
	"github.com/knirv/nexus-backend/internal/database"
	"github.com/knirv/nexus-backend/internal/services/dvemanager"
	"github.com/knirv/nexus-backend/pkg/gui"
	"github.com/knirv/nexus-backend/pkg/p2p"
	"github.com/spf13/viper"
)

func main() {
	// Parse command line flags
	var (
		guiMode    = flag.Bool("gui", false, "Enable GUI mode for local administration")
		configFile = flag.String("config", "", "Configuration file path")
		port       = flag.Int("port", 0, "Service port (overrides config)")
		guiPort    = flag.Int("gui-port", 0, "GUI port (overrides config)")
	)
	flag.Parse()

	// Initialize configuration with viper
	cfg, err := config.LoadWithDefaults()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
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
	p2pManager, err := p2p.NewDVEP2PManager(chainID, "dve-manager", db.GetDB())
	if err != nil {
		log.Fatalf("Failed to initialize P2P manager: %v", err)
	}
	defer p2pManager.Stop()

	// Initialize DVE Manager service
	dveManager, err := dvemanager.NewDVEManager(db.GetDB(), p2pManager, cfg)
	if err != nil {
		log.Fatalf("Failed to initialize DVE Manager: %v", err)
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

	// Start DVE Manager service

	go func() {
		if err := dveManager.Start(ctx); err != nil {
			log.Printf("DVE Manager error: %v", err)
		}
	}()

	log.Printf("DVE Manager started on chain %s", chainID)

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down DVE Manager...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := dveManager.Stop(shutdownCtx); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}

	// Stop GUI server if running
	if guiServer != nil {
		if err := guiServer.Stop(shutdownCtx); err != nil {
			log.Printf("Error stopping GUI server: %v", err)
		}
	}

	log.Println("DVE Manager stopped")
}

func init() {
	// Set default configuration values
	viper.SetDefault("database.path", "/app/data/dve-manager.db")
	viper.SetDefault("chain_id", "knirv-nexus-mainnet")
	viper.SetDefault("node_role", "dve-manager")
	viper.SetDefault("p2p.port", 4001)
	viper.SetDefault("api.port", 8080)
	viper.SetDefault("log.level", "info")

	// Environment variable bindings
	viper.BindEnv("database.path", "KNIRV_DATABASE_PATH")
	viper.BindEnv("chain_id", "KNIRV_CHAIN_ID")
	viper.BindEnv("node_role", "KNIRV_NODE_ROLE")
	viper.BindEnv("p2p.port", "KNIRV_P2P_PORT")
	viper.BindEnv("api.port", "KNIRV_API_PORT")
	viper.BindEnv("log.level", "KNIRV_LOG_LEVEL")
}
