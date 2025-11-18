// agent_plugin_impl.go
package agentify

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sync"
	"time"
)

// BaseAgentPlugin provides a base implementation of the AgentPlugin interface
type BaseAgentPlugin struct {
	config      map[string]interface{}
	tools       map[string]ToolFunc
	resources   map[string]interface{}
	prompts     map[string]string
	memory      map[string]interface{}
	tee         TEE
	mutex       sync.RWMutex
	initialized bool
	running     bool
}

// ToolFunc represents a function that implements a tool
type ToolFunc func(ctx context.Context, params map[string]interface{}) (interface{}, error)

// NewBaseAgentPlugin creates a new base agent plugin
func NewBaseAgentPlugin() *BaseAgentPlugin {
	return &BaseAgentPlugin{
		tools:     make(map[string]ToolFunc),
		resources: make(map[string]interface{}),
		prompts:   make(map[string]string),
		memory:    make(map[string]interface{}),
	}
}

// Initialize initializes the agent with configuration
func (p *BaseAgentPlugin) Initialize(config map[string]interface{}) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.initialized {
		return nil
	}

	p.config = config

	// Initialize the TEE
	teeConfig, ok := config["tee"].(map[string]interface{})
	if !ok {
		// Use default TEE configuration if not provided
		teeConfig = map[string]interface{}{
			"isolationLevel": "process",
		}
	}

	// Create the appropriate TEE based on the configuration
	isolationLevel, _ := teeConfig["isolationLevel"].(string)
	switch isolationLevel {
	case "process":
		p.tee = NewProcessTEE(TEEConfig{
			// Configure from teeConfig
			WorkingDir: teeConfig["workingDir"].(string),
			Env:        teeConfig["env"].(map[string]string),
		})
	case "container":
		p.tee = NewContainerTEE(TEEConfig{
			// Configure from teeConfig
			Image:      teeConfig["image"].(string),
			Tag:        teeConfig["tag"].(string),
			WorkingDir: teeConfig["workingDir"].(string),
			Env:        teeConfig["env"].(map[string]string),
		})
	case "vm":
		p.tee = NewVMTEE(TEEConfig{
			// Configure from teeConfig
			Image:      teeConfig["image"].(string),
			Memory:     teeConfig["memory"].(int),
			CPU:        teeConfig["cpu"].(int),
			WorkingDir: teeConfig["workingDir"].(string),
			Env:        teeConfig["env"].(map[string]string),
		})
	default:
		// Default to process TEE
		p.tee = NewProcessTEE(TEEConfig{
			WorkingDir: "/tmp",
			Env:        map[string]string{},
		})
	}

	// Register tools
	tools, ok := config["tools"].([]interface{})
	if ok {
		for _, toolConfig := range tools {
			toolMap, ok := toolConfig.(map[string]interface{})
			if !ok {
				continue
			}

			name, _ := toolMap["name"].(string)
			implementation, _ := toolMap["implementation"].(string)

			// Register the tool
			p.RegisterTool(name, func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
				// Convert params to JSON
				paramsJSON, err := json.Marshal(params)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal tool parameters: %v", err)
				}

				// Execute the tool implementation in the TEE
				stdout, stderr, exitCode, err := p.tee.Execute("go", []string{"run", "-e", implementation, string(paramsJSON)})
				if err != nil {
					return nil, err
				}

				if exitCode != 0 {
					return nil, fmt.Errorf("tool execution failed: %s", stderr)
				}

				// Parse the output as JSON
				var result interface{}
				if err := json.Unmarshal([]byte(stdout), &result); err != nil {
					return stdout, nil // Return as string if not JSON
				}

				return result, nil
			})
		}
	}

	// Load resources
	resources, ok := config["resources"].([]interface{})
	if ok {
		for _, resourceConfig := range resources {
			resourceMap, ok := resourceConfig.(map[string]interface{})
			if !ok {
				continue
			}

			name, _ := resourceMap["name"].(string)
			content, _ := resourceMap["content"]

			// Store the resource
			p.resources[name] = content
		}
	}

	// Load prompts
	prompts, ok := config["prompts"].([]interface{})
	if ok {
		for _, promptConfig := range prompts {
			promptMap, ok := promptConfig.(map[string]interface{})
			if !ok {
				continue
			}

			name, _ := promptMap["name"].(string)
			content, _ := promptMap["content"].(string)

			// Store the prompt
			p.prompts[name] = content
		}
	}

	p.initialized = true
	return nil
}

// RegisterTool registers a tool with the agent
func (p *BaseAgentPlugin) RegisterTool(name string, tool ToolFunc) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.tools[name] = tool
}

