package embedded

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
	
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVORACLE/internal/config"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVORACLE/internal/oracle"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVORACLE/internal/runtime"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVORACLE/internal/server"
	"go.uber.org/zap"
)

// Embed all service directories (excluding node_modules, build artifacts)
//
//go:embed all:webgui
var WebGUIFS embed.FS

//go:embed all:network-website/*
var NetworkWebsiteFS embed.FS

//go:embed knirv-oracle
var OracleBinary []byte

// OracleConfig represents the configuration for the embedded oracle
type OracleConfig struct {
	Port              int    `json:"port"`
	ChainID           string `json:"chainID"`
	Mode              string `json:"mode"`
	OracleOwnerKey    string `json:"oracleOwnerKey"`
	AutoOpenBrowser   bool   `json:"autoOpenBrowser"`
	EnableOracle      bool   `json:"enableOracle"`
	EnableModelServer bool   `json:"enableModelServer"`
}

// DefaultConfig returns a default configuration for the embedded oracle
func DefaultConfig() *OracleConfig {
	return &OracleConfig{
		Port:              8080,
		ChainID:           "testnet",
		Mode:              "oracle_nest",
		OracleOwnerKey:    "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		AutoOpenBrowser:   false,
		EnableOracle:      true,
		EnableModelServer: true,
	}
}

// Oracle represents an embedded KNIRVORACLE instance
type Oracle struct {
	cfg     *OracleConfig
	logger  *zap.Logger
	runtime *runtime.Runtime
	server  *server.Server
	oracle  interface{} // Can be *oracle.Oracle or *os.Process
	stopCh  chan struct{}
}

// NewOracle creates a new embedded oracle instance
func NewOracle(cfg *OracleConfig, logger *zap.Logger) (*Oracle, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	if logger == nil {
		var err error
		logger, err = zap.NewProduction()
		if err != nil {
			return nil, err
		}
	}

	rt, err := runtime.NewRuntime(logger)
	if err != nil {
		return nil, err
	}

	return &Oracle{
		cfg:     cfg,
		logger:  logger,
		runtime: rt,
		stopCh:  make(chan struct{}),
	}, nil
}

// Start initializes and starts the embedded oracle
func (g *Oracle) Start() error {
	g.logger.Info("Starting embedded KNIRVORACLE",
		zap.String("mode", g.cfg.Mode),
		zap.Int("port", g.cfg.Port),
		zap.String("chainID", g.cfg.ChainID),
	)

	// Setup runtime
	if err := g.runtime.Setup(); err != nil {
		return err
	}

	// Set environment variables
	os.Setenv("KNIRV_MODE", g.cfg.Mode)
	os.Setenv("PORT", string(rune(g.cfg.Port)))
	os.Setenv("CHAIN_ID", g.cfg.ChainID)
	os.Setenv("ORACLE_OWNER_KEY", g.cfg.OracleOwnerKey)
	os.Setenv("ENABLE_ORACLE", "true")
	os.Setenv("ENABLE_MODEL_SERVER", "true")

	// Initialize oracle configuration
	oracleCfg, err := oracle.LoadConfigFromEnv()
	if err != nil {
		return err
	}

	// Start oracle
	if g.cfg.EnableOracle {
		if err := g.startOracle(oracleCfg); err != nil {
			return err
		}
	}

	// Initialize HTTP server
	coreCfg, err := config.Load()
	if err != nil {
		return err
	}

	g.server, err = server.New(coreCfg, g.runtime.GetWebGUIStaticPath(), g.runtime.GetNetworkWebsitePath(), g.logger)
	if err != nil {
		return err
	}

	// Start server
	go func() {
		g.logger.Info("Starting HTTP server", zap.Int("port", g.cfg.Port))
		if err := g.server.Start(); err != nil {
			g.logger.Fatal("Server failed", zap.Error(err))
		}
	}()

	// Wait for server to start
	time.Sleep(2 * time.Second)

	g.logger.Info("Embedded KNIRVORACLE started successfully")

	// Start signal handler
	go g.handleSignals()

	return nil
}

func (g *Oracle) startOracle(cfg *oracle.OracleConfig) error {
	// Check if oracle binary exists and owner key is set
	oracleBinaryPath := g.runtime.GetOracleBinaryPath()
	if _, err := os.Stat(oracleBinaryPath); err == nil && cfg.OwnerPrivateKey != "" {
		// Binary exists and owner key is set, start as separate process
		g.logger.Info("Starting knirv-oracle as separate process...", zap.String("path", oracleBinaryPath))
		// TODO: Implement process management
	} else {
		// Binary doesn't exist or owner key not set, initialize directly
		g.logger.Warn("knirv-oracle binary not found or owner key not set, initializing directly...")

		// Validate oracle configuration
		if err := oracle.ValidateConfig(cfg); err != nil {
			return err
		}

		g.logger.Info("Oracle configuration summary", zap.String("config", oracle.ConfigSummary(cfg)))

		// Create oracle instance
		oracleNode, err := oracle.NewOracle(cfg, g.logger)
		if err != nil {
			return err
		}

		g.oracle = oracleNode

		// Start oracle services
		go func() {
			g.logger.Info("Starting knirv-oracle services...")
			if err := oracleNode.Start(); err != nil {
				g.logger.Fatal("Oracle node failed to start", zap.Error(err))
			}
		}()
	}

	return nil
}

