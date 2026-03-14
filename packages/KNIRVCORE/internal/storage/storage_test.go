package storage

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewKNIRVBASEStorage(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "testdb")

	storage, err := NewKNIRVBASEStorage(dbPath)
	require.NoError(t, err)
	require.NotNil(t, storage)

	defer storage.Close()
}

func TestPutAndGet(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "testdb")

	storage, err := NewKNIRVBASEStorage(dbPath)
	require.NoError(t, err)
	defer storage.Close()

	key := []byte("test-key")
	value := []byte("test-value")

	// Put
	err = storage.Put(key, value)
	assert.NoError(t, err)

	// Get
	retrieved, err := storage.Get(key)
	assert.NoError(t, err)
	assert.Equal(t, value, retrieved)
}

func TestGetNonExistentKey(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "testdb")

	storage, err := NewKNIRVBASEStorage(dbPath)
	require.NoError(t, err)
	defer storage.Close()

	key := []byte("non-existent")

	// Get non-existent key
	_, err = storage.Get(key)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "key not found")
}

func TestHas(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "testdb")

	storage, err := NewKNIRVBASEStorage(dbPath)
	require.NoError(t, err)
	defer storage.Close()

	key := []byte("test-key")
	value := []byte("test-value")

	// Check before putting
	has, err := storage.Has(key)
	assert.NoError(t, err)
	assert.False(t, has)

	// Put
	err = storage.Put(key, value)
	assert.NoError(t, err)

	// Check after putting
	has, err = storage.Has(key)
	assert.NoError(t, err)
	assert.True(t, has)
}

func TestDelete(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "testdb")

	storage, err := NewKNIRVBASEStorage(dbPath)
	require.NoError(t, err)
	defer storage.Close()

	key := []byte("test-key")
	value := []byte("test-value")

	// Put
	err = storage.Put(key, value)
	assert.NoError(t, err)

	// Verify exists
	has, err := storage.Has(key)
	assert.NoError(t, err)
	assert.True(t, has)

	// Delete
	err = storage.Delete(key)
	assert.NoError(t, err)

	// Verify deleted
	has, err = storage.Has(key)
	assert.NoError(t, err)
	assert.False(t, has)
}

func TestBatch(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "testdb")

	storage, err := NewKNIRVBASEStorage(dbPath)
	require.NoError(t, err)
	defer storage.Close()

	batch := storage.Batch()

	// Add multiple operations to batch
	err = batch.Put([]byte("key1"), []byte("value1"))
	assert.NoError(t, err)

	err = batch.Put([]byte("key2"), []byte("value2"))
	assert.NoError(t, err)

	err = batch.Put([]byte("key3"), []byte("value3"))
	assert.NoError(t, err)

	// Write batch atomically
	err = batch.Write()
	assert.NoError(t, err)

	// Verify all keys exist
	for i := 1; i <= 3; i++ {
		key := []byte("key" + string(rune('0'+i)))
		value := []byte("value" + string(rune('0'+i)))

		retrieved, err := storage.Get(key)
		assert.NoError(t, err)
		assert.Equal(t, value, retrieved)
	}
}

func TestBatchDelete(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "testdb")

	storage, err := NewKNIRVBASEStorage(dbPath)
	require.NoError(t, err)
	defer storage.Close()

	// Put some keys first
	keys := [][]byte{
		[]byte("key1"),
		[]byte("key2"),
		[]byte("key3"),
	}
	for _, key := range keys {
		err = storage.Put(key, []byte("value"))
		assert.NoError(t, err)
	}

	// Batch delete
	batch := storage.Batch()
	for _, key := range keys {
		err = batch.Delete(key)
		assert.NoError(t, err)
	}

	err = batch.Write()
	assert.NoError(t, err)

	// Verify all keys deleted
	for _, key := range keys {
		has, err := storage.Has(key)
		assert.NoError(t, err)
		assert.False(t, has)
	}
}

func TestBatchReset(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "testdb")

	storage, err := NewKNIRVBASEStorage(dbPath)
	require.NoError(t, err)
	defer storage.Close()

	batch := storage.Batch()

	// Add operations
	err = batch.Put([]byte("key1"), []byte("value1"))
	assert.NoError(t, err)

	// Reset batch
	batch.Reset()

	// Add different operation
	err = batch.Put([]byte("key2"), []byte("value2"))
	assert.NoError(t, err)

	// Write
	err = batch.Write()
	assert.NoError(t, err)

	// Verify only key2 exists
	has, err := storage.Has([]byte("key1"))
	assert.NoError(t, err)
	assert.False(t, has)

	has, err = storage.Has([]byte("key2"))
	assert.NoError(t, err)
	assert.True(t, has)
}

