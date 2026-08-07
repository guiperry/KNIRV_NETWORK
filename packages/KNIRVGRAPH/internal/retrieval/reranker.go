package retrieval

import (
	"KNIRVGRAPH/internal/types"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

type Reranker struct {
	endpoint string
	model    string
	client   *http.Client
}

func NewReranker(endpoint, model string) *Reranker {
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}
	if model == "" {
		model = "cross-encoder/ms-marco-MiniLM-L-6-v2"
	}
	return &Reranker{
		endpoint: strings.TrimRight(endpoint, "/"),
		model:    model,
		client:   &http.Client{Timeout: 15 * time.Second},
	}
}

func (r *Reranker) Rerank(ctx context.Context, query string, candidates []types.VectorSearchResult, topK int) ([]types.VectorSearchResult, error) {
	if topK <= 0 || topK > len(candidates) {
		topK = len(candidates)
	}
	if len(candidates) == 0 {
		return nil, nil
	}
	type scored struct {
		idx   int
		score float64
	}
	scores := make([]scored, len(candidates))
	for i, c := range candidates {
		text, _ := c.Metadata["text"].(string)
		score, err := r.scorePair(ctx, query, text)
		if err != nil {
			scores[i] = scored{idx: i, score: c.Score}
		} else {
			scores[i] = scored{idx: i, score: score}
		}
	}
	sort.Slice(scores, func(i, j int) bool { return scores[i].score > scores[j].score })
	if len(scores) > topK {
		scores = scores[:topK]
	}
	out := make([]types.VectorSearchResult, len(scores))
	for i, s := range scores {
		c := candidates[s.idx]
		out[i] = types.VectorSearchResult{
			ID:       c.ID,
			Score:    s.score,
			Metadata: c.Metadata,
			Vector:   c.Vector,
		}
	}
	return out, nil
}

func (r *Reranker) scorePair(ctx context.Context, query, candidate string) (float64, error) {
	prompt := fmt.Sprintf("Query: %s\nDocument: %s\nScore relevance 0-1:", query, truncate(candidate, 2000))
	reqBody := map[string]interface{}{
		"model":  r.model,
		"prompt": prompt,
		"stream": false,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return 0, err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", r.endpoint+"/api/generate", strings.NewReader(string(body)))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("reranker returned status %d", resp.StatusCode)
	}
	var result struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}
	var score float64
	if _, err := fmt.Sscanf(strings.TrimSpace(result.Response), "%f", &score); err != nil {
		return 0, nil
	}
	if score > 1 {
		score = 1
	}
	if score < 0 {
		score = 0
	}
	return score, nil
}

func (r *Reranker) ScorePairs(ctx context.Context, query string, candidates []types.VectorSearchResult) ([]float64, error) {
	scores := make([]float64, len(candidates))
	for i, c := range candidates {
		text, _ := c.Metadata["text"].(string)
		score, err := r.scorePair(ctx, query, text)
		if err != nil {
			scores[i] = c.Score
		} else {
			scores[i] = score
		}
	}
	return scores, nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
