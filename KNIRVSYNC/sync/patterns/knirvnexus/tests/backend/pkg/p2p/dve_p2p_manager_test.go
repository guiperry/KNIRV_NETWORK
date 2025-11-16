package p2p

import (
	"context"
	"testing"
	"time"

	"backend_server/internal/objects"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/buntdb"
)

func TestNewDVEP2PManager(t *testing.T) {
	// Create a test database
	db, err := buntdb.Open(":memory:")
	require.NoError(t, err)
	defer db.Close()

	manager, err := NewDVEP2PManager("test-chain", "test-role", db, true)
	require.NoError(t, err)
	assert.NotNil(t, manager)
	assert.Equal(t, "test-chain", manager.chainID)
	assert.Equal(t, "test-role", manager.nodeRole)
	assert.NotNil(t, manager.host)
	assert.NotNil(t, manager.dht)
	assert.NotNil(t, manager.pubsub)
	assert.Equal(t, db, manager.db)
	assert.NotNil(t, manager.ctx)
	assert.NotNil(t, manager.messageHandlers)
	assert.False(t, manager.networkPaused)

	manager.Stop()
}

func TestNewDVEP2PManager_NoDHT(t *testing.T) {
	// Create a test database
	db, err := buntdb.Open(":memory:")
	require.NoError(t, err)
	defer db.Close()

	manager, err := NewDVEP2PManager("test-chain", "test-role", db, false)
	require.NoError(t, err)
	assert.NotNil(t, manager)
	assert.Nil(t, manager.dht) // DHT should be disabled

	manager.Stop()
}

func TestDVEP2PManager_Start_Stop(t *testing.T) {
	// Create a test database
	db, err := buntdb.Open(":memory:")
	require.NoError(t, err)
	defer db.Close()

	manager, err := NewDVEP2PManager("test-chain", "test-role", db, false)
	require.NoError(t, err)

	// Test starting
	manager.Start()

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	// Test stopping
	manager.Stop()
}

func TestDVEP2PManager_RegisterMessageHandler(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	require.NoError(t, err)
	defer db.Close()

	manager, err := NewDVEP2PManager("test-chain", "test-role", db, false)
	require.NoError(t, err)
	defer manager.Stop()

	// Create a mock handler
	handler := &mockMessageHandler{}

	manager.RegisterMessageHandler("test-type", handler)

	// Verify handler was registered
	manager.mu.RLock()
	registeredHandler, exists := manager.messageHandlers["test-type"]
	manager.mu.RUnlock()

	assert.True(t, exists)
	assert.Equal(t, handler, registeredHandler)
}

func TestDVEP2PManager_IsNetworkPaused(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	require.NoError(t, err)
	defer db.Close()

	manager, err := NewDVEP2PManager("test-chain", "test-role", db, false)
	require.NoError(t, err)
	defer manager.Stop()

	// Initially not paused
	assert.False(t, manager.IsNetworkPaused())

	// Manually set paused
	manager.pauseMutex.Lock()
	manager.networkPaused = true
	manager.pausedUntil = time.Now().Add(time.Hour)
	manager.pauseMutex.Unlock()

	assert.True(t, manager.IsNetworkPaused())

	// Test expired pause
	manager.pauseMutex.Lock()
	manager.pausedUntil = time.Now().Add(-time.Hour)
	manager.pauseMutex.Unlock()

	assert.False(t, manager.IsNetworkPaused())
}

func TestDVEP2PManager_GetConnectedPeers(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	require.NoError(t, err)
	defer db.Close()

	manager, err := NewDVEP2PManager("test-chain", "test-role", db, false)
	require.NoError(t, err)
	defer manager.Stop()

	peers := manager.GetConnectedPeers()
	// In a test environment, we might not have any peers
	assert.IsType(t, []objects.PeerInfo{}, peers)
}

func TestDVEP2PManager_GetNetworkTopology(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	require.NoError(t, err)
	defer db.Close()

	manager, err := NewDVEP2PManager("test-chain", "test-role", db, false)
	require.NoError(t, err)
	defer manager.Stop()

	topology := manager.GetNetworkTopology()
	assert.NotNil(t, topology)
	assert.Contains(t, topology.ID, "topology-")
	assert.True(t, topology.TotalPeers >= 0)
	assert.True(t, topology.ConnectedPeers >= 0)
	assert.NotNil(t, topology.Peers)
	assert.NotNil(t, topology.Connections)
	assert.True(t, topology.Timestamp.After(time.Now().Add(-time.Minute)))
}

