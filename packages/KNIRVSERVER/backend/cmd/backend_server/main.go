package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"backend_server/internal/api"
	"backend_server/internal/config"
	data_engine "backend_server/internal/data_engine"
	"backend_server/internal/database"
	"backend_server/internal/ebpf"
	"backend_server/internal/logging"
	"backend_server/internal/password"
	pb "backend_server/internal/proto"
	"backend_server/internal/reasoning/graph"
	"backend_server/internal/runtime"
	nexus "backend_server/internal/server"
	"backend_server/internal/services/active_memory"
	agentsvc "backend_server/internal/services/agent"
	transactionchain "backend_server/internal/services/blockchain/transactionchain"
	"backend_server/internal/services/blockchain/validationchain"
	"backend_server/internal/services/cognitiveengine"
	"backend_server/internal/services/container"
	"backend_server/internal/services/controllerintegration"
	"backend_server/internal/services/dns"
	"backend_server/internal/services/dvecreation"
	"backend_server/internal/services/dvemanager"
	dverental "backend_server/internal/services/dverental"
	"backend_server/internal/services/endpoints"
	"backend_server/internal/services/evidence"
	"backend_server/internal/services/guardrails"
	icme "backend_server/internal/services/icme"
	inference "backend_server/internal/services/inferencer"
	"backend_server/internal/services/knowledge_base"
	"backend_server/internal/services/onboarding"
	"backend_server/internal/services/p2p"
	fabricmanagement "backend_server/internal/services/pluginmanagement"
	pluginserver "backend_server/internal/services/pluginserver"
	"backend_server/internal/services/rollup"
	secrets "backend_server/internal/services/secrets"
	"backend_server/internal/services/session"
	"backend_server/internal/services/systemhealth"
	"backend_server/internal/services/teesecurity"
	"backend_server/internal/services/validation"
	"backend_server/internal/utils/host"

	knirvshell "github.com/KNIRV/KNIRV_NETWORK/KNIRVSERVER/pkg/knirvshell"

	knirvchain "KNIRVCHAIN"
	knirvgraph "KNIRVGRAPH"
	"backend_server/internal/services/vault"
	"backend_server/internal/services/websocket"
	"backend_server/internal/services/workflow"
	"backend_server/internal/storage/mdstorage"
	"backend_server/internal/storage/pqc"
	"backend_server/internal/web"
	"backend_server/internal/web/middleware"

	knirvgateway "github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY"
	knirvoracle "github.com/KNIRV/KNIRV_NETWORK/KNIRVSERVER/pkg/knirvoracle"

	"github.com/apache/arrow/go/v14/arrow/memory"
	"github.com/gorilla/mux"
	"github.com/spf13/viper"
	"github.com/subosito/gotenv"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/term"
)

// Version information (set by build flags)
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

// Server represents the KNIRV-SERVER backend server
type Server struct {
	config                 *config.Config
	db                     *database.BuntDBManager
	router                 *mux.Router
	httpServer             *http.Server
	p2pManager             *p2p.DVEP2PManager
	logger                 *zap.Logger
	transactionChainClient *transactionchain.Client
	validationChainClient  *validationchain.Client

	// eBPF subsystem
	ebpfManager             ebpf.ManagerInterface
	virtualContainerManager *ebpf.VirtualContainerManager

	// SERVER Memory Fabric
	nexusServer    *nexus.NexusMemoryServer
	nexusAllocator memory.Allocator

	// All services are held here
	dveManager                   *dvemanager.DVEManager
	dveCreationService           *dvecreation.DVECreationService
	dveRentalService             *dverental.DVERentalService
	validationCore               *validation.ValidationCore
	dnsService                   *dns.DynamicDNSService
	pluginServer                 *pluginserver.PluginServer
	dataEngine                   *data_engine.BuntDBDataEngine
	inferenceService             *inference.InferenceService
	websocketService             *websocket.WebSocketService
	teeSecurityService           *teesecurity.TEESecurityService
	systemHealthService          *systemhealth.SystemHealthService
	fabricManagementService      *fabricmanagement.PluginManagementService
	controllerIntegrationService *controllerintegration.ControllerIntegrationService
	cognitiveEngine              *cognitiveengine.CognitiveEngine
	containerOrchestrator        *container.ContainerOrchestrator
	sessionManager               *session.SessionManager
	endpointRegistry             *endpoints.EndpointRegistry
	gatewayManager               *knirvgateway.Manager
	graphManager                 *knirvgraph.Manager
	graphSyncManager             *knirvgraph.SyncManager
	chainManager                 *knirvchain.Manager
	transactionChainManager      *transactionchain.Manager
	validationChainManager       *validationchain.Manager

	// Active Memory Layer (Markdown Fabric)
	pqcManager          *pqc.EncryptionManager
	mdStorage           *mdstorage.MarkdownStorageDriver
	vaultService        *vault.VaultService
	reasoningEngine     *graph.ReasoningEngine
	activeMemoryService *active_memory.ActiveMemoryService

	// Object Nest subsystem
	unifiedContainerManager *runtime.UnifiedContainerManager

	// Agent runtime (oh-my-pi)
	agentService *agentsvc.AgentService

	// ICME - Intentional Context Memory Engine
	icmeService *icme.Service

	// KNIRVSHELL Backend Integration Service
	knirvshellService *knirvshell.KNIRVSHELLService

	// Onboarding Service - Value System and Ontology Ingestion
	onboardingService *onboarding.OnboardingService

	// Production Services (Phase 3, 4, 8)
	eventBroadcaster   *websocket.EventBroadcaster
	anchoringService   *evidence.AnchoringService
	secretManager      *secrets.SecretManager
	workflowService    *workflow.WorkflowService
	rollupService      *rollup.Service
	rollupPollInterval time.Duration
	guardrailManager   *guardrails.DynamicGuardrailManager
	policyEngine       *guardrails.PolicyEngine

	// KNIRVHASHER integration
	hasherGRPCServer  *dvemanager.HasherGRPCServer
	hasherIntegration *dvemanager.HasherIntegration

	// Oracle service (root-only — managed via knirvoracle Manager)
	oracleManager *knirvoracle.Manager

	// GraphRAG Knowledge Base Engine
	graphRAGClient *knowledge_base.GraphRAGClient

	// Context for managing service lifecycle
	ctx    context.Context
	cancel context.CancelFunc

	// State management
	running bool
}

type telemetryServiceAdapter struct {
	engine *cognitiveengine.CognitiveEngine
}

func (a *telemetryServiceAdapter) Latest() *cognitiveengine.SystemResourceSnapshot {
	if a == nil || a.engine == nil {
		return &cognitiveengine.SystemResourceSnapshot{Timestamp: time.Now()}
	}
	return a.engine.GetLatestTelemetry()
}

type ontologyServiceAdapter struct {
	engine *cognitiveengine.CognitiveEngine
}

func (a *ontologyServiceAdapter) GetEntity(_ string) (*cognitiveengine.OntologyEntity, bool) {
	return nil, false
}

func (a *ontologyServiceAdapter) QueryByType(_ cognitiveengine.OntologyEntityType) []*cognitiveengine.OntologyEntity {
	return []*cognitiveengine.OntologyEntity{}
}

func (a *ontologyServiceAdapter) FindRelations(_ string) []cognitiveengine.OntologyRelation {
	return []cognitiveengine.OntologyRelation{}
}

func (a *ontologyServiceAdapter) Stats() (entityCount int, relationCount int) {
	if a == nil || a.engine == nil {
		return 0, 0
	}
	return a.engine.GetOntologyStats()
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

	// Get application data directory for log storage
	appDataDir, err := getOSAppDataDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get application data directory: %w", err)
	}

	// Primary log path: application data directory
	appDataLogPath := filepath.Join(appDataDir, "logs", "server.log")

	// Ensure app data log directory exists
	appDataLogDir := filepath.Dir(appDataLogPath)
	if err := os.MkdirAll(appDataLogDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create app data log directory: %w", err)
	}

	// Create file writer for app data log
	appDataFile, err := os.OpenFile(appDataLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to open app data log file: %w", err)
	}

	// Secondary log path: project directory. Prefer an explicit absolute path from the outer
	// process, but fall back to the KNIRVSERVER project logs directory when we can locate it.
	var multiWriter io.Writer
	projectLogPath := ""
	if projectLogDir := getProjectLogDir(); projectLogDir != "" {
		projectLogPath = filepath.Join(projectLogDir, "server.log")
		if err := os.MkdirAll(projectLogDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create project log directory: %w", err)
		}
		projectFile, err := os.OpenFile(projectLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, fmt.Errorf("failed to open project log file: %w", err)
		}
		multiWriter = io.MultiWriter(appDataFile, projectFile)
	} else {
		multiWriter = appDataFile
	}

	// Create cores for file (both locations) and stdout
	fileCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(multiWriter),
		level,
	)

	stdoutCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(os.Stdout),
		level,
	)

	// Combine cores - logs go to both file locations AND stdout
	core := zapcore.NewTee(fileCore, stdoutCore)

	// Create logger
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	// Also redirect standard log output to file(s) for compatibility
	log.SetOutput(multiWriter)

	if projectLogPath != "" {
		logger.Info("Logging initialized",
			zap.String("app_data_log", appDataLogPath),
			zap.String("project_log", projectLogPath),
		)
	} else {
		logger.Info("Logging initialized",
			zap.String("app_data_log", appDataLogPath),
		)
	}

	return logger, nil
}

