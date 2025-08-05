

---

**Source**: KNIRVROOT/docs/pending_implementation_plans/Plugin_Developer_Plan.md

# KNIRVCHAIN Plugin Developer Plan

## Overview

This document outlines the development plan for developers who want to create and register capabilities on the KNIRVCHAIN network. It focuses specifically on the capability development workflow, registration process, and the tools needed to make capabilities accessible to other nodes in the network. The plan incorporates both direct plugin implementation and MCP server integration through extrapolation and interpolation routes.

## MCP Integration Routes

KNIRVCHAIN supports two distinct routes for integrating MCP (Model Context Protocol) capabilities:

### 1. MCP Extrapolation Route

The Extrapolation route follows a server-first approach where capabilities are derived from MCP servers.

#### Server Instance Schema
```yaml
mcpServer:
  id: string                    # Unique identifier
  deploymentType: string        # "github" or "live"
  source: string               # GitHub repo URL or live server URL
  version: string              # Server version
  capabilities: string[]       # List of capability IDs
  status: string              # "active", "inactive", "maintenance"
  deploymentConfig:
    github:
      branch: string
      buildCommand: string
      testCommand: string
    live:
      healthCheckEndpoint: string
      authConfig: object
```

#### Implementation Steps
1. Create MCP server registration system
2. Implement automatic capability discovery
3. Add server health monitoring
4. Develop capability registration pipeline

### 2. MCP Interpolation Route

The Interpolation route follows a capability-first approach where MCP servers are components of capabilities.

## Capability Development Workflow

### 1. Capability Structure and Requirements

Capabilties in KNIRVCHAIN are implemented as shared plugin libraries (.so files) that expose specific interfaces required by the network. Each plugin must be accompanied by an operational schema file (opSchema.yaml, also referred to as manifest.yaml) that describes its tools, dependencies, configuration, and MCP server integration.

#### Plugin File Format

- Plugins must be designed as a single sequential process.
- Plugins must be compiled as shared libraries (.so files on Linux/macOS, .dll files on Windows)
- Plugins must implement the required interfaces based on their capability type
- Plugins must be thread-safe and handle concurrent invocations
- Plugins should properly handle initialization and cleanup

#### Required Interfaces

Depending on the capability type, plugins must implement specific interfaces:

```go
// For RESOURCE type capabilities
type ResourceProvider interface {
    // Initialize is called when the plugin is loaded
    Initialize(config map[string]interface{}) error
    
    // Provide returns the resource data for the given parameters
    Provide(params map[string]interface{}) ([]byte, error)
    
    // GetMetadata returns metadata about the resource
    GetMetadata() map[string]interface{}
    
    // Cleanup is called when the plugin is unloaded
    Cleanup() error
}

// For TOOL type capabilities
type ToolProvider interface {
    // Initialize is called when the plugin is loaded
    Initialize(config map[string]interface{}) error
    
    // Execute runs the tool with the given input and returns the result
    Execute(input []byte) ([]byte, error)
    
    // GetMetadata returns metadata about the tool
    GetMetadata() map[string]interface{}
    
    // Cleanup is called when the plugin is unloaded
    Cleanup() error
}
```

#### Operational Schema Format (opSchema.yaml / manifest.yaml)

The operational schema file (opSchema.yaml, also referred to as manifest.yaml) defines both the plugin's capabilities and its integration with MCP servers. This unified schema supports both direct plugin execution and MCP server orchestration.

*   **Purpose:**
    *   Provides structured metadata about the plugin beyond what's in the on-chain ResourceDescriptor
    *   Defines the sequence of operations or "events" the plugin performs
    *   Specifies points where the agent Client (Inference Engine) can or should inject context, use other MCP Tools/Resources, or prompt the user
    *   Defines MCP server integration through both extrapolation and interpolation routes
    *   Helps clients understand how to interact with more complex, multi-step plugins

*   **Unified Schema Structure:**

