package types

import "time"

// Transaction represents a blockchain transaction
type Transaction struct {
	Data            string    `json:"data"`
	Signature       string    `json:"signature"`
	TransactionHash string    `json:"transaction_hash,omitempty"`
	From            Address   `json:"from"`
	To              Address   `json:"to"`
	Amount          uint64    `json:"amount,omitempty"`
	Nonce           uint64    `json:"nonce"`
	Timestamp       time.Time `json:"timestamp"`
}

// TransactionReceipt represents a transaction receipt
type TransactionReceipt struct {
	TransactionHash string   `json:"transaction_hash"`
	BlockHeight     uint64   `json:"block_height"`
	BlockHash       string   `json:"block_hash"`
	Status          TxStatus `json:"status"`
	GasUsed         uint64   `json:"gas_used"`
	Timestamp       int64    `json:"timestamp"`
	Error           string   `json:"error,omitempty"`
}

// TxStatus represents transaction status
type TxStatus int

const (
	TxStatusPending TxStatus = iota
	TxStatusConfirmed
	TxStatusFailed
	TxStatusReverted
)

// String returns the string representation of TxStatus
func (s TxStatus) String() string {
	switch s {
	case TxStatusPending:
		return "pending"
	case TxStatusConfirmed:
		return "confirmed"
	case TxStatusFailed:
		return "failed"
	case TxStatusReverted:
		return "reverted"
	default:
		return "unknown"
	}
}
