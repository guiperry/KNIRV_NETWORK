// Package ulora provides the local content-addressed store used by KNIRVCHAIN
// to host portable adapter bundles. It intentionally stores raw bundle bytes;
// parsing and validating a bundle remains the compiler service's job.
package ulora

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var sha256Hex = regexp.MustCompile(`^[a-f0-9]{64}$`)

type Store struct{ root string }

func NewStore(appDataDir string) (*Store, error) {
	if appDataDir == "" {
		return nil, fmt.Errorf("app data directory is required")
	}
	root := filepath.Join(appDataDir, "ulora_blobs")
	if err := os.MkdirAll(root, 0755); err != nil {
		return nil, fmt.Errorf("create ulora blob store: %w", err)
	}
	return &Store{root: root}, nil
}

func (s *Store) Root() string { return s.root }

// Put streams bytes to a temporary file, then atomically places it under its
// real SHA-256 digest. The returned hash always describes the bytes on disk.
func (s *Store) Put(r io.Reader) (string, error) {
	tmp, err := os.CreateTemp(s.root, ".bundle-*")
	if err != nil {
		return "", fmt.Errorf("create bundle temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	h := sha256.New()
	if _, err = io.Copy(io.MultiWriter(tmp, h), r); err != nil {
		tmp.Close()
		return "", fmt.Errorf("write bundle: %w", err)
	}
	if err = tmp.Close(); err != nil {
		return "", fmt.Errorf("close bundle: %w", err)
	}
	hash := hex.EncodeToString(h.Sum(nil))
	dest := filepath.Join(s.root, hash)
	if err = os.Rename(tmpName, dest); err != nil && !os.IsExist(err) {
		return "", fmt.Errorf("store bundle: %w", err)
	}
	return hash, nil
}

// OpenDefault resolves and opens the blob store the way KNIRVCHAIN's other
// components do: KNIRV_APP_DATA_DIR when set, otherwise the blockchain
// database's parent directory, otherwise a temp directory.
//
// This exists so the bundle *writer* (the mint route in internal/blockchain)
// and the bundle *reader* (GET /api/ulora/{hash} in internal/api) resolve the
// same directory by construction. They previously each had their own inline
// derivation, which is exactly how a store ends up written in one place and
// read from another.
//
// blockchainDBPath is the live database path when the caller has one; pass ""
// to fall back to the environment.
func OpenDefault(blockchainDBPath string) (*Store, error) {
	if dir := strings.TrimSpace(os.Getenv("KNIRV_APP_DATA_DIR")); dir != "" {
		return NewStore(dir)
	}
	if path := strings.TrimSpace(blockchainDBPath); path != "" {
		return NewStore(filepath.Dir(path))
	}
	return NewStore(filepath.Join(os.TempDir(), "knirvchain"))
}

// Open streams the stored bundle under hash.
func (s *Store) Open(hash string) (*os.File, error) {
	if !sha256Hex.MatchString(hash) {
		return nil, fmt.Errorf("invalid bundle hash")
	}
	return os.Open(filepath.Join(s.root, hash))
}
