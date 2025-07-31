

---

**Source**: KNIRVROOT/docs/protocols/Blockchain_README.md

# KNIRVCHAIN: A Decentralized Model Context Protocol Network

KNIRVCHAIN is a blockchain implementation written in Go. It features dev-to-dev networking using libp2p (including a private DHT and mDNS for discovery), HTTP APIs for interaction, a separate wallet server for key management, proof-of-work mining, and mechanisms for chain synchronization. It utilizes LevelDB for persistent storage and defines a custom `agent://` URI scheme for identifying chain resources within its private network.

## Features

*   **Blockchain Core:** Implements basic block and transaction structures.
*   **Proof-of-Work (PoW):** Simple PoW mining implementation.
*   **Persistence:** Uses LevelDB (`syndtr/goleveldb`) to store blockchain state.
*   **P2P Networking (libp2p):**
    *   Uses libp2p for dev-to-dev communication.
    *   **Private DHT Discovery:** Employs a Kademlia DHT configured with custom bootstrap devs to create an isolated network. Used for discovering devs providing specific chain resources (e.g., `myPeer.chain`) and MCP capabilities.
    *   **mDNS Discovery:** Uses mDNS for discovering devs on the local network.
    *   **NAT Traversal:** Includes AutoRelay, NATPortMap, and Hole Punching support to facilitate connections between nodes behind NATs.
    *   **P2P Chain Synchronization:** Implements a `/agent/chain-sync/1.0.0` protocol for nodes to directly exchange status and blocks to synchronize their chain state.
*   **Consensus (Current):** Implements a basic longest-chain consensus rule by polling configured reflection nodes via HTTP (intended for root node coordination). Peer nodes rely on the P2P sync protocol.
*   **Wallet:** ECDSA key pairs, address generation (using Base58Check encoding for public keys), transaction signing.
*   **URI Scheme:** Defines and utilizes a `agent://<ID>.<ResourceType>/...` scheme for identifying resources on the network. Includes URI minting via transactions and announcement on the private DHT.


### Model Context Protocol (MCP) Layer
*   **MCP:** A set of standards for defining, discovering, and interacting with AI capabilities. KNIRVCHAIN serves as a decentralized backend for MCP.
*   **Capabilities:** These are the core building blocks of the MCP ecosystem on KNIRVCHAIN:
    *   **Resources:** Expose structured or dynamic content. This includes:
        *   **Plugins:** Developer-uploaded executables (e.g., Go `.so`, Wasm) hosted by devs, discovered on-chain, downloaded, and run by clients. (See `docs/plugin_templates.md`)
        *   **Datasets:** References to datasets with on-chain metadata and hashes.
        *   **Model Artifacts:** References to AI model files.
        *   **APIs:** Descriptors for external APIs.
    *   **Tools:** Executable actions defined with JSON schemas for input and output. Can be backed by plugins or external APIs.
    *   **Prompts:** Reusable, parameterized prompt templates for LLMs.
    *   **Memory Services:** Capabilities providing persistent, graph-like memory stores.
*   **Capability Descriptors:** Metadata structures (e.g., `ResourceDescriptor`, `ToolDescriptor`) registered on the blockchain. These descriptors define the capability's properties, schemas, owner, NRN gas fee, and for resources like plugins, a `ContentHash` and `LocationHints` for off-chain retrieval.
*   **ContextRecord:** An on-chain record logged for every significant MCP interaction (e.g., capability invocation, registration). It details what happened, when, by whom, and with what references (hashes, fees), forming an immutable audit trail.
*   **NRN Tokens:** The native utility token of KNIRVCHAIN, used for:
    *   Paying gas fees for registering and invoking MCP capabilities.
    *   Compensating capability owners (e.g., plugin developers).
*   **agent Client (Inference Engine):** An application that interacts with KNIRVCHAIN to discover, download (for plugins), execute, and log the usage of MCP capabilities.

## Features Summary

*   Core Blockchain (PoW, LevelDB Persistence)
*   Advanced P2P Networking with Private DHT & NAT Traversal
*   P2P Chain Synchronization
*   Custom `agent://` URI Scheme with DHT-based Discovery

*   **Startup Modes:**
    *   **Network Mode (`-network`):** Starts two nodes (main and reflection) configured for root network operation.
    *   **Peer Mode (`-dev`):** Starts a single node configured as a dev, using dev-specific ports defined during installation. Requires installation to be complete.
    *   **Single Node Mode (Default if no flags):** Starts a single node, typically using root configuration unless overridden by flags.
    *   **Client-Only Mode (`-client-only`):** Runs a node with reduced resource usage (no DHT server, no mDNS), optimized for client operations.
