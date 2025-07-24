# agent Tunnel Relay Implementation Plan - Updated

## 1. Current State Analysis

### 1.1 Existing Components

#### Go Components
- **URI Resolver**: Basic implementation exists in `uri/uri_resolver.go` with support for different connection types
- **Node.js Manager**: Implemented in `nodejsmanager.go` with ability to start and manage the tunnel registry service
- **Discovery Manager**: Implemented in `discovery_manager.go` with DHT functionality
- **Discovery Interface**: Defined in `discovery_interface.go` with methods for finding and announcing resources

#### Node.js Components
- **Registry Manager**: Implemented in `agent-tunnel-registry/registry/registryManager.js` with in-memory storage
- **URI Routes**: Basic implementation in `agent-tunnel-registry/api/uriRoutes.js` with URI generation and resolution
- **Control Listener**: Implemented in `agent-tunnel-registry/tunneling/controlListener.js` for handling control connections
- **Tunnel Manager**: Implemented in `agent-tunnel-registry/tunneling/tunnelManager.js` for managing tunneled connections
- **Public Relay Listener**: Implemented in `agent-tunnel-registry/tunneling/publicRelayListener.js` for handling external connections

### 1.2 Missing Components

#### Go Components
- **Internal API Endpoints**: The blockchain server does not have the required internal API endpoints for DHT queries and DB access
- **Discovery Manager Integration**: No direct integration between the Node.js manager and the discovery manager

#### Node.js Components
- **DHT Integration**: The registry manager does not properly integrate with the Go DHT via internal APIs
- **Blockchain Integration**: No integration with the blockchain database for capability lookups

## 1.3 Architecture Overview

The implementation follows a decentralized network architecture where bootnodes use a shared blockchain and DHT as the central registry, rather than maintaining separate namespaces. This approach aligns with the core concept of a decentralized network where:
- The blockchain (LevelDB) provides the immutable record
- The DHT provides resource discoverability

### Key Components

#### Shared Registry
- **Blockchain (LevelDB)**: Single, shared registry for the entire network
- **Global DHT**: Managed by the Go libp2p host across all nodes

#### Bootnode Roles
- **Access Points**: Run Node.js agent-tunnel-registry service to expose HTTP API endpoints
  - `/api/uri/resolve`
  - `/api/mcp/capability/prepare_registration`
  - Tunnel relay service
- **DHT Nodes**: Participate in the global DHT via Go processes
  - Manage libp2p host
  - Manage DHT client

#### Inter-Process Communication
- Node.js service communicates with co-located Go process via internal HTTP APIs
- Go process exposes endpoints for:
  - DHT queries
  - Blockchain database access

## 2. Key Processes

### 2.1 URI Resolution
**Format**: `agent://<bootnode_authority>/<identifier>.<type>/`

#### Current Implementation
- The URI resolver in Go (`uri/uri_resolver.go`) can parse and resolve URIs
- The Node.js URI routes (`agent-tunnel-registry/api/uriRoutes.js`) handle URI resolution requests
- The registry manager (`agent-tunnel-registry/registry/registryManager.js`) maintains a local registry of nodes

#### Missing Components
- The Node.js service does not call the Go internal API for DHT queries
- The Go blockchain server does not expose the required internal API endpoints

#### Process Flow (To Be Implemented)
1. Client extracts `<bootnode_authority>` (public hostname/IP of a bootnode)
2. Client sends HTTP GET request to `http://<bootnode_authority>:3003/api/uri/resolve?uri=<full_uri>`
3. Node.js service parses URI to extract `<identifier>` and `<type>`
4. Node.js service calls internal Go API: `http://localhost:<go_internal_api_port>/internal/dht/findResource?id=<identifier>&type=<type>`
5. Response handling:
   - If DHT returns provider information (multiaddresses):
     - Construct `DIRECT_P2P` or `RELAYED_P2P_CIRCUIT` connection details
   - If not found on DHT:
     - Call internal Go API: `http://localhost:<go_internal_api_port>/internal/db/getCapability?id=<identifier>`
     - If capability record found:
       - Return `BLOCKCHAIN_RECORD` connection details
     - If identifier not found:
       - Return 404
   - If identifier corresponds to tunneled node on this bootnode:
     - Return `TUNNELED` connection details with relay information

### 2.2 URI Generation
**Used for**: New nodes/resources during installation

#### Current Implementation
- The URI generation functionality exists in Go (`uri/uri_generation.go`)
- The Node.js URI routes (`agent-tunnel-registry/api/uriRoutes.js`) handle URI generation requests
- The registry manager can map tunneled resources to node IDs

