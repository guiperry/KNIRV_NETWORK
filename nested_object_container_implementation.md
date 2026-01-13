# Nested Object Container Architecture

**Version:** 2.0
**Status:** Design & Implementation Document
**Last Updated:** 2026-01-12

## Executive Summary

Nested Object Container (NOC) is the KNIRV Network's universal containerized object deployment system that runs **any servable object** inside KNIRVNEXUS DVE containers. NOCs support diverse content types including web applications, blockchain nodes, model servers, 3D GLB objects, and more. Each NOC is accessible via cryptographic hash-based URLs routed through KNIRVROUTER, with browser-based viewport functionality for real-time container inspection.

Every KNIRVNEXUS deployment automatically provisions **two demo NOCs** on startup:
1. **KNIRVGATEWAY NOC**: Full-stack gateway with embedded oracle daemon and model server
2. **KNIRVROUTER NOC**: P2P network node providing DHT and routing services

## Architecture Overview

### Core Concepts

**Nested Object Container (NOC)** is a general-purpose container deployment architecture that can host:
- **Web Services**: APIs, documentation sites, admin panels
- **Blockchain Nodes**: Full nodes, validators, oracles
- **Model Servers**: LLM inference, embeddings, LoRA adapters
- **3D Assets**: GLB/GLTF models with viewport rendering
- **P2P Services**: Routers, DHT nodes, TURN/STUN servers
- **File Servers**: IPFS nodes, static content delivery
- **Any Containerizable Service**: Docker/Kata/Native-Go compatible

### Universal Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                       KNIRVNEXUS Host                           │
│                                                                 │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │           Nested Object Container (Universal)            │  │
│  │                                                          │  │
│  │  ┌────────────────────────────────────────────────────┐  │  │
│  │  │         Servable Object (Any Type)                 │  │  │
│  │  │                                                    │  │  │
│  │  │  - Web Application / API Service                   │  │  │
│  │  │  - Blockchain Node / Oracle Daemon                 │  │  │
│  │  │  - Model Server / Inference Engine                 │  │  │
│  │  │  - 3D GLB Object + Renderer                        │  │  │
│  │  │  - P2P Router / DHT Node                           │  │  │
│  │  │  - File Server / IPFS Node                         │  │  │
│  │  │  - Custom Containerized Service                    │  │  │
│  │  └────────────────────────────────────────────────────┘  │  │
│  │                                                          │  │
│  │  ┌────────────────────────────────────────────────────┐  │  │
│  │  │         Viewport Manager                           │  │  │
│  │  │  - HTTP Proxy (Service Access)                     │  │  │
│  │  │  - WebRTC Streaming (Live Video/Audio)            │  │  │
│  │  │  - WebGL Renderer (3D GLB Objects)                │  │  │
│  │  │  - VNC/Terminal (Shell Access)                    │  │  │
│  │  └────────────────────────────────────────────────────┘  │  │
│  │                                                           │  │
│  │  Cryptographic Hash: blake3(container_id +                │  │
│  │                              node_pubkey +                │  │
│  │                              genesis_time +               │  │
│  │                              object_type)                 │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
│  Container Orchestrator (Unified DVE/TEE/Container)            │
│  eBPF Security Layer (Syscall monitoring, LSM hooks)           │
│  Virtual Container Manager                                     │
│  3D Asset Registry (GLB/GLTF metadata + rendering)             │
└─────────────────────────────────────────────────────────────────┘
                           │
                           │ P2P libp2p + DHT
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│                    KNIRVROUTER Network                          │
│                                                                 │
│  DHT-based Content Routing (Hash → Multiaddr)                  │
│  PubSub Message Broadcasting                                   │
│  TURN/STUN NAT Traversal                                       │
│  Content-Type Negotiation                                      │
└─────────────────────────────────────────────────────────────────┘
                           │
                           │ Browser Access
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│                    User Browser                                 │
│                                                                 │
│  URL: knirv://<blake3-hash>.object.nest                        │
│  Viewport: Live container view with content-aware rendering    │
│  - Web apps: HTTP proxy + iframe                              │
│  - 3D objects: WebGL renderer (three.js)                       │
│  - Blockchain nodes: API + metrics dashboard                   │
│  - Terminals: XTerm.js + WebSocket                            │
└─────────────────────────────────────────────────────────────────┘
```

## Container Convergence: Unified DVE/TEE/Container System

### Current State

The KNIRVNEXUS backend currently has three overlapping concepts:

1. **DVE (Distributed Validation Environment)**: Logical validation units
2. **TEE (Trusted Execution Environment)**: Security boundary enforcement
3. **Container**: Physical runtime isolation

### Unified Architecture

```go
// pkg/runtime/unified_container.go

package runtime

import (
    "context"
    "github.com/KNIRV/KNIRV_NETWORK/KNIRVNEXUS/backend/internal/services/teesecurity"
    "github.com/KNIRV/KNIRV_NETWORK/KNIRVNEXUS/backend/internal/ebpf"
)

// RuntimeMode defines the execution mode for a container
type RuntimeMode string

const (
    RuntimeModeDVE      RuntimeMode = "dve"       // Standard validation environment
    RuntimeModeTEE      RuntimeMode = "tee"       // Hardware TEE (SGX/SEV/TDX)
    RuntimeModeKata     RuntimeMode = "kata"      // Kata containers (VM-based)
    RuntimeModeDocker   RuntimeMode = "docker"    // Docker containers
    RuntimeModeNative   RuntimeMode = "native-go" // Native Go isolation (Kali)
    RuntimeModeObject   RuntimeMode = "object"    // NOC mode (universal)
)

// ObjectType defines the type of servable object
type ObjectType string

const (
    ObjectTypeWebApp      ObjectType = "webapp"       // Web application
    ObjectTypeAPI         ObjectType = "api"          // API service
    ObjectTypeBlockchain  ObjectType = "blockchain"   // Blockchain node
    ObjectTypeOracle      ObjectType = "oracle"       // Oracle daemon
    ObjectTypeModel       ObjectType = "model"        // Model server
    ObjectType3D          ObjectType = "3d"           // 3D GLB/GLTF object
    ObjectTypeP2P         ObjectType = "p2p"          // P2P router/node
    ObjectTypeFile        ObjectType = "file"         // File server
    ObjectTypeCustom      ObjectType = "custom"       // Custom service
)

// SecurityLevel defines the isolation strength
type SecurityLevel string

const (
    SecurityLevelBasic    SecurityLevel = "basic"    // Namespace isolation
    SecurityLevelStrong   SecurityLevel = "strong"   // + seccomp + AppArmor/SELinux
    SecurityLevelExtreme  SecurityLevel = "extreme"  // + hardware TEE
)

// UnifiedContainer represents a single execution unit with DVE/TEE/Container capabilities
type UnifiedContainer struct {
    ID              string                    `json:"id"`
    Mode            RuntimeMode               `json:"mode"`
    ObjectType      ObjectType                `json:"object_type"`
    SecurityLevel   SecurityLevel             `json:"security_level"`
    Spec            *ContainerSpec            `json:"spec"`

    // DVE capabilities
    ValidationRole  string                    `json:"validation_role,omitempty"`
    P2PEndpoints    []string                  `json:"p2p_endpoints,omitempty"`

    // TEE capabilities
    TEEAttester     *teesecurity.TEEAttester  `json:"tee_attester,omitempty"`
    Attestation     *teesecurity.Attestation  `json:"attestation,omitempty"`

    // Container runtime
    Runtime         ContainerRuntime          `json:"runtime"`
    PID             int                       `json:"pid,omitempty"`
    Namespace       *LinuxNamespace           `json:"namespace,omitempty"`

    // eBPF security
    eBPFMonitor     *ebpf.SyscallMonitor      `json:"-"`
    SandboxLSM      *ebpf.SandboxLSM          `json:"-"`

    // NOC specific
    ObjectConfig    *NestedObjectConfig         `json:"object_config,omitempty"`
    ViewportProxy   *ViewportProxy            `json:"-"`
    CryptoHash      string                    `json:"crypto_hash,omitempty"`

    // 3D Asset support
    GLBRenderer     *GLBRenderer              `json:"-"`
    AssetMetadata   *AssetMetadata            `json:"asset_metadata,omitempty"`

    // Model server integration (for model-type objects)
    ModelServer     *EmbeddedModelServer      `json:"-"`

    // Lifecycle
    CreatedAt       time.Time                 `json:"created_at"`
    Status          ContainerStatus           `json:"status"`
}

