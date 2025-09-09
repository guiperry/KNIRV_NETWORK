// comprehensive_agent_plugin.go
package main

import (
	"KNIRVENGINE/desktop-client/agentify"
	"KNIRVENGINE/desktop-client/utils"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ComprehensiveAgentPlugin demonstrates all capabilities of the Agent Inferencer system
type ComprehensiveAgentPlugin struct {
	*agentify.BaseAgentPlugin
	workingDir string
}

// Plugin is the exported symbol that will be loaded by the AgentPluginLoader
var Plugin = &ComprehensiveAgentPlugin{
	BaseAgentPlugin: agentify.NewBaseAgentPlugin(),
}

// Initialize initializes the agent with configuration
func (p *ComprehensiveAgentPlugin) Initialize(config map[string]interface{}) error {
	// Call the base implementation
	if err := p.BaseAgentPlugin.Initialize(config); err != nil {
		return err
	}

	// Set up working directory
	if workDir, ok := config["workingDir"].(string); ok {
		p.workingDir = workDir
	} else {
		// Use system-specific temp workspace directory
		tempWorkspace, err := utils.GetTempWorkspaceDir()
		if err != nil {
			// Fallback to /tmp/agent_workspace if utils package is not available
			p.workingDir = "/tmp/agent_workspace"
		} else {
			p.workingDir = tempWorkspace
		}
	}

	// Create working directory if it doesn't exist
	if err := utils.EnsureDir(p.workingDir); err != nil {
		// Fallback to os.MkdirAll if utils package is not available
		if err := os.MkdirAll(p.workingDir, 0755); err != nil {
			return fmt.Errorf("failed to create working directory: %v", err)
		}
	}

	// Register comprehensive tools
	p.RegisterTool("file_operations", p.fileOperationsTool)
	p.RegisterTool("web_search", p.webSearchTool)
	p.RegisterTool("calculator", p.calculatorTool)
	p.RegisterTool("system_info", p.systemInfoTool)
	p.RegisterTool("memory_search", p.memorySearchTool)
	p.RegisterTool("terminal_command", p.terminalCommandTool)

	// Set up resources
	p.SetMemory("resource:faq", map[string]interface{}{
		"content": "Frequently Asked Questions about the Comprehensive Agent",
		"type":    "text",
		"data": []map[string]string{
			{"question": "What can this agent do?", "answer": "This agent can perform file operations, web searches, calculations, system information gathering, memory management, and terminal commands."},
			{"question": "How do I use the terminal?", "answer": "You can create a terminal session and execute commands through the terminal_command tool or directly via the terminal API."},
			{"question": "Is my data persistent?", "answer": "Yes, the agent uses vector-based memory storage for persistent data across sessions."},
		},
	})

	// Set up prompts
	p.SetMemory("prompt:system", "You are a comprehensive AI agent with access to file operations, web search, calculations, system information, memory management, and terminal access. You can help users with a wide variety of tasks. Always be helpful, accurate, and secure in your operations.")
	p.SetMemory("prompt:greeting", "Hello {{name}}! I'm your comprehensive AI agent. I can help you with file operations, web searches, calculations, system information, memory management, and terminal commands. What would you like to do today?")
	p.SetMemory("prompt:error", "I encountered an error: {{error}}. Let me try to help you resolve this issue.")

	return nil
}

// fileOperationsTool handles file system operations
func (p *ComprehensiveAgentPlugin) fileOperationsTool(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	operation, ok := params["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid operation parameter")
	}

	switch operation {
	case "list":
		path, ok := params["path"].(string)
		if !ok {
			path = p.workingDir
		}

		files, err := os.ReadDir(path)
		if err != nil {
			return nil, fmt.Errorf("failed to list directory: %v", err)
		}

		result := make([]map[string]interface{}, 0, len(files))
		for _, file := range files {
			info, _ := file.Info()
			result = append(result, map[string]interface{}{
				"name":    file.Name(),
				"isDir":   file.IsDir(),
				"size":    info.Size(),
				"modTime": info.ModTime().Format(time.RFC3339),
			})
		}

		return map[string]interface{}{
			"operation": "list",
			"path":      path,
			"files":     result,
			"count":     len(result),
		}, nil

	case "read":
		filePath, ok := params["path"].(string)
		if !ok {
			return nil, fmt.Errorf("missing path parameter for read operation")
		}

		// Ensure the path is within the working directory for security
		if !strings.HasPrefix(filepath.Clean(filePath), filepath.Clean(p.workingDir)) {
			filePath = filepath.Join(p.workingDir, filepath.Base(filePath))
		}

		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read file: %v", err)
		}

		return map[string]interface{}{
			"operation": "read",
			"path":      filePath,
			"content":   string(content),
			"size":      len(content),
		}, nil

	case "write":
		filePath, ok := params["path"].(string)
		if !ok {
			return nil, fmt.Errorf("missing path parameter for write operation")
		}

		content, ok := params["content"].(string)
		if !ok {
			return nil, fmt.Errorf("missing content parameter for write operation")
		}

		// Ensure the path is within the working directory for security
		if !strings.HasPrefix(filepath.Clean(filePath), filepath.Clean(p.workingDir)) {
			filePath = filepath.Join(p.workingDir, filepath.Base(filePath))
		}

		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to write file: %v", err)
		}

		return map[string]interface{}{
			"operation": "write",
			"path":      filePath,
			"size":      len(content),
			"success":   true,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported file operation: %s", operation)
	}
}