```yaml
# Core capability metadata
capability:
  id: "com.example.skyrimmodvalidator" # Unique identifier, could match on-chain ID
  name: "Skyrim Mod Validator"
  version: "1.2.0"
  description: "Validates Skyrim mod files for compatibility and errors using TES5Edit."
  type: "RESOURCE"  # Capability type
  owner: "0x123abc..." # Owner address
  gasFeeNRN: 100  # Gas fee in NRN tokens
  author: "PluginDevX"
  license: "MIT"

# Entry point for the client to call in the plugin binary/module (for direct plugin execution)
entryFunction: "ValidateAsset" # For Go .so, this is the exported function name

# Input/Output schemas (can reference external JSON schema files or be inline)
inputSchema: "./schemas/input.json" # Path relative to manifest
outputSchema: "./schemas/output.json"

# Sequential process definition
process:
  # Process steps
  steps:
    - id: "step1"
      name: "Initialize TES5Edit"
      description: "Sets up the TES5Edit environment. Requires path to TES5Edit."
      type: "plugin_function" # Can be plugin_function, mcp_server, tool, etc.
      # For plugin function execution
      pluginAction:
        function: "initializeTES5Edit"
        params: ["tes5EditPath"]
      # For MCP server integration (interpolation)
      mcpServer:
        deploymentType: "github"
        source: "https://github.com/example/tes5edit-server"
        version: "1.0.0"
        config:
          installPath: "{{tes5EditPath}}"
      # Client injection points
      clientInjection:
        point: "before" # or "after" or "during" (if plugin calls back)
        type: "config" # "config", "resource", "tool", "userPrompt"
        prompt: "Please provide the full path to your TES5Edit.exe:" # If type is userPrompt
      input: 
        tes5EditPath: "string"
      output:
        tes5editSessionId: "string"
      next: "step2" # Next step ID
      errorHandling:
        retry: 3
        fallback: "error"

    - id: "step2"
      name: "Load Mod File"
      description: "Loads the specified mod file into TES5Edit."
      type: "plugin_function"
      # Input mapping from previous steps and plugin input
      input:
        assetUrl: "{{pluginInput.AssetURL}}"
        tes5editSessionId: "{{step1.output.tes5editSessionId}}"
      # Client injection for MCP tool usage
      clientInjection:
        point: "before"
        type: "tool"
        mcpCapabilityId: "tool-file-downloader"
        mcpInput: { "url": "{{pluginInput.AssetURL}}" }
        mcpOutputMapping:
          downloadedFilePath: "localAssetPath"
      # Plugin function execution
      pluginAction:
        function: "loadMod"
        params: ["localAssetPath", "tes5editSessionId"]
      output:
        loadStatus: "boolean"
      next: "step3"

    - id: "step3"
      name: "Run Validation Checks"
      description: "Performs validation checks using TES5Edit."
      type: "mcp_server" # This step uses an MCP server directly
      input:
        tes5editSessionId: "{{step1.output.tes5editSessionId}}"
      # MCP server configuration for this step
      mcpServer:
        deploymentType: "live"
        source: "https://api.tes5edit-validator.com"
        version: "2.1.0"
        config:
          validationLevel: "comprehensive"
      # Client injection for LLM reasoning
      clientInjection:
        point: "before"
        type: "inference"
        promptTemplate: |
          Given the mod file being validated (details not directly available here, but assume context),
          and the user's goal of ensuring compatibility, what specific TES5Edit checks
          should be prioritized? List up to 3.
        llmConfig:
          model: "user_preferred_reasoning_model"
        outputMapping:
          llmSuggestions: "prioritizedChecks"
      output:
        validationReport: "object"
      next: "end"

  # Process flow control
  flow:
    start: "step1"
    end: "step3"
    error: "handleError"

  # Resource requirements
  resources:
    memory: "512Mi"
    cpu: "0.5"
    storage: "1Gi"

  # Deployment configuration
  deployment:
    type: "github"
    config:
      repository: "https://github.com/example/skyrim-validator"
      branch: "main"
      buildCommand: "make build"

# Required MCP capabilities and servers
requiredClientCapabilities:
  - capabilityId: "tool-file-downloader"
    description: "Needed to download assets specified by URL."
  - capabilityId: "mcp-logging-service"
    description: "For the plugin to log its progress via the client's MCP context."

# MCP servers that this plugin can serve or reference
mcpServers:
  - id: "tes5edit-validator"
    deploymentType: "github"
    source: "https://github.com/example/tes5edit-validator"
    version: "1.0.0"
    capabilities: ["validate-mod", "analyze-compatibility"]
    status: "active"
```

This unified schema supports both direct plugin execution and MCP server integration. It allows plugins to:

1. **Serve as MCP Servers**: A plugin can expose its functionality as an MCP server, making it available to other capabilities.
2. **Reference MCP Servers**: A plugin can use existing MCP servers as part of its process flow.
3. **Hybrid Approach**: A plugin can both serve as an MCP server and reference other MCP servers.

The schema is designed to be flexible, allowing developers to choose the approach that best fits their needs while maintaining compatibility with the KNIRVCHAIN network.

### 2. Plugin Development Process

The development process varies slightly depending on whether you're following the direct plugin approach, the extrapolation route, or the interpolation route.

#### Direct Plugin Development

1. **Set up development environment**
   - Install Go 1.18 or later
   - Set up the KNIRVCHAIN SDK development environment
   - Configure build tools for plugin compilation

2. **Create plugin implementation**
   - Implement the required interfaces
   - Add necessary dependencies
   - Implement business logic

3. **Create opSchema.yaml file**
   - Define plugin metadata
   - Specify required interfaces
   - Configure resource limits
   - Define process steps and flow

4. **Build plugin**
   - Compile as a shared library
   - Ensure proper exports
   - Validate against interface requirements

5. **Test plugin locally**
   - Create test harness
   - Verify functionality
   - Test error handling

6. **Register capability**
   - Use the SDK tool to register the capability
   - Provide access to plugin and opSchema files
   - Verify registration on the blockchain

#### MCP Extrapolation Development

1. **Set up MCP server environment**
   - Choose deployment type (GitHub or live)
   - Configure server environment
   - Set up necessary dependencies

2. **Implement MCP server**
   - Create server implementation
   - Implement capability interfaces
   - Configure server endpoints

3. **Create server instance schema**
   - Define server metadata
   - List provided capabilities
   - Configure deployment settings

4. **Test MCP server**
   - Verify server functionality
   - Test capability interfaces
   - Validate deployment process

5. **Register MCP server**
   - Use the SDK tool to register the server
   - System automatically discovers capabilities
   - Capabilities are registered on the blockchain

