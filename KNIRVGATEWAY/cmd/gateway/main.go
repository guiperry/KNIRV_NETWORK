package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/config"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/oracle"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/runtime"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/server"
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

	logger.Info("KNIRVORACLE starting",
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

	// Initialize and start knirv-oracle
	logger.Info("Initializing knirv-oracle...")
	oracleCfg, err := oracle.LoadConfigFromEnv()
	if err != nil {
		logger.Fatal("Failed to load oracle configuration", zap.Error(err))
	}

	// Check if oracle binary exists and owner key is set
	oracleBinaryPath := rt.GetOracleBinaryPath()
	if _, err := os.Stat(oracleBinaryPath); err == nil && oracleCfg.OwnerPrivateKey != "" {
		// Binary exists and owner key is set, start as separate process
		logger.Info("Starting knirv-oracle as separate process...", zap.String("path", oracleBinaryPath))
		oracleCmd := exec.Command(oracleBinaryPath)
		oracleCmd.Stdout = os.Stdout
		oracleCmd.Stderr = os.Stderr
		if err := oracleCmd.Start(); err != nil {
			logger.Fatal("Failed to start knirv-oracle daemon", zap.Error(err))
		}

		// Ensure oracle daemon is stopped on exit
		defer func() {
			logger.Info("Stopping knirv-oracle daemon...")
			if err := oracleCmd.Process.Signal(syscall.SIGTERM); err != nil {
				logger.Error("Failed to send SIGTERM to knirv-oracle", zap.Error(err))
				// If SIGTERM fails, force kill
				if err := oracleCmd.Process.Kill(); err != nil {
					logger.Error("Failed to kill knirv-oracle", zap.Error(err))
				}
			}
			// Wait for the process to exit
			_, err := oracleCmd.Process.Wait()
			if err != nil {
				logger.Error("Error waiting for knirv-oracle to exit", zap.Error(err))
			}
		}()
	} else {
		// Binary doesn't exist or owner key not set, initialize directly
		logger.Warn("knirv-oracle binary not found or owner key not set, initializing directly...")

		// If ORACLE_OWNER_KEY is not set, use a default for development/testing
		if oracleCfg.OwnerPrivateKey == "" {
			logger.Warn("ORACLE_OWNER_KEY not set, using development default")
			// This is a dummy private key for testing purposes only
			oracleCfg.OwnerPrivateKey = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
		}

		// Validate oracle configuration
		if err := oracle.ValidateConfig(oracleCfg); err != nil {
			logger.Fatal("Invalid oracle configuration", zap.Error(err))
		}

		logger.Info("Oracle configuration summary", zap.String("config", oracle.ConfigSummary(oracleCfg)))

		// Create oracle instance
		oracleNode, err := oracle.NewOracle(oracleCfg, logger)
		if err != nil {
			logger.Fatal("Failed to create oracle instance", zap.Error(err))
		}

		// Start oracle services in a goroutine
		go func() {
			logger.Info("Starting knirv-oracle services...")
			if err := oracleNode.Start(); err != nil {
				logger.Fatal("Oracle node failed to start", zap.Error(err))
			}
		}()

		// Ensure oracle is stopped on exit
		defer func() {
			logger.Info("Stopping knirv-oracle services...")
			if err := oracleNode.Stop(); err != nil {
				logger.Error("Error stopping oracle node", zap.Error(err))
			}
		}()
	}

	// Initialize HTTP server with webgui static and network website directories
	srv, err := server.New(cfg, rt.GetWebGUIStaticPath(), rt.GetNetworkWebsitePath(), logger)
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

	// Wait a moment for server to start, then open browser if enabled
	if cfg.AutoOpenBrowser {
		go func() {
			time.Sleep(2 * time.Second) // Wait for server to be ready
			url := fmt.Sprintf("http://localhost:%d", cfg.Port)
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

	logger.Info("Shutting down KNIRVORACLE")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop HTTP server
	if err := srv.Stop(ctx); err != nil {
		logger.Error("Error stopping server", zap.Error(err))
	}

	logger.Info("KNIRVORACLE stopped")
}
