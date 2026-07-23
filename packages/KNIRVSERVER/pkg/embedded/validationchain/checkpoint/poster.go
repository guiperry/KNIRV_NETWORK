package checkpoint

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DefaultOracleSocketPath resolves KNIRVORACLE's Unix domain socket, the
// same one pkg/knirvoracle/client.go dials — no TCP port is exposed for it.
func DefaultOracleSocketPath() string {
	if envPath := strings.TrimSpace(os.Getenv("ORACLE_SOCKET_PATH")); envPath != "" {
		return envPath
	}
	if appDataDir := strings.TrimSpace(os.Getenv("KNIRV_APP_DATA_DIR")); appDataDir != "" {
		return filepath.Join(appDataDir, "sockets", "oracle.sock")
	}
	return filepath.Join("/var/lib/knirvserver", "sockets", "oracle.sock")
}

// Poster delivers signed checkpoints and chain registration to KNIRVORACLE
// over its Unix domain socket.
type Poster struct {
	token      string
	httpClient *http.Client
}

// NewPoster builds a Poster dialing socketPath. socketPath defaults to
// DefaultOracleSocketPath() when empty.
func NewPoster(socketPath string) *Poster {
	if strings.TrimSpace(socketPath) == "" {
		socketPath = DefaultOracleSocketPath()
	}
	return &Poster{
		token: os.Getenv("KNIRV_GATEWAY_TOKEN"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
					return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
				},
			},
		},
	}
}

func (p *Poster) postJSON(ctx context.Context, path string, body any) (map[string]any, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://unix"+path, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if p.token != "" {
		req.Header.Set("Authorization", "Bearer "+p.token)
	}
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("POST %s: status %d, decode: %w", path, resp.StatusCode, err)
	}
	if resp.StatusCode/100 != 2 {
		message, _ := out["error"].(string)
		return nil, fmt.Errorf("POST %s rejected (%d): %s", path, resp.StatusCode, message)
	}
	return out, nil
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

// Health checks KNIRVORACLE's health endpoint.
func (p *Poster) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://unix/oracle/v3/health", nil)
	if err != nil {
		return err
	}
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("oracle health returned %d", resp.StatusCode)
	}
	return nil
}
