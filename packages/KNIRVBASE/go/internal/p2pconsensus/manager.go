package p2pconsensus

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

// P2PConsensusManager is the top-level orchestrator for P2P consensus in KNIRVBASE.
// It detects gateway presence on init and routes to GatewayProxy or StandaloneConsensus.
type P2PConsensusManager struct {
	config ConsensusConfig
	mode   string

	gatewayProxy *GatewayProxy
	standalone   *StandaloneConsensus
	socketServer *SocketServer

	handler       EventHandler
	mu            sync.RWMutex
	running       bool
	monitorCancel context.CancelFunc
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
	m.config.SocketPath = socketPath

	switch mode {
	case "gateway":
		m.gatewayProxy = NewGatewayProxy(cfg.GatewayURL, cfg.NetworkID, socketPath, cfg.NetworkSecret)
		if handler != nil && socketPath != "" {
			m.socketServer = NewSocketServer(socketPath, &consensusHandlerBridge{m: m, handler: handler}, m.config.NetworkID, m.config.NetworkSecret)
		}
	case "standalone":
		m.standalone = NewStandaloneConsensus(cfg, handler)
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
		// Start the callback listener before registering it so the gateway cannot
		// publish into a socket that is not accepting connections yet.
		if m.socketServer != nil {
			go func() {
				if err := m.socketServer.Serve(); err != nil {
					log.Printf("[p2pconsensus] Socket server error: %v", err)
				}
			}()
			deadline := time.Now().Add(2 * time.Second)
			for time.Now().Before(deadline) {
				if _, err := os.Stat(m.config.SocketPath); err == nil {
					break
				}
				time.Sleep(10 * time.Millisecond)
			}
		}
		// Register callback socket with gateway
		if err := m.gatewayProxy.Register(); err != nil {
			log.Printf("[p2pconsensus] Warning: gateway registration failed: %v (falling back to standalone)", err)
			m.mode = "standalone"
			m.standalone = NewStandaloneConsensus(m.config, m.handler)
			if err := m.standalone.Start(ctx); err != nil {
				return err
			}
			m.running = true
			return nil
		}
	case "standalone":
		if m.standalone == nil {
			m.standalone = NewStandaloneConsensus(m.config, m.handler)
		}
		if err := m.standalone.Start(ctx); err != nil {
			return fmt.Errorf("standalone start: %w", err)
		}
	}

	m.running = true
	if m.mode == "gateway" {
		monitorCtx, cancel := context.WithCancel(ctx)
		m.monitorCancel = cancel
		go m.monitorGateway(monitorCtx)
	}
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
	if m.monitorCancel != nil {
		m.monitorCancel()
		m.monitorCancel = nil
	}

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

func (m *P2PConsensusManager) monitorGateway(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	failures := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.mu.RLock()
			mode, proxy, cfg, running := m.mode, m.gatewayProxy, m.config, m.running
			m.mu.RUnlock()
			if !running {
				return
			}
			if mode == "gateway" {
				if proxy != nil && proxy.Health() {
					failures = 0
					continue
				}
				failures++
				if failures < 3 {
					continue
				}
				m.mu.Lock()
				if m.mode == "gateway" {
					m.mode = "standalone"
					m.standalone = NewStandaloneConsensus(cfg, m.handler)
					standalone := m.standalone
					m.mu.Unlock()
					if err := standalone.Start(ctx); err != nil {
						log.Printf("[p2pconsensus] standalone failover failed: %v", err)
					}
				} else {
					m.mu.Unlock()
				}
				failures = 0
			} else if mode == "standalone" && cfg.GatewayURL != "" && detectGateway(cfg.GatewayURL, 2*time.Second) {
				m.mu.Lock()
				if m.mode == "standalone" {
					proxy = NewGatewayProxy(cfg.GatewayURL, cfg.NetworkID, cfg.SocketPath, cfg.NetworkSecret)
					if err := proxy.Register(); err == nil {
						if m.standalone != nil {
							_ = m.standalone.Stop()
						}
						m.gatewayProxy, m.mode = proxy, "gateway"
					}
				}
				m.mu.Unlock()
			}
		}
	}
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

// RequestSync sends a network-scoped sync request through the active transport.
func (m *P2PConsensusManager) RequestSync(ctx context.Context, req SyncRequest) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if !m.config.Enabled {
		return fmt.Errorf("consensus disabled")
	}
	if req.NetworkID == "" {
		req.NetworkID = m.config.NetworkID
	}
	if m.mode == "standalone" && m.standalone != nil {
		return m.standalone.RequestSync(ctx, req)
	}
	if m.mode == "gateway" && m.gatewayProxy != nil {
		return m.gatewayProxy.RequestSync(ctx, req)
	}
	return fmt.Errorf("consensus transport unavailable")
}

// PublishOperation sends an operation through the active consensus channel.
func (m *P2PConsensusManager) PublishOperation(ctx context.Context, op OperationEnvelope) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if !m.config.Enabled {
		return fmt.Errorf("consensus disabled")
	}
	if op.NetworkID == "" {
		op.NetworkID = m.config.NetworkID
	}
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
