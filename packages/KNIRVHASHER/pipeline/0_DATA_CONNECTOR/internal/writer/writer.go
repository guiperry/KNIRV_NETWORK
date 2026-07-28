package writer

import (
	"fmt"
	hasherpb "knirvhasher/proto/hasher/training/v1"
	"os"
	"path/filepath"
	"sync"
)

// MDWriter saves each decrypted chunk as a raw .md file in the connector_raw
// KNIRVBASE collection. No normalisation or encoding happens here.
type MDWriter struct {
	path string
	mu   sync.Mutex
}

// NewMDWriter creates a new MDWriter for the given collection.
func NewMDWriter(path string) *MDWriter {
	return &MDWriter{path: path}
}

// WriteChunk decrypts the chunk (placeholder for now) and writes the raw
// content as a .md file to the KNIRVBASE collection.
func (w *MDWriter) WriteChunk(chunk *hasherpb.EncryptedChunk) error {
	// TODO: Implement PQC decryption here
	// For now, assume chunk.Data is already decrypted
	decryptedData := chunk.Data

	if err := os.MkdirAll(filepath.Dir(w.path), 0755); err != nil {
		return err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := os.WriteFile(filepath.Join(filepath.Dir(w.path), fmt.Sprintf("%s.md", chunk.ChunkId)), decryptedData, 0644); err != nil {
		return fmt.Errorf("write chunk %s: %w", chunk.ChunkId, err)
	}

	return nil
}
