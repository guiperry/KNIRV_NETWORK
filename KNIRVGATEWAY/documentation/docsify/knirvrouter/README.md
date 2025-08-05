

---

**Source**: KNIRVROUTER/p2p/README.md

# P2P Network for KNIRVCHAIN Verifier Nodes

This package implements the peer-to-peer networking layer for KNIRVCHAIN Verifier nodes, enabling them to communicate, share transactions, and maintain consensus on the private chain.

## Core Features

- **Decentralized Hash Table (DHT)**: Uses libp2p's Kademlia DHT for peer discovery and content routing.
- **knirv:// URI Scheme**: Supports the new unified URI scheme for all KNIRVCHAIN resources.
- **Peer Discovery**: Nodes can discover other nodes in the network through DHT, bootstrap nodes, and peer exchange.
- **Message Protocol**: Supports various message types for transaction broadcasting, block propagation, and network maintenance.
- **Connection Management**: Handles establishing connections, maintaining them, and gracefully handling disconnections.
- **Transaction Broadcasting**: Efficiently propagates private transactions across the network.
- **Block Propagation**: Shares newly mined blocks with all peers.
- **P2P Consensus**: Implements a decentralized consensus mechanism using libp2p pubsub.
- **Chain Synchronization**: Provides an efficient protocol for synchronizing blockchain state between peers.
- **Fork Resolution**: Automatically detects and resolves blockchain forks to maintain consensus.

## Message Types

- `TX`: Transaction data
- `BLOCK`: Block data
- `PEERS`: List of known peers
- `GET_BLOCKS`: Request for blocks
- `GET_PEERS`: Request for peer list
- `PING`/`PONG`: Connection health check

## Usage

### Traditional P2P Server

```go
// Create a P2P server
server := p2p.NewServer(
    "127.0.0.1:9000",                // Listen address
    "node1",                         // Node ID
    "chain1",                        // Chain ID
    []string{"127.0.0.1:9001"},      // Bootstrap peers
    handleNewTransaction,            // Transaction callback
    handleNewBlock,                  // Block callback
)

// Start the server
if err := server.Start(); err != nil {
    log.Fatalf("Failed to start P2P server: %v", err)
}

// Broadcast a transaction
server.BroadcastTransaction(transaction)

// Broadcast a block
server.BroadcastBlock(block)

// Stop the server
server.Stop()
```

### P2P Consensus

```go
// Initialize the discovery manager
discoveryManager, err := p2p.NewDiscoveryManager(blockchain.ChainID)
if err != nil {
    log.Fatalf("Failed to initialize discovery manager: %v", err)
}

// Start the discovery manager
go discoveryManager.Run()

// Initialize the P2P consensus manager
db := blockchain.GetLevelDBInstance()
err = p2p.InitializeP2PConsensus(blockchain, db, discoveryManager)
if err != nil {
    log.Printf("Failed to initialize P2P consensus: %v", err)
}

// Broadcast a block
block := blockchain.NewBlock(...)
p2p.BroadcastBlockViaP2P(block)

// Broadcast a transaction
transaction := blockchain.NewTransaction(...)
p2p.BroadcastTransactionViaP2P(transaction)

// Shutdown
p2p.ShutdownP2PConsensus()
discoveryManager.Stop()
```

## Implementation Details

The P2P network uses a hybrid approach:
1. Traditional TCP connections for reliable direct communication between nodes
2. libp2p with Kademlia DHT for decentralized peer discovery and content routing

Messages are JSON-encoded and delimited by newlines for simplicity. Each node maintains connections to multiple peers for redundancy and network resilience.

### knirv:// URI Scheme

The new unified URI scheme follows this format:
```
knirv://<ID>.<ResourceType>/[path][?query][#fragment]
```

Examples:
- `knirv://abc123.chain/` - References a chain with ID "abc123"
- `knirv://xyz789.nrn/asset/123` - References an NRN asset

The DHT is used to resolve these URIs to actual peer addresses, enabling fully decentralized resource discovery.

### Peer Discovery Process

1. On startup, a node connects to configured bootstrap peers for the DHT
2. It announces itself to the DHT with its chain ID
3. It can discover other nodes by querying the DHT for a specific chain ID
4. It also uses traditional peer exchange as a fallback mechanism
5. Local network discovery is supported via mDNS

### Transaction Flow

1. When a node creates or receives a private transaction:
   - It validates the transaction locally
   - Adds it to the transaction pool
   - Broadcasts it to all connected peers

2. When a node receives a transaction from a peer:
   - It validates the transaction
   - If valid and not already in the pool, adds it to the transaction pool
   - Does NOT rebroadcast (to prevent flooding)

### Block Propagation

1. When a node mines a new block:
   - It adds the block to its local chain
   - Broadcasts the block to all connected peers

2. When a node receives a block from a peer:
   - It validates the block
   - If valid and builds on the current chain, adds it to the local chain
   - Removes included transactions from the transaction pool

### Chain Synchronization Protocol

The chain synchronization protocol allows nodes to efficiently synchronize their blockchain state with peers. It follows a request-response pattern with these message types:

- `GetStatusRequest`: Request a peer's current chain status
- `StatusResponse`: Response containing the latest block number and hash
- `GetBlocksRequest`: Request blocks starting after a specific block number
- `BlocksResponse`: Response containing the requested blocks
- `ErrorResponse`: Response indicating an error occurred

The protocol workflow:

1. **Status Exchange**:
   - Node A sends a `GetStatusRequest` to Node B
   - Node B responds with a `StatusResponse` containing its latest block number and hash

2. **Block Request**:
   - If Node B has a longer chain, Node A sends a `GetBlocksRequest` specifying the block number to start after
   - Node B responds with a `BlocksResponse` containing the requested blocks

3. **Validation and Integration**:
   - Node A validates the received blocks (hash, PoW, transactions, etc.)
   - If valid, Node A integrates the blocks into its local chain

4. **Repeat if Necessary**:
   - If Node B's chain is longer than what was received, Node A requests more blocks

### Fork Resolution

The consensus manager periodically checks for and resolves blockchain forks by:

1. Discovering peers via the `DiscoveryManager`
2. Requesting their chain status
3. Comparing with the local chain
4. Requesting blocks if a peer has a longer valid chain
5. Validating the received chain
6. Switching to the longer valid chain if found

---

<div class="footer-links">


© 2025 KNIRV Network
</div>
