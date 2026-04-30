package p2p

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"KNIRVCHAIN/config"
)

const (
	BlockTopic          = "blocks"
	TransactionTopic    = "transactions"
	ChainSyncProtocolID = "/knirv/chain-sync/1.0.0"

	BlockValidationTimeout  = 10 * time.Second
	TransactionValidTimeout = 5 * time.Second
	NetworkPauseTimeout     = 5 * time.Minute
	GossipHeartbeat         = 1 * time.Second
)

// Blockchain defines the subset of blockchain operations needed by P2PConsensusManager.
// All block/transaction data is passed as raw JSON to avoid cross-package type dependencies.
type Blockchain interface {
	GetChainID() string
	GetChainAddress() string
	IsActivelyMining() bool
	Lock()
	Unlock()
	GetChainLength() int
	GetLatestBlockNumber() uint64
	GetLatestBlockHash() string
	AddBlockFromJSON(data []byte) error
	AddTransactionFromJSON(data []byte) error
	GetBlocksJSONAfter(startAfter uint64) ([]byte, error)
}

// P2PConsensusManager manages consensus via KNIRVGATEWAY P2P proxy.
type P2PConsensusManager struct {
	blockchain Blockchain
	nodeRole   config.Role
	ctx        context.Context
	cancel     context.CancelFunc

	gatewaySocket string
	gatewayClient *http.Client

	miningLocked   bool
	isSyncing      bool
	networkPaused  bool
	pausedUntil    time.Time
	updateRequired bool
	mu             sync.Mutex
	stopChan       chan struct{}
}

// NewP2PConsensusManager creates a new gateway-proxied P2P consensus manager.
func NewP2PConsensusManager(blockchain Blockchain, discoveryManager DiscoveryService, role config.Role) (*P2PConsensusManager, error) {
	ctx, cancel := context.WithCancel(context.Background())

	gatewaySocket := ""
	if dm, ok := discoveryManager.(interface{ GetGatewaySocket() string }); ok {
		gatewaySocket = dm.GetGatewaySocket()
	}

	var gatewayClient *http.Client
	if gatewaySocket != "" {
		gatewayClient = &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				DialContext: func(gctx context.Context, _, _ string) (net.Conn, error) {
					return (&net.Dialer{}).DialContext(gctx, "unix", gatewaySocket)
				},
			},
		}
	}

	manager := &P2PConsensusManager{
		blockchain:    blockchain,
		nodeRole:      role,
		ctx:           ctx,
		cancel:        cancel,
		gatewaySocket: gatewaySocket,
		gatewayClient: gatewayClient,
		stopChan:      make(chan struct{}),
	}

	return manager, nil
}

// Start begins the consensus process.
func (pcm *P2PConsensusManager) Start() {
	log.Printf("[%s][%s] Starting P2P consensus manager...", pcm.nodeRole.String(), pcm.blockchain.GetChainID())
	go pcm.runForkResolution()
	log.Printf("[%s][%s] P2P consensus manager started successfully.", pcm.nodeRole.String(), pcm.blockchain.GetChainID())
}

// HandleReceivedBlockData processes a block received from KNIRVGATEWAY.
func (pcm *P2PConsensusManager) HandleReceivedBlockData(data []byte) {
	var header struct {
		BlockNumber uint64 `json:"block_number"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		log.Printf("[%s][%s] Error decoding block from gateway: %v", pcm.nodeRole.String(), pcm.blockchain.GetChainID(), err)
		return
	}
	log.Printf("[%s][%s] Received block #%d via KNIRVGATEWAY", pcm.nodeRole.String(), pcm.blockchain.GetChainID(), header.BlockNumber)
	pcm.processReceivedBlock(header.BlockNumber, data)
}

// HandleReceivedTransactionData processes a transaction received from KNIRVGATEWAY.
func (pcm *P2PConsensusManager) HandleReceivedTransactionData(data []byte) {
	var header struct {
		TransactionHash string `json:"transaction_hash"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		log.Printf("[%s][%s] Error decoding transaction from gateway: %v", pcm.nodeRole.String(), pcm.blockchain.GetChainID(), err)
		return
	}
	log.Printf("[%s][%s] Received transaction %s via KNIRVGATEWAY", pcm.nodeRole.String(), pcm.blockchain.GetChainID(), header.TransactionHash)
	pcm.processReceivedTransaction(data)
}

