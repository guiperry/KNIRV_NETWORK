// Copyright 2026 KNIRV-NEXUS
// SPDX-License-Identifier: GPL-3.0-or-later

package p2p

import (
	"fmt"
	"net"
	"testing"
	"time"

	"backend_server/internal/database"
	"backend_server/internal/ebpf"

	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestP2PSecurityServiceIntegration tests the P2P security service integration with eBPF/XDP
func TestP2PSecurityServiceIntegration(t *testing.T) {
	// Initialize eBPF Manager
	manager := ebpf.NewManager()
	require.NotNil(t, manager)

	// Initialize XDP Manager
	xdpManager := ebpf.NewXDPManager(manager)
	require.NotNil(t, xdpManager)

	// Create P2P Security Service
	p2pService, err := NewP2PService(xdpManager)
	require.NoError(t, err)
	require.NotNil(t, p2pService)

	// Start the service
	err = p2pService.Start()
	require.NoError(t, err)
	defer p2pService.Stop()

	t.Run("Add peer to whitelist", func(t *testing.T) {
		peerID := "peer-test-001"
		ipAddr := "192.168.1.100"

		err := p2pService.OnPeerConnected(peerID, ipAddr)
		require.NoError(t, err)

		// Verify peer was added
		peerInfo, err := p2pService.GetPeerInfo(peerID)
		require.NoError(t, err)
		assert.Equal(t, peerID, peerInfo.PeerID)
		assert.Equal(t, net.ParseIP(ipAddr).String(), peerInfo.IP.String())
		assert.True(t, peerInfo.IsConnected)
	})

	t.Run("Remove peer from whitelist", func(t *testing.T) {
		peerID := "peer-test-002"
		ipAddr := "192.168.1.101"

		// Add peer
		err := p2pService.OnPeerConnected(peerID, ipAddr)
		require.NoError(t, err)

		// Remove peer
		err = p2pService.OnPeerDisconnected(peerID, ipAddr)
		require.NoError(t, err)

		// Verify peer is marked as disconnected
		peerInfo, err := p2pService.GetPeerInfo(peerID)
		require.NoError(t, err)
		assert.False(t, peerInfo.IsConnected)
	})

	t.Run("Invalid IP address handling", func(t *testing.T) {
		peerID := "peer-invalid"
		invalidIP := "not-an-ip"

		err := p2pService.OnPeerConnected(peerID, invalidIP)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid IP address")
	})

	t.Run("List all peers", func(t *testing.T) {
		// Clear existing peers by creating a new service
		p2pService2, err := NewP2PService(xdpManager)
		require.NoError(t, err)
		err = p2pService2.Start()
		require.NoError(t, err)
		defer p2pService2.Stop()

		// Add multiple peers
		peers := []struct {
			id string
			ip string
		}{
			{"peer-list-001", "10.0.0.1"},
			{"peer-list-002", "10.0.0.2"},
			{"peer-list-003", "10.0.0.3"},
		}

		for _, p := range peers {
			err := p2pService2.OnPeerConnected(p.id, p.ip)
			require.NoError(t, err)
		}

		// List all peers
		allPeers := p2pService2.ListPeers()
		assert.Len(t, allPeers, len(peers))
	})

	t.Run("Peer connection lifecycle", func(t *testing.T) {
		peerID := "peer-lifecycle"
		ipAddr := "172.16.0.10"

		// Connect peer
		err := p2pService.OnPeerConnected(peerID, ipAddr)
		require.NoError(t, err)

		peerInfo, err := p2pService.GetPeerInfo(peerID)
		require.NoError(t, err)
		assert.True(t, peerInfo.IsConnected)
		initialLastSeen := peerInfo.LastSeen

		// Wait a bit
		time.Sleep(100 * time.Millisecond)

		// Disconnect peer
		err = p2pService.OnPeerDisconnected(peerID, ipAddr)
		require.NoError(t, err)

		peerInfo, err = p2pService.GetPeerInfo(peerID)
		require.NoError(t, err)
		assert.False(t, peerInfo.IsConnected)
		assert.True(t, peerInfo.LastSeen.After(initialLastSeen))
	})

	t.Run("Service start/stop", func(t *testing.T) {
		newService, err := NewP2PService(xdpManager)
		require.NoError(t, err)

		// Start service
		err = newService.Start()
		require.NoError(t, err)
		assert.True(t, newService.IsRunning())

		// Stop service
		err = newService.Stop()
		require.NoError(t, err)
		assert.False(t, newService.IsRunning())

		// Starting already running service should fail
		err = newService.Start()
		require.NoError(t, err)
		err = newService.Start()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already running")
		newService.Stop()
	})
}

// TestDVEP2PManagerSecurityIntegration tests the DVE P2P Manager with security integration
func TestDVEP2PManagerSecurityIntegration(t *testing.T) {
	// This test would require a full libp2p stack and BuntDB setup
	// Here we test the integration points

	t.Run("DVE P2P Manager with nil security service", func(t *testing.T) {
		// Create temporary DB
		tmpDB := setupTestDB(t)
		defer tmpDB.Close()

		// Create manager without security service (nil)
		manager, err := NewDVEP2PManager("test-chain", "test-role", tmpDB, false, nil)
		require.NoError(t, err)
		require.NotNil(t, manager)
		defer manager.Stop()

		// Manager should work without security service
		assert.NotNil(t, manager.host)
		assert.Nil(t, manager.p2pSecurityService)
	})

	t.Run("DVE P2P Manager with security service", func(t *testing.T) {
		// Create temporary DB
		tmpDB := setupTestDB(t)
		defer tmpDB.Close()

		// Initialize eBPF and XDP
		ebpfManager := ebpf.NewManager()
		xdpManager := ebpf.NewXDPManager(ebpfManager)

		// Create P2P security service
		securityService, err := NewP2PService(xdpManager)
		require.NoError(t, err)
		err = securityService.Start()
		require.NoError(t, err)
		defer securityService.Stop()

		// Create manager with security service
		manager, err := NewDVEP2PManager("test-chain", "test-role", tmpDB, false, securityService)
		require.NoError(t, err)
		require.NotNil(t, manager)
		defer manager.Stop()

		// Manager should have security service configured
		assert.NotNil(t, manager.p2pSecurityService)
	})
}

// TestExtractIPFromMultiaddr tests the IP extraction from multiaddr
func TestExtractIPFromMultiaddr(t *testing.T) {
	tests := []struct {
		name     string
		addr     string
		expected string
	}{
		{
			name:     "IPv4 address",
			addr:     "/ip4/192.168.1.1/tcp/4001",
			expected: "192.168.1.1",
		},
		{
			name:     "IPv6 address",
			addr:     "/ip6/::1/tcp/4001",
			expected: "::1",
		},
		{
			name:     "Complex multiaddr with IPv4",
			addr:     "/ip4/10.0.0.1/tcp/4001/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt",
			expected: "10.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse multiaddr
			ma, err := multiaddr.NewMultiaddr(tt.addr)
			require.NoError(t, err)

			// Extract IP
			ip := extractIPFromMultiaddr(ma)
			assert.Equal(t, tt.expected, ip)
		})
	}
}

