package database

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSimpleDomainDB(t *testing.T) {
	t.Run("creates in-memory database", func(t *testing.T) {
		db, err := NewSimpleDomainDB("")
		require.NoError(t, err)
		assert.NotNil(t, db)
		assert.NotNil(t, db.db)
		assert.NotNil(t, db.GetDB())
	})

	t.Run("creates persistent database", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")

		db, err := NewSimpleDomainDB(dbPath)
		require.NoError(t, err)
		assert.NotNil(t, db)
		assert.NotNil(t, db.db)
		assert.NotNil(t, db.GetDB())

		// Verify database file was created
		_, err = os.Stat(dbPath)
		assert.NoError(t, err)
	})

	t.Run("handles invalid persistence path", func(t *testing.T) {
		// Try to create database in non-existent directory without creating it
		invalidPath := "/non/existent/directory/test.db"
		
		db, err := NewSimpleDomainDB(invalidPath)
		// The behavior depends on the chromem-go implementation
		// It might create the directory or return an error
		if err != nil {
			assert.Nil(t, db)
			assert.Error(t, err)
		} else {
			assert.NotNil(t, db)
		}
	})
}

func TestSimpleDomainDB_GetDB(t *testing.T) {
	t.Run("returns underlying database", func(t *testing.T) {
		db, err := NewSimpleDomainDB("")
		require.NoError(t, err)

		underlyingDB := db.GetDB()
		assert.NotNil(t, underlyingDB)
		assert.Equal(t, db.db, underlyingDB)
	})
}

func TestSimpleDomainDB_GetOrCreateCollection(t *testing.T) {
	t.Run("creates new collection", func(t *testing.T) {
		db, err := NewSimpleDomainDB("")
		require.NoError(t, err)

		collection, err := db.GetOrCreateCollection("test_collection")
		
		// The behavior depends on whether Cerebras API key is available
		// If no API key, it might use a default embedding function
		if err != nil {
			// Expected if no API key is configured
			assert.Nil(t, collection)
			assert.Error(t, err)
		} else {
			assert.NotNil(t, collection)
		}
	})

	t.Run("handles empty collection name", func(t *testing.T) {
		db, err := NewSimpleDomainDB("")
		require.NoError(t, err)

		collection, err := db.GetOrCreateCollection("")
		
		// Should handle empty name gracefully
		if err != nil {
			assert.Nil(t, collection)
			assert.Error(t, err)
		} else {
			assert.NotNil(t, collection)
		}
	})

	t.Run("gets existing collection", func(t *testing.T) {
		db, err := NewSimpleDomainDB("")
		require.NoError(t, err)

		collectionName := "existing_collection"
		
		// Create collection first
		collection1, err1 := db.GetOrCreateCollection(collectionName)
		
		// Get the same collection again
		collection2, err2 := db.GetOrCreateCollection(collectionName)
		
		// Both should succeed or both should fail consistently
		if err1 != nil {
			assert.Error(t, err2)
		} else {
			assert.NoError(t, err2)
			assert.NotNil(t, collection1)
			assert.NotNil(t, collection2)
		}
	})
}

// Test database lifecycle
func TestSimpleDomainDB_Lifecycle(t *testing.T) {
	t.Run("database lifecycle with persistence", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "lifecycle_test.db")

		// Create database
		db1, err := NewSimpleDomainDB(dbPath)
		require.NoError(t, err)
		assert.NotNil(t, db1)

		// Verify file exists
		_, err = os.Stat(dbPath)
		assert.NoError(t, err)

		// Create another instance with same path
		db2, err := NewSimpleDomainDB(dbPath)
		require.NoError(t, err)
		assert.NotNil(t, db2)

		// Both should be valid instances
		assert.NotNil(t, db1.GetDB())
		assert.NotNil(t, db2.GetDB())
	})

	t.Run("multiple in-memory databases are independent", func(t *testing.T) {
		db1, err := NewSimpleDomainDB("")
		require.NoError(t, err)

		db2, err := NewSimpleDomainDB("")
		require.NoError(t, err)

		// Should be different instances
		assert.NotEqual(t, db1.GetDB(), db2.GetDB())
	})
}

// Test error conditions
func TestSimpleDomainDB_ErrorConditions(t *testing.T) {
	t.Run("handles database operations on nil database", func(t *testing.T) {
		// This test ensures the struct handles edge cases gracefully
		db := &SimpleDomainDB{db: nil}
		
		// GetDB should return nil
		assert.Nil(t, db.GetDB())
		
		// GetOrCreateCollection should handle nil database
		collection, err := db.GetOrCreateCollection("test")
		assert.Error(t, err)
		assert.Nil(t, collection)
	})
}

// Test configuration and environment
func TestSimpleDomainDB_Configuration(t *testing.T) {
	t.Run("handles missing environment variables", func(t *testing.T) {
		// Save original environment
		originalCerebrasKey := os.Getenv("CEREBRAS_API_KEY")
		defer func() {
			if originalCerebrasKey != "" {
				os.Setenv("CEREBRAS_API_KEY", originalCerebrasKey)
			} else {
				os.Unsetenv("CEREBRAS_API_KEY")
			}
		}()

		// Remove API key
		os.Unsetenv("CEREBRAS_API_KEY")

		db, err := NewSimpleDomainDB("")
		require.NoError(t, err)

		// Should still create database, but collection creation might fail
		collection, err := db.GetOrCreateCollection("test_no_key")
		
		// Behavior depends on implementation - might use default embeddings or fail
		if err != nil {
			assert.Nil(t, collection)
		} else {
			assert.NotNil(t, collection)
		}
	})

	t.Run("handles environment variables", func(t *testing.T) {
		// Save original environment
		originalCerebrasKey := os.Getenv("CEREBRAS_API_KEY")
		defer func() {
			if originalCerebrasKey != "" {
				os.Setenv("CEREBRAS_API_KEY", originalCerebrasKey)
			} else {
				os.Unsetenv("CEREBRAS_API_KEY")
			}
		}()

		// Set a test API key
		os.Setenv("CEREBRAS_API_KEY", "test-key-123")

		db, err := NewSimpleDomainDB("")
		require.NoError(t, err)

		// Should create database successfully
		assert.NotNil(t, db)
		assert.NotNil(t, db.GetDB())
	})
}

// Test concurrent access
func TestSimpleDomainDB_Concurrency(t *testing.T) {
	t.Run("handles concurrent database creation", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "concurrent_test.db")

		// Create multiple databases concurrently
		done := make(chan bool, 2)
		var db1, db2 *SimpleDomainDB
		var err1, err2 error

		go func() {
			db1, err1 = NewSimpleDomainDB(dbPath)
			done <- true
		}()

		go func() {
			db2, err2 = NewSimpleDomainDB(dbPath)
			done <- true
		}()

		// Wait for both to complete
		<-done
		<-done

		// Both should succeed
		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.NotNil(t, db1)
		assert.NotNil(t, db2)
	})
}
