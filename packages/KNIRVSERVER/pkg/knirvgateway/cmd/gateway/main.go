package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/config"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/embedded"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/runtime"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/server"
	"go.uber.org/zap"
)

func main() {
	socketPath := flag.String("socket", "", "Unix socket path for HTTP server (overrides PORT)")

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

	// Override socket path from CLI flag if provided
	if *socketPath != "" {
		cfg.SocketPath = *socketPath
	}

	logger.Info("KNIRVGATEWAY starting",
		zap.String("mode", cfg.GatewayMode),
		zap.String("socketPath", cfg.SocketPath),
		zap.Int("port", cfg.Port),
		zap.String("chainID", cfg.ChainID),
	)

	// Initialize runtime and extract embedded files (oracle binary removed)
	rt, err := runtime.NewRuntime(logger, embedded.WebGUIFS, embedded.NetworkWebsiteFS, nil)
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

	// Oracle has moved to KNIRVSERVER — no oracle initialisation in the gateway.
	logger.Info("Oracle is managed by KNIRVSERVER (root node only)")

	// Initialize HTTP server with webgui static and network website directories
	srv, err := server.New(cfg, rt.GetWebGUIStaticPath(), rt.GetNetworkWebsitePath(), logger)
	if err != nil {
		logger.Fatal("Failed to initialize server", zap.Error(err))
	}

	// Start server in a goroutine
	go func() {
		if cfg.SocketPath != "" {
			logger.Info("Starting HTTP server", zap.String("socket", cfg.SocketPath))
		} else {
			logger.Info("Starting HTTP server", zap.Int("port", cfg.Port))
		}
		if err := srv.Start(); err != nil {
			logger.Fatal("Server failed", zap.Error(err))
		}
	}()

	// Wait a moment for server to start, then open browser if enabled
	if cfg.AutoOpenBrowser {
		go func() {
			time.Sleep(2 * time.Second) // Wait for server to be ready
			var url string
			if cfg.SocketPath != "" {
				url = "http://unix/" + cfg.SocketPath
			} else {
				url = fmt.Sprintf("http://localhost:%d", cfg.Port)
			}
			logger.Info("Opening browser to oracle", zap.String("url", url))
			if err := config.OpenBrowser(url); err != nil {
				logger.Warn("Failed to open browser", zap.Error(err))
			}
		}()
	}

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down KNIRVGATEWAY")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop HTTP server
	if err := srv.Stop(ctx); err != nil {
		logger.Error("Error stopping server", zap.Error(err))
	}

	logger.Info("KNIRVGATEWAY stopped")
}
