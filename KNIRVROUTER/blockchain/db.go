// /home/gperry/Documents/GitHub/KNIRVCHAIN_GO_Verifyer/blockchain/db.go
package blockchain

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"

	// "path/filepath" // Removed unused import
	"strconv" // Added missing import

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"

	constants "KNIRVCHAIN_GO_Verifyer/constants" // Ensure constants is imported
	"KNIRVCHAIN_GO_Verifyer/types"
)

const (
	blockPrefix = "block:"
	txnPrefix   = "txn:"
	// chainTipKey        = "chain_tip" // No longer used
	transactionPoolKey = "transaction_pool"
	chainTipHeightKey  = "chain_tip_height" // Use height key
)

// ensureDBDirectoryExists creates the database directory if it doesn't exist
func ensureDBDirectoryExists() error {
	// Use the path from the constants package
	dbDir := constants.BLOCKCHAIN_DB_PATH
	if dbDir == "" {
		// Add a fallback or log a critical error if the constant is somehow empty
		log.Println("CRITICAL: constants.BLOCKCHAIN_DB_PATH is empty, using default 'database'")
		dbDir = "database" // Fallback, but indicates a setup issue
	}
	log.Printf("Checking database directory: %s", dbDir)

	if _, err := os.Stat(dbDir); os.IsNotExist(err) {
		log.Printf("Creating database directory: %s", dbDir)
		// Use MkdirAll to create parent directories if needed
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			log.Printf("Error creating directory '%s': %v", dbDir, err)
			return fmt.Errorf("failed to create database directory '%s': %w", dbDir, err)
		}
		log.Printf("Database directory created successfully: %s", dbDir)
	} else if err != nil {
		log.Printf("Error checking directory '%s': %v", dbDir, err)
		return fmt.Errorf("failed to check database directory '%s': %w", dbDir, err)
	}
	return nil
}

// openDB opens the LevelDB database with proper error handling
func openDB() (*leveldb.DB, error) {
	if err := ensureDBDirectoryExists(); err != nil {
		// Error already logged in ensureDBDirectoryExists
		return nil, err // Return the wrapped error
	}

	// Use the path from the constants package
	dbPath := constants.BLOCKCHAIN_DB_PATH
	if dbPath == "" {
		log.Println("CRITICAL: constants.BLOCKCHAIN_DB_PATH is empty during openDB, using default 'database'")
		dbPath = "database" // Fallback
	}

	options := &opt.Options{
		ErrorIfMissing: false, // Create if missing
	}

	log.Printf("Attempting to open LevelDB at path: %s", dbPath) // Log the path being opened
	db, err := leveldb.OpenFile(dbPath, options)
	if err != nil {
		log.Printf("ERROR opening LevelDB at '%s': %v", dbPath, err)
		return nil, fmt.Errorf("failed to open database at '%s': %w", dbPath, err)
	}
	log.Printf("Successfully opened LevelDB at path: %s", dbPath)
	return db, nil
}

// --- Functions below use openDB() ---

// PutBlock stores a single block in the database
func PutBlock(block *Block) error {
	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Close()

	key := []byte(fmt.Sprintf("%s%d", blockPrefix, block.BlockNumber()))
	value, err := json.Marshal(block)
	if err != nil {
		return fmt.Errorf("failed to marshal block %d: %w", block.BlockNumber(), err)
	}

	log.Printf("Putting block %d into DB", block.BlockNumber())
	return db.Put(key, value, nil)
}

// GetBlock retrieves a single block by height
func GetBlock(height uint64) (*Block, error) {
	db, err := openDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	key := []byte(fmt.Sprintf("%s%d", blockPrefix, height))
	data, err := db.Get(key, nil)
	if err != nil {
		// Don't log ErrNotFound as an error here, let caller handle it
		if !errors.Is(err, leveldb.ErrNotFound) {
			log.Printf("Error getting block %d from DB: %v", height, err)
		}
		return nil, fmt.Errorf("failed to get block %d: %w", height, err) // Return wrapped error including ErrNotFound
	}

	var block Block
	if err := json.Unmarshal(data, &block); err != nil {
		log.Printf("Error unmarshalling block %d from DB: %v", height, err)
		return nil, fmt.Errorf("failed to unmarshal block %d: %w", height, err)
	}
	// Removed redundant log, GetBlock caller in NewBlockchain logs success
	// log.Printf("Retrieved block %d from DB", height)
	return &block, nil
}

