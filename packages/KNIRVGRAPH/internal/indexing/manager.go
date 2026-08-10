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
	"sort"
	"sync"
	"time"

	"go.uber.org/zap"
)

const (
	docPrefix      = "rag_doc_"
	statusPrefix   = "rag_status_"
	manifestPrefix = "rag_manifest_"
)

type DocumentRecord struct {
	ID                string                 `json:"id"`
	Status            types.DocumentStatus   `json:"status"`
	ChunkCount        int                    `json:"chunk_count"`
	EntityCount       int                    `json:"entity_count"`
	RelationshipCount int                    `json:"relationship_count"`
	Error             string                 `json:"error,omitempty"`
	Metadata          map[string]interface{} `json:"metadata"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
}
type artifactManifest struct {
	DocumentID                 string `json:"document_id"`
	ChunkIDs, NodeIDs, EdgeIDs []string
}
type indexJob struct {
	cancel context.CancelFunc
	done   chan struct{}
}

type IndexManager struct {
	storage   storage.GraphStorage
	chunker   *processing.Chunker
	extractor *processing.Extractor
	embedding *embeddings.EmbeddingService
	pipeline  *retrieval.RetrievalPipeline
	logger    *zap.Logger
	mu        sync.RWMutex
	documents map[string]*DocumentRecord
	jobs      map[string]*indexJob
	observer  func(time.Duration, error)
	config    types.ChunkingConfig
	extConfig types.ExtractionConfig
}

func (m *IndexManager) SetObserver(observer func(time.Duration, error)) {
	m.mu.Lock()
	m.observer = observer
	m.mu.Unlock()
}

func NewIndexManager(store storage.GraphStorage, chunker *processing.Chunker, extractor *processing.Extractor, embedding *embeddings.EmbeddingService, pipeline *retrieval.RetrievalPipeline, logger *zap.Logger, chunkConfig types.ChunkingConfig, extConfig types.ExtractionConfig) *IndexManager {
	if logger == nil {
		logger = zap.NewNop()
	}
	m := &IndexManager{storage: store, chunker: chunker, extractor: extractor, embedding: embedding, pipeline: pipeline, logger: logger, documents: make(map[string]*DocumentRecord), jobs: make(map[string]*indexJob), config: chunkConfig, extConfig: extConfig}
	if err := m.Recover(context.Background()); err != nil {
		logger.Warn("recover retrieval index", zap.Error(err))
	}
	return m
}

func (m *IndexManager) IndexDocument(ctx context.Context, doc types.ProcessedDocument) error {
	return m.IndexDocumentWithOptions(ctx, doc, false)
}
func (m *IndexManager) IndexDocumentWithOptions(ctx context.Context, doc types.ProcessedDocument, overwrite bool) error {
	if doc.ID == "" || doc.Content == "" {
		return fmt.Errorf("document id and content are required")
	}
	m.mu.Lock()
	if rec, ok := m.documents[doc.ID]; ok {
		if rec.Status == types.DocumentStatusProcessing {
			m.mu.Unlock()
			return fmt.Errorf("document %s is already being processed", doc.ID)
		}
		if !overwrite {
			m.mu.Unlock()
			return fmt.Errorf("document %s already exists", doc.ID)
		}
	}
	now := time.Now()
	record := &DocumentRecord{ID: doc.ID, Status: types.DocumentStatusProcessing, Metadata: doc.Metadata, CreatedAt: now, UpdatedAt: now}
	m.documents[doc.ID] = record
	workCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	job := &indexJob{cancel: cancel, done: make(chan struct{})}
	m.jobs[doc.ID] = job
	m.mu.Unlock()
	if err := m.persistRecord(record); err != nil {
		cancel()
		m.mu.Lock()
		delete(m.jobs, doc.ID)
		delete(m.documents, doc.ID)
		close(job.done)
		m.mu.Unlock()
		return err
	}
	go func() {
		started := time.Now()
		defer func() {
			m.mu.Lock()
			if m.jobs[doc.ID] == job {
				delete(m.jobs, doc.ID)
			}
			close(job.done)
			m.mu.Unlock()
		}()
		processed, err := m.processDocument(workCtx, doc)
		m.mu.Lock()
		rec := m.documents[doc.ID]
		if rec == nil {
			m.mu.Unlock()
			return
		}
		if err != nil {
			rec.Status = types.DocumentStatusFailed
			rec.Error = err.Error()
		} else {
			rec.Status = types.DocumentStatusIndexed
			rec.Error = ""
			rec.ChunkCount = len(processed.Chunks)
			rec.EntityCount = len(processed.Entities)
			rec.RelationshipCount = len(processed.Relationships)
		}
		rec.UpdatedAt = time.Now()
		snapshot := *rec
		m.mu.Unlock()
		_ = m.persistRecord(&snapshot)
		m.mu.RLock()
		observer := m.observer
		m.mu.RUnlock()
		if observer != nil {
			observer(time.Since(started), err)
		}
		if err != nil {
			m.logger.Error("document indexing failed", zap.String("document_id", doc.ID), zap.Error(err))
		} else {
			m.logger.Info("document indexed", zap.String("document_id", doc.ID), zap.Int("chunks", len(processed.Chunks)), zap.Int("entities", len(processed.Entities)))
		}
	}()
	return nil
}

func (m *IndexManager) processDocument(ctx context.Context, doc types.ProcessedDocument) (types.ProcessedDocument, error) {
	if old, err := m.loadManifest(doc.ID); err == nil {
		if err := m.deleteArtifacts(old); err != nil {
			return doc, fmt.Errorf("remove previous artifacts: %w", err)
		}
	}
	if len(doc.Chunks) == 0 && m.chunker != nil {
		chunks, err := m.chunker.Chunk(doc.ID, doc.Content)
		if err != nil {
			return doc, fmt.Errorf("chunking failed: %w", err)
		}
		doc.Chunks = chunks
	}
	for i := range doc.Chunks {
		if doc.Chunks[i].Metadata == nil {
			doc.Chunks[i].Metadata = map[string]interface{}{}
		}
		for k, v := range doc.Metadata {
			doc.Chunks[i].Metadata[k] = v
		}
		doc.Chunks[i].Metadata["document_id"] = doc.ID
		doc.Chunks[i].Metadata["source_id"] = doc.SourceID
	}
	if m.extractor != nil && len(doc.Entities) == 0 && len(doc.Relationships) == 0 {
		entities, rels, err := m.extractor.ExtractContext(ctx, doc.ID, doc.Content)
		if err != nil {
			return doc, fmt.Errorf("extraction failed: %w", err)
		}
		doc.Entities, doc.Relationships = entities, rels
	}
	if m.embedding == nil {
		return doc, fmt.Errorf("embedding service is not configured")
	}
	texts := make([]string, len(doc.Chunks))
	for i, c := range doc.Chunks {
		texts[i] = c.Text
	}
	vecs, err := m.embedding.EmbedBatch(ctx, texts)
	if err != nil {
		return doc, fmt.Errorf("embedding failed: %w", err)
	}
	if len(vecs) != len(doc.Chunks) {
		return doc, fmt.Errorf("embedding count mismatch")
	}
	for i := range doc.Chunks {
		doc.Chunks[i].Embedding = vecs[i]
	}
	doc.Status = types.DocumentStatusIndexed
	doc.UpdatedAt = time.Now()
	manifest, err := m.persistDocumentArtifacts(doc)
	if err != nil {
		return doc, err
	}
	if m.pipeline != nil {
		if err := m.pipeline.IndexChunksWithError(doc.Chunks); err != nil {
			return doc, fmt.Errorf("vector indexing failed: %w", err)
		}
	}
	_ = manifest
	return doc, nil
}

func (m *IndexManager) persistDocumentArtifacts(doc types.ProcessedDocument) (*artifactManifest, error) {
	manifest := &artifactManifest{DocumentID: doc.ID}
	docNode := types.NewGraphNode("rag:doc:"+doc.ID, nil, types.GraphData{Payload: map[string]interface{}{"kind": "document", "source_id": doc.SourceID, "metadata": doc.Metadata}})
	manifest.NodeIDs = append(manifest.NodeIDs, docNode.ID)
	if err := m.putNode(docNode); err != nil {
		return nil, err
	}
	for _, c := range doc.Chunks {
		manifest.ChunkIDs = append(manifest.ChunkIDs, c.ID)
		n := types.NewGraphNode("rag:chunk:"+c.ID, nil, types.GraphData{Payload: map[string]interface{}{"kind": "chunk", "document_id": doc.ID, "text": c.Text, "index": c.Index}})
		manifest.NodeIDs = append(manifest.NodeIDs, n.ID)
		if err := m.putNode(n); err != nil {
			return nil, err
		}
		e := semanticEdge("rag:doc:"+doc.ID, n.ID, "document_has_chunk", 1)
		manifest.EdgeIDs = append(manifest.EdgeIDs, e.ID)
		if err := m.putEdge(e); err != nil {
			return nil, err
		}
	}
	for _, entity := range doc.Entities {
		n := types.NewGraphNode("rag:"+entity.ID, nil, types.GraphData{Payload: map[string]interface{}{"kind": "entity", "name": entity.Name, "entity_type": entity.Type, "confidence": entity.Confidence, "document_id": doc.ID}})
		manifest.NodeIDs = append(manifest.NodeIDs, n.ID)
		if err := m.putNode(n); err != nil {
			return nil, err
		}
		e := semanticEdge("rag:doc:"+doc.ID, n.ID, "mentions", entity.Confidence)
		manifest.EdgeIDs = append(manifest.EdgeIDs, e.ID)
		if err := m.putEdge(e); err != nil {
			return nil, err
		}
	}
	for _, rel := range doc.Relationships {
		from, to := "rag:"+rel.Source, "rag:"+rel.Target
		e := semanticEdge(from, to, rel.Type, rel.Weight)
		e.Metadata["document_id"] = doc.ID
		e.Metadata["evidence"] = rel.Evidence
		manifest.EdgeIDs = append(manifest.EdgeIDs, e.ID)
		if err := m.putEdge(e); err != nil {
			return nil, err
		}
	}
	rawDoc, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}
	rawManifest, err := json.Marshal(manifest)
	if err != nil {
		return nil, err
	}
	batch := m.storage.Batch()
	batch.Put([]byte(docPrefix+doc.ID), rawDoc)
	batch.Put([]byte(manifestPrefix+doc.ID), rawManifest)
	if err := batch.Write(); err != nil {
		return nil, err
	}
	return manifest, nil
}

func semanticEdge(from, to, kind string, weight float64) *types.Edge {
	e := types.NewEdge(from, to, types.CustomEdge, weight)
	e.ID = stableArtifactID("rag:edge", from, to, kind)
	e.Metadata["relationship_type"] = kind
	return e
}
func stableArtifactID(prefix string, parts ...string) string {
	return processingStableID(append([]string{prefix}, parts...)...)
}
func processingStableID(parts ...string) string {
	b, _ := json.Marshal(parts)
	var h uint64 = 1469598103934665603
	for _, c := range b {
		h ^= uint64(c)
		h *= 1099511628211
	}
	return fmt.Sprintf("%s:%016x", parts[0], h)
}
func (m *IndexManager) putNode(n *types.GraphNode) error {
	raw, err := json.Marshal(n)
	if err != nil {
		return err
	}
	return m.storage.PutNode(n.ID, raw)
}
func (m *IndexManager) putEdge(e *types.Edge) error {
	raw, err := json.Marshal(e)
	if err != nil {
		return err
	}
	return m.storage.PutEdge(e.ID, raw)
}

func (m *IndexManager) DeleteDocument(ctx context.Context, docID string) error {
	_ = ctx
	m.mu.RLock()
	job := m.jobs[docID]
	m.mu.RUnlock()
	if job != nil {
		job.cancel()
		<-job.done
	}
	manifest, err := m.loadManifest(docID)
	if err != nil && err != storage.ErrNotFound {
		return err
	}
	if manifest != nil {
		if err := m.deleteArtifacts(manifest); err != nil {
			return err
		}
	}
	batch := m.storage.Batch()
	batch.Delete([]byte(docPrefix + docID))
	batch.Delete([]byte(statusPrefix + docID))
	batch.Delete([]byte(manifestPrefix + docID))
	if err := batch.Write(); err != nil {
		return err
	}
	m.mu.Lock()
	delete(m.documents, docID)
	m.mu.Unlock()
	return nil
}
func (m *IndexManager) deleteArtifacts(manifest *artifactManifest) error {
	if m.pipeline != nil {
		_ = m.pipeline.DeleteChunks(manifest.ChunkIDs)
	}
	for _, id := range manifest.EdgeIDs {
		if err := m.storage.DeleteEdge(id); err != nil {
			return err
		}
	}
	for _, id := range manifest.NodeIDs {
		if err := m.storage.DeleteNode(id); err != nil {
			return err
		}
	}
	return nil
}

func (m *IndexManager) GetDocument(docID string) (*DocumentRecord, error) {
	m.mu.RLock()
	rec := m.documents[docID]
	m.mu.RUnlock()
	if rec != nil {
		copy := *rec
		return &copy, nil
	}
	raw, err := m.storage.Get([]byte(statusPrefix + docID))
	if err != nil {
		return nil, err
	}
	var loaded DocumentRecord
	if err := json.Unmarshal(raw, &loaded); err != nil {
		return nil, err
	}
	m.mu.Lock()
	m.documents[docID] = &loaded
	m.mu.Unlock()
	return &loaded, nil
}
func (m *IndexManager) persistRecord(rec *DocumentRecord) error {
	raw, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	return m.storage.Put([]byte(statusPrefix+rec.ID), raw)
}
func (m *IndexManager) loadManifest(id string) (*artifactManifest, error) {
	raw, err := m.storage.Get([]byte(manifestPrefix + id))
	if err != nil {
		return nil, err
	}
	var out artifactManifest
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (m *IndexManager) Recover(ctx context.Context) error {
	_ = ctx
	records, err := m.storage.ScanPrefix([]byte(statusPrefix))
	if err != nil {
		return err
	}
	ids := make([]string, 0, len(records))
	for _, raw := range records {
		var rec DocumentRecord
		if json.Unmarshal(raw, &rec) == nil {
			if rec.Status == types.DocumentStatusProcessing {
				rec.Status = types.DocumentStatusFailed
				rec.Error = "interrupted during indexing"
				rec.UpdatedAt = time.Now()
				_ = m.persistRecord(&rec)
			}
			m.documents[rec.ID] = &rec
			ids = append(ids, rec.ID)
		}
	}
	sort.Strings(ids)
	for _, id := range ids {
		raw, err := m.storage.Get([]byte(docPrefix + id))
		if err != nil {
			continue
		}
		var doc types.ProcessedDocument
		if json.Unmarshal(raw, &doc) == nil && m.pipeline != nil {
			if err := m.pipeline.IndexChunksWithError(doc.Chunks); err != nil {
				return err
			}
		}
	}
	return nil
}
func (m *IndexManager) Optimize() error {
	if m.pipeline == nil {
		return nil
	}
	return m.pipeline.Optimize()
}
func (m *IndexManager) ListDocuments() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ids := make([]string, 0, len(m.documents))
	for id := range m.documents {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
func (m *IndexManager) Stats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	stats := map[string]interface{}{"total_documents": len(m.documents)}
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
	if m.pipeline != nil {
		stats["vectors"] = m.pipeline.VectorCount()
	}
	return stats
}
