package host

import (
	"context"
	"os/exec"
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

	// Test that a quick docker command works (to avoid hanging)
	testCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	testCmd := exec.CommandContext(testCtx, "docker", "ps", "--format", "json")
	if err := testCmd.Run(); err != nil {
		t.Skip("Docker daemon not responding or no permissions")
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
		ID:               "knirv123",
		Name:             "knirv-nexus",
		IsKNIRVContainer: true,
		ServiceType:      "nexus",
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
		name            string
		containerName   string
		containerImage  string
		expectedKNIRV   bool
		expectedService string
		expectedP2P     bool
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
		name          string
		networkName   string
		expectedKNIRV bool
		expectedP2P   bool
		expectedEnc   bool
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

func TestContainerManager_enrichContainerInfo(t *testing.T) {
	ctx := context.Background()
	config := &HostConfig{
		ContainerRuntime: "docker",
	}

	cm, err := NewContainerManager(ctx, config)
	if err != nil {
		t.Skip("Docker not available in test environment")
		return
	}

	// Test with a non-existent container ID
	container := &Container{
		ID: "nonexistent123",
	}

	err = cm.enrichContainerInfo(container)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to inspect container")
}

func TestContainerManager_scanContainers(t *testing.T) {
	ctx := context.Background()
	config := &HostConfig{
		ContainerRuntime: "docker",
	}

	cm, err := NewContainerManager(ctx, config)
	if err != nil {
		t.Skip("Docker not available in test environment")
		return
	}

	err = cm.scanContainers()
	// Should not error even if no containers exist
	assert.NoError(t, err)

	// Verify lastUpdate was set
	cm.mu.RLock()
	assert.True(t, time.Since(cm.lastUpdate) < time.Second)
	cm.mu.RUnlock()
}

func TestContainerManager_scanNetworks(t *testing.T) {
	ctx := context.Background()
	config := &HostConfig{
		ContainerRuntime: "docker",
	}

	cm, err := NewContainerManager(ctx, config)
	if err != nil {
		t.Skip("Docker not available in test environment")
		return
	}

	err = cm.scanNetworks()
	// Should not error even if no networks exist
	assert.NoError(t, err)
}

func TestContainerManager_setupKNIRVNetworks(t *testing.T) {
	ctx := context.Background()
	config := &HostConfig{
		ContainerRuntime: "docker",
	}

	cm, err := NewContainerManager(ctx, config)
	if err != nil {
		t.Skip("Docker not available in test environment")
		return
	}

	// Test when network doesn't exist (would require docker permissions)
	err = cm.setupKNIRVNetworks()
	// This might fail due to permissions, but should not panic
	_ = err // We don't assert since it depends on docker permissions
}

func TestContainerManager_monitorLoop(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	config := &HostConfig{
		ContainerRuntime: "docker",
	}

	cm, err := NewContainerManager(ctx, config)
	if err != nil {
		t.Skip("Docker not available in test environment")
		return
	}

	// Start monitoring
	cm.running = true
	go cm.monitorLoop()

	// Wait for context to timeout
	time.Sleep(150 * time.Millisecond)

	// The monitorLoop should have stopped due to context timeout
	// We can't easily check if the goroutine stopped, but the test passes if no panic
}

func TestContainerManager_GetContainerList_Empty(t *testing.T) {
	ctx := context.Background()
	config := &HostConfig{
		ContainerRuntime: "docker",
	}

	cm, err := NewContainerManager(ctx, config)
	if err != nil {
		t.Skip("Docker not available in test environment")
		return
	}

	containers, err := cm.GetContainerList()
	require.NoError(t, err)
	assert.Empty(t, containers)
}

func TestContainerManager_GetKNIRVContainers_Empty(t *testing.T) {
	ctx := context.Background()
	config := &HostConfig{
		ContainerRuntime: "docker",
	}

	cm, err := NewContainerManager(ctx, config)
	if err != nil {
		t.Skip("Docker not available in test environment")
		return
	}

	containers, err := cm.GetKNIRVContainers()
	require.NoError(t, err)
	assert.Empty(t, containers)
}

func TestContainerManager_verifyRuntime_Invalid(t *testing.T) {
	ctx := context.Background()
	config := &HostConfig{
		ContainerRuntime: "invalidruntime",
	}

	cm, err := NewContainerManager(ctx, config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "container runtime verification failed")

	// Test that cm is nil when creation fails
	assert.Nil(t, cm)
}

func TestContainerManager_identifyKNIRVContainer_EdgeCases(t *testing.T) {
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
		name            string
		containerName   string
		containerImage  string
		expectedKNIRV   bool
		expectedService string
		expectedP2P     bool
	}{
		{"empty name and image", "", "", false, "", false},
		{"uppercase KNIRV", "KNIRV-NEXUS", "KNIRV/NEXUS:LATEST", true, "knirv-other", false},
		{"mixed case", "Dve-Manager", "knirv/dve:latest", true, "dve-manager", true},
		{"partial match", "myknirvapp", "nginx:latest", true, "knirv-other", false},
		{"no match", "postgres", "postgres:13", false, "", false},
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

func TestContainerManager_identifyKNIRVNetwork_EdgeCases(t *testing.T) {
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
		name          string
		networkName   string
		expectedKNIRV bool
		expectedP2P   bool
		expectedEnc   bool
	}{
		{"empty name", "", false, false, false},
		{"uppercase KNIRV", "KNIRV-NEXUS", true, false, true},
		{"partial match", "myknirvnet", true, false, true},
		{"no match", "bridge", false, false, false},
		{"p2p network", "knirv-p2p-net", true, true, true},
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

func Test_getString_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		key      string
		expected string
	}{
		{"nil map", nil, "key", ""},
		{"empty map", map[string]interface{}{}, "key", ""},
		{"non-string value", map[string]interface{}{"key": 123}, "key", ""},
		{"nil value", map[string]interface{}{"key": nil}, "key", ""},
		{"valid string", map[string]interface{}{"key": "value"}, "key", "value"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getString(tt.input, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}
