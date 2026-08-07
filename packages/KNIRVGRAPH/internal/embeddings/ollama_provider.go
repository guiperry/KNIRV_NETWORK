package embeddings

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type ollamaProvider struct {
	endpoint  string
	model     string
	dimension int
	client    *http.Client
}

func NewOllamaProvider(config ProviderConfig) (*ollamaProvider, error) {
	if config.Endpoint == "" {
		config.Endpoint = "http://localhost:11434"
	}
	if config.Model == "" {
		config.Model = "nomic-embed-text"
	}
	if config.Dimension <= 0 {
		config.Dimension = 768
	}
	timeout := time.Duration(config.TimeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	return &ollamaProvider{
		endpoint:  strings.TrimRight(config.Endpoint, "/"),
		model:     config.Model,
		dimension: config.Dimension,
		client:    &http.Client{Timeout: timeout},
	}, nil
}

func (p *ollamaProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	result := make([][]float32, len(texts))
	for i, text := range texts {
		emb, err := p.embedSingle(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("ollama embed failed for text %d: %w", i, err)
		}
		result[i] = emb
	}
	return result, nil
}

func (p *ollamaProvider) embedSingle(ctx context.Context, text string) ([]float32, error) {
	reqBody := map[string]interface{}{
		"model":  p.model,
		"prompt": text,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", p.endpoint+"/api/embeddings", strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}
	var result struct {
		Embedding []float64 `json:"embedding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	out := make([]float32, len(result.Embedding))
	for i, v := range result.Embedding {
		out[i] = float32(v)
	}
	return out, nil
}

func (p *ollamaProvider) Dimension() int {
	return p.dimension
}

func (p *ollamaProvider) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", p.endpoint+"/api/tags", nil)
	if err != nil {
		return err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (p *ollamaProvider) Close() error {
	return nil
}
