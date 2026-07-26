package p2pconsensus

import "time"

// PeerInfo represents a remote peer in the consensus network.
type PeerInfo struct {
	ID        string    `json:"id"`
	Addresses []string  `json:"addresses"`
	Latency   string    `json:"latency,omitempty"`
	LastSeen  time.Time `json:"last_seen,omitempty"`
}

// P2PMessage is the wire format for all P2P messages.
type P2PMessage struct {
	Type      string      `json:"type"`
	NetworkID string      `json:"network_id"`
	SenderID  string      `json:"sender_id"`
	Timestamp int64       `json:"timestamp"`
	Payload   interface{} `json:"payload"`
}

// SyncRequest asks a peer for operations newer than the given vector clock.
type SyncRequest struct {
	NetworkID   string           `json:"network_id"`
	Collection  string           `json:"collection"`
	VectorClock map[string]int64 `json:"vector_clock"`
	MaxEntries  int              `json:"max_entries,omitempty"`
}

// SyncResponse returns operations the requester is missing.
type SyncResponse struct {
	NetworkID     string              `json:"network_id"`
	Collection    string              `json:"collection"`
	Operations    []OperationEnvelope `json:"operations"`
	MoreAvailable bool                `json:"more_available"`
}

// OperationEnvelope wraps a serialized CRDT operation with metadata.
type OperationEnvelope struct {
	Collection  string           `json:"collection"`
	DocumentID  string           `json:"document_id"`
	Data        []byte           `json:"data"` // JSON-encoded CRDTOperation
	VectorClock map[string]int64 `json:"vector_clock"`
	Timestamp   int64            `json:"timestamp"`
	PeerID      string           `json:"peer_id"`
}

// MessageType constants
const (
	MsgOperation          = "operation"
	MsgSyncRequest        = "sync_request"
	MsgSyncResponse       = "sync_response"
	MsgHeartbeat          = "heartbeat"
	MsgCollectionAnnounce = "collection_announce"
)
