package database

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// TextEmbedderClient uses the deterministic G-Text Embedder service launched
// by KNIRVSERVER. It deliberately fails closed: production indexing must not
// silently switch vector spaces when the service is unavailable.
type TextEmbedderClient struct {
	baseURL string
	client  *http.Client
}

func NewTextEmbedderClient(baseURL string) *TextEmbedderClient {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "http://127.0.0.1:8089"
	}
	return &TextEmbedderClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *TextEmbedderClient) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	payload, err := json.Marshal(map[string]string{"text": text, "dtype": "float_signed"})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/embed", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("text embedder request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("text embedder returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var result struct {
		EmbeddingSigned []float32 `json:"embedding_signed"`
		EmbeddingFloat  []float32 `json:"embedding_float"`
		Embedding       []int32   `json:"embedding"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode text embedder response: %w", err)
	}
	if len(result.EmbeddingSigned) > 0 {
		return result.EmbeddingSigned, nil
	}
	if len(result.EmbeddingFloat) > 0 {
		return result.EmbeddingFloat, nil
	}
	if len(result.Embedding) > 0 {
		out := make([]float32, len(result.Embedding))
		for i, value := range result.Embedding {
			out[i] = float32(value) / 10000
		}
		return out, nil
	}
	return nil, fmt.Errorf("text embedder returned no embedding vector")
}