func (g *Oracle) handleSignals() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigChan:
		g.logger.Info("Received shutdown signal")
		g.Stop()
	case <-g.stopCh:
		g.logger.Info("Received stop signal")
	}
}

// Stop gracefully stops the embedded oracle
func (g *Oracle) Stop() error {
	g.logger.Info("Stopping embedded KNIRVORACLE")

	// Close stop channel
	close(g.stopCh)

	// Stop server
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if g.server != nil {
		if err := g.server.Stop(ctx); err != nil {
			g.logger.Error("Error stopping server", zap.Error(err))
		}
	}

	// Stop oracle
	if g.oracle != nil {
		switch oracleInstance := g.oracle.(type) {
		case *oracle.Oracle:
			if err := oracleInstance.Stop(); err != nil {
				g.logger.Error("Error stopping oracle node", zap.Error(err))
			}
		case *os.Process:
			// TODO: Implement process termination
		}
	}

	// Cleanup runtime
	if err := g.runtime.Cleanup(); err != nil {
		g.logger.Error("Failed to cleanup runtime", zap.Error(err))
	}

	g.logger.Info("Embedded KNIRVORACLE stopped")
	return nil
}

// GetRuntime returns the runtime instance
func (g *Oracle) GetRuntime() *runtime.Runtime {
	return g.runtime
}

// GetServer returns the server instance
func (g *Oracle) GetServer() *server.Server {
	return g.server
}

// GetPort returns the configured port
func (g *Oracle) GetPort() int {
	return g.cfg.Port
}

// GetChainID returns the configured chain ID
func (g *Oracle) GetChainID() string {
	return g.cfg.ChainID
}

// IsRunning returns true if the oracle is running
func (g *Oracle) IsRunning() bool {
	select {
	case <-g.stopCh:
		return false
	default:
		return true
	}
}

// ExtractOracleBinary extracts the embedded knirv-oracle binary to the target directory
func ExtractOracleBinary(targetPath string, logger *zap.Logger) error {
	logger.Info("Extracting embedded knirv-oracle binary", zap.String("target", targetPath))

	// Ensure target directory exists
	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Write the embedded binary to file
	if err := os.WriteFile(targetPath, OracleBinary, 0755); err != nil {
		return fmt.Errorf("failed to write oracle binary: %w", err)
	}

	logger.Info("Successfully extracted knirv-oracle binary", zap.String("path", targetPath))
	return nil
}

