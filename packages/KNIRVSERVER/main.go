package main

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

// Embed the Next.js build output
//
//go:embed all:frontend/out/*
var embeddedFiles embed.FS

// Embed the unified backend binary
//
//go:embed bin/backend_server
var backendBinary []byte

// Embed the config files
//
//go:embed all:config/*
var configFiles embed.FS

// Embed environment files
//
//go:embed .env.development
var envDevelopment []byte

//go:embed .env.testnet
var envTestnet []byte

// Embed the dashboard dist files
//
//go:embed all:dashboard/dist/*
var dashboardFiles embed.FS

// Version information (set by build flags)
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

// Config represents the application configuration
type Config struct {
	Host        string `mapstructure:"host"`
	Port        int    `mapstructure:"port"`
	BackendPort int    `mapstructure:"backend_port"`
	LogLevel    string `mapstructure:"log_level"`
	Testnet     bool   `mapstructure:"testnet"`
	Dashboard   bool   `mapstructure:"dashboard"`
}

// EmbeddedFS wraps the embedded filesystem for serving static files
type EmbeddedFS struct {
	files fs.FS
}

// NewEmbeddedFS creates a new embedded filesystem
func NewEmbeddedFS() (*EmbeddedFS, error) {
	return &EmbeddedFS{
		files: embeddedFiles,
	}, nil
}

// DashboardFS wraps the embedded dashboard filesystem
type DashboardFS struct {
	files fs.FS
}

// NewDashboardFS creates a new dashboard filesystem
func NewDashboardFS() (*DashboardFS, error) {
	return &DashboardFS{
		files: dashboardFiles,
	}, nil
}

// ServeHTTP implements http.Handler for serving embedded files
func (efs *EmbeddedFS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Clean the path
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		path = "index.html"
	}

	// Prepend the embedded path prefix
	fullPath := "frontend/out/" + path

	// Try to open the file
	file, err := efs.files.Open(fullPath)
	if err != nil {
		// If file not found, try with .html extension
		if !strings.Contains(path, ".") {
			htmlPath := "frontend/out/" + path + ".html"
			if file, err = efs.files.Open(htmlPath); err != nil {
				// If still not found, serve index.html for SPA routing
				if file, err = efs.files.Open("frontend/out/index.html"); err != nil {
					http.NotFound(w, r)
					return
				}
			}
		} else {
			http.NotFound(w, r)
			return
		}
	}
	defer file.Close()

	// Get file info for content type
	stat, err := file.Stat()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Set content type based on file extension
	ext := filepath.Ext(path)
	switch ext {
	case ".html":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	case ".css":
		w.Header().Set("Content-Type", "text/css")
	case ".js":
		w.Header().Set("Content-Type", "application/javascript")
	case ".json":
		w.Header().Set("Content-Type", "application/json")
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".jpg", ".jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	case ".gif":
		w.Header().Set("Content-Type", "image/gif")
	case ".svg":
		w.Header().Set("Content-Type", "image/svg+xml")
	case ".ico":
		w.Header().Set("Content-Type", "image/x-icon")
	}

	// Serve the file
	http.ServeContent(w, r, stat.Name(), stat.ModTime(), file.(io.ReadSeeker))
}

// NexusApp represents the main application
type NexusApp struct {
	config        *Config
	router        *gin.Engine
	server        *http.Server
	backendCmd    *exec.Cmd
	backendPath   string
	dashboardCmd  *exec.Cmd
	tempDir       string
	shutdownToken string
	shutdownChan  chan struct{}
}

