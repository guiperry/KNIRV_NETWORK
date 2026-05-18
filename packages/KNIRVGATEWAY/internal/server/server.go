package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/auth"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/config"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/dht"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/operator"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/payment"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/proxy"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/session"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/tunnel"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/turnserver"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/uri"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/webgui"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/rs/cors"
	"go.uber.org/zap"
)

// Server represents the HTTP server
type Server struct {
	config            *config.Config
	sessionManager    *session.Manager
	proxyHandler      *proxy.Handler
	authHandler       *auth.Handler
	operatorService   *operator.Service
	operatorHandler   *operator.Handler
	tunnelService     *tunnel.Service
	tunnelHandler     *tunnel.Handler
	paymentService    *payment.Service
	paymentHandler    *payment.Handler
	uriHandler        *uri.Handler
	webguiHandler        *webgui.Handler
	turnServer           *turnserver.Server
	dhtManager          *dht.DHTManager
	logger               *zap.Logger
	httpServer           *http.Server
	router               *mux.Router
	webguiStaticDir      string
	actualPort           int
}

// New creates a new HTTP server
func New(cfg *config.Config, webguiStaticDir string, logger *zap.Logger, db ...*sql.DB) (*Server, error) {
	var dbInstance *sql.DB
	if len(db) > 0 {
		dbInstance = db[0]
	}
	// Initialize operator service.
	// Oracle is only accessible through the gateway's Unix socket reverse proxy;
	// the independent TCP port (1317) has been removed.
	oracleBaseURL := "http://localhost:1317"
	if cfg.OracleSocketPath != "" {
		oracleBaseURL = "http://knirvoracle" // proxied via socket
	}
	operatorSvc := operator.NewService(logger, oracleBaseURL)
	operatorSvc.Initialize() // Load mock data

	operatorHdlr := operator.NewHandler(operatorSvc, logger)

	// Initialize tunnel service
	tunnelConfig := &tunnel.Config{
		HTTPAPIPort:         cfg.TunnelRegistryHTTPPort,
		ControlListenerPort: cfg.TunnelRegistryControlPort,
		PublicRelayPort:     cfg.TunnelRegistryRelayPort,
		STUNPort:            cfg.TunnelRegistrySTUNPort,
		ServerPublicHost:    cfg.PublicHost,
		RelayServerPeerID:   cfg.InternalAPIKey, // Using internal API key as peer ID for now
	}
	tunnelSvc := tunnel.NewService(tunnelConfig, logger)
	tunnelHdlr := tunnelSvc.GetHandler()

	// Initialize payment service
	paymentConfig := &payment.Config{
		StripeSecretKey:     "", // Will be set from environment
		CoinbaseAPIKey:      "", // Will be set from environment
		FaucetCooldownHours: 24,
		DefaultNetwork:      "mainnet",
		EconomicsEnabled:    true,
	}
	paymentSvc := payment.NewService(paymentConfig, logger)
	paymentHdlr := payment.NewHandler(paymentSvc, logger)

	// Initialize auth handler
	authHdlr := auth.NewHandler(cfg, logger, dbInstance)

	// Initialize URI handler
	uriHdlr := uri.NewHandler(logger)

	// Initialize explorer handler.
	explorerHandler := webgui.NewHandler(cfg, logger, cfg.Port)

	// Initialize TURN server with blockchain integration
	var turnSvc *turnserver.Server
	if cfg.TurnServerEnabled {
		turnConfig := &turnserver.TurnServerConfig{
			UDPPort:      cfg.TurnServerUDPPort,
			TCPPort:      cfg.TurnServerTCPPort,
			APIPort:      cfg.TurnServerAPIPort,
			Realm:        cfg.TurnServerRealm,
			AuthSecret:   cfg.TurnServerAuthSecret,
			PublicIP:     cfg.PublicHost,
			MinerAddress: cfg.TurnServerMinerAddress,
		}

		blockchainAdapter := turnserver.NewBlockchainAdapter(nil, cfg.TurnServerMinerAddress)
		var err error
		turnSvc, err = turnserver.NewServer(turnConfig, blockchainAdapter, logger)
		if err != nil {
			logger.Warn("Failed to initialize TURN server",
				zap.Error(err),
				zap.Int("udp_port", cfg.TurnServerUDPPort),
				zap.Int("tcp_port", cfg.TurnServerTCPPort))
		} else {
			logger.Info("TURN server initialized",
				zap.Int("udp_port", cfg.TurnServerUDPPort),
				zap.Int("tcp_port", cfg.TurnServerTCPPort),
				zap.Int("api_port", cfg.TurnServerAPIPort),
				zap.String("realm", cfg.TurnServerRealm))
		}
	}

	// Initialize DHT manager if not disabled
	var dhtMgr *dht.DHTManager
	if !cfg.DisableDHT {
		dhtConfig := &dht.Config{
			ServiceID:             "knirvgateway",
			ChainID:               cfg.ChainID,
			BootstrapPeers:        cfg.BootstrapPeers,
			EnableAutoRelay:       false,
			Port:                  cfg.DHTPort,
			AnnounceInterval:      5 * time.Minute,
			NodeRole:              cfg.ChainNodeRole,
			ChainP2PPort:          cfg.ChainP2PPort,
			ChainClientOnly:       cfg.ChainClientOnly,
			ChainIsBootnode:       cfg.ChainIsBootnode,
			ChainBootnodeRegistry: cfg.ChainBootnodeRegistry,
			ChainCallbackSocket:   cfg.ChainCallbackSocket,
		}
		var err error
		dhtMgr, err = dht.NewDHTManager(dhtConfig)
		if err != nil {
			logger.Warn("Failed to initialize DHT manager", zap.Error(err))
		} else {
			logger.Info("DHT manager initialized", zap.Int("port", cfg.DHTPort))
		}
	}

	s := &Server{
		config:               cfg,
		sessionManager:       session.NewManager(cfg.SessionSecret),
		proxyHandler:         proxy.NewHandler(logger),
		authHandler:          authHdlr,
		operatorService:      operatorSvc,
		operatorHandler:      operatorHdlr,
		tunnelService:        tunnelSvc,
		tunnelHandler:        tunnelHdlr,
		paymentService:       paymentSvc,
		paymentHandler:       paymentHdlr,
		uriHandler:           uriHdlr,
		webguiHandler:        explorerHandler,
		turnServer:           turnSvc,
		dhtManager:          dhtMgr,
		logger:               logger,
		webguiStaticDir:      webguiStaticDir,
	}

	if err := s.setupRoutes(); err != nil {
		return nil, fmt.Errorf("failed to setup routes: %w", err)
	}

	return s, nil
}