// ContainerSpec defines the container specification
type ContainerSpec struct {
    ID              string                    `json:"id"`
    Image           string                    `json:"image"`
    Command         []string                  `json:"command,omitempty"`
    Environment     map[string]string         `json:"environment,omitempty"`
    Volumes         []VolumeMount             `json:"volumes,omitempty"`
    Ports           []PortMapping             `json:"ports,omitempty"`
    Resources       ResourceLimits            `json:"resources"`
    NetworkMode     string                    `json:"network_mode"`
    CapabilitySet   []string                  `json:"capabilities,omitempty"`
}

// NestedObjectConfig defines NOC-specific configuration
type NestedObjectConfig struct {
    ObjectType         ObjectType             `json:"object_type"`
    EnableViewport     bool                   `json:"enable_viewport"`
    ViewportRenderers  []string               `json:"viewport_renderers"` // http, webrtc, webgl, vnc
    ServicePorts       map[string]int         `json:"service_ports"`
    Metadata           map[string]interface{} `json:"metadata"`
}

// AssetMetadata defines metadata for 3D assets
type AssetMetadata struct {
    Format          string                    `json:"format"`           // glb, gltf, obj, etc.
    FileSize        int64                     `json:"file_size"`
    Dimensions      *Dimensions3D             `json:"dimensions"`
    Materials       []string                  `json:"materials"`
    Animations      []string                  `json:"animations"`
    Textures        []TextureInfo             `json:"textures"`
    Polycount       int                       `json:"polycount"`
    BoundingBox     *BoundingBox3D            `json:"bounding_box"`
}

// Dimensions3D represents 3D object dimensions
type Dimensions3D struct {
    Width   float64 `json:"width"`
    Height  float64 `json:"height"`
    Depth   float64 `json:"depth"`
}

// BoundingBox3D represents a 3D bounding box
type BoundingBox3D struct {
    Min Vector3D `json:"min"`
    Max Vector3D `json:"max"`
}

// Vector3D represents a 3D vector
type Vector3D struct {
    X float64 `json:"x"`
    Y float64 `json:"y"`
    Z float64 `json:"z"`
}

// TextureInfo represents texture metadata
type TextureInfo struct {
    Name        string `json:"name"`
    Type        string `json:"type"`
    Resolution  string `json:"resolution"`
    Format      string `json:"format"`
}

// UnifiedContainerManager manages all container types
type UnifiedContainerManager struct {
    teeService      *teesecurity.TEESecurityService
    ebpfManager     ebpf.ManagerInterface
    vcManager       *ebpf.VirtualContainerManager
    runtimeSelector *RuntimeSelector
    assetRegistry   *AssetRegistry
    containers      map[string]*UnifiedContainer
    mu              sync.RWMutex
}

// CreateContainer creates a new unified container with the specified mode
func (ucm *UnifiedContainerManager) CreateContainer(
    ctx context.Context,
    spec *ContainerSpec,
    mode RuntimeMode,
    objectType ObjectType,
    securityLevel SecurityLevel,
) (*UnifiedContainer, error) {
    // Select appropriate runtime based on mode and host capabilities
    runtime, err := ucm.runtimeSelector.SelectRuntime(mode, securityLevel)
    if err != nil {
        return nil, fmt.Errorf("runtime selection failed: %w", err)
    }

    container := &UnifiedContainer{
        ID:            generateContainerID(),
        Mode:          mode,
        ObjectType:    objectType,
        SecurityLevel: securityLevel,
        Spec:          spec,
        Runtime:       runtime,
        CreatedAt:     time.Now(),
        Status:        ContainerStatusCreating,
    }

    // Initialize eBPF security if available
    if ucm.ebpfManager != nil {
        container.eBPFMonitor = ebpf.NewSyscallMonitor(ucm.ebpfManager)
        container.SandboxLSM = ebpf.NewSandboxLSM(ucm.ebpfManager)
    }

    // Initialize TEE attestation if requested
    if securityLevel == SecurityLevelExtreme {
        attester := teesecurity.NewTEEAttester(ucm.teeService)
        attestation, err := attester.GenerateAttestation(ctx)
        if err != nil {
            log.Printf("Warning: TEE attestation failed: %v", err)
        } else {
            container.TEEAttester = attester
            container.Attestation = attestation
        }
    }

    // Initialize 3D renderer if object is 3D type
    if objectType == ObjectType3D {
        glbRenderer := NewGLBRenderer()
        container.GLBRenderer = glbRenderer
    }

    // Create container via runtime
    if err := runtime.Create(ctx, container); err != nil {
        return nil, fmt.Errorf("container creation failed: %w", err)
    }

    // Generate cryptographic hash for routing
    container.CryptoHash = ucm.generateCryptoHash(container)

    // Register with KNIRVROUTER DHT
    if err := ucm.registerWithRouter(ctx, container); err != nil {
        log.Printf("Warning: Router registration failed: %v", err)
    }

    ucm.mu.Lock()
    ucm.containers[container.ID] = container
    ucm.mu.Unlock()

    return container, nil
}

// CreateNestedObject creates a new NOC container (universal method)
func (ucm *UnifiedContainerManager) CreateNestedObject(
    ctx context.Context,
    config *NestedObjectConfig,
) (*UnifiedContainer, error) {
    // Build container spec based on object type
    spec := ucm.buildSpecForObjectType(config)

    container, err := ucm.CreateContainer(
        ctx,
        spec,
        RuntimeModeObject,
        config.ObjectType,
        SecurityLevelStrong,
    )
    if err != nil {
        return nil, err
    }

    container.ObjectConfig = config

    // Initialize viewport proxy if enabled
    if config.EnableViewport {
        viewportProxy := NewViewportProxy(container, config.ViewportRenderers)
        if err := viewportProxy.Start(ctx); err != nil {
            log.Printf("Warning: Viewport proxy failed: %v", err)
        } else {
            container.ViewportProxy = viewportProxy
        }
    }

    return container, nil
}

// buildSpecForObjectType builds container spec based on object type
func (ucm *UnifiedContainerManager) buildSpecForObjectType(config *NestedObjectConfig) *ContainerSpec {
    spec := &ContainerSpec{
        NetworkMode: "bridge",
        Resources: ResourceLimits{
            CPUCores:  4,
            MemoryMB:  8192,
            DiskMB:    20480,
        },
    }

    // Configure based on object type
    switch config.ObjectType {
    case ObjectType3D:
        spec.Image = "glb-renderer:latest"
        spec.Command = []string{"/usr/local/bin/renderer", "--serve"}
        spec.Environment = map[string]string{
            "RENDERER_MODE": "webgl",
            "ENABLE_SHADOWS": "true",
            "ENABLE_PBR":    "true",
        }
        spec.Ports = []PortMapping{
            {ContainerPort: 8080, HostPort: 0},
        }

    case ObjectTypeAPI, ObjectTypeWebApp:
        // Generic web service container
        spec.Image = "alpine:latest"
        spec.Ports = []PortMapping{
            {ContainerPort: 8080, HostPort: 0},
        }

    default:
        // Custom object type - use provided metadata
        if config.Metadata != nil {
            if image, ok := config.Metadata["image"].(string); ok {
                spec.Image = image
            }
            if cmd, ok := config.Metadata["command"].([]string); ok {
                spec.Command = cmd
            }
        }
    }

    return spec
}

