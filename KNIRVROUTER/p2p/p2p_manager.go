// p2p/p2p_manager.go
package p2p

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"KNIRVROUTER_GO_Verifyer/blockchain"
	"KNIRVROUTER_GO_Verifyer/types"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
)

// Network pause constants
const (
	NetworkControlTopic = "network-control"
	NetworkPauseTimeout = 30 * time.Minute
)

// NetworkControlMessage represents network control messages
type NetworkControlMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// NetworkPausePayload represents network pause message payload
type NetworkPausePayload struct {
	InitiatorPeerID string `json:"initiator_peer_id"`
	Reason          string `json:"reason"`
	Timestamp       int64  `json:"timestamp"`
}

// P2PManager is the main entry point for all P2P functionality
// It consolidates the functionality from DHTManager, DiscoveryManager, and P2PConsensusManager
type P2PManager struct {
	// Core components
	host           host.Host
	pubsub         *pubsub.PubSub
	kadDHT         *dht.IpfsDHT
	blockchain     *blockchain.BlockchainStruct
	db             *blockchain.LevelDB
	ctx            context.Context
	cancel         context.CancelFunc
	bootstrapPeers []peer.AddrInfo

	// PubSub topics and subscriptions
	blockTopic       *pubsub.Topic
	blockSub         *pubsub.Subscription
	transactionTopic *pubsub.Topic
	transactionSub   *pubsub.Subscription

	// Consensus state
	miningLocked bool
	isSyncing    bool
	mu           sync.Mutex
	stopChan     chan struct{}

	// Chain ID
	chainID string

	// Network pause state
	networkControlTopic *pubsub.Topic
	networkControlSub   *pubsub.Subscription
	networkPaused       bool
	pausedUntil         time.Time
	pauseMutex          sync.RWMutex

	// Callbacks for processing received blocks and transactions
	// These will be set by the blockchain connector
	processReceivedBlock       func(*blockchain.Block)
	processReceivedTransaction func(*types.Transaction)
}

// NewP2PManager creates a new P2P manager
func NewP2PManager(blockchain *blockchain.BlockchainStruct, db *blockchain.LevelDB, bootstrapAddrs []string) (*P2PManager, error) {
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
		// Disable auto relay for testnet mode to avoid relay finder issues
		// libp2p.EnableAutoRelay(),
		libp2p.EnableHolePunching(),
	)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create libp2p host: %w", err)
	}

	// Log the host's addresses
	log.Println("P2P node started with addresses:")
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

	// Create a new pubsub instance
	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		h.Close()
		kadDHT.Close()
		cancel()
		return nil, fmt.Errorf("failed to create pubsub: %w", err)
	}

	manager := &P2PManager{
		host:           h,
		pubsub:         ps,
		kadDHT:         kadDHT,
		blockchain:     blockchain,
		db:             db,
		ctx:            ctx,
		cancel:         cancel,
		bootstrapPeers: bootstrapPeers,
		chainID:        blockchain.GetChainID(),
		stopChan:       make(chan struct{}),
		isSyncing:      false,
	}

	return manager, nil
}

