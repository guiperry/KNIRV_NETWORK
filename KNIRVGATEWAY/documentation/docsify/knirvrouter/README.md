# KNIRVROUTER Verifier Node User Guide

## Overview

The KNIRVROUTER Verifier Node is a blockchain application that verifies and records network traffic data. It uses a Proof-of-Work consensus mechanism and includes a user-friendly GUI and command-line interface.

## Getting Started

### System Requirements

* Go 1.21 or higher
* Linux, macOS, or Windows
* For the Fyne GUI on Linux (Ubuntu/Debian): `sudo apt-get install libgl1-mesa-dev xorg-dev`

### Installation

1. Clone the repository: `git clone <repository_url>` (replace `<repository_url>` with the actual URL)
2. Navigate to the project directory: `cd KNIRVROUTER_GO_Verifyer`
3. Install dependencies: `go mod tidy`
4. Build the application: `chmod +x build.sh && ./build.sh` This creates the `KNIRVROUTER` executable.

## Running the KNIRVROUTER Node

You can run the KNIRVROUTER node using either the graphical user interface (GUI) or the command-line interface (CLI).

### Using the GUI

1. Run the executable: `./KNIRVROUTER`
2. The GUI provides options to:
    * Start the verifier blockchain node.
    * Start the wallet server.
    * Generate new wallets.
    * Manage the integrated TURN server for peer-to-peer communication.

### Using the CLI

The CLI uses subcommands: `chain` for the blockchain node and `wallet` for the wallet server.

#### Starting the Verifier Blockchain Node

```bash
./KNIRVROUTER chain --port=5000 --miners_address=<your_address> --dbpath=<path_to_db> --root_chain=<root_chain_address>
```

* `--port`: Port for the node (default: 5000).
* `--miners_address`: Your wallet address to receive mining rewards. **Required.**
* `--dbpath`: Path to the database (defaults to a system-specific location).
* `--root_chain`: Address of the root chain for federation (optional).

#### Starting the Wallet Server

```bash
./KNIRVROUTER wallet --port=8080 --node_address=http://127.0.0.1:5000
```

* `--port`: Port for the wallet server (default: 8080).
* `--node_address`: Address of the verifier blockchain node (default: `http://127.0.0.1:5000`).

## Configuration

KNIRVROUTER uses a hierarchical configuration system: command-line flags override environment variables, which override default values. You can set environment variables in a `.env` file in the project root directory. See the table below for key parameters and their defaults.

| Parameter             | CLI Flag (`chain`) | CLI Flag (`wallet`) | Environment Variable      | Default Value | Description                                      |
|------------------------|----------------------|----------------------|---------------------------|---------------|--------------------------------------------------|
| Verifier Node Port     | `--port`             |                      | `PORT`                     | 5000          | Port for the verifier blockchain node.           |
| Wallet Port            |                      | `--port`             | `WALLET_PORT`             | 8080          | Port for the wallet server API.                 |
| Miner's Address        | `--miners_address`   |                      | `MINERS_ADDRESS`           | KNIRVROUTER-... | Address receiving mining rewards.                |
| Node Address           |                      | `--node_address`      | `BLOCKCHAIN_NODE_ADDRESS` | `http://127.0.0.1:5000` | URL for wallet server to connect to verifier node. |
| Database Path         | `--dbpath`           |                      | `BLOCKCHAIN_DB_PATH`      | System Default | Path to the LevelDB database directory.         |
| Mining Difficulty     |                      |                      | `MINING_DIFFICULTY`       | 3             | PoW difficulty target.                           |
| Mining Reward          |                      |                      | `MINING_REWARD`           | 100           | Reward amount per block.                          |
| Root Chain Address    | `--root_chain`       |                      | `ROOT_CHAIN_ADDRESS`      |  (See Note)   | Address of the root chain for federation.       |
| TURN Server Port      |                      |                      | `TURN_PORT`               | 3478          | Port for the TURN server.                        |
| Consensus Pause Time  |                      |                      | `CONSENSUS_PAUSE_TIME`    | 60            | Pause time (seconds) in consensus logic.        |

**Note:** The default `ROOT_CHAIN_ADDRESS` is not explicitly defined in the provided README. You will need to provide this value.

**Database Path Defaults:** If not specified, the database will be stored in a standard OS-specific application data directory (e.g., `~/.config/KNIRVROUTER/data/` on Linux).

## Troubleshooting

* **Dependencies:** Run `go mod tidy` to resolve dependency issues.
* **Build Errors:** Ensure you have Go 1.21+ installed correctly.
* **GUI Issues (Linux):** Install Fyne prerequisites: `sudo apt-get install libgl1-mesa-dev xorg-dev`.
* **TURN Server Errors:** Check if the TURN port is already in use.
* **Database Errors:** Verify database directory permissions and existence. Check application logs for LevelDB errors.
* **Port Conflicts:** Ensure that the required ports are not in use.
* **Root Chain Connection Issues:** Verify the root chain address and ensure the root chain is running and accessible.

Improvements Needed:

* Add more detailed information about the hierarchical configuration system and how to set environment variables in a `.env` file.
* Provide more examples of how to use the CLI and GUI.
* Consider adding a section on security best practices for running the KNIRVROUTER node.
* Add more information about the LevelDB database and how to troubleshoot database errors.
* Consider adding a section on how to migrate the database to a new location or upgrade the database schema.
* Add more information about the TURN server and how to troubleshoot TURN server errors.
* Consider adding a section on how to integrate the KNIRVROUTER node with other blockchain networks or applications.

<div class="footer-links">
<a href="#/legal/CODE_OF_CONDUCT.md" class="footer-link">Contributor Covenant Code of Conduct</a> | <a href="#/legal/PRIVACY_POLICY.md" class="footer-link">PRIVACY_POLICY.md</a> | <a href="#/legal/TERMS_AND_CONDITIONS.md" class="footer-link">TERMS AND CONDITIONS</a>

© 2025 KNIRV Network
</div>
