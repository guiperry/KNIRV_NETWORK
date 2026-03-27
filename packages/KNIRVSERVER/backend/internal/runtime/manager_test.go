// Copyright 2026 KNIRV-SERVER
// SPDX-License-Identifier: GPL-3.0-or-later

package runtime

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockCognitiveEngine implements CognitiveEngineInterface for testing
type MockCognitiveEngine struct {
	mock.Mock
}

func (m *MockCognitiveEngine) OnAgentTaskComplexity(dveID string, complexity int) error {
	args := m.Called(dveID, complexity)
	return args.Error(0)
}

func (m *MockCognitiveEngine) OnAgentResourceUsage(dveID string, cpuPercent float64, memoryMB int64) error {
	args := m.Called(dveID, cpuPercent, memoryMB)
	return args.Error(0)
}

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

func TestCreateAgentContainer(t *testing.T) {
	mockRuntime := &MockContainerRuntime{}
	mockRuntime.On("Create", mock.Anything).Return(nil)
	mockRuntime.On("Start").Return(nil)
	mockRuntime.On("GetStatus").Return(ContainerStatusRunning)

	manager := NewUnifiedContainerManager(nil, nil, nil)
	manager.runtimeSelector.SetTestRuntime(mockRuntime)

	spec := &ContainerSpec{
		Image: "knirv-agent-oh-my-pi:latest",
		Resources: ResourceLimits{
			CPUCores: 4,
			MemoryMB: 8192,
			DiskMB:   40960,
		},
		Environment: map[string]string{
			"OH_MY_PI_MODE":      "server",
			"OH_MY_PI_WORKSPACE": "/workspace/active-memory",
		},
	}

	ctx := context.Background()
	container, err := manager.CreateContainer(ctx, spec, RuntimeModeObject, ObjectTypeAgent, SecurityLevelStrong)
	assert.NoError(t, err)
	assert.NotNil(t, container)
	assert.Equal(t, ObjectTypeAgent, container.ObjectType)
	assert.Equal(t, RuntimeModeObject, container.Mode)
	assert.NotNil(t, container.ViewportProxy)
}

func TestSetCognitiveEngine(t *testing.T) {
	manager := NewUnifiedContainerManager(nil, nil, nil)

	mockEngine := &MockCognitiveEngine{}
	manager.SetCognitiveEngine(mockEngine)

	// Verify the cognitive engine was set
	assert.NotNil(t, manager.cognitiveEngine)
}

func TestCreateAgentRuntimeCapability(t *testing.T) {
	manager := NewUnifiedContainerManager(nil, nil, nil)

	capability := manager.CreateAgentRuntimeCapability("node-123")

	assert.Equal(t, "node-123", capability.NodeID)
	assert.Equal(t, "oh-my-pi-1.0", capability.AgentEngineVer)
	assert.Contains(t, capability.SupportedTools, "git")
	assert.Contains(t, capability.SupportedTools, "python")
	assert.Contains(t, capability.SupportedTools, "curl")
	assert.Contains(t, capability.SupportedTools, "browser")
	assert.Equal(t, "/workspace/active-memory", capability.ActiveMemoryMount)
	assert.Equal(t, 4, capability.MaxConcurrent)
	assert.True(t, capability.ViewportEnabled)
}

func TestEstimateTaskComplexity(t *testing.T) {
	tests := []struct {
		name          string
		resources     ResourceLimits
		minComplexity int
		maxComplexity int
	}{
		{
			name: "low resource task",
			resources: ResourceLimits{
				CPUCores: 1,
				MemoryMB: 1024,
				DiskMB:   5120,
			},
			minComplexity: 1,
			maxComplexity: 3,
		},
		{
			name: "medium resource task",
			resources: ResourceLimits{
				CPUCores: 2,
				MemoryMB: 4096,
				DiskMB:   10240,
			},
			minComplexity: 3,
			maxComplexity: 5,
		},
		{
			name: "high resource task",
			resources: ResourceLimits{
				CPUCores: 8,
				MemoryMB: 16384,
				DiskMB:   20480,
			},
			minComplexity: 7,
			maxComplexity: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			complexity := estimateTaskComplexity(tt.resources)
			assert.GreaterOrEqual(t, complexity, tt.minComplexity)
			assert.LessOrEqual(t, complexity, tt.maxComplexity)
		})
	}
}

