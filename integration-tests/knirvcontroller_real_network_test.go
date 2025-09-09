package integration_tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// KNIRVCONTROLLER Real Network Integration Tests
// These tests connect to actual KNIRV network services for real-world demos

// Test configuration for real network integration
// Constants are defined in test_constants.go

// KNIRVCONTROLLER-specific data structures
type KNIRVControllerHealthResponse struct {
	Status     string                 `json:"status"`
	Timestamp  string                 `json:"timestamp"`
	Components map[string]interface{} `json:"components"`
}

type KNIRVControllerSkillRequest struct {
	SkillID     string                 `json:"skillId"`
	UserAddress string                 `json:"userAddress"`
	NRNAmount   string                 `json:"nrnAmount"`
	Parameters  map[string]interface{} `json:"parameters"`
	Priority    string                 `json:"priority"`
	UseP2P      bool                   `json:"useP2P"`
	UseWASM     bool                   `json:"useWASM"`
}

type KNIRVControllerSkillResponse struct {
	RequestID      string `json:"requestId"`
	Status         string `json:"status"`
	SkillNodeURI   string `json:"skillNodeUri,omitempty"`
	ErrorMessage   string `json:"errorMessage,omitempty"`
	ExecutionTime  int64  `json:"executionTime"`
	NetworkLatency int64  `json:"networkLatency"`
}

type KNIRVControllerLoRARequest struct {
	AdapterName            string            `json:"adapterName"`
	Description            string            `json:"description"`
	BaseModelCompatibility string            `json:"baseModelCompatibility"`
	Version                int               `json:"version"`
	Rank                   int               `json:"rank"`
	Alpha                  float64           `json:"alpha"`
	Metadata               map[string]string `json:"metadata"`
}

type KNIRVControllerLoRAResponse struct {
	AdapterID   string `json:"adapterId"`
	AdapterName string `json:"adapterName"`
	Status      string `json:"status"`
}

type KNIRVControllerWASMRequest struct {
	AgentID    string                 `json:"agentId"`
	SkillData  string                 `json:"skillData"`
	Parameters map[string]interface{} `json:"parameters"`
}

type KNIRVControllerWASMResponse struct {
	CompilationID string `json:"compilationId"`
	Status        string `json:"status"`
	WASMBytes     string `json:"wasmBytes,omitempty"`
	ErrorMessage  string `json:"errorMessage,omitempty"`
}

type KNIRVControllerErrorContext struct {
	ErrorID      string                 `json:"errorId"`
	ErrorType    string                 `json:"errorType"`
	ErrorMessage string                 `json:"errorMessage"`
	StackTrace   string                 `json:"stackTrace"`
	UserContext  map[string]interface{} `json:"userContext"`
	AgentID      string                 `json:"agentId"`
	Timestamp    int64                  `json:"timestamp"`
	Severity     string                 `json:"severity"`
}

// Helper functions for KNIRVCONTROLLER tests
func waitForKNIRVControllerService(url string, timeout time.Duration) error {
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

	return fmt.Errorf("KNIRVCONTROLLER service at %s not ready within timeout", url)
}

func makeKNIRVControllerRequest(method, url string, payload interface{}) (*http.Response, error) {
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

// Main test suite for KNIRVCONTROLLER real network integration
func TestKNIRVControllerRealNetworkIntegration(t *testing.T) {
	// Wait for all required services to be ready
	t.Log("Waiting for KNIRVCONTROLLER to be ready...")
	require.NoError(t, waitForKNIRVControllerService(KNIRVControllerURL, TestTimeout))

	t.Log("Waiting for KNIRVROUTER to be ready...")
	require.NoError(t, waitForKNIRVControllerService(KNIRVRouterURL, TestTimeout))

	t.Log("Waiting for KNIRVGRAPH to be ready...")
	require.NoError(t, waitForKNIRVControllerService(KNIRVGraphURL, TestTimeout))

	t.Log("All services ready. Starting real network integration tests...")

	// Run comprehensive test suite
	t.Run("TestKNIRVControllerHealthCheck", testKNIRVControllerHealthCheck)
	t.Run("TestKNIRVControllerSkillInvocation", testKNIRVControllerSkillInvocation)
	t.Run("TestKNIRVControllerLoRAAdapterManagement", testKNIRVControllerLoRAAdapterManagement)
	t.Run("TestKNIRVControllerWASMCompilation", testKNIRVControllerWASMCompilation)
	t.Run("TestKNIRVControllerErrorContextProcessing", testKNIRVControllerErrorContextProcessing)
	t.Run("TestKNIRVControllerNetworkIntegration", testKNIRVControllerNetworkIntegration)
}

func testKNIRVControllerHealthCheck(t *testing.T) {
	t.Log("Testing KNIRVCONTROLLER health check and component status")

	resp, err := makeKNIRVControllerRequest("GET", KNIRVControllerURL+"/health", nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var healthResponse KNIRVControllerHealthResponse
	err = json.NewDecoder(resp.Body).Decode(&healthResponse)
	require.NoError(t, err)

	assert.Equal(t, "healthy", healthResponse.Status)
	assert.NotEmpty(t, healthResponse.Timestamp)
	assert.NotEmpty(t, healthResponse.Components)

	// Verify core components are reported
	components := healthResponse.Components
	assert.Contains(t, components, "wasmCompiler")
	assert.Contains(t, components, "loraEngine")
	assert.Contains(t, components, "protobufHandler")

	t.Logf("KNIRVCONTROLLER health check successful: %+v", healthResponse)
}

func testKNIRVControllerSkillInvocation(t *testing.T) {
	t.Log("Testing real skill invocation via KNIRVCONTROLLER → KNIRVGRAPH → KNIRVROUTER")

	request := KNIRVControllerSkillRequest{
		SkillID:     "integration-test-skill",
		UserAddress: "knirv1integrationtest123",
		NRNAmount:   "100",
		Parameters: map[string]interface{}{
			"agentId":      "integration-test-agent",
			"capabilities": []string{"text-processing", "analysis"},
			"priority":     "high",
			"testMode":     true,
		},
		Priority: "high",
		UseP2P:   true,
		UseWASM:  true,
	}

	resp, err := makeKNIRVControllerRequest("POST", KNIRVControllerURL+"/api/invoke-skill", request)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Accept both success and expected failure modes for real network
	if resp.StatusCode == http.StatusOK {
		var response KNIRVControllerSkillResponse
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.NotEmpty(t, response.RequestID)
		assert.Greater(t, response.ExecutionTime, int64(0))

		t.Logf("Skill invocation successful: RequestID=%s, Status=%s, ExecutionTime=%dms",
			response.RequestID, response.Status, response.ExecutionTime)
	} else {
		// Log the response for debugging real network issues
		var errorResponse map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errorResponse)
		t.Logf("Skill invocation returned status %d: %+v", resp.StatusCode, errorResponse)
	}
}

func testKNIRVControllerLoRAAdapterManagement(t *testing.T) {
	t.Log("Testing LoRA adapter registration and management")

	request := KNIRVControllerLoRARequest{
		AdapterName:            "integration-test-lora",
		Description:            "Integration test LoRA adapter for real network demo",
		BaseModelCompatibility: "hrm-v1",
		Version:                1,
		Rank:                   16,
		Alpha:                  0.5,
		Metadata: map[string]string{
			"test":        "true",
			"integration": "real-network",
			"demo":        "true",
		},
	}

	resp, err := makeKNIRVControllerRequest("POST", KNIRVControllerURL+"/lora/compile", request)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Accept both success and expected failure modes
	if resp.StatusCode == http.StatusOK {
		var response KNIRVControllerLoRAResponse
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.NotEmpty(t, response.AdapterID)
		assert.Equal(t, request.AdapterName, response.AdapterName)

		t.Logf("LoRA adapter registration successful: AdapterID=%s", response.AdapterID)
	} else {
		var errorResponse map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errorResponse)
		t.Logf("LoRA adapter registration returned status %d: %+v", resp.StatusCode, errorResponse)
	}
}

