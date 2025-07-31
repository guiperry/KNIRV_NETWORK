package integration_tests

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type E2EWorkflowTester struct {
	suite *IntegrationTestSuite
}

func NewE2EWorkflowTester(suite *IntegrationTestSuite) *E2EWorkflowTester {
	return &E2EWorkflowTester{
		suite: suite,
	}
}

// Test Complete Developer Workflow
func (e2e *E2EWorkflowTester) TestDeveloperWorkflow(t *testing.T) {
	t.Run("CompleteCodeFixingWorkflow", func(t *testing.T) {
		// Step 1: Developer encounters a coding error
		t.Log("Step 1: Creating error scenario...")

		errorData := map[string]interface{}{
			"error_type":  "syntax_error",
			"description": "Missing closing bracket in function definition",
			"context": map[string]interface{}{
				"language": "javascript",
				"line":     15,
				"file":     "main.js",
				"code":     "function calculateSum(a, b { return a + b; }",
			},
			"severity": 4,
		}

		resp, err := e2e.suite.makeRequest("POST", e2e.suite.knirvgraphURL+"/nrv/errors", errorData)
		require.NoError(t, err)

		var errorNode map[string]interface{}
		err = json.Unmarshal(resp, &errorNode)
		require.NoError(t, err)

		errorID := errorNode["id"].(string)
		t.Logf("Error node created: %s", errorID)

		// Step 2: System identifies appropriate skill for fixing the error
		t.Log("Step 2: Creating skill to fix the error...")

		skillData := map[string]interface{}{
			"skill_type":   "syntax_fixer",
			"capabilities": []string{"javascript", "bracket_completion", "syntax_repair"},
			"requirements": map[string]interface{}{
				"min_confidence": 0.85,
				"max_latency":    "3s",
			},
			"error_types": []string{"syntax_error"},
		}

		resp, err = e2e.suite.makeRequest("POST", e2e.suite.knirvgraphURL+"/nrv/skills", skillData)
		require.NoError(t, err)

		var skillNode map[string]interface{}
		err = json.Unmarshal(resp, &skillNode)
		require.NoError(t, err)

		skillID := skillNode["id"].(string)
		t.Logf("Skill node created: %s", skillID)

		// Step 3: Developer invokes skill using NRN tokens
		t.Log("Step 3: Invoking skill with NRN payment...")

		invokeData := map[string]interface{}{
			"skill_id":     skillID,
			"amount":       "750000",
			"user_address": e2e.suite.testWallet.Address,
			"error_id":     errorID,
		}

		resp, err = e2e.suite.makeRequest("POST", e2e.suite.knirvchainURL+"/skill/invoke", invokeData)
		require.NoError(t, err)

		var invokeResult map[string]interface{}
		err = json.Unmarshal(resp, &invokeResult)
		require.NoError(t, err)

		assert.True(t, invokeResult["success"].(bool))
		txHash := invokeResult["tx_hash"].(string)
		t.Logf("Skill invoked successfully: %s", txHash)

		// Step 4: Wait for skill execution and verify results
		t.Log("Step 4: Waiting for skill execution...")
		time.Sleep(10 * time.Second)

		// Check skill execution results
		resp, err = e2e.suite.makeRequest("GET", e2e.suite.knirvgraphURL+"/nrv/skills/"+skillID+"/executions/"+txHash, nil)
		require.NoError(t, err)

		var execution map[string]interface{}
		err = json.Unmarshal(resp, &execution)
		require.NoError(t, err)

		assert.Equal(t, "completed", execution["status"])
		assert.NotEmpty(t, execution["result"])
		t.Logf("Skill execution completed: %+v", execution)

		// Step 5: Verify error is marked as resolved
		resp, err = e2e.suite.makeRequest("GET", e2e.suite.knirvgraphURL+"/nrv/errors/"+errorID, nil)
		require.NoError(t, err)

		var updatedError map[string]interface{}
		err = json.Unmarshal(resp, &updatedError)
		require.NoError(t, err)

		assert.Equal(t, "resolved", updatedError["status"])
		t.Log("Error successfully marked as resolved")
	})
}

