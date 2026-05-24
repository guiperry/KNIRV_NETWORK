package knirvarena

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	_ "embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

//go:embed bin/knirvarena.tar.gz
var embeddedBundle []byte

func defaultExtractDir() (string, error) {
	if override := strings.TrimSpace(os.Getenv("KNIRV_ARENA_EXTRACT_DIR")); override != "" {
		return override, nil
	}
	if appDataDir := strings.TrimSpace(os.Getenv("KNIRV_APP_DATA_DIR")); appDataDir != "" {
		return filepath.Join(appDataDir, "knirvarena"), nil
	}
	if err := os.MkdirAll("/var/lib/knirvserver/knirvarena", 0755); err == nil {
		return "/var/lib/knirvserver/knirvarena", nil
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to resolve home directory: %w", err)
	}
	return filepath.Join(homeDir, ".local", "share", "knirvserver", "knirvarena"), nil
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
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() { _ = tmpFile.Close(); _ = os.Remove(tmpPath) }()
	if _, err := tmpFile.Write(data); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	if err := tmpFile.Chmod(perm); err != nil {
		return fmt.Errorf("failed to chmod temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to replace atomically: %w", err)
	}
	return nil
}

func ExtractEmbeddedApp(destDir string) (string, error) {
	if len(embeddedBundle) == 0 {
		return "", fmt.Errorf("embedded knirvarena bundle is empty")
	}

	if len(embeddedBundle) < 2 || embeddedBundle[0] != 0x1f || embeddedBundle[1] != 0x8b {
		return "", fmt.Errorf("embedded bundle is not gzip compressed")
	}

	gzr, err := gzip.NewReader(bytes.NewReader(embeddedBundle))
	if err != nil {
		return "", fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzr.Close()

	if strings.TrimSpace(destDir) == "" {
		var err error
		destDir, err = defaultExtractDir()
		if err != nil {
			return "", err
		}
	}

	if err := os.RemoveAll(destDir); err != nil {
		return "", fmt.Errorf("failed to clean old extraction: %w", err)
	}
	if err := ensureDir(destDir); err != nil {
		return "", err
	}

	tr := tar.NewReader(gzr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("failed to read tar entry: %w", err)
		}

		target := filepath.Join(destDir, hdr.Name)
		cleanTarget := filepath.Clean(target)
		cleanDest := filepath.Clean(destDir)

		// Skip the archive root entry (./ or .) — it maps to destDir itself.
		if cleanTarget == cleanDest {
			continue
		}

		if !strings.HasPrefix(cleanTarget, cleanDest+string(os.PathSeparator)) {
			return "", fmt.Errorf("illegal tar path: %s", hdr.Name)
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := ensureDir(target); err != nil {
				return "", err
			}
		case tar.TypeReg:
			if err := ensureDir(filepath.Dir(target)); err != nil {
				return "", err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode))
			if err != nil {
				return "", fmt.Errorf("failed to create %s: %w", target, err)
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return "", fmt.Errorf("failed to write %s: %w", target, err)
			}
			f.Close()
		}
	}

	return destDir, nil
}
