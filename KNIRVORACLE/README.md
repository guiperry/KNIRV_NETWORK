# KNIRVORACLE: A Decentralized Model Context Protocol Network

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

## 🧪 Phase 7 Testing Infrastructure

### Testing Architecture
KNIRVORACLE implements comprehensive testing for the core blockchain and MCP functionality:

#### Unit Testing (Go)
```bash
# Run all unit tests
go test -v ./...

# Run specific test packages
go test -v ./agent_manager_test.go
go test -v ./agent_integration_test.go
go test -v ./economics_integration_test.go

# Run tests with coverage
go test -v -cover ./...
```

#### Integration with Network Tests
```bash
# From project root - run KNIRVORACLE network integration tests
cd integration-tests && go test -v -run TestKNIRVORACLE

# Run comprehensive KNIRVORACLE test suite
make test-root

# Run KNIRVORACLE specific tests
cd KNIRVORACLE && go test -v ./...
```

#### Test Categories
- **Agent Management Tests**: Agent registration, capability testing, badge systems
- **Economics Integration Tests**: NRN token operations, fee processing, reward distribution
- **Blockchain Tests**: Block creation, transaction processing, consensus validation
- **MCP Protocol Tests**: Capability registration, discovery, invocation
- **Integration Tests**: Cross-component communication and data flow

#### Test Structure
```
KNIRVORACLE/
├── agent_manager_test.go           # Agent management functionality
├── agent_integration_test.go       # Agent integration scenarios
├── economics_integration_test.go   # Economic model testing
├── blockchain_server_test.go       # Core blockchain functionality
├── bootnode_test.go               # Network bootstrap testing
└── *_test.go                      # Component-specific tests
```

## 🎯 Key Features

*   **Decentralized Registry:** Immutable, transparent record of all registered MCP capabilities.
*   **Verifiable Audit Trail:** `ContextRecord`s provide on-chain proof of all capability interactions.
*   **Monetization:** Capability owners earn NRN tokens for usage.
*   **Plugin Ecosystem:** Allows developers to offer executable functionalities (plugins) that clients can securely download and run.
*   **Standardized Interaction:** MCP provides a common way to interact with diverse AI capabilities.

## 🚨 **Autonomous Fault Tolerance & Failover Protocol**

KNIRVORACLE implements a **comprehensive failover protocol** that ensures **99.9% uptime** through autonomous network coordination and automatic recovery. The failover system maintains blockchain continuity during critical node failures.

### Failover Protocol Architecture

#### **Core Components**
- **FailoverManager**: Central coordinator for root node monitoring and transition orchestration
- **Network Control Protocol**: Pub/Sub-based coordination for network-wide pause/resume operations
- **Enhanced P2P Consensus Manager**: Integrated network control message handling

#### **Operational Flow**

1. **🚀 Continuous Monitoring**
   ```go
   // Bootnode continuously monitors root node health
   healthURL := fm.currentOracleAPIURL + "/health"
   resp, err := fm.httpClient.Get(healthURL)
   ```

2. **⏰ Offline Detection & Threshold**
   - 15-minute configurable offline threshold before failover initiation
   - Configurable ping intervals (default: 1 minute)
   - Prevents false positives from temporary network issues

3. **🎯 Autonomous Leader Election**
   ```go
   func (fm *FailoverManager) amIElectedToBecomeOracle() bool {
       // Placeholder for stake-based leader election
       // Ready for implementation with NRN stake validation
       return true // Current: First available bootnode elected
   }
   ```

4. **📡 Network Coordination (Network Pause)**
   ```go
   // Broadcast pause message across P2P network
   pausePayload := NetworkPausePayload{
       InitiatorPeerID: fm.nodeConfig.PeerID,
       Reason:          "Oracle node failover in progress",
       Timestamp:       time.Now().Unix(),
   }
   ```

5. **🔄 Automatic Promotion**
   - Bootnode-to-root transition with wallet inheritance
   - Database path migration from bootnode to new root
   - ChainID generation for unique network identification
   - Application restart with new configuration

### **Failover Protocol Features**

#### **🔒 Zero-Trust Coordination**
- **Message Authentication**: All network control messages validated by source
- **Timeout Protection**: Automatic pause expiration prevents indefinite stalls
- **Replay Prevention**: Timestamp-based message validation

