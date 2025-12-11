package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/config"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/operator"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/payment"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/proxy"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/session"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/tunnel"
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
	operatorService   *operator.Service
	operatorHandler   *operator.Handler
	tunnelService     *tunnel.Service
	tunnelHandler     *tunnel.Handler
	paymentService    *payment.Service
	paymentHandler    *payment.Handler
	webguiHandler     *webgui.Handler
	logger            *zap.Logger
	httpServer        *http.Server
	router            *mux.Router
	webguiStaticDir   string
	networkWebsiteDir string
}

// New creates a new HTTP server
func New(cfg *config.Config, webguiStaticDir, networkWebsiteDir string, logger *zap.Logger) (*Server, error) {
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

	// Initialize webgui handler
	webguiHdlr := webgui.NewHandler(cfg, logger)

	s := &Server{
		config:            cfg,
		sessionManager:    session.NewManager(cfg.SessionSecret),
		proxyHandler:      proxy.NewHandler(logger),
		operatorService:   operatorSvc,
		operatorHandler:   operatorHdlr,
		tunnelService:     tunnelSvc,
		tunnelHandler:     tunnelHdlr,
		paymentService:    paymentSvc,
		paymentHandler:    paymentHdlr,
		webguiHandler:     webguiHdlr,
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

	// Register operator registry routes directly
	s.operatorHandler.RegisterRoutes(r)

	// Register tunnel registry routes directly
	s.tunnelHandler.RegisterRoutes(r)

	// Register payment gateway routes directly
	s.paymentHandler.RegisterRoutes(r)

	// Register webgui API routes directly
	s.webguiHandler.RegisterRoutes(r)

	// Dynamic controller proxy
	r.PathPrefix("/controller").Handler(s.handleControllerProxy())

	// Mock API endpoint (fallback for any unmatched /api routes)
	r.PathPrefix("/api").HandlerFunc(s.handleMockAPI)

	// Serve webgui static files at /dashboard
	r.PathPrefix("/dashboard").Handler(http.StripPrefix("/dashboard", http.FileServer(http.Dir(s.webguiStaticDir))))

	// Serve network-website at root (this should be last to catch all remaining routes)
	r.PathPrefix("/").Handler(http.FileServer(http.Dir(s.networkWebsiteDir)))

	s.router = r
	return nil
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
			"message":   "Mock KNIRV central API gateway",
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
