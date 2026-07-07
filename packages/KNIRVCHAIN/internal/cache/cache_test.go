package cache

import (
	"context"
	"testing"
	"time"

	"KNIRVCHAIN/internal/blockchain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMemoryCache(t *testing.T) {
	cache, err := NewMemoryCache("redis://localhost:6379")
	require.NoError(t, err)
	assert.NotNil(t, cache)
}

func TestMemoryCache_SetAndGetByBlockNumber(t *testing.T) {
	cache, err := NewMemoryCache("")
	require.NoError(t, err)

	block := &blockchain.Block{
		Header:    blockchain.BlockHeader{Height: 1},
		Timestamp: time.Now().Unix(),
		BlockHash: []byte("test-hash"),
	}

	err = cache.Set(context.Background(), block)
	require.NoError(t, err)

	retrieved, err := cache.GetByBlockNumber(context.Background(), 1)
	require.NoError(t, err)
	require.NotNil(t, retrieved)
	assert.Equal(t, uint64(1), retrieved.Header.Height)
}

func TestMemoryCache_GetByBlockNumberNonExistent(t *testing.T) {
	cache, err := NewMemoryCache("")
	require.NoError(t, err)

	result, err := cache.GetByBlockNumber(context.Background(), 999)
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestMemoryCache_InvalidateByBlockNumber(t *testing.T) {
	cache, err := NewMemoryCache("")
	require.NoError(t, err)

	block := &blockchain.Block{
		Header:    blockchain.BlockHeader{Height: 1},
		Timestamp: time.Now().Unix(),
		BlockHash: []byte("test-hash"),
	}

	err = cache.Set(context.Background(), block)
	require.NoError(t, err)

	err = cache.InvalidateByBlockNumber(context.Background(), 1)
	require.NoError(t, err)

	result, err := cache.GetByBlockNumber(context.Background(), 1)
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestMemoryCache_SetAndGetTransaction(t *testing.T) {
	cache, err := NewMemoryCache("")
	require.NoError(t, err)

	tx := &blockchain.Transaction{
		TransactionHash: "tx-hash-123",
		From:            "sender",
		To:              "receiver",
		Value:           100,
		Timestamp:       time.Now().Unix(),
	}

	err = cache.SetTransaction(context.Background(), tx)
	require.NoError(t, err)

	retrieved, err := cache.GetTransaction(context.Background(), "tx-hash-123")
	require.NoError(t, err)
	require.NotNil(t, retrieved)
	assert.Equal(t, tx.TransactionHash, retrieved.TransactionHash)
	assert.Equal(t, tx.From, retrieved.From)
}

func TestMemoryCache_GetTransactionNonExistent(t *testing.T) {
	cache, err := NewMemoryCache("")
	require.NoError(t, err)

	result, err := cache.GetTransaction(context.Background(), "non-existent")
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestMemoryCache_SetAndGetQueryResult(t *testing.T) {
	cache, err := NewMemoryCache("")
	require.NoError(t, err)

	queryHash := "test-query-hash"
	results := []uuid.UUID{uuid.New(), uuid.New()}

	err = cache.SetQueryResult(context.Background(), queryHash, results)
	require.NoError(t, err)

	retrieved, err := cache.GetQueryResult(context.Background(), queryHash)
	require.NoError(t, err)
	require.NotNil(t, retrieved)
	assert.Equal(t, len(results), len(retrieved))
}

func TestMemoryCache_GetQueryResultNonExistent(t *testing.T) {
	cache, err := NewMemoryCache("")
	require.NoError(t, err)

	result, err := cache.GetQueryResult(context.Background(), "non-existent-query")
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestMemoryCache_Warm(t *testing.T) {
	cache, err := NewMemoryCache("")
	require.NoError(t, err)

	blocks := []*blockchain.Block{
		{Header: blockchain.BlockHeader{Height: 1}, Timestamp: time.Now().Unix(), BlockHash: []byte("hash1")},
		{Header: blockchain.BlockHeader{Height: 2}, Timestamp: time.Now().Unix(), BlockHash: []byte("hash2")},
		{Header: blockchain.BlockHeader{Height: 3}, Timestamp: time.Now().Unix(), BlockHash: []byte("hash3")},
	}

	err = cache.Warm(context.Background(), blocks)
	require.NoError(t, err)

	stats, err := cache.Stats(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 3, stats["entries"])
}

func TestMemoryCache_Stats(t *testing.T) {
	cache, err := NewMemoryCache("")
	require.NoError(t, err)

	stats, err := cache.Stats(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, stats["entries"])
}

func TestMemoryCache_Close(t *testing.T) {
	cache, err := NewMemoryCache("")
	require.NoError(t, err)

	err = cache.Close()
	assert.NoError(t, err)
}
