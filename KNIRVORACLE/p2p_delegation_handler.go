package main

import (
	"bufio"
	"encoding/json"
	"fmt"

	"github.com/libp2p/go-libp2p/core/network"

	agentlog "KNIRVORACLE/log"
	"KNIRVORACLE/types"
)

const DelegationProtocolID = "/agent/delegate-tx/1.0.0"

// Node represents the main node structure (this should match your actual Node struct)
type Node struct {
	Blockchain             *BlockchainStruct
	TransactionPoolManager *TransactionPoolManager
	Wallet                 *Wallet
	Host                   interface{} // This should be the actual libp2p host type
}

// RegisterDelegationHandler registers the delegation protocol handler
func RegisterDelegationHandler(node *Node) {
	// Only register if PoAu-D is enabled
	if !node.Blockchain.PoAuDEnabled {
		agentlog.LogInfo("PoAu-D is disabled, not registering delegation handler")
		return
	}

	// Note: This is a placeholder implementation
	// In the actual implementation, you would set the stream handler on the libp2p host
	// node.Host.SetStreamHandler(DelegationProtocolID, func(stream network.Stream) {
	//     handleDelegatedTransactionStream(stream, node)
	// })

	agentlog.LogInfo("Registered delegation protocol handler")
}

// handleDelegatedTransactionStream processes incoming delegated transactions
func handleDelegatedTransactionStream(stream network.Stream, node *Node) {
	defer stream.Close()

	// Read the transaction from the stream
	buf := bufio.NewReader(stream)
	txBytes, err := buf.ReadBytes('\n')
	if err != nil {
		agentlog.LogError("Failed to read transaction from stream", err)
		return
	}

	// Decode the transaction
	var delegatedTx Transaction
	if err := json.Unmarshal(txBytes, &delegatedTx); err != nil {
		agentlog.LogError("Failed to unmarshal transaction", err)
		return
	}

	agentlog.LogInfo(fmt.Sprintf("Received delegated transaction %s", delegatedTx.TransactionHash))

	// Validate the transaction
	if !delegatedTx.VerifyTxn() {
		agentlog.LogError(fmt.Sprintf("Received invalid delegated transaction %s", delegatedTx.TransactionHash), nil)
		return
	}

	// Verify this is an MCPInvokeCapability transaction
	if delegatedTx.Type != TransactionTypeMCPInvokeCapability {
		agentlog.LogError(fmt.Sprintf("Received non-MCPInvokeCapability transaction %s for delegation", delegatedTx.TransactionHash), nil)
		return
	}

	// Verify this node is the legitimate owner of the capability
	var mcpInvokeData types.MCPInvokeCapabilityData
	if err := json.Unmarshal(delegatedTx.Data, &mcpInvokeData); err != nil {
		agentlog.LogError("Failed to deserialize MCPInvokeCapabilityData", err)
		return
	}

	capID := mcpInvokeData.ContextRecord.CapabilityID
	if capID == "" {
		agentlog.LogError("Missing capability ID in delegated transaction", nil)
		return
	}

	// Get the capability descriptor to verify ownership
	capDesc, err := node.Blockchain.mcpProcessor.GetCapabilityDescriptor(capID)
	if err != nil {
		agentlog.LogError(fmt.Sprintf("Capability %s not found", capID), err)
		return
	}

	// Extract owner from the capability descriptor
	var capOwner string
	switch desc := capDesc.(type) {
	case types.ResourceDescriptor:
		capOwner = desc.BaseDescriptor.Owner
	case types.ToolDescriptor:
		capOwner = desc.BaseDescriptor.Owner
	case types.PromptDescriptor:
		capOwner = desc.BaseDescriptor.Owner
	case types.MemoryServiceDescriptor:
		capOwner = desc.BaseDescriptor.Owner
	default:
		agentlog.LogError(fmt.Sprintf("Unknown capability descriptor type for capability %s", capID), nil)
		return
	}

	// Verify this node owns the capability
	if node.Wallet == nil || node.Wallet.GetAddress() != capOwner {
		agentlog.LogError(fmt.Sprintf("Not the owner of capability %s (owner: %s, this node: %s)",
			capID, capOwner, getNodeAddress(node.Wallet)), nil)
		return
	}

	// Add to local PASPool
	node.TransactionPoolManager.AddToPASPool(&delegatedTx)
	agentlog.LogInfo(fmt.Sprintf("Accepted delegated transaction %s for capability %s into PASPool",
		delegatedTx.TransactionHash, capID))

	// Send acknowledgment
	ack := []byte("OK\n")
	if _, err := stream.Write(ack); err != nil {
		agentlog.LogError("Failed to send acknowledgment", err)
	}
}

