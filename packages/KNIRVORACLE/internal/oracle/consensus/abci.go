package consensus

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/knirvcorp/knirvoracle/internal/oracle/mmr"
	"go.uber.org/zap"
)

// ABCIApplication implements a simplified ABCI interface
// In production, this would implement github.com/cometbft/cometbft/abci/types.Application
type ABCIApplication struct {
	state           *State
	validators      *ValidatorSet
	consensusParams *ConsensusParams
	auditMMR        *mmr.MMR
	// onCheckpointTx routes a consensus-delivered checkpoint/finality transaction
	// to the Oracle's admission path (merkle-math.md §3.3 — same path as HTTP).
	onCheckpointTx func(txType string, payload []byte) error
	logger         *zap.Logger
	mu             sync.RWMutex
}

// NewABCIApplication creates a new ABCI application. The audit MMR starts
// empty here and is never independently loaded from or persisted to disk by
// this type — the Oracle (see oracle.go's NewOracle) wires in the checkpoint
// pipeline's own MMR via SetAuditMMR before any commit can be observed, so
// checkpoint_store.go's mmr_leaf_log.json is the single durable copy of this
// data. This type used to keep a second, independently-persisted copy
// (audit_mmr_leaf_log.json, written from Commit), which was the actual cause
// of the "checkpoint MMR and persisted AppHash MMR diverged; refusing unsafe
// recovery" startup failures: the two on-disk logs were written by different
// call sites (Commit vs. the checkpoint admission path) and could fall out
// of sync whenever the process was killed between the two writes — trivially
// reproducible with an unclean shutdown, not actual data corruption.
func NewABCIApplication(chainID string, logger *zap.Logger) *ABCIApplication {
	return &ABCIApplication{
		state: &State{
			Version: ConsensusVersion{
				Block: 11,
				App:   1,
			},
			ChainID:         chainID,
			InitialHeight:   1,
			LastBlockHeight: 0,
			Validators:      NewValidatorSet([]*ConsensusValidator{}),
			NextValidators:  NewValidatorSet([]*ConsensusValidator{}),
			LastValidators:  NewValidatorSet([]*ConsensusValidator{}),
			ConsensusParams: *DefaultConsensusParams(),
		},
		consensusParams: DefaultConsensusParams(),
		auditMMR:        mmr.New(),
		logger:          logger,
	}
}

// SetCheckpointTxHandler registers the consensus-path admission hook used by
// DeliverTx for checkpoint/finality transactions.
func (app *ABCIApplication) SetCheckpointTxHandler(fn func(txType string, payload []byte) error) {
	app.mu.Lock()
	defer app.mu.Unlock()
	app.onCheckpointTx = fn
}

// SetAuditMMR installs the Oracle checkpoint log as the single MMR instance
// used for both inclusion proofs and AppHash commits. This is the only audit
// MMR any part of the consensus app ever observes past startup — see the
// NewABCIApplication doc comment for why a second, independently-persisted
// copy is deliberately not kept here.
func (app *ABCIApplication) SetAuditMMR(shared *mmr.MMR) {
	app.mu.Lock()
	defer app.mu.Unlock()
	if shared == nil {
		shared = mmr.New()
	}
	root := shared.BagRoot()
	app.logger.Info("Audit MMR installed",
		zap.Uint64("size", shared.Size()),
		zap.String("root", fmt.Sprintf("%x", root)),
	)
	app.auditMMR = shared
}

// AddAuditLeaf appends through the application lock to the canonical MMR.
func (app *ABCIApplication) AddAuditLeaf(data []byte) (uint64, mmr.Hash) {
	app.mu.Lock()
	defer app.mu.Unlock()
	return app.auditMMR.AddRaw(mmr.LeafHash(data))
}

// Info returns information about the application state
func (app *ABCIApplication) Info() map[string]interface{} {
	app.mu.RLock()
	defer app.mu.RUnlock()

	return map[string]interface{}{
		"chain_id":          app.state.ChainID,
		"last_block_height": app.state.LastBlockHeight,
		"app_hash":          fmt.Sprintf("%x", app.state.AppHash),
		"version":           app.state.Version,
	}
}

// InitChain initializes the blockchain
func (app *ABCIApplication) InitChain(chainID string, validators []*ConsensusValidator) error {
	app.mu.Lock()
	defer app.mu.Unlock()

	app.state.ChainID = chainID
	app.state.InitialHeight = 1
	app.state.Validators = NewValidatorSet(validators)
	app.state.NextValidators = NewValidatorSet(validators)
	app.state.LastValidators = NewValidatorSet([]*ConsensusValidator{})

	app.logger.Info("Chain initialized",
		zap.String("chain_id", chainID),
		zap.Int("validators", len(validators)),
	)

	return nil
}

// BeginBlock signals the beginning of a new block
func (app *ABCIApplication) BeginBlock(header *BlockHeader) error {
	app.mu.Lock()
	defer app.mu.Unlock()

	app.logger.Debug("Begin block",
		zap.Uint64("height", uint64(header.Height)),
		zap.String("proposer", header.ProposerAddress.String()),
	)

	return nil
}

