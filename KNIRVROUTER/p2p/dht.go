// p2p/dht.go
package p2p

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/multiformats/go-multiaddr"
	"github.com/multiformats/go-multihash"
)

// DHT service discovery constants
const (
	DiscoveryInterval  = time.Hour
	DiscoveryNamespace = "knirvchain"
)

// DHTManager handles the Distributed Hash Table functionality
type DHTManager struct {
	host           host.Host
	kadDHT         *dht.IpfsDHT
	ctx            context.Context
	cancel         context.CancelFunc
	bootstrapPeers []peer.AddrInfo
	mutex          sync.RWMutex
	server         *Server
}

// NewDHTManager creates a new DHT manager
func NewDHTManager(server *Server, bootstrapAddrs []string) (*DHTManager, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Generate or load a persistent key pair
	priv, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	// Parse bootstrap addresses
	bootstrapPeers, err := parseBootstrapPeers(bootstrapAddrs)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to parse bootstrap peers: %w", err)
	}

	// Create a new libp2p Host
	h, err := libp2p.New(
		libp2p.Identity(priv),
		libp2p.ListenAddrStrings(
			"/ip4/0.0.0.0/tcp/9000",
			"/ip6/::/tcp/9000",
		),
		libp2p.EnableNATService(),
		libp2p.EnableAutoRelay(),
		libp2p.EnableHolePunching(),
	)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create libp2p host: %w", err)
	}

	// Log the host's addresses
	log.Println("DHT node started with addresses:")
	for _, addr := range h.Addrs() {
		log.Printf("  %s/p2p/%s", addr, h.ID().String())
	}

	// Create a DHT instance in server mode
	kadDHT, err := dht.New(ctx, h, dht.Mode(dht.ModeServer))
	if err != nil {
		h.Close()
		cancel()
		return nil, fmt.Errorf("failed to create DHT: %w", err)
	}

	// Create the DHT manager
	manager := &DHTManager{
		host:           h,
		kadDHT:         kadDHT,
		ctx:            ctx,
		cancel:         cancel,
		bootstrapPeers: bootstrapPeers,
		server:         server,
	}

	return manager, nil
}

// Start initializes the DHT and connects to bootstrap peers
func (dm *DHTManager) Start() error {
	// Bootstrap the DHT
	if err := dm.kadDHT.Bootstrap(dm.ctx); err != nil {
		return fmt.Errorf("failed to bootstrap DHT: %w", err)
	}

	// Connect to bootstrap peers
	var wg sync.WaitGroup
	for _, peerInfo := range dm.bootstrapPeers {
		wg.Add(1)
		go func(peerInfo peer.AddrInfo) {
			defer wg.Done()
			if err := dm.host.Connect(dm.ctx, peerInfo); err != nil {
				log.Printf("Failed to connect to bootstrap peer %s: %v", peerInfo.ID, err)
			} else {
				log.Printf("Connected to bootstrap peer: %s", peerInfo.ID)
			}
		}(peerInfo)
	}
	wg.Wait()

	// Setup local mDNS discovery
	discovery := mdns.NewMdnsService(dm.host, DiscoveryNamespace, &discoveryNotifee{dm: dm})
	if err := discovery.Start(); err != nil {
		return fmt.Errorf("failed to start mDNS discovery: %w", err)
	}

	// Start periodic DHT record refresh
	go dm.refreshLoop()

	return nil
}

// Stop shuts down the DHT manager
func (dm *DHTManager) Stop() {
	dm.cancel()
	if err := dm.host.Close(); err != nil {
		log.Printf("Error closing libp2p host: %v", err)
	}
}

// refreshLoop periodically refreshes DHT records
func (dm *DHTManager) refreshLoop() {
	ticker := time.NewTicker(DiscoveryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-dm.ctx.Done():
			return
		case <-ticker.C:
			// Refresh DHT records
			if err := dm.AnnounceChain(); err != nil {
				log.Printf("Failed to refresh DHT records: %v", err)
			}
		}
	}
}

