package types

import "time"

type RollupStatus string

const (
	RollupStatusSubmitted RollupStatus = "submitted"
	RollupStatusFinalized RollupStatus = "finalized"
	RollupStatusDisputed  RollupStatus = "disputed"
)

type RollupRecord struct {
	ID          string                 `json:"id"`
	BatchRoot   string                 `json:"batch_root"`
	ChainID     string                 `json:"chain_id"`
	StartHeight uint64                 `json:"start_height"`
	EndHeight   uint64                 `json:"end_height"`
	BlockCount  int                    `json:"block_count"`
	TxCount     int                    `json:"tx_count"`
	Status      RollupStatus           `json:"status"`
	SubmittedAt time.Time              `json:"submitted_at"`
	FinalizedAt *time.Time             `json:"finalized_at,omitempty"`
	DisputedAt  *time.Time             `json:"disputed_at,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Dispute     string                 `json:"dispute,omitempty"`
}
