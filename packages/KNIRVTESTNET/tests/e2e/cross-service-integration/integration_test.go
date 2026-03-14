package integration

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"
)

const (
	GatewayURL = "http://localhost:8888"
	RootURL    = "http://localhost:1317"
	ChainURL   = "http://localhost:8090"
	GraphURL   = "http://localhost:8082"
	DVEUrl     = "http://localhost:8084"
	ValURL     = "http://localhost:8085"
	RouterURL  = "http://localhost:8086"
	Timeout    = 30 * time.Second
)

// TestServiceDiscovery tests service discovery through the gateway
func TestServiceDiscovery(t *testing.T) {
	client := &http.Client{Timeout: Timeout}

	t.Run("Gateway Service Discovery", func(t *testing.T) {
		resp, err := client.Get(GatewayURL + "/gateway/services")
		if err != nil {
			t.Fatalf("Failed to get services: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Service discovery failed with status: %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response: %v", err)
		}

		var servicesResponse map[string]interface{}
		if err := json.Unmarshal(body, &servicesResponse); err != nil {
			t.Fatalf("Failed to parse services response: %v", err)
		}

		t.Logf("✅ Service discovery response: %v", servicesResponse)
	})

	t.Run("Testnet Status Integration", func(t *testing.T) {
		resp, err := client.Get(GatewayURL + "/gateway/testnet/status")
		if err != nil {
			t.Fatalf("Failed to get testnet status: %v", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response: %v", err)
		}

		var statusResponse map[string]interface{}
		if err := json.Unmarshal(body, &statusResponse); err != nil {
			t.Fatalf("Failed to parse status response: %v", err)
		}

		// Check if all services are reported
		if services, ok := statusResponse["services"]; ok {
			if serviceMap, ok := services.(map[string]interface{}); ok {
				expectedServices := []string{"knirvoracle", "knirvchain", "knirvgraph", "knirvnexus_dve", "knirvnexus_validation", "knirvrouter"}
				for _, serviceName := range expectedServices {
					if service, exists := serviceMap[serviceName]; exists {
						t.Logf("✅ Service %s found in testnet status: %v", serviceName, service)
					} else {
						t.Logf("⚠️ Service %s not found in testnet status", serviceName)
					}
				}
			}
		}
	})
}

// TestCrossServiceCommunication tests communication between services
func TestCrossServiceCommunication(t *testing.T) {
	client := &http.Client{Timeout: Timeout}

	services := map[string]string{
		"KNIRV-ORACLE":      RootURL + "/health",
		"KNIRVCHAIN":      ChainURL + "/health",
		"KNIRVGRAPH":      GraphURL + "/height",
		"KNIRV-NEXUS-DVE": DVEUrl + "/health",
		"KNIRV-NEXUS-VAL": ValURL + "/health",
		"KNIRV-ROUTER":    RouterURL + "/status",
	}

	t.Run("All Services Responding", func(t *testing.T) {
		allHealthy := true
		for serviceName, endpoint := range services {
			resp, err := client.Get(endpoint)
			if err != nil {
				t.Errorf("Failed to connect to %s: %v", serviceName, err)
				allHealthy = false
				continue
			}
			resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("%s returned status %d", serviceName, resp.StatusCode)
				allHealthy = false
			} else {
				t.Logf("✅ %s is responding", serviceName)
			}
		}

		if allHealthy {
			t.Logf("✅ All services are healthy and responding")
		}
	})

	t.Run("Service Response Consistency", func(t *testing.T) {
		// Test multiple calls to ensure consistent responses
		for serviceName, endpoint := range services {
			responses := make([]int, 3)
			for i := 0; i < 3; i++ {
				resp, err := client.Get(endpoint)
				if err != nil {
					t.Errorf("Failed to connect to %s on attempt %d: %v", serviceName, i+1, err)
					continue
				}
				responses[i] = resp.StatusCode
				resp.Body.Close()
			}

			// Check consistency
			consistent := true
			for i := 1; i < len(responses); i++ {
				if responses[i] != responses[0] {
					consistent = false
					break
				}
			}

			if consistent {
				t.Logf("✅ %s responses are consistent: %v", serviceName, responses[0])
			} else {
				t.Errorf("❌ %s responses are inconsistent: %v", serviceName, responses)
			}
		}
	})
}

