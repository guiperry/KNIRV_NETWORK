// Copyright 2026 KNIRV-NEXUS
// SPDX-License-Identifier: GPL-3.0-or-later

package ebpf

import (
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestXDPManagerBasic tests basic XDP manager functionality
func TestXDPManagerBasic(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping XDP basic test in short mode")
	}

	// eBPF operations require root privileges
	if os.Getuid() != 0 {
		t.Skip("Skipping XDP test: requires root privileges (eBPF operations)")
	}

	// Create a mock manager
	manager := NewManager()
	xdpManager := NewXDPManager(manager)

	// Test initialization
	err := xdpManager.InitializeXDP()
	assert.NoError(t, err, "XDP manager initialization should succeed")

	// Test metrics retrieval
	metrics, err := xdpManager.GetNetworkMetrics()
	assert.NoError(t, err, "Getting network metrics should succeed")
	assert.NotNil(t, metrics, "Metrics should not be nil")

	// Test IP whitelisting
	testIP := net.IPv4(192, 168, 1, 100)
	err = xdpManager.AddWhitelistedIP(testIP)
	assert.NoError(t, err, "Adding whitelisted IP should succeed")

	// Test IP removal
	err = xdpManager.RemoveWhitelistedIP(testIP)
	assert.NoError(t, err, "Removing whitelisted IP should succeed")

	// Test shutdown
	err = xdpManager.ShutdownXDP()
	assert.NoError(t, err, "XDP manager shutdown should succeed")
}

// TestVirtualContainerManagerBasic tests basic virtual container manager functionality
func TestVirtualContainerManagerBasic(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping virtual container basic test in short mode")
	}

	// eBPF operations require root privileges
	if os.Getuid() != 0 {
		t.Skip("Skipping virtual container test: requires root privileges (eBPF operations)")
	}

	// Create a mock manager
	manager := NewManager()
	vcManager := NewVirtualContainerManager(manager)

	// Test initialization
	err := vcManager.InitializeVirtualContainers()
	assert.NoError(t, err, "Virtual container manager initialization should succeed")

	// Test container creation
	container, err := vcManager.CreateVirtualContainer(1234, "/tmp/test-rootfs")
	assert.NoError(t, err, "Creating virtual container should succeed")
	assert.NotNil(t, container, "Container should not be nil")
	assert.Equal(t, uint32(1234), container.RootPID, "Root PID should match")
	assert.Equal(t, "/tmp/test-rootfs", container.RootFS, "RootFS should match")

	// Test container retrieval
	retrievedContainer, err := vcManager.GetVirtualContainer(container.ID)
	assert.NoError(t, err, "Getting virtual container should succeed")
	assert.Equal(t, container.ID, retrievedContainer.ID, "Container IDs should match")

	// Test container listing
	containers, err := vcManager.ListVirtualContainers()
	assert.NoError(t, err, "Listing virtual containers should succeed")
	assert.Len(t, containers, 1, "Should have one container")

	// Test network access control
	err = vcManager.SetVirtualContainerNetworkAccess(container.ID, true)
	assert.NoError(t, err, "Setting network access should succeed")

	retrievedContainer, err = vcManager.GetVirtualContainer(container.ID)
	assert.NoError(t, err, "Getting virtual container should succeed")
	assert.True(t, retrievedContainer.NetworkAllowed, "Network should be allowed")

	// Test container destruction
	err = vcManager.DestroyVirtualContainer(container.ID)
	assert.NoError(t, err, "Destroying virtual container should succeed")

	// Test shutdown
	err = vcManager.ShutdownVirtualContainers()
	assert.NoError(t, err, "Virtual container manager shutdown should succeed")
}

// TestXDPTypes tests the XDP data types
func TestXDPTypes(t *testing.T) {
	// Test NetworkMetrics
	metrics := &NetworkMetrics{
		DroppedPackets: 100,
		AllowedPackets: 1000,
		TopAttackers: []IPStats{
			{
				IP:            net.IPv4(192, 168, 1, 1),
				PacketsDropped: 50,
				Timestamp:     time.Now(),
			},
		},
	}

	assert.Equal(t, uint64(100), metrics.DroppedPackets, "Dropped packets should match")
	assert.Equal(t, uint64(1000), metrics.AllowedPackets, "Allowed packets should match")
	assert.Len(t, metrics.TopAttackers, 1, "Should have one attacker")

	// Test VirtualContainer
	container := &VirtualContainer{
		ID:            1234,
		RootPID:       5678,
		RootFS:        "/tmp/test-rootfs",
		NetworkAllowed: true,
	}

	assert.Equal(t, uint64(1234), container.ID, "Container ID should match")
	assert.Equal(t, uint32(5678), container.RootPID, "Root PID should match")
	assert.Equal(t, "/tmp/test-rootfs", container.RootFS, "RootFS should match")
	assert.True(t, container.NetworkAllowed, "Network should be allowed")
}

