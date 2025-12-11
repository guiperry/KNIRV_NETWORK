package runtime

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/embedded"
	"go.uber.org/zap"
)

// Runtime manages the runtime directory for extracted services
type Runtime struct {
	BaseDir           string
	ServicesDir       string
	NetworkWebsiteDir string
	WebGUIStaticDir   string
	logger            *zap.Logger
	mu                sync.Mutex
	extracted         bool
}

// NewRuntime creates a new runtime manager
func NewRuntime(logger *zap.Logger) (*Runtime, error) {
	// Create runtime directory in system temp
	baseDir := filepath.Join(os.TempDir(), "knirvgateway-runtime")

	r := &Runtime{
		BaseDir:           baseDir,
		ServicesDir:       filepath.Join(baseDir, "services"),
		NetworkWebsiteDir: filepath.Join(baseDir, "network-website"),
		WebGUIStaticDir:   filepath.Join(baseDir, "services", "webgui-static"),
		logger:            logger,
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

	// Extract services
	r.logger.Info("Extracting embedded services...")
	if err := embedded.ExtractServices(r.ServicesDir, r.logger); err != nil {
		return fmt.Errorf("failed to extract services: %w", err)
	}

	// Network-website is now extracted as part of ExtractServices

	// Install npm dependencies for services
	if err := r.installServiceDependencies(); err != nil {
		r.logger.Warn("Failed to install some service dependencies", zap.Error(err))
		// Don't fail - services might still work
	}

	r.extracted = true
	r.logger.Info("Runtime setup complete",
		zap.String("servicesDir", r.ServicesDir),
		zap.String("networkWebsiteDir", r.NetworkWebsiteDir),
	)

	return nil
}

// installServiceDependencies runs npm install for each service
func (r *Runtime) installServiceDependencies() error {
	services := []string{
		"payment-gateway",
		"tunnel-registry",
		"webgui",
		"operator-registry",
	}

	for _, svc := range services {
		svcDir := filepath.Join(r.ServicesDir, svc)
		packageJSON := filepath.Join(svcDir, "package.json")

		// Check if package.json exists
		if _, err := os.Stat(packageJSON); os.IsNotExist(err) {
			r.logger.Debug("No package.json found for service, skipping",
				zap.String("service", svc),
			)
			continue
		}

		r.logger.Info("Installing dependencies for service", zap.String("service", svc))

		// Run npm install
		// Note: This is done synchronously during startup
		// In production, you might want to pre-install these in the Docker image
		cmd := fmt.Sprintf("cd %s && npm install --production --no-optional --no-audit 2>&1 | head -20", svcDir)
		if output, err := runCommand(cmd); err != nil {
			r.logger.Warn("Failed to install dependencies",
				zap.String("service", svc),
				zap.Error(err),
				zap.String("output", output),
			)
		}
	}

	return nil
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
func (r *Runtime) GetServicePath(serviceName string) string {
	return filepath.Join(r.ServicesDir, serviceName)
}

// GetNetworkWebsitePath returns the path to the network-website
func (r *Runtime) GetNetworkWebsitePath() string {
	return filepath.Join(r.ServicesDir, "network-website")
}

// GetWebGUIStaticPath returns the path to the webgui static files
func (r *Runtime) GetWebGUIStaticPath() string {
	return r.WebGUIStaticDir
}

// IsExtracted returns whether the runtime has been extracted
func (r *Runtime) IsExtracted() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.extracted
}

// runCommand is a helper to run shell commands
func runCommand(cmd string) (string, error) {
	// Execute command via shell
	output, err := exec.Command("sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return string(output), err
	}
	return string(output), nil
}
