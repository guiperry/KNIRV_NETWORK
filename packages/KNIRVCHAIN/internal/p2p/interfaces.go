package p2p

import (
	"context"
	"net"
	"time"
)

// Types used by the ConsensusManager interface are defined later in this file

// ConsensusManager defines the interface for P2P consensus operations
type ConsensusManager interface {
	// Consensus operations
	StartConsensus(ctx context.Context) error
	StopConsensus() error
	ProposeValue(value interface{}) error
	VoteOnProposal(proposalID string, vote bool) error
	GetConsensusState() ConsensusState

	// Participant management
	AddParticipant(peerID string) error
	RemoveParticipant(peerID string) error
	GetParticipants() []string

	// Event handling
	OnConsensusReached(handler ConsensusHandler) error
	OnProposalReceived(handler ProposalHandler) error

	// Mining state management
	GetMiningLockState() bool
	GetUpdateRequired() bool

	// Runtime enable/disable
	SetEnabled(bool)
}

// DiscoveryService interface removed to avoid conflict with discovery_interface.go

// DelegationHandler defines the interface for delegation operations
type DelegationHandler interface {
	// Delegation operations
	HandleDelegation(delegation *Delegation) error
	ProcessDelegationRequest(request *DelegationRequest) (*DelegationResponse, error)
	ValidateDelegation(delegation *Delegation) error

	// Delegation management
	GetActiveDelegations() []*Delegation
	GetDelegationByID(delegationID string) (*Delegation, error)
	RevokeDelegation(delegationID string) error

	// Authority management
	GrantAuthority(peerID string, authority Authority) error
	RevokeAuthority(peerID string, authority Authority) error
	CheckAuthority(peerID string, authority Authority) bool
}

// StatusHandler defines the interface for peer status management
type StatusHandler interface {
	// Status operations
	UpdateStatus(status PeerStatus) error
	GetStatus() PeerStatus
	BroadcastStatus() error

	// Status monitoring
	MonitorPeerStatus(peerID string) error
	GetPeerStatuses() map[string]PeerStatus
	OnStatusChange(handler StatusChangeHandler) error

	// Health monitoring
	StartHealthCheck(ctx context.Context) error
	StopHealthCheck() error
	GetHealthMetrics() HealthMetrics
}

// SyncManagerInterface defines the interface for data synchronization (renamed to avoid conflict)
type SyncManagerInterface interface {
	// Sync operations
	StartSync(ctx context.Context) error
	StopSync() error
	SyncWithPeer(peerID string) error
	SyncData(dataType string) error

	// Sync state management
	GetSyncState() SyncState
	IsSyncing() bool
	GetSyncProgress() float64

	// Conflict resolution
	ResolveConflict(conflict *SyncConflict) error
	OnConflictDetected(handler ConflictHandler) error
}

// NetworkManager defines the interface for network management
type NetworkManager interface {
	// Network operations
	Start(ctx context.Context) error
	Stop() error
	Connect(address string) error
	Disconnect(peerID string) error

	// Message handling
	SendMessage(peerID string, message *Message) error
	BroadcastMessage(message *Message) error
	OnMessageReceived(handler MessageHandler) error

	// Connection management
	GetConnections() []Connection
	GetConnectionCount() int
	IsConnected(peerID string) bool
}

