package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"KNIRVORACLE/uri"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTunnelRegistryIntegration performs an integration test of the tunnel registry system
func TestTunnelRegistryIntegration(t *testing.T) {
	// Skip this test in CI environments or if the integration flag is not set
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration test")
	}

	// Start the Go server with the internal API endpoints
	goServerCmd := startGoServer(t)
	defer goServerCmd.Process.Kill()

	// Start the Node.js tunnel registry
	nodeServerCmd := startNodeServer(t)
	if nodeServerCmd != nil {
		defer nodeServerCmd.Process.Kill()
	}

	// Wait for servers to start
	time.Sleep(5 * time.Second)

	// Test node registration
	testNodeRegistration(t)

	// Test URI generation and resolution
	testURIGenerationAndResolution(t)
}

func startGoServer(t *testing.T) *exec.Cmd {
	// Try to run the server with proper flags
	cmd := exec.Command("go", "run", ".", "-role", "client", "-port", "8080", "-p2p.port", "4001", "-skip-install", "-no-wallet-server")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set working directory to project root
	wd, _ := os.Getwd()
	if strings.HasSuffix(wd, "tests") {
		cmd.Dir = filepath.Dir(wd)
	}

	err := cmd.Start()
	require.NoError(t, err, "Failed to start Go server")

	return cmd
}

func startNodeServer(t *testing.T) *exec.Cmd {
	// Check if the agent-tunnel-registry directory exists
	registryDir := filepath.Join("..", "agent-tunnel-registry")
	if _, err := os.Stat(registryDir); os.IsNotExist(err) {
		t.Skip("Skipping Node.js tunnel registry test - directory not found")
		return nil
	}

	// Start the Node.js tunnel registry
	cmd := exec.Command("node", "server.js")
	cmd.Dir = registryDir
	cmd.Env = append(os.Environ(),
		"HTTP_API_PORT=3003",
		"CONTROL_PORT=4001",
		"PUBLIC_RELAY_PORT=4000",
		"STUN_PORT=3478",
		"PUBLIC_HOST=localhost",
		"RELAY_SERVER_PEER_ID=12D3KooWEKxzRUXdYBDzZbZvbzEaWfH4RaYJWZpLgKKQGKKGKKGK",
		"GO_INTERNAL_API_PORT=8080",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	require.NoError(t, err, "Failed to start Node.js tunnel registry")

	return cmd
}

func testNodeRegistration(t *testing.T) {
	// Register a node via the API
	nodeInfo := map[string]interface{}{
		"devId":         "QmTestPeerId123",
		"chainId":       "test-chain-1",
		"publicIp":      "localhost",
		"publicP2pPort": 4001,
		"type":          "dev",
	}

	nodeInfoJSON, err := json.Marshal(nodeInfo)
	require.NoError(t, err, "Failed to marshal node info")

	resp, err := http.Post("http://localhost:3003/api/registry/register", "application/json", bytes.NewBuffer(nodeInfoJSON))
	require.NoError(t, err, "Failed to register node")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode, "Node registration failed")

	// Verify the node was registered
	resp, err = http.Get("http://localhost:3003/api/registry/node/dev/QmTestPeerId123")
	require.NoError(t, err, "Failed to get node info")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Failed to get node info")

	var registeredNode map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&registeredNode)
	require.NoError(t, err, "Failed to decode node info")

	assert.Equal(t, "QmTestPeerId123", registeredNode["devId"], "Node ID mismatch")
	assert.Equal(t, "test-chain-1", registeredNode["chainId"], "Chain ID mismatch")
	assert.Equal(t, "localhost", registeredNode["publicIp"], "Public IP mismatch")
	assert.Equal(t, float64(4001), registeredNode["publicP2pPort"], "Public P2P port mismatch")
}

func testURIGenerationAndResolution(t *testing.T) {
	// Generate a URI
	uriRequest := map[string]interface{}{
		"devId":        "QmTestPeerId123",
		"resourceType": "chain",
		"subPath":      "test/path",
	}

	uriRequestJSON, err := json.Marshal(uriRequest)
	require.NoError(t, err, "Failed to marshal URI request")

	resp, err := http.Post("http://localhost:3003/api/uri/generate", "application/json", bytes.NewBuffer(uriRequestJSON))
	require.NoError(t, err, "Failed to generate URI")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "URI generation failed")

	var uriResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&uriResponse)
	require.NoError(t, err, "Failed to decode URI response")

	assert.Contains(t, uriResponse, "uri", "URI response missing URI")
	generatedURI := uriResponse["uri"].(string)

	// Resolve the URI
	resp, err = http.Get(fmt.Sprintf("http://localhost:3003/api/uri/resolve?uri=%s", generatedURI))
	require.NoError(t, err, "Failed to resolve URI")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "URI resolution failed")

	var resolvedURI map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&resolvedURI)
	require.NoError(t, err, "Failed to decode resolved URI")

	assert.Equal(t, generatedURI, resolvedURI["originalUri"], "Original URI mismatch")
	assert.Equal(t, "QmTestPeerId123", resolvedURI["targetPeerId"], "Target peer ID mismatch")

	// Test URI resolution using the Go URI resolver
	resolver := uri.NewURIResolver()
	resolved, err := resolver.ResolveURI(generatedURI)
	require.NoError(t, err, "Failed to resolve URI using Go resolver")

	assert.Equal(t, generatedURI, resolved.OriginalURI, "Original URI mismatch in Go resolver")
	assert.Equal(t, "QmTestPeerId123", resolved.TargetPeerID, "Target peer ID mismatch in Go resolver")
}

// TestTunnelRegistryUnitTests runs the unit tests for the tunnel registry components
func TestTunnelRegistryUnitTests(t *testing.T) {
	// Test the URI resolver
	testURIResolver(t)

	// Test the internal API endpoints
	testInternalAPIEndpoints(t)
}

func testURIResolver(t *testing.T) {
	resolver := uri.NewURIResolver()

	// Test URI parsing
	authority, identifier, resourceType, subPath, err := resolver.ParseURI("knirv://example.com/resource-123.chain/test/path")
	require.NoError(t, err, "Failed to parse URI")

	assert.Equal(t, "example.com", authority, "Authority mismatch")
	assert.Equal(t, "resource-123", identifier, "Identifier mismatch")
	assert.Equal(t, "chain", resourceType, "Resource type mismatch")
	assert.Equal(t, "/test/path", subPath, "Subpath mismatch")

	// Test invalid URI
	_, _, _, _, err = resolver.ParseURI("invalid-uri")
	assert.Error(t, err, "Expected error for invalid URI")
}

func testInternalAPIEndpoints(t *testing.T) {
	// Test the internal DHT find resource endpoint
	resp, err := http.Get("http://localhost:8080/internal/dht/findResource?id=test-resource&type=chain")
	if err != nil {
		t.Skip("Skipping internal API test - server not available")
		return
	}
	defer resp.Body.Close()

	// We don't care about the result, just that the endpoint exists and responds
	assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNotFound,
		"Unexpected status code for DHT find resource endpoint")

	// Test the internal DB ID exists endpoint
	resp, err = http.Get("http://localhost:8080/internal/db/idExists?id=test-id")
	require.NoError(t, err, "Failed to call ID exists endpoint")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Unexpected status code for ID exists endpoint")

	var idExistsResponse map[string]bool
	err = json.NewDecoder(resp.Body).Decode(&idExistsResponse)
	require.NoError(t, err, "Failed to decode ID exists response")

	assert.Contains(t, idExistsResponse, "exists", "ID exists response missing 'exists' field")
}
