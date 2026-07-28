package writer

import (
	"fmt"
	"os"
	"path/filepath"
)

// Chunk is a decrypted data chunk from 0_DATA_CONNECTOR. It mirrors the
// relevant fields of knirvhasher/proto.EncryptedChunk without pulling in the
// full gRPC / protobuf dependency tree.
type Chunk struct {
	ChunkId string
	Data    []byte
	IsLast  bool
}

// MDWriter saves each decrypted chunk as a raw .md document in the
// connector_raw KNIRVBASE collection.
type MDWriter struct {
	outputDir string
}

// NewMDWriter creates a new MDWriter for the given collection.
func NewMDWriter(outputDir string) *MDWriter {
	return &MDWriter{outputDir: outputDir}
}

// WriteChunk writes the chunk's raw content as a .md document to the
// KNIRVBASE collection.
func (w *MDWriter) WriteChunk(chunk *Chunk) error {
	if err := os.MkdirAll(w.outputDir, 0755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(w.outputDir, chunk.ChunkId+".md"), chunk.Data, 0644); err != nil {
		return fmt.Errorf("insert chunk %s: %w", chunk.ChunkId, err)
	}
	return nil
}
