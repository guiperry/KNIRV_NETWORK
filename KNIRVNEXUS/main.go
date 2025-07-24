package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"Agentic_Engine/agent"
	"Agentic_Engine/agent/migration"
	"Agentic_Engine/agentify"
	"Agentic_Engine/api"
	"Agentic_Engine/database"
	"Agentic_Engine/inference"
	"Agentic_Engine/utils"

	"github.com/joho/godotenv"
)

var staticGUIServer *http.Server // To manage the static GUI server's lifecycle

// InferenceServiceAdapter adapts the inference service to the Agent Inferencer interface
type InferenceServiceAdapter struct {
	service      *inference.InferenceService
	defaultModel string
}

func (a *InferenceServiceAdapter) GenerateText(promptText string, instructionText string) (string, error) {
	return a.service.GenerateText(a.defaultModel, promptText, instructionText)
}

func (a *InferenceServiceAdapter) GenerateTextWithCoT(promptText string) (string, error) {
	return a.service.GenerateTextWithCoT(promptText)
}

func (a *InferenceServiceAdapter) GenerateStructuredOutput(content string, schema string) (string, error) {
	// The inference service doesn't have this method, so we'll use regular generation
	prompt := fmt.Sprintf("Generate structured output for: %s\nSchema: %s", content, schema)
	return a.service.GenerateText(a.defaultModel, prompt, "Generate structured JSON output")
}

func (a *InferenceServiceAdapter) IsRunning() bool {
	return a.service.IsRunning()
}

