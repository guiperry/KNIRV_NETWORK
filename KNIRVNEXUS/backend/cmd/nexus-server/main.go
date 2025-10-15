package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"nexus-backend/internal/config"
	dataengine "nexus-backend/internal/data-engine"
	"nexus-backend/internal/database"
	"nexus-backend/internal/inference"
	"nexus-backend/internal/services/cde"
	"nexus-backend/internal/services/dns"
	"nexus-backend/internal/services/dvemanager"
	modelserver "nexus-backend/internal/services/model-server"
	"nexus-backend/internal/services/validation"
	"nexus-backend/internal/web"
	"nexus-backend/internal/web/middleware"
	"nexus-backend/pkg/p2p"

	"github.com/gorilla/mux"
	"github.com/spf13/viper"
)

// Version information (set by build flags)
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

// Server represents the unified KNIRV-NEXUS server
type Server struct {
	config     *config.Config
	db         *database.BuntDBManager
	router     *mux.Router
	httpServer *http.Server
	p2pManager *p2p.DVEP2PManager

	// All services are held here
	dveManager         *dvemanager.DVEManager
	validationCore     *validation.ValidationCore
	cdeService         *cde.CDEService
	dnsService         *dns.DynamicDNSService
	modelServer        *modelserver.ModelServer
	dataEngine         *dataengine.BuntDBDataEngine
	inferenceService   *inference.InferenceService

	// State management
	running bool
}

// NewServer creates a new unified KNIRV-NEXUS server instance
func NewServer(cfg *config.Config) (*Server, error) {
	// Initialize database
	dbManager, err := database.NewBuntDB(cfg.Database.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Initialize P2P manager
	p2pManager, err := p2p.NewDVEP2PManager(cfg.ChainID, cfg.NodeRole, dbManager.GetDB())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize P2P manager: %w", err)
	}

	// Create router
	router := mux.NewRouter()

	// Initialize services
	dveManager, err := dvemanager.NewDVEManager(dbManager.GetDB(), p2pManager, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize DVE manager: %w", err)
	}

	// Initialize inference service
	inferenceService, err := inference.NewInferenceService(dbManager)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize inference service: %w", err)
	}

	validationCore, err := validation.NewValidationCore(dbManager.GetDB(), p2pManager, cfg, inferenceService)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize validation core: %w", err)
	}

	// For now, initialize CDE and DNS services with minimal configuration
	// TODO: Add proper configuration support
	modelServer, err := modelserver.NewModelServer(cfg, dbManager)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize model server: %w", err)
	}

	// Initialize BuntDB data engine
	dataEngineConfig := dataengine.BuntDBDataEngineConfig{
		DatabasePath:     cfg.Database.Path,
		EnableWebSocket:  false, // Handled by unified server
		EnableRESTAPI:    false, // Handled by unified server
		WindowSize:       5 * time.Minute,
		MetricsInterval:  30 * time.Second,
		MetricsRetention: 24 * time.Hour,
		AlertsRetention:  7 * 24 * time.Hour,
		EventsRetention:  24 * time.Hour,
		BatchSize:        100,
		FlushInterval:    10 * time.Second,
		MaxMemoryUsage:   100 * 1024 * 1024, // 100MB
	}
	dataEngine, err := dataengine.NewBuntDBDataEngine(dataEngineConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize data engine: %w", err)
	}

	server := &Server{
		config:           cfg,
		db:               dbManager,
		router:           router,
		p2pManager:       p2pManager,
		dveManager:       dveManager,
		validationCore:   validationCore,
		cdeService:       nil, // TODO: Initialize when configuration is available
		dnsService:       nil, // TODO: Initialize when configuration is available
		modelServer:      modelServer,
		dataEngine:       dataEngine,
		inferenceService: inferenceService,
		running:          false,
	}

	// Setup routes for all services
	server.setupRoutes()

	return server, nil
}

// setupRoutes configures all routes for the unified server
func (s *Server) setupRoutes() {
	// Setup CORS middleware
	s.router.Use(s.corsMiddleware)

	// Health check endpoint
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")
	s.router.HandleFunc("/version", s.handleVersion).Methods("GET")

	// API routes
	api := s.router.PathPrefix("/api/v1").Subrouter()

	// Create auth middleware
	authMiddleware, err := middleware.NewAuthMiddleware(s.db, s.config.Security.JWTSecret)
	if err != nil {
		log.Printf("Warning: Failed to create auth middleware: %v", err)
		// Continue without auth middleware for now
	}

	// Auth routes (before other protected routes)
	if authMiddleware != nil {
		authHandlers := web.NewAuthHandlers(s.db, authMiddleware)
		api.HandleFunc("/auth/login", authHandlers.Login).Methods("POST")
		api.HandleFunc("/auth/refresh", authHandlers.Refresh).Methods("POST")
		api.HandleFunc("/auth/revoke", authHandlers.Revoke).Methods("POST")
		api.HandleFunc("/auth/me", authHandlers.Me).Methods("GET")
		log.Println("Auth routes configured")
	}

	// Register service routes
	if s.dveManager != nil && authMiddleware != nil {
		s.dveManager.RegisterRoutes(api, authMiddleware)
	}

	if s.validationCore != nil && authMiddleware != nil {
		s.validationCore.RegisterRoutes(api, authMiddleware)
	}

	if s.modelServer != nil {
		s.modelServer.RegisterRoutes(api)
	}

	if s.cdeService != nil && authMiddleware != nil {
		s.cdeService.RegisterRoutes(api, authMiddleware)
	}

	if s.dnsService != nil && authMiddleware != nil {
		s.dnsService.RegisterRoutes(api, authMiddleware)
	}

	if s.dataEngine != nil {
		s.dataEngine.RegisterRoutes(api)
	}
}

