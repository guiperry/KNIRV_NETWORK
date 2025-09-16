package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileManager(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "file-manager-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create file manager
	fileManager, err := NewFileManager(tempDir)
	require.NoError(t, err)

	// Create test files
	testJsonFile := filepath.Join(tempDir, "test.json")
	err = os.WriteFile(testJsonFile, []byte(`{"name":"test","version":"1.0.0","description":"Test"}`), 0644)
	require.NoError(t, err)

	testTextFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testTextFile, []byte("test"), 0644)
	require.NoError(t, err)

	t.Run("ValidateManifestFile", func(t *testing.T) {
		// Valid JSON manifest
		err := fileManager.ValidateManifestFile(testJsonFile)
		assert.NoError(t, err)

		// Invalid extension
		err = fileManager.ValidateManifestFile(testTextFile)
		assert.Error(t, err)

		// Non-existent file
		err = fileManager.ValidateManifestFile(filepath.Join(tempDir, "nonexistent.json"))
		assert.Error(t, err)

		// Create invalid JSON
		invalidJsonFile := filepath.Join(tempDir, "invalid.json")
		err = os.WriteFile(invalidJsonFile, []byte(`{"name":"test",`), 0644)
		require.NoError(t, err)

		err = fileManager.ValidateManifestFile(invalidJsonFile)
		assert.Error(t, err)

		// Create JSON with missing fields
		missingFieldsFile := filepath.Join(tempDir, "missing.json")
		err = os.WriteFile(missingFieldsFile, []byte(`{"name":"test"}`), 0644)
		require.NoError(t, err)

		err = fileManager.ValidateManifestFile(missingFieldsFile)
		assert.Error(t, err)
	})

	t.Run("GenerateFileReference", func(t *testing.T) {
		// Generate reference for test file
		ref, err := fileManager.GenerateFileReference(testJsonFile)
		require.NoError(t, err)

		// Check reference fields
		assert.Equal(t, filepath.Base(testJsonFile), filepath.Base(ref.RelativePath))
		assert.NotEmpty(t, ref.ContentHash)
		assert.Equal(t, int64(47), ref.FileSize) // Size of the test JSON
		assert.NotEmpty(t, ref.LocationHint)
	})

	t.Run("GetRelativePath", func(t *testing.T) {
		// Create subdirectory
		subDir := filepath.Join(tempDir, "subdir")
		err := os.MkdirAll(subDir, 0755)
		require.NoError(t, err)

		// Create file in subdirectory
		subFile := filepath.Join(subDir, "subfile.txt")
		err = os.WriteFile(subFile, []byte("test"), 0644)
		require.NoError(t, err)

		// Get relative path
		relPath, err := fileManager.GetRelativePath(subFile)
		require.NoError(t, err)
		assert.Equal(t, "subdir/subfile.txt", relPath)
	})

	t.Run("CopyFileToBaseDir", func(t *testing.T) {
		// Create source file outside base directory
		sourceDir, err := os.MkdirTemp("", "source-dir")
		require.NoError(t, err)
		defer os.RemoveAll(sourceDir)

		sourceFile := filepath.Join(sourceDir, "source.txt")
		err = os.WriteFile(sourceFile, []byte("test content"), 0644)
		require.NoError(t, err)

		// Copy file to base directory
		destPath, err := fileManager.CopyFileToBaseDir(sourceFile)
		require.NoError(t, err)

		// Check destination file exists
		_, err = os.Stat(destPath)
		assert.NoError(t, err)

		// Check file contents
		content, err := os.ReadFile(destPath)
		require.NoError(t, err)
		assert.Equal(t, "test content", string(content))
	})

	t.Run("calculateContentHash", func(t *testing.T) {
		// Calculate hash for test file
		hash1, err := calculateContentHash(testJsonFile)
		require.NoError(t, err)
		assert.NotEmpty(t, hash1)

		// Calculate hash again to verify it's deterministic
		hash2, err := calculateContentHash(testJsonFile)
		require.NoError(t, err)
		assert.Equal(t, hash1, hash2)

		// Modify file and check hash changes
		err = os.WriteFile(testJsonFile, []byte(`{"name":"modified","version":"1.0.0","description":"Test"}`), 0644)
		require.NoError(t, err)

		hash3, err := calculateContentHash(testJsonFile)
		require.NoError(t, err)
		assert.NotEqual(t, hash1, hash3)
	})
}
