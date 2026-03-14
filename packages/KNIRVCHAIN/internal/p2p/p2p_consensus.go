package p2p

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"KNIRVCHAIN/config" // Added for config.Role
	"KNIRVCHAIN/internal/utils"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

const (
	// Topic names for pubsub
	BlockTopic       = "blocks"
	TransactionTopic = "transactions"

	// Chain sync protocol
	ChainSyncProtocolID = "/knirv/chain-sync/1.0.0"

	// Validation parameters
	BlockValidationTimeout  = 10 * time.Second
	TransactionValidTimeout = 5 * time.Second

	// Network control parameters (use shared constants from failover_manager.go)
	NetworkPauseTimeout = 5 * time.Minute

	// Gossip parameters
	GossipHeartbeat = 1 * time.Second
)

// Note: Network control message types are defined in failover_manager.go

// Chain sync message types
type GetStatusRequest struct {
	// Empty for now, could include requester's chain info
}

type StatusResponse struct {
	LatestBlockNumber uint64 `json:"latest_block_number"`
	LatestBlockHash   string `json:"latest_block_hash"` // Hex encoded
}

type GetBlocksRequest struct {
	StartAfter uint64 `json:"start_after"`
	Limit      uint64 `json:"limit,omitempty"` // Optional limit
}

type BlocksResponse struct {
	Blocks []*Block `json:"blocks"`
}

type ErrorResponse struct {
	Message string `json:"message"`
}

// DB defines the subset of database methods used by the P2P manager
type DB interface {
	PutIntoDb(interface{}, string) error
	GetContextRecord(txHash string) (interface{}, error)
}

// Blockchain defines the subset of blockchain methods used by the P2P manager
type Blockchain interface {
	GetChainID() string
	GetChainAddress() string
	IsActivelyMining() bool
	Lock()
	Unlock()
	GetBlocks() interface{}                            // Returns []*Block but via interface{} to avoid cross-package type requirements
	SetBlocks(interface{})                             // Accepts []*Block but via interface{}
	AddBlock(interface{}) error                        // Accepts *Block but via interface{}
	GetTransactionPool() interface{}                   // Returns []*Transaction but via interface{}
	SetTransactionPool(interface{})                    // Accepts []*Transaction but via interface{}
	AddTransactionToTransactionPool(interface{}) error // Accepts *Transaction but via interface{}
}

// P2PConsensusManager implements a decentralized consensus mechanism using libp2p pubsub
type P2PConsensusManager struct {
	// Core components
	host             host.Host
	pubsub           *pubsub.PubSub
	blockchain       Blockchain
	db               DB
	discoveryManager *DiscoveryManager
	ctx              context.Context
	cancel           context.CancelFunc

	// PubSub topics and subscriptions
	blockTopic          *pubsub.Topic
	blockSub            *pubsub.Subscription
	transactionTopic    *pubsub.Topic
	transactionSub      *pubsub.Subscription
	networkControlTopic *pubsub.Topic
	networkControlSub   *pubsub.Subscription

	// Consensus state
	miningLocked   bool
	isSyncing      bool
	networkPaused  bool
	pausedUntil    time.Time
	updateRequired bool
	mu             sync.Mutex
	stopChan       chan struct{}
	nodeRole       config.Role // Added

	// Fork resolution (removed unused longestChain)
}

// NewP2PConsensusManager creates a new P2P consensus manager
func NewP2PConsensusManager(blockchain Blockchain, db DB, discoveryManager DiscoveryService, role config.Role) (*P2PConsensusManager, error) {
	// Create a new context
	ctx, cancel := context.WithCancel(context.Background())

	// Get the host from the discovery manager, if possible
	var host host.Host
	if dm, ok := discoveryManager.(*DiscoveryManager); ok {
		host = dm.host
	}

	// Create a new pubsub instance with options based on role
	// Using GossipSub as it's more efficient than FloodSub for larger networks
	var ps *pubsub.PubSub
	var err error

	// Configure pubsub options based on role
	switch role {
	case config.Root:
		// Root nodes need higher message limits and caching for reliability
		ps, err = pubsub.NewGossipSub(ctx, host,
			pubsub.WithMessageIdFn(pubsub.DefaultMsgIdFn), // Use default message ID function
			pubsub.WithValidateQueueSize(1024),            // Larger validation queue
			pubsub.WithPeerOutboundQueueSize(1024),        // Larger outbound queue
			pubsub.WithValidateThrottle(2048))             // Higher validation throughput
	case config.RoleBootnode:
		// Bootnodes need to handle more connections and messages
		ps, err = pubsub.NewGossipSub(ctx, host,
			pubsub.WithMessageIdFn(pubsub.DefaultMsgIdFn), // Use default message ID function
			pubsub.WithValidateQueueSize(512),             // Medium validation queue
			pubsub.WithPeerOutboundQueueSize(512),         // Medium outbound queue
			pubsub.WithValidateThrottle(1024))             // Medium validation throughput
	default:
		// Default options for nodes and clients
		ps, err = pubsub.NewGossipSub(ctx, host)
	}

	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create pubsub: %w", err)
	}

	manager := &P2PConsensusManager{
		host:             host,
		pubsub:           ps,
		blockchain:       blockchain,
		db:               db,
		discoveryManager: discoveryManager.(*DiscoveryManager),
		ctx:              ctx,
		cancel:           cancel,
		stopChan:         make(chan struct{}),
		isSyncing:        false,
		nodeRole:         role, // Initialize role
	}

	// Initialize pubsub topics and subscriptions
	if err := manager.setupPubSub(); err != nil {
		cancel()
		return nil, err
	}

	// Register chain sync handler
	manager.registerSyncHandler()

	return manager, nil
}

