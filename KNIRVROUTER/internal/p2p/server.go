// p2p/server.go
package p2p

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"KNIRVROUTER/internal/constants"
	"KNIRVROUTER/internal/types"
)

// Message types for P2P communication
const (
	MessageTypeTx        = "TX"
	MessageTypeBlock     = "BLOCK"
	MessageTypePeerList  = "PEERS"
	MessageTypeGetBlocks = "GET_BLOCKS"
	MessageTypeGetPeers  = "GET_PEERS"
	MessageTypePing      = "PING"
	MessageTypePong      = "PONG"
)

// Message represents a P2P network message
type Message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// Server handles P2P communication
type Server struct {
	listenAddr     string
	bootstrapPeers []string
	peers          map[string]*Peer
	peersMutex     sync.RWMutex
	nodeID         string
	chainID        string

	// Callbacks to blockchain logic
	handleNewTx    func(*types.Transaction)
	handleNewBlock func(types.Block)

	// Channels for communication
	quit          chan struct{}
	peerDiscovery *time.Ticker

	// DHT for decentralized peer discovery
	dht    *DHTManager
	useDHT bool
}

// NewServer creates a new P2P server
func NewServer(listenAddr string, nodeID string, chainID string, bootstrapPeers []string,
	handleNewTx func(*types.Transaction),
	handleNewBlock func(types.Block)) *Server {
	return &Server{
		listenAddr:     listenAddr,
		bootstrapPeers: bootstrapPeers,
		peers:          make(map[string]*Peer),
		nodeID:         nodeID,
		chainID:        chainID,
		handleNewTx:    handleNewTx,
		handleNewBlock: handleNewBlock,
		quit:           make(chan struct{}),
		useDHT:         true, // Enable DHT by default
	}
}

// GetChainID returns the chain ID of the server
func (s *Server) GetChainID() string {
	return s.chainID
}

// SetChainID sets the chain ID of the server
func (s *Server) SetChainID(chainID string) {
	s.chainID = chainID
}

// NewServerWithDHT creates a new P2P server with DHT enabled or disabled
func NewServerWithDHT(listenAddr string, nodeID string, chainID string, bootstrapPeers []string,
	handleNewTx func(*types.Transaction),
	handleNewBlock func(types.Block), useDHT bool) *Server {
	return &Server{
		listenAddr:     listenAddr,
		bootstrapPeers: bootstrapPeers,
		peers:          make(map[string]*Peer),
		nodeID:         nodeID,
		chainID:        chainID,
		handleNewTx:    handleNewTx,
		handleNewBlock: handleNewBlock,
		quit:           make(chan struct{}),
		useDHT:         useDHT,
	}
}

// Start begins listening for peers and connects to bootstrap peers
func (s *Server) Start() error {
	// Initialize DHT if enabled
	if s.useDHT {
		dhtManager, err := NewDHTManager(s, s.bootstrapPeers)
		if err != nil {
			return fmt.Errorf("failed to initialize DHT: %v", err)
		}
		s.dht = dhtManager

		// Start the DHT
		if err := s.dht.Start(); err != nil {
			return fmt.Errorf("failed to start DHT: %v", err)
		}

		log.Println("DHT initialized and started successfully")
	}

	// Start listening for incoming connections (traditional TCP)
	listener, err := net.Listen("tcp", s.listenAddr)
	if err != nil {
		return fmt.Errorf("failed to start P2P server: %v", err)
	}

	log.Printf("P2P server listening on %s", s.listenAddr)

	// Connect to bootstrap peers (traditional TCP)
	for _, addr := range s.bootstrapPeers {
		go s.connectToPeer(addr)
	}

	// Start peer discovery ticker
	s.peerDiscovery = time.NewTicker(constants.PEER_DISCOVERY_INTERVAL * time.Second)
	go func() {
		for {
			select {
			case <-s.peerDiscovery.C:
				s.discoverPeers()
			case <-s.quit:
				return
			}
		}
	}()

	// Accept incoming connections in a goroutine
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-s.quit:
					return // Server is shutting down
				default:
					log.Printf("Error accepting connection: %v", err)
					continue
				}
			}

			go s.handleConnection(conn)
		}
	}()

	return nil
}

