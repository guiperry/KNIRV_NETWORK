package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/cloudflare"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/config"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/embedded"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/nginx"
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
		zap.String("network", cfg.NetworkMode),
		zap.Int("port", cfg.Port),
		zap.String("chainID", cfg.ChainID),
		zap.String("nodeRole", cfg.ChainNodeRole),
	)

	// Initialize runtime and extract embedded files (oracle binary removed)
	rt, err := runtime.NewRuntime(logger, embedded.WebGUIFS, nil)
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

	isProduction := strings.EqualFold(cfg.NetworkMode, "production")
	isRoot := strings.EqualFold(cfg.ChainNodeRole, "root")
	isBootnode := cfg.ChainIsBootnode || strings.EqualFold(cfg.ChainNodeRole, "bootnode")

	// Resolve the public hostname early so server subsystems can reuse it.
	// Bootnodes advertise a hyphenated Cloudflare tunnel hostname, but the
	// registry still receives the resolved public IP address.
	nodeRegID := cfg.NodeRegistrationID
	if nodeRegID == "" && isBootnode && cfg.CloudflareD1DatabaseID != "" {
		hostname, _ := os.Hostname()
		cfAPI := cloudflare.NewAPI(os.Getenv("CLOUDFLARE_API_TOKEN"), os.Getenv("CLOUDFLARE_ZONE_ID"), os.Getenv("CLOUDFLARE_ACCOUNT_ID"))
		if id, err := cfAPI.GetNodeRegistrationID(cfg.CloudflareD1DatabaseID, hostname); err != nil {
			logger.Warn("D1 bootnode registration ID lookup failed", zap.Error(err))
		} else {
			nodeRegID = id
			logger.Info("Bootnode registration ID resolved from D1", zap.String("id", nodeRegID))
		}
	}
	if nodeRegID != "" {
		cfg.NodeRegistrationID = nodeRegID
	}

	if currentHost := strings.TrimSpace(cfg.PublicHost); currentHost == "" || strings.EqualFold(currentHost, "localhost") {
		switch {
		case isProduction && isRoot:
			cfg.PublicHost = "gateway.knirv.network"
		case isProduction && isBootnode && nodeRegID != "":
			cfg.PublicHost = fmt.Sprintf("gateway-%s.knirv.network", nodeRegID)
		case !isProduction && isRoot:
			cfg.PublicHost = "testnet-gateway.knirv.network"
		case !isProduction && isBootnode && nodeRegID != "":
			cfg.PublicHost = fmt.Sprintf("testnet-gateway-%s.knirv.network", nodeRegID)
		}
	}

	// Initialize HTTP server with webgui static directory
	srv, err := server.New(cfg, rt.GetWebGUIStaticPath(), logger)
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

	// ---- nginx TLS reverse-proxy provisioning ----
	//
	// Only runs in production mode. nginx is installed if absent, configured to
	// terminate TLS for knirv.network (root) or gateway-{N}.knirv.network (bootnode)
	// and proxy all traffic to this KNIRVGATEWAY instance.
	// Oracle traffic is NOT separately exposed — it flows through KNIRVGATEWAY's
	// /api/oracle/* proxy internally.
	//
	// Non-fatal: Cloudflare Tunnels remain the primary public ingress path.
	if strings.EqualFold(cfg.NetworkMode, "production") {
		nginxDomain := resolveNginxDomain(cfg)
		if nginxDomain != "" {
			certFile := os.Getenv("KNIRV_TLS_CERT_FILE")
			keyFile := os.Getenv("KNIRV_TLS_KEY_FILE")
			if certFile == "" {
				certFile = "/etc/knirvserver/tls/origin.crt"
			}
			if keyFile == "" {
				keyFile = "/etc/knirvserver/tls/origin.key"
			}
			if err := nginx.EnsureNginx(nginxDomain, cfg.Port, certFile, keyFile, logger); err != nil {
				logger.Warn("nginx provisioning failed — continuing without nginx",
					zap.String("domain", nginxDomain), zap.Error(err))
			}
		} else {
			logger.Info("nginx provisioning skipped — node role not root or bootnode")
		}
	} else {
		logger.Info("nginx provisioning skipped in testnet/non-production mode")
	}

	// ---- Cloudflare Tunnel setup ----
	//
	// cloudflared is auto-installed if not present on the system.
	//
	// Tunnel hostname convention (N = KNIRV_NODE_REGISTRATION_ID from D1):
	//   Production + root:     gateway.knirv.network  + gateway-oracle.knirv.network
	//   Production + bootnode: gateway-{N}.knirv.network
	//   Testnet    + root:     testnet-gateway.knirv.network + testnet-gateway-oracle.knirv.network
	//   Testnet    + bootnode: testnet-gateway-{N}.knirv.network
	//
	// The oracle tunnel points to the SAME local port as the gateway tunnel (KNIRVGATEWAY),
	// NOT to port 1317. Oracle has no independent TCP listener; all oracle traffic flows
	// through KNIRVGATEWAY's /api/oracle/* socket proxy.
	//
	// Tunnels are only started when CLOUDFLARE_API_TOKEN and CLOUDFLARE_ZONE_ID
	// are present. CLOUDFLARE_GATEWAY_TUNNEL_TOKEN / CLOUDFLARE_ORACLE_TUNNEL_TOKEN
	// are optional: when set, cloudflared uses the pre-provisioned token instead
	// of running full API provisioning on startup.

	var tunnelRunners []*cloudflare.TunnelRunner
	tunnelCtx, tunnelCancel := context.WithCancel(context.Background())

	if cfAPIToken := os.Getenv("CLOUDFLARE_API_TOKEN"); cfAPIToken != "" {
		cfZone := os.Getenv("CLOUDFLARE_ZONE_ID")
		if cfZone == "" {
			logger.Warn("CLOUDFLARE_API_TOKEN set but CLOUDFLARE_ZONE_ID missing — skipping Cloudflare Tunnels")
		} else {
			servicePort := cfg.Port
			if servicePort == 0 {
				servicePort = 8081
			}

			startTunnel := func(name, hostname string, port int, token string) {
				runner := cloudflare.NewTunnelRunner(cloudflare.TunnelRunnerConfig{
					APIToken:    cfAPIToken,
					ZoneID:      cfZone,
					AccountID:   os.Getenv("CLOUDFLARE_ACCOUNT_ID"),
					TunnelToken: token,
					TunnelName:  name,
					Hostname:    hostname,
					ServicePort: port,
				})
				tunnelRunners = append(tunnelRunners, runner)
				go func() {
					logger.Info("Starting Cloudflare Tunnel",
						zap.String("hostname", hostname),
						zap.String("tunnel", name),
						zap.Int("port", port))
					if err := runner.Run(tunnelCtx); err != nil {
						logger.Warn("Cloudflare Tunnel failed",
							zap.String("hostname", hostname), zap.Error(err))
					}
				}()
			}

			switch {
			case isProduction && isRoot:
				startTunnel("knirv-gateway", "gateway.knirv.network", servicePort,
					os.Getenv("CLOUDFLARE_GATEWAY_TUNNEL_TOKEN"))
				startTunnel("knirv-gateway-oracle", "gateway-oracle.knirv.network", servicePort,
					os.Getenv("CLOUDFLARE_ORACLE_TUNNEL_TOKEN"))

			case isProduction && isBootnode:
				if nodeRegID == "" {
					logger.Warn("Production bootnode tunnel skipped — set KNIRV_NODE_REGISTRATION_ID or CLOUDFLARE_D1_DATABASE_ID")
				} else {
					startTunnel(
						fmt.Sprintf("knirv-gateway-%s", nodeRegID),
						fmt.Sprintf("gateway-%s.knirv.network", nodeRegID),
						servicePort,
						os.Getenv("CLOUDFLARE_GATEWAY_TUNNEL_TOKEN"),
					)
				}

			case !isProduction && isRoot:
				startTunnel("knirv-testnet-gateway", "testnet-gateway.knirv.network", servicePort,
					os.Getenv("CLOUDFLARE_GATEWAY_TUNNEL_TOKEN"))
				startTunnel("knirv-testnet-gateway-oracle", "testnet-gateway-oracle.knirv.network", servicePort,
					os.Getenv("CLOUDFLARE_ORACLE_TUNNEL_TOKEN"))

			case !isProduction && isBootnode:
				if nodeRegID == "" {
					logger.Warn("Testnet bootnode tunnel skipped — set KNIRV_NODE_REGISTRATION_ID or CLOUDFLARE_D1_DATABASE_ID")
				} else {
					startTunnel(
						fmt.Sprintf("knirv-testnet-gateway-%s", nodeRegID),
						fmt.Sprintf("testnet-gateway-%s.knirv.network", nodeRegID),
						servicePort,
						os.Getenv("CLOUDFLARE_GATEWAY_TUNNEL_TOKEN"),
					)
				}

			default:
				logger.Info("No Cloudflare Tunnel for this node role — skipping",
					zap.String("role", cfg.ChainNodeRole),
					zap.String("network", cfg.NetworkMode))
			}
		}
	}

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down KNIRVGATEWAY")

	// Stop all Cloudflare Tunnels.
	if len(tunnelRunners) > 0 {
		logger.Info("Stopping Cloudflare Tunnels", zap.Int("count", len(tunnelRunners)))
		tunnelCancel()
		for _, r := range tunnelRunners {
			r.Stop()
		}
		for _, r := range tunnelRunners {
			r.Wait()
		}
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

// resolveNginxDomain returns the public domain nginx should serve for this node,
// or empty string when nginx provisioning should be skipped.
func resolveNginxDomain(cfg *config.Config) string {
	isRoot := strings.EqualFold(cfg.ChainNodeRole, "root")
	isBootnode := cfg.ChainIsBootnode || strings.EqualFold(cfg.ChainNodeRole, "bootnode")

	switch {
	case isRoot:
		return "knirv.network"
	case isBootnode && cfg.NodeRegistrationID != "":
		return fmt.Sprintf("gateway-%s.knirv.network", cfg.NodeRegistrationID)
	default:
		return ""
	}
}
