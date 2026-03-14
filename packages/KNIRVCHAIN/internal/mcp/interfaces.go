package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// MCPProcessor defines the interface for Model Context Protocol processing
type MCPProcessor interface {
	// Processing operations
	ProcessRequest(request *MCPRequest) (*MCPResponse, error)
	ProcessBatch(requests []*MCPRequest) ([]*MCPResponse, error)

	// Context management
	CreateContext(contextID string, metadata map[string]interface{}) (*MCPContext, error)
	UpdateContext(contextID string, updates map[string]interface{}) error
	GetContext(contextID string) (*MCPContext, error)
	DeleteContext(contextID string) error

	// Model operations
	LoadModel(modelConfig *ModelConfig) error
	UnloadModel(modelID string) error
	GetLoadedModels() ([]*ModelInfo, error)

	// Lifecycle
	Start(ctx context.Context) error
	Stop() error
	IsRunning() bool
}

// ContextManager defines the interface for context management
type ContextManager interface {
	// Context lifecycle
	CreateContext(config *ContextConfig) (*MCPContext, error)
	DestroyContext(contextID string) error

	// Context operations
	AddToContext(contextID string, data *ContextData) error
	RemoveFromContext(contextID string, dataID string) error
	GetContextData(contextID string) ([]*ContextData, error)

	// Context queries
	SearchContext(contextID string, query *ContextQuery) ([]*ContextData, error)
	GetContextSummary(contextID string) (*ContextSummary, error)

	// Context optimization
	OptimizeContext(contextID string) error
	CompressContext(contextID string) error
	GetContextMetrics(contextID string) (*ContextMetrics, error)
}

// ModelManager defines the interface for model management
type ModelManager interface {
	// Model lifecycle
	RegisterModel(model *ModelDefinition) error
	UnregisterModel(modelID string) error

	// Model operations
	LoadModel(modelID string, config *ModelConfig) error
	UnloadModel(modelID string) error
	ReloadModel(modelID string) error

	// Model queries
	GetModel(modelID string) (*ModelInfo, error)
	ListModels() ([]*ModelInfo, error)
	GetModelCapabilities(modelID string) (*ModelCapabilities, error)

	// Model execution
	ExecuteModel(modelID string, input *ModelInput) (*ModelOutput, error)
	StreamModel(modelID string, input *ModelInput) (<-chan *ModelOutput, error)
}

// MCPRequest represents an MCP request
type MCPRequest struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	ModelID   string                 `json:"model_id"`
	ContextID string                 `json:"context_id,omitempty"`
	Input     *ModelInput            `json:"input"`
	Config    *RequestConfig         `json:"config,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// MCPResponse represents an MCP response
type MCPResponse struct {
	ID        string                 `json:"id"`
	RequestID string                 `json:"request_id"`
	Success   bool                   `json:"success"`
	Output    *ModelOutput           `json:"output,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Duration  time.Duration          `json:"duration"`
	Timestamp time.Time              `json:"timestamp"`
}

// MCPContext represents a model context
type MCPContext struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	Data       []*ContextData         `json:"data"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
	ExpiresAt  *time.Time             `json:"expires_at,omitempty"`
	Size       int64                  `json:"size"`
	TokenCount int                    `json:"token_count"`
}

// ContextData represents data within a context
type ContextData struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Content   string                 `json:"content"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Weight    float64                `json:"weight"`
	Tags      []string               `json:"tags,omitempty"`
}

// ContextConfig represents context configuration
type ContextConfig struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	Type         string        `json:"type"`
	MaxSize      int64         `json:"max_size"`
	MaxTokens    int           `json:"max_tokens"`
	TTL          time.Duration `json:"ttl,omitempty"`
	Compression  bool          `json:"compression"`
	Optimization bool          `json:"optimization"`
}

