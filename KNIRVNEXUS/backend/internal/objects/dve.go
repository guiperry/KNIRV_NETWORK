package objects

import (
	"time"
)

// DVENode represents a DVE (Decentralized Validation Environment) node
type DVENode struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Status          string    `json:"status"`   // "online", "offline", "maintenance", "error"
	TEEType         string    `json:"tee_type"` // "sgx", "sev-snp", "tdx", "software"
	StakeAmount     int64     `json:"stake_amount"`
	ReputationScore int       `json:"reputation_score"`
	Location        string    `json:"location"`
	IPAddress       string    `json:"ip_address"`
	PublicKey       string    `json:"public_key"`
	Capabilities    []string  `json:"capabilities"`
	LastHeartbeat   time.Time `json:"last_heartbeat"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	// Performance metrics
	CPUUsage       float64 `json:"cpu_usage"`
	MemoryUsage    float64 `json:"memory_usage"`
	NetworkLatency int64   `json:"network_latency"`

	// Geographic coordinates for spatial indexing
	Latitude  float64 `json:"latitude,omitempty"`
	Longitude float64 `json:"longitude,omitempty"`
}

// ValidationTask represents a validation task in the DVE network
type ValidationTask struct {
	ID              string                 `json:"id"`
	Type            string                 `json:"type"`     // "skillnode", "base_llm", "custom"
	Status          string                 `json:"status"`   // "pending", "assigned", "running", "completed", "failed"
	Priority        int                    `json:"priority"` // 1-10, higher is more urgent
	SkillCode       string                 `json:"skill_code,omitempty"`
	FailureContext  string                 `json:"failure_context,omitempty"`
	TestCases       []TestCase             `json:"test_cases"`
	RequiredTEEType string                 `json:"required_tee_type"`
	AssignedNodeID  string                 `json:"assigned_node_id,omitempty"`
	RequestedBy     string                 `json:"requested_by"`
	Parameters      map[string]interface{} `json:"parameters"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	StartedAt       *time.Time             `json:"started_at,omitempty"`
	CompletedAt     *time.Time             `json:"completed_at,omitempty"`
	TimeoutAt       time.Time              `json:"timeout_at"`
}

// TestCase represents a test case for validation
type TestCase struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Input       string  `json:"input"`
	Expected    string  `json:"expected"`
	Weight      float64 `json:"weight"` // Importance weight for scoring
}