// Start initializes the P2P manager and starts all services
func (pm *P2PManager) Start() error {
	// Bootstrap the DHT
	if err := pm.kadDHT.Bootstrap(pm.ctx); err != nil {
		return fmt.Errorf("failed to bootstrap DHT: %w", err)
	}

	// Connect to bootstrap peers
	var wg sync.WaitGroup
	for _, peerInfo := range pm.bootstrapPeers {
		wg.Add(1)
		go func(peerInfo peer.AddrInfo) {
			defer wg.Done()
			if err := pm.host.Connect(pm.ctx, peerInfo); err != nil {
				log.Printf("Failed to connect to bootstrap peer %s: %v", peerInfo.ID, err)
			} else {
				log.Printf("Connected to bootstrap peer: %s", peerInfo.ID)
			}
		}(peerInfo)
	}
	wg.Wait()

	// Setup local mDNS discovery
	notifee := &p2pDiscoveryNotifee{pm: pm}
	discovery := mdns.NewMdnsService(pm.host, "knirvchain", notifee)
	if err := discovery.Start(); err != nil {
		return fmt.Errorf("failed to start mDNS discovery: %w", err)
	}

	// Initialize pubsub topics and subscriptions
	if err := pm.setupPubSub(); err != nil {
		return err
	}

	// Register chain sync handler
	pm.registerSyncHandler()

	// Announce our chain to the DHT
	if err := pm.AnnounceChain(); err != nil {
		log.Printf("Warning: Failed to announce chain: %v", err)
	}

	// Start periodic DHT record refresh
	go pm.refreshLoop()

	// Start the block handler
	go pm.handleBlocks()

	// Start the transaction handler
	go pm.handleTransactions()

	// Start the fork resolution process
	go pm.runForkResolution()

	// Start the network control handler
	go pm.handleNetworkControl()

	log.Println("P2P manager started successfully")
	return nil
}

// Stop shuts down the P2P manager
func (pm *P2PManager) Stop() {
	log.Println("Stopping P2P manager...")

	// Signal fork resolution to stop
	close(pm.stopChan)

	// Cancel context to stop all goroutines
	pm.cancel()

	// Close the host
	if err := pm.host.Close(); err != nil {
		log.Printf("Error closing libp2p host: %v", err)
	}

	log.Println("P2P manager stopped")
}

// setupPubSub initializes the pubsub topics and subscriptions
func (pm *P2PManager) setupPubSub() error {
	var err error

	// Join the block topic
	pm.blockTopic, err = pm.pubsub.Join(fmt.Sprintf("%s.blocks", pm.chainID))
	if err != nil {
		return fmt.Errorf("failed to join block topic: %w", err)
	}

	// Subscribe to the block topic
	pm.blockSub, err = pm.blockTopic.Subscribe()
	if err != nil {
		return fmt.Errorf("failed to subscribe to block topic: %w", err)
	}

	// Join the transaction topic
	pm.transactionTopic, err = pm.pubsub.Join(fmt.Sprintf("%s.transactions", pm.chainID))
	if err != nil {
		return fmt.Errorf("failed to join transaction topic: %w", err)
	}

	// Subscribe to the transaction topic
	pm.transactionSub, err = pm.transactionTopic.Subscribe()
	if err != nil {
		return fmt.Errorf("failed to subscribe to transaction topic: %w", err)
	}

	// Join the network control topic
	pm.networkControlTopic, err = pm.pubsub.Join(NetworkControlTopic)
	if err != nil {
		return fmt.Errorf("failed to join network control topic: %w", err)
	}

	// Subscribe to the network control topic
	pm.networkControlSub, err = pm.networkControlTopic.Subscribe()
	if err != nil {
		return fmt.Errorf("failed to subscribe to network control topic: %w", err)
	}

	log.Printf("P2P manager subscribed to topics: %s.blocks, %s.transactions, %s", pm.chainID, pm.chainID, NetworkControlTopic)
	return nil
}

// refreshLoop periodically refreshes DHT records
func (pm *P2PManager) refreshLoop() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-pm.ctx.Done():
			return
		case <-ticker.C:
			// Refresh DHT records
			if err := pm.AnnounceChain(); err != nil {
				log.Printf("Failed to refresh DHT records: %v", err)
			}
		}
	}
}

// AnnounceChain announces this node's chain to the DHT
func (pm *P2PManager) AnnounceChain() error {
	// Create a CID from the chain ID
	cid, err := createCIDFromChainID(pm.chainID)
	if err != nil {
		return fmt.Errorf("failed to create CID: %w", err)
	}

	// Announce that this node provides the chain
	if err := pm.kadDHT.Provide(pm.ctx, cid, true); err != nil {
		return fmt.Errorf("failed to announce chain: %w", err)
	}

	log.Printf("Announced chain %s to DHT", pm.chainID)
	return nil
}

