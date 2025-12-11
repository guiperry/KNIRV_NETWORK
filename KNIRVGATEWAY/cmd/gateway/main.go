package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/config"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/runtime"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/server"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/services"
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

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
	}

	logger.Info("KNIRVGATEWAY starting",
		zap.String("mode", cfg.GatewayMode),
		zap.Int("port", cfg.Port),
		zap.String("chainID", cfg.ChainID),
	)

	// Initialize runtime and extract embedded files
	rt, err := runtime.NewRuntime(logger)
	if err != nil {
		logger.Fatal("Failed to initialize runtime", zap.Error(err))
	}

	logger.Info("Extracting embedded services and website...")
	if err := rt.Setup(); err != nil {
		logger.Fatal("Failed to setup runtime", zap.Error(err))
	}

	// Cleanup runtime on exit
	defer func() {
		logger.Info("Cleaning up runtime...")
		if err := rt.Cleanup(); err != nil {
			logger.Error("Failed to cleanup runtime", zap.Error(err))
		}
	}()

	// Initialize service manager with extracted services directory
	serviceManager, err := services.NewManager(cfg, logger, rt.ServicesDir)
	if err != nil {
		logger.Fatal("Failed to initialize service manager", zap.Error(err))
	}

	// Start services if auto-start is enabled
	if cfg.NodeJSServicesAutoStart {
		logger.Info("Auto-starting Node.js services")
		if err := serviceManager.StartAll(context.Background()); err != nil {
			logger.Warn("Failed to auto-start services", zap.Error(err))
		}
	}

	// Initialize HTTP server with network-website directory
	srv, err := server.New(cfg, serviceManager, rt.GetNetworkWebsitePath(), logger)
	if err != nil {
		logger.Fatal("Failed to initialize server", zap.Error(err))
	}

	// Start server in a goroutine
	go func() {
		logger.Info("Starting HTTP server", zap.Int("port", cfg.Port))
		if err := srv.Start(); err != nil {
			logger.Fatal("Server failed", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down KNIRVGATEWAY")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop services
	if err := serviceManager.StopAll(ctx); err != nil {
		logger.Error("Error stopping services", zap.Error(err))
	}

	// Stop HTTP server
	if err := srv.Stop(ctx); err != nil {
		logger.Error("Error stopping server", zap.Error(err))
	}

	logger.Info("KNIRVGATEWAY stopped")
}
