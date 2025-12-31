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

	"backend_server/internal/config"
	dataengine "backend_server/internal/data-engine"
	"backend_server/internal/database"
	inference "backend_server/internal/inference_engine"
	"backend_server/internal/services/blockchain"
	"backend_server/internal/services/cde"
	"backend_server/internal/services/cognitiveengine"
	"backend_server/internal/services/container"
	"backend_server/internal/services/controllerintegration"
	"backend_server/internal/services/dns"
	"backend_server/internal/services/dvemanager"
	"backend_server/internal/services/dverental"
	"backend_server/internal/services/endpoints"
	modelserver "backend_server/internal/services/model-server"
	"backend_server/internal/services/modelmanagement"
	"backend_server/internal/services/payment"
	"backend_server/internal/services/session"
	"backend_server/internal/services/systemhealth"
	"backend_server/internal/services/teesecurity"
	"backend_server/internal/services/validation"
	"backend_server/internal/services/websocket"
	"backend_server/internal/web"
	"backend_server/internal/web/middleware"
	"backend_server/pkg/p2p"

	"github.com/gorilla/mux"
	"github.com/spf13/viper"
	"github.com/tidwall/buntdb"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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
	logger     *zap.Logger

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
	cognitiveEngine              *cognitiveengine.CognitiveEngine
	containerOrchestrator        *container.ContainerOrchestrator
	sessionManager               *session.SessionManager
	endpointRegistry             *endpoints.EndpointRegistry
	stripeService                *payment.StripeService
	paypalService                *payment.PayPalService

	// Context for managing service lifecycle
	ctx    context.Context
	cancel context.CancelFunc

	// State management
	running bool
}

// initLogging initializes the logging system based on configuration
func initLogging(cfg *config.Config) (*zap.Logger, error) {
	// Determine log level
	var level zapcore.Level
	switch cfg.Log.Level {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	default:
		level = zapcore.InfoLevel
	}

	// Create encoder config
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// Determine log file path
	logFilePath := cfg.Log.Output
	if logFilePath == "" {
		// Use OS-specific application data directory
		appDataDir, err := getOSAppDataDir()
		if err != nil {
			// Fallback to relative path
			logFilePath = "logs/server.log"
		} else {
			logFilePath = filepath.Join(appDataDir, "logs", "server.log")
		}
	}

	// Ensure the log directory exists
	logDir := filepath.Dir(logFilePath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Create file writer
	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	// Create cores for both file and stdout
	fileCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(file),
		level,
	)

	stdoutCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(os.Stdout),
		level,
	)

	// Combine cores
	core := zapcore.NewTee(fileCore, stdoutCore)

	// Create logger
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	// Also redirect standard log output to file for compatibility
	log.SetOutput(file)

	return logger, nil
}

// getOSAppDataDir returns the OS-specific application data directory
func getOSAppDataDir() (string, error) {
	var appDataDir string
	var err error

	// Try to get user config directory (XDG Base Directory on Linux)
	if userConfigDir, configErr := os.UserConfigDir(); configErr == nil {
		appDataDir = filepath.Join(userConfigDir, "knirvnexus")
	} else {
		// Fallback to home directory
		if homeDir, homeErr := os.UserHomeDir(); homeErr == nil {
			// Use XDG_DATA_HOME or fallback to ~/.local/share
			if xdgDataHome := os.Getenv("XDG_DATA_HOME"); xdgDataHome != "" {
				appDataDir = filepath.Join(xdgDataHome, "knirvnexus")
			} else {
				appDataDir = filepath.Join(homeDir, ".local", "share", "knirvnexus")
			}
		} else {
			return "", fmt.Errorf("could not determine application data directory")
		}
	}

	// Ensure directory exists
	if err = os.MkdirAll(appDataDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create application data directory: %w", err)
	}

	return appDataDir, nil
}