// generateCryptoHash generates a cryptographic hash for container routing
func (ucm *UnifiedContainerManager) generateCryptoHash(c *UnifiedContainer) string {
    // Use BLAKE3 for cryptographic hash derivation
    hasher := blake3.New()

    // Hash components:
    // 1. Container ID
    // 2. Node public key (from P2P identity)
    // 3. Genesis timestamp
    // 4. Container creation time
    // 5. Object type (NEW: allows type-based routing)

    hasher.Write([]byte(c.ID))
    hasher.Write(ucm.runtimeSelector.nodePublicKey)
    hasher.Write([]byte(ucm.runtimeSelector.genesisTime.Format(time.RFC3339)))
    hasher.Write([]byte(c.CreatedAt.Format(time.RFC3339)))
    hasher.Write([]byte(c.ObjectType))

    return hex.EncodeToString(hasher.Sum(nil))
}

// registerWithRouter registers the container with KNIRVROUTER DHT
func (ucm *UnifiedContainerManager) registerWithRouter(
    ctx context.Context,
    c *UnifiedContainer,
) error {
    // Connect to local KNIRVROUTER instance
    routerClient, err := router.NewClient(":5001")
    if err != nil {
        return fmt.Errorf("router client creation failed: %w", err)
    }

    // Publish container multiaddr to DHT with object type prefix
    multiaddr := fmt.Sprintf("/dns4/%s/tcp/%d/p2p/%s/object/%s/%s",
        ucm.runtimeSelector.nodeHostname,
        c.Spec.Ports[0].HostPort,
        ucm.runtimeSelector.nodePeerID,
        c.ObjectType,
        c.CryptoHash,
    )

    if err := routerClient.Provide(ctx, c.CryptoHash, multiaddr); err != nil {
        return fmt.Errorf("DHT provide failed: %w", err)
    }

    return nil
}
```

## Cryptographic Routing via KNIRVROUTER

### Hash-Based URL Resolution with Object Type Awareness

```go
// pkg/routing/crypto_router.go

package routing

import (
    "context"
    "fmt"
    dht "github.com/libp2p/go-libp2p-kad-dht"
    "github.com/libp2p/go-libp2p/core/peer"
    ma "github.com/multiformats/go-multiaddr"
)

// CryptoRouter resolves cryptographic hashes to container multiaddrs
type CryptoRouter struct {
    dht         *dht.IpfsDHT
    peerID      peer.ID
    localCache  *HashCache
}

// ResolveNestedObject resolves an NOC hash to its multiaddr
func (cr *CryptoRouter) ResolveNestedObject(
    ctx context.Context,
    hash string,
    objectType string, // Optional: filter by object type
) (ma.Multiaddr, ObjectType, error) {
    // Check local cache first
    if cached, found := cr.localCache.Get(hash); found {
        return cached.Multiaddr, cached.ObjectType, nil
    }

    // Query DHT for hash → multiaddr mapping
    peerChan := cr.dht.FindProvidersAsync(ctx, hashToCID(hash), 10)

    select {
    case peerInfo := <-peerChan:
        if peerInfo.ID == "" {
            return nil, "", fmt.Errorf("no providers found for hash: %s", hash)
        }

        // Extract multiaddr from peer info
        if len(peerInfo.Addrs) == 0 {
            return nil, "", fmt.Errorf("no addresses for peer: %s", peerInfo.ID)
        }

        // Parse object type from multiaddr
        parsedType := parseObjectTypeFromMultiaddr(peerInfo.Addrs[0])

        // Filter by object type if specified
        if objectType != "" && parsedType != ObjectType(objectType) {
            return nil, "", fmt.Errorf("object type mismatch: expected %s, got %s", objectType, parsedType)
        }

        // Cache the result
        cr.localCache.Set(hash, CacheEntry{
            Multiaddr:  peerInfo.Addrs[0],
            ObjectType: parsedType,
        })

        return peerInfo.Addrs[0], parsedType, nil

    case <-ctx.Done():
        return nil, "", ctx.Err()
    }
}

// PublishNestedObject publishes an NOC to the DHT
func (cr *CryptoRouter) PublishNestedObject(
    ctx context.Context,
    hash string,
    objectType ObjectType,
    multiaddr ma.Multiaddr,
) error {
    // Announce to DHT that we provide this hash
    if err := cr.dht.Provide(ctx, hashToCID(hash), true); err != nil {
        return fmt.Errorf("DHT provide failed: %w", err)
    }

    // Update local peer store
    cr.dht.Host().Peerstore().AddAddrs(
        cr.peerID,
        []ma.Multiaddr{multiaddr},
        time.Hour*24,
    )

    return nil
}

// SubscribeToObjectUpdates subscribes to NOC update events via pubsub
func (cr *CryptoRouter) SubscribeToObjectUpdates(
    ctx context.Context,
) (<-chan ObjectUpdate, error) {
    // Use libp2p pubsub for real-time updates
    sub, err := cr.pubsub.Subscribe("object-nest-updates")
    if err != nil {
        return nil, err
    }

    updateChan := make(chan ObjectUpdate, 100)

    go func() {
        defer close(updateChan)
        for {
            msg, err := sub.Next(ctx)
            if err != nil {
                if ctx.Err() != nil {
                    return
                }
                log.Printf("Pubsub error: %v", err)
                continue
            }

            var update ObjectUpdate
            if err := json.Unmarshal(msg.Data, &update); err != nil {
                log.Printf("Invalid update message: %v", err)
                continue
            }

            select {
            case updateChan <- update:
            case <-ctx.Done():
                return
            }
        }
    }()

    return updateChan, nil
}

// ObjectUpdate represents an NOC status update
type ObjectUpdate struct {
    Hash        string     `json:"hash"`
    ObjectType  ObjectType `json:"object_type"`
    Status      string     `json:"status"`
    Multiaddr   string     `json:"multiaddr"`
    Timestamp   time.Time  `json:"timestamp"`
}
```

### Browser URL Scheme Handler

```javascript
// knirvrouter/wasm_integration/object_resolver.js

/**
 * NOC URL resolver for browser integration
 *
 * URL Format: knirv://<blake3-hash>.object.nest[?type=<object-type>]
 *
 * Resolution Flow:
 * 1. Extract BLAKE3 hash from URL
 * 2. Query local KNIRVROUTER instance via WebAssembly
 * 3. Resolve hash to multiaddr via DHT
 * 4. Connect to container via HTTP proxy
 * 5. Initialize appropriate renderer based on object type
 */

class NestedObjectResolver {
    constructor(routerWASM) {
        this.router = routerWASM;
        this.cache = new Map();
        this.renderers = {
            'webapp': WebAppRenderer,
            'api': APIRenderer,
            '3d': GLBRenderer,
            'blockchain': BlockchainDashboard,
            'p2p': P2PMonitor,
            'custom': IFrameRenderer
        };
    }

    /**
     * Resolve NOC URL to HTTP endpoint with object type
     * @param {string} objectURL - knirv://<hash>.object.nest
     * @returns {Promise<{httpURL: string, objectType: string}>}
     */
    async resolve(objectURL) {
        const hash = this.extractHash(objectURL);
        const typeHint = this.extractTypeHint(objectURL);

        // Check cache
        const cacheKey = `${hash}:${typeHint || ''}`;
        if (this.cache.has(cacheKey)) {
            return this.cache.get(cacheKey);
        }

        // Query KNIRVROUTER WASM module
        const result = await this.router.resolveHash(hash, typeHint);
        if (!result) {
            throw new Error(`Failed to resolve NOC: ${hash}`);
        }

        // Parse multiaddr to HTTP URL and extract object type
        const httpURL = this.multiaddrToHTTP(result.multiaddr);
        const objectType = result.objectType || 'custom';

        // Cache result
        const resolved = { httpURL, objectType };
        this.cache.set(cacheKey, resolved);

        return resolved;
    }

