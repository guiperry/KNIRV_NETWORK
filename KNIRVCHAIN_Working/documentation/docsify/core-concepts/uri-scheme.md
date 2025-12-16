# URI Scheme



## 1. Introduction

This document outlines the design, structure, resolution process, and implementation considerations for the `agent://` Uniform Resource Identifier (URI) scheme used within the KNIRVCHAIN network. This scheme provides a standardized way to locate and interact with various resources, such as specific blockchain instances or tokenized content, in a decentralized manner.

## 2. URI Structure

The standard format for a KNIRVCHAIN URI is defined as follows, adhering to common URI patterns:

```plaintext
agent://<ID>.<ResourceType>/<OptionalSubPath>?param1=value1&param2=value2
```
**2.1. Components**

*   **Scheme:** `agent`
    *   **Description:** The mandatory protocol identifier, indicating that the URI pertains to the KNIRVCHAIN network and its resolution mechanisms.

*   **Authority:** `<ID>.<ResourceType>`
    *   **Description:** This part functions similarly to the host/authority in standard URLs. It uniquely identifies the primary resource within the KNIRVCHAIN ecosystem and specifies its general category. This component is critical for discovery via the Distributed Hash Table (DHT).
        *   `<ID>`: A unique identifier string assigned to a specific blockchain instance, content item, node, or other network entity. Its format should be compatible with DHT key requirements (e.g., UTF-8 string, potentially case-sensitive depending on DHT implementation).
        *   `<ResourceType>`: Acts as a namespace or category indicator, similar to a top-level domain (TLD). It helps clients determine the expected nature and interaction methods for the resource. See Section 3 for defined types.

*   **Path:** `/<OptionalSubPath>`
    *   **Description:** An optional hierarchical path component specifying a specific action, sub-resource, or endpoint related to the resource identified by the Authority.
    *   **Default:** If omitted, the path defaults to `/`, typically representing the root or general information endpoint for the resource.
    *   **Examples:** `/block`, `/transaction`, `/devs`, `/content`. See Section 4.

*   **Query:** `?param1=value1&param2=value2`
    *   **Description:** Standard URL query parameters used to provide additional details, filters, or arguments for the request targeting the resource specified by the Authority and Path.
    *   **Format:** Key-value pairs separated by `&`. Values should be URL-encoded if they contain special characters. See Section 5.

**3. Resource Types (`<ResourceType>`)**

The `<ResourceType>` defines the category of the resource being addressed.

*   `.chain`
    *   **Description:** Represents a specific KNIRVCHAIN blockchain instance.
    *   **Purpose:** Used for accessing blockchain-specific data and functionalities like blocks, transactions, account states, consensus status, and dev lists for that particular chain network.

*   `.nrn`
    *   **Description:** Represents tokenized content or data assets managed within the KNIRVCHAIN ecosystem (assuming NRN is the token/content system).
    *   **Purpose:** Used for accessing content metadata, the content data itself (potentially indirectly), version information, provenance, and dev lists related to hosting or accessing that specific content.

*   **Future Types:** The scheme is extensible. Future types like `.node` (for specific node information/management) or `.service` (for decentralized services running on the network) could be defined.

**4. Common Sub-Paths (`/<OptionalSubPath>`)**

The interpretation of the path depends on the `<ResourceType>`.

**4.1. For `.chain` Resources:**

| Path                   | Description                                                                  | Example Query Parameters                               |
| :--------------------- | :--------------------------------------------------------------------------- | :----------------------------------------------------- |
| `/` (Default)          | General chain information (e.g., genesis, height, ID, status summary).       |                                                        |
| `/block`               | Access specific block data.                                                  | `?hash=<hash>`, `?number=<num>`                      |
| `/transaction`         | Access specific transaction data.                                              | `?hash=<hash>`                                        |
| `/account`             | Access account state (balance, nonce, etc.).                                | `?address=<addr>`                                     |
| `/status`              | Detailed operational status of the node providing the resource.               |                                                        |
| `/devs`               | List of known P2P devs participating in this specific chain network.       | `?limit=<num>`                                        |
| `/mempool` or `/txpool` | Current pool of pending transactions for this chain.                         | `?limit=<num>`                                        |
| `/submit/transaction`   | Endpoint for submitting a new transaction (data typically in request body). |                                                        |
| `/submit/block`         | Endpoint for submitting a new block (data typically in request body).       |                                                        |
| `/uri/check`           | Check availability of a potential new ID on the DHT (used internally).     | `?id=<desired_id>&type=<type>`                         |
| `/uri/mint`            | Endpoint for initiating the URI minting transaction (internal/specific use). |                                                        |

