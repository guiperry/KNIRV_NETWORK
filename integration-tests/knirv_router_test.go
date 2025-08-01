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

// Month 12 KNIRV-ROUTER Connectivity Testing Suite
type KNIRVROUTERTestSuite struct {
	suite.Suite
	gatewayURL   string
	routerURL    string
	httpClient   *http.Client
	authToken    string
	testWallet   *TestWallet
	testRouterID string
}

type ConnectivityProof struct {
	ProofID   string                 `json:"proof_id"`
	RouterID  string                 `json:"router_id"`
	Status    string                 `json:"status"`
	Timestamp string                 `json:"timestamp"`
	Peers     []string               `json:"peers"`
	Metrics   map[string]interface{} `json:"metrics"`
	Reward    string                 `json:"reward,omitempty"`
}

type TURNServerStatus struct {
	Active      bool     `json:"active"`
	Connections int      `json:"connections"`
	Bandwidth   string   `json:"bandwidth"`
	Peers       []string `json:"peers"`
	Uptime      string   `json:"uptime"`
}

type NRNMintingResult struct {
	Success     bool   `json:"success"`
	TxHash      string `json:"tx_hash"`
	Amount      string `json:"amount"`
	Recipient   string `json:"recipient"`
	ProofID     string `json:"proof_id"`
	BlockHeight int64  `json:"block_height"`
}

func (suite *KNIRVROUTERTestSuite) SetupSuite() {
	suite.gatewayURL = "http://localhost:8000"
	suite.routerURL = "http://localhost:8000/knirvrouter"
	suite.httpClient = &http.Client{Timeout: 30 * time.Second}
	suite.testRouterID = "test_router_" + fmt.Sprintf("%d", time.Now().Unix())

	// Wait for KNIRV-ROUTER service
	suite.waitForKNIRVROUTER()

	// Authenticate
	suite.authenticate()

	// Create test wallet
	suite.createTestWallet()
}