#### Missing Components
- The Node.js service does not call the Go internal API to check if an ID exists
- The Go blockchain server does not expose the required internal API endpoint

#### Process Flow (To Be Implemented)
1. Node connects to chosen bootnode (specified in install config)
2. Node requests URI from bootnode's `/api/uri/generate` endpoint
3. Node.js service generates unique ID (UUID)
4. Node.js service calls internal Go API: `http://localhost:<go_internal_api_port>/internal/db/idExists?id=<generated_id>`
5. If ID is available:
   - Generate URI using bootnode's public hostname: `agent://<this_bootnode_public_host>/<generated_id>.<type>/`
   - Call internal Go API: `http://localhost:<go_internal_api_port>/internal/dht/announceResource`
   - Return generated URI to requesting node
6. Requesting node registers connectivity details with a bootnode

### 2.3 Node Registration

#### Current Implementation
- The control listener (`agent-tunnel-registry/tunneling/controlListener.js`) handles control connections
- The registry manager can register nodes via control socket or API
- The tunnel manager can relay connections between external clients and internal nodes

#### Missing Components
- The Node.js service does not call the Go internal API to announce resources on the DHT
- The Go blockchain server does not expose the required internal API endpoint

#### NAT-Restricted Nodes (To Be Implemented)
1. Node establishes control connection to bootnode
2. Node.js service registers node as `isTunneled: true` in local `registryManager.js`

#### Publicly Reachable Nodes (To Be Implemented)
1. Node registers via bootnode's HTTP API: `/api/registry/register`
2. Node.js service registers node as `isTunneled: false` in local `registryManager.js`
3. Node.js service calls Go internal API: `/internal/dht/announceResource`
4. Go process announces node's multiaddress on global DHT

## 3. Implementation Details

### 3.1 Go Components

#### 3.1.1 Current Implementation vs. Required Changes

##### Main Application (`main.go`)
```go
// Node.js services (for root and bootnode roles)
var nodejsManager *NodeJSManager // Declare nodejsManager
if (cfg.IsRoot || cfg.IsBootnode) && cfg.NodeJSServices.Enabled {
    var errNodeJS error
    // Pass the discoveryMgr and bc (blockchain) to the Node.js manager initializer
    // so it can provide access to the DHT and DB via internal APIs.
    nodejsManager, errNodeJS = initNodeJSServices(&cfg, discoveryMgr, bc)
    if errNodeJS != nil {
        log.Printf("[%s][%s] ERROR: Failed to initialize Node.js services: %v", cfg.ChainID, role.String(), errNodeJS)
        // Continue execution even if Node.js services fail to start
    }
    if nodejsManager != nil {
        defer func() {
            log.Printf("[%s][%s] Stopping Node.js services...", cfg.ChainID, role.String())
            nodejsManager.StopAllServices()
            log.Printf("[%s][%s] Node.js services stopped", cfg.ChainID, role.String())
        }()
    }
}

// Blockchain HTTP Server
blockchainSrv := NewBlockchainServer(uint64(cfg.Port), bc, db, discoveryMgr, int(cfg.P2PPort))
```

#### 3.1.2 Current Implementation vs. Required Changes for Blockchain Server (`blockchain_server.go`)
```go
// Add internal API endpoints for Node.js services
mux.HandleFunc("/internal/dht/findResource", bcs.handleInternalDHTFindResource)
mux.HandleFunc("/internal/db/getCapability", bcs.handleInternalDBGetCapability)
mux.HandleFunc("/internal/db/idExists", bcs.handleInternalDBIDExists)
mux.HandleFunc("/wallet/info", bcs.handleWalletInfo) // New handler
```

