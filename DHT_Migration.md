# DHT Migration Plan: KNIRVGRAPH + KNIRVCHAIN → KNIRVGATEWAY
## With Exclusive Unix Socket P2P Communication

**Date:** 2026-04-28  
**Status:** Draft  
**Author:** opencode/big-pickle  

---

## Executive Summary

**Goal:** Consolidate all direct DHT functionality into KNIRVGATEWAY (operated via KNIRVSERVER), removing DHT code from KNIRVGRAPH and KNIRVCHAIN, with **all service-to-service P2P communication routed exclusively through Unix sockets**.

**Key Constraint:** KNIRVCHAIN and KNIRVGRAPH must communicate with KNIRVGATEWAY's DHT API exclusively via Unix sockets (not TCP loopback).

**NEW: Resource Broadcast System**
KNIRVGRAPH and KNIRVCHAIN will use a **Resource Broadcast System** where:
1. Newly added resources are sent to KNIRVGATEWAY via `CacheResource()` over Unix socket
2. KNIRVGATEWAY maintains a `ResourceCache` list of all resources from both services
3. A background worker in KNIRVGATEWAY periodically announces all cached resources to the DHT
4. Manual trigger available via `/dht/announce-all-cached` endpoint

**Current State:**
- **KNIRVGRAPH** has full Kademlia DHT in `internal/p2p/dht_manager.go` (645 lines)
- **KNIRVCHAIN** has DHT in `internal/p2p/discovery_manager.go` with HTTP endpoints `/internal/dht/*`
- **KNIRVGATEWAY** has only placeholder DHT endpoints - needs full implementation
- **KNIRVSERVER** already embeds KNIRVGATEWAY and communicates via `gateway.sock`

---

## Target Architecture

### BEFORE (Current State)
```
KNIRVGRAPH ───(own DHT over TCP)──> Kademlia Network
KNIRVCHAIN ───(own DHT over TCP)──> Kademlia Network
KNIRVGATEWAY ──(no DHT)──> Placeholder endpoints
```

### AFTER (Target State)
```
KNIRVGRAPH ──Unix Socket──> KNIRVGATEWAY ───(DHT over TCP)──> Kademlia Network
KNIRVCHAIN ──Unix Socket──> KNIRVGATEWAY ───(DHT over TCP)──> Kademlia Network
                    ↑
           (operated by KNIRVSERVER via gateway.sock)
```