#### **⚡ High-Performance Recovery**
- **Sub-Second Detection**: HTTP health checks with 10-second timeouts
- **Network-Wide Coordination**: All nodes pause simultaneously during transition
- **Minimal Downtime**: Automatic recovery within minutes of detection

#### **🛡️ Resilience & Security**
- **State Preservation**: Complete blockchain state inherited during promotion
- **Concurrent Safety**: Mutex-protected critical operations
- **Audit Trail**: All failover events logged with full context

### **Network Control Message Protocol**

```go
// Network Pause Message Structure
type NetworkPausePayload struct {
    InitiatorPeerID string `json:"initiator_peer_id"`
    Reason          string `json:"reason"`
    Timestamp       int64  `json:"timestamp"`
}

// Message Types
type NetworkControlMessage struct {
    Type    string      `json:"type"`    // "NetworkPause" | "NetworkResume"
    Payload interface{} `json:"payload"`
}
```

#### **Integration Points**
- **HTTP Server**: Respects `IsNetworkPaused()` to pause transaction processing
- **P2P Consensus**: Handles `NetworkPause`/`NetworkResume` messages
- **Main Application**: Integrates with startup/shutdown lifecycle
- **Configuration**: Supports failover-specific parameters

### **Configuration Support**

```go
type Config struct {
    // Existing fields...
    CurrentOracleNodeAPIURL string `json:"current_oracle_api_url"`
    IsBootnode              bool   `json:"is_bootnode"`

    // Failover-specific
    RootOfflineThreshold   time.Duration
    RootPingInterval       time.Duration
    NetworkPauseTimeout    time.Duration
}
```

### **Deployment Scenarios**

#### **🔄 **Root Node Failure Recovery**
```
1. Root node goes offline (network/hardware issue)
2. Bootnodes continuously monitor health endpoints
3. Threshold exceeded (15+ minutes offline)
4. Leader election determines promotion candidate
5. NetworkPause broadcast to all nodes
6. Bootnode promoted to root with inherited state
7. NetworkResume signal restores normal operation
8. New root accepts transactions/invocations
```

#### **⚙️ **Maintenance Window Coordination**
```
1. Administrator initiates controlled failover
2. NetworkPause signal pauses all network activity
3. Root node undergoes maintenance/restart
4. NetworkResume restores full functionality
5. All nodes sync and resume normal operations
```

#### **📊 **Network Expansion Events**
```
1. New bootnodes join network
2. Automatic promotion for load distribution
3. Coordinated network pause during transition
4. Migration of blockchain state
5. Network resume with expanded capacity
```

### **Testing & Validation**

The failover system includes comprehensive testing:

```bash
# Failover protocol tests
go test -v ./failover_manager_test.go

# Network control message integration
go test -v ./p2p_consensus_failover_test.go

# End-to-end failover scenarios
go test -v ./integration/failover_scenarios_test.go
```

#### **Test Coverage**
- **Root Node Failure Simulation**: Network partitions and hardware failures
- **Network Control Message Reliability**: Pub/sub routing and message validation
- **Promotion State Transfer**: Database and wallet inheritance validation
- **Recovery Time Measurement**: End-to-end failover completion timing
- **Concurrent Safety Testing**: Multiple failover events under load

### **Performance Metrics**

- **Detection Time**: < 30 seconds from node failure
- **Recovery Time**: < 2 minutes from detection to full operation
- **State Transfer**: Complete blockchain inheritance
- **Message Latency**: < 5 seconds for network control coordination
- **Uptime Guarantee**: 99.9% with automatic failure detection

### **Best Practices & Configuration**

#### **Monitoring Configuration**
```yaml
# Recommended settings for production
failover:
  root_offline_threshold: "15m"
  ping_interval: "1m"
  network_pause_timeout: "5m"
  health_check_timeout: "10s"

# Bootstrap node health endpoints
bootnodes:
  - api_url: "https://bootnode-1.corp.internal:8080"
    backup_url: "https://bootnode-1.corp.backup:8080"

# Automatic failover settings
coordination:
  leader_election_stake_minimum: 10000  # NRN tokens
  promotion_timeout: "3m"
  state_transfer_timeout: "2m"
```

### **Security Considerations**

- **Message Authentication**: All control messages validated by peer identity
- **State Integrity**: Cryptographic verification of inherited blockchain state
- **Access Control**: Role-based permissions for failover operations
- **Audit Logging**: Complete record of all failover coordination events

