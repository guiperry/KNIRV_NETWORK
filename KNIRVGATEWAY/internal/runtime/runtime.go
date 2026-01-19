package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/embedded"
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
}

// NewRuntime creates a new runtime manager
func NewRuntime(logger *zap.Logger) (*Runtime, error) {
	// Create runtime directory in system temp
	baseDir := filepath.Join(os.TempDir(), "knirvoracle-runtime")

	r := &Runtime{
		BaseDir:           baseDir,
		NetworkWebsiteDir: filepath.Join(baseDir, "network-website"),
		WebGUIStaticDir:   filepath.Join(baseDir, "webgui-static"),
		OracleBinaryPath:  filepath.Join(baseDir, "knirv-oracle"),
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

	// Extract webgui static files
	r.logger.Info("Extracting embedded webgui static files...")
	if err := embedded.ExtractWebGUI(embedded.WebGUIFS, r.WebGUIStaticDir, r.logger); err != nil {
		return fmt.Errorf("failed to extract webgui static files: %w", err)
	}

	// Extract network-website
	r.logger.Info("Extracting embedded network-website...")
	if err := embedded.ExtractNetworkWebsite(embedded.NetworkWebsiteFS, r.NetworkWebsiteDir, r.logger); err != nil {
		return fmt.Errorf("failed to extract network-website: %w", err)
	}

	// Extract knirv-oracle binary
	r.logger.Info("Extracting embedded knirv-oracle binary...")
	if err := embedded.ExtractOracleBinary(r.OracleBinaryPath, r.logger); err != nil {
		return fmt.Errorf("failed to extract knirv-oracle binary: %w", err)
	}

	r.extracted = true
	r.logger.Info("Runtime setup complete",
		zap.String("webguiStaticDir", r.WebGUIStaticDir),
		zap.String("networkWebsiteDir", r.NetworkWebsiteDir),
		zap.String("oracleBinaryPath", r.OracleBinaryPath),
	)

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
