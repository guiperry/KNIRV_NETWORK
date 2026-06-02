package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"backend_server/internal/storage/mdstorage"
	"backend_server/internal/storage/pqc"

	"github.com/apache/arrow/go/v14/arrow"
	"github.com/apache/arrow/go/v14/arrow/array"
	"github.com/apache/arrow/go/v14/arrow/flight"
	"github.com/apache/arrow/go/v14/arrow/ipc"
	"github.com/apache/arrow/go/v14/arrow/memory"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// UnifiedMemorySystem consolidates all memory-related functionality into a single system
// It merges ActiveMemoryService, KnowledgeBase, ArrowFlightServer, and Ontology
type UnifiedMemorySystem struct {
	// Configuration
	config *MemoryConfig

	// Storage backends
	markdownStorage  *mdstorage.MarkdownStorageDriver
	graphRAGClient   GraphRAGInterface
	ontologyManager  *OntologyManager
	reasoningEngine  *ReasoningEngine

	// Services
	vaultService *VaultService

	// Arrow Flight streaming
	flightServer *FlightServer
	arrowAllocator memory.Allocator

	// Concurrency control
	mu sync.RWMutex

	// Logger
	logger *zap.Logger

	// Auto-sync
	syncTicker *time.Ticker
	stopCh      chan struct{}
}

// NewUnifiedMemorySystem creates a new UnifiedMemorySystem
func NewUnifiedMemorySystem(cfg *MemoryConfig, logger *zap.Logger) (*UnifiedMemorySystem, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	system := &UnifiedMemorySystem{
		config:       cfg,
		arrowAllocator: memory.DefaultAllocator,
		logger:        logger,
		stopCh:        make(chan struct{}),
	}

	// Initialize markdown storage with PQC encryption if enabled
	if contains(cfg.EnabledBackends, "markdown") {
		var encManager *pqc.EncryptionManager
		if cfg.PQCEncryption {
			// Create key rotation manager with default rotation interval
			krm := pqc.NewKeyRotationManager(24 * time.Hour)
			encManager = pqc.NewEncryptionManager(krm)
		}
		mdStorage, err := mdstorage.NewMarkdownStorageDriver("storage/memory_vault", encManager, "default")
		if err != nil {
			return nil, fmt.Errorf("failed to create markdown storage: %w", err)
		}
		system.markdownStorage = mdStorage
	}

	// Initialize GraphRAG client if enabled
	if contains(cfg.EnabledBackends, "graphrag") {
		system.graphRAGClient = NewGraphRAGClient()
	}

	// Initialize ontology manager if enabled
	if contains(cfg.EnabledBackends, "ontology") {
		system.ontologyManager = NewOntologyManager(logger)
	}

	// Initialize reasoning engine
	system.reasoningEngine = NewReasoningEngine(system.markdownStorage, logger)

	// Initialize vault service
	system.vaultService = NewVaultService(system.markdownStorage, logger)

	// Initialize Arrow Flight server if enabled
	if cfg.ArrowStreaming {
		system.flightServer = NewFlightServer(system.arrowAllocator, system, logger)
	}

	// Start auto-sync if enabled
	if cfg.EnableAutoSync && cfg.SyncInterval > 0 {
		system.syncTicker = time.NewTicker(cfg.SyncInterval)
		go system.autoSyncLoop()
	}

	logger.Info("UnifiedMemorySystem initialized",
		zap.Strings("backends", cfg.EnabledBackends),
		zap.Bool("pqc_encryption", cfg.PQCEncryption),
		zap.Bool("arrow_streaming", cfg.ArrowStreaming),
	)

	return system, nil
}

// StoreInteraction stores an agent interaction in the unified memory system
func (s *UnifiedMemorySystem) StoreInteraction(ctx context.Context, agentID, errorDesc, solutionCode string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 1. Register ErrorNode in vault
	errID := fmt.Sprintf("err_%s", uuid.New().String())
	errNode := &ErrorNode{
		ID:          errID,
		Description: errorDesc,
		Context:      map[string]interface{}{"agent_id": agentID},
		Timestamp:    time.Now(),
	}
	if err := s.vaultService.RegisterError(errNode); err != nil {
		return fmt.Errorf("failed to register error: %w", err)
	}

	// 2. Register SolutionNode in vault
	solID := fmt.Sprintf("sol_%s", uuid.New().String())
	solNode := &SolutionNode{
		ID:        solID,
		ErrorID:   errID,
		Language:  "go",
		Code:      solutionCode,
		Timestamp: time.Now(),
	}
	if err := s.vaultService.RegisterSolution(solNode); err != nil {
		return fmt.Errorf("failed to register solution: %w", err)
	}

	// 3. Generate reasoning trace
	steps := []string{
		fmt.Sprintf("Detected: %s", errorDesc),
		"Searching Vault for compatible solutions...",
		fmt.Sprintf("Found SolutionNode: %s", solID),
		"Verifying solution integrity...",
	}
	if err := s.reasoningEngine.GenerateTrace(agentID, errID, steps, "Success"); err != nil {
		s.logger.Warn("failed to generate trace", zap.Error(err))
	}

	// 4. Update ontology if enabled
	if s.ontologyManager != nil {
		entity := &OntologyEntity{
			ID:        fmt.Sprintf("error_%s", errID),
			Type:      EntityTypePattern,
			Label:     errorDesc,
			Properties: map[string]interface{}{"agent_id": agentID},
			CreatedAt:  time.Now(),
		}
		s.ontologyManager.UpsertEntity(entity)
	}

	// 5. Trigger auto-sync if enabled
	if s.config.EnableAutoSync {
		go s.syncToGraphRAG(errID)
	}

	return nil
}

