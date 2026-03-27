// Copyright 2026 KNIRV-SERVER
// SPDX-License-Identifier: GPL-3.0-or-later

package runtime

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOhMyPiAgentSyscalls_ContainsEssentialSyscalls(t *testing.T) {
	// Verify essential syscalls for agent execution are included
	essentialSyscalls := map[uint32]string{
		56:  "clone",
		57:  "fork",
		58:  "vfork",
		59:  "execve",
		41:  "socket",
		42:  "connect",
		43:  "accept",
		257: "openat",
		259: "read",
		281: "write",
	}

	for syscallID, name := range essentialSyscalls {
		found := false
		for _, s := range OhMyPiAgentSyscalls {
			if s == syscallID {
				found = true
				break
			}
		}
		assert.True(t, found, "Essential syscall %s (%d) should be in OhMyPiAgentSyscalls", name, syscallID)
	}
}

func TestOhMyPiAgentPaths_ContainsWorkspacePaths(t *testing.T) {
	requiredPaths := []string{
		"/workspace",
		"/tmp",
		"/usr",
		"/lib",
		"/bin",
	}

	for _, path := range requiredPaths {
		found := false
		for _, p := range OhMyPiAgentPaths {
			if p == path {
				found = true
				break
			}
		}
		assert.True(t, found, "Required path %s should be in OhMyPiAgentPaths", path)
	}
}

func TestOhMyPiAgentNetworks_ContainsInternalNetworks(t *testing.T) {
	requiredNetworks := []string{
		"127.0.0.0/8",
		"10.0.0.0/8",
		"172.16.0.0/12",
	}

	for _, network := range requiredNetworks {
		found := false
		for _, n := range OhMyPiAgentNetworks {
			if n == network {
				found = true
				break
			}
		}
		assert.True(t, found, "Required network %s should be in OhMyPiAgentNetworks", network)
	}
}

func TestNewOhMyPiAgentPolicyConfig(t *testing.T) {
	policy := NewOhMyPiAgentPolicyConfig()

	assert.NotNil(t, policy)
	assert.True(t, policy.AllowNetwork, "Agent should have network access")
	assert.True(t, policy.AllowFilesystem, "Agent should have filesystem access")
	assert.False(t, policy.RequireTEE, "TEE should not be required for agents")
	assert.Equal(t, uint64(8192), policy.MaxMemoryMB, "Agent should have 8GB memory limit")
	assert.Equal(t, 80, policy.MaxCPUPercent, "Agent should have 80% CPU limit")
	assert.NotEmpty(t, policy.AllowedSyscalls, "Agent should have allowed syscalls")
	assert.NotEmpty(t, policy.AllowedPaths, "Agent should have allowed paths")
	assert.NotEmpty(t, policy.AllowedNetworks, "Agent should have allowed networks")
}

func TestOhMyPiAgentSyscalls_Length(t *testing.T) {
	// Verify we have a reasonable number of syscalls for agent tools
	// We need git, python, curl, browser, and LSP server support
	assert.GreaterOrEqual(t, len(OhMyPiAgentSyscalls), 50,
		"Should have at least 50 syscalls for comprehensive tool support")
}

func TestOhMyPiAgentPaths_Length(t *testing.T) {
	// Verify we have necessary paths for agent operation
	assert.GreaterOrEqual(t, len(OhMyPiAgentPaths), 10,
		"Should have at least 10 allowed paths")
}

func TestOhMyPiAgentNetworks_Length(t *testing.T) {
	// Verify we have internal and loopback networks
	assert.GreaterOrEqual(t, len(OhMyPiAgentNetworks), 3,
		"Should have at least 3 allowed networks")
}
