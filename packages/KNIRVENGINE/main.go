package main

import (
	"context"
	"embed"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"time"

	"KNIRVENGINE/desktop-client/internal/agent"
	"KNIRVENGINE/desktop-client/internal/agent/migration"
	"KNIRVENGINE/desktop-client/internal/agentify"
	"KNIRVENGINE/desktop-client/internal/api"
	"KNIRVENGINE/desktop-client/internal/database"
	"KNIRVENGINE/desktop-client/internal/desktop"
	"KNIRVENGINE/desktop-client/internal/inference"
	"KNIRVENGINE/desktop-client/internal/utils"

	"github.com/joho/godotenv"
)

var staticGUIServer *http.Server // To manage the static GUI server's lifecycle

const (
	enginePIDFile = "knirv-engine.pid"
	guiPIDFile    = "knirv-engine-gui.pid"
)

var (
	guiProcessMu   sync.Mutex
	guiProcess     *os.Process
	guiProcessDone chan struct{}
)

// selectAvailablePort chooses the first available port at or above the
// preferred value, while avoiding ports owned by other KNIRVENGINE services.
func selectAvailablePort(preferred int, reserved map[int]struct{}) (int, error) {
	const maxPortAttempts = 100

	for candidate := preferred; candidate < preferred+maxPortAttempts; candidate++ {
		if _, isReserved := reserved[candidate]; isReserved {
			continue
		}

		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", candidate))
		if err != nil {
			continue
		}
		if closeErr := listener.Close(); closeErr != nil {
			return 0, fmt.Errorf("release GUI port %d: %w", candidate, closeErr)
		}
		return candidate, nil
	}

	return 0, fmt.Errorf("no available GUI port found in range %d-%d", preferred, preferred+maxPortAttempts-1)
}

func processPIDFile(name string) (string, error) {
	appDataDir, err := utils.GetAppDataDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(appDataDir, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(appDataDir, name), nil
}

func writeProcessPIDFile(name string, pid int) (string, error) {
	path, err := processPIDFile(name)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(fmt.Sprintf("%d\n", pid)), 0o600); err != nil {
		return "", err
	}
	return path, nil
}

// stopReactGUI forwards SIGINT to npm/Vite and gives it time to cleanly stop.
func stopReactGUI() {
	guiProcessMu.Lock()
	process := guiProcess
	done := guiProcessDone
	guiProcessMu.Unlock()
	if process == nil {
		return
	}

	log.Printf("Sending SIGINT to GUI process %d", process.Pid)
	if err := process.Signal(os.Interrupt); err != nil && !errors.Is(err, os.ErrProcessDone) {
		log.Printf("Failed to signal GUI process: %v", err)
		return
	}

	if done != nil {
		select {
		case <-done:
			log.Println("GUI process stopped cleanly")
		case <-time.After(5 * time.Second):
			log.Printf("GUI process %d did not stop after SIGINT", process.Pid)
		}
	}
}

