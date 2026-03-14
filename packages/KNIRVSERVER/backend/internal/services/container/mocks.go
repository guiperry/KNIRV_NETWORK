package container

import (
	"context"
	"fmt"
	"time"
)

// MockContainerOrchestrator is a mock implementation of ContainerOrchestrator for testing
type MockContainerOrchestrator struct {
	containers map[string]*Container
	endpoints  map[string]*Endpoints
}

func (m *MockContainerOrchestrator) ProvisionContainer(ctx context.Context, rentalID string) (*Container, error) {
	containerID := fmt.Sprintf("mock-container-%s", rentalID)
	container := &Container{
		ID:        containerID,
		Status:    ContainerStatusRunning,
		CreatedAt: time.Now(),
		Runtime:   "mock",
	}

	m.containers[containerID] = container
	return container, nil
}

func (m *MockContainerOrchestrator) AllocateEndpoints(ctx context.Context, rentalID string) (*Endpoints, error) {
	endpoints := &Endpoints{
		SSHPort:        22145,
		ValidationPort: 23145,
		ErrorResPort:   24145,
		Host:           "10.0.1.42",
	}

	m.endpoints[rentalID] = endpoints
	return endpoints, nil
}

func (m *MockContainerOrchestrator) InjectSSHKeys(ctx context.Context, containerID string, publicKey string) error {
	// Mock implementation - always succeeds
	return nil
}

func (m *MockContainerOrchestrator) GetContainerStatus(ctx context.Context, containerID string) (ContainerStatus, error) {
	if container, exists := m.containers[containerID]; exists {
		return container.Status, nil
	}
	return ContainerStatusRunning, nil // Default for unknown containers
}

func (m *MockContainerOrchestrator) TerminateContainer(ctx context.Context, containerID string) error {
	if container, exists := m.containers[containerID]; exists {
		container.Status = ContainerStatusTerminated
	}
	return nil
}

// MockSSHProvisioner is a mock implementation of SSHProvisioner for testing
type MockSSHProvisioner struct{}

func (m *MockSSHProvisioner) InjectSSHKeys(ctx context.Context, containerID string, publicKey string) error {
	// Mock implementation - always succeeds
	return nil
}
