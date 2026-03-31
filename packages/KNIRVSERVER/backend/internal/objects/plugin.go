package objects

import (
	"time"
)

// Plugin represents a WASM-based Agentic Memory Plugin unit
type Plugin struct {
	ID             string                 `json:"id" buntdb:"id"`
	Name           string                 `json:"name" buntdb:"name"`
	Description    string                 `json:"description" buntdb:"description"`
	Version        string                 `json:"version" buntdb:"version"`
	Type           string                 `json:"type" buntdb:"type"` // WASM, LoRA, CodeT5, SEAL, NRN
	Author         string                 `json:"author" buntdb:"author"`
	UploadedBy     string                 `json:"uploaded_by" buntdb:"uploaded_by"`
	UploadedAt     time.Time              `json:"uploaded_at" buntdb:"uploaded_at"`
	LastModified   time.Time              `json:"last_modified" buntdb:"last_modified"`
	FilePath       string                 `json:"file_path" buntdb:"file_path"`
	FileSize       int64                  `json:"file_size" buntdb:"file_size"`
	Hash           string                 `json:"hash" buntdb:"hash"`
	Status         string                 `json:"status" buntdb:"status"` // uploaded, deployed, running, stopped, error
	Configuration  map[string]interface{} `json:"configuration" buntdb:"configuration"`
	ResourceLimits *PluginResourceLimits  `json:"resource_limits" buntdb:"resource_limits"`
	Tags           []string               `json:"tags" buntdb:"tags"`
	DeployedAt     *time.Time             `json:"deployed_at,omitempty" buntdb:"deployed_at"`
	LastActivity   *time.Time             `json:"last_activity,omitempty" buntdb:"last_activity"`

	// Runtime information
	RuntimeInstance *PluginRuntimeInstance `json:"runtime_instance,omitempty" buntdb:"runtime_instance"`
}

// PluginResourceLimits defines resource constraints for plugin execution
type PluginResourceLimits struct {
	MaxMemoryMB      int     `json:"max_memory_mb" buntdb:"max_memory_mb"`
	MaxCPUPercent    float64 `json:"max_cpu_percent" buntdb:"max_cpu_percent"`
	MaxExecutionTime int     `json:"max_execution_time" buntdb:"max_execution_time"` // seconds
	MaxConcurrency   int     `json:"max_concurrency" buntdb:"max_concurrency"`
	MaxDiskMB        int     `json:"max_disk_mb" buntdb:"max_disk_mb"`
	NetworkAccess    bool    `json:"network_access" buntdb:"network_access"`
	FileSystemAccess bool    `json:"filesystem_access" buntdb:"filesystem_access"`
}

// PluginRuntimeInstance represents a running plugin instance
type PluginRuntimeInstance struct {
	InstanceID      string                 `json:"instance_id" buntdb:"instance_id"`
	StartedAt       time.Time              `json:"started_at" buntdb:"started_at"`
	Status          string                 `json:"status" buntdb:"status"` // starting, running, stopping, stopped, failed
	Configuration   map[string]interface{} `json:"configuration" buntdb:"configuration"`
	Environment     map[string]string      `json:"environment" buntdb:"environment"`
	HealthStatus    string                 `json:"health_status" buntdb:"health_status"`
	RestartCount    int                    `json:"restart_count" buntdb:"restart_count"`
	ResourceUsage   *PluginResourceUsage   `json:"resource_usage" buntdb:"resource_usage"`
	ResourceLimits  *PluginResourceLimits  `json:"resource_limits" buntdb:"resource_limits"`
	LastHealthCheck *time.Time             `json:"last_health_check,omitempty" buntdb:"last_health_check"`
	HealthEndpoint  string                 `json:"health_endpoint" buntdb:"health_endpoint"`
}

// PluginResourceUsage tracks actual resource consumption
type PluginResourceUsage struct {
	MemoryUsageMB   float64   `json:"memory_usage_mb" buntdb:"memory_usage_mb"`
	CPUUsagePercent float64   `json:"cpu_usage_percent" buntdb:"cpu_usage_percent"`
	DiskUsageMB     float64   `json:"disk_usage_mb" buntdb:"disk_usage_mb"`
	ExecutionTime   int64     `json:"execution_time" buntdb:"execution_time"` // seconds
	RequestCount    int64     `json:"request_count" buntdb:"request_count"`
	ErrorCount      int64     `json:"error_count" buntdb:"error_count"`
	LastUpdated     time.Time `json:"last_updated" buntdb:"last_updated"`
}

