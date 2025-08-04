package integration_tests

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type IntegrationTestSuite struct {
	knirvchainURL  string
	knirvgraphURL  string
	knirvnexusURL  string
	knirvwalletURL string
	knirvshellURL  string
	knirvroterURL  string
	knirvRootURL   string
	xionRPC        string
	testWallet     *TestWallet
}

type TestWallet struct {
	Address  string `json:"address"`
	Mnemonic string `json:"mnemonic"`
	Balance  string `json:"balance"`
	Type     string `json:"type,omitempty"`
}

func NewIntegrationTestSuite() *IntegrationTestSuite {
	return &IntegrationTestSuite{
		knirvchainURL:  "http://localhost:8080",
		knirvgraphURL:  "http://localhost:8081",
		knirvnexusURL:  "http://localhost:8083", // KNIRVNEXUS API port
		knirvwalletURL: "http://localhost:8084", // KNIRVWALLET (not implemented yet)
		knirvshellURL:  "http://localhost:8085", // KNIRVSHELL (not implemented yet)
		knirvroterURL:  "http://localhost:8086", // KNIRVROUTER (mocked)
		knirvRootURL:   "http://localhost:8087", // KNIRVROOT
		xionRPC:        "https://rpc.xion-testnet-1.burnt.com:443",
	}
}

func (suite *IntegrationTestSuite) SetupTest(t *testing.T) {
	// Try to create test wallet (skip if KNIRVWALLET service not available)
	wallet, err := suite.createTestWallet()
	if err != nil {
		t.Logf("Warning: Could not create test wallet (KNIRVWALLET service may not be running): %v", err)
		suite.testWallet = nil
		return
	}
	suite.testWallet = wallet

	// Fund wallet with test tokens
	err = suite.fundTestWallet()
	if err != nil {
		t.Logf("Warning: Could not fund test wallet: %v", err)
		return
	}

	// Wait for funding to confirm
	time.Sleep(5 * time.Second)
}

