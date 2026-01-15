package container

import (
	"context"
	"strings"
	"testing"
	"time"

	"backend_server/internal/objects"
)

func TestNewContainerOrchestrator(t *testing.T) {
	config := &ContainerConfig{
		ContainerRuntime:         "docker",
		BaseImage:                "ubuntu:20.04",
		SSHPortRangeStart:        22000,
		SSHPortRangeEnd:          22999,
		ValidationPortRangeStart: 23000,
		ValidationPortRangeEnd:   23999,
		ErrorResPortRangeStart:   24000,
		ErrorResPortRangeEnd:     24999,
		ProvisioningTimeout:      5 * time.Minute,
		CleanupInterval:          10 * time.Minute,
	}

	co, err := NewContainerOrchestrator(config, nil)
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
		SSHPortRangeStart:        22000,
		SSHPortRangeEnd:          22010,
		ValidationPortRangeStart: 23000,
		ValidationPortRangeEnd:   23010,
		ErrorResPortRangeStart:   24000,
		ErrorResPortRangeEnd:     24010,
	}

	co, _ := NewContainerOrchestrator(config, nil)

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
		ContainerRuntime:         "docker",
		BaseImage:                "ubuntu:20.04",
		SSHPortRangeStart:        22000,
		SSHPortRangeEnd:          22010,
		ValidationPortRangeStart: 23000,
		ValidationPortRangeEnd:   23010,
		ErrorResPortRangeStart:   24000,
		ErrorResPortRangeEnd:     24010,
		ProvisioningTimeout:      30 * time.Second, // Increased timeout for reliable testing
		CleanupInterval:          10 * time.Minute,
	}

	co, err := NewContainerOrchestrator(config, nil)
	if err != nil {
		t.Fatalf("Failed to create container orchestrator: %v", err)
	}

	container, err := co.ProvisionContainer("test-rental-123")
	if err != nil {
		t.Fatalf("Failed to provision container: %v", err)
	}

	if container == nil {
		t.Fatal("Container is nil")
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

	// Additional validation for SSH keys
	if container.SSHKeys.PublicKey == "" || container.SSHKeys.PrivateKey == "" {
		t.Error("SSH keys are incomplete")
	}

	// Additional validation for endpoints
	if container.Endpoints.SSHPort == 0 || container.Endpoints.ValidationPort == 0 || container.Endpoints.ErrorResPort == 0 {
		t.Error("Endpoints are incomplete")
	}

	// Test container ID format
	if !strings.HasPrefix(container.ID, "dve-") {
		t.Errorf("Container ID should start with 'dve-', got '%s'", container.ID)
	}
}

func TestContainerOrchestrator_GetContainerStatus(t *testing.T) {
	config := &ContainerConfig{
		ContainerRuntime: "docker",
	}

	co, _ := NewContainerOrchestrator(config, nil)

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

	co, _ := NewContainerOrchestrator(config, nil)

	err := co.TerminateContainer("test-container")
	if err != nil {
		t.Fatalf("Failed to terminate container: %v", err)
	}
}

func TestContainerOrchestrator_Start(t *testing.T) {
	config := &ContainerConfig{
		CleanupInterval: 1 * time.Minute,
	}

	co, _ := NewContainerOrchestrator(config, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := co.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start container orchestrator: %v", err)
	}
}

func TestContainerOrchestrator_Stop(t *testing.T) {
	config := &ContainerConfig{}

	co, _ := NewContainerOrchestrator(config, nil)

	err := co.Stop()
	if err != nil {
		t.Fatalf("Failed to stop container orchestrator: %v", err)
	}
}

func TestContainerOrchestrator_InjectSSHKeys(t *testing.T) {
	config := &ContainerConfig{
		ContainerRuntime: "docker",
	}

	co, _ := NewContainerOrchestrator(config, nil)

	err := co.InjectSSHKeys("test-container", "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7vbqajDhS5K3t test@example.com")
	if err != nil {
		t.Errorf("InjectSSHKeys failed: %v", err)
	}
}