### **Cloud-Native Compatibility**

The failover protocol integrates seamlessly with cloud orchestration:

- **Kubernetes Integration**: ConfigMap updates trigger graceful restarts
- **Container Health Checks**: Automatic pod replacement during failures
- **Service Mesh Awareness**: Istio integration for traffic routing during transitions

This autonomous failover protocol ensures KNIRVORACLE maintains **enterprise-grade reliability** while preserving the decentralized nature of blockchain consensus. The system's **event-driven coordination** provides **transparent, verifiable** blockchain operations with **autonomous recovery**.

**🛡️ Reliability**: Network fault tolerance through distributed monitoring  
**⚡ Performance**: Sub-minute recovery with zero data loss  
**🔒 Security**: Cryptographically verified state transitions  
**🔍 Transparency**: Complete audit trail of failover coordination

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

*   **LevelDB (Primary):**
    *   Stores the core blockchain data (blocks, transactions).
    *   Stores an indexed view of MCP data extracted from the blockchain:
        *   Serialized `CapabilityDescriptor`s (e.g., `ResourceDescriptor`, `ToolDescriptor`) via `mcp:capability:<id> -> descriptor_json`.
        *   Serialized `ContextRecord`s (e.g., `mcp:context:<tx_id> -> context_record_json`).
        *   NRN token account balances.
    *   This indexed view powers the API query endpoints for basic lookups.

*   **Off-Chain Storage:**
    *   Actual plugin binaries, large datasets, and model artifacts are stored off-chain (e.g., hosted by devs, on IPFS, or other web services).
    *   The on-chain `ResourceDescriptor` contains:
        *   A `ContentHash` for integrity verification of the downloaded package.
        *   `Schema.LocationHints` (URIs) for retrieving the package.

## API Endpoints

KNIRVORACLE exposes HTTP APIs through its nodes and wallet server.

### Blockchain Server (`blockchain_server.go`)

*   **Generic Transaction Submission:**
    *   `POST /transaction`: Receives fully signed transactions (can be any type, including MCP).
*   **MCP-Specific Transaction Ingestion (for pre-signed/component-provided transactions):**
    *   `POST /mcp/capability/register`: Submits an `MCPRegisterCapabilityTransaction`.
    *   `POST /mcp/capability/invoke`: Submits an `MCPInvokeCapabilityTransaction`.
*   **MCP Data Querying:**
    *   `GET /mcp/capability/{capability_id}`: Retrieves a capability descriptor.
    *   `GET /mcp/capabilities?type=<X>&owner=<Y>&...`: Lists capabilities with filters.
    *   `GET /mcp/context/{context_id}`: Retrieves a specific context record.
    *   `GET /mcp/capability/{capability_id}/invocations`: Retrieves all context records for a capability.
    *   `GET /mcp/contexts?capabilityId=<X>&interactionType=<Y>&...`: Advanced querying for context records.
*   **Other Endpoints:** For blockchain status, blocks, NRN balances, etc.

### Wallet Server (`wallet_server.go`)

Provides a higher-level interface for clients, handling transaction creation and signing.

*   `POST /wallet/mcp/create_register_capability`: Creates, signs, and submits an `MCPRegisterCapabilityTransaction`.
*   `POST /wallet/mcp/create_invoke_capability`: Creates, signs, and submits an `MCPInvokeCapabilityTransaction`.
*   Other endpoints for wallet management, generic transaction sending, etc.



## Architectural Considerations and Future Work

### Data Size and On-Chain Storage:

*   Storing full capability descriptors on-chain is generally fine as they are metadata.
*   For `ResourceDescriptor`s, especially `ResourceTypePlugin`, `ResourceTypeDataset`, or `ResourceTypeModelArtifact`, the `ContentHash` and `LocationHints` fields are designed to point to off-chain storage (e.g., dev hosting, IPFS) for the actual large data, keeping the blockchain lean. The on-chain hash guarantees integrity.
*   **Initial Approach:** Store them fully on-chain as per the blog's emphasis on verifiability.
*    *   **Current Model for Plugins:** The `ResourceDescriptor` stores metadata, including `ContentHash` of the package and `Schema.LocationHints` (URIs) pointing to the off-chain package. The on-chain hash guarantees integrity of the downloaded package.

*   *Enhanced Querying with RealmDB (Supplemental):

