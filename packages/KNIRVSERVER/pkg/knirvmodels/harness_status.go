package knirvmodels

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

// HarnessStatus reads the inference instance's /harness/status document over its
// Unix socket and returns its top-level blocks untouched. KNIRVMODELS owns what the
// document says (which backend is active, the embedding server's phase and digest,
// ...); SERVER only carries it, so it neither parses nor names those blocks.
func (i *Inference) HarnessStatus(ctx context.Context) (map[string]json.RawMessage, error) {
	client := &http.Client{
		Timeout: 3 * time.Second,
		Transport: &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, "unix", i.cfg.SocketPath)
		}},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://inference/harness/status", nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("harness status: HTTP %d", resp.StatusCode)
	}
	var doc map[string]json.RawMessage
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&doc); err != nil {
		return nil, fmt.Errorf("harness status: %w", err)
	}
	return doc, nil
}
