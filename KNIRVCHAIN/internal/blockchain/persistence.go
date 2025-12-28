package blockchain

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/knirvchain/internal/storage"
)

// BlockStore provides persistent storage for blocks
type BlockStore struct {
	storage storage.Storage
}

// NewBlockStore creates a new BlockStore instance
func NewBlockStore(storage storage.Storage) *BlockStore {
	return &BlockStore{
		storage: storage,
	}
}

// PutBlock stores a block in the database
func (bs *BlockStore) PutBlock(block *Block) error {
	// Serialize block to JSON
	data, err := json.Marshal(block)
	if err != nil {
		return fmt.Errorf("failed to marshal block: %w", err)
	}

	batch := bs.storage.Batch()

	// Store block by ID: block:{blockID} -> Block JSON
	blockKey := []byte("block:" + block.BlockID.String())
	if err := batch.Put(blockKey, data); err != nil {
		return fmt.Errorf("failed to add block to batch: %w", err)
	}

	// Store semantic vector separately for efficient retrieval
	// vector:{blockID} -> SemanticVector JSON
	if len(block.SemanticVector) > 0 {
		vectorData, err := json.Marshal(block.SemanticVector)
		if err != nil {
			return fmt.Errorf("failed to marshal semantic vector: %w", err)
		}
		vectorKey := []byte("vector:" + block.BlockID.String())
		if err := batch.Put(vectorKey, vectorData); err != nil {
			return fmt.Errorf("failed to add vector to batch: %w", err)
		}
	}

	// Store category index: category:{category}:{blockID} -> timestamp
	categoryKey := []byte(fmt.Sprintf("category:%s:%s", block.Category, block.BlockID.String()))
	timestampData := []byte(strconv.FormatInt(block.Timestamp, 10))
	if err := batch.Put(categoryKey, timestampData); err != nil {
		return fmt.Errorf("failed to add category index to batch: %w", err)
	}

	// Store timestamp index: time:{timestamp}:{blockID} -> ""
	timeKey := []byte(fmt.Sprintf("time:%d:%s", block.Timestamp, block.BlockID.String()))
	if err := batch.Put(timeKey, []byte{}); err != nil {
		return fmt.Errorf("failed to add timestamp index to batch: %w", err)
	}

	// Write all changes atomically
	if err := batch.Write(); err != nil {
		return fmt.Errorf("failed to write batch: %w", err)
	}

	return nil
}

// GetBlock retrieves a block by its ID
func (bs *BlockStore) GetBlock(blockID uuid.UUID) (*Block, error) {
	blockKey := []byte("block:" + blockID.String())

	data, err := bs.storage.Get(blockKey)
	if err != nil {
		return nil, fmt.Errorf("block not found: %w", err)
	}

	var block Block
	if err := json.Unmarshal(data, &block); err != nil {
		return nil, fmt.Errorf("failed to unmarshal block: %w", err)
	}

	return &block, nil
}

// HasBlock checks if a block exists
func (bs *BlockStore) HasBlock(blockID uuid.UUID) (bool, error) {
	blockKey := []byte("block:" + blockID.String())
	return bs.storage.Has(blockKey)
}

// DeleteBlock removes a block and its indexes
func (bs *BlockStore) DeleteBlock(blockID uuid.UUID) error {
	// First get the block to access its metadata
	block, err := bs.GetBlock(blockID)
	if err != nil {
		return fmt.Errorf("failed to get block for deletion: %w", err)
	}

	batch := bs.storage.Batch()

	// Delete block
	blockKey := []byte("block:" + blockID.String())
	if err := batch.Delete(blockKey); err != nil {
		return fmt.Errorf("failed to add block deletion to batch: %w", err)
	}

	// Delete semantic vector
	vectorKey := []byte("vector:" + blockID.String())
	if err := batch.Delete(vectorKey); err != nil {
		return fmt.Errorf("failed to add vector deletion to batch: %w", err)
	}

	// Delete category index
	categoryKey := []byte(fmt.Sprintf("category:%s:%s", block.Category, blockID.String()))
	if err := batch.Delete(categoryKey); err != nil {
		return fmt.Errorf("failed to add category index deletion to batch: %w", err)
	}

	// Delete timestamp index
	timeKey := []byte(fmt.Sprintf("time:%d:%s", block.Timestamp, blockID.String()))
	if err := batch.Delete(timeKey); err != nil {
		return fmt.Errorf("failed to add time index deletion to batch: %w", err)
	}

	// Write all deletions atomically
	if err := batch.Write(); err != nil {
		return fmt.Errorf("failed to write deletion batch: %w", err)
	}

	return nil
}

