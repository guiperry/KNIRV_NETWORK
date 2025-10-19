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

	"backend-server/internal/config"
	dataengine "backend-server/internal/data-engine"
	"backend-server/internal/database"
	"backend-server/internal/inference"
	"backend-server/internal/services/cde"
	"backend-server/internal/services/controllerintegration"
	"backend-server/internal/services/dns"
	"backend-server/internal/services/dvemanager"
	"backend-server/internal/services/dverental"
	modelserver "backend-server/internal/services/model-server"
	"backend-server/internal/services/modelmanagement"
	"backend-server/internal/services/systemhealth"
	"backend-server/internal/services/teesecurity"
	"backend-server/internal/services/validation"
	"backend-server/internal/services/websocket"
	"backend-server/internal/web"
	"backend-server/internal/web/middleware"
	"backend-server/pkg/p2p"

	"github.com/gorilla/mux"
	"github.com/spf13/viper"
)

// Version information (set by build flags)
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

// Server represents the KNIRV-NEXUS backend server
type Server struct {
	config     *config.Config
	db         *database.BuntDBManager
	router     *mux.Router
	httpServer *http.Server
	p2pManager *p2p.DVEP2PManager

	// All services are held here
	dveManager                   *dvemanager.DVEManager
	validationCore               *validation.ValidationCore
	cdeService                   *cde.CDEService
	dnsService                   *dns.DynamicDNSService
	modelServer                  *modelserver.ModelServer
	dataEngine                   *dataengine.BuntDBDataEngine
	inferenceService             *inference.InferenceService
	websocketService             *websocket.WebSocketService
	teeSecurityService           *teesecurity.TEESecurityService
	systemHealthService          *systemhealth.SystemHealthService
	modelManagementService       *modelmanagement.ModelManagementService
	controllerIntegrationService *controllerintegration.ControllerIntegrationService
	dveRentalService             *dverental.DVERentalService

	// Context for managing service lifecycle
	ctx    context.Context
	cancel context.CancelFunc

	// State management
	running bool
}