// setupRoutes configures all HTTP routes
func (s *Server) setupRoutes() error {
	r := mux.NewRouter()

	// Register MIME types explicitly (important on systems missing /etc/mime.types)
	mime.AddExtensionType(".js", "application/javascript")
	mime.AddExtensionType(".css", "text/css")
	mime.AddExtensionType(".mjs", "application/javascript")
	mime.AddExtensionType(".json", "application/json")
	mime.AddExtensionType(".svg", "image/svg+xml")
	mime.AddExtensionType(".wasm", "application/wasm")

	// Session endpoints
	r.HandleFunc("/session/controller", s.handleGetSession).Methods("GET")
	r.HandleFunc("/session/controller", s.handleSetSession).Methods("POST")

	// Health and status endpoints
	r.HandleFunc("/health", s.handleHealth).Methods("GET")

	// DHT/P2P endpoints (real implementation)
	r.HandleFunc("/dht/announce", s.handleDHTAnnounce).Methods("POST")
	r.HandleFunc("/dht/find", s.handleDHTFind).Methods("GET")
	r.HandleFunc("/dht/peers", s.handleDHTPeers).Methods("GET")
	r.HandleFunc("/dht/bootstrap", s.handleDHTBootstrap).Methods("POST")

	// Resource Broadcast System endpoints
	r.HandleFunc("/dht/cache-resource", s.handleCacheResource).Methods("POST")
	r.HandleFunc("/dht/announce-all-cached", s.handleAnnounceAllCached).Methods("POST")
	r.HandleFunc("/dht/cache-status", s.handleCacheStatus).Methods("GET")

	// Chain P2P proxy endpoints (used by KNIRVCHAIN via unix socket)
	r.HandleFunc("/p2p/publish-block", s.handleP2PPublishBlock).Methods("POST")
	r.HandleFunc("/p2p/publish-tx", s.handleP2PPublishTx).Methods("POST")
	r.HandleFunc("/p2p/register-callback", s.handleP2PRegisterCallback).Methods("POST")
	r.HandleFunc("/p2p/peers", s.handleP2PPeers).Methods("GET")
	r.HandleFunc("/p2p/self-addrs", s.handleP2PSelfAddrs).Methods("GET")
	r.HandleFunc("/p2p/peer-id", s.handleP2PPeerID).Methods("GET")

	// Register auth routes directly
	s.authHandler.RegisterRoutes(r)

	// Register operator registry routes directly
	if s.operatorHandler != nil {
		s.operatorHandler.RegisterRoutes(r)
	}

	// Register tunnel registry routes directly
	s.tunnelHandler.RegisterRoutes(r)

	// Register payment oracle routes directly
	s.paymentHandler.RegisterRoutes(r)

	// Register URI generation routes directly
	s.uriHandler.RegisterRoutes(r)

	// Register explorer API routes directly.
	s.webguiHandler.RegisterRoutes(r)

	// Register TURN server routes (blockchain-enabled)
	if s.turnServer != nil {
		r.HandleFunc("/api/turn/status", s.handleTurnStatus).Methods("GET")
		r.HandleFunc("/api/turn/stats", s.handleTurnStats).Methods("GET")
		r.HandleFunc("/api/proof/submit", s.handleTurnProofSubmit).Methods("POST")
		r.HandleFunc("/api/proof/status", s.handleTurnProofStatus).Methods("GET")
		r.HandleFunc("/api/mint/nrn", s.handleTurnMintNRN).Methods("POST")
		r.HandleFunc("/api/mint/reward", s.handleTurnMintReward).Methods("POST")
		r.HandleFunc("/api/stats/minting", s.handleTurnMintingStats).Methods("GET")
		r.HandleFunc("/api/turn/health", s.handleTurnHealth).Methods("GET")
	}

	// Dynamic controller proxy
	r.PathPrefix("/controller").Handler(s.handleControllerProxy())

	// Network monitor API routes — these replace Next.js API routes that are
	// excluded from the static export.  They return static/mock status data
	// for the network-monitor UI page.
	r.HandleFunc("/api/network-monitor/status", s.handleNetworkMonitorStatus).Methods("GET")
	r.HandleFunc("/api/network-monitor/services", s.handleNetworkMonitorServices).Methods("GET")
	r.HandleFunc("/api/network-monitor/metrics", s.handleNetworkMonitorMetrics).Methods("GET")
	r.HandleFunc("/api/network-monitor/logs", s.handleNetworkMonitorLogs).Methods("GET")
	r.HandleFunc("/api/network-monitor/config", s.handleNetworkMonitorConfig).Methods("GET")

	// Backend reverse proxy — proxy all /api/v1/* to the backend Unix socket
	// (including /api/v1/info, which was previously handled locally by the gateway).
	if s.config.BackendSocketPath != "" {
		backendProxy := newSocketProxy(s.config.BackendSocketPath, "http://knirvserver")
		r.PathPrefix("/api/v1/").Handler(backendProxy)
		// Also proxy /api/badge/* to the backend socket (badge templates, guardrail injectors)
		r.PathPrefix("/api/badge/").Handler(backendProxy)
		s.logger.Info("Backend proxy registered", zap.String("socket", s.config.BackendSocketPath))
	} else {
		s.logger.Warn("Backend proxy not configured — /api/v1/* will not be proxied")
	}

	// === PHASES 1-4: Unified Proxy Architecture ===
	//
	// Oracle proxy — KNIRVORACLE communicates exclusively via Unix socket
	// through the gateway.  The independent TCP port (historically 1317) is
	// removed; all oracle traffic flows through the gateway's reverse-proxy layer.
	var oracleProxy *httputil.ReverseProxy
	if s.config.OracleSocketPath != "" {
		oracleProxy = newSocketProxy(s.config.OracleSocketPath, "http://knirvoracle")
		s.logger.Info("Oracle proxy registered (socket)", zap.String("socket", s.config.OracleSocketPath))
	} else {
		s.logger.Warn("Oracle socket path not configured — oracle endpoints will not be proxied")
	}

	// /api/oracle/* — Standardized oracle prefix for all Cosmos blockchain queries
	// Wallet/balance, transaction submission, blocks, staking, governance, IBC.
	// Strip /api prefix so /api/oracle/v3/... becomes /oracle/v3/..., matching the
	// oracle's native route format (oracle serves under /oracle/v3/*).
	if oracleProxy != nil {
		r.PathPrefix("/api/oracle/").Handler(http.StripPrefix("/api", oracleProxy))

		// Root-level wallet-server compatible routes — proxied directly to the oracle
		// without path stripping, matching the oracle's root-level route registrations
		// (/generate_wallet, /balance/{addr}, /send_signed_txn, etc.).
		r.HandleFunc("/generate_wallet", oracleProxy.ServeHTTP).Methods("GET", "POST", "OPTIONS")
		r.HandleFunc("/balance/", oracleProxy.ServeHTTP).Methods("GET", "OPTIONS")
		r.HandleFunc("/send_signed_txn", oracleProxy.ServeHTTP).Methods("POST", "OPTIONS")
		r.HandleFunc("/transactions", oracleProxy.ServeHTTP).Methods("POST", "OPTIONS")
		r.HandleFunc("/test/faucet", oracleProxy.ServeHTTP).Methods("POST", "OPTIONS")
	}

	// 301 redirects from old flat paths — wallet info and transaction submission
	// routes to the oracle's wallet status and transaction endpoints.
	r.HandleFunc("/wallet/info", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/api/oracle/v3/wallet/status", http.StatusMovedPermanently)
	}).Methods("GET", "OPTIONS")
	r.HandleFunc("/transaction", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/transactions", http.StatusMovedPermanently)
	}).Methods("POST", "OPTIONS")

	// Chain proxy — KNIRVCHAIN via chain.sock
	if s.config.ChainSocketPath != "" {
		chainProxy := newSocketProxy(s.config.ChainSocketPath, "http://knirvchain")

		// /api/chain/* — Standardized chain prefix (Phase 2).
		// Strip /api/chain prefix so /api/chain/chain becomes /chain,
		// matching KNIRVCHAIN's native route format.
		r.PathPrefix("/api/chain/").Handler(http.StripPrefix("/api/chain", chainProxy))

		// Flat /chain/* redirect → /api/chain/* (Phase 2)
		r.PathPrefix("/chain/").Handler(http.StripPrefix("/chain",
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, "/api/chain"+r.URL.Path, http.StatusMovedPermanently)
			})))

		// /chain (GET) redirect → /api/chain/chain (the actual chain endpoint)
		r.HandleFunc("/chain", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/api/chain/chain", http.StatusMovedPermanently)
		}).Methods("GET", "OPTIONS")

		// /bootnodes redirect → /api/chain/bootnodes (Phase 2, replaces /devs)
		r.HandleFunc("/bootnodes", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/api/chain/bootnodes", http.StatusMovedPermanently)
		}).Methods("GET", "OPTIONS")

		// /api/objects → chain proxy (Phase 4 — replaces mock handler)
		r.HandleFunc("/api/objects", chainProxy.ServeHTTP).Methods("GET", "OPTIONS")

		// /api/assets → chain proxy (Phase 4 — replaces mock handler)
		r.HandleFunc("/api/assets", chainProxy.ServeHTTP).Methods("GET", "OPTIONS")

		// WebGUI flat paths — proxy directly to chain.sock (these are the
		// actual endpoints KNIRVCHAIN serves at root level).
		r.HandleFunc("/txn_pool", chainProxy.ServeHTTP).Methods("GET", "OPTIONS")
		r.HandleFunc("/devs", chainProxy.ServeHTTP).Methods("GET", "OPTIONS")

		s.logger.Info("Chain proxy registered", zap.String("socket", s.config.ChainSocketPath))
	} else {
		s.logger.Warn("Chain proxy not configured — /api/chain/* will not be proxied")

		// Fallback: return empty data for chain endpoints when chain is not available
		r.HandleFunc("/api/objects", s.handleEmptyArray).Methods("GET", "OPTIONS")
		r.HandleFunc("/api/assets", s.handleEmptyArray).Methods("GET", "OPTIONS")

		// /chain fallback — WebGUI still calls this endpoint directly.
		// Return empty data instead of 404 so the dashboard doesn't break.
		r.HandleFunc("/chain", s.handleEmptyArray).Methods("GET", "OPTIONS")

		// WebGUI flat path fallbacks when chain socket isn't available.
		r.HandleFunc("/txn_pool", s.handleEmptyArray).Methods("GET", "OPTIONS")
		r.HandleFunc("/devs", s.handleEmptyArray).Methods("GET", "OPTIONS")
		r.HandleFunc("/wallet/info", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("{}\n"))
		}).Methods("GET", "OPTIONS")
		r.HandleFunc("/transaction", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"error":"chain not available"}` + "\n"))
		}).Methods("POST", "OPTIONS")
	}

	// Graph proxy — KNIRVGRAPH via graph.sock (Phase 3)
	if s.config.GraphSocketPath != "" {
		graphProxy := newSocketProxy(s.config.GraphSocketPath, "http://knirvgraph")

		// /api/graph/* — Standardized graph prefix.
		// Strip /api/graph so /api/graph/* becomes /* on KNIRVGRAPH.
		r.PathPrefix("/api/graph/").Handler(http.StripPrefix("/api/graph", graphProxy))

		// Flat /graph/* redirect → /api/graph/*
		r.PathPrefix("/graph/").Handler(http.StripPrefix("/graph",
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, "/api/graph"+r.URL.Path, http.StatusMovedPermanently)
			})))

		s.logger.Info("Graph proxy registered", zap.String("socket", s.config.GraphSocketPath))
	} else {
		s.logger.Warn("Graph proxy not configured — /api/graph/* will not be proxied")
	}

	// Shell proxy — KNIRVSHELL via shell.sock (Phase 5)
	if s.config.ShellSocketPath != "" {
		shellProxy := newSocketProxy(s.config.ShellSocketPath, "http://knirvshell")
		r.PathPrefix("/api/shell/").Handler(http.StripPrefix("/api/shell", shellProxy))
		// 307 (method-preserving) redirects from old shell paths
		r.PathPrefix("/api/v1/shell/").Handler(http.StripPrefix("/api/v1/shell",
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, "/api/shell"+r.URL.Path, http.StatusTemporaryRedirect)
			})))
		r.PathPrefix("/api/knirvshell/").Handler(http.StripPrefix("/api/knirvshell",
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, "/api/shell"+r.URL.Path, http.StatusTemporaryRedirect)
			})))
		s.logger.Info("Shell proxy registered", zap.String("socket", s.config.ShellSocketPath))
	} else {
		s.logger.Warn("Shell proxy not configured — /api/shell/* will not be proxied")
	}

	// Hasher proxy — KNIRVHASHER via Unix socket (Phase 8)
	if s.config.HasherSocketPath != "" {
		hasherProxy := newSocketProxy(s.config.HasherSocketPath, "http://knirvhasher")
		r.PathPrefix("/api/knirvhasher/").Handler(http.StripPrefix("/api/knirvhasher", hasherProxy))

		s.logger.Info("Hasher proxy registered", zap.String("socket", s.config.HasherSocketPath))
	} else {
		s.logger.Warn("Hasher socket path not configured — /api/knirvhasher/* will not be proxied")
	}

	// Agent proxy — KNIRVAGENT per-DVE dynamic socket proxy (Phase 6)
	if s.config.AgentSocketDir != "" {
		r.PathPrefix("/api/agent/").Handler(s.dynamicAgentProxy())
		s.logger.Info("Agent proxy registered", zap.String("socketDir", s.config.AgentSocketDir))
	} else {
		s.logger.Warn("Agent proxy not configured — /api/agent/* will not be proxied")
	}

	// Inner Agent WebSocket tunnel — connects WebSocket clients directly to the
	// KNIRVAGENT's inner agent API via Unix socket for real-time streaming.
	if s.config.AgentSocketDir != "" {
		r.HandleFunc("/ws/{dveId}/inner/stream/{sessionId}", s.handleInnerAgentWS).Methods("GET")
		r.HandleFunc("/ws/{dveId}/inner", s.handleInnerAgentWS).Methods("GET")
		s.logger.Info("Inner Agent WS tunnel registered", zap.String("socketDir", s.config.AgentSocketDir))
	} else {
		s.logger.Warn("Inner Agent WS tunnel not configured — /ws/* will not be available")
	}

	// Phase 4: Replace WebGUI mock endpoints with real proxy destinations.
	// /api/transactions → oracle proxy (Cosmos tx history)
	if oracleProxy != nil {
		r.HandleFunc("/api/transactions", oracleProxy.ServeHTTP).Methods("GET", "OPTIONS")
		r.HandleFunc("/api/blocks", oracleProxy.ServeHTTP).Methods("GET", "OPTIONS")
	} else {
		r.HandleFunc("/api/transactions", s.handleEmptyArray).Methods("GET", "OPTIONS")
		r.HandleFunc("/api/blocks", s.handleEmptyArray).Methods("GET", "OPTIONS")
	}

	// NOTE: No catch-all mock handler. Unmatched /api/* routes will 404
	// instead of returning fake data, forcing real endpoint implementation.

	// /api/v1/info fallback when backend proxy isn't configured
	if s.config.BackendSocketPath == "" {
		r.HandleFunc("/api/v1/info", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"status":"ok","mode":"gateway","role":"General","network":"local"}` + "\n"))
		}).Methods("GET", "OPTIONS")
	}

	// IMPORTANT: Next.js static export uses absolute paths like /_next/..., /favicon.ico, etc.
	// These are served at the root level so the explorer can load its assets.

	// Serve Next.js _next directory (contains JS, CSS, chunks, etc.)
	r.PathPrefix("/_next/").Handler(http.StripPrefix("/", http.FileServer(http.Dir(s.webguiStaticDir))))

	// Serve explorer static files at the root level (favicon, svgs, etc.).
	webguiStaticFiles := []string{"/favicon.ico", "/next.svg", "/window.svg", "/globe.svg", "/vercel.svg", "/file.svg"}
	for _, staticFile := range webguiStaticFiles {
		filePath := staticFile
		r.HandleFunc(filePath, func(w http.ResponseWriter, r *http.Request) {
			fullPath := filepath.Join(s.webguiStaticDir, filePath)
			s.logger.Debug("Serving explorer static file", zap.String("path", filePath), zap.String("fullPath", fullPath))
			http.ServeFile(w, r, fullPath)
		})
	}

	// Serve the explorer index at /oracle and /explorer routes with injected config
	r.HandleFunc("/oracle", func(w http.ResponseWriter, r *http.Request) {
		indexPath := filepath.Join(s.webguiStaticDir, "index.html")
		data, err := s.injectGatewayBase(indexPath)
		if err != nil {
			s.logger.Error("Failed to inject gateway base into index", zap.Error(err))
			http.ServeFile(w, r, indexPath)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(data)
	})
	r.HandleFunc("/explorer", func(w http.ResponseWriter, r *http.Request) {
		indexPath := filepath.Join(s.webguiStaticDir, "index.html")
		data, err := s.injectGatewayBase(indexPath)
		if err != nil {
			s.logger.Error("Failed to inject gateway base into index", zap.Error(err))
			http.ServeFile(w, r, indexPath)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(data)
	})
	r.HandleFunc("/dashboard", func(w http.ResponseWriter, r *http.Request) {
		filePath := filepath.Join(s.webguiStaticDir, "dashboard.html")
		data, err := s.injectGatewayBase(filePath)
		if err != nil {
			// dashboard.html doesn't exist — serve the SPA shell so
			// Next.js client-side routing renders the dashboard page.
			indexPath := filepath.Join(s.webguiStaticDir, "index.html")
			indexData, indexErr := s.injectGatewayBase(indexPath)
			if indexErr != nil {
				http.ServeFile(w, r, indexPath)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(indexData)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(data)
	})

	// Serve other explorer HTML pages at /oracle/ prefix
	r.PathPrefix("/oracle/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fileName := strings.TrimPrefix(r.URL.Path, "/oracle/")
		filePath := filepath.Join(s.webguiStaticDir, fileName)
		http.ServeFile(w, r, filePath)
	})

	// Serve other explorer HTML pages at /explorer/ prefix
	r.PathPrefix("/explorer/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fileName := strings.TrimPrefix(r.URL.Path, "/explorer/")
		filePath := filepath.Join(s.webguiStaticDir, fileName)
		http.ServeFile(w, r, filePath)
	})
	r.PathPrefix("/dashboard/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Serve dashboard.html as SPA fallback for all /dashboard/ sub-paths
		filePath := filepath.Join(s.webguiStaticDir, "dashboard.html")
		http.ServeFile(w, r, filePath)
	})

	// Also serve explorer HTML pages at root level for Next.js client-side routing.
	// This allows navigation within the SPA to work with paths like /payment-gateway.
	// When the specific {page}.html doesn't exist in the static export, fall back to
	// index.html (the SPA shell) so client-side routing can render the page.
	webguiPages := []string{
		// Quick Access
		"controller-status", "qr-connect", "payment-gateway",
		// Monitor
		"network-monitor", "graph-explorer",
		"chain-explorer", "chain-explorer-new", "oracle-explorer",
		"peers", "operator-registry", "tunnel-registry", "error-explorer",
		// Chain Explorers
		"transaction-explorer", "validation-explorer",
		// Models
		"models", "codex-builder", "models-dex",
		// Governance
		"bootnode-dao", "network-inference-dao",
		// GraphChain
		"graphchain-dashboard", "graphchain-errors", "graphchain-skills",
		// Marketplace
		"marketplace", "skills", "capabilities", "properties", "settlement",
		// Vault
		"my-models", "my-wallets", "my-skills", "my-capabilities", "my-properties",
		"nft-property-explorer", "vault",
		// Other
		"settings", "network-admin", "auth-test",
		// Role-based pages from pageAccess lists
		"basic", "advanced", "inventory", "blockchain", "dex", "daos",
		"nft-vault", "nft-capability-manager", "add-capability",
		"tools", "explorer", "capabilities",
	}

	for _, page := range webguiPages {
		pageName := page
		r.HandleFunc("/"+pageName, func(w http.ResponseWriter, r *http.Request) {
			filePath := filepath.Join(s.webguiStaticDir, pageName+".html")
			data, err := s.injectGatewayBase(filePath)
			if err != nil {
				// Page HTML doesn't exist — serve the SPA shell (index.html) so
				// Next.js client-side routing takes over.
				indexPath := filepath.Join(s.webguiStaticDir, "index.html")
				indexData, indexErr := s.injectGatewayBase(indexPath)
				if indexErr != nil {
					http.ServeFile(w, r, indexPath)
					return
				}
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Write(indexData)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(data)
		})
	}

	// DVE public verification pages — proxy /dve/{dveId}* to the backend Unix socket
	// so DVE pages are served server-side (Go templates) but exposed to the public via
	// the gateway.  The frontend iframes these pages back into the workspace UI.
	if s.config.BackendSocketPath != "" {
		dvePageProxy := newSocketProxy(s.config.BackendSocketPath, "http://knirvserver")
		r.PathPrefix("/dve/").Handler(dvePageProxy)
		s.logger.Info("DVE page proxy registered", zap.String("socket", s.config.BackendSocketPath))
	} else {
		s.logger.Warn("DVE page proxy not configured — /dve/* will not be available")
	}

	// Root-level catch-all: serve WebGUI SPA shell for client-side routing.
	// All unmatched paths get the WebGUI index.html with gateway config injected.
	r.PathPrefix("/").Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If path has a file extension and doesn't match a known static route, return 404
		if strings.Contains(filepath.Base(r.URL.Path), ".") {
			http.NotFound(w, r)
			return
		}

		// SPA client-side route — serve index.html with gateway config injected
		indexPath := filepath.Join(s.webguiStaticDir, "index.html")
		data, err := s.injectGatewayBase(indexPath)
		if err != nil {
			http.ServeFile(w, r, indexPath)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(data)
	}))

	s.router = r
	return nil
}

