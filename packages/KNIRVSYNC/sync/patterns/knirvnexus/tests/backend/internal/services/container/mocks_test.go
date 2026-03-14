package container

import (
	"context"
	"testing"
	"time"
)

func TestMockContainerOrchestrator_ProvisionContainer(t *testing.T) {
	mock := &MockContainerOrchestrator{
		containers: make(map[string]*Container),
		endpoints:  make(map[string]*Endpoints),
	}

	container, err := mock.ProvisionContainer(context.Background(), "test-rental")
	if err != nil {
		t.Fatalf("Failed to provision mock container: %v", err)
	}

	if container.ID != "mock-container-test-rental" {
		t.Errorf("Expected container ID 'mock-container-test-rental', got '%s'", container.ID)
	}

	if container.Status != ContainerStatusRunning {
		t.Errorf("Expected status 'running', got '%s'", container.Status)
	}

	if container.Runtime != "mock" {
		t.Errorf("Expected runtime 'mock', got '%s'", container.Runtime)
	}

	// Verify container was stored
	if stored, exists := mock.containers[container.ID]; !exists {
		t.Error("Container was not stored in mock orchestrator")
	} else if stored != container {
		t.Error("Stored container does not match returned container")
	}
}

func TestMockContainerOrchestrator_AllocateEndpoints(t *testing.T) {
	mock := &MockContainerOrchestrator{
		containers: make(map[string]*Container),
		endpoints:  make(map[string]*Endpoints),
	}

	endpoints, err := mock.AllocateEndpoints(context.Background(), "test-rental")
	if err != nil {
		t.Fatalf("Failed to allocate mock endpoints: %v", err)
	}

	if endpoints.SSHPort != 22145 {
		t.Errorf("Expected SSH port 22145, got %d", endpoints.SSHPort)
	}

	if endpoints.ValidationPort != 23145 {
		t.Errorf("Expected validation port 23145, got %d", endpoints.ValidationPort)
	}

	if endpoints.ErrorResPort != 24145 {
		t.Errorf("Expected error resolution port 24145, got %d", endpoints.ErrorResPort)
	}

	if endpoints.Host != "10.0.1.42" {
		t.Errorf("Expected host '10.0.1.42', got '%s'", endpoints.Host)
	}

	// Verify endpoints were stored
	if stored, exists := mock.endpoints["test-rental"]; !exists {
		t.Error("Endpoints were not stored in mock orchestrator")
	} else if stored != endpoints {
		t.Error("Stored endpoints do not match returned endpoints")
	}
}

func TestMockContainerOrchestrator_InjectSSHKeys(t *testing.T) {
	mock := &MockContainerOrchestrator{}

	err := mock.InjectSSHKeys(context.Background(), "test-container", "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7vbqajDhS5K3t test@example.com")
	if err != nil {
		t.Errorf("Mock SSH key injection failed: %v", err)
	}
}

func TestMockContainerOrchestrator_GetContainerStatus(t *testing.T) {
	mock := &MockContainerOrchestrator{
		containers: make(map[string]*Container),
	}

	// Test with existing container
	mock.containers["test-container"] = &Container{
		ID:     "test-container",
		Status: ContainerStatusStopped,
	}

	status, err := mock.GetContainerStatus(context.Background(), "test-container")
	if err != nil {
		t.Fatalf("Failed to get container status: %v", err)
	}

	if status != ContainerStatusStopped {
		t.Errorf("Expected status 'stopped', got '%s'", status)
	}

	// Test with non-existing container
	status, err = mock.GetContainerStatus(context.Background(), "non-existing")
	if err != nil {
		t.Fatalf("Failed to get status for non-existing container: %v", err)
	}

	if status != ContainerStatusRunning {
		t.Errorf("Expected default status 'running' for non-existing container, got '%s'", status)
	}
}

func TestMockContainerOrchestrator_TerminateContainer(t *testing.T) {
	mock := &MockContainerOrchestrator{
		containers: make(map[string]*Container),
	}

	// Create a container
	container := &Container{
		ID:        "test-container",
		Status:    ContainerStatusRunning,
		CreatedAt: time.Now(),
		Runtime:   "mock",
	}
	mock.containers[container.ID] = container

	// Terminate it
	err := mock.TerminateContainer(context.Background(), "test-container")
	if err != nil {
		t.Fatalf("Failed to terminate container: %v", err)
	}

	// Verify status changed
	if container.Status != ContainerStatusTerminated {
		t.Errorf("Expected status 'terminated', got '%s'", container.Status)
	}

	// Test terminating non-existing container (should not error)
	err = mock.TerminateContainer(context.Background(), "non-existing")
	if err != nil {
		t.Errorf("Terminating non-existing container should not error: %v", err)
	}
}

func TestMockSSHProvisioner_InjectSSHKeys(t *testing.T) {
	mock := &MockSSHProvisioner{}

	err := mock.InjectSSHKeys(context.Background(), "test-container", "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7vbqajDhS5K3t test@example.com")
	if err != nil {
		t.Errorf("Mock SSH key injection failed: %v", err)
	}
}

func TestMockContainerOrchestrator_New(t *testing.T) {
	mock := &MockContainerOrchestrator{
		containers: make(map[string]*Container),
		endpoints:  make(map[string]*Endpoints),
	}

	if mock.containers == nil {
		t.Error("Containers map should be initialized")
	}

	if mock.endpoints == nil {
		t.Error("Endpoints map should be initialized")
	}
}