// FindChainProviders finds nodes that provide a specific chain
func (pm *P2PManager) FindChainProviders(chainID string) ([]peer.AddrInfo, error) {
	// Create a CID from the chain ID
	cid, err := createCIDFromChainID(chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to create CID: %w", err)
	}

	// Find providers for the chain
	ctx, cancel := context.WithTimeout(pm.ctx, 30*time.Second)
	defer cancel()

	providers := pm.kadDHT.FindProvidersAsync(ctx, cid, 20)

	var results []peer.AddrInfo
	for p := range providers {
		results = append(results, p)
	}

	return results, nil
}

// ResolveKnirvURI resolves a knirv:// URI to peer addresses
func (pm *P2PManager) ResolveKnirvURI(uri string) ([]peer.AddrInfo, error) {
	// Parse the URI
	id, resourceType, _, err := ParseKnirvURI(uri)
	if err != nil {
		return nil, err
	}

	// Handle different resource types
	switch resourceType {
	case ResourceTypeChain:
		// Find providers for the chain
		return pm.FindChainProviders(id)
	case ResourceTypeNRN:
		// Find providers for the NRN asset
		// For now, we'll use the same method as for chains
		return pm.FindChainProviders(id)
	default:
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}
}

// BroadcastBlock publishes a block to the network using pubsub
func (pm *P2PManager) BroadcastBlock(block *blockchain.Block) error {
	// Marshal the block to JSON
	blockData, err := block.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal block: %w", err)
	}

	// Publish the block to the network
	err = pm.blockTopic.Publish(pm.ctx, []byte(blockData))
	if err != nil {
		return fmt.Errorf("failed to publish block: %w", err)
	}

	log.Printf("Block #%d broadcast to the network", block.BlockNumber())
	return nil
}

// BroadcastTransaction publishes a transaction to the network using pubsub
func (pm *P2PManager) BroadcastTransaction(transaction *types.Transaction) error {
	// Marshal the transaction to JSON
	txData := transaction.ToJson()

	// Publish the transaction to the network
	if err := pm.transactionTopic.Publish(pm.ctx, []byte(txData)); err != nil {
		return fmt.Errorf("failed to publish transaction: %w", err)
	}

	log.Printf("Transaction %s broadcast to the network", transaction.Hash())
	return nil
}

// p2pDiscoveryNotifee gets notified when we find a new peer via mDNS discovery
type p2pDiscoveryNotifee struct {
	pm *P2PManager
}

// HandlePeerFound connects to peers discovered via mDNS discovery
func (n *p2pDiscoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	log.Printf("Discovered new peer %s via mDNS", pi.ID.String())
	err := n.pm.host.Connect(n.pm.ctx, pi)
	if err != nil {
		log.Printf("Error connecting to mDNS peer %s: %v", pi.ID.String(), err)
	}
}

// The following methods are imported from p2p_consensus.go

// handleBlocks processes incoming blocks from the network
func (pm *P2PManager) handleBlocks() {
	for {
		msg, err := pm.blockSub.Next(pm.ctx)
		if err != nil {
			if pm.ctx.Err() != nil {
				// Context was canceled, exit gracefully
				return
			}
			log.Printf("Error receiving block from pubsub: %v", err)
			continue
		}

		// Skip messages from ourselves
		if msg.ReceivedFrom == pm.host.ID() {
			continue
		}

		// Decode the block
		block, err := blockchain.BlockFromJSON(string(msg.Data))
		if err != nil {
			log.Printf("Error decoding block: %v", err)
			continue
		}

		log.Printf("Received block #%d from peer %s", block.BlockNumber(), msg.ReceivedFrom.String())

		// Process the block using the internal method
		pm.processReceivedBlockInternal(block)
	}
}

