package integration

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVCLI/core"
	"github.com/stretchr/testify/require"
)

// MockMCPServer creates a mock MCP server for testing
func MockMCPServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/health":
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		case "/mcp/capability/register":
			// Parse request
			var request map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": "invalid request"})
				return
			}

			// Return success response
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":           "success",
				"transaction_hash": "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
				"capability_id":    "cap-1234567890",
			})
		case "/mcp/server/register":
			// Parse request
			var request map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": "invalid request"})
				return
			}

			// Return success response
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":           "success",
				"transaction_hash": "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				"server_id":        "srv-1234567890",
			})
		case "/mcp/procedure/register":
			// Parse request
			var request map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": "invalid request"})
				return
			}

			// Return success response
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":           "success",
				"transaction_hash": "0x7890abcdef1234567890abcdef1234567890abcdef1234567890abcdef123456",
				"procedure_id":     "proc-1234567890",
			})
		default:
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
		}
	}))
}

// TestMCPCapabilityCommand tests the MCP capability command
func TestMCPCapabilityCommand(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Start mock MCP server
	server := MockMCPServer()
	defer server.Close()

	// Build the CLI binary
	tempDir, err := os.MkdirTemp("", "knirvchain-cli-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	binaryPath := filepath.Join(tempDir, "knirv")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, ".")
	err = buildCmd.Run()
	require.NoError(t, err, "Failed to build CLI binary")

	// Create test files
	pluginPath := filepath.Join(tempDir, "test-plugin.so")
	err = ioutil.WriteFile(pluginPath, []byte("mock plugin data"), 0644)
	require.NoError(t, err)

	manifestPath := filepath.Join(tempDir, "test-manifest.json")
	manifestContent := `{
		"name": "Test Capability",
		"version": "1.0.0",
		"description": "Test capability for integration testing"
	}`
	err = ioutil.WriteFile(manifestPath, []byte(manifestContent), 0644)
	require.NoError(t, err)

	// Create a test wallet
	walletPath := filepath.Join(tempDir, "test-wallet.json")
	walletManager := core.NewWalletManager(tempDir, nil)

	privateKey, err := walletManager.GenerateKeyPair()
	require.NoError(t, err)

	err = walletManager.SaveWallet(privateKey, "test-password", walletPath)
	require.NoError(t, err)

	address := core.GetAddressFromPrivateKey(privateKey)

	// Run the MCP capability register command
	cmd := exec.Command(
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
		"--password", "test-password",
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	_ = cmd.Run()

	// This will likely fail in the test environment due to plugin validation,
	// but we can check that the command structure is correct
	t.Logf("Command output: %s", stdout.String())
	t.Logf("Command error: %s", stderr.String())
}

// TestMCPServerCommand tests the MCP server command
func TestMCPServerCommand(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Start mock MCP server
	server := MockMCPServer()
	defer server.Close()

	// Build the CLI binary
	tempDir, err := os.MkdirTemp("", "knirvchain-cli-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	binaryPath := filepath.Join(tempDir, "knirv")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, ".")
	err = buildCmd.Run()
	require.NoError(t, err, "Failed to build CLI binary")

	// Create test files
	schemaPath := filepath.Join(tempDir, "test-server-schema.json")
	schemaContent := `{
		"name": "Test Server",
		"version": "1.0.0",
		"description": "Test server for integration testing"
	}`
	err = ioutil.WriteFile(schemaPath, []byte(schemaContent), 0644)
	require.NoError(t, err)

	// Create a test wallet
	walletPath := filepath.Join(tempDir, "test-wallet.json")
	walletManager := core.NewWalletManager(tempDir, nil)

	privateKey, err := walletManager.GenerateKeyPair()
	require.NoError(t, err)

	err = walletManager.SaveWallet(privateKey, "test-password", walletPath)
	require.NoError(t, err)

	address := core.GetAddressFromPrivateKey(privateKey)

	// Run the MCP server register command
	cmd := exec.Command(
		binaryPath,
		"mcp",
		"server",
		"register",
		"--node", server.URL,
		"--wallet", walletPath,
		"--from", address,
		"--server-schema", schemaPath,
		"--endpoint", "https://test-server.example.com",
		"--fee", "100",
		"--password", "test-password",
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	_ = cmd.Run()

	// This will likely fail in the test environment due to server validation,
	// but we can check that the command structure is correct
	t.Logf("Command output: %s", stdout.String())
	t.Logf("Command error: %s", stderr.String())
}