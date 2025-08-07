package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/knirv/nexus-backend/internal/config"
	"github.com/knirv/nexus-backend/internal/database"
	"github.com/knirv/nexus-backend/internal/services/dvemanager"
	"github.com/knirv/nexus-backend/pkg/p2p"
	"github.com/spf13/viper"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database
	db, err := database.NewBuntDB(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize P2P manager
	p2pManager, err := p2p.NewDVEP2PManager(cfg.ChainID, "dve-manager", db.GetDB())
	if err != nil {
		log.Fatalf("Failed to initialize P2P manager: %v", err)
	}
	defer p2pManager.Stop()

	// Initialize DVE Manager service
	dveManager, err := dvemanager.NewDVEManager(db.GetDB(), p2pManager, cfg)
	if err != nil {
		log.Fatalf("Failed to initialize DVE Manager: %v", err)
	}

	// Start P2P networking
	p2pManager.Start()

	// Start DVE Manager service
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := dveManager.Start(ctx); err != nil {
			log.Printf("DVE Manager error: %v", err)
		}
	}()

	log.Printf("DVE Manager started on chain %s", cfg.ChainID)

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
