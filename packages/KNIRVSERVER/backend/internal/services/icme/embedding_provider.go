package icme

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/guiperry/text-embedder/pkg/embed"
	"go.uber.org/zap"
)

type EmbeddingProvider struct {
	CloudflareURL string
	OllamaURL     string
	Model         string
	HTTPClient    *http.Client
	UseCloudflare bool
	mu            sync.RWMutex
	logger        *zap.Logger
	cache         sync.Map // key: sha256(text) -> []float32
}

type CloudflareRequest struct {
	Text string `json:"text"`
}

type CloudflareResponse struct {
	Success   bool      `json:"success"`
	Embedding []float32 `json:"embedding"`
}

type OllamaEmbeddingRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type OllamaEmbeddingResponse struct {
	Embedding []float32 `json:"embedding"`
}

func NewEmbeddingProvider(logger *zap.Logger) (*EmbeddingProvider, error) {
	cloudflareURL := os.Getenv("CLOUDFLARE_EMBEDDINGS_URL")
	if cloudflareURL == "" {
		cloudflareURL = "https://embeddings.knirv.com"
	}

	ollamaURL := os.Getenv("OLLAMA_BASE_URL")
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}

	model := os.Getenv("OLLAMA_EMBEDDING_MODEL")
	if model == "" {
		model = "nomic-embed-text"
	}

	provider := &EmbeddingProvider{
		CloudflareURL: cloudflareURL,
		OllamaURL:     ollamaURL,
		Model:         model,
		HTTPClient:    &http.Client{Timeout: 30 * time.Second},
		UseCloudflare: true,
		logger:        logger,
	}

	return provider, nil
}

// GetEmbedding returns an embedding vector for text.
// Primary: local deterministic hash-ngram embedder (github.com/guiperry/text-embedder).
// Fallback 1: Cloudflare embeddings endpoint.
// Fallback 2: Ollama local model.
// Uses sync.Map cache for repeated texts.
func (p *EmbeddingProvider) GetEmbedding(text string) ([]float32, error) {
	cleanText := strings.TrimSpace(text)
	if len(cleanText) == 0 {
		return nil, fmt.Errorf("empty text provided")
	}

	key := cacheKey(cleanText)
	if cached, ok := p.cache.Load(key); ok {
		return cached.([]float32), nil
	}

	// Primary: local text-embedder (deterministic, no network, always available)
	if vec := embed.Embed(cleanText); isNonZero(vec) {
		p.cache.Store(key, vec)
		return vec, nil
	}
	p.logger.Warn("text-embedder returned zero vector, falling back to cloudflare")

	p.mu.RLock()
	useCF := p.UseCloudflare
	p.mu.RUnlock()

	// Fallback 1: Cloudflare
	if useCF {
		embedding, err := p.getCloudflareEmbedding(cleanText)
		if err == nil {
			p.cache.Store(key, embedding)
			return embedding, nil
		}
		p.logger.Warn("cloudflare failed, falling back to ollama", zap.Error(err))
	}

	// Fallback 2: Ollama
	embedding, err := p.getOllamaEmbedding(cleanText)
	if err == nil {
		p.cache.Store(key, embedding)
	}
	return embedding, err
}

func cacheKey(text string) string {
	h := sha256.Sum256([]byte(text))
	return hex.EncodeToString(h[:])
}

// isNonZero reports whether the vector contains at least one non-zero element.
func isNonZero(v []float32) bool {
	for _, x := range v {
		if x != 0 {
			return true
		}
	}
	return false
}

func (p *EmbeddingProvider) getCloudflareEmbedding(text string) ([]float32, error) {
	req := CloudflareRequest{Text: text}
	jsonData, _ := json.Marshal(req)

	httpReq, _ := http.NewRequest("POST", p.CloudflareURL, bytes.NewBuffer(jsonData))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("cloudflare request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("cloudflare returned %d: %s", resp.StatusCode, string(body))
	}

	var cfResp CloudflareResponse
	if err := json.NewDecoder(resp.Body).Decode(&cfResp); err != nil {
		return nil, fmt.Errorf("decode failed: %w", err)
	}

	if !cfResp.Success || len(cfResp.Embedding) == 0 {
		return nil, fmt.Errorf("empty embedding")
	}

	return cfResp.Embedding, nil
}

func (p *EmbeddingProvider) getOllamaEmbedding(text string) ([]float32, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	req := OllamaEmbeddingRequest{Model: p.Model, Prompt: text}
	jsonData, _ := json.Marshal(req)

	httpReq, _ := http.NewRequestWithContext(ctx, "POST", p.OllamaURL+"/api/embeddings", bytes.NewBuffer(jsonData))
	httpReq.Header.Set("Content-Type", "application/json")

	ollamaClient := &http.Client{Timeout: 90 * time.Second}
	resp, err := ollamaClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama returned %d: %s", resp.StatusCode, string(body))
	}

	var ollamaResp OllamaEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("decode failed: %w", err)
	}

	if len(ollamaResp.Embedding) == 0 {
		return nil, fmt.Errorf("empty embedding")
	}

	return ollamaResp.Embedding, nil
}
