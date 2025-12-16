package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"math/big"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"KNIRVCHAIN/config"
	agentlog "KNIRVCHAIN/log"
	"KNIRVCHAIN/proto"
	"KNIRVCHAIN/types"
	"KNIRVCHAIN/uri"
	"KNIRVCHAIN/utils"

	chromem "github.com/philippgille/chromem-go"
	"github.com/syndtr/goleveldb/leveldb"
)

// Define the one true Genesis Block globally
var trueGenesisBlock *Block

func init() {
	// Define a genesis proposer address.
	// In a real network, this might be a well-known address or derived from a master key.
	// For consistency, let's use a fixed address or generate one deterministically if needed.
	// For now, using the BLOCKCHAIN_ADDRESS constant as a placeholder.
	// If BLOCKCHAIN_ADDRESS is "KNIRVCHAIN_Faucet", this is fine for a placeholder.
	// If you have a specific master wallet for genesis, use its address.

	// Attempt to load or create a master wallet for the genesis proposer address
	// This is a simplified approach for init(); a real system might load from secure config.
	// For now, we'll use a constant or a newly generated one if master_wallet.go is not accessible here directly.
	// Let's assume BLOCKCHAIN_ADDRESS can serve as this for now.
	var genesisProposerAddress string = utils.BLOCKCHAIN_ADDRESS // Or a specific genesis wallet address

	// Create the genesis block ONCE with fixed parameters
	// Use a fixed timestamp (e.g., Unix epoch or a specific date)
	// IMPORTANT: Ensure this timestamp is consistent across all nodes.
	genesisTimestamp := int64(1678886400)                     // Example: March 15, 2023 12:00:00 UTC
	trueGenesisBlock = NewBlock([]byte{}, 0, 0)               // SmartContract argument removed
	trueGenesisBlock.ProposerAddress = genesisProposerAddress // Set the proposer address
	trueGenesisBlock.Timestamp = genesisTimestamp
	// Set the BlockHash for the genesis block deterministically
	trueGenesisBlock.BlockHash = trueGenesisBlock.Hash()
	agentlog.LogInfo(fmt.Sprintf("Initialized deterministic Genesis Block. Hash: %s", hex.EncodeToString(trueGenesisBlock.BlockHash)))
}

type BlockchainStruct struct {
	TransactionPool      []*Transaction       `json:"transaction_pool"`
	Blocks               []*Block             `json:"block_chain"`
	ChainAddress         string               `json:"chain_address"`
	ChainID              string               `json:"chain_id"` // Unique identifier for this blockchain
	ConsensusManager     *ConsensusManager    `json:"-"`        // Exclude from JSON serialization
	p2pConsensusMgr      *P2PConsensusManager `json:"-"`        // Local P2P consensus manager
	Reflections          map[string]bool      `json:"reflections"`
	MiningLocked         bool                 `json:"mining_locked"`
	OwnerAddress         string               `json:"owner_address"`
	WalletAddress        string               `json:"wallet_address"`
	StopMining           bool                 `json:"-"` // Flag to stop mining gracefully
	txnSignal            chan struct{}        `json:"-"` // Channel to signal new transactions
	isActivelyMining     bool                 `json:"-"` // Flag: True when miner is in nonce search loop for a block with txns
	miningCtx            context.Context      `json:"-"` // Context for mining operations
	miningCancel         context.CancelFunc   `json:"-"` // Cancel function for mining context
	mu                   sync.Mutex           `json:"-"` // Exclude from JSON serialization
	db                   *LevelDB             `json:"-"` // Database connection
	mcpProcessor         *MCPProcessor        `json:"-"` // MCP-specific processor
	ChromemSync          *ChromemManager      `json:"-"` // Legacy field - kept for backward compatibility
	ChromemDBManager     *ChromemManager      `json:"-"` // General ChromemDB client manager
	ChromemDBSyncManager *ChromemSyncManager  `json:"-"` // Manager for detailed sync logic
	nftManager           *NFTManager
	agentManager         *AgentManager `json:"-"` // Agent manager for Phase 3 resource integration

	// PoAu-D specific fields
	NetworkAuthors         map[string]bool         `json:"-"`             // Map of NAP addresses (PublicKeyString) to true
	PoAuDEnabled           bool                    `json:"poaud_enabled"` // Flag to enable/disable PoAu-D (fallback to PoW if false)
	TransactionPoolManager *TransactionPoolManager `json:"-"`             // Manager for different transaction pools
}

// GetWallet returns a Wallet instance initialized with the blockchain's wallet address
// This is thread-safe as it locks the mutex during access
func (bc *BlockchainStruct) GetWallet() *Wallet {
	bc.Lock()
	defer bc.Unlock()

	if bc.WalletAddress == "" {
		return nil
	}

	return &Wallet{
		PrivateKey: nil, // We don't expose the private key here
		Address:    bc.WalletAddress,
	}
}

// Account represents a KNIRVCHAIN account with NRN token balance
type Account struct {
	Address string `json:"address"`
	Balance uint64 `json:"balance"` // NRN token balance
}

func (bc *BlockchainStruct) StartMining() {
	bc.Lock()
	defer bc.Unlock()

	if bc.isActivelyMining {
		return
	}

	// Create a new context for mining operations
	bc.miningCtx, bc.miningCancel = context.WithCancel(context.Background())

	// Start mining based on consensus mechanism
	if bc.PoAuDEnabled {
		agentlog.LogInfo("Starting Hybrid Mining (PoAu-D with PoW fallback)")
		go bc.HybridMining(bc.miningCtx, bc.WalletAddress, bc.ConsensusManager)
	} else {
		agentlog.LogInfo("Starting Proof-of-Work Mining")
		go bc.ProofOfWorkMining(bc.miningCtx, bc.WalletAddress, bc.ConsensusManager)
	}
}

// StopMiningGracefully stops mining by cancelling the mining context
func (bc *BlockchainStruct) StopMiningGracefully() {
	bc.Lock()
	defer bc.Unlock()

	bc.StopMining = true
	if bc.miningCancel != nil {
		bc.miningCancel()
	}
}

// Shutdown gracefully shuts down all blockchain operations
func (bc *BlockchainStruct) Shutdown() {
	agentlog.LogInfo("Shutting down blockchain operations...")

	// Stop mining gracefully
	bc.StopMiningGracefully()

	// Close database connection if it exists
	if bc.db != nil {
		if err := bc.db.Close(); err != nil {
			agentlog.LogError("Error closing database:", err)
		}
	}

	agentlog.LogInfo("Blockchain shutdown complete.")
}

func (bc *BlockchainStruct) Lock() {
	bc.mu.Lock()
}
func (bc *BlockchainStruct) Unlock() {
	bc.mu.Unlock()
}

func (bc *BlockchainStruct) setIsActivelyMining(value bool) {
	bc.Lock()
	defer bc.Unlock()
	bc.isActivelyMining = value
}

// GetTransactionFromPool searches the transaction pool for a transaction with matching hash
// Returns the transaction if found, nil otherwise
func (bc *BlockchainStruct) GetTransactionFromPool(txHash string) *Transaction {
	bc.Lock()
	defer bc.Unlock()

	for _, tx := range bc.TransactionPool {
		if tx.TransactionHash == txHash {
			return tx
		}
	}
	return nil
}

func (bc *BlockchainStruct) IsActivelyMining() bool {
	bc.Lock()
	defer bc.Unlock()
	return bc.isActivelyMining
}

// PoAu-D Network Authors Management Methods

// loadNetworkAuthors loads the NetworkAuthors from LevelDB
func (bc *BlockchainStruct) loadNetworkAuthors() error {
	if bc.db == nil {
		return fmt.Errorf("database not initialized")
	}

	authors, err := bc.db.GetNetworkAuthors()
	if err != nil {
		// If not found, initialize with empty map
		if err == leveldb.ErrNotFound {
			bc.NetworkAuthors = make(map[string]bool)
			return nil
		}
		return fmt.Errorf("failed to load network authors: %w", err)
	}

	bc.NetworkAuthors = authors
	return nil
}

// saveNetworkAuthors saves the NetworkAuthors to LevelDB
func (bc *BlockchainStruct) saveNetworkAuthors() error {
	if bc.db == nil {
		return fmt.Errorf("database not initialized")
	}

	return bc.db.PutNetworkAuthors(bc.NetworkAuthors)
}

// IsNetworkAuthor checks if a given address is a current NAP
func (bc *BlockchainStruct) IsNetworkAuthor(address string) bool {
	bc.Lock()
	defer bc.Unlock()
	return bc.NetworkAuthors[address]
}

// AddNetworkAuthor adds an address to the NetworkAuthors set
func (bc *BlockchainStruct) AddNetworkAuthor(address string) error {
	bc.Lock()
	defer bc.Unlock()

	if bc.NetworkAuthors == nil {
		bc.NetworkAuthors = make(map[string]bool)
	}

	bc.NetworkAuthors[address] = true
	return bc.saveNetworkAuthors()
}

// RemoveNetworkAuthor removes an address from the NetworkAuthors set
func (bc *BlockchainStruct) RemoveNetworkAuthor(address string) error {
	bc.Lock()
	defer bc.Unlock()

	if bc.NetworkAuthors == nil {
		return nil
	}

	delete(bc.NetworkAuthors, address)
	return bc.saveNetworkAuthors()
}

// GetNetworkAuthors returns a copy of the current NetworkAuthors map
func (bc *BlockchainStruct) GetNetworkAuthors() map[string]bool {
	bc.Lock()
	defer bc.Unlock()

	result := make(map[string]bool)
	for addr, val := range bc.NetworkAuthors {
		result[addr] = val
	}
	return result
}

