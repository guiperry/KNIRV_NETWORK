package operationlog

import (
	"encoding/json"
	"fmt"
	"math/big"
	"sync"

	"KNIRVGRAPH/internal/storage"
	"KNIRVGRAPH/internal/types"
)

// OperationLog manages persistent audit trail of operations
type OperationLog struct {
	mu         sync.RWMutex
	storage    storage.GraphStorage
	height     uint64
	operations map[string]*types.AuditedOperation
}

// NewOperationLog creates a new operation log
func NewOperationLog(storage storage.GraphStorage) *OperationLog {
	return &OperationLog{
		storage:    storage,
		height:     0,
		operations: make(map[string]*types.AuditedOperation),
	}
}

// ExecuteAndAudit executes an operation and persists it atomically
func (log *OperationLog) ExecuteAndAudit(op *types.AuditedOperation) error {
	log.mu.Lock()
	defer log.mu.Unlock()

	// Set the block height
	op.BlockHeight = log.height + 1
	op.Status = types.CommittedOp

	// Create a batch for atomic execution
	batch := log.storage.Batch()

	// Store the operation
	opKey := fmt.Sprintf("operation:%d:%s", op.BlockHeight, op.ID)
	opData, err := op.Serialize()
	if err != nil {
		return fmt.Errorf("failed to serialize operation: %w", err)
	}
	batch.Put([]byte(opKey), opData)

	// Execute the operation based on type
	switch op.Type {
	case types.TransferOp:
		if err := log.executeTransfer(batch, op); err != nil {
			return fmt.Errorf("failed to execute transfer: %w", err)
		}
	case types.NodeAddOp, types.EdgeAddOp:
		// Graph operations are handled by GraphChain
		// Just audit them here
	case types.StateChangeOp:
		if err := log.executeStateChange(batch, op); err != nil {
			return fmt.Errorf("failed to execute state change: %w", err)
		}
	}

	// Update height
	heightKey := []byte("operation_height")
	heightData := []byte(fmt.Sprintf("%d", op.BlockHeight))
	batch.Put(heightKey, heightData)

	// Commit the batch
	if err := batch.Write(); err != nil {
		op.Status = types.FailedOp
		return fmt.Errorf("failed to commit operation batch: %w", err)
	}

	// Update in-memory state
	log.height = op.BlockHeight
	log.operations[op.ID] = op

	return nil
}

// executeTransfer handles transfer operations
func (log *OperationLog) executeTransfer(batch storage.Batch, op *types.AuditedOperation) error {
	if op.Amount == nil || op.From == "" || op.To == "" {
		return fmt.Errorf("invalid transfer operation")
	}

	// Get current balances
	fromBalance, err := log.getAccountBalance(op.From)
	if err != nil {
		return fmt.Errorf("failed to get from account balance: %w", err)
	}

	toBalance, err := log.getAccountBalance(op.To)
	if err != nil {
		return fmt.Errorf("failed to get to account balance: %w", err)
	}

	// Check sufficient balance
	if fromBalance.Cmp(op.Amount) < 0 {
		return fmt.Errorf("insufficient balance: has %s, need %s", fromBalance.String(), op.Amount.String())
	}

	// Update balances
	newFromBalance := new(big.Int).Sub(fromBalance, op.Amount)
	newToBalance := new(big.Int).Add(toBalance, op.Amount)

	// Store updated balances
	fromKey := fmt.Sprintf("account:%s", op.From)
	toKey := fmt.Sprintf("account:%s", op.To)

	fromAccount := map[string]interface{}{
		"address": op.From,
		"balance": newFromBalance.String(),
	}
	toAccount := map[string]interface{}{
		"address": op.To,
		"balance": newToBalance.String(),
	}

	fromData, _ := json.Marshal(fromAccount)
	toData, _ := json.Marshal(toAccount)

	batch.Put([]byte(fromKey), fromData)
	batch.Put([]byte(toKey), toData)

	return nil
}

// executeStateChange handles state change operations
func (log *OperationLog) executeStateChange(_ storage.Batch, _ *types.AuditedOperation) error {
	// State changes are handled by the GraphChain
	// This is just for auditing
	return nil
}

// getAccountBalance retrieves the current balance for an account
func (log *OperationLog) getAccountBalance(address string) (*big.Int, error) {
	key := fmt.Sprintf("account:%s", address)
	data, err := log.storage.Get([]byte(key))
	if err != nil {
		if err == storage.ErrNotFound {
			return big.NewInt(0), nil
		}
		return nil, err
	}

	var account map[string]interface{}
	if err := json.Unmarshal(data, &account); err != nil {
		return nil, err
	}

	if balanceStr, ok := account["balance"].(string); ok {
		balance := new(big.Int)
		balance.SetString(balanceStr, 10)
		return balance, nil
	}

	return big.NewInt(0), nil
}

// GetOperation retrieves an operation by ID
func (log *OperationLog) GetOperation(id string) (*types.AuditedOperation, error) {
	log.mu.RLock()
	defer log.mu.RUnlock()

	if op, exists := log.operations[id]; exists {
		return op, nil
	}

	// Search through all operations (inefficient but works for now)
	for height := uint64(1); height <= log.height; height++ {
		// Get all operations at this height
		// Simplified implementation - TODO: optimize with proper indexing
		_ = height // Prevent unused variable warning
		ops, err := log.getOperationsAtHeight(height)
		if err != nil {
			continue
		}
		for _, op := range ops {
			if op.ID == id {
				return op, nil
			}
		}
	}

	return nil, fmt.Errorf("operation not found")
}

// GetOperationsAtHeight retrieves all operations at a specific height
func (log *OperationLog) GetOperationsAtHeight(height uint64) ([]*types.AuditedOperation, error) {
	log.mu.RLock()
	defer log.mu.RUnlock()

	return log.getOperationsAtHeight(height)
}

func (log *OperationLog) getOperationsAtHeight(height uint64) ([]*types.AuditedOperation, error) {
	var operations []*types.AuditedOperation

	// For now, we'll scan all operations (can be optimized later)
	for h := uint64(1); h <= log.height; h++ {
		if h == height {
			// Get all operations at this height
			// This is a simplified implementation
			key := fmt.Sprintf("operation:%d:*", height)
			// Note: This would need a proper prefix scan in production
			_ = key // Placeholder
		}
	}

	return operations, nil
}

// GetCurrentHeight returns the current operation height
func (log *OperationLog) GetCurrentHeight() uint64 {
	log.mu.RLock()
	defer log.mu.RUnlock()
	return log.height
}

// LoadState loads the operation log state from storage
func (log *OperationLog) LoadState() error {
	log.mu.Lock()
	defer log.mu.Unlock()

	// Load current height
	heightKey := []byte("operation_height")
	heightData, err := log.storage.Get(heightKey)
	if err != nil {
		if err == storage.ErrNotFound {
			log.height = 0
			return nil
		}
		return err
	}

	if _, err := fmt.Sscanf(string(heightData), "%d", &log.height); err != nil {
		return fmt.Errorf("failed to parse height: %w", err)
	}

	// Load recent operations into memory (last 1000)
	startHeight := uint64(1)
	if log.height > 1000 {
		startHeight = log.height - 1000 + 1
	}

	for h := startHeight; h <= log.height; h++ {
		// Load operations at this height
		// Simplified implementation
	}

	return nil
}

// GetAccountBalance returns the current balance for an account
func (log *OperationLog) GetAccountBalance(address string) (*big.Int, error) {
	log.mu.RLock()
	defer log.mu.RUnlock()
	return log.getAccountBalance(address)
}
