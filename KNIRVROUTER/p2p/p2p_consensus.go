package p2p

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"KNIRVROUTER_GO_Verifyer/blockchain"
	"KNIRVROUTER_GO_Verifyer/constants"
	"KNIRVROUTER_GO_Verifyer/types"

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

	// Gossip parameters
	GossipHeartbeat = 1 * time.Second
)

// Type aliases for blockchain types
type Block = blockchain.Block
type BlockchainStruct = blockchain.BlockchainStruct
type Transaction = types.Transaction
type LevelDB = blockchain.LevelDB

// Chain sync message types
type GetStatusRequest struct {
	GetStatus bool `json:"getStatus"` // Set to true to request status
}

type StatusResponse struct {
	LatestBlockNumber uint64 `json:"latest_block_number"`
	LatestBlockHash   string `json:"latest_block_hash"` // Hex encoded
}

type GetBlocksRequest struct {
	GetBlocks struct {
		StartAfter uint64 `json:"start_after"`
		Limit      uint64 `json:"limit,omitempty"` // Optional limit
	} `json:"getBlocks"`
}

type BlocksResponse struct {
	Blocks []*Block `json:"blocks"`
}

type ErrorResponse struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

// P2PConsensusManager implements a decentralized consensus mechanism using libp2p pubsub
type P2PConsensusManager struct {
	// Core components
	host             host.Host
	pubsub           *pubsub.PubSub
	blockchain       *BlockchainStruct
	db               *LevelDB
	discoveryManager *DiscoveryManager
	ctx              context.Context
	cancel           context.CancelFunc

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

	// Fork resolution
	longestChain []*Block
}

// NewP2PConsensusManager creates a new P2P consensus manager
func NewP2PConsensusManager(blockchain *BlockchainStruct, db *LevelDB, discoveryManager *DiscoveryManager) (*P2PConsensusManager, error) {
	// Create a new context
	ctx, cancel := context.WithCancel(context.Background())

	// Get the host from the discovery manager
	host := discoveryManager.host

	// Create a new pubsub instance
	// Using GossipSub as it's more efficient than FloodSub for larger networks
	ps, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create pubsub: %w", err)
	}

	manager := &P2PConsensusManager{
		host:             host,
		pubsub:           ps,
		blockchain:       blockchain,
		db:               db,
		discoveryManager: discoveryManager,
		ctx:              ctx,
		cancel:           cancel,
		stopChan:         make(chan struct{}),
		isSyncing:        false,
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

	// Join the block topic
	pcm.blockTopic, err = pcm.pubsub.Join(fmt.Sprintf("%s.%s", pcm.blockchain.ChainID, BlockTopic))
	if err != nil {
		return fmt.Errorf("failed to join block topic: %w", err)
	}

	// Subscribe to the block topic
	pcm.blockSub, err = pcm.blockTopic.Subscribe()
	if err != nil {
		return fmt.Errorf("failed to subscribe to block topic: %w", err)
	}

	// Join the transaction topic
	pcm.transactionTopic, err = pcm.pubsub.Join(fmt.Sprintf("%s.%s", pcm.blockchain.ChainID, TransactionTopic))
	if err != nil {
		return fmt.Errorf("failed to join transaction topic: %w", err)
	}

	// Subscribe to the transaction topic
	pcm.transactionSub, err = pcm.transactionTopic.Subscribe()
	if err != nil {
		return fmt.Errorf("failed to subscribe to transaction topic: %w", err)
	}

	log.Printf("P2P consensus manager subscribed to topics: %s.%s, %s.%s",
		pcm.blockchain.ChainID, BlockTopic,
		pcm.blockchain.ChainID, TransactionTopic)

	return nil
}

