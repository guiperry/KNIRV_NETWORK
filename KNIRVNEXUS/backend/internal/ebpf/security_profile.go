// Copyright 2026 KNIRV-NEXUS
// SPDX-License-Identifier: GPL-3.0-or-later

package ebpf

import (
	"fmt"
	"log"
	"sync"

	"golang.org/x/sys/unix"
)

type SecurityPolicy struct {
	Name                  string
	ContainerID           uint64
	AllowedSyscalls       map[uint32]bool
	AllowedPaths          map[string]bool
	AllowedNetworkNets    map[string]bool
	DenyByDefault         bool
	EnableNetwork         bool
	EnableFilesystem      bool
	EnableCapabilities    bool
	MaxMemoryBytes        uint64
	MaxCPUPercent         int
	RequireTEEAttestation bool
	CreatedAt             int64
	Version               uint32
	mu                    sync.RWMutex
}

type SecurityProfileManager struct {
	policies      map[uint64]*SecurityPolicy
	defaultPolicy *SecurityPolicy
	mu            sync.RWMutex
}

func NewSecurityProfileManager() *SecurityProfileManager {
	spm := &SecurityProfileManager{
		policies: make(map[uint64]*SecurityPolicy),
	}
	spm.defaultPolicy = spm.createDenyByDefaultPolicy()
	return spm
}

func (spm *SecurityProfileManager) createDenyByDefaultPolicy() *SecurityPolicy {
	return &SecurityPolicy{
		Name:                  "knirv-deny-by-default",
		AllowedSyscalls:       spm.getMinimalSyscalls(),
		AllowedPaths:          map[string]bool{},
		AllowedNetworkNets:    map[string]bool{},
		DenyByDefault:         true,
		EnableNetwork:         false,
		EnableFilesystem:      false,
		EnableCapabilities:    false,
		MaxMemoryBytes:        512 * 1024 * 1024,
		MaxCPUPercent:         100,
		RequireTEEAttestation: true,
	}
}

func (spm *SecurityProfileManager) getMinimalSyscalls() map[uint32]bool {
	return map[uint32]bool{
		1:   true, // exit
		9:   true, // mmap
		10:  true, // munmap
		11:  true, // brk
		12:  true, // rt_sigaction
		14:  true, // rt_sigprocmask
		21:  true, // access
		35:  true, // nanosleep
		60:  true, // exit_group
		231: true, // clock_nanosleep
		257: true, // openat
		259: true, // read
		262: true, // newfstatat
		280: true, // utimensat
		281: true, // write
	}
}

func (spm *SecurityProfileManager) CreateAgentPolicy(containerID uint64, config *AgentPolicyConfig) (*SecurityPolicy, error) {
	spm.mu.Lock()
	defer spm.mu.Unlock()

	policy := &SecurityPolicy{
		Name:                  fmt.Sprintf("agent-policy-%d", containerID),
		ContainerID:           containerID,
		AllowedSyscalls:       make(map[uint32]bool),
		AllowedPaths:          make(map[string]bool),
		AllowedNetworkNets:    make(map[string]bool),
		DenyByDefault:         true,
		EnableNetwork:         config.AllowNetwork,
		EnableFilesystem:      config.AllowFilesystem,
		EnableCapabilities:    false,
		MaxMemoryBytes:        config.MaxMemoryMB * 1024 * 1024,
		MaxCPUPercent:         config.MaxCPUPercent,
		RequireTEEAttestation: config.RequireTEE,
		CreatedAt:             nowUnix(),
		Version:               1,
	}

	for _, sc := range config.AllowedSyscalls {
		policy.AllowedSyscalls[sc] = true
	}

	if len(config.AllowedPaths) > 0 {
		for _, p := range config.AllowedPaths {
			policy.AllowedPaths[p] = true
		}
	} else {
		policy.AllowedPaths["/tmp"] = true
		policy.AllowedPaths["/proc/self"] = true
	}

	if config.AllowNetwork {
		policy.AllowedNetworkNets["127.0.0.0/8"] = true
		for _, net := range config.AllowedNetworks {
			policy.AllowedNetworkNets[net] = true
		}
	}

	spm.policies[containerID] = policy
	log.Printf("SecurityProfileManager: Created agent policy for container %d", containerID)

	return policy, nil
}

