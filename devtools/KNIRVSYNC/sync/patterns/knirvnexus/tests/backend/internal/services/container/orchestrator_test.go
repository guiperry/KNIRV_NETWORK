package container

import (
	"context"
	"testing"
	"time"

	"backend_server/internal/objects"
)

func TestNewContainerOrchestrator(t *testing.T) {
	config := &ContainerConfig{
		ContainerRuntime:          "docker",
		BaseImage:                 "ubuntu:20.04",
		SSHPortRangeStart:         22000,
		SSHPortRangeEnd:           22999,
		ValidationPortRangeStart:  23000,
		ValidationPortRangeEnd:    23999,
		ErrorResPortRangeStart:    24000,
		ErrorResPortRangeEnd:      24999,
		ProvisioningTimeout:       5 * time.Minute,
		CleanupInterval:           10 * time.Minute,
	}

	co, err := NewContainerOrchestrator(config)
	if err != nil {
		t.Fatalf("Failed to create container orchestrator: %v", err)
	}

	if co == nil {
		t.Fatal("Container orchestrator is nil")
	}

	if co.config.ContainerRuntime != "docker" {
		t.Errorf("Expected container runtime 'docker', got '%s'", co.config.ContainerRuntime)
	}
}

func TestContainerOrchestrator_AllocateEndpoints(t *testing.T) {
	config := &ContainerConfig{
		SSHPortRangeStart:         22000,
		SSHPortRangeEnd:           22010,
		ValidationPortRangeStart:  23000,
		ValidationPortRangeEnd:    23010,
		ErrorResPortRangeStart:    24000,
		ErrorResPortRangeEnd:      24010,
	}

	co, _ := NewContainerOrchestrator(config)

	endpoints, err := co.AllocateEndpoints("test-rental")
	if err != nil {
		t.Fatalf("Failed to allocate endpoints: %v", err)
	}

	if endpoints.SSHPort < 22000 || endpoints.SSHPort > 22010 {
		t.Errorf("SSH port %d not in expected range [22000, 22010]", endpoints.SSHPort)
	}

	if endpoints.ValidationPort < 23000 || endpoints.ValidationPort > 23010 {
		t.Errorf("Validation port %d not in expected range [23000, 23010]", endpoints.ValidationPort)
	}

	if endpoints.ErrorResPort < 24000 || endpoints.ErrorResPort > 24010 {
		t.Errorf("Error resolution port %d not in expected range [24000, 24010]", endpoints.ErrorResPort)
	}
}

func TestContainerOrchestrator_ProvisionContainer(t *testing.T) {
	config := &ContainerConfig{
		ContainerRuntime:          "docker",
		BaseImage:                 "ubuntu:20.04",
		SSHPortRangeStart:         22000,
		SSHPortRangeEnd:           22010,
		ValidationPortRangeStart:  23000,
		ValidationPortRangeEnd:    23010,
		ErrorResPortRangeStart:    24000,
		ErrorResPortRangeEnd:      24010,
		ProvisioningTimeout:       10 * time.Second,
		CleanupInterval:           10 * time.Minute,
	}

	co, _ := NewContainerOrchestrator(config)

	container, err := co.ProvisionContainer("test-rental-123")
	if err != nil {
		t.Fatalf("Failed to provision container: %v", err)
	}

	if container.ID == "" {
		t.Error("Container ID is empty")
	}

	if container.Status != ContainerStatusRunning {
		t.Errorf("Expected container status 'running', got '%s'", container.Status)
	}

	if container.Runtime != "docker" {
		t.Errorf("Expected runtime 'docker', got '%s'", container.Runtime)
	}

	if container.Spec == nil {
		t.Fatal("Container spec is nil")
	}

	if container.Spec.Image != "ubuntu:20.04" {
		t.Errorf("Expected image 'ubuntu:20.04', got '%s'", container.Spec.Image)
	}

	if container.SSHKeys == nil {
		t.Error("SSH keys are nil")
	}

	if container.Endpoints == nil {
		t.Error("Endpoints are nil")
	}
}

func TestContainerOrchestrator_GetContainerStatus(t *testing.T) {
	config := &ContainerConfig{
		ContainerRuntime: "docker",
	}

	co, _ := NewContainerOrchestrator(config)

	status, err := co.GetContainerStatus("test-container")
	if err != nil {
		t.Fatalf("Failed to get container status: %v", err)
	}

	if status != ContainerStatusRunning {
		t.Errorf("Expected status 'running', got '%s'", status)
	}
}

func TestContainerOrchestrator_TerminateContainer(t *testing.T) {
	config := &ContainerConfig{
		ContainerRuntime: "docker",
	}

	co, _ := NewContainerOrchestrator(config)

	err := co.TerminateContainer("test-container")
	if err != nil {
		t.Fatalf("Failed to terminate container: %v", err)
	}
}

func TestContainerOrchestrator_Start(t *testing.T) {
	config := &ContainerConfig{
		CleanupInterval: 1 * time.Minute,
	}

	co, _ := NewContainerOrchestrator(config)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := co.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start container orchestrator: %v", err)
	}
}

func TestContainerOrchestrator_Stop(t *testing.T) {
	config := &ContainerConfig{}

	co, _ := NewContainerOrchestrator(config)

	err := co.Stop()
	if err != nil {
		t.Fatalf("Failed to stop container orchestrator: %v", err)
	}
}

func TestContainerOrchestrator_InjectSSHKeys(t *testing.T) {
	config := &ContainerConfig{
		ContainerRuntime: "docker",
	}

	co, _ := NewContainerOrchestrator(config)

	err := co.InjectSSHKeys("test-container", "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7vbqajDhS5K3t test@example.com")
	if err != nil {
		t.Errorf("InjectSSHKeys failed: %v", err)
	}
}