// getNodeAddress safely gets the node's wallet address
func getNodeAddress(wallet *Wallet) string {
	if wallet == nil {
		return "unknown"
	}
	return wallet.GetAddress()
}

// ValidateDelegatedTransaction performs additional validation on delegated transactions
func ValidateDelegatedTransaction(tx *Transaction, node *Node) error {
	// Verify transaction type
	if tx.Type != TransactionTypeMCPInvokeCapability {
		return fmt.Errorf("invalid transaction type for delegation: %s", tx.Type)
	}

	// Verify transaction signature
	if !tx.VerifyTxn() {
		return fmt.Errorf("transaction signature verification failed")
	}

	// Parse the MCP invoke data
	var mcpInvokeData types.MCPInvokeCapabilityData
	if err := json.Unmarshal(tx.Data, &mcpInvokeData); err != nil {
		return fmt.Errorf("failed to parse MCPInvokeCapabilityData: %w", err)
	}

	// Verify capability ID exists
	capID := mcpInvokeData.ContextRecord.CapabilityID
	if capID == "" {
		return fmt.Errorf("missing capability ID")
	}

	// Verify the capability exists and this node owns it
	capDesc, err := node.Blockchain.mcpProcessor.GetCapabilityDescriptor(capID)
	if err != nil {
		return fmt.Errorf("capability %s not found: %w", capID, err)
	}

	// Extract and verify owner
	var capOwner string
	switch desc := capDesc.(type) {
	case types.ResourceDescriptor:
		capOwner = desc.BaseDescriptor.Owner
	case types.ToolDescriptor:
		capOwner = desc.BaseDescriptor.Owner
	case types.PromptDescriptor:
		capOwner = desc.BaseDescriptor.Owner
	case types.MemoryServiceDescriptor:
		capOwner = desc.BaseDescriptor.Owner
	default:
		return fmt.Errorf("unknown capability descriptor type")
	}

	if node.Wallet == nil || node.Wallet.GetAddress() != capOwner {
		return fmt.Errorf("not the owner of capability %s", capID)
	}

	return nil
}

// ProcessDelegatedTransaction processes a validated delegated transaction
func ProcessDelegatedTransaction(tx *Transaction, node *Node) error {
	// Validate the transaction first
	if err := ValidateDelegatedTransaction(tx, node); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Add to the Plugin Author Subpool
	node.TransactionPoolManager.AddToPASPool(tx)

	agentlog.LogInfo(fmt.Sprintf("Successfully processed delegated transaction %s", tx.TransactionHash))
	return nil
}

// GetDelegationHandlerStats returns statistics about delegation handling
func GetDelegationHandlerStats(node *Node) map[string]interface{} {
	stats := map[string]interface{}{
		"delegation_protocol_id": DelegationProtocolID,
		"poaud_enabled":          node.Blockchain.PoAuDEnabled,
	}

	if node.TransactionPoolManager != nil {
		poolStats := node.TransactionPoolManager.GetPoolStats()
		for k, v := range poolStats {
			stats[k] = v
		}
	}

	return stats
}

// IsDelegationHandlerActive checks if the delegation handler is active
func IsDelegationHandlerActive(node *Node) bool {
	return node.Blockchain.PoAuDEnabled
}

// EnableDelegationHandler enables the delegation handler
func EnableDelegationHandler(node *Node) {
	if !node.Blockchain.PoAuDEnabled {
		agentlog.LogInfo("Cannot enable delegation handler: PoAu-D is disabled")
		return
	}

	RegisterDelegationHandler(node)
	agentlog.LogInfo("Delegation handler enabled")
}

// DisableDelegationHandler disables the delegation handler
func DisableDelegationHandler(node *Node) {
	// In a real implementation, you would unregister the stream handler
	agentlog.LogInfo("Delegation handler disabled")
}

// HandleDelegationError handles errors that occur during delegation processing
func HandleDelegationError(err error, txHash string) {
	agentlog.LogError(fmt.Sprintf("Delegation error for transaction %s: %v", txHash, err), err)
}

// CreateDelegationResponse creates a response for delegation requests
func CreateDelegationResponse(success bool, message string) map[string]interface{} {
	return map[string]interface{}{
		"success": success,
		"message": message,
	}
}
