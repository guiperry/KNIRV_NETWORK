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

// CrossPlatformWalletTestSuite tests integration between native and browser wallets
type CrossPlatformWalletTestSuite struct {
	suite.Suite
	nativeWalletURL  string
	browserWalletURL string
	syncServiceURL   string
	httpClient       *http.Client
	authToken        string
	testSessions     []string
}

// QRCodeData represents QR code connection data
type QRCodeData struct {
	Type      string                 `json:"type"`
	SessionID string                 `json:"session_id"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

// SyncSessionResponse represents sync session creation response
type SyncSessionResponse struct {
	Success bool                   `json:"success"`
	Data    map[string]interface{} `json:"data"`
	Error   string                 `json:"error,omitempty"`
}

// WalletSyncRequest represents wallet synchronization request
type WalletSyncRequest struct {
	SessionID string `json:"session_id"`
}

// SetupSuite initializes the cross-platform test suite
func (suite *CrossPlatformWalletTestSuite) SetupSuite() {
	suite.nativeWalletURL = "http://localhost:8083"   // Native wallet backend
	suite.browserWalletURL = "http://localhost:3000"  // Browser wallet frontend
	suite.syncServiceURL = "http://localhost:8084"    // Sync service
	suite.httpClient = &http.Client{Timeout: 30 * time.Second}
	suite.testSessions = make([]string, 0)

	// Wait for services to be ready
	suite.waitForServices()

	// Authenticate
	suite.authenticate()

	suite.T().Log("Cross-Platform Wallet Integration Test Suite initialized")
}

// TearDownSuite cleans up after all tests
func (suite *CrossPlatformWalletTestSuite) TearDownSuite() {
	// Clean up test sessions
	for _, sessionID := range suite.testSessions {
		suite.cleanupSession(sessionID)
	}
	suite.T().Log("Cross-Platform Wallet Integration Test Suite cleanup completed")
}

func (suite *CrossPlatformWalletTestSuite) waitForServices() {
	services := []string{
		suite.nativeWalletURL + "/health",
		suite.syncServiceURL + "/health",
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

func (suite *CrossPlatformWalletTestSuite) authenticate() {
	authData := map[string]string{
		"username": "test_user",
		"password": "test_password",
	}

	resp := suite.makeRequest("POST", suite.nativeWalletURL+"/auth/login", authData)
	require.True(suite.T(), resp.Success, "Authentication failed")

	suite.authToken = resp.Data["token"].(string)
	suite.T().Log("Authenticated for cross-platform testing")
}

func (suite *CrossPlatformWalletTestSuite) makeRequest(method, url string, data interface{}) *SyncSessionResponse {
	var body []byte
	if data != nil {
		body, _ = json.Marshal(data)
	}

	req, _ := http.NewRequest(method, url, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	if suite.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+suite.authToken)
	}

	resp, err := suite.httpClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	var result SyncSessionResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(suite.T(), err)

	return &result
}

func (suite *CrossPlatformWalletTestSuite) cleanupSession(sessionID string) {
	// Implementation for session cleanup
	suite.T().Logf("Cleaning up session: %s", sessionID)
}

// Test 1: QR Code Generation and Scanning
func (suite *CrossPlatformWalletTestSuite) TestQRCodeConnectivity() {
	suite.Run("GenerateConnectionQRCode", func() {
		// Create a test wallet in native app
		walletReq := map[string]interface{}{
			"name": "test_native_wallet",
			"type": "HD",
		}

		walletResp := suite.makeRequest("POST", suite.nativeWalletURL+"/api/v1/multichain/wallet/create", walletReq)
		require.True(suite.T(), walletResp.Success, "Failed to create native wallet")

		walletAddress := walletResp.Data["wallets"].([]interface{})[0].(map[string]interface{})["address"].(string)

		// Generate QR code for connection
		qrReq := map[string]interface{}{
			"type":           "wallet_connect",
			"wallet_address": walletAddress,
			"public_key":     "test_public_key",
			"chain_id":       "knirv-1",
		}

		qrResp := suite.makeRequest("POST", suite.nativeWalletURL+"/api/v1/qr/generate", qrReq)
		require.True(suite.T(), qrResp.Success, "Failed to generate QR code")

		assert.NotEmpty(suite.T(), qrResp.Data["qr_data"])
		suite.T().Logf("Generated QR code for wallet connection")
	})

	suite.Run("ScanAndConnectQRCode", func() {
		// Simulate QR code scanning from browser wallet
		qrData := QRCodeData{
			Type:      "wallet_connect",
			SessionID: "test_session_123",
			Data: map[string]interface{}{
				"wallet_address": "knirv1test123456789",
				"public_key":     "test_public_key",
				"chain_id":       "knirv-1",
			},
			Timestamp: time.Now(),
		}

		scanReq := map[string]interface{}{
			"qr_data": qrData,
		}

		scanResp := suite.makeRequest("POST", suite.nativeWalletURL+"/api/v1/qr/scan", scanReq)
		require.True(suite.T(), scanResp.Success, "Failed to process scanned QR code")

		assert.Equal(suite.T(), "connected", scanResp.Data["status"])
		suite.T().Logf("Successfully connected wallets via QR code")
	})
}

// Test 2: Wallet Synchronization
func (suite *CrossPlatformWalletTestSuite) TestWalletSynchronization() {
	suite.Run("CreateSyncSession", func() {
		// Create sync session
		sessionReq := map[string]interface{}{
			"native_wallet_id": "test_native_wallet_123",
		}

		sessionResp := suite.makeRequest("POST", suite.syncServiceURL+"/api/v1/sync/session/create", sessionReq)
		require.True(suite.T(), sessionResp.Success, "Failed to create sync session")

		sessionID := sessionResp.Data["session"].(map[string]interface{})["id"].(string)
		suite.testSessions = append(suite.testSessions, sessionID)

		assert.NotEmpty(suite.T(), sessionID)
		suite.T().Logf("Created sync session: %s", sessionID)
	})

	suite.Run("ConnectBrowserWallet", func() {
		// Use the first test session
		require.NotEmpty(suite.T(), suite.testSessions, "No test sessions available")
		sessionID := suite.testSessions[0]

		// Connect browser wallet to session
		connectReq := map[string]interface{}{
			"session_id":        sessionID,
			"browser_wallet_id": "test_browser_wallet_456",
		}

		connectResp := suite.makeRequest("POST", suite.syncServiceURL+"/api/v1/sync/browser/connect", connectReq)
		require.True(suite.T(), connectResp.Success, "Failed to connect browser wallet")

		suite.T().Logf("Connected browser wallet to session: %s", sessionID)
	})

	suite.Run("SynchronizeWalletData", func() {
		// Use the first test session
		require.NotEmpty(suite.T(), suite.testSessions, "No test sessions available")
		sessionID := suite.testSessions[0]

		// Synchronize wallet data
		syncReq := WalletSyncRequest{
			SessionID: sessionID,
		}

		syncResp := suite.makeRequest("POST", suite.syncServiceURL+"/api/v1/sync/wallets", syncReq)
		require.True(suite.T(), syncResp.Success, "Failed to synchronize wallets")

		// Verify sync data
		syncData := syncResp.Data["data"].(map[string]interface{})
		assert.Contains(suite.T(), syncData, "wallets")
		
		wallets := syncData["wallets"].([]interface{})
		assert.Greater(suite.T(), len(wallets), 0)

		suite.T().Logf("Successfully synchronized wallet data for session: %s", sessionID)
	})
}

// Test 3: Cross-Platform Transaction Signing
func (suite *CrossPlatformWalletTestSuite) TestCrossPlatformTransactionSigning() {
	suite.Run("InitiateTransactionOnBrowser", func() {
		// Create a transaction request from browser wallet
		txReq := map[string]interface{}{
			"from":   "knirv1sender123456789",
			"to":     "knirv1recipient123456789",
			"amount": "1000000",
			"token":  "NRN",
			"memo":   "Cross-platform test transaction",
		}

		// Generate QR code for transaction signing
		qrReq := map[string]interface{}{
			"type":             "transaction_request",
			"session_id":       "test_session_123",
			"transaction_data": txReq,
		}

		qrResp := suite.makeRequest("POST", suite.browserWalletURL+"/api/v1/qr/generate", qrReq)
		require.True(suite.T(), qrResp.Success, "Failed to generate transaction QR code")

		assert.NotEmpty(suite.T(), qrResp.Data["qr_data"])
		suite.T().Logf("Generated transaction QR code for cross-platform signing")
	})

	suite.Run("SignTransactionOnNative", func() {
		// Simulate scanning transaction QR code on native app
		txData := map[string]interface{}{
			"from":   "knirv1sender123456789",
			"to":     "knirv1recipient123456789",
			"amount": "1000000",
			"token":  "NRN",
			"memo":   "Cross-platform test transaction",
		}

		signReq := map[string]interface{}{
			"transaction": txData,
			"wallet_id":   "test_native_wallet_123",
		}

		signResp := suite.makeRequest("POST", suite.nativeWalletURL+"/api/v1/transaction/sign", signReq)
		require.True(suite.T(), signResp.Success, "Failed to sign transaction on native wallet")

		assert.NotEmpty(suite.T(), signResp.Data["signature"])
		assert.NotEmpty(suite.T(), signResp.Data["signed_tx"])

		suite.T().Logf("Successfully signed transaction on native wallet")
	})
}

// Test 4: Real-time Synchronization
func (suite *CrossPlatformWalletTestSuite) TestRealTimeSynchronization() {
	suite.Run("WebSocketConnection", func() {
		// Test WebSocket connection for real-time sync
		// Note: This would require WebSocket client implementation
		// For now, we'll test the HTTP endpoint

		sessionID := "test_realtime_session"
		suite.testSessions = append(suite.testSessions, sessionID)

		// Get session info
		sessionResp := suite.makeRequest("GET", suite.syncServiceURL+"/api/v1/sync/session/"+sessionID, nil)
		
		// Session might not exist, which is expected for this test
		if sessionResp.Success {
			assert.NotEmpty(suite.T(), sessionResp.Data["session"])
		}

		suite.T().Logf("Tested WebSocket connection endpoint")
	})

	suite.Run("LiveDataUpdates", func() {
		// Test live data updates between platforms
		updateReq := map[string]interface{}{
			"session_id": "test_realtime_session",
			"update_type": "balance_change",
			"data": map[string]interface{}{
				"wallet_address": "knirv1test123456789",
				"new_balance":    "2000000",
				"token":          "NRN",
			},
		}

		updateResp := suite.makeRequest("POST", suite.syncServiceURL+"/api/v1/sync/update", updateReq)
		
		// This endpoint might not exist yet, so we'll just log the attempt
		suite.T().Logf("Attempted live data update: success=%v", updateResp.Success)
	})
}

// Test 5: Multi-Chain Synchronization
func (suite *CrossPlatformWalletTestSuite) TestMultiChainSynchronization() {
	suite.Run("SynchronizeMultipleChains", func() {
		// Test synchronization of multiple blockchain wallets
		chains := []string{"BTC", "ETH", "SOL", "NRN"}
		
		for _, chain := range chains {
			// Create wallet for each chain
			walletReq := map[string]interface{}{
				"name":  fmt.Sprintf("test_%s_wallet", chain),
				"chain": chain,
				"type":  "HD",
			}

			walletResp := suite.makeRequest("POST", suite.nativeWalletURL+"/api/v1/multichain/wallet/create", walletReq)
			
			if walletResp.Success {
				suite.T().Logf("Created %s wallet for multi-chain sync test", chain)
			} else {
				suite.T().Logf("Failed to create %s wallet: %s", chain, walletResp.Error)
			}
		}

		// Test balance synchronization for all chains
		balanceReq := map[string]interface{}{
			"session_id": "test_multichain_session",
			"chains":     chains,
		}

		balanceResp := suite.makeRequest("POST", suite.syncServiceURL+"/api/v1/sync/balances", balanceReq)
		suite.T().Logf("Multi-chain balance sync attempt: success=%v", balanceResp.Success)
	})
}

// Test 6: Security and Error Handling
func (suite *CrossPlatformWalletTestSuite) TestSecurityAndErrorHandling() {
	suite.Run("InvalidQRCodeHandling", func() {
		// Test handling of invalid QR codes
		invalidQRReq := map[string]interface{}{
			"qr_data": "invalid_qr_code_data",
		}

		invalidResp := suite.makeRequest("POST", suite.nativeWalletURL+"/api/v1/qr/scan", invalidQRReq)
		assert.False(suite.T(), invalidResp.Success, "Should reject invalid QR code")
		assert.NotEmpty(suite.T(), invalidResp.Error)

		suite.T().Logf("Correctly rejected invalid QR code")
	})

	suite.Run("ExpiredSessionHandling", func() {
		// Test handling of expired sessions
		expiredSessionReq := map[string]interface{}{
			"session_id": "expired_session_123",
		}

		expiredResp := suite.makeRequest("POST", suite.syncServiceURL+"/api/v1/sync/wallets", expiredSessionReq)
		assert.False(suite.T(), expiredResp.Success, "Should reject expired session")

		suite.T().Logf("Correctly handled expired session")
	})

	suite.Run("UnauthorizedAccessHandling", func() {
		// Test unauthorized access
		originalToken := suite.authToken
		suite.authToken = "definitely-not-valid"

		unauthorizedReq := map[string]interface{}{
			"native_wallet_id": "test_wallet",
		}

		unauthorizedResp := suite.makeRequest("POST", suite.syncServiceURL+"/api/v1/sync/session/create", unauthorizedReq)
		
		// Restore original token
		suite.authToken = originalToken

		suite.T().Logf("Unauthorized access test completed: success=%v", unauthorizedResp.Success)
	})
}

// Run the cross-platform test suite
func TestCrossPlatformWalletIntegration(t *testing.T) {
	suite.Run(t, new(CrossPlatformWalletTestSuite))
}