// HandleSyncRequest handles a chain sync request proxied from KNIRVGATEWAY.
// Returns JSON-encoded response.
func (pcm *P2PConsensusManager) HandleSyncRequest(data []byte) ([]byte, error) {
	var request map[string]interface{}
	if err := json.Unmarshal(data, &request); err != nil {
		return nil, fmt.Errorf("decode sync request: %w", err)
	}

	if _, ok := request["getStatus"]; ok {
		blockNum := pcm.blockchain.GetLatestBlockNumber()
		blockHash := pcm.blockchain.GetLatestBlockHash()
		return json.Marshal(map[string]interface{}{
			"latest_block_number": blockNum,
			"latest_block_hash":   blockHash,
		})
	}

	if payload, ok := request["getBlocks"]; ok {
		m, _ := payload.(map[string]interface{})
		startAfterF, _ := m["start_after"].(float64)
		startAfter := uint64(startAfterF)
		return pcm.blockchain.GetBlocksJSONAfter(startAfter)
	}

	return nil, fmt.Errorf("unknown sync request type")
}

// processReceivedBlock validates and adds a block received from the network.
func (pcm *P2PConsensusManager) processReceivedBlock(blockNum uint64, blockData []byte) {
	if pcm.blockchain.IsActivelyMining() {
		log.Printf("[%s][%s] Skipping received block processing as we're actively mining", pcm.nodeRole.String(), pcm.blockchain.GetChainID())
		return
	}

	pcm.lockMining()
	defer pcm.unlockMining()
	log.Printf("[%s][%s] Processing received block #%d", pcm.nodeRole.String(), pcm.blockchain.GetChainID(), blockNum)

	currentBlockNum := pcm.blockchain.GetLatestBlockNumber()

	if blockNum == currentBlockNum+1 {
		log.Printf("[%s][%s] Adding block #%d to our chain", pcm.nodeRole.String(), pcm.blockchain.GetChainID(), blockNum)
		if err := pcm.blockchain.AddBlockFromJSON(blockData); err != nil {
			log.Printf("[%s][%s] Failed to add block #%d: %v", pcm.nodeRole.String(), pcm.blockchain.GetChainID(), blockNum, err)
		}
		return
	}

	if blockNum > currentBlockNum {
		log.Printf("[%s][%s] Received block #%d is ahead of our chain (at #%d), triggering fork resolution",
			pcm.nodeRole.String(), pcm.blockchain.GetChainID(), blockNum, currentBlockNum)
		pcm.requestChainFromPeers()
	}
}

// processReceivedTransaction validates and adds a transaction received from the network.
func (pcm *P2PConsensusManager) processReceivedTransaction(txData []byte) {
	if err := pcm.blockchain.AddTransactionFromJSON(txData); err != nil {
		log.Printf("[%s][%s] Failed to add received transaction: %v", pcm.nodeRole.String(), pcm.blockchain.GetChainID(), err)
	}
}

// BroadcastBlock publishes a block via KNIRVGATEWAY.
func (pcm *P2PConsensusManager) BroadcastBlock(block *Block) error {
	blockData, err := json.Marshal(block)
	if err != nil {
		return fmt.Errorf("failed to marshal block: %w", err)
	}

	if err := pcm.publishToGateway("/p2p/publish-block", blockData); err != nil {
		return fmt.Errorf("failed to publish block: %w", err)
	}

	log.Printf("[%s][%s] Block #%d broadcast via KNIRVGATEWAY", pcm.nodeRole.String(), pcm.blockchain.GetChainID(), block.BlockNumber)
	return nil
}

// BroadcastTransaction publishes a transaction via KNIRVGATEWAY.
func (pcm *P2PConsensusManager) BroadcastTransaction(transaction *Transaction) error {
	txData, err := json.Marshal(transaction)
	if err != nil {
		return fmt.Errorf("failed to marshal transaction: %w", err)
	}

	if err := pcm.publishToGateway("/p2p/publish-tx", txData); err != nil {
		return fmt.Errorf("failed to publish transaction: %w", err)
	}

	log.Printf("[%s][%s] Transaction %s broadcast via KNIRVGATEWAY", pcm.nodeRole.String(), pcm.blockchain.GetChainID(), transaction.TransactionHash)
	return nil
}

