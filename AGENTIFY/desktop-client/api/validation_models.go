package api

import (
	"time"
)

// AgentCreateRequest represents the request body for creating an agent
type AgentCreateRequest struct {
	Name         string                 `json:"name" validate:"required,min=1,max=100"`
	Type         string                 `json:"type" validate:"required,agent_type"`
	Description  string                 `json:"description" validate:"max=500"`
	Version      string                 `json:"version" validate:"max=50"`
	BuildTarget  string                 `json:"build_target" validate:"required,build_target"`
	Config       map[string]interface{} `json:"config" validate:"required"`
	Collection   string                 `json:"collection" validate:"max=100"`
	ImageURL     string                 `json:"image_url" validate:"omitempty,url"`
	Capabilities []string               `json:"capabilities" validate:"dive,capability"`
	TargetTypes  []string               `json:"target_types" validate:"dive,min=1,max=50"`
	Tags         []string               `json:"tags" validate:"dive,min=1,max=50"`
	APIKeys      map[string]string      `json:"api_keys,omitempty"`
}

// AgentUpdateRequest represents the request body for updating an agent
type AgentUpdateRequest struct {
	Name         string                 `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	Type         string                 `json:"type,omitempty" validate:"omitempty,agent_type"`
	Description  string                 `json:"description,omitempty" validate:"omitempty,max=500"`
	Version      string                 `json:"version,omitempty" validate:"omitempty,max=50"`
	BuildTarget  string                 `json:"build_target,omitempty" validate:"omitempty,build_target"`
	Config       map[string]interface{} `json:"config,omitempty"`
	Collection   string                 `json:"collection,omitempty" validate:"omitempty,max=100"`
	ImageURL     string                 `json:"image_url,omitempty" validate:"omitempty,url"`
	Status       string                 `json:"status,omitempty" validate:"omitempty,status"`
	Capabilities []string               `json:"capabilities,omitempty" validate:"omitempty,dive,capability"`
	TargetTypes  []string               `json:"target_types,omitempty" validate:"omitempty,dive,min=1,max=50"`
	Tags         []string               `json:"tags,omitempty" validate:"omitempty,dive,min=1,max=50"`
	APIKeys      map[string]string      `json:"api_keys,omitempty"`
}

// TargetSystemCreateRequest represents the request body for creating a target system
type TargetSystemCreateRequest struct {
	Name         string                 `json:"name" validate:"required,min=1,max=100"`
	Type         string                 `json:"type" validate:"required,min=1,max=50"`
	Description  string                 `json:"description" validate:"max=500"`
	Config       map[string]interface{} `json:"config" validate:"required"`
	Capabilities []string               `json:"capabilities" validate:"dive,capability"`
	Tags         []string               `json:"tags" validate:"dive,min=1,max=50"`
}

// TargetSystemUpdateRequest represents the request body for updating a target system
type TargetSystemUpdateRequest struct {
	Name         string                 `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	Type         string                 `json:"type,omitempty" validate:"omitempty,min=1,max=50"`
	Description  string                 `json:"description,omitempty" validate:"omitempty,max=500"`
	Config       map[string]interface{} `json:"config,omitempty"`
	Status       string                 `json:"status,omitempty" validate:"omitempty,oneof=connected disconnected error"`
	Capabilities []string               `json:"capabilities,omitempty" validate:"omitempty,dive,capability"`
	Tags         []string               `json:"tags,omitempty" validate:"omitempty,dive,min=1,max=50"`
}

// CapabilityCreateRequest represents the request body for creating a capability
type CapabilityCreateRequest struct {
	Name        string                 `json:"name" validate:"required,min=1,max=100"`
	Type        string                 `json:"type" validate:"required,min=1,max=50"`
	Description string                 `json:"description" validate:"required,min=1,max=500"`
	Provider    string                 `json:"provider" validate:"required,min=1,max=100"`
	Config      map[string]interface{} `json:"config" validate:"required"`
	Schema      map[string]interface{} `json:"schema,omitempty"`
	Tags        []string               `json:"tags" validate:"dive,min=1,max=50"`
}

// CapabilityUpdateRequest represents the request body for updating a capability
type CapabilityUpdateRequest struct {
	Name        string                 `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	Type        string                 `json:"type,omitempty" validate:"omitempty,min=1,max=50"`
	Description string                 `json:"description,omitempty" validate:"omitempty,min=1,max=500"`
	Provider    string                 `json:"provider,omitempty" validate:"omitempty,min=1,max=100"`
	Config      map[string]interface{} `json:"config,omitempty"`
	Schema      map[string]interface{} `json:"schema,omitempty"`
	Status      string                 `json:"status,omitempty" validate:"omitempty,oneof=active inactive deprecated"`
	Tags        []string               `json:"tags,omitempty" validate:"omitempty,dive,min=1,max=50"`
}

// WorkflowCreateRequest represents the request body for creating a workflow
type WorkflowCreateRequest struct {
	AgentID      string                 `json:"agent_id" validate:"required,uuid"`
	TargetID     string                 `json:"target_id" validate:"required,uuid"`
	CapabilityID string                 `json:"capability_id" validate:"required,uuid"`
	Config       map[string]interface{} `json:"config,omitempty"`
	Priority     int                    `json:"priority" validate:"gte=0,lte=10"`
	Timeout      int                    `json:"timeout" validate:"gte=0,lte=3600"` // Max 1 hour
}

// WorkflowUpdateRequest represents the request body for updating a workflow
type WorkflowUpdateRequest struct {
	Status   string                 `json:"status,omitempty" validate:"omitempty,oneof=pending running completed failed cancelled"`
	Result   string                 `json:"result,omitempty" validate:"omitempty,max=1000"`
	Config   map[string]interface{} `json:"config,omitempty"`
	Priority int                    `json:"priority,omitempty" validate:"omitempty,gte=0,lte=10"`
}

// UserCreateRequest represents the request body for creating a user
type UserCreateRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50,alphanum"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=128"`
	Role     string `json:"role" validate:"required,oneof=admin user viewer"`
}

