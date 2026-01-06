// Copyright 2026 KNIRV-NEXUS
// SPDX-License-Identifier: GPL-3.0-or-later

package ebpf

import (
	"fmt"
	"log"
	"sync/atomic"

	"github.com/cilium/ebpf"
)

// VirtualContainerManager handles eBPF-based virtual containers
type VirtualContainerManager struct {
	manager        *Manager
	collection     *ebpf.Collection
	containersMap  *ebpf.Map
	pidToContainer *ebpf.Map
	nextContainerID uint64
}

// NewVirtualContainerManager creates a new Virtual Container Manager instance
func NewVirtualContainerManager(manager *Manager) *VirtualContainerManager {
	return &VirtualContainerManager{
		manager: manager,
	}
}

// InitializeVirtualContainers initializes virtual container maps
func (v *VirtualContainerManager) InitializeVirtualContainers() error {
	// Create virtual container maps
	spec := &ebpf.CollectionSpec{
		Maps: map[string]*ebpf.MapSpec{
			"containers": {
				Type:       ebpf.Hash,
				KeySize:    8, // uint64
				ValueSize:  272, // struct virtual_container
				MaxEntries: 1000,
			},
			"pid_to_container": {
				Type:       ebpf.Hash,
				KeySize:    4, // uint32
				ValueSize:  8, // uint64
				MaxEntries: 10000,
			},
		},
	}

	coll, err := ebpf.NewCollection(spec)
	if err != nil {
		return fmt.Errorf("create virtual container collection: %w", err)
	}

	v.collection = coll
	v.containersMap = coll.Maps["containers"]
	v.pidToContainer = coll.Maps["pid_to_container"]

	log.Println("Virtual Container Manager initialized successfully")
	return nil
}

// CreateVirtualContainer creates a new eBPF-based virtual container
func (v *VirtualContainerManager) CreateVirtualContainer(rootPID uint32, rootFS string) (*VirtualContainer, error) {
	// Check if initialized
	if v.collection == nil {
		return nil, fmt.Errorf("virtual container manager not initialized")
	}

	containerID := atomic.AddUint64(&v.nextContainerID, 1)

	container := &VirtualContainer{
		ID:            containerID,
		RootPID:       rootPID,
		RootFS:        rootFS,
		NetworkAllowed: false,
	}

	// Add to containers map
	var kContainer struct {
		ContainerID    uint64
		RootPID        uint32
		RootFS         [256]byte
		NetworkAllowed uint32
	}
	kContainer.ContainerID = containerID
	kContainer.RootPID = rootPID
	copy(kContainer.RootFS[:], rootFS)

	containersMap := v.collection.Maps["containers"]
	if err := containersMap.Put(containerID, kContainer); err != nil {
		return nil, err
	}

	// Map PID to container
	pidMap := v.collection.Maps["pid_to_container"]
	if err := pidMap.Put(rootPID, containerID); err != nil {
		return nil, err
	}

	return container, nil
}

// DestroyVirtualContainer destroys a virtual container
func (v *VirtualContainerManager) DestroyVirtualContainer(id uint64) error {
	// Remove from containers map
	if err := v.containersMap.Delete(id); err != nil {
		return fmt.Errorf("delete container: %w", err)
	}

	// Note: PIDs are automatically cleaned up as processes exit
	return nil
}

// GetVirtualContainer retrieves a virtual container by ID
func (v *VirtualContainerManager) GetVirtualContainer(id uint64) (*VirtualContainer, error) {
	// Check if initialized
	if v.containersMap == nil {
		return nil, fmt.Errorf("virtual container manager not initialized")
	}

	var kContainer struct {
		ContainerID    uint64
		RootPID        uint32
		RootFS         [256]byte
		NetworkAllowed uint32
	}

	if err := v.containersMap.Lookup(id, &kContainer); err != nil {
		return nil, fmt.Errorf("lookup container: %w", err)
	}

	container := &VirtualContainer{
		ID:            kContainer.ContainerID,
		RootPID:       kContainer.RootPID,
		RootFS:        string(kContainer.RootFS[:]),
		NetworkAllowed: kContainer.NetworkAllowed == 1,
	}

	return container, nil
}

// ListVirtualContainers lists all virtual containers
func (v *VirtualContainerManager) ListVirtualContainers() ([]*VirtualContainer, error) {
	// Check if initialized
	if v.containersMap == nil {
		return nil, fmt.Errorf("virtual container manager not initialized")
	}

	var containers []*VirtualContainer
	var key, nextKey uint64

	for {
		if err := v.containersMap.NextKey(&key, &nextKey); err != nil {
			break
		}

		container, err := v.GetVirtualContainer(nextKey)
		if err != nil {
			continue
		}

		containers = append(containers, container)
		key = nextKey
	}

	return containers, nil
}

// SetVirtualContainerNetworkAccess sets network access for a virtual container
func (v *VirtualContainerManager) SetVirtualContainerNetworkAccess(id uint64, allowed bool) error {
	var kContainer struct {
		ContainerID    uint64
		RootPID        uint32
		RootFS         [256]byte
		NetworkAllowed uint32
	}

	if err := v.containersMap.Lookup(id, &kContainer); err != nil {
		return fmt.Errorf("lookup container: %w", err)
	}

	kContainer.NetworkAllowed = 0
	if allowed {
		kContainer.NetworkAllowed = 1
	}

	return v.containersMap.Put(id, kContainer)
}

// ShutdownVirtualContainers cleans up virtual container resources
func (v *VirtualContainerManager) ShutdownVirtualContainers() error {
	// Close collection
	if v.collection != nil {
		v.collection.Close()
		v.collection = nil
	}

	log.Println("Virtual Container Manager shutdown complete")
	return nil
}