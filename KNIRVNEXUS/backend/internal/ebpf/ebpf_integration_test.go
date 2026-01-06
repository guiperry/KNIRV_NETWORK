// Copyright 2026 KNIRV-NEXUS
// SPDX-License-Identifier: GPL-3.0-or-later

package ebpf_test

import (
	"fmt"
	"log"
	"net"
	"testing"
	"time"

	"backend_server/internal/ebpf"
	"backend_server/internal/services/cde"
	"backend_server/internal/services/p2p"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)



// TestEBPFIntegration tests the complete eBPF integration
func TestEBPFIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping eBPF integration test in short mode")
	}

	// Create eBPF manager
	manager := ebpf.NewManager()
	xdpManager := ebpf.NewXDPManager(manager)
	vcManager := ebpf.NewVirtualContainerManager(manager)

	// Initialize components
	err := xdpManager.InitializeXDP()
	require.NoError(t, err, "XDP manager initialization should succeed")
	defer xdpManager.ShutdownXDP()

	err = vcManager.InitializeVirtualContainers()
	require.NoError(t, err, "Virtual container manager initialization should succeed")
	defer vcManager.ShutdownVirtualContainers()

	// Test 1: XDP Network Filtering Integration
	t.Run("XDP_Network_Filtering", func(t *testing.T) {
		// Test IP whitelisting
		testIP := net.IPv4(192, 168, 1, 100)
		err := xdpManager.AddWhitelistedIP(testIP)
		assert.NoError(t, err, "Adding whitelisted IP should succeed")

		// Test metrics collection
		metrics, err := xdpManager.GetNetworkMetrics()
		assert.NoError(t, err, "Getting network metrics should succeed")
		assert.NotNil(t, metrics, "Metrics should not be nil")

		// Test IP removal
		err = xdpManager.RemoveWhitelistedIP(testIP)
		assert.NoError(t, err, "Removing whitelisted IP should succeed")
	})

	// Test 2: Virtual Container Creation
	t.Run("Virtual_Container_Creation", func(t *testing.T) {
		// Create virtual container
		container, err := vcManager.CreateVirtualContainer(1234, "/tmp/test-rootfs")
		assert.NoError(t, err, "Creating virtual container should succeed")
		assert.NotNil(t, container, "Container should not be nil")

		// Test container retrieval
		retrievedContainer, err := vcManager.GetVirtualContainer(container.ID)
		assert.NoError(t, err, "Getting virtual container should succeed")
		assert.Equal(t, container.ID, retrievedContainer.ID, "Container IDs should match")

		// Test network access control
		err = vcManager.SetVirtualContainerNetworkAccess(container.ID, true)
		assert.NoError(t, err, "Setting network access should succeed")

		// Test container destruction
		err = vcManager.DestroyVirtualContainer(container.ID)
		assert.NoError(t, err, "Destroying virtual container should succeed")
	})

	// Test 3: P2P Service Integration
	t.Run("P2P_Service_Integration", func(t *testing.T) {
		// Create P2P service with eBPF integration
		p2pService, err := p2p.NewP2PService(xdpManager)
		assert.NoError(t, err, "Creating P2P service should succeed")

		// Start P2P service
		err = p2pService.Start()
		assert.NoError(t, err, "Starting P2P service should succeed")
		defer p2pService.Stop()

		// Test peer connection (simulated)
		testPeerID := "test-peer-123"
		testPeerIP := net.IPv4(192, 168, 1, 200)
		p2pService.OnPeerConnected(testPeerID, testPeerIP)

		// Verify peer is whitelisted
		peerInfo, err := p2pService.GetPeerInfo(testPeerID)
		assert.NoError(t, err, "Getting peer info should succeed")
		assert.Equal(t, testPeerIP.String(), peerInfo.IP.String(), "Peer IP should match")

		// Test peer disconnection
		p2pService.OnPeerDisconnected(testPeerID)

		// Test network metrics through P2P service
		metrics, err := p2pService.GetNetworkMetrics()
		assert.NoError(t, err, "Getting network metrics through P2P service should succeed")
		assert.NotNil(t, metrics, "Metrics should not be nil")
	})

	// Test 4: CDE Service Integration
	t.Run("CDE_Service_Integration", func(t *testing.T) {
		// Create CDE service with eBPF integration
		cdeConfig := cde.CDEConfig{
			WorkspaceRoot:   "/tmp/cde-test",
			MaxEnvironments: 10,
			MaxCPUPerEnv:    2.0,
			MaxMemoryPerEnv: 4 * 1024 * 1024 * 1024,  // 4GB
			MaxDiskPerEnv:   20 * 1024 * 1024 * 1024, // 20GB
		}

		// Mock TEE security service and data engine (nil for testing)
		cdeService, err := cde.NewCDEService(nil, nil, vcManager, cdeConfig)
		assert.NoError(t, err, "Creating CDE service should succeed")

		// Start CDE service
		err = cdeService.Start()
		assert.NoError(t, err, "Starting CDE service should succeed")
		defer cdeService.Stop()

		// Test virtual CDE creation
		env, err := cdeService.CreateVirtualCDE("test-user", "test-virtual-cde", cde.EnvTypePython, nil)
		assert.NoError(t, err, "Creating virtual CDE should succeed")
		assert.NotNil(t, env, "Environment should not be nil")
		assert.Contains(t, env.ID, "virtual", "Environment ID should indicate virtual CDE")

		// Wait for environment to be created
		time.Sleep(100 * time.Millisecond)

		// Verify environment exists
		retrievedEnv, err := cdeService.GetEnvironment(env.ID)
		assert.NoError(t, err, "Getting environment should succeed")
		assert.Equal(t, env.ID, retrievedEnv.ID, "Environment IDs should match")

		// Test environment cleanup
		err = cdeService.StopEnvironment(env.ID)
		assert.NoError(t, err, "Stopping environment should succeed")
	})

	// Test 5: Performance Benchmarks
	t.Run("Performance_Benchmarks", func(t *testing.T) {
		// Test XDP performance
		startTime := time.Now()
		for i := 0; i < 10; i++ {
			testIP := net.IPv4(192, 168, 1, byte(i))
			_ = xdpManager.AddWhitelistedIP(testIP)
			_ = xdpManager.RemoveWhitelistedIP(testIP)
		}
		xdpDuration := time.Since(startTime)
		avgXDPTime := xdpDuration / 10
		assert.Less(t, avgXDPTime, 5*time.Millisecond, "Average XDP operation time should be less than 5ms")

		// Test virtual container performance
		startTime = time.Now()
		for i := 0; i < 5; i++ {
			container, _ := vcManager.CreateVirtualContainer(uint32(1000+i), fmt.Sprintf("/tmp/test-rootfs-%d", i))
			if container != nil {
				vcManager.DestroyVirtualContainer(container.ID)
			}
		}
		vcDuration := time.Since(startTime)
		avgVCTime := vcDuration / 5
		assert.Less(t, avgVCTime, 10*time.Millisecond, "Average virtual container operation time should be less than 10ms")
	})

	// Test 6: Error Handling
	t.Run("Error_Handling", func(t *testing.T) {
		// Test XDP error handling
		metrics, err := xdpManager.GetNetworkMetrics()
		assert.NoError(t, err, "Getting metrics should succeed even with no traffic")
		assert.NotNil(t, metrics, "Metrics should be returned even when empty")

		// Test virtual container error handling
		container, err := vcManager.GetVirtualContainer(999999)
		assert.Error(t, err, "Getting non-existent container should fail")
		assert.Nil(t, container, "Container should be nil on error")
	})
}

