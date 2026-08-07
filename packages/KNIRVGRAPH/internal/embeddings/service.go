package embeddings

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"sync"
	"time"

	"go.uber.org/zap"
)

type EmbeddingService struct {
	provider      Provider
	cache         map[uint64][]float32
	cacheTTL      time.Duration
	cacheExpiry   map[uint64]time.Time
	mu            sync.RWMutex
	batchSize     int
	logger        *zap.Logger
	metrics       *EmbeddingMetrics
}

type EmbeddingMetrics struct {
	TotalEmbeddings  int64
	CacheHits        int64
	CacheMisses      int64
	ProviderFailures int64
}

func NewEmbeddingService(config ProviderConfig, logger *zap.Logger) (*EmbeddingService, error) {
	provider, err := NewProvider(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding provider: %w", err)
	}
	batchSize := config.BatchSize
	if batchSize <= 0 {
		batchSize = 32
	}
	return &EmbeddingService{
		provider:    provider,
		cache:       make(map[uint64][]float32),
		cacheTTL:    1 * time.Hour,
		cacheExpiry: make(map[uint64]time.Time),
		batchSize:   batchSize,
		logger:      logger,
		metrics:     &EmbeddingMetrics{},
	}, nil
}

func (s *EmbeddingService) Embed(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return make([]float32, s.provider.Dimension()), nil
	}
	key := hashText(text)
	s.mu.RLock()
	if vec, ok := s.cache[key]; ok && time.Now().Before(s.cacheExpiry[key]) {
		s.mu.RUnlock()
		s.metrics.CacheHits++
		return vec, nil
	}
	s.mu.RUnlock()
	s.metrics.CacheMisses++
	vecs, err := s.provider.Embed(ctx, []string{text})
	if err != nil {
		s.metrics.ProviderFailures++
		return nil, err
	}
	if len(vecs) == 0 {
		return make([]float32, s.provider.Dimension()), nil
	}
	result := vecs[0]
	s.mu.Lock()
	s.cache[key] = result
	s.cacheExpiry[key] = time.Now().Add(s.cacheTTL)
	s.mu.Unlock()
	s.metrics.TotalEmbeddings++
	return result, nil
}

func (s *EmbeddingService) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	result := make([][]float32, len(texts))
	toEmbed := make([]string, 0, len(texts))
	indices := make([]int, 0, len(texts))
	s.mu.RLock()
	for i, text := range texts {
		if text == "" {
			result[i] = make([]float32, s.provider.Dimension())
			continue
		}
		key := hashText(text)
		if vec, ok := s.cache[key]; ok && time.Now().Before(s.cacheExpiry[key]) {
			result[i] = vec
			s.metrics.CacheHits++
			continue
		}
		toEmbed = append(toEmbed, text)
		indices = append(indices, i)
		s.metrics.CacheMisses++
	}
	s.mu.RUnlock()
	if len(toEmbed) == 0 {
		return result, nil
	}
	var allEmbeddings [][]float32
	for batchStart := 0; batchStart < len(toEmbed); batchStart += s.batchSize {
		batchEnd := batchStart + s.batchSize
		if batchEnd > len(toEmbed) {
			batchEnd = len(toEmbed)
		}
		batch := toEmbed[batchStart:batchEnd]
		embeddings, err := s.provider.Embed(ctx, batch)
		if err != nil {
			s.metrics.ProviderFailures++
			return nil, fmt.Errorf("batch embedding failed: %w", err)
		}
		allEmbeddings = append(allEmbeddings, embeddings...)
	}
	for k, idx := range indices {
		if k < len(allEmbeddings) {
			result[idx] = allEmbeddings[k]
			key := hashText(toEmbed[k])
			s.mu.Lock()
			s.cache[key] = allEmbeddings[k]
			s.cacheExpiry[key] = time.Now().Add(s.cacheTTL)
			s.mu.Unlock()
			s.metrics.TotalEmbeddings++
		}
	}
	return result, nil
}

func (s *EmbeddingService) Dimension() int {
	return s.provider.Dimension()
}

func (s *EmbeddingService) Health(ctx context.Context) error {
	return s.provider.Health(ctx)
}

func (s *EmbeddingService) Close() error {
	return s.provider.Close()
}

func (s *EmbeddingService) Metrics() EmbeddingMetrics {
	return *s.metrics
}

func hashText(text string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(text))
	return h.Sum64()
}

func (s *EmbeddingService) StoreVectorIndex(id string, vector []float32) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := hashText(id)
	s.cache[key] = vector
	s.cacheExpiry[key] = time.Now().Add(s.cacheTTL)
	return nil
}

func (s *EmbeddingService) MarshalMetrics() []byte {
	data, _ := json.Marshal(s.metrics)
	return data
}
