package embedder

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	textembed "github.com/guiperry/text-embedder/pkg/embed"
)

// EmbeddingProvider is the interface that all embedding providers must implement
type EmbeddingProvider interface {
	// GetEmbedding returns the vector together with the exact embedding-space
	// identity that produced it. Callers must persist/check the returned id,
	// not infer it from mutable provider configuration.
	GetEmbedding(text string) (vec []float32, embedderID string, err error)
	GetBatchEmbeddings(texts []string) (vecs [][]float32, embedderID string, err error)
	GetProviderStats() map[string]interface{}
	// EmbedderID names the model behind every vector this provider returns.
	// Vectors with different ids must never share an index.
	EmbedderID() string
}

// DeterministicProvider provides deterministic local embeddings using text-embedder
type DeterministicProvider struct {
	model      string
	dimensions int
}

// NewDeterministicProvider creates a new deterministic embedding provider
func NewDeterministicProvider() *DeterministicProvider {
	return &DeterministicProvider{
		model:      textembed.ModelID,
		dimensions: textembed.Dims,
	}
}

// GetEmbedding gets embedding using deterministic text-embedder
func (p *DeterministicProvider) GetEmbedding(text string) ([]float32, string, error) {
	cleanText := strings.TrimSpace(text)
	if len(cleanText) == 0 {
		return nil, "", fmt.Errorf("empty text provided")
	}
	return textembed.Embed(cleanText), p.EmbedderID(), nil
}

// GetBatchEmbeddings gets embeddings for multiple texts
func (p *DeterministicProvider) GetBatchEmbeddings(texts []string) ([][]float32, string, error) {
	embeddings := make([][]float32, len(texts))
	for i, text := range texts {
		cleanText := strings.TrimSpace(text)
		if len(cleanText) == 0 {
			embeddings[i] = make([]float32, textembed.Dims)
			continue
		}
		embeddings[i] = textembed.Embed(cleanText)
	}
	return embeddings, p.EmbedderID(), nil
}

// EmbedderID describes the complete embedding space.  The model id is also
// the revision for the bundled deterministic model: it is shipped as one
// immutable implementation rather than selected from a mutable remote tag.
func (p *DeterministicProvider) EmbedderID() string {
	return embeddingSpaceID("deterministic", p.model, p.model, p.dimensions, "plain")
}

// GetProviderStats returns current provider statistics
func (p *DeterministicProvider) GetProviderStats() map[string]interface{} {
	return map[string]interface{}{
		"backend":           "deterministic",
		"model":             p.model,
		"dimensions":        p.dimensions,
		"reproducible":      true,
		"requires_internet": false,
	}
}

// HybridEmbeddingProvider provides both Cloudflare and Ollama embeddings
type HybridEmbeddingProvider struct {
	CloudflareURL  string
	OllamaURL      string
	Model          string
	HTTPClient     *http.Client
	RequestTracker *RequestTracker
	UseCloudflare  bool
	dimMu          sync.RWMutex
	dimensions     int
}

// NewHybridEmbeddingProvider creates a new hybrid embeddings provider
func NewHybridEmbeddingProvider(cloudflareURL, ollamaURL, model string, maxDailyRequests int) *HybridEmbeddingProvider {
	return &HybridEmbeddingProvider{
		CloudflareURL: cloudflareURL,
		OllamaURL:     ollamaURL,
		Model:         model,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		RequestTracker: NewRequestTracker(maxDailyRequests),
		UseCloudflare:  true,
		dimensions:     768, // BGE Base is pinned to 768 dimensions.
	}
}

// NewEmbeddingProvider creates an embedding provider based on the backend config
func NewEmbeddingProvider(backend, cloudflareURL, ollamaURL, model string, maxDailyRequests int) EmbeddingProvider {
	switch backend {
	case "deterministic":
		log.Println("Using deterministic (local) embeddings")
		return NewDeterministicProvider()
	case "cloudflare":
		log.Println("Using Cloudflare Workers AI embeddings")
		return NewHybridEmbeddingProvider(cloudflareURL, ollamaURL, model, maxDailyRequests)
	case "ollama":
		log.Println("Using Ollama embeddings")
		provider := NewHybridEmbeddingProvider(cloudflareURL, ollamaURL, model, maxDailyRequests)
		provider.UseCloudflare = false
		provider.dimensions = 0 // Learn the Ollama dimensionality from its first vector.
		return provider
	default:
		log.Printf("Unknown EMBEDDING_BACKEND: %s, defaulting to deterministic", backend)
		return NewDeterministicProvider()
	}
}

// CloudflareEmbeddingRequest represents a request to Cloudflare API
type CloudflareEmbeddingRequest struct {
	Text string `json:"text"`
}

// CloudflareEmbeddingResponse represents response from Cloudflare API
type CloudflareEmbeddingResponse struct {
	Success   bool      `json:"success"`
	Timestamp string    `json:"timestamp"`
	Model     string    `json:"model"`
	Embedding []float32 `json:"embedding"`
}