// NewNexusApp creates a new KNIRV-NEXUS application
func NewNexusApp(config *Config) (*NexusApp, error) {
	// Set Gin mode
	if config.LogLevel == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Generate a random single-use shutdown token so only the dashboard can trigger shutdown
	tokenBytes := make([]byte, 16)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("failed to generate shutdown token: %w", err)
	}

	app := &NexusApp{
		config:        config,
		router:        gin.New(),
		shutdownToken: hex.EncodeToString(tokenBytes),
		shutdownChan:  make(chan struct{}, 1),
	}

	// Extract backend binary
	if err := app.extractBackend(); err != nil {
		return nil, fmt.Errorf("failed to extract backend: %w", err)
	}

	// Setup middleware
	app.router.Use(gin.Logger())
	app.router.Use(gin.Recovery())

	// CORS middleware
	app.router.Use(func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
		} else {
			c.Header("Access-Control-Allow-Origin", "*")
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")

		reqHeaders := c.GetHeader("Access-Control-Request-Headers")
		if reqHeaders != "" {
			c.Header("Access-Control-Allow-Headers", reqHeaders)
		} else {
			c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Requested-With, X-Auth-Token, X-CSRF-Token")
		}

		c.Header("Access-Control-Expose-Headers", "X-Request-ID, Content-Length, Content-Range")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(200)
			return
		}

		c.Next()
	})

	// Setup routes
	if err := app.setupRoutes(); err != nil {
		return nil, fmt.Errorf("failed to setup routes: %w", err)
	}

	return app, nil
}

// extractBackend extracts the embedded unified backend binary
func (app *NexusApp) extractBackend() error {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "knirv-server-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	// Make temp dir traversable by non-root users (e.g. Electron running as SUDO_USER).
	if err := os.Chmod(tempDir, 0755); err != nil {
		return fmt.Errorf("failed to chmod temp directory: %w", err)
	}
	app.tempDir = tempDir

	// Extract unified backend binary
	app.backendPath = filepath.Join(tempDir, "backend_server")
	file, err := os.Create(app.backendPath)
	if err != nil {
		return fmt.Errorf("failed to create backend file: %w", err)
	}
	defer file.Close()

	if _, err := file.Write(backendBinary); err != nil {
		return fmt.Errorf("failed to write backend binary: %w", err)
	}

	// Make executable
	if err := os.Chmod(app.backendPath, 0755); err != nil {
		return fmt.Errorf("failed to make backend executable: %w", err)
	}

	return nil
}

// extractDashboard extracts the embedded dashboard files
func (app *NexusApp) extractDashboard() error {
	// Create dashboard directory in temp directory
	dashboardDir := filepath.Join(app.tempDir, "dashboard")
	if err := os.MkdirAll(dashboardDir, 0755); err != nil {
		return fmt.Errorf("failed to create dashboard directory: %w", err)
	}

	// Walk through embedded dashboard files
	err := fs.WalkDir(dashboardFiles, "dashboard/dist", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Create subdirectories
		if d.IsDir() {
			relPath := strings.TrimPrefix(path, "dashboard/dist/")
			if relPath != "" {
				fullPath := filepath.Join(dashboardDir, relPath)
				if err := os.MkdirAll(fullPath, 0755); err != nil {
					return fmt.Errorf("failed to create dashboard subdirectory: %w", err)
				}
			}
			return nil
		}

		// Read embedded file
		data, err := dashboardFiles.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read dashboard file %s: %w", path, err)
		}

		// Extract just the filename (remove "dashboard/dist/" prefix)
		filename := strings.TrimPrefix(path, "dashboard/dist/")
		destPath := filepath.Join(dashboardDir, filename)

		// Write to local filesystem
		if err := os.WriteFile(destPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write dashboard file %s: %w", destPath, err)
		}

		return nil
	})

	return err
}

