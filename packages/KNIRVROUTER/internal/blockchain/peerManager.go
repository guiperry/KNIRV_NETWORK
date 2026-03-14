package blockchain

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"KNIRVROUTER/internal/constants"
	"KNIRVROUTER/internal/interfaces"
	"KNIRVROUTER/internal/types"
)

type PeerTransactionBroadcaster struct {
	PeerManager *PeerManager
}

func (ptb *PeerTransactionBroadcaster) BroadcastTransaction(event interfaces.TransactionEvent) {
	peers := ptb.PeerManager.peerManager.GetPeers()
	for id, peer := range peers {
		if id != ptb.PeerManager.peerManager.Address && peer.Status {
			// event.Transaction is already a *types.Transaction, no conversion needed
			ptb.PeerManager.SendTxnToThePeer(peer.Address, event.Transaction)
			time.Sleep(constants.TXN_BROADCAST_PAUSE_TIME * time.Second)
		}
	}
}

// PeerChainData represents the blockchain data received from a peer
type PeerChainData struct {
	Blocks []*Block
	// Include other relevant data if needed, like transaction pool
	TransactionPool []*types.Transaction
}

type PeerManager struct {
	peerManager                  *types.PeerManager
	TransactionPool              []*types.Transaction
	Blocks                       []*Block
	Mutex                        sync.Mutex
	Broadcaster                  interfaces.Broadcaster
	TransactionAddedSubscription chan interfaces.TransactionEvent
	BlockAddedSubscription       chan interfaces.BlockEvent
}

// GetBlockchain returns the current blockchain
func (pm *PeerManager) GetBlockchain() []*Block {
	pm.Mutex.Lock()
	defer pm.Mutex.Unlock()
	return pm.Blocks
}

// GetTransactionPool returns the current transaction pool
func (pm *PeerManager) GetTransactionPool() []*types.Transaction {
	pm.Mutex.Lock()
	defer pm.Mutex.Unlock()
	return pm.TransactionPool
}

// GetBlockchainLength returns the length of the blockchain
func (pm *PeerManager) GetBlockchainLength() int {
	pm.Mutex.Lock()
	defer pm.Mutex.Unlock()
	return len(pm.Blocks)
}

// GetTransactionPoolLength returns the length of the transaction pool
func (pm *PeerManager) GetTransactionPoolLength() int {
	pm.Mutex.Lock()
	defer pm.Mutex.Unlock()
	return len(pm.TransactionPool)
}

// processTransactionAdded handles new transactions being added to the pool
func (pm *PeerManager) processTransactionAdded(txn *types.Transaction) {
	pm.Mutex.Lock()
	defer pm.Mutex.Unlock()

	// Add transaction to pool
	pm.TransactionPool = append(pm.TransactionPool, txn)

	// Broadcast to peers
	if pm.Broadcaster != nil {
		// No conversion needed, use types.Transaction directly
		pm.Broadcaster.BroadcastTransaction(interfaces.TransactionEvent{
			Transaction: txn,
		})
	}
}

