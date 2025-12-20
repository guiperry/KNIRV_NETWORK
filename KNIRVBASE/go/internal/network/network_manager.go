package distributed

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/knirv/knirvbase/internal/types"
)

// MessageHandler receives a ProtocolMessage
type MessageHandler func(msg types.ProtocolMessage)

// Network defines the behaviour used by the distributed components. It enables tests to use a mock implementation.
type Network interface {
	Initialize() error
	CreateNetwork(cfg types.NetworkConfig) (string, error)
	JoinNetwork(networkID string, bootstrapPeers []string) error
	LeaveNetwork(networkID string) error

	AddCollectionToNetwork(networkID, collectionName string) error
	RemoveCollectionFromNetwork(networkID, collectionName string) error
	GetNetworkCollections(networkID string) []string

	BroadcastMessage(networkID string, msg types.ProtocolMessage) error
	SendToPeer(peerID string, networkID string, msg types.ProtocolMessage) error
	OnMessage(mt types.MessageType, handler MessageHandler)

	GetNetworkStats(networkID string) *types.NetworkStats
	GetNetworks() []*types.NetworkConfig
	GetPeerID() string
	Shutdown() error
}

// MockNetwork is a simple in-memory network for testing and single-node operation
type MockNetwork struct {
	peerID   string
	mu       sync.RWMutex
	networks map[string]*types.NetworkConfig
	stats    map[string]*types.NetworkStats
	handlers map[types.MessageType][]MessageHandler
}

func NewNetworkManager(ctx context.Context) *MockNetwork {
	return &MockNetwork{
		peerID:   "mock-peer-1",
		networks: make(map[string]*types.NetworkConfig),
		stats:    make(map[string]*types.NetworkStats),
		handlers: make(map[types.MessageType][]MessageHandler),
	}
}

func (n *MockNetwork) Initialize() error { return nil }

func (n *MockNetwork) CreateNetwork(cfg types.NetworkConfig) (string, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if _, exists := n.networks[cfg.NetworkID]; exists {
		return cfg.NetworkID, nil
	}

	cfg.Collections = make(map[string]bool)
	n.networks[cfg.NetworkID] = &cfg
	n.stats[cfg.NetworkID] = &types.NetworkStats{NetworkID: cfg.NetworkID}

	return cfg.NetworkID, nil
}

func (n *MockNetwork) JoinNetwork(networkID string, bootstrapPeers []string) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if _, exists := n.networks[networkID]; !exists {
		n.networks[networkID] = &types.NetworkConfig{NetworkID: networkID, Name: fmt.Sprintf("Network %s", networkID), Collections: make(map[string]bool)}
		n.stats[networkID] = &types.NetworkStats{NetworkID: networkID}
	}

	return nil
}

func (n *MockNetwork) LeaveNetwork(networkID string) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	delete(n.networks, networkID)
	delete(n.stats, networkID)

	return nil
}

func (n *MockNetwork) AddCollectionToNetwork(networkID, collectionName string) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	netCfg, ok := n.networks[networkID]
	if !ok {
		return errors.New("network not found")
	}

	netCfg.Collections[collectionName] = true
	if st, ok := n.stats[networkID]; ok {
		st.CollectionsShared = len(netCfg.Collections)
	}

	return nil
}

func (n *MockNetwork) RemoveCollectionFromNetwork(networkID, collectionName string) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	netCfg, ok := n.networks[networkID]
	if !ok {
		return nil
	}

	delete(netCfg.Collections, collectionName)
	if st, ok := n.stats[networkID]; ok {
		st.CollectionsShared = len(netCfg.Collections)
	}

	return nil
}

func (n *MockNetwork) GetNetworkCollections(networkID string) []string {
	n.mu.RLock()
	defer n.mu.RUnlock()

	netCfg, ok := n.networks[networkID]
	if !ok {
		return nil
	}

	out := make([]string, 0, len(netCfg.Collections))
	for c := range netCfg.Collections {
		out = append(out, c)
	}

	return out
}

func (n *MockNetwork) BroadcastMessage(networkID string, msg types.ProtocolMessage) error {
	// In mock, just handle locally
	n.handleMessage(msg)
	return nil
}

func (n *MockNetwork) SendToPeer(peerID string, networkID string, msg types.ProtocolMessage) error {
	// In mock, just handle locally
	n.handleMessage(msg)
	return nil
}

func (n *MockNetwork) OnMessage(mt types.MessageType, handler MessageHandler) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.handlers[mt] = append(n.handlers[mt], handler)
}

func (n *MockNetwork) GetNetworkStats(networkID string) *types.NetworkStats {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.stats[networkID]
}

func (n *MockNetwork) GetNetworks() []*types.NetworkConfig {
	n.mu.RLock()
	defer n.mu.RUnlock()

	out := make([]*types.NetworkConfig, 0, len(n.networks))
	for _, v := range n.networks {
		out = append(out, v)
	}

	return out
}

func (n *MockNetwork) GetPeerID() string { return n.peerID }

func (n *MockNetwork) Shutdown() error { return nil }

func (n *MockNetwork) handleMessage(msg types.ProtocolMessage) {
	n.mu.RLock()
	handlers := n.handlers[msg.Type]
	n.mu.RUnlock()

	for _, h := range handlers {
		go func(fn MessageHandler) {
			defer func() { _ = recover() }()
			fn(msg)
		}(h)
	}
}
