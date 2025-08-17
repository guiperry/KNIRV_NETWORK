package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"

	agentlog "KNIRVORACLE/log"
	"KNIRVORACLE/types"
)

const (
	// Configuration constants for transaction delegation
	DelegationScanInterval = 10 * time.Second
	MaxSubpoolStaleTime    = 5 * time.Minute
	MaxPapSubpoolQueue     = 100 // Maximum number of transactions in a PAP's subpool
)

// StartTransactionDelegator starts the transaction delegator goroutine
func StartTransactionDelegator(bc *BlockchainStruct, tpm *TransactionPoolManager, discoveryMgr *DiscoveryManager, nodeWallet *Wallet) {
	// Only start if PoAu-D is enabled
	if !bc.PoAuDEnabled {
		agentlog.LogInfo("PoAu-D is disabled, not starting Transaction Delegator")
		return
	}

	agentlog.LogInfo("Starting Transaction Delegator with PoAu-D enabled")

	go func() {
		ticker := time.NewTicker(DelegationScanInterval)
		defer ticker.Stop()

		for range ticker.C {
			// Skip if PoAu-D was disabled
			if !bc.PoAuDEnabled {
				continue
			}

			// Get transactions from the main pool
			bc.Lock()
			mainPoolTxs := make([]*Transaction, len(bc.TransactionPool))
			copy(mainPoolTxs, bc.TransactionPool)
			bc.Unlock()

			delegatedCount := 0
			for _, tx := range mainPoolTxs {
				// Only delegate MCPInvokeCapability transactions
				if tx.Type == TransactionTypeMCPInvokeCapability && !tpm.IsDelegated(tx.TransactionHash) {
					// Perform delegation logic
					err := DelegateTransaction(tx, bc, tpm, discoveryMgr, nodeWallet)
					if err != nil {
						agentlog.LogInfo(fmt.Sprintf("Delegation failed for tx %s: %v", tx.TransactionHash, err))
						// Tx remains in main pool for fallback via PoW
					} else {
						// If delegation was successful, mark as delegated
						tpm.MarkAsDelegated(tx.TransactionHash)
						delegatedCount++
					}
				}
				// Other transaction types stay in main pool
			}

			if delegatedCount > 0 {
				agentlog.LogInfo(fmt.Sprintf("Delegated %d transactions in this cycle", delegatedCount))
			}

			// Reclaim stale delegated transactions
			tpm.ReclaimStaleTransactions(MaxSubpoolStaleTime)
		}
	}()
}

// DelegateTransaction attempts to delegate an MCPInvokeCapability transaction to its owner
func DelegateTransaction(tx *Transaction, bc *BlockchainStruct, tpm *TransactionPoolManager, discoveryMgr *DiscoveryManager, nodeWallet *Wallet) error {
	// 1. Extract capability ID from transaction data
	// This will need to be adapted based on how MCPInvokeCapability transactions store their data
	var mcpInvokeData types.MCPInvokeCapabilityData
	if err := json.Unmarshal(tx.Data, &mcpInvokeData); err != nil {
		return fmt.Errorf("failed to deserialize MCPInvokeCapabilityData: %w", err)
	}

	capabilityID := mcpInvokeData.ContextRecord.CapabilityID
	if capabilityID == "" {
		return fmt.Errorf("context record missing capability ID")
	}

	// 2. Get capability owner (PAP address)
	capDesc, err := bc.mcpProcessor.GetCapabilityDescriptor(capabilityID)
	if err != nil {
		return fmt.Errorf("capability %s not found: %w", capabilityID, err)
	}

	// Extract owner from the capability descriptor
	var papAddress string
	switch desc := capDesc.(type) {
	case types.ResourceDescriptor:
		papAddress = desc.BaseDescriptor.Owner
	case types.ToolDescriptor:
		papAddress = desc.BaseDescriptor.Owner
	case types.PromptDescriptor:
		papAddress = desc.BaseDescriptor.Owner
	case types.MemoryServiceDescriptor:
		papAddress = desc.BaseDescriptor.Owner
	default:
		return fmt.Errorf("unknown capability descriptor type for capability %s", capabilityID)
	}

	if papAddress == "" {
		return fmt.Errorf("capability %s has no owner", capabilityID)
	}

	// 3. If this node IS the PAP, move to local subpool directly
	if nodeWallet != nil && nodeWallet.GetAddress() == papAddress {
		tpm.AddToPASPool(tx)
		agentlog.LogInfo(fmt.Sprintf("Delegated transaction %s to local PASPool", tx.TransactionHash))
		return nil // Successfully delegated to self
	}

	// 4. Check PAP availability & capacity
	papChainID, err := getChainIDForAddress(papAddress, discoveryMgr)
	if err != nil {
		return fmt.Errorf("failed to get ChainID for PAP address %s: %w", papAddress, err)
	}

	statusURI := fmt.Sprintf("agent://%s.chain/status", papChainID)
	status, err := PingPAPStatus(statusURI, discoveryMgr)
	if err != nil {
		return fmt.Errorf("failed to ping PAP status at %s: %w", statusURI, err)
	}

	if status.Status != "ONLINE" || status.SubpoolQueueLength > MaxPapSubpoolQueue {
		return fmt.Errorf("PAP %s status is %s or busy (queue %d)", papAddress, status.Status, status.SubpoolQueueLength)
	}

	// 5. Delegate transaction via P2P
	// TODO: Implement proper peer discovery for PAP
	papPeerInfo := peer.AddrInfo{} // Placeholder for now
	_ = papPeerInfo                // Avoid unused variable error

	err = SendDelegatedTransaction(tx, papPeerInfo, discoveryMgr.host)
	if err != nil {
		return fmt.Errorf("failed to send delegated transaction to PAP %s: %w", papAddress, err)
	}

	agentlog.LogInfo(fmt.Sprintf("Successfully delegated transaction %s to PAP %s", tx.TransactionHash, papAddress))
	return nil
}

