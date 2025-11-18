# KNIRVORACLE: A Decentralized Model Context Protocol Network

## Table of Contents

- [Overview](#overview)
- [Core Concepts](#core-concepts)
- [Architecture Overview](#architecture-overview)
- [Phase 7 Testing Infrastructure](#phase-7-testing-infrastructure)
- [Key Features](#key-features)
- [`ContextRecord` Scenarios](#contextrecord-scenarios)
- [Data Storage](#data-storage)
- [API Endpoints](#api-endpoints)
- [Architectural Considerations and Future Work](#architectural-considerations-and-future-work)
- [Getting Started](#getting-started)
- [Contributing](#contributing)
- [License](#license)
- [Changes](#changes)
- [Makefile Embedded Services Update](#makefile-embedded-services-update)
- [Protobuf Issue Resolution](#protobuf-issue-resolution)
- [Root Payment Processor Implementation](#root-payment-processor-implementation)
- [agent Tunnel Relay Implementation Plan](#agent-tunnel-relay-implementation-plan)
- [WebGUI Fix Summary](#webgui-fix-summary)
- [KNIRV Network Final Test Fixes Report](#knirv-network-final-test-fixes-report)


## Overview

KNIRVORACLE is a blockchain-based platform designed to facilitate a transparent, verifiable, and monetizable ecosystem for AI capabilities. It implements the Model Context Protocol (MCP), allowing various AI-related functionalities (Tools, Prompts, Plugins, Memory Services) to be registered, discovered, invoked, and audited on-chain. Each interaction with an MCP capability results in a blockchain transaction, creating an immutable record and potentially involving NRN token fees.


## Core Concepts

* **Model Context Protocol (MCP):** A set of standards for defining, discovering, and interacting with AI capabilities. KNIRVORACLE serves as a decentralized backend for MCP.
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
* **NRN Tokens:** The native utility token of KNIRVORACLE, used for paying gas fees for registering and invoking MCP capabilities and compensating capability owners.
* **KNIRVORACLE Node:** The core blockchain dev software that maintains the ledger, processes MCP transactions, and exposes an API.
* **Wallet Server:** A service that helps users create, sign, and submit MCP transactions to the KNIRVORACLE network.
* **agent Client (Inference Engine):** An application that interacts with KNIRVORACLE to discover, download (for plugins), execute, and log the usage of MCP capabilities.


## Architecture Overview

1. **Capability Registration:** Developers register capabilities by submitting an `MCPRegisterCapabilityTransaction` containing the serialized `CapabilityDescriptor`. The descriptor is stored on-chain.

2. **Capability Discovery:** agent Clients query the KNIRVORACLE API to find available capabilities.

3. **Capability Invocation & Plugin Execution:**  Clients download (for plugins), execute, and log usage.  Logging involves creating an `MCPInvokeCapabilityTransaction` with a `ContextRecord`.


## Phase 7 Testing Infrastructure

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


## Key Features

* **Decentralized Registry:** Immutable, transparent record of all registered MCP capabilities.
* **Verifiable Audit Trail:** `ContextRecord`s provide on-chain proof of all capability interactions.
* **Monetization:** Capability owners earn NRN tokens for usage.
* **Plugin Ecosystem:** Allows developers to offer executable functionalities (plugins).
* **Standardized Interaction:** MCP provides a common way to interact with diverse AI capabilities.


## `ContextRecord` Scenarios

* **Using a Registered Tool:** A `ContextRecord` is logged with details of the tool invocation.
* **Utilizing a Registered Prompt:** A `ContextRecord` logs details of prompt usage.
* **Executing a Registered Plugin:** A `ContextRecord` logs details of plugin execution.
* **Interacting with a Memory Service:** A `ContextRecord` logs details of memory interactions.
* **Logging a Sampling Event (Server-Side):** A `ContextRecord` logs server-initiated LLM tasks.
* **Auditing Capability Registrations:**  A `ContextRecord` can log capability registrations.


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
* Other endpoints for wallet management, etc.


## Architectural Considerations and Future Work

### Data Size and On-Chain Storage

Storing full capability descriptors on-chain is generally fine.  For large resources, `ContentHash` and `LocationHints` point to off-chain storage.  Enhanced querying with RealmDB is a future consideration.

### Immutability vs. Updates

Capability descriptors may need updates.  Updates are implemented by creating new registration transactions.

### Query Capabilities

The current LevelDB indexing supports basic lookups.  More complex queries might require an off-chain indexing service.

### Smart Contracts

Future implementation of MCP logic as smart contracts is planned.

### Access Control and Permissions

Initially, any valid transaction sender can register/invoke capabilities.  Future enhancement of on-chain access control is planned.

### Standardization

Schemas for all `Descriptor` types and `ContextRecord` should be well-documented.

### Gas/Fees

NRN token gas fees are incorporated via the `Transaction.Fee` field and `BaseDescriptor.GasFeeNRN`.


## Getting Started

```bash
git clone <repository_url>
cd KNIRVORACLE
go build -o KNIRVORACLE_node ./cmd/KNIRVORACLE/main.go
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

This document summarizes changes made to implement role-specific builds and improve the build system.  Interactive role selection was removed, role-specific entry points were created (`main_root.go`, `main_bootnode.go`, `main_developer.go`, `main_client.go`), and the role determination logic was updated.  The Makefile was updated with targets for role-specific builds and cross-compiling.  Terminal UI implementation was updated, and `BUILD.md` and `code_cleanup_worklog.md` were created.


## Makefile Embedded Services Update

The KNIRVORACLE Makefile was updated to manage embedded Node.js and binary services in the `embedded/` directory. New targets were added for building, cleaning, and managing these services.  The update handles both Go binaries and Node.js services, includes automatic dependency installation, and provides error handling.


## Protobuf Issue Resolution

A runtime panic due to slice bounds out of range was resolved by regenerating protobuf files.  An unused import was also removed.


## Root Payment Processor Implementation

This document outlines how to programmatically accept fiat/crypto payments and disburse KNIRVCHAIN tokens.  It involves a frontend, payment gateway, backend service, KNIRVCHAIN node, and disbursement wallet.  A conceptual Go implementation of the backend service is provided, highlighting secure key management, exchange rate calculation, transaction creation/signing/broadcasting, and webhook handling (with Stripe as an example).


## agent Tunnel Relay Implementation Plan

This document outlines the implementation plan for a decentralized tunnel relay system. It analyzes the existing components (Go and Node.js), identifies missing components, and describes the architecture and key processes (URI resolution, generation, and node registration).  It details implementation steps and benefits of the approach.


## WebGUI Fix Summary

The KNIRVORACLE WebGUI (Next.js application) was failing to start due to a missing CSS import and an incompatible start command.  The CSS import was fixed, the start command was updated to use `npx serve`, and the Makefile build target was enhanced.


## KNIRV Network Final Test Fixes Report

All critical test failures have been resolved across the KNIRV ecosystem.  Specific fixes for badge attachment retrieval, tunnel registry URI resolution, Python SDK module installation, KNIRVCORTEX mock implementation, and KNIRVGATEWAY build dependencies are detailed.  Enhanced test scripts and Makefile integration are also described.  Performance metrics and next steps are outlined.
