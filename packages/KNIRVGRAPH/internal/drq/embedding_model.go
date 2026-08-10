package drq

import (
	"KNIRVGRAPH/internal/embeddings"
	"KNIRVGRAPH/internal/types"
	"context"
	"hash/fnv"
	"math"
	"sync"

	"go.uber.org/zap"
)

type EmbeddingCache struct {
	mu    sync.RWMutex
	store map[uint64][]float64
}

func NewEmbeddingCache() *EmbeddingCache { return &EmbeddingCache{store: make(map[uint64][]float64)} }
func (ec *EmbeddingCache) Get(key uint64) ([]float64, bool) {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	v, ok := ec.store[key]
	return append([]float64(nil), v...), ok
}
func (ec *EmbeddingCache) Set(key uint64, embedding []float64) {
	ec.mu.Lock()
	ec.store[key] = append([]float64(nil), embedding...)
	ec.mu.Unlock()
}
func (ec *EmbeddingCache) Len() int { ec.mu.RLock(); defer ec.mu.RUnlock(); return len(ec.store) }

type EmbeddingType int

const (
	BERT_BASE EmbeddingType = iota
	BERT_LARGE
	OPENAI_SMALL
)

// EmbeddingModel adapts KNIRVGRAPH's shared EmbeddingService to DRQ's legacy
// float64 clustering API. It no longer dials KNIRVSERVER or GraphRAG.
type EmbeddingModel struct {
	modelType  EmbeddingType
	dimensions int
	cache      *EmbeddingCache
	service    *embeddings.EmbeddingService
}

func NewEmbeddingModel(modelType EmbeddingType, endpoint string) *EmbeddingModel {
	dim := dimsForType(modelType)
	cfg := embeddings.ProviderConfig{Type: types.EmbeddingProviderTextEmbedder, Endpoint: endpoint, Model: "text-embedder", Dimension: dim, BatchSize: 32, TimeoutSeconds: 30}
	if endpoint == "" {
		cfg = embeddings.DefaultProviderConfig(types.EmbeddingProviderTextEmbedder)
		dim = cfg.Dimension
	}
	svc, _ := embeddings.NewEmbeddingService(cfg, zap.NewNop())
	return NewEmbeddingModelWithService(modelType, svc)
}

func NewEmbeddingModelWithService(modelType EmbeddingType, service *embeddings.EmbeddingService) *EmbeddingModel {
	dim := dimsForType(modelType)
	if service != nil {
		dim = service.Dimension()
	}
	return &EmbeddingModel{modelType: modelType, dimensions: dim, cache: NewEmbeddingCache(), service: service}
}
func dimsForType(t EmbeddingType) int {
	switch t {
	case BERT_BASE:
		return 768
	case BERT_LARGE:
		return 1024
	case OPENAI_SMALL:
		return 1536
	default:
		return 128
	}
}

func (em *EmbeddingModel) Encode(failureContext []byte) []float64 {
	vec, _ := em.EncodeContext(context.Background(), failureContext)
	return vec
}
func (em *EmbeddingModel) EncodeContext(ctx context.Context, failureContext []byte) ([]float64, error) {
	h := hashBytes(failureContext)
	if em.cache == nil {
		em.cache = NewEmbeddingCache()
	}
	if cached, ok := em.cache.Get(h); ok {
		return cached, nil
	}
	if em.service == nil {
		vec := hashEmbedding(failureContext, em.dimensions)
		em.cache.Set(h, vec)
		return vec, nil
	}
	raw, err := em.service.Embed(ctx, string(failureContext))
	if err != nil {
		return nil, err
	}
	vec := make([]float64, len(raw))
	for i, v := range raw {
		vec[i] = float64(v)
	}
	vec = l2Normalize(vec)
	em.cache.Set(h, vec)
	return vec, nil
}

func CosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		panic("dimension mismatch")
	}
	var dot, na, nb float64
	for i := range a {
		dot += a[i] * b[i]
		na += a[i] * a[i]
		nb += b[i] * b[i]
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}
func hashBytes(data []byte) uint64 { h := fnv.New64a(); _, _ = h.Write(data); return h.Sum64() }
func hashEmbedding(data []byte, dims int) []float64 {
	vec := make([]float64, dims)
	for i, b := range data {
		idx := i % dims
		if i%2 == 0 {
			vec[idx] += float64(b)
		} else {
			vec[idx] -= float64(b)
		}
	}
	return l2Normalize(vec)
}
func l2Normalize(vec []float64) []float64 {
	var mag float64
	for _, v := range vec {
		mag += v * v
	}
	if mag == 0 {
		return vec
	}
	mag = math.Sqrt(mag)
	out := make([]float64, len(vec))
	for i, v := range vec {
		out[i] = v / mag
	}
	return out
}
