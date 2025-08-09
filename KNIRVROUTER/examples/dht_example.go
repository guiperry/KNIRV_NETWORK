package examples

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"KNIRVROUTER_GO_Verifyer/p2p"
	"KNIRVROUTER_GO_Verifyer/types"
)

func dht_example_main() {
	// Define bootstrap peers (these would be well-known nodes in a real network)
	bootstrapPeers := []string{
		"/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
		"/ip4/104.236.179.241/tcp/4001/p2p/QmSoLPppuBtQSGwKDZT2M73ULpjvfd3aZ6ha4oFGL1KrGM",
	}

	// Create a P2P server with DHT enabled
	server := p2p.NewServerWithDHT(
		"0.0.0.0:9000",       // Listen address
		"example-node-1",     // Node ID
		"example-chain-1",    // Chain ID
		bootstrapPeers,       // Bootstrap peers
		handleNewTransaction, // Transaction callback
		handleNewBlock,       // Block callback
		true,                 // Enable DHT
	)

	// Start the server
	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start P2P server: %v", err)
	}
	defer server.Stop()

	// Wait for the DHT to initialize
	time.Sleep(5 * time.Second)

	// Example: Resolve a knirv:// URI
	uri := "knirv://example-chain-id.chain/"
	log.Printf("Resolving URI: %s", uri)

	addresses, err := server.ResolveKnirvURI(uri)
	if err != nil {
		log.Printf("Failed to resolve URI: %v", err)
	} else {
		log.Printf("Found %d peers for URI %s:", len(addresses), uri)
		for i, addr := range addresses {
			log.Printf("  %d. %s", i+1, addr)
		}
	}

	// Keep the program running until interrupted
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
}

// handleNewTransaction is called when a new transaction is received
func handleNewTransaction(tx *types.Transaction) {
	fmt.Printf("Received new transaction: %s\n", tx.Hash())
}

// handleNewBlock is called when a new block is received
func handleNewBlock(block types.Block) {
	fmt.Printf("Received new block: %s\n", block.Hash())
}
