// p2p/p2p_blockchain_connector.go
package p2p

import (
	"log"

	"KNIRVROUTER_GO_Verifyer/blockchain"
	"KNIRVROUTER_GO_Verifyer/types"
)

// ConnectBlockchainToP2P sets up the callbacks between the blockchain and P2P manager
func ConnectBlockchainToP2P(bc *blockchain.BlockchainStruct, p2pManager *P2PManager) {
	// Set up callbacks from blockchain to P2P manager
	bc.OnBlockMined = func(block *blockchain.Block) error {
		log.Printf("Block mined callback: Broadcasting block #%d", block.BlockNumber())
		return p2pManager.BroadcastBlock(block)
	}

	bc.OnTransactionAdded = func(tx *types.Transaction) error {
		log.Printf("Transaction added callback: Broadcasting transaction %s", tx.Hash())
		return p2pManager.BroadcastTransaction(tx)
	}

	// Set up callbacks from P2P manager to blockchain
	p2pManager.processReceivedBlock = func(block *blockchain.Block) {
		// Skip if we're actively mining
		if bc.IsActivelyMining() {
			log.Println("Skipping received block processing as we're actively mining")
			return
		}

		// Lock mining during block processing
		bc.Lock()
		defer bc.Unlock()

		// Verify the block
		if !block.VerifyHash() {
			log.Printf("Received invalid block #%d, ignoring", block.Number)
			return
		}

		// Get the current last block
		if len(bc.GetBlocks()) == 0 {
			log.Println("Blockchain is empty, cannot process received block")
			return
		}

		currentLastBlock := bc.GetBlocks()[len(bc.GetBlocks())-1]

		// If the block extends our current chain, add it
		if block.Number == currentLastBlock.Number+1 && block.PreviousHash == currentLastBlock.Hash() {
			log.Printf("Adding block #%d to our chain", block.Number)
			bc.AddBlock(p2pManager.db, block)
			return
		}

		// If the block is part of a potentially longer chain, trigger fork resolution
		if block.Number > currentLastBlock.Number {
			log.Printf("Received block #%d is ahead of our chain (at #%d), triggering fork resolution",
				block.Number, currentLastBlock.Number)

			// Request the full chain from the peer who sent this block
			// This will be handled by the fork resolution process
			p2pManager.requestChainFromPeers()
		}
	}

	p2pManager.processReceivedTransaction = func(transaction *types.Transaction) {
		// Verify and add the transaction to the pool
		if err := bc.AddTransactionToTransactionPool(transaction); err != nil {
			log.Printf("Failed to add received transaction %s: %v", transaction.Hash(), err)
		} else {
			log.Printf("Added transaction %s to pool", transaction.Hash())
		}
	}

	log.Println("Blockchain successfully connected to P2P manager")
}
