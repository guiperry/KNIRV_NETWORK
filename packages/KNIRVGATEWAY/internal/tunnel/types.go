package tunnel

import (
	"net"
	"sync"
	"time"
)

// NodeInfo represents information about a registered node
type NodeInfo struct {
	DevID            string    `json:"devId"`
	ChainID          string    `json:"chainId,omitempty"`
	InternalIP       string    `json:"internalIp,omitempty"`
	InternalP2PPort  int       `json:"internalP2pPort,omitempty"`
	PublicIP         string    `json:"publicIp,omitempty"`
	PublicP2PPort    int       `json:"publicP2pPort,omitempty"`
	Type             string    `json:"type"`
	LastSeen         time.Time `json:"lastSeen"`
	ControlSocketID  string    `json:"controlSocketId,omitempty"`
	IsTunneled       bool      `json:"isTunneled"`
	IsBootnode       bool      `json:"isBootnode,omitempty"`
	PublicRelayURL  string    `json:"publicRelayUrl,omitempty"`
}

// ConnectionInfo represents information about an active connection
type ConnectionInfo struct {
	ID         string    `json:"id"`
	Type       string    `json:"type"`
	SourceIP   string    `json:"sourceIp"`
	SourcePort int       `json:"sourcePort"`
	LastSeen   time.Time `json:"lastSeen"`
	Socket     net.Conn  `json:"-"`
}

// URIGenerateRequest represents a request to generate a URI
type URIGenerateRequest struct {
	DevID        string `json:"devId"`
	ResourceType string `json:"resourceType,omitempty"`
	SubPath      string `json:"subPath,omitempty"`
}

// URIGenerateResponse represents the response from URI generation
type URIGenerateResponse struct {
	URI          string                `json:"uri"`
	ResourceID   string                `json:"resourceId"`
	ResourceType string                `json:"resourceType"`
	SubPath      string                `json:"subPath"`
	DirectInfo   *DirectConnectionInfo `json:"directInfo,omitempty"`
}

// DirectConnectionInfo represents direct connection information
type DirectConnectionInfo struct {
	IP   string `json:"ip"`
	Port int    `json:"port"`
}

// URIResolveRequest represents a request to resolve a URI
type URIResolveRequest struct {
	URI string `json:"uri"`
}

// URIResolveResponse represents the response from URI resolution
type URIResolveResponse struct {
	OriginalURI        string `json:"originalUri"`
	ResolvedIdentifier string `json:"resolvedIdentifier"`
	ResourceType       string `json:"resourceType"`
	SubPathWithQuery   string `json:"subPathWithQuery"`
	Authority          string `json:"authority"`
	ConnectionType     string `json:"connectionType,omitempty"`
	TargetPeerID       string `json:"targetPeerId,omitempty"`
	Multiaddress       string `json:"multiaddress,omitempty"`
	TunnelServerHost   string `json:"tunnelServerHost,omitempty"`
	TunnelServerPort   int    `json:"tunnelServerPort,omitempty"`
	RelayProtocolInfo  string `json:"relayProtocolInfo,omitempty"`
}

// ControlMessage represents messages sent over the control channel
type ControlMessage struct {
	Action          string `json:"action"`
	DevID           string `json:"devId,omitempty"`
	ChainID         string `json:"chainId,omitempty"`
	InternalIP      string `json:"internalIp,omitempty"`
	InternalP2PPort int    `json:"internalP2pPort,omitempty"`
	Type            string `json:"type,omitempty"`
	Timestamp       int64  `json:"timestamp,omitempty"`
	Status          string `json:"status,omitempty"`
	Message         string `json:"message,omitempty"`
}

// RelayMessage represents messages for relay setup
type RelayMessage struct {
	TargetPeerID string `json:"targetPeerId"`
}

// StatusResponse represents the status endpoint response
type StatusResponse struct {
	Status                string                 `json:"status"`
	Version               string                 `json:"version"`
	Uptime                string                 `json:"uptime"`
	UptimeMs              int64                  `json:"uptimeMs"`
	RegisteredConnections int                    `json:"registeredConnections"`
	ActiveConnections     int                    `json:"activeConnections"`
	MemoryUsage           MemoryUsage            `json:"memoryUsage"`
	HTTPAPIPort           int                    `json:"httpApiPort"`
	ControlListenerPort   int                    `json:"controlListenerPort"`
	PublicRelayPort       int                    `json:"publicRelayPort"`
	Timestamp             string                 `json:"timestamp"`
	Connections           map[string]interface{} `json:"connections,omitempty"`
}

// MemoryUsage represents memory usage information
type MemoryUsage struct {
	RSS       uint64 `json:"rss"`
	HeapTotal uint64 `json:"heapTotal"`
	HeapUsed  uint64 `json:"heapUsed"`
}

// Global connection registry (similar to Node.js global)
var (
	connectionRegistry   = make(map[string]*ConnectionInfo)
	connectionRegistryMu sync.RWMutex
)
