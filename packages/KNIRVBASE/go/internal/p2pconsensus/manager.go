package p2pconsensus

import (
	"context"
	"fmt"
	"log"
	"sync"
)

// P2PConsensusManager is the top-level orchestrator for P2P consensus in KNIRVBASE.
// It detects gateway presence on init and routes to GatewayProxy or StandaloneConsensus.
type P2PConsensusManager struct {
	config ConsensusConfig
	mode   string

	gatewayProxy *GatewayProxy
	standalone   *StandaloneConsensus
	socketServer *SocketServer

	handler EventHandler
	mu      sync.RWMutex
	running bool
}

// NewP2PConsensusManager creates a new P2PConsensusManager and selects the mode.
// socketPath is the Unix socket path for gateway callbacks (empty for standalone).
func NewP2PConsensusManager(cfg ConsensusConfig, socketPath string, handler EventHandler) *P2PConsensusManager {
	mode := ResolveMode(cfg)

	m := &P2PConsensusManager{
		config:  cfg,
		mode:    mode,
		handler: handler,
	}

	switch mode {
	case "gateway":
		m.gatewayProxy = NewGatewayProxy(cfg.GatewayURL, cfg.NetworkID, socketPath, cfg.NetworkSecret)
		if handler != nil {
			m.socketServer = NewSocketServer(socketPath, &consensusHandlerBridge{m: m, handler: handler}, m.config.NetworkID, m.config.NetworkSecret)
		}
	case "standalone":
		m.standalone = NewStandaloneConsensus(cfg)
	}

	return m
}

// Start begins consensus operations.
func (m *P2PConsensusManager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.running {
		return fmt.Errorf("already running")
	}

	switch m.mode {
	case "gateway":
		if m.gatewayProxy == nil {
			return fmt.Errorf("gateway proxy not initialized")
		}
		// Register callback socket with gateway
		if err := m.gatewayProxy.Register(); err != nil {
			log.Printf("[p2pconsensus] Warning: gateway registration failed: %v (falling back to standalone)", err)
			m.mode = "standalone"
			m.standalone = NewStandaloneConsensus(m.config)
			return m.standalone.Start(ctx)
		}
		// Start socket server for gateway callbacks
		if m.socketServer != nil {
			go func() {
				if err := m.socketServer.Serve(); err != nil {
					log.Printf("[p2pconsensus] Socket server error: %v", err)
				}
			}()
		}
	case "standalone":
		if m.standalone == nil {
			m.standalone = NewStandaloneConsensus(m.config)
		}
		if err := m.standalone.Start(ctx); err != nil {
			return fmt.Errorf("standalone start: %w", err)
		}
	}

	m.running = true
	return nil
}

// Stop shuts down consensus operations.
func (m *P2PConsensusManager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.running {
		return nil
	}
	m.running = false

	if m.socketServer != nil {
		m.socketServer.Stop(context.Background())
	}
	if m.gatewayProxy != nil {
		m.gatewayProxy.Close()
	}
	if m.standalone != nil {
		m.standalone.Stop()
	}
	return nil
}

// Status returns the current consensus status.
func (m *P2PConsensusManager) Status() ConsensusStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	switch m.mode {
	case "gateway":
		if m.standalone != nil && m.standalone.running {
			return m.standalone.Status()
		}
		return ConsensusStatus{
			Mode:      "gateway",
			NetworkID: m.config.NetworkID,
			PeerCount: 0,
			Running:   m.running,
		}
	case "standalone":
		if m.standalone != nil {
			return m.standalone.Status()
		}
	}
	return ConsensusStatus{Mode: "disabled", Running: false}
}

// Peers returns the list of connected peers.
func (m *P2PConsensusManager) Peers() []PeerInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	switch m.mode {
	case "gateway":
		if m.gatewayProxy != nil {
			peers, err := m.gatewayProxy.DiscoverPeers("")
			if err == nil {
				return peers
			}
		}
	case "standalone":
		if m.standalone != nil {
			return m.standalone.Peers()
		}
	}
	return nil
}

// Enabled returns whether the consensus manager is active.
func (m *P2PConsensusManager) Enabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.Enabled
}

// SetEnabled toggles consensus on or off.
func (m *P2PConsensusManager) SetEnabled(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config.Enabled = enabled
}

// PublishOperation sends an operation through the active consensus channel.
func (m *P2PConsensusManager) PublishOperation(ctx context.Context, op OperationEnvelope) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	switch m.mode {
	case "gateway":
		if m.gatewayProxy != nil {
			return m.gatewayProxy.PublishOperation(ctx, op)
		}
		return fmt.Errorf("gateway proxy not initialized")
	case "standalone":
		if m.standalone != nil {
			return m.standalone.BroadcastOperation(ctx, op)
		}
		return fmt.Errorf("standalone not initialized")
	default:
		return fmt.Errorf("consensus disabled")
	}
}

// consensusHandlerBridge adapts EventHandler to ConsensusHandler for SocketServer.
type consensusHandlerBridge struct {
	m       *P2PConsensusManager
	handler EventHandler
}

func (b *consensusHandlerBridge) HandleOperation(op OperationEnvelope) error {
	return b.handler.OnOperationReceived(op)
}

func (b *consensusHandlerBridge) HandleSyncRequest(req SyncRequest) (*SyncResponse, error) {
	return b.handler.OnSyncRequestReceived(req)
}
