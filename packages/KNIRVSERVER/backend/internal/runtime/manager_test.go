// Copyright 2026 KNIRV-SERVER
// SPDX-License-Identifier: GPL-3.0-or-later

package runtime

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockContainerRuntime is a mock implementation of ContainerRuntime
type MockContainerRuntime struct {
	mock.Mock
}

func (m *MockContainerRuntime) Create(spec *ContainerSpec) error {
	args := m.Called(spec)
	return args.Error(0)
}

func (m *MockContainerRuntime) Start() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockContainerRuntime) Stop() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockContainerRuntime) Delete() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockContainerRuntime) GetStatus() ContainerStatus {
	args := m.Called()
	if status, ok := args.Get(0).(ContainerStatus); ok {
		return status
	}
	return ContainerStatusStopped
}

func TestNewUnifiedContainerManager(t *testing.T) {
	// This test just verifies that the manager can be created
	// In a real implementation, you would pass real or mock dependencies
	manager := NewUnifiedContainerManager(nil, nil, nil)
	assert.NotNil(t, manager)
	assert.NotNil(t, manager.runtimeSelector)
	assert.NotNil(t, manager.assetRegistry)
	assert.Empty(t, manager.containers)
}

func TestCreateContainer(t *testing.T) {
	// Create a mock runtime
	mockRuntime := &MockContainerRuntime{}
	mockRuntime.On("Create", mock.Anything).Return(nil)
	mockRuntime.On("Start").Return(nil)
	mockRuntime.On("GetStatus").Return(ContainerStatusRunning)

	manager := NewUnifiedContainerManager(nil, nil, nil)

	// Set the test runtime in the selector
	manager.runtimeSelector.SetTestRuntime(mockRuntime)

	// Create a container spec
	spec := &ContainerSpec{
		Image: "test-image:latest",
		Resources: ResourceLimits{
			CPUCores: 1,
			MemoryMB: 256,
			DiskMB:   1024,
		},
	}

	// Create container
	ctx := context.Background()
	container, err := manager.CreateContainer(ctx, spec, RuntimeModeDocker, ObjectTypeWebApp, SecurityLevelBasic)
	assert.NoError(t, err)
	assert.NotNil(t, container)
	assert.Equal(t, RuntimeModeDocker, container.Mode)
	assert.Equal(t, ObjectTypeWebApp, container.ObjectType)
	assert.Equal(t, SecurityLevelBasic, container.SecurityLevel)
	assert.Equal(t, spec.Image, container.Spec.Image)
	assert.Equal(t, ContainerStatusRunning, container.Status)
	assert.NotEmpty(t, container.CryptoHash)

	// Verify container is in the map
	storedContainer, err := manager.GetContainer(container.ID)
	assert.NoError(t, err)
	assert.Equal(t, container.ID, storedContainer.ID)
}

func TestCreateNestedObject(t *testing.T) {
	// Create a mock runtime
	mockRuntime := &MockContainerRuntime{}
	mockRuntime.On("Create", mock.Anything).Return(nil)
	mockRuntime.On("Start").Return(nil)
	mockRuntime.On("GetStatus").Return(ContainerStatusRunning)

	manager := NewUnifiedContainerManager(nil, nil, nil)

	// Set the test runtime in the selector
	manager.runtimeSelector.SetTestRuntime(mockRuntime)

	// Create nested object config
	config := &NestedObjectConfig{
		ObjectType:        ObjectTypeAPI,
		EnableViewport:    true,
		ViewportRenderers: []string{"http", "webrtc"},
		ServicePorts: map[string]int{
			"api": 8080,
		},
		Metadata: map[string]interface{}{
			"image": "api-server:latest",
		},
	}

	// Create nested object
	ctx := context.Background()
	container, err := manager.CreateNestedObject(ctx, config)
	assert.NoError(t, err)
	assert.NotNil(t, container)
	assert.Equal(t, RuntimeModeObject, container.Mode)
	assert.Equal(t, ObjectTypeAPI, container.ObjectType)
	assert.Equal(t, SecurityLevelStrong, container.SecurityLevel)
	assert.Equal(t, "api-server:latest", container.Spec.Image)
	assert.NotNil(t, container.ViewportProxy)
	assert.NotEmpty(t, container.CryptoHash)
}

