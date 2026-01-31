package query

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"sync"

	"github.com/google/uuid"
	"github.com/knirvchain/internal/blockchain"
	"github.com/knirvchain/internal/cache"
	"github.com/knirvchain/internal/indexing"
)

type QueryOptimizer struct {
	indexManager *indexing.MultiIndexManager
	cache        *cache.MemoryCache
	chain        *blockchain.ChainNode
}

func NewQueryOptimizer(
	indexManager *indexing.MultiIndexManager,
	cache *cache.MemoryCache,
	chain *blockchain.ChainNode,
) *QueryOptimizer {
	return &QueryOptimizer{
		indexManager: indexManager,
		cache:        cache,
		chain:        chain,
	}
}

type QueryRequest struct {
	Vector            []float32
	Category          *blockchain.MemoryCategory
	TimeRange         *indexing.TimeRangeQuery
	Keywords          []string
	Limit             int
	IncludeDeprecated bool
}

type QueryResult struct {
	Block           *blockchain.Block
	SimilarityScore float64
	Rank            int
}

// OptimizedSearch performs a multi-index optimized search
func (qo *QueryOptimizer) OptimizedSearch(ctx context.Context, req QueryRequest) ([]QueryResult, error) {
	queryHash := qo.hashQuery(req)

	cachedIDs, err := qo.cache.GetQueryResult(ctx, queryHash)
	if err == nil && cachedIDs != nil {
		return qo.loadBlocks(ctx, cachedIDs, req.Limit)
	}

	var wg sync.WaitGroup
	resultChan := make(chan []uuid.UUID, 3)
	errChan := make(chan error, 3)

	wg.Add(1)
	go func() {
		defer wg.Done()
		ids, err := qo.semanticSearch(ctx, req.Vector, 100)
		if err != nil {
			errChan <- err
			return
		}
		resultChan <- ids
	}()

	if req.Category != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ids, err := qo.categorySearch(ctx, *req.Category)
			if err != nil {
				errChan <- err
				return
			}
			resultChan <- ids
		}()
	}

	if req.TimeRange != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ids, err := qo.temporalSearch(ctx, *req.TimeRange)
			if err != nil {
				errChan <- err
				return
			}
			resultChan <- ids
		}()
	}

	wg.Wait()
	close(resultChan)
	close(errChan)

	for err := range errChan {
		if err != nil {
			return nil, fmt.Errorf("search error: %w", err)
		}
	}

	candidateIDs := qo.intersectResults(resultChan)

	go qo.cache.SetQueryResult(ctx, queryHash, candidateIDs)

	return qo.rankAndLoad(ctx, candidateIDs, req)
}

func (qo *QueryOptimizer) semanticSearch(ctx context.Context, vector []float32, _ int) ([]uuid.UUID, error) {
	index := qo.indexManager.GetIndex(indexing.IndexTypeSemantic)
	if index == nil {
		return nil, fmt.Errorf("semantic index not available")
	}

	return index.Search(ctx, vector)
}

func (qo *QueryOptimizer) categorySearch(ctx context.Context, category blockchain.MemoryCategory) ([]uuid.UUID, error) {
	index := qo.indexManager.GetIndex(indexing.IndexTypeCategory)
	if index == nil {
		return nil, fmt.Errorf("category index not available")
	}

	return index.Search(ctx, category)
}

func (qo *QueryOptimizer) temporalSearch(ctx context.Context, timeRange indexing.TimeRangeQuery) ([]uuid.UUID, error) {
	index := qo.indexManager.GetIndex(indexing.IndexTypeTemporal)
	if index == nil {
		return nil, fmt.Errorf("temporal index not available")
	}

	return index.Search(ctx, timeRange)
}

func (qo *QueryOptimizer) intersectResults(resultChan <-chan []uuid.UUID) []uuid.UUID {
	var allResults [][]uuid.UUID
	for results := range resultChan {
		allResults = append(allResults, results)
	}

	if len(allResults) == 0 {
		return []uuid.UUID{}
	}

	if len(allResults) == 1 {
		return allResults[0]
	}

	counts := make(map[uuid.UUID]int)
	for _, results := range allResults {
		for _, id := range results {
			counts[id]++
		}
	}

	var intersection []uuid.UUID
	minCount := len(allResults)
	for id, count := range counts {
		if count == minCount {
			intersection = append(intersection, id)
		}
	}

	return intersection
}

func (qo *QueryOptimizer) rankAndLoad(ctx context.Context, candidateIDs []uuid.UUID, req QueryRequest) ([]QueryResult, error) {
	var wg sync.WaitGroup
	resultChan := make(chan QueryResult, len(candidateIDs))
	semaphore := make(chan struct{}, 10) // Limit concurrent loads

	for _, id := range candidateIDs {
		wg.Add(1)
		go func(blockID uuid.UUID) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			block, err := qo.loadBlock(ctx, blockID)
			if err != nil {
				return
			}

			similarity := qo.calculateSimilarity(req.Vector, block.SemanticVector)
			resultChan <- QueryResult{
				Block:           block,
				SimilarityScore: similarity,
			}
		}(id)
	}

	wg.Wait()
	close(resultChan)

	var results []QueryResult
	for result := range resultChan {
		results = append(results, result)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].SimilarityScore > results[j].SimilarityScore
	})

	if len(results) > req.Limit {
		results = results[:req.Limit]
	}

	for i := range results {
		results[i].Rank = i + 1
	}

	return results, nil
}

func (qo *QueryOptimizer) loadBlock(ctx context.Context, blockID uuid.UUID) (*blockchain.Block, error) {
	block, err := qo.cache.Get(ctx, blockID)
	if err == nil && block != nil {
		return block, nil
	}

	block, err = qo.chain.GetBlock(ctx, blockID)
	if err != nil {
		return nil, err
	}

	go qo.cache.Set(context.Background(), block)

	return block, nil
}

func (qo *QueryOptimizer) loadBlocks(ctx context.Context, ids []uuid.UUID, limit int) ([]QueryResult, error) {
	if len(ids) > limit {
		ids = ids[:limit]
	}

	results := make([]QueryResult, 0, len(ids))
	for i, id := range ids {
		block, err := qo.loadBlock(ctx, id)
		if err != nil {
			continue
		}

		results = append(results, QueryResult{
			Block: block,
			Rank:  i + 1,
		})
	}

	return results, nil
}

func (qo *QueryOptimizer) calculateSimilarity(v1, v2 []float32) float64 {
	if len(v1) != len(v2) {
		return 0.0
	}

	var dot, norm1, norm2 float64
	for i := range v1 {
		dot += float64(v1[i] * v2[i])
		norm1 += float64(v1[i] * v1[i])
		norm2 += float64(v2[i] * v2[i])
	}

	if norm1 == 0 || norm2 == 0 {
		return 0.0
	}

	return dot / (sqrt(norm1) * sqrt(norm2))
}

func (qo *QueryOptimizer) hashQuery(req QueryRequest) string {
	h := sha256.New()
	fmt.Fprintf(h, "%v", req)
	return hex.EncodeToString(h.Sum(nil))
}

func sqrt(x float64) float64 {
	if x < 0 {
		return 0
	}
	z := 1.0
	for i := 0; i < 10; i++ {
		z -= (z*z - x) / (2 * z)
	}
	return z
}