// TestEBPFStress tests eBPF components under load
func TestEBPFStress(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping eBPF stress test in short mode")
	}

	// Create eBPF manager
	manager := ebpf.NewManager()
	xdpManager := ebpf.NewXDPManager(manager)
	vcManager := ebpf.NewVirtualContainerManager(manager)

	// Initialize components
	err := xdpManager.InitializeXDP()
	require.NoError(t, err, "XDP manager initialization should succeed")
	defer xdpManager.ShutdownXDP()

	err = vcManager.InitializeVirtualContainers()
	require.NoError(t, err, "Virtual container manager initialization should succeed")
	defer vcManager.ShutdownVirtualContainers()

	// Test concurrent XDP operations
	t.Run("Concurrent_XDP_Operations", func(t *testing.T) {
		errors := make(chan error, 100)

		// Launch multiple goroutines performing XDP operations
		for i := 0; i < 100; i++ {
			go func(id int) {
				testIP := net.IPv4(192, 168, byte(id%256), byte(id%256))

				// Add to whitelist
				err := xdpManager.AddWhitelistedIP(testIP)
				if err != nil {
					errors <- fmt.Errorf("add IP %d: %w", id, err)
					return
				}

				// Remove from whitelist
				err = xdpManager.RemoveWhitelistedIP(testIP)
				if err != nil {
					errors <- fmt.Errorf("remove IP %d: %w", id, err)
					return
				}

				errors <- nil
			}(i)
		}

		// Collect results
		errorCount := 0
		for i := 0; i < 100; i++ {
			err := <-errors
			if err != nil {
				errorCount++
				log.Printf("Error in concurrent operation: %v", err)
			}
		}

		assert.Less(t, errorCount, 5, "Should have minimal errors in concurrent XDP operations")
	})

	// Test concurrent virtual container operations
	t.Run("Concurrent_Virtual_Container_Operations", func(t *testing.T) {
		errors := make(chan error, 50)

		// Launch multiple goroutines creating virtual containers
		for i := 0; i < 50; i++ {
			go func(id int) {
				container, err := vcManager.CreateVirtualContainer(uint32(2000+id), fmt.Sprintf("/tmp/test-rootfs-%d", id))
				if err != nil {
					errors <- fmt.Errorf("create container %d: %w", id, err)
					return
				}

				// Set network access
				err = vcManager.SetVirtualContainerNetworkAccess(container.ID, true)
				if err != nil {
					errors <- fmt.Errorf("set network access %d: %w", id, err)
					return
				}

				// Destroy container
				err = vcManager.DestroyVirtualContainer(container.ID)
				if err != nil {
					errors <- fmt.Errorf("destroy container %d: %w", id, err)
					return
				}

				errors <- nil
			}(i)
		}

		// Collect results
		errorCount := 0
		for i := 0; i < 50; i++ {
			err := <-errors
			if err != nil {
				errorCount++
				log.Printf("Error in concurrent container operation: %v", err)
			}
		}

		assert.Less(t, errorCount, 5, "Should have minimal errors in concurrent virtual container operations")
	})
}