// webSearchTool performs web searches (mock implementation)
func (p *ComprehensiveAgentPlugin) webSearchTool(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	query, ok := params["query"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid query parameter")
	}

	// Mock web search results
	return map[string]interface{}{
		"query": query,
		"results": []map[string]interface{}{
			{
				"title":       "Example Search Result 1",
				"description": fmt.Sprintf("This is a mock search result for: %s", query),
				"url":         "https://example.com/result1",
				"score":       0.95,
			},
			{
				"title":       "Example Search Result 2",
				"description": fmt.Sprintf("Another mock result for: %s", query),
				"url":         "https://example.com/result2",
				"score":       0.87,
			},
		},
		"total":     2,
		"timestamp": time.Now().Format(time.RFC3339),
	}, nil
}

// calculatorTool performs mathematical calculations
func (p *ComprehensiveAgentPlugin) calculatorTool(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	expression, ok := params["expression"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid expression parameter")
	}

	// Simple calculator implementation (mock)
	// In a real implementation, you would use a proper expression parser
	result := 0.0
	operation := "unknown"

	// Basic parsing for demonstration
	if strings.Contains(expression, "+") {
		operation = "addition"
		result = 42.0 // Mock result
	} else if strings.Contains(expression, "-") {
		operation = "subtraction"
		result = 10.0
	} else if strings.Contains(expression, "*") {
		operation = "multiplication"
		result = 100.0
	} else if strings.Contains(expression, "/") {
		operation = "division"
		result = 5.0
	} else {
		result = 42.0 // Default mock result
	}

	return map[string]interface{}{
		"expression": expression,
		"result":     result,
		"operation":  operation,
		"timestamp":  time.Now().Format(time.RFC3339),
	}, nil
}

// systemInfoTool provides system information
func (p *ComprehensiveAgentPlugin) systemInfoTool(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	infoType, ok := params["type"].(string)
	if !ok {
		infoType = "general"
	}

	switch infoType {
	case "general":
		return map[string]interface{}{
			"hostname":   "agent-host",
			"platform":   "linux",
			"arch":       "amd64",
			"workingDir": p.workingDir,
			"timestamp":  time.Now().Format(time.RFC3339),
		}, nil

	case "memory":
		return map[string]interface{}{
			"type":      "memory",
			"total":     "8GB",
			"available": "4GB",
			"used":      "4GB",
			"timestamp": time.Now().Format(time.RFC3339),
		}, nil

	case "disk":
		return map[string]interface{}{
			"type":      "disk",
			"total":     "100GB",
			"available": "50GB",
			"used":      "50GB",
			"timestamp": time.Now().Format(time.RFC3339),
		}, nil

	default:
		return nil, fmt.Errorf("unsupported system info type: %s", infoType)
	}
}

// memorySearchTool searches the agent's memory
func (p *ComprehensiveAgentPlugin) memorySearchTool(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	query, ok := params["query"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid query parameter")
	}

	// Try to search using vector memory if available
	memoryManager := p.BaseAgentPlugin.GetMemoryManager()
	if memoryManager != nil {
		if vectorMemory, ok := memoryManager.(*agentify.VectorMemoryManager); ok {
			results, err := vectorMemory.Search(query, 5)
			if err != nil {
				return nil, fmt.Errorf("memory search failed: %v", err)
			}

			return map[string]interface{}{
				"query":     query,
				"results":   results,
				"count":     len(results),
				"timestamp": time.Now().Format(time.RFC3339),
			}, nil
		}
	}

	// Fall back to simple memory search
	return map[string]interface{}{
		"query":     query,
		"results":   []interface{}{},
		"count":     0,
		"message":   "Vector memory not available, no results found",
		"timestamp": time.Now().Format(time.RFC3339),
	}, nil
}