func (pm *PeerManager) StartListening() {
	go func() {
		for {
			select {
			//case event := <-pm.BlockAddedSubscription:
			//	pm.processBlockAdded(event.Block)

			case event := <-pm.TransactionAddedSubscription:
				pm.processTransactionAdded(event.Transaction)
			default:
				// Add a default case to prevent busy waiting
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()
}

func (pm *PeerManager) UpdatePeers(peersList map[string]bool) {
	pm.peerManager.UpdatePeers(peersList)
	log.Println("Updated peers list:", peersList)
}

func (pm *PeerManager) UpdatePeer(peer types.Peer) {
	pm.peerManager.UpdatePeers(map[string]bool{peer.ID: peer.Status})
}

// HTTP-based sync removed - now handled by libp2p P2P consensus manager

// HTTP-based peer list sending removed - now handled by libp2p P2P manager
func (pm *PeerManager) SendPeersList(address string) {
	// This function is deprecated - peer list sharing is now handled by libp2p DHT
	log.Println("SendPeersList is deprecated - peer discovery handled by libp2p DHT")
}

// HTTP-based status check removed - now handled by libp2p P2P manager

func (pm *PeerManager) BroadcastPeerList() {
	peers := pm.peerManager.GetPeers()
	for id, peer := range peers {
		if id != pm.peerManager.Address && peer.Status {
			pm.SendPeersList(peer.Address)
			time.Sleep(constants.PEER_BROADCAST_PAUSE_TIME * time.Second)
		}
	}
}

// HTTP-based peer dialing removed - peer management now handled by libp2p P2P manager
func (pm *PeerManager) DialAndUpdatePeers() {
	// This function is deprecated - peer discovery and status management
	// is now handled by the libp2p P2P manager and DHT
	log.Println("DialAndUpdatePeers is deprecated - peer management handled by libp2p")
}

// HTTP-based transaction sending removed - now handled by libp2p P2P manager
func (pm *PeerManager) SendTxnToThePeer(address string, txn *types.Transaction) {
	// This function is deprecated - transaction broadcasting is now handled by libp2p pubsub
	log.Println("SendTxnToThePeer is deprecated - transaction broadcasting handled by libp2p pubsub")
}

// HTTP-based block fetching removed - now handled by libp2p chain sync protocol

// HTTP-based chain header fetching removed - now handled by libp2p chain sync protocol

type PeerBlockHeader struct {
	BlockNumber uint64 `json:"block_number"`
	Hash        string `json:"hash"`
	PrevHash    string `json:"prev_hash"`
	Timestamp   int64  `json:"timestamp"`
}

// HTTP-based blocks range fetching removed - now handled by libp2p chain sync protocol

// HTTP-based full chain fetching removed - now handled by libp2p chain sync protocol

// HTTP-based direct chain fetching removed - now handled by libp2p chain sync protocol

func (pm *PeerManager) VerifyLastNBlocks(blocks []*Block) bool {
	if blocks[0].Number != 0 && !isValidBlockHash(blocks[0].Hash(), constants.MINING_DIFFICULTY) {
		log.Println("Chain verification failed for block", blocks[0].Number)
		return false
	}

	for i := 1; i < len(blocks); i++ {
		if blocks[i-1].Hash() != blocks[i].PreviousHash {
			log.Println("Failed to verify prevHash for block number", blocks[i].Number)
			return false
		}

		if !isValidBlockHash(blocks[i].Hash(), constants.MINING_DIFFICULTY) {
			log.Println("Chain verification failed for block", blocks[i].Number)
			return false
		}
	}

	return true
}

func isValidBlockHash(hash string, difficulty int) bool {
	return hash[2:2+difficulty] == strings.Repeat("0", difficulty)
}

// HTTP-based request handling removed - peer management now handled by libp2p P2P manager
func (pm *PeerManager) HandleRequest(w http.ResponseWriter, req *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	if req.Method == http.MethodGet {
		// Return current peers list without HTTP-based fetching
		peersJson, _ := json.Marshal(pm.peerManager.GetPeers())
		w.Write(peersJson)
	} else {
		http.Error(w, "Invalid Method", http.StatusBadRequest)
	}
}

func (pm *PeerManager) AddPeer(p types.Peer) {
	pm.peerManager.AddPeer(p)
}

func (pm *PeerManager) RemovePeer(id string) {
	pm.peerManager.RemovePeer(id)
}

func (pm *PeerManager) GetPeers() map[string]types.Peer {
	return pm.peerManager.GetPeers()
}

func (pm *PeerManager) GetPeer(id string) (types.Peer, bool) {
	return pm.peerManager.GetPeer(id)
}

func (pm *PeerManager) String() string {
	bytes, _ := json.Marshal(pm.peerManager.GetPeers())
	return string(bytes)
}