// Test Complete Agent Development Workflow
func (e2e *E2EWorkflowTester) TestAgentDevelopmentWorkflow(t *testing.T) {
	t.Run("CompleteAgentLifecycle", func(t *testing.T) {
		// Step 1: Developer creates a new agent
		t.Log("Step 1: Creating new agent...")

		agentData := map[string]interface{}{
			"name":         "E2ETestAgent",
			"description":  "End-to-end test agent for workflow validation",
			"type":         "go_plugin",
			"capabilities": []string{"data_processing", "file_analysis"},
			"config": map[string]interface{}{
				"max_memory": "1GB",
				"timeout":    "60s",
				"env_vars": map[string]string{
					"DEBUG": "true",
				},
			},
		}

		resp, err := e2e.suite.makeRequest("POST", e2e.suite.knirvnexusURL+"/api/v1/agents", agentData)
		require.NoError(t, err)

		var agent map[string]interface{}
		err = json.Unmarshal(resp, &agent)
		require.NoError(t, err)

		agentID := agent["id"].(string)
		t.Logf("Agent created: %s", agentID)

		// Step 2: Register agent on blockchain
		t.Log("Step 2: Registering agent on blockchain...")

		registrationData := map[string]interface{}{
			"agent_id":     agentID,
			"owner":        e2e.suite.testWallet.Address,
			"capabilities": agentData["capabilities"],
			"fee":          "2000000",
		}

		resp, err = e2e.suite.makeRequest("POST", e2e.suite.knirvRootURL+"/agents/register", registrationData)
		require.NoError(t, err)

		var regResult map[string]interface{}
		err = json.Unmarshal(resp, &regResult)
		require.NoError(t, err)

		assert.NotEmpty(t, regResult["tx_hash"])
		t.Logf("Agent registered on blockchain: %s", regResult["tx_hash"])

		// Step 3: Deploy agent to execution environment
		t.Log("Step 3: Deploying agent...")

		deployData := map[string]interface{}{
			"agent_id":           agentID,
			"target_environment": "production",
			"resources": map[string]interface{}{
				"cpu":    "2",
				"memory": "1GB",
			},
		}

		resp, err = e2e.suite.makeRequest("POST", e2e.suite.knirvnexusURL+"/api/v1/agents/"+agentID+"/deploy", deployData)
		require.NoError(t, err)

		var deployResult map[string]interface{}
		err = json.Unmarshal(resp, &deployResult)
		require.NoError(t, err)

		assert.Equal(t, "deployed", deployResult["status"])
		t.Logf("Agent deployed successfully: %+v", deployResult)

		// Step 4: Execute agent with test data
		t.Log("Step 4: Executing agent...")

		execData := map[string]interface{}{
			"input": map[string]interface{}{
				"data":   "test data for processing",
				"format": "text",
			},
			"parameters": map[string]interface{}{
				"mode":          "analysis",
				"output_format": "json",
			},
			"wallet_address": e2e.suite.testWallet.Address,
			"token_amount":   "500000",
		}

		resp, err = e2e.suite.makeRequest("POST", e2e.suite.knirvnexusURL+"/api/v1/agents/"+agentID+"/execute", execData)
		require.NoError(t, err)

		var execResult map[string]interface{}
		err = json.Unmarshal(resp, &execResult)
		require.NoError(t, err)

		executionID := execResult["execution_id"].(string)
		t.Logf("Agent execution started: %s", executionID)

		// Step 5: Monitor execution progress
		t.Log("Step 5: Monitoring execution...")

		maxWait := 30 * time.Second
		checkInterval := 2 * time.Second
		startTime := time.Now()

		var finalStatus string
		for time.Since(startTime) < maxWait {
			resp, err = e2e.suite.makeRequest("GET", e2e.suite.knirvnexusURL+"/api/v1/agents/"+agentID+"/executions/"+executionID, nil)
			if err == nil {
				var status map[string]interface{}
				if json.Unmarshal(resp, &status) == nil {
					finalStatus = status["status"].(string)
					if finalStatus == "completed" || finalStatus == "failed" {
						t.Logf("Execution completed with status: %s", finalStatus)
						break
					}
				}
			}
			time.Sleep(checkInterval)
		}

		assert.Equal(t, "completed", finalStatus, "Agent execution should complete successfully")

		// Step 6: Verify blockchain transaction for execution payment
		t.Log("Step 6: Verifying blockchain transaction...")

		resp, err = e2e.suite.makeRequest("GET", e2e.suite.knirvRootURL+"/blockchain/transactions?type=agent_execution&execution_id="+executionID, nil)
		require.NoError(t, err)

		var transactions []map[string]interface{}
		err = json.Unmarshal(resp, &transactions)
		require.NoError(t, err)

		assert.GreaterOrEqual(t, len(transactions), 1, "Should have at least one transaction for agent execution")

		tx := transactions[0]
		assert.Equal(t, "agent_execution", tx["type"])
		assert.Equal(t, executionID, tx["execution_id"])
		t.Logf("Blockchain transaction verified: %+v", tx)
	})
}

