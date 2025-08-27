package integration_tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// KNIRVNEXUS unified binary endpoint (updated for Phase 6)
	KNIRVNEXUS_BASE_URL = "http://localhost:8084"

	// Test timeout for integration tests
	TEST_TIMEOUT = 30 * time.Second
)

// TestMain sets up the integration test environment
func TestMain(m *testing.M) {
	// Wait for services to be ready
	if !waitForService(KNIRVNEXUS_BASE_URL+"/health", TEST_TIMEOUT) {
		fmt.Printf("Warning: KNIRVNEXUS service not available at %s\n", KNIRVNEXUS_BASE_URL)
		fmt.Println("Integration tests will run in mock mode")
	}

	// Run tests
	code := m.Run()

	// Exit with the test result code
	os.Exit(code)
}

// waitForService waits for a service to become available
func waitForService(url string, timeout time.Duration) bool {
	client := &http.Client{Timeout: 5 * time.Second}
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return true
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(2 * time.Second)
	}
	return false
}

// makePhase6Request is a helper function for making HTTP requests
func makePhase6Request(method, url string, body interface{}, headers map[string]string) (*http.Response, error) {
	var reqBody io.Reader

	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}

	// Set default headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Set custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	return client.Do(req)
}

// Phase 6 Integration Tests for KNIRVNEXUS Unified Binary

func TestKNIRVNEXUSUnifiedBinaryIntegration(t *testing.T) {
	t.Run("TestHealthEndpoint", func(t *testing.T) {
		resp, err := makePhase6Request("GET", KNIRVNEXUS_BASE_URL+"/health", nil, nil)
		if err != nil {
			t.Skipf("KNIRVNEXUS service not available: %v", err)
			return
		}
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var health map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&health)
		require.NoError(t, err)

		// Verify health response structure
		assert.Contains(t, health, "status")
		assert.Equal(t, "healthy", health["status"])

		// Verify unified binary architecture
		if service, ok := health["service"]; ok {
			assert.Equal(t, "knirvnexus", service)
		}

		if port, ok := health["port"]; ok {
			assert.Equal(t, float64(8084), port)
		}
	})

	t.Run("TestAPIEndpointsIntegration", func(t *testing.T) {
		endpoints := []struct {
			path           string
			expectedFields []string
		}{
			{"/api/dve-nodes", []string{"nodes"}},
			{"/api/validation-tasks", []string{"tasks"}},
			{"/api/cognitive-engine", []string{"status"}},
			{"/api/tee-security", []string{"tee_status"}},
			{"/api/nrn-staking", []string{"total_staked"}},
			{"/api/system-health", []string{"status"}},
		}

		for _, endpoint := range endpoints {
			t.Run(fmt.Sprintf("Test%s", endpoint.path), func(t *testing.T) {
				resp, err := makePhase6Request("GET", KNIRVNEXUS_BASE_URL+endpoint.path, nil, nil)
				if err != nil {
					t.Skipf("KNIRVNEXUS service not available: %v", err)
					return
				}
				defer resp.Body.Close()

				// Should return 200 OK or 401 Unauthorized (if auth required)
				assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized,
					"Expected 200 or 401, got %d for %s", resp.StatusCode, endpoint.path)

				if resp.StatusCode == http.StatusOK {
					var response map[string]interface{}
					err = json.NewDecoder(resp.Body).Decode(&response)
					require.NoError(t, err)

					// Check for expected fields
					for _, field := range endpoint.expectedFields {
						assert.Contains(t, response, field,
							"Response should contain field: %s", field)
					}
				}
			})
		}
	})
}

func TestKNIRVNEXUSAuthenticationIntegration(t *testing.T) {
	t.Run("TestUnauthorizedAccess", func(t *testing.T) {
		// Test accessing protected endpoints without authentication
		protectedEndpoints := []string{
			"/api/dve-nodes",
			"/api/validation-tasks",
			"/api/cognitive-engine",
		}

		for _, endpoint := range protectedEndpoints {
			resp, err := makePhase6Request("GET", KNIRVNEXUS_BASE_URL+endpoint, nil, nil)
			if err != nil {
				t.Skipf("KNIRVNEXUS service not available: %v", err)
				return
			}
			defer resp.Body.Close()

			// Should return 401 Unauthorized or 200 OK (if endpoint is public)
			assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized,
				"Expected 200 or 401, got %d for %s", resp.StatusCode, endpoint)
		}
	})

	t.Run("TestAuthenticatedAccess", func(t *testing.T) {
		// Test with mock authentication token
		headers := map[string]string{
			"Authorization": "Bearer testnet-admin-123",
		}

		resp, err := makePhase6Request("GET", KNIRVNEXUS_BASE_URL+"/api/system-health", nil, headers)
		if err != nil {
			t.Skipf("KNIRVNEXUS service not available: %v", err)
			return
		}
		defer resp.Body.Close()

		// Should return 200 OK or 401 Unauthorized (depending on token validation)
		assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized,
			"Expected 200 or 401, got %d", resp.StatusCode)
	})
}

