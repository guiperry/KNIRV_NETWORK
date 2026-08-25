package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"knirvclient/data_engine"
	"knirvclient/inference_engine"
	"knirvclient/internal/auth"
	"knirvclient/internal/config"
	"knirvclient/internal/guardian"
	"knirvclient/internal/project"
	"knirvclient/internal/scan"
	"knirvclient/internal/server"
	"knirvclient/internal/utils"
	"knirvclient/internal/websocket"

	"github.com/joho/godotenv"
	"github.com/philippgille/chromem-go"
	"github.com/pkg/browser"
)

// Version is set at compile time
var Version = "dev"

func main() {
	log.Println("╔════════════════════════════════════════════════════════════════╗")
	log.Println("║            KNIRV Client - AI-Powered Code Guardian             ║")
	log.Println("║          Deep Visibility • Risk Detection • Auto-Fix           ║")
	log.Println("╚════════════════════════════════════════════════════════════════╝")

	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  No .env file found or failed to load, using environment variables only")
	} else {
		log.Println("✅ .env file loaded successfully")
	}

	// Load configuration from environment
	cfg := config.Load()
	log.Println("✅ Configuration loaded successfully")

	// Validate configuration
	if cfg.ProjectPath == "" {
		log.Fatal("❌ PROJECT_PATH is required")
	}

	// Initialize chromem-go persistent database in OS application data directory
	appDataDir, err := utils.GetAppDataDir()
	if err != nil {
		log.Fatalf("❌ Failed to get application data directory: %v", err)
	}
	chromemDBPath := filepath.Join(appDataDir, "chromem-db")
	log.Printf("📁 Using chromem-go database at: %s", chromemDBPath)

	globalDB, err := chromem.NewPersistentDB(chromemDBPath, true)
	if err != nil {
		log.Fatalf("❌ Failed to initialize chromem-go database: %v", err)
	}
	log.Println("✅ Chromem-go embedded database initialized successfully")

	// Create collections for different data types
	// Note: Collections are created with nil embedding function here
	// The actual embedding function will be provided when accessing the collection
	// This is fine because chromem-go allows getting a collection with a different embedding function
	collections := map[string]string{
		"projects":         "Project metadata and configuration",
		"knowledge-graphs": "Scan results with node/edge data",
		"security-issues":  "Discovered vulnerabilities and risks",
		"test-coverage":    "Code coverage reports",
		"scan-history":     "Historical scan metadata",
		"settings-history": "Configuration change audit trail",
		"remediation-logs": "AI remediation attempts and results",
	}

	for name, description := range collections {
		_, err := globalDB.GetOrCreateCollection(name, map[string]string{"type": description}, nil)
		if err != nil {
			log.Fatalf("❌ Failed to create collection %s: %v", name, err)
		}
		log.Printf("✅ Created/verified collection: %s", name)
	}

	// Initialize ChromemDB Manager for knowledge graph and risk storage
	chromemManagerPath := filepath.Join(appDataDir, "chromem-manager-db")
	chromemManager, err := data_engine.NewChromemManager(chromemManagerPath)
	if err != nil {
		log.Fatalf("❌ Failed to initialize ChromemDB Manager: %v", err)
	}
	log.Println("✅ ChromemDB Manager initialized successfully")
	defer chromemManager.Close()

	// Initialize Data Engine (optional external Chroma-db + event processing)
	var dataEngineInstance *data_engine.DataEngine
	if cfg.DataEngine.Enable {
		log.Println("📈 Initializing Data Engine...")
		dataEngineConfig := data_engine.DataEngineConfig{
			EnableKafka:      cfg.DataEngine.EnableKafka,
			KafkaBrokers:     cfg.DataEngine.KafkaBrokers,
			ChromaDBURL:      cfg.DataEngine.ChromaDBURL,
			ChromaCollection: cfg.DataEngine.ChromaCollection,
			EnableChromaDB:   cfg.DataEngine.EnableChromaDB,
			EnableWebSocket:  cfg.DataEngine.EnableWebSocket,
			WebSocketPort:    cfg.DataEngine.WebSocketPort,
			EnableRESTAPI:    cfg.DataEngine.EnableRESTAPI,
			RESTAPIPort:      cfg.DataEngine.RESTAPIPort,
			WindowSize:       1 * time.Minute,
			MetricsInterval:  30 * time.Second,
		}

		dataEngineInstance = data_engine.NewDataEngine(dataEngineConfig)
		if err := dataEngineInstance.Start(); err != nil {
			log.Printf("⚠️  Data Engine failed to start: %v. Continuing without it.", err)
			dataEngineInstance = nil
		} else {
			log.Println("✅ Data Engine started successfully")
			if cfg.DataEngine.EnableChromaDB {
				log.Printf("   📊 External Chroma-db: %s", cfg.DataEngine.ChromaDBURL)
			}
			if cfg.DataEngine.EnableWebSocket {
				log.Printf("   🔌 WebSocket server: port %d", cfg.DataEngine.WebSocketPort)
			}
			if cfg.DataEngine.EnableRESTAPI {
				log.Printf("   🌐 REST API server: port %d", cfg.DataEngine.RESTAPIPort)
			}
		}
	} else {
		log.Println("ℹ️  Data Engine disabled in configuration")
	}

	// Initialize project store
	projectStore := project.NewProjectStore(globalDB)
	log.Println("✅ Project store initialized successfully")

	// Initialize authentication service
	authService := auth.NewAuthService()
	log.Println("✅ Authentication service initialized successfully")

	// Initialize AI inference engine
	aiEngine, err := initializeInferenceEngine(cfg)
	if err != nil {
		// In CI/Docker environments, allow server to start without AI engine for health checks
		if os.Getenv("CI") != "" || os.Getenv("DOCKER_ENV") != "" {
			log.Printf("⚠️  Failed to initialize AI engine in CI/Docker environment: %v", err)
			log.Println("⚠️  Continuing without AI engine for health check purposes")
			aiEngine = nil
		} else {
			log.Fatalf("❌ Failed to initialize AI engine: %v", err)
		}
	} else {
		log.Println("✅ AI inference engine initialized successfully")
	}

	// Initialize KNIRVClient core
	guardianCore := guardian.NewKNIRVClient(cfg, aiEngine, chromemManager)
	log.Println("✅ KNIRV Client Core initialized successfully")

	// Initialize Scan Manager
	log.Println("📋 Initializing Scan Manager...")
	scanManager := scan.NewScanManager(cfg, guardianCore)
	log.Println("✅ Scan Manager initialized successfully")

	// Initialize WebSocket manager
	wsManager := websocket.NewWebSocketManager()
	log.Println("✅ WebSocket manager initialized successfully")

	// Create the main KNIRVClient server instance
	knirvClient := &server.KNIRVClient{
		Config:       cfg,
		ProjectStore: projectStore,
		AuthService:  authService,
		WSManager:    wsManager,
		Guardian:     guardianCore,
		ScanManager:  scanManager,
	}

	// Set KNIRVClient reference in WebSocketManager for delegating broadcasts
	wsManager.SetKNIRVClient(guardianCore)

	// Start the consolidated server. It reserves its port before reporting
	// readiness, so the browser, OAuth callbacks, and backend use one port.
	serverReady := make(chan int, 1)
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Start(context.Background(), knirvClient, authService, serverReady)
	}()

	select {
	case port := <-serverReady:
		printStartupBanner(cfg)
		go openDashboardInBrowser(port)
	case err := <-serverErr:
		log.Fatalf("❌ Server failed to start: %v", err)
	}

	// Run KNIRVClient main loop
	ctx := context.Background()

	// Ensure proper cleanup on exit
	defer func() {
		if dataEngineInstance != nil {
			log.Println("🛑 Shutting down Data Engine...")
			if err := dataEngineInstance.Stop(); err != nil {
				log.Printf("⚠️  Error stopping Data Engine: %v", err)
			}
		}
	}()

	if err := guardianCore.Run(ctx); err != nil {
		log.Fatalf("❌ KNIRV Client failed: %v", err)
	}
}