// ProcessInference processes an inference request
func (p *BaseAgentPlugin) ProcessInference(ctx context.Context, request *InferenceRequest) (*InferenceResponse, error) {
	p.mutex.RLock()
	if !p.initialized || !p.running {
		p.mutex.RUnlock()
		return nil, fmt.Errorf("agent not initialized or not running")
	}
	p.mutex.RUnlock()

	// Create the system prompt
	systemPrompt, ok := p.prompts["system"]
	if !ok {
		systemPrompt = "You are a helpful AI assistant."
	}

	// Create the conversation history
	history := request.History
	if history == nil {
		history = []*ConversationMessage{
			{
				Role:      "system",
				Content:   systemPrompt,
				Timestamp: time.Now().Unix(),
			},
		}
	}

	// Add the user's input to the history
	history = append(history, &ConversationMessage{
		Role:      "user",
		Content:   request.Input,
		Timestamp: time.Now().Unix(),
	})

	// Create the inference payload
	payload := map[string]interface{}{
		"messages": history,
		"tools":    p.getToolsForLLM(),
	}

	// Add any additional parameters
	for k, v := range request.Parameters {
		payload[k] = v
	}

	// Convert the payload to JSON
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal inference payload: %v", err)
	}

	// Execute the inference in the TEE
	stdout, stderr, exitCode, err := p.tee.Execute("python", []string{"-c", fmt.Sprintf(`
import json
import os
import sys
from google_adk import Agent

# Load the payload
payload = json.loads('%s')

# Create the agent
agent = Agent()

# Process the inference
response = agent.process(payload)

# Print the response
print(json.dumps(response))
	`, string(payloadJSON))})

	if err != nil {
		return nil, fmt.Errorf("inference execution failed: %v", err)
	}

	if exitCode != 0 {
		return nil, fmt.Errorf("inference execution failed: %s", stderr)
	}

	// Parse the response
	var response InferenceResponse
	if err := json.Unmarshal([]byte(stdout), &response); err != nil {
		return nil, fmt.Errorf("failed to parse inference response: %v", err)
	}

	return &response, nil
}

// getToolsForLLM converts the tools to the format expected by the LLM
func (p *BaseAgentPlugin) getToolsForLLM() []map[string]interface{} {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	tools := make([]map[string]interface{}, 0, len(p.tools))

	schema := p.GetSchema()
	for _, toolSchema := range schema.Tools {
		tool := map[string]interface{}{
			"name":        toolSchema.Name,
			"description": toolSchema.Description,
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": func() map[string]interface{} {
					props := make(map[string]interface{})
					for name, param := range toolSchema.Parameters {
						props[name] = map[string]interface{}{
							"type":        param.Type,
							"description": param.Description,
						}
					}
					return props
				}(),
				"required": func() []string {
					required := make([]string, 0)
					for name, param := range toolSchema.Parameters {
						if param.Required {
							required = append(required, name)
						}
					}
					return required
				}(),
			},
		}

		tools = append(tools, tool)
	}

	return tools
}

// GetCapabilities gets the agent's capabilities
func (p *BaseAgentPlugin) GetCapabilities() *AgentCapabilities {
	return &AgentCapabilities{
		SupportsStreaming:   true,
		SupportsToolCalls:   true,
		SupportsReasoning:   true,
		MaxContextLength:    16384,
		SupportedParameters: []string{"temperature", "top_p", "max_tokens"},
	}
}