// TestXDPManagerErrorHandling tests XDP manager error handling
func TestXDPManagerErrorHandling(t *testing.T) {
	// Create XDP manager without initialization
	manager := NewManager()
	xdpManager := NewXDPManager(manager)

	// Test metrics retrieval before initialization
	metrics, err := xdpManager.GetNetworkMetrics()
	assert.Error(t, err, "Getting metrics before initialization should fail")
	assert.NotNil(t, metrics, "Metrics should still be returned (empty)")

	// Test IP operations before initialization
	testIP := net.IPv4(192, 168, 1, 100)
	err = xdpManager.AddWhitelistedIP(testIP)
	assert.Error(t, err, "Adding IP before initialization should fail")

	err = xdpManager.RemoveWhitelistedIP(testIP)
	assert.Error(t, err, "Removing IP before initialization should fail")

	// Test shutdown before initialization
	err = xdpManager.ShutdownXDP()
	assert.NoError(t, err, "Shutdown should succeed even without initialization")
}

// TestVirtualContainerManagerErrorHandling tests virtual container manager error handling
func TestVirtualContainerManagerErrorHandling(t *testing.T) {
	// Create virtual container manager without initialization
	manager := NewManager()
	vcManager := NewVirtualContainerManager(manager)

	// Test container operations before initialization
	container, err := vcManager.CreateVirtualContainer(1234, "/tmp/test-rootfs")
	assert.Error(t, err, "Creating container before initialization should fail")
	assert.Nil(t, container, "Container should be nil on error")

	container, err = vcManager.GetVirtualContainer(1234)
	assert.Error(t, err, "Getting container before initialization should fail")
	assert.Nil(t, container, "Container should be nil on error")

	containers, err := vcManager.ListVirtualContainers()
	assert.Error(t, err, "Listing containers before initialization should fail")
	assert.Nil(t, containers, "Containers list should be nil on error")

	// Test shutdown before initialization
	err = vcManager.ShutdownVirtualContainers()
	assert.NoError(t, err, "Shutdown should succeed even without initialization")
}

// TestXDPPerformance tests XDP performance with multiple operations
func TestXDPPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// eBPF operations require root privileges
	if os.Getuid() != 0 {
		t.Skip("Skipping XDP performance test: requires root privileges (eBPF operations)")
	}

	// Create XDP manager
	manager := NewManager()
	xdpManager := NewXDPManager(manager)
	err := xdpManager.InitializeXDP()
	require.NoError(t, err, "XDP manager initialization should succeed")
	defer xdpManager.ShutdownXDP()

	// Test multiple IP additions and removals
	startTime := time.Now()
	batchSize := 10

	for i := 0; i < batchSize; i++ {
		testIP := net.IPv4(192, 168, 1, byte(i))
		err := xdpManager.AddWhitelistedIP(testIP)
		assert.NoError(t, err, "Adding whitelisted IP should succeed")
	}

	// Measure performance
	elapsed := time.Since(startTime)
	avgTimePerOp := elapsed / time.Duration(batchSize)

	// Verify performance is reasonable
	assert.Less(t, avgTimePerOp, 10*time.Millisecond, "Average operation time should be less than 10ms")

	// Clean up
	for i := 0; i < batchSize; i++ {
		testIP := net.IPv4(192, 168, 1, byte(i))
		err := xdpManager.RemoveWhitelistedIP(testIP)
		assert.NoError(t, err, "Removing whitelisted IP should succeed")
	}
}

// TestVirtualContainerPerformance tests virtual container performance
func TestVirtualContainerPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// eBPF operations require root privileges
	if os.Getuid() != 0 {
		t.Skip("Skipping virtual container performance test: requires root privileges (eBPF operations)")
	}

	// Create virtual container manager
	manager := NewManager()
	vcManager := NewVirtualContainerManager(manager)
	err := vcManager.InitializeVirtualContainers()
	require.NoError(t, err, "Virtual container manager initialization should succeed")
	defer vcManager.ShutdownVirtualContainers()

	// Test creating multiple containers
	startTime := time.Now()
	containerCount := 5
	var containerIDs []uint64

	for i := 0; i < containerCount; i++ {
		container, err := vcManager.CreateVirtualContainer(uint32(1000+i), fmt.Sprintf("/tmp/test-rootfs-%d", i))
		assert.NoError(t, err, "Creating virtual container should succeed")
		containerIDs = append(containerIDs, container.ID)
	}

	// Measure performance
	elapsed := time.Since(startTime)
	avgTimePerOp := elapsed / time.Duration(containerCount)

	// Verify performance is reasonable
	assert.Less(t, avgTimePerOp, 5*time.Millisecond, "Average container creation time should be less than 5ms")

	// Test listing all containers
	containers, err := vcManager.ListVirtualContainers()
	assert.NoError(t, err, "Listing virtual containers should succeed")
	assert.Len(t, containers, containerCount, "Should have correct number of containers")

	// Clean up
	for _, id := range containerIDs {
		err := vcManager.DestroyVirtualContainer(id)
		assert.NoError(t, err, "Destroying virtual container should succeed")
	}

	// Verify cleanup
	finalContainers, err := vcManager.ListVirtualContainers()
	assert.NoError(t, err, "Listing virtual containers should succeed")
	assert.Len(t, finalContainers, 0, "Should have no containers after cleanup")
}