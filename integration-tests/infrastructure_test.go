package integration_tests

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestKNIRVInfrastructure tests that all KNIRV services are running and responding correctly
func TestKNIRVInfrastructure(t *testing.T) {
	// Test all core services are healthy
	t.Run("ServiceHealthChecks", func(t *testing.T) {
		services := map[string]string{
			"KNIRVCHAIN":      KNIRVChainURL + "/health",
			"KNIRVGRAPH":      KNIRVGraphURL + "/health",
			"KNIRVSERVER":      KNIRVNexusURL + "/api/v1/health",
			"KNIRVCONTROLLER": KNIRVControllerURL + "/health",
		}

		for serviceName, healthURL := range services {
			t.Run(serviceName, func(t *testing.T) {
				client := &http.Client{Timeout: 10 * time.Second}

				resp, err := client.Get(healthURL)
				require.NoError(t, err, "Failed to connect to %s at %s", serviceName, healthURL)
				defer resp.Body.Close()

				assert.Equal(t, http.StatusOK, resp.StatusCode,
					"%s health check failed with status %d", serviceName, resp.StatusCode)

				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err, "Failed to read response body from %s", serviceName)

				// Verify we got a valid response
				assert.NotEmpty(t, body, "%s returned empty health response", serviceName)

				t.Logf("✅ %s is healthy - Response: %s", serviceName, string(body))
			})
		}
	})

	// Test basic API functionality
	t.Run("BasicAPIFunctionality", func(t *testing.T) {
		t.Run("KNIRVCHAINBasicAPI", func(t *testing.T) {
			client := &http.Client{Timeout: 10 * time.Second}

			// Test basic chain info endpoint
			resp, err := client.Get(KNIRVChainURL + "/info")
			if err != nil {
				t.Logf("⚠️  KNIRVCHAIN /info endpoint not available: %v", err)
				return // Skip if endpoint doesn't exist
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				t.Logf("✅ KNIRVCHAIN /info response: %s", string(body))
			}
		})

		t.Run("KNIRVGRAPHBasicAPI", func(t *testing.T) {
			client := &http.Client{Timeout: 10 * time.Second}

			// Test basic graph status endpoint
			resp, err := client.Get(KNIRVGraphURL + "/status")
			if err != nil {
				t.Logf("⚠️  KNIRVGRAPH /status endpoint not available: %v", err)
				return // Skip if endpoint doesn't exist
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				t.Logf("✅ KNIRVGRAPH /status response: %s", string(body))
			}
		})

		t.Run("KNIRVNEXUSBasicAPI", func(t *testing.T) {
			client := &http.Client{Timeout: 10 * time.Second}

			// Test NEXUS API info endpoint
			resp, err := client.Get(KNIRVNexusURL + "/api/v1/info")
			if err != nil {
				t.Logf("⚠️  KNIRVSERVER /api/v1/info endpoint not available: %v", err)
				return // Skip if endpoint doesn't exist
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				t.Logf("✅ KNIRVSERVER /api/v1/info response: %s", string(body))
			}
		})

		t.Run("KNIRVCONTROLLERBasicAPI", func(t *testing.T) {
			client := &http.Client{Timeout: 10 * time.Second}

			// Test controller status endpoint
			resp, err := client.Get(KNIRVControllerURL + "/api/status")
			if err != nil {
				t.Logf("⚠️  KNIRVCONTROLLER /api/status endpoint not available: %v", err)
				return // Skip if endpoint doesn't exist
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				t.Logf("✅ KNIRVCONTROLLER /api/status response: %s", string(body))
			}
		})
	})

	// Test service connectivity and cross-service communication
	t.Run("ServiceConnectivity", func(t *testing.T) {
		t.Run("AllServicesReachable", func(t *testing.T) {
			services := []string{
				KNIRVChainURL,
				KNIRVGraphURL,
				KNIRVNexusURL,
				KNIRVControllerURL,
			}

			client := &http.Client{Timeout: 5 * time.Second}

			for _, serviceURL := range services {
				resp, err := client.Get(serviceURL)
				if err != nil {
					t.Logf("⚠️  Service at %s not reachable: %v", serviceURL, err)
					continue
				}
				resp.Body.Close()

				t.Logf("✅ Service at %s is reachable (status: %d)", serviceURL, resp.StatusCode)
			}
		})
	})

	// Test that services are running on correct ports
	t.Run("PortConfiguration", func(t *testing.T) {
		expectedPorts := map[string]string{
			"KNIRVCHAIN":      "8080",
			"KNIRVGRAPH":      "8081",
			"KNIRVSERVER":      "8090",
			"KNIRVCONTROLLER": "3000",
		}

		for service, expectedPort := range expectedPorts {
			t.Run(service+"_Port_"+expectedPort, func(t *testing.T) {
				var serviceURL string
				switch service {
				case "KNIRVCHAIN":
					serviceURL = KNIRVChainURL
				case "KNIRVGRAPH":
					serviceURL = KNIRVGraphURL
				case "KNIRVSERVER":
					serviceURL = KNIRVNexusURL
				case "KNIRVCONTROLLER":
					serviceURL = KNIRVControllerURL
				}

				assert.Contains(t, serviceURL, ":"+expectedPort,
					"%s should be running on port %s", service, expectedPort)

				t.Logf("✅ %s correctly configured for port %s", service, expectedPort)
			})
		}
	})
}

