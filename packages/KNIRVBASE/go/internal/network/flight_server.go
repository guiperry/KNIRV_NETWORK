package network

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/apache/arrow/go/v15/arrow"
	"github.com/apache/arrow/go/v15/arrow/array"
	"github.com/apache/arrow/go/v15/arrow/ipc"
	"github.com/apache/arrow/go/v15/arrow/memory"

	stor "github.com/knirvcorp/knirvbase/go/internal/storage"
	"github.com/knirvcorp/knirvbase/go/pkg/nrv"
)

type FlightServer struct {
	storage *stor.NRVStorage
	memPool memory.Allocator
	schema  *arrow.Schema
}

func NewFlightServer(storage *stor.NRVStorage) *FlightServer {
	fs := &FlightServer{
		storage: storage,
		memPool: memory.NewGoAllocator(),
	}
	fs.schema = fs.bracketArrowSchema()
	return fs
}

func (s *FlightServer) bracketArrowSchema() *arrow.Schema {
	asicType := &arrow.FixedSizeBinaryType{ByteWidth: nrv.BracketSize}
	return arrow.NewSchema([]arrow.Field{
		{Name: "bracket_id", Type: arrow.BinaryTypes.String},
		{Name: "frame_id", Type: arrow.BinaryTypes.String},
		{Name: "frame_timestamp_unix", Type: arrow.PrimitiveTypes.Int64},
		{Name: "payload_asic", Type: asicType},
		{Name: "drift_score", Type: arrow.PrimitiveTypes.Float64},
		{Name: "bracket_type", Type: arrow.BinaryTypes.String},
	}, nil)
}

func (s *FlightServer) Schema() *arrow.Schema {
	return s.schema
}

func (s *FlightServer) parseTicket(ticket string) (collection string, goldOnly bool, err error) {
	parts := splitTicket(ticket)
	if len(parts) != 2 {
		return "", false, fmt.Errorf("flight: invalid ticket format, expected <stream>.<collection>")
	}
	streamType, collection := parts[0], parts[1]
	goldOnly = streamType == "gold"
	return collection, goldOnly, nil
}

func splitTicket(ticket string) []string {
	var parts []string
	var current []byte
	for i := 0; i < len(ticket); i++ {
		if ticket[i] == '.' {
			parts = append(parts, string(current))
			current = nil
		} else {
			current = append(current, ticket[i])
		}
	}
	if len(current) > 0 {
		parts = append(parts, string(current))
	}
	return parts
}

type BracketStreamServer interface {
	Send(*BracketBatch) error
	Context() context.Context
}

type BracketBatch struct {
	Data []byte
}

func (s *FlightServer) StreamBrackets(ticket string, server BracketStreamServer) error {
	collection, goldOnly, err := s.parseTicket(ticket)
	if err != nil {
		return err
	}

	bracketCh, err := s.storage.StreamBrackets(server.Context(), collection, goldOnly)
	if err != nil {
		return err
	}

	batchSize := 1024
	return s.streamBatches(server, bracketCh, batchSize)
}

func bracketFrameUnix(b *nrv.Bracket) int64 {
	if b == nil {
		return 0
	}
	if b.FrameUnix != 0 {
		return b.FrameUnix
	}
	return int64(b.SubSecondUS)
}

func bracketDriftScore(b *nrv.Bracket) float64 {
	if b == nil || b.Meta == nil {
		return 0
	}
	return b.Meta.DriftScore
}

func bracketTypeStr(b *nrv.Bracket) string {
	if b == nil || b.Meta == nil || b.Meta.Type == "" {
		return string(nrv.DeltaTypeI)
	}
	return string(b.Meta.Type)
}

func appendBracketRecord(recordBuilder *array.RecordBuilder, bracket *nrv.Bracket) {
	wire := nrv.EncodeBracket(bracket)
	recordBuilder.Field(0).(*array.StringBuilder).Append(bracket.ID)
	recordBuilder.Field(1).(*array.StringBuilder).Append(bracket.FrameID)
	recordBuilder.Field(2).(*array.Int64Builder).Append(bracketFrameUnix(bracket))
	recordBuilder.Field(3).(*array.FixedSizeBinaryBuilder).Append(wire[:])
	recordBuilder.Field(4).(*array.Float64Builder).Append(bracketDriftScore(bracket))
	recordBuilder.Field(5).(*array.StringBuilder).Append(bracketTypeStr(bracket))
}

