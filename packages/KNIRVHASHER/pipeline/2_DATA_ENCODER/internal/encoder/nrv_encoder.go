package encoder

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"data-encoder/internal"
	"data-encoder/internal/writer"
	"data-encoder/pkg/nrvio"
	"data-encoder/pkg/store"
	"github.com/apache/arrow/go/v14/arrow/array"
	"github.com/apache/arrow/go/v14/arrow/ipc"
)

type DB interface {
	Collection(name string) store.Collection
}

const VectorDims = 16

const (
	defaultPollInterval = 500 * time.Millisecond
	defaultWaitTimeout  = 5 * time.Minute
)

type NRVEncoder struct {
	db     DB
	writer *writer.NRVWriter
	packer *internal.TensorPacker
}

func NewNRVEncoder(db DB) *NRVEncoder {
	return &NRVEncoder{
		db:     db,
		writer: writer.NewNRVWriter(db.Collection("encoder_output")),
		packer: internal.NewTensorPacker(),
	}
}

func (e *NRVEncoder) Run(ctx context.Context) error {
	log.Printf("[ENCODER] PHASE 1/4: Waiting for miner_processed collection...")
	coll := e.db.Collection("miner_processed")
	count, err := WaitForCollectionReady(ctx, coll)
	if err != nil {
		return fmt.Errorf("encoder aborted: %w", err)
	}
	log.Printf("[ENCODER] PHASE 1/4: complete - %d entries found", count)

	log.Printf("[ENCODER] PHASE 2/4: Loading %d entries from collection...", count)
	var entries []map[string]interface{}
	if err := retryWithBackoff(ctx, func() error {
		var findErr error
		entries, findErr = coll.FindAll(ctx)
		return findErr
	}); err != nil {
		return fmt.Errorf("FindAll after retries: %w", err)
	}
	log.Printf("[ENCODER] PHASE 2/4: complete - loaded %d entries", len(entries))

	bracketCount := 0
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}

		ready, ok := entry["ready"].(bool)
		if !ok || !ready {
			log.Printf("nrv_encoder: skipping entry %v (not ready)", entry["id"])
			continue
		}

		payload, ok := entry["payload"].(map[string]interface{})
		if !ok {
			log.Printf("nrv_encoder: entry %v has no payload, skipping", entry["id"])
			continue
		}
		arrowPath, ok := payload["arrow_path"].(string)
		if !ok || arrowPath == "" {
			log.Printf("nrv_encoder: entry %v has no arrow_path, skipping", entry["id"])
			continue
		}

		var records []*internal.SecurityRecord
		if err := retryWithBackoff(ctx, func() error {
			var loadErr error
			records, loadErr = loadArrowBatch(arrowPath)
			return loadErr
		}); err != nil {
			log.Printf("nrv_encoder: skipping %s after retries: %v", arrowPath, err)
			continue
		}

		log.Printf("[ENCODER] PHASE 3/4: Encoding %d records from %s...", len(records), arrowPath)
		for i, rec := range records {
			slots := e.packer.Orchestrate(rec.ToSlotVector(), uint16(i), uint16(rec.DomainSig))

			projBytes := internal.SlotsToProjections(slots[:4])
			var proj [32]byte
			copy(proj[:], projBytes)

			memBytes := internal.Slots6to8(slots[6:9])
			var mem [14]byte
			copy(mem[:], memBytes)

			bracket := &nrvio.Bracket{
				Projections: proj,
				POSTag:      uint8(slots[4] & 0xFF),
				DepHead:     uint8(slots[5]),
				IntentFlags: uint8(slots[9]),
				DomainSig:   uint16(slots[10]),
				Memory:      mem,
				GoldenSeed:  slots[11],
			}
			if err := e.writer.WriteBracket(bracket); err != nil {
				log.Printf("nrv_encoder: write bracket %d: %v", i, err)
				continue
			}
			bracketCount++
		}
	}

	log.Printf("[ENCODER] PHASE 4/4: Writing %d brackets to encoder_output...", bracketCount)
	log.Printf("nrv_encoder: completed, processed %d entries, wrote %d brackets", len(entries), bracketCount)
	return nil
}

func loadArrowBatch(path string) ([]*internal.SecurityRecord, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open arrow file: %w", err)
	}
	defer f.Close()

	reader, err := ipc.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("new arrow reader: %w", err)
	}

	schema := reader.Schema()
	colIndex := make(map[string]int)
	for i, field := range schema.Fields() {
		colIndex[field.Name] = i
	}

	var records []*internal.SecurityRecord
	for reader.Next() {
		rec := reader.Record()
		for row := 0; row < int(rec.NumRows()); row++ {
			sr := &internal.SecurityRecord{}
			if idx, ok := colIndex["file_name"]; ok {
				if col := rec.Column(idx); col != nil {
					if arr, ok := col.(*array.String); ok {
						sr.FileName = arr.Value(row)
					}
				}
			}
			if idx, ok := colIndex["chunk_id"]; ok {
				if col := rec.Column(idx); col != nil {
					if arr, ok := col.(*array.Int32); ok {
						sr.ChunkID = arr.Value(row)
					}
				}
			}
			if idx, ok := colIndex["domain_sig"]; ok {
				if col := rec.Column(idx); col != nil {
					if arr, ok := col.(*array.Uint32); ok {
						sr.DomainSig = arr.Value(row)
					}
				}
			}
			records = append(records, sr)
		}
	}

	if err := reader.Err(); err != nil {
		return nil, fmt.Errorf("read arrow stream: %w", err)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("no SecurityRecords found in %s", path)
	}
	return records, nil
}

func WaitForCollectionReady(ctx context.Context, coll store.Collection) (int, error) {
	deadline, cancel := context.WithTimeout(ctx, defaultWaitTimeout)
	defer cancel()

	ticker := time.NewTicker(defaultPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-deadline.Done():
			return 0, fmt.Errorf("timeout waiting for miner_processed: %w", deadline.Err())
		case <-ticker.C:
			entries, err := coll.FindAll(ctx)
			if err != nil {
				log.Printf("readiness: transient error on FindAll: %v", err)
				continue
			}
			if len(entries) > 0 {
				log.Printf("readiness: miner_processed ready with %d entries", len(entries))
				return len(entries), nil
			}
			log.Printf("readiness: miner_processed empty, retrying...")
		}
	}
}