// setupPubSub initializes the pubsub topics and subscriptions
func (pcm *P2PConsensusManager) setupPubSub() error {
	var err error

	// Define topic names with chain ID prefix
	blockTopicName := fmt.Sprintf("%s.%s", pcm.blockchain.GetChainID(), BlockTopic)
	transactionTopicName := fmt.Sprintf("%s.%s", pcm.blockchain.GetChainID(), TransactionTopic)

	// We don't need topic-specific options for now
	// The pubsub instance is already configured with role-specific options
	// Just use empty slices for topic options
	var blockTopicOpts []pubsub.TopicOpt
	var txTopicOpts []pubsub.TopicOpt

	// Log different validation strategies based on node role
	switch pcm.nodeRole {
	case config.Root:
		// Root nodes need stricter validation but higher throughput
		log.Printf("[%s][%s] Using Root node topic validation settings",
			pcm.nodeRole.String(), pcm.blockchain.GetChainID())
	case config.RoleBootnode:
		// Bootnodes need balanced validation and throughput
		log.Printf("[%s][%s] Using Bootnode topic validation settings",
			pcm.nodeRole.String(), pcm.blockchain.GetChainID())
	default:
		// Default options for nodes and clients
		log.Printf("[%s][%s] Using default topic validation settings",
			pcm.nodeRole.String(), pcm.blockchain.GetChainID())
	}

	// Join the block topic with role-specific options
	pcm.blockTopic, err = pcm.pubsub.Join(blockTopicName, blockTopicOpts...)
	if err != nil {
		return fmt.Errorf("failed to join block topic: %w", err)
	}

	// Subscribe to the block topic
	pcm.blockSub, err = pcm.blockTopic.Subscribe()
	if err != nil {
		return fmt.Errorf("failed to subscribe to block topic: %w", err)
	}

	// Join the transaction topic with role-specific options
	pcm.transactionTopic, err = pcm.pubsub.Join(transactionTopicName, txTopicOpts...)
	if err != nil {
		return fmt.Errorf("failed to join transaction topic: %w", err)
	}

	// Subscribe to the transaction topic
	pcm.transactionSub, err = pcm.transactionTopic.Subscribe()
	if err != nil {
		return fmt.Errorf("failed to subscribe to transaction topic: %w", err)
	}

	// Join the network control topic
	networkControlTopicName := NetworkControlTopic // Using shared constant
	pcm.networkControlTopic, err = pcm.pubsub.Join(networkControlTopicName)
	if err != nil {
		return fmt.Errorf("failed to join network control topic: %w", err)
	}

	// Subscribe to the network control topic
	pcm.networkControlSub, err = pcm.networkControlTopic.Subscribe()
	if err != nil {
		return fmt.Errorf("failed to subscribe to network control topic: %w", err)
	}

	log.Printf("[%s][%s] P2P consensus manager subscribed to topics: %s, %s, %s",
		pcm.nodeRole.String(), pcm.blockchain.GetChainID(),
		blockTopicName, transactionTopicName, networkControlTopicName)

	return nil
}

// Start begins the consensus process
func (pcm *P2PConsensusManager) Start() {
	log.Printf("[%s][%s] Starting P2P consensus manager...", pcm.nodeRole.String(), pcm.blockchain.GetChainID())

	// Start the block handler
	go pcm.handleBlocks()

	// Start the transaction handler
	go pcm.handleTransactions()

	// Start the network control handler
	go pcm.handleNetworkControl()

	// Start the fork resolution process
	go pcm.runForkResolution()

	log.Printf("[%s][%s] P2P consensus manager started successfully.", pcm.nodeRole.String(), pcm.blockchain.GetChainID())
}

// handleBlocks processes incoming blocks from the network
func (pcm *P2PConsensusManager) handleBlocks() {
	for {
		msg, err := pcm.blockSub.Next(pcm.ctx)
		if err != nil {
			if pcm.ctx.Err() != nil {
				// Context was canceled, exit gracefully
				return
			}
			log.Printf("[%s][%s] Error receiving block from pubsub: %v", pcm.nodeRole.String(), pcm.blockchain.GetChainID(), err)
			continue
		}

		// Skip messages from ourselves
		if msg.ReceivedFrom == pcm.host.ID() {
			continue
		}

		// Decode the block
		var block Block
		if err := json.Unmarshal(msg.Data, &block); err != nil {
			log.Printf("[%s][%s] Error decoding block: %v", pcm.nodeRole.String(), pcm.blockchain.GetChainID(), err)
			continue
		}

		log.Printf("[%s][%s] Received block #%d from node %s", pcm.nodeRole.String(), pcm.blockchain.GetChainID(), block.BlockNumber, msg.ReceivedFrom.String())

		// Process the block
		pcm.processReceivedBlock(&block)
	}
}