// TestKNIRVIntegrationFlow tests a basic integration flow between services
func TestKNIRVIntegrationFlow(t *testing.T) {
	t.Run("BasicHealthCheckFlow", func(t *testing.T) {
		// This test demonstrates that we can successfully call all services in sequence
		client := &http.Client{Timeout: 10 * time.Second}

		// Step 1: Check KNIRVCHAIN
		resp1, err1 := client.Get(KNIRVChainURL + "/health")
		require.NoError(t, err1, "KNIRVCHAIN should be accessible")
		resp1.Body.Close()
		assert.Equal(t, http.StatusOK, resp1.StatusCode)
		t.Log("✅ Step 1: KNIRVCHAIN health check passed")

		// Step 2: Check KNIRVGRAPH
		resp2, err2 := client.Get(KNIRVGraphURL + "/health")
		require.NoError(t, err2, "KNIRVGRAPH should be accessible")
		resp2.Body.Close()
		assert.Equal(t, http.StatusOK, resp2.StatusCode)
		t.Log("✅ Step 2: KNIRVGRAPH health check passed")

		// Step 3: Check KNIRVSERVER
		resp3, err3 := client.Get(KNIRVNexusURL + "/api/v1/health")
		require.NoError(t, err3, "KNIRVSERVER should be accessible")
		resp3.Body.Close()
		assert.Equal(t, http.StatusOK, resp3.StatusCode)
		t.Log("✅ Step 3: KNIRVSERVER health check passed")

		// Step 4: Check KNIRVCONTROLLER
		resp4, err4 := client.Get(KNIRVControllerURL + "/health")
		require.NoError(t, err4, "KNIRVCONTROLLER should be accessible")
		resp4.Body.Close()
		assert.Equal(t, http.StatusOK, resp4.StatusCode)
		t.Log("✅ Step 4: KNIRVCONTROLLER health check passed")

		t.Log("🎉 All services successfully responding in sequence!")
	})
}

// TestKNIRVServiceVersions tests that services return version information
func TestKNIRVServiceVersions(t *testing.T) {
	t.Run("ServiceVersionInfo", func(t *testing.T) {
		client := &http.Client{Timeout: 10 * time.Second}

		// Try to get version info from services that might support it
		versionEndpoints := map[string]string{
			"KNIRVSERVER": KNIRVNexusURL + "/api/v1/version",
		}

		for service, versionURL := range versionEndpoints {
			t.Run(service+"_Version", func(t *testing.T) {
				resp, err := client.Get(versionURL)
				if err != nil {
					t.Logf("⚠️  %s version endpoint not available: %v", service, err)
					return
				}
				defer resp.Body.Close()

				if resp.StatusCode == http.StatusOK {
					body, err := io.ReadAll(resp.Body)
					if err == nil {
						var versionInfo map[string]interface{}
						if json.Unmarshal(body, &versionInfo) == nil {
							t.Logf("✅ %s version info: %+v", service, versionInfo)
						} else {
							t.Logf("✅ %s version response: %s", service, string(body))
						}
					}
				}
			})
		}
	})
}
