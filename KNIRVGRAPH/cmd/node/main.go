package main

import (
    "blockchain-app/internal/app"
    "context"
    "flag"
    "log"
)

func main() {
    var (
        homeDir = flag.String("home", "./data", "Home directory for GraphChain data")
        rpcPort = flag.Int("rpc-port", 8080, "RPC server port")
    )
    flag.Parse()
    
    ctx := context.Background()
    
    // Create and start GraphChain application
    app, err := app.NewApp(*homeDir, *rpcPort)
    if err != nil {
        log.Fatalf("Failed to create app: %v", err)
    }
    
    if err := app.Start(ctx); err != nil {
        log.Fatalf("Failed to start app: %v", err)
    }
}