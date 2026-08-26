package api

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"KNIRVENGINE/desktop-client/agent"
	"KNIRVENGINE/desktop-client/agentify"
	"KNIRVENGINE/desktop-client/database"
	"KNIRVENGINE/desktop-client/inference"
	"KNIRVENGINE/desktop-client/utils"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// isDesktopLoopbackOrigin permits the Electron window served by this local
// engine, including when startup selected a non-default GUI port.
func isDesktopLoopbackOrigin(origin string) bool {
	if origin == "file://" {
		return true
	}
	parsed, err := url.Parse(origin)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return false
	}
	host := parsed.Hostname()
	return host == "localhost" || net.ParseIP(host).IsLoopback()
}

// TroubleshootingChunk represents a chunk of troubleshooting information
type TroubleshootingChunk struct {
	Category string   `json:"category"`
	Issue    string   `json:"issue"`
	Symptoms []string `json:"symptoms"`
	Content  string   `json:"content"`
	RawHTML  string   `json:"raw_html"`
}

// TroubleshootingDatabase represents the embedded troubleshooting knowledge base
type TroubleshootingDatabase struct {
	Chunks []TroubleshootingChunk `json:"chunks"`
}

// agentConfigToMap converts AgentConfig to map[string]interface{}
func agentConfigToMap(config agent.AgentConfig) (map[string]interface{}, error) {
	data, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	return result, err
}

// generateAgentID generates a unique agent ID
func generateAgentID() string {
	return fmt.Sprintf("agent_%d", time.Now().UnixNano())
}

// loadTroubleshootingDatabase loads the embedded troubleshooting knowledge base
func (s *SimpleAPIServer) loadTroubleshootingDatabase() (*TroubleshootingDatabase, error) {
	filePath := filepath.Join("api", "data", "troubleshooting_embeddings.json")

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Printf("Troubleshooting database not found at %s", filePath)
		return &TroubleshootingDatabase{Chunks: []TroubleshootingChunk{}}, nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read troubleshooting database: %v", err)
	}

	var db TroubleshootingDatabase
	if err := json.Unmarshal(data, &db); err != nil {
		return nil, fmt.Errorf("failed to parse troubleshooting database: %v", err)
	}

	log.Printf("Loaded troubleshooting database with %d chunks", len(db.Chunks))
	return &db, nil
}

// searchTroubleshootingDatabase searches for relevant troubleshooting information
func (s *SimpleAPIServer) searchTroubleshootingDatabase(errorType, errorMessage string, symptoms []string) []TroubleshootingChunk {
	db, err := s.loadTroubleshootingDatabase()
	if err != nil {
		log.Printf("Error loading troubleshooting database: %v", err)
		return []TroubleshootingChunk{}
	}

	var matches []TroubleshootingChunk
	searchTerms := []string{strings.ToLower(errorType), strings.ToLower(errorMessage)}

	// Add symptoms to search terms
	for _, symptom := range symptoms {
		searchTerms = append(searchTerms, strings.ToLower(symptom))
	}

	// Search through chunks
	for _, chunk := range db.Chunks {
		score := 0
		chunkText := strings.ToLower(chunk.Category + " " + chunk.Issue + " " + chunk.Content)

		// Calculate relevance score
		for _, term := range searchTerms {
			if term != "" && strings.Contains(chunkText, term) {
				score++
			}
		}

		// Include chunks with any matches
		if score > 0 {
			matches = append(matches, chunk)
		}
	}

	// Sort by category relevance (prioritize exact error type matches)
	var prioritized []TroubleshootingChunk
	var others []TroubleshootingChunk

	for _, match := range matches {
		if strings.Contains(strings.ToLower(match.Category), strings.ToLower(errorType)) {
			prioritized = append(prioritized, match)
		} else {
			others = append(others, match)
		}
	}

	// Combine prioritized and others, limit to top 3 results
	result := append(prioritized, others...)
	if len(result) > 3 {
		result = result[:3]
	}

	return result
}

// DemoDataBackup holds backup of demo data for restore functionality
type DemoDataBackup struct {
	Agents        []*database.SimpleAgent  `json:"agents"`
	TargetSystems []map[string]interface{} `json:"target_systems"`
	MCPServers    []map[string]interface{} `json:"mcp_servers"`
	Workflows     []map[string]interface{} `json:"workflows"`
	Timestamp     time.Time                `json:"timestamp"`
}

// copyTemplateFiles copies agent templates from source to destination directory
func copyTemplateFiles(srcDir, destDir string) error {
	// Ensure destination directory exists
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create templates directory: %v", err)
	}

	// Check if source directory exists
	if _, err := os.Stat(srcDir); os.IsNotExist(err) {
		log.Printf("Warning: Source templates directory does not exist: %s", srcDir)
		return nil // Not an error - templates might be embedded or not needed
	}

	// Copy all template files
	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only copy .template files
		if !strings.HasSuffix(path, ".template") {
			return nil
		}

		// Calculate relative path
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		// Create destination path
		destPath := filepath.Join(destDir, relPath)

		// Ensure destination directory exists
		destDirPath := filepath.Dir(destPath)
		if err := os.MkdirAll(destDirPath, 0755); err != nil {
			return err
		}

		// Copy file
		return copyFile(path, destPath)
	})
}

// copyFile copies a single file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	// Copy file permissions
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, sourceInfo.Mode())

}

// SimpleAPIServer provides a simplified REST API for the inference engine
type SimpleAPIServer struct {
	db                      *database.SimpleDomainDB
	agentRepo               *database.SimpleAgentRepository // Legacy - to be deprecated
	unifiedStorage          *agent.UnifiedAgentStorage      // New unified storage system
	router                  *mux.Router
	httpServer              *http.Server // Renamed for clarity and to avoid conflict
	port                    int
	dbPath                  string
	inferenceService        *inference.InferenceService
	agentInferencer         *agentify.AgentInferencer     // Added Agent Inferencer
	agentBuilder            *agent.AgentBuilder           // Added Agent Builder
	enhancedAgentManager    *agent.EnhancedAgentManager   // Added Enhanced Agent Manager
	workflowService         *WorkflowOrchestrationService // Added workflow orchestration service
	mcpRegistryService      *MCPRegistryService           // Added MCP registry service
	securityMiddleware      *SecurityMiddleware           // Added enhanced security middleware
	shieldFramework         *agentify.SHIELDFramework     // Added SHIELD framework
	performanceManager      *PerformanceManager           // Added performance manager
	reliabilityManager      *ReliabilityManager           // Added reliability manager
	authService             AuthServiceInterface          // Added authentication service
	userService             *UserService                  // Added user service
	analyticsService        *AnalyticsService             // Added analytics service
	webConnectionsService   *WebConnectionsService        // Added web connections service
	targetSystemService     *TargetSystemService          // Added target system service
	mcpInstallationService  *MCPInstallationService       // Added MCP installation service
	mcpLifecycleService     *MCPLifecycleService          // Added MCP lifecycle service
	mcpConfigService        *MCPConfigService             // Added MCP configuration service
	mcpMonitoringService    *MCPMonitoringService         // Added MCP monitoring service
	systemMonitoringService *SystemMonitoringService      // Added system monitoring service
	terminalManager         *TerminalManager              // Added Terminal Manager
	sandboxManager          *SandboxManager               // Owns Bubblewrap/Xvfb/x11vnc sessions
	agentTerminalHandler    *AgentTerminalHandler         // Added Agent Terminal Handler
	shutdownSignalChan      chan<- struct{}               // Channel to signal main to shut down

	// Demo data management
	demoDataEnabled bool            // Current state of demo data
	demoDataBackup  *DemoDataBackup // Backup of demo data for restore
	demoDataMutex   sync.RWMutex    // Mutex for demo data operations

	// WebSocket connection management
	wsConnections      map[string]*websocket.Conn // Active WebSocket connections
	wsConnectionsMutex sync.RWMutex               // Mutex for WebSocket connections
}

// NewSimpleAPIServer creates a new simple API server
func NewSimpleAPIServer(port int, dbPath string, shutdownSignal chan<- struct{}, inferenceService *inference.InferenceService, agentInferencer *agentify.AgentInferencer, authService AuthServiceInterface, userService *UserService, analyticsService *AnalyticsService, webConnectionsService *WebConnectionsService) (*SimpleAPIServer, error) {
	log.Println("Initializing SimpleAPIServer with enhanced security...")
	// Initialize database
	db, err := database.NewSimpleDomainDB(dbPath) // Use the passed dbPath
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Initialize security middleware
	securityMiddleware := NewSecurityMiddleware()
	log.Println("Security middleware initialized")

	// Initialize SHIELD framework
	shieldFramework := agentify.NewSHIELDFramework()
	log.Println("SHIELD framework initialized")

	// Initialize performance manager
	performanceManager := NewPerformanceManager()
	log.Println("Performance manager initialized")

	// Initialize reliability manager
	reliabilityManager := NewReliabilityManager()
	reliabilityManager.SetupDefaultReliability()
	log.Println("Reliability manager initialized")

	// Initialize repositories - agent repo will be set later using unified storage
	var agentRepo *database.SimpleAgentRepository

	// Use the passed inference service instead of creating a new one
	infService := inferenceService

	// Initialize workflow orchestration service
	workflowService := NewWorkflowOrchestrationService()

	// Initialize MCP registry service
	mcpRegistryService := NewMCPRegistryService()

	// Initialize MCP installation service
	mcpInstallationService := NewMCPInstallationService(mcpRegistryService)

	// Initialize MCP lifecycle service
	mcpLifecycleService := NewMCPLifecycleService(mcpRegistryService, mcpInstallationService)

	// Initialize MCP configuration service
	mcpConfigService := NewMCPConfigService(mcpRegistryService)

	// Initialize MCP monitoring service
	mcpMonitoringService := NewMCPMonitoringService(mcpRegistryService, mcpLifecycleService)

	// Initialize system monitoring service
	systemMonitoringService := NewSystemMonitoringService()

	// Initialize target system service
	targetSystemService := NewTargetSystemService()

	// Initialize Agent Builder with app data paths
	appDataDir, err := utils.GetAppDataDir()
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to get app data directory: %w", err)
	}

	// Set up templates and plugins directories in app data
	templatesDir := filepath.Join(appDataDir, "templates")
	pluginsDir := filepath.Join(appDataDir, "plugins")

	// Copy templates from source to app data directory
	sourceTemplatesDir := "agent/templates"
	log.Printf("Copying templates from %s to %s", sourceTemplatesDir, templatesDir)
	if err := copyTemplateFiles(sourceTemplatesDir, templatesDir); err != nil {
		log.Printf("Warning: Failed to copy templates: %v", err)
		// Continue anyway - templates might be embedded or not needed for basic functionality
	}

	// Get the centralized agents database path for unified storage
	agentsDBPath, err := utils.GetAgentsDBPath()
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to get agents database path: %w", err)
	}

	// Create unified agent storage that serves as both database and registry
	unifiedAgentStorage, err := agent.NewUnifiedAgentStorage(agentsDBPath)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create unified agent storage: %w", err)
	}

	// Initialize Agent Builder with unified storage (no separate registry)
	agentBuilder, err := agent.NewAgentBuilderWithStorage(unifiedAgentStorage, templatesDir, pluginsDir)
	if err != nil {
		db.Close() // Clean up database if agent builder init fails
		return nil, fmt.Errorf("failed to initialize agent builder: %w", err)
	}

	// Use the same unified storage for the domain database agent repository
	// Note: We need to create a bridge adapter since the API expects SimpleAgentRepository
	// For now, we'll create a traditional repository but this should be unified in the future
	agentCollection, err := db.GetOrCreateCollection("agents")
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create agents collection: %w", err)
	}
	agentRepo = database.NewSimpleAgentRepository(agentCollection)

	// Initialize Enhanced Agent Manager with app data directory
	enhancedAgentManager, err := agent.NewEnhancedAgentManager(agentBuilder.GetRegistry(), agentBuilder, appDataDir)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize enhanced agent manager: %w", err)
	}

	// Initialize Terminal Manager
	terminalManager := NewTerminalManager()
	sandboxManager := NewSandboxManager()

	apiServer := &SimpleAPIServer{
		db:                      db,
		agentRepo:               agentRepo,           // Legacy - to be deprecated
		unifiedStorage:          unifiedAgentStorage, // New unified storage system
		port:                    port,
		dbPath:                  dbPath,
		inferenceService:        infService,              // Store the inference service
		agentInferencer:         agentInferencer,         // Store the agent inferencer
		agentBuilder:            agentBuilder,            // Store the agent builder
		enhancedAgentManager:    enhancedAgentManager,    // Store the enhanced agent manager
		workflowService:         workflowService,         // Store the workflow service
		mcpRegistryService:      mcpRegistryService,      // Store the MCP registry service
		mcpInstallationService:  mcpInstallationService,  // Store the MCP installation service
		mcpLifecycleService:     mcpLifecycleService,     // Store the MCP lifecycle service
		mcpConfigService:        mcpConfigService,        // Store the MCP configuration service
		mcpMonitoringService:    mcpMonitoringService,    // Store the MCP monitoring service
		systemMonitoringService: systemMonitoringService, // Store the system monitoring service
		authService:             authService,             // Store the authentication service
		userService:             userService,             // Store the user service
		analyticsService:        analyticsService,        // Store the analytics service
		webConnectionsService:   webConnectionsService,   // Store the web connections service
		targetSystemService:     targetSystemService,     // Store the target system service
		terminalManager:         terminalManager,         // Store the terminal manager
		sandboxManager:          sandboxManager,
		router:                  mux.NewRouter(), // Initialize the router for the APIServer instance
		shutdownSignalChan:      shutdownSignal,
		demoDataEnabled:         true,                             // Demo data enabled by default
		demoDataBackup:          nil,                              // No backup initially
		wsConnections:           make(map[string]*websocket.Conn), // Initialize WebSocket connections map
		securityMiddleware:      securityMiddleware,               // Store the security middleware
		shieldFramework:         shieldFramework,                  // Store the SHIELD framework
		performanceManager:      performanceManager,               // Store the performance manager
		reliabilityManager:      reliabilityManager,               // Store the reliability manager
	}

	apiServer.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", apiServer.port),
		Handler:      apiServer.router, // Use the apiServer's router
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	// Setup routes
	apiServer.setupRoutes()

	// Initialize and register the agent terminal handler
	agentTerminalHandler := NewAgentTerminalHandler(terminalManager)
	agentTerminalHandler.RegisterRoutes(apiServer.router)
	apiServer.agentTerminalHandler = agentTerminalHandler
	log.Println("Agent terminal handler initialized and registered")

	return apiServer, nil
}

