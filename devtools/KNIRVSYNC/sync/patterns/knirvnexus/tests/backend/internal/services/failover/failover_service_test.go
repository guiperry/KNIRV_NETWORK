package failover

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"backend_server/internal/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFailoverService(t *testing.T) {
	config := &FailoverConfig{
		HealthCheckInterval: 30 * time.Second,
		HealthCheckTimeout:  5 * time.Second,
		FailoverThreshold:   3,
		EnableAutoFailover:  true,
		RootNodes:           []string{"http://root1:8080", "http://root2:8080"},
	}

	// Create a temporary database for testing
	db, err := database.NewBuntDB(":memory:")
	require.NoError(t, err)
	defer db.Close()

	service := NewFailoverService(config, db)

	assert.NotNil(t, service)
	assert.Equal(t, config, service.config)
	assert.NotNil(t, service.db)
	assert.NotNil(t, service.nodes)
	assert.NotNil(t, service.ctx)
	assert.NotNil(t, service.cancel)
}

func TestFailoverService_Start_Stop(t *testing.T) {
	config := &FailoverConfig{
		HealthCheckInterval: 100 * time.Millisecond, // Very short for testing
		HealthCheckTimeout:  50 * time.Millisecond,
		FailoverThreshold:   2,
		EnableAutoFailover:  false, // Disable to avoid complexity in test
		RootNodes:           []string{},
	}

	db, err := database.NewBuntDB(":memory:")
	require.NoError(t, err)
	defer db.Close()

	service := NewFailoverService(config, db)

	// Test Start
	err = service.Start()
	assert.NoError(t, err)

	// Give it a moment to start
	time.Sleep(10 * time.Millisecond)

	// Test Stop
	err = service.Stop()
	assert.NoError(t, err)
}

func TestFailoverService_RegisterBootnode(t *testing.T) {
	config := &FailoverConfig{
		HealthCheckInterval: 30 * time.Second,
		HealthCheckTimeout:  5 * time.Second,
		FailoverThreshold:   3,
		EnableAutoFailover:  true,
		RootNodes:           []string{},
	}

	db, err := database.NewBuntDB(":memory:")
	require.NoError(t, err)
	defer db.Close()

	service := NewFailoverService(config, db)

	// Register a bootnode
	service.RegisterBootnode("bootnode-1", "http://bootnode1:8080", 10)

	// Check that it was registered
	nodes := service.GetAllNodes()
	assert.Contains(t, nodes, "bootnode-1")

	node := nodes["bootnode-1"]
	assert.Equal(t, "bootnode-1", node.ID)
	assert.Equal(t, RoleBootnode, node.Role)
	assert.Equal(t, "http://bootnode1:8080", node.Endpoint)
	assert.Equal(t, 10, node.Priority)
}

func TestFailoverService_GetCurrentRoot(t *testing.T) {
	config := &FailoverConfig{
		HealthCheckInterval: 30 * time.Second,
		HealthCheckTimeout:  5 * time.Second,
		FailoverThreshold:   3,
		EnableAutoFailover:  true,
		RootNodes:           []string{"http://root1:8080"},
	}

	db, err := database.NewBuntDB(":memory:")
	require.NoError(t, err)
	defer db.Close()

	service := NewFailoverService(config, db)

	// Initially should be empty since no health checks have run
	assert.Equal(t, "", service.GetCurrentRoot())
}

func TestFailoverService_GetNodeStatus(t *testing.T) {
	config := &FailoverConfig{
		HealthCheckInterval: 30 * time.Second,
		HealthCheckTimeout:  5 * time.Second,
		FailoverThreshold:   3,
		EnableAutoFailover:  true,
		RootNodes:           []string{},
	}

	db, err := database.NewBuntDB(":memory:")
	require.NoError(t, err)
	defer db.Close()

	service := NewFailoverService(config, db)

	// Register a bootnode
	service.RegisterBootnode("bootnode-1", "http://bootnode1:8080", 10)

	// Check status of existing node
	status, exists := service.GetNodeStatus("bootnode-1")
	assert.True(t, exists)
	assert.Equal(t, StatusUnknown, status)

	// Check status of non-existing node
	status, exists = service.GetNodeStatus("nonexistent")
	assert.False(t, exists)
	assert.Equal(t, StatusUnknown, status)
}

func TestFailoverService_IsFailoverEnabled(t *testing.T) {
	config := &FailoverConfig{
		HealthCheckInterval: 30 * time.Second,
		HealthCheckTimeout:  5 * time.Second,
		FailoverThreshold:   3,
		EnableAutoFailover:  true,
		RootNodes:           []string{},
	}

	db, err := database.NewBuntDB(":memory:")
	require.NoError(t, err)
	defer db.Close()

	service := NewFailoverService(config, db)

	assert.True(t, service.IsFailoverEnabled())
}