**Key Points:**
- Service-to-service API calls: Unix socket only (`gateway.sock`)
- DHT P2P network: TCP (libp2p limitation - Go implementation doesn't support Unix sockets)
- KNIRVSERVER operates KNIRVGATEWAY as embedded subprocess (existing pattern)

---

## Resource Broadcast System Architecture

### Overview

Instead of KNIRVGRAPH and KNIRVCHAIN directly announcing resources to the DHT (which would require each to maintain their own DHT node), they now **cache resources** in KNIRVGATEWAY. A background worker in KNIRVGATEWAY then announces all cached resources to the DHT periodically.

### Benefits

1. **Centralized DHT Management:** Only KNIRVGATEWAY runs a DHT node (operated by KNIRVSERVER)
2. **Reduced Network Traffic:** Resources are batched and announced together
3. **Simplified Clients:** KNIRVGRAPH and KNIRVCHAIN only need Unix socket clients (no DHT code)
4. **Consistent State:** KNIRVGATEWAY's cache is the single source of truth for resources to announce

### Architecture Diagram

```
┌─────────────┐         ┌─────────────┐
│ KNIRVGRAPH  │         │ KNIRVCHAIN  │
└──────┬──────┘         └──────┬──────┘
       │                       │
       │ CacheResource()       │ CacheResource()
       │ (Unix Socket)        │ (Unix Socket)
       │                       │
       ▼                       ▼
┌─────────────────────────────────────────┐
│         KNIRVGATEWAY (gateway.sock)    │
│  ┌─────────────────────────────────┐   │
│  │     ResourceCache                │   │
│  │  - knirvgraph resources        │   │
│  │  - knirvchain resources        │   │
│  │  - timestamps                  │   │
│  └──────────────┬──────────────────┘   │
│                 │                        │
│                 │ startAnnouncementWorker()
│                 │ (every 5 minutes)     │
│                 ▼                        │
│  ┌─────────────────────────────────┐   │
│  │     DHT Manager                 │   │
│  │     - Provide(cid)             │   │
│  │     - FindProviders()           │   │
│  └──────────────┬──────────────────┘   │
└──────────────────┼───────────────────────┘
                   │ (TCP)
                   ▼
            Kademlia DHT Network
```

### Data Flow

1. **Resource Addition (KNIRVGRAPH/KNIRVCHAIN):**
   ```go
   // KNIRVGRAPH announces a new skill
   dhtClient.CacheResource(ctx, skillID, "skill", multiaddr)
   
   // KNIRVCHAIN announces a new chain resource
   dhtClient.CacheResource(ctx, chainID, "chain")
   ```

2. **Cache Storage (KNIRVGATEWAY):**
   ```go
   // POST /dht/cache-resource
   {
       "id": "skill123",
       "type": "skill",
       "source": "knirvgraph",
       "timestamp": "2026-04-28T10:30:00Z"
   }
   ```

3. **Background Announcement (KNIRVGATEWAY worker):**
   ```go
   // Every 5 minutes (configurable)
   for _, resource := range resourceCache.GetAllResources() {
       cid := generateCID(resource.ID, resource.Type)
       dhtManager.Provide(ctx, cid, true)
   }
   ```

4. **Manual Trigger (Optional):**
   ```bash
   curl -X POST http://localhost/dht/announce-all-cached
   ```

### Cache Structure

```go
type ResourceCache struct {
    resources map[string]ResourceEntry // key: "id:type"
}

type ResourceEntry struct {
    ID           string    // Resource identifier (e.g., skill ID, chain ID)
    Type         string    // Resource type ("skill", "capability", "property", "chain", "mcp:tool")
    Multiaddress string    // Optional multiaddress (for direct peer connection)
    Timestamp    time.Time // When resource was cached
    Source       string    // "knirvgraph" or "knirvchain"
}
```

### Endpoints Summary

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/dht/cache-resource` | POST | Add resource to KNIRVGATEWAY cache |
| `/dht/cache-status` | GET | View all cached resources |
| `/dht/announce-all-cached` | POST | Manually trigger announcement of all cached resources |
| `/dht/find` | GET | Query DHT for resource providers (unchanged) |
| `/dht/peers` | GET | List DHT peers (unchanged) |

---

## Migration Phases

### Phase 1: Implement DHT in KNIRVGATEWAY

**1.1 Add Go libp2p dependencies**
```bash
cd packages/KNIRVGATEWAY
go get github.com/libp2p/go-libp2p
go get github.com/libp2p/go-libp2p-kad-dht
go get github.com/libp2p/go-libp2p-pubsub
```

**1.2 Create `internal/dht/` package**

New files:
- `internal/dht/manager.go` - Main DHT manager (port from KNIRVGRAPH's `dht_manager.go`)
- `internal/dht/options.go` - Production/test DHT options
- `internal/dht/types.go` - Shared DHT types
- `internal/dht/cache.go` - **Resource list cache for broadcast system**

Key interface:
```go
type DHTManager interface {
    Start(ctx context.Context) error
    Stop() error
    Provide(ctx context.Context, cid cid.Cid) error
    FindProviders(ctx context.Context, cid cid.Cid) ([]peer.AddrInfo, error)
    AnnounceService(ctx context.Context, serviceID string) error
    FindServices(ctx context.Context, serviceID string) ([]peer.AddrInfo, error)
    PublishAnnouncement(ctx context.Context, topic string, data []byte) error
    SubscribeAnnouncements(ctx context.Context, topic string) (<-chan []byte, error)
}

// ResourceCache manages the list of resources to announce to DHT
type ResourceCache interface {
    AddResource(resource ResourceEntry)
    GetAllResources() []ResourceEntry
    ClearCache()
    GetResourceCount() int
}

type ResourceEntry struct {
    ID           string    `json:"id"`
    Type         string    `json:"type"`
    Multiaddress string    `json:"multiaddress,omitempty"`
    Timestamp    time.Time `json:"timestamp"`
    Source       string    `json:"source"` // "knirvgraph" or "knirvchain"
}
```

**1.3 Replace placeholder endpoints in `internal/server/server.go`**

Remove stubs:
```go
r.HandleFunc("/dht/status", s.handleDHTStatus).Methods("GET")
r.HandleFunc("/dht/start", s.handleDHTStart).Methods("POST")  
r.HandleFunc("/dht/stop", s.handleDHTStop).Methods("POST")
```

Add working endpoints:
```go
// Unix socket only - no TCP listener
r.HandleFunc("/dht/announce", s.handleDHTAnnounce).Methods("POST")
r.HandleFunc("/dht/find", s.handleDHTFind).Methods("GET")
r.HandleFunc("/dht/peers", s.handleDHTPeers).Methods("GET")
r.HandleFunc("/dht/bootstrap", s.handleDHTBootstrap).Methods("POST")

// NEW: Resource broadcast system endpoints
r.HandleFunc("/dht/cache-resource", s.handleCacheResource).Methods("POST")
r.HandleFunc("/dht/announce-all-cached", s.handleAnnounceAllCached).Methods("POST")
r.HandleFunc("/dht/cache-status", s.handleCacheStatus).Methods("GET")
```

**1.4 Configure DHT to listen on Unix socket (for local API) + TCP (for P2P)**

Note: libp2p DHT uses TCP for P2P. The Unix socket is for the HTTP API that KNIRVCHAIN/KNIRVGRAPH will call.

- Use existing config: `config.DisableDHT`, `config.DHTPort`, `config.BootstrapPeers`
- Initialize DHT manager in `server.go` on startup
- HTTP API listens on `gateway.sock` (already implemented in KNIRVGATEWAY main.go)

**1.5 Implement Resource Broadcast System in KNIRVGATEWAY**

Create `internal/dht/cache.go`:
```go
package dht

import (
    "sync"
    "time"
)

type ResourceCache struct {
    mu        sync.RWMutex
    resources map[string]ResourceEntry // key: "id:type"
}

func NewResourceCache() *ResourceCache {
    return &ResourceCache{
        resources: make(map[string]ResourceEntry),
    }
}

func (rc *ResourceCache) AddResource(resource ResourceEntry) {
    rc.mu.Lock()
    defer rc.mu.Unlock()
    resource.Timestamp = time.Now()
    key := fmt.Sprintf("%s:%s", resource.ID, resource.Type)
    rc.resources[key] = resource
}

func (rc *ResourceCache) GetAllResources() []ResourceEntry {
    rc.mu.RLock()
    defer rc.mu.RUnlock()
    result := make([]ResourceEntry, 0, len(rc.resources))
    for _, entry := range rc.resources {
        result = append(result, entry)
    }
    return result
}

func (rc *ResourceCache) GetResourceCount() int {
    rc.mu.RLock()
    defer rc.mu.RUnlock()
    return len(rc.resources)
}
```

**1.6 Background Announcement Worker**

In `internal/dht/manager.go`, add a background worker that periodically announces all cached resources:
```go
func (dm *DHTManager) startAnnouncementWorker(ctx context.Context, interval time.Duration) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            resources := dm.resourceCache.GetAllResources()
            for _, res := range resources {
                cid := generateCID(res.ID, res.Type)
                if err := dm.kadDHT.Provide(ctx, cid, true); err != nil {
                    log.Printf("Failed to announce cached resource %s: %v", res.ID, err)
                }
            }
        case <-ctx.Done():
            return
        }
    }
}
```

---

### Phase 2: Create Unix Socket DHT Client for KNIRVGRAPH (Resource Broadcast)

**2.1 Remove DHT manager from KNIRVGRAPH**

- Delete: `packages/KNIRVGRAPH/internal/p2p/dht_manager.go` (645 lines)
- Delete: `internal/p2p/dht_manager_*.go`
- Keep: Non-DHT files in `internal/p2p/` (if any)

**2.2 Create Unix socket DHT client with broadcast system**

New file: `packages/KNIRVGRAPH/internal/dht/client.go`

```go
package dht

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net"
    "net/http"
    "path/filepath"
    "time"
)

