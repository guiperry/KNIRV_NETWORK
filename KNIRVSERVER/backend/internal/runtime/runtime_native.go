// Copyright 2026 KNIRV-SERVER
// SPDX-License-Identifier: GPL-3.0-or-later

package runtime

import (
	"fmt"
	"log"
)

// NativeRuntime implements ContainerRuntime using native Go isolation
type NativeRuntime struct {
	containerID string
	spec        *ContainerSpec
	status      ContainerStatus
}

// NewNativeRuntime creates a new native runtime instance
func NewNativeRuntime() ContainerRuntime {
	return &NativeRuntime{
		status: ContainerStatusCreating,
	}
}

// Create creates a native container with Go isolation
func (nr *NativeRuntime) Create(spec *ContainerSpec) error {
	if spec == nil {
		return fmt.Errorf("container spec is required")
	}

	nr.spec = spec
	nr.containerID = spec.ID

	// Native runtime uses Linux namespaces and cgroups directly
	// This is a placeholder for actual namespace/cgroup setup
	log.Printf("Native container created: %s", spec.ID)

	nr.status = ContainerStatusCreating

	return nil
}

// Start starts the native container
func (nr *NativeRuntime) Start() error {
	if nr.containerID == "" {
		return fmt.Errorf("container not created")
	}

	nr.status = ContainerStatusRunning

	log.Printf("Native container started: %s", nr.containerID)

	return nil
}

// Stop stops the native container
func (nr *NativeRuntime) Stop() error {
	if nr.containerID == "" {
		return fmt.Errorf("container not created")
	}

	nr.status = ContainerStatusStopped

	log.Printf("Native container stopped: %s", nr.containerID)

	return nil
}

// Delete removes the native container
func (nr *NativeRuntime) Delete() error {
	if nr.containerID == "" {
		return fmt.Errorf("container not created")
	}

	log.Printf("Native container deleted: %s", nr.containerID)

	return nil
}

// GetStatus returns the current container status
func (nr *NativeRuntime) GetStatus() ContainerStatus {
	return nr.status
}
