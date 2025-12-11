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
//go:embed all:webgui all:network-website
var ServicesFS embed.FS

// ExtractServices extracts embedded services to a target directory
func ExtractServices(targetDir string, logger *zap.Logger) error {
	logger.Info("Extracting embedded services", zap.String("target", targetDir))

	// Create target directory
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Extract services (only network-website and webgui)
	return fs.WalkDir(ServicesFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Get relative path
		relPath, err := filepath.Rel(".", path)
		if err != nil {
			return err
		}

		// Skip directories we don't want to extract
		if relPath != "." && relPath != "network-website" && !strings.HasPrefix(relPath, "network-website/") &&
			relPath != "webgui" && !strings.HasPrefix(relPath, "webgui/") {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		// For webgui, only extract the built static files from out/ directory
		if relPath == "webgui" || (strings.HasPrefix(relPath, "webgui/") && !strings.HasPrefix(relPath, "webgui/out/")) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		// Determine target path
		var targetPath string
		if strings.HasPrefix(relPath, "webgui/out/") {
			// Remove the "webgui/out/" prefix and put in "webgui-static/" directory
			staticPath := strings.TrimPrefix(relPath, "webgui/out/")
			targetPath = filepath.Join(targetDir, "webgui-static", staticPath)
		} else {
			targetPath = filepath.Join(targetDir, relPath)
		}

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

		logger.Debug("Extracted file", zap.String("file", relPath))
		return nil
	})
}

// CleanupExtracted removes extracted files
func CleanupExtracted(targetDir string, logger *zap.Logger) error {
	logger.Info("Cleaning up extracted files", zap.String("dir", targetDir))
	return os.RemoveAll(targetDir)
}