// PluginMetrics represents performance and health metrics
type PluginMetrics struct {
	PluginID          string               `json:"plugin_id" buntdb:"plugin_id"`
	InstanceID        string               `json:"instance_id" buntdb:"instance_id"`
	Timestamp         time.Time            `json:"timestamp" buntdb:"timestamp"`
	RequestsPerSecond float64              `json:"requests_per_second" buntdb:"requests_per_second"`
	AverageLatency    float64              `json:"average_latency" buntdb:"average_latency"` // milliseconds
	ErrorRate         float64              `json:"error_rate" buntdb:"error_rate"`
	Throughput        float64              `json:"throughput" buntdb:"throughput"` // requests/second
	ResourceUsage     *PluginResourceUsage `json:"resource_usage" buntdb:"resource_usage"`
}

// PluginLog represents a log entry from plugin execution
type PluginLog struct {
	ID         string    `json:"id" buntdb:"id"`
	PluginID   string    `json:"plugin_id" buntdb:"plugin_id"`
	InstanceID string    `json:"instance_id" buntdb:"instance_id"`
	Level      string    `json:"level" buntdb:"level"` // DEBUG, INFO, WARN, ERROR
	Message    string    `json:"message" buntdb:"message"`
	Timestamp  time.Time `json:"timestamp" buntdb:"timestamp"`
}

// PluginEvent represents a lifecycle event for a plugin item
type PluginEvent struct {
	ID          string    `json:"id" buntdb:"id"`
	PluginID    string    `json:"plugin_id" buntdb:"plugin_id"`
	InstanceID  string    `json:"instance_id,omitempty" buntdb:"instance_id"`
	Type        string    `json:"type" buntdb:"type"` // uploaded, deployed, started, stopped, failed, scaled
	Description string    `json:"description" buntdb:"description"`
	Timestamp   time.Time `json:"timestamp" buntdb:"timestamp"`
	UserID      string    `json:"user_id,omitempty" buntdb:"user_id"`
}

// PluginFilter provides filtering options for plugin queries
type PluginFilter struct {
	Status        []string   `json:"status,omitempty"`
	Type          []string   `json:"type,omitempty"`
	Author        string     `json:"author,omitempty"`
	CreatedAfter  *time.Time `json:"created_after,omitempty"`
	CreatedBefore *time.Time `json:"created_before,omitempty"`
	Limit         int        `json:"limit,omitempty"`
	Offset        int        `json:"offset,omitempty"`
}

// PluginAction represents an action to perform on a plugin item
type PluginAction struct {
	Action     string                 `json:"action"` // deploy, start, stop, restart, scale
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	UserID     string                 `json:"user_id,omitempty"`
}

// PluginSummary provides a summary of plugin statistics
type PluginSummary struct {
	TotalPlugins    int `json:"total_plugins"`
	RunningPlugins  int `json:"running_plugins"`
	StoppedPlugins  int `json:"stopped_plugins"`
	ErrorPlugins    int `json:"error_plugins"`
	DeployedPlugins int `json:"deployed_plugins"`
	UploadedPlugins int `json:"uploaded_plugins"`
}

// PluginDeployment represents a deployment configuration
type PluginDeployment struct {
	ID          string                   `json:"id" buntdb:"id"`
	PluginID    string                   `json:"plugin_id" buntdb:"plugin_id"`
	Name        string                   `json:"name" buntdb:"name"`
	Description string                   `json:"description" buntdb:"description"`
	Status      string                   `json:"status" buntdb:"status"` // pending, deploying, deployed, failed
	CreatedAt   time.Time                `json:"created_at" buntdb:"created_at"`
	UpdatedAt   time.Time                `json:"updated_at" buntdb:"updated_at"`
	Config      *PluginDeploymentConfig  `json:"config" buntdb:"config"`
	Instances   []*PluginRuntimeInstance `json:"instances" buntdb:"instances"`
	Replicas    int                      `json:"replicas" buntdb:"replicas"`
}