// DeliverTx delivers a transaction to the application
func (app *ABCIApplication) DeliverTx(tx []byte) error {
	// Parse and validate transaction
	var txData map[string]interface{}
	if err := json.Unmarshal(tx, &txData); err != nil {
		return fmt.Errorf("invalid transaction format: %w", err)
	}

	// Route checkpoint/finality transactions arriving through consensus to the
	// same Oracle admission path used by the HTTP endpoints (merkle-math.md §3.3).
	txType, _ := txData["type"].(string)
	if txType == "checkpoint" || txType == "finality" {
		app.mu.RLock()
		handler := app.onCheckpointTx
		app.mu.RUnlock()
		if handler != nil {
			payload, _ := json.Marshal(txData["payload"])
			return handler(txType, payload)
		}
	}

	app.logger.Debug("Transaction delivered",
		zap.Int("size", len(tx)),
	)

	return nil
}

// EndBlock signals the end of a block
func (app *ABCIApplication) EndBlock(height BlockHeight) (*ValidatorSet, error) {
	app.mu.Lock()
	defer app.mu.Unlock()

	app.logger.Debug("End block",
		zap.Uint64("height", uint64(height)),
	)

	// Return validator updates (if any)
	return app.state.NextValidators, nil
}

// Commit commits the current state
func (app *ABCIApplication) Commit() ([]byte, error) {
	app.mu.Lock()
	defer app.mu.Unlock()

	// Update state
	app.state.LastBlockHeight++

	// Every Oracle block commits to the append-only audit log. Before the first
	// audit leaf, this is the canonical SHA256("knirv-mmr-empty") root. The
	// audit MMR itself is durably persisted exactly once, by the checkpoint
	// pipeline that owns it (checkpoint_store.go's mmr_leaf_log.json, written
	// via persistCheckpointLocked before this Commit is ever called) — nothing
	// further to write here. See NewABCIApplication's doc comment for why a
	// second, independently-persisted copy used to live here and why that was
	// the cause of the checkpoint/AppHash MMR "diverged" startup failures.
	root := app.auditMMR.BagRoot()
	appHash := append([]byte(nil), root[:]...)
	app.state.AppHash = appHash

	app.logger.Info("Block committed",
		zap.Uint64("height", uint64(app.state.LastBlockHeight)),
		zap.String("app_hash", fmt.Sprintf("%x", appHash)),
	)

	return appHash, nil
}

// Query queries the application state
func (app *ABCIApplication) Query(path string, data []byte) ([]byte, error) {
	app.mu.RLock()
	defer app.mu.RUnlock()

	switch path {
	case "/info":
		info := app.Info()
		return json.Marshal(info)
	case "/state":
		return json.Marshal(app.state)
	case "/validators":
		return json.Marshal(app.state.Validators)
	default:
		return nil, fmt.Errorf("unknown query path: %s", path)
	}
}

// CheckTx checks if a transaction is valid
func (app *ABCIApplication) CheckTx(tx []byte) error {
	// Parse transaction
	var txData map[string]interface{}
	if err := json.Unmarshal(tx, &txData); err != nil {
		return fmt.Errorf("invalid transaction format: %w", err)
	}

	// Basic validation
	if len(tx) == 0 {
		return fmt.Errorf("empty transaction")
	}

	if len(tx) > int(app.consensusParams.Block.MaxBytes) {
		return fmt.Errorf("transaction too large")
	}

	return nil
}

// SetValidators updates the validator set
func (app *ABCIApplication) SetValidators(validators []*ConsensusValidator) error {
	app.mu.Lock()
	defer app.mu.Unlock()

	app.state.NextValidators = NewValidatorSet(validators)
	app.state.LastHeightValidatorsChanged = app.state.LastBlockHeight

	app.logger.Info("Validator set updated",
		zap.Int("validators", len(validators)),
		zap.Uint64("height", uint64(app.state.LastBlockHeight)),
	)

	return nil
}

// GetState returns the current state
func (app *ABCIApplication) GetState() *State {
	app.mu.RLock()
	defer app.mu.RUnlock()

	return app.state
}

// GetValidators returns the current validator set
func (app *ABCIApplication) GetValidators() *ValidatorSet {
	app.mu.RLock()
	defer app.mu.RUnlock()

	return app.state.Validators
}

// UpdateConsensusParams updates consensus parameters
func (app *ABCIApplication) UpdateConsensusParams(params *ConsensusParams) error {
	app.mu.Lock()
	defer app.mu.Unlock()

	app.state.ConsensusParams = *params
	app.consensusParams = params
	app.state.LastHeightConsensusParamsChanged = app.state.LastBlockHeight

	app.logger.Info("Consensus params updated",
		zap.Uint64("height", uint64(app.state.LastBlockHeight)),
	)

	return nil
}