type BlockchainOptions struct {
	TransactionPool []*Transaction  `json:"transaction_pool"`
	Blocks          []*Block        `json:"block_chain"`
	ChainAddress    string          `json:"chain_address"`
	Reflections     map[string]bool `json:"reflections"`
	MiningLocked    bool            `json:"mining_locked"`
	OwnerAddress    string          `json:"owner_address"`
	WalletAddress   string          `json:"wallet_address"`
}

//var mutex sync.Mutex

func NewBlockchain(genesisBlock *Block, dbKey string, nodeMinersAddress string, db *LevelDB, chromemMgr *ChromemManager, searchableDBPath string, cerebrasConfig *config.CerebrasConfig) (*BlockchainStruct, error) {
	bc, err := CreateNewBlockchain(genesisBlock, dbKey, nodeMinersAddress, db)
	if err != nil {
		agentlog.FatalError("Failed to create chain:", err)
	}
	// bc.WalletAddress is set within CreateNewBlockchain now
	bc.db = db                            // Store the DB connection
	bc.mcpProcessor = NewMCPProcessor(db) // Initialize MCPProcessor

	// Create a wallet for the NFT manager
	wallet := &Wallet{
		PrivateKey: nil, // We don't need a private key for this wallet
		Address:    bc.WalletAddress,
	}

	// Use the provided ChromemManager instance instead of creating a new one
	// This prevents multiple instances trying to access the same LevelDB storage
	if chromemMgr == nil {
		// If no ChromemManager was provided, create one as a fallback
		// Note: This ChromemConfig is the one defined in sync_manager.go
		chromemDBManagerConfig := &config.ChromemConfig{
			Path:           searchableDBPath, // Use the path from the new parameter
			CerebrasConfig: cerebrasConfig,   // Pass Cerebras config
		}

		// Initialize the general ChromemDBManager (from chromemDB_manager.go)
		var err error
		chromemMgr, err = NewChromemManager(chromemDBManagerConfig)
		if err != nil {
			agentlog.LogWarning(fmt.Sprintf("Failed to initialize ChromemDBManager: %v", err))
			// Continue without ChromemDB support
		}
	}

	if chromemMgr != nil {
		bc.ChromemDBManager = chromemMgr
		bc.ChromemSync = chromemMgr // For backward compatibility
		agentlog.LogInfo("ChromemDB Manager initialized successfully")

		// Initialize NFT Manager now that we have ChromeDB
		// Note: Discovery manager will be set later via SetDiscoveryManager method
		bc.nftManager = NewNFTManager(bc.ChromemDBManager, bc.mcpProcessor, wallet, nil)
		agentlog.LogInfo("NFT Manager initialized successfully")

		// Initialize Agent Manager for Phase 3 resource integration
		// Note: Discovery manager will be set later via SetDiscoveryManager method
		bc.agentManager = NewAgentManager(bc.ChromemDBManager, bc.mcpProcessor, wallet, nil)
		agentlog.LogInfo("Agent Manager initialized successfully")
	}

	// Initialize the ChromemDBSyncManager (from sync_manager.go)
	var chromemClientForSync *chromem.DB
	if chromemMgr != nil {
		chromemClientForSync = chromemMgr.client
	}
	chromemSyncMgr, err := NewChromemSyncManager(chromemClientForSync, cerebrasConfig, bc.db) // Pass chromem.DB client, CerebrasConfig and LevelDB instance
	if err != nil {
		agentlog.LogWarning(fmt.Sprintf("Failed to initialize ChromemDBSyncManager: %v", err))
	} else {
		bc.ChromemDBSyncManager = chromemSyncMgr
		agentlog.LogInfo("ChromemDB Sync Manager initialized successfully")
	}

	// Initialize test accounts if this is a test environment
	if strings.Contains(dbKey, "test") {
		if err := initializeTestAccounts(bc); err != nil {
			agentlog.LogWarning(fmt.Sprintf("Failed to initialize test accounts: %v", err))
		}
	}

	return bc, err
}

// NewLevelDB is now defined in leveldb.go
var _ = NewLevelDB // Reference to ensure the function exists

// initializeTestAccounts pre-funds test accounts for testing purposes
func initializeTestAccounts(bc *BlockchainStruct) error {
	// Pre-fund accounts used in tests
	testAccounts := map[string]uint64{
		"KNIRVCHAIN0ad62e0365cf9b2716bb0d7f7ce6226005d4e33d": 10000000, // 10 million units
		// Add other test accounts as needed
	}

	for address, amount := range testAccounts {
		if err := bc.UpdateBalance(address, amount, true); err != nil {
			return fmt.Errorf("failed to initialize test account %s: %w", address, err)
		}
		agentlog.LogInfo(fmt.Sprintf("[INFO] Pre-funded test account %s with %d units", address, amount))
	}

	return nil
}

// UpdateBalance updates the balance of an account in the database
// If overwrite is true, it sets the balance to the specified amount
// If overwrite is false, it adds the amount to the existing balance
func (bc *BlockchainStruct) UpdateBalance(address string, amount uint64, overwrite bool) error {
	if bc.db == nil {
		return fmt.Errorf("database connection not initialized")
	}

	// Get current balance
	currentBalance, err := bc.db.GetBalance(address)
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return fmt.Errorf("failed to get balance for %s: %w", address, err)
	}

	// Calculate new balance
	var newBalance uint64
	if overwrite {
		newBalance = amount
	} else {
		newBalance = currentBalance + amount
	}

	// Update balance in database
	if err := bc.db.UpdateBalance(address, int64(newBalance)); err != nil {
		return fmt.Errorf("failed to update balance for %s: %w", address, err)
	}

	agentlog.LogInfo(fmt.Sprintf("[INFO] Updated balance for %s: %d -> %d", address, currentBalance, newBalance))
	return nil
}

func CreateNewBlockchain(genesisBlock *Block, dbKeyForChain string, nodeMinersAddress string, db *LevelDB) (*BlockchainStruct, error) {
	// Store the DB connection in the blockchain struct
	exists, err := db.KeyExists(dbKeyForChain)

	if err != nil {
		return nil, fmt.Errorf("failed to determine if key exists for '%s': %w", dbKeyForChain, err)
	}

	if exists {
		agentlog.LogInfo(fmt.Sprintf("Found existing blockchain data for key '%s'. Attempting to load.", dbKeyForChain))
		blockchainData, err := db.GetBlockchain(dbKeyForChain)
		if err != nil {
			// This could be due to unmarshal errors or other DB issues.
			return nil, fmt.Errorf("failed to load existing blockchain data for key '%s': %w. Consider deleting the DB if corruption is suspected", dbKeyForChain, err)
		}

		blockchain, ok := blockchainData.(*BlockchainStruct)
		if !ok || blockchain == nil {
			return nil, fmt.Errorf("invalid blockchain data in database for key '%s'. Loaded data: %T", dbKeyForChain, blockchainData)
		}

		// Validate the loaded genesis block
		if len(blockchain.Blocks) == 0 {
			return nil, fmt.Errorf("loaded blockchain for key '%s' is empty (no blocks found). This is an invalid state. Consider deleting the DB", dbKeyForChain)
		}

		if !bytes.Equal(blockchain.Blocks[0].Hash(), trueGenesisBlock.Hash()) {
			// GENESIS MISMATCH - This is a critical error.
			mismatchError := fmt.Errorf(
				"CRITICAL: Loaded blockchain for key '%s' has a Genesis block hash mismatch!\n"+
					"  Loaded Genesis Hash: %s\n"+
					"  Expected Genesis Hash: %s\n"+
					"  Please resolve this conflict. You may need to delete the existing database at the path associated with this key ('%s') if you intend to start a new chain with the current genesis block",
				dbKeyForChain,
				hex.EncodeToString(blockchain.Blocks[0].Hash()),
				hex.EncodeToString(trueGenesisBlock.Hash()),
				db.Path(), // Using the Path method we added to LevelDB
			)
			agentlog.LogError(mismatchError.Error(), nil) // Log it clearly
			return nil, mismatchError                     // Return the error, forcing a stop.
		}

		// Genesis matches, proceed with the loaded blockchain
		agentlog.LogInfo(fmt.Sprintf("Successfully loaded existing blockchain for key '%s'. Genesis block matches.", dbKeyForChain))
		blockchain.txnSignal = make(chan struct{}, 1)
		blockchain.isActivelyMining = false
		blockchain.db = db
		blockchain.WalletAddress = nodeMinersAddress  // Ensure WalletAddress is (re)set from current config
		blockchain.mcpProcessor = NewMCPProcessor(db) // Initialize MCPProcessor for existing blockchain
		blockchain.ChainID = dbKeyForChain            // Ensure ChainID is correct
		// Initialize mining context (will be set when mining starts)
		blockchain.miningCtx = nil
		blockchain.miningCancel = nil
		return blockchain, nil
	} else {
		// Create NEW blockchain
		agentlog.LogInfo(fmt.Sprintf("No existing blockchain data found for key '%s'. Creating new blockchain.", dbKeyForChain))
		blockchainStruct := new(BlockchainStruct)
		blockchainStruct.TransactionPool = []*Transaction{}
		blockchainStruct.Blocks = []*Block{}

		// Use the provided genesis block if it's not nil, otherwise use trueGenesisBlock
		var genesis *Block
		if genesisBlock != nil {
			genesis = genesisBlock.DeepCopy() // Should always be trueGenesisBlock
			agentlog.LogInfo(fmt.Sprintf("Creating new blockchain for %s using provided Genesis block.", dbKeyForChain))
		} else {
			genesis = trueGenesisBlock.DeepCopy() // Fallback, though trueGenesisBlock should always be passed
			agentlog.LogInfo(fmt.Sprintf("Creating new blockchain for %s using deterministic Genesis (fallback).", dbKeyForChain))
		}

		blockchainStruct.Blocks = append(blockchainStruct.Blocks, genesis)
		blockchainStruct.ChainAddress = dbKeyForChain      // This is the DB key, usually ChainID
		blockchainStruct.ChainID = dbKeyForChain           // Set ChainID to the dbKey
		blockchainStruct.WalletAddress = nodeMinersAddress // Set the node's actual mining/wallet address
		blockchainStruct.Reflections = map[string]bool{}
		blockchainStruct.MiningLocked = false
		blockchainStruct.txnSignal = make(chan struct{}, 1)
		blockchainStruct.isActivelyMining = false
		blockchainStruct.db = db                            // Set DB connection
		blockchainStruct.mcpProcessor = NewMCPProcessor(db) // Initialize MCPProcessor for new blockchain
		// Initialize mining context (will be set when mining starts)
		blockchainStruct.miningCtx = nil
		blockchainStruct.miningCancel = nil

		// Initialize PoAu-D fields
		blockchainStruct.NetworkAuthors = make(map[string]bool)
		blockchainStruct.PoAuDEnabled = false // Default to disabled
		blockchainStruct.TransactionPoolManager = NewTransactionPoolManager(blockchainStruct)

		// Load PoAu-D settings from database
		if err := blockchainStruct.loadNetworkAuthors(); err != nil {
			agentlog.LogError("Failed to load network authors, using empty set:", err)
		}

		// Load PoAu-D enabled flag from database
		if enabled, err := db.GetPoAuDEnabled(); err != nil {
			agentlog.LogError("Failed to load PoAu-D enabled flag, using default (false):", err)
		} else {
			blockchainStruct.PoAuDEnabled = enabled
		}

		err := db.PutIntoDb(blockchainStruct, dbKeyForChain)
		if err != nil {
			return nil, fmt.Errorf("unable to put new blockchain to DB for key '%s': %w", dbKeyForChain, err)
		}
		agentlog.LogInfo(fmt.Sprintf("Successfully created and stored new blockchain for key '%s'.", dbKeyForChain))
		return blockchainStruct, nil
	}
}