##### Internal API Handlers
```go
// handleInternalDHTFindResource handles internal requests to find a resource on the DHT
func (bcs *BlockchainServer) handleInternalDHTFindResource(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    if bcs.discoveryManager == nil {
        http.Error(w, "Discovery service not available", http.StatusInternalServerError)
        return
    }

    id := r.URL.Query().Get("id")
    resourceTypeStr := r.URL.Query().Get("type") // e.g., "chain", "capability"

    if id == "" || resourceTypeStr == "" {
        http.Error(w, "Missing 'id' or 'type' query parameter", http.StatusBadRequest)
        return
    }

    resourceType := DiscoveryResourceType(resourceTypeStr)
    if resourceType == "" {
        http.Error(w, "Invalid resource type", http.StatusBadRequest)
        return
    }

    // Use a context with timeout for the DHT query
    ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
    defer cancel()

    providers, err := bcs.discoveryManager.FindResource(ctx, id, resourceType)
    if err != nil {
        // Log the error but return 404 if no providers found
        if strings.Contains(err.Error(), "no providers found") || strings.Contains(err.Error(), "failed to find any dev in table") {
            http.Error(w, fmt.Sprintf("Resource '%s' of type '%s' not found on DHT", id, resourceTypeStr), http.StatusNotFound)
        } else {
            log.Printf("Error finding resource '%s' on DHT: %v", id, err)
            http.Error(w, "Internal server error during DHT lookup", http.StatusInternalServerError)
        }
        return
    }

    // Return the provider information (e.g., multiaddrs)
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(providers) // Assuming providers is a slice of peer.AddrInfo
}

// handleInternalDBGetCapability handles internal requests to get a capability from the database
func (bcs *BlockchainServer) handleInternalDBGetCapability(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    id := r.URL.Query().Get("id")
    if id == "" {
        http.Error(w, "Missing 'id' query parameter", http.StatusBadRequest)
        return
    }

    capability, err := bcs.db.GetCapabilityByID(id)
    if err != nil {
        http.Error(w, fmt.Sprintf("Capability '%s' not found in DB: %v", id, err), http.StatusNotFound)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(capability) // Assuming capability is a serializable struct/map
}

// handleInternalDBIDExists handles internal requests to check if an ID exists in the blockchain (DB)
func (bcs *BlockchainServer) handleInternalDBIDExists(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    id := r.URL.Query().Get("id")
    if id == "" {
        http.Error(w, "Missing 'id' query parameter", http.StatusBadRequest)
        return
    }

    // Check if the ID exists in blocks or the transaction pool
    // Note: This only checks the local node's view of the blockchain.
    // For true global uniqueness, a DHT check is also necessary.
    bcs.BlockchainPtr.Lock()
    existsInBlocks := bcs.BlockchainPtr.CheckIfIDExistsInBlocks(id)
    existsInPool := bcs.BlockchainPtr.CheckIfIDExistsInTransactionPool(id)
    bcs.BlockchainPtr.Unlock()

    exists := existsInBlocks || existsInPool

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]bool{"exists": exists})
}
```

#### 3.1.3 Current Implementation vs. Required Changes for Node.js Manager (`nodejsmanager.go`)
```go
// NewNodeJSManager creates a new Node.js process manager, accepting DiscoveryService and Blockchain
func NewNodeJSManager(cfg *config.NodeJSServicesConfig, nodeID string, discoveryMgr DiscoveryService, bc *BlockchainStruct) *NodeJSManager {
    return &NodeJSManager{
        Config:    cfg,
        Processes: make(map[string]*NodeJSProcess),
        PeerID:    nodeID,
        DiscoveryMgr: discoveryMgr,
        Blockchain: bc,
    }
}

// StartTunnelRegistry starts the tunnel registry service
func (m *NodeJSManager) StartTunnelRegistry() error {
    // Environment variables for the Node.js process
    env := []string{
        fmt.Sprintf("PUBLIC_HOST=%s", m.Config.TunnelRegistry.ServerPublicHost),
        fmt.Sprintf("RELAY_SERVER_PEER_ID=%s", m.PeerID),
    }
    // Note: Node.js service will call back to Go via internal HTTP APIs for DHT/DB access

    // Prepare the command
    cmd := exec.Command("node", scriptPath)
    // Additional implementation details...
}
```

#### 3.1.4 Current Implementation vs. Required Changes for URI Resolver (`uri_resolver.go`)
```go
// Add Authority field to ResolvedURI struct
type ResolvedURI struct {
    ResolvedIdentifier string `json:"resolvedIdentifier"`
    ResourceType       string `json:"resourceType"`
    SubPathWithQuery   string `json:"subPathWithQuery"`
    Authority          string `json:"authority"` // Add Authority field
    ConnectionType     string `json:"connectionType"`
    TargetPeerID       string `json:"targetPeerId"`
    Multiaddress       string `json:"multiaddress,omitempty"`
}
```

### 3.2 Node.js Components