#### MCP Interpolation Development

1. **Set up development environment**
   - Configure for both plugin and MCP server development
   - Install necessary dependencies
   - Set up build tools

2. **Create opSchema.yaml file**
   - Define capability metadata
   - Specify process steps and flow
   - Configure MCP server integration
   - Define resource requirements

3. **Implement required components**
   - Create plugin implementation (if needed)
   - Implement MCP server components
   - Configure integration points

4. **Test end-to-end flow**
   - Verify process execution
   - Test MCP server integration
   - Validate error handling

5. **Register operational procedure**
   - Use the SDK tool to register the procedure
   - System deploys MCP servers as needed
   - Capability is registered on the blockchain

## Capability Registration

KNIRVCHAIN supports multiple registration commands depending on the integration route you're following.

### MCP Capability Registration Commands

#### Direct Plugin Registration

The `mcp register-capability` command is used to register a new capability with a direct plugin implementation:

```
agent-sdk-tool mcp register-capability --node <node_url> --wallet <wallet_file_path> --from <from_address> --type <RESOURCE|TOOL|...> --descriptor <json_string_or_file_path> --plugin <plugin_so_file_path> --opschema <opschema_file_path> --fee <amount> [--password <pwd>] [--location-hint <uri>] [--relative-path-base <path>] [--start-file-server] [--file-server-port <port>]
```

#### MCP Server Registration (Extrapolation)

The `mcp register-server` command is used to register an MCP server for the extrapolation route:

```
agent-sdk-tool mcp register-server --node <node_url> --wallet <wallet_file_path> --from <from_address> --server-schema <server_schema_file_path> --fee <amount> [--password <pwd>] [--auto-register-capabilities]
```

#### Operational Procedure Registration (Interpolation)

The `mcp register-procedure` command is used to register an operational procedure for the interpolation route:

```
agent-sdk-tool mcp register-procedure --node <node_url> --wallet <wallet_file_path> --from <from_address> --opschema <opschema_file_path> --plugin <plugin_so_file_path> --fee <amount> [--password <pwd>] [--deploy-servers] [--location-hint <uri>]
```

#### Command Implementations

```go
// Direct Plugin Registration Command
var registerCapabilityCmd = &cobra.Command{
    Use:   "register-capability",
    Short: "Register a new capability on the KNIRVCHAIN blockchain",
    Long: `Register a new capability (resource, tool, etc.) on the KNIRVCHAIN blockchain.

For plugin resources, this command will validate the plugin .so file and opSchema file,
create references to these files in the descriptor, and add location hints for dev access.

Example:
  agent-sdk-tool mcp register-capability --node http://localhost:8080 --wallet ./my-wallet.json \
    --from 0x123abc... --type RESOURCE --descriptor ./descriptor.json \
    --plugin ./plugin.so --opschema ./opschema.yaml --fee 100`,
    RunE: func(cmd *cobra.Command, args []string) error {
        // Implementation of the command
        return registerCapability(cmd, args)
    },
}

// MCP Server Registration Command (Extrapolation)
var registerServerCmd = &cobra.Command{
    Use:   "register-server",
    Short: "Register an MCP server on the KNIRVCHAIN blockchain",
    Long: `Register an MCP server on the KNIRVCHAIN blockchain using the extrapolation route.

This command will register the server and optionally auto-register all capabilities
provided by the server.

Example:
  agent-sdk-tool mcp register-server --node http://localhost:8080 --wallet ./my-wallet.json \
    --from 0x123abc... --server-schema ./server-schema.yaml --fee 100 --auto-register-capabilities`,
    RunE: func(cmd *cobra.Command, args []string) error {
        // Implementation of the command
        return registerServer(cmd, args)
    },
}

// Operational Procedure Registration Command (Interpolation)
var registerProcedureCmd = &cobra.Command{
    Use:   "register-procedure",
    Short: "Register an operational procedure on the KNIRVCHAIN blockchain",
    Long: `Register an operational procedure on the KNIRVCHAIN blockchain using the interpolation route.

This command will register the procedure, deploy any required MCP servers, and register
the capability on the blockchain.

Example:
  agent-sdk-tool mcp register-procedure --node http://localhost:8080 --wallet ./my-wallet.json \
    --from 0x123abc... --opschema ./opschema.yaml --plugin ./plugin.so --fee 100 --deploy-servers`,
    RunE: func(cmd *cobra.Command, args []string) error {
        // Implementation of the command
        return registerProcedure(cmd, args)
    },
}
```

#### Required Parameters