// handleTransactions processes incoming transactions from the network
func (pcm *P2PConsensusManager) handleTransactions() {
	for {
		msg, err := pcm.transactionSub.Next(pcm.ctx)
		if err != nil {
			if pcm.ctx.Err() != nil {
				// Context was canceled, exit gracefully
				return
			}
			log.Printf("[%s][%s] Error receiving transaction from pubsub: %v", pcm.nodeRole.String(), pcm.blockchain.GetChainID(), err)
			continue
		}

		// Skip messages from ourselves
		if msg.ReceivedFrom == pcm.host.ID() {
			continue
		}

		// Decode the transaction
		var transaction Transaction
		if err := json.Unmarshal(msg.Data, &transaction); err != nil {
			log.Printf("[%s][%s] Error decoding transaction: %v", pcm.nodeRole.String(), pcm.blockchain.GetChainID(), err)
			continue
		}

		log.Printf("[%s][%s] Received transaction %s from node %s", pcm.nodeRole.String(), pcm.blockchain.GetChainID(), transaction.TransactionHash, msg.ReceivedFrom.String())

		// Process the transaction
		pcm.processReceivedTransaction(&transaction)
	}
}

// processReceivedBlock validates and adds a block received from the network
func (pcm *P2PConsensusManager) processReceivedBlock(block *Block) {
	// Skip if we're actively mining
	if pcm.blockchain.IsActivelyMining() {
		log.Printf("[%s][%s] Skipping received block processing as we're actively mining", pcm.nodeRole.String(), pcm.blockchain.GetChainID())
		return
	}

	// Lock mining during block processing
	pcm.lockMining()
	defer pcm.unlockMining()
	log.Printf("[%s][%s] Processing received block #%d", pcm.nodeRole.String(), pcm.blockchain.GetChainID(), block.BlockNumber)
	// Verify the block
	if !block.VerifyBlock() {
		log.Printf("Received invalid block #%d, ignoring", block.BlockNumber)
		return
	}

	// Check if the block extends our current chain
	pcm.blockchain.Lock()
	blocks := pcm.blockchain.GetBlocks().([]*Block)
	currentLastBlock := blocks[len(blocks)-1]
	pcm.blockchain.Unlock()

	// If the block extends our current chain, add it
	if block.BlockNumber == currentLastBlock.BlockNumber+1 && block.PrevHashString() == currentLastBlock.HashString() {
		log.Printf("[%s][%s] Adding block #%d to our chain", pcm.nodeRole.String(), pcm.blockchain.GetChainID(), block.BlockNumber)
		pcm.blockchain.AddBlock(block)
		return
	}

	// If the block is part of a potentially longer chain, trigger fork resolution
	if block.BlockNumber > currentLastBlock.BlockNumber {
		log.Printf("[%s][%s] Received block #%d is ahead of our chain (at #%d), triggering fork resolution",
			pcm.nodeRole.String(), pcm.blockchain.GetChainID(), block.BlockNumber, currentLastBlock.BlockNumber)

		// Request the full chain from the node who sent this block
		// This will be handled by the fork resolution process
		pcm.requestChainFromPeers()
	}
}

// processReceivedTransaction validates and adds a transaction received from the network
func (pcm *P2PConsensusManager) processReceivedTransaction(transaction *Transaction) {
	// Verify the transaction
	if !transaction.VerifyTxn() {
		log.Printf("[%s][%s] Received invalid transaction %s, ignoring", pcm.nodeRole.String(), pcm.blockchain.GetChainID(), transaction.TransactionHash)
		return
	}

	// Add the transaction to our pool
	pcm.blockchain.AddTransactionToTransactionPool(transaction)
}

// BroadcastBlock publishes a block to the network using pubsub
func (pcm *P2PConsensusManager) BroadcastBlock(block *Block) error {
	// Marshal the block to JSON
	blockData, err := json.Marshal(block)
	if err != nil {
		return fmt.Errorf("failed to marshal block: %w", err)
	}

	// Publish the block to the network
	err = pcm.blockTopic.Publish(pcm.ctx, blockData)
	if err != nil {
		return fmt.Errorf("failed to publish block: %w", err)
	}

	log.Printf("[%s][%s] Block #%d broadcast to the network", pcm.nodeRole.String(), pcm.blockchain.GetChainID(), block.BlockNumber)
	return nil
}

// BroadcastTransaction publishes a transaction to the network using pubsub
func (pcm *P2PConsensusManager) BroadcastTransaction(transaction *Transaction) error {
	// Marshal the transaction to JSON
	txData, err := json.Marshal(transaction)
	if err != nil {
		return fmt.Errorf("failed to marshal transaction: %w", err)
	}

	// Publish the transaction to the network
	err = pcm.transactionTopic.Publish(pcm.ctx, txData)
	if err != nil {
		return fmt.Errorf("failed to publish transaction: %w", err)
	}

	log.Printf("[%s][%s] Transaction %s broadcast to the network", pcm.nodeRole.String(), pcm.blockchain.GetChainID(), transaction.TransactionHash)
	return nil
}

// runForkResolution periodically checks for and resolves blockchain forks
func (pcm *P2PConsensusManager) runForkResolution() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Skip if we're actively mining
			if pcm.blockchain.IsActivelyMining() {
				continue
			}

			// Request chain data from nodes
			pcm.requestChainFromPeers()

		case <-pcm.stopChan:
			return
		}
	}
}

