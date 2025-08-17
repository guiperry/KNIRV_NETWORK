package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"strings"
	"sync"
	"time"

	"KNIRVORACLE/config" // Added for config.Role
	"KNIRVORACLE/types"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/libp2p/go-libp2p/p2p/host/autorelay"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	libp2ptls "github.com/libp2p/go-libp2p/p2p/security/tls"
	"github.com/multiformats/go-multiaddr"
	"github.com/multiformats/go-multihash"
)

// Default bootstrap nodes - these would be replaced with actual stable nodes in production
var DefaultBootstrapPeers = []string{
	"/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
	"/ip4/104.236.179.241/tcp/4001/p2p/QmSoLPppuBtQSGwKDZT2M73ULpjvfd3aZ6ha4oFGL1KrGM",
	"/ip4/128.199.219.111/tcp/4001/p2p/QmSoLSafTMBsPKadTEgaXctDQVcqN88CNLHXMkTNwMKPnu",
	"/ip4/104.236.76.40/tcp/4001/p2p/QmSoLV4Bbm51jM9C4gDYZQ9Cy3U6aXMJDAbzgu2fzaDs64",
}

// BootnodeRegistryURL is the URL for the KNIRVORACLE Node Registry
var BootnodeRegistryURL = "https://registry.knirv.com"

const (
	DiscoveryResourceTypeChain types.DiscoveryResourceType = "chain"
	DiscoveryResourceTypeNRN   types.DiscoveryResourceType = "nrn" // Example, if you have NRN specific resources in DHT
	// MCP capability types like "resource", "tool" will be handled as strings directly
)

// BootnodeInfo represents information about a bootnode from the registry
type BootnodeInfo struct {
	IP       string `json:"ip"`
	Port     int    `json:"port"` // This should be the P2P port
	LastSeen int64  `json:"lastSeen"`
	PeerID   string `json:"peerID,omitempty"` // Add PeerID
	Type     string `json:"type,omitempty"`   // To filter for bootnodes
}

// DiscoveryManager handles decentralized discovery using libp2p DHT
type DiscoveryManager struct {
	host         host.Host
	dht          *dual.DHT
	bootstrapped bool
	clients      map[string]*ReflectionClient
	mu           sync.Mutex
	stopChan     chan struct{}
	ctx          context.Context
	cancel       context.CancelFunc
	chainID      string
	p2pPort      int
	clientOnly   bool                 // Add this flag
	IsBootnode   bool                 // Flag indicating if this is a bootnode
	nodeRole     config.Role          // Added
	closeOnce    sync.Once            // Add sync.Once
	cfg          *config.Config       // Add config
	closed       bool                 // Optional: flag to check if already closed
	closeMu      sync.Mutex           // Mutex to protect the closed flag if used
	recentPeers  map[string]time.Time // Track recently registered peers
}

// IsRecentlyRegistered checks if a peer was recently registered (within last 5 minutes)
func (dm *DiscoveryManager) IsRecentlyRegistered(peerID string) bool {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if timestamp, exists := dm.recentPeers[peerID]; exists {
		return time.Since(timestamp) < 5*time.Minute
	}
	return false
}