// Start begins the consensus process
func (pcm *P2PConsensusManager) Start() {
	log.Println("Starting P2P consensus manager...")

	// Start the block handler
	go pcm.handleBlocks()

	// Start the transaction handler
	go pcm.handleTransactions()

	// Start the fork resolution process
	go pcm.runForkResolution()

	log.Println("P2P consensus manager started successfully")
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
			log.Printf("Error receiving block from pubsub: %v", err)
			continue
		}

		// Skip messages from ourselves
		if msg.ReceivedFrom == pcm.host.ID() {
			continue
		}

		// Decode the block
		var block Block
		if err := json.Unmarshal(msg.Data, &block); err != nil {
			log.Printf("Error decoding block: %v", err)
			continue
		}

		log.Printf("Received block #%d from peer %s", block.Number, msg.ReceivedFrom.String())

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
			log.Printf("Error receiving transaction from pubsub: %v", err)
			continue
		}

		// Skip messages from ourselves
		if msg.ReceivedFrom == pcm.host.ID() {
			continue
		}

		// Decode the transaction
		var transaction Transaction
		if err := json.Unmarshal(msg.Data, &transaction); err != nil {
			log.Printf("Error decoding transaction: %v", err)
			continue
		}

		log.Printf("Received transaction %s from peer %s", transaction.Hash(), msg.ReceivedFrom.String())

		// Process the transaction
		pcm.processReceivedTransaction(&transaction)
	}
}

// processReceivedBlock validates and adds a block received from the network
func (pcm *P2PConsensusManager) processReceivedBlock(block *Block) {
	// Skip if we're actively mining
	if pcm.blockchain.IsActivelyMining() {
		log.Println("Skipping received block processing as we're actively mining")
		return
	}

	// Lock mining during block processing
	pcm.lockMining()
	defer pcm.unlockMining()

	// Verify the block
	if !block.VerifyBlock() {
		log.Printf("Received invalid block #%d, ignoring", block.Number)
		return
	}

	// Check if the block extends our current chain
	pcm.blockchain.Lock()
	currentLastBlock := pcm.blockchain.Blocks[len(pcm.blockchain.Blocks)-1]
	pcm.blockchain.Unlock()

	// If the block extends our current chain, add it
	if block.Number == currentLastBlock.Number+1 && block.PreviousHash == currentLastBlock.Hash() {
		log.Printf("Adding block #%d to our chain", block.Number)
		pcm.blockchain.AddBlock(pcm.db, block)
		return
	}

	// If the block is part of a potentially longer chain, trigger fork resolution
	if block.Number > currentLastBlock.Number {
		log.Printf("Received block #%d is ahead of our chain (at #%d), triggering fork resolution",
			block.Number, currentLastBlock.Number)

		// Request the full chain from the peer who sent this block
		// This will be handled by the fork resolution process
		pcm.requestChainFromPeers()
	}
}