type Client struct {
    socketPath string
    httpClient  *http.Client
    source      string // "knirvgraph"
}

func NewClient(socketPath string) *Client {
    return &Client{
        socketPath: socketPath,
        httpClient: &http.Client{
            Timeout: 10 * time.Second,
            Transport: &http.Transport{
                DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
                    return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
                },
            },
        },
        source: "knirvgraph",
    }
}

// CacheResource caches a resource in KNIRVGATEWAY for later DHT announcement
// NEW: Uses broadcast system - resource goes to cache, KNIRVGATEWAY announces to DHT
func (c *Client) CacheResource(ctx context.Context, id string, resourceType string, multiaddr string) error {
    url := "http://localhost/dht/cache-resource"
    
    payload := map[string]interface{}{
        "id":           id,
        "type":         resourceType,
        "multiaddress": multiaddr,
        "source":       c.source,
    }
    
    jsonPayload, _ := json.Marshal(payload)
    
    resp, err := c.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonPayload))
    if err != nil {
        return fmt.Errorf("failed to cache resource: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("cache resource returned status %d", resp.StatusCode)
    }
    
    return nil
}

// AnnounceSkill caches skill resource for DHT announcement via broadcast system
func (c *Client) AnnounceSkill(ctx context.Context, skillID string, multiaddr string) error {
    return c.CacheResource(ctx, skillID, "skill", multiaddr)
}