// NewDiscoveryManager creates a new discovery manager with DHT capabilities
func NewDiscoveryManager(chainID string, p2pPort int, clientOnly bool, IsBootnode bool, role config.Role, cfg *config.Config) (*DiscoveryManager, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Generate or load a persistent key
	priv, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to generate node key: %w", err)
	}

	// Create a connection manager
	connManager, err := connmgr.NewConnManager(
		100, // Low watermark
		400, // High watermark
		connmgr.WithGracePeriod(time.Minute),
	)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create connection manager: %w", err)
	}

	// Setup listening addresses
	listenAddrs := []multiaddr.Multiaddr{}

	// Add IPv4 listening address
	addr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", p2pPort))
	if err == nil {
		listenAddrs = append(listenAddrs, addr)
	}

	// Add IPv6 listening address
	addr, err = multiaddr.NewMultiaddr(fmt.Sprintf("/ip6/::/tcp/%d", p2pPort))
	if err == nil {
		listenAddrs = append(listenAddrs, addr)
	}

	// --- Conditional Libp2p Options ---
	opts := []libp2p.Option{
		libp2p.Identity(priv),
		libp2p.ListenAddrs(listenAddrs...),
		libp2p.Security(libp2ptls.ID, libp2ptls.New),
		libp2p.Security(noise.ID, noise.New),
		libp2p.DefaultTransports,
		libp2p.ConnectionManager(connManager),
		libp2p.EnableHolePunching(), // Keep hole punching
		libp2p.EnableNATService(),   // Keep NAT service
	}

	if !clientOnly {
		// Only enable AutoRelay and NATPortMap for full participants
		opts = append(opts, libp2p.NATPortMap())
		opts = append(opts, libp2p.EnableAutoRelayWithPeerSource(
			func(ctx context.Context, numPeers int) <-chan peer.AddrInfo {
				peerChan := make(chan peer.AddrInfo, len(DefaultBootstrapPeers))
				for _, addr := range DefaultBootstrapPeers {
					pi, err := peer.AddrInfoFromString(addr)
					if err != nil {
						log.Printf("Warning: Failed to parse bootstrap peer for AutoRelay: %s, %v", addr, err)
						continue
					}
					select {
					case peerChan <- *pi:
					case <-ctx.Done():
						close(peerChan)
						return nil
					}
				}
				close(peerChan)
				return peerChan
			},
			autorelay.WithMinInterval(time.Minute),
		))
	}
	// --- End Conditional Options ---

	h, err := libp2p.New(opts...)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create libp2p host: %w", err)
	}

	// Setup DHT for peer discovery with unique protocol prefix for test isolation
	dhtProtocolPrefix := protocol.ID(fmt.Sprintf("/KNIRVORACLE-dht/%s/%s", chainID, h.ID().String()[:8]))
	idht, err := dual.New(ctx, h,
		dual.DHTOption(dht.Mode(dht.ModeServer)),
		dual.DHTOption(dht.BootstrapPeers(convertToAddrInfo(DefaultBootstrapPeers)...)),
		dual.DHTOption(dht.ProtocolPrefix(dhtProtocolPrefix)),
	)
	if err != nil {
		h.Close()
		cancel()
		return nil, fmt.Errorf("failed to create DHT: %w", err)
	}
	log.Printf("[%s][%s] DHT initialized with protocol prefix: %s", role.String(), chainID, dhtProtocolPrefix)

	// --- Conditional mDNS Start ---
	var disc mdns.Service // Declare disc here
	if !clientOnly {
		disc = mdns.NewMdnsService(h, "KNIRVORACLE", &discoveryNotifee{h: h})
		if err := disc.Start(); err != nil {
			h.Close()
			cancel()
			return nil, fmt.Errorf("failed to start mDNS: %w", err)
		}
	}
	// --- End Conditional mDNS Start ---

	log.Printf("[%s][%s] Discovery Manager initialized. PeerID: %s (ClientOnly: %t)", role.String(), chainID, h.ID().String(), clientOnly)
	for _, addr := range h.Addrs() {
		log.Printf("[%s][%s] Listening on: %s/p2p/%s", role.String(), chainID, addr.String(), h.ID().String())
	}

	return &DiscoveryManager{
		host:        h,
		dht:         idht,
		clients:     make(map[string]*ReflectionClient),
		ctx:         ctx,
		cancel:      cancel,
		chainID:     chainID,
		p2pPort:     p2pPort,
		clientOnly:  clientOnly,                 // Store the flag
		IsBootnode:  IsBootnode,                 // Store the bootnode flag
		nodeRole:    role,                       // Store role
		cfg:         cfg,                        // Store config
		recentPeers: make(map[string]time.Time), // Initialize recent peers map
		stopChan:    make(chan struct{}),        // Initialize stopChan
	}, nil
}

// discoveryNotifee gets notified when we find a new peer via mDNS discovery
type discoveryNotifee struct {
	h host.Host
}

// HandlePeerFound connects to peers discovered via mDNS
func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	if n.h.Network().Connectedness(pi.ID) != network.Connected {
		// Consider adding role/chainID to this log if context is available, or make it a method of DiscoveryManager
		log.Printf("mDNS: Found local peer: %s, attempting connection", pi.ID.String())
		err := n.h.Connect(context.Background(), pi)
		if err != nil {
			log.Printf("mDNS: Error connecting to local peer %s: %s", pi.ID.String(), err)
		}
	}
}

// convertToAddrInfo converts string multiaddrs to AddrInfo
func convertToAddrInfo(addrs []string) []peer.AddrInfo {
	var pis []peer.AddrInfo
	for _, addr := range addrs {
		ma, err := multiaddr.NewMultiaddr(addr)
		if err != nil {
			log.Printf("Invalid bootstrap address: %s, %s", addr, err)
			continue
		}

		ai, err := peer.AddrInfoFromP2pAddr(ma)
		if err != nil {
			log.Printf("Failed to convert to AddrInfo: %s, %s", addr, err)
			continue
		}

		pis = append(pis, *ai)
	}
	return pis
}