func (suite *IntegrationTestSuite) TestFullWorkflow(t *testing.T) {
	suite.SetupTest(t)

	// Test 1: Register LLM on KNIRVCHAIN
	t.Run("RegisterLLM", func(t *testing.T) {
		llmData := map[string]interface{}{
			"name":             "TestLLM",
			"version":          "1.0.0",
			"capabilities":     []string{"text-generation", "code-completion"},
			"model_data":       "dGVzdCBtb2RlbCBkYXRh", // base64 encoded "test model data"
			"registration_fee": "1000000",
			"usage_fee":        "100000",
		}

		resp, err := suite.makeRequest("POST", suite.knirvchainURL+"/llm/register", llmData)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(resp, &result)
		require.NoError(t, err)

		assert.True(t, result["success"].(bool))
		assert.NotEmpty(t, result["tx_hash"])

		t.Logf("LLM registered with tx_hash: %s", result["tx_hash"])
	})

	// Test 2: Create Error Node in KNIRVGRAPH
	t.Run("CreateErrorNode", func(t *testing.T) {
		errorData := map[string]interface{}{
			"error_type":  "compilation_error",
			"description": "Missing semicolon in JavaScript code",
			"context": map[string]interface{}{
				"language": "javascript",
				"line":     42,
				"file":     "test.js",
			},
			"severity": 3,
		}

		resp, err := suite.makeRequest("POST", suite.knirvgraphURL+"/nrv/errors", errorData)
		require.NoError(t, err)

		var errorNode map[string]interface{}
		err = json.Unmarshal(resp, &errorNode)
		require.NoError(t, err)

		assert.NotEmpty(t, errorNode["id"])
		assert.Equal(t, "compilation_error", errorNode["error_type"])

		t.Logf("Error node created with ID: %s", errorNode["id"])
	})

	// Test 3: Create Skill Node in KNIRVGRAPH
	t.Run("CreateSkillNode", func(t *testing.T) {
		skillData := map[string]interface{}{
			"skill_type":   "code_fixer",
			"capabilities": []string{"javascript", "syntax_repair", "semicolon_insertion"},
			"requirements": map[string]interface{}{
				"min_confidence": 0.8,
				"max_latency":    "5s",
			},
		}

		resp, err := suite.makeRequest("POST", suite.knirvgraphURL+"/nrv/skills", skillData)
		require.NoError(t, err)

		var skillNode map[string]interface{}
		err = json.Unmarshal(resp, &skillNode)
		require.NoError(t, err)

		assert.NotEmpty(t, skillNode["id"])
		assert.Equal(t, "code_fixer", skillNode["skill_type"])

		t.Logf("Skill node created with ID: %s", skillNode["id"])
	})

	// Test 4: Test NRV Resolution
	t.Run("TestNRVResolution", func(t *testing.T) {
		// Create a vector for resolution testing
		vectorData := map[string]interface{}{
			"target_hash": "test_hash_123",
			"coordinates": []float64{1.0, 2.0, 3.0},
			"metadata": map[string]interface{}{
				"type": "test_vector",
			},
		}

		resp, err := suite.makeRequest("POST", suite.knirvgraphURL+"/nrv/vectors", vectorData)
		require.NoError(t, err)

		var vector map[string]interface{}
		err = json.Unmarshal(resp, &vector)
		require.NoError(t, err)

		// Test resolution
		resp, err = suite.makeRequest("GET", suite.knirvgraphURL+"/nrv/resolve/test_hash_123", nil)
		require.NoError(t, err)

		var vectors []map[string]interface{}
		err = json.Unmarshal(resp, &vectors)
		require.NoError(t, err)

		assert.Len(t, vectors, 1)
		assert.Equal(t, "test_hash_123", vectors[0]["target_hash"])

		t.Logf("Successfully resolved vector: %+v", vectors[0])
	})

	// Test 5: Test Cross-Chain Token Bridge
	t.Run("TestTokenBridge", func(t *testing.T) {
		// Skip if wallet is not available
		if suite.testWallet == nil {
			t.Skip("KNIRVWALLET service not available - skipping token bridge test")
			return
		}

		// Test bridge transfer from KNIRVROOT to XION
		bridgeData := map[string]interface{}{
			"target_chain": "xion",
			"amount":       "1000000",
			"recipient":    suite.testWallet.Address,
		}

		resp, err := suite.makeRequest("POST", suite.knirvchainURL+"/bridge/transfer", bridgeData)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(resp, &result)
		require.NoError(t, err)

		assert.NotEmpty(t, result["tx_hash"])
		assert.Equal(t, "pending", result["status"])

		txHash := result["tx_hash"].(string)
		t.Logf("Bridge transfer initiated with tx_hash: %s", txHash)

		// Wait for bridge processing
		time.Sleep(10 * time.Second)

		// Check bridge status
		resp, err = suite.makeRequest("GET", suite.knirvchainURL+"/bridge/status?tx_hash="+txHash, nil)
		require.NoError(t, err)

		var status map[string]interface{}
		err = json.Unmarshal(resp, &status)
		require.NoError(t, err)

		t.Logf("Bridge status: %+v", status)
	})

	// Test 6: Test Skill Invocation with NRN Burning
	t.Run("TestSkillInvocation", func(t *testing.T) {
		// Skip if wallet is not available
		if suite.testWallet == nil {
			t.Skip("KNIRVWALLET service not available - skipping skill invocation test")
			return
		}

		skillData := map[string]interface{}{
			"skill_id":     "test_skill_123",
			"amount":       "500000",
			"user_address": suite.testWallet.Address,
		}

		resp, err := suite.makeRequest("POST", suite.knirvchainURL+"/skill/invoke", skillData)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(resp, &result)
		require.NoError(t, err)

		assert.True(t, result["success"].(bool))
		assert.NotEmpty(t, result["tx_hash"])

		t.Logf("Skill invoked with tx_hash: %s", result["tx_hash"])
	})
}

