package runtime

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"go.uber.org/zap"
)

// Runtime manages the runtime directory for extracted files
type Runtime struct {
	BaseDir           string
	NetworkWebsiteDir string
	WebGUIStaticDir   string
	OracleBinaryPath  string
	logger            *zap.Logger
	mu                sync.Mutex
	extracted         bool

	// Embedded assets
	webGUIFS         embed.FS
	networkWebsiteFS embed.FS
	oracleBinary     []byte
}

// NewRuntime creates a new runtime manager with embedded assets
func NewRuntime(logger *zap.Logger, webGUIFS embed.FS, networkWebsiteFS embed.FS, oracleBinary []byte) (*Runtime, error) {
	// Create runtime directory in system temp
	baseDir := filepath.Join(os.TempDir(), "knirvoracle-runtime")

	r := &Runtime{
		BaseDir:           baseDir,
		NetworkWebsiteDir: filepath.Join(baseDir, "network-website"),
		WebGUIStaticDir:   filepath.Join(baseDir, "webgui-static"),
		OracleBinaryPath:  filepath.Join(baseDir, "knirv-oracle"),
		logger:            logger,
		webGUIFS:          webGUIFS,
		networkWebsiteFS:  networkWebsiteFS,
		oracleBinary:      oracleBinary,
	}

	return r, nil
}

// Setup extracts all embedded files to the runtime directory
func (r *Runtime) Setup() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.extracted {
		r.logger.Info("Runtime already setup")
		return nil
	}

	r.logger.Info("Setting up runtime environment", zap.String("baseDir", r.BaseDir))

	// Clean existing directory if it exists
	if err := os.RemoveAll(r.BaseDir); err != nil {
		r.logger.Warn("Failed to clean existing runtime directory", zap.Error(err))
	}

	// Create base directory
	if err := os.MkdirAll(r.BaseDir, 0755); err != nil {
		return fmt.Errorf("failed to create runtime directory: %w", err)
	}

	// Extract explorer static files.
	r.logger.Info("Extracting embedded explorer static files...")
	if err := r.extractWebGUI(r.WebGUIStaticDir); err != nil {
		return fmt.Errorf("failed to extract explorer static files: %w", err)
	}

	// Use an on-disk network-website tree for the standalone gateway when available.
	if sourceDir := r.resolveNetworkWebsiteSourceDir(); sourceDir != "" {
		r.NetworkWebsiteDir = sourceDir
		r.logger.Info("Using on-disk network-website directory", zap.String("networkWebsiteDir", r.NetworkWebsiteDir))
	} else {
		r.logger.Info("Extracting embedded network-website...")
		if err := r.extractNetworkWebsite(r.NetworkWebsiteDir); err != nil {
			return fmt.Errorf("failed to extract network-website: %w", err)
		}
	}

	// Oracle binary extraction removed (oracle moved to KNIRVSERVER)
	// No longer extracting knirv-oracle binary

	r.extracted = true
	r.logger.Info("Runtime setup complete",
		zap.String("explorerStaticDir", r.WebGUIStaticDir),
		zap.String("networkWebsiteDir", r.NetworkWebsiteDir),
	)

	return nil
}

// extractWebGUI extracts only the explorer static files to a target directory.
func (r *Runtime) extractWebGUI(targetDir string) error {
	r.logger.Info("Extracting embedded explorer static files", zap.String("target", targetDir))

	// Create target directory
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Extract only explorer files from the embedded webgui bundle.
	err := fs.WalkDir(r.webGUIFS, ".", func(path string, d fs.DirEntry, err error) error {
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

		// Skip everything except explorer files.
		if !strings.HasPrefix(relPath, "webgui") {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		// Only extract the built explorer files from out/.
		// Allow descending into "webgui" and "webgui/out" directories, skip everything else.
		if d.IsDir() {
			// Allow the embedded explorer root directory.
			if relPath == "webgui" {
				return nil
			}
			// Allow webgui/out and its subdirectories.
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
			// Skip all other embedded source directories (src, node_modules, etc.).
			return fs.SkipDir
		}

		// Skip files that are not in the built explorer bundle.
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
		data, err := fs.ReadFile(r.webGUIFS, path)
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

	// If nothing was extracted, create a basic explorer landing page.
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		r.logger.Info("No explorer static files found, creating basic explorer page")
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return fmt.Errorf("failed to create explorer directory: %w", err)
		}

		// Create a basic explorer HTML file.
		dashboardHTML := `<!DOCTYPE html>
<html lang="en">
<head>
	  <meta charset="UTF-8">
	  <meta name="viewport" content="width=device-width, initial-scale=1.0">
	  <title>KNIRV Explorer</title>
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
	      <h1>KNIRV Explorer</h1>
	      <p>Welcome to the KNIRV Network Explorer</p>
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
			return fmt.Errorf("failed to create explorer HTML: %w", err)
		}
		r.logger.Info("Created basic explorer HTML", zap.String("path", indexPath))
	}

	return nil
}

// extractNetworkWebsite extracts only the network-website files to a target directory
func (r *Runtime) extractNetworkWebsite(targetDir string) error {
	r.logger.Info("Extracting embedded network-website", zap.String("target", targetDir))

	if _, err := fs.ReadDir(r.networkWebsiteFS, "network-website"); err != nil {
		return fmt.Errorf("embedded network-website assets unavailable")
	}

	// Create target directory
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Extract only network-website files
	err := fs.WalkDir(r.networkWebsiteFS, ".", func(path string, d fs.DirEntry, err error) error {
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
		data, err := fs.ReadFile(r.networkWebsiteFS, path)
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

func (r *Runtime) resolveNetworkWebsiteSourceDir() string {
	candidates := []string{}

	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(cwd, "internal", "embedded", "network-website"),
			filepath.Join(cwd, "..", "internal", "embedded", "network-website"),
			filepath.Join(cwd, "..", "..", "internal", "embedded", "network-website"),
		)
	}

	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		candidates = append(candidates,
			filepath.Join(exeDir, "internal", "embedded", "network-website"),
			filepath.Join(exeDir, "..", "internal", "embedded", "network-website"),
			filepath.Join(exeDir, "..", "..", "internal", "embedded", "network-website"),
		)
	}

	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err == nil && info.IsDir() {
			return candidate
		}
	}

	return ""
}

// Cleanup removes the runtime directory
func (r *Runtime) Cleanup() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.extracted {
		return nil
	}

	r.logger.Info("Cleaning up runtime environment", zap.String("baseDir", r.BaseDir))

	if err := os.RemoveAll(r.BaseDir); err != nil {
		return fmt.Errorf("failed to cleanup runtime directory: %w", err)
	}

	r.extracted = false
	return nil
}

// GetServicePath returns the path to a service directory
// Note: This function is deprecated since services are no longer extracted
func (r *Runtime) GetServicePath(serviceName string) string {
	return ""
}

// GetNetworkWebsitePath returns the path to the network-website
func (r *Runtime) GetNetworkWebsitePath() string {
	return r.NetworkWebsiteDir
}

// GetWebGUIStaticPath returns the path to the webgui static files
func (r *Runtime) GetWebGUIStaticPath() string {
	return r.WebGUIStaticDir
}

// GetOracleBinaryPath returns the path to the extracted knirv-oracle binary
func (r *Runtime) GetOracleBinaryPath() string {
	return r.OracleBinaryPath
}

// IsExtracted returns whether the runtime has been extracted
func (r *Runtime) IsExtracted() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.extracted
}
