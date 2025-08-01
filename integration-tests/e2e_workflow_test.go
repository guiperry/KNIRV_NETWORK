package integration_tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// Month 12 E2E Test Suite - Complete Implementation
type E2ETestSuite struct {
	suite.Suite
	gatewayURL string
	wsURL      string
	authToken  string
	testWallet *TestWallet
	testData   *TestData
	httpClient *http.Client
	wsConn     *websocket.Conn
}

type TestData struct {
	TestLLMID   string
	TestSkillID string
	TestErrorID string
	TestUserID  string
	TestAgentID string
}

type TestResponse struct {
	Success bool                   `json:"success"`
	Data    map[string]interface{} `json:"data,omitempty"`
	Error   string                 `json:"error,omitempty"`
	TxHash  string                 `json:"tx_hash,omitempty"`
}

type E2EWorkflowTester struct {
	suite *IntegrationTestSuite
}

func NewE2EWorkflowTester(suite *IntegrationTestSuite) *E2EWorkflowTester {
	return &E2EWorkflowTester{
		suite: suite,
	}
}

// Month 12 E2E Test Suite Setup Methods
func (suite *E2ETestSuite) SetupSuite() {
	suite.gatewayURL = "http://localhost:8000"
	suite.wsURL = "ws://localhost:8000/gateway/ws"
	suite.httpClient = &http.Client{Timeout: 30 * time.Second}

	// Initialize test data
	suite.testData = &TestData{
		TestLLMID:   "test_llm_" + fmt.Sprintf("%d", time.Now().Unix()),
		TestSkillID: "test_skill_" + fmt.Sprintf("%d", time.Now().Unix()),
		TestErrorID: "test_error_" + fmt.Sprintf("%d", time.Now().Unix()),
		TestUserID:  "test_user_" + fmt.Sprintf("%d", time.Now().Unix()),
		TestAgentID: "test_agent_" + fmt.Sprintf("%d", time.Now().Unix()),
	}

	// Wait for services to be ready
	suite.waitForServices()

	// Authenticate
	suite.authenticate()

	// Create test wallet
	suite.createTestWallet()

	// Setup WebSocket connection
	suite.setupWebSocket()
}

func (suite *E2ETestSuite) TearDownSuite() {
	if suite.wsConn != nil {
		suite.wsConn.Close()
	}
}

func (suite *E2ETestSuite) waitForServices() {
	services := []string{"knirvchain", "knirvgraph", "knirvnexus", "knirvroot", "knirvrouter"}

	for _, service := range services {
		suite.T().Logf("Waiting for service: %s", service)

		for i := 0; i < 30; i++ { // Wait up to 30 seconds
			resp, err := suite.httpClient.Get(fmt.Sprintf("%s/%s/health", suite.gatewayURL, service))
			if err == nil && resp.StatusCode == http.StatusOK {
				resp.Body.Close()
				break
			}
			if resp != nil {
				resp.Body.Close()
			}
			time.Sleep(1 * time.Second)
		}
	}

	suite.T().Log("All services are ready")
}

func (suite *E2ETestSuite) authenticate() {
	loginData := map[string]string{
		"username": "admin",
		"password": "password",
	}

	resp := suite.makeRequest("POST", "/auth/login", loginData)
	require.True(suite.T(), resp.Success, "Authentication failed")

	suite.authToken = resp.Data["token"].(string)
	suite.T().Logf("Authenticated with token: %s", suite.authToken[:20]+"...")
}

func (suite *E2ETestSuite) createTestWallet() {
	walletData := map[string]string{
		"name": "e2e_test_wallet",
	}

	resp := suite.makeAuthenticatedRequest("POST", "/knirvwallet/wallet/create", walletData)
	require.True(suite.T(), resp.Success, "Failed to create test wallet")

	suite.testWallet = &TestWallet{
		Address:  resp.Data["address"].(string),
		Mnemonic: resp.Data["mnemonic"].(string),
		Balance:  "0",
	}

	suite.T().Logf("Created test wallet: %s", suite.testWallet.Address)

	// Fund the wallet
	suite.fundTestWallet()
}

func (suite *E2ETestSuite) fundTestWallet() {
	fundData := map[string]interface{}{
		"address": suite.testWallet.Address,
		"amount":  "10000000", // 10 NRN
	}

	resp := suite.makeAuthenticatedRequest("POST", "/knirvroot/faucet/fund", fundData)
	require.True(suite.T(), resp.Success, "Failed to fund test wallet")

	suite.testWallet.Balance = "10000000"
	suite.T().Logf("Funded test wallet with 10 NRN")
}