*   **Installation Process:** Includes an installer flow (`install.go`) triggered in dev mode (if needed) to generate a unique ChainID via the root node, configure dev-specific ports, and save the configuration.
*   **Model Context Protocol (MCP) Integration:**
    *   On-chain registration and discovery of AI Capabilities (Tools, Prompts, Plugins, Resources, Memory).
    *   Immutable `ContextRecord` logging for all MCP interactions, providing a verifiable audit trail.
    *   NRN token-based gas fees for MCP operations.
    *   Support for client-side execution of Plugins.
*   **Payment Processing:** Includes conceptual design (`docs/paymentProcessor.md`) for integrating fiat/crypto payments and disbursing native tokens.

## Architecture Overview

*   **`main.go`:** The main entry point, parses flags, manages configuration loading (searching standard paths), determines run mode, initializes and manages the lifecycle of node components, and handles graceful shutdown.
*   **`BlockchainServer` (`blockchain_server.go`):** Provides the primary HTTP API for interacting with a blockchain node. Handles requests for chain data, transactions, blocks, dev info, URI generation (including DHT checks/announcements), etc.
*   **`WalletServer` (`wallet_server.go`):** A separate HTTP server dedicated to wallet operations (generation, signing). Communicates with a specified `BlockchainServer`.
*   **`DiscoveryManager` (`discovery_manager.go`):** Encapsulates libp2p host setup, connection to the private DHT via custom bootstrap devs, resource announcement (`Provide` for chain URIs and potentially MCP capabilities), dev discovery (`FindProviders`), mDNS, and dev connection logic.
*   **`SelfConsensusManager` (`self_consensus_manager.go`):** Implements the HTTP-based longest-chain polling mechanism used primarily for root/reflection node confirmation & coordination.
*   **`P2PConsensusManager` (`p2p_consensus.go`):** Implements the `/agent/chain-sync/1.0.0` protocol for direct dev-to-dev blockchain synchronization, including status exchange, block requests, and validation.
*   **`BlockchainStruct` (`blockchain_struct.go`):** Represents the blockchain, holding blocks, transaction pool, metadata, and methods for adding blocks/transactions, mining, balance calculation, etc. **It also handles validation and processing of MCP transactions (capability registration, invocation), including NRN fee deductions and state updates for MCP descriptors and context records.**
*   **`Block` (`block.go`) & `Transaction` (`transaction.go`):** Core data structures with hashing and verification methods.
*   **`Wallet` (`wallet.go`):** Handles ECDSA key generation, storage, address derivation, and transaction signing.
*   **MCP Types (`mcp_types.go`):** Defines Go structs for `CapabilityDescriptor` (Base, Resource, Tool, Prompt, MemoryService) and `ContextRecord`.
*   **URI Handling (`uri_generation.go`,`uri_registration.go`, `uri_parsing.go`):** Defines, registers, parses, and generates the `agent://` URI scheme.
*   **Installation (`install.go`):** Handles the interactive process for dev nodes to obtain a ChainID from the root, configure ports, and prepare the configuration file.
*   **Configuration (`config/config.go`):** Defines the configuration structure, loading logic (with search paths), saving, and default values.


## Data Storage

*   **LevelDB (Primary):**
    *   Stores the core blockchain data (blocks, raw transactions).
    *   Stores an indexed view of MCP data extracted from the blockchain:
        *   Serialized `CapabilityDescriptor`s (e.g., `mcp:capability:<id> -> descriptor_json`).
        *   Serialized `ContextRecord`s (e.g., `mcp:context:<tx_id> -> context_record_json`).
        *   NRN token account balances (e.g., `account:balance:<address> -> balance_uint64`).
    *   This indexed view powers the API query endpoints for basic lookups.

*   **Off-Chain Storage:**
    *   Actual plugin binaries, large datasets, and model artifacts are stored off-chain (e.g., hosted by devs, on IPFS, or other web services).
    *   The on-chain `ResourceDescriptor` contains a `ContentHash` for integrity verification and `LocationHints` (URIs) for retrieval by the agent Client.

## `ContextRecord` Scenarios: The Audit Trail

The `ContextRecord` is central to KNIRVCHAIN's MCP, providing an on-chain, verifiable log for:
*   **Tool Invocations:** Logging `InteractionType: "TOOL_INVOCATION"`, `CapabilityID`, `Initiator`, input/output hashes, and NRN fee.
*   **Prompt Usage:** Logging `InteractionType: "PROMPT_USAGE"`, `CapabilityID`, `Initiator`, and input hash (of prompt parameters).
*   **Plugin Executions:** Logging `InteractionType: "PLUGIN_EXECUTION"`, `CapabilityID`, `Initiator`, input/output hashes of plugin data.
*   **Memory Service Interactions:** Logging `InteractionType: "MEMORY_WRITE"` or `"MEMORY_READ"`.
*   **Sampling Events:** Logging `InteractionType: "SAMPLING_REQUEST_SENT"` or `"SAMPLING_RESPONSE_RECEIVED"`.
*   **Capability Registrations:** Optionally logging `InteractionType: "CAPABILITY_REGISTRATION"` for a unified event feed.
These records are queried via endpoints like `GET /mcp/context/{id}` and `GET /mcp/capability/{id}/invocations`.

