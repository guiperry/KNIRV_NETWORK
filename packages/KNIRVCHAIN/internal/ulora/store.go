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

func (s *Store) Open(hash string) (*os.File, error) {
	if !sha256Hex.MatchString(hash) {
		return nil, fmt.Errorf("invalid bundle hash")
	}
	return os.Open(filepath.Join(s.root, hash))
}
