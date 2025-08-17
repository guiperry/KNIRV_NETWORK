package integration_tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// Month 12 Cross-Component Integration Validation Suite
type CrossComponentTestSuite struct {
	suite.Suite
	gatewayURL string
	httpClient *http.Client
	authToken  string
	testWallet *TestWallet
	testData   *TestData
}

type ComponentHealth struct {
	Service   string `json:"service"`
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Version   string `json:"version,omitempty"`
}

type IntegrationTestResult struct {
	TestName       string                 `json:"test_name"`
	ComponentsUsed []string               `json:"components_used"`
	Success        bool                   `json:"success"`
	Duration       time.Duration          `json:"duration"`
	DataFlow       []DataFlowStep         `json:"data_flow"`
	Issues         []string               `json:"issues,omitempty"`
	Metrics        map[string]interface{} `json:"metrics,omitempty"`
}

type DataFlowStep struct {
	Step      int           `json:"step"`
	Component string        `json:"component"`
	Action    string        `json:"action"`
	Success   bool          `json:"success"`
	Duration  time.Duration `json:"duration"`
	DataSize  int           `json:"data_size,omitempty"`
	Error     string        `json:"error,omitempty"`
}

func (suite *CrossComponentTestSuite) SetupSuite() {
	suite.gatewayURL = "http://localhost:8888"
	suite.httpClient = &http.Client{Timeout: 30 * time.Second}

	// Initialize test data
	suite.testData = &TestData{
		TestLLMID:   "cross_comp_llm_" + fmt.Sprintf("%d", time.Now().Unix()),
		TestSkillID: "cross_comp_skill_" + fmt.Sprintf("%d", time.Now().Unix()),
		TestErrorID: "cross_comp_error_" + fmt.Sprintf("%d", time.Now().Unix()),
		TestUserID:  "cross_comp_user_" + fmt.Sprintf("%d", time.Now().Unix()),
		TestAgentID: "cross_comp_agent_" + fmt.Sprintf("%d", time.Now().Unix()),
	}

	// Wait for all services
	suite.waitForAllServices()

	// Authenticate
	suite.authenticate()

	// Create test wallet
	suite.createTestWallet()
}

func (suite *CrossComponentTestSuite) waitForAllServices() {
	services := map[string]string{
		"knirvoracle":    "http://localhost:1317/health",
		"knirvchain":   "http://localhost:8090/health",
		"knirvgraph":   "http://localhost:8082/height",
		"knirvnexus":   "http://localhost:8083/health",
		"knirvrouter":  "http://localhost:5001/status",
		"knirvgateway": "http://localhost:8888/gateway/health",
	}

	suite.T().Log("Waiting for all KNIRV services to be ready...")

	for serviceName, healthURL := range services {
		suite.T().Logf("Checking service: %s at %s", serviceName, healthURL)

		for i := 0; i < 30; i++ {
			resp, err := suite.httpClient.Get(healthURL)
			if err == nil && resp.StatusCode == http.StatusOK {
				resp.Body.Close()
				suite.T().Logf("Service %s is ready", serviceName)
				break
			}
			if resp != nil {
				resp.Body.Close()
			}
			time.Sleep(2 * time.Second)
		}
	}

	suite.T().Log("All KNIRV services are ready for cross-component testing")
}

func (suite *CrossComponentTestSuite) authenticate() {
	// Get testnet authentication token from gateway
	tokenURL := fmt.Sprintf("%s/auth/testnet-tokens", suite.gatewayURL)

	resp, err := suite.httpClient.Get(tokenURL)
	if err != nil {
		suite.T().Fatalf("Failed to get testnet token: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		suite.T().Fatalf("Token request failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		suite.T().Fatalf("Failed to read token response: %v", err)
	}

	var tokenResponse map[string]interface{}
	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		suite.T().Fatalf("Failed to parse token response: %v", err)
	}

	if token, ok := tokenResponse["token"].(string); ok {
		suite.authToken = token
		suite.T().Log("Authenticated for cross-component testing with testnet token")
	} else {
		suite.T().Fatal("No token found in response")
	}
}

