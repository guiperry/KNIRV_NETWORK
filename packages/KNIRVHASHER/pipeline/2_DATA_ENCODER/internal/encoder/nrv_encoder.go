package encoder

import (
	"context"
	"fmt"
	"log"

	"github.com/apache/arrow/go/v14/arrow/memory"
	"github.com/knirvcorp/knirvbase"
	"github.com/knirvcorp/knirvhasher/pipeline/2_DATA_ENCODER/internal/writer"
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
	packer *TensorPacker
}

// NewNRVEncoder creates a new NRVEncoder instance.
func NewNRVEncoder(db knirvbase.DB) *NRVEncoder {
	return &NRVEncoder{
		db:     db,
		writer: writer.NewNRVWriter(db.Collection("encoder_output")),
		packer: NewTensorPacker(),
	}
}

// Run opens a Flight stream on the miner_processed collection, reads each
// .arrow file path from the entry payload, memory-maps the IPC file, and
// encodes every SecurityRecord row into a .nrv Bracket via NRVWriter.
func (e *NRVEncoder) Run(ctx context.Context) error {
	stream, err := e.db.Collection("miner_processed").FlightStream(ctx)
	if err != nil {
		return fmt.Errorf("open miner_processed flight stream: %w", err)
	}

	pool := memory.NewGoAllocator()
	defer pool.Release()

	for entry := range stream {
		arrowPath, _ := entry.Payload["arrow_path"].(string)
		records, err := loadArrowBatch(arrowPath, pool)
		if err != nil {
			log.Printf("nrv_encoder: load %s: %v", arrowPath, err)
			continue
		}

		for i, rec := range records {
			slots := e.packer.Orchestrate(rec.ToSlotVector(), uint16(i), rec.DomainSig)
			bracket := &knirvbase.Bracket{
				// Slots 0-3: 16-dim LSH projections packed as 16 × uint16 (32 bytes)
				Projections: slotsToProjections(slots),
				// Slot 4: packed syntax byte (POSTag | Tense | Plurality)
				SyntacticByte: uint8(slots[4] & 0xFF),
				// Slot 5: dependency head
				DepHead: uint8(slots[5]),
				// Slot 10: domain signature (e.g. 0x2400 = Security domain)
				DomainSig: uint16(slots[10]),
				// Slots 6-8: recursive context memory (18 bytes)
				ContextMemory: slots6to8(slots[6:9]),
				// Slot 11: GoldenSeed / LSH Salt
				GoldenSeed: slots[11],
			}
			if err := e.writer.WriteBracket(bracket); err != nil {
				log.Printf("nrv_encoder: write bracket %d: %v", i, err)
			}
		}
	}
	return nil
}

// loadArrowBatch memory-maps the IPC file and decodes SecurityRecords.
// This is a placeholder - actual implementation would use arrow IPC reader.
func loadArrowBatch(path string, pool memory.Allocator) ([]*SecurityRecord, error) {
	// Placeholder: return empty slice for now
	return []*SecurityRecord{}, nil
}

// slotsToProjections converts slots 0-3 to Bracket.Projections format.
func slotsToProjections(slots []uint32) [32]byte {
	var p [32]byte
	// Implementation would pack slots[0], slots[1], slots[2], slots[3] into 32 bytes
	return p
}

// slots6to8 packs slots 6-8 into 18-byte context memory format.
func slots6to8(slots []uint32) [18]byte {
	var cm [18]byte
	// Implementation would pack slots[6], slots[7], slots[8] into 18 bytes
	return cm
}