// RequestTracker tracks daily API usage
type RequestTracker struct {
	mu          sync.RWMutex
	requests    int
	lastReset   time.Time
	maxRequests int
}

// NewRequestTracker creates a new request tracker
func NewRequestTracker(maxDaily int) *RequestTracker {
	now := time.Now()
	return &RequestTracker{
		requests:    0,
		lastReset:   time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()),
		maxRequests: maxDaily,
	}
}

// NewRequestTrackerWithCount creates a new request tracker with an initial count
func NewRequestTrackerWithCount(maxDaily, initialCount int) *RequestTracker {
	now := time.Now()
	return &RequestTracker{
		requests:    initialCount,
		lastReset:   time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()),
		maxRequests: maxDaily,
	}
}

// SetRequests manually sets the request count
func (rt *RequestTracker) SetRequests(count int) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.requests = count
}

// CanMakeRequest checks if we can make a request
func (rt *RequestTracker) CanMakeRequest() bool {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	if today.After(rt.lastReset) {
		rt.requests = 0
		rt.lastReset = today
	}
	return rt.requests < rt.maxRequests
}

// IncrementRequest increments the request counter
func (rt *RequestTracker) IncrementRequest() {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.requests++
}

// GetStats returns current usage statistics
func (rt *RequestTracker) GetStats() (int, int, time.Time) {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	return rt.requests, rt.maxRequests, rt.lastReset
}

// ErrQuotaExhausted is returned in Cloudflare mode once the daily cap is hit.
// The caller stops or waits; it must not be served from a different model
// (E1: BGE and nomic vectors are both 768-d but live in different spaces).
var ErrQuotaExhausted = fmt.Errorf("cloudflare embedding quota exhausted")

// EmbedderID names the complete embedding space that produces this provider's
// vectors: <backend>:<model>@<digest|revision>:<dims>:<prefix scheme>. Model
// tags are used as revisions for Ollama; deployments must use an immutable
// tag or digest when rebuilding an index. Cloudflare exposes a pinned model
// name, which is its revision for this API contract.
func (p *HybridEmbeddingProvider) EmbedderID() string {
	if p.UseCloudflare {
		return embeddingSpaceID("cloudflare", CloudflareModelName, CloudflareModelName, p.Dimensions(), "plain")
	}
	return embeddingSpaceID("ollama", p.Model, p.Model, p.Dimensions(), "plain")
}

// Dimensions returns the declared embedding width. Ollama starts unknown and
// learns this only after the first successful response; returned vectors are
// always verified before their ID is emitted.
func (p *HybridEmbeddingProvider) Dimensions() int {
	p.dimMu.RLock()
	defer p.dimMu.RUnlock()
	return p.dimensions
}

func embeddingSpaceID(backend, model, revision string, dimensions int, prefixScheme string) string {
	return strings.Join([]string{
		backend,
		canonicalIDPart(model) + "@" + canonicalIDPart(revision),
		strconv.Itoa(dimensions),
		canonicalIDPart(prefixScheme),
	}, ":")
}

// Model tags commonly contain ':' (for example, "nomic-embed-text:latest").
// Escape delimiters so IDs remain unambiguous and therefore safe as index
// keys. This is deliberately a minimal, stable percent encoding rather than
// URL encoding, whose treatment of '/' is not relevant to this identifier.
func canonicalIDPart(value string) string {
	return strings.NewReplacer("%", "%25", ":", "%3A", "@", "%40").Replace(value)
}

// verifyDimensions rejects a changed vector space before it reaches a batch.
// Ollama does not announce its output dimensions, so the first successful
// response establishes the per-run contract.
func (p *HybridEmbeddingProvider) verifyDimensions(vec []float32) error {
	if len(vec) == 0 {
		return fmt.Errorf("embedding is empty")
	}
	p.dimMu.Lock()
	defer p.dimMu.Unlock()
	if p.dimensions == 0 {
		p.dimensions = len(vec)
		return nil
	}
	if len(vec) != p.dimensions {
		return fmt.Errorf("embedding dimensions changed: got %d, expected %d", len(vec), p.dimensions)
	}
	return nil
}

// CloudflareModelName is the Workers AI model behind embeddings.knirv.com
// (BGE-Base, 768-d; see MATHASHER_Implementation.md).
const CloudflareModelName = "bge-base-en-v1.5"

// GetEmbedding embeds with the one backend this provider was built for. It
// never falls back to the other backend: a failure or an exhausted quota is an
// error.
func (p *HybridEmbeddingProvider) GetEmbedding(text string) ([]float32, string, error) {
	cleanText := strings.TrimSpace(text)
	if len(cleanText) == 0 {
		return nil, "", fmt.Errorf("empty text provided")
	}

	var embedding []float32
	var err error
	if !p.UseCloudflare {
		embedding, err = p.getOllamaEmbedding(cleanText)
		if err != nil {
			return nil, "", err
		}
		if err := p.verifyDimensions(embedding); err != nil {
			return nil, "", err
		}
		return embedding, p.EmbedderID(), nil
	}

	if !p.RequestTracker.CanMakeRequest() {
		used, max, _ := p.RequestTracker.GetStats()
		return nil, "", fmt.Errorf("%w (%d/%d)", ErrQuotaExhausted, used, max)
	}
	embedding, err = p.getCloudflareEmbedding(cleanText)
	if err != nil {
		return nil, "", fmt.Errorf("cloudflare embedding failed (no fallback to another model): %w", err)
	}
	if err := p.verifyDimensions(embedding); err != nil {
		return nil, "", err
	}
	p.RequestTracker.IncrementRequest()
	return embedding, p.EmbedderID(), nil
}

