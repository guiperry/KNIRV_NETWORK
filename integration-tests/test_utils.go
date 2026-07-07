package integration_tests

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// IntegrationTestSuite holds common utilities and configurations for integration tests.
type IntegrationTestSuite struct {
	knirvchainURL            string
	knirvgraphURL            string
	knirvserverFrontendURL   string
	knirvserverAPIGatewayURL string
	knirvserverDVEManagerURL string
	knirvserverValidationURL string
	knirvwalletURL           string
	knirvshellURL            string
	knirvroterURL            string
	knirvGatewayAPIURL       string // Renamed from knirvRootURL (old oracle/web gateway)
	knirvOracledRPCURL       string // New field for the activated knirv-oracled blockchain RPC
	xionRPC                  string
	testWallet               *TestWallet
}

// TestWallet represents a simple wallet structure for testing purposes.
type TestWallet struct {
	Address  string `json:"address"`
	Mnemonic string `json:"mnemonic"`
	Balance  string `json:"balance"`
	Type     string `json:"type,omitempty"`
}

// NewIntegrationTestSuite creates a new IntegrationTestSuite instance with default URLs.
func NewIntegrationTestSuite() *IntegrationTestSuite {
	return &IntegrationTestSuite{
		knirvchainURL:            "http://localhost:8080",
		knirvgraphURL:            "http://localhost:8081",
		knirvserverFrontendURL:   "http://localhost:3000",  // KNIRVSERVER Frontend (Next.js)
		knirvserverAPIGatewayURL: "http://localhost:8080",  // KNIRVSERVER API Gateway
		knirvserverDVEManagerURL: "http://localhost:8081",  // KNIRVSERVER DVE Manager
		knirvserverValidationURL: "http://localhost:8082",  // KNIRVSERVER Validation Core
		knirvwalletURL:           "http://localhost:8083",  // KNIRVWALLET
		knirvshellURL:            "http://localhost:8084",  // KNIRVCORTEX
		knirvroterURL:            "http://localhost:8085",  // KNIRVROUTER
		knirvGatewayAPIURL:       "http://localhost:8086",  // KNIRVORACLE (now web gateway)
		knirvOracledRPCURL:       "http://localhost:26657", // Activated knirv-oracled blockchain RPC
		xionRPC:                  "https://rpc.xion-testnet-1.burnt.com:443",
	}
}

// SetupTest initializes common test resources, like creating and funding a test wallet.
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

// createTestWallet creates a new test wallet using the KNIRVWALLET service.
func (suite *IntegrationTestSuite) createTestWallet() (*TestWallet, error) {
	resp, err := suite.makeRequest("POST", suite.knirvwalletURL+"/wallet/create", map[string]interface{}{
		"name": "integration_test_wallet",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create test wallet: %w", err)
	}

	var wallet TestWallet
	if err := json.Unmarshal(resp, &wallet); err != nil {
		return nil, fmt.Errorf("failed to unmarshal wallet response: %w", err)
	}
	return &wallet, nil
}

// fundTestWallet funds the test wallet with NRN tokens using the KNIRVCHAIN faucet.
func (suite *IntegrationTestSuite) fundTestWallet() error {
	fundData := map[string]interface{}{
		"address": suite.testWallet.Address,
		"amount":  "10000000", // 10 NRN
	}

	resp, err := suite.makeRequest("POST", suite.knirvchainURL+"/faucet/fund", fundData)
	if err != nil {
		return fmt.Errorf("failed to fund test wallet: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("failed to unmarshal fund response: %w", err)
	}
	// Assuming fund endpoint returns success: true or similar
	if success, ok := result["success"].(bool); !ok || !success {
		return fmt.Errorf("faucet funding not successful: %v", result)
	}
	return nil
}

// makeRequest is a helper to make HTTP requests to services.
func (suite *IntegrationTestSuite) makeRequest(method, url string, data interface{}) ([]byte, error) {
	var body io.Reader
	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request data: %w", err)
		}
		body = strings.NewReader(string(jsonData))
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	if data != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make HTTP request to %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		responseBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP error %d for %s: %s", resp.StatusCode, url, string(responseBody))
	}

	return io.ReadAll(resp.Body)
}

// CheckServiceHealth pings a given URL to ensure the service is running.
func (suite *IntegrationTestSuite) CheckServiceHealth(t *testing.T, serviceName, url, healthEndpoint string) {
	fullURL := url + healthEndpoint
	client := &http.Client{Timeout: 5 * time.Second}

	for i := 0; i < 10; i++ { // Retry a few times
		resp, err := client.Get(fullURL)
		if err == nil && resp.StatusCode == http.StatusOK {
			t.Logf("✓ %s service at %s is healthy", serviceName, url)
			resp.Body.Close()
			return
		}
		t.Logf("Waiting for %s service at %s... (Attempt %d)", serviceName, url, i+1)
		time.Sleep(2 * time.Second)
		if resp != nil {
			resp.Body.Close()
		}
	}
	require.Failf(t, "Service not healthy", "%s service at %s did not become healthy after multiple attempts", serviceName, url)
}

// OracleStatus represents the structure of the status response from knirv-oracled.
// This struct should match the output of KNIRVORACLE/internal/oracle/oracle.go's GetStatus() method.
type OracleStatus struct {
	ChainID   string `json:"chain_id"`
	NetworkID string `json:"network_id"`
	Token     struct {
		Name            string `json:"name"`
		Symbol          string `json:"symbol"`
		TotalSupply     string `json:"total_supply"`
		MaxSupply       string `json:"max_supply"`
		ContractAddress string `json:"contract_address"`
	} `json:"token"`
	Consensus struct {
		LatestBlockHeight int64 `json:"latest_block_height"`
		ValidatorCount    int   `json:"validator_count"`
	} `json:"consensus"`
	Governance map[string]interface{} `json:"governance"` // Assuming this is dynamic
	Economics  map[string]interface{} `json:"economics"`  // Assuming this is dynamic
	P2P        map[string]interface{} `json:"p2p"`        // Assuming this is dynamic
	IBC        struct {
		Enabled bool `json:"enabled"`
	} `json:"ibc"`
}

// GetOracledStatus fetches the current status of the knirv-oracled node.
func (suite *IntegrationTestSuite) GetOracledStatus(t *testing.T) (*OracleStatus, error) {
	url := suite.knirvOracledRPCURL + "/status"
	client := &http.Client{Timeout: 10 * time.Second} // 10-second timeout for RPC calls

	req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for oracled status: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request to oracled RPC at %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("oracled RPC at %s returned non-200 status: %d - %s", url, resp.StatusCode, string(bodyBytes))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body from oracled RPC: %w", err)
	}

	var status OracleStatus
	if err := json.Unmarshal(bodyBytes, &status); err != nil {
		return nil, fmt.Errorf("failed to unmarshal oracled status response: %w", err)
	}

	return &status, nil
}