func TestFailoverService_GetAllNodes(t *testing.T) {
	config := &FailoverConfig{
		HealthCheckInterval: 30 * time.Second,
		HealthCheckTimeout:  5 * time.Second,
		FailoverThreshold:   3,
		EnableAutoFailover:  true,
		RootNodes:           []string{"http://root1:8080"},
	}

	db, err := database.NewBuntDB(":memory:")
	require.NoError(t, err)
	defer db.Close()

	service := NewFailoverService(config, db)

	// Initialize nodes (this is normally done in Start())
	service.initializeNodes()

	// Register a bootnode
	service.RegisterBootnode("bootnode-1", "http://bootnode1:8080", 10)

	// Get all nodes
	nodes := service.GetAllNodes()

	// Should have root node and bootnode
	assert.Contains(t, nodes, "root-1")
	assert.Contains(t, nodes, "bootnode-1")

	// Verify root node
	rootNode := nodes["root-1"]
	assert.NotNil(t, rootNode)
	assert.Equal(t, RoleRoot, rootNode.Role)
	assert.Equal(t, "http://root1:8080", rootNode.Endpoint)

	// Verify bootnode
	bootNode := nodes["bootnode-1"]
	assert.NotNil(t, bootNode)
	assert.Equal(t, RoleBootnode, bootNode.Role)
	assert.Equal(t, "http://bootnode1:8080", bootNode.Endpoint)
}

func TestFailoverService_HealthCheck(t *testing.T) {
	config := &FailoverConfig{
		HealthCheckInterval: 30 * time.Second,
		HealthCheckTimeout:  5 * time.Second,
		FailoverThreshold:   3,
		EnableAutoFailover:  true,
		RootNodes:           []string{},
	}

	db, err := database.NewBuntDB(":memory:")
	require.NoError(t, err)
	defer db.Close()

	service := NewFailoverService(config, db)

	// Create a test server that responds with 200 OK
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "healthy"}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer testServer.Close()

	// Register a bootnode pointing to our test server
	service.RegisterBootnode("test-bootnode", testServer.URL, 10)

	// Manually trigger health check
	node := service.nodes["test-bootnode"]
	service.checkNodeHealth(node)

	// Check that status was updated to healthy
	status, exists := service.GetNodeStatus("test-bootnode")
	assert.True(t, exists)
	assert.Equal(t, StatusHealthy, status)
}

func TestFailoverService_UnhealthyNode(t *testing.T) {
	config := &FailoverConfig{
		HealthCheckInterval: 30 * time.Second,
		HealthCheckTimeout:  5 * time.Second,
		FailoverThreshold:   3,
		EnableAutoFailover:  true,
		RootNodes:           []string{},
	}

	db, err := database.NewBuntDB(":memory:")
	require.NoError(t, err)
	defer db.Close()

	service := NewFailoverService(config, db)

	// Register a bootnode pointing to a non-existent endpoint
	service.RegisterBootnode("test-bootnode", "http://nonexistent:9999", 10)

	// Manually trigger health check
	node := service.nodes["test-bootnode"]
	service.checkNodeHealth(node)

	// Check that status was updated to unhealthy
	status, exists := service.GetNodeStatus("test-bootnode")
	assert.True(t, exists)
	assert.Equal(t, StatusUnhealthy, status)
}

func TestFailoverService_FailoverLogic(t *testing.T) {
	config := &FailoverConfig{
		HealthCheckInterval: 30 * time.Second,
		HealthCheckTimeout:  5 * time.Second,
		FailoverThreshold:   3,
		EnableAutoFailover:  true,
		RootNodes:           []string{"http://failing-root:8080"},
	}

	db, err := database.NewBuntDB(":memory:")
	require.NoError(t, err)
	defer db.Close()

	service := NewFailoverService(config, db)

	// Initialize nodes first
	service.initializeNodes()

	// Set up initial state: root is unhealthy, bootnode is healthy
	if rootNode, exists := service.nodes["root-1"]; exists {
		rootNode.Status = StatusUnhealthy
		service.currentRoot = "root-1"
	}

	// Register a healthy bootnode
	service.RegisterBootnode("bootnode-1", "http://healthy-bootnode:8080", 10)
	if bootNode, exists := service.nodes["bootnode-1"]; exists {
		bootNode.Status = StatusHealthy
	}

	// Simulate failover conditions
	failoverCount := make(map[string]int)
	failoverCount["root-1"] = config.FailoverThreshold // Meet threshold

	// Check failover conditions
	service.checkFailoverConditions(failoverCount)

	// After failover, current root should be the bootnode
	assert.Equal(t, "bootnode-1", service.GetCurrentRoot())

	// Bootnode should now be promoted to root
	if bootnode, exists := service.nodes["bootnode-1"]; exists {
		assert.Equal(t, RoleRoot, bootnode.Role)
	}
}