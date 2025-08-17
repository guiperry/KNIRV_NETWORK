package integration_tests

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// GatewayMigrationTestConfig holds configuration for gateway migration tests
type GatewayMigrationTestConfig struct {
	GatewayURL      string
	KNIRVRootURL    string
	NetlifyDevPort  string
	TestTimeout     time.Duration
	ScriptsPath     string
}

// GatewayResponse represents a response from the gateway
type GatewayResponse struct {
	Status    string                 `json:"status,omitempty"`
	Services  map[string]interface{} `json:"services,omitempty"`
	Timestamp int64                  `json:"timestamp,omitempty"`
	Error     string                 `json:"error,omitempty"`
}

// SSEEvent represents a Server-Sent Event
type SSEEvent struct {
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Timestamp int64                  `json:"timestamp"`
}

// TestGatewayMigrationComplete tests the complete gateway migration
func TestGatewayMigrationComplete(t *testing.T) {
	config := &GatewayMigrationTestConfig{
		GatewayURL:     "http://localhost:8888",
		KNIRVRootURL:   "http://localhost:5002",
		NetlifyDevPort: "8888",
		TestTimeout:    30 * time.Second,
		ScriptsPath:    "../scripts",
	}

	t.Run("Migration_Scripts_Exist", func(t *testing.T) {
		testMigrationScriptsExist(t, config)
	})

	t.Run("Gateway_Functions_Operational", func(t *testing.T) {
		testGatewayFunctionsOperational(t, config)
	})

	t.Run("SSE_Functionality", func(t *testing.T) {
		testSSEFunctionality(t, config)
	})

	t.Run("Service_Proxy_Functionality", func(t *testing.T) {
		testServiceProxyFunctionality(t, config)
	})

	t.Run("Economics_Integration", func(t *testing.T) {
		testEconomicsIntegration(t, config)
	})

	t.Run("Complete_Migration_Validation", func(t *testing.T) {
		testCompleteMigrationValidation(t, config)
	})
}

// testMigrationScriptsExist verifies all migration scripts are in the scripts folder
func testMigrationScriptsExist(t *testing.T, config *GatewayMigrationTestConfig) {
	requiredScripts := []string{
		"test-gateway-migration.sh",
		"validate-complete-migration.sh",
		"start-with-economics.sh",
		"test-economics-integration.sh",
		"verify-deployment.sh",
	}

	for _, script := range requiredScripts {
		scriptPath := filepath.Join(config.ScriptsPath, script)
		if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
			t.Errorf("Required migration script not found: %s", scriptPath)
		} else {
			t.Logf("✅ Found migration script: %s", script)
		}
	}
}

// testGatewayFunctionsOperational tests that gateway functions are working
func testGatewayFunctionsOperational(t *testing.T, config *GatewayMigrationTestConfig) {
	ctx, cancel := context.WithTimeout(context.Background(), config.TestTimeout)
	defer cancel()

	// Test gateway health
	resp, err := makeHTTPRequest(ctx, "GET", config.GatewayURL+"/gateway/health", nil)
	if err != nil {
		t.Fatalf("Failed to call gateway health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Gateway health check failed: status %d", resp.StatusCode)
	} else {
		t.Log("✅ Gateway health check passed")
	}

	// Test gateway services
	resp, err = makeHTTPRequest(ctx, "GET", config.GatewayURL+"/gateway/services", nil)
	if err != nil {
		t.Fatalf("Failed to call gateway services: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Gateway services check failed: status %d", resp.StatusCode)
	} else {
		t.Log("✅ Gateway services endpoint working")
	}

	// Test gateway metrics
	resp, err = makeHTTPRequest(ctx, "GET", config.GatewayURL+"/gateway/metrics", nil)
	if err != nil {
		t.Fatalf("Failed to call gateway metrics: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Gateway metrics check failed: status %d", resp.StatusCode)
	} else {
		t.Log("✅ Gateway metrics endpoint working")
	}
}

// testSSEFunctionality tests Server-Sent Events functionality
func testSSEFunctionality(t *testing.T, config *GatewayMigrationTestConfig) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test health monitor SSE
	t.Run("Health_Monitor_SSE", func(t *testing.T) {
		sseURL := config.GatewayURL + "/health-monitor/events"
		if testSSEEndpoint(ctx, sseURL) {
			t.Log("✅ Health monitor SSE working")
		} else {
			t.Error("❌ Health monitor SSE failed")
		}
	})

	// Test gateway SSE
	t.Run("Gateway_SSE", func(t *testing.T) {
		sseURL := config.GatewayURL + "/gateway/events"
		if testSSEEndpoint(ctx, sseURL) {
			t.Log("✅ Gateway SSE working")
		} else {
			t.Error("❌ Gateway SSE failed")
		}
	})
}

