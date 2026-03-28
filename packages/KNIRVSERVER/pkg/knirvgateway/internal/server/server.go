package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/auth"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/config"
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
	logger            *zap.Logger
	httpServer        *http.Server
	router            *mux.Router
	webguiStaticDir   string
	networkWebsiteDir string
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

	// Initialize webgui handler
	webguiHdlr := webgui.NewHandler(cfg, logger)

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
		webguiHandler:     webguiHdlr,
		turnServer:        turnSvc,
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

	// DHT/P2P endpoints (placeholder)
	r.HandleFunc("/provision", s.handleProvision).Methods("GET")
	r.HandleFunc("/dht/status", s.handleDHTStatus).Methods("GET")
	r.HandleFunc("/dht/start", s.handleDHTStart).Methods("POST")
	r.HandleFunc("/dht/stop", s.handleDHTStop).Methods("POST")

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

	// Register webgui API routes directly
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

	// Mock API endpoint (fallback for any unmatched /api routes)
	r.PathPrefix("/api").HandlerFunc(s.handleMockAPI)

	// IMPORTANT: Next.js static export uses absolute paths like /_next/..., /favicon.ico, etc.
	// We need to serve these at the root level so the webgui can load its assets

	// Serve Next.js _next directory (contains JS, CSS, chunks, etc.)
	r.PathPrefix("/_next/").Handler(http.StripPrefix("/", http.FileServer(http.Dir(s.webguiStaticDir))))

	// Serve webgui static files at root level (favicon, svgs, etc.)
	webguiStaticFiles := []string{"/favicon.ico", "/next.svg", "/window.svg", "/globe.svg", "/vercel.svg", "/file.svg"}
	for _, staticFile := range webguiStaticFiles {
		filePath := staticFile
		r.HandleFunc(filePath, func(w http.ResponseWriter, r *http.Request) {
			fullPath := filepath.Join(s.webguiStaticDir, filePath)
			s.logger.Debug("Serving webgui static file", zap.String("path", filePath), zap.String("fullPath", fullPath))
			http.ServeFile(w, r, fullPath)
		})
	}

	// Serve webgui index.html at /oracle and /dashboard routes
	r.HandleFunc("/oracle", func(w http.ResponseWriter, r *http.Request) {
		indexPath := filepath.Join(s.webguiStaticDir, "index.html")
		s.logger.Info("Serving webgui index at /oracle", zap.String("indexPath", indexPath))
		http.ServeFile(w, r, indexPath)
	})
	r.HandleFunc("/dashboard", func(w http.ResponseWriter, r *http.Request) {
		indexPath := filepath.Join(s.webguiStaticDir, "index.html")
		s.logger.Info("Serving webgui index at /dashboard", zap.String("indexPath", indexPath))
		http.ServeFile(w, r, indexPath)
	})

	// Serve other webgui HTML pages at /oracle/ prefix
	r.PathPrefix("/oracle/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fileName := strings.TrimPrefix(r.URL.Path, "/oracle/")
		filePath := filepath.Join(s.webguiStaticDir, fileName)
		http.ServeFile(w, r, filePath)
	})

	// Serve other webgui HTML pages at /dashboard/ prefix
	r.PathPrefix("/dashboard/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fileName := strings.TrimPrefix(r.URL.Path, "/dashboard/")
		filePath := filepath.Join(s.webguiStaticDir, fileName)
		http.ServeFile(w, r, filePath)
	})

	// Also serve webgui HTML pages at root level for Next.js client-side routing
	// This allows navigation within the SPA to work with paths like /payment-gateway
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
			http.ServeFile(w, r, filePath)
		})
	}

	// Serve network-website at root (this should be last to catch all remaining routes)
	r.PathPrefix("/").Handler(http.FileServer(http.Dir(s.networkWebsiteDir)))

	s.router = r
	return nil
}

// Start starts the HTTP server
func (s *Server) Start() error {
	ctx := context.Background()

	// Start tunnel service — non-fatal: port conflicts (e.g. VS Code on :3003)
	// must not prevent the HTTP server and WebGUI from starting.
	if err := s.tunnelService.Start(ctx); err != nil {
		s.logger.Warn("Tunnel service unavailable — P2P tunneling disabled",
			zap.Error(err))
	}

	// Start payment service — non-fatal: payment failure should not block WebGUI.
	if err := s.paymentService.Start(ctx); err != nil {
		s.logger.Warn("Payment service unavailable", zap.Error(err))
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

	handler := c.Handler(s.router)

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.config.Port),
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	s.logger.Info("HTTP server listening",
		zap.String("address", s.httpServer.Addr),
	)

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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
