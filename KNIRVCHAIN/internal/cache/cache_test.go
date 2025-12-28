package cache

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/knirvchain/internal/blockchain"
)

func TestNewMemoryCache(t *testing.T) {
	cache, err := NewMemoryCache("redis://localhost:6379")
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	if cache == nil {
		t.Fatal("Expected MemoryCache, got nil")
	}
	if cache.data == nil {
		t.Error("Expected data map to be initialized")
	}
	if cache.ttl != 1*time.Hour {
		t.Errorf("Expected TTL 1h, got %v", cache.ttl)
	}
}

func TestMemoryCacheSetGet(t *testing.T) {
	cache, _ := NewMemoryCache("")
	blockID := uuid.New()
	block := &blockchain.Block{
		BlockID:        blockID,
		PayloadHash:    "test-hash",
		SemanticVector: []float32{0.1, 0.2},
		Timestamp:      time.Now().Unix(),
		Category:       blockchain.CategoryGeneral,
	}

	// Set block
	err := cache.Set(context.Background(), block)
	if err != nil {
		t.Fatalf("Failed to set block: %v", err)
	}

	// Get block
	retrieved, err := cache.Get(context.Background(), blockID)
	if err != nil {
		t.Fatalf("Failed to get block: %v", err)
	}
	if retrieved == nil {
		t.Fatal("Expected block, got nil")
	}
	if retrieved.BlockID != blockID {
		t.Errorf("Expected BlockID %s, got %s", blockID, retrieved.BlockID)
	}
	if retrieved.PayloadHash != "test-hash" {
		t.Errorf("Expected PayloadHash 'test-hash', got '%s'", retrieved.PayloadHash)
	}
}

func TestMemoryCacheGetNonExistent(t *testing.T) {
	cache, _ := NewMemoryCache("")
	blockID := uuid.New()

	retrieved, err := cache.Get(context.Background(), blockID)
	if err != nil {
		t.Fatalf("Failed to get block: %v", err)
	}
	if retrieved != nil {
		t.Error("Expected nil for non-existent block")
	}
}

func TestMemoryCacheInvalidate(t *testing.T) {
	cache, _ := NewMemoryCache("")
	blockID := uuid.New()
	block := &blockchain.Block{
		BlockID:        blockID,
		PayloadHash:    "test-hash",
		SemanticVector: []float32{0.1, 0.2},
		Timestamp:      time.Now().Unix(),
		Category:       blockchain.CategoryGeneral,
	}

	// Set block
	cache.Set(context.Background(), block)

	// Verify it's cached
	retrieved, _ := cache.Get(context.Background(), blockID)
	if retrieved == nil {
		t.Fatal("Expected block to be cached")
	}

	// Invalidate
	err := cache.Invalidate(context.Background(), blockID)
	if err != nil {
		t.Fatalf("Failed to invalidate: %v", err)
	}

	// Verify it's gone
	retrieved, _ = cache.Get(context.Background(), blockID)
	if retrieved != nil {
		t.Error("Expected block to be invalidated")
	}
}

func TestMemoryCacheSetQueryResult(t *testing.T) {
	cache, _ := NewMemoryCache("")
	queryHash := "test-query-hash"
	results := []uuid.UUID{uuid.New(), uuid.New()}

	err := cache.SetQueryResult(context.Background(), queryHash, results)
	if err != nil {
		t.Fatalf("Failed to set query result: %v", err)
	}

	retrieved, err := cache.GetQueryResult(context.Background(), queryHash)
	if err != nil {
		t.Fatalf("Failed to get query result: %v", err)
	}
	if retrieved == nil {
		t.Fatal("Expected query results, got nil")
	}
	if len(retrieved) != 2 {
		t.Errorf("Expected 2 results, got %d", len(retrieved))
	}
	for i, id := range results {
		if retrieved[i] != id {
			t.Errorf("Expected result %d to be %s, got %s", i, id, retrieved[i])
		}
	}
}

func TestMemoryCacheGetQueryResultNonExistent(t *testing.T) {
	cache, _ := NewMemoryCache("")
	queryHash := "non-existent"

	retrieved, err := cache.GetQueryResult(context.Background(), queryHash)
	if err != nil {
		t.Fatalf("Failed to get query result: %v", err)
	}
	if retrieved != nil {
		t.Error("Expected nil for non-existent query result")
	}
}

func TestMemoryCacheWarm(t *testing.T) {
	cache, _ := NewMemoryCache("")
	block1 := &blockchain.Block{
		BlockID:        uuid.New(),
		PayloadHash:    "hash1",
		SemanticVector: []float32{0.1},
		Timestamp:      time.Now().Unix(),
		Category:       blockchain.CategoryGeneral,
	}
	block2 := &blockchain.Block{
		BlockID:        uuid.New(),
		PayloadHash:    "hash2",
		SemanticVector: []float32{0.2},
		Timestamp:      time.Now().Unix(),
		Category:       blockchain.CategoryGeneral,
	}

	blocks := []*blockchain.Block{block1, block2}

	err := cache.Warm(context.Background(), blocks)
	if err != nil {
		t.Fatalf("Failed to warm cache: %v", err)
	}

	// Verify blocks are cached
	retrieved1, _ := cache.Get(context.Background(), block1.BlockID)
	if retrieved1 == nil {
		t.Error("Expected block1 to be cached after warm")
	}

	retrieved2, _ := cache.Get(context.Background(), block2.BlockID)
	if retrieved2 == nil {
		t.Error("Expected block2 to be cached after warm")
	}
}

func TestMemoryCacheStats(t *testing.T) {
	cache, _ := NewMemoryCache("")

	stats, err := cache.Stats(context.Background())
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}
	if stats == nil {
		t.Fatal("Expected stats map, got nil")
	}

	entries, ok := stats["entries"]
	if !ok {
		t.Error("Expected 'entries' in stats")
	}
	if entries != 0 {
		t.Errorf("Expected 0 entries initially, got %v", entries)
	}
}

func TestMemoryCacheClose(t *testing.T) {
	cache, _ := NewMemoryCache("")

	err := cache.Close()
	if err != nil {
		t.Errorf("Expected no error on close, got %v", err)
	}
}