**4.2. For `.nrn` Resources:**

| Path         | Description                                                                 | Example Query Parameters                      |
| :----------- | :-------------------------------------------------------------------------- | :-------------------------------------------- |
| `/` (Default) | General metadata about the NRN content item.                              |                                               |
| `/content`   | Access the actual content data (might redirect or provide fetch info).    | `?version=<ver>`, `?format=<fmt>`           |
| `/metadata`  | Explicitly request only the metadata associated with the content ID.       | `?version=<ver>`                              |
| `/devs`     | List of devs known to host or be interested in this specific content.   | `?limit=<num>`                              |
| `/versions`  | List available versions of the content.                                     |                                               |
| `/provenance` | History or ownership trail of the content.                                  |                                               |

**5. Common Query Parameters (`?param=value`)**

These parameters refine requests made via the URI.

*   `hash=<hash_string>`: Specifies a block or transaction hash (typically hex-encoded).
*   `number=<integer>`: Specifies a block number/height.
*   `address=<agent_address>`: Specifies a KNIRVCHAIN wallet or contract address (Base58Check encoded).
*   `version=<version_string>`: Specifies a version for content or APIs (e.g., "1.2", "latest").
*   `limit=<integer>`: Limits the number of items returned in list-based responses (e.g., `/devs`, `/mempool`).
*   `offset=<integer>` or `page=<integer>`: Used for paginating through large lists.
*   `format=<string>`: Specifies a desired format for content retrieval (e.g., "json", "binary").



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

**6. URI Resolution Process**

Resolving a `agent://` URI to interact with the target resource typically involves these steps:

1.  **Parse URI:** A client application (e.g., wallet, browser extension, another node) parses the URI string into its components (Scheme, ID, ResourceType, Path, Query) using standard URL parsing libraries, supplemented with custom logic to split the Authority part.

2.  **Construct DHT Key/CID:** The core identifier for DHT lookup is formed by combining the `<ID>` and `<ResourceType>` (e.g., `mychainid.chain`). This string is then hashed (e.g., SHA2-256) and encoded into a Content Identifier (CID) suitable for DHT queries.

3.  **Find Providers (DHT Lookup):** The client uses its libp2p DHT instance to perform a `FindProviders` query using the resource CID. The DHT network routes the query and returns a stream of `dev.AddrInfo` structures for nodes that have previously announced (via `Provide`) that they host this resource CID.

4.  **Select & Connect:** The client selects one or more suitable providers from the results. It attempts to establish a direct P2P connection using `host.Connect` with the provider's `dev.AddrInfo`. This process leverages libp2p's transport mechanisms and may involve NAT traversal techniques (like hole punching) facilitated by `libp2p.EnableHolePunching()`.

5.  **Make Request (Over P2P Stream):**
    *   Once a connection is established, the client opens a new stream to the provider using a KNIRVCHAIN-specific protocol ID (e.g., `/agent/chain/1.0`, `/agent/nrn/1.0`).
    *   The client sends request data over the stream, encoding the target Path (e.g., `/block`) and Query parameters (e.g., `number=123`). The exact encoding format (e.g., JSON, Protobuf, simple string commands) needs to be defined per protocol ID.
    *   The provider dev's stream handler receives the request, parses the Path and Query, performs the requested action (e.g., retrieves the block from its database), and sends the response back over the stream.

6.  **Process Response:** The client receives the response data over the stream, decodes it, and presents the result to the user or calling application.

**Note:** Direct HTTP requests to a node's public IP are generally discouraged in favor of P2P stream communication for better decentralization and security, although HTTP might be used for initial bootstrapping or specific gateway services.

**7. URI Minting and DHT Interaction**

Creating new, globally unique identifiers and associating them with resources involves the root KNIRVCHAIN network and the DHT.

