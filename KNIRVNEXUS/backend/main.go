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
	"path/filepath"
	"syscall"
	"time"

	"nexus-backend/internal/config"
	dataengine "nexus-backend/internal/data-engine"
	"nexus-backend/internal/database"
	agentserver "nexus-backend/internal/services/agent-server"
	"nexus-backend/internal/services/cde"
	"nexus-backend/internal/services/dns"
	"nexus-backend/internal/services/dvemanager"
	"nexus-backend/internal/services/validation"
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
	dveManager     *dvemanager.DVEManager
	validationCore *validation.ValidationCore
	cdeService     *cde.CDEService
	dnsService     *dns.DynamicDNSService
	agentServer    *agentserver.AgentServer
	dataEngine     *dataengine.BuntDBDataEngine

	// Context for managing service lifecycle
	ctx    context.Context
	cancel context.CancelFunc

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

	validationCore, err := validation.NewValidationCore(dbManager.GetDB(), p2pManager, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize validation core: %w", err)
	}

	agentServer, err := agentserver.NewAgentServer(cfg, dbManager)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize agent server: %w", err)
	}

	// Initialize data engine
	dataEngineConfig := dataengine.BuntDBDataEngineConfig{
		DatabasePath:     cfg.Database.Path,
		EnableWebSocket:  true,
		EnableRESTAPI:    true,
		WebSocketPort:    8080,
		RESTAPIPort:      7080,
		WindowSize:       time.Minute * 5,
		MetricsInterval:  time.Second * 30,
		MetricsRetention: time.Hour * 24 * 7,  // 7 days
		AlertsRetention:  time.Hour * 24 * 30, // 30 days
		EventsRetention:  time.Hour * 24 * 7,  // 7 days
		BatchSize:        100,
		FlushInterval:    time.Second * 10,
		MaxMemoryUsage:   1024 * 1024 * 100, // 100MB
	}
	dataEngine, err := dataengine.NewBuntDBDataEngine(dataEngineConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize data engine: %w", err)
	}

	// Initialize CDE service (if configuration is available)
	var cdeService *cde.CDEService
	// TODO: Initialize when CDE configuration is available

	// Initialize DNS service (if configuration is available)
	var dnsService *dns.DynamicDNSService
	// TODO: Initialize when DNS configuration is available

	// Create context for service lifecycle management
	ctx, cancel := context.WithCancel(context.Background())

	server := &Server{
		config:         cfg,
		db:             dbManager,
		router:         router,
		p2pManager:     p2pManager,
		dveManager:     dveManager,
		validationCore: validationCore,
		cdeService:     cdeService,
		dnsService:     dnsService,
		agentServer:    agentServer,
		dataEngine:     dataEngine,
		ctx:            ctx,
		cancel:         cancel,
		running:        false,
	}

	// Setup routes for all services
	server.setupRoutes()

	return server, nil
}

// setupRoutes configures all HTTP routes for the unified server
func (s *Server) setupRoutes() {
	// Add CORS middleware
	s.router.Use(middleware.CORSMiddleware)

	// Health check endpoint
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")
	s.router.HandleFunc("/api/health", s.handleHealth).Methods("GET")

	// Create auth middleware
	authMiddleware, err := middleware.NewAuthMiddleware(s.db, s.config.Security.JWTSecret)
	if err != nil {
		log.Printf("Warning: Failed to create auth middleware: %v", err)
		authMiddleware = nil
	}

	// Register data engine routes
	if s.dataEngine != nil {
		s.dataEngine.RegisterRoutes(s.router)
	}

	// Register DVE manager routes
	if s.dveManager != nil {
		s.dveManager.RegisterRoutes(s.router, authMiddleware)
	}

	// Register validation core routes
	if s.validationCore != nil {
		s.validationCore.RegisterRoutes(s.router, authMiddleware)
	}

	// Register agent server routes
	if s.agentServer != nil {
		s.agentServer.RegisterRoutes(s.router)
	}

	// Register CDE service routes (when available)
	if s.cdeService != nil {
		s.cdeService.RegisterRoutes(s.router, authMiddleware)
	}

	// Register DNS service routes (when available)
	if s.dnsService != nil {
		s.dnsService.RegisterRoutes(s.router, authMiddleware)
	}

	log.Println("All routes configured successfully")
}

