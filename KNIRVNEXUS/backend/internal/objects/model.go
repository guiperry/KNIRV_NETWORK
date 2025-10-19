package objects

import (
	"time"
)

// Model represents a WASM model in the system
type Model struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	Version         string                 `json:"version"`
	Author          string                 `json:"author"`
	Type            string                 `json:"type"`   // "WASM", "LoRA", "CodeT5", "SEAL", "NRN"
	Status          string                 `json:"status"` // "uploaded", "deployed", "running", "stopped", "error", "archived"
	FilePath        string                 `json:"file_path"`
	FileSize        int64                  `json:"file_size"`
	FileHash        string                 `json:"file_hash"`
	Capabilities    []string               `json:"capabilities"`
	Dependencies    []string               `json:"dependencies"`
	ResourceLimits  *ModelResourceLimits   `json:"resource_limits"`
	Configuration   map[string]interface{} `json:"configuration"`
	Metadata        map[string]interface{} `json:"metadata"`
	Tags            []string               `json:"tags"`
	UploadedAt      time.Time              `json:"uploaded_at"`
	DeployedAt      *time.Time             `json:"deployed_at,omitempty"`
	LastModified    time.Time              `json:"last_modified"`
	LastActivity    *time.Time             `json:"last_activity,omitempty"`
	UploadedBy      string                 `json:"uploaded_by"`
	DeployedBy      string                 `json:"deployed_by,omitempty"`
	RuntimeInstance *ModelRuntimeInstance  `json:"runtime_instance,omitempty"`
}

// ModelResourceLimits defines resource constraints for an model
type ModelResourceLimits struct {
	MaxMemoryMB      int     `json:"max_memory_mb"`
	MaxCPUPercent    float64 `json:"max_cpu_percent"`
	MaxExecutionTime int     `json:"max_execution_time_seconds"`
	MaxConcurrency   int     `json:"max_concurrency"`
	MaxDiskMB        int     `json:"max_disk_mb"`
	NetworkAccess    bool    `json:"network_access"`
	FileSystemAccess bool    `json:"file_system_access"`
}

// ModelRuntimeInstance represents a running instance of an model
type ModelRuntimeInstance struct {
	InstanceID      string                 `json:"instance_id"`
	ProcessID       int                    `json:"process_id,omitempty"`
	StartedAt       time.Time              `json:"started_at"`
	Status          string                 `json:"status"` // "starting", "running", "stopping", "stopped", "crashed"
	ResourceUsage   *ModelResourceUsage    `json:"resource_usage"`
	Configuration   map[string]interface{} `json:"configuration"`
	Environment     map[string]string      `json:"environment"`
	Port            int                    `json:"port,omitempty"`
	HealthCheckURL  string                 `json:"health_check_url,omitempty"`
	LastHealthCheck *time.Time             `json:"last_health_check,omitempty"`
	HealthStatus    string                 `json:"health_status"` // "healthy", "unhealthy", "unknown"
	RestartCount    int                    `json:"restart_count"`
	ErrorMessage    string                 `json:"error_message,omitempty"`
}

// ModelResourceUsage tracks current resource consumption
type ModelResourceUsage struct {
	MemoryUsageMB   float64   `json:"memory_usage_mb"`
	CPUUsagePercent float64   `json:"cpu_usage_percent"`
	DiskUsageMB     float64   `json:"disk_usage_mb"`
	NetworkBytesIn  int64     `json:"network_bytes_in"`
	NetworkBytesOut int64     `json:"network_bytes_out"`
	ExecutionTime   int64     `json:"execution_time_seconds"`
	RequestCount    int64     `json:"request_count"`
	ErrorCount      int64     `json:"error_count"`
	LastUpdated     time.Time `json:"last_updated"`
}

// ModelDeployment represents a deployment configuration
type ModelDeployment struct {
	ID             string                  `json:"id"`
	ModelID        string                  `json:"model_id"`
	Name           string                  `json:"name"`
	Description    string                  `json:"description"`
	Environment    string                  `json:"environment"` // "development", "staging", "production"
	Replicas       int                     `json:"replicas"`
	Strategy       string                  `json:"strategy"` // "rolling", "blue-green", "canary"
	Configuration  map[string]interface{}  `json:"configuration"`
	ResourceLimits *ModelResourceLimits    `json:"resource_limits"`
	HealthCheck    *ModelHealthCheck       `json:"health_check"`
	AutoRestart    bool                    `json:"auto_restart"`
	RestartPolicy  string                  `json:"restart_policy"` // "always", "on-failure", "never"
	CreatedAt      time.Time               `json:"created_at"`
	UpdatedAt      time.Time               `json:"updated_at"`
	CreatedBy      string                  `json:"created_by"`
	Status         string                  `json:"status"` // "pending", "deploying", "deployed", "failed", "stopped"
	Instances      []*ModelRuntimeInstance `json:"instances"`
}

