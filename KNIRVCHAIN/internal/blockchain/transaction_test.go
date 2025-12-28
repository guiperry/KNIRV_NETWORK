package blockchain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTransaction(t *testing.T) {
	tx := NewTransaction(TxTypeStore, "user1", 100)

	assert.NotEqual(t, uuid.Nil, tx.TxID)
	assert.Equal(t, TxTypeStore, tx.Type)
	assert.Equal(t, "user1", tx.From)
	assert.Equal(t, uint64(100), tx.NRNCost)
	assert.Equal(t, TxStatusPending, tx.Status)
	assert.NotZero(t, tx.CreatedAt)
	assert.NotNil(t, tx.Metadata)
}

func TestNewStoreTransaction(t *testing.T) {
	data := []byte("test data")
	tx := NewStoreTransaction("user1", data, 50)

	assert.Equal(t, TxTypeStore, tx.Type)
	assert.Equal(t, data, tx.Data)
	assert.Equal(t, uint64(50), tx.NRNCost)
}

func TestNewRetrieveTransaction(t *testing.T) {
	blockID := uuid.New()
	tx := NewRetrieveTransaction("user1", blockID)

	assert.Equal(t, TxTypeRetrieve, tx.Type)
	assert.NotNil(t, tx.BlockID)
	assert.Equal(t, blockID, *tx.BlockID)
	assert.Equal(t, uint64(0), tx.NRNCost) // Retrieval is free
}

func TestNewUpdateTransaction(t *testing.T) {
	blockID := uuid.New()
	data := []byte("updated data")
	tx := NewUpdateTransaction("user1", blockID, data, 30)

	assert.Equal(t, TxTypeUpdate, tx.Type)
	assert.NotNil(t, tx.BlockID)
	assert.Equal(t, blockID, *tx.BlockID)
	assert.Equal(t, data, tx.Data)
	assert.Equal(t, uint64(30), tx.NRNCost)
}

func TestNewDeleteTransaction(t *testing.T) {
	blockID := uuid.New()
	tx := NewDeleteTransaction("user1", blockID)

	assert.Equal(t, TxTypeDelete, tx.Type)
	assert.NotNil(t, tx.BlockID)
	assert.Equal(t, blockID, *tx.BlockID)
	assert.Equal(t, uint64(0), tx.NRNCost) // Deletion is free
}

func TestTransaction_Validate(t *testing.T) {
	// Test valid STORE transaction
	t.Run("valid store", func(t *testing.T) {
		tx := NewStoreTransaction("user1", []byte("data"), 100)
		err := tx.Validate()
		assert.NoError(t, err)
	})

	// Test invalid transaction ID
	t.Run("nil transaction ID", func(t *testing.T) {
		tx := NewTransaction(TxTypeStore, "user1", 100)
		tx.TxID = uuid.Nil
		err := tx.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "transaction ID cannot be nil")
	})

	// Test invalid type
	t.Run("invalid type", func(t *testing.T) {
		tx := NewTransaction("INVALID", "user1", 100)
		err := tx.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid transaction type")
	})

	// Test empty from
	t.Run("empty from", func(t *testing.T) {
		tx := NewTransaction(TxTypeStore, "", 100)
		err := tx.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "from address cannot be empty")
	})

	// Test STORE without data
	t.Run("store without data", func(t *testing.T) {
		tx := NewStoreTransaction("user1", []byte{}, 100)
		err := tx.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must have data")
	})

	// Test STORE without cost
	t.Run("store without cost", func(t *testing.T) {
		tx := NewStoreTransaction("user1", []byte("data"), 0)
		err := tx.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must have non-zero cost")
	})

	// Test RETRIEVE without block ID
	t.Run("retrieve without block ID", func(t *testing.T) {
		tx := NewTransaction(TxTypeRetrieve, "user1", 0)
		err := tx.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must specify block ID")
	})

	// Test UPDATE without block ID
	t.Run("update without block ID", func(t *testing.T) {
		tx := NewTransaction(TxTypeUpdate, "user1", 10)
		tx.Data = []byte("data")
		err := tx.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must specify block ID")
	})

	// Test UPDATE without data
	t.Run("update without data", func(t *testing.T) {
		blockID := uuid.New()
		tx := NewUpdateTransaction("user1", blockID, []byte{}, 10)
		err := tx.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must have data")
	})

	// Test DELETE without block ID
	t.Run("delete without block ID", func(t *testing.T) {
		tx := NewTransaction(TxTypeDelete, "user1", 0)
		err := tx.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must specify block ID")
	})
}

func TestTransaction_StatusMethods(t *testing.T) {
	tx := NewStoreTransaction("user1", []byte("data"), 100)

	// Test pending
	assert.True(t, tx.IsPending())
	assert.False(t, tx.IsProcessing())
	assert.False(t, tx.IsConfirmed())
	assert.False(t, tx.IsFailed())
	assert.False(t, tx.IsRejected())
	assert.False(t, tx.IsFinalized())

	// Mark processing
	tx.MarkProcessing()
	assert.False(t, tx.IsPending())
	assert.True(t, tx.IsProcessing())
	assert.False(t, tx.IsFinalized())

	// Mark confirmed
	blockID := uuid.New()
	tx.MarkConfirmed(&blockID)
	assert.False(t, tx.IsPending())
	assert.False(t, tx.IsProcessing())
	assert.True(t, tx.IsConfirmed())
	assert.True(t, tx.IsFinalized())
	assert.NotNil(t, tx.ProcessedAt)
	assert.Equal(t, blockID, *tx.ResultBlockID)
}

