package p2pconsensus

import (
	"context"
	"net/http"
	"time"
)

// detectGateway checks if KNIRVGATEWAY is reachable.
func detectGateway(gatewayURL string, timeout time.Duration) bool {
	if gatewayURL == "" {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, gatewayURL+"/p2p/health", nil)
	if err != nil {
		return false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// ResolveMode determines which consensus mode to use based on config and gateway availability.
func ResolveMode(cfg ConsensusConfig) string {
	if !cfg.Enabled {
		return "disabled"
	}
	switch cfg.Mode {
	case "gateway":
		return "gateway"
	case "standalone":
		return "standalone"
	case "disabled":
		return "disabled"
	case "auto":
		fallthrough
	default:
		timeout, err := time.ParseDuration(cfg.GatewayTimeout)
		if err != nil {
			timeout = 2 * time.Second
		}
		if cfg.GatewayURL != "" && detectGateway(cfg.GatewayURL, timeout) {
			return "gateway"
		}
		return "standalone"
	}
}
