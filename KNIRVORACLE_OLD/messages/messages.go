package messages

import (
	"time"
)

// TickMsg represents a periodic tick message for updates
type TickMsg time.Time

// BlockchainEventMsg represents blockchain events
type BlockchainEventMsg struct {
	Type      string
	Data      interface{}
	Timestamp time.Time
}

// NetworkUpdateMsg represents network status updates
type NetworkUpdateMsg struct {
	PeerCount     int
	Latency       time.Duration
	UploadSpeed   float64 // KB/s
	DownloadSpeed float64 // KB/s
	Timestamp     time.Time
}

// MetricsUpdateMsg represents system metrics updates
type MetricsUpdateMsg struct {
	CPUUsage    float64
	MemoryUsage float64
	DiskUsage   float64
	Timestamp   time.Time
}

// AlertMsg represents system alerts
type AlertMsg struct {
	Level     AlertLevel
	Title     string
	Message   string
	Timestamp time.Time
}

// AlertLevel represents the severity of an alert
type AlertLevel int

const (
	InfoAlert AlertLevel = iota
	WarningAlert
	ErrorAlert
	CriticalAlert
)

// String returns the string representation of AlertLevel
func (a AlertLevel) String() string {
	switch a {
	case InfoAlert:
		return "INFO"
	case WarningAlert:
		return "WARNING"
	case ErrorAlert:
		return "ERROR"
	case CriticalAlert:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// TransactionMsg represents transaction-related messages
type TransactionMsg struct {
	Type        string
	Transaction Transaction
	Timestamp   time.Time
}

// Transaction represents a blockchain transaction
type Transaction struct {
	Hash      string
	From      string
	To        string
	Amount    float64
	Fee       float64
	Status    string
	Timestamp time.Time
}

// BlockMsg represents block-related messages
type BlockMsg struct {
	Type      string
	Block     Block
	Timestamp time.Time
}

// Block represents a blockchain block
type Block struct {
	Hash         string
	Number       int
	PreviousHash string
	Timestamp    time.Time
	TxCount      int
	Size         int64
	Miner        string
}

// WalletMsg represents wallet-related messages
type WalletMsg struct {
	Type      string
	Address   string
	Balance   float64
	Timestamp time.Time
}

// PeerMsg represents dev-related messages
type PeerMsg struct {
	Type      string
	PeerID    string
	Address   string
	Action    string // "connected", "disconnected", "updated"
	Timestamp time.Time
}

// CommandMsg represents terminal command messages
type CommandMsg struct {
	Command   string
	Args      []string
	Timestamp time.Time
}

// CommandResultMsg represents the result of a terminal command
type CommandResultMsg struct {
	Command   string
	Output    string
	Error     error
	Timestamp time.Time
}

// ViewSwitchMsg represents view switching messages
type ViewSwitchMsg struct {
	FromView int
	ToView   int
}

// ConfigUpdateMsg represents configuration update messages
type ConfigUpdateMsg struct {
	Key       string
	Value     interface{}
	Timestamp time.Time
}

// StatusUpdateMsg represents general status updates
type StatusUpdateMsg struct {
	Component string
	Status    string
	Details   map[string]interface{}
	Timestamp time.Time
}

// DataStreamMsg represents real-time data stream messages
type DataStreamMsg struct {
	StreamType string
	Data       interface{}
	Timestamp  time.Time
}

// ErrorMsg represents error messages
type ErrorMsg struct {
	Component string
	Error     error
	Context   map[string]interface{}
	Timestamp time.Time
}

// SuccessMsg represents success messages
type SuccessMsg struct {
	Component string
	Message   string
	Details   map[string]interface{}
	Timestamp time.Time
}

// InitCompleteMsg indicates that initialization is complete
type InitCompleteMsg struct {
	Component string
	Timestamp time.Time
}

// ShutdownMsg represents shutdown messages
type ShutdownMsg struct {
	Reason    string
	Timestamp time.Time
}

// ResizeMsg represents window resize messages
type ResizeMsg struct {
	Width  int
	Height int
}

// KeyPressMsg represents key press events
type KeyPressMsg struct {
	Key       string
	Modifiers []string
	Timestamp time.Time
}

// MouseMsg represents mouse events
type MouseMsg struct {
	X         int
	Y         int
	Button    string
	Action    string // "click", "drag", "scroll"
	Timestamp time.Time
}

// ThemeChangeMsg represents theme change messages
type ThemeChangeMsg struct {
	ThemeName string
	Timestamp time.Time
}

// PermissionMsg represents permission-related messages
type PermissionMsg struct {
	Action    string
	Resource  string
	Granted   bool
	Timestamp time.Time
}

// LogMsg represents log messages
type LogMsg struct {
	Level     string
	Component string
	Message   string
	Fields    map[string]interface{}
	Timestamp time.Time
}

// HealthCheckMsg represents health check messages
type HealthCheckMsg struct {
	Component string
	Healthy   bool
	Details   map[string]interface{}
	Timestamp time.Time
}

// PerformanceMsg represents performance metrics
type PerformanceMsg struct {
	Component string
	Metric    string
	Value     float64
	Unit      string
	Timestamp time.Time
}

// SecurityMsg represents security-related messages
type SecurityMsg struct {
	EventType string
	Severity  string
	Details   map[string]interface{}
	Timestamp time.Time
}

// BackupMsg represents backup-related messages
type BackupMsg struct {
	Type      string // "started", "completed", "failed"
	Path      string
	Size      int64
	Error     error
	Timestamp time.Time
}

// UpdateMsg represents software update messages
type UpdateMsg struct {
	Type      string // "available", "downloading", "installing", "completed"
	Version   string
	Progress  float64
	Error     error
	Timestamp time.Time
}

// ExternalLogMsg is a message to send externally captured log lines to the UI
type ExternalLogMsg struct {
	Timestamp time.Time
	Source    string
	Level     string
	Message   string
	Data      map[string]interface{}
}
