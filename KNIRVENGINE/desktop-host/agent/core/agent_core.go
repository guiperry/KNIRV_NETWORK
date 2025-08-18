package core

import (
	"context"
	"time"
)

// UnifiedAgent represents the complete agent data model
type UnifiedAgent struct {
	// Core fields
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Version     string    `json:"version"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Configuration
	Config map[string]interface{} `json:"config"`

	// Runtime information
	BuildTarget string `json:"build_target"` // "plugin", "wasm", etc.
	PluginPath  string `json:"plugin_path,omitempty"`
	Status      string `json:"status"`

	// Metadata
	Collection   string   `json:"collection"`
	ImageURL     string   `json:"image_url,omitempty"`
	Capabilities []string `json:"capabilities"`
	TargetTypes  []string `json:"target_types"`
	Tags         []string `json:"tags,omitempty"`

	// Security
	OwnerID     int64             `json:"owner_id"`
	APIKeys     map[string]string `json:"api_keys,omitempty"`
	Permissions map[string]bool   `json:"permissions,omitempty"`

	// Terminal configuration
	DefaultTerminalConfig *TerminalConfig `json:"default_terminal_config,omitempty"`
}

// TerminalConfig represents terminal configuration for an agent
type TerminalConfig struct {
	DefaultRows    int               `json:"default_rows"`
	DefaultCols    int               `json:"default_cols"`
	FontSize       int               `json:"font_size"`
	FontFamily     string            `json:"font_family"`
	Theme          string            `json:"theme"`
	ScrollbackSize int               `json:"scrollback_size"`
	AutoOpen       bool              `json:"auto_open"`
	CustomCSS      string            `json:"custom_css,omitempty"`
	CustomOptions  map[string]string `json:"custom_options,omitempty"`
}

// AgentCoreService provides a unified interface for agent operations
type AgentCoreService interface {
	// Core CRUD operations
	CreateAgent(ctx context.Context, agent *UnifiedAgent) error
	GetAgent(ctx context.Context, id string) (*UnifiedAgent, error)
	UpdateAgent(ctx context.Context, agent *UnifiedAgent) error
	DeleteAgent(ctx context.Context, id string) error
	ListAgents(ctx context.Context, filter map[string]interface{}) ([]*UnifiedAgent, error)

	// Discovery operations
	DiscoverAgents(ctx context.Context) ([]*UnifiedAgent, error)
	RegisterDiscoveredAgent(ctx context.Context, agentPath string) (*UnifiedAgent, error)

	// Configuration operations
	GetAgentConfig(ctx context.Context, id string) (map[string]interface{}, error)
	UpdateAgentConfig(ctx context.Context, id string, config map[string]interface{}) error

	// Search operations
	SearchAgents(ctx context.Context, query string, limit int) ([]*UnifiedAgent, error)

	// Agent loading and lifecycle operations
	LoadAgent(ctx context.Context, agent *UnifiedAgent) error
	StartAgent(ctx context.Context, agentID string) error
	StopAgent(ctx context.Context, agentID string) error

	// Lifecycle hooks
	OnAgentCreated(agent *UnifiedAgent)
	OnAgentUpdated(agent *UnifiedAgent)
	OnAgentDeleted(id string)
}

// AgentStorage provides the storage interface for agents
type AgentStorage interface {
	// Core storage operations
	Store(ctx context.Context, agent *UnifiedAgent) error
	Get(ctx context.Context, id string) (*UnifiedAgent, error)
	Update(ctx context.Context, agent *UnifiedAgent) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filter map[string]interface{}) ([]*UnifiedAgent, error)

	// Search operations
	Search(ctx context.Context, query string, limit int) ([]*UnifiedAgent, error)

	// Utility operations
	Exists(ctx context.Context, id string) (bool, error)
	Count(ctx context.Context, filter map[string]interface{}) (int, error)
}

// AgentDiscovery provides agent discovery capabilities
type AgentDiscovery interface {
	// Discover agents from various sources
	DiscoverFromPlugins(ctx context.Context, pluginDir string) ([]*UnifiedAgent, error)
	DiscoverFromWASM(ctx context.Context, wasmDir string) ([]*UnifiedAgent, error)
	DiscoverFromTemplates(ctx context.Context, templateDir string) ([]*UnifiedAgent, error)

	// Extract metadata from agent files
	ExtractMetadataFromPlugin(ctx context.Context, pluginPath string) (*UnifiedAgent, error)
	ExtractMetadataFromWASM(ctx context.Context, wasmPath string) (*UnifiedAgent, error)
	
	// Discover agents from zip archives
	DiscoverFromZip(ctx context.Context, zipPath string) ([]*UnifiedAgent, error)
}

// AgentLifecycleHook represents a lifecycle hook function
type AgentLifecycleHook func(agent *UnifiedAgent)

// AgentLifecycleManager manages agent lifecycle events
type AgentLifecycleManager interface {
	// Register lifecycle hooks
	RegisterCreatedHook(hook AgentLifecycleHook)
	RegisterUpdatedHook(hook AgentLifecycleHook)
	RegisterDeletedHook(hook func(id string))

	// Trigger lifecycle events
	TriggerCreated(agent *UnifiedAgent)
	TriggerUpdated(agent *UnifiedAgent)
	TriggerDeleted(id string)
}

// ServiceConfig represents configuration for the agent core service
type ServiceConfig struct {
	DBPath       string `json:"db_path"`
	PluginsDir   string `json:"plugins_dir"`
	WASMDir      string `json:"wasm_dir"`
	TemplatesDir string `json:"templates_dir"`
	OutputDir    string `json:"output_dir"`
	DataDir      string `json:"data_dir"`
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e ValidationError) Error() string {
	return e.Message
}

// AgentValidator provides validation for agent data
type AgentValidator interface {
	ValidateAgent(agent *UnifiedAgent) []ValidationError
	ValidateConfig(config map[string]interface{}) []ValidationError
	ValidateTerminalConfig(config *TerminalConfig) []ValidationError
}

// DefaultAgentValidator provides default validation logic
type DefaultAgentValidator struct{}

// ValidateAgent validates a unified agent
func (v *DefaultAgentValidator) ValidateAgent(agent *UnifiedAgent) []ValidationError {
	var errors []ValidationError

	if agent.ID == "" {
		errors = append(errors, ValidationError{Field: "id", Message: "Agent ID is required"})
	}

	if agent.Name == "" {
		errors = append(errors, ValidationError{Field: "name", Message: "Agent name is required"})
	}

	if agent.Type == "" {
		errors = append(errors, ValidationError{Field: "type", Message: "Agent type is required"})
	}

	if agent.Version == "" {
		errors = append(errors, ValidationError{Field: "version", Message: "Agent version is required"})
	}

	if agent.BuildTarget == "" {
		errors = append(errors, ValidationError{Field: "build_target", Message: "Build target is required"})
	}

	// Validate terminal config if present
	if agent.DefaultTerminalConfig != nil {
		terminalErrors := v.ValidateTerminalConfig(agent.DefaultTerminalConfig)
		errors = append(errors, terminalErrors...)
	}

	return errors
}

// ValidateConfig validates agent configuration
func (v *DefaultAgentValidator) ValidateConfig(config map[string]interface{}) []ValidationError {
	var errors []ValidationError

	// Parse the configuration using our utility functions
	parser := NewConfigParser()
	parsedConfig := parser.ParseAgentConfig(config)

	// Validate required fields
	if parsedConfig.Name == "" {
		errors = append(errors, ValidationError{
			Field:   "name",
			Message: "Agent name is required",
		})
	}

	if parsedConfig.Type == "" {
		errors = append(errors, ValidationError{
			Field:   "type",
			Message: "Agent type is required",
		})
	}

	// Validate numeric fields
	if parsedConfig.MaxTokens < 0 {
		errors = append(errors, ValidationError{
			Field:   "max_tokens",
			Message: "Max tokens must be a positive number",
		})
	}

	// Validate temperature if provided
	if parsedConfig.Temperature != nil {
		temp := *parsedConfig.Temperature
		if temp < 0 || temp > 1 {
			errors = append(errors, ValidationError{
				Field:   "temperature",
				Message: "Temperature must be between 0 and 1",
			})
		}
	}

	// Validate terminal config if present
	if terminalConfig, ok := config["terminal_config"].(map[string]interface{}); ok {
		parsedTerminalConfig := parser.ParseTerminalConfig(terminalConfig)
		if parsedTerminalConfig.DefaultRows <= 0 || parsedTerminalConfig.DefaultCols <= 0 {
			errors = append(errors, ValidationError{
				Field:   "terminal_config",
				Message: "Terminal rows and columns must be positive numbers",
			})
		}
	}

	return errors
}

// ValidateTerminalConfig validates terminal configuration
func (v *DefaultAgentValidator) ValidateTerminalConfig(config *TerminalConfig) []ValidationError {
	var errors []ValidationError

	if config.DefaultRows <= 0 {
		errors = append(errors, ValidationError{Field: "default_rows", Message: "Default rows must be positive"})
	}

	if config.DefaultCols <= 0 {
		errors = append(errors, ValidationError{Field: "default_cols", Message: "Default cols must be positive"})
	}

	if config.FontSize <= 0 {
		errors = append(errors, ValidationError{Field: "font_size", Message: "Font size must be positive"})
	}

	if config.ScrollbackSize < 0 {
		errors = append(errors, ValidationError{Field: "scrollback_size", Message: "Scrollback size cannot be negative"})
	}

	return errors
}

// NewDefaultAgentValidator creates a new default agent validator
func NewDefaultAgentValidator() *DefaultAgentValidator {
	return &DefaultAgentValidator{}
}
