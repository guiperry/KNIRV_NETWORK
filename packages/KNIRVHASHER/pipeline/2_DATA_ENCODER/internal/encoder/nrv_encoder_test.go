package encoder

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

	"data-encoder/internal"
)

func writeTestArrowFile(t *testing.T, path string, records []*internal.SecurityRecord) {
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
		tsb.Append("token1")
		posTagsB.Append(true)
		ptb := posTagsB.ValueBuilder().(*array.Int32Builder)
		ptb.Append(0)
		depHashesB.Append(true)
		dhb := depHashesB.ValueBuilder().(*array.Uint32Builder)
		dhb.Append(0)
		secTagsB.Append(true)
		stb := secTagsB.ValueBuilder().(*array.StringBuilder)
		stb.Append("test")
		domainSigB.Append(r.DomainSig)
		slot4RawB.Append(0)
	}

	rec := b.NewRecord()
	defer rec.Release()
	if err := wr.Write(rec); err != nil {
		t.Fatalf("write arrow record: %v", err)
	}
}

func TestLoadArrowBatch_DecodesIPC(t *testing.T) {
	dir := t.TempDir()
	arrowPath := filepath.Join(dir, "test.arrow")

	input := []*internal.SecurityRecord{
		{FileName: "doc1.md", ChunkID: 0, DomainSig: 0x2000},
		{FileName: "doc1.md", ChunkID: 1, DomainSig: 0x2000},
		{FileName: "doc2.md", ChunkID: 0, DomainSig: 0x1000},
	}
	writeTestArrowFile(t, arrowPath, input)

	records, err := loadArrowBatch(arrowPath)
	if err != nil {
		t.Fatalf("loadArrowBatch failed: %v", err)
	}
	if len(records) != len(input) {
		t.Fatalf("expected %d records, got %d", len(input), len(records))
	}
	for i, rec := range records {
		if rec.FileName != input[i].FileName {
			t.Errorf("record %d FileName: expected %q, got %q", i, input[i].FileName, rec.FileName)
		}
		if rec.ChunkID != input[i].ChunkID {
			t.Errorf("record %d ChunkID: expected %d, got %d", i, input[i].ChunkID, rec.ChunkID)
		}
		if rec.DomainSig != input[i].DomainSig {
			t.Errorf("record %d DomainSig: expected %d, got %d", i, input[i].DomainSig, rec.DomainSig)
		}
	}
}

func TestLoadArrowBatch_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	arrowPath := filepath.Join(dir, "empty.arrow")

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

	f, err := os.Create(arrowPath)
	if err != nil {
		t.Fatalf("create arrow file: %v", err)
	}

	mem := memory.NewGoAllocator()
	wr := ipc.NewWriter(f, ipc.WithSchema(schema), ipc.WithAllocator(mem))
	wr.Close()
	f.Close()

	_, err = loadArrowBatch(arrowPath)
	if err == nil {
		t.Fatal("expected error for empty arrow file, got nil")
	}
}

func TestLoadArrowBatch_FileNotFound(t *testing.T) {
	_, err := loadArrowBatch("/nonexistent/path.arrow")
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
}

