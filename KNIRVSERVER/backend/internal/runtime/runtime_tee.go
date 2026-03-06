// Copyright 2026 KNIRV-SERVER
// SPDX-License-Identifier: GPL-3.0-or-later

package runtime

import (
	"fmt"
	"log"
)

// TEERuntime implements ContainerRuntime using hardware TEE
type TEERuntime struct {
	teeType     string // sgx, sev, tdx
	containerID string
	spec        *ContainerSpec
	status      ContainerStatus
}

// NewTEERuntime creates a new TEE runtime instance
func NewTEERuntime(teeType string) ContainerRuntime {
	return &TEERuntime{
		teeType: teeType,
		status:  ContainerStatusCreating,
	}
}

// Create creates a TEE container
func (tr *TEERuntime) Create(spec *ContainerSpec) error {
	if spec == nil {
		return fmt.Errorf("container spec is required")
	}

	tr.spec = spec
	tr.containerID = spec.ID

	// TEE runtime would integrate with hardware-specific APIs
	// This is a placeholder for actual TEE integration
	log.Printf("TEE container created (type: %s): %s", tr.teeType, spec.ID)

	tr.status = ContainerStatusCreating

	return nil
}

// Start starts the TEE container
func (tr *TEERuntime) Start() error {
	if tr.containerID == "" {
		return fmt.Errorf("container not created")
	}

	tr.status = ContainerStatusRunning

	log.Printf("TEE container started (type: %s): %s", tr.teeType, tr.containerID)

	return nil
}

// Stop stops the TEE container
func (tr *TEERuntime) Stop() error {
	if tr.containerID == "" {
		return fmt.Errorf("container not created")
	}

	tr.status = ContainerStatusStopped

	log.Printf("TEE container stopped (type: %s): %s", tr.teeType, tr.containerID)

	return nil
}

// Delete removes the TEE container
func (tr *TEERuntime) Delete() error {
	if tr.containerID == "" {
		return fmt.Errorf("container not created")
	}

	log.Printf("TEE container deleted (type: %s): %s", tr.teeType, tr.containerID)

	return nil
}

// GetStatus returns the current container status
func (tr *TEERuntime) GetStatus() ContainerStatus {
	return tr.status
}