// processReceivedTransaction validates and adds a transaction received from the network
func (pcm *P2PConsensusManager) processReceivedTransaction(transaction *Transaction) {
	// Verify the transaction
	if !transaction.VerifyTxn() {
		log.Printf("Received invalid transaction %s, ignoring", transaction.Hash())
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

	log.Printf("Block #%d broadcast to the network", block.Number)
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

	log.Printf("Transaction %s broadcast to the network", transaction.Hash())
	return nil
}

// runForkResolution periodically checks for and resolves blockchain forks
func (pcm *P2PConsensusManager) runForkResolution() {
	// Use a configurable sync interval with a reasonable default
	syncInterval := 60 * time.Second
	if syncIntervalStr := os.Getenv("CHAIN_SYNC_INTERVAL"); syncIntervalStr != "" {
		if interval, err := strconv.Atoi(syncIntervalStr); err == nil && interval > 0 {
			syncInterval = time.Duration(interval) * time.Second
			log.Printf("[%s] Using custom chain sync interval: %v", pcm.blockchain.ChainID, syncInterval)
		}
	}

	ticker := time.NewTicker(syncInterval)
	defer ticker.Stop()

	log.Printf("[%s] Starting fork resolution with sync interval: %v", pcm.blockchain.ChainID, syncInterval)

	for {
		select {
		case <-ticker.C:
			// Skip if we're actively mining
			if pcm.blockchain.IsActivelyMining() {
				log.Printf("[%s] Skipping chain sync while actively mining", pcm.blockchain.ChainID)
				continue
			}

			log.Printf("[%s] Performing periodic chain sync check", pcm.blockchain.ChainID)
			// Request chain data from peers
			pcm.requestChainFromPeers()

		case <-pcm.stopChan:
			log.Printf("[%s] Fork resolution stopped", pcm.blockchain.ChainID)
			return
		}
	}
}

// registerSyncHandler registers the chain sync stream handler
func (pcm *P2PConsensusManager) registerSyncHandler() {
	pcm.host.SetStreamHandler(ChainSyncProtocolID, pcm.handleSyncStream)
	log.Printf("[%s] Registered chain sync handler for protocol %s", pcm.blockchain.ChainID, ChainSyncProtocolID)
}

// handleSyncStream handles incoming chain sync requests
func (pcm *P2PConsensusManager) handleSyncStream(stream network.Stream) {
	defer stream.Close()
	peerID := stream.Conn().RemotePeer()
	log.Printf("[%s] Received chain sync stream from %s", pcm.blockchain.ChainID, peerID)

	reader := bufio.NewReader(stream)
	writer := bufio.NewWriter(stream)
	decoder := json.NewDecoder(reader)
	encoder := json.NewEncoder(writer)

	// Set a deadline for reading from the stream
	if err := stream.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		log.Printf("[%s] Error setting read deadline: %v", pcm.blockchain.ChainID, err)
	}

	// Read request type
	var request map[string]interface{}
	if err := decoder.Decode(&request); err != nil {
		log.Printf("[%s] Error decoding sync request from %s: %v", pcm.blockchain.ChainID, peerID, err)
		// Send error response
		sendErrorResponse(encoder, writer, "Invalid request format")
		return
	}

	// Handle different request types
	if _, ok := request["getStatus"]; ok {
		pcm.handleStatusRequest(encoder, writer)
	} else if getBlocksData, ok := request["getBlocks"]; ok {
		if getBlocks, ok := getBlocksData.(map[string]interface{}); ok {
			startAfter := uint64(getBlocks["start_after"].(float64))

			// Get limit if provided, otherwise use default
			var limit uint64 = 100 // Default limit
			if limitVal, ok := getBlocks["limit"]; ok {
				limit = uint64(limitVal.(float64))
				if limit == 0 {
					limit = 100 // Use default if 0
				}
			}

			pcm.handleBlocksRequest(startAfter, limit, encoder, writer)
		} else {
			sendErrorResponse(encoder, writer, "Invalid getBlocks format")
		}
	} else {
		log.Printf("[%s] Received unknown sync request from %s", pcm.blockchain.ChainID, peerID)
		sendErrorResponse(encoder, writer, "Unknown request type")
	}
}

// sendErrorResponse sends an error response to the peer
func sendErrorResponse(encoder *json.Encoder, writer *bufio.Writer, message string) {
	errResp := ErrorResponse{}
	errResp.Error.Message = message

	if err := encoder.Encode(errResp); err != nil {
		log.Printf("Error encoding error response: %v", err)
		return
	}
	if err := writer.Flush(); err != nil {
		log.Printf("Error flushing error response: %v", err)
	}
}

func (pcm *P2PConsensusManager) handleStatusRequest(encoder *json.Encoder, writer *bufio.Writer) {
	pcm.blockchain.Lock()
	lastBlock := pcm.blockchain.Blocks[len(pcm.blockchain.Blocks)-1]
	pcm.blockchain.Unlock()

	response := StatusResponse{
		LatestBlockNumber: lastBlock.Number,
		LatestBlockHash:   lastBlock.Hash(),
	}

	log.Printf("[%s] Sending status response: block #%d with hash %s",
		pcm.blockchain.ChainID, response.LatestBlockNumber, response.LatestBlockHash)

	if err := encoder.Encode(response); err != nil {
		log.Printf("[%s] Error encoding status response: %v", pcm.blockchain.ChainID, err)
		return
	}
	if err := writer.Flush(); err != nil {
		log.Printf("[%s] Error flushing status response: %v", pcm.blockchain.ChainID, err)
	}
}

