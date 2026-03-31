package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"KNIRVCHAIN/internal/blockchain"
	"github.com/google/uuid"
)

type cacheEntry struct {
	data    []byte
	expires time.Time
}

type MemoryCache struct {
	data map[string]cacheEntry
	mu   sync.RWMutex
	ttl  time.Duration
}

func NewMemoryCache(redisURL string) (*MemoryCache, error) {
	return &MemoryCache{
		data: make(map[string]cacheEntry),
		ttl:  1 * time.Hour,
	}, nil
}

func (mc *MemoryCache) Get(ctx context.Context, blockID uuid.UUID) (*blockchain.Block, error) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	key := fmt.Sprintf("block:%s", blockID.String())

	entry, ok := mc.data[key]
	if !ok || time.Now().After(entry.expires) {
		return nil, nil
	}

	var block blockchain.Block
	if err := json.Unmarshal(entry.data, &block); err != nil {
		return nil, fmt.Errorf("failed to unmarshal block: %w", err)
	}

	return &block, nil
}

func (mc *MemoryCache) GetByBlockNumber(ctx context.Context, blockNumber uint64) (*blockchain.Block, error) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	key := fmt.Sprintf("block:%d", blockNumber)

	entry, ok := mc.data[key]
	if !ok || time.Now().After(entry.expires) {
		return nil, nil
	}

	var block blockchain.Block
	if err := json.Unmarshal(entry.data, &block); err != nil {
		return nil, fmt.Errorf("failed to unmarshal block: %w", err)
	}

	return &block, nil
}

func (mc *MemoryCache) Set(ctx context.Context, block *blockchain.Block) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	key := fmt.Sprintf("block:%d", block.Header.Height)

	data, err := json.Marshal(block)
	if err != nil {
		return fmt.Errorf("failed to marshal block: %w", err)
	}

	mc.data[key] = cacheEntry{
		data:    data,
		expires: time.Now().Add(mc.ttl),
	}

	return nil
}

func (mc *MemoryCache) Invalidate(ctx context.Context, blockID uuid.UUID) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	key := fmt.Sprintf("block:%s", blockID.String())
	delete(mc.data, key)

	return nil
}

func (mc *MemoryCache) InvalidateByBlockNumber(ctx context.Context, blockNumber uint64) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	key := fmt.Sprintf("block:%d", blockNumber)
	delete(mc.data, key)

	return nil
}

func (mc *MemoryCache) SetQueryResult(ctx context.Context, queryHash string, results []uuid.UUID) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	key := fmt.Sprintf("query:%s", queryHash)

	data, err := json.Marshal(results)
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}

	mc.data[key] = cacheEntry{
		data:    data,
		expires: time.Now().Add(10 * time.Minute),
	}

	return nil
}

func (mc *MemoryCache) GetQueryResult(ctx context.Context, queryHash string) ([]uuid.UUID, error) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	key := fmt.Sprintf("query:%s", queryHash)

	entry, ok := mc.data[key]
	if !ok || time.Now().After(entry.expires) {
		return nil, nil
	}

	var results []uuid.UUID
	if err := json.Unmarshal(entry.data, &results); err != nil {
		return nil, fmt.Errorf("failed to unmarshal results: %w", err)
	}

	return results, nil
}

func (mc *MemoryCache) SetTransaction(ctx context.Context, tx *blockchain.Transaction) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	key := fmt.Sprintf("tx:%s", tx.TransactionHash)

	data, err := json.Marshal(tx)
	if err != nil {
		return fmt.Errorf("failed to marshal transaction: %w", err)
	}

	mc.data[key] = cacheEntry{
		data:    data,
		expires: time.Now().Add(mc.ttl),
	}

	return nil
}

func (mc *MemoryCache) GetTransaction(ctx context.Context, txHash string) (*blockchain.Transaction, error) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	key := fmt.Sprintf("tx:%s", txHash)

	entry, ok := mc.data[key]
	if !ok || time.Now().After(entry.expires) {
		return nil, nil
	}

	var tx blockchain.Transaction
	if err := json.Unmarshal(entry.data, &tx); err != nil {
		return nil, fmt.Errorf("failed to unmarshal transaction: %w", err)
	}

	return &tx, nil
}

func (mc *MemoryCache) Warm(ctx context.Context, blocks []*blockchain.Block) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	for _, block := range blocks {
		key := fmt.Sprintf("block:%d", block.Header.Height)
		data, err := json.Marshal(block)
		if err != nil {
			continue
		}
		mc.data[key] = cacheEntry{
			data:    data,
			expires: time.Now().Add(mc.ttl),
		}
	}

	return nil
}

func (mc *MemoryCache) Stats(ctx context.Context) (map[string]interface{}, error) {
	return map[string]interface{}{
		"entries": len(mc.data),
	}, nil
}

func (mc *MemoryCache) Close() error {
	return nil
}
