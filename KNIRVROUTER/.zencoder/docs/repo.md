# KNIRVCHAIN Verifier Node Information

## Summary
KNIRVCHAIN Verifier Node is a blockchain application written in Go, designed to verify and record network traffic data. It features a Proof-of-Work consensus mechanism, peer-to-peer capabilities using DHT (Distributed Hash Table), wallet management, and an integrated TURN server for P2P communication.

## Structure
- **blockchain/**: Core blockchain logic (blocks, mining, PoW, DB interaction)
- **blockchainserver/**: HTTP server API for the verifier blockchain node
- **constants/**: System-wide constants and default path logic
- **gui/**: Desktop GUI (Fyne) implementation
- **interfaces/**: Go interfaces for component interaction
- **p2p/**: P2P networking logic using libp2p and DHT
- **starter/**: Handles command-line parsing and delegates startup
- **transaction_turnserver/**: TURN server implementation for P2P communication
- **types/**: Core data structures (Transaction, Block, Peer)
- **utils/**: Utility functions and helpers
- **wallet/**: Wallet generation, signing logic (ECDSA)
- **walletserver/**: HTTP server API for wallet functions

## Language & Runtime
**Language**: Go
**Version**: Go 1.23 (toolchain go1.23.1)
**Build System**: Go modules
**Package Manager**: Go modules

## Dependencies
**Main Dependencies**:
- fyne.io/fyne/v2 v2.5.5 (GUI framework)
- github.com/libp2p/go-libp2p v0.39.1 (P2P networking)
- github.com/libp2p/go-libp2p-kad-dht v0.25.2 (DHT implementation)
- github.com/pion/turn/v2 v2.1.6 (TURN server)
- github.com/syndtr/goleveldb v1.0.0 (Database)
- github.com/gorilla/mux v1.8.1 (HTTP routing)
- github.com/joho/godotenv v1.5.1 (Environment configuration)

## Build & Installation
```bash
# Install dependencies
go mod tidy

# Build the application
./build.sh
# or
go build -o KNIRVCHAIN
```

## Usage & Operations
```bash
# Start the desktop GUI (default)
./KNIRVCHAIN

# Start the verifier blockchain node
./KNIRVCHAIN chain --port=5000 --miners_address=<your_address> --dbpath=<path_to_db> --root_chain=<root_chain_address>

# Start the wallet server
./KNIRVCHAIN wallet --port=8080 --node_address=http://127.0.0.1:5000
```

## Configuration
The application uses a hierarchical configuration system:
1. Command-line flags
2. Environment variables
3. Default values

Key configuration parameters:
- **Verifier Node Port**: Default 5000, configurable via `--port` flag or `PORT` env var
- **Wallet Port**: Default 8080, configurable via `--port` flag (wallet mode) or `WALLET_PORT` env var
- **Database Path**: Configurable via `--dbpath` flag or `BLOCKCHAIN_DB_PATH` env var
- **Mining Difficulty**: Default 3, configurable via `MINING_DIFFICULTY` env var
- **Mining Reward**: Default 100, configurable via `MINING_REWARD` env var
- **TURN Port**: Default 3478, configurable via `TURN_PORT` env var

## Testing
**Framework**: Go's built-in testing package
**Test Location**: `transaction_turnserver/server_test.go`
**Run Command**:
```bash
go test ./...
```

## Architecture
The application has three main operating modes:
1. **Desktop GUI**: Starts a Fyne-based GUI that can launch and manage other components
2. **Blockchain Node**: Runs the verifier blockchain with P2P networking and consensus
3. **Wallet Server**: Provides wallet functionality via an HTTP API

The P2P networking uses libp2p with a Kademlia DHT for peer discovery and a TURN server to facilitate connections through NATs.