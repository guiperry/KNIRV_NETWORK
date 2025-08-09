# KNIRVROUTER Verifier Node - Blockchain Implementation in Go

KNIRVROUTER Verifier Node is a specialized blockchain application written in Go, designed to verify and record network traffic data. It features a Proof-of-Work consensus mechanism, peer-to-peer capabilities, wallet management, and an integrated TURN server for P2P communication.

## Features

-   **Core Blockchain:**
    -   Basic Proof-of-Work (PoW) consensus mechanism.
    -   Block creation and mining with configurable difficulty.
    -   Transaction creation and inclusion in blocks.
    -   Blockchain state persistence using LevelDB.
-   **Wallet System:**
    -   Wallet creation using ECDSA cryptography.
    -   Generation of public/private key pairs and addresses with a custom prefix.
    -   Basic transaction signing (verification implementation may be limited).
-   **Networking (Enhanced):**
    -   Structures for peer management and network communication.
    -   Broadcasting mechanisms for transactions and blocks.
    -   **NEW: Integrated TURN server** for facilitating P2P connections.
    -   **NEW: Federation with Root Chain** for submitting verified blocks.
-   **User Interfaces:**
    -   **Desktop GUI (Fyne):** Provides controls to manage a verifier node, generate wallets, start a wallet server, and control the TURN server. Includes an embedded terminal for verifier node/wallet output.
    -   **Command-Line Interface:** Enables starting different node types (verifier chain, wallet server) via subcommands and flags.
-   **Configuration:**
    -   Configurable mining difficulty and reward.
    -   Flexible database path configuration using standard OS application data directories, environment variables, or command-line flags.
    -   Environment variable support via `.env` files.

## System Requirements

-   Go 1.21 or higher
-   Linux, macOS, or Windows operating system
-   For Fyne Desktop GUI on Linux (Ubuntu/Debian): `sudo apt-get install libgl1-mesa-dev xorg-dev`

## Installation

1.  Clone the repository:
    ```bash
    git clone https://github.com/yourusername/KNIRVROUTER_GO_Verifyer.git # Replace with your actual repo URL
    cd KNIRVROUTER_GO_Verifyer
    ```

2.  Install dependencies:
    ```bash
    go mod tidy
    ```

## Building the Application

A build script is provided for convenience:

1.  Make the build script executable:
    ```bash
    chmod +x build.sh
    ```

2.  Run the build script:
    ```bash
    ./build.sh
    ```

This will create an executable binary named `KNIRVROUTER` in the project's root directory.

## Running KNIRVROUTER Verifier Node

The application uses subcommands to determine its operating mode.

### 1. Default Mode (Desktop GUI)

Simply run the binary without any arguments or use `go run`:

```bash
./KNIRVROUTER
```

or

```bash
go run .
```

This starts the Fyne-based desktop GUI, which acts as a manager for the verifier node components.

### 2. Verifier Blockchain Node (Command Line)

Starts the verifier blockchain node, listens for connections, mines, and manages consensus. Output appears in the terminal.

```bash
# Using compiled binary
./KNIRVROUTER chain --port=5000 --miners_address=<your_address> --dbpath=<path_to_db> --root_chain=<root_chain_address>

# Using go run
go run . chain --port=5000 --miners_address=<your_address> --dbpath=<path_to_db> --root_chain=<root_chain_address>
```

- `--port`: (Optional) Port for the verifier node (default: 5000).
- `--miners_address`: Required. Wallet address to receive mining rewards.
- `--dbpath`: (Optional) Path to the database directory. Defaults to a standard OS application data location.
- `--root_chain`: (Optional) Address of the root chain for federation.

### 3. Wallet Server (Command Line)

Starts a server providing wallet functionalities via an API.

```bash
# Using compiled binary
./KNIRVROUTER wallet --port=8080 --node_address=http://127.0.0.1:5000

# Using go run
go run . wallet --port=8080 --node_address=http://127.0.0.1:5000
```

