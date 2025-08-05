

---

**Source**: KNIRVROOT/docs/Troubleshooting/completedFixes/peernode_log.md

gperry@cloud-eq:~/Documents/GitHub/cloud-equities/KNIRVCHAIN_GO_ROOT_MCP_PROTO$ go run . -dev
2025/05/23 17:28:58 [INFO] Block.Hash: Block #0 being converted to proto. Timestamp: 1678886400, Nonce: 0, NumTx: 0
2025/05/23 17:28:58 [INFO] ProtoBlock fields for block #0:
2025/05/23 17:28:58 [INFO] - BlockNumber: 0
2025/05/23 17:28:58 [INFO] - PrevHash: 
2025/05/23 17:28:58 [INFO] - Nonce: 0
2025/05/23 17:28:58 [INFO] - Timestamp: 2023-03-15 13:20:00 +0000 UTC
2025/05/23 17:28:58 [INFO] - Transactions count: 0
2025/05/23 17:28:58 [INFO] - ProposerAddress: KNIRVCHAINb53c1e30b8a578c091dd40612bfd1433991b4e09
2025/05/23 17:28:58 [INFO] Initialized deterministic Genesis Block. Hash: c3a7b1ecbfa373db8a37da060e1f2f8927ed5d09d2e4b1bc11d53797d0bd4d3a
2025/05/23 17:28:58 ***********STARTING KNIRVCHAIN***********
2025/05/23 17:28:58 VERSION: dev, OS: linux, Arch: amd64
2025/05/23 17:28:58 LOGFILE: KNIRVCHAIN.log
2025/05/23 17:28:58 Determined node role: Peer
2025/05/23 17:28:58 Using role-specific data directory: /home/gperry/.config/KNIRVCHAIN/dev_data for role Peer
2025/05/23 17:28:58 Viper: Role-specific config file 'dev_config.json' not found. Using defaults.
2025/05/23 17:28:58 Applied default settings for role Peer
2025/05/23 17:28:58 Viper: Applied role-specific defaults from settings matrix for role: Peer
2025/05/23 17:28:58 Using role-specific data directory: /home/gperry/.config/KNIRVCHAIN/dev_data for role Peer
2025/05/23 17:28:58 Viper: Resolved BlockchainDatabasePath for role Peer to: /home/gperry/.config/KNIRVCHAIN/dev_data/dev_KNIRVCHAIN.db
2025/05/23 17:28:58 Using role-specific data directory: /home/gperry/.config/KNIRVCHAIN/dev_data for role Peer
2025/05/23 17:28:58 Viper: Resolved BlockchainDatabasePath for role Peer to: /home/gperry/.config/KNIRVCHAIN/dev_data/local_KNIRVCHAIN.db
2025/05/23 17:28:58 Attempting to fetch and store public IP information...
2025/05/23 17:29:03 Successfully parsed IPInfo response: IP=172.58.132.90, Country=US, ASN=
2025/05/23 17:29:03 Using role-specific data directory: /home/gperry/.config/KNIRVCHAIN/dev_data for role Peer
2025/05/23 17:29:03 Saving minimal configuration to: /home/gperry/.config/KNIRVCHAIN/dev_data/Peer_config.json
2025/05/23 17:29:03 Saved minimal config to role-specific data dir: /home/gperry/.config/KNIRVCHAIN/dev_data/Peer_config.json
2025/05/23 17:29:03 Successfully stored public IP information in config for role Peer.
2025/05/23 17:29:03 No config file found for Peer role. Proceeding with default config (installer will run if InstallComplete is false).
2025/05/23 17:29:03 Using role-specific data directory: /home/gperry/.config/KNIRVCHAIN/dev_data for role Peer
2025/05/23 17:29:03 Wallet path for role Peer: /home/gperry/.config/KNIRVCHAIN/dev_data/wallet.dat
2025/05/23 17:29:03 WARNING: No wallet file found and MinersAddress not configured.
2025/05/23 17:29:03 This node requires reinstallation. Forcing installer.
2025/05/23 17:29:03 Final BlockchainDatabasePath after Viper and flag processing: /home/gperry/.config/KNIRVCHAIN/dev_data/dev_KNIRVCHAIN.db
2025/05/23 17:29:03 Final BlockchainDatabasePath: /home/gperry/.config/KNIRVCHAIN/dev_data/dev_KNIRVCHAIN.db
2025/05/23 17:29:03 Installation not complete - running installer...

