

---

**Source**: KNIRVROOT/inference/agentify/agentic_UX_Enhancement_Imp.md

# Agentify UX Enhancement Implementation Plan

## Executive Summary

This document outlines a comprehensive implementation plan for enhancing the developer experience (UX) in the Agentify framework. The plan focuses on three key areas:

1. **Tool Definition Improvements** - Making tool creation more intuitive and IDE-friendly
2. **LLM Integration Architecture** - Streamlining how agents interact with language models
3. **Debugging & Security Enhancements** - Providing better visibility and secure credential management

The implementation will transform how developers create, debug, and deploy AI agents within the Agentify ecosystem, significantly reducing friction points and improving overall developer productivity.

## 1. Tool Definition Enhancement

### Current Limitations
In the current implementation, `ToolConfig.implementation` is defined as a string of Go code within the AgentPluginConfig. While flexible, this approach has significant drawbacks:
- Lack of IDE support within JSON/TypeScript strings
- Difficult management of complex Go code snippets
- No syntax highlighting or error checking until compile time

### Implementation Plan

#### Option A: File-Based Tools
Allow ToolConfig to reference separate `.go` files for implementation:

```typescript
export interface ToolConfig {
  name: string;
  description: string;
  parameters: ParameterConfig[];
  returnType: string;
  // NEW: Specify source
  sourceType: 'inlineBody' | 'filePath';
  sourceValue: string; // Go function body or path to .go file
}
```

The AgentCompiler will need to be updated to:
- Check `sourceType` to determine how to handle the implementation
- For `filePath`, copy the referenced file into the build directory
- For `inlineBody`, maintain current behavior of wrapping in Go function structure

#### Option B: Structured Tool Logic
For tools following common patterns, enhance the template system:
- Keep the current `tool.go.template` with `{{toolImplementation}}` placeholder
- Add more robust parameter handling and boilerplate generation

### Parameter Handling Improvements

#### Current Limitations
The `{{parameterParsing}}` in `tool.go.template` relies on manual parsing from `map[string]interface{}`, which is error-prone for complex types.

#### Implementation Plan
Generate Go structs for tool parameters based on ParameterConfig:

```go
// {{toolName}}Params defines the structure for {{toolName}} parameters
type {{toolName}}Params struct {
  {{range .Parameters}}
  {{.Name | Title}} {{.GoType}} `json:"{{.Name}}"`{{end}}
}

// {{toolName}} implements the {{toolDescription}}
func {{toolName}}(ctx context.Context, genericParams map[string]interface{}) (interface{}, error) {
  // Parse parameters into strongly-typed struct
  var params {{toolName}}Params
  jsonData, _ := json.Marshal(genericParams)
  if err := json.Unmarshal(jsonData, &params); err != nil {
    return nil, fmt.Errorf("error parsing parameters for {{toolName}}: %v", err)
  }
  
  // Tool implementation
  {{toolImplementation}}
}
```

The AgentCompiler will generate the `{{toolName}}Params` struct and potentially a wrapper function for type conversion to maintain compatibility with the `ToolFunc` signature in `AgentPlugin`.

## 2. Debugging and Interactivity

### Current Limitations
Debugging compiled Go plugins running inside a TEE is challenging, with limited visibility into runtime behavior.

### Implementation Plan

#### Interactive Debug Interface
Extend the AgentPluginInterface to support debugging capabilities:

```go
type AgentPluginInterface interface {
    // ... existing methods ...
    ExecuteDebugCommand(command string, args []string) (string, error)
    // Optional: For full terminal interaction
    // StartInteractiveSession() (io.ReadWriteCloser, error)
}
```

Implementation requirements:
1. Update `main.go.template` to implement debug commands for:
   - Memory inspection
   - Tool listing and testing
   - TEE status checking
   - Log viewing

2. Enhance TEE implementation to facilitate debug I/O:
   - Expose channels or file descriptors for debug interface
   - Ensure proper isolation of debug commands

3. Update AgentPluginLoader to expose debug interface to developers:
   - Add methods to send debug commands
   - Potentially add a CLI or UI for interactive debugging

## 3. LLM Integration Architecture

### Current Implementation
The primary LLM interaction happens within the embedded Python service using google-adk. The Go plugin's RunAgent calls the Python service, and the ProcessInference method executes a Python script via the TEE.

### Implementation Plan

#### Streaming Support
Enhance the TEE interface to support streaming for LLM interactions:

```go
type TEE interface {
    Start() error
    Stop() error
    Execute(command string, args []string) (stdout string, stderr string, exitCode int, err error)
    ExecuteStream(command string, args []string, stdin io.Reader) (stdout io.ReadCloser, stderr io.ReadCloser, wait func() (int, error), err error)
    CopyFileIn(localPath, teePath string) error
    CopyFileOut(teePath, localPath string) error
}
```

