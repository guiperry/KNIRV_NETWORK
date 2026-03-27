# MLC-LLM Integration for KNIRVCORE (Simplified)

This document defines a streamlined, minimum-viable integration of MLC-LLM into KNIRVCORE. It is designed for fast implementation, easy testing, and incremental feature expansion.

## Goals

- Add local inference to KNIRVCORE with minimal architecture changes.
- Keep KNIRVCORE Go node as controller and sidecar model inference engine as worker.
- Enable OpenAI-compatible request shape so existing logic can be adapted from current frameworks.
- Provide easy toggles for dev/test/production (local vs remote inference).

## Architecture (MVP)

- `knirvcore` (Go): orchestrates GLB memory retrieval, prompt assembly, and callouts.
- `cortex` sidecar (MLC-LLM): receives OpenAI-style requests and returns chat completions.
- Internal client package: `internal/cortex` for sidecar connectivity (health + complete).

```mermaid
graph TD
  A[KNIRVCORE Node] -->|HTTP(gRPC optional)| B[Cortex Sidecar]
  B -->|LLM Inference| C[MLC-LLM Engine]
  A -->|writes| D[Chain/GLB]
```

## MVP implementation (phase names are optional)

1. `internal/cortex` package. Provide:
   - `CortexClient.HealthCheck(ctx)`
   - `CortexClient.ChatComplete(ctx, req)`
2. Config flags in `config.yaml`:
   ```yaml
   cortex:
     enabled: true
     endpoint: "http://localhost:8000/v1"
     model: "Llama-3-8B-Instruct-q4f16_1"
     timeout_ms: 30000
   ```
3. Sidecar docker-compose entry:
   ```yaml
   services:
     cortex:
       image: mlcai/mlc-llm:latest
       ports: ["8000:8000"]
       command: ["mlc_llm", "serve", "Llama-3-8B-Instruct-q4f16_1", "--host", "0.0.0.0"]
       volumes: ["./models:/models"]
   ```
4. Add a `generator` tool route in MCP/mcp server that:
   - fetches relevant memory from GLB
   - builds system+user prompt
   - calls cortex client
   - returns/stores output

## Minimal code sketch

`internal/cortex/client.go`

```go
package cortex

import (
  "bytes"
  "context"
  "encoding/json"
  "fmt"
  "net/http"
  "time"
)

type Client struct {
  Endpoint string
  HTTP     *http.Client
}

type ChatRequest struct {
  Model    string    `json:"model"`
  Messages []Message `json:"messages"`
  Stream   bool      `json:"stream"`
}

type Message struct {
  Role    string `json:"role"`
  Content string `json:"content"`
}

func NewClient(endpoint string, timeout time.Duration) *Client {
  return &Client{Endpoint: endpoint, HTTP: &http.Client{Timeout: timeout}}
}

func (c *Client) HealthCheck(ctx context.Context) bool {
  req, _ := http.NewRequestWithContext(ctx, "GET", c.Endpoint+"/v1/health", nil)
  r, err := c.HTTP.Do(req)
  if err != nil || r.StatusCode != http.StatusOK {
    return false
  }
  return true
}

func (c *Client) ChatComplete(ctx context.Context, req ChatRequest) (string, error) {
  body, err := json.Marshal(req)
  if err != nil {
    return "", err
  }
  httpReq, err := http.NewRequestWithContext(ctx, "POST", c.Endpoint+"/v1/chat/completions", bytes.NewReader(body))
  if err != nil {
    return "", err
  }
  httpReq.Header.Set("Content-Type", "application/json")

  resp, err := c.HTTP.Do(httpReq)
  if err != nil {
    return "", err
  }
  defer resp.Body.Close()

  if resp.StatusCode != http.StatusOK {
    return "", fmt.Errorf("unexpected status %d", resp.StatusCode)
  }

  var out struct {
    Choices []struct { Message Message `json:"message"` } `json:"choices"`
  }
  if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
    return "", err
  }
  if len(out.Choices) < 1 {
    return "", fmt.Errorf("empty response")
  }

  return out.Choices[0].Message.Content, nil
}
```

## Optional extension (phase 2)

- RAG: GLB chunk retrieval, ranking, token-limited context assembly.
- KV caching: reuse system prompt + local embeddings on sidecar.
- Batching / queued requests: reduce burst load (5-10 statements) for high throughput.
- Speculative decoding and pipeline optimizations via MLC flags.

## Validation and testing

- Integration test: `curl http://localhost:8000/v1/health` should respond 200.
- Craft a black-box `generate_insight` endpoint test with sample query + known GLB content.
- Add Go unit tests for `internal/cortex/client.go` to validate request encoding and error paths.

## Next steps

1. Implement minimal sidecar code path only (no RAG).
2. Add a dedicated doc snippet for `knirv-core -> cortex` schema and mapping.
3. After success, add optional “memory retrieval + context join” section in a short addendum.
