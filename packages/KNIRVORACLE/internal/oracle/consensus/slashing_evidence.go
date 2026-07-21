package consensus

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"
)

// CheckpointSlashingEvidence is consensus evidence emitted whenever a
// checkpoint rejection burns a chain bond.
type CheckpointSlashingEvidence struct {
	SchemaVersion  string    `json:"schema_version"`
	ChainID        string    `json:"chain_id"`
	BondOwner      string    `json:"bond_owner"`
	CheckpointLeaf uint64    `json:"checkpoint_leaf"`
	Reason         string    `json:"reason"`
	Amount         uint64    `json:"amount"`
	OracleHeight   uint64    `json:"oracle_height"`
	OccurredAt     time.Time `json:"occurred_at"`
}

func (e CheckpointSlashingEvidence) Height() BlockHeight { return BlockHeight(e.OracleHeight) }
func (e CheckpointSlashingEvidence) Bytes() []byte {
	payload, _ := json.Marshal(e)
	return payload
}
func (e CheckpointSlashingEvidence) Hash() []byte {
	sum := sha256.Sum256(e.Bytes())
	return sum[:]
}
func (e CheckpointSlashingEvidence) ValidateBasic() error {
	if e.SchemaVersion != "knirv.slashing-evidence.v1" || e.ChainID == "" || e.Reason == "" || e.Amount == 0 {
		return fmt.Errorf("invalid checkpoint slashing evidence")
	}
	return nil
}
