package types

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"time"
)

// AuditedOperation represents a database transaction operation
type AuditedOperation struct {
	ID          string                 `json:"id"`
	Type        OperationType          `json:"type"`
	From        string                 `json:"from,omitempty"`
	To          string                 `json:"to,omitempty"`
	Amount      *big.Int               `json:"amount,omitempty"`
	NodeID      string                 `json:"node_id,omitempty"`
	EdgeID      string                 `json:"edge_id,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	BlockHeight uint64                 `json:"block_height"`
	Status      OperationStatus        `json:"status"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Hash        string                 `json:"hash"`
}

type OperationType int

const (
	TransferOp OperationType = iota
	NodeAddOp
	EdgeAddOp
	StateChangeOp
)

type OperationStatus int

const (
	PendingOp OperationStatus = iota
	CommittedOp
	FailedOp
)

// NewAuditedOperation creates a new audited operation
func NewAuditedOperation(opType OperationType, from, to string, amount *big.Int, nodeID, edgeID string, metadata map[string]interface{}) *AuditedOperation {
	op := &AuditedOperation{
		ID:        generateOperationID(),
		Type:      opType,
		From:      from,
		To:        to,
		Amount:    amount,
		NodeID:    nodeID,
		EdgeID:    edgeID,
		Timestamp: time.Now(),
		Status:    PendingOp,
		Metadata:  metadata,
	}
	op.Hash = op.calculateHash()
	return op
}

// calculateHash computes the hash of the operation
func (op *AuditedOperation) calculateHash() string {
	data, _ := json.Marshal(struct {
		ID        string                 `json:"id"`
		Type      OperationType          `json:"type"`
		From      string                 `json:"from,omitempty"`
		To        string                 `json:"to,omitempty"`
		Amount    *big.Int               `json:"amount,omitempty"`
		NodeID    string                 `json:"node_id,omitempty"`
		EdgeID    string                 `json:"edge_id,omitempty"`
		Timestamp time.Time              `json:"timestamp"`
		Metadata  map[string]interface{} `json:"metadata,omitempty"`
	}{
		ID:        op.ID,
		Type:      op.Type,
		From:      op.From,
		To:        op.To,
		Amount:    op.Amount,
		NodeID:    op.NodeID,
		EdgeID:    op.EdgeID,
		Timestamp: op.Timestamp,
		Metadata:  op.Metadata,
	})
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// Serialize converts the operation to JSON bytes
func (op *AuditedOperation) Serialize() ([]byte, error) {
	return json.Marshal(op)
}

// DeserializeOperation converts JSON bytes to an AuditedOperation
func DeserializeOperation(data []byte) (*AuditedOperation, error) {
	var op AuditedOperation
	if err := json.Unmarshal(data, &op); err != nil {
		return nil, err
	}
	return &op, nil
}

func generateOperationID() string {
	hash := sha256.Sum256([]byte(time.Now().String() + "operation"))
	return hex.EncodeToString(hash[:])[:16]
}
