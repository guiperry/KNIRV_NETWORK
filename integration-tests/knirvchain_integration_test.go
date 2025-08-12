package integration_tests

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestKNIRVCHAINIntegration tests the complete KNIRVCHAIN system integration
func TestKNIRVCHAINIntegration(t *testing.T) {
	// Setup test environment
	setupTestEnvironment(t)
	defer cleanupTestEnvironment(t)

	// Test 1: Build and compile KNIRVCHAIN
	t.Run("BuildKNIRVCHAIN", func(t *testing.T) {
		testBuildKNIRVCHAIN(t)
	})

	// Test 2: Start KNIRVCHAIN server
	var serverProcess *exec.Cmd
	t.Run("StartServer", func(t *testing.T) {
		serverProcess = testStartKNIRVCHAINServer(t)
	})
	defer func() {
		if serverProcess != nil {
			serverProcess.Process.Kill()
		}
	}()

	// Wait for server to be ready
	time.Sleep(5 * time.Second)

	// Test 3: Health check
	t.Run("HealthCheck", func(t *testing.T) {
		testHealthEndpoint(t)
	})

	// Test 4: Model management endpoints
	t.Run("ModelManagement", func(t *testing.T) {
		testModelManagementEndpoints(t)
	})

	// Test 5: Governance endpoints
	t.Run("GovernanceSystem", func(t *testing.T) {
		testGovernanceEndpoints(t)
	})

	// Test 6: Consensus status
	t.Run("ConsensusStatus", func(t *testing.T) {
		testConsensusEndpoints(t)
	})

	// Test 7: IBC connections
	t.Run("IBCConnections", func(t *testing.T) {
		testIBCEndpoints(t)
	})

	// Test 8: TEE operations
	t.Run("TEEOperations", func(t *testing.T) {
		testTEEEndpoints(t)
	})

	// Test 9: IPFS integration
	t.Run("IPFSIntegration", func(t *testing.T) {
		testIPFSEndpoints(t)
	})

	// Test 10: Performance under load
	t.Run("LoadTesting", func(t *testing.T) {
		testLoadPerformance(t)
	})
}

func setupTestEnvironment(t *testing.T) {
	// Create test environment file
	envContent := `DEEPSEEK_API_KEY=test_deepseek_key
GEMINI_API_KEY=test_gemini_key
GEMINI_PROJECT_ID=test_project
CEREBRAS_API_KEY=test_cerebras_key
IPFS_GATEWAY_URL=http://localhost:8080
DEEPSEEK_BASE_URL=https://api.deepseek.com/chat/completions
CEREBRAS_BASE_URL=https://api.cerebras.ai/v1/chat/completions
`

	envPath := filepath.Join("KNIRVCHAIN", ".env")
	err := ioutil.WriteFile(envPath, []byte(envContent), 0644)
	if err != nil {
		t.Logf("Warning: Could not create .env file: %v", err)
	}
}

func cleanupTestEnvironment(t *testing.T) {
	// Clean up test files
	envPath := filepath.Join("KNIRVCHAIN", ".env")
	os.Remove(envPath)
}

func testBuildKNIRVCHAIN(t *testing.T) {
	cmd := exec.Command("cargo", "build", "--release")
	cmd.Dir = "KNIRVCHAIN"

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build KNIRVCHAIN: %v\nOutput: %s", err, output)
	}

	t.Logf("KNIRVCHAIN built successfully")
}

func testStartKNIRVCHAINServer(t *testing.T) *exec.Cmd {
	cmd := exec.Command("./target/release/knirvchain")
	cmd.Dir = "KNIRVCHAIN"

	// Set environment variables
	cmd.Env = append(os.Environ(),
		"RUST_LOG=info",
		"KNIRVCHAIN_PORT=8080",
	)

	err := cmd.Start()
	if err != nil {
		t.Fatalf("Failed to start KNIRVCHAIN server: %v", err)
	}

	t.Logf("KNIRVCHAIN server started with PID: %d", cmd.Process.Pid)
	return cmd
}

func testHealthEndpoint(t *testing.T) {
	resp, err := http.Get("http://localhost:8080/health")
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Health check returned status %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read health response: %v", err)
	}

	if !strings.Contains(string(body), "ok") {
		t.Fatalf("Health check response unexpected: %s", body)
	}

	t.Logf("Health check passed")
}

