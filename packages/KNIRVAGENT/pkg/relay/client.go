package relay

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// Config points a WASM/WASI agent at KNIRVGATEWAY. The gateway owns public HTTP
// access and proxies /api/v1/* to KNIRVSERVER over its backend Unix socket.
type Config struct {
	GatewayURL string `json:"gateway_url,omitempty"`
	DVEID      string `json:"dve_id,omitempty"`
	AuthToken  string `json:"auth_token,omitempty"`
	Enabled    bool   `json:"relay,omitempty"`
}

type executeRequest struct {
	Command string `json:"command"`
}

type executeResponse struct {
	Output  string `json:"output"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// FromEnv reads relay settings for WASI runtimes.
func FromEnv() Config {
	cfg := Config{
		GatewayURL: firstEnv("KNIRVAGENT_GATEWAY_URL", "KNIRVGATEWAY_URL", "KNIRV_GATEWAY_URL"),
		DVEID:      firstEnv("KNIRVAGENT_DVE_ID", "KNIRV_DVE_ID", "DVE_ID"),
		AuthToken:  firstEnv("KNIRVAGENT_AUTH_TOKEN", "KNIRV_AUTH_TOKEN", "AUTH_TOKEN"),
		Enabled:    truthy(os.Getenv("KNIRVAGENT_RELAY")),
	}
	if cfg.GatewayURL != "" && cfg.DVEID != "" {
		cfg.Enabled = true
	}
	return cfg
}

func (c Config) Ready() bool {
	return strings.TrimSpace(c.GatewayURL) != "" && strings.TrimSpace(c.DVEID) != ""
}

func (c Config) Execute(ctx context.Context, command string) (string, error) {
	if strings.TrimSpace(command) == "" {
		return "", fmt.Errorf("command is required")
	}
	if !c.Ready() {
		return "", fmt.Errorf("relay requires KNIRVAGENT_GATEWAY_URL and KNIRVAGENT_DVE_ID")
	}

	body, err := json.Marshal(executeRequest{Command: command})
	if err != nil {
		return "", fmt.Errorf("encode relay command: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.executeURL(), bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build relay request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AuthToken)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("relay request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("read relay response: %w", err)
	}

	var result executeResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return "", fmt.Errorf("relay HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
		}
		return strings.TrimSpace(string(respBody)), nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if result.Error != "" {
			return "", fmt.Errorf("relay HTTP %d: %s", resp.StatusCode, result.Error)
		}
		return "", fmt.Errorf("relay HTTP %d", resp.StatusCode)
	}
	if !result.Success && result.Error != "" {
		return "", fmt.Errorf("%s", result.Error)
	}
	return result.Output, nil
}

func (c Config) executeURL() string {
	base := strings.TrimRight(strings.TrimSpace(c.GatewayURL), "/")
	dveID := url.PathEscape(strings.TrimSpace(c.DVEID))
	return base + "/api/v1/dve/" + dveID + "/supervisor-agent/execute"
}

func firstEnv(names ...string) string {
	for _, name := range names {
		if value := strings.TrimSpace(os.Getenv(name)); value != "" {
			return value
		}
	}
	return ""
}

func truthy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
