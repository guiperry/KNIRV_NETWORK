// Copyright 2026 KNIRV-NEXUS
// SPDX-License-Identifier: GPL-3.0-or-later

package ebpf

import (
	"fmt"
	"log"
)

// PolicyManager handles sandbox policies for containers
type PolicyManager struct {
	manager *Manager
}

// NewPolicyManager creates a new PolicyManager
func NewPolicyManager(manager *Manager) *PolicyManager {
	return &PolicyManager{manager: manager}
}

// SetSandboxPolicy sets a security policy for a container
func (pm *PolicyManager) SetSandboxPolicy(containerID uint64, policy *SandboxPolicy) error {
	if pm.manager.collection == nil {
		return fmt.Errorf("eBPF collection not initialized")
	}

	policiesMap := pm.manager.collection.Maps["policies"]
	if policiesMap == nil {
		return fmt.Errorf("policies map not found in collection")
	}

	var kPolicy struct {
		AllowedPrefix  [256]byte
		NetworkAllowed uint32
	}

	// Copy allowed path prefix
	copy(kPolicy.AllowedPrefix[:], policy.AllowedPathPrefix)
	if policy.NetworkAllowed {
		kPolicy.NetworkAllowed = 1
	}

	if err := policiesMap.Put(containerID, kPolicy); err != nil {
		return fmt.Errorf("set sandbox policy: %w", err)
	}

	log.Printf("Set sandbox policy for container %d: path=%s, network=%v",
		containerID, policy.AllowedPathPrefix, policy.NetworkAllowed)

	return nil
}

// RemoveSandboxPolicy removes a security policy for a container
func (pm *PolicyManager) RemoveSandboxPolicy(containerID uint64) error {
	if pm.manager.collection == nil {
		return fmt.Errorf("eBPF collection not initialized")
	}

	policiesMap := pm.manager.collection.Maps["policies"]
	if policiesMap == nil {
		return fmt.Errorf("policies map not found in collection")
	}

	if err := policiesMap.Delete(containerID); err != nil {
		return fmt.Errorf("remove sandbox policy: %w", err)
	}

	log.Printf("Removed sandbox policy for container %d", containerID)
	return nil
}

// GetSandboxPolicy retrieves a security policy for a container
func (pm *PolicyManager) GetSandboxPolicy(containerID uint64) (*SandboxPolicy, error) {
	if pm.manager.collection == nil {
		return nil, fmt.Errorf("eBPF collection not initialized")
	}

	policiesMap := pm.manager.collection.Maps["policies"]
	if policiesMap == nil {
		return nil, fmt.Errorf("policies map not found in collection")
	}

	var kPolicy struct {
		AllowedPrefix  [256]byte
		NetworkAllowed uint32
	}

	if err := policiesMap.Lookup(containerID, &kPolicy); err != nil {
		return nil, fmt.Errorf("get sandbox policy: %w", err)
	}

	policy := &SandboxPolicy{
		AllowedPathPrefix: string(kPolicy.AllowedPrefix[:]),
		NetworkAllowed:    kPolicy.NetworkAllowed == 1,
	}

	return policy, nil
}