// publishToGateway sends data to KNIRVGATEWAY via unix socket.
func (pcm *P2PConsensusManager) publishToGateway(path string, data []byte) error {
	if pcm.gatewayClient == nil {
		return fmt.Errorf("gateway client not available")
	}
	resp, err := pcm.gatewayClient.Post("http://localhost"+path, "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("gateway returned %d", resp.StatusCode)
	}
	return nil
}

// requestChainFromPeers asks KNIRVGATEWAY to find peers and sync chain.
func (pcm *P2PConsensusManager) requestChainFromPeers() {
	pcm.mu.Lock()
	if pcm.isSyncing {
		log.Printf("[%s][%s] Sync already in progress, skipping.", pcm.nodeRole.String(), pcm.blockchain.GetChainID())
		pcm.mu.Unlock()
		return
	}
	pcm.isSyncing = true
	pcm.mu.Unlock()

	defer func() {
		pcm.mu.Lock()
		pcm.isSyncing = false
		pcm.mu.Unlock()
		log.Printf("[%s][%s] P2P chain sync check finished.", pcm.nodeRole.String(), pcm.blockchain.GetChainID())
	}()

	log.Printf("[%s][%s] Chain sync delegated to KNIRVGATEWAY P2P layer.", pcm.nodeRole.String(), pcm.blockchain.GetChainID())
}

// runForkResolution periodically triggers sync checks.
func (pcm *P2PConsensusManager) runForkResolution() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !pcm.blockchain.IsActivelyMining() {
				pcm.requestChainFromPeers()
			}
		case <-pcm.stopChan:
			return
		}
	}
}


func (pcm *P2PConsensusManager) lockMining() {
	pcm.mu.Lock()
	defer pcm.mu.Unlock()
	pcm.miningLocked = true
}

func (pcm *P2PConsensusManager) unlockMining() {
	pcm.mu.Lock()
	defer pcm.mu.Unlock()
	pcm.miningLocked = false
}

// Stop gracefully shuts down the consensus manager.
func (pcm *P2PConsensusManager) Stop() {
	log.Printf("[%s][%s] Stopping P2P consensus manager...", pcm.nodeRole.String(), pcm.blockchain.GetChainID())
	pcm.mu.Lock()
	if pcm.stopChan != nil {
		select {
		case <-pcm.stopChan:
		default:
			close(pcm.stopChan)
		}
	}
	pcm.mu.Unlock()
	if pcm.cancel != nil {
		pcm.cancel()
	}
	log.Printf("[%s][%s] P2P consensus manager stopped.", pcm.nodeRole.String(), pcm.blockchain.GetChainID())
}

// GetStatus returns the consensus status string.
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

// GetPeerCount returns 0 — peers are managed by KNIRVGATEWAY.
func (pcm *P2PConsensusManager) GetPeerCount() int {
	if pcm.gatewayClient == nil {
		return 0
	}
	resp, err := pcm.gatewayClient.Get("http://localhost/p2p/peers")
	if err != nil {
		return 0
	}
	defer resp.Body.Close()
	var result struct {
		Peers []string `json:"peers"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0
	}
	return len(result.Peers)
}

// IsSyncing returns true if chain sync is in progress.
func (pcm *P2PConsensusManager) IsSyncing() bool {
	pcm.mu.Lock()
	defer pcm.mu.Unlock()
	return pcm.isSyncing
}

// GetMiningLockState returns whether mining is currently locked.
func (pcm *P2PConsensusManager) GetMiningLockState() bool {
	pcm.mu.Lock()
	defer pcm.mu.Unlock()
	return pcm.miningLocked
}

// GetUpdateRequired returns whether a chain update is pending.
func (pcm *P2PConsensusManager) GetUpdateRequired() bool {
	pcm.mu.Lock()
	defer pcm.mu.Unlock()
	return pcm.updateRequired
}

// IsNetworkPaused returns network pause state.
func (pcm *P2PConsensusManager) IsNetworkPaused() bool {
	pcm.mu.Lock()
	defer pcm.mu.Unlock()
	if pcm.networkPaused && time.Now().After(pcm.pausedUntil) {
		pcm.networkPaused = false
	}
	return pcm.networkPaused
}

// GetGatewaySocket returns the unix socket path for KNIRVGATEWAY.
func (pcm *P2PConsensusManager) GetGatewaySocket() string {
	return pcm.gatewaySocket
}

// Unused request context for ContextNode compatibility.
func (pcm *P2PConsensusManager) GetContext() context.Context {
	return pcm.ctx
}