// GetAllBlocks retrieves all blocks
func (bs *BlockStore) GetAllBlocks() ([]*Block, error) {
	iter := bs.storage.NewIterator([]byte("block:"))
	defer iter.Release()

	var blocks []*Block
	for iter.Next() {
		var block Block
		if err := json.Unmarshal(iter.Value(), &block); err != nil {
			return nil, fmt.Errorf("failed to unmarshal block: %w", err)
		}
		blocks = append(blocks, &block)
	}

	if err := iter.Error(); err != nil {
		return nil, fmt.Errorf("iterator error: %w", err)
	}

	return blocks, nil
}

// GetBlocksByCategory retrieves all blocks of a specific category
func (bs *BlockStore) GetBlocksByCategory(category MemoryCategory) ([]*Block, error) {
	prefix := []byte(fmt.Sprintf("category:%s:", category))
	iter := bs.storage.NewIterator(prefix)
	defer iter.Release()

	var blocks []*Block
	for iter.Next() {
		// Extract blockID from key: category:{category}:{blockID}
		key := string(iter.Key())
		// Find the second colon to get blockID
		firstColon := len(fmt.Sprintf("category:%s:", category))
		blockIDStr := key[firstColon:]

		blockID, err := uuid.Parse(blockIDStr)
		if err != nil {
			continue // Skip invalid entries
		}

		block, err := bs.GetBlock(blockID)
		if err != nil {
			continue // Skip if block not found
		}

		blocks = append(blocks, block)
	}

	if err := iter.Error(); err != nil {
		return nil, fmt.Errorf("iterator error: %w", err)
	}

	return blocks, nil
}

// GetBlocksInTimeRange retrieves blocks within a time range
func (bs *BlockStore) GetBlocksInTimeRange(startTime, endTime int64) ([]*Block, error) {
	// Iterate over time index
	iter := bs.storage.NewIterator([]byte("time:"))
	defer iter.Release()

	var blocks []*Block
	for iter.Next() {
		// Extract timestamp and blockID from key: time:{timestamp}:{blockID}
		key := string(iter.Key())

		// Parse the key to extract timestamp and blockID
		var timestamp int64
		var blockIDStr string
		_, err := fmt.Sscanf(key, "time:%d:%s", &timestamp, &blockIDStr)
		if err != nil {
			continue // Skip invalid entries
		}

		// Filter by time range
		if timestamp < startTime || timestamp > endTime {
			continue
		}

		blockID, err := uuid.Parse(blockIDStr)
		if err != nil {
			continue // Skip invalid entries
		}

		block, err := bs.GetBlock(blockID)
		if err != nil {
			continue // Skip if block not found
		}

		blocks = append(blocks, block)
	}

	if err := iter.Error(); err != nil {
		return nil, fmt.Errorf("iterator error: %w", err)
	}

	return blocks, nil
}

// GetBlockCount returns the total number of blocks
func (bs *BlockStore) GetBlockCount() (int, error) {
	iter := bs.storage.NewIterator([]byte("block:"))
	defer iter.Release()

	count := 0
	for iter.Next() {
		count++
	}

	if err := iter.Error(); err != nil {
		return 0, fmt.Errorf("iterator error: %w", err)
	}

	return count, nil
}

// GetSemanticVector retrieves just the semantic vector for a block
func (bs *BlockStore) GetSemanticVector(blockID uuid.UUID) ([]float32, error) {
	vectorKey := []byte("vector:" + blockID.String())

	data, err := bs.storage.Get(vectorKey)
	if err != nil {
		return nil, fmt.Errorf("semantic vector not found: %w", err)
	}

	var vector []float32
	if err := json.Unmarshal(data, &vector); err != nil {
		return nil, fmt.Errorf("failed to unmarshal semantic vector: %w", err)
	}

	return vector, nil
}

// GetAllSemanticVectors retrieves all semantic vectors with their block IDs
func (bs *BlockStore) GetAllSemanticVectors() (map[uuid.UUID][]float32, error) {
	iter := bs.storage.NewIterator([]byte("vector:"))
	defer iter.Release()

	vectors := make(map[uuid.UUID][]float32)
	for iter.Next() {
		// Extract blockID from key: vector:{blockID}
		key := string(iter.Key())
		blockIDStr := key[len("vector:"):]

		blockID, err := uuid.Parse(blockIDStr)
		if err != nil {
			continue // Skip invalid entries
		}

		var vector []float32
		if err := json.Unmarshal(iter.Value(), &vector); err != nil {
			continue // Skip invalid entries
		}

		vectors[blockID] = vector
	}

	if err := iter.Error(); err != nil {
		return nil, fmt.Errorf("iterator error: %w", err)
	}

	return vectors, nil
}
