package models

import (
	"time"
)

// Model represents a WASM-based AI model
type Model struct {
	ID              string                 `json:"id" buntdb:"id"`
	Name            string                 `json:"name" buntdb:"name"`
	Description     string                 `json:"description" buntdb:"description"`
	Version         string                 `json:"version" buntdb:"version"`
	Type            string                 `json:"type" buntdb:"type"` // WASM, LoRA, CodeT5, SEAL, NRN
	Author          string                 `json:"author" buntdb:"author"`
	UploadedBy      string                 `json:"uploaded_by" buntdb:"uploaded_by"`
	UploadedAt      time.Time              `json:"uploaded_at" buntdb:"uploaded_at"`
	LastModified    time.Time              `json:"last_modified" buntdb:"last_modified"`
	FilePath        string                 `json:"file_path" buntdb:"file_path"`
	FileSize        int64                  `json:"file_size" buntdb:"file_size"`
	Hash            string                 `json:"hash" buntdb:"hash"`
	Status          string                 `json:"status" buntdb:"status"` // uploaded, deployed, running, stopped, error
	Configuration   map[string]interface{} `json:"configuration" buntdb:"configuration"`
	ResourceLimits  *ModelResourceLimits   `json:"resource_limits" buntdb:"resource_limits"`
	Tags            []string               `json:"tags" buntdb:"tags"`
	DeployedAt      *time.Time             `json:"deployed_at,omitempty" buntdb:"deployed_at"`
	LastActivity    *time.Time             `json:"last_activity,omitempty" buntdb:"last_activity"`

	// Runtime information
	RuntimeInstance *ModelRuntimeInstance `json:"runtime_instance,omitempty" buntdb:"runtime_instance"`
}

// ModelResourceLimits defines resource constraints for model execution
type ModelResourceLimits struct {
	MaxMemoryMB      int     `json:"max_memory_mb" buntdb:"max_memory_mb"`
	MaxCPUPercent    float64 `json:"max_cpu_percent" buntdb:"max_cpu_percent"`
	MaxExecutionTime int     `json:"max_execution_time" buntdb:"max_execution_time"` // seconds
	MaxConcurrency   int     `json:"max_concurrency" buntdb:"max_concurrency"`
	MaxDiskMB        int     `json:"max_disk_mb" buntdb:"max_disk_mb"`
	NetworkAccess    bool    `json:"network_access" buntdb:"network_access"`
	FileSystemAccess bool    `json:"filesystem_access" buntdb:"filesystem_access"`
}

// ModelRuntimeInstance represents a running model instance
type ModelRuntimeInstance struct {
	InstanceID      string                    `json:"instance_id" buntdb:"instance_id"`
	StartedAt       time.Time                 `json:"started_at" buntdb:"started_at"`
	Status          string                    `json:"status" buntdb:"status"` // starting, running, stopping, stopped, failed
	Configuration   map[string]interface{}    `json:"configuration" buntdb:"configuration"`
	Environment     map[string]string         `json:"environment" buntdb:"environment"`
	HealthStatus    string                    `json:"health_status" buntdb:"health_status"`
	RestartCount    int                       `json:"restart_count" buntdb:"restart_count"`
	ResourceUsage   *ModelResourceUsage       `json:"resource_usage" buntdb:"resource_usage"`
	ResourceLimits  *ModelResourceLimits      `json:"resource_limits" buntdb:"resource_limits"`
	LastHealthCheck *time.Time                `json:"last_health_check,omitempty" buntdb:"last_health_check"`
	HealthEndpoint  string                    `json:"health_endpoint" buntdb:"health_endpoint"`
}

// ModelResourceUsage tracks actual resource consumption
type ModelResourceUsage struct {
	MemoryUsageMB   float64 `json:"memory_usage_mb" buntdb:"memory_usage_mb"`
	CPUUsagePercent float64 `json:"cpu_usage_percent" buntdb:"cpu_usage_percent"`
	DiskUsageMB     float64 `json:"disk_usage_mb" buntdb:"disk_usage_mb"`
	ExecutionTime   int64   `json:"execution_time" buntdb:"execution_time"` // seconds
	RequestCount    int64   `json:"request_count" buntdb:"request_count"`
	ErrorCount      int64   `json:"error_count" buntdb:"error_count"`
	LastUpdated     time.Time `json:"last_updated" buntdb:"last_updated"`
}

