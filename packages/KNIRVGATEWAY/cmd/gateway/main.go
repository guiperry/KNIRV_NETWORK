package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/cloudflare"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/config"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/embedded"
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

	logger.Info("KNIRVGATEWAY starting",
		zap.String("mode", cfg.GatewayMode),
		zap.Int("port", cfg.Port),
		zap.String("chainID", cfg.ChainID),
	)

	// Initialize runtime and extract embedded files (oracle binary removed)
	rt, err := runtime.NewRuntime(logger, embedded.WebGUIFS, embedded.GraphChainExplorerFS, embedded.KnirvChainPortalFS, nil)
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

	// Initialize HTTP server with webgui static and explorer directories
	srv, err := server.New(cfg, rt.GetWebGUIStaticPath(), rt.GetGraphChainExplorerPath(), rt.GetKnirvChainPortalPath(), logger)
	if err != nil {
		logger.Fatal("Failed to initialize server", zap.Error(err))
	}

	// Start server in a goroutine
	go func() {
		logger.Info("Starting HTTP server on port", zap.Int("port", cfg.Port))
		if err := srv.Start(); err != nil {
			logger.Fatal("Server failed", zap.Error(err))
		}
	}()

	// ---- Cloudflare Tunnel (cloudflared) ----
	//
	// When CLOUDFLARE_API_TOKEN and CLOUDFLARE_ZONE_ID are set (propagated
	// from root.key by KNIRVSERVER), provision a Cloudflare Tunnel for
	// gateway.knirv.network and run cloudflared to connect from behind NAT.
	// This follows the same conventions as proxy-penguin.

	var tunnelRunner *cloudflare.TunnelRunner
	tunnelCtx, tunnelCancel := context.WithCancel(context.Background())

	if cfToken := os.Getenv("CLOUDFLARE_API_TOKEN"); cfToken != "" {
		cfZone := os.Getenv("CLOUDFLARE_ZONE_ID")
		if cfZone == "" {
			logger.Warn("CLOUDFLARE_API_TOKEN set but CLOUDFLARE_ZONE_ID missing — skipping Cloudflare Tunnel")
		} else {
			servicePort := cfg.Port
			if servicePort == 0 {
				servicePort = 8081
			}

			tunnelRunner = cloudflare.NewTunnelRunner(cloudflare.TunnelRunnerConfig{
				APIToken:    cfToken,
				ZoneID:      cfZone,
				TunnelName:  "knirv-gateway",
				Hostname:    "gateway.knirv.network",
				ServicePort: servicePort,
			})

			go func() {
				logger.Info("Initialising Cloudflare Tunnel for gateway.knirv.network",
					zap.Int("service_port", servicePort))
				if err := tunnelRunner.Run(tunnelCtx); err != nil {
					logger.Warn("Cloudflare Tunnel failed to start — gateway will operate without tunnel",
						zap.Error(err))
				}
			}()

			// Log the tunnel ID after a brief provisioning window.
			go func() {
				time.Sleep(5 * time.Second)
				if tid := tunnelRunner.TunnelID(); tid != "" {
					logger.Info("Cloudflare Tunnel active",
						zap.String("tunnel_id", tid),
						zap.String("hostname", "gateway.knirv.network"))
				}
			}()
		}
	}

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down KNIRVGATEWAY")

	// Stop Cloudflare Tunnel if running.
	if tunnelRunner != nil {
		logger.Info("Stopping Cloudflare Tunnel")
		tunnelCancel()
		tunnelRunner.Stop()
		tunnelRunner.Wait()
	}

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop HTTP server
	if err := srv.Stop(ctx); err != nil {
		logger.Error("Error stopping server", zap.Error(err))
	}

	logger.Info("KNIRVGATEWAY stopped")
}