## Getting Started

### Prerequisites

*   Go (version 1.21 or later recommended)
*   Standard build tools (`git`, etc.)
*   **(For Private DHT)**: Access to 2-3 stable machines with public IPs to run dedicated bootstrap nodes.

### Building

```bash
# Navigate to the project root directory
cd /path/to/KNIRVCHAIN_GO_ROOT

# Build the executable
go build -o KNIRVCHAIN_node .
```

#### Configuration

The application searches for `config.json` in the following order:

1.  Path specified by the `-config` flag.
2.  Path specified by the `KNIRVCHAIN_CONFIG_PATH` environment variable.
3.  User config directory (`~/.config/KNIRVCHAIN/config.json` on Linux).
4.  Directory containing the executable.
5.  Current working directory.

If not found, a default `config.json` is created (usually in the user config directory or CWD).

#### Private DHT Setup:

1.  Run the `KNIRVCHAIN_node` on your chosen bootstrap machines. Note their full multiaddresses (e.g., `/ip4/x.x.x.x/tcp/4001/p2p/12D3Koo...`).
2.  Update the `bootstrap_devs` list in your `config.json` (or the hardcoded list in `discovery_manager.go`) on *all* nodes (root, devs, bootstrap nodes themselves) to contain *only* the multiaddresses of your dedicated bootstrap nodes.

### Running

1.  **Root Network Mode:** Starts the main and reflection nodes for the root network. Uses ports defined in `config.json` (e.g., 5000/4001 and 5001/4002). Requires `bootstrap_devs` to be set for the private DHT.

    ```bash
    # Ensure config.json has correct root ports and private bootstrap_devs
    ./KNIRVCHAIN_node -network
    # Or using go run
    # go run . -network
    ```

2.  **Peer Node Mode:** Starts a dev node. Requires prior installation. Uses dev-specific ports defined in `config.json`. Requires `bootstrap_devs` to be set.

    ```bash
    # First-time run (triggers installer if install_complete=false):
    ./KNIRVCHAIN_node -dev

    # Subsequent runs (loads config with dev ports):
    ./KNIRVCHAIN_node -dev
    ```

3.  **Single Node Mode:** Starts a single node, typically using root configuration unless overridden. Useful for development or specific setups.

    ```bash
    # Example: Start a single node on different ports
    ./KNIRVCHAIN_node -port 5005 -p2p.port 4005 -wallet_port 6005 -shared_database_path database_node5005/agent.db -miners_address your_miner_address
    ```

Press `Ctrl+C` to initiate graceful shutdown.

### Key Command-Line Flags

*   `-network`: (boolean) Run in multi-node network mode (root + reflection).
*   `-dev`: (boolean) Run as a dev node (requires installation complete, uses dev ports from config).
*   `-port`: (uint) HTTP port for the blockchain node API (overrides config for non-dev modes).
*   `-p2p.port`: (uint) Port for the libp2p P2P host (overrides config for non-dev modes).
*   `-wallet_port`: (uint) Preferred HTTP port for the wallet server (overrides config for non-dev modes).
*   `-shared_database_path`: (string) Path for the LevelDB database files (overrides config).
*   `-miners_address`: (string) Address to receive mining rewards (overrides config).
*   `-no-wallet-server`: (boolean) Disable the startup of the node's associated wallet server.
*   `-client-only`: (boolean) Run as a client-only node (no DHT server, no mDNS).
*   `-config`: (string) Explicitly specify the path to the `config.json` file, overriding the default search paths.
*   `-gui`: (boolean) Enable the Fyne graphical user interface (if compiled with GUI support).

### API Endpoints

(Refer to `docs/URI_Generation_GO.md` for detailed path/query usage)

#### Blockchain Node API (Ports vary by mode/config)

*   `GET /chain`: Returns current blockchain state.
*   `POST /block`: Submit a new block.
*   `POST /transaction`: Submit a new transaction.
*   `GET /txn_pool`: Returns the transaction pool.
*   `GET /ping`: Simple health check.
*   `GET /health`: Detailed health check.
*   `POST /uriGenerator`: Generates a `agent://` URI, checks DHT, announces on DHT.
*   `GET /info`: Returns server information.
*   `GET /devs`: Finds devs for the node's chain via the private DHT.
*   `POST /test/faucet`: (Test Only) Funds an address.
*   **MCP Endpoints:**
    *   `POST /mcp/capability/register`: Submits a pre-signed/component-provided `MCPRegisterCapabilityTransaction`.
    *   `POST /mcp/capability/invoke`: Submits a pre-signed/component-provided `MCPInvokeCapabilityTransaction`.
    *   `GET /mcp/capability/{capability_id}`: Retrieves a capability descriptor.
    *   `GET /mcp/capabilities?...`: Lists capabilities with filters.
    *   `GET /mcp/context/{context_id}`: Retrieves a specific context record.
    *   `GET /mcp/capability/{capability_id}/invocations`: Retrieves all context records for a capability.
    *   `GET /mcp/contexts?...`: Advanced querying for context records.

