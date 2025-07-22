// /home/gperry/Documents/GitHub/KNIRVCHAIN_GO_Verifyer/blockchain/blockchain_struct.go
package blockchain

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	constants "KNIRVCHAIN_GO_Verifyer/constants"
	"KNIRVCHAIN_GO_Verifyer/types"

	"github.com/syndtr/goleveldb/leveldb"
)

type BlockchainStruct struct {
	TransactionPool []*types.Transaction `json:"transaction_pool"`
	Blocks          []*Block             `json:"block_chain"`
	Address         string               `json:"address"`
	MiningLocked    bool                 `json:"mining_locked"`
	Mutex           sync.Mutex           `json:"-"`
	ChainID         string               `json:"chain_id"`

	// Event callbacks for transaction and block broadcasting
	// These will be set by the P2P manager
	OnBlockMined       func(*Block) error             `json:"-"`
	OnTransactionAdded func(*types.Transaction) error `json:"-"`
}

// New getter methods for p2p integration
func (bc *BlockchainStruct) GetChainID() string {
	return bc.ChainID
}

func (bc *BlockchainStruct) Lock() {
	bc.Mutex.Lock()
}

func (bc *BlockchainStruct) Unlock() {
	bc.Mutex.Unlock()
}

func (bc *BlockchainStruct) GetBlocks() []*Block {
	return bc.Blocks
}

// IsActivelyMining returns true if the blockchain is currently mining
func (bc *BlockchainStruct) IsActivelyMining() bool {
	bc.Mutex.Lock()
	defer bc.Mutex.Unlock()
	return !bc.MiningLocked
}

// Duplicate methods removed - already defined above

// BroadcastTransaction broadcasts a transaction to the network
// This now uses the callback set by the P2P manager
func (bc *BlockchainStruct) BroadcastTransaction(transaction *types.Transaction) error {
	if bc.OnTransactionAdded != nil {
		return bc.OnTransactionAdded(transaction)
	}
	log.Println("Warning: OnTransactionAdded callback not set, transaction not broadcast")
	return nil
}

var mutex sync.Mutex

func NewBlockchain(genesisBlock Block, address string) *BlockchainStruct {
	// Generate a unique chain ID based on the genesis block hash
	chainID := genesisBlock.Hash()

	// Get the LevelDB instance
	db := GetLevelDBInstance()

	// Check if blockchain exists by looking for chain tip height
	tipHeight, err := db.GetChainTipHeight()
	if err != nil {
		// Log errors other than NotFound, but proceed if NotFound (means new DB)
		log.Printf("Error checking chain tip height: %v", err)
	}

	// If tipHeight > 0 (or even == 0 if genesis was saved), it means DB exists
	// A more robust check might be needed if GetChainTipHeight returns 0 for NotFound
	// Let's assume GetChainTipHeight returns 0, nil for a truly empty DB after genesis save fails
	// Or returns 0, ErrNotFound before genesis is saved.
	// If err is nil (meaning height was found, even if 0), load from DB.
	if err == nil {
		log.Printf("Loading existing blockchain from database (tip height: %d)", tipHeight)

		// Load blocks one by one starting from genesis up to tipHeight
		var blocks []*Block
		var height uint64 = 0
		for {
			// Load up to and including the tipHeight block
			block, getErr := db.GetBlock(height)
			if getErr != nil {
				if errors.Is(getErr, leveldb.ErrNotFound) && height > tipHeight {
					// We've loaded all blocks up to the known tip
					log.Printf("Finished loading blocks up to height %d", tipHeight)
					break
				}
				// Log other errors
				log.Printf("Error loading block %d: %v. Stopping load.", height, getErr)
				// Decide how to handle partial load - maybe return error or truncated chain?
				// For now, let's break and use what we have.
				break
			}
			blocks = append(blocks, block)
			if height == tipHeight {
				// Stop after loading the tip block
				log.Printf("Loaded tip block %d", tipHeight)
				break
			}
			height++
		}

		// Load transaction pool
		pool, poolErr := GetTransactionPool()
		if poolErr != nil {
			log.Printf("Error loading transaction pool: %v", poolErr)
			pool = []*types.Transaction{} // Default to empty pool on error
		}

		if len(blocks) > 0 {
			log.Printf("Successfully loaded blockchain from database with %d blocks", len(blocks))
			log.Printf("Last block number: %d, Hash: %s",
				blocks[len(blocks)-1].Number,
				blocks[len(blocks)-1].Hash())
			log.Printf("Transaction pool has %d pending transactions", len(pool))

			// Verify the loaded chain (optional but recommended)
			if !VerifyChain(blocks) { // Use standalone VerifyChain function
				log.Println("CRITICAL: Loaded blockchain failed verification!")
				// Handle this critical error - maybe panic or return nil?
				return nil
			}

		} else {
			// This case might happen if DB exists but is corrupted/empty
			log.Println("Warning: Database exists but no blocks were loaded. Re-initializing with genesis.")
			// Fall through to create new blockchain logic below
			goto CreateNew // Use goto to jump to creation logic
		}

		return &BlockchainStruct{
			TransactionPool: pool,
			Blocks:          blocks,
			Address:         address,
			MiningLocked:    false,
			Mutex:           sync.Mutex{},
			ChainID:         chainID,
		}
	}

	// Label for goto
CreateNew:

	// Create new blockchain with genesis block
	log.Printf("Creating new blockchain")
	blockchainStruct := &BlockchainStruct{
		TransactionPool: []*types.Transaction{},
		Blocks:          []*Block{&genesisBlock}, // Start with genesis
		Address:         address,
		MiningLocked:    false,
		Mutex:           sync.Mutex{},
		ChainID:         chainID,
	}

	// Save genesis block and set as chain tip
	// Genesis block number should be 0
	if err := db.AtomicPutBlockAndUpdateTip(&genesisBlock); err != nil {
		log.Printf("Error saving genesis block: %v", err)
		// Consider returning nil or handling this more gracefully
		return nil
	}
	if err := PutTransactionPool([]*types.Transaction{}); err != nil {
		log.Printf("Error initializing empty transaction pool: %v", err)
		return nil
	}
	log.Println("New blockchain created and genesis block saved.")

	return blockchainStruct
}