// Bootstrap connects to the bootstrap nodes and joins the DHT network
func (dm *DiscoveryManager) Bootstrap() error {
	if dm.bootstrapped {
		return nil
	}

	// Connect to bootstrap nodes
	log.Printf("[%s][%s] Connecting to bootstrap nodes...", dm.nodeRole.String(), dm.chainID)

	ctx, cancel := context.WithTimeout(dm.ctx, 60*time.Second)
	defer cancel()

	// Try to fetch bootnodes from registry first
	var bootstrapPeers []peer.AddrInfo
	bootnodes, err := FetchBootnodesFromRegistry(dm.nodeRole.String(), dm.chainID) // Pass context for logging
	if err != nil {
		log.Printf("[%s][%s] WARNING: Failed to fetch bootnodes from registry: %v", dm.nodeRole.String(), dm.chainID, err)
		log.Println("Falling back to default bootstrap peers")
		bootstrapPeers = convertToAddrInfo(DefaultBootstrapPeers)
	} else {
		// Convert registry bootnodes to peer.AddrInfo
		for chainID, info := range bootnodes {
			// Filter for actual bootnodes if type information is present
			if info.Type != "" && strings.ToLower(info.Type) != "bootnode" {
				log.Printf("[%s][%s] Skipping non-bootnode entry from registry: %s (type: %s)", dm.nodeRole.String(), dm.chainID, chainID, info.Type)
				continue
			}

			// Skip self if we're a bootnode
			if dm.IsBootnode && chainID == dm.chainID {
				continue
			}

			// Create multiaddr from IP and port
			addrStr := fmt.Sprintf("/ip4/%s/tcp/%d", info.IP, info.Port)
			// If PeerID is available from registry, append it.
			if info.PeerID != "" {
				addrStr = fmt.Sprintf("%s/p2p/%s", addrStr, info.PeerID)
			}

			ma, err := multiaddr.NewMultiaddr(addrStr)
			if err != nil {
				log.Printf("[%s][%s] WARNING: Invalid bootnode multiaddr for %s (%s): %v", dm.nodeRole.String(), dm.chainID, chainID, addrStr, err)
				continue
			}

			ai, err := peer.AddrInfoFromP2pAddr(ma)
			if err != nil {
				log.Printf("[%s][%s] WARNING: Failed to create AddrInfo for %s from multiaddr %s: %v", dm.nodeRole.String(), dm.chainID, chainID, ma.String(), err)
				continue
			}

			bootstrapPeers = append(bootstrapPeers, *ai)
		}

		// If no valid bootnodes from registry, fall back to defaults
		if len(bootstrapPeers) == 0 {
			log.Println("No valid bootnodes found in registry, falling back to default bootstrap peers")
			bootstrapPeers = convertToAddrInfo(DefaultBootstrapPeers)
		} else {
			log.Printf("Found %d bootnodes from registry", len(bootstrapPeers))
		}
	}
	log.Printf("[%s][%s] Attempting to connect to %d bootstrap peers...", dm.nodeRole.String(), dm.chainID, len(bootstrapPeers))
	// Connect to bootstrap peers
	wg := sync.WaitGroup{}
	for _, peerInfo := range bootstrapPeers {
		wg.Add(1)
		go func(peerInfo peer.AddrInfo) {
			defer wg.Done()
			if err := dm.host.Connect(ctx, peerInfo); err != nil {
				log.Printf("[%s][%s] Error connecting to bootstrap node %s: %s", dm.nodeRole.String(), dm.chainID, peerInfo.ID.String(), err)
			} else {
				log.Printf("[%s][%s] Connected to bootstrap node: %s", dm.nodeRole.String(), dm.chainID, peerInfo.ID.String())
			}
		}(peerInfo)
	}

	// Wait with a timeout for connections to bootstrap nodes
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Printf("[%s][%s] Bootstrap connections completed.", dm.nodeRole.String(), dm.chainID)
	case <-time.After(60 * time.Second):
		log.Printf("[%s][%s] Bootstrap timed out, continuing with available connections.", dm.nodeRole.String(), dm.chainID)
	}

	// Bootstrap the DHT
	if err := dm.dht.Bootstrap(dm.ctx); err != nil {
		return fmt.Errorf("[%s][%s] failed to bootstrap DHT: %w", dm.nodeRole.String(), dm.chainID, err)
	}

	dm.bootstrapped = true
	log.Printf("[%s][%s] DHT bootstrapped successfully.", dm.nodeRole.String(), dm.chainID)

	return nil
}

// AnnounceGenericResource announces a generic resource (like "chain") to the DHT
func (dm *DiscoveryManager) AnnounceGenericResource(id string, resourceType types.DiscoveryResourceType) error {
	// Create a CID from the resource identifier
	// For generic resources, resourceType is from the enum (e.g., "chain")
	resourceKey := fmt.Sprintf("%s.%s", id, string(resourceType))
	h := sha256.New()
	h.Write([]byte(resourceKey))
	mh, err := multihash.Encode(h.Sum(nil), multihash.SHA2_256)
	if err != nil {
		return fmt.Errorf("failed to create multihash: %w", err)
	}

	resourceCID := cid.NewCidV1(cid.Raw, mh)

	// Announce that we can provide this resource
	log.Printf("[%s][%s] Announcing resource: %s (CID: %s)", dm.nodeRole.String(), dm.chainID, resourceKey, resourceCID.String())

	ctx, cancel := context.WithTimeout(dm.ctx, 60*time.Second)
	defer cancel()

	if err := dm.dht.Provide(ctx, resourceCID, true); err != nil {
		return fmt.Errorf("[%s][%s] failed to announce resource %s: %w", dm.nodeRole.String(), dm.chainID, resourceKey, err)
	}

	log.Printf("[%s][%s] Successfully announced resource: %s", dm.nodeRole.String(), dm.chainID, resourceKey)
	return nil
}

