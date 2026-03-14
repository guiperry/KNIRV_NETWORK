// /home/gperry/Documents/GitHub/KNIRVROUTER/internal/starter/starter.go

package starter

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	// "os/exec" // No longer needed in this file
	"os/signal"

	"strconv"
	"strings"
	"syscall"

	"github.com/joho/godotenv"

	"KNIRVROUTER/internal/blockchain"
	"KNIRVROUTER/internal/blockchainserver"
	"KNIRVROUTER/internal/connectivity"
	constants "KNIRVROUTER/internal/constants"
	"KNIRVROUTER/internal/embedded_knirvchain"
	"KNIRVROUTER/internal/p2p"
	"KNIRVROUTER/internal/transaction_turnserver"
	"KNIRVROUTER/internal/utils"
	"KNIRVROUTER/internal/walletserver"
	"KNIRVROUTER/internal/wasm_integration"

	"math/big"

	"github.com/libp2p/go-libp2p/core/peer"
)

// ... (Config struct, loadConfig, init remain the same) ...
type Config struct {
	Port                   uint64
	MinersAddress          string
	DatabasePath           string
	BlockchainName         string
	HexPrefix              string
	Success                bool
	Failed                 bool
	Pending                string
	MiningDifficulty       int
	MiningReward           int64
	CurrencyName           string
	Decimal                int
	BlockchainAddress      string
	BlockchainDbPath       string
	BlockchainKey          string
	AddressPrefix          string
	TxnVerificationSuccess string
	TxnVerificationFailure string
	BlockchainStatus       string
	PeerBroadcastPauseTime int
	PeerPingPauseTime      int
	TxnBroadcastPauseTime  int
	FetchLastNBlocks       int
	ConsensusPauseTime     int
	PeerAddresses          []string
	// NOTE: WalletPort is NOT defined here
	// Testnet-specific configuration
	TestnetMode         bool
	LocalNetworkMode    bool
	MockNRNMinting      bool
	TestnetChainID      string
	TestnetValidators   int
	TestnetInitialNRN   int64
	SimplifiedConsensus bool
	DisableXIONBridge   bool
}

// FaucetClientAdapter adapts the main package FaucetIntegration to the connectivity.FaucetClient interface
type FaucetClientAdapter struct {
	faucetIntegration FaucetIntegrationInterface
}

// FaucetIntegrationInterface defines the interface for faucet integration
type FaucetIntegrationInterface interface {
	RequestConnectivityReward(nodeID peer.ID, proofID string, score float64, amount *big.Int) (FaucetRequestInterface, error)
}

// FaucetRequestInterface defines the interface for faucet requests
type FaucetRequestInterface interface {
	GetRequestID() string
	GetNodeID() peer.ID
	GetAmount() *big.Int
	GetReason() string
	GetTimestamp() time.Time
	GetStatus() string
	GetTxHash() string
	GetErrorMessage() string
}

// NewFaucetClientAdapter creates a new adapter
func NewFaucetClientAdapter(faucetIntegration FaucetIntegrationInterface) *FaucetClientAdapter {
	return &FaucetClientAdapter{
		faucetIntegration: faucetIntegration,
	}
}

// RequestConnectivityReward implements the connectivity.FaucetClient interface
func (fca *FaucetClientAdapter) RequestConnectivityReward(nodeID peer.ID, proofID string, score float64, amount *big.Int) (*connectivity.FaucetRequest, error) {
	request, err := fca.faucetIntegration.RequestConnectivityReward(nodeID, proofID, score, amount)
	if err != nil {
		return nil, err
	}

	// Convert to connectivity.FaucetRequest
	return &connectivity.FaucetRequest{
		RequestID:    request.GetRequestID(),
		NodeID:       request.GetNodeID(),
		Amount:       request.GetAmount(),
		Reason:       request.GetReason(),
		Timestamp:    request.GetTimestamp(),
		Status:       request.GetStatus(),
		TxHash:       request.GetTxHash(),
		ErrorMessage: request.GetErrorMessage(),
	}, nil
}

