package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/oracle"
	"go.uber.org/zap"
)

func main() {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Load configuration from environment variables
	cfg, err := oracle.LoadConfigFromEnv()
	if err != nil {
		logger.Fatal("Failed to load oracle configuration", zap.Error(err))
	}

	// Validate configuration
	if err := oracle.ValidateConfig(cfg); err != nil {
		logger.Fatal("Invalid oracle configuration", zap.Error(err))
	}

	logger.Info("Starting knirv-oracled")
	logger.Info(oracle.ConfigSummary(cfg))

	// Create a new oracle instance
	oracleNode, err := oracle.NewOracle(cfg, logger)
	if err != nil {
		logger.Fatal("Failed to create oracle instance", zap.Error(err))
	}

	// Create a context that can be cancelled
	// defer cancel() // Cancel function is now managed internally by oracleNode

	// Start the oracle services in a goroutine
	go func() {
		if err := oracleNode.Start(); err != nil {
			logger.Fatal("Oracle node failed to start", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down knirv-oracled")

	// Stop the oracle services
	if err := oracleNode.Stop(); err != nil {
		logger.Error("Error stopping oracle node", zap.Error(err))
	}

	logger.Info("knirv-oracled stopped")
}
