package integration_tests

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type CrossComponentValidator struct {
	suite *IntegrationTestSuite
}

func NewCrossComponentValidator(suite *IntegrationTestSuite) *CrossComponentValidator {
	return &CrossComponentValidator{
		suite: suite,
	}
}

// Test KNIRVCHAIN <-> KNIRVGRAPH Integration
func (v *CrossComponentValidator) TestKNIRVCHAINGraphIntegration(t *testing.T) {
	// Test 1: Register LLM on KNIRVCHAIN and verify it appears in KNIRVGRAPH
	t.Run("LLMRegistrationPropagation", func(t *testing.T) {
		// Register LLM on KNIRVCHAIN
		llmData := map[string]interface{}{
			"name":             "CrossTestLLM",
			"version":          "1.0.0",
			"capabilities":     []string{"cross-component-test"},
			"model_data":       "Y3Jvc3MgdGVzdCBkYXRh", // base64 encoded "cross test data"
			"registration_fee": "2000000",
			"usage_fee":        "200000",
		}

		resp, err := v.suite.makeRequest("POST", v.suite.knirvchainURL+"/llm/register", llmData)
		require.NoError(t, err)

		var chainResult map[string]interface{}
		err = json.Unmarshal(resp, &chainResult)
		require.NoError(t, err)
		require.True(t, chainResult["success"].(bool))

		txHash := chainResult["tx_hash"].(string)
		t.Logf("LLM registered on KNIRVCHAIN with tx_hash: %s", txHash)

		// Wait for propagation
		time.Sleep(5 * time.Second)

		// Verify LLM appears in KNIRVGRAPH
		resp, err = v.suite.makeRequest("GET", v.suite.knirvgraphURL+"/llm/"+txHash, nil)
		require.NoError(t, err)

		var graphResult map[string]interface{}
		err = json.Unmarshal(resp, &graphResult)
		require.NoError(t, err)

		assert.Equal(t, "CrossTestLLM", graphResult["name"])
		assert.Equal(t, txHash, graphResult["chain_tx_hash"])
		t.Logf("LLM successfully propagated to KNIRVGRAPH: %+v", graphResult)
	})

	// Test 2: Skill invocation burns NRN on chain and creates skill node in graph
	t.Run("SkillInvocationPropagation", func(t *testing.T) {
		// Create skill node in KNIRVGRAPH first
		skillData := map[string]interface{}{
			"skill_type":   "cross_test_skill",
			"capabilities": []string{"cross-component-validation"},
			"requirements": map[string]interface{}{
				"min_confidence": 0.9,
				"max_latency":    "3s",
			},
		}

		resp, err := v.suite.makeRequest("POST", v.suite.knirvgraphURL+"/nrv/skills", skillData)
		require.NoError(t, err)

		var skillNode map[string]interface{}
		err = json.Unmarshal(resp, &skillNode)
		require.NoError(t, err)

		skillID := skillNode["id"].(string)
		t.Logf("Skill node created in KNIRVGRAPH with ID: %s", skillID)

		// Invoke skill on KNIRVCHAIN
		invokeData := map[string]interface{}{
			"skill_id":     skillID,
			"amount":       "1000000",
			"user_address": v.suite.testWallet.Address,
		}

		resp, err = v.suite.makeRequest("POST", v.suite.knirvchainURL+"/skill/invoke", invokeData)
		require.NoError(t, err)

		var invokeResult map[string]interface{}
		err = json.Unmarshal(resp, &invokeResult)
		require.NoError(t, err)

		assert.True(t, invokeResult["success"].(bool))
		txHash := invokeResult["tx_hash"].(string)
		t.Logf("Skill invoked on KNIRVCHAIN with tx_hash: %s", txHash)

		// Wait for propagation
		time.Sleep(5 * time.Second)

		// Verify skill invocation is recorded in KNIRVGRAPH
		resp, err = v.suite.makeRequest("GET", v.suite.knirvgraphURL+"/nrv/skills/"+skillID+"/invocations", nil)
		require.NoError(t, err)

		var invocations []map[string]interface{}
		err = json.Unmarshal(resp, &invocations)
		require.NoError(t, err)

		assert.GreaterOrEqual(t, len(invocations), 1)

		// Find our invocation
		found := false
		for _, inv := range invocations {
			if inv["chain_tx_hash"] == txHash {
				found = true
				assert.Equal(t, v.suite.testWallet.Address, inv["user_address"])
				break
			}
		}
		assert.True(t, found, "Skill invocation not found in KNIRVGRAPH")
	})
}

