package encoder

import (
	"bytes"
	"fmt"

	"github.com/apache/arrow/go/v18/arrow"
	"github.com/apache/arrow/go/v18/arrow/array"
	"github.com/apache/arrow/go/v18/arrow/ipc"
	"github.com/apache/arrow/go/v18/arrow/memory"
	"github.com/knirvhasher/pipeline/0_DATA_CONNECTOR/internal/normalizer"
)

type ArrowEncoder struct {
	schema *arrow.Schema
	pool   memory.Allocator
}

func NewArrowEncoder() *ArrowEncoder {
	schema := arrow.NewSchema([]arrow.Field{
		{Name: "file_name", Type: &arrow.StringType{}, Nullable: false},
		{Name: "chunk_id", Type: arrow.PrimitiveTypes.Int32, Nullable: false},
		{Name: "content", Type: &arrow.StringType{}, Nullable: false},
		{Name: "tokens", Type: arrow.ListOf(&arrow.StringType{}), Nullable: false},
		{Name: "pos_tags", Type: arrow.ListOf(arrow.PrimitiveTypes.Int32), Nullable: false},
		{Name: "dep_hashes", Type: arrow.ListOf(arrow.PrimitiveTypes.Uint32), Nullable: false},
		{Name: "security_tags", Type: arrow.ListOf(&arrow.StringType{}), Nullable: false},
		{Name: "slot4", Type: arrow.PrimitiveTypes.Int32, Nullable: false},
		{Name: "slot10", Type: arrow.PrimitiveTypes.Int32, Nullable: false},
		{Name: "weight", Type: arrow.PrimitiveTypes.Float64, Nullable: false},
		{Name: "embedding", Type: arrow.ListOf(arrow.PrimitiveTypes.Float32), Nullable: false},
	}, nil)

	return &ArrowEncoder{
		schema: schema,
		pool:   memory.NewGoAllocator(),
	}
}

func (e *ArrowEncoder) Encode(records []*normalizer.SecurityRecord) (arrow.Table, error) {
	bldr := NewRecordBuilder(e.pool, e.schema)
	defer bldr.Release()

	for _, rec := range records {
		if err := bldr.Append(rec); err != nil {
			return nil, fmt.Errorf("append record: %w", err)
		}
	}

	rec, err := bldr.NewRecord()
	if err != nil {
		return nil, fmt.Errorf("new record: %w", err)
	}
	defer rec.Release()

	return array.NewTableFromRecords(e.schema, []arrow.Record{rec}), nil
}