```go
func init() {
    // Common flags for all commands
    commonFlags := func(cmd *cobra.Command) {
        cmd.Flags().String("node", "", "URL of the KNIRVCHAIN node")
        cmd.Flags().String("wallet", "", "Path to wallet file")
        cmd.Flags().String("from", "", "Address of the sender")
        cmd.Flags().Uint64("fee", 0, "Transaction fee")
        cmd.Flags().String("password", "", "Wallet password (will prompt if not provided)")
        
        // Mark required common flags
        cmd.MarkFlagRequired("node")
        cmd.MarkFlagRequired("wallet")
        cmd.MarkFlagRequired("from")
        cmd.MarkFlagRequired("fee")
    }
    
    // Direct Plugin Registration Command Flags
    commonFlags(registerCapabilityCmd)
    registerCapabilityCmd.Flags().String("type", "", "Capability type (RESOURCE, TOOL, etc.)")
    registerCapabilityCmd.Flags().String("descriptor", "", "JSON string or path to descriptor file")
    registerCapabilityCmd.Flags().String("plugin", "", "Path to plugin .so file (required for RESOURCE type)")
    registerCapabilityCmd.Flags().String("opschema", "", "Path to opSchema file (required for RESOURCE type)")
    registerCapabilityCmd.Flags().StringSlice("location-hint", nil, "Additional location hints for file access")
    registerCapabilityCmd.Flags().String("relative-path-base", "", "Base directory for relative paths")
    registerCapabilityCmd.Flags().Bool("start-file-server", false, "Start a local file server for hosting plugin files")
    registerCapabilityCmd.Flags().Int("file-server-port", 8090, "Port for the local file server")
    
    // Mark required flags for direct plugin registration
    registerCapabilityCmd.MarkFlagRequired("type")
    registerCapabilityCmd.MarkFlagRequired("descriptor")
    
    // MCP Server Registration Command Flags (Extrapolation)
    commonFlags(registerServerCmd)
    registerServerCmd.Flags().String("server-schema", "", "Path to server schema file")
    registerServerCmd.Flags().Bool("auto-register-capabilities", false, "Automatically register all capabilities provided by the server")
    registerServerCmd.Flags().Bool("deploy-server", false, "Deploy the server if it's not already running")
    registerServerCmd.Flags().String("deployment-config", "", "Path to deployment configuration file")
    
    // Mark required flags for server registration
    registerServerCmd.MarkFlagRequired("server-schema")
    
    // Operational Procedure Registration Command Flags (Interpolation)
    commonFlags(registerProcedureCmd)
    registerProcedureCmd.Flags().String("opschema", "", "Path to opSchema file")
    registerProcedureCmd.Flags().String("plugin", "", "Path to plugin .so file (if procedure includes plugin functions)")
    registerProcedureCmd.Flags().Bool("deploy-servers", false, "Deploy any MCP servers required by the procedure")
    registerProcedureCmd.Flags().StringSlice("location-hint", nil, "Additional location hints for file access")
    registerProcedureCmd.Flags().Bool("validate-servers", true, "Validate that all referenced MCP servers are accessible")
    
    // Mark required flags for procedure registration
    registerProcedureCmd.MarkFlagRequired("opschema")
}
```

#### Schema Formats

##### Capability Descriptor

The capability descriptor is a JSON document that describes the capability being registered through the direct plugin route:

```json
{
  "name": "Example Resource",
  "description": "An example resource capability",
  "version": "1.0.0",
  "type": "RESOURCE",
  "interfaces": ["ResourceProvider"],
  "parameters": {
    "required": ["param1"],
    "optional": ["param2", "param3"]
  },
  "metadata": {
    "author": "KNIRVCHAIN Developer",
    "license": "MIT",
    "tags": ["example", "resource"]
  }
}
```

##### Server Schema (Extrapolation)

The server schema is a YAML document that describes an MCP server for the extrapolation route:

```yaml
mcpServer:
  id: "example-mcp-server"
  name: "Example MCP Server"
  description: "An example MCP server providing various capabilities"
  deploymentType: "github"
  source: "https://github.com/example/mcp-server"
  version: "1.0.0"
  capabilities:
    - "resource-provider"
    - "tool-executor"
    - "prompt-generator"
  status: "active"
  deploymentConfig:
    github:
      branch: "main"
      buildCommand: "make build"
      testCommand: "make test"
    resources:
      memory: "512Mi"
      cpu: "0.5"
      storage: "1Gi"
  metadata:
    author: "KNIRVCHAIN Developer"
    license: "MIT"
    tags: ["example", "server"]
```

##### Operational Schema (Interpolation)

The operational schema (opSchema.yaml) is a YAML document that describes a capability through the interpolation route, as shown in the earlier section. It combines aspects of both the capability descriptor and the server schema, providing a unified approach to capability registration.

### End-to-End Registration Flows

The registration process varies depending on the integration route being used.

#### Direct Plugin Registration Flow

1. **Validate inputs**
   - Check that all required parameters are provided
   - Validate plugin and opSchema files
   - Ensure the wallet exists and can be decrypted

2. **Set up file reference strategy**
   - Determine how plugin files will be made accessible
   - Configure the appropriate strategy
   - Generate location hints

3. **Prepare capability registration**
   - Create the capability registration request
   - Include file references and location hints
   - Calculate transaction fee

4. **Sign transaction**
   - Load and decrypt the wallet
   - Sign the transaction with the private key
   - Verify the signature

5. **Submit transaction**
   - Send the signed transaction to the blockchain
   - Wait for confirmation
   - Display the result

#### MCP Server Registration Flow (Extrapolation)

1. **Validate server schema**
   - Check that the server schema is valid
   - Verify server accessibility if it's a live server
   - Ensure the wallet exists and can be decrypted

2. **Deploy server if needed**
   - Clone the repository if it's a GitHub deployment
   - Build and test the server
   - Deploy the server

3. **Discover capabilities**
   - Query the server for its capabilities
   - Validate capability interfaces
   - Prepare capability registrations

