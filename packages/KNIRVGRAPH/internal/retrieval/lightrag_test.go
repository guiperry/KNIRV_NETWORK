package retrieval

import (
	"KNIRVGRAPH/internal/types"
	"context"
	"testing"
)

func TestLightRAGRetriever(t *testing.T) {
	pipeline := NewRetrievalPipeline(0.5)
	pipeline.IndexChunks([]types.Chunk{
		{ID: "c1", DocumentID: "d1", Text: "hello world", Embedding: []float32{1, 0, 0}},
	})
	retriever := NewLightRAGRetriever(pipeline)
	results, err := retriever.Retrieve(context.Background(), "hello", []float32{1, 0, 0}, 1)
	if err != nil {
		t.Fatalf("Retrieve failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}
	if results[0].Metadata["strategy"] != "lightrag" {
		t.Errorf("Expected strategy lightrag, got %v", results[0].Metadata["strategy"])
	}
}

func TestHippoRAGRetriever(t *testing.T) {
	pipeline := NewRetrievalPipeline(0.5)
	pipeline.IndexChunks([]types.Chunk{
		{ID: "c1", DocumentID: "d1", Text: "hello world", Embedding: []float32{1, 0, 0}},
		{ID: "c2", DocumentID: "d1", Text: "hello world", Embedding: []float32{1, 0, 0}},
	})
	retriever := NewHippoRAGRetriever(pipeline)
	results, err := retriever.Retrieve(context.Background(), "hello", []float32{1, 0, 0}, 1)
	if err != nil {
		t.Fatalf("Retrieve failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Expected 1 result after dedup, got %d", len(results))
	}
}
