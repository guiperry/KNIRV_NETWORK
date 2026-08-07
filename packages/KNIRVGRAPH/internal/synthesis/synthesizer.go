package synthesis

import (
	"KNIRVGRAPH/internal/types"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Synthesizer struct {
	endpoint string
	model    string
	client   *http.Client
}

func NewSynthesizer(endpoint, model string) *Synthesizer {
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}
	if model == "" {
		model = "llama3"
	}
	return &Synthesizer{
		endpoint: strings.TrimRight(endpoint, "/"),
		model:    model,
		client:   &http.Client{Timeout: 60 * time.Second},
	}
}

func (s *Synthesizer) Synthesize(ctx context.Context, req types.SynthesisRequest) (*types.SynthesisResponse, error) {
	contextText := s.buildContextText(req.Contexts)
	prompt := s.buildPrompt(req.Query, contextText, req.MaxTokens)
	reqBody := map[string]interface{}{
		"model":       s.model,
		"prompt":      prompt,
		"stream":      false,
		"options":     map[string]interface{}{"temperature": req.Temperature},
		"max_tokens":  req.MaxTokens,
	}
	if req.LLMEndpoint != "" {
		reqBody["model"] = req.LLMModel
		reqBody["endpoint"] = req.LLMEndpoint
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", s.endpoint+"/api/generate", strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("synthesis endpoint returned status %d", resp.StatusCode)
	}
	var result struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	answer, reasoning := splitAnswerReasoning(result.Response)
	sources := s.extractSources(req.Contexts)
	return &types.SynthesisResponse{
		Answer:     answer,
		Reasoning:  reasoning,
		Confidence: 0.85,
		Sources:    sources,
		LatencyMs:  0,
	}, nil
}

func (s *Synthesizer) buildContextText(contexts []types.RetrievalResult) string {
	var b strings.Builder
	for i, ctx := range contexts {
		b.WriteString(fmt.Sprintf("[%d] %s\n", i+1, truncate(ctx.Results[0].Metadata["text"].(string), 1000)))
	}
	return b.String()
}

func (s *Synthesizer) buildPrompt(query, contextText string, maxTokens int) string {
	return fmt.Sprintf("You are a helpful assistant. Use the following context to answer the question. If the answer is not in the context, say you don't know.\n\nContext:\n%s\n\nQuestion: %s\n\nAnswer:", contextText, query)
}

func (s *Synthesizer) extractSources(contexts []types.RetrievalResult) []string {
	sources := make([]string, 0, len(contexts))
	for _, ctx := range contexts {
		for _, r := range ctx.Results {
			sources = append(sources, r.ID)
		}
	}
	return sources
}

func splitAnswerReasoning(text string) (string, string) {
	if idx := strings.Index(text, "Reasoning:"); idx >= 0 {
		return strings.TrimSpace(text[:idx]), strings.TrimSpace(text[idx+10:])
	}
	return strings.TrimSpace(text), ""
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