// startDashboard starts the Electron dashboard application
func (app *NexusApp) startDashboard() error {
	log.Println("Starting KNIRV Dashboard...")

	// Resolve electron binary: ELECTRON_PATH env > node_modules relative to exe > node_modules relative to CWD > PATH
	electronPath := os.Getenv("ELECTRON_PATH")
	if electronPath == "" {
		// Build candidate paths to search
		candidates := []string{filepath.Join("dashboard", "node_modules", ".bin", "electron")}
		if exePath, err := os.Executable(); err == nil {
			exeDir := filepath.Dir(exePath)
			// Binary may live at dist/knirv-server; dashboard is at ../dashboard relative to dist/
			candidates = append(candidates,
				filepath.Join(exeDir, "dashboard", "node_modules", ".bin", "electron"),
				filepath.Join(exeDir, "..", "dashboard", "node_modules", ".bin", "electron"),
			)
		}
		for _, c := range candidates {
			if abs, err := filepath.Abs(c); err == nil {
				if _, statErr := os.Stat(abs); statErr == nil {
					electronPath = abs
					break
				}
			}
		}
	}
	if electronPath == "" {
		if p, err := exec.LookPath("electron"); err == nil {
			electronPath = p
		}
	}
	if electronPath == "" {
		log.Printf("Warning: Electron not found. Dashboard will not be started.")
		log.Printf("Run 'make dashboard-build' or set ELECTRON_PATH to the electron binary.")
		return nil
	}

	// Extract dashboard files
	if err := app.extractDashboard(); err != nil {
		return fmt.Errorf("failed to extract dashboard: %w", err)
	}

	dashboardDir := filepath.Join(app.tempDir, "dashboard")

	// Set the KNIRV server URL for the dashboard
	serverUrl := fmt.Sprintf("http://localhost:%d", app.config.Port)

	// Launch Electron with the dashboard.
	// Run electron with the explicit entry point so no package.json is needed.
	// If running under sudo, launch as the original user so Electron has access
	// to the user's display server. We use a login shell to load the user's
	// profile (nvm, etc.) so that node is on PATH.
	mainJsPath := filepath.Join(dashboardDir, "main.js")

	sudoUser := os.Getenv("SUDO_USER")
	if sudoUser != "" {
		// Run as the original user with --set-home so $HOME is correct.
		// Explicitly source ~/.nvm/nvm.sh so node is on PATH regardless of
		// whether nvm is in .bashrc (interactive) or .bash_profile (login).
		shellCmd := fmt.Sprintf(
			`. "$HOME/.nvm/nvm.sh" 2>/dev/null; `+
				`export XDG_RUNTIME_DIR="/run/user/$(id -u)"; `+
				`export DBUS_SESSION_BUS_ADDRESS="unix:path=/run/user/$(id -u)/bus"; `+
				`KNIRV_SERVER_URL=%s KNIRV_SHUTDOWN_TOKEN=%s KNIRV_SERVER_PORT=%s exec %s %s`,
			shellEscape(serverUrl), shellEscape(app.shutdownToken),
			shellEscape(fmt.Sprintf("%d", app.config.Port)),
			shellEscape(electronPath), shellEscape(mainJsPath),
		)
		app.dashboardCmd = exec.Command("sudo", "-u", sudoUser, "--set-home", "--", "bash", "-c", shellCmd)
	} else {
		app.dashboardCmd = exec.Command(electronPath, mainJsPath)
		app.dashboardCmd.Env = append(os.Environ(),
			fmt.Sprintf("KNIRV_SERVER_URL=%s", serverUrl),
			fmt.Sprintf("KNIRV_SHUTDOWN_TOKEN=%s", app.shutdownToken),
			fmt.Sprintf("KNIRV_SERVER_PORT=%d", app.config.Port),
		)
	}
	app.dashboardCmd.Dir = dashboardDir
	app.dashboardCmd.Stdout = os.Stdout
	app.dashboardCmd.Stderr = os.Stderr

	if err := app.dashboardCmd.Start(); err != nil {
		return fmt.Errorf("failed to start dashboard: %w", err)
	}

	log.Printf("KNIRV Dashboard started (PID: %d)", app.dashboardCmd.Process.Pid)
	return nil
}

// stopDashboard stops the Electron dashboard application
func (app *NexusApp) stopDashboard() {
	if app.dashboardCmd != nil && app.dashboardCmd.Process != nil {
		log.Printf("Stopping KNIRV Dashboard (PID: %d)", app.dashboardCmd.Process.Pid)
		app.dashboardCmd.Process.Signal(syscall.SIGTERM)
		app.dashboardCmd.Wait()
	}
}

// getAppDataDir returns the application data directory path
func getAppDataDir() (string, error) {
	// Try XDG_DATA_HOME first
	if xdgDataHome := os.Getenv("XDG_DATA_HOME"); xdgDataHome != "" {
		return filepath.Join(xdgDataHome, "knirvserver"), nil
	}

	// Fallback to ~/.local/share/knirvserver
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	return filepath.Join(homeDir, ".local", "share", "knirvserver"), nil
}

