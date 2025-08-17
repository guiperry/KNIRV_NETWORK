// agent_plugin.go
package agentify

import (
	"context"
	"time"
)

// AgentPlugin represents a plugin that acts as an agent
type AgentPlugin interface {
	// Initialize the agent with configuration
	Initialize(config map[string]interface{}) error

	// Process an inference request and return a response
	ProcessInference(ctx context.Context, request *InferenceRequest) (*InferenceResponse, error)

	// Get the agent's capabilities
	GetCapabilities() *AgentCapabilities

	// Get the agent's schema (tools, resources, prompts)
	GetSchema() *AgentSchema

	// Memory management
	GetMemory(key string) (interface{}, error)
	SetMemory(key string, value interface{}) error

	// Lifecycle management
	Start() error
	Stop() error
}

// InferenceRequest represents a request to the agent
type InferenceRequest struct {
	// The input text from the user or system
	Input string `json:"input"`

	// The conversation history
	History []*ConversationMessage `json:"history,omitempty"`

	// The session ID for tracking conversation state
	SessionID string `json:"sessionId"`

	// Additional parameters for the inference
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// InferenceResponse represents a response from the agent
type InferenceResponse struct {
	// The output text from the agent
	Output string `json:"output"`

	// The tool calls made during inference
	ToolCalls []*ToolCall `json:"toolCalls,omitempty"`

	// The reasoning trace (if enabled)
	Reasoning string `json:"reasoning,omitempty"`

	// Additional metadata about the response
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ConversationMessage represents a message in the conversation history
type ConversationMessage struct {
	// The role of the message sender (user, assistant, system)
	Role string `json:"role"`

	// The content of the message
	Content string `json:"content"`

	// The timestamp of the message
	Timestamp int64 `json:"timestamp"`
}

// ToolCall represents a call to a tool during inference
type ToolCall struct {
	// The name of the tool
	Name string `json:"name"`

	// The input to the tool
	Input map[string]interface{} `json:"input"`

	// The output from the tool
	Output interface{} `json:"output"`

	// The timestamp of the tool call
	Timestamp int64 `json:"timestamp"`
}

// AgentCapabilities represents the capabilities of an agent
type AgentCapabilities struct {
	// Whether the agent supports streaming responses
	SupportsStreaming bool `json:"supportsStreaming"`

	// Whether the agent supports tool calls
	SupportsToolCalls bool `json:"supportsToolCalls"`

	// Whether the agent supports reasoning traces
	SupportsReasoning bool `json:"supportsReasoning"`

	// The maximum context length supported by the agent
	MaxContextLength int `json:"maxContextLength"`

	// The supported inference parameters
	SupportedParameters []string `json:"supportedParameters"`
}

// AgentSchema represents the schema of an agent
type AgentSchema struct {
	// The tools available to the agent
	Tools []*ToolSchema `json:"tools"`

	// The resources available to the agent
	Resources []*ResourceSchema `json:"resources"`

	// The prompts available to the agent
	Prompts []*PromptSchema `json:"prompts"`
}

// ToolSchema represents the schema of a tool
type ToolSchema struct {
	// The name of the tool
	Name string `json:"name"`

	// The description of the tool
	Description string `json:"description"`

	// The parameters of the tool
	Parameters map[string]*ParameterSchema `json:"parameters"`

	// The return type of the tool
	ReturnType string `json:"returnType"`
}

// ResourceSchema represents the schema of a resource
type ResourceSchema struct {
	// The name of the resource
	Name string `json:"name"`

	// The type of the resource
	Type string `json:"type"`

	// The description of the resource
	Description string `json:"description"`
}

// PromptSchema represents the schema of a prompt
type PromptSchema struct {
	// The name of the prompt
	Name string `json:"name"`

	// The description of the prompt
	Description string `json:"description"`

	// The variables in the prompt
	Variables []string `json:"variables"`
}

// ParameterSchema represents the schema of a parameter
type ParameterSchema struct {
	// The type of the parameter
	Type string `json:"type"`

	// The description of the parameter
	Description string `json:"description"`

	// Whether the parameter is required
	Required bool `json:"required"`

	// The default value of the parameter
	DefaultValue interface{} `json:"defaultValue,omitempty"`
}

// NewConversationMessage creates a new conversation message with the current timestamp
func NewConversationMessage(role, content string) *ConversationMessage {
	return &ConversationMessage{
		Role:      role,
		Content:   content,
		Timestamp: time.Now().Unix(),
	}
}

// NewToolCall creates a new tool call with the current timestamp
func NewToolCall(name string, input map[string]interface{}, output interface{}) *ToolCall {
	return &ToolCall{
		Name:      name,
		Input:     input,
		Output:    output,
		Timestamp: time.Now().Unix(),
	}
}
