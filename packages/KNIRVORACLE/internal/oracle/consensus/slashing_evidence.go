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

// AuthorityViolationEvidence is consensus evidence that an authority operator
// misused its authority: a record outside its granted scope or tier
// ("scope_violation"), or two records chained onto one predecessor
// ("equivocation"). It is queued whether or not a bond existed to burn, so Amount
// may be zero.
type AuthorityViolationEvidence struct {
	SchemaVersion string    `json:"schema_version"`
	Kind          string    `json:"kind"`
	Operator      string    `json:"operator"`
	ChainID       string    `json:"chain_id"`
	RecordHash    string    `json:"record_hash"`
	OtherHash     string    `json:"other_record_hash,omitempty"`
	Reason        string    `json:"reason"`
	BondOwner     string    `json:"bond_owner,omitempty"`
	Amount        uint64    `json:"amount"`
	OracleHeight  uint64    `json:"oracle_height"`
	OccurredAt    time.Time `json:"occurred_at"`
}

// AuthorityViolationSchema identifies the evidence payload version.
const AuthorityViolationSchema = "knirv.authority-violation.v1"

func (e AuthorityViolationEvidence) Height() BlockHeight { return BlockHeight(e.OracleHeight) }
func (e AuthorityViolationEvidence) Bytes() []byte {
	payload, _ := json.Marshal(e)
	return payload
}
func (e AuthorityViolationEvidence) Hash() []byte {
	sum := sha256.Sum256(e.Bytes())
	return sum[:]
}
func (e AuthorityViolationEvidence) ValidateBasic() error {
	if e.SchemaVersion != AuthorityViolationSchema || e.Operator == "" || e.ChainID == "" || e.RecordHash == "" || e.Reason == "" {
		return fmt.Errorf("invalid authority violation evidence")
	}
	if e.Kind != "scope_violation" && e.Kind != "equivocation" {
		return fmt.Errorf("unknown authority violation kind %q", e.Kind)
	}
	return nil
}