// BlockchainAdapterWrapper wraps transaction_turnserver.BlockchainAdapter to implement connectivity.BlockchainAdapter
type BlockchainAdapterWrapper struct {
	adapter *transaction_turnserver.BlockchainAdapter
}

// NewBlockchainAdapterWrapper creates a new wrapper
func NewBlockchainAdapterWrapper(adapter *transaction_turnserver.BlockchainAdapter) *BlockchainAdapterWrapper {
	return &BlockchainAdapterWrapper{
		adapter: adapter,
	}
}

// SubmitNRNMintTx implements the connectivity.BlockchainAdapter interface
func (baw *BlockchainAdapterWrapper) SubmitNRNMintTx(recipient, amount, reason, proofID string) error {
	return baw.adapter.SubmitNRNMintTx(recipient, amount, reason, proofID)
}

func loadConfig() (*Config, error) {
	// Load .env file
	err := godotenv.Load("test.env")
	if err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
		log.Println("Will use default or environment values")
	}

	// Set default values for required environment variables if not set
	if os.Getenv("MINING_DIFFICULTY") == "" {
		os.Setenv("MINING_DIFFICULTY", "3")
	}
	if os.Getenv("MINING_REWARD") == "" {
		os.Setenv("MINING_REWARD", "100")
	}
	if os.Getenv("DECIMAL") == "" {
		os.Setenv("DECIMAL", "8")
	}
	if os.Getenv("CONSENSUS_PAUSE_TIME") == "" {
		os.Setenv("CONSENSUS_PAUSE_TIME", "60")
	}
	if os.Getenv("PORT") == "" {
		os.Setenv("PORT", "5000")
	}
	if os.Getenv("MINERS_ADDRESS") == "" {
		os.Setenv("MINERS_ADDRESS", "KNIRVROUTER-3dd025e8fec7eda7cdd012ddde9c8e978ee7fa33")
	}
	dbPath := os.Getenv("BLOCKCHAIN_DB_PATH")
	if dbPath == "" {
		// If Env Var not set, use the standard default for the root node
		dbPath = utils.GetDefaultRootDBPath() // Use helper function
		log.Printf("BLOCKCHAIN_DB_PATH not set, using default root path: %s", dbPath)
	} else {
		log.Printf("Using BLOCKCHAIN_DB_PATH from environment: %s", dbPath)
	}
	// NOTE: This dbPath is primarily used for the baseCfg defaults below.
	// The actual path used by 'chain' or 'webgui' is set later.

	// ... rest of loadConfig, parsing other env vars ...
	cfg := &Config{
		// ... other fields ...
		BlockchainDbPath: dbPath, // Store the determined path (might be overridden by flag)
	}

	// ... parse numeric values ...

	// DO NOT set constants.BLOCKCHAIN_DB_PATH here globally yet.

	var err1 error
	cfg.Port, err1 = strconv.ParseUint(os.Getenv("PORT"), 10, 64)
	if err1 != nil {
		return nil, fmt.Errorf("error parsing PORT: %w", err1)
	}
	cfg.MinersAddress = os.Getenv("MINERS_ADDRESS")
	cfg.BlockchainName = os.Getenv("BLOCKCHAIN_NAME")
	cfg.HexPrefix = os.Getenv("HEX_PREFIX")
	cfg.Success = os.Getenv("SUCCESS") == "true"
	cfg.Failed = os.Getenv("FAILED") == "true"
	cfg.Pending = os.Getenv("PENDING")
	cfg.MiningDifficulty, err1 = strconv.Atoi(os.Getenv("MINING_DIFFICULTY"))
	if err1 != nil {
		return nil, fmt.Errorf("error parsing MINING_DIFFICULTY: %w", err1)
	}
	cfg.MiningReward, err1 = strconv.ParseInt(os.Getenv("MINING_REWARD"), 10, 64)
	if err1 != nil {
		return nil, fmt.Errorf("error parsing MINING_REWARD: %w", err1)
	}
	cfg.CurrencyName = os.Getenv("CURRENCY_NAME")
	cfg.Decimal, err1 = strconv.Atoi(os.Getenv("DECIMAL"))
	if err1 != nil {
		return nil, fmt.Errorf("error parsing DECIMAL: %w", err1)
	}
	cfg.BlockchainAddress = os.Getenv("BLOCKCHAIN_ADDRESS") // This seems unused
	//cfg.BlockchainDbPath = os.Getenv("BLOCKCHAIN_DB_PATH")
	cfg.BlockchainKey = os.Getenv("BLOCKCHAIN_KEY") // This seems unused
	cfg.AddressPrefix = os.Getenv("ADDRESS_PREFIX")
	cfg.TxnVerificationSuccess = os.Getenv("TXN_VERIFICATION_SUCCESS")
	cfg.TxnVerificationFailure = os.Getenv("TXN_VERIFICATION_FAILURE")
	cfg.BlockchainStatus = os.Getenv("BLOCKCHAIN_STATUS")
	cfg.ConsensusPauseTime, err1 = strconv.Atoi(os.Getenv("CONSENSUS_PAUSE_TIME"))
	if err1 != nil {
		return nil, fmt.Errorf("error parsing CONSENSUS_PAUSE_TIME: %w", err1)
	}

	peerString := os.Getenv("PEER_ADDRESSES")
	if peerString != "" {
		cfg.PeerAddresses = strings.Split(peerString, ",")
	}

	// Load testnet configuration
	cfg.TestnetMode = os.Getenv("TESTNET_MODE") == "true"
	cfg.LocalNetworkMode = os.Getenv("LOCAL_NETWORK_MODE") == "true"
	cfg.MockNRNMinting = os.Getenv("MOCK_NRN_MINTING") == "true"
	cfg.TestnetChainID = os.Getenv("TESTNET_CHAIN_ID")
	if cfg.TestnetChainID == "" {
		cfg.TestnetChainID = "knirvrouter-testnet-1"
	}
	cfg.TestnetValidators, err1 = strconv.Atoi(os.Getenv("TESTNET_VALIDATORS"))
	if err1 != nil || cfg.TestnetValidators == 0 {
		cfg.TestnetValidators = 3
	}
	cfg.TestnetInitialNRN, err1 = strconv.ParseInt(os.Getenv("TESTNET_INITIAL_NRN"), 10, 64)
	if err1 != nil || cfg.TestnetInitialNRN == 0 {
		cfg.TestnetInitialNRN = 1000000000000
	}
	cfg.SimplifiedConsensus = os.Getenv("SIMPLIFIED_CONSENSUS") == "true"
	cfg.DisableXIONBridge = os.Getenv("DISABLE_XION_BRIDGE") == "true"

	// Ensure the database directory exists - Moved to where it's needed

	return cfg, nil
}