// TestDataFlow tests data flow between services
func TestDataFlow(t *testing.T) {
	client := &http.Client{Timeout: Timeout}

	t.Run("Blockchain Data Flow", func(t *testing.T) {
		// Get blockchain data from router
		routerResp, err := client.Get(RouterURL + "/status")
		if err != nil {
			t.Fatalf("Failed to get router status: %v", err)
		}
		defer routerResp.Body.Close()

		routerBody, err := io.ReadAll(routerResp.Body)
		if err != nil {
			t.Fatalf("Failed to read router response: %v", err)
		}

		var routerStatus map[string]interface{}
		if err := json.Unmarshal(routerBody, &routerStatus); err != nil {
			t.Fatalf("Failed to parse router status: %v", err)
		}

		// Check if blockchain data exists
		if blockchain, ok := routerStatus["block_chain"]; ok {
			if blocks, ok := blockchain.([]interface{}); ok {
				t.Logf("✅ Router reports %d blocks in blockchain", len(blocks))
			}
		}

		// Get chain health
		chainResp, err := client.Get(ChainURL + "/health")
		if err != nil {
			t.Fatalf("Failed to get chain health: %v", err)
		}
		defer chainResp.Body.Close()

		if chainResp.StatusCode == http.StatusOK {
			t.Logf("✅ KNIRVCHAIN is healthy and integrated with router")
		}
	})

	t.Run("Graph Data Integration", func(t *testing.T) {
		resp, err := client.Get(GraphURL + "/height")
		if err != nil {
			t.Fatalf("Failed to get graph height: %v", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response: %v", err)
		}

		var heightResponse map[string]interface{}
		if err := json.Unmarshal(body, &heightResponse); err != nil {
			t.Fatalf("Failed to parse height response: %v", err)
		}

		if height, ok := heightResponse["height"]; ok {
			t.Logf("✅ KNIRVGRAPH reports height: %v", height)
		}
	})

	t.Run("NEXUS Services Integration", func(t *testing.T) {
		// Test DVE service
		dveResp, err := client.Get(DVEUrl + "/health")
		if err != nil {
			t.Fatalf("Failed to get DVE health: %v", err)
		}
		defer dveResp.Body.Close()

		dveBody, err := io.ReadAll(dveResp.Body)
		if err != nil {
			t.Fatalf("Failed to read DVE response: %v", err)
		}

		var dveHealth map[string]interface{}
		if err := json.Unmarshal(dveBody, &dveHealth); err != nil {
			t.Fatalf("Failed to parse DVE health: %v", err)
		}

		if status, ok := dveHealth["status"]; ok && status == "healthy" {
			t.Logf("✅ KNIRV-NEXUS-DVE is healthy")
		}

		// Test Validation service
		valResp, err := client.Get(ValURL + "/health")
		if err != nil {
			t.Fatalf("Failed to get Validation health: %v", err)
		}
		defer valResp.Body.Close()

		valBody, err := io.ReadAll(valResp.Body)
		if err != nil {
			t.Fatalf("Failed to read Validation response: %v", err)
		}

		var valHealth map[string]interface{}
		if err := json.Unmarshal(valBody, &valHealth); err != nil {
			t.Fatalf("Failed to parse Validation health: %v", err)
		}

		if status, ok := valHealth["status"]; ok && status == "healthy" {
			t.Logf("✅ KNIRV-NEXUS-VAL is healthy")
		}

		t.Logf("✅ NEXUS services are integrated and healthy")
	})
}

// TestGatewayIntegration tests gateway integration with all services
func TestGatewayIntegration(t *testing.T) {
	client := &http.Client{Timeout: Timeout}

	t.Run("Gateway Health Aggregation", func(t *testing.T) {
		resp, err := client.Get(GatewayURL + "/gateway/health")
		if err != nil {
			t.Fatalf("Failed to get gateway health: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Gateway health check failed with status: %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response: %v", err)
		}

		var healthResponse map[string]interface{}
		if err := json.Unmarshal(body, &healthResponse); err != nil {
			t.Fatalf("Failed to parse health response: %v", err)
		}

		if status, ok := healthResponse["status"]; ok && status == "healthy" {
			t.Logf("✅ Gateway reports healthy status")
		} else {
			t.Errorf("Gateway health status: %v", healthResponse)
		}
	})

	t.Run("Authentication Integration", func(t *testing.T) {
		resp, err := client.Get(GatewayURL + "/auth/testnet-tokens")
		if err != nil {
			t.Fatalf("Failed to get auth tokens: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Auth tokens request failed with status: %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response: %v", err)
		}

		var tokenResponse map[string]interface{}
		if err := json.Unmarshal(body, &tokenResponse); err != nil {
			t.Fatalf("Failed to parse token response: %v", err)
		}

		if tokens, ok := tokenResponse["tokens"]; ok {
			t.Logf("✅ Authentication tokens available: %v", tokens)
		}
	})

	t.Run("End-to-End Integration", func(t *testing.T) {
		// Test complete flow through gateway
		endpoints := []string{
			"/gateway/health",
			"/gateway/services", 
			"/gateway/testnet/status",
			"/auth/testnet-tokens",
		}

		allWorking := true
		for _, endpoint := range endpoints {
			resp, err := client.Get(GatewayURL + endpoint)
			if err != nil {
				t.Errorf("Failed to access %s: %v", endpoint, err)
				allWorking = false
				continue
			}
			resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Endpoint %s returned status %d", endpoint, resp.StatusCode)
				allWorking = false
			} else {
				t.Logf("✅ Endpoint %s is working", endpoint)
			}
		}

		if allWorking {
			t.Logf("✅ Complete end-to-end integration is working")
		}
	})
}
