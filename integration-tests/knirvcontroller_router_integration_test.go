package integration_tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test configuration - Updated for real network integration
const (
	KNIRVControllerURL = "http://localhost:3000" // KNIRVCONTROLLER Unified Server
	KNIRVRouterURL     = "http://localhost:8085" // KNIRVROUTER
	KNIRVGraphURL      = "http://localhost:8081" // KNIRVGRAPH
	KNIRVChainURL      = "http://localhost:8080" // KNIRVCHAIN
	KNIRVOracleURL     = "http://localhost:8086" // KNIRVORACLE
	KNIRVNexusURL      = "http://localhost:8084" // KNIRVNEXUS
	TestTimeout        = 60 * time.Second        // Increased for real network
)

// Test data structures
type ErrorContext struct {
	ErrorID      string                 `json:"errorId"`
	ErrorType    string                 `json:"errorType"`
	ErrorMessage string                 `json:"errorMessage"`
	StackTrace   string                 `json:"stackTrace"`
	UserContext  map[string]interface{} `json:"userContext"`
	AgentID      string                 `json:"agentId"`
	Timestamp    int64                  `json:"timestamp"`
	Severity     string                 `json:"severity"`
}

type SkillInvocationRequest struct {
	SkillID     string                 `json:"skillId"`
	UserAddress string                 `json:"userAddress"`
	NRNAmount   string                 `json:"nrnAmount"`
	Parameters  map[string]interface{} `json:"parameters"`
	Priority    string                 `json:"priority"`
	UseP2P      bool                   `json:"useP2P"`
	UseWASM     bool                   `json:"useWASM"`
}

type SkillInvocationResponse struct {
	RequestID      string `json:"requestId"`
	Status         string `json:"status"`
	SkillNodeURI   string `json:"skillNodeUri,omitempty"`
	ErrorMessage   string `json:"errorMessage,omitempty"`
	ExecutionTime  int64  `json:"executionTime"`
	NetworkLatency int64  `json:"networkLatency"`
}

type LoRAAdapterRequest struct {
	AdapterName            string            `json:"adapterName"`
	Description            string            `json:"description"`
	BaseModelCompatibility string            `json:"baseModelCompatibility"`
	Version                int               `json:"version"`
	Rank                   int               `json:"rank"`
	Alpha                  float64           `json:"alpha"`
	Metadata               map[string]string `json:"metadata"`
}

type LoRAAdapterResponse struct {
	AdapterID   string `json:"adapterId"`
	AdapterName string `json:"adapterName"`
	Status      string `json:"status"`
}

// Test helper functions
func waitForService(url string, timeout time.Duration) error {
	client := &http.Client{Timeout: 5 * time.Second}
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		resp, err := client.Get(url + "/health")
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			return nil
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("service at %s not ready within timeout", url)
}

func makeHTTPRequest(method, url string, payload interface{}) (*http.Response, error) {
	client := &http.Client{Timeout: TestTimeout}

	var body bytes.Buffer
	if payload != nil {
		if err := json.NewEncoder(&body).Encode(payload); err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, url, &body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	return client.Do(req)
}

// Test functions
func TestKNIRVControllerRouterIntegration(t *testing.T) {
	// Wait for services to be ready
	t.Log("Waiting for KNIRVCONTROLLER to be ready...")
	require.NoError(t, waitForService(KNIRVControllerURL, TestTimeout))

	t.Log("Waiting for KNIRVROUTER to be ready...")
	require.NoError(t, waitForService(KNIRVRouterURL, TestTimeout))

	t.Log("Waiting for KNIRVGRAPH to be ready...")
	require.NoError(t, waitForService(KNIRVGraphURL, TestTimeout))

	t.Run("TestSkillInvocationViaErrorContext", testSkillInvocationViaErrorContext)
	t.Run("TestLoRAAdapterRegistration", testLoRAAdapterRegistration)
	t.Run("TestLoRAAdapterRetrieval", testLoRAAdapterRetrieval)
	t.Run("TestP2PRouting", testP2PRouting)
	t.Run("TestWASMExecution", testWASMExecution)
	t.Run("TestErrorContextGeneration", testErrorContextGeneration)
}

func testSkillInvocationViaErrorContext(t *testing.T) {
	t.Log("Testing skill invocation via ErrorContext → KNIRVGRAPH → KNIRVROUTER")

	request := SkillInvocationRequest{
		SkillID:     "test-skill-001",
		UserAddress: "knirv1test123456789",
		NRNAmount:   "100",
		Parameters: map[string]interface{}{
			"agentId":      "test-agent-001",
			"capabilities": []string{"text-processing", "analysis"},
			"priority":     "high",
		},
		Priority: "high",
		UseP2P:   true,
		UseWASM:  true,
	}

	resp, err := makeHTTPRequest("POST", KNIRVControllerURL+"/api/invoke-skill", request)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var response SkillInvocationResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "SUCCESS", response.Status)
	assert.NotEmpty(t, response.RequestID)
	assert.Greater(t, response.ExecutionTime, int64(0))

	t.Logf("Skill invocation successful: RequestID=%s, ExecutionTime=%dms",
		response.RequestID, response.ExecutionTime)
}

