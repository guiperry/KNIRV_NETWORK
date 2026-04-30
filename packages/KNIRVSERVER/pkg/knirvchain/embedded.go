package knirvchain

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:embed bin/knirvchain
var embeddedBinary []byte

func defaultBinaryDir() (string, error) {
	if override := strings.TrimSpace(os.Getenv("KNIRV_CHAIN_BINARY_DIR")); override != "" {
		return override, nil
	}
	if appDataDir := strings.TrimSpace(os.Getenv("KNIRV_APP_DATA_DIR")); appDataDir != "" {
		return filepath.Join(appDataDir, "bin"), nil
	}
	if err := os.MkdirAll("/var/lib/knirvserver/bin", 0755); err == nil {
		return "/var/lib/knirvserver/bin", nil
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to resolve home directory: %w", err)
	}
	return filepath.Join(homeDir, ".local", "share", "knirvserver", "bin"), nil
}

func ensureDir(path string) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", path, err)
	}
	return nil
}

func writeFileAtomically(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := ensureDir(dir); err != nil {
		return err
	}
	tmpFile, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file for %s: %w", path, err)
	}
	tmpPath := tmpFile.Name()
	defer func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
	}()
	if _, err := tmpFile.Write(data); err != nil {
		return fmt.Errorf("failed to write temp file for %s: %w", path, err)
	}
	if err := tmpFile.Chmod(perm); err != nil {
		return fmt.Errorf("failed to chmod temp file for %s: %w", path, err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file for %s: %w", path, err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to replace %s atomically: %w", path, err)
	}
	return nil
}

func ExtractEmbeddedBinary(destDir string) (string, error) {
	if len(embeddedBinary) == 0 {
		return "", fmt.Errorf("embedded knirvchain binary is empty")
	}
	if strings.TrimSpace(destDir) == "" {
		var err error
		destDir, err = defaultBinaryDir()
		if err != nil {
			return "", err
		}
	}
	if err := ensureDir(destDir); err != nil {
		return "", err
	}
	destPath := filepath.Join(destDir, "knirvchain")
	if err := os.RemoveAll(destPath); err != nil {
		return "", err
	}
	if err := writeFileAtomically(destPath, embeddedBinary, 0755); err != nil {
		return "", err
	}
	return destPath, nil
}
