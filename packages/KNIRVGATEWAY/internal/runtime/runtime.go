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
	BaseDir                string
	WebGUIStaticDir        string
	GraphChainExplorerDir  string
	KnirvChainPortalDir    string
	OracleBinaryPath       string
	logger                 *zap.Logger
	mu                     sync.Mutex
	extracted              bool

	// Embedded assets
	webGUIFS              embed.FS
	graphChainExplorerFS  embed.FS
	knirvChainPortalFS    embed.FS
	oracleBinary          []byte
}

// NewRuntime creates a new runtime manager with embedded assets
func NewRuntime(logger *zap.Logger, webGUIFS embed.FS, graphChainExplorerFS embed.FS, knirvChainPortalFS embed.FS, oracleBinary []byte) (*Runtime, error) {
	var baseDir string

	if appDataDir := os.Getenv("KNIRV_APP_DATA_DIR"); appDataDir != "" {
		baseDir = filepath.Join(appDataDir, "knirvgateway", "runtime")
	} else if err := os.MkdirAll("/var/lib/knirvserver/knirvgateway/runtime", 0750); err == nil {
		baseDir = "/var/lib/knirvserver/knirvgateway/runtime"
	} else if homeDir, homeErr := os.UserHomeDir(); homeErr == nil {
		baseDir = filepath.Join(homeDir, ".local", "share", "knirvserver", "knirvgateway", "runtime")
	} else {
		logger.Warn("Could not determine runtime data directory, falling back to system temp", zap.Error(err))
		baseDir = filepath.Join(os.TempDir(), "knirvgateway-runtime")
	}

	// Ensure base directory exists with proper permissions
	if err := os.MkdirAll(baseDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create runtime base directory: %w", err)
	}

	r := &Runtime{
		BaseDir:               baseDir,
		WebGUIStaticDir:       filepath.Join(baseDir, "webgui-static"),
		GraphChainExplorerDir: filepath.Join(baseDir, "graphchain-explorer"),
		KnirvChainPortalDir:   filepath.Join(baseDir, "knirvchain-portal"),
		OracleBinaryPath:      filepath.Join(baseDir, "knirv-oracle"),
		logger:                logger,
		webGUIFS:              webGUIFS,
		graphChainExplorerFS:  graphChainExplorerFS,
		knirvChainPortalFS:    knirvChainPortalFS,
		oracleBinary:          oracleBinary,
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

	// Extract WebGUI static files (Next.js built output from webgui/out/).
	r.logger.Info("Extracting embedded WebGUI static files...")
	if err := r.extractWebGUI(r.WebGUIStaticDir); err != nil {
		return fmt.Errorf("failed to extract WebGUI static files: %w", err)
	}

	// Extract GraphChain Explorer static files (static HTML site).
	r.logger.Info("Extracting embedded GraphChain Explorer...")
	if err := r.extractGraphChainExplorer(r.GraphChainExplorerDir); err != nil {
		return fmt.Errorf("failed to extract GraphChain Explorer: %w", err)
	}

	// Extract KNIRVChain Portal (Vite React site — looks for dist/ first).
	r.logger.Info("Extracting embedded KNIRVChain Portal...")
	if err := r.extractKnirvChainPortal(r.KnirvChainPortalDir); err != nil {
		return fmt.Errorf("failed to extract KNIRVChain Portal: %w", err)
	}

	// Oracle binary extraction removed (oracle moved to KNIRVSERVER).

	r.extracted = true
	r.logger.Info("Runtime setup complete",
		zap.String("webguiStaticDir", r.WebGUIStaticDir),
		zap.String("graphChainExplorerDir", r.GraphChainExplorerDir),
		zap.String("knirvChainPortalDir", r.KnirvChainPortalDir),
	)

	return nil
}

// extractWebGUI extracts only the built WebGUI static files (webgui/out/).
// The embed directive now targets all:webgui/out exclusively, so only the
// Next.js static export is available — no source files, node_modules, or
// intermediate build artifacts are embedded.
func (r *Runtime) extractWebGUI(targetDir string) error {
	r.logger.Info("Extracting embedded WebGUI static files", zap.String("target", targetDir))

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	err := fs.WalkDir(r.webGUIFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Skip root and container directories (webgui, webgui/out).
		if path == "." || path == "webgui" || path == "webgui/out" {
			return nil
		}
		// Everything under webgui/out/ is a build artifact.
		if !strings.HasPrefix(path, "webgui/out/") {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		// Strip webgui/out/ prefix to get the clean path.
		staticPath := strings.TrimPrefix(path, "webgui/out/")
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(targetDir, staticPath), 0755)
		}
		data, err := fs.ReadFile(r.webGUIFS, path)
		if err != nil {
			return fmt.Errorf("failed to read embedded file %s: %w", path, err)
		}
		return os.WriteFile(filepath.Join(targetDir, staticPath), data, 0644)
	})
	if err != nil {
		return err
	}

	// Create a basic fallback HTML if nothing was extracted.
	if isEmpty, _ := isDirEmpty(targetDir); isEmpty {
		r.logger.Info("No WebGUI static files found, creating basic page")
		return r.createFallbackHTML(targetDir, "KNIRV WebGUI", "Welcome to the KNIRV Network WebGUI")
	}
	return nil
}

