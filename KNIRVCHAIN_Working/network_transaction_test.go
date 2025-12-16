package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"KNIRVCHAIN/config"
	"KNIRVCHAIN/utils"

	"path/filepath"
	"testing"
	"time"
)

// Helper function to wait for a node to become healthy
// Helper function to get the blockchain from a node
func getChainFromNode(client *http.Client, nodeURL string) (map[string]interface{}, error) {
	resp, err := client.Get(fmt.Sprintf("%s/chain", nodeURL))
	if err != nil {
		return nil, fmt.Errorf("failed to get chain: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var chain map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&chain); err != nil {
		return nil, fmt.Errorf("failed to decode chain response: %w", err)
	}

	return chain, nil
}

// triggerMining starts mining by connecting the node to another dev
func triggerMining(t *testing.T, sourceNodeURL string, targetNodeURL string) error {
	client := &http.Client{Timeout: 5 * time.Second}

	// Get target node's P2P address
	infoResp, err := client.Get(fmt.Sprintf("%s/info", targetNodeURL))
	if err != nil {
		return fmt.Errorf("failed to get target node info: %w", err)
	}
	defer infoResp.Body.Close()

	if infoResp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", infoResp.StatusCode)
	}

	var nodeInfo struct {
		Multiaddrs []string `json:"multiaddrs"`
	}
	if err := json.NewDecoder(infoResp.Body).Decode(&nodeInfo); err != nil {
		return fmt.Errorf("failed to decode info response: %w", err)
	}

	if len(nodeInfo.Multiaddrs) == 0 {
		return fmt.Errorf("target node info did not contain multiaddrs")
	}

	// Connect source node to target node to trigger mining
	devAddr := nodeInfo.Multiaddrs[0]
	t.Logf("Attempting to connect %s to %s with multiaddr: %s", sourceNodeURL, targetNodeURL, devAddr)

	connectURL := fmt.Sprintf("%s/p2p/connect?dev=%s", sourceNodeURL, devAddr)
	t.Logf("Making connection request to: %s", connectURL)

	req, err := http.NewRequest("POST", connectURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	connectResp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect nodes: %w", err)
	}
	defer connectResp.Body.Close()

	if connectResp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", connectResp.StatusCode)
	}

	return nil
}

