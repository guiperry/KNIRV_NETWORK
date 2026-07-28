package main

import (
	"fmt"
	"strings"

	"github.com/apache/arrow/go/v15/arrow"
	"github.com/apache/arrow/go/v15/arrow/array"
	"github.com/apache/arrow/go/v15/arrow/flight"
	"github.com/apache/arrow/go/v15/arrow/ipc"
	"github.com/apache/arrow/go/v15/arrow/memory"
	"github.com/knirvcorp/knirvbase/pkg/knirvbase"
	"github.com/knirvcorp/knirvbase/pkg/nrv"
)

type flightAdapter struct {
	flight.BaseFlightServer
	db   *knirvbase.DB
	pool memory.Allocator
}

func newFlightAdapter(db *knirvbase.DB) *flightAdapter {
	return &flightAdapter{db: db, pool: memory.NewGoAllocator()}
}
func (s *flightAdapter) DoGet(ticket *flight.Ticket, stream flight.FlightService_DoGetServer) error {
	if ticket == nil {
		return fmt.Errorf("missing flight ticket")
	}
	parts := strings.Split(string(ticket.Ticket), ".")
	if len(parts) != 2 || parts[1] == "" {
		return fmt.Errorf("ticket must be <gold|research>.<domain")
	}
	ch, err := s.db.Dataset(parts[1]).StreamBrackets(stream.Context(), parts[0] == "gold")
	if err != nil {
		return err
	}
	schema := flightSchema()
	writer := flight.NewRecordWriter(stream, ipc.WithSchema(schema))
	defer writer.Close()
	builder := array.NewRecordBuilder(s.pool, schema)
	defer builder.Release()
	count := 0
	flush := func() error {
		if count == 0 {
			return nil
		}
		rec := builder.NewRecord()
		defer rec.Release()
		if err := writer.Write(rec); err != nil {
			return err
		}
		builder.Release()
		builder = array.NewRecordBuilder(s.pool, schema)
		count = 0
		return nil
	}
	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case b, ok := <-ch:
			if !ok {
				return flush()
			}
			raw := nrv.EncodeBracket(b)
			builder.Field(0).(*array.StringBuilder).Append(b.ID)
			builder.Field(1).(*array.StringBuilder).Append(b.FrameID)
			builder.Field(2).(*array.Int64Builder).Append(int64(b.SubSecondUS))
			builder.Field(3).(*array.FixedSizeBinaryBuilder).Append(raw[:])
			builder.Field(4).(*array.Float64Builder).Append(0)
			builder.Field(5).(*array.StringBuilder).Append("I")
			count++
			if count >= 1024 {
				if err := flush(); err != nil {
					return err
				}
			}
		}
	}
}
func flightSchema() *arrow.Schema {
	return arrow.NewSchema([]arrow.Field{{Name: "bracket_id", Type: arrow.BinaryTypes.String}, {Name: "frame_id", Type: arrow.BinaryTypes.String}, {Name: "frame_timestamp_unix", Type: arrow.PrimitiveTypes.Int64}, {Name: "payload_asic", Type: &arrow.FixedSizeBinaryType{ByteWidth: nrv.BracketSize}}, {Name: "drift_score", Type: arrow.PrimitiveTypes.Float64}, {Name: "bracket_type", Type: arrow.BinaryTypes.String}}, nil)
}