func TestListContainers(t *testing.T) {
	// Create a mock runtime
	mockRuntime := &MockContainerRuntime{}
	mockRuntime.On("Create", mock.Anything).Return(nil)
	mockRuntime.On("Start").Return(nil)
	mockRuntime.On("GetStatus").Return(ContainerStatusRunning)

	manager := NewUnifiedContainerManager(nil, nil, nil)

	// Set the test runtime in the selector
	manager.runtimeSelector.SetTestRuntime(mockRuntime)

	// Create container spec
	spec := &ContainerSpec{
		Image: "test-image:latest",
		Resources: ResourceLimits{
			CPUCores: 1,
			MemoryMB: 256,
			DiskMB:   1024,
		},
	}

	// Create multiple containers
	ctx := context.Background()
	container1, err := manager.CreateContainer(ctx, spec, RuntimeModeDocker, ObjectTypeWebApp, SecurityLevelBasic)
	assert.NoError(t, err)

	container2, err := manager.CreateContainer(ctx, spec, RuntimeModeDocker, ObjectTypeAPI, SecurityLevelBasic)
	assert.NoError(t, err)

	// List containers
	containers := manager.ListContainers()
	assert.Len(t, containers, 2)

	// Check if both containers are in the list
	hasContainer1 := false
	hasContainer2 := false
	for _, container := range containers {
		if container.ID == container1.ID {
			hasContainer1 = true
		}
		if container.ID == container2.ID {
			hasContainer2 = true
		}
	}

	assert.True(t, hasContainer1)
	assert.True(t, hasContainer2)
}

func TestDestroyContainer(t *testing.T) {
	// Create a mock runtime
	mockRuntime := &MockContainerRuntime{}
	mockRuntime.On("Create", mock.Anything).Return(nil)
	mockRuntime.On("Start").Return(nil)
	mockRuntime.On("Stop").Return(nil)
	mockRuntime.On("Delete").Return(nil)
	mockRuntime.On("GetStatus").Return(ContainerStatusRunning)

	manager := NewUnifiedContainerManager(nil, nil, nil)

	// Set the test runtime in the selector
	manager.runtimeSelector.SetTestRuntime(mockRuntime)

	// Create container spec
	spec := &ContainerSpec{
		Image: "test-image:latest",
		Resources: ResourceLimits{
			CPUCores: 1,
			MemoryMB: 256,
			DiskMB:   1024,
		},
	}

	// Create container
	ctx := context.Background()
	container, err := manager.CreateContainer(ctx, spec, RuntimeModeDocker, ObjectTypeWebApp, SecurityLevelBasic)
	assert.NoError(t, err)

	// Destroy container
	err = manager.DestroyContainer(ctx, container.ID)
	assert.NoError(t, err)

	// Verify container is removed
	_, err = manager.GetContainer(container.ID)
	assert.Error(t, err)
}