// Test Complete Cross-Chain Bridge Workflow
func (e2e *E2EWorkflowTester) TestCrossChainBridgeWorkflow(t *testing.T) {
	t.Run("CompleteBridgeTransfer", func(t *testing.T) {
		// Step 1: Check initial balances
		t.Log("Step 1: Checking initial balances...")

		resp, err := e2e.suite.makeRequest("GET", e2e.suite.knirvRootURL+"/wallet/"+e2e.suite.testWallet.Address+"/balance", nil)
		require.NoError(t, err)

		var initialBalance map[string]interface{}
		err = json.Unmarshal(resp, &initialBalance)
		require.NoError(t, err)

		initialAmount := initialBalance["balance"].(string)
		t.Logf("Initial KNIRVROOT balance: %s", initialAmount)

		// Step 2: Initiate bridge transfer from KNIRVROOT to XION
		t.Log("Step 2: Initiating bridge transfer...")

		bridgeData := map[string]interface{}{
			"target_chain": "xion",
			"amount":       "2000000",
			"recipient":    e2e.suite.testWallet.Address,
			"bridge_fee":   "50000",
		}

		resp, err = e2e.suite.makeRequest("POST", e2e.suite.knirvchainURL+"/bridge/transfer", bridgeData)
		require.NoError(t, err)

		var bridgeResult map[string]interface{}
		err = json.Unmarshal(resp, &bridgeResult)
		require.NoError(t, err)

		assert.NotEmpty(t, bridgeResult["tx_hash"])
		assert.Equal(t, "pending", bridgeResult["status"])

		txHash := bridgeResult["tx_hash"].(string)
		t.Logf("Bridge transfer initiated: %s", txHash)

		// Step 3: Monitor bridge transfer progress
		t.Log("Step 3: Monitoring bridge transfer...")

		maxWait := 60 * time.Second
		checkInterval := 5 * time.Second
		startTime := time.Now()

		var finalStatus string
		for time.Since(startTime) < maxWait {
			resp, err = e2e.suite.makeRequest("GET", e2e.suite.knirvchainURL+"/bridge/status?tx_hash="+txHash, nil)
			if err == nil {
				var status map[string]interface{}
				if json.Unmarshal(resp, &status) == nil {
					finalStatus = status["status"].(string)
					t.Logf("Bridge status: %s", finalStatus)
					if finalStatus == "completed" || finalStatus == "failed" {
						break
					}
				}
			}
			time.Sleep(checkInterval)
		}

		assert.Equal(t, "completed", finalStatus, "Bridge transfer should complete successfully")

		// Step 4: Verify balances after bridge transfer
		t.Log("Step 4: Verifying final balances...")

		resp, err = e2e.suite.makeRequest("GET", e2e.suite.knirvRootURL+"/wallet/"+e2e.suite.testWallet.Address+"/balance", nil)
		require.NoError(t, err)

		var finalBalance map[string]interface{}
		err = json.Unmarshal(resp, &finalBalance)
		require.NoError(t, err)

		finalAmount := finalBalance["balance"].(string)
		t.Logf("Final KNIRVROOT balance: %s", finalAmount)

		// Balance should have decreased by the transfer amount plus fees
		assert.NotEqual(t, initialAmount, finalAmount, "Balance should have changed after bridge transfer")

		// Step 5: Verify bridge transaction is recorded on both chains
		t.Log("Step 5: Verifying cross-chain transaction records...")

		// Check KNIRVROOT record
		resp, err = e2e.suite.makeRequest("GET", e2e.suite.knirvRootURL+"/blockchain/transactions?type=bridge_out&tx_hash="+txHash, nil)
		require.NoError(t, err)

		var rootTxs []map[string]interface{}
		err = json.Unmarshal(resp, &rootTxs)
		require.NoError(t, err)

		assert.GreaterOrEqual(t, len(rootTxs), 1, "Should have bridge out transaction on KNIRVROOT")

		rootTx := rootTxs[0]
		assert.Equal(t, "bridge_out", rootTx["type"])
		assert.Equal(t, "xion", rootTx["target_chain"])
		t.Logf("KNIRVROOT bridge transaction verified: %+v", rootTx)
	})
}

