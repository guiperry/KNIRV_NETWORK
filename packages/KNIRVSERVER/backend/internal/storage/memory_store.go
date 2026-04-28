package storage

import (
	"sync"

	"backend_server/internal/services/memory"
	"backend_server/internal/storage/mdstorage"
)

// MemoryStore unifies access to all memory storage backends
type MemoryStore struct {
	markdown  *mdstorage.MarkdownStorageDriver
	graphrag  memory.GraphRAGInterface
	ontology  *memory.OntologyManager
	autoSync  bool
	mu        sync.RWMutex
}

// NewMemoryStore creates a new unified memory store
func NewMemoryStore(
	markdown *mdstorage.MarkdownStorageDriver,
	graphrag memory.GraphRAGInterface,
	ontology *memory.OntologyManager,
	autoSync bool,
) *MemoryStore {
	return &MemoryStore{
		markdown: markdown,
		graphrag: graphrag,
		ontology: ontology,
		autoSync: autoSync,
	}
}

// SaveDocument saves a document to markdown storage and optionally syncs to other backends
func (s *MemoryStore) SaveDocument(doc *mdstorage.MarkdownDocument) error {
	if s.markdown == nil {
		return nil
	}

	if err := s.markdown.SaveDocument(doc); err != nil {
		return err
	}

	// Auto-sync to GraphRAG if enabled
	if s.autoSync && s.graphrag != nil {
		go s.syncToGraphRAG(doc)
	}

	// Update ontology if this is an error or solution node
	if s.ontology != nil && (doc.Type == "ERROR" || doc.Type == "SOLUTION") {
		s.updateOntology(doc)
	}

	return nil
}

// LoadDocument loads a document from markdown storage
func (s *MemoryStore) LoadDocument(docType, id string) (*mdstorage.MarkdownDocument, error) {
	if s.markdown == nil {
		return nil, nil
	}
	return s.markdown.LoadDocument(docType, id)
}

// ListDocuments lists all documents from markdown storage
func (s *MemoryStore) ListDocuments() ([]*mdstorage.MarkdownDocument, error) {
	if s.markdown == nil {
		return nil, nil
	}
	return s.markdown.ListDocuments()
}

// syncToGraphRAG syncs a document to the GraphRAG index
func (s *MemoryStore) syncToGraphRAG(doc *mdstorage.MarkdownDocument) {
	if s.graphrag == nil {
		return
	}

	// Build or update GraphRAG index with document content
	// This is a placeholder - actual implementation would call the FFI bridge
	_ = doc
}

// updateOntology updates the ontology with information from a document
func (s *MemoryStore) updateOntology(doc *mdstorage.MarkdownDocument) {
	if s.ontology == nil {
		return
	}

	entity := &memory.OntologyEntity{
		ID:        doc.ID,
		Type:      memory.EntityTypePattern,
		Label:     string(doc.Content)[:min(100, len(doc.Content))],
		Properties: doc.Metadata,
		CreatedAt:  doc.Timestamp,
	}

	s.ontology.UpsertEntity(entity)
}

// SyncAll syncs all markdown documents to GraphRAG
func (s *MemoryStore) SyncAll() error {
	if s.graphrag == nil {
		return nil
	}

	docs, err := s.ListDocuments()
	if err != nil {
		return err
	}

	for _, doc := range docs {
		s.syncToGraphRAG(doc)
	}

	return nil
}

// Stats returns statistics about all storage backends
func (s *MemoryStore) Stats() map[string]interface{} {
	stats := make(map[string]interface{})

	if s.markdown != nil {
		// Add markdown stats
		stats["markdown"] = "active"
	}

	if s.graphrag != nil {
		// Add graphrag stats
		stats["graphrag"] = "active"
	}

	if s.ontology != nil {
		entityCount, relationCount := s.ontology.Stats()
		stats["ontology_entities"] = entityCount
		stats["ontology_relations"] = relationCount
	}

	stats["auto_sync"] = s.autoSync
	return stats
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