// extractEnvFile extracts the embedded environment file based on the specified environment
// shellEscape wraps a string in single quotes for safe use in a shell command.
func shellEscape(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func extractEnvFile(environment string) error {
	var envData []byte
	var envName string

	switch environment {
	case "development":
		envData = envDevelopment
		envName = ".env.development"
	case "testnet":
		envData = envTestnet
		envName = ".env.testnet"
	case "production":
		// For production, we expect the .env file to be provided externally
		// or configured through environment variables
		log.Println("Production mode: skipping .env file extraction (use external .env or environment variables)")
		return nil
	default:
		return fmt.Errorf("unknown environment: %s (must be development, testnet, or production)", environment)
	}

	// Extract to current working directory so the application can find it
	envPath := filepath.Join(".", ".env")
	if err := os.WriteFile(envPath, envData, 0644); err != nil {
		return fmt.Errorf("failed to write %s to %s: %w", envName, envPath, err)
	}

	log.Printf("Extracted %s environment file to %s", environment, envPath)
	return nil
}

// extractConfigFiles extracts embedded config files to the application data directory
func extractConfigFiles() error {
	// Get application data directory
	appDataDir, err := getAppDataDir()
	if err != nil {
		return err
	}

	// Create config directory in app data dir
	configDir := filepath.Join(appDataDir, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Walk through embedded config files
	err = fs.WalkDir(configFiles, "config", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Read embedded file
		data, err := configFiles.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read embedded file %s: %w", path, err)
		}

		// Extract just the filename (remove "config/" prefix)
		filename := filepath.Base(path)
		destPath := filepath.Join(configDir, filename)

		// Write to local filesystem
		if err := os.WriteFile(destPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write config file %s: %w", destPath, err)
		}

		log.Printf("Extracted config file: %s", destPath)
		return nil
	})

	return err
}

// setupRoutes configures the application routes
func (app *NexusApp) setupRoutes() error {
	// Create embedded filesystem
	embeddedFS, err := NewEmbeddedFS()
	if err != nil {
		return fmt.Errorf("failed to create embedded filesystem: %w", err)
	}

	// Debug: List embedded files
	log.Println("DEBUG: Listing embedded files:")
	if rdf, ok := embeddedFS.files.(fs.ReadDirFS); ok {
		entries, err := rdf.ReadDir(".")
		if err != nil {
			log.Printf("DEBUG: Error reading embedded dir: %v", err)
		} else {
			log.Printf("DEBUG: Found %d embedded entries", len(entries))
			for _, entry := range entries {
				log.Printf("DEBUG: Embedded file: %s (dir: %v)", entry.Name(), entry.IsDir())
			}
		}
	} else {
		log.Println("DEBUG: Embedded FS does not support ReadDir")
	}

	// Health check endpoint
	app.router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":     "healthy",
			"version":    Version,
			"build_time": BuildTime,
			"git_commit": GitCommit,
			"timestamp":  time.Now().UTC().Format(time.RFC3339),
		})
	})

	// Version endpoint
	app.router.GET("/version", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"version":    Version,
			"build_time": BuildTime,
			"git_commit": GitCommit,
		})
	})

	// Shutdown endpoint — only accepts requests bearing the per-run token
	app.router.POST("/shutdown", func(c *gin.Context) {
		if c.GetHeader("X-Shutdown-Token") != app.shutdownToken {
			c.JSON(http.StatusForbidden, gin.H{"error": "invalid token"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "shutting down"})
		select {
		case app.shutdownChan <- struct{}{}:
		default:
		}
	})

	// API proxy to backend
	api := app.router.Group("/api")
	{
		api.Any("/*path", func(c *gin.Context) {
			// Proxy to backend running on configured port
			// Use the full original URL path and query string
			backendURL := fmt.Sprintf("http://localhost:%d%s", app.config.BackendPort, c.Request.RequestURI)

			// Create proxy request
			req, err := http.NewRequest(c.Request.Method, backendURL, c.Request.Body)
			if err != nil {
				c.JSON(500, gin.H{"error": "Failed to create proxy request"})
				return
			}

			// Copy headers
			for key, values := range c.Request.Header {
				for _, value := range values {
					req.Header.Add(key, value)
				}
			}

			// Make request to backend
			client := &http.Client{
				Timeout: 60 * time.Second,
				// Do not follow redirects automatically
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					return http.ErrUseLastResponse
				},
			}
			resp, err := client.Do(req)
			if err != nil {
				log.Printf("Proxy error: %v", err)
				c.JSON(500, gin.H{"error": "Backend service unavailable"})
				return
			}
			defer resp.Body.Close()

			// Copy response headers
			for key, values := range resp.Header {
				for _, value := range values {
					c.Header(key, value)
				}
			}

			// Copy response status and body
			c.Status(resp.StatusCode)
			io.Copy(c.Writer, resp.Body)
		})
	}

	// WebSocket proxy — must be registered before NoRoute so the upgrade
	// request reaches the backend instead of being served as a static file.
	// httputil.ReverseProxy handles the 101 Switching Protocols upgrade by
	// hijacking the underlying net.Conn, which works through Gin's wrapper.
	backendWS, _ := url.Parse(fmt.Sprintf("http://localhost:%d", app.config.BackendPort))
	wsProxy := httputil.NewSingleHostReverseProxy(backendWS)
	wsProxy.FlushInterval = -1 // flush immediately; required for streaming / WebSocket
	wsProxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		// The error handler is only invoked before the connection is hijacked,
		// so it is safe to write an HTTP error response here.
		log.Printf("WebSocket proxy error for %s: %v", r.URL.Path, err)
		http.Error(w, "WebSocket backend unavailable", http.StatusBadGateway)
	}

	wsHandler := func(c *gin.Context) {
		wsProxy.ServeHTTP(c.Writer, c.Request)
	}
	app.router.GET("/ws", wsHandler)
	app.router.GET("/ws/*path", wsHandler)

	// Serve embedded frontend files
	app.router.NoRoute(func(c *gin.Context) {
		embeddedFS.ServeHTTP(c.Writer, c.Request)
	})

	return nil
}

