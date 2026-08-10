package embeddings

import (
	"KNIRVGRAPH/internal/types"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// CandleProvider uses an out-of-process Candle runtime, keeping the Go binary
// CGo-free. Endpoint may be HTTP(S) or exec:///path/to/candle-worker. The
// executable receives a JSON request on stdin and returns JSON on stdout.
type CandleProvider struct {
	config ProviderConfig
	client *http.Client
	mu     sync.RWMutex
	closed bool
}

func NewCandleProvider(config ProviderConfig) (*CandleProvider, error) {
	defaults := DefaultProviderConfig(typesEmbeddingCandle())
	config = mergeProviderConfig(config, defaults)
	if config.Dimension <= 0 {
		return nil, fmt.Errorf("candle dimension must be positive")
	}
	if !strings.HasPrefix(config.Endpoint, "http://") && !strings.HasPrefix(config.Endpoint, "https://") && !strings.HasPrefix(config.Endpoint, "exec://") {
		return nil, fmt.Errorf("candle endpoint must use http, https, or exec scheme")
	}
	timeout := time.Duration(config.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &CandleProvider{config: config, client: &http.Client{Timeout: timeout}}, nil
}

// Kept local to avoid exposing provider-specific defaulting details.
func typesEmbeddingCandle() types.EmbeddingProviderType { return types.EmbeddingProviderCandle }

func mergeProviderConfig(got, defaults ProviderConfig) ProviderConfig {
	if got.Type == "" {
		got.Type = defaults.Type
	}
	if got.Endpoint == "" {
		got.Endpoint = defaults.Endpoint
	}
	if got.Model == "" {
		got.Model = defaults.Model
	}
	if got.Dimension <= 0 {
		got.Dimension = defaults.Dimension
	}
	if got.BatchSize <= 0 {
		got.BatchSize = defaults.BatchSize
	}
	if got.TimeoutSeconds <= 0 {
		got.TimeoutSeconds = defaults.TimeoutSeconds
	}
	return got
}

func (p *CandleProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	p.mu.RLock()
	closed := p.closed
	p.mu.RUnlock()
	if closed {
		return nil, fmt.Errorf("candle provider is closed")
	}
	payload, err := json.Marshal(map[string]any{"model": p.config.Model, "texts": texts})
	if err != nil {
		return nil, err
	}
	var raw []byte
	if strings.HasPrefix(p.config.Endpoint, "exec://") {
		path := strings.TrimPrefix(p.config.Endpoint, "exec://")
		cmd := exec.CommandContext(ctx, path, "embed")
		cmd.Stdin = bytes.NewReader(payload)
		raw, err = cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("candle worker: %w", err)
		}
	} else {
		url := strings.TrimRight(p.config.Endpoint, "/") + "/embed"
		req, e := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
		if e != nil {
			return nil, e
		}
		req.Header.Set("Content-Type", "application/json")
		resp, e := p.client.Do(req)
		if e != nil {
			return nil, fmt.Errorf("candle request: %w", e)
		}
		defer resp.Body.Close()
		raw, e = io.ReadAll(io.LimitReader(resp.Body, 64<<20))
		if e != nil {
			return nil, e
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("candle endpoint returned %d: %s", resp.StatusCode, string(raw))
		}
	}
	var envelope struct {
		Embeddings [][]float32 `json:"embeddings"`
	}
	if err = json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("decode candle response: %w", err)
	}
	if len(envelope.Embeddings) != len(texts) {
		return nil, fmt.Errorf("candle returned %d embeddings for %d texts", len(envelope.Embeddings), len(texts))
	}
	for i, v := range envelope.Embeddings {
		if len(v) != p.config.Dimension {
			return nil, fmt.Errorf("candle embedding %d dimension mismatch: expected %d, got %d", i, p.config.Dimension, len(v))
		}
	}
	return envelope.Embeddings, nil
}
func (p *CandleProvider) Dimension() int { return p.config.Dimension }
func (p *CandleProvider) Health(ctx context.Context) error {
	_, err := p.Embed(ctx, []string{"health"})
	return err
}
func (p *CandleProvider) Close() error {
	p.mu.Lock()
	p.closed = true
	p.mu.Unlock()
	p.client.CloseIdleConnections()
	return nil
}