    /**
     * Extract BLAKE3 hash from NOC URL
     */
    extractHash(url) {
        const match = url.match(/^knirv:\/\/([a-f0-9]{64})\.object\.nest/);
        if (!match) {
            throw new Error(`Invalid NOC URL: ${url}`);
        }
        return match[1];
    }

    /**
     * Extract object type hint from query parameters
     */
    extractTypeHint(url) {
        const urlObj = new URL(url);
        return urlObj.searchParams.get('type');
    }

    /**
     * Convert libp2p multiaddr to HTTP URL
     */
    multiaddrToHTTP(multiaddr) {
        // Parse: /dns4/hostname/tcp/port/p2p/peerID/object/<type>/hash
        const parts = multiaddr.split('/');
        const hostname = parts[2];
        const port = parts[4];

        return `http://${hostname}:${port}`;
    }

    /**
     * Get appropriate renderer for object type
     */
    getRenderer(objectType) {
        return this.renderers[objectType] || this.renderers['custom'];
    }
}

// Register custom protocol handler
if ('registerProtocolHandler' in navigator) {
    navigator.registerProtocolHandler(
        'knirv',
        '/object-redirect?url=%s',
        'KNIRV NOC'
    );
}
```

## Viewport Functionality: Content-Aware Rendering

### Universal Viewport Proxy

```go
// pkg/viewport/proxy.go

package viewport

import (
    "context"
    "net/http"
    "net/http/httputil"
    "github.com/gorilla/websocket"
    "github.com/pion/webrtc/v3"
)

// ViewportProxy manages browser access to a container with content-aware rendering
type ViewportProxy struct {
    containerID     string
    objectType      ObjectType
    containerPorts  map[string]int
    httpProxy       *httputil.ReverseProxy
    webrtcPeer      *webrtc.PeerConnection
    vncSession      *VNCSession
    glbRenderer     *GLBRenderer
    upgrader        websocket.Upgrader
    renderers       []string // Enabled renderers: http, webrtc, webgl, vnc
}

// NewViewportProxy creates a new viewport proxy for a container
func NewViewportProxy(container *UnifiedContainer, renderers []string) *ViewportProxy {
    return &ViewportProxy{
        containerID:    container.ID,
        objectType:     container.ObjectType,
        containerPorts: extractPorts(container.Spec.Ports),
        renderers:      renderers,
        upgrader: websocket.Upgrader{
            CheckOrigin: func(r *http.Request) bool { return true },
        },
    }
}

// Start initializes the viewport proxy with content-aware renderers
func (vp *ViewportProxy) Start(ctx context.Context) error {
    // Initialize HTTP reverse proxy
    vp.httpProxy = &httputil.ReverseProxy{
        Director: func(req *http.Request) {
            targetPort := vp.selectPort(req.URL.Path)
            req.URL.Scheme = "http"
            req.URL.Host = fmt.Sprintf("localhost:%d", targetPort)
        },
    }

    // Initialize renderers based on object type
    for _, renderer := range vp.renderers {
        switch renderer {
        case "webrtc":
            if err := vp.initializeWebRTC(); err != nil {
                log.Printf("Warning: WebRTC initialization failed: %v", err)
            }

        case "webgl":
            if vp.objectType == ObjectType3D {
                glbRenderer := NewGLBRenderer()
                vp.glbRenderer = glbRenderer
            }

        case "vnc":
            vncSession, err := NewVNCSession(vp.containerID)
            if err != nil {
                log.Printf("Warning: VNC session creation failed: %v", err)
            } else {
                vp.vncSession = vncSession
            }
        }
    }

    return nil
}

// HandleHTTP handles HTTP requests to the container
func (vp *ViewportProxy) HandleHTTP(w http.ResponseWriter, r *http.Request) {
    // Add CORS headers
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
    w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
    w.Header().Set("X-Object-Type", string(vp.objectType))

    if r.Method == "OPTIONS" {
        w.WriteHeader(http.StatusOK)
        return
    }

    // Route to appropriate handler based on object type
    switch vp.objectType {
    case ObjectType3D:
        vp.handle3DObject(w, r)
    default:
        vp.httpProxy.ServeHTTP(w, r)
    }
}

// handle3DObject handles requests for 3D GLB objects
func (vp *ViewportProxy) handle3DObject(w http.ResponseWriter, r *http.Request) {
    if r.URL.Path == "/asset.glb" {
        // Serve the GLB file directly
        vp.serveGLBAsset(w, r)
    } else {
        // Serve the WebGL viewer interface
        vp.serveGLBViewer(w, r)
    }
}

// serveGLBAsset serves the GLB file with proper headers
func (vp *ViewportProxy) serveGLBAsset(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "model/gltf-binary")
    w.Header().Set("Cache-Control", "public, max-age=31536000")

    // Proxy to container's asset endpoint
    vp.httpProxy.ServeHTTP(w, r)
}

// serveGLBViewer serves the WebGL viewer interface
func (vp *ViewportProxy) serveGLBViewer(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/html")
    w.Write([]byte(glbViewerHTML))
}

// initializeWebRTC initializes WebRTC peer connection
func (vp *ViewportProxy) initializeWebRTC() error {
    config := webrtc.Configuration{
        ICEServers: []webrtc.ICEServer{
            {URLs: []string{"stun:stun.l.google.com:19302"}},
        },
    }

    pc, err := webrtc.NewPeerConnection(config)
    if err != nil {
        return fmt.Errorf("WebRTC peer creation failed: %w", err)
    }
    vp.webrtcPeer = pc

    return nil
}

// selectPort selects the appropriate container port based on request path
func (vp *ViewportProxy) selectPort(path string) int {
    // Default to first port
    for _, port := range vp.containerPorts {
        return port
    }
    return 8080
}

// GLB Viewer HTML template
const glbViewerHTML = `
<!DOCTYPE html>
<html>
<head>
    <title>3D Object Viewer</title>
    <style>
        body { margin: 0; overflow: hidden; }
        #canvas { width: 100vw; height: 100vh; }
        #controls { position: absolute; top: 10px; left: 10px; background: rgba(0,0,0,0.7); color: white; padding: 10px; }
    </style>
</head>
<body>
    <div id="controls">
        <button onclick="resetCamera()">Reset Camera</button>
        <button onclick="toggleWireframe()">Wireframe</button>
        <button onclick="toggleAnimation()">Animation</button>
    </div>
    <canvas id="canvas"></canvas>
    <script src="https://cdn.jsdelivr.net/npm/three@0.150.0/build/three.min.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/three@0.150.0/examples/js/loaders/GLTFLoader.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/three@0.150.0/examples/js/controls/OrbitControls.js"></script>
    <script>
        const scene = new THREE.Scene();
        const camera = new THREE.PerspectiveCamera(75, window.innerWidth / window.innerHeight, 0.1, 1000);
        const renderer = new THREE.WebGLRenderer({ canvas: document.getElementById('canvas'), antialias: true });
        renderer.setSize(window.innerWidth, window.innerHeight);
        renderer.setClearColor(0x222222);

        const controls = new THREE.OrbitControls(camera, renderer.domElement);
        camera.position.z = 5;

        const loader = new THREE.GLTFLoader();
        loader.load('/asset.glb', (gltf) => {
            scene.add(gltf.scene);
            if (gltf.animations.length > 0) {
                mixer = new THREE.AnimationMixer(gltf.scene);
                gltf.animations.forEach((clip) => mixer.clipAction(clip).play());
            }
        });

        const ambientLight = new THREE.AmbientLight(0xffffff, 0.5);
        scene.add(ambientLight);
        const directionalLight = new THREE.DirectionalLight(0xffffff, 0.5);
        directionalLight.position.set(5, 10, 7.5);
        scene.add(directionalLight);

        function animate() {
            requestAnimationFrame(animate);
            controls.update();
            if (mixer) mixer.update(0.016);
            renderer.render(scene, camera);
        }
        animate();

        window.addEventListener('resize', () => {
            camera.aspect = window.innerWidth / window.innerHeight;
            camera.updateProjectionMatrix();
            renderer.setSize(window.innerWidth, window.innerHeight);
        });

        function resetCamera() {
            camera.position.set(0, 0, 5);
            controls.reset();
        }

        function toggleWireframe() {
            scene.traverse((obj) => {
                if (obj.isMesh) obj.material.wireframe = !obj.material.wireframe;
            });
        }

        let mixer;
        function toggleAnimation() {
            if (mixer) mixer.paused = !mixer.paused;
        }
    </script>
</body>
</html>
`
```

## Demo NOC Implementations

### Demo 1: KNIRVGATEWAY NOC

The KNIRVGATEWAY NOC is a full-featured demo that showcases a complete web gateway with embedded oracle daemon and model server capabilities. This demonstrates the NOC architecture's ability to host complex, multi-service applications.

#### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│            KNIRVGATEWAY NOC Container               │
│                                                             │
│  ┌────────────────────────────────────────────────────┐   │
│  │         KNIRVGATEWAY Service                       │   │
│  │  - REST API endpoints                              │   │
│  │  - Documentation UI (Swagger/OpenAPI)             │   │
│  │  - WebSocket streams                               │   │
│  │  - Blockchain integration APIs                     │   │
│  └────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌────────────────────────────────────────────────────┐   │
│  │         knirv-oracled (Embedded Daemon)            │   │
│  │  - Cross-chain transfer coordination               │   │
│  │  - Governance proposal management                  │   │
│  │  - Token economics tracking                        │   │
│  │  - IBC connection management                       │   │
│  └────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌────────────────────────────────────────────────────┐   │
│  │         Embedded Model Server                      │   │
│  │  - LLM inference (Ollama-compatible API)           │   │
│  │  - Embedding generation                            │   │
│  │  - LoRA adapter loading from KNIRVCHAIN           │   │
│  └────────────────────────────────────────────────────┘   │
│                                                             │
│  Ports: 8080 (gateway), 9090 (oracle), 11434 (model)      │
└─────────────────────────────────────────────────────────────┘
```