func (pcm *P2PConsensusManager) handleBlocksRequest(startAfter uint64, limit uint64, encoder *json.Encoder, writer *bufio.Writer) {
	pcm.blockchain.Lock()
	defer pcm.blockchain.Unlock()

	// Validate request parameters
	if startAfter >= uint64(len(pcm.blockchain.Blocks)) {
		log.Printf("[%s] Invalid block range requested: startAfter=%d, chain length=%d",
			pcm.blockchain.ChainID, startAfter, len(pcm.blockchain.Blocks))
		sendErrorResponse(encoder, writer, "Invalid block range requested")
		return
	}

	// Enforce reasonable limits
	if limit > 100 {
		limit = 100 // Cap at 100 blocks per response
	}

	var blocks []*Block
	count := uint64(0)
	for _, block := range pcm.blockchain.Blocks {
		if block.Number > startAfter {
			blocks = append(blocks, block)
			count++
			if count >= limit {
				break
			}
		}
	}

	log.Printf("[%s] Sending blocks response: %d blocks starting after #%d",
		pcm.blockchain.ChainID, len(blocks), startAfter)

	response := BlocksResponse{Blocks: blocks}
	if err := encoder.Encode(response); err != nil {
		log.Printf("[%s] Error encoding blocks response: %v", pcm.blockchain.ChainID, err)
		return
	}
	if err := writer.Flush(); err != nil {
		log.Printf("[%s] Error flushing blocks response: %v", pcm.blockchain.ChainID, err)
	}
}

