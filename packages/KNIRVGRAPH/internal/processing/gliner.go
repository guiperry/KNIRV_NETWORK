package processing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"KNIRVGRAPH/internal/types"
)

// GLiNERClient implements the common GLiNER server /predict contract. Keeping
// inference behind a local process avoids ONNX/CGo linkage in KNIRVGRAPH.
type GLiNERClient struct {
	endpoint, model string
	client          *http.Client
}

func NewGLiNERClient(endpoint, model string, timeoutSeconds int) *GLiNERClient {
	d := time.Duration(timeoutSeconds) * time.Second
	if d <= 0 {
		d = 15 * time.Second
	}
	return &GLiNERClient{endpoint: strings.TrimRight(endpoint, "/"), model: model, client: &http.Client{Timeout: d}}
}

func (g *GLiNERClient) Extract(ctx context.Context, documentID, text string, labels []string, threshold float64) ([]types.ExtractedEntity, error) {
	body, err := json.Marshal(map[string]any{"text": text, "labels": labels, "threshold": threshold, "model": g.model})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.endpoint+"/predict", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GLiNER request: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GLiNER returned %d: %s", resp.StatusCode, string(raw))
	}
	var result struct {
		Entities []struct {
			Text  string  `json:"text"`
			Label string  `json:"label"`
			Score float64 `json:"score"`
			Start int     `json:"start"`
			End   int     `json:"end"`
		} `json:"entities"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("decode GLiNER response: %w", err)
	}
	out := make([]types.ExtractedEntity, 0, len(result.Entities))
	for _, e := range result.Entities {
		if e.Text == "" || e.Score < threshold {
			continue
		}
		out = append(out, types.ExtractedEntity{DocumentID: documentID, Type: strings.ToUpper(e.Label), Name: e.Text, Confidence: e.Score, Properties: map[string]any{"start": e.Start, "end": e.End, "extractor": "gliner"}})
	}
	return out, nil
}