// corsMiddleware provides CORS support
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// handleHealth handles the /health endpoint
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := map[string]any{
		"status":     "healthy",
		"version":    Version,
		"build_time": BuildTime,
		"git_commit": GitCommit,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
		"services": map[string]bool{
			"dve_manager":       s.dveManager != nil,
			"validation_core":   s.validationCore != nil,
			"model_server":      s.modelServer != nil && s.modelServer.IsRunning(),
			"inference_service": s.inferenceService != nil,
			"cde_service":       s.cdeService != nil,
			"dns_service":       s.dnsService != nil,
		},
	}
	json.NewEncoder(w).Encode(response)
}

// handleVersion handles the /version endpoint
func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := map[string]string{
		"version":    Version,
		"build_time": BuildTime,
		"git_commit": GitCommit,
	}
	json.NewEncoder(w).Encode(response)
}

// Start starts the unified server
func (s *Server) Start() error {
	log.Println("Starting KNIRV-NEXUS unified server...")

	// Start all services
	ctx := context.Background()

	if s.dveManager != nil {
		if err := s.dveManager.Start(ctx); err != nil {
			return fmt.Errorf("failed to start DVE manager: %w", err)
		}
		log.Println("DVE Manager started")
	}

	if s.validationCore != nil {
		if err := s.validationCore.Start(ctx); err != nil {
			return fmt.Errorf("failed to start validation core: %w", err)
		}
		log.Println("Validation Core started")
	}

	if s.modelServer != nil {
		if err := s.modelServer.Start(); err != nil {
			return fmt.Errorf("failed to start model server: %w", err)
		}
		log.Println("Model Server started")
	}

	if s.inferenceService != nil {
		if err := s.inferenceService.Start(); err != nil {
			return fmt.Errorf("failed to start inference service: %w", err)
		}
		log.Println("Inference Service started")
	}

	if s.cdeService != nil {
		// CDE service start will be implemented when available
	}

	if s.dnsService != nil {
		// DNS service start will be implemented when available
	}

	if s.dataEngine != nil {
		if err := s.dataEngine.Start(); err != nil {
			return fmt.Errorf("failed to start data engine: %w", err)
		}
		log.Println("Data Engine started")
	}

	// Create HTTP server
	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", s.config.API.BindAddress, s.config.API.Port),
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start HTTP server in goroutine
	go func() {
		log.Printf("Starting HTTP server on %s:%d", s.config.API.BindAddress, s.config.API.Port)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	s.running = true
	log.Println("KNIRV-NEXUS unified server started successfully")
	return nil
}

// Stop stops the unified server
func (s *Server) Stop() error {
	log.Println("Stopping KNIRV-NEXUS unified server...")

	s.running = false

	// Create context for shutdown operations
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop HTTP server
	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			log.Printf("HTTP server shutdown error: %v", err)
		}
	}

	// Stop all services in reverse order
	if s.modelServer != nil {
		if err := s.modelServer.Stop(); err != nil {
			log.Printf("Error stopping model server: %v", err)
		}
	}

	if s.inferenceService != nil {
		if err := s.inferenceService.Stop(); err != nil {
			log.Printf("Error stopping inference service: %v", err)
		}
	}

	if s.dnsService != nil {
		// DNS service stop will be implemented when available
	}

	if s.cdeService != nil {
		// CDE service stop will be implemented when available
	}

	if s.validationCore != nil {
		if err := s.validationCore.Stop(ctx); err != nil {
			log.Printf("Error stopping validation core: %v", err)
		}
	}

	if s.dveManager != nil {
		if err := s.dveManager.Stop(ctx); err != nil {
			log.Printf("Error stopping DVE manager: %v", err)
		}
	}

	if s.dataEngine != nil {
		if err := s.dataEngine.Stop(); err != nil {
			log.Printf("Error stopping data engine: %v", err)
		}
	}

	// Close database
	if s.db != nil {
		if err := s.db.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		}
	}

	log.Println("KNIRV-NEXUS unified server stopped")
	return nil
}

func main() {
	// Parse command line flags
	var configFile = flag.String("config", "", "Path to configuration file")
	flag.Parse()

	// Print version information
	fmt.Printf("KNIRV-NEXUS Unified Server v%s (built %s, commit %s)\n", Version, BuildTime, GitCommit)

	// Set config file if provided
	if *configFile != "" {
		viper.SetConfigFile(*configFile)
	}

	// Load configuration
	config, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create unified server
	server, err := NewServer(config)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Start the server
	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	if err := server.Stop(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}
	log.Println("Server stopped")
}
