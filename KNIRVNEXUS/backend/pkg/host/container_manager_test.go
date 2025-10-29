package host

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewContainerManager(t *testing.T) {
	ctx := context.Background()
	config := &HostConfig{
		ContainerRuntime: "docker",
	}

	cm, err := NewContainerManager(ctx, config)
	// Docker may not be available in test environment, so we expect it might fail
	if err != nil {
		assert.Contains(t, err.Error(), "container runtime verification failed")
		return
	}

	assert.NotNil(t, cm)
	assert.Equal(t, "docker", cm.runtime)
	assert.NotNil(t, cm.containers)
	assert.False(t, cm.running)
}

func TestContainerManager_Start_Stop(t *testing.T) {
	ctx := context.Background()
	config := &HostConfig{
		ContainerRuntime: "docker",
	}

	cm, err := NewContainerManager(ctx, config)
	if err != nil {
		// Skip test if docker is not available
		t.Skip("Docker not available in test environment")
		return
	}

	// Test starting
	err = cm.Start()
	require.NoError(t, err)
	assert.True(t, cm.running)

	// Test stopping
	err = cm.Stop()
	assert.NoError(t, err)
	assert.False(t, cm.running)
}

func TestContainerManager_Start_AlreadyRunning(t *testing.T) {
	ctx := context.Background()
	config := &HostConfig{
		ContainerRuntime: "docker",
	}

	cm, err := NewContainerManager(ctx, config)
	if err != nil {
		t.Skip("Docker not available in test environment")
		return
	}

	// Manually set running to true
	cm.running = true

	err = cm.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")
}

func TestContainerManager_GetContainerList(t *testing.T) {
	ctx := context.Background()
	config := &HostConfig{
		ContainerRuntime: "docker",
	}

	cm, err := NewContainerManager(ctx, config)
	if err != nil {
		t.Skip("Docker not available in test environment")
		return
	}

	// Add a test container
	testContainer := &Container{
		ID:   "test123",
		Name: "test-container",
	}
	cm.containers["test123"] = testContainer

	containers, err := cm.GetContainerList()
	require.NoError(t, err)
	assert.Len(t, containers, 1)
	assert.Equal(t, "test123", containers[0].ID)
	assert.Equal(t, "test-container", containers[0].Name)

	// Verify it's a copy (not the same pointer)
	assert.NotSame(t, testContainer, &containers[0])
}

func TestContainerManager_GetKNIRVContainers(t *testing.T) {
	ctx := context.Background()
	config := &HostConfig{
		ContainerRuntime: "docker",
	}

	cm, err := NewContainerManager(ctx, config)
	if err != nil {
		t.Skip("Docker not available in test environment")
		return
	}

	// Add test containers
	knirvContainer := &Container{
		ID:                "knirv123",
		Name:              "knirv-nexus",
		IsKNIRVContainer:  true,
		ServiceType:       "nexus",
	}
	regularContainer := &Container{
		ID:   "regular123",
		Name: "nginx",
	}

	cm.containers["knirv123"] = knirvContainer
	cm.containers["regular123"] = regularContainer

	knirvContainers, err := cm.GetKNIRVContainers()
	require.NoError(t, err)
	assert.Len(t, knirvContainers, 1)
	assert.Equal(t, "knirv123", knirvContainers[0].ID)
	assert.True(t, knirvContainers[0].IsKNIRVContainer)
}

func TestContainerManager_HealthCheck(t *testing.T) {
	ctx := context.Background()
	config := &HostConfig{
		ContainerRuntime: "docker",
	}

	cm, err := NewContainerManager(ctx, config)
	if err != nil {
		t.Skip("Docker not available in test environment")
		return
	}

	// Test when not running
	err = cm.HealthCheck()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")

	// Set as running and test stale data
	cm.running = true
	cm.lastUpdate = time.Now().Add(-120 * time.Second) // 2 minutes ago

	err = cm.HealthCheck()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "data is stale")
}

func TestContainerManager_verifyRuntime(t *testing.T) {
	ctx := context.Background()
	config := &HostConfig{
		ContainerRuntime: "nonexistent",
	}

	_, err := NewContainerManager(ctx, config)
	// This should fail because nonexistent runtime doesn't exist
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "container runtime verification failed")
}