#### Wallet Server API (Ports vary by mode/config)

*   `GET /generate_wallet`: Generates a new wallet.
*   `POST /send_signed_txn`: Signs and forwards a transaction.
*   `GET /ping`: Simple health check.
*   **MCP Endpoints:**
    *   `POST /wallet/mcp/create_register_capability`: Creates, signs, and submits an `MCPRegisterCapabilityTransaction`.
    *   `POST /wallet/mcp/create_invoke_capability`: Creates, signs, and submits an `MCPInvokeCapabilityTransaction`.

### URI Scheme

KNIRVCHAIN uses a custom URI scheme for resource identification within its private network:

```
agent://<ID>.<ResourceType>/<OptionalSubPath>?param1=value1
```

*   `<ID>`: Unique identifier (e.g., `agent-root-5000`, `myPeer`).
*   `<ResourceType>`: `chain` or `nrn`.
*   `<OptionalSubPath>`: e.g., `/block`, `/transaction`, `/devs`.
*   `?params`: Optional query parameters.

Resolution involves parsing the URI, deriving a resource key (e.g., `myPeer.chain`), generating a CID, querying the private KNIRVCHAIN DHT for providers using the CID, connecting to a provider via libp2p, and making the request over a P2P stream.

### Testing

Run unit and integration tests using the standard Go tool:

```bash
go test ./... -v
```

*   `transaction_test.go`: Single node transaction tests.
*   `network_transaction_test.go`: Multi-node transaction tests.
*   `uri_generator_test.go`: Tests `/uriGenerator` endpoint and DHT interaction.
*   `dev_lifecycle_test.go`: Integration test for dev installation, startup, and DHT announcement.

### Troubleshooting

*   **UDP Buffer Warning:** `failed to sufficiently increase receive buffer size...` - See [QUIC-Go UDP Buffer Sizes Wiki](https://github.com/quic-go/quic-go/wiki/UDP-Buffer-Sizes) for OS tuning instructions (e.g., `sysctl -w net.core.rmem_max=2500000`).
*   **Port Conflicts:** Ensure required ports (HTTP, P2P) are free or configure nodes to use different ports via config/flags. Check dev ports vs. root ports.
*   **DHT Bootstrap Failures:** Verify `bootstrap_devs` in `config.json` are correct and reachable. Check firewalls on bootstrap nodes.
*   **Peer Not Found:** Ensure the dev node started correctly on its configured P2P port, connected to the private DHT, and announced its resource (`dht.Provide`). Check logs on both the searching node and the dev node.

### Future Work / Improvements

*   **Consensus:** Replace the simple HTTP polling/longest-chain rule with a more robust P2P consensus algorithm (e.g., Raft, PBFT, PoS).
*   **P2P Sync Robustness:** Enhance the P2P sync protocol (error handling, block limits, streaming for large chains).
*   **Configuration:** Refine configuration management, potentially prioritizing env vars over file for sensitive data.
*   **State Management:** Optimize blockchain state saving/loading.
*   **State Management:** Optimize blockchain state saving/loading. +* State Management & Querying:
*   *Optimize blockchain state saving/loading.
*   *Enhanced MCP Querying with RealmDB (Supplemental): For more complex queries over MCP data (e.g., "find all tools with input schema X and owned by Y"), consider using RealmDB as a supplemental, rich secondary index. LevelDB would remain the source of truth for the blockchain, while RealmDB would provide a highly queryable, object-oriented cache of MCP descriptors and context records. This would require careful synchronization logic, especially during chain reorganizations.
*   **Transaction Verification:** Implement more robust transaction validation (nonces, replay protection).
*   **Smart Contracts:** Develop the placeholder SmartContract functionality.
*   **Security:** Conduct thorough security audits (key handling, network, input validation).
*   **GUI:** Continue refining the Fyne GUI, ensuring thread safety for updates.
*   **Dependencies:** Keep libp2p and other dependencies updated.
*   **MCP Access Control:** Implement on-chain access control mechanisms for MCP capabilities, possibly tied to ownership fields or smart contracts. 
*   **MCP Standardization:** Further formalize schemas for Descriptor types and ContextRecord details.


### License

(GNU License)


---

<div class="footer-links">


© 2025 KNIRV Network
</div>
