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
	peers         map[PeerID]*Peer
	dhtEnabled    bool
	gossipEnabled bool
	paused        bool
	logger        *zap.Logger
	ctx           context.Context
	cancel        context.CancelFunc
	mu            sync.RWMutex
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
		listenAddr:    config.ListenAddr,
		peers:         make(map[PeerID]*Peer),
		dhtEnabled:    config.EnableDHT,
		gossipEnabled: config.EnableGossip,
		paused:        false,
		logger:        logger,
		ctx:           ctx,
		cancel:        cancel,
	}

	return pm
}

// Start starts the P2P manager
func (pm *P2PManager) Start() error {
	pm.logger.Info("Starting P2P manager",
		zap.String("listen_addr", pm.listenAddr),
		zap.String("transport_owner", "KNIRVGATEWAY"),
	)
	if pm.dhtEnabled || pm.gossipEnabled {
		return fmt.Errorf("Oracle-local DHT/GossipSub is disabled; configure KNIRVGATEWAY as the network transport")
	}

	return nil
}

// Stop stops the P2P manager
func (pm *P2PManager) Stop() error {
	pm.logger.Info("Stopping P2P manager")

	pm.cancel()

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

	return fmt.Errorf("Oracle cannot publish topic %q directly; publish through KNIRVGATEWAY", topic)
}

// Subscribe subscribes to a topic
func (pm *P2PManager) Subscribe(topic string, handler func([]byte) error) error {
	return fmt.Errorf("Oracle cannot subscribe to topic %q directly; subscriptions are owned by KNIRVGATEWAY", topic)
}

// FindPeers finds peers using DHT
func (pm *P2PManager) FindPeers(count int) ([]*Peer, error) {
	return nil, fmt.Errorf("Oracle cannot query peers directly; query KNIRVGATEWAY /dht/peers")
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
		"transport_owner": "KNIRVGATEWAY",
	}

	return stats
}
