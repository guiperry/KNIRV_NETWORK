package blockchain

import (
	"KNIRVROUTER/internal/types"
)

// Blockchain defines the interface for blockchain operations needed by P2P consensus
type Blockchain interface {
	// Chain identification
	ChainID() string

	// Block operations
	AddBlock(block *Block) error
	GetBlocks() []*Block
	GetLastBlock() *Block
	VerifyBlock(block *Block) bool

	// Transaction operations
	AddTransactionToTransactionPool(tx *types.Transaction) error
	GetTransactionPool() []*types.Transaction

	// Locking mechanisms
	Lock()
	Unlock()

	// Mining status
	IsActivelyMining() bool
}
