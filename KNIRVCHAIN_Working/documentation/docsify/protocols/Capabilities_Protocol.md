# Capabilities Protocol

KNIRVCHAIN **Capabilities** are executable software **plugins** registered on the blockchain. They enable developers to offer specific functionalities to client applications (like an "Inference Engine"). These Capabilities are packaged (including an executable and a manifest file), hosted by the registering dev or on decentralized storage, discovered via their on-chain `ResourceDescriptor`, downloaded by the client, and executed locally by the client. Each invocation whose usage is logged on-chain incurs an NRN token fee, paid to the Capability's owner.

Here's a breakdown of what that architecture would look like:

**Core Idea:**

Capabilities are self-contained, executable software plugins (e.g., compiled Go plugins, WebAssembly modules) that provide specific functionalities. They are not executed *by* the KNIRVCHAIN nodes themselves as part of blockchain consensus or transaction processing. Instead, KNIRVCHAIN acts as a decentralized registry and marketplace for these Capabilities.

**Components:**

1.  **Custom Blockchain Core (Go):**
    *   Implements the Decentralized Model Context Protocol (MCP).
    *   Handles MCP transactions:
        *   `TransactionTypeMCPRegisterCapability`: Allows developers to register their Capabilities. The transaction data includes a `ResourceDescriptor` (with `ResourceType` typically set to `PLUGIN` or a similar new `CAPABILITY` type).
        *   `TransactionTypeMCPInvokeCapability`: Allows clients to log the usage of a Capability. The transaction data includes a `ContextRecord` detailing the invocation and the Network Retreival Number (NRN) token fee.
    *   Provides API endpoints for clients to:
        *   Discover registered Capabilities (e.g., `GET /mcp/capabilities?type=RESOURCE&resourceType=PLUGIN`).
        *   Submit registration and invocation transactions.
    *   Manages NRN token balances and fee distribution.

2.  **Example Capabilities (Plugins):**
    *   Self-contained executables or loadable modules (e.g., Go code compiled as a plugin, Wasm modules) that perform specific tasks. Examples:
        *   **Asset Validation Template:** Code to validate asset formats, check for malicious code, extract metadata.
        *   **Engine Integration Template (e.g., Skyrim):** Code to interact with the Skyrim modding tools (TES5Edit, BAI, LOOT).
        *   **Data Transformation Capability:** Converts data from one format to another.
        *   **Specialized Computation Capability:** Performs a specific calculation.
    
    *   **Capability Structure (Example for a Go capability):** Each capability should have a defined structure and interface for the client to interact with.
        *   **Input:** A clearly defined set of inputs (e.g., transaction data, asset data).
        *   **Output:** A clearly defined set of outputs (e.g., validation result, payment status).
        *   **Dependencies:** A list of external Go libraries required by the Capability.
    *   **Example Capability Plugin (Asset Validation - Executed by Client):**

```go
package main

import (
	"fmt"
    "github.com/shiroco16/go-tes5edit" // TES5Edit for asset analysis
    // This capability would be compiled to .so or an executable
)

// ValidateAssetInput defines the input for the asset validation capability.
type ValidateAssetInput struct {
	AssetURL  string // URL of the asset file.
	AssetType string // Type of asset (NIF, DDS, etc.).
}

// ValidateAssetOutput defines the output for the asset validation template.
type ValidateAssetOutput struct {
	IsValid  bool    // True if the asset is valid.
	Errors   []string // List of errors.
	Metadata map[string]interface{} // Extracted metadata.
}

// ValidateAsset is the function the client application would call after loading/executing this capability.
func ValidateAsset(input ValidateAssetInput) ValidateAssetOutput { // This could be the exported function from a .so capability
	output := ValidateAssetOutput{IsValid: true}

	// Download the asset file.
	assetFile := input.AssetURL // Replace with actual download logic

	// TES5Edit Example (run by the client that invoked this capability)
    tes5editInstance := tes5edit.NewTES5Edit("/path/to/TES5Edit.exe")
    records, err := tes5editInstance.GetRecords()
    if err != nil {
        output.IsValid = false
		output.Errors = append(output.Errors, fmt.Sprintf("TES5Edit: %s", err.Error()))
        return output
	}
    fmt.Println(records)
	// Asset validation logic (e.g., check format, scan for malicious code).
	// ... client-side asset validation code here ...

	// Extract metadata.
    output.Metadata = map[string]interface{}{
		"author": "Example Author",
		"version": "1.0",
	}


	return output
}
```


