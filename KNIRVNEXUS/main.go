package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
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
//go:embed all:out
var embeddedFiles embed.FS

// Embed the unified backend binary
//
//go:embed bin/backend_server
var backendBinary []byte

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
}

// EmbeddedFS wraps the embedded filesystem for serving static files
type EmbeddedFS struct {
	files fs.FS
}

// NewEmbeddedFS creates a new embedded filesystem
func NewEmbeddedFS() (*EmbeddedFS, error) {
	// Get the subdirectory containing the actual files
	files, err := fs.Sub(embeddedFiles, "out")
	if err != nil {
		return nil, err
	}

	return &EmbeddedFS{
		files: files,
	}, nil
}

// ServeHTTP implements http.Handler for serving embedded files
func (efs *EmbeddedFS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Clean the path
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		path = "index.html"
	}

	// Try to open the file
	file, err := efs.files.Open(path)
	if err != nil {
		// If file not found, try with .html extension
		if !strings.Contains(path, ".") {
			htmlPath := path + ".html"
			if file, err = efs.files.Open(htmlPath); err != nil {
				// If still not found, serve index.html for SPA routing
				if file, err = efs.files.Open("index.html"); err != nil {
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
	config      *Config
	router      *gin.Engine
	server      *http.Server
	backendCmd  *exec.Cmd
	backendPath string
	tempDir     string
}

// NewNexusApp creates a new KNIRV-NEXUS application
func NewNexusApp(config *Config) (*NexusApp, error) {
	// Set Gin mode
	if config.LogLevel == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	app := &NexusApp{
		config: config,
		router: gin.New(),
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
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
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
	tempDir, err := os.MkdirTemp("", "knirv-nexus-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
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

// setupRoutes configures the application routes
func (app *NexusApp) setupRoutes() error {
	// Create embedded filesystem
	embeddedFS, err := NewEmbeddedFS()
	if err != nil {
		return fmt.Errorf("failed to create embedded filesystem: %w", err)
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

	// API proxy to backend
	api := app.router.Group("/api")
	{
		api.Any("/*path", func(c *gin.Context) {
			// Proxy to backend running on configured port
			backendURL := fmt.Sprintf("http://localhost:%d%s", app.config.BackendPort, c.Request.URL.Path)

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
			client := &http.Client{Timeout: 30 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
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

			// Copy response
			c.Status(resp.StatusCode)
			io.Copy(c.Writer, resp.Body)
		})
	}

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

	return nil
}

// Stop stops the KNIRV-NEXUS application
func (app *NexusApp) Stop() error {
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

// loadConfig loads configuration from file and environment
func loadConfig() (*Config, error) {
	// Parse command line flags
	var (
		configFile = flag.String("config", "", "Path to configuration file")
		testnet    = flag.Bool("testnet", false, "Enable testnet mode")
		port       = flag.Int("port", 0, "Server port (overrides config)")
		host       = flag.String("host", "", "Server host (overrides config)")
	)
	flag.Parse()

	// Set config file if provided
	if *configFile != "" {
		viper.SetConfigFile(*configFile)
	} else {
		// Set config file name and paths
		viper.SetConfigName("nexus")
		viper.SetConfigType("yaml")
		viper.AddConfigPath("./config")
		viper.AddConfigPath(".")
	}

	// Set default values
	viper.SetDefault("host", "0.0.0.0")
	viper.SetDefault("port", 8090)
	viper.SetDefault("backend_port", 8082)
	viper.SetDefault("log_level", "info")
	viper.SetDefault("testnet", false)

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

	// Load configuration
	config, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Log testnet mode if enabled
	if config.Testnet {
		log.Println("🧪 Starting KNIRV-NEXUS in testnet mode")
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

	// Wait for shutdown signal
	<-sigChan
	log.Println("Shutting down...")

	if err := app.Stop(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}

	log.Println("KNIRV-NEXUS stopped")
}
