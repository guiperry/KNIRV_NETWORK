//go:build test

package p2p

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
)

// NewDHTManager creates a new DHT manager for KNIRVGRAPH for test builds
// This test-specific version explicitly avoids any problematic libp2p options.
func NewDHTManager(serviceID, chainID string, bootstrapPeers []string, enableAutoRelay bool) (*DHTManager, error) {
	log.Printf("NewDHTManager (TEST VERSION) called with enableAutoRelay: %t", enableAutoRelay)
	ctx, cancel := context.WithCancel(context.Background())

	// Generate a new key pair for this node
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.Ed25519, -1, rand.Reader)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	opts := []libp2p.Option{
		libp2p.Identity(priv),
		libp2p.ListenAddrStrings(
			"/ip4/0.0.0.0/tcp/9001",
			"/ip6/::/tcp/9001",
		),
		// Explicitly disable relay functionality for tests to avoid panics
		libp2p.DisableRelay(),
		// Explicitly disable NAT and Hole Punching for a cleaner test environment
		// libp2p.NATPortMap(false), // This is not a valid option
		// libp2p.EnableRelayService(false), // This is not a valid option
	}
	
	// Create a new libp2p Host
	h, err := libp2p.New(opts...)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create libp2p host: %w", err)
	}

	// Log the host's addresses
	log.Println("KNIRVGRAPH DHT node (TEST VERSION) started with addresses:")
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

	// Create a PubSub instance
	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		h.Close()
		cancel()
		return nil, fmt.Errorf("failed to create pubsub: %w", err)
	}

	// Parse bootstrap peers
	var bootstrapPeerInfos []peer.AddrInfo
	for _, peerAddr := range bootstrapPeers {
		if peerAddr == "" {
			continue
		}
		addr, err := multiaddr.NewMultiaddr(peerAddr)
		if err != nil {
			log.Printf("Invalid bootstrap peer address %s: %v", peerAddr, err)
			continue
		}
		peerInfo, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			log.Printf("Failed to parse bootstrap peer %s: %v", peerAddr, err)
			continue
		}
		bootstrapPeerInfos = append(bootstrapPeerInfos, *peerInfo)
	}

	return &DHTManager{
		host:           h,
		kadDHT:         kadDHT,
		pubsub:         ps,
		ctx:            ctx,
		cancel:         cancel,
		bootstrapPeers: bootstrapPeerInfos,
		serviceID:      serviceID,
		chainID:        chainID,
		networkPaused:  false,
	}, nil
}