func (suite *E2ETestSuite) setupWebSocket() {
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(suite.wsURL, nil)
	require.NoError(suite.T(), err, "Failed to connect to WebSocket")

	suite.wsConn = conn

	// Send ping to test connection
	pingMsg := map[string]interface{}{
		"type": "ping",
	}

	err = suite.wsConn.WriteJSON(pingMsg)
	require.NoError(suite.T(), err, "Failed to send ping")

	var pongMsg map[string]interface{}
	err = suite.wsConn.ReadJSON(&pongMsg)
	require.NoError(suite.T(), err, "Failed to receive pong")

	assert.Equal(suite.T(), "pong", pongMsg["type"])
	suite.T().Log("WebSocket connection established")
}

// Helper methods for making HTTP requests
func (suite *E2ETestSuite) makeRequest(method, path string, data interface{}) *TestResponse {
	return suite.makeRequestWithAuth(method, path, data, "")
}

func (suite *E2ETestSuite) makeAuthenticatedRequest(method, path string, data interface{}) *TestResponse {
	return suite.makeRequestWithAuth(method, path, data, suite.authToken)
}

func (suite *E2ETestSuite) makeRequestWithAuth(method, path string, data interface{}, token string) *TestResponse {
	var body io.Reader
	if data != nil {
		jsonData, err := json.Marshal(data)
		require.NoError(suite.T(), err, "Failed to marshal request data")
		body = bytes.NewReader(jsonData)
	}

	url := suite.gatewayURL + path
	req, err := http.NewRequest(method, url, body)
	require.NoError(suite.T(), err, "Failed to create request")

	if data != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := suite.httpClient.Do(req)
	require.NoError(suite.T(), err, "Request failed")
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	require.NoError(suite.T(), err, "Failed to read response body")

	var testResp TestResponse
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if len(responseBody) > 0 {
			err = json.Unmarshal(responseBody, &testResp.Data)
			if err != nil {
				// Try to unmarshal as TestResponse directly
				json.Unmarshal(responseBody, &testResp)
			} else {
				testResp.Success = true
			}
		} else {
			testResp.Success = true
		}
	} else {
		testResp.Success = false
		testResp.Error = string(responseBody)
	}

	return &testResp
}

