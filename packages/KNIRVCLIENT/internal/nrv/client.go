package nrv

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

// commitPath is the public KNIRVGATEWAY route that proxies to KNIRVGRAPH's
// POST /nrv/errors/commit (see packages/KNIRVGATEWAY/internal/server/server.go,
// which strips /api/graph before forwarding to KNIRVGRAPH over graph.sock).
const commitPath = "/api/graph/nrv/errors/commit"

// Client submits .nrv error commits to KNIRVGRAPH via KNIRVGATEWAY.
type Client struct {
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a Client targeting baseURL (e.g.
// "https://testnet-gateway.knirv.network").
func NewClient(baseURL string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		baseURL:    strings.TrimRight(baseURL, "/"),
	}
}

// SubmitErrorCommit POSTs commit to KNIRVGRAPH's error-commit endpoint.
// Any non-2xx response is returned as an error; callers should treat
// submission failures as best-effort and never let them abort a scan.
func (c *Client) SubmitErrorCommit(ctx context.Context, commit *ErrorNodeCommit) error {
	body, err := json.Marshal(commit)
	if err != nil {
		return fmt.Errorf("failed to marshal error commit: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+commitPath, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to build error commit request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to submit error commit: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("error commit rejected with status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	return nil
}
