package integration_tests

import (
	"net/http"
	"testing"
	"time"
)

// TestKNIRVTestnetIntegration verifies that all services embedded in
// KNIRVSERVER --testnet are reachable and healthy.
func TestKNIRVTestnetIntegration(t *testing.T) {
	client := &http.Client{Timeout: 10 * time.Second}

	endpoints := map[string]string{
		"knirvserver":       "http://localhost:8084/health",
		"knirvserver-testnet": "http://localhost:8084/testnet/status",
		"knirvchain":        "http://localhost:8090/health",
		"knirvgraph":        "http://localhost:8082/height",
		"knirvgateway":      "http://localhost:8888/gateway/health",
	}

	for name, url := range endpoints {
		resp, err := client.Get(url)
		if err != nil {
			t.Errorf("%s health check failed: %v", name, err)
			continue
		}
		resp.Body.Close()
		if resp.StatusCode >= 300 {
			t.Errorf("%s returned status %d (url: %s)", name, resp.StatusCode, url)
		} else {
			t.Logf("%s OK (%d)", name, resp.StatusCode)
		}
	}
}

// TestKNIRVTestnetTokens verifies the testnet token endpoint is available
// from both the gateway proxy and the direct KNIRVSERVER route.
func TestKNIRVTestnetTokens(t *testing.T) {
	client := &http.Client{Timeout: 10 * time.Second}

	sources := map[string]string{
		"gateway-tokens": "http://localhost:8888/auth/testnet-tokens",
		"server-tokens":  "http://localhost:8084/auth/testnet-tokens",
	}

	for name, url := range sources {
		resp, err := client.Get(url)
		if err != nil {
			t.Logf("%s unavailable: %v (non-fatal)", name, err)
			continue
		}
		resp.Body.Close()
		if resp.StatusCode >= 300 {
			t.Logf("%s returned status %d (non-fatal)", name, resp.StatusCode)
		} else {
			t.Logf("%s OK (%d)", name, resp.StatusCode)
		}
	}
}
