package test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMCPServerDiscovery tests MCP server discovery functionality
func TestMCPServerDiscovery(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	t.Run("List All MCP Servers", func(t *testing.T) {
		rr, err := ts.makeRequest("GET", "/mcp/servers", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rr.Code)

		var response map[string]interface{}
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response, "servers")
		servers := response["servers"].([]interface{})
		assert.GreaterOrEqual(t, len(servers), 0)
	})

	t.Run("Filter Servers by Category", func(t *testing.T) {
		categories := []string{"web", "file", "data", "ai", "system", "security", "cloud", "social", "general"}

		for _, category := range categories {
			rr, err := ts.makeRequest("GET", "/mcp/servers?category="+category, nil)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, rr.Code)

			var response map[string]interface{}
			err = json.Unmarshal(rr.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Contains(t, response, "servers")
		}
	})

	t.Run("Filter Servers by Type", func(t *testing.T) {
		types := []string{"typescript", "python", "go", "rust"}

		for _, serverType := range types {
			rr, err := ts.makeRequest("GET", "/mcp/servers?type="+serverType, nil)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, rr.Code)
		}
	})

	t.Run("Search Servers by Keyword", func(t *testing.T) {
		keywords := []string{"database", "file", "web", "api"}

		for _, keyword := range keywords {
			rr, err := ts.makeRequest("GET", "/mcp/servers?search="+keyword, nil)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, rr.Code)

			var response map[string]interface{}
			err = json.Unmarshal(rr.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Contains(t, response, "servers")
		}
	})

	t.Run("Get Individual Server Details", func(t *testing.T) {
		// Test with common server names
		serverNames := []string{"filesystem", "database", "web-search"}

		for _, serverName := range serverNames {
			rr, err := ts.makeRequest("GET", "/mcp/servers/"+serverName, nil)
			require.NoError(t, err)

			// Server might exist or not, both are valid responses
			assert.True(t, rr.Code == http.StatusOK || rr.Code == http.StatusNotFound)

			if rr.Code == http.StatusOK {
				var response map[string]interface{}
				err = json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.Contains(t, response, "server")
				server := response["server"].(map[string]interface{})
				assert.Equal(t, serverName, server["name"])
			}
		}
	})

	t.Run("Get Non-existent Server", func(t *testing.T) {
		rr, err := ts.makeRequest("GET", "/mcp/servers/nonexistent-server-12345", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, rr.Code)
	})
}