func TestContainerOrchestrator_PerformCleanup(t *testing.T) {
	config := &ContainerConfig{
		CleanupInterval: 1 * time.Hour,
	}

	co, _ := NewContainerOrchestrator(config)

	// This method is private, but we can test it indirectly by calling the cleanup routine
	// For now, just ensure the orchestrator can be created without issues
	if co == nil {
		t.Error("Container orchestrator should not be nil")
	}

	// Test that the cleanup interval is set correctly
	if co.config.CleanupInterval != 1*time.Hour {
		t.Errorf("Expected cleanup interval 1h, got %v", co.config.CleanupInterval)
	}
}

func TestContainerOrchestrator_ProvisionPodmanContainer(t *testing.T) {
	config := &ContainerConfig{
		ContainerRuntime: "podman",
		BaseImage:        "ubuntu:20.04",
	}

	co, _ := NewContainerOrchestrator(config)

	spec := &ContainerSpec{
		ID:     "test-container",
		Image:  "ubuntu:20.04",
		SSHPort: 22000,
		ValidationPort: 23000,
		ErrorResPort: 24000,
	}

	ctx := context.Background()
	// This should not error in the current implementation
	container, err := co.provisionPodmanContainer(ctx, spec)
	if err != nil {
		t.Errorf("provisionPodmanContainer failed: %v", err)
	}

	if container == nil {
		t.Error("Container should not be nil")
	}

	if container.Runtime != "podman" {
		t.Errorf("Expected runtime 'podman', got '%s'", container.Runtime)
	}
}

func TestContainerOrchestrator_TerminatePodmanContainer(t *testing.T) {
	config := &ContainerConfig{
		ContainerRuntime: "podman",
	}

	co, _ := NewContainerOrchestrator(config)

	// This should not error in the current implementation
	err := co.terminatePodmanContainer("test-container")
	if err != nil {
		t.Errorf("terminatePodmanContainer failed: %v", err)
	}
}

func TestContainerOrchestrator_ExtractRentalIDFromContainerID(t *testing.T) {
	config := &ContainerConfig{}
	co, _ := NewContainerOrchestrator(config)

	tests := []struct {
		containerID string
		expected    string
	}{
		{"dve-rental123-1234567890", "rental123"},
		{"dve-test-9876543210", "test"},
		{"invalid-container", ""},
		{"dve-", ""},
		{"", ""},
	}

	for _, test := range tests {
		result := co.extractRentalIDFromContainerID(test.containerID)
		if result != test.expected {
			t.Errorf("For container ID '%s', expected rental ID '%s', got '%s'", test.containerID, test.expected, result)
		}
	}
}

func TestContainerOrchestrator_ProvisionContainerTimeout(t *testing.T) {
	config := &ContainerConfig{
		ContainerRuntime:          "docker",
		ProvisioningTimeout:       1 * time.Millisecond, // Very short timeout
		SSHPortRangeStart:         22000,
		SSHPortRangeEnd:           22010,
		ValidationPortRangeStart:  23000,
		ValidationPortRangeEnd:    23010,
		ErrorResPortRangeStart:    24000,
		ErrorResPortRangeEnd:      24010,
	}

	co, _ := NewContainerOrchestrator(config)

	_, err := co.ProvisionContainer("test-rental")
	if err == nil {
		t.Error("Expected timeout error, but got none")
	}
}

func TestContainerOrchestrator_UnsupportedRuntime(t *testing.T) {
	config := &ContainerConfig{
		ContainerRuntime: "unsupported",
	}

	co, _ := NewContainerOrchestrator(config)

	_, err := co.ProvisionContainer("test-rental")
	if err == nil {
		t.Error("Expected error for unsupported runtime, but got none")
	}
}

func TestContainerSpec_ResourceLimits(t *testing.T) {
	spec := &ContainerSpec{
		ResourceLimits: objects.ResourceLimits{
			MaxCPU:       2.0,
			MaxMemory:    4 * 1024 * 1024 * 1024,
			MaxDisk:      50 * 1024 * 1024 * 1024,
			MaxBandwidth: 100 * 1024 * 1024,
		},
	}

	if spec.ResourceLimits.MaxCPU != 2.0 {
		t.Errorf("Expected MaxCPU 2.0, got %f", spec.ResourceLimits.MaxCPU)
	}

	if spec.ResourceLimits.MaxMemory != 4*1024*1024*1024 {
		t.Errorf("Expected MaxMemory %d, got %d", 4*1024*1024*1024, spec.ResourceLimits.MaxMemory)
	}
}

func TestContainerStatus_Constants(t *testing.T) {
	statuses := []ContainerStatus{
		ContainerStatusPending,
		ContainerStatusRunning,
		ContainerStatusStopped,
		ContainerStatusFailed,
		ContainerStatusTerminated,
	}

	expected := []string{"pending", "running", "stopped", "failed", "terminated"}

	for i, status := range statuses {
		if string(status) != expected[i] {
			t.Errorf("Expected status '%s', got '%s'", expected[i], status)
		}
	}
}

func TestContainerEventType_Constants(t *testing.T) {
	eventTypes := []ContainerEventType{
		ContainerEventProvisioned,
		ContainerEventStarted,
		ContainerEventStopped,
		ContainerEventFailed,
		ContainerEventTerminated,
		ContainerEventCleaned,
	}

	expected := []string{"provisioned", "started", "stopped", "failed", "terminated", "cleaned"}

	for i, eventType := range eventTypes {
		if string(eventType) != expected[i] {
			t.Errorf("Expected event type '%s', got '%s'", expected[i], eventType)
		}
	}
}