// FindGenericResource looks up a generic resource (like "chain") in the DHT
func (dm *DiscoveryManager) FindGenericResource(id string, resourceType types.DiscoveryResourceType) ([]peer.AddrInfo, error) {
	// Create the same CID as used in AnnounceResource
	// For generic resources, resourceType is from the enum
	resourceKey := fmt.Sprintf("%s.%s", id, string(resourceType))
	h := sha256.New()
	h.Write([]byte(resourceKey))
	mh, err := multihash.Encode(h.Sum(nil), multihash.SHA2_256)
	if err != nil {
		return nil, fmt.Errorf("failed to create multihash: %w", err)
	}

	resourceCID := cid.NewCidV1(cid.Raw, mh)

	// Find providers for this resource
	log.Printf("[%s][%s] Looking for providers of resource: %s (CID: %s)", dm.nodeRole.String(), dm.chainID, resourceKey, resourceCID.String())

	ctx, cancel := context.WithTimeout(dm.ctx, 30*time.Second)
	defer cancel()

	providers := dm.dht.FindProvidersAsync(ctx, resourceCID, 20)

	var results []peer.AddrInfo
	for p := range providers {
		if p.ID == dm.host.ID() {
			// Skip ourselves
			continue
		}
		results = append(results, p)
		log.Printf("[%s][%s] Found provider for %s: %s", dm.nodeRole.String(), dm.chainID, resourceKey, p.ID.String())
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("[%s][%s] no providers found for resource: %s", dm.nodeRole.String(), dm.chainID, resourceKey)
	}

	return results, nil
}

// AnnounceMCPCapability announces an MCP capability to the DHT using its specific type string.
func (dm *DiscoveryManager) AnnounceMCPCapability(ctx context.Context, id string, mcpTypeString string) error {
	if id == "" || mcpTypeString == "" {
		return fmt.Errorf("capability ID and type string cannot be empty for DHT announcement")
	}
	// Use lowercase type string for consistency in DHT keys
	dhtResourceType := strings.ToLower(mcpTypeString)
	resourceKey := fmt.Sprintf("%s.%s", id, dhtResourceType)

	h := sha256.New()
	h.Write([]byte(resourceKey))
	mh, err := multihash.Encode(h.Sum(nil), multihash.SHA2_256)
	if err != nil {
		return fmt.Errorf("failed to create multihash for MCP capability %s: %w", resourceKey, err)
	}
	resourceCID := cid.NewCidV1(cid.Raw, mh)

	log.Printf("[%s][%s] Announcing MCP Capability: %s (CID: %s)", dm.nodeRole.String(), dm.chainID, resourceKey, resourceCID.String())

	if err := dm.dht.Provide(ctx, resourceCID, true); err != nil {
		log.Printf("[%s][%s] ERROR: Failed to announce MCP capability %s via DHT: %v", dm.nodeRole.String(), dm.chainID, resourceKey, err)
		return fmt.Errorf("[%s][%s] failed to announce MCP capability %s: %w", dm.nodeRole.String(), dm.chainID, resourceKey, err)
	}

	log.Printf("[%s][%s] Successfully announced MCP capability: %s", dm.nodeRole.String(), dm.chainID, resourceKey)
	return nil
}

// FindMCPCapabilityProviders looks up providers for an MCP capability in the DHT using its specific type string.
func (dm *DiscoveryManager) FindMCPCapabilityProviders(ctx context.Context, id string, mcpTypeString string) ([]peer.AddrInfo, error) {
	if id == "" || mcpTypeString == "" {
		return nil, fmt.Errorf("capability ID and type string cannot be empty for DHT lookup")
	}
	dhtResourceType := strings.ToLower(mcpTypeString)
	resourceKey := fmt.Sprintf("%s.%s", id, dhtResourceType)

	h := sha256.New()
	h.Write([]byte(resourceKey))
	mh, err := multihash.Encode(h.Sum(nil), multihash.SHA2_256)
	if err != nil {
		return nil, fmt.Errorf("failed to create multihash for MCP capability %s: %w", resourceKey, err)
	}
	resourceCID := cid.NewCidV1(cid.Raw, mh)

	log.Printf("[%s][%s] Finding providers for MCP Capability: %s (CID: %s)", dm.nodeRole.String(), dm.chainID, resourceKey, resourceCID.String())

	providers := dm.dht.FindProvidersAsync(ctx, resourceCID, 20) // 20 is a common value for number of providers to find
	var results []peer.AddrInfo
	for p := range providers {
		if p.ID != dm.host.ID() { // Don't include self
			results = append(results, p)
		}
	}
	return results, nil // Returns empty slice if no providers found, error only on DHT issues
}

// ConnectToPeer establishes a connection to a peer
// Pass context for timeout/cancellation
func (dm *DiscoveryManager) ConnectToPeer(peerInfo peer.AddrInfo, ctx context.Context) error {
	if dm.host.Network().Connectedness(peerInfo.ID) == network.Connected {
		return nil // Already connected
	}
	// Use the passed context for the connection attempt
	return dm.host.Connect(ctx, peerInfo)
}

// GetPeerMultiaddrs returns the multiaddresses of a peer
func (dm *DiscoveryManager) GetPeerMultiaddrs(peerID peer.ID) ([]multiaddr.Multiaddr, error) {
	// If we're already connected, get the addresses directly
	if dm.host.Network().Connectedness(peerID) == network.Connected {
		return dm.host.Peerstore().Addrs(peerID), nil
	}

	// Otherwise, try to find the peer in the DHT
	ctx, cancel := context.WithTimeout(dm.ctx, 10*time.Second)
	defer cancel()

	peerInfo, err := dm.dht.FindPeer(ctx, peerID)
	if err != nil {
		return nil, fmt.Errorf("[%s][%s] failed to find peer %s: %w", dm.nodeRole.String(), dm.chainID, peerID.String(), err)
	}

	return peerInfo.Addrs, nil
}