// AnnounceCapability caches capability resource for DHT announcement via broadcast system
func (c *Client) AnnounceCapability(ctx context.Context, capID string, multiaddr string) error {
    return c.CacheResource(ctx, capID, "capability", multiaddr)
}

// AnnounceProperty caches property resource for DHT announcement via broadcast system
func (c *Client) AnnounceProperty(ctx context.Context, propID string, multiaddr string) error {
    return c.CacheResource(ctx, propID, "property", multiaddr)
}

// FindResource finds resource providers via KNIRVGATEWAY Unix socket (direct DHT query)
func (c *Client) FindResource(ctx context.Context, id string, resourceType string) ([]peer.AddrInfo, error) {
    url := fmt.Sprintf("http://localhost/dht/find?id=%s&type=%s", id, resourceType)
    
    resp, err := c.httpClient.Get(url)
    if err != nil {
        return nil, fmt.Errorf("DHT find failed: %w", err)
    }
    defer resp.Body.Close()
    
    var peers []peer.AddrInfo
    if err := json.NewDecoder(resp.Body).Decode(&peers); err != nil {
        return nil, err
    }
    
    return peers, nil
}
```

**2.3 Update NRV system**

- File: `internal/nrv/nrv_system.go`
- Remove direct DHT calls to Kademlia
- Replace with calls to `dht.Client.CacheResource()` (adds to KNIRVGATEWAY broadcast cache)

**2.4 Update `app.go`**

- Remove DHT manager initialization
- Remove direct `AnnounceSkill()`, `AnnounceCapability()`, `AnnounceProperty()` DHT calls
- Add DHT client initialization with socket path:

```go
// In app.go initialization
gatewaySocketPath := filepath.Join(appDataDir, "sockets", "gateway.sock")
dhtClient := dht.NewClient(gatewaySocketPath)
// Resources are now cached and announced via KNIRVGATEWAY broadcast system
```

---

### Phase 3: Create Unix Socket DHT Client for KNIRVCHAIN (Resource Broadcast)

**3.1 Remove discovery manager from KNIRVCHAIN**

- Delete: `packages/KNIRVCHAIN/internal/p2p/discovery_manager.go`

**3.2 Create Unix socket DHT client with broadcast system**

New file: `packages/KNIRVCHAIN/internal/dht/client.go`

```go
package dht

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net"
    "net/http"
    "path/filepath"
    "time"
)

type DiscoveryClient struct {
    socketPath string
    httpClient  *http.Client
    source      string // "knirvchain"
}