// VerifyChain validates a complete blockchain (moved outside struct methods)
func VerifyChain(chain []*Block) bool {
	if len(chain) == 0 {
		log.Println("Verification failed: Chain is empty")
		return false
	}

	// Verify genesis block
	if chain[0].Number != 0 {
		log.Printf("Verification failed: Genesis block number is %d, expected 0", chain[0].Number)
		return false
	}
	// Note: Genesis PrevHash might be "" or "0x0" depending on initialization, adjust if needed
	// if chain[0].PrevHash != "" && chain[0].PrevHash != "0x0" {
	//  log.Printf("Verification failed: Genesis block PrevHash is '%s', expected empty or '0x0'", chain[0].PrevHash)
	//  return false
	// }

	// --- Start Change ---
	// REMOVED: Difficulty check for Genesis Block
	// The genesis block doesn't necessarily need to meet the current mining difficulty.
	// Its validity comes from being the agreed-upon starting point.
	/*
	   if !chain[0].VerifyHash(constants.MINING_DIFFICULTY) { // Verify genesis hash too
	           log.Printf("Verification failed: Genesis block hash %s invalid for difficulty %d", chain[0].Hash(), constants.MINING_DIFFICULTY)
	           return false
	   }
	*/
	log.Println("Skipping difficulty verification for Genesis block (Block 0).")

	// Verify each subsequent block
	for i := 1; i < len(chain); i++ {
		prevBlock := chain[i-1]
		currentBlock := chain[i]

		// Check block number sequence
		if currentBlock.Number != prevBlock.Number+1 {
			log.Printf("Verification failed: Block %d number mismatch (prev %d)", currentBlock.Number, prevBlock.Number)
			return false
		}

		// Check previous hash matches
		if currentBlock.PreviousHash != prevBlock.Hash() {
			log.Printf("Verification failed: Block %d PrevHash '%s' does not match previous block hash '%s'", currentBlock.Number, currentBlock.PreviousHash, prevBlock.Hash())
			return false
		}

		// Verify block hash meets difficulty
		if !currentBlock.VerifyHash() {
			log.Printf("Verification failed: Block %d hash %s invalid for difficulty %d", currentBlock.Number, currentBlock.Hash(), constants.MINING_DIFFICULTY)
			return false
		}
	}
	log.Printf("Chain verification successful for %d blocks.", len(chain))
	return true
}

// --- Rest of blockchain_struct.go ---
// ... (NewBlockchainFromSync, ToJson, AddBlock, etc. remain the same) ...