// launchElectronDesktop hands a direct desktop-binary launch to the Electron
// host. The Go executable remains the backend child of that host, identified
// through ELECTRON_MODE, so this cannot recurse.
func launchElectronDesktop() (bool, error) {
	if os.Getenv("ELECTRON_MODE") == "true" || os.Getenv("KNIRVENGINE_DISABLE_DESKTOP") == "true" {
		return false, nil
	}

	workingDir, err := os.Getwd()
	if err != nil {
		return false, err
	}
	executablePath, err := os.Executable()
	if err != nil {
		return false, err
	}

	rootCandidates := []string{
		workingDir,
		filepath.Dir(executablePath),
		filepath.Dir(filepath.Dir(executablePath)), // dist/knirv-engine-*
	}
	for _, root := range rootCandidates {
		guiDir := filepath.Join(root, "gui")
		electronPath := filepath.Join(guiDir, "node_modules", ".bin", "electron")
		entryPoint := filepath.Join(guiDir, "desktop", "main.cjs")
		if _, err := os.Stat(electronPath); err != nil {
			continue
		}
		if _, err := os.Stat(entryPoint); err != nil {
			continue
		}

		cmd := exec.Command(electronPath, entryPoint)
		cmd.Dir = guiDir
		cmd.Env = append(os.Environ(), "KNIRVENGINE_BACKEND="+executablePath)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return true, fmt.Errorf("run Electron desktop host: %w", err)
		}
		return true, nil
	}

	return false, nil
}

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
	var browser = flag.Bool("browser", false, "Open the web interface in a browser instead of the desktop application")
	var guiPort = flag.Int("gui-port", 8080, "Port for GUI server")
	var guiPortFile = flag.String("gui-port-file", "", "Write the selected GUI port to this file")
	var cleanDB = flag.Bool("clean-db", false, "Clean the database directory before starting")
	var migrateDB = flag.Bool("migrate", false, "Run agent data migration")
	flag.Parse()

	// Electron deliberately refuses to run as root because doing so disables
	// Chromium's sandbox. Keep the desktop process unprivileged: operations
	// requiring capabilities elevate their individual child command in the
	// sandbox tool layer instead. This makes an accidental `sudo ./knirv-engine`
	// fail safely and explains the supported launch path.
	if shouldRefuseRootLaunch(os.Geteuid()) {
		log.Println("KNIRVENGINE must not be launched with sudo. Start it as your normal user; the bundled nsenter helper carries the capabilities it needs to attach tools, and only the one-time sandbox dependency install ever asks for sudo.")
		return
	}

	if !*browser {
		launched, err := launchElectronDesktop()
		if err != nil {
			log.Printf("Desktop host failed: %v", err)
			return
		}
		if launched {
			return
		}
	}

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

	apiPort, err := selectAvailablePort(portConfig.APIPort, map[int]struct{}{
		8082: {}, // Desktop host port
	})
	if err != nil {
		log.Fatalf("Failed to select API port: %v", err)
	}
	if apiPort != portConfig.APIPort {
		log.Printf("API port %d is unavailable; using port %d", portConfig.APIPort, apiPort)
	}

	desktopPort, err := selectAvailablePort(8082, map[int]struct{}{
		apiPort: {},
	})
	if err != nil {
		log.Fatalf("Failed to select desktop host port: %v", err)
	}
	if desktopPort != 8082 {
		log.Printf("Desktop host port 8082 is unavailable; using port %d", desktopPort)
	}

	// Override GUI port from config if not set via command line
	if *guiPort == 8080 {
		*guiPort = portConfig.GUIPort
	}

	selectedGUIPort, err := selectAvailablePort(*guiPort, map[int]struct{}{
		apiPort:     {},
		desktopPort: {},
	})
	if err != nil {
		log.Fatalf("Failed to select GUI port: %v", err)
	}
	if selectedGUIPort != *guiPort {
		log.Printf("GUI port %d is unavailable; using port %d", *guiPort, selectedGUIPort)
	}
	*guiPort = selectedGUIPort
	if *guiPortFile != "" {
		if err := os.WriteFile(*guiPortFile, []byte(strconv.Itoa(*guiPort)), 0o600); err != nil {
			log.Fatalf("Failed to write selected GUI port: %v", err)
		}
	}

	// Vite reads this file before it starts its proxy. Keep it in sync with the
	// port selected above so a manual Vite launch and the launcher use the same
	// backend, even when 8081 is occupied.
	if err := syncViteAPIProxyPort(apiPort); err != nil {
		log.Printf("Warning: failed to sync Vite API proxy port: %v", err)
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
	if err := extractEmbeddedSandboxScripts(); err != nil {
		log.Printf("Warning: failed to extract embedded sandbox scripts: %v", err)
	}
	if pidFile, err := writeProcessPIDFile(enginePIDFile, os.Getpid()); err != nil {
		log.Printf("Warning: failed to write engine PID file: %v", err)
	} else {
		defer os.Remove(pidFile)
	}

	// Database paths
	dbDir, err := utils.GetDatabaseDir()
	if err != nil {
		log.Fatalf("Failed to get database directory: %v", err)
	}
	domainDBPath := filepath.Join(dbDir, "domain.db")
	authDBPath := filepath.Join(dbDir, "auth_chromem") // Directory for chromem-go database

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

	// Initialize chromem-based auth database
	authDB, err := database.NewChromemAuthDB(authDBPath)
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
		defaultModel: "gpt-oss-120b", // Public Cerebras model available to the error-analysis service
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

	// TODO: Create chromem-compatible auth service
	// For now, we'll skip auth service initialization to get the system running
	// The ChromemAuthDB is initialized and ready for future auth integration
	log.Println("Auth database initialized with chromem-go (auth service disabled for now)")

	// Create additional services (simplified for now)
	// userService := api.NewUserService(userRepo, permissionRepo, roleRepo)
	workflowService := api.NewWorkflowOrchestrationService()
	workflowService.SetInferenceService(inferenceService)
	analyticsService := api.NewAnalyticsService(workflowService)
	webConnectionsService := api.NewWebConnectionsService()

	// Create enhanced connection security manager (not used directly in main)

	// Create terminal manager
	terminalManager := api.NewTerminalManager()

	// Create target system service with enhanced security
	targetSystemService := api.NewTargetSystemService()

	// Create simple auth service for development (no complex database required)
	authService := api.NewSimpleAuthService()
	log.Println("✅ Simple auth service initialized for development")

	// Create and start API server with all services
	apiServer, err := api.NewSimpleAPIServer(apiPort, domainDBPath, apiShutdownChan, inferenceService, agentInferencer, authService, nil, analyticsService, webConnectionsService)
	if err != nil {
		log.Fatalf("Failed to create API server: %v", err)
	}

	// Initialize Desktop Host with HRM and QR linkage
	log.Println("🧠 Initializing Desktop Host with HRM cognitive engine...")
	desktopClient, err := desktop.NewDesktopClient(desktopPort)
	if err != nil {
		log.Fatalf("Failed to create desktop host: %v", err)
	}

	// Initialize desktop host components
	if err := desktopClient.Initialize(); err != nil {
		log.Printf("⚠️  Warning: Desktop host initialization failed: %v", err)
	} else {
		log.Println("✅ Desktop host initialized successfully")
	}

	// Start desktop host server
	if err := desktopClient.Start(); err != nil {
		log.Printf("⚠️  Warning: Failed to start desktop host server: %v", err)
	} else {
		log.Printf("🚀 Desktop host server started on :%d", desktopPort)
	}

	// Ensure desktop host is stopped on shutdown
	defer func() {
		if err := desktopClient.Stop(); err != nil {
			log.Printf("Error stopping desktop host: %v", err)
		}
	}()

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
			if err := serveStaticGUI(*guiPort, apiPort, &staticGUIServer); err != nil && err != http.ErrServerClosed {
				log.Printf("Static GUI server error: %v", err)
				// Optionally trigger shutdown if GUI server fails to start
				// close(shutdownFromAPIChan)
			} else if err == http.ErrServerClosed {
				log.Printf("Static GUI server error: %v", err)
			}
		} else {
			// Development mode: use npm dev server
			if err := startReactGUI(*guiPort, apiPort); err != nil {
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
	stopReactGUI()
	if err := apiServer.Stop(ctx); err != nil {
		log.Printf("Error stopping API server: %v", err)
	}

	log.Println("✅ Shutdown complete")
}

func shouldRefuseRootLaunch(euid int) bool {
	return euid == 0
}

// startReactGUI starts the React development server
func startReactGUI(port, apiPort int) error {
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
	cmd.Env = append(os.Environ(), fmt.Sprintf("VITE_API_BASE_URL=http://localhost:%d", apiPort))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start GUI process: %w", err)
	}

	guiProcessMu.Lock()
	guiProcess = cmd.Process
	guiProcessDone = make(chan struct{})
	done := guiProcessDone
	guiProcessMu.Unlock()

	pidFile, pidFileErr := writeProcessPIDFile(guiPIDFile, cmd.Process.Pid)
	if pidFileErr != nil {
		log.Printf("Warning: failed to write GUI PID file: %v", pidFileErr)
	}

	err := cmd.Wait()
	if pidFile != "" {
		_ = os.Remove(pidFile)
	}
	guiProcessMu.Lock()
	guiProcess = nil
	close(done)
	guiProcessDone = nil
	guiProcessMu.Unlock()
	return err
}

