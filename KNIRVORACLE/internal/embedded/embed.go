package embedded

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
)

// Embed all service directories (excluding node_modules, build artifacts)
//
//go:embed all:webgui
//go:embed all:network-website
var ServicesFS embed.FS

// ExtractWebGUI extracts only the webgui static files to a target directory
func ExtractWebGUI(targetDir string, logger *zap.Logger) error {
	logger.Info("Extracting embedded webgui static files", zap.String("target", targetDir))

	// Create target directory
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}


	// Extract only webgui files
	err := fs.WalkDir(ServicesFS, ".", func(path string, d fs.DirEntry, err error) error {
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
		data, err := fs.ReadFile(ServicesFS, path)
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
func ExtractNetworkWebsite(targetDir string, logger *zap.Logger) error {
	logger.Info("Extracting embedded network-website", zap.String("target", targetDir))

	// Create target directory
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Extract only network-website files
	err := fs.WalkDir(ServicesFS, "network-website", func(path string, d fs.DirEntry, err error) error {
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
		data, err := fs.ReadFile(ServicesFS, path)
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
