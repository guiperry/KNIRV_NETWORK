package p2pconsensus

import "context"

// ConsensusManager is the top-level consensus orchestrator.
type ConsensusManager interface {
	Start(ctx context.Context) error
	Stop() error
	Status() ConsensusStatus
	Peers() []PeerInfo
	Enabled() bool
	SetEnabled(enabled bool)
	PublishOperation(ctx context.Context, op OperationEnvelope) error
}

// ConsensusStatus represents the current state of the consensus manager.
type ConsensusStatus struct {
	Mode      string `json:"mode"` // "gateway", "standalone", "disabled"
	NetworkID string `json:"network_id"`
	PeerCount int    `json:"peer_count"`
	Running   bool   `json:"running"`
}

// EventHandler processes incoming consensus events.
type EventHandler interface {
	OnOperationReceived(op OperationEnvelope) error
	OnSyncRequestReceived(req SyncRequest) (*SyncResponse, error)
	OnPeerDiscovered(peer PeerInfo) error
}

// SyncResponseHandler is optional. Managers that receive a direct sync
// response call this method; keeping it separate preserves existing handlers.
type SyncResponseHandler interface {
	OnSyncResponseReceived(resp SyncResponse) error
}

// SyncRequester is implemented by managers that can request the current
// operation set from peers. It is intentionally separate from
// ConsensusManager so existing callers can provide lightweight test managers.
type SyncRequester interface {
	RequestSync(ctx context.Context, req SyncRequest) error
}

// DiscoveryService handles peer discovery.
type DiscoveryService interface {
	FindPeers(ctx context.Context, networkID string) ([]PeerInfo, error)
	Announce(ctx context.Context, networkID string) error
	ResolvePeer(ctx context.Context, peerID string) (*PeerInfo, error)
}