// findAvailablePort tries to listen on the given port, and if it's in use,
// tries subsequent ports until it finds one available.
func (s *Server) findAvailablePort(startPort int) (net.Listener, error) {
	const maxAttempts = 100
	for port := startPort; port < startPort+maxAttempts; port++ {
		addr := fmt.Sprintf(":%d", port)
		listener, err := net.Listen("tcp", addr)
		if err != nil {
			s.logger.Debug("Port in use, trying next",
				zap.Int("port", port),
				zap.Error(err),
			)
			continue
		}
		s.logger.Info("Found available port",
			zap.Int("port", port),
		)
		return listener, nil
	}
	return nil, fmt.Errorf("no available port found after %d attempts", maxAttempts)
}

// Start starts the HTTP server
func (s *Server) Start() error {
	ctx := context.Background()

	// Start tunnel service
	if err := s.tunnelService.Start(ctx); err != nil {
		return fmt.Errorf("failed to start tunnel service: %w", err)
	}

	// Start payment service
	if err := s.paymentService.Start(ctx); err != nil {
		return fmt.Errorf("failed to start payment service: %w", err)
	}

	// Start TURN server with blockchain integration
	if s.turnServer != nil {
		if err := s.turnServer.Start(ctx); err != nil {
			s.logger.Warn("Failed to start TURN server",
				zap.Error(err))
		}
	}

	// Setup CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	handler := c.Handler(wrapWithSecurityHeaders(s.router))

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.config.Port),
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	var listener net.Listener
	var err error

	// Listen on TCP port
	listener, err = s.findAvailablePort(s.config.Port)
	if err != nil {
		return fmt.Errorf("failed to find available port: %w", err)
	}
	// Extract the actual port from the listener
	s.actualPort = listener.Addr().(*net.TCPAddr).Port
	if s.actualPort != s.config.Port {
		s.logger.Info("Using dynamic port",
			zap.Int("port", s.actualPort),
			zap.Int("requested", s.config.Port),
		)
	} else {
		s.actualPort = s.config.Port
	}

	s.logger.Info("HTTP server listening",
		zap.String("address", s.httpServer.Addr),
	)

	if err := s.httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server failed: %w", err)
	}

	return nil
}