// GetSelfMultiaddrs returns this node's multiaddresses
func (dm *DiscoveryManager) GetSelfMultiaddrs() []string {
	var addrs []string
	for _, addr := range dm.host.Addrs() {
		addrs = append(addrs, fmt.Sprintf("%s/p2p/%s", addr.String(), dm.host.ID().String()))
	}
	return addrs
}

// GetPeerID returns this node's peer ID
func (dm *DiscoveryManager) GetPeerID() string {
	return dm.host.ID().String()
}

// GetPeerCount returns the number of connected peers
func (dm *DiscoveryManager) GetPeerCount() int {
	if dm.host == nil {
		return 0
	}
	return len(dm.host.Network().Conns())
}

// Run starts the discovery and health check loop
func (dm *DiscoveryManager) Run(interval time.Duration) {
	// Use mutex to safely handle the stopChan
	dm.closeMu.Lock()
	// We don't need to recreate the stopChan as it's now initialized in NewDiscoveryManager
	// Just make sure it's not nil for backward compatibility
	if dm.stopChan == nil {
		dm.stopChan = make(chan struct{})
	}
	dm.closeMu.Unlock()

	// Bootstrap the DHT
	if err := dm.Bootstrap(); err != nil {
		log.Printf("[%s][%s] Failed to bootstrap DHT: %s", dm.nodeRole.String(), dm.chainID, err)
	}

	// --- Bootnode Registration ---
	if dm.IsBootnode {
		log.Printf("[%s][%s] Running as a bootnode. Registering with registry...", dm.nodeRole.String(), dm.chainID)
		// Register with the registry
		if err := dm.RegisterBootnodeWithRegistry(); err != nil {
			log.Printf("[%s][%s] WARNING: Failed to register with registry: %v", dm.nodeRole.String(), dm.chainID, err)
		} else {
			log.Printf("[%s][%s] Successfully registered as bootnode with registry", dm.nodeRole.String(), dm.chainID)
		}

		// Set up periodic re-registration
		go func() {
			registrationTicker := time.NewTicker(30 * time.Minute) // Re-register every 30 minutes
			defer registrationTicker.Stop()

			for {
				select {
				case <-registrationTicker.C:
					if err := dm.RegisterBootnodeWithRegistry(); err != nil {
						log.Printf("[%s][%s] WARNING: Failed to re-register with registry: %v", dm.nodeRole.String(), dm.chainID, err)
					} else {
						log.Printf("[%s][%s] Successfully re-registered as bootnode with registry", dm.nodeRole.String(), dm.chainID)
					}
				case <-dm.stopChan:
					return
				}
			}
		}()
	}

	// --- Conditional Announce/Discover Loop ---
	if dm.clientOnly {
		log.Printf("[%s][%s] DiscoveryManager running in client-only mode. Skipping self-announce/discover loop.", dm.nodeRole.String(), dm.chainID)
		// Keep running to maintain DHT connection for lookups, but don't participate actively
		<-dm.stopChan // Wait for stop signal
		return
	}
	// --- End Conditional ---

	// --- Existing Announce/Discover Loop (Only runs if !dm.clientOnly) --- // Role added to log
	log.Printf("[%s][%s] DiscoveryManager running in full participant mode.", dm.nodeRole.String(), dm.chainID)
	// Announce our chain resource (for non-client-only nodes)
	if err := dm.AnnounceGenericResource(dm.chainID, DiscoveryResourceTypeChain); err != nil {
		log.Printf("[%s][%s] Failed to announce initial chain resource: %s", dm.nodeRole.String(), dm.chainID, err)
	}

	// Set up periodic re-announcements and health checks
	announceTicker := time.NewTicker(interval)
	defer announceTicker.Stop()

	healthTicker := time.NewTicker(interval / 2)
	defer healthTicker.Stop()

	for {
		select {
		case <-announceTicker.C:
			// Re-announce our resources periodically
			if err := dm.AnnounceGenericResource(dm.chainID, DiscoveryResourceTypeChain); err != nil {
				log.Printf("[%s][%s] Failed to re-announce chain resource: %s", dm.nodeRole.String(), dm.chainID, err)
			}

			// Try to discover new peers for our chain
			peersInfo, err := dm.FindGenericResource(dm.chainID, DiscoveryResourceTypeChain)
			if err != nil {
				if !strings.Contains(err.Error(), "no providers found") {
					log.Printf("[%s][%s] Peer discovery FindResource error: %v", dm.nodeRole.String(), dm.chainID, err)
				}
				continue
			}

			dm.mu.Lock()
			for _, discoveredPeerInfo := range peersInfo {
				peerID := discoveredPeerInfo.ID

				// --- Check Self and Existing Connection (includes mDNS) ---
				if peerID == dm.host.ID() {
					continue // Don't try to connect to self
				}
				// This check is crucial. If mDNS (or a previous attempt) already connected, skip.
				if dm.host.Network().Connectedness(peerID) == network.Connected {
					continue
				}

				// --- If Not Connected, Attempt Connection ---
				log.Printf("[%s][%s] Found potential peer %s via FindResource, and not currently connected.", dm.nodeRole.String(), dm.chainID, peerID.String())

				// 1. Try connecting using addresses directly from FindResource first (if any direct ones exist)
				var connectAddrInfo peer.AddrInfo
				filteredAddrs := filterDirectAddrs(discoveredPeerInfo.Addrs)
				connectionSucceeded := false

				if len(filteredAddrs) > 0 {
					log.Printf("[%s][%s] Attempting connection using direct addresses from FindResource for %s: %v", dm.nodeRole.String(), dm.chainID, peerID.String(), filteredAddrs)
					connectAddrInfo = peer.AddrInfo{ID: peerID, Addrs: filteredAddrs}
					connectCtx, connectCancel := context.WithTimeout(dm.ctx, 10*time.Second) // Shorter timeout for direct attempt
					err = dm.ConnectToPeer(connectAddrInfo, connectCtx)
					connectCancel()
					if err == nil {
						log.Printf("[%s][%s] Successfully connected to peer %s using FindResource addresses.", dm.nodeRole.String(), dm.chainID, peerID.String())
						connectionSucceeded = true
					} else {
						log.Printf("[%s][%s] Failed to connect to peer %s using FindResource addresses: %v", dm.nodeRole.String(), dm.chainID, peerID.String(), err)
						// Don't 'continue' yet, fall through to FindPeer
					}
				}

				// 2. If connection hasn't succeeded yet, try FindPeer as a fallback
				if !connectionSucceeded {
					log.Printf("[%s][%s] Querying DHT via FindPeer for %s...", dm.nodeRole.String(), dm.chainID, peerID.String())

					var findPeerAddrInfo peer.AddrInfo
					var findPeerErr error
					// Keep retries, but reduce attempts/timeout since mDNS should handle local
					const maxFindPeerAttempts = 2
					const findPeerRetryDelay = 3 * time.Second

					for attempt := 1; attempt <= maxFindPeerAttempts; attempt++ {
						findCtx, findCancel := context.WithTimeout(dm.ctx, 10*time.Second) // Shorter timeout per attempt
						findPeerAddrInfo, findPeerErr = dm.dht.FindPeer(findCtx, peerID)
						findCancel()
						if findPeerErr == nil {
							break // Success
						}
						log.Printf("[%s][%s] FindPeer attempt %d/%d failed for %s: %v.", dm.nodeRole.String(), dm.chainID, attempt, maxFindPeerAttempts, peerID.String(), findPeerErr)
						if attempt < maxFindPeerAttempts {
							select {
							case <-dm.ctx.Done():
								findPeerErr = dm.ctx.Err()
								goto afterRetryLoop
							case <-time.After(findPeerRetryDelay): // Wait
							}
						}
					}
				afterRetryLoop:

					if findPeerErr != nil {
						log.Printf("[%s][%s] FindPeer failed for %s after %d attempts. Skipping connection.", dm.nodeRole.String(), dm.chainID, peerID.String(), maxFindPeerAttempts)
						continue // Give up on this peer for this cycle
					}

					// Filter addresses found via FindPeer
					filteredAddrs = filterDirectAddrs(findPeerAddrInfo.Addrs)
					if len(filteredAddrs) == 0 {
						log.Printf("[%s][%s] No direct addresses found for peer %s even after FindPeer. Skipping connection.", dm.nodeRole.String(), dm.chainID, peerID.String())
						continue
					}

					// Attempt connection using FindPeer addresses
					log.Printf("[%s][%s] Attempting connection using direct addresses from FindPeer for %s: %v", dm.nodeRole.String(), dm.chainID, peerID.String(), filteredAddrs)
					connectAddrInfo = peer.AddrInfo{ID: peerID, Addrs: filteredAddrs}
					connectCtx, connectCancel := context.WithTimeout(dm.ctx, 15*time.Second) // Slightly longer timeout
					err = dm.ConnectToPeer(connectAddrInfo, connectCtx)
					connectCancel()

					if err != nil {
						log.Printf("[%s][%s] Failed to connect to discovered peer %s using FindPeer addresses: %v", dm.nodeRole.String(), dm.chainID, peerID.String(), err)
						continue // Give up if FindPeer addresses also failed
					}
					log.Printf("Successfully connected to discovered peer %s using FindPeer addresses.", peerID.String())
					connectionSucceeded = true
				}

				// --- Reflection Manager Update Logic (Only if connection succeeded or was already established) ---
				// if connectionSucceeded || dm.host.Network().Connectedness(peerID) == network.Connected {
				//    /* ... Your reflection update logic ... */
				// }
			}
			dm.mu.Unlock()

		case <-healthTicker.C:
			dm.mu.Lock()
			for url, client := range dm.clients {
				isActive := client.Ping()
				// Update blockchain manager's reflection status
				reflections := GetReflectionManager().GetReflections()
				for i, r := range reflections {
					if r.ReflectionAddress == url {
						reflections[i].IsActive = isActive
						reflections[i].LastSeen = time.Now()
						break
					}
				}
			}
			dm.mu.Unlock()

		case <-dm.stopChan:
			return
		}
	}
	// --- End Existing Loop ---
}

