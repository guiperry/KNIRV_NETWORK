//go:build ignore

package tests

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/apache/arrow/go/v14/arrow"
	"github.com/apache/arrow/go/v14/arrow/array"
	"github.com/apache/arrow/go/v14/arrow/ipc"
	"github.com/apache/arrow/go/v14/arrow/memory"
	"github.com/knirvcorp/knirvbase/pkg/knirvbase"
	"github.com/knirvcorp/knirvbase/pkg/nrv"

	"data-encoder/internal/encoder"
	"data-encoder/internal/writer"
)

// writeArrowFile writes test SecurityRecords as an Arrow IPC stream file
// matching the schema expected by loadArrowBatch in nrv_encoder.go.
func writeArrowFile(t *testing.T, path string, records []testSecurityRecord) {
	t.Helper()

	schema := arrow.NewSchema([]arrow.Field{
		{Name: "file_name", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "chunk_id", Type: arrow.PrimitiveTypes.Int32, Nullable: false},
		{Name: "content", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "embedding", Type: arrow.ListOf(arrow.PrimitiveTypes.Float32), Nullable: true},
		{Name: "tokens", Type: arrow.ListOf(arrow.BinaryTypes.String), Nullable: false},
		{Name: "pos_tags", Type: arrow.ListOf(arrow.PrimitiveTypes.Int32), Nullable: false},
		{Name: "dep_hashes", Type: arrow.ListOf(arrow.PrimitiveTypes.Uint32), Nullable: false},
		{Name: "security_tags", Type: arrow.ListOf(arrow.BinaryTypes.String), Nullable: false},
		{Name: "domain_sig", Type: arrow.PrimitiveTypes.Uint32, Nullable: false},
		{Name: "slot4_raw", Type: arrow.PrimitiveTypes.Uint32, Nullable: false},
	}, nil)

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create arrow file: %v", err)
	}
	defer f.Close()

	mem := memory.NewGoAllocator()
	wr := ipc.NewWriter(f, ipc.WithSchema(schema), ipc.WithAllocator(mem))
	defer wr.Close()

	b := array.NewRecordBuilder(mem, schema)
	defer b.Release()

	fileNameB := b.Field(0).(*array.StringBuilder)
	chunkIDB := b.Field(1).(*array.Int32Builder)
	contentB := b.Field(2).(*array.StringBuilder)
	embeddingB := b.Field(3).(*array.ListBuilder)
	tokensB := b.Field(4).(*array.ListBuilder)
	posTagsB := b.Field(5).(*array.ListBuilder)
	depHashesB := b.Field(6).(*array.ListBuilder)
	secTagsB := b.Field(7).(*array.ListBuilder)
	domainSigB := b.Field(8).(*array.Uint32Builder)
	slot4RawB := b.Field(9).(*array.Uint32Builder)

	for _, r := range records {
		fileNameB.Append(r.FileName)
		chunkIDB.Append(r.ChunkID)
		contentB.Append("test content")
		embeddingB.Append(true)
		tokensB.Append(true)
		tsb := tokensB.ValueBuilder().(*array.StringBuilder)
		tsb.Append("test-token")
		posTagsB.Append(true)
		ptb := posTagsB.ValueBuilder().(*array.Int32Builder)
		ptb.Append(0)
		depHashesB.Append(true)
		dhb := depHashesB.ValueBuilder().(*array.Uint32Builder)
		dhb.Append(uint32(r.ChunkID))
		secTagsB.Append(true)
		stb := secTagsB.ValueBuilder().(*array.StringBuilder)
		stb.Append("test")
		domainSigB.Append(uint32(r.DomainSig))
		slot4RawB.Append(0)
	}

	rec := b.NewRecord()
	defer rec.Release()
	if err := wr.Write(rec); err != nil {
		t.Fatalf("write arrow record: %v", err)
	}
}

type testSecurityRecord struct {
	FileName  string
	ChunkID   int32
	DomainSig uint16
}

func TestEncoderIntegration(t *testing.T) {
	// 1. Create a real KNIRVBASE instance backed by a temp directory
	dataDir := t.TempDir()
	db, err := knirvbase.New(context.Background(), knirvbase.Options{
		DataDir: dataDir,
	})
	if err != nil {
		t.Fatalf("knirvbase.New: %v", err)
	}
	defer func() {
		if err := db.Shutdown(); err != nil {
			t.Errorf("shutdown: %v", err)
		}
	}()

	// 2. Prepare test .arrow files and register them in miner_processed
	arrowDir := t.TempDir()
	minerColl := db.Collection("miner_processed")

	docCount := 3
	recordsPerDoc := 10

	for i := 0; i < docCount; i++ {
		docID := fmt.Sprintf("doc_%d", i)
		arrowPath := filepath.Join(arrowDir, fmt.Sprintf("doc_%d.arrow", i))

		var records []testSecurityRecord
		for j := 0; j < recordsPerDoc; j++ {
			records = append(records, testSecurityRecord{
				FileName:  fmt.Sprintf("test_%d.md", i),
				ChunkID:   int32(j),
				DomainSig: uint16(0x2000 + i),
			})
		}
		writeArrowFile(t, arrowPath, records)

		_, err := minerColl.Insert(context.Background(), map[string]interface{}{
			"id":    docID,
			"ready": true,
			"payload": map[string]interface{}{
				"arrow_path":  arrowPath,
				"num_records": recordsPerDoc,
				"domain_sig":  uint32(0x2000 + i),
			},
		})
		if err != nil {
			t.Fatalf("insert entry %s: %v", docID, err)
		}
	}

	// 3. Run NRVEncoder against the real KNIRVBASE DB
	enc := encoder.NewNRVEncoder(db)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := enc.Run(ctx); err != nil {
		t.Fatalf("NRVEncoder.Run failed: %v", err)
	}

	// 4. Verify encoder_output collection has expected brackets
	outputColl := db.Collection("encoder_output")
	brackets, err := outputColl.FindAll(context.Background())
	if err != nil {
		t.Fatalf("FindAll encoder_output: %v", err)
	}

	expectedBrackets := docCount * recordsPerDoc
	if len(brackets) != expectedBrackets {
		t.Fatalf("expected %d brackets in encoder_output, got %d", expectedBrackets, len(brackets))
	}

	// 5. Verify bracket structure — each should have required fields
	for i, b := range brackets {
		if _, ok := b["Projections"]; !ok {
			t.Errorf("bracket %d missing Projections", i)
		}
		if _, ok := b["DomainSig"]; !ok {
			t.Errorf("bracket %d missing DomainSig", i)
		}
		if _, ok := b["GoldenSeed"]; !ok {
			t.Errorf("bracket %d missing GoldenSeed", i)
		}
	}

	t.Logf("integration test passed: %d brackets written to encoder_output", len(brackets))
}