// extractGraphChainExplorer extracts the static graphchain-explorer site.
func (r *Runtime) extractGraphChainExplorer(targetDir string) error {
	r.logger.Info("Extracting GraphChain Explorer", zap.String("target", targetDir))

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	err := fs.WalkDir(r.graphChainExplorerFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relPath, _ := filepath.Rel(".", path)
		if relPath == "." {
			return nil
		}
		if !strings.HasPrefix(relPath, "graphchain-explorer") {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		staticPath := strings.TrimPrefix(relPath, "graphchain-explorer/")
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(targetDir, staticPath), 0755)
		}
		data, err := fs.ReadFile(r.graphChainExplorerFS, path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}
		return os.WriteFile(filepath.Join(targetDir, staticPath), data, 0644)
	})
	if err != nil {
		return err
	}

	if isEmpty, _ := isDirEmpty(targetDir); isEmpty {
		r.logger.Info("No GraphChain Explorer files found, creating basic page")
		return r.createFallbackHTML(targetDir, "KNIRV GraphChain Explorer", "GraphChain Explorer — static HTML site")
	}
	return nil
}

// extractKnirvChainPortal extracts the built Vite React output (dist/ only).
// The embed directive now targets knirvchain-portal/dist exclusively.
func (r *Runtime) extractKnirvChainPortal(targetDir string) error {
	r.logger.Info("Extracting KNIRVChain Portal (dist)", zap.String("target", targetDir))

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	err := fs.WalkDir(r.knirvChainPortalFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relPath, _ := filepath.Rel(".", path)
		if relPath == "." {
			return nil
		}
		if !strings.HasPrefix(relPath, "knirvchain-portal") {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		// Strip "knirvchain-portal/dist/" prefix to get the clean file path.
		base := strings.TrimPrefix(relPath, "knirvchain-portal/dist/")
		if base == "" || base == relPath {
			if d.IsDir() {
				return nil // descend into knirvchain-portal/dist/
			}
			return nil
		}
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(targetDir, base), 0755)
		}
		data, err := fs.ReadFile(r.knirvChainPortalFS, path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}
		return os.WriteFile(filepath.Join(targetDir, base), data, 0644)
	})
	if err != nil {
		return err
	}

	if isEmpty, _ := isDirEmpty(targetDir); isEmpty {
		r.logger.Info("No KNIRVChain Portal dist files found — run 'npm run build' in knirvchain-portal/ first")
		return r.createFallbackHTML(targetDir, "KNIRVChain Portal", "KNIRVChain Portal — build with: cd knirvchain-portal && npm install && npm run build")
	}
	return nil
}

// createFallbackHTML writes a minimal index.html when no built files are available.
func (r *Runtime) createFallbackHTML(dir, title, subtitle string) error {
	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>%s</title>
<style>body{font-family:Arial,sans-serif;display:flex;align-items:center;justify-content:center;min-height:100vh;margin:0;background:#0a0e17;color:#a0b4cc}.card{text-align:center;padding:40px;background:#111827;border-radius:12px;border:1px solid rgba(99,102,241,0.3)}h1{color:#818cf8}</style>
</head><body><div class="card"><h1>%s</h1><p>%s</p></div></body></html>`, title, title, subtitle)
	return os.WriteFile(filepath.Join(dir, "index.html"), []byte(html), 0644)
}

// isDirEmpty returns true if the directory contains no files or subdirectories.
func isDirEmpty(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}
	return len(entries) == 0, nil
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

// GetWebGUIStaticPath returns the path to the webgui static files
func (r *Runtime) GetWebGUIStaticPath() string {
	return r.WebGUIStaticDir
}

// GetGraphChainExplorerPath returns the path to the graphchain-explorer static files
func (r *Runtime) GetGraphChainExplorerPath() string {
	return r.GraphChainExplorerDir
}

// GetKnirvChainPortalPath returns the path to the knirvchain-portal static files
func (r *Runtime) GetKnirvChainPortalPath() string {
	return r.KnirvChainPortalDir
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