// PeerInfo represents information about a peer
type PeerInfo struct {
	ID        string                 `json:"id"`
	Address   string                 `json:"address"`
	Port      int                    `json:"port"`
	PublicKey string                 `json:"public_key"`
	LastSeen  time.Time              `json:"last_seen"`
	Version   string                 `json:"version"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// PeerStatus represents the status of a peer
type PeerStatus struct {
	PeerID       string                 `json:"peer_id"`
	Online       bool                   `json:"online"`
	LastUpdate   time.Time              `json:"last_update"`
	BlockHeight  uint64                 `json:"block_height"`
	Latency      time.Duration          `json:"latency"`
	Bandwidth    BandwidthMetrics       `json:"bandwidth"`
	Capabilities []string               `json:"capabilities"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// BandwidthMetrics represents bandwidth usage metrics
type BandwidthMetrics struct {
	Upload   uint64 `json:"upload"`   // bytes per second
	Download uint64 `json:"download"` // bytes per second
}

// ConsensusState represents the current consensus state
type ConsensusState struct {
	Round        uint64          `json:"round"`
	Phase        string          `json:"phase"`
	Proposer     string          `json:"proposer"`
	Proposal     interface{}     `json:"proposal,omitempty"`
	Votes        map[string]bool `json:"votes"`
	Participants []string        `json:"participants"`
	Timestamp    time.Time       `json:"timestamp"`
}

// Delegation represents a delegation of authority
type Delegation struct {
	ID        string    `json:"id"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	Authority Authority `json:"authority"`
	Expiry    time.Time `json:"expiry"`
	Signature string    `json:"signature"`
	CreatedAt time.Time `json:"created_at"`
}

// DelegationRequest represents a request for delegation
type DelegationRequest struct {
	RequestID string        `json:"request_id"`
	From      string        `json:"from"`
	To        string        `json:"to"`
	Authority Authority     `json:"authority"`
	Duration  time.Duration `json:"duration"`
	Reason    string        `json:"reason,omitempty"`
}

// DelegationResponse represents a response to a delegation request
type DelegationResponse struct {
	RequestID  string      `json:"request_id"`
	Approved   bool        `json:"approved"`
	Delegation *Delegation `json:"delegation,omitempty"`
	Reason     string      `json:"reason,omitempty"`
}

// Authority represents different types of authority that can be delegated
type Authority struct {
	Type        string   `json:"type"`
	Permissions []string `json:"permissions"`
	Scope       string   `json:"scope,omitempty"`
}

// SyncState represents the current synchronization state
type SyncState struct {
	Syncing    bool      `json:"syncing"`
	Progress   float64   `json:"progress"`
	StartTime  time.Time `json:"start_time"`
	LastUpdate time.Time `json:"last_update"`
	PeerCount  int       `json:"peer_count"`
	DataTypes  []string  `json:"data_types"`
}

// SyncConflict represents a synchronization conflict
type SyncConflict struct {
	ID          string      `json:"id"`
	DataType    string      `json:"data_type"`
	LocalValue  interface{} `json:"local_value"`
	RemoteValue interface{} `json:"remote_value"`
	PeerID      string      `json:"peer_id"`
	Timestamp   time.Time   `json:"timestamp"`
}

// Message represents a P2P message
type Message struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	From      string                 `json:"from"`
	To        string                 `json:"to,omitempty"`
	Payload   map[string]interface{} `json:"payload"`
	Timestamp time.Time              `json:"timestamp"`
	Signature string                 `json:"signature,omitempty"`
}

// Connection represents a P2P connection
type Connection struct {
	PeerID      string        `json:"peer_id"`
	Address     string        `json:"address"`
	Connected   bool          `json:"connected"`
	ConnectedAt time.Time     `json:"connected_at"`
	LastPing    time.Time     `json:"last_ping"`
	Latency     time.Duration `json:"latency"`
	Conn        net.Conn      `json:"-"`
}

// HealthMetrics represents health monitoring metrics
type HealthMetrics struct {
	ActiveConnections int           `json:"active_connections"`
	MessagesSent      uint64        `json:"messages_sent"`
	MessagesReceived  uint64        `json:"messages_received"`
	AverageLatency    time.Duration `json:"average_latency"`
	ErrorCount        uint64        `json:"error_count"`
	LastHealthCheck   time.Time     `json:"last_health_check"`
}

// Event handler function types
type ConsensusHandler func(state ConsensusState) error
type ProposalHandler func(proposal interface{}) error
type StatusChangeHandler func(peerID string, oldStatus, newStatus PeerStatus) error
type ConflictHandler func(conflict *SyncConflict) error
type MessageHandler func(message *Message) error

// P2PConfig represents P2P network configuration
type P2PConfig struct {
	ListenAddress       string        `json:"listen_address"`
	Port                int           `json:"port"`
	MaxPeers            int           `json:"max_peers"`
	BootstrapPeers      []string      `json:"bootstrap_peers"`
	DiscoveryEnabled    bool          `json:"discovery_enabled"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	SyncInterval        time.Duration `json:"sync_interval"`
	MessageTimeout      time.Duration `json:"message_timeout"`
}

// Error types for P2P operations
var (
	ErrPeerNotFound     = NewP2PError("peer not found")
	ErrConnectionFailed = NewP2PError("connection failed")
	ErrConsensusTimeout = NewP2PError("consensus timeout")
	ErrInvalidMessage   = NewP2PError("invalid message")
	ErrSyncFailed       = NewP2PError("synchronization failed")
	ErrDelegationDenied = NewP2PError("delegation denied")
)

// P2PError represents a P2P-specific error
type P2PError struct {
	Message string
	Code    string
}

func (e *P2PError) Error() string {
	return e.Message
}

func NewP2PError(message string) *P2PError {
	return &P2PError{Message: message}
}