// initializeInferenceEngine initializes the AI inference engine with configured providers
func initializeInferenceEngine(cfg *config.Config) (*inference_engine.InferenceService, error) {
	log.Println("🤖 Initializing AI Inference Engine...")

	// Create inference service with database accessor
	dbAccessor := &DatabaseAccessorImpl{} // You may need to implement this based on your needs
	aiEngine, err := inference_engine.NewInferenceService(dbAccessor)
	if err != nil {
		return nil, err
	}

	// Configure LLM attempts based on available API keys
	var attemptConfigs []inference_engine.LLMAttemptConfig

	// Track if we have a primary provider configured
	hasPrimary := false

	// Add CloudFlare as primary if configured (uses OpenAI-compatible API)
	if cfg.AIProviders.CloudFlare.Endpoint != "" {
		attemptConfigs = append(attemptConfigs, inference_engine.LLMAttemptConfig{
			ProviderName: "openai",
			ModelName:    cfg.AIProviders.CloudFlare.Model,
			APIKeyEnvVar: "CLOUDFLARE_API_KEY",
			BaseURL:      cfg.AIProviders.CloudFlare.Endpoint,
			MaxTokens:    8000,
			IsPrimary:    true,
		})
		hasPrimary = true
		log.Println("✅ CloudFlare provider configured as primary")
	}

	// Add Cerebras as primary/fallback if configured
	if cfg.AIProviders.Cerebras.APIKey != "" {
		isPrimary := !hasPrimary
		attemptConfigs = append(attemptConfigs, inference_engine.LLMAttemptConfig{
			ProviderName: "cerebras",
			ModelName:    cfg.AIProviders.Cerebras.Model,
			APIKeyEnvVar: "CEREBRAS_API_KEY",
			MaxTokens:    8000,
			IsPrimary:    isPrimary,
		})
		if isPrimary {
			hasPrimary = true
			log.Println("✅ Cerebras provider configured as primary")
		} else {
			log.Println("✅ Cerebras provider configured as fallback")
		}
	}

	// Add Gemini - as primary if no other primary exists, otherwise as fallback
	if cfg.AIProviders.Gemini.APIKey != "" {
		// The planner model defined in the orchestrator config is the source of truth.
		// We use this name to configure the Gemini provider itself.
		plannerModelName := cfg.Orchestrator.PlannerModel
		if plannerModelName == "" {
			// Fallback to the generic Gemini model name if the specific orchestrator one isn't set.
			plannerModelName = cfg.AIProviders.Gemini.Model
		}

		isPrimary := !hasPrimary // Set as primary if no other primary provider exists
		attemptConfigs = append(attemptConfigs, inference_engine.LLMAttemptConfig{
			ProviderName: "gemini",
			ModelName:    plannerModelName, // Ensure the loaded model has the exact name the orchestrator will request.
			APIKeyEnvVar: "GEMINI_API_KEY",
			MaxTokens:    100000, // Gemini 2.5 Flash supports up to 1M tokens
			IsPrimary:    isPrimary,
		})
		if isPrimary {
			hasPrimary = true
			log.Println("✅ Gemini provider configured as primary")
		} else {
			log.Println("✅ Gemini provider configured as fallback")
		}
	}

	// Add Anthropic as fallback if configured
	if cfg.AIProviders.Anthropic.APIKey != "" {
		attemptConfigs = append(attemptConfigs, inference_engine.LLMAttemptConfig{
			ProviderName: "anthropic",
			ModelName:    cfg.AIProviders.Anthropic.Model,
			APIKeyEnvVar: "ANTHROPIC_API_KEY",
			MaxTokens:    8000,
			IsPrimary:    false,
		})
		log.Println("✅ Anthropic provider configured")
	}

	// Add OpenAI as fallback if configured
	if cfg.AIProviders.OpenAI.APIKey != "" {
		attemptConfigs = append(attemptConfigs, inference_engine.LLMAttemptConfig{
			ProviderName: "openai",
			ModelName:    cfg.AIProviders.OpenAI.Model,
			APIKeyEnvVar: "OPENAI_API_KEY",
			MaxTokens:    8000,
			IsPrimary:    false,
		})
		log.Println("✅ OpenAI provider configured")
	}

	// Add DeepSeek as fallback if configured
	if cfg.AIProviders.DeepSeek.APIKey != "" {
		attemptConfigs = append(attemptConfigs, inference_engine.LLMAttemptConfig{
			ProviderName: "deepseek",
			ModelName:    cfg.AIProviders.DeepSeek.Model,
			APIKeyEnvVar: "DEEPSEEK_API_KEY",
			MaxTokens:    8000,
			IsPrimary:    false,
		})
		log.Println("✅ DeepSeek provider configured")
	}

	if len(attemptConfigs) == 0 {
		return nil, fmt.Errorf("no AI providers configured. Please set at least one API key")
	}

	// Start the inference service with configured models from orchestrator config
	err = aiEngine.StartWithConfig(
		attemptConfigs,
		cfg.Orchestrator.PlannerModel,   // Use orchestrator planner model from env
		cfg.Orchestrator.ExecutorModels, // Use orchestrator executor models from env
		cfg.Orchestrator.FinalizerModel, // Use orchestrator finalizer model from env
		cfg.Orchestrator.VerifierModel,  // Use orchestrator verifier model from env
	)
	if err != nil {
		return nil, err
	}

	log.Printf("✅ AI Engine initialized with %d provider(s)", len(attemptConfigs))
	return aiEngine, nil
}

