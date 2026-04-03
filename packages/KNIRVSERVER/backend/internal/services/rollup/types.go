package rollup

import "time"

type BatchStatus string

const (
	StatusPending   BatchStatus = "pending"
	StatusBuilt     BatchStatus = "built"
	StatusSubmitted BatchStatus = "submitted"
	StatusSettled   BatchStatus = "settled"
	StatusDisputed  BatchStatus = "disputed"
	StatusFailed    BatchStatus = "failed"
)

type TransactionRecord struct {
	TransactionHash string `json:"transaction_hash"`
	From            string `json:"from"`
	To              string `json:"to"`
	Amount          int64  `json:"amount"`
	Value           int64  `json:"value"`
	Type            string `json:"type"`
	Timestamp       int64  `json:"timestamp"`
	BlockHeight     uint64 `json:"block_height"`
	Status          string `json:"status"`
}

type BlockRecord struct {
	BlockNumber  uint64              `json:"block_number"`
	Timestamp    int64               `json:"timestamp"`
	Hash         string              `json:"hash"`
	PrevHash     string              `json:"prev_hash"`
	Transactions []TransactionRecord `json:"transactions"`
}

type ChainSnapshot struct {
	ChainID string        `json:"chain_id"`
	Height  uint64        `json:"height"`
	Blocks  []BlockRecord `json:"blocks"`
}

type SettlementMetadata struct {
	BatchID      string    `json:"batch_id"`
	BatchRoot    string    `json:"batch_root"`
	ChainID      string    `json:"chain_id"`
	StartHeight  uint64    `json:"start_height"`
	EndHeight    uint64    `json:"end_height"`
	BlockCount   int       `json:"block_count"`
	TxCount      int       `json:"tx_count"`
	SubmittedAt  time.Time `json:"submitted_at"`
	SettledAt    time.Time `json:"settled_at,omitempty"`
	SettlementID string    `json:"settlement_id,omitempty"`
}

type RollupBatch struct {
	ID          string             `json:"id"`
	ChainID     string             `json:"chain_id"`
	StartHeight uint64             `json:"start_height"`
	EndHeight   uint64             `json:"end_height"`
	Blocks      []BlockRecord      `json:"blocks"`
	BatchRoot   string             `json:"batch_root"`
	Status      BatchStatus        `json:"status"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
	Error       string             `json:"error,omitempty"`
	Settlement  SettlementMetadata `json:"settlement,omitempty"`
}

type ServiceStatus struct {
	LastProcessed     uint64              `json:"last_processed"`
	BatchCount        int                 `json:"batch_count"`
	BatchesByStatus   map[BatchStatus]int `json:"batches_by_status"`
	LatestBatchID     string              `json:"latest_batch_id,omitempty"`
	LatestBatchStatus BatchStatus         `json:"latest_batch_status,omitempty"`
	LatestUpdatedAt   time.Time           `json:"latest_updated_at,omitempty"`
}
