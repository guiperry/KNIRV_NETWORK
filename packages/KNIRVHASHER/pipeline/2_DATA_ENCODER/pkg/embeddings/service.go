package embeddings

import (
	"bytes"
	"data-encoder/pkg/config"
	"data-encoder/pkg/ollama"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const (
	// DefaultBatchSize is the maximum number of texts to send in one API call
	DefaultBatchSize = 32
	// MaxRetries is the maximum number of retry attempts for failed API calls
	MaxRetries = 3
	// CloudflareWorkersBGE is the BGE-Base model identifier for Cloudflare Workers AI
	CloudflareWorkersBGE = "@cf/baai/bge-base-en-v1.5"
)

// EmbeddingService is the interface that all embedding services must implement
type EmbeddingService interface {
	GetEmbedding(text string) ([]float32, error)
	GetBatchEmbeddings(texts []string) ([][]float32, error)
	GetBatchSize() int
	ValidateEndpoint() error
	SetTimeout(timeout interface{})
}

// Service handles embedding generation via Cloudflare Workers AI API or Ollama
type Service struct {
	baseURL     string
	httpClient  *http.Client
	batchSize   int
	model       string
	ollamaHost  string
	ollamaModel string
}

// CloudflareWorkersRequest represents the Cloudflare Workers AI API request structure
type CloudflareWorkersRequest struct {
	Texts []string `json:"texts"`
}

// CloudflareWorkersResponse represents the Cloudflare Workers AI API response structure
type CloudflareWorkersResponse struct {
	Success    bool        `json:"success"`
	Timestamp  string      `json:"timestamp"`
	Model      string      `json:"model"`
	Embeddings [][]float32 `json:"embeddings"`
	Count      int         `json:"count"`
}

// New creates a new embeddings service based on EMBEDDING_BACKEND config
func New() EmbeddingService {
	config.LoadEnv()
	backend := config.GetEmbeddingBackend()

	switch backend {
	case "deterministic":
		log.Println("Using deterministic (local) embeddings")
		return NewDeterministicService()
	case "cloudflare":
		log.Println("Using Cloudflare Workers AI embeddings")
		return newCloudflareService()
	case "ollama":
		log.Println("Using Ollama embeddings")
		return newOllamaService()
	default:
		log.Printf("Unknown EMBEDDING_BACKEND: %s, defaulting to deterministic", backend)
		return NewDeterministicService()
	}
}

// NewWithBatchSize creates a new embeddings service with custom batch size
func NewWithBatchSize(batchSize int) EmbeddingService {
	config.LoadEnv()
	backend := config.GetEmbeddingBackend()

	switch backend {
	case "deterministic":
		svc := NewDeterministicService()
		svc.batchSize = batchSize
		return svc
	case "cloudflare":
		return newCloudflareServiceWithBatchSize(batchSize)
	case "ollama":
		return newOllamaServiceWithBatchSize(batchSize)
	default:
		svc := NewDeterministicService()
		svc.batchSize = batchSize
		return svc
	}
}

func newCloudflareService() *Service {
	endpoint := config.GetCloudflareEndpoint()
	svc := &Service{
		baseURL:     endpoint,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		batchSize:   DefaultBatchSize,
		model:       CloudflareWorkersBGE,
		ollamaHost:  "http://localhost:11434/api/embeddings",
		ollamaModel: "nomic-embed-text",
	}

	if endpoint != "" {
		if err := svc.ValidateEndpoint(); err != nil {
			log.Fatalf("❌ Embeddings endpoint validation failed: %v", err)
		}
		log.Printf("✅ Embeddings endpoint validated: %s", endpoint)
	}

	return svc
}

func newOllamaService() *Service {
	return &Service{
		baseURL:     "",
		httpClient:  &http.Client{Timeout: 120 * time.Second},
		batchSize:   DefaultBatchSize,
		model:       CloudflareWorkersBGE,
		ollamaHost:  "http://localhost:11434/api/embeddings",
		ollamaModel: "nomic-embed-text",
	}
}

func newCloudflareServiceWithBatchSize(batchSize int) *Service {
	endpoint := config.GetCloudflareEndpoint()
	svc := &Service{
		baseURL:     endpoint,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		batchSize:   batchSize,
		model:       CloudflareWorkersBGE,
		ollamaHost:  "http://localhost:11434/api/embeddings",
		ollamaModel: "nomic-embed-text",
	}

	if endpoint != "" {
		if err := svc.ValidateEndpoint(); err != nil {
			log.Fatalf("❌ Embeddings endpoint validation failed: %v", err)
		}
		log.Printf("✅ Embeddings endpoint validated: %s", endpoint)
	}

	return svc
}

func newOllamaServiceWithBatchSize(batchSize int) *Service {
	return &Service{
		baseURL:     "",
		httpClient:  &http.Client{Timeout: 120 * time.Second},
		batchSize:   batchSize,
		model:       CloudflareWorkersBGE,
		ollamaHost:  "http://localhost:11434/api/embeddings",
		ollamaModel: "nomic-embed-text",
	}
}

// GetEmbedding returns embedding for a single text
func (s *Service) GetEmbedding(text string) ([]float32, error) {
	embeddings, err := s.GetBatchEmbeddings([]string{text})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}
	return embeddings[0], nil
}