// Start starts all services and the HTTP server
func (s *Server) Start() error {
	if s.running {
		return fmt.Errorf("server is already running")
	}

	log.Println("Starting KNIRV-NEXUS unified server...")

	// Start P2P manager
	s.p2pManager.Start() // P2P manager start doesn't return error
	log.Println("P2P Manager started")

	// Start DVE manager
	if s.dveManager != nil {
		if err := s.dveManager.Start(s.ctx); err != nil {
			return fmt.Errorf("failed to start DVE manager: %w", err)
		}
		log.Println("DVE Manager started")
	}

	// Start validation core
	if s.validationCore != nil {
		if err := s.validationCore.Start(s.ctx); err != nil {
			return fmt.Errorf("failed to start validation core: %w", err)
		}
		log.Println("Validation Core started")
	}

	// Start agent server
	if s.agentServer != nil {
		if err := s.agentServer.Start(); err != nil {
			return fmt.Errorf("failed to start agent server: %w", err)
		}
		log.Println("Agent Server started")
	}

	// Start CDE service (when available)
	if s.cdeService != nil {
		// CDE service start will be implemented when available
	}

	// Start DNS service (when available)
	if s.dnsService != nil {
		// DNS service start will be implemented when available
	}

	// Start data engine
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

// Stop stops all services and the HTTP server
func (s *Server) Stop() error {
	if !s.running {
		return nil
	}

	log.Println("Stopping KNIRV-NEXUS unified server...")

	// Cancel context to stop all background goroutines
	if s.cancel != nil {
		s.cancel()
	}

	// Stop HTTP server
	if s.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := s.httpServer.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down HTTP server: %v", err)
		}
	}

	// Stop services in reverse order
	if s.dataEngine != nil {
		if err := s.dataEngine.Stop(); err != nil {
			log.Printf("Error stopping data engine: %v", err)
		}
	}

	if s.agentServer != nil {
		if err := s.agentServer.Stop(); err != nil {
			log.Printf("Error stopping agent server: %v", err)
		}
	}

	// Create context for stopping services
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer stopCancel()

	if s.validationCore != nil {
		if err := s.validationCore.Stop(stopCtx); err != nil {
			log.Printf("Error stopping validation core: %v", err)
		}
	}

	if s.dveManager != nil {
		if err := s.dveManager.Stop(stopCtx); err != nil {
			log.Printf("Error stopping DVE manager: %v", err)
		}
	}

	if s.p2pManager != nil {
		s.p2pManager.Stop() // P2P manager stop doesn't return error
	}

	// Close database
	if s.db != nil {
		if err := s.db.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		}
	}

	s.running = false
	log.Println("KNIRV-NEXUS unified server stopped")
	return nil
}

// handleHealth handles the /health endpoint
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := map[string]any{
		"status":     "healthy",
		"version":    Version,
		"build_time": BuildTime,
		"git_commit": GitCommit,
		"services": map[string]bool{
			"database":        s.db != nil,
			"p2p_manager":     s.p2pManager != nil,
			"dve_manager":     s.dveManager != nil,
			"validation_core": s.validationCore != nil,
			"agent_server":    s.agentServer != nil,
			"data_engine":     s.dataEngine != nil,
			"cde_service":     s.cdeService != nil,
			"dns_service":     s.dnsService != nil,
		},
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func main() {
	// Parse command line flags
	var configFile = flag.String("config", "", "Path to configuration file")
	flag.Parse()

	// Print version information
	fmt.Printf("KNIRV-NEXUS Complete Backend Server v%s (built %s, commit %s)\n", Version, BuildTime, GitCommit)

	// Set config file if provided, otherwise use relative path from backend directory
	if *configFile != "" {
		viper.SetConfigFile(*configFile)
	} else {
		// Set default config path relative to backend directory
		configPath := filepath.Join("config", "knirv-nexus.yaml")
		if _, err := os.Stat(configPath); err == nil {
			viper.SetConfigFile(configPath)
		}
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