// printStartupBanner prints the startup information banner
func printStartupBanner(cfg *config.Config) {
	log.Println("╔════════════════════════════════════════════════════════════════╗")
	log.Println("║                    🚀 Starting KNIRV Client 🚀                 ║")
	log.Printf("║  📁 Project: %s", cfg.ProjectPath)
	log.Println("║  🤖 AI Engine: Initialized (not running)")
	log.Println("║  👷 Scan Workers: 5")
	log.Printf("║  🌐 Server: http://localhost:%d", cfg.ServerPort)
	log.Printf("║  📊 Dashboard: http://localhost:%d", cfg.ServerPort)
	log.Printf("║  🔗 WebSocket: ws://localhost:%d/ws", cfg.ServerPort)
	log.Println("╚════════════════════════════════════════════════════════════════╝")
}

// openDashboardInBrowser opens the dashboard URL in the default web browser
func openDashboardInBrowser(port int) {
	// Don't open browser in CI or Docker environments
	if os.Getenv("CI") != "" || os.Getenv("DOCKER_ENV") != "" {
		return
	}

	dashboardURL := os.Getenv("DASHBOARD_URL")
	if dashboardURL == "" {
		dashboardURL = fmt.Sprintf("http://localhost:%d", port)
	}

	log.Printf("🚀 Opening dashboard in your browser: %s", dashboardURL)
	err := browser.OpenURL(dashboardURL)
	if err != nil {
		log.Printf("⚠️  Could not open browser: %v", err)
	}
}

// DatabaseAccessorImpl is a placeholder implementation of the DatabaseAccessor interface
// You may need to implement this properly based on your database requirements
type DatabaseAccessorImpl struct{}

func (d *DatabaseAccessorImpl) GetValue(key string) (string, error) {
	// Implement key-value retrieval if needed
	return "", nil
}

func (d *DatabaseAccessorImpl) SetValue(key, value string) error {
	// Implement key-value storage if needed
	return nil
}
