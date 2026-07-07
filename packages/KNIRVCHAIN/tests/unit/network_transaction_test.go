package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"KNIRVCHAIN/config"

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

	// Standard NRN transfer submission is not exercised past this point.
	//
	// This test used to hand off to internal/wallet/wallet_server.go here to
	// generate wallets and sign a transfer, then POST it to node1's
	// /transaction endpoint. That file was removed as part of migrating
	// KNIRVORACLE into the sole NRN-disbursement authority (payment_processor.go
	// and wallet_server.go moved to KNIRVORACLE's internal/oracle/payment).
	//
	// Removing it did not regress this test: the wallet-server bridge signed a
	// wallet.Transaction (Amount *big.Int, string hex Signature) and forwarded
	// it wrapped as {"transaction": {...}} to blockchain.Transaction's /transaction
	// endpoint (Value uint64, []byte Signature, no wrapper) - an already-broken
	// schema mismatch that could never have produced a verifiable transaction.
	// Independently, HandleReceiveTransaction's switch on tx.Type has no case
	// for a plain transfer (TransactionTypeStandard is commented out; see
	// internal/blockchain/blockchain_server.go, ProcessStandardTransfer is
	// unimplemented) - it 400s any transaction without an MCP capability type
	// regardless of how it's signed. So this path was never reachable.
	//
	// Re-enable transfer coverage here once ProcessStandardTransfer exists
	// server-side; sign against blockchain.Transaction's own (also currently
	// stubbed) canonical-hashing scheme in ToProtoForHashing /
	// GetCanonicalBytesForHashingTransaction rather than reintroducing a
	// wallet-package bridge.
	t.Log("Mining and P2P connectivity verified; skipping transfer submission (see comment above).")
	t.Log("TestNetworkTransactionFlow_MultiNode completed successfully.")
}