// ResolveURI resolves a KNIRVORACLE URI to peer information
// FindResource looks up a resource in the DHT using its type
func (dm *DiscoveryManager) FindResource(ctx context.Context, id string, resourceType types.DiscoveryResourceType) ([]peer.AddrInfo, error) {
	// Convert the resource type to string and use FindGenericResource
	return dm.FindGenericResource(id, resourceType)
}

// AnnounceMintedResource announces a newly minted resource to the DHT
func (dm *DiscoveryManager) AnnounceMintedResource(ctx context.Context, id string, resourceType types.DiscoveryResourceType) error {
	// Convert the resource type to string and use AnnounceGenericResource
	return dm.AnnounceGenericResource(id, resourceType)
}

func (dm *DiscoveryManager) ResolveURI(uri string) ([]peer.AddrInfo, error) {
	// Parse the URI
	if !strings.HasPrefix(uri, "agent://") {
		return nil, fmt.Errorf("invalid URI scheme: %s", uri)
	}

	// Extract the authority part (ID.ResourceType)
	parts := strings.SplitN(uri[8:], "/", 2)
	if len(parts) == 0 {
		return nil, fmt.Errorf("invalid URI format: %s", uri)
	}

	authority := parts[0]
	authorityParts := strings.Split(authority, ".")
	if len(authorityParts) != 2 {
		return nil, fmt.Errorf("invalid authority format: %s", authority)
	}

	id := authorityParts[0]
	resourceType := types.DiscoveryResourceType(authorityParts[1])

	// Validate resource type
	if resourceType != DiscoveryResourceTypeChain && resourceType != DiscoveryResourceTypeNRN {
		return nil, fmt.Errorf("invalid resource type: %s", resourceType)
	}

	// Find the resource in the DHT
	return dm.FindGenericResource(id, resourceType) // Assuming generic resource for now
}

