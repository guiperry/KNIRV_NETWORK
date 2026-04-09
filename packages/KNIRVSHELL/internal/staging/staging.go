package staging

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

type Result struct {
	SourcePath string
	StagedPath string
	Digest     string
	Size       int64
}

func DefaultDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".knirv/staging"
	}
	return filepath.Join(home, ".knirv", "staging")
}

func StageFile(sourcePath, stagingDir string) (*Result, error) {
	if stagingDir == "" {
		stagingDir = DefaultDir()
	}

	source, err := os.Open(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("open source file: %w", err)
	}
	defer source.Close()

	info, err := source.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat source file: %w", err)
	}

	if err := os.MkdirAll(stagingDir, 0755); err != nil {
		return nil, fmt.Errorf("create staging directory: %w", err)
	}

	stagedPath := filepath.Join(stagingDir, fmt.Sprintf("%d_%s", time.Now().UTC().Unix(), filepath.Base(sourcePath)))
	destination, err := os.Create(stagedPath)
	if err != nil {
		return nil, fmt.Errorf("create staged file: %w", err)
	}
	defer destination.Close()

	hash := sha256.New()
	written, err := io.Copy(io.MultiWriter(destination, hash), source)
	if err != nil {
		return nil, fmt.Errorf("copy staged file: %w", err)
	}

	return &Result{
		SourcePath: sourcePath,
		StagedPath: stagedPath,
		Digest:     hex.EncodeToString(hash.Sum(nil)),
		Size:       minInt64(info.Size(), written),
	}, nil
}

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
