package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// ServiceConfig represents configuration for a KNIRV service
type ServiceConfig struct {
	Name            string   `json:"name"`
	URL             string   `json:"url"`
	HealthEndpoint  string   `json:"health_endpoint"`
	APIEndpoints    []string `json:"api_endpoints"`
	Timeout         string   `json:"timeout"`
	RetryCount      int      `json:"retry_count"`
	AuthRequired    bool     `json:"auth_required"`
}

// ServiceDiscovery manages service discovery and health checks
type ServiceDiscovery struct {
	Services map[string]ServiceConfig
	AuthToken string
	Client   *http.Client
}

// NewServiceDiscovery creates a new service discovery instance
func NewServiceDiscovery() *ServiceDiscovery {
	return &ServiceDiscovery{
		Services: make(map[string]ServiceConfig),
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// LoadServices loads service configuration from environment variables
func (sd *ServiceDiscovery) LoadServices() error {
	// Load service URLs from environment
	services := map[string]ServiceConfig{
		"knirvoracle": {
			Name:           "KNIRV-ORACLE",
			URL:            getEnvOrDefault("KNIRVORACLE_URL", "http://localhost:1317"),
			HealthEndpoint: "/health",
			APIEndpoints:   []string{"/cosmos/bank/v1beta1/balances", "/nrn/mint", "/nrn/burn"},
			RetryCount:     3,
		},
		"knirvchain": {
			Name:           "KNIRVCHAIN",
			URL:            getEnvOrDefault("KNIRVCHAIN_URL", "http://localhost:8090"),
			HealthEndpoint: "/health",
			APIEndpoints:   []string{"/skills", "/skills/register", "/llm/inference"},
			RetryCount:     3,
		},
		"knirvgraph": {
			Name:           "KNIRVGRAPH",
			URL:            getEnvOrDefault("KNIRVGRAPH_URL", "http://localhost:8082"),
			HealthEndpoint: "/height",
			APIEndpoints:   []string{"/graphql", "/nodes", "/edges"},
			RetryCount:     3,
		},
		"knirvnexus": {
			Name:           "KNIRV-NEXUS",
			URL:            getEnvOrDefault("KNIRVNEXUS_URL", "http://localhost:8083"),
			HealthEndpoint: "/health",
			APIEndpoints:   []string{"/validate", "/nodes", "/tasks"},
			RetryCount:     3,
			AuthRequired:   true,
		},
		"knirvrouter": {
			Name:           "KNIRV-ROUTER",
			URL:            getEnvOrDefault("KNIRVROUTER_URL", "http://localhost:5001"),
			HealthEndpoint: "/status",
			APIEndpoints:   []string{"/connectivity/proof", "/nrn/mint", "/peers"},
			RetryCount:     3,
		},
		"knirvgateway": {
			Name:           "KNIRV-GATEWAY",
			URL:            getEnvOrDefault("KNIRVGATEWAY_URL", "http://localhost:8888"),
			HealthEndpoint: "/gateway/health",
			APIEndpoints:   []string{"/gateway/services", "/auth/testnet-tokens"},
			RetryCount:     3,
		},
	}
	
	sd.Services = services
	return nil
}

// GetAuthToken retrieves authentication token from the gateway
func (sd *ServiceDiscovery) GetAuthToken() error {
	gatewayURL := sd.Services["knirvgateway"].URL
	tokenURL := fmt.Sprintf("%s/auth/testnet-tokens", gatewayURL)
	
	resp, err := sd.Client.Get(tokenURL)
	if err != nil {
		return fmt.Errorf("failed to get auth token: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("auth token request failed with status: %d", resp.StatusCode)
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read auth response: %w", err)
	}
	
	var tokenResponse map[string]interface{}
	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return fmt.Errorf("failed to parse auth response: %w", err)
	}
	
	if token, ok := tokenResponse["token"].(string); ok {
		sd.AuthToken = token
		return nil
	}
	
	return fmt.Errorf("no token found in auth response")
}

// CheckServiceHealth checks if a service is healthy
func (sd *ServiceDiscovery) CheckServiceHealth(serviceName string) (bool, error) {
	service, exists := sd.Services[serviceName]
	if !exists {
		return false, fmt.Errorf("service %s not found", serviceName)
	}
	
	healthURL := fmt.Sprintf("%s%s", service.URL, service.HealthEndpoint)
	
	req, err := http.NewRequest("GET", healthURL, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create health check request: %w", err)
	}
	
	// Add auth token if required
	if service.AuthRequired && sd.AuthToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", sd.AuthToken))
	}
	
	resp, err := sd.Client.Do(req)
	if err != nil {
		return false, fmt.Errorf("health check request failed: %w", err)
	}
	defer resp.Body.Close()
	
	return resp.StatusCode == http.StatusOK, nil
}

// CheckAllServices checks health of all services
func (sd *ServiceDiscovery) CheckAllServices() map[string]bool {
	results := make(map[string]bool)
	
	for serviceName := range sd.Services {
		healthy, err := sd.CheckServiceHealth(serviceName)
		if err != nil {
			fmt.Printf("Error checking %s: %v\n", serviceName, err)
			results[serviceName] = false
		} else {
			results[serviceName] = healthy
		}
	}
	
	return results
}

// WaitForService waits for a service to become healthy
func (sd *ServiceDiscovery) WaitForService(serviceName string, timeout time.Duration) error {
	start := time.Now()
	
	for time.Since(start) < timeout {
		healthy, err := sd.CheckServiceHealth(serviceName)
		if err == nil && healthy {
			return nil
		}
		
		time.Sleep(2 * time.Second)
	}
	
	return fmt.Errorf("service %s did not become healthy within %v", serviceName, timeout)
}

// MakeAuthenticatedRequest makes an authenticated request to a service
func (sd *ServiceDiscovery) MakeAuthenticatedRequest(serviceName, endpoint string) (*http.Response, error) {
	service, exists := sd.Services[serviceName]
	if !exists {
		return nil, fmt.Errorf("service %s not found", serviceName)
	}
	
	url := fmt.Sprintf("%s%s", service.URL, endpoint)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Add auth token if required
	if service.AuthRequired && sd.AuthToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", sd.AuthToken))
	}
	
	return sd.Client.Do(req)
}

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
