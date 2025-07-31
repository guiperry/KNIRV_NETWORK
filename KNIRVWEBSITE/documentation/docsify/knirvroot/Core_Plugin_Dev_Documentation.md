

---

**Source**: KNIRVROOT/docs/TODOs/Core_Plugin_Dev_Documentation.md

## I. KNIRVCHAIN Capability Developer Documentation (The "How-To" Guide)

This is the main guide that walks developers through creating, registering, and managing their capabilities.

**Introduction to KNIRVCHAIN Capabilities:**

*   What are capabilities in the KNIRVCHAIN ecosystem? (Client-side execution, on-chain registration, MCP integration).
*   Benefits for developers (monetization via NRN, reaching users, standardized interaction).
*   High-level architecture (Client-Blockchain-Plugin Host interaction).

**Getting Started:**

*   Prerequisites (Go, Wasm tools, etc., depending on supported plugin types).
*   Setting up a KNIRVCHAIN development environment (running a local node, getting test NRN tokens).
*   Overview of the capability lifecycle (Develop -> Compile -> Host -> Register -> (Client Invokes) -> Get Paid).

**Plugin Development - The "How-To":**

*   **Supported Plugin Types & Languages:**
    *   Detailed instructions for each supported type (e.g., Go `.so` plugins, WebAssembly modules).
    *   **Go Plugins (`-buildmode=plugin`):**
        *   Project structure.
        *   Defining exported functions (the plugin's entry points).
        *   Input/output data structures (how to define them, best practices for serialization if needed beyond simple types).
        *   Managing dependencies.
        *   Compilation steps.
        *   Example "Hello World" Go plugin.
    *   **WebAssembly (Wasm) Plugins:**
        *   Supported source languages (Rust, C++, AssemblyScript, etc.) and toolchains for compiling to Wasm.
        *   Interface definition (e.g., WASI, or custom import/export functions).
        *   How to pass complex data to/from Wasm.
        *   Example "Hello World" Wasm plugin.

*   **Plugin Input/Output Schemas:**
    *   How to define the JSON Schema for your plugin's expected input and output. This schema will be part of the ResourceDescriptor on-chain.
    *   Why this is important (client validation, discoverability, type safety).
    *   Tools for creating and validating JSON Schemas.

*   **Best Practices:**
    *   Writing secure plugin code (input validation, avoiding side effects where possible if the plugin is meant to be pure).
    *   Error handling within the plugin and how to report errors back to the client.
    *   Performance considerations.
    *   Idempotency (if applicable to the plugin's function).

**Plugin Manifest / Flow Definition (Your YAML Idea - Excellent!)**

This is where your idea of a YAML file specifying the plugin's flow comes in. Let's call it a `agent-plugin-manifest.yaml`.

*   **Purpose:**
    *   Provides structured metadata about the plugin beyond what's in the on-chain ResourceDescriptor.
    *   Defines the sequence of operations or "events" the plugin performs.
    *   Specifies points where the agent Client (Inference Engine) can or should inject context, use other MCP Tools/Resources, or prompt the user.
    *   Helps clients understand how to interact with more complex, multi-step plugins.

*   **Potential YAML Structure:**

```yaml
pluginApiVersion: "1.0"
id: "com.example.skyrimmodvalidator" # Unique identifier, could match on-chain ID
name: "Skyrim Mod Validator"
version: "1.2.0"
description: "Validates Skyrim mod files for compatibility and errors using TES5Edit."
author: "PluginDevX"
license: "MIT"

# Entry point for the client to call in the plugin binary/module
entryFunction: "ValidateAsset" # For Go .so, this is the exported function name

# Input/Output schemas (can reference external JSON schema files or be inline)
inputSchema: "./schemas/input.json" # Path relative to manifest
outputSchema: "./schemas/output.json"

# Defines the operational flow of the plugin
flow:
  - step: 1
    name: "Initialize TES5Edit"
    description: "Sets up the TES5Edit environment. Requires path to TES5Edit."
    # Specifies if the client (Inference Engine) needs to provide something
    # or can use an MCP capability.
    clientInjection:
      point: "before" # or "after" or "during" (if plugin calls back)
      type: "config" # "config", "resource", "tool", "userPrompt"
      prompt: "Please provide the full path to your TES5Edit.exe:" # If type is userPrompt
      # If type is "resource" or "tool", specify MCP capability details
      # mcpCapabilityId: "tool-filesystem-picker"
      # mcpInputSchema: { "type": "string", "description": "Path to TES5Edit" }
    outputs:
      - name: "tes5editSessionId"
        type: "string"
        description: "ID for the initialized TES5Edit session."

  - step: 2
    name: "Load Mod File"
    description: "Loads the specified mod file into TES5Edit."
    # This step uses the main input provided to the plugin's entryFunction
    # as defined by inputSchema.
    # It might also use outputs from previous steps.
    inputs:
      - name: "assetUrl" # From main plugin input
        source: "pluginInput.AssetURL"
      - name: "tes5editSessionId" # From previous step
        source: "step[1].outputs.tes5editSessionId"
    clientInjection: # Example: Client uses an MCP tool to download the assetUrl
      point: "before"
      type: "tool"
      mcpCapabilityId: "tool-file-downloader"
      mcpInput: { "url": "{{pluginInput.AssetURL}}" } # Templating from plugin input
      mcpOutputMapping:
        downloadedFilePath: "localAssetPath" # Maps tool output to a variable for this step
    pluginAction: # Describes what the plugin code for this step does
      function: "loadMod" # Hypothetical internal function or part of entryFunction
      params: ["localAssetPath", "tes5editSessionId"]
    outputs:
      - name: "loadStatus"
        type: "boolean"

  - step: 3
    name: "Run Validation Checks"
    description: "Performs validation checks using TES5Edit."
    inputs:
      - name: "tes5editSessionId"
        source: "step[1].outputs.tes5editSessionId"
    # Example: Inference engine injects reasoning before this step
    clientInjection:
      point: "before"
      type: "inference" # Special type for client-side LLM call
      promptTemplate: |
        Given the mod file being validated (details not directly available here, but assume context),
        and the user's goal of ensuring compatibility, what specific TES5Edit checks
        should be prioritized? List up to 3.
      llmConfig: # Optional: hints for the client's LLM
        model: "user_preferred_reasoning_model"
      outputMapping:
        llmSuggestions: "prioritizedChecks" # LLM output becomes available to plugin
    pluginAction:
      function: "runChecks"
      params: ["tes5editSessionId", "prioritizedChecks"] # Plugin uses LLM suggestions
    outputs:
      - name: "validationReport" # This would map to ValidateAssetOutput
        source: "pluginOutput" # Indicates this is the final output

# Optional: Define required MCP capabilities the client must provide access to
requiredClientCapabilities:
  - capabilityId: "tool-file-downloader"
    description: "Needed to download assets specified by URL."
  - capabilityId: "mcp-logging-service" # For structured logging by the plugin via client
    description: "For the plugin to log its progress via the client's MCP context."
```

This manifest would be packaged with the plugin binary/code. The agent Client would download the plugin and its manifest.
The manifest helps the client orchestrate more complex interactions, especially if the plugin itself isn't a single monolithic function but a series of steps where the client can intervene or provide resources.

*   **Hosting Your Plugin:**

    *   Recommendations for making the plugin binary and its manifest accessible (e.g., direct dev hosting, IPFS, public web server).
    *   How `LocationHints` in the ResourceDescriptor are used.

*   **Registering Your Plugin on KNIRVCHAIN:**

    *   Step-by-step guide on creating the `ResourceDescriptor` for a plugin.
    *   Calculating the `ContentHash` of the plugin package (binary + manifest).
    *   Populating the `ResourceDescriptor.Schema` field:
        *   `Schema.Summary`: A concise description of your plugin.
        *   `Schema.LocationHints`: URIs where your plugin package can be downloaded.
        *   `Schema.ManifestFile`: The relative path to your `agent-plugin-manifest.yaml` within the package.
        *   `Schema.ExecutableFile`: The relative path to your plugin's main executable within the package.
        *   `Schema.OutputDirectoryHint` (optional): A suggestion for clients on where the plugin might store outputs or configurations.
    *   Using the Wallet Server (`/wallet/mcp/create_register_capability`) or constructing the transaction manually to submit it.
    *   Setting the `GasFeeNRN`.

*   **Client Interaction & Execution Flow (from Plugin Dev's Perspective):**

    *   How the agent Client discovers the plugin.
    *   How it downloads the plugin and manifest.
    *   How it uses the manifest (if present) to prepare inputs and call the plugin's `entryFunction`.
    *   How the client handles `clientInjection` points defined in the manifest.
    *   How NRN fees are processed.

*   **Updating Your Plugin:**

    *   Process for releasing new versions (new ResourceDescriptor with updated version and ContentHash).

*   **Security Best Practices for Plugin Developers:**

    *   Signing your plugin binaries (off-chain).
    *   Input validation within your plugin.
    *   Principle of least privilege.

*   **Troubleshooting & Support:**

    *   Common issues and how to resolve them.
    *   Where to get help.

## II. Plugin Manifest Specification (Detailed Document)

This document would formalize the structure and semantics of the `agent-plugin-manifest.yaml` file.

*   Detailed explanation of each top-level field (`pluginApiVersion`, `id`, `name`, etc.).
*   Specification for `inputSchema` and `outputSchema` (e.g., must be valid JSON Schema).
*   Detailed specification for the `flow` array and `step` objects:
    *   `name`, `description`.
    *   `inputs`: how to specify sources (`pluginInput.<field>`, `step[<index>].outputs.<name>`).
    *   `clientInjection`:
        *   `point`: `before`, `after`, `onRequest` (plugin calls out to client).
        *   `type`: `config` (simple value), `resource` (fetch via MCP), `tool` (invoke MCP tool), `userPrompt` (ask user), `inference` (client runs LLM).
        *   Details for each type (e.g., `mcpCapabilityId`, `promptTemplate`, `llmConfig`, `outputMapping`).
    *   `pluginAction`: (Optional) If a step involves calling a specific internal function of the plugin beyond the main entry point.
    *   `outputs`: defining outputs of a step.
*   Specification for `requiredClientCapabilities`.
*   Versioning for the manifest specification itself.

## III. API Reference for Client-Plugin Interaction (if plugins can call back to the client)

If your `clientInjection.point` includes something like `onRequest` where the plugin needs to pause and request something from the client (e.g., "Fetch this URL for me using your network stack and security context"), you'd need to define that callback API.

*   What functions can the plugin call on the client?
*   How is data passed back and forth?
*   Security implications.

## IV. Examples and Tutorials

*   Multiple complete examples of different types of plugins (simple data transformer, one using an external library like your TES5Edit example, a multi-step plugin using the manifest).
*   Tutorials for each supported language.

**Why the Manifest is a Powerful Idea:**

Your YAML manifest idea is excellent because:

*   **Clarity:** It makes the plugin's operation explicit and understandable by the client.
*   **Orchestration:** It allows the agent Client (Inference Engine) to be a true orchestrator, not just a dumb executor. The client can:
    *   Fulfill resource dependencies for a plugin step using other MCP capabilities (tools/resources).
    *   Inject LLM-driven reasoning or planning between plugin steps.
    *   Prompt the user for input at specific points in a complex plugin's workflow.
*   **Modularity:** Complex tasks can be broken down into smaller, manageable plugin steps, with the client managing the flow.
*   **Flexibility:** Different clients could potentially interpret the manifest slightly differently or provide different implementations for clientInjection points (e.g., one client uses a local LLM for inference steps, another uses a cloud LLM).
*   **Discoverability of Needs:** The `requiredClientCapabilities` section clearly tells the client what other MCP services the plugin relies on the client to provide access to.

This comprehensive set of documentation, especially with the detailed manifest specification, will empower developers to build powerful and well-integrated plugins for KNIRVCHAIN, and allow your "Inference Engine" client to interact with them in very sophisticated ways.
```

---

<div class="footer-links">


© 2025 KNIRV Network
</div>
