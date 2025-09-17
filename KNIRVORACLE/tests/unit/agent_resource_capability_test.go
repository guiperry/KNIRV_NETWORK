package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResourceCapabilityIntegration tests the complete resource capability workflow
func TestResourceCapabilityIntegration(t *testing.T) {
	// Setup test environment
	tempDir, _, wallet, agentManager := setupTestEnvironment(t)
	defer os.RemoveAll(tempDir)

	// Create a test agent
	agent, err := agentManager.CreateAgent(
		"Test Agent",
		"Agent for resource capability testing",
		"https://example.com/image.png",
		wallet.GetAddress(),
		"test",
		map[string]interface{}{
			"version": "1.0.0",
		},
	)
	require.NoError(t, err)
	require.NotNil(t, agent)

	// Test adding different types of resource capabilities
	t.Run("AddFileResourceCapability", func(t *testing.T) {
		metadata := map[string]interface{}{
			"file_path":     "/path/to/test/file.txt",
			"access_method": "read",
			"data_format":   "text/plain",
			"size_bytes":    float64(1024),
		}

		capability, err := agentManager.AddResourceCapabilityToAgent(
			agent.ID,
			"Test File Resource",
			"A test file resource capability",
			"FILE",
			metadata,
		)
		require.NoError(t, err)
		assert.NotNil(t, capability)
		assert.NotEmpty(t, capability.ID)

		// Verify the capability was added
		updatedAgent, err := agentManager.GetAgent(agent.ID)
		require.NoError(t, err)
		assert.Len(t, updatedAgent.Capabilities, 1)
		assert.Equal(t, "resource", updatedAgent.Capabilities[0].CapabilityType)
		assert.Equal(t, "FILE", updatedAgent.Capabilities[0].Metadata["resource_type"])
	})

	t.Run("AddAPIResourceCapability", func(t *testing.T) {
		metadata := map[string]interface{}{
			"api_endpoint":  "https://api.example.com/v1/data",
			"access_method": "GET",
			"auth_type":     "bearer",
		}

		capability, err := agentManager.AddResourceCapabilityToAgent(
			agent.ID,
			"Test API Resource",
			"A test API resource capability",
			"API",
			metadata,
		)
		require.NoError(t, err)
		assert.NotNil(t, capability)
		assert.NotEmpty(t, capability.ID)
	})

	t.Run("AddDatasetResourceCapability", func(t *testing.T) {
		metadata := map[string]interface{}{
			"dataset_path": "/datasets/test_dataset.csv",
			"data_format":  "csv",
			"size_bytes":   float64(5120),
			"schema": map[string]interface{}{
				"columns": []string{"id", "name", "value"},
				"types":   []string{"int", "string", "float"},
			},
		}

		capabilityID, err := agentManager.AddResourceCapabilityToAgent(
			agent.ID,
			"Test Dataset Resource",
			"A test dataset resource capability",
			"DATASET",
			metadata,
		)
		require.NoError(t, err)
		assert.NotEmpty(t, capabilityID)
	})

	t.Run("AddModelArtifactResourceCapability", func(t *testing.T) {
		metadata := map[string]interface{}{
			"model_path": "/models/test_model.pkl",
			"model_type": "classification",
			"version":    "1.0.0",
			"framework":  "scikit-learn",
			"size_bytes": float64(2048),
		}

		capabilityID, err := agentManager.AddResourceCapabilityToAgent(
			agent.ID,
			"Test Model Resource",
			"A test model artifact resource capability",
			"MODEL_ARTIFACT",
			metadata,
		)
		require.NoError(t, err)
		assert.NotEmpty(t, capabilityID)
	})

	t.Run("AddServiceResourceCapability", func(t *testing.T) {
		metadata := map[string]interface{}{
			"service_url":  "https://service.example.com",
			"service_type": "inference",
			"protocol":     "REST",
		}

		capabilityID, err := agentManager.AddResourceCapabilityToAgent(
			agent.ID,
			"Test Service Resource",
			"A test service resource capability",
			"SERVICE",
			metadata,
		)
		require.NoError(t, err)
		assert.NotEmpty(t, capabilityID)
	})
}

