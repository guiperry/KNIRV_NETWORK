package blockchain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// AssertionAttestation is KNIRVHASHER's chain-owned witness format. Once in a
// block it is committed by TxMerkleRoot, never by a Bitcoin header field.
type AssertionAttestation struct {
	SchemaVersion    int32  `json:"schema_version"`
	AssertionKey     string `json:"assertion_key"`
	CommitmentTarget uint32 `json:"commitment_target"`
	ContextHash      uint32 `json:"context_hash"`
	BestSeed         string `json:"best_seed"`
}

func NewAssertionAttestationTransaction(from string, a AssertionAttestation, fee uint64, timestamp ...int64) (*Transaction, error) {
	if err := ValidateAssertionAttestation(a); err != nil {
		return nil, err
	}
	data, err := json.Marshal(a)
	if err != nil {
		return nil, fmt.Errorf("marshal assertion attestation: %w", err)
	}
	tx := NewTransaction(from, "", 0, data, timestamp...)
	tx.Fee, tx.Type, tx.TransactionHash = fee, TransactionTypeAssertionAttestation, tx.Hash()
	return tx, nil
}

func ValidateAssertionAttestation(a AssertionAttestation) error {
	if a.SchemaVersion < 2 || a.AssertionKey == "" || a.BestSeed == "" {
		return fmt.Errorf("invalid assertion attestation")
	}
	if _, err := hex.DecodeString(a.BestSeed); err != nil {
		return fmt.Errorf("best_seed must be hexadecimal: %w", err)
	}
	return nil
}

func AssertionMerkleCommitment(tx *Transaction) ([sha256.Size]byte, error) {
	if tx == nil || tx.Type != TransactionTypeAssertionAttestation {
		return [sha256.Size]byte{}, fmt.Errorf("not an assertion attestation transaction")
	}
	var a AssertionAttestation
	if err := json.Unmarshal(tx.Data, &a); err != nil {
		return [sha256.Size]byte{}, err
	}
	if err := ValidateAssertionAttestation(a); err != nil {
		return [sha256.Size]byte{}, err
	}
	return txMerkleLeafHash(canonicalTransactionBytes(tx)), nil
}