// ContextQuery represents a context query
type ContextQuery struct {
	Text      string                 `json:"text,omitempty"`
	Type      string                 `json:"type,omitempty"`
	Tags      []string               `json:"tags,omitempty"`
	TimeRange *TimeRange             `json:"time_range,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Limit     int                    `json:"limit,omitempty"`
}

// TimeRange represents a time range
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// ContextSummary represents a summary of context
type ContextSummary struct {
	ContextID   string    `json:"context_id"`
	DataCount   int       `json:"data_count"`
	TotalSize   int64     `json:"total_size"`
	TokenCount  int       `json:"token_count"`
	LastUpdated time.Time `json:"last_updated"`
	TopTags     []string  `json:"top_tags"`
	Summary     string    `json:"summary"`
}

// ContextMetrics represents context metrics
type ContextMetrics struct {
	ContextID         string        `json:"context_id"`
	AccessCount       int64         `json:"access_count"`
	LastAccessed      time.Time     `json:"last_accessed"`
	AverageLatency    time.Duration `json:"average_latency"`
	CompressionRatio  float64       `json:"compression_ratio"`
	OptimizationScore float64       `json:"optimization_score"`
}

// ModelDefinition represents a model definition
type ModelDefinition struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Version      string                 `json:"version"`
	Type         string                 `json:"type"`
	Provider     string                 `json:"provider"`
	Capabilities *ModelCapabilities     `json:"capabilities"`
	Config       *ModelConfig           `json:"config"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// ModelInfo represents information about a model
type ModelInfo struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Type        string                 `json:"type"`
	Provider    string                 `json:"provider"`
	Status      string                 `json:"status"`
	LoadedAt    *time.Time             `json:"loaded_at,omitempty"`
	MemoryUsage int64                  `json:"memory_usage"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ModelCapabilities represents model capabilities
type ModelCapabilities struct {
	TextGeneration   bool     `json:"text_generation"`
	TextEmbedding    bool     `json:"text_embedding"`
	ImageGeneration  bool     `json:"image_generation"`
	CodeGeneration   bool     `json:"code_generation"`
	Reasoning        bool     `json:"reasoning"`
	MaxTokens        int      `json:"max_tokens"`
	SupportedFormats []string `json:"supported_formats"`
	Languages        []string `json:"languages"`
}

// ModelConfig represents model configuration
type ModelConfig struct {
	Temperature      float64                `json:"temperature"`
	MaxTokens        int                    `json:"max_tokens"`
	TopP             float64                `json:"top_p"`
	TopK             int                    `json:"top_k"`
	FrequencyPenalty float64                `json:"frequency_penalty"`
	PresencePenalty  float64                `json:"presence_penalty"`
	StopSequences    []string               `json:"stop_sequences,omitempty"`
	Seed             *int64                 `json:"seed,omitempty"`
	Parameters       map[string]interface{} `json:"parameters,omitempty"`
}

// ModelInput represents input to a model
type ModelInput struct {
	Text     string                 `json:"text,omitempty"`
	Messages []*Message             `json:"messages,omitempty"`
	Images   []string               `json:"images,omitempty"`
	Context  string                 `json:"context,omitempty"`
	Format   string                 `json:"format,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ModelOutput represents output from a model
type ModelOutput struct {
	Text       string                 `json:"text,omitempty"`
	Tokens     []string               `json:"tokens,omitempty"`
	Embeddings []float32              `json:"embeddings,omitempty"`
	Images     []string               `json:"images,omitempty"`
	Confidence float64                `json:"confidence"`
	Usage      *UsageStats            `json:"usage,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// Message represents a message in a conversation
type Message struct {
	Role      string                 `json:"role"`
	Content   string                 `json:"content"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// RequestConfig represents request configuration
type RequestConfig struct {
	Timeout  time.Duration          `json:"timeout"`
	Retries  int                    `json:"retries"`
	Stream   bool                   `json:"stream"`
	Cache    bool                   `json:"cache"`
	Priority int                    `json:"priority"`
	Options  map[string]interface{} `json:"options,omitempty"`
}

// UsageStats represents usage statistics
type UsageStats struct {
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	TotalTokens  int     `json:"total_tokens"`
	Cost         float64 `json:"cost,omitempty"`
}

// MCPEvent represents MCP events
type MCPEvent struct {
	Type      string                 `json:"type"`
	ModelID   string                 `json:"model_id,omitempty"`
	ContextID string                 `json:"context_id,omitempty"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

// MCPMetrics represents MCP metrics
type MCPMetrics struct {
	RequestCount   int64         `json:"request_count"`
	SuccessCount   int64         `json:"success_count"`
	ErrorCount     int64         `json:"error_count"`
	AverageLatency time.Duration `json:"average_latency"`
	TotalTokens    int64         `json:"total_tokens"`
	ActiveContexts int           `json:"active_contexts"`
	LoadedModels   int           `json:"loaded_models"`
	MemoryUsage    int64         `json:"memory_usage"`
}

// MCPConfig represents MCP configuration
type MCPConfig struct {
	MaxConcurrentRequests int           `json:"max_concurrent_requests"`
	DefaultTimeout        time.Duration `json:"default_timeout"`
	MaxContextSize        int64         `json:"max_context_size"`
	MaxContexts           int           `json:"max_contexts"`
	EnableCaching         bool          `json:"enable_caching"`
	EnableOptimization    bool          `json:"enable_optimization"`
	LogLevel              string        `json:"log_level"`
}

// EventHandler defines the function signature for MCP event handlers
type EventHandler func(event *MCPEvent) error

// Error types for MCP operations
var (
	ErrModelNotFound    = NewMCPError("model not found")
	ErrContextNotFound  = NewMCPError("context not found")
	ErrInvalidInput     = NewMCPError("invalid input")
	ErrModelLoadFailed  = NewMCPError("model load failed")
	ErrContextFull      = NewMCPError("context full")
	ErrProcessingFailed = NewMCPError("processing failed")
	ErrTimeout          = NewMCPError("request timeout")
)

// MCPError represents an MCP-specific error
type MCPError struct {
	Message string
	Code    string
}

func (e *MCPError) Error() string {
	return e.Message
}

func NewMCPError(message string) *MCPError {
	return &MCPError{Message: message}
}

// ===== MCP CAPABILITY TYPES (Transferred from types/mcp_types.go) =====

// CapabilityType defines the type of MCP primitive.
type CapabilityType string

const (
	CapabilityTypeUnspecified   CapabilityType = "UNSPECIFIED"
	CapabilityTypeResource      CapabilityType = "RESOURCE"
	CapabilityTypeTool          CapabilityType = "TOOL"
	CapabilityTypePrompt        CapabilityType = "PROMPT"
	CapabilityTypeMemoryService CapabilityType = "MEMORY_SERVICE"
)

// IsValidCapabilityType checks if the provided capability type is valid
func IsValidCapabilityType(capType CapabilityType) bool {
	switch capType {
	case CapabilityTypeUnspecified, CapabilityTypeResource, CapabilityTypeTool, CapabilityTypePrompt, CapabilityTypeMemoryService:
		return true
	default:
		return false
	}
}

// BaseDescriptor contains common fields for all MCP capabilities.
type BaseDescriptor struct {
	ID             string                 `json:"id"`             // Unique identifier (e.g., hash of key attributes or UUID)
	CapabilityType CapabilityType         `json:"capabilityType"` // Type of capability (resource/tool/etc)
	Name           string                 `json:"name"`           // Human-readable name
	Owner          string                 `json:"owner"`          // Public key or address of the owner/root
	Version        string                 `json:"version"`
	Description    string                 `json:"description"`
	GasFeeNRN      uint64                 `json:"gasFeeNRN"` // NRN token gas fee for invoking/using this capability
	Timestamp      int64                  `json:"timestamp"` // Creation/registration timestamp
	CustomMetadata map[string]interface{} `json:"customMetadata,omitempty"`
}

// DiscoveryResourceType defines the specific kind of resource for discovery purposes.
type DiscoveryResourceType string

const (
	DiscoveryResourceTypeFile          DiscoveryResourceType = "FILE"
	DiscoveryResourceTypeAPI           DiscoveryResourceType = "API"
	DiscoveryResourceTypePlugin        DiscoveryResourceType = "PLUGIN" // For developer-uploaded plugins
	DiscoveryResourceTypeGeneratedDoc  DiscoveryResourceType = "GENERATED_DOCUMENT"
	DiscoveryResourceTypeDataset       DiscoveryResourceType = "DATASET"        // Replaces original DataCard concept
	DiscoveryResourceTypeModelArtifact DiscoveryResourceType = "MODEL_ARTIFACT" // Replaces original ModelCard concept
	DiscoveryResourceTypeService       DiscoveryResourceType = "SERVICE"        // For network services that are not simple APIs
	ResourceTypeFile                   DiscoveryResourceType = "FILE"
)

// ResourceDescriptor defines metadata for a discoverable resource.
// This includes files, APIs, datasets, model artifacts, and plugins.
type ResourceDescriptor struct {
	BaseDescriptor
	ResourceType DiscoveryResourceType `json:"resourceType"`
	ContentHash  string                `json:"contentHash,omitempty"` // For files/plugins: SHA256 hash of the Capability package.
	Schema       PluginSchemaDetail    `json:"schema,omitempty"`      // Contains summary, location hints, manifest/executable paths for plugins.
}

// PluginSchemaDetail holds structured schema information specific to plugin-type resources.
type PluginSchemaDetail struct {
	Summary             string                 `json:"summary,omitempty"`             // Brief, human-readable summary of the plugin's purpose.
	AccessInfo          map[string]interface{} `json:"accessInfo,omitempty"`          // For APIs: endpoint, auth details, etc.
	LocationHints       []string               `json:"locationHints,omitempty"`       // Array of URIs where the Capability package can be downloaded.
	ManifestFile        string                 `json:"manifestFile,omitempty"`        // Relative path to the manifest file within the downloaded package.
	ExecutableFile      string                 `json:"executableFile,omitempty"`      // Relative path to the main executable file within the downloaded package.
	OutputDirectoryHint string                 `json:"outputDirectoryHint,omitempty"` // Optional: Hint for plugin's output/config directory.
}

// ToolDescriptor defines metadata for an executable tool.
type ToolDescriptor struct {
	BaseDescriptor
	InputSchemaJSON  string `json:"inputSchemaJSON"`  // JSON Schema for tool input
	OutputSchemaJSON string `json:"outputSchemaJSON"` // JSON Schema for tool output
	// ExecutionPointer could be a reference to a Plugin (ResourceID), an API endpoint, or embedded logic.
	ExecutionPointer string `json:"executionPointer,omitempty"`
}

// PromptDescriptor describes an MCP Prompt.
type PromptDescriptor struct {
	BaseDescriptor
	Template             string `json:"template"`             // The prompt template string
	ParametersSchemaJSON string `json:"parametersSchemaJSON"` // JSON Schema for template parameters
}

// MemoryServiceDescriptor describes an MCP Memory Service.
type MemoryServiceDescriptor struct {
	BaseDescriptor
	GraphSchema string `json:"graphSchema,omitempty"` // Describes the structure/ontology of the memory graph if applicable
}

// CapabilityDescriptor defines metadata for a general capability
type CapabilityDescriptor struct {
	BaseDescriptor
	// Additional capability-specific fields can be added here
}

// InteractionType defines the type of interaction with an MCP capability.
type InteractionType string

const (
	InteractionTypeToolInvocation           InteractionType = "TOOL_INVOCATION"
	InteractionTypePromptUsage              InteractionType = "PROMPT_USAGE"
	InteractionTypeResourceAccess           InteractionType = "RESOURCE_ACCESS"
	InteractionTypePluginExecution          InteractionType = "PLUGIN_EXECUTION"
	InteractionTypeSamplingRequestSent      InteractionType = "SAMPLING_REQUEST_SENT"
	InteractionTypeSamplingResponseReceived InteractionType = "SAMPLING_RESPONSE_RECEIVED"
	InteractionTypeMemoryWrite              InteractionType = "MEMORY_WRITE"
	InteractionTypeCapabilityRegistration   InteractionType = "CAPABILITY_REGISTRATION"
	InteractionTypeCapabilityUpdate         InteractionType = "CAPABILITY_UPDATE"
)

// ContextRecord represents a record of an interaction with an MCP capability.
type ContextRecord struct {
	ID                 string                 `json:"id"`                   // Unique identifier for the context record (e.g., transaction ID or a UUID)
	CapabilityID       string                 `json:"capabilityID"`         // Identifier of the MCP capability involved (references BaseDescriptor.ID)
	InteractionType    InteractionType        `json:"interactionType"`      // Type of interaction
	Initiator          string                 `json:"initiator"`            // Public key or address of the entity initiating the context
	Status             string                 `json:"status"`               // Status of the interaction (e.g., "success", "failed")
	InvokerNRN         string                 `json:"invokerNRN"`           // NRN of the invoking entity
	ProviderNRN        string                 `json:"providerNRN"`          // NRN of the capability provider
	Error              string                 `json:"error,omitempty"`      // Error message if interaction failed
	Timestamp          int64                  `json:"timestamp"`            // Timestamp of the event
	TimestampInitiated int64                  `json:"timestampInitiated"`   // Timestamp when interaction was initiated
	TimestampCompleted int64                  `json:"timestampCompleted"`   // Timestamp when interaction was completed
	InputHash          string                 `json:"inputHash,omitempty"`  // Optional hash of the input data to the interaction
	OutputHash         string                 `json:"outputHash,omitempty"` // Optional hash of the output data from the interaction
	GasFeeNRN          uint64                 `json:"gasFeeNRN,omitempty"`  // Gas fee paid in NRN tokens
	Details            map[string]interface{} `json:"details,omitempty"`    // Primitive-specific information
	Signature          []byte                 `json:"signature,omitempty"`  // Cryptographic signature of the ContextRecord content by the initiator
}

// NewContextRecord creates a new ContextRecord with the specified parameters
func NewContextRecord(id, capabilityID string, interactionType InteractionType, initiator string, inputHash, outputHash string, details map[string]interface{}) *ContextRecord {
	return &ContextRecord{
		ID:              id,
		CapabilityID:    capabilityID,
		InteractionType: interactionType,
		Initiator:       initiator,
		Timestamp:       time.Now().Unix(),
		InputHash:       inputHash,
		OutputHash:      outputHash,
		Details:         details,
	}
}

// MCPRegisterCapabilityData holds the data for an MCPRegisterCapability transaction
type MCPRegisterCapabilityData struct {
	CapabilityID         string                 `json:"capabilityID"`         // Unique identifier for the capability
	Name                 string                 `json:"name"`                 // Human-readable name
	Owner                string                 `json:"owner"`                // Public key or address of the owner/root
	Version              string                 `json:"version"`              // Version string
	Description          string                 `json:"description"`          // Description of the capability
	CapabilityType       string                 `json:"capabilityType"`       // Type of capability (resource/tool/etc)
	GasFeeNRN            uint64                 `json:"gasFeeNRN"`            // NRN token gas fee for registration
	CapabilityDescriptor map[string]interface{} `json:"capabilityDescriptor"` // Specific descriptor type (Resource/Tool/etc)
	Descriptor           map[string]interface{} `json:"descriptor,omitempty"` // Deprecated: Use capabilityDescriptor instead
}

// MCPInvokeCapabilityData holds the data for an MCPInvokeCapability transaction
type MCPInvokeCapabilityData struct {
	ContextRecord ContextRecord `json:"contextRecord"`
	// We might also include actual InputData here if it's small and we want it on-chain,
	// though often only InputHash from ContextRecord is used.
}

// MCPUpdateCapabilityData holds the data for an MCPUpdateCapability transaction
type MCPUpdateCapabilityData struct {
	CapabilityID   string      `json:"capabilityID"`   // Unique identifier for the capability
	Name           string      `json:"name"`           // Human-readable name
	Owner          string      `json:"owner"`          // Public key or address of the owner/root
	Version        string      `json:"version"`        // Version string
	Description    string      `json:"description"`    // Description of the capability
	CapabilityType string      `json:"capabilityType"` // Type of capability (resource/tool/etc)
	GasFeeNRN      uint64      `json:"gasFeeNRN"`      // NRN token gas fee for update
	Descriptor     interface{} `json:"descriptor"`     // Specific descriptor type (Resource/Tool/etc)
}

// ConvertMapToCapability converts a map[string]interface{} (from JSON) to a concrete Capability type.
// This is used when processing JSON fallback data in the transaction pipeline.
func ConvertMapToCapability(data map[string]interface{}) (interface{}, error) {
	capType, ok := data["capabilityType"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid capabilityType")
	}

	switch CapabilityType(capType) {
	case CapabilityTypeResource:
		var desc ResourceDescriptor
		if err := mapToStruct(data, &desc); err != nil {
			return nil, fmt.Errorf("resource descriptor conversion failed: %w", err)
		}
		return &desc, nil

	case CapabilityTypeTool:
		var desc ToolDescriptor
		if err := mapToStruct(data, &desc); err != nil {
			return nil, fmt.Errorf("tool descriptor conversion failed: %w", err)
		}
		return &desc, nil

	case CapabilityTypePrompt:
		var desc PromptDescriptor
		if err := mapToStruct(data, &desc); err != nil {
			return nil, fmt.Errorf("prompt descriptor conversion failed: %w", err)
		}
		return &desc, nil

	case CapabilityTypeMemoryService:
		var desc MemoryServiceDescriptor
		if err := mapToStruct(data, &desc); err != nil {
			return nil, fmt.Errorf("memory service descriptor conversion failed: %w", err)
		}
		return &desc, nil

	default:
		return nil, fmt.Errorf("unsupported capability type: %s", capType)
	}
}

// mapToStruct converts a map to a struct using JSON marshaling/unmarshaling
func mapToStruct(input map[string]interface{}, output interface{}) error {
	jsonData, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("marshal failed: %w", err)
	}
	if err := json.Unmarshal(jsonData, output); err != nil {
		return fmt.Errorf("unmarshal failed: %w", err)
	}
	return nil
}
