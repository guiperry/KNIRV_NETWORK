package bundling

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const (
	ManifestFile = "manifest.json"
	WeightsFile  = "core_weights.safetensors"
	AnchorsFile  = "calibration_anchors.parquet"
)

func ContentHashOfFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open bundle file: %w", err)
	}
	defer f.Close()
	return HashReader(f)
}

func HashReader(r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", fmt.Errorf("compute hash: %w", err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func ReadFile(bundleOrDirPath string, name string) ([]byte, error) {
	if info, err := os.Stat(bundleOrDirPath); err == nil && !info.IsDir() {
		return readFileFromArchive(bundleOrDirPath, name)
	}
	return os.ReadFile(filepath.Join(bundleOrDirPath, name))
}

func WriteBundle(dirPath, bundlePath string) error {
	f, err := os.Create(bundlePath)
	if err != nil {
		return fmt.Errorf("create bundle: %w", err)
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	entries := []string{ManifestFile, WeightsFile, AnchorsFile}
	for _, name := range entries {
		fullPath := filepath.Join(dirPath, name)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			continue
		}
		if err := addToTar(tw, dirPath, name); err != nil {
			return fmt.Errorf("add %s to bundle: %w", name, err)
		}
	}
	return nil
}

func ExtractBundle(bundlePath, destDir string) error {
	f, err := os.Open(bundlePath)
	if err != nil {
		return fmt.Errorf("open bundle: %w", err)
	}
	defer f.Close()

	var tr *tar.Reader
	gz, err := gzip.NewReader(f)
	if err == nil {
		tr = tar.NewReader(gz)
	} else {
		_, err = f.Seek(0, io.SeekStart)
		if err != nil {
			return fmt.Errorf("rewind bundle: %w", err)
		}
		tr = tar.NewReader(f)
	}

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read tar entry: %w", err)
		}

		target := filepath.Join(destDir, header.Name)
		if header.Typeflag != tar.TypeReg {
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("mkdir: %w", err)
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
		if err != nil {
			return fmt.Errorf("create file: %w", err)
		}
		if _, err := io.Copy(out, tr); err != nil {
			out.Close()
			return fmt.Errorf("write file: %w", err)
		}
		out.Close()
	}
	return nil
}

func addToTar(tw *tar.Writer, rootDir, name string) error {
	fullPath := filepath.Join(rootDir, name)
	info, err := os.Stat(fullPath)
	if err != nil {
		return err
	}
	hdr, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	hdr.Name = name
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	f, err := os.Open(fullPath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(tw, f)
	return err
}

func readFileFromArchive(archivePath, name string) ([]byte, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return nil, fmt.Errorf("open bundle: %w", err)
	}
	defer f.Close()

	var r io.Reader = f
	gz, err := gzip.NewReader(f)
	if err == nil {
		r = gz
	} else {
		_, err = f.Seek(0, io.SeekStart)
		if err != nil {
			return nil, fmt.Errorf("rewind bundle: %w", err)
		}
		r = f
	}

	tr := tar.NewReader(r)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read tar entry: %w", err)
		}
		if header.Name == name {
			return io.ReadAll(tr)
		}
	}
	return nil, fmt.Errorf("file %s not found in bundle", name)
}

func EnsureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}