func main() {
	// Command line flags
	var production = flag.Bool("production", false, "Run in production mode (serve static files)")
	var guiPort = flag.Int("gui-port", 8080, "Port for GUI server")
	var cleanDB = flag.Bool("clean-db", false, "Clean the database directory before starting")
	var migrateDB = flag.Bool("migrate", false, "Run agent data migration")
	flag.Parse()

	// Handle migration command
	if *migrateDB {
		migration.MigrateAgentData()
		return
	}

	// Load environment variables from .env file
	// Try multiple locations to find the .env file
	envLocations := []string{
		".env", // Current directory
	}

	// Add app data directory if available
	if appDataDir, err := utils.GetAppDataDir(); err == nil {
		envLocations = append(envLocations, filepath.Join(appDataDir, ".env"))
	}

	// Try to get executable path for additional locations
	if execPath, err := os.Executable(); err == nil {
		execDir := filepath.Dir(execPath)

		// Add executable directory and parent
		envLocations = append(envLocations,
			filepath.Join(execDir, ".env"),
			filepath.Join(filepath.Dir(execDir), ".env"))

		// Special handling for Electron app structure
		// Check for resources directory (common in Electron apps)
		resourcesDir := filepath.Join(execDir, "resources")
		if _, err := os.Stat(resourcesDir); err == nil {
			envLocations = append(envLocations,
				filepath.Join(resourcesDir, ".env"),
				filepath.Join(resourcesDir, "backend", ".env"))

			// Also check for app directory in resources
			appDir := filepath.Join(resourcesDir, "app")
			if _, err := os.Stat(appDir); err == nil {
				envLocations = append(envLocations, filepath.Join(appDir, ".env"))
			}
		}
	}

	// Try each location
	envLoaded := false
	for _, location := range envLocations {
		if _, err := os.Stat(location); err == nil {
			log.Printf("Found .env file at: %s", location)
			if err := godotenv.Load(location); err == nil {
				log.Printf("Successfully loaded environment variables from %s", location)
				envLoaded = true
				break
			} else {
				log.Printf("Error loading .env file from %s: %v", location, err)
			}
		}
	}

	if !envLoaded {
		log.Println("No .env file found, loading embedded defaults and system environment variables")
		// Load embedded default environment variables
		if err := utils.LoadEmbeddedEnv(); err != nil {
			log.Printf("Warning: Failed to load embedded environment variables: %v", err)
		}
	}

	// Read port configuration from ports.config file
	portConfig, err := utils.ReadPortConfig("ports.config")
	if err != nil {
		log.Printf("⚠️  Warning: Failed to read ports.config: %v. Using defaults.", err)
		portConfig = &utils.PortConfig{APIPort: 8081, GUIPort: 8080}
	}

	apiPort := portConfig.APIPort

	// Override GUI port from config if not set via command line
	if *guiPort == 8080 {
		*guiPort = portConfig.GUIPort
	}

	if *production {
		log.Println("🚀 Starting Inference Engine in PRODUCTION mode...")
	} else {
		log.Println("🚀 Starting Inference Engine in DEVELOPMENT mode...")
	}

	// Ensure all application data directories exist
	if err := utils.EnsureAppDataDirs(); err != nil {
		log.Fatalf("Failed to create application data directories: %v", err)
	}

	// Database paths
	dbDir, err := utils.GetDatabaseDir()
	if err != nil {
		log.Fatalf("Failed to get database directory: %v", err)
	}
	domainDBPath := filepath.Join(dbDir, "domain.db")
	authDBPath := filepath.Join(dbDir, "auth.db")

	if *cleanDB {
		log.Printf("🧹 Attempting to clean database directory: %s", dbDir)
		if err := os.RemoveAll(dbDir); err != nil {
			log.Printf("⚠️  Warning: Failed to remove database directory %s: %v. Proceeding anyway.", dbDir, err)
		} else {
			log.Printf("✅ Database directory %s cleaned successfully.", dbDir)
		}
		// Recreate the directory after cleaning
		if err := utils.EnsureDir(dbDir); err != nil {
			log.Fatalf("Failed to recreate database directory: %v", err)
		}
	}

	// Initialize auth database
	authDB, err := database.NewAuthDB(authDBPath)
	if err != nil {
		log.Fatalf("Failed to initialize auth database: %v", err)
	}
	defer authDB.Close()

	// Initialize domain database
	domainDB, err := database.NewSimpleDomainDB(domainDBPath)
	if err != nil {
		log.Fatalf("Failed to initialize domain database: %v", err)
	}
	defer domainDB.Close()

	// Initialize inference service
	inferenceService, err := inference.NewInferenceService(domainDB)
	if err != nil {
		log.Fatalf("Failed to initialize inference service: %v", err)
	}

	// Start the inference service
	log.Println("Starting inference service...")
	if err := inferenceService.Start(); err != nil {
		log.Fatalf("Failed to start inference service: %v", err)
	}
	log.Println("Inference service started successfully")
	defer inferenceService.Stop()

	// Initialize Agent Inferencer service with unified storage
	pluginsDir, err := utils.GetPluginsDir()
	if err != nil {
		log.Fatalf("Failed to get plugins directory: %v", err)
	}

	// Get the centralized agents database path for unified storage
	agentsDBPath, err := utils.GetAgentsDBPath()
	if err != nil {
		log.Fatalf("Failed to get agents database path: %v", err)
	}

	// Create unified agent storage for the agent inferencer
	unifiedAgentStorage, err := agent.NewUnifiedAgentStorage(agentsDBPath)
	if err != nil {
		log.Fatalf("Failed to create unified agent storage: %v", err)
	}

	// Create adapter for the agent inferencer interface
	unifiedStorageAdapter := agent.NewUnifiedAgentStorageAdapter(unifiedAgentStorage)

	agentInferencer := agentify.NewAgentInferencerWithStorage(pluginsDir, unifiedStorageAdapter)

	// Create an adapter to bridge the inference service interface
	inferenceAdapter := &InferenceServiceAdapter{
		service:      inferenceService,
		defaultModel: "llama-4-scout-17b-16e-instruct", // Default model to use
	}

	// Connect the inference service to the agent inferencer
	agentInferencer.SetInferenceService(inferenceAdapter)

	// Create Agent Inferencer service
	agentInferencerService := agentify.NewAgentInferencerService(pluginsDir)
	if err := agentInferencerService.Start(); err != nil {
		log.Fatalf("Failed to start Agent Inferencer service: %v", err)
	}
	defer agentInferencerService.Stop()

	// Agent discovery and registration will be handled by the unified database in the API server
	log.Println("Agent discovery will be handled by unified database system...")

	// Get JWT secret from environment
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Println("⚠️  Warning: Using embedded default JWT secret. Set JWT_SECRET environment variable in production.")
	}

	// Create shutdown channel for API server
	apiShutdownChan := make(chan struct{}, 1)

	// Create authentication service
	userRepo := database.NewUserRepository(authDB.GetDB())
	tokenRepo := database.NewTokenRepository(authDB.GetDB())
	permissionRepo := database.NewPermissionRepository(authDB.GetDB())
	roleRepo := database.NewRoleRepository(authDB.GetDB())
	authService := api.NewAuthService(userRepo, tokenRepo, permissionRepo, jwtSecret, 24*time.Hour)

	// Create additional services
	userService := api.NewUserService(userRepo, permissionRepo, roleRepo)
	workflowService := api.NewWorkflowOrchestrationService()
	workflowService.SetInferenceService(inferenceService)
	analyticsService := api.NewAnalyticsService(workflowService)
	webConnectionsService := api.NewWebConnectionsService()

	// Create enhanced connection security manager (not used directly in main)

	// Create terminal manager
	terminalManager := api.NewTerminalManager()

	// Create target system service with enhanced security
	targetSystemService := api.NewTargetSystemService()

	// Create and start API server with all services
	apiServer, err := api.NewSimpleAPIServer(apiPort, domainDBPath, apiShutdownChan, inferenceService, agentInferencer, authService, userService, analyticsService, webConnectionsService)
	if err != nil {
		log.Fatalf("Failed to create API server: %v", err)
	}

	// Create sample data for testing (only in development mode)
	if !*production {
		log.Println("Creating sample data for development...")
		if err := apiServer.CreateSampleData(); err != nil {
			log.Printf("Warning: Failed to create sample data: %v", err)
		} else {
			log.Println("Sample data created successfully")
		}
	}

	// Register additional services with the API server
	router := apiServer.GetRouter()
	if router != nil {
		// Register terminal manager handlers
		terminalManager.RegisterHandlers(router)

		// Register target system service handlers
		targetSystemService.RegisterHandlers(router)

		log.Println("✅ Registered additional service handlers")
	}

	// Start API server in background
	go func() {
		log.Printf("📡 Starting API server on :%d", apiPort)
		if err := apiServer.Start(); err != nil && err != http.ErrServerClosed {
			log.Printf("API server error: %v", err)
		}
	}()

	// Start the React GUI server
	go func() {
		if *production {
			// Production mode: serve static files
			if err := serveStaticGUI(*guiPort, &staticGUIServer); err != nil && err != http.ErrServerClosed {
				log.Printf("Static GUI server error: %v", err)
				// Optionally trigger shutdown if GUI server fails to start
				// close(shutdownFromAPIChan)
			} else if err == http.ErrServerClosed {
				log.Printf("Static GUI server error: %v", err)
			}
		} else {
			// Development mode: use npm dev server
			if err := startReactGUI(*guiPort); err != nil {
				log.Printf("Dev GUI server error: %v", err)
			}
		}
	}()

	// Wait a moment for servers to start
	time.Sleep(2 * time.Second)

	// Open browser to the React GUI (unless running in Electron mode)
	electronMode := os.Getenv("ELECTRON_MODE") == "true"
	if !electronMode {
		guiURL := fmt.Sprintf("http://localhost:%d", *guiPort)
		if *production {
			log.Printf("🌐 Opening Production GUI at %s", guiURL)
		} else {
			log.Printf("🌐 Opening Development GUI at %s", guiURL)
		}

		if err := openBrowser(guiURL); err != nil {
			log.Printf("Could not open browser automatically: %v", err)
			log.Printf("Please open your browser and navigate to: %s", guiURL)
		}
	} else {
		log.Printf("🖥️  Running in Electron mode - GUI handled by Electron")
	}

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigChan:
		log.Println("🛑 Received OS signal. Initiating shutdown...")
	case <-apiShutdownChan:
		log.Println("🛑 Received shutdown signal from frontend. Initiating shutdown...")
	}

	log.Println("🛑 Shutting down servers...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if staticGUIServer != nil {
		log.Println("Shutting down static GUI server...")
		if err := staticGUIServer.Shutdown(ctx); err != nil {
			log.Printf("Error stopping static GUI server: %v", err)
		}
	}
	if err := apiServer.Stop(ctx); err != nil {
		log.Printf("Error stopping API server: %v", err)
	}

	log.Println("✅ Shutdown complete")
}