// TestResourceCapabilityValidation tests validation of resource capabilities
func TestResourceCapabilityValidation(t *testing.T) {
	// Setup test environment
	tempDir, _, wallet, agentManager := setupTestEnvironment(t)
	defer os.RemoveAll(tempDir)

	// Create a test agent
	agent, err := agentManager.CreateAgent(
		"Test Agent",
		"Agent for validation testing",
		"https://example.com/image.png",
		wallet.GetAddress(),
		"test",
		map[string]interface{}{},
	)
	require.NoError(t, err)

	t.Run("InvalidResourceType", func(t *testing.T) {
		metadata := map[string]interface{}{
			"test_field": "test_value",
		}

		_, err := agentManager.AddResourceCapabilityToAgent(
			agent.ID,
			"Invalid Resource",
			"A resource with invalid type",
			"INVALID_TYPE",
			metadata,
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported resource type")
	})

	t.Run("EmptyAgentID", func(t *testing.T) {
		metadata := map[string]interface{}{
			"file_path":     "/test/file.txt",
			"access_method": "read",
		}

		_, err := agentManager.AddResourceCapabilityToAgent(
			"",
			"Test Resource",
			"A test resource",
			"FILE",
			metadata,
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "agent ID is required")
	})

	t.Run("NonexistentAgent", func(t *testing.T) {
		metadata := map[string]interface{}{
			"file_path":     "/test/file.txt",
			"access_method": "read",
		}

		_, err := agentManager.AddResourceCapabilityToAgent(
			"nonexistent-agent-id",
			"Test Resource",
			"A test resource",
			"FILE",
			metadata,
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "agent not found")
	})
}

// TestResourceCapabilityInvocation tests resource capability invocation
func TestResourceCapabilityInvocation(t *testing.T) {
	// Setup test environment
	tempDir, _, wallet, agentManager := setupTestEnvironment(t)
	defer os.RemoveAll(tempDir)

	// Create a test agent
	agent, err := agentManager.CreateAgent(
		"Test Agent",
		"Agent for invocation testing",
		"https://example.com/image.png",
		wallet.GetAddress(),
		"test",
		map[string]interface{}{},
	)
	require.NoError(t, err)

	// Add a file resource capability
	metadata := map[string]interface{}{
		"file_path":     "/path/to/test/file.txt",
		"access_method": "read",
		"data_format":   "text/plain",
		"size_bytes":    float64(1024),
	}

	capability, err := agentManager.AddResourceCapabilityToAgent(
		agent.ID,
		"Test File Resource",
		"A test file resource capability",
		"FILE",
		metadata,
	)
	require.NoError(t, err)

	t.Run("ValidInvocation", func(t *testing.T) {
		parameters := map[string]interface{}{
			"operation": "read",
		}

		result, err := agentManager.InvokeResourceCapability(
			agent.ID,
			capability.ID,
			parameters,
			wallet.GetAddress(), // Owner has access
		)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "FILE", result["resource_type"])
		assert.Equal(t, capability.ID, result["capability_id"])
		assert.NotNil(t, result["invoked_at"])
	})

	t.Run("UnauthorizedInvocation", func(t *testing.T) {
		parameters := map[string]interface{}{
			"operation": "read",
		}

		_, err := agentManager.InvokeResourceCapability(
			agent.ID,
			capability.ID,
			parameters,
			"unauthorized-user",
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("NonexistentCapability", func(t *testing.T) {
		parameters := map[string]interface{}{
			"operation": "read",
		}

		_, err := agentManager.InvokeResourceCapability(
			agent.ID,
			"nonexistent-capability-id",
			parameters,
			wallet.GetAddress(),
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "resource capability not found")
	})
}

// TestResourceCapabilityGroups tests resource capability grouping functionality
func TestResourceCapabilityGroups(t *testing.T) {
	// Setup test environment
	tempDir, _, wallet, agentManager := setupTestEnvironment(t)
	defer os.RemoveAll(tempDir)

	// Create a test agent
	agent, err := agentManager.CreateAgent(
		"Test Agent",
		"Agent for group testing",
		"https://example.com/image.png",
		wallet.GetAddress(),
		"test",
		map[string]interface{}{},
	)
	require.NoError(t, err)

	// Add multiple resource capabilities
	var capabilityIDs []string

	for i := 0; i < 3; i++ {
		metadata := map[string]interface{}{
			"file_path":     fmt.Sprintf("/path/to/file%d.txt", i),
			"access_method": "read",
			"data_format":   "text/plain",
		}

		capability, err := agentManager.AddResourceCapabilityToAgent(
			agent.ID,
			fmt.Sprintf("Test File Resource %d", i),
			fmt.Sprintf("Test file resource capability %d", i),
			"FILE",
			metadata,
		)
		require.NoError(t, err)
		capabilityIDs = append(capabilityIDs, capability.ID)
	}

	t.Run("CreateResourceCapabilityGroup", func(t *testing.T) {
		err := agentManager.CreateResourceCapabilityGroup(
			agent.ID,
			"Test File Group",
			"A group of test file resources",
			capabilityIDs,
		)
		require.NoError(t, err)

		// Verify the group was created
		groups, err := agentManager.GetResourceCapabilityGroups(agent.ID)
		require.NoError(t, err)
		assert.Len(t, groups, 1)
		if len(groups) > 0 {
			assert.Equal(t, "Test File Group", groups[0]["name"])

			// Handle both []string and []interface{} types for resource_capability_ids
			if resourceCapIds, ok := groups[0]["resource_capability_ids"].([]string); ok {
				assert.Len(t, resourceCapIds, 3)
			} else if resourceCapIdsInterface, ok := groups[0]["resource_capability_ids"].([]interface{}); ok {
				assert.Len(t, resourceCapIdsInterface, 3)
				// Verify they are all strings
				for _, id := range resourceCapIdsInterface {
					assert.IsType(t, "", id, "resource capability ID should be a string")
				}
			} else {
				t.Errorf("resource_capability_ids field not found or wrong type. Group: %+v", groups[0])
			}
		}
	})

	t.Run("CreateGroupWithNonexistentCapability", func(t *testing.T) {
		invalidCapabilityIDs := append(capabilityIDs, "nonexistent-capability-id")

		err := agentManager.CreateResourceCapabilityGroup(
			agent.ID,
			"Invalid Group",
			"A group with invalid capability",
			invalidCapabilityIDs,
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "resource capability nonexistent-capability-id not found")
	})
}