func init() {
	log.SetPrefix("KNIRVROUTER: ")
}

// StartWallet initializes and starts the wallet server
func StartWallet(port uint64, nodeAddress string) {
	// Ensure wallet port is available or find a new one if needed
	actualPort := port
	if !isPortAvailable(actualPort) {
		log.Printf("Wallet port %d in use, finding available port...", actualPort)
		actualPort = findAvailablePort(actualPort + 1) // Start searching from next port
		log.Printf("Using available wallet port %d", actualPort)
	}

	// Use the actual port found
	ws := walletserver.NewWalletServer(actualPort, nodeAddress)
	ws.Start() // This function likely blocks or starts its own goroutines
}

// --- REMOVED openBrowser function from starter.go ---
// func openBrowser(url string) { ... }

// StartCommandLine is the exported function that will be called from main.go
func StartCommandLine() {
	// Load configuration first to get defaults and env vars
	// This cfg is mostly for default flag values now
	baseCfg, err := loadConfig()
	if err != nil {
		log.Println("Error loading base configuration:", err)
		os.Exit(1)
	}

	// Log testnet configuration if enabled
	if baseCfg.TestnetMode {
		log.Println("🧪 TESTNET MODE ENABLED")
		log.Printf("   - Chain ID: %s", baseCfg.TestnetChainID)
		log.Printf("   - Validators: %d", baseCfg.TestnetValidators)
		log.Printf("   - Initial NRN: %d", baseCfg.TestnetInitialNRN)
		log.Printf("   - Local Network: %v", baseCfg.LocalNetworkMode)
		log.Printf("   - Mock NRN Minting: %v", baseCfg.MockNRNMinting)
		log.Printf("   - Simplified Consensus: %v", baseCfg.SimplifiedConsensus)
		log.Printf("   - Disable XION Bridge: %v", baseCfg.DisableXIONBridge)
	}

	// Define command-line flags for subcommands
	chainCmdSet := flag.NewFlagSet("chain", flag.ExitOnError)
	walletCmdSet := flag.NewFlagSet("wallet", flag.ExitOnError)

	// Chain command flags
	chainPort := chainCmdSet.Uint64("port", baseCfg.Port, "HTTP port for blockchain server")
	chainMiner := chainCmdSet.String("miners_address", baseCfg.MinersAddress, "Miner's address")
	// Use the standard default path as the flag's default value
	chainDbPath := chainCmdSet.String("dbpath", utils.GetDefaultRootDBPath(), "Database path") // Updated default

	// Wallet command flags
	defaultWalletPort := uint64(8080)
	walletPort := walletCmdSet.Uint64("port", defaultWalletPort, "HTTP port for wallet server")
	defaultNodeAddress := fmt.Sprintf("http://127.0.0.1:%d", baseCfg.Port)
	blockchainNodeAddress := walletCmdSet.String("node_address", defaultNodeAddress, "Blockchain node address")

	// Check for subcommand
	if len(os.Args) < 2 {
		fmt.Println("Error: Expected 'chain', 'wallet', or 'webgui' subcommand")
		fmt.Println("Usage:")
		fmt.Println("  KNIRVROUTER chain [--port=<port>] [--miners_address=<address>] [--dbpath=<path>]")
		fmt.Println("  KNIRVROUTER wallet [--port=<port>] [--node_address=<url>]")
		fmt.Println("  KNIRVROUTER webgui [--port=<port>]") // <<< NEW Usage
		os.Exit(1)
	}

	switch os.Args[1] {
	case "chain", "-chain":
		chainCmdSet.Parse(os.Args[2:])

		// Validate miner address
		if *chainMiner == "" {
			fmt.Println("Error: Miner's address is required for the chain node")
			chainCmdSet.PrintDefaults()
			os.Exit(1)
		}

		// --- Set constants.BLOCKCHAIN_DB_PATH specifically for 'chain' command ---
		// Priority: Flag > Env Var > Default
		finalDbPath := *chainDbPath // Start with flag value (which defaults to standard path)
		envDbPath := os.Getenv("BLOCKCHAIN_DB_PATH")
		if envDbPath != "" && *chainDbPath == utils.GetDefaultRootDBPath() {
			// If flag wasn't explicitly set (still default) but env var exists, use env var
			finalDbPath = envDbPath
			log.Printf("Overriding default DB path with environment variable: %s", finalDbPath)
		}
		constants.BLOCKCHAIN_DB_PATH = finalDbPath // Set the constant for this process run

		if err := os.MkdirAll(constants.BLOCKCHAIN_DB_PATH, 0755); err != nil {
			log.Fatalf("Error creating database directory '%s': %v", constants.BLOCKCHAIN_DB_PATH, err)
		}
		log.Printf("Chain command using final database path: %s", constants.BLOCKCHAIN_DB_PATH)

		// Start the blockchain logic (which now decides between root and peer/webgui)
		// startVerifyerBlockchain should now ONLY handle the root node case.
		// The peer case is handled by the 'webgui' command.
		StartRootBlockchain(*chainPort, *chainMiner) // Renamed for clarity

	case "wallet", "-wallet":
		walletCmdSet.Parse(os.Args[2:])
		StartWallet(*walletPort, *blockchainNodeAddress)

	default:
		fmt.Println("Error: Expected 'chain', or 'wallet', subcommand")
		os.Exit(1)
	}
}