// verifyMiningStatus checks if mining is active by verifying block production
func verifyMiningStatus(t *testing.T, nodeURL string, expectedActive bool) error {
	if !expectedActive {
		t.Log("Mining expected to be inactive - skipping verification")
		return nil // No verification needed if mining shouldn't be active
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(nodeURL + "/mining/status")
	if err != nil {
		t.Errorf("Failed to get mining status: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Unexpected status code: %d", resp.StatusCode)
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var status struct {
		Active bool `json:"active"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		t.Errorf("Failed to decode mining status: %v", err)
		return err
	}

	if !status.Active {
		t.Error("Mining is not active when expected to be")
		return fmt.Errorf("mining not active")
	}

	t.Log("Mining status API reports active, now verifying block production...")

	// Get initial block count
	initialResp, err := client.Get(fmt.Sprintf("%s/chain", nodeURL))
	if err != nil {
		return fmt.Errorf("failed to get initial chain: %w", err)
	}
	defer initialResp.Body.Close()

	if initialResp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", initialResp.StatusCode)
	}

	var initialChain map[string]interface{}
	if err := json.NewDecoder(initialResp.Body).Decode(&initialChain); err != nil {
		return fmt.Errorf("failed to decode initial chain: %w", err)
	}

	blocks, ok := initialChain["blocks"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid blocks format in chain response")
	}
	initialBlocks := len(blocks)
	t.Logf("Initial block count: %d", initialBlocks)

	// Wait for new block production with longer timeout
	timeout := time.After(60 * time.Second)
	tick := time.Tick(2 * time.Second)

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for block production")
		case <-tick:
			currentResp, err := client.Get(fmt.Sprintf("%s/chain", nodeURL))
			if err != nil {
				t.Logf("Error getting current chain (will retry): %v", err)
				continue // Retry on error
			}
			defer currentResp.Body.Close()

			var currentChain map[string]interface{}
			if err := json.NewDecoder(currentResp.Body).Decode(&currentChain); err != nil {
				t.Logf("Error decoding current chain (will retry): %v", err)
				continue // Retry on decode error
			}

			currentBlocks, ok := currentChain["blocks"].([]interface{})
			if !ok {
				t.Logf("Invalid blocks format in current chain response (will retry)")
				continue
			}

			if len(currentBlocks) > initialBlocks {
				t.Logf("Block production verified: initial=%d, current=%d", initialBlocks, len(currentBlocks))
				return nil // New block found, mining is active
			}

			t.Logf("Waiting for new blocks: initial=%d, current=%d", initialBlocks, len(currentBlocks))
		}
	}
}

// Helper function to check if a transaction exists in the blockchain
func checkTxnInChain(chain map[string]interface{}, expectedTxnHash string) bool {
	blocks, ok := chain["blocks"].([]interface{})
	if !ok {
		return false
	}

	for _, blockInterface := range blocks {
		block, ok := blockInterface.(map[string]interface{})
		if !ok {
			continue
		}

		transactions, ok := block["transactions"].([]interface{})
		if !ok {
			continue
		}

		for _, txnInterface := range transactions {
			txn, ok := txnInterface.(map[string]interface{})
			if !ok {
				continue
			}

			txnHash, hashOk := txn["transaction_hash"].(string) // Check the hash field

			if hashOk && txnHash == expectedTxnHash {
				return true
			}
		}
	}
	return false
}

// TestWalletResponse represents the test response from the wallet server
type TestWalletResponse struct {
	Address    string `json:"address"`
	PrivateKey string `json:"private_key"`
	PublicKey  string `json:"public_key"`
}

// TestTransactionRequest represents transaction requests in tests
type TestTransactionRequest struct {
	FromAddress string `json:"from_address"`
	ToAddress   string `json:"to_address"`
	Value       uint64 `json:"value"`
	PrivateKey  string `json:"private_key"`
	PublicKey   string `json:"public_key"`
}

// nodeServerInfo matches the ServerInfo structure from blockchain_server.go
type nodeServerInfo struct {
	HTTPPort    uint64   `json:"http_port"`
	P2PPort     int      `json:"p2p_port"`
	ChainID     string   `json:"chain_id"`
	PeerID      string   `json:"dev_id,omitempty"`
	Multiaddrs  []string `json:"multiaddrs,omitempty"`
	Version     string   `json:"version"`
	Connections int      `json:"connections,omitempty"`
}

func TestNetworkTransactionFlow_MultiNode(t *testing.T) {
	// Test configuration variables
	const (
		configDir1        = "./testdata/node1"
		configDir2        = "./testdata/node2"
		dbPathNode1       = "./testdata/node1/db"
		dbPathNode2       = "./testdata/node2/db"
		node1Port         = 6001
		node2Port         = 6002
		node1P2PPort      = 7001
		node2P2PPort      = 7002
		node1URL          = "http://localhost:6001"
		node2URL          = "http://localhost:6002"
		minerAddr1        = "0x1111111111111111111111111111111111111111"
		minerAddr2        = "0x2222222222222222222222222222222222222222"
		maxRetries        = 5
		retryDelay        = 2 * time.Second
		initialWalletPort = 8000
	)

	// --- Add Cleanup for test directories ---
	t.Logf("Cleaning up test directories: %s and %s", dbPathNode1, dbPathNode2)
	if err := os.RemoveAll(dbPathNode1); err != nil {
		t.Logf("Warning: failed to remove %s: %v", dbPathNode1, err)
	}
	if err := os.RemoveAll(dbPathNode2); err != nil {
		t.Logf("Warning: failed to remove %s: %v", dbPathNode2, err)
	}
	// --- End Cleanup ---

	if err := os.MkdirAll(configDir1, 0755); err != nil {
		t.Fatalf("Failed to create config directory for Node 1: %v", err)
	}
	if err := os.MkdirAll(configDir2, 0755); err != nil {
		t.Fatalf("Failed to create config directory for Node 2: %v", err)
	}

	configPath1 := filepath.Join(configDir1, "config.json")
	configPath2 := filepath.Join(configDir2, "config.json")

	// Create config for Node 1
	node1Config := map[string]interface{}{
		"role":                 "Peer",
		"installComplete":      true,
		"port":                 node1Port,
		"httpPort":             node1Port, // Changed from "HTTPPort" to "httpPort" to match mapstructure tag
		"minersAddress":        minerAddr1,
		"p2p_port":             node1P2PPort,
		"p2pPort":              node1P2PPort, // Changed from "P2PPort" to "p2pPort" to match mapstructure tag
		"chainID":              "test-network-chain",
		"IsPeer":               true,
		"clientOnly":           false,
		"mining_enabled":       true, // Explicitly enable mining
		"consensus_pause_time": 1,    // Faster consensus for tests
		// Explicitly set to false to prevent default settings from overriding
		"IsBootnode": false,
		"IsRoot":     false,
		"paths": map[string]interface{}{
			"blockchain_database_path": filepath.Join(dbPathNode1, "blockchain.db"),
			"searchable_database_path": filepath.Join(dbPathNode1, "searchable_db"),
			"wallet_path":              filepath.Join(dbPathNode1, "wallet.dat"),
		},
	}

	node1ConfigJSON, err := json.Marshal(node1Config)
	if err != nil {
		t.Fatalf("Failed to marshal config for Node 1: %v", err)
	}

	if err := os.WriteFile(configPath1, node1ConfigJSON, 0644); err != nil {
		t.Fatalf("Failed to write config file for Node 1: %v", err)
	}

	// Create config for Node 2
	node2Config := map[string]interface{}{
		"role":                 "Peer",
		"installComplete":      true,
		"consensus_pause_time": 1, // Faster consensus for tests
		"port":                 node2Port,
		"httpPort":             node2Port, // Changed from "HTTPPort" to "httpPort" to match mapstructure tag
		"minersAddress":        minerAddr2,
		"p2p_port":             node2P2PPort,
		"p2pPort":              node2P2PPort, // Changed from "P2PPort" to "p2pPort" to match mapstructure tag
		"chainID":              "test-network-chain",
		"IsPeer":               true,
		"clientOnly":           false,
		"mining_enabled":       true, // Explicitly enable mining
		// Explicitly set to false to prevent default settings from overriding
		"IsBootnode": false,
		"IsRoot":     false,
		"paths": map[string]interface{}{
			"blockchain_database_path": filepath.Join(dbPathNode2, "blockchain.db"),
			"searchable_database_path": filepath.Join(dbPathNode2, "searchable_db"),
			"wallet_path":              filepath.Join(dbPathNode2, "wallet.dat"),
		},
	}

	node2ConfigJSON, err := json.Marshal(node2Config)
	if err != nil {
		t.Fatalf("Failed to marshal config for Node 2: %v", err)
	}

	if err := os.WriteFile(configPath2, node2ConfigJSON, 0644); err != nil {
		t.Fatalf("Failed to write config file for Node 2: %v", err)
	}

	// Start Node 1
	t.Logf("Starting Node 1 on port %d (P2P: %d)...", node1Port, node1P2PPort)
	t.Logf("Node 1 config path: %s", configPath1)
	node1Server, err := config.StartTestNodeWithDB(node1Port, minerAddr1, dbPathNode1, []string{
		"--config", configPath1,
		"--skip-install",                              // Skip the installer
		"--p2p.port", fmt.Sprintf("%d", node1P2PPort), // Explicitly set P2P port
		"--dev",          // Run as a dev node (enables mining)
		"--role", "Peer", // Explicitly set role to Peer
		"--logLevel", "debug", // Add debug logging
	})
	if err != nil {
		t.Fatalf("Failed to start Node 1: %v", err)
	}
	defer node1Server.Cleanup()

	// Start Node 2
	t.Logf("Starting Node 2 on port %d (P2P: %d)...", node2Port, node2P2PPort)
	t.Logf("Node 2 config path: %s", configPath2)
	node2Server, err := config.StartTestNodeWithDB(node2Port, minerAddr2, dbPathNode2, []string{
		"--config", configPath2,
		"--skip-install",                              // Skip the installer
		"--p2p.port", fmt.Sprintf("%d", node2P2PPort), // Explicitly set P2P port
		"--dev",          // Run as a dev node (enables mining)
		"--role", "Peer", // Explicitly set role to Peer
		"--logLevel", "debug", // Add debug logging
	})
	if err != nil {
		node1Server.Cleanup()
		t.Fatalf("Failed to start Node 2: %v", err)
	}
	defer node2Server.Cleanup()

	// Wait for Node 2 to become healthy before connecting
	t.Logf("Waiting for Node 2 at %s to become healthy...", node2URL)
	config.WaitForNode(t, node2URL, 30*time.Second)

	// Get Node 1's full P2P address from its /info endpoint
	var node1FullP2PAddr string
	// Create HTTP client for all requests
	client := &http.Client{Timeout: 15 * time.Second}
	infoResp, err := client.Get(fmt.Sprintf("%s/info", node1URL))
	if err != nil {
		t.Fatalf("Failed to get /info from Node 1: %v", err)
	}
	defer infoResp.Body.Close()

	if infoResp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(infoResp.Body)
		t.Fatalf("Node 1 /info endpoint returned status %d: %s", infoResp.StatusCode, string(bodyBytes))
	}

	var node1Info nodeServerInfo
	if err := json.NewDecoder(infoResp.Body).Decode(&node1Info); err != nil {
		t.Fatalf("Failed to decode /info response from Node 1: %v", err)
	}

	if len(node1Info.Multiaddrs) == 0 {
		t.Fatalf("Node 1 /info response did not contain multiaddrs")
	}

	// Find a suitable IPv4 TCP multiaddress from the list
	for _, ma := range node1Info.Multiaddrs {
		if strings.Contains(ma, "/ip4/") && strings.Contains(ma, "/tcp/") {
			node1FullP2PAddr = ma // This will be like /ip4/127.0.0.1/tcp/6000/p2p/QmPeerID
			break
		}
	}

	if node1FullP2PAddr == "" {
		t.Fatalf("Could not find a suitable IPv4 TCP multiaddress for Node 1 from /info: %v", node1Info.Multiaddrs)
	}

	// Connect nodes to each other explicitly with retries
	t.Logf("Connecting Node 2 to Node 1 at P2P address %s...", node1FullP2PAddr)
	var connectResp *http.Response

	for i := 0; i < maxRetries; i++ {
		connectResp, err = http.Post(
			fmt.Sprintf("%s/p2p/connect?dev=%s", node2URL, node1FullP2PAddr),
			"application/json",
			nil,
		)
		if err == nil {
			break
		}
		t.Logf("Connection attempt %d/%d failed: %v", i+1, maxRetries, err)
		if i < maxRetries-1 {
			time.Sleep(retryDelay)
		}
	}

	// Explicitly trigger mining on both nodes
	t.Log("Triggering mining by connecting nodes...")
	// Connect Node1 to Node2
	if err := triggerMining(t, node1URL, node2URL); err != nil {
		t.Fatalf("Failed to connect Node1 to Node2: %v", err)
	}
	// Connect Node2 to Node1
	if err := triggerMining(t, node2URL, node1URL); err != nil {
		t.Fatalf("Failed to connect Node2 to Node1: %v", err)
	}

	// Explicitly start mining on both nodes
	t.Log("Starting mining on both nodes...")
	node1Server.Blockchain.StartMining()
	node2Server.Blockchain.StartMining()

	// Verify mining is active on both nodes
	t.Log("Verifying mining status on nodes...")
	if err := verifyMiningStatus(t, node1URL, true); err != nil {
		t.Fatalf("Node 1 mining not active: %v", err)
	}
	if err := verifyMiningStatus(t, node2URL, true); err != nil {
		t.Fatalf("Node 2 mining not active: %v", err)
	}

	if err != nil {
		t.Fatalf("Failed to connect Node 2 to Node 1 after %d attempts: %v", maxRetries, err)
	}
	connectResp.Body.Close()
	if connectResp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to connect Node 2 to Node 1, status: %d", connectResp.StatusCode)
	}

	// Start Wallet Server (connected to Node 1)
	t.Logf("Starting Wallet Server, attempting port %d...", initialWalletPort)
	walletServer := NewWalletServer(uint64(initialWalletPort), node1URL) // Connect to node 1
	// walletServer.portChan is already initialized in NewWalletServer
	walletStop := walletServer.Start()
	defer walletStop()

	// --- Port Channel Waiting Logic ---
	var actualWalletPort int
	select {
	case p := <-walletServer.portChan:
		actualWalletPort = int(p)
		if actualWalletPort != initialWalletPort {
			t.Logf("Wallet server running on alternate port %d", actualWalletPort)
		} else {
			t.Logf("Wallet server running on initial port %d", actualWalletPort)
		}
	case <-time.After(15 * time.Second): // Slightly longer timeout for port finding + startup
		t.Fatalf("Timeout waiting for wallet server port signal (tried ports %d-%d)", initialWalletPort, initialWalletPort+10)
	}
	// --- End Port Channel Waiting Logic ---

	// Use the actual port for subsequent requests and waiting
	walletURL := fmt.Sprintf("http://localhost:%d", actualWalletPort)

	// Wait for all components to be healthy
	t.Log("Waiting for nodes and wallet server to become healthy...")
	t.Logf("Waiting for node 1 at %s to become healthy...", node1URL)
	config.WaitForNode(t, node1URL, 30*time.Second)
	t.Logf("Waiting for node 2 at %s to become healthy...", node2URL)
	config.WaitForNode(t, node2URL, 30*time.Second)
	// Wait for the wallet server on its actual port
	t.Logf("Waiting for wallet server at %s to become healthy...", walletURL)
	config.WaitForNode(t, walletURL, 20*time.Second)

	t.Log("All components are healthy and ready for testing.")

	// --- Test Execution Phase ---
	// client is already defined above

	// Generate wallets via Wallet Server API (Use walletURL)
	t.Log("Generating wallets...")
	senderResp, err := client.Get(fmt.Sprintf("%s/generate_wallet", walletURL)) // Use walletURL
	if err != nil {
		t.Fatalf("Failed to generate sender wallet: %v", err)
	}
	// ... (rest of sender wallet handling, checking status code, decoding) ...
	defer senderResp.Body.Close()
	if senderResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(senderResp.Body)
		t.Fatalf("Failed to generate sender wallet, status: %d, body: %s", senderResp.StatusCode, string(body))
	}
	var senderWallet TestWalletResponse
	if err := json.NewDecoder(senderResp.Body).Decode(&senderWallet); err != nil {
		t.Fatalf("Failed to decode sender wallet response: %v", err)
	}

	receiverResp, err := client.Get(fmt.Sprintf("%s/generate_wallet", walletURL)) // Use walletURL
	if err != nil {
		t.Fatalf("Failed to generate receiver wallet: %v", err)
	}
	// ... (rest of receiver wallet handling, checking status code, decoding) ...
	defer receiverResp.Body.Close()
	if receiverResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(receiverResp.Body)
		t.Fatalf("Failed to generate receiver wallet, status: %d, body: %s", receiverResp.StatusCode, string(body))
	}
	var receiverWallet TestWalletResponse
	if err := json.NewDecoder(receiverResp.Body).Decode(&receiverWallet); err != nil {
		t.Fatalf("Failed to decode receiver wallet response: %v", err)
	}

	t.Logf("Sender: %s, Receiver: %s", senderWallet.Address, receiverWallet.Address)

	// --- Fund Sender Wallet using Faucet ---
	t.Logf("Funding sender wallet %s via faucet...", senderWallet.Address)
	fundingAmount := uint64(10000 * utils.DECIMAL) // e.g., 10000 NRN
	faucetReqBody := map[string]interface{}{
		"address": senderWallet.Address,
		"amount":  fundingAmount,
	}
	faucetReqJSON, _ := json.Marshal(faucetReqBody)
	faucetURL := fmt.Sprintf("%s/test/faucet", node1URL) // Target Node 1's faucet

	faucetResp, err := client.Post(faucetURL, "application/json", bytes.NewBuffer(faucetReqJSON))
	if err != nil {
		t.Fatalf("Failed to send faucet request: %v", err)
	}

	// Read faucet response body to get the transaction hash
	faucetRespBodyBytes, readErr := io.ReadAll(faucetResp.Body)
	faucetResp.Body.Close() // Close body immediately

	if faucetResp.StatusCode != http.StatusCreated {
		bodyStr := string(faucetRespBodyBytes)
		if readErr != nil {
			bodyStr = fmt.Sprintf("(failed to read response body: %v)", readErr)
		}
		t.Fatalf("Faucet request failed, status: %d, body: %s", faucetResp.StatusCode, bodyStr)
	}

	// Decode the faucet transaction from the response body
	var faucetTxnResp map[string]interface{}
	if err := json.Unmarshal(faucetRespBodyBytes, &faucetTxnResp); err != nil {
		t.Fatalf("Failed to decode faucet transaction response: %v, body: %s", err, string(faucetRespBodyBytes))
	}
	faucetTxnHash, hashOk := faucetTxnResp["transaction_hash"].(string)
	if !hashOk || faucetTxnHash == "" {
		t.Fatalf("Could not get transaction_hash from faucet response: %v", faucetTxnResp)
	}
	t.Logf("Faucet funding transaction created: %s. Waiting for it to be mined...", faucetTxnHash)

	// --- Wait for Faucet Transaction to be Mined ---
	faucetMineTimeout := time.After(60 * time.Second) // Increased timeout to 60s
	faucetTick := time.Tick(1 * time.Second)
	faucetFound := false
	t.Logf("Starting mining wait for faucet transaction %s", faucetTxnHash)

	for !faucetFound {
		select {
		case <-faucetMineTimeout:
			t.Fatalf("Timeout waiting for faucet transaction %s to be mined", faucetTxnHash)
		case <-faucetTick:
			// Check chain on Node 1 (where mining reward goes)
			chain1, err1 := getChainFromNode(client, node1URL)
			if err1 == nil && checkTxnInChain(chain1, faucetTxnHash) {
				t.Logf("Faucet transaction %s found mined on Node 1.", faucetTxnHash)
				faucetFound = true
				break
			}
			// Optionally check Node 2 as well
			chain2, err2 := getChainFromNode(client, node2URL)
			if err2 == nil && checkTxnInChain(chain2, faucetTxnHash) {
				t.Logf("Faucet transaction %s found mined on Node 2.", faucetTxnHash)
				faucetFound = true
				break
			}
			if err1 != nil {
				t.Logf("Error checking Node 1 for faucet txn: %v", err1)
			}
			if err2 != nil {
				t.Logf("Error checking Node 2 for faucet txn: %v", err2)
			}
		}
	}
	// --- End Wait for Faucet Mining ---

	// --- End Funding ---

	// Create transaction request
	t.Log("Preparing transaction...")
	txnValue := uint64(500 * utils.DECIMAL) // 500 NRN
	txnRequest := TestTransactionRequest{
		FromAddress: senderWallet.Address,
		ToAddress:   receiverWallet.Address,
		Value:       txnValue,
		PrivateKey:  senderWallet.PrivateKey, // Wallet server needs these to sign
		PublicKey:   senderWallet.PublicKey,
	}
	txnData, err := json.Marshal(txnRequest)
	if err != nil {
		t.Fatalf("Failed to marshal transaction request: %v", err)
	}

	// Submit transaction via Wallet Server API (Use walletURL)
	t.Logf("Submitting transaction (%d units) from %s...", txnValue, senderWallet.Address)
	submitURL := fmt.Sprintf("%s/send_signed_txn", walletURL) // Use walletURL
	resp, err := client.Post(submitURL, "application/json", bytes.NewBuffer(txnData))
	if err != nil {
		t.Fatalf("Failed to submit transaction request: %v", err)
	}
	// Read body *before* checking status for better error messages
	respBodyBytes, readErr := io.ReadAll(resp.Body)
	resp.Body.Close() // Close body immediately after reading

	if resp.StatusCode != http.StatusCreated {
		bodyStr := string(respBodyBytes)
		if readErr != nil {
			bodyStr = fmt.Sprintf("(failed to read response body: %v)", readErr)
		}
		t.Fatalf("Transaction submission failed, status: %d, body: %s", resp.StatusCode, bodyStr)
	}
	t.Log("Transaction submitted successfully.")

	// Decode the signed transaction from the response body
	var signedTxnResp map[string]interface{} // Use map to decode generic JSON
	if err := json.Unmarshal(respBodyBytes, &signedTxnResp); err != nil {
		t.Fatalf("Failed to decode signed transaction response: %v, body: %s", err, string(respBodyBytes))
	}
	expectedHash, hashOk := signedTxnResp["transaction_hash"].(string)
	if !hashOk || expectedHash == "" {
		t.Fatalf("Could not get transaction_hash from wallet server response: %v", signedTxnResp)
	}
	t.Logf("Submitted transaction hash: %s", expectedHash)

	// --- Verification Phase: Wait for Transaction to be Mined ---
	t.Logf("Transaction %s submitted, waiting for it to be mined...", expectedHash)
	// --- Verification Phase ---
	var (
		found       bool
		minedOnNode string
		ticker      = time.NewTicker(2 * time.Second)
		timeout     = time.After(60 * time.Second) // Increased timeout to 60s
	)
	t.Log("Mining verification started with 60s timeout")
	defer ticker.Stop()

	select {
	case <-time.After(30 * time.Second):
		t.Fatal("Test timed out waiting for transaction confirmation")
	case <-ticker.C:
		{
			// Check Node 1's chain
			chain1, err1 := getChainFromNode(client, node1URL)
			if err1 == nil && checkTxnInChain(chain1, expectedHash) {
				t.Logf("Transaction %s found mined on Node 1", expectedHash)
				found = true
				minedOnNode = node1URL
				break // Exit select
			}

			// Check Node 2's chain
			chain2, err2 := getChainFromNode(client, node2URL)
			if err2 == nil && checkTxnInChain(chain2, expectedHash) {
				t.Logf("Transaction %s found mined on Node 2", expectedHash)
				found = true
				minedOnNode = node2URL
				break // Exit select
			}
		}
	}

	for !found {
		select {
		case <-timeout:
			t.Fatalf("Timeout waiting for transaction %s confirmation", expectedHash)
		case <-ticker.C:
			// Check Node 1's chain
			chain1, err1 := getChainFromNode(client, node1URL)
			if err1 == nil && checkTxnInChain(chain1, expectedHash) {
				t.Logf("Transaction %s found mined on Node 1", expectedHash)
				found = true
				minedOnNode = node1URL
				break // Exit select
			}
			// Check Node 2's chain
			chain2, err2 := getChainFromNode(client, node2URL)
			if err2 == nil && checkTxnInChain(chain2, expectedHash) {
				t.Logf("Transaction %s found mined on Node 2", expectedHash)
				found = true
				minedOnNode = node2URL
				break // Exit select
			}

			// Log errors if checks failed during polling
			if err1 != nil {
				t.Logf("Polling Node 1 chain error: %v", err1)
			}
			if err2 != nil {
				t.Logf("Polling Node 2 chain error: %v", err2)
			}
		}

		// --- Optional: Wait for Sync ---
		// If the transaction was mined, wait a bit and check if the other node synced up.
		if found {
			t.Logf("Waiting briefly (%v) for potential sync...", 5*time.Second)
			time.Sleep(5 * time.Second) // Adjust delay as needed

			otherNodeURL := node2URL
			if minedOnNode == node2URL {
				otherNodeURL = node1URL
			}

			finalChainOther, errOther := getChainFromNode(client, otherNodeURL)
			if errOther == nil && checkTxnInChain(finalChainOther, expectedHash) {
				t.Logf("Transaction %s confirmed synced to other node (%s)", expectedHash, otherNodeURL)
			} else {
				// This is not necessarily a failure, consensus might take longer or resolve differently
				t.Logf("Warn: Transaction %s not confirmed synced to other node (%s) after delay. Error: %v", expectedHash, otherNodeURL, errOther)
			}
		}
	}

	select {
	case <-time.After(30 * time.Second):
		t.Fatal("Test timed out waiting for transaction confirmation")
	case <-ticker.C:
		// Check both nodes' chains
		{
			chain1, err1 := getChainFromNode(client, node1URL)
			if err1 == nil && checkTxnInChain(chain1, expectedHash) {
				t.Logf("Transaction %s found mined on Node 1", expectedHash)
				found = true
				minedOnNode = node1URL
				break // Exit select
			}
		}

		chain2, err2 := getChainFromNode(client, node2URL)
		if err2 == nil && checkTxnInChain(chain2, expectedHash) {
			t.Logf("Transaction %s found mined on Node 2", expectedHash)
			found = true
			minedOnNode = node2URL
			break // Exit select
		}

		// Log errors if checks failed (optional)
		// if err1 != nil { t.Logf("Error checking Node 1: %v", err1) }
		// if err2 != nil { t.Logf("Error checking Node 2: %v", err2) }
	}

	// Optional: Add a small delay and check if the other node eventually syncs
	if found {
		t.Logf("Waiting briefly for potential sync...")
		time.Sleep(5 * time.Second) // Adjust as needed
		otherNodeURL := node2URL
		if minedOnNode == node2URL {
			otherNodeURL = node1URL
		}
		chainOther, errOther := getChainFromNode(client, otherNodeURL)
		if errOther == nil && checkTxnInChain(chainOther, expectedHash) {
			t.Logf("Transaction %s confirmed synced to other node (%s)", expectedHash, otherNodeURL)
		} else {
			t.Logf("Warn: Transaction %s not yet synced to other node (%s) after delay. Error: %v", expectedHash, otherNodeURL, errOther)
		}
	}

	t.Log("TestNetworkTransactionFlow_MultiNode completed successfully.")
}