// Test Complete P2P Network Workflow
func (e2e *E2EWorkflowTester) TestP2PNetworkWorkflow(t *testing.T) {
	t.Run("CompleteConnectivityWorkflow", func(t *testing.T) {
		// Step 1: Router joins network
		t.Log("Step 1: Router joining network...")

		joinData := map[string]interface{}{
			"router_id":      "e2e_test_router",
			"capabilities":   []string{"relay", "storage", "compute"},
			"stake_amount":   "5000000",
			"wallet_address": e2e.suite.testWallet.Address,
		}

		resp, err := e2e.suite.makeRequest("POST", e2e.suite.knirvroterURL+"/network/join", joinData)
		require.NoError(t, err)

		var joinResult map[string]interface{}
		err = json.Unmarshal(resp, &joinResult)
		require.NoError(t, err)

		assert.Equal(t, "joined", joinResult["status"])
		t.Logf("Router joined network: %+v", joinResult)

		// Step 2: Discover peers
		t.Log("Step 2: Discovering peers...")

		resp, err = e2e.suite.makeRequest("GET", e2e.suite.knirvroterURL+"/peers", nil)
		require.NoError(t, err)

		var peers []map[string]interface{}
		err = json.Unmarshal(resp, &peers)
		require.NoError(t, err)

		t.Logf("Discovered %d peers", len(peers))

		// Step 3: Generate connectivity proof
		t.Log("Step 3: Generating connectivity proof...")

		proofData := map[string]interface{}{
			"router_id":      "e2e_test_router",
			"target_peers":   []string{"peer1", "peer2", "peer3"},
			"proof_type":     "comprehensive",
			"reward_address": e2e.suite.testWallet.Address,
		}

		resp, err = e2e.suite.makeRequest("POST", e2e.suite.knirvroterURL+"/connectivity/proof", proofData)
		require.NoError(t, err)

		var proof map[string]interface{}
		err = json.Unmarshal(resp, &proof)
		require.NoError(t, err)

		proofID := proof["proof_id"].(string)
		t.Logf("Connectivity proof generated: %s", proofID)

		// Step 4: Wait for proof validation and rewards
		t.Log("Step 4: Waiting for proof validation...")
		time.Sleep(20 * time.Second)

		// Step 5: Verify rewards were distributed
		resp, err = e2e.suite.makeRequest("GET", e2e.suite.knirvRootURL+"/blockchain/transactions?type=connectivity_reward&proof_id="+proofID, nil)
		require.NoError(t, err)

		var rewardTxs []map[string]interface{}
		err = json.Unmarshal(resp, &rewardTxs)
		require.NoError(t, err)

		assert.GreaterOrEqual(t, len(rewardTxs), 1, "Should have connectivity reward transaction")

		rewardTx := rewardTxs[0]
		assert.Equal(t, "connectivity_reward", rewardTx["type"])
		assert.Equal(t, proofID, rewardTx["proof_id"])
		t.Logf("Connectivity reward verified: %+v", rewardTx)
	})
}

func TestE2EWorkflows(t *testing.T) {
	suite := NewIntegrationTestSuite()
	suite.SetupTest(t)

	tester := NewE2EWorkflowTester(suite)

	t.Run("DeveloperWorkflow", tester.TestDeveloperWorkflow)
	t.Run("AgentDevelopmentWorkflow", tester.TestAgentDevelopmentWorkflow)
	t.Run("CrossChainBridgeWorkflow", tester.TestCrossChainBridgeWorkflow)
	t.Run("P2PNetworkWorkflow", tester.TestP2PNetworkWorkflow)
}
