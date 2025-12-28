package blockchain

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Mempool manages pending transactions waiting to be processed
type Mempool struct {
	transactions map[uuid.UUID]*Transaction
	maxSize      int
	maxTxAge     time.Duration
	mu           sync.RWMutex

	// Metrics
	totalAdded    uint64
	totalRejected uint64
	totalExpired  uint64
}

// MempoolConfig contains mempool configuration
type MempoolConfig struct {
	// MaxSize is the maximum number of pending transactions
	MaxSize int
	// MaxTxAge is the maximum age of a transaction before expiration
	MaxTxAge time.Duration
}

// DefaultMempoolConfig returns default mempool configuration
func DefaultMempoolConfig() MempoolConfig {
	return MempoolConfig{
		MaxSize:  10000,              // 10k pending transactions max
		MaxTxAge: 24 * time.Hour,     // Transactions expire after 24 hours
	}
}

// NewMempool creates a new transaction mempool
func NewMempool(config MempoolConfig) *Mempool {
	return &Mempool{
		transactions: make(map[uuid.UUID]*Transaction),
		maxSize:      config.MaxSize,
		maxTxAge:     config.MaxTxAge,
	}
}

// NewDefaultMempool creates a mempool with default configuration
func NewDefaultMempool() *Mempool {
	return NewMempool(DefaultMempoolConfig())
}

// Add adds a transaction to the mempool
func (mp *Mempool) Add(tx *Transaction) error {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	// Validate transaction
	if err := tx.Validate(); err != nil {
		mp.totalRejected++
		return fmt.Errorf("invalid transaction: %w", err)
	}

	// Check if already exists
	if _, exists := mp.transactions[tx.TxID]; exists {
		mp.totalRejected++
		return fmt.Errorf("transaction %s already in mempool", tx.TxID)
	}

	// Check if mempool is full
	if len(mp.transactions) >= mp.maxSize {
		mp.totalRejected++
		return fmt.Errorf("mempool is full (size: %d)", mp.maxSize)
	}

	// Only accept pending transactions
	if tx.Status != TxStatusPending {
		mp.totalRejected++
		return fmt.Errorf("only pending transactions can be added to mempool")
	}

	// Add to mempool
	mp.transactions[tx.TxID] = tx
	mp.totalAdded++

	return nil
}

// Get retrieves a transaction from the mempool
func (mp *Mempool) Get(txID uuid.UUID) (*Transaction, bool) {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	tx, exists := mp.transactions[txID]
	return tx, exists
}

// Remove removes a transaction from the mempool
func (mp *Mempool) Remove(txID uuid.UUID) {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	delete(mp.transactions, txID)
}

// RemoveBatch removes multiple transactions from the mempool
func (mp *Mempool) RemoveBatch(txIDs []uuid.UUID) {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	for _, txID := range txIDs {
		delete(mp.transactions, txID)
	}
}

// GetPending returns all pending transactions
func (mp *Mempool) GetPending(limit int) []*Transaction {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	var pending []*Transaction
	for _, tx := range mp.transactions {
		if tx.IsPending() {
			pending = append(pending, tx)
			if len(pending) >= limit {
				break
			}
		}
	}

	return pending
}

// GetOldest returns the oldest pending transactions
func (mp *Mempool) GetOldest(limit int) []*Transaction {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	var pending []*Transaction
	for _, tx := range mp.transactions {
		if tx.IsPending() {
			pending = append(pending, tx)
		}
	}

	// Sort by creation time (oldest first)
	for i := 0; i < len(pending); i++ {
		for j := i + 1; j < len(pending); j++ {
			if pending[i].CreatedAt.After(pending[j].CreatedAt) {
				pending[i], pending[j] = pending[j], pending[i]
			}
		}
	}

	if len(pending) > limit {
		pending = pending[:limit]
	}

	return pending
}

// GetByType returns transactions of a specific type
func (mp *Mempool) GetByType(txType TransactionType, limit int) []*Transaction {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	var matched []*Transaction
	for _, tx := range mp.transactions {
		if tx.Type == txType {
			matched = append(matched, tx)
			if len(matched) >= limit {
				break
			}
		}
	}

	return matched
}

