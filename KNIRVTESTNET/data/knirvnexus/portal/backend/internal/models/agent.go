package models

import (
	"time"
)

// Agent represents a WASM agent in the system
type Agent struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	Version         string                 `json:"version"`
	Author          string                 `json:"author"`
	Type            string                 `json:"type"` // "WASM", "LoRA", "CodeT5", "SEAL", "NRN"
	Status          string                 `json:"status"` // "uploaded", "deployed", "running", "stopped", "error", "archived"
	FilePath        string                 `json:"file_path"`
	FileSize        int64                  `json:"file_size"`
	FileHash        string                 `json:"file_hash"`
	Capabilities    []string               `json:"capabilities"`
	Dependencies    []string               `json:"dependencies"`
	ResourceLimits  *AgentResourceLimits   `json:"resource_limits"`
	Configuration   map[string]interface{} `json:"configuration"`
	Metadata        map[string]interface{} `json:"metadata"`
	Tags            []string               `json:"tags"`
	UploadedAt      time.Time              `json:"uploaded_at"`
	DeployedAt      *time.Time             `json:"deployed_at,omitempty"`
	LastModified    time.Time              `json:"last_modified"`
	LastActivity    *time.Time             `json:"last_activity,omitempty"`
	UploadedBy      string                 `json:"uploaded_by"`
	DeployedBy      string                 `json:"deployed_by,omitempty"`
	RuntimeInstance *AgentRuntimeInstance  `json:"runtime_instance,omitempty"`
}

// AgentResourceLimits defines resource constraints for an agent
type AgentResourceLimits struct {
	MaxMemoryMB      int     `json:"max_memory_mb"`
	MaxCPUPercent    float64 `json:"max_cpu_percent"`
	MaxExecutionTime int     `json:"max_execution_time_seconds"`
	MaxConcurrency   int     `json:"max_concurrency"`
	MaxDiskMB        int     `json:"max_disk_mb"`
	NetworkAccess    bool    `json:"network_access"`
	FileSystemAccess bool    `json:"file_system_access"`
}

// AgentRuntimeInstance represents a running instance of an agent
type AgentRuntimeInstance struct {
	InstanceID       string                 `json:"instance_id"`
	ProcessID        int                    `json:"process_id,omitempty"`
	StartedAt        time.Time              `json:"started_at"`
	Status           string                 `json:"status"` // "starting", "running", "stopping", "stopped", "crashed"
	ResourceUsage    *AgentResourceUsage    `json:"resource_usage"`
	Configuration    map[string]interface{} `json:"configuration"`
	Environment      map[string]string      `json:"environment"`
	Port             int                    `json:"port,omitempty"`
	HealthCheckURL   string                 `json:"health_check_url,omitempty"`
	LastHealthCheck  *time.Time             `json:"last_health_check,omitempty"`
	HealthStatus     string                 `json:"health_status"` // "healthy", "unhealthy", "unknown"
	RestartCount     int                    `json:"restart_count"`
	ErrorMessage     string                 `json:"error_message,omitempty"`
}

// AgentResourceUsage tracks current resource consumption
type AgentResourceUsage struct {
	MemoryUsageMB    float64   `json:"memory_usage_mb"`
	CPUUsagePercent  float64   `json:"cpu_usage_percent"`
	DiskUsageMB      float64   `json:"disk_usage_mb"`
	NetworkBytesIn   int64     `json:"network_bytes_in"`
	NetworkBytesOut  int64     `json:"network_bytes_out"`
	ExecutionTime    int64     `json:"execution_time_seconds"`
	RequestCount     int64     `json:"request_count"`
	ErrorCount       int64     `json:"error_count"`
	LastUpdated      time.Time `json:"last_updated"`
}

// AgentDeployment represents a deployment configuration
type AgentDeployment struct {
	ID               string                 `json:"id"`
	AgentID          string                 `json:"agent_id"`
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	Environment      string                 `json:"environment"` // "development", "staging", "production"
	Replicas         int                    `json:"replicas"`
	Strategy         string                 `json:"strategy"` // "rolling", "blue-green", "canary"
	Configuration    map[string]interface{} `json:"configuration"`
	ResourceLimits   *AgentResourceLimits   `json:"resource_limits"`
	HealthCheck      *AgentHealthCheck      `json:"health_check"`
	AutoRestart      bool                   `json:"auto_restart"`
	RestartPolicy    string                 `json:"restart_policy"` // "always", "on-failure", "never"
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
	CreatedBy        string                 `json:"created_by"`
	Status           string                 `json:"status"` // "pending", "deploying", "deployed", "failed", "stopped"
	Instances        []*AgentRuntimeInstance `json:"instances"`
}

