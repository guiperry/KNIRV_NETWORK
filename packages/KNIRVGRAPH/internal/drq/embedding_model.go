package drq

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"math"
	"net"
	"net/http"
	"os"
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

// defaultGraphRAGSocketPath is the last-resort fallback KNIRVSERVER itself
// falls back to when it cannot resolve its app data directory (see
// embeddedChainSocketPath in KNIRV_NETWORK's packages/KNIRVSERVER/main.go).
const defaultGraphRAGSocketPath = "/var/lib/knirvserver/sockets/graphrag.sock"

// GraphRAGEmbedder dials the Unix domain socket KNIRVSERVER's embedded
// graphrag-rs engine bridges over (pkg/embedded/graphrag/server.go,
// KNIRV_NETWORK). KNIRVGRAPH runs as a separate OS process from
// KNIRVSERVER, so — like backend_server (KNIRV_CORP) — this socket is the
// only way to reach that CGo-linked engine; there is no HTTP route exposed
// over TCP for it.
type GraphRAGEmbedder struct {
	socketPath string
	client     *http.Client
	cache      *EmbeddingCache
	dimension  int
	timeout    time.Duration
}

// unixHTTPClient builds an http.Client that dials a Unix domain socket
// regardless of the URL host given to it — callers use the "http://unix"
// base URL by convention (matching KNIRVSERVER's pkg/embedded/graphrag
// Client).
func unixHTTPClient(socketPath string, timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
			},
		},
	}
}

// NewGraphRAGEmbedder builds an embedder dialing socketPath. An empty
// socketPath falls back to the KNIRV_GRAPHRAG_SOCKET_PATH env var
// (KNIRVSERVER honors the same variable), then the shared well-known
// default, so both sides agree on a path without an explicit override in
// most deployments.
func NewGraphRAGEmbedder(socketPath string, dim int) *GraphRAGEmbedder {
	if socketPath == "" {
		socketPath = os.Getenv("KNIRV_GRAPHRAG_SOCKET_PATH")
	}
	if socketPath == "" {
		socketPath = defaultGraphRAGSocketPath
	}
	if dim <= 0 {
		dim = 128
	}
	timeout := 30 * time.Second
	return &GraphRAGEmbedder{
		socketPath: socketPath,
		client:     unixHTTPClient(socketPath, timeout),
		cache:      NewEmbeddingCache(),
		dimension:  dim,
		timeout:    timeout,
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

	body, err := json.Marshal(map[string]interface{}{"texts": texts})
	if err != nil {
		return nil, fmt.Errorf("embed marshal: %w", err)
	}

	req, err := http.NewRequest("POST", "http://unix/embed", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("embed request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embed call to graphrag socket %s: %w", e.socketPath, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embed call to graphrag socket %s returned %d", e.socketPath, resp.StatusCode)
	}

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
