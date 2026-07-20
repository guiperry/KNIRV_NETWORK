package checkpoint

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"KNIRVCHAIN/internal/database"
)

// DefaultOracleBaseURL is the primary testnet Oracle entry (merkle-math.md §1.4).
// Non-root KNIRVCHAIN deployments post here; co-located root nodes may instead set
// SocketPath to talk to the Oracle over its unix socket (no bearer token needed).
const DefaultOracleBaseURL = "https://testnet-gateway.knirv.network"

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
	baseURL    string
	socketPath string
	httpClient *http.Client
	db         *database.LevelDB
}

// PosterConfig tunes the poster. SocketPath, if set, is used as the dialer for
// all Oracle calls (root co-location). BaseURL is the HTTPS entry otherwise.
func NewPoster(db *database.LevelDB, baseURL, socketPath string) *Poster {
	if baseURL == "" {
		baseURL = DefaultOracleBaseURL
	}
	p := &Poster{
		baseURL:    baseURL,
		socketPath: socketPath,
		db:         db,
	}
	transport := &http.Transport{}
	if socketPath != "" {
		transport.DialContext = func(ctx context.Context, _, _ string) (net.Conn, error) {
			return net.Dial("unix", socketPath)
		}
	}
	p.httpClient = &http.Client{Transport: transport, Timeout: 30 * time.Second}
	return p
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
	url := p.baseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		// Non-JSON error bodies still surface via status.
		return nil, fmt.Errorf("oracle %s %s: status %d", http.MethodPost, path, resp.StatusCode)
	}
	if resp.StatusCode/100 != 2 {
		if msg, ok := out["error"].(string); ok {
			return nil, fmt.Errorf("oracle rejected %s: %s", path, msg)
		}
		return nil, fmt.Errorf("oracle %s %s: status %d", http.MethodPost, path, resp.StatusCode)
	}
	return out, nil
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