// handleTransactions processes incoming transactions from the network
func (pm *P2PManager) handleTransactions() {
	for {
		msg, err := pm.transactionSub.Next(pm.ctx)
		if err != nil {
			if pm.ctx.Err() != nil {
				// Context was canceled, exit gracefully
				return
			}
			log.Printf("Error receiving transaction from pubsub: %v", err)
			continue
		}

		// Skip messages from ourselves
		if msg.ReceivedFrom == pm.host.ID() {
			continue
		}

		// Decode the transaction
		var transaction types.Transaction
		if err := json.Unmarshal(msg.Data, &transaction); err != nil {
			log.Printf("Error decoding transaction: %v", err)
			continue
		}

		log.Printf("Received transaction %s from peer %s", transaction.Hash(), msg.ReceivedFrom.String())

		// Process the transaction using the internal method
		pm.processReceivedTransactionInternal(&transaction)
	}
}

// processReceivedBlockInternal validates and adds a block received from the network
// This is the internal implementation that calls the callback if set
func (pm *P2PManager) processReceivedBlockInternal(block *blockchain.Block) {
	if pm.processReceivedBlock != nil {
		// Use the callback set by the blockchain connector
		pm.processReceivedBlock(block)
	} else {
		log.Println("Warning: processReceivedBlock callback not set, block not processed")
	}
}

// processReceivedTransactionInternal validates and adds a transaction received from the network
// This is the internal implementation that calls the callback if set
func (pm *P2PManager) processReceivedTransactionInternal(transaction *types.Transaction) {
	if pm.processReceivedTransaction != nil {
		// Use the callback set by the blockchain connector
		pm.processReceivedTransaction(transaction)
	} else {
		log.Println("Warning: processReceivedTransaction callback not set, transaction not processed")
	}
}

// lockMining locks mining during block processing
func (pm *P2PManager) lockMining() {
	pm.mu.Lock()
	pm.miningLocked = true
	pm.mu.Unlock()
}

// unlockMining unlocks mining after block processing
func (pm *P2PManager) unlockMining() {
	pm.mu.Lock()
	pm.miningLocked = false
	pm.mu.Unlock()
}

// The following methods are imported from p2p_consensus.go for chain sync

// registerSyncHandler registers the chain sync stream handler
func (pm *P2PManager) registerSyncHandler() {
	pm.host.SetStreamHandler(ChainSyncProtocolID, pm.handleSyncStream)
	log.Printf("[%s] Registered chain sync handler for protocol %s", pm.chainID, ChainSyncProtocolID)
}

// runForkResolution periodically checks for and resolves blockchain forks
func (pm *P2PManager) runForkResolution() {
	// Use a configurable sync interval with a reasonable default
	syncInterval := 60 * time.Second

	ticker := time.NewTicker(syncInterval)
	defer ticker.Stop()

	log.Printf("[%s] Starting fork resolution with sync interval: %v", pm.chainID, syncInterval)

	for {
		select {
		case <-ticker.C:
			// Skip if we're actively mining
			if pm.blockchain.IsActivelyMining() {
				log.Printf("[%s] Skipping chain sync while actively mining", pm.chainID)
				continue
			}

			log.Printf("[%s] Performing periodic chain sync check", pm.chainID)
			// Request chain data from peers
			pm.requestChainFromPeers()

		case <-pm.stopChan:
			log.Printf("[%s] Fork resolution stopped", pm.chainID)
			return
		}
	}
}

// requestChainFromPeers requests chain data from peers
// This is a placeholder that will be implemented in the full version
func (pm *P2PManager) requestChainFromPeers() {
	// Implementation will be added in the full version
	log.Printf("[%s] Chain sync from peers requested (not fully implemented)", pm.chainID)
}