func getProjectLogDir() string {
	if projectLogDir := os.Getenv("KNIRV_PROJECT_LOG_DIR"); projectLogDir != "" {
		return projectLogDir
	}

	wd, err := os.Getwd()
	if err != nil {
		return ""
	}

	dir := wd
	for {
		if filepath.Base(dir) == "KNIRVSERVER" {
			if _, err := os.Stat(filepath.Join(dir, "main.go")); err == nil {
				return filepath.Join(dir, "logs")
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// getOSAppDataDir returns the OS-specific application data directory
func getOSAppDataDir() (string, error) {
	var appDataDir string
	var err error

	if explicit := os.Getenv("KNIRV_APP_DATA_DIR"); explicit != "" {
		appDataDir = explicit
	} else if xdgDataHome := os.Getenv("XDG_DATA_HOME"); xdgDataHome != "" {
		appDataDir = filepath.Join(xdgDataHome, "knirvserver")
	} else if homeDir, homeErr := os.UserHomeDir(); homeErr == nil {
		appDataDir = filepath.Join(homeDir, ".local", "share", "knirvserver")
	} else {
		return "", fmt.Errorf("could not determine application data directory")
	}

	// Ensure directory exists
	if err = os.MkdirAll(appDataDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create application data directory: %w", err)
	}

	return appDataDir, nil
}

// NewServer creates a new KNIRV-SERVER backend server instance
// initOracleFromKeyFile loads the encrypted root.key file and initialises the oracle service.
// Returns nil (no error) when the key file is absent — oracle is simply not started.
// The password is read from ORACLE_KEY_PASSWORD env var (non-interactive/CI) or prompted
// from stdin when running interactively.
func loadSecretsFromKeyFile(logger *zap.Logger) (*pb.RootKeyFileContentProto, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	// Ensure .env files are loaded before checking environment variables.
	// Try multiple search paths to find .env files with different environment names.
	envSearchPaths := []string{
		".env.development",
		".env.testnet",
		".env",
		"../.env.development",
		"../.env.testnet",
		"../.env",
		"../../.env.development",
		"../../.env.testnet",
		"../../.env",
	}
	for _, envPath := range envSearchPaths {
		if err := gotenv.Load(envPath); err == nil {
			logger.Debug("Loaded environment file", zap.String("path", envPath))
			break
		}
	}

	keyPath, err := config.GetRootKeyPath()
	if err != nil {
		return nil, fmt.Errorf("secrets: could not resolve root key path: %w", err)
	}

	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		logger.Info("Secrets not loaded: no root.key found", zap.String("expected_path", keyPath))
		return nil, nil
	}

	var keyPassword []byte
	if envPwd := os.Getenv("ORACLE_KEY_PASSWORD"); envPwd != "" {
		keyPassword = []byte(envPwd)
		logger.Debug("Using ORACLE_KEY_PASSWORD from environment")
	} else {
		// Check if stdin is a terminal
		if !term.IsTerminal(int(os.Stdin.Fd())) {
			logger.Info("Secrets not loaded: ORACLE_KEY_PASSWORD is unset for non-interactive startup")
			return nil, nil
		}
		keyPassword, err = password.PromptForPassword("Enter root key password to load secrets: ")
		if err != nil {
			return nil, fmt.Errorf("secrets: failed to read password: %w", err)
		}
	}

	content, err := password.LoadEncryptedKeyFile(keyPath, keyPassword)
	if err != nil {
		return nil, fmt.Errorf("secrets: failed to decrypt root.key: %w", err)
	}

	logger.Info("Secrets loaded from root.key", zap.String("key_path", keyPath))
	return content, nil
}

func applyRootKeySecretsToConfig(cfg *config.Config, content *pb.RootKeyFileContentProto) {
	if content == nil {
		return
	}

	if content.JwtSecret != "" {
		cfg.Security.JWTSecret = content.JwtSecret
		log.Printf("JWT Secret loaded from root.key")
	}

	if content.KnirvJwtSecret != "" {
		log.Printf("KNIRV JWT Secret loaded from root.key")
	}

	if content.GeminiApiKey != "" {
		log.Printf("Gemini API Key loaded from root.key")
	}

	if content.DeepseekApiKey != "" {
		log.Printf("DeepSeek API Key loaded from root.key")
	}

	if content.CerebrasApiKey != "" {
		log.Printf("Cerebras API Key loaded from root.key")
	}

	if content.TlsCert != "" || content.TlsKey != "" {
		cfg.Security.TLSCert = content.TlsCert
		cfg.Security.TLSKey = content.TlsKey
		log.Printf("TLS certificates loaded from root.key")
	}
}

func initOracleManager(logger *zap.Logger, _ *config.Config) *knirvoracle.Manager {
	if logger == nil {
		logger = zap.NewNop()
	}

	appDataDir, err := getOSAppDataDir()
	if err != nil {
		logger.Error("Failed to get app data dir for oracle", zap.Error(err))
		return nil
	}

	rootKeyPath, err := config.GetRootKeyPath()
	if err != nil {
		logger.Info("Oracle disabled: could not resolve root key path", zap.Error(err))
		return nil
	}

	oracleCfg := &knirvoracle.ManagerConfig{
		BinaryPath:   "knirvoracle",
		SocketPath:   filepath.Join(appDataDir, "sockets", "oracle.sock"),
		DataPath:     filepath.Join(appDataDir, "oracle"),
		RootKeyPath:  rootKeyPath,
		StartTimeout: 30 * time.Second,
		StopTimeout:  10 * time.Second,
	}

	return knirvoracle.NewManager(oracleCfg, logger)
}

func NewServer(cfg *config.Config, rootKeySecrets *pb.RootKeyFileContentProto) (*Server, error) {
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
	dbManager.EnableKNIRVBASE(cfg.Database.UseKNIRVBASE)
	log.Printf("Database initialized (KNIRVBASE bridge: %v)", cfg.Database.UseKNIRVBASE)

	// Initialize eBPF Manager first (required for P2P security integration)
	ebpfManager := ebpf.NewManager()
	ebpfConfig := &ebpf.Config{
		Programs: []ebpf.ProgramConfig{
			{Name: "syscall_trace", Enabled: true},
			{Name: "sandbox_lsm", Enabled: true},
			{Name: "virtual_ns", Enabled: true},
			{Name: "telemetry", Enabled: true},
		},
	}

	// Initialize eBPF manager with context
	ebpfCtx := context.Background()
	if err := ebpfManager.Initialize(ebpfCtx, ebpfConfig); err != nil {
		log.Printf("Warning: Failed to initialize eBPF manager: %v", err)
		log.Printf("eBPF features will be disabled. This may be due to insufficient kernel capabilities.")
		ebpfManager = nil // Disable eBPF if initialization fails
	} else {
		log.Println("eBPF Manager initialized successfully")
	}

	// Initialize XDP Manager and P2P Security Service (eBPF/XDP firewall integration)
	var p2pSecurityService *p2p.P2PService
	if ebpfManager != nil {
		xdpManager := ebpf.NewXDPManager(ebpfManager)
		if err := xdpManager.InitializeXDP(); err != nil {
			log.Printf("Warning: Failed to initialize XDP manager: %v", err)
		} else {
			p2pSecurityService, err = p2p.NewP2PService(xdpManager)
			if err != nil {
				log.Printf("Warning: Failed to initialize P2P security service: %v", err)
				p2pSecurityService = nil
			} else {
				if err := p2pSecurityService.Start(); err != nil {
					log.Printf("Warning: Failed to start P2P security service: %v", err)
					p2pSecurityService = nil
				} else {
					log.Println("P2P Security Service initialized successfully")
				}
			}
		}
	}

	// Initialize P2P manager with security integration
	p2pManager, err := p2p.NewDVEP2PManager(cfg.ChainID, cfg.NodeRole, dbManager, cfg.P2P.DHTEnabled, p2pSecurityService)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize P2P manager: %w", err)
	}

	// Create router
	router := mux.NewRouter()

	var transactionChainClient *transactionchain.Client
	var validationChainClient *validationchain.Client

	// Initialize services
	dveManager, err := dvemanager.NewDVEManager(dbManager, p2pManager, ebpfManager, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize DVE manager: %w", err)
	}

	// Initialize DVE Creation Service
	dveCreationService, err := dvecreation.NewDVECreationService(dbManager)
	if err != nil {
		log.Printf("Warning: Failed to initialize DVE creation service: %v", err)
		dveCreationService = nil
	}

	// Merge dvecreation into dvemanager: wire creation service into the manager
	// so all DVE lifecycle operations are accessible through a single boundary.
	if dveCreationService != nil {
		dveManager.SetCreationService(dveCreationService)
		dveCreationService.SetDveManager(dveManager)
		log.Println("DVE creation service merged into DVE manager")
	}

	// Register the capability query stream handler so peers can interrogate
	// this node's capabilities via the /knirv/dve/capabilities/1.0.0 protocol.
	p2pManager.SetupCapabilityStreamHandler()

	// Ensure fabric server storage path is explicitly set to app data directory if empty
	if cfg.ModelServer.StoragePath == "" {
		appDataDir, err := getOSAppDataDir()
		if err != nil {
			// Log the error but proceed with a fallback that logs a warning.
			logger.Warn("Could not determine app data directory for fabric server storage, falling back to a relative 'models' directory", zap.Error(err))
			cfg.ModelServer.StoragePath = "models"
		} else {
			cfg.ModelServer.StoragePath = filepath.Join(appDataDir, "models")
			logger.Info("Setting fabric server storage path to app data directory", zap.String("path", cfg.ModelServer.StoragePath))
		}
	}

	pluginServer, err := pluginserver.NewPluginServer(cfg, dbManager)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize fabric server: %w", err)
	}

	// Initialize data engine
	dataEngineConfig := data_engine.BuntDBDataEngineConfig{
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
	dataEngine, err := data_engine.NewBuntDBDataEngine(dataEngineConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize data engine: %w", err)
	}

	// Initialize inference service
	inferenceService, err := inference.NewInferenceService(dbManager)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize inference service: %w", err)
	}

	// Initialize TEE Security service with Kali environment detection
	teeSecurityService, err := teesecurity.NewTEESecurityService(dbManager)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize TEE security service: %w", err)
	}

	// Initialize TEE environment with Kali-focused detection
	if err := initializeTEEEnvironment(context.Background(), dbManager); err != nil {
		log.Printf("Warning: TEE environment initialization failed: %v", err)
		// Continue - TEE initialization is not critical for basic operation
	}

	// Initialize Virtual Container Manager with eBPF (using the already initialized ebpfManager)
	var virtualContainerManager *ebpf.VirtualContainerManager
	if ebpfManager != nil {
		virtualContainerManager = ebpf.NewVirtualContainerManager(ebpfManager)
		if err := virtualContainerManager.InitializeVirtualContainers(); err != nil {
			log.Printf("Warning: Failed to initialize virtual container manager: %v", err)
			virtualContainerManager = nil
		} else {
			log.Println("Virtual Container Manager initialized successfully")
		}
	}

	// Initialize System Health service
	systemHealthService := systemhealth.NewSystemHealthService(dbManager)

	// Initialize Fabric Management service
	fabricManagementService := fabricmanagement.NewPluginManagementService(dbManager)
	fabricManagementService.SetPluginServerReference(pluginServer)

	// Initialize Controller Integration service
	controllerIntegrationService := controllerintegration.NewControllerIntegrationService(dbManager)

	// Initialize DNS service with minimal configuration for development
	dnsConfig := dns.DNSConfig{
		CloudFlareAPIToken:  "dev-token", // Placeholder for development
		ZoneName:            "knirv.com",
		UpdateInterval:      time.Minute * 5,
		ForceUpdateInterval: time.Hour,
		Records: []dns.DNSRecordConfig{
			{
				Name:         "server.knirv.com",
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

	// Initialize Session Manager
	sessionManager := session.NewSessionManager(dbManager)

	// Initialize Endpoint Registry
	endpointRegistry := endpoints.NewEndpointRegistry(dbManager)

	// Initialize DVE Rental Service
	dveRentalService, err := dverental.NewDVERentalService(dbManager)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize DVE rental service: %w", err)
	}

	// Initialize Active Memory Layer (Markdown Fabric)
	keyRotationManager := pqc.NewKeyRotationManager(24 * time.Hour)
	pqcManager := pqc.NewEncryptionManager(keyRotationManager)

	// Generate or Load Master Key for PQC
	masterKey, err := pqc.GeneratePQCKeyPair("server-master", "master")
	if err != nil {
		return nil, fmt.Errorf("failed to generate PQC master key: %w", err)
	}
	if err := keyRotationManager.SetMasterKey(masterKey); err != nil {
		return nil, fmt.Errorf("failed to set master key: %w", err)
	}
	pqcManager.CacheKey(masterKey.ID, masterKey)

	// Markdown Storage Driver
	appDataDir, _ := getOSAppDataDir()
	memoryDir := filepath.Join(appDataDir, "active_memory")
	if err := os.MkdirAll(memoryDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create active memory directory: %w", err)
	}
	mdStorage, err := mdstorage.NewMarkdownStorageDriver(memoryDir, pqcManager, masterKey.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize markdown storage: %w", err)
	}

	// Upgrade KNIRVBASE bridge with real storage and PQC
	if dbManager != nil {
		kbManager := database.NewKNIRVBASEManager(cfg.Database.UseKNIRVBASE, mdStorage, pqcManager, memoryDir)
		dbManager.SetKNIRVBASE(kbManager)
		log.Printf("KNIRVBASE persistence layer upgraded with PQC support (dir: %s)", memoryDir)
	}

	// Vault Service (Error/Solution Nodes)
	vaultService := vault.NewVaultService(mdStorage)

	// Reasoning Engine (Graph Traces)
	reasoningEngine := graph.NewReasoningEngine(mdStorage)

	// Active Memory Coordinator
	solutionValidator, err := pqc.NewSolutionNodeValidator(false)
	if err != nil {
		return nil, fmt.Errorf("failed to create solution node validator: %w", err)
	}
	activeMemoryService := active_memory.NewActiveMemoryService(vaultService, reasoningEngine, mdStorage, solutionValidator)

	// Initialize embedded KNIRVGATEWAY for P2P TURN/Tunnel services
	var gatewayManager *knirvgateway.Manager
	if cfg.Gateway.Enabled {
		if cfg.Gateway.Port == 0 {
			cfg.Gateway.Port = 8080
		}

		// Initialize Gateway socket path if not specified
		if cfg.Gateway.SocketPath == "" && cfg.SocketDir != "" {
			cfg.Gateway.SocketPath = filepath.Join(cfg.SocketDir, "gateway.sock")
		}

		gatewayConfig := &knirvgateway.ManagerConfig{
			BinaryPath:     cfg.Gateway.BinaryPath,
			SocketPath:     cfg.Gateway.SocketPath,
			Port:           cfg.Gateway.Port,
			BackendAPIPort: cfg.API.Port,
			Ports: &knirvgateway.PortConfig{
				TurnUDP:     cfg.Gateway.TurnUDPPort,
				TurnTCP:     cfg.Gateway.TurnTCPPort,
				TurnAPI:     cfg.Gateway.TurnAPIPort,
				TunnelHTTP:  cfg.Gateway.TunnelHTTPPort,
				TunnelCtrl:  cfg.Gateway.TunnelCtrlPort,
				TunnelRelay: cfg.Gateway.TunnelRelayPort,
				TunnelSTUN:  cfg.Gateway.TunnelSTUNPort,
			},
			DBPath:       cfg.Database.Path,
			AuthSecret:   cfg.Gateway.AuthSecret,
			MinerAddress: cfg.Gateway.MinerAddress,
			StartTimeout: time.Duration(cfg.Gateway.StartTimeout) * time.Second,
			StopTimeout:  time.Duration(cfg.Gateway.StopTimeout) * time.Second,
			ChainID:      cfg.ChainID,
			Stdout:       logging.NewSubprocessWriter("knirvgateway", os.Stdout),
			Stderr:       logging.NewSubprocessWriter("knirvgateway", os.Stderr),
		}
		gatewayManager = knirvgateway.NewManager(gatewayConfig, logger)
		log.Println("KNIRVGATEWAY manager initialized")

		// Wire the embedded gateway into the P2P manager so that setupNATTraversal
		// can query live TURN server status once the gateway starts.
		p2pManager.SetGatewayTURNQuery(func(ctx context.Context) (*p2p.GatewayTURNStatus, error) {
			if !gatewayManager.IsRunning() {
				return nil, nil
			}
			turnStatus, err := gatewayManager.GetTurnStatus(ctx)
			if err != nil {
				return nil, err
			}
			ports := gatewayManager.GetPorts()
			return &p2p.GatewayTURNStatus{
				Running:      turnStatus.Running,
				UDPPort:      ports.TurnUDP,
				TCPPort:      ports.TurnTCP,
				Realm:        turnStatus.Realm,
				SessionCount: turnStatus.SessionCount,
				ActiveRelays: turnStatus.ActiveRelays,
			}, nil
		})
		log.Println("KNIRVGATEWAY TURN query wired into P2P manager")
	}

	// Initialize embedded KNIRVGRAPH for knowledge graph and graphchain
	var graphManager *knirvgraph.Manager
	var graphSyncManager *knirvgraph.SyncManager
	if cfg.Graph.Enabled {
		graphConfig := &knirvgraph.ManagerConfig{
			BinaryPath:    cfg.Graph.BinaryPath,
			SocketPath:    cfg.Graph.SocketPath,
			P2PSocketPath: cfg.Graph.P2PSocketPath,
			Port:          cfg.Graph.Port,
			P2PPort:       cfg.Graph.P2PPort,
			APIPort:       cfg.Graph.APIPort,
			DataPath:      cfg.Graph.DataPath,
			StartTimeout:  time.Duration(cfg.Graph.StartTimeout) * time.Second,
			StopTimeout:   time.Duration(cfg.Graph.StopTimeout) * time.Second,
			Stdout:        logging.NewSubprocessWriter("knirvgraph", os.Stdout),
			Stderr:        logging.NewSubprocessWriter("knirvgraph", os.Stderr),
		}
		graphManager = knirvgraph.NewManager(graphConfig, logger)
		log.Println("KNIRVGRAPH manager initialized")

		// Initialize SyncManager for staging to embedded sync
		syncInterval, _ := time.ParseDuration(cfg.Graph.SyncInterval)
		if syncInterval == 0 {
			syncInterval = 30 * time.Second
		}
		graphSyncConfig := &knirvgraph.SyncManagerConfig{
			GraphURL: fmt.Sprintf("http://unix/%s", cfg.Graph.SocketPath),
			Interval: syncInterval,
		}
		graphSyncManager = knirvgraph.NewSyncManager(graphSyncConfig, logger)
		log.Println("KNIRVGRAPH sync manager initialized")
	}

	// Initialize embedded KNIRVCHAIN for blockchain and mining
	var chainManager *knirvchain.Manager
	log.Printf("DEBUG: cfg.Chain.Enabled = %v", cfg.Chain.Enabled)
	if cfg.Chain.Enabled {
		chainConfig := &knirvchain.ManagerConfig{
			BinaryPath:    cfg.Chain.BinaryPath,
			SocketPath:    cfg.Chain.SocketPath,
			P2PSocketPath: cfg.Chain.P2PSocketPath,
			Port:          cfg.Chain.Port,
			P2PPort:       cfg.Chain.P2PPort,
			APIPort:       cfg.Chain.APIPort,
			DataPath:      cfg.Chain.DataPath,
			Role:          cfg.Chain.Role,
			ChainID:       cfg.Chain.ChainID,
			StartTimeout:  time.Duration(cfg.Chain.StartTimeout) * time.Second,
			StopTimeout:   time.Duration(cfg.Chain.StopTimeout) * time.Second,
			Stdout:        logging.NewSubprocessWriter("knirvchain", os.Stdout),
			Stderr:        logging.NewSubprocessWriter("knirvchain", os.Stderr),
		}
		chainManager = knirvchain.NewManager(chainConfig, logger)
		log.Println("KNIRVCHAIN manager initialized")
	}

	var transactionChainManager *transactionchain.Manager
	if cfg.TransactionChain.Enabled {
		transactionChainConfig := &transactionchain.ManagerConfig{
			BinaryPath:   cfg.TransactionChain.BinaryPath,
			ScriptPath:   cfg.TransactionChain.ScriptPath,
			WorkDir:      cfg.TransactionChain.WorkDir,
			Port:         cfg.TransactionChain.Port,
			DataPath:     cfg.TransactionChain.DataPath,
			ChainID:      cfg.TransactionChain.ChainID,
			StartTimeout: time.Duration(cfg.TransactionChain.StartTimeout) * time.Second,
			StopTimeout:  time.Duration(cfg.TransactionChain.StopTimeout) * time.Second,
			Stdout:       logging.NewSubprocessWriter("transactionchain", os.Stdout),
			Stderr:       logging.NewSubprocessWriter("transactionchain", os.Stderr),
		}
		transactionChainManager = transactionchain.NewManager(transactionChainConfig, logger)
		log.Println("Transaction chain manager initialized")
	}

	var validationChainManager *validationchain.Manager
	if cfg.ValidationChain.Enabled {
		validationChainConfig := &validationchain.ManagerConfig{
			BinaryPath:   cfg.ValidationChain.BinaryPath,
			WorkDir:      cfg.ValidationChain.WorkDir,
			Port:         cfg.ValidationChain.Port,
			DataPath:     cfg.ValidationChain.DataPath,
			ChainID:      cfg.ValidationChain.ChainID,
			StartTimeout: time.Duration(cfg.ValidationChain.StartTimeout) * time.Second,
			StopTimeout:  time.Duration(cfg.ValidationChain.StopTimeout) * time.Second,
			Stdout:       logging.NewSubprocessWriter("validationchain", os.Stdout),
			Stderr:       logging.NewSubprocessWriter("validationchain", os.Stderr),
		}
		validationChainManager = validationchain.NewManager(validationChainConfig, logger)
		log.Println("Validation chain manager initialized")
	}

	transactionChainURL := cfg.Blockchain.URL
	if transactionChainURL == "" && transactionChainManager != nil {
		transactionChainURL = transactionChainManager.GetBaseURL()
	}

	if transactionChainURL != "" {
		client, err := transactionchain.NewClient(transactionChainURL, cfg.Blockchain.UseTLS, cfg.Blockchain.CertFile)
		if err != nil {
			log.Printf("Warning: Failed to initialize transaction chain client: %v", err)
		} else {
			transactionChainClient = client
			log.Println("Transaction chain client initialized")
		}
	}

	if validationChainManager != nil {
		validationChainClient = validationchain.NewClient(validationChainManager.GetBaseURL())
		log.Println("Validation chain client initialized")
	}

	validationCore, err := validation.NewValidationCore(dbManager, p2pManager, cfg, inferenceService)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize validation core: %w", err)
	}

	// Wire validation-aware service references only after the chain clients and
	// validation core are initialized in dependency order.
	systemHealthService.SetServiceReferences(dveManager, validationCore, inferenceService, teeSecurityService, nil)

	// Initialize Cognitive Engine with configurable parameters
	cognitiveEngine := cognitiveengine.NewCognitiveEngine(dbManager, validationCore, inferenceService, fabricManagementService)
	cognitiveEngine.SetGraphRAGClient(graphRAGClient)
	log.Println("CognitiveEngine: GraphRAG FFI engine wired for knowledge graph retrieval")

	// Wire eBPF manager for real resource telemetry and kernel-level guardrails
	if ebpfManager != nil {
		cognitiveEngine.SetEBPFManager(ebpfManager)
		log.Println("CognitiveEngine: eBPF manager wired for real resource telemetry")
	}

	// Initialize KNIRVSHELL Backend Integration Service
	var knirvshellService *knirvshell.KNIRVSHELLService
	knirvshellService, err = knirvshell.NewKNIRVSHELLService()
	if err != nil {
		log.Printf("Warning: Failed to initialize KNIRVSHELL service: %v", err)
	} else {
		log.Println("KNIRVSHELL service initialized")
	}

	// Initialize Object Nest subsystem
	unifiedContainerManager := runtime.NewUnifiedContainerManager(
		teeSecurityService,
		ebpfManager,
		virtualContainerManager,
	)

	// Wire the outer (server-wide) CognitiveEngine as the inner per-DVE
	// callback receiver so agent task-complexity and resource-usage events
	// flow into the learning and guardrail subsystems.
	unifiedContainerManager.SetCognitiveEngine(cognitiveEngine)
	log.Println("UCM: inner cognitive engine wired to outer CognitiveEngine")

	// Wire P2P router callbacks so newly started containers are announced
	// to the DVE announcement topic and removed on teardown.
	unifiedContainerManager.SetRouterFuncs(
		func(ctx context.Context, containerID, cryptoHash, objectType string) error {
			return p2pManager.PublishContainerRegistration(ctx, containerID, cryptoHash, objectType)
		},
		func(ctx context.Context, containerID string) error {
			return p2pManager.PublishContainerDeregistration(ctx, containerID)
		},
	)
	log.Println("UCM: P2P router registration callbacks wired")

	// Initialize Agent Service (oh-my-pi agentic runtime)
	agentService := agentsvc.NewAgentService(dbManager, unifiedContainerManager, activeMemoryService)
	if err := agentService.Start(); err != nil {
		log.Printf("Warning: Failed to start agent service: %v", err)
	}

	// Initialize GraphRAG Knowledge Base Engine
	graphRAGClient := knowledge_base.NewGraphRAGClient()
	log.Println("GraphRAG engine initialized with CGO FFI bridge")

	// Initialize AnchoringService for PQC-signed evidence pack anchoring
	anchoringService := evidence.NewAnchoringService(dbManager, pqcManager, "server-master")
	if validationChainClient != nil {
		anchoringService.SetValidationChainClient(validationChainClient)
		log.Println("AnchoringService: validation chain client wired for chain anchoring")
	} else {
		log.Println("AnchoringService: validation chain client unavailable, chain anchoring will remain disabled")
	}
	if err := anchoringService.LoadEvidencePacks(); err != nil {
		log.Printf("Warning: Failed to load evidence packs: %v", err)
	}
	log.Println("AnchoringService initialized")

	// Initialize ICME - Intentional Context Memory Engine
	var icmeService *icme.Service
	if cfg.ICME.Enabled {
		icmeConfig := cfg.ICME

		embeddingProvider, err := icme.NewEmbeddingProvider(logger)
		if err != nil {
			log.Printf("Warning: Failed to initialize ICME embedding provider: %v", err)
		} else {
			nerProvider, err := icme.NewNERProvider(logger)
			if err != nil {
				log.Printf("Warning: Failed to initialize ICME NER provider: %v", err)
			} else {
				faissManager, err := icme.NewFAISSIndexManager(dbManager, logger)
				if err != nil {
					log.Printf("Warning: Failed to initialize ICME FAISS manager: %v", err)
				} else {
					graphEngine := icme.NewTemporalHypergraph(
						icmeConfig.GraphWindowSize,
						icmeConfig.GraphMaxNodes,
						logger,
					)

					// Wire KNIRVGRAPH into the Cognitive Engine for ontological data organization
					cognitiveEngine.SetKNIRVGRAPHEngine(graphEngine, logger)
					log.Println("CognitiveEngine: KNIRVGRAPH temporal hypergraph wired for ontology indexing")

					intentRegistry, err := icme.NewIntentRegistry(dbManager, logger)
					if err != nil {
						log.Printf("Warning: Failed to initialize ICME intent registry: %v", err)
					} else {
						factualityAdapter := icme.NewFactualityAdapter(
							"",
							intentRegistry,
							dbManager,
							logger,
						)

						delegation := icme.NewDelegationFramework(intentRegistry, logger)

						alignmentLoop := icme.NewAlignmentLoop(
							intentRegistry,
							factualityAdapter,
							icmeConfig.AlignmentEvalInterval,
							icmeConfig.DriftThreshold,
							logger,
						)

						signalRouter := icme.NewSignalRouter(
							intentRegistry,
							nerProvider,
							embeddingProvider,
							graphEngine,
							faissManager,
							nil,
							logger,
						)

						searchEngine := icme.NewHybridSearchEngine(
							faissManager,
							graphEngine,
							embeddingProvider,
							intentRegistry,
							logger,
						)

						icmeService = icme.NewService(
							intentRegistry,
							graphEngine,
							faissManager,
							searchEngine,
							delegation,
							alignmentLoop,
							signalRouter,
							logger,
							dbManager,
						)
						if validationChainClient != nil {
							icmeService.SetValidationChainClient(validationChainClient)
							log.Println("ICME: validation chain client wired during initialization")
						}

						icmeCtx, icmeCancel := context.WithCancel(context.Background())
						go signalRouter.Start(icmeCtx)
						go alignmentLoop.Start(icmeCtx)
						_ = icmeCancel

						log.Println("ICME Service initialized successfully")
					}
				}
			}
		}
	} else {
		log.Println("ICME Service disabled via configuration")
	}

	if transactionChainClient != nil {
		if dveCreationService != nil {
			dveCreationService.SetTransactionChainClient(transactionChainClient)
			log.Println("Transaction chain client integrated with DVE creation service")
		}
		if dveRentalService != nil {
			dveRentalService.SetTransactionChainClient(transactionChainClient)
			log.Println("Transaction chain client integrated with DVE rental service")
		}
	}

	// Initialize Nexus Memory Fabric
	nexusAllocator := memory.DefaultAllocator
	nexusServer := nexus.NewNexusMemoryServer(nexusAllocator)
	if activeMemoryService != nil {
		nexusServer.SetMemoryProvider(activeMemoryService)
	}
	if err := nexusServer.StartGuardian(); err != nil {
		log.Printf("Warning: Failed to start Nexus Guardian (eBPF): %v", err)
	}

	// ============================================
	// Production Services (Phase 3, 4, 8)
	// ============================================

	// Initialize EventBroadcaster for comprehensive WebSocket event streaming
	eventBroadcaster := websocket.NewEventBroadcaster(nil)
	eventBroadcaster.Start()
	log.Println("EventBroadcaster initialized")

	// Initialize SecretManager for session-based secret management
	secretManager := secrets.NewSecretManager(dbManager)
	secretManager.SetEncryptionKey([]byte(cfg.Security.JWTSecret))
	if err := secretManager.LoadSecrets(); err != nil {
		log.Printf("Warning: Failed to load secrets: %v", err)
	}
	if err := secretManager.LoadSessions(); err != nil {
		log.Printf("Warning: Failed to load secret sessions: %v", err)
	}
	log.Println("SecretManager initialized")

	// Initialize WorkflowService for real workflow execution orchestration
	workflowService := workflow.NewWorkflowService(dbManager)
	workflowService.SetDVEManager(dveManager)
	workflowService.SetValidationCore(validationCore)
	dveTaskExecutor := workflow.NewDVETaskExecutor(dveManager, validationCore)
	if validationChainClient != nil {
		dveTaskExecutor.SetValidationChainClient(validationChainClient)
	}
	workflowService.RegisterExecutor("validation", dveTaskExecutor)
	workflowService.SetEventBroadcaster(eventBroadcaster)
	if err := workflowService.LoadExecutionsFromDB(); err != nil {
		log.Printf("Warning: Failed to load workflow executions: %v", err)
	}
	log.Println("WorkflowService initialized")

	// Initialize GuardrailManager for dynamic guardrails and policy enforcement
	guardrailManager := guardrails.NewDynamicGuardrailManager(dbManager)
	guardrailManager.SetEventBroadcaster(eventBroadcaster)
	if err := guardrailManager.LoadConfigurations(); err != nil {
		log.Printf("Warning: Failed to load guardrail configurations: %v", err)
	}
	if err := guardrailManager.LoadOntologyRules(); err != nil {
		log.Printf("Warning: Failed to load ontology rules: %v", err)
	}

	// Initialize Onboarding Service - Value System and Ontology Ingestion
	onboardingService := onboarding.NewOnboardingService(dbManager, cognitiveEngine, guardrailManager)
	log.Println("Onboarding service initialized")

	// Initialize PolicyEngine for ICME policy integration
	policyEngine := guardrails.NewPolicyEngine(dbManager, guardrailManager)
	if icmeService != nil {
		icmeAdapter := &icmePolicyAdapter{icme: icmeService}
		policyEngine.SetICMEService(icmeAdapter)
		log.Println("PolicyEngine: ICME service wired for intent objective management")
	}
	if err := policyEngine.LoadPolicies(); err != nil {
		log.Printf("Warning: Failed to load policies: %v", err)
	}
	log.Println("GuardrailManager and PolicyEngine initialized")

	// Initialize Hasher gRPC Server for KNIRVHASHER integration
	hasherSocketPath := os.Getenv("HASHER_SOCKET_PATH")
	if hasherSocketPath == "" {
		hasherSocketPath = dvemanager.DefaultSocketPath
	}
	hasherGRPCServer := dvemanager.NewHasherGRPCServer(hasherSocketPath, dbManager)
	if err := hasherGRPCServer.Start(); err != nil {
		log.Printf("Warning: Failed to start Hasher gRPC server: %v (hasher integration disabled)", err)
	} else {
		log.Printf("Hasher gRPC server started on %s", hasherSocketPath)
	}

	// Initialize Hasher Client for connecting to external hasher service
	hasherIntegration := dvemanager.NewHasherIntegration(hasherSocketPath, dbManager, guardrailManager)
	if err := hasherIntegration.Connect(context.Background()); err != nil {
		log.Printf("Warning: Failed to connect to hasher service: %v (hasher validation disabled)", err)
	} else {
		log.Printf("Hasher client connected to %s", hasherSocketPath)
	}

	// Wire hasher validator into guardrail manager for security validation
	guardrailManager.SetHasherValidator(hasherIntegration)
	log.Println("Hasher integration wired into GuardrailManager")

	// Wire EventBroadcaster to WebSocket service (will be set in setupRoutes)
	_ = eventBroadcaster

	// Create context for service lifecycle management
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize SocketPath if SocketDir is present but SocketPath is empty
	if cfg.API.SocketPath == "" && cfg.SocketDir != "" {
		cfg.API.SocketPath = filepath.Join(cfg.SocketDir, "backend.sock")
		log.Printf("Using default API socket path: %s", cfg.API.SocketPath)
	}

	server := &Server{
		config:                       cfg,
		db:                           dbManager,
		router:                       router,
		p2pManager:                   p2pManager,
		logger:                       logger,
		ebpfManager:                  ebpfManager,
		virtualContainerManager:      virtualContainerManager,
		nexusServer:                  nexusServer,
		nexusAllocator:               nexusAllocator,
		dveManager:                   dveManager,
		dveCreationService:           dveCreationService,
		dveRentalService:             dveRentalService,
		validationCore:               validationCore,
		dnsService:                   dnsService,
		pluginServer:                 pluginServer,
		dataEngine:                   dataEngine,
		inferenceService:             inferenceService,
		websocketService:             nil, // Will be set in setupRoutes
		teeSecurityService:           teeSecurityService,
		systemHealthService:          systemHealthService,
		fabricManagementService:      fabricManagementService,
		controllerIntegrationService: controllerIntegrationService,
		cognitiveEngine:              cognitiveEngine,
		containerOrchestrator:        containerOrchestrator,
		sessionManager:               sessionManager,
		endpointRegistry:             endpointRegistry,
		gatewayManager:               gatewayManager,
		graphManager:                 graphManager,
		graphSyncManager:             graphSyncManager,
		chainManager:                 chainManager,
		transactionChainManager:      transactionChainManager,
		validationChainManager:       validationChainManager,
		pqcManager:                   pqcManager,
		mdStorage:                    mdStorage,
		vaultService:                 vaultService,
		reasoningEngine:              reasoningEngine,
		activeMemoryService:          activeMemoryService,
		unifiedContainerManager:      unifiedContainerManager,
		agentService:                 agentService,
		icmeService:                  icmeService,
		eventBroadcaster:             eventBroadcaster,
		anchoringService:             anchoringService,
		secretManager:                secretManager,
		workflowService:              workflowService,
		rollupService:                nil,
		guardrailManager:             guardrailManager,
		policyEngine:                 policyEngine,
		knirvshellService:            knirvshellService,
		onboardingService:            onboardingService,
		hasherGRPCServer:             hasherGRPCServer,
		hasherIntegration:            hasherIntegration,
		transactionChainClient:       transactionChainClient,
		validationChainClient:        validationChainClient,
		graphRAGClient:               graphRAGClient,
		ctx:                          ctx,
		cancel:                       cancel,
		running:                      false,
	}

	// Initialise oracle manager if root.key is present (root node only)
	oracleManager := initOracleManager(logger, cfg)
	if oracleManager != nil {
		if err := oracleManager.Start(ctx); err != nil {
			logger.Error("Failed to start oracle manager — continuing without oracle", zap.Error(err))
		}
	}
	server.oracleManager = oracleManager
	if oracleManager != nil && oracleManager.IsRunning() {
		balanceReader := &oracleBalanceAdapter{manager: oracleManager}
		if transactionChainClient != nil {
			transactionChainClient.SetBalanceReader(balanceReader)
		}
	}
	if cfg.Rollup.PollInterval <= 0 {
		cfg.Rollup.PollInterval = 30 * time.Second
	}

	if cfg.Rollup.Enabled && transactionChainManager != nil && oracleManager.IsRunning() {
		reader := rollup.NewHTTPTransactionChainReader(transactionChainManager.GetBaseURL())
		server.rollupService = rollup.NewService(reader, &oracleRollupAdapter{manager: oracleManager})
		appDataDir, appDataErr := getOSAppDataDir()
		if appDataErr != nil {
			return nil, fmt.Errorf("failed to determine app data dir for rollup persistence: %w", appDataErr)
		}
		if err := server.rollupService.SetPersistencePath(filepath.Join(appDataDir, "rollups", "transaction_rollups.json")); err != nil {
			return nil, fmt.Errorf("failed to initialize rollup persistence: %w", err)
		}
		server.rollupPollInterval = cfg.Rollup.PollInterval
		log.Println("Rollup service initialized")
	}

	// Setup routes for all services
	server.setupRoutes()

	return server, nil
}

type oracleRollupAdapter struct {
	manager *knirvoracle.Manager
}

type oracleBalanceAdapter struct {
	manager *knirvoracle.Manager
}

func (a *oracleBalanceAdapter) GetAccountBalance(address string) (int64, error) {
	if a == nil || a.manager == nil || !a.manager.IsRunning() {
		return 0, fmt.Errorf("oracle not available")
	}

	client := a.manager.GetClient()
	balanceStr, err := client.GetBalance(address)
	if err != nil {
		return 0, fmt.Errorf("failed to get balance: %w", err)
	}

	var balance int64
	if _, err := fmt.Sscanf(balanceStr, "%d", &balance); err != nil {
		return 0, fmt.Errorf("invalid balance format: %w", err)
	}

	return balance, nil
}

func (a *oracleRollupAdapter) SubmitRollup(batch *rollup.RollupBatch) (string, error) {
	record := &knirvoracle.RollupRecord{
		ID:          batch.ID,
		BatchRoot:   batch.BatchRoot,
		ChainID:     batch.ChainID,
		StartHeight: batch.StartHeight,
		EndHeight:   batch.EndHeight,
		BlockCount:  len(batch.Blocks),
		TxCount:     batch.Settlement.TxCount,
		Status:      knirvoracle.RollupStatusSubmitted,
		SubmittedAt: time.Now().UTC(),
		Metadata: map[string]interface{}{
			"batch_root": batch.BatchRoot,
		},
	}
	if a.manager == nil || !a.manager.IsRunning() {
		return "", fmt.Errorf("oracle not available")
	}
	client := a.manager.GetClient()
	rollupID, err := client.SubmitRollup(record)
	if err != nil {
		return "", err
	}
	return rollupID, nil
}

func (a *oracleRollupAdapter) GetRollup(id string) (map[string]interface{}, error) {
	if a.manager == nil || !a.manager.IsRunning() {
		return nil, fmt.Errorf("oracle not available")
	}
	client := a.manager.GetClient()
	return client.GetRollup(id)
}

func (a *oracleRollupAdapter) FinalizeRollup(id string) error {
	return fmt.Errorf("finalize not implemented via client")
}

func (a *oracleRollupAdapter) DisputeRollup(id string, reason string) error {
	return fmt.Errorf("dispute not implemented via client")
}

type icmePolicyAdapter struct {
	icme *icme.Service
}

func (a *icmePolicyAdapter) RegisterObjective(obj *guardrails.IntentObjective) error {
	icmeObj := &icme.IntentObjective{
		Name:           obj.Name,
		Scope:          icme.IntentScope(obj.Scope),
		HardBoundaries: []string{},
	}
	for k, v := range obj.HardBoundaries {
		icmeObj.HardBoundaries = append(icmeObj.HardBoundaries, fmt.Sprintf("%s=%.2f", k, v))
	}
	for k, v := range obj.SoftLimits {
		icmeObj.HardBoundaries = append(icmeObj.HardBoundaries, fmt.Sprintf("%s=%.2f(soft)", k, v))
	}
	for k, v := range obj.TradeOffs {
		icmeObj.TradeOffs[k] = v
	}
	return a.icme.RegisterObjective(icmeObj)
}

func (a *icmePolicyAdapter) GetObjectiveForAgent(agentID, dveID string) *guardrails.IntentObjective {
	icmeObj := a.icme.GetObjectiveForAgent(agentID, dveID)
	if icmeObj == nil {
		return nil
	}
	obj := &guardrails.IntentObjective{
		Name:    icmeObj.Name,
		Scope:   string(icmeObj.Scope),
		Version: icmeObj.Version,
	}
	for _, hb := range icmeObj.HardBoundaries {
		if parts := strings.SplitN(hb, "=", 2); len(parts) == 2 {
			var val float64
			fmt.Sscanf(parts[1], "%f", &val)
			if strings.HasSuffix(parts[1], "(soft)") {
				if obj.SoftLimits == nil {
					obj.SoftLimits = make(map[string]float64)
				}
				obj.SoftLimits[strings.TrimSuffix(parts[0], "(soft)")] = val
			} else {
				if obj.HardBoundaries == nil {
					obj.HardBoundaries = make(map[string]float64)
				}
				obj.HardBoundaries[parts[0]] = val
			}
		}
	}
	for k, v := range icmeObj.TradeOffs {
		if obj.TradeOffs == nil {
			obj.TradeOffs = make(map[string]float64)
		}
		obj.TradeOffs[k] = v
	}
	return obj
}

// setupRoutes configures all HTTP routes for the unified server
func (s *Server) setupRoutes() {
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
	// Wire real Cognitive Engine into WebSocket for live metric broadcasts (Gap 8 + 12)
	if s.cognitiveEngine != nil {
		wsService.SetCognitiveEngine(s.cognitiveEngine)
	}
	s.websocketService = wsService

	// Register log streaming routes
	logStreamHandler := websocket.GetLogStreamHandler()
	logStreamHandler.RegisterRoutes(s.router)
	log.Println("Log stream handler routes configured")

	// Wire module logging to SSE handler
	logging.SetLogStreamHandler(logStreamHandler)

	// Wire EventBroadcaster into WebSocket for comprehensive event streaming
	if s.eventBroadcaster != nil {
		s.eventBroadcaster = websocket.NewEventBroadcaster(wsService)
		s.eventBroadcaster.Start()
		log.Println("EventBroadcaster wired into WebSocket service")
	}

	// Wire Controller Integration service with WebSocket service
	if s.controllerIntegrationService != nil {
		s.controllerIntegrationService.SetWebSocketService(wsService)
	}

	// Wire GuardrailManager with EventBroadcaster
	if s.guardrailManager != nil {
		s.guardrailManager.SetEventBroadcaster(s.eventBroadcaster)
	}

	// Wire WorkflowService with EventBroadcaster
	if s.workflowService != nil {
		s.workflowService.SetEventBroadcaster(s.eventBroadcaster)
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
		protectedAuthRouter.HandleFunc("/preferences", authHandlers.GetPreferences).Methods("GET", "OPTIONS")
		protectedAuthRouter.HandleFunc("/preferences", authHandlers.UpdatePreferences).Methods("PUT", "OPTIONS")
		log.Println("Auth routes configured")
	}

	// Register KNIRVGRAPH routes
	knirvGraphHandlers := web.NewKnirvGraphHandlers(s.db, s.graphSyncManager)
	s.router.HandleFunc("/api/knirvgraph/error-node", knirvGraphHandlers.CreateErrorNode).Methods("POST", "OPTIONS")
	s.router.HandleFunc("/api/knirvgraph/error-nodes", knirvGraphHandlers.GetErrorNodes).Methods("GET", "OPTIONS")
	s.router.HandleFunc("/api/knirvgraph/error-queue", knirvGraphHandlers.GetErrorQueue).Methods("GET", "OPTIONS")
	log.Println("KNIRVGRAPH routes configured")

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

	// Register fabric server routes
	if s.pluginServer != nil {
		s.pluginServer.RegisterRoutes(s.router)
	}

	// Register Nexus Memory Fabric routes
	if s.nexusServer != nil {
		s.nexusServer.RegisterRoutes(s.router)
	}

	// Register DNS service routes (when available)
	if s.dnsService != nil {
		s.dnsService.RegisterRoutes(s.router, authMiddleware)
	}

	// Register inference service routes
	if s.inferenceService != nil {
		inferenceHandlers := web.NewInferenceHandlers(s.inferenceService)
		// Wire real Cognitive Engine into inference endpoint (Gap 12)
		if s.cognitiveEngine != nil {
			inferenceHandlers.SetCognitiveEngine(s.cognitiveEngine)
		}
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
		dveHandlers := web.NewDVEHandlers(s.dveManager)
		if s.sessionManager != nil {
			dveHandlers.SetSessionManager(s.sessionManager)
		}
		dveHandlers.RegisterRoutes(s.router, authMiddleware)
		log.Println("DVE manager routes configured")
	}

	// Register DVE creation routes
	if s.dveCreationService != nil {
		dveCreationHandlers := web.NewDVECreationHandlers(s.dveManager, s.containerOrchestrator, s.sessionManager, s.endpointRegistry, s.db)
		dveCreationHandlers.RegisterRoutes(s.router, authMiddleware)
		log.Println("DVE creation routes configured")
	}

	// DVE Rental has been removed/deprecated in exchange for direct DVE Creation

	// Register workflow execution routes with real WorkflowService (Phase 4)
	if s.workflowService != nil {
		workflowHandlers := web.NewWorkflowHandlers(s.dveManager, s.workflowService)
		workflowHandlers.RegisterRoutes(s.router)
		log.Println("Workflow execution routes configured with WorkflowService")
	}

	// Register guardrail and policy routes
	if s.guardrailManager != nil {
		guardrailHandlers := web.NewGuardrailHandlers(s.guardrailManager, s.policyEngine)
		if s.eventBroadcaster != nil {
			guardrailHandlers.SetEventBroadcaster(s.eventBroadcaster)
		}
		guardrailHandlers.RegisterRoutes(s.router)
		log.Println("Guardrail and Policy routes configured")
	}

	// Register secret manager routes
	if s.secretManager != nil {
		secretHandlers := web.NewSecretHandlers(s.secretManager)
		secretHandlers.RegisterRoutes(s.router)
		log.Println("Secret manager routes configured")
	}

	// Register anchoring service routes
	if s.anchoringService != nil {
		anchoringHandlers := web.NewAnchoringHandlers(s.anchoringService)
		anchoringHandlers.RegisterRoutes(s.router)
		log.Println("Anchoring service routes configured")
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

	// Register system info API routes (for HUD/PWA)
	systemInfoCollector, err := host.NewSystemInfoCollector(context.Background(), 2*time.Second)
	if err != nil {
		log.Printf("Warning: failed to create system info collector: %v", err)
	} else {
		systemHandler := api.NewSystemHandler(systemInfoCollector)
		systemHandler.RegisterRoutes(s.router)
		log.Println("System info API routes configured")
	}

	// Register fabric management service routes
	if s.fabricManagementService != nil {
		fabricManagementHandlers := web.NewPluginManagementHandlers(s.fabricManagementService)
		fabricManagementHandlers.RegisterRoutes(s.router, authMiddleware)
		log.Println("Fabric management service routes configured")
	}

	// Register controller integration service routes
	if s.controllerIntegrationService != nil {
		controllerIntegrationHandlers := web.NewControllerIntegrationHandlers(s.controllerIntegrationService)
		controllerIntegrationHandlers.RegisterRoutes(s.router, authMiddleware)
		log.Println("Controller integration service routes configured")
	}

	// Register cognitive engine routes
	if s.cognitiveEngine != nil {
		cognitiveEngineHandlers := web.NewCognitiveEngineHandlers(s.cognitiveEngine)
		cognitiveEngineHandlers.RegisterRoutes(s.router, authMiddleware)
		log.Println("Cognitive engine routes configured")

		telemetryHandlers := web.NewTelemetryHandlers(&telemetryServiceAdapter{engine: s.cognitiveEngine})
		telemetryHandlers.RegisterRoutes(s.router)
		log.Println("Cognitive telemetry routes configured")

		ontologyHandlers := web.NewOntologyHandlers(&ontologyServiceAdapter{engine: s.cognitiveEngine})
		ontologyHandlers.RegisterRoutes(s.router)
		log.Println("Cognitive ontology routes configured")

		analyticsHandlers := web.NewAnalyticsHandlers(cognitiveengine.NewPredictiveAnalytics(100))
		analyticsHandlers.RegisterRoutes(s.router)
		log.Println("Predictive analytics routes configured")
	}

	// Register Active Memory handlers
	if s.activeMemoryService != nil {
		memoryHandlers := web.NewActiveMemoryHandlers(s.activeMemoryService)
		memoryHandlers.RegisterRoutes(s.router, authMiddleware)
		log.Println("Active Memory (Markdown Fabric) routes configured")
	}

	// Register NRN payment routes
	if s.transactionChainClient != nil {
		nrnHandlers := web.NewNRNPaymentHandlers(s.transactionChainClient)
		nrnHandlers.RegisterRoutes(s.router, authMiddleware)
		log.Println("NRN payment routes configured")
	}

	// Register Agent Command Center routes (oh-my-pi)
	if s.agentService != nil {
		agentHandlers := web.NewAgentHandlers(s.agentService)
		agentHandlers.RegisterRoutes(s.router, authMiddleware)
		log.Println("Agent Command Center routes configured")
	}

	// Register ICME routes (Intentional Context Memory Engine)
	if s.icmeService != nil {
		s.icmeService.RegisterRoutes(s.router)
		log.Println("ICME routes configured")
	}

	// Register KNIRVSHELL routes
	if s.knirvshellService != nil {
		web.NewKNIRVSHELLHandlers(s.knirvshellService).RegisterRoutes(s.router, authMiddleware)
		log.Println("KNIRVSHELL routes configured")
	}

	// Register Onboarding routes
	if s.onboardingService != nil {
		web.NewOnboardingHandlers(s.onboardingService).RegisterRoutes(s.router, authMiddleware)
		log.Println("Onboarding routes configured")
	}

	// Register unified API router for path unification (Gap 6)
	apiRouter := web.NewAPIRouter(
		web.NewDVEHandlers(s.dveManager),
		web.NewPluginManagementHandlers(s.fabricManagementService),
		web.NewAgentHandlers(s.agentService),
		web.NewPaymentHandlers(nil, nil, s.eventBroadcaster),
		web.NewKNIRVSHELLHandlers(s.knirvshellService),
		web.NewOnboardingHandlers(s.onboardingService),
		web.NewCognitiveEngineHandlers(s.cognitiveEngine),
		nil, // knowledgeBaseHandlers (to be implemented)
		authMiddleware,
	)
	apiRouter.RegisterRoutes(s.router)
	log.Println("Unified API router configured")

	// Register system settings routes
	systemSettingsHandlers := web.NewSystemSettingsHandlers(s.config)
	systemSettingsHandlers.RegisterRoutes(s.router, authMiddleware)
	log.Println("System settings routes configured")

	// Register oracle routes (root node only — only wired when oracle is active)
	// NOTE: Oracle routes are now served via external knirvoracle binary
	// Requests to /oracle/ are proxied to the Unix socket (handled separately)
	if s.oracleManager != nil && s.oracleManager.IsRunning() {
		log.Println("Oracle manager running (routes served via knirvoracle binary)")
	}

	if s.rollupService != nil {
		rollupHandlers := web.NewRollupHandlers(s.rollupService, s.rollupPollInterval)
		rollupHandlers.RegisterRoutes(s.router, authMiddleware)
		log.Println("Rollup routes configured")
	}

	log.Println("All routes configured successfully")
}

func (s *Server) runRollupLoop() {
	if s.rollupService == nil {
		return
	}

	pollInterval := s.rollupPollInterval
	if pollInterval <= 0 {
		pollInterval = 30 * time.Second
	}

	ticker := time.NewTicker(pollInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-s.ctx.Done():
				return
			case <-ticker.C:
				batch, err := s.rollupService.BuildNextBatch(s.ctx)
				if err != nil {
					log.Printf("Warning: Failed to build rollup batch: %v", err)
					continue
				}
				if batch == nil {
				} else {
					submitted, err := s.rollupService.SubmitBatch(s.ctx, batch.ID)
					if err != nil {
						log.Printf("Warning: Failed to submit rollup batch %s: %v", batch.ID, err)
						continue
					}

					log.Printf("Rollup batch submitted: %s (heights %d-%d)", submitted.ID, submitted.StartHeight, submitted.EndHeight)
				}

				reconciled, err := s.rollupService.ReconcileWithOracle(s.ctx)
				if err != nil {
					log.Printf("Warning: Failed to reconcile rollup batches with oracle: %v", err)
					continue
				}
				if reconciled > 0 {
					log.Printf("Reconciled %d rollup batch statuses from oracle", reconciled)
				}
			}
		}
	}()
}