// registerSyncHandler registers the chain sync stream handler
func (pcm *P2PConsensusManager) registerSyncHandler() {
	pcm.host.SetStreamHandler(ChainSyncProtocolID, pcm.handleSyncStream)
	log.Printf("[%s][%s] Registered chain sync handler for protocol %s", pcm.nodeRole.String(), pcm.blockchain.GetChainID(), ChainSyncProtocolID)
}

// handleSyncStream handles incoming chain sync requests
func (pcm *P2PConsensusManager) handleSyncStream(stream network.Stream) {
	defer stream.Close()
	nodeID := stream.Conn().RemotePeer()
	log.Printf("[%s][%s] Received chain sync stream from %s", pcm.nodeRole.String(), pcm.blockchain.GetChainID(), nodeID)

	reader := bufio.NewReader(stream)
	writer := bufio.NewWriter(stream)
	decoder := json.NewDecoder(reader)
	encoder := json.NewEncoder(writer)

	// Read request type
	var request map[string]interface{}
	if err := decoder.Decode(&request); err != nil {
		log.Printf("[%s][%s] Error decoding sync request from %s: %v", pcm.nodeRole.String(), pcm.blockchain.GetChainID(), nodeID, err)
		return
	}

	// Handle different request types
	if _, ok := request["getStatus"]; ok {
		// The value associated with "getStatus" would be an empty map if GetStatusRequest{} was sent.
		// This is fine as handleStatusRequest doesn't use the content of GetStatusRequest.
		pcm.handleStatusRequest(encoder, writer)
	} else if getBlocksPayload, ok := request["getBlocks"]; ok {
		// getBlocksPayload is the GetBlocksRequest struct, likely unmarshaled as map[string]interface{}
		getBlocksMap, ok := getBlocksPayload.(map[string]interface{})
		if !ok {
			log.Printf("[%s][%s] Invalid 'getBlocks' payload structure from %s", pcm.nodeRole.String(), pcm.blockchain.GetChainID(), nodeID)
			return
		}
		startAfterFloat, ok := getBlocksMap["start_after"].(float64) // JSON numbers are float64
		if !ok {
			log.Printf("[%s][%s] Invalid 'start_after' in getBlocks request from %s", pcm.nodeRole.String(), pcm.blockchain.GetChainID(), nodeID)
			return
		}
		pcm.handleBlocksRequest(uint64(startAfterFloat), encoder, writer)
	} else {
		log.Printf("[%s][%s] Received unknown sync request from %s: %v", pcm.nodeRole.String(), pcm.blockchain.GetChainID(), nodeID, request)
	}
}

func (pcm *P2PConsensusManager) handleStatusRequest(encoder *json.Encoder, writer *bufio.Writer) {
	pcm.blockchain.Lock()
	blocks := pcm.blockchain.GetBlocks().([]*Block)
	lastBlock := blocks[len(blocks)-1]
	pcm.blockchain.Unlock()

	response := StatusResponse{
		LatestBlockNumber: lastBlock.BlockNumber,
		LatestBlockHash:   lastBlock.HashString(),
	}

	if err := encoder.Encode(response); err != nil {
		log.Printf("[%s][%s] Error encoding status response: %v", pcm.nodeRole.String(), pcm.blockchain.GetChainID(), err)
		return
	}
	if err := writer.Flush(); err != nil {
		log.Printf("[%s][%s] Error flushing status response: %v", pcm.nodeRole.String(), pcm.blockchain.GetChainID(), err)
	}
}

func (pcm *P2PConsensusManager) handleBlocksRequest(startAfter uint64, encoder *json.Encoder, writer *bufio.Writer) {
	pcm.blockchain.Lock()
	defer pcm.blockchain.Unlock()

	var blocks []*Block
	for _, block := range pcm.blockchain.GetBlocks().([]*Block) {
		if block.BlockNumber > startAfter {
			blocks = append(blocks, block)
		}
	}

	response := BlocksResponse{Blocks: blocks}
	if err := encoder.Encode(response); err != nil {
		log.Printf("[%s][%s] Error encoding blocks response: %v", pcm.nodeRole.String(), pcm.blockchain.GetChainID(), err)
		return
	}
	if err := writer.Flush(); err != nil {
		log.Printf("[%s][%s] Error flushing blocks response: %v", pcm.nodeRole.String(), pcm.blockchain.GetChainID(), err)
	}
}

