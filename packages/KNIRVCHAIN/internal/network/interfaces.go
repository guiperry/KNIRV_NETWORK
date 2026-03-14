package network

import (
	"context"
	"net"
	"time"
)

// TunnelManager defines the interface for tunnel management
type TunnelManager interface {
	// Tunnel lifecycle
	CreateTunnel(config *TunnelConfig) (*Tunnel, error)
	DestroyTunnel(tunnelID string) error
	GetTunnel(tunnelID string) (*Tunnel, error)
	ListTunnels() ([]*Tunnel, error)
	
	// Tunnel operations
	OpenTunnel(tunnelID string) error
	CloseTunnel(tunnelID string) error
	IsTunnelOpen(tunnelID string) bool
	
	// Data transmission
	SendData(tunnelID string, data []byte) error
	ReceiveData(tunnelID string) ([]byte, error)
	
	// Lifecycle
	Start(ctx context.Context) error
	Stop() error
}

// TunnelClient defines the interface for tunnel client operations
type TunnelClient interface {
	// Connection management
	Connect(address string, config *ClientConfig) error
	Disconnect() error
	IsConnected() bool
	
	// Communication
	Send(data []byte) error
	Receive() ([]byte, error)
	SendMessage(message *TunnelMessage) error
	ReceiveMessage() (*TunnelMessage, error)
	
	// Event handling
	OnConnected(handler ConnectionHandler) error
	OnDisconnected(handler DisconnectionHandler) error
	OnDataReceived(handler DataHandler) error
	OnError(handler ErrorHandler) error
	
	// Health monitoring
	Ping() error
	GetConnectionStatus() *ConnectionStatus
}

// NetworkMonitor defines the interface for network monitoring
type NetworkMonitor interface {
	// Monitoring operations
	StartMonitoring(ctx context.Context) error
	StopMonitoring() error
	IsMonitoring() bool
	
	// Metrics collection
	GetNetworkMetrics() (*NetworkMetrics, error)
	GetInterfaceMetrics(interfaceName string) (*InterfaceMetrics, error)
	GetConnectionMetrics() (*ConnectionMetrics, error)
	
	// Health checking
	CheckConnectivity(target string) (*ConnectivityResult, error)
	CheckLatency(target string) (time.Duration, error)
	CheckBandwidth(target string) (*BandwidthResult, error)
	
	// Event handling
	OnNetworkChange(handler NetworkChangeHandler) error
	OnConnectivityLoss(handler ConnectivityHandler) error
	OnHighLatency(handler LatencyHandler) error
}

// Tunnel represents a network tunnel
type Tunnel struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	LocalAddr   string                 `json:"local_addr"`
	RemoteAddr  string                 `json:"remote_addr"`
	Status      string                 `json:"status"`
	CreatedAt   time.Time              `json:"created_at"`
	LastUsed    time.Time              `json:"last_used"`
	BytesSent   uint64                 `json:"bytes_sent"`
	BytesRecv   uint64                 `json:"bytes_recv"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// TunnelConfig represents tunnel configuration
type TunnelConfig struct {
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	LocalAddr   string                 `json:"local_addr"`
	RemoteAddr  string                 `json:"remote_addr"`
	Protocol    string                 `json:"protocol"`
	Encryption  bool                   `json:"encryption"`
	Compression bool                   `json:"compression"`
	Timeout     time.Duration          `json:"timeout"`
	KeepAlive   time.Duration          `json:"keep_alive"`
	Options     map[string]interface{} `json:"options,omitempty"`
}

// ClientConfig represents client configuration
type ClientConfig struct {
	Protocol      string                 `json:"protocol"`
	Timeout       time.Duration          `json:"timeout"`
	RetryAttempts int                    `json:"retry_attempts"`
	RetryDelay    time.Duration          `json:"retry_delay"`
	KeepAlive     bool                   `json:"keep_alive"`
	Compression   bool                   `json:"compression"`
	Encryption    bool                   `json:"encryption"`
	Options       map[string]interface{} `json:"options,omitempty"`
}

// TunnelMessage represents a message sent through a tunnel
type TunnelMessage struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Payload   []byte                 `json:"payload"`
	Headers   map[string]string      `json:"headers"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// ConnectionStatus represents the status of a connection