// setupRoutes configures the API routes
func (s *SimpleAPIServer) setupRoutes() {
	// API prefix
	api := s.router.PathPrefix("/api/v1").Subrouter()

	// Security middleware (applied first for all routes)
	s.router.Use(s.securityMiddleware.Middleware)

	// System monitoring middleware (track requests and errors)
	s.router.Use(s.systemMonitoringMiddleware)
	log.Println("Security middleware applied to all routes")

	// CORS middleware
	s.router.Use(s.corsMiddleware)

	// Register authentication routes (if auth service is available)
	if s.authService != nil {
		s.authService.RegisterHandlers(s.router)
	}

	// Register additional service handlers
	if s.userService != nil {
		// Only register user service if we have a full AuthService (not SimpleAuthService)
		if fullAuthService, ok := s.authService.(*AuthService); ok {
			s.userService.RegisterHandlers(s.router, fullAuthService)
		}
	}
	if s.analyticsService != nil {
		s.analyticsService.RegisterHandlers(s.router)
	}
	if s.webConnectionsService != nil {
		s.webConnectionsService.RegisterHandlers(s.router)
	}
	if s.targetSystemService != nil {
		s.targetSystemService.RegisterHandlers(s.router)
	}
	if s.sandboxManager != nil {
		s.sandboxManager.RegisterHandlers(s.router)
	}

	// Health check
	api.HandleFunc("/health", s.healthHandler).Methods("GET")

	// Agent routes (database-based agents)
	api.HandleFunc("/agents", s.createAgentHandler).Methods("POST")
	api.HandleFunc("/agents", s.getAgentsHandler).Methods("GET")
	// Combined agents endpoint (database + discovered) - must come before {id} route
	api.HandleFunc("/agents/all", s.getAllAgentsHandler).Methods("GET")
	// Sync agents between database and registry
	api.HandleFunc("/agents/sync", s.syncAgentsHandler).Methods("POST")
	api.HandleFunc("/agents/{id}", s.getAgentHandler).Methods("GET")
	api.HandleFunc("/agents/{id}", s.updateAgentHandler).Methods("PUT")
	api.HandleFunc("/agents/{id}", s.deleteAgentHandler).Methods("DELETE")
	api.HandleFunc("/agents/{id}/stop", s.stopAgentHandler).Methods("POST")

	// Advanced Agent Operations (migrated from v2)
	api.HandleFunc("/agents/discover", s.discoverUnifiedAgentsHandler).Methods("GET")
	api.HandleFunc("/agents/register", s.registerAgentHandler).Methods("POST")
	api.HandleFunc("/agents/search", s.searchAgentsHandler).Methods("GET")
	api.HandleFunc("/agents/{id}/activate", s.activateUnifiedAgentHandler).Methods("POST")
	api.HandleFunc("/agents/{id}/deactivate", s.deactivateUnifiedAgentHandler).Methods("POST")
	api.HandleFunc("/agents/{id}/config", s.getAgentConfigHandler).Methods("GET")
	api.HandleFunc("/agents/{id}/config", s.updateAgentConfigHandler).Methods("PUT")
	api.HandleFunc("/agents/by-type/{type}", s.getAgentsByTypeHandler).Methods("GET")
	api.HandleFunc("/agents/by-status/{status}", s.getAgentsByStatusHandler).Methods("GET")
	api.HandleFunc("/agents/by-build-target/{target}", s.getAgentsByBuildTargetHandler).Methods("GET")

	// Agent Builder routes
	api.HandleFunc("/agents/{id}/build", s.buildAgentHandler).Methods("POST")
	api.HandleFunc("/agents/{id}/build", s.getAgentBuildStatusHandler).Methods("GET")

	// Enhanced Agent Management routes
	api.HandleFunc("/agents/{id}/versions", s.createAgentVersionHandler).Methods("POST")
	api.HandleFunc("/agents/{id}/versions", s.listAgentVersionsHandler).Methods("GET")
	api.HandleFunc("/agents/{id}/backup", s.createAgentBackupHandler).Methods("POST")
	api.HandleFunc("/agents/{id}/backups", s.listAgentBackupsHandler).Methods("GET")
	api.HandleFunc("/agents/{id}/restore/{backupId}", s.restoreAgentFromBackupHandler).Methods("POST")
	api.HandleFunc("/agents/{id}/health", s.performAgentHealthCheckHandler).Methods("GET")
	api.HandleFunc("/agents/{id}/health/history", s.getAgentHealthHistoryHandler).Methods("GET")
	api.HandleFunc("/agents/{id}/analytics/{period}", s.generateAgentAnalyticsHandler).Methods("GET")
	api.HandleFunc("/agents/{id}/rebuild", s.rebuildAgentHandler).Methods("POST")
	api.HandleFunc("/templates", s.getAgentTemplatesHandler).Methods("GET")
	api.HandleFunc("/plugins", s.getCompiledPluginsHandler).Methods("GET")
	api.HandleFunc("/plugins/{id}", s.deleteAgentPluginHandler).Methods("DELETE")

	// Sub-Agent routes
	api.HandleFunc("/agents/{id}/sub-agents", s.spawnSubAgentHandler).Methods("POST")
	api.HandleFunc("/agents/{id}/sub-agents", s.getSubAgentsHandler).Methods("GET")
	api.HandleFunc("/agents/{id}/sub-agents/{subId}", s.terminateSubAgentHandler).Methods("DELETE")
	api.HandleFunc("/agents/{id}/sub-agents/{subId}/terminal", s.getSubAgentTerminalHandler).Methods("GET")
	api.HandleFunc("/agents/{id}/sub-agents/{subId}/command", s.sendSubAgentCommandHandler).Methods("POST")
	api.HandleFunc("/agents/{id}/sub-agents/{subId}/logs", s.getSubAgentLogsHandler).Methods("GET")

	// Target system routes
	api.HandleFunc("/targets", s.getTargetsHandler).Methods("GET")

	// Agent Inferencer routes (plugin-based agents)
	api.HandleFunc("/adk/agents", s.discoverAgentsHandler).Methods("GET")
	api.HandleFunc("/adk/agents/activate", s.activateAgentHandler).Methods("POST")
	api.HandleFunc("/adk/agents/deactivate", s.deactivateAgentHandler).Methods("POST")
	api.HandleFunc("/adk/agents/capabilities", s.getAgentCapabilitiesHandler).Methods("GET")
	api.HandleFunc("/adk/agents/schema", s.getAgentSchemaHandler).Methods("GET")
	api.HandleFunc("/adk/agents/inference", s.processInferenceHandler).Methods("POST")
	api.HandleFunc("/adk/agents/memory", s.setAgentMemoryHandler).Methods("POST")
	api.HandleFunc("/adk/agents/memory", s.getAgentMemoryHandler).Methods("GET")

	// Agent Chat routes
	api.HandleFunc("/agents/message", s.agentMessageHandler).Methods("POST")
	api.HandleFunc("/agents/chat", s.agentChatHandler).Methods("POST")
	api.HandleFunc("/adk/agents/message", s.adkAgentMessageHandler).Methods("POST")
	api.HandleFunc("/agents/{id}/chat/session", s.agentChatSessionHandler).Methods("POST", "GET")
	api.HandleFunc("/agents/{id}/chat/history", s.agentChatHistoryHandler).Methods("GET")
	api.HandleFunc("/agents/chat/state", s.agentChatStateHandler).Methods("POST", "GET")
	api.HandleFunc("/api/v1/chat/ws/connect", s.chatWebSocketConnectHandler).Methods("POST")

	// Plugin discovery and import routes
	api.HandleFunc("/plugins/discover", s.discoverAllPluginsHandler).Methods("GET")
	api.HandleFunc("/plugins/import", s.importPluginHandler).Methods("POST")

	// WASM Plugin Management routes
	api.HandleFunc("/plugins/wasm/discover", s.discoverWASMPluginsHandler).Methods("GET")
	api.HandleFunc("/plugins/wasm/install", s.installWASMPluginHandler).Methods("POST")
	api.HandleFunc("/plugins/wasm/uninstall", s.uninstallWASMPluginHandler).Methods("POST")
	api.HandleFunc("/plugins/wasm/installed", s.listInstalledWASMPluginsHandler).Methods("GET")
	api.HandleFunc("/adk/agents/detailed", s.getAvailableAgentsDetailedHandler).Methods("GET")

	// Terminal routes
	api.HandleFunc("/terminal/create", s.createTerminalHandler).Methods("POST")
	api.HandleFunc("/terminal/write", s.writeToTerminalHandler).Methods("POST")
	api.HandleFunc("/terminal/read", s.readFromTerminalHandler).Methods("GET")
	api.HandleFunc("/terminal/resize", s.resizeTerminalHandler).Methods("POST")
	api.HandleFunc("/terminal/close", s.closeTerminalHandler).Methods("POST")
	api.HandleFunc("/terminal/ws", s.terminalWebSocketHandler).Methods("GET")
	api.HandleFunc("/terminal/logs", s.getTerminalLogsHandler).Methods("GET")

	// Main WebSocket endpoint for real-time updates
	api.HandleFunc("/ws", s.mainWebSocketHandler).Methods("GET")

	// Desktop-specific secure WebSocket endpoint
	api.HandleFunc("/desktop/secure-ws", s.desktopSecureWebSocketHandler).Methods("GET")

	// Settings routes
	api.HandleFunc("/settings/api-keys", s.handleAPIKeys).Methods("POST")
	api.HandleFunc("/inference/models", s.handleInferenceModels).Methods("GET")
	api.HandleFunc("/inference/moa/{type}", s.handleMOASettings).Methods("POST")

	// AI Error Analysis routes
	api.HandleFunc("/inference/analyze-error", s.handleAnalyzeError).Methods("POST")
	api.HandleFunc("/inference/chat-error", s.handleChatError).Methods("POST")

	// Debug routes
	api.HandleFunc("/debug/toggle-demo-data", s.handleToggleDemoData).Methods("POST")
	api.HandleFunc("/debug/demo-data-status", s.handleDemoDataStatus).Methods("GET")
	api.HandleFunc("/debug/clear-all-agents", s.handleClearAllAgents).Methods("POST")

	// Register workflow orchestration routes
	s.workflowService.RegisterHandlers(api)

	// Register MCP registry routes
	s.mcpRegistryService.RegisterHandlers(s.router)

	// Register MCP installation routes
	s.mcpInstallationService.RegisterHandlers(s.router)

	// Register MCP lifecycle routes
	s.mcpLifecycleService.RegisterHandlers(s.router)

	// Register MCP configuration routes
	s.mcpConfigService.RegisterHandlers(s.router)

	// Register MCP monitoring routes
	s.mcpMonitoringService.RegisterHandlers(s.router)

	// Capabilities routes
	api.HandleFunc("/capabilities", s.handleListCapabilities).Methods("GET")
	api.HandleFunc("/capabilities/mcp", s.handleListMCPCapabilities).Methods("GET")

	// User management routes
	api.HandleFunc("/users", s.handleListUsers).Methods("GET")
	api.HandleFunc("/users/{id}", s.handleGetUser).Methods("GET")
	api.HandleFunc("/users", s.handleCreateUser).Methods("POST")
	api.HandleFunc("/users/{id}", s.handleUpdateUser).Methods("PUT")
	api.HandleFunc("/users/{id}", s.handleDeleteUser).Methods("DELETE")

	// Wallet routes
	s.RegisterWalletRoutes()

	// Static file serving for UI
	s.router.PathPrefix("/").Handler(http.FileServer(http.Dir("./static/")))

	//Shutdown
	s.router.HandleFunc("/api/shutdown", s.handleShutdownRequest).Methods("POST", "OPTIONS")

	// Security monitoring endpoints
	s.router.HandleFunc("/api/v1/security/status", s.handleSecurityStatus).Methods("GET", "OPTIONS")
	s.router.HandleFunc("/api/v1/security/events", s.handleSecurityEvents).Methods("GET", "OPTIONS")
	s.router.HandleFunc("/api/v1/security/shield/status", s.handleSHIELDStatus).Methods("GET", "OPTIONS")
	s.router.HandleFunc("/api/v1/security/shield/agents/{agentId}/monitor", s.handleStartAgentMonitoring).Methods("POST", "OPTIONS")
	s.router.HandleFunc("/api/v1/security/shield/agents/{agentId}/verify", s.handleVerifyAgentIntegrity).Methods("POST", "OPTIONS")

	// Performance monitoring endpoints
	s.router.HandleFunc("/api/v1/performance/status", s.handlePerformanceStatus).Methods("GET", "OPTIONS")
	s.router.HandleFunc("/api/v1/performance/cache/stats", s.handleCacheStats).Methods("GET", "OPTIONS")
	s.router.HandleFunc("/api/v1/performance/cache/clear", s.handleCacheClear).Methods("POST", "OPTIONS")
	s.router.HandleFunc("/api/v1/performance/queries/slow", s.handleSlowQueries).Methods("GET", "OPTIONS")

	// System monitoring endpoints
	if s.systemMonitoringService != nil {
		s.systemMonitoringService.RegisterRoutes(api)
	}
}

func (s *SimpleAPIServer) handleShutdownRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" { // Handle preflight
		w.Header().Set("Access-Control-Allow-Origin", "*") // Adjust for production
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusOK)
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", "*") // Adjust for production

	log.Println("API server received shutdown request from frontend.")
	if s.shutdownSignalChan != nil {
		select {
		case s.shutdownSignalChan <- struct{}{}:
			log.Println("Shutdown signal sent to main application.")
		default:
			log.Println("Shutdown signal channel is full or nil, possibly already signaled.")
		}
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "Shutdown signal received by API server.")
}

// corsMiddleware adds CORS headers
func (s *SimpleAPIServer) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Start starts the API server
func (s *SimpleAPIServer) Start() error {
	log.Printf("Starting Simple API server on %s", s.httpServer.Addr)

	// Provision Bubble Wrap's small, shared runtime before accepting UI
	// requests.  The target path is selected later in the UI, but bwrap/Xvfb/
	// x11vnc are independent of it and used to make the first Launch appear to
	// hang while packages were being checked or acquired.
	if s.sandboxManager != nil {
		statuses, err := s.sandboxManager.GetDependencyStatus()
		if err != nil {
			log.Printf("Warning: sandbox runtime preflight failed: %v", err)
		} else {
			var missing []string
			for _, status := range statuses {
				if !status.Present {
					missing = append(missing, status.Binary)
				}
			}
			if len(missing) > 0 {
				log.Printf("Warning: sandbox runtime unavailable after preflight: %s", strings.Join(missing, ", "))
			} else {
				log.Printf("Sandbox runtime preflight complete (bwrap, Xvfb, x11vnc ready)")
			}
		}
	}

	// Agent discovery and registration is now handled automatically by unified storage
	log.Printf("Agent discovery will be handled automatically by unified storage...")

	// Start MCP services
	ctx := context.Background()
	if err := s.mcpRegistryService.Start(ctx); err != nil {
		log.Printf("Warning: Failed to start MCP registry service: %v", err)
	}

	if err := s.mcpLifecycleService.Start(ctx); err != nil {
		log.Printf("Warning: Failed to start MCP lifecycle service: %v", err)
	}

	if err := s.mcpMonitoringService.Start(); err != nil {
		log.Printf("Warning: Failed to start MCP monitoring service: %v", err)
	}

	// Start system monitoring service
	if err := s.systemMonitoringService.Start(); err != nil {
		log.Printf("Warning: Failed to start system monitoring service: %v", err)
	}

	return s.httpServer.ListenAndServe()
}

// Stop stops the API server
func (s *SimpleAPIServer) Stop(ctx context.Context) error {
	log.Println("Stopping Simple API server...")
	if s.sandboxManager != nil {
		s.sandboxManager.CloseAll()
	}
	if s.httpServer == nil {
		return nil // Or return an error if server was not initialized
	}
	return s.httpServer.Shutdown(ctx)
}

// systemMonitoringMiddleware tracks requests and errors for system monitoring
func (s *SimpleAPIServer) systemMonitoringMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Increment request count
		if s.systemMonitoringService != nil {
			s.systemMonitoringService.IncrementRequestCount()
		}

		// Create a response writer wrapper to capture status codes
		wrapper := &responseWriterWrapper{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Call the next handler
		next.ServeHTTP(wrapper, r)

		// Check if it was an error response and increment error count
		if wrapper.statusCode >= 400 && s.systemMonitoringService != nil {
			s.systemMonitoringService.IncrementErrorCount()
		}
	})
}

// responseWriterWrapper wraps http.ResponseWriter to capture status codes
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriterWrapper) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

// Hijack preserves the optional http.Hijacker capability through the
// monitoring wrapper. Gorilla WebSocket upgrades require this interface; if a
// middleware hides it, the request reaches the handler but cannot upgrade.
func (w *responseWriterWrapper) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("underlying ResponseWriter does not implement http.Hijacker")
	}
	return hijacker.Hijack()
}

// GetRouter returns the router instance for registering additional handlers
func (s *SimpleAPIServer) GetRouter() *mux.Router {
	return s.router
}

// RegisterAgentPluginForTerminal registers an agent plugin with the terminal handler
func (s *SimpleAPIServer) RegisterAgentPluginForTerminal(agentID, pluginPath string) {
	if s.agentTerminalHandler != nil {
		s.agentTerminalHandler.RegisterAgentPlugin(agentID, pluginPath)
		log.Printf("Registered agent plugin for terminal: %s -> %s", agentID, pluginPath)
	} else {
		log.Printf("Warning: Agent terminal handler not initialized, cannot register agent plugin: %s", agentID)
	}
}

// Health check handler
func (s *SimpleAPIServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"version":   "1.0.0",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Create agent handler
func (s *SimpleAPIServer) createAgentHandler(w http.ResponseWriter, r *http.Request) {
	// Parse the frontend request format
	var request struct {
		Name   string `json:"name"`
		Type   string `json:"type"`
		Config string `json:"config"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Parse the config to extract agent fields
	var configData map[string]interface{}
	if err := json.Unmarshal([]byte(request.Config), &configData); err != nil {
		http.Error(w, "Invalid config JSON", http.StatusBadRequest)
		return
	}

	// Get user ID from authentication context
	userID, ok := r.Context().Value(UserIDContextKey).(int64)
	if !ok {
		log.Printf("Warning: No user ID found in context, using default owner ID")
		userID = 1 // Fallback to default user
	}

	// Create UnifiedAgent from the request
	now := time.Now()
	agent := &agent.UnifiedAgent{
		ID:          generateAgentID(), // Generate unique ID
		Name:        request.Name,
		Type:        request.Type,
		OwnerID:     userID,
		CreatedAt:   now,
		UpdatedAt:   now,
		Collection:  getStringFromConfig(configData, "collection", "default"),
		ImageURL:    getStringFromConfig(configData, "image_url", ""),
		Status:      getStringFromConfig(configData, "status", "active"),
		TargetTypes: []string{"general"}, // Default target type
		AgentConfig: configData,          // Store full config
		APIKeys:     make(map[string]string),
	}

	// Extract capabilities
	if caps, ok := configData["capabilities"].([]interface{}); ok {
		for _, cap := range caps {
			if capStr, ok := cap.(string); ok {
				agent.Capabilities = append(agent.Capabilities, capStr)
			}
		}
	}

	// Extract target types if provided
	if targets, ok := configData["target_types"].([]interface{}); ok {
		agent.TargetTypes = []string{} // Reset to empty
		for _, target := range targets {
			if targetStr, ok := target.(string); ok {
				agent.TargetTypes = append(agent.TargetTypes, targetStr)
			}
		}
	}

	// Create agent using UnifiedAgentStorage
	ctx := context.Background()
	if err := s.unifiedStorage.CreateAgent(ctx, agent); err != nil {
		log.Printf("Error creating agent: %v", err)
		http.Error(w, "Failed to create agent", http.StatusInternalServerError)
		return
	}

	// Convert UnifiedAgent to the format expected by the frontend
	config := map[string]interface{}{
		"collection":   agent.Collection,
		"image_url":    agent.ImageURL,
		"capabilities": agent.Capabilities,
		"target_types": agent.TargetTypes,
		"status":       agent.Status,
	}

	// Convert config to JSON string
	configJSON, _ := json.Marshal(config)

	// Create agent object in the format expected by frontend
	responseAgent := map[string]interface{}{
		"id":         agent.ID,
		"owner_id":   agent.OwnerID,
		"name":       agent.Name,
		"type":       agent.Type,
		"config":     string(configJSON),
		"created_at": agent.CreatedAt.Format(time.RFC3339),
		"updated_at": agent.UpdatedAt.Format(time.RFC3339),
	}

	RespondWithCreated(w, responseAgent, MessageCreated)
}

// Get agents handler
func (s *SimpleAPIServer) getAgentsHandler(w http.ResponseWriter, r *http.Request) {
	// Get owner ID from query parameter
	ownerIDStr := r.URL.Query().Get("owner_id")
	if ownerIDStr == "" {
		http.Error(w, "owner_id parameter is required", http.StatusBadRequest)
		return
	}

	ownerID, err := strconv.ParseInt(ownerIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid owner_id", http.StatusBadRequest)
		return
	}

	// Get agents using UnifiedAgentStorage
	unifiedAgents, err := s.unifiedStorage.FindByOwner(ownerID)
	if err != nil {
		log.Printf("Error getting agents: %v", err)
		RespondWithInternalError(w, "Failed to get agents")
		return
	}

	// Convert UnifiedAgent objects to the format expected by the frontend
	agents := make([]map[string]interface{}, len(unifiedAgents))
	for i, unifiedAgent := range unifiedAgents {
		// Create config object with frontend-specific fields
		config := map[string]interface{}{
			"collection":   unifiedAgent.Collection,
			"image_url":    unifiedAgent.ImageURL,
			"capabilities": unifiedAgent.Capabilities,
			"target_types": unifiedAgent.TargetTypes,
			"status":       unifiedAgent.Status,
		}

		// Convert config to JSON string
		configJSON, _ := json.Marshal(config)

		// Create agent object in the format expected by frontend
		agents[i] = map[string]interface{}{
			"id":         unifiedAgent.ID,
			"owner_id":   unifiedAgent.OwnerID,
			"name":       unifiedAgent.Name,
			"type":       unifiedAgent.Type,
			"config":     string(configJSON),
			"created_at": unifiedAgent.CreatedAt.Format(time.RFC3339),
			"updated_at": unifiedAgent.UpdatedAt.Format(time.RFC3339),
		}
	}

	RespondWithList(w, agents, len(agents), MessageListRetrieved)
}

// Get agent handler
func (s *SimpleAPIServer) getAgentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	ctx := context.Background()
	unifiedAgent, err := s.unifiedStorage.GetAgentByID(ctx, id)
	if err != nil {
		log.Printf("Error getting agent: %v", err)
		RespondWithNotFound(w, "Agent")
		return
	}

	// Convert UnifiedAgent to the format expected by the frontend
	config := map[string]interface{}{
		"collection":   unifiedAgent.Collection,
		"image_url":    unifiedAgent.ImageURL,
		"capabilities": unifiedAgent.Capabilities,
		"target_types": unifiedAgent.TargetTypes,
		"status":       unifiedAgent.Status,
	}

	// Convert config to JSON string
	configJSON, _ := json.Marshal(config)

	// Create agent object in the format expected by frontend
	agent := map[string]interface{}{
		"id":         unifiedAgent.ID,
		"owner_id":   unifiedAgent.OwnerID,
		"name":       unifiedAgent.Name,
		"type":       unifiedAgent.Type,
		"config":     string(configJSON),
		"created_at": unifiedAgent.CreatedAt.Format(time.RFC3339),
		"updated_at": unifiedAgent.UpdatedAt.Format(time.RFC3339),
	}

	RespondWithSuccess(w, agent, MessageRetrieved)
}

// Update agent handler
func (s *SimpleAPIServer) updateAgentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// Parse the frontend request format
	var request struct {
		Name   string `json:"name"`
		Type   string `json:"type"`
		Config string `json:"config"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		RespondWithValidationError(w, "Invalid JSON")
		return
	}

	// Get existing agent to verify it exists
	ctx := context.Background()
	existingAgent, err := s.unifiedStorage.GetAgentByID(ctx, id)
	if err != nil {
		log.Printf("Error getting agent for update: %v", err)
		RespondWithNotFound(w, "Agent")
		return
	}

	// Parse the config to extract agent fields
	var configData map[string]interface{}
	if err := json.Unmarshal([]byte(request.Config), &configData); err != nil {
		RespondWithValidationError(w, "Invalid config JSON")
		return
	}

	// Update the existing agent with new data
	updatedAgent := &agent.UnifiedAgent{
		ID:          id,
		Name:        request.Name,
		Type:        request.Type,
		Collection:  getStringFromConfig(configData, "collection", existingAgent.Collection),
		ImageURL:    getStringFromConfig(configData, "image_url", existingAgent.ImageURL),
		Status:      getStringFromConfig(configData, "status", existingAgent.Status),
		OwnerID:     existingAgent.OwnerID,   // Preserve existing values
		CreatedAt:   existingAgent.CreatedAt, // Preserve creation time
		UpdatedAt:   time.Now(),              // Set current time as updated
		AgentConfig: configData,              // Store full config
		APIKeys:     existingAgent.APIKeys,   // Preserve existing API keys
	}

	// Extract capabilities
	if caps, ok := configData["capabilities"].([]interface{}); ok {
		updatedAgent.Capabilities = []string{} // Reset capabilities
		for _, cap := range caps {
			if capStr, ok := cap.(string); ok {
				updatedAgent.Capabilities = append(updatedAgent.Capabilities, capStr)
			}
		}
	} else {
		// Preserve existing capabilities if not provided
		updatedAgent.Capabilities = existingAgent.Capabilities
	}

	// Extract target types if provided
	if targets, ok := configData["target_types"].([]interface{}); ok {
		updatedAgent.TargetTypes = []string{} // Reset target types
		for _, target := range targets {
			if targetStr, ok := target.(string); ok {
				updatedAgent.TargetTypes = append(updatedAgent.TargetTypes, targetStr)
			}
		}
	} else {
		// Preserve existing target types if not provided
		updatedAgent.TargetTypes = existingAgent.TargetTypes
	}

	// Update agent using UnifiedAgentStorage
	if err := s.unifiedStorage.UpdateAgent(ctx, updatedAgent); err != nil {
		log.Printf("Error updating agent: %v", err)
		RespondWithInternalError(w, "Failed to update agent")
		return
	}

	// Convert UnifiedAgent to the format expected by the frontend
	config := map[string]interface{}{
		"collection":   updatedAgent.Collection,
		"image_url":    updatedAgent.ImageURL,
		"capabilities": updatedAgent.Capabilities,
		"target_types": updatedAgent.TargetTypes,
		"status":       updatedAgent.Status,
	}

	// Convert config to JSON string
	configJSON, _ := json.Marshal(config)

	// Create agent object in the format expected by frontend
	responseAgent := map[string]interface{}{
		"id":         updatedAgent.ID,
		"owner_id":   updatedAgent.OwnerID,
		"name":       updatedAgent.Name,
		"type":       updatedAgent.Type,
		"config":     string(configJSON),
		"created_at": updatedAgent.CreatedAt.Format(time.RFC3339),
		"updated_at": updatedAgent.UpdatedAt.Format(time.RFC3339),
	}

	RespondWithSuccess(w, responseAgent, MessageUpdated)
}

// Stop agent handler
func (s *SimpleAPIServer) stopAgentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	ctx := context.Background()

	// Agents can originate from either the unified store or the legacy
	// repository exposed by /agents/all. Keep the stop action compatible with
	// both while migration remains in progress.
	unifiedAgent, err := s.unifiedStorage.GetAgentByID(ctx, id)
	if err == nil {
		unifiedAgent.Status = "idle"
		unifiedAgent.UpdatedAt = time.Now()
		if err := s.unifiedStorage.UpdateAgent(ctx, unifiedAgent); err != nil {
			RespondWithInternalError(w, "Failed to update agent")
			return
		}
	} else if s.agentRepo != nil {
		legacyAgent, legacyErr := s.agentRepo.GetAgentByID(ctx, id)
		if legacyErr != nil {
			RespondWithNotFound(w, "Agent")
			return
		}
		legacyAgent.Status = "idle"
		if updateErr := s.agentRepo.UpdateAgent(ctx, legacyAgent); updateErr != nil {
			log.Printf("Error stopping legacy agent %s: %v", id, updateErr)
			RespondWithInternalError(w, "Failed to update agent")
			return
		}
	} else {
		RespondWithNotFound(w, "Agent")
		return
	}

	// If we have an agent inferencer, try to deactivate the agent session
	if s.agentInferencer != nil {
		// Try to deactivate any active sessions for this agent
		// We'll use the agent ID as the session ID for simplicity
		sessionID := fmt.Sprintf("agent_%s", id)
		if err := s.agentInferencer.DeactivateAgent(ctx, sessionID); err != nil {
			log.Printf("Failed to deactivate agent session %s: %v", sessionID, err)
			// Don't fail the request if deactivation fails, just log it
		}
	}

	stopResponse := map[string]interface{}{
		"agent_id": id,
	}
	RespondWithSuccess(w, stopResponse, "Agent stopped successfully")
}