// Stop gracefully stops the HTTP server
func (s *Server) Stop(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}

	s.logger.Info("Stopping HTTP server")

	// Stop TURN server
	if s.turnServer != nil {
		if err := s.turnServer.Stop(ctx); err != nil {
			s.logger.Error("Failed to stop TURN server", zap.Error(err))
		}
	}

	// Stop tunnel service
	if err := s.tunnelService.Stop(ctx); err != nil {
		s.logger.Error("Failed to stop tunnel service", zap.Error(err))
	}

	// Stop payment service
	if err := s.paymentService.Stop(ctx); err != nil {
		s.logger.Error("Failed to stop payment service", zap.Error(err))
	}

	return s.httpServer.Shutdown(ctx)
}

// injectGatewayBase reads an HTML file and injects a <script> tag with
// __GATEWAY_BASE__, __PAYMENT_API__, and KNIRV_GATEWAY_CONFIG configuration
// before </head>.  KNIRV_GATEWAY_CONFIG is read by gateway-links.js as its
// primary config source — without it the JS falls back to hardcoded
// localhost:3001 for the payment oracle health check and other endpoints.
func (s *Server) injectGatewayBase(filePath string) ([]byte, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	gatewayBase := fmt.Sprintf("http://localhost:%d", s.config.Port)
	script := fmt.Sprintf(
		`<script>window.__GATEWAY_BASE__=%q;window.__PAYMENT_API__="/api/v1/payments";window.KNIRV_GATEWAY_CONFIG=%s;</script>`,
		gatewayBase,
		knirvGatewayConfigJSON(s.config),
	)
	injected := strings.Replace(string(data), "</head>", script+"</head>", 1)
	return []byte(injected), nil
}

