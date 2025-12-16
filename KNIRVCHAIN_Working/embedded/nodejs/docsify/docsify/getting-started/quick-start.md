# Quick Start Guide



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
*   `POST /uriGenerator`: Generates a `knirv://` URI, checks DHT, announces on DHT.
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
knirv://<ID>.<ResourceType>/<OptionalSubPath>?param1=value1
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
