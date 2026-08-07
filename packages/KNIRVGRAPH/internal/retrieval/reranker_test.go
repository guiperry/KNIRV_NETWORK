package retrieval

import (
	"KNIRVGRAPH/internal/types"
	"context"
	"testing"
)

func TestReranker(t *testing.T) {
	reranker := NewReranker("", "")
	candidates := []types.VectorSearchResult{
		{ID: "a", Score: 0.5, Metadata: map[string]interface{}{"text": "hello world"}},
		{ID: "b", Score: 0.3, Metadata: map[string]interface{}{"text": "goodbye world"}},
	}
	results, err := reranker.Rerank(context.Background(), "hello", candidates, 2)
	if err != nil {
		t.Fatalf("Rerank failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}
}

func TestRerankerEmpty(t *testing.T) {
	reranker := NewReranker("", "")
	results, err := reranker.Rerank(context.Background(), "query", nil, 5)
	if err != nil {
		t.Fatalf("Rerank failed: %v", err)
	}
	if results != nil {
		t.Error("Expected nil results for empty input")
	}
}

func TestScorePairs(t *testing.T) {
	reranker := NewReranker("", "")
	candidates := []types.VectorSearchResult{
		{ID: "a", Score: 0.5, Metadata: map[string]interface{}{"text": "hello"}},
	}
	scores, err := reranker.ScorePairs(context.Background(), "hello", candidates)
	if err != nil {
		t.Fatalf("ScorePairs failed: %v", err)
	}
	if len(scores) != 1 {
		t.Fatalf("Expected 1 score, got %d", len(scores))
	}
}
