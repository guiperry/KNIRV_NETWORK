

---

**Source**: KNIRVROOT/docs/protocols/SYNC_Protocol.md

## KNIRVCHAIN P2P Chain Synchronization Protocol v1.0.0

**1. Overview**

This document specifies the protocol used by KNIRVCHAIN nodes to synchronize their blockchain state directly with devs over the libp2p network. This mechanism allows nodes to discover longer valid chains from their devs and update their local state accordingly.

**2. Protocol Details**

*   **Protocol ID:** `/agent/chain-sync/1.0.0`
*   **Transport:** Libp2p Streams (`network.Stream`)
*   **Message Encoding:** JSON (Newline-delimited JSON objects sent over the stream)

**3. Message Definitions**

Nodes exchange JSON-encoded messages over the stream.

**3.1. GetStatusRequest**

*   **Direction:** Initiator -> Responder
*   **Purpose:** Ask the responder for its current chain status (latest block).
*   **Format:**

    ```json
    {
      "getStatus": true
    }
    ```

    *(Note: Could be extended in the future to include the initiator's status to optimize block requests.)*

**3.2. StatusResponse**

*   **Direction:** Responder -> Initiator
*   **Purpose:** Reply to a `GetStatusRequest` with the responder's latest block information.
*   **Format:**

    ```json
    {
      "latest_block_number": 123,
      "latest_block_hash": "0xabcdef123..."
    }
    ```

    *   `latest_block_number` (uint64): The block number of the responder's current chain head.
    *   `latest_block_hash` (string): The hex-encoded hash of the responder's current chain head.

**3.3. GetBlocksRequest**

*   **Direction:** Initiator -> Responder
*   **Purpose:** Request a sequence of blocks from the responder, starting after a specific block number.
*   **Format:**

    ```json
    {
      "getBlocks": {
        "start_after": 100,
        "limit": 50
      }
    }
    ```

    *   `start_after` (uint64): The block number after which the initiator wants blocks. (e.g., if initiator's head is #100, `start_after` would be 100).
    *   `limit` (uint64, optional): Maximum number of blocks to return in the response. Responders SHOULD respect this limit to prevent sending excessively large messages. If omitted or 0, a reasonable default (e.g., 100) should be assumed by the responder.

**3.4. BlocksResponse**

*   **Direction:** Responder -> Initiator
*   **Purpose:** Reply to a `GetBlocksRequest` with the requested blocks.
*   **Format:**

    ```json
    {
      "blocks": [
        { /* Block Object 1 */ },
        { /* Block Object 2 */ },
        ...
      ]
    }
    ```

    *   `blocks` (array): An array containing the requested Block objects (using the same JSON structure as your existing Block type). The array might be empty if the responder doesn't have the requested blocks. Blocks MUST be ordered sequentially by block number.

**3.5. ErrorResponse (Optional but Recommended)**s

*   **Direction:** Responder -> Initiator
*   **Purpose:** Indicate an error occurred while processing a request.
*   **Format:**

    ```json
    {
      "error": {
        "message": "Invalid block range requested"
      }
    }
    ```

    *   `message` (string): A human-readable error message.

**4. Workflow**

1.  **Peer Discovery:** The Initiator node uses the `DiscoveryManager` (`FindResource`) to identify other KNIRVCHAIN devs for its `ChainID`.

2.  **Stream Initiation:** The Initiator selects a dev and attempts to open a new libp2p stream using the `ChainSyncProtocolID`.

3.  **Status Exchange:**
    *   The Initiator sends a `GetStatusRequest` message.
    *   The Responder processes the request and sends back a `StatusResponse` message containing its `latest_block_number` and `latest_block_hash`.

4.  **Chain Comparison:** The Initiator receives the `StatusResponse` and compares the `latest_block_number` with its own current chain head.

5.  **Block Request (If Necessary):**
    *   If the Responder's `latest_block_number` is greater than the Initiator's, the Initiator determines which blocks it needs (typically starting from its current head).
    *   The Initiator sends a `GetBlocksRequest` message, specifying `start_after` as its current block number and potentially a `limit`.

6.  **Block Transfer:**
    *   The Responder receives the `GetBlocksRequest`. It retrieves the requested blocks from its local storage (LevelDB).
    *   The Responder sends a `BlocksResponse` message containing the requested blocks (up to the specified or default limit).

7.  **Validation and Integration:**
    *   The Initiator receives the `BlocksResponse`.
    *   **CRITICAL:** The Initiator MUST meticulously validate the received blocks:
        *   Verify block hashes.
        *   Verify linkage (`PrevHash` matches the previous block's hash).
        *   Verify Proof-of-Work difficulty.
        *   Verify block timestamps are sequential and reasonable.
        *   Verify all transactions within the blocks (including signatures, if applicable, and checking for double-spends against the potential new chain state).
    *   If the received blocks are valid and form a longer chain when appended to the Initiator's current chain (or a valid segment linking to an earlier block), the Initiator calls `switchToChain` to update its local state (memory and LevelDB).

8.  **Repeat (If Necessary):** If the `BlocksResponse` did not contain all the blocks needed to reach the Responder's head (due to the limit), the Initiator may send another `GetBlocksRequest` starting from the new head it just integrated.

9.  **Stream Closure:** Once synchronization is complete (or if an error occurs), both sides should close the stream.

**5. Validation Rules**

**NEVER TRUST DATA FROM PEERS.** All received blocks and transactions MUST be independently validated according to the KNIRVCHAIN consensus rules before being integrated into the local chain.

*   Validate block hashes.
*   Validate `PrevHash` linkage between consecutive blocks.
*   Validate Proof-of-Work difficulty.
*   Validate transaction signatures (for non-reward transactions).
*   Validate transaction contents and check for double-spends within the context of the chain being built.
*   Validate block timestamps.

**6. Error Handling**

*   Implement timeouts for stream opening and read/write operations.
*   Handle invalid JSON messages gracefully (log error, close stream).
*   Handle validation failures (log error, potentially penalize dev score if implemented, close stream).
*   Handle network errors (connection drops, stream resets).

**7. Implementation Notes (`p2p_consensus.go`)**

*   The `handleSyncStream` function will act as the Responder. It needs to decode incoming requests (`GetStatusRequest`, `GetBlocksRequest`) and encode/send appropriate responses (`StatusResponse`, `BlocksResponse`).
*   The `requestChainFromPeers` function will act as the Initiator. It needs to:
    *   Use `DiscoveryManager` to find devs.
    *   Open streams (`host.NewStream`).
    *   Send `GetStatusRequest`.
    *   Receive `StatusResponse`.
    *   Compare chain heights.
    *   Send `GetBlocksRequest` if needed.
    *   Receive `BlocksResponse`.
    *   Call `validateChain` and `switchToChain`.
    *   Use `bufio.Reader` and `bufio.Writer` for efficient stream I/O.
    *   Use `json.Encoder` and `json.Decoder` for message handling.
    *   Protect access to shared blockchain state (`pcm.blockchain`) using `pcm.blockchain.Lock()` and `pcm.blockchain.Unlock()` appropriately.
    *   Implement reasonable limits on the number of blocks requested/sent in a single `BlocksResponse` (e.g., 50-100) to avoid memory issues and overly long blocking operations.

**How Verifier Peers Use This:**

Your verifier devs simply need to run the same KNIRVCHAIN node software as your root node.

*   **Initialization:** When a verifier node starts, `InitP2PConsensusManager` should be called, which creates the `P2PConsensusManager`, sets up the libp2p host via `DiscoveryManager`, joins the PubSub topics, and crucially, registers the `handleSyncStream` handler using `pcm.host.SetStreamHandler(ChainSyncProtocolID, pcm.handleSyncStream)`.
*   **Discovery:** The `DiscoveryManager` (running in the background via `go discoveryMgr.Run(...)`) will connect to bootstrap devs and use the DHT to find other nodes announcing the same `ChainID`.
*   **Synchronization:** The `P2PConsensusManager`'s `runForkResolution` goroutine periodically calls `requestChainFromPeers`. This function will:
    *   Find other KNIRVCHAIN devs via the `DiscoveryManager`.
    *   Attempt to open streams to those devs using `/agent/chain-sync/1.0.0`.
    *   If a dev responds (because it also has the `handleSyncStream` handler registered), the protocol described above is executed.
*   The verifier node acts as both an Initiator (when `requestChainFromPeers` runs) and a Responder (when `handleSyncStream` is invoked by another dev).

So, by implementing the logic described in this specification within `p2p_consensus.go`, all your nodes (root and verifiers) running the same code will automatically participate in this P2P synchronization.

---

<div class="footer-links">


© 2025 KNIRV Network
</div>
