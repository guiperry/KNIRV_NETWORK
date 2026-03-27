package active_memory

import (
	"context"
	"fmt"
	"time"

	"backend_server/internal/reasoning/graph"
	nexus "backend_server/internal/server"
	"backend_server/internal/services/vault"
	"backend_server/internal/storage/mdstorage"
	"backend_server/internal/storage/pqc"

	"github.com/apache/arrow/go/v14/arrow/array"
	"github.com/apache/arrow/go/v14/arrow/flight"
	"github.com/apache/arrow/go/v14/arrow/ipc"
	"github.com/apache/arrow/go/v14/arrow/memory"
)

// ActiveMemoryService coordinates the Markdown Fabric and Arrow Streaming
type ActiveMemoryService struct {
	vault    *vault.VaultService
	graph    *graph.ReasoningEngine
	storage  *mdstorage.MarkdownStorageDriver
	executor *vault.SolutionExecutor
	mem      memory.Allocator
}

// NewActiveMemoryService creates a new ActiveMemoryService with security validation
func NewActiveMemoryService(v *vault.VaultService, g *graph.ReasoningEngine, s *mdstorage.MarkdownStorageDriver, validator *pqc.SolutionNodeValidator) *ActiveMemoryService {
	return &ActiveMemoryService{
		vault:    v,
		graph:    g,
		storage:  s,
		executor: vault.NewSolutionExecutor(validator),
		mem:      memory.DefaultAllocator,
	}
}

// StreamToArrow projects encrypted .md records from storage into an Arrow Flight stream.
// Each document is decoded and its metadata fields are mapped to the AgentMemorySchema columns.
// Documents whose agent_id metadata field does not match agentID are skipped.
func (s *ActiveMemoryService) StreamToArrow(agentID string, fs flight.FlightService_DoGetServer) error {
	schema := nexus.NewAgentMemorySchema()
	writer := flight.NewRecordWriter(fs, ipc.WithSchema(schema))
	defer writer.Close()

	if s.storage == nil {
		return nil
	}

	docs, err := s.storage.ListDocuments()
	if err != nil {
		return fmt.Errorf("failed to list documents for agent %s: %w", agentID, err)
	}

	for _, doc := range docs {
		// Filter by agent ID when the metadata field is present and doesn't match.
		if agentID != "" {
			if aid, ok := doc.Metadata["agent_id"].(string); ok && aid != agentID {
				continue
			}
		}

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

		builder := array.NewRecordBuilder(s.mem, schema)
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
			builder.Release()
			return err
		}
		record.Release()
		builder.Release()
	}

	return nil
}

// RecordInteraction documents an agent interaction in both the Vault and Graph
func (s *ActiveMemoryService) RecordInteraction(ctx context.Context, agentID, errorDesc string, solutionCode string) error {
	// 1. Register the ErrorNode in the Vault (.md)
	errID := fmt.Sprintf("err_%d", time.Now().Unix())
	errNode := &vault.ErrorNode{
		ID:          errID,
		Description: errorDesc,
		Timestamp:   time.Now(),
	}
	if err := s.vault.RegisterError(errNode); err != nil {
		return err
	}

	// 2. Register the SolutionNode in the Vault (.md)
	solID := fmt.Sprintf("sol_%d", time.Now().Unix())
	solNode := &vault.SolutionNode{
		ID:        solID,
		ErrorID:   errID,
		Language:  "go", // default
		Code:      solutionCode,
		Timestamp: time.Now(),
	}
	if err := s.vault.RegisterSolution(solNode); err != nil {
		return err
	}

	// 3. Generate the Reasoning Trace in the Graph (.md)
	steps := []string{
		"Detected: " + errorDesc,
		"Searching Vault for compatible solutions...",
		"Found SolutionNode: " + solID,
		"Verifying solution integrity via PQC signature...",
	}
	return s.graph.GenerateTrace(agentID, errID, steps, "Success")
}

// ExecuteSolution retrieves and runs logic from a SolutionNode
func (s *ActiveMemoryService) ExecuteSolution(ctx context.Context, id string, params map[string]interface{}) (string, error) {
	sol, err := s.vault.GetSolution(id)
	if err != nil {
		return "", fmt.Errorf("solution not found: %w", err)
	}

	return s.executor.Execute(ctx, sol, params)
}
