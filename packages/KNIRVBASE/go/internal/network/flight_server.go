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
	projType := &arrow.FixedSizeBinaryType{ByteWidth: 64}
	return arrow.NewSchema([]arrow.Field{
		{Name: "frame_id", Type: arrow.BinaryTypes.String},
		{Name: "lsh_salt", Type: arrow.PrimitiveTypes.Uint32},
		{Name: "subsecond_us", Type: arrow.PrimitiveTypes.Uint32},
		{Name: "asic_loops", Type: arrow.PrimitiveTypes.Uint32},
		{Name: "golden_seed", Type: arrow.PrimitiveTypes.Uint32},
		{Name: "drift_score", Type: arrow.PrimitiveTypes.Float64},
		{Name: "bracket_type", Type: arrow.BinaryTypes.String},
		{Name: "projections", Type: projType},
		{Name: "frame_timestamp", Type: arrow.PrimitiveTypes.Int64},
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

func bracketFrameTimestamp(bracket *nrv.Bracket) int64 {
	if bracket == nil {
		return 0
	}
	return int64(bracket.SubSecondUS)
}

func bracketDriftScore(bracket *nrv.Bracket) float64 {
	if bracket == nil || bracket.Meta == nil {
		return 0
	}
	return bracket.Meta.DriftScore
}

func bracketType(bracket *nrv.Bracket) string {
	if bracket == nil || bracket.Meta == nil || bracket.Meta.Type == "" {
		return string(nrv.DeltaTypeI)
	}
	return string(bracket.Meta.Type)
}

func appendBracketRecord(recordBuilder *array.RecordBuilder, bracket *nrv.Bracket) {
	recordBuilder.Field(0).(*array.StringBuilder).Append(bracket.ID)
	recordBuilder.Field(1).(*array.Uint32Builder).Append(bracket.SubSecondUS)
	recordBuilder.Field(2).(*array.Uint32Builder).Append(bracket.GoldenSeed)
	recordBuilder.Field(3).(*array.Float64Builder).Append(bracketDriftScore(bracket))
	recordBuilder.Field(4).(*array.StringBuilder).Append(bracketType(bracket))

	projBytes := make([]byte, 32)
	copy(projBytes, bracket.Projections[:])
	recordBuilder.Field(5).(*array.FixedSizeBinaryBuilder).Append(projBytes)
	recordBuilder.Field(6).(*array.Int64Builder).Append(bracketFrameTimestamp(bracket))
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

		frameIDCol := record.Column(0).(*array.String)
		subsecCol := record.Column(1).(*array.Uint32)
		goldenCol := record.Column(2).(*array.Uint32)
		driftCol := record.Column(3).(*array.Float64)
		typeCol := record.Column(4).(*array.String)
		projCol := record.Column(5).(*array.FixedSizeBinary)

		for i := 0; i < count; i++ {
			b := &nrv.Bracket{
				ID:          frameIDCol.Value(i),
				SubSecondUS: subsecCol.Value(i),
				GoldenSeed:  goldenCol.Value(i),
				Meta: &nrv.BracketMeta{
					ID:         frameIDCol.Value(i),
					Type:       nrv.DeltaType(typeCol.Value(i)),
					DriftScore: driftCol.Value(i),
				},
			}

			b.Projections = [32]byte{}
			projBytes := projCol.Value(i)
			copy(b.Projections[:], projBytes)

			brackets = append(brackets, b)
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
	reader, err := ipc.NewReader(c.conn)
	if err != nil {
		return nil, err
	}

	var brackets []*nrv.Bracket

	for reader.Next() {
		record := reader.Record()
		count := int(record.NumRows())

		frameIDCol := record.Column(0).(*array.String)
		subsecCol := record.Column(1).(*array.Uint32)
		goldenCol := record.Column(2).(*array.Uint32)
		driftCol := record.Column(3).(*array.Float64)
		typeCol := record.Column(4).(*array.String)
		projCol := record.Column(5).(*array.FixedSizeBinary)

		for i := 0; i < count; i++ {
			b := &nrv.Bracket{
				ID:          frameIDCol.Value(i),
				SubSecondUS: subsecCol.Value(i),
				GoldenSeed:  goldenCol.Value(i),
				Meta: &nrv.BracketMeta{
					ID:         frameIDCol.Value(i),
					Type:       nrv.DeltaType(typeCol.Value(i)),
					DriftScore: driftCol.Value(i),
				},
			}

			b.Projections = [32]byte{}
			projBytes := projCol.Value(i)
			copy(b.Projections[:], projBytes)

			brackets = append(brackets, b)
		}
		record.Release()
	}

	return brackets, nil
}

func euclideanDistanceFloat32(a, b [16]float32) float64 {
	var sum float64
	for i := 0; i < 16; i++ {
		diff := float64(a[i] - b[i])
		sum += diff * diff
	}
	return sum
}