// UserUpdateRequest represents the request body for updating a user
type UserUpdateRequest struct {
	Username string `json:"username,omitempty" validate:"omitempty,min=3,max=50,alphanum"`
	Email    string `json:"email,omitempty" validate:"omitempty,email"`
	Role     string `json:"role,omitempty" validate:"omitempty,oneof=admin user viewer"`
}

// PasswordUpdateRequest represents the request body for updating a password
type PasswordUpdateRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=8,max=128"`
	ConfirmPassword string `json:"confirm_password" validate:"required,eqfield=NewPassword"`
}

// LoginRequest represents the request body for user login
type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// TokenCreateRequest represents the request body for creating an API token
type TokenCreateRequest struct {
	Description string     `json:"description" validate:"required,min=1,max=200"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// ConfigUpdateRequest represents the request body for updating configuration
type ConfigUpdateRequest struct {
	Key   string      `json:"key" validate:"required,min=1,max=100"`
	Value interface{} `json:"value" validate:"required"`
}

// BulkConfigUpdateRequest represents the request body for bulk configuration updates
type BulkConfigUpdateRequest struct {
	Configs map[string]interface{} `json:"configs" validate:"required,min=1"`
}

// InferenceRequest represents the request body for inference operations
type InferenceRequest struct {
	Prompt      string                 `json:"prompt" validate:"required,min=1,max=10000"`
	Model       string                 `json:"model,omitempty" validate:"omitempty,min=1,max=100"`
	MaxTokens   int                    `json:"max_tokens,omitempty" validate:"omitempty,gte=1,lte=8192"`
	Temperature float64                `json:"temperature,omitempty" validate:"omitempty,gte=0,lte=2"`
	Context     map[string]interface{} `json:"context,omitempty"`
}

// AgentBuildRequest represents the request body for building an agent
type AgentBuildRequest struct {
	AgentID     string                 `json:"agent_id" validate:"required,uuid"`
	BuildTarget string                 `json:"build_target" validate:"required,build_target"`
	Config      map[string]interface{} `json:"config,omitempty"`
	Force       bool                   `json:"force,omitempty"`
}

// AgentDeployRequest represents the request body for deploying an agent
type AgentDeployRequest struct {
	AgentID  string                 `json:"agent_id" validate:"required,uuid"`
	TargetID string                 `json:"target_id" validate:"required,uuid"`
	Config   map[string]interface{} `json:"config,omitempty"`
}

// MCPServerInstallRequest represents the request body for installing an MCP server
type MCPServerInstallRequest struct {
	Name        string                 `json:"name" validate:"required,min=1,max=100"`
	Source      string                 `json:"source" validate:"required,url"`
	Version     string                 `json:"version,omitempty" validate:"omitempty,max=50"`
	Config      map[string]interface{} `json:"config,omitempty"`
	AutoStart   bool                   `json:"auto_start,omitempty"`
}

// MCPServerUpdateRequest represents the request body for updating an MCP server
type MCPServerUpdateRequest struct {
	Config    map[string]interface{} `json:"config,omitempty"`
	AutoStart bool                   `json:"auto_start,omitempty"`
	Status    string                 `json:"status,omitempty" validate:"omitempty,oneof=running stopped error"`
}

// AnalyticsQueryRequest represents the request body for analytics queries
type AnalyticsQueryRequest struct {
	StartTime time.Time              `json:"start_time" validate:"required"`
	EndTime   time.Time              `json:"end_time" validate:"required,gtfield=StartTime"`
	Metrics   []string               `json:"metrics" validate:"required,dive,min=1"`
	Filters   map[string]interface{} `json:"filters,omitempty"`
	GroupBy   []string               `json:"group_by,omitempty" validate:"dive,min=1"`
}

// SystemMetricsRequest represents the request body for system metrics
type SystemMetricsRequest struct {
	Duration string   `json:"duration" validate:"required,oneof=1h 6h 24h 7d 30d"`
	Metrics  []string `json:"metrics" validate:"required,dive,oneof=cpu memory disk network"`
}

// BackupCreateRequest represents the request body for creating a backup
type BackupCreateRequest struct {
	Name        string   `json:"name" validate:"required,min=1,max=100"`
	Description string   `json:"description,omitempty" validate:"omitempty,max=500"`
	Components  []string `json:"components" validate:"required,dive,oneof=agents targets capabilities workflows users config"`
	Compress    bool     `json:"compress,omitempty"`
}

// RestoreRequest represents the request body for restoring from backup
type RestoreRequest struct {
	BackupID   string   `json:"backup_id" validate:"required,uuid"`
	Components []string `json:"components" validate:"required,dive,oneof=agents targets capabilities workflows users config"`
	Force      bool     `json:"force,omitempty"`
}