// NewServer creates a new KNIRV-NEXUS backend server instance
func NewServer(cfg *config.Config) (*Server, error) {
	// Initialize logging
	logger, err := initLogging(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logging: %w", err)
	}

	// Initialize database
	dbManager, err := database.NewBuntDB(cfg.Database.Path)
	if err != nil {
		logger.Error("Failed to initialize database", zap.Error(err))
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Initialize P2P manager
	p2pManager, err := p2p.NewDVEP2PManager(cfg.ChainID, cfg.NodeRole, dbManager.GetDB(), cfg.P2P.DHTEnabled)
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
		WebSocketPort:    8082,
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

	// Initialize TEE environment with Kali-focused detection
	if err := initializeTEEEnvironment(context.Background(), dbManager.GetDB()); err != nil {
		log.Printf("Warning: TEE environment initialization failed: %v", err)
		// Continue - TEE initialization is not critical for basic operation
	}

	// Initialize System Health service
	systemHealthService := systemhealth.NewSystemHealthService(dbManager.GetDB())
	systemHealthService.SetServiceReferences(dveManager, validationCore, inferenceService, teeSecurityService)

	// Initialize Model Management service
	modelManagementService := modelmanagement.NewModelManagementService(dbManager.GetDB())
	modelManagementService.SetModelServerReference(modelServer)

	// Initialize Controller Integration service
	controllerIntegrationService := controllerintegration.NewControllerIntegrationService(dbManager.GetDB())

	// Initialize CDE service with TEE security integration
	cdeService, err := cde.NewCDEService(teeSecurityService, dataEngine, cde.CDEConfig{
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

	// Initialize Container Orchestrator with Kali-aware runtime selection
	containerConfig := &container.ContainerConfig{
		ContainerRuntime:         getContainerRuntime(teeSecurityService),
		BaseImage:                "ubuntu:20.04",
		SSHPortRangeStart:        22000,
		SSHPortRangeEnd:          22999,
		ValidationPortRangeStart: 23000,
		ValidationPortRangeEnd:   23999,
		ErrorResPortRangeStart:   24000,
		ErrorResPortRangeEnd:     24999,
		ProvisioningTimeout:      5 * time.Minute,
		CleanupInterval:          10 * time.Minute,
	}
	containerOrchestrator, err := container.NewContainerOrchestrator(containerConfig, teeSecurityService)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize container orchestrator: %w", err)
	}

	// Initialize DVE Rental service
	dveRentalService, err := dverental.NewDVERentalService(dbManager.GetDB())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize DVE rental service: %w", err)
	}
	dveRentalService.SetServiceReferences(dveManager, cdeService)
	dveRentalService.SetContainerOrchestrator(containerOrchestrator)

	// Initialize NRN blockchain client for payment verification
	nrnClient := blockchain.NewNRNClient("http://localhost:8082") // TODO: Make configurable
	dveRentalService.SetBlockchainClient(nrnClient)

	// Initialize Session Manager
	sessionManager := session.NewSessionManager(dbManager.GetDB())

	// Initialize Endpoint Registry
	endpointRegistry := endpoints.NewEndpointRegistry(dbManager.GetDB())

	// Initialize Payment Services
	stripeService := payment.NewStripeService(
		"s_k_test_example", // TODO: Load from environment
		"pk_test_example",  // TODO: Load from environment
		"whsec_example",    // TODO: Load from environment
		"usd",
		"2023-10-16",
	)

	paypalService := payment.NewPayPalService(
		"client_id_example", // TODO: Load from environment
		"secret_example",    // TODO: Load from environment
		"sandbox",           // TODO: Load from environment
		"USD",
	)

	// Initialize Cognitive Engine
	cognitiveEngine := cognitiveengine.NewCognitiveEngine(dbManager.GetDB(), validationCore, inferenceService, modelManagementService)

	// Create context for service lifecycle management
	ctx, cancel := context.WithCancel(context.Background())

	server := &Server{
		config:                       cfg,
		db:                           dbManager,
		router:                       router,
		p2pManager:                   p2pManager,
		logger:                       logger,
		dveManager:                   dveManager,
		validationCore:               validationCore,
		cdeService:                   cdeService,
		dnsService:                   dnsService,
		modelServer:                  modelServer,
		dataEngine:                   dataEngine,
		inferenceService:             inferenceService,
		websocketService:             nil, // Will be set in setupRoutes
		teeSecurityService:           teeSecurityService,
		systemHealthService:          systemHealthService,
		modelManagementService:       modelManagementService,
		controllerIntegrationService: controllerIntegrationService,
		dveRentalService:             dveRentalService,
		cognitiveEngine:              cognitiveEngine,
		containerOrchestrator:        containerOrchestrator,
		sessionManager:               sessionManager,
		endpointRegistry:             endpointRegistry,
		stripeService:                stripeService,
		paypalService:                paypalService,
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
	var authMiddleware *middleware.AuthMiddleware
	log.Printf("DEBUG: Auth required: %v, JWT secret length: %d", s.config.Security.AuthRequired, len(s.config.Security.JWTSecret))
	if s.config.Security.AuthRequired {
		var err error
		authMiddleware, err = middleware.NewAuthMiddleware(s.db, s.config.Security.JWTSecret)
		if err != nil {
			log.Printf("Warning: Failed to create auth middleware: %v", err)
			authMiddleware = nil
		}
	} else {
		log.Println("Auth disabled for testnet mode")
		authMiddleware = nil
	}
	log.Printf("DEBUG: authMiddleware is nil: %v", authMiddleware == nil)

	// Initialize WebSocket service after auth middleware is created
	wsService := websocket.NewWebSocketService(s.inferenceService, s.dveManager, s.validationCore, s.sessionManager, s.teeSecurityService)
	if authMiddleware != nil {
		wsService.SetAuthMiddleware(authMiddleware)
	}
	wsService.SetDatabase(s.db)
	s.websocketService = wsService

	// Wire Controller Integration service with WebSocket service
	if s.controllerIntegrationService != nil {
		s.controllerIntegrationService.SetWebSocketService(wsService)
	}

	// Auth routes (before other protected routes)
	if authMiddleware != nil {
		authHandlers := web.NewAuthHandlers(s.db, authMiddleware)
		// Public auth routes
		s.router.HandleFunc("/api/auth/register", authHandlers.Register).Methods("POST", "OPTIONS")
		s.router.HandleFunc("/api/auth/login", authHandlers.Login).Methods("POST", "OPTIONS")
		s.router.HandleFunc("/api/auth/refresh", authHandlers.Refresh).Methods("POST", "OPTIONS")
		s.router.HandleFunc("/api/auth/revoke", authHandlers.Revoke).Methods("POST", "OPTIONS")
		s.router.HandleFunc("/api/auth/verify-email", authHandlers.VerifyEmail).Methods("POST", "OPTIONS")
		s.router.HandleFunc("/api/auth/request-password-reset", authHandlers.RequestPasswordReset).Methods("POST", "OPTIONS")
		s.router.HandleFunc("/api/auth/reset-password", authHandlers.ResetPassword).Methods("POST", "OPTIONS")

		// Protected auth routes
		protectedAuthRouter := s.router.PathPrefix("/api/auth").Subrouter()
		protectedAuthRouter.Use(authMiddleware.RequireAuth)
		protectedAuthRouter.HandleFunc("/me", authHandlers.Me).Methods("GET", "OPTIONS")
		protectedAuthRouter.HandleFunc("/change-password", authHandlers.ChangePassword).Methods("POST", "OPTIONS")
		protectedAuthRouter.HandleFunc("/update-profile", authHandlers.UpdateProfile).Methods("PUT", "OPTIONS")
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
		dveRentalHandlers := web.NewDVERentalHandlers(s.dveRentalService, s.containerOrchestrator, s.sessionManager, s.endpointRegistry, s.db.GetDB())

		// Pass the main router directly - the handler will create its own subrouter with the correct prefix
		dveRentalHandlers.RegisterRoutes(s.router, authMiddleware)
		log.Println("DVE rental service routes configured")
	}

	// Register cognitive engine routes
	if s.cognitiveEngine != nil {
		cognitiveEngineHandlers := web.NewCognitiveEngineHandlers(s.cognitiveEngine)
		cognitiveEngineHandlers.RegisterRoutes(s.router, authMiddleware)
		log.Println("Cognitive engine routes configured")
	}

	// Register payment service routes
	paymentHandlers := web.NewPaymentHandlers(s.stripeService, s.paypalService)
	paymentHandlers.RegisterRoutes(s.router, authMiddleware)
	log.Println("Payment service routes configured")

	// Register system settings routes
	systemSettingsHandlers := web.NewSystemSettingsHandlers(s.config)
	systemSettingsHandlers.RegisterRoutes(s.router, authMiddleware)
	log.Println("System settings routes configured")

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
			"database":           s.db != nil,
			"p2p_manager":        s.p2pManager != nil,
			"dve_manager":        s.dveManager != nil,
			"validation_core":    s.validationCore != nil,
			"model_server":       s.modelServer != nil,
			"data_engine":        s.dataEngine != nil,
			"inference_service":  s.inferenceService != nil,
			"websocket_service":  s.websocketService != nil,
			"cde_service":        s.cdeService != nil,
			"dns_service":        s.dnsService != nil,
			"dve_rental_service": s.dveRentalService != nil,
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

	// Validate server state before starting
	if s.config == nil {
		return fmt.Errorf("server configuration is nil")
	}
	if s.db == nil {
		return fmt.Errorf("database is not initialized")
	}
	if s.router == nil {
		return fmt.Errorf("router is not initialized")
	}

	// Start P2P manager
	if s.p2pManager != nil {
		s.p2pManager.Start() // P2P manager start doesn't return error
		log.Println("P2P Manager started")
	} else {
		log.Println("Warning: P2P Manager not initialized, skipping")
	}

	// Start DVE manager
	if s.dveManager != nil {
		if err := s.dveManager.Start(s.ctx); err != nil {
			log.Printf("Warning: Failed to start DVE manager: %v", err)
			// Continue starting other services - DVE manager failure shouldn't stop the server
		} else {
			log.Println("DVE Manager started")
		}
	}

	// Start validation core
	if s.validationCore != nil {
		if err := s.validationCore.Start(s.ctx); err != nil {
			log.Printf("Warning: Failed to start validation core: %v", err)
			// Continue - validation core failure shouldn't stop basic server operation
		} else {
			log.Println("Validation Core started")
		}
	}

	// Start model server
	if s.modelServer != nil {
		if err := s.modelServer.Start(); err != nil {
			log.Printf("Warning: Failed to start model server: %v", err)
			// Continue - model server failure shouldn't stop basic server operation
		} else {
			log.Println("Model Server started")
		}
	}

	// Start CDE service
	if s.cdeService != nil {
		if err := s.cdeService.Start(); err != nil {
			log.Printf("Warning: Failed to start CDE service: %v", err)
			// Continue - CDE service failure shouldn't stop basic server operation
		} else {
			log.Println("CDE Service started")
		}
	}

	// Start DNS service (when available)
	if s.dnsService != nil {
		// DNS service start will be implemented when available
	}

	// Start data engine
	if s.dataEngine != nil {
		if err := s.dataEngine.Start(); err != nil {
			log.Printf("Warning: Failed to start data engine: %v", err)
			// Continue - data engine failure shouldn't stop basic server operation
		} else {
			log.Println("Data Engine started")
		}
	}

	// Start inference service
	if s.inferenceService != nil {
		if err := s.inferenceService.Start(); err != nil {
			log.Printf("Warning: Failed to start inference service: %v", err)
			// Continue - inference service failure shouldn't stop basic server operation
		} else {
			log.Println("Inference Service started")
		}
	}

	// Start TEE Security service
	if s.teeSecurityService != nil {
		if err := s.teeSecurityService.Start(); err != nil {
			log.Printf("Warning: Failed to start TEE Security service: %v", err)
			// Continue - TEE security failure shouldn't stop basic server operation
		} else {
			log.Println("TEE Security Service started")
		}
	}

	// Start System Health service
	if s.systemHealthService != nil {
		if err := s.systemHealthService.Start(); err != nil {
			log.Printf("Warning: Failed to start System Health service: %v", err)
			// Continue - system health failure shouldn't stop basic server operation
		} else {
			log.Println("System Health Service started")
		}
	}

	// Start Model Management service
	if s.modelManagementService != nil {
		if err := s.modelManagementService.Start(); err != nil {
			log.Printf("Warning: Failed to start Model Management service: %v", err)
			// Continue - model management failure shouldn't stop basic server operation
		} else {
			log.Println("Model Management Service started")
		}
	}

	// Start Controller Integration service
	if s.controllerIntegrationService != nil {
		if err := s.controllerIntegrationService.Start(); err != nil {
			log.Printf("Warning: Failed to start Controller Integration service: %v", err)
			// Continue - controller integration failure shouldn't stop basic server operation
		} else {
			log.Println("Controller Integration Service started")
		}
	}

	// Start Container Orchestrator
	if s.containerOrchestrator != nil {
		if err := s.containerOrchestrator.Start(s.ctx); err != nil {
			log.Printf("Warning: Failed to start Container Orchestrator: %v", err)
			// Continue - container orchestrator failure shouldn't stop basic server operation
		} else {
			log.Println("Container Orchestrator started")
		}
	}

	// Start DVE Rental service
	if s.dveRentalService != nil {
		if err := s.dveRentalService.Start(); err != nil {
			log.Printf("Warning: Failed to start DVE Rental service: %v", err)
			// Continue - DVE rental failure shouldn't stop basic server operation
		} else {
			log.Println("DVE Rental Service started")
		}
	}

	// Start WebSocket service
	if s.websocketService != nil {
		if err := s.websocketService.Start(); err != nil {
			log.Printf("Warning: Failed to start WebSocket service: %v", err)
			// Continue - WebSocket service failure shouldn't stop basic server operation
		} else {
			log.Println("WebSocket Service started")
		}
	}

	// Start Cognitive Engine
	if s.cognitiveEngine != nil {
		if err := s.cognitiveEngine.Start(); err != nil {
			log.Printf("Warning: Failed to start Cognitive Engine: %v", err)
			// Continue - cognitive engine failure shouldn't stop basic server operation
		} else {
			log.Println("Cognitive Engine started")
		}
	}

	// Validate server configuration before creating HTTP server
	if s.config == nil || s.config.API.BindAddress == "" || s.config.API.Port <= 0 {
		return fmt.Errorf("invalid server configuration: bind address and port must be specified")
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
			log.Printf("HTTP server error: %v", err)
			// Don't call log.Fatalf here as it would terminate the entire process
			// Instead, let the server continue and handle the error gracefully
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

	// Validate server state before stopping
	if s.config == nil {
		log.Println("Warning: Server configuration is nil during shutdown")
	}

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
			// Continue with shutdown even if HTTP server shutdown fails
		} else {
			log.Println("HTTP server shut down gracefully")
		}
	}

	// Stop services in reverse order
	if s.cognitiveEngine != nil {
		if err := s.cognitiveEngine.Stop(); err != nil {
			log.Printf("Error stopping Cognitive Engine: %v", err)
		}
	}

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

	if s.containerOrchestrator != nil {
		if err := s.containerOrchestrator.Stop(); err != nil {
			log.Printf("Error stopping Container Orchestrator: %v", err)
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
			// Continue with shutdown even if database close fails
		} else {
			log.Println("Database closed successfully")
		}
	}

	s.running = false
	log.Println("KNIRV-NEXUS backend server stopped")
	return nil
}

// initializeTEEEnvironment sets up the TEE environment with Kali-focused detection
func initializeTEEEnvironment(ctx context.Context, db *buntdb.DB) error {
	// Initialize TEE Security Service (detects Kali and available tools)
	teeService, err := teesecurity.NewTEESecurityService(db)
	if err != nil {
		return fmt.Errorf("TEE service initialization failed: %v", err)
	}

	kaliProfile := teeService.GetKaliProfile()
	log.Printf("Detected OS: %s (Kali: %v)", kaliProfile.OS, kaliProfile.IsKaliLinux)
	log.Printf("Active Runtime: %s", teeService.GetRuntimeManager().GetActiveRuntime())

	// Create security tools validator
	validator := teesecurity.NewKaliSecurityValidator(kaliProfile)

	// Validate all Kali security tools and frameworks
	validationReport, err := validator.ValidateSecurityCapabilities(ctx)
	if err != nil {
		return fmt.Errorf("security validation failed: %v", err)
	}

	// Log validation results
	logSecurityValidationReport(validationReport)

	// Log recommendations
	if len(validationReport.Recommendations) > 0 {
		log.Println("\nSecurity Tools Recommendations:")
		for i, rec := range validationReport.Recommendations {
			log.Printf("  %d. %s", i+1, rec)
		}
	}

	return nil
}

// getContainerRuntime determines the appropriate container runtime based on Kali Linux detection
func getContainerRuntime(teeService *teesecurity.TEESecurityService) string {
	if teeService == nil {
		log.Println("Warning: TEE service not available, defaulting to docker runtime")
		return "docker"
	}

	kaliProfile := teeService.GetKaliProfile()
	if kaliProfile.IsKaliLinux {
		log.Printf("Kali Linux detected, using native-go container runtime with security tools")
		return "native-go"
	}

	log.Printf("Non-Kali system detected (%s), using podman fallback runtime", kaliProfile.OS)
	return "podman"
}

// logSecurityValidationReport logs the Kali security validation report
func logSecurityValidationReport(report *teesecurity.KaliSecurityValidationReport) {
	log.Println("\n=== Kali Linux Security Tools Validation Report ===")
	log.Printf("OS: %s (Kali: %v)", report.OS, report.IsKaliLinux)
	log.Printf("Timestamp: %s", report.Timestamp.String())

	log.Println("\nTools Availability:")
	for tool, available := range report.ToolsAvailable {
		status := "✓ Available"
		if !available {
			status = "✗ Missing"
		}
		log.Printf("  %s - %s", tool, status)
	}

	log.Println("\nSecurity Frameworks:")
	for framework, loaded := range report.FrameworksLoaded {
		status := "✓ Loaded"
		if !loaded {
			status = "✗ Not Loaded"
		}
		log.Printf("  %s - %s", framework, status)
	}

	log.Println("\nSystem Resources:")
	log.Printf("  Memory: %s KB", report.SystemMemoryKB)
	log.Printf("  Disk Space: %s KB", report.DiskSpaceKB)
}

func run() error {
	// Parse command line flags
	var configFile = flag.String("config", "", "Path to configuration file")
	flag.Parse()

	// Print version information
	fmt.Printf("KNIRV-NEXUS Complete Backend Server v%s (built %s, commit %s)\n", Version, BuildTime, GitCommit)

	// Validate command line arguments
	if len(flag.Args()) > 0 {
		return fmt.Errorf("unexpected arguments: %v", flag.Args())
	}

	// Set config file if provided, otherwise use relative path from backend directory
	if *configFile != "" {
		if _, err := os.Stat(*configFile); os.IsNotExist(err) {
			return fmt.Errorf("config file does not exist: %s", *configFile)
		}
		viper.SetConfigFile(*configFile)
	} else {
		// Try app data directory first
		appDataDir, _ := getOSAppDataDir()
		appDataConfigPath := filepath.Join(appDataDir, "config", "production.yaml")
		if _, err := os.Stat(appDataConfigPath); err == nil {
			viper.SetConfigFile(appDataConfigPath)
			log.Printf("Using config from app data directory: %s", appDataConfigPath)
		} else {
			// Fallback to local config path
			configPath := filepath.Join("config", "production.yaml")
			if _, err := os.Stat(configPath); err == nil {
				viper.SetConfigFile(configPath)
			} else {
				log.Printf("Warning: Default config file not found at %s or %s", appDataConfigPath, configPath)
			}
		}
	}

	// Load configuration
	config, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %v", err)
	}

	// Validate essential configuration
	if config == nil {
		return fmt.Errorf("configuration is nil")
	}
	if config.Database.Path == "" {
		return fmt.Errorf("database path is not configured")
	}

	// Create unified server
	server, err := NewServer(config)
	if err != nil {
		return fmt.Errorf("failed to create server: %v", err)
	}

	// Start the server
	if err := server.Start(); err != nil {
		return fmt.Errorf("failed to start server: %v", err)
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
			return fmt.Errorf("error during shutdown: %v", err)
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

	return nil
}

func main() {
	if err := run(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}