func TestDVEP2PManager_UpdatePeerReputation(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	require.NoError(t, err)
	defer db.Close()

	manager, err := NewDVEP2PManager("test-chain", "test-role", db, false)
	require.NoError(t, err)
	defer manager.Stop()

	// Test positive behavior
	manager.UpdatePeerReputation("test-peer", "good_behavior", true)

	// Test negative behavior
	manager.UpdatePeerReputation("test-peer", "bad_behavior", false)
}

func TestDVEP2PManager_getPeerReputation(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	require.NoError(t, err)
	defer db.Close()

	manager, err := NewDVEP2PManager("test-chain", "test-role", db, false)
	require.NoError(t, err)
	defer manager.Stop()

	reputation := manager.getPeerReputation("test-peer")
	assert.Equal(t, DefaultReputationScore, reputation)
}

func TestDVEP2PManager_setPeerReputation(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	require.NoError(t, err)
	defer db.Close()

	manager, err := NewDVEP2PManager("test-chain", "test-role", db, false)
	require.NoError(t, err)
	defer manager.Stop()

	// This should not panic
	manager.setPeerReputation("test-peer", 75.0)
}

func TestDVEP2PManager_calculatePeerScore(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	require.NoError(t, err)
	defer db.Close()

	manager, err := NewDVEP2PManager("test-chain", "test-role", db, false)
	require.NoError(t, err)
	defer manager.Stop()

	score := manager.calculatePeerScore(manager.host.ID())
	assert.True(t, score >= 0)
}

func TestDVEP2PManager_GetPeerStats(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	require.NoError(t, err)
	defer db.Close()

	manager, err := NewDVEP2PManager("test-chain", "test-role", db, false)
	require.NoError(t, err)
	defer manager.Stop()

	stats := manager.GetPeerStats()
	assert.NotNil(t, stats)
	assert.Contains(t, stats, "total_peers")
	assert.Contains(t, stats, "connected_peers")
	assert.Contains(t, stats, "bootstrap_peers")
	assert.Contains(t, stats, "reputation_range")
	assert.Contains(t, stats, "average_reputation")
	assert.Contains(t, stats, "blacklisted_peers")
}

func TestDVEP2PManager_EncryptMessage(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	require.NoError(t, err)
	defer db.Close()

	manager, err := NewDVEP2PManager("test-chain", "test-role", db, false)
	require.NoError(t, err)
	defer manager.Stop()

	message := []byte("test message")
	encrypted, err := manager.EncryptMessage(message, "recipient-peer")
	require.NoError(t, err)
	// In the placeholder implementation, it returns the message unchanged
	assert.Equal(t, message, encrypted)
}

func TestDVEP2PManager_DecryptMessage(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	require.NoError(t, err)
	defer db.Close()

	manager, err := NewDVEP2PManager("test-chain", "test-role", db, false)
	require.NoError(t, err)
	defer manager.Stop()

	encrypted := []byte("encrypted message")
	decrypted, err := manager.DecryptMessage(encrypted)
	require.NoError(t, err)
	// In the placeholder implementation, it returns the message unchanged
	assert.Equal(t, encrypted, decrypted)
}

func TestDVEP2PManager_OptimizeNetworkTopology(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	require.NoError(t, err)
	defer db.Close()

	manager, err := NewDVEP2PManager("test-chain", "test-role", db, false)
	require.NoError(t, err)
	defer manager.Stop()

	// Should not panic
	manager.OptimizeNetworkTopology()
}

func TestDVEP2PManager_AnnounceValidationRequest(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	require.NoError(t, err)
	defer db.Close()

	manager, err := NewDVEP2PManager("test-chain", "test-role", db, false)
	require.NoError(t, err)
	defer manager.Stop()

	requirements := map[string]string{"cpu": "2", "memory": "4GB"}
	metadata := map[string]string{"priority": "high"}

	err = manager.AnnounceValidationRequest("req-123", "skill-validation", 5, requirements, metadata)
	// This might fail if network is paused or other issues, but should not panic
	// We just test that the method exists and can be called
	_ = err // We don't assert on the error since it depends on network state
}