// AgentHealthCheck defines health check configuration
type AgentHealthCheck struct {
	Enabled         bool   `json:"enabled"`
	Path            string `json:"path"`
	IntervalSeconds int    `json:"interval_seconds"`
	TimeoutSeconds  int    `json:"timeout_seconds"`
	FailureThreshold int   `json:"failure_threshold"`
	SuccessThreshold int   `json:"success_threshold"`
}

// AgentLog represents a log entry from an agent
type AgentLog struct {
	ID         string    `json:"id"`
	AgentID    string    `json:"agent_id"`
	InstanceID string    `json:"instance_id,omitempty"`
	Level      string    `json:"level"` // "debug", "info", "warn", "error", "fatal"
	Message    string    `json:"message"`
	Timestamp  time.Time `json:"timestamp"`
	Source     string    `json:"source"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// AgentMetrics represents performance metrics for an agent
type AgentMetrics struct {
	AgentID           string    `json:"agent_id"`
	InstanceID        string    `json:"instance_id,omitempty"`
	Timestamp         time.Time `json:"timestamp"`
	RequestsPerSecond float64   `json:"requests_per_second"`
	AverageLatency    float64   `json:"average_latency_ms"`
	ErrorRate         float64   `json:"error_rate_percent"`
	Throughput        float64   `json:"throughput_mb_per_second"`
	ResourceUsage     *AgentResourceUsage `json:"resource_usage"`
}

// AgentEvent represents an event in the agent lifecycle
type AgentEvent struct {
	ID          string                 `json:"id"`
	AgentID     string                 `json:"agent_id"`
	InstanceID  string                 `json:"instance_id,omitempty"`
	Type        string                 `json:"type"` // "uploaded", "deployed", "started", "stopped", "crashed", "updated", "deleted"
	Description string                 `json:"description"`
	Timestamp   time.Time              `json:"timestamp"`
	UserID      string                 `json:"user_id,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// AgentTemplate represents a template for creating agents
type AgentTemplate struct {
	ID               string                 `json:"id"`
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	Version          string                 `json:"version"`
	Type             string                 `json:"type"`
	SourceCode       string                 `json:"source_code,omitempty"`
	BuildScript      string                 `json:"build_script,omitempty"`
	DefaultConfig    map[string]interface{} `json:"default_config"`
	ResourceLimits   *AgentResourceLimits   `json:"resource_limits"`
	RequiredCapabilities []string           `json:"required_capabilities"`
	Tags             []string               `json:"tags"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
	CreatedBy        string                 `json:"created_by"`
	IsPublic         bool                   `json:"is_public"`
	UsageCount       int                    `json:"usage_count"`
}

// AgentAction represents an action that can be performed on an agent
type AgentAction struct {
	Action     string                 `json:"action"` // "deploy", "start", "stop", "restart", "update", "delete", "scale"
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// AgentFilter represents filtering options for agent queries
type AgentFilter struct {
	Status       []string `json:"status,omitempty"`
	Type         []string `json:"type,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	Author       string   `json:"author,omitempty"`
	Environment  string   `json:"environment,omitempty"`
	CreatedAfter *time.Time `json:"created_after,omitempty"`
	CreatedBefore *time.Time `json:"created_before,omitempty"`
	Search       string   `json:"search,omitempty"`
	Limit        int      `json:"limit,omitempty"`
	Offset       int      `json:"offset,omitempty"`
}

// AgentSummary represents a summary view of agents
type AgentSummary struct {
	TotalAgents    int `json:"total_agents"`
	RunningAgents  int `json:"running_agents"`
	StoppedAgents  int `json:"stopped_agents"`
	ErrorAgents    int `json:"error_agents"`
	DeployedAgents int `json:"deployed_agents"`
	UploadedAgents int `json:"uploaded_agents"`
}

// Validation methods
func (a *Agent) IsValid() bool {
	return a.ID != "" && a.Name != "" && a.Type != "" && a.Status != ""
}

func (a *Agent) IsRunning() bool {
	return a.Status == "running" && a.RuntimeInstance != nil && a.RuntimeInstance.Status == "running"
}

func (a *Agent) CanDeploy() bool {
	return a.Status == "uploaded" || a.Status == "stopped"
}

func (a *Agent) CanStart() bool {
	return a.Status == "deployed" || a.Status == "stopped"
}

func (a *Agent) CanStop() bool {
	return a.Status == "running"
}

func (rl *AgentResourceLimits) IsValid() bool {
	return rl.MaxMemoryMB > 0 && rl.MaxCPUPercent > 0 && rl.MaxExecutionTime > 0
}

func (ru *AgentResourceUsage) IsWithinLimits(limits *AgentResourceLimits) bool {
	if limits == nil {
		return true
	}
	
	return ru.MemoryUsageMB <= float64(limits.MaxMemoryMB) &&
		   ru.CPUUsagePercent <= limits.MaxCPUPercent &&
		   ru.ExecutionTime <= int64(limits.MaxExecutionTime)
}