func (p *HybridEmbeddingProvider) getCloudflareEmbedding(text string) ([]float32, error) {
	req := CloudflareEmbeddingRequest{
		Text: text,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Cloudflare request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", p.CloudflareURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create Cloudflare request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to make Cloudflare request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Cloudflare API returned status %d: %s", resp.StatusCode, string(body))
	}

	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/html") || (len(body) > 0 && body[0] == '<') {
		preview := string(body)
		if len(preview) > 300 {
			preview = preview[:300] + "..."
		}
		return nil, fmt.Errorf("Cloudflare worker returned HTML instead of JSON: %s", preview)
	}

	var cloudflareResp CloudflareEmbeddingResponse
	if err := json.Unmarshal(body, &cloudflareResp); err != nil {
		return nil, fmt.Errorf("failed to decode Cloudflare response: %w", err)
	}

	if !cloudflareResp.Success {
		return nil, fmt.Errorf("Cloudflare API returned unsuccessful response")
	}

	if len(cloudflareResp.Embedding) == 0 {
		return nil, fmt.Errorf("Cloudflare API returned empty embedding")
	}

	return cloudflareResp.Embedding, nil
}

func (p *HybridEmbeddingProvider) getOllamaEmbedding(text string) ([]float32, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	req := OllamaEmbeddingRequest{
		Model:  p.Model,
		Prompt: text,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Ollama request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.OllamaURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create Ollama request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	ollamaClient := &http.Client{
		Timeout: 90 * time.Second,
	}

	var ollamaResp OllamaEmbeddingResponse
	var lastErr error

	for retry := 0; retry < 3; retry++ {
		if retry > 0 {
			backoffDuration := time.Duration(retry) * 2 * time.Second
			log.Printf("Ollama request retry %d, waiting %v...", retry+1, backoffDuration)
			time.Sleep(backoffDuration)
		}

		resp, err := ollamaClient.Do(httpReq)
		if err != nil {
			lastErr = err
			log.Printf("Ollama request failed (attempt %d): %v", retry+1, err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			lastErr = fmt.Errorf("Ollama returned status %d", resp.StatusCode)
			log.Printf("Ollama request failed (attempt %d): HTTP %d", retry+1, resp.StatusCode)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			log.Printf("Failed to read Ollama response (attempt %d): %v", retry+1, err)
			continue
		}

		if err := json.Unmarshal(body, &ollamaResp); err != nil {
			lastErr = err
			log.Printf("Failed to decode Ollama response (attempt %d): %v", retry+1, err)
			continue
		}

		if len(ollamaResp.Embedding) == 0 {
			lastErr = fmt.Errorf("received empty embedding")
			log.Printf("Empty Ollama embedding (attempt %d)", retry+1)
			continue
		}

		lastErr = nil
		break
	}

	if lastErr != nil {
		return nil, fmt.Errorf("failed to get Ollama embedding after 3 attempts: %w", lastErr)
	}

	log.Printf("Used Ollama embedding")
	return ollamaResp.Embedding, nil
}

// GetBatchEmbeddings gets embeddings for multiple texts
func (p *HybridEmbeddingProvider) GetBatchEmbeddings(texts []string) ([][]float32, string, error) {
	embeddings := make([][]float32, len(texts))
	var embedderID string

	for i, text := range texts {
		embedding, id, err := p.GetEmbedding(text)
		if err != nil {
			return nil, "", fmt.Errorf("failed to get embedding for text %d: %w", i, err)
		}
		if embedderID != "" && embedderID != id {
			return nil, "", fmt.Errorf("mixed embedding spaces in batch: %q then %q", embedderID, id)
		}
		embedderID = id
		embeddings[i] = embedding
	}

	return embeddings, embedderID, nil
}

// GetProviderStats returns current provider statistics
func (p *HybridEmbeddingProvider) GetProviderStats() map[string]interface{} {
	used, max, lastReset := p.RequestTracker.GetStats()
	return map[string]interface{}{
		"cloudflare_enabled":    p.UseCloudflare,
		"cloudflare_quota_used": used,
		"cloudflare_quota_max":  max,
		"cloudflare_last_reset": lastReset.Format("2006-01-02"),
		"remaining_quota":       max - used,
		"embedder_id":           p.EmbedderID(),
	}
}

// OllamaEmbeddingRequest represents a request to Ollama API
type OllamaEmbeddingRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// OllamaEmbeddingResponse represents response from Ollama API
type OllamaEmbeddingResponse struct {
	Embedding []float32 `json:"embedding"`
}