// TestNetworkMetrics tests network metrics retrieval
func TestNetworkMetrics(t *testing.T) {
	// Initialize eBPF Manager
	manager := ebpf.NewManager()
	require.NotNil(t, manager)

	// Initialize XDP Manager
	xdpManager := ebpf.NewXDPManager(manager)
	require.NotNil(t, xdpManager)

	// Create P2P Security Service
	p2pService, err := NewP2PService(xdpManager)
	require.NoError(t, err)
	require.NotNil(t, p2pService)

	err = p2pService.Start()
	require.NoError(t, err)
	defer p2pService.Stop()

	t.Run("Get network metrics", func(t *testing.T) {
		metrics, err := p2pService.GetNetworkMetrics()

		// Metrics might not be available in test environment
		// but the method should not panic
		if err != nil {
			assert.Contains(t, err.Error(), "XDP collection not initialized")
		} else {
			assert.NotNil(t, metrics)
		}
	})
}

// TestConcurrentPeerOperations tests concurrent peer add/remove operations
func TestConcurrentPeerOperations(t *testing.T) {
	// Initialize eBPF Manager
	manager := ebpf.NewManager()
	require.NotNil(t, manager)

	// Initialize XDP Manager
	xdpManager := ebpf.NewXDPManager(manager)
	require.NotNil(t, xdpManager)

	// Create P2P Security Service
	p2pService, err := NewP2PService(xdpManager)
	require.NoError(t, err)
	require.NotNil(t, p2pService)

	err = p2pService.Start()
	require.NoError(t, err)
	defer p2pService.Stop()

	t.Run("Concurrent peer additions", func(t *testing.T) {
		done := make(chan bool, 10)

		// Add 10 peers concurrently
		for i := 0; i < 10; i++ {
			go func(id int) {
				peerID := fmt.Sprintf("concurrent-peer-%d", id)
				ipAddr := fmt.Sprintf("10.0.0.%d", id+1)
				err := p2pService.OnPeerConnected(peerID, ipAddr)
				assert.NoError(t, err)
				done <- true
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < 10; i++ {
			<-done
		}

		// Verify all peers were added
		peers := p2pService.ListPeers()
		assert.GreaterOrEqual(t, len(peers), 10)
	})
}

// Helper function to setup test database
func setupTestDB(t *testing.T) *database.BuntDBManager {
	db, err := database.NewBuntDB(":memory:")
	require.NoError(t, err)
	return db
}
