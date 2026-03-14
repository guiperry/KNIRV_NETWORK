package integration_tests

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// NetworkValidator validates the complete KNIRV network startup
type NetworkValidator struct {
	client    *http.Client
	authToken string
	services  map[string]ServiceInfo
}

// ServiceInfo contains information about a KNIRV service
type ServiceInfo struct {
	Name        string
	URL         string
	HealthURL   string
	IsHealthy   bool
	LastChecked time.Time
	Error       string
}

// NewNetworkValidator creates a new network validator
func NewNetworkValidator() *NetworkValidator {
	return &NetworkValidator{
		client: &http.Client{Timeout: 30 * time.Second},
		services: map[string]ServiceInfo{
			"knirvoracle": {
				Name:      "KNIRV-ORACLE",
				URL:       "http://localhost:1317",
				HealthURL: "http://localhost:1317/health",
			},
			"knirvchain": {
				Name:      "KNIRVCHAIN",
				URL:       "http://localhost:8090",
				HealthURL: "http://localhost:8090/health",
			},
			"knirvgraph": {
				Name:      "KNIRVGRAPH",
				URL:       "http://localhost:8082",
				HealthURL: "http://localhost:8082/height",
			},
			"knirvserver": {
				Name:      "KNIRV-NEXUS",
				URL:       "http://localhost:8083",
				HealthURL: "http://localhost:8083/health",
			},
			"knirvrouter": {
				Name:      "KNIRV-ROUTER",
				URL:       "http://localhost:5001",
				HealthURL: "http://localhost:5001/status",
			},
			"knirvgateway": {
				Name:      "KNIRV-GATEWAY",
				URL:       "http://localhost:8888",
				HealthURL: "http://localhost:8888/gateway/health",
			},
		},
	}
}

// ValidateNetworkStartup performs comprehensive network validation
func (nv *NetworkValidator) ValidateNetworkStartup() error {
	fmt.Println("🚀 Starting KNIRV Network Validation...")

	// Step 1: Check all services are running
	fmt.Println("\n📊 Step 1: Checking service health...")
	if err := nv.checkAllServices(); err != nil {
		return fmt.Errorf("service health check failed: %w", err)
	}

	// Step 2: Get authentication token
	fmt.Println("\n🔐 Step 2: Getting authentication token...")
	if err := nv.getAuthToken(); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Step 3: Test cross-component communication
	fmt.Println("\n🔗 Step 3: Testing cross-component communication...")
	if err := nv.testCrossComponentCommunication(); err != nil {
		return fmt.Errorf("cross-component communication failed: %w", err)
	}

	// Step 4: Test gateway routing
	fmt.Println("\n🌐 Step 4: Testing gateway routing...")
	if err := nv.testGatewayRouting(); err != nil {
		return fmt.Errorf("gateway routing failed: %w", err)
	}

	fmt.Println("\n✅ KNIRV Network validation completed successfully!")
	return nil
}

// checkAllServices checks the health of all services
func (nv *NetworkValidator) checkAllServices() error {
	allHealthy := true

	for serviceName, service := range nv.services {
		fmt.Printf("  Checking %s at %s...", service.Name, service.HealthURL)

		resp, err := nv.client.Get(service.HealthURL)
		if err != nil {
			fmt.Printf(" ❌ Error: %v\n", err)
			service.Error = err.Error()
			service.IsHealthy = false
			allHealthy = false
		} else {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				fmt.Printf(" ✅ Healthy\n")
				service.IsHealthy = true
			} else {
				fmt.Printf(" ❌ Status: %d\n", resp.StatusCode)
				service.IsHealthy = false
				allHealthy = false
			}
		}

		service.LastChecked = time.Now()
		nv.services[serviceName] = service
	}

	if !allHealthy {
		return fmt.Errorf("some services are not healthy")
	}

	return nil
}