func (suite *IntegrationTestSuite) createTestWallet() (*TestWallet, error) {
	// Create wallet using KNIRVWALLET service
	resp, err := suite.makeRequest("POST", suite.knirvwalletURL+"/wallet/create", map[string]interface{}{
		"name": "integration_test_wallet",
	})
	if err != nil {
		return nil, err
	}

	var wallet TestWallet
	err = json.Unmarshal(resp, &wallet)
	return &wallet, err
}

func (suite *IntegrationTestSuite) fundTestWallet() error {
	// Fund wallet with test NRN tokens
	fundData := map[string]interface{}{
		"address": suite.testWallet.Address,
		"amount":  "10000000", // 10 NRN
	}

	_, err := suite.makeRequest("POST", suite.knirvchainURL+"/faucet/fund", fundData)
	return err
}

func (suite *IntegrationTestSuite) makeRequest(method, url string, data interface{}) ([]byte, error) {
	var body io.Reader
	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}
		body = strings.NewReader(string(jsonData))
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	if data != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP error %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// Test KNIRVNEXUS Agent Management
func (suite *IntegrationTestSuite) TestKNIRVNEXUSAgentManagement(t *testing.T) {
	// Test 1: Create Agent
	t.Run("CreateAgent", func(t *testing.T) {
		agentData := map[string]interface{}{
			"name":         "TestAgent",
			"description":  "Test agent for integration testing",
			"type":         "go_plugin",
			"owner_id":     1,
			"capabilities": []string{"text_processing", "data_analysis"},
			"config": map[string]interface{}{
				"max_memory": "512MB",
				"timeout":    "30s",
			},
		}

		resp, err := suite.makeRequest("POST", suite.knirvnexusURL+"/api/v1/agents", agentData)
		require.NoError(t, err)

		var response map[string]interface{}
		err = json.Unmarshal(resp, &response)
		require.NoError(t, err)

		assert.True(t, response["success"].(bool))
		data := response["data"].(map[string]interface{})
		assert.NotEmpty(t, data["id"])
		assert.Equal(t, "TestAgent", data["name"])

		t.Logf("Agent created with ID: %s", data["id"])
	})

	// Test 2: List Agents
	t.Run("ListAgents", func(t *testing.T) {
		resp, err := suite.makeRequest("GET", suite.knirvnexusURL+"/api/v1/agents?owner_id=1", nil)
		require.NoError(t, err)

		var response map[string]interface{}
		err = json.Unmarshal(resp, &response)
		require.NoError(t, err)

		assert.True(t, response["success"].(bool))
		data := response["data"].([]interface{})
		t.Logf("Found %d agents", len(data))
	})

	// Test 3: Agent Execution
	t.Run("ExecuteAgent", func(t *testing.T) {
		// First get an agent ID
		resp, err := suite.makeRequest("GET", suite.knirvnexusURL+"/api/v1/agents?owner_id=1", nil)
		require.NoError(t, err)

		var response map[string]interface{}
		err = json.Unmarshal(resp, &response)
		require.NoError(t, err)

		data := response["data"].([]interface{})
		require.GreaterOrEqual(t, len(data), 1)

		agent := data[0].(map[string]interface{})
		agentID := agent["id"].(string)

		// Execute the agent
		execData := map[string]interface{}{
			"input": "Test input for agent execution",
			"parameters": map[string]interface{}{
				"mode": "test",
			},
		}

		resp, err = suite.makeRequest("POST", suite.knirvnexusURL+"/api/v1/agents/"+agentID+"/execute", execData)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(resp, &result)
		require.NoError(t, err)

		assert.NotEmpty(t, result["execution_id"])
		t.Logf("Agent execution started with ID: %s", result["execution_id"])
	})
}

