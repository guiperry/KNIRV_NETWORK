package writer

import (
	"context"
	"fmt"

	"github.com/knirvcorp/knirvbase/pkg/knirvbase"
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
	collection knirvbase.Collection
}

// NewMDWriter creates a new MDWriter for the given collection.
func NewMDWriter(collection knirvbase.Collection) *MDWriter {
	return &MDWriter{collection: collection}
}

// WriteChunk writes the chunk's raw content as a .md document to the
// KNIRVBASE collection.
func (w *MDWriter) WriteChunk(chunk *Chunk) error {
	if _, err := w.collection.Insert(context.Background(), map[string]interface{}{
		"id":       chunk.ChunkId,
		"payload":  map[string]interface{}{"chunk_id": chunk.ChunkId},
		"raw_data": string(chunk.Data),
	}); err != nil {
		return fmt.Errorf("insert chunk %s: %w", chunk.ChunkId, err)
	}
	return nil
}