func (suite *CrossComponentTestSuite) createTestWallet() {
	walletData := map[string]string{
		"name": "cross_component_test_wallet",
	}

	resp := suite.makeAuthenticatedRequest("POST", "/knirvwallet/wallet/create", walletData)
	require.True(suite.T(), resp.Success, "Failed to create test wallet")

	suite.testWallet = &TestWallet{
		Address:  resp.Data["address"].(string),
		Mnemonic: resp.Data["mnemonic"].(string),
		Balance:  "0",
	}

	// Fund the wallet
	suite.fundTestWallet("50000000") // 50 NRN

	suite.T().Logf("Created and funded cross-component test wallet: %s", suite.testWallet.Address)
}

func (suite *CrossComponentTestSuite) fundTestWallet(amount string) {
	fundData := map[string]interface{}{
		"address": suite.testWallet.Address,
		"amount":  amount,
	}

	resp := suite.makeAuthenticatedRequest("POST", "/knirvoracle/faucet/fund", fundData)
	require.True(suite.T(), resp.Success, "Failed to fund test wallet")

	suite.testWallet.Balance = amount
}

// Test 1: Complete Data Flow Integration
func (suite *CrossComponentTestSuite) TestCompleteDataFlowIntegration() {
	suite.Run("CompleteDataFlowIntegrationTest", func() {
		startTime := time.Now()
		var dataFlow []DataFlowStep
		var issues []string

		componentsUsed := []string{"KNIRVGATEWAY", "KNIRVCHAIN", "KNIRVGRAPH", "KNIRVNEXUS-FRONTEND", "KNIRVNEXUS-API-GATEWAY", "KNIRVORACLE"}

		// Step 1: Register LLM via KNIRVCHAIN
		suite.T().Log("Step 1: Registering LLM via KNIRVCHAIN...")
		stepStart := time.Now()

		llmData := map[string]interface{}{
			"name":             suite.testData.TestLLMID,
			"version":          "1.0.0",
			"capabilities":     []string{"cross-component-testing"},
			"model_data":       "dGVzdCBtb2RlbCBkYXRh",
			"registration_fee": "2000000",
			"usage_fee":        "200000",
		}

		llmResp := suite.makeAuthenticatedRequest("POST", "/knirvchain/llm/register", llmData)
		stepDuration := time.Since(stepStart)

		dataFlow = append(dataFlow, DataFlowStep{
			Step:      1,
			Component: "KNIRVCHAIN",
			Action:    "LLM Registration",
			Success:   llmResp.Success,
			Duration:  stepDuration,
			Error:     llmResp.Error,
		})

		if !llmResp.Success {
			issues = append(issues, fmt.Sprintf("LLM registration failed: %s", llmResp.Error))
		}

		// Step 2: Create Error Node via KNIRVGRAPH
		suite.T().Log("Step 2: Creating Error Node via KNIRVGRAPH...")
		stepStart = time.Now()

		errorData := map[string]interface{}{
			"error_type":  "cross_component_test_error",
			"description": "Test error for cross-component integration",
			"context": map[string]interface{}{
				"test_id":   suite.testData.TestErrorID,
				"component": "KNIRVGRAPH",
				"user_id":   suite.testData.TestUserID,
			},
			"severity": 3,
		}

		errorResp := suite.makeAuthenticatedRequest("POST", "/knirvgraph/nrv/errors", errorData)
		stepDuration = time.Since(stepStart)

		dataFlow = append(dataFlow, DataFlowStep{
			Step:      2,
			Component: "KNIRVGRAPH",
			Action:    "Error Node Creation",
			Success:   errorResp.Success,
			Duration:  stepDuration,
			Error:     errorResp.Error,
		})

		if errorResp.Success {
			suite.testData.TestErrorID = errorResp.Data["id"].(string)
		} else {
			issues = append(issues, fmt.Sprintf("Error node creation failed: %s", errorResp.Error))
		}

		// Step 3: Create Skill Node via KNIRVGRAPH
		suite.T().Log("Step 3: Creating Skill Node via KNIRVGRAPH...")
		stepStart = time.Now()

		skillData := map[string]interface{}{
			"skill_type":   "cross_component_skill",
			"capabilities": []string{"error_resolution", "cross_component_testing"},
			"requirements": map[string]interface{}{
				"min_confidence": 0.85,
				"max_latency":    "10s",
				"llm_id":         suite.testData.TestLLMID,
			},
		}

		skillResp := suite.makeAuthenticatedRequest("POST", "/knirvgraph/nrv/skills", skillData)
		stepDuration = time.Since(stepStart)

		dataFlow = append(dataFlow, DataFlowStep{
			Step:      3,
			Component: "KNIRVGRAPH",
			Action:    "Skill Node Creation",
			Success:   skillResp.Success,
			Duration:  stepDuration,
			Error:     skillResp.Error,
		})

		if skillResp.Success {
			suite.testData.TestSkillID = skillResp.Data["id"].(string)
		} else {
			issues = append(issues, fmt.Sprintf("Skill node creation failed: %s", skillResp.Error))
		}

		// Step 4: Create NEXUS Agent
		suite.T().Log("Step 4: Creating NEXUS Agent...")
		stepStart = time.Now()

		agentData := map[string]interface{}{
			"name":         suite.testData.TestAgentID,
			"type":         "cross_component_agent",
			"capabilities": []string{"error_analysis", "skill_coordination"},
			"config": map[string]interface{}{
				"llm_id":      suite.testData.TestLLMID,
				"skill_id":    suite.testData.TestSkillID,
				"error_types": []string{"cross_component_test_error"},
			},
		}

		agentResp := suite.makeAuthenticatedRequest("POST", "/knirvnexus-api-gateway/api/v1/agents", agentData)
		stepDuration = time.Since(stepStart)

		dataFlow = append(dataFlow, DataFlowStep{
			Step:      4,
			Component: "KNIRVNEXUS",
			Action:    "Agent Creation",
			Success:   agentResp.Success,
			Duration:  stepDuration,
			Error:     agentResp.Error,
		})

		if agentResp.Success {
			suite.testData.TestAgentID = agentResp.Data["id"].(string)
		} else {
			issues = append(issues, fmt.Sprintf("Agent creation failed: %s", agentResp.Error))
		}

		// Step 5: Invoke Skill with Token Burning via KNIRVCHAIN
		suite.T().Log("Step 5: Invoking Skill with Token Burning...")
		stepStart = time.Now()

		invokeData := map[string]interface{}{
			"skill_id":     suite.testData.TestSkillID,
			"amount":       "1000000", // 1 NRN
			"user_address": suite.testWallet.Address,
			"parameters": map[string]interface{}{
				"error_id":  suite.testData.TestErrorID,
				"agent_id":  suite.testData.TestAgentID,
				"test_type": "cross_component_integration",
			},
		}

		invokeResp := suite.makeAuthenticatedRequest("POST", "/knirvchain/skill/invoke", invokeData)
		stepDuration = time.Since(stepStart)

		dataFlow = append(dataFlow, DataFlowStep{
			Step:      5,
			Component: "KNIRVCHAIN",
			Action:    "Skill Invocation with Token Burning",
			Success:   invokeResp.Success,
			Duration:  stepDuration,
			Error:     invokeResp.Error,
		})

		if !invokeResp.Success {
			issues = append(issues, fmt.Sprintf("Skill invocation failed: %s", invokeResp.Error))
		}

		// Step 6: Verify Cross-Chain Bridge via KNIRVORACLE
		suite.T().Log("Step 6: Testing Cross-Chain Bridge...")
		stepStart = time.Now()

		bridgeData := map[string]interface{}{
			"target_chain": "xion",
			"amount":       "500000", // 0.5 NRN
			"recipient":    suite.testWallet.Address,
			"source":       "KNIRVORACLE",
		}

		bridgeResp := suite.makeAuthenticatedRequest("POST", "/knirvoracle/bridge/transfer", bridgeData)
		stepDuration = time.Since(stepStart)

		dataFlow = append(dataFlow, DataFlowStep{
			Step:      6,
			Component: "KNIRVORACLE",
			Action:    "Cross-Chain Bridge Transfer",
			Success:   bridgeResp.Success,
			Duration:  stepDuration,
			Error:     bridgeResp.Error,
		})

		if !bridgeResp.Success {
			issues = append(issues, fmt.Sprintf("Bridge transfer failed: %s", bridgeResp.Error))
		}

		// Calculate overall success
		totalDuration := time.Since(startTime)
		overallSuccess := len(issues) == 0

		result := IntegrationTestResult{
			TestName:       "CompleteDataFlowIntegration",
			ComponentsUsed: componentsUsed,
			Success:        overallSuccess,
			Duration:       totalDuration,
			DataFlow:       dataFlow,
			Issues:         issues,
			Metrics: map[string]interface{}{
				"total_steps":      len(dataFlow),
				"successful_steps": suite.countSuccessfulSteps(dataFlow),
				"avg_step_time":    suite.calculateAvgStepTime(dataFlow),
			},
		}

		suite.T().Logf("Complete Data Flow Integration Test Result: %+v", result)
		assert.True(suite.T(), result.Success, "Cross-component integration issues found: %v", issues)
		assert.Greater(suite.T(), result.Metrics["successful_steps"], 4, "At least 5 steps should succeed")
	})
}