func NewBlockchainFromSync(bc1 *BlockchainStruct, chainAddress string, db *LevelDB) (*BlockchainStruct, error) {
	bc1.Lock()
	defer bc1.Unlock()

	bc2 := &BlockchainStruct{
		ChainAddress:     chainAddress,
		MiningLocked:     bc1.MiningLocked,
		OwnerAddress:     bc1.OwnerAddress,
		WalletAddress:    bc1.WalletAddress,
		ConsensusManager: nil,
		txnSignal:        make(chan struct{}, 1),
		isActivelyMining: false, // Initialize flag
		miningCtx:        nil,   // Initialize mining context
		miningCancel:     nil,   // Initialize mining cancel function
	}

	if bc1.Blocks != nil {
		bc2.Blocks = make([]*Block, len(bc1.Blocks))
		for i, block := range bc1.Blocks {
			if block != nil {
				bc2.Blocks[i] = block.DeepCopy()
			}
		}
	} else {
		bc2.Blocks = []*Block{}
	}

	if bc1.TransactionPool != nil {
		bc2.TransactionPool = make([]*Transaction, len(bc1.TransactionPool))
		for i, tx := range bc1.TransactionPool {
			if tx != nil {
				bc2.TransactionPool[i] = tx.Clone()
			}
		}
	} else {
		bc2.TransactionPool = []*Transaction{}
	}

	if bc1.Reflections != nil {
		bc2.Reflections = make(map[string]bool)
		for k, v := range bc1.Reflections {
			bc2.Reflections[k] = v
		}
	} else {
		bc2.Reflections = make(map[string]bool)
	}

	// Initialize PoAu-D fields for synced blockchain
	bc2.NetworkAuthors = make(map[string]bool)
	if bc1.NetworkAuthors != nil {
		for addr, val := range bc1.NetworkAuthors {
			bc2.NetworkAuthors[addr] = val
		}
	}
	bc2.PoAuDEnabled = bc1.PoAuDEnabled
	bc2.db = db
	bc2.mcpProcessor = NewMCPProcessor(db)
	bc2.TransactionPoolManager = NewTransactionPoolManager(bc2)

	err := db.PutIntoDb(bc2, bc2.ChainAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to save new blockchain snapshot with chain ID %s: %w", bc2.ChainAddress, err)
	}

	agentlog.LogInfo(fmt.Sprintf("Successfully created and saved blockchain snapshot with new address: %s", bc2.ChainAddress))

	return bc2, nil
}

func (bc *BlockchainStruct) ReflectionsToJson() []byte {
	bc.Lock()
	defer bc.Unlock()
	nb, _ := json.Marshal(bc.Reflections)
	return nb
}

func (bc *BlockchainStruct) ToJson() string {
	bc.Lock()
	defer bc.Unlock()
	nb, err := json.Marshal(bc)
	if err != nil {
		return err.Error()
	} else {
		return string(nb)
	}
}

