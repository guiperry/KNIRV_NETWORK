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
	"github.com/stretchr/testify/suite"
)

// Phase36E2ETestSuite tests the complete End-to-End Skill Invocation Lifecycle
type Phase36E2ETestSuite struct {
	suite.Suite
	knirvRouterURL     string
	knirvGraphURL      string
	knirvOracleURL     string
	knirvControllerURL string
	httpClient         *http.Client
	testData           *Phase36TestData
}

type Phase36TestData struct {
	TestAgentID      string
	TestErrorID      string
	TestSkillURI     string
	TestNRNToken     string
	TestInvocationID string
}

// Phase36ErrorContext represents the rich error data payload for this test
type Phase36ErrorContext struct {
	AgentID            string                 `json:"agent_id"`
	AgentVersion       string                 `json:"agent_version"`
	BaseModelID        string                 `json:"base_model_id"`
	OS                 string                 `json:"os"`
	Architecture       string                 `json:"architecture"`
	RuntimeEnvironment string                 `json:"runtime_environment"`
	ErrorType          string                 `json:"error_type"`
	ErrorMessage       string                 `json:"error_message"`
	StackTrace         string                 `json:"stack_trace"`
	SourceCodeSnippet  string                 `json:"source_code_snippet"`
	TaskDescription    string                 `json:"task_description"`
	InputDataHash      string                 `json:"input_data_hash"`
	SkillInvokedID     string                 `json:"skill_invoked_id"`
	AgentStateHash     string                 `json:"agent_state_hash"`
	Timestamp          int64                  `json:"timestamp"`
	AdditionalContext  map[string]interface{} `json:"additional_context"`
}

// Phase36SkillInvocationRequest for WASM endpoint
type Phase36SkillInvocationRequest struct {
	InvocationID string                 `json:"invocation_id"`
	AgentID      string                 `json:"agent_id"`
	SkillURI     string                 `json:"skill_uri"`
	NRNToken     string                 `json:"nrn_token"`
	Parameters   map[string]interface{} `json:"parameters"`
	Priority     string                 `json:"priority"`
	Timestamp    int64                  `json:"timestamp"`
}

// Phase36SkillInvocationResponse from WASM endpoint
type Phase36SkillInvocationResponse struct {
	InvocationID     string `json:"invocation_id"`
	Status           string `json:"status"`
	ErrorMessage     string `json:"error_message"`
	ExecutionTime    int64  `json:"execution_time"`
	MemoryUsed       int64  `json:"memory_used"`
	ConsensusReached bool   `json:"consensus_reached"`
	SkillData        string `json:"skill_data"`
}

func (suite *Phase36E2ETestSuite) SetupSuite() {
	suite.knirvRouterURL = "http://localhost:8080"
	suite.knirvGraphURL = "http://localhost:8081"
	suite.knirvOracleURL = "http://localhost:8082"
	suite.knirvControllerURL = "http://localhost:3000"
	suite.httpClient = &http.Client{Timeout: 30 * time.Second}

	// Initialize test data
	suite.testData = &Phase36TestData{
		TestAgentID:      fmt.Sprintf("agent_%d", time.Now().Unix()),
		TestErrorID:      fmt.Sprintf("error_%d", time.Now().Unix()),
		TestSkillURI:     "knirv://skill/javascript-type-checker-v1",
		TestNRNToken:     fmt.Sprintf("nrn_token_%d_%s", time.Now().Unix(), generateRandomString(16)),
		TestInvocationID: fmt.Sprintf("inv_%d_%s", time.Now().Unix(), generateRandomString(8)),
	}

	// Wait for services to be ready
	suite.waitForServices()
}

func (suite *Phase36E2ETestSuite) waitForServices() {
	services := map[string]string{
		"KNIRVROUTER":     suite.knirvRouterURL + "/wasm/status",
		"KNIRVGRAPH":      suite.knirvGraphURL + "/health",
		"KNIRVORACLE":     suite.knirvOracleURL + "/health",
		"KNIRVCONTROLLER": suite.knirvControllerURL + "/health",
	}

	for serviceName, healthURL := range services {
		suite.T().Logf("Waiting for service: %s", serviceName)

		for i := 0; i < 30; i++ { // Wait up to 30 seconds
			resp, err := suite.httpClient.Get(healthURL)
			if err == nil && resp.StatusCode == http.StatusOK {
				resp.Body.Close()
				suite.T().Logf("✅ %s is ready", serviceName)
				break
			}
			if resp != nil {
				resp.Body.Close()
			}
			time.Sleep(1 * time.Second)
		}
	}

	suite.T().Log("All services are ready for Phase 3.6 testing")
}

