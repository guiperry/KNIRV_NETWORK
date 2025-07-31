

---

**Source**: KNIRVROOT/documentation/docsify/core-concepts/capabilities.md

# Capabilities System



## System Design Document: KNIRVROOT MCP Capabilities

### 1. Introduction

#### 1.1. Purpose:

This document outlines the design for a capability system within the KNIRVROOT ecosystem, leveraging the Model Context Protocol (MCP).

Capabilities are plugins registered on the KNIRVROOT, hosted and executed by client applications (e.g., "Inference Engines").

#### 1.2. Scope:

This document covers capability registration, discovery, download, client-side execution, and on-chain usage logging with NRN token fees.

It does not cover the internal implementation of the KNIRVROOT node or the client application beyond their interaction with capabilities.

#### 1.3. Audience:

*   Blockchain developers
*   Software architects
*   Client application developers
*   Security engineers

### 2. System Overview (Based on MCP\_Implementation.md)

#### 2.1. KNIRVROOT Node:

*   Implements MCP, handling `MCPRegisterCapability` and `MCPInvokeCapability` transactions.
*   Stores `ResourceDescriptor` (for capabilities) and `ContextRecord` (for invocations) on-chain.
*   Manages NRN token accounts and processes fees.
*   Exposes HTTP APIs for capability discovery and transaction submission.

#### 2.2. KNIRVROOT Hosting Client (Inference Engine):

*   Interacts with KNIRVROOT nodes to discover and log usage of capabilities.
*   Downloads capability binaries from hosting devs or other locations.
*   Executes capabilities locally in a sandboxed environment.
*   Registers capabilities on KNIRVROOT via `MCPRegisterCapabilityTransaction`.
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

*   **Capability Registration:** Developers must be able to register Capabilities on KNIRVROOT. The `ResourceDescriptor` must include metadata, `ContentHash` of the Capability package, and NRN usage fee. Its `Schema` field should contain:
    *   `Summary` (string): Brief description.
    *   `LocationHints` (array of strings): URIs for downloading the package.
    *   `ManifestFile` (string): Path to the manifest within the package.
    *   `ExecutableFile` (string): Path to the executable within the package.
    *   `OutputDirectoryHint` (string, optional): Hint for output/config directory.
*   **Capability Discovery:** Clients must be able to query KNIRVROOT to find registered Capabilities.
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
| agent Client          |----->| KNIRVROOT Node         |<-----| Capability Developer/       |
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

*   **KNIRVROOT Node:** As described in `MCP_Implementation.md`. Manages Decentralized Model Context Protocol (MCP) transactions and state.
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

*   **KNIRVROOT Node API:** As defined in `MCP_Implementation.md` (e.g., `POST /mcp/register_capability`, `POST /mcp/invoke/resource/{resource_id}` (where resource\_id is capability\_id), `GET /mcp/capability/{capability_id}`, `GET /mcp/capabilities`).
*   **Capability Execution Interface (Client-Capability):** Defined by the capability's `Schema`. For Go capabilities loaded via capability package, this would be exported functions and their signatures.

#### 4.7. Workflow:

1.  **Developer:** Creates capability, hosts it, registers it on KNIRVROOT (submits `MCPRegisterCapabilityTransaction`).
2.  **Client:**

    a. Discovers capability via KNIRVROOT API.

    b. Downloads capability package from `LocationHints`.

    c. Verifies package integrity using `ContentHash`. Locates manifest and executable using `Schema.ManifestFile` and `Schema.ExecutableFile`.

    d. Parses manifest and executes Capability plugin locally in a sandboxed environment, providing input according to the manifest's input schema.

    e. Receives output from capability.

    f. Submits `MCPInvokeCapabilityTransaction` (with `ContextRecord` and NRN fee) to KNIRVROOT API.

3.  **KNIRVROOT Node:**

    a. Validates and processes registration and invocation transactions.

    b. Updates blockchain state (adds `ResourceDescriptor`, `ContextRecord`).

    c. Transfers NRN fee from client to capability owner.

### 5. Security Design

*   **5.1. On-Chain Integrity:** `ContentHash` in `ResourceDescriptor` allows clients to verify downloaded capability integrity.
*   **5.2. Developer Signing (Recommended Off-Chain Practice):** Capability developers should sign their binaries. Clients can verify these signatures. This is an off-chain trust mechanism but can be supported by on-chain identity if developer public keys are also registered.
*   **5.3. Client-Side Sandboxing (CRITICAL):** The agent Client application must execute downloaded capabilities in a sandboxed environment (e.g., OS-level sandboxing, Wasm runtime sandbox, containerization) to limit potential harm from malicious or buggy capabilities.
*   **5.4. Client-Side Resource Limits:** The client should impose resource limits (CPU, memory, network access) on executed capabilities.
*   **5.5. Secure Download:** `LocationHints` should ideally use secure protocols (HTTPS, IPFS with verification).
*   **5.6. Blockchain Security:** Standard blockchain security practices apply to KNIRVROOT nodes and transactions.

### 6. Error Handling

*   **6.1. Client-Side Errors:**
    *   Capability download failures.
    *   Capability integrity verification failures (`ContentHash` mismatch).
    *   Capability loading errors (e.g., incompatible Go capability).
    *   Capability execution errors (runtime errors within the capability).
    *   The client application is responsible for handling these gracefully.

*   **6.2. Blockchain-Side Errors:**
    *   Invalid `MCPRegisterCapability` or `MCPInvokeCapability` transactions (e.g., insufficient NRN fee, malformed data).
    *   Handled by KNIRVROOT node validation logic.

### 7. Performance Considerations

*   **7.1. Client-Side Performance:** Capability loading and execution time are client-side concerns.
*   **7.2. Blockchain Performance:** Throughput of registration and invocation transactions depends on KNIRVROOT's underlying performance.
*   **7.3. Network Performance:** Capability download speed depends on the hosting dev's bandwidth and network conditions.

### 8. Testing

*   **8.1. Unit Tests:**
    *   Write unit tests for the capability system to ensure that it is working correctly.
    *   For individual capabilities (by developers).
    *   For client-side capability loading/execution logic.
    *   For KNIRVROOT node's MCP transaction handling.
*   **8.2. Integration Tests:**
    *   Write integration tests to ensure that the capability system integrates correctly with the blockchain core.
    *   End-to-end flow: capability registration -> client discovery -> download -> execution -> usage logging.
*   **8.3. Security Tests:**
    *   Perform security tests to ensure that the capability system is secure.
    *   Test client-side sandboxing effectiveness.
    *   Test integrity verification.
    *   Test KNIRVROOT API security.

### 9. Deployment

*   **9.1. Deployment Process:**

Capability Developer: Compiles capability, makes it available for download, registers it on KNIRVROOT.

agent Client: Deployed independently. Users interact with it to use capabilities.

KNIRVROOT Network: Nodes are deployed and run the blockchain.

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

---

<div class="footer-links">


© 2025 KNIRV Network
</div>