// requestChainFromPeers requests blockchain data from KNIRVCHAIN nodes
func (pcm *P2PConsensusManager) requestChainFromPeers() {
	// --- Prevent Concurrent Sync Runs (within this node) ---
	pcm.mu.Lock()
	if pcm.isSyncing {
		log.Printf("[%s][%s] Sync already in progress, skipping this cycle.", pcm.nodeRole.String(), pcm.blockchain.GetChainID())
		pcm.mu.Unlock()
		return
	}
	pcm.isSyncing = true
	pcm.mu.Unlock()
	// Ensure isSyncing is set back to false when the function finishes
	defer func() {
		pcm.mu.Lock()
		pcm.isSyncing = false
		pcm.mu.Unlock()
		log.Printf("[%s][%s] P2P chain sync check finished.", pcm.nodeRole.String(), pcm.blockchain.GetChainID())
	}()

	// --- Use DiscoveryManager to find relevant nodes ---
	if pcm.discoveryManager == nil {
		log.Printf("[%s][%s] DiscoveryManager not available, cannot find KNIRVCHAIN nodes.", pcm.nodeRole.String(), pcm.blockchain.GetChainID())
		return
	}

	// Find nodes providing our specific chain resource
	agentPeersInfo, err := pcm.discoveryManager.FindResource(context.Background(), pcm.blockchain.GetChainID(), DiscoveryResourceTypeChain)
	if err != nil {
		// Log non-critical errors (like "no providers found") less verbosely for the role
		if !strings.Contains(err.Error(), "no providers found") {
			log.Printf("[%s][%s] Error finding KNIRVCHAIN nodes via DHT: %v", pcm.nodeRole.String(), pcm.blockchain.GetChainID(), err)
		} else {
			log.Printf("[%s][%s] No other KNIRVCHAIN nodes found via DHT for chain sync.", pcm.nodeRole.String(), pcm.blockchain.GetChainID())
		}
		return
	}

	if len(agentPeersInfo) == 0 {
		log.Printf("[%s][%s] No relevant KNIRVCHAIN nodes found to sync with.", pcm.nodeRole.String(), pcm.blockchain.GetChainID())
		return
	}

	log.Printf("[%s][%s] Starting P2P chain sync check with %d potential KNIRVCHAIN node(s)", pcm.nodeRole.String(), pcm.blockchain.GetChainID(), len(agentPeersInfo))

	// --- Iterate through KNIRVCHAIN nodes only ---
	for _, nodeInfo := range agentPeersInfo {
		nodeID := nodeInfo.ID // Extract PeerID from AddrInfo
		if nodeID == pcm.host.ID() {
			continue // Skip self
		}

		// --- Connection Check (Optional but good) ---
		// Ensure we are actually connected before trying to open a stream
		if pcm.host.Network().Connectedness(nodeID) != network.Connected {
			log.Printf("[%s][%s] Found node %s via DHT but not connected, attempting connection...", pcm.nodeRole.String(), pcm.blockchain.GetChainID(), nodeID)
			// Use a timeout for the connection attempt
			connectCtx, connectCancel := context.WithTimeout(pcm.ctx, 15*time.Second)
			err := pcm.host.Connect(connectCtx, nodeInfo) // Use the AddrInfo from FindResource
			connectCancel()
			if err != nil {
				log.Printf("[%s][%s] Failed to connect to node %s: %v", pcm.nodeRole.String(), pcm.blockchain.GetChainID(), nodeID, err)
				continue // Skip to next node if connection fails
			}
			log.Printf("[%s][%s] Successfully connected to node %s", pcm.nodeRole.String(), pcm.blockchain.GetChainID(), nodeID)
		}

		// Open stream with timeout and handle in goroutine
		streamCtx, streamCancel := context.WithTimeout(pcm.ctx, 30*time.Second)
		go func(nodeID peer.ID, ctx context.Context, cancel context.CancelFunc) {
			defer cancel()

			stream, err := pcm.host.NewStream(ctx, nodeID, ChainSyncProtocolID)
			if err != nil {
				log.Printf("[%s][%s] Failed to open sync stream to node %s: %v", pcm.nodeRole.String(), pcm.blockchain.GetChainID(), nodeID, err)
				return
			}
			defer stream.Close()

			reader := bufio.NewReader(stream)
			writer := bufio.NewWriter(stream)
			encoder := json.NewEncoder(writer)
			decoder := json.NewDecoder(reader)

			// 1. Get node status
			statusReqPayload := map[string]interface{}{"getStatus": GetStatusRequest{}}
			if err := encoder.Encode(statusReqPayload); err != nil {
				log.Printf("[%s][%s] Error encoding status request: %v", pcm.nodeRole.String(), pcm.blockchain.GetChainID(), err)
				return
			}
			if err := writer.Flush(); err != nil { // Ensure writer is flushed after encoding
				log.Printf("[%s][%s] Error flushing status request: %v", pcm.nodeRole.String(), pcm.blockchain.GetChainID(), err)
				return
			}

			// 2. Read status response
			var status StatusResponse
			if err := decoder.Decode(&status); err != nil {
				log.Printf("[%s][%s] Error decoding status response: %v", pcm.nodeRole.String(), pcm.blockchain.GetChainID(), err)
				return
			}

			// 3. Compare with our chain
			pcm.blockchain.Lock()
			localLast := pcm.blockchain.GetBlocks().([]*Block)[len(pcm.blockchain.GetBlocks().([]*Block))-1]
			pcm.blockchain.Unlock()

			if status.LatestBlockNumber > localLast.BlockNumber {
				// Request blocks we're missing
				blocksReqPayload := map[string]interface{}{
					"getBlocks": GetBlocksRequest{
						StartAfter: localLast.BlockNumber,
						Limit:      100, // Reasonable batch size
					},
				}
				if err := encoder.Encode(blocksReqPayload); err != nil {
					log.Printf("[%s][%s] Error encoding blocks request: %v", pcm.nodeRole.String(), pcm.blockchain.GetChainID(), err)
					return
				}
				if err := writer.Flush(); err != nil {
					log.Printf("[%s][%s] Error flushing blocks request: %v", pcm.nodeRole.String(), pcm.blockchain.GetChainID(), err)
					return
				}

				// 4. Receive and validate blocks
				var blocksResp BlocksResponse
				if err := decoder.Decode(&blocksResp); err != nil {
					log.Printf("[%s][%s] Error decoding blocks response: %v", pcm.nodeRole.String(), pcm.blockchain.GetChainID(), err)
					return
				}

				if len(blocksResp.Blocks) > 0 {
					// Validate and potentially switch chains
					if valid, err := pcm.validateChain(blocksResp.Blocks); err != nil {
						log.Printf("[%s][%s] Chain validation error: %v", pcm.nodeRole.String(), pcm.blockchain.GetChainID(), err)
						return
					} else if valid {
						newChain := append(pcm.blockchain.GetBlocks().([]*Block)[:len(pcm.blockchain.GetBlocks().([]*Block))-1], blocksResp.Blocks...)
						pcm.switchToChain(newChain)
					}
				}
			}
		}(nodeID, streamCtx, streamCancel)
	}
}

