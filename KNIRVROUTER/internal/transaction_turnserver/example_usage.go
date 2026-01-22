package transaction_turnserver

import (
	"log"

	"KNIRVROUTER/internal/blockchain"
)

// ExampleUsage demonstrates how to use the new BlockchainAdapter with a real blockchain
func ExampleUsage() {
	// Create a genesis block and blockchain instance
	genesisBlock := blockchain.NewBlock("", 0, 0)
	bc := blockchain.NewBlockchain(*genesisBlock, "KNIRVROUTER-example-address")

	// Create a BlockchainAdapter with the real blockchain
	// This is the preferred way to create the adapter
	adapter := NewBlockchainAdapterWithBlockchain(bc, "KNIRVROUTER-miner-address")

	// Create and start the TURN server with the real adapter
	server, err := NewServer(3478, 3479, 8080, adapter)
	if err != nil {
		log.Fatalf("Failed to create TURN server: %v", err)
	}

	// Start the server
	server.Start()
	defer server.Stop()

	log.Println("TURN server is running with real blockchain integration")

	// The server will now submit real transactions to the blockchain
	// when TURN sessions are established and NRN tokens are minted
}

// ExampleLegacyUsage demonstrates the legacy way (deprecated)
func ExampleLegacyUsage() {
	// Create adapter using the legacy function-based approach (deprecated)
	adapter := NewBlockchainAdapter(
		func(from, to string, data []byte) error {
			// This is the old way - manually handling transaction creation
			// Use NewBlockchainAdapterWithBlockchain instead
			log.Printf("Legacy transaction: from=%s, to=%s, data=%s", from, to, string(data))
			return nil
		},
		"KNIRVROUTER-miner-address",
	)

	// Create server with legacy adapter
	server, err := NewServer(3478, 3479, 8080, adapter)
	if err != nil {
		log.Fatalf("Failed to create TURN server: %v", err)
	}

	server.Start()
	defer server.Stop()

	log.Println("TURN server is running with legacy adapter (deprecated)")
}
