package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p/core/network"

	agentlog "KNIRVROOT/log"
)

const StatusProtocolID = "/agent/status/1.0.0"

// RegisterStatusHandler registers the status protocol handler
func RegisterStatusHandler(node *Node) {
	// Only register if PoAu-D is enabled
	if !node.Blockchain.PoAuDEnabled {
		agentlog.LogInfo("PoAu-D is disabled, not registering status handler")
		return
	}

	// Note: This is a placeholder implementation
	// In the actual implementation, you would set the stream handler on the libp2p host
	// node.Host.SetStreamHandler(StatusProtocolID, func(stream network.Stream) {
	//     handleStatusStream(stream, node)
	// })

	agentlog.LogInfo("Registered status protocol handler")
}

// handleStatusStream serves PAP status information
func handleStatusStream(stream network.Stream, node *Node) {
	defer stream.Close()

	// Prepare status response
	status := PAPStatus{
		Status:             getNodeStatus(node),
		SubpoolQueueLength: getSubpoolQueueLength(node),
	}

	// Add additional status information
	extendedStatus := map[string]interface{}{
		"status":               status.Status,
		"subpoolQueueLength":   status.SubpoolQueueLength,
		"timestamp":            time.Now().Unix(),
		"poaudEnabled":         node.Blockchain.PoAuDEnabled,
		"nodeAddress":          getNodeAddress(node.Wallet),
		"chainID":              node.Blockchain.ChainID,
		"isActivelyMining":     node.Blockchain.IsActivelyMining(),
		"mainPoolSize":         getMainPoolSize(node),
		"delegatedTxCount":     getDelegatedTransactionCount(node),
	}

	// Encode and send status
	statusBytes, err := json.Marshal(extendedStatus)
	if err != nil {
		agentlog.LogError("Failed to marshal status", err)
		return
	}

	statusBytes = append(statusBytes, '\n')
	if _, err := stream.Write(statusBytes); err != nil {
		agentlog.LogError("Failed to write status to stream", err)
	}

	agentlog.LogInfo(fmt.Sprintf("Served status to peer: %s", stream.Conn().RemotePeer().String()))
}

// getNodeStatus determines the current status of the node
func getNodeStatus(node *Node) string {
	if !node.Blockchain.PoAuDEnabled {
		return "OFFLINE"
	}

	// Check if the node is too busy
	if node.TransactionPoolManager != nil {
		pasPoolSize := node.TransactionPoolManager.GetPASPoolSize()
		if pasPoolSize > MaxPapSubpoolQueue {
			return "BUSY"
		}
	}

	// Check if the node is actively mining
	if node.Blockchain.IsActivelyMining() {
		return "MINING"
	}

	return "ONLINE"
}

// getSubpoolQueueLength returns the current size of the PAS pool
func getSubpoolQueueLength(node *Node) int {
	if node.TransactionPoolManager == nil {
		return 0
	}
	return node.TransactionPoolManager.GetPASPoolSize()
}

// getMainPoolSize returns the size of the main transaction pool
func getMainPoolSize(node *Node) int {
	if node.TransactionPoolManager == nil {
		return 0
	}
	return node.TransactionPoolManager.GetMainPoolSize()
}

// getDelegatedTransactionCount returns the number of delegated transactions
func getDelegatedTransactionCount(node *Node) int {
	if node.TransactionPoolManager == nil {
		return 0
	}
	return node.TransactionPoolManager.GetDelegatedTransactionsCount()
}

// StartStatusAdvertising periodically advertises the node's status URI
func StartStatusAdvertising(node *Node) {
	// Only start if PoAu-D is enabled
	if !node.Blockchain.PoAuDEnabled {
		agentlog.LogInfo("PoAu-D is disabled, not starting status advertising")
		return
	}

	agentlog.LogInfo("Starting status advertising")

	go func() {
		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()

		statusURI := fmt.Sprintf("agent://%s.chain/status", node.Blockchain.ChainID)

		for range ticker.C {
			// Skip if PoAu-D was disabled
			if !node.Blockchain.PoAuDEnabled {
				continue
			}

			// Advertise status URI on DHT
			// Implementation depends on DHT provider logic
			agentlog.LogInfo(fmt.Sprintf("Advertised status URI %s on DHT", statusURI))
		}
	}()
}