type ConnectionStatus struct {
	Connected    bool          `json:"connected"`
	ConnectedAt  time.Time     `json:"connected_at"`
	LastActivity time.Time     `json:"last_activity"`
	BytesSent    uint64        `json:"bytes_sent"`
	BytesRecv    uint64        `json:"bytes_recv"`
	Latency      time.Duration `json:"latency"`
	Error        string        `json:"error,omitempty"`
}

// NetworkMetrics represents overall network metrics
type NetworkMetrics struct {
	Interfaces      []*InterfaceMetrics `json:"interfaces"`
	TotalConnections int                `json:"total_connections"`
	ActiveTunnels   int                 `json:"active_tunnels"`
	TotalBandwidth  *BandwidthMetrics   `json:"total_bandwidth"`
	PacketLoss      float64             `json:"packet_loss"`
	AverageLatency  time.Duration       `json:"average_latency"`
	LastUpdated     time.Time           `json:"last_updated"`
}

// InterfaceMetrics represents metrics for a network interface
type InterfaceMetrics struct {
	Name         string            `json:"name"`
	Type         string            `json:"type"`
	Status       string            `json:"status"`
	MTU          int               `json:"mtu"`
	Speed        uint64            `json:"speed"`
	Bandwidth    *BandwidthMetrics `json:"bandwidth"`
	PacketStats  *PacketStats      `json:"packet_stats"`
	ErrorStats   *ErrorStats       `json:"error_stats"`
	LastUpdated  time.Time         `json:"last_updated"`
}

// ConnectionMetrics represents connection-related metrics
type ConnectionMetrics struct {
	ActiveConnections int                    `json:"active_connections"`
	TotalConnections  int64                  `json:"total_connections"`
	FailedConnections int64                  `json:"failed_connections"`
	ConnectionsByType map[string]int         `json:"connections_by_type"`
	AverageLatency    time.Duration          `json:"average_latency"`
	Throughput        *BandwidthMetrics      `json:"throughput"`
	LastUpdated       time.Time              `json:"last_updated"`
}

// BandwidthMetrics represents bandwidth metrics
type BandwidthMetrics struct {
	Upload   uint64 `json:"upload"`   // bytes per second
	Download uint64 `json:"download"` // bytes per second
	Peak     uint64 `json:"peak"`     // peak bandwidth
	Average  uint64 `json:"average"`  // average bandwidth
}

// PacketStats represents packet statistics
type PacketStats struct {
	Sent     uint64 `json:"sent"`
	Received uint64 `json:"received"`
	Dropped  uint64 `json:"dropped"`
	Errors   uint64 `json:"errors"`
}

// ErrorStats represents error statistics
type ErrorStats struct {
	TotalErrors    uint64 `json:"total_errors"`
	TimeoutErrors  uint64 `json:"timeout_errors"`
	NetworkErrors  uint64 `json:"network_errors"`
	ProtocolErrors uint64 `json:"protocol_errors"`
	LastError      string `json:"last_error,omitempty"`
	LastErrorTime  time.Time `json:"last_error_time"`
}

// ConnectivityResult represents the result of a connectivity check
type ConnectivityResult struct {
	Target      string        `json:"target"`
	Reachable   bool          `json:"reachable"`
	Latency     time.Duration `json:"latency"`
	PacketLoss  float64       `json:"packet_loss"`
	Error       string        `json:"error,omitempty"`
	Timestamp   time.Time     `json:"timestamp"`
}

// BandwidthResult represents the result of a bandwidth test
type BandwidthResult struct {
	Target      string            `json:"target"`
	Upload      uint64            `json:"upload"`
	Download    uint64            `json:"download"`
	Duration    time.Duration     `json:"duration"`
	Error       string            `json:"error,omitempty"`
	Timestamp   time.Time         `json:"timestamp"`
}