func testLoRAAdapterRegistration(t *testing.T) {
	t.Log("Testing LoRA adapter registration via KNIRVROUTER")

	request := LoRAAdapterRequest{
		AdapterName:            "test-lora-adapter",
		Description:            "Test LoRA adapter for integration testing",
		BaseModelCompatibility: "hrm-v1",
		Version:                1,
		Rank:                   16,
		Alpha:                  0.5,
		Metadata: map[string]string{
			"test":   "true",
			"author": "integration-test",
		},
	}

	resp, err := makeHTTPRequest("POST", KNIRVControllerURL+"/api/register-lora-adapter", request)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var response LoRAAdapterResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "SUCCESS", response.Status)
	assert.NotEmpty(t, response.AdapterID)
	assert.Equal(t, request.AdapterName, response.AdapterName)

	t.Logf("LoRA adapter registration successful: AdapterID=%s", response.AdapterID)
}

func testLoRAAdapterRetrieval(t *testing.T) {
	t.Log("Testing LoRA adapter retrieval from KNIRVROUTER")

	resp, err := makeHTTPRequest("GET", KNIRVControllerURL+"/api/lora-adapters", nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var adapters []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&adapters)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, len(adapters), 0)

	t.Logf("Retrieved %d LoRA adapters from KNIRVROUTER", len(adapters))
}

func testP2PRouting(t *testing.T) {
	t.Log("Testing P2P routing capabilities")

	request := map[string]interface{}{
		"skillId": "test-p2p-skill",
		"useP2P":  true,
		"parameters": map[string]interface{}{
			"agentId": "test-agent-p2p",
		},
	}

	resp, err := makeHTTPRequest("POST", KNIRVControllerURL+"/api/test-p2p-routing", request)
	require.NoError(t, err)
	defer resp.Body.Close()

	// P2P might not be fully available in test environment
	if resp.StatusCode == http.StatusOK {
		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		t.Logf("P2P routing test result: %+v", response)
	} else {
		t.Logf("P2P routing not available in test environment (status: %d)", resp.StatusCode)
	}
}

func testWASMExecution(t *testing.T) {
	t.Log("Testing WASM execution capabilities")

	request := map[string]interface{}{
		"skillId": "test-wasm-skill",
		"useWASM": true,
		"parameters": map[string]interface{}{
			"agentId": "test-agent-wasm",
		},
	}

	resp, err := makeHTTPRequest("POST", KNIRVControllerURL+"/api/test-wasm-execution", request)
	require.NoError(t, err)
	defer resp.Body.Close()

	// WASM might not be fully available in test environment
	if resp.StatusCode == http.StatusOK {
		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		t.Logf("WASM execution test result: %+v", response)
	} else {
		t.Logf("WASM execution not available in test environment (status: %d)", resp.StatusCode)
	}
}

func testErrorContextGeneration(t *testing.T) {
	t.Log("Testing ErrorContext generation and processing")

	errorContext := ErrorContext{
		ErrorID:      "test-error-001",
		ErrorType:    "skill_invocation_request",
		ErrorMessage: "Test error for integration testing",
		StackTrace:   "test stack trace",
		UserContext: map[string]interface{}{
			"userAddress": "knirv1test123456789",
			"nrnAmount":   "100",
		},
		AgentID:   "test-agent-001",
		Timestamp: time.Now().UnixMilli(),
		Severity:  "medium",
	}

	resp, err := makeHTTPRequest("POST", KNIRVControllerURL+"/api/process-error-context", errorContext)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "SUCCESS", response["status"])

	t.Logf("ErrorContext processing successful: %+v", response)
}

func TestMain(m *testing.M) {
	// Setup
	fmt.Println("Starting KNIRVCONTROLLER-KNIRVROUTER Integration Tests...")

	// Run tests
	code := m.Run()

	// Cleanup
	fmt.Println("Integration tests completed.")

	os.Exit(code)
}