// syncViteAPIProxyPort writes the runtime API port consumed by vite.config.ts.
// It is intentionally a small committed-format file so it also works if the
// GUI is launched directly with npm instead of through this Go process.
func syncViteAPIProxyPort(apiPort int) error {
	contents := fmt.Sprintf("# Frontend Port Configuration\n# Generated by KNIRVENGINE at startup; do not hard-code an API port here.\nAPI_PORT=%d\n", apiPort)
	return os.WriteFile(filepath.Join("gui", "ports.config"), []byte(contents), 0o644)
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

// Project-owned sandbox helper scripts are embedded in the binary and
// extracted to the app data directory at startup, rather than shipped as
// loose files a separate bundling step has to copy next to the executable
// (that's still how third-party sandbox binaries like bwrap/frida-server are
// handled — see scripts/bundle-sandbox-tools.sh — because they're large,
// vendored, and need setcap applied to a persistent on-disk file; extracting
// a fresh copy on every launch would strip that capability every time).
//
//go:embed scripts/frida-bridge.py
var embeddedFridaBridge []byte

// extractEmbeddedSandboxScripts writes every embedded sandbox script to
// utils.GetSandboxScriptsDir(), overwriting any existing copy so the
// extracted file always matches what's actually compiled into this binary.
func extractEmbeddedSandboxScripts() error {
	scriptsDir, err := utils.GetSandboxScriptsDir()
	if err != nil {
		return fmt.Errorf("resolve sandbox scripts directory: %w", err)
	}
	scripts := map[string][]byte{
		"frida-bridge.py": embeddedFridaBridge,
	}
	for name, content := range scripts {
		dest := filepath.Join(scriptsDir, name)
		if err := os.WriteFile(dest, content, 0o755); err != nil {
			return fmt.Errorf("write %s: %w", name, err)
		}
	}
	return nil
}

// serveStaticGUI serves the built React GUI from embedded files
func serveStaticGUI(port, apiPort int, serverPtr **http.Server) error {
	// Create filesystem from embedded assets
	subFS, err := fs.Sub(embeddedGUI, "gui/dist")
	if err != nil {
		return fmt.Errorf("failed to create embedded filesystem: %w", err)
	}

	mux := http.NewServeMux()
	apiURL, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", apiPort))
	if err != nil {
		return fmt.Errorf("build API proxy URL: %w", err)
	}
	apiProxy := httputil.NewSingleHostReverseProxy(apiURL)
	mux.Handle("/api/", apiProxy)

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
