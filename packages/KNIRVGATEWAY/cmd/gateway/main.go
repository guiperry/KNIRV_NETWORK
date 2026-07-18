package main

import (
	"context"
	"fmt"
	"net/url"
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

func canStartCloudflareTunnel(apiToken, zoneID, tunnelToken string) bool {
	return tunnelToken != "" || (apiToken != "" && zoneID != "")
}

func canOwnCloudflareTunnel(cfg *config.Config) bool {
	mode := strings.ToLower(strings.TrimSpace(cfg.NetworkMode))
	isRoot := strings.EqualFold(cfg.ChainNodeRole, "root")
	isBootnode := cfg.ChainIsBootnode || strings.EqualFold(cfg.ChainNodeRole, "bootnode")

	switch {
	case isRoot:
		return mode == "" || mode == "testnet" || mode == "production" || mode == "prod" || mode == "mainnet"
	case isBootnode:
		return mode == "" || mode == "testnet" || mode == "development" || mode == "dev" || mode == "devnet"
	default:
		return false
	}
}

type publicEndpoint struct {
	URL        string
	Hostname   string
	TunnelName string
}

func normalizeUserIDTag(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var tag strings.Builder
	lastWasHyphen := false
	for _, r := range value {
		valid := r >= 'a' && r <= 'z' || r >= '0' && r <= '9'
		if valid {
			tag.WriteRune(r)
			lastWasHyphen = false
			continue
		}
		if tag.Len() > 0 && !lastWasHyphen {
			tag.WriteByte('-')
			lastWasHyphen = true
		}
	}
	normalized := strings.Trim(tag.String(), "-")
	// enterprise- is the longest hostname prefix (11 bytes), leaving 52 bytes
	// for the tag within the DNS label's 63-byte limit.
	if len(normalized) > 52 {
		normalized = strings.TrimRight(normalized[:52], "-")
	}
	return normalized
}

func resolvePublicEndpoint(cfg *config.Config) (publicEndpoint, error) {
	tag := normalizeUserIDTag(cfg.UserIDTag)
	mode := strings.ToLower(strings.TrimSpace(cfg.NetworkMode))
	isRoot := strings.EqualFold(cfg.ChainNodeRole, "root")
	isBootnode := cfg.ChainIsBootnode || strings.EqualFold(cfg.ChainNodeRole, "bootnode")

	// Key-derived node roles own fixed hostname classes. KNIRVSERVER is the
	// sole key reader: root.key produces Root and boot.key produces Bootnode.
	switch {
	case isRoot && (mode == "production" || mode == "prod" || mode == "mainnet"):
		return publicEndpoint{"https://gateway.knirv.network", "gateway.knirv.network", "knirv-gateway"}, nil
	case isRoot && (mode == "" || mode == "testnet"):
		return publicEndpoint{"https://testnet-gateway.knirv.network", "testnet-gateway.knirv.network", "knirv-testnet-gateway"}, nil
	case isBootnode && (mode == "" || mode == "testnet"):
		if tag == "" {
			return publicEndpoint{}, fmt.Errorf("testnet bootnode requires KNIRV_USER_ID_TAG")
		}
		host := fmt.Sprintf("testnet-%s-gateway.knirv.network", tag)
		return publicEndpoint{"https://" + host, host, "knirv-testnet-" + tag + "-gateway"}, nil
	case isBootnode && (mode == "development" || mode == "dev" || mode == "devnet"):
		if tag == "" {
			return publicEndpoint{}, fmt.Errorf("devnet bootnode requires KNIRV_USER_ID_TAG")
		}
		host := fmt.Sprintf("devnet-%s-gateway.knirv.network", tag)
		return publicEndpoint{"https://" + host, host, "knirv-devnet-" + tag + "-gateway"}, nil
	}

	if configured := strings.TrimRight(strings.TrimSpace(cfg.PublicURL), "/"); configured != "" {
		parsed, err := url.Parse(configured)
		if err != nil || parsed.Scheme == "" || parsed.Hostname() == "" {
			return publicEndpoint{}, fmt.Errorf("invalid KNIRV_PUBLIC_URL %q", configured)
		}
		return publicEndpoint{
			URL:        configured,
			Hostname:   parsed.Hostname(),
			TunnelName: "knirv-" + strings.ReplaceAll(parsed.Hostname(), ".", "-"),
		}, nil
	}

	switch {
	case cfg.EnterpriseMode || mode == "enterprise":
		if tag == "" {
			return publicEndpoint{}, fmt.Errorf("enterprise mode requires KNIRV_USER_ID_TAG")
		}
		host := fmt.Sprintf("enterprise-%s.knirv.network", tag)
		return publicEndpoint{"https://" + host, host, "knirv-enterprise-" + tag}, nil
	case mode == "development" || mode == "dev" || mode == "devnet":
		if tag == "" {
			return publicEndpoint{}, fmt.Errorf("development mode requires KNIRV_USER_ID_TAG")
		}
		host := fmt.Sprintf("devnet-%s.knirv.network", tag)
		return publicEndpoint{"https://" + host, host, "knirv-devnet-" + tag}, nil
	case mode == "production" || mode == "prod" || mode == "mainnet":
		return publicEndpoint{"https://gateway.knirv.network", "gateway.knirv.network", "knirv-gateway"}, nil
	default:
		return publicEndpoint{"https://testnet-gateway.knirv.network", "testnet-gateway.knirv.network", "knirv-testnet-gateway"}, nil
	}
}

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

	isBootnode := cfg.ChainIsBootnode || strings.EqualFold(cfg.ChainNodeRole, "bootnode")

	// Resolve bootnode registration metadata before the public endpoint. The
	// registration ID remains useful for DHT identity, but public hostnames are
	// selected exclusively by deployment class and UserIDTag.
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

	publicEndpoint, err := resolvePublicEndpoint(cfg)
	if err != nil {
		logger.Fatal("Failed to resolve public KNIRV endpoint", zap.Error(err))
	}
	cfg.PublicURL = publicEndpoint.URL
	cfg.PublicHost = publicEndpoint.Hostname
	_ = os.Setenv("KNIRV_PUBLIC_URL", publicEndpoint.URL)
	logger.Info("Public KNIRV endpoint resolved",
		zap.String("url", publicEndpoint.URL),
		zap.String("deploymentClass", cfg.NetworkMode))

	// Initialize HTTP server with webgui static directory
	srv, err := server.New(cfg, rt.GetWebGUIStaticPath(), logger)
	if err != nil {
		logger.Fatal("Failed to initialize server", zap.Error(err))
	}

	// ---- nginx TLS reverse-proxy provisioning ----
	//
	// Only runs in production mode. nginx is installed if absent, configured to
	// terminate TLS for the resolved production public host, and proxies all
	// traffic to this KNIRVGATEWAY instance.
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
	// The public hostname is resolved above from the key-derived role and
	// deployment class.
	// Every server, oracle, and subsystem API shares this one gateway tunnel.
	//
	// Tunnels are only started when CLOUDFLARE_API_TOKEN and CLOUDFLARE_ZONE_ID
	// are present. CLOUDFLARE_GATEWAY_TUNNEL_TOKEN / CLOUDFLARE_ORACLE_TUNNEL_TOKEN
	// are optional: when set, cloudflared uses the pre-provisioned token instead
	// of running full API provisioning on startup.

	var tunnelRunners []*cloudflare.TunnelRunner
	tunnelCtx, tunnelCancel := context.WithCancel(context.Background())

	cfGatewayToken := os.Getenv("CLOUDFLARE_GATEWAY_TUNNEL_TOKEN")
	cfOracleToken := os.Getenv("CLOUDFLARE_ORACLE_TUNNEL_TOKEN")
	if cfGatewayToken == "" && cfOracleToken != "" {
		// Migration compatibility for root keys created before the unified public
		// gateway: use the former oracle token when no gateway token is present.
		cfGatewayToken = cfOracleToken
	}
	if cfAPIToken := os.Getenv("CLOUDFLARE_API_TOKEN"); cfAPIToken != "" || cfGatewayToken != "" || cfOracleToken != "" {
		if !canOwnCloudflareTunnel(cfg) {
			logger.Info("Cloudflare Tunnel ownership denied for node role and network",
				zap.String("role", cfg.ChainNodeRole),
				zap.String("network", cfg.NetworkMode))
		} else {
			cfZone := os.Getenv("CLOUDFLARE_ZONE_ID")
			if cfZone == "" && cfGatewayToken == "" && cfOracleToken == "" {
				logger.Warn("CLOUDFLARE_API_TOKEN set but CLOUDFLARE_ZONE_ID missing — skipping Cloudflare Tunnels")
			} else {
				servicePort := cfg.Port
				if servicePort == 0 {
					servicePort = 8081
				}

				startTunnel := func(name, hostname string, port int, token string) {
					if !canStartCloudflareTunnel(cfAPIToken, cfZone, token) {
						logger.Warn("Cloudflare Tunnel skipped — tunnel token or API credentials required",
							zap.String("hostname", hostname), zap.String("tunnel", name))
						return
					}
					runner := cloudflare.NewTunnelRunner(cloudflare.TunnelRunnerConfig{
						APIToken:    cfAPIToken,
						ZoneID:      cfZone,
						AccountID:   os.Getenv("CLOUDFLARE_ACCOUNT_ID"),
						TunnelToken: token,
						TunnelName:  name,
						Hostname:    hostname,
						ServicePort: port,
					})
					logger.Info("Starting Cloudflare Tunnel",
						zap.String("hostname", hostname),
						zap.String("tunnel", name),
						zap.Int("port", port))
					if err := runner.Run(tunnelCtx); err != nil {
						logger.Warn("Cloudflare Tunnel failed",
							zap.String("hostname", hostname), zap.Error(err))
						return
					}
					tunnelRunners = append(tunnelRunners, runner)
					logger.Info("Cloudflare Tunnel initialized",
						zap.String("publicUrl", publicEndpoint.URL))
				}

				startTunnel(publicEndpoint.TunnelName, publicEndpoint.Hostname, servicePort, cfGatewayToken)
			}
		}
	}

	// Publish the HTTP API only after cloudflared has been initialized (or
	// explicitly skipped because this is a credential-less local run). This
	// prevents device authorization from advertising a stale local origin while
	// tunnel creation is still in progress.
	go func() {
		logger.Info("Starting HTTP server on port", zap.Int("port", cfg.Port))
		if err := srv.Start(); err != nil {
			logger.Fatal("Server failed", zap.Error(err))
		}
	}()

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
	case isRoot || isBootnode:
		return strings.TrimSpace(cfg.PublicHost)
	default:
		return ""
	}
}
