package modelmanagement

import (
	. "backend-server/internal/objects"
	"fmt"
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

func TestNewModelManagementService(t *testing.T) {
	db := setupTestDB(t)

	service := NewModelManagementService(db)

	assert.NotNil(t, service)
	assert.Equal(t, db, service.db)
	assert.False(t, service.running)
	assert.NotNil(t, service.objects)
	assert.NotNil(t, service.deployments)
	assert.NotNil(t, service.templates)
	assert.NotNil(t, service.runtimeInstances)
	assert.NotNil(t, service.metrics)
	assert.NotNil(t, service.logs)
	assert.NotNil(t, service.events)
	assert.Equal(t, 100, service.maxModels)
	assert.Equal(t, 10, service.maxInstancesPerModel)
	assert.NotNil(t, service.defaultResourceLimits)
	assert.Equal(t, 30*time.Second, service.monitoringInterval)
}

func TestModelManagementService_Start(t *testing.T) {
	db := setupTestDB(t)
	service := NewModelManagementService(db)

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

func TestModelManagementService_Stop(t *testing.T) {
	db := setupTestDB(t)
	service := NewModelManagementService(db)

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

func TestModelManagementService_CreateModel(t *testing.T) {
	db := setupTestDB(t)
	service := NewModelManagementService(db)

	err := service.Start()
	require.NoError(t, err)
	defer service.Stop()

	model := &Model{
		ID:           "test-model-1",
		Name:         "Test Model",
		Description:  "A test model",
		Type:         "WASM",
		Author:       "test-author",
		Version:      "1.0.0",
		FileSize:     1024,
		Tags:         []string{"test", "demo"},
		Status:       "uploaded",
		UploadedAt:   time.Now(),
		LastModified: time.Now(),
	}

	err = service.CreateModel(model)
	assert.NoError(t, err)

	// Verify model is stored in memory
	service.mu.RLock()
	storedModel, exists := service.objects[model.ID]
	service.mu.RUnlock()
	assert.True(t, exists)
	assert.Equal(t, model, storedModel)
}

func TestModelManagementService_GetModel(t *testing.T) {
	db := setupTestDB(t)
	service := NewModelManagementService(db)

	err := service.Start()
	require.NoError(t, err)
	defer service.Stop()

	// Test getting non-existent model
	model, err := service.GetModel("non-existent")
	assert.Error(t, err)
	assert.Nil(t, model)
	assert.Contains(t, err.Error(), "not found")

	// Create and get model
	createdModel := &Model{
		ID:           "test-model-2",
		Name:         "Test Model",
		Description:  "A test model",
		Type:         "WASM",
		Status:       "uploaded",
		UploadedAt:   time.Now(),
		LastModified: time.Now(),
	}

	err = service.CreateModel(createdModel)
	require.NoError(t, err)

	retrievedModel, err := service.GetModel(createdModel.ID)
	assert.NoError(t, err)
	assert.NotNil(t, retrievedModel)
	assert.Equal(t, createdModel.ID, retrievedModel.ID)
	assert.Equal(t, createdModel.Name, retrievedModel.Name)
}

func TestModelManagementService_GetAllModels(t *testing.T) {
	db := setupTestDB(t)
	service := NewModelManagementService(db)

	err := service.Start()
	require.NoError(t, err)
	defer service.Stop()

	// Test empty list
	objects, err := service.GetAllModels(nil)
	assert.NoError(t, err)
	assert.Empty(t, objects)

	// Create multiple objects
	for i := 0; i < 3; i++ {
		model := &Model{
			ID:           fmt.Sprintf("test-model-%d", i+1),
			Name:         fmt.Sprintf("Test Model %d", i+1),
			Description:  fmt.Sprintf("Test model %d", i+1),
			Type:         "WASM",
			Status:       "uploaded",
			UploadedAt:   time.Now(),
			LastModified: time.Now(),
		}
		err := service.CreateModel(model)
		require.NoError(t, err)
	}

	objects, err = service.GetAllModels(nil)
	assert.NoError(t, err)
	assert.Len(t, objects, 3)
}

func TestModelManagementService_UpdateModel(t *testing.T) {
	db := setupTestDB(t)
	service := NewModelManagementService(db)

	err := service.Start()
	require.NoError(t, err)
	defer service.Stop()

	// Create model
	model := &Model{
		ID:           "test-model-update",
		Name:         "Test Model",
		Description:  "A test model",
		Type:         "WASM",
		Status:       "uploaded",
		UploadedAt:   time.Now(),
		LastModified: time.Now(),
	}

	err = service.CreateModel(model)
	require.NoError(t, err)

	// Update model name
	model.Name = "Updated Model"
	model.Description = "An updated test model"
	model.Tags = []string{"updated", "test"}

	err = service.UpdateModel(model.ID, model)
	assert.NoError(t, err)

	// Test updating non-existent model
	nonExistentModel := &Model{ID: "non-existent"}
	err = service.UpdateModel("non-existent", nonExistentModel)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestModelManagementService_DeleteModel(t *testing.T) {
	db := setupTestDB(t)
	service := NewModelManagementService(db)

	err := service.Start()
	require.NoError(t, err)
	defer service.Stop()

	// Test deleting non-existent model
	err = service.DeleteModel("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Create and delete model
	model := &Model{
		ID:           "test-model-delete",
		Name:         "Test Model",
		Type:         "WASM",
		Status:       "uploaded",
		UploadedAt:   time.Now(),
		LastModified: time.Now(),
	}

	err = service.CreateModel(model)
	require.NoError(t, err)

	err = service.DeleteModel(model.ID)
	assert.NoError(t, err)

	// Verify model is removed
	service.mu.RLock()
	_, exists := service.objects[model.ID]
	service.mu.RUnlock()
	assert.False(t, exists)

	// Verify model cannot be retrieved
	_, err = service.GetModel(model.ID)
	assert.Error(t, err)
}

func TestModelManagementService_IsRunning(t *testing.T) {
	db := setupTestDB(t)
	service := NewModelManagementService(db)

	// Test service not running
	assert.False(t, service.IsRunning())

	// Start service
	err := service.Start()
	require.NoError(t, err)
	defer service.Stop()

	// Test service running
	assert.True(t, service.IsRunning())
}