func NewDiscoveryClient(socketPath string) *DiscoveryClient {
    return &DiscoveryClient{
        socketPath: socketPath,
        httpClient: &http.Client{
            Timeout: 10 * time.Second,
            Transport: &http.Transport{
                DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
                    return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
                },
            },
        },
        source: "knirvchain",
    }
}

// CacheResource caches a resource in KNIRVGATEWAY for later DHT announcement
// NEW: Uses broadcast system - resource goes to cache, KNIRVGATEWAY announces to DHT
func (c *DiscoveryClient) CacheResource(ctx context.Context, id, resourceType string) error {
    url := "http://localhost/dht/cache-resource"
    
    payload := map[string]interface{}{
        "id":     id,
        "type":   resourceType,
        "source": c.source,
    }
    
    jsonPayload, _ := json.Marshal(payload)
    resp, err := c.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonPayload))
    if err != nil {
        return fmt.Errorf("failed to cache resource: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("cache resource returned status %d", resp.StatusCode)
    }
    
    return nil
}

// AnnounceGenericResource caches resource for DHT announcement via broadcast system
func (c *DiscoveryClient) AnnounceGenericResource(ctx context.Context, id string, resourceType types.DiscoveryResourceType) error {
    return c.CacheResource(ctx, id, string(resourceType))
}

// AnnounceMCPCapability caches MCP capability for DHT announcement
func (c *DiscoveryClient) AnnounceMCPCapability(ctx context.Context, id, mcpType string) error {
    resourceType := fmt.Sprintf("mcp:%s", mcpType)
    return c.CacheResource(ctx, id, resourceType)
}

// AnnounceMintedResource caches minted resource for DHT announcement
func (c *DiscoveryClient) AnnounceMintedResource(ctx context.Context, id string, resourceType types.DiscoveryResourceType) error {
    return c.CacheResource(ctx, id, string(resourceType))
}

// FindGenericResource finds resource providers via KNIRVGATEWAY Unix socket (direct DHT query)
func (c *DiscoveryClient) FindGenericResource(ctx context.Context, id string, resourceType types.DiscoveryResourceType) ([]peer.AddrInfo, error) {
    url := fmt.Sprintf("http://localhost/dht/find?id=%s&type=%s", id, string(resourceType))
    
    resp, err := c.httpClient.Get(url)
    if err != nil {
        return nil, fmt.Errorf("DHT find failed: %w", err)
    }
    defer resp.Body.Close()
    
    var peers []peer.AddrInfo
    if err := json.NewDecoder(resp.Body).Decode(&peers); err != nil {
        return nil, err
    }
    
    return peers, nil
}

// FindMCPCapabilityProviders finds MCP capability providers
func (c *DiscoveryClient) FindMCPCapabilityProviders(ctx context.Context, id string, mcpTypeString string) ([]peer.AddrInfo, error) {
    resourceType := fmt.Sprintf("mcp:%s", mcpTypeString)
    return c.FindGenericResource(ctx, id, types.DiscoveryResourceType(resourceType))
}
```

**3.3 Update HTTP endpoints**

- File: `blockchain_server.go`
- Remove: `/internal/dht/findResource`, `/internal/dht/announceResource`
- These functions now call KNIRVGATEWAY's DHT API via Unix socket client (cache for broadcast)

**3.4 Update `main.go` and `p2p_consensus.go`**

- Remove DHT initialization
- Replace with DHT client calls to KNIRVGATEWAY (resources cached for broadcast):

```go
// In main.go or p2p_consensus.go
gatewaySocketPath := filepath.Join(appDataDir, "sockets", "gateway.sock")
dhtClient := dht.NewDiscoveryClient(gatewaySocketPath)
// Resources are now cached and announced via KNIRVGATEWAY broadcast system

