package blockchain

import (
	"fmt"
	"sync"
	"time"

	agentlog "KNIRVCHAIN/pkg/log"
)

// TransactionPoolManager manages different transaction pools
type TransactionPoolManager struct {
	blockchain *BlockchainStruct
	mu         sync.Mutex

	// Main transaction pool (reference to blockchain's pool)
	// This is a reference, not a copy, to avoid duplication
	mainPoolRef *[]*Transaction

	// For PAPs: Plugin Author Subpool
	pluginAuthorSubpool map[string]*Transaction // Map txHash -> transaction

	// For tracking delegated transactions
	delegatedTransactions map[string]time.Time // Map txHash -> delegation time
}

// NewTransactionPoolManager creates a new transaction pool manager
func NewTransactionPoolManager(bc *BlockchainStruct) *TransactionPoolManager {
	return &TransactionPoolManager{
		blockchain:            bc,
		mainPoolRef:           &bc.TransactionPool,
		pluginAuthorSubpool:   make(map[string]*Transaction),
		delegatedTransactions: make(map[string]time.Time),
	}
}

// Methods to manage the Plugin Author Subpool

// AddToPASPool adds a transaction to the Plugin Author Subpool
func (tpm *TransactionPoolManager) AddToPASPool(tx *Transaction) {
	tpm.mu.Lock()
	defer tpm.mu.Unlock()
	tpm.pluginAuthorSubpool[tx.TransactionHash] = tx
	agentlog.LogInfo(fmt.Sprintf("Added transaction %s to PASPool", tx.TransactionHash))
}

// RemoveFromPASPool removes a transaction from the Plugin Author Subpool
func (tpm *TransactionPoolManager) RemoveFromPASPool(txHash string) {
	tpm.mu.Lock()
	defer tpm.mu.Unlock()
	delete(tpm.pluginAuthorSubpool, txHash)
	agentlog.LogInfo(fmt.Sprintf("Removed transaction %s from PASPool", txHash))
}

// GetPASPoolTxs returns all transactions in the Plugin Author Subpool
func (tpm *TransactionPoolManager) GetPASPoolTxs() []*Transaction {
	tpm.mu.Lock()
	defer tpm.mu.Unlock()

	txs := make([]*Transaction, 0, len(tpm.pluginAuthorSubpool))
	for _, tx := range tpm.pluginAuthorSubpool {
		txs = append(txs, tx)
	}
	return txs
}

// GetPASPoolSize returns the number of transactions in the Plugin Author Subpool
func (tpm *TransactionPoolManager) GetPASPoolSize() int {
	tpm.mu.Lock()
	defer tpm.mu.Unlock()
	return len(tpm.pluginAuthorSubpool)
}

// Methods to track delegated transactions

// MarkAsDelegated marks a transaction as delegated
func (tpm *TransactionPoolManager) MarkAsDelegated(txHash string) {
	tpm.mu.Lock()
	defer tpm.mu.Unlock()
	tpm.delegatedTransactions[txHash] = time.Now()
	agentlog.LogInfo(fmt.Sprintf("Marked transaction %s as delegated", txHash))
}

// IsDelegated checks if a transaction is currently delegated
func (tpm *TransactionPoolManager) IsDelegated(txHash string) bool {
	tpm.mu.Lock()
	defer tpm.mu.Unlock()
	_, exists := tpm.delegatedTransactions[txHash]
	return exists
}

// UnmarkAsDelegated removes a transaction from the delegated tracking
func (tpm *TransactionPoolManager) UnmarkAsDelegated(txHash string) {
	tpm.mu.Lock()
	defer tpm.mu.Unlock()
	delete(tpm.delegatedTransactions, txHash)
	agentlog.LogInfo(fmt.Sprintf("Unmarked transaction %s as delegated", txHash))
}

// GetDelegatedTransactionsCount returns the number of currently delegated transactions
func (tpm *TransactionPoolManager) GetDelegatedTransactionsCount() int {
	tpm.mu.Lock()
	defer tpm.mu.Unlock()
	return len(tpm.delegatedTransactions)
}

// ReclaimStaleTransactions reclaims transactions that have been delegated for too long
func (tpm *TransactionPoolManager) ReclaimStaleTransactions(maxStaleTime time.Duration) {
	tpm.mu.Lock()
	defer tpm.mu.Unlock()

	now := time.Now()
	reclaimedCount := 0

	for txHash, delegationTime := range tpm.delegatedTransactions {
		if now.Sub(delegationTime) > maxStaleTime {
			// Transaction is stale, remove from delegated tracking
			delete(tpm.delegatedTransactions, txHash)
			reclaimedCount++

			// If we still have the transaction in our reference, it will be
			// processed by NAPs in the next mining cycle
			agentlog.LogInfo(fmt.Sprintf("Reclaimed stale delegated transaction %s", txHash))
		}
	}

	if reclaimedCount > 0 {
		agentlog.LogInfo(fmt.Sprintf("Reclaimed %d stale delegated transactions", reclaimedCount))
	}
}

// GetMainPoolSize returns the size of the main transaction pool
func (tpm *TransactionPoolManager) GetMainPoolSize() int {
	if tpm.mainPoolRef == nil {
		return 0
	}
	tpm.blockchain.Lock()
	defer tpm.blockchain.Unlock()
	return len(*tpm.mainPoolRef)
}

// GetMainPoolTxs returns a copy of transactions from the main pool
func (tpm *TransactionPoolManager) GetMainPoolTxs() []*Transaction {
	if tpm.mainPoolRef == nil {
		return nil
	}

	tpm.blockchain.Lock()
	defer tpm.blockchain.Unlock()

	txs := make([]*Transaction, len(*tpm.mainPoolRef))
	copy(txs, *tpm.mainPoolRef)
	return txs
}

// IsTransactionInPASPool checks if a transaction exists in the PAS pool
func (tpm *TransactionPoolManager) IsTransactionInPASPool(txHash string) bool {
	tpm.mu.Lock()
	defer tpm.mu.Unlock()
	_, exists := tpm.pluginAuthorSubpool[txHash]
	return exists
}

// GetPASPoolTransaction retrieves a specific transaction from the PAS pool
func (tpm *TransactionPoolManager) GetPASPoolTransaction(txHash string) *Transaction {
	tpm.mu.Lock()
	defer tpm.mu.Unlock()
	return tpm.pluginAuthorSubpool[txHash]
}

// ClearPASPool removes all transactions from the Plugin Author Subpool
func (tpm *TransactionPoolManager) ClearPASPool() {
	tpm.mu.Lock()
	defer tpm.mu.Unlock()

	count := len(tpm.pluginAuthorSubpool)
	tpm.pluginAuthorSubpool = make(map[string]*Transaction)

	if count > 0 {
		agentlog.LogInfo(fmt.Sprintf("Cleared %d transactions from PASPool", count))
	}
}

// GetPoolStats returns statistics about all pools
func (tpm *TransactionPoolManager) GetPoolStats() map[string]interface{} {
	tpm.mu.Lock()
	defer tpm.mu.Unlock()

	mainPoolSize := 0
	if tpm.mainPoolRef != nil {
		tpm.blockchain.Lock()
		mainPoolSize = len(*tpm.mainPoolRef)
		tpm.blockchain.Unlock()
	}

	return map[string]interface{}{
		"main_pool_size":         mainPoolSize,
		"pas_pool_size":          len(tpm.pluginAuthorSubpool),
		"delegated_transactions": len(tpm.delegatedTransactions),
	}
}