// GetLatestBlock retrieves the latest block using the chain tip height
func GetLatestBlock() (*Block, error) {
	// Directly get the height of the tip block
	tipHeight, err := GetChainTipHeight()
	if err != nil {
		// Check if the error is NotFound, meaning the DB is likely empty or only has genesis
		if errors.Is(err, leveldb.ErrNotFound) || tipHeight == 0 { // Also check if height is 0 explicitly
			log.Println("Chain tip height not found or is 0, attempting to get genesis block (0)")
			genesis, getErr := GetBlock(0)
			if getErr != nil {
				// If genesis also not found, the DB is truly empty/corrupted
				return nil, errors.New("no chain tip available and failed to get genesis block")
			}
			return genesis, nil // Return genesis if found
		}
		// For other errors getting the tip height
		return nil, fmt.Errorf("failed to get chain tip height: %w", err)
	}

	log.Printf("Chain tip height is %d, retrieving block", tipHeight)
	// Get the block using the retrieved height
	latestBlock, err := GetBlock(tipHeight)
	if err != nil {
		log.Printf("Error retrieving latest block at height %d: %v", tipHeight, err)
		return nil, fmt.Errorf("failed to retrieve block at tip height %d: %w", tipHeight, err)
	}
	return latestBlock, nil
}

// UpdateChainTipHeight updates the chain tip height reference
func UpdateChainTipHeight(height uint64) error {
	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Close()

	heightBytes := []byte(fmt.Sprintf("%d", height))
	log.Printf("Updating chain tip height to: %d", height)
	return db.Put([]byte(chainTipHeightKey), heightBytes, nil)
}

// GetChainTipHeight retrieves the current chain tip height
func GetChainTipHeight() (uint64, error) {
	db, err := openDB()
	if err != nil {
		return 0, err
	}
	defer db.Close()

	data, err := db.Get([]byte(chainTipHeightKey), nil)
	if err != nil {
		// Return ErrNotFound explicitly if that's the error
		if errors.Is(err, leveldb.ErrNotFound) {
			log.Println("Chain tip height not found in DB")
			return 0, err // Return the original ErrNotFound
		}
		// Log and wrap other errors
		log.Printf("Error getting chain tip height from DB: %v", err)
		return 0, fmt.Errorf("failed to get chain tip height: %w", err)
	}

	height, err := strconv.ParseUint(string(data), 10, 64) // Use strconv
	if err != nil {
		log.Printf("Error parsing chain tip height '%s': %v", string(data), err)
		return 0, fmt.Errorf("failed to parse chain tip height: %w", err)
	}
	log.Printf("Retrieved chain tip height: %d", height)
	return height, nil
}

// --- Rest of db.go ---
// ... (PutTransaction, GetTransaction, PutTransactionPool, GetTransactionPool, AtomicPutBlockAndUpdateTip remain the same) ...

// PutTransaction stores a single transaction in the database
func PutTransaction(txn *types.Transaction) error {
	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Close()

	// Use transaction hash as key (ensure txn.Hash() is reliable)
	txnHash := txn.Hash() // Assuming txn.Hash() returns a unique string hash
	key := []byte(fmt.Sprintf("%s%s", txnPrefix, txnHash))
	value, err := json.Marshal(txn)
	if err != nil {
		return fmt.Errorf("failed to marshal transaction %s: %w", txnHash, err)
	}

	log.Printf("Putting transaction %s into DB", txnHash)
	return db.Put(key, value, nil)
}