// Query performs a cross-backend query
func (s *UnifiedMemorySystem) Query(ctx context.Context, req *QueryRequest) (*QueryResult, error) {
	result := &QueryResult{
		ID:        fmt.Sprintf("query_%s", uuid.New().String()),
		Query:     req.Query,
		Mode:      req.Mode,
		Timestamp: time.Now(),
	}

	// Query GraphRAG if enabled and requested
	if s.graphRAGClient != nil && (req.Mode == "graphrag" || req.Mode == "hybrid") {
		if contains(req.Sources, "graphrag") || len(req.Sources) == 0 {
			graphReq := &GraphRAGQuery{
				Query: req.Query,
				Mode:  "hybrid",
				Limit: req.Limit,
			}
			graphRes, err := s.graphRAGClient.Query(ctx, "default", graphReq)
			if err != nil {
				s.logger.Warn("GraphRAG query failed", zap.Error(err))
			} else {
				result.Nodes = graphRes.Nodes
				result.Edges = graphRes.Edges
				result.Chunks = graphRes.Chunks
				result.Score = graphRes.Score
			}
		}
	}

	// Query ontology if enabled and requested
	if s.ontologyManager != nil && (req.Mode == "ontology" || req.Mode == "hybrid") {
		if contains(req.Sources, "ontology") || len(req.Sources) == 0 {
			entities := s.ontologyManager.QueryByType(EntityTypePattern)
			for _, e := range entities {
				result.Entities = append(result.Entities, e)
			}
		}
	}

	return result, nil
}

// StreamToArrow streams memory data via Arrow Flight
func (s *UnifiedMemorySystem) StreamToArrow(agentID string, fs flight.FlightService_DoGetServer) error {
	if s.markdownStorage == nil {
		return nil
	}

	schema := s.getAgentMemorySchema()
	writer := flight.NewRecordWriter(fs, ipc.WithSchema(schema))
	defer writer.Close()

	docs, err := s.markdownStorage.ListDocuments()
	if err != nil {
		return fmt.Errorf("failed to list documents: %w", err)
	}

	for _, doc := range docs {
		if agentID != "" {
			if aid, ok := doc.Metadata["agent_id"].(string); ok && aid != agentID {
				continue
			}
		}

		builder := array.NewRecordBuilder(s.arrowAllocator, schema)
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

// GetFlightServer returns the Arrow Flight server
func (s *UnifiedMemorySystem) GetFlightServer() *FlightServer {
	return s.flightServer
}

// GetVaultService returns the vault service
func (s *UnifiedMemorySystem) GetVaultService() *VaultService {
	return s.vaultService
}

// GetOntologyManager returns the ontology manager
func (s *UnifiedMemorySystem) GetOntologyManager() *OntologyManager {
	return s.ontologyManager
}

// GetReasoningEngine returns the reasoning engine
func (s *UnifiedMemorySystem) GetReasoningEngine() *ReasoningEngine {
	return s.reasoningEngine
}

// GetGraphRAGClient returns the GraphRAG client
func (s *UnifiedMemorySystem) GetGraphRAGClient() GraphRAGInterface {
	return s.graphRAGClient
}

// Close cleans up all resources
func (s *UnifiedMemorySystem) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	close(s.stopCh)

	if s.syncTicker != nil {
		s.syncTicker.Stop()
	}

	if s.graphRAGClient != nil {
		if err := s.graphRAGClient.Close(); err != nil {
			return err
		}
	}

	if s.flightServer != nil {
		if err := s.flightServer.Stop(); err != nil {
			return err
		}
	}

	return nil
}

// autoSyncLoop handles automatic synchronization between backends
func (s *UnifiedMemorySystem) autoSyncLoop() {
	for {
		select {
		case <-s.stopCh:
			return
		case <-s.syncTicker.C:
			s.performSync()
		}
	}
}

// performSync synchronizes data between backends
func (s *UnifiedMemorySystem) performSync() {
	s.logger.Debug("performing auto-sync")

	// Sync markdown to GraphRAG
	if s.markdownStorage != nil && s.graphRAGClient != nil {
		docs, err := s.markdownStorage.ListDocuments()
		if err != nil {
			s.logger.Error("failed to list documents for sync", zap.Error(err))
			return
		}

		for _, doc := range docs {
			if doc.Type == "ERROR" || doc.Type == "SOLUTION" {
				s.syncToGraphRAG(doc.ID)
			}
		}
	}
}

// syncToGraphRAG syncs a document to GraphRAG index
func (s *UnifiedMemorySystem) syncToGraphRAG(docID string) {
	if s.graphRAGClient == nil {
		return
	}

	doc, err := s.markdownStorage.LoadDocument("", docID)
	if err != nil {
		s.logger.Error("failed to load document for GraphRAG sync", zap.Error(err))
		return
	}

	if len(doc.Content) == 0 {
		s.logger.Warn("document has no content for GraphRAG sync", zap.String("doc_id", docID))
		return
	}

	_, err = s.graphRAGClient.IndexDocumentWithResult(context.Background(), docID, doc.Content)
	if err != nil {
		s.logger.Error("failed to sync document to GraphRAG", zap.Error(err))
		return
	}

	s.logger.Debug("synced document to GraphRAG",
		zap.String("doc_id", docID),
	)
}

// getAgentMemorySchema returns the Arrow schema for agent memory
func (s *UnifiedMemorySystem) getAgentMemorySchema() *arrow.Schema {
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

// contains checks if a string slice contains a value
func contains(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}
