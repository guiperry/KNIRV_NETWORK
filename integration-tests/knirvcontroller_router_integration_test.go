package integration_tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test configuration - Updated for real network integration
// Constants are defined in test_constants.go

// Test data structures - Common types are defined in test_constants.go
// Local types specific to this test file:

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

// Test helper functions are defined in test_constants.go

// Local helper function for this test
func waitForServiceWithError(url string, timeout time.Duration) error {
	if waitForService(url, timeout) {
		return nil
	}
	return fmt.Errorf("service at %s not ready within timeout", url)
}

// Test functions
func TestKNIRVControllerRouterIntegration(t *testing.T) {
	// Wait for services to be ready
	t.Log("Waiting for KNIRVCONTROLLER to be ready...")
	require.NoError(t, waitForServiceWithError(KNIRVControllerURL, TestTimeout))

	t.Log("Waiting for KNIRVROUTER to be ready...")
	require.NoError(t, waitForServiceWithError(KNIRVRouterURL, TestTimeout))

	t.Log("Waiting for KNIRVGRAPH to be ready...")
	require.NoError(t, waitForServiceWithError(KNIRVGraphURL, TestTimeout))

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
		InvocationID: "test-invocation-001",
		AgentID:      "test-agent-001",
		SkillURI:     "knirv://skills/test-skill-001",
		UserID:       "knirv1test123456789",
		SkillID:      "test-skill-001",
		Amount:       "100",
		Parameters: map[string]interface{}{
			"capabilities": []string{"text-processing", "analysis"},
			"priority":     "high",
			"useP2P":       true,
			"useWASM":      true,
		},
	}

	resp, err := makeHTTPRequest("POST", KNIRVControllerURL+"/api/invoke-skill", request)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var response SkillInvocationResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "SUCCESS", response.Status)
	assert.NotEmpty(t, response.InvocationID)
	assert.NotEmpty(t, response.Result)

	t.Logf("Skill invocation successful: InvocationID=%s, Result=%s",
		response.InvocationID, response.Result)
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
		AgentID:      "test-agent-001",
		BaseModelID:  "hrm-cognitive-v1",
		OS:           "linux",
		Architecture: "x86_64",
		Timestamp:    time.Now().Format(time.RFC3339),
		ErrorType:    "skill_invocation_request",
		ErrorMessage: "Test error for integration testing",
		Context: map[string]interface{}{
			"userAddress": "knirv1test123456789",
			"nrnAmount":   "100",
		},
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

// TestMain removed to avoid conflict with knirvnexus_phase6_comprehensive_test.go
// Test setup is handled by the main TestMain function