// GetSchema gets the agent's schema
func (p *BaseAgentPlugin) GetSchema() *AgentSchema {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	// Create the tool schemas
	tools := make([]*ToolSchema, 0, len(p.tools))
	for name := range p.tools {
		// In a real implementation, we would extract parameter information
		// from the tool function signature or configuration
		tools = append(tools, &ToolSchema{
			Name:        name,
			Description: fmt.Sprintf("Tool for %s", name),
			Parameters:  make(map[string]*ParameterSchema),
			ReturnType:  "object",
		})
	}

	// Create the resource schemas
	resources := make([]*ResourceSchema, 0, len(p.resources))
	for name := range p.resources {
		resources = append(resources, &ResourceSchema{
			Name:        name,
			Type:        "object",
			Description: fmt.Sprintf("Resource for %s", name),
		})
	}

	// Create the prompt schemas
	prompts := make([]*PromptSchema, 0, len(p.prompts))
	for name, content := range p.prompts {
		// Extract variables from the prompt template (e.g., {{variable}})
		var variables []string
		re := regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_]+)\s*\}\}`)
		matches := re.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if len(match) > 1 {
				variables = append(variables, match[1])
			}
		}

		prompts = append(prompts, &PromptSchema{
			Name:        name,
			Description: fmt.Sprintf("Prompt for %s", name),
			Variables:   variables,
		})
	}

	return &AgentSchema{
		Tools:     tools,
		Resources: resources,
		Prompts:   prompts,
	}
}

// GetMemory gets a value from the agent's memory
func (p *BaseAgentPlugin) GetMemory(key string) (interface{}, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	value, ok := p.memory[key]
	if !ok {
		return nil, fmt.Errorf("key not found in memory: %s", key)
	}

	return value, nil
}

// SetMemory sets a value in the agent's memory
func (p *BaseAgentPlugin) SetMemory(key string, value interface{}) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.memory[key] = value
	return nil
}

// Start starts the agent
func (p *BaseAgentPlugin) Start() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !p.initialized {
		return fmt.Errorf("agent not initialized")
	}

	if p.running {
		return nil
	}

	// Start the TEE
	if err := p.tee.Start(); err != nil {
		return fmt.Errorf("failed to start TEE: %v", err)
	}

	p.running = true
	return nil
}

// Stop stops the agent
func (p *BaseAgentPlugin) Stop() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !p.running {
		return nil
	}

	// Stop the TEE
	if err := p.tee.Stop(); err != nil {
		return fmt.Errorf("failed to stop TEE: %v", err)
	}

	p.running = false
	return nil
}

// GetTEEInfo gets information about the TEE
func (p *BaseAgentPlugin) GetTEEInfo() map[string]interface{} {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	if p.tee == nil {
		return map[string]interface{}{
			"status": "not_initialized",
		}
	}

	return p.tee.GetInfo()
}

// Legacy memory management methods for backward compatibility

// StoreContext stores a context
func (p *BaseAgentPlugin) StoreContext(contextID string, context map[string]interface{}) error {
	return p.SetMemory("context:"+contextID, context)
}

// GetContext gets a context
func (p *BaseAgentPlugin) GetContext(contextID string) (map[string]interface{}, error) {
	value, err := p.GetMemory("context:" + contextID)
	if err != nil {
		return nil, err
	}

	context, ok := value.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("value is not a context: %s", contextID)
	}

	return context, nil
}

// TransferContext transfers a context to another agent
func (p *BaseAgentPlugin) TransferContext(contextID string, targetAgentID string) error {
	// In a real implementation, we would transfer the context to another agent
	return nil
}

// StoreCredential stores a credential
func (p *BaseAgentPlugin) StoreCredential(credentialID string, credential map[string]interface{}) error {
	return p.SetMemory("credential:"+credentialID, credential)
}

// GetCredential gets a credential
func (p *BaseAgentPlugin) GetCredential(credentialID string) (map[string]interface{}, error) {
	value, err := p.GetMemory("credential:" + credentialID)
	if err != nil {
		return nil, err
	}

	credential, ok := value.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("value is not a credential: %s", credentialID)
	}

	return credential, nil
}

// StoreRAGResult stores a RAG result
func (p *BaseAgentPlugin) StoreRAGResult(queryHash string, result map[string]interface{}, ttl int64) error {
	return p.SetMemory("rag:"+queryHash, map[string]interface{}{
		"result": result,
		"ttl":    ttl,
		"time":   time.Now().Unix(),
	})
}

// GetRAGResult gets a RAG result
func (p *BaseAgentPlugin) GetRAGResult(queryHash string) (map[string]interface{}, error) {
	value, err := p.GetMemory("rag:" + queryHash)
	if err != nil {
		return nil, err
	}

	ragResult, ok := value.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("value is not a RAG result: %s", queryHash)
	}

	// Check if the result has expired
	ttl, _ := ragResult["ttl"].(int64)
	timestamp, _ := ragResult["time"].(int64)
	if ttl > 0 && time.Now().Unix() > timestamp+ttl {
		// Remove the expired result
		p.mutex.Lock()
		delete(p.memory, "rag:"+queryHash)
		p.mutex.Unlock()

		return nil, fmt.Errorf("RAG result expired: %s", queryHash)
	}

	result, _ := ragResult["result"].(map[string]interface{})
	return result, nil
}

// StoreCOTPlan stores a COT plan
func (p *BaseAgentPlugin) StoreCOTPlan(planID string, plan map[string]interface{}) error {
	return p.SetMemory("cot:"+planID, plan)
}

// GetCOTPlan gets a COT plan
func (p *BaseAgentPlugin) GetCOTPlan(planID string) (map[string]interface{}, error) {
	value, err := p.GetMemory("cot:" + planID)
	if err != nil {
		return nil, err
	}

	plan, ok := value.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("value is not a COT plan: %s", planID)
	}

	return plan, nil
}

// StoreUserPreference stores a user preference
func (p *BaseAgentPlugin) StoreUserPreference(userID string, preference map[string]interface{}) error {
	return p.SetMemory("user:"+userID, preference)
}

// GetUserPreferences gets all user preferences
func (p *BaseAgentPlugin) GetUserPreferences(userID string) (map[string]interface{}, error) {
	value, err := p.GetMemory("user:" + userID)
	if err != nil {
		return nil, err
	}

	preferences, ok := value.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("value is not a user preference: %s", userID)
	}

	return preferences, nil
}

// GetUserPreference gets a specific user preference
func (p *BaseAgentPlugin) GetUserPreference(userID string, key string) (interface{}, error) {
	preferences, err := p.GetUserPreferences(userID)
	if err != nil {
		return nil, err
	}

	value, ok := preferences[key]
	if !ok {
		return nil, fmt.Errorf("preference not found: %s", key)
	}

	return value, nil
}