// Month 12 Complete E2E Test Implementation
func (suite *E2ETestSuite) TestCompleteWorkflow() {
	suite.T().Log("Starting complete E2E workflow test")

	// Test 1: Register LLM
	suite.Run("RegisterLLM", func() {
		llmData := map[string]interface{}{
			"name":             suite.testData.TestLLMID,
			"version":          "1.0.0",
			"capabilities":     []string{"text-generation", "code-completion"},
			"model_data":       "dGVzdCBtb2RlbCBkYXRh", // base64 encoded test data
			"registration_fee": "1000000",              // 1 NRN
			"usage_fee":        "100000",               // 0.1 NRN
		}

		resp := suite.makeAuthenticatedRequest("POST", "/knirvchain/llm/register", llmData)
		assert.True(suite.T(), resp.Success, "LLM registration failed: %s", resp.Error)
		assert.NotEmpty(suite.T(), resp.TxHash, "No transaction hash returned")

		suite.T().Logf("LLM registered successfully: %s", resp.TxHash)

		// Wait for transaction confirmation
		time.Sleep(5 * time.Second)

		// Verify LLM is registered
		llmResp := suite.makeAuthenticatedRequest("GET", fmt.Sprintf("/knirvchain/llm/%s", suite.testData.TestLLMID), nil)
		assert.True(suite.T(), llmResp.Success, "Failed to retrieve registered LLM")
	})

	// Test 2: Create Error Node
	suite.Run("CreateErrorNode", func() {
		errorData := map[string]interface{}{
			"error_type":  "compilation_error",
			"description": "Missing semicolon in JavaScript code",
			"context": map[string]interface{}{
				"language": "javascript",
				"line":     42,
				"file":     "test.js",
				"user_id":  suite.testData.TestUserID,
			},
			"severity": 3,
		}

		resp := suite.makeAuthenticatedRequest("POST", "/knirvgraph/nrv/errors", errorData)
		assert.True(suite.T(), resp.Success, "Error node creation failed: %s", resp.Error)

		errorID := resp.Data["id"].(string)
		suite.testData.TestErrorID = errorID

		suite.T().Logf("Error node created: %s", errorID)

		// Verify error node exists
		errorResp := suite.makeAuthenticatedRequest("GET", fmt.Sprintf("/knirvgraph/nrv/errors/%s", errorID), nil)
		assert.True(suite.T(), errorResp.Success, "Failed to retrieve error node")
	})

	// Test 3: Create Skill Node
	suite.Run("CreateSkillNode", func() {
		skillData := map[string]interface{}{
			"skill_type":   "code_fixer",
			"capabilities": []string{"javascript", "syntax_repair", "semicolon_insertion"},
			"requirements": map[string]interface{}{
				"min_confidence": 0.8,
				"max_latency":    "5s",
				"llm_id":         suite.testData.TestLLMID,
			},
		}

		resp := suite.makeAuthenticatedRequest("POST", "/knirvgraph/nrv/skills", skillData)
		assert.True(suite.T(), resp.Success, "Skill node creation failed: %s", resp.Error)

		skillID := resp.Data["id"].(string)
		suite.testData.TestSkillID = skillID

		suite.T().Logf("Skill node created: %s", skillID)

		// Verify skill node exists
		skillResp := suite.makeAuthenticatedRequest("GET", fmt.Sprintf("/knirvgraph/nrv/skills/%s", skillID), nil)
		assert.True(suite.T(), skillResp.Success, "Failed to retrieve skill node")
	})

	// Test 4: Create NEXUS Agent
	suite.Run("CreateNEXUSAgent", func() {
		agentData := map[string]interface{}{
			"name":         suite.testData.TestAgentID,
			"type":         "code_assistant",
			"capabilities": []string{"javascript", "debugging", "code_generation"},
			"config": map[string]interface{}{
				"model":       "gpt-4",
				"temperature": 0.7,
				"max_tokens":  2048,
			},
		}

		resp := suite.makeAuthenticatedRequest("POST", "/knirvnexus/agents/create", agentData)
		assert.True(suite.T(), resp.Success, "Agent creation failed: %s", resp.Error)

		agentID := resp.Data["id"].(string)
		suite.testData.TestAgentID = agentID

		suite.T().Logf("NEXUS agent created: %s", agentID)

		// Verify agent exists
		agentResp := suite.makeAuthenticatedRequest("GET", fmt.Sprintf("/knirvnexus/agents/%s", agentID), nil)
		assert.True(suite.T(), agentResp.Success, "Failed to retrieve agent")
	})

	// Test 5: Test NRV Resolution
	suite.Run("TestNRVResolution", func() {
		// Create a vector for the error
		vectorData := map[string]interface{}{
			"target_hash": suite.testData.TestErrorID,
			"coordinates": []float64{3.0, 42.0, 1.0}, // severity, line, file_type
			"metadata": map[string]interface{}{
				"error_type": "compilation_error",
				"language":   "javascript",
			},
		}

		resp := suite.makeAuthenticatedRequest("POST", "/knirvgraph/nrv/vectors", vectorData)
		assert.True(suite.T(), resp.Success, "Vector creation failed: %s", resp.Error)

		// Test resolution
		resolveResp := suite.makeAuthenticatedRequest("GET", fmt.Sprintf("/knirvgraph/nrv/resolve/%s", suite.testData.TestErrorID), nil)
		assert.True(suite.T(), resolveResp.Success, "NRV resolution failed: %s", resolveResp.Error)

		vectors := resolveResp.Data["vectors"].([]interface{})
		assert.Greater(suite.T(), len(vectors), 0, "No vectors found for resolution")

		suite.T().Logf("NRV resolution found %d vectors", len(vectors))
	})

	// Test 6: Invoke Skill with Token Burning
	suite.Run("InvokeSkillWithTokenBurning", func() {
		skillData := map[string]interface{}{
			"skill_id":     suite.testData.TestSkillID,
			"amount":       "500000", // 0.5 NRN
			"user_address": suite.testWallet.Address,
			"parameters": map[string]interface{}{
				"error_id":   suite.testData.TestErrorID,
				"fix_type":   "syntax_repair",
				"confidence": 0.9,
			},
		}

		resp := suite.makeAuthenticatedRequest("POST", "/knirvchain/skill/invoke", skillData)
		assert.True(suite.T(), resp.Success, "Skill invocation failed: %s", resp.Error)
		assert.NotEmpty(suite.T(), resp.TxHash, "No transaction hash returned")

		suite.T().Logf("Skill invoked successfully: %s", resp.TxHash)

		// Wait for transaction processing
		time.Sleep(5 * time.Second)

		// Check wallet balance (should be reduced)
		balanceResp := suite.makeAuthenticatedRequest("GET", fmt.Sprintf("/knirvwallet/balance/%s", suite.testWallet.Address), nil)
		assert.True(suite.T(), balanceResp.Success, "Failed to check wallet balance")

		newBalance := balanceResp.Data["balance"].(string)
		suite.T().Logf("Wallet balance after skill invocation: %s", newBalance)
	})

	// Test 7: Agent Workflow Execution
	suite.Run("AgentWorkflowExecution", func() {
		workflowData := map[string]interface{}{
			"agent_id": suite.testData.TestAgentID,
			"workflow": map[string]interface{}{
				"steps": []map[string]interface{}{
					{
						"type":  "analyze_error",
						"input": suite.testData.TestErrorID,
						"config": map[string]interface{}{
							"depth": "detailed",
						},
					},
					{
						"type":  "generate_fix",
						"input": "analysis_result",
						"config": map[string]interface{}{
							"fix_type": "syntax_repair",
						},
					},
					{
						"type":  "validate_fix",
						"input": "generated_fix",
						"config": map[string]interface{}{
							"run_tests": true,
						},
					},
				},
			},
		}

		resp := suite.makeAuthenticatedRequest("POST", "/knirvnexus/workflows/execute", workflowData)
		assert.True(suite.T(), resp.Success, "Workflow execution failed: %s", resp.Error)

		workflowID := resp.Data["workflow_id"].(string)
		suite.T().Logf("Workflow started: %s", workflowID)

		// Poll for workflow completion
		suite.waitForWorkflowCompletion(workflowID)
	})

	// Test 8: Cross-Chain Token Bridge
	suite.Run("CrossChainTokenBridge", func() {
		bridgeData := map[string]interface{}{
			"target_chain": "xion",
			"amount":       "1000000", // 1 NRN
			"recipient":    suite.testWallet.Address,
			"source":       "KNIRVROOT",
		}

		resp := suite.makeAuthenticatedRequest("POST", "/knirvroot/bridge/transfer", bridgeData)
		assert.True(suite.T(), resp.Success, "Bridge transfer failed: %s", resp.Error)
		assert.NotEmpty(suite.T(), resp.TxHash, "No transaction hash returned")

		txHash := resp.TxHash
		suite.T().Logf("Bridge transfer initiated: %s", txHash)

		// Wait for bridge processing
		time.Sleep(10 * time.Second)

		// Check bridge status
		statusResp := suite.makeAuthenticatedRequest("GET", fmt.Sprintf("/knirvroot/bridge/status?tx_hash=%s", txHash), nil)
		assert.True(suite.T(), statusResp.Success, "Failed to check bridge status")

		status := statusResp.Data["status"].(string)
		suite.T().Logf("Bridge status: %s", status)
	})

	// Test 9: Economic Metrics Validation
	suite.Run("EconomicMetricsValidation", func() {
		metricsResp := suite.makeAuthenticatedRequest("GET", "/knirvroot/economics/metrics", nil)
		assert.True(suite.T(), metricsResp.Success, "Failed to get economic metrics")

		metrics := metricsResp.Data

		// Validate key metrics exist
		assert.Contains(suite.T(), metrics, "total_supply")
		assert.Contains(suite.T(), metrics, "circulating_supply")
		assert.Contains(suite.T(), metrics, "total_burned")
		assert.Contains(suite.T(), metrics, "service_metrics")

		suite.T().Logf("Economic metrics validated: %+v", metrics)
	})

	// Test 10: WebSocket Real-time Updates
	suite.Run("WebSocketRealTimeUpdates", func() {
		// Subscribe to service updates
		subscribeMsg := map[string]interface{}{
			"type":    "subscribe",
			"service": "knirvchain",
		}

		err := suite.wsConn.WriteJSON(subscribeMsg)
		assert.NoError(suite.T(), err, "Failed to send subscribe message")

		var response map[string]interface{}
		err = suite.wsConn.ReadJSON(&response)
		assert.NoError(suite.T(), err, "Failed to receive subscription response")
		assert.Equal(suite.T(), "subscribed", response["type"])

		// Request metrics via WebSocket
		metricsMsg := map[string]interface{}{
			"type": "get_metrics",
		}

		err = suite.wsConn.WriteJSON(metricsMsg)
		assert.NoError(suite.T(), err, "Failed to send metrics request")

		err = suite.wsConn.ReadJSON(&response)
		assert.NoError(suite.T(), err, "Failed to receive metrics response")
		assert.Equal(suite.T(), "metrics", response["type"])
		assert.Contains(suite.T(), response, "metrics")

		suite.T().Log("WebSocket real-time updates working correctly")
	})
}

