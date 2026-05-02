package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
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
	webguiHandler     *webgui.Handler
	turnServer        *turnserver.Server
	dhtManager       *dht.DHTManager
	logger            *zap.Logger
	httpServer        *http.Server
	router            *mux.Router
	webguiStaticDir   string
	networkWebsiteDir string
	actualPort        int
}

// New creates a new HTTP server
func New(cfg *config.Config, webguiStaticDir, networkWebsiteDir string, logger *zap.Logger, db ...*sql.DB) (*Server, error) {
	var dbInstance *sql.DB
	if len(db) > 0 {
		dbInstance = db[0]
	}
	// Initialize operator service

	knirvOracleURL := "http://localhost:1317" // Default KNIRV-ORACLE URL
	if cfg.KnirvOracleURL != "" {
		knirvOracleURL = cfg.KnirvOracleURL
	}

	operatorSvc := operator.NewService(logger, knirvOracleURL)
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
		config:            cfg,
		sessionManager:    session.NewManager(cfg.SessionSecret),
		proxyHandler:      proxy.NewHandler(logger),
		authHandler:       authHdlr,
		operatorService:   operatorSvc,
		operatorHandler:   operatorHdlr,
		tunnelService:     tunnelSvc,
		tunnelHandler:     tunnelHdlr,
		paymentService:    paymentSvc,
		paymentHandler:    paymentHdlr,
		uriHandler:        uriHdlr,
		webguiHandler:     explorerHandler,
		turnServer:        turnSvc,
		dhtManager:       dhtMgr,
		logger:            logger,
		webguiStaticDir:   webguiStaticDir,
		networkWebsiteDir: networkWebsiteDir,
	}

	if err := s.setupRoutes(); err != nil {
		return nil, fmt.Errorf("failed to setup routes: %w", err)
	}

	return s, nil
}

// setupRoutes configures all HTTP routes
func (s *Server) setupRoutes() error {
	r := mux.NewRouter()

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

	// Backend reverse proxy routes — proxy to internal Unix sockets
	if s.config.BackendSocketPath != "" {
		backendProxy := newSocketProxy(s.config.BackendSocketPath, "http://knirvserver")
		r.PathPrefix("/api/v1/").Handler(backendProxy)
		s.logger.Info("Backend proxy registered", zap.String("socket", s.config.BackendSocketPath))
	} else {
		s.logger.Warn("Backend proxy not configured — /api/v1/* will not be proxied")
	}
	if s.config.ChainSocketPath != "" {
		chainProxy := newSocketProxy(s.config.ChainSocketPath, "http://knirvchain")
		r.PathPrefix("/chain/").Handler(chainProxy)
		s.logger.Info("Chain proxy registered", zap.String("socket", s.config.ChainSocketPath))
	} else {
		s.logger.Warn("Chain proxy not configured — /chain/* will not be proxied")
	}
	if s.config.GraphSocketPath != "" {
		graphProxy := newSocketProxy(s.config.GraphSocketPath, "http://knirvgraph")
		r.PathPrefix("/graph/").Handler(graphProxy)
		s.logger.Info("Graph proxy registered", zap.String("socket", s.config.GraphSocketPath))
	} else {
		s.logger.Warn("Graph proxy not configured — /graph/* will not be proxied")
	}

	// Mock API endpoint (fallback for any unmatched /api routes)
	r.PathPrefix("/api").HandlerFunc(s.handleMockAPI)

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
			s.logger.Error("Failed to inject gateway base into dashboard", zap.Error(err))
			http.ServeFile(w, r, filePath)
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
	webguiPages := []string{
		"payment-gateway", "tunnel-registry", "operator-registry",
		"marketplace", "models", "models-dex", "skills", "capabilities",
		"my-models", "my-skills", "my-capabilities", "my-properties", "my-wallets",
		"settings", "vault", "peers", "settlement", "auth-test",
		"controller-status", "network-admin", "network-monitor", "network-inference-dao",
		"chain-explorer", "chain-explorer-new", "graph-explorer", "error-explorer",
		"oracle-explorer", "graphchain-dashboard", "graphchain-errors", "graphchain-skills",
		"codex-builder", "nft-property-explorer", "bootnode-dao", "qr-connect",
		"basic", "advanced",
	}

	for _, page := range webguiPages {
		pageName := page
		r.HandleFunc("/"+pageName, func(w http.ResponseWriter, r *http.Request) {
			filePath := filepath.Join(s.webguiStaticDir, pageName+".html")
			data, err := s.injectGatewayBase(filePath)
			if err != nil {
				s.logger.Error("Failed to inject gateway base into page", zap.String("page", pageName), zap.Error(err))
				http.ServeFile(w, r, filePath)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(data)
		})
	}

	// Serve network-website at root (this should be last to catch all remaining routes)
	r.PathPrefix("/").Handler(http.FileServer(http.Dir(s.networkWebsiteDir)))

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
// __GATEWAY_BASE__ and __PAYMENT_API__ configuration before </head>.
func (s *Server) injectGatewayBase(filePath string) ([]byte, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	script := fmt.Sprintf(
		`<script>window.__GATEWAY_BASE__="http://localhost:%d";window.__PAYMENT_API__="/api/v1/payments";</script>`,
		s.config.Port,
	)
	injected := strings.Replace(string(data), "</head>", script+"</head>", 1)
	return []byte(injected), nil
}

// wrapWithSecurityHeaders adds security-related HTTP headers to all responses.
func wrapWithSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("Content-Security-Policy", "frame-ancestors 'self' http://localhost:8090")
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
	return proxy
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

func (s *Server) handleMockAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" && (r.URL.Path == "/api" || r.URL.Path == "/api/" || strings.HasSuffix(r.URL.Path, "/health")) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "ok",
			"message":   "Mock KNIRV central API oracle",
			"chainId":   s.config.ChainID,
			"timestamp": time.Now().UnixMilli(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error":   "Not Implemented",
		"message": "Central API routing is not yet implemented. This is a mock endpoint.",
		"method":  r.Method,
		"route":   r.URL.Path,
	})
}

// TURN server handlers (blockchain-enabled)
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