#### 3.2.1 Current Implementation vs. Required Changes for Registry Manager (`registryManager.js`)
```javascript
// agent-tunnel-registry/registry/registryManager.js
const axios = require('axios'); // For HTTP calls
const config = require('../config');
const { v4: uuidv4 } = require('uuid');

class RegistryManager {
    constructor() {
        // In-memory state tracking nodes directly connected to this bootnode
        this.nodes = new Map(); // devId -> nodeInfo
        this.chainIdToPeerId = new Map(); // chainId -> devId
        this.tunneledResourceIds = new Map(); // uniqueId -> devId
    }
    
    // Node registration methods
    registerNodeViaControlSocket(devId, chainId, internalIp, internalP2pPort, type, controlSocketId, serverPublicHost, publicRelayPort) {
        // Implementation for tunneled nodes
        let nodeInfo = this.nodes.get(devId);
        if (!nodeInfo) {
            nodeInfo = {
                devId,
                chainId: chainId || devId, // Fallback if chainId not distinct
                type,
                lastSeen: Date.now(),
                controlSocketId, // ID of the control socket for tunneling
                publicIp: serverPublicHost, // Public IP of *this* tunnel server
                isTunneled: true,
                // The public address clients should use to reach this node via THIS tunnel server
                publicRelayUrl: `tcp://${serverPublicHost}:${publicRelayPort}/p2p_tunnel/${devId}`
            };
        } else {
            // Update existing node info
            nodeInfo.controlSocketId = controlSocketId;
            nodeInfo.isTunneled = true;
        }
        this.nodes.set(devId, nodeInfo);
        if (nodeInfo.chainId) {
            this.chainIdToPeerId.set(nodeInfo.chainId, devId);
        }
        return nodeInfo;
    }
    
    registerPublicNode(devId, chainId, publicIp, publicP2pPort, type) {
        // Implementation for public nodes
        let nodeInfo = {
            devId,
            chainId: chainId || devId,
            publicIp,
            publicP2pPort,
            type,
            lastSeen: Date.now(),
            controlSocketId: null, // Not connected via control channel
            isTunneled: false,
            publicRelayUrl: null // Not tunneled through this server
        };
        this.nodes.set(devId, nodeInfo);
        
        // For publicly registered nodes, announce on the global DHT via the Go process
        axios.post(`http://localhost:${config.goInternalApiPort}/internal/dht/announceResource`, { 
            id: devId, 
            type: type || 'dev', 
            multiaddress: `TODO: Construct Multiaddress from publicIp and publicP2pPort` 
        }).catch(err => console.error(`[Registry] Failed to announce node ${devId} on DHT: ${err.message}`));
        
        if (nodeInfo.chainId) {
            this.chainIdToPeerId.set(nodeInfo.chainId, devId);
        }
        return nodeInfo;
    }
    
    // Additional methods...
}
```

#### 3.2.2 URI Routes (`uriRoutes.js`)
```javascript
// agent-tunnel-registry/api/uriRoutes.js
const express = require('express');
const registryManager = require('../registry/registryManager');
const axios = require('axios');
const config = require('../config');
const { v4: uuidv4 } = require('uuid');

// URI generation endpoint
router.post('/generate', (req, res) => {
    const { devId, resourceType = 'dev', subPath = '' } = req.body;
    
    // Verify the node is registered
    const nodeInfo = registryManager.getNodeByPeerId(devId);
    if (!nodeInfo) {
        return res.status(404).json({ error: `Node with devId ${devId} not registered. Register first or connect via control channel.` });
    }

    // Generate a unique ID for this resource
    const resourceSpecificId = uuidv4();

    // Check if this generated ID already exists globally (on blockchain/DHT)
    const idExistsUrl = `http://localhost:${config.goInternalApiPort}/internal/db/idExists?id=${resourceSpecificId}`;
    axios.get(idExistsUrl).then(response => {
        if (response.data && response.data.exists) {
            // If the randomly generated ID exists, try again
            console.warn(`[URI Generate] Generated ID ${resourceSpecificId} already exists, retrying...`);
            router.handle('POST', req, res); // Re-route the request internally
            return;
        }

        // Map the unique resource ID to the node's devId in the local registry
        registryManager.mapTunneledResource(resourceSpecificId, devId);
        
        // Generate and return the URI
        let uri;
        if (nodeInfo.isTunneled) {
            uri = `agent://${config.serverPublicHost}/${resourceSpecificId}.${resourceType}${subPath ? '/' + subPath.replace(/^\//, '') : ''}`;
        } else {
            uri = `agent://${nodeInfo.publicIp || nodeInfo.devId}/${resourceSpecificId}.${resourceType}${subPath ? '/' + subPath.replace(/^\//, '') : ''}`;
        }
        
        res.json({
            uri,
            resourceId: resourceSpecificId,
            resourceType,
            subPath,
            directInfo: !nodeInfo.isTunneled ? { ip: nodeInfo.publicIp, port: nodeInfo.publicP2pPort } : null
        });
    });
});

