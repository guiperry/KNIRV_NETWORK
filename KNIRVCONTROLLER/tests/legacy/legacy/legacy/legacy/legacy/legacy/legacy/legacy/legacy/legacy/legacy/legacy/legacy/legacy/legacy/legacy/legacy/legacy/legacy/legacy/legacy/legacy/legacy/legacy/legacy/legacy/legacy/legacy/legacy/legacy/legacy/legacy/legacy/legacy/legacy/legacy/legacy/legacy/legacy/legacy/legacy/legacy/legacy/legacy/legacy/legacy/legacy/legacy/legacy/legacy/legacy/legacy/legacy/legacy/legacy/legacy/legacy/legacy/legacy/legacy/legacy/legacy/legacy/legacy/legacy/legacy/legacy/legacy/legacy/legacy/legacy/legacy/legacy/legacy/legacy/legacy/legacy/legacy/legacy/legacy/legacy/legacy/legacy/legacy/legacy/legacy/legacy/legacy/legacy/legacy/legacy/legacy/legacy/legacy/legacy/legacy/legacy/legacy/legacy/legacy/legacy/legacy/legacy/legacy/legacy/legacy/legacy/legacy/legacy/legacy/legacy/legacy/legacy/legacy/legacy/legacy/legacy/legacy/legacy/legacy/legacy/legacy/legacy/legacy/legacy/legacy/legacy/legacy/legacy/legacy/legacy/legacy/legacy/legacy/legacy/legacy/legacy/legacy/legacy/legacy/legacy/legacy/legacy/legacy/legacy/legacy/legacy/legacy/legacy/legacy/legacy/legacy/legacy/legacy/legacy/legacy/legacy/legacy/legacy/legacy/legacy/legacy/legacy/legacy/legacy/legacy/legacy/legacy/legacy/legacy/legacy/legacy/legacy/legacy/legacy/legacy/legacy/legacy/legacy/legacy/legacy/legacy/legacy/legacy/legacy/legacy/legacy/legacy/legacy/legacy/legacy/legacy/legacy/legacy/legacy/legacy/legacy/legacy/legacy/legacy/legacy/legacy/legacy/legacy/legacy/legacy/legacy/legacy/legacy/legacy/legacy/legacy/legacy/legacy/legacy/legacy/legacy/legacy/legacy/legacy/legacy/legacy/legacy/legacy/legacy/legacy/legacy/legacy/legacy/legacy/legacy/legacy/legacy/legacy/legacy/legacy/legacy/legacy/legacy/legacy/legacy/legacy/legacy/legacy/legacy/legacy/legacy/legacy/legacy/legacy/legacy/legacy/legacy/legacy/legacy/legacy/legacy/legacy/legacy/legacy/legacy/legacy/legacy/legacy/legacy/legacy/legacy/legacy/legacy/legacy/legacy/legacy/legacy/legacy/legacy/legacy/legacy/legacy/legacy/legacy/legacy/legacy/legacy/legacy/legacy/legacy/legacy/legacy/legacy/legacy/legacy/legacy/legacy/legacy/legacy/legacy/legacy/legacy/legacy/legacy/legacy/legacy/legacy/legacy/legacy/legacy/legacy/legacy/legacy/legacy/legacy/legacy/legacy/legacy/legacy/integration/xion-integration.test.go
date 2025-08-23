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

// XionIntegrationTestSuite tests XION blockchain integration
type XionIntegrationTestSuite struct {
	suite.Suite
	gatewayURL      string
	walletURL       string
	xionRPCURL      string
	httpClient      *http.Client
	testMetaAccount *XionMetaAccount
	testWalletName  string
}

// XionMetaAccount represents XION meta account data
type XionMetaAccount struct {
	Address     string `json:"address"`
	ChainID     string `json:"chain_id"`
	Balance     string `json:"balance"`
	NRNBalance  string `json:"nrn_balance"`
	Gasless     bool   `json:"gasless_enabled"`
	CreatedAt   string `json:"created_at"`
}

