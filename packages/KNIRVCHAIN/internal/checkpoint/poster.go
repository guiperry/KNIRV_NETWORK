package checkpoint

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"KNIRVCHAIN/internal/database"
)

// DefaultOracleBaseURL is the primary testnet Oracle entry (merkle-math.md §1.4).
// Non-root KNIRVCHAIN deployments post here; co-located root nodes may instead set
// SocketPath to talk to the Oracle over its unix socket (no bearer token needed).
const DefaultOracleBaseURL = "https://testnet-gateway.knirv.network"
const DefaultOracleFailoverURL = "https://testnet-oracle.knirv.network"

// SubmitStatus is the persisted posting state of a checkpoint.
type SubmitStatus string

const (
	SubmitPending   SubmitStatus = "pending"
	SubmitSubmitted SubmitStatus = "submitted"
	SubmitFailed    SubmitStatus = "failed"
)

// Poster delivers signed checkpoints (and chain registration) to the Oracle,
// following the §1.4 transport tier rule: unix socket when co-located, otherwise
// HTTPS to the gateway with failover. It records per-checkpoint submit status in
// LevelDB so a restart can resume.
type Poster struct {
	baseURL          string
	failoverURL      string
	socketPath       string
	verifierSocket   string
	token            string
	httpClient       *http.Client
	oracleUnixClient *http.Client
	verifierClient   *http.Client
	db               *database.LevelDB
	retries          int
}

// PosterConfig tunes the poster. SocketPath, if set, is used as the dialer for
// all Oracle calls (root co-location). BaseURL is the HTTPS entry otherwise.
func NewPoster(db *database.LevelDB, baseURL, socketPath string) *Poster {
	if baseURL == "" {
		baseURL = DefaultOracleBaseURL
	}
	p := &Poster{
		baseURL: baseURL, failoverURL: strings.TrimRight(os.Getenv("KNIRV_ORACLE_FAILOVER_URL"), "/"),
		socketPath: socketPath, verifierSocket: os.Getenv("KNIRV_VERIFIER_SOCKET_PATH"),
		token: os.Getenv("KNIRV_GATEWAY_TOKEN"), db: db, retries: 3,
	}
	if strings.TrimSpace(p.verifierSocket) == "" {
		p.verifierSocket = fmt.Sprintf("%s/knirv-checkpoint-verifier-%d.sock", os.TempDir(), os.Getuid())
	}
	if p.failoverURL == "" {
		p.failoverURL = DefaultOracleFailoverURL
	}
	p.baseURL = strings.TrimRight(p.baseURL, "/")
	p.httpClient = &http.Client{Timeout: 30 * time.Second}
	if socketPath != "" {
		p.oracleUnixClient = unixHTTPClient(socketPath)
	}
	if p.verifierSocket != "" {
		p.verifierClient = unixHTTPClient(p.verifierSocket)
	}
	return p
}

func unixHTTPClient(socketPath string) *http.Client {
	return &http.Client{Timeout: 2 * time.Minute, Transport: &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
		},
	}}
}

// submitStatusKey is the LevelDB key for a checkpoint's posting state.
func submitStatusKey(chainID string, endHeight uint64) string {
	return fmt.Sprintf("checkpoint:submit:%s:%d", chainID, endHeight)
}

// SetSubmitStatus records the posting state of a checkpoint.
func (p *Poster) SetSubmitStatus(cp *Checkpoint, status SubmitStatus) error {
	if p.db == nil {
		return nil
	}
	return p.db.PutBytes(submitStatusKey(cp.ChainID, cp.EndHeight), []byte(status))
}

// SubmitStatusOf returns the recorded posting state, or empty if unknown.
func (p *Poster) SubmitStatusOf(cp *Checkpoint) (SubmitStatus, error) {
	if p.db == nil {
		return "", nil
	}
	b, err := p.db.GetBytes(submitStatusKey(cp.ChainID, cp.EndHeight))
	if err != nil || len(b) == 0 {
		return "", err
	}
	return SubmitStatus(b), nil
}

// postJSON sends a JSON POST to path and decodes a JSON response.
func (p *Poster) postJSON(ctx context.Context, path string, body interface{}) (map[string]interface{}, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	type target struct {
		base   string
		client *http.Client
	}
	targets := make([]target, 0, 3)
	if p.oracleUnixClient != nil {
		targets = append(targets, target{base: "http://oracle", client: p.oracleUnixClient})
	}
	targets = append(targets, target{base: p.baseURL, client: p.httpClient})
	if p.failoverURL != "" && p.failoverURL != p.baseURL {
		targets = append(targets, target{base: p.failoverURL, client: p.httpClient})
	}
	var lastErr error
	for attempt := 0; attempt < p.retries; attempt++ {
		for _, target := range targets {
			out, retry, err := p.doPost(ctx, target.client, target.base+path, payload)
			if err == nil {
				return out, nil
			}
			lastErr = err
			if !retry {
				return nil, err
			}
		}
		timer := time.NewTimer(time.Duration(attempt+1) * 250 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
	return nil, fmt.Errorf("all Oracle transports failed: %w", lastErr)
}

func (p *Poster) doPost(ctx context.Context, client *http.Client, url string, payload []byte) (map[string]interface{}, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("Content-Type", "application/json")
	if p.token != "" {
		req.Header.Set("Authorization", "Bearer "+p.token)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer resp.Body.Close()
	var out map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, resp.StatusCode >= 500, fmt.Errorf("POST %s: status %d", url, resp.StatusCode)
	}
	if resp.StatusCode/100 != 2 {
		message, _ := out["error"].(string)
		return nil, resp.StatusCode == 429 || resp.StatusCode >= 500, fmt.Errorf("POST %s rejected (%d): %s", url, resp.StatusCode, message)
	}
	return out, false, nil
}

// PostToLocalVerifier is the only transition-proof handoff used by
// KNIRVCHAIN. The verifier endpoint is deliberately reachable only over its
// dedicated Unix socket; it owns Oracle lookup, proof verification, and the
// tiered Oracle submission path.
func (p *Poster) PostToLocalVerifier(ctx context.Context, request interface{}) (map[string]interface{}, error) {
	if p.verifierClient == nil || strings.TrimSpace(p.verifierSocket) == "" {
		return nil, fmt.Errorf("local verifier socket is not configured")
	}
	payload, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	out, _, err := p.doPost(ctx, p.verifierClient, "http://verifier/internal/verify-checkpoint", payload)
	return out, err
}

// RegisterChain submits a chain registration to the Oracle. The body is the
// canonical JSON of types.ChainRegistration-shaped data; post as raw map to
// avoid cross-package imports.
func (p *Poster) RegisterChain(ctx context.Context, reg interface{}) (map[string]interface{}, error) {
	return p.postJSON(ctx, "/oracle/v3/registry/register", reg)
}

// PostCheckpoint posts a signed checkpoint to the Oracle and records its status.
func (p *Poster) PostCheckpoint(ctx context.Context, cp *Checkpoint) (map[string]interface{}, error) {
	if err := p.SetSubmitStatus(cp, SubmitPending); err != nil {
		return nil, err
	}
	out, err := p.postJSON(ctx, "/oracle/v3/checkpoints", cp)
	if err != nil {
		_ = p.SetSubmitStatus(cp, SubmitFailed)
		return nil, err
	}
	if err := p.SetSubmitStatus(cp, SubmitSubmitted); err != nil {
		return nil, err
	}
	return out, nil
}
