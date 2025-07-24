# Agent Inferencer Usage Guide

## Overview

The Agent Inferencer is a system that enables LLMs to operate through plugin binaries as agents. In this architecture, the plugins ARE the agents, providing LLMs with full access to all tools, prompts, and resources configured within the plugin.

Architectural Summary

The Agentify system architecture consists of these key components:

### TypeScript Compiler (AgentCompiler)
- **Input**: Enhanced AgentPluginConfig with improved tool definitions, credential declarations, and model specifications
- **Process**: Generates Go and Python code from templates, bundles tool code, compiles Go into plugin binary
- **Output**: .so/.dll plugin

### Go Agent Plugin (Compiled Binary)
- **Core**: Implements AgentPluginInterface
- **Initialization**: Receives config and resolved credentials, stores securely
- **Memory**: Manages state for context, credentials, cache, plans, preferences
- **Tools**: Defined via config, executed within TEE with robust parameter parsing
- **TEE**: Provides isolation with resource limits, access controls, streaming capabilities
- **Python Service**: Embedded via TEE for Python-specific logic and LLM calls
- **LLM Interaction**: Handles communication with LLMs with streaming support
- **Debug Interface**: Exposes commands for easier debugging

### Agent Inferencer (Plugin Loader)
- Discovers and loads plugin binaries
- Securely resolves credentials and passes to plugin
- Manages plugin lifecycle
- Interacts via AgentPluginInterface

### Inference-Enabled Client/Orchestrator
- External system that uses AgentPluginLoader to interact with AI agent plugins

## Getting Started

### Installation

To use the Agent Inferencer, you need to include the `agentify` package in your Go project:

```go
import "github.com/cloud-equities/KNIRVROOT/inference/agentify"
```

### Creating an Agent Inferencer

To create an Agent Inferencer, you need to specify the directory where the agent plugins are located:

```go
inferencer := NewAgentInferencer("/path/to/plugins")
```

### Activating an Agent

To activate an agent for a session, you need to specify the agent ID, version, and session ID:

```go
ctx := context.Background()
sessionID := "my-session"
err := inferencer.ActivateAgent(ctx, "example", "1.0", sessionID, nil)
if err != nil {
    // Handle error
}
```

### Processing an Inference Request

To process an inference request, you need to create an `InferenceRequest` and pass it to the `ProcessInference` method:

```go
request := &InferenceRequest{
    Input:     "Hello, world!",
    SessionID: sessionID,
}

response, err := inferencer.ProcessInference(ctx, sessionID, request)
if err != nil {
    // Handle error
}

fmt.Println(response.Output)
```

### Deactivating an Agent

When you're done with an agent, you should deactivate it to free up resources:

```go
err := inferencer.DeactivateAgent(ctx, sessionID)
if err != nil {
    // Handle error
}
```

## HTTP API

The Agent Inferencer provides an HTTP API for clients to interact with agents:

### Creating an HTTP API

To create an HTTP API, you need to pass an Agent Inferencer to the `NewAgentHTTPAPI` function:

```go
api := NewAgentHTTPAPI(inferencer)
```

### Registering Handlers

To register the API handlers with an HTTP server, you need to call the `RegisterHandlers` method:

```go
mux := http.NewServeMux()
api.RegisterHandlers(mux)

server := &http.Server{
    Addr:    ":8080",
    Handler: mux,
}

server.ListenAndServe()
```

### API Endpoints

The HTTP API provides the following endpoints:

- `GET /v1/agents`: List available agents
- `POST /v1/agents/activate`: Activate an agent
- `POST /v1/agents/deactivate`: Deactivate an agent
- `POST /v1/inference`: Process an inference request
- `GET /v1/schema`: Get the schema of an agent
- `GET /v1/capabilities`: Get the capabilities of an agent
- `GET /v1/memory`: Get a value from the agent's memory
- `POST /v1/memory`: Set a value in the agent's memory
- `GET /v1/tee`: Get information about the TEE for an agent

## Client Libraries

The Agent Inferencer provides client libraries for Python and JavaScript:

### Python Client

To use the Python client, you need to import the `AgentClient` class:

```python
from agent_client import AgentClient

# Create a client
client = AgentClient("http://localhost:8080", "test-api-key")

# List available agents
agents = client.list_agents()
print(f"Available agents: {agents}")

# Create a session ID
session_id = "my-session"

# Activate an agent
client.activate_agent("example", "1.0", session_id)

try:
    # Process an inference request
    response = client.process_inference(session_id, "Hello, world!")
    print(f"Response: {response.output}")
finally:
    # Deactivate the agent
    client.deactivate_agent(session_id)
```

### JavaScript Client

To use the JavaScript client, you need to import the `AgentClient` class:

```javascript
const { AgentClient } = require('./agent_client');

// Create a client
const client = new AgentClient('http://localhost:8080', 'test-api-key');

// Create a session ID
const sessionId = 'my-session';

async function example() {
    // List available agents
    const agents = await client.listAgents();
    console.log(`Available agents: ${agents}`);
    
    // Activate an agent
    await client.activateAgent('example', '1.0', sessionId);
    
    try {
        // Process an inference request
        const response = await client.processInference(sessionId, 'Hello, world!');
        console.log(`Response: ${response.output}`);
    } finally {
        // Deactivate the agent
        await client.deactivateAgent(sessionId);
    }
}

example().catch(console.error);
```

## Framework Integrations

The Agent Inferencer provides integrations with popular frameworks:

### LangChain Integration

To use the LangChain integration, you need to import the `AgentPluginLLM` class:

```python
from langchain_integration import AgentPluginLLM

# Create an LLM
llm = AgentPluginLLM(
    base_url="http://localhost:8080",
    api_key="test-api-key",
    agent_id="example",
    version="1.0"
)

# Generate text
response = llm("Hello, world!")
print(f"Response: {response}")
```

### LlamaIndex Integration

To use the LlamaIndex integration, you need to import the `AgentPluginLLM` class:

```python
from llamaindex_integration import AgentPluginLLM

# Create an LLM
llm = AgentPluginLLM(
    base_url="http://localhost:8080",
    api_key="test-api-key",
    agent_id="example",
    version="1.0"
)

# Complete a prompt
response = llm.complete("Hello, world!")
print(f"Response: {response.text}")
```

### Hugging Face Integration

To use the Hugging Face integration, you need to import the `AgentPluginPipeline` class:

```python
from huggingface_integration import AgentPluginPipeline

# Create a pipeline
pipeline = AgentPluginPipeline(
    base_url="http://localhost:8080",
    api_key="test-api-key",
    agent_id="example",
    version="1.0"
)

# Generate text
result = pipeline("Hello, world!")
print(f"Response: {result['generated_text']}")
```

## Creating Agent Plugins

To create an agent plugin, you need to implement the `AgentPluginInterface` interface:

```go
// example_agent_plugin.go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/cloud-equities/KNIRVROOT/inference/agentify"
)

// ExampleAgentPlugin is an example implementation of the AgentPluginInterface
type ExampleAgentPlugin struct {
    *BaseAgentPlugin
}

// Plugin is the exported symbol that will be loaded by the AgentPluginLoader
var Plugin = &ExampleAgentPlugin{
    BaseAgentPlugin: NewBaseAgentPlugin(),
}

// Initialize initializes the agent with configuration
func (p *ExampleAgentPlugin) Initialize(config map[string]interface{}) error {
    // Call the base implementation
    if err := p.BaseAgentPlugin.Initialize(config); err != nil {
        return err
    }

    // Register custom tools
    p.RegisterTool("search", p.searchTool)
    p.RegisterTool("calculator", p.calculatorTool)

    return nil
}

// searchTool is an example tool implementation
func (p *ExampleAgentPlugin) searchTool(ctx context.Context, params map[string]interface{}) (interface{}, error) {
    query, ok := params["query"].(string)
    if !ok {
        return nil, fmt.Errorf("missing or invalid query parameter")
    }

    // In a real implementation, we would perform a search
    // For now, we'll just return a mock result
    return map[string]interface{}{
        "results": []map[string]interface{}{
            {
                "title":       "Example Result 1",
                "description": fmt.Sprintf("This is an example result for the query: %s", query),
                "url":         "https://example.com/result1",
            },
            {
                "title":       "Example Result 2",
                "description": fmt.Sprintf("Another example result for: %s", query),
                "url":         "https://example.com/result2",
            },
        },
        "query": query,
        "time":  time.Now().Format(time.RFC3339),
    }, nil
}

// calculatorTool is an example tool implementation
func (p *ExampleAgentPlugin) calculatorTool(ctx context.Context, params map[string]interface{}) (interface{}, error) {
    expression, ok := params["expression"].(string)
    if !ok {
        return nil, fmt.Errorf("missing or invalid expression parameter")
    }

    // In a real implementation, we would evaluate the expression
    // For now, we'll just return a mock result
    return map[string]interface{}{
        "expression": expression,
        "result":     42,
        "time":       time.Now().Format(time.RFC3339),
    }, nil
}

// main function is required for building the plugin
func main() {
    // This function is not used when the file is built as a plugin
    fmt.Println("This is an example agent plugin")
}
```

To build the plugin, you need to use the `go build` command with the `-buildmode=plugin` flag:

```bash
go build -buildmode=plugin -o plugins/agent_example_1.0.so example_agent_plugin.go
```

## Conclusion

The Agent Inferencer provides a powerful way to enable LLMs to operate through plugin binaries as agents. By implementing the `AgentPluginInterface` interface, you can create custom agents with their own tools, prompts, and resources.