// Close stops the discovery manager and cleans up resources
func (dm *DiscoveryManager) Close() {
	dm.closeOnce.Do(func() {
		log.Printf("[%s][%s] Closing Discovery Manager...", dm.nodeRole.String(), dm.chainID)

		// Set the closed flag first to prevent new operations
		dm.closeMu.Lock()
		if dm.closed {
			dm.closeMu.Unlock()
			log.Printf("[%s][%s] Discovery Manager already closed.", dm.nodeRole.String(), dm.chainID)
			return // Already closed
		}
		dm.closed = true
		dm.closeMu.Unlock()

		// Close stopChan first to signal all goroutines to stop
		// If this is a bootnode, deregister from the registry before shutting down
		if dm.IsBootnode {
			log.Printf("[%s][%s] Bootnode shutting down, attempting to deregister from registry...", dm.nodeRole.String(), dm.chainID)
			// Implement deregistration logic here - this could be a call to a DeregisterBootnodeFromRegistry method
			// For now, we'll just log that we're shutting down
			// TODO: Implement proper deregistration when registry API supports it
		}
		// This is especially important for bootnode registration goroutine
		dm.closeMu.Lock()
		if dm.stopChan != nil {
			select {
			case <-dm.stopChan:
				log.Printf("[%s][%s] Discovery Manager stopChan already closed.", dm.nodeRole.String(), dm.chainID)
			default:
				log.Printf("[%s][%s] Closing Discovery Manager stopChan...", dm.nodeRole.String(), dm.chainID)
				close(dm.stopChan)
				log.Printf("[%s][%s] Discovery Manager stopChan closed.", dm.nodeRole.String(), dm.chainID)
			}
		}
		dm.closeMu.Unlock()

		// Give goroutines a moment to react to stopChan closing
		// Increase the wait time for bootnode mode to ensure all goroutines have time to shut down
		if dm.IsBootnode {
			log.Printf("[%s][%s] Bootnode mode: waiting longer for goroutines to shut down...", dm.nodeRole.String(), dm.chainID)
			time.Sleep(500 * time.Millisecond)
		} else {
			time.Sleep(100 * time.Millisecond)
		}

		// Cancel the context to stop all ongoing operations
		if dm.cancel != nil {
			log.Printf("[%s][%s] Canceling Discovery Manager context...", dm.nodeRole.String(), dm.chainID)
			dm.cancel()
			log.Printf("[%s][%s] Discovery Manager context canceled.", dm.nodeRole.String(), dm.chainID)
		}

		// Close the DHT if it exists
		if dm.dht != nil {
			log.Printf("[%s][%s] Closing DHT...", dm.nodeRole.String(), dm.chainID)
			// Close the DHT
			if err := dm.dht.Close(); err != nil {
				log.Printf("[%s][%s] Error closing DHT: %v", dm.nodeRole.String(), dm.chainID, err)
			} else {
				log.Printf("[%s][%s] DHT closed successfully.", dm.nodeRole.String(), dm.chainID)
			}
		}

		// Close the libp2p host last
		if dm.host != nil {
			log.Printf("[%s][%s] Closing libp2p host...", dm.nodeRole.String(), dm.chainID)
			if err := dm.host.Close(); err != nil {
				log.Printf("[%s][%s] Error closing libp2p host: %v", dm.nodeRole.String(), dm.chainID, err)
			} else {
				log.Printf("[%s][%s] Libp2p host closed successfully.", dm.nodeRole.String(), dm.chainID)
			}
		}

		log.Printf("[%s][%s] Discovery Manager closed successfully.", dm.nodeRole.String(), dm.chainID)
	})
}

