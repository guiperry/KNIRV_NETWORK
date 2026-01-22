package examples

import (
	"log"
	"time"

	"KNIRVROUTER/internal/blockchain"
)

func verifyerNodeMain() {
	// Create a genesis block (block 0 with empty previous hash)
	genesisBlock := blockchain.NewBlock("", 0, 0)

	// Create a new blockchain instance with the genesis block
	bc := blockchain.NewBlockchain(*genesisBlock, "KNIRVROUTER-example-address")

	// In the real application, transactions would be created through the P2P network
	log.Printf("Current blockchain height: %d", len(bc.GetBlocks()))

	// Example: Show current state
	blocks := bc.GetBlocks()
	if len(blocks) > 0 {
		lastBlock := blocks[len(blocks)-1]
		log.Printf("Last block hash: %s", lastBlock.Hash())
		log.Printf("Last block has %d transactions", len(lastBlock.Txs))
	}

	// Wait a bit for demonstration
	time.Sleep(2 * time.Second)
	log.Println("Example completed")
}
