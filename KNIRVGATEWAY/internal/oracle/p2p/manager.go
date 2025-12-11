package p2p

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// PeerID represents a peer identifier
type PeerID string

// Peer represents a network peer
type Peer struct {
	ID        PeerID    `json:"id"`
	Addresses []string  `json:"addresses"`
	Connected bool      `json:"connected"`
	LastSeen  time.Time `json:"last_seen"`
}

// P2PManager manages peer-to-peer networking
// In production, this would use github.com/libp2p/go-libp2p
type P2PManager struct {
	listenAddr    string
	bootstrapPeers []string
	peers          map[PeerID]*Peer
	dht            *DHTManager
	gossip         *GossipManager
	paused         bool
	logger         *zap.Logger
	ctx            context.Context
	cancel         context.CancelFunc
	mu             sync.RWMutex
}

// P2PConfig contains P2P configuration
type P2PConfig struct {
	ListenAddr     string   `json:"listen_addr"`
	BootstrapPeers []string `json:"bootstrap_peers"`
	EnableDHT      bool     `json:"enable_dht"`
	EnableGossip   bool     `json:"enable_gossip"`
}

// NewP2PManager creates a new P2P manager
func NewP2PManager(config *P2PConfig, logger *zap.Logger) *P2PManager {
	ctx, cancel := context.WithCancel(context.Background())

	pm := &P2PManager{
		listenAddr:     config.ListenAddr,
		bootstrapPeers: config.BootstrapPeers,
		peers:          make(map[PeerID]*Peer),
		paused:         false,
		logger:         logger,
		ctx:            ctx,
		cancel:         cancel,
	}

	// Initialize DHT if enabled
	if config.EnableDHT {
		pm.dht = NewDHTManager(logger)
	}

	// Initialize GossipSub if enabled
	if config.EnableGossip {
		pm.gossip = NewGossipManager(logger)
	}

	return pm
}

// Start starts the P2P manager
func (pm *P2PManager) Start() error {
	pm.logger.Info("Starting P2P manager",
		zap.String("listen_addr", pm.listenAddr),
		zap.Int("bootstrap_peers", len(pm.bootstrapPeers)),
	)

	// Start DHT
	if pm.dht != nil {
		if err := pm.dht.Start(); err != nil {
			return fmt.Errorf("failed to start DHT: %w", err)
		}
	}

	// Start GossipSub
	if pm.gossip != nil {
		if err := pm.gossip.Start(); err != nil {
			return fmt.Errorf("failed to start GossipSub: %w", err)
		}
	}

	// Connect to bootstrap peers
	go pm.connectToBootstrapPeers()

	return nil
}

// Stop stops the P2P manager
func (pm *P2PManager) Stop() error {
	pm.logger.Info("Stopping P2P manager")

	pm.cancel()

	// Stop DHT
	if pm.dht != nil {
		pm.dht.Stop()
	}

	// Stop GossipSub
	if pm.gossip != nil {
		pm.gossip.Stop()
	}

	return nil
}

// Pause pauses P2P networking
func (pm *P2PManager) Pause() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.paused = true
	pm.logger.Info("P2P networking paused")

	return nil
}

// Resume resumes P2P networking
func (pm *P2PManager) Resume() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.paused = false
	pm.logger.Info("P2P networking resumed")

	return nil
}

// IsPaused returns whether P2P is paused
func (pm *P2PManager) IsPaused() bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	return pm.paused
}

// AddPeer adds a peer to the manager
func (pm *P2PManager) AddPeer(peer *Peer) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.paused {
		return fmt.Errorf("P2P is paused")
	}

	pm.peers[peer.ID] = peer
	peer.LastSeen = time.Now()

	pm.logger.Debug("Peer added",
		zap.String("peer_id", string(peer.ID)),
		zap.Strings("addresses", peer.Addresses),
	)

	return nil
}

// RemovePeer removes a peer from the manager
func (pm *P2PManager) RemovePeer(peerID PeerID) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	delete(pm.peers, peerID)

	pm.logger.Debug("Peer removed",
		zap.String("peer_id", string(peerID)),
	)

	return nil
}

// GetPeer retrieves a peer by ID
func (pm *P2PManager) GetPeer(peerID PeerID) (*Peer, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	peer, exists := pm.peers[peerID]
	if !exists {
		return nil, fmt.Errorf("peer not found: %s", peerID)
	}

	return peer, nil
}

// ListPeers returns all connected peers
func (pm *P2PManager) ListPeers() []*Peer {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	peers := make([]*Peer, 0, len(pm.peers))
	for _, peer := range pm.peers {
		if peer.Connected {
			peers = append(peers, peer)
		}
	}

	return peers
}

// GetPeerCount returns the number of connected peers
func (pm *P2PManager) GetPeerCount() int {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	count := 0
	for _, peer := range pm.peers {
		if peer.Connected {
			count++
		}
	}

	return count
}

// Broadcast broadcasts a message to all connected peers
func (pm *P2PManager) Broadcast(topic string, data []byte) error {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if pm.paused {
		return fmt.Errorf("P2P is paused")
	}

	if pm.gossip != nil {
		return pm.gossip.Publish(topic, data)
	}

	pm.logger.Debug("Message broadcast",
		zap.String("topic", topic),
		zap.Int("size", len(data)),
	)

	return nil
}

// Subscribe subscribes to a topic
func (pm *P2PManager) Subscribe(topic string, handler func([]byte) error) error {
	if pm.gossip != nil {
		return pm.gossip.Subscribe(topic, handler)
	}

	return fmt.Errorf("GossipSub not enabled")
}

// FindPeers finds peers using DHT
func (pm *P2PManager) FindPeers(count int) ([]*Peer, error) {
	if pm.dht != nil {
		return pm.dht.FindPeers(count)
	}

	return nil, fmt.Errorf("DHT not enabled")
}

// connectToBootstrapPeers connects to bootstrap peers
func (pm *P2PManager) connectToBootstrapPeers() {
	for _, addr := range pm.bootstrapPeers {
		peer := &Peer{
			ID:        PeerID(addr),
			Addresses: []string{addr},
			Connected: true,
			LastSeen:  time.Now(),
		}

		if err := pm.AddPeer(peer); err != nil {
			pm.logger.Warn("Failed to add bootstrap peer",
				zap.String("addr", addr),
				zap.Error(err),
			)
		}
	}
}

// GetStats returns P2P statistics
func (pm *P2PManager) GetStats() map[string]interface{} {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	connectedCount := 0
	for _, peer := range pm.peers {
		if peer.Connected {
			connectedCount++
		}
	}

	stats := map[string]interface{}{
		"total_peers":     len(pm.peers),
		"connected_peers": connectedCount,
		"paused":          pm.paused,
		"listen_addr":     pm.listenAddr,
	}

	if pm.dht != nil {
		stats["dht"] = pm.dht.GetStats()
	}

	if pm.gossip != nil {
		stats["gossip"] = pm.gossip.GetStats()
	}

	return stats
}