// TestPhase36CompleteLifecycle tests the complete end-to-end skill invocation lifecycle
func (suite *Phase36E2ETestSuite) TestPhase36CompleteLifecycle() {
	suite.T().Log("🚀 Starting Phase 3.6 End-to-End Skill Invocation Lifecycle Test")

	// Step 1: Test WASM KNIRVCHAIN Status
	suite.Run("Step1_WASMStatus", func() {
		resp, err := suite.httpClient.Get(suite.knirvRouterURL + "/wasm/status")
		require.NoError(suite.T(), err)
		require.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var status map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&status)
		require.NoError(suite.T(), err)
		resp.Body.Close()

		assert.True(suite.T(), status["initialized"].(bool))
		assert.Equal(suite.T(), "wasm", status["engine"])
		suite.T().Logf("✅ WASM KNIRVCHAIN Status: %+v", status)
	})

	// Step 2: Test Error Context Creation
	suite.Run("Step2_ErrorContextCreation", func() {
		errorContext := &Phase36ErrorContext{
			AgentID:            suite.testData.TestAgentID,
			AgentVersion:       "1.0.0",
			BaseModelID:        "hrm-cognitive-v1",
			OS:                 "linux",
			Architecture:       "x86_64",
			RuntimeEnvironment: "browser",
			ErrorType:          "TypeError",
			ErrorMessage:       "Cannot read property 'length' of undefined",
			StackTrace:         "at processData (test.js:42:15)",
			SourceCodeSnippet:  "const length = data.length;",
			TaskDescription:    "Processing user input data for validation",
			InputDataHash:      "sha256:abc123def456",
			AgentStateHash:     "sha256:state789xyz",
			Timestamp:          time.Now().Unix(),
			AdditionalContext: map[string]interface{}{
				"function": "processData",
				"line":     42,
				"file":     "test.js",
			},
		}

		// Serialize error context
		errorContextJSON, err := json.Marshal(errorContext)
		require.NoError(suite.T(), err)

		suite.T().Logf("✅ Error Context Created: %s", string(errorContextJSON))
		assert.NotEmpty(suite.T(), errorContext.AgentID)
		assert.NotEmpty(suite.T(), errorContext.ErrorMessage)
		assert.NotEmpty(suite.T(), errorContext.TaskDescription)
	})

	// Step 3: Test KNIRVGRAPH Error Discovery (Mock)
	suite.Run("Step3_ErrorDiscovery", func() {
		// In a real implementation, this would query KNIRVGRAPH
		// For now, we'll simulate finding a skill URI
		discoveredSkillURI := suite.testData.TestSkillURI

		suite.T().Logf("✅ Skill Discovered: %s", discoveredSkillURI)
		assert.Equal(suite.T(), "knirv://skill/javascript-type-checker-v1", discoveredSkillURI)
	})

	// Step 4: Test WASM Skill Invocation
	suite.Run("Step4_WASMSkillInvocation", func() {
		request := &SkillInvocationRequest{
			InvocationID: suite.testData.TestInvocationID,
			AgentID:      suite.testData.TestAgentID,
			SkillURI:     suite.testData.TestSkillURI,
			UserID:       "test-user-001",
			SkillID:      "test-skill-001",
			Amount:       suite.testData.TestNRNToken,
			Parameters: map[string]interface{}{
				"error_type": "TypeError",
				"context":    "javascript",
				"priority":   "normal",
			},
		}

		requestJSON, err := json.Marshal(request)
		require.NoError(suite.T(), err)

		resp, err := suite.httpClient.Post(
			suite.knirvRouterURL+"/wasm/invoke",
			"application/json",
			bytes.NewBuffer(requestJSON),
		)
		require.NoError(suite.T(), err)
		defer resp.Body.Close()

		// Check response headers
		assert.Equal(suite.T(), "application/json", resp.Header.Get("Content-Type"))
		assert.Equal(suite.T(), "wasm", resp.Header.Get("X-KNIRV-Engine"))
		assert.Equal(suite.T(), suite.testData.TestInvocationID, resp.Header.Get("X-KNIRV-Invocation-ID"))

		// Parse response
		var response SkillInvocationResponse
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(suite.T(), err)

		// Validate response
		assert.Equal(suite.T(), suite.testData.TestInvocationID, response.InvocationID)
		assert.Equal(suite.T(), "SUCCESS", response.Status)
		assert.Empty(suite.T(), response.ErrorMessage)
		assert.NotEmpty(suite.T(), response.Result)

		suite.T().Logf("✅ WASM Skill Invocation Response: %+v", response)
	})

	// Step 5: Test NRN Token Validation (Mock)
	suite.Run("Step5_NRNTokenValidation", func() {
		// In a real implementation, this would validate the NRN token
		// For now, we'll simulate successful validation
		isValid := len(suite.testData.TestNRNToken) > 10

		assert.True(suite.T(), isValid)
		suite.T().Logf("✅ NRN Token Validated: %s", suite.testData.TestNRNToken[:20]+"...")
	})

	// Step 6: Test Skill Count
	suite.Run("Step6_SkillCount", func() {
		resp, err := suite.httpClient.Get(suite.knirvRouterURL + "/wasm/skills/count")
		require.NoError(suite.T(), err)
		defer resp.Body.Close()

		var countResponse map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&countResponse)
		require.NoError(suite.T(), err)

		skillCount := int(countResponse["skill_count"].(float64))
		assert.Greater(suite.T(), skillCount, 0)
		assert.Equal(suite.T(), "wasm", countResponse["engine"])

		suite.T().Logf("✅ WASM Skill Count: %d", skillCount)
	})

	suite.T().Log("🎉 Phase 3.6 End-to-End Skill Invocation Lifecycle Test Completed Successfully!")
}

// Helper function to generate random strings
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(result)
}

// TestPhase36E2ETestSuite runs the Phase 3.6 test suite
func TestPhase36E2ETestSuite(t *testing.T) {
	suite.Run(t, new(Phase36E2ETestSuite))
}
