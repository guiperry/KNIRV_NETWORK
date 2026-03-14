package core

import (
	"time"
)

// NodeInfo represents information about a KNIRVCHAIN node
type NodeInfo struct {
	Version     string    `json:"version"`
	NodeID      string    `json:"node_id"`
	StartTime   time.Time `json:"start_time"`
	Uptime      string    `json:"uptime"`
	NetworkID   string    `json:"network_id"`
	PeerCount   int       `json:"peer_count"`
	SyncStatus  string    `json:"sync_status"`
	BlockHeight uint64    `json:"block_height"`
}

// ChainInfo represents information about the KNIRVCHAIN blockchain
type ChainInfo struct {
	NetworkID       string    `json:"network_id"`
	BlockHeight     uint64    `json:"block_height"`
	LastBlockTime   time.Time `json:"last_block_time"`
	TotalTxns       uint64    `json:"total_txns"`
	PendingTxns     uint64    `json:"pending_txns"`
	ActiveCapCount  uint64    `json:"active_cap_count"`
	ValidatorCount  int       `json:"validator_count"`
	ConsensusStatus string    `json:"consensus_status"`
}

// BlockInfo represents information about a block in the KNIRVCHAIN blockchain
type BlockInfo struct {
	Height       uint64    `json:"height"`
	Hash         string    `json:"hash"`
	PreviousHash string    `json:"previous_hash"`
	Timestamp    time.Time `json:"timestamp"`
	TxnCount     int       `json:"txn_count"`
	Size         int       `json:"size"`
	Proposer     string    `json:"proposer"`
	Transactions []string  `json:"transactions"`
}

// TransactionInfo represents information about a transaction
type TransactionInfo struct {
	Hash        string    `json:"hash"`
	BlockHeight uint64    `json:"block_height"`
	BlockHash   string    `json:"block_hash"`
	From        string    `json:"from"`
	To          string    `json:"to"`
	Value       string    `json:"value"`
	Fee         string    `json:"fee"`
	Timestamp   time.Time `json:"timestamp"`
	Status      string    `json:"status"`
	Type        string    `json:"type"`
	Data        string    `json:"data"`
}

// TransactionPoolInfo represents information about the transaction pool
type TransactionPoolInfo struct {
	PendingCount int               `json:"pending_count"`
	QueuedCount  int               `json:"queued_count"`
	Transactions []TransactionInfo `json:"transactions"`
}

// MCPPrepareCapabilityRegistrationRequest represents a request to prepare a capability registration
type MCPPrepareCapabilityRegistrationRequest struct {
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Description string                 `json:"description"`
	Author      string                 `json:"author"`
	License     string                 `json:"license"`
	Descriptor  map[string]interface{} `json:"descriptor"`
}

// MCPPrepareCapabilityRegistrationResponse represents a response from preparing a capability registration
type MCPPrepareCapabilityRegistrationResponse struct {
	CapabilityID                 string                     `json:"capability_id"`
	TxnData                      string                     `json:"txn_data"`
	Fee                          string                     `json:"fee"`
	Descriptor                   map[string]interface{}     `json:"descriptor"`
	TransactionDetailsForSigning UnsignedTransactionDetails `json:"transaction_details_for_signing"`
}

// SubmitTransactionRequest represents a request to submit a transaction
type SubmitTransactionRequest struct {
	From            string `json:"from"`
	To              string `json:"to,omitempty"`
	Value           uint64 `json:"value"`
	Data            []byte `json:"data"`
	Timestamp       int64  `json:"timestamp"`
	Fee             uint64 `json:"fee"`
	Type            string `json:"type"`
	PublicKey       string `json:"publicKey"`
	Signature       []byte `json:"signature"`
	TransactionHash string `json:"transactionHash"`
}

// SubmitTransactionResponse represents a response from submitting a transaction
type SubmitTransactionResponse struct {
	TransactionHash string `json:"transactionHash"`
	Status          string `json:"status"`
	BlockHeight     uint64 `json:"blockHeight,omitempty"`
	BlockHash       string `json:"blockHash,omitempty"`
	Timestamp       int64  `json:"timestamp,omitempty"`
}

// ErrorResponse represents an error response from the API
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}
