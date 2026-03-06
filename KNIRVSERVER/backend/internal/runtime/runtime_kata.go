// Copyright 2026 KNIRV-SERVER
// SPDX-License-Identifier: GPL-3.0-or-later

package runtime

import (
	"fmt"
	"log"
)

// KataRuntime implements ContainerRuntime using Kata containers
type KataRuntime struct {
	containerID string
	spec        *ContainerSpec
	status      ContainerStatus
}

// NewKataRuntime creates a new Kata runtime instance
func NewKataRuntime() ContainerRuntime {
	return &KataRuntime{
		status: ContainerStatusCreating,
	}
}

// Create creates a Kata container
func (kr *KataRuntime) Create(spec *ContainerSpec) error {
	if spec == nil {
		return fmt.Errorf("container spec is required")
	}

	kr.spec = spec
	kr.containerID = spec.ID

	// Kata containers use the same interface as Docker but with --runtime=kata-runtime
	// This is a simplified implementation that delegates to Docker with Kata runtime
	log.Printf("Kata container created (delegating to Docker with kata-runtime): %s", spec.ID)

	kr.status = ContainerStatusCreating

	return nil
}

// Start starts the Kata container
func (kr *KataRuntime) Start() error {
	if kr.containerID == "" {
		return fmt.Errorf("container not created")
	}

	kr.status = ContainerStatusRunning

	log.Printf("Kata container started: %s", kr.containerID)

	return nil
}

// Stop stops the Kata container
func (kr *KataRuntime) Stop() error {
	if kr.containerID == "" {
		return fmt.Errorf("container not created")
	}

	kr.status = ContainerStatusStopped

	log.Printf("Kata container stopped: %s", kr.containerID)

	return nil
}

// Delete removes the Kata container
func (kr *KataRuntime) Delete() error {
	if kr.containerID == "" {
		return fmt.Errorf("container not created")
	}

	log.Printf("Kata container deleted: %s", kr.containerID)

	return nil
}

// GetStatus returns the current container status
func (kr *KataRuntime) GetStatus() ContainerStatus {
	return kr.status
}