func (s *FlightServer) streamBatches(server BracketStreamServer, bracketCh <-chan *nrv.Bracket, batchSize int) error {
	recordBuilder := array.NewRecordBuilder(s.memPool, s.schema)
	defer func() {
		if recordBuilder != nil {
			recordBuilder.Release()
		}
	}()

	flushBatch := func() error {
		record := recordBuilder.NewRecord()
		if record == nil {
			return nil
		}
		defer record.Release()

		var buf bytes.Buffer
		writer := ipc.NewWriter(&buf, ipc.WithSchema(s.schema))
		if err := writer.Write(record); err != nil {
			writer.Close()
			return err
		}
		writer.Close()

		return server.Send(&BracketBatch{Data: buf.Bytes()})
	}

	var batchCount int

	for {
		select {
		case <-server.Context().Done():
			return server.Context().Err()
		case bracket, ok := <-bracketCh:
			if !ok {
				if batchCount > 0 {
					return flushBatch()
				}
				return nil
			}

			appendBracketRecord(recordBuilder, bracket)
			batchCount++

			if batchCount >= batchSize {
				if err := flushBatch(); err != nil {
					return err
				}
				recordBuilder.Release()
				recordBuilder = array.NewRecordBuilder(s.memPool, s.schema)
				batchCount = 0
			}
		}
	}
}

func BracketsToFlightData(brackets []*nrv.Bracket) ([]byte, error) {
	if len(brackets) == 0 {
		return nil, nil
	}

	memPool := memory.NewGoAllocator()
	server := &FlightServer{memPool: memPool}
	schema := server.bracketArrowSchema()

	recordBuilder := array.NewRecordBuilder(memPool, schema)
	defer recordBuilder.Release()

	for _, b := range brackets {
		appendBracketRecord(recordBuilder, b)
	}

	record := recordBuilder.NewRecord()
	if record == nil {
		return nil, nil
	}
	defer record.Release()

	var buf bytes.Buffer
	writer := ipc.NewWriter(&buf, ipc.WithSchema(schema))
	if err := writer.Write(record); err != nil {
		writer.Close()
		return nil, err
	}
	writer.Close()

	return buf.Bytes(), nil
}

func FlightDataToBrackets(data []byte) ([]*nrv.Bracket, error) {
	if len(data) == 0 {
		return nil, nil
	}

	reader, err := ipc.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	var brackets []*nrv.Bracket

	for reader.Next() {
		record := reader.Record()
		count := int(record.NumRows())

		idCol := record.Column(0).(*array.String)
		frameIDCol := record.Column(1).(*array.String)
		frameUnixCol := record.Column(2).(*array.Int64)
		asicCol := record.Column(3).(*array.FixedSizeBinary)
		driftCol := record.Column(4).(*array.Float64)
		typeCol := record.Column(5).(*array.String)

		for i := 0; i < count; i++ {
			var raw [nrv.BracketSize]byte
			copy(raw[:], asicCol.Value(i))
			dec := nrv.DecodeBracket(raw)
			bp := new(nrv.Bracket)
			*bp = dec
			bp.ID = idCol.Value(i)
			bp.FrameID = frameIDCol.Value(i)
			bp.FrameUnix = frameUnixCol.Value(i)
			bp.Meta = &nrv.BracketMeta{
				ID:         idCol.Value(i),
				Type:       nrv.DeltaType(typeCol.Value(i)),
				DriftScore: driftCol.Value(i),
			}
			brackets = append(brackets, bp)
		}
		record.Release()
	}

	return brackets, nil
}

type FlightClient struct {
	conn    io.Reader
	memPool memory.Allocator
}

func NewFlightClient(conn io.Reader) *FlightClient {
	return &FlightClient{
		conn:    conn,
		memPool: memory.NewGoAllocator(),
	}
}

func (c *FlightClient) StreamBrackets(ctx context.Context, ticket string) ([]*nrv.Bracket, error) {
	_, _ = ctx, ticket
	reader, err := ipc.NewReader(c.conn)
	if err != nil {
		return nil, err
	}

	var brackets []*nrv.Bracket

	for reader.Next() {
		record := reader.Record()
		count := int(record.NumRows())

		idCol := record.Column(0).(*array.String)
		frameIDCol := record.Column(1).(*array.String)
		frameUnixCol := record.Column(2).(*array.Int64)
		asicCol := record.Column(3).(*array.FixedSizeBinary)
		driftCol := record.Column(4).(*array.Float64)
		typeCol := record.Column(5).(*array.String)

		for i := 0; i < count; i++ {
			var raw [nrv.BracketSize]byte
			copy(raw[:], asicCol.Value(i))
			dec := nrv.DecodeBracket(raw)
			bp := new(nrv.Bracket)
			*bp = dec
			bp.ID = idCol.Value(i)
			bp.FrameID = frameIDCol.Value(i)
			bp.FrameUnix = frameUnixCol.Value(i)
			bp.Meta = &nrv.BracketMeta{
				ID:         idCol.Value(i),
				Type:       nrv.DeltaType(typeCol.Value(i)),
				DriftScore: driftCol.Value(i),
			}
			brackets = append(brackets, bp)
		}
		record.Release()
	}

	return brackets, nil
}
