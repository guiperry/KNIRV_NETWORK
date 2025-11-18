# KNIRVORACLE Documentation

KNIRVORACLE is a blockchain-based platform designed to facilitate a transparent, verifiable, and monetizable ecosystem for AI capabilities. It implements the Model Context Protocol (MCP), allowing various AI-related functionalities (Tools, Prompts, Plugins, Memory Services) to be registered, discovered, invoked, and audited on-chain. Each interaction with an MCP capability results in a blockchain transaction, creating an immutable record and potentially involving NRN token fees.

## Core Concepts

*   **Model Context Protocol (MCP):** A set of standards for defining, discovering, and interacting with AI capabilities. KNIRVORACLE serves as a decentralized backend for MCP.
*   **Capabilities:** These are the core building blocks of the MCP ecosystem on KNIRVORACLE. They include:
    *   **Resources:** Expose structured or dynamic content. This includes:
        *   **Plugins:** Developer-uploaded executables (e.g., Go `.so`, Wasm) hosted by devs, discovered on-chain, downloaded, and run by clients.
        *   **Datasets:** References to datasets with on-chain metadata and hashes.
        *   **Model Artifacts:** References to AI model files.
        *   **APIs:** Descriptors for external APIs.
    *   **Tools:** Executable actions defined with JSON schemas for input and output. Can be backed by plugins or external APIs.
    *   **Prompts:** Reusable, parameterized prompt templates for LLMs.
    *   **Memory Services:** Capabilities providing persistent, graph-like memory stores.
*   **Capability Descriptors:** Metadata structures (e.g., `ResourceDescriptor`, `ToolDescriptor`) registered on the blockchain.
    *   `ResourceDescriptor` (for plugins/resources): Defines properties, owner, NRN gas fee. Includes a `ContentHash` of the capability package. Its nested `Schema` field contains a `Summary`, `LocationHints` for package download, and paths to the `ManifestFile` and `ExecutableFile` within the package.
    *   Other descriptors like `ToolDescriptor`, `PromptDescriptor` define their respective capability types.
*   **ContextRecord:** An on-chain record logged for every significant MCP interaction (e.g., capability invocation, registration). It details what happened, when, by whom, and with what references (hashes, fees), forming an immutable audit trail.
*   **NRN Tokens:** The native utility token of KNIRVORACLE, used for:
    *   Paying gas fees for registering and invoking MCP capabilities.
    *   Compensating capability owners (e.g., plugin developers).
*   **KNIRVORACLE Node:** The core blockchain dev software that maintains the ledger, processes MCP transactions, and exposes an API.
*   **Wallet Server:** A service that helps users create, sign, and submit MCP transactions to the KNIRVORACLE network.
*   **agent Client (Inference Engine):** An application that interacts with KNIRVORACLE to discover, download (for plugins), execute, and log the usage of MCP capabilities.

## Architecture Overview

1.  **Capability Registration:**
    *   Developers register their capabilities (e.g., a new Plugin) by submitting an `MCPRegisterCapabilityTransaction` to the KNIRVORACLE.
    *   This transaction contains the serialized `CapabilityDescriptor` (e.g., `ResourceDescriptor` for a plugin, including its `ContentHash`, `LocationHints`, and `GasFeeNRN`).
    *   The `ResourceDescriptor` is stored on-chain.

2.  **Capability Discovery:**
    *   agent Clients query the KNIRVORACLE API (e.g., `GET /mcp/capabilities?resourceType=PLUGIN`) to find available capabilities.