// terminalCommandTool executes terminal commands
func (p *ComprehensiveAgentPlugin) terminalCommandTool(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	command, ok := params["command"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid command parameter")
	}

	// Security check - only allow safe commands
	safeCommands := []string{"ls", "pwd", "echo", "date", "whoami", "uname"}
	commandParts := strings.Fields(command)
	if len(commandParts) == 0 {
		return nil, fmt.Errorf("empty command")
	}

	baseCommand := commandParts[0]
	isSafe := false
	for _, safe := range safeCommands {
		if baseCommand == safe {
			isSafe = true
			break
		}
	}

	if !isSafe {
		return nil, fmt.Errorf("command not allowed for security reasons: %s", baseCommand)
	}

	// Execute the command
	cmd := exec.CommandContext(ctx, "/bin/bash", "-c", command)
	cmd.Dir = p.workingDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return map[string]interface{}{
			"command":   command,
			"output":    string(output),
			"error":     err.Error(),
			"success":   false,
			"timestamp": time.Now().Format(time.RFC3339),
		}, nil
	}

	return map[string]interface{}{
		"command":   command,
		"output":    string(output),
		"success":   true,
		"timestamp": time.Now().Format(time.RFC3339),
	}, nil
}

// GetCapabilities returns the agent's capabilities
func (p *ComprehensiveAgentPlugin) GetCapabilities() *agentify.AgentCapabilities {
	return &agentify.AgentCapabilities{
		SupportsStreaming:   true,
		SupportsToolCalls:   true,
		SupportsReasoning:   true,
		MaxContextLength:    16384,
		SupportedParameters: []string{"temperature", "top_p", "max_tokens", "frequency_penalty", "presence_penalty"},
	}
}

// GetSchema returns the agent's schema
func (p *ComprehensiveAgentPlugin) GetSchema() *agentify.AgentSchema {
	return &agentify.AgentSchema{
		Tools: []*agentify.ToolSchema{
			{
				Name:        "file_operations",
				Description: "Perform file system operations (list, read, write)",
				Parameters: map[string]*agentify.ParameterSchema{
					"operation": {
						Type:        "string",
						Description: "The operation to perform (list, read, write)",
						Required:    true,
					},
					"path": {
						Type:        "string",
						Description: "The file or directory path",
						Required:    false,
					},
					"content": {
						Type:        "string",
						Description: "The content to write (for write operation)",
						Required:    false,
					},
				},
				ReturnType: "object",
			},
			{
				Name:        "web_search",
				Description: "Search the web for information",
				Parameters: map[string]*agentify.ParameterSchema{
					"query": {
						Type:        "string",
						Description: "The search query",
						Required:    true,
					},
				},
				ReturnType: "object",
			},
			{
				Name:        "calculator",
				Description: "Perform mathematical calculations",
				Parameters: map[string]*agentify.ParameterSchema{
					"expression": {
						Type:        "string",
						Description: "The mathematical expression to evaluate",
						Required:    true,
					},
				},
				ReturnType: "object",
			},
			{
				Name:        "system_info",
				Description: "Get system information",
				Parameters: map[string]*agentify.ParameterSchema{
					"type": {
						Type:        "string",
						Description: "The type of system info (general, memory, disk)",
						Required:    false,
					},
				},
				ReturnType: "object",
			},
			{
				Name:        "memory_search",
				Description: "Search the agent's memory using semantic search",
				Parameters: map[string]*agentify.ParameterSchema{
					"query": {
						Type:        "string",
						Description: "The search query",
						Required:    true,
					},
				},
				ReturnType: "object",
			},
			{
				Name:        "terminal_command",
				Description: "Execute safe terminal commands",
				Parameters: map[string]*agentify.ParameterSchema{
					"command": {
						Type:        "string",
						Description: "The command to execute",
						Required:    true,
					},
				},
				ReturnType: "object",
			},
		},
		Resources: []*agentify.ResourceSchema{
			{
				Name:        "faq",
				Type:        "text",
				Description: "Frequently asked questions about the agent",
			},
		},
		Prompts: []*agentify.PromptSchema{
			{
				Name:        "system",
				Description: "The system prompt for the agent",
				Variables:   []string{},
			},
			{
				Name:        "greeting",
				Description: "The greeting prompt for the agent",
				Variables:   []string{"name"},
			},
			{
				Name:        "error",
				Description: "The error handling prompt",
				Variables:   []string{"error"},
			},
		},
	}
}

// main function is required for building the plugin
func main() {
	fmt.Println("Comprehensive Agent Plugin - demonstrates all Agent Inferencer capabilities")
}