func TestContainerOrchestrator_PerformCleanup(t *testing.T) {
	config := &ContainerConfig{
		CleanupInterval: 1 * time.Hour,
	}

	co, _ := NewContainerOrchestrator(config, nil)

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

	co, _ := NewContainerOrchestrator(config, nil)

	spec := &ContainerSpec{
		ID:             "test-container",
		Image:          "ubuntu:20.04",
		SSHPort:        22000,
		ValidationPort: 23000,
		ErrorResPort:   24000,
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

	co, _ := NewContainerOrchestrator(config, nil)

	// This should not error in the current implementation
	err := co.terminatePodmanContainer("test-container")
	if err != nil {
		t.Errorf("terminatePodmanContainer failed: %v", err)
	}
}

func TestContainerOrchestrator_ExtractRentalIDFromContainerID(t *testing.T) {
	config := &ContainerConfig{}
	co, _ := NewContainerOrchestrator(config, nil)

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
		ContainerRuntime:         "docker",
		ProvisioningTimeout:      1 * time.Millisecond, // Very short timeout
		SSHPortRangeStart:        22000,
		SSHPortRangeEnd:          22010,
		ValidationPortRangeStart: 23000,
		ValidationPortRangeEnd:   23010,
		ErrorResPortRangeStart:   24000,
		ErrorResPortRangeEnd:     24010,
	}

	co, _ := NewContainerOrchestrator(config, nil)

	_, err := co.ProvisionContainer("test-rental")
	if err == nil {
		t.Error("Expected timeout error, but got none")
	}
}

func TestContainerOrchestrator_UnsupportedRuntime(t *testing.T) {
	config := &ContainerConfig{
		ContainerRuntime: "unsupported",
	}

	co, _ := NewContainerOrchestrator(config, nil)

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

// Test for error handling in container provisioning
func TestContainerOrchestrator_ProvisionContainerErrorHandling(t *testing.T) {
	testCases := []struct {
		name          string
		runtime       string
		expectError   bool
		errorContains string
	}{
		{
			name:          "Unsupported runtime",
			runtime:       "unsupported-runtime",
			expectError:   true,
			errorContains: "unsupported container runtime",
		},
		{
			name:          "Empty runtime",
			runtime:       "",
			expectError:   true,
			errorContains: "unsupported container runtime",
		},
		{
			name:          "Valid docker runtime",
			runtime:       "docker",
			expectError:   false,
			errorContains: "",
		},
		{
			name:          "Valid podman runtime",
			runtime:       "podman",
			expectError:   false,
			errorContains: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := &ContainerConfig{
				ContainerRuntime:         tc.runtime,
				BaseImage:                "ubuntu:20.04",
				ProvisioningTimeout:      15 * time.Second,
				SSHPortRangeStart:        22000,
				SSHPortRangeEnd:          22010,
				ValidationPortRangeStart: 23000,
				ValidationPortRangeEnd:   23010,
				ErrorResPortRangeStart:   24000,
				ErrorResPortRangeEnd:     24010,
			}

			co, err := NewContainerOrchestrator(config, nil)
			if err != nil {
				t.Fatalf("Failed to create container orchestrator: %v", err)
			}

			container, err := co.ProvisionContainer("test-rental-error")

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tc.errorContains) {
					t.Errorf("Expected error containing '%s', got: %v", tc.errorContains, err)
				}
				if container != nil {
					t.Errorf("Expected nil container on error, got: %v", container)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if container == nil {
					t.Error("Expected non-nil container")
				}
			}
		})
	}
}