// Example: Announce chain resource
err := dhtClient.AnnounceGenericResource(ctx, chainID, types.DiscoveryResourceTypeChain)
```

**3.5 Background Re-announcement in KNIRVCHAIN**

Remove the periodic re-announcement ticker from KNIRVCHAIN - this is now handled by KNIRVGATEWAY's background worker (see Phase 1.6).

---

### Phase 4: Update KNIRVSERVER (Minimal Changes)

**4.1 Ensure KNIRVGATEWAY DHT config passes through**

File: `packages/KNIRVSERVER/backend/cmd/backend_server/main.go`

```go
gatewayConfig := &knirvgateway.ManagerConfig{
    SocketPath:      cfg.Gateway.SocketPath, // "gateway.sock"
    DHTEnabled:      true, // Enable DHT in KNIRVGATEWAY
    DHTPort:         cfg.P2P.DHTPort,
    BootstrapPeers:  cfg.P2P.BootstrapPeers,
    // ... other config
}
```

**4.2 KNIRVSERVER's own DHT (optional removal)**

- KNIRVSERVER's DHT is already disabled by default (`cfg.P2P.DHTEnabled = false`)
- Consider removing `dve_p2p_manager.go` DHT code if redundant (low priority)

---

## Unix Socket Communication Pattern

### Socket Path Resolution

All services use XDG-compliant path: `~/.local/share/knirvserver/sockets/gateway.sock`

```
~/.local/share/knirvserver/sockets/
├── backend.sock        # KNIRVSERVER API
├── gateway.sock        # KNIRVGATEWAY DHT API + Resource Broadcast Cache
├── chain.sock          # KNIRVCHAIN HTTP API
├── chain-p2p.sock     # KNIRVCHAIN P2P
├── graph.sock          # KNIRVGRAPH RPC
├── graph-p2p.sock     # KNIRVGRAPH P2P
├── oracle.sock         # KNIRVORACLE
└── hasher.sock         # KNIRVHASHER gRPC
```

### Resource Broadcast Flow (NEW)

```
KNIRVGRAPH ──(CacheResource)──> KNIRVGATEWAY ResourceCache
                                         ↓
                              (background worker announces)
                                         ↓
KNIRVCHAIN ──(CacheResource)──> KNIRVGATEWAY ResourceCache ──(DHT Provide)──> Kademlia Network
                                         ↓
                                    All cached resources announced together
```

### Client Pattern (used by KNIRVCHAIN and KNIRVGRAPH)

```go
// Create HTTP client with Unix socket transport
client := &http.Client{
    Transport: &http.Transport{
        DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
            return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
        },
    },
}

// Cache resource for broadcast (NEW - replaces direct DHT Provide)
resp, err := client.Post(
    "http://localhost/dht/cache-resource",
    "application/json",
    bytes.NewBuffer(json.Marshal(map[string]interface{}{
        "id":     "resource-id",
        "type":   "skill",
        "source": "knirvgraph",
    })),
)

// Direct DHT query (unchanged)
resp, err := client.Get("http://localhost/dht/find?id=xxx&type=chain")
```

### Server Pattern (KNIRVGATEWAY - already implemented)

From `packages/KNIRVGATEWAY/internal/server/server.go`:
```go
// Listen on Unix socket only (no TCP for DHT API)
listener, err := net.Listen("unix", socketPath)
os.Chmod(socketPath, 0666) // World read/write

