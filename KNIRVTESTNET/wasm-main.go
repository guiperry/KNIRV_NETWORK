package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

// initializeEnvironment sets up the KNIRV environment variables
func initializeEnvironment() {
	fmt.Println("🔧 Initializing KNIRV Native Environment...")

	// Set up environment variables
	os.Setenv("KNIRV_ENV", "native-linux")
	os.Setenv("KNIRV_TOOLCHAIN_PATH", "./bin")
	os.Setenv("KNIRV_WORKSPACE", "./workspace")
	os.Setenv("PYTHON_EXECUTABLE", "python3")

	// Add current bin to PATH for KNIRV services
	currentPath := os.Getenv("PATH")
	newPath := "./bin:" + currentPath
	os.Setenv("PATH", newPath)

	fmt.Println("✅ KNIRV native environment initialized successfully")
}

// testToolchainAccess tests access to the toolchain
func testToolchainAccess() {
	fmt.Println("🔍 Testing toolchain access...")

	// Test if Python is available on system
	if _, err := exec.LookPath("python3"); err == nil {
		fmt.Println("✅ Python3 found on system")

		// Try to run a simple Python command
		fmt.Println("🐍 Testing Python execution...")
		cmd := exec.Command("python3", "-c", "print('Python is working!'); import sys; print(f'Python version: {sys.version}')")
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("⚠️ Python execution error: %v\n", err)
		} else {
			fmt.Printf("Python output:\n%s\n", string(output))
		}
	} else {
		fmt.Println("⚠️ Python3 not found on system")
	}

	// Test if bash is available
	if _, err := exec.LookPath("bash"); err == nil {
		fmt.Println("✅ Bash found on system")
	} else {
		fmt.Println("⚠️ Bash not found on system")
	}

	// List current bin directory contents
	fmt.Println("📁 Local bin directory contents:")
	if entries, err := os.ReadDir("./bin"); err == nil {
		for _, entry := range entries {
			fmt.Printf("  - %s\n", entry.Name())
		}
	} else {
		fmt.Printf("⚠️ Could not read ./bin directory: %v\n", err)
	}
}

// Service represents a KNIRV service to be managed by the wizened orchestrator
type Service struct {
	Name    string   `json:"name"`
	Path    string   `json:"path"` // Path to the binary
	Args    []string `json:"args"` // Arguments for the command
	LogFile string   `json:"log_file"`
	Port    string   `json:"port"` // Service port for health checks
}

// main is the entry point for the KNIRV orchestrator running natively on Linux
func main() {
	fmt.Println("🚀 KNIRV Native Orchestrator starting on Linux...")

	// Initialize the environment
	initializeEnvironment()

	// Display the configured environment
	fmt.Printf("Environment: %s\n", os.Getenv("KNIRV_ENV"))
	fmt.Printf("Python executable: %s\n", os.Getenv("PYTHON_EXECUTABLE"))
	fmt.Printf("Go workspace: %s\n", os.Getenv("KNIRV_WORKSPACE"))
	fmt.Printf("PATH: %s\n", os.Getenv("PATH"))

	// Test toolchain access
	testToolchainAccess()

	// Define the KNIRV services to be orchestrated (using host paths)
	// These binaries run natively on the host, orchestrated by this WASM module
	services := []Service{
		{Name: "KNIRV-ORACLE", Path: "./bin/knirvoracle", LogFile: "./logs/knirvoracle.log", Port: "1317"},
		{Name: "KNIRVCHAIN", Path: "./bin/knirvchain", LogFile: "./logs/knirvchain.log", Port: "8080"},
		{Name: "KNIRVGRAPH", Path: "./bin/knirvgraph", LogFile: "./logs/knirvgraph.log", Port: "8081"},
		{Name: "KNIRV-NEXUS", Path: "./bin/knirvnexus", LogFile: "./logs/knirvnexus.log", Port: "8082"},
		{Name: "KNIRV-ROUTER", Path: "./bin/knirvrouter", LogFile: "./logs/knirvrouter.log", Port: "8086"},
	}

	// Create necessary directories
	createDirectories()

	// Start all KNIRV services
	var wg sync.WaitGroup
	for _, s := range services {
		wg.Add(1)
		go startService(s, &wg)
	}

	// Start the main web server for the testnet portal
	go startWebServer()

	// Start health monitoring
	go startHealthMonitoring(services)

	// Wait for shutdown signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	fmt.Println("🛑 Shutting down KNIRV Wizened Orchestrator...")
	// In a production environment, we would gracefully stop all services here
}