4. **Register server**
   - Create the server registration request
   - Sign and submit the transaction
   - Wait for confirmation

5. **Register capabilities**
   - If auto-register is enabled, register each capability
   - Link capabilities to the server
   - Display results

#### Operational Procedure Registration Flow (Interpolation)

1. **Validate opSchema**
   - Check that the opSchema is valid
   - Validate plugin file if included
   - Ensure the wallet exists and can be decrypted

2. **Validate MCP servers**
   - Check that all referenced MCP servers are accessible
   - Deploy servers if needed and if deploy-servers is enabled
   - Verify server capabilities

3. **Prepare procedure registration**
   - Create the procedure registration request
   - Include file references and server references
   - Calculate transaction fee

4. **Sign and submit transaction**
   - Load and decrypt the wallet
   - Sign the transaction with the private key
   - Submit the transaction and wait for confirmation

5. **Register capability**
   - Register the capability described in the opSchema
   - Link the capability to the procedure
   - Display results

```go
func registerCapability(cmd *cobra.Command, args []string) error {
    // 1. Parse and validate flags
    nodeURL, _ := cmd.Flags().GetString("node")
    walletPath, _ := cmd.Flags().GetString("wallet")
    fromAddress, _ := cmd.Flags().GetString("from")
    capabilityType, _ := cmd.Flags().GetString("type")
    descriptorInput, _ := cmd.Flags().GetString("descriptor")
    pluginPath, _ := cmd.Flags().GetString("plugin")
    manifestPath, _ := cmd.Flags().GetString("manifest")
    fee, _ := cmd.Flags().GetUint64("fee")
    password, _ := cmd.Flags().GetString("password")
    
    // 2. Validate inputs based on capability type
    if capabilityType == "RESOURCE" {
        if pluginPath == "" || manifestPath == "" {
            return fmt.Errorf("plugin and manifest files are required for RESOURCE capability type")
        }
    }
    
    // 3. Load and parse descriptor
    descriptor, err := loadDescriptor(descriptorInput)
    if err != nil {
        return fmt.Errorf("failed to load descriptor: %w", err)
    }
    
    // 4. Set up file reference strategy
    strategy, err := setupFileReferenceStrategy(cmd)
    if err != nil {
        return fmt.Errorf("failed to set up file reference strategy: %w", err)
    }
    
    // 5. Load wallet and get private key
    if password == "" {
        password, err = promptForPassword("Enter wallet password: ")
        if err != nil {
            return fmt.Errorf("failed to get password: %w", err)
        }
    }
    
    privateKey, err := wallet.LoadAndDecryptWallet(walletPath, password)
    if err != nil {
        return fmt.Errorf("failed to load wallet: %w", err)
    }
    
    // 6. Prepare capability registration request
    request := &MCPPrepareCapabilityRegistrationRequest{
        FromAddress:    fromAddress,
        CapabilityType: capabilityType,
        Descriptor:     descriptor,
        Fee:            fee,
    }
    
    // 7. Call API to prepare registration
    client := api.NewAPIClient(nodeURL, 30*time.Second, 3, logger)
    response, err := client.PrepareCapabilityRegistration(context.Background(), *request, pluginPath, manifestPath)
    if err != nil {
        return fmt.Errorf("failed to prepare capability registration: %w", err)
    }
    
    // 8. Extract transaction details for signing
    unsignedTxDetails := response.TransactionDetailsForSigning
    mcpPayloadBytes, err := json.Marshal(unsignedTxDetails.Data)
    if err != nil {
        return fmt.Errorf("failed to marshal transaction data: %w", err)
    }
    
    // 9. Sign transaction
    signature, txHash, err := signer.SignTransactionData(privateKey, unsignedTxDetails, mcpPayloadBytes)
    if err != nil {
        return fmt.Errorf("failed to sign transaction: %w", err)
    }
    
    // 10. Assemble signed transaction
    publicKeyHex := wallet.GetPublicKeyHex(&privateKey.PublicKey)
    signedTx, err := signer.AssembleSignedTransaction(unsignedTxDetails, publicKeyHex, signature, txHash)
    if err != nil {
        return fmt.Errorf("failed to assemble signed transaction: %w", err)
    }
    
    // 11. Submit transaction
    submitResponse, err := client.SubmitTransaction(context.Background(), *signedTx)
    if err != nil {
        return fmt.Errorf("failed to submit transaction: %w", err)
    }
    
    // 12. Display result
    fmt.Printf("Capability registered successfully!\n")
    fmt.Printf("Transaction Hash: %s\n", submitResponse.TransactionHash)
    fmt.Printf("Capability ID: %s\n", response.CapabilityID)
    fmt.Printf("Capability URI: %s\n", generateCapabilityURI(response.CapabilityID, capabilityType))
    
    return nil
}
```

## File Reference Strategy

The CLI tool supports multiple strategies for making plugin files accessible to devs. This is a critical component of the capability registration process, as it ensures that other nodes in the network can access the plugin and manifest files referenced in the capability descriptor.

### 1. **Local File Server Strategy**