func (e *ArrowEncoder) EncodeToBytes(records []*normalizer.SecurityRecord) ([]byte, error) {
	table, err := e.Encode(records)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	writer, err := ipc.NewFileWriter(&buf, ipc.WithSchema(e.schema))
	if err != nil {
		return nil, err
	}
	defer writer.Close()

	reader := array.NewTableReader(table, table.NumRows())
	defer reader.Release()

	for reader.Next() {
		rec := reader.Record()
		if err := writer.Write(rec); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

type RecordBuilder struct {
	pool       memory.Allocator
	schema     *arrow.Schema
	fieldBldrs []interface {
		Append(interface{}) error
		NewArray() array.Builder
		Release()
	}
}

func NewRecordBuilder(pool memory.Allocator, schema *arrow.Schema) *RecordBuilder {
	rb := &RecordBuilder{
		pool:   pool,
		schema: schema,
	}

	for _, field := range schema.Fields() {
		switch field.Type.ID() {
		case arrow.INT32:
			bldr := array.NewInt32Builder(pool)
			rb.fieldBldrs = append(rb.fieldBldrs, &int32Bldr{bldr})
		case arrow.INT64:
			bldr := array.NewInt64Builder(pool)
			rb.fieldBldrs = append(rb.fieldBldrs, &int64Bldr{bldr})
		case arrow.UINT32:
			bldr := array.NewUint32Builder(pool)
			rb.fieldBldrs = append(rb.fieldBldrs, &uint32Bldr{bldr})
		case arrow.FLOAT64:
			bldr := array.NewFloat64Builder(pool)
			rb.fieldBldrs = append(rb.fieldBldrs, &float64Bldr{bldr})
		case arrow.STRING:
			bldr := array.NewStringBuilder(pool)
			rb.fieldBldrs = append(rb.fieldBldrs, &stringBldr{bldr})
		case arrow.LIST:
			bldr := array.NewListBuilder(pool, field.Type.(*arrow.ListType).Elem())
			rb.fieldBldrs = append(rb.fieldBldrs, &listBldr{builder: bldr})
		}
	}

	return rb
}

func (rb *RecordBuilder) Append(rec *normalizer.SecurityRecord) error {
	if err := rb.fieldBldrs[0].Append(rec.FileName); err != nil {
		return err
	}
	if err := rb.fieldBldrs[1].Append(int32(rec.ChunkID)); err != nil {
		return err
	}
	if err := rb.fieldBldrs[2].Append(rec.Content); err != nil {
		return err
	}

	if err := rb.fieldBldrs[3].Append(rec.Tokens); err != nil {
		return err
	}
	if err := rb.fieldBldrs[4].Append(rec.POSTags); err != nil {
		return err
	}
	if err := rb.fieldBldrs[5].Append(rec.DepHashes); err != nil {
		return err
	}
	if err := rb.fieldBldrs[6].Append(rec.SecurityTags); err != nil {
		return err
	}

	if err := rb.fieldBldrs[7].Append(int32(rec.Slot4)); err != nil {
		return err
	}
	if err := rb.fieldBldrs[8].Append(int32(rec.Slot10)); err != nil {
		return err
	}
	if err := rb.fieldBldrs[9].Append(rec.Weight); err != nil {
		return err
	}

	if err := rb.fieldBldrs[10].Append(rec.GetEmbedding()); err != nil {
		return err
	}

	return nil
}

func (rb *RecordBuilder) NewRecord() (arrow.Record, error) {
	bldrs := make([]array.Builder, len(rb.fieldBldrs))
	for i, bldr := range rb.fieldBldrs {
		bldrs[i] = bldr.NewArray()
		defer bldrs[i].Release()
	}

	cols := make([]arrow.Array, len(bldrs))
	for i, bldr := range bldrs {
		cols[i] = bldr.NewArray()
		defer cols[i].Release()
	}

	record := array.NewRecord(rb.schema, cols, int64(cols[0].Len()))
	return record, nil
}

func (rb *RecordBuilder) Release() {
	for _, bldr := range rb.fieldBldrs {
		bldr.Release()
	}
}

type int32Bldr struct{ bldr *array.Int32Builder }

func (b *int32Bldr) Append(v interface{}) error { b.bldr.Append(v.(int32)); return nil }
func (b *int32Bldr) NewArray() array.Builder    { return b.bldr }
func (b *int32Bldr) Release()                   { b.bldr.Release() }

type int64Bldr struct{ bldr *array.Int64Builder }

func (b *int64Bldr) Append(v interface{}) error { b.bldr.Append(v.(int64)); return nil }
func (b *int64Bldr) NewArray() array.Builder    { return b.bldr }
func (b *int64Bldr) Release()                   { b.bldr.Release() }

type uint32Bldr struct{ bldr *array.Uint32Builder }

func (b *uint32Bldr) Append(v interface{}) error { b.bldr.Append(v.(uint32)); return nil }
func (b *uint32Bldr) NewArray() array.Builder    { return b.bldr }
func (b *uint32Bldr) Release()                   { b.bldr.Release() }

type float64Bldr struct{ bldr *array.Float64Builder }

func (b *float64Bldr) Append(v interface{}) error { b.bldr.Append(v.(float64)); return nil }
func (b *float64Bldr) NewArray() array.Builder    { return b.bldr }
func (b *float64Bldr) Release()                   { b.bldr.Release() }

type stringBldr struct{ bldr *array.StringBuilder }

func (b *stringBldr) Append(v interface{}) error { b.bldr.Append(v.(string)); return nil }
func (b *stringBldr) NewArray() array.Builder    { return b.bldr }
func (b *stringBldr) Release()                   { b.bldr.Release() }

type listBldr struct{ builder *array.ListBuilder }

func (b *listBldr) Append(v interface{}) error {
	slice := v.([]interface{})
	b.builder.Append(true)
	valBldr := b.builder.ValueBuilder()
	switch slice[0].(type) {
	case string:
		inner := valBldr.(*array.StringBuilder)
		for _, item := range slice {
			inner.Append(item.(string))
		}
	case int:
		inner := valBldr.(*array.Int32Builder)
		for _, item := range slice {
			inner.Append(int32(item.(int)))
		}
	case uint32:
		inner := valBldr.(*array.Uint32Builder)
		for _, item := range slice {
			inner.Append(item.(uint32))
		}
	case float32:
		inner := valBldr.(*array.Float32Builder)
		for _, item := range slice {
			inner.Append(item.(float32))
		}
	}
	return nil
}
func (b *listBldr) NewArray() array.Builder { return b.builder }
func (b *listBldr) Release()                { b.builder.Release() }
