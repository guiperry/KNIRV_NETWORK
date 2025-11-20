package uri

import (
	"KNIRVCHAIN/config"
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestURIGeneratorHandler_Integration(t *testing.T) {
	// Setup: Start a real node
	tempDir, err := os.MkdirTemp("", "uri-test-node-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test_uri_gen_node.db")
	nodePort := 9090
	minerAddr := "test_miner_uri_gen"
	nodeURL := fmt.Sprintf("http://localhost:%d", nodePort)

	// Build the main executable
	tempBinPath := filepath.Join(tempDir, "_test_app")
	buildCmd := exec.Command("go", "build", "-o", tempBinPath, "../")
	buildOutput, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build main executable: %v, output: %s", err, string(buildOutput))
	}

	// Create a test config file with InstallComplete set to true
	configDir := filepath.Join(tempDir, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	configPath := filepath.Join(configDir, "config.json")
	testConfig := map[string]interface{}{
		"InstallComplete": true,
		"Port":            nodePort,
		"MinersAddress":   minerAddr,
		"DatabasePath":    dbPath,
	}

	configJSON, err := json.Marshal(testConfig)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configPath, configJSON, 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Start the node process using the built executable
	cmd := exec.Command(
		tempBinPath,
		"--port", fmt.Sprintf("%d", nodePort),
		"--miners_address", minerAddr,
		"--shared_database_path", dbPath,
		"--config", configPath,
		"--skip-install",     // Skip the installer
		"--no-wallet-server", // Prevent node from starting its own wallet server
	)

	// Setup stdout and stderr pipes
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("Failed to create stdout pipe: %v", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatalf("Failed to create stderr pipe: %v", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start node: %v", err)
	}

	// Handle stdout and stderr in background goroutines
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			t.Logf("[NODE STDOUT] %s", scanner.Text())
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			t.Logf("[NODE STDERR] %s", scanner.Text())
		}
	}()

	// Create a cleanup function
	cleanup := func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
			cmd.Wait()
		}
	}

	// Ensure cleanup happens
	defer cleanup()

	// Wait for node to be ready
	config.WaitForNode(t, nodeURL, 20*time.Second)
	t.Log("Test node started and healthy")

	client := &http.Client{Timeout: 35 * time.Second}
	uriGeneratorURL := fmt.Sprintf("%s/uriGenerator", nodeURL)

	t.Run("Default UUID Generation", func(t *testing.T) {
		// Send POST request with empty JSON body
		emptyBody := bytes.NewBuffer([]byte(`{}`))
		resp, err := client.Post(uriGeneratorURL, "application/json", emptyBody)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if status := resp.StatusCode; status != http.StatusCreated {
			bodyBytes, _ := io.ReadAll(resp.Body)
			t.Fatalf("Handler returned wrong status code: got %v want %v, body: %s",
				status, http.StatusCreated, string(bodyBytes))
		}

		var response struct {
			URI     string `json:"uri"`
			TxnHash string `json:"txn_hash"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			t.Fatal("Failed to decode response:", err)
		}

		t.Logf("Generated URI: %s", response.URI)
		t.Logf("Generated Txn Hash: %s", response.TxnHash)

		// Validate URI format
		if !strings.HasPrefix(response.URI, "knirv://") {
			t.Errorf("Invalid URI format: %s, expected knirv:// prefix", response.URI)
		}

		// Parse the URI to validate its structure
		id, resourceType, path, params, err := ParseResourceURI(response.URI)
		if err != nil {
			t.Errorf("Failed to parse generated URI: %v", err)
		} else {
			if id == "" {
				t.Error("Empty ID in generated URI")
			}
			if resourceType != ResourceTypeChainStr {
				t.Errorf("Expected resource type '%s', got '%s'", ResourceTypeChainStr, resourceType)
			} // Using string constant for URI compatibility
			if path != "/" {
				t.Errorf("Expected root path '/', got '%s'", path)
			}
			t.Logf("Parsed URI - ID: %s, ResourceType: %s, Path: %s, Params: %v", id, resourceType, path, params)
		}

		// Validate transaction hash
		if len(response.TxnHash) == 0 {
			t.Error("Empty transaction hash in response")
		}
	})

	t.Run("With Available Desired ID", func(t *testing.T) {
		desiredID := "my-test-id-123"
		requestBodyJSON := fmt.Sprintf(`{"desired_id": "%s"}`, desiredID)
		requestBody := bytes.NewBuffer([]byte(requestBodyJSON))

		resp, err := client.Post(uriGeneratorURL, "application/json", requestBody)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if status := resp.StatusCode; status != http.StatusCreated {
			bodyBytes, _ := io.ReadAll(resp.Body)
			t.Fatalf("Handler returned wrong status code: got %v want %v, body: %s",
				status, http.StatusCreated, string(bodyBytes))
		}

		var response struct {
			URI     string `json:"uri"`
			TxnHash string `json:"txn_hash"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			t.Fatal("Failed to decode response:", err)
		}

		// Verify the ID was used in the URI
		expectedURIStart := fmt.Sprintf("knirv://%s.%s/", desiredID, ResourceTypeChainStr) // Using string constant for URI compatibility
		if !strings.HasPrefix(response.URI, expectedURIStart) {
			t.Errorf("Expected URI to start with '%s', got: %s", expectedURIStart, response.URI)
		}
		t.Logf("Generated URI with desired ID: %s", response.URI)
	})

	t.Run("Conflict Scenario", func(t *testing.T) {
		// First request succeeds and registers the ID
		desiredID := "taken-id-456"
		requestBodyJSON := fmt.Sprintf(`{"desired_id": "%s"}`, desiredID)
		requestBody1 := bytes.NewBuffer([]byte(requestBodyJSON))

		resp1, err1 := client.Post(uriGeneratorURL, "application/json", requestBody1)
		if err1 != nil {
			t.Fatalf("Failed to send first request: %v", err1)
		}
		defer resp1.Body.Close()
		if status := resp1.StatusCode; status != http.StatusCreated {
			bodyBytes, _ := io.ReadAll(resp1.Body)
			t.Fatalf("First request failed unexpectedly: got %v want %v, body: %s",
				status, http.StatusCreated, string(bodyBytes))
		}
		t.Logf("Successfully registered ID '%s' for conflict test.", desiredID)

		// Allow some time for DHT announcement
		time.Sleep(500 * time.Millisecond)

		// Second request with same ID should conflict
		requestBody2 := bytes.NewBuffer([]byte(requestBodyJSON))
		resp2, err2 := client.Post(uriGeneratorURL, "application/json", requestBody2)
		if err2 != nil {
			t.Fatalf("Failed to send second request: %v", err2)
		}
		defer resp2.Body.Close()

		if status := resp2.StatusCode; status != http.StatusConflict {
			bodyBytes, _ := io.ReadAll(resp2.Body)
			t.Fatalf("Handler returned wrong status code for conflict: got %v want %v, body: %s",
				status, http.StatusConflict, string(bodyBytes))
		}
		t.Logf("Correctly received Conflict status for already taken ID '%s'.", desiredID)
	})
}

