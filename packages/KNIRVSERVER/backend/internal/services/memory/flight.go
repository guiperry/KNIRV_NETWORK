package memory

import (
	"fmt"
	"log"

	"github.com/apache/arrow/go/v14/arrow"
	"github.com/apache/arrow/go/v14/arrow/array"
	"github.com/apache/arrow/go/v14/arrow/flight"
	"github.com/apache/arrow/go/v14/arrow/ipc"
	"github.com/apache/arrow/go/v14/arrow/memory"
	"go.uber.org/zap"
)

// FlightServer wraps the Arrow Flight server for memory streaming
type FlightServer struct {
	flight.BaseFlightServer
	allocator     memory.Allocator
	memorySystem  *UnifiedMemorySystem
	logger        *zap.Logger
	server        flight.Server
}

// NewFlightServer creates a new FlightServer
func NewFlightServer(allocator memory.Allocator, memSystem *UnifiedMemorySystem, logger *zap.Logger) *FlightServer {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &FlightServer{
		allocator:    allocator,
		memorySystem: memSystem,
		logger:        logger,
	}
}

// Start starts the Flight server on the given address
func (s *FlightServer) Start(addr string) error {
	s.server = flight.NewFlightServer()
	s.server.RegisterFlightService(s)

	if err := s.server.Init(addr); err != nil {
		return fmt.Errorf("failed to init flight server: %w", err)
	}

	s.logger.Info("starting Flight server", zap.String("addr", addr))

	go func() {
		if err := s.server.Serve(); err != nil {
			log.Printf("Flight server error: %v", err)
		}
	}()

	return nil
}

// Stop stops the Flight server
func (s *FlightServer) Stop() error {
	if s.server != nil {
		s.server.Shutdown()
	}
	return nil
}

// DoGet streams memory data to the client
func (s *FlightServer) DoGet(tkt *flight.Ticket, fs flight.FlightService_DoGetServer) error {
	agentID := string(tkt.Ticket)
	s.logger.Info("streaming memory for agent", zap.String("agent_id", agentID))

	if s.memorySystem != nil {
		return s.memorySystem.StreamToArrow(agentID, fs)
	}

	return nil
}

// getAgentMemorySchema returns the Arrow schema for agent memory
func (s *FlightServer) getAgentMemorySchema() *arrow.Schema {
	return arrow.NewSchema([]arrow.Field{
		{Name: "timestamp", Type: arrow.PrimitiveTypes.Int64},
		{Name: "agent_id", Type: arrow.BinaryTypes.String},
		{Name: "intent", Type: arrow.BinaryTypes.String},
		{Name: "observed_action", Type: arrow.BinaryTypes.String},
		{Name: "token_usage", Type: arrow.PrimitiveTypes.Int32},
		{Name: "relevance", Type: arrow.PrimitiveTypes.Float64},
		{Name: "verified", Type: arrow.FixedWidthTypes.Boolean},
	}, nil)
}

// streamRecords streams records from markdown storage using Arrow
func (s *FlightServer) streamRecords(agentID string, fs flight.FlightService_DoGetServer) error {
	if s.memorySystem == nil || s.memorySystem.markdownStorage == nil {
		return nil
	}

	schema := s.getAgentMemorySchema()
	writer := flight.NewRecordWriter(fs, ipc.WithSchema(schema))
	defer writer.Close()

	docs, err := s.memorySystem.markdownStorage.ListDocuments()
	if err != nil {
		return fmt.Errorf("failed to list documents: %w", err)
	}

	for _, doc := range docs {
		if agentID != "" {
			if aid, ok := doc.Metadata["agent_id"].(string); ok && aid != agentID {
				continue
			}
		}

		builder := array.NewRecordBuilder(s.allocator, schema)
		defer builder.Release()

		intent := ""
		if v, ok := doc.Metadata["intent"].(string); ok {
			intent = v
		}
		observed := string(doc.Content)
		if len(observed) > 256 {
			observed = observed[:256]
		}
		tokenUsage := int32(0)
		if v, ok := doc.Metadata["token_usage"].(float64); ok {
			tokenUsage = int32(v)
		}
		relevance := 1.0
		if v, ok := doc.Metadata["relevance"].(float64); ok {
			relevance = v
		}
		verified := true
		if v, ok := doc.Metadata["verified"].(bool); ok {
			verified = v
		}

		builder.Field(0).(*array.Int64Builder).Append(doc.Timestamp.Unix())
		builder.Field(1).(*array.StringBuilder).Append(agentID)
		builder.Field(2).(*array.StringBuilder).Append(intent)
		builder.Field(3).(*array.StringBuilder).Append(observed)
		builder.Field(4).(*array.Int32Builder).Append(tokenUsage)
		builder.Field(5).(*array.Float64Builder).Append(relevance)
		builder.Field(6).(*array.BooleanBuilder).Append(verified)

		record := builder.NewRecord()
		if err := writer.Write(record); err != nil {
			record.Release()
			return err
		}
		record.Release()
	}

	return nil
}
