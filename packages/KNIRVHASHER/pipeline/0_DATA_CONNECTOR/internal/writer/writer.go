package writer

import (
	"context"
	"fmt"

	"github.com/knirvcorp/knirvbase/pkg/knirvbase"
	hasherpb "knirvhasher/proto/hasher/training/v1"
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
	if _, err := w.collection.Insert(context.Background(), map[string]interface{}{
		"id":       chunk.ChunkId,
		"payload":  map[string]interface{}{"chunk_id": chunk.ChunkId},
		"raw_data": string(decryptedData),
	}); err != nil {
		return fmt.Errorf("insert chunk %s: %w", chunk.ChunkId, err)
	}

	return nil
}
