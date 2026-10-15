// Package knirvulora embeds, extracts and runs the ulorad adapter-compiler daemon
// (source: packages/KNIRVULORA/cmd/ulorad), the same way the other
// KNIRVSERVER/pkg/knirv* modules do: `make ulora-build` writes a gzip of the binary
// to bin/ulorad, which is go:embed'ed here.
package knirvulora

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

//go:embed bin/ulorad
var embeddedBinary []byte

// BinaryName is the extracted executable's file name.
const BinaryName = "ulorad"

func defaultBinaryDir() (string, error) {
	if override := strings.TrimSpace(os.Getenv("KNIRV_ULORA_BINARY_DIR")); override != "" {
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

func writeFileAtomically(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file for %s: %w", path, err)
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}()
	if _, err := tmp.Write(data); err != nil {
		return fmt.Errorf("failed to write temp file for %s: %w", path, err)
	}
	if err := tmp.Chmod(perm); err != nil {
		return fmt.Errorf("failed to chmod temp file for %s: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to close temp file for %s: %w", path, err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to replace %s atomically: %w", path, err)
	}
	return nil
}

// decompress returns data gunzipped, or data itself when it is not gzip (a
// development build that skipped compression).
func decompress(data []byte) ([]byte, error) {
	if len(data) < 2 || data[0] != 0x1f || data[1] != 0x8b {
		return data, nil
	}
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer r.Close()
	out, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress embedded ulorad: %w", err)
	}
	return out, nil
}

// ExtractEmbeddedBinary writes the embedded ulorad into destDir (the app-data bin
// directory when empty) and returns its path.
func ExtractEmbeddedBinary(destDir string) (string, error) {
	if len(embeddedBinary) == 0 {
		return "", fmt.Errorf("embedded ulorad binary is empty")
	}
	data, err := decompress(embeddedBinary)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(destDir) == "" {
		if destDir, err = defaultBinaryDir(); err != nil {
			return "", err
		}
	}
	dest := filepath.Join(destDir, BinaryName)
	if err := os.RemoveAll(dest); err != nil {
		return "", err
	}
	if err := writeFileAtomically(dest, data, 0755); err != nil {
		return "", err
	}
	return dest, nil
}
