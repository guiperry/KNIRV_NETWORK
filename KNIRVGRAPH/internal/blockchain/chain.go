package blockchain

import (
	"blockchain-app/internal/storage"
	"blockchain-app/internal/types"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

type Chain struct {
	mu      sync.RWMutex
	storage storage.Storage
	state   *types.State
	height  uint64
}

func NewChain(storage storage.Storage) *Chain {
	return &Chain{
		storage: storage,
		state:   types.NewState(),
		height:  0,
	}
}

func (c *Chain) AddBlock(block *types.Block) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.validateBlock(block); err != nil {
		return fmt.Errorf("invalid block: %w", err)
	}

	if err := c.executeBlock(block); err != nil {
		return fmt.Errorf("failed to execute block: %w", err)
	}

	blockData, err := block.Serialize()
	if err != nil {
		return fmt.Errorf("failed to serialize block: %w", err)
	}

	key := fmt.Sprintf("block_%d", block.Header.Height)
	if err := c.storage.Put([]byte(key), blockData); err != nil {
		return fmt.Errorf("failed to store block: %w", err)
	}

	c.height = block.Header.Height
	return nil
}

// GetHeight returns the current height of the blockchain
func (c *Chain) GetHeight() uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.height
}

func (c *Chain) validateBlock(block *types.Block) error {
	// Validate block height
	if block.Header.Height != c.height+1 {
		return fmt.Errorf("invalid block height: expected %d, got %d", c.height+1, block.Header.Height)
	}

	// Validate previous hash
	if c.height > 0 {
		lastBlock, err := c.getBlockUnsafe(c.height)
		if err != nil {
			return fmt.Errorf("failed to get last block: %w", err)
		}

		// Check both PreviousHash and PrevHash for compatibility
		prevHash := block.Header.PreviousHash
		if prevHash == "" {
			prevHash = block.Header.PrevHash
		}
		if prevHash != lastBlock.Hash {
			return fmt.Errorf("invalid previous hash")
		}
	}

	// Validate timestamp (block cannot be from the future)
	if block.Header.Timestamp.After(time.Now()) {
		return fmt.Errorf("block timestamp is in the future")
	}

	// Validate transactions
	for _, tx := range block.Transactions {
		if !tx.Verify() {
			return fmt.Errorf("invalid transaction signature")
		}
	}

	return nil
}

func (c *Chain) executeBlock(block *types.Block) error {
	// Execute transactions
	for _, tx := range block.Transactions {
		if err := c.executeTransaction(tx); err != nil {
			return fmt.Errorf("failed to execute transaction %s: %w", tx.ID, err)
		}
	}

	c.state.Height = block.Header.Height
	return nil
}

func (c *Chain) executeTransaction(tx *types.Transaction) error {
	switch {
	case tx.To != "":
		// Transfer transaction
		return c.state.Transfer(tx.From, tx.To, tx.Amount)
	default:
		// Contract deployment or other transaction types
		return nil
	}
}

func (c *Chain) GetBlock(height uint64) (*types.Block, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.getBlockUnsafe(height)
}

// getBlockUnsafe retrieves a block without acquiring locks (assumes caller holds appropriate lock)
func (c *Chain) getBlockUnsafe(height uint64) (*types.Block, error) {
	key := fmt.Sprintf("block_%d", height)
	data, err := c.storage.Get([]byte(key))
	if err != nil {
		return nil, fmt.Errorf("block not found: %w", err)
	}

	var block types.Block
	if err := json.Unmarshal(data, &block); err != nil {
		return nil, fmt.Errorf("failed to unmarshal block: %w", err)
	}

	return &block, nil
}

func (c *Chain) GetCurrentHeight() uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.height
}

func (c *Chain) GetState() *types.State {
	return c.state
}
