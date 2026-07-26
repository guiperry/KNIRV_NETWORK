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
	Mode      string `json:"mode"`       // "gateway", "standalone", "disabled"
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

// DiscoveryService handles peer discovery.
type DiscoveryService interface {
	FindPeers(ctx context.Context, networkID string) ([]PeerInfo, error)
	Announce(ctx context.Context, networkID string) error
	ResolvePeer(ctx context.Context, peerID string) (*PeerInfo, error)
}
