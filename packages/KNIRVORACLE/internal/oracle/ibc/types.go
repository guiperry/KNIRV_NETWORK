package ibc

import (
	"time"

	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
)

// ChannelState represents the state of an IBC channel
type ChannelState int

const (
	ChannelStateInit ChannelState = iota
	ChannelStateTryOpen
	ChannelStateOpen
	ChannelStateClosed
)

// String returns the string representation of ChannelState
func (s ChannelState) String() string {
	switch s {
	case ChannelStateInit:
		return "init"
	case ChannelStateTryOpen:
		return "try_open"
	case ChannelStateOpen:
		return "open"
	case ChannelStateClosed:
		return "closed"
	default:
		return "unknown"
	}
}

// ChannelOrdering represents the ordering type of a channel
type ChannelOrdering int

const (
	ChannelOrderingUnordered ChannelOrdering = iota
	ChannelOrderingOrdered
)

// String returns the string representation of ChannelOrdering
func (s ChannelOrdering) String() string {
	switch s {
	case ChannelOrderingUnordered:
		return "unordered"
	case ChannelOrderingOrdered:
		return "ordered"
	default:
		return "unknown"
	}
}

// IBCChannel represents an IBC channel
type IBCChannel struct {
	ChannelID             string          `json:"channel_id"`
	ConnectionID          string          `json:"connection_id"`
	PortID                string          `json:"port_id"`
	CounterpartyPortID    string          `json:"counterparty_port_id"`
	CounterpartyChannelID string          `json:"counterparty_channel_id"`
	State                 ChannelState    `json:"state"`
	Version               string          `json:"version"`
	Ordering              ChannelOrdering `json:"ordering"`
	ChainID               types.ChainID   `json:"chain_id"`
	CreatedAt             time.Time       `json:"created_at"`
	UpdatedAt             time.Time       `json:"updated_at"`
}

// ConnectionState represents the state of an IBC connection
type ConnectionState int

const (
	ConnectionStateInit ConnectionState = iota
	ConnectionStateTryOpen
	ConnectionStateOpen
	ConnectionStateClosed
)

// String returns the string representation of ConnectionState
func (s ConnectionState) String() string {
	switch s {
	case ConnectionStateInit:
		return "init"
	case ConnectionStateTryOpen:
		return "try_open"
	case ConnectionStateOpen:
		return "open"
	case ConnectionStateClosed:
		return "closed"
	default:
		return "unknown"
	}
}

// IBCConnection represents an IBC connection
type IBCConnection struct {
	ConnectionID       string          `json:"connection_id"`
	ClientID           string          `json:"client_id"`
	CounterpartyConnID string          `json:"counterparty_connection_id"`
	CounterpartyClient string          `json:"counterparty_client_id"`
	State              ConnectionState `json:"state"`
	DelayPeriod        uint64          `json:"delay_period"`
	ChainID            types.ChainID   `json:"chain_id"`
	Endpoint           string          `json:"endpoint"` // gRPC or HTTP endpoint
	Transport          TransportType   `json:"transport"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
}

// TransportType represents the transport protocol for IBC
type TransportType int

const (
	TransportGRPC TransportType = iota
	TransportHTTP
	TransportWebSocket
)

// String returns the string representation of TransportType
func (t TransportType) String() string {
	switch t {
	case TransportGRPC:
		return "grpc"
	case TransportHTTP:
		return "http"
	case TransportWebSocket:
		return "websocket"
	default:
		return "unknown"
	}
}

// ClientState represents the state of a light client
type ClientState int

const (
	ClientStateActive ClientState = iota
	ClientStateFrozen
	ClientStateExpired
)

// String returns the string representation of ClientState
func (s ClientState) String() string {
	switch s {
	case ClientStateActive:
		return "active"
	case ClientStateFrozen:
		return "frozen"
	case ClientStateExpired:
		return "expired"
	default:
		return "unknown"
	}
}

// IBCClient represents a light client
type IBCClient struct {
	ClientID        string        `json:"client_id"`
	ClientType      string        `json:"client_type"` // "07-tendermint", "08-wasm", etc.
	ChainID         types.ChainID `json:"chain_id"`
	State           ClientState   `json:"state"`
	LatestHeight    uint64        `json:"latest_height"`
	TrustingPeriod  uint64        `json:"trusting_period"`  // seconds
	UnbondingPeriod uint64        `json:"unbonding_period"` // seconds
	MaxClockDrift   uint64        `json:"max_clock_drift"`  // seconds
	FrozenHeight    uint64        `json:"frozen_height,omitempty"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
}

// IBCMessage represents different types of IBC messages
type IBCMessage struct {
	Type      MessageType            `json:"type"`
	Data      map[string]interface{} `json:"data"`
	Timestamp int64                  `json:"timestamp"`
	Sender    string                 `json:"sender"`
}

// MessageType represents the type of IBC message
type MessageType int

const (
	MessageTypeNRNBurn MessageType = iota
	MessageTypeSkillRegistration
	MessageTypeModelTransitionUpdate
	MessageTypeProofVerificationRequest
	MessageTypeProofVerificationResponse
	MessageTypeStateSync
	MessageTypeNetworkStatus
	MessageTypeCrossChainTransfer
	MessageTypeGovernanceProposal
	MessageTypeValidatorUpdate
	MessageTypeEmergencyHalt
)

// String returns the string representation of MessageType
func (t MessageType) String() string {
	switch t {
	case MessageTypeNRNBurn:
		return "nrn_burn"
	case MessageTypeSkillRegistration:
		return "skill_registration"
	case MessageTypeModelTransitionUpdate:
		return "model_transition_update"
	case MessageTypeProofVerificationRequest:
		return "proof_verification_request"
	case MessageTypeProofVerificationResponse:
		return "proof_verification_response"
	case MessageTypeStateSync:
		return "state_sync"
	case MessageTypeNetworkStatus:
		return "network_status"
	case MessageTypeCrossChainTransfer:
		return "cross_chain_transfer"
	case MessageTypeGovernanceProposal:
		return "governance_proposal"
	case MessageTypeValidatorUpdate:
		return "validator_update"
	case MessageTypeEmergencyHalt:
		return "emergency_halt"
	default:
		return "unknown"
	}
}

// IBCPacket represents an IBC packet
type IBCPacket struct {
	Sequence         uint64        `json:"sequence"`
	SourcePort       string        `json:"source_port"`
	SourceChannel    string        `json:"source_channel"`
	DestPort         string        `json:"dest_port"`
	DestChannel      string        `json:"dest_channel"`
	DestChainID      types.ChainID `json:"dest_chain_id"`
	Data             []byte        `json:"data"`
	TimeoutHeight    uint64        `json:"timeout_height"`
	TimeoutTimestamp uint64        `json:"timeout_timestamp"`
	CreatedAt        time.Time     `json:"created_at"`
}

// PacketAcknowledgement represents an acknowledgement of a packet
type PacketAcknowledgement struct {
	PacketSequence uint64    `json:"packet_sequence"`
	Success        bool      `json:"success"`
	Data           []byte    `json:"data"`
	Error          string    `json:"error,omitempty"`
	Timestamp      time.Time `json:"timestamp"`
}

// PendingPacket represents a packet waiting to be processed
type PendingPacket struct {
	Packet     *IBCPacket `json:"packet"`
	Retries    int        `json:"retries"`
	LastRetry  time.Time  `json:"last_retry"`
	MaxRetries int        `json:"max_retries"`
}