// URI resolution endpoint
router.get('/resolve', (req, res) => {
    const { uri } = req.query;
    if (!uri) {
        return res.status(400).json({ error: "Missing 'uri' query parameter" });
    }
    
    try {
        // Parse the URI to extract components
        // agent://<bootnode_authority>/<identifier>.<type>/[<subpath>][?<query>]
        const parsedUri = new URL(uri);
        const pathParts = parsedUri.pathname.split('/');
        const identifierWithType = pathParts[1];
        const [identifier, resourceType] = identifierWithType.split('.');
        const subPathWithQuery = parsedUri.pathname.substring(identifierWithType.length + 1) + 
                                (parsedUri.search || '');
        
        // Determine if this is a tunneled resource or direct dev
        let connectionDetails = null;
        
        // Check if this is a tunneled resource ID
        const targetPeerIdForTunneled = registryManager.getPeerIdForTunneledResource(identifier);
        
        if (targetPeerIdForTunneled) {
            // Handle tunneled resource
            // Implementation details...
        }
        
        // If not tunneled, check if it's a direct dev ID or chain ID
        let nodeInfo = registryManager.getNodeByPeerId(identifier) || 
                      registryManager.getNodeByChainId(identifier);
        
        if (nodeInfo) {
            // Construct connection details based on node info
            if (nodeInfo.isTunneled) {
                // For tunneled nodes, provide relay information
                // Implementation details...
            } else if (nodeInfo.publicIp && nodeInfo.publicP2pPort && nodeInfo.devId) {
                // For directly reachable nodes
                // Implementation details...
            }
        }
        
        if (connectionDetails === null) {
            // If not a tunneled resource ID handled by this specific bootnode,
            // check the global DHT via the Go process.
            const dhtLookupUrl = `http://localhost:${config.goInternalApiPort}/internal/dht/findResource?id=${identifier}&type=${resourceType}`;
            axios.get(dhtLookupUrl).then(response => {
                if (response.data && response.data.length > 0) {
                    // Found providers on the DHT
                    // Implementation details...
                } else {
                    // If not found on DHT, check the blockchain for capability records
                    const dbLookupUrl = `http://localhost:${config.goInternalApiPort}/internal/db/getCapability?id=${identifier}`;
                    axios.get(dbLookupUrl).then(dbResponse => {
                        // Found a capability record on the blockchain
                        // Implementation details...
                    }).catch(dbError => {
                        // Not found in DB either
                        res.status(404).json({ error: `Resource with identifier '${identifier}' not found.` });
                    });
                }
            }).catch(dhtError => {
                res.status(500).json({ error: 'Internal server error during DHT lookup.' });
            });
            return; // Exit here as the response will be sent asynchronously
        }
        
        // If connectionDetails was set, send the response immediately
        res.json({
            originalUri: uri,
            resolvedIdentifier: identifier,
            resourceType: resourceType,
            subPathWithQuery: subPathWithQuery,
            ...connectionDetails
        });
        
    } catch (error) {
        console.error(`[URI Resolve] Error processing URI '${uri}':`, error);
        res.status(500).json({ error: 'Internal server error during URI resolution.' });
    }
});

module.exports = router;
```

## 4. Benefits of This Approach

1. **Decentralized Architecture**: Bootnodes become interchangeable access points to a single network state
2. **Robust Design**: No single point of failure for resource discovery
3. **Efficient Resource Location**: Leverages both DHT and blockchain for comprehensive resource discovery
4. **Separation of Concerns**: 
   - Go components handle core P2P and blockchain functionality
   - Node.js components handle HTTP APIs and tunneling
5. **Scalability**: New bootnodes can join the network and immediately participate in the shared registry

## 5. Implementation Roadmap

1. **Phase 1**: Implement internal API endpoints in Go components
   - Add handlers for DHT queries, capability lookups, and ID existence checks
   - Update BlockchainServer to expose these endpoints

2. **Phase 2**: Update Node.js components to use internal APIs
   - Modify registryManager.js to communicate with Go process
   - Update URI routes to leverage the shared registry

3. **Phase 3**: Implement URI resolution and generation logic
   - Complete the URI parsing and resolution flow
   - Implement the resource ID generation and verification

4. **Phase 4**: Implement node registration and tunneling
   - Finalize the control socket connection handling
   - Complete the public node registration process

5. **Phase 5**: Testing and optimization
   - Test URI resolution across multiple bootnodes
   - Verify DHT announcements and lookups
   - Optimize performance of inter-process communication

This implementation plan provides a comprehensive approach to creating a decentralized tunnel relay system that leverages both blockchain and DHT technologies for resource discovery and connectivity.