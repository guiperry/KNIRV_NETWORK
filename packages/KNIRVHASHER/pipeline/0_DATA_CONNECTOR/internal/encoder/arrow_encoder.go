package encoder

import (
	"fmt"

	"github.com/apache/arrow/go/v18/arrow"
	"github.com/apache/arrow/go/v18/arrow/ipc"
	"github.com/apache/arrow/go/v18/arrow/memory"
	"github.com/knirv/hasher/pipeline/0_DATA_CONNECTOR/internal/normalizer"
)

type ArrowEncoder struct {
	schema *arrow.Schema
	pool   memory.Allocator
}

func NewArrowEncoder() *ArrowEncoder {
	schema := arrow.NewSchema([]arrow.Field{
		{Name: "file_name", Type: arrow.PrimitiveTypes.UTF8, Nullable: false},
		{Name: "chunk_id", Type: arrow.PrimitiveTypes.Int32, Nullable: false},
		{Name: "content", Type: arrow.PrimitiveTypes.UTF8, Nullable: false},
		{Name: "tokens", Type: arrow.ListOf(arrow.PrimitiveTypes.UTF8), Nullable: false},
		{Name: "pos_tags", Type: arrow.ListOf(arrow.PrimitiveTypes.Int32), Nullable: false},
		{Name: "dep_hashes", Type: arrow.ListOf(arrow.PrimitiveTypes.Uint32), Nullable: false},
		{Name: "security_tags", Type: arrow.ListOf(arrow.PrimitiveTypes.UTF8), Nullable: false},
		{Name: "slot4", Type: arrow.PrimitiveTypes.Int32, Nullable: false},
		{Name: "slot10", Type: arrow.PrimitiveTypes.Int32, Nullable: false},
		{Name: "weight", Type: arrow.PrimitiveTypes.Float64, Nullable: false},
		{Name: "embedding", Type: arrow.ListOf(arrow.PrimitiveTypes.Float32), Nullable: false},
	})

	return &ArrowEncoder{
		schema: schema,
		pool:   memory.NewGoAllocator(),
	}
}

func (e *ArrowEncoder) Encode(records []*normalizer.SecurityRecord) (*arrow.Table, error) {
	bldr := NewRecordBuilder(e.pool, e.schema)
	defer bldr.Release()

	for _, rec := range records {
		if err := bldr.Append(rec); err != nil {
			return nil, fmt.Errorf("append record: %w", err)
		}
	}

	record, err := bldr.NewRecord()
	if err != nil {
		return nil, fmt.Errorf("new record: %w", err)
	}
	defer record.Release()

	table := arrow.NewTable(e.schema, []arrow.Column{
		*arrow.NewColumn(e.schema.Field(0), arrow.NewArrayData(record.Column(0).Data())),
		*arrow.NewColumn(e.schema.Field(1), arrow.NewArrayData(record.Column(1).Data())),
		*arrow.NewColumn(e.schema.Field(2), arrow.NewArrayData(record.Column(2).Data())),
		*arrow.NewColumn(e.schema.Field(3), arrow.NewArrayData(record.Column(3).Data())),
		*arrow.NewColumn(e.schema.Field(4), arrow.NewArrayData(record.Column(4).Data())),
		*arrow.NewColumn(e.schema.Field(5), arrow.NewArrayData(record.Column(5).Data())),
		*arrow.NewColumn(e.schema.Field(6), arrow.NewArrayData(record.Column(6).Data())),
		*arrow.NewColumn(e.schema.Field(7), arrow.NewArrayData(record.Column(7).Data())),
		*arrow.NewColumn(e.schema.Field(8), arrow.NewArrayData(record.Column(8).Data())),
		*arrow.NewColumn(e.schema.Field(9), arrow.NewArrayData(record.Column(9).Data())),
		*arrow.NewColumn(e.schema.Field(10), arrow.NewArrayData(record.Column(10).Data())),
	})

	return table, nil
}

func (e *ArrowEncoder) EncodeToBytes(records []*normalizer.SecurityRecord) ([]byte, error) {
	table, err := e.Encode(records)
	if err != nil {
		return nil, err
	}
	defer table.Release()

	var buf arrow.TableBuffer
	writer := ipc.NewWriter(&buf, ipc.WithSchema(e.schema))
	defer writer.Close()

	for i := 0; i < int(table.NumChunks()); i++ {
		for j := 0; j < int(table.NumCols()); j++ {
			col := table.Column(j)
			if err := writer.Write(*col.Data().Slice(0, col.Len())); err != nil {
				return nil, err
			}
		}
	}

	return buf.Bytes(), nil
}

