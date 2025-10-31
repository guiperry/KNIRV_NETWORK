package dataengine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuntDBManager_CreateModel(t *testing.T) {
	manager, err := NewBuntDBManager(":memory:")
	require.NoError(t, err)
	defer manager.Close()

	model := &ModelEntry{
		ID:      "test-model-1",
		Name:    "Test Model",
		Type:    "llm",
		Status:  "active",
		Version: "1.0.0",
		Binary:  "model.bin",
		Config: map[string]interface{}{
			"param1": "value1",
		},
		OwnerID: "user-123",
		Metadata: map[string]interface{}{
			"description": "Test model",
		},
	}

	err = manager.CreateModel(model)
	assert.NoError(t, err)

	// Verify the model was created
	retrieved, err := manager.GetModel("test-model-1")
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "test-model-1", retrieved.ID)
	assert.Equal(t, "Test Model", retrieved.Name)
	assert.Equal(t, "user-123", retrieved.OwnerID)
	assert.NotZero(t, retrieved.CreatedAt)
	assert.NotZero(t, retrieved.UpdatedAt)
}

func TestBuntDBManager_GetModel(t *testing.T) {
	manager, err := NewBuntDBManager(":memory:")
	require.NoError(t, err)
	defer manager.Close()

	model := &ModelEntry{
		ID:      "test-model-2",
		Name:    "Test Model 2",
		OwnerID: "user-456",
	}

	err = manager.CreateModel(model)
	require.NoError(t, err)

	retrieved, err := manager.GetModel("test-model-2")
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "test-model-2", retrieved.ID)

	// Test non-existent model
	_, err = manager.GetModel("non-existent")
	assert.Error(t, err)
}

func TestBuntDBManager_UpdateModel(t *testing.T) {
	manager, err := NewBuntDBManager(":memory:")
	require.NoError(t, err)
	defer manager.Close()

	model := &ModelEntry{
		ID:     "test-model-3",
		Name:   "Original Name",
		Status: "inactive",
	}

	err = manager.CreateModel(model)
	require.NoError(t, err)

	// Update the model
	model.Name = "Updated Name"
	model.Status = "active"
	err = manager.UpdateModel(model)
	assert.NoError(t, err)

	// Verify the update
	retrieved, err := manager.GetModel("test-model-3")
	assert.NoError(t, err)
	assert.Equal(t, "Updated Name", retrieved.Name)
	assert.Equal(t, "active", retrieved.Status)
	assert.True(t, retrieved.UpdatedAt.After(retrieved.CreatedAt))
}

func TestBuntDBManager_ListModelsByOwner(t *testing.T) {
	manager, err := NewBuntDBManager(":memory:")
	require.NoError(t, err)
	defer manager.Close()

	// Create models for different owners
	model1 := &ModelEntry{ID: "model-1", Name: "Model 1", OwnerID: "owner-1"}
	model2 := &ModelEntry{ID: "model-2", Name: "Model 2", OwnerID: "owner-1"}
	model3 := &ModelEntry{ID: "model-3", Name: "Model 3", OwnerID: "owner-2"}

	err = manager.CreateModel(model1)
	require.NoError(t, err)
	err = manager.CreateModel(model2)
	require.NoError(t, err)
	err = manager.CreateModel(model3)
	require.NoError(t, err)

	// List models for owner-1
	models, err := manager.ListModelsByOwner("owner-1")
	assert.NoError(t, err)
	assert.Len(t, models, 2)

	// List models for owner-2
	models, err = manager.ListModelsByOwner("owner-2")
	assert.NoError(t, err)
	assert.Len(t, models, 1)

	// List models for non-existent owner
	models, err = manager.ListModelsByOwner("non-existent")
	assert.NoError(t, err)
	assert.Len(t, models, 0)
}

func TestBuntDBManager_CreateDVENode(t *testing.T) {
	manager, err := NewBuntDBManager(":memory:")
	require.NoError(t, err)
	defer manager.Close()

	node := &DVENodeEntry{
		ID:           "test-node-1",
		Name:         "Test Node",
		Type:         "validator",
		Status:       "active",
		IPAddress:    "192.168.1.100",
		Port:         8080,
		Capabilities: []string{"validation", "computation"},
		Resources: map[string]interface{}{
			"cpu":    4,
			"memory": "8GB",
		},
		Metadata: map[string]interface{}{
			"region": "us-east-1",
		},
	}

	err = manager.CreateDVENode(node)
	assert.NoError(t, err)

	// Verify the node was created
	retrieved, err := manager.GetDVENode("test-node-1")
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "test-node-1", retrieved.ID)
	assert.Equal(t, "Test Node", retrieved.Name)
	assert.NotZero(t, retrieved.CreatedAt)
	assert.NotZero(t, retrieved.UpdatedAt)
}