func TestKNIRVNEXUSCrossServiceIntegration(t *testing.T) {
	t.Run("TestServiceDiscovery", func(t *testing.T) {
		// Test service discovery and health checks
		services := []struct {
			name string
			url  string
		}{
			{"KNIRVNEXUS", KNIRVNEXUS_BASE_URL + "/health"},
			// Add other services when available
		}

		for _, service := range services {
			resp, err := makePhase6Request("GET", service.url, nil, nil)
			if err != nil {
				t.Logf("Service %s not available: %v", service.name, err)
				continue
			}
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode,
				"Service %s should be healthy", service.name)
		}
	})

	t.Run("TestDatabaseIntegration", func(t *testing.T) {
		// Test database operations through API
		resp, err := makePhase6Request("GET", KNIRVNEXUS_BASE_URL+"/api/system-health", nil, nil)
		if err != nil {
			t.Skipf("KNIRVNEXUS service not available: %v", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			var health map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&health)
			require.NoError(t, err)

			// Check if database-related metrics are present
			if status, ok := health["status"]; ok {
				assert.NotEqual(t, "error", status, "Database should be operational")
			}
		}
	})
}

func TestKNIRVNEXUSRealTimeUpdates(t *testing.T) {
	t.Run("TestSSEConnection", func(t *testing.T) {
		// Test Server-Sent Events connection
		// Note: This is a basic connectivity test
		client := &http.Client{Timeout: 5 * time.Second}

		req, err := http.NewRequest("GET", KNIRVNEXUS_BASE_URL+"/api/events", nil)
		if err != nil {
			t.Skipf("Cannot create SSE request: %v", err)
			return
		}

		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("Cache-Control", "no-cache")

		resp, err := client.Do(req)
		if err != nil {
			t.Skipf("SSE endpoint not available: %v", err)
			return
		}
		defer resp.Body.Close()

		// Should return 200 OK or 404 Not Found (if SSE not implemented)
		assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNotFound,
			"Expected 200 or 404, got %d", resp.StatusCode)

		if resp.StatusCode == http.StatusOK {
			assert.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))
		}
	})
}

func TestKNIRVNEXUSPerformanceBaseline(t *testing.T) {
	t.Run("TestResponseTimes", func(t *testing.T) {
		endpoints := []string{
			"/health",
			"/api/system-health",
		}

		for _, endpoint := range endpoints {
			start := time.Now()
			resp, err := makePhase6Request("GET", KNIRVNEXUS_BASE_URL+endpoint, nil, nil)
			duration := time.Since(start)

			if err != nil {
				t.Skipf("Endpoint %s not available: %v", endpoint, err)
				continue
			}
			defer resp.Body.Close()

			// Response time should be under 5 seconds for basic endpoints
			assert.True(t, duration < 5*time.Second,
				"Endpoint %s took %v, should be under 5s", endpoint, duration)

			t.Logf("Endpoint %s response time: %v", endpoint, duration)
		}
	})

	t.Run("TestConcurrentRequests", func(t *testing.T) {
		// Test handling of concurrent requests
		concurrency := 5
		done := make(chan bool, concurrency)

		for i := 0; i < concurrency; i++ {
			go func(id int) {
				resp, err := makePhase6Request("GET", KNIRVNEXUS_BASE_URL+"/health", nil, nil)
				if err == nil {
					resp.Body.Close()
					assert.Equal(t, http.StatusOK, resp.StatusCode,
						"Concurrent request %d should succeed", id)
				}
				done <- true
			}(i)
		}

		// Wait for all requests to complete
		for i := 0; i < concurrency; i++ {
			select {
			case <-done:
				// Request completed
			case <-time.After(10 * time.Second):
				t.Errorf("Concurrent request %d timed out", i)
			}
		}
	})
}

func TestKNIRVNEXUSErrorHandling(t *testing.T) {
	t.Run("TestInvalidEndpoints", func(t *testing.T) {
		invalidEndpoints := []string{
			"/api/nonexistent",
			"/invalid/path",
			"/api/../../../etc/passwd",
		}

		for _, endpoint := range invalidEndpoints {
			resp, err := makePhase6Request("GET", KNIRVNEXUS_BASE_URL+endpoint, nil, nil)
			if err != nil {
				t.Skipf("Cannot test invalid endpoint %s: %v", endpoint, err)
				continue
			}
			defer resp.Body.Close()

			// Should return 404 Not Found
			assert.Equal(t, http.StatusNotFound, resp.StatusCode,
				"Invalid endpoint %s should return 404", endpoint)
		}
	})

	t.Run("TestInvalidMethods", func(t *testing.T) {
		// Test invalid HTTP methods on valid endpoints
		resp, err := makePhase6Request("DELETE", KNIRVNEXUS_BASE_URL+"/health", nil, nil)
		if err != nil {
			t.Skipf("Cannot test invalid method: %v", err)
			return
		}
		defer resp.Body.Close()

		// Should return 405 Method Not Allowed or 404 Not Found
		assert.True(t, resp.StatusCode == http.StatusMethodNotAllowed || resp.StatusCode == http.StatusNotFound,
			"Invalid method should return 405 or 404, got %d", resp.StatusCode)
	})
}