httpServer := &http.Server{Handler: router}
httpServer.Serve(listener)
```

### Integration with Existing KNIRVSERVER Pattern

KNIRVSERVER already manages KNIRVGATEWAY via `gateway.sock` (see `packages/KNIRVSERVER/backend/cmd/backend_server/main.go:804`):
```go
// KNIRVSERVER config passes DHT config to KNIRVGATEWAY
gatewayConfig := &knirvgateway.ManagerConfig{
    SocketPath: cfg.Gateway.SocketPath, // "gateway.sock"
    DHTEnabled: true,
    // ... DHT config passed through
}
```

KNIRVGRAPH and KNIRVCHAIN use the same `gateway.sock` path via KNIRVSERVER's config:
- `cfg.Graph.SocketPath` → KNIRVGRAPH DHT client
- `cfg.Chain.SocketPath` → KNIRVCHAIN DHT client
- Both point to `gateway.sock` for DHT API access

---

## API Contract (KNIRVGATEWAY DHT Endpoints over Unix Socket)

### Cache Resource (NEW - Resource Broadcast System)
```
POST http://localhost/dht/cache-resource
Content-Type: application/json
Body: { 
    "id": "...", 
    "type": "...", 
    "multiaddress": "...",  // optional
    "source": "knirvgraph" | "knirvchain"
}
Response: 200 OK
```
**Note:** This adds the resource to KNIRVGATEWAY's cache. A background worker will announce all cached resources to the DHT periodically.

### Announce All Cached Resources (NEW - Manual Trigger)
```
POST http://localhost/dht/announce-all-cached
Response: 200 OK with { "announced": <count> }
```

### Get Cache Status (NEW)
```
GET http://localhost/dht/cache-status
Response: { 
    "count": <cached_resources>,
    "resources": [...]
}
```

### Announce Resource (Direct DHT - Legacy)
```
POST http://localhost/dht/announce
Content-Type: application/json
Body: { "id": "...", "type": "...", "multiaddress": "..." }
Response: 200 OK
```

### Find Resource
```
GET http://localhost/dht/find?id=<id>&type=<type>
Response: [{ "ID": "...", "Addrs": ["..."] }]
```

### Get Peers
```
GET http://localhost/dht/peers
Response: [{ "ID": "...", "Addrs": ["..."] }]
```

### Bootstrap
```
POST http://localhost/dht/bootstrap
Body: { "peers": ["..."] }
Response: 200 OK
```

---

## File Change Summary

| Package | Action | Files |
|---------|--------|-------|
| **KNIRVGATEWAY** | **Create** | `internal/dht/manager.go`, `internal/dht/client.go`, `internal/dht/types.go`, `internal/dht/cache.go` (Resource Broadcast Cache) |
| **KNIRVGATEWAY** | **Modify** | `internal/server/server.go` (add cache endpoints), `internal/config/config.go` |
| **KNIRVGRAPH** | **Delete** | `internal/p2p/dht_manager.go`, `internal/p2p/dht_manager_*.go` |
| **KNIRVGRAPH** | **Create** | `internal/dht/client.go` (Unix socket client with `CacheResource()`) |
| **KNIRVGRAPH** | **Modify** | `internal/app/app.go`, `internal/nrv/nrv_system.go` (use `CacheResource()` instead of direct DHT) |
| **KNIRVCHAIN** | **Delete** | `internal/p2p/discovery_manager.go` |
| **KNIRVCHAIN** | **Create** | `internal/dht/client.go` (Unix socket client with `CacheResource()`) |
| **KNIRVCHAIN** | **Modify** | `blockchain_server.go`, `main.go`, `p2p_consensus.go` (use `CacheResource()` instead of direct DHT) |
| **KNIRVSERVER** | **Modify** | `backend/cmd/backend_server/main.go` (pass DHT config to KNIRVGATEWAY) |

### Resource Broadcast System Files

| File | Purpose |
|------|---------|
| `packages/KNIRVGATEWAY/internal/dht/cache.go` | ResourceCache implementation - stores resources from KNIRVGRAPH/KNIRVCHAIN |
| `packages/KNIRVGATEWAY/internal/dht/manager.go` | Background worker `startAnnouncementWorker()` - announces all cached resources to DHT |
| `packages/KNIRVGRAPH/internal/dht/client.go` | `CacheResource()` - sends resources to KNIRVGATEWAY cache |
| `packages/KNIRVCHAIN/internal/dht/client.go` | `CacheResource()` - sends resources to KNIRVGATEWAY cache |

---

## Testing Strategy

### 1. Unit Tests for KNIRVGATEWAY DHT + Broadcast System

- Port existing tests from KNIRVGRAPH/KNIRVCHAIN
- Create `internal/dht/manager_test.go`
- Test DHT operations over local TCP (P2P) and Unix socket (API)
- **NEW:** Test `ResourceCache` operations (`AddResource`, `GetAllResources`, `GetResourceCount`)
- **NEW:** Test background announcement worker

### 2. Unix Socket Integration Tests

```go
// Test KNIRVCHAIN → KNIRVGATEWAY via Unix socket (with broadcast system)
func TestResourceBroadcast(t *testing.T) {
    // Start KNIRVGATEWAY with DHT on Unix socket
    // Create KNIRVCHAIN DHT client with socket path
    // Test CacheResource() - adds to KNIRVGATEWAY cache
    // Verify cache status endpoint returns the resource
    // Trigger announce-all-cached
    // Verify resource is announced to DHT
}
```

### 3. Migration Validation

- Run `make testnet-tests`
- Verify no DHT initialization in KNIRVGRAPH/KNIRVCHAIN startup logs
- Verify all DHT operations go through `gateway.sock`
- **NEW:** Verify resources are cached in KNIRVGATEWAY (check `/dht/cache-status`)
- **NEW:** Verify background worker announces cached resources to DHT
- Use `ss -x` to verify Unix socket connections

---

## Risks & Mitigations

| Risk | Mitigation |
|------|-------------|
| DHT feature parity loss | Port ALL functions from KNIRVGRAPH/KNIRVCHAIN before deleting |
| Network disruption during migration | Phase rollout with feature flags (`DisableDHT`) |
| Unix socket permissions | Set `os.Chmod(socketPath, 0666)` for world read/write |
| libp2p doesn't support Unix sockets for P2P | Only API uses Unix socket; P2P still uses TCP (expected) |
| NRV system breakage | Keep NRV types, only replace DHT calls with `CacheResource()` |
| KNIRVGATEWAY SPoF | KNIRVGATEWAY already runs embedded in KNIRVSERVER (same process) |
| Resource broadcast delays | Background worker interval configurable (default 5 min); manual trigger via `/dht/announce-all-cached` |
| Cache memory growth | ResourceCache uses map with timestamps; consider TTL/expiration for stale entries |

---

## Success Criteria

1. ✅ No DHT code in `packages/KNIRVGRAPH/internal/p2p/` (except non-DHT files)
2. ✅ No DHT code in `packages/KNIRVCHAIN/internal/p2p/`
3. ✅ KNIRVGATEWAY has working DHT with all former KNIRVGRAPH/KNIRVCHAIN features
4. ✅ All KNIRVCHAIN/KNIRVGRAPH → KNIRVGATEWAY DHT calls use Unix socket (`gateway.sock`)
5. ✅ No TCP loopback connections for DHT API (use `ss -x | grep gateway.sock` to verify)
6. ✅ `make testnet-tests` pass
7. ✅ KNIRVSERVER can operate KNIRVGATEWAY with full DHT functionality
8. ✅ **NEW:** KNIRVGRAPH and KNIRVCHAIN use `CacheResource()` to add resources to KNIRVGATEWAY broadcast cache
9. ✅ **NEW:** KNIRVGATEWAY maintains a `ResourceCache` with all resources from both services
10. ✅ **NEW:** KNIRVGATEWAY background worker announces all cached resources to DHT periodically
11. ✅ **NEW:** `/dht/cache-resource` endpoint accepts resources from KNIRVGRAPH and KNIRVCHAIN
12. ✅ **NEW:** `/dht/cache-status` shows all cached resources with timestamps and sources
13. ✅ **NEW:** `/dht/announce-all-cached` triggers immediate announcement of all cached resources

---

## Notes on "Unix WebSockets"

The request mentioned "Unix WebSockets" - clarification:
- **Unix sockets**: Used for local IPC between services (KNIRVCHAIN → KNIRVGATEWAY)
- **WebSockets (ws://)**: Used in JavaScript DHT implementation (`lib/p2p/dht_manager.js`) for P2P communication
- **libp2p Go limitation**: Does NOT support Unix sockets for P2P transport (only TCP/WebSocket/QUIC)

This plan uses:
- **Unix sockets** for service-to-service API (KNIRVCHAIN/KNIRVGRAPH ↔ KNIRVGATEWAY)
- **TCP** for DHT P2P network (libp2p Kademlia DHT)

If "Unix WebSockets" meant WebSocket over Unix socket, that's not currently supported by Go's `net/http` package (would require custom `websocket.Dialer` with Unix socket).

---

**End of Plan**