func NewBlockchainFromSync(Blocks []*Block, address string) *BlockchainStruct {
	// 1. Convert Block to Block: Deep copy is essential to avoid modification issues
	blocks := make([]*Block, len(Blocks))
	for i, rb := range Blocks {
		// Create a new Block instance
		block := &Block{
			Number:       rb.Number,
			Nonce:        rb.Nonce,
			PreviousHash: rb.PreviousHash,
			Time:         rb.Time,
			// Deep copy transactions as well
			Txs: make([]*types.Transaction, len(rb.Txs)),
		}
		for j, txn := range rb.Txs {
			// Create a new Transaction instance
			txnCopy := &types.Transaction{
				From:      txn.From,
				To:        txn.To,
				Value:     txn.Value,
				Data:      txn.Data,
				Status:    txn.Status,
				Timestamp: txn.Timestamp,
				PublicKey: txn.PublicKey,
				Signature: txn.Signature, // Assuming signature is immutable or copied correctly
			}
			block.Txs[j] = txnCopy
		}
		blocks[i] = block
	}

	return &BlockchainStruct{
		Blocks:       blocks,
		Address:      address,
		MiningLocked: false,
		Mutex:        sync.Mutex{},
		ChainID:      blocks[0].Hash(), // Use the genesis block hash as chain ID
	}
}

func (bc *BlockchainStruct) ToJson() string {
	nb, err := json.Marshal(bc)

	if err != nil {
		return err.Error()
	} else {
		return string(nb)
	}
}

func (bc *BlockchainStruct) AddBlock(db *LevelDB, b *Block) {
	bc.Mutex.Lock()
	defer bc.Mutex.Unlock()

	m := map[string]bool{}
	for _, txn := range b.Txs {
		m[txn.Hash()] = true
	}

	// remove txn from txn pool
	newTxnPool := []*types.Transaction{}
	for _, txn := range bc.TransactionPool {
		_, ok := m[txn.Hash()]
		if !ok {
			newTxnPool = append(newTxnPool, txn)
		}
	}

	bc.TransactionPool = newTxnPool

	bc.Blocks = append(bc.Blocks, b)

	// Log block information before saving
	log.Printf("Adding block #%d with hash %s to blockchain", b.Number, b.Hash())
	log.Printf("Blockchain now has %d blocks", len(bc.Blocks))
	log.Printf("Saving blockchain to database at path: %s", constants.BLOCKCHAIN_DB_PATH)

	// Save block and update chain tip atomically using the provided LevelDB instance
	if err := db.AtomicPutBlockAndUpdateTip(b); err != nil {
		log.Printf("Error saving block to database: %v", err)
	}

	// Save updated transaction pool
	// TODO: Update this to use the LevelDB instance as well
	if err := PutTransactionPool(bc.TransactionPool); err != nil {
		log.Printf("Error saving transaction pool: %v", err)
	}

	log.Printf("Block #%d added successfully", b.Number)
}

// AddTransactionToTransactionPool adds a transaction to the transaction pool
// and broadcasts it to the network
func (bc *BlockchainStruct) AddTransactionToTransactionPool(transaction *types.Transaction) error {
	bc.Mutex.Lock()
	defer bc.Mutex.Unlock()

	// Check if transaction already exists in the pool
	for _, txn := range bc.TransactionPool {
		if txn.Hash() == transaction.Hash() {
			return nil // Transaction already exists, no error
		}
	}

	log.Println("Adding txn to the Transaction pool")

	// Verify the transaction
	valid := transaction.VerifyTxn()

	// Set transaction status based on verification
	if valid {
		transaction.Status = constants.TXN_VERIFICATION_SUCCESS
	} else {
		transaction.Status = constants.TXN_VERIFICATION_FAILURE
		log.Printf("Transaction %s failed verification", transaction.Hash())
		return fmt.Errorf("transaction failed verification")
	}

	// Clear public key for storage
	transaction.PublicKey = ""

	// Add to transaction pool
	bc.TransactionPool = append(bc.TransactionPool, transaction)

	// Save transaction pool
	if err := PutTransactionPool(bc.TransactionPool); err != nil {
		log.Printf("Error saving transaction pool: %v", err)
	}

	// Broadcast transaction to network
	bc.BroadcastLocalTransaction(transaction)

	return nil
}
func (bc *BlockchainStruct) BroadcastLocalTransaction(txn *types.Transaction) {
	log.Println("Broadcasting transaction:", txn.Hash())

	// Use the new BroadcastTransaction method that takes a types.Transaction
	if err := bc.BroadcastTransaction(txn); err != nil {
		log.Printf("Error broadcasting transaction: %v", err)
	}
}