// startBackend starts the embedded unified backend service
func (app *NexusApp) startBackend() error {
	log.Printf("Starting unified backend service on port %d...", app.config.BackendPort)

	// Pass the config file path used by the wrapper to the backend.
	// This ensures the backend uses the same configuration.
	configFile := viper.ConfigFileUsed()
	if configFile != "" {
		log.Printf("Passing config file to backend: %s", configFile)
		app.backendCmd = exec.Command(app.backendPath, "--config", configFile)
	} else {
		app.backendCmd = exec.Command(app.backendPath)
	}

	// Set environment variables for backend
	env := append(os.Environ(),
		fmt.Sprintf("KNIRV_API_PORT=%d", app.config.BackendPort),
		"KNIRV_API_HOST=127.0.0.1",
		"KNIRV_SECURITY_JWT_SECRET=testnet-jwt-secret-change-this-in-production",
		"KNIRV_JWT_SECRET=testnet-jwt-secret-change-this-in-production",
	)

	// Add testnet environment variable if enabled
	if app.config.Testnet {
		env = append(env,
			"KNIRV_TESTNET=true",
			"KNIRV_SECURITY_AUTH_REQUIRED=false",
			"KNIRV_MODE=headless",
		)
		log.Println("Starting backend in testnet mode with simplified security")
	}

	app.backendCmd.Env = env
	app.backendCmd.Stdout = os.Stdout
	app.backendCmd.Stderr = os.Stderr

	if err := app.backendCmd.Start(); err != nil {
		return fmt.Errorf("failed to start unified backend: %w", err)
	}

	log.Printf("Unified backend started (PID: %d)", app.backendCmd.Process.Pid)

	// Wait for backend to be ready
	time.Sleep(3 * time.Second)

	return nil
}

// stopBackend stops the unified backend service
func (app *NexusApp) stopBackend() {
	if app.backendCmd != nil && app.backendCmd.Process != nil {
		log.Printf("Stopping unified backend (PID: %d)", app.backendCmd.Process.Pid)
		app.backendCmd.Process.Signal(syscall.SIGTERM)
		app.backendCmd.Wait()
	}
}

// Start starts the KNIRV-NEXUS application
func (app *NexusApp) Start() error {
	// Start backend first
	if err := app.startBackend(); err != nil {
		return err
	}

	// Create HTTP server
	app.server = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", app.config.Host, app.config.Port),
		Handler:      app.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Starting KNIRV-NEXUS on %s:%d", app.config.Host, app.config.Port)
		if err := app.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for server to be ready
	time.Sleep(2 * time.Second)

	// Start dashboard if enabled
	if app.config.Dashboard {
		if err := app.startDashboard(); err != nil {
			log.Printf("Warning: Failed to start dashboard: %v", err)
		}
	}

	return nil
}