// ExtractWebGUI extracts only the webgui static files to a target directory
func ExtractWebGUI(webGUIFS embed.FS, targetDir string, logger *zap.Logger) error {
	logger.Info("Extracting embedded webgui static files", zap.String("target", targetDir))

	// Create target directory
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Extract only webgui files
	err := fs.WalkDir(webGUIFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Get relative path
		relPath, err := filepath.Rel(".", path)
		if err != nil {
			return err
		}

		// Skip the root "." directory itself, but allow walking into it
		if relPath == "." {
			return nil
		}

		// Skip everything except webgui files
		if !strings.HasPrefix(relPath, "webgui") {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		// For webgui, only extract the built static files from out/ directory
		// Allow descending into "webgui" and "webgui/out" directories, skip everything else
		if d.IsDir() {
			// Allow webgui root directory
			if relPath == "webgui" {
				return nil
			}
			// Allow webgui/out directory and its subdirectories
			if strings.HasPrefix(relPath, "webgui/out") {
				// Create the directory in the target location
				staticPath := strings.TrimPrefix(relPath, "webgui/out/")
				staticPath = strings.TrimPrefix(staticPath, "/")
				targetPath := filepath.Join(targetDir, staticPath)

				if err := os.MkdirAll(targetPath, 0755); err != nil {
					return fmt.Errorf("failed to create directory %s: %w", targetPath, err)
				}
				return nil
			}
			// Skip all other webgui subdirectories (src, node_modules, etc.)
			return fs.SkipDir
		}

		// Skip files that are not in webgui/out
		if !strings.HasPrefix(relPath, "webgui/out/") {
			return nil
		}

		// Determine target path - remove the "webgui/out" prefix
		staticPath := strings.TrimPrefix(relPath, "webgui/out/")
		// If still has leading slash after trim (shouldn't happen but be safe), remove it
		staticPath = strings.TrimPrefix(staticPath, "/")
		targetPath := filepath.Join(targetDir, staticPath)

		if d.IsDir() {
			// Create directory
			return os.MkdirAll(targetPath, 0755)
		}

		// Read embedded file
		data, err := fs.ReadFile(WebGUIFS, path)
		if err != nil {
			return fmt.Errorf("failed to read embedded file %s: %w", path, err)
		}

		// Write to target
		if err := os.WriteFile(targetPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", targetPath, err)
		}

		return nil
	})

	if err != nil {
		return err
	}

	// Check if any files were extracted, if not create a basic dashboard
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		logger.Info("No webgui static files found, creating basic dashboard")
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return fmt.Errorf("failed to create webgui directory: %w", err)
		}

		// Create a basic dashboard HTML file
		dashboardHTML := `<!DOCTYPE html>
<html lang="en">
<head>
	  <meta charset="UTF-8">
	  <meta name="viewport" content="width=device-width, initial-scale=1.0">
	  <title>KNIRV WebGUI Dashboard</title>
	  <style>
	      body {
	          font-family: Arial, sans-serif;
	          margin: 0;
	          padding: 20px;
	          background-color: #f5f5f5;
	      }
	      .header {
	          background-color: #2b56f5;
	          color: white;
	          padding: 20px;
	          border-radius: 8px;
	          margin-bottom: 20px;
	      }
	      .dashboard {
	          display: grid;
	          grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
	          gap: 20px;
	      }
	      .card {
	          background: white;
	          padding: 20px;
	          border-radius: 8px;
	          box-shadow: 0 2px 4px rgba(0,0,0,0.1);
	      }
	      .logout-btn {
	          background: #dc3545;
	          color: white;
	          border: none;
	          padding: 10px 20px;
	          border-radius: 5px;
	          cursor: pointer;
	          float: right;
	      }
	  </style>
</head>
<body>
	  <div class="header">
	      <h1>KNIRV WebGUI Dashboard</h1>
	      <p>Welcome to the KNIRV Network Dashboard</p>
	      <button class="logout-btn" onclick="logout()">Logout</button>
	  </div>

	  <div class="dashboard">
	      <div class="card">
	          <h3>Network Status</h3>
	          <p>Status: Connected</p>
	          <p>Chain ID: testnet</p>
	      </div>

	      <div class="card">
	          <h3>Wallet</h3>
	          <p>Balance: 0 NRN</p>
	          <p>Address: 0x...</p>
	      </div>

	      <div class="card">
	          <h3>Recent Transactions</h3>
	          <p>No recent transactions</p>
	      </div>

	      <div class="card">
	          <h3>AI Agents</h3>
	          <p>Active Agents: 0</p>
	          <p>Skills Available: 0</p>
	      </div>
	  </div>

	  <script>
	      function logout() {
	          // Clear auth token
	          localStorage.removeItem('knirv_auth_token');
	          // Redirect to main site
	          window.location.href = '/';
	      }

	      // Check if user is logged in
	      const token = localStorage.getItem('knirv_auth_token');
	      if (!token) {
	          window.location.href = '/';
	      }
	  </script>
</body>
</html>`
		indexPath := filepath.Join(targetDir, "index.html")
		if err := os.WriteFile(indexPath, []byte(dashboardHTML), 0644); err != nil {
			return fmt.Errorf("failed to create dashboard HTML: %w", err)
		}
		logger.Info("Created basic dashboard HTML", zap.String("path", indexPath))
	}

	return nil
}

// ExtractNetworkWebsite extracts only the network-website files to a target directory
func ExtractNetworkWebsite(networkWebsiteFS embed.FS, targetDir string, logger *zap.Logger) error {
	logger.Info("Extracting embedded network-website", zap.String("target", targetDir))

	// Create target directory
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Extract only network-website files
	err := fs.WalkDir(networkWebsiteFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Get relative path
		relPath, err := filepath.Rel("network-website", path)
		if err != nil {
			return err
		}

		// Skip the root
		if relPath == "." {
			return nil
		}

		// Skip node_modules directories
		if strings.Contains(relPath, "node_modules") {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		// Determine target path
		websitePath := relPath
		targetPath := filepath.Join(targetDir, websitePath)

		if d.IsDir() {
			// Create directory
			return os.MkdirAll(targetPath, 0755)
		}

		// Read embedded file
		data, err := fs.ReadFile(NetworkWebsiteFS, path)
		if err != nil {
			return fmt.Errorf("failed to read embedded file %s: %w", path, err)
		}

		// Write to target
		if err := os.WriteFile(targetPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", targetPath, err)
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

// CleanupExtracted removes extracted files
func CleanupExtracted(targetDir string, logger *zap.Logger) error {
	logger.Info("Cleaning up extracted files", zap.String("dir", targetDir))
	return os.RemoveAll(targetDir)
}