// NewServer creates a new KNIRV-NEXUS backend server instance
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

	modelServer, err := modelserver.NewModelServer(cfg, dbManager)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize model server: %w", err)
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

	// Initialize inference service
	inferenceService, err := inference.NewInferenceService(dbManager)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize inference service: %w", err)
	}

	validationCore, err := validation.NewValidationCore(dbManager.GetDB(), p2pManager, cfg, inferenceService)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize validation core: %w", err)
	}

	// Initialize TEE Security service with Kali environment detection
	teeSecurityService, err := teesecurity.NewTEESecurityService(dbManager.GetDB())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize TEE security service: %w", err)
	}

	// Initialize System Health service
	systemHealthService := systemhealth.NewSystemHealthService(dbManager.GetDB())
	systemHealthService.SetServiceReferences(dveManager, validationCore, inferenceService, teeSecurityService)

	// Initialize Model Management service
	modelManagementService := modelmanagement.NewModelManagementService(dbManager.GetDB())
	modelManagementService.SetModelServerReference(modelServer)

	// Initialize Controller Integration service
	controllerIntegrationService := controllerintegration.NewControllerIntegrationService(dbManager.GetDB())

	// Initialize WebSocket service
	websocketService := websocket.NewWebSocketService(inferenceService, dveManager, validationCore, teeSecurityService)

	// Initialize CDE service
	cdeService, err := cde.NewCDEService(nil, dataEngine, cde.CDEConfig{
		BaseImagePath:          cfg.CDE.BaseImagePath,
		WorkspaceRoot:          cfg.CDE.WorkspaceRoot,
		MaxEnvironments:        cfg.CDE.MaxEnvironments,
		DefaultTimeout:         cfg.CDE.DefaultTimeout,
		MaxCPUPerEnv:           cfg.CDE.MaxCPUPerEnv,
		MaxMemoryPerEnv:        cfg.CDE.MaxMemoryPerEnv,
		MaxDiskPerEnv:          cfg.CDE.MaxDiskPerEnv,
		EnableSandboxing:       cfg.CDE.EnableSandboxing,
		EnableNetworkIsolation: cfg.CDE.EnableNetworkIsolation,
		AllowedPorts:           cfg.CDE.AllowedPorts,
		SessionTimeout:         cfg.CDE.SessionTimeout,
		MaxSessionsPerUser:     cfg.CDE.MaxSessionsPerUser,
		MaxProjectsPerUser:     cfg.CDE.MaxProjectsPerUser,
		ProjectStoragePath:     cfg.CDE.ProjectStoragePath,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize CDE service: %w", err)
	}

	// Initialize DNS service with minimal configuration for development
	dnsConfig := dns.DNSConfig{
		CloudFlareAPIToken:  "dev-token", // Placeholder for development
		ZoneName:            "knirv.com",
		UpdateInterval:      time.Minute * 5,
		ForceUpdateInterval: time.Hour,
		Records: []dns.DNSRecordConfig{
			{
				Name:         "nexus.knirv.com",
				Type:         "A",
				TTL:          300,
				UpdateWithIP: true,
			},
		},
		EnableHealthCheck: false, // Disable for development
		MaxRetries:        3,
		RetryDelay:        time.Second * 5,
		BackoffFactor:     2.0,
	}

	dnsService, err := dns.NewDynamicDNSService(dataEngine, dnsConfig)
	if err != nil {
		log.Printf("Warning: Failed to initialize DNS service: %v", err)
		// Continue without DNS service for now
		dnsService = nil
	}

	// Initialize DVE Rental service
	dveRentalService, err := dverental.NewDVERentalService(dbManager.GetDB())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize DVE rental service: %w", err)
	}
	dveRentalService.SetServiceReferences(dveManager, cdeService)

	// Create context for service lifecycle management
	ctx, cancel := context.WithCancel(context.Background())

	server := &Server{
		config:                       cfg,
		db:                           dbManager,
		router:                       router,
		p2pManager:                   p2pManager,
		dveManager:                   dveManager,
		validationCore:               validationCore,
		cdeService:                   cdeService,
		dnsService:                   dnsService,
		modelServer:                  modelServer,
		dataEngine:                   dataEngine,
		inferenceService:             inferenceService,
		websocketService:             websocketService,
		teeSecurityService:           teeSecurityService,
		systemHealthService:          systemHealthService,
		modelManagementService:       modelManagementService,
		controllerIntegrationService: controllerIntegrationService,
		dveRentalService:             dveRentalService,
		ctx:                          ctx,
		cancel:                       cancel,
		running:                      false,
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

	// Auth routes (before other protected routes)
	if authMiddleware != nil {
		authHandlers := web.NewAuthHandlers(s.db, authMiddleware)
		s.router.HandleFunc("/api/auth/login", authHandlers.Login).Methods("POST")
		s.router.HandleFunc("/api/auth/refresh", authHandlers.Refresh).Methods("POST")
		s.router.HandleFunc("/api/auth/revoke", authHandlers.Revoke).Methods("POST")
		s.router.HandleFunc("/api/auth/me", authHandlers.Me).Methods("GET")
		log.Println("Auth routes configured")
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

	// Register model server routes
	if s.modelServer != nil {
		s.modelServer.RegisterRoutes(s.router)
	}

	// Register CDE service routes (when available)
	if s.cdeService != nil {
		s.cdeService.RegisterRoutes(s.router, authMiddleware)
	}

	// Register DNS service routes (when available)
	if s.dnsService != nil {
		s.dnsService.RegisterRoutes(s.router, authMiddleware)
	}

	// Register inference service routes
	if s.inferenceService != nil {
		inferenceHandlers := web.NewInferenceHandlers(s.inferenceService)
		inferenceHandlers.RegisterRoutes(s.router, authMiddleware)
		log.Println("Inference service routes configured")
	}

	// Register WebSocket service routes
	if s.websocketService != nil {
		s.websocketService.RegisterRoutes(s.router)
		log.Println("WebSocket service routes configured")
	}

	// Register DVE manager routes
	if s.dveManager != nil {
		dveHandlers := web.NewDVEHandlers(s.dveManager, s.dveRentalService)
		dveHandlers.RegisterRoutes(s.router, authMiddleware)
		log.Println("DVE manager routes configured")
	}

	// Register validation service routes
	if s.validationCore != nil {
		validationHandlers := web.NewValidationHandlers(s.validationCore)
		validationHandlers.RegisterRoutes(s.router, authMiddleware)
		log.Println("Validation service routes configured")
	}

	// Register TEE security service routes
	if s.teeSecurityService != nil {
		teeSecurityHandlers := web.NewTEESecurityHandlers(s.teeSecurityService)
		teeSecurityHandlers.RegisterRoutes(s.router, authMiddleware)
		log.Println("TEE security service routes configured")
	}

	// Register system health service routes
	if s.systemHealthService != nil {
		systemHealthHandlers := web.NewSystemHealthHandlers(s.systemHealthService)
		systemHealthHandlers.RegisterRoutes(s.router, authMiddleware)
		log.Println("System health service routes configured")
	}

	// Register model management service routes
	if s.modelManagementService != nil {
		modelManagementHandlers := web.NewModelManagementHandlers(s.modelManagementService)
		modelManagementHandlers.RegisterRoutes(s.router, authMiddleware)
		log.Println("Model management service routes configured")
	}

	// Register controller integration service routes
	if s.controllerIntegrationService != nil {
		controllerIntegrationHandlers := web.NewControllerIntegrationHandlers(s.controllerIntegrationService)
		controllerIntegrationHandlers.RegisterRoutes(s.router, authMiddleware)
		log.Println("Controller integration service routes configured")
	}

	// Register DVE rental service routes
	if s.dveRentalService != nil {
		dveRentalHandlers := web.NewDVERentalHandlers(s.dveRentalService)
		dveRentalHandlers.RegisterRoutes(s.router, authMiddleware)
		log.Println("DVE rental service routes configured")
	}

	log.Println("All routes configured successfully")
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
			"database":          s.db != nil,
			"p2p_manager":       s.p2pManager != nil,
			"dve_manager":       s.dveManager != nil,
			"validation_core":   s.validationCore != nil,
			"model_server":      s.modelServer != nil,
			"data_engine":       s.dataEngine != nil,
			"inference_service": s.inferenceService != nil,
			"websocket_service": s.websocketService != nil,
			"cde_service":       s.cdeService != nil,
			"dns_service":       s.dnsService != nil,
		},
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
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

	// Start model server
	if s.modelServer != nil {
		if err := s.modelServer.Start(); err != nil {
			return fmt.Errorf("failed to start model server: %w", err)
		}
		log.Println("Model Server started")
	}

	// Start CDE service
	if s.cdeService != nil {
		if err := s.cdeService.Start(); err != nil {
			return fmt.Errorf("failed to start CDE service: %w", err)
		}
		log.Println("CDE Service started")
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

	// Start inference service
	if s.inferenceService != nil {
		if err := s.inferenceService.Start(); err != nil {
			return fmt.Errorf("failed to start inference service: %w", err)
		}
		log.Println("Inference Service started")
	}

	// Start TEE Security service
	if s.teeSecurityService != nil {
		if err := s.teeSecurityService.Start(); err != nil {
			return fmt.Errorf("failed to start TEE Security service: %w", err)
		}
		log.Println("TEE Security Service started")
	}

	// Start System Health service
	if s.systemHealthService != nil {
		if err := s.systemHealthService.Start(); err != nil {
			return fmt.Errorf("failed to start System Health service: %w", err)
		}
		log.Println("System Health Service started")
	}

	// Start Model Management service
	if s.modelManagementService != nil {
		if err := s.modelManagementService.Start(); err != nil {
			return fmt.Errorf("failed to start Model Management service: %w", err)
		}
		log.Println("Model Management Service started")
	}

	// Start Controller Integration service
	if s.controllerIntegrationService != nil {
		if err := s.controllerIntegrationService.Start(); err != nil {
			return fmt.Errorf("failed to start Controller Integration service: %w", err)
		}
		log.Println("Controller Integration Service started")
	}

	// Start DVE Rental service
	if s.dveRentalService != nil {
		if err := s.dveRentalService.Start(); err != nil {
			return fmt.Errorf("failed to start DVE Rental service: %w", err)
		}
		log.Println("DVE Rental Service started")
	}

	// Start WebSocket service
	if s.websocketService != nil {
		if err := s.websocketService.Start(); err != nil {
			return fmt.Errorf("failed to start WebSocket service: %w", err)
		}
		log.Println("WebSocket Service started")
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
	log.Println("KNIRV-NEXUS backend server started successfully")
	return nil
}

// Stop stops all services and the HTTP server
func (s *Server) Stop() error {
	if !s.running {
		return nil
	}

	log.Println("Stopping KNIRV-NEXUS backend server...")

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
	if s.websocketService != nil {
		if err := s.websocketService.Stop(); err != nil {
			log.Printf("Error stopping WebSocket service: %v", err)
		}
	}

	if s.teeSecurityService != nil {
		if err := s.teeSecurityService.Stop(); err != nil {
			log.Printf("Error stopping TEE Security service: %v", err)
		}
	}

	if s.systemHealthService != nil {
		if err := s.systemHealthService.Stop(); err != nil {
			log.Printf("Error stopping System Health service: %v", err)
		}
	}

	if s.modelManagementService != nil {
		if err := s.modelManagementService.Stop(); err != nil {
			log.Printf("Error stopping Model Management service: %v", err)
		}
	}

	if s.controllerIntegrationService != nil {
		if err := s.controllerIntegrationService.Stop(); err != nil {
			log.Printf("Error stopping Controller Integration service: %v", err)
		}
	}

	if s.dveRentalService != nil {
		if err := s.dveRentalService.Stop(); err != nil {
			log.Printf("Error stopping DVE Rental service: %v", err)
		}
	}

	if s.cdeService != nil {
		if err := s.cdeService.Stop(); err != nil {
			log.Printf("Error stopping CDE service: %v", err)
		}
	}

	if s.inferenceService != nil {
		if err := s.inferenceService.Stop(); err != nil {
			log.Printf("Error stopping inference service: %v", err)
		}
	}

	if s.dataEngine != nil {
		if err := s.dataEngine.Stop(); err != nil {
			log.Printf("Error stopping data engine: %v", err)
		}
	}

	if s.modelServer != nil {
		if err := s.modelServer.Stop(); err != nil {
			log.Printf("Error stopping model server: %v", err)
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
	log.Println("KNIRV-NEXUS backend server stopped")
	return nil
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

	// Create a context with timeout for graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Channel to signal when shutdown is complete
	shutdownComplete := make(chan error, 1)

	// Start shutdown in a goroutine
	go func() {
		shutdownComplete <- server.Stop()
	}()

	// Wait for either shutdown completion or timeout
	select {
	case err := <-shutdownComplete:
		if err != nil {
			log.Printf("Error during shutdown: %v", err)
		} else {
			log.Println("Server stopped gracefully")
		}
	case <-shutdownCtx.Done():
		log.Println("Shutdown timeout reached, forcing exit")
		// Cancel the server context to force stop all services
		server.cancel()
		// Give a brief moment for forced shutdown
		time.Sleep(1 * time.Second)
	}
}