// Test KNIRVROOT Blockchain Operations
func (suite *IntegrationTestSuite) TestKNIRVROOTBlockchain(t *testing.T) {
	// Test 1: Create Transaction
	t.Run("CreateTransaction", func(t *testing.T) {
		// Skip if wallet is not available (KNIRVWALLET service not running)
		if suite.testWallet == nil {
			t.Skip("KNIRVWALLET service not available - skipping blockchain tests")
			return
		}

		txData := map[string]interface{}{
			"from":   suite.testWallet.Address,
			"to":     "test_recipient_address",
			"amount": "1000000",
			"type":   "transfer",
		}

		resp, err := suite.makeRequest("POST", suite.knirvRootURL+"/transaction", txData)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(resp, &result)
		require.NoError(t, err)

		assert.NotEmpty(t, result["tx_hash"])
		t.Logf("Transaction created with hash: %s", result["tx_hash"])
	})

	// Test 2: Query Blockchain State
	t.Run("QueryBlockchainState", func(t *testing.T) {
		resp, err := suite.makeRequest("GET", suite.knirvRootURL+"/blockchain/state", nil)
		require.NoError(t, err)

		var state map[string]interface{}
		err = json.Unmarshal(resp, &state)
		require.NoError(t, err)

		assert.NotEmpty(t, state["latest_block"])
		assert.NotEmpty(t, state["total_transactions"])
		t.Logf("Blockchain state: %+v", state)
	})

	// Test 3: Wallet Balance
	t.Run("CheckWalletBalance", func(t *testing.T) {
		if suite.testWallet == nil {
			t.Skip("Test wallet not available")
			return
		}

		resp, err := suite.makeRequest("GET", suite.knirvRootURL+"/wallet/"+suite.testWallet.Address+"/balance", nil)
		require.NoError(t, err)

		var balance map[string]interface{}
		err = json.Unmarshal(resp, &balance)
		require.NoError(t, err)

		assert.NotEmpty(t, balance["balance"])
		t.Logf("Wallet balance: %s", balance["balance"])
	})
}

// Test KNIRVROUTER P2P and Connectivity
func (suite *IntegrationTestSuite) TestKNIRVROUTERConnectivity(t *testing.T) {
	// Test 1: Router Status
	t.Run("RouterStatus", func(t *testing.T) {
		resp, err := suite.makeRequest("GET", suite.knirvroterURL+"/status", nil)
		require.NoError(t, err)

		var status map[string]interface{}
		err = json.Unmarshal(resp, &status)
		require.NoError(t, err)

		assert.NotEmpty(t, status["node_id"])
		assert.NotEmpty(t, status["peer_count"])
		t.Logf("Router status: %+v", status)
	})

	// Test 2: Peer Discovery
	t.Run("PeerDiscovery", func(t *testing.T) {
		resp, err := suite.makeRequest("GET", suite.knirvroterURL+"/peers", nil)
		require.NoError(t, err)

		var peers []map[string]interface{}
		err = json.Unmarshal(resp, &peers)
		require.NoError(t, err)

		t.Logf("Found %d peers", len(peers))
	})

	// Test 3: Connectivity Proof
	t.Run("ConnectivityProof", func(t *testing.T) {
		proofData := map[string]interface{}{
			"target_peers": []string{"peer1", "peer2"},
			"proof_type":   "latency",
		}

		resp, err := suite.makeRequest("POST", suite.knirvroterURL+"/connectivity/proof", proofData)
		require.NoError(t, err)

		var proof map[string]interface{}
		err = json.Unmarshal(resp, &proof)
		require.NoError(t, err)

		assert.NotEmpty(t, proof["proof_id"])
		t.Logf("Connectivity proof generated: %s", proof["proof_id"])
	})
}

func TestIntegrationSuite(t *testing.T) {
	// Skip if services are not running
	suite := NewIntegrationTestSuite()

	// Test basic connectivity first
	client := &http.Client{Timeout: 2 * time.Second}
	_, err := client.Get(suite.knirvchainURL + "/health")
	if err != nil {
		t.Skip("Services not running - skipping integration tests. Run with setup script to start services.")
		return
	}

	// Run all test suites
	t.Run("FullWorkflow", suite.TestFullWorkflow)
	t.Run("KNIRVNEXUSAgentManagement", suite.TestKNIRVNEXUSAgentManagement)
	t.Run("KNIRVROOTBlockchain", suite.TestKNIRVROOTBlockchain)
	t.Run("KNIRVROUTERConnectivity", suite.TestKNIRVROUTERConnectivity)
}
