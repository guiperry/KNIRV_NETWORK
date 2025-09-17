package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"KNIRVORACLE/config"
	"KNIRVORACLE/utils"

	"github.com/skratchdot/open-golang/open"
)

// Global variables to store the main context cancel function and wait group
var mainCancelFunc sync.Map
var mainWaitGroup sync.Map

// GUI represents the alternative GUI using browser redirection
type GUI struct {
	blockchain       *BlockchainStruct
	db               *LevelDB
	discoveryMgr     *DiscoveryManager
	p2pConsensusMgr  *P2PConsensusManager
	cfg              *config.Config
	paymentProcessor *PaymentProcessor
	role             config.Role
	wallet           *Wallet
	apiServer        *http.Server
	apiPort          uint64
	apiServerReady   chan struct{}
	walletServerURL  string
	walletPortChan   <-chan uint64
	walletPort       uint64
	cancelFunc       context.CancelFunc
	wg               *sync.WaitGroup
}

// NewGUI creates a new GUI instance for the browser-based GUI
func NewGUI(blockchain *BlockchainStruct, db *LevelDB, discoveryMgr *DiscoveryManager, p2pConsensusMgr *P2PConsensusManager, cfg *config.Config, paymentProcessor *PaymentProcessor, role config.Role, wallet *Wallet, walletPortChan <-chan uint64, cancelFunc context.CancelFunc, wg *sync.WaitGroup) *GUI {
	gui := &GUI{
		blockchain:       blockchain,
		db:               db,
		discoveryMgr:     discoveryMgr,
		p2pConsensusMgr:  p2pConsensusMgr,
		cfg:              cfg,
		paymentProcessor: paymentProcessor,
		role:             role,
		wallet:           wallet,
		apiPort:          cfg.AltGUIPort,
		apiServerReady:   make(chan struct{}),
		walletPortChan:   walletPortChan,
		walletPort:       0,
		walletServerURL:  "",
		cancelFunc:       cancelFunc,
		wg:               wg,
	}

	// Start goroutine to listen for wallet port changes
	go func() {
		for port := range walletPortChan {
			gui.walletPort = port
			gui.walletServerURL = fmt.Sprintf("http://localhost:%d", port)
			log.Printf("Updated wallet server URL to: %s", gui.walletServerURL)
		}
	}()

	if gui.apiPort == 0 {
		log.Println("Warning: AltGUIPort is 0 in config, defaulting to 3000 for embedded UI server.")
		gui.apiPort = 3000 // Fallback if AltGUIPort was somehow 0
	}
	return gui
}

// Run starts the alternative GUI
func (g *GUI) Run() {
	// Lock this goroutine to the main OS thread
	runtime.LockOSThread()

	// Start the API server in a goroutine
	go g.startAPIServer()

	// Wait for the API server to be ready
	<-g.apiServerReady

	// Check if we're in Root mode and should redirect to Developer Portal
	if g.role == config.Root && g.cfg.NodeJSServices.Enabled && g.cfg.NodeJSServices.DeveloperPortal.Enabled {
		portalURL := fmt.Sprintf("http://localhost:%d", g.cfg.NodeJSServices.DeveloperPortal.HTTPPort)
		log.Printf("Root node detected with Developer Portal enabled. Opening browser to %s", portalURL)

		// Open the browser to the Developer Portal
		if err := open.Run(portalURL); err != nil {
			log.Printf("Failed to open browser to Developer Portal: %v", err)
		}
	} else {
		// For non-Root nodes or if Developer Portal is disabled, open the local API server
		localURL := fmt.Sprintf("http://localhost:%d", g.apiPort)
		log.Printf("Opening browser to local API server at %s", localURL)

		// Open the browser to the local API server
		if err := open.Run(localURL); err != nil {
			log.Printf("Failed to open browser: %v", err)
		}
	}

	// Keep the application running
	select {}
}

