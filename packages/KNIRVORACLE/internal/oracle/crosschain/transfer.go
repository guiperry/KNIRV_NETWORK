package crosschain

import (
	"time"

	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
)

// TransferStatus represents the status of a cross-chain transfer
type TransferStatus int

const (
	StatusPending TransferStatus = iota
	StatusSourceLocked
	StatusInTransit
	StatusDestReceived
	StatusCompleted
	StatusFailed
	StatusRefunded
	StatusTimedOut
)

// String returns the string representation of TransferStatus
func (s TransferStatus) String() string {
	switch s {
	case StatusPending:
		return "pending"
	case StatusSourceLocked:
		return "source_locked"
	case StatusInTransit:
		return "in_transit"
	case StatusDestReceived:
		return "dest_received"
	case StatusCompleted:
		return "completed"
	case StatusFailed:
		return "failed"
	case StatusRefunded:
		return "refunded"
	case StatusTimedOut:
		return "timed_out"
	default:
		return "unknown"
	}
}

// CrossChainTransfer represents a cross-chain transfer request
type CrossChainTransfer struct {
	TransferID       string         `json:"transfer_id"`
	SourceChain      types.ChainID  `json:"source_chain"`
	DestChain        types.ChainID  `json:"dest_chain"`
	Sender           string         `json:"sender"`
	Recipient        string         `json:"recipient"`
	Amount           uint64         `json:"amount"`
	Denom            string         `json:"denom"`
	TimeoutHeight    uint64         `json:"timeout_height"`
	TimeoutTimestamp uint64         `json:"timeout_timestamp"`
	Memo             string         `json:"memo,omitempty"`
	Status           TransferStatus `json:"status"`
	FeeAmount        uint64         `json:"fee_amount"`
	FeeDenom         string         `json:"fee_denom"`
	CreatedAt        int64          `json:"created_at"`
	CompletedAt      *int64         `json:"completed_at,omitempty"`
	Error            string         `json:"error,omitempty"`
	Proof            *TransferProof `json:"proof,omitempty"`
}

// TransferRequest represents a transfer initiation request
type TransferRequest struct {
	SourceChain      types.ChainID `json:"source_chain"`
	DestChain        types.ChainID `json:"dest_chain"`
	Sender           string        `json:"sender"`
	Recipient        string        `json:"recipient"`
	Amount           uint64        `json:"amount"`
	Denom            string        `json:"denom"`
	TimeoutHeight    uint64        `json:"timeout_height"`
	TimeoutTimestamp uint64        `json:"timeout_timestamp"`
	Memo             string        `json:"memo,omitempty"`
}

// TransferReceipt represents a transfer receipt
type TransferReceipt struct {
	TransferID      string        `json:"transfer_id"`
	SourceChain     types.ChainID `json:"source_chain"`
	DestChain       types.ChainID `json:"dest_chain"`
	Status          string        `json:"status"`
	FeeAmount       uint64        `json:"fee_amount"`
	FeeDenom        string        `json:"fee_denom"`
	TransactionHash string        `json:"transaction_hash"`
	Timestamp       int64         `json:"timestamp"`
}

// TransferProof represents proof of a cross-chain transfer
type TransferProof struct {
	MerkleRoot    string               `json:"merkle_root"`
	MerkleProof   []string             `json:"merkle_proof"`
	ValidatorSigs []ValidatorSignature `json:"validator_signatures"`
	BlockHeight   uint64               `json:"block_height"`
	BlockHash     string               `json:"block_hash"`
	Timestamp     int64                `json:"timestamp"`
}

// ValidatorSignature represents a validator's signature on a transfer
type ValidatorSignature struct {
	ValidatorAddress string `json:"validator_address"`
	Signature        string `json:"signature"`              // JSON-encoded canonical SignedMessage
	VotingPower      uint64 `json:"voting_power,omitempty"` // deprecated; power comes from the registered validator set
}

// TransferEvent represents an event in the transfer lifecycle
type TransferEvent struct {
	TransferID string            `json:"transfer_id"`
	EventType  TransferEventType `json:"event_type"`
	Status     TransferStatus    `json:"status"`
	Message    string            `json:"message,omitempty"`
	Timestamp  time.Time         `json:"timestamp"`
}

// TransferEventType represents the type of transfer event
type TransferEventType int

const (
	EventInitiated TransferEventType = iota
	EventLocked
	EventInTransit
	EventReceived
	EventCompleted
	EventFailed
	EventRefunded
	EventTimedOut
)

// String returns the string representation of TransferEventType
func (t TransferEventType) String() string {
	switch t {
	case EventInitiated:
		return "initiated"
	case EventLocked:
		return "locked"
	case EventInTransit:
		return "in_transit"
	case EventReceived:
		return "received"
	case EventCompleted:
		return "completed"
	case EventFailed:
		return "failed"
	case EventRefunded:
		return "refunded"
	case EventTimedOut:
		return "timed_out"
	default:
		return "unknown"
	}
}

// Validate validates the transfer request
func (r *TransferRequest) Validate() error {
	if r.Sender == "" {
		return types.ErrInvalidAddress
	}
	if r.Recipient == "" {
		return types.ErrInvalidAddress
	}
	if r.Amount == 0 {
		return types.ErrInvalidAmount
	}
	if r.Denom == "" {
		return types.ErrInvalidAmount
	}
	if r.TimeoutHeight == 0 && r.TimeoutTimestamp == 0 {
		return types.ErrTimeout
	}
	return nil
}

// IsTimedOut checks if a transfer has timed out
func (t *CrossChainTransfer) IsTimedOut(currentHeight uint64, currentTimestamp uint64) bool {
	if t.TimeoutHeight > 0 && currentHeight >= t.TimeoutHeight {
		return true
	}
	if t.TimeoutTimestamp > 0 && currentTimestamp >= t.TimeoutTimestamp {
		return true
	}
	return false
}

// CanBeRefunded checks if a transfer can be refunded
func (t *CrossChainTransfer) CanBeRefunded() bool {
	return t.Status == StatusFailed || t.Status == StatusTimedOut
}

// IsCompleted checks if a transfer is in a terminal state
func (t *CrossChainTransfer) IsCompleted() bool {
	return t.Status == StatusCompleted ||
		t.Status == StatusFailed ||
		t.Status == StatusRefunded ||
		t.Status == StatusTimedOut
}
