package p2p

import (
	"net/http"
	"time"
)

// DetectGateway checks if KNIRVGATEWAY is reachable at the given URL.
// It performs a simple HTTP GET to /p2p/health with a 2-second timeout.
func DetectGateway(gatewayURL string) bool {
	if gatewayURL == "" {
		gatewayURL = "http://localhost:8080"
	}
	client := &http.Client{
		Timeout: 2 * time.Second,
	}
	resp, err := client.Get(gatewayURL + "/p2p/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