// ModelHealthCheck defines health check configuration
type ModelHealthCheck struct {
	Enabled          bool   `json:"enabled"`
	Path             string `json:"path"`
	IntervalSeconds  int    `json:"interval_seconds"`
	TimeoutSeconds   int    `json:"timeout_seconds"`
	FailureThreshold int    `json:"failure_threshold"`
	SuccessThreshold int    `json:"success_threshold"`
}

// ModelLog represents a log entry from an model
type ModelLog struct {
	ID         string                 `json:"id"`
	ModelID    string                 `json:"model_id"`
	InstanceID string                 `json:"instance_id,omitempty"`
	Level      string                 `json:"level"` // "debug", "info", "warn", "error", "fatal"
	Message    string                 `json:"message"`
	Timestamp  time.Time              `json:"timestamp"`
	Source     string                 `json:"source"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// ModelMetrics represents performance metrics for an model
type ModelMetrics struct {
	ModelID           string              `json:"model_id"`
	InstanceID        string              `json:"instance_id,omitempty"`
	Timestamp         time.Time           `json:"timestamp"`
	RequestsPerSecond float64             `json:"requests_per_second"`
	AverageLatency    float64             `json:"average_latency_ms"`
	ErrorRate         float64             `json:"error_rate_percent"`
	Throughput        float64             `json:"throughput_mb_per_second"`
	ResourceUsage     *ModelResourceUsage `json:"resource_usage"`
}

// ModelEvent represents an event in the model lifecycle
type ModelEvent struct {
	ID          string                 `json:"id"`
	ModelID     string                 `json:"model_id"`
	InstanceID  string                 `json:"instance_id,omitempty"`
	Type        string                 `json:"type"` // "uploaded", "deployed", "started", "stopped", "crashed", "updated", "deleted"
	Description string                 `json:"description"`
	Timestamp   time.Time              `json:"timestamp"`
	UserID      string                 `json:"user_id,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ModelTemplate represents a template for creating objects
type ModelTemplate struct {
	ID                   string                 `json:"id"`
	Name                 string                 `json:"name"`
	Description          string                 `json:"description"`
	Version              string                 `json:"version"`
	Type                 string                 `json:"type"`
	SourceCode           string                 `json:"source_code,omitempty"`
	BuildScript          string                 `json:"build_script,omitempty"`
	DefaultConfig        map[string]interface{} `json:"default_config"`
	ResourceLimits       *ModelResourceLimits   `json:"resource_limits"`
	RequiredCapabilities []string               `json:"required_capabilities"`
	Tags                 []string               `json:"tags"`
	CreatedAt            time.Time              `json:"created_at"`
	UpdatedAt            time.Time              `json:"updated_at"`
	CreatedBy            string                 `json:"created_by"`
	IsPublic             bool                   `json:"is_public"`
	UsageCount           int                    `json:"usage_count"`
}

// ModelAction represents an action that can be performed on an model
type ModelAction struct {
	Action     string                 `json:"action"` // "deploy", "start", "stop", "restart", "update", "delete", "scale"
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// ModelFilter represents filtering options for model queries
type ModelFilter struct {
	Status        []string   `json:"status,omitempty"`
	Type          []string   `json:"type,omitempty"`
	Tags          []string   `json:"tags,omitempty"`
	Author        string     `json:"author,omitempty"`
	Environment   string     `json:"environment,omitempty"`
	CreatedAfter  *time.Time `json:"created_after,omitempty"`
	CreatedBefore *time.Time `json:"created_before,omitempty"`
	Search        string     `json:"search,omitempty"`
	Limit         int        `json:"limit,omitempty"`
	Offset        int        `json:"offset,omitempty"`
}

// ModelSummary represents a summary view of objects
type ModelSummary struct {
	TotalModels    int `json:"total_objects"`
	RunningModels  int `json:"running_objects"`
	StoppedModels  int `json:"stopped_objects"`
	ErrorModels    int `json:"error_objects"`
	DeployedModels int `json:"deployed_objects"`
	UploadedModels int `json:"uploaded_objects"`
}

// Validation methods
func (a *Model) IsValid() bool {
	return a.ID != "" && a.Name != "" && a.Type != "" && a.Status != ""
}

func (a *Model) IsRunning() bool {
	return a.Status == "running" && a.RuntimeInstance != nil && a.RuntimeInstance.Status == "running"
}

func (a *Model) CanDeploy() bool {
	return a.Status == "uploaded" || a.Status == "stopped"
}

func (a *Model) CanStart() bool {
	return a.Status == "deployed" || a.Status == "stopped"
}

func (a *Model) CanStop() bool {
	return a.Status == "running"
}

func (rl *ModelResourceLimits) IsValid() bool {
	return rl.MaxMemoryMB > 0 && rl.MaxCPUPercent > 0 && rl.MaxExecutionTime > 0
}

func (ru *ModelResourceUsage) IsWithinLimits(limits *ModelResourceLimits) bool {
	if limits == nil {
		return true
	}

	return ru.MemoryUsageMB <= float64(limits.MaxMemoryMB) &&
		ru.CPUUsagePercent <= limits.MaxCPUPercent &&
		ru.ExecutionTime <= int64(limits.MaxExecutionTime)
}