// PluginDeploymentConfig defines deployment configuration
type PluginDeploymentConfig struct {
	AutoScaling    *AutoScalingConfig    `json:"auto_scaling,omitempty"`
	LoadBalancing  *LoadBalancingConfig  `json:"load_balancing,omitempty"`
	HealthChecks   *HealthCheckConfig    `json:"health_checks,omitempty"`
	ResourceLimits *PluginResourceLimits `json:"resource_limits,omitempty"`
	NetworkConfig  *NetworkConfig        `json:"network_config,omitempty"`
	SecurityConfig *SecurityConfig       `json:"security_config,omitempty"`
}

// AutoScalingConfig defines auto-scaling parameters
type AutoScalingConfig struct {
	Enabled            bool    `json:"enabled"`
	MinReplicas        int     `json:"min_replicas"`
	MaxReplicas        int     `json:"max_replicas"`
	TargetCPUUtil      float64 `json:"target_cpu_util"`
	TargetMemoryUtil   float64 `json:"target_memory_util"`
	ScaleUpThreshold   float64 `json:"scale_up_threshold"`
	ScaleDownThreshold float64 `json:"scale_down_threshold"`
}

// LoadBalancingConfig defines load balancing settings
type LoadBalancingConfig struct {
	Algorithm      string `json:"algorithm"` // round_robin, least_connections, ip_hash
	StickySessions bool   `json:"sticky_sessions"`
}

// HealthCheckConfig defines health check parameters
type HealthCheckConfig struct {
	Path               string        `json:"path"`
	Interval           time.Duration `json:"interval"`
	Timeout            time.Duration `json:"timeout"`
	UnhealthyThreshold int           `json:"unhealthy_threshold"`
	HealthyThreshold   int           `json:"healthy_threshold"`
}

// NetworkConfig defines network settings
type NetworkConfig struct {
	Port       int              `json:"port"`
	Protocol   string           `json:"protocol"` // http, https, grpc
	AllowedIPs []string         `json:"allowed_ips,omitempty"`
	RateLimit  *RateLimitConfig `json:"rate_limit,omitempty"`
}

// RateLimitConfig defines rate limiting settings
type RateLimitConfig struct {
	RequestsPerSecond float64 `json:"requests_per_second"`
	BurstSize         int     `json:"burst_size"`
}

// SecurityConfig defines security settings
type SecurityConfig struct {
	AuthenticationRequired bool     `json:"authentication_required"`
	AllowedRoles           []string `json:"allowed_roles,omitempty"`
	EncryptionEnabled      bool     `json:"encryption_enabled"`
	AuditLogging           bool     `json:"audit_logging"`
}

// PluginTemplate represents a reusable plugin template
type PluginTemplate struct {
	ID             string                 `json:"id" buntdb:"id"`
	Name           string                 `json:"name" buntdb:"name"`
	Description    string                 `json:"description" buntdb:"description"`
	Type           string                 `json:"type" buntdb:"type"`
	Category       string                 `json:"category" buntdb:"category"`
	Config         map[string]interface{} `json:"config" buntdb:"config"`
	ResourceLimits *PluginResourceLimits  `json:"resource_limits" buntdb:"resource_limits"`
	CreatedAt      time.Time              `json:"created_at" buntdb:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at" buntdb:"updated_at"`
	UsageCount     int                    `json:"usage_count" buntdb:"usage_count"`
	Tags           []string               `json:"tags" buntdb:"tags"`
}

// IsValid validates the plugin data
func (m *Plugin) IsValid() bool {
	return m.ID != "" && m.Name != "" && m.Type != "" && m.FilePath != ""
}

// CanDeploy checks if the plugin item can be deployed
func (m *Plugin) CanDeploy() bool {
	return m.Status == "uploaded"
}

// CanStart checks if the plugin item can be started
func (m *Plugin) CanStart() bool {
	return m.Status == "deployed" || m.Status == "stopped"
}

// CanStop checks if the plugin item can be stopped
func (m *Plugin) CanStop() bool {
	return m.Status == "running"
}

// IsRunning checks if the plugin item is currently running
func (m *Plugin) IsRunning() bool {
	return m.Status == "running" && m.RuntimeInstance != nil
}