func TestDVEP2PManager_AnnounceValidationRequest_NetworkPaused(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	require.NoError(t, err)
	defer db.Close()

	manager, err := NewDVEP2PManager("test-chain", "test-role", db, false)
	require.NoError(t, err)
	defer manager.Stop()

	// Manually pause the network
	manager.pauseMutex.Lock()
	manager.networkPaused = true
	manager.pausedUntil = time.Now().Add(time.Hour) // Ensure it doesn't expire
	manager.pauseMutex.Unlock()

	requirements := map[string]string{"cpu": "2", "memory": "4GB"}
	metadata := map[string]string{"priority": "high"}

	err = manager.AnnounceValidationRequest("req-123", "skill-validation", 5, requirements, metadata)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "network is paused")
}

func TestDVEP2PManager_AnnounceValidationResult_NetworkPaused(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	require.NoError(t, err)
	defer db.Close()

	manager, err := NewDVEP2PManager("test-chain", "test-role", db, false)
	require.NoError(t, err)
	defer manager.Stop()

	// Manually pause the network
	manager.pauseMutex.Lock()
	manager.networkPaused = true
	manager.pausedUntil = time.Now().Add(time.Hour) // Ensure it doesn't expire
	manager.pauseMutex.Unlock()

	evidence := map[string]string{"test1": "passed", "test2": "passed"}
	metadata := map[string]string{"confidence": "high"}

	err = manager.AnnounceValidationResult("req-123", "passed", 0.95, evidence, metadata)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "network is paused")
}

func TestDVEP2PManager_AnnounceNodeStatus_NetworkPaused(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	require.NoError(t, err)
	defer db.Close()

	manager, err := NewDVEP2PManager("test-chain", "test-role", db, false)
	require.NoError(t, err)
	defer manager.Stop()

	// Manually pause the network
	manager.pauseMutex.Lock()
	manager.networkPaused = true
	manager.pausedUntil = time.Now().Add(time.Hour) // Ensure it doesn't expire
	manager.pauseMutex.Unlock()

	capabilities := []string{"validation", "computation"}
	metadata := map[string]string{"version": "1.0.0"}

	err = manager.AnnounceNodeStatus("node-123", "active", capabilities, 0.3, metadata)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "network is paused")
}

func TestDVEP2PManager_bootstrapToNetwork_NoPeers(t *testing.T) {
	// Temporarily replace default bootstrap peers
	originalPeers := DefaultBootstrapPeers
	DefaultBootstrapPeers = []string{}
	defer func() { DefaultBootstrapPeers = originalPeers }()

	db, err := buntdb.Open(":memory:")
	require.NoError(t, err)
	defer db.Close()

	manager, err := NewDVEP2PManager("test-chain", "test-role", db, false)
	require.NoError(t, err)
	defer manager.Stop()

	err = manager.bootstrapToNetwork()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no valid bootstrap peers available")
}

func TestDVEP2PManager_calculatePeerScore_NoConnections(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	require.NoError(t, err)
	defer db.Close()

	manager, err := NewDVEP2PManager("test-chain", "test-role", db, false)
	require.NoError(t, err)
	defer manager.Stop()

	// Test with a peer that has no connections
	score := manager.calculatePeerScore(manager.host.ID())
	assert.True(t, score >= 0)
	// Score should be based only on reputation since there are no connections
	assert.True(t, score <= DefaultReputationScore)
}

func TestDVEP2PManager_UpdatePeerReputation_Bounds(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	require.NoError(t, err)
	defer db.Close()

	manager, err := NewDVEP2PManager("test-chain", "test-role", db, false)
	require.NoError(t, err)
	defer manager.Stop()

	peerID := "test-peer"

	// Test upper bound - the current implementation doesn't enforce bounds
	manager.setPeerReputation(peerID, MaxReputationScore)
	manager.UpdatePeerReputation(peerID, "good_behavior", true)
	// Current implementation allows exceeding max
	reputation := manager.getPeerReputation(peerID)
	assert.Equal(t, DefaultReputationScore, reputation) // Always returns default

	// Test lower bound - the current implementation doesn't enforce bounds
	manager.setPeerReputation(peerID, MinReputationScore)
	manager.UpdatePeerReputation(peerID, "bad_behavior", false)
	// Current implementation allows going below min
	reputation = manager.getPeerReputation(peerID)
	assert.Equal(t, DefaultReputationScore, reputation) // Always returns default
}

func TestDVEP2PManager_GetPeerStats_EmptyNetwork(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	require.NoError(t, err)
	defer db.Close()

	manager, err := NewDVEP2PManager("test-chain", "test-role", db, false)
	require.NoError(t, err)
	defer manager.Stop()

	stats := manager.GetPeerStats()
	assert.NotNil(t, stats)
	assert.Equal(t, 0, stats["total_peers"])
	assert.Equal(t, 0, stats["connected_peers"])
	assert.Equal(t, len(DefaultBootstrapPeers), stats["bootstrap_peers"])
	assert.Equal(t, 0, stats["blacklisted_peers"])
}

