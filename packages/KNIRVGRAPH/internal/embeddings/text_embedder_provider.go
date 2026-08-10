package embeddings

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type textEmbedderProvider struct {
	endpoint  string
	model     string
	dimension int
	client    *http.Client
}

func NewTextEmbedderProvider(config ProviderConfig) (*textEmbedderProvider, error) {
	if config.Endpoint == "" {
		config.Endpoint = "http://localhost:8089"
	}
	if config.Model == "" {
		config.Model = "text-embedder"
	}
	if config.Dimension <= 0 {
		config.Dimension = 384
	}
	timeout := time.Duration(config.TimeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = 15 * time.Second
	}
	return &textEmbedderProvider{
		endpoint:  strings.TrimRight(config.Endpoint, "/"),
		model:     config.Model,
		dimension: config.Dimension,
		client:    &http.Client{Timeout: timeout},
	}, nil
}

func (p *textEmbedderProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	reqBody := map[string]interface{}{
		"model": p.model,
		"texts": texts,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", p.endpoint+"/embed", strings.NewReader(string(body)))
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
		return nil, fmt.Errorf("text-embedder returned status %d", resp.StatusCode)
	}
	var raw [][]float64
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	if len(raw) != len(texts) {
		return nil, fmt.Errorf("text-embedder returned %d embeddings for %d texts", len(raw), len(texts))
	}
	out := make([][]float32, len(raw))
	for i, vec := range raw {
		if len(vec) != p.dimension {
			return nil, fmt.Errorf("text-embedder embedding %d dimension mismatch: expected %d, got %d", i, p.dimension, len(vec))
		}
		out[i] = make([]float32, len(vec))
		for j, v := range vec {
			out[i][j] = float32(v)
		}
	}
	return out, nil
}

func (p *textEmbedderProvider) Dimension() int {
	return p.dimension
}

func (p *textEmbedderProvider) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", p.endpoint+"/health", nil)
	if err != nil {
		return err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("text-embedder health check failed: %d", resp.StatusCode)
	}
	return nil
}

func (p *textEmbedderProvider) Close() error {
	return nil
}
