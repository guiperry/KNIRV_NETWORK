package websocket

// Additional event types for comprehensive real-time streaming

// PolicyViolationUpdate represents a policy violation event
type PolicyViolationUpdate struct {
	ID          string `json:"id"`
	NodeID      string `json:"node_id"`
	RuleID      string `json:"rule_id"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Timestamp   string `json:"timestamp"`
	Resolved    bool   `json:"resolved"`
}

// WorkflowExecutionUpdate represents a workflow execution event
type WorkflowExecutionUpdate struct {
	ExecutionID string `json:"execution_id"`
	WorkflowID  string `json:"workflow_id"`
	NodeID      string `json:"node_id"`
	Status      string `json:"status"`
	Progress    int    `json:"progress"`
	CurrentStep string `json:"current_step,omitempty"`
	StepsTotal  int    `json:"steps_total"`
	StepsDone   int    `json:"steps_done"`
	Timestamp   string `json:"timestamp"`
}

// AgentStatusUpdate represents an agent status change event
type AgentStatusUpdate struct {
	AgentID     string `json:"agent_id"`
	NodeID      string `json:"node_id"`
	Status      string `json:"status"`
	CurrentTask string `json:"current_task,omitempty"`
	TasksDone   int    `json:"tasks_done"`
	TasksFailed int    `json:"tasks_failed"`
	Timestamp   string `json:"timestamp"`
}

// NetworkTopologyUpdate represents a network topology change event
type NetworkTopologyUpdate struct {
	EventType  string        `json:"event_type"` // "node_joined", "node_left", "connection_changed"
	NodeID     string        `json:"node_id"`
	PeerCount  int           `json:"peer_count"`
	Peers      []PeerInfo    `json:"peers,omitempty"`
	Timestamp  string        `json:"timestamp"`
}

// PeerInfo represents information about a peer node
type PeerInfo struct {
	ID       string `json:"id"`
	Address  string `json:"address"`
	Latency  int64  `json:"latency_ms"`
	Status   string `json:"status"`
}

// ResourceMetricsUpdate represents system resource metrics
type ResourceMetricsUpdate struct {
	NodeID        string  `json:"node_id"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryPercent float64 `json:"memory_percent"`
	DiskPercent   float64 `json:"disk_percent"`
	NetworkIn     int64   `json:"network_in_bytes"`
	NetworkOut    int64   `json:"network_out_bytes"`
	Timestamp     string  `json:"timestamp"`
}
