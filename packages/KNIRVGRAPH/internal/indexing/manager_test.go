package indexing

import (
	"KNIRVGRAPH/internal/processing"
	"KNIRVGRAPH/internal/storage"
	"KNIRVGRAPH/internal/types"
	"context"
	"testing"

	"go.uber.org/zap"
)

func TestNewIndexManager(t *testing.T) {
	store, err := storage.NewMemoryStorage()
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	logger, _ := zap.NewDevelopment()
	chunker := processing.NewChunker(types.ChunkingConfig{ChunkSize: 100})
	extractor := processing.NewExtractor(types.ExtractionConfig{})
	mgr := NewIndexManager(store, chunker, extractor, nil, nil, logger, types.ChunkingConfig{}, types.ExtractionConfig{})
	if mgr == nil {
		t.Fatal("Expected non-nil index manager")
	}
}

func TestIndexManagerStats(t *testing.T) {
	store, err := storage.NewMemoryStorage()
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	logger, _ := zap.NewDevelopment()
	mgr := NewIndexManager(store, nil, nil, nil, nil, logger, types.ChunkingConfig{}, types.ExtractionConfig{})
	stats := mgr.Stats()
	if stats["total_documents"] != 0 {
		t.Errorf("Expected 0 documents, got %v", stats["total_documents"])
	}
}

func TestIndexManagerListDocuments(t *testing.T) {
	store, err := storage.NewMemoryStorage()
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	logger, _ := zap.NewDevelopment()
	mgr := NewIndexManager(store, nil, nil, nil, nil, logger, types.ChunkingConfig{}, types.ExtractionConfig{})
	docs := mgr.ListDocuments()
	if len(docs) != 0 {
		t.Errorf("Expected 0 documents, got %d", len(docs))
	}
}

func TestIndexManagerDeleteDocument(t *testing.T) {
	store, err := storage.NewMemoryStorage()
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	logger, _ := zap.NewDevelopment()
	chunker := processing.NewChunker(types.ChunkingConfig{ChunkSize: 100})
	mgr := NewIndexManager(store, chunker, nil, nil, nil, logger, types.ChunkingConfig{}, types.ExtractionConfig{})
	doc := types.ProcessedDocument{ID: "doc1", Content: "hello world", Status: types.DocumentStatusIndexed}
	if err := mgr.IndexDocument(context.Background(), doc); err != nil {
		t.Fatalf("IndexDocument failed: %v", err)
	}
	if err := mgr.DeleteDocument(context.Background(), "doc1"); err != nil {
		t.Fatalf("DeleteDocument failed: %v", err)
	}
}