// Stop shuts down the P2P server
func (s *Server) Stop() {
	close(s.quit)
	s.peerDiscovery.Stop()

	// Close all peer connections
	s.peersMutex.Lock()
	for _, peer := range s.peers {
		peer.disconnect()
	}
	s.peersMutex.Unlock()

	// Stop DHT if it's running
	if s.dht != nil {
		s.dht.Stop()
		log.Println("DHT stopped")
	}

	log.Println("P2P server stopped")
}

// connectToPeer attempts to establish a connection to a peer
func (s *Server) connectToPeer(addr string) {
	// Don't connect to self
	if addr == s.listenAddr {
		return
	}

	// Check if already connected
	s.peersMutex.RLock()
	_, exists := s.peers[addr]
	s.peersMutex.RUnlock()

	if exists {
		return
	}

	log.Printf("Connecting to peer: %s", addr)
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		log.Printf("Failed to connect to peer %s: %v", addr, err)
		return
	}

	// Send handshake message
	handshake := Message{
		Type:    "HANDSHAKE",
		Payload: []byte(fmt.Sprintf(`{"nodeID":"%s","listenAddr":"%s"}`, s.nodeID, s.listenAddr)),
	}

	handshakeData, err := json.Marshal(handshake)
	if err != nil {
		log.Printf("Failed to marshal handshake: %v", err)
		conn.Close()
		return
	}

	_, err = conn.Write(append(handshakeData, '\n'))
	if err != nil {
		log.Printf("Failed to send handshake: %v", err)
		conn.Close()
		return
	}

	// Handle the connection
	s.handleConnection(conn)
}

// handleConnection processes a new connection
func (s *Server) handleConnection(conn net.Conn) {
	// Read handshake
	decoder := json.NewDecoder(conn)
	var handshake Message
	if err := decoder.Decode(&handshake); err != nil {
		log.Printf("Failed to decode handshake: %v", err)
		conn.Close()
		return
	}

	if handshake.Type != "HANDSHAKE" {
		log.Printf("Expected handshake, got %s", handshake.Type)
		conn.Close()
		return
	}

	// Parse handshake data
	var handshakeData struct {
		NodeID     string `json:"nodeID"`
		ListenAddr string `json:"listenAddr"`
	}

	if err := json.Unmarshal(handshake.Payload, &handshakeData); err != nil {
		log.Printf("Failed to parse handshake data: %v", err)
		conn.Close()
		return
	}

	// Create peer
	peer := NewPeer(handshakeData.NodeID, handshakeData.ListenAddr, conn, s)

	// Add peer to list
	s.peersMutex.Lock()
	s.peers[handshakeData.ListenAddr] = peer
	s.peersMutex.Unlock()

	log.Printf("New peer connected: %s at %s", handshakeData.NodeID, handshakeData.ListenAddr)

	// Start peer message handling
	peer.start()

	// Request peer list
	peer.sendMessage(Message{
		Type:    MessageTypeGetPeers,
		Payload: []byte("{}"),
	})
}

// discoverPeers asks all connected peers for their peer lists
func (s *Server) discoverPeers() {
	s.peersMutex.RLock()
	defer s.peersMutex.RUnlock()

	for _, peer := range s.peers {
		peer.sendMessage(Message{
			Type:    MessageTypeGetPeers,
			Payload: []byte("{}"),
		})
	}
}

// BroadcastTransaction sends a transaction to all connected peers
func (s *Server) BroadcastTransaction(tx *types.Transaction) {
	// Only broadcast private transactions
	if tx.Origin != "nrn" {
		return
	}

	txData, err := json.Marshal(tx)
	if err != nil {
		log.Printf("Failed to marshal transaction: %v", err)
		return
	}

	msg := Message{
		Type:    MessageTypeTx,
		Payload: txData,
	}

	s.broadcastMessage(msg)
}