// createDirectories ensures all necessary directories exist on host
func createDirectories() {
	dirs := []string{"./logs", "./data", "./config"}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Printf("Error creating host directory %s: %v", dir, err)
		} else {
			log.Printf("✅ Created host directory: %s", dir)
		}
	}
}

// startService launches and monitors a single KNIRV service natively
func startService(s Service, wg *sync.WaitGroup) {
	defer wg.Done()
	fmt.Printf("🚀 Starting KNIRV service: %s\n", s.Name)

	// Create log directory
	logDir := filepath.Dir(s.LogFile)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Printf("Error creating log directory for %s: %v", s.Name, err)
		return
	}

	// Create log file
	logFile, err := os.Create(s.LogFile)
	if err != nil {
		log.Printf("Error creating log file for %s: %v", s.Name, err)
		return
	}
	defer logFile.Close()

	// Start the actual service
	cmd := exec.Command(s.Path, s.Args...)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Stdin = nil // No stdin needed

	// Set working directory to current directory
	cmd.Dir = "."

	if err := cmd.Start(); err != nil {
		log.Printf("❌ Failed to start service %s: %v", s.Name, err)
		return
	}

	log.Printf("✅ Service %s started with PID %d. Logs at %s", s.Name, cmd.Process.Pid, s.LogFile)

	// Wait for the service to complete
	err = cmd.Wait()
	log.Printf("Service %s exited. Error: %v", s.Name, err)
}

// startWebServer starts the main web server for the testnet portal
func startWebServer() {
	// Set up routes for the testnet portal
	http.HandleFunc("/", handleRoot)
	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/api/status", handleStatus)

	// Serve static files
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./assets/"))))
	http.Handle("/agent-developer-portal/", http.StripPrefix("/agent-developer-portal/", http.FileServer(http.Dir("./agent-developer-portal/"))))
	http.Handle("/graphchain-explorer/", http.StripPrefix("/graphchain-explorer/", http.FileServer(http.Dir("./graphchain-explorer/"))))

	port := os.Getenv("PORT")
	if port == "" {
		port = "10000" // Default port for KNIRV testnet
	}

	log.Printf("🌐 KNIRV Testnet Portal starting on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Web server failed: %v", err)
	}
}

// handleRoot serves the main testnet portal page
func handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		http.ServeFile(w, r, "./index.html")
		return
	}
	http.NotFound(w, r)
}

// handleHealth provides health check endpoint for Render
func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, `{"status": "healthy", "server": "knirv-wizened-orchestrator", "environment": "wasm"}`)
}

// handleStatus provides status information about KNIRV services
func handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, `{
		"status": "running",
		"environment": "wizened-wasm",
		"services": {
			"knirvoracle": "running",
			"knirvchain": "running", 
			"knirvgraph": "running",
			"knirvnexus": "running",
			"knirvrouter": "running"
		}
	}`)
}

// startHealthMonitoring monitors the health of all KNIRV services
func startHealthMonitoring(services []Service) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			for _, service := range services {
				go checkServiceHealth(service)
			}
		}
	}
}

// checkServiceHealth performs a health check on a single service
func checkServiceHealth(service Service) {
	if service.Port == "" {
		return // Skip services without defined ports
	}

	url := fmt.Sprintf("http://localhost:%s/health", service.Port)
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Health check failed for %s: %v", service.Name, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		log.Printf("✅ %s health check passed", service.Name)
	} else {
		log.Printf("⚠️ %s health check returned status %d", service.Name, resp.StatusCode)
	}
}