// Stop stops the KNIRV-NEXUS application
func (app *NexusApp) Stop() error {
	// Stop dashboard first
	app.stopDashboard()

	// Stop HTTP server
	if app.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := app.server.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}

	// Stop backend
	app.stopBackend()

	// Clean up temp directory
	if app.tempDir != "" {
		os.RemoveAll(app.tempDir)
	}

	return nil
}

// isFlagSet checks if a flag was explicitly set on the command line
func isFlagSet(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

// loadConfig loads configuration from file and environment
func loadConfig() (*Config, error) {
	// Parse command line flags
	var (
		configFile  = flag.String("config", "", "Path to configuration file")
		testnet     = flag.Bool("testnet", false, "Enable testnet mode")
		dashboard   = flag.Bool("dashboard", false, "Enable the KNIRV Dashboard GUI")
		environment = flag.String("env", "production", "Environment: development, testnet, or production")
		port        = flag.Int("port", 0, "Server port (overrides config)")
		host        = flag.String("host", "", "Server host (overrides config)")
	)
	flag.Parse()

	// Check for KNIRV_ENV environment variable if --env flag not explicitly set
	if !isFlagSet("env") {
		if envVar := os.Getenv("KNIRV_ENV"); envVar != "" {
			*environment = envVar
			log.Printf("Using environment from KNIRV_ENV: %s", *environment)
		}
	}

	// Extract environment file based on flag or environment variable
	if err := extractEnvFile(*environment); err != nil {
		log.Printf("Warning: Failed to extract environment file: %v", err)
	}

	// Set config file if provided
	if *configFile != "" {
		viper.SetConfigFile(*configFile)
	} else {
		// Set config file name based on environment
		viper.SetConfigName(*environment)
		viper.SetConfigType("yaml")

		// Add app data directory config path first (highest priority)
		if appDataDir, err := getAppDataDir(); err == nil {
			viper.AddConfigPath(filepath.Join(appDataDir, "config"))
		}

		// Add local paths as fallback
		viper.AddConfigPath("./config")
		viper.AddConfigPath(".")
	}

	// Set default values
	viper.SetDefault("host", "0.0.0.0")
	viper.SetDefault("port", 8090)
	viper.SetDefault("backend_port", 8082)
	viper.SetDefault("log_level", "info")
	viper.SetDefault("testnet", false)
	viper.SetDefault("dashboard", false)

	// Enable environment variable support
	viper.AutomaticEnv()
	viper.SetEnvPrefix("NEXUS")

	// Try to read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		log.Println("No config file found, using defaults and environment variables")
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Override with command line flags
	if *testnet {
		config.Testnet = true
	}
	if *dashboard {
		config.Dashboard = true
	}
	if *port != 0 {
		config.Port = *port
	}
	if *host != "" {
		config.Host = *host
	}

	return &config, nil
}

func main() {
	// Print version information
	fmt.Printf("KNIRV-NEXUS v%s (built %s, commit %s)\n", Version, BuildTime, GitCommit)

	// Extract embedded config files to app data directory
	if err := extractConfigFiles(); err != nil {
		log.Printf("Warning: Failed to extract config files: %v", err)
	}

	// Load configuration
	config, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Log testnet mode if enabled
	if config.Testnet {
		log.Println("🧪 Starting KNIRV-NEXUS in testnet mode")
	}

	// Log dashboard mode if enabled
	if config.Dashboard {
		log.Println("🖥️  Starting KNIRV-NEXUS with Dashboard GUI")
	}

	// Create application
	app, err := NewNexusApp(config)
	if err != nil {
		log.Fatalf("Failed to create application: %v", err)
	}

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start application
	if err := app.Start(); err != nil {
		log.Fatalf("Failed to start application: %v", err)
	}

	// Wait for shutdown signal (OS signal or dashboard /shutdown request)
	select {
	case <-sigChan:
	case <-app.shutdownChan:
		log.Println("Shutdown requested by dashboard")
	}
	log.Println("Shutting down...")

	if err := app.Stop(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}

	log.Println("KNIRV-NEXUS stopped")
}