// BroadcastBlock sends a block to all connected peers
func (s *Server) BroadcastBlock(block types.Block) {
	// Marshal the block interface directly
	blockData, err := json.Marshal(block)
	if err != nil {
		log.Printf("Failed to marshal block: %v", err)
		return
	}

	msg := Message{
		Type:    MessageTypeBlock,
		Payload: blockData,
	}

	s.broadcastMessage(msg)
}

// broadcastMessage sends a message to all connected peers
func (s *Server) broadcastMessage(msg Message) {
	s.peersMutex.RLock()
	defer s.peersMutex.RUnlock()

	for _, peer := range s.peers {
		peer.sendMessage(msg)
	}
}

// handlePeerList processes a peer list received from a peer
func (s *Server) handlePeerList(peerList []string) {
	for _, addr := range peerList {
		// Don't connect to self
		if addr == s.listenAddr {
			continue
		}

		// Check if already connected
		s.peersMutex.RLock()
		_, exists := s.peers[addr]
		s.peersMutex.RUnlock()

		if !exists {
			go s.connectToPeer(addr)
		}
	}
}

// GetPeerList returns a list of connected peer addresses
func (s *Server) GetPeerList() []string {
	s.peersMutex.RLock()
	defer s.peersMutex.RUnlock()

	peerList := make([]string, 0, len(s.peers))
	for addr := range s.peers {
		peerList = append(peerList, addr)
	}

	return peerList
}

// ResolveKnirvURI resolves a knirv:// URI to peer addresses
// This is the main entry point for handling the new URI scheme
func (s *Server) ResolveKnirvURI(uri string) ([]string, error) {
	// Check if DHT is enabled
	if !s.useDHT || s.dht == nil {
		return nil, fmt.Errorf("DHT is not enabled, cannot resolve knirv:// URIs")
	}

	// Use the DHT to resolve the URI
	peerInfos, err := s.dht.ResolveKnirvURI(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve URI: %w", err)
	}

	// Convert peer.AddrInfo to string addresses
	addresses := make([]string, 0, len(peerInfos))
	for _, pi := range peerInfos {
		for _, addr := range pi.Addrs {
			addresses = append(addresses, fmt.Sprintf("%s/p2p/%s", addr.String(), pi.ID.String()))
		}
	}

	return addresses, nil
}

// handleIncomingMessage processes messages from peers
func (s *Server) handleIncomingMessage(peer *Peer, msg Message) {
	switch msg.Type {
	case MessageTypeTx:
		var tx types.Transaction
		if err := json.Unmarshal(msg.Payload, &tx); err != nil {
			log.Printf("Failed to unmarshal transaction: %v", err)
			return
		}

		// Only process private transactions
		if tx.Origin == "nrn" {
			s.handleNewTx(&tx)
		}

	case MessageTypeBlock:
		var block types.Block
		if err := json.Unmarshal(msg.Payload, &block); err != nil {
			log.Printf("Failed to unmarshal block: %v", err)
			return
		}
		s.handleNewBlock(block)

	case MessageTypeGetPeers:
		// Send our peer list
		peerList := s.GetPeerList()
		peerListData, err := json.Marshal(peerList)
		if err != nil {
			log.Printf("Failed to marshal peer list: %v", err)
			return
		}

		peer.sendMessage(Message{
			Type:    MessageTypePeerList,
			Payload: peerListData,
		})

	case MessageTypePeerList:
		var peerList []string
		if err := json.Unmarshal(msg.Payload, &peerList); err != nil {
			log.Printf("Failed to unmarshal peer list: %v", err)
			return
		}

		s.handlePeerList(peerList)

	case MessageTypePing:
		// Respond with pong
		peer.sendMessage(Message{
			Type:    MessageTypePong,
			Payload: []byte("{}"),
		})
	}
}

// removePeer removes a peer from the peer list
func (s *Server) removePeer(peer *Peer) {
	s.peersMutex.Lock()
	defer s.peersMutex.Unlock()

	delete(s.peers, peer.addr)
	log.Printf("Peer disconnected: %s", peer.addr)
}