func TestEncoderIntegration_EmptyCollection(t *testing.T) {
	dataDir := t.TempDir()
	db, err := knirvbase.New(context.Background(), knirvbase.Options{
		DataDir: dataDir,
	})
	if err != nil {
		t.Fatalf("knirvbase.New: %v", err)
	}
	defer func() {
		if err := db.Shutdown(); err != nil {
			t.Errorf("shutdown: %v", err)
		}
	}()

	enc := encoder.NewNRVEncoder(db)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err = enc.Run(ctx)
	if err == nil {
		t.Fatal("expected error for empty miner_processed collection, got nil")
	}
}

func TestEncoderIntegration_SkipsUnreadyEntries(t *testing.T) {
	dataDir := t.TempDir()
	db, err := knirvbase.New(context.Background(), knirvbase.Options{
		DataDir: dataDir,
	})
	if err != nil {
		t.Fatalf("knirvbase.New: %v", err)
	}
	defer func() {
		if err := db.Shutdown(); err != nil {
			t.Errorf("shutdown: %v", err)
		}
	}()

	arrowDir := t.TempDir()
	minerColl := db.Collection("miner_processed")

	// Insert an unready entry (should be skipped)
	arrowPath := filepath.Join(arrowDir, "unready.arrow")
	writeArrowFile(t, arrowPath, []testSecurityRecord{
		{FileName: "test.md", ChunkID: 0, DomainSig: 0x2000},
	})
	_, err = minerColl.Insert(context.Background(), map[string]interface{}{
		"id":    "unready_doc",
		"ready": false,
		"payload": map[string]interface{}{
			"arrow_path": arrowPath,
		},
	})
	if err != nil {
		t.Fatalf("insert unready entry: %v", err)
	}

	// Insert a ready entry
	readyPath := filepath.Join(arrowDir, "ready.arrow")
	writeArrowFile(t, readyPath, []testSecurityRecord{
		{FileName: "ready.md", ChunkID: 0, DomainSig: 0x2000},
	})
	_, err = minerColl.Insert(context.Background(), map[string]interface{}{
		"id":    "ready_doc",
		"ready": true,
		"payload": map[string]interface{}{
			"arrow_path": readyPath,
		},
	})
	if err != nil {
		t.Fatalf("insert ready entry: %v", err)
	}

	enc := encoder.NewNRVEncoder(db)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := enc.Run(ctx); err != nil {
		t.Fatalf("NRVEncoder.Run failed: %v", err)
	}

	outputColl := db.Collection("encoder_output")
	brackets, err := outputColl.FindAll(context.Background())
	if err != nil {
		t.Fatalf("FindAll encoder_output: %v", err)
	}

	// Only the ready entry (1 record) should produce brackets
	if len(brackets) != 1 {
		t.Fatalf("expected 1 bracket (from ready entry only), got %d", len(brackets))
	}
}

func TestWriterWriteBracket_Integration(t *testing.T) {
	dataDir := t.TempDir()
	db, err := knirvbase.New(context.Background(), knirvbase.Options{
		DataDir: dataDir,
	})
	if err != nil {
		t.Fatalf("knirvbase.New: %v", err)
	}
	defer func() {
		if err := db.Shutdown(); err != nil {
			t.Errorf("shutdown: %v", err)
		}
	}()

	outputColl := db.Collection("encoder_output")
	w := writer.NewNRVWriter(outputColl)

	bracket := &nrv.Bracket{
		Projections: [32]byte{1, 2, 3, 4},
		Syntactic:   0x42,
		DepHead:     5,
		IntentFlags: 0x01,
		DomainSig:   0x2000,
		Memory:      [14]byte{10, 20, 30},
		GoldenSeed:  12345,
	}

	if err := w.WriteBracket(bracket); err != nil {
		t.Fatalf("WriteBracket failed: %v", err)
	}

	// Verify the bracket was persisted
	brackets, err := outputColl.FindAll(context.Background())
	if err != nil {
		t.Fatalf("FindAll: %v", err)
	}
	if len(brackets) != 1 {
		t.Fatalf("expected 1 bracket, got %d", len(brackets))
	}

	t.Logf("writer integration test passed: bracket persisted correctly")
}