// validateChain checks if a chain is valid
func (pcm *P2PConsensusManager) validateChain(chain []*Block) (bool, error) {
	if len(chain) == 0 {
		return false, nil
	}

	// Validate each block in the chain
	for i := 1; i < len(chain); i++ {
		// Check block linkage
		if chain[i].PrevHashString() != chain[i-1].HashString() {
			return false, nil
		}

		// Verify the block
		if !chain[i].VerifyBlock() {
			return false, nil
		}

		// Verify transactions in the block
		for _, tx := range chain[i].Transactions {
			if tx.From != utils.BLOCKCHAIN_ADDRESS {
				valid, err := tx.VerifySignature()
				if err != nil {
					return false, fmt.Errorf("signature verification failed for tx %s: %w", tx.TransactionHash, err)
				}
				if !valid {
					return false, nil
				}
			}
		}
	}

	return true, nil
}

// switchToChain switches to a new chain if it's valid and longer
func (pcm *P2PConsensusManager) switchToChain(newChain []*Block) {
	// Lock mining during chain switch
	pcm.lockMining()
	defer pcm.unlockMining()

	// Validate the new chain
	valid, err := pcm.validateChain(newChain)
	if err != nil {
		log.Printf("[%s][%s] Error validating chain: %v", pcm.nodeRole.String(), pcm.blockchain.GetChainID(), err)
		return
	}
	if !valid {
		log.Printf("[%s][%s] Received invalid chain, ignoring", pcm.nodeRole.String(), pcm.blockchain.GetChainID())
		return
	}

	// Check if the new chain is longer than our current chain
	pcm.blockchain.Lock()
	currentChainLength := len(pcm.blockchain.GetBlocks().([]*Block))
	pcm.blockchain.Unlock()

	if len(newChain) <= currentChainLength {
		log.Printf("[%s][%s] Received chain is not longer than our current chain, ignoring", pcm.nodeRole.String(), pcm.blockchain.GetChainID())
		return
	}

	log.Printf("[%s][%s] Switching to longer chain (length: %d -> %d)", pcm.nodeRole.String(), pcm.blockchain.GetChainID(), currentChainLength, len(newChain))

	// Update our blockchain with the new chain
	pcm.blockchain.SetBlocks(newChain)

	// Save the updated blockchain to the database
	err = pcm.db.PutIntoDb(pcm.blockchain, pcm.blockchain.GetChainAddress())
	if err != nil {
		log.Printf("[%s][%s] Failed to save updated blockchain to database: %v", pcm.nodeRole.String(), pcm.blockchain.GetChainID(), err)
	}

	// Update the transaction pool
	pcm.updateTransactionPool(newChain)
}

// updateTransactionPool removes confirmed transactions from the pool
func (pcm *P2PConsensusManager) updateTransactionPool(chain []*Block) {
	pcm.blockchain.Lock()
	defer pcm.blockchain.Unlock()

	// Create a map of confirmed transactions
	confirmedTxns := make(map[string]bool)
	for _, block := range chain {
		for _, tx := range block.Transactions {
			confirmedTxns[tx.TransactionHash] = true
		}
	}

	// Filter out confirmed transactions from the pool
	var newPool []*Transaction
	txPool := pcm.blockchain.GetTransactionPool().([]*Transaction)
	for _, tx := range txPool {
		if !confirmedTxns[tx.TransactionHash] {
			newPool = append(newPool, tx)
		}
	}

	// Update the transaction pool
	pcm.blockchain.SetTransactionPool(newPool)
}

// lockMining prevents mining during consensus operations
func (pcm *P2PConsensusManager) lockMining() {
	pcm.mu.Lock()
	defer pcm.mu.Unlock()
	pcm.miningLocked = true
}

// unlockMining allows mining to resume
func (pcm *P2PConsensusManager) unlockMining() {
	pcm.mu.Lock()
	defer pcm.mu.Unlock()
	pcm.miningLocked = false
}