// Size returns the current number of transactions in the mempool
func (mp *Mempool) Size() int {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	return len(mp.transactions)
}

// Clear removes all transactions from the mempool
func (mp *Mempool) Clear() {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	mp.transactions = make(map[uuid.UUID]*Transaction)
}

// PurgeExpired removes expired transactions from the mempool
func (mp *Mempool) PurgeExpired() int {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	now := time.Now()
	expiredCount := 0

	for txID, tx := range mp.transactions {
		if now.Sub(tx.CreatedAt) > mp.maxTxAge {
			delete(mp.transactions, txID)
			expiredCount++
			mp.totalExpired++
		}
	}

	return expiredCount
}

// PurgeByStatus removes transactions with specific status
func (mp *Mempool) PurgeByStatus(status TransactionStatus) int {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	purgedCount := 0
	for txID, tx := range mp.transactions {
		if tx.Status == status {
			delete(mp.transactions, txID)
			purgedCount++
		}
	}

	return purgedCount
}

// UpdateStatus updates the status of a transaction
func (mp *Mempool) UpdateStatus(txID uuid.UUID, status TransactionStatus) error {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	tx, exists := mp.transactions[txID]
	if !exists {
		return fmt.Errorf("transaction %s not found in mempool", txID)
	}

	tx.Status = status
	return nil
}

// GetStats returns mempool statistics
func (mp *Mempool) GetStats() MempoolStats {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	stats := MempoolStats{
		Size:          len(mp.transactions),
		MaxSize:       mp.maxSize,
		TotalAdded:    mp.totalAdded,
		TotalRejected: mp.totalRejected,
		TotalExpired:  mp.totalExpired,
		ByType:        make(map[TransactionType]int),
		ByStatus:      make(map[TransactionStatus]int),
	}

	// Count by type and status
	for _, tx := range mp.transactions {
		stats.ByType[tx.Type]++
		stats.ByStatus[tx.Status]++
	}

	return stats
}

// MempoolStats contains mempool statistics
type MempoolStats struct {
	Size          int                            `json:"size"`
	MaxSize       int                            `json:"max_size"`
	TotalAdded    uint64                         `json:"total_added"`
	TotalRejected uint64                         `json:"total_rejected"`
	TotalExpired  uint64                         `json:"total_expired"`
	ByType        map[TransactionType]int        `json:"by_type"`
	ByStatus      map[TransactionStatus]int      `json:"by_status"`
}

// StartPurgeLoop starts a background goroutine that periodically purges expired transactions
func (mp *Mempool) StartPurgeLoop(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			mp.PurgeExpired()
		}
	}
}

// WaitForTransaction waits for a transaction to reach a final state or timeout
func (mp *Mempool) WaitForTransaction(ctx context.Context, txID uuid.UUID, timeout time.Duration) (*Transaction, error) {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			tx, exists := mp.Get(txID)
			if !exists {
				// Transaction was removed (likely processed)
				return nil, fmt.Errorf("transaction %s not found in mempool", txID)
			}

			if tx.IsFinalized() {
				return tx, nil
			}

			if time.Now().After(deadline) {
				return nil, fmt.Errorf("timeout waiting for transaction %s", txID)
			}
		}
	}
}

// GetTransactionsByFrom returns transactions from a specific address
func (mp *Mempool) GetTransactionsByFrom(from string, limit int) []*Transaction {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	var matched []*Transaction
	for _, tx := range mp.transactions {
		if tx.From == from {
			matched = append(matched, tx)
			if len(matched) >= limit {
				break
			}
		}
	}

	return matched
}

// HasTransaction checks if a transaction exists in the mempool
func (mp *Mempool) HasTransaction(txID uuid.UUID) bool {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	_, exists := mp.transactions[txID]
	return exists
}

// IsFull returns true if the mempool is at capacity
func (mp *Mempool) IsFull() bool {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	return len(mp.transactions) >= mp.maxSize
}

// AvailableCapacity returns how many more transactions can be added
func (mp *Mempool) AvailableCapacity() int {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	return mp.maxSize - len(mp.transactions)
}
