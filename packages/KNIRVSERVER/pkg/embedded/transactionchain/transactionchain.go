package transactionchain

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

//go:embed bundle/*
var embeddedFiles embed.FS

const (
	extractionSubdir = "transaction_chain"
	hashFilename     = ".version_hash"
)

// calculateContentHash computes SHA256 hash of all embedded files
func calculateContentHash() (string, error) {
	h := sha256.New()

	err := fs.WalkDir(embeddedFiles, "bundle", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		f, err := embeddedFiles.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		if _, err := io.Copy(h, f); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// extractFiles extracts embedded files to destination directory, stripping the bundle/ prefix
func extractFiles(destDir string) error {
	return fs.WalkDir(embeddedFiles, "bundle", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Strip the leading "bundle/" prefix so files land at destDir directly
		rel := strings.TrimPrefix(path, "bundle/")
		if rel == "bundle" || rel == "" {
			return nil
		}

		destPath := filepath.Join(destDir, rel)
		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		srcFile, err := embeddedFiles.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		destFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		defer destFile.Close()

		_, err = io.Copy(destFile, srcFile)
		return err
	})
}

// ExtractEmbeddedApp extracts the embedded transaction chain bundle to a stable
// system cache directory and returns its path. backend_server's own
// transactionchain.Manager (KNIRV_CORP) spawns the actual Node.js process from
// this directory; KNIRVSERVER's role is only to stage the bundle on disk.
func ExtractEmbeddedApp() (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		cacheDir = os.TempDir()
	}

	destDir := filepath.Join(cacheDir, "knirvserver", extractionSubdir)
	currentHash, err := calculateContentHash()
	if err != nil {
		return "", fmt.Errorf("failed to calculate content hash: %w", err)
	}

	hashFile := filepath.Join(destDir, hashFilename)
	if existingHash, err := os.ReadFile(hashFile); err == nil {
		if string(existingHash) == currentHash {
			return destDir, nil
		}
	}

	if err := os.RemoveAll(destDir); err != nil {
		return "", fmt.Errorf("failed to clean old extraction: %w", err)
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create destination directory: %w", err)
	}

	if err := extractFiles(destDir); err != nil {
		return "", fmt.Errorf("failed to extract files: %w", err)
	}

	if err := os.WriteFile(hashFile, []byte(currentHash), 0644); err != nil {
		return "", fmt.Errorf("failed to write version hash: %w", err)
	}

	return destDir, nil
}