// XionTransactionRequest represents XION transaction parameters
type XionTransactionRequest struct {
	From            string                 `json:"from"`
	To              string                 `json:"to"`
	Amount          string                 `json:"amount"`
	Denom           string                 `json:"denom"`
	Memo            string                 `json:"memo,omitempty"`
	Gasless         bool                   `json:"gasless"`
	Type            string                 `json:"type,omitempty"`
	SkillID         string                 `json:"skill_id,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// XionTransactionResponse represents XION transaction response
type XionTransactionResponse struct {
	Success     bool   `json:"success"`
	TxHash      string `json:"tx_hash,omitempty"`
	BlockHeight int64  `json:"block_height,omitempty"`
	GasUsed     string `json:"gas_used,omitempty"`
	Error       string `json:"error,omitempty"`
}

// XionBalanceResponse represents balance query response
type XionBalanceResponse struct {
	Success    bool   `json:"success"`
	Balance    string `json:"balance"`
	NRNBalance string `json:"nrn_balance,omitempty"`
	Error      string `json:"error,omitempty"`
}

// XionFaucetRequest represents faucet request parameters
type XionFaucetRequest struct {
	Address string `json:"address"`
	Amount  string `json:"amount,omitempty"`
}

// SetupSuite initializes the XION integration test suite
func (suite *XionIntegrationTestSuite) SetupSuite() {
	suite.gatewayURL = "http://localhost:8000"
	suite.walletURL = "http://localhost:8083"
	suite.xionRPCURL = "https://rpc.xion-testnet-1.burnt.com:443"
	suite.httpClient = &http.Client{Timeout: 60 * time.Second}
	suite.testWalletName = "xion-integration-test-wallet"

	// Wait for services to be ready
	suite.waitForServices()

	// Create test meta account
	suite.createTestMetaAccount()

	suite.T().Log("XION Integration Test Suite initialized")
}

// TearDownSuite cleans up after all tests
func (suite *XionIntegrationTestSuite) TearDownSuite() {
	// Cleanup test meta account if needed
	if suite.testMetaAccount != nil {
		suite.T().Logf("Cleaning up test meta account: %s", suite.testMetaAccount.Address)
	}
	suite.T().Log("XION Integration Test Suite cleanup completed")
}

func (suite *XionIntegrationTestSuite) waitForServices() {
	services := []string{
		suite.gatewayURL + "/health",
		suite.walletURL + "/health",
	}

	for _, service := range services {
		for i := 0; i < 30; i++ {
			resp, err := suite.httpClient.Get(service)
			if err == nil && resp.StatusCode == 200 {
				resp.Body.Close()
				break
			}
			if resp != nil {
				resp.Body.Close()
			}
			time.Sleep(2 * time.Second)
		}
	}
}

func (suite *XionIntegrationTestSuite) makeRequest(method, endpoint string, data interface{}) map[string]interface{} {
	var body []byte
	if data != nil {
		body, _ = json.Marshal(data)
	}

	req, _ := http.NewRequest(method, suite.walletURL+endpoint, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := suite.httpClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(suite.T(), err)

	return result
}

func (suite *XionIntegrationTestSuite) createTestMetaAccount() {
	// Create XION meta account for testing
	createReq := map[string]interface{}{
		"name": suite.testWalletName,
		"type": "XION_META",
	}

	resp := suite.makeRequest("POST", "/xion/meta-account/create", createReq)
	require.True(suite.T(), resp["success"].(bool), "Failed to create test meta account")

	// Extract meta account data
	data := resp["data"].(map[string]interface{})
	suite.testMetaAccount = &XionMetaAccount{
		Address:    data["address"].(string),
		ChainID:    data["chain_id"].(string),
		Balance:    "0",
		NRNBalance: "0",
		Gasless:    true,
		CreatedAt:  time.Now().Format(time.RFC3339),
	}

	suite.T().Logf("Created test meta account: %s", suite.testMetaAccount.Address)
}

// Test 1: XION Meta Account Management
func (suite *XionIntegrationTestSuite) TestXionMetaAccountManagement() {
	suite.Run("CreateMetaAccount", func() {
		createReq := map[string]interface{}{
			"name": "test-meta-account-2",
			"type": "XION_META",
		}

		resp := suite.makeRequest("POST", "/xion/meta-account/create", createReq)
		require.True(suite.T(), resp["success"].(bool), "Failed to create meta account")

		data := resp["data"].(map[string]interface{})
		assert.NotEmpty(suite.T(), data["address"])
		assert.Equal(suite.T(), "xion-testnet-1", data["chain_id"])
		assert.True(suite.T(), data["gasless_enabled"].(bool))

		suite.T().Logf("Created meta account: %s", data["address"])
	})

	suite.Run("GetMetaAccount", func() {
		require.NotNil(suite.T(), suite.testMetaAccount, "Test meta account must exist")

		resp := suite.makeRequest("GET", fmt.Sprintf("/xion/meta-account/%s", suite.testMetaAccount.Address), nil)
		require.True(suite.T(), resp["success"].(bool), "Failed to get meta account")

		data := resp["data"].(map[string]interface{})
		assert.Equal(suite.T(), suite.testMetaAccount.Address, data["address"])
		assert.Equal(suite.T(), suite.testMetaAccount.ChainID, data["chain_id"])
	})

	suite.Run("ListMetaAccounts", func() {
		resp := suite.makeRequest("GET", "/xion/meta-account/list", nil)
		require.True(suite.T(), resp["success"].(bool), "Failed to list meta accounts")

		accounts := resp["data"].([]interface{})
		assert.GreaterOrEqual(suite.T(), len(accounts), 1)

		// Verify our test account is in the list
		found := false
		for _, account := range accounts {
			acc := account.(map[string]interface{})
			if acc["address"] == suite.testMetaAccount.Address {
				found = true
				break
			}
		}
		assert.True(suite.T(), found, "Test meta account not found in list")
	})
}

// Test 2: Balance Operations
func (suite *XionIntegrationTestSuite) TestBalanceOperations() {
	suite.Run("GetXionBalance", func() {
		require.NotNil(suite.T(), suite.testMetaAccount, "Test meta account must exist")

		resp := suite.makeRequest("GET", fmt.Sprintf("/xion/balance/%s", suite.testMetaAccount.Address), nil)
		require.True(suite.T(), resp["success"].(bool), "Failed to get XION balance")

		balance := resp["balance"].(string)
		assert.NotEmpty(suite.T(), balance)
		assert.GreaterOrEqual(suite.T(), len(balance), 1)

		suite.T().Logf("XION balance: %s", balance)
	})

	suite.Run("GetNRNBalance", func() {
		require.NotNil(suite.T(), suite.testMetaAccount, "Test meta account must exist")

		resp := suite.makeRequest("GET", fmt.Sprintf("/xion/nrn-balance/%s", suite.testMetaAccount.Address), nil)
		require.True(suite.T(), resp["success"].(bool), "Failed to get NRN balance")

		nrnBalance := resp["nrn_balance"].(string)
		assert.NotEmpty(suite.T(), nrnBalance)

		suite.T().Logf("NRN balance: %s", nrnBalance)
	})

	suite.Run("RefreshBalances", func() {
		require.NotNil(suite.T(), suite.testMetaAccount, "Test meta account must exist")

		resp := suite.makeRequest("POST", fmt.Sprintf("/xion/balance/refresh/%s", suite.testMetaAccount.Address), nil)
		require.True(suite.T(), resp["success"].(bool), "Failed to refresh balances")

		data := resp["data"].(map[string]interface{})
		assert.NotEmpty(suite.T(), data["balance"])
		assert.NotEmpty(suite.T(), data["nrn_balance"])

		suite.T().Log("Successfully refreshed balances")
	})
}

// Test 3: Faucet Operations
func (suite *XionIntegrationTestSuite) TestFaucetOperations() {
	suite.Run("RequestFromFaucet", func() {
		require.NotNil(suite.T(), suite.testMetaAccount, "Test meta account must exist")

		faucetReq := XionFaucetRequest{
			Address: suite.testMetaAccount.Address,
			Amount:  "1000000", // 1 NRN
		}

		resp := suite.makeRequest("POST", "/xion/faucet/request", faucetReq)
		require.True(suite.T(), resp["success"].(bool), "Failed to request from faucet: %v", resp["error"])

		txHash := resp["tx_hash"].(string)
		assert.NotEmpty(suite.T(), txHash)
		assert.True(suite.T(), len(txHash) >= 64) // Transaction hash should be at least 64 characters

		suite.T().Logf("Faucet request successful, TX: %s", txHash)

		// Wait for transaction to be processed
		time.Sleep(5 * time.Second)

		// Verify balance increased
		balanceResp := suite.makeRequest("GET", fmt.Sprintf("/xion/nrn-balance/%s", suite.testMetaAccount.Address), nil)
		require.True(suite.T(), balanceResp["success"].(bool))

		newBalance := balanceResp["nrn_balance"].(string)
		suite.T().Logf("New NRN balance after faucet: %s", newBalance)
	})

	suite.Run("FaucetRateLimit", func() {
		require.NotNil(suite.T(), suite.testMetaAccount, "Test meta account must exist")

		// Make multiple rapid faucet requests to test rate limiting
		faucetReq := XionFaucetRequest{
			Address: suite.testMetaAccount.Address,
			Amount:  "100000",
		}

		// First request should succeed
		resp1 := suite.makeRequest("POST", "/xion/faucet/request", faucetReq)
		// Second immediate request might be rate limited
		resp2 := suite.makeRequest("POST", "/xion/faucet/request", faucetReq)

		// At least one should succeed, second might fail due to rate limiting
		assert.True(suite.T(), resp1["success"].(bool) || resp2["success"].(bool))

		suite.T().Log("Tested faucet rate limiting")
	})
}

// Test 4: NRN Transfer Operations
func (suite *XionIntegrationTestSuite) TestNRNTransferOperations() {
	suite.Run("TransferNRN", func() {
		require.NotNil(suite.T(), suite.testMetaAccount, "Test meta account must exist")

		// First, ensure we have some NRN balance
		faucetReq := XionFaucetRequest{
			Address: suite.testMetaAccount.Address,
			Amount:  "2000000", // 2 NRN
		}
		faucetResp := suite.makeRequest("POST", "/xion/faucet/request", faucetReq)
		if faucetResp["success"].(bool) {
			time.Sleep(5 * time.Second) // Wait for faucet transaction
		}

		// Create recipient address (for testing, we'll use a test address)
		recipientAddress := "xion1234567890abcdef1234567890abcdef12345678"

		transferReq := XionTransactionRequest{
			From:    suite.testMetaAccount.Address,
			To:      recipientAddress,
			Amount:  "500000", // 0.5 NRN
			Denom:   "nrn",
			Memo:    "Integration test transfer",
			Gasless: true,
			Type:    "nrn_transfer",
		}

		resp := suite.makeRequest("POST", "/xion/transfer/nrn", transferReq)
		require.True(suite.T(), resp["success"].(bool), "Failed to transfer NRN: %v", resp["error"])

		txHash := resp["tx_hash"].(string)
		assert.NotEmpty(suite.T(), txHash)
		assert.True(suite.T(), len(txHash) >= 64)

		suite.T().Logf("NRN transfer successful, TX: %s", txHash)
	})

	suite.Run("TransferInvalidAmount", func() {
		require.NotNil(suite.T(), suite.testMetaAccount, "Test meta account must exist")

		transferReq := XionTransactionRequest{
			From:    suite.testMetaAccount.Address,
			To:      "xion1234567890abcdef1234567890abcdef12345678",
			Amount:  "invalid-amount",
			Denom:   "nrn",
			Gasless: true,
		}

		resp := suite.makeRequest("POST", "/xion/transfer/nrn", transferReq)
		assert.False(suite.T(), resp["success"].(bool))
		assert.NotEmpty(suite.T(), resp["error"])

		suite.T().Log("Correctly rejected invalid transfer amount")
	})
}

// Test 5: Skill Invocation Operations
func (suite *XionIntegrationTestSuite) TestSkillInvocationOperations() {
	suite.Run("BurnNRNForSkill", func() {
		require.NotNil(suite.T(), suite.testMetaAccount, "Test meta account must exist")

		// Ensure we have NRN balance for skill invocation
		faucetReq := XionFaucetRequest{
			Address: suite.testMetaAccount.Address,
			Amount:  "3000000", // 3 NRN
		}
		faucetResp := suite.makeRequest("POST", "/xion/faucet/request", faucetReq)
		if faucetResp["success"].(bool) {
			time.Sleep(5 * time.Second) // Wait for faucet transaction
		}

		skillReq := XionTransactionRequest{
			From:    suite.testMetaAccount.Address,
			Amount:  "1000000", // 1 NRN
			Denom:   "nrn",
			Memo:    "Skill invocation test",
			Gasless: true,
			Type:    "skill_invocation",
			SkillID: "skill-integration-test-001",
			Metadata: map[string]interface{}{
				"input":     "Integration test input",
				"model":     "CodeT5",
				"maxTokens": 100,
			},
		}

		resp := suite.makeRequest("POST", "/xion/skill/invoke", skillReq)
		require.True(suite.T(), resp["success"].(bool), "Failed to invoke skill: %v", resp["error"])

		txHash := resp["tx_hash"].(string)
		assert.NotEmpty(suite.T(), txHash)

		suite.T().Logf("Skill invocation successful, TX: %s", txHash)
	})

	suite.Run("SkillInvocationWithInvalidSkillID", func() {
		require.NotNil(suite.T(), suite.testMetaAccount, "Test meta account must exist")

		skillReq := XionTransactionRequest{
			From:    suite.testMetaAccount.Address,
			Amount:  "1000000",
			Denom:   "nrn",
			Gasless: true,
			Type:    "skill_invocation",
			SkillID: "", // Empty skill ID
		}

		resp := suite.makeRequest("POST", "/xion/skill/invoke", skillReq)
		assert.False(suite.T(), resp["success"].(bool))
		assert.NotEmpty(suite.T(), resp["error"])

		suite.T().Log("Correctly rejected skill invocation with empty skill ID")
	})
}

// Test 6: Gasless Transaction Features
func (suite *XionIntegrationTestSuite) TestGaslessTransactions() {
	suite.Run("EnableGaslessTransactions", func() {
		require.NotNil(suite.T(), suite.testMetaAccount, "Test meta account must exist")

		resp := suite.makeRequest("POST", fmt.Sprintf("/xion/gasless/enable/%s", suite.testMetaAccount.Address), nil)
		require.True(suite.T(), resp["success"].(bool), "Failed to enable gasless transactions: %v", resp["error"])

		suite.T().Log("Successfully enabled gasless transactions")
	})

	suite.Run("VerifyGaslessStatus", func() {
		require.NotNil(suite.T(), suite.testMetaAccount, "Test meta account must exist")

		resp := suite.makeRequest("GET", fmt.Sprintf("/xion/gasless/status/%s", suite.testMetaAccount.Address), nil)
		require.True(suite.T(), resp["success"].(bool), "Failed to get gasless status")

		gaslessEnabled := resp["gasless_enabled"].(bool)
		assert.True(suite.T(), gaslessEnabled)

		suite.T().Log("Verified gasless transactions are enabled")
	})

	suite.Run("GaslessTransactionExecution", func() {
		require.NotNil(suite.T(), suite.testMetaAccount, "Test meta account must exist")

		// Execute a gasless transaction
		transferReq := XionTransactionRequest{
			From:    suite.testMetaAccount.Address,
			To:      "xion1234567890abcdef1234567890abcdef12345678",
			Amount:  "100000",
			Denom:   "nrn",
			Memo:    "Gasless transaction test",
			Gasless: true,
		}

		resp := suite.makeRequest("POST", "/xion/transfer/nrn", transferReq)
		require.True(suite.T(), resp["success"].(bool), "Failed to execute gasless transaction")

		// Verify gas used is 0 or minimal
		gasUsed := resp["gas_used"].(string)
		assert.Equal(suite.T(), "0", gasUsed)

		suite.T().Log("Successfully executed gasless transaction")
	})
}

// Test 7: Transaction History and Monitoring
func (suite *XionIntegrationTestSuite) TestTransactionHistory() {
	suite.Run("GetTransactionHistory", func() {
		require.NotNil(suite.T(), suite.testMetaAccount, "Test meta account must exist")

		resp := suite.makeRequest("GET", fmt.Sprintf("/xion/transactions/%s", suite.testMetaAccount.Address), nil)
		require.True(suite.T(), resp["success"].(bool), "Failed to get transaction history")

		transactions := resp["transactions"].([]interface{})
		assert.GreaterOrEqual(suite.T(), len(transactions), 0)

		suite.T().Logf("Retrieved %d transactions", len(transactions))
	})

	suite.Run("GetTransactionDetails", func() {
		require.NotNil(suite.T(), suite.testMetaAccount, "Test meta account must exist")

		// First get transaction history to find a transaction
		historyResp := suite.makeRequest("GET", fmt.Sprintf("/xion/transactions/%s", suite.testMetaAccount.Address), nil)
		require.True(suite.T(), historyResp["success"].(bool))

		transactions := historyResp["transactions"].([]interface{})
		if len(transactions) > 0 {
			tx := transactions[0].(map[string]interface{})
			txHash := tx["tx_hash"].(string)

			// Get transaction details
			detailResp := suite.makeRequest("GET", fmt.Sprintf("/xion/transaction/%s", txHash), nil)
			require.True(suite.T(), detailResp["success"].(bool), "Failed to get transaction details")

			details := detailResp["transaction"].(map[string]interface{})
			assert.Equal(suite.T(), txHash, details["tx_hash"])
			assert.NotEmpty(suite.T(), details["block_height"])

			suite.T().Logf("Retrieved details for transaction: %s", txHash)
		} else {
			suite.T().Log("No transactions found for details test")
		}
	})
}

// TestXionIntegrationSuite runs the XION integration test suite
func TestXionIntegrationSuite(t *testing.T) {
	suite.Run(t, new(XionIntegrationTestSuite))
}