// ModelMetrics represents performance and health metrics
type ModelMetrics struct {
	ModelID           string              `json:"model_id" buntdb:"model_id"`
	InstanceID        string              `json:"instance_id" buntdb:"instance_id"`
	Timestamp         time.Time           `json:"timestamp" buntdb:"timestamp"`
	RequestsPerSecond float64             `json:"requests_per_second" buntdb:"requests_per_second"`
	AverageLatency    float64             `json:"average_latency" buntdb:"average_latency"` // milliseconds
	ErrorRate         float64             `json:"error_rate" buntdb:"error_rate"`
	Throughput        float64             `json:"throughput" buntdb:"throughput"` // requests/second
	ResourceUsage     *ModelResourceUsage `json:"resource_usage" buntdb:"resource_usage"`
}

// ModelLog represents a log entry from model execution
type ModelLog struct {
	ID        string    `json:"id" buntdb:"id"`
	ModelID   string    `json:"model_id" buntdb:"model_id"`
	InstanceID string   `json:"instance_id" buntdb:"instance_id"`
	Level     string    `json:"level" buntdb:"level"` // DEBUG, INFO, WARN, ERROR
	Message   string    `json:"message" buntdb:"message"`
	Timestamp time.Time `json:"timestamp" buntdb:"timestamp"`
}

// ModelEvent represents a lifecycle event for a model
type ModelEvent struct {
	ID          string    `json:"id" buntdb:"id"`
	ModelID     string    `json:"model_id" buntdb:"model_id"`
	InstanceID  string    `json:"instance_id,omitempty" buntdb:"instance_id"`
	Type        string    `json:"type" buntdb:"type"` // uploaded, deployed, started, stopped, failed, scaled
	Description string    `json:"description" buntdb:"description"`
	Timestamp   time.Time `json:"timestamp" buntdb:"timestamp"`
	UserID      string    `json:"user_id,omitempty" buntdb:"user_id"`
}

// ModelFilter provides filtering options for model queries
type ModelFilter struct {
	Status        []string  `json:"status,omitempty"`
	Type          []string  `json:"type,omitempty"`
	Author        string    `json:"author,omitempty"`
	CreatedAfter  *time.Time `json:"created_after,omitempty"`
	CreatedBefore *time.Time `json:"created_before,omitempty"`
	Limit         int       `json:"limit,omitempty"`
	Offset        int       `json:"offset,omitempty"`
}

// ModelAction represents an action to perform on a model
type ModelAction struct {
	Action     string                 `json:"action"` // deploy, start, stop, restart, scale
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	UserID     string                 `json:"user_id,omitempty"`
}

// ModelSummary provides a summary of model statistics
type ModelSummary struct {
	TotalModels     int `json:"total_models"`
	RunningModels   int `json:"running_models"`
	StoppedModels   int `json:"stopped_models"`
	ErrorModels     int `json:"error_models"`
	DeployedModels  int `json:"deployed_models"`
	UploadedModels  int `json:"uploaded_models"`
}

// ModelDeployment represents a deployment configuration
type ModelDeployment struct {
	ID          string                      `json:"id" buntdb:"id"`
	ModelID     string                      `json:"model_id" buntdb:"model_id"`
	Name        string                      `json:"name" buntdb:"name"`
	Description string                      `json:"description" buntdb:"description"`
	Status      string                      `json:"status" buntdb:"status"` // pending, deploying, deployed, failed
	CreatedAt   time.Time                   `json:"created_at" buntdb:"created_at"`
	UpdatedAt   time.Time                   `json:"updated_at" buntdb:"updated_at"`
	Config      *ModelDeploymentConfig      `json:"config" buntdb:"config"`
	Instances   []*ModelRuntimeInstance     `json:"instances" buntdb:"instances"`
	Replicas    int                         `json:"replicas" buntdb:"replicas"`
}