type RecordBuilder struct {
	pool       memory.Allocator
	schema     *arrow.Schema
	fieldBldrs []interface {
		Append(interface{}) error
		NewArray() arrow.Array
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
			bldr := arrow.NewInt32Builder(pool)
			rb.fieldBldrs = append(rb.fieldBldrs, &int32Bldr{bldr})
		case arrow.INT64:
			bldr := arrow.NewInt64Builder(pool)
			rb.fieldBldrs = append(rb.fieldBldrs, &int64Bldr{bldr})
		case arrow.UINT32:
			bldr := arrow.NewUint32Builder(pool)
			rb.fieldBldrs = append(rb.fieldBldrs, &uint32Bldr{bldr})
		case arrow.FLOAT64:
			bldr := arrow.NewFloat64Builder(pool)
			rb.fieldBldrs = append(rb.fieldBldrs, &float64Bldr{bldr})
		case arrow.UTF8:
			bldr := arrow.NewStringBuilder(pool)
			rb.fieldBldrs = append(rb.fieldBldrs, &stringBldr{bldr})
		case arrow.LIST:
			bldr := arrow.NewListBuilder(pool, field.Type.(*arrow.ListType).Elem())
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
	arrays := make([]arrow.Array, len(rb.fieldBldrs))
	for i, bldr := range rb.fieldBldrs {
		arrays[i] = bldr.NewArray()
		defer arrays[i].Release()
	}

	return arrow.NewRecord(rb.schema, arrays, nil)
}

func (rb *RecordBuilder) Release() {
	for _, bldr := range rb.fieldBldrs {
		bldr.Release()
	}
}

type int32Bldr struct{ bldr *arrow.Int32Builder }

func (b *int32Bldr) Append(v interface{}) error { return b.bldr.Append(v.(int32)) }
func (b *int32Bldr) NewArray() arrow.Array      { return b.bldr.NewArray() }
func (b *int32Bldr) Release()                   { b.bldr.Release() }

type int64Bldr struct{ bldr *arrow.Int64Builder }

func (b *int64Bldr) Append(v interface{}) error { return b.bldr.Append(v.(int64)) }
func (b *int64Bldr) NewArray() arrow.Array      { return b.bldr.NewArray() }
func (b *int64Bldr) Release()                   { b.bldr.Release() }

type uint32Bldr struct{ bldr *arrow.Uint32Builder }

func (b *uint32Bldr) Append(v interface{}) error { return b.bldr.Append(v.(uint32)) }
func (b *uint32Bldr) NewArray() arrow.Array      { return b.bldr.NewArray() }
func (b *uint32Bldr) Release()                   { b.bldr.Release() }

type float64Bldr struct{ bldr *arrow.Float64Builder }

func (b *float64Bldr) Append(v interface{}) error { return b.bldr.Append(v.(float64)) }
func (b *float64Bldr) NewArray() arrow.Array      { return b.bldr.NewArray() }
func (b *float64Bldr) Release()                   { b.bldr.Release() }

type stringBldr struct{ bldr *arrow.StringBuilder }

func (b *stringBldr) Append(v interface{}) error { return b.bldr.Append(v.(string)) }
func (b *stringBldr) NewArray() arrow.Array      { return b.bldr.NewArray() }
func (b *stringBldr) Release()                   { b.bldr.Release() }

type listBldr struct{ builder *arrow.ListBuilder }

func (b *listBldr) Append(v interface{}) error {
	slice := v.([]interface{})
	b.builder.Append()
	valBldr := b.builder.ValueBuilder()
	switch v := slice[0].(type) {
	case string:
		inner := valBldr.(*arrow.StringBuilder)
		for _, item := range slice {
			inner.Append(item.(string))
		}
	case int:
		inner := valBldr.(*arrow.Int32Builder)
		for _, item := range slice {
			inner.Append(int32(item.(int)))
		}
	case uint32:
		inner := valBldr.(*arrow.Uint32Builder)
		for _, item := range slice {
			inner.Append(item.(uint32))
		}
	case float32:
		inner := valBldr.(*arrow.Float32Builder)
		for _, item := range slice {
			inner.Append(item.(float32))
		}
	}
	return nil
}
func (b *listBldr) NewArray() arrow.Array { return b.builder.NewArray() }
func (b *listBldr) Release()              { b.builder.Release() }
