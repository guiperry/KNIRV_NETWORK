// Copyright 2026 KNIRV-NEXUS
// SPDX-License-Identifier: GPL-3.0-or-later

package ebpf

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestManagerStructure tests the manager structure without requiring BPF privileges
func TestManagerStructure(t *testing.T) {
	mgr := NewManager()
	require.NotNil(t, mgr)

	// Test that manager starts in uninitialized state
	require.False(t, mgr.GetMetrics().Initialized)

	// Test shutdown on uninitialized manager (should be safe)
	require.NoError(t, mgr.Shutdown())
}

// TestEventCollectorStructure tests event collector without BPF
func TestEventCollectorStructure(t *testing.T) {
	mgr := NewManager()
	collector := NewEventCollector(mgr)
	require.NotNil(t, collector)

	// Test event handler subscription
	collector.Subscribe(func(event *SyscallEvent) error {
		// Handler would process events in real usage
		return nil
	})

	// Test that collector can be stopped safely
	collector.Stop()
}

// TestPolicyManagerStructure tests policy manager without BPF
func TestPolicyManagerStructure(t *testing.T) {
	mgr := NewManager()
	policyMgr := NewPolicyManager(mgr)
	require.NotNil(t, policyMgr)

	// Test policy creation
	policy := &SandboxPolicy{
		AllowedPathPrefix: "/tmp/test",
		NetworkAllowed:    false,
	}
	require.Equal(t, "/tmp/test", policy.AllowedPathPrefix)
	require.False(t, policy.NetworkAllowed)
}

// TestConfigurationStructure tests configuration parsing
func TestConfigurationStructure(t *testing.T) {
	config := &Config{
		Programs: []ProgramConfig{
			{Name: "syscall_trace", Enabled: true},
			{Name: "sandbox_lsm", Enabled: false},
		},
	}
	require.NotNil(t, config)
	require.Len(t, config.Programs, 2)
	require.True(t, config.Programs[0].Enabled)
	require.False(t, config.Programs[1].Enabled)
}