Implementation of ExecuteStream:
```go
func (t *ProcessTEE) ExecuteStream(command string, args []string, stdin io.Reader) (stdout io.ReadCloser, stderr io.ReadCloser, wait func() (int, error), err error) {
    cmd := exec.CommandContext(context.Background(), command, args...)
    cmd.Dir = t.workingDir
    // ... env, limits, access controls ...

    cmd.Stdin = stdin
    stdoutPipe, err := cmd.StdoutPipe()
    if err != nil { return nil, nil, nil, fmt.Errorf("failed to get stdout pipe: %v", err) }
    stderrPipe, err := cmd.StderrPipe()
    if err != nil { return nil, nil, nil, fmt.Errorf("failed to get stderr pipe: %v", err) }

    if err := cmd.Start(); err != nil { return nil, nil, nil, fmt.Errorf("failed to start command: %v", err) }

    waitFunc := func() (int, error) {
        err := cmd.Wait()
        if err != nil {
            if exitErr, ok := err.(*exec.ExitError); ok {
                return exitErr.ExitCode(), err
            }
            return -1, err
        }
        return 0, nil
    }
    return stdoutPipe, stderrPipe, waitFunc, nil
}
```

#### Native Go LLM SDKs
For `agentType: 'llm'`, allow direct use of Go LLM SDKs:

1. Update AgentPluginConfig to specify model provider:
   ```typescript
   export interface AgentPluginConfig {
     // ...
     modelProvider?: 'google_adk' | 'openai_go' | 'anthropic_go';
     // ...
   }
   ```

2. Update `go.mod.template` to conditionally include relevant SDKs
3. Implement provider-specific LLM clients in Go

#### Tool Schema for LLMs
Enhance the `getToolsForLLM()` function to generate rich JSON schemas:

1. Expand ParameterSchema to support complex types:
   - Objects with nested properties
   - Arrays with specific item types
   - Enums with allowed values
   - Patterns for string validation

2. Ensure schema generation follows OpenAI function calling format or equivalent standards for other LLMs

## 4. API Key and Model Management

### Current Implementation
The MemoryManager includes StoreCredential/GetCredential for secure storage, and the Python service template gets the model name from its config.

### Implementation Plan

#### Credential Declaration and Resolution
Decouple credential declaration from values in AgentPluginConfig:

```typescript
export interface RequiredCredential {
  name: string;  // e.g., "OPENAI_API_KEY", "GOOGLE_API_KEY"
  description?: string;
  optional?: boolean;
}

export interface AgentPluginConfig {
  // ...
  requiredCredentials?: RequiredCredential[];
  llmModel?: string;  // e.g., "gemini-1.5-pro", "gpt-4o"
  // ...
}
```

#### Secure Credential Handling
1. Update AgentPluginLoader to resolve credentials at runtime:
   - Source from environment variables, secrets manager, or user prompts
   - Pass to plugin's Initialize method

2. Update Initialize method signature:
   ```go
   func (p *AgentPlugin) Initialize(config map[string]interface{}, resolvedCredentials map[string]string) error {
       // Store credentials securely in memory manager
       for keyName, keyValue := range resolvedCredentials {
           p.memory.Set(fmt.Sprintf("credential_%s", keyName), []byte(keyValue))
       }
       // ...
   }
   ```

3. Pass credentials to Python service:
   ```go
   // Prepare environment variables for Python service
   pythonEnvVars := make(map[string]string)
   if apiKey, ok := resolvedCredentials["GOOGLE_API_KEY"]; ok {
       pythonEnvVars["AGENT_API_KEY"] = apiKey
   }
   // Update TEE.Execute to accept environment variables
   ```

4. Update Python service to access credentials:
   ```python
   # Attempt to get API key from environment variable set by the Go host
   AGENT_API_KEY = os.environ.get("AGENT_API_KEY")
   
   # Use in LLM configuration
   agent = genai.Agent(
       model=config.get('model', 'gemini-2.0-flash'),
       instruction=config.get('instruction', ''),
       description=config.get('description', ''),
       tools=tools,
       model_client_options={"api_key": AGENT_API_KEY}  # If ADK supports direct key pass
   )
   ```

## 5. Architectural Summary

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

### Agent Plugin Loader
- Discovers and loads plugin binaries
- Securely resolves credentials and passes to plugin
- Manages plugin lifecycle
- Interacts via AgentPluginInterface

### Inference-Enabled Client/Orchestrator
- External system that uses AgentPluginLoader to interact with AI agent plugins

## 6. Implementation Roadmap

1. **Phase 1: Tool Definition Enhancement**
   - Implement file-based tool references
   - Add struct-based parameter handling
   - Update templates and compiler

2. **Phase 2: Debugging Interface**
   - Extend AgentPluginInterface with debug commands
   - Implement debug command handlers
   - Create debug UI/CLI for developers

3. **Phase 3: LLM Integration**
   - Add streaming support to TEE
   - Implement native Go LLM SDK integration
   - Enhance tool schema generation

4. **Phase 4: Credential Management**
   - Update AgentPluginConfig for credential declaration
   - Implement secure credential resolution and storage
   - Add credential passing to Python services

5. **Phase 5: Testing & Documentation**
   - Create comprehensive examples
   - Write developer documentation
   - Build automated tests for all components

This implementation plan provides a clear path to significantly enhance the developer experience while maintaining the security and flexibility of the Agentify framework.

---

<div class="footer-links">


© 2025 KNIRV Network
</div>
