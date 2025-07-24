package main

import (
	"context"
	"encoding/json"
	"fmt"
	"plugin"
	"time"

	"Agentic_Engine/agentify"
)

// CGOPluginWrapper wraps a CGO-based plugin to work with Agent Inferencer
type CGOPluginWrapper struct {
	*agentify.BaseAgentPlugin
	cgoPlugin      *plugin.Plugin
	initializeFunc func() string
	processFunc    func(string) string
	shutdownFunc   func() string
	isInitialized  bool
	isRunning      bool
}

// NewCGOPluginWrapper creates a new wrapper for CGO plugins
func NewCGOPluginWrapper(pluginPath string) (*CGOPluginWrapper, error) {
	// Load the CGO plugin
	p, err := plugin.Open(pluginPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load CGO plugin: %v", err)
	}

	wrapper := &CGOPluginWrapper{
		BaseAgentPlugin: agentify.NewBaseAgentPlugin(),
		cgoPlugin:       p,
	}

	// Look up the CGO functions
	if initSym, err := p.Lookup("Initialize"); err == nil {
		if initFunc, ok := initSym.(func() string); ok {
			wrapper.initializeFunc = initFunc
		}
	}

	if procSym, err := p.Lookup("ProcessMessage"); err == nil {
		if procFunc, ok := procSym.(func(string) string); ok {
			wrapper.processFunc = procFunc
		}
	}

	if shutSym, err := p.Lookup("Shutdown"); err == nil {
		if shutFunc, ok := shutSym.(func() string); ok {
			wrapper.shutdownFunc = shutFunc
		}
	}

	return wrapper, nil
}

// Initialize initializes the CGO plugin wrapper
func (w *CGOPluginWrapper) Initialize(config map[string]interface{}) error {
	// Initialize the base plugin first
	if err := w.BaseAgentPlugin.Initialize(config); err != nil {
		return err
	}

	// Initialize the CGO plugin if available
	if w.initializeFunc != nil {
		result := w.initializeFunc()
		fmt.Printf("CGO Plugin Initialize result: %s\n", result)
	}

	w.isInitialized = true
	return nil
}

// Start starts the CGO plugin wrapper
func (w *CGOPluginWrapper) Start() error {
	if !w.isInitialized {
		return fmt.Errorf("plugin not initialized")
	}

	// Start the base plugin
	if err := w.BaseAgentPlugin.Start(); err != nil {
		return err
	}

	w.isRunning = true
	return nil
}

// Stop stops the CGO plugin wrapper
func (w *CGOPluginWrapper) Stop() error {
	if !w.isRunning {
		return nil
	}

	// Shutdown the CGO plugin if available
	if w.shutdownFunc != nil {
		result := w.shutdownFunc()
		fmt.Printf("CGO Plugin Shutdown result: %s\n", result)
	}

	// Stop the base plugin
	if err := w.BaseAgentPlugin.Stop(); err != nil {
		return err
	}

	w.isRunning = false
	return nil
}

// ProcessInference processes an inference request through the CGO plugin
func (w *CGOPluginWrapper) ProcessInference(ctx context.Context, request *agentify.InferenceRequest) (*agentify.InferenceResponse, error) {
	if !w.isRunning {
		return nil, fmt.Errorf("plugin not running")
	}

	// If we have a CGO process function, use it
	if w.processFunc != nil {
		// Convert the request to JSON for the CGO plugin
		requestData := map[string]interface{}{
			"input":      request.Input,
			"sessionId":  request.SessionID,
			"parameters": request.Parameters,
			"history":    request.History,
		}

		requestJSON, err := json.Marshal(requestData)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %v", err)
		}

		// Call the CGO plugin
		result := w.processFunc(string(requestJSON))

		// Try to parse the result as JSON
		var responseData map[string]interface{}
		if err := json.Unmarshal([]byte(result), &responseData); err != nil {
			// If it's not JSON, treat it as plain text output
			return &agentify.InferenceResponse{
				Output:    result,
				Reasoning: "Processed by CGO plugin",
				Metadata: map[string]interface{}{
					"cgo_plugin": true,
					"timestamp":  time.Now().Unix(),
				},
			}, nil
		}

		// Convert the parsed response
		response := &agentify.InferenceResponse{
			Metadata: map[string]interface{}{
				"cgo_plugin": true,
				"timestamp":  time.Now().Unix(),
			},
		}

		if output, ok := responseData["output"].(string); ok {
			response.Output = output
		}

		if reasoning, ok := responseData["reasoning"].(string); ok {
			response.Reasoning = reasoning
		}

		// Handle tool calls if present
		if toolCalls, ok := responseData["toolCalls"].([]interface{}); ok {
			for _, tc := range toolCalls {
				if toolCall, ok := tc.(map[string]interface{}); ok {
					name, _ := toolCall["name"].(string)
					input, _ := toolCall["input"].(map[string]interface{})
					output := toolCall["output"]

					response.ToolCalls = append(response.ToolCalls, &agentify.ToolCall{
						Name:      name,
						Input:     input,
						Output:    output,
						Timestamp: time.Now().Unix(),
					})
				}
			}
		}

		return response, nil
	}

	// Fall back to base plugin processing
	return w.BaseAgentPlugin.ProcessInference(ctx, request)
}

// GetCapabilities returns the capabilities of the CGO plugin wrapper
func (w *CGOPluginWrapper) GetCapabilities() *agentify.AgentCapabilities {
	return &agentify.AgentCapabilities{
		SupportsStreaming:   false, // CGO plugins typically don't support streaming
		SupportsToolCalls:   true,
		SupportsReasoning:   true,
		MaxContextLength:    8192,
		SupportedParameters: []string{"temperature", "max_tokens"},
	}
}

// GetSchema returns the schema of the CGO plugin wrapper
func (w *CGOPluginWrapper) GetSchema() *agentify.AgentSchema {
	// For CGO plugins, we provide a basic schema
	// In a real implementation, this could be loaded from plugin metadata
	return &agentify.AgentSchema{
		Tools: []*agentify.ToolSchema{
			{
				Name:        "process_message",
				Description: "Process a message through the CGO plugin",
				Parameters: map[string]*agentify.ParameterSchema{
					"message": {
						Type:        "string",
						Description: "The message to process",
						Required:    true,
					},
				},
				ReturnType: "string",
			},
		},
		Resources: []*agentify.ResourceSchema{
			{
				Name:        "cgo_plugin",
				Type:        "external",
				Description: "CGO-based plugin resource",
			},
		},
		Prompts: []*agentify.PromptSchema{
			{
				Name:        "default",
				Description: "Default prompt for CGO plugin",
				Variables:   []string{"input", "context"},
			},
		},
	}
}

// Export the plugin instance
var Plugin agentify.AgentPluginInterface

func init() {
	// This would be set when the wrapper is created
	// For now, we'll use the base plugin
	Plugin = agentify.NewBaseAgentPlugin()
}

// CreateCGOWrapper creates a CGO wrapper plugin
func CreateCGOWrapper(pluginPath string) (agentify.AgentPluginInterface, error) {
	return NewCGOPluginWrapper(pluginPath)
}

// main function for testing
func main() {
	fmt.Println("CGO Plugin Wrapper")
	fmt.Println("This file demonstrates how to wrap CGO plugins for use with Agent Inferencer")
}
