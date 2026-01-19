// Copyright 2026 KNIRV-NEXUS
// SPDX-License-Identifier: GPL-3.0-or-later

package runtime

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
)

// DockerRuntime implements ContainerRuntime using Docker
type DockerRuntime struct {
	containerID string
	spec        *ContainerSpec
	status      ContainerStatus
}

// NewDockerRuntime creates a new Docker runtime instance
func NewDockerRuntime() ContainerRuntime {
	return &DockerRuntime{
		status: ContainerStatusCreating,
	}
}

// Create creates a Docker container
func (dr *DockerRuntime) Create(spec *ContainerSpec) error {
	if spec == nil {
		return fmt.Errorf("container spec is required")
	}

	dr.spec = spec

	// Build docker run command
	args := []string{"create"}

	// Add resource limits
	if spec.Resources.CPUCores > 0 {
		args = append(args, "--cpus", fmt.Sprintf("%d", spec.Resources.CPUCores))
	}
	if spec.Resources.MemoryMB > 0 {
		args = append(args, "--memory", fmt.Sprintf("%dm", spec.Resources.MemoryMB))
	}

	// Add network mode
	if spec.NetworkMode != "" {
		args = append(args, "--network", spec.NetworkMode)
	}

	// Add port mappings
	for _, port := range spec.Ports {
		if port.HostPort > 0 {
			args = append(args, "-p", fmt.Sprintf("%d:%d/%s", port.HostPort, port.ContainerPort, port.Protocol))
		} else {
			args = append(args, "-p", fmt.Sprintf("%d/%s", port.ContainerPort, port.Protocol))
		}
	}

	// Add environment variables
	for k, v := range spec.Environment {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	// Add volume mounts
	for _, vol := range spec.Volumes {
		mountArg := fmt.Sprintf("%s:%s", vol.Source, vol.Target)
		if vol.ReadOnly {
			mountArg += ":ro"
		}
		args = append(args, "-v", mountArg)
	}

	// Add capabilities
	for _, cap := range spec.CapabilitySet {
		args = append(args, "--cap-add", cap)
	}

	// Add container name
	args = append(args, "--name", spec.ID)

	// Add image
	args = append(args, spec.Image)

	// Add command
	if len(spec.Command) > 0 {
		args = append(args, spec.Command...)
	}

	// Execute docker create
	cmd := exec.Command("docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker create failed: %w: %s", err, string(output))
	}

	dr.containerID = strings.TrimSpace(string(output))
	dr.status = ContainerStatusCreating

	log.Printf("Docker container created: %s (ID: %s)", spec.ID, dr.containerID)

	return nil
}

// Start starts the Docker container
func (dr *DockerRuntime) Start() error {
	if dr.containerID == "" {
		return fmt.Errorf("container not created")
	}

	cmd := exec.Command("docker", "start", dr.containerID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker start failed: %w: %s", err, string(output))
	}

	dr.status = ContainerStatusRunning

	log.Printf("Docker container started: %s", dr.containerID)

	return nil
}

// Stop stops the Docker container
func (dr *DockerRuntime) Stop() error {
	if dr.containerID == "" {
		return fmt.Errorf("container not created")
	}

	cmd := exec.Command("docker", "stop", dr.containerID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker stop failed: %w: %s", err, string(output))
	}

	dr.status = ContainerStatusStopped

	log.Printf("Docker container stopped: %s", dr.containerID)

	return nil
}

// Delete removes the Docker container
func (dr *DockerRuntime) Delete() error {
	if dr.containerID == "" {
		return fmt.Errorf("container not created")
	}

	cmd := exec.Command("docker", "rm", "-f", dr.containerID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker rm failed: %w: %s", err, string(output))
	}

	log.Printf("Docker container deleted: %s", dr.containerID)

	return nil
}

// GetStatus returns the current container status
func (dr *DockerRuntime) GetStatus() ContainerStatus {
	return dr.status
}
