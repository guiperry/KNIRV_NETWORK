

---

**Source**: KNIRVROOT/docs/protocols/DHT_Protocol.md

# KNIRVCHAIN Decentralized Discovery Implementation Plan (DHT)

## 1. Goal

To replace the current centralized Node.js registry and STUN service with a decentralized dev discovery mechanism using a Distributed Hash Table (DHT), enhancing network resilience and removing reliance on a central authority (`agent.com`) for lookups.

## 2. Core Technology

*   **Language:** Go
*   **Library:** `go-libp2p` (specifically `go-libp2p-kad-dht` and related core modules)

## 3. Key Components & Concepts

*   **DHT Integration:** Each participating KNIRVCHAIN node will run a Kademlia DHT client/server instance.
*   **Peer Identity:** Nodes will use their `libp2p` PeerID (derived from cryptographic keys) as their fundamental network identifier.
*   **Content Routing:** The DHT will be used to map a discoverable identifier (e.g., the `chainID` or the PeerID) to the node's current reachable network address (multiaddr).
*   **Bootstrap Nodes:** A small set of stable, publicly reachable nodes will serve as initial entry points for new nodes joining the DHT network.
*   **Multiaddresses:** Standard `libp2p` multiaddrs (e.g., `/ip4/1.2.3.4/tcp/5001`) will represent node connection endpoints.
*   **Chain Identifier:** For simplicity, we'll assume the `chainID` serves as both the primary identifier for a KNIRVCHAIN instance and the key under which it registers its multiaddress(es) in the DHT.
*   **Node Discovery:** Nodes will periodically refresh their DHT records and query the DHT to discover other nodes.
*   **Connection Establishment:** After discovering another node's multiaddress through the DHT, establish a direct P2P connection using `libp2p`'s built-in networking capabilities.
*   **NAT Traversal:** Utilize `libp2p`'s AutoNAT service to assist nodes behind NATs in establishing connections.
*   **Data Storage:** While primarily designed for dev discovery, consider how the DHT could also store additional metadata about chains/nodes (e.g., version info, health status) if needed.
*   **Security Considerations:** Ensure proper security measures like encryption and authentication mechanisms provided by `libp2p` are utilized where applicable.
*   **Error Handling:** Properly handle errors during DHT queries and ensure retries/recovery strategies are implemented where appropriate.
*   **Logging & Monitoring:** Implement logging and monitoring solutions to track DHT activity, potential issues, and performance metrics.

**URI Scheme**:

The URI scheme for KNIRVCHAIN will be updated to reflect the decentralized nature of the network and provide a clear structure resembling familiar domain patterns. The general format will be:

```plaintext
agent://<ID>.<ResourceType>/<OptionalSubPath>?param1=value1&param2=value2
```

Where:

*   **`agent://`**: Protocol prefix indicating a KNIRVCHAIN-specific URI.
*   **`<ID>.<ResourceType>`**: The authority/host part, combining the unique identifier and the resource type.
    *   **`<ID>`**: Unique identifier for the chain, content, or node (e.g., `chainID`, `contentID`).
    *   **`<ResourceType>`**: Specifies the type of resource, acting like an internal TLD.
        *   `.chain`: Access blockchain-related resources.
        *   `.nrn`: Access NRN tokenized content-related resources.
*   **`/<OptionalSubPath>`**: Specifies sub-resources or actions related to the primary resource type. Examples: `/block`, `/transaction`, `/account`, `/status`, `/devs`.  Can be omitted if not needed (defaults to `/`).
*   **`?param1=value1¶m2=value2`**: Standard query parameters to filter or specify details.

**Common Parameters:**

*   `hash`
*   `number`
*   `address`
*   `version`

**Examples:**

*   General info about a specific chain: `agent://<chainID>.chain/`
*   Get a specific block by number: `agent://<chainID>.chain/block?number=<BLOCK_NUMBER>`
*   Get a specific block by hash: `agent://<chainID>.chain/block?hash=<BLOCK_HASH>`
*   Get a specific transaction: `agent://<chainID>.chain/transaction?hash=<TX_HASH>`
*   Get account info on a chain: `agent://<chainID>.chain/account?address=<ADDRESS>`
*   Get specific NRN content: `agent://<contentID>.nrn/` or 
`agent://<contentID>.nrn/content?version=1.2`
*   Get chain status: `agent://<chainID>.chain/status`
*   Get chain devs: `agent://<chainID>.chain/devs`
*   Get specific NRN content: `agent://<contentID>.nrn/content?version=1.2`
*   Get devs for a specific NRN content: `agent://<contentID>.nrn/devs`
*   Get devs for a specific chain network: `agent://<chainID>.chain/devs`

**Note:** The exact structure and usage of the URI scheme may evolve. Flexibility is key, but this structure provides a more standard and extensible foundation compared to previous iterations.

### 2. Implementation Steps

**Dependency Integration:**

1.  Add necessary go-libp2p modules (libp2p, go-libp2p-kad-dht, go-libp2p-core, go-multiaddr, etc.) to the `go.mod` file.
2.  Run `go mod tidy`.

**Node Identity & libp2p Host:**

1.  Ensure each KNIRVCHAIN node generates or loads a persistent cryptographic key pair (e.g., Ed25519) on startup.
2.  Derive the libp2p PeerID from the public key.
3.  Initialize a libp2p Host instance (`libp2p.New`) configured with the node's identity and listening multiaddrs (e.g., `/ip4/0.0.0.0/tcp/YOUR_P2P_PORT`, `/ip6/::/tcp/YOUR_P2P_PORT`).