#### Implementation

```go
// pkg/demos/gateway_nest.go

package demos

import (
    "context"
    "fmt"
    "github.com/KNIRV/KNIRV_NETWORK/KNIRVNEXUS/pkg/runtime"
)

// GatewayNestDeployer deploys KNIRVGATEWAY NOC
type GatewayNestDeployer struct {
    containerManager *runtime.UnifiedContainerManager
    container        *runtime.UnifiedContainer
}

// Deploy deploys the KNIRVGATEWAY NOC
func (gnd *GatewayNestDeployer) Deploy(ctx context.Context) error {
    log.Println("🌐 Deploying KNIRVGATEWAY NOC...")

    config := &runtime.NestedObjectConfig{
        ObjectType:        runtime.ObjectTypeWebApp,
        EnableViewport:    true,
        ViewportRenderers: []string{"http", "webrtc", "vnc"},
        ServicePorts: map[string]int{
            "gateway":      8080,
            "oracle":       9090,
            "model_server": 11434,
        },
        Metadata: map[string]interface{}{
            "image":   "knirvgateway:latest",
            "command": []string{"/usr/local/bin/gateway", "--oracle-embedded"},
            "environment": map[string]string{
                "KNIRV_MODE":          "gateway_nest",
                "ENABLE_ORACLE":       "true",
                "ENABLE_MODEL_SERVER": "true",
                "GATEWAY_PORT":        "8080",
                "ORACLE_PORT":         "9090",
                "MODEL_SERVER_PORT":   "11434",
            },
            "volumes": map[string]string{
                "/data/gateway": "/var/lib/knirvgateway",
                "/data/models":  "/models",
            },
        },
    }

    container, err := gnd.containerManager.CreateNestedObject(ctx, config)
    if err != nil {
        return fmt.Errorf("KNIRVGATEWAY NOC creation failed: %w", err)
    }

    gnd.container = container

    log.Printf("✅ KNIRVGATEWAY NOC deployed at knirv://%s.object.nest", container.CryptoHash)
    gnd.printAccessInfo(container)

    return nil
}

// printAccessInfo prints access information for the gateway nest
func (gnd *GatewayNestDeployer) printAccessInfo(container *runtime.UnifiedContainer) {
    fmt.Println("\n" + strings.Repeat("=", 70))
    fmt.Println("🌐 KNIRVGATEWAY OBJECT NEST")
    fmt.Println(strings.Repeat("=", 70))
    fmt.Println()
    fmt.Printf("📡 SERVICES:\n")
    fmt.Printf("  Gateway API:      http://localhost:8080\n")
    fmt.Printf("  Documentation:    http://localhost:8080/docs\n")
    fmt.Printf("  Oracle Daemon:    http://localhost:9090\n")
    fmt.Printf("  Model Server:     http://localhost:11434\n")
    fmt.Println()
    fmt.Printf("🔗 OBJECT NEST URL:\n")
    fmt.Printf("  knirv://%s.object.nest?type=webapp\n", container.CryptoHash)
    fmt.Println()
    fmt.Printf("🧪 QUICK TESTS:\n")
    fmt.Printf("  curl http://localhost:8080/health\n")
    fmt.Printf("  curl http://localhost:9090/api/chains\n")
    fmt.Printf("  curl http://localhost:11434/v1/models\n")
    fmt.Println()
    fmt.Println("💡 NOTE: The oracle daemon (knirv-oracled) runs as an embedded")
    fmt.Println("   service within the gateway. It will consume container resources")
    fmt.Println("   as needed for cross-chain operations and governance.")
    fmt.Println()
    fmt.Println(strings.Repeat("=", 70))
    fmt.Println()
}
```

### Demo 2: KNIRVROUTER NOC

The KNIRVROUTER NOC demonstrates P2P network services running inside an NOC. This shows how decentralized infrastructure components can be deployed as nested objects.

#### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│            KNIRVROUTER NOC Container                │
│                                                             │
│  ┌────────────────────────────────────────────────────┐   │
│  │         KNIRVROUTER P2P Node                       │   │
│  │  - libp2p host with DHT                            │   │
│  │  - Content routing (hash → multiaddr)              │   │
│  │  - PubSub message broadcasting                     │   │
│  │  - Peer discovery and connectivity                 │   │
│  └────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌────────────────────────────────────────────────────┐   │
│  │         TURN/STUN Server                           │   │
│  │  - NAT traversal for browsers                      │   │
│  │  - WebRTC signaling support                        │   │
│  └────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌────────────────────────────────────────────────────┐   │
│  │         Router Management API                      │   │
│  │  - Peer statistics dashboard                       │   │
│  │  - DHT query interface                             │   │
│  │  - Network topology visualization                  │   │
│  └────────────────────────────────────────────────────┘   │
│                                                             │
│  Ports: 5001 (API), 4001 (P2P), 3478 (STUN), 3479 (TURN)  │
└─────────────────────────────────────────────────────────────┘
```

#### Implementation

```go
// pkg/demos/router_nest.go

package demos

import (
    "context"
    "fmt"
    "github.com/KNIRV/KNIRV_NETWORK/KNIRVNEXUS/pkg/runtime"
)

// RouterNestDeployer deploys KNIRVROUTER NOC
type RouterNestDeployer struct {
    containerManager *runtime.UnifiedContainerManager
    container        *runtime.UnifiedContainer
}

