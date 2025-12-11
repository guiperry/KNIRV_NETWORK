package consensus

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ConsensusEngine manages the consensus process
type ConsensusEngine struct {
	app          *ABCIApplication
	chainID      string
	blockTime    time.Duration
	currentState *RoundState
	logger       *zap.Logger
	ctx          context.Context
	cancel       context.CancelFunc
	mu           sync.RWMutex
}

// NewConsensusEngine creates a new consensus engine
func NewConsensusEngine(chainID string, blockTime time.Duration, logger *zap.Logger) *ConsensusEngine {
	ctx, cancel := context.WithCancel(context.Background())

	app := NewABCIApplication(chainID, logger)

	ce := &ConsensusEngine{
		app:       app,
		chainID:   chainID,
		blockTime: blockTime,
		currentState: &RoundState{
			Height:    1,
			Round:     0,
			Step:      RoundStepNewHeight,
			StartTime: time.Now(),
		},
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
	}

	return ce
}

// Start starts the consensus engine
func (ce *ConsensusEngine) Start() error {
	ce.logger.Info("Starting consensus engine",
		zap.String("chain_id", ce.chainID),
		zap.Duration("block_time", ce.blockTime),
	)

	// Start consensus loop
	go ce.consensusLoop()

	return nil
}

// Stop stops the consensus engine
func (ce *ConsensusEngine) Stop() error {
	ce.logger.Info("Stopping consensus engine")
	ce.cancel()
	return nil
}

// InitChain initializes the chain with genesis validators
func (ce *ConsensusEngine) InitChain(validators []*ConsensusValidator) error {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	if err := ce.app.InitChain(ce.chainID, validators); err != nil {
		return fmt.Errorf("failed to init chain: %w", err)
	}

	state := ce.app.GetState()
	ce.currentState.Validators = state.Validators
	ce.currentState.LastValidators = state.LastValidators

	ce.logger.Info("Chain initialized",
		zap.Int("validators", len(validators)),
	)

	return nil
}

// GetState returns the current consensus state
func (ce *ConsensusEngine) GetState() *RoundState {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	return ce.currentState
}

// GetHeight returns the current block height
func (ce *ConsensusEngine) GetHeight() BlockHeight {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	return ce.currentState.Height
}

// GetValidators returns the current validator set
func (ce *ConsensusEngine) GetValidators() *ValidatorSet {
	return ce.app.GetValidators()
}

// UpdateValidators updates the validator set
func (ce *ConsensusEngine) UpdateValidators(validators []*ConsensusValidator) error {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	if err := ce.app.SetValidators(validators); err != nil {
		return fmt.Errorf("failed to update validators: %w", err)
	}

	ce.logger.Info("Validators updated",
		zap.Int("count", len(validators)),
	)

	return nil
}

// ProposeBlock proposes a new block
func (ce *ConsensusEngine) ProposeBlock(txs [][]byte) (*Block, error) {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	validators := ce.app.GetValidators()

	// Get proposer
	proposer := validators.GetProposer()
	if proposer == nil {
		return nil, fmt.Errorf("no proposer available")
	}

	// Create block header
	header := &BlockHeader{
		Version: ConsensusVersion{
			Block: 11,
			App:   1,
		},
		ChainID:         ce.chainID,
		Height:          ce.currentState.Height,
		Time:            time.Now(),
		ProposerAddress: proposer.Address,
		ValidatorsHash:  []byte("validators_hash"), // Simplified
	}

	// Create block
	block := &Block{
		Header: *header,
		Data: BlockData{
			Txs: txs,
		},
		Evidence: EvidenceData{
			Evidence: []Evidence{},
		},
	}

	ce.logger.Debug("Block proposed",
		zap.Uint64("height", uint64(header.Height)),
		zap.Int("txs", len(txs)),
		zap.String("proposer", proposer.Address.String()),
	)

	return block, nil
}

// CommitBlock commits a block
func (ce *ConsensusEngine) CommitBlock(block *Block) error {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	// Begin block
	if err := ce.app.BeginBlock(&block.Header); err != nil {
		return fmt.Errorf("begin block failed: %w", err)
	}

	// Deliver transactions
	for _, tx := range block.Data.Txs {
		if err := ce.app.DeliverTx(tx); err != nil {
			ce.logger.Warn("Transaction failed", zap.Error(err))
			// Continue with other transactions
		}
	}

	// End block
	validatorUpdates, err := ce.app.EndBlock(block.Header.Height)
	if err != nil {
		return fmt.Errorf("end block failed: %w", err)
	}

	// Commit
	appHash, err := ce.app.Commit()
	if err != nil {
		return fmt.Errorf("commit failed: %w", err)
	}

	// Update state
	ce.currentState.Height++
	ce.currentState.Round = 0
	ce.currentState.Step = RoundStepNewHeight
	ce.currentState.CommitTime = time.Now()

	// Update validators if needed
	if validatorUpdates != nil && validatorUpdates.Size() > 0 {
		ce.currentState.Validators = validatorUpdates
	}

	// Increment proposer priority
	ce.currentState.Validators.IncrementProposerPriority(1)

	ce.logger.Info("Block committed",
		zap.Uint64("height", uint64(block.Header.Height)),
		zap.String("app_hash", fmt.Sprintf("%x", appHash)),
	)

	return nil
}

// consensusLoop runs the main consensus loop
func (ce *ConsensusEngine) consensusLoop() {
	ticker := time.NewTicker(ce.blockTime)
	defer ticker.Stop()

	for {
		select {
		case <-ce.ctx.Done():
			return
		case <-ticker.C:
			if err := ce.produceBlock(); err != nil {
				ce.logger.Error("Failed to produce block", zap.Error(err))
			}
		}
	}
}

// produceBlock produces and commits a new block
func (ce *ConsensusEngine) produceBlock() error {
	// Propose block with no transactions for now
	block, err := ce.ProposeBlock([][]byte{})
	if err != nil {
		return fmt.Errorf("failed to propose block: %w", err)
	}

	// Commit block
	if err := ce.CommitBlock(block); err != nil {
		return fmt.Errorf("failed to commit block: %w", err)
	}

	return nil
}

// Query queries the application state
func (ce *ConsensusEngine) Query(path string, data []byte) ([]byte, error) {
	return ce.app.Query(path, data)
}

// GetInfo returns information about the consensus engine
func (ce *ConsensusEngine) GetInfo() map[string]interface{} {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	appInfo := ce.app.Info()
	state := ce.currentState

	return map[string]interface{}{
		"chain_id":      ce.chainID,
		"height":        state.Height,
		"round":         state.Round,
		"step":          state.Step.String(),
		"validators":    state.Validators.Size(),
		"app_info":      appInfo,
	}
}