3.  **Client Application: Inference Engine (Go):** Template Loading, LLM interaction, and Execution Mechanism.

*   **Plugin System (Preferred):** Use Go's plugin system (`plugin` package) to dynamically load and execute the code templates. This allows you to load the templates at runtime without recompiling the Inference Engine.

    *   **Compilation:** Templates are compiled into Go plugin files (`.so` on Linux, `.dylib` on macOS, `.dll` on Windows).
    *   **Loading:** The Inference Engine loads the plugin file at startup or when a new template is registered.
    *   **Execution:** The Inference Engine calls on the KNIRVCHAIN to process NRN transactions within the capability (e.g., `ValidateAsset`, `ImportAsset`) when parsing functions.
*   **Reflection (Less Preferred):** Use Go's reflection package to inspect and execute code in the templates. This is more flexible but can be slower and more complex.
*   **Code Generation (Potentially):** Generate Go code from the templates and compile it into the blockchain core. This is the least flexible but can be the most performant.

4. Configuration System:

Allows each dev chain to configure which templates to use and how they are executed.

Configuration file (e.g., YAML, JSON) specifies:

*   The path to the template file.
*   The functions to call within the template.
*   The logical process of functions.
*   The input/output parameters to pass to the functions.
*   When, where, and how inference can interact with the process.

5. Security Model:

*   **Template Signing:** Sign the code templates with a digital signature to ensure that they haven't been tampered with. The blockchain core verifies the signature before loading the template.
*   **Sandboxing (Highly Recommended):** Execute the code templates in a sandboxed environment to limit the damage that malicious code can cause. Use a containerization technology like Docker or a virtual machine.
*   **Resource Limits:** Impose resource limits (CPU, memory, disk) on the code templates to prevent them from consuming too many resources.

6. SDK/Template Library:

Provide a set of pre-built, well-documented Go code templates that developers can use as a starting point.

Include templates for common tasks like:

*   Asset validation.
*   Chain-of-Thought Reasoning, etc.
*   Documentation and examples for each template.
*   Best practices for developing secure and efficient templates.
*   Basic client integration.

Clearly define the template interface (input/output parameters).

### Workflow Example (Asset Validation):

1.  **Transaction Received:** The dev chain receives a transaction containing an asset URL.
2.  **Transaction Pre-Processing:** The Inference Engine triggers the transaction pre-processing hook.
3.  **Asset Validation Template Loaded:** The Inference Engine loads the configured "Asset Validation" template.
4.  **Template Executed:** The Inference Engine calls the `ValidateAsset` function in the template, passing the asset URL as input.
5.  **Validation Logic Runs:** The `ValidateAsset` function downloads the asset, validates its format, scans it for malicious code, and extracts metadata.
6.  **Validation Result Returned:** The `ValidateAsset` function returns a `ValidateAssetOutput` struct containing the validation result.
7.  **Transaction Processed (or Rejected):** If the asset is valid, the Inference Engine sends the transaction to the KNIRVCHAIN for an NRN URI minted. If the asset is invalid, the blockchain core rejects the transaction.

### agent Client (Inference Engine):

An application that enables llm inference within custom capability functionalities.

*   **Discovers capabilities:** Queries the KNIRVCHAIN API.
*   **Registers capabilities:** Allows developers to upload new capability plugins and registers the capability on the blockchain.
*   **Invokes capabilities:** Fetches the capability binary/code from the location specified in the capability's `ResourceDescriptor` (e.g., from the hosting dev or IPFS). Verifies integrity using `ContentHash` in exchange for a small NRN token fee.
*   **Executes capabilities locally:** Loads and runs the capability code in its own environment. This might involve using Go's plugin package for `.so` files, a Wasm runtime, or simply executing a downloaded binary.
*   **Logs capability usage:** Submits an `MCPInvokeCapabilityTransaction` to the KNIRVCHAIN, including a `ContextRecord` and the NRN fee.
*   **Capability Hosting Client:** Makes the capability binary/code available for download at the `LocationHints` specified in the `ResourceDescriptor` via the KNIRVCHAIN DHT. Receives NRN token fees when clients log usage of their registered capabilities.

### Capability Lifecycle & Workflow: 

