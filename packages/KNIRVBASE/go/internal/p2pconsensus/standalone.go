package p2pconsensus

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// StandaloneConsensus manages direct P2P consensus without the gateway.
// In a full implementation this would use libp2p; this provides the structure.
type StandaloneConsensus struct {
	config    ConsensusConfig
	peers     map[string]*PeerInfo
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	running   bool
	startedAt time.Time
}

// NewStandaloneConsensus creates a new standalone consensus manager.
func NewStandaloneConsensus(cfg ConsensusConfig) *StandaloneConsensus {
	ctx, cancel := context.WithCancel(context.Background())
	return &StandaloneConsensus{
		config:    cfg,
		peers:     make(map[string]*PeerInfo),
		ctx:       ctx,
		cancel:    cancel,
		startedAt: time.Now(),
	}
}

// Start initializes the standalone consensus mode.
func (s *StandaloneConsensus) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return fmt.Errorf("already running")
	}
	s.running = true
	s.startedAt = time.Now()
	// In a full implementation, this would initialize libp2p host,
	// DHT, and GossipSub, then join the {networkID}.ops topic
	return nil
}

// Stop shuts down the standalone consensus mode.
func (s *StandaloneConsensus) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return nil
	}
	s.running = false
	s.cancel()
	return nil
}

// Status returns the current status.
func (s *StandaloneConsensus) Status() ConsensusStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return ConsensusStatus{
		Mode:      "standalone",
		NetworkID: s.config.NetworkID,
		PeerCount: len(s.peers),
		Running:   s.running,
	}
}

// Peers returns the list of connected peers.
func (s *StandaloneConsensus) Peers() []PeerInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]PeerInfo, 0, len(s.peers))
	for _, p := range s.peers {
		result = append(result, *p)
	}
	return result
}

// BroadcastOperation publishes an operation to the network.
func (s *StandaloneConsensus) BroadcastOperation(ctx context.Context, op OperationEnvelope) error {
	// In a full implementation, this would publish to GossipSub topic {networkID}.ops
	return nil
}