func (bc *BlockchainStruct) MineNewBlock(minersAddress string) (*Block, error) {
	bc.MiningLocked = true // Lock mining during block creation

	defer func() { bc.MiningLocked = false }() // Unlock *always*, even if error

	// Ensure there's at least one block (genesis) before proceeding
	if len(bc.Blocks) == 0 {
		return nil, errors.New("cannot mine new block: blockchain is empty")
	}

	prevHash := bc.Blocks[len(bc.Blocks)-1].Hash()
	newBlock := NewBlock(prevHash, 0, uint64(len(bc.Blocks))) // nonce starts at 0

	// Deep copy transactions from the pool
	for _, txn := range bc.TransactionPool {
		// Create a new Transaction instance for the block
		newTxn := types.NewTransaction(txn.From, txn.To, txn.Value, txn.Data, txn.Origin)
		newTxn.Timestamp = txn.Timestamp
		newTxn.Status = txn.Status // Status should already be set (SUCCESS/FAILED)
		newTxn.Signature = txn.Signature
		newTxn.PublicKey = txn.PublicKey // Ensure public key is copied

		// Add only successful transactions to the block (optional, depends on rules)
		if newTxn.Status == constants.SUCCESS {
			if err := newBlock.AddTransactionToTheBlock(newTxn); err != nil {
				// This AddTransactionToTheBlock shouldn't fail if status is already set
				log.Printf("Warning: Error adding transaction %s to block: %v", newTxn.Hash(), err)
				// Continue adding other transactions? Or return error?
				// return nil, fmt.Errorf("failed to add transaction to block: %w", err)
			}
		} else {
			log.Printf("Skipping failed transaction %s from block mining", newTxn.Hash())
		}
	}

	rewardTxn := types.NewTransaction(constants.BLOCKCHAIN_ADDRESS, minersAddress, constants.MINING_REWARD, []byte{}, constants.ORIGIN_PUBLIC)
	rewardTxn.Status = constants.SUCCESS // Reward is always successful
	if err := newBlock.AddTransactionToTheBlock(rewardTxn); err != nil {
		return nil, fmt.Errorf("failed to add reward transaction: %w", err)
	}

	if err := newBlock.Mine(constants.MINING_DIFFICULTY); err != nil {
		return nil, fmt.Errorf("mining error: %w", err)
	}
	return newBlock, nil
}