func TestBuntDBManager_GetDVENode(t *testing.T) {
	manager, err := NewBuntDBManager(":memory:")
	require.NoError(t, err)
	defer manager.Close()

	node := &DVENodeEntry{
		ID:   "test-node-2",
		Name: "Test Node 2",
		Type: "storage",
	}

	err = manager.CreateDVENode(node)
	require.NoError(t, err)

	retrieved, err := manager.GetDVENode("test-node-2")
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "test-node-2", retrieved.ID)

	// Test non-existent node
	_, err = manager.GetDVENode("non-existent")
	assert.Error(t, err)
}

func TestBuntDBManager_UpdateDVENode(t *testing.T) {
	manager, err := NewBuntDBManager(":memory:")
	require.NoError(t, err)
	defer manager.Close()

	node := &DVENodeEntry{
		ID:     "test-node-3",
		Name:   "Original Node",
		Status: "inactive",
	}

	err = manager.CreateDVENode(node)
	require.NoError(t, err)

	// Update the node
	node.Name = "Updated Node"
	node.Status = "active"
	err = manager.UpdateDVENode(node)
	assert.NoError(t, err)

	// Verify the update
	retrieved, err := manager.GetDVENode("test-node-3")
	assert.NoError(t, err)
	assert.Equal(t, "Updated Node", retrieved.Name)
	assert.Equal(t, "active", retrieved.Status)
	assert.True(t, retrieved.UpdatedAt.After(retrieved.CreatedAt))
}

func TestBuntDBManager_ListDVENodes(t *testing.T) {
	manager, err := NewBuntDBManager(":memory:")
	require.NoError(t, err)
	defer manager.Close()

	// Create nodes with different statuses
	node1 := &DVENodeEntry{ID: "node-1", Name: "Node 1", Status: "active"}
	node2 := &DVENodeEntry{ID: "node-2", Name: "Node 2", Status: "active"}
	node3 := &DVENodeEntry{ID: "node-3", Name: "Node 3", Status: "inactive"}

	err = manager.CreateDVENode(node1)
	require.NoError(t, err)
	err = manager.CreateDVENode(node2)
	require.NoError(t, err)
	err = manager.CreateDVENode(node3)
	require.NoError(t, err)

	// List all nodes
	nodes, err := manager.ListDVENodes("")
	assert.NoError(t, err)
	assert.Len(t, nodes, 3)

	// List active nodes only
	nodes, err = manager.ListDVENodes("active")
	assert.NoError(t, err)
	assert.Len(t, nodes, 2)

	// List inactive nodes only
	nodes, err = manager.ListDVENodes("inactive")
	assert.NoError(t, err)
	assert.Len(t, nodes, 1)
}

func TestBuntDBManager_CreateValidationTask(t *testing.T) {
	manager, err := NewBuntDBManager(":memory:")
	require.NoError(t, err)
	defer manager.Close()

	task := &ValidationTaskEntry{
		ID:       "test-task-1",
		Type:     "skillnode",
		Status:   "pending",
		Priority: 5,
		Data: map[string]interface{}{
			"model_id":  "model-123",
			"test_data": "sample input",
		},
		CreatedBy: "user-123",
		Metadata: map[string]interface{}{
			"description": "Test validation task",
		},
	}

	err = manager.CreateValidationTask(task)
	assert.NoError(t, err)

	// Verify the task was created
	retrieved, err := manager.GetValidationTask("test-task-1")
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "test-task-1", retrieved.ID)
	assert.Equal(t, "skillnode", retrieved.Type)
	assert.NotZero(t, retrieved.CreatedAt)
	assert.NotZero(t, retrieved.UpdatedAt)
}

func TestBuntDBManager_GetValidationTask(t *testing.T) {
	manager, err := NewBuntDBManager(":memory:")
	require.NoError(t, err)
	defer manager.Close()

	task := &ValidationTaskEntry{
		ID:     "test-task-2",
		Type:   "model",
		Status: "running",
	}

	err = manager.CreateValidationTask(task)
	require.NoError(t, err)

	retrieved, err := manager.GetValidationTask("test-task-2")
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "test-task-2", retrieved.ID)

	// Test non-existent task
	_, err = manager.GetValidationTask("non-existent")
	assert.Error(t, err)
}

func TestBuntDBManager_UpdateValidationTask(t *testing.T) {
	manager, err := NewBuntDBManager(":memory:")
	require.NoError(t, err)
	defer manager.Close()

	task := &ValidationTaskEntry{
		ID:     "test-task-3",
		Status: "pending",
		Type:   "custom",
	}

	err = manager.CreateValidationTask(task)
	require.NoError(t, err)

	// Update the task
	task.Status = "completed"
	task.Result = map[string]interface{}{
		"score":  95.5,
		"passed": true,
	}
	err = manager.UpdateValidationTask(task)
	assert.NoError(t, err)

	// Verify the update
	retrieved, err := manager.GetValidationTask("test-task-3")
	assert.NoError(t, err)
	assert.Equal(t, "completed", retrieved.Status)
	assert.NotNil(t, retrieved.Result)
	assert.True(t, retrieved.UpdatedAt.After(retrieved.CreatedAt))
}

