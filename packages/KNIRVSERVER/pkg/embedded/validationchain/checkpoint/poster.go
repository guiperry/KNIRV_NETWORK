package checkpoint

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// DefaultOracleGatewayURL is the public mainnet KNIRVGATEWAY entry, mirroring
// KNIRVCHAIN's internal/checkpoint/poster.go (DefaultOracleBaseURL there).
// KNIRVORACLE only runs on the root node — never assume it's reachable on a
// local socket; every non-root instance (the common case) must reach it
// through the public gateway, same as KNIRVCHAIN's own checkpoint poster
// does. Callers should prefer passing KNIRVSERVER's own resolvePublicURL()
// result (network-mode-aware: testnet/production/devnet/enterprise) instead
// of relying on this default where possible.
const DefaultOracleGatewayURL = "https://gateway.knirv.network"

// DefaultOracleFailoverURL is tried if the primary gateway URL fails. There is
// no built-in default: KNIRVORACLE has no standalone public domain of its
// own (it is only ever reached through KNIRVGATEWAY — see
// DefaultOracleGatewayURL), so a fabricated "oracle" subdomain here would
// just be a permanent, unresolvable DNS lookup masking the real failure from
// the primary gateway attempt. Set KNIRV_ORACLE_FAILOVER_URL explicitly if a
// deployment genuinely has a secondary gateway to fail over to.
const DefaultOracleFailoverURL = "https://testnet-gateway.knirv.network"
const DefaultOracleLocalURL = "http://localhost:8080"

// Poster delivers signed checkpoints and chain registration to KNIRVORACLE
// via the public KNIRVGATEWAY (gateway.knirv.network / testnet-gateway.knirv.network
// / devnet-<tag>.knirv.network, depending on network mode) — never a local
// socket. The registry/checkpoint endpoints are themselves authenticated by
// the registered-author signature quorum carried in the request body, so
// posting over the public gateway is the intended, safe transport (same
// model KNIRVCHAIN's own checkpoint poster already uses).
type Poster struct {
	baseURL     string
	gatewayURL  string
	failoverURL string
	token       string
	httpClient  *http.Client
}

// NewPoster builds a Poster targeting gatewayURL (the network-mode-aware
// public KNIRVGATEWAY) and baseURL (the actual KNIRVORACLE endpoint, which
// may be the same as gatewayURL or an explicit override like KNIRV_ORACLE_URL).
// Both are attempted in order: gateway first, then baseURL, then failoverURL.
func NewPoster(gatewayURL, baseURL string) *Poster {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = DefaultOracleGatewayURL
	}
	if strings.TrimSpace(gatewayURL) == "" {
		gatewayURL = DefaultOracleGatewayURL
	}
	failoverURL := strings.TrimRight(strings.TrimSpace(os.Getenv("KNIRV_ORACLE_FAILOVER_URL")), "/")
	if failoverURL == "" {
		failoverURL = DefaultOracleFailoverURL
	}
	return &Poster{
		baseURL:     strings.TrimRight(baseURL, "/"),
		gatewayURL:  strings.TrimRight(gatewayURL, "/"),
		failoverURL: failoverURL,
		token:       os.Getenv("KNIRV_GATEWAY_TOKEN"),
		httpClient:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (p *Poster) postJSON(ctx context.Context, path string, body any) (map[string]any, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	targets := make([]string, 0, 4)
	if p.gatewayURL != "" && p.gatewayURL != p.baseURL {
		targets = append(targets, p.gatewayURL)
	}
	if p.baseURL != "" {
		targets = append(targets, p.baseURL)
	}
	if p.failoverURL != "" && p.failoverURL != p.baseURL && p.failoverURL != p.gatewayURL {
		targets = append(targets, p.failoverURL)
	}
	if DefaultOracleLocalURL != p.baseURL && DefaultOracleLocalURL != p.gatewayURL && DefaultOracleLocalURL != p.failoverURL {
		targets = append(targets, DefaultOracleLocalURL)
	}

	var lastErr error
	for _, base := range targets {
		out, retry, err := p.doPost(ctx, base+path, payload)
		if err == nil {
			return out, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("all oracle gateway targets failed: %w", lastErr)
}

func (p *Poster) doPost(ctx context.Context, url string, payload []byte) (map[string]any, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("Content-Type", "application/json")
	if p.token != "" {
		req.Header.Set("Authorization", "Bearer "+p.token)
	}
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return nil, resp.StatusCode >= 500, fmt.Errorf("POST %s: status %d, read body: %w", url, resp.StatusCode, err)
	}
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		// Surface the raw (truncated) body — a non-object response here almost
		// always means the request never reached KNIRVORACLE's own handler at
		// all (e.g. an unrelated service or an older deployment answering the
		// hostname), which "decode: %w" alone doesn't make obvious.
		return nil, resp.StatusCode >= 500, fmt.Errorf("POST %s: status %d, non-JSON-object response: %q", url, resp.StatusCode, bytes.TrimSpace(body))
	}
	if resp.StatusCode/100 != 2 {
		message, _ := out["error"].(string)
		return nil, resp.StatusCode == 429 || resp.StatusCode >= 500, fmt.Errorf("POST %s rejected (%d): %s", url, resp.StatusCode, message)
	}
	return out, false, nil
}

// RegisterChain submits a chain registration to KNIRVORACLE. Tolerates
// "already registered" so callers can invoke this unconditionally on every
// startup.
func (p *Poster) RegisterChain(ctx context.Context, reg *ChainRegistration) (map[string]any, error) {
	return p.postJSON(ctx, "/oracle/v3/registry/register", reg)
}

// PostCheckpoint posts a signed checkpoint to KNIRVORACLE.
func (p *Poster) PostCheckpoint(ctx context.Context, cp *Checkpoint) (map[string]any, error) {
	return p.postJSON(ctx, "/oracle/v3/checkpoints", cp)
}

// Health checks KNIRVORACLE's health endpoint through the gateway.
func (p *Poster) Health(ctx context.Context) error {
	targets := []string{p.gatewayURL, p.baseURL, p.failoverURL, DefaultOracleLocalURL}
	seen := make(map[string]struct{}, len(targets))
	var lastErr error
	for _, base := range targets {
		base = strings.TrimRight(strings.TrimSpace(base), "/")
		if base == "" {
			continue
		}
		if _, ok := seen[base]; ok {
			continue
		}
		seen[base] = struct{}{}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/oracle/v3/health", nil)
		if err != nil {
			lastErr = err
			continue
		}
		resp, err := p.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			return nil
		}
		lastErr = fmt.Errorf("oracle health at %s returned %d", base, resp.StatusCode)
	}
	return fmt.Errorf("all oracle health targets failed: %w", lastErr)
}