func (spm *SecurityProfileManager) GetPolicy(containerID uint64) *SecurityPolicy {
	spm.mu.RLock()
	defer spm.mu.RUnlock()

	if policy, exists := spm.policies[containerID]; exists {
		return policy
	}
	return spm.defaultPolicy
}

func (spm *SecurityProfileManager) UpdatePolicy(containerID uint64, updates *PolicyUpdates) error {
	spm.mu.Lock()
	defer spm.mu.Unlock()

	policy, exists := spm.policies[containerID]
	if !exists {
		return fmt.Errorf("policy not found for container %d", containerID)
	}

	if updates.AddSyscalls != nil {
		for _, sc := range updates.AddSyscalls {
			policy.AllowedSyscalls[sc] = true
		}
	}

	if updates.RemoveSyscalls != nil {
		for _, sc := range updates.RemoveSyscalls {
			delete(policy.AllowedSyscalls, sc)
		}
	}

	if updates.AddPaths != nil {
		for _, p := range updates.AddPaths {
			policy.AllowedPaths[p] = true
		}
	}

	if updates.EnableNetwork != nil {
		policy.EnableNetwork = *updates.EnableNetwork
	}

	policy.Version++
	log.Printf("SecurityProfileManager: Updated policy v%d for container %d", policy.Version, containerID)
	return nil
}

func (spm *SecurityProfileManager) DeletePolicy(containerID uint64) error {
	spm.mu.Lock()
	defer spm.mu.Unlock()

	if _, exists := spm.policies[containerID]; !exists {
		return fmt.Errorf("policy not found for container %d", containerID)
	}

	delete(spm.policies, containerID)
	log.Printf("SecurityProfileManager: Deleted policy for container %d", containerID)
	return nil
}

func (spm *SecurityProfileManager) IsSyscallAllowed(containerID uint64, syscallNr uint32) bool {
	policy := spm.GetPolicy(containerID)
	if policy == nil {
		return false
	}

	policy.mu.RLock()
	defer policy.mu.RUnlock()

	if policy.DenyByDefault {
		_, allowed := policy.AllowedSyscalls[syscallNr]
		return allowed
	}
	return true
}

func (spm *SecurityProfileManager) IsPathAllowed(containerID uint64, path string) bool {
	policy := spm.GetPolicy(containerID)
	if policy == nil {
		return false
	}

	policy.mu.RLock()
	defer policy.mu.RUnlock()

	if !policy.EnableFilesystem {
		return false
	}

	if !policy.DenyByDefault {
		return true
	}

	for allowedPath, _ := range policy.AllowedPaths {
		if len(path) >= len(allowedPath) && path[:len(allowedPath)] == allowedPath {
			return true
		}
	}
	return false
}

func (spm *SecurityProfileManager) IsNetworkAllowed(containerID uint64, ip string) bool {
	policy := spm.GetPolicy(containerID)
	if policy == nil {
		return false
	}

	policy.mu.RLock()
	defer policy.mu.RUnlock()

	if !policy.EnableNetwork {
		return false
	}

	if !policy.DenyByDefault {
		return true
	}

	for net, _ := range policy.AllowedNetworkNets {
		if matchesCIDR(ip, net) {
			return true
		}
	}
	return false
}

func (spm *SecurityProfileManager) GetAllPolicies() []*SecurityPolicy {
	spm.mu.RLock()
	defer spm.mu.RUnlock()

	policies := make([]*SecurityPolicy, 0, len(spm.policies))
	for _, p := range spm.policies {
		policies = append(policies, p)
	}
	return policies
}

type AgentPolicyConfig struct {
	AllowNetwork    bool
	AllowFilesystem bool
	RequireTEE      bool
	MaxMemoryMB     uint64
	MaxCPUPercent   int
	AllowedSyscalls []uint32
	AllowedPaths    []string
	AllowedNetworks []string
}

type PolicyUpdates struct {
	AddSyscalls    []uint32
	RemoveSyscalls []uint32
	AddPaths       []string
	RemovePaths    []string
	EnableNetwork  *bool
}

func matchesCIDR(ip, cidr string) bool {
	if cidr == "0.0.0.0/0" {
		return true
	}
	return ip == cidr
}

func nowUnix() int64 {
	return int64(unixTimeNow())
}

func unixTimeNow() uint64 {
	var ts unix.Timespec
	unix.ClockGettime(unix.CLOCK_REALTIME, &ts)
	return uint64(ts.Sec)*1e9 + uint64(ts.Nsec)
}