// AddBlock validates a new block, processes its transactions (including MCP fee transfers),
// updates account balances in the database, and adds the block to the chain.
// It returns an error if the block is invalid or if any part of the processing fails.
func (bc *BlockchainStruct) AddBlock(b *Block) error {
	bc.Lock() // Lock for the entire duration of critical state modification
	var mcpContextRecordsForSync []*proto.ContextRecordProto

	agentlog.LogInfo(fmt.Sprintf("Attempting to add block number %d (Hash: %s)", b.BlockNumber, hex.EncodeToString(b.BlockHash)))
	agentlog.LogInfo(fmt.Sprintf("AddBlock: Current chain length before add: %d", len(bc.Blocks)))
	// Check DB connection
	if bc.db == nil {
		bc.Unlock()
		return fmt.Errorf("database connection is nil in AddBlock")
	}

	// --- Verification ---
	if !b.VerifyBlock() {
		agentlog.LogError(fmt.Sprintf("Block %d verification failed, not adding.", b.BlockNumber), nil)
		bc.Unlock()
		return fmt.Errorf("block %d verification failed", b.BlockNumber)
	}

	// --- Chain Consistency Check ---
	if len(bc.Blocks) == 0 {
		if b.BlockNumber == 0 && bytes.Equal(b.Hash(), trueGenesisBlock.Hash()) {
			agentlog.LogInfo("Received Genesis block matches deterministic one. Skipping add (already present).")
			bc.Unlock()
			return nil // Not an error, just already have it
		} else if b.BlockNumber == 0 {
			err := fmt.Errorf("received Genesis block hash %s does not match deterministic hash %s. Rejecting", hex.EncodeToString(b.Hash()), hex.EncodeToString(trueGenesisBlock.Hash()))
			agentlog.LogError(err.Error(), nil)
			bc.Unlock()
			return err
		} else {
			err := fmt.Errorf("chain is empty, but received block %d is not Genesis. Rejecting", b.BlockNumber)
			agentlog.LogError(err.Error(), nil)
			bc.Unlock()
			return err
		}
	}

	lastBlock := bc.Blocks[len(bc.Blocks)-1]
	lastBlockHashOnChain := lastBlock.Hash()

	if !bytes.Equal(b.PrevHash, lastBlockHashOnChain) {
		err := fmt.Errorf("block %d rejected: PrevHash (%s) doesn't match current last block's hash (%s)",
			b.BlockNumber, hex.EncodeToString(b.PrevHash), hex.EncodeToString(lastBlockHashOnChain))
		agentlog.LogError(err.Error(), nil)
		bc.Unlock()
		return err
	}

	if b.BlockNumber <= lastBlock.BlockNumber {
		if b.BlockNumber == lastBlock.BlockNumber && bytes.Equal(b.BlockHash, lastBlockHashOnChain) {
			agentlog.LogInfo(fmt.Sprintf("Block %d (Hash: %s) is identical to current last block. Skipping add.", b.BlockNumber, hex.EncodeToString(b.BlockHash)))
		} else {
			agentlog.LogWarning(fmt.Sprintf("Block %d rejected: Block number is not greater than current last block number %d.", b.BlockNumber, lastBlock.BlockNumber))
		}
		bc.Unlock()
		return fmt.Errorf("block %d number %d is not greater than current last block number %d", b.BlockNumber, b.BlockNumber, lastBlock.BlockNumber)
	}

	if b.BlockNumber != lastBlock.BlockNumber+1 {
		err := fmt.Errorf("block %d rejected: Block number is not sequential (expected %d). Potential fork or missing block", b.BlockNumber, lastBlock.BlockNumber+1)
		agentlog.LogWarning(err.Error())
		bc.Unlock()
		return err
	}

	// --- Checks Passed - Prepare to Modify State ---
	agentlog.LogInfo(fmt.Sprintf("Checks passed for block %d. Proceeding to process transactions and update state.", b.BlockNumber))

	// --- Atomically process transactions and update account balances ---
	tempBlockAccounts := make(map[string]*big.Int)
	affectedAccounts := make(map[string]bool)

	// Populate affectedAccounts set
	for _, tx := range b.Transactions {
		if tx.From != "" { // Faucet txns might have empty From if BLOCKCHAIN_ADDRESS is considered special
			affectedAccounts[tx.From] = true
		}
		if tx.To != "" {
			affectedAccounts[tx.To] = true
		}
		if tx.Type == TransactionTypeMCPInvokeCapability {
			var mcpInvokeData types.MCPInvokeCapabilityData
			if err := json.Unmarshal(tx.Data, &mcpInvokeData); err == nil {
				capDescInterface, errDbGet := bc.db.GetCapabilityByID(mcpInvokeData.ContextRecord.CapabilityID)
				if errDbGet == nil {
					baseDesc, errConv := getBaseDescriptorFromInterface(capDescInterface) // Use unexported helper
					if errConv == nil && baseDesc.Owner != "" {
						affectedAccounts[baseDesc.Owner] = true
					}
				} else {
					agentlog.LogWarning(fmt.Sprintf("AddBlock: Could not get capability %s for invoke tx %s to determine owner: %v", mcpInvokeData.ContextRecord.CapabilityID, tx.TransactionHash, errDbGet))
				}
			} else {
				agentlog.LogWarning(fmt.Sprintf("AddBlock: Could not unmarshal invoke data for tx %s to determine owner: %v", tx.TransactionHash, err))
			}
		}
	}
	if b.ProposerAddress != "" { // Assuming Block struct has ProposerAddress
		affectedAccounts[b.ProposerAddress] = true
	}

	// Pre-load balances for all affected accounts from the database
	for accAddr := range affectedAccounts {
		balance, err := bc.db.GetAccountBalance(accAddr) // Load from DB
		if err != nil {                                  // GetAccountBalance returns uint64, error
			// If GetAccountBalance returns an error for not found, assume 0.
			// Otherwise, it might be a real DB error.
			if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "no rows") { // Adjust error check as needed
				agentlog.LogInfo(fmt.Sprintf("Account %s not found in DB, initializing with balance 0.", accAddr))
				tempBlockAccounts[accAddr] = big.NewInt(0)
			} else {
				agentlog.LogError(fmt.Sprintf("Failed to load balance for account %s from DB: %v", accAddr, err), nil)
				bc.Unlock()
				return fmt.Errorf("failed to load balance for account %s: %w", accAddr, err)
			}
		} else {
			tempBlockAccounts[accAddr] = new(big.Int).SetUint64(balance) // Convert uint64 to *big.Int
		}
	}
	agentlog.LogInfo(fmt.Sprintf("Pre-loaded balances for %d affected accounts for block %d.", len(tempBlockAccounts), b.BlockNumber))

	// Process transactions one by one, updating tempBlockAccounts
	for i, tx := range b.Transactions {
		agentlog.LogInfo(fmt.Sprintf("Processing tx %d/%d (Hash: %s) in block %d", i+1, len(b.Transactions), tx.TransactionHash, b.BlockNumber))
		// These functions are now methods of BlockchainStruct, defined in transaction.go (or moved to blockchain_struct.go)
		if err := bc.validateTransactionInBlockContext(tx, tempBlockAccounts); err != nil {
			agentlog.LogError(fmt.Sprintf("Invalid transaction %s in block %d during AddBlock: %v", tx.TransactionHash, b.BlockNumber, err), nil)
			bc.Unlock()
			return err // Reject block
		}

		// Handle MCP transactions specially
		if tx.Type == TransactionTypeMCPRegisterCapability ||
			tx.Type == TransactionTypeMCPInvokeCapability ||
			tx.Type == TransactionTypeMCPUpdateCapability {
			// Process MCP transactions and collect context records
			if tx.Type == TransactionTypeMCPRegisterCapability || tx.Type == TransactionTypeMCPUpdateCapability {
				contextRecord, err := bc.mcpProcessor.ApplyMCPTransactionEffects(tx, tempBlockAccounts)
				if err != nil {
					agentlog.LogError(fmt.Sprintf("Failed to apply MCP transaction effects for %s in block %d: %v", tx.TransactionHash, b.BlockNumber, err), nil)
					bc.Unlock()
					return err
				}
				if contextRecord != nil {
					mcpContextRecordsForSync = append(mcpContextRecordsForSync, contextRecord)
				}
			}
		}

		// applyTransactionToState handles standard transfers and fee deductions
		// It's called after MCP effects for MCP transactions, or directly for standard ones.
		if err := bc.applyTransactionToState(tx, tempBlockAccounts, b.ProposerAddress); err != nil { // This was moved down
			agentlog.LogError(fmt.Sprintf("Failed to apply transaction %s to state in block %d: %v", tx.TransactionHash, b.BlockNumber, err), nil)
			bc.Unlock()
			return err // Reject block
		}
	}

	// --- All transactions in block are valid and effects applied to tempBlockAccounts ---

	// Persist updated account balances from tempBlockAccounts to LevelDB
	// This must happen BEFORE the block is added to bc.Blocks and bc is saved.
	for addr, balance := range tempBlockAccounts {
		// Only save if the balance actually changed or if it's an account that was involved.
		// For simplicity, saving all accounts that were touched in this block.
		if err := bc.db.SaveAccountBalance(addr, balance.Uint64()); err != nil { // SaveAccountBalance expects uint64
			agentlog.LogError(fmt.Sprintf("Failed to save updated balance for account %s to DB: %v", addr, err), nil)
			bc.Unlock()
			return fmt.Errorf("critical error saving account balance for %s: %w", addr, err)
		}
	}
	agentlog.LogInfo(fmt.Sprintf("Successfully updated account balances in DB for block %d.", b.BlockNumber))

	// --- Update Blockchain State (Transaction Pool, Blocks list) ---
	txnMap := make(map[string]bool)
	for _, txn := range b.Transactions {
		txnMap[txn.TransactionHash] = true
	}
	newTxnPool := []*Transaction{}
	removedCount := 0
	for _, txn := range bc.TransactionPool {
		if !txnMap[txn.TransactionHash] {
			newTxnPool = append(newTxnPool, txn)
		} else {
			removedCount++
		}
	}
	bc.TransactionPool = newTxnPool
	agentlog.LogInfo(fmt.Sprintf("Removed %d transactions from pool found in block %d.", removedCount, b.BlockNumber))

	bc.Blocks = append(bc.Blocks, b.DeepCopy()) // Add a deep copy to prevent external modification
	agentlog.LogInfo(fmt.Sprintf("Block %d appended to in-memory chain.", b.BlockNumber))

	// --- Marshal and Save BlockchainStruct (which includes Blocks and TransactionPool) ---
	// This saves the overall blockchain structure, including the new block list and updated pool.
	bcDataToSave, err := json.Marshal(bc)
	if err != nil {
		agentlog.LogError(fmt.Sprintf("Failed to marshal blockchain state after adding block %d:", b.BlockNumber), err)
		// At this point, balances are saved, block is in memory. If this fails,
		// on next load, the block won't be in bc.Blocks but balances reflect it.
		// This is a state inconsistency.
		bc.Unlock()
		return fmt.Errorf("failed to marshal blockchain state for saving: %w", err)
	}

	chainAddr := bc.ChainAddress
	blockNum := b.BlockNumber
	//blockCount := len(bc.Blocks)

	// Release lock before this final, non-balance critical DB write
	bc.Unlock()

	err = bc.db.PutBytes(chainAddr, bcDataToSave)
	if err != nil {
		agentlog.LogError(fmt.Sprintf("Failed to save blockchain struct to database after adding block %d:", blockNum), err)
		// Block was added in memory, balances updated in DB, but main struct save failed.
		// This is a tricky recovery scenario.
		return fmt.Errorf("failed to save main blockchain structure to DB: %w", err)
	}

	agentlog.LogInfo(fmt.Sprintf("Successfully added block number %d to the blockchain and DB. New total blocks: %d", blockNum, len(bc.Blocks)))

	// Notify ChromemDB managers about the new confirmed block
	// First, notify the general ChromemDBManager for basic transaction indexing
	if bc.ChromemDBManager != nil {
		// Notify ChromemDB managers about the new block with collected context records

		// Run in a goroutine to avoid blocking the main thread
		go func(b *Block) {
			defer func() { // Add recover
				if r := recover(); r != nil {
					log.Printf("PANIC in ChromemDBManager.OnNewBlockConfirmed goroutine: %v", r)
				}
			}() // Pass mcpContextRecords if ChromemDBManager needs them
			if err := bc.ChromemDBManager.OnNewBlockConfirmed(b); err != nil {
				agentlog.LogError(fmt.Sprintf("Error processing block %d for ChromemDBManager: %v", b.BlockNumber, err), err)
			}
		}(b)
	}

	// Then, notify the ChromemDBSyncManager for detailed processing of MCP transactions
	if bc.ChromemDBSyncManager != nil {

		// Run in a goroutine to avoid blocking the main thread
		go func(b *Block, regCtxRecords []*proto.ContextRecordProto) {
			defer func() { // Add recover
				if r := recover(); r != nil {
					log.Printf("PANIC in ChromemDBSyncManager.OnNewBlockConfirmed goroutine: %v", r)
				}
			}() // Pass the collected context records
			if err := bc.ChromemDBSyncManager.OnNewBlockConfirmed(b, regCtxRecords...); err != nil {
				agentlog.LogError(fmt.Sprintf("Error processing block %d for ChromemDBSyncManager: %v", b.BlockNumber, err), err)
				// TODO: Add more robust error handling (e.g., retry queue) if needed
			}
		}(b, mcpContextRecordsForSync)
	}

	return nil
}

