package checkpoint

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"

	knirvsigning "github.com/guiperry/knirv-sdk-go/signing"
)

// DefaultOracleGatewayURL is the public mainnet KNIRVGATEWAY entry, mirroring
// KNIRVCHAIN's internal/checkpoint/poster.go (DefaultOracleBaseURL there).
// KNIRVORACLE only runs on the root node — never assume it's reachable on a
// local socket; every non-root instance (the common case) must reach it
// through the public gateway, same as KNIRVCHAIN's own checkpoint poster
// does. Callers should prefer passing KNIRVSERVER's own resolvePublicURL()
// result (network-mode-aware: testnet/production/devnet/enterprise) instead
// of relying on this default where possible.
const DefaultOracleGatewayURL = "https://gateway.knirv.com"

// DefaultOracleFailoverURL is tried if the primary gateway URL fails. There is
// no built-in default: KNIRVORACLE has no standalone public domain of its
// own (it is only ever reached through KNIRVGATEWAY — see
// DefaultOracleGatewayURL), so a fabricated "oracle" subdomain here would
// just be a permanent, unresolvable DNS lookup masking the real failure from
// the primary gateway attempt. Set KNIRV_ORACLE_FAILOVER_URL explicitly if a
// deployment genuinely has a secondary gateway to fail over to.
const DefaultOracleFailoverURL = "https://testnet-gateway.knirv.com"
const DefaultOracleLocalURL = "http://localhost:8080"

// Poster delivers signed checkpoints and chain registration to KNIRVORACLE
// via the public KNIRVGATEWAY (gateway.knirv.com / testnet-gateway.knirv.com
// / devnet-<tag>.knirv.com, depending on network mode) — never a local
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


// OracleStatus is the subset of /oracle/v3/status needed for bond holds.
type OracleStatus struct {
	ChainID       string `json:"chain_id"`
	RootRegistryID string `json:"root_registry_id"`
}

func (p *Poster) getJSON(ctx context.Context, path string, out any) error {
	targets := []string{}
	if p.gatewayURL != "" {
		targets = append(targets, p.gatewayURL)
	}
	if p.baseURL != "" && p.baseURL != p.gatewayURL {
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
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+path, nil)
		if err != nil { lastErr = err; continue }
		resp, err := p.httpClient.Do(req)
		if err != nil { lastErr = err; continue }
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			lastErr = fmt.Errorf("GET %s%s: status %d: %s", base, path, resp.StatusCode, strings.TrimSpace(string(body)))
			continue
		}
		if err := json.Unmarshal(body, out); err != nil { lastErr = err; continue }
		return nil
	}
	if lastErr == nil { lastErr = fmt.Errorf("no oracle targets for GET %s", path) }
	return lastErr
}

// EnsureBondHold creates the D14 enrolment hold used by bonded authority
// registration when missing. It is best-effort idempotent: an existing/pending
// hold is treated as success.
func (p *Poster) EnsureBondHold(ctx context.Context, signer knirvsigning.RemoteSigner, chainID, holdID string, bond uint64) error {
	if holdID == "" || bond == 0 || signer == nil {
		return fmt.Errorf("missing hold parameters")
	}
	var st OracleStatus
	if err := p.getJSON(ctx, "/oracle/v3/status", &st); err != nil {
		return fmt.Errorf("fetch oracle status: %w", err)
	}
	if st.ChainID == "" || st.RootRegistryID == "" {
		return fmt.Errorf("oracle status missing chain/root registry context")
	}
	hold := knirvsigning.BondHold{
		HoldID:         holdID,
		Operator:       signer.Address(),
		Amount:         new(big.Int).SetUint64(bond).String(),
		RootRegistryID: st.RootRegistryID,
	}
	payload, err := json.Marshal(hold)
	if err != nil {
		return fmt.Errorf("encode bond hold: %w", err)
	}
	now := time.Now().Unix()
	signed, err := signer.SignMessage(ctx, knirvsigning.MessageEnvelope{
		Domain:        knirvsigning.RegistryDomain,
		Purpose:       knirvsigning.PurposeBondHold,
		ChainID:       st.ChainID,
		Nonce:         fmt.Sprintf("bond-hold:%s:%d", holdID, now),
		IssuedAtUnix:  now,
		ExpiresAtUnix: now + 300,
		Payload:       payload,
	})
	if err != nil {
		return fmt.Errorf("sign bond hold: %w", err)
	}
	sigJSON, err := json.Marshal(signed)
	if err != nil {
		return err
	}
	body := map[string]any{"hold_id": holdID, "hold": hold, "signature": string(sigJSON)}
	_, err = p.postJSON(ctx, "/oracle/v3/bond/hold", body)
	if err == nil {
		return nil
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "already pending") || strings.Contains(msg, "already consumed") || strings.Contains(msg, "already released") || strings.Contains(msg, "already refunded") || strings.Contains(msg, "already exists") {
		return nil
	}
	return err
}
