package consensus

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/knirvcorp/knirvoracle/internal/oracle/mmr"
	"go.uber.org/zap"
)

// ConsensusEngine manages the consensus process
type ConsensusEngine struct {
	app             *ABCIApplication
	chainID         string
	blockTime       time.Duration
	currentState    *RoundState
	validatorMode   bool
	logger          *zap.Logger
	ctx             context.Context
	cancel          context.CancelFunc
	mu              sync.RWMutex
	pendingEvidence []Evidence
	heightPath      string
}

// NewConsensusEngine creates a new consensus engine
func NewConsensusEngine(chainID string, blockTime time.Duration, validatorMode bool, dataDir string, logger *zap.Logger) *ConsensusEngine {
	ctx, cancel := context.WithCancel(context.Background())

	app := NewABCIApplication(chainID, dataDir, logger)

	ce := &ConsensusEngine{
		app:           app,
		chainID:       chainID,
		blockTime:     blockTime,
		validatorMode: validatorMode,
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
	if dataDir != "" {
		ce.heightPath = filepath.Join(dataDir, "oracle_height.json")
		var persisted struct {
			Height BlockHeight `json:"height"`
		}
		if data, err := os.ReadFile(ce.heightPath); err == nil && json.Unmarshal(data, &persisted) == nil && persisted.Height > 0 {
			ce.currentState.Height = persisted.Height
		}
	}

	return ce
}

// Start starts the consensus engine
func (ce *ConsensusEngine) Start() error {
	ce.mu.RLock()
	validatorCount := ce.app.GetValidators().Size()
	ce.mu.RUnlock()

	if ce.validatorMode && validatorCount == 0 {
		return fmt.Errorf("consensus engine started in validator mode but no validators are present")
	}

	ce.logger.Info("Starting consensus engine",
		zap.String("chain_id", ce.chainID),
		zap.Duration("block_time", ce.blockTime),
		zap.Bool("validator_mode", ce.validatorMode),
		zap.Int("validators", validatorCount),
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

// GetApp returns the underlying ABCI application.
func (ce *ConsensusEngine) GetApp() *ABCIApplication {
	ce.mu.RLock()
	defer ce.mu.RUnlock()
	return ce.app
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

// AddAuditLeaf appends a leaf to the consensus audit MMR so that the next
// Commit() bags it into the Oracle AppHash. Used by the checkpoint pipeline to
// commit foreign-chain checkpoints into the block header.
func (ce *ConsensusEngine) AddAuditLeaf(data []byte) (uint64, mmr.Hash) {
	return ce.app.AddAuditLeaf(data)
}

// SetAuditMMR makes the checkpoint index MMR the AppHash MMR as well. Keeping
// one pointer removes the possibility of a partial two-write divergence.
func (ce *ConsensusEngine) SetAuditMMR(shared *mmr.MMR) {
	ce.app.SetAuditMMR(shared)
}

// AddEvidence queues validated evidence for the next proposed Oracle block.
func (ce *ConsensusEngine) AddEvidence(evidence Evidence) error {
	if evidence == nil {
		return fmt.Errorf("evidence required")
	}
	if err := evidence.ValidateBasic(); err != nil {
		return err
	}
	ce.mu.Lock()
	defer ce.mu.Unlock()
	ce.pendingEvidence = append(ce.pendingEvidence, evidence)
	return nil
}

func (ce *ConsensusEngine) PendingEvidence() []Evidence {
	ce.mu.RLock()
	defer ce.mu.RUnlock()
	return append([]Evidence(nil), ce.pendingEvidence...)
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
			Evidence: append([]Evidence(nil), ce.pendingEvidence...),
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
	if err := ce.persistHeightLocked(); err != nil {
		return err
	}

	// Update validators if needed
	if validatorUpdates != nil && validatorUpdates.Size() > 0 {
		ce.currentState.Validators = validatorUpdates
	}

	// Increment proposer priority
	ce.currentState.Validators.IncrementProposerPriority(1)
	if consumed := len(block.Evidence.Evidence); consumed > 0 {
		if consumed >= len(ce.pendingEvidence) {
			ce.pendingEvidence = nil
		} else {
			ce.pendingEvidence = ce.pendingEvidence[consumed:]
		}
	}

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
			if ce.validatorMode {
				if err := ce.produceBlock(); err != nil {
					ce.logger.Error("Failed to produce block", zap.Error(err))
				}
			} else if err := ce.advanceClock(); err != nil {
				ce.logger.Error("Failed to advance Oracle clock", zap.Error(err))
			}
		}
	}
}

// advanceClock produces a persisted Oracle height even when this process is a
// non-validator. Proof windows are defined in Oracle blocks, so their clock
// cannot depend on validator-mode block proposal being enabled.
func (ce *ConsensusEngine) advanceClock() error {
	ce.mu.Lock()
	defer ce.mu.Unlock()
	if _, err := ce.app.Commit(); err != nil {
		return fmt.Errorf("commit clock block: %w", err)
	}
	ce.currentState.Height++
	ce.currentState.Round = 0
	ce.currentState.Step = RoundStepNewHeight
	ce.currentState.CommitTime = time.Now()
	return ce.persistHeightLocked()
}

func (ce *ConsensusEngine) persistHeightLocked() error {
	if ce.heightPath == "" {
		return nil
	}
	payload, err := json.Marshal(struct {
		Height BlockHeight `json:"height"`
	}{ce.currentState.Height})
	if err != nil {
		return err
	}
	tmp := ce.heightPath + ".tmp"
	if err := os.WriteFile(tmp, payload, 0644); err != nil {
		return fmt.Errorf("write Oracle height: %w", err)
	}
	if err := os.Rename(tmp, ce.heightPath); err != nil {
		return fmt.Errorf("persist Oracle height: %w", err)
	}
	return nil
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
		"chain_id":   ce.chainID,
		"height":     state.Height,
		"round":      state.Round,
		"step":       state.Step.String(),
		"validators": state.Validators.Size(),
		"app_info":   appInfo,
	}
}