// startAPIServer starts the HTTP API server
func (g *GUI) startAPIServer() {
	router := http.NewServeMux()

	// Add API routes here
	// Health endpoint for both /api/health and /health
	healthHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	}
	router.HandleFunc("/api/health", healthHandler)
	router.HandleFunc("/health", healthHandler)

	// Info endpoint
	router.HandleFunc("/info", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		info := map[string]string{
			"version":  "1.0.0",
			"nodeName": "KNIRVORACLE Node",
			"role":     string(g.role),
		}
		json.NewEncoder(w).Encode(info)
	})

	// Redirect to Developer Portal for Root nodes
	if g.role == config.Root && g.cfg.NodeJSServices.Enabled && g.cfg.NodeJSServices.DeveloperPortal.Enabled {
		portalURL := fmt.Sprintf("http://localhost:%d", g.cfg.NodeJSServices.DeveloperPortal.HTTPPort)
		router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, portalURL, http.StatusTemporaryRedirect)
		})
	} else {
		// Serve a simple HTML page for non-Root nodes
		router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			html := `
			<!DOCTYPE html>
			<html>
			<head>
				<title>KNIRVORACLE Node</title>
				<style>
					body { font-family: Arial, sans-serif; margin: 0; padding: 20px; background-color: #f5f5f5; }
					.container { max-width: 800px; margin: 0 auto; background-color: white; padding: 20px; border-radius: 5px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
					h1 { color: #333; }
					.info { margin-top: 20px; }
					.info p { margin: 5px 0; }
					.info strong { display: inline-block; width: 100px; }
				</style>
			</head>
			<body>
				<div class="container">
					<h1>KNIRVORACLE Node</h1>
					<div class="info">
						<p><strong>Role:</strong> ` + string(g.role) + `</p>
						<p><strong>Chain ID:</strong> ` + g.cfg.ChainID + `</p>
						<p><strong>Status:</strong> Running</p>
					</div>
					<div class="info">
						<p>This is a simplified interface for the KNIRVORACLE node.</p>
						<p>For the full Developer Portal experience, please connect to a Root node.</p>
					</div>
				</div>
			</body>
			</html>
			`
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(html))
		})
	}

	// --- Find an available port for the API server ---
	initialAPIPort := g.apiPort
	currentAPIPort := initialAPIPort
	maxPortAttempts := 100 // Prevent infinite loop

	for i := 0; i < maxPortAttempts; i++ {
		if utils.IsPortAvailable(currentAPIPort) {
			g.apiPort = currentAPIPort // Update to the available port
			log.Printf("AltGUI API Server will use port %d", g.apiPort)
			break
		}
		log.Printf("AltGUI API Port %d is in use, trying next port %d...", currentAPIPort, currentAPIPort+1)
		currentAPIPort++
		if i == maxPortAttempts-1 {
			log.Fatalf("Failed to find an available port for AltGUI API server after %d attempts starting from %d.", maxPortAttempts, initialAPIPort)
		}
	}
	// --- End port finding ---

	// Start the server with appropriate CORS middleware
	var handler http.Handler = router
	if g.role == config.Root && g.cfg.NodeJSServices.Enabled && g.cfg.NodeJSServices.DeveloperPortal.Enabled {
		handler = devPortalCorsMiddleware(router)
	} else {
		handler = corsMiddleware(router)
	}

	g.apiServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", g.apiPort),
		Handler: handler,
	}

	// Start the server in a goroutine
	go func() {
		log.Printf("Starting API server on port %d", g.apiPort)
		if err := g.apiServer.ListenAndServe(); err != http.ErrServerClosed {
			log.Printf("API server error on port %d: %v", g.apiPort, err)
		} else {
			log.Printf("API server on port %d shut down.", g.apiPort)
		}
	}()

	// Goroutine to check server readiness and signal
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		timeout := time.After(5 * time.Second) // 5-second timeout for server to start

		for {
			select {
			case <-ticker.C:
				conn, err := net.DialTimeout("tcp", g.apiServer.Addr, 200*time.Millisecond)
				if err == nil {
					conn.Close()
					// Ensure channel is closed only once
					select {
					case <-g.apiServerReady: // Already closed
					default:
						close(g.apiServerReady)
					}
					log.Printf("API server confirmed listening on %s", g.apiServer.Addr)
					return
				}
			case <-timeout:
				log.Printf("Timeout waiting for API server to start on %s.", g.apiServer.Addr)
				return
			}
		}
	}()
}

// Cleanup performs cleanup when the GUI is closed
func (g *GUI) Cleanup() {
	// Shutdown the API server
	if g.apiServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		g.apiServer.Shutdown(ctx)
	}
}

