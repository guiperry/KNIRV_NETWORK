package writer

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/apache/arrow/go/v14/arrow"
	"github.com/apache/arrow/go/v14/arrow/array"
	"github.com/apache/arrow/go/v14/arrow/ipc"
	"github.com/apache/arrow/go/v14/arrow/memory"

	"data-mapper/internal/normalizer"
)

// securitySchema is the Arrow IPC schema written by ArrowWriter.
var securitySchema = arrow.NewSchema([]arrow.Field{
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

// ArrowWriter serialises batches of SecurityRecords as Arrow IPC files on disk.
type ArrowWriter struct {
	outputDir string
}

// NewArrowWriter returns an ArrowWriter that writes <docID>.arrow files into
// outputDir, creating the directory if it does not exist.
func NewArrowWriter(outputDir string) *ArrowWriter {
	return &ArrowWriter{outputDir: outputDir}
}

// SetKnirvBaseCollection attaches a KNIRVBASE collection for registration.
// WriteBatch encodes records as a single Arrow IPC file named <docID>.arrow.
func (w *ArrowWriter) WriteBatch(docID string, records []*normalizer.SecurityRecord) error {
	if err := os.MkdirAll(w.outputDir, 0o755); err != nil {
		return fmt.Errorf("create output dir %s: %w", w.outputDir, err)
	}

	outPath := filepath.Join(w.outputDir, sanitiseID(docID)+".arrow")
	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("create %s: %w", outPath, err)
	}
	defer f.Close()

	mem := memory.NewGoAllocator()
	wr := ipc.NewWriter(f, ipc.WithSchema(securitySchema), ipc.WithAllocator(mem))
	defer wr.Close()

	rec, err := buildSecurityRecord(records, mem)
	if err != nil {
		return fmt.Errorf("build arrow record: %w", err)
	}
	defer rec.Release()

	if err := wr.Write(rec); err != nil {
		return fmt.Errorf("write arrow record: %w", err)
	}

	return nil
}

func extractDomainSig(records []*normalizer.SecurityRecord) uint32 {
	if len(records) == 0 {
		return 0
	}
	return uint32(records[0].DomainSig)
}

func buildSecurityRecord(records []*normalizer.SecurityRecord, mem memory.Allocator) (arrow.Record, error) {
	b := array.NewRecordBuilder(mem, securitySchema)
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
		contentB.Append(r.Content)

		// embedding — written as empty list; 2_DATA_ENCODER fills this in.
		embeddingB.Append(true)

		// tokens
		tokensB.Append(true)
		tsb := tokensB.ValueBuilder().(*array.StringBuilder)
		for _, t := range r.Tokens {
			tsb.Append(t)
		}

		// pos_tags
		posTagsB.Append(true)
		ptb := posTagsB.ValueBuilder().(*array.Int32Builder)
		for _, p := range r.POSTags {
			ptb.Append(int32(p))
		}

		// dep_hashes
		depHashesB.Append(true)
		dhb := depHashesB.ValueBuilder().(*array.Uint32Builder)
		for _, d := range r.DepHashes {
			dhb.Append(d)
		}

		// security_tags
		secTagsB.Append(true)
		stb := secTagsB.ValueBuilder().(*array.StringBuilder)
		for _, st := range r.SecurityTags {
			stb.Append(st)
		}

		domainSigB.Append(uint32(r.DomainSig))
		slot4RawB.Append(r.Slot4Raw)
	}

	return b.NewRecord(), nil
}

// sanitiseID replaces characters that are invalid in file-system paths.
func sanitiseID(id string) string {
	out := make([]byte, len(id))
	for i := 0; i < len(id); i++ {
		switch id[i] {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			out[i] = '_'
		default:
			out[i] = id[i]
		}
	}
	return string(out)
}