func (suite *KNIRVROUTERTestSuite) waitForKNIRVROUTER() {
	suite.T().Log("Waiting for KNIRV-ROUTER service to be ready...")

	for i := 0; i < 30; i++ {
		resp, err := suite.httpClient.Get(suite.routerURL + "/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			suite.T().Log("KNIRV-ROUTER service is ready")
			return
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(1 * time.Second)
	}

	suite.T().Fatal("KNIRV-ROUTER service failed to start within timeout")
}

func (suite *KNIRVROUTERTestSuite) authenticate() {
	loginData := map[string]string{
		"username": "admin",
		"password": "password",
	}

	resp := suite.makeRequest("POST", "/auth/login", loginData)
	require.True(suite.T(), resp.Success, "Authentication failed")

	suite.authToken = resp.Data["token"].(string)
	suite.T().Log("Authenticated for KNIRV-ROUTER testing")
}

func (suite *KNIRVROUTERTestSuite) createTestWallet() {
	walletData := map[string]string{
		"name": "knirv_router_test_wallet",
	}

	resp := suite.makeAuthenticatedRequest("POST", "/knirvwallet/wallet/create", walletData)
	require.True(suite.T(), resp.Success, "Failed to create test wallet")

	suite.testWallet = &TestWallet{
		Address:  resp.Data["address"].(string),
		Mnemonic: resp.Data["mnemonic"].(string),
		Balance:  "0",
	}

	// Fund the wallet
	suite.fundTestWallet("20000000") // 20 NRN

	suite.T().Logf("Created and funded KNIRV-ROUTER test wallet: %s", suite.testWallet.Address)
}

func (suite *KNIRVROUTERTestSuite) fundTestWallet(amount string) {
	fundData := map[string]interface{}{
		"address": suite.testWallet.Address,
		"amount":  amount,
	}

	resp := suite.makeAuthenticatedRequest("POST", "/knirvroot/faucet/fund", fundData)
	require.True(suite.T(), resp.Success, "Failed to fund test wallet")

	suite.testWallet.Balance = amount
}

// Test 1: KNIRV-ROUTER Proof-of-Connectivity Engine
func (suite *KNIRVROUTERTestSuite) TestProofOfConnectivityEngine() {
	suite.Run("ProofOfConnectivityEngineTest", func() {
		// Test 1.1: Check proof engine status
		suite.T().Log("Testing proof-of-connectivity engine status...")

		statusResp := suite.makeAuthenticatedRequest("GET", "/knirvrouter/api/connectivity/status", nil)
		assert.True(suite.T(), statusResp.Success, "Failed to get connectivity status")

		if statusResp.Success {
			status := statusResp.Data
			assert.Contains(suite.T(), status, "proof_engine_active")

			proofEngineActive := status["proof_engine_active"].(bool)
			assert.True(suite.T(), proofEngineActive, "Proof-of-connectivity engine is not active")

			suite.T().Logf("Proof-of-connectivity engine status: %+v", status)
		}

		// Test 1.2: Initiate connectivity proof
		suite.T().Log("Testing connectivity proof initiation...")

		proofData := map[string]interface{}{
			"router_id":      suite.testRouterID,
			"proof_type":     "comprehensive",
			"target_peers":   []string{"peer1", "peer2", "peer3"},
			"reward_address": suite.testWallet.Address,
		}

		proofResp := suite.makeAuthenticatedRequest("POST", "/knirvrouter/api/connectivity/proofs", proofData)
		assert.True(suite.T(), proofResp.Success, "Failed to initiate connectivity proof")

		var proof ConnectivityProof
		if proofResp.Success {
			proofBytes, _ := json.Marshal(proofResp.Data)
			json.Unmarshal(proofBytes, &proof)

			assert.NotEmpty(suite.T(), proof.ProofID, "Proof ID should not be empty")
			assert.Equal(suite.T(), suite.testRouterID, proof.RouterID, "Router ID mismatch")
			assert.Contains(suite.T(), []string{"initiated", "processing", "completed"}, proof.Status, "Invalid proof status")

			suite.T().Logf("Connectivity proof initiated: %+v", proof)
		}

		// Test 1.3: Monitor proof progress
		suite.T().Log("Monitoring proof progress...")

		if proofResp.Success {
			proofID := proof.ProofID

			// Poll for proof completion (up to 60 seconds)
			for i := 0; i < 60; i++ {
				statusResp := suite.makeAuthenticatedRequest("GET", fmt.Sprintf("/knirvrouter/api/connectivity/proofs/%s", proofID), nil)

				if statusResp.Success {
					var currentProof ConnectivityProof
					proofBytes, _ := json.Marshal(statusResp.Data)
					json.Unmarshal(proofBytes, &currentProof)

					suite.T().Logf("Proof status: %s", currentProof.Status)

					if currentProof.Status == "completed" || currentProof.Status == "failed" {
						assert.Equal(suite.T(), "completed", currentProof.Status, "Proof should complete successfully")
						break
					}
				}

				time.Sleep(1 * time.Second)
			}
		}

		// Test 1.4: Verify proof history
		suite.T().Log("Verifying proof history...")

		historyResp := suite.makeAuthenticatedRequest("GET", "/knirvrouter/api/connectivity/proofs", nil)
		assert.True(suite.T(), historyResp.Success, "Failed to get proof history")

		if historyResp.Success {
			proofs := historyResp.Data["proofs"].([]interface{})
			assert.Greater(suite.T(), len(proofs), 0, "Should have at least one proof in history")

			suite.T().Logf("Found %d proofs in history", len(proofs))
		}
	})
}

// Test 2: TURN Server Functionality
func (suite *KNIRVROUTERTestSuite) TestTURNServerFunctionality() {
	suite.Run("TURNServerFunctionalityTest", func() {
		// Test 2.1: TURN server status
		suite.T().Log("Testing TURN server status...")

		turnStatusResp := suite.makeAuthenticatedRequest("GET", "/knirvrouter/turn/status", nil)

		if turnStatusResp.Success {
			var turnStatus TURNServerStatus
			statusBytes, _ := json.Marshal(turnStatusResp.Data)
			json.Unmarshal(statusBytes, &turnStatus)

			suite.T().Logf("TURN server status: %+v", turnStatus)

			// TURN server should be active for connectivity testing
			assert.True(suite.T(), turnStatus.Active, "TURN server should be active")
			assert.GreaterOrEqual(suite.T(), turnStatus.Connections, 0, "Connection count should be non-negative")
		} else {
			suite.T().Logf("TURN server status endpoint returned error: %s", turnStatusResp.Error)
		}

		// Test 2.2: TURN server configuration
		suite.T().Log("Testing TURN server configuration...")

		configResp := suite.makeAuthenticatedRequest("GET", "/knirvrouter/turn/config", nil)

		if configResp.Success {
			config := configResp.Data
			suite.T().Logf("TURN server configuration: %+v", config)

			// Verify essential configuration parameters
			assert.Contains(suite.T(), config, "server_url", "TURN server URL should be configured")
			assert.Contains(suite.T(), config, "realm", "TURN realm should be configured")
		} else {
			suite.T().Logf("TURN server config endpoint returned error: %s", configResp.Error)
		}

		// Test 2.3: TURN server peer discovery
		suite.T().Log("Testing TURN server peer discovery...")

		peersResp := suite.makeAuthenticatedRequest("GET", "/knirvrouter/turn/peers", nil)

		if peersResp.Success {
			peers := peersResp.Data["peers"].([]interface{})
			suite.T().Logf("Discovered %d peers via TURN server", len(peers))

			// Log peer information
			for i, peer := range peers {
				suite.T().Logf("Peer %d: %+v", i+1, peer)
			}
		} else {
			suite.T().Logf("TURN server peers endpoint returned error: %s", peersResp.Error)
		}
	})
}

// Test 3: NRN Minting Capabilities
func (suite *KNIRVROUTERTestSuite) TestNRNMintingCapabilities() {
	suite.Run("NRNMintingCapabilitiesTest", func() {
		// Test 3.1: Check minting permissions
		suite.T().Log("Testing NRN minting permissions...")

		permissionsResp := suite.makeAuthenticatedRequest("GET", "/knirvrouter/api/minting/permissions", nil)

		if permissionsResp.Success {
			permissions := permissionsResp.Data
			suite.T().Logf("Minting permissions: %+v", permissions)

			canMint := permissions["can_mint"].(bool)
			assert.True(suite.T(), canMint, "KNIRV-ROUTER should have NRN minting permissions")
		} else {
			suite.T().Logf("Minting permissions endpoint returned error: %s", permissionsResp.Error)
		}

		// Test 3.2: Test NRN minting for connectivity rewards
		suite.T().Log("Testing NRN minting for connectivity rewards...")

		// First create a connectivity proof to earn rewards
		proofData := map[string]interface{}{
			"router_id":      suite.testRouterID + "_mint_test",
			"proof_type":     "reward_test",
			"reward_address": suite.testWallet.Address,
		}

		proofResp := suite.makeAuthenticatedRequest("POST", "/knirvrouter/api/connectivity/proofs", proofData)

		if proofResp.Success {
			proofID := proofResp.Data["proof_id"].(string)

			// Wait for proof processing
			time.Sleep(10 * time.Second)

			// Test minting rewards for the proof
			mintData := map[string]interface{}{
				"proof_id":  proofID,
				"amount":    "1000000", // 1 NRN reward
				"recipient": suite.testWallet.Address,
				"reason":    "connectivity_proof_reward",
			}

			mintResp := suite.makeAuthenticatedRequest("POST", "/knirvrouter/api/minting/mint", mintData)

			if mintResp.Success {
				var mintResult NRNMintingResult
				mintBytes, _ := json.Marshal(mintResp.Data)
				json.Unmarshal(mintBytes, &mintResult)

				assert.True(suite.T(), mintResult.Success, "NRN minting should succeed")
				assert.NotEmpty(suite.T(), mintResult.TxHash, "Transaction hash should not be empty")
				assert.Equal(suite.T(), "1000000", mintResult.Amount, "Minted amount should match")
				assert.Equal(suite.T(), suite.testWallet.Address, mintResult.Recipient, "Recipient should match")

				suite.T().Logf("NRN minting result: %+v", mintResult)
			} else {
				suite.T().Logf("NRN minting failed: %s", mintResp.Error)
			}
		}

		// Test 3.3: Verify minting history
		suite.T().Log("Verifying NRN minting history...")

		historyResp := suite.makeAuthenticatedRequest("GET", "/knirvrouter/api/minting/history", nil)

		if historyResp.Success {
			history := historyResp.Data["minting_history"].([]interface{})
			suite.T().Logf("Found %d minting transactions in history", len(history))

			// Verify recent minting transaction
			if len(history) > 0 {
				latestMint := history[0].(map[string]interface{})
				suite.T().Logf("Latest minting transaction: %+v", latestMint)
			}
		} else {
			suite.T().Logf("Minting history endpoint returned error: %s", historyResp.Error)
		}
	})
}

// Test 4: Network Connectivity and Peer Management
func (suite *KNIRVROUTERTestSuite) TestNetworkConnectivityAndPeerManagement() {
	suite.Run("NetworkConnectivityAndPeerManagementTest", func() {
		// Test 4.1: Router network join
		suite.T().Log("Testing router network join...")

		joinData := map[string]interface{}{
			"router_id":      suite.testRouterID,
			"capabilities":   []string{"relay", "storage", "compute"},
			"stake_amount":   "5000000", // 5 NRN stake
			"wallet_address": suite.testWallet.Address,
		}

		joinResp := suite.makeAuthenticatedRequest("POST", "/knirvrouter/network/join", joinData)

		if joinResp.Success {
			joinResult := joinResp.Data
			assert.Equal(suite.T(), "joined", joinResult["status"], "Router should successfully join network")

			suite.T().Logf("Router network join result: %+v", joinResult)
		} else {
			suite.T().Logf("Router network join failed: %s", joinResp.Error)
		}

		// Test 4.2: Peer discovery
		suite.T().Log("Testing peer discovery...")

		peersResp := suite.makeAuthenticatedRequest("GET", "/knirvrouter/peers", nil)

		if peersResp.Success {
			peers := peersResp.Data["peers"].([]interface{})
			suite.T().Logf("Discovered %d peers in the network", len(peers))

			// Log peer details
			for i, peer := range peers {
				peerMap := peer.(map[string]interface{})
				suite.T().Logf("Peer %d: ID=%s, Capabilities=%v", i+1, peerMap["id"], peerMap["capabilities"])
			}
		} else {
			suite.T().Logf("Peer discovery failed: %s", peersResp.Error)
		}

		// Test 4.3: Network status
		suite.T().Log("Testing network status...")

		networkStatusResp := suite.makeAuthenticatedRequest("GET", "/knirvrouter/network/status", nil)

		if networkStatusResp.Success {
			networkStatus := networkStatusResp.Data
			suite.T().Logf("Network status: %+v", networkStatus)

			// Verify network health metrics
			assert.Contains(suite.T(), networkStatus, "total_peers", "Network status should include total peers")
			assert.Contains(suite.T(), networkStatus, "active_connections", "Network status should include active connections")
		} else {
			suite.T().Logf("Network status check failed: %s", networkStatusResp.Error)
		}
	})
}

// Test 5: Integration with KNIRVGATEWAY
func (suite *KNIRVROUTERTestSuite) TestKNIRVGATEWAYIntegration() {
	suite.Run("KNIRVGATEWAYIntegrationTest", func() {
		// Test 5.1: Gateway routing to KNIRV-ROUTER
		suite.T().Log("Testing KNIRVGATEWAY routing to KNIRV-ROUTER...")

		routingResp := suite.makeAuthenticatedRequest("GET", "/knirvrouter/health", nil)
		assert.True(suite.T(), routingResp.Success, "KNIRVGATEWAY should successfully route to KNIRV-ROUTER")

		if routingResp.Success {
			suite.T().Log("KNIRVGATEWAY successfully routes to KNIRV-ROUTER")
		}

		// Test 5.2: Authentication through gateway
		suite.T().Log("Testing authentication through KNIRVGATEWAY...")

		authTestResp := suite.makeAuthenticatedRequest("GET", "/knirvrouter/api/connectivity/status", nil)
		assert.True(suite.T(), authTestResp.Success, "Authentication should work through KNIRVGATEWAY")

		// Test 5.3: Data forwarding through gateway
		suite.T().Log("Testing data forwarding through KNIRVGATEWAY...")

		testData := map[string]interface{}{
			"test":      true,
			"timestamp": time.Now().Unix(),
			"router_id": suite.testRouterID,
		}

		forwardResp := suite.makeAuthenticatedRequest("POST", "/knirvrouter/test/echo", testData)

		if forwardResp.Success {
			suite.T().Log("KNIRVGATEWAY successfully forwards data to KNIRV-ROUTER")
		} else {
			suite.T().Logf("Data forwarding through gateway failed: %s", forwardResp.Error)
		}
	})
}

// Helper methods
func (suite *KNIRVROUTERTestSuite) makeRequest(method, path string, data interface{}) *TestResponse {
	return suite.makeRequestWithAuth(method, path, data, "")
}

func (suite *KNIRVROUTERTestSuite) makeAuthenticatedRequest(method, path string, data interface{}) *TestResponse {
	return suite.makeRequestWithAuth(method, path, data, suite.authToken)
}

func (suite *KNIRVROUTERTestSuite) makeRequestWithAuth(method, path string, data interface{}, token string) *TestResponse {
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

// Main test function for the KNIRV-ROUTER Test Suite
func TestKNIRVROUTERTestSuite(t *testing.T) {
	suite.Run(t, new(KNIRVROUTERTestSuite))
}