func TestBuildSpecForObjectType(t *testing.T) {
	manager := NewUnifiedContainerManager(nil, nil, nil)

	tests := []struct {
		name        string
		objectType  ObjectType
		config      *NestedObjectConfig
		expectedImg string
		expectedCPU int
		expectedRAM int64
	}{
		{
			name:        "webapp type",
			objectType:  ObjectTypeWebApp,
			config:      &NestedObjectConfig{ObjectType: ObjectTypeWebApp},
			expectedImg: "alpine:latest",
			expectedCPU: 4,
			expectedRAM: 8192,
		},
		{
			name:        "api type",
			objectType:  ObjectTypeAPI,
			config:      &NestedObjectConfig{ObjectType: ObjectTypeAPI},
			expectedImg: "alpine:latest",
			expectedCPU: 4,
			expectedRAM: 8192,
		},
		{
			name:        "3d type",
			objectType:  ObjectType3D,
			config:      &NestedObjectConfig{ObjectType: ObjectType3D},
			expectedImg: "glb-renderer:latest",
			expectedCPU: 4,
			expectedRAM: 8192,
		},
		{
			name:        "p2p type",
			objectType:  ObjectTypeP2P,
			config:      &NestedObjectConfig{ObjectType: ObjectTypeP2P},
			expectedImg: "knirvrouter:latest",
			expectedCPU: 4,
			expectedRAM: 8192,
		},
		{
			name:        "blockchain type",
			objectType:  ObjectTypeBlockchain,
			config:      &NestedObjectConfig{ObjectType: ObjectTypeBlockchain},
			expectedImg: "blockchain-node:latest",
			expectedCPU: 8,
			expectedRAM: 16384,
		},
		{
			name:        "oracle type",
			objectType:  ObjectTypeOracle,
			config:      &NestedObjectConfig{ObjectType: ObjectTypeOracle},
			expectedImg: "blockchain-node:latest",
			expectedCPU: 8,
			expectedRAM: 16384,
		},
		{
			name:        "model type",
			objectType:  ObjectTypeModel,
			config:      &NestedObjectConfig{ObjectType: ObjectTypeModel},
			expectedImg: "model-server:latest",
			expectedCPU: 8,
			expectedRAM: 32768,
		},
		{
			name:       "custom type",
			objectType: ObjectTypeCustom,
			config: &NestedObjectConfig{
				ObjectType: ObjectTypeCustom,
				Metadata: map[string]interface{}{
					"image": "custom-image:1.0.0",
				},
			},
			expectedImg: "custom-image:1.0.0",
			expectedCPU: 4,
			expectedRAM: 8192,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := manager.buildSpecForObjectType(tt.config)
			assert.Equal(t, tt.expectedImg, spec.Image)
			assert.Equal(t, tt.expectedCPU, spec.Resources.CPUCores)
			assert.Equal(t, tt.expectedRAM, spec.Resources.MemoryMB)
		})
	}
}

func TestGenerateCryptoHash(t *testing.T) {
	manager := NewUnifiedContainerManager(nil, nil, nil)

	// Create a mock runtime
	mockRuntime := &MockContainerRuntime{}
	mockRuntime.On("Create", mock.Anything).Return(nil)
	mockRuntime.On("Start").Return(nil)
	mockRuntime.On("GetStatus").Return(ContainerStatusRunning)

	// Set the test runtime in the selector
	manager.runtimeSelector.SetTestRuntime(mockRuntime)

	// Create container spec
	spec := &ContainerSpec{
		Image: "test-image:latest",
		Resources: ResourceLimits{
			CPUCores: 1,
			MemoryMB: 256,
			DiskMB:   1024,
		},
	}

	ctx := context.Background()
	container, err := manager.CreateContainer(ctx, spec, RuntimeModeDocker, ObjectTypeWebApp, SecurityLevelBasic)
	assert.NoError(t, err)

	// Verify crypto hash is generated
	assert.NotEmpty(t, container.CryptoHash)
	assert.Len(t, container.CryptoHash, 64) // BLAKE3-256 is 64 hex characters
}

func TestCreateContainerWithInvalidSpec(t *testing.T) {
	manager := NewUnifiedContainerManager(nil, nil, nil)

	// Create container with nil spec
	_, err := manager.CreateContainer(context.Background(), nil, RuntimeModeDocker, ObjectTypeWebApp, SecurityLevelBasic)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "container spec is required")

	// Create container with empty image
	_, err = manager.CreateContainer(context.Background(), &ContainerSpec{}, RuntimeModeDocker, ObjectTypeWebApp, SecurityLevelBasic)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "container image is required")
}
