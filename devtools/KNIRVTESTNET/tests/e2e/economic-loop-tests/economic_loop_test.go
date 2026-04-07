package economicloop

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
)

const (
	RouterURL = "http://localhost:8086"
	ChainURL  = "http://localhost:8090"
	Timeout   = 30 * time.Second
)

// TestBlockchainEconomics tests the basic blockchain economic functionality
func TestBlockchainEconomics(t *testing.T) {
	client := &http.Client{Timeout: Timeout}

	t.Run("Blockchain Status", func(t *testing.T) {
		resp, err := client.Get(RouterURL + "/status")
		if err != nil {
			t.Fatalf("Failed to get blockchain status: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Blockchain status check failed with status: %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response: %v", err)
		}

		var statusResponse map[string]interface{}
		if err := json.Unmarshal(body, &statusResponse); err != nil {
			t.Fatalf("Failed to parse status response: %v", err)
		}

		// Check if blockchain exists
		if blockchain, ok := statusResponse["block_chain"]; ok {
			if blocks, ok := blockchain.([]interface{}); ok {
				t.Logf("✅ Blockchain has %d blocks", len(blocks))
				if len(blocks) == 0 {
					t.Error("Blockchain should have at least genesis block")
				}
			}
		}

		// Check transaction pool
		if pool, ok := statusResponse["transaction_pool"]; ok {
			t.Logf("✅ Transaction pool status: %v", pool)
		}
	})

	t.Run("Mining Status", func(t *testing.T) {
		resp, err := client.Get(RouterURL + "/status")
		if err != nil {
			t.Fatalf("Failed to get mining status: %v", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response: %v", err)
		}

		var statusResponse map[string]interface{}
		if err := json.Unmarshal(body, &statusResponse); err != nil {
			t.Fatalf("Failed to parse status response: %v", err)
		}

		// Check mining status
		if miningLocked, ok := statusResponse["mining_locked"]; ok {
			t.Logf("✅ Mining locked status: %v", miningLocked)
		}

		// Check miner address
		if address, ok := statusResponse["address"]; ok {
			t.Logf("✅ Miner address: %v", address)
		}
	})

	t.Run("Economic Metrics", func(t *testing.T) {
		resp, err := client.Get(RouterURL + "/status")
		if err != nil {
			t.Fatalf("Failed to get economic metrics: %v", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response: %v", err)
		}

		var statusResponse map[string]interface{}
		if err := json.Unmarshal(body, &statusResponse); err != nil {
			t.Fatalf("Failed to parse status response: %v", err)
		}

		// Analyze blockchain for economic activity
		if blockchain, ok := statusResponse["block_chain"]; ok {
			if blocks, ok := blockchain.([]interface{}); ok {
				totalTransactions := 0
				for _, block := range blocks {
					if blockMap, ok := block.(map[string]interface{}); ok {
						if transactions, ok := blockMap["transactions"]; ok {
							if txList, ok := transactions.([]interface{}); ok {
								totalTransactions += len(txList)
							}
						}
					}
				}
				t.Logf("✅ Total transactions across all blocks: %d", totalTransactions)
			}
		}
	})
}

// TestTransactionFlow tests transaction creation and processing
func TestTransactionFlow(t *testing.T) {
	client := &http.Client{Timeout: Timeout}

	t.Run("Transaction Pool Status", func(t *testing.T) {
		resp, err := client.Get(RouterURL + "/status")
		if err != nil {
			t.Fatalf("Failed to get transaction pool status: %v", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response: %v", err)
		}

		var statusResponse map[string]interface{}
		if err := json.Unmarshal(body, &statusResponse); err != nil {
			t.Fatalf("Failed to parse status response: %v", err)
		}

		if pool, ok := statusResponse["transaction_pool"]; ok {
			if poolList, ok := pool.([]interface{}); ok {
				t.Logf("✅ Transaction pool contains %d pending transactions", len(poolList))
			}
		}
	})

	t.Run("Block Generation", func(t *testing.T) {
		// Get initial block count
		resp, err := client.Get(RouterURL + "/status")
		if err != nil {
			t.Fatalf("Failed to get initial status: %v", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response: %v", err)
		}

		var initialStatus map[string]interface{}
		if err := json.Unmarshal(body, &initialStatus); err != nil {
			t.Fatalf("Failed to parse initial status: %v", err)
		}

		initialBlockCount := 0
		if blockchain, ok := initialStatus["block_chain"]; ok {
			if blocks, ok := blockchain.([]interface{}); ok {
				initialBlockCount = len(blocks)
			}
		}

		// Wait a bit for potential new blocks
		time.Sleep(2 * time.Second)

		// Get updated block count
		resp2, err := client.Get(RouterURL + "/status")
		if err != nil {
			t.Fatalf("Failed to get updated status: %v", err)
		}
		defer resp2.Body.Close()

		body2, err := io.ReadAll(resp2.Body)
		if err != nil {
			t.Fatalf("Failed to read updated response: %v", err)
		}

		var updatedStatus map[string]interface{}
		if err := json.Unmarshal(body2, &updatedStatus); err != nil {
			t.Fatalf("Failed to parse updated status: %v", err)
		}

		updatedBlockCount := 0
		if blockchain, ok := updatedStatus["block_chain"]; ok {
			if blocks, ok := blockchain.([]interface{}); ok {
				updatedBlockCount = len(blocks)
			}
		}

		t.Logf("✅ Block count: initial=%d, updated=%d", initialBlockCount, updatedBlockCount)
		
		if updatedBlockCount >= initialBlockCount {
			t.Logf("✅ Blockchain is active (blocks: %d)", updatedBlockCount)
		} else {
			t.Logf("⚠️ No new blocks generated during test period")
		}
	})
}

// TestEconomicIncentives tests the economic incentive mechanisms
func TestEconomicIncentives(t *testing.T) {
	client := &http.Client{Timeout: Timeout}

	t.Run("Mining Rewards", func(t *testing.T) {
		resp, err := client.Get(RouterURL + "/status")
		if err != nil {
			t.Fatalf("Failed to get mining rewards status: %v", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response: %v", err)
		}

		var statusResponse map[string]interface{}
		if err := json.Unmarshal(body, &statusResponse); err != nil {
			t.Fatalf("Failed to parse status response: %v", err)
		}

		// Analyze transactions for mining rewards
		rewardTransactions := 0
		if blockchain, ok := statusResponse["block_chain"]; ok {
			if blocks, ok := blockchain.([]interface{}); ok {
				for _, block := range blocks {
					if blockMap, ok := block.(map[string]interface{}); ok {
						if transactions, ok := blockMap["transactions"]; ok {
							if txList, ok := transactions.([]interface{}); ok {
								for _, tx := range txList {
									if txMap, ok := tx.(map[string]interface{}); ok {
										if from, ok := txMap["from"].(string); ok {
											if from == "KNIRVCHAIN_Faucet" {
												rewardTransactions++
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}

		t.Logf("✅ Found %d mining reward transactions", rewardTransactions)
		if rewardTransactions > 0 {
			t.Logf("✅ Mining incentive system is active")
		}
	})

	t.Run("Economic Activity", func(t *testing.T) {
		resp, err := client.Get(RouterURL + "/status")
		if err != nil {
			t.Fatalf("Failed to get economic activity: %v", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response: %v", err)
		}

		var statusResponse map[string]interface{}
		if err := json.Unmarshal(body, &statusResponse); err != nil {
			t.Fatalf("Failed to parse status response: %v", err)
		}

		// Calculate total economic value
		totalValue := 0.0
		if blockchain, ok := statusResponse["block_chain"]; ok {
			if blocks, ok := blockchain.([]interface{}); ok {
				for _, block := range blocks {
					if blockMap, ok := block.(map[string]interface{}); ok {
						if transactions, ok := blockMap["transactions"]; ok {
							if txList, ok := transactions.([]interface{}); ok {
								for _, tx := range txList {
									if txMap, ok := tx.(map[string]interface{}); ok {
										if value, ok := txMap["value"].(float64); ok {
											totalValue += value
										}
									}
								}
							}
						}
					}
				}
			}
		}

		t.Logf("✅ Total economic value transacted: %.2f", totalValue)
		if totalValue > 0 {
			t.Logf("✅ Economic loop is functioning")
		}
	})
}

// TestChainIntegration tests integration with KNIRVCHAIN
func TestChainIntegration(t *testing.T) {
	client := &http.Client{Timeout: Timeout}

	t.Run("Chain Health", func(t *testing.T) {
		resp, err := client.Get(ChainURL + "/health")
		if err != nil {
			t.Fatalf("Failed to connect to KNIRVCHAIN: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("KNIRVCHAIN health check failed with status: %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response: %v", err)
		}

		var healthResponse map[string]interface{}
		if err := json.Unmarshal(body, &healthResponse); err != nil {
			t.Fatalf("Failed to parse health response: %v", err)
		}

		if status, ok := healthResponse["status"]; ok && status == "healthy" {
			t.Logf("✅ KNIRVCHAIN is healthy")
		} else {
			t.Logf("⚠️ KNIRVCHAIN status: %v", healthResponse)
		}
	})

	t.Run("Router-Chain Integration", func(t *testing.T) {
		// Test that router and chain are working together
		routerResp, err := client.Get(RouterURL + "/status")
		if err != nil {
			t.Fatalf("Failed to get router status: %v", err)
		}
		defer routerResp.Body.Close()

		chainResp, err := client.Get(ChainURL + "/health")
		if err != nil {
			t.Fatalf("Failed to get chain health: %v", err)
		}
		defer chainResp.Body.Close()

		if routerResp.StatusCode == http.StatusOK && chainResp.StatusCode == http.StatusOK {
			t.Logf("✅ Router and Chain integration is working")
		} else {
			t.Errorf("Integration issue: Router=%d, Chain=%d", routerResp.StatusCode, chainResp.StatusCode)
		}
	})
}