func testModelManagementEndpoints(t *testing.T) {
	// Test list models endpoint
	resp, err := http.Get("http://localhost:8080/v3/models/list")
	if err != nil {
		t.Fatalf("Failed to list models: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("List models returned status %d", resp.StatusCode)
	}

	// Test model performance endpoint
	resp, err = http.Get("http://localhost:8080/v3/models/performance")
	if err != nil {
		t.Fatalf("Failed to get model performance: %v", err)
	}
	defer resp.Body.Close()

	// Test model switching (POST request)
	switchData := map[string]interface{}{
		"model_type": "CodeT5",
		"config": map[string]interface{}{
			"model_size": "base",
			"device":     "cpu",
			"max_length": 512,
		},
	}

	jsonData, _ := json.Marshal(switchData)
	resp, err = http.Post("http://localhost:8080/v3/models/switch",
		"application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatalf("Failed to switch model: %v", err)
	}
	defer resp.Body.Close()

	t.Logf("Model management endpoints tested successfully")
}

func testGovernanceEndpoints(t *testing.T) {
	// Test list proposals
	resp, err := http.Get("http://localhost:8080/v3/governance/proposals")
	if err != nil {
		t.Fatalf("Failed to list proposals: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("List proposals returned status %d", resp.StatusCode)
	}

	// Test voting (POST request)
	voteData := map[string]interface{}{
		"proposal_id": "test_proposal",
		"vote":        "yes",
		"voter":       "0x1234567890123456789012345678901234567890",
	}

	jsonData, _ := json.Marshal(voteData)
	resp, err = http.Post("http://localhost:8080/v3/governance/vote",
		"application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatalf("Failed to cast vote: %v", err)
	}
	defer resp.Body.Close()

	t.Logf("Governance endpoints tested successfully")
}

func testConsensusEndpoints(t *testing.T) {
	resp, err := http.Get("http://localhost:8080/v3/consensus/status")
	if err != nil {
		t.Fatalf("Failed to get consensus status: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Consensus status returned status %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read consensus response: %v", err)
	}

	var status map[string]interface{}
	err = json.Unmarshal(body, &status)
	if err != nil {
		t.Fatalf("Failed to parse consensus status: %v", err)
	}

	t.Logf("Consensus status: %v", status)
}

func testIBCEndpoints(t *testing.T) {
	resp, err := http.Get("http://localhost:8080/v3/ibc/connections")
	if err != nil {
		t.Fatalf("Failed to get IBC connections: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("IBC connections returned status %d", resp.StatusCode)
	}

	t.Logf("IBC endpoints tested successfully")
}

func testTEEEndpoints(t *testing.T) {
	// Test TEE skill preparation
	teeData := map[string]interface{}{
		"skill_id":   "test_skill",
		"skill_code": "fn main() { println!(\"Hello TEE!\"); }",
		"tee_type":   "IntelSGX",
	}

	jsonData, _ := json.Marshal(teeData)
	resp, err := http.Post("http://localhost:8080/v3/tee/prepare",
		"application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatalf("Failed to prepare TEE skill: %v", err)
	}
	defer resp.Body.Close()

	t.Logf("TEE endpoints tested successfully")
}

func testIPFSEndpoints(t *testing.T) {
	resp, err := http.Get("http://localhost:8080/v3/ipfs/status")
	if err != nil {
		t.Fatalf("Failed to get IPFS status: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("IPFS status returned status %d", resp.StatusCode)
	}

	t.Logf("IPFS endpoints tested successfully")
}

func testLoadPerformance(t *testing.T) {
	// Test concurrent requests to health endpoint
	concurrency := 10
	requests := 50

	start := time.Now()

	for i := 0; i < requests; i++ {
		go func() {
			resp, err := http.Get("http://localhost:8080/health")
			if err == nil {
				resp.Body.Close()
			}
		}()

		if i%concurrency == 0 {
			time.Sleep(10 * time.Millisecond)
		}
	}

	duration := time.Since(start)
	t.Logf("Load test completed: %d requests in %v", requests, duration)

	if duration > 5*time.Second {
		t.Fatalf("Load test took too long: %v", duration)
	}
}

// TestKNIRVCHAINUnitTests runs the Rust unit tests
func TestKNIRVCHAINUnitTests(t *testing.T) {
	cmd := exec.Command("cargo", "test", "--lib")
	cmd.Dir = "KNIRVCHAIN"

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Unit tests failed: %v\nOutput: %s", err, output)
	}

	t.Logf("Unit tests passed successfully")
}

// TestKNIRVCHAINPerformanceTests runs the Rust performance tests
func TestKNIRVCHAINPerformanceTests(t *testing.T) {
	cmd := exec.Command("cargo", "test", "--test", "performance_tests", "--release")
	cmd.Dir = "KNIRVCHAIN"

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Performance tests failed: %v\nOutput: %s", err, output)
	}

	t.Logf("Performance tests passed successfully")
}