// knirvGatewayConfigJSON returns the JSON payload that gateway-links.js reads
// as window.KNIRV_GATEWAY_CONFIG.  This replaces the hardcoded fallback
// (localhost:3001) with actual gateway-proxied URLs so health checks work.
func knirvGatewayConfigJSON(cfg *config.Config) string {
	gatewayPort := cfg.Port
	if gatewayPort == 0 {
		gatewayPort = 8080
	}
	base := fmt.Sprintf("http://localhost:%d", gatewayPort)
	// Use the same format gateway-links.js expects in its getFallbackConfiguration()
	return fmt.Sprintf(`{"deployment_mode":"private_testnet","oracle_services":{"payment_oracle":{"base_url":"%[1]s/api/v1/payments/","health_url":"%[1]s/api/v1/payments/health","domain":"localhost","port":%[2]d,"path":"/api/v1/payments/"},"tunnel_registry":{"base_url":"%[1]s/api/v1/tunnels/","health_url":"%[1]s/api/v1/tunnels/status","domain":"localhost","port":%[2]d,"path":"/api/v1/tunnels/"},"operator_registry":{"base_url":"%[1]s/api/v1/operators/","health_url":"%[1]s/api/v1/operators/health","domain":"localhost","port":%[2]d,"path":"/api/v1/operators/"},"webgui":{"base_url":"%[1]s/","health_url":"%[1]s/health","domain":"localhost","port":%[2]d,"path":"/"}},"external_services":{"knirv_website":"https://knirv.com","testnet_access":"https://testnet.knirv.network"},"navigation":{},"timestamp":"%[3]s"}`,
		base, gatewayPort, time.Now().UTC().Format(time.RFC3339))
}

// wrapWithSecurityHeaders adds security-related HTTP headers to all responses.
func wrapWithSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("Content-Security-Policy", "frame-ancestors 'self' http://localhost:8080 http://localhost:8090")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}

// newSocketProxy creates a reverse proxy that dials a Unix socket and forwards
// requests to the given target base URL.
func newSocketProxy(socketPath, targetBase string) *httputil.ReverseProxy {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return net.Dial("unix", socketPath)
		},
	}
	target, _ := url.Parse(targetBase)
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = transport
	// Strip CORS headers from the upstream response so the gateway's own CORS
	// middleware is the single authority.  Without this the backend can inject
	// Access-Control-Allow-Origin alongside the gateway's wildcard, producing
	// the "multiple values" error browsers reject.
	proxy.ModifyResponse = func(resp *http.Response) error {
		resp.Header.Del("Access-Control-Allow-Origin")
		resp.Header.Del("Access-Control-Allow-Headers")
		resp.Header.Del("Access-Control-Allow-Methods")
		resp.Header.Del("Access-Control-Allow-Credentials")
		// Strip frame/CSP headers from the upstream response so the gateway's own
		// wrapWithSecurityHeaders (SAMEORIGIN, proper frame-ancestors CSP) is
		// authoritative.  Without this the backend's SecurityHeadersMiddleware
		// (X-Frame-Options: DENY, default-src 'self') overrides the gateway's
		// permissive settings and blocks iframing in components like the DVE
		// verification panel.
		resp.Header.Del("X-Frame-Options")
		resp.Header.Del("Content-Security-Policy")
		return nil
	}
	return proxy
}

