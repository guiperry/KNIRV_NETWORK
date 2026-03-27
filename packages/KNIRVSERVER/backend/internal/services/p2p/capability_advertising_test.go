// Copyright 2026 KNIRV-SERVER
// SPDX-License-Identifier: GPL-3.0-or-later

package p2p

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCapabilityAnnouncement(t *testing.T) {
	announcement := CapabilityAnnouncement{
		NodeID: "test-node-123",
		Capabilities: []NodeCapability{
			{
				Capability: CapabilityAgenticRuntime,
				Version:    "oh-my-pi-1.0",
				Metadata: map[string]interface{}{
					"supported_tools": []string{"git", "python", "curl"},
					"max_concurrent":  4,
				},
			},
		},
		Timestamp: 1700000000,
	}

	assert.Equal(t, "test-node-123", announcement.NodeID)
	assert.Len(t, announcement.Capabilities, 1)
	assert.Equal(t, CapabilityAgenticRuntime, announcement.Capabilities[0].Capability)
	assert.Equal(t, "oh-my-pi-1.0", announcement.Capabilities[0].Version)
}

func TestNodeCapability(t *testing.T) {
	cap := NodeCapability{
		Capability: CapabilityDVERouting,
		Version:    "1.0.0",
		Metadata: map[string]interface{}{
			"priority": 1,
		},
	}

	assert.Equal(t, CapabilityDVERouting, cap.Capability)
	assert.Equal(t, "1.0.0", cap.Version)
	assert.Equal(t, 1, cap.Metadata["priority"])
}

func TestCapabilityConstants(t *testing.T) {
	assert.Equal(t, "agentic-runtime-support", CapabilityAgenticRuntime)
	assert.Equal(t, "dve-routing", CapabilityDVERouting)
	assert.Equal(t, "tee-attestation", CapabilityTEEAttestation)
}