1.  **Development:** A developer creates a capability (e.g., a Go program compiled with `go build -buildmode=plugin` or a standalone executable).
2.  **Registration (On-Chain):**

    The developer (or their client node) creates an `MCPRegisterCapabilityTransaction`.

    The transaction's `Data` field contains a `ResourceDescriptor`:

    *   `CapabilityType`: `CapabilityTypeResource`
    *   `ResourceType`: `ResourceTypeCapability`
    *   `ID`: A unique ID for the capability.
    *   `Name`, `Version`, `Description`.
    *   `Owner`: The developer's KNIRVCHAIN address.
    *   `GasFeeNRN`: The NRN fee required for a client to log usage of this capability.
    *   `ContentHash`: SHA256 hash of the capability binary/package.
    *   `Schema`: JSON Schema defining the capability's expected input and output.
    *   `LocationHints`: Array of URIs (e.g., `agent://<dev_id>.chain/capabilities/<capability_id>`, IPFS CID) where the capability binary can be downloaded.

    This transaction is submitted to the KNIRVCHAIN network.
3.  **Discovery (By Client):**

    The agent Client queries the KNIRVCHAIN API (e.g., `GET /mcp/capabilities?resourceType=TOOL&name=AssetValidator`) to find suitable capabilities.
4.  **Download & Verification (By Client):**

    The client selects a capability and retrieves its `ResourceDescriptor`.

    It uses one of the `LocationHints` (now found within `ResourceDescriptor.Schema.LocationHints`) to download the **Capability package**.

    The client verifies the downloaded package's integrity by comparing its hash against the `ContentHash` from the `ResourceDescriptor`.

    The client then uses `ResourceDescriptor.Schema.ManifestFile` and `ResourceDescriptor.Schema.ExecutableFile` to locate the manifest and executable within the package.
5.  **Local Execution (By Client):**

    The client parses the `agent-plugin-manifest.yaml`.

    The client prepares input data according to the input schema defined *in the manifest*.

    The client receives the output from the capability.
6.  **Usage Logging & Fee Payment (By Client):**

    The client creates an `MCPInvokeCapabilityTransaction`.

    The transaction's `Data` field contains a `ContextRecord`:

    *   `CapabilityID`: The ID of the capability that was executed.
    *   `InteractionType`: `PLUGIN_EXECUTION`.
    *   `Initiator`: The client's KNIRVCHAIN address.
    *   `InputHash` (optional): Hash of the input data provided to the capability.
    *   `OutputHash` (optional): Hash of the output data received from the capability.
    *   `Details`: Other relevant metadata (e.g., capability version used, execution duration).

    The transaction's `Fee` field is set to the `GasFeeNRN` specified in the capability's `ResourceDescriptor`.

    The client signs and submits this transaction to the KNIRVCHAIN network. The fee is ultimately credited to the capability's `Owner`.

### Benefits:

*   Flexibility: Allows a diverse ecosystem of functionalities to be offered to clients.
*   Extensibility: New capabilities can be added to the network without core blockchain changes.
*   Monetization for Developers: Capability developers earn NRN tokens for usage.
*   Decentralized Registry: Blockchain provides a transparent and immutable record of available capabilities and their terms.
*   Client-Side Execution: Reduces load on blockchain nodes and gives clients control over the code they run.

### Drawbacks:

*   Client-Side Security Burden: The client application is responsible for safely downloading, verifying, and executing potentially untrusted capability code (sandboxing is crucial).
*   Capability Availability: Relies on the hosting client to keep the capability binary available for download. `LocationHints` could point to decentralized storage like IPFS to mitigate this.
*   Complexity for Client: Client applications need to manage capability discovery, download, and execution.

### Important Considerations:

*   Capability Signing (Developer-Side): Developers should sign their capability binaries. Clients can verify these signatures against a known developer public key (potentially also registered on-chain or discovered through other means).
*   Client-Side Sandboxing: This is paramount. Clients should execute downloaded capabilities in a restricted environment to prevent malicious behavior.

This architecture focuses the "capability" concept to a client-side plugin facilitated by the blockchain network availability and persistance.


## System Design Document: KNIRVCHAIN MCP Capabilities

### 1. Introduction

#### 1.1. Purpose:

This document outlines the design for a capability system within the KNIRVCHAIN ecosystem, leveraging the Model Context Protocol (MCP).

Capabilities are plugins registered on the KNIRVCHAIN, hosted and executed by client applications (e.g., "Inference Engines").

#### 1.2. Scope:

This document covers capability registration, discovery, download, client-side execution, and on-chain usage logging with NRN token fees.

It does not cover the internal implementation of the KNIRVCHAIN node or the client application beyond their interaction with capabilities.

#### 1.3. Audience:

*   Blockchain developers
*   Software architects
*   Client application developers
*   Security engineers

