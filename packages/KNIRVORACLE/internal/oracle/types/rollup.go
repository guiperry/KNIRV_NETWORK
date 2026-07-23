package types

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
)

type RollupStatus string

const (
	RollupStatusSubmitted RollupStatus = "submitted"
	RollupStatusFinalized RollupStatus = "finalized"
	RollupStatusDisputed  RollupStatus = "disputed"
)

// RollupRecord is submitted by a registered chain (e.g. KNIRVSERVER's
// Transaction Chain rollup service) via POST /oracle/v3/rollups/submit.
// Proposer/Signatures authorize the submission the same way Checkpoint's do
// — the submitting chain must be registered (POST /oracle/v3/registry/register)
// and Proposer must be one of its registered authors.
type RollupRecord struct {
	ID          string                 `json:"id"`
	BatchRoot   string                 `json:"batch_root"`
	ChainID     string                 `json:"chain_id"`
	StartHeight uint64                 `json:"start_height"`
	EndHeight   uint64                 `json:"end_height"`
	BlockCount  int                    `json:"block_count"`
	TxCount     int                    `json:"tx_count"`
	Proposer    string                 `json:"proposer,omitempty"`
	Signatures  []AuthorSig            `json:"signatures,omitempty"`
	Status      RollupStatus           `json:"status"`
	SubmittedAt time.Time              `json:"submitted_at"`
	FinalizedAt *time.Time             `json:"finalized_at,omitempty"`
	DisputedAt  *time.Time             `json:"disputed_at,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Dispute     string                 `json:"dispute,omitempty"`
}

// Digest is the canonical hash of the rollup submission body that Proposer
// signs — everything that identifies the batch, excluding the signatures
// themselves and any post-submission mutable state (Status, timestamps).
func (rec *RollupRecord) Digest() [32]byte {
	parts := [][]byte{
		[]byte(rec.ID), {0x00},
		[]byte(rec.BatchRoot), {0x00},
		[]byte(rec.ChainID), {0x00},
		u64Bytes(rec.StartHeight),
		u64Bytes(rec.EndHeight),
		u64Bytes(uint64(rec.BlockCount)),
		u64Bytes(uint64(rec.TxCount)),
		[]byte(rec.Proposer), {0x00},
	}
	var data []byte
	for _, p := range parts {
		data = append(data, p...)
	}
	var out [32]byte
	copy(out[:], crypto.Keccak256(data))
	return out
}

// RollupActionDigest is the canonical hash signed to authorize a
// finalize/dispute transition on an already-submitted rollup. Deliberately
// separate from Digest (the submission digest) since a status transition
// isn't re-authorizing the batch contents, just the action.
func RollupActionDigest(id, chainID, action string) [32]byte {
	data := fmt.Sprintf("%s\x00%s\x00%s\x00", id, chainID, action)
	var out [32]byte
	copy(out[:], crypto.Keccak256([]byte(data)))
	return out
}