// Test KNIRVNEXUS <-> KNIRVROOT Integration
func (v *CrossComponentValidator) TestKNIRVNEXUSRootIntegration(t *testing.T) {
	// Test 1: Agent creation in NEXUS triggers blockchain transaction in ROOT
	t.Run("AgentCreationBlockchainRecord", func(t *testing.T) {
		// Create agent in KNIRVNEXUS
		agentData := map[string]interface{}{
			"name":         "CrossTestAgent",
			"description":  "Agent for cross-component testing",
			"type":         "go_plugin",
			"capabilities": []string{"blockchain_integration"},
		}

		resp, err := v.suite.makeRequest("POST", v.suite.knirvnexusURL+"/api/v1/agents", agentData)
		require.NoError(t, err)

		var agent map[string]interface{}
		err = json.Unmarshal(resp, &agent)
		require.NoError(t, err)

		agentID := agent["id"].(string)
		t.Logf("Agent created in KNIRVNEXUS with ID: %s", agentID)

		// Wait for blockchain propagation
		time.Sleep(5 * time.Second)

		// Verify agent creation is recorded in KNIRVROOT blockchain
		resp, err = v.suite.makeRequest("GET", v.suite.knirvRootURL+"/blockchain/transactions?type=agent_creation&agent_id="+agentID, nil)
		require.NoError(t, err)

		var transactions []map[string]interface{}
		err = json.Unmarshal(resp, &transactions)
		require.NoError(t, err)

		assert.GreaterOrEqual(t, len(transactions), 1)

		// Verify transaction details
		tx := transactions[0]
		assert.Equal(t, "agent_creation", tx["type"])
		assert.Equal(t, agentID, tx["agent_id"])
		t.Logf("Agent creation recorded in blockchain: %+v", tx)
	})

	// Test 2: Agent execution consumes NRN tokens
	t.Run("AgentExecutionTokenConsumption", func(t *testing.T) {
		// Get initial wallet balance
		resp, err := v.suite.makeRequest("GET", v.suite.knirvRootURL+"/wallet/"+v.suite.testWallet.Address+"/balance", nil)
		require.NoError(t, err)

		var initialBalance map[string]interface{}
		err = json.Unmarshal(resp, &initialBalance)
		require.NoError(t, err)

		initialAmount := initialBalance["balance"].(string)
		t.Logf("Initial wallet balance: %s", initialAmount)

		// Get an agent to execute
		resp, err = v.suite.makeRequest("GET", v.suite.knirvnexusURL+"/api/v1/agents", nil)
		require.NoError(t, err)

		var agents []map[string]interface{}
		err = json.Unmarshal(resp, &agents)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(agents), 1)

		agentID := agents[0]["id"].(string)

		// Execute agent with token consumption
		execData := map[string]interface{}{
			"input":          "Test execution with token consumption",
			"wallet_address": v.suite.testWallet.Address,
			"token_amount":   "500000",
		}

		resp, err = v.suite.makeRequest("POST", v.suite.knirvnexusURL+"/api/v1/agents/"+agentID+"/execute", execData)
		require.NoError(t, err)

		var execResult map[string]interface{}
		err = json.Unmarshal(resp, &execResult)
		require.NoError(t, err)

		executionID := execResult["execution_id"].(string)
		t.Logf("Agent execution started with ID: %s", executionID)

		// Wait for execution and token consumption
		time.Sleep(10 * time.Second)

		// Check final wallet balance
		resp, err = v.suite.makeRequest("GET", v.suite.knirvRootURL+"/wallet/"+v.suite.testWallet.Address+"/balance", nil)
		require.NoError(t, err)

		var finalBalance map[string]interface{}
		err = json.Unmarshal(resp, &finalBalance)
		require.NoError(t, err)

		finalAmount := finalBalance["balance"].(string)
		t.Logf("Final wallet balance: %s", finalAmount)

		// Verify tokens were consumed (final balance should be less than initial)
		// Note: This is a simplified check - in reality we'd parse the amounts properly
		assert.NotEqual(t, initialAmount, finalAmount, "Wallet balance should have changed after agent execution")
	})
}

// Test KNIRVROUTER <-> KNIRVROOT Integration
func (v *CrossComponentValidator) TestKNIRVROUTERRootIntegration(t *testing.T) {
	// Test 1: Connectivity proof generates NRN rewards
	t.Run("ConnectivityProofRewards", func(t *testing.T) {
		// Get initial balance
		resp, err := v.suite.makeRequest("GET", v.suite.knirvRootURL+"/wallet/"+v.suite.testWallet.Address+"/balance", nil)
		require.NoError(t, err)

		var initialBalance map[string]interface{}
		err = json.Unmarshal(resp, &initialBalance)
		require.NoError(t, err)

		// Generate connectivity proof
		proofData := map[string]interface{}{
			"router_id":      "test_router_001",
			"target_peers":   []string{"peer1", "peer2", "peer3"},
			"proof_type":     "comprehensive",
			"reward_address": v.suite.testWallet.Address,
		}

		resp, err = v.suite.makeRequest("POST", v.suite.knirvroterURL+"/connectivity/proof", proofData)
		require.NoError(t, err)

		var proof map[string]interface{}
		err = json.Unmarshal(resp, &proof)
		require.NoError(t, err)

		proofID := proof["proof_id"].(string)
		t.Logf("Connectivity proof generated: %s", proofID)

		// Wait for proof validation and reward processing
		time.Sleep(15 * time.Second)

		// Check if rewards were distributed
		resp, err = v.suite.makeRequest("GET", v.suite.knirvRootURL+"/wallet/"+v.suite.testWallet.Address+"/balance", nil)
		require.NoError(t, err)

		var finalBalance map[string]interface{}
		err = json.Unmarshal(resp, &finalBalance)
		require.NoError(t, err)

		t.Logf("Balance after connectivity proof - Initial: %s, Final: %s",
			initialBalance["balance"], finalBalance["balance"])

		// Verify reward transaction exists
		resp, err = v.suite.makeRequest("GET", v.suite.knirvRootURL+"/blockchain/transactions?type=connectivity_reward&proof_id="+proofID, nil)
		require.NoError(t, err)

		var rewardTxs []map[string]interface{}
		err = json.Unmarshal(resp, &rewardTxs)
		require.NoError(t, err)

		assert.GreaterOrEqual(t, len(rewardTxs), 1)
		t.Logf("Connectivity reward transaction recorded: %+v", rewardTxs[0])
	})
}

func TestCrossComponentValidation(t *testing.T) {
	suite := NewIntegrationTestSuite()
	suite.SetupTest(t)

	validator := NewCrossComponentValidator(suite)

	t.Run("KNIRVCHAINGraphIntegration", validator.TestKNIRVCHAINGraphIntegration)
	t.Run("KNIRVNEXUSRootIntegration", validator.TestKNIRVNEXUSRootIntegration)
	t.Run("KNIRVROUTERRootIntegration", validator.TestKNIRVROUTERRootIntegration)
}