// Stop gracefully shuts down the consensus manager
func (pcm *P2PConsensusManager) Stop() {
	log.Printf("[%s][%s] Stopping P2P consensus manager...", pcm.nodeRole.String(), pcm.blockchain.GetChainID())

	// Use a sync.Once to ensure we only close the channel once
	pcm.mu.Lock()
	if pcm.stopChan != nil {
		select {
		case <-pcm.stopChan:
			// Channel already closed
			log.Printf("[%s][%s] P2P consensus manager stopChan was already closed", pcm.nodeRole.String(), pcm.blockchain.GetChainID())
		default:
			// Close the channel
			close(pcm.stopChan)
			log.Printf("[%s][%s] P2P consensus manager stopChan closed", pcm.nodeRole.String(), pcm.blockchain.GetChainID())
		}
	}
	pcm.mu.Unlock()

	// Cancel the context if it exists
	if pcm.cancel != nil {
		pcm.cancel()
		log.Printf("[%s][%s] P2P consensus manager context canceled", pcm.nodeRole.String(), pcm.blockchain.GetChainID())
	}

	// Close any subscriptions
	pcm.mu.Lock()
	if pcm.blockSub != nil {
		pcm.blockSub.Cancel()
		log.Printf("[%s][%s] Block subscription canceled", pcm.nodeRole.String(), pcm.blockchain.GetChainID())
	}
	if pcm.transactionSub != nil {
		pcm.transactionSub.Cancel()
		log.Printf("[%s][%s] Transaction subscription canceled", pcm.nodeRole.String(), pcm.blockchain.GetChainID())
	}
	pcm.mu.Unlock()

	log.Printf("[%s][%s] P2P consensus manager stopped.", pcm.nodeRole.String(), pcm.blockchain.GetChainID())
}

// GetStatus returns the current P2P consensus status as a string
func (pcm *P2PConsensusManager) GetStatus() string {
	pcm.mu.Lock()
	defer pcm.mu.Unlock()

	if pcm.stopChan == nil {
		return "stopped"
	}
	if pcm.miningLocked {
		return "consensus-active"
	}
	if pcm.isSyncing {
		return "syncing"
	}
	return "active"
}

// GetPeerCount returns the number of active P2P nodes
func (pcm *P2PConsensusManager) GetPeerCount() int {
	if pcm.host == nil {
		return 0
	}

	// Get all connected nodes from libp2p host
	nodes := pcm.host.Network().Peers()
	count := 0

	// Count nodes that have our chain sync protocol
	for _, node := range nodes {
		protocols, err := pcm.host.Peerstore().GetProtocols(node)
		if err == nil {
			for _, proto := range protocols {
				if proto == ChainSyncProtocolID {
					count++
					break
				}
			}
		}
	}
	return count
}

// IsSyncing returns true if the node is currently syncing with nodes
func (pcm *P2PConsensusManager) IsSyncing() bool {
	pcm.mu.Lock()
	defer pcm.mu.Unlock()
	return pcm.isSyncing
}

// handleNetworkControl processes network control messages (pause/resume)
func (pcm *P2PConsensusManager) handleNetworkControl() {
	for {
		select {
		case <-pcm.ctx.Done():
			return
		default:
			msg, err := pcm.networkControlSub.Next(pcm.ctx)
			if err != nil {
				if pcm.ctx.Err() == nil {
					log.Printf("[%s][%s] Error receiving network control message: %v", pcm.nodeRole.String(), pcm.blockchain.GetChainID(), err)
				}
				continue
			}

			// Skip messages from ourselves
			if msg.ReceivedFrom == pcm.host.ID() {
				continue
			}

			// Decode the network control message
			var networkMsg NetworkControlMessage
			if err := json.Unmarshal(msg.Data, &networkMsg); err != nil {
				log.Printf("[%s][%s] Error decoding network control message: %v", pcm.nodeRole.String(), pcm.blockchain.GetChainID(), err)
				continue
			}

			log.Printf("[%s][%s] Received network control message: %s", pcm.nodeRole.String(), pcm.blockchain.GetChainID(), networkMsg.Type)

			// Handle different network control message types
			switch networkMsg.Type {
			case "NetworkPause":
				pcm.handleNetworkPause(networkMsg.Payload, msg.ReceivedFrom.String())
			case "NetworkResume":
				pcm.handleNetworkResume(networkMsg.Payload, msg.ReceivedFrom.String())
			default:
				log.Printf("[%s][%s] Unknown network control message type: %s", pcm.nodeRole.String(), pcm.blockchain.GetChainID(), networkMsg.Type)
			}
		}
	}
}

// handleNetworkPause processes a NetworkPause message and pauses network operations
func (pcm *P2PConsensusManager) handleNetworkPause(payload interface{}, fromPeerID string) {
	pcm.mu.Lock()
	defer pcm.mu.Unlock()

	log.Printf("[%s][%s] Received NetworkPause message from peer: %s", pcm.nodeRole.String(), pcm.blockchain.GetChainID(), fromPeerID)

	// Parse the NetworkPausePayload
	pauseData, ok := payload.(map[string]interface{})
	if !ok {
		log.Printf("[%s][%s] Invalid NetworkPause payload format", pcm.nodeRole.String(), pcm.blockchain.GetChainID())
		return
	}

	// Extract pause information
	initiatorPeerID, ok := pauseData["initiator_peer_id"].(string)
	if !ok {
		log.Printf("[%s][%s] Invalid initiator_peer_id in NetworkPause", pcm.nodeRole.String(), pcm.blockchain.GetChainID())
		return
	}

	reason, ok := pauseData["reason"].(string)
	if !ok {
		log.Printf("[%s][%s] Invalid reason in NetworkPause", pcm.nodeRole.String(), pcm.blockchain.GetChainID())
		return
	}

	// Extract timestamp (optional for future use, but we don't use it for now)
	_, ok = pauseData["timestamp"].(*time.Timer) // We'll extract but won't use
	if !ok {
		log.Printf("[%s][%s] Invalid timestamp in NetworkPause", pcm.nodeRole.String(), pcm.blockchain.GetChainID())
		return
	}

	// Validate and normalize the pause duration
	currentTime := time.Now()

	// Set network paused state
	pcm.networkPaused = true
	pcm.pausedUntil = currentTime.Add(NetworkPauseTimeout)

	log.Printf("[%s][%s] Network PAUSED by %s until %s - Reason: %s",
		pcm.nodeRole.String(),
		pcm.blockchain.GetChainID(),
		initiatorPeerID,
		pcm.pausedUntil.Format("2006-01-02 15:04:05 UTC"),
		reason)

	// Stop accepting new blocks and transactions during pause
	log.Printf("[%s][%s] Network operations paused - rejecting new blocks and transactions", pcm.nodeRole.String(), pcm.blockchain.GetChainID())

	// TODO: Signal to other components (HTTP server, blockchain miner, etc.) to pause
}

