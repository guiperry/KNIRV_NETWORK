package core

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpProcedureManager(t *testing.T) {
	// Create temporary directories
	tempDir, err := os.MkdirTemp("", "op-procedure-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	configDir := filepath.Join(tempDir, "config")
	require.NoError(t, os.MkdirAll(configDir, 0755))

	// Create logger
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Create API client
	apiClient := NewAPIClient(
		"http://localhost:8080",
		WithLogger(logger),
	)

	// Create file manager
	fileManager, err := NewFileManager(tempDir)
	require.NoError(t, err)

	// Create operational procedure manager
	procedureManager, err := NewOpProcedureManager(apiClient, fileManager, configDir)
	require.NoError(t, err)

	// Create context
	ctx := context.Background()

	t.Run("RegisterProcedure", func(t *testing.T) {
		// Create procedure config
		config := OpProcedureConfig{
			Name:        "test-procedure",
			Version:     "1.0.0",
			Description: "Test procedure",
			Author:      "Test Author",
			License:     "MIT",
			Steps: []OpProcedureStep{
				{
					Name:        "Step 1",
					Description: "First step",
					Type:        "command",
					Action:      "echo",
					Parameters: map[string]interface{}{
						"message": "Hello, world!",
					},
				},
				{
					Name:        "Step 2",
					Description: "Second step",
					Type:        "command",
					Action:      "ls",
					Parameters: map[string]interface{}{
						"path": "/tmp",
					},
				},
			},
			Parameters: []OpProcedureParameter{
				{
					Name:        "verbose",
					Description: "Enable verbose output",
					Type:        "boolean",
					Required:    false,
					Default:     false,
				},
				{
					Name:        "output",
					Description: "Output format",
					Type:        "string",
					Required:    false,
					Default:     "text",
					Enum:        []string{"text", "json", "yaml"},
				},
			},
			Metadata: map[string]string{
				"category": "utility",
				"tags":     "test,example",
			},
		}

		// Register procedure
		err := procedureManager.RegisterProcedure(ctx, config)
		require.NoError(t, err)

		// Check if procedure exists
		assert.True(t, procedureManager.procedureExists("test-procedure", "1.0.0"))

		// Check config file
		configPath := procedureManager.getProcedureConfigPath("test-procedure", "1.0.0")
		_, err = os.Stat(configPath)
		assert.NoError(t, err)
	})

	t.Run("GetProcedure", func(t *testing.T) {
		// Get procedure
		config, err := procedureManager.GetProcedure("test-procedure", "1.0.0")
		require.NoError(t, err)

		// Check procedure config
		assert.Equal(t, "test-procedure", config.Name)
		assert.Equal(t, "1.0.0", config.Version)
		assert.Equal(t, "Test procedure", config.Description)
		assert.Equal(t, "Test Author", config.Author)
		assert.Equal(t, "MIT", config.License)
		assert.Len(t, config.Steps, 2)
		assert.Equal(t, "Step 1", config.Steps[0].Name)
		assert.Equal(t, "command", config.Steps[0].Type)
		assert.Equal(t, "echo", config.Steps[0].Action)
		assert.Equal(t, "Hello, world!", config.Steps[0].Parameters["message"])
		assert.Len(t, config.Parameters, 2)
		assert.Equal(t, "verbose", config.Parameters[0].Name)
		assert.Equal(t, "boolean", config.Parameters[0].Type)
		assert.Equal(t, false, config.Parameters[0].Default)
		assert.Equal(t, "utility", config.Metadata["category"])
		assert.False(t, config.CreatedAt.IsZero())
		assert.False(t, config.UpdatedAt.IsZero())
	})

	t.Run("UpdateProcedure", func(t *testing.T) {
		// Get procedure
		config, err := procedureManager.GetProcedure("test-procedure", "1.0.0")
		require.NoError(t, err)

		// Update procedure config
		config.Description = "Updated test procedure"
		config.Metadata["category"] = "advanced"

		// Update procedure
		err = procedureManager.UpdateProcedure(ctx, *config)
		require.NoError(t, err)

		// Get updated procedure
		updatedConfig, err := procedureManager.GetProcedure("test-procedure", "1.0.0")
		require.NoError(t, err)

		// Check updated procedure config
		assert.Equal(t, "Updated test procedure", updatedConfig.Description)
		assert.Equal(t, "advanced", updatedConfig.Metadata["category"])
		assert.Equal(t, config.CreatedAt, updatedConfig.CreatedAt)
		assert.True(t, updatedConfig.UpdatedAt.After(config.UpdatedAt))
	})

	t.Run("ListProcedures", func(t *testing.T) {
		// Register another procedure
		config := OpProcedureConfig{
			Name:        "test-procedure-2",
			Version:     "1.0.0",
			Description: "Test procedure 2",
			Author:      "Test Author",
			License:     "MIT",
			Steps: []OpProcedureStep{
				{
					Name:        "Step 1",
					Description: "First step",
					Type:        "command",
					Action:      "echo",
					Parameters: map[string]interface{}{
						"message": "Hello, world!",
					},
				},
			},
		}

		err := procedureManager.RegisterProcedure(ctx, config)
		require.NoError(t, err)

		// List procedures
		procedures, err := procedureManager.ListProcedures()
		require.NoError(t, err)

		// Check procedures
		assert.Len(t, procedures, 2)
		assert.Contains(t, []string{procedures[0].Name, procedures[1].Name}, "test-procedure")
		assert.Contains(t, []string{procedures[0].Name, procedures[1].Name}, "test-procedure-2")
	})

	t.Run("ExecuteProcedure", func(t *testing.T) {
		// Execute procedure
		err := procedureManager.ExecuteProcedure(ctx, "test-procedure", "1.0.0", map[string]interface{}{
			"verbose": true,
			"output":  "json",
		})
		require.NoError(t, err)
	})

	t.Run("DeleteProcedure", func(t *testing.T) {
		// Delete procedure
		err := procedureManager.DeleteProcedure("test-procedure", "1.0.0")
		require.NoError(t, err)

		// Check if procedure exists
		assert.False(t, procedureManager.procedureExists("test-procedure", "1.0.0"))

		// Try to get deleted procedure
		_, err = procedureManager.GetProcedure("test-procedure", "1.0.0")
		assert.Error(t, err)
	})

	t.Run("ValidationErrors", func(t *testing.T) {
		// Empty name
		config := OpProcedureConfig{
			Name:        "",
			Version:     "1.0.0",
			Description: "Test procedure",
			Steps: []OpProcedureStep{
				{
					Name:   "Step 1",
					Type:   "command",
					Action: "echo",
				},
			},
		}
		err := procedureManager.RegisterProcedure(ctx, config)
		assert.Error(t, err)

		// Empty version
		config = OpProcedureConfig{
			Name:        "test-procedure-3",
			Version:     "",
			Description: "Test procedure",
			Steps: []OpProcedureStep{
				{
					Name:   "Step 1",
					Type:   "command",
					Action: "echo",
				},
			},
		}
		err = procedureManager.RegisterProcedure(ctx, config)
		assert.Error(t, err)

		// No steps
		config = OpProcedureConfig{
			Name:        "test-procedure-3",
			Version:     "1.0.0",
			Description: "Test procedure",
			Steps:       []OpProcedureStep{},
		}
		err = procedureManager.RegisterProcedure(ctx, config)
		assert.Error(t, err)

		// Invalid step (no name)
		config = OpProcedureConfig{
			Name:        "test-procedure-3",
			Version:     "1.0.0",
			Description: "Test procedure",
			Steps: []OpProcedureStep{
				{
					Name:   "",
					Type:   "command",
					Action: "echo",
				},
			},
		}
		err = procedureManager.RegisterProcedure(ctx, config)
		assert.Error(t, err)

		// Non-existent procedure
		_, err = procedureManager.GetProcedure("non-existent", "1.0.0")
		assert.Error(t, err)

		// Delete non-existent procedure
		err = procedureManager.DeleteProcedure("non-existent", "1.0.0")
		assert.Error(t, err)
	})
}