=== Node Role Selection ===
Please select the role for this node:
1. Root - A root node that initializes a new chain
2. Bootnode - A bootnode that helps with dev discovery
3. Peer - A dev node that participates in the network
4. Client - A client-only node with reduced functionality
Enter your choice [1-4] (default: 3): 
2025/05/23 17:29:09 Using role-specific data directory: /home/gperry/.config/KNIRVCHAIN/dev_data for role Peer
2025/05/23 17:29:09 Config path for role Peer: /home/gperry/.config/KNIRVCHAIN/dev_data/config.json
2025/05/23 17:29:09 Checking config path: /home/gperry/.config/KNIRVCHAIN/dev_data/config.json for role Peer
2025/05/23 17:29:09 No config file found for role Peer. Returning default config to trigger installer.
=== KNIRVCHAIN Node Installation ===
Installing as a Peer node

This installer will:
1. Connect to the KNIRVCHAIN Bootnode
2. Generate a unique chain URI for this node
4. Generate dev wallet
5. Detect host operating system
6. Register URI handler for agent:// protocol
7. Find next available ports for node
8. Update the application configuration
9. Start the node

Enter the KNIRVCHAIN Root endpoint [default: http://localhost:9999]: 

You can request a specific URI (optional).
This will be used to request a specific URI from the server.
Leave empty for a randomly generated URI.
Enter desired URI: devs
Connecting to http://localhost:9999/uriGenerator...
Requesting specific URI: devs
2025/05/23 17:29:25 Received HTTP status code: 201
2025/05/23 17:29:25 Raw response body: {"txn_hash":"0x15df47d20157b1f34a4e21d9aec650307d20605276e9303fb83e43f90239f310","uri":"agent://devs.chain/"}
Successfully generated Chain ID: devs (from URI: agent://devs.chain/)
Transaction Hash ID for URI: 0x15df47d20157b1f34a4e21d9aec650307d20605276e9303fb83e43f90239f310
Generating wallet for this node...
2025/05/23 17:29:25 Using role-specific data directory: /home/gperry/.config/KNIRVCHAIN/dev_data for role Peer
2025/05/23 17:29:25 Wallet path for role Peer: /home/gperry/.config/KNIRVCHAIN/dev_data/wallet.dat
2025/05/23 17:29:25 Wallet saved successfully to /home/gperry/.config/KNIRVCHAIN/dev_data/wallet.dat
Generated Wallet Address: KNIRVCHAINe95dbc1bd1e061adbf0d8adcfc2a57a00902b0f5
2025/05/23 17:29:25 Using role-specific data directory: /home/gperry/.config/KNIRVCHAIN/dev_data for role Peer
2025/05/23 17:29:25 Wallet path for role Peer: /home/gperry/.config/KNIRVCHAIN/dev_data/wallet.dat
IMPORTANT: Securely back up the wallet file: /home/gperry/.config/KNIRVCHAIN/dev_data/wallet.dat

Configure ports for this dev node:
Enter Peer HTTP Port [default: 1000]: 
Enter Peer P2P (Libp2p) Port [default: 1000]: 1001
Skipping registry registration (only required for bootnodes)
Detected operating system: linux
URI scheme 'agent://' definition files created successfully on Linux.
Updating MIME database...
MIME database updated successfully.
Updating desktop database...
Desktop database updated successfully.
URI handlers registered successfully.
Setting up system service...
Configuring as a Peer service...
2025/05/23 17:29:59 Using role-specific data directory: /home/gperry/.config/KNIRVCHAIN/dev_data for role Peer

=== Manual Service Installation Instructions ===
A service file template has been created at: /home/gperry/.config/KNIRVCHAIN/dev_data/services/KNIRVCHAIN-dev.service

To install the service manually, run the following commands as root:
  sudo cp /home/gperry/.config/KNIRVCHAIN/dev_data/services/KNIRVCHAIN-dev.service /etc/systemd/system/
  sudo systemctl daemon-reload
  sudo systemctl enable KNIRVCHAIN-dev
  sudo systemctl start KNIRVCHAIN-dev

To check the service status:
  sudo systemctl status KNIRVCHAIN-dev

2025/05/23 17:29:59 Warning: Failed to set up system service: must run as root to install system service
You can manually configure the service later.
2025/05/23 17:29:59 Using role-specific data directory: /home/gperry/.config/KNIRVCHAIN/dev_data for role Peer
2025/05/23 17:29:59 Saving configuration for Peer node with ChainID: devs, InstallComplete: true
2025/05/23 17:29:59 Installer: Saving configuration for Peer role
2025/05/23 17:29:59 Using role-specific data directory: /home/gperry/.config/KNIRVCHAIN/dev_data for role Peer
2025/05/23 17:29:59 Saving minimal configuration to: /home/gperry/.config/KNIRVCHAIN/dev_data/Peer_config.json
2025/05/23 17:29:59 Saved minimal config to role-specific data dir: /home/gperry/.config/KNIRVCHAIN/dev_data/Peer_config.json
2025/05/23 17:29:59 Using role-specific data directory: /home/gperry/.config/KNIRVCHAIN/dev_data for role Peer
Configuration updated and saved to /home/gperry/.config/KNIRVCHAIN/dev_data/dev_config.json

=== Installation Complete ===
Your KNIRVCHAIN Peer Node is now configured with a unique chain URI.
This node will participate in the network and process transactions.
Finalizing installation and launching KNIRVCHAIN Node...
2025/05/23 17:29:59 Saving final configuration with InstallComplete=true for Peer role
2025/05/23 17:29:59 Using role-specific data directory: /home/gperry/.config/KNIRVCHAIN/dev_data for role Peer
2025/05/23 17:29:59 Saving minimal configuration to: /home/gperry/.config/KNIRVCHAIN/dev_data/Peer_config.json
2025/05/23 17:29:59 Saved minimal config to role-specific data dir: /home/gperry/.config/KNIRVCHAIN/dev_data/Peer_config.json
Installation complete. Attempting to restart application...
2025/05/23 17:30:00 Restart will use args: [-dev --skip-install]
2025/05/23 17:30:00 Attempting restart using executable path: /tmp/go-build461153484/b001/exe/KNIRVCHAIN
2025/05/23 17:30:00 Successfully restarted application using executable path
gperry@cloud-eq:~/Documents/GitHub/cloud-equities/KNIRVCHAIN_GO_ROOT_MCP_PROTO$ 2025/05/23 17:30:00 [INFO] Block.Hash: Block #0 being converted to proto. Timestamp: 1678886400, Nonce: 0, NumTx: 0
2025/05/23 17:30:00 [INFO] ProtoBlock fields for block #0:
2025/05/23 17:30:00 [INFO] - BlockNumber: 0
2025/05/23 17:30:00 [INFO] - PrevHash: 
2025/05/23 17:30:00 [INFO] - Nonce: 0
2025/05/23 17:30:00 [INFO] - Timestamp: 2023-03-15 13:20:00 +0000 UTC
2025/05/23 17:30:00 [INFO] - Transactions count: 0
2025/05/23 17:30:00 [INFO] - ProposerAddress: KNIRVCHAINb53c1e30b8a578c091dd40612bfd1433991b4e09
2025/05/23 17:30:00 [INFO] Initialized deterministic Genesis Block. Hash: c3a7b1ecbfa373db8a37da060e1f2f8927ed5d09d2e4b1bc11d53797d0bd4d3a
2025/05/23 17:30:00 ***********STARTING KNIRVCHAIN***********
2025/05/23 17:30:00 VERSION: dev, OS: linux, Arch: amd64
2025/05/23 17:30:00 LOGFILE: KNIRVCHAIN.log
2025/05/23 17:30:00 Determined node role: Peer
2025/05/23 17:30:00 Using role-specific data directory: /home/gperry/.config/KNIRVCHAIN/dev_data for role Peer
2025/05/23 17:30:00 Viper: Role-specific config file 'dev_config.json' not found. Using defaults.
2025/05/23 17:30:00 Applied default settings for role Peer
2025/05/23 17:30:00 Viper: Applied role-specific defaults from settings matrix for role: Peer
2025/05/23 17:30:00 Using role-specific data directory: /home/gperry/.config/KNIRVCHAIN/dev_data for role Peer
2025/05/23 17:30:00 Viper: Resolved BlockchainDatabasePath for role Peer to: /home/gperry/.config/KNIRVCHAIN/dev_data/dev_KNIRVCHAIN.db
2025/05/23 17:30:00 Using role-specific data directory: /home/gperry/.config/KNIRVCHAIN/dev_data for role Peer
2025/05/23 17:30:00 Viper: Resolved BlockchainDatabasePath for role Peer to: /home/gperry/.config/KNIRVCHAIN/dev_data/local_KNIRVCHAIN.db
2025/05/23 17:30:00 Attempting to fetch and store public IP information...
2025/05/23 17:30:00 Successfully parsed IPInfo response: IP=172.58.132.90, Country=US, ASN=
2025/05/23 17:30:00 Using role-specific data directory: /home/gperry/.config/KNIRVCHAIN/dev_data for role Peer
2025/05/23 17:30:00 Saving minimal configuration to: /home/gperry/.config/KNIRVCHAIN/dev_data/Peer_config.json
2025/05/23 17:30:00 Saved minimal config to role-specific data dir: /home/gperry/.config/KNIRVCHAIN/dev_data/Peer_config.json
2025/05/23 17:30:00 Successfully stored public IP information in config for role Peer.
2025/05/23 17:30:00 No config file found for Peer role. Proceeding with default config (installer will run if InstallComplete is false).
2025/05/23 17:30:00 Using role-specific data directory: /home/gperry/.config/KNIRVCHAIN/dev_data for role Peer
2025/05/23 17:30:00 Wallet path for role Peer: /home/gperry/.config/KNIRVCHAIN/dev_data/wallet.dat
2025/05/23 17:30:00 Using role-specific data directory: /home/gperry/.config/KNIRVCHAIN/dev_data for role Peer
2025/05/23 17:30:00 Wallet path for role Peer: /home/gperry/.config/KNIRVCHAIN/dev_data/wallet.dat
2025/05/23 17:30:00 Loaded wallet with address 'KNIRVCHAINe95dbc1bd1e061adbf0d8adcfc2a57a00902b0f5', updating config
2025/05/23 17:30:00 Using role-specific data directory: /home/gperry/.config/KNIRVCHAIN/dev_data for role Peer
2025/05/23 17:30:00 Saving minimal configuration to: /home/gperry/.config/KNIRVCHAIN/dev_data/Peer_config.json
2025/05/23 17:30:00 Saved minimal config to role-specific data dir: /home/gperry/.config/KNIRVCHAIN/dev_data/Peer_config.json
2025/05/23 17:30:00 Final BlockchainDatabasePath after Viper and flag processing: /home/gperry/.config/KNIRVCHAIN/dev_data/dev_KNIRVCHAIN.db
2025/05/23 17:30:00 Final BlockchainDatabasePath: /home/gperry/.config/KNIRVCHAIN/dev_data/dev_KNIRVCHAIN.db
2025/05/23 17:30:00 Skipping installation due to --skip-install flag. Continuing with node initialization...
2025/05/23 17:30:00 Configuring Peer Reflection Mode...
2025/05/23 17:30:00 FATAL: Peer ports not set in config.json. Run installation or set manually.

---

<div class="footer-links">


© 2025 KNIRV Network
</div>
