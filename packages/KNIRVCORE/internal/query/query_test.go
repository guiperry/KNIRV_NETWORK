package query

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/knirvchain/internal/blockchain"
	"github.com/knirvchain/internal/cache"
	"github.com/knirvchain/internal/indexing"
)

func TestIntersectResults(t *testing.T) {
	// Create a mock QueryOptimizer (we'll only test the intersectResults method)
	qo := &QueryOptimizer{}

	// Test with no results
	resultChan := make(chan []uuid.UUID, 1)
	close(resultChan)
	intersection := qo.intersectResults(resultChan)
	if len(intersection) != 0 {
		t.Errorf("Expected empty intersection, got %d items", len(intersection))
	}

	// Test with single result set
	resultChan = make(chan []uuid.UUID, 1)
	id1 := uuid.New()
	id2 := uuid.New()
	resultChan <- []uuid.UUID{id1, id2}
	close(resultChan)
	intersection = qo.intersectResults(resultChan)
	if len(intersection) != 2 {
		t.Errorf("Expected 2 items in intersection, got %d", len(intersection))
	}

	// Test with multiple result sets
	resultChan = make(chan []uuid.UUID, 2)
	id3 := uuid.New()
	resultChan <- []uuid.UUID{id1, id2, id3}
	resultChan <- []uuid.UUID{id1, id3}
	close(resultChan)
	intersection = qo.intersectResults(resultChan)
	if len(intersection) != 2 {
		t.Errorf("Expected 2 items in intersection, got %d", len(intersection))
	}
	// Check that id1 and id3 are in the intersection
	found := make(map[uuid.UUID]bool)
	for _, id := range intersection {
		found[id] = true
	}
	if !found[id1] || !found[id3] {
		t.Error("Expected id1 and id3 in intersection")
	}
	if found[id2] {
		t.Error("Expected id2 not in intersection")
	}
}

func TestCalculateSimilarity(t *testing.T) {
	qo := &QueryOptimizer{}

	// Test identical vectors
	v1 := []float32{1.0, 0.0, 0.0}
	v2 := []float32{1.0, 0.0, 0.0}
	similarity := qo.calculateSimilarity(v1, v2)
	if similarity != 1.0 {
		t.Errorf("Expected similarity 1.0 for identical vectors, got %f", similarity)
	}

	// Test orthogonal vectors
	v1 = []float32{1.0, 0.0}
	v2 = []float32{0.0, 1.0}
	similarity = qo.calculateSimilarity(v1, v2)
	if similarity != 0.0 {
		t.Errorf("Expected similarity 0.0 for orthogonal vectors, got %f", similarity)
	}

	// Test different lengths
	v1 = []float32{1.0, 0.0}
	v2 = []float32{1.0}
	similarity = qo.calculateSimilarity(v1, v2)
	if similarity != 0.0 {
		t.Errorf("Expected similarity 0.0 for different length vectors, got %f", similarity)
	}

	// Test zero vectors
	v1 = []float32{0.0, 0.0}
	v2 = []float32{1.0, 1.0}
	similarity = qo.calculateSimilarity(v1, v2)
	if similarity != 0.0 {
		t.Errorf("Expected similarity 0.0 for zero vector, got %f", similarity)
	}
}

func TestHashQuery(t *testing.T) {
	qo := &QueryOptimizer{}

	req1 := QueryRequest{
		Vector:   []float32{1.0, 2.0},
		Limit:    10,
		Keywords: []string{"test"},
	}

	req2 := QueryRequest{
		Vector:   []float32{1.0, 2.0},
		Limit:    10,
		Keywords: []string{"test"},
	}

	hash1 := qo.hashQuery(req1)
	hash2 := qo.hashQuery(req2)

	if hash1 != hash2 {
		t.Error("Expected identical requests to have same hash")
	}

	// Test different requests
	req3 := QueryRequest{
		Vector: []float32{1.0, 3.0},
		Limit:  10,
	}
	hash3 := qo.hashQuery(req3)

	if hash1 == hash3 {
		t.Error("Expected different requests to have different hashes")
	}
}

func TestSqrt(t *testing.T) {
	// Test sqrt function (it's a package-level function, approximate implementation)
	result := sqrt(4.0)
	if result < 1.9 || result > 2.1 {
		t.Errorf("Expected sqrt(4.0) ≈ 2.0, got %f", result)
	}
	result = sqrt(9.0)
	if result < 2.9 || result > 3.1 {
		t.Errorf("Expected sqrt(9.0) ≈ 3.0, got %f", result)
	}
	result = sqrt(0.0)
	if result < -0.1 || result > 0.1 {
		t.Errorf("Expected sqrt(0.0) ≈ 0.0, got %f", result)
	}
	result = sqrt(-1.0)
	if result != 0.0 {
		t.Errorf("Expected sqrt(-1.0) = 0.0, got %f", result)
	}
}

func TestNewQueryOptimizer(t *testing.T) {
	// Create mock dependencies
	indexManager := &indexing.MultiIndexManager{}
	cache := &cache.MemoryCache{}
	chain := &blockchain.ChainNode{}

	optimizer := NewQueryOptimizer(indexManager, cache, chain)

	if optimizer == nil {
		t.Error("Expected non-nil QueryOptimizer")
	}
	if optimizer.indexManager != indexManager {
		t.Error("Expected indexManager to be set")
	}
	if optimizer.cache != cache {
		t.Error("Expected cache to be set")
	}
	if optimizer.chain != chain {
		t.Error("Expected chain to be set")
	}
}

// Mock index for testing
type mockIndex struct {
	results []uuid.UUID
	err     error
}

func (m *mockIndex) Search(ctx context.Context, query interface{}) ([]uuid.UUID, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.results, nil
}

func TestSemanticSearch(t *testing.T) {
	qo := &QueryOptimizer{
		indexManager: &indexing.MultiIndexManager{},
	}

	// Mock the index manager to return our mock index
	// Since we can't easily inject, we'll test the method directly with a nil indexManager
	// which should return an error

	_, err := qo.semanticSearch(context.Background(), []float32{1.0}, 10)
	if err == nil {
		t.Error("Expected error when index not available")
	}
}

func TestCategorySearch(t *testing.T) {
	qo := &QueryOptimizer{
		indexManager: &indexing.MultiIndexManager{},
	}

	_, err := qo.categorySearch(context.Background(), blockchain.CategoryGeneral)
	if err == nil {
		t.Error("Expected error when index not available")
	}
}

func TestTemporalSearch(t *testing.T) {
	qo := &QueryOptimizer{
		indexManager: &indexing.MultiIndexManager{},
	}

	timeRange := indexing.TimeRangeQuery{
		StartTime: 1000,
		EndTime:   2000,
	}

	_, err := qo.temporalSearch(context.Background(), timeRange)
	if err == nil {
		t.Error("Expected error when index not available")
	}
}
