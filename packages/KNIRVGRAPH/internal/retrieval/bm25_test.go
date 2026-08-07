package retrieval

import (
	"KNIRVGRAPH/internal/types"
	"testing"
)

func TestBM25Index(t *testing.T) {
	idx := NewBM25Index()
	idx.Add("doc1", "the quick brown fox jumps over the lazy dog near the riverbank")
	idx.Add("doc2", "a quick brown dog runs fast through the green meadow")
	idx.Add("doc3", "the lazy cat sleeps all day long in the warm sun")
	idx.Add("doc4", "quick brown foxes are common in the forest during autumn")
	idx.Add("doc5", "dogs are loyal companions that love to run and play in the park")

	results, err := idx.Search("quick brown", 2)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}
	if results[0].Score <= 0 {
		t.Errorf("Expected positive BM25 score, got %f", results[0].Score)
	}
}

func TestBM25IndexRemove(t *testing.T) {
	idx := NewBM25Index()
	idx.Add("doc1", "hello world this is a test")
	idx.Add("doc2", "another test document with more words")
	idx.Remove("doc1")
	results, _ := idx.Search("hello", 1)
	if len(results) != 0 {
		t.Errorf("Expected 0 results after removal, got %d", len(results))
	}
}

func TestInMemoryChunkStore(t *testing.T) {
	store := NewInMemoryChunkStore()
	chunk := &types.Chunk{ID: "c1", DocumentID: "d1", Text: "hello"}
	store.Put(chunk)

	c, err := store.GetChunk("c1")
	if err != nil {
		t.Fatalf("GetChunk failed: %v", err)
	}
	if c.Text != "hello" {
		t.Errorf("Expected text 'hello', got %s", c.Text)
	}

	chunks, err := store.GetChunksByDoc("d1")
	if err != nil {
		t.Fatalf("GetChunksByDoc failed: %v", err)
	}
	if len(chunks) != 1 {
		t.Errorf("Expected 1 chunk, got %d", len(chunks))
	}

	all, err := store.GetAllChunks()
	if err != nil {
		t.Fatalf("GetAllChunks failed: %v", err)
	}
	if len(all) != 1 {
		t.Errorf("Expected 1 chunk, got %d", len(all))
	}
}

func TestRetrievalPipeline(t *testing.T) {
	pipeline := NewRetrievalPipeline(0.5)
	chunks := []types.Chunk{
		{ID: "c1", DocumentID: "d1", Text: "hello world test", Embedding: []float32{1, 0, 0}},
		{ID: "c2", DocumentID: "d1", Text: "goodbye world test", Embedding: []float32{0, 1, 0}},
	}
	pipeline.IndexChunks(chunks)

	results, err := pipeline.Search("test", []float32{1, 0, 0}, 2)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}
}

func TestNormalizeScores(t *testing.T) {
	results := []types.VectorSearchResult{
		{ID: "a", Score: 10},
		{ID: "b", Score: 5},
	}
	normalized := NormalizeScores(results)
	if normalized[0].Score != 1.0 {
		t.Errorf("Expected normalized score 1.0, got %f", normalized[0].Score)
	}
	if normalized[1].Score != 0.5 {
		t.Errorf("Expected normalized score 0.5, got %f", normalized[1].Score)
	}
}
