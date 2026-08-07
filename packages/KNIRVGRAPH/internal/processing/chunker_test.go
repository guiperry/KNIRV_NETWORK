package processing

import (
	"testing"

	"KNIRVGRAPH/internal/types"
)

func TestChunkerRecursive(t *testing.T) {
	chunker := NewChunker(types.ChunkingConfig{
		Strategy:  types.ChunkStrategyRecursive,
		ChunkSize: 50,
		Overlap:   10,
	})
	text := "This is sentence one. This is sentence two. This is sentence three. This is sentence four."
	chunks, err := chunker.Chunk("doc1", text)
	if err != nil {
		t.Fatalf("Chunk failed: %v", err)
	}
	if len(chunks) == 0 {
		t.Fatal("Expected at least one chunk")
	}
	for _, c := range chunks {
		if c.DocumentID != "doc1" {
			t.Errorf("Expected document_id doc1, got %s", c.DocumentID)
		}
		if len(c.Text) == 0 {
			t.Error("Expected non-empty chunk text")
		}
	}
}

func TestChunkerToken(t *testing.T) {
	chunker := NewChunker(types.ChunkingConfig{
		Strategy:  types.ChunkStrategyToken,
		ChunkSize: 5,
		Overlap:   1,
	})
	text := "one two three four five six seven eight"
	chunks, err := chunker.Chunk("doc2", text)
	if err != nil {
		t.Fatalf("Chunk failed: %v", err)
	}
	if len(chunks) == 0 {
		t.Fatal("Expected at least one chunk")
	}
	found := false
	for _, c := range chunks {
		if c.Metadata["tokens"] != nil {
			found = true
		}
	}
	if !found {
		t.Error("Expected token metadata in at least one chunk")
	}
}

func TestChunkerSemantic(t *testing.T) {
	chunker := NewChunker(types.ChunkingConfig{
		Strategy:  types.ChunkStrategySemantic,
		ChunkSize: 100,
		Overlap:   0,
	})
	text := "First sentence. Second sentence. Third sentence."
	chunks, err := chunker.Chunk("doc3", text)
	if err != nil {
		t.Fatalf("Chunk failed: %v", err)
	}
	if len(chunks) == 0 {
		t.Fatal("Expected at least one chunk")
	}
}

func TestChunkerEmptyText(t *testing.T) {
	chunker := NewChunker(types.ChunkingConfig{
		Strategy:  types.ChunkStrategyRecursive,
		ChunkSize: 100,
	})
	chunks, err := chunker.Chunk("doc4", "")
	if err != nil {
		t.Fatalf("Chunk failed: %v", err)
	}
	if len(chunks) != 1 {
		t.Fatalf("Expected 1 chunk for empty text, got %d", len(chunks))
	}
}