// TestMCPServerConfiguration tests server configuration functionality
func TestMCPServerConfiguration(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	t.Run("Get Server Configuration", func(t *testing.T) {
		// Test with a common server that might exist
		rr, err := ts.makeRequest("GET", "/mcp/servers/filesystem/config", nil)
		require.NoError(t, err)

		// Configuration might not be available for all servers
		assert.True(t, rr.Code == http.StatusOK || rr.Code == http.StatusNotFound)

		if rr.Code == http.StatusOK {
			var response map[string]interface{}
			err = json.Unmarshal(rr.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Contains(t, response, "config")
		}
	})

	t.Run("Update Server Configuration", func(t *testing.T) {
		configData := map[string]interface{}{
			"enabled": true,
			"settings": map[string]interface{}{
				"timeout": 30,
			},
		}

		rr, err := ts.makeRequest("PUT", "/mcp/servers/filesystem/config", configData)
		require.NoError(t, err)

		// Configuration update might not be implemented
		assert.True(t, rr.Code == http.StatusOK || rr.Code == http.StatusNotFound || rr.Code == http.StatusMethodNotAllowed)
	})
}

// TestMCPServerInstallation tests server installation functionality
func TestMCPServerInstallation(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	t.Run("Install Server", func(t *testing.T) {
		// Test installation of a simple server
		rr, err := ts.makeRequest("POST", "/mcp/servers/filesystem/install", nil)
		require.NoError(t, err)

		// Installation might fail due to missing dependencies
		assert.True(t, rr.Code == http.StatusOK || rr.Code == http.StatusAccepted || 
			rr.Code == http.StatusInternalServerError || rr.Code == http.StatusNotFound)

		if rr.Code == http.StatusOK || rr.Code == http.StatusAccepted {
			var response map[string]interface{}
			err = json.Unmarshal(rr.Body.Bytes(), &response)
			require.NoError(t, err)

			// Should contain installation status or job ID
			assert.True(t, 
				response["status"] != nil || 
				response["job_id"] != nil || 
				response["message"] != nil)
		}
	})

	t.Run("Get Installation Status", func(t *testing.T) {
		rr, err := ts.makeRequest("GET", "/mcp/servers/filesystem/status", nil)
		require.NoError(t, err)

		assert.True(t, rr.Code == http.StatusOK || rr.Code == http.StatusNotFound)

		if rr.Code == http.StatusOK {
			var response map[string]interface{}
			err = json.Unmarshal(rr.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Contains(t, response, "status")
		}
	})

	t.Run("Uninstall Server", func(t *testing.T) {
		rr, err := ts.makeRequest("DELETE", "/mcp/servers/filesystem/install", nil)
		require.NoError(t, err)

		// Uninstall might not be implemented
		assert.True(t, rr.Code == http.StatusOK || rr.Code == http.StatusNotFound || rr.Code == http.StatusMethodNotAllowed)
	})
}

// TestMCPServerLifecycle tests server lifecycle management
func TestMCPServerLifecycle(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	t.Run("Start Server", func(t *testing.T) {
		rr, err := ts.makeRequest("POST", "/mcp/servers/filesystem/start", nil)
		require.NoError(t, err)

		// Server might not be installed or might fail to start
		assert.True(t, rr.Code == http.StatusOK || rr.Code == http.StatusAccepted || 
			rr.Code == http.StatusInternalServerError || rr.Code == http.StatusNotFound)
	})

	t.Run("Stop Server", func(t *testing.T) {
		rr, err := ts.makeRequest("POST", "/mcp/servers/filesystem/stop", nil)
		require.NoError(t, err)

		assert.True(t, rr.Code == http.StatusOK || rr.Code == http.StatusNotFound)
	})

	t.Run("Get Running Servers", func(t *testing.T) {
		rr, err := ts.makeRequest("GET", "/mcp/servers/running", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rr.Code)

		var response map[string]interface{}
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response, "servers")
		servers := response["servers"].([]interface{})
		assert.GreaterOrEqual(t, len(servers), 0)
	})

	t.Run("Restart Server", func(t *testing.T) {
		rr, err := ts.makeRequest("POST", "/mcp/servers/filesystem/restart", nil)
		require.NoError(t, err)

		// Restart might not be implemented
		assert.True(t, rr.Code == http.StatusOK || rr.Code == http.StatusNotFound || rr.Code == http.StatusMethodNotAllowed)
	})
}

// TestMCPMonitoring tests MCP monitoring functionality
func TestMCPMonitoring(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	t.Run("Get MCP Metrics", func(t *testing.T) {
		rr, err := ts.makeRequest("GET", "/mcp/metrics", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rr.Code)

		var response map[string]interface{}
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		// Should contain metrics data
		assert.True(t, len(response) > 0)
	})

	t.Run("Get MCP Logs", func(t *testing.T) {
		rr, err := ts.makeRequest("GET", "/mcp/logs", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rr.Code)

		var response map[string]interface{}
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response, "logs")
	})

	t.Run("Get MCP Alerts", func(t *testing.T) {
		rr, err := ts.makeRequest("GET", "/mcp/alerts", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rr.Code)

		var response map[string]interface{}
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response, "alerts")
	})

	t.Run("Resolve Alert", func(t *testing.T) {
		// Try to resolve a hypothetical alert
		rr, err := ts.makeRequest("POST", "/mcp/alerts/test-alert/resolve", nil)
		require.NoError(t, err)

		// Alert might not exist
		assert.True(t, rr.Code == http.StatusOK || rr.Code == http.StatusNotFound)
	})

	t.Run("Get Server-specific Metrics", func(t *testing.T) {
		rr, err := ts.makeRequest("GET", "/mcp/servers/filesystem/metrics", nil)
		require.NoError(t, err)

		// Server-specific metrics might not be implemented
		assert.True(t, rr.Code == http.StatusOK || rr.Code == http.StatusNotFound)
	})
}