// GetTransaction retrieves a transaction by hash
func GetTransaction(hash string) (*types.Transaction, error) {
	db, err := openDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	key := []byte(fmt.Sprintf("%s%s", txnPrefix, hash))
	data, err := db.Get(key, nil)
	if err != nil {
		if !errors.Is(err, leveldb.ErrNotFound) {
			log.Printf("Error getting transaction %s from DB: %v", hash, err)
		}
		return nil, fmt.Errorf("failed to get transaction %s: %w", hash, err)
	}

	var txn types.Transaction
	if err := json.Unmarshal(data, &txn); err != nil {
		log.Printf("Error unmarshalling transaction %s from DB: %v", hash, err)
		return nil, fmt.Errorf("failed to unmarshal transaction %s: %w", hash, err)
	}
	log.Printf("Retrieved transaction %s from DB", hash)
	return &txn, nil
}

// PutTransactionPool stores the current transaction pool
func PutTransactionPool(pool []*types.Transaction) error {
	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Close()

	value, err := json.Marshal(pool)
	if err != nil {
		return fmt.Errorf("failed to marshal transaction pool: %w", err)
	}

	log.Printf("Putting transaction pool (%d txns) into DB", len(pool))
	return db.Put([]byte(transactionPoolKey), value, nil)
}

// GetTransactionPool retrieves the current transaction pool
func GetTransactionPool() ([]*types.Transaction, error) {
	db, err := openDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	data, err := db.Get([]byte(transactionPoolKey), nil)
	if err != nil {
		if errors.Is(err, leveldb.ErrNotFound) {
			log.Println("Transaction pool not found in DB, returning empty pool")
			return []*types.Transaction{}, nil // Return empty slice if not found
		}
		log.Printf("Error getting transaction pool from DB: %v", err)
		return nil, fmt.Errorf("failed to get transaction pool: %w", err)
	}

	var pool []*types.Transaction
	if err := json.Unmarshal(data, &pool); err != nil {
		log.Printf("Error unmarshalling transaction pool from DB: %v", err)
		return nil, fmt.Errorf("failed to unmarshal transaction pool: %w", err)
	}
	log.Printf("Retrieved transaction pool (%d txns) from DB", len(pool))
	return pool, nil
}

// AtomicPutBlockAndUpdateTip atomically stores a block and updates the chain tip height
func AtomicPutBlockAndUpdateTip(block *Block) error {
	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Close()

	// Start a transaction
	txn, err := db.OpenTransaction()
	if err != nil {
		return fmt.Errorf("failed to start DB transaction: %w", err)
	}
	defer txn.Discard() // Ensure transaction is discarded if not committed

	// Prepare block data
	blockKey := []byte(fmt.Sprintf("%s%d", blockPrefix, block.BlockNumber()))
	blockValue, err := json.Marshal(block)
	if err != nil {
		// No need to discard here, defer handles it
		return fmt.Errorf("failed to marshal block %d: %w", block.BlockNumber(), err)
	}

	// Put block
	log.Printf("Atomically putting block %d into DB", block.BlockNumber())
	if err := txn.Put(blockKey, blockValue, nil); err != nil {
		return fmt.Errorf("failed to put block %d in transaction: %w", block.BlockNumber(), err)
	}

	// Update chain tip height
	heightBytes := []byte(fmt.Sprintf("%d", block.BlockNumber()))
	log.Printf("Atomically updating chain tip height to: %d", block.BlockNumber())
	if err := txn.Put([]byte(chainTipHeightKey), heightBytes, nil); err != nil {
		return fmt.Errorf("failed to update chain tip height in transaction: %w", err)
	}

	// Commit the transaction
	if err := txn.Commit(); err != nil {
		log.Printf("ERROR committing DB transaction for block %d: %v", block.BlockNumber(), err)
		return fmt.Errorf("failed to commit DB transaction: %w", err)
	}

	log.Printf("Successfully committed block %d and tip height to DB", block.BlockNumber())
	return nil
}