### 2. System Overview (Based on MCP\_Implementation.md)

#### 2.1. KNIRVCHAIN Node:

*   Implements MCP, handling `MCPRegisterCapability` and `MCPInvokeCapability` transactions.
*   Stores `ResourceDescriptor` (for capabilities) and `ContextRecord` (for invocations) on-chain.
*   Manages NRN token accounts and processes fees.
*   Exposes HTTP APIs for capability discovery and transaction submission.

#### 2.2. KNIRVCHAIN Hosting Client (Inference Engine):

*   Interacts with KNIRVCHAIN nodes to discover and log usage of capabilities.
*   Downloads capability binaries from hosting devs or other locations.
*   Executes capabilities locally in a sandboxed environment.
*   Registers capabilities on KNIRVCHAIN via `MCPRegisterCapabilityTransaction`.
*   Hosts capability binaries for download.
*   Sends & Receives NRN fees for capability usage.

#### 2.3. Capability Developer:

*   Develops capabilities.
*   Uses the Inference Engine.


### 3. Goals and Requirements

#### 3.1. Goals:

*   Enable a decentralized marketplace for LLM executable functionalities (capabilities).
*   Allow clients to discover and utilize these capabilities.
*   Provide a mechanism for capability developers to be compensated via NRN tokens.
*   Maintain the integrity and transparency of LLM capability registration and usage logging via the blockchain.

#### 3.2. Functional Requirements:

*   **Capability Registration:** Developers must be able to register Capabilities on KNIRVCHAIN. The `ResourceDescriptor` must include metadata, `ContentHash` of the Capability package, and NRN usage fee. Its `Schema` field should contain:
    *   `Summary` (string): Brief description.
    *   `LocationHints` (array of strings): URIs for downloading the package.
    *   `ManifestFile` (string): Path to the manifest within the package.
    *   `ExecutableFile` (string): Path to the executable within the package.
    *   `OutputDirectoryHint` (string, optional): Hint for output/config directory.
*   **Capability Discovery:** Clients must be able to query KNIRVCHAIN to find registered Capabilities.
*   **Capability Package Download:** Clients must be able to download Capability packages.
*   **Capability Package Verification:** Clients must verify downloaded package integrity using the on-chain `ContentHash`.the integrity of downloaded capabilities using the on-chain content hash.
*   **Client-Side Execution:** Clients must execute capabilities locally.
*   **Usage Logging & Fee Payment:** Clients must be able to log capability usage on-chain via an MCP transaction, which includes payment of the NRN fee.

#### 3.3. Non-Functional Requirements:

*   **Security:** Focus on client-side security (sandboxing, signature verification) and integrity of on-chain records.
*   **Performance:** Capability execution performance is a client-side concern. Blockchain performance relates to transaction processing for registration/invocation.
*   **Decentralization:** Capability discovery and usage logging are decentralized. Capability hosting can be decentralized (e.g., IPFS).

### 4. System Design

#### 4.1. Architecture Diagram:

```
+-----------------------+ +-------------------------+ +-----------------------+
| agent Client          |----->| KNIRVCHAIN Node         |<-----| Capability Developer/       |
| (Inference Engine)    | | (Blockchain API, MCP)     | |         |
+-----------------------+ +-------------------------+ +-----------------------+
   |  1. Discover Capability (API GET) |      ^                              |  A. Develops Capability
   |                               |      |                              |  B. Registers Capability
   |  2. Download Capability ----------|------|------------------------------|     (MCPRegisterCapabilityTx)
   |     (from Hosting Peer/LocationHints)                              |  C. Hosts Capability Binary
   |                               |      |
   |  3. Execute Capability Locally    |      |
   |     (Sandboxed, Verified)     |      |
   |                               |      |
   |  4. Log Usage (API POST) -----|----->|
   |     (MCPInvokeCapabilityTx + NRN Fee)
   |                               |
   V                               |
+-------------------------+ +---------------------------+
| Capability Execution Result | | Blockchain State Update     |
| (Used by Client)        | | (ContextRecord logged,      |
+-------------------------+ | Fee transferred)           |
                               +---------------------------+

```

#### 4.2. Component Description:

*   **KNIRVCHAIN Node:** As described in `MCP_Implementation.md`. Manages Decentralized Model Context Protocol (MCP) transactions and state.
*   **agent Client (Inference Engine):** Application that discovers, registers, downloads, hosts, executes, and logs usage of capabilities. Responsible for local security.
*   **Capability:** A compiled piece of code (e.g., Go `.so`, Wasm, executable) with defined inputs/outputs. Its `ResourceDescriptor` is stored on-chain.
*   **Capability Developer:** Creates, registers, and hosts the capability binary. Receives NRN fees.