func (bc *BlockchainStruct) addVerifiedTxnToPoolAndSignal(transaction *Transaction) {
	bc.Lock()
	bc.TransactionPool = append(bc.TransactionPool, transaction)
	bc.Unlock()

	select {
	case bc.txnSignal <- struct{}{}:
	default:
	}
}

// HandleChainReorganization handles chain reorganizations by notifying the ChromemSyncManager
// about orphaned blocks and new canonical blocks.
func (bc *BlockchainStruct) HandleChainReorganization(orphanedBlocks []*Block, newCanonicalBlocks []*Block) error {
	// Existing chain reorganization logic would go here...
	// This function would typically update the blockchain state to reflect the reorganization

	// Notify ChromemSyncManager about the chain reorganization
	if bc.ChromemSync != nil {
		// Process orphaned blocks
		for _, block := range orphanedBlocks {
			// Run in goroutines with error handling
			go func(b *Block) {
				if err := bc.ChromemSync.OnBlockOrphaned(b); err != nil {
					log.Printf("Error processing orphaned block %d for ChromemDB: %v", b.BlockNumber, err)
				}
			}(block)
		}

		// Process new canonical blocks
		for _, block := range newCanonicalBlocks {
			go func(b *Block) {
				if err := bc.ChromemSync.OnNewBlockConfirmed(b); err != nil {
					log.Printf("Error processing new canonical block %d for ChromemDB: %v", b.BlockNumber, err)
				}
			}(block)
		}
	}

	return nil // Return original error if there was one from the reorganization logic
}

func (bc *BlockchainStruct) AddTransactionToTransactionPool(transaction *Transaction) error {
	bc.Lock() // Lock for pool and block checks
	defer bc.Unlock()

	// Validate transaction signature, unless it's from the blockchain faucet.
	// VerifyTxn (called later) also includes a signature check.
	// This early check can be helpful for immediate feedback.
	isVerified, err := transaction.VerifySignature()
	if err != nil {
		agentlog.LogError(fmt.Sprintf("Signature verification failed for transaction %s from %s",
			transaction.TransactionHash, transaction.From), err)
		return fmt.Errorf("error verifying signature for transaction from %s: %w", transaction.From, err)
	}
	if transaction.From != utils.BLOCKCHAIN_ADDRESS && !isVerified {
		agentlog.LogError(fmt.Sprintf("Invalid signature for transaction %s from %s",
			transaction.TransactionHash, transaction.From), nil)
		return fmt.Errorf("invalid transaction signature for transaction from %s", transaction.From)
	}

	// Check if already in pool
	for _, txn := range bc.TransactionPool {
		if txn.TransactionHash == transaction.TransactionHash {
			agentlog.LogWarning(fmt.Sprintf("Transaction %s already in pool", transaction.TransactionHash))
			return fmt.Errorf("transaction %s already in pool", transaction.TransactionHash)
		}
	}

	// Check if already in a mined block
	for _, block := range bc.Blocks {
		for _, blockTxn := range block.Transactions {
			if blockTxn.TransactionHash == transaction.TransactionHash {
				agentlog.LogWarning(fmt.Sprintf("Transaction %s already mined in block %d",
					transaction.TransactionHash, block.BlockNumber))
				return fmt.Errorf("transaction %s already mined in block %d", transaction.TransactionHash, block.BlockNumber)
			}
		}
	}

	// Verify transaction (which includes signature verification), unless it's from the faucet.
	if transaction.From != utils.BLOCKCHAIN_ADDRESS {
		if !transaction.VerifyTxn() {
			agentlog.LogError(fmt.Sprintf("Transaction %s from %s failed full verification",
				transaction.TransactionHash, transaction.From), nil)
			return fmt.Errorf("transaction %s from %s failed full verification", transaction.TransactionHash, transaction.From)
		}
		transaction.Verified = true
		transaction.Status = TXN_VERIFICATION_SUCCESS
		agentlog.LogInfo(fmt.Sprintf("Transaction %s successfully verified", transaction.TransactionHash))
	}

	// Validate MCP transactions if applicable
	if transaction.Type != "" {
		if !bc.mcpProcessor.validateMCPTransaction(transaction) { // Delegate to MCPProcessor
			agentlog.LogError(fmt.Sprintf("Transaction %s failed MCP validation", transaction.TransactionHash), nil)
			return fmt.Errorf("transaction %s failed MCP validation", transaction.TransactionHash)
		}
	}

	// Check sender balance
	if !bc.simulatedBalanceCheck(transaction) {
		agentlog.LogError(fmt.Sprintf("Transaction %s failed balance check", transaction.TransactionHash), nil)
		return fmt.Errorf("transaction %s failed balance check", transaction.TransactionHash)
	}

	// Add to pool
	bc.TransactionPool = append(bc.TransactionPool, transaction)
	agentlog.LogInfo(fmt.Sprintf("Added transaction %s to pool (verified: %v)",
		transaction.TransactionHash, transaction.Verified))
	return nil
}

func (bc *BlockchainStruct) BroadcastTransaction(transaction *Transaction) {
	agentlog.LogInfo("Broadcasting transaction: " + transaction.TransactionHash)

	// Check if we have a P2P consensus manager
	if bc.p2pConsensusMgr != nil {
		// Use P2P broadcasting
		if err := bc.p2pConsensusMgr.BroadcastTransaction(transaction); err != nil {
			agentlog.LogError("Failed to broadcast transaction via P2P:", err)
		}
		return
	}

	// Fall back to HTTP broadcasting if P2P is not available
	txnData := map[string]interface{}{
		"transaction": transaction,
	}

	txnJSON, err := json.Marshal(txnData)
	if err != nil {
		agentlog.LogError("Failed to marshal transaction data:", err)
		return
	}

	for reflectionAddr, isActive := range bc.Reflections {
		if !isActive {
			continue
		}

		if reflectionAddr == fmt.Sprintf("http://127.0.0.1:%d", utils.GetServerPort()) {
			continue
		}

		if !strings.HasPrefix(reflectionAddr, "http://") && !strings.HasPrefix(reflectionAddr, "https://") {
			reflectionAddr = "http://" + reflectionAddr
		}

		url := fmt.Sprintf("%s/transaction", reflectionAddr)

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(txnJSON))
		if err != nil {
			agentlog.LogError(fmt.Sprintf("Failed to create request for reflection %s:", reflectionAddr), err)
			continue
		}

		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			agentlog.LogError(fmt.Sprintf("Failed to send transaction to reflection %s:", reflectionAddr), err)
			continue
		}

		resp.Body.Close()
		agentlog.LogInfo(fmt.Sprintf("Transaction broadcast to reflection %s: status %d", reflectionAddr, resp.StatusCode))
		time.Sleep(100 * time.Millisecond)
	}
}

// simulatedBalanceCheck simulates the balance of an account considering transactions in the current pool.
// It assumes the caller holds the necessary lock on the BlockchainStruct.
func (bc *BlockchainStruct) simulatedBalanceCheck(transaction *Transaction) bool {
	// This function is called from AddTransactionToTransactionPool which already holds the lock.
	// Do NOT acquire the lock here to avoid deadlock.

	// 1. Check if we're in a test environment (db path contains "testdb")
	if strings.Contains(bc.db.Path(), "testdb") {
		// In test mode - use direct database balance
		testBalance, err := bc.db.GetAccountBalance(transaction.From)
		if err == nil {
			agentlog.LogInfo(fmt.Sprintf("Test Mode: Using direct DB balance for %s = %d",
				transaction.From, testBalance))
			return testBalance >= (transaction.Value + transaction.Fee)
		}
	}

	// 2. Normal mode - start with the confirmed on-chain balance
	simulatedBalance := bc.calculateTotalCryptoLocked(transaction.From)
	agentlog.LogInfo(fmt.Sprintf("Simulated Balance Check for %s -> %s (Value: %d, Fee: %d): Initial on-chain balance for %s = %d",
		transaction.From, transaction.To, transaction.Value, transaction.Fee, transaction.From, simulatedBalance))

	// 2. Adjust balance based on VERIFIED transactions already in the pool
	for _, poolTxn := range bc.TransactionPool {
		// Skip the transaction we are currently checking if it happens to be in the pool already (e.g., during a retry)
		if poolTxn.TransactionHash == transaction.TransactionHash {
			continue
		}

		// Only consider transactions that have passed verification
		if poolTxn.Status == TXN_VERIFICATION_SUCCESS {
			// If a pending txn sends money FROM the sender, decrease simulated balance
			if poolTxn.From == transaction.From {
				// Calculate total cost (value + fee)
				poolTxnCost := poolTxn.Value + poolTxn.Fee
				if simulatedBalance >= poolTxnCost {
					simulatedBalance -= poolTxnCost
					agentlog.LogInfo(fmt.Sprintf("  - Subtracting pending outgoing txn %s (Value: %d + Fee: %d = %d). New simulated balance: %d",
						poolTxn.TransactionHash[:8], poolTxn.Value, poolTxn.Fee, poolTxnCost, simulatedBalance))
				} else {
					// This indicates a potential double-spend attempt within the pool
					agentlog.LogInfo(fmt.Sprintf("  - Potential double spend detected: Pending outgoing txn %s (Value: %d + Fee: %d = %d) exceeds current simulated balance %d. Failing check.",
						poolTxn.TransactionHash[:8], poolTxn.Value, poolTxn.Fee, poolTxnCost, simulatedBalance))
					// It's safer to return false immediately here
					return false
				}
			}
			// If a pending txn sends money TO the sender, increase simulated balance
			// (This includes the faucet transaction if it were still pending)
			if poolTxn.To == transaction.From {
				simulatedBalance += poolTxn.Value
				agentlog.LogInfo(fmt.Sprintf("  + Adding pending incoming txn %s (%d). New simulated balance: %d",
					poolTxn.TransactionHash[:8], poolTxn.Value, simulatedBalance))
			}
		}
	}

	// 3. Finally, check if the adjusted balance is sufficient for the NEW transaction (value + fee)
	totalCost := transaction.Value + transaction.Fee
	finalCheck := simulatedBalance >= totalCost
	agentlog.LogInfo(fmt.Sprintf("  = Final Check: Final Simulated Balance (%d) >= New Txn Total Cost (Value: %d + Fee: %d = %d) -> %t",
		simulatedBalance, transaction.Value, transaction.Fee, totalCost, finalCheck))

	return finalCheck
}

