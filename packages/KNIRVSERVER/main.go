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
	"net"
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

// Embed the KNIRVGATEWAY binary for P2P TURN/Tunnel services
//
//go:embed bin/knirvgateway
var knirvgatewayBinary []byte

// Embed the KNIRVGRAPH binary for knowledge graph and graphchain
//
//go:embed bin/knirvgraph
var knirvgraphBinary []byte

// Embed the KNIRVCHAIN binary for blockchain and mining
//
//go:embed bin/knirvchain
var knirvchainBinary []byte

// Embed the root key (present only on root-node builds; absent on client builds).
// The build tag "rootnode" is used to conditionally include this file via
// the go:embed directive below.  If the file is absent the byte slice stays nil.
//
//go:embed bin/root.key
var rootKeyBytes []byte

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

// Embed the desktop dist files
//
//go:embed all:desktop/dist/*
var desktopFiles embed.FS

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
	Desktop     bool   `mapstructure:"desktop"`
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

// DesktopFS wraps the embedded desktop filesystem
type DesktopFS struct {
	files fs.FS
}

// NewDesktopFS creates a new desktop filesystem
func NewDesktopFS() (*DesktopFS, error) {
	return &DesktopFS{
		files: desktopFiles,
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
	desktopCmd    *exec.Cmd
	tempDir       string
	shutdownToken string
	shutdownChan  chan struct{}
}

// NewNexusApp creates a new KNIRV-SERVER application
func NewNexusApp(config *Config) (*NexusApp, error) {
	// Set Gin mode
	if config.LogLevel == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Generate a random single-use shutdown token so only the desktop can trigger shutdown
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

// extractBinaries extracts all embedded binaries to the app data bin directory.
// Returns the bin directory path.
func extractBinaries() (string, error) {
	appDataDir, err := getAppDataDir()
	if err != nil {
		return "", fmt.Errorf("failed to get app data directory: %w", err)
	}

	binDir := filepath.Join(appDataDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create bin directory: %w", err)
	}

	type entry struct {
		name string
		data []byte
	}
	bins := []entry{
		{"backend_server", backendBinary},
		{"knirvgateway", knirvgatewayBinary},
		{"knirvgraph", knirvgraphBinary},
		{"knirvchain", knirvchainBinary},
	}

	for _, b := range bins {
		dest := filepath.Join(binDir, b.name)
		if err := os.WriteFile(dest, b.data, 0755); err != nil {
			return "", fmt.Errorf("failed to extract %s: %w", b.name, err)
		}
		log.Printf("Extracted %s to %s", b.name, dest)
	}

	return binDir, nil
}

// extractBackend extracts all embedded binaries to the app data directory and
// sets app.backendPath. A small temp directory is still kept for ephemeral
// artefacts (e.g. desktop files).
func (app *NexusApp) extractBackend() error {
	// Create temp directory for ephemeral artefacts (desktop assets, etc.)
	tempDir, err := os.MkdirTemp("", "knirv-server-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	// Make temp dir traversable by non-root users (e.g. Electron running as SUDO_USER).
	if err := os.Chmod(tempDir, 0755); err != nil {
		return fmt.Errorf("failed to chmod temp directory: %w", err)
	}
	app.tempDir = tempDir

	// Extract all binaries to the persistent app data bin directory.
	binDir, err := extractBinaries()
	if err != nil {
		return fmt.Errorf("failed to extract binaries: %w", err)
	}

	app.backendPath = filepath.Join(binDir, "backend_server")
	return nil
}

// extractDesktop extracts the embedded desktop files
func (app *NexusApp) extractDesktop() error {
	// Create desktop directory in temp directory
	desktopDir := filepath.Join(app.tempDir, "desktop")
	if err := os.MkdirAll(desktopDir, 0755); err != nil {
		return fmt.Errorf("failed to create desktop directory: %w", err)
	}

	// Walk through embedded desktop files
	err := fs.WalkDir(desktopFiles, "desktop/dist", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Create subdirectories
		if d.IsDir() {
			relPath := strings.TrimPrefix(path, "desktop/dist/")
			if relPath != "" {
				fullPath := filepath.Join(desktopDir, relPath)
				if err := os.MkdirAll(fullPath, 0755); err != nil {
					return fmt.Errorf("failed to create desktop subdirectory: %w", err)
				}
			}
			return nil
		}

		// Read embedded file
		data, err := desktopFiles.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read desktop file %s: %w", path, err)
		}

		// Extract just the filename (remove "desktop/dist/" prefix)
		filename := strings.TrimPrefix(path, "desktop/dist/")
		destPath := filepath.Join(desktopDir, filename)

		// Write to local filesystem
		if err := os.WriteFile(destPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write desktop file %s: %w", destPath, err)
		}

		return nil
	})

	return err
}

// startDesktop starts the Electron desktop application
func (app *NexusApp) startDesktop() error {
	log.Println("Starting KNIRV Desktop...")

	// Resolve electron binary: ELECTRON_PATH env > node_modules relative to exe > node_modules relative to CWD > PATH
	electronPath := os.Getenv("ELECTRON_PATH")
	if electronPath == "" {
		// Build candidate paths to search
		candidates := []string{filepath.Join("desktop", "node_modules", ".bin", "electron")}
		if exePath, err := os.Executable(); err == nil {
			exeDir := filepath.Dir(exePath)
			// Binary may live at dist/knirv-server; desktop is at ../desktop relative to dist/
			candidates = append(candidates,
				filepath.Join(exeDir, "desktop", "node_modules", ".bin", "electron"),
				filepath.Join(exeDir, "..", "desktop", "node_modules", ".bin", "electron"),
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
		log.Printf("Warning: Electron not found. Desktop will not be started.")
		log.Printf("Run 'make desktop-build' or set ELECTRON_PATH to the electron binary.")
		return nil
	}

	// Extract desktop files
	if err := app.extractDesktop(); err != nil {
		return fmt.Errorf("failed to extract desktop: %w", err)
	}

	desktopDir := filepath.Join(app.tempDir, "desktop")

	// Set the KNIRV server URL for the desktop
	serverUrl := fmt.Sprintf("http://localhost:%d", app.config.Port)

	// Launch Electron with the desktop.
	// Run electron with the explicit entry point so no package.json is needed.
	// If running under sudo, launch as the original user so Electron has access
	// to the user's display server. We use a login shell to load the user's
	// profile (nvm, etc.) so that node is on PATH.
	mainJsPath := filepath.Join(desktopDir, "main.js")

	sudoUser := os.Getenv("SUDO_USER")
	if sudoUser != "" {
		// Run as the original user with --set-home so $HOME is correct.
		// Explicitly source ~/.nvm/nvm.sh so node is on PATH regardless of
		// whether nvm is in .bashrc (interactive) or .bash_profile (login).
		// Disable Vulkan/GPU via env vars to suppress driver warnings on headless systems.
		shellCmd := fmt.Sprintf(
			`. "$HOME/.nvm/nvm.sh" 2>/dev/null; `+
				`export XDG_RUNTIME_DIR="/run/user/$(id -u)"; `+
				`export DBUS_SESSION_BUS_ADDRESS="unix:path=/run/user/$(id -u)/bus"; `+
				`export VK_ICD_FILENAMES=""; `+
				`export ELECTRON_OZONE_PLATFORM_HINT=auto; `+
				`KNIRV_SERVER_URL=%s KNIRV_SHUTDOWN_TOKEN=%s KNIRV_SERVER_PORT=%s exec %s --disable-gpu --disable-software-rasterizer %s`,
			shellEscape(serverUrl), shellEscape(app.shutdownToken),
			shellEscape(fmt.Sprintf("%d", app.config.Port)),
			shellEscape(electronPath), shellEscape(mainJsPath),
		)
		app.desktopCmd = exec.Command("sudo", "-u", sudoUser, "--set-home", "--", "bash", "-c", shellCmd)
	} else {
		app.desktopCmd = exec.Command(electronPath, "--disable-gpu", "--disable-software-rasterizer", mainJsPath)
		app.desktopCmd.Env = append(os.Environ(),
			fmt.Sprintf("KNIRV_SERVER_URL=%s", serverUrl),
			fmt.Sprintf("KNIRV_SHUTDOWN_TOKEN=%s", app.shutdownToken),
			fmt.Sprintf("KNIRV_SERVER_PORT=%d", app.config.Port),
			"VK_ICD_FILENAMES=",
			"ELECTRON_OZONE_PLATFORM_HINT=auto",
		)
	}
	app.desktopCmd.Dir = desktopDir
	app.desktopCmd.Stdout = os.Stdout
	app.desktopCmd.Stderr = os.Stderr

	if err := app.desktopCmd.Start(); err != nil {
		return fmt.Errorf("failed to start desktop: %w", err)
	}

	log.Printf("KNIRV Desktop started (PID: %d)", app.desktopCmd.Process.Pid)
	return nil
}

// stopDesktop stops the Electron desktop application
func (app *NexusApp) stopDesktop() {
	if app.desktopCmd != nil && app.desktopCmd.Process != nil {
		log.Printf("Stopping KNIRV Desktop (PID: %d)", app.desktopCmd.Process.Pid)
		app.desktopCmd.Process.Signal(syscall.SIGTERM)
		app.desktopCmd.Wait()
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

	// Extract to app data directory so the application can find it.
	appDataDir, err := getAppDataDir()
	if err != nil {
		return fmt.Errorf("failed to get app data directory: %w", err)
	}
	envPath := filepath.Join(appDataDir, ".env")
	if err := os.WriteFile(envPath, envData, 0644); err != nil {
		return fmt.Errorf("failed to write %s to %s: %w", envName, envPath, err)
	}

	// Propagate the path so subprocesses can locate it.
	os.Setenv("KNIRV_ENV_FILE", envPath)

	log.Printf("Extracted %s environment file to %s", environment, envPath)
	return nil
}

// extractRootKey copies the embedded root.key to the path the backend expects:
//
//	os.UserConfigDir()/knirv-server/root.key
//
// The file is only written when rootKeyBytes is non-nil (i.e. the key was
// compiled in) and the destination does NOT already exist (we never silently
// overwrite an operator-managed key).
func extractRootKey() error {
	if len(rootKeyBytes) == 0 {
		// No key compiled in — nothing to do (non-root-node build).
		return nil
	}

	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return fmt.Errorf("failed to locate user config dir: %w", err)
	}

	destDir := filepath.Join(userConfigDir, "knirv-server")
	if err := os.MkdirAll(destDir, 0700); err != nil {
		return fmt.Errorf("failed to create config dir %s: %w", destDir, err)
	}

	destPath := filepath.Join(destDir, "root.key")

	// Never overwrite an existing key — the operator owns it once deployed.
	if _, err := os.Stat(destPath); err == nil {
		log.Printf("root.key already present at %s — skipping extraction", destPath)
		return nil
	}

	if err := os.WriteFile(destPath, rootKeyBytes, 0600); err != nil {
		return fmt.Errorf("failed to write root.key to %s: %w", destPath, err)
	}

	log.Printf("Extracted root.key to %s", destPath)
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
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Backend service unavailable"})
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

// killStaleBackend sends SIGTERM then SIGKILL to any running backend_server processes
// so port 4001 (P2P/libp2p) and other ports are free before a new instance starts.
func killStaleBackend(binaryPath string) {
	binaryName := filepath.Base(binaryPath)
	if binaryName == "" || binaryName == "." {
		binaryName = "backend_server"
	}
	if cmd := exec.Command("pkill", "-TERM", "-x", binaryName); cmd.Run() == nil {
		time.Sleep(2 * time.Second)
	}
	exec.Command("pkill", "-KILL", "-x", binaryName).Run() //nolint:errcheck
}

// waitForPortFreeMain blocks until the given TCP port is no longer occupied or the deadline.
func waitForPortFreeMain(port int, deadline time.Duration) bool {
	end := time.Now().Add(deadline)
	for time.Now().Before(end) {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			ln.Close()
			return true
		}
		time.Sleep(300 * time.Millisecond)
	}
	return false
}

// startBackend starts the embedded unified backend service
func (app *NexusApp) startBackend() error {
	log.Printf("Starting unified backend service on port %d...", app.config.BackendPort)

	// Kill any stale backend_server processes from previous runs so they do not
	// hold the P2P port (4001) or other resources that would cause the new
	// instance to crash on startup.
	log.Println("Killing any stale backend_server processes...")
	killStaleBackend(app.backendPath)
	if !waitForPortFreeMain(4001, 8*time.Second) {
		log.Println("Warning: P2P port 4001 still occupied after kill — proceeding anyway")
	}

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

	// Propagate paths to bundled binaries so the backend does not rely on CWD
	// or system PATH to locate knirvgateway and knirvcli.  Also propagate the
	// app data directory so the gateway runtime extracts files there instead of
	// /tmp, and so the backend can locate config/log directories consistently.
	if appDataDir, err := getAppDataDir(); err == nil {
		binDir := filepath.Join(appDataDir, "bin")
		env = append(env,
			fmt.Sprintf("KNIRV_APP_DATA_DIR=%s", appDataDir),
			fmt.Sprintf("KNIRV_GATEWAY_BINARY_PATH=%s", filepath.Join(binDir, "knirvgateway")),
			fmt.Sprintf("KNIRV_KNIRVCLI_PATH=%s", filepath.Join(binDir, "knirvcli")),
		)
	}

	// Pass the project log directory as an absolute path so the backend writes
	// logs to <cwd>/logs/server.log regardless of where the backend binary runs from.
	if cwd, err := os.Getwd(); err == nil {
		env = append(env, fmt.Sprintf("KNIRV_PROJECT_LOG_DIR=%s", filepath.Join(cwd, "logs")))
	}

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

	// Wait for backend to accept connections on its health endpoint.
	healthURL := fmt.Sprintf("http://localhost:%d/health", app.config.BackendPort)
	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := client.Get(healthURL)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode < 500 {
				log.Printf("Backend ready on port %d", app.config.BackendPort)
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	log.Printf("Warning: backend did not become healthy within 30s — proceeding anyway")

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

// Start starts the KNIRV-SERVER application
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
		log.Printf("Starting KNIRV-SERVER on %s:%d", app.config.Host, app.config.Port)
		if err := app.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for server to be ready
	time.Sleep(2 * time.Second)

	// Start desktop if enabled
	if app.config.Desktop {
		if err := app.startDesktop(); err != nil {
			log.Printf("Warning: Failed to start desktop: %v", err)
		}
	}

	return nil
}

// Stop stops the KNIRV-SERVER application
func (app *NexusApp) Stop() error {
	// Stop desktop first
	app.stopDesktop()

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
		desktop     = flag.Bool("desktop", false, "Enable the KNIRV Desktop GUI")
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
	viper.SetDefault("desktop", false)

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
	if *desktop {
		config.Desktop = true
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
	fmt.Printf("KNIRV-SERVER v%s (built %s, commit %s)\n", Version, BuildTime, GitCommit)

	// Extract embedded config files to app data directory
	if err := extractConfigFiles(); err != nil {
		log.Printf("Warning: Failed to extract config files: %v", err)
	}

	// Extract root.key to the backend config directory (no-op if absent or already present)
	if err := extractRootKey(); err != nil {
		log.Printf("Warning: Failed to extract root.key: %v", err)
	}

	// Load configuration
	config, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Log testnet mode if enabled
	if config.Testnet {
		log.Println("🧪 Starting KNIRV-SERVER in testnet mode")
	}

	// Log desktop mode if enabled
	if config.Desktop {
		log.Println("🖥️  Starting KNIRV-SERVER with Desktop GUI")
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

	// Wait for shutdown signal (OS signal or desktop /shutdown request)
	select {
	case <-sigChan:
	case <-app.shutdownChan:
		log.Println("Shutdown requested by desktop")
	}
	log.Println("Shutting down...")

	if err := app.Stop(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}

	log.Println("KNIRV-SERVER stopped")
}
