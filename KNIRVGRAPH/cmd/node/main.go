package main

import (
	"blockchain-app/internal/app"
	"context"
	"flag"
	"log"
)

func main() {
	var (
		homeDir     = flag.String("home", "./data", "Home directory for GraphChain data")
		rpcPort     = flag.Int("rpc-port", 8080, "RPC server port")
		testnetMode = flag.Bool("testnet", false, "Run in testnet mode")
		inMemory    = flag.Bool("memory", false, "Use in-memory storage")
		prePopulate = flag.Bool("populate", false, "Pre-populate test data")
		maxNodes    = flag.Int("max-nodes", 1000, "Maximum nodes in testnet mode")
	)
	flag.Parse()

	ctx := context.Background()

	// Create testnet configuration if enabled
	var config *app.Config
	if *testnetMode {
		config = &app.Config{
			Testnet: app.TestnetConfig{
				Enabled:     true,
				InMemory:    *inMemory,
				PrePopulate: *prePopulate,
				MaxNodes:    *maxNodes,
				ChainID:     "knirvgraph-testnet-1",
				Port:        *rpcPort,
				LocalMode:   true,
			},
		}
		log.Println("Starting KNIRVGRAPH in testnet mode")
	}

	// Create and start GraphChain application
	app, err := app.NewAppWithConfig(*homeDir, *rpcPort, config)
	if err != nil {
		log.Fatalf("Failed to create app: %v", err)
	}

	if err := app.Start(ctx); err != nil {
		log.Fatalf("Failed to start app: %v", err)
	}
}