// GetNodeStatusInfo returns comprehensive status information about the node
func GetNodeStatusInfo(node *Node) map[string]interface{} {
	status := map[string]interface{}{
		"status":               getNodeStatus(node),
		"subpoolQueueLength":   getSubpoolQueueLength(node),
		"timestamp":            time.Now().Unix(),
		"poaudEnabled":         node.Blockchain.PoAuDEnabled,
		"nodeAddress":          getNodeAddress(node.Wallet),
		"chainID":              node.Blockchain.ChainID,
		"isActivelyMining":     node.Blockchain.IsActivelyMining(),
		"mainPoolSize":         getMainPoolSize(node),
		"delegatedTxCount":     getDelegatedTransactionCount(node),
		"protocolID":           StatusProtocolID,
	}

	// Add network authors information if available
	if node.Blockchain.NetworkAuthors != nil {
		status["networkAuthorsCount"] = len(node.Blockchain.NetworkAuthors)
		status["isNetworkAuthor"] = node.Blockchain.IsNetworkAuthor(getNodeAddress(node.Wallet))
	}

	return status
}

// IsNodeHealthy checks if the node is in a healthy state
func IsNodeHealthy(node *Node) bool {
	status := getNodeStatus(node)
	return status == "ONLINE" || status == "MINING"
}

// GetStatusProtocolID returns the status protocol ID
func GetStatusProtocolID() string {
	return StatusProtocolID
}

// EnableStatusHandler enables the status handler
func EnableStatusHandler(node *Node) {
	if !node.Blockchain.PoAuDEnabled {
		agentlog.LogInfo("Cannot enable status handler: PoAu-D is disabled")
		return
	}

	RegisterStatusHandler(node)
	StartStatusAdvertising(node)
	agentlog.LogInfo("Status handler enabled")
}

// DisableStatusHandler disables the status handler
func DisableStatusHandler(node *Node) {
	// In a real implementation, you would unregister the stream handler and stop advertising
	agentlog.LogInfo("Status handler disabled")
}

// CreateStatusResponse creates a standardized status response
func CreateStatusResponse(node *Node) map[string]interface{} {
	return GetNodeStatusInfo(node)
}

// ValidateStatusRequest validates incoming status requests
func ValidateStatusRequest(request map[string]interface{}) error {
	// Add validation logic for status requests if needed
	return nil
}

// HandleStatusError handles errors that occur during status processing
func HandleStatusError(err error, peerID string) {
	agentlog.LogError(fmt.Sprintf("Status error for peer %s: %v", peerID, err), err)
}

// GetStatusHandlerStats returns statistics about status handling
func GetStatusHandlerStats(node *Node) map[string]interface{} {
	return map[string]interface{}{
		"status_protocol_id":    StatusProtocolID,
		"poaud_enabled":         node.Blockchain.PoAuDEnabled,
		"status_handler_active": node.Blockchain.PoAuDEnabled,
		"node_status":           getNodeStatus(node),
		"last_status_check":     time.Now().Unix(),
	}
}

// IsStatusHandlerActive checks if the status handler is active
func IsStatusHandlerActive(node *Node) bool {
	return node.Blockchain.PoAuDEnabled
}

// UpdateNodeStatus updates the node's status (for testing or manual control)
func UpdateNodeStatus(node *Node, newStatus string) {
	agentlog.LogInfo(fmt.Sprintf("Node status updated to: %s", newStatus))
	// In a real implementation, you might want to store this status
}

// GetStatusAdvertisingInterval returns the status advertising interval
func GetStatusAdvertisingInterval() time.Duration {
	return 30 * time.Minute
}

// CheckPeerStatus checks the status of a remote peer
func CheckPeerStatus(peerID string, node *Node) (map[string]interface{}, error) {
	// Implementation to check remote peer status
	// This would involve making a request to the peer's status endpoint
	return map[string]interface{}{
		"peer_id": peerID,
		"status":  "unknown",
	}, nil
}