- **Implementation**:
  ```go
  type LocalFileServerStrategy struct {
      BaseDir    string
      ServerPort int
      LocalIP    string
      server     *http.Server
  }
  
  func NewLocalFileServerStrategy(baseDir string, port int) (*LocalFileServerStrategy, error) {
      // Get local IP address
      localIP, err := getLocalIP()
      if err != nil {
          return nil, fmt.Errorf("failed to determine local IP: %w", err)
      }
      
      strategy := &LocalFileServerStrategy{
          BaseDir:    baseDir,
          ServerPort: port,
          LocalIP:    localIP,
      }
      
      return strategy, nil
  }
  
  func (s *LocalFileServerStrategy) StartServer() error {
      // Create file server handler
      fileHandler := http.FileServer(http.Dir(s.BaseDir))
      
      // Create server mux
      mux := http.NewServeMux()
      mux.Handle("/", fileHandler)
      
      // Create server
      s.server = &http.Server{
          Addr:    fmt.Sprintf(":%d", s.ServerPort),
          Handler: mux,
      }
      
      // Start server in a goroutine
      go func() {
          if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
              log.Printf("File server error: %v", err)
          }
      }()
      
      return nil
  }
  
  func (s *LocalFileServerStrategy) GenerateLocationHint(filePath string) (string, error) {
      relPath, err := s.GetRelativePath(filePath)
      if err != nil {
          return "", err
      }
      
      return fmt.Sprintf("http://%s:%d/%s", s.LocalIP, s.ServerPort, relPath), nil
  }
  
  func (s *LocalFileServerStrategy) GetRelativePath(filePath string) (string, error) {
      absFilePath, err := filepath.Abs(filePath)
      if err != nil {
          return "", err
      }
      
      absBaseDir, err := filepath.Abs(s.BaseDir)
      if err != nil {
          return "", err
      }
      
      relPath, err := filepath.Rel(absBaseDir, absFilePath)
      if err != nil {
          return "", err
      }
      
      return filepath.ToSlash(relPath), nil
  }
  
  func (s *LocalFileServerStrategy) EnsureAccessibility(filePath string) error {
      // Check if file exists and is readable
      if _, err := os.Stat(filePath); err != nil {
          return fmt.Errorf("file accessibility check failed: %w", err)
      }
      
      // Check if file is within base directory
      relPath, err := s.GetRelativePath(filePath)
      if err != nil {
          return fmt.Errorf("file is not within base directory: %w", err)
      }
      
      // Try to access the file via HTTP to verify server is working
      url := fmt.Sprintf("http://%s:%d/%s", s.LocalIP, s.ServerPort, relPath)
      resp, err := http.Get(url)
      if err != nil {
          return fmt.Errorf("file is not accessible via HTTP: %w", err)
      }
      defer resp.Body.Close()
      
      if resp.StatusCode != http.StatusOK {
          return fmt.Errorf("file is not accessible via HTTP, status code: %d", resp.StatusCode)
      }
      
      return nil
  }
  
  func (s *LocalFileServerStrategy) StopServer() error {
      if s.server != nil {
          ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
          defer cancel()
          return s.server.Shutdown(ctx)
      }
      return nil
  }
  ```

- **Features**:
  - Automatically configures a local HTTP file server
  - Determines the local IP address for external accessibility
  - Generates appropriate URIs based on local IP and port
  - Adds these as location hints in the capability descriptor
  - Validates that files are accessible via the server
  - Provides options to keep the server running after registration

- **Advantages**:
  - Simple to use with minimal configuration
  - Works well for development and testing
  - No need for external hosting services

- **Limitations**:
  - Requires the developer's machine to be accessible from the internet
  - May not work behind NAT or firewalls without port forwarding
  - Not suitable for production use without additional security measures

### 2. **Existing Web Server Strategy**

- **Implementation**:
  ```go
  type WebServerStrategy struct {
      BaseDir  string
      BaseURL  string
  }
  
  func NewWebServerStrategy(baseDir, baseURL string) (*WebServerStrategy, error) {
      // Validate base URL format
      if _, err := url.Parse(baseURL); err != nil {
          return nil, fmt.Errorf("invalid base URL: %w", err)
      }
      
      // Ensure base URL ends with a slash
      if !strings.HasSuffix(baseURL, "/") {
          baseURL += "/"
      }
      
      return &WebServerStrategy{
          BaseDir: baseDir,
          BaseURL: baseURL,
      }, nil
  }
  
  func (s *WebServerStrategy) GenerateLocationHint(filePath string) (string, error) {
      relPath, err := s.GetRelativePath(filePath)
      if err != nil {
          return "", err
      }
      
      return s.BaseURL + relPath, nil
  }
  
  func (s *WebServerStrategy) GetRelativePath(filePath string) (string, error) {
      absFilePath, err := filepath.Abs(filePath)
      if err != nil {
          return "", err
      }
      
      absBaseDir, err := filepath.Abs(s.BaseDir)
      if err != nil {
          return "", err
      }
      
      relPath, err := filepath.Rel(absBaseDir, absFilePath)
      if err != nil {
          return "", err
      }
      
      return filepath.ToSlash(relPath), nil
  }
  
  func (s *WebServerStrategy) EnsureAccessibility(filePath string) error {
      // Check if file exists and is readable
      if _, err := os.Stat(filePath); err != nil {
          return fmt.Errorf("file accessibility check failed: %w", err)
      }
      
      // Get location hint
      hint, err := s.GenerateLocationHint(filePath)
      if err != nil {
          return fmt.Errorf("failed to generate location hint: %w", err)
      }
      
      // Try to access the file via HTTP to verify server is working
      resp, err := http.Get(hint)
      if err != nil {
          return fmt.Errorf("file is not accessible via HTTP: %w", err)
      }
      defer resp.Body.Close()
      
      if resp.StatusCode != http.StatusOK {
          return fmt.Errorf("file is not accessible via HTTP, status code: %d", resp.StatusCode)
      }
      
      return nil
  }
  ```

