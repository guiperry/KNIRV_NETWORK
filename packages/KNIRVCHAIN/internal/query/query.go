package query

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"sort"
	"sync"

	"KNIRVCHAIN/internal/blockchain"
	"KNIRVCHAIN/internal/cache"
	"KNIRVCHAIN/internal/indexing"
	"github.com/google/uuid"
)

type QueryOptimizer struct {
	indexManager *indexing.MultiIndexManager
	cache        *cache.MemoryCache
	chain        BlockchainReader
}

type BlockchainReader interface {
	GetBlock(ctx context.Context, blockIDOrHeight string) (*blockchain.Block, error)
	GetTransaction(ctx context.Context, txHash string) (*blockchain.Transaction, error)
}

func NewQueryOptimizer(
	indexManager *indexing.MultiIndexManager,
	cache *cache.MemoryCache,
	chain BlockchainReader,
) *QueryOptimizer {
	return &QueryOptimizer{
		indexManager: indexManager,
		cache:        cache,
		chain:        chain,
	}
}

type QueryRequest struct {
	Vector            []float32
	Category          *string
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
	if len(vector) == 0 {
		return []uuid.UUID{}, nil
	}
	if qo.indexManager == nil {
		return []uuid.UUID{}, nil
	}
	index := qo.indexManager.GetIndex(indexing.IndexTypeSemantic)
	if index == nil {
		return []uuid.UUID{}, nil
	}

	return index.Search(ctx, vector)
}

func (qo *QueryOptimizer) categorySearch(ctx context.Context, category string) ([]uuid.UUID, error) {
	if qo.indexManager == nil {
		return []uuid.UUID{}, nil
	}
	index := qo.indexManager.GetIndex(indexing.IndexTypeCategory)
	if index == nil {
		return []uuid.UUID{}, nil
	}

	return index.Search(ctx, category)
}

func (qo *QueryOptimizer) temporalSearch(ctx context.Context, timeRange indexing.TimeRangeQuery) ([]uuid.UUID, error) {
	if qo.indexManager == nil {
		return []uuid.UUID{}, nil
	}
	index := qo.indexManager.GetIndex(indexing.IndexTypeTemporal)
	if index == nil {
		return []uuid.UUID{}, nil
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
		if len(allResults[0]) == 0 {
			return []uuid.UUID{}
		}
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
	if len(candidateIDs) == 0 {
		return []QueryResult{}, nil
	}

	var wg sync.WaitGroup
	resultChan := make(chan QueryResult, len(candidateIDs))
	semaphore := make(chan struct{}, 10)

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

			resultChan <- QueryResult{
				Block:           block,
				SimilarityScore: 0,
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
	if qo.chain == nil {
		return nil, fmt.Errorf("no chain configured")
	}
	return qo.chain.GetBlock(ctx, blockID.String())
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

	return dot / (math.Sqrt(norm1) * math.Sqrt(norm2))
}

func (qo *QueryOptimizer) hashQuery(req QueryRequest) string {
	h := sha256.New()
	fmt.Fprintf(h, "%v", req)
	return hex.EncodeToString(h.Sum(nil))
}