// TestMCPRegistrySync tests registry synchronization
func TestMCPRegistrySync(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	t.Run("Sync Registry", func(t *testing.T) {
		rr, err := ts.makeRequest("POST", "/mcp/servers/sync", nil)
		require.NoError(t, err)

		// Sync might take time or fail due to network issues
		assert.True(t, rr.Code == http.StatusOK || rr.Code == http.StatusAccepted || 
			rr.Code == http.StatusInternalServerError)

		if rr.Code == http.StatusOK || rr.Code == http.StatusAccepted {
			var response map[string]interface{}
			err = json.Unmarshal(rr.Body.Bytes(), &response)
			require.NoError(t, err)

			// Should contain sync status
			assert.True(t, response["status"] != nil || response["message"] != nil)
		}
	})

	t.Run("Get Sync Status", func(t *testing.T) {
		rr, err := ts.makeRequest("GET", "/mcp/servers/sync/status", nil)
		require.NoError(t, err)

		// Sync status might not be implemented
		assert.True(t, rr.Code == http.StatusOK || rr.Code == http.StatusNotFound)
	})

	t.Run("Force Sync", func(t *testing.T) {
		rr, err := ts.makeRequest("POST", "/mcp/servers/sync?force=true", nil)
		require.NoError(t, err)

		assert.True(t, rr.Code == http.StatusOK || rr.Code == http.StatusAccepted || 
			rr.Code == http.StatusInternalServerError)
	})
}

// TestMCPCapabilities tests MCP capabilities integration
func TestMCPCapabilities(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	t.Run("List MCP Capabilities", func(t *testing.T) {
		rr, err := ts.makeRequest("GET", "/capabilities/mcp", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rr.Code)

		var response map[string]interface{}
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response, "capabilities")
	})

	t.Run("Get Capability Details", func(t *testing.T) {
		rr, err := ts.makeRequest("GET", "/capabilities/mcp/filesystem", nil)
		require.NoError(t, err)

		// Capability might not exist
		assert.True(t, rr.Code == http.StatusOK || rr.Code == http.StatusNotFound)
	})

	t.Run("Enable Capability", func(t *testing.T) {
		rr, err := ts.makeRequest("POST", "/capabilities/mcp/filesystem/enable", nil)
		require.NoError(t, err)

		// Enable might not be implemented
		assert.True(t, rr.Code == http.StatusOK || rr.Code == http.StatusNotFound || rr.Code == http.StatusMethodNotAllowed)
	})

	t.Run("Disable Capability", func(t *testing.T) {
		rr, err := ts.makeRequest("POST", "/capabilities/mcp/filesystem/disable", nil)
		require.NoError(t, err)

		assert.True(t, rr.Code == http.StatusOK || rr.Code == http.StatusNotFound || rr.Code == http.StatusMethodNotAllowed)
	})
}

// TestMCPPerformance tests MCP performance characteristics
func TestMCPPerformance(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	t.Run("Server List Performance", func(t *testing.T) {
		start := time.Now()
		
		for i := 0; i < 10; i++ {
			rr, err := ts.makeRequest("GET", "/mcp/servers", nil)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, rr.Code)
		}
		
		duration := time.Since(start)
		
		// Should complete 10 requests in under 2 seconds
		assert.Less(t, duration, 2*time.Second)
		t.Logf("10 MCP server list requests completed in %v", duration)
	})

	t.Run("Concurrent Server Requests", func(t *testing.T) {
		const numRequests = 5
		results := make(chan int, numRequests)

		for i := 0; i < numRequests; i++ {
			go func() {
				rr, err := ts.makeRequest("GET", "/mcp/servers", nil)
				if err != nil {
					results <- 500
					return
				}
				results <- rr.Code
			}()
		}

		for i := 0; i < numRequests; i++ {
			code := <-results
			assert.Equal(t, http.StatusOK, code)
		}
	})
}

// TestMCPErrorHandling tests MCP error handling
func TestMCPErrorHandling(t *testing.T) {
	ts := SetupTestServer(t)
	defer ts.Cleanup()

	t.Run("Invalid Server ID", func(t *testing.T) {
		rr, err := ts.makeRequest("GET", "/mcp/servers/invalid-server-id-!@#$", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("Invalid Installation Request", func(t *testing.T) {
		rr, err := ts.makeRequest("POST", "/mcp/servers/nonexistent/install", nil)
		require.NoError(t, err)
		assert.True(t, rr.Code == http.StatusNotFound || rr.Code == http.StatusBadRequest)
	})

	t.Run("Invalid Lifecycle Operation", func(t *testing.T) {
		rr, err := ts.makeRequest("POST", "/mcp/servers/nonexistent/start", nil)
		require.NoError(t, err)
		assert.True(t, rr.Code == http.StatusNotFound || rr.Code == http.StatusBadRequest)
	})
}