func TestContainerManager_identifyKNIRVContainer(t *testing.T) {
	ctx := context.Background()
	config := &HostConfig{
		ContainerRuntime: "docker",
	}

	cm, err := NewContainerManager(ctx, config)
	if err != nil {
		t.Skip("Docker not available in test environment")
		return
	}

	tests := []struct {
		name             string
		containerName    string
		containerImage   string
		expectedKNIRV    bool
		expectedService  string
		expectedP2P      bool
	}{
		{"KNIRV nexus", "knirv-nexus", "knirv/nexus:latest", true, "knirv-other", false},
		{"DVE manager", "dve-manager", "knirv/dve:latest", true, "dve-manager", true},
		{"Validation core", "validation-core", "knirv/validation:latest", true, "validation-core", false},
		{"Model server", "model-server", "knirv/model:latest", true, "model-server", false},
		{"Data engine", "data-engine", "knirv/data:latest", true, "data-engine", false},
		{"Inference service", "inference-service", "knirv/inference:latest", true, "inference", false},
		{"P2P enabled", "knirv-p2p", "knirv/p2p:latest", true, "knirv-other", true},
		{"Regular container", "nginx", "nginx:latest", false, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container := &Container{
				Name:  tt.containerName,
				Image: tt.containerImage,
			}

			cm.identifyKNIRVContainer(container)

			assert.Equal(t, tt.expectedKNIRV, container.IsKNIRVContainer)
			if tt.expectedKNIRV {
				assert.Equal(t, tt.expectedService, container.ServiceType)
				assert.Equal(t, tt.expectedP2P, container.P2PEnabled)
			}
		})
	}
}

func TestContainerManager_identifyKNIRVNetwork(t *testing.T) {
	ctx := context.Background()
	config := &HostConfig{
		ContainerRuntime: "docker",
	}

	cm, err := NewContainerManager(ctx, config)
	if err != nil {
		t.Skip("Docker not available in test environment")
		return
	}

	tests := []struct {
		name           string
		networkName    string
		expectedKNIRV  bool
		expectedP2P    bool
		expectedEnc    bool
	}{
		{"KNIRV network", "knirv-nexus", true, false, true},
		{"P2P network", "knirv-p2p", true, true, true},
		{"DVE network", "dve-network", true, false, true},
		{"Regular network", "bridge", false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			network := &ContainerNetwork{
				Name: tt.networkName,
			}

			cm.identifyKNIRVNetwork(network)

			assert.Equal(t, tt.expectedKNIRV, network.IsKNIRVNetwork)
			if tt.expectedKNIRV {
				assert.Equal(t, tt.expectedP2P, network.P2PEnabled)
				assert.Equal(t, tt.expectedEnc, network.Encrypted)
			}
		})
	}
}

func Test_getString(t *testing.T) {
	m := map[string]interface{}{
		"string_key": "test_value",
		"int_key":    123,
		"nil_key":    nil,
	}

	assert.Equal(t, "test_value", getString(m, "string_key"))
	assert.Equal(t, "", getString(m, "int_key"))
	assert.Equal(t, "", getString(m, "missing_key"))
	assert.Equal(t, "", getString(m, "nil_key"))
}

func TestContainerManager_parseContainerJSON(t *testing.T) {
	ctx := context.Background()
	config := &HostConfig{
		ContainerRuntime: "docker",
	}

	cm, err := NewContainerManager(ctx, config)
	if err != nil {
		t.Skip("Docker not available in test environment")
		return
	}

	jsonLine := `{
		"ID": "abc123",
		"Names": "test-container",
		"Image": "nginx:latest",
		"Status": "running",
		"State": "running",
		"CreatedAt": "2023-01-01T00:00:00Z"
	}`

	container, err := cm.parseContainerJSON(jsonLine)
	require.NoError(t, err)
	assert.Equal(t, "abc123", container.ID)
	assert.Equal(t, "test-container", container.Name)
	assert.Equal(t, "nginx:latest", container.Image)
	assert.Equal(t, "running", container.Status)
	assert.Equal(t, "running", container.State)
	assert.True(t, container.CreatedAt.Year() == 2023)
}

func TestContainerManager_parseContainerJSON_Invalid(t *testing.T) {
	ctx := context.Background()
	config := &HostConfig{
		ContainerRuntime: "docker",
	}

	cm, err := NewContainerManager(ctx, config)
	if err != nil {
		t.Skip("Docker not available in test environment")
		return
	}

	_, err = cm.parseContainerJSON("invalid json")
	assert.Error(t, err)
}

func TestContainerManager_parseNetworkJSON(t *testing.T) {
	ctx := context.Background()
	config := &HostConfig{
		ContainerRuntime: "docker",
	}

	cm, err := NewContainerManager(ctx, config)
	if err != nil {
		t.Skip("Docker not available in test environment")
		return
	}

	jsonLine := `{
		"ID": "net123",
		"Name": "test-network",
		"Driver": "bridge",
		"Scope": "local"
	}`

	network, err := cm.parseNetworkJSON(jsonLine)
	require.NoError(t, err)
	assert.Equal(t, "net123", network.ID)
	assert.Equal(t, "test-network", network.Name)
	assert.Equal(t, "bridge", network.Driver)
	assert.Equal(t, "local", network.Scope)
}

func TestContainerManager_parseNetworkJSON_Invalid(t *testing.T) {
	ctx := context.Background()
	config := &HostConfig{
		ContainerRuntime: "docker",
	}

	cm, err := NewContainerManager(ctx, config)
	if err != nil {
		t.Skip("Docker not available in test environment")
		return
	}

	_, err = cm.parseNetworkJSON("invalid json")
	assert.Error(t, err)
}