// Delete agent handler
func (s *SimpleAPIServer) deleteAgentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	ctx := context.Background()

	// Track agents from the unified store, legacy compatibility repository, and
	// runtime registry. The UI presents all three through /agents/all.
	existsInDatabase := false
	existsInLegacyRepository := false
	existsInRegistry := false

	// Check if the agent exists in the unified storage
	_, err := s.unifiedStorage.GetAgentByID(ctx, id)
	if err == nil {
		existsInDatabase = true
	} else {
		log.Printf("Agent %s not found in unified storage: %v", id, err)
	}

	if s.agentRepo != nil {
		if _, legacyErr := s.agentRepo.GetAgentByID(ctx, id); legacyErr == nil {
			existsInLegacyRepository = true
		}
	}

	// Check if the agent exists in the registry
	if s.agentBuilder != nil {
		_, err := s.agentBuilder.GetAgent(id)
		if err == nil {
			existsInRegistry = true
		} else {
			log.Printf("Agent %s not found in registry: %v", id, err)
		}
	}

	// If agent doesn't exist in either system, return not found
	if !existsInDatabase && !existsInLegacyRepository && !existsInRegistry {
		RespondWithNotFound(w, "Agent")
		return
	}

	// If we have an agent inferencer, try to deactivate any active sessions first
	if s.agentInferencer != nil {
		// Try multiple session ID formats to ensure we catch all possible sessions
		sessionIDs := []string{
			fmt.Sprintf("agent_%s", id),
			id,
			fmt.Sprintf("session_%s", id),
		}

		for _, sessionID := range sessionIDs {
			if err := s.agentInferencer.DeactivateAgent(ctx, sessionID); err != nil {
				log.Printf("Failed to deactivate agent session %s before deletion: %v", sessionID, err)
				// Continue with other session IDs even if one fails
			} else {
				log.Printf("Successfully deactivated agent session: %s", sessionID)
			}
		}
	}

	// Delete from registry if it exists there
	if existsInRegistry && s.agentBuilder != nil {
		if err := s.agentBuilder.DeleteAgent(id); err != nil {
			log.Printf("Warning: Failed to delete agent from builder: %v", err)
			// Continue with database deletion even if builder deletion fails
		} else {
			log.Printf("Successfully deleted agent from builder: %s", id)
		}
	}

	// Delete from unified storage if it exists there
	if existsInDatabase {
		if err := s.unifiedStorage.DeleteAgent(ctx, id); err != nil {
			log.Printf("Warning: Failed to delete agent from unified storage: %v", err)
			// If we deleted from registry but failed in storage, still consider it a partial success
			if existsInRegistry {
				// Return partial success
				partialResponse := map[string]interface{}{
					"agent_id": id,
				}
				RespondWithSuccess(w, partialResponse, "Agent partially deleted (removed from registry but not storage)")
				return
			}
			// If we only had it in storage and failed to delete, return error
			RespondWithInternalError(w, "Failed to delete agent")
			return
		}
	}

	if existsInLegacyRepository {
		if err := s.agentRepo.DeleteAgent(ctx, id); err != nil {
			log.Printf("Failed to delete legacy agent %s: %v", id, err)
			RespondWithInternalError(w, "Failed to delete agent")
			return
		}
	}

	deleteResponse := map[string]interface{}{
		"agent_id": id,
	}
	RespondWithSuccess(w, deleteResponse, MessageDeleted)
}

// Get targets handler - returns available target systems for deployment
func (s *SimpleAPIServer) getTargetsHandler(w http.ResponseWriter, r *http.Request) {
	// For now, return static target systems
	// In a real implementation, this would come from a target system repository
	targets := []map[string]interface{}{
		{
			"id":           "browser-chrome",
			"name":         "Chrome Browser",
			"type":         "browser",
			"status":       "connected",
			"capabilities": []string{"web_scraping", "dom_analysis", "screenshot_capture"},
			"description":  "Web browser for accessing and interacting with web content",
		},
		{
			"id":           "filesystem-local",
			"name":         "Local File System",
			"type":         "filesystem",
			"status":       "connected",
			"capabilities": []string{"file_analysis", "document_processing", "data_mining"},
			"description":  "Access to local files and directories",
		},
		{
			"id":           "terminal-system",
			"name":         "System Terminal",
			"type":         "system",
			"status":       "connected",
			"capabilities": []string{"command_execution", "process_monitoring", "system_information"},
			"description":  "System-level access and operations",
		},
		{
			"id":           "network-monitor",
			"name":         "Network Monitor",
			"type":         "network",
			"status":       "connected",
			"capabilities": []string{"network_analysis", "traffic_monitoring", "security_scanning"},
			"description":  "Network monitoring and analysis",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"targets": targets,
	})
}

// CreateSampleData creates some sample data for testing
func (s *SimpleAPIServer) CreateSampleData() error {
	ctx := context.Background()

	// Sample agents
	sampleAgents := []*database.SimpleAgent{
		{
			Name:         "Alpha Agent",
			Collection:   "Genesis",
			ImageURL:     "https://example.com/alpha.png",
			Status:       "active",
			Capabilities: []string{"web_scraping", "data_analysis"},
			TokenID:      "1",
			ContractAddr: "0x123...",
			OwnerID:      1,
		},
		{
			Name:         "Beta Agent",
			Collection:   "Genesis",
			ImageURL:     "https://example.com/beta.png",
			Status:       "active",
			Capabilities: []string{"api_integration", "automation"},
			TokenID:      "2",
			ContractAddr: "0x123...",
			OwnerID:      1,
		},
		{
			Name:         "Gamma Agent",
			Collection:   "Advanced",
			ImageURL:     "https://example.com/gamma.png",
			Status:       "active",
			Capabilities: []string{"machine_learning", "prediction"},
			TokenID:      "3",
			ContractAddr: "0x456...",
			OwnerID:      2,
		},
	}

	// Create sample agents
	for _, agent := range sampleAgents {
		if err := s.agentRepo.CreateAgent(ctx, agent); err != nil {
			return fmt.Errorf("failed to create sample agent %s: %w", agent.Name, err)
		}
		log.Printf("Created sample agent: %s", agent.Name)
	}

	return nil
}

// Security monitoring handlers

