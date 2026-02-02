package indexing

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/knirvchain/internal/blockchain"
)

type IndexType string

const (
	IndexTypeSemantic IndexType = "semantic"
	IndexTypeTemporal IndexType = "temporal"
	IndexTypeCategory IndexType = "category"
	IndexTypeFullText IndexType = "fulltext"
)

type Index interface {
	Add(ctx context.Context, block *blockchain.Block) error
	Search(ctx context.Context, query interface{}) ([]uuid.UUID, error)
	Remove(ctx context.Context, blockID uuid.UUID) error
	Rebuild(ctx context.Context) error
}

type MultiIndexManager struct {
	indexes map[IndexType]Index
	mu      sync.RWMutex
}

func NewMultiIndexManager() *MultiIndexManager {
	return &MultiIndexManager{
		indexes: make(map[IndexType]Index),
	}
}

func (mim *MultiIndexManager) RegisterIndex(indexType IndexType, index Index) {
	mim.mu.Lock()
	defer mim.mu.Unlock()
	mim.indexes[indexType] = index
}

func (mim *MultiIndexManager) GetIndex(indexType IndexType) Index {
	mim.mu.RLock()
	defer mim.mu.RUnlock()
	return mim.indexes[indexType]
}

func (mim *MultiIndexManager) AddBlock(ctx context.Context, block *blockchain.Block) error {
	mim.mu.RLock()
	defer mim.mu.RUnlock()

	var wg sync.WaitGroup
	errChan := make(chan error, len(mim.indexes))

	for _, index := range mim.indexes {
		wg.Add(1)
		go func(idx Index) {
			defer wg.Done()
			if err := idx.Add(ctx, block); err != nil {
				errChan <- err
			}
		}(index)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			return fmt.Errorf("index error: %w", err)
		}
	}

	return nil
}

// SemanticIndex implements HNSW vector similarity search
type SemanticIndex struct {
	vectors map[uuid.UUID][]float32
	hnsw    *HNSWIndex
	mu      sync.RWMutex
}

func NewSemanticIndex(dimension int) *SemanticIndex {
	return &SemanticIndex{
		vectors: make(map[uuid.UUID][]float32),
		hnsw:    NewHNSWIndex(dimension, 16, 200),
	}
}

func (si *SemanticIndex) Add(ctx context.Context, block *blockchain.Block) error {
	si.mu.Lock()
	defer si.mu.Unlock()

	si.vectors[block.BlockID] = block.SemanticVector
	return si.hnsw.Add(block.BlockID, block.SemanticVector)
}

func (si *SemanticIndex) Search(ctx context.Context, query interface{}) ([]uuid.UUID, error) {
	si.mu.RLock()
	defer si.mu.RUnlock()

	vector, ok := query.([]float32)
	if !ok {
		return nil, fmt.Errorf("invalid query type for semantic search")
	}

	return si.hnsw.Search(vector, 100)
}

func (si *SemanticIndex) Remove(ctx context.Context, blockID uuid.UUID) error {
	si.mu.Lock()
	defer si.mu.Unlock()

	delete(si.vectors, blockID)
	return si.hnsw.Remove(blockID)
}

func (si *SemanticIndex) Rebuild(ctx context.Context) error {
	si.mu.Lock()
	defer si.mu.Unlock()

	si.hnsw = NewHNSWIndex(len(si.vectors[uuid.Nil]), 16, 200)

	for id, vector := range si.vectors {
		if err := si.hnsw.Add(id, vector); err != nil {
			return err
		}
	}

	return nil
}

// TemporalIndex implements B-tree for time-range queries
type TemporalIndex struct {
	timeline map[int64][]uuid.UUID
	mu       sync.RWMutex
}

func NewTemporalIndex() *TemporalIndex {
	return &TemporalIndex{
		timeline: make(map[int64][]uuid.UUID),
	}
}

func (ti *TemporalIndex) Add(ctx context.Context, block *blockchain.Block) error {
	ti.mu.Lock()
	defer ti.mu.Unlock()

	ti.timeline[block.Timestamp] = append(ti.timeline[block.Timestamp], block.BlockID)
	return nil
}

type TimeRangeQuery struct {
	StartTime int64
	EndTime   int64
}

func (ti *TemporalIndex) Search(ctx context.Context, query interface{}) ([]uuid.UUID, error) {
	ti.mu.RLock()
	defer ti.mu.RUnlock()

	timeRange, ok := query.(TimeRangeQuery)
	if !ok {
		return nil, fmt.Errorf("invalid query type for temporal search")
	}

	var results []uuid.UUID
	for timestamp, ids := range ti.timeline {
		if timestamp >= timeRange.StartTime && timestamp <= timeRange.EndTime {
			results = append(results, ids...)
		}
	}

	return results, nil
}

func (ti *TemporalIndex) Remove(ctx context.Context, blockID uuid.UUID) error {
	ti.mu.Lock()
	defer ti.mu.Unlock()

	for timestamp, ids := range ti.timeline {
		for i, id := range ids {
			if id == blockID {
				ti.timeline[timestamp] = append(ids[:i], ids[i+1:]...)
				break
			}
		}
	}

	return nil
}

func (ti *TemporalIndex) Rebuild(ctx context.Context) error {
	return nil
}

// CategoryIndex implements hash-based category filtering
type CategoryIndex struct {
	categories map[blockchain.MemoryCategory][]uuid.UUID
	mu         sync.RWMutex
}

func NewCategoryIndex() *CategoryIndex {
	return &CategoryIndex{
		categories: make(map[blockchain.MemoryCategory][]uuid.UUID),
	}
}

func (ci *CategoryIndex) Add(ctx context.Context, block *blockchain.Block) error {
	ci.mu.Lock()
	defer ci.mu.Unlock()

	ci.categories[block.Category] = append(ci.categories[block.Category], block.BlockID)
	return nil
}

func (ci *CategoryIndex) Search(ctx context.Context, query interface{}) ([]uuid.UUID, error) {
	ci.mu.RLock()
	defer ci.mu.RUnlock()

	category, ok := query.(blockchain.MemoryCategory)
	if !ok {
		return nil, fmt.Errorf("invalid query type for category search")
	}

	return ci.categories[category], nil
}

func (ci *CategoryIndex) Remove(ctx context.Context, blockID uuid.UUID) error {
	ci.mu.Lock()
	defer ci.mu.Unlock()

	for category, ids := range ci.categories {
		for i, id := range ids {
			if id == blockID {
				ci.categories[category] = append(ids[:i], ids[i+1:]...)
				break
			}
		}
	}

	return nil
}

func (ci *CategoryIndex) Rebuild(ctx context.Context) error {
	return nil
}