// handleSyncStream handles incoming chain sync requests
// This is a placeholder that will be implemented in the full version
func (pm *P2PManager) handleSyncStream(stream network.Stream) {
	// Implementation will be added in the full version
	log.Printf("[%s] Received chain sync stream from %s", pm.chainID, stream.Conn().RemotePeer())
	defer stream.Close()
}

// GetHost returns the libp2p host
func (pm *P2PManager) GetHost() host.Host {
	return pm.host
}

// GetDHT returns the Kademlia DHT
func (pm *P2PManager) GetDHT() *dht.IpfsDHT {
	return pm.kadDHT
}

// GetPubSub returns the pubsub instance
func (pm *P2PManager) GetPubSub() *pubsub.PubSub {
	return pm.pubsub
}

// IsNetworkPaused returns whether the network is currently paused
func (pm *P2PManager) IsNetworkPaused() bool {
	pm.pauseMutex.RLock()
	defer pm.pauseMutex.RUnlock()

	if pm.networkPaused && time.Now().After(pm.pausedUntil) {
		// Pause has expired
		pm.pauseMutex.RUnlock()
		pm.pauseMutex.Lock()
		pm.networkPaused = false
		pm.pauseMutex.Unlock()
		pm.pauseMutex.RLock()
		log.Printf("Network pause expired for KNIRVROUTER")
	}

	return pm.networkPaused
}

// handleNetworkControl processes network control messages (pause/resume)
func (pm *P2PManager) handleNetworkControl() {
	if pm.networkControlSub == nil {
		return
	}

	for {
		select {
		case <-pm.ctx.Done():
			return
		default:
			msg, err := pm.networkControlSub.Next(pm.ctx)
			if err != nil {
				if pm.ctx.Err() != nil {
					return // Context cancelled
				}
				log.Printf("Error receiving network control message: %v", err)
				continue
			}

			// Parse network control message
			var networkMsg NetworkControlMessage
			if err := json.Unmarshal(msg.Data, &networkMsg); err != nil {
				log.Printf("Error parsing network control message: %v", err)
				continue
			}

			log.Printf("KNIRVROUTER received network control message: %s", networkMsg.Type)

			// Handle different network control message types
			switch networkMsg.Type {
			case "NetworkPause":
				pm.handleNetworkPause(networkMsg.Payload, msg.ReceivedFrom.String())
			case "NetworkResume":
				pm.handleNetworkResume(networkMsg.Payload, msg.ReceivedFrom.String())
			default:
				log.Printf("Unknown network control message type: %s", networkMsg.Type)
			}
		}
	}
}

// handleNetworkPause processes network pause messages
func (pm *P2PManager) handleNetworkPause(payload interface{}, senderPeerID string) {
	// Parse the pause payload
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshaling pause payload: %v", err)
		return
	}

	var pausePayload NetworkPausePayload
	if err := json.Unmarshal(payloadBytes, &pausePayload); err != nil {
		log.Printf("Error unmarshaling pause payload: %v", err)
		return
	}

	initiatorPeerID := pausePayload.InitiatorPeerID
	reason := pausePayload.Reason

	// Set network paused state
	pm.pauseMutex.Lock()
	pm.networkPaused = true
	pm.pausedUntil = time.Now().Add(NetworkPauseTimeout)
	pm.pauseMutex.Unlock()

	log.Printf("KNIRVROUTER Network PAUSED by %s until %s - Reason: %s",
		initiatorPeerID,
		pm.pausedUntil.Format("2006-01-02 15:04:05 UTC"),
		reason)

	log.Printf("KNIRVROUTER operations paused - rejecting new route announcements")
}

// handleNetworkResume processes network resume messages
func (pm *P2PManager) handleNetworkResume(payload interface{}, senderPeerID string) {
	pm.pauseMutex.Lock()
	pm.networkPaused = false
	pm.pauseMutex.Unlock()

	log.Printf("KNIRVROUTER Network RESUMED by %s", senderPeerID)
}