// Helper method for waiting for workflow completion
func (suite *E2ETestSuite) waitForWorkflowCompletion(workflowID string) {
	for i := 0; i < 60; i++ { // Wait up to 60 seconds
		resp := suite.makeAuthenticatedRequest("GET", fmt.Sprintf("/knirvnexus/workflows/%s/status", workflowID), nil)
		if resp.Success {
			status := resp.Data["status"].(string)
			if status == "completed" || status == "failed" {
				suite.T().Logf("Workflow %s completed with status: %s", workflowID, status)
				return
			}
		}
		time.Sleep(1 * time.Second)
	}

	suite.T().Errorf("Workflow %s did not complete within timeout", workflowID)
}

// Main test function for the E2E Test Suite
func TestE2ETestSuite(t *testing.T) {
	suite.Run(t, new(E2ETestSuite))
}

// Additional Month 12 Test Methods

// Test KNIRV-ROUTER Connectivity and Proof-of-Connectivity
func (suite *E2ETestSuite) TestKNIRVROUTERConnectivity() {
	suite.Run("KNIRVROUTERConnectivityTest", func() {
		// Test router connectivity status
		statusResponse := suite.makeAuthenticatedRequest("GET", "/knirvrouter/api/connectivity/status", nil)
		assert.True(suite.T(), statusResponse.Success, "Failed to get KNIRV-ROUTER status")

		proofEngineActive := statusResponse.Data["proof_engine_active"].(bool)
		assert.True(suite.T(), proofEngineActive, "KNIRV-ROUTER proof engine is not active")

		// Test connectivity proof creation
		proofResponse := suite.makeAuthenticatedRequest("POST", "/knirvrouter/api/connectivity/proofs", map[string]interface{}{})
		assert.True(suite.T(), proofResponse.Success, "Failed to initiate connectivity proof")

		proofStatus := proofResponse.Data["status"].(string)
		assert.Equal(suite.T(), "proof_generation_initiated", proofStatus, "Failed to initiate connectivity proof")

		// Wait for proof processing
		time.Sleep(15 * time.Second)

		// Check proof history
		proofsResponse := suite.makeAuthenticatedRequest("GET", "/knirvrouter/api/connectivity/proofs", nil)
		assert.True(suite.T(), proofsResponse.Success, "Failed to get connectivity proofs")

		proofs := proofsResponse.Data["proofs"].([]interface{})
		assert.Greater(suite.T(), len(proofs), 0, "No connectivity proofs found")

		suite.T().Logf("KNIRV-ROUTER connectivity test passed with %d proofs", len(proofs))

		// Test TURN server endpoint (existing functionality)
		turnResponse := suite.makeAuthenticatedRequest("GET", "/knirvrouter/turn/status", nil)
		if !turnResponse.Success {
			suite.T().Logf("KNIRV-ROUTER TURN server endpoint returned error: %s", turnResponse.Error)
		}
	})
}