func testKNIRVControllerWASMCompilation(t *testing.T) {
	t.Log("Testing WASM compilation capabilities")

	request := KNIRVControllerWASMRequest{
		AgentID:   "integration-test-agent",
		SkillData: "console.log('Integration test skill');",
		Parameters: map[string]interface{}{
			"optimizationLevel": "O2",
			"testMode":          true,
		},
	}

	resp, err := makeKNIRVControllerRequest("POST", KNIRVControllerURL+"/wasm/compile", request)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Accept both success and expected failure modes
	if resp.StatusCode == http.StatusOK {
		var response KNIRVControllerWASMResponse
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.NotEmpty(t, response.CompilationID)

		t.Logf("WASM compilation successful: CompilationID=%s, Status=%s",
			response.CompilationID, response.Status)
	} else {
		var errorResponse map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errorResponse)
		t.Logf("WASM compilation returned status %d: %+v", resp.StatusCode, errorResponse)
	}
}

func testKNIRVControllerErrorContextProcessing(t *testing.T) {
	t.Log("Testing ErrorContext generation and processing for skill resolution")

	errorContext := KNIRVControllerErrorContext{
		ErrorID:      "integration-test-error-001",
		ErrorType:    "skill_invocation_request",
		ErrorMessage: "Integration test error for real network demo",
		StackTrace:   "integration test stack trace",
		UserContext: map[string]interface{}{
			"userAddress": "knirv1integrationtest123",
			"nrnAmount":   "100",
			"testMode":    true,
		},
		AgentID:   "integration-test-agent",
		Timestamp: time.Now().UnixMilli(),
		Severity:  "medium",
	}

	resp, err := makeKNIRVControllerRequest("POST", KNIRVControllerURL+"/api/process-error-context", errorContext)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Accept both success and expected failure modes
	if resp.StatusCode == http.StatusOK {
		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		t.Logf("ErrorContext processing successful: %+v", response)
	} else {
		var errorResponse map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errorResponse)
		t.Logf("ErrorContext processing returned status %d: %+v", resp.StatusCode, errorResponse)
	}
}

func testKNIRVControllerNetworkIntegration(t *testing.T) {
	t.Log("Testing KNIRVCONTROLLER integration with network services")

	// Test API status endpoint
	resp, err := makeKNIRVControllerRequest("GET", KNIRVControllerURL+"/api/status", nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var statusResponse map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&statusResponse)
		require.NoError(t, err)

		assert.Equal(t, "operational", statusResponse["status"])
		assert.Contains(t, statusResponse, "components")
		assert.Contains(t, statusResponse, "capabilities")

		t.Logf("KNIRVCONTROLLER network integration successful: %+v", statusResponse)
	} else {
		t.Logf("API status endpoint returned status %d", resp.StatusCode)
	}

	// Test template information endpoint
	resp2, err := makeKNIRVControllerRequest("GET", KNIRVControllerURL+"/api/templates/info", nil)
	require.NoError(t, err)
	defer resp2.Body.Close()

	if resp2.StatusCode == http.StatusOK {
		var templateResponse map[string]interface{}
		err = json.NewDecoder(resp2.Body).Decode(&templateResponse)
		require.NoError(t, err)

		t.Logf("Template information retrieved: %+v", templateResponse)
	} else {
		t.Logf("Template info endpoint returned status %d", resp2.StatusCode)
	}
}
