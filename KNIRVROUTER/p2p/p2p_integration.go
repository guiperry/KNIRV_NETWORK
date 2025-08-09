package p2p

import (
	"log"

	"KNIRVROUTER_GO_Verifyer/blockchain"
	"KNIRVROUTER_GO_Verifyer/constants"
	"KNIRVROUTER_GO_Verifyer/types"
)

// InitializeP2PConsensus initializes the P2P consensus manager with the blockchain
// This function should be called after the blockchain is initialized
func InitializeP2PConsensus(blockchain *blockchain.BlockchainStruct, db *blockchain.LevelDB, discoveryManager *DiscoveryManager) error {
	log.Printf("[%s] Initializing P2P consensus manager", blockchain.ChainID)

	// Initialize the P2P consensus manager
	err := InitP2PConsensusManager(blockchain, db, discoveryManager)
	if err != nil {
		log.Printf("[%s] Failed to initialize P2P consensus manager: %v", blockchain.ChainID, err)
		return err
	}

	// Start the P2P consensus process
	StartP2PConsensus()

	log.Printf("[%s] P2P consensus manager initialized and started successfully", blockchain.ChainID)
	return nil
}

// ShutdownP2PConsensus gracefully shuts down the P2P consensus manager
func ShutdownP2PConsensus() {
	log.Println("Shutting down P2P consensus manager...")

	// Get the P2P consensus manager
	manager := GetP2PConsensusManager()
	if manager == nil {
		log.Println("P2P consensus manager not initialized, nothing to shut down")
		return
	}

	// Stop the P2P consensus process
	StopP2PConsensus()

	log.Println("P2P consensus manager shut down successfully")
}

// BroadcastBlockViaP2P broadcasts a block to the network using the P2P consensus manager
func BroadcastBlockViaP2P(block *blockchain.Block) error {
	manager := GetP2PConsensusManager()
	if manager == nil {
		return constants.ErrP2PConsensusManagerNotInitialized
	}

	return manager.BroadcastBlock(block)
}

// BroadcastTransactionViaP2P broadcasts a transaction to the network using the P2P consensus manager
func BroadcastTransactionViaP2P(transaction *types.Transaction) error {
	manager := GetP2PConsensusManager()
	if manager == nil {
		return constants.ErrP2PConsensusManagerNotInitialized
	}

	return manager.BroadcastTransaction(transaction)
}