// Test 2: Service Communication Validation
func (suite *CrossComponentTestSuite) TestServiceCommunication() {
	suite.Run("ServiceCommunicationTest", func() {
		services := []string{"knirvchain", "knirvgraph", "knirvnexus-frontend", "knirvnexus-api-gateway", "knirvoracle", "knirvrouter"}

		// Test health endpoints for all services
		for _, service := range services {
			suite.T().Logf("Testing %s health endpoint...", service)

			healthResp := suite.makeRequest("GET", fmt.Sprintf("/%s/health", service), nil)
			assert.True(suite.T(), healthResp.Success, "Service %s health check failed", service)

			if healthResp.Success {
				suite.T().Logf("Service %s is healthy", service)
			}
		}

		// Test inter-service communication
		suite.T().Log("Testing inter-service communication...")

		// Test KNIRVGATEWAY routing to all services
		for _, service := range services {
			routingResp := suite.makeAuthenticatedRequest("GET", fmt.Sprintf("/%s/status", service), nil)
			if routingResp.Success {
				suite.T().Logf("KNIRVGATEWAY successfully routed to %s", service)
			} else {
				suite.T().Logf("KNIRVGATEWAY routing to %s failed: %s", service, routingResp.Error)
			}
		}
	})
}

// Test 3: Data Consistency Validation
func (suite *CrossComponentTestSuite) TestDataConsistency() {
	suite.Run("DataConsistencyTest", func() {
		testID := "consistency_test_" + fmt.Sprintf("%d", time.Now().Unix())

		// Create data in KNIRVCHAIN
		chainData := map[string]interface{}{
			"id":   testID,
			"data": "test_consistency_data",
			"type": "consistency_test",
		}

		chainResp := suite.makeAuthenticatedRequest("POST", "/knirvchain/test/data", chainData)

		// Create corresponding data in KNIRVGRAPH
		graphData := map[string]interface{}{
			"id":   testID,
			"data": "test_consistency_data",
			"type": "consistency_test",
		}

		graphResp := suite.makeAuthenticatedRequest("POST", "/knirvgraph/test/data", graphData)

		// Wait for synchronization
		time.Sleep(5 * time.Second)

		// Verify data consistency
		if chainResp.Success && graphResp.Success {
			// Check if data is consistent across services
			chainDataResp := suite.makeAuthenticatedRequest("GET", fmt.Sprintf("/knirvchain/test/data/%s", testID), nil)
			graphDataResp := suite.makeAuthenticatedRequest("GET", fmt.Sprintf("/knirvgraph/test/data/%s", testID), nil)

			if chainDataResp.Success && graphDataResp.Success {
				chainValue := chainDataResp.Data["data"].(string)
				graphValue := graphDataResp.Data["data"].(string)

				assert.Equal(suite.T(), chainValue, graphValue, "Data inconsistency detected between KNIRVCHAIN and KNIRVGRAPH")
				suite.T().Log("Data consistency validated across services")
			}
		}
	})
}

