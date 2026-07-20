package blockchain

import (
	"context"
	"time"
)

// BlockchainManager defines the interface for blockchain operations
type BlockchainManager interface {
	// Block operations
	CreateBlock(transactions []TransactionInterface) (*BlockInterface, error)
	AddBlock(block *BlockInterface) error
	GetBlock(hash string) (*BlockInterface, error)
	GetBlockByHeight(height uint64) (*BlockInterface, error)
	GetLatestBlock() (*BlockInterface, error)
	GetBlockHeight() uint64

	// Chain operations
	ValidateChain() error
	GetChainLength() uint64
	SyncChain(ctx context.Context) error

	// Lifecycle
	Start(ctx context.Context) error
	Stop() error
	IsRunning() bool
}

// TransactionPool defines the interface for transaction pool operations
type TransactionPool interface {
	// Transaction management
	AddTransaction(tx *TransactionInterface) error
	RemoveTransaction(txID string) error
	GetTransaction(txID string) (*TransactionInterface, error)
	GetPendingTransactions() ([]*TransactionInterface, error)
	GetTransactionCount() int

	// Pool operations
	Clear() error
	ValidateTransaction(tx *TransactionInterface) error

	// Lifecycle
	Start(ctx context.Context) error
	Stop() error
}

// TransactionSearcher defines the interface for transaction search operations
type TransactionSearcher interface {
	// Search operations
	SearchTransactions(query SearchQuery) ([]*TransactionInterface, error)
	SearchByAddress(address string) ([]*TransactionInterface, error)
	SearchByTimeRange(start, end time.Time) ([]*TransactionInterface, error)
	SearchByAmount(minAmount, maxAmount uint64) ([]*TransactionInterface, error)

	// Indexing
	IndexTransaction(tx *TransactionInterface) error
	ReindexAll() error
}

// BlockchainServerInterface defines the interface for blockchain server operations
type BlockchainServerInterface interface {
	// Server operations
	Start(ctx context.Context) error
	Stop() error
	GetStatus() ServerStatus

	// API endpoints
	HandleGetBlock(blockHash string) (*BlockInterface, error)
	HandleGetTransaction(txID string) (*TransactionInterface, error)
	HandleSubmitTransaction(tx *TransactionInterface) error
	HandleGetChainInfo() (*ChainInfo, error)
}

// BlockInterface represents a blockchain block interface
type BlockInterface struct {
	Hash         string                 `json:"hash"`
	PreviousHash string                 `json:"previous_hash"`
	Height       uint64                 `json:"height"`
	Timestamp    time.Time              `json:"timestamp"`
	Transactions []TransactionInterface `json:"transactions"`
	Nonce        uint64                 `json:"nonce"`
	Difficulty   uint64                 `json:"difficulty"`
	MerkleRoot   string                 `json:"merkle_root"`
	AccumRoot    string                 `json:"accum_root"`
}