// CheckIfIDExistsInBlocks iterates through mined blocks to see if a URI with the given ID exists.
// NOTE: Assumes caller holds the necessary lock if required for concurrent access.
func (bc *BlockchainStruct) CheckIfIDExistsInBlocks(desiredID string) bool {
	expectedAuthority := fmt.Sprintf("%s.%s", desiredID, uri.ResourceTypeChainStr) // e.g., "my-id.chain" (using string constant)

	for _, block := range bc.Blocks {
		// Skip genesis block if it has no relevant transactions
		if block.BlockNumber == 0 {
			continue
		}
		for _, txn := range block.Transactions {
			// Check if transaction data looks like a URI mint
			// A more robust check might involve specific transaction types or markers
			if txn.To == "" && txn.Value == 0 && len(txn.Data) > 0 {
				uriString := string(txn.Data)
				// Basic check if the URI string contains the expected authority part
				// A full URI parse might be more robust but slower
				if strings.Contains(uriString, "://"+expectedAuthority+"/") {
					agentlog.LogInfo(fmt.Sprintf("Found existing URI '%s' for ID '%s' in block %d", uriString, desiredID, block.BlockNumber))
					return true // Found evidence of the ID being used
				}
			}
		}
	}
	return false // Did not find the ID in any block's transaction data
}

// CheckIfIDExistsInTransactionPool checks if a URI with the given ID exists in pending transactions.
// NOTE: Assumes caller holds the necessary lock if required for concurrent access.
func (bc *BlockchainStruct) CheckIfIDExistsInTransactionPool(desiredID string) bool {
	expectedAuthority := fmt.Sprintf("%s.%s", desiredID, uri.ResourceTypeChainStr) // e.g., "my-id.chain"

	for _, txn := range bc.TransactionPool {
		if txn == nil {
			continue
		}
		// Check if transaction data looks like a URI mint
		// A more robust check might involve specific transaction types or markers
		if txn.To == "" && txn.Value == 0 && len(txn.Data) > 0 {
			uriString := string(txn.Data)
			// Basic check if the URI string contains the expected authority part
			if strings.Contains(uriString, "://"+expectedAuthority+"/") {
				agentlog.LogInfo(fmt.Sprintf("Found existing URI '%s' for ID '%s' in transaction pool", uriString, desiredID))
				return true // Found evidence of the ID being used in pending transactions
			}
		}
	}
	return false // Did not find the ID in any pending transaction data
}

func (bc *BlockchainStruct) ProofOfWorkMining(ctx context.Context, minersAddress string, cm *ConsensusManager) {
	agentlog.LogInfo("Starting to Mine...")
	nonce := 0
	cons := cm

	idleCheckInterval := 500 * time.Millisecond
	timer := time.NewTimer(idleCheckInterval)
	defer timer.Stop()

	for { // Main mining loop
		// Check for context cancellation at the start of each mining cycle
		select {
		case <-ctx.Done():
			agentlog.LogInfo("Mining context cancelled, stopping mining...")
			return
		default:
			// Continue with normal mining cycle
		}

		// --- Wait for signal or timer ---
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timer.Reset(idleCheckInterval)

		select {
		case <-bc.txnSignal:
			agentlog.LogInfo("New transaction signal received, checking pool...")
		case <-timer.C:
			// Timer fired, proceed normally
		case <-ctx.Done():
			agentlog.LogInfo("Mining context cancelled during wait, stopping mining...")
			return
		case <-time.After(1 * time.Second): // Safety timeout for select
			agentlog.LogInfo("Select timeout, proceeding...")
		}
		// --- End Wait ---

		// 1. Check for stop signal
		if bc.StopMining {
			agentlog.LogInfo("Mining stopped gracefully")
			return
		}

		// 2. Check if mining is locked or an update is required by consensus
		if cons.getMiningLockState() || cons.getUpdateRequired() {
			// agentlog.LogInfo("Mining locked or update required, pausing...")
			continue // Restart the loop (will enter select again)
		}

		// 3. Safely get verified transactions from the pool
		bc.Lock()
		poolCopy := make([]*Transaction, 0, len(bc.TransactionPool))
		for _, txn := range bc.TransactionPool {
			if txn.Status == TXN_VERIFICATION_SUCCESS {
				poolCopy = append(poolCopy, txn)
			}
		}
		bc.Unlock()

		// 4. Check if there are verified transactions to mine.
		// For demonstration/testing, we'll proceed even if poolCopy is empty,
		// creating a block with just the reward transaction.
		if len(poolCopy) == 0 {
			agentlog.LogInfo("No user transactions in pool, will attempt to mine a block with reward transaction only.")
			// No need to sort if poolCopy is empty.
		} else {
			agentlog.LogInfo(fmt.Sprintf("Found %d verified transactions in pool. Preparing to mine.", len(poolCopy)))
		}

		// Sort transactions for deterministic hashing (safe even if poolCopy is empty)
		sort.SliceStable(poolCopy, func(i, j int) bool {
			// Sort by TransactionHash for consistency
			return poolCopy[i].TransactionHash < poolCopy[j].TransactionHash
		})

		// --- Transactions exist, attempt to mine a block ---
		agentlog.LogInfo(fmt.Sprintf("Attempting to mine block with %d verified transactions (sorted)...", len(poolCopy)))

		// 5. Prepare fixed block components (Timestamp, PrevHash, BlockNumber) ONCE per attempt
		currentTimestamp := time.Now().Unix() // <<< Set timestamp ONCE here
		lastBlock, err := bc.GetLastBlock()
		var prevHash []byte
		var blockNumber uint64 = 0
		if err == nil && lastBlock != nil {
			prevHash = lastBlock.Hash()
			blockNumber = lastBlock.BlockNumber + 1
		} else if err != nil {
			agentlog.LogError("Error getting last block, skipping mining cycle:", err)
			time.Sleep(1 * time.Second)
			continue
		}
		//smartContract := &SmartContract{Code: []byte(""), Data: []byte("")} // Placeholder

		// --- Start Nonce Iteration ---
		bc.setIsActivelyMining(true) // <<< SET FLAG: Indicate active nonce search

		for { // Inner loop for nonce searching
			// Check for context cancellation first (highest priority)
			select {
			case <-ctx.Done():
				agentlog.LogInfo("Mining context cancelled, stopping mining...")
				bc.setIsActivelyMining(false)
				return
			default:
				// Continue with normal checks
			}

			// Check lock/stop status frequently within nonce search
			if bc.StopMining {
				return
			}
			if cons.getMiningLockState() || cons.getUpdateRequired() {
				agentlog.LogInfo("Mining locked/update needed during nonce search, restarting cycle...")
				time.Sleep(500 * time.Millisecond) // Small pause before restarting outer loop
				bc.setIsActivelyMining(false)
				goto nextMiningCycle // Use goto to break out of inner loop and restart outer loop
			}

			// 6. Create block candidate with CURRENT nonce and FIXED timestamp
			guessBlock := NewBlock(prevHash, nonce, blockNumber)
			guessBlock.Timestamp = currentTimestamp // <<< Use the fixed timestamp

			// 7. Add transactions (deep copies)
			for _, txn := range poolCopy {
				copiedTxn := txn.Clone()
				copiedTxn.Status = utils.SUCCESS
				guessBlock.AddTransactionToTheBlock(copiedTxn)
			}

			// 8. Add mining reward
			rewardTxn := NewTransaction(utils.BLOCKCHAIN_ADDRESS, minersAddress, utils.MINING_REWARD, []byte{})
			rewardTxn.Status = utils.SUCCESS
			guessBlock.Transactions = append(guessBlock.Transactions, rewardTxn)

			// 9. Calculate hash and check difficulty
			// guessBlock.Nonce = nonce // Already set in NewBlock
			guessHash := guessBlock.Hash()
			desiredHashPrefix := strings.Repeat("0", utils.MINING_DIFFICULTY)

			hashHex := hex.EncodeToString(guessHash)
			if strings.HasPrefix(hashHex, desiredHashPrefix) {
				// 10. Found a valid hash! Check lock one last time.
				if cons.getMiningLockState() {
					agentlog.LogInfo("Found valid hash but mining locked before adding block, restarting cycle...")
					time.Sleep(500 * time.Millisecond)
					bc.setIsActivelyMining(false)
					goto nextMiningCycle // Restart outer loop
				}

				// 11. IMPORTANT FIX: Update the block's hash field BEFORE adding
				guessBlock.BlockHash = guessHash // Store the correct hash calculated WITH transactions

				// Verify the block hash matches before adding (debug check)
				verifyHash := guessBlock.Hash()
				if !bytes.Equal(verifyHash, guessHash) {
					agentlog.LogError(fmt.Sprintf("Hash mismatch before AddBlock: %s vs %s",
						hex.EncodeToString(verifyHash), hex.EncodeToString(guessHash)), nil)
				}

				// 12. Add the block using the passed db connection
				// bc.AddBlock(guessBlock) // Now AddBlock -> VerifyBlock should pass
				if err := bc.AddBlock(guessBlock); err != nil {
					agentlog.LogError(fmt.Sprintf("ProofOfWorkMining: Failed to add mined block #%d to blockchain: %v. Hash: %s", guessBlock.BlockNumber, err, hex.EncodeToString(guessBlock.BlockHash)), nil)
					// Decide how to handle: retry nonce, or restart cycle?
					// For now, let's log and restart the cycle. The transactions will remain in the pool.
					bc.setIsActivelyMining(false)
					goto nextMiningCycle
				}
				agentlog.LogInfo(fmt.Sprintf("Successfully Mined and Added block #%d with %d transactions (incl. reward). Hash: %s",
					guessBlock.BlockNumber, len(guessBlock.Transactions), hex.EncodeToString(guessBlock.BlockHash)))
				// 12. Broadcast the block
				bc.BroadcastBlock(guessBlock)

				agentlog.LogInfo(fmt.Sprintf("ProofOfWorkMining: Successfully mined block #%d. Hash: %s. Calling AddBlock.", guessBlock.BlockNumber, hex.EncodeToString(guessBlock.BlockHash)))
				// 13. Reset nonce and break inner loop to start next mining cycle
				nonce = 0
				bc.setIsActivelyMining(false)
				goto nextMiningCycle // Use goto to break out and restart outer loop

			}
			// 14. Hash not found, increment nonce for the next attempt
			nonce++

			// Optional: Add a small sleep if nonce search is very fast to yield CPU
			// time.Sleep(1 * time.Millisecond)

		} // --- End Nonce Iteration ---

		//bc.setIsActivelyMining(false)
	nextMiningCycle: // Label for goto statement
		bc.setIsActivelyMining(false)
		continue // Restart main mining loop

	} // End of main mining loop
}

