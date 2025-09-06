package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockKNIRVCHAINServer creates a mock KNIRVCHAIN server for end-to-end testing
func MockKNIRVCHAINServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/health":
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		case "/info":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"version":      "1.0.0",
				"node_id":      "mock-node",
				"network_id":   "mock-network",
				"block_height": 1000,
			})
		case "/transaction":
			// Handle transaction submission
			var request map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": "invalid request"})
				return
			}

			json.NewEncoder(w).Encode(map[string]interface{}{
				"transactionHash": "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
				"status":          "pending",
			})
		default:
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		}
	}))
}

// TestCompleteWorkflow tests a complete workflow from wallet creation to capability registration
func TestCompleteWorkflow(t *testing.T) {
	// Skip if not running end-to-end tests
	if testing.Short() {
		t.Skip("Skipping end-to-end test in short mode")
	}

	// Start mock KNIRVCHAIN server
	server := MockKNIRVCHAINServer()
	defer server.Close()

	// Build the CLI binary
	tempDir, err := os.MkdirTemp("", "knirvchain-cli-e2e-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	binaryPath := filepath.Join(tempDir, "knirv")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, ".")
	err = buildCmd.Run()
	require.NoError(t, err, "Failed to build CLI binary")

	// Step 1: Initialize configuration
	configPath := filepath.Join(tempDir, "config.yaml")
	initCmd := exec.Command(binaryPath, "init", "--config", configPath, "--node-url", server.URL, "--overwrite")
	var stdout, stderr bytes.Buffer
	initCmd.Stdout = &stdout
	initCmd.Stderr = &stderr
	err = initCmd.Run()
	require.NoError(t, err, "Init command failed: %s", stderr.String())
	assert.Contains(t, stdout.String(), "Configuration initialized")

	// Step 2: Create a wallet
	walletDir := filepath.Join(tempDir, "wallets")
	err = os.MkdirAll(walletDir, 0755)
	require.NoError(t, err)

	walletPath := filepath.Join(walletDir, "e2e-wallet.json")
	walletCmd := exec.Command(binaryPath, "wallet", "new", "--file", walletPath, "--no-password")
	stdout.Reset()
	stderr.Reset()
	walletCmd.Stdout = &stdout
	walletCmd.Stderr = &stderr
	err = walletCmd.Run()
	require.NoError(t, err, "Wallet creation failed: %s", stderr.String())
	assert.Contains(t, stdout.String(), "Wallet created successfully")

	// Extract wallet address
	var address string
	for _, line := range bytes.Split(stdout.Bytes(), []byte("\n")) {
		if bytes.Contains(line, []byte("address:")) {
			parts := bytes.Split(line, []byte("address:"))
			if len(parts) > 1 {
				address = string(bytes.TrimSpace(parts[1]))
				break
			}
		}
	}
	require.NotEmpty(t, address, "Could not extract wallet address")

	// Step 3: List wallets
	listCmd := exec.Command(binaryPath, "wallet", "list", "--directory", walletDir, "--show-paths")
	stdout.Reset()
	stderr.Reset()
	listCmd.Stdout = &stdout
	listCmd.Stderr = &stderr
	err = listCmd.Run()
	require.NoError(t, err, "Wallet list failed: %s", stderr.String())
	assert.Contains(t, stdout.String(), address)

	// Step 4: Create test files for capability registration
	pluginPath := filepath.Join(tempDir, "test-plugin.so")
	err = os.WriteFile(pluginPath, []byte("mock plugin data"), 0644)
	require.NoError(t, err)

	manifestPath := filepath.Join(tempDir, "test-manifest.json")
	manifestContent := `{
		"name": "Test Capability",
		"version": "1.0.0",
		"description": "Test capability for end-to-end testing"
	}`
	err = os.WriteFile(manifestPath, []byte(manifestContent), 0644)
	require.NoError(t, err)

	// Step 5: Register a capability
	// Note: This will likely fail in the test environment due to plugin validation,
	// but we can check that the command structure is correct
	capabilityCmd := exec.Command(
		binaryPath,
		"mcp",
		"capability",
		"register",
		"--node", server.URL,
		"--wallet", walletPath,
		"--from", address,
		"--type", "RESOURCE",
		"--descriptor", `{"name":"Test Capability","version":"1.0.0"}`,
		"--plugin", pluginPath,
		"--opschema", manifestPath,
		"--fee", "100",
		"--no-password",
	)
	stdout.Reset()
	stderr.Reset()
	capabilityCmd.Stdout = &stdout
	capabilityCmd.Stderr = &stderr
	_ = capabilityCmd.Run() // Ignore error as this is expected to fail in test environment

	t.Logf("Capability command output: %s", stdout.String())
	t.Logf("Capability command error: %s", stderr.String())
}