// requestChainFromPeers requests blockchain data from KNIRVROUTER peers
func (pcm *P2PConsensusManager) requestChainFromPeers() {
	// --- Prevent Concurrent Sync Runs (within this node) ---
	pcm.mu.Lock()
	if pcm.isSyncing {
		log.Printf("[%s] Sync already in progress, skipping this cycle.", pcm.blockchain.ChainID)
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
		log.Printf("[%s] P2P chain sync check finished.", pcm.blockchain.ChainID)
	}()

	// --- Use DiscoveryManager to find relevant peers ---
	if pcm.discoveryManager == nil {
		log.Printf("[%s] DiscoveryManager not available, cannot find Knirvchain peers.", pcm.blockchain.ChainID)
		return
	}

	// Find peers providing our specific chain resource
	knirvPeersInfo, err := pcm.discoveryManager.FindResource(pcm.blockchain.ChainID, ResourceTypeChain)
	if err != nil {
		// Log non-critical errors (like "no providers found") less verbosely
		if !strings.Contains(err.Error(), "no providers found") {
			log.Printf("[%s] Error finding Knirvchain peers via DHT: %v", pcm.blockchain.ChainID, err)
		} else {
			log.Printf("[%s] No other Knirvchain peers found via DHT for chain sync.", pcm.blockchain.ChainID)
		}
		return
	}

	if len(knirvPeersInfo) == 0 {
		log.Printf("[%s] No relevant Knirvchain peers found to sync with.", pcm.blockchain.ChainID)
		return
	}

	log.Printf("[%s] Starting P2P chain sync check with %d potential Knirvchain peer(s)", pcm.blockchain.ChainID, len(knirvPeersInfo))

	// --- Iterate through KNIRVROUTER peers only ---
	for _, peerInfo := range knirvPeersInfo {
		peerID := peerInfo.ID // Extract PeerID from AddrInfo
		if peerID == pcm.host.ID() {
			continue // Skip self
		}

		// --- Connection Check (Optional but good) ---
		// Ensure we are actually connected before trying to open a stream
		if pcm.host.Network().Connectedness(peerID) != network.Connected {
			log.Printf("[%s] Found peer %s via DHT but not connected, attempting connection...", pcm.blockchain.ChainID, peerID)
			// Use a timeout for the connection attempt
			connectCtx, connectCancel := context.WithTimeout(pcm.ctx, 15*time.Second)
			err := pcm.host.Connect(connectCtx, peerInfo) // Use the AddrInfo from FindResource
			connectCancel()
			if err != nil {
				log.Printf("[%s] Failed to connect to peer %s: %v", pcm.blockchain.ChainID, peerID, err)
				continue // Skip to next peer if connection fails
			}
			log.Printf("[%s] Successfully connected to peer %s", pcm.blockchain.ChainID, peerID)
		}

		// Open stream with timeout and handle in goroutine
		streamCtx, streamCancel := context.WithTimeout(pcm.ctx, 30*time.Second)
		go func(peerID peer.ID, ctx context.Context, cancel context.CancelFunc) {
			defer cancel()

			stream, err := pcm.host.NewStream(ctx, peerID, ChainSyncProtocolID)
			if err != nil {
				log.Printf("[%s] Failed to open sync stream to peer %s: %v", pcm.blockchain.ChainID, peerID, err)
				return
			}
			defer stream.Close()

			// Set deadlines for the stream
			if err := stream.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
				log.Printf("[%s] Error setting read deadline: %v", pcm.blockchain.ChainID, err)
			}
			if err := stream.SetWriteDeadline(time.Now().Add(30 * time.Second)); err != nil {
				log.Printf("[%s] Error setting write deadline: %v", pcm.blockchain.ChainID, err)
			}

			reader := bufio.NewReader(stream)
			writer := bufio.NewWriter(stream)
			encoder := json.NewEncoder(writer)
			decoder := json.NewDecoder(reader)

			// 1. Get peer status
			statusReq := GetStatusRequest{GetStatus: true}
			log.Printf("[%s] Sending GetStatusRequest to peer %s", pcm.blockchain.ChainID, peerID)
			if err := encoder.Encode(statusReq); err != nil {
				log.Printf("[%s] Error encoding status request: %v", pcm.blockchain.ChainID, err)
				return
			}
			if err := writer.Flush(); err != nil {
				log.Printf("[%s] Error flushing status request: %v", pcm.blockchain.ChainID, err)
				return
			}

			// 2. Read status response
			var status StatusResponse
			if err := decoder.Decode(&status); err != nil {
				log.Printf("[%s] Error decoding status response: %v", pcm.blockchain.ChainID, err)
				return
			}

			log.Printf("[%s] Received StatusResponse from peer %s: block #%d with hash %s",
				pcm.blockchain.ChainID, peerID, status.LatestBlockNumber, status.LatestBlockHash)

			// 3. Compare with our chain
			pcm.blockchain.Lock()
			localLast := pcm.blockchain.Blocks[len(pcm.blockchain.Blocks)-1]
			pcm.blockchain.Unlock()

			if status.LatestBlockNumber > localLast.Number {
				log.Printf("[%s] Peer %s has longer chain (their #%d vs our #%d), requesting blocks",
					pcm.blockchain.ChainID, peerID, status.LatestBlockNumber, localLast.Number)

				// Request blocks we're missing
				req := GetBlocksRequest{}
				req.GetBlocks.StartAfter = localLast.Number
				req.GetBlocks.Limit = 50 // Reasonable batch size

				if err := encoder.Encode(req); err != nil {
					log.Printf("[%s] Error encoding blocks request: %v", pcm.blockchain.ChainID, err)
					return
				}
				if err := writer.Flush(); err != nil {
					log.Printf("[%s] Error flushing blocks request: %v", pcm.blockchain.ChainID, err)
					return
				}

				// 4. Receive and validate blocks
				var blocksResp BlocksResponse
				if err := decoder.Decode(&blocksResp); err != nil {
					log.Printf("[%s] Error decoding blocks response: %v", pcm.blockchain.ChainID, err)

					// Check if it's an error response
					var errResp ErrorResponse
					if err := json.NewDecoder(reader).Decode(&errResp); err == nil && errResp.Error.Message != "" {
						log.Printf("[%s] Received error from peer %s: %s", pcm.blockchain.ChainID, peerID, errResp.Error.Message)
					}
					return
				}

				log.Printf("[%s] Received %d blocks from peer %s", pcm.blockchain.ChainID, len(blocksResp.Blocks), peerID)

				if len(blocksResp.Blocks) > 0 {
					// Validate the received blocks
					log.Printf("[%s] Validating %d blocks received from peer %s", pcm.blockchain.ChainID, len(blocksResp.Blocks), peerID)

					// CRITICAL: Meticulously validate the received blocks
					if pcm.validateReceivedBlocks(blocksResp.Blocks, localLast) {
						log.Printf("[%s] Blocks from peer %s validated successfully, switching chains", pcm.blockchain.ChainID, peerID)

						// Create a new chain by appending the validated blocks to our existing chain
						newChain := make([]*Block, len(pcm.blockchain.Blocks))
						copy(newChain, pcm.blockchain.Blocks)
						newChain = append(newChain, blocksResp.Blocks...)

						// Switch to the new chain
						pcm.switchToChain(newChain)

						// If we didn't get all blocks up to the peer's latest, request more
						if blocksResp.Blocks[len(blocksResp.Blocks)-1].Number < status.LatestBlockNumber {
							log.Printf("[%s] More blocks available from peer %s, will request in next sync cycle",
								pcm.blockchain.ChainID, peerID)
						}
					} else {
						log.Printf("[%s] Blocks from peer %s failed validation, ignoring", pcm.blockchain.ChainID, peerID)
					}
				} else {
					log.Printf("[%s] Peer %s sent empty blocks response despite reporting higher block number",
						pcm.blockchain.ChainID, peerID)
				}
			} else {
				log.Printf("[%s] Our chain (at #%d) is at least as long as peer %s (at #%d), no sync needed",
					pcm.blockchain.ChainID, localLast.Number, peerID, status.LatestBlockNumber)
			}
		}(peerID, streamCtx, streamCancel)
	}
}