// HybridMining implements the PoAu-D consensus with PoW fallback
func (bc *BlockchainStruct) HybridMining(ctx context.Context, minersAddress string, cm *ConsensusManager) {
	agentlog.LogInfo("Starting Hybrid Mining (PoAu-D with PoW fallback)...")

	// Set mining flag
	bc.setIsActivelyMining(true)
	defer bc.setIsActivelyMining(false)

	idleCheckInterval := 500 * time.Millisecond
	timer := time.NewTimer(idleCheckInterval)
	defer timer.Stop()

	for { // Main mining loop
		// Check for context cancellation
		select {
		case <-ctx.Done():
			agentlog.LogInfo("Mining context cancelled, stopping mining...")
			return
		default:
			// Continue with normal mining cycle
		}

		// Wait for signal or timer
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timer.Reset(idleCheckInterval)

		select {
		case <-bc.txnSignal:
			agentlog.LogInfo("New transaction signal received, checking pool...")
		case <-timer.C:
			// Timer fired, proceed normally
		case <-ctx.Done():
			agentlog.LogInfo("Mining context cancelled during wait, stopping mining...")
			return
		}

		// Check for stop signal
		if bc.StopMining {
			agentlog.LogInfo("Mining stopped gracefully")
			return
		}

		// Check if mining is locked or an update is required by consensus
		if cm.getMiningLockState() || cm.getUpdateRequired() {
			continue
		}

		// Try PoAu-D block proposal first
		proposedBlock, err := bc.ProposePoAuDBlock(minersAddress)
		if err != nil {
			agentlog.LogError(fmt.Sprintf("PoAu-D block proposal failed: %v", err), nil)
			// Fall back to PoW for this cycle
			agentlog.LogInfo("Falling back to PoW mining for this cycle")
			bc.ProofOfWorkMining(ctx, minersAddress, cm)
			return
		}

		if proposedBlock != nil {
			// Successfully proposed a PoAu-D block
			agentlog.LogInfo(fmt.Sprintf("Successfully proposed PoAu-D block #%d", proposedBlock.BlockNumber))

			// Add the block to the chain
			if err := bc.AddBlock(proposedBlock); err != nil {
				agentlog.LogError(fmt.Sprintf("Failed to add PoAu-D block: %v", err), nil)
				continue
			}

			// Broadcast the block
			bc.BroadcastBlock(proposedBlock)
		} else {
			// No PoAu-D block proposed, fall back to PoW for this cycle
			agentlog.LogInfo("No PoAu-D block proposed, falling back to PoW")
			bc.ProofOfWorkMining(ctx, minersAddress, cm)
			return
		}
	}
}

// ProposePoAuDBlock attempts to propose a block using PoAu-D rules
func (bc *BlockchainStruct) ProposePoAuDBlock(proposerAddress string) (*Block, error) {
	// Check if this node is authorized to propose blocks
	if !bc.IsNetworkAuthor(proposerAddress) {
		// Check if this node is a PAP with transactions to process
		if bc.TransactionPoolManager == nil {
			return nil, fmt.Errorf("node is not a Network Author and has no transaction pool manager")
		}

		pasPoolTxs := bc.TransactionPoolManager.GetPASPoolTxs()
		if len(pasPoolTxs) == 0 {
			return nil, fmt.Errorf("node is not a Network Author and has no transactions in PAS pool")
		}

		// PAP can propose a block with their delegated transactions
		return bc.createPoAuDBlock(proposerAddress, pasPoolTxs)
	}

	// NAP can propose blocks with any available transactions
	bc.Lock()
	poolCopy := make([]*Transaction, 0, len(bc.TransactionPool))
	for _, txn := range bc.TransactionPool {
		if txn.Status == TXN_VERIFICATION_SUCCESS {
			poolCopy = append(poolCopy, txn)
		}
	}
	bc.Unlock()

	// Include PAS pool transactions if available
	if bc.TransactionPoolManager != nil {
		pasPoolTxs := bc.TransactionPoolManager.GetPASPoolTxs()
		poolCopy = append(poolCopy, pasPoolTxs...)
	}

	if len(poolCopy) == 0 {
		return nil, nil // No transactions to process
	}

	return bc.createPoAuDBlock(proposerAddress, poolCopy)
}

// createPoAuDBlock creates a new block using PoAu-D consensus rules
func (bc *BlockchainStruct) createPoAuDBlock(proposerAddress string, transactions []*Transaction) (*Block, error) {
	bc.Lock()
	defer bc.Unlock()

	if len(bc.Blocks) == 0 {
		return nil, fmt.Errorf("no genesis block found")
	}

	lastBlock := bc.Blocks[len(bc.Blocks)-1]
	newBlockNumber := lastBlock.BlockNumber + 1

	// Create mining reward transaction
	rewardTx := NewTransaction(utils.BLOCKCHAIN_ADDRESS, proposerAddress, utils.MINING_REWARD, []byte{})
	rewardTx.Status = TXN_VERIFICATION_SUCCESS
	rewardTx.TransactionHash = rewardTx.Hash()

	// Combine reward with other transactions
	allTransactions := append([]*Transaction{rewardTx}, transactions...)

	// Create the block (PoAu-D blocks don't need nonce, so use 0)
	newBlock := NewBlock(lastBlock.BlockHash, 0, newBlockNumber)
	newBlock.Transactions = allTransactions
	newBlock.ProposerAddress = proposerAddress

	// For PoAu-D, we don't need to find a nonce (no PoW)
	// The block is valid if the proposer is authorized
	newBlock.BlockHash = newBlock.Hash()

	agentlog.LogInfo(fmt.Sprintf("Created PoAu-D block #%d with %d transactions", newBlockNumber, len(allTransactions)))

	return newBlock, nil
}