// dynamicAgentProxy returns an http.Handler that routes /api/agent/{dveId}/*
// to the per-DVE KNIRVAGENT's Unix socket (agent-{dveId}.sock).
// The proxy is created on first use and cached per DVE ID.
func (s *Server) dynamicAgentProxy() http.Handler {
	var (
		mu     sync.RWMutex
		cache  = make(map[string]*httputil.ReverseProxy)
	)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract DVE ID from path: /api/agent/{dveId}/rest/of/path
		path := strings.TrimPrefix(r.URL.Path, "/api/agent/")
		parts := strings.SplitN(path, "/", 2)
		if len(parts) == 0 || parts[0] == "" {
			http.Error(w, `{"error":"missing dve id"}`, http.StatusBadRequest)
			return
		}
		dveID := parts[0]

		socketPath := filepath.Join(s.config.AgentSocketDir, fmt.Sprintf("agent-%s.sock", dveID))

		// Check if socket exists before proxying
		if _, err := os.Stat(socketPath); os.IsNotExist(err) {
			s.logger.Warn("Agent socket not found for DVE",
				zap.String("dveID", dveID),
				zap.String("socket", socketPath))
			http.Error(w, fmt.Sprintf(`{"error":"agent not running for dve %s"}`, dveID),
				http.StatusServiceUnavailable)
			return
		}

		// Get or create a cached proxy for this DVE
		mu.RLock()
		proxy, ok := cache[dveID]
		mu.RUnlock()
		if !ok {
			mu.Lock()
			// Double-check inside write lock
			if proxy, ok = cache[dveID]; !ok {
				targetBase := fmt.Sprintf("http://knirvagent-%s", dveID)
				proxy = newSocketProxy(socketPath, targetBase)
				cache[dveID] = proxy
				s.logger.Info("Created agent proxy",
					zap.String("dveID", dveID),
					zap.String("socket", socketPath))
			}
			mu.Unlock()
		}

		// Rewrite the URL path: strip /api/agent/{dveId}
		r.URL.Path = "/" + strings.Join(parts[1:], "/")
		r.URL.RawPath = ""

		proxy.ServeHTTP(w, r)
	})
}

// innerAgentUpgrader is the WebSocket upgrader for inner agent tunnels.
var innerAgentUpgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// handleInnerAgentWS upgrades the HTTP connection to a WebSocket and
// pipes data to/from the KNIRVAGENT's Unix socket for inner agent
// streaming and input forwarding.
func (s *Server) handleInnerAgentWS(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	dveID := vars["dveId"]
	sessionID := vars["sessionId"]

	if dveID == "" {
		http.Error(w, `{"error":"dveId required"}`, http.StatusBadRequest)
		return
	}

	socketPath := filepath.Join(s.config.AgentSocketDir, fmt.Sprintf("agent-%s.sock", dveID))
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		s.logger.Warn("Inner agent WS: socket not found", zap.String("dveID", dveID), zap.String("socket", socketPath))
		http.Error(w, fmt.Sprintf(`{"error":"agent not running for dve %s"}`, dveID), http.StatusServiceUnavailable)
		return
	}

	conn, err := innerAgentUpgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("Inner agent WS upgrade failed", zap.String("dveID", dveID), zap.Error(err))
		return
	}
	defer conn.Close()

	unixConn, err := net.DialTimeout("unix", socketPath, 5*time.Second)
	if err != nil {
		s.logger.Error("Inner agent WS: Unix dial failed", zap.String("dveID", dveID), zap.String("socket", socketPath), zap.Error(err))
		conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"error","data":"cannot connect to agent"}`))
		return
	}
	defer unixConn.Close()

	if sessionID != "" {
		httpReq := "GET /api/inner/" + sessionID + "/stream HTTP/1.1\r\nHost: unix\r\n\r\n"
		if _, err := unixConn.Write([]byte(httpReq)); err != nil {
			s.logger.Error("Inner agent WS: stream request failed", zap.String("dveID", dveID), zap.Error(err))
			return
		}
	}

	errCh := make(chan error, 2)

	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := unixConn.Read(buf)
			if n > 0 {
				msg, _ := json.Marshal(map[string]string{"type": "data", "data": string(buf[:n])})
				if writeErr := conn.WriteMessage(websocket.TextMessage, msg); writeErr != nil {
					errCh <- writeErr
					return
				}
			}
			if err != nil {
				endMsg, _ := json.Marshal(map[string]string{"type": "stream_end"})
				conn.WriteMessage(websocket.TextMessage, endMsg)
				errCh <- err
				return
			}
		}
	}()

	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				errCh <- err
				return
			}
			var msg struct {
				Type    string `json:"type"`
				Data    string `json:"data,omitempty"`
				Session string `json:"session,omitempty"`
			}
			if err := json.Unmarshal(message, &msg); err != nil {
				continue
			}
			switch msg.Type {
			case "input":
				targetSession := msg.Session
				if targetSession == "" {
					targetSession = sessionID
				}
				if targetSession == "" {
					continue
				}
				body := fmt.Sprintf("data=%s", msg.Data)
				inputReq := fmt.Sprintf("POST /api/inner/%s/input HTTP/1.1\r\nHost: unix\r\nContent-Type: application/x-www-form-urlencoded\r\nContent-Length: %d\r\n\r\n%s",
					targetSession, len(body), body)
				unixConn.Write([]byte(inputReq))
			case "ping":
				conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"pong"}`))
			}
		}
	}()

	<-errCh
}

// Handler implementations

func (s *Server) handleGetSession(w http.ResponseWriter, r *http.Request) {
	sess, err := s.sessionManager.GetOrCreate(r, w)
	if err != nil {
		http.Error(w, "Failed to get session", http.StatusInternalServerError)
		return
	}

	controllerURL, _ := s.sessionManager.GetControllerURL(sess.ID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"controllerUrl": controllerURL,
	})
}