// AnnounceChain announces this node's chain to the DHT
func (dm *DHTManager) AnnounceChain() error {
	// Get the chain ID from the server
	chainID := dm.server.GetChainID()
	if chainID == "" {
		return fmt.Errorf("chain ID not available")
	}

	// Create a CID from the chain ID
	cid, err := createCIDFromChainID(chainID)
	if err != nil {
		return fmt.Errorf("failed to create CID: %w", err)
	}

	// Announce that this node provides the chain
	if err := dm.kadDHT.Provide(dm.ctx, cid, true); err != nil {
		return fmt.Errorf("failed to announce chain: %w", err)
	}

	log.Printf("Announced chain %s to DHT", chainID)
	return nil
}

// FindChainProviders finds nodes that provide a specific chain
func (dm *DHTManager) FindChainProviders(chainID string) ([]peer.AddrInfo, error) {
	// Create a CID from the chain ID
	cid, err := createCIDFromChainID(chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to create CID: %w", err)
	}

	// Find providers for the chain
	ctx, cancel := context.WithTimeout(dm.ctx, 30*time.Second)
	defer cancel()

	providers := dm.kadDHT.FindProvidersAsync(ctx, cid, 20)

	var results []peer.AddrInfo
	for p := range providers {
		results = append(results, p)
	}

	return results, nil
}

// ResolveKnirvURI resolves a knirv:// URI to peer addresses
func (dm *DHTManager) ResolveKnirvURI(uri string) ([]peer.AddrInfo, error) {
	// Parse the URI
	if !strings.HasPrefix(uri, "knirv://") {
		return nil, fmt.Errorf("invalid URI scheme, expected knirv://")
	}

	// Extract the ID and resource type
	authority := strings.TrimPrefix(uri, "knirv://")
	parts := strings.SplitN(authority, "/", 2)
	hostPart := parts[0]

	// Split the host part into ID and resource type
	idParts := strings.Split(hostPart, ".")
	if len(idParts) < 2 {
		return nil, fmt.Errorf("invalid URI format, expected knirv://<ID>.<ResourceType>/...")
	}

	id := idParts[0]
	resourceType := idParts[1]

	// Handle different resource types
	switch resourceType {
	case "chain":
		// Find providers for the chain
		return dm.FindChainProviders(id)
	case "nrn":
		// Find providers for the NRN asset
		// For now, we'll use the same method as for chains
		return dm.FindChainProviders(id)
	case "node":
		// Find a specific node by ID
		// This is a direct lookup of a peer ID
		peerID, err := peer.Decode(id)
		if err != nil {
			return nil, fmt.Errorf("invalid node ID: %w", err)
		}

		// Find the peer's addresses
		peerInfo, err := dm.kadDHT.FindPeer(dm.ctx, peerID)
		if err != nil {
			return nil, fmt.Errorf("peer not found: %w", err)
		}

		return []peer.AddrInfo{peerInfo}, nil
	default:
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}
}

// parseBootstrapPeers converts string multiaddrs to AddrInfo
func parseBootstrapPeers(addrs []string) ([]peer.AddrInfo, error) {
	maddrs := make([]multiaddr.Multiaddr, len(addrs))
	for i, addr := range addrs {
		var err error
		maddrs[i], err = multiaddr.NewMultiaddr(addr)
		if err != nil {
			return nil, err
		}
	}
	return peer.AddrInfosFromP2pAddrs(maddrs...)
}

// createCIDFromChainID creates a Content ID from a chain ID
func createCIDFromChainID(chainID string) (cid.Cid, error) {
	// Create a CID v1 with raw codec
	pref := cid.Prefix{
		Version:  1,
		Codec:    cid.Raw,
		MhType:   multihash.SHA2_256,
		MhLength: -1, // default length
	}

	// Create a new CID from the chain ID bytes
	newCID, err := pref.Sum([]byte(chainID))
	if err != nil {
		return cid.Undef, fmt.Errorf("failed to create CID: %w", err)
	}
	return newCID, nil
}

// discoveryNotifee gets notified when we find a new peer via mDNS discovery
type discoveryNotifee struct {
	dm *DHTManager
}

// HandlePeerFound connects to peers discovered via mDNS discovery
func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	log.Printf("Discovered new peer %s via mDNS", pi.ID.String())
	err := n.dm.host.Connect(n.dm.ctx, pi)
	if err != nil {
		log.Printf("Error connecting to mDNS peer %s: %v", pi.ID.String(), err)
	}
}