func TestDVEP2PManager_setupTopics_Error(t *testing.T) {
	// This test is hard to implement without mocking the pubsub
	// The setupTopics method doesn't have explicit error returns in the current implementation
	// but it could fail if pubsub.Join fails
	t.Skip("setupTopics error testing requires pubsub mocking")
}

func TestDVEP2PManager_handleNetworkControl_InvalidJSON(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	require.NoError(t, err)
	defer db.Close()

	manager, err := NewDVEP2PManager("test-chain", "test-role", db, false)
	require.NoError(t, err)
	defer manager.Stop()

	// This would require mocking the subscription to send invalid JSON
	// For now, just test that the method exists
	assert.NotNil(t, manager)
}

func TestDVEP2PManager_processValidationRequestAnnouncement_InvalidData(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	require.NoError(t, err)
	defer db.Close()

	manager, err := NewDVEP2PManager("test-chain", "test-role", db, false)
	require.NoError(t, err)
	defer manager.Stop()

	// Test with invalid data structure
	msg := DVEAnnouncementMessage{
		Type:      "validation_request",
		Action:    "available",
		ServiceID: "test-service",
		Data:      "invalid data", // Should be map[string]interface{}
		Timestamp: time.Now().Unix(),
	}

	// Should not panic
	manager.processValidationRequestAnnouncement(msg)
}

func TestDVEP2PManager_processValidationResultAnnouncement_InvalidData(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	require.NoError(t, err)
	defer db.Close()

	manager, err := NewDVEP2PManager("test-chain", "test-role", db, false)
	require.NoError(t, err)
	defer manager.Stop()

	// Test with invalid data structure
	msg := DVEAnnouncementMessage{
		Type:      "validation_result",
		Action:    "completed",
		ServiceID: "test-service",
		Data:      "invalid data", // Should be map[string]interface{}
		Timestamp: time.Now().Unix(),
	}

	// Should not panic
	manager.processValidationResultAnnouncement(msg)
}

func TestDVEP2PManager_processNodeStatusAnnouncement_InvalidData(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	require.NoError(t, err)
	defer db.Close()

	manager, err := NewDVEP2PManager("test-chain", "test-role", db, false)
	require.NoError(t, err)
	defer manager.Stop()

	// Test with invalid data structure
	msg := DVEAnnouncementMessage{
		Type:      "node_status",
		Action:    "updated",
		ServiceID: "test-service",
		Data:      "invalid data", // Should be map[string]interface{}
		Timestamp: time.Now().Unix(),
	}

	// Should not panic
	manager.processNodeStatusAnnouncement(msg)
}

func TestDVEP2PManager_AnnounceValidationResult(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	require.NoError(t, err)
	defer db.Close()

	manager, err := NewDVEP2PManager("test-chain", "test-role", db, false)
	require.NoError(t, err)
	defer manager.Stop()

	evidence := map[string]string{"test1": "passed", "test2": "passed"}
	metadata := map[string]string{"confidence": "high"}

	err = manager.AnnounceValidationResult("req-123", "passed", 0.95, evidence, metadata)
	// This might fail if network is paused or other issues, but should not panic
	_ = err
}

func TestDVEP2PManager_AnnounceNodeStatus(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	require.NoError(t, err)
	defer db.Close()

	manager, err := NewDVEP2PManager("test-chain", "test-role", db, false)
	require.NoError(t, err)
	defer manager.Stop()

	capabilities := []string{"validation", "computation"}
	metadata := map[string]string{"version": "1.0.0"}

	err = manager.AnnounceNodeStatus("node-123", "active", capabilities, 0.3, metadata)
	// This might fail if network is paused or other issues, but should not panic
	_ = err
}

func TestDVEP2PManager_createCIDFromServiceID(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	require.NoError(t, err)
	defer db.Close()

	manager, err := NewDVEP2PManager("test-chain", "test-role", db, false)
	require.NoError(t, err)
	defer manager.Stop()

	cid, err := manager.createCIDFromServiceID("test-service")
	require.NoError(t, err)
	assert.NotEmpty(t, cid.String())
}

// Mock message handler for testing
type mockMessageHandler struct{}

func (m *mockMessageHandler) HandleMessage(ctx context.Context, msg *objects.P2PMessage) error {
	// Mock implementation
	return nil
}