func (bc *BlockchainStruct) ProofOfWorkMining(minersAddress string, stopMining <-chan bool, miningStopped chan<- bool) {
	log.Println("Starting mining process with miner's address:", minersAddress)
	blocksMined := 0
	startTime := time.Now()
	targetBlockTime := 10 * time.Second // Target 10 seconds per block
	lastBlockTime := time.Now()

	for {
		if bc.MiningLocked {
			log.Println("Mining is locked, waiting...")
			time.Sleep(constants.CONSENSUS_PAUSE_TIME * time.Second)
			continue
		}

		select {
		case <-stopMining:
			log.Println("Received stop mining signal. Mining stopped.")
			miningStopped <- true
			return

		default:
			// Calculate time since last block
			timeSinceLast := time.Since(lastBlockTime)
			if timeSinceLast < targetBlockTime {
				// Sleep for remaining time to hit target block time
				sleepTime := targetBlockTime - timeSinceLast
				// Reduce log frequency for waiting
				// log.Printf("Waiting %.2f seconds to maintain block time target", sleepTime.Seconds())
				time.Sleep(sleepTime)
				continue // Check stopMining channel again after sleep
			}

			txCount := len(bc.TransactionPool)
			if txCount == 0 {
				// Check if EMPTY_BLOCK_INTERVAL is defined and valid
				emptyBlockIntervalSeconds := constants.EMPTY_BLOCK_INTERVAL // Assuming this is int seconds
				if emptyBlockIntervalSeconds <= 0 {
					emptyBlockIntervalSeconds = 60 // Default to 60 seconds if invalid
				}

				if timeSinceLast < time.Duration(emptyBlockIntervalSeconds)*time.Second {
					log.Println("No transactions to mine, waiting...")
					time.Sleep(5 * time.Second) // Check stop channel more frequently
					continue
				}
				log.Println("Mining empty block after timeout")
			}

			log.Printf("Starting to mine a new block with %d transactions in the pool", txCount)

			newBlock, err := bc.MineNewBlock(minersAddress)
			if err != nil {
				log.Println("Error Mining Block: ", err)
				time.Sleep(5 * time.Second) // Add delay after error
				continue
			}

			// Re-check lock before adding block, consensus might have locked it
			if !bc.MiningLocked {
				db := GetLevelDBInstance()
				bc.AddBlock(db, newBlock) // AddBlock handles mutex, saving, pool cleanup
				blocksMined++
				lastBlockTime = time.Now()

				// Calculate mining statistics
				elapsedTotal := time.Since(startTime)
				var blocksPerHour float64
				if elapsedTotal.Hours() > 0 {
					blocksPerHour = float64(blocksMined) / elapsedTotal.Hours()
				} else {
					blocksPerHour = 0 // Avoid division by zero
				}

				log.Printf("BLOCK MINED: #%d with %d transactions",
					newBlock.Number, len(newBlock.Txs))
				log.Printf("Mining stats: %d blocks mined, %.2f blocks/hour",
					blocksMined, blocksPerHour)

				// Log reward information (find the reward transaction)
				for _, rewardTx := range newBlock.Txs {
					if rewardTx.From == constants.BLOCKCHAIN_ADDRESS {
						log.Printf("Mining reward of %d sent to %s", rewardTx.Value, rewardTx.To)
						// Assuming only one reward txn per block
						// Broadcast the reward transaction? Usually not needed unless peers verify rewards
						// bc.BroadcastLocalTransaction(rewardTx)
						break
					}
				}

			} else {
				log.Println("Mining was locked after block was mined but before adding. Discarding block.")
				// Block is discarded because consensus likely updated the chain
			}
		}
	}
}

func (bc *BlockchainStruct) CalculateTotalCrypto(address string) uint64 {
	sum := uint64(0)

	for _, blocks := range bc.Blocks {
		for _, txns := range blocks.Txs {
			if txns.Status == constants.SUCCESS {
				if txns.To == address {
					sum += txns.Value
				} else if txns.From == address {
					// Prevent underflow if balance calculation is wrong
					if sum >= txns.Value {
						sum -= txns.Value
					} else {
						log.Printf("Warning: Potential balance underflow for address %s (block %d)", address, blocks.Number)
						sum = 0 // Reset to 0 on underflow? Or handle differently?
					}
				}
			}
		}
	}
	return sum
}

func (bc *BlockchainStruct) GetAllTxns() []types.Transaction {

	nTxns := []types.Transaction{}

	// Add pending transactions first (most recent)
	for i := len(bc.TransactionPool) - 1; i >= 0; i-- {
		nTxns = append(nTxns, *bc.TransactionPool[i])
	}

	// Add confirmed transactions from blocks (most recent block first)
	txns := []types.Transaction{}
	for blockIdx := len(bc.Blocks) - 1; blockIdx >= 0; blockIdx-- {
		block := bc.Blocks[blockIdx]
		// Iterate transactions within block in reverse to keep order consistent?
		// Usually block order is sufficient.
		for txnIdx := len(block.Txs) - 1; txnIdx >= 0; txnIdx-- {
			txn := block.Txs[txnIdx]
			// Exclude reward transactions
			if txn.From != constants.BLOCKCHAIN_ADDRESS {
				txns = append(txns, *txn)
			}
		}
	}
	// Append confirmed transactions after pending ones
	nTxns = append(nTxns, txns...)

	return nTxns
}