// GetBatchEmbeddings returns embeddings for multiple texts with automatic batching
func (s *Service) GetBatchEmbeddings(texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	if s.baseURL != "" {
		var allEmbeddings [][]float32
		// Process in chunks to respect API batch size limits
		for i := 0; i < len(texts); i += s.batchSize {
			end := i + s.batchSize
			if end > len(texts) {
				end = len(texts)
			}

			chunk := texts[i:end]
			embeddings, err := s.getBatchChunkWithRetry(chunk, MaxRetries)
			if err != nil {
				return nil, fmt.Errorf("chunk %d-%d failed: %w", i, end, err)
			}

			allEmbeddings = append(allEmbeddings, embeddings...)
		}
		return allEmbeddings, nil
	}

	// Fallback to Ollama
	log.Println("Cloudflare endpoint not set, falling back to Ollama")
	var allEmbeddings [][]float32
	for _, text := range texts {
		embedding, err := ollama.GetOllamaEmbedding(text, s.ollamaModel, s.ollamaHost)
		if err != nil {
			return nil, fmt.Errorf("failed to get ollama embedding: %w", err)
		}
		allEmbeddings = append(allEmbeddings, embedding)
	}
	return allEmbeddings, nil
}

// getBatchChunk processes a single chunk of texts using Cloudflare Workers AI public endpoint
func (s *Service) getBatchChunk(texts []string) ([][]float32, error) {
	// Build Cloudflare Workers AI request
	reqBody := CloudflareWorkersRequest{
		Texts: texts,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Use baseURL directly - model is specified in request body
	httpReq, err := http.NewRequest("POST", s.baseURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Public endpoint - no authentication required
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	var cfResp CloudflareWorkersResponse
	if err := json.NewDecoder(resp.Body).Decode(&cfResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// Check for API errors
	if !cfResp.Success {
		return nil, fmt.Errorf("API request failed")
	}

	if len(cfResp.Embeddings) != len(texts) {
		return nil, fmt.Errorf("expected %d embeddings, got %d", len(texts), len(cfResp.Embeddings))
	}

	embeddings := make([][]float32, len(cfResp.Embeddings))
	copy(embeddings, cfResp.Embeddings)

	return embeddings, nil
}

// getBatchChunkWithRetry processes a batch with retries on failure
func (s *Service) getBatchChunkWithRetry(texts []string, maxRetries int) ([][]float32, error) {
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		embeddings, err := s.getBatchChunk(texts)
		if err == nil {
			return embeddings, nil
		}
		lastErr = err

		// Don't sleep on the last attempt
		if attempt < maxRetries {
			// Exponential backoff: 100ms * 2^attempt
			backoff := time.Duration(100*(1<<attempt)) * time.Millisecond
			time.Sleep(backoff)
		}
	}
	return nil, fmt.Errorf("embedding request failed after %d retries: %w", maxRetries, lastErr)
}

// GetBatchSize returns the configured batch size
func (s *Service) GetBatchSize() int {
	return s.batchSize
}

// ValidateEndpoint checks if the embeddings endpoint is accessible
func (s *Service) ValidateEndpoint() error {
	testReq := CloudflareWorkersRequest{
		Texts: []string{"test"},
	}

	jsonBody, err := json.Marshal(testReq)
	if err != nil {
		return fmt.Errorf("failed to marshal test request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", s.baseURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create test request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// Use a shorter timeout for validation
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("endpoint unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("endpoint returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// SetTimeout sets the HTTP client timeout
func (s *Service) SetTimeout(timeout interface{}) {
	switch t := timeout.(type) {
	case time.Duration:
		s.httpClient.Timeout = t
	case int:
		s.httpClient.Timeout = time.Duration(t) * time.Second
	}
}
