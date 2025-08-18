# Agent Inferencer Examples

This directory contains examples demonstrating how to use the Agent Inferencer system with real plugins and integrations.

## Examples

### 1. Real Plugin Demo (`real_plugin_demo/main.go`)

**🚀 NEW!** A comprehensive example that demonstrates loading and using your actual plugin from the `/plugins` folder. This example shows:

- **Plugin Discovery**: Automatically finding plugins in the plugins directory
- **Plugin Loading**: Loading your actual `.so` plugin file
- **Plugin Management**: Activating, testing, and deactivating plugins
- **LLM Integration**: Full integration with inference service
- **Terminal Sessions**: Creating and managing terminal sessions with your plugin
- **Memory Operations**: Storing and retrieving plugin state
- **Error Handling**: Proper error handling and fallback strategies

#### Running the Real Plugin Demo

```bash
cd agentify/examples/real_plugin_demo
go run main.go
```

### 2. Plugin Loading Example (`plugin_loading/main.go`)

Shows how to discover, load, and manage plugins with the Agent Inferencer system. This example demonstrates:

- **LLM Integration**: How to connect an inference service to the Agent Inferencer
- **Plugin Discovery**: Finding available plugins in the system
- **Plugin Naming**: Understanding the plugin naming convention
- **Plugin Activation**: Loading and activating plugins
- **Interactive Setup**: User-guided plugin configuration

#### Running the Plugin Loading Example

```bash
cd agentify/examples/plugin_loading
go run main.go
```

### 3. CGO Plugin Wrapper (`cgo_plugin_wrapper/cgo_wrapper.go`)

Demonstrates how to create a wrapper that bridges CGO-based plugins (like yours) with the Agent Inferencer interface. This example shows:

- **CGO Integration**: How to wrap existing CGO plugins
- **Interface Bridging**: Converting between CGO and Agent Inferencer interfaces
- **Plugin Compatibility**: Making existing plugins work with the system

### 4. Inference Integration (`inference_integration/main.go`)

A foundational example showing LLM integration patterns. This example demonstrates:

- **LLM Integration**: Connecting inference services
- **Tool Calling**: Automatic tool execution
- **Chain of Thought**: CoT reasoning patterns
- **Memory Management**: State persistence

## Integration with Existing Inference Service

To integrate with the actual inference service from the `inference` package:

```go
import "Agentic_Engine/inference"

// Create the inference service
inferenceService := inference.NewInferenceService()
if err := inferenceService.Start(); err != nil {
    log.Fatalf("Failed to start inference service: %v", err)
}

// Create the agent inferencer
agentInferencer := agentify.NewAgentInferencer("./plugins")

// Connect them
agentInferencer.SetInferenceService(inferenceService)
```

## Client Library Examples

For client library usage examples, see:
- `../clients/python/agent_client.py` - Python client with terminal support
- `../clients/javascript/agent_client.js` - JavaScript client with WebSocket support

## Plugin Development

To create custom agent plugins, implement the `AgentPluginInterface`:

```go
type MyCustomAgent struct {
    *agentify.BaseAgentPlugin
}

func (a *MyCustomAgent) ProcessInference(ctx context.Context, request *agentify.InferenceRequest) (*agentify.InferenceResponse, error) {
    // Custom inference logic
    return &agentify.InferenceResponse{
        Output: "Custom response",
    }, nil
}

// Compile as plugin
// go build -buildmode=plugin -o agent_mycustom_1.0.so mycustom_agent.go
```

## Testing

Run the test suite to verify everything is working:

```bash
cd agentify
go test -v
```

## Architecture Overview

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Client Apps   │    │  Agent Inferencer │    │ Inference Service│
│                 │    │                  │    │                 │
│ - Python Client │◄──►│ - Agent Management│◄──►│ - LLM Providers │
│ - JS Client     │    │ - Terminal Mgmt   │    │ - Delegator     │
│ - HTTP API      │    │ - Memory Mgmt     │    │ - MOA Support   │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                │
                                ▼
                       ┌─────────────────┐
                       │  Agent Plugins  │
                       │                 │
                       │ - Tools         │
                       │ - Resources     │
                       │ - TEE Support   │
                       └─────────────────┘
```

## Features Demonstrated

- ✅ **LLM Integration**: Full integration with inference service
- ✅ **Tool Calling**: Automatic tool execution from LLM output
- ✅ **Terminal Management**: PTY-based terminal sessions
- ✅ **Memory Management**: Vector and in-memory storage
- ✅ **Client Libraries**: Python and JavaScript clients
- ✅ **WebSocket Support**: Real-time terminal communication
- ✅ **Plugin Architecture**: Extensible agent system
- ✅ **TEE Support**: Trusted execution environments
- ✅ **Comprehensive Testing**: Unit and integration tests