// NetworkEvent represents network events
type NetworkEvent struct {
	Type        string                 `json:"type"`
	Interface   string                 `json:"interface,omitempty"`
	Connection  string                 `json:"connection,omitempty"`
	Description string                 `json:"description"`
	Severity    string                 `json:"severity"`
	Data        map[string]interface{} `json:"data,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
}

// Event handler function types
type ConnectionHandler func(conn net.Conn) error
type DisconnectionHandler func(reason string) error
type DataHandler func(data []byte) error
type ErrorHandler func(error) error
type NetworkChangeHandler func(event *NetworkEvent) error
type ConnectivityHandler func(target string, lost bool) error
type LatencyHandler func(target string, latency time.Duration) error

// NetworkConfig represents network configuration
type NetworkConfig struct {
	ListenAddress     string        `json:"listen_address"`
	Port              int           `json:"port"`
	MaxConnections    int           `json:"max_connections"`
	ConnectionTimeout time.Duration `json:"connection_timeout"`
	ReadTimeout       time.Duration `json:"read_timeout"`
	WriteTimeout      time.Duration `json:"write_timeout"`
	KeepAlive         bool          `json:"keep_alive"`
	KeepAliveInterval time.Duration `json:"keep_alive_interval"`
	BufferSize        int           `json:"buffer_size"`
	EnableTLS         bool          `json:"enable_tls"`
	TLSCertFile       string        `json:"tls_cert_file,omitempty"`
	TLSKeyFile        string        `json:"tls_key_file,omitempty"`
}

// TunnelRegistry defines the interface for tunnel registry operations
type TunnelRegistry interface {
	// Registry operations
	RegisterTunnel(tunnel *TunnelRegistration) error
	UnregisterTunnel(tunnelID string) error
	GetTunnelInfo(tunnelID string) (*TunnelInfo, error)
	ListTunnels() ([]*TunnelInfo, error)
	
	// Discovery operations
	DiscoverTunnels(criteria *DiscoveryCriteria) ([]*TunnelInfo, error)
	FindTunnelsByType(tunnelType string) ([]*TunnelInfo, error)
	FindTunnelsByLocation(location string) ([]*TunnelInfo, error)
	
	// Health monitoring
	UpdateTunnelHealth(tunnelID string, health *HealthStatus) error
	GetTunnelHealth(tunnelID string) (*HealthStatus, error)
	GetHealthyTunnels() ([]*TunnelInfo, error)
}

// TunnelRegistration represents tunnel registration information
type TunnelRegistration struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Endpoint    string                 `json:"endpoint"`
	Capabilities []string              `json:"capabilities"`
	Location    string                 `json:"location,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	RegisteredAt time.Time             `json:"registered_at"`
}

// TunnelInfo represents information about a registered tunnel
type TunnelInfo struct {
	Registration *TunnelRegistration `json:"registration"`
	Health       *HealthStatus       `json:"health"`
	Stats        *TunnelStats        `json:"stats"`
	LastSeen     time.Time           `json:"last_seen"`
}

// DiscoveryCriteria represents criteria for tunnel discovery
type DiscoveryCriteria struct {
	Type         string   `json:"type,omitempty"`
	Location     string   `json:"location,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
	HealthyOnly  bool     `json:"healthy_only"`
	Limit        int      `json:"limit,omitempty"`
}

// HealthStatus represents the health status of a tunnel
type HealthStatus struct {
	Healthy      bool          `json:"healthy"`
	LastCheck    time.Time     `json:"last_check"`
	ResponseTime time.Duration `json:"response_time"`
	Error        string        `json:"error,omitempty"`
	Uptime       time.Duration `json:"uptime"`
}

// TunnelStats represents statistics for a tunnel
type TunnelStats struct {
	ConnectionCount   int64     `json:"connection_count"`
	ActiveConnections int       `json:"active_connections"`
	BytesTransferred  uint64    `json:"bytes_transferred"`
	MessagesProcessed int64     `json:"messages_processed"`
	ErrorCount        int64     `json:"error_count"`
	LastActivity      time.Time `json:"last_activity"`
}

// Error types for network operations
var (
	ErrTunnelNotFound     = NewNetworkError("tunnel not found")
	ErrConnectionFailed   = NewNetworkError("connection failed")
	ErrTunnelExists       = NewNetworkError("tunnel already exists")
	ErrInvalidAddress     = NewNetworkError("invalid address")
	ErrNetworkTimeout     = NewNetworkError("network timeout")
	ErrTunnelClosed       = NewNetworkError("tunnel closed")
	ErrRegistrationFailed = NewNetworkError("registration failed")
)

// NetworkError represents a network-specific error
type NetworkError struct {
	Message string
	Code    string
}

func (e *NetworkError) Error() string {
	return e.Message
}

func NewNetworkError(message string) *NetworkError {
	return &NetworkError{Message: message}
}
