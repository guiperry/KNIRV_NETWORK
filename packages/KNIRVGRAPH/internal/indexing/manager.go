package indexing

import (
	"KNIRVGRAPH/internal/embeddings"
	"KNIRVGRAPH/internal/processing"
	"KNIRVGRAPH/internal/retrieval"
	"KNIRVGRAPH/internal/storage"
	"KNIRVGRAPH/internal/types"
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

type DocumentRecord struct {
	ID          string                 `json:"id"`
	Status      types.DocumentStatus   `json:"status"`
	ChunkCount  int                    `json:"chunk_count"`
	EntityCount int                    `json:"entity_count"`
	Error       string                 `json:"error,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type IndexManager struct {
	storage     storage.GraphStorage
	chunker     *processing.Chunker
	extractor   *processing.Extractor
	embedding   *embeddings.EmbeddingService
	pipeline    *retrieval.RetrievalPipeline
	logger      *zap.Logger
	mu          sync.RWMutex
	documents   map[string]*DocumentRecord
	config      types.ChunkingConfig
	extConfig   types.ExtractionConfig
}

func NewIndexManager(
	store storage.GraphStorage,
	chunker *processing.Chunker,
	extractor *processing.Extractor,
	embedding *embeddings.EmbeddingService,
	pipeline *retrieval.RetrievalPipeline,
	logger *zap.Logger,
	chunkConfig types.ChunkingConfig,
	extConfig types.ExtractionConfig,
) *IndexManager {
	return &IndexManager{
		storage:   store,
		chunker:   chunker,
		extractor: extractor,
		embedding: embedding,
		pipeline:  pipeline,
		logger:    logger,
		documents: make(map[string]*DocumentRecord),
		config:    chunkConfig,
		extConfig: extConfig,
	}
}

func (m *IndexManager) IndexDocument(ctx context.Context, doc types.ProcessedDocument) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if rec, exists := m.documents[doc.ID]; exists && rec.Status == types.DocumentStatusProcessing {
		return fmt.Errorf("document %s is already being processed", doc.ID)
	}
	record := &DocumentRecord{
		ID:       doc.ID,
		Status:   types.DocumentStatusProcessing,
		Metadata: doc.Metadata,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.documents[doc.ID] = record
	go func() {
		err := m.processDocument(ctx, doc)
		m.mu.Lock()
		defer m.mu.Unlock()
		rec := m.documents[doc.ID]
		if err != nil {
			rec.Status = types.DocumentStatusFailed
			rec.Error = err.Error()
		} else {
			rec.Status = types.DocumentStatusIndexed
			rec.ChunkCount = len(doc.Chunks)
			rec.EntityCount = len(doc.Entities)
		}
		rec.UpdatedAt = time.Now()
	}()
	return nil
}

func (m *IndexManager) processDocument(ctx context.Context, doc types.ProcessedDocument) error {
	if len(doc.Chunks) == 0 && m.chunker != nil {
		chunks, err := m.chunker.Chunk(doc.ID, doc.Content)
		if err != nil {
			return fmt.Errorf("chunking failed: %w", err)
		}
		doc.Chunks = chunks
	}
	if m.extractor != nil && len(doc.Entities) == 0 && len(doc.Relationships) == 0 {
		entities, relationships, err := m.extractor.Extract(doc.ID, doc.Content)
		if err == nil {
			doc.Entities = entities
			doc.Relationships = relationships
		}
	}
	if m.embedding != nil && len(doc.Chunks) > 0 {
		texts := make([]string, len(doc.Chunks))
		for i, c := range doc.Chunks {
			texts[i] = c.Text
		}
		vecs, err := m.embedding.EmbedBatch(ctx, texts)
		if err == nil {
			for i := range doc.Chunks {
				if i < len(vecs) {
					doc.Chunks[i].Embedding = vecs[i]
				}
			}
		}
	}
	if m.pipeline != nil {
		m.pipeline.IndexChunks(doc.Chunks)
	}
	data, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	return m.storage.Put([]byte("doc_"+doc.ID), data)
}

func (m *IndexManager) DeleteDocument(ctx context.Context, docID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.documents, docID)
	return m.storage.Delete([]byte("doc_" + docID))
}

func (m *IndexManager) GetDocument(docID string) (*DocumentRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	rec, ok := m.documents[docID]
	if !ok {
		data, err := m.storage.Get([]byte("doc_" + docID))
		if err != nil {
			return nil, err
		}
		var doc types.ProcessedDocument
		if err := json.Unmarshal(data, &doc); err != nil {
			return nil, err
		}
		rec = &DocumentRecord{
			ID:       doc.ID,
			Status:   doc.Status,
			Metadata: doc.Metadata,
			CreatedAt: doc.CreatedAt,
			UpdatedAt: doc.UpdatedAt,
		}
		m.documents[docID] = rec
	}
	return rec, nil
}

func (m *IndexManager) ListDocuments() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ids := make([]string, 0, len(m.documents))
	for id := range m.documents {
		ids = append(ids, id)
	}
	return ids
}

func (m *IndexManager) Stats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	stats := map[string]interface{}{
		"total_documents": len(m.documents),
	}
	var pending, processing, indexed, failed int
	for _, rec := range m.documents {
		switch rec.Status {
		case types.DocumentStatusPending:
			pending++
		case types.DocumentStatusProcessing:
			processing++
		case types.DocumentStatusIndexed:
			indexed++
		case types.DocumentStatusFailed:
			failed++
		}
	}
	stats["pending"] = pending
	stats["processing"] = processing
	stats["indexed"] = indexed
	stats["failed"] = failed
	return stats
}