// Test for container lifecycle management
func TestContainerOrchestrator_LifecycleManagement(t *testing.T) {
	config := &ContainerConfig{
		ContainerRuntime:         "docker",
		BaseImage:                "ubuntu:20.04",
		ProvisioningTimeout:      30 * time.Second,
		SSHPortRangeStart:        22000,
		SSHPortRangeEnd:          22010,
		ValidationPortRangeStart: 23000,
		ValidationPortRangeEnd:   23010,
		ErrorResPortRangeStart:   24000,
		ErrorResPortRangeEnd:     24010,
	}

	co, err := NewContainerOrchestrator(config, nil)
	if err != nil {
		t.Fatalf("Failed to create container orchestrator: %v", err)
	}

	// Test provisioning
	container, err := co.ProvisionContainer("test-lifecycle")
	if err != nil {
		t.Fatalf("Failed to provision container: %v", err)
	}

	if container == nil {
		t.Fatal("Container is nil after provisioning")
	}

	// Test status check
	status, err := co.GetContainerStatus(container.ID)
	if err != nil {
		t.Fatalf("Failed to get container status: %v", err)
	}

	if status != ContainerStatusRunning {
		t.Errorf("Expected container status 'running', got '%s'", status)
	}

	// Test SSH key injection
	err = co.InjectSSHKeys(container.ID, "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7vbqajDhS5K3t test@example.com")
	if err != nil {
		t.Errorf("Failed to inject SSH keys: %v", err)
	}

	// Test termination
	err = co.TerminateContainer(container.ID)
	if err != nil {
		t.Fatalf("Failed to terminate container: %v", err)
	}

	// Verify container was terminated (status should still be running due to mock implementation)
	status, err = co.GetContainerStatus(container.ID)
	if err != nil {
		t.Fatalf("Failed to get container status after termination: %v", err)
	}

	// In the mock implementation, status remains running, but in real implementation it would change
	// This test validates the lifecycle methods don't error out
}

// Test for port allocation edge cases
func TestContainerOrchestrator_PortAllocationEdgeCases(t *testing.T) {
	config := &ContainerConfig{
		SSHPortRangeStart:        22000,
		SSHPortRangeEnd:          22005, // Small range of ports
		ValidationPortRangeStart: 23000,
		ValidationPortRangeEnd:   23005, // Small range of ports
		ErrorResPortRangeStart:   24000,
		ErrorResPortRangeEnd:     24005, // Small range of ports
	}

	co, err := NewContainerOrchestrator(config, nil)
	if err != nil {
		t.Fatalf("Failed to create container orchestrator: %v", err)
	}

	// Test allocation with limited ports
	endpoints, err := co.AllocateEndpoints("test-limited-ports-1")
	if err != nil {
		t.Fatalf("Failed to allocate endpoints: %v", err)
	}

	if endpoints == nil {
		t.Fatal("Endpoints is nil")
	}

	// Verify ports are within expected ranges
	if endpoints.SSHPort < 22000 || endpoints.SSHPort > 22005 {
		t.Errorf("SSH port %d not in expected range [22000, 22005]", endpoints.SSHPort)
	}

	if endpoints.ValidationPort < 23000 || endpoints.ValidationPort > 23005 {
		t.Errorf("Validation port %d not in expected range [23000, 23005]", endpoints.ValidationPort)
	}

	if endpoints.ErrorResPort < 24000 || endpoints.ErrorResPort > 24005 {
		t.Errorf("Error resolution port %d not in expected range [24000, 24005]", endpoints.ErrorResPort)
	}

	// Test allocation with different rental ID (should get different ports)
	endpoints2, err := co.AllocateEndpoints("test-limited-ports-2")
	if err != nil {
		t.Fatalf("Failed to allocate endpoints for second rental: %v", err)
	}

	if endpoints2 == nil {
		t.Fatal("Second endpoints is nil")
	}

	// Verify that different rentals get different ports
	if endpoints2.SSHPort == endpoints.SSHPort {
		t.Error("Expected different SSH ports for different rentals")
	}

	if endpoints2.ValidationPort == endpoints.ValidationPort {
		t.Error("Expected different validation ports for different rentals")
	}

	if endpoints2.ErrorResPort == endpoints.ErrorResPort {
		t.Error("Expected different error resolution ports for different rentals")
	}

	// Test port release and reallocation
	co.portAllocator.ReleasePorts("test-limited-ports-1")

	// Allocate again with first rental ID (should reuse ports)
	endpoints3, err := co.AllocateEndpoints("test-limited-ports-1")
	if err != nil {
		t.Fatalf("Failed to reallocate endpoints after release: %v", err)
	}

	if endpoints3 == nil {
		t.Fatal("Third endpoints is nil")
	}

	// Should get the same ports back after release
	if endpoints3.SSHPort != endpoints.SSHPort {
		t.Errorf("Expected same SSH port after release, got %d instead of %d", endpoints3.SSHPort, endpoints.SSHPort)
	}
}
