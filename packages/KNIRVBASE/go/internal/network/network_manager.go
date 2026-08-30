// Package network provides the legacy P2P networking layer for KNIRVBASE.
//
// Deprecated: The custom TCP-based P2P implementation in this package is superseded by
// the p2pconsensus package (internal/p2pconsensus), which supports gateway-proxy and
// standalone consensus modes. New code should use p2pconsensus.P2PConsensusManager
// directly. The Network interface and NetworkManager are retained for backward
// compatibility with existing DistributedCollection tests.
package network

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/knirvcorp/knirvbase/internal/types"
)

// MessageHandler receives a ProtocolMessage
type MessageHandler func(msg types.ProtocolMessage)

// Network defines the behaviour used by the distributed components. It enables tests to use a mock implementation.
//
// Deprecated: Use p2pconsensus.P2PConsensusManager for new development.
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

// DHTNode represents a node in our simplified DHT
type DHTNode struct {
	ID       string
	Address  string
	LastSeen time.Time
}

// DHTValue represents a value stored in the DHT
type DHTValue struct {
	Key       string      `json:"key"`
	Value     interface{} `json:"value"`
	TTL       int64       `json:"ttl"`
	Timestamp int64       `json:"timestamp"`
}

// NetworkManager is a custom P2P implementation with DHT-like functionality.
//
// Deprecated: Use p2pconsensus.P2PConsensusManager for new development.
type NetworkManager struct {
	ctx      context.Context
	cancel   context.CancelFunc
	listener *net.TCPListener
	peerID   string

	mu          sync.RWMutex
	networks    map[string]*types.NetworkConfig
	peers       map[string]*types.PeerInfo
	dht         map[string][]DHTNode  // Simple DHT: key -> list of nodes holding the value
	kvStore     map[string][]DHTValue // Simple DHT: key -> values
	connections map[string]net.Conn   // peerID -> connection
	stats       map[string]*types.NetworkStats
	handlers    map[types.MessageType][]MessageHandler
	initialized bool
}

// NewNetworkManager creates a new custom P2P network manager
func NewNetworkManager(ctx context.Context) *NetworkManager {
	// Generate a unique peer ID
	h := sha256.Sum256([]byte(fmt.Sprintf("%d-%d", time.Now().UnixNano(), rand.Int63())))
	peerID := hex.EncodeToString(h[:16])

	c, cancel := context.WithCancel(ctx)
	return &NetworkManager{
		ctx:         c,
		cancel:      cancel,
		peerID:      peerID,
		networks:    make(map[string]*types.NetworkConfig),
		peers:       make(map[string]*types.PeerInfo),
		dht:         make(map[string][]DHTNode),
		kvStore:     make(map[string][]DHTValue),
		connections: make(map[string]net.Conn),
		stats:       make(map[string]*types.NetworkStats),
		handlers:    make(map[types.MessageType][]MessageHandler),
	}
}

// DHT methods for global indexing

func (n *NetworkManager) PutDHT(key string, value interface{}, ttl time.Duration) error {
	n.mu.Lock()
	entry := DHTValue{
		Key:       key,
		Value:     value,
		TTL:       int64(ttl.Seconds()),
		Timestamp: time.Now().Unix(),
	}
	n.kvStore[key] = append(n.kvStore[key], entry)
	n.mu.Unlock()

	// In a real DHT, we would find the closest nodes to the key and store it there.
	// For this implementation, we broadcast a DHT announcement.
	return n.BroadcastMessage("", types.ProtocolMessage{
		Type:      types.MsgDHTPut,
		SenderID:  n.peerID,
		Timestamp: time.Now().UnixMilli(),
		Payload: map[string]interface{}{
			"key":   key,
			"value": value,
			"ttl":   ttl.Seconds(),
		},
	})
}

func (n *NetworkManager) GetDHT(key string) ([]interface{}, error) {
	n.mu.RLock()
	entries, ok := n.kvStore[key]
	n.mu.RUnlock()

	var results []interface{}
	now := time.Now().Unix()

	if ok {
		for _, entry := range entries {
			if entry.TTL == 0 || entry.Timestamp+entry.TTL > now {
				results = append(results, entry.Value)
			}
		}
	}

	// In a real DHT, if not found locally, we would query other nodes.
	// For now we just return what we have.
	return results, nil
}

func (n *NetworkManager) Initialize() error {
	n.mu.Lock()
	if n.initialized {
		n.mu.Unlock()
		return nil
	}

	n.initialized = true

	// Register DHT handlers - unlock first to avoid deadlock
	n.mu.Unlock()

	n.OnMessage(types.MsgDHTPut, func(msg types.ProtocolMessage) {
		payload, ok := msg.Payload.(map[string]interface{})
		if !ok {
			return
		}

		key, _ := payload["key"].(string)
		value := payload["value"]
		ttlFloat, _ := payload["ttl"].(float64)

		n.mu.Lock()
		n.kvStore[key] = append(n.kvStore[key], DHTValue{
			Key:       key,
			Value:     value,
			TTL:       int64(ttlFloat),
			Timestamp: time.Now().Unix(),
		})
		n.mu.Unlock()
	})

	// Start the transport layer (TCP when built with the legacy_tcp tag,
	// otherwise a no-op since the gateway/consensus path is preferred).
	go n.startTransport()

	log.Printf("Custom P2P node initialized: %s", n.peerID)
	return nil
}

