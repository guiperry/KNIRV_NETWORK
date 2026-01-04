// Copyright 2026 KNIRV-NEXUS
// SPDX-License-Identifier: GPL-3.0-or-later

package ebpf

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestManagerInitialization(t *testing.T) {
	mgr := NewManager()

	err := mgr.Initialize(context.Background(), &Config{
		Programs: []ProgramConfig{
			{Name: "syscall_trace", Enabled: true},
		},
	})
	require.NoError(t, err)
	defer mgr.Shutdown()

	require.NotNil(t, mgr.GetMetrics())
	require.True(t, mgr.GetMetrics().Initialized)
}

func TestEventCollection(t *testing.T) {
	mgr := NewManager()
	err := mgr.Initialize(context.Background(), &Config{
		Programs: []ProgramConfig{
			{Name: "syscall_trace", Enabled: true},
		},
	})
	require.NoError(t, err)
	defer mgr.Shutdown()

	// Create event collector
	collector := NewEventCollector(mgr)

	// Set up event handler
	eventsReceived := 0
	collector.Subscribe(func(event *SyscallEvent) error {
		eventsReceived++
		return nil
	})

	// Start collection
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err = collector.Start(ctx)
	require.NoError(t, err)

	// Wait for collection to complete
	time.Sleep(100 * time.Millisecond)
	collector.Stop()

	// We should have received at least some events
	t.Logf("Received %d events", eventsReceived)
}

func TestPolicyManagement(t *testing.T) {
	mgr := NewManager()
	err := mgr.Initialize(context.Background(), &Config{
		Programs: []ProgramConfig{
			{Name: "sandbox_lsm", Enabled: true},
		},
	})
	require.NoError(t, err)
	defer mgr.Shutdown()

	// Create policy manager
	policyMgr := NewPolicyManager(mgr)

	// Test setting a policy
	containerID := uint64(1234)
	policy := &SandboxPolicy{
		AllowedPathPrefix: "/tmp/test",
		NetworkAllowed:    false,
	}

	err = policyMgr.SetSandboxPolicy(containerID, policy)
	require.NoError(t, err)

	// Test retrieving the policy
	retrievedPolicy, err := policyMgr.GetSandboxPolicy(containerID)
	require.NoError(t, err)
	require.Equal(t, policy.AllowedPathPrefix, retrievedPolicy.AllowedPathPrefix)
	require.Equal(t, policy.NetworkAllowed, retrievedPolicy.NetworkAllowed)

	// Test removing the policy
	err = policyMgr.RemoveSandboxPolicy(containerID)
	require.NoError(t, err)

	// Verify policy is removed
	_, err = policyMgr.GetSandboxPolicy(containerID)
	require.Error(t, err) // Should get an error since policy is removed
}
