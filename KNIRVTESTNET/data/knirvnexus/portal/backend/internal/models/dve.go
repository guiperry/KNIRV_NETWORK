package models

import (
	"time"
)

// DVENode represents a DVE (Decentralized Validation Environment) node
type DVENode struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Status          string    `json:"status"` // "online", "offline", "maintenance", "error"
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
	CPUUsage      float64 `json:"cpu_usage"`
	MemoryUsage   float64 `json:"memory_usage"`
	NetworkLatency int64  `json:"network_latency"`
	
	// Geographic coordinates for spatial indexing
	Latitude  float64 `json:"latitude,omitempty"`
	Longitude float64 `json:"longitude,omitempty"`
}

// ValidationTask represents a validation task in the DVE network
type ValidationTask struct {
	ID               string                 `json:"id"`
	Type             string                 `json:"type"` // "skillnode", "base_llm", "custom"
	Status           string                 `json:"status"` // "pending", "assigned", "running", "completed", "failed"
	Priority         int                    `json:"priority"` // 1-10, higher is more urgent
	SkillCode        string                 `json:"skill_code,omitempty"`
	FailureContext   string                 `json:"failure_context,omitempty"`
	TestCases        []TestCase             `json:"test_cases"`
	RequiredTEEType  string                 `json:"required_tee_type"`
	AssignedNodeID   string                 `json:"assigned_node_id,omitempty"`
	RequestedBy      string                 `json:"requested_by"`
	Parameters       map[string]interface{} `json:"parameters"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
	StartedAt        *time.Time             `json:"started_at,omitempty"`
	CompletedAt      *time.Time             `json:"completed_at,omitempty"`
	TimeoutAt        time.Time              `json:"timeout_at"`
}

// TestCase represents a test case for validation
type TestCase struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Input       map[string]interface{} `json:"input"`
	Expected    map[string]interface{} `json:"expected"`
	Weight      float64                `json:"weight"` // Importance weight for scoring
}

// ValidationResult represents the result of a validation task
type ValidationResult struct {
	ID              string                 `json:"id"`
	TaskID          string                 `json:"task_id"`
	ValidatorNodeID string                 `json:"validator_node_id"`
	Status          string                 `json:"status"` // "success", "failure", "error"
	Score           float64                `json:"score"` // 0.0 - 1.0
	Results         map[string]interface{} `json:"results"`
	TestResults     []TestResult           `json:"test_results"`
	Proof           string                 `json:"proof"` // Cryptographic proof
	TEEAttestation  string                 `json:"tee_attestation"`
	ExecutionTime   time.Duration          `json:"execution_time"`
	ErrorMessage    string                 `json:"error_message,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	Signature       string                 `json:"signature"`
}

// TestResult represents the result of a single test case
type TestResult struct {
	TestCaseID   string                 `json:"test_case_id"`
	Status       string                 `json:"status"` // "passed", "failed", "error"
	ActualOutput map[string]interface{} `json:"actual_output"`
	Score        float64                `json:"score"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	ExecutionTime time.Duration         `json:"execution_time"`
}

// TEEAttestation represents a TEE attestation record
type TEEAttestation struct {
	ID            string    `json:"id"`
	NodeID        string    `json:"node_id"`
	TEEType       string    `json:"tee_type"`
	Status        string    `json:"status"` // "valid", "invalid", "expired", "pending"
	Quote         string    `json:"quote"`
	Signature     string    `json:"signature"`
	CertChain     string    `json:"cert_chain"`
	Measurements  []string  `json:"measurements"`
	CreatedAt     time.Time `json:"created_at"`
	ExpiresAt     time.Time `json:"expires_at"`
	VerifiedAt    *time.Time `json:"verified_at,omitempty"`
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
}

// Alert represents a system alert
type Alert struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"` // "error", "warning", "info"
	Severity    string                 `json:"severity"` // "critical", "high", "medium", "low"
	Title       string                 `json:"title"`
	Message     string                 `json:"message"`
	Source      string                 `json:"source"` // Component that generated the alert
	NodeID      string                 `json:"node_id,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
	Status      string                 `json:"status"` // "active", "acknowledged", "resolved"
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	ResolvedAt  *time.Time             `json:"resolved_at,omitempty"`
}

// P2PMessage represents a P2P network message
type P2PMessage struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"` // "validation_request", "validation_result", "node_announcement"
	From      string                 `json:"from"` // Sender peer ID
	To        string                 `json:"to,omitempty"` // Recipient peer ID (empty for broadcast)
	Topic     string                 `json:"topic"`
	Payload   map[string]interface{} `json:"payload"`
	Timestamp time.Time              `json:"timestamp"`
	Signature string                 `json:"signature"`
}

// NetworkTopology represents the P2P network topology
type NetworkTopology struct {
	ID            string              `json:"id"`
	TotalPeers    int                 `json:"total_peers"`
	ConnectedPeers int                `json:"connected_peers"`
	Peers         []PeerInfo          `json:"peers"`
	Connections   []ConnectionInfo    `json:"connections"`
	Timestamp     time.Time           `json:"timestamp"`
}

// PeerInfo represents information about a P2P peer
type PeerInfo struct {
	ID          string    `json:"id"`
	Address     string    `json:"address"`
	Role        string    `json:"role"`
	Status      string    `json:"status"`
	Latency     int64     `json:"latency"`
	LastSeen    time.Time `json:"last_seen"`
}

// ConnectionInfo represents a P2P connection
type ConnectionInfo struct {
	From      string    `json:"from"`
	To        string    `json:"to"`
	Status    string    `json:"status"`
	Latency   int64     `json:"latency"`
	Bandwidth int64     `json:"bandwidth"`
	CreatedAt time.Time `json:"created_at"`
}