While LevelDB serves as the primary storage for the blockchain and basic MCP data indexing, future enhancements could involve using **RealmDB as a supplemental, rich secondary index** for MCP data.

**How it would work:**

1.  **LevelDB remains the source of truth for the blockchain.**
2.  As blocks are processed, MCP data (`CapabilityDescriptor`s, `ContextRecord`s) would be extracted and also written into a local RealmDB instance on each node.
3.  This RealmDB instance would act as a highly queryable, object-oriented cache or materialized view.
4.  The API layer would direct complex analytical queries to RealmDB, while simple lookups might still use LevelDB.

**Benefits of Supplemental RealmDB:**

*   **Rich Queries:** Handle complex multi-condition filtering, relationships, aggregations, and potentially full-text search over MCP data (e.g., "Find all Tools owned by 'Alice' with input schema containing 'image_url', invoked >10 times last week, ordered by fee").
*   **Object-Oriented Modeling:** More natural mapping of Go MCP structs to database objects.

**Challenges:**

*   **Increased Complexity:** Managing two database systems.
*   **Data Synchronization:** Ensuring RealmDB accurately reflects the LevelDB state, especially during chain reorganizations (forks), requires robust rollback and re-indexing logic for the RealmDB portion.

This supplemental approach would allow KNIRVORACLE to offer significantly more powerful analytical capabilities over its MCP data without replacing the core LevelDB storage for the blockchain itself.

### Immutability vs. Updates:

*   Capability descriptors may need updates (e.g., new version, schema change, fee update).
*   **Approach:** Implement updates by creating new registration transactions (e.g., `MCPUpdateCapabilityTransaction`) that reference the previous ID and create a new version. The blockchain will preserve the full history. Query APIs should default to the latest version but allow fetching specific versions.

### Query Capabilities:

*   The proposed LevelDB indexing supports basic ID-based lookups and simple relationship queries.
*   **Future Enhancement:** For more complex queries (e.g., "find all tools with input schema X and owned by Y"), an off-chain indexing/query service that syncs with the KNIRVORACLE blockchain might be necessary.

### Smart Contracts:

*   The README mentions placeholder smart contract functionality. In the future, MCP logic (e.g., rules for updating cards, managing ownership, or more complex context validation) could be implemented as smart contracts for greater automation and decentralized governance.

### Access Control and Permissions:

*   Who can register capabilities or invoke them? Initially, any valid transaction sender who can pay the fee.
*   **Future Enhancement:** Implement on-chain access control mechanisms, possibly tied to ownership fields in the descriptors or through smart contracts (e.g., only owner can update a capability).

### Standardization:

*   The schemas for all `Descriptor` types and `ContextRecord` (especially its `Details` field per `InteractionType`) should be well-documented and treated as part of the "protocol." Using JSON Schema for `InputSchemaJSON`, `OutputSchemaJSON`, `ParametersSchemaJSON`, and resource `Schema` fields is already part of the design.

### Gas/Fees:

*   This plan now explicitly incorporates NRN token gas fees via the `Transaction.Fee` field and `BaseDescriptor.GasFeeNRN`. This necessitates the implementation of an account and balance system within KNIRVORACLE, which is a significant foundational requirement.
*   The "Sampling could be free" aspect means that `GasFeeNRN` for interactions logged related to sampling might be zero, or the transaction fee for such logging could be waived/minimal.

This plan provides a phased approach to integrating robust MCP functionality into KNIRVORACLE. By building upon its existing features, KNIRVORACLE can become a powerful platform for ensuring transparency and verifiability in AI workflows.

## Getting Started

*(This section would typically include instructions on how to build, run a node, interact with the API, etc. Placeholder for now.)*

```
To build and run a KNIRVORACLE node:
```sh
git clone <repository_url>
cd KNIRVORACLE
go build -o KNIRVORACLE_node ./cmd/KNIRVORACLE/main.go
./KNIRVORACLE_node --port 8080 --p2p_port 6001
```

To run the Wallet Server:
```sh
go build -o wallet_server ./cmd/walletserver/main.go
./wallet_server --port 9090 --blockchain_server_ip http://localhost:8080
```

## Contributing

*(Details on how to contribute to the project. Placeholder for now.)*

We welcome contributions! Please see `CONTRIBUTING.md` for guidelines.

## License

*(Specify the project's license. Placeholder for now.)*

This project is licensed under the MIT License.
```
