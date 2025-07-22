package p2p

import (
	"log"
	"sync"
)

// Global P2P consensus manager instance
var (
	p2pConsensusManager *P2PConsensusManager
	managerMutex        sync.Mutex
)

// InitP2PConsensusManager initializes the P2P consensus manager
func InitP2PConsensusManager(blockchain *BlockchainStruct, db *LevelDB, discoveryManager *DiscoveryManager) error {
	managerMutex.Lock()
	defer managerMutex.Unlock()

	// Check if manager already exists
	if p2pConsensusManager != nil {
		log.Printf("[%s] P2P consensus manager already initialized", blockchain.ChainID)
		return nil
	}

	// Create a new P2P consensus manager
	manager, err := NewP2PConsensusManager(blockchain, db, discoveryManager)
	if err != nil {
		return err
	}

	// Set the global instance
	p2pConsensusManager = manager

	return nil
}

// GetP2PConsensusManager returns the global P2P consensus manager instance
func GetP2PConsensusManager() *P2PConsensusManager {
	managerMutex.Lock()
	defer managerMutex.Unlock()
	return p2pConsensusManager
}

// StartP2PConsensus starts the P2P consensus process
func StartP2PConsensus() {
	managerMutex.Lock()
	manager := p2pConsensusManager
	managerMutex.Unlock()

	if manager != nil {
		go manager.Start()
	}
}

// StopP2PConsensus stops the P2P consensus process
func StopP2PConsensus() {
	managerMutex.Lock()
	manager := p2pConsensusManager
	p2pConsensusManager = nil
	managerMutex.Unlock()

	if manager != nil {
		if manager.cancel != nil {
			manager.cancel()
		}
	}
}