// isPortAvailable checks if a TCP port is free.
func isPortAvailable(port uint64) bool {
	address := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return false
	}
	listener.Close()
	return true
}

// findAvailablePort starts searching from startPort and finds the next available TCP port.
func findAvailablePort(startPort uint64) uint64 {
	port := startPort
	for {
		if isPortAvailable(port) {
			log.Printf("Found available port: %d", port)
			return port
		}
		port++
		if port > startPort+1000 {
			log.Fatalf("Could not find an available port between %d and %d", startPort, port-1)
		}
	}
}

// StartRootBlockchain handles the ROOT blockchain initialization and startup
// Renamed from startVerifyerBlockchain
func StartRootBlockchain(port uint64, minerAddress string) {
	// --- Root Node Logic ---
	// Check if the specified port is available. If not, FATAL error for root node.
	if !isPortAvailable(port) {
		log.Fatalf("FATAL: Root node port %d is already in use. Cannot start.", port)
	}

	log.Printf("Starting Root blockchain node on port %d", port)
	// Constants (like DB path) should have been set by the 'chain' command handler

	stopMining := make(chan bool)
	miningStopped := make(chan bool)

	var bcs *blockchainserver.BlockchainServer
	var routerIntegration *embedded_knirvchain.RouterIntegration
	var wasmIntegration *wasm_integration.WASMIntegration

	// Initialize core components for the root node
	genesisBlock := blockchain.NewBlock("0x0", 0, 0)

	// Get the LevelDB instance
	db := blockchain.GetLevelDBInstance()

	// Create the blockchain instance
	blockchain1 := blockchain.NewBlockchain(*genesisBlock, minerAddress)
	if blockchain1 == nil {
		log.Fatalf("FATAL: Failed to initialize blockchain instance.") // Handle potential nil return
	}

	// Initialize P2P manager with bootstrap nodes
	bootstrapAddrs := []string{
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
	}

	log.Printf("Initializing P2P manager")
	p2pManager, err := p2p.NewP2PManager(blockchain1, db, bootstrapAddrs)
	if err != nil {
		log.Fatalf("Failed to initialize P2P manager: %v", err)
	}

	// Initialize connectivity proof engine
	log.Printf("Initializing connectivity proof engine")

	// Get faucet endpoint from environment or use default
	faucetEndpoint := os.Getenv("KNIRVORACLE_ENDPOINT")
	if faucetEndpoint == "" {
		// Try alternative environment variable names
		if endpoint := os.Getenv("KNIRVORACLE_FAUCET_ENDPOINT"); endpoint != "" {
			faucetEndpoint = endpoint
		} else {
			faucetEndpoint = "http://localhost:1317" // Default from testnet config
		}
	}
	// Append the NRN mint endpoint path if not already present
	if !strings.Contains(faucetEndpoint, "/api/mint/nrn") {
		faucetEndpoint = faucetEndpoint + "/api/mint/nrn"
	}

	// Initialize blockchain adapter with real blockchain integration
	log.Printf("Initializing blockchain adapter with real blockchain")
	turnBlockchainAdapter := transaction_turnserver.NewBlockchainAdapterWithBlockchain(
		blockchain1,
		minerAddress,
	)

	// Create wrapper for connectivity package
	blockchainAdapter := NewBlockchainAdapterWrapper(turnBlockchainAdapter)

	// Create a simple faucet client adapter for now
	// TODO: Integrate with actual FaucetIntegration from main package
	var faucetClientAdapter connectivity.FaucetClient = nil

	// For now, we'll use nil and let the proof engine fall back to HTTP requests
	log.Printf("Note: Using fallback faucet integration - direct HTTP requests to %s", faucetEndpoint)

	proofEngineConfig := connectivity.ProofEngineConfig{
		NRNMintingEnabled: true,
		FaucetEndpoint:    faucetEndpoint,
		FaucetClient:      faucetClientAdapter,
		BlockchainAdapter: blockchainAdapter,
		PauseChecker:      p2pManager, // P2PManager implements NetworkPauseChecker
		MinConnectivity:   70.0,       // Minimum connectivity score for rewards
		MeasurementWindow: time.Minute * 5,
		RewardMultiplier:  1.0,
	}

	// Get the actual host from P2P manager
	proofEngine := connectivity.NewConnectivityProofEngine(p2pManager.GetHost(), proofEngineConfig)

	// Initialize TURN server with the blockchain adapter
	log.Printf("Initializing TURN server with blockchain adapter")
	turnServer, err := transaction_turnserver.NewServer(3478, 3479, 8080, turnBlockchainAdapter)
	if err != nil {
		log.Printf("Warning: Failed to initialize TURN server: %v", err)
	}

	// Create the blockchain server
	bcs = blockchainserver.NewBlockchainServer(port, blockchain1)

	// Initialize Revolutionary Embedded KNIRVCHAIN
	log.Printf("🚀 Initializing Revolutionary Embedded KNIRVCHAIN...")
	embeddedChainConfig := embedded_knirvchain.GetDefaultConfig()
	embeddedChainConfig.ModelKernel = "hrm"
	embeddedChainConfig.MaxMemoryMB = 1024
	embeddedChainConfig.ConsensusThreshold = 0.75
	embeddedChainConfig.LoRAAdapterCacheSize = 200
	embeddedChainConfig.SkillChainDepth = 15
	embeddedChainConfig.EnableRealTimeUpdates = true

	routerIntegration = embedded_knirvchain.NewRouterIntegration(embeddedChainConfig)
	if err := routerIntegration.Initialize(); err != nil {
		log.Printf("Warning: Failed to initialize embedded KNIRVCHAIN: %v", err)
	} else {
		// Create default skills for testing
		if err := routerIntegration.CreateDefaultSkills(); err != nil {
			log.Printf("Warning: Failed to create default skills: %v", err)
		}

		// Start embedded KNIRVCHAIN HTTP server on a different port
		embeddedChainPort := "8081"
		go func() {
			if err := routerIntegration.StartHTTPServer(embeddedChainPort); err != nil {
				log.Printf("Warning: Failed to start embedded KNIRVCHAIN server: %v", err)
			}
		}()
		log.Printf("🚀 Revolutionary Embedded KNIRVCHAIN started on port %s", embeddedChainPort)
		log.Printf("🔗 Embedded KNIRVCHAIN endpoints available at: http://localhost:%s/embedded-chain/", embeddedChainPort)
		log.Printf("🎯 Revolutionary /invoke endpoint: http://localhost:%s/embedded-chain/invoke", embeddedChainPort)
		log.Printf("🎯 Revolutionary /invoke/protobuf endpoint: http://localhost:%s/embedded-chain/invoke/protobuf", embeddedChainPort)
	}

	// Check if WASM integration should be enabled
	enableWASM := os.Getenv("KNIRV_ENABLE_WASM")
	if enableWASM == "true" || enableWASM == "1" {
		log.Printf("🚀 Initializing Revolutionary WASM KNIRVCHAIN Integration...")

		wasmIntegration = wasm_integration.NewWASMIntegration(wasm_integration.GetAssetsPath())
		if err := wasmIntegration.Initialize(); err != nil {
			log.Printf("Warning: Failed to initialize WASM integration: %v", err)
		} else {
			// Start WASM integration HTTP server on a different port
			wasmPort := "8082"
			go func() {
				if err := wasmIntegration.StartHTTPServer(wasmPort); err != nil {
					log.Printf("Warning: Failed to start WASM integration server: %v", err)
				}
			}()
			log.Printf("🚀 Revolutionary WASM KNIRVCHAIN started on port %s", wasmPort)
			log.Printf("🔗 WASM KNIRVCHAIN endpoints available at: http://localhost:%s/wasm/", wasmPort)
			log.Printf("🎯 Revolutionary WASM /invoke endpoint: http://localhost:%s/wasm/invoke", wasmPort)
			log.Printf("📊 WASM status endpoint: http://localhost:%s/wasm/status", wasmPort)
		}
	} else {
		log.Printf("ℹ️  WASM integration disabled. Set KNIRV_ENABLE_WASM=true to enable")
	}

	// Connect blockchain to P2P manager
	log.Printf("Connecting blockchain to P2P manager")
	p2p.ConnectBlockchainToP2P(blockchain1, p2pManager)

	// Start the P2P manager
	log.Printf("Starting P2P manager")
	if err := p2pManager.Start(); err != nil {
		log.Fatalf("Failed to start P2P manager: %v", err)
	}

	// Start connectivity proof engine
	var connectivityAPI *connectivity.APIServer
	if proofEngine != nil {
		log.Printf("Starting connectivity proof engine")
		if err := proofEngine.Start(); err != nil {
			log.Printf("Warning: Failed to start proof engine: %v", err)
		}

		// Start connectivity API server
		apiPort := 9090 // Default API port for connectivity endpoints
		connectivityAPI = connectivity.NewAPIServer(proofEngine, apiPort)
		go func() {
			log.Printf("Starting connectivity API server on port %d", apiPort)
			if err := connectivityAPI.Start(); err != nil {
				log.Printf("Warning: Failed to start connectivity API server: %v", err)
			}
		}()
	}

	// Start TURN server
	if turnServer != nil {
		log.Printf("Starting TURN server")
		go turnServer.Start()
	}

	// Start the blockchain server
	log.Printf("Starting blockchain server on port %d", port)
	go bcs.Start()

	// Start mining
	log.Printf("Starting mining with miner address: %s", minerAddress)
	go blockchain1.ProofOfWorkMining(minerAddress, stopMining, miningStopped)

	// Consensus is now handled by the P2P manager

	// Setup interrupt handler
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	log.Println("Root node running. Press Ctrl+C to shut down.")
	<-interrupt

	// --- Graceful Shutdown Logic ---
	log.Println("Shutdown signal received. Stopping components...")
	log.Println("Stopping mining...")
	stopMining <- true
	<-miningStopped
	log.Println("Mining stopped.")

	// Consensus is handled by P2P manager, no separate stop needed

	log.Println("Stopping P2P manager...")
	p2pManager.Stop()
	log.Println("P2P manager stopped.")

	// Stop connectivity proof engine and API server
	if proofEngine != nil {
		log.Println("Stopping connectivity proof engine...")
		proofEngine.Stop()
		log.Println("Connectivity proof engine stopped.")
	}

	// Stop connectivity API server
	if connectivityAPI != nil {
		log.Println("Stopping connectivity API server...")
		if err := connectivityAPI.Stop(); err != nil {
			log.Printf("Error stopping connectivity API server: %v", err)
		}
		log.Println("Connectivity API server stopped.")
	}

	// Stop TURN server
	if turnServer != nil {
		log.Println("Stopping TURN server...")
		if err := turnServer.Stop(); err != nil {
			log.Printf("Error stopping TURN server: %v", err)
		}
		log.Println("TURN server stopped.")
	}

	// Stop embedded KNIRVCHAIN
	if routerIntegration != nil {
		log.Println("Stopping Revolutionary Embedded KNIRVCHAIN...")
		if err := routerIntegration.Shutdown(); err != nil {
			log.Printf("Error stopping embedded KNIRVCHAIN: %v", err)
		}
		log.Println("Revolutionary Embedded KNIRVCHAIN stopped.")
	}

	// Stop WASM integration
	if wasmIntegration != nil {
		log.Println("Stopping Revolutionary WASM KNIRVCHAIN Integration...")
		if err := wasmIntegration.Shutdown(); err != nil {
			log.Printf("Error stopping WASM integration: %v", err)
		}
		log.Println("Revolutionary WASM KNIRVCHAIN Integration stopped.")
	}

	// Add bcs.Stop() if implemented
	log.Println("Shutdown complete.")
	os.Exit(0)
}
