package query

import (
	"context"
	"testing"

	"KNIRVCHAIN/internal/cache"
	"KNIRVCHAIN/internal/indexing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewQueryOptimizer(t *testing.T) {
	indexManager := indexing.NewMultiIndexManager()
	cacheInstance, err := cache.NewMemoryCache("")
	require.NoError(t, err)

	optimizer := NewQueryOptimizer(indexManager, cacheInstance, nil)
	assert.NotNil(t, optimizer)
	assert.Equal(t, indexManager, optimizer.indexManager)
	assert.Equal(t, cacheInstance, optimizer.cache)
}

func TestQueryOptimizer_OptimizedSearch_EmptyVector(t *testing.T) {
	indexManager := indexing.NewMultiIndexManager()
	cacheInstance, _ := cache.NewMemoryCache("")

	optimizer := NewQueryOptimizer(indexManager, cacheInstance, nil)

	req := QueryRequest{
		Vector: []float32{},
		Limit:  10,
	}

	results, err := optimizer.OptimizedSearch(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, results)
}

func TestQueryOptimizer_OptimizedSearch_NoIndexes(t *testing.T) {
	indexManager := indexing.NewMultiIndexManager()
	cacheInstance, _ := cache.NewMemoryCache("")

	optimizer := NewQueryOptimizer(indexManager, cacheInstance, nil)

	req := QueryRequest{
		Vector: []float32{1.0, 2.0, 3.0, 4.0},
		Limit:  10,
	}

	results, err := optimizer.OptimizedSearch(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, results)
}

func TestQueryRequest_Defaults(t *testing.T) {
	req := QueryRequest{
		Limit: 10,
	}

	assert.Equal(t, 10, req.Limit)
	assert.Nil(t, req.Vector)
	assert.Nil(t, req.Category)
	assert.Nil(t, req.TimeRange)
}

func TestQueryOptimizer_hashQuery(t *testing.T) {
	indexManager := indexing.NewMultiIndexManager()
	cacheInstance, _ := cache.NewMemoryCache("")

	optimizer := NewQueryOptimizer(indexManager, cacheInstance, nil)

	req1 := QueryRequest{
		Vector: []float32{1.0, 2.0, 3.0},
		Limit:  10,
	}

	req2 := QueryRequest{
		Vector: []float32{1.0, 2.0, 3.0},
		Limit:  10,
	}

	hash1 := optimizer.hashQuery(req1)
	hash2 := optimizer.hashQuery(req2)

	assert.Equal(t, hash1, hash2)

	req3 := QueryRequest{
		Vector: []float32{4.0, 5.0, 6.0},
		Limit:  10,
	}

	hash3 := optimizer.hashQuery(req3)
	assert.NotEqual(t, hash1, hash3)
}