// handleHealth handles the /health endpoint
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get eBPF metrics if available
	var ebpfMetrics map[string]any
	if s.ebpfManager != nil {
		// Check if the concrete value is not nil
		if m, ok := s.ebpfManager.(*ebpf.Manager); ok && m != nil {
			metrics := s.ebpfManager.GetMetrics()
			ebpfMetrics = map[string]any{
				"enabled":           true,
				"initialized":       metrics.Initialized,
				"programs_attached": metrics.ProgramsAttached,
			}
		} else {
			ebpfMetrics = map[string]any{
				"enabled": false,
				"reason":  "eBPF manager not properly initialized",
			}
		}
	} else {
		ebpfMetrics = map[string]any{
			"enabled": false,
			"reason":  "eBPF initialization failed or not supported",
		}
	}

	knirvbaseStatus, _ := s.db.GetKNIRVBASEStatus()

	response := map[string]any{
		"status":     "healthy",
		"version":    Version,
		"build_time": BuildTime,
		"git_commit": GitCommit,
		"services": map[string]bool{
			"database":              s.db != nil,
			"knirvbase":             knirvbaseStatus == "healthy",
			"p2p_manager":           s.p2pManager != nil,
			"ebpf_manager":          s.ebpfManager != nil,
			"virtual_container_mgr": s.virtualContainerManager != nil,
			"dve_manager":           s.dveManager != nil,
			"validation_core":       s.validationCore != nil,
			"fabric_server":         s.pluginServer != nil,
			"data_engine":           s.dataEngine != nil,
			"inference_service":     s.inferenceService != nil,
			"websocket_service":     s.websocketService != nil,
			"dns_service":           s.dnsService != nil,
			"gateway_manager":       s.gatewayManager != nil,
			"active_memory_service": s.activeMemoryService != nil,
			"pqc_manager":           s.pqcManager != nil,
			"dve_creation_service":  s.dveCreationService != nil,
			"cde_service":           s.cognitiveEngine != nil,
			"model_server":          s.unifiedContainerManager != nil,
			"agent_service":         s.agentService != nil,
			"icme_service":          s.icmeService != nil,
			"event_broadcaster":     s.eventBroadcaster != nil,
			"anchoring_service":     s.anchoringService != nil,
			"secret_manager":        s.secretManager != nil,
			"workflow_service":      s.workflowService != nil,
			"guardrail_manager":     s.guardrailManager != nil,
			"policy_engine":         s.policyEngine != nil,
		},
		"ebpf": ebpfMetrics,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// Start starts all services and the HTTP server
func (s *Server) Start() error {
	if s.running {
		return fmt.Errorf("server is already running")
	}

	log.Println("Starting KNIRV-SERVER unified server...")

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

	// Start embedded KNIRVGRAPH
	if s.graphManager != nil && s.config.Graph.Enabled {
		if err := s.graphManager.Start(); err != nil {
			log.Printf("Warning: Failed to start KNIRVGRAPH: %v", err)
			logging.EmitModuleLog("knirvgraph", "error", fmt.Sprintf("Failed to start: %v", err))
		} else {
			if socketPath := s.graphManager.GetConfig().SocketPath; socketPath != "" {
				log.Printf("KNIRVGRAPH started on socket %s", socketPath)
				logging.EmitModuleLog("knirvgraph", "info", fmt.Sprintf("Started on socket %s", socketPath))
			} else {
				log.Printf("KNIRVGRAPH started on port %d", s.graphManager.GetConfig().Port)
				logging.EmitModuleLog("knirvgraph", "info", fmt.Sprintf("Started on port %d", s.graphManager.GetConfig().Port))
			}
		}
	}

	// Start embedded KNIRVCHAIN
	if s.chainManager != nil && s.config.Chain.Enabled {
		if err := s.chainManager.Start(s.ctx); err != nil {
			log.Printf("Warning: Failed to start KNIRVCHAIN: %v", err)
			logging.EmitModuleLog("knirvchain", "error", fmt.Sprintf("Failed to start: %v", err))
		} else {
			if socketPath := s.chainManager.GetConfig().SocketPath; socketPath != "" {
				log.Printf("KNIRVCHAIN started on socket %s", socketPath)
				logging.EmitModuleLog("knirvchain", "info", fmt.Sprintf("Started on socket %s", socketPath))
			} else {
				log.Printf("KNIRVCHAIN started on port %d", s.chainManager.GetConfig().APIPort)
				logging.EmitModuleLog("knirvchain", "info", fmt.Sprintf("Started on port %d", s.chainManager.GetConfig().APIPort))
			}
		}
	}
	if s.transactionChainManager != nil && s.config.TransactionChain.Enabled {
		if err := s.transactionChainManager.Start(s.ctx); err != nil {
			log.Printf("Warning: Failed to start transaction chain: %v", err)
			logging.EmitModuleLog("transactionchain", "error", fmt.Sprintf("Failed to start: %v", err))
		} else {
			log.Printf("Transaction chain started on port %d", s.transactionChainManager.GetConfig().Port)
			logging.EmitModuleLog("transactionchain", "info", fmt.Sprintf("Started on port %d", s.transactionChainManager.GetConfig().Port))
		}
	}
	if s.validationChainManager != nil && s.config.ValidationChain.Enabled {
		if err := s.validationChainManager.Start(s.ctx); err != nil {
			log.Printf("Warning: Failed to start validation chain: %v", err)
			logging.EmitModuleLog("validationchain", "error", fmt.Sprintf("Failed to start: %v", err))
		} else {
			log.Printf("Validation chain started on port %d", s.validationChainManager.GetConfig().Port)
			logging.EmitModuleLog("validationchain", "info", fmt.Sprintf("Started on port %d", s.validationChainManager.GetConfig().Port))
		}
	}

	// Start embedded KNIRVGATEWAY for P2P TURN/Tunnel before DVE services rely on it.
	if s.gatewayManager != nil {
		if err := s.gatewayManager.Start(s.ctx); err != nil {
			log.Printf("Warning: Failed to start KNIRVGATEWAY: %v", err)
			logging.EmitModuleLog("knirvgateway", "error", fmt.Sprintf("Failed to start: %v", err))
		} else {
			if socketPath := s.gatewayManager.GetConfig().SocketPath; socketPath != "" {
				log.Printf("KNIRVGATEWAY started on socket %s (public port identity %d)", socketPath, s.gatewayManager.GetConfig().Port)
				logging.EmitModuleLog("knirvgateway", "info", fmt.Sprintf("Started on socket %s", socketPath))
			} else {
				log.Printf("KNIRVGATEWAY started on port %d", s.gatewayManager.GetConfig().Port)
				logging.EmitModuleLog("knirvgateway", "info", fmt.Sprintf("Started on port %d", s.gatewayManager.GetConfig().Port))
			}
		}
	}

	if s.validationCore != nil {
		if err := s.validationCore.Start(s.ctx); err != nil {
			log.Printf("Warning: Failed to start validation core: %v", err)
			// Continue - validation core failure shouldn't stop basic server operation
		} else {
			log.Println("Validation Core started")
		}
	}

	// Start DVE manager after its dependent embedded services are ready.
	if s.dveManager != nil {
		if err := s.dveManager.Start(s.ctx); err != nil {
			log.Printf("Warning: Failed to start DVE manager: %v", err)
			// Continue starting other services - DVE manager failure shouldn't stop the server
		} else {
			log.Println("DVE Manager started")
		}
	}

	// Start DVE creation service
	if s.dveCreationService != nil {
		if err := s.dveCreationService.Start(); err != nil {
			log.Printf("Warning: Failed to start DVE creation service: %v", err)
		} else {
			log.Println("DVE Creation Service started")
		}
	}

	// Start fabric server
	if s.pluginServer != nil {
		if err := s.pluginServer.Start(); err != nil {
			log.Printf("Warning: Failed to start fabric server: %v", err)
			// Continue - fabric server failure shouldn't stop basic server operation
		} else {
			log.Println("Fabric Server started")
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

	// Start Fabric Management service
	if s.fabricManagementService != nil {
		if err := s.fabricManagementService.Start(); err != nil {
			log.Printf("Warning: Failed to start Fabric Management service: %v", err)
			// Continue - fabric management failure shouldn't stop basic server operation
		} else {
			log.Println("Fabric Management Service started")
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

	// Start WebSocket service
	if s.websocketService != nil {
		if err := s.websocketService.Start(); err != nil {
			log.Printf("Warning: Failed to start WebSocket service: %v", err)
		} else {
			log.Println("WebSocket Service started")
		}
	}

	// Start EventBroadcaster (Phase 8 - comprehensive event streaming)
	if s.eventBroadcaster != nil {
		s.eventBroadcaster.Start()
		log.Println("EventBroadcaster started")
	}

	// Start SecretManager session cleanup routine
	if s.secretManager != nil {
		go func() {
			ticker := time.NewTicker(5 * time.Minute)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					s.secretManager.CleanupExpiredSessions()
				case <-s.ctx.Done():
					return
				}
			}
		}()
		log.Println("SecretManager started")
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

	// Oracle is now started via Manager in NewServer
	// The knirvoracle binary is spawned when root.key is present
	if s.oracleManager != nil && s.oracleManager.IsRunning() {
		log.Println("Oracle service started via knirvoracle binary")
		logging.EmitModuleLog("oracle", "info", "Oracle service started via knirvoracle binary")
	}

	if s.rollupService != nil {
		s.runRollupLoop()
		log.Println("Rollup service started")
	}

	// Start Nexus Memory Fabric (Arrow Flight)
	if s.nexusServer != nil {
		go func() {
			if err := s.nexusServer.Serve(":50051"); err != nil {
				log.Printf("Nexus Memory Fabric server error: %v", err)
			}
		}()
	}

	// Validate server configuration before creating HTTP server
	if s.config == nil {
		return fmt.Errorf("invalid server configuration: config is nil")
	}
	// When using Unix socket, port is not required
	if s.config.API.SocketPath == "" && (s.config.API.BindAddress == "" || s.config.API.Port <= 0) {
		return fmt.Errorf("invalid server configuration: bind address and port must be specified when not using Unix socket")
	}

	serverAddr := fmt.Sprintf("%s:%d", s.config.API.BindAddress, s.config.API.Port)
	if s.config.API.SocketPath != "" {
		serverAddr = s.config.API.SocketPath
	}

	s.httpServer = &http.Server{
		Addr:         serverAddr,
		Handler:      middleware.CORSMiddlewareHTTP()(s.router),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	var listener net.Listener
	var err error
	if s.config.API.SocketPath != "" {
		if err := os.MkdirAll(filepath.Dir(s.config.API.SocketPath), 0755); err != nil {
			return fmt.Errorf("failed to create API socket directory: %w", err)
		}
		if err := os.Remove(s.config.API.SocketPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove existing API socket %s: %w", s.config.API.SocketPath, err)
		}
		listener, err = net.Listen("unix", s.config.API.SocketPath)
		if err != nil {
			return fmt.Errorf("failed to listen on unix socket %s: %w", s.config.API.SocketPath, err)
		}
		if err := os.Chmod(s.config.API.SocketPath, 0666); err != nil {
			log.Printf("Warning: Failed to set API socket permissions: %v", err)
		}
	} else {
		listener, err = net.Listen("tcp", fmt.Sprintf("%s:%d", s.config.API.BindAddress, s.config.API.Port))
		if err != nil {
			return fmt.Errorf("failed to listen on %s:%d: %w", s.config.API.BindAddress, s.config.API.Port, err)
		}
	}

	// Start HTTP server in goroutine after the listener is bound so startup
	// fails fast on socket/port issues instead of surfacing asynchronously.
	go func() {
		if s.config.API.SocketPath != "" {
			log.Printf("Starting HTTP server on unix socket: %s", s.config.API.SocketPath)
		} else {
			log.Printf("Starting HTTP server on %s:%d", s.config.API.BindAddress, s.config.API.Port)
		}
		if err := s.httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	s.running = true
	log.Println("KNIRV-SERVER backend server started successfully")
	return nil
}

// Stop stops all services and the HTTP server
func (s *Server) Stop() error {
	if !s.running {
		return nil
	}

	log.Println("Stopping KNIRV-SERVER backend server...")

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

	// Clean up Unix socket file after HTTP server shutdown
	if s.config != nil && s.config.API.SocketPath != "" {
		if err := os.Remove(s.config.API.SocketPath); err != nil && !os.IsNotExist(err) {
			log.Printf("Warning: Failed to remove API socket file %s: %v", s.config.API.SocketPath, err)
		} else {
			log.Printf("API socket file removed: %s", s.config.API.SocketPath)
		}
	}

	// Stop oracle manager (root node only)
	if s.oracleManager != nil {
		if err := s.oracleManager.Stop(context.Background()); err != nil {
			log.Printf("Error stopping oracle manager: %v", err)
		} else {
			log.Println("Oracle manager stopped")
		}
	}

	// Stop Hasher gRPC server
	if s.hasherGRPCServer != nil {
		if err := s.hasherGRPCServer.Stop(); err != nil {
			log.Printf("Error stopping Hasher gRPC server: %v", err)
		} else {
			log.Println("Hasher gRPC server stopped")
		}
	}

	// Stop Hasher integration client
	if s.hasherIntegration != nil {
		if err := s.hasherIntegration.Close(); err != nil {
			log.Printf("Error closing Hasher integration: %v", err)
		} else {
			log.Println("Hasher integration closed")
		}
	}

	// Stop services in reverse order
	if s.cognitiveEngine != nil {
		if err := s.cognitiveEngine.Stop(); err != nil {
			log.Printf("Error stopping Cognitive Engine: %v", err)
		}
	}

	if s.nexusServer != nil {
		if err := s.nexusServer.Stop(); err != nil {
			log.Printf("Error stopping Nexus Memory Fabric: %v", err)
		}
	}

	if s.websocketService != nil {
		if err := s.websocketService.Stop(); err != nil {
			log.Printf("Error stopping WebSocket service: %v", err)
		}
	}

	// Stop EventBroadcaster
	if s.eventBroadcaster != nil {
		s.eventBroadcaster.Stop()
		log.Println("EventBroadcaster stopped")
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

	if s.fabricManagementService != nil {
		if err := s.fabricManagementService.Stop(); err != nil {
			log.Printf("Error stopping Fabric Management service: %v", err)
		}
	}

	if s.controllerIntegrationService != nil {
		if err := s.controllerIntegrationService.Stop(); err != nil {
			log.Printf("Error stopping Controller Integration service: %v", err)
		}
	}

	if s.gatewayManager != nil {
		if err := s.gatewayManager.Stop(s.ctx); err != nil {
			log.Printf("Error stopping KNIRVGATEWAY: %v", err)
		} else {
			log.Println("KNIRVGATEWAY stopped")
		}
	}

	// Stop embedded KNIRVGRAPH
	if s.graphManager != nil {
		if err := s.graphManager.Stop(); err != nil {
			log.Printf("Error stopping KNIRVGRAPH: %v", err)
		} else {
			log.Println("KNIRVGRAPH stopped")
		}
	}

	// Stop graph sync manager
	if s.graphSyncManager != nil {
		s.graphSyncManager.Stop()
		log.Println("KNIRVGRAPH sync manager stopped")
	}

	// Stop embedded KNIRVCHAIN
	if s.chainManager != nil {
		if err := s.chainManager.Stop(s.ctx); err != nil {
			log.Printf("Error stopping KNIRVCHAIN: %v", err)
		} else {
			log.Println("KNIRVCHAIN stopped")
		}
	}

	if s.validationChainManager != nil {
		if err := s.validationChainManager.Stop(s.ctx); err != nil {
			log.Printf("Error stopping validation chain: %v", err)
		}
	}
	if s.transactionChainManager != nil {
		if err := s.transactionChainManager.Stop(s.ctx); err != nil {
			log.Printf("Error stopping transaction chain: %v", err)
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

	if s.pluginServer != nil {
		if err := s.pluginServer.Stop(); err != nil {
			log.Printf("Error stopping fabric server: %v", err)
		}
	}

	// Create context for stopping services
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer stopCancel()

	if s.dveCreationService != nil {
		if err := s.dveCreationService.Stop(); err != nil {
			log.Printf("Error stopping DVE creation service: %v", err)
		}
	}

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

	// Shutdown eBPF resources
	if s.virtualContainerManager != nil {
		if err := s.virtualContainerManager.ShutdownVirtualContainers(); err != nil {
			log.Printf("Error shutting down virtual container manager: %v", err)
		} else {
			log.Println("Virtual Container Manager shutdown successfully")
		}
	}

	if s.ebpfManager != nil {
		if err := s.ebpfManager.Shutdown(); err != nil {
			log.Printf("Error shutting down eBPF manager: %v", err)
		} else {
			log.Println("eBPF Manager shutdown successfully")
		}
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
	log.Println("KNIRV-SERVER backend server stopped")
	return nil
}

// initializeTEEEnvironment sets up the TEE environment with Kali-focused detection
func initializeTEEEnvironment(ctx context.Context, db *database.BuntDBManager) error {
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
	fmt.Printf("KNIRV-SERVER Complete Backend Server v%s (built %s, commit %s)\n", Version, BuildTime, GitCommit)

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

	// Load secrets from root.key and apply to config
	rootKeySecrets, err := loadSecretsFromKeyFile(nil)
	if err != nil {
		log.Printf("Warning: Failed to load secrets from root.key: %v", err)
	}
	if rootKeySecrets != nil {
		applyRootKeySecretsToConfig(config, rootKeySecrets)
		log.Printf("Applied secrets from root.key to configuration")
		// Re-expand paths in case secrets contained paths (like database URL)
		if err := config.ExpandPaths(); err != nil {
			log.Printf("Warning: Failed to re-expand paths after applying secrets: %v", err)
		}
	}

	// Create unified server
	server, err := NewServer(config, rootKeySecrets)
	if err != nil {
		return fmt.Errorf("failed to create server: %v", err)
	}

	// Start the server
	if err := server.Start(); err != nil {
		return fmt.Errorf("failed to start server: %v", err)
	}

	log.Println("--- KNIRVSERVER STARTUP ---")

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