// Keep existing TestParseResourceURI and TestGenerateResourceURI tests unchanged

func TestParseResourceURI(t *testing.T) {
	testCases := []struct {
		name          string
		uri           string
		expectedID    string
		expectedType  string
		expectedPath  string
		expectedError bool
	}{
		{
			name:         "Valid Chain URI",
			uri:          "knirv://abc123.chain/",
			expectedID:   "abc123",
			expectedType: "chain",
			expectedPath: "/",
		},
		{
			name:         "Valid Chain URI with Path",
			uri:          "knirv://abc123.chain/block",
			expectedID:   "abc123",
			expectedType: "chain",
			expectedPath: "/block",
		},
		{
			name:         "Valid Chain URI with Query",
			uri:          "knirv://abc123.chain/block?hash=xyz789",
			expectedID:   "abc123",
			expectedType: "chain",
			expectedPath: "/block",
		},
		{
			name:         "Valid NRN URI",
			uri:          "knirv://content123.nrn/",
			expectedID:   "content123",
			expectedType: "nrn",
			expectedPath: "/",
		},
		{
			name:          "Invalid Scheme",
			uri:           "http://abc123.chain/",
			expectedError: true,
		},
		{
			name:          "Invalid Authority Format",
			uri:           "knirv://abc123/",
			expectedError: true,
		},
		{
			name:          "Invalid Resource Type",
			uri:           "knirv://abc123.invalid/",
			expectedError: false, // We don't validate resource type in ParseResourceURI
			expectedID:    "abc123",
			expectedType:  "invalid",
			expectedPath:  "/",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			id, resourceType, path, params, err := ParseResourceURI(tc.uri)

			if tc.expectedError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if id != tc.expectedID {
				t.Errorf("Expected ID '%s', got '%s'", tc.expectedID, id)
			}

			if resourceType != tc.expectedType {
				t.Errorf("Expected resource type '%s', got '%s'", tc.expectedType, resourceType)
			}

			if path != tc.expectedPath {
				t.Errorf("Expected path '%s', got '%s'", tc.expectedPath, path)
			}

			t.Logf("Params: %v", params)
		})
	}
}

func TestGenerateResourceURI(t *testing.T) {
	testCases := []struct {
		name           string
		id             string
		resourceType   string
		path           string
		params         map[string]string
		expectedPrefix string
	}{
		{
			name:           "Chain URI",
			id:             "abc123",
			resourceType:   "chain",
			path:           "",
			params:         nil,
			expectedPrefix: "knirv://abc123.chain/",
		},
		{
			name:           "Chain URI with Path",
			id:             "abc123",
			resourceType:   "chain",
			path:           "block",
			params:         nil,
			expectedPrefix: "knirv://abc123.chain/block",
		},
		{
			name:           "Chain URI with Path and Params",
			id:             "abc123",
			resourceType:   "chain",
			path:           "block",
			params:         map[string]string{"hash": "xyz789"},
			expectedPrefix: "knirv://abc123.chain/block?hash=xyz789",
		},
		{
			name:           "NRN URI",
			id:             "content123",
			resourceType:   "nrn",
			path:           "",
			params:         nil,
			expectedPrefix: "knirv://content123.nrn/",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			uri := GenerateResourceURI(tc.id, tc.resourceType, tc.path, tc.params)

			if !strings.HasPrefix(uri, tc.expectedPrefix) {
				t.Errorf("Expected URI to start with '%s', got '%s'", tc.expectedPrefix, uri)
			}

			t.Logf("Generated URI: %s", uri)
		})
	}
}