- `--port`: (Optional) Port for the wallet server (default: 8080).
- `--node_address`: (Optional) URL of the verifier blockchain node to interact with (default: http://127.0.0.1:5000).

### Desktop GUI Interaction Summary

- **Start Verifier Blockchain**: Launches `./KNIRVROUTER chain [flags]` based on GUI settings. Output appears in the GUI's embedded terminal.
- **Start Wallet**: Launches `./KNIRVROUTER wallet [flags]` based on GUI settings. Output appears in the GUI's embedded terminal.
- **Generate Wallet**: Creates a new wallet and displays the address, private key, and public key in the GUI.
- **Start TURN Server**: Initializes and starts the integrated TURN server for P2P communication.
- **Stop TURN Server**: Stops the running TURN server.

Configuration

Configuration is managed through a hierarchy: Command-line Flags > Environment Variables > Default Values.

### Key Parameters & Defaults

| Parameter | Flag (--chain) | Flag (--wallet) | Environment Variable | Default Value | Description |
|-----------|---------------|-----------------|----------------------|---------------|-------------|
| Verifier Node Port | --port | | PORT | 5000 | Port for the verifier blockchain node. |
| Wallet Port | | --port | WALLET_PORT | 8080 | Port for the wallet server API. |
| Miner's Address | --miners_address | | MINERS_ADDRESS | KNIRVROUTER-... | Address receiving mining rewards. |
| Node Address | | --node_address | BLOCKCHAIN_NODE_ADDRESS | http://127.0.0.1:5000 | URL for wallet server to connect to verifier node. |
| Database Path | --dbpath | | BLOCKCHAIN_DB_PATH | Standard App Data Dir | Path to the LevelDB database directory. |
| Mining Difficulty | | | MINING_DIFFICULTY | 3 | PoW difficulty target. |
| Mining Reward | | | MINING_REWARD | 100 | Reward amount per block. |
| Root Chain Address | --root_chain | | ROOT_CHAIN_ADDRESS | http://root.knirvchain.com:5000 | Address of the root chain for federation. |
| TURN Port | | | TURN_PORT | 3478 | Port for the TURN server. |
| Consensus Pause | | | CONSENSUS_PAUSE_TIME | 60 | Pause time (seconds) in consensus logic. |

Note 1: When launching webgui via the command line, the --port flag is primary. The PORT env var might be read internally by webgui if the flag isn't set, but using the flag is recommended. When launched by the Fyne GUI, the port is set via the flag.

#Database Path Defaults

If --dbpath (for chain) or BLOCKCHAIN_DB_PATH (for webgui launched by Fyne GUI) are not provided, the application defaults to standard OS-specific application data directories:

Linux: ~/.config/KNIRVROUTER/data/
macOS: ~/Library/Application Support/KNIRVROUTER/data/
Windows: %APPDATA%\KNIRVROUTER\data\ (e.g., C:\Users\<User>\AppData\Roaming\KNIRVROUTER\data\)

Within this base directory:

-The root node (chain command) defaults to: <base_dir>/root/knirvdb
-A peer node (webgui command on port P) defaults to: <base_dir>/P/knirvdb (e.g., <base_dir>/5001/knirvdb)

#.env File


You can create a test.env file in the project root to set environment variables easily during development:


MINERS_ADDRESS=KNIRVROUTER-3dd025e8fec7eda7cdd012ddde9c8e978ee7fa33
PORT=5000 # Default for root node
MINING_DIFFICULTY=3
MINING_REWARD=100
# BLOCKCHAIN_DB_PATH=custom/path/db # Optional: Override default path
CONSENSUS_PAUSE_TIME=60
# WALLET_PORT=8080 # Optional
# BLOCKCHAIN_NODE_ADDRESS=http://127.0.0.1:5000 # Optional
## Project Structure

```
KNIRVROUTER_GO_Verifyer/
├── blockchain/         # Core blockchain logic (blocks, mining, PoW, DB interaction)
├── blockchainserver/   # HTTP server API for the verifier blockchain node
├── constants/          # System-wide constants and default path logic
├── gui/                # Desktop GUI (Fyne) implementation
├── interfaces/         # Go interfaces for component interaction (listeners, events)
├── network/            # P2P networking logic
├── starter/            # Handles command-line parsing and delegates startup
├── turnserver/         # TURN server implementation for P2P communication
├── types/              # Core data structures (Transaction, PeerManager, etc.)
├── utils/              # Utility functions and helpers
├── wallet/             # Wallet generation, signing logic (ECDSA)
├── walletserver/       # HTTP server API for wallet functions
├── build.sh            # Build script
├── main.go             # Application entry point (handles flags/subcommands)
├── test.env            # Example environment configuration file
└── README.md           # This file
```
## Architecture Overview

The application starts via main.go, which parses initial arguments to determine the mode:

1. **No args or unrecognized**: Starts the Fyne Desktop GUI (`gui.StartFyneGUI`).
2. **chain, wallet**: Calls `starter.StartCommandLine`.
   - `starter` further parses subcommand-specific flags.
   - The "chain" flag calls `starter.StartVerifierBlockchain` (runs verifier node logic).
   - The "wallet" flag calls `starter.StartWallet` (runs wallet server).

The Fyne GUI acts as a controller, launching chain and wallet commands as separate processes using `exec.Command`. The TURN server is integrated directly into the GUI application and runs in the same process.

## Known Issues & Limitations

- **Consensus**: The current consensus mechanism is basic and lacks robust validation, chain selection (longest-chain rule), and fork resolution.
- **Security**: Lacks comprehensive transaction signature verification, peer authentication, and encrypted network communication.
- **Federation**: The root chain submission is implemented as a "fire-and-forget" mechanism without confirmation or complex consensus rules.
- **TURN Server**: The TURN server uses a simple static authentication mechanism. In production, a more robust credential system should be implemented.
- **Performance**: Potential bottlenecks exist (e.g., related to mining locks, database operations).
- **Error Handling**: Chain repair and robust error recovery mechanisms are limited.
- **Testing**: Test coverage (unit, integration, network simulation) needs expansion.
## Troubleshooting

- **Dependencies**: Run `go mod tidy` to ensure all dependencies are properly installed.
- **Build Errors**: Ensure you have Go 1.21+ installed correctly.
- **GUI Won't Start (Linux)**: Install Fyne prerequisites: `sudo apt-get install libgl1-mesa-dev xorg-dev`.
- **TURN Server Errors**: Check that the specified TURN port is not already in use by another application.
- **Database Errors**: Check permissions for the database directory. Ensure the directory exists or can be created by the application. Check logs for specific LevelDB errors.
- **Port Conflicts**: Ensure the required ports (5000, 8080, 3478, etc.) are not already in use by other applications.
- **Root Chain Connection Issues**: Verify that the root chain address is correct and that the root chain is running and accessible.
License
This project is licensed under the MIT License.