// InitializeDevPortalGUI is the entry point for the alternative GUI
func InitializeDevPortalGUI(bc *BlockchainStruct, db *LevelDB, discoveryMgr *DiscoveryManager, p2pConsensusMgr *P2PConsensusManager, cfg *config.Config, paymentProcessor *PaymentProcessor, role config.Role, wallet *Wallet, ws *WalletServer, chromemManager *ChromemManager) *GUI {
	var portChan <-chan uint64
	if ws != nil {
		portChan = ws.portChan
	}

	// Get the cancel function and wait group from the main function
	var val any
	mainCancelFunc.Load(&val)
	cancelFunc, ok := val.(context.CancelFunc)
	if !ok {
		log.Println("Warning: Could not get cancel function from main, creating a new one")
		var dummyCancel context.CancelFunc
		_, dummyCancel = context.WithCancel(context.Background())
		cancelFunc = dummyCancel
	}

	var val2 any
	mainWaitGroup.Load(&val2)
	wg, ok := val2.(*sync.WaitGroup)
	if !ok {
		log.Println("Warning: Could not get wait group from main, creating a new one")
		wg = &sync.WaitGroup{}
	}

	// If this is a Root node, launch the Developer Portal
	if role == config.Root && cfg.NodeJSServices.Enabled && cfg.NodeJSServices.DeveloperPortal.Enabled {
		// Import the launch function from the agent-developer-portal package
		if err := LaunchDeveloperPortal(cfg); err != nil {
			log.Printf("Failed to launch Developer Portal: %v", err)
		}
	}

	gui := NewGUI(bc, db, discoveryMgr, p2pConsensusMgr, cfg, paymentProcessor, role, wallet, portChan, cancelFunc, wg)
	go gui.Run()

	return gui
}

// devPortalCorsMiddleware adds CORS headers to all responses for the developer portal
func devPortalCorsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Call the next handler
		next.ServeHTTP(w, r)
	})
}

// LaunchDeveloperPortal starts the Developer Portal Node.js service
// This function should be called only from the Root node
func LaunchDeveloperPortal(cfg *config.Config) error {
	if !cfg.IsRoot {
		return fmt.Errorf("developer portal can only be launched from a Root node")
	}

	if !cfg.NodeJSServices.Enabled || !cfg.NodeJSServices.DeveloperPortal.Enabled {
		log.Println("Developer Portal is disabled in configuration")
		return nil
	}

	scriptPath := cfg.NodeJSServices.DeveloperPortal.ScriptPath
	if scriptPath == "" {
		scriptPath = "agent-developer-portal/server.js"
	}

	// Resolve the script path relative to the executable
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %v", err)
	}
	execDir := filepath.Dir(execPath)
	scriptFullPath := filepath.Join(execDir, scriptPath)

	// Check if the script exists
	if _, err := os.Stat(scriptFullPath); os.IsNotExist(err) {
		return fmt.Errorf("developer portal script not found at %s", scriptFullPath)
	}

	// Prepare environment variables
	env := os.Environ()
	env = append(env, fmt.Sprintf("HTTP_API_PORT=%d", cfg.NodeJSServices.DeveloperPortal.HTTPPort))
	env = append(env, fmt.Sprintf("API_KEY=%s", cfg.NodeJSServices.DeveloperPortal.APIKey))
	env = append(env, fmt.Sprintf("CHAIN_ID=%s", cfg.ChainID))
	env = append(env, fmt.Sprintf("NODE_ENV=%s", "production"))

	// Start the Node.js process
	cmd := exec.Command("node", scriptFullPath)
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Start the process in the background
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start Developer Portal: %v", err)
	}

	log.Printf("Developer Portal started with PID %d on port %d", cmd.Process.Pid, cfg.NodeJSServices.DeveloperPortal.HTTPPort)

	// Don't wait for the process to complete
	go func() {
		if err := cmd.Wait(); err != nil {
			if strings.Contains(err.Error(), "signal: killed") {
				log.Println("Developer Portal process was terminated")
			} else {
				log.Printf("Developer Portal process exited with error: %v", err)
			}
		} else {
			log.Println("Developer Portal process exited normally")
		}
	}()

	return nil
}
