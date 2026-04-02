package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"backend_server/internal/config"
	data_engine "backend_server/internal/data_engine"
	"backend_server/internal/database"
	"backend_server/internal/ebpf"
	"backend_server/internal/logging"
	"backend_server/internal/oracle"
	oracleroutes "backend_server/internal/oracle/routes"
	"backend_server/internal/password"
	pb "backend_server/internal/proto"
	"backend_server/internal/reasoning/graph"
	"backend_server/internal/runtime"
	nexus "backend_server/internal/server"
	"backend_server/internal/services/active_memory"
	agentsvc "backend_server/internal/services/agent"
	"backend_server/internal/services/blockchain"
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
	"backend_server/internal/services/knirvcli"
	"backend_server/internal/services/onboarding"
	"backend_server/internal/services/p2p"
	fabricmanagement "backend_server/internal/services/pluginmanagement"
	pluginserver "backend_server/internal/services/pluginserver"
	secrets "backend_server/internal/services/secrets"
	"backend_server/internal/services/session"
	"backend_server/internal/services/systemhealth"
	"backend_server/internal/services/teesecurity"
	"backend_server/internal/services/validation"

	"backend_server/internal/services/vault"
	"backend_server/internal/services/websocket"
	"backend_server/internal/services/workflow"
	"backend_server/internal/storage/mdstorage"
	"backend_server/internal/storage/pqc"
	"backend_server/internal/web"
	"backend_server/internal/web/middleware"
	knirvchain "github.com/KNIRV/KNIRV_NETWORK/KNIRVSERVER/pkg/knirvchain"
	knirvgateway "github.com/KNIRV/KNIRV_NETWORK/KNIRVSERVER/pkg/knirvgateway"
	knirvgraph "github.com/KNIRV/KNIRV_NETWORK/KNIRVSERVER/pkg/knirvgraph"

	"github.com/apache/arrow/go/v14/arrow/memory"
	"github.com/gorilla/mux"
	"github.com/spf13/viper"
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
	config     *config.Config
	db         *database.BuntDBManager
	router     *mux.Router
	httpServer *http.Server
	p2pManager *p2p.DVEP2PManager
	nrnClient  *blockchain.NRNClient
	logger     *zap.Logger

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

	// KNIRVCLI Backend Integration Service
	knirvcliService *knirvcli.KNIRVCLIService

	// Onboarding Service - Value System and Ontology Ingestion
	onboardingService *onboarding.OnboardingService

	// Production Services (Phase 3, 4, 8)
	eventBroadcaster *websocket.EventBroadcaster
	anchoringService *evidence.AnchoringService
	secretManager    *secrets.SecretManager
	workflowService  *workflow.WorkflowService
	guardrailManager *guardrails.DynamicGuardrailManager
	policyEngine     *guardrails.PolicyEngine

	// Oracle service (root-only — only present when root.key is loaded)
	oracleService *oracle.Oracle

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

	// Secondary log path: project directory, only when KNIRV_PROJECT_LOG_DIR is explicitly set
	// to an absolute path by the outer process. Using a relative path here causes spurious log
	// files scattered across the filesystem depending on CWD at runtime.
	var multiWriter io.Writer
	projectLogPath := ""
	if projectLogDir := os.Getenv("KNIRV_PROJECT_LOG_DIR"); projectLogDir != "" {
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

// getOSAppDataDir returns the OS-specific application data directory
func getOSAppDataDir() (string, error) {
	var appDataDir string
	var err error

	// Try to get user config directory (XDG Base Directory on Linux)
	if userConfigDir, configErr := os.UserConfigDir(); configErr == nil {
		appDataDir = filepath.Join(userConfigDir, "knirvserver")
	} else {
		// Fallback to home directory
		if homeDir, homeErr := os.UserHomeDir(); homeErr == nil {
			// Use XDG_DATA_HOME or fallback to ~/.local/share
			if xdgDataHome := os.Getenv("XDG_DATA_HOME"); xdgDataHome != "" {
				appDataDir = filepath.Join(xdgDataHome, "knirvserver")
			} else {
				appDataDir = filepath.Join(homeDir, ".local", "share", "knirvserver")
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

// NewServer creates a new KNIRV-SERVER backend server instance
// initOracleFromKeyFile loads the encrypted root.key file and initialises the oracle service.
// Returns nil (no error) when the key file is absent — oracle is simply not started.
// The password is read from ORACLE_KEY_PASSWORD env var (non-interactive/CI) or prompted
// from stdin when running interactively.
func loadSecretsFromKeyFile(logger *zap.Logger) (*pb.RootKeyFileContentProto, error) {
	if logger == nil {
		logger = zap.NewNop()
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
	} else {
		// Check if stdin is a terminal
		if !term.IsTerminal(int(os.Stdin.Fd())) {
			return nil, fmt.Errorf("secrets: password not provided via ORACLE_KEY_PASSWORD environment variable and stdin is not a terminal")
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

	if content.DatabaseUrl != "" {
		cfg.Database.Path = content.DatabaseUrl
		viper.Set("database.path", content.DatabaseUrl)
		log.Printf("Database URL loaded from root.key: %s", content.DatabaseUrl)
	}

	if content.TlsCert != "" || content.TlsKey != "" {
		cfg.Security.TLSCert = content.TlsCert
		cfg.Security.TLSKey = content.TlsKey
		log.Printf("TLS certificates loaded from root.key")
	}
}

func initOracleWithSecrets(content *pb.RootKeyFileContentProto, logger *zap.Logger) (*oracle.Oracle, error) {
	if content == nil {
		return nil, nil
	}

	rootPrivateKey := content.GetRootPrivateKeyHex()
	if rootPrivateKey == "" {
		logger.Warn("Oracle disabled: root.key decrypted but ROOT_PRIVATE_KEY is empty")
		return nil, nil
	}

	oracleCfg, err := oracle.LoadConfigFromEnv()
	if err != nil {
		return nil, fmt.Errorf("oracle: failed to load config from env: %w", err)
	}
	oracleCfg.OwnerPrivateKey = rootPrivateKey

	if err := oracle.ValidateConfig(oracleCfg); err != nil {
		return nil, fmt.Errorf("oracle: invalid config: %w", err)
	}

	oracleInstance, err := oracle.NewOracle(oracleCfg, logger)
	if err != nil {
		return nil, fmt.Errorf("oracle: failed to create instance: %w", err)
	}

	return oracleInstance, nil
}

func initOracleFromKeyFile(logger *zap.Logger) (*oracle.Oracle, error) {
	if logger == nil {
		logger = zap.NewNop()
	}
	keyPath, err := config.GetRootKeyPath()
	if err != nil {
		return nil, fmt.Errorf("oracle: could not resolve root key path: %w", err)
	}

	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		logger.Info("Oracle disabled: no root.key found (not a root node)", zap.String("expected_path", keyPath))
		return nil, nil
	}

	// Determine password source.
	var keyPassword []byte
	if envPwd := os.Getenv("ORACLE_KEY_PASSWORD"); envPwd != "" {
		keyPassword = []byte(envPwd)
	} else {
		// Check if stdin is a terminal
		if !term.IsTerminal(int(os.Stdin.Fd())) {
			return nil, fmt.Errorf("oracle: password not provided via ORACLE_KEY_PASSWORD environment variable and stdin is not a terminal")
		}
		keyPassword, err = password.PromptForPassword("Enter root key password to start oracle: ")
		if err != nil {
			return nil, fmt.Errorf("oracle: failed to read password: %w", err)
		}
	}

	content, err := password.LoadEncryptedKeyFile(keyPath, keyPassword)
	if err != nil {
		return nil, fmt.Errorf("oracle: failed to decrypt root.key: %w", err)
	}

	oracleInstance, err := initOracleWithSecrets(content, logger)
	if err != nil {
		return nil, err
	}

	if oracleInstance != nil {
		logger.Info("Oracle initialised from root.key", zap.String("key_path", keyPath))
	}

	return oracleInstance, nil
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

	// Initialize blockchain client
	var nrnClient *blockchain.NRNClient
	if cfg.Blockchain.URL != "" {
		client, err := blockchain.NewNRNClient(cfg.Blockchain.URL, cfg.Blockchain.UseTLS, cfg.Blockchain.CertFile)
		if err != nil {
			log.Printf("Warning: Failed to initialize blockchain client: %v", err)
		} else {
			nrnClient = client
			if dveCreationService != nil {
				dveCreationService.SetChainClient(nrnClient)
				log.Println("Blockchain client integrated with DVE creation service")
			}
		}
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

	validationCore, err := validation.NewValidationCore(dbManager, p2pManager, cfg, inferenceService)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize validation core: %w", err)
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

	// Update system health service references
	systemHealthService.SetServiceReferences(dveManager, validationCore, inferenceService, teeSecurityService, nil)

	// Initialize embedded KNIRVGATEWAY for P2P TURN/Tunnel services
	var gatewayManager *knirvgateway.Manager
	if cfg.Gateway.Enabled {
		gatewayConfig := &knirvgateway.ManagerConfig{
			BinaryPath:     cfg.Gateway.BinaryPath,
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
			BinaryPath:   cfg.Graph.BinaryPath,
			Port:         cfg.Graph.Port,
			P2PPort:      cfg.Graph.P2PPort,
			APIPort:      cfg.Graph.APIPort,
			DataPath:     cfg.Graph.DataPath,
			StartTimeout: time.Duration(cfg.Graph.StartTimeout) * time.Second,
			StopTimeout:  time.Duration(cfg.Graph.StopTimeout) * time.Second,
			Stdout:       logging.NewSubprocessWriter("knirvgraph", os.Stdout),
			Stderr:       logging.NewSubprocessWriter("knirvgraph", os.Stderr),
		}
		graphManager = knirvgraph.NewManager(graphConfig, logger)
		log.Println("KNIRVGRAPH manager initialized")

		// Initialize SyncManager for staging to embedded sync
		syncInterval, _ := time.ParseDuration(cfg.Graph.SyncInterval)
		if syncInterval == 0 {
			syncInterval = 30 * time.Second
		}
		graphSyncConfig := &knirvgraph.SyncManagerConfig{
			GraphURL: fmt.Sprintf("http://localhost:%d", cfg.Graph.Port),
			Interval: syncInterval,
		}
		graphSyncManager = knirvgraph.NewSyncManager(graphSyncConfig, logger)
		log.Println("KNIRVGRAPH sync manager initialized")
	}

	// Initialize embedded KNIRVCHAIN for blockchain and mining
	var chainManager *knirvchain.Manager
	if cfg.Chain.Enabled {
		chainConfig := &knirvchain.ManagerConfig{
			BinaryPath:   cfg.Chain.BinaryPath,
			Port:         cfg.Chain.Port,
			P2PPort:      cfg.Chain.P2PPort,
			APIPort:      cfg.Chain.APIPort,
			DataPath:     cfg.Chain.DataPath,
			Role:         cfg.Chain.Role,
			ChainID:      cfg.Chain.ChainID,
			StartTimeout: time.Duration(cfg.Chain.StartTimeout) * time.Second,
			StopTimeout:  time.Duration(cfg.Chain.StopTimeout) * time.Second,
			Stdout:       logging.NewSubprocessWriter("knirvchain", os.Stdout),
			Stderr:       logging.NewSubprocessWriter("knirvchain", os.Stderr),
		}
		chainManager = knirvchain.NewManager(chainConfig, logger)
		log.Println("KNIRVCHAIN manager initialized")
	}

	// Initialize Cognitive Engine with configurable parameters
	cognitiveEngine := cognitiveengine.NewCognitiveEngine(dbManager, validationCore, inferenceService, fabricManagementService)

	// Wire eBPF manager for real resource telemetry and kernel-level guardrails
	if ebpfManager != nil {
		cognitiveEngine.SetEBPFManager(ebpfManager)
		log.Println("CognitiveEngine: eBPF manager wired for real resource telemetry")
	}

	// Initialize KNIRVCLI Backend Integration Service
	knirvcliService := knirvcli.NewKNIRVCLIService(dbManager, dveManager, validationCore)
	log.Println("KNIRVCLI service initialized")

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

	// Initialize AnchoringService for PQC-signed evidence pack anchoring
	anchoringService := evidence.NewAnchoringService(dbManager, pqcManager, "server-master")
	if nrnClient != nil {
		anchoringService.SetChainClient(&nrnChainAdapter{client: nrnClient})
		log.Println("AnchoringService: blockchain client wired for chain anchoring")
	}
	if err := anchoringService.LoadEvidencePacks(); err != nil {
		log.Printf("Warning: Failed to load evidence packs: %v", err)
	}
	log.Println("AnchoringService initialized")

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
	workflowService.RegisterExecutor("validation", workflow.NewDVETaskExecutor(dveManager, validationCore))
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

	// Wire EventBroadcaster to WebSocket service (will be set in setupRoutes)
	_ = eventBroadcaster

	// Create context for service lifecycle management
	ctx, cancel := context.WithCancel(context.Background())

	server := &Server{
		config:                       cfg,
		db:                           dbManager,
		router:                       router,
		p2pManager:                   p2pManager,
		nrnClient:                    nrnClient,
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
		guardrailManager:             guardrailManager,
		policyEngine:                 policyEngine,
		knirvcliService:              knirvcliService,
		onboardingService:            onboardingService,
		ctx:                          ctx,
		cancel:                       cancel,
		running:                      false,
	}

	// Initialise oracle if root.key is present (root node only)
	var oracleInstance *oracle.Oracle
	if rootKeySecrets != nil {
		oracleInstance, err = initOracleWithSecrets(rootKeySecrets, logger)
	} else {
		oracleInstance, err = initOracleFromKeyFile(logger)
	}
	if err != nil {
		logger.Error("Failed to initialise oracle — continuing without it", zap.Error(err))
	}
	server.oracleService = oracleInstance

	// Setup routes for all services
	server.setupRoutes()

	// Integrate blockchain client with services after server creation
	if nrnClient != nil && server.icmeService != nil {
		blockchainAdapter := &blockchainClientAdapter{client: nrnClient}
		server.icmeService.SetBlockchainClient(blockchainAdapter)
		log.Println("Blockchain client integrated with ICME service")
	}

	return server, nil
}

type blockchainClientAdapter struct {
	client *blockchain.NRNClient
}

func (a *blockchainClientAdapter) SubmitTransaction(tx interface{}) (string, error) {
	txMap, ok := tx.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid transaction format")
	}

	txType, _ := txMap["type"].(string)
	txData, _ := txMap["data"].(string)

	blockchainTx := &blockchain.Transaction{
		Type: txType,
		Data: []byte(txData),
	}

	return a.client.SubmitTransaction(blockchainTx)
}

type nrnChainAdapter struct {
	client *blockchain.NRNClient
}

func (a *nrnChainAdapter) SubmitTransaction(tx *evidence.ChainTransaction) (string, error) {
	return a.client.SubmitTransaction(&blockchain.Transaction{
		Type: tx.Type,
		Data: tx.Data,
	})
}

func (a *nrnChainAdapter) GetBlockHeight() (uint64, error) {
	return 0, nil
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
	}

	// Register Active Memory handlers
	if s.activeMemoryService != nil {
		memoryHandlers := web.NewActiveMemoryHandlers(s.activeMemoryService)
		memoryHandlers.RegisterRoutes(s.router, authMiddleware)
		log.Println("Active Memory (Markdown Fabric) routes configured")
	}

	// Register NRN payment routes
	if s.nrnClient != nil {
		nrnHandlers := web.NewNRNPaymentHandlers(s.nrnClient)
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

	// Register KNIRVCLI routes
	if s.knirvcliService != nil {
		web.NewKNIRVCLIHandlers(s.knirvcliService).RegisterRoutes(s.router, authMiddleware)
		log.Println("KNIRVCLI routes configured")
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
		web.NewKNIRVCLIHandlers(s.knirvcliService),
		web.NewOnboardingHandlers(s.onboardingService),
		web.NewCognitiveEngineHandlers(s.cognitiveEngine),
		authMiddleware,
	)
	apiRouter.RegisterRoutes(s.router)
	log.Println("Unified API router configured")

	// Register system settings routes
	systemSettingsHandlers := web.NewSystemSettingsHandlers(s.config)
	systemSettingsHandlers.RegisterRoutes(s.router, authMiddleware)
	log.Println("System settings routes configured")

	// Register oracle routes (root node only — only wired when oracle is active)
	if s.oracleService != nil {
		oracleMux := http.NewServeMux()
		oracleRoutes := oracleroutes.NewOracleRoutes(s.oracleService, s.logger)
		oracleRoutes.RegisterRoutes(oracleMux)
		s.router.PathPrefix("/oracle/").Handler(oracleMux)
		log.Println("Oracle routes configured")
	}

	log.Println("All routes configured successfully")
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

	// Start oracle (root node only)
	if s.oracleService != nil {
		if err := s.oracleService.Start(); err != nil {
			log.Printf("Warning: Failed to start oracle service: %v", err)
			logging.EmitModuleLog("oracle", "error", fmt.Sprintf("Failed to start: %v", err))
		} else {
			log.Println("Oracle service started")
			logging.EmitModuleLog("oracle", "info", "Oracle service started")
		}
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

	// Start embedded KNIRVGATEWAY for P2P TURN/Tunnel
	if s.gatewayManager != nil {
		if err := s.gatewayManager.Start(s.ctx); err != nil {
			log.Printf("Warning: Failed to start KNIRVGATEWAY: %v", err)
			logging.EmitModuleLog("knirvgateway", "error", fmt.Sprintf("Failed to start: %v", err))
		} else {
			log.Printf("KNIRVGATEWAY started on port %d", s.gatewayManager.GetConfig().Port)
			logging.EmitModuleLog("knirvgateway", "info", fmt.Sprintf("Started on port %d", s.gatewayManager.GetConfig().Port))
		}
	}

	// Start embedded KNIRVGRAPH
	if s.graphManager != nil && s.config.Graph.Enabled {
		if err := s.graphManager.Start(); err != nil {
			log.Printf("Warning: Failed to start KNIRVGRAPH: %v", err)
			logging.EmitModuleLog("knirvgraph", "error", fmt.Sprintf("Failed to start: %v", err))
		} else {
			log.Printf("KNIRVGRAPH started on port %d", s.graphManager.GetConfig().Port)
			logging.EmitModuleLog("knirvgraph", "info", fmt.Sprintf("Started on port %d", s.graphManager.GetConfig().Port))
		}
	}

	// Start embedded KNIRVCHAIN
	if s.chainManager != nil && s.config.Chain.Enabled {
		if err := s.chainManager.Start(s.ctx); err != nil {
			log.Printf("Warning: Failed to start KNIRVCHAIN: %v", err)
			logging.EmitModuleLog("knirvchain", "error", fmt.Sprintf("Failed to start: %v", err))
		} else {
			log.Printf("KNIRVCHAIN started on port %d", s.chainManager.GetConfig().APIPort)
			logging.EmitModuleLog("knirvchain", "info", fmt.Sprintf("Started on port %d", s.chainManager.GetConfig().APIPort))
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

	// Start Nexus Memory Fabric (Arrow Flight)
	if s.nexusServer != nil {
		go func() {
			if err := s.nexusServer.Serve(":50051"); err != nil {
				log.Printf("Nexus Memory Fabric server error: %v", err)
			}
		}()
	}

	// Validate server configuration before creating HTTP server
	if s.config == nil || s.config.API.BindAddress == "" || s.config.API.Port <= 0 {
		return fmt.Errorf("invalid server configuration: bind address and port must be specified")
	}

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", s.config.API.BindAddress, s.config.API.Port),
		Handler:      middleware.CORSMiddlewareHTTP()(s.router),
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

	// Stop oracle service (root node only)
	if s.oracleService != nil {
		if err := s.oracleService.Stop(); err != nil {
			log.Printf("Error stopping oracle service: %v", err)
		} else {
			log.Println("Oracle service stopped")
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

	if s.nrnClient != nil {
		if err := s.nrnClient.Close(); err != nil {
			log.Printf("Error closing blockchain client: %v", err)
		}
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