3.  **Capability Invocation & Plugin Execution:**
    *   **For Plugins:**
        1.  The agent Client downloads the plugin binary from a `LocationHint` provided in its `ResourceDescriptor`.
        2.  The client verifies the downloaded package's integrity using the on-chain `ContentHash`.
        3.  The client executes the plugin locally (ideally in a sandboxed environment).
    *   **For other capabilities (Tools, Prompts):** The client prepares input according to the capability's schema.
    *   **Logging Usage:** The client (or wallet server on its behalf) creates an `MCPInvokeCapabilityTransaction`.
        *   This transaction includes a `ContextRecord` detailing the invocation (e.g., `CapabilityID`, `InteractionType`, `Initiator`, `InputHash`, `OutputHash`).
        *   It also includes the NRN `Fee` specified by the capability.
        *   This transaction is submitted to the KNIRVORACLE, creating an on-chain audit log.

## Key Features

*   **Decentralized Registry:** Immutable, transparent record of all registered MCP capabilities.
*   **Verifiable Audit Trail:** `ContextRecord`s provide on-chain proof of all capability interactions.
*   **Monetization:** Capability owners earn NRN tokens for usage.
*   **Plugin Ecosystem:** Allows developers to offer executable functionalities (plugins) that clients can securely download and run.
*   **Standardized Interaction:** MCP provides a common way to interact with diverse AI capabilities.

## `ContextRecord` Scenarios: The Audit Trail in Action

The `ContextRecord` is central to KNIRVORACLE's value, providing an on-chain, verifiable log. Here are some examples:

*   **Using a Registered Tool:**
    *   An AI agent uses a "PriceOracleTool." A `ContextRecord` is logged with `InteractionType: "TOOL_INVOCATION"`, `CapabilityID` of the tool, `Initiator` (agent's address), `InputHash`, `OutputHash`, and the NRN `Fee` paid.
    *   *Querying:* `GET /mcp/capability/{tool_id}/invocations` to see all tool uses.

*   **Utilizing a Registered Prompt:**
    *   An application uses a "StoryGeneratorPrompt." A `ContextRecord` logs `InteractionType: "PROMPT_USAGE"`, `CapabilityID`, `Initiator`, and `InputHash` (of prompt parameters).
    *   *Querying:* `GET /mcp/context/{tx_id}` to see specific prompt parameters used.

*   **Executing a Registered Plugin:**
    *   A agent Client runs a "ModValidatorPlugin." A `ContextRecord` logs `InteractionType: "PLUGIN_EXECUTION"`, `CapabilityID`, `Initiator`, `InputHash` (of the mod file), and `OutputHash` (of the validation report).
    *   *Querying:* `GET /mcp/contexts?initiator={client_address}&capabilityId={plugin_id}` for a client's usage history of a plugin.

*   **Interacting with a Memory Service:**
    *   An AI writes to a "UserPreferencesMemory" service. A `ContextRecord` logs `InteractionType: "MEMORY_WRITE"`, `CapabilityID`, `Initiator`, and `InputHash` (of the preference data).
    *   *Querying:* `GET /mcp/contexts?capabilityId={memory_id}&interactionType=MEMORY_WRITE` to audit memory writes.

*   **Logging a Sampling Event (Server-Side):**
    *   An MCP server initiates a client-side LLM task. A `ContextRecord` logs `InteractionType: "SAMPLING_REQUEST_SENT"`, `Initiator` (server's address), and `InputHash` (of the task).

*   **Auditing Capability Registrations:**
    *   (Optional but recommended) When a new capability is registered via `MCPRegisterCapabilityTransaction`, a `ContextRecord` with `InteractionType: "CAPABILITY_REGISTRATION"` can also be logged, providing a unified feed of all MCP events.
    *   *Querying:* `GET /mcp/contexts?interactionType=CAPABILITY_REGISTRATION` for a list of all new capabilities.

These records are crucial for billing, provenance, debugging, and analytics within the KNIRVORACLE MCP ecosystem.

## Data Storage


## Documentation Sections

- [Getting Started](./getting-started/)
- [Core Concepts](./core-concepts/)
- [Protocols](./protocols/)
- [API Reference](./api-reference/)
- [Components](./components/)
- [Guides](./guides/)
- [Troubleshooting](./troubleshooting/)
- [SDK Documentation](./sdk/)