// Deploy deploys the KNIRVROUTER NOC
func (rnd *RouterNestDeployer) Deploy(ctx context.Context) error {
    log.Println("🔀 Deploying KNIRVROUTER NOC...")

    config := &runtime.NestedObjectConfig{
        ObjectType:        runtime.ObjectTypeP2P,
        EnableViewport:    true,
        ViewportRenderers: []string{"http", "vnc"}, // No WebRTC to avoid recursion
        ServicePorts: map[string]int{
            "api":   5001,
            "p2p":   4001,
            "stun":  3478,
            "turn":  3479,
        },
        Metadata: map[string]interface{}{
            "image":   "knirvrouter:latest",
            "command": []string{"/usr/local/bin/knirvrouter", "--config", "/config/nest.yaml"},
            "environment": map[string]string{
                "KNIRV_MODE":       "router_nest",
                "ENABLE_DHT":       "true",
                "ENABLE_PUBSUB":    "true",
                "ENABLE_TURN":      "true",
                "API_PORT":         "5001",
                "P2P_PORT":         "4001",
            },
            "volumes": map[string]string{
                "/data/router": "/var/lib/knirvrouter",
            },
            "capabilities": []string{
                "NET_ADMIN",        // For network operations
                "NET_BIND_SERVICE", // For privileged ports
            },
        },
    }

    container, err := rnd.containerManager.CreateNestedObject(ctx, config)
    if err != nil {
        return fmt.Errorf("KNIRVROUTER NOC creation failed: %w", err)
    }

    rnd.container = container

    log.Printf("✅ KNIRVROUTER NOC deployed at knirv://%s.object.nest", container.CryptoHash)
    rnd.printAccessInfo(container)

    return nil
}

// printAccessInfo prints access information for the router nest
func (rnd *RouterNestDeployer) printAccessInfo(container *runtime.UnifiedContainer) {
    fmt.Println("\n" + strings.Repeat("=", 70))
    fmt.Println("🔀 KNIRVROUTER OBJECT NEST")
    fmt.Println(strings.Repeat("=", 70))
    fmt.Println()
    fmt.Printf("📡 SERVICES:\n")
    fmt.Printf("  Router API:       http://localhost:5001\n")
    fmt.Printf("  P2P Listen:       tcp://localhost:4001\n")
    fmt.Printf("  STUN Server:      udp://localhost:3478\n")
    fmt.Printf("  TURN Server:      udp://localhost:3479\n")
    fmt.Println()
    fmt.Printf("🔗 OBJECT NEST URL:\n")
    fmt.Printf("  knirv://%s.object.nest?type=p2p\n", container.CryptoHash)
    fmt.Println()
    fmt.Printf("🧪 QUICK TESTS:\n")
    fmt.Printf("  curl http://localhost:5001/stats\n")
    fmt.Printf("  curl http://localhost:5001/dht/peers\n")
    fmt.Printf("  curl http://localhost:5001/topology\n")
    fmt.Println()
    fmt.Println("💡 NOTE: This router instance runs inside a container and can")
    fmt.Println("   be used to test multi-hop routing and nested P2P networks.")
    fmt.Println()
    fmt.Println(strings.Repeat("=", 70))
    fmt.Println()
}
```

### Unified Demo Deployment

```go
// pkg/deployment/demo_deployer.go

package deployment

import (
    "context"
    "sync"
    "github.com/KNIRV/KNIRV_NETWORK/KNIRVNEXUS/pkg/demos"
    "github.com/KNIRV/KNIRV_NETWORK/KNIRVNEXUS/pkg/runtime"
)

// DemoDeployer manages automatic deployment of all demo NOCs
type DemoDeployer struct {
    containerManager    *runtime.UnifiedContainerManager
    gatewayDeployer     *demos.GatewayNestDeployer
    routerDeployer      *demos.RouterNestDeployer
    deployed            bool
}

// NewDemoDeployer creates a new demo deployer
func NewDemoDeployer(containerManager *runtime.UnifiedContainerManager) *DemoDeployer {
    return &DemoDeployer{
        containerManager: containerManager,
        gatewayDeployer:  &demos.GatewayNestDeployer{containerManager: containerManager},
        routerDeployer:   &demos.RouterNestDeployer{containerManager: containerManager},
        deployed:         false,
    }
}

// DeployAll deploys all demo NOCs in parallel
func (dd *DemoDeployer) DeployAll(ctx context.Context) error {
    if dd.deployed {
        return fmt.Errorf("demos already deployed")
    }

    log.Println("🚀 Deploying demo NOCs...")

    var wg sync.WaitGroup
    errChan := make(chan error, 2)

    // Deploy KNIRVGATEWAY in parallel
    wg.Add(1)
    go func() {
        defer wg.Done()
        if err := dd.gatewayDeployer.Deploy(ctx); err != nil {
            errChan <- fmt.Errorf("gateway deployment failed: %w", err)
        }
    }()

    // Deploy KNIRVROUTER in parallel
    wg.Add(1)
    go func() {
        defer wg.Done()
        if err := dd.routerDeployer.Deploy(ctx); err != nil {
            errChan <- fmt.Errorf("router deployment failed: %w", err)
        }
    }()

    wg.Wait()
    close(errChan)

    // Check for errors
    for err := range errChan {
        return err
    }

    dd.deployed = true
    log.Println("✅ All demo NOCs deployed successfully")

    return nil
}

// Cleanup removes all demo NOCs
func (dd *DemoDeployer) Cleanup(ctx context.Context) error {
    if !dd.deployed {
        return nil
    }

    log.Println("🧹 Cleaning up demo NOCs...")

    var wg sync.WaitGroup
    errChan := make(chan error, 2)

    // Cleanup gateway
    if dd.gatewayDeployer.container != nil {
        wg.Add(1)
        go func() {
            defer wg.Done()
            if err := dd.containerManager.DestroyContainer(ctx, dd.gatewayDeployer.container.ID); err != nil {
                errChan <- fmt.Errorf("gateway cleanup failed: %w", err)
            }
        }()
    }

    // Cleanup router
    if dd.routerDeployer.container != nil {
        wg.Add(1)
        go func() {
            defer wg.Done()
            if err := dd.containerManager.DestroyContainer(ctx, dd.routerDeployer.container.ID); err != nil {
                errChan <- fmt.Errorf("router cleanup failed: %w", err)
            }
        }()
    }

    wg.Wait()
    close(errChan)

    for err := range errChan {
        log.Printf("Cleanup error: %v", err)
    }

    dd.deployed = false
    log.Println("✅ Demo NOCs cleaned up")

    return nil
}
```

### Backend Server Integration

```go
// KNIRVNEXUS/backend/cmd/backend_server/main.go

// Add to NewServer function:

func NewServer(cfg *config.Config) (*Server, error) {
    // ... existing initialization ...

    // Initialize unified container manager
    unifiedContainerManager := runtime.NewUnifiedContainerManager(
        teeSecurityService,
        ebpfManager,
        virtualContainerManager,
    )

    // Initialize demo deployer (manages all demo NOCs)
    demoDeployer := deployment.NewDemoDeployer(unifiedContainerManager)

    server := &Server{
        // ... existing fields ...
        unifiedContainerManager: unifiedContainerManager,
        demoDeployer:            demoDeployer,
    }

    return server, nil
}

// Add to Start function:

func (s *Server) Start() error {
    // ... existing startup ...

    // Deploy demo NOCs if enabled
    if s.config.DemoMode || s.config.AutoDeployDemos {
        log.Println("🚀 Auto-deploying demo NOCs...")
        go func() {
            // Wait for services to be ready
            time.Sleep(5 * time.Second)

            ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
            defer cancel()

            if err := s.demoDeployer.DeployAll(ctx); err != nil {
                log.Printf("ERROR: Demo deployment failed: %v", err)
            }
        }()
    }

    // ... rest of startup ...
}

// Add to Stop function:

func (s *Server) Stop() error {
    // ... existing shutdown ...

    // Cleanup demo NOCs
    if s.demoDeployer != nil {
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()

        if err := s.demoDeployer.Cleanup(ctx); err != nil {
            log.Printf("Error cleaning up demos: %v", err)
        }
    }

    // ... rest of shutdown ...
}
```

### Configuration

```yaml
# KNIRVNEXUS/config/production.yaml

# ... existing config ...

# Demo NOC configuration
demo_mode: true
auto_deploy_demos: true

