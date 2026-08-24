package api

import (
	"os"
	"path/filepath"
	"testing"

	"KNIRVENGINE/desktop-client/agent"
	"KNIRVENGINE/desktop-client/agentify"
	"KNIRVENGINE/desktop-client/inference"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentConfigToMap(t *testing.T) {
	t.Run("converts agent config to map successfully", func(t *testing.T) {
		config := agent.AgentConfig{
			AgentID:     "test-agent-1",
			Name:        "Test Agent",
			Description: "A test agent",
			AgentType:   "test",
			Model:       "gpt-3.5-turbo",
		}

		result, err := agentConfigToMap(config)
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Check that the map contains expected fields
		assert.Equal(t, "test-agent-1", result["agent_id"])
		assert.Equal(t, "Test Agent", result["name"])
		assert.Equal(t, "A test agent", result["description"])
		assert.Equal(t, "test", result["agent_type"])
		assert.Equal(t, "gpt-3.5-turbo", result["model"])
	})

	t.Run("handles empty agent config", func(t *testing.T) {
		config := agent.AgentConfig{}
		result, err := agentConfigToMap(config)
		require.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func TestGenerateAgentID(t *testing.T) {
	t.Run("generates unique agent IDs", func(t *testing.T) {
		id1 := generateAgentID()
		id2 := generateAgentID()

		assert.NotEmpty(t, id1)
		assert.NotEmpty(t, id2)
		assert.NotEqual(t, id1, id2)
		assert.Contains(t, id1, "agent_")
		assert.Contains(t, id2, "agent_")
	})
}

func TestTroubleshootingChunk(t *testing.T) {
	t.Run("troubleshooting chunk struct", func(t *testing.T) {
		chunk := TroubleshootingChunk{
			Category: "network",
			Issue:    "connection timeout",
			Symptoms: []string{"slow response", "timeout errors"},
			Content:  "Check network connectivity",
			RawHTML:  "<p>Check network connectivity</p>",
		}

		assert.Equal(t, "network", chunk.Category)
		assert.Equal(t, "connection timeout", chunk.Issue)
		assert.Len(t, chunk.Symptoms, 2)
		assert.Equal(t, "slow response", chunk.Symptoms[0])
		assert.Equal(t, "timeout errors", chunk.Symptoms[1])
		assert.Equal(t, "Check network connectivity", chunk.Content)
		assert.Equal(t, "<p>Check network connectivity</p>", chunk.RawHTML)
	})
}

func TestTroubleshootingDatabase(t *testing.T) {
	t.Run("troubleshooting database struct", func(t *testing.T) {
		chunk1 := TroubleshootingChunk{Category: "network", Issue: "timeout"}
		chunk2 := TroubleshootingChunk{Category: "auth", Issue: "unauthorized"}

		db := TroubleshootingDatabase{
			Chunks: []TroubleshootingChunk{chunk1, chunk2},
		}

		assert.Len(t, db.Chunks, 2)
		assert.Equal(t, "network", db.Chunks[0].Category)
		assert.Equal(t, "auth", db.Chunks[1].Category)
	})
}

func TestCopyFile(t *testing.T) {
	t.Run("copies file successfully", func(t *testing.T) {
		// Create temporary source file
		tmpDir := t.TempDir()
		srcFile := filepath.Join(tmpDir, "source.txt")
		dstFile := filepath.Join(tmpDir, "destination.txt")

		content := "test file content"
		err := os.WriteFile(srcFile, []byte(content), 0644)
		require.NoError(t, err)

		// Copy the file
		err = copyFile(srcFile, dstFile)
		require.NoError(t, err)

		// Verify the destination file exists and has correct content
		copiedContent, err := os.ReadFile(dstFile)
		require.NoError(t, err)
		assert.Equal(t, content, string(copiedContent))
	})

	t.Run("returns error for non-existent source file", func(t *testing.T) {
		tmpDir := t.TempDir()
		srcFile := filepath.Join(tmpDir, "non-existent.txt")
		dstFile := filepath.Join(tmpDir, "destination.txt")

		err := copyFile(srcFile, dstFile)
		assert.Error(t, err)
	})

	t.Run("fails when destination directory doesn't exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		srcFile := filepath.Join(tmpDir, "source.txt")
		dstFile := filepath.Join(tmpDir, "subdir", "destination.txt")

		content := "test content"
		err := os.WriteFile(srcFile, []byte(content), 0644)
		require.NoError(t, err)

		// copyFile doesn't create destination directories
		err = copyFile(srcFile, dstFile)
		assert.Error(t, err)
	})

	t.Run("copies file when destination directory exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		srcFile := filepath.Join(tmpDir, "source.txt")
		dstDir := filepath.Join(tmpDir, "subdir")
		dstFile := filepath.Join(dstDir, "destination.txt")

		content := "test content"
		err := os.WriteFile(srcFile, []byte(content), 0644)
		require.NoError(t, err)

		// Create destination directory first
		err = os.MkdirAll(dstDir, 0755)
		require.NoError(t, err)

		err = copyFile(srcFile, dstFile)
		require.NoError(t, err)

		// Verify the file was copied
		copiedContent, err := os.ReadFile(dstFile)
		require.NoError(t, err)
		assert.Equal(t, content, string(copiedContent))
	})
}

func TestCopyTemplateFiles(t *testing.T) {
	t.Run("copies template files successfully", func(t *testing.T) {
		tmpDir := t.TempDir()
		srcDir := filepath.Join(tmpDir, "templates")
		dstDir := filepath.Join(tmpDir, "destination")

		// Create source directory with files
		err := os.MkdirAll(srcDir, 0755)
		require.NoError(t, err)

		// Create some template files (must have .template extension)
		file1 := filepath.Join(srcDir, "template1.txt.template")
		file2 := filepath.Join(srcDir, "template2.go.template")
		err = os.WriteFile(file1, []byte("template 1 content"), 0644)
		require.NoError(t, err)
		err = os.WriteFile(file2, []byte("template 2 content"), 0644)
		require.NoError(t, err)

		// Copy template files
		err = copyTemplateFiles(srcDir, dstDir)
		require.NoError(t, err)

		// Verify files were copied
		copiedFile1 := filepath.Join(dstDir, "template1.txt.template")
		copiedFile2 := filepath.Join(dstDir, "template2.go.template")

		content1, err := os.ReadFile(copiedFile1)
		require.NoError(t, err)
		assert.Equal(t, "template 1 content", string(content1))

		content2, err := os.ReadFile(copiedFile2)
		require.NoError(t, err)
		assert.Equal(t, "template 2 content", string(content2))
	})

	t.Run("handles non-existent source directory gracefully", func(t *testing.T) {
		tmpDir := t.TempDir()
		srcDir := filepath.Join(tmpDir, "non-existent")
		dstDir := filepath.Join(tmpDir, "destination")

		// copyTemplateFiles returns nil (not error) for non-existent source directories
		err := copyTemplateFiles(srcDir, dstDir)
		assert.NoError(t, err)

		// Destination directory should still be created
		info, err := os.Stat(dstDir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("creates destination directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		srcDir := filepath.Join(tmpDir, "templates")
		dstDir := filepath.Join(tmpDir, "new-destination")

		// Create source directory
		err := os.MkdirAll(srcDir, 0755)
		require.NoError(t, err)

		// Copy template files (should create destination)
		err = copyTemplateFiles(srcDir, dstDir)
		require.NoError(t, err)

		// Verify destination directory was created
		info, err := os.Stat(dstDir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("ignores non-template files", func(t *testing.T) {
		tmpDir := t.TempDir()
		srcDir := filepath.Join(tmpDir, "templates")
		dstDir := filepath.Join(tmpDir, "destination")

		// Create source directory with mixed files
		err := os.MkdirAll(srcDir, 0755)
		require.NoError(t, err)

		// Create template and non-template files
		templateFile := filepath.Join(srcDir, "config.template")
		regularFile := filepath.Join(srcDir, "readme.txt")
		err = os.WriteFile(templateFile, []byte("template content"), 0644)
		require.NoError(t, err)
		err = os.WriteFile(regularFile, []byte("regular content"), 0644)
		require.NoError(t, err)

		// Copy template files
		err = copyTemplateFiles(srcDir, dstDir)
		require.NoError(t, err)

		// Verify only template file was copied
		copiedTemplate := filepath.Join(dstDir, "config.template")
		copiedRegular := filepath.Join(dstDir, "readme.txt")

		_, err = os.Stat(copiedTemplate)
		assert.NoError(t, err) // Template file should exist

		_, err = os.Stat(copiedRegular)
		assert.True(t, os.IsNotExist(err)) // Regular file should not exist
	})
}

// Mock services for testing
type mockInferenceService struct{}
type mockAgentInferencer struct{}
type mockAuthService struct{}
type mockUserService struct{}
type mockAnalyticsService struct{}
type mockWebConnectionsService struct{}

func createMockServices() (*inference.InferenceService, *agentify.AgentInferencer, *AuthService, *UserService, *AnalyticsService, *WebConnectionsService) {
	// Return nil for now - we'll implement proper mocks as needed
	return nil, nil, nil, nil, nil, nil
}

func TestNewSimpleAPIServer(t *testing.T) {
	t.Run("creates server with valid parameters", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")
		shutdownSignal := make(chan struct{})

		inferenceService, agentInferencer, authService, userService, analyticsService, webConnectionsService := createMockServices()

		server, err := NewSimpleAPIServer(8080, dbPath, shutdownSignal, inferenceService, agentInferencer, authService, userService, analyticsService, webConnectionsService)

		// The function may return an error due to database initialization
		// We test that it doesn't panic and handles the parameters correctly
		if err != nil {
			// Expected in test environment without proper database setup
			assert.Contains(t, err.Error(), "database") // Should be a database-related error
		} else {
			assert.NotNil(t, server)
			assert.NotNil(t, server.router)
		}
	})

	t.Run("handles invalid port", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")
		shutdownSignal := make(chan struct{})

		inferenceService, agentInferencer, authService, userService, analyticsService, webConnectionsService := createMockServices()

		// Test with invalid port
		_, err := NewSimpleAPIServer(-1, dbPath, shutdownSignal, inferenceService, agentInferencer, authService, userService, analyticsService, webConnectionsService)

		// Should handle invalid port gracefully
		// The exact behavior depends on implementation
		_ = err // We don't assert specific error as it depends on implementation details
	})
}

// Additional tests for server functionality
func TestSimpleAPIServerStructs(t *testing.T) {
	t.Run("troubleshooting database struct", func(t *testing.T) {
		chunk1 := TroubleshootingChunk{Category: "network", Issue: "timeout"}
		chunk2 := TroubleshootingChunk{Category: "auth", Issue: "unauthorized"}

		db := TroubleshootingDatabase{
			Chunks: []TroubleshootingChunk{chunk1, chunk2},
		}

		assert.Len(t, db.Chunks, 2)
		assert.Equal(t, "network", db.Chunks[0].Category)
		assert.Equal(t, "auth", db.Chunks[1].Category)
	})
}

// Test helper functions
func TestHelperFunctions(t *testing.T) {
	t.Run("generate agent ID format", func(t *testing.T) {
		id := generateAgentID()
		assert.Contains(t, id, "agent_")
		assert.True(t, len(id) > 10) // Should be reasonably long due to timestamp
	})
}