func TestTransaction_MarkFailed(t *testing.T) {
	tx := NewStoreTransaction("user1", []byte("data"), 100)

	tx.MarkFailed(assert.AnError)
	assert.True(t, tx.IsFailed())
	assert.True(t, tx.IsFinalized())
	assert.NotNil(t, tx.ProcessedAt)
	assert.NotEmpty(t, tx.Error)
}

func TestTransaction_MarkRejected(t *testing.T) {
	tx := NewStoreTransaction("user1", []byte("data"), 100)

	tx.MarkRejected("validation failed")
	assert.True(t, tx.IsRejected())
	assert.True(t, tx.IsFinalized())
	assert.NotNil(t, tx.ProcessedAt)
	assert.Equal(t, "validation failed", tx.Error)
}

func TestTransaction_Age(t *testing.T) {
	tx := NewStoreTransaction("user1", []byte("data"), 100)

	time.Sleep(10 * time.Millisecond)

	age := tx.Age()
	assert.Greater(t, age, 10*time.Millisecond)
}

func TestTransaction_ProcessingTime(t *testing.T) {
	tx := NewStoreTransaction("user1", []byte("data"), 100)

	// No processing time when pending
	assert.Equal(t, time.Duration(0), tx.ProcessingTime())

	time.Sleep(10 * time.Millisecond)
	tx.MarkConfirmed(nil)

	// Should have processing time after confirmed
	processingTime := tx.ProcessingTime()
	assert.Greater(t, processingTime, 10*time.Millisecond)
}

func TestTransaction_Metadata(t *testing.T) {
	tx := NewStoreTransaction("user1", []byte("data"), 100)

	// Set metadata
	tx.SetMetadata("category", "personal")
	tx.SetMetadata("priority", "high")

	// Get metadata
	category, ok := tx.GetMetadata("category")
	assert.True(t, ok)
	assert.Equal(t, "personal", category)

	priority, ok := tx.GetMetadata("priority")
	assert.True(t, ok)
	assert.Equal(t, "high", priority)

	// Get non-existent key
	_, ok = tx.GetMetadata("nonexistent")
	assert.False(t, ok)
}

func TestTransaction_GetReceipt(t *testing.T) {
	tx := NewStoreTransaction("user1", []byte("data"), 100)
	blockID := uuid.New()
	tx.MarkConfirmed(&blockID)

	receipt := tx.GetReceipt()
	assert.NotNil(t, receipt)
	assert.Equal(t, tx.TxID, receipt.TxID)
	assert.Equal(t, tx.Type, receipt.Type)
	assert.Equal(t, tx.Status, receipt.Status)
	assert.Equal(t, blockID, *receipt.ResultBlockID)
	assert.Equal(t, tx.NRNCost, receipt.NRNCost)
	assert.NotZero(t, receipt.ProcessedAt)
}

func TestTransaction_GetReceiptFailed(t *testing.T) {
	tx := NewStoreTransaction("user1", []byte("data"), 100)
	tx.MarkFailed(assert.AnError)

	receipt := tx.GetReceipt()
	assert.NotNil(t, receipt)
	assert.Equal(t, TxStatusFailed, receipt.Status)
	assert.NotEmpty(t, receipt.Error)
}

func TestIsValidTransactionType(t *testing.T) {
	assert.True(t, isValidTransactionType(TxTypeStore))
	assert.True(t, isValidTransactionType(TxTypeRetrieve))
	assert.True(t, isValidTransactionType(TxTypeUpdate))
	assert.True(t, isValidTransactionType(TxTypeDelete))
	assert.False(t, isValidTransactionType("INVALID"))
}

func TestIsValidTransactionStatus(t *testing.T) {
	assert.True(t, isValidTransactionStatus(TxStatusPending))
	assert.True(t, isValidTransactionStatus(TxStatusProcessing))
	assert.True(t, isValidTransactionStatus(TxStatusConfirmed))
	assert.True(t, isValidTransactionStatus(TxStatusFailed))
	assert.True(t, isValidTransactionStatus(TxStatusRejected))
	assert.False(t, isValidTransactionStatus("INVALID"))
}

func TestTransactionLifecycle(t *testing.T) {
	// Create transaction
	tx := NewStoreTransaction("user1", []byte("test memory data"), 100)
	require.NoError(t, tx.Validate())

	// Verify initial state
	assert.True(t, tx.IsPending())
	assert.Nil(t, tx.ProcessedAt)

	// Start processing
	tx.MarkProcessing()
	assert.True(t, tx.IsProcessing())

	// Simulate processing delay
	time.Sleep(10 * time.Millisecond)

	// Complete successfully
	resultBlockID := uuid.New()
	tx.MarkConfirmed(&resultBlockID)

	// Verify final state
	assert.True(t, tx.IsConfirmed())
	assert.True(t, tx.IsFinalized())
	assert.NotNil(t, tx.ProcessedAt)
	assert.Equal(t, resultBlockID, *tx.ResultBlockID)
	assert.Greater(t, tx.ProcessingTime(), 10*time.Millisecond)

	// Verify receipt
	receipt := tx.GetReceipt()
	assert.Equal(t, tx.TxID, receipt.TxID)
	assert.Equal(t, TxStatusConfirmed, receipt.Status)
}