object_nests:
  # Demo 1: KNIRVGATEWAY
  gateway:
    enabled: true
    image: "knirvgateway:latest"
    ports:
      gateway: 8080
      oracle: 9090
      model_server: 11434
    oracle:
      enabled: true
      chains:
        - chain_id: "knirv-testnet"
          rpc: "http://localhost:26657"
          grpc: "localhost:9090"
    model_server:
      enabled: true
      models:
        - name: "llama3-8b"
          type: "llm"
          path: "/models/llama3-8b.gguf"
          quantization: "q4_0"

  # Demo 2: KNIRVROUTER
  router:
    enabled: true
    image: "knirvrouter:latest"
    ports:
      api: 5001
      p2p: 4001
      stun: 3478
      turn: 3479
    p2p:
      enable_dht: true
      enable_pubsub: true
      bootstrap_peers:
        - "/dns4/bootstrap.knirv.network/tcp/4001/p2p/QmBootstrap..."

  # General NOC settings
  viewport:
    enabled: true
    default_renderers: ["http", "webrtc", "webgl", "vnc"]

  security:
    level: "strong"
    enable_ebpf: true
    enable_attestation: false

  # 3D Asset support
  glb_renderer:
    enabled: true
    max_polycount: 1000000
    enable_pbr: true
    enable_shadows: true
    texture_quality: "high"
```

## 3D Object Support (Native GLB)

### GLB Renderer Service

```go
// pkg/renderer/glb_renderer.go

package renderer

import (
    "context"
    "fmt"
    "net/http"
)

// GLBRenderer provides WebGL-based rendering for 3D GLB objects
type GLBRenderer struct {
    config        *GLBConfig
    assetRegistry *AssetRegistry
    httpServer    *http.Server
}

// GLBConfig defines GLB renderer configuration
type GLBConfig struct {
    Port            int    `json:"port"`
    MaxPolycount    int    `json:"max_polycount"`
    EnablePBR       bool   `json:"enable_pbr"`
    EnableShadows   bool   `json:"enable_shadows"`
    TextureQuality  string `json:"texture_quality"` // low, medium, high, ultra
}

// NewGLBRenderer creates a new GLB renderer
func NewGLBRenderer() *GLBRenderer {
    return &GLBRenderer{
        config: &GLBConfig{
            Port:           8080,
            MaxPolycount:   1000000,
            EnablePBR:      true,
            EnableShadows:  true,
            TextureQuality: "high",
        },
        assetRegistry: NewAssetRegistry(),
    }
}

// Start starts the GLB renderer HTTP server
func (gr *GLBRenderer) Start(ctx context.Context) error {
    router := http.NewServeMux()
    router.HandleFunc("/", gr.serveViewer)
    router.HandleFunc("/asset.glb", gr.serveAsset)
    router.HandleFunc("/metadata", gr.serveMetadata)

    gr.httpServer = &http.Server{
        Addr:    fmt.Sprintf(":%d", gr.config.Port),
        Handler: router,
    }

    go func() {
        if err := gr.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Printf("GLB renderer error: %v", err)
        }
    }()

    return nil
}

// LoadAsset loads a GLB asset into the registry
func (gr *GLBRenderer) LoadAsset(path string) (*AssetMetadata, error) {
    // Parse GLB file and extract metadata
    metadata, err := parseGLBFile(path)
    if err != nil {
        return nil, fmt.Errorf("GLB parsing failed: %w", err)
    }

    // Validate polycount
    if metadata.Polycount > gr.config.MaxPolycount {
        return nil, fmt.Errorf("polycount %d exceeds limit %d",
            metadata.Polycount, gr.config.MaxPolycount)
    }

    // Register asset
    gr.assetRegistry.Register(metadata)

    return metadata, nil
}

// serveViewer serves the WebGL viewer interface
func (gr *GLBRenderer) serveViewer(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/html")
    w.Write([]byte(glbViewerHTML))
}

// serveAsset serves the GLB binary data
func (gr *GLBRenderer) serveAsset(w http.ResponseWriter, r *http.Request) {
    assetID := r.URL.Query().Get("id")
    if assetID == "" {
        http.Error(w, "asset ID required", http.StatusBadRequest)
        return
    }

    asset, err := gr.assetRegistry.Get(assetID)
    if err != nil {
        http.Error(w, "asset not found", http.StatusNotFound)
        return
    }

    w.Header().Set("Content-Type", "model/gltf-binary")
    w.Header().Set("Content-Length", fmt.Sprintf("%d", asset.FileSize))
    http.ServeFile(w, r, asset.FilePath)
}

