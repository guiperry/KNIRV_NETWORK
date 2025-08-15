package userjourney

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
)

const (
	GatewayURL = "http://localhost:8888"
	Timeout    = 30 * time.Second
)

// TestBasicUserJourney tests the basic user journey through the KNIRV testnet
func TestBasicUserJourney(t *testing.T) {
	client := &http.Client{Timeout: Timeout}

	t.Run("Gateway Health Check", func(t *testing.T) {
		resp, err := client.Get(GatewayURL + "/gateway/health")
		if err != nil {
			t.Fatalf("Failed to connect to gateway: %v", err)
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

		if status, ok := healthResponse["status"]; !ok || status != "healthy" {
			t.Errorf("Gateway is not healthy: %v", healthResponse)
		}

		t.Logf("✅ Gateway health check passed")
	})

	t.Run("Service Discovery", func(t *testing.T) {
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

		if services, ok := servicesResponse["services"]; ok {
			t.Logf("✅ Discovered services: %v", services)
		} else {
			t.Logf("⚠️ No services field in response, but endpoint is accessible")
		}
	})

	t.Run("Testnet Status", func(t *testing.T) {
		resp, err := client.Get(GatewayURL + "/gateway/testnet/status")
		if err != nil {
			t.Fatalf("Failed to get testnet status: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Testnet status check failed with status: %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response: %v", err)
		}

		var statusResponse map[string]interface{}
		if err := json.Unmarshal(body, &statusResponse); err != nil {
			t.Fatalf("Failed to parse status response: %v", err)
		}

		t.Logf("✅ Testnet status: %v", statusResponse)
	})

	t.Run("Authentication Tokens", func(t *testing.T) {
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

		t.Logf("✅ Authentication tokens available: %v", tokenResponse)
	})
}

// TestServiceConnectivity tests connectivity to individual services
func TestServiceConnectivity(t *testing.T) {
	client := &http.Client{Timeout: Timeout}

	services := map[string]string{
		"KNIRV-ROOT":      "http://localhost:1317/health",
		"KNIRVCHAIN":      "http://localhost:8090/health",
		"KNIRVGRAPH":      "http://localhost:8082/height",
		"KNIRV-NEXUS-DVE": "http://localhost:8084/health",
		"KNIRV-NEXUS-VAL": "http://localhost:8085/health",
		"KNIRV-ROUTER":    "http://localhost:8086/status",
	}

	for serviceName, endpoint := range services {
		t.Run(fmt.Sprintf("Connect to %s", serviceName), func(t *testing.T) {
			resp, err := client.Get(endpoint)
			if err != nil {
				t.Errorf("Failed to connect to %s: %v", serviceName, err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("%s returned status %d", serviceName, resp.StatusCode)
				return
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Errorf("Failed to read %s response: %v", serviceName, err)
				return
			}

			t.Logf("✅ %s is responding: %s", serviceName, string(body))
		})
	}
}

// TestEndToEndWorkflow tests a complete end-to-end workflow
func TestEndToEndWorkflow(t *testing.T) {
	client := &http.Client{Timeout: Timeout}

	t.Run("Complete Workflow", func(t *testing.T) {
		// Step 1: Check gateway health
		t.Log("Step 1: Checking gateway health...")
		resp, err := client.Get(GatewayURL + "/gateway/health")
		if err != nil {
			t.Fatalf("Gateway health check failed: %v", err)
		}
		resp.Body.Close()

		// Step 2: Get testnet tokens
		t.Log("Step 2: Getting testnet tokens...")
		resp, err = client.Get(GatewayURL + "/auth/testnet-tokens")
		if err != nil {
			t.Fatalf("Failed to get testnet tokens: %v", err)
		}
		resp.Body.Close()

		// Step 3: Check service discovery
		t.Log("Step 3: Checking service discovery...")
		resp, err = client.Get(GatewayURL + "/gateway/services")
		if err != nil {
			t.Fatalf("Service discovery failed: %v", err)
		}
		resp.Body.Close()

		// Step 4: Verify testnet status
		t.Log("Step 4: Verifying testnet status...")
		resp, err = client.Get(GatewayURL + "/gateway/testnet/status")
		if err != nil {
			t.Fatalf("Testnet status check failed: %v", err)
		}
		resp.Body.Close()

		t.Log("✅ Complete end-to-end workflow successful!")
	})
}

// TestPerformanceBasics tests basic performance characteristics
func TestPerformanceBasics(t *testing.T) {
	client := &http.Client{Timeout: Timeout}

	t.Run("Response Time Check", func(t *testing.T) {
		start := time.Now()
		resp, err := client.Get(GatewayURL + "/gateway/health")
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("Health check failed: %v", err)
		}
		resp.Body.Close()

		if duration > 5*time.Second {
			t.Errorf("Health check took too long: %v", duration)
		} else {
			t.Logf("✅ Health check completed in %v", duration)
		}
	})

	t.Run("Concurrent Requests", func(t *testing.T) {
		const numRequests = 10
		results := make(chan error, numRequests)

		for i := 0; i < numRequests; i++ {
			go func() {
				resp, err := client.Get(GatewayURL + "/gateway/health")
				if err != nil {
					results <- err
					return
				}
				resp.Body.Close()
				results <- nil
			}()
		}

		var errors []error
		for i := 0; i < numRequests; i++ {
			if err := <-results; err != nil {
				errors = append(errors, err)
			}
		}

		if len(errors) > 0 {
			t.Errorf("Concurrent requests failed: %v", errors)
		} else {
			t.Logf("✅ All %d concurrent requests succeeded", numRequests)
		}
	})
}