// Helper functions for delegation

// getChainIDForAddress maps wallet address to chain ID
func getChainIDForAddress(address string, discoveryMgr *DiscoveryManager) (string, error) {
	// Implementation to map wallet address to chain ID
	// This could use DHT lookups or other discovery mechanisms
	// For now, return a placeholder implementation
	return address, nil // Simplified mapping for initial implementation
}

// PAPStatus represents the status of a Plugin Author Peer
type PAPStatus struct {
	Status             string `json:"status"` // ONLINE, BUSY, OFFLINE
	SubpoolQueueLength int    `json:"subpoolQueueLength"`
}

// PingPAPStatus fetches PAP status via URI resolver
func PingPAPStatus(statusURI string, discoveryMgr *DiscoveryManager) (*PAPStatus, error) {
	// Implementation to fetch PAP status via URI resolver
	// For now, return a placeholder implementation
	return &PAPStatus{Status: "ONLINE", SubpoolQueueLength: 0}, nil
}

// SendDelegatedTransaction sends transaction via libp2p
func SendDelegatedTransaction(tx *Transaction, peerInfo peer.AddrInfo, host host.Host) error {
	// Implementation to send transaction via libp2p
	// For now, return a placeholder implementation
	agentlog.LogInfo(fmt.Sprintf("Sending delegated transaction %s to peer %s", tx.TransactionHash, peerInfo.ID.String()))
	return nil
}

// StopTransactionDelegator stops the transaction delegator (placeholder for future use)
func StopTransactionDelegator() {
	agentlog.LogInfo("Transaction Delegator stopped")
}

// GetDelegationStats returns statistics about the delegation process
func GetDelegationStats(tpm *TransactionPoolManager) map[string]interface{} {
	stats := tpm.GetPoolStats()
	stats["delegation_scan_interval"] = DelegationScanInterval.String()
	stats["max_subpool_stale_time"] = MaxSubpoolStaleTime.String()
	stats["max_pap_subpool_queue"] = MaxPapSubpoolQueue
	return stats
}

// IsDelegationEnabled checks if PoAu-D delegation is currently enabled
func IsDelegationEnabled(bc *BlockchainStruct) bool {
	bc.Lock()
	defer bc.Unlock()
	return bc.PoAuDEnabled
}

// EnableDelegation enables PoAu-D delegation
func EnableDelegation(bc *BlockchainStruct) error {
	bc.Lock()
	defer bc.Unlock()

	bc.PoAuDEnabled = true

	// Save to database
	if bc.db != nil {
		if err := bc.db.PutPoAuDEnabled(true); err != nil {
			bc.PoAuDEnabled = false // Rollback on error
			return fmt.Errorf("failed to save PoAu-D enabled state: %w", err)
		}
	}

	agentlog.LogInfo("PoAu-D delegation enabled")
	return nil
}

// DisableDelegation disables PoAu-D delegation
func DisableDelegation(bc *BlockchainStruct) error {
	bc.Lock()
	defer bc.Unlock()

	bc.PoAuDEnabled = false

	// Save to database
	if bc.db != nil {
		if err := bc.db.PutPoAuDEnabled(false); err != nil {
			bc.PoAuDEnabled = true // Rollback on error
			return fmt.Errorf("failed to save PoAu-D disabled state: %w", err)
		}
	}

	agentlog.LogInfo("PoAu-D delegation disabled")
	return nil
}

// ValidateTransactionForDelegation checks if a transaction is eligible for delegation
func ValidateTransactionForDelegation(tx *Transaction) bool {
	// Only MCPInvokeCapability transactions can be delegated
	if tx.Type != TransactionTypeMCPInvokeCapability {
		return false
	}

	// Check if transaction data is valid
	var mcpInvokeData types.MCPInvokeCapabilityData
	if err := json.Unmarshal(tx.Data, &mcpInvokeData); err != nil {
		return false
	}

	// Must have a capability ID
	if mcpInvokeData.ContextRecord.CapabilityID == "" {
		return false
	}

	return true
}