// ValidationResult represents the result of a validation task
type ValidationResult struct {
	ID              string                 `json:"id"`
	TaskID          string                 `json:"task_id"`
	ValidatorNodeID string                 `json:"validator_node_id"`
	Status          string                 `json:"status"` // "success", "failure", "error"
	Score           float64                `json:"score"`  // 0.0 - 1.0
	Results         map[string]interface{} `json:"results"`
	TestResults     []TestResult           `json:"test_results"`
	Proof           string                 `json:"proof"` // Cryptographic proof
	TEEAttestation  string                 `json:"tee_attestation"`
	ExecutionTime   time.Duration          `json:"execution_time"`
	ErrorMessage    string                 `json:"error_message,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	Signature       string                 `json:"signature"`
	Metrics         *ValidationMetrics     `json:"metrics,omitempty"`
}

// TestResult represents the result of a single test case
type TestResult struct {
	TestCaseID    string                 `json:"test_case_id"`
	Status        string                 `json:"status"` // "passed", "failed", "error"
	ActualOutput  string                 `json:"actual_output"`
	Score         float64                `json:"score"`
	ErrorMessage  string                 `json:"error_message,omitempty"`
	ExecutionTime time.Duration          `json:"execution_time"`
	Details       map[string]interface{} `json:"details,omitempty"`
}

// ValidationMetrics represents comprehensive validation performance metrics
type ValidationMetrics struct {
	AverageLatency      time.Duration `json:"average_latency"`
	SuccessRate         float64       `json:"success_rate"`
	TokenConsumption    int64         `json:"token_consumption"`
	ThroughputPerSecond float64       `json:"throughput_per_second"`
}

// TEEAttestation represents a TEE attestation record
type TEEAttestation struct {
	ID           string     `json:"id"`
	NodeID       string     `json:"node_id"`
	TEEType      string     `json:"tee_type"`
	Status       string     `json:"status"` // "valid", "invalid", "expired", "pending"
	Quote        string     `json:"quote"`
	Signature    string     `json:"signature"`
	CertChain    string     `json:"cert_chain"`
	Measurements []string   `json:"measurements"`
	CreatedAt    time.Time  `json:"created_at"`
	ExpiresAt    time.Time  `json:"expires_at"`
	VerifiedAt   *time.Time `json:"verified_at,omitempty"`
}

// CognitiveEngineMetrics represents cognitive engine performance metrics
type CognitiveEngineMetrics struct {
	ID                    string    `json:"id"`
	NodeID                string    `json:"node_id"`
	TasksProcessed        int64     `json:"tasks_processed"`
	AverageProcessingTime float64   `json:"average_processing_time"`
	SuccessRate           float64   `json:"success_rate"`
	AdaptationScore       float64   `json:"adaptation_score"`
	LearningProgress      float64   `json:"learning_progress"`
	ResourceUtilization   float64   `json:"resource_utilization"`
	Timestamp             time.Time `json:"timestamp"`
}

// SystemHealth represents overall system health metrics
type SystemHealth struct {
	ID                  string    `json:"id"`
	OverallStatus       string    `json:"overall_status"` // "healthy", "degraded", "critical"
	ActiveNodes         int       `json:"active_nodes"`
	TotalNodes          int       `json:"total_nodes"`
	PendingTasks        int       `json:"pending_tasks"`
	CompletedTasks      int       `json:"completed_tasks"`
	FailedTasks         int       `json:"failed_tasks"`
	AverageResponseTime float64   `json:"average_response_time"`
	NetworkLatency      float64   `json:"network_latency"`
	TEEHealthScore      float64   `json:"tee_health_score"`
	Timestamp           time.Time `json:"timestamp"`

	// Extended fields for comprehensive health monitoring
	Uptime           int64                       `json:"uptime"`
	Components       map[string]*ComponentHealth `json:"components"`
	ComponentSummary *ComponentSummary           `json:"component_summary,omitempty"`
	ActiveAlerts     int                         `json:"active_alerts,omitempty"`
	Alerts           []*SystemAlert              `json:"alerts"`
	Metrics          *SystemMetrics              `json:"metrics"`
}

type ComponentHealth struct {
	Status  string                 `json:"status"`
	Message string                 `json:"message"`
	Metrics map[string]interface{} `json:"metrics,omitempty"`
}

type ComponentSummary struct {
	TotalComponents    int `json:"total_components"`
	HealthyComponents  int `json:"healthy_components"`
	WarningComponents  int `json:"warning_components"`
	CriticalComponents int `json:"critical_components"`
}

type SystemAlert struct {
	ID         string                 `json:"id"`
	Severity   string                 `json:"severity"`
	Component  string                 `json:"component"`
	Message    string                 `json:"message"`
	Timestamp  string                 `json:"timestamp"`
	Resolved   bool                   `json:"resolved"`
	ResolvedAt string                 `json:"resolved_at,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

type SystemMetrics struct {
	SystemLoad        float64 `json:"system_load"`
	MemoryUsage       float64 `json:"memory_usage"`
	DiskUsage         float64 `json:"disk_usage"`
	NetworkThroughput float64 `json:"network_throughput"`
	ActiveConnections int     `json:"active_connections"`
	GoroutineCount    int     `json:"goroutine_count"`
	CPUUsage          float64 `json:"cpu_usage"`
}

type DiagnosticsResult struct {
	ID        string            `json:"id"`
	Timestamp string            `json:"timestamp"`
	Status    string            `json:"status"`
	Summary   string            `json:"summary"`
	Tests     []*DiagnosticTest `json:"tests"`
}

type DiagnosticTest struct {
	Name     string                 `json:"name"`
	Status   string                 `json:"status"`
	Duration int64                  `json:"duration"`
	Message  string                 `json:"message"`
	Details  map[string]interface{} `json:"details,omitempty"`
}

type SystemHealthAction struct {
	Action     string                 `json:"action"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// Controller Integration objects
type ControllerSession struct {
	ID                string                 `json:"id"`
	SessionID         string                 `json:"session_id"`
	DesktopID         string                 `json:"desktop_id"`
	UserID            string                 `json:"user_id"`
	MobileDeviceID    string                 `json:"mobile_device_id"`
	Status            string                 `json:"status"` // "active", "inactive", "terminated", "expired"
	CreatedAt         time.Time              `json:"created_at"`
	LastActivity      time.Time              `json:"last_activity"`
	ExpiresAt         time.Time              `json:"expires_at"`
	TerminatedAt      *time.Time             `json:"terminated_at,omitempty"`
	TerminationReason string                 `json:"termination_reason,omitempty"`
	Capabilities      []string               `json:"capabilities"`
	DeviceInfo        *DeviceInfo            `json:"device_info"`
	ConnectionInfo    *ConnectionInfo        `json:"connection_info"`
	SessionData       map[string]interface{} `json:"session_data"`
	MessageCount      int                    `json:"message_count"`
}

type QRCode struct {
	ID            string      `json:"id"`
	SessionID     string      `json:"session_id"`
	DesktopID     string      `json:"desktop_id"`
	UserID        string      `json:"user_id"`
	DeviceType    string      `json:"device_type"`
	Capabilities  []string    `json:"capabilities"`
	Status        string      `json:"status"` // "active", "used", "expired"
	CreatedAt     time.Time   `json:"created_at"`
	ExpiresAt     time.Time   `json:"expires_at"`
	UsedAt        *time.Time  `json:"used_at,omitempty"`
	LastScannedAt *time.Time  `json:"last_scanned_at,omitempty"`
	ScanCount     int         `json:"scan_count"`
	MaxScans      int         `json:"max_scans"`
	Data          *QRCodeData `json:"data"`
}

type QRCodeData struct {
	Version      string   `json:"version"`
	Type         string   `json:"type"`
	SessionID    string   `json:"session_id"`
	DesktopID    string   `json:"desktop_id"`
	UserID       string   `json:"user_id"`
	DeviceType   string   `json:"device_type"`
	Capabilities []string `json:"capabilities"`
	ExpiresAt    int64    `json:"expires_at"`
	Timestamp    int64    `json:"timestamp"`
	Signature    string   `json:"signature"`
}

type PairingRequest struct {
	ID             string      `json:"id"`
	QRCodeID       string      `json:"qr_code_id"`
	SessionID      string      `json:"session_id"`
	DesktopID      string      `json:"desktop_id"`
	UserID         string      `json:"user_id"`
	MobileDeviceID string      `json:"mobile_device_id"`
	Status         string      `json:"status"` // "pending", "confirmed", "rejected", "expired"
	CreatedAt      time.Time   `json:"created_at"`
	ExpiresAt      time.Time   `json:"expires_at"`
	ConfirmedAt    *time.Time  `json:"confirmed_at,omitempty"`
	RejectedAt     *time.Time  `json:"rejected_at,omitempty"`
	Capabilities   []string    `json:"capabilities"`
	DeviceInfo     *DeviceInfo `json:"device_info"`
}

type DeviceInfo struct {
	DeviceID   string `json:"device_id"`
	DeviceType string `json:"device_type"`
	Platform   string `json:"platform"`
	Version    string `json:"version"`
	UserModel  string `json:"user_model,omitempty"`
}

type ConnectionInfo struct {
	IPAddress      string `json:"ip_address"`
	UserModel      string `json:"user_model"`
	ConnectionType string `json:"connection_type"`
	Encrypted      bool   `json:"encrypted"`
}

type WebSocketConnection struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	UserID    string    `json:"user_id"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type ControllerMessage struct {
	ID        string                 `json:"id"`
	SessionID string                 `json:"session_id"`
	Type      string                 `json:"type"`      // "command", "response", "notification", "heartbeat"
	Direction string                 `json:"direction"` // "inbound", "outbound"
	Payload   map[string]interface{} `json:"payload"`
	Timestamp time.Time              `json:"timestamp"`
	Processed bool                   `json:"processed"`
}

type ControllerAction struct {
	Action     string                 `json:"action"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// Alert represents a system alert
type Alert struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`     // "error", "warning", "info"
	Severity   string                 `json:"severity"` // "critical", "high", "medium", "low"
	Title      string                 `json:"title"`
	Message    string                 `json:"message"`
	Source     string                 `json:"source"` // Component that generated the alert
	NodeID     string                 `json:"node_id,omitempty"`
	Metadata   map[string]interface{} `json:"metadata"`
	Status     string                 `json:"status"` // "active", "acknowledged", "resolved"
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
	ResolvedAt *time.Time             `json:"resolved_at,omitempty"`
}

// P2PMessage represents a P2P network message
type P2PMessage struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`         // "validation_request", "validation_result", "node_announcement"
	From      string                 `json:"from"`         // Sender peer ID
	To        string                 `json:"to,omitempty"` // Recipient peer ID (empty for broadcast)
	Topic     string                 `json:"topic"`
	Payload   map[string]interface{} `json:"payload"`
	Timestamp time.Time              `json:"timestamp"`
	Signature string                 `json:"signature"`
}

// NetworkTopology represents the P2P network topology
type NetworkTopology struct {
	ID             string           `json:"id"`
	TotalPeers     int              `json:"total_peers"`
	ConnectedPeers int              `json:"connected_peers"`
	Peers          []PeerInfo       `json:"peers"`
	Connections    []ConnectionInfo `json:"connections"`
	Timestamp      time.Time        `json:"timestamp"`
}

// PeerInfo represents information about a P2P peer
type PeerInfo struct {
	ID       string    `json:"id"`
	Address  string    `json:"address"`
	Role     string    `json:"role"`
	Status   string    `json:"status"`
	Latency  int64     `json:"latency"`
	LastSeen time.Time `json:"last_seen"`
}

// P2PConnectionInfo represents a P2P connection
type P2PConnectionInfo struct {
	From      string    `json:"from"`
	To        string    `json:"to"`
	Status    string    `json:"status"`
	Latency   int64     `json:"latency"`
	Bandwidth int64     `json:"bandwidth"`
	CreatedAt time.Time `json:"created_at"`
}