// GetMultiaddrFromURI extracts a multiaddr from a KNIRVORACLE URI
func GetMultiaddrFromURI(uri string) (string, error) {
	// This is a placeholder for the actual implementation
	// In a real implementation, this would resolve the URI using the DHT
	// and return a valid multiaddr

	// For now, we'll just convert the old format to a multiaddr
	if strings.HasPrefix(uri, "chain://") {
		// Extract the chain ID
		parts := strings.Split(uri[8:], ".")
		if len(parts) < 1 {
			return "", fmt.Errorf("invalid chain URI format: %s", uri)
		}

		chainID := parts[0]
		// This would be replaced with actual DHT lookup
		return fmt.Sprintf("/ip4/127.0.0.1/tcp/5000/p2p/%s", chainID), nil
	}

	return "", fmt.Errorf("unsupported URI format: %s", uri)
}

// GenerateResourceCID creates a CID for a resource
func GenerateResourceCID(id string, resourceType types.DiscoveryResourceType) (cid.Cid, error) {
	resourceKey := fmt.Sprintf("%s.%s", id, resourceType)
	h := sha256.New()
	h.Write([]byte(resourceKey))
	mh, err := multihash.Encode(h.Sum(nil), multihash.SHA2_256)
	if err != nil {
		return cid.Cid{}, err
	}

	return cid.NewCidV1(cid.Raw, mh), nil
}

// ResourceKeyToString converts a resource key to a string representation
func ResourceKeyToString(id string, resourceType types.DiscoveryResourceType) string {
	return fmt.Sprintf("%s.%s", id, resourceType)
}

// StringToResourceKey parses a string into id and resource type
func StringToResourceKey(key string) (string, types.DiscoveryResourceType, error) {
	parts := strings.Split(key, ".")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid resource key format: %s", key)
	}

	return parts[0], types.DiscoveryResourceType(parts[1]), nil
}

// PeerIDToHex converts a peer.ID to a hex string
func PeerIDToHex(id peer.ID) string {
	return hex.EncodeToString([]byte(id))
}

// FetchBootnodesFromRegistry fetches the list of bootnodes from the registry
func FetchBootnodesFromRegistry(requestingRole, requestingChainID string) (map[string]BootnodeInfo, error) { // Added params for logging
	// Set up the HTTP client with timeout
	client := &http.Client{
		Timeout: 120 * time.Second,
	}

	// Registry endpoint for listing all nodes
	registryURL := fmt.Sprintf("%s/nodes", BootnodeRegistryURL)

	// Create the request
	req, err := http.NewRequest("GET", registryURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	req.Header.Set("User-Agent", "KNIRVORACLE-Bootnode/1.0")

	// Send the request
	log.Printf("[%s][%s] Fetching bootnodes from registry: %s", requestingRole, requestingChainID, registryURL)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send registry request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check the status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry returned non-OK status: %d - %s", resp.StatusCode, string(body))
	}

	// Parse the response
	var bootnodes map[string]BootnodeInfo
	if err := json.Unmarshal(body, &bootnodes); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return bootnodes, nil
}

// RegisterBootnodeWithRegistry registers this node as a bootnode with the registry
func (dm *DiscoveryManager) RegisterBootnodeWithRegistry() error {
	if !dm.IsBootnode {
		return fmt.Errorf("[%s][%s] cannot register with registry: node is not a bootnode", dm.nodeRole.String(), dm.chainID)
	}

	var publicIP string

	// 1. Try to get IP from config first
	if dm.cfg != nil && dm.cfg.PublicIPInfo != nil {
		if ip, ok := dm.cfg.PublicIPInfo["ip"].(string); ok && ip != "" {
			log.Printf("[%s][%s] Using public IP from config for registry: %s", dm.nodeRole.String(), dm.chainID, ip)
			publicIP = ip
		}
	}
	// 2. If not in IP Config, then make a new call to IPInfo.io
	// Fetch and store public IP info
	if err := fetchAndStorePublicIPInfo(dm.cfg, nodeRole); err != nil {
		log.Printf("Warning: Could not access public IP information: %v", err)
		// This is not a fatal error, the application can continue
	}

	// Register with the registry
	// Use RegisterWithRegistry directly, as RegisterWithNodeRegistry is specific to install.go's flow
	return RegisterWithRegistry(BootnodeRegistryURL, dm.chainID, dm.p2pPort, publicIP, dm.host.ID().String())
}

// Use the DiscoverPublicAddress function from install.go

// Helper function to filter out non-direct addresses
func filterDirectAddrs(addrs []multiaddr.Multiaddr) []multiaddr.Multiaddr {
	filtered := make([]multiaddr.Multiaddr, 0, len(addrs))
	for _, addr := range addrs {
		if !strings.Contains(addr.String(), "/p2p-circuit") { // Basic filter for circuit relay
			// You could add more filtering here if needed (e.g., remove WebRTC, WebTransport if not desired)
			filtered = append(filtered, addr)
		}
	}
	return filtered
}

// HexToPeerID converts a hex string to a peer.ID
func HexToPeerID(hexID string) (peer.ID, error) {
	bytes, err := hex.DecodeString(hexID)
	if err != nil {
		return "", err
	}

	return peer.ID(bytes), nil
}
