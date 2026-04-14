package writer

import (
	"fmt"
	"path/filepath"

	"github.com/knirvcorp/knirvbase"
	hasherpb "github.com/knirvcorp/knirvserver/backend/internal/proto"
)

// MDWriter saves each decrypted chunk as a raw .md file in the connector_raw
// KNIRVBASE collection. No normalisation or encoding happens here.
type MDWriter struct {
	collection knirvbase.Collection
}

// NewMDWriter creates a new MDWriter for the given collection.
func NewMDWriter(collection knirvbase.Collection) *MDWriter {
	return &MDWriter{collection: collection}
}

// WriteChunk decrypts the chunk (placeholder for now) and writes the raw
// content as a .md file to the KNIRVBASE collection.
func (w *MDWriter) WriteChunk(chunk *hasherpb.EncryptedChunk) error {
	// TODO: Implement PQC decryption here
	// For now, assume chunk.Data is already decrypted
	decryptedData := chunk.Data

	// Write as .md file in the collection
	mdPath := filepath.Join(w.collection.DataDir(), fmt.Sprintf("%s.md", chunk.ChunkId))
	if err := w.collection.Insert(map[string]interface{}{
		"id":       chunk.ChunkId,
		"payload":  map[string]interface{}{"md_path": mdPath},
		"raw_data": string(decryptedData),
	}); err != nil {
		return fmt.Errorf("insert chunk %s: %w", chunk.ChunkId, err)
	}

	return nil
}