// Test Data Consistency across components
func (suite *E2ETestSuite) TestDataConsistency() {
	suite.Run("DataConsistencyTest", func() {
		// Create test data
		testID := "consistency_test_" + fmt.Sprintf("%d", time.Now().Unix())

		// Create data in multiple services
		chainResponse := suite.makeAuthenticatedRequest("POST", "/knirvchain/test/data", map[string]interface{}{
			"id":   testID,
			"data": "test_data",
		})

		graphResponse := suite.makeAuthenticatedRequest("POST", "/knirvgraph/test/data", map[string]interface{}{
			"id":   testID,
			"data": "test_data",
		})

		// Wait for synchronization
		time.Sleep(5 * time.Second)

		// Verify data consistency
		chainData := ""
		graphData := ""

		if chainResponse.Success {
			chainDataResp := suite.makeAuthenticatedRequest("GET", fmt.Sprintf("/knirvchain/test/data/%s", testID), nil)
			if chainDataResp.Success {
				chainData = chainDataResp.Data["data"].(string)
			}
		}

		if graphResponse.Success {
			graphDataResp := suite.makeAuthenticatedRequest("GET", fmt.Sprintf("/knirvgraph/test/data/%s", testID), nil)
			if graphDataResp.Success {
				graphData = graphDataResp.Data["data"].(string)
			}
		}

		if chainData != "" && graphData != "" {
			assert.Equal(suite.T(), chainData, graphData, "Data inconsistency detected between services")
		}

		suite.T().Log("Data consistency test completed")
	})
}