#### 4.3. Capability System Implementation:

*   **Go capability Package (Client-Side):**
    *   Use the `plugin` package from the Go standard library to dynamically load and execute Go code.
    *   The agent Client can use Go's `plugin` package if it's a Go application and the capabilities are Go `.so` files.
    *   Pros:
        *   Native support for Go capabilities.
        *   Relatively simple to use.
    *   Cons:
        *   Capabilities must be compiled separately from the blockchain core.
        *   Capabilities must be compatible with the same Go version as the blockchain core.
        *   Capabilities must be compatible with the client's Go version and OS/architecture.
        *   Limited support for cross-platform compatibility.

*   **Other Capability Formats (Client-Side):** Clients can support Wasm capabilities (via a Wasm runtime) or standalone executables (run as subprocesses). This choice depends on the client's architecture.

*   **Capability Interface:** The `Schema` field in the `ResourceDescriptor` (e.g., JSON Schema) defines the expected input/output structure for the capability, allowing clients to interact with it correctly.

#### 4.4. Capability Lifecycle:

*   **Capability Development:**
    *   Developers create Go code that implements the desired functionality.
*   **Capability Compilation:**
    *   Developers compile the Go code into a capability (`.so`, `.dylib`, or `.dll`) using the `go build -buildmode=capability` command.
        *   `go build -buildmode=capability -o validateAsset.so validateAsset/validateAsset.go`
*   **Capability Registration (On-Chain):** As described in the "Capability Lifecycle & Workflow" section above, using `MCPRegisterCapabilityTransaction`. The `ResourceDescriptor`'s `Schema` field will include:
    *   `ContentHash` of the package.
    *   `Schema.Summary`
    *   `Schema.LocationHints` for the package.
    *   `Schema.ManifestFile` path within the package.
    *   `Schema.ExecutableFile` path within the package.
    *   `Schema.OutputDirectoryHint` (optional).
*   **Client-Side Discovery, Download, Verification, Execution, and Usage Logging:** As described in the "Capability Lifecycle & Workflow" section. The client will use the manifest for execution details.
*   **Capability Update:** The developer releases a new package, registers a new version of the Capability (new `ResourceDescriptor` with an updated `Version`, `ContentHash`, etc.). Clients can choose to use the new version.Old versions remain on-chain unless explicitly deprecated by some mechanism.

#### 4.5. Data Model:

*   **On-Chain Data:**
    *   `ResourceDescriptor` (for Capabilities, with `ResourceType: "PLUGIN"` or similar):
        *   Key fields from `BaseDescriptor`: `ID`, `Owner`, `Name`, `Version`, `Description`, `GasFeeNRN`, `Timestamp`.
        *   `ResourceType`, `ContentHash` (of package).
        *   `Schema` (PluginSchemaDetail struct):
            *   `Summary`
            *   `LocationHints` (for package download).
            *   `ManifestFile` (path within package).
            *   `ExecutableFile` (path within package).
            *   `OutputDirectoryHint` (optional).
    *   `ContextRecord` (for invocations): Defined in `MCP_Implementation.md`. Key fields include `CapabilityID` (capability ID), `InteractionType: PLUGIN_EXECUTION`, `Initiator`, `InputHash`, `OutputHash`.
    *   NRN Token Account Balances.
*   **Off-Chain Data:**
    *   Capability binaries/code hosted by devs or decentralized storage.

#### 4.6. Interface Design:

*   **KNIRVCHAIN Node API:** As defined in `MCP_Implementation.md` (e.g., `POST /mcp/register_capability`, `POST /mcp/invoke/resource/{resource_id}` (where resource\_id is capability\_id), `GET /mcp/capability/{capability_id}`, `GET /mcp/capabilities`).
*   **Capability Execution Interface (Client-Capability):** Defined by the capability's `Schema`. For Go capabilities loaded via capability package, this would be exported functions and their signatures.

#### 4.7. Workflow:

1.  **Developer:** Creates capability, hosts it, registers it on KNIRVCHAIN (submits `MCPRegisterCapabilityTransaction`).
2.  **Client:**

    a. Discovers capability via KNIRVCHAIN API.

    b. Downloads capability package from `LocationHints`.

    c. Verifies package integrity using `ContentHash`. Locates manifest and executable using `Schema.ManifestFile` and `Schema.ExecutableFile`.

    d. Parses manifest and executes Capability plugin locally in a sandboxed environment, providing input according to the manifest's input schema.

    e. Receives output from capability.

    f. Submits `MCPInvokeCapabilityTransaction` (with `ContextRecord` and NRN fee) to KNIRVCHAIN API.

3.  **KNIRVCHAIN Node:**

    a. Validates and processes registration and invocation transactions.

    b. Updates blockchain state (adds `ResourceDescriptor`, `ContextRecord`).

    c. Transfers NRN fee from client to capability owner.

### 5. Security Design

*   **5.1. On-Chain Integrity:** `ContentHash` in `ResourceDescriptor` allows clients to verify downloaded capability integrity.
*   **5.2. Developer Signing (Recommended Off-Chain Practice):** Capability developers should sign their binaries. Clients can verify these signatures. This is an off-chain trust mechanism but can be supported by on-chain identity if developer public keys are also registered.
*   **5.3. Client-Side Sandboxing (CRITICAL):** The agent Client application must execute downloaded capabilities in a sandboxed environment (e.g., OS-level sandboxing, Wasm runtime sandbox, containerization) to limit potential harm from malicious or buggy capabilities.
*   **5.4. Client-Side Resource Limits:** The client should impose resource limits (CPU, memory, network access) on executed capabilities.
*   **5.5. Secure Download:** `LocationHints` should ideally use secure protocols (HTTPS, IPFS with verification).
*   **5.6. Blockchain Security:** Standard blockchain security practices apply to KNIRVCHAIN nodes and transactions.

### 6. Error Handling

*   **6.1. Client-Side Errors:**
    *   Capability download failures.
    *   Capability integrity verification failures (`ContentHash` mismatch).
    *   Capability loading errors (e.g., incompatible Go capability).
    *   Capability execution errors (runtime errors within the capability).
    *   The client application is responsible for handling these gracefully.

*   **6.2. Blockchain-Side Errors:**
    *   Invalid `MCPRegisterCapability` or `MCPInvokeCapability` transactions (e.g., insufficient NRN fee, malformed data).
    *   Handled by KNIRVCHAIN node validation logic.

### 7. Performance Considerations

*   **7.1. Client-Side Performance:** Capability loading and execution time are client-side concerns.
*   **7.2. Blockchain Performance:** Throughput of registration and invocation transactions depends on KNIRVCHAIN's underlying performance.
*   **7.3. Network Performance:** Capability download speed depends on the hosting dev's bandwidth and network conditions.

### 8. Testing

*   **8.1. Unit Tests:**
    *   Write unit tests for the capability system to ensure that it is working correctly.
    *   For individual capabilities (by developers).
    *   For client-side capability loading/execution logic.
    *   For KNIRVCHAIN node's MCP transaction handling.
*   **8.2. Integration Tests:**
    *   Write integration tests to ensure that the capability system integrates correctly with the blockchain core.
    *   End-to-end flow: capability registration -> client discovery -> download -> execution -> usage logging.
*   **8.3. Security Tests:**
    *   Perform security tests to ensure that the capability system is secure.
    *   Test client-side sandboxing effectiveness.
    *   Test integrity verification.
    *   Test KNIRVCHAIN API security.

### 9. Deployment

*   **9.1. Deployment Process:**

Capability Developer: Compiles capability, makes it available for download, registers it on KNIRVCHAIN.

agent Client: Deployed independently. Users interact with it to use capabilities.

KNIRVCHAIN Network: Nodes are deployed and run the blockchain.

*   **9.2. Rollback:**

Capability Update: Developers can register new versions. Clients can choose which version to use.

Client/Node Software: Standard software rollback procedures.

### 10. Future Considerations

*   **10.1. Capability Versioning:**

The `Version` field in `BaseDescriptor` (part of `ResourceDescriptor`) already supports this.

Clients need logic to select/prefer versions.

*   **10.2. Enhanced Discovery:** More sophisticated querying for capabilities (e.g., by tags, categories, reputation).
*   **10.3. Capability Reputation System:** On-chain or off-chain mechanisms for rating or vouching for capabilities.
*   **10.4. Decentralized Capability Hosting:** Stronger emphasis on using IPFS or other decentralized storage for `LocationHints` to improve availability and censorship resistance.

This revised SDD aligns the capability concept with the MCP architecture, emphasizing client-side execution and blockchain-as-a-registry.
```