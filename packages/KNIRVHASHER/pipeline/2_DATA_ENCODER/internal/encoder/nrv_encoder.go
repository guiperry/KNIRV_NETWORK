package encoder

import (
	"context"

	"data-encoder/internal"
	"data-encoder/internal/writer"

	"github.com/apache/arrow/go/v14/arrow/memory"
	"github.com/knirvcorp/knirvbase/pkg/knirvbase"
)

// VectorDims is the fixed dimensionality of all BGE embedding vectors in the pipeline.
// Must match vector_mapper.go variance selection, NRV KB index, and KNIRVBASE
// Projections layout (32 bytes → 16 × uint16 dimensions).
const VectorDims = 16

// NRVEncoder reads .arrow IPC batches from the miner_processed KNIRVBASE collection
// and encodes each SecurityRecord into an 80-byte .nrv Tier-3 Bracket.
type NRVEncoder struct {
	db     knirvbase.DB
	writer *writer.NRVWriter
	packer *internal.TensorPacker
}

// NewNRVEncoder creates a new NRVEncoder instance.
func NewNRVEncoder(db knirvbase.DB) *NRVEncoder {
	return &NRVEncoder{
		db:     db,
		writer: writer.NewNRVWriter(db.Collection("encoder_output")),
		packer: internal.NewTensorPacker(),
	}
}

// Run opens a Flight stream on the miner_processed collection, reads each
// .arrow file path from the entry payload, memory-maps the IPC file, and
// encodes every SecurityRecord row into a .nrv Bracket via NRVWriter.
func (e *NRVEncoder) Run(ctx context.Context) error {
	// TODO: Implement Flight stream reading from knirvbase
	// This is a placeholder implementation
	_ = memory.NewGoAllocator()
	return nil
}