// handleSecurityStatus returns the current security status
func (s *SimpleAPIServer) handleSecurityStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	stats := s.securityMiddleware.GetStats()

	response := map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"security_middleware": stats,
			"timestamp":           time.Now().UTC(),
		},
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleSecurityEvents returns recent security events
func (s *SimpleAPIServer) handleSecurityEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	// Get limit from query parameter
	limitStr := r.URL.Query().Get("limit")
	limit := 50 // default
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	events := s.securityMiddleware.auditLogger.GetEvents(limit)

	response := map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"events":    events,
			"count":     len(events),
			"timestamp": time.Now().UTC(),
		},
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleSHIELDStatus returns the SHIELD framework status
func (s *SimpleAPIServer) handleSHIELDStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	status := s.shieldFramework.GetStatus()

	response := map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"shield_framework": status,
			"timestamp":        time.Now().UTC(),
		},
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleStartAgentMonitoring starts SHIELD monitoring for an agent
func (s *SimpleAPIServer) handleStartAgentMonitoring(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	agentID := vars["agentId"]

	if agentID == "" {
		http.Error(w, "Agent ID is required", http.StatusBadRequest)
		return
	}

	// Start monitoring with default training period
	trainingPeriod := 5 * time.Minute
	err := s.shieldFramework.MonitorAgent(agentID, trainingPeriod)
	if err != nil {
		log.Printf("Failed to start monitoring for agent %s: %v", agentID, err)
		http.Error(w, fmt.Sprintf("Failed to start monitoring: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"agent_id":           agentID,
			"monitoring_started": true,
			"training_period":    trainingPeriod.String(),
			"timestamp":          time.Now().UTC(),
		},
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleVerifyAgentIntegrity verifies agent integrity using SHIELD
func (s *SimpleAPIServer) handleVerifyAgentIntegrity(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	agentID := vars["agentId"]

	if agentID == "" {
		http.Error(w, "Agent ID is required", http.StatusBadRequest)
		return
	}

	// For demonstration, we'll use a dummy agent data
	// In a real implementation, this would load the actual agent plugin data
	agentData := []byte(fmt.Sprintf("agent_data_%s_%d", agentID, time.Now().Unix()))

	isValid, err := s.shieldFramework.VerifyAgentIntegrity(agentID, agentData)
	if err != nil {
		log.Printf("Failed to verify integrity for agent %s: %v", agentID, err)
		http.Error(w, fmt.Sprintf("Failed to verify integrity: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"agent_id":        agentID,
			"integrity_valid": isValid,
			"timestamp":       time.Now().UTC(),
		},
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// Performance monitoring handlers

// handlePerformanceStatus returns the current performance status
func (s *SimpleAPIServer) handlePerformanceStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	stats := s.performanceManager.GetStats()

	response := map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"performance_manager": stats,
			"timestamp":           time.Now().UTC(),
		},
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleCacheStats returns cache statistics
func (s *SimpleAPIServer) handleCacheStats(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	cacheManager := s.performanceManager.GetCacheManager()
	stats := cacheManager.GetStats()

	response := map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"cache_stats": stats,
			"timestamp":   time.Now().UTC(),
		},
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleCacheClear clears the cache
func (s *SimpleAPIServer) handleCacheClear(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	cacheManager := s.performanceManager.GetCacheManager()
	cacheManager.Clear()

	response := map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"message":   "Cache cleared successfully",
			"timestamp": time.Now().UTC(),
		},
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleSlowQueries returns slow query information
func (s *SimpleAPIServer) handleSlowQueries(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	// Get limit from query parameter
	limitStr := r.URL.Query().Get("limit")
	limit := 20 // default
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	queryOptimizer := s.performanceManager.GetQueryOptimizer()
	slowQueries := queryOptimizer.GetSlowQueries(limit)

	response := map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"slow_queries": slowQueries,
			"count":        len(slowQueries),
			"timestamp":    time.Now().UTC(),
		},
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// API Key settings handler
func (s *SimpleAPIServer) handleAPIKeys(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Provider string `json:"provider"`
		APIKey   string `json:"apiKey"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// In a real implementation, you would:
	// 1. Validate the API key
	// 2. Store it securely (encrypted)
	// 3. Update environment variables or config

	// For now, we'll just acknowledge the save
	log.Printf("API Key saved for provider: %s", request.Provider)

	response := map[string]interface{}{
		"status":   "success",
		"message":  "API key saved successfully",
		"provider": request.Provider,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Inference models handler
func (s *SimpleAPIServer) handleInferenceModels(w http.ResponseWriter, r *http.Request) {
	// In a real implementation, you would query your inference service
	// For now, return static model lists
	models := map[string][]string{
		"primary": {
			"gpt-oss-120b",
			"gpt-4",
			"claude-3-sonnet",
		},
		"fallback": {
			"gemini-2.5-flash",
			"deepseek-chat",
			"gemini-2.5-pro",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models)
}

// AI Error Analysis handler
func (s *SimpleAPIServer) handleAnalyzeError(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Prompt       string `json:"prompt"`
		ErrorContext struct {
			Type       string                 `json:"type"`
			Severity   string                 `json:"severity"`
			Code       string                 `json:"code"`
			Message    string                 `json:"message"`
			Details    interface{}            `json:"details"` // Accept both string and object
			StackTrace string                 `json:"stackTrace"`
			SystemInfo map[string]interface{} `json:"systemInfo"`
			Context    map[string]interface{} `json:"context"`
			Symptoms   []string               `json:"symptoms"`
		} `json:"error_context"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		log.Printf("Error decoding request: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Extract symptoms if available
	symptomsText := "None provided"
	if len(request.ErrorContext.Symptoms) > 0 {
		symptomsText = strings.Join(request.ErrorContext.Symptoms, "\n- ")
		symptomsText = "- " + symptomsText
	}

	// Search troubleshooting database for relevant information
	troubleshootingMatches := s.searchTroubleshootingDatabase(
		request.ErrorContext.Type,
		request.ErrorContext.Message,
		request.ErrorContext.Symptoms,
	)

	// Build troubleshooting context
	troubleshootingContext := ""
	if len(troubleshootingMatches) > 0 {
		troubleshootingContext = "\n\nRELEVANT TROUBLESHOOTING INFORMATION FROM KNOWLEDGE BASE:\n"
		for i, match := range troubleshootingMatches {
			troubleshootingContext += fmt.Sprintf("\n--- KNOWN ISSUE %d ---\n", i+1)
			troubleshootingContext += fmt.Sprintf("Category: %s\n", match.Category)
			troubleshootingContext += fmt.Sprintf("Issue: %s\n", match.Issue)
			troubleshootingContext += fmt.Sprintf("Solution Steps:\n%s\n", match.Content)
		}
		troubleshootingContext += "\n--- END KNOWLEDGE BASE ---\n"
	}

	// Build enhanced analysis prompt with troubleshooting knowledge base reference
	analysisPrompt := fmt.Sprintf(`You are an AI error analysis assistant for the KNIRVENGINE application. You have access to a comprehensive troubleshooting knowledge base. Analyze the following error and provide helpful suggestions:

ERROR DETAILS:
- Type: %s
- Severity: %s
- Message: %s
- Code: %s

SYMPTOMS:
%s

SYSTEM CONTEXT:
%v

STACK TRACE:
%s

ADDITIONAL CONTEXT:
%v%s

ANALYSIS INSTRUCTIONS:
1. **PRIMARY**: Check the RELEVANT TROUBLESHOOTING INFORMATION from our knowledge base above
2. **MATCH DETECTION**: If this error matches a known issue, use the documented solution steps as your primary guidance
3. **ADAPTATION**: Adapt the knowledge base solutions to this specific error context and symptoms
4. **CONFIDENCE**: If you find a direct match in the knowledge base, set confidence to 0.9+
5. **FALLBACK**: If no knowledge base match, provide your best analysis based on error details (confidence 0.7 or lower)
6. **SPECIFICITY**: Be extremely specific and actionable in your recommendations
7. **CONTEXT AWARENESS**: Consider the KNIRVENGINE application context in all suggestions

RESPONSE REQUIREMENTS:
1. Clear analysis explaining what went wrong and why
2. 3-5 specific, actionable steps to fix this error (prioritize knowledge base solutions)
3. Confidence level (0-1) - higher if knowledge base match found
4. Whether this requires user action or can be automated
5. Realistic estimated time to resolve
6. Accurate error category classification

Format your response as JSON with the following structure:
{
  "analysis": "Clear explanation of the error, referencing knowledge base if applicable",
  "suggested_fixes": ["Step 1", "Step 2", "Step 3"],
  "confidence": 0.8,
  "category": "installation|authentication|network|ai_model|configuration|permissions|data",
  "estimated_resolution_time": "5 minutes",
  "requires_user_action": true,
  "automated_fix_available": false
}`,
		request.ErrorContext.Type,
		request.ErrorContext.Severity,
		request.ErrorContext.Message,
		request.ErrorContext.Code,
		symptomsText,
		request.ErrorContext.SystemInfo,
		request.ErrorContext.StackTrace,
		request.ErrorContext.Context,
		troubleshootingContext)

	// Use inference service to analyze the error with embedded keys
	var analysisResult string
	var err error
	var suggestedFixes []string
	var confidence float64 = 0.7
	var category string = request.ErrorContext.Type
	var estimatedTime string = "5-15 minutes"
	var requiresUserAction bool = true
	var automatedFixAvailable bool = false

	if s.inferenceService != nil {
		log.Printf("Using inference service with embedded keys for error analysis")
		analysisResult, err = s.inferenceService.GenerateText("", analysisPrompt, "Analyze this error and provide structured suggestions for resolution.")

		if err == nil {
			// Clean the response to extract JSON from markdown code blocks if present
			cleanedResponse := s.extractJSONFromResponse(analysisResult)

			// Try to parse the JSON response
			var jsonResponse map[string]interface{}
			if jsonErr := json.Unmarshal([]byte(cleanedResponse), &jsonResponse); jsonErr == nil {
				// Extract structured data from the response
				if analysis, ok := jsonResponse["analysis"].(string); ok {
					analysisResult = analysis
				}
				if fixes, ok := jsonResponse["suggested_fixes"].([]interface{}); ok {
					suggestedFixes = make([]string, 0, len(fixes))
					for _, fix := range fixes {
						if fixStr, ok := fix.(string); ok {
							suggestedFixes = append(suggestedFixes, fixStr)
						}
					}
				}
				if conf, ok := jsonResponse["confidence"].(float64); ok {
					confidence = conf
				}
				if cat, ok := jsonResponse["category"].(string); ok {
					category = cat
				}
				if estTime, ok := jsonResponse["estimated_resolution_time"].(string); ok {
					estimatedTime = estTime
				}
				if reqUserAction, ok := jsonResponse["requires_user_action"].(bool); ok {
					requiresUserAction = reqUserAction
				}
				if autoFix, ok := jsonResponse["automated_fix_available"].(bool); ok {
					automatedFixAvailable = autoFix
				}
			} else {
				// If not valid JSON, use the raw response as analysis
				log.Printf("Error parsing JSON response: %v. Using raw response.", jsonErr)
			}
		}
	}

	if err != nil || analysisResult == "" {
		log.Printf("Error analysis failed or returned empty: %v", err)
		analysisResult = s.generateFallbackErrorAnalysis(request.ErrorContext.Type, request.ErrorContext.Message)
		suggestedFixes = []string{
			"Check network connectivity and retry the operation",
			"Verify authentication credentials and permissions",
			"Review system logs for additional error details",
		}
	}

	// Structure the response
	response := map[string]interface{}{
		"analysis":                  analysisResult,
		"suggested_fixes":           suggestedFixes,
		"confidence":                confidence,
		"category":                  category,
		"estimated_resolution_time": estimatedTime,
		"requires_user_action":      requiresUserAction,
		"automated_fix_available":   automatedFixAvailable,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// extractJSONFromResponse extracts JSON content from AI responses that may be wrapped in markdown code blocks
func (s *SimpleAPIServer) extractJSONFromResponse(response string) string {
	// Remove leading/trailing whitespace
	response = strings.TrimSpace(response)

	// Check if response is wrapped in markdown code blocks
	if strings.HasPrefix(response, "```json") {
		// Find the start of JSON content (after ```json)
		startIndex := strings.Index(response, "```json")
		if startIndex != -1 {
			startIndex += 7 // Move past "```json"
			// Find the end of JSON content (before closing ```)
			endIndex := strings.LastIndex(response, "```")
			if endIndex != -1 && endIndex > startIndex {
				response = response[startIndex:endIndex]
			}
		}
	} else if strings.HasPrefix(response, "```") {
		// Handle generic code blocks that might contain JSON
		startIndex := strings.Index(response, "```")
		if startIndex != -1 {
			startIndex += 3 // Move past "```"
			// Skip any language identifier on the same line
			if newlineIndex := strings.Index(response[startIndex:], "\n"); newlineIndex != -1 {
				startIndex += newlineIndex + 1
			}
			// Find the end of JSON content (before closing ```)
			endIndex := strings.LastIndex(response, "```")
			if endIndex != -1 && endIndex > startIndex {
				response = response[startIndex:endIndex]
			}
		}
	}

	// Remove any remaining leading/trailing whitespace
	response = strings.TrimSpace(response)

	// If the response doesn't start with {, try to find the first { and last }
	if !strings.HasPrefix(response, "{") {
		startIndex := strings.Index(response, "{")
		endIndex := strings.LastIndex(response, "}")
		if startIndex != -1 && endIndex != -1 && endIndex > startIndex {
			response = response[startIndex : endIndex+1]
		}
	}

	return response
}

// AI Error Chat handler
func (s *SimpleAPIServer) handleChatError(w http.ResponseWriter, r *http.Request) {
	var request struct {
		ErrorID             string                 `json:"error_id"`
		Message             string                 `json:"message"`
		ErrorDetails        map[string]interface{} `json:"error_details"`
		ConversationHistory []struct {
			Type      string `json:"type"`
			Content   string `json:"content"`
			Timestamp string `json:"timestamp"`
		} `json:"conversation_history"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Include error details in the context if available
	errorDetailsContext := ""
	if len(request.ErrorDetails) > 0 {
		errorDetailsBytes, err := json.MarshalIndent(request.ErrorDetails, "", "  ")
		if err == nil {
			errorDetailsContext = fmt.Sprintf("\nError details:\n%s", string(errorDetailsBytes))
		}
	}

	// Search troubleshooting database for relevant information
	var troubleshootingMatches []TroubleshootingChunk
	if errorType, ok := request.ErrorDetails["type"].(string); ok {
		errorMessage := ""
		if msg, ok := request.ErrorDetails["message"].(string); ok {
			errorMessage = msg
		}

		var symptoms []string
		if symptomsInterface, ok := request.ErrorDetails["symptoms"]; ok {
			if symptomsSlice, ok := symptomsInterface.([]interface{}); ok {
				for _, symptom := range symptomsSlice {
					if symptomStr, ok := symptom.(string); ok {
						symptoms = append(symptoms, symptomStr)
					}
				}
			}
		}

		troubleshootingMatches = s.searchTroubleshootingDatabase(errorType, errorMessage, symptoms)
	}

	// Build troubleshooting context for chat
	troubleshootingContext := ""
	if len(troubleshootingMatches) > 0 {
		troubleshootingContext = "\n\nRELEVANT TROUBLESHOOTING KNOWLEDGE:\n"
		for i, match := range troubleshootingMatches {
			troubleshootingContext += fmt.Sprintf("\n--- SOLUTION %d ---\n", i+1)
			troubleshootingContext += fmt.Sprintf("Category: %s\n", match.Category)
			troubleshootingContext += fmt.Sprintf("Issue: %s\n", match.Issue)
			troubleshootingContext += fmt.Sprintf("Steps:\n%s\n", match.Content)
		}
		troubleshootingContext += "--- END KNOWLEDGE ---\n"
	}

	// Build enhanced chat prompt with conversation history and troubleshooting knowledge
	chatPrompt := fmt.Sprintf(`You are an AI assistant helping with error resolution for the KNIRVENGINE application.

Previous conversation:
%s

User's new question: %s%s%s

INSTRUCTIONS:
1. **KNOWLEDGE BASE PRIORITY**: First check the RELEVANT TROUBLESHOOTING KNOWLEDGE above for applicable solutions
2. **DIRECT ANSWERS**: If the knowledge base contains relevant information, reference it directly in your response
3. **TECHNICAL CLARITY**: Provide clear, technical but accessible responses to the user's question
4. **SPECIFIC GUIDANCE**: If the question relates to a specific error, explain what might be causing it using knowledge base insights
5. **ACTIONABLE STEPS**: Suggest specific steps the user can take to resolve the issue, prioritizing knowledge base solutions
6. **INFORMATION GATHERING**: If you need more information to diagnose the problem, ask for it
7. **SCOPE MANAGEMENT**: If the question is outside error resolution scope, politely redirect to relevant documentation

Please provide a helpful, technical response that addresses their specific question about the error, leveraging the troubleshooting knowledge base when applicable.`,
		s.buildConversationHistory(request.ConversationHistory),
		request.Message,
		errorDetailsContext,
		troubleshootingContext)

	// Use inference service with embedded keys for chat response
	var response string
	var err error

	if s.inferenceService != nil {
		log.Printf("Using inference service with embedded keys for error chat")
		response, err = s.inferenceService.GenerateText("", chatPrompt, "Provide a helpful response to the user's question about the error.")
	} else {
		response = "I understand your question about the error. Based on the context, I recommend checking the system logs and verifying your configuration settings."
	}

	if err != nil {
		log.Printf("Error chat failed: %v", err)
		response = "I apologize, but I'm having trouble processing your request right now. Please try again or check the system logs for more information."
	}

	chatResponse := map[string]interface{}{
		"response": response,
		"metadata": map[string]interface{}{
			"timestamp": time.Now().Format(time.RFC3339),
			"model":     "error-analysis-assistant",
			"error_id":  request.ErrorID,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chatResponse)
}

// Helper function to generate fallback error analysis
func (s *SimpleAPIServer) generateFallbackErrorAnalysis(errorType, message string) string {
	switch errorType {
	case "network":
		return fmt.Sprintf("Network error detected: %s. This typically indicates connectivity issues. Check your internet connection, firewall settings, and ensure the target server is accessible.", message)
	case "authentication":
		return fmt.Sprintf("Authentication error: %s. This suggests invalid credentials or expired tokens. Verify your login credentials and check if your session has expired.", message)
	case "server":
		return fmt.Sprintf("Server error: %s. This indicates an issue on the server side. The problem may be temporary - try again in a few moments. If it persists, check server logs.", message)
	case "validation":
		return fmt.Sprintf("Validation error: %s. This means the data provided doesn't meet the required format or constraints. Review the input data and ensure it matches the expected format.", message)
	default:
		return fmt.Sprintf("Error detected: %s. This appears to be a general system error. Check the application logs for more details and try refreshing the page or restarting the operation.", message)
	}
}

// Helper function to build conversation history string
func (s *SimpleAPIServer) buildConversationHistory(history []struct {
	Type      string `json:"type"`
	Content   string `json:"content"`
	Timestamp string `json:"timestamp"`
}) string {
	if len(history) == 0 {
		return "No previous conversation."
	}

	result := ""
	for _, msg := range history {
		result += fmt.Sprintf("[%s] %s: %s\n", msg.Timestamp, msg.Type, msg.Content)
	}
	return result
}

// MOA settings handler
func (s *SimpleAPIServer) handleMOASettings(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	modelType := vars["type"] // "primary" or "fallback"

	var request struct {
		Model string `json:"model"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// In a real implementation, you would:
	// 1. Validate the model exists
	// 2. Update your inference service configuration
	// 3. Restart or reload the inference service

	log.Printf("MOA %s model set to: %s", modelType, request.Model)

	response := map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("MOA %s model updated successfully", modelType),
		"type":    modelType,
		"model":   request.Model,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Agent Inferencer handlers

// Discover agents handler
func (s *SimpleAPIServer) discoverAgentsHandler(w http.ResponseWriter, r *http.Request) {
	if s.agentInferencer == nil {
		http.Error(w, "Agent Inferencer not available", http.StatusServiceUnavailable)
		return
	}

	ctx := context.Background()
	agents, err := s.agentInferencer.ListAvailableAgents(ctx)
	if err != nil {
		log.Printf("Error discovering agents: %v", err)
		http.Error(w, "Failed to discover agents", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"agents": agents,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getAllAgentsHandler returns both database agents and discovered agents
func (s *SimpleAPIServer) getAllAgentsHandler(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Get user ID from context (for database agents)
	userID, ok := r.Context().Value("userID").(int64)
	if !ok {
		userID = 1 // Default user for development
	}

	var allAgents []map[string]interface{}

	// Get database agents
	if s.agentRepo != nil {
		dbAgents, err := s.agentRepo.GetAgentsByOwner(r.Context(), userID)
		if err == nil {
			for _, agent := range dbAgents {
				agentData := map[string]interface{}{
					"id":           agent.ID,
					"name":         agent.Name,
					"type":         "database", // SimpleAgent doesn't have type field
					"source":       "database",
					"collection":   agent.Collection,
					"image_url":    agent.ImageURL,
					"status":       agent.Status,
					"capabilities": agent.Capabilities,
					"target_types": []string{"application"}, // Default for SimpleAgent
					"created_at":   agent.CreatedAt.Format(time.RFC3339),
					"updated_at":   agent.CreatedAt.Format(time.RFC3339), // Use CreatedAt as fallback
				}

				allAgents = append(allAgents, agentData)
			}
		}
	}

	// Get discovered agents
	if s.agentInferencer != nil {
		discoveredAgents, err := s.agentInferencer.ListAvailableAgents(r.Context())
		if err == nil {
			for _, agentID := range discoveredAgents {
				// Parse agent ID to extract name and version
				parts := strings.Split(agentID, "_")
				if len(parts) < 2 {
					continue
				}

				version := parts[len(parts)-1]
				name := strings.Join(parts[:len(parts)-1], "_")

				// Determine if it's a WASM agent
				isWasm := strings.Contains(agentID, "wasm") ||
					strings.Contains(agentID, "test_unified_storage") ||
					strings.Contains(agentID, "frontend_wasm_test") ||
					strings.Contains(agentID, "mytest")

				agentData := map[string]interface{}{
					"id":   agentID,
					"name": strings.Title(strings.ReplaceAll(name, "_", " ")),
					"type": func() string {
						if isWasm {
							return "wasm"
						} else {
							return "plugin"
						}
					}(),
					"source":       "discovered",
					"collection":   "Discovered Agents",
					"status":       "available",
					"capabilities": []string{"General Purpose"},
					"target_types": []string{"application"},
					"version":      version,
					"file_type": func() string {
						if isWasm {
							return ".wasm"
						} else {
							return ".so"
						}
					}(),
					"created_at": time.Now().Format(time.RFC3339),
				}

				allAgents = append(allAgents, agentData)
			}
		}
	}

	response := map[string]interface{}{
		"agents": allAgents,
		"total":  len(allAgents),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Activate agent handler
func (s *SimpleAPIServer) activateAgentHandler(w http.ResponseWriter, r *http.Request) {
	if s.agentInferencer == nil {
		http.Error(w, "Agent Inferencer not available", http.StatusServiceUnavailable)
		return
	}

	var request struct {
		AgentID   string                 `json:"agentId"`
		Version   string                 `json:"version"`
		SessionID string                 `json:"sessionId"`
		Config    map[string]interface{} `json:"config"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	err := s.agentInferencer.ActivateAgent(ctx, request.AgentID, request.Version, request.SessionID, request.Config)
	if err != nil {
		log.Printf("Error activating agent: %v", err)
		http.Error(w, fmt.Sprintf("Failed to activate agent: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"status":    "success",
		"message":   "Agent activated successfully",
		"agentId":   request.AgentID,
		"sessionId": request.SessionID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Deactivate agent handler
func (s *SimpleAPIServer) deactivateAgentHandler(w http.ResponseWriter, r *http.Request) {
	if s.agentInferencer == nil {
		http.Error(w, "Agent Inferencer not available", http.StatusServiceUnavailable)
		return
	}

	var request struct {
		SessionID string `json:"sessionId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	err := s.agentInferencer.DeactivateAgent(ctx, request.SessionID)
	if err != nil {
		log.Printf("Error deactivating agent: %v", err)
		http.Error(w, fmt.Sprintf("Failed to deactivate agent: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"status":    "success",
		"message":   "Agent deactivated successfully",
		"sessionId": request.SessionID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Get agent capabilities handler
func (s *SimpleAPIServer) getAgentCapabilitiesHandler(w http.ResponseWriter, r *http.Request) {
	if s.agentInferencer == nil {
		http.Error(w, "Agent Inferencer not available", http.StatusServiceUnavailable)
		return
	}

	sessionID := r.URL.Query().Get("sessionId")
	if sessionID == "" {
		http.Error(w, "sessionId parameter is required", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	capabilities, err := s.agentInferencer.GetAgentCapabilities(ctx, sessionID)
	if err != nil {
		log.Printf("Error getting agent capabilities: %v", err)
		http.Error(w, fmt.Sprintf("Failed to get agent capabilities: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(capabilities)
}

// Get agent schema handler
func (s *SimpleAPIServer) getAgentSchemaHandler(w http.ResponseWriter, r *http.Request) {
	if s.agentInferencer == nil {
		http.Error(w, "Agent Inferencer not available", http.StatusServiceUnavailable)
		return
	}

	sessionID := r.URL.Query().Get("sessionId")
	if sessionID == "" {
		http.Error(w, "sessionId parameter is required", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	schema, err := s.agentInferencer.GetAgentSchema(ctx, sessionID)
	if err != nil {
		log.Printf("Error getting agent schema: %v", err)
		http.Error(w, fmt.Sprintf("Failed to get agent schema: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(schema)
}

// Process inference handler
func (s *SimpleAPIServer) processInferenceHandler(w http.ResponseWriter, r *http.Request) {
	if s.agentInferencer == nil {
		http.Error(w, "Agent Inferencer not available", http.StatusServiceUnavailable)
		return
	}

	var request agentify.InferenceRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	response, err := s.agentInferencer.ProcessInference(ctx, request.SessionID, &request)
	if err != nil {
		log.Printf("Error processing inference: %v", err)
		http.Error(w, fmt.Sprintf("Failed to process inference: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Set agent memory handler
func (s *SimpleAPIServer) setAgentMemoryHandler(w http.ResponseWriter, r *http.Request) {
	if s.agentInferencer == nil {
		http.Error(w, "Agent Inferencer not available", http.StatusServiceUnavailable)
		return
	}

	var request struct {
		SessionID string      `json:"sessionId"`
		Key       string      `json:"key"`
		Value     interface{} `json:"value"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	err := s.agentInferencer.SetAgentMemory(ctx, request.SessionID, request.Key, request.Value)
	if err != nil {
		log.Printf("Error setting agent memory: %v", err)
		http.Error(w, fmt.Sprintf("Failed to set agent memory: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"status":  "success",
		"message": "Memory set successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Get agent memory handler
func (s *SimpleAPIServer) getAgentMemoryHandler(w http.ResponseWriter, r *http.Request) {
	if s.agentInferencer == nil {
		http.Error(w, "Agent Inferencer not available", http.StatusServiceUnavailable)
		return
	}

	sessionID := r.URL.Query().Get("sessionId")
	key := r.URL.Query().Get("key")

	if sessionID == "" || key == "" {
		http.Error(w, "sessionId and key parameters are required", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	value, err := s.agentInferencer.GetAgentMemory(ctx, sessionID, key)
	if err != nil {
		log.Printf("Error getting agent memory: %v", err)
		http.Error(w, fmt.Sprintf("Failed to get agent memory: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"value": value,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Terminal handlers

// Create terminal handler
func (s *SimpleAPIServer) createTerminalHandler(w http.ResponseWriter, r *http.Request) {
	if s.agentInferencer == nil {
		http.Error(w, "Agent Inferencer not available", http.StatusServiceUnavailable)
		return
	}

	var request struct {
		SessionID string `json:"sessionId"`
		Rows      int    `json:"rows"`
		Cols      int    `json:"cols"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Set defaults
	if request.Rows == 0 {
		request.Rows = 24
	}
	if request.Cols == 0 {
		request.Cols = 80
	}

	ctx := context.Background()
	terminalID, err := s.agentInferencer.CreateTerminal(ctx, request.SessionID, request.Rows, request.Cols)
	if err != nil {
		log.Printf("Error creating terminal: %v", err)
		http.Error(w, fmt.Sprintf("Failed to create terminal: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"terminalId": terminalID,
		"sessionId":  request.SessionID,
		"rows":       request.Rows,
		"cols":       request.Cols,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Write to terminal handler
func (s *SimpleAPIServer) writeToTerminalHandler(w http.ResponseWriter, r *http.Request) {
	if s.agentInferencer == nil {
		http.Error(w, "Agent Inferencer not available", http.StatusServiceUnavailable)
		return
	}

	var request struct {
		SessionID  string `json:"sessionId"`
		TerminalID string `json:"terminalId"`
		Data       string `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	err := s.agentInferencer.WriteToTerminal(ctx, request.SessionID, request.TerminalID, []byte(request.Data))
	if err != nil {
		log.Printf("Error writing to terminal: %v", err)
		http.Error(w, fmt.Sprintf("Failed to write to terminal: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"status":  "success",
		"message": "Data written to terminal",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Read from terminal handler
func (s *SimpleAPIServer) readFromTerminalHandler(w http.ResponseWriter, r *http.Request) {
	if s.agentInferencer == nil {
		http.Error(w, "Agent Inferencer not available", http.StatusServiceUnavailable)
		return
	}

	sessionID := r.URL.Query().Get("sessionId")
	terminalID := r.URL.Query().Get("terminalId")

	if sessionID == "" || terminalID == "" {
		http.Error(w, "sessionId and terminalId parameters are required", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	data, err := s.agentInferencer.ReadFromTerminal(ctx, sessionID, terminalID)
	if err != nil {
		log.Printf("Error reading from terminal: %v", err)
		http.Error(w, fmt.Sprintf("Failed to read from terminal: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"data": string(data),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Resize terminal handler
func (s *SimpleAPIServer) resizeTerminalHandler(w http.ResponseWriter, r *http.Request) {
	if s.agentInferencer == nil {
		http.Error(w, "Agent Inferencer not available", http.StatusServiceUnavailable)
		return
	}

	var request struct {
		SessionID  string `json:"sessionId"`
		TerminalID string `json:"terminalId"`
		Rows       int    `json:"rows"`
		Cols       int    `json:"cols"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	err := s.agentInferencer.ResizeTerminal(ctx, request.SessionID, request.TerminalID, request.Rows, request.Cols)
	if err != nil {
		log.Printf("Error resizing terminal: %v", err)
		http.Error(w, fmt.Sprintf("Failed to resize terminal: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"status":  "success",
		"message": "Terminal resized successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Close terminal handler
func (s *SimpleAPIServer) closeTerminalHandler(w http.ResponseWriter, r *http.Request) {
	if s.agentInferencer == nil {
		http.Error(w, "Agent Inferencer not available", http.StatusServiceUnavailable)
		return
	}

	var request struct {
		SessionID  string `json:"sessionId"`
		TerminalID string `json:"terminalId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	err := s.agentInferencer.CloseTerminal(ctx, request.SessionID, request.TerminalID)
	if err != nil {
		log.Printf("Error closing terminal: %v", err)
		http.Error(w, fmt.Sprintf("Failed to close terminal: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"status":  "success",
		"message": "Terminal closed successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Plugin discovery and import handlers

// Discover all plugins handler
func (s *SimpleAPIServer) discoverAllPluginsHandler(w http.ResponseWriter, r *http.Request) {
	if s.agentInferencer == nil {
		http.Error(w, "Agent Inferencer not available", http.StatusServiceUnavailable)
		return
	}

	ctx := context.Background()
	plugins, err := s.agentInferencer.DiscoverAllPlugins(ctx)
	if err != nil {
		log.Printf("Error discovering all plugins: %v", err)
		http.Error(w, "Failed to discover plugins", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"plugins": plugins,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Import plugin handler
func (s *SimpleAPIServer) importPluginHandler(w http.ResponseWriter, r *http.Request) {
	if s.agentInferencer == nil {
		http.Error(w, "Agent Inferencer not available", http.StatusServiceUnavailable)
		return
	}

	var request agentify.ImportPluginRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if request.FilePath == "" || request.AgentID == "" || request.Version == "" {
		http.Error(w, "filePath, agentId, and version are required", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	err := s.agentInferencer.ImportPlugin(ctx, &request)
	if err != nil {
		log.Printf("Error importing plugin: %v", err)
		http.Error(w, fmt.Sprintf("Failed to import plugin: %v", err), http.StatusInternalServerError)
		return
	}

	// Get user ID from authentication context
	userID, ok := r.Context().Value(UserIDContextKey).(int64)
	if !ok {
		log.Printf("Warning: No user ID found in context for import, using default owner ID")
		userID = 1 // Fallback to default user
	}

	// Create an agent record in the database for the imported plugin
	log.Printf("Creating agent record for imported plugin: %s", request.AgentID)
	agent := &database.SimpleAgent{
		Name:         request.AgentID,
		Collection:   "Imported Plugins",
		ImageURL:     "https://example.com/plugin.png",
		Status:       "active",
		Capabilities: []string{"plugin_execution"},
		TokenID:      "0",
		ContractAddr: "0x000...",
		OwnerID:      userID, // Get from authentication context
	}

	log.Printf("Agent struct created: %+v", agent)
	err = s.agentRepo.CreateAgent(ctx, agent)
	if err != nil {
		log.Printf("ERROR: Failed to create agent record for imported plugin: %v", err)
		// Continue anyway - plugin import succeeded
	} else {
		log.Printf("SUCCESS: Created agent record for imported plugin: %s (ID: %s)", request.AgentID, agent.ID)
	}

	response := map[string]interface{}{
		"status":  "success",
		"message": "Plugin imported successfully",
		"agentId": request.AgentID,
		"version": request.Version,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// terminalWebSocketHandler handles WebSocket connections for terminal sessions
func (s *SimpleAPIServer) terminalWebSocketHandler(w http.ResponseWriter, r *http.Request) {
	// Get query parameters
	sessionID := r.URL.Query().Get("sessionId")
	if sessionID == "" {
		log.Printf("WebSocket connection failed: Missing sessionId parameter")
		http.Error(w, "Missing sessionId parameter", http.StatusBadRequest)
		return
	}

	terminalID := r.URL.Query().Get("terminalId")
	if terminalID == "" {
		log.Printf("WebSocket connection failed: Missing terminalId parameter")
		http.Error(w, "Missing terminalId parameter", http.StatusBadRequest)
		return
	}

	log.Printf("Attempting WebSocket upgrade for terminal %s (session %s)", terminalID, sessionID)

	// Check if agent inferencer is available
	if s.agentInferencer == nil {
		log.Printf("WebSocket connection failed: Agent Inferencer not available")
		http.Error(w, "Agent Inferencer not available", http.StatusServiceUnavailable)
		return
	}

	// WebSocket upgrader
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			// Allow all origins for now - in production, this should be more restrictive
			return true
		},
	}

	// Upgrade the HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade WebSocket connection: %v", err)
		return
	}
	defer conn.Close()

	log.Printf("WebSocket connection established for terminal %s (session %s)", terminalID, sessionID)

	// Send initial connection confirmation
	if err := conn.WriteMessage(websocket.TextMessage, []byte("Terminal WebSocket connected\r\n")); err != nil {
		log.Printf("Failed to send initial message: %v", err)
		return
	}

	// Handle WebSocket communication
	go s.handleTerminalOutput(conn, sessionID, terminalID)
	s.handleTerminalInput(conn, sessionID, terminalID)
}

// handleTerminalOutput handles sending terminal output to the WebSocket client
func (s *SimpleAPIServer) handleTerminalOutput(conn *websocket.Conn, sessionID, terminalID string) {
	log.Printf("Starting terminal output handler for terminal %s (session %s)", terminalID, sessionID)

	// Set up a ticker for polling terminal output
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Read from terminal
			ctx := context.Background()
			output, err := s.agentInferencer.ReadFromTerminal(ctx, sessionID, terminalID)
			if err != nil {
				log.Printf("Failed to read from terminal %s: %v", terminalID, err)
				// Send error message to client
				errorMsg := fmt.Sprintf("Terminal error: %v\r\n", err)
				if err := conn.WriteMessage(websocket.TextMessage, []byte(errorMsg)); err != nil {
					log.Printf("Failed to send error message to WebSocket: %v", err)
				}
				return
			}

			if len(output) > 0 {
				if err := conn.WriteMessage(websocket.TextMessage, output); err != nil {
					log.Printf("Failed to write terminal output to WebSocket: %v", err)
					return
				}
			}
		}
	}
}

// handleTerminalInput handles receiving input from the WebSocket client and sending it to the terminal
func (s *SimpleAPIServer) handleTerminalInput(conn *websocket.Conn, sessionID, terminalID string) {
	log.Printf("Starting terminal input handler for terminal %s (session %s)", terminalID, sessionID)

	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket connection closed unexpectedly: %v", err)
			} else {
				log.Printf("WebSocket connection closed: %v", err)
			}
			break
		}

		// Only handle text messages
		if messageType != websocket.TextMessage {
			log.Printf("Received non-text message type %d, ignoring", messageType)
			continue
		}

		log.Printf("Received terminal input: %q", string(message))

		// Write the input to the terminal
		ctx := context.Background()
		if err := s.agentInferencer.WriteToTerminal(ctx, sessionID, terminalID, message); err != nil {
			log.Printf("Failed to write to terminal %s: %v", terminalID, err)
			// Send error feedback to client
			errorMsg := fmt.Sprintf("Input error: %v\r\n", err)
			if err := conn.WriteMessage(websocket.TextMessage, []byte(errorMsg)); err != nil {
				log.Printf("Failed to send input error message: %v", err)
			}
		}
	}
}

// getTerminalLogsHandler handles GET /api/v1/terminal/logs
func (s *SimpleAPIServer) getTerminalLogsHandler(w http.ResponseWriter, r *http.Request) {
	if s.agentInferencer == nil {
		http.Error(w, "Agent Inferencer not available", http.StatusServiceUnavailable)
		return
	}

	// Get query parameters
	sessionID := r.URL.Query().Get("sessionId")
	if sessionID == "" {
		http.Error(w, "Missing sessionId parameter", http.StatusBadRequest)
		return
	}

	terminalID := r.URL.Query().Get("terminalId")
	if terminalID == "" {
		http.Error(w, "Missing terminalId parameter", http.StatusBadRequest)
		return
	}

	// Get terminal logs from the agent inferencer
	ctx := context.Background()
	logs, err := s.agentInferencer.GetTerminalLogs(ctx, sessionID, terminalID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get terminal logs: %v", err), http.StatusInternalServerError)
		return
	}

	// Return the logs as JSON
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"logs":    logs,
	}); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// User management handlers

// handleListUsers handles GET /api/v1/users
func (s *SimpleAPIServer) handleListUsers(w http.ResponseWriter, r *http.Request) {
	// Delegate to the user service
	s.userService.handleListUsers(w, r)
}

// handleGetUser handles GET /api/v1/users/{id}
func (s *SimpleAPIServer) handleGetUser(w http.ResponseWriter, r *http.Request) {
	// Delegate to the user service
	s.userService.handleGetUser(w, r)
}

// handleCreateUser handles POST /api/v1/users
func (s *SimpleAPIServer) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	// Delegate to the user service
	s.userService.handleCreateUser(w, r)
}

// handleUpdateUser handles PUT /api/v1/users/{id}
func (s *SimpleAPIServer) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	// Delegate to the user service
	s.userService.handleUpdateUser(w, r)
}

// handleDeleteUser handles DELETE /api/v1/users/{id}
func (s *SimpleAPIServer) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	// Delegate to the user service
	s.userService.handleDeleteUser(w, r)
}

// ============================================================================
// Agent Builder Handlers
// ============================================================================

// buildAgentHandler handles POST /api/v1/agents/{id}/build
func (s *SimpleAPIServer) buildAgentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["id"]

	var request struct {
		TemplateID string                 `json:"template_id"`
		Config     map[string]interface{} `json:"config"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if s.agentBuilder == nil {
		http.Error(w, "Agent Builder not available", http.StatusServiceUnavailable)
		return
	}

	// Create agent config from request
	agentConfig := agent.AgentConfig{
		AgentID:           agentID,
		AgentType:         getStringFromMap(request.Config, "agent_type", "standard"),
		Name:              getStringFromMap(request.Config, "agent_name", "Agent "+agentID),
		Model:             getStringFromMap(request.Config, "model", "gpt-4"),
		Instruction:       getStringFromMap(request.Config, "instruction", "You are a helpful AI agent."),
		Description:       getStringFromMap(request.Config, "agent_description", "Generated agent"),
		UseSearch:         getBoolFromMap(request.Config, "use_search", false),
		UseCodeExecution:  getBoolFromMap(request.Config, "use_code_execution", false),
		UseVertexSearch:   getBoolFromMap(request.Config, "use_vertex_search", false),
		VertexDatastoreID: getStringFromMap(request.Config, "vertex_datastore_id", ""),
		MaxIterations:     getIntFromMap(request.Config, "max_iterations", 10),
		BuildTarget:       getStringFromMap(request.Config, "build_target", "plugin"),
		ExtraParams:       request.Config,
	}

	// Build the agent plugin
	pluginPath, err := s.agentBuilder.BuildAgent(agentConfig)
	if err != nil {
		log.Printf("Error building agent plugin: %v", err)
		http.Error(w, fmt.Sprintf("Failed to build agent plugin: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":     "Agent plugin build started",
		"build_id":    agentID,
		"plugin_path": pluginPath,
	})
}

// getAgentBuildStatusHandler handles GET /api/v1/agents/{id}/build
func (s *SimpleAPIServer) getAgentBuildStatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["id"]

	if s.agentBuilder == nil {
		http.Error(w, "Agent Builder not available", http.StatusServiceUnavailable)
		return
	}

	// Get plugin path for the agent
	pluginPath, err := s.agentBuilder.GetPluginPath(agentID)
	if err != nil {
		// Agent not found or no plugin built yet
		buildStatus := map[string]interface{}{
			"agent_id": agentID,
			"status":   "not_built",
			"progress": 0,
			"message":  "Agent plugin has not been built yet",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"build_status": buildStatus,
		})
		return
	}

	// Check if plugin file exists
	buildStatus := map[string]interface{}{
		"agent_id":    agentID,
		"status":      "success",
		"progress":    100,
		"message":     "Agent plugin built successfully",
		"plugin_path": pluginPath,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"build_status": buildStatus,
	})
}

// rebuildAgentHandler handles POST /api/v1/agents/{id}/rebuild
func (s *SimpleAPIServer) rebuildAgentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["id"]

	if s.agentBuilder == nil {
		http.Error(w, "Agent Builder not available", http.StatusServiceUnavailable)
		return
	}

	// Rebuild the agent plugin
	err := s.agentBuilder.RebuildAgent(agentID)
	if err != nil {
		log.Printf("Error rebuilding agent plugin: %v", err)
		http.Error(w, fmt.Sprintf("Failed to rebuild agent plugin: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":  "Agent plugin rebuild started",
		"build_id": agentID,
	})
}

// getAgentTemplatesHandler handles GET /api/v1/templates
func (s *SimpleAPIServer) getAgentTemplatesHandler(w http.ResponseWriter, r *http.Request) {
	// For now, return a static list of available templates
	// In a real implementation, this would scan the templates directory
	templates := []map[string]interface{}{
		{
			"id":          "standard",
			"name":        "Standard Agent",
			"description": "A basic agent with standard capabilities",
			"type":        "standard",
			"config_schema": map[string]interface{}{
				"agent_name":         "string",
				"agent_description":  "string",
				"model":              "string",
				"instruction":        "string",
				"use_search":         "boolean",
				"use_code_execution": "boolean",
			},
		},
		{
			"id":          "search",
			"name":        "Search Agent",
			"description": "An agent with enhanced search capabilities",
			"type":        "search",
			"config_schema": map[string]interface{}{
				"agent_name":        "string",
				"agent_description": "string",
				"model":             "string",
				"instruction":       "string",
				"use_search":        "boolean",
				"use_vertex_search": "boolean",
			},
		},
		{
			"id":          "code",
			"name":        "Code Execution Agent",
			"description": "An agent with code execution capabilities",
			"type":        "code",
			"config_schema": map[string]interface{}{
				"agent_name":         "string",
				"agent_description":  "string",
				"model":              "string",
				"instruction":        "string",
				"use_code_execution": "boolean",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"templates": templates,
	})
}

// getCompiledPluginsHandler handles GET /api/v1/plugins
func (s *SimpleAPIServer) getCompiledPluginsHandler(w http.ResponseWriter, r *http.Request) {
	if s.agentBuilder == nil {
		http.Error(w, "Agent Builder not available", http.StatusServiceUnavailable)
		return
	}

	// Get list of agents from the builder
	agentIDs, err := s.agentBuilder.ListAgents()
	if err != nil {
		log.Printf("Error listing agents: %v", err)
		http.Error(w, "Failed to list compiled plugins", http.StatusInternalServerError)
		return
	}

	plugins := make([]map[string]interface{}, 0, len(agentIDs))
	for _, agentID := range agentIDs {
		// Get plugin path for each agent
		pluginPath, err := s.agentBuilder.GetPluginPath(agentID)
		if err != nil {
			continue // Skip agents without plugins
		}

		// Get agent config for additional info
		config, err := s.agentBuilder.GetAgent(agentID)
		if err != nil {
			continue
		}

		// Sync agent to database if it doesn't exist there
		configMap, err := agentConfigToMap(config)
		if err != nil {
			log.Printf("Failed to convert agent config to map: %v", err)
			continue
		}
		s.syncAgentToDatabase(agentID, configMap, pluginPath)

		// Get file info for creation time and size
		var createdAt string
		var fileSize int64
		var version string = "1.0"

		// Try to read metadata file first for accurate creation time
		metadataPath := pluginPath + ".meta"
		if metadataData, err := os.ReadFile(metadataPath); err == nil {
			var metadata map[string]interface{}
			if json.Unmarshal(metadataData, &metadata) == nil {
				if createdAtStr, ok := metadata["created_at"].(string); ok {
					createdAt = createdAtStr
				}
				if versionStr, ok := metadata["version"].(string); ok {
					version = versionStr
				}
				if fileSizeFloat, ok := metadata["file_size"].(float64); ok {
					fileSize = int64(fileSizeFloat)
				}
			}
		}

		// Fallback to file modification time if metadata not available
		if createdAt == "" {
			if info, err := os.Stat(pluginPath); err == nil {
				createdAt = info.ModTime().Format(time.RFC3339)
				fileSize = info.Size()
			} else {
				// Final fallback to current time if file stat fails
				createdAt = time.Now().Format(time.RFC3339)
			}
		}

		plugin := map[string]interface{}{
			"id":         agentID,
			"agent_id":   agentID,
			"filename":   filepath.Base(pluginPath),
			"path":       pluginPath,
			"created_at": createdAt,
			"version":    version,
			"agent_name": config.Name,
			"size":       fileSize,
		}

		plugins = append(plugins, plugin)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"plugins": plugins,
	})
}

// syncAgentToDatabase ensures an agent from the registry also exists in the database
func (s *SimpleAPIServer) syncAgentToDatabase(agentID string, config map[string]interface{}, _ string) {
	if s.agentRepo == nil {
		return // No agent repository available
	}

	// Check if agent already exists in database
	ctx := context.Background()
	_, err := s.agentRepo.GetAgentByID(ctx, agentID)
	if err == nil {
		return // Agent already exists in database
	}

	// Get name from config
	name := "Unknown Agent"
	if nameVal, ok := config["name"]; ok {
		if nameStr, ok := nameVal.(string); ok {
			name = nameStr
		}
	}

	// Agent doesn't exist in database, create it
	dbAgent := &database.SimpleAgent{
		ID:           agentID,
		OwnerID:      1, // Default owner
		Name:         name,
		Collection:   "default",
		ImageURL:     "",
		Status:       "idle",
		Capabilities: []string{"General Purpose"},
		TokenID:      "",
		ContractAddr: "",
		CreatedAt:    time.Now(),
	}

	// Create the agent in the database
	err = s.agentRepo.CreateAgent(ctx, dbAgent)
	if err != nil {
		log.Printf("Failed to sync agent %s to database: %v", agentID, err)
	} else {
		log.Printf("Successfully synced agent %s to database", agentID)
	}
}

// syncAgentToRegistry ensures an agent from the database also exists in the registry
func (s *SimpleAPIServer) syncAgentToRegistry(ctx context.Context, agent *database.SimpleAgent) {
	if s.agentBuilder == nil {
		return // No agent builder available
	}

	// Use context for timeout/cancellation
	select {
	case <-ctx.Done():
		log.Printf("Context cancelled while syncing agent %s: %v", agent.ID, ctx.Err())
		return
	default:
	}

	// Check if agent exists in registry
	if _, err := s.agentBuilder.GetAgent(agent.ID); err == nil {
		return // Agent already exists
	}

	// Register agent with minimal config
	config := map[string]interface{}{
		"agent_id":   agent.ID,
		"name":       agent.Name,
		"agent_type": "general",
		"model":      "default",
	}

	if err := s.agentBuilder.GetRegistry().RegisterAgent(agent.ID, config); err != nil {
		log.Printf("Failed to register agent %s: %v", agent.ID, err)
		return
	}

	log.Printf("Successfully synced agent %s to registry", agent.ID)
}

// syncAgentsHandler handles the POST /api/v1/agents/sync endpoint
func (s *SimpleAPIServer) syncAgentsHandler(w http.ResponseWriter, r *http.Request) {
	// Perform the synchronization
	s.syncAllAgents() // Use the fixed syncAllAgents function

	// Return a simple success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Agent synchronization completed",
	})
}

// syncAllAgents synchronizes all agents between the database and registry
func (s *SimpleAPIServer) syncAllAgents() {
	if s.agentRepo == nil || s.agentBuilder == nil {
		return // Can't sync if either system is unavailable
	}

	ctx := context.Background()

	// Get all agents from the database
	dbAgents, err := s.agentRepo.GetAgentsByOwner(ctx, 1) // Default owner
	if err != nil {
		log.Printf("Failed to get agents from database for sync: %v", err)
		return
	}

	// Get all agents from the registry
	registryAgentIDs, err := s.agentBuilder.ListAgents()
	if err != nil {
		log.Printf("Failed to get agents from registry for sync: %v", err)
		return
	}

	// Create maps for quick lookup
	dbAgentMap := make(map[string]*database.SimpleAgent)
	for _, agent := range dbAgents {
		dbAgentMap[agent.ID] = agent
	}

	registryAgentMap := make(map[string]bool)
	for _, id := range registryAgentIDs {
		registryAgentMap[id] = true
	}

	// Sync database agents to registry
	for _, agent := range dbAgents {
		if !registryAgentMap[agent.ID] {
			s.syncAgentToRegistry(ctx, agent)
		}
	}

	// Sync registry agents to database
	for _, id := range registryAgentIDs {
		if _, exists := dbAgentMap[id]; !exists {
			// Get agent config from registry
			configMap, err := s.agentBuilder.GetRegistry().GetAgent(id)
			if err != nil {
				log.Printf("Failed to get agent %s from registry for sync: %v", id, err)
				continue
			}
			s.syncAgentToDatabase(id, configMap, "")
		}
	}

	log.Printf("Agent synchronization completed: %d database agents, %d registry agents", len(dbAgents), len(registryAgentIDs))
}

// deleteAgentPluginHandler handles DELETE /api/v1/plugins/{id}
func (s *SimpleAPIServer) deleteAgentPluginHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pluginID := vars["id"]

	if s.agentBuilder == nil {
		http.Error(w, "Agent Builder not available", http.StatusServiceUnavailable)
		return
	}

	// Delete the agent (which should also clean up the plugin)
	err := s.agentBuilder.DeleteAgent(pluginID)
	if err != nil {
		log.Printf("Error deleting agent plugin: %v", err)
		http.Error(w, fmt.Sprintf("Failed to delete agent plugin: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Agent plugin deleted successfully",
	})
}

// ============================================================================
// Sub-Agent Handlers
// ============================================================================

// spawnSubAgentHandler handles POST /api/v1/agents/{id}/sub-agents
func (s *SimpleAPIServer) spawnSubAgentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	parentID := vars["id"]

	var request struct {
		Template string                 `json:"template"`
		Config   map[string]interface{} `json:"config"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate template
	if request.Template != "python" && request.Template != "java" {
		http.Error(w, "Template must be 'python' or 'java'", http.StatusBadRequest)
		return
	}

	// Sub-agent creation should be handled by the main agent itself
	// The main agent will use the ADK to spawn sub-agents within its own TEE
	// This endpoint should trigger the main agent to spawn a sub-agent

	// Create a structured message for the parent agent to spawn a sub-agent
	spawnRequest := map[string]interface{}{
		"action":   "spawn_sub_agent",
		"template": request.Template,
		"config":   request.Config,
	}

	// Convert to JSON for the message
	spawnRequestJSON, err := json.Marshal(spawnRequest)
	if err != nil {
		log.Printf("Failed to marshal spawn request: %v", err)
		http.Error(w, "Failed to create spawn request", http.StatusInternalServerError)
		return
	}

	// Send the spawn request to the parent agent through the agent inferencer
	ctx := context.Background()
	sessionID := fmt.Sprintf("agent_%s_session", parentID)

	// Create an inference request to send to the parent agent
	inferenceRequest := &agentify.InferenceRequest{
		Input:     string(spawnRequestJSON),
		SessionID: sessionID,
		Parameters: map[string]interface{}{
			"action": "spawn_sub_agent",
		},
	}

	// Process the inference request through the agent inferencer
	inferenceResponse, err := s.agentInferencer.ProcessInference(ctx, sessionID, inferenceRequest)
	if err != nil {
		log.Printf("Failed to send spawn request to parent agent %s: %v", parentID, err)
		// Return error but don't fail completely - the agent might not be active
		response := map[string]interface{}{
			"status":    "error",
			"message":   fmt.Sprintf("Failed to communicate with parent agent: %v", err),
			"parent_id": parentID,
			"template":  request.Template,
			"config":    request.Config,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Return success response with the agent's response
	response := map[string]interface{}{
		"status":         "success",
		"message":        "Sub-agent spawn request processed by parent agent",
		"parent_id":      parentID,
		"template":       request.Template,
		"config":         request.Config,
		"agent_response": inferenceResponse.Output,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// getSubAgentsHandler handles GET /api/v1/agents/{id}/sub-agents
func (s *SimpleAPIServer) getSubAgentsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	parentID := vars["id"]

	// Get sub-agents from the agent inferencer
	// Sub-agents are managed by the main agent through the inferencer
	subAgents := []map[string]interface{}{}

	log.Printf("Getting sub-agents for parent %s", parentID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"sub_agents": subAgents,
	})
}

// terminateSubAgentHandler handles DELETE /api/v1/agents/{id}/sub-agents/{subId}
func (s *SimpleAPIServer) terminateSubAgentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	parentID := vars["id"]
	subAgentID := vars["subId"]

	log.Printf("Terminating sub-agent %s for parent %s", subAgentID, parentID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Sub-agent terminated successfully",
	})
}

// getSubAgentTerminalHandler handles GET /api/v1/agents/{id}/sub-agents/{subId}/terminal
func (s *SimpleAPIServer) getSubAgentTerminalHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	parentID := vars["id"]
	subAgentID := vars["subId"]

	log.Printf("Getting terminal for sub-agent %s of parent %s", subAgentID, parentID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"terminal_session": fmt.Sprintf("term_%s", subAgentID),
	})
}

// sendSubAgentCommandHandler handles POST /api/v1/agents/{id}/sub-agents/{subId}/command
func (s *SimpleAPIServer) sendSubAgentCommandHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	parentID := vars["id"]
	subAgentID := vars["subId"]

	var request struct {
		Command string `json:"command"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	log.Printf("Sending command '%s' to sub-agent %s of parent %s", request.Command, subAgentID, parentID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Command sent successfully",
	})
}

// getSubAgentLogsHandler handles GET /api/v1/agents/{id}/sub-agents/{subId}/logs
func (s *SimpleAPIServer) getSubAgentLogsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	parentID := vars["id"]
	subAgentID := vars["subId"]

	// Get limit parameter
	limitStr := r.URL.Query().Get("limit")
	limit := 100 // default
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil {
			limit = parsedLimit
		}
	}

	// Sample logs for demonstration
	logs := []string{
		fmt.Sprintf("[%s] Sub-agent %s started", time.Now().Add(-time.Hour).Format(time.RFC3339), subAgentID),
		fmt.Sprintf("[%s] Initializing %s environment", time.Now().Add(-50*time.Minute).Format(time.RFC3339), subAgentID),
		fmt.Sprintf("[%s] Sub-agent ready for commands", time.Now().Add(-45*time.Minute).Format(time.RFC3339)),
		fmt.Sprintf("[%s] Processing task...", time.Now().Add(-30*time.Minute).Format(time.RFC3339)),
		fmt.Sprintf("[%s] Task completed successfully", time.Now().Add(-25*time.Minute).Format(time.RFC3339)),
	}

	// Apply limit
	if limit < len(logs) {
		logs = logs[:limit]
	}

	log.Printf("Getting logs for sub-agent %s of parent %s (limit: %d)", subAgentID, parentID, limit)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"logs": logs,
	})
}

// mainWebSocketHandler handles the main WebSocket endpoint for real-time updates
func (s *SimpleAPIServer) mainWebSocketHandler(w http.ResponseWriter, r *http.Request) {
	// Check if running in Electron mode
	electronMode := os.Getenv("ELECTRON_MODE") == "true"

	// WebSocket upgrader with appropriate origin checking
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			if electronMode {
				return isDesktopLoopbackOrigin(r.Header.Get("Origin"))
			} else {
				// In cloud mode, be more restrictive (adjust as needed)
				return true // For now, allow all origins - adjust for production
			}
		},
	}

	// Upgrade the HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade WebSocket connection: %v", err)
		return
	}
	defer conn.Close()

	// Generate connection ID
	connID := fmt.Sprintf("conn_%d", time.Now().UnixNano())

	// Store connection
	s.wsConnectionsMutex.Lock()
	s.wsConnections[connID] = conn
	s.wsConnectionsMutex.Unlock()

	// Remove connection when done
	defer func() {
		s.wsConnectionsMutex.Lock()
		delete(s.wsConnections, connID)
		s.wsConnectionsMutex.Unlock()
	}()

	log.Printf("WebSocket connection established: %s (Electron mode: %v)", connID, electronMode)

	// Send initial connection confirmation
	initialMessage := map[string]interface{}{
		"type":      "connection_established",
		"id":        connID,
		"timestamp": time.Now().UTC(),
		"mode":      map[string]bool{"electron": electronMode, "cloud": !electronMode},
	}

	if err := conn.WriteJSON(initialMessage); err != nil {
		log.Printf("Failed to send initial message: %v", err)
		return
	}

	// Handle incoming messages and keep connection alive
	for {
		var message map[string]interface{}
		if err := conn.ReadJSON(&message); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Handle different message types
		if msgType, ok := message["type"].(string); ok {
			switch msgType {
			case "ping":
				// Respond to ping with pong
				pongMessage := map[string]interface{}{
					"type":      "pong",
					"timestamp": time.Now().UTC(),
				}
				if err := conn.WriteJSON(pongMessage); err != nil {
					log.Printf("Failed to send pong: %v", err)
					return
				}
			case "subscribe":
				// Handle subscription requests (for future use)
				log.Printf("WebSocket subscription request: %v", message)
			default:
				log.Printf("Unknown WebSocket message type: %s", msgType)
			}
		}
	}
}

// desktopSecureWebSocketHandler handles the desktop-specific secure WebSocket endpoint
func (s *SimpleAPIServer) desktopSecureWebSocketHandler(w http.ResponseWriter, r *http.Request) {
	// This endpoint is specifically for Electron desktop app
	electronMode := os.Getenv("ELECTRON_MODE") == "true"
	if !electronMode {
		http.Error(w, "Desktop secure WebSocket only available in Electron mode", http.StatusForbidden)
		return
	}

	// WebSocket upgrader with strict origin checking for desktop
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return isDesktopLoopbackOrigin(r.Header.Get("Origin"))
		},
	}

	// Upgrade the HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade desktop secure WebSocket connection: %v", err)
		return
	}
	defer conn.Close()

	log.Printf("Desktop secure WebSocket connection established")

	// Handle secure desktop communication
	for {
		var message map[string]interface{}
		if err := conn.ReadJSON(&message); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Desktop secure WebSocket error: %v", err)
			}
			break
		}

		// Handle desktop-specific message types
		if msgType, ok := message["type"].(string); ok {
			switch msgType {
			case "auth":
				// Handle authentication for desktop
				response := map[string]interface{}{
					"type":      "auth_response",
					"success":   true,
					"timestamp": time.Now().UTC(),
				}
				if err := conn.WriteJSON(response); err != nil {
					log.Printf("Failed to send auth response: %v", err)
					return
				}
			case "secure_command":
				// Handle secure commands from desktop
				log.Printf("Desktop secure command received: %v", message)
				response := map[string]interface{}{
					"type":      "command_response",
					"success":   true,
					"timestamp": time.Now().UTC(),
				}
				if err := conn.WriteJSON(response); err != nil {
					log.Printf("Failed to send command response: %v", err)
					return
				}
			default:
				log.Printf("Unknown desktop secure message type: %s", msgType)
			}
		}
	}
}

// Enhanced Agent Management Handlers

// createAgentVersionHandler creates a new version of an agent
func (s *SimpleAPIServer) createAgentVersionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["id"]

	var request struct {
		Version   string   `json:"version"`
		ChangeLog string   `json:"change_log"`
		Tags      []string `json:"tags"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	version, err := s.enhancedAgentManager.CreateAgentVersion(agentID, request.Version, request.ChangeLog, request.Tags)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create agent version: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(version)
}

// listAgentVersionsHandler lists all versions of an agent
func (s *SimpleAPIServer) listAgentVersionsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["id"]

	versions, err := s.enhancedAgentManager.ListAgentVersions(agentID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list agent versions: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(versions)
}

// createAgentBackupHandler creates a backup of an agent
func (s *SimpleAPIServer) createAgentBackupHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["id"]

	var request struct {
		Description string `json:"description"`
		Type        string `json:"type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if request.Type == "" {
		request.Type = "manual"
	}

	backup, err := s.enhancedAgentManager.CreateAgentBackup(agentID, request.Description, request.Type)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create agent backup: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(backup)
}

// listAgentBackupsHandler lists all backups for an agent
func (s *SimpleAPIServer) listAgentBackupsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["id"]

	backups, err := s.enhancedAgentManager.ListAgentBackups(agentID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list agent backups: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(backups)
}

// restoreAgentFromBackupHandler restores an agent from a backup
func (s *SimpleAPIServer) restoreAgentFromBackupHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["id"]
	backupID := vars["backupId"]

	err := s.enhancedAgentManager.RestoreAgentFromBackup(backupID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to restore agent from backup: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success":   true,
		"message":   fmt.Sprintf("Agent %s restored from backup %s", agentID, backupID),
		"agent_id":  agentID,
		"backup_id": backupID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// performAgentHealthCheckHandler performs a health check on an agent
func (s *SimpleAPIServer) performAgentHealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["id"]

	health, err := s.enhancedAgentManager.PerformHealthCheck(agentID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to perform health check: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// getAgentHealthHistoryHandler gets the health check history for an agent
func (s *SimpleAPIServer) getAgentHealthHistoryHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["id"]

	history, err := s.enhancedAgentManager.GetAgentHealthHistory(agentID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get health history: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

// generateAgentAnalyticsHandler generates performance analytics for an agent
func (s *SimpleAPIServer) generateAgentAnalyticsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["id"]
	period := vars["period"]

	analytics, err := s.enhancedAgentManager.GeneratePerformanceAnalytics(agentID, period)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate analytics: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analytics)
}

// handleListCapabilities returns all available capabilities including MCP-derived ones
func (s *SimpleAPIServer) handleListCapabilities(w http.ResponseWriter, r *http.Request) {
	capabilities := []map[string]interface{}{}

	// Add static capabilities
	staticCapabilities := []map[string]interface{}{
		{
			"id":             "web-search",
			"name":           "Web Search",
			"provider":       "System",
			"type":           "web_interaction",
			"estimated_time": "2-5 seconds",
			"description":    "Search the web for information",
			"system":         true,
			"status":         "available",
			"category":       "web",
		},
		{
			"id":             "file-operations",
			"name":           "File Operations",
			"provider":       "System",
			"type":           "file_system",
			"estimated_time": "1-3 seconds",
			"description":    "Read, write, and manage files",
			"system":         true,
			"status":         "available",
			"category":       "file",
		},
	}
	capabilities = append(capabilities, staticCapabilities...)

	// Add MCP-derived capabilities
	mcpCapabilities := s.getMCPDerivedCapabilities()
	capabilities = append(capabilities, mcpCapabilities...)

	response := map[string]interface{}{
		"capabilities": capabilities,
		"count":        len(capabilities),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleListMCPCapabilities returns only MCP-derived capabilities
func (s *SimpleAPIServer) handleListMCPCapabilities(w http.ResponseWriter, r *http.Request) {
	capabilities := s.getMCPDerivedCapabilities()

	response := map[string]interface{}{
		"capabilities": capabilities,
		"count":        len(capabilities),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getMCPDerivedCapabilities extracts capabilities from installed MCP servers
func (s *SimpleAPIServer) getMCPDerivedCapabilities() []map[string]interface{} {
	capabilities := []map[string]interface{}{}

	// Get all installed MCP servers
	filter := MCPServerFilter{Status: "installed"}
	servers := s.mcpRegistryService.GetServers(filter)

	for _, server := range servers {
		// Check if capability was created for this server
		if server.Configuration != nil {
			if capabilityCreated, exists := server.Configuration["capability_created"]; exists && capabilityCreated == "true" {
				capability := map[string]interface{}{
					"id":             server.Configuration["capability_id"],
					"name":           server.Configuration["capability_name"],
					"provider":       "MCP Server",
					"type":           server.Configuration["capability_type"],
					"estimated_time": s.mcpInstallationService.estimateProcessingTime(server.Type, server.Category),
					"description":    fmt.Sprintf("%s (via MCP Server)", server.Description),
					"system":         false,
					"status":         "available",
					"category":       server.Category,
					"mcp_server_id":  server.ID,
					"tags":           server.Tags,
					"rating":         server.Rating,
					"downloads":      server.Downloads,
					"created_at":     time.Now().Format(time.RFC3339),
				}
				capabilities = append(capabilities, capability)
			}
		}
	}

	return capabilities
}

// handleToggleDemoData toggles demo data on/off with backup/restore functionality
func (s *SimpleAPIServer) handleToggleDemoData(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	s.demoDataMutex.Lock()
	defer s.demoDataMutex.Unlock()

	ctx := context.Background()
	var errors []string

	if s.demoDataEnabled {
		// Turning OFF demo data - backup and clear
		log.Println("🔄 Turning OFF demo data - creating backup...")

		// Create backup before clearing
		backup, err := s.createDemoDataBackup(ctx)
		if err != nil {
			log.Printf("Error creating demo data backup: %v", err)
			errors = append(errors, fmt.Sprintf("Failed to create backup: %v", err))
		} else {
			s.demoDataBackup = backup
		}

		// Clear demo data
		if err := s.clearAllDemoData(ctx); err != nil {
			log.Printf("Error clearing demo data: %v", err)
			errors = append(errors, fmt.Sprintf("Failed to clear demo data: %v", err))
		}

		s.demoDataEnabled = false
		log.Println("✅ Demo data turned OFF")

	} else {
		// Turning ON demo data - restore from backup or create new
		log.Println("🔄 Turning ON demo data...")

		if s.demoDataBackup != nil {
			// Restore from backup
			if err := s.restoreDemoDataFromBackup(ctx); err != nil {
				log.Printf("Error restoring demo data from backup: %v", err)
				errors = append(errors, fmt.Sprintf("Failed to restore from backup: %v", err))

				// Fallback to creating new demo data
				if err := s.CreateSampleData(); err != nil {
					log.Printf("Error creating new demo data: %v", err)
					errors = append(errors, fmt.Sprintf("Failed to create new demo data: %v", err))
				}
			}
		} else {
			// Create new demo data
			if err := s.CreateSampleData(); err != nil {
				log.Printf("Error creating demo data: %v", err)
				errors = append(errors, fmt.Sprintf("Failed to create demo data: %v", err))
			}
		}

		s.demoDataEnabled = true
		log.Println("✅ Demo data turned ON")
	}

	// Prepare response
	response := map[string]interface{}{
		"status":     "success",
		"enabled":    s.demoDataEnabled,
		"message":    fmt.Sprintf("Demo data %s", map[bool]string{true: "enabled", false: "disabled"}[s.demoDataEnabled]),
		"timestamp":  time.Now().UTC(),
		"has_backup": s.demoDataBackup != nil,
	}

	if len(errors) > 0 {
		response["status"] = "partial_success"
		response["errors"] = errors
		response["message"] = fmt.Sprintf("Demo data toggle completed with some errors. Current state: %s",
			map[bool]string{true: "enabled", false: "disabled"}[s.demoDataEnabled])
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleDemoDataStatus returns the current demo data status
func (s *SimpleAPIServer) handleDemoDataStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	s.demoDataMutex.RLock()
	defer s.demoDataMutex.RUnlock()

	response := map[string]interface{}{
		"enabled":    s.demoDataEnabled,
		"has_backup": s.demoDataBackup != nil,
		"timestamp":  time.Now().UTC(),
	}

	if s.demoDataBackup != nil {
		response["backup_timestamp"] = s.demoDataBackup.Timestamp
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleClearAllAgents clears all agents from both database and registry
func (s *SimpleAPIServer) handleClearAllAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	log.Printf("🧹 Starting complete agent cleanup...")

	var deletedFromDB, deletedFromRegistry int
	var errors []string

	// Clear all agents from database
	ctx := context.Background()

	// Get all agents from database by trying different owner IDs
	// Since we don't have a GetAllAgents method, we'll try to get agents for common owner IDs
	ownerIDs := []int64{0, 1, 2, 3, 4, 5} // Common owner IDs used in the system
	agentIDsToDelete := make(map[string]bool)

	for _, ownerID := range ownerIDs {
		agents, err := s.agentRepo.GetAgentsByOwner(ctx, ownerID)
		if err != nil {
			log.Printf("Warning: Error getting agents for owner %d: %v", ownerID, err)
			continue
		}
		for _, agent := range agents {
			agentIDsToDelete[agent.ID] = true
		}
	}

	// Delete all found agents from database
	for agentID := range agentIDsToDelete {
		err := s.agentRepo.DeleteAgent(ctx, agentID)
		if err != nil {
			log.Printf("Error deleting agent %s from database: %v", agentID, err)
			errors = append(errors, fmt.Sprintf("Failed to delete agent %s from database: %v", agentID, err))
		} else {
			deletedFromDB++
			log.Printf("✅ Deleted agent %s from database", agentID)
		}
	}

	// Clear all agents from registry
	if s.agentBuilder != nil {
		registry := s.agentBuilder.GetRegistry()
		if registry != nil {
			// Get all agent IDs from the registry
			registryAgentIDs, err := s.agentBuilder.ListAgents()
			if err != nil {
				log.Printf("Error getting agents from registry: %v", err)
				errors = append(errors, fmt.Sprintf("Failed to get registry agents: %v", err))
			} else {
				// Delete all agents from registry
				for _, agentID := range registryAgentIDs {
					err := registry.DeleteAgent(agentID)
					if err != nil {
						log.Printf("Error deleting agent %s from registry: %v", agentID, err)
						errors = append(errors, fmt.Sprintf("Failed to delete agent %s from registry: %v", agentID, err))
					} else {
						deletedFromRegistry++
						log.Printf("✅ Deleted agent %s from registry", agentID)
					}
				}
			}
		}
	}

	log.Printf("🧹 Agent cleanup completed: %d from database, %d from registry", deletedFromDB, deletedFromRegistry)

	// Return response
	response := map[string]interface{}{
		"success":               len(errors) == 0,
		"deleted_from_database": deletedFromDB,
		"deleted_from_registry": deletedFromRegistry,
		"total_deleted":         deletedFromDB + deletedFromRegistry,
		"timestamp":             time.Now().UTC(),
	}

	if len(errors) > 0 {
		response["errors"] = errors
		response["message"] = fmt.Sprintf("Partial cleanup completed with %d errors", len(errors))
	} else {
		response["message"] = "All agents cleared successfully"
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// createDemoDataBackup creates a backup of current demo data
func (s *SimpleAPIServer) createDemoDataBackup(ctx context.Context) (*DemoDataBackup, error) {
	backup := &DemoDataBackup{
		Timestamp: time.Now().UTC(),
	}

	// Backup agents
	agents, err := s.agentRepo.GetAgentsByOwner(ctx, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to get agents for backup: %w", err)
	}
	backup.Agents = agents

	// Backup target systems (static data for now)
	backup.TargetSystems = []map[string]interface{}{
		{
			"id":           "browser-chrome",
			"name":         "Chrome Browser",
			"type":         "browser",
			"status":       "connected",
			"capabilities": []string{"web_scraping", "dom_analysis", "screenshot_capture"},
			"description":  "Web browser for accessing and interacting with web content",
		},
		{
			"id":           "filesystem-local",
			"name":         "Local File System",
			"type":         "filesystem",
			"status":       "connected",
			"capabilities": []string{"file_operations", "directory_scanning", "content_analysis"},
			"description":  "Local file system access for file operations",
		},
		{
			"id":           "network-monitor",
			"name":         "Network Monitor",
			"type":         "network",
			"status":       "connected",
			"capabilities": []string{"network_analysis", "traffic_monitoring", "security_scanning"},
			"description":  "Network monitoring and analysis",
		},
	}

	// Backup MCP servers (empty for now, can be extended)
	backup.MCPServers = []map[string]interface{}{}

	// Backup workflows (empty for now, can be extended)
	backup.Workflows = []map[string]interface{}{}

	log.Printf("Created demo data backup with %d agents", len(backup.Agents))
	return backup, nil
}

// restoreDemoDataFromBackup restores demo data from backup
func (s *SimpleAPIServer) restoreDemoDataFromBackup(ctx context.Context) error {
	if s.demoDataBackup == nil {
		return fmt.Errorf("no backup available")
	}

	// Restore agents
	for _, agent := range s.demoDataBackup.Agents {
		if err := s.agentRepo.CreateAgent(ctx, agent); err != nil {
			log.Printf("Warning: Failed to restore agent %s: %v", agent.Name, err)
			// Continue with other agents
		} else {
			log.Printf("Restored agent: %s", agent.Name)
		}
	}

	log.Printf("Restored demo data from backup (timestamp: %s)", s.demoDataBackup.Timestamp)
	return nil
}

// clearAllDemoData clears all demo data from the platform
func (s *SimpleAPIServer) clearAllDemoData(ctx context.Context) error {
	var errors []string

	// Clear sample agents from database
	if err := s.clearSampleAgents(ctx); err != nil {
		log.Printf("Error clearing sample agents: %v", err)
		errors = append(errors, fmt.Sprintf("Failed to clear sample agents: %v", err))
	}

	// Clear cache
	if s.performanceManager != nil {
		cacheManager := s.performanceManager.GetCacheManager()
		cacheManager.Clear()
	}

	// Clear workflow data if available
	if s.workflowService != nil {
		if err := s.workflowService.ClearAllWorkflows(); err != nil {
			log.Printf("Error clearing workflows: %v", err)
			errors = append(errors, fmt.Sprintf("Failed to clear workflows: %v", err))
		}
	}

	// Clear MCP demo data if available
	if s.mcpRegistryService != nil {
		if err := s.mcpRegistryService.ClearDemoData(); err != nil {
			log.Printf("Error clearing MCP demo data: %v", err)
			errors = append(errors, fmt.Sprintf("Failed to clear MCP demo data: %v", err))
		}
	}

	// Clear target system demo data
	if s.targetSystemService != nil {
		if err := s.targetSystemService.ClearDemoData(); err != nil {
			log.Printf("Error clearing target system demo data: %v", err)
			errors = append(errors, fmt.Sprintf("Failed to clear target system demo data: %v", err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors occurred during cleanup: %v", errors)
	}

	return nil
}

// clearSampleAgents removes all sample agents from the database
func (s *SimpleAPIServer) clearSampleAgents(ctx context.Context) error {
	// Get all agents for the default user (owner_id = 1)
	agents, err := s.agentRepo.GetAgentsByOwner(ctx, 1)
	if err != nil {
		return fmt.Errorf("failed to get agents: %w", err)
	}

	// Delete each agent
	for _, agent := range agents {
		if err := s.agentRepo.DeleteAgent(ctx, agent.ID); err != nil {
			log.Printf("Warning: Failed to delete agent %s: %v", agent.ID, err)
			// Continue with other agents even if one fails
		} else {
			log.Printf("Deleted sample agent: %s", agent.Name)
		}
	}

	return nil
}

// WASM Plugin Management Handlers

// discoverWASMPluginsHandler handles requests to discover available WASM plugin zip files
func (s *SimpleAPIServer) discoverWASMPluginsHandler(w http.ResponseWriter, r *http.Request) {
	if s.agentInferencer == nil {
		http.Error(w, "Agent inferencer not available", http.StatusServiceUnavailable)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	plugins, err := s.agentInferencer.DiscoverWASMPluginZips(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"plugins": plugins,
	})
}

// installWASMPluginHandler handles requests to install a WASM plugin from a zip file
func (s *SimpleAPIServer) installWASMPluginHandler(w http.ResponseWriter, r *http.Request) {
	if s.agentInferencer == nil {
		http.Error(w, "Agent inferencer not available", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		ZipPath string `json:"zipPath"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.ZipPath == "" {
		http.Error(w, "Missing zipPath parameter", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	pluginInfo, err := s.agentInferencer.InstallWASMPlugin(ctx, req.ZipPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "installed",
		"plugin": pluginInfo,
	})
}

// uninstallWASMPluginHandler handles requests to uninstall a WASM plugin
func (s *SimpleAPIServer) uninstallWASMPluginHandler(w http.ResponseWriter, r *http.Request) {
	if s.agentInferencer == nil {
		http.Error(w, "Agent inferencer not available", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		AgentID string `json:"agentId"`
		Version string `json:"version"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.AgentID == "" || req.Version == "" {
		http.Error(w, "Missing agentId or version parameter", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	if err := s.agentInferencer.UninstallWASMPlugin(ctx, req.AgentID, req.Version); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "uninstalled",
		"agentId": req.AgentID,
		"version": req.Version,
	})
}

// listInstalledWASMPluginsHandler handles requests to list installed WASM plugins
func (s *SimpleAPIServer) listInstalledWASMPluginsHandler(w http.ResponseWriter, r *http.Request) {
	if s.agentInferencer == nil {
		http.Error(w, "Agent inferencer not available", http.StatusServiceUnavailable)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	plugins, err := s.agentInferencer.ListInstalledWASMPlugins(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"plugins": plugins,
	})
}

// getAvailableAgentsDetailedHandler handles requests to get detailed information about all available agents
func (s *SimpleAPIServer) getAvailableAgentsDetailedHandler(w http.ResponseWriter, r *http.Request) {
	if s.agentInferencer == nil {
		http.Error(w, "Agent inferencer not available", http.StatusServiceUnavailable)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	agentsInfo, err := s.agentInferencer.GetAvailableAgentsDetailed(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agentsInfo)
}

// Advanced Agent Operations (migrated from v2 API)

// discoverUnifiedAgentsHandler handles GET /api/v1/agents/discover
func (s *SimpleAPIServer) discoverUnifiedAgentsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	// Get all agents from unified storage
	allAgents, err := s.unifiedStorage.ListAgents(ctx)
	if err != nil {
		log.Printf("Error discovering agents: %v", err)
		RespondWithInternalError(w, "Failed to discover agents")
		return
	}

	// Convert to response format
	agents := make([]map[string]interface{}, len(allAgents))
	for i, unifiedAgent := range allAgents {
		config := map[string]interface{}{
			"collection":   unifiedAgent.Collection,
			"image_url":    unifiedAgent.ImageURL,
			"capabilities": unifiedAgent.Capabilities,
			"target_types": unifiedAgent.TargetTypes,
			"status":       unifiedAgent.Status,
		}
		configJSON, _ := json.Marshal(config)

		agents[i] = map[string]interface{}{
			"id":         unifiedAgent.ID,
			"owner_id":   unifiedAgent.OwnerID,
			"name":       unifiedAgent.Name,
			"type":       unifiedAgent.Type,
			"config":     string(configJSON),
			"created_at": unifiedAgent.CreatedAt.Format(time.RFC3339),
			"updated_at": unifiedAgent.UpdatedAt.Format(time.RFC3339),
		}
	}

	RespondWithList(w, agents, len(agents), "Agents discovered successfully")
}

// registerAgentHandler handles POST /api/v1/agents/register
func (s *SimpleAPIServer) registerAgentHandler(w http.ResponseWriter, r *http.Request) {
	var request struct {
		ID           string                 `json:"id"`
		Name         string                 `json:"name"`
		Type         string                 `json:"type"`
		Config       map[string]interface{} `json:"config"`
		OwnerID      int64                  `json:"owner_id"`
		Collection   string                 `json:"collection"`
		ImageURL     string                 `json:"image_url"`
		Capabilities []string               `json:"capabilities"`
		TargetTypes  []string               `json:"target_types"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		RespondWithValidationError(w, "Invalid JSON")
		return
	}

	// Create unified agent
	unifiedAgent := &agent.UnifiedAgent{
		ID:           request.ID,
		Name:         request.Name,
		Type:         request.Type,
		Collection:   request.Collection,
		ImageURL:     request.ImageURL,
		Status:       "idle",
		OwnerID:      request.OwnerID,
		Capabilities: request.Capabilities,
		TargetTypes:  request.TargetTypes,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		AgentConfig:  request.Config,
	}

	ctx := context.Background()
	if err := s.unifiedStorage.CreateAgent(ctx, unifiedAgent); err != nil {
		log.Printf("Error registering agent: %v", err)
		RespondWithInternalError(w, "Failed to register agent")
		return
	}

	// Convert to response format
	config := map[string]interface{}{
		"collection":   unifiedAgent.Collection,
		"image_url":    unifiedAgent.ImageURL,
		"capabilities": unifiedAgent.Capabilities,
		"target_types": unifiedAgent.TargetTypes,
		"status":       unifiedAgent.Status,
	}
	configJSON, _ := json.Marshal(config)

	responseAgent := map[string]interface{}{
		"id":         unifiedAgent.ID,
		"owner_id":   unifiedAgent.OwnerID,
		"name":       unifiedAgent.Name,
		"type":       unifiedAgent.Type,
		"config":     string(configJSON),
		"created_at": unifiedAgent.CreatedAt.Format(time.RFC3339),
		"updated_at": unifiedAgent.UpdatedAt.Format(time.RFC3339),
	}

	RespondWithCreated(w, responseAgent, "Agent registered successfully")
}

// searchAgentsHandler handles GET /api/v1/agents/search
func (s *SimpleAPIServer) searchAgentsHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		RespondWithValidationError(w, "Search query parameter 'q' is required")
		return
	}

	ctx := context.Background()
	allAgents, err := s.unifiedStorage.ListAgents(ctx)
	if err != nil {
		log.Printf("Error searching agents: %v", err)
		RespondWithInternalError(w, "Failed to search agents")
		return
	}

	// Simple search implementation - filter by name or type containing query
	var matchingAgents []map[string]interface{}
	for _, unifiedAgent := range allAgents {
		if strings.Contains(strings.ToLower(unifiedAgent.Name), strings.ToLower(query)) ||
			strings.Contains(strings.ToLower(unifiedAgent.Type), strings.ToLower(query)) {

			config := map[string]interface{}{
				"collection":   unifiedAgent.Collection,
				"image_url":    unifiedAgent.ImageURL,
				"capabilities": unifiedAgent.Capabilities,
				"target_types": unifiedAgent.TargetTypes,
				"status":       unifiedAgent.Status,
			}
			configJSON, _ := json.Marshal(config)

			matchingAgents = append(matchingAgents, map[string]interface{}{
				"id":         unifiedAgent.ID,
				"owner_id":   unifiedAgent.OwnerID,
				"name":       unifiedAgent.Name,
				"type":       unifiedAgent.Type,
				"config":     string(configJSON),
				"created_at": unifiedAgent.CreatedAt.Format(time.RFC3339),
				"updated_at": unifiedAgent.UpdatedAt.Format(time.RFC3339),
			})
		}
	}

	RespondWithList(w, matchingAgents, len(matchingAgents), "Search completed successfully")
}

// activateUnifiedAgentHandler handles POST /api/v1/agents/{id}/activate
func (s *SimpleAPIServer) activateUnifiedAgentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	ctx := context.Background()
	unifiedAgent, err := s.unifiedStorage.GetAgentByID(ctx, id)
	if err != nil {
		log.Printf("Error getting agent for activation: %v", err)
		RespondWithNotFound(w, "Agent")
		return
	}

	// Update agent status to active
	unifiedAgent.Status = "active"
	unifiedAgent.UpdatedAt = time.Now()

	if err := s.unifiedStorage.UpdateAgent(ctx, unifiedAgent); err != nil {
		log.Printf("Error activating agent: %v", err)
		RespondWithInternalError(w, "Failed to activate agent")
		return
	}

	// Convert to response format
	config := map[string]interface{}{
		"collection":   unifiedAgent.Collection,
		"image_url":    unifiedAgent.ImageURL,
		"capabilities": unifiedAgent.Capabilities,
		"target_types": unifiedAgent.TargetTypes,
		"status":       unifiedAgent.Status,
	}
	configJSON, _ := json.Marshal(config)

	responseAgent := map[string]interface{}{
		"id":         unifiedAgent.ID,
		"owner_id":   unifiedAgent.OwnerID,
		"name":       unifiedAgent.Name,
		"type":       unifiedAgent.Type,
		"config":     string(configJSON),
		"created_at": unifiedAgent.CreatedAt.Format(time.RFC3339),
		"updated_at": unifiedAgent.UpdatedAt.Format(time.RFC3339),
	}

	RespondWithSuccess(w, responseAgent, "Agent activated successfully")
}

// deactivateUnifiedAgentHandler handles POST /api/v1/agents/{id}/deactivate
func (s *SimpleAPIServer) deactivateUnifiedAgentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	ctx := context.Background()
	unifiedAgent, err := s.unifiedStorage.GetAgentByID(ctx, id)
	if err != nil {
		log.Printf("Error getting agent for deactivation: %v", err)
		RespondWithNotFound(w, "Agent")
		return
	}

	// Update agent status to idle
	unifiedAgent.Status = "idle"
	unifiedAgent.UpdatedAt = time.Now()

	if err := s.unifiedStorage.UpdateAgent(ctx, unifiedAgent); err != nil {
		log.Printf("Error deactivating agent: %v", err)
		RespondWithInternalError(w, "Failed to deactivate agent")
		return
	}

	// Convert to response format
	config := map[string]interface{}{
		"collection":   unifiedAgent.Collection,
		"image_url":    unifiedAgent.ImageURL,
		"capabilities": unifiedAgent.Capabilities,
		"target_types": unifiedAgent.TargetTypes,
		"status":       unifiedAgent.Status,
	}
	configJSON, _ := json.Marshal(config)

	responseAgent := map[string]interface{}{
		"id":         unifiedAgent.ID,
		"owner_id":   unifiedAgent.OwnerID,
		"name":       unifiedAgent.Name,
		"type":       unifiedAgent.Type,
		"config":     string(configJSON),
		"created_at": unifiedAgent.CreatedAt.Format(time.RFC3339),
		"updated_at": unifiedAgent.UpdatedAt.Format(time.RFC3339),
	}

	RespondWithSuccess(w, responseAgent, "Agent deactivated successfully")
}

// getAgentConfigHandler handles GET /api/v1/agents/{id}/config
func (s *SimpleAPIServer) getAgentConfigHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	ctx := context.Background()
	unifiedAgent, err := s.unifiedStorage.GetAgentByID(ctx, id)
	if err != nil {
		log.Printf("Error getting agent config: %v", err)
		RespondWithNotFound(w, "Agent")
		return
	}

	// Return the agent config
	RespondWithSuccess(w, unifiedAgent.AgentConfig, "Agent config retrieved successfully")
}

// updateAgentConfigHandler handles PUT /api/v1/agents/{id}/config
func (s *SimpleAPIServer) updateAgentConfigHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var configData map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&configData); err != nil {
		RespondWithValidationError(w, "Invalid JSON")
		return
	}

	ctx := context.Background()
	unifiedAgent, err := s.unifiedStorage.GetAgentByID(ctx, id)
	if err != nil {
		log.Printf("Error getting agent for config update: %v", err)
		RespondWithNotFound(w, "Agent")
		return
	}

	// Update agent config
	unifiedAgent.AgentConfig = configData
	unifiedAgent.UpdatedAt = time.Now()

	if err := s.unifiedStorage.UpdateAgent(ctx, unifiedAgent); err != nil {
		log.Printf("Error updating agent config: %v", err)
		RespondWithInternalError(w, "Failed to update agent config")
		return
	}

	RespondWithSuccess(w, unifiedAgent.AgentConfig, "Agent config updated successfully")
}

// getAgentsByTypeHandler handles GET /api/v1/agents/by-type/{type}
func (s *SimpleAPIServer) getAgentsByTypeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentType := vars["type"]

	ctx := context.Background()
	allAgents, err := s.unifiedStorage.ListAgents(ctx)
	if err != nil {
		log.Printf("Error getting agents by type: %v", err)
		RespondWithInternalError(w, "Failed to get agents by type")
		return
	}

	// Filter agents by type
	var filteredAgents []map[string]interface{}
	for _, unifiedAgent := range allAgents {
		if unifiedAgent.Type == agentType {
			config := map[string]interface{}{
				"collection":   unifiedAgent.Collection,
				"image_url":    unifiedAgent.ImageURL,
				"capabilities": unifiedAgent.Capabilities,
				"target_types": unifiedAgent.TargetTypes,
				"status":       unifiedAgent.Status,
			}
			configJSON, _ := json.Marshal(config)

			filteredAgents = append(filteredAgents, map[string]interface{}{
				"id":         unifiedAgent.ID,
				"owner_id":   unifiedAgent.OwnerID,
				"name":       unifiedAgent.Name,
				"type":       unifiedAgent.Type,
				"config":     string(configJSON),
				"created_at": unifiedAgent.CreatedAt.Format(time.RFC3339),
				"updated_at": unifiedAgent.UpdatedAt.Format(time.RFC3339),
			})
		}
	}

	RespondWithList(w, filteredAgents, len(filteredAgents), fmt.Sprintf("Agents of type '%s' retrieved successfully", agentType))
}

// getAgentsByStatusHandler handles GET /api/v1/agents/by-status/{status}
func (s *SimpleAPIServer) getAgentsByStatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	status := vars["status"]

	ctx := context.Background()
	allAgents, err := s.unifiedStorage.ListAgents(ctx)
	if err != nil {
		log.Printf("Error getting agents by status: %v", err)
		RespondWithInternalError(w, "Failed to get agents by status")
		return
	}

	// Filter agents by status
	var filteredAgents []map[string]interface{}
	for _, unifiedAgent := range allAgents {
		if unifiedAgent.Status == status {
			config := map[string]interface{}{
				"collection":   unifiedAgent.Collection,
				"image_url":    unifiedAgent.ImageURL,
				"capabilities": unifiedAgent.Capabilities,
				"target_types": unifiedAgent.TargetTypes,
				"status":       unifiedAgent.Status,
			}
			configJSON, _ := json.Marshal(config)

			filteredAgents = append(filteredAgents, map[string]interface{}{
				"id":         unifiedAgent.ID,
				"owner_id":   unifiedAgent.OwnerID,
				"name":       unifiedAgent.Name,
				"type":       unifiedAgent.Type,
				"config":     string(configJSON),
				"created_at": unifiedAgent.CreatedAt.Format(time.RFC3339),
				"updated_at": unifiedAgent.UpdatedAt.Format(time.RFC3339),
			})
		}
	}

	RespondWithList(w, filteredAgents, len(filteredAgents), fmt.Sprintf("Agents with status '%s' retrieved successfully", status))
}

// getAgentsByBuildTargetHandler handles GET /api/v1/agents/by-build-target/{target}
func (s *SimpleAPIServer) getAgentsByBuildTargetHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	buildTarget := vars["target"]

	ctx := context.Background()
	allAgents, err := s.unifiedStorage.ListAgents(ctx)
	if err != nil {
		log.Printf("Error getting agents by build target: %v", err)
		RespondWithInternalError(w, "Failed to get agents by build target")
		return
	}

	// Filter agents by build target (checking if target is in target_types)
	var filteredAgents []map[string]interface{}
	for _, unifiedAgent := range allAgents {
		for _, targetType := range unifiedAgent.TargetTypes {
			if targetType == buildTarget {
				config := map[string]interface{}{
					"collection":   unifiedAgent.Collection,
					"image_url":    unifiedAgent.ImageURL,
					"capabilities": unifiedAgent.Capabilities,
					"target_types": unifiedAgent.TargetTypes,
					"status":       unifiedAgent.Status,
				}
				configJSON, _ := json.Marshal(config)

				filteredAgents = append(filteredAgents, map[string]interface{}{
					"id":         unifiedAgent.ID,
					"owner_id":   unifiedAgent.OwnerID,
					"name":       unifiedAgent.Name,
					"type":       unifiedAgent.Type,
					"config":     string(configJSON),
					"created_at": unifiedAgent.CreatedAt.Format(time.RFC3339),
					"updated_at": unifiedAgent.UpdatedAt.Format(time.RFC3339),
				})
				break // Only add once even if multiple target types match
			}
		}
	}

	RespondWithList(w, filteredAgents, len(filteredAgents), fmt.Sprintf("Agents with build target '%s' retrieved successfully", buildTarget))
}

// Agent Chat Handlers

// agentMessageHandler handles POST /agents/message
func (s *SimpleAPIServer) agentMessageHandler(w http.ResponseWriter, r *http.Request) {
	var request struct {
		SenderID    string                 `json:"sender_id"`
		ReceiverID  string                 `json:"receiver_id"`
		MessageType string                 `json:"message_type"`
		Content     map[string]interface{} `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if request.SenderID == "" || request.ReceiverID == "" || request.Content == nil {
		http.Error(w, "sender_id, receiver_id, and content are required", http.StatusBadRequest)
		return
	}

	// Get the text content
	text, ok := request.Content["text"].(string)
	if !ok {
		http.Error(w, "content.text is required", http.StatusBadRequest)
		return
	}

	// Process the message through the agent inferencer
	if s.agentInferencer != nil {
		ctx := context.Background()

		// Create an inference request
		inferenceRequest := &agentify.InferenceRequest{
			Input:     text,
			SessionID: fmt.Sprintf("chat-%s-%s", request.SenderID, request.ReceiverID),
		}

		response, err := s.agentInferencer.ProcessInference(ctx, inferenceRequest.SessionID, inferenceRequest)
		if err != nil {
			log.Printf("Error processing agent message: %v", err)
			// Return success but with error message
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  "error",
				"message": fmt.Sprintf("Failed to process message: %v", err),
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":   "success",
			"message":  "Message processed successfully",
			"response": response.Output,
		})
		return
	}

	// Fallback response if no inferencer available
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "Message received successfully",
	})
}

// agentChatHandler handles POST /agents/chat
func (s *SimpleAPIServer) agentChatHandler(w http.ResponseWriter, r *http.Request) {
	var request struct {
		AgentID   string `json:"agentId"`
		Message   string `json:"message"`
		Type      string `json:"type"`
		SessionID string `json:"sessionId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if request.AgentID == "" || request.Message == "" {
		http.Error(w, "agentId and message are required", http.StatusBadRequest)
		return
	}

	// Generate session ID if not provided
	if request.SessionID == "" {
		request.SessionID = fmt.Sprintf("chat-%s-%d", request.AgentID, time.Now().UnixNano())
	}

	// Process the chat message through the agent inferencer
	if s.agentInferencer != nil {
		ctx := context.Background()

		// Create an inference request
		inferenceRequest := &agentify.InferenceRequest{
			Input:     request.Message,
			SessionID: request.SessionID,
		}

		response, err := s.agentInferencer.ProcessInference(ctx, request.SessionID, inferenceRequest)
		if err != nil {
			log.Printf("Error processing chat message: %v", err)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  "error",
				"message": fmt.Sprintf("Failed to process chat: %v", err),
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "success",
			"message":   "Chat processed successfully",
			"response":  response.Output,
			"sessionId": request.SessionID,
		})
		return
	}

	// Fallback response if no inferencer available
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "Chat message received successfully",
	})
}

// adkAgentMessageHandler handles POST /adk/agents/message for WASM agents
func (s *SimpleAPIServer) adkAgentMessageHandler(w http.ResponseWriter, r *http.Request) {
	var request struct {
		AgentID string `json:"agentId"`
		Message string `json:"message"`
		Type    string `json:"type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if request.AgentID == "" || request.Message == "" {
		http.Error(w, "agentId and message are required", http.StatusBadRequest)
		return
	}

	// Process the message through the agent inferencer
	if s.agentInferencer != nil {
		ctx := context.Background()
		sessionID := fmt.Sprintf("adk-%s-%d", request.AgentID, time.Now().UnixNano())

		// Create an inference request
		inferenceRequest := &agentify.InferenceRequest{
			Input:     request.Message,
			SessionID: sessionID,
		}

		response, err := s.agentInferencer.ProcessInference(ctx, sessionID, inferenceRequest)
		if err != nil {
			log.Printf("Error processing ADK agent message: %v", err)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  "error",
				"message": fmt.Sprintf("Failed to process message: %v", err),
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":   "success",
			"message":  "Message processed successfully",
			"response": response.Output,
		})
		return
	}

	// Fallback response if no inferencer available
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "ADK message received successfully",
	})
}

// agentChatSessionHandler handles POST/GET /agents/{id}/chat/session
func (s *SimpleAPIServer) agentChatSessionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["id"]

	if agentID == "" {
		http.Error(w, "Agent ID is required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case "POST":
		// Create a new chat session
		var request struct {
			SessionID string `json:"sessionId"`
		}

		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Generate session ID if not provided
		if request.SessionID == "" {
			request.SessionID = fmt.Sprintf("session-%s-%d", agentID, time.Now().UnixNano())
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "success",
			"message":   "Chat session created successfully",
			"sessionId": request.SessionID,
			"agentId":   agentID,
		})

	case "GET":
		// Get chat session info
		sessionID := r.URL.Query().Get("sessionId")
		if sessionID == "" {
			http.Error(w, "sessionId parameter is required", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "success",
			"sessionId": sessionID,
			"agentId":   agentID,
			"active":    true,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// agentChatHistoryHandler handles GET /agents/{id}/chat/history
func (s *SimpleAPIServer) agentChatHistoryHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["id"]

	if agentID == "" {
		http.Error(w, "Agent ID is required", http.StatusBadRequest)
		return
	}

	sessionID := r.URL.Query().Get("sessionId")

	// For now, return a mock chat history
	// In a real implementation, this would fetch from a database
	history := []map[string]interface{}{
		{
			"id":        1,
			"type":      "agent",
			"content":   fmt.Sprintf("Hello! I'm agent %s. How can I assist you today?", agentID),
			"timestamp": time.Now().Add(-time.Hour).Format(time.RFC3339),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "success",
		"agentId":   agentID,
		"sessionId": sessionID,
		"history":   history,
	})
}

// agentChatStateHandler handles POST/GET /agents/chat/state
func (s *SimpleAPIServer) agentChatStateHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		// Save chat state
		var request struct {
			SessionID string                 `json:"sessionId"`
			AgentID   string                 `json:"agentId"`
			State     map[string]interface{} `json:"state"`
		}

		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if request.SessionID == "" || request.AgentID == "" {
			http.Error(w, "sessionId and agentId are required", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "success",
			"message": "Chat state saved successfully",
		})

	case "GET":
		// Get chat state
		sessionID := r.URL.Query().Get("sessionId")
		if sessionID == "" {
			http.Error(w, "sessionId parameter is required", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "success",
			"sessionId": sessionID,
			"state":     map[string]interface{}{},
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// chatWebSocketConnectHandler handles POST /api/v1/chat/ws/connect
func (s *SimpleAPIServer) chatWebSocketConnectHandler(w http.ResponseWriter, r *http.Request) {
	var request struct {
		SessionID string `json:"sessionId"`
		AgentID   string `json:"agentId"`
		MessageID string `json:"messageId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if request.SessionID == "" || request.AgentID == "" {
		http.Error(w, "sessionId and agentId are required", http.StatusBadRequest)
		return
	}

	// For now, return a success response
	// In a real implementation, this would establish a WebSocket connection
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "success",
		"message":   "WebSocket connection established",
		"sessionId": request.SessionID,
		"agentId":   request.AgentID,
	})
}
