package agentmanagement

import (
	"fmt"
	"nexus-backend/internal/models"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/buntdb"
)

func setupTestDB(t *testing.T) *buntdb.DB {
	// Create temporary database file
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := buntdb.Open(dbPath)
	require.NoError(t, err)

	t.Cleanup(func() {
		db.Close()
		os.Remove(dbPath)
	})

	return db
}

func TestNewAgentManagementService(t *testing.T) {
	db := setupTestDB(t)

	service := NewAgentManagementService(db)

	assert.NotNil(t, service)
	assert.Equal(t, db, service.db)
	assert.False(t, service.running)
	assert.NotNil(t, service.agents)
	assert.NotNil(t, service.deployments)
	assert.NotNil(t, service.templates)
	assert.NotNil(t, service.runtimeInstances)
	assert.NotNil(t, service.metrics)
	assert.NotNil(t, service.logs)
	assert.NotNil(t, service.events)
	assert.Equal(t, 100, service.maxAgents)
	assert.Equal(t, 10, service.maxInstancesPerAgent)
	assert.NotNil(t, service.defaultResourceLimits)
	assert.Equal(t, 30*time.Second, service.monitoringInterval)
}

func TestAgentManagementService_Start(t *testing.T) {
	db := setupTestDB(t)
	service := NewAgentManagementService(db)

	err := service.Start()
	assert.NoError(t, err)
	assert.True(t, service.running)

	// Test starting already running service
	err = service.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")

	// Cleanup
	err = service.Stop()
	assert.NoError(t, err)
}

func TestAgentManagementService_Stop(t *testing.T) {
	db := setupTestDB(t)
	service := NewAgentManagementService(db)

	// Test stopping non-running service
	err := service.Stop()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")

	// Start and then stop
	err = service.Start()
	require.NoError(t, err)

	err = service.Stop()
	assert.NoError(t, err)
	assert.False(t, service.running)
}

func TestAgentManagementService_CreateAgent(t *testing.T) {
	db := setupTestDB(t)
	service := NewAgentManagementService(db)

	err := service.Start()
	require.NoError(t, err)
	defer service.Stop()

	agent := &models.Agent{
		ID:           "test-agent-1",
		Name:         "Test Agent",
		Description:  "A test agent",
		Type:         "WASM",
		Author:       "test-author",
		Version:      "1.0.0",
		FileSize:     1024,
		Tags:         []string{"test", "demo"},
		Status:       "uploaded",
		UploadedAt:   time.Now(),
		LastModified: time.Now(),
	}

	err = service.CreateAgent(agent)
	assert.NoError(t, err)

	// Verify agent is stored in memory
	service.mu.RLock()
	storedAgent, exists := service.agents[agent.ID]
	service.mu.RUnlock()
	assert.True(t, exists)
	assert.Equal(t, agent, storedAgent)
}

func TestAgentManagementService_GetAgent(t *testing.T) {
	db := setupTestDB(t)
	service := NewAgentManagementService(db)

	err := service.Start()
	require.NoError(t, err)
	defer service.Stop()

	// Test getting non-existent agent
	agent, err := service.GetAgent("non-existent")
	assert.Error(t, err)
	assert.Nil(t, agent)
	assert.Contains(t, err.Error(), "not found")

	// Create and get agent
	createdAgent := &models.Agent{
		ID:           "test-agent-2",
		Name:         "Test Agent",
		Description:  "A test agent",
		Type:         "WASM",
		Status:       "uploaded",
		UploadedAt:   time.Now(),
		LastModified: time.Now(),
	}

	err = service.CreateAgent(createdAgent)
	require.NoError(t, err)

	retrievedAgent, err := service.GetAgent(createdAgent.ID)
	assert.NoError(t, err)
	assert.NotNil(t, retrievedAgent)
	assert.Equal(t, createdAgent.ID, retrievedAgent.ID)
	assert.Equal(t, createdAgent.Name, retrievedAgent.Name)
}

func TestAgentManagementService_GetAllAgents(t *testing.T) {
	db := setupTestDB(t)
	service := NewAgentManagementService(db)

	err := service.Start()
	require.NoError(t, err)
	defer service.Stop()

	// Test empty list
	agents, err := service.GetAllAgents(nil)
	assert.NoError(t, err)
	assert.Empty(t, agents)

	// Create multiple agents
	for i := 0; i < 3; i++ {
		agent := &models.Agent{
			ID:           fmt.Sprintf("test-agent-%d", i+1),
			Name:         fmt.Sprintf("Test Agent %d", i+1),
			Description:  fmt.Sprintf("Test agent %d", i+1),
			Type:         "WASM",
			Status:       "uploaded",
			UploadedAt:   time.Now(),
			LastModified: time.Now(),
		}
		err := service.CreateAgent(agent)
		require.NoError(t, err)
	}

	agents, err = service.GetAllAgents(nil)
	assert.NoError(t, err)
	assert.Len(t, agents, 3)
}

func TestAgentManagementService_UpdateAgent(t *testing.T) {
	db := setupTestDB(t)
	service := NewAgentManagementService(db)

	err := service.Start()
	require.NoError(t, err)
	defer service.Stop()

	// Create agent
	agent := &models.Agent{
		ID:           "test-agent-update",
		Name:         "Test Agent",
		Description:  "A test agent",
		Type:         "WASM",
		Status:       "uploaded",
		UploadedAt:   time.Now(),
		LastModified: time.Now(),
	}

	err = service.CreateAgent(agent)
	require.NoError(t, err)

	// Update agent name
	agent.Name = "Updated Agent"
	agent.Description = "An updated test agent"
	agent.Tags = []string{"updated", "test"}

	err = service.UpdateAgent(agent.ID, agent)
	assert.NoError(t, err)

	// Test updating non-existent agent
	nonExistentAgent := &models.Agent{ID: "non-existent"}
	err = service.UpdateAgent("non-existent", nonExistentAgent)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestAgentManagementService_DeleteAgent(t *testing.T) {
	db := setupTestDB(t)
	service := NewAgentManagementService(db)

	err := service.Start()
	require.NoError(t, err)
	defer service.Stop()

	// Test deleting non-existent agent
	err = service.DeleteAgent("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Create and delete agent
	agent := &models.Agent{
		ID:           "test-agent-delete",
		Name:         "Test Agent",
		Type:         "WASM",
		Status:       "uploaded",
		UploadedAt:   time.Now(),
		LastModified: time.Now(),
	}

	err = service.CreateAgent(agent)
	require.NoError(t, err)

	err = service.DeleteAgent(agent.ID)
	assert.NoError(t, err)

	// Verify agent is removed
	service.mu.RLock()
	_, exists := service.agents[agent.ID]
	service.mu.RUnlock()
	assert.False(t, exists)

	// Verify agent cannot be retrieved
	_, err = service.GetAgent(agent.ID)
	assert.Error(t, err)
}

func TestAgentManagementService_IsRunning(t *testing.T) {
	db := setupTestDB(t)
	service := NewAgentManagementService(db)

	// Test service not running
	assert.False(t, service.IsRunning())

	// Start service
	err := service.Start()
	require.NoError(t, err)
	defer service.Stop()

	// Test service running
	assert.True(t, service.IsRunning())
}