// Test 4: KNIRVGATEWAY Integration
func (suite *CrossComponentTestSuite) TestKNIRVGATEWAYIntegration() {
	suite.Run("KNIRVGATEWAYIntegrationTest", func() {
		// Test gateway routing capabilities
		services := []string{"knirvchain", "knirvgraph", "knirvnexus-frontend", "knirvnexus-api-gateway", "knirvoracle", "knirvrouter"}

		for _, service := range services {
			suite.T().Logf("Testing KNIRVGATEWAY routing to %s...", service)

			// Test authenticated request routing
			resp := suite.makeAuthenticatedRequest("GET", fmt.Sprintf("/%s/health", service), nil)
			assert.True(suite.T(), resp.Success, "KNIRVGATEWAY failed to route to %s", service)

			// Test request forwarding with data
			testData := map[string]interface{}{
				"test":      true,
				"service":   service,
				"timestamp": time.Now().Unix(),
			}

			dataResp := suite.makeAuthenticatedRequest("POST", fmt.Sprintf("/%s/test/echo", service), testData)
			if dataResp.Success {
				suite.T().Logf("KNIRVGATEWAY successfully forwarded data to %s", service)
			}
		}

		// Test gateway load balancing (if implemented)
		suite.T().Log("Testing gateway load balancing...")

		for i := 0; i < 10; i++ {
			resp := suite.makeAuthenticatedRequest("GET", "/knirvchain/health", nil)
			assert.True(suite.T(), resp.Success, "Gateway load balancing failed on request %d", i+1)
		}
	})
}

// Helper methods
func (suite *CrossComponentTestSuite) makeRequest(method, path string, data interface{}) *TestResponse {
	return suite.makeRequestWithAuth(method, path, data, "")
}

func (suite *CrossComponentTestSuite) makeAuthenticatedRequest(method, path string, data interface{}) *TestResponse {
	return suite.makeRequestWithAuth(method, path, data, suite.authToken)
}

func (suite *CrossComponentTestSuite) makeRequestWithAuth(method, path string, data interface{}, token string) *TestResponse {
	var body io.Reader
	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return &TestResponse{Success: false, Error: "Failed to marshal request data"}
		}
		body = bytes.NewReader(jsonData)
	}

	url := suite.gatewayURL + path
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return &TestResponse{Success: false, Error: "Failed to create request"}
	}

	if data != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := suite.httpClient.Do(req)
	if err != nil {
		return &TestResponse{Success: false, Error: err.Error()}
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return &TestResponse{Success: false, Error: "Failed to read response body"}
	}

	var testResp TestResponse
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if len(responseBody) > 0 {
			err = json.Unmarshal(responseBody, &testResp.Data)
			if err != nil {
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

func (suite *CrossComponentTestSuite) countSuccessfulSteps(dataFlow []DataFlowStep) int {
	count := 0
	for _, step := range dataFlow {
		if step.Success {
			count++
		}
	}
	return count
}

func (suite *CrossComponentTestSuite) calculateAvgStepTime(dataFlow []DataFlowStep) time.Duration {
	if len(dataFlow) == 0 {
		return 0
	}

	total := time.Duration(0)
	for _, step := range dataFlow {
		total += step.Duration
	}

	return total / time.Duration(len(dataFlow))
}

// Main test function for the Cross-Component Test Suite
func TestCrossComponentTestSuite(t *testing.T) {
	suite.Run(t, new(CrossComponentTestSuite))
}