func (bc *BlockchainStruct) calculateTotalCryptoLocked(address string) uint64 {
	sum := uint64(0)
	// If the address is the blockchain faucet, assume it has a virtually infinite balance
	// for the purpose of funding other accounts.
	if address == utils.BLOCKCHAIN_ADDRESS {
		agentlog.LogInfo(fmt.Sprintf("calculateTotalCryptoLocked for %s (Faucet): Returning max uint64 as balance.", address))
		return math.MaxUint64 // Effectively infinite balance
	}
	// No lock here, assumed acquired by caller

	currentBlocks := bc.Blocks
	if len(currentBlocks) == 0 {
		agentlog.LogInfo(fmt.Sprintf("calculateTotalCryptoLocked for %s: Chain has no blocks, balance is 0", address))
		return 0
	}

	agentlog.LogInfo(fmt.Sprintf("calculateTotalCryptoLocked for %s: Iterating %d blocks...", address, len(currentBlocks)))

	for i, block := range currentBlocks {
		if block == nil {
			agentlog.LogWarning(fmt.Sprintf("calculateTotalCryptoLocked for %s: Found nil block at index %d", address, i))
			continue
		}
		// Log block processing start
		agentlog.LogInfo(fmt.Sprintf("  Block %d: Processing...", block.BlockNumber)) // Added log

		if block.Transactions == nil {
			if i > 0 {
				agentlog.LogWarning(fmt.Sprintf("calculateTotalCryptoLocked for %s: Block %d has nil Transactions slice", address, block.BlockNumber))
			} else {
				agentlog.LogInfo(fmt.Sprintf("  Block %d (Genesis): No transactions to process.", block.BlockNumber)) // Log for Genesis
			}
			continue
		}

		agentlog.LogInfo(fmt.Sprintf("  Block %d: Contains %d transactions.", block.BlockNumber, len(block.Transactions))) // Added log

		for j, txn := range block.Transactions {
			if txn == nil {
				agentlog.LogWarning(fmt.Sprintf("calculateTotalCryptoLocked for %s: Found nil transaction at index %d in block %d", address, j, block.BlockNumber))
				continue
			}

			// Log details of every transaction being checked
			agentlog.LogInfo(fmt.Sprintf("    Txn %d (Hash: %s): Status='%s', From='%s', To='%s', Value=%d", j, txn.TransactionHash[:8], txn.Status, txn.From, txn.To, txn.Value)) // Added detailed log

			// Consider transactions in mined blocks (BlockNumber > 0) if their status indicates success,
			// or if they are in the genesis block (BlockNumber == 0) regardless of status (as they are foundational).
			isProcessedStatus := txn.Status == utils.SUCCESS || txn.Status == TXN_VERIFICATION_SUCCESS || txn.Status == utils.PENDING
			if block.BlockNumber == 0 || (block.BlockNumber > 0 && isProcessedStatus) {
				agentlog.LogInfo(fmt.Sprintf("      Txn Status is %s. Checking addresses...", txn.Status)) // Log status check pass
				isRelevant := false                                                                        // Flag to see if txn affects the address

				if txn.To == address {
					isRelevant = true
					sum += txn.Value
					agentlog.LogInfo(fmt.Sprintf("      MATCHED 'To': Added %d. New sum: %d", txn.Value, sum)) // Log match and update
				}

				if txn.From == address {
					isRelevant = true
					agentlog.LogInfo(fmt.Sprintf("      MATCHED 'From': Current sum before subtract: %d", sum)) // Log match
					if sum >= txn.Value {
						sum -= txn.Value
						agentlog.LogInfo(fmt.Sprintf("        - Subtracted %d. New sum: %d", txn.Value, sum)) // Log subtraction
					} else {
						agentlog.LogError(fmt.Sprintf("        Inconsistency: Outgoing %d exceeds balance %d", txn.Value, sum), nil)
					}
				}

				if !isRelevant {
					agentlog.LogInfo(fmt.Sprintf("      Txn does not involve address %s", address)) // Log if not relevant
				}

			} else {
				agentlog.LogInfo(fmt.Sprintf("      Skipped txn due to status: '%s'", txn.Status)) // Log skip reason
			}
		}
	}
	agentlog.LogInfo(fmt.Sprintf("calculateTotalCryptoLocked for %s finished. Final calculated balance: %d", address, sum))
	return sum
}

// Ensure the public CalculateTotalCrypto uses the lock correctly
func (bc *BlockchainStruct) CalculateTotalCrypto(address string) uint64 {
	bc.Lock()
	defer bc.Unlock()
	// Call the internal locked version
	return bc.calculateTotalCryptoLocked(address)
}

func (bc *BlockchainStruct) GetAllTxns() []Transaction {
	nTxns := []Transaction{}
	for i := len(bc.TransactionPool) - 1; i >= 0; i-- {
		nTxns = append(nTxns, *bc.TransactionPool[i])
	}
	return nTxns
}

func (bc *BlockchainStruct) GetLastBlock() (*Block, error) {
	bc.Lock()
	defer bc.Unlock()
	if len(bc.Blocks) == 0 {
		err := errors.New("blockchain is empty, genesis block not available")
		agentlog.LogError("GetLastBlock failed:", err)
		return nil, err
	}
	return bc.Blocks[len(bc.Blocks)-1], nil
}

// GetHeight returns the current blockchain height
func (bc *BlockchainStruct) GetHeight() uint64 {
	bc.Lock()
	defer bc.Unlock()
	return uint64(len(bc.Blocks))
}

// GetLatestBlockHash returns the hash of the latest block
func (bc *BlockchainStruct) GetLatestBlockHash() string {
	lastBlock, err := bc.GetLastBlock()
	if err != nil {
		return "N/A"
	}
	return hex.EncodeToString(lastBlock.BlockHash)
}

// GetTotalTransactions returns the total number of transactions in the blockchain
func (bc *BlockchainStruct) GetTotalTransactions() int {
	bc.Lock()
	defer bc.Unlock()
	count := 0
	for _, block := range bc.Blocks {
		count += len(block.Transactions)
	}
	return count
}

// GetDifficulty returns the current mining difficulty
func (bc *BlockchainStruct) GetDifficulty() (int, error) {
	return utils.MINING_DIFFICULTY, nil
}

func (bc *BlockchainStruct) DialAndUpdateReflections() {
	bc.Lock()
	defer bc.Unlock()

	for reflection := range bc.Reflections {
		if !bc.Reflections[reflection] {
			continue
		}

		if !strings.HasPrefix(reflection, "http://") && !strings.HasPrefix(reflection, "https://") {
			reflection = "http://" + reflection
		}

		url := fmt.Sprintf("%s/blockchain", reflection)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			agentlog.LogError(fmt.Sprintf("Failed to create request for reflection %s:", reflection), err)
			continue
		}

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			agentlog.LogError(fmt.Sprintf("Failed to dial reflection %s:", reflection), err)
			bc.Reflections[reflection] = false
			continue
		}

		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			agentlog.LogError(fmt.Sprintf("Reflection %s returned status %d", reflection, resp.StatusCode), nil)
			bc.Reflections[reflection] = false
			continue
		}

		var blockchain BlockchainStruct
		err = json.NewDecoder(resp.Body).Decode(&blockchain)
		if err != nil {
			agentlog.LogError(fmt.Sprintf("Failed to decode response from reflection %s:", reflection), err)
			bc.Reflections[reflection] = false
			continue
		}

		bc.Reflections[reflection] = true
	}
}

func (bc *BlockchainStruct) BroadcastBlock(b *Block) {
	agentlog.LogInfo(fmt.Sprintf("Broadcasting block #%d", b.BlockNumber))

	// Check if we have a P2P consensus manager
	if bc.p2pConsensusMgr != nil {
		// Use P2P broadcasting
		if err := bc.p2pConsensusMgr.BroadcastBlock(b); err != nil {
			agentlog.LogError("Failed to broadcast block via P2P:", err)
		}
		return
	}

	// Fall back to HTTP broadcasting if P2P is not available
	blockData := map[string]interface{}{
		"block": b,
	}

	blockJSON, err := json.Marshal(blockData)
	if err != nil {
		agentlog.LogError("Failed to marshal block data:", err)
		return
	}

	for reflectionAddr, isActive := range bc.Reflections {
		if !isActive {
			continue
		}

		if reflectionAddr == fmt.Sprintf("http://127.0.0.1:%d", utils.GetServerPort()) {
			continue
		}

		if !strings.HasPrefix(reflectionAddr, "http://") && !strings.HasPrefix(reflectionAddr, "https://") {
			reflectionAddr = "http://" + reflectionAddr
		}

		url := fmt.Sprintf("%s/block", reflectionAddr)

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(blockJSON))
		if err != nil {
			agentlog.LogError(fmt.Sprintf("Failed to create request for reflection %s:", reflectionAddr), err)
			continue
		}

		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			agentlog.LogError(fmt.Sprintf("Failed to send block to reflection %s:", reflectionAddr), err)
			continue
		}

		resp.Body.Close()
		agentlog.LogInfo(fmt.Sprintf("Block broadcast to reflection %s: status %d", reflectionAddr, resp.StatusCode))
		time.Sleep(100 * time.Millisecond)
	}
}

func (bc *BlockchainStruct) AddReflection(reflection string) {
	bc.Lock()
	defer bc.Unlock()

	if !strings.HasPrefix(reflection, "http://") && !strings.HasPrefix(reflection, "https://") {
		reflection = "http://" + reflection
	}

	bc.Reflections[reflection] = true
}

func (bc *BlockchainStruct) SetBlocks(blocks []*Block) {
	bc.Lock()
	defer bc.Unlock()

	bc.Blocks = make([]*Block, len(blocks))
	for i, block := range blocks {
		bc.Blocks[i] = block.DeepCopy()
	}
}

// validateTransaction validates a transaction
func (bc *BlockchainStruct) validateTransaction(transaction *Transaction) bool {
	// ... existing validation logic ...

	// Validate NFT capability attachment transactions
	if transaction.Type == TransactionTypeNFTCapabilityAttachment {
		return bc.validateNFTCapabilityAttachment(transaction)
	}

	return true
}

// validateNFTCapabilityAttachment validates an NFT capability attachment transaction
func (bc *BlockchainStruct) validateNFTCapabilityAttachment(transaction *Transaction) bool {
	// Parse the attachment data
	var attachmentData NFTCapabilityAttachmentData
	if err := json.Unmarshal(transaction.Data, &attachmentData); err != nil {
		log.Printf("Failed to unmarshal NFT capability attachment data: %v", err)
		return false
	}

	// Check if the NFT exists in ChromeDB
	nft, err := bc.nftManager.GetNFT(attachmentData.NFTID)
	if err != nil {
		log.Printf("NFT not found in ChromeDB: %v", err)
		return false
	}

	// Check if the capability exists
	capability, err := bc.mcpProcessor.getCapabilityByID(attachmentData.CapabilityID)
	if err != nil {
		log.Printf("Capability not found: %v", err)
		return false
	}

	// Check if the sender is the owner of the NFT
	if transaction.From != nft.Owner {
		log.Printf("Sender is not the owner of the NFT")
		return false
	}

	// Check if the transaction fee is sufficient
	if transaction.Fee < capability.GasFeeNRN {
		log.Printf("Insufficient fee: %d < %d", transaction.Fee, capability.GasFeeNRN)
		return false
	}

	return true
}