// SyncChain replaces the current chain with a new valid chain
func (bc *BlockchainStruct) SyncChain(newChain []*Block) error {
	bc.Mutex.Lock()
	defer bc.Mutex.Unlock()

	// Lock mining during sync
	bc.MiningLocked = true
	defer func() { bc.MiningLocked = false }() // Ensure unlock

	if !VerifyChain(newChain) { // Use standalone VerifyChain
		return fmt.Errorf("invalid chain provided for sync")
	}

	log.Printf("Starting chain sync. New chain height: %d", len(newChain)-1)

	// Save all blocks to database atomically
	// It might be more efficient to batch writes if LevelDB supports it well,
	// but atomic block/tip updates are safer.
	for _, block := range newChain {
		if err := AtomicPutBlockAndUpdateTip(block); err != nil {
			// Rollback might be complex here. Log error and potentially stop sync.
			log.Printf("CRITICAL: Failed to save block %d during sync: %v", block.Number, err)
			return fmt.Errorf("failed to save block %d during sync: %w", block.Number, err)
		}
	}

	// Update in-memory chain
	bc.Blocks = newChain

	// Rebuild transaction pool excluding transactions already in blocks
	newPool := []*types.Transaction{}
	blockTxns := make(map[string]bool)
	for _, block := range newChain {
		for _, txn := range block.Txs {
			blockTxns[txn.Hash()] = true
		}
	}

	// Keep transactions from the *old* pool if they are not in the *new* chain
	for _, txn := range bc.TransactionPool { // Iterate over the pool *before* replacement
		if !blockTxns[txn.Hash()] {
			// Optional: Re-verify transaction against the new chain state if needed
			newPool = append(newPool, txn)
		}
	}

	bc.TransactionPool = newPool
	if err := PutTransactionPool(newPool); err != nil {
		// Log error but continue, as chain sync was successful
		log.Printf("Error saving updated transaction pool after sync: %v", err)
		// return fmt.Errorf("failed to save transaction pool after sync: %w", err)
	}

	log.Printf("Successfully synced chain to height %d. New pool size: %d", len(newChain)-1, len(newPool))
	return nil
}

func (bc *BlockchainStruct) AddTransaction(txn types.Transaction) error {
	bc.Mutex.Lock()
	defer bc.Mutex.Unlock()
	if bc.MiningLocked {
		return fmt.Errorf("mining is locked, cannot add transaction")
	}

	// Check if transaction already exists in pool or chain
	if _, exists := bc.findTransaction(txn.Hash()); exists {
		log.Printf("Transaction %s already exists, ignoring.", txn.Hash())
		return errors.New("transaction already exists")
	}

	if !txn.VerifyTxn() {
		log.Printf("Transaction %s verification failed.", txn.Hash())
		return fmt.Errorf("txn verification failed")
	}

	// Create a copy of the transaction to avoid potential issues with shared memory
	txnCopy := types.Transaction{
		From:      txn.From,
		To:        txn.To,
		Value:     txn.Value,
		Data:      txn.Data,
		Status:    constants.TXN_VERIFICATION_SUCCESS, // Mark as verified
		Timestamp: txn.Timestamp,
		PublicKey: txn.PublicKey,
		Signature: txn.Signature,
	}

	bc.TransactionPool = append(bc.TransactionPool, &txnCopy)
	log.Printf("Added transaction %s to pool. Pool size: %d", txnCopy.Hash(), len(bc.TransactionPool))

	// Save updated pool
	if err := PutTransactionPool(bc.TransactionPool); err != nil {
		log.Printf("Error saving transaction pool after adding txn %s: %v", txnCopy.Hash(), err)
		// Don't necessarily return error here, as txn is in memory pool
	}

	// Broadcast the transaction *after* adding it locally
	bc.BroadcastLocalTransaction(&txnCopy)

	return nil
}

// findTransaction searches for a transaction by hash in both the pool and blocks
func (bc *BlockchainStruct) findTransaction(hash string) (*types.Transaction, bool) {
	// Check pool first
	for _, txn := range bc.TransactionPool {
		if txn.Hash() == hash {
			return txn, true
		}
	}
	// Check blocks (might be slow for large chains, consider indexing if needed)
	for i := len(bc.Blocks) - 1; i >= 0; i-- {
		for _, txn := range bc.Blocks[i].Txs {
			if txn.Hash() == hash {
				return txn, true
			}
		}
	}
	return nil, false
}

// VerifyChain validates a complete blockchain
func (bc *BlockchainStruct) VerifyChain(chain []*Block) bool {
	if len(chain) == 0 {
		return false
	}

	// Verify genesis block
	if chain[0].Number != 0 || chain[0].PreviousHash != "" {
		return false
	}

	// Verify each subsequent block
	for i := 1; i < len(chain); i++ {
		prevBlock := chain[i-1]
		currentBlock := chain[i]

		// Check block number sequence
		if currentBlock.Number != prevBlock.Number+1 {
			return false
		}

		// Check previous hash matches
		if currentBlock.PreviousHash != prevBlock.Hash() {
			return false
		}

		// Verify block hash meets difficulty
		if !currentBlock.VerifyHash() {
			return false
		}
	}

	return true
}