func TestGetMinimalSyscalls(t *testing.T) {
	syscalls := getMinimalSyscalls()

	// Verify essential syscalls are included
	assert.Contains(t, syscalls, uint32(1))   // exit
	assert.Contains(t, syscalls, uint32(9))   // mmap
	assert.Contains(t, syscalls, uint32(10))  // munmap
	assert.Contains(t, syscalls, uint32(257)) // openat
	assert.Contains(t, syscalls, uint32(259)) // read
	assert.Contains(t, syscalls, uint32(281)) // write
}

func TestAgentPoliciesMapInitialized(t *testing.T) {
	manager := NewUnifiedContainerManager(nil, nil, nil)

	// Verify agent policies map is initialized
	assert.NotNil(t, manager.agentPolicies)
	assert.Empty(t, manager.agentPolicies)
}

func TestOhMyPiAgentPolicyConfig(t *testing.T) {
	policy := NewOhMyPiAgentPolicyConfig()

	assert.NotNil(t, policy)
	assert.True(t, policy.AllowNetwork)
	assert.True(t, policy.AllowFilesystem)
	assert.False(t, policy.RequireTEE)
	assert.Equal(t, uint64(8192), policy.MaxMemoryMB)
	assert.Equal(t, 80, policy.MaxCPUPercent)
	assert.NotEmpty(t, policy.AllowedSyscalls)
	assert.Contains(t, policy.AllowedSyscalls, uint32(59)) // execve
	assert.NotEmpty(t, policy.AllowedPaths)
	assert.Contains(t, policy.AllowedPaths, "/workspace")
	assert.NotEmpty(t, policy.AllowedNetworks)
	assert.Contains(t, policy.AllowedNetworks, "127.0.0.0/8")
}

func TestBuildSpecForObjectTypeAgent(t *testing.T) {
	manager := NewUnifiedContainerManager(nil, nil, nil)

	config := &NestedObjectConfig{
		ObjectType:        ObjectTypeAgent,
		EnableViewport:    true,
		ViewportRenderers: []string{"http", "websocket"},
		ServicePorts:      map[string]int{"viewport": 8080, "jupyter": 8888},
	}

	spec := manager.buildSpecForObjectType(config)

	assert.Equal(t, "knirv-agent-oh-my-pi:latest", spec.Image)
	assert.Equal(t, "bridge", spec.NetworkMode)
	assert.Equal(t, 4, spec.Resources.CPUCores)
	assert.Equal(t, int64(8192), spec.Resources.MemoryMB)
	assert.Equal(t, int64(40960), spec.Resources.DiskMB)
	assert.Equal(t, "server", spec.Environment["OH_MY_PI_MODE"])
	assert.Equal(t, "/workspace/active-memory", spec.Environment["OH_MY_PI_WORKSPACE"])
	assert.Contains(t, spec.Environment["OH_MY_PI_TOOLS"], "git")
	assert.Contains(t, spec.Environment["OH_MY_PI_TOOLS"], "python")
	assert.Contains(t, spec.Environment["OH_MY_PI_TOOLS"], "browser")
	assert.True(t, len(spec.Ports) > 0)
	assert.True(t, len(spec.Volumes) > 0)

	// Verify Active Memory mount
	foundActiveMemory := false
	for _, vol := range spec.Volumes {
		if vol.Target == "/workspace/active-memory" {
			foundActiveMemory = true
			break
		}
	}
	assert.True(t, foundActiveMemory, "Active Memory volume mount should be present")
}
