package blockchain

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMempool(t *testing.T) {
	config := MempoolConfig{
		MaxSize:  100,
		MaxTxAge: 1 * time.Hour,
	}

	mp := NewMempool(config)
	assert.NotNil(t, mp)
	assert.Equal(t, 100, mp.maxSize)
	assert.Equal(t, 1*time.Hour, mp.maxTxAge)
	assert.Equal(t, 0, mp.Size())
}

func TestNewDefaultMempool(t *testing.T) {
	mp := NewDefaultMempool()
	assert.NotNil(t, mp)
	assert.Equal(t, 10000, mp.maxSize)
	assert.Equal(t, 24*time.Hour, mp.maxTxAge)
}

func TestMempool_Add(t *testing.T) {
	mp := NewDefaultMempool()

	tx := NewStoreTransaction("user1", []byte("data"), 100)

	err := mp.Add(tx)
	require.NoError(t, err)
	assert.Equal(t, 1, mp.Size())
}

func TestMempool_AddInvalid(t *testing.T) {
	mp := NewDefaultMempool()

	// Create invalid transaction (no data for STORE)
	tx := NewStoreTransaction("user1", []byte{}, 100)

	err := mp.Add(tx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid transaction")
	assert.Equal(t, 0, mp.Size())
}

func TestMempool_AddDuplicate(t *testing.T) {
	mp := NewDefaultMempool()

	tx := NewStoreTransaction("user1", []byte("data"), 100)

	err := mp.Add(tx)
	require.NoError(t, err)

	// Try to add same transaction again
	err = mp.Add(tx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already in mempool")
	assert.Equal(t, 1, mp.Size())
}

func TestMempool_AddNonPending(t *testing.T) {
	mp := NewDefaultMempool()

	tx := NewStoreTransaction("user1", []byte("data"), 100)
	tx.MarkProcessing()

	err := mp.Add(tx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "only pending transactions")
}

func TestMempool_AddFull(t *testing.T) {
	config := MempoolConfig{
		MaxSize:  2,
		MaxTxAge: 1 * time.Hour,
	}
	mp := NewMempool(config)

	// Add 2 transactions
	tx1 := NewStoreTransaction("user1", []byte("data1"), 100)
	tx2 := NewStoreTransaction("user2", []byte("data2"), 100)

	require.NoError(t, mp.Add(tx1))
	require.NoError(t, mp.Add(tx2))
	assert.Equal(t, 2, mp.Size())

	// Try to add third
	tx3 := NewStoreTransaction("user3", []byte("data3"), 100)
	err := mp.Add(tx3)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mempool is full")
	assert.Equal(t, 2, mp.Size())
}

func TestMempool_GetAndRemove(t *testing.T) {
	mp := NewDefaultMempool()

	tx := NewStoreTransaction("user1", []byte("data"), 100)
	require.NoError(t, mp.Add(tx))

	// Get transaction
	retrieved, exists := mp.Get(tx.TxID)
	assert.True(t, exists)
	assert.Equal(t, tx.TxID, retrieved.TxID)

	// Remove transaction
	mp.Remove(tx.TxID)
	assert.Equal(t, 0, mp.Size())

	// Try to get removed transaction
	_, exists = mp.Get(tx.TxID)
	assert.False(t, exists)
}

func TestMempool_RemoveBatch(t *testing.T) {
	mp := NewDefaultMempool()

	tx1 := NewStoreTransaction("user1", []byte("data1"), 100)
	tx2 := NewStoreTransaction("user2", []byte("data2"), 100)
	tx3 := NewStoreTransaction("user3", []byte("data3"), 100)

	require.NoError(t, mp.Add(tx1))
	require.NoError(t, mp.Add(tx2))
	require.NoError(t, mp.Add(tx3))
	assert.Equal(t, 3, mp.Size())

	// Remove batch
	mp.RemoveBatch([]uuid.UUID{tx1.TxID, tx3.TxID})
	assert.Equal(t, 1, mp.Size())

	// Verify only tx2 remains
	_, exists := mp.Get(tx2.TxID)
	assert.True(t, exists)
}

func TestMempool_GetPending(t *testing.T) {
	mp := NewDefaultMempool()

	// Add 3 pending transactions
	tx1 := NewStoreTransaction("user1", []byte("data1"), 100)
	tx2 := NewStoreTransaction("user2", []byte("data2"), 100)
	tx3 := NewStoreTransaction("user3", []byte("data3"), 100)

	require.NoError(t, mp.Add(tx1))
	require.NoError(t, mp.Add(tx2))
	require.NoError(t, mp.Add(tx3))

	// Mark one as processing
	mp.UpdateStatus(tx2.TxID, TxStatusProcessing)

	// Get pending should return 2
	pending := mp.GetPending(10)
	assert.Len(t, pending, 2)
}

func TestMempool_GetOldest(t *testing.T) {
	mp := NewDefaultMempool()

	// Add transactions with delays
	tx1 := NewStoreTransaction("user1", []byte("data1"), 100)
	require.NoError(t, mp.Add(tx1))

	time.Sleep(10 * time.Millisecond)

	tx2 := NewStoreTransaction("user2", []byte("data2"), 100)
	require.NoError(t, mp.Add(tx2))

	time.Sleep(10 * time.Millisecond)

	tx3 := NewStoreTransaction("user3", []byte("data3"), 100)
	require.NoError(t, mp.Add(tx3))

	// Get oldest
	oldest := mp.GetOldest(2)
	assert.Len(t, oldest, 2)
	assert.Equal(t, tx1.TxID, oldest[0].TxID)
	assert.Equal(t, tx2.TxID, oldest[1].TxID)
}

func TestMempool_GetByType(t *testing.T) {
	mp := NewDefaultMempool()

	tx1 := NewStoreTransaction("user1", []byte("data1"), 100)
	blockID := uuid.New()
	tx2 := NewRetrieveTransaction("user2", blockID)
	tx3 := NewStoreTransaction("user3", []byte("data3"), 100)

	require.NoError(t, mp.Add(tx1))
	require.NoError(t, mp.Add(tx2))
	require.NoError(t, mp.Add(tx3))

	// Get STORE transactions
	storeTxs := mp.GetByType(TxTypeStore, 10)
	assert.Len(t, storeTxs, 2)

	// Get RETRIEVE transactions
	retrieveTxs := mp.GetByType(TxTypeRetrieve, 10)
	assert.Len(t, retrieveTxs, 1)
	assert.Equal(t, tx2.TxID, retrieveTxs[0].TxID)
}

func TestMempool_Clear(t *testing.T) {
	mp := NewDefaultMempool()

	tx1 := NewStoreTransaction("user1", []byte("data1"), 100)
	tx2 := NewStoreTransaction("user2", []byte("data2"), 100)

	require.NoError(t, mp.Add(tx1))
	require.NoError(t, mp.Add(tx2))
	assert.Equal(t, 2, mp.Size())

	mp.Clear()
	assert.Equal(t, 0, mp.Size())
}

func TestMempool_PurgeExpired(t *testing.T) {
	config := MempoolConfig{
		MaxSize:  100,
		MaxTxAge: 50 * time.Millisecond, // Very short for testing
	}
	mp := NewMempool(config)

	// Add transaction
	tx := NewStoreTransaction("user1", []byte("data"), 100)
	require.NoError(t, mp.Add(tx))

	// Wait for expiration
	time.Sleep(60 * time.Millisecond)

	// Purge expired
	purged := mp.PurgeExpired()
	assert.Equal(t, 1, purged)
	assert.Equal(t, 0, mp.Size())
}

func TestMempool_PurgeByStatus(t *testing.T) {
	mp := NewDefaultMempool()

	tx1 := NewStoreTransaction("user1", []byte("data1"), 100)
	tx2 := NewStoreTransaction("user2", []byte("data2"), 100)
	tx3 := NewStoreTransaction("user3", []byte("data3"), 100)

	require.NoError(t, mp.Add(tx1))
	require.NoError(t, mp.Add(tx2))
	require.NoError(t, mp.Add(tx3))

	// Mark some as processing
	mp.UpdateStatus(tx1.TxID, TxStatusProcessing)
	mp.UpdateStatus(tx2.TxID, TxStatusProcessing)

	// Purge processing
	purged := mp.PurgeByStatus(TxStatusProcessing)
	assert.Equal(t, 2, purged)
	assert.Equal(t, 1, mp.Size())
}

func TestMempool_UpdateStatus(t *testing.T) {
	mp := NewDefaultMempool()

	tx := NewStoreTransaction("user1", []byte("data"), 100)
	require.NoError(t, mp.Add(tx))

	err := mp.UpdateStatus(tx.TxID, TxStatusProcessing)
	require.NoError(t, err)

	retrieved, _ := mp.Get(tx.TxID)
	assert.Equal(t, TxStatusProcessing, retrieved.Status)

	// Try to update non-existent transaction
	err = mp.UpdateStatus(uuid.New(), TxStatusConfirmed)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestMempool_GetStats(t *testing.T) {
	mp := NewDefaultMempool()

	tx1 := NewStoreTransaction("user1", []byte("data1"), 100)
	blockID := uuid.New()
	tx2 := NewRetrieveTransaction("user2", blockID)
	tx3 := NewStoreTransaction("user3", []byte("data3"), 100)

	require.NoError(t, mp.Add(tx1))
	require.NoError(t, mp.Add(tx2))
	require.NoError(t, mp.Add(tx3))

	// Add one that will be rejected
	invalidTx := NewStoreTransaction("user4", []byte{}, 100)
	mp.Add(invalidTx) // Will be rejected

	stats := mp.GetStats()
	assert.Equal(t, 3, stats.Size)
	assert.Equal(t, 10000, stats.MaxSize)
	assert.Equal(t, uint64(3), stats.TotalAdded)
	assert.Equal(t, uint64(1), stats.TotalRejected)
	assert.Equal(t, 2, stats.ByType[TxTypeStore])
	assert.Equal(t, 1, stats.ByType[TxTypeRetrieve])
	assert.Equal(t, 3, stats.ByStatus[TxStatusPending])
}

func TestMempool_StartPurgeLoop(t *testing.T) {
	config := MempoolConfig{
		MaxSize:  100,
		MaxTxAge: 50 * time.Millisecond,
	}
	mp := NewMempool(config)

	// Add transaction
	tx := NewStoreTransaction("user1", []byte("data"), 100)
	require.NoError(t, mp.Add(tx))

	// Start purge loop
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go mp.StartPurgeLoop(ctx, 30*time.Millisecond)

	// Wait for expiration and purge
	time.Sleep(100 * time.Millisecond)

	// Transaction should be purged
	assert.Equal(t, 0, mp.Size())
}

func TestMempool_WaitForTransaction(t *testing.T) {
	mp := NewDefaultMempool()

	tx := NewStoreTransaction("user1", []byte("data"), 100)
	require.NoError(t, mp.Add(tx))

	ctx := context.Background()

	// Start goroutine to mark transaction as confirmed after delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		mp.UpdateStatus(tx.TxID, TxStatusConfirmed)
	}()

	// Wait for transaction
	result, err := mp.WaitForTransaction(ctx, tx.TxID, 200*time.Millisecond)
	require.NoError(t, err)
	assert.Equal(t, TxStatusConfirmed, result.Status)
}

func TestMempool_WaitForTransactionTimeout(t *testing.T) {
	mp := NewDefaultMempool()

	tx := NewStoreTransaction("user1", []byte("data"), 100)
	require.NoError(t, mp.Add(tx))

	ctx := context.Background()

	// Wait for transaction with short timeout
	_, err := mp.WaitForTransaction(ctx, tx.TxID, 50*time.Millisecond)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
}

func TestMempool_GetTransactionsByFrom(t *testing.T) {
	mp := NewDefaultMempool()

	tx1 := NewStoreTransaction("user1", []byte("data1"), 100)
	tx2 := NewStoreTransaction("user2", []byte("data2"), 100)
	tx3 := NewStoreTransaction("user1", []byte("data3"), 100)

	require.NoError(t, mp.Add(tx1))
	require.NoError(t, mp.Add(tx2))
	require.NoError(t, mp.Add(tx3))

	// Get transactions from user1
	user1Txs := mp.GetTransactionsByFrom("user1", 10)
	assert.Len(t, user1Txs, 2)

	// Get transactions from user2
	user2Txs := mp.GetTransactionsByFrom("user2", 10)
	assert.Len(t, user2Txs, 1)
}

func TestMempool_HasTransaction(t *testing.T) {
	mp := NewDefaultMempool()

	tx := NewStoreTransaction("user1", []byte("data"), 100)
	require.NoError(t, mp.Add(tx))

	assert.True(t, mp.HasTransaction(tx.TxID))
	assert.False(t, mp.HasTransaction(uuid.New()))
}

func TestMempool_IsFull(t *testing.T) {
	config := MempoolConfig{
		MaxSize:  2,
		MaxTxAge: 1 * time.Hour,
	}
	mp := NewMempool(config)

	assert.False(t, mp.IsFull())

	tx1 := NewStoreTransaction("user1", []byte("data1"), 100)
	tx2 := NewStoreTransaction("user2", []byte("data2"), 100)

	require.NoError(t, mp.Add(tx1))
	assert.False(t, mp.IsFull())

	require.NoError(t, mp.Add(tx2))
	assert.True(t, mp.IsFull())
}

func TestMempool_AvailableCapacity(t *testing.T) {
	config := MempoolConfig{
		MaxSize:  5,
		MaxTxAge: 1 * time.Hour,
	}
	mp := NewMempool(config)

	assert.Equal(t, 5, mp.AvailableCapacity())

	tx1 := NewStoreTransaction("user1", []byte("data1"), 100)
	tx2 := NewStoreTransaction("user2", []byte("data2"), 100)

	require.NoError(t, mp.Add(tx1))
	assert.Equal(t, 4, mp.AvailableCapacity())

	require.NoError(t, mp.Add(tx2))
	assert.Equal(t, 3, mp.AvailableCapacity())
}

func TestMempool_ConcurrentAccess(t *testing.T) {
	mp := NewDefaultMempool()

	// Concurrent adds
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			tx := NewStoreTransaction("user1", []byte("data"), 100)
			mp.Add(tx)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	assert.Equal(t, 10, mp.Size())
}