1.  **Generation Request:** A user requests a new URI via a trusted interface (e.g., a root node's `/uriGenerator` HTTP endpoint). They might suggest a human-readable `<ID>`.

2.  **Availability Check:** The service handling the request (e.g., the root node) must query the DHT using `FindProviders` for the CID derived from `<requested_ID>.<ResourceType>`.
    *   If providers are found, the ID is unavailable. The service should either reject the request or generate a guaranteed unique ID (e.g., based on a UUID or transaction hash).
    *   If no providers are found after a reasonable timeout, the ID is considered available.

3.  **On-Chain Minting:** A transaction is created and included in a block on the root blockchain. This transaction immutably records the association of the chosen `<ID>` with its `<ResourceType>` and potentially other metadata (root, timestamp, initial properties). This on-chain record is the ultimate source of truth for the URI's existence.

4.  **DHT Announcement:** Crucially, after the minting transaction is confirmed, the service handling the minting (or the root's node) must announce itself as a provider for the new resource on the DHT. This is done by calling `dht.Provide` with the CID derived from `<chosen_ID>.<ResourceType>`. This step makes the newly minted URI discoverable by other devs via the resolution process described in Section 6. This announcement should be periodically refreshed.

**8. Go Implementation Notes**

*   **Parsing:** Use `net/url.Parse` for initial parsing. Implement a helper function to split `parsedURL.Host` into `id` and `resourceType` based on the last dot (`.`). Validate the scheme and resource type.

```go
import (
    "fmt"
    "net/url"
    "strings"
)

type agentURI struct {
    ID           string
    ResourceType string
    Path         string
    Query        url.Values
    Raw          string
}

func ParseagentURI(uriString string) (*agentURI, error) {
    parsedURL, err := url.Parse(uriString)
    if err != nil {
        return nil, fmt.Errorf("failed to parse URI: %w", err)
    }

    if parsedURL.Scheme != "agent" {
        return nil, fmt.Errorf("invalid scheme: expected 'agent', got '%s'", parsedURL.Scheme)
    }

    host := parsedURL.Host
    lastDot := strings.LastIndex(host, ".")
    if lastDot <= 0 || lastDot == len(host)-1 { // Ensure dot exists and isn't first/last char
         return nil, fmt.Errorf("invalid authority format: expected '<ID>.<ResourceType>', got '%s'", host)
    }

    id := host[:lastDot]
    resourceType := host[lastDot+1:] // Exclude the dot

    // Basic validation
    switch resourceType {
    case "chain", "nrn":
        // Known types
    default:
        // Allow unknown types for future extensibility, or return error if strict
        // return nil, fmt.Errorf("unknown resource type: %s", resourceType)
    }

    path := parsedURL.Path
    if path == "" {
        path = "/" // Default path
    }

    return &agentURI{
        ID:           id,
        ResourceType: resourceType,
        Path:         path,
        Query:        parsedURL.Query(),
        Raw:          uriString,
    }, nil
}
```

*   **Resolution:** The `DiscoveryManager` should expose functions like `ResolveAndConnect(uri *agentURI)` that encapsulate steps 2-4 from Section 6.

*   **P2P Request Handling:** Define specific libp2p protocol IDs (e.g., `/agent/chain/req/1.0`). Implement stream handlers on the server-side nodes that parse incoming requests (containing path/query info) and route them to appropriate blockchain or content logic. Use libraries like `go-libp2p-gorpc` or define custom message formats (Protobuf, JSON) for stream communication.

*   **DHT Interaction:** Use the `dual.DHT` (or underlying `kaddht`) interface methods: `dht.Provide(ctx, resourceCID, true)` for announcing and `routingDiscovery.FindProviders(ctx, resourceCID)` for discovery. Ensure proper CID generation from the `<ID>.<ResourceType>` string.

**9. Security Considerations**

*   **ID Squatting:** The "first come, first served" nature of the DHT availability check means desirable IDs could be claimed early. The on-chain minting record provides authority, but discovery relies on the DHT announcement.

*   **DHT Pollution:** Malicious nodes could announce providers for CIDs they don't actually serve. Clients should verify devs after connection.

*   **Authentication/Authorization:** Access control for modifying resources (e.g., submitting transactions to a `.chain`, updating `.nrn` content) must be handled at the application layer, typically involving cryptographic signatures verified against the relevant blockchain state. The URI scheme itself doesn't provide authentication.

**10. Conclusion**

The `agent://<ID>.<ResourceType>/<Path>?<Query>` scheme provides a flexible and decentralized method for addressing resources within the KNIRVCHAIN network. Its resolution relies heavily on the libp2p DHT for discovering providers, followed by P2P stream communication for actual interaction. Proper implementation of minting, announcement, and resolution is key to the system's functionality.
```