func TestIterator(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "testdb")

	storage, err := NewKNIRVBASEStorage(dbPath)
	require.NoError(t, err)
	defer storage.Close()

	// Put test data
	testData := map[string]string{
		"block:001": "block-data-1",
		"block:002": "block-data-2",
		"block:003": "block-data-3",
		"tx:001":    "tx-data-1",
		"tx:002":    "tx-data-2",
	}

	for k, v := range testData {
		err = storage.Put([]byte(k), []byte(v))
		assert.NoError(t, err)
	}

	// Iterate with prefix "block:"
	iter := storage.NewIterator([]byte("block:"))
	defer iter.Release()

	count := 0
	for iter.Next() {
		key := string(iter.Key())
		value := string(iter.Value())

		assert.Contains(t, key, "block:")
		assert.Equal(t, testData[key], value)
		count++
	}

	assert.NoError(t, iter.Error())
	assert.Equal(t, 3, count)
}

func TestIteratorNoPrefix(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "testdb")

	storage, err := NewKNIRVBASEStorage(dbPath)
	require.NoError(t, err)
	defer storage.Close()

	// Put test data
	testData := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	for k, v := range testData {
		err = storage.Put([]byte(k), []byte(v))
		assert.NoError(t, err)
	}

	// Iterate without prefix
	iter := storage.NewIterator(nil)
	defer iter.Release()

	count := 0
	for iter.Next() {
		count++
	}

	assert.NoError(t, iter.Error())
	assert.Equal(t, len(testData), count)
}

func TestConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "testdb")

	storage, err := NewKNIRVBASEStorage(dbPath)
	require.NoError(t, err)
	defer storage.Close()

	// Concurrent writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(n int) {
			key := []byte("key-" + string(rune('0'+n)))
			value := []byte("value-" + string(rune('0'+n)))
			err := storage.Put(key, value)
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all keys exist
	for i := 0; i < 10; i++ {
		key := []byte("key-" + string(rune('0'+i)))
		has, err := storage.Has(key)
		assert.NoError(t, err)
		assert.True(t, has)
	}
}

func TestPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "testdb")

	// Create storage and write data
	storage1, err := NewKNIRVBASEStorage(dbPath)
	require.NoError(t, err)

	key := []byte("persist-key")
	value := []byte("persist-value")

	err = storage1.Put(key, value)
	assert.NoError(t, err)

	// Close storage
	err = storage1.Close()
	assert.NoError(t, err)

	// Reopen storage
	storage2, err := NewKNIRVBASEStorage(dbPath)
	require.NoError(t, err)
	defer storage2.Close()

	// Verify data persisted
	retrieved, err := storage2.Get(key)
	assert.NoError(t, err)
	assert.Equal(t, value, retrieved)
}

func TestClose(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "testdb")

	storage, err := NewKNIRVBASEStorage(dbPath)
	require.NoError(t, err)

	err = storage.Close()
	assert.NoError(t, err)

	// Attempting operations after close should fail
	// (LevelDB will panic or error on closed DB)
	// We just verify Close doesn't error
}

func BenchmarkPut(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "benchdb")

	storage, err := NewKNIRVBASEStorage(dbPath)
	require.NoError(b, err)
	defer storage.Close()

	key := []byte("benchmark-key")
	value := []byte("benchmark-value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = storage.Put(key, value)
	}
}

func BenchmarkGet(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "benchdb")

	storage, err := NewKNIRVBASEStorage(dbPath)
	require.NoError(b, err)
	defer storage.Close()

	key := []byte("benchmark-key")
	value := []byte("benchmark-value")

	err = storage.Put(key, value)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = storage.Get(key)
	}
}

func BenchmarkBatchWrite(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "benchdb")

	storage, err := NewKNIRVBASEStorage(dbPath)
	require.NoError(b, err)
	defer storage.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		batch := storage.Batch()
		for j := 0; j < 100; j++ {
			key := []byte("key-" + string(rune('0'+j)))
			value := []byte("value-" + string(rune('0'+j)))
			_ = batch.Put(key, value)
		}
		_ = batch.Write()
	}
}