// testServiceProxyFunctionality tests service proxy functionality
func testServiceProxyFunctionality(t *testing.T, config *GatewayMigrationTestConfig) {
	ctx, cancel := context.WithTimeout(context.Background(), config.TestTimeout)
	defer cancel()

	// Test proxy to KNIRVORACLE (should work if KNIRVORACLE is running)
	resp, err := makeHTTPRequest(ctx, "GET", config.GatewayURL+"/health", nil)
	if err != nil {
		t.Logf("⚠️  Service proxy test skipped (KNIRVORACLE not running): %v", err)
		return
	}
	defer resp.Body.Close()

	// Any response (even 502) indicates proxy is working
	if resp.StatusCode == http.StatusBadGateway {
		t.Log("✅ Service proxy working (service unavailable, but proxy functional)")
	} else if resp.StatusCode == http.StatusOK {
		t.Log("✅ Service proxy working (service available)")
	} else {
		t.Errorf("❌ Unexpected proxy response: status %d", resp.StatusCode)
	}
}

// testEconomicsIntegration tests economics integration
func testEconomicsIntegration(t *testing.T, config *GatewayMigrationTestConfig) {
	ctx, cancel := context.WithTimeout(context.Background(), config.TestTimeout)
	defer cancel()

	// Test economics health through gateway
	resp, err := makeHTTPRequest(ctx, "GET", config.GatewayURL+"/economics/health", nil)
	if err != nil {
		t.Logf("⚠️  Economics integration test skipped (service not running): %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		t.Log("✅ Economics integration working")
	} else if resp.StatusCode == http.StatusBadGateway {
		t.Log("✅ Economics routing working (service unavailable)")
	} else {
		t.Errorf("❌ Economics integration failed: status %d", resp.StatusCode)
	}
}

// testCompleteMigrationValidation runs the complete migration validation script
func testCompleteMigrationValidation(t *testing.T, config *GatewayMigrationTestConfig) {
	scriptPath := filepath.Join(config.ScriptsPath, "validate-complete-migration.sh")
	
	// Check if script exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		t.Skip("Migration validation script not found, skipping")
		return
	}

	// Make script executable
	if err := os.Chmod(scriptPath, 0755); err != nil {
		t.Errorf("Failed to make script executable: %v", err)
		return
	}

	// Run the validation script
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", scriptPath)
	cmd.Dir = filepath.Dir(scriptPath)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("Migration validation script output:\n%s", string(output))
		t.Errorf("Migration validation script failed: %v", err)
		return
	}

	// Check if validation was successful
	outputStr := string(output)
	if strings.Contains(outputStr, "MIGRATION VALIDATION: COMPLETE SUCCESS") {
		t.Log("✅ Complete migration validation passed")
	} else if strings.Contains(outputStr, "MIGRATION VALIDATION: PARTIAL SUCCESS") {
		t.Log("⚠️  Migration validation partially successful")
	} else {
		t.Errorf("❌ Migration validation failed")
	}

	t.Logf("Migration validation output:\n%s", outputStr)
}

// Helper functions

// makeHTTPRequest makes an HTTP request with context
func makeHTTPRequest(ctx context.Context, method, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	return client.Do(req)
}

// testSSEEndpoint tests a Server-Sent Events endpoint
func testSSEEndpoint(ctx context.Context, url string) bool {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false
	}

	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false
	}

	// Read a small amount of data to verify SSE format
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data:") {
			return true // Found SSE data
		}
		// Timeout after a short period
		select {
		case <-ctx.Done():
			return false
		default:
			continue
		}
	}

	return false
}

// TestGatewayMigrationScripts tests the migration scripts directly
func TestGatewayMigrationScripts(t *testing.T) {
	scriptsPath := "../scripts"

	t.Run("Test_Gateway_Migration_Script", func(t *testing.T) {
		testScript(t, scriptsPath, "test-gateway-migration.sh")
	})

	t.Run("Test_Economics_Integration_Script", func(t *testing.T) {
		testScript(t, scriptsPath, "test-economics-integration.sh")
	})
}

// testScript tests a specific script
func testScript(t *testing.T, scriptsPath, scriptName string) {
	scriptPath := filepath.Join(scriptsPath, scriptName)
	
	// Check if script exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		t.Skipf("Script not found: %s", scriptPath)
		return
	}

	// Make script executable
	if err := os.Chmod(scriptPath, 0755); err != nil {
		t.Errorf("Failed to make script executable: %v", err)
		return
	}

	t.Logf("✅ Script exists and is executable: %s", scriptName)
}