// handleNetworkResume processes a NetworkResume message and resumes network operations
func (pcm *P2PConsensusManager) handleNetworkResume(payload interface{}, fromPeerID string) {
	pcm.mu.Lock()
	defer pcm.mu.Unlock()

	log.Printf("[%s][%s] Received NetworkResume message from peer: %s", pcm.nodeRole.String(), pcm.blockchain.GetChainID(), fromPeerID)

	// Validate payload format (even if we don't use specific fields yet)
	if payload != nil {
		if resumeData, ok := payload.(map[string]interface{}); ok {
			if timestamp, exists := resumeData["timestamp"]; exists {
				log.Printf("[%s][%s] Resume message timestamp: %v", pcm.nodeRole.String(), pcm.blockchain.GetChainID(), timestamp)
			}
		}
	}

	// Resume network operations
	pcm.networkPaused = false
	pcm.pausedUntil = time.Time{}

	log.Printf("[%s][%s] Network operations RESUMED by %s", pcm.nodeRole.String(), pcm.blockchain.GetChainID(), fromPeerID)

	// TODO: Signal to other components to resume normal operations
}

// IsNetworkPaused returns true if the network is currently paused
func (pcm *P2PConsensusManager) IsNetworkPaused() bool {
	pcm.mu.Lock()
	defer pcm.mu.Unlock()

	// Check if pause has expired
	if pcm.networkPaused && time.Now().After(pcm.pausedUntil) {
		log.Printf("[%s][%s] Network pause timeout expired, auto-resuming", pcm.nodeRole.String(), pcm.blockchain.GetChainID())
		pcm.networkPaused = false
		pcm.pausedUntil = time.Time{}
		return false
	}

	return pcm.networkPaused
}

// GetPauseStatus returns information about the current network pause state
func (pcm *P2PConsensusManager) GetPauseStatus() (bool, time.Time) {
	pcm.mu.Lock()
	defer pcm.mu.Unlock()

	if pcm.networkPaused && time.Now().After(pcm.pausedUntil) {
		log.Printf("[%s][%s] Network pause timeout expired, auto-resuming", pcm.nodeRole.String(), pcm.blockchain.GetChainID())
		pcm.networkPaused = false
		pcm.pausedUntil = time.Time{}
		return false, time.Time{}
	}

	return pcm.networkPaused, pcm.pausedUntil
}

// GetMiningLockState returns the current mining lock state
func (pcm *P2PConsensusManager) GetMiningLockState() bool {
	pcm.mu.Lock()
	defer pcm.mu.Unlock()
	return pcm.miningLocked
}

// GetUpdateRequired returns whether an update is required
func (pcm *P2PConsensusManager) GetUpdateRequired() bool {
	pcm.mu.Lock()
	defer pcm.mu.Unlock()
	return pcm.updateRequired
}

// Implement missing ConsensusManager interface methods
func (pcm *P2PConsensusManager) StartConsensus(ctx context.Context) error {
	return fmt.Errorf("start consensus not implemented")
}

func (pcm *P2PConsensusManager) StopConsensus() error {
	return fmt.Errorf("stop consensus not implemented")
}

func (pcm *P2PConsensusManager) ProposeValue(value interface{}) error {
	return fmt.Errorf("propose value not implemented")
}

func (pcm *P2PConsensusManager) VoteOnProposal(proposalID string, vote bool) error {
	return fmt.Errorf("vote on proposal not implemented")
}

func (pcm *P2PConsensusManager) GetConsensusState() ConsensusState {
	return ConsensusState{} // Placeholder
}

func (pcm *P2PConsensusManager) AddParticipant(peerID string) error {
	return fmt.Errorf("add participant not implemented")
}

func (pcm *P2PConsensusManager) RemoveParticipant(peerID string) error {
	return fmt.Errorf("remove participant not implemented")
}

func (pcm *P2PConsensusManager) GetParticipants() []string {
	return nil // Placeholder
}

func (pcm *P2PConsensusManager) OnConsensusReached(handler ConsensusHandler) error {
	return fmt.Errorf("on consensus reached not implemented")
}

func (pcm *P2PConsensusManager) OnProposalReceived(handler ProposalHandler) error {
	return fmt.Errorf("on proposal received not implemented")
}