**DHT Initialization:**

1.  Create a new Kademlia DHT instance using `dht.New`.
2.  Configure the DHT to run in server mode (`dht.ModeServer`) for nodes that will actively participate in routing and storing data.  Client-only mode (`dht.ModeClient`) might be suitable for some nodes if desired.
3.  Pass the initialized libp2p Host to the DHT constructor.

**Bootstrap Process:**

1.  Define a list of bootstrap node multiaddresses (these will be the stable nodes, potentially including the initial "root" node). This list can be hardcoded initially or loaded from configuration.
2.  Upon DHT initialization, trigger the `dht.Bootstrap` process to connect to the specified bootstrap devs and join the network. Implement retry logic for robustness.

**Announcing Reachable Addresses (Publishing):**

1.  **Self-Address Discovery:** Nodes behind NAT need to determine their public IP.
    *   **Option A (STUN):** Integrate a Go STUN client to query an external STUN server (could still be the one previously running alongside the registry, or public ones).
    *   **Option B (libp2p AutoNAT):** Leverage libp2p's AutoNAT service, which uses other devs to determine reachability.
2.  **Construct Multiaddr:** Create the node's public multiaddr(s) using the discovered public IP and listening port.
3.  **DHT Provide:** Use the DHT's `Provide` mechanism. This typically involves announcing that this node can provide content associated with a specific Content ID (CID). You might hash the `<ID>.<ResourceType>` combination or just the `<ID>` to create a CID, then announce that your PeerID provides this CID. This allows others to find your PeerID when searching for the resource. Alternatively, explore storing the multiaddress directly associated with a key derived from `<ID>.<ResourceType>` if the DHT implementation supports `PutValue/GetValue` suitably for this use case.

**Peer Discovery (Lookup):**

1.  **Input:** Target URI (e.g., `agent://<chainID>.chain/block?hash=...`).
2.  **Parse URI:** Extract `<ID>`, `<ResourceType>`, `<OptionalSubPath>`, and query parameters.
3.  **DHT Query:**
    *   **Option A (FindProviders via CID):** Generate the relevant CID (e.g., from `<ID>` or `<ID>.<ResourceType>`). Use `dht.FindProviders` to find PeerIDs associated with that CID.
    *   **Option B (GetValue):** If using direct key-value storage, use `dht.GetValue` with a key derived from `<ID>.<ResourceType>`.
4.  **Get Peer Info:** Once a PeerID is found (via Option A), use `dht.FindPeer` with the PeerID to get its known multiaddresses.
5.  **Handle Results:** Process the returned addresses, handle timeouts, and manage "not found" scenarios.

**Establishing Connection:**

1.  Once a target dev's multiaddr is obtained from the DHT lookup, use the libp2p Host's `host.Connect(ctx, devInfo)` method (where `devInfo` contains the PeerID and multiaddrs) to establish a direct P2P connection.
2.  Once connected, use the `<OptionalSubPath>` and query parameters from the original URI to make the specific request over the established P2P stream (using your application-level protocol).

**URI Scheme Adaptation:**

1.  Modify all code that generates, parses, or uses the `agent://` URI to conform to the new structure (`agent://<ID>.<ResourceType>/<OptionalSubPath>?...`).
2.  Update internal logic to trigger DHT lookups based on the authority part (`<ID>.<ResourceType>`) of this URI instead of DNS/HTTP lookups.

**Bootstrap Node Setup:**

1.  Designate 1-3 initial nodes as bootstrap nodes.
2.  Ensure these nodes have static IPs or resolvable DNS names (a minimal DNS requirement might remain just for bootstrapping).
3.  Configure them to run the DHT in server mode and listen on publicly accessible ports.
4.  Generate their multiaddresses and embed them in the configuration of regular nodes.

**Testing Strategy:**

1.  **Local Network:** Test DHT bootstrapping, announcing, and discovery between multiple nodes running locally.
2.  **Simulated NAT:** Use tools (like pumba or Docker network configurations) to simulate NAT environments and test STUN/AutoNAT integration.
3.  **Wider Network:** Deploy nodes on different cloud servers/networks to test real-world discovery and connectivity.
4.  **Churn Testing:** Simulate nodes joining and leaving the network frequently to test DHT stability and information propagation.

**Documentation Update:**

1.  Revise all developer and user documentation to explain the new decentralized discovery mechanism.
2.  Document the updated URI scheme clearly.
3.  Provide instructions on configuring bootstrap nodes.

**Deprecation Plan:**

Once the DHT mechanism is stable and deployed, formally deprecate and eventually decommission the Node.js registry/STUN service. Ensure all nodes are updated to use the new system.

### 3. Rollout Strategy (Example)

*   **Phase 1 (Development & Local Testing):** Implement core DHT logic, test extensively on local machines.
*   **Phase 2 (Internal Testnet):** Deploy bootstrap nodes and a small number of test nodes on cloud infrastructure. Test discovery and basic chain operations.
*   **Phase 3 (Feature Flag/Beta):** Release updated node software with the DHT logic disabled by default but enable-able via a flag. Encourage beta testers. Monitor stability.
*   **Phase 4 (Full Rollout):** Enable DHT logic by default in new releases. Announce deprecation timeline for the old registry service.
*   **Phase 5 (Decommission):** Shut down the old registry service after the transition period.

---

<div class="footer-links">


© 2025 KNIRV Network
</div>