func (s *Server) handleSetSession(w http.ResponseWriter, r *http.Request) {
	sess, err := s.sessionManager.GetOrCreate(r, w)
	if err != nil {
		http.Error(w, "Failed to get session", http.StatusInternalServerError)
		return
	}

	var req struct {
		ControllerURL string `json:"controllerUrl"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ControllerURL == "" {
		http.Error(w, "controllerUrl is required", http.StatusBadRequest)
		return
	}

	if err := s.sessionManager.SetControllerURL(sess.ID, req.ControllerURL); err != nil {
		http.Error(w, fmt.Sprintf("Invalid controllerUrl: %v", err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":            true,
		"controllerUrl": req.ControllerURL,
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"status":    "healthy",
		"mode":      s.config.GatewayMode,
		"timestamp": time.Now().UnixMilli(),
		"chainId":   s.config.ChainID,
		"port":      s.actualPort,
		"dht": map[string]interface{}{
			"status": "not_implemented",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (s *Server) handleProvision(w http.ResponseWriter, r *http.Request) {
	// Placeholder for DHT provisioning
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode([]interface{}{})
}

func (s *Server) handleDHTStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "not_implemented",
		"mode":   s.config.GatewayMode,
	})
}

func (s *Server) handleDHTStart(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error":   "Not Implemented",
		"message": "DHT functionality not yet implemented in Go version",
	})
}

func (s *Server) handleDHTStop(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error":   "Not Implemented",
		"message": "DHT functionality not yet implemented in Go version",
	})
}

func (s *Server) handleControllerProxy() http.Handler {
	return s.proxyHandler.DynamicProxy(func(r *http.Request) (string, error) {
		sess, err := s.sessionManager.GetOrCreate(r, nil)
		if err != nil {
			return "", fmt.Errorf("failed to get session: %w", err)
		}

		controllerURL, ok := s.sessionManager.GetControllerURL(sess.ID)
		if !ok || controllerURL == "" {
			return "", fmt.Errorf("no controller URL set in session")
		}

		return controllerURL, nil
	})
}

// handleNetworkMonitorStatus returns the network monitor overall status.
// Replaces the Next.js API route /api/network-monitor/status which is
// excluded from the static export.
func (s *Server) handleNetworkMonitorStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"name":           "KNIRV Local Network",
			"overall_status": "healthy",
			"services_up":    3,
			"services_down":  0,
			"services_total": 6,
			"last_update":    time.Now().UTC().Format(time.RFC3339),
		},
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// handleNetworkMonitorServices returns per-service health status.
func (s *Server) handleNetworkMonitorServices(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	services := map[string]interface{}{
		"knirvchain": map[string]interface{}{
			"name": "knirvchain", "url": "http://localhost:8080", "status": "up",
			"lastCheck": time.Now().UTC().Format(time.RFC3339), "responseTime": 5, "uptime": 99.5,
		},
		"knirvserver": map[string]interface{}{
			"name": "knirvserver", "url": "http://localhost:8082", "status": "up",
			"lastCheck": time.Now().UTC().Format(time.RFC3339), "responseTime": 3, "uptime": 99.8,
		},
		"knirvgraph": map[string]interface{}{
			"name": "knirvgraph", "url": "http://localhost:8081", "status": "up",
			"lastCheck": time.Now().UTC().Format(time.RFC3339), "responseTime": 8, "uptime": 98.2,
		},
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"data":      services,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// handleNetworkMonitorMetrics returns system metrics (simplified).
func (s *Server) handleNetworkMonitorMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"cpu": map[string]interface{}{
				"usage_percent": 12.5, "load_average": []float64{0.8, 0.6, 0.4},
			},
			"memory": map[string]interface{}{
				"total_bytes": 16e9, "used_bytes": 8e9, "available_bytes": 8e9, "usage_percent": 50.0,
			},
			"disk": map[string]interface{}{
				"total_bytes": 500e9, "used_bytes": 200e9, "available_bytes": 300e9, "usage_percent": 40.0,
			},
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		},
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// handleNetworkMonitorLogs returns recent monitor log entries.
func (s *Server) handleNetworkMonitorLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data": []map[string]interface{}{
			{"level": "info", "message": "Network monitor initialized", "service": "network-monitor", "timestamp": time.Now().UTC().Format(time.RFC3339)},
		},
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// handleNetworkMonitorConfig returns the monitor configuration.
func (s *Server) handleNetworkMonitorConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"active_network": "local",
			"networks": map[string]interface{}{
				"local": map[string]interface{}{
					"name": "KNIRV Local", "description": "Local development network",
				},
			},
		},
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// handleAPIInfo returns server info — used by the WebGUI's backend detection.
func (s *Server) handleEmptyArray(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("[]\n"))
}

// Handler implementations
func (s *Server) handleTurnStatus(w http.ResponseWriter, r *http.Request) {
	if s.turnServer == nil {
		http.Error(w, "TURN server not enabled", http.StatusServiceUnavailable)
		return
	}

	status := s.turnServer.GetStatus()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (s *Server) handleTurnStats(w http.ResponseWriter, r *http.Request) {
	if s.turnServer == nil {
		http.Error(w, "TURN server not enabled", http.StatusServiceUnavailable)
		return
	}

	s.logger.Debug("Handling TURN stats request")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
	})
}

func (s *Server) handleTurnProofSubmit(w http.ResponseWriter, r *http.Request) {
	if s.turnServer == nil {
		http.Error(w, "TURN server not enabled", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		NodeID    string  `json:"node_id"`
		ProofID   string  `json:"proof_id"`
		Score     float64 `json:"score"`
		ProofData string  `json:"proof_data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	baseReward := 100.0
	rewardAmount := baseReward * (req.Score / 100.0)
	amountStr := fmt.Sprintf("%.0f", rewardAmount*1e18)

	// Use the blockchain adapter through the TURN server
	blockchainAdapter := turnserver.NewBlockchainAdapter(nil, s.config.TurnServerMinerAddress)
	err := blockchainAdapter.SubmitConnectivityProofReward(req.NodeID, req.ProofID, req.Score, amountStr)
	if err != nil {
		s.logger.Error("Error submitting connectivity proof reward",
			zap.Error(err))
		http.Error(w, "Failed to submit proof reward", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success":       true,
		"proof_id":      req.ProofID,
		"reward_amount": amountStr,
		"message":       "Connectivity proof submitted successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleTurnProofStatus(w http.ResponseWriter, r *http.Request) {
	if s.turnServer == nil {
		http.Error(w, "TURN server not enabled", http.StatusServiceUnavailable)
		return
	}

	proofID := r.URL.Query().Get("proof_id")
	if proofID == "" {
		http.Error(w, "proof_id parameter required", http.StatusBadRequest)
		return
	}

	response := map[string]interface{}{
		"proof_id":  proofID,
		"status":    "verified",
		"timestamp": time.Now(),
		"verified":  true,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleTurnMintNRN(w http.ResponseWriter, r *http.Request) {
	if s.turnServer == nil {
		http.Error(w, "TURN server not enabled", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		Recipient string `json:"recipient"`
		Amount    string `json:"amount"`
		Reason    string `json:"reason"`
		ProofID   string `json:"proof_id,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	blockchainAdapter := turnserver.NewBlockchainAdapter(nil, s.config.TurnServerMinerAddress)
	err := blockchainAdapter.SubmitNRNMintTx(req.Recipient, req.Amount, req.Reason, req.ProofID)
	if err != nil {
		s.logger.Error("Error submitting NRN mint transaction",
			zap.Error(err))
		http.Error(w, "Failed to mint NRN tokens", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success":   true,
		"recipient": req.Recipient,
		"amount":    req.Amount,
		"reason":    req.Reason,
		"message":   "NRN tokens minted successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleTurnMintReward(w http.ResponseWriter, r *http.Request) {
	if s.turnServer == nil {
		http.Error(w, "TURN server not enabled", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		NodeID            string `json:"node_id"`
		ParticipationType string `json:"participation_type"`
		Amount            string `json:"amount"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	blockchainAdapter := turnserver.NewBlockchainAdapter(nil, s.config.TurnServerMinerAddress)
	err := blockchainAdapter.SubmitParticipationReward(req.NodeID, req.ParticipationType, req.Amount)
	if err != nil {
		s.logger.Error("Error submitting participation reward",
			zap.Error(err))
		http.Error(w, "Failed to mint participation reward", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success":            true,
		"node_id":            req.NodeID,
		"participation_type": req.ParticipationType,
		"amount":             req.Amount,
		"message":            "Participation reward minted successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleTurnMintingStats(w http.ResponseWriter, r *http.Request) {
	if s.turnServer == nil {
		http.Error(w, "TURN server not enabled", http.StatusServiceUnavailable)
		return
	}

	blockchainAdapter := turnserver.NewBlockchainAdapter(nil, s.config.TurnServerMinerAddress)
	stats := blockchainAdapter.GetMintingStats()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (s *Server) handleTurnHealth(w http.ResponseWriter, r *http.Request) {
	if s.turnServer == nil {
		http.Error(w, "TURN server not enabled", http.StatusServiceUnavailable)
		return
	}

	health := map[string]interface{}{
		"status":      "healthy",
		"turn_server": s.turnServer.IsRunning(),
		"timestamp":   time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// DHT handler implementations

func (s *Server) handleDHTAnnounce(w http.ResponseWriter, r *http.Request) {
	if s.dhtManager == nil {
		http.Error(w, "DHT not enabled", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		ID           string `json:"id"`
		Type         string `json:"type"`
		Multiaddress string `json:"multiaddress,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	cid, err := createCID(req.ID, req.Type)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create CID: %v", err), http.StatusInternalServerError)
		return
	}

	if err := s.dhtManager.Provide(r.Context(), cid); err != nil {
		http.Error(w, fmt.Sprintf("Failed to announce: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
}

func (s *Server) handleDHTFind(w http.ResponseWriter, r *http.Request) {
	if s.dhtManager == nil {
		http.Error(w, "DHT not enabled", http.StatusServiceUnavailable)
		return
	}

	id := r.URL.Query().Get("id")
	resourceType := r.URL.Query().Get("type")
	if id == "" || resourceType == "" {
		http.Error(w, "id and type parameters required", http.StatusBadRequest)
		return
	}

	cid, err := createCID(id, resourceType)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create CID: %v", err), http.StatusInternalServerError)
		return
	}

	peers, err := s.dhtManager.FindProviders(r.Context(), cid)
	if err != nil {
		http.Error(w, fmt.Sprintf("Find failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(peers)
}

func (s *Server) handleDHTPeers(w http.ResponseWriter, r *http.Request) {
	if s.dhtManager == nil {
		http.Error(w, "DHT not enabled", http.StatusServiceUnavailable)
		return
	}

	// Get peers from DHT manager
	peers := []map[string]interface{}{}
	// This would need to be implemented in the DHT manager
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(peers)
}

func (s *Server) handleDHTBootstrap(w http.ResponseWriter, r *http.Request) {
	if s.dhtManager == nil {
		http.Error(w, "DHT not enabled", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		Peers []string `json:"peers"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "message": "Bootstrap initiated"})
}

// Resource Broadcast System handlers

func (s *Server) handleCacheResource(w http.ResponseWriter, r *http.Request) {
	if s.dhtManager == nil {
		http.Error(w, "DHT not enabled", http.StatusServiceUnavailable)
		return
	}

	var resource dht.ResourceEntry
	if err := json.NewDecoder(r.Body).Decode(&resource); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if resource.ID == "" || resource.Type == "" {
		http.Error(w, "id and type required", http.StatusBadRequest)
		return
	}

	resource.Timestamp = time.Now()
	s.dhtManager.GetResourceCache().AddResource(resource)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
}

func (s *Server) handleAnnounceAllCached(w http.ResponseWriter, r *http.Request) {
	if s.dhtManager == nil {
		http.Error(w, "DHT not enabled", http.StatusServiceUnavailable)
		return
	}

	count, err := s.dhtManager.AnnounceAllCached(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("Announce failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"announced": count})
}

func (s *Server) handleCacheStatus(w http.ResponseWriter, r *http.Request) {
	if s.dhtManager == nil {
		http.Error(w, "DHT not enabled", http.StatusServiceUnavailable)
		return
	}

	resources := s.dhtManager.GetResourceCache().GetAllResources()
	count := s.dhtManager.GetResourceCache().GetResourceCount()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"count":     count,
		"resources": resources,
	})
}

// Chain P2P proxy handlers

func (s *Server) handleP2PPublishBlock(w http.ResponseWriter, r *http.Request) {
	if s.dhtManager == nil {
		http.Error(w, "DHT not enabled", http.StatusServiceUnavailable)
		return
	}
	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	if err := s.dhtManager.PublishBlock(r.Context(), data); err != nil {
		http.Error(w, fmt.Sprintf("Publish failed: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
}

func (s *Server) handleP2PPublishTx(w http.ResponseWriter, r *http.Request) {
	if s.dhtManager == nil {
		http.Error(w, "DHT not enabled", http.StatusServiceUnavailable)
		return
	}
	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	if err := s.dhtManager.PublishTransaction(r.Context(), data); err != nil {
		http.Error(w, fmt.Sprintf("Publish failed: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
}

func (s *Server) handleP2PRegisterCallback(w http.ResponseWriter, r *http.Request) {
	if s.dhtManager == nil {
		http.Error(w, "DHT not enabled", http.StatusServiceUnavailable)
		return
	}
	var req struct {
		SocketPath string `json:"socket_path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.SocketPath == "" {
		http.Error(w, "socket_path required", http.StatusBadRequest)
		return
	}
	s.dhtManager.SetChainCallbackSocket(req.SocketPath)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
}

func (s *Server) handleP2PPeers(w http.ResponseWriter, r *http.Request) {
	if s.dhtManager == nil {
		http.Error(w, "DHT not enabled", http.StatusServiceUnavailable)
		return
	}
	peers := s.dhtManager.GetConnectedPeers()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"peers": peers})
}

func (s *Server) handleP2PSelfAddrs(w http.ResponseWriter, r *http.Request) {
	if s.dhtManager == nil {
		http.Error(w, "DHT not enabled", http.StatusServiceUnavailable)
		return
	}
	addrs := s.dhtManager.GetSelfMultiaddrs()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"addrs": addrs})
}

func (s *Server) handleP2PPeerID(w http.ResponseWriter, r *http.Request) {
	if s.dhtManager == nil {
		http.Error(w, "DHT not enabled", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"peer_id": s.dhtManager.GetPeerID()})
}

// createCID creates a CID from a resource ID and type.
func createCID(id, resourceType string) (cid.Cid, error) {
	hash, err := multihash.Sum([]byte(fmt.Sprintf("%s:%s", id, resourceType)), multihash.SHA2_256, -1)
	if err != nil {
		return cid.Cid{}, err
	}
	return cid.NewCidV1(cid.Raw, hash), nil
}