// startReactGUI starts the React development server
func startReactGUI(port int) error {
	guiDir := "./gui"

	// Check if gui directory exists
	if _, err := os.Stat(guiDir); os.IsNotExist(err) {
		return fmt.Errorf("GUI directory not found: %s", guiDir)
	}

	// Check if node_modules exists, if not run npm install
	nodeModulesPath := filepath.Join(guiDir, "node_modules")
	if _, err := os.Stat(nodeModulesPath); os.IsNotExist(err) {
		log.Println("📦 Installing GUI dependencies...")
		cmd := exec.Command("npm", "install")
		cmd.Dir = guiDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to install GUI dependencies: %w", err)
		}
	}

	// Start the development server
	log.Printf("🎨 Starting React GUI server on port %d...", port)
	cmd := exec.Command("npm", "run", "dev", "--", "--port", fmt.Sprintf("%d", port), "--host")
	cmd.Dir = guiDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// openBrowser opens the default browser to the specified URL
func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}

//go:embed gui/dist/*
var embeddedGUI embed.FS

// serveStaticGUI serves the built React GUI from embedded files
func serveStaticGUI(port int, serverPtr **http.Server) error {
	// Create filesystem from embedded assets
	subFS, err := fs.Sub(embeddedGUI, "gui/dist")
	if err != nil {
		return fmt.Errorf("failed to create embedded filesystem: %w", err)
	}

	mux := http.NewServeMux()

	// Create a custom handler to support client-side routing
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Try to serve the requested file
		filePath := r.URL.Path
		if filePath == "/" {
			filePath = "/index.html"
		}

		// Check if file exists in embedded FS
		_, err := embeddedGUI.Open("gui/dist" + filePath)
		if err == nil {
			// Serve the requested file directly
			http.FileServer(http.FS(subFS)).ServeHTTP(w, r)
			return
		}

		// For all other routes, serve index.html for SPA routing
		// For SPA routing, serve index.html from the embedded filesystem
		http.ServeFile(w, r, filepath.Join("gui/dist", "index.html"))
	})

	log.Printf("🎨 Serving static GUI on port %d...", port)
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}
	*serverPtr = server // Assign the server instance to the provided pointer

	return server.ListenAndServe()

}
