package blockchain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// TransactionType represents the type of blockchain transaction
type TransactionType string

const (
	// TxTypeStore represents a memory storage transaction
	TxTypeStore TransactionType = "STORE"
	// TxTypeRetrieve represents a memory retrieval transaction
	TxTypeRetrieve TransactionType = "RETRIEVE"
	// TxTypeUpdate represents a memory update transaction
	TxTypeUpdate TransactionType = "UPDATE"
	// TxTypeDelete represents a memory deletion transaction
	TxTypeDelete TransactionType = "DELETE"
)

// TransactionStatus represents the status of a transaction
type TransactionStatus string

const (
	// TxStatusPending - transaction is in mempool awaiting processing
	TxStatusPending TransactionStatus = "PENDING"
	// TxStatusProcessing - transaction is being processed
	TxStatusProcessing TransactionStatus = "PROCESSING"
	// TxStatusConfirmed - transaction has been included in a block
	TxStatusConfirmed TransactionStatus = "CONFIRMED"
	// TxStatusFailed - transaction processing failed
	TxStatusFailed TransactionStatus = "FAILED"
	// TxStatusRejected - transaction was rejected (validation failed)
	TxStatusRejected TransactionStatus = "REJECTED"
)

// Transaction represents a blockchain transaction
type Transaction struct {
	// TxID is the unique transaction identifier
	TxID uuid.UUID `json:"tx_id"`

	// Type specifies the transaction type
	Type TransactionType `json:"type"`

	// From is the sender's address/ID
	From string `json:"from"`

	// To is the recipient's address/ID (may be empty for STORE)
	To string `json:"to,omitempty"`

	// BlockID is the ID of the target block (for UPDATE/DELETE/RETRIEVE)
	BlockID *uuid.UUID `json:"block_id,omitempty"`

	// Data contains transaction-specific data (GLB data for STORE, query for RETRIEVE, etc.)
	Data []byte `json:"data,omitempty"`

	// NRNCost is the NRN token cost for this transaction
	NRNCost uint64 `json:"nrn_cost"`

	// Status tracks the current transaction status
	Status TransactionStatus `json:"status"`

	// CreatedAt is when the transaction was created
	CreatedAt time.Time `json:"created_at"`

	// ProcessedAt is when the transaction was processed (nil if pending)
	ProcessedAt *time.Time `json:"processed_at,omitempty"`

	// ResultBlockID is the block ID created by this transaction (for STORE)
	ResultBlockID *uuid.UUID `json:"result_block_id,omitempty"`

	// Error contains error message if transaction failed
	Error string `json:"error,omitempty"`

	// Metadata contains additional transaction metadata
	Metadata map[string]string `json:"metadata,omitempty"`
}

// NewTransaction creates a new transaction
func NewTransaction(txType TransactionType, from string, nrnCost uint64) *Transaction {
	now := time.Now()
	return &Transaction{
		TxID:      uuid.New(),
		Type:      txType,
		From:      from,
		NRNCost:   nrnCost,
		Status:    TxStatusPending,
		CreatedAt: now,
		Metadata:  make(map[string]string),
	}
}

// NewStoreTransaction creates a transaction for storing a memory block
func NewStoreTransaction(from string, data []byte, nrnCost uint64) *Transaction {
	tx := NewTransaction(TxTypeStore, from, nrnCost)
	tx.Data = data
	return tx
}

// NewRetrieveTransaction creates a transaction for retrieving a memory block
func NewRetrieveTransaction(from string, blockID uuid.UUID) *Transaction {
	tx := NewTransaction(TxTypeRetrieve, from, 0) // Retrieval is free
	tx.BlockID = &blockID
	return tx
}

// NewUpdateTransaction creates a transaction for updating a memory block
func NewUpdateTransaction(from string, blockID uuid.UUID, data []byte, nrnCost uint64) *Transaction {
	tx := NewTransaction(TxTypeUpdate, from, nrnCost)
	tx.BlockID = &blockID
	tx.Data = data
	return tx
}

// NewDeleteTransaction creates a transaction for deleting a memory block
func NewDeleteTransaction(from string, blockID uuid.UUID) *Transaction {
	tx := NewTransaction(TxTypeDelete, from, 0) // Deletion is free
	tx.BlockID = &blockID
	return tx
}

// Validate performs basic validation on the transaction
func (tx *Transaction) Validate() error {
	// Validate transaction ID
	if tx.TxID == uuid.Nil {
		return fmt.Errorf("transaction ID cannot be nil")
	}

	// Validate type
	if !isValidTransactionType(tx.Type) {
		return fmt.Errorf("invalid transaction type: %s", tx.Type)
	}

	// Validate from address
	if tx.From == "" {
		return fmt.Errorf("from address cannot be empty")
	}

	// Validate status
	if !isValidTransactionStatus(tx.Status) {
		return fmt.Errorf("invalid transaction status: %s", tx.Status)
	}

	// Type-specific validation
	switch tx.Type {
	case TxTypeStore:
		if len(tx.Data) == 0 {
			return fmt.Errorf("STORE transaction must have data")
		}
		if tx.NRNCost == 0 {
			return fmt.Errorf("STORE transaction must have non-zero cost")
		}

	case TxTypeRetrieve:
		if tx.BlockID == nil {
			return fmt.Errorf("RETRIEVE transaction must specify block ID")
		}

	case TxTypeUpdate:
		if tx.BlockID == nil {
			return fmt.Errorf("UPDATE transaction must specify block ID")
		}
		if len(tx.Data) == 0 {
			return fmt.Errorf("UPDATE transaction must have data")
		}
		if tx.NRNCost == 0 {
			return fmt.Errorf("UPDATE transaction must have non-zero cost")
		}

	case TxTypeDelete:
		if tx.BlockID == nil {
			return fmt.Errorf("DELETE transaction must specify block ID")
		}
	}

	return nil
}