// validateReceivedBlocks performs thorough validation of blocks received from peers
func (pcm *P2PConsensusManager) validateReceivedBlocks(blocks []*Block, localLastBlock *Block) bool {
	if len(blocks) == 0 {
		log.Println("No blocks to validate")
		return false
	}

	// Check that the first block links to our last block
	if blocks[0].Number != localLastBlock.Number+1 {
		log.Printf("First received block number (%d) doesn't follow our last block (%d)",
			blocks[0].Number, localLastBlock.Number)
		return false
	}

	if blocks[0].PreviousHash != localLastBlock.Hash() {
		log.Printf("First received block prevHash (%s) doesn't match our last block hash (%s)",
			blocks[0].PreviousHash, localLastBlock.Hash())
		return false
	}

	// Validate each block in sequence
	for i := 0; i < len(blocks); i++ {
		block := blocks[i]

		// Verify block hash meets difficulty
		if !block.VerifyHash() {
			log.Printf("Block #%d has invalid hash for difficulty %d",
				block.Number, constants.MINING_DIFFICULTY)
			return false
		}

		// Verify block number sequence
		if i > 0 && block.Number != blocks[i-1].Number+1 {
			log.Printf("Block number sequence broken: #%d followed by #%d",
				blocks[i-1].Number, block.Number)
			return false
		}

		// Verify block linkage
		if i > 0 && block.PreviousHash != blocks[i-1].Hash() {
			log.Printf("Block linkage broken: block #%d prevHash (%s) doesn't match previous block's hash (%s)",
				block.Number, block.PreviousHash, blocks[i-1].Hash())
			return false
		}

		// Verify timestamps are sequential and reasonable
		if i > 0 && block.Time <= blocks[i-1].Time {
			log.Printf("Block timestamps not sequential: block #%d (%d) <= block #%d (%d)",
				block.Number, block.Time, blocks[i-1].Number, blocks[i-1].Time)
			return false
		}

		// Verify transactions in the block
		for _, tx := range block.Txs {
			// Skip verification for reward transactions
			if tx.From == constants.BLOCKCHAIN_ADDRESS {
				continue
			}

			// Verify transaction signature
			if !tx.VerifyTxn() {
				log.Printf("Transaction %s in block #%d has invalid signature",
					tx.Hash(), block.Number)
				return false
			}

			// Additional transaction validation could be added here
			// e.g., checking for double-spends, valid balances, etc.
		}
	}

	log.Printf("Successfully validated %d blocks", len(blocks))
	return true
}

