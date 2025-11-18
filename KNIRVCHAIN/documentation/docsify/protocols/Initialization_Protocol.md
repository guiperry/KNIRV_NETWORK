Deep Analysis of Blockchain Initialization in main.go

Your main.go orchestrates the startup of different node types (Root, Bootnode, Peer, Client) and handles configuration loading, installation, and component initialization. The blockchain (*BlockchainStruct) is a central component initialized via the NewBlockchain function.

Here are the primary paths leading to blockchain initialization:

Initial Configuration Loading:

The process starts by determining the nodeRole based on command-line flags (-root, -bootnode, -dev, -client-only, -role).
config.LoadConfigurationViper is called to load the configuration (cfg) based on the determined role and an optional --config path. This cfg is the main configuration instance that will be used for the primary node process.
If nodeRole is config.Root, loadRootNodeParameters is called to potentially override some config values from env.local.
fetchAndStorePublicIPInfo is called, which might update cfg.PublicIPInfo and potentially save it back to the config file (for non-Root roles) or env.local (for Root).
Installation Process:

If cfg.InstallComplete is false and --skip-install is not set, the Install function is called.
Install guides the user (or uses defaults/non-interactive mode) to set up essential configuration like ChainID, MinersAddress, BlockchainDatabasePath, SearchableDatabasePath, etc.
Install saves the updated configuration to a role-specific file (e.g., ~/.config/KNIRVCHAIN/dev_data/config.json for a Peer).
The cfg variable in main is updated with the configuration returned by Install. This is crucial because the rest of main uses this potentially updated cfg.
Mode-Based Node Configuration (cfg.IsPeer, isNetworkMode):

After configuration loading/installation, main determines the operating mode:
Peer Mode (cfg.IsPeer is true): A devCfg is created based on the main cfg. This devCfg is explicitly prepared with dev-specific logic (e.g., ensuring generic ports are set, determining the SearchableDatabasePath). The finalPeerConfig variable holds this prepared config.
Network Mode (isNetworkMode is true): A mainNodeConfig (based on cfg) and a separate reflectionNodeConfig are prepared. The ReflectionDatabasePath is determined for the reflection node.
Single Node Mode (default): The main cfg is used directly.
GUI Node Identification (guiNodeConfig):

The code then determines which of the prepared node configurations (main node, dev node, or reflection node) will run the GUI, if --gui was enabled.
guiNodeConfig is assigned to point to the configuration (&mainNodeConfig, &devCfg, or reflectionNodeConfig) that has UseGUI set to true. If --gui was not set or forced off for the role, guiNodeConfig remains nil.
GUI Component Pre-initialization (if guiNodeConfig != nil block):

If guiNodeConfig is not nil, a block of code executes to pre-initialize components specifically for the node that will run the GUI.
This block initializes:
guiDB using the database path from guiNodeConfig (BlockchainDatabasePath or ReflectionDatabasePath).
guiDiscoveryMgr using parameters from guiNodeConfig.
guiBC by calling NewBlockchain. This call uses trueGenesisBlock, guiNodeConfig.ChainID, guiNodeConfig.MinersAddress, the pre-initialized guiDB, guiNodeConfig.SearchableDatabasePath, and guiNodeConfig.Chromem.CerebrasConfig. This is the first point where NewBlockchain is called and where the ChromemDB path/config from the specific GUI node's config is available.
Node Startup (startNode or startNodeWithComponents):

Based on the mode (cfg.IsPeer, isNetworkMode), either startNode or startNodeWithComponents is called, typically within a goroutine managed by the wg.
startNode initializes its own LevelDB, DiscoveryManager, and calls NewBlockchain using the cfg passed to it. This is another point where NewBlockchain is called.
startNodeWithComponents is used when the GUI is enabled. It takes the pre-initialized guiDB, guiDiscoveryMgr, and guiBC as arguments. It then initializes the P2PConsensusManager and other server components using these pre-initialized components and the cfg passed to it (which is the same config as guiNodeConfig).
GUI Initialization (InitializeGUI):

If guiNodeConfig is not nil, the main goroutine waits for the P2P manager to be ready, then calls InitializeGUI.
InitializeGUI is passed the pre-initialized guiBC, guiDB, guiDiscoveryMgr, the P2P manager, guiNodeConfig, etc.
The ChromemManager is needed by the API handlers registered by InitializeGUI (specifically RegisterTransactionSearchHandlers).