func (n *NetworkManager) CreateNetwork(cfg types.NetworkConfig) (string, error) {
	if err := n.Initialize(); err != nil {
		return "", err
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	if _, exists := n.networks[cfg.NetworkID]; exists {
		return cfg.NetworkID, nil
	}

	cfg.Collections = make(map[string]bool)
	n.networks[cfg.NetworkID] = &cfg
	n.stats[cfg.NetworkID] = &types.NetworkStats{NetworkID: cfg.NetworkID}

	log.Printf("Created network %s", cfg.NetworkID)
	return cfg.NetworkID, nil
}

func (n *NetworkManager) JoinNetwork(networkID string, bootstrapPeers []string) error {
	if err := n.Initialize(); err != nil {
		return err
	}

	n.mu.Lock()
	if _, exists := n.networks[networkID]; !exists {
		n.networks[networkID] = &types.NetworkConfig{
			NetworkID:   networkID,
			Name:        fmt.Sprintf("Network %s", networkID),
			Collections: make(map[string]bool),
		}
		n.stats[networkID] = &types.NetworkStats{NetworkID: networkID}
	}
	n.mu.Unlock()

	// Connect to bootstrap peers (only meaningful when a TCP transport is built)
	for _, addr := range bootstrapPeers {
		go n.connectToPeer(addr)
	}

	return nil
}

func (n *NetworkManager) LeaveNetwork(networkID string) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if _, ok := n.networks[networkID]; !ok {
		return nil
	}

	delete(n.networks, networkID)
	delete(n.stats, networkID)

	log.Printf("Left network %s", networkID)
	return nil
}

func (n *NetworkManager) AddCollectionToNetwork(networkID, collectionName string) error {
	n.mu.Lock()
	netCfg, ok := n.networks[networkID]
	n.mu.Unlock()
	if !ok {
		return errors.New("network not found")
	}

	n.mu.Lock()
	netCfg.Collections[collectionName] = true
	if st, ok := n.stats[networkID]; ok {
		st.CollectionsShared = len(netCfg.Collections)
	}
	n.mu.Unlock()

	// Announce collection to all connected peers
	return n.BroadcastMessage(networkID, types.ProtocolMessage{
		Type:      types.MsgCollectionAnnounce,
		NetworkID: networkID,
		SenderID:  n.GetPeerID(),
		Timestamp: time.Now().UnixMilli(),
		Payload:   map[string]string{"collection": collectionName},
	})
}

func (n *NetworkManager) RemoveCollectionFromNetwork(networkID, collectionName string) error {
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

func (n *NetworkManager) GetNetworkCollections(networkID string) []string {
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

func (n *NetworkManager) BroadcastMessage(networkID string, msg types.ProtocolMessage) error {
	if !n.initialized {
		return errors.New("not initialized")
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	n.mu.RLock()
	conns := make([]net.Conn, 0, len(n.connections))
	for _, conn := range n.connections {
		conns = append(conns, conn)
	}
	st := n.stats[networkID]
	n.mu.RUnlock()

	for _, conn := range conns {
		go func(c net.Conn) {
			_, err := fmt.Fprintf(c, "%s\n", data)
			if err != nil {
				log.Printf("Failed to send message: %v", err)
				return
			}

			if st != nil {
				n.mu.Lock()
				st.OperationsSent++
				st.BytesTransferred += int64(len(data))
				n.mu.Unlock()
			}
		}(conn)
	}

	return nil
}

func (n *NetworkManager) SendToPeer(peerID string, networkID string, msg types.ProtocolMessage) error {
	if !n.initialized {
		return errors.New("not initialized")
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	n.mu.RLock()
	conn, ok := n.connections[peerID]
	st := n.stats[networkID]
	n.mu.RUnlock()

	if !ok {
		return errors.New("peer not connected")
	}

	_, err = fmt.Fprintf(conn, "%s\n", data)
	if err != nil {
		return err
	}

	if st != nil {
		n.mu.Lock()
		st.OperationsSent++
		st.BytesTransferred += int64(len(data))
		n.mu.Unlock()
	}

	return nil
}

func (n *NetworkManager) OnMessage(mt types.MessageType, handler MessageHandler) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.handlers[mt] = append(n.handlers[mt], handler)
}

func (n *NetworkManager) GetNetworkStats(networkID string) *types.NetworkStats {
	n.mu.Lock()
	defer n.mu.Unlock()
	st := n.stats[networkID]
	if st == nil {
		return nil
	}
	// Do not return the mutable internal counter object. HTTP monitoring reads
	// concurrently with replication and must never race a sender updating it.
	snapshot := *st
	snapshot.ConnectedPeers = len(n.connections)
	snapshot.TotalPeers = len(n.peers)
	return &snapshot
}

func (n *NetworkManager) GetNetworks() []*types.NetworkConfig {
	n.mu.RLock()
	defer n.mu.RUnlock()
	out := make([]*types.NetworkConfig, 0, len(n.networks))
	for _, v := range n.networks {
		out = append(out, v)
	}
	return out
}

func (n *NetworkManager) GetPeerID() string { return n.peerID }

func (n *NetworkManager) Shutdown() error {
	n.cancel()

	n.mu.Lock()
	defer n.mu.Unlock()

	if n.listener != nil {
		n.listener.Close()
	}

	for _, conn := range n.connections {
		conn.Close()
	}

	n.connections = make(map[string]net.Conn)
	n.initialized = false

	return nil
}

func (n *NetworkManager) handleMessage(msg types.ProtocolMessage) {
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