// serveMetadata serves asset metadata as JSON
func (gr *GLBRenderer) serveMetadata(w http.ResponseWriter, r *http.Request) {
    assetID := r.URL.Query().Get("id")
    if assetID == "" {
        http.Error(w, "asset ID required", http.StatusBadRequest)
        return
    }

    asset, err := gr.assetRegistry.Get(assetID)
    if err != nil {
        http.Error(w, "asset not found", http.StatusNotFound)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(asset.Metadata)
}
```

### Asset Registry

```go
// pkg/renderer/asset_registry.go

package renderer

import (
    "sync"
)

// AssetRegistry manages 3D assets
type AssetRegistry struct {
    assets map[string]*Asset
    mu     sync.RWMutex
}

// Asset represents a 3D asset
type Asset struct {
    ID          string           `json:"id"`
    FilePath    string           `json:"file_path"`
    FileSize    int64            `json:"file_size"`
    Metadata    *AssetMetadata   `json:"metadata"`
    LoadedAt    time.Time        `json:"loaded_at"`
}

// NewAssetRegistry creates a new asset registry
func NewAssetRegistry() *AssetRegistry {
    return &AssetRegistry{
        assets: make(map[string]*Asset),
    }
}

// Register registers a new asset
func (ar *AssetRegistry) Register(metadata *AssetMetadata) error {
    ar.mu.Lock()
    defer ar.mu.Unlock()

    asset := &Asset{
        ID:       generateAssetID(),
        FilePath: metadata.FilePath,
        FileSize: metadata.FileSize,
        Metadata: metadata,
        LoadedAt: time.Now(),
    }

    ar.assets[asset.ID] = asset
    return nil
}

// Get retrieves an asset by ID
func (ar *AssetRegistry) Get(id string) (*Asset, error) {
    ar.mu.RLock()
    defer ar.mu.RUnlock()

    asset, ok := ar.assets[id]
    if !ok {
        return nil, fmt.Errorf("asset not found: %s", id)
    }

    return asset, nil
}

// List returns all registered assets
func (ar *AssetRegistry) List() []*Asset {
    ar.mu.RLock()
    defer ar.mu.RUnlock()

    assets := make([]*Asset, 0, len(ar.assets))
    for _, asset := range ar.assets {
        assets = append(assets, asset)
    }

    return assets
}
```

## Implementation Roadmap

### Phase 1: Universal Container System (Week 1-2)

- [ ] Implement `UnifiedContainer` with `ObjectType` support
- [ ] Create content-aware `ViewportProxy` with multiple renderers
- [ ] Implement `NestedObjectConfig` for flexible object deployment
- [ ] Add 3D asset support (GLB/GLTF parsing and metadata)
- [ ] Write comprehensive tests

### Phase 2: Cryptographic Routing Enhancement (Week 2-3)

- [ ] Extend `CryptoRouter` with object type filtering
- [ ] Update multiaddr format to include object type prefix
- [ ] Implement browser resolver with content negotiation
- [ ] Test cross-object-type routing
- [ ] Performance optimization for DHT queries

### Phase 3: GLB Renderer Implementation (Week 3-4)

- [ ] Implement `GLBRenderer` with Three.js integration
- [ ] Create `AssetRegistry` for 3D asset management
- [ ] Build WebGL viewer with PBR and shadows
- [ ] Add animation support and camera controls
- [ ] Test with various GLB files

### Phase 4: Demo NOCs (Week 4-5)

- [ ] Implement `GatewayNestDeployer` (KNIRVGATEWAY)
- [ ] Implement `RouterNestDeployer` (KNIRVROUTER)
- [ ] Create unified `DemoDeployer` for parallel deployment
- [ ] Build container images for both demos
- [ ] Test demo deployment flow

### Phase 5: Viewport Enhancement (Week 5-6)

- [ ] Add content-type detection to viewport
- [ ] Implement renderer switching (HTTP/WebRTC/WebGL/VNC)
- [ ] Build unified browser client with tabbed interface
- [ ] Add real-time metrics and log streaming
- [ ] Test cross-browser compatibility

### Phase 6: Testing & Documentation (Week 6-7)

- [ ] End-to-end integration tests for all object types
- [ ] Performance benchmarking (3D rendering, routing latency)
- [ ] Security audit (container isolation, crypto routing)
- [ ] Complete API documentation
- [ ] User guides for each demo

### Phase 7: Production Deployment (Week 7-8)

- [ ] Production container images (optimized builds)
- [ ] Kubernetes manifests for KNIRVNEXUS + demos
- [ ] Monitoring and alerting (Prometheus/Grafana)
- [ ] Disaster recovery procedures
- [ ] Launch demo NOCs publicly

## Deployment Guide

### Prerequisites

```bash
# Install dependencies
sudo apt-get update
sudo apt-get install -y \
    docker.io \
    make \
    golang-1.21 \
    nodejs \
    npm \
    git

# Build container images
cd KNIRVGATEWAY && make docker-build
cd KNIRVROUTER && make docker-build
cd glb-renderer && docker build -t glb-renderer:latest .
```

### Deployment Steps

```bash
# 1. Configure KNIRVNEXUS
cd KNIRVNEXUS
cp config/production.yaml.example config/production.yaml

# Edit config/production.yaml:
# - Set demo_mode: true
# - Set auto_deploy_demos: true

# 2. Build KNIRVNEXUS
make build

# 3. Start KNIRVNEXUS (demos deploy automatically)
sudo ./bin/knirvnexus-backend --config config/production.yaml

# Wait ~10 seconds for parallel demo deployment

# 4. Verify deployments
curl http://localhost:8080/health  # KNIRVGATEWAY
curl http://localhost:9090/api/chains  # Oracle daemon
curl http://localhost:11434/v1/models  # Model server
curl http://localhost:5001/stats  # KNIRVROUTER
curl http://localhost:4001/peers  # P2P network

# 5. Access viewports in browser
# Gateway: http://localhost:8080/viewport
# Router:  http://localhost:5001/dashboard
# Or use: knirv://<hash>.object.nest
```

### Docker Compose Deployment

```yaml
# docker-compose.object-nest.yml

version: '3.8'

services:
  knirvnexus:
    image: knirvnexus:latest
    ports:
      - "7070:7070"  # KNIRVNEXUS API
      - "8080:8080"  # KNIRVGATEWAY
      - "9090:9090"  # Oracle daemon
      - "11434:11434" # Model server
      - "5001:5001"  # KNIRVROUTER API
      - "4001:4001"  # P2P network
      - "3478:3478/udp" # STUN
      - "3479:3479/udp" # TURN
    volumes:
      - nexus-data:/data
      - /var/run/docker.sock:/var/run/docker.sock
    environment:
      - DEMO_MODE=true
      - AUTO_DEPLOY_DEMOS=true
    command: ["--config", "/config/production.yaml"]
    privileged: true

volumes:
  nexus-data:
```

```bash
# Deploy with Docker Compose
docker-compose -f docker-compose.object-nest.yml up -d

# View logs
docker-compose -f docker-compose.object-nest.yml logs -f knirvnexus

# Teardown
docker-compose -f docker-compose.object-nest.yml down -v
```

## Security Considerations

### Isolation Levels

1. **Basic**: Namespace isolation + seccomp
2. **Strong**: Basic + AppArmor/SELinux + eBPF monitoring
3. **Extreme**: Strong + Hardware TEE (SGX/SEV/TDX)

### eBPF Security Layer

- Syscall monitoring for anomaly detection
- LSM hooks for mandatory access control
- Network filtering via XDP
- Process sandboxing

### Cryptographic Routing Security

- BLAKE3 hash prevents address prediction
- Object type prefix enables content filtering
- DHT provides decentralized discovery
- libp2p encryption for all P2P traffic
- Optional TLS for HTTP endpoints

### 3D Asset Security

- Polycount limits prevent DoS
- Texture size validation
- Shader sanitization
- Resource usage monitoring

## Monitoring & Observability

### Metrics Collection

```go
// Prometheus metrics
object_nest_containers_total{type="webapp|api|3d|blockchain|p2p"}
object_nest_container_cpu_usage{type="webapp|api|3d|blockchain|p2p"}
object_nest_container_memory_usage{type="webapp|api|3d|blockchain|p2p"}
object_nest_viewport_connections_active{renderer="http|webrtc|webgl|vnc"}
object_nest_glb_render_fps
object_nest_glb_polycount
object_nest_routing_latency_seconds{object_type="webapp|api|3d|blockchain|p2p"}
```

### Logging

```json
{
  "timestamp": "2026-01-12T10:00:00Z",
  "level": "info",
  "component": "object_nest",
  "container_id": "abc123",
  "object_type": "webapp",
  "crypto_hash": "def456",
  "message": "Container started successfully"
}
```

### Alerting

- Container crash detection (all types)
- Resource exhaustion warnings
- Viewport connection failures
- 3D rendering performance degradation
- P2P network connectivity issues
- Security policy violations

## FAQ

### Q: What types of objects can I deploy in an NOC?

Any containerizable service: web apps, APIs, blockchain nodes, model servers, 3D GLB objects, P2P routers, file servers, or custom Docker containers.

### Q: How do I deploy my own object (not a demo)?

Create an `NestedObjectConfig` with your container image, ports, and metadata, then call `containerManager.CreateNestedObject(ctx, config)`.

### Q: Can I deploy multiple NOCs on one KNIRVNEXUS instance?

Yes. The `UnifiedContainerManager` supports unlimited NOC containers, constrained only by host resources.

### Q: How does the KNIRVGATEWAY oracle daemon get resources?

The oracle daemon runs as an embedded process within the KNIRVGATEWAY container. It shares the container's resource limits (4 CPU cores, 8GB RAM) and will consume resources as needed for operations.

### Q: Can I disable the demo NOCs?

Yes. Set `auto_deploy_demos: false` in the configuration file, or disable specific demos via `object_nests.gateway.enabled: false` or `object_nests.router.enabled: false`.

### Q: What happens if the KNIRVROUTER demo is unavailable?

If the KNIRVROUTER NOC fails, cryptographic routing will fall back to the host KNIRVROUTER instance (not containerized). Direct IP access remains functional.

### Q: How do I render custom 3D models?

Deploy an NOC with `ObjectType: ObjectType3D`, mount your GLB file, and access via `knirv://<hash>.object.nest?type=3d`. The WebGL viewer loads automatically.

### Q: Can I run KNIRVROUTER inside KNIRVROUTER recursively?

While technically possible, nested routers create routing complexity. Use the demo for testing multi-hop scenarios, not production deployments.

### Q: How does object type routing work?

The cryptographic hash includes the object type. Resolvers can filter by type during DHT queries, enabling content-aware routing and renderer selection.

## References

- [KNIRVNEXUS Documentation](./KNIRVNEXUS/README.md)
- [KNIRVGATEWAY Documentation](./KNIRVGATEWAY/README.md)
- [KNIRVROUTER Documentation](./KNIRVROUTER/README.md)
- [libp2p Specification](https://github.com/libp2p/specs)
- [WebRTC Specification](https://www.w3.org/TR/webrtc/)
- [eBPF Documentation](https://ebpf.io/what-is-ebpf/)
- [glTF 2.0 Specification](https://www.khronos.org/gltf/)
- [Three.js Documentation](https://threejs.org/docs/)

## License

Copyright © 2026 KNIRV Network. All rights reserved.