func TestBuntDBManager_ListValidationTasksByStatus(t *testing.T) {
	manager, err := NewBuntDBManager(":memory:")
	require.NoError(t, err)
	defer manager.Close()

	// Create tasks with different statuses
	task1 := &ValidationTaskEntry{ID: "task-1", Type: "skillnode", Status: "pending"}
	task2 := &ValidationTaskEntry{ID: "task-2", Type: "model", Status: "running"}
	task3 := &ValidationTaskEntry{ID: "task-3", Type: "custom", Status: "completed"}
	task4 := &ValidationTaskEntry{ID: "task-4", Type: "skillnode", Status: "pending"}

	err = manager.CreateValidationTask(task1)
	require.NoError(t, err)
	err = manager.CreateValidationTask(task2)
	require.NoError(t, err)
	err = manager.CreateValidationTask(task3)
	require.NoError(t, err)
	err = manager.CreateValidationTask(task4)
	require.NoError(t, err)

	// List pending tasks
	tasks, err := manager.ListValidationTasksByStatus("pending")
	assert.NoError(t, err)
	assert.Len(t, tasks, 2)

	// List running tasks
	tasks, err = manager.ListValidationTasksByStatus("running")
	assert.NoError(t, err)
	assert.Len(t, tasks, 1)

	// List completed tasks
	tasks, err = manager.ListValidationTasksByStatus("completed")
	assert.NoError(t, err)
	assert.Len(t, tasks, 1)
}

func TestBuntDBManager_CreateValidationResult(t *testing.T) {
	manager, err := NewBuntDBManager(":memory:")
	require.NoError(t, err)
	defer manager.Close()

	result := &ValidationResultEntry{
		ID:     "test-result-1",
		TaskID: "task-123",
		Status: "passed",
		Score:  92.5,
		Details: map[string]interface{}{
			"accuracy": 0.95,
			"latency":  150,
		},
		Errors:      []string{},
		Warnings:    []string{"Minor warning"},
		ValidatedBy: "validator-123",
		Metadata: map[string]interface{}{
			"environment": "test",
		},
	}

	err = manager.CreateValidationResult(result)
	assert.NoError(t, err)

	// Verify the result was created
	retrieved, err := manager.GetValidationResult("test-result-1")
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "test-result-1", retrieved.ID)
	assert.Equal(t, "task-123", retrieved.TaskID)
	assert.Equal(t, 92.5, retrieved.Score)
	assert.NotZero(t, retrieved.CreatedAt)
}

func TestBuntDBManager_GetValidationResult(t *testing.T) {
	manager, err := NewBuntDBManager(":memory:")
	require.NoError(t, err)
	defer manager.Close()

	result := &ValidationResultEntry{
		ID:     "test-result-2",
		TaskID: "task-456",
		Status: "failed",
		Score:  45.0,
	}

	err = manager.CreateValidationResult(result)
	require.NoError(t, err)

	retrieved, err := manager.GetValidationResult("test-result-2")
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "test-result-2", retrieved.ID)

	// Test non-existent result
	_, err = manager.GetValidationResult("non-existent")
	assert.Error(t, err)
}

func TestBuntDBManager_GetValidationResultsByTask(t *testing.T) {
	manager, err := NewBuntDBManager(":memory:")
	require.NoError(t, err)
	defer manager.Close()

	// Create results for different tasks
	result1 := &ValidationResultEntry{ID: "result-1", TaskID: "task-1", Score: 90.0}
	result2 := &ValidationResultEntry{ID: "result-2", TaskID: "task-1", Score: 85.0}
	result3 := &ValidationResultEntry{ID: "result-3", TaskID: "task-2", Score: 95.0}

	err = manager.CreateValidationResult(result1)
	require.NoError(t, err)
	err = manager.CreateValidationResult(result2)
	require.NoError(t, err)
	err = manager.CreateValidationResult(result3)
	require.NoError(t, err)

	// Get results for task-1
	results, err := manager.GetValidationResultsByTask("task-1")
	assert.NoError(t, err)
	assert.Len(t, results, 2)

	// Get results for task-2
	results, err = manager.GetValidationResultsByTask("task-2")
	assert.NoError(t, err)
	assert.Len(t, results, 1)

	// Get results for non-existent task
	results, err = manager.GetValidationResultsByTask("non-existent")
	assert.NoError(t, err)
	assert.Len(t, results, 0)
}