// TransactionInterface represents a blockchain transaction interface
type TransactionInterface struct {
	ID        string    `json:"id"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	Amount    uint64    `json:"amount"`
	Fee       uint64    `json:"fee"`
	Timestamp time.Time `json:"timestamp"`
	Signature string    `json:"signature"`
	Data      []byte    `json:"data,omitempty"`
	Nonce     uint64    `json:"nonce"`
}

// SearchQuery represents a transaction search query
type SearchQuery struct {
	Address   string    `json:"address,omitempty"`
	StartTime time.Time `json:"start_time,omitempty"`
	EndTime   time.Time `json:"end_time,omitempty"`
	MinAmount uint64    `json:"min_amount,omitempty"`
	MaxAmount uint64    `json:"max_amount,omitempty"`
	Limit     int       `json:"limit,omitempty"`
	Offset    int       `json:"offset,omitempty"`
}

// ServerStatus represents the status of the blockchain server
type ServerStatus struct {
	Running     bool      `json:"running"`
	BlockHeight uint64    `json:"block_height"`
	PeerCount   int       `json:"peer_count"`
	LastBlock   time.Time `json:"last_block"`
	Syncing     bool      `json:"syncing"`
}

// ChainInfo represents information about the blockchain
type ChainInfo struct {
	ChainID     string    `json:"chain_id"`
	Height      uint64    `json:"height"`
	LatestHash  string    `json:"latest_hash"`
	TotalTxs    uint64    `json:"total_txs"`
	GenesisTime time.Time `json:"genesis_time"`
}

// BlockValidator defines the interface for block validation
type BlockValidator interface {
	ValidateBlock(block *BlockInterface) error
	ValidateTransaction(tx *TransactionInterface) error
	ValidateChain(blocks []*BlockInterface) error
}

// BlockStorage defines the interface for block storage operations
type BlockStorage interface {
	StoreBlock(block *BlockInterface) error
	GetBlock(hash string) (*BlockInterface, error)
	GetBlockByHeight(height uint64) (*BlockInterface, error)
	DeleteBlock(hash string) error
	GetLatestBlock() (*BlockInterface, error)
	GetBlockRange(startHeight, endHeight uint64) ([]*BlockInterface, error)
}

// TransactionStorage defines the interface for transaction storage operations
type TransactionStorage interface {
	StoreTransaction(tx *TransactionInterface) error
	GetTransaction(txID string) (*TransactionInterface, error)
	DeleteTransaction(txID string) error
	GetTransactionsByAddress(address string) ([]*TransactionInterface, error)
	GetTransactionsByTimeRange(start, end time.Time) ([]*TransactionInterface, error)
}

// MiningInterface defines the interface for mining operations
type MiningInterface interface {
	StartMining(ctx context.Context) error
	StopMining() error
	IsMining() bool
	SetDifficulty(difficulty uint64) error
	GetHashRate() uint64
}

// ConsensusInterface defines the interface for consensus operations
type ConsensusInterface interface {
	ProposeBlock(block *BlockInterface) error
	ValidateProposal(block *BlockInterface) error
	ReachConsensus(ctx context.Context, block *BlockInterface) error
	GetConsensusState() ConsensusState
}

// ConsensusState represents the current consensus state
type ConsensusState struct {
	Round    uint64 `json:"round"`
	Step     string `json:"step"`
	Proposer string `json:"proposer"`
	Votes    int    `json:"votes"`
	Required int    `json:"required"`
}

// EventEmitter defines the interface for blockchain event emission
type EventEmitter interface {
	EmitBlockAdded(block *BlockInterface) error
	EmitTransactionAdded(tx *TransactionInterface) error
	EmitChainReorg(oldBlocks, newBlocks []*BlockInterface) error
	Subscribe(eventType string, handler EventHandler) error
	Unsubscribe(eventType string, handler EventHandler) error
}

// EventHandler defines the function signature for event handlers
type EventHandler func(event interface{}) error

// BlockchainConfig represents blockchain configuration
type BlockchainConfig struct {
	ChainID                      string          `json:"chain_id"`
	GenesisBlock                 *BlockInterface `json:"genesis_block"`
	BlockTime                    time.Duration   `json:"block_time"`
	MaxBlockSize                 uint64          `json:"max_block_size"`
	MaxTxPerBlock                int             `json:"max_tx_per_block"`
	MinTxFee                     uint64          `json:"min_tx_fee"`
	DifficultyAdjustmentInterval uint64          `json:"difficulty_adjustment_interval"`
}

// Error types for blockchain operations
var (
	ErrBlockNotFound       = NewBlockchainError("block not found")
	ErrTransactionNotFound = NewBlockchainError("transaction not found")
	ErrInvalidBlock        = NewBlockchainError("invalid block")
	ErrInvalidTransaction  = NewBlockchainError("invalid transaction")
	ErrChainInvalid        = NewBlockchainError("chain invalid")
	ErrConsensusFailure    = NewBlockchainError("consensus failure")
)

// BlockchainError represents a blockchain-specific error
type BlockchainError struct {
	Message string
	Code    string
}

func (e *BlockchainError) Error() string {
	return e.Message
}

func NewBlockchainError(message string) *BlockchainError {
	return &BlockchainError{Message: message}
}