func TestRetryWithBackoff_SuccessOnThirdAttempt(t *testing.T) {
	ctx := context.Background()
	attempts := 0

	err := retryWithBackoff(ctx, func() error {
		attempts++
		if attempts < 3 {
			return fmt.Errorf("attempt %d failed", attempts)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

func TestRetryWithBackoff_AllFail(t *testing.T) {
	ctx := context.Background()
	attempts := 0

	err := retryWithBackoff(ctx, func() error {
		attempts++
		return fmt.Errorf("always fail")
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if attempts != maxRetries+1 {
		t.Fatalf("expected %d attempts (maxRetries+1), got %d", maxRetries+1, attempts)
	}
}

func TestRetryWithBackoff_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := retryWithBackoff(ctx, func() error {
		return fmt.Errorf("fail")
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestWaitForCollectionReady_Timeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	mock := &mockCollectionEmpty{}
	_, err := WaitForCollectionReady(ctx, mock)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestWaitForCollectionReady_Success(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mock := &mockCollectionDelayed{delay: 100 * time.Millisecond}
	count, err := WaitForCollectionReady(ctx, mock)
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 entry, got %d", count)
	}
}

type mockCollectionEmpty struct{}

func (m *mockCollectionEmpty) Insert(ctx context.Context, doc map[string]interface{}) (map[string]interface{}, error) {
	return doc, nil
}
func (m *mockCollectionEmpty) Update(ctx context.Context, id string, update map[string]interface{}) (int, error) {
	return 0, nil
}
func (m *mockCollectionEmpty) Delete(ctx context.Context, id string) (int, error) {
	return 0, nil
}
func (m *mockCollectionEmpty) Find(ctx context.Context, id string) (map[string]interface{}, error) {
	return nil, fmt.Errorf("not found")
}
func (m *mockCollectionEmpty) FindAll(ctx context.Context) ([]map[string]interface{}, error) {
	return []map[string]interface{}{}, nil
}
func (m *mockCollectionEmpty) AttachToNetwork(networkID string) error { return nil }
func (m *mockCollectionEmpty) DetachFromNetwork() error              { return nil }
func (m *mockCollectionEmpty) ForceSync() error                      { return nil }

type mockCollectionDelayed struct {
	delay time.Duration
	calls int
}

func (m *mockCollectionDelayed) Insert(ctx context.Context, doc map[string]interface{}) (map[string]interface{}, error) {
	return doc, nil
}
func (m *mockCollectionDelayed) Update(ctx context.Context, id string, update map[string]interface{}) (int, error) {
	return 0, nil
}
func (m *mockCollectionDelayed) Delete(ctx context.Context, id string) (int, error) {
	return 0, nil
}
func (m *mockCollectionDelayed) Find(ctx context.Context, id string) (map[string]interface{}, error) {
	return nil, fmt.Errorf("not found")
}
func (m *mockCollectionDelayed) FindAll(ctx context.Context) ([]map[string]interface{}, error) {
	m.calls++
	if m.calls < 3 {
		return []map[string]interface{}{}, nil
	}
	return []map[string]interface{}{{"id": "doc1", "ready": true}}, nil
}
func (m *mockCollectionDelayed) AttachToNetwork(networkID string) error { return nil }
func (m *mockCollectionDelayed) DetachFromNetwork() error              { return nil }
func (m *mockCollectionDelayed) ForceSync() error                      { return nil }

type mockCollectionWithEntries struct {
	entries []map[string]interface{}
}

func (m *mockCollectionWithEntries) Insert(ctx context.Context, doc map[string]interface{}) (map[string]interface{}, error) {
	return doc, nil
}
func (m *mockCollectionWithEntries) Update(ctx context.Context, id string, update map[string]interface{}) (int, error) {
	return 0, nil
}
func (m *mockCollectionWithEntries) Delete(ctx context.Context, id string) (int, error) {
	return 0, nil
}
func (m *mockCollectionWithEntries) Find(ctx context.Context, id string) (map[string]interface{}, error) {
	return nil, fmt.Errorf("not found")
}
func (m *mockCollectionWithEntries) FindAll(ctx context.Context) ([]map[string]interface{}, error) {
	return m.entries, nil
}
func (m *mockCollectionWithEntries) AttachToNetwork(networkID string) error { return nil }
func (m *mockCollectionWithEntries) DetachFromNetwork() error              { return nil }
func (m *mockCollectionWithEntries) ForceSync() error                      { return nil }

type mockDB struct {
	collections map[string]knirvbase.Collection
}

func (m *mockDB) Collection(name string) knirvbase.Collection {
	return m.collections[name]
}

func TestNRVEncoder_RunCollectsAllEntries(t *testing.T) {
	dir := t.TempDir()

	entries := []map[string]interface{}{
		{
			"id":    "doc1",
			"ready": true,
			"payload": map[string]interface{}{
				"arrow_path": filepath.Join(dir, "doc1.arrow"),
			},
		},
		{
			"id":    "doc2",
			"ready": true,
			"payload": map[string]interface{}{
				"arrow_path": filepath.Join(dir, "doc2.arrow"),
			},
		},
		{
			"id":    "doc3",
			"ready": true,
			"payload": map[string]interface{}{
				"arrow_path": filepath.Join(dir, "doc3.arrow"),
			},
		},
	}

	for _, entry := range entries {
		payload := entry["payload"].(map[string]interface{})
		arrowPath := payload["arrow_path"].(string)
		writeTestArrowFile(t, arrowPath, []*internal.SecurityRecord{
			{FileName: "test.md", ChunkID: 0, DomainSig: 0x2000},
		})
	}

	minerColl := &mockCollectionWithEntries{entries: entries}
	outputColl := &mockCollectionWithEntries{entries: []map[string]interface{}{}}
	db := &mockDB{collections: map[string]knirvbase.Collection{
		"miner_processed": minerColl,
		"encoder_output":  outputColl,
	}}

	enc := NewNRVEncoder(db)
	err := enc.Run(context.Background())
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
}

func TestNRVEncoder_RunEmptyCollection(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	minerColl := &mockCollectionEmpty{}
	outputColl := &mockCollectionWithEntries{}
	db := &mockDB{collections: map[string]knirvbase.Collection{
		"miner_processed": minerColl,
		"encoder_output":  outputColl,
	}}

	enc := NewNRVEncoder(db)
	err := enc.Run(ctx)
	if err == nil {
		t.Fatal("expected error for empty collection, got nil")
	}
}

func TestNRVEncoder_SkipsUnreadyEntries(t *testing.T) {
	dir := t.TempDir()

	entries := []map[string]interface{}{
		{
			"id":    "doc1",
			"ready": false,
			"payload": map[string]interface{}{
				"arrow_path": filepath.Join(dir, "doc1.arrow"),
			},
		},
	}

	writeTestArrowFile(t, filepath.Join(dir, "doc1.arrow"), []*internal.SecurityRecord{
		{FileName: "test.md", ChunkID: 0, DomainSig: 0x2000},
	})

	minerColl := &mockCollectionWithEntries{entries: entries}
	outputColl := &mockCollectionWithEntries{entries: []map[string]interface{}{}}
	db := &mockDB{collections: map[string]knirvbase.Collection{
		"miner_processed": minerColl,
		"encoder_output":  outputColl,
	}}

	enc := NewNRVEncoder(db)
	err := enc.Run(context.Background())
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
}

func TestSecurityRecordToSlotVector(t *testing.T) {
	sr := &internal.SecurityRecord{
		FileName:  "test.md",
		ChunkID:   1,
		DomainSig: 0x2000,
	}
	sv := sr.ToSlotVector()
	if sv == nil {
		t.Fatal("ToSlotVector returned nil")
	}
}
