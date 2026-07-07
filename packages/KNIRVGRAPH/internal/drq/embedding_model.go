package drq

import (
	"bytes"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"math"
	"net/http"
	"sync"
	"time"
)

type EmbeddingCache struct {
	mu    sync.RWMutex
	store map[uint64][]float64
}

func NewEmbeddingCache() *EmbeddingCache {
	return &EmbeddingCache{store: make(map[uint64][]float64)}
}

func (ec *EmbeddingCache) Get(key uint64) ([]float64, bool) {
	ec.mu.RLock()
	v, ok := ec.store[key]
	ec.mu.RUnlock()
	return v, ok
}

func (ec *EmbeddingCache) Set(key uint64, embedding []float64) {
	ec.mu.Lock()
	ec.store[key] = embedding
	ec.mu.Unlock()
}

func (ec *EmbeddingCache) Len() int {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	return len(ec.store)
}

type EmbeddingType int

const (
	BERT_BASE    EmbeddingType = iota // 768-dim
	BERT_LARGE                        // 1024-dim
	OPENAI_SMALL                      // 1536-dim
)

type GraphRAGEmbedder struct {
	endpoint     string
	client       *http.Client
	cache        *EmbeddingCache
	dimension    int
	timeout      time.Duration
}

func NewGraphRAGEmbedder(endpoint string, dim int) *GraphRAGEmbedder {
	if endpoint == "" {
		endpoint = "http://localhost:8084" // default KNIRVSERVER
	}
	if dim <= 0 {
		dim = 128
	}
	return &GraphRAGEmbedder{
		endpoint: endpoint,
		client: &http.Client{Timeout: 30 * time.Second},
		cache:   NewEmbeddingCache(),
		dimension: dim,
		timeout:   30 * time.Second,
	}
}

func (e *GraphRAGEmbedder) Embed(text string) ([]float64, error) {
	embeddings, err := e.EmbedBatch([]string{text})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("empty embedding result")
	}
	return embeddings[0], nil
}

func (e *GraphRAGEmbedder) EmbedBatch(texts []string) ([][]float64, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	body, err := json.Marshal(texts)
	if err != nil {
		return nil, fmt.Errorf("embed marshal: %w", err)
	}

	req, err := http.NewRequest("POST", e.endpoint+"/oracle/embed", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("embed request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embed call: %w", err)
	}
	defer resp.Body.Close()

	var raw [][]float64
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("embed decode: %w", err)
	}

	return raw, nil
}

type EmbeddingModel struct {
	modelType  EmbeddingType
	dimensions int
	cache      *EmbeddingCache
	embedder   *GraphRAGEmbedder
}

func NewEmbeddingModel(modelType EmbeddingType, endpoint string) *EmbeddingModel {
	dim := dimsForType(modelType)
	return &EmbeddingModel{
		modelType:  modelType,
		dimensions: dim,
		cache:      NewEmbeddingCache(),
		embedder:   NewGraphRAGEmbedder(endpoint, dim),
	}
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

func (em *EmbeddingModel) Encode(
	failureContext []byte,
) []float64 {
	contextHash := hashBytes(failureContext)
	if cached, exists := em.cache.Get(contextHash); exists {
		return cached
	}

	embeddings, err := em.embedder.EmbedBatch([]string{string(failureContext)})
	if err != nil || len(embeddings) == 0 {
		// fallback: hash-based local embedding
		vec := hashEmbedding(failureContext, em.dimensions)
		em.cache.Set(contextHash, vec)
		return vec
	}

	vec := l2Normalize(embeddings[0])
	em.cache.Set(contextHash, vec)
	return vec
}

func CosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		panic("dimension mismatch")
	}
	dotProduct := 0.0
	normA := 0.0
	normB := 0.0
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

func hashBytes(data []byte) uint64 {
	h := fnv.New64a()
	h.Write(data)
	return h.Sum64()
}

// hashEmbedding generates a deterministic embedding from input bytes
// using hash-based feature hashing (no external deps).
func hashEmbedding(data []byte, dims int) []float64 {
	vec := make([]float64, dims)
	for i, b := range data {
		idx := i % dims
		// alternate sign based on bit to reduce bias
		if i%2 == 0 {
			vec[idx] += float64(b)
		} else {
			vec[idx] -= float64(b)
		}
	}
	return l2Normalize(vec)
}

func l2Normalize(vec []float64) []float64 {
	mag := 0.0
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
