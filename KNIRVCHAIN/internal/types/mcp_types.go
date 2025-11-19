package types

import (
	"encoding/json"
	"fmt"
	"time"
)

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
	// Removed: LocationHints (moved into Schema)
	// Removed: InputSchemaJSON (role absorbed by Schema.ManifestFile, Schema.ExecutableFile)
	// Removed: OutputSchemaJSON (role absorbed by Schema.OutputDirectoryHint)
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
	// Other relevant details for interacting with the memory service
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

// QueryEnum defines options for what to include in ChromemDB query results
type QueryEnum string

const (
	IncludeMetadatas QueryEnum = "INCLUDE_METADATAS"
	IncludeDistances QueryEnum = "INCLUDE_DISTANCES"
	IncludeDocuments QueryEnum = "INCLUDE_DOCUMENTS"
)

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