// MarkProcessing marks the transaction as being processed
func (tx *Transaction) MarkProcessing() {
	tx.Status = TxStatusProcessing
}

// MarkConfirmed marks the transaction as confirmed with optional result block ID
func (tx *Transaction) MarkConfirmed(resultBlockID *uuid.UUID) {
	now := time.Now()
	tx.Status = TxStatusConfirmed
	tx.ProcessedAt = &now
	if resultBlockID != nil {
		tx.ResultBlockID = resultBlockID
	}
}

// MarkFailed marks the transaction as failed with an error message
func (tx *Transaction) MarkFailed(err error) {
	now := time.Now()
	tx.Status = TxStatusFailed
	tx.ProcessedAt = &now
	if err != nil {
		tx.Error = err.Error()
	}
}

// MarkRejected marks the transaction as rejected with a reason
func (tx *Transaction) MarkRejected(reason string) {
	now := time.Now()
	tx.Status = TxStatusRejected
	tx.ProcessedAt = &now
	tx.Error = reason
}

// IsPending returns true if the transaction is pending
func (tx *Transaction) IsPending() bool {
	return tx.Status == TxStatusPending
}

// IsProcessing returns true if the transaction is being processed
func (tx *Transaction) IsProcessing() bool {
	return tx.Status == TxStatusProcessing
}

// IsConfirmed returns true if the transaction is confirmed
func (tx *Transaction) IsConfirmed() bool {
	return tx.Status == TxStatusConfirmed
}

// IsFailed returns true if the transaction failed
func (tx *Transaction) IsFailed() bool {
	return tx.Status == TxStatusFailed
}

// IsRejected returns true if the transaction was rejected
func (tx *Transaction) IsRejected() bool {
	return tx.Status == TxStatusRejected
}

// IsFinalized returns true if the transaction is in a final state
func (tx *Transaction) IsFinalized() bool {
	return tx.IsConfirmed() || tx.IsFailed() || tx.IsRejected()
}

// Age returns how long ago the transaction was created
func (tx *Transaction) Age() time.Duration {
	return time.Since(tx.CreatedAt)
}

// ProcessingTime returns how long the transaction took to process (if finalized)
func (tx *Transaction) ProcessingTime() time.Duration {
	if tx.ProcessedAt == nil {
		return 0
	}
	return tx.ProcessedAt.Sub(tx.CreatedAt)
}

// SetMetadata sets a metadata key-value pair
func (tx *Transaction) SetMetadata(key, value string) {
	if tx.Metadata == nil {
		tx.Metadata = make(map[string]string)
	}
	tx.Metadata[key] = value
}

// GetMetadata retrieves a metadata value
func (tx *Transaction) GetMetadata(key string) (string, bool) {
	if tx.Metadata == nil {
		return "", false
	}
	val, ok := tx.Metadata[key]
	return val, ok
}

// isValidTransactionType checks if a transaction type is valid
func isValidTransactionType(txType TransactionType) bool {
	switch txType {
	case TxTypeStore, TxTypeRetrieve, TxTypeUpdate, TxTypeDelete:
		return true
	default:
		return false
	}
}

// isValidTransactionStatus checks if a transaction status is valid
func isValidTransactionStatus(status TransactionStatus) bool {
	switch status {
	case TxStatusPending, TxStatusProcessing, TxStatusConfirmed, TxStatusFailed, TxStatusRejected:
		return true
	default:
		return false
	}
}

// TransactionReceipt represents a receipt for a completed transaction
type TransactionReceipt struct {
	TxID          uuid.UUID         `json:"tx_id"`
	Type          TransactionType   `json:"type"`
	Status        TransactionStatus `json:"status"`
	ResultBlockID *uuid.UUID        `json:"result_block_id,omitempty"`
	NRNCost       uint64            `json:"nrn_cost"`
	ProcessedAt   time.Time         `json:"processed_at"`
	Error         string            `json:"error,omitempty"`
}

// GetReceipt returns a receipt for this transaction
func (tx *Transaction) GetReceipt() *TransactionReceipt {
	receipt := &TransactionReceipt{
		TxID:          tx.TxID,
		Type:          tx.Type,
		Status:        tx.Status,
		ResultBlockID: tx.ResultBlockID,
		NRNCost:       tx.NRNCost,
		Error:         tx.Error,
	}

	if tx.ProcessedAt != nil {
		receipt.ProcessedAt = *tx.ProcessedAt
	}

	return receipt
}