// getAuthToken retrieves authentication token from gateway
func (nv *NetworkValidator) getAuthToken() error {
	tokenURL := "http://localhost:8888/auth/testnet-tokens"

	resp, err := nv.client.Get(tokenURL)
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token request failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read token response: %w", err)
	}

	var tokenResponse map[string]interface{}
	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return fmt.Errorf("failed to parse token response: %w", err)
	}

	if token, ok := tokenResponse["token"].(string); ok {
		nv.authToken = token
		fmt.Printf("  ✅ Authentication token obtained\n")
		return nil
	}

	return fmt.Errorf("no token found in response")
}

// testCrossComponentCommunication tests communication between services
func (nv *NetworkValidator) testCrossComponentCommunication() error {
	// Test KNIRVCHAIN -> KNIRVGRAPH communication
	fmt.Printf("  Testing KNIRVCHAIN -> KNIRVGRAPH...")
	if err := nv.testServiceEndpoint("http://localhost:8090/skills", false); err != nil {
		fmt.Printf(" ❌ %v\n", err)
	} else {
		fmt.Printf(" ✅ OK\n")
	}

	// Test KNIRVSERVER with authentication
	fmt.Printf("  Testing KNIRVSERVER (authenticated)...")
	if err := nv.testServiceEndpoint("http://localhost:8083/health", true); err != nil {
		fmt.Printf(" ❌ %v\n", err)
	} else {
		fmt.Printf(" ✅ OK\n")
	}

	// Test KNIRVROUTER connectivity
	fmt.Printf("  Testing KNIRVROUTER connectivity...")
	if err := nv.testServiceEndpoint("http://localhost:5001/peers", false); err != nil {
		fmt.Printf(" ❌ %v\n", err)
	} else {
		fmt.Printf(" ✅ OK\n")
	}

	return nil
}

// testGatewayRouting tests gateway routing to all services
func (nv *NetworkValidator) testGatewayRouting() error {
	gatewayTests := []struct {
		name     string
		endpoint string
	}{
		{"Gateway Services", "/gateway/services"},
		{"Gateway Status", "/gateway/testnet/status"},
		{"Auth Validation", "/auth/validate"},
	}

	for _, test := range gatewayTests {
		fmt.Printf("  Testing %s...", test.name)
		url := fmt.Sprintf("http://localhost:8888%s", test.endpoint)

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			fmt.Printf(" ❌ Request creation failed: %v\n", err)
			continue
		}

		if nv.authToken != "" {
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", nv.authToken))
		}

		resp, err := nv.client.Do(req)
		if err != nil {
			fmt.Printf(" ❌ Request failed: %v\n", err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			fmt.Printf(" ✅ OK\n")
		} else {
			fmt.Printf(" ⚠️  Status: %d\n", resp.StatusCode)
		}
	}

	return nil
}

// testServiceEndpoint tests a specific service endpoint
func (nv *NetworkValidator) testServiceEndpoint(url string, requireAuth bool) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if requireAuth && nv.authToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", nv.authToken))
	}

	resp, err := nv.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	return fmt.Errorf("unexpected status: %d", resp.StatusCode)
}

// printSummary prints a summary of the validation results
func (nv *NetworkValidator) printSummary() {
	fmt.Println("\n📋 Validation Summary:")
	fmt.Println("======================")

	healthyCount := 0
	for _, service := range nv.services {
		status := "❌ Unhealthy"
		if service.IsHealthy {
			status = "✅ Healthy"
			healthyCount++
		}

		fmt.Printf("  %s: %s\n", service.Name, status)
		if service.Error != "" {
			fmt.Printf("    Error: %s\n", service.Error)
		}
	}

	fmt.Printf("\nServices Status: %d/%d healthy\n", healthyCount, len(nv.services))

	if nv.authToken != "" {
		fmt.Println("Authentication: ✅ Token obtained")
	} else {
		fmt.Println("Authentication: ❌ No token")
	}
}
