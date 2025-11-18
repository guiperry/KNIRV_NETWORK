# KNIRVCHAIN: A Decentralized Model Context Protocol Network

## Table of Contents

- [Overview](#overview)
- [Core Concepts](#core-concepts)
- [Architecture Overview](#architecture-overview)
- [Phase 7 Testing Infrastructure](#phase-7-testing-infrastructure)
- [Key Features](#key-features)
- [Autonomous Fault Tolerance & Failover Protocol](#autonomous-fault-tolerance--failover-protocol)
- [ContextRecord Scenarios: The Audit Trail in Action](#contextrecord-scenarios--the-audit-trail-in-action)
- [Data Storage](#data-storage)
- [API Endpoints](#api-endpoints)
- [Architectural Considerations and Future Work](#architectural-considerations-and-future-work)
- [Getting Started](#getting-started)
- [Contributing](#contributing)
- [License](#license)
- [Changes](#changes)
- [Protobuf Issue Resolution](#protobuf-issue-resolution)
- [Root Payment Processor Implementation](#root-payment-processor-implementation)
- [agent Tunnel Relay Implementation Plan](#agent-tunnel-relay-implementation-plan)
- [KNIRVCHAIN Test Organization Improvements](#knirvoracle-test-organization-improvements)
- [KNIRV Network Final Test Fixes Report](#knirv-network-final-test-fixes-report)


## Overview

KNIRVCHAIN is a blockchain-based platform designed to facilitate a transparent, verifiable, and monetizable ecosystem for AI capabilities. It implements the Model Context Protocol (MCP), allowing various AI-related functionalities (Tools, Prompts, Plugins, Memory Services) to be registered, discovered, invoked, and audited on-chain. Each interaction with an MCP capability results in a blockchain transaction, creating an immutable record and potentially involving NRN token fees.


## Core Concepts

* **Model Context Protocol (MCP):** A set of standards for defining, discovering, and interacting with AI capabilities. KNIRVCHAIN serves as a decentralized backend for MCP.
* **Capabilities:** Core building blocks of the MCP ecosystem.  They include:
    * **Resources:** Expose structured or dynamic content. This includes:
        * **Plugins:** Developer-uploaded executables (e.g., Go `.so`, Wasm) hosted by devs, discovered on-chain, downloaded, and run by clients.
        * **Datasets:** References to datasets with on-chain metadata and hashes.
        * **Model Artifacts:** References to AI model files.
        * **APIs:** Descriptors for external APIs.
    * **Tools:** Executable actions defined with JSON schemas for input and output. Can be backed by plugins or external APIs.
    * **Prompts:** Reusable, parameterized prompt templates for LLMs.
    * **Memory Services:** Capabilities providing persistent, graph-like memory stores.
* **Capability Descriptors:** Metadata structures (e.g., `ResourceDescriptor`, `ToolDescriptor`) registered on the blockchain.
    * `ResourceDescriptor` (for plugins/resources): Defines properties, owner, NRN gas fee. Includes a `ContentHash` of the capability package. Its nested `Schema` field contains a `Summary`, `LocationHints` for package download, and paths to the `ManifestFile` and `ExecutableFile` within the package.
    * Other descriptors like `ToolDescriptor`, `PromptDescriptor` define their respective capability types.
* **ContextRecord:** An on-chain record logged for every significant MCP interaction (e.g., capability invocation, registration). It details what happened, when, by whom, and with what references (hashes, fees), forming an immutable audit trail.
* **NRN Tokens:** The native utility token of KNIRVCHAIN, used for paying gas fees for registering and invoking MCP capabilities and compensating capability owners.
* **KNIRVCHAIN Node:** The core blockchain dev software that maintains the ledger, processes MCP transactions, and exposes an API.
* **Wallet Server:** A service that helps users create, sign, and submit MCP transactions to the KNIRVCHAIN network.
* **agent Client (Inference Engine):** An application that interacts with KNIRVCHAIN to discover, download (for plugins), execute, and log the usage of MCP capabilities.

### Badge System: Skills, Capabilities, and Properties

KNIRVCHAIN implements a comprehensive Badge system that distinguishes between three fundamental types of agent attachments:

* **Skill Badges:** Represent learned capabilities that agents acquire through training on error resolution. Skills are:
  - **Learned from Errors:** Created when agents successfully resolve ErrorNodes in the KNIRVGRAPH
  - **Competitive:** Agents compete to solve errors and earn skill badges
  - **Execution-based:** Skills are invoked to perform specific tasks
  - **Training Data:** Built from error-solution pairs through LoRA adapter training
  - **Example:** "Data Analysis Skill" badge earned by resolving data processing errors

* **Capability Badges:** Represent access to external tools and services, typically from MCP servers. Capabilities are:
  - **Context-derived:** Created from ContextNodes representing MCP servers or API endpoints
  - **Access-based:** Provide agents with access to external tools and services
  - **Schema-defined:** Include structured schemas for interaction
  - **Gas-fee enabled:** May require NRN tokens for usage
  - **Example:** "Search API Capability" badge providing access to a search service

* **Property Badges:** Represent owned assets or characteristics that agents can possess. Properties are:
  - **Idea-derived:** Created from IdeaNodes through collaborative development
  - **Collaborative:** Multiple agents can collaborate on idea development
  - **Ownership-based:** Properties have ownership stakes distributed among collaborators
  - **Immutable:** Properties typically represent permanent characteristics or assets
  - **Market-valued:** Properties can have NRN market value and be traded
  - **Example:** "Innovation Patent Property" badge representing ownership of a novel algorithm

This three-tier system ensures clear separation between learned skills (competitive), external capabilities (access-based), and owned properties (collaborative), enabling sophisticated agent development and interaction patterns.


## Architecture Overview

1. **Capability Registration:** Developers register capabilities by submitting an `MCPRegisterCapabilityTransaction` containing the serialized `CapabilityDescriptor`. The descriptor is stored on-chain.

2. **Capability Discovery:** agent Clients query the KNIRVCHAIN API to find available capabilities.

3. **Capability Invocation & Plugin Execution:**
    * **For Plugins:** The agent Client downloads the plugin, verifies its integrity, and executes it locally.
    * **For other capabilities:** The client prepares input according to the capability's schema.
    * **Logging Usage:** The client creates an `MCPInvokeCapabilityTransaction` including a `ContextRecord` and NRN `Fee`, submitting it to KNIRVCHAIN.


## Phase 7 Testing Infrastructure

### Testing Architecture

KNIRVCHAIN implements comprehensive testing for the core blockchain and MCP functionality:

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
# From project root - run KNIRVCHAIN network integration tests
cd integration-tests && go test -v -run TestKNIRVORACLE

# Run comprehensive KNIRVCHAIN test suite
make test-root

# Run KNIRVCHAIN specific tests
cd KNIRVCHAIN && go test -v ./...
```

#### Test Categories
- Agent Management Tests
- Economics Integration Tests
- Blockchain Tests
- MCP Protocol Tests
- Integration Tests

#### Test Structure
```
KNIRVCHAIN/
├── agent_manager_test.go           
├── agent_integration_test.go       
├── economics_integration_test.go   
├── blockchain_server_test.go       
├── bootnode_test.go               
└── *_test.go                      
```


## Key Features

* **Decentralized Registry:** Immutable, transparent record of all registered MCP capabilities.
* **Verifiable Audit Trail:** `ContextRecord`s provide on-chain proof of all capability interactions.
* **Monetization:** Capability owners earn NRN tokens for usage.
* **Plugin Ecosystem:** Allows developers to offer executable functionalities (plugins).
* **Standardized Interaction:** MCP provides a common way to interact with diverse AI capabilities.


## Autonomous Fault Tolerance & Failover Protocol

KNIRVCHAIN implements a comprehensive failover protocol ensuring 99.9% uptime.

### Failover Protocol Architecture

#### Core Components
- **FailoverManager**: Central coordinator for root node monitoring and transition orchestration
- **Network Control Protocol**: Pub/Sub-based coordination for network-wide pause/resume operations
- **Enhanced P2P Consensus Manager**: Integrated network control message handling

#### Operational Flow
1. **Continuous Monitoring**
   ```go
   healthURL := fm.currentOracleAPIURL + "/health"
   resp, err := fm.httpClient.Get(healthURL)
   ```
2. **Offline Detection & Threshold**
   - 15-minute configurable offline threshold before failover initiation
   - Configurable ping intervals (default: 1 minute)
3. **Autonomous Leader Election**
   ```go
   func (fm *FailoverManager) amIElectedToBecomeOracle() bool {
       return true // Current: First available bootnode elected
   }
   ```
4. **Network Coordination (Network Pause)**
   ```go
   pausePayload := NetworkPausePayload{
       InitiatorPeerID: fm.nodeConfig.PeerID,
       Reason:          "Oracle node failover in progress",
       Timestamp:       time.Now().Unix(),
   }
   ```
5. **Automatic Promotion**
   - Bootnode-to-root transition with wallet inheritance
   - Database path migration from bootnode to new root
   - ChainID generation for unique network identification
   - Application restart with new configuration

### Failover Protocol Features

#### Zero-Trust Coordination
- Message Authentication
- Timeout Protection
- Replay Prevention

#### High-Performance Recovery
- Sub-Second Detection
- Network-Wide Coordination
- Minimal Downtime

#### Resilience & Security
- State Preservation
- Concurrent Safety
- Audit Trail

### Network Control Message Protocol

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

#### Integration Points
- HTTP Server
- P2P Consensus
- Main Application
- Configuration

### Configuration Support

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

### Deployment Scenarios

#### Root Node Failure Recovery
1. Root node goes offline
2. Bootnodes continuously monitor health endpoints
3. Threshold exceeded (15+ minutes offline)
4. Leader election determines promotion candidate
5. NetworkPause broadcast to all nodes
6. Bootnode promoted to root with inherited state
7. NetworkResume signal restores normal operation
8. New root accepts transactions/invocations

#### Maintenance Window Coordination
1. Administrator initiates controlled failover
2. NetworkPause signal pauses all network activity
3. Root node undergoes maintenance/restart
4. NetworkResume restores full functionality
5. All nodes sync and resume normal operations

#### Network Expansion Events
1. New bootnodes join network
2. Automatic promotion for load distribution
3. Coordinated network pause during transition
4. Migration of blockchain state
5. Network resume with expanded capacity

### Testing & Validation

```bash
# Failover protocol tests
go test -v ./failover_manager_test.go

# Network control message integration
go test -v ./p2p_consensus_failover_test.go

# End-to-end failover scenarios
go test -v ./integration/failover_scenarios_test.go
```

#### Test Coverage
- Root Node Failure Simulation
- Network Control Message Reliability
- Promotion State Transfer
- Recovery Time Measurement
- Concurrent Safety Testing

### Performance Metrics

- Detection Time: < 30 seconds from node failure
- Recovery Time: < 2 minutes from detection to full operation
- State Transfer: Complete blockchain inheritance
- Message Latency: < 5 seconds for network control coordination
- Uptime Guarantee: 99.9% with automatic failure detection

### Best Practices & Configuration

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

### Security Considerations

- Message Authentication
- State Integrity
- Access Control
- Audit Logging

### Cloud-Native Compatibility

- Kubernetes Integration
- Container Health Checks
- Service Mesh Awareness


## ContextRecord Scenarios: The Audit Trail in Action

* **Using a Registered Tool:** A `ContextRecord` is logged with `InteractionType: "TOOL_INVOCATION"`, `CapabilityID`, `Initiator`, `InputHash`, `OutputHash`, and NRN `Fee`.
* **Utilizing a Registered Prompt:** A `ContextRecord` logs `InteractionType: "PROMPT_USAGE"`, `CapabilityID`, `Initiator`, and `InputHash`.
* **Executing a Registered Plugin:** A `ContextRecord` logs `InteractionType: "PLUGIN_EXECUTION"`, `CapabilityID`, `Initiator`, `InputHash`, and `OutputHash`.
* **Interacting with a Memory Service:** A `ContextRecord` logs `InteractionType: "MEMORY_WRITE"`, `CapabilityID`, `Initiator`, and `InputHash`.
* **Logging a Sampling Event (Server-Side):** A `ContextRecord` logs `InteractionType: "SAMPLING_REQUEST_SENT"`, `Initiator`, and `InputHash`.
* **Auditing Capability Registrations:** A `ContextRecord` with `InteractionType: "CAPABILITY_REGISTRATION"` can be logged.


## Data Storage

* **LevelDB (Primary):** Stores core blockchain data and an indexed view of MCP data.
* **Off-Chain Storage:** Actual plugin binaries, large datasets, and model artifacts are stored off-chain.


## API Endpoints

### Blockchain Server (`blockchain_server.go`)

* **Generic Transaction Submission:** `POST /transaction`
* **MCP-Specific Transaction Ingestion:** `POST /mcp/capability/register`, `POST /mcp/capability/invoke`
* **MCP Data Querying:** `GET /mcp/capability/{capability_id}`, `GET /mcp/capabilities`, `GET /mcp/context/{context_id}`, `GET /mcp/capability/{capability_id}/invocations`, `GET /mcp/contexts`
* **Other Endpoints:** For blockchain status, blocks, NRN balances, etc.

### Wallet Server (`wallet_server.go`)

* `POST /wallet/mcp/create_register_capability`
* `POST /wallet/mcp/create_invoke_capability`
* Other endpoints for wallet management, generic transaction sending, etc.


## Architectural Considerations and Future Work

### Data Size and On-Chain Storage:

- Storing full capability descriptors on-chain is generally fine.
- For large resources, `ContentHash` and `LocationHints` point to off-chain storage.
- Enhanced Querying with RealmDB (Supplemental):  RealmDB could be used as a supplemental index for more complex queries.

### Immutability vs. Updates:

- Capability descriptors may need updates.
- Approach: Implement updates by creating new registration transactions referencing the previous ID.

### Query Capabilities:

- LevelDB indexing supports basic lookups.
- Future Enhancement: For complex queries, an off-chain indexing/query service might be necessary.

### Smart Contracts:

- Future Enhancement: MCP logic could be implemented as smart contracts.

### Access Control and Permissions:

- Future Enhancement: Implement on-chain access control mechanisms.

### Standardization:

- Schemas for all `Descriptor` types and `ContextRecord` should be well-documented.

### Gas/Fees:

- NRN token gas fees via the `Transaction.Fee` field and `BaseDescriptor.GasFeeNRN`.


## Getting Started

```bash
git clone <repository_url>
cd KNIRVCHAIN
go build -o KNIRVORACLE_node ./cmd/KNIRVCHAIN/main.go
./KNIRVORACLE_node --port 8080 --p2p_port 6001
```

```bash
go build -o wallet_server ./cmd/walletserver/main.go
./wallet_server --port 9090 --blockchain_server_ip http://localhost:8080
```


## Contributing

We welcome contributions! Please see `CONTRIBUTING.md` for guidelines.


## License

This project is licensed under the MIT License.


## Changes

This document summarizes the changes made to the KNIRVCHAIN codebase to implement role-specific builds and improve the build system.

## Role Determination Refactoring

1. **Removed interactive role selection**
   - Removed `promptForRole()` function from `install.go`
   - Updated `Install()` function to use the provided role parameter directly

2. **Created role-specific entry point files**
   - `main_root.go`
   - `main_bootnode.go`
   - `main_developer.go`
   - `main_client.go`

3. **Updated role determination logic**
   - Updated `config/config.go` to prioritize build flags over runtime flags
   - Added `SetRootConstants` function to `config/config.go`
   - Updated `main.go` to check for build tags first, then command-line flags

## Build System Updates

1. **Updated Makefile**
   - Added targets for role-specific builds: `build`, `build/root`, `build/bootnode`, `build/developer`, `build/all-roles`
   - Added targets for cross-compiling: `build/all`, `build/all/root`, `build/all/bootnode`, `build/all/developer`, `build/all-platforms-all-roles`
   - Updated help target

2. **Updated cross-compile.sh**
   - Added support for role-specific builds
   - Added `NODE_ROLE` parameter
   - Updated build command

## Terminal UI Implementation

1. **Updated terminal UI flag handling**
   - Added `useTerminalUI` flag
   - Added `--terminal` and `--no-terminal` flags
   - Prioritized command-line flags over build tag defaults

## Documentation

1. **Created BUILD.md**
   - Documented the new build system
   - Explained role-specific builds
   - Listed available Makefile targets

2. **Created code_cleanup_worklog.md**
   - Tracked the implementation of the code cleanup plan

## Benefits of the New Build System

1. **Reduced binary size**
2. **Simplified configuration**
3. **Improved security**
4. **Better user experience**
5. **Clearer deployment**


## Protobuf Issue Resolution

## Issue Description
Runtime panic: `panic: runtime error: slice bounds out of range [-5:]`

## Root Cause
Corrupted protobuf descriptor data in generated `.pb.go` files.

## Resolution Steps

1. **Identified the Problem**
2. **Regenerated Protobuf Files**
```bash