// validateChain checks if a chain is valid
func (pcm *P2PConsensusManager) validateChain(chain []*Block) bool {
	if len(chain) == 0 {
		log.Println("Validation failed: Chain is empty")
		return false
	}

	// Verify genesis block
	if chain[0].Number != 0 {
		log.Printf("Validation failed: Genesis block number is %d, expected 0", chain[0].Number)
		return false
	}

	// Validate each subsequent block
	for i := 1; i < len(chain); i++ {
		prevBlock := chain[i-1]
		currentBlock := chain[i]

		// Check block number sequence
		if currentBlock.Number != prevBlock.Number+1 {
			log.Printf("Validation failed: Block %d number mismatch (prev %d)",
				currentBlock.Number, prevBlock.Number)
			return false
		}

		// Check previous hash matches
		if currentBlock.PreviousHash != prevBlock.Hash() {
			log.Printf("Validation failed: Block %d PrevHash '%s' does not match previous block hash '%s'",
				currentBlock.Number, currentBlock.PreviousHash, prevBlock.Hash())
			return false
		}

		// Verify block hash meets difficulty
		if !currentBlock.VerifyHash() {
			log.Printf("Validation failed: Block %d hash %s invalid for difficulty %d",
				currentBlock.Number, currentBlock.Hash(), constants.MINING_DIFFICULTY)
			return false
		}

		// Verify transactions in the block
		for _, tx := range currentBlock.Txs {
			if tx.From != constants.BLOCKCHAIN_ADDRESS && !tx.VerifyTxn() {
				log.Printf("Validation failed: Transaction %s in block %d has invalid signature",
					tx.Hash(), currentBlock.Number)
				return false
			}
		}
	}

	log.Printf("Chain validation successful for %d blocks", len(chain))
	return true
}

// switchToChain switches to a new chain if it's valid and longer
func (pcm *P2PConsensusManager) switchToChain(newChain []*Block) {
	// Lock mining during chain switch
	pcm.lockMining()
	defer pcm.unlockMining()

	// Validate the new chain
	if !pcm.validateChain(newChain) {
		log.Println("Received invalid chain, ignoring")
		return
	}

	// Check if the new chain is longer than our current chain
	pcm.blockchain.Lock()
	currentChainLength := len(pcm.blockchain.Blocks)
	pcm.blockchain.Unlock()

	if len(newChain) <= currentChainLength {
		log.Println("Received chain is not longer than our current chain, ignoring")
		return
	}

	log.Printf("Switching to longer chain (length: %d -> %d)", currentChainLength, len(newChain))

	// Update our blockchain with the new chain
	pcm.blockchain.Lock()
	pcm.blockchain.Blocks = newChain
	pcm.blockchain.Unlock()

	// Save each block to the database and update the chain tip
	for i := currentChainLength; i < len(newChain); i++ {
		block := newChain[i]
		// Use the blockchain's method to save blocks to the database
		pcm.blockchain.AddBlock(pcm.db, block)
	}

	// Update the transaction pool
	pcm.updateTransactionPool(newChain)

	log.Printf("Successfully switched to new chain with %d blocks", len(newChain))
}

// updateTransactionPool removes confirmed transactions from the pool
func (pcm *P2PConsensusManager) updateTransactionPool(chain []*Block) {
	pcm.blockchain.Lock()
	defer pcm.blockchain.Unlock()

	// Create a map of confirmed transactions
	confirmedTxns := make(map[string]bool)
	for _, block := range chain {
		for _, tx := range block.Txs {
			confirmedTxns[tx.Hash()] = true
		}
	}

	// Filter out confirmed transactions from the pool
	var newPool []*Transaction
	for _, tx := range pcm.blockchain.TransactionPool {
		if !confirmedTxns[tx.Hash()] {
			newPool = append(newPool, tx)
		}
	}

	// Update the transaction pool
	pcm.blockchain.TransactionPool = newPool
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

// getMiningLockState returns the current mining lock state
func (pcm *P2PConsensusManager) getMiningLockState() bool {
	pcm.mu.Lock()
	defer pcm.mu.Unlock()
	return pcm.miningLocked
}

// Stop gracefully shuts down the consensus manager
func (pcm *P2PConsensusManager) Stop() {
	log.Println("Stopping P2P consensus manager...")

	// Signal all goroutines to stop
	close(pcm.stopChan)

	// Cancel the context
	pcm.cancel()

	log.Println("P2P consensus manager stopped")
}