- **Features**:
  - Allows specifying a base URL for an existing web server
  - Generates relative paths from the base directory
  - Combines base URL and relative paths to create complete URIs
  - Validates that files are accessible via the web server

- **Advantages**:
  - Works with existing web hosting infrastructure
  - More reliable for production use
  - Better security and control over file access

- **Limitations**:
  - Requires manual setup of web server
  - Developer must ensure files are properly uploaded to the server
  - May require additional authentication or access control

### 3. **Distributed Storage Strategy (IPFS)**

- **Implementation**:
  ```go
  type IPFSStrategy struct {
      IPFSNode *ipfs.Node
      Gateway  string
  }
  
  func NewIPFSStrategy(gateway string) (*IPFSStrategy, error) {
      // Initialize IPFS node
      ctx := context.Background()
      node, err := ipfs.NewNode(ctx, &ipfs.Config{
          Gateway: gateway,
      })
      if err != nil {
          return nil, fmt.Errorf("failed to initialize IPFS node: %w", err)
      }
      
      return &IPFSStrategy{
          IPFSNode: node,
          Gateway:  gateway,
      }, nil
  }
  
  func (s *IPFSStrategy) GenerateLocationHint(filePath string) (string, error) {
      // Add file to IPFS
      file, err := os.Open(filePath)
      if err != nil {
          return "", fmt.Errorf("failed to open file: %w", err)
      }
      defer file.Close()
      
      // Add file to IPFS
      ctx := context.Background()
      cid, err := s.IPFSNode.Add(ctx, file)
      if err != nil {
          return "", fmt.Errorf("failed to add file to IPFS: %w", err)
      }
      
      // Pin the file to ensure it remains available
      if err := s.IPFSNode.Pin(ctx, cid); err != nil {
          return "", fmt.Errorf("failed to pin file: %w", err)
      }
      
      // Generate IPFS URI
      return fmt.Sprintf("ipfs://%s", cid.String()), nil
  }
  
  func (s *IPFSStrategy) GetRelativePath(filePath string) (string, error) {
      // For IPFS, the relative path is not used
      // Instead, we use the CID as the identifier
      return filepath.Base(filePath), nil
  }
  
  func (s *IPFSStrategy) EnsureAccessibility(filePath string) error {
      // Check if file exists and is readable
      if _, err := os.Stat(filePath); err != nil {
          return fmt.Errorf("file accessibility check failed: %w", err)
      }
      
      // Get location hint
      hint, err := s.GenerateLocationHint(filePath)
      if err != nil {
          return fmt.Errorf("failed to generate location hint: %w", err)
      }
      
      // Extract CID from hint
      cid := strings.TrimPrefix(hint, "ipfs://")
      
      // Try to access the file via IPFS gateway to verify it's accessible
      gatewayURL := fmt.Sprintf("%s/ipfs/%s", s.Gateway, cid)
      resp, err := http.Get(gatewayURL)
      if err != nil {
          return fmt.Errorf("file is not accessible via IPFS gateway: %w", err)
      }
      defer resp.Body.Close()
      
      if resp.StatusCode != http.StatusOK {
          return fmt.Errorf("file is not accessible via IPFS gateway, status code: %d", resp.StatusCode)
      }
      
      return nil
  }
  ```

