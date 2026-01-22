package blockchain

import (
	"KNIRVROUTER/internal/types"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"sync"

	"github.com/syndtr/goleveldb/leveldb"
)

// Constants are defined in db.go to avoid duplication

var (
	levelDBInstance *LevelDB
	levelDBOnce     sync.Once
	levelDBMutex    sync.Mutex
	rawDBInstance   *leveldb.DB
	rawDBOnce       sync.Once
)

// GetLevelDBInstance returns a singleton instance of LevelDB
// This ensures that only one instance of LevelDB is used throughout the application
func GetLevelDBInstance() *LevelDB {
	levelDBOnce.Do(func() {
		var err error
		levelDBInstance, err = NewLevelDB()
		if err != nil {
			log.Fatalf("Failed to create LevelDB instance: %v", err)
		}
	})
	return levelDBInstance
}

// getRawDBInstance returns a singleton raw LevelDB instance
func getRawDBInstance() (*leveldb.DB, error) {
	var err error
	rawDBOnce.Do(func() {
		rawDBInstance, err = openDB()
	})
	return rawDBInstance, err
}

// NewLevelDB creates a new LevelDB instance using the singleton raw database
func NewLevelDB() (*LevelDB, error) {
	// Use the singleton raw database instance
	db, err := getRawDBInstance()
	if err != nil {
		return nil, err
	}

	// Create a new LevelDB instance
	return &LevelDB{
		db: db,
	}, nil
}

// LevelDB represents a LevelDB database instance
type LevelDB struct {
	db *leveldb.DB
	mu sync.Mutex
}

// AtomicPutBlockAndUpdateTip atomically stores a block and updates the chain tip height
func (ldb *LevelDB) AtomicPutBlockAndUpdateTip(block *Block) error {
	ldb.mu.Lock()
	defer ldb.mu.Unlock()

	// Start a transaction
	txn, err := ldb.db.OpenTransaction()
	if err != nil {
		return fmt.Errorf("failed to start DB transaction: %w", err)
	}
	defer txn.Discard() // Ensure transaction is discarded if not committed

	// Prepare block data
	blockKey := []byte(fmt.Sprintf("%s%d", blockPrefix, block.Number))
	blockValue, err := json.Marshal(block)
	if err != nil {
		return fmt.Errorf("failed to marshal block %d: %w", block.Number, err)
	}

	// Put block
	log.Printf("Atomically putting block %d into DB", block.BlockNumber())
	if err := txn.Put(blockKey, blockValue, nil); err != nil {
		return fmt.Errorf("failed to put block %d in transaction: %w", block.Number, err)
	}

	// Update chain tip height
	heightBytes := []byte(strconv.FormatUint(block.Number, 10))
	log.Printf("Atomically updating chain tip height to: %d", block.Number)
	if err := txn.Put([]byte(chainTipHeightKey), heightBytes, nil); err != nil {
		return fmt.Errorf("failed to update chain tip height in transaction: %w", err)
	}

	// Commit the transaction
	if err := txn.Commit(); err != nil {
		log.Printf("ERROR committing DB transaction for block %d: %v", block.Number, err)
		return fmt.Errorf("failed to commit DB transaction: %w", err)
	}

	log.Printf("Successfully committed block %d and tip height to DB", block.BlockNumber())
	return nil
}

// GetBlock retrieves a block by its height
func (ldb *LevelDB) GetBlock(height uint64) (*Block, error) {
	ldb.mu.Lock()
	defer ldb.mu.Unlock()

	key := []byte(fmt.Sprintf("%s%d", blockPrefix, height))
	data, err := ldb.db.Get(key, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get block %d: %w", height, err)
	}

	var block Block
	if err := json.Unmarshal(data, &block); err != nil {
		return nil, fmt.Errorf("failed to unmarshal block %d: %w", height, err)
	}

	return &block, nil
}

// GetChainTipHeight retrieves the current chain tip height
func (ldb *LevelDB) GetChainTipHeight() (uint64, error) {
	ldb.mu.Lock()
	defer ldb.mu.Unlock()

	data, err := ldb.db.Get([]byte(chainTipHeightKey), nil)
	if err != nil {
		return 0, fmt.Errorf("failed to get chain tip height: %w", err)
	}

	height, err := strconv.ParseUint(string(data), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse chain tip height: %w", err)
	}

	return height, nil
}

// GetTransactionPool retrieves the current transaction pool
func (ldb *LevelDB) GetTransactionPool() ([]*types.Transaction, error) {
	ldb.mu.Lock()
	defer ldb.mu.Unlock()

	data, err := ldb.db.Get([]byte(transactionPoolKey), nil)
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

// Close closes the database
func (ldb *LevelDB) Close() error {
	ldb.mu.Lock()
	defer ldb.mu.Unlock()

	return ldb.db.Close()
}
