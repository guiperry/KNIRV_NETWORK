package heartqwen

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// HTTPLLM talks to the llama wrapper's OpenAI-compatible API. Base is
// "http://llama" when Client dials the wrapper's unix socket.
type HTTPLLM struct {
	Client *http.Client
	Base   string
	Model  string
}

// Complete sends the frame as the only system message and carries its claim in
// body fields, which the wrapper checks and strips.
func (l *HTTPLLM) Complete(ctx context.Context, r Request) (string, error) {
	msgs := []map[string]string{{"role": "system", "content": r.Frame.System}}
	for _, m := range r.Messages {
		msgs = append(msgs, map[string]string{"role": m.Role, "content": m.Content})
	}
	body := map[string]any{
		"model": l.Model, "messages": msgs, "temperature": 0, "cache_prompt": true,
		"knirv_stage": r.Frame.Stage, "knirv_frame_hash": r.Frame.Hash, "knirv_priority": int(r.Priority),
	}
	if r.MaxTokens > 0 {
		body["max_tokens"] = r.MaxTokens
	}
	if len(r.Schema) > 0 {
		body["response_format"] = map[string]any{"type": "json_schema",
			"json_schema": map[string]any{"name": "output", "strict": true, "schema": r.Schema}}
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, l.Base+"/v1/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := l.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("llama HTTP %d: %s", res.StatusCode, strings.TrimSpace(string(b)))
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(b, &out); err != nil || len(out.Choices) == 0 {
		return "", fmt.Errorf("llama returned no completion")
	}
	return out.Choices[0].Message.Content, nil
}

// Ready reports whether the wrapper and llama-server are up.
func (l *HTTPLLM) Ready(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, l.Base+"/health", nil)
	if err != nil {
		return false
	}
	res, err := l.Client.Do(req)
	if err != nil {
		return false
	}
	defer res.Body.Close()
	return res.StatusCode == http.StatusOK
}