- **Features**:
  - Adds files to the IPFS network
  - Generates content-addressed URIs (ipfs://CID)
  - Pins files to ensure persistence
  - Validates accessibility through IPFS gateways

- **Advantages**:
  - Decentralized storage with no single point of failure
  - Content addressing ensures integrity
  - No need for dedicated hosting infrastructure
  - Files remain accessible as long as they are pinned somewhere in the network

- **Limitations**:
  - Requires IPFS node setup or access to a pinning service
  - May have slower access times compared to direct HTTP
  - Requires devs to have IPFS support or access to an IPFS gateway

### Strategy Selection and Configuration

The CLI tool provides a flexible mechanism for selecting and configuring the appropriate file reference strategy:

```go
// FileReferenceConfig defines the configuration for file reference strategies
type FileReferenceConfig struct {
    Strategy          string   // "local", "web", "ipfs"
    BaseDir           string   // Base directory for relative paths
    ServerPort        int      // Port for local file server
    BaseURL           string   // Base URL for web server
    IPFSGateway       string   // IPFS gateway URL
    AdditionalHints   []string // Additional location hints
    ValidateAccess    bool     // Whether to validate accessibility
}

// CreateFileReferenceStrategy creates a file reference strategy based on configuration
func CreateFileReferenceStrategy(config FileReferenceConfig) (FileReferenceStrategy, error) {
    switch config.Strategy {
    case "local":
        return NewLocalFileServerStrategy(config.BaseDir, config.ServerPort)
    case "web":
        return NewWebServerStrategy(config.BaseDir, config.BaseURL)
    case "ipfs":
        return NewIPFSStrategy(config.IPFSGateway)
    default:
        return nil, fmt.Errorf("unsupported file reference strategy: %s", config.Strategy)
    }
}
```

## Testing and Validation

### Plugin Testing

1. **Unit Testing**
   - Test each plugin function independently
   - Verify correct behavior with various inputs
   - Test error handling and edge cases

2. **Integration Testing**
   - Test the plugin with the KNIRVCHAIN SDK
   - Verify correct interaction with the blockchain
   - Test end-to-end capability registration and invocation

3. **Performance Testing**
   - Measure resource usage (CPU, memory)
   - Test with high concurrency
   - Verify compliance with resource limits

### Validation Tools

The CLI tool provides methods for validating plugins and manifest files:

```go
// ValidatePlugin validates a plugin file
func ValidatePlugin(pluginPath string) error {
    // Check if file exists and is readable
    if _, err := os.Stat(pluginPath); err != nil {
        return fmt.Errorf("plugin file not found or not readable: %w", err)
    }
    
    // Check file extension
    if !strings.HasSuffix(pluginPath, ".so") && !strings.HasSuffix(pluginPath, ".dll") {
        return fmt.Errorf("plugin file must have .so or .dll extension")
    }
    
    // Try to load the plugin
    p, err := plugin.Open(pluginPath)
    if err != nil {
        return fmt.Errorf("failed to load plugin: %w", err)
    }
    
    // Check for required symbols based on manifest
    // ...
    
    return nil
}

// ValidateManifest validates a manifest file
func ValidateManifest(manifestPath string) error {
    // Check if file exists and is readable
    if _, err := os.Stat(manifestPath); err != nil {
        return fmt.Errorf("manifest file not found or not readable: %w", err)
    }
    
    // Read manifest file
    manifestData, err := ioutil.ReadFile(manifestPath)
    if err != nil {
        return fmt.Errorf("failed to read manifest file: %w", err)
    }
    
    // Parse JSON
    var manifest map[string]interface{}
    if err := json.Unmarshal(manifestData, &manifest); err != nil {
        return fmt.Errorf("failed to parse manifest JSON: %w", err)
    }
    
    // Validate required fields
    requiredFields := []string{"name", "version", "main", "interfaces"}
    for _, field := range requiredFields {
        if _, ok := manifest[field]; !ok {
            return fmt.Errorf("manifest is missing required field: %s", field)
        }
    }
    
    // Validate interfaces
    interfaces, ok := manifest["interfaces"].([]interface{})
    if !ok {
        return fmt.Errorf("manifest 'interfaces' field must be an array")
    }
    
    if len(interfaces) == 0 {
        return fmt.Errorf("manifest must specify at least one interface")
    }
    
    // Validate other fields
    // ...
    
    return nil
}
```

## Best Practices for Plugin Development

1. **Security**
   - Avoid using unsafe packages
   - Validate all inputs
   - Handle errors gracefully
   - Avoid excessive resource usage
   - Don't store sensitive information in the plugin

2. **Performance**
   - Optimize for speed and memory usage
   - Use efficient algorithms and data structures
   - Minimize external dependencies
   - Implement proper caching where appropriate
   - Handle concurrent invocations correctly

3. **Compatibility**
   - Follow the interface specifications exactly
   - Test with different versions of the SDK
   - Ensure backward compatibility when updating
   - Document any version-specific behavior
   - Handle different input formats gracefully

4. **Documentation**
   - Document the plugin's purpose and functionality
   - Provide clear examples of usage
   - Document all configuration options
   - Include troubleshooting information
   - Keep documentation up to date

## Implementation Priorities

1. **Core Components**
   - Interface definitions
   - Plugin loading and validation
   - opSchema validation
   - MCP server manager
   - Process execution engine
   - Example implementations

2. **MCP Integration Routes**
   - Extrapolation route implementation
   - Interpolation route implementation
   - Server deployment manager
   - Process orchestration

3. **File Reference Strategies**
   - Local file server implementation
   - Web server integration
   - IPFS support

4. **Registration Commands**
   - Direct plugin registration command
   - MCP server registration command
   - Operational procedure registration command
   - End-to-end registration flows

5. **Testing and Validation Tools**
   - Plugin validation tools
   - opSchema validation tools
   - MCP server validation tools
   - End-to-end testing framework

## Success Criteria

1. **Functionality**
   - Plugins can be developed and registered successfully
   - MCP servers can be registered and deployed
   - Operational procedures can be defined and executed
   - File reference strategies work correctly
   - All registration routes work end-to-end

2. **Usability**
   - Clear and helpful documentation
   - Intuitive error messages
   - Simple development workflow for all integration routes
   - Seamless transition between different approaches

3. **Security**
   - Secure plugin and server validation
   - Safe handling of plugin files and server deployments
   - Proper validation of all inputs
   - Secure communication between components

4. **Reliability**
   - Robust error handling
   - Consistent behavior across different environments
   - High test coverage
   - Reliable server deployment and management

5. **Integration**
   - Seamless integration between plugins and MCP servers
   - Effective orchestration of complex processes
   - Proper handling of dependencies between components
   - Efficient resource utilization

---

<div class="footer-links">


© 2025 KNIRV Network
</div>