// ModelDeploymentConfig defines deployment configuration
type ModelDeploymentConfig struct {
	AutoScaling     *AutoScalingConfig     `json:"auto_scaling,omitempty"`
	LoadBalancing   *LoadBalancingConfig   `json:"load_balancing,omitempty"`
	HealthChecks    *HealthCheckConfig     `json:"health_checks,omitempty"`
	ResourceLimits  *ModelResourceLimits   `json:"resource_limits,omitempty"`
	NetworkConfig   *NetworkConfig         `json:"network_config,omitempty"`
	SecurityConfig  *SecurityConfig        `json:"security_config,omitempty"`
}

// AutoScalingConfig defines auto-scaling parameters
type AutoScalingConfig struct {
	Enabled         bool    `json:"enabled"`
	MinReplicas     int     `json:"min_replicas"`
	MaxReplicas     int     `json:"max_replicas"`
	TargetCPUUtil   float64 `json:"target_cpu_util"`
	TargetMemoryUtil float64 `json:"target_memory_util"`
	ScaleUpThreshold  float64 `json:"scale_up_threshold"`
	ScaleDownThreshold float64 `json:"scale_down_threshold"`
}

// LoadBalancingConfig defines load balancing settings
type LoadBalancingConfig struct {
	Algorithm string `json:"algorithm"` // round_robin, least_connections, ip_hash
	StickySessions bool `json:"sticky_sessions"`
}

// HealthCheckConfig defines health check parameters
type HealthCheckConfig struct {
	Path            string        `json:"path"`
	Interval        time.Duration `json:"interval"`
	Timeout         time.Duration `json:"timeout"`
	UnhealthyThreshold int        `json:"unhealthy_threshold"`
	HealthyThreshold   int        `json:"healthy_threshold"`
}

// NetworkConfig defines network settings
type NetworkConfig struct {
	Port            int      `json:"port"`
	Protocol        string   `json:"protocol"` // http, https, grpc
	AllowedIPs      []string `json:"allowed_ips,omitempty"`
	RateLimit       *RateLimitConfig `json:"rate_limit,omitempty"`
}

// RateLimitConfig defines rate limiting settings
type RateLimitConfig struct {
	RequestsPerSecond float64 `json:"requests_per_second"`
	BurstSize         int     `json:"burst_size"`
}

// SecurityConfig defines security settings
type SecurityConfig struct {
	AuthenticationRequired bool     `json:"authentication_required"`
	AllowedRoles          []string `json:"allowed_roles,omitempty"`
	EncryptionEnabled     bool     `json:"encryption_enabled"`
	AuditLogging          bool     `json:"audit_logging"`
}

// ModelTemplate represents a reusable model template
type ModelTemplate struct {
	ID          string                 `json:"id" buntdb:"id"`
	Name        string                 `json:"name" buntdb:"name"`
	Description string                 `json:"description" buntdb:"description"`
	Type        string                 `json:"type" buntdb:"type"`
	Category    string                 `json:"category" buntdb:"category"`
	Config      map[string]interface{} `json:"config" buntdb:"config"`
	ResourceLimits *ModelResourceLimits `json:"resource_limits" buntdb:"resource_limits"`
	CreatedAt   time.Time              `json:"created_at" buntdb:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" buntdb:"updated_at"`
	UsageCount  int                    `json:"usage_count" buntdb:"usage_count"`
	Tags        []string               `json:"tags" buntdb:"tags"`
}

// IsValid validates the model data
func (m *Model) IsValid() bool {
	return m.ID != "" && m.Name != "" && m.Type != "" && m.FilePath != ""
}

// CanDeploy checks if the model can be deployed
func (m *Model) CanDeploy() bool {
	return m.Status == "uploaded"
}

// CanStart checks if the model can be started
func (m *Model) CanStart() bool {
	return m.Status == "deployed" || m.Status == "stopped"
}

// CanStop checks if the model can be stopped
func (m *Model) CanStop() bool {
	return m.Status == "running"
}

// IsRunning checks if the model is currently running
func (m *Model) IsRunning() bool {
	return m.Status == "running" && m.RuntimeInstance != nil
}