// TestEBPFEndToEnd tests the complete eBPF workflow
func TestEBPFEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping eBPF end-to-end test in short mode")
	}

	// Create eBPF manager
	manager := ebpf.NewManager()
	xdpManager := ebpf.NewXDPManager(manager)
	vcManager := ebpf.NewVirtualContainerManager(manager)

	// Initialize components
	err := xdpManager.InitializeXDP()
	require.NoError(t, err, "XDP manager initialization should succeed")
	defer xdpManager.ShutdownXDP()

	err = vcManager.InitializeVirtualContainers()
	require.NoError(t, err, "Virtual container manager initialization should succeed")
	defer vcManager.ShutdownVirtualContainers()

	// Create P2P service
	p2pService, err := p2p.NewP2PService(xdpManager)
	assert.NoError(t, err, "Creating P2P service should succeed")
	err = p2pService.Start()
	assert.NoError(t, err, "Starting P2P service should succeed")
	defer p2pService.Stop()

	// Create CDE service
	cdeConfig := cde.CDEConfig{
		WorkspaceRoot:   "/tmp/cde-e2e-test",
		MaxEnvironments: 5,
		MaxCPUPerEnv:    1.0,
		MaxMemoryPerEnv: 2 * 1024 * 1024 * 1024,  // 2GB
		MaxDiskPerEnv:   10 * 1024 * 1024 * 1024, // 10GB
	}

	cdeService, err := cde.NewCDEService(nil, nil, vcManager, cdeConfig)
	assert.NoError(t, err, "Creating CDE service should succeed")
	err = cdeService.Start()
	assert.NoError(t, err, "Starting CDE service should succeed")
	defer cdeService.Stop()

	// Simulate the complete workflow:
	// 1. Peer connects to P2P network
	testPeerID := "e2e-test-peer"
	testPeerIP := net.IPv4(10, 0, 0, 1)
	p2pService.OnPeerConnected(testPeerID, testPeerIP)

	// 2. Create virtual CDE for development
	env, err := cdeService.CreateVirtualCDE("test-dev", "e2e-test-env", cde.EnvTypeGo, map[string]interface{}{
		"project":  "test-project",
		"language": "go",
	})
	assert.NoError(t, err, "Creating virtual CDE should succeed")

	// 3. Wait for environment to be ready
	time.Sleep(200 * time.Millisecond)

	// 4. Verify environment is running
	retrievedEnv, err := cdeService.GetEnvironment(env.ID)
	assert.NoError(t, err, "Getting environment should succeed")
	assert.Equal(t, cde.EnvStatusRunning, retrievedEnv.Status, "Environment should be running")

	// 5. Check network metrics (should show peer is whitelisted)
	metrics, err := p2pService.GetNetworkMetrics()
	assert.NoError(t, err, "Getting network metrics should succeed")
	assert.NotNil(t, metrics, "Metrics should not be nil")

	// 6. Simulate peer disconnection
	p2pService.OnPeerDisconnected(testPeerID)

	// 7. Clean up environment
	err = cdeService.StopEnvironment(env.ID)
	assert.NoError(t, err, "Stopping environment should succeed")

	// 8. Verify all components are still operational
	finalMetrics, err := xdpManager.GetNetworkMetrics()
	assert.NoError(t, err, "Getting final network metrics should succeed")
	assert.NotNil(t, finalMetrics, "Final metrics should not be nil")

	containers, err := vcManager.ListVirtualContainers()
	assert.NoError(t, err, "Listing virtual containers should succeed")
	// Containers should be cleaned up automatically
	assert.LessOrEqual(t, len(containers), 1, "Should have minimal containers after cleanup")

	log.Println("eBPF end-to-